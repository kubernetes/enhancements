---
title: Support Out-of-Tree IBM Cloud Provider
authors:
  - @rtheis
owning-sig: sig-cloud-provider
participating-sigs:
  - sig-ibmcloud
reviewers:
  - @andrewsykim
  - @spzala
approvers:
  - @andrewsykim
  - @spzala
editor: TBD
creation-date: 2019-01-25
last-updated: 2019-04-23
status: provisional

---

# Supporting Out-of-Tree IBM Cloud Provider

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Release Signoff Checklist](#release-signoff-checklist)
* [Terms](#terms)
* [Summary](#summary)
* [Motivation](#motivation)
   * [Goals](#goals)
   * [Non-Goals](#non-goals)
* [Proposal](#proposal)
   * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
   * [Risks and Mitigations](#risks-and-mitigations)
* [Design Details](#design-details)
   * [Test Plan](#test-plan)
   * [Graduation Criteria](#graduation-criteria)
   * [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
   * [Version Skew Strategy](#version-skew-strategy)
* [Implementation History](#implementation-history)

## Release Signoff Checklist

- [X] k/enhancements issue in release milestone and linked to KEP (https://github.com/kubernetes/enhancements/issues/671)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Terms

- **CCM:** [Cloud Controller Manager](https://kubernetes.io/docs/reference/command-line-tools-reference/cloud-controller-manager/) - The controller manager responsible for
  running cloud provider dependent logic, such as the service and route
  controllers.
- **KCM:** [Kubernetes Controller Manager](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/) - The controller manager responsible
  for running generic Kubernetes logic, such as job and node_lifecycle controllers.
- **IKS:** [IBM Cloud Kubernetes Service](https://www.ibm.com/cloud/container-service)
- **IBM CCM:** IBM cloud provider using CCM out-of-tree cloud provider architecture.
- **IBM KCM:** IBM cloud provider using KCM in-tree cloud provider architecture.
  This cloud provider is currently used by IKS.

## Summary

Build support for the out-of-tree IBM cloud provider based on the current
IBM cloud provider used by IBM's managed Kubernetes service (IKS). This involves
a well-tested version of IBM CCM that has feature parity to the current
IBM KCM.

Unlike many other cloud providers, IBM's cloud provider was never in-tree due
to the deprecation of in-tree providers at the time IBM's cloud provider was
written.

## Motivation

Motivation for supporting out-of-tree providers can be found in
[KEP-20180530-cloud-controller-manager](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/20180530-cloud-controller-manager.md).
This KEP is specifically tracking progress for the IBM cloud provider.

### Goals

* Develop, test, document and release IBM CCM.
* Kubernetes clusters running on IBM Cloud should be running IBM CCM.
* IKS clusters should be running IBM CCM.
* Clusters should be able to easily migrate from IBM KCM to IBM CCM.

### Non-Goals

* Re-architecting the IBM cloud provider beyond transition from IBM KCM to IBM CCM.
* Open source other components of IKS.

## Proposal

### Implementation Details/Notes/Constraints

IBM has already completed most of the IBM CCM enablement work. Furthermore, IBM
has obtained the necessary internal approvals to open source the IBM cloud
provider. The remaining implementation details covered by this KEP are for
IBM CCM implementation, test and documentation.

IBM KCM only implements the node and service controllers (router controller
is handled by a [Calico](https://www.projectcalico.org/) overlay). These
components are ready for the switch to the IBM CCM per
[KEP-20180530-cloud-controller-manager](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/20180530-cloud-controller-manager.md).
This leaves implementation of the actual CCM command along with development of a
build and release process for the IBM CCM artifacts. The CCM command and artifact
implementation should be straight-forward. As for build and release, IBM CCM will
deliver version `X.Y.*` releases that are compatible with Kubernetes version
`X.Y.*` releases. The patch version may not align. In general, branching,
tagging and the release process will be similar to that of Kubernetes.

IKS already has a CI/CD pipeline that runs conformance testing using both
methods outlined in
[KEP-0018-testgrid-conformance-e2e](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/0018-testgrid-conformance-e2e.md).
This testing is done with IBM KCM and against every patch version of Kubernetes
supported by IKS, which may include pre-release versions but does not include
master. To close the remaining gaps, the IKS CI/CD pipeline will be updated to
implement following requirements:
- Run conformance testing using IBM CCM
- Report conformance test results to [Testgrid](https://github.com/kubernetes/test-infra/tree/master/testgrid)
- Ensure every patch version of Kubernetes is tested independent of IKS support

The current [IKS documentation](https://cloud.ibm.com/docs/containers) implements
most of the load balancer service documentation requirements per
[KEP-20180731-cloud-provider-docs](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/20180731-cloud-provider-docs.md).
The remaining gaps must be filled and included within the IBM cloud provider
repository. This will include IBM cloud provider documentation for the following:
- DevOps
- Service annotations, labels, etc.
- Node annotations, labels, etc.

Migrating existing clusters from IBM KCM to IBM CCM will follow the recommended
process developed by the [Cloud Provider SIG](https://github.com/kubernetes/community/tree/master/sig-cloud-provider).

### Risks and Mitigations

The biggest risk is migrating existing clusters from IBM KCM to IBM CCM.
This risk will be mitigated by only supporting migrations on a specific
Kubernetes `major.minor` release boundary with all necessary master and
worker node patches. Furthermore, cloud controller loops may be temporarily
disabled during the migration.

## Design Details

### Test Plan

Beyond the test plans referenced earlier in this KEP, IKS submits
[conformance test results](https://github.com/cncf/k8s-conformance) for every
Kubernetes `major.minor.patch` version supported by IKS. It is expected that IKS
will adopt IBM CCM and continue submitting conformance tests results as-is
done today.

### Graduation Criteria

**Beta:** IBM CCM will graduate to beta after sufficient adoption and usage
in IBM Cloud. In addition, graduation to beta will require a solid migration
path from IBM KCM to IBM CCM. Beta is targeted for Kubernetes v1.16.

**GA:** IBM CCM will graduate to GA after widespread adoption and usage in
IBM Cloud. Widespread adoption will likely be lead by IKS usage. GA is targeted
for Kubernetes v1.17.

### Upgrade / Downgrade Strategy

Upgrades will align with the Kubernetes upgrade strategy.

Downgrades won't be supported.

### Version Skew Strategy

Version skew will align with the Kubernetes version skew strategy for upgrades,
which is at most n-2.

## Implementation History

- 2019-04-23: Initial KEP sent out for review.
