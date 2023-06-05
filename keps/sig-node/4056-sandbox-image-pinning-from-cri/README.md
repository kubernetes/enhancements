# KEP-4056: Sandbox image pinning from CRI

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Proposed API](#proposed-api)
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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The purpose of this KEP is to modify the behavior of the kubelet, allowing Sandbox image pinning from CRI instead of relying on its own sandbox image. Currently, the kubelet adheres to the [`Pinned`](https://github.com/kubernetes/kubernetes/pull/103299) field, avoiding garbage collection of the pinned image, while also maintaining its own sandbox image version. By removing the reliance on the kubelet's sandbox image and emphasizing the use of the `Pinned` field, image management will be simplified and alignment with the desired behavior at the Container Runtime Interface (CRI) level will be achieved.

## Motivation

The kubelet currently abides by the `Pinned` field and avoids garbage collecting the pinned image. However, it also keeps its own sandbox image version and requests the runtime to retain it. This dual approach introduces complexity and potential issues in image management. In particular, issues can arise due to node configuration discrepancies and version skew between the kubelet and the CRI runtime. By modifying the kubelet behavior to rely solely on the `Pinned` field, we can simplify the process and ensure better integration with the underlying CRI implementation.  As a result, it is reasonable to deprecate the use of kubelet's own sandbox image and rely on the underlying CRI implementation to manage sandbox image pinning.

### Goals

- Modify the kubelet behavior to allow Sandbox image pinning from the Container Runtime Interface (CRI) instead of relying on its own sandbox image.
- Simplify image management within the kubelet by aligning it with the desired behavior at the Container Runtime Interface (CRI) level.
- Streamline the codebase by removing the unnecessary logic related to the kubelet's sandbox image.

### Non-Goals

- Introducing new functionality related to sandbox image management.
- Addressing issues outside the scope of the kubelet behavior to rely on the pinned field instead of maintaining its own sandbox image.

## Proposal

We propose deprecating the use of the kubelet's own sandbox image and transitioning to allowing Sandbox image pinning from the underlying Container Runtime Interface (CRI) implementation. Currently, the kubelet utilizes its own sandbox image version and checks with the container runtime to ensure its presence. However, this approach introduces unnecessary complexity and can lead to potential issues in image management within the kubelet. To facilitate a smooth upgrade procedure, we propose implementing a soft upgrade approach for the kubelet. The kubelet will detect if any images use the `Pinned` field, and if so, it will assume that the Container Runtime Interface (CRI) is responsible for managing the sandbox image. As a result, the kubelet will no longer keep track of its own sandbox image. This soft upgrade procedure allows for coexistence of both approaches during the deprecation period, ensuring compatibility and smooth transition for users.

During the alpha phase of the deprecation process, the kubelet will emit deprecation warnings when the `Pinned` field and the sandbox image are used together. This provides early notification to users about the upcoming change and encourages them to migrate to the new approach. 

In the beta phase, we will remove the reference to the kubelet's `sandboxImage` tracking from the codebase. This step signifies the transition towards relying solely on the `Pinned` field for sandbox image retention. Users are encouraged to update their configurations and pod specifications to eliminate dependencies on the kubelet's own sandbox image. 

Finally, in the GA phase, we will completely drop support for the kubelet's own sandbox image. The kubelet will no longer consider its own sandbox image for image retention and will rely exclusively on the CRI implementation.

### Risks and Mitigations

## Design Details

### Proposed API

##### Unit tests

##### Integration tests

##### e2e tests

### Graduation Criteria

The proposed process for deprecating the use of Kubelet's own sandbox image and relying on the `Pinned` field for sandbox image retention. The graduation of the deprecation process will be considered successful when the following conditions are met:

#### Alpha

- [ ] Modify the kubelet code to emit deprecation warnings when the sandbox image is considered in use.
- [ ] The warnings effectively communicate the deprecation and provide guidance on using the `Pinned` field.
- [ ] Validate the functionality and effectiveness of the deprecation warnings.
- [ ] Ensure that the kubelet chooses to ignore its own sandbox image when any image specifies the `Pinned` field.
- [ ] No critical regressions or issues are reported related to the deprecation warnings.

#### Beta

- [ ] Remove the reference to the Kubelet's own sandbox image in the codebase.
- [ ] Ensure that the sandbox image is solely managed through the `Pinned` field.
- [ ] Validate the behavior of the kubelet after removing the reference to the sandbox image.

#### GA

- [ ] Completely remove all code related to the Kubelet's own sandbox image.
- [ ] Verify that the `Pinned` field is the exclusive mechanism for retaining the sandbox image.
- [ ] Conduct comprehensive testing to ensure proper functioning without the deprecated sandbox image code.

### Upgrade / Downgrade Strategy

Before upgrading the kubelet, verify that the CRI runtime version supports the required code for handling sandbox image retention. Once confirmed, proceed with upgrading the kubelet to the desired version. Validate the seamless integration between the upgraded kubelet and CRI runtime, ensuring that the kubelet relies on the `Pinned` field for sandbox image retention.

In the event of a downgrade, the behavior of sandbox image retention remains unaffected as the kubelet continues to manage it internally.

### Version Skew Strategy

N/A.


## Production Readiness Review Questionnaire
### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [x] Other
  - Describe the mechanism: Sandbox image pinning from CRI
  - Will enabling / disabling the feature require downtime of the control
    plane?
     **No**
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
    **No**

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

After the deprecation period, the kubelet will no longer maintain its own sandbox image and will rely solely on the `Pinned` field for sandbox image retention. To continue using the deprecated behavior, users would need to downgrade the kubelet version that still supports the behavior of maintaining its own sandbox image.

###### What happens if we reenable the feature if it was previously rolled back?

We need to upgrade the version of kubelet that removes support for its own sandbox image.

###### Are there any tests for feature enablement/disablement?

N/A.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

There is no explicit escape hatch or fallback mechanism for changing the kubelet behavior to no longer maintain its own sandbox image. However, if the soft upgrade approach is followed, users who upgrade the kubelet without upgrading the Container Runtime Interface (CRI) to a version that supports this functionality will still have a functioning cluster. The only potential risk arises if they subsequently upgrade the kubelet again while the CRI remains unsupported. In such a scenario, where the kubelet supports the new behavior but the CRI does not, there is a higher risk compared to other combinations of component versions and rolling directions. These modifications should be clearly communicated, and users will need to make the necessary adjustments before the deprecation period concludes. While we understand that this may require some adaptation, we believe that this change is essential to simplify image management and ensure better alignment with the desired behavior of relying solely on the `Pinned` field.

###### What specific metrics should inform a rollback?

N/A.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

Yes, as discussed, we will be removing the behavior where the kubelet manages its own sandbox image.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

N/A.

###### How can someone using this feature know that it is working for their instance?

N/A.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A.

### Dependencies

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

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A.

###### What are other known failure modes?

N/A.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A.

## Implementation History

- 2023-06-06: KEP created

## Drawbacks

## Alternatives

- Continuing to use Kubelet's own sandbox image for sandbox image retention would perpetuate the complexities and potential issues discussed earlier.
- Allowing the kubelet to have complete control over the sandbox image would deviate from the typical responsibility of the container runtime in managing images, leading to an unconventional and potentially confusing setup.