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
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [X] KEP approvers have set the KEP status to `implementable`
- [NA] Design details are appropriately documented
- [X] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [NA] Graduation criteria is in place
- [X] "Implementation History" section is up-to-date for milestone
- [NA] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [NA] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


## Summary

The mount library in k8s/utils/mount is shared by both in-tree volume plugins and external CSI drivers. We propose moving this library from k8s/utils to a separate k/k staging repo in order to leverage existing k/k e2e tests and infrastructure, and make it easier to backport fixes to the mount library to Kubernetes

## Motivation

The main context is that the mount library is useful for csi drivers. Main motivations are: 
  * We can't backport fixes to the mount library to older Kubernetes releases without potentially pulling in a number of unrelated and potentially breaking changes in other areas.
  * There have been a few instances of breaking changes being made in k8s/utils/mount, but not detected until after we try to update vendor in Kubernetes. We're not able to take advantage of the tests and infrastructure we have in k/k.

### Goals

To move the mount library into a location that can be shared by kubernetes and external CSI drivers while still being easy to maintain and test. 

### Non-Goals



## Proposal

Move the code and maintain under k/k similar to csi-translation-lib. The mount library repo can be one level below staging/src/k8s.io as mount or mount-utils.

### Risks and Mitigations

None, as the code should just compile and work fine from the new location.

## Design Details

### Test Plan

All existing e2e tests and unit test should work fine and will go through normal PR testing.

### Graduation Criteria

None

### Upgrade / Downgrade Strategy

No upgrade/downgrade concerns.

### Version Skew Strategy

None

## Implementation History

2020-05-08: KEP opened
2020-05-08: KEP marked implementable
1.20: mount library moved to staging, all callers except cadvisor switched

## Alternatives

We can have a separate repo under k8s for the library, but the main drawbacks are that we would still need to invest heavily to build and maintain tests and infrastructure with the same coverage that we have in k/k.

