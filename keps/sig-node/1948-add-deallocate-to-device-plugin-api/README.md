# KEP-20201808: Add Deallocate and PostStopContainer to device plugin API

## Table of Contents

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
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes adding two extra API calls:
- `Deallocate`: (Optional). Which is the opposite of allocate, and is needed to inform device plugins that some devices are no longer being used.
- `PostStopContainer`: (Optional). Which allow the device plugins to do device cleanup, driver unloading, and any other actions that may be needed.

Since both additions are optional, existing device plugins should continue functioning properly with no needed modifications. Only device plugins that wish to utilize the new API calls will need to be modified.

## Motivation

The following are some use cases and motivations for the proposed change:
- `PostStopContainer`:
  - For use with some devices like FPGAs. Devices like these will need to be cleaned up (i.e. de-programmed) after each use. Otherwise they run the possibility of 2 risks:
    * If whatever is programmed on the FPGA is not cleaned up, it will keep running and consuming power for no reason, on a large scale (datacenter scale) this is unacceptable.
    * If whatever is programmed on the FPGA has network access, it runs the risk of continuing to send and respond to packets and pollute the network.
  - For dynamically binding/unbinding drivers for the devices as needed.
- `Deallocate`:
  - For use with complex device plugins that require tracking the state of their devices and learning when they are no longer in use. For example, multi modal devices. Multi modal devices can operate in more than one mode of operation, and thus have to be advertised by the device plugin as two separate devices, and the device plugin has to take care to stop advertising a device when its being used in the other mode, and so on. An example of multi modal devices is also using FPGAs in the following 2 modes:
    * Use the entire FPGA as a device
    * Split the FPGA between multiple users, essentially advertising one FPGA as multiple smaller FPGAs.
    * A device plugin needs to know when a full FPGA will stop being used so it can go back to advertise the FPGA partitions, and vice versa.
  - To maintain the same logical splitting of `Allocate` and `PreStartContainer`

### Goals

- Add the `PostStopContainer` and `Deallocate` API calls to the device plugin API.
- Make the new added API calls optional, as they are not needed for all devices.
- Maintain compatibility with existing device plugins.

### Non-Goals

- Make any modifications to the main API calls of the device plugin API.
- Make changes specific to one type of devices.

## Proposal

The device plugin API includes API calls for:
- `Allocate`: Which is used to instruct device plugins to allocate device(s) to requesting containers.
- `PreStartContainer`: (Optional). Which allow the device plugins to do device initialization, loading drivers, and any other initialization actions that may be needed.

This KEP proposes adding two extra API calls, maintaining the same logical reasoning of the previous two. Those are:
- `Deallocate`: (Optional). Which is the opposite of allocate, and is needed to inform device plugins that some devices are no longer being used (this used to happen silently before)
- `PostStopContainer`: (Optional). Which allow the device plugins to do device cleanup, driver unloading, and any other cleanup actions that may be needed.

Since both additions are optional, existing device plugins should continue functioning properly with no needed modifications. Only device plugins that wish to utilize the new API calls will need to be modified.

### Risks and Mitigations

Only risk is breaking existing device plugins by introducing non optional changes. Can be mitigated by enough test coverage.

## Design Details

- Move `PreStartContainer` in `DeviceManager` to be used as a container lifecycle hook. (This change isn't truly required, but useful for compatibility and organization with the next steps).
- Add `PostStopContainer` and `Deallocate` calls in the DevicePlugin API.
- Add `PostStopContainer` as a container lifecycle hook.
- Add `Deallocate` calls in container manager, taking care to only do so for devices that are no longer in the reuse list.
- Add and modify test cases for both calls.
- Test with existing device plugins to ensure the changes are non-breaking.
- Test with new device plugins utilizing such changes to ensure the changes are working.

### Test Plan

- Unit tests will be updated to include the new API calls
- E2E tests should be added with a sample device plugin for verification.

### Graduation Criteria

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys, specially about the device reuse and possible alternatives.
- Complete implementation for the new API calls.
- Tests are in Testgrid and linked in KEP.

#### Beta -> GA Graduation

- More rigorous testing as needed/discussed by developers.
- Larger scale use/testing by interested users with no reported major bugs.

### Upgrade / Downgrade Strategy

As part of the device plugin API, this will follow the same API versioning system. This means that it is up to an application (a device plugin) to choose the required API version it wants. As long as the cluster has a recent enough (to include the required API version) kubernetes, upgrade or downgrades require no cluster modifications at all, and are decided on the application level.

### Version Skew Strategy

As part of the device plugin API, this will follow the same API versioning system. This means that it is up to an application (a device plugin) to choose the required API version it wants. No version skew issues will arise.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
    - The API versioning meechanism can be usedto enable/disable this feature per application as needed. The feature itself also adds optional API calls, so enabling/disabling the features is not really required.
    - No downtime required for enabling/disabling this feature.

* **Does enabling the feature change any default behavior?**
  No. The feature is optional, no default behavior will change.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  yes. the feature is optional, so not using it should suffice. Fully disabling/rolling back
  can happen by simply using an older version of the API from the application side.

* **What happens if we reenable the feature if it was previously rolled back?**
  No side effects expected.

* **Are there any tests for feature enablement/disablement?**
  No

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  No effect on already running workloads. Feature has to be specifically enabled/requested 
  from the application side.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  No

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
  Currently, as far as I know, there's no way to monitor device plugin API calls (and
  whether or not devices are in use) except by checking logs from the kubelet iself and/or
  user created device plugins.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  N/A (needs checking)

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  N/A

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  N/A

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  No

### Scalability

* **Will enabling / using this feature result in any new API calls?**
  The mere enablement of this feature has no effect. However, using this API by user 
  applications may result in extra calls to the Device Manager API. These extra calls
  only happen at the end of the lifecycle of some containers (those who have previously 
  requested devices).
  The extra API calls originate from the kubelet to the device plugin running on the
  same node. Since they are only happening on the node level, there's no risk of
  congestion, or a need to measure their throughput, etc.

### Troubleshooting

Detection of failures can only be done through using test/mock device plugins, along with checking the kubelet logs. Extra tests according to the test plan mentioned above should help mitigating issues.

## Implementation History

A possible fix has already been submitted as a PR: https://github.com/kubernetes/kubernetes/pull/91190 (outdated, needs rebase)

## Drawbacks

None, as this is an optional feature, it can simply be overlooked when not needed.
