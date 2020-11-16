---
title: Promoting Cloud Provider Labels to GA
authors:
  - "@andrewsykim"
owning-sig: sig-cloud-provider
participating-groups:
  - sig-node
  - sig-storage
reviewers:
  - "@dims"
  - "@liggit"
  - "@msau42"
  - "@saad-ali"
  - "@thockin"
approvers:
  - "@thockin"
  - "@liggit"
editor: TBD
creation-date: 2019-02-15
last-updated: 2019-02-15
status: implementable
see-also:
  - "/keps/sig-node/20190130-node-os-arch-labels.md"
---

# Promoting Cloud Provider Labels to GA

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
<!-- /toc -->

## Release Signoff Checklist

- [X] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

When node and volume resources are created in Kubernetes, labels should be applied based on the underlying cloud provider of the Kubernetes cluster.
These labels contain cloud provider information that may be critical to some advanced features (mainly scheduling).
When these labels were first introduced, they were prefixed with "beta" as the maturity and usage of these labels were not known at the time.

Today, the cloud provider specific labels are:
* `beta.kubernetes.io/instance-type`
* `failure-domain.beta.kubernetes.io/zone`
* `failure-domain.beta.kubernetes.io/region`

This KEP proposes to remove the beta labels and replace them with their GA equivalents:
* `node.kubernetes.io/instance-type`
* `topology.kubernetes.io/zone`
* `topology.kubernetes.io/region`

## Motivation

The labels mentioned above are consumed by almost all Kubernetes clusters that have cloud providers enabled. Given their maturity and widespread use, we should
promote these labels from beta to GA.

### Goals

* promote cloud provider node/volume labels to GA with minimal visible changes to users.
* remove the usage of "beta" cloud provider node/volume labels without breaking compatibility guaranetees. This will span many Kubernetes versions as per the Kubernetes deprecation policy.

### Non-Goals

* introducing more labels
* changing the behaviour of these labels within the Kubernetes system.

## Proposal

In order to promote these labels to GA safely, there will be a period in which both the "beta" and "GA" labels are applied to node and volume objects.
This is required in order to maintain backwards compatibility as many clusters rely on the beta labels today.

For the case of existing resources, keeping the beta labels is a requirement in order for existing workloads to behave as expected. A mechanism to populate existing resources
with the new GA versions of the labels will also be needed. For the case of new resources, both labels are still required as workloads may still consume the beta labels in some other resource
that was not updated yet to use the GA labels. One possible example is where a deployment may still use the beta zone label (`failure-domain.beta.kubernetes.io/zone`) as a
nodeSelector and not applying the beta labels to new nodes would mean new nodes in that zone would not be considered when pods are being scheduled.

### Implementation Details/Notes/Constraints [optional]

Here is a break down of the implementation steps:

1) [v1.17] update components to apply both the GA and beta labels to nodes & volumes.
2) [v1.17] deprecate the beta labels.
3) [v1.17] update the appropriate release notes & documentation to promote the use of GA labels over beta labels.
4) [v1.18] continue to promote usage of GA labels over beta labels.
5) [v1.19] continue to promote usage of GA labels over beta labels.
6) [v1.20] continue to promote usage of GA labels over beta labels.
7) [v1.21] components that consume the beta labels will be updated to only check for GA labels.
8) [v1.21] stop applying beta labels to new resources, existing resources will continue to have those labels unless manually removed.

### Risks and Mitigations

* duplicate labels that do the same thing can be confusing/annoying for users
* post v1.18 Kubernetes clusters may have danging labels that provide no function
* improper handling of labels can lead to critical bugs in scheduling / volume topology / node registration / etc.

## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

TBD

### Graduation Criteria

Labels for zones, regions and instance-type have been beta since v1.3, they are widely adopted by Kubernetes users.


### Upgrade / Downgrade Strategy

There is relatively low risk when it comes to upgrade / downgrade of clusters with respect to this enhancement.
Because we will apply both beta and GA labels to resources, a downgrade scenario would result in resources having a new label that may not necessarily be
consumed by anything else in the system yet. With the beta labels still in place, any features relying on these labels should continue to function as expected.
When we stop applying beta labels to resources in v1.18, newly created resources will have the GA label _only_, but any existing resources carried over will have both
the GA labels and the beta labels. In this scearnio, a downgrade would only cause issues if a new node/volume resource was created
in the newer version (v1.18 or greater) and other resources in the cluster still referenced the deprecated beta resource after a downgrade.
This edge case would only occur if users have not replaced usage of the beta labels with GA labels by v1.18.

### Version Skew Strategy

No issues should arise from version skew assuming users do not replace usage of beta and GA labels until after all Kubernetes components are upgraded.
In the event that users attempt to update a workload to consume the GA labels in the middle of a cluster upgrade, workloads should eventually run as
expected once the upgrade is complete.

## Implementation History

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design

## Drawbacks [optional]

There are valid reasons why we should not move forward with this KEP. Replacing labels requires a lot of work to ensure plenty of time for deprecating warnings
and that no existing behavior has changed. There is also a chance that users may choose (for whatever reason) to never replace beta labels with GA labels until something in the
Kubernetes cluster no longer works. This poses a risk to Kubernetes users that may indicate this effort is not worth the risk/time involved.

## Alternatives [optional]

* continue to use beta labels until a V2 of Nodes / PersistentVolumes is developed and breaking changes are acceptable.
* continue to use existing beta labels forever
