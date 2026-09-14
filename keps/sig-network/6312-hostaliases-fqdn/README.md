# KEP-6312: Relaxed validation for HostAliases hostnames (allow trailing dot / FQDN)

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

`spec.hostAliases[].hostnames` is currently validated as a DNS-1123 subdomain, which
rejects a trailing dot. This KEP proposes relaxing that validation, behind a feature
gate, to accept a single trailing dot, so users can express a hostname in FQDN
notation the same way they already can in a standard Unix hosts file.

## Motivation

A trailing dot on a hostname is standard notation, on Unix systems a name ending in
`.` is treated as already fully-qualified, so the resolver does not append any
search-domain suffix when looking it up. This is a deliberate, widely used
convention, not a typo, and it is exactly the notation many applications and
operators reach for when they want a static hosts entry to resolve without any
ambiguity from the pod's search list.

`hostAliases` is the only supported way to inject a static entry into a pod's
`/etc/hosts`. Today, doing so with a trailing-dot hostname is simply rejected by
the API server, with no workaround: it cannot be baked into the image generically
(that breaks portability across environments), and there is no other field that
accepts a static hosts entry.

- Original report: [kubernetes/kubernetes#135273](https://github.com/kubernetes/kubernetes/issues/135273)

### Goals

- Allow a single trailing dot on entries in `spec.hostAliases[].hostnames`.

### Non-Goals

- Changing validation for any other hostname-bearing field (Pod `hostname`/`subdomain`,
  Service names, etc.). Those are tracked by their own KEPs if relaxed at all.
- Changing DNS resolution behavior itself. `hostAliases` only affects the pod's
  `/etc/hosts` file; it does not touch `resolv.conf` or search-domain configuration,
  so this is scoped to what string is accepted in that one field, not to how
  resolution works.

## Proposal

Today, `HostAlias.Hostnames` entries are validated with the same DNS-1123 subdomain
check used for most other Kubernetes name fields, which does not permit a trailing
dot. The proposal is to introduce a feature gate that, when enabled, accepts a
single trailing dot at the end of a `hostAliases[].hostnames` entry, by stripping it
before delegating to the existing subdomain validation.

### Risks and Mitigations

1. **libc differences in trailing-dot handling.** A closely related change to DNS
   search-domain validation ([KEP-4427](/keps/sig-network/4427-relaxed-dns-search-validation))
   previously shipped a fix that worked under glibc but broke resolution under musl
   libc (used by `alpine` and other minimal base images), and had to be re-fixed. This
   KEP touches a different code path (`/etc/hosts` content, not `resolv.conf` search
   domains), but the lesson applies directly: correctness has to be verified by
   actually resolving a trailing-dot hostname from inside both a glibc-based and a
   musl-based container, not just by confirming the API server accepts the object and
   writes the expected line into `/etc/hosts`. This is called out explicitly in the
   [test plan](#e2e-tests) below.
2. **Downstream tooling.** Admission webhooks, policy engines, or client-side
   generators that independently re-validate `hostAliases` entries (e.g. assuming the
   existing DNS-1123 subdomain shape) may reject the new format until updated. This is
   a real but bounded cost, the same category of risk called out in
   [KEP-5311](/keps/sig-network/5311-relaxed-validation-for-service-names) for Service
   names, and mitigated the same way: gated rollout, so nobody is affected until they
   opt in.

## Design Details

Introduce a new feature gate, `RelaxedHostAliasesValidation`, disabled by default in
alpha.

When the feature gate is disabled, `hostAliases[].hostnames` entries continue to be
validated exactly as they are today (DNS-1123 subdomain, no trailing dot).

When the feature gate is enabled, an entry may additionally carry a single trailing
dot. Validation strips at most one trailing `.` before running the existing DNS-1123
subdomain check against the remainder, so this is additive: every value that is
valid today remains valid.

`hostAliases` is set at pod creation and is not part of the mutable subset of the pod
spec, so unlike some other relaxed-validation KEPs there is no separate "update"
validation path to reconcile, only pod creation is affected.

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None identified.

##### Unit tests

The validation function covering `HostAlias.Hostnames` in `pkg/apis/core/validation`
will be updated to cover both the gate-enabled and gate-disabled paths.

- `pkg/apis/core/validation`: coverage to be measured against `main` at KEP
  implementation time.

##### Integration tests

**Alpha:**

1. With the feature gate enabled, create a Pod with a trailing-dot `hostAliases`
   entry and confirm it is accepted.
2. With the feature gate disabled, create the same Pod and confirm it is rejected
   with the existing validation error.
3. With the feature gate enabled, confirm a `hostAliases` entry without a trailing
   dot is still accepted (no regression on the existing behavior).

##### e2e tests

**Alpha:**

- Create a Pod with a trailing-dot `hostAliases` entry on both a glibc-based and a
  musl-based (`alpine`) container image, and confirm the entry actually resolves
  correctly from inside the container (e.g. via `getent hosts` / `nslookup`), not
  just that the API server accepted the object. This directly covers the libc risk
  noted above.

### Graduation Criteria

#### Alpha

- Feature implemented behind `RelaxedHostAliasesValidation`, disabled by default.
- Initial e2e tests completed and enabled, including the glibc/musl resolution check.

#### Beta

- Integration and e2e tests completed and enabled.
- No open issues from alpha usage.

#### GA

- Time passes with no major objections.
- Promote the e2e resolution test to conformance, if applicable.

### Upgrade / Downgrade Strategy

Upgrade: existing pods are unaffected, since `hostAliases` is only validated at
creation. Newly created pods can use the relaxed format once the gate is enabled.

Downgrade: a pod already running with a trailing-dot `hostAliases` entry continues
running unaffected (validation is not re-run against live objects). New pod creation
using the relaxed format will fail once the gate is disabled, the same downgrade
behavior as KEP-5311.

### Version Skew Strategy

Not applicable, this only changes kube-apiserver validation for a single component;
no other component's behavior depends on it.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: RelaxedHostAliasesValidation
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

No. This only changes what is accepted by validation; it does not change any
existing default.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, via the feature gate. Pods already created with the relaxed format keep
running; new pod creation using it will be rejected until re-enabled.

###### What happens if we reenable the feature if it was previously rolled back?

The relaxed validation is enabled again for new pod creation.

###### Are there any tests for feature enablement/disablement?

Yes, see the [integration tests](#integration-tests) above.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout cannot affect already-running workloads: the change only affects
validation of new pod creations. A rollback likewise cannot impact already-running
workloads, and only affects the ability to *create new* pods using the relaxed
format.

###### What specific metrics should inform a rollback?

`apiserver_request_total{code=500, resource=pods, verb=POST}` would indicate the
validation change itself is causing unexpected server errors, though this is a
validation-only change and no such errors are expected.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

To be completed and documented here at implementation time, following the same
approach as [KEP-5311](/keps/sig-network/5311-relaxed-validation-for-service-names#were-upgrade-and-rollback-tested-was-the-upgrade-downgrade-upgrade-path-tested).

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Existence of any Pod with a `hostAliases[].hostnames` entry ending in a dot.

###### How can someone using this feature know that it is working for their instance?

If they can successfully create a Pod with a trailing-dot `hostAliases` entry, and
that entry resolves correctly from inside the container.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A, validation-only change with no independent SLOs.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. This is a change to API validation only.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No, the relaxed validation does not change the maximum length of the field.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A. This is a change to validation within the API server.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- 2026-09-02: KEP created, enhancement tracking issue filed as
  [kubernetes/enhancements#6312](https://github.com/kubernetes/enhancements/issues/6312).

## Drawbacks

Downstream tooling (admission webhooks, policy engines, client-side generators) that
independently re-validates `hostAliases` entries against the current DNS-1123
subdomain shape could reject the new trailing-dot format until updated.

## Alternatives

- **Bake the hosts entry into the container image.** Works for a single fixed
  environment, but defeats the purpose of `hostAliases` being a runtime-configurable
  field and breaks image portability across clusters/environments.
- **Use `dnsConfig`/`dnsPolicy: None` with a custom resolver configuration.** This
  operates at a different layer (DNS resolution and search domains, not static hosts
  entries) and does not provide an equivalent to a static `/etc/hosts` entry; it also
  requires taking over DNS resolution entirely rather than adding one static entry.
