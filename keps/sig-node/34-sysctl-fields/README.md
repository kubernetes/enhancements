# KEP-34: Promote sysctl annotations to fields

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals/Non-Goals](#goalsnon-goals)
- [Proposal](#proposal)
  - [Promote annotations to fields (beta)](#promote-annotations-to-fields-beta)
  - [Promote <code>--experimental-allowed-unsafe-sysctls</code> kubelet flag to kubelet config api option](#promote---experimental-allowed-unsafe-sysctls-kubelet-flag-to-kubelet-config-api-option)
  - [Gate the feature](#gate-the-feature)
  - [User Stories (Optional)](#user-stories-optional)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [Graduation](#graduation)
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
- [Drawbacks / Alternatives](#drawbacks--alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This proposal aims at extending the current pod specification with support
for namespaced kernel parameters (sysctls) set for each pod.

See the [abstract and motivation](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/sysctl.md#abstract) from the original proposal in v1.4.

## Motivation

See the original design proposal's [motivation section](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/sysctl.md#motivation).

As mentioned in [contributors/devel/api_changes.md#alpha-field-in-existing-api-version](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#alpha-field-in-existing-api-version):

> Previously, annotations were used for experimental alpha features, but are no longer recommended for several reasons:
>
>    They expose the cluster to "time-bomb" data added as unstructured annotations against an earlier API server (https://issue.k8s.io/30819)
>    They cannot be migrated to first-class fields in the same API version (see the issues with representing a single value in multiple places in backward compatibility gotchas)
>
> The preferred approach adds an alpha field to the existing object, and ensures it is disabled by default:
>
> ...

The annotations as a means to set `sysctl` are no longer necessary.
The original intent of annotations was to provide additional description of Kubernetes
objects through metadata.
It's time to separate the ability to annotate from the ability to change sysctls settings
so a cluster operator can elevate the distinction between experimental and supported usage
of the feature.


### Goals/Non-Goals

See: original [constraints and assumptions](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/sysctl.md#constraints-and-assumptions)

## Proposal

See the [original design proposal](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/sysctl.md#proposed-design) for alpha.

### Promote annotations to fields (beta)

Setting the `sysctl` parameters through annotations provided a successful story
for defining better constraints of running applications.
The `sysctl` feature has been tested by a number of people without any serious
complaints. Promoting the annotations to fields (i.e. to beta) is another step in making the
`sysctl` feature closer towards the stable API.

Currently, the `sysctl` provides `security.alpha.kubernetes.io/sysctls` and `security.alpha.kubernetes.io/unsafe-sysctls` annotations that can be used
in the following way:
  ```yaml
  apiVersion: v1
  kind: Pod
  metadata:
    name: sysctl-example
    annotations:
      security.alpha.kubernetes.io/sysctls: kernel.shm_rmid_forced=1
      security.alpha.kubernetes.io/unsafe-sysctls: net.ipv4.route.min_pmtu=1000,kernel.msgmax=1 2 3
  spec:
    ...
  ```

  The goal is to transition into native fields on pods:

  ```yaml
  apiVersion: v1
  kind: Pod
  metadata:
    name: sysctl-example
  spec:
    securityContext:
      sysctls:
      - name: kernel.shm_rmid_forced
        value: 1
      - name: net.ipv4.route.min_pmtu
        value: 1000
        unsafe: true
      - name: kernel.msgmax
        value: "1 2 3"
        unsafe: true
    ...
  ```

The `sysctl` design document with more details and rationals is available at [design-proposals/node/sysctl.md](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/sysctl.md#pod-api-changes)

* Introduce native `sysctl` fields in pods through `spec.securityContext.sysctl` field as:

  ```yaml
  sysctl:
  - name: SYSCTL_PATH_NAME
    value: SYSCTL_PATH_VALUE
    unsafe: true    # optional field
  ```

* Introduce native `sysctl` fields in [PSP](https://kubernetes.io/docs/concepts/policy/pod-security-policy/) as:

  ```yaml
  apiVersion: v1
  kind: PodSecurityPolicy
  metadata:
    name: psp-example
  spec:
    sysctls:
    - kernel.shmmax
    - kernel.shmall
    - net.*
  ```

  More examples at [design-proposals/node/sysctl.md#allowing-only-certain-sysctls](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/sysctl.md#allowing-only-certain-sysctls)

### Promote `--experimental-allowed-unsafe-sysctls` kubelet flag to kubelet config api option

As there is no longer a need to consider the `sysctl` feature experimental,
the list of unsafe sysctls can be configured accordingly through:

```go
// KubeletConfiguration contains the configuration for the Kubelet
type KubeletConfiguration struct {
  ...
  // Whitelist of unsafe sysctls or unsafe sysctl patterns (ending in *).
  // Default: nil
  // +optional
  AllowedUnsafeSysctls []string `json:"allowedUnsafeSysctls,omitempty"`
}
```

Upstream issue: https://github.com/kubernetes/kubernetes/issues/61669

### Gate the feature

As the `sysctl` feature stabilizes, it's time to gate the feature [1] and enable it by default.

* Expected feature gate key: `Sysctls`
* Expected default value: `true`

With the `Sysctl` feature enabled, both sysctl fields in `Pod` and `PodSecurityPolicy`
and the whitelist of unsafed sysctls are acknowledged.
If disabled, the fields and the whitelist are just ignored.

[1] https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/

### User Stories (Optional)

See also: [original sysctl proposal](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/sysctl.md#abstract-use-cases)

* As a cluster admin, I want to have `sysctl` feature versioned so I can assure backward compatibility
  and proper transformation between versioned to internal representation and back..
* As a cluster admin, I want to be confident the `sysctl` feature is stable enough and well supported so
  applications are properly isolated
* As a cluster admin, I want to be able to apply the `sysctl` constraints on the cluster level so
  I can define the default constraints for all pods.

### Notes/Constraints/Caveats

Extending `SecurityContext` struct with `Sysctls` field:

```go
// PodSecurityContext holds pod-level security attributes and common container settings.
// Some fields are also present in container.securityContext.  Field values of
// container.securityContext take precedence over field values of PodSecurityContext.
type PodSecurityContext struct {
    ...
    // Sysctls is a white list of allowed sysctls in a pod spec.
    Sysctls []Sysctl `json:"sysctls,omitempty"`
}
```

Extending `PodSecurityPolicySpec` struct with `Sysctls` field:

```go
// PodSecurityPolicySpec defines the policy enforced on sysctls.
type PodSecurityPolicySpec struct {
    ...
    // Sysctls is a white list of allowed sysctls in a pod spec.
    Sysctls []Sysctl `json:"sysctls,omitempty"`
}
```

Following steps in [devel/api_changes.md#alpha-field-in-existing-api-version](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#alpha-field-in-existing-api-version)
during implementation.

Validation checks implemented as part of [#27180](https://github.com/kubernetes/kubernetes/pull/27180).

### Risks and Mitigations

We need to assure backward compatibility, i.e. object specifications with `sysctl` annotations
must still work after the graduation.

## Design Details

All of the above details were copied out of earlier proposals. For graduation, the PRR template below is completed.

### Test Plan

- Unit tests and e2es for all applicable changes.
- Any required conformance tests for graduation.

### Graduation Criteria

#### Alpha

* add sysctl support to pods
* e2e tests

Alpha since 1.4.

#### Beta

* API changes allowing to configure the pod-scoped `sysctl` via `spec.securityContext` field.
* API changes allowing to configure the cluster-scoped `sysctl` via `PodSecurityPolicy` object
* feature gate enabled by default

Beta since 1.11.

#### Graduation

* Promote `--experimental-allowed-unsafe-sysctls` kubelet flag to kubelet config api option
* lock feature gate on

### Upgrade / Downgrade Strategy

There are [e2es](https://github.com/kubernetes/kubernetes/blob/28e2e12b887fe082929d3ece4b3cbd572dc15d39/test/e2e/upgrades/sysctl.go) for sysctl behaviour on upgrades.

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: Sysctls
  - Components depending on the feature gate: kubelet, apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

No. Enabling the feature allows the use of sysctls.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, disable the feature flag.

###### What happens if we reenable the feature if it was previously rolled back?

Feature will become available again on the component.

###### Are there any tests for feature enablement/disablement?

Not currently. Feature has defaulted to on since 1.11; graduation criteria would lock feature to on.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes: https://github.com/kubernetes/kubernetes/blob/28e2e12b887fe082929d3ece4b3cbd572dc15d39/test/e2e/upgrades/sysctl.go

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

No metric currently exists. Feature flag will be set to on and Pod or PSP specifications will include sysctl fields set.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A, not a service.

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

N/A, not a service.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

Underlying kernel support for sysctls.

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes: pods and PSPs have new fields for sysctl values.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Feature is an API field on pod specification; kubelets behave as usual when API server/etcd are unavailable.

###### What are other known failure modes?

- https://github.com/kubernetes/kubernetes/issues/72593 
- https://github.com/kubernetes/kubernetes/issues/74151

There may be some follow-ups required to improve usability, but I do not
believe this should block graduation.

Any scheduling enhancement we make around a node that is configured to allow
unsafe sysctls would be a distinct feature.

###### What steps should be taken if SLOs are not being met to determine the problem?

SLOs do not apply, N/A.

## Implementation History

- 2017-06-12: [Original design proposal](https://github.com/kubernetes/community/pull/700)
- 2018-05-14: [Update KEP with beta criteria](https://github.com/kubernetes/community/pull/2093)
- 2018-06-06: [Promote sysctl annotations to fields](https://github.com/kubernetes/kubernetes/pull/63717)
- 2018-06-14: [Update sysctls to beta on website](https://github.com/kubernetes/website/pull/8804)
- 2019-07-02: [Add allowed sysctl to KubeletConfiguration](https://github.com/kubernetes/kubernetes/pull/72974)
- 2021-02-08: [Update KEP with final graduation criteria/complete PRR questionnaire](https://github.com/kubernetes/enhancements/pull/2471)
- 2021-02-24: [Sysctls graduated to GA](https://github.com/kubernetes/kubernetes/pull/99158)
- 2021-03-26: [Sysctls added to conformance tests](https://github.com/kubernetes/kubernetes/pull/99734)

## Drawbacks / Alternatives

See also: [original design alternatives and considerations](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/sysctl.md#design-alternatives-and-considerations)

## Infrastructure Needed (Optional)

N/A
