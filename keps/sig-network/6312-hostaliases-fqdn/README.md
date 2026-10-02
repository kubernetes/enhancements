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
  - [/etc/hosts rendering](#etchosts-rendering)
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
notation the same way they already can in a standard Unix hosts file. The kubelet
then renders such a hostname into the pod's `/etc/hosts` so that it resolves both
with and without the trailing dot.

[kubernetes/enhancements#6075](https://github.com/kubernetes/enhancements/pull/6075)
proposes the same field relaxation. The two proposals overlap, and the differences
are discussed under [Alternatives](#alternatives).

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

Two facts, measured on glibc 2.31, glibc 2.41, musl 1.1.24 and musl 1.2.6
(scripts and raw output: https://github.com/MU5A/hostaliases-fqdn-evidence), explain
why this matters in practice:

- **The absolute form avoids a real cost.** With `ndots:5` and the stock three
  Kubernetes search domains, resolving an external name such as `api.example.com`
  sends 8 DNS queries (6 of them wasted on search-suffix NXDOMAIN lookups), against 2
  for `api.example.com.`. With five search domains it is 12 against 2. Applications
  that care about this use the absolute form.
- **Today's hostAliases cannot override that form.** Resolvers match `/etc/hosts`
  names literally. An entry `10.10.10.10 api.example.com` is not returned for a
  lookup of `api.example.com.`, and an entry for `api.example.com.` is not returned
  for `api.example.com`. A pod that uses the absolute form therefore cannot be pointed
  at a different IP (for example a sandbox endpoint in a staging cluster) with
  `hostAliases`.

- Original report: [kubernetes/kubernetes#135273](https://github.com/kubernetes/kubernetes/issues/135273)

### Goals

- Allow a single trailing dot on entries in `spec.hostAliases[].hostnames`.
- Make an alias written with a trailing dot resolve for lookups with and without the
  trailing dot, without changing the name that reverse lookups return.

### Non-Goals

- Changing validation for any other hostname-bearing field (Pod `hostname`/`subdomain`,
  Service names, etc.). Those are tracked by their own KEPs if relaxed at all.
- Changing DNS resolution behavior itself. `hostAliases` only affects the pod's
  `/etc/hosts` file, and only for hostnames that carry a trailing dot; it does not
  touch `resolv.conf` or search-domain configuration.

## Proposal

Today, `HostAlias.Hostnames` entries are validated with the same DNS-1123 subdomain
check used for most other Kubernetes name fields, which does not permit a trailing
dot. The proposal is to introduce a feature gate that, when enabled, accepts a
single trailing dot at the end of a `hostAliases[].hostnames` entry, by stripping it
before delegating to the existing subdomain validation.

Because resolvers match names literally, a hosts line containing only `name.` would
answer the absolute lookup but not the relative one. When the gate is enabled, the
kubelet therefore writes the undotted name followed by the dotted name on the same
line (`IP name name.`). Both forms resolve, and reverse lookups, which return the
first name on the line, keep returning the undotted name, as they do today.

### Risks and Mitigations

1. **libc differences in trailing-dot handling.** A closely related change to DNS
   search-domain validation ([KEP-4427](/keps/sig-network/4427-relaxed-dns-search-validation))
   previously shipped a fix that worked under glibc but broke resolution under musl
   libc (used by `alpine` and other minimal base images), and had to be re-fixed. This
   KEP touches a different code path (`/etc/hosts` content, not `resolv.conf` search
   domains), but the lesson applies directly: correctness has to be verified by
   actually resolving a trailing-dot hostname from inside both a glibc-based and a
   musl-based container, not just by confirming the API server accepts the object and
   writes the expected line into `/etc/hosts`. The behavior proposed here was
   measured rather than assumed: dotted tokens parse without rejecting the line,
   aliases after the dotted token still resolve, and the single-line form resolves
   both query forms on glibc 2.31, glibc 2.41, musl 1.1.24, musl 1.2.6 and Go's pure
   resolver. The [e2e test](#e2e-tests) repeats this inside real pods.
2. **Downstream tooling.** Admission webhooks, policy engines, or client-side
   generators that independently re-validate `hostAliases` entries (e.g. assuming the
   existing DNS-1123 subdomain shape) may reject the new format until updated. This is
   a real but bounded cost, the same category of risk called out in
   [KEP-5311](/keps/sig-network/5311-relaxed-validation-for-service-names) for Service
   names, and mitigated the same way: gated rollout, so nobody is affected until they
   opt in.
3. **Reverse-lookup output.** `gethostbyaddr` and `getnameinfo` return the first name
   on the matching line (measured on glibc 2.41 and musl 1.2.6). Writing the dotted
   name first would make them return a name with a trailing dot, which differs from
   today's output. Writing the undotted name first avoids that.
4. **Duplicate addresses in Go.** Go's pure resolver returns the address twice when a
   line lists both forms. This is cosmetic.

## Design Details

Introduce a new feature gate, `RelaxedHostAliasesValidation`, disabled by default in
alpha.

When the feature gate is disabled, `hostAliases[].hostnames` entries continue to be
validated exactly as they are today (DNS-1123 subdomain, no trailing dot).

When the feature gate is enabled, an entry may additionally carry a single trailing
dot. Validation strips at most one trailing `.` before running the existing DNS-1123
subdomain check against the remainder, so this is additive: every value that is
valid today remains valid.

`hostAliases` is not part of the mutable subset of the pod spec: a pod update that
actually changes the value of `hostAliases` is rejected outright as an illegal
mutation, independent of this KEP. However, `ValidatePodUpdate` also runs full
content validation of the *unchanged* spec on every update (e.g. a label-only
change), via `validatePodMetadataAndSpec` → `ValidatePodSpec` →
`ValidateHostAliases`, before the immutable-fields check is reached. Without
ratcheting, this means: a pod created with a trailing-dot entry while the gate was
enabled would fail *any* future update, including unrelated ones, once the gate is
disabled, since its existing (unchanged) `hostAliases` value would be re-validated
against the now-stricter rules.

To avoid this, `ValidateHostAliases` is ratcheted on update: when validating an
update, the new `hostAliases` value is only checked against the gate-disabled rules
if it differs from `oldPod.Spec.HostAliases`. If unchanged, validation is skipped
entirely, regardless of gate state, the same pattern used for Ingress's
`backend.service.name` in
[KEP-5311](/keps/sig-network/5311-relaxed-validation-for-service-names#design-details).
An update that does change `hostAliases` is still validated fresh against
whatever rules are active at the time of that update.

### /etc/hosts rendering

When the gate is enabled, `hostsEntriesFromHostAliases` in
`pkg/kubelet/kubelet_pods.go` expands each hostname that ends in a single dot. For a
hostname `h.` it writes `h` immediately before `h.`, unless `h` is already present in
the same entry's `hostnames` list. All other hostnames are written unchanged and in
the order given. When both forms are listed, the user's order is kept.

| `hostnames` in the Pod | Line written to `/etc/hosts` |
|---|---|
| `["api.example.com."]` | `10.10.10.10 api.example.com api.example.com.` |
| `["api.example.com.", "alt"]` | `10.10.10.10 api.example.com api.example.com. alt` |
| `["api.example.com", "api.example.com."]` | `10.10.10.10 api.example.com api.example.com.` |
| `["api.example.com."]`, kubelet gate disabled | `10.10.10.10 api.example.com.` |

The rule is a pure function of the hostnames and only changes the output when a
hostname ends in a dot, so it can be unit tested without a kubelet. It cannot fail:
the existing generator returns bytes with no error path, and the only failures in
`ensureHostsFile` are `os.WriteFile` and `Chmod`, which fail identically for any
content. No fallback path is therefore specified.

The kubelet behavior is controlled by the same `RelaxedHostAliasesValidation` gate,
so operators can keep the previous verbatim rendering on the kubelet independently of
the API server.

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

The `/etc/hosts` rendering in `pkg/kubelet` gets table-driven tests covering the rows
in the table above, a hostname of `.` (never expanded into an empty token), and the
gate-disabled verbatim path.

##### Integration tests

**Alpha:**

1. With the feature gate enabled, create a Pod with a trailing-dot `hostAliases`
   entry and confirm it is accepted.
2. With the feature gate disabled, create the same Pod and confirm it is rejected
   with the existing validation error.
3. With the feature gate enabled, confirm a `hostAliases` entry without a trailing
   dot is still accepted (no regression on the existing behavior).
4. Create a Pod with a trailing-dot `hostAliases` entry while the gate is enabled,
   disable the gate, then update an unrelated field (e.g. a label) on that Pod and
   confirm the update succeeds (ratcheting: the unchanged `hostAliases` value is not
   re-validated).
5. With the gate disabled and the same Pod from (4), attempt to change the
   `hostAliases` value itself and confirm that update is rejected (a real change to
   the field is validated fresh against the currently active rules).

##### e2e tests

**Alpha:**

- Create a Pod with `hostAliases` hostnames `["api.example.test."]` on both a
  glibc-based and a musl-based (`alpine`) container image. From inside the container,
  confirm that `getent hosts api.example.test` and `getent hosts api.example.test.`
  both return the alias IP, and that `getent hosts <alias IP>` returns
  `api.example.test` without a trailing dot. This checks real resolution, not just
  that the API server accepted the object, and directly covers the libc risk noted
  above. The expected results are those of the measurements linked in the
  Motivation.

### Graduation Criteria

#### Alpha

- Feature implemented behind `RelaxedHostAliasesValidation` in kube-apiserver and
  kubelet, disabled by default.
- Initial e2e tests completed and enabled, including the glibc/musl resolution check.

#### Beta

- Integration and e2e tests completed and enabled.
- No open issues from alpha usage.

#### GA

- Time passes with no major objections.
- Promote the e2e resolution test to conformance, if applicable.

### Upgrade / Downgrade Strategy

Upgrade: existing pods are unaffected. Newly created pods can use the relaxed
format once the gate is enabled.

Downgrade: a pod already running with a trailing-dot `hostAliases` entry keeps
running unaffected, and, thanks to the update-time ratcheting described in
[Design Details](#design-details), can still be updated on unrelated fields (e.g.
labels) without that pre-existing value being re-validated. New pod creation using
the relaxed format, and any update that actually *changes* `hostAliases`, will fail
once the gate is disabled, the same downgrade behavior as KEP-5311. Disabling the gate
on the kubelet makes it write trailing-dot names verbatim (`IP name.`), which resolves
absolute lookups but not relative ones; nothing breaks.

### Version Skew Strategy

Two components are involved: kube-apiserver validates the field and kubelet renders it
into `/etc/hosts`. kubelet does not validate the hostname string, so there is no
validation skew.

- **New kube-apiserver, old kubelet:** an accepted trailing-dot name reaches a kubelet
  that writes it verbatim, producing `IP name.`. Absolute lookups resolve, relative
  lookups of the same name do not (measured on glibc and musl). Nothing fails, and the
  pod gets that behavior until the kubelet is upgraded.
- **Old kube-apiserver, new kubelet:** trailing-dot names are rejected at admission,
  so the kubelet never renders one and its output is unchanged.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: RelaxedHostAliasesValidation
  - Components depending on the feature gate: kube-apiserver, kubelet

###### Does enabling the feature change any default behavior?

No. It only affects pods whose `hostAliases` contain a trailing-dot hostname, which
are rejected when the gate is disabled.

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
- 2026-10-02: Design updated to render both name forms into `/etc/hosts`, based on
  libc and ndots measurements (linked in the Motivation).

## Drawbacks

Downstream tooling (admission webhooks, policy engines, client-side generators) that
independently re-validates `hostAliases` entries against the current DNS-1123
subdomain shape could reject the new trailing-dot format until updated. The kubelet
also gains a small, gated rendering rule.

## Alternatives

- **Bake the hosts entry into the container image.** Works for a single fixed
  environment, but defeats the purpose of `hostAliases` being a runtime-configurable
  field and breaks image portability across clusters/environments.
- **Use `dnsConfig`/`dnsPolicy: None` with a custom resolver configuration.** This
  operates at a different layer (DNS resolution and search domains, not static hosts
  entries) and does not provide an equivalent to a static `/etc/hosts` entry; it also
  requires taking over DNS resolution entirely rather than adding one static entry.
- **Write only the dotted name (this KEP's original design).** Rejected after
  measurement: a line containing only `name.` is returned for the absolute lookup but
  not the relative one, on every libc tested.
- **Alias only the undotted name (what is possible today).** Does not satisfy absolute
  lookups, which is the gap this KEP exists to close.
- **Write one line per form, dotted line first
  ([#6075](https://github.com/kubernetes/enhancements/pull/6075)).** Resolves both
  forms, but reverse lookups return the dotted name because it is first on the
  matching line, and the address is listed twice. A single line with the undotted
  name first resolves both forms without those effects. #6075 also describes a
  fallback to a legacy rendering on write errors; the generator has no error path (see
  [/etc/hosts rendering](#etc-hosts-rendering)), so this KEP specifies none.
