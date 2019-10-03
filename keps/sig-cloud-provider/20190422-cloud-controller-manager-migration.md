---
title: Cloud Controller Manager Migration
authors:
  - "@andrewsykim"
owning-sig: sig-cloud-provider
participating-sigs:
  - sig-apimachinery
reviewers:
  - "@cheftako"
  - "@mcrute"
approvers:
  - "@lavalamp"
editor: TBD
creation-date: 2019-04-22
last-updated: 2019-04-22
status: implementable
see-also:
  - "/keps/sig-cloud-provider/20180530-cloud-controller-manager.md"
---

# Cloud Controller Manager Migration

## Table of Contents

* [Cloud Controller Manager Migration](#cloud-controller-manager-migration)
  * [Table of Contents](#table-of-contents)
  * [Release Signoff Checklist](#release-signoff-checklist)
  * [Summary](#summary)
  * [Motivation](#motivation)
     * [Goals](#goals)
     * [Non-Goals](#non-goals)
  * [Proposal](#proposal)
     * [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
	* [Splitting the Main Run Func](#splitting-the-main-run-func)
	* [Process Handling when Lock is Lost](#process-handling-when-lock-is-lost)
     * [Risks and Mitigations](#risks-and-mitigations)
  * [Design Details](#design-details)
     * [Test Plan](#test-plan)
     * [Graduation Criteria](#graduation-criteria)
	* [Examples](#examples)
	   * [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
	   * [Beta -&gt; GA Graduation](#beta---ga-graduation)
     * [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
     * [Version Skew Strategy](#version-skew-strategy)
  * [Implementation History](#implementation-history)
  * [Alternatives [optional]](#alternatives-optional)
  * [Infrastructure Needed [optional]](#infrastructure-needed-optional)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Support a migration process that **safely** migrates existing Kubernetes clusters using the in-tree cloud providers (via kube-controller-manager)
to their out-of-tree equivalents (via cloud-controller-manager). The migration process laid out here should be reusable by other components in the future if desired.

## Motivation

SIG Cloud Provider is in the process of migrating the cloud specific code from the core Kubernetes tree to external packages
and removing them from the kube-controller-manager, where they are today embedded. Once the extraction has been completed, existing users
running older versions of Kubernetes need a process to migrate their existing clusters to use the new cloud-controller-manager component
with minimal risk. This KEP proposes a mechanism in which HA clusters can safely migrate “cloud specific” controllers between the
kube-controller-manager and the cloud-controller-manager via a shared resource lock between the two components. The pattern proposed
in this KEP should be reusable by other components in the future if desired.

### Goals

* Define a clear migration process for existing clusters to use the cloud-controller-manager instead of the kube-controller-manager for cluster integrations with any cloud provider.
* The migration process should work for single node control planes and HA control planes with minimal risk to users.

### Non-Goals

* Removing cloud provider code from the core Kubernetes tree, this effort is separate and is covered in [KEP-removing-in-tree-providers](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/20190125-removing-in-tree-providers.md)
* Improving the scalability of controllers by running controllers across multiple components (with or without leader election).
* Migrating cloud-based volume plugins to CSI. This is a separate effort led by SIG Storage. See [this proposal](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/csi-migration.md) for more details.

## Proposal

Today, the kube-controller-manager (KCM) and cloud-controller-manager (CCM) run independent of each other.
This means that both the KCM or the CCM can run the cloud specific control loops for a given cluster.
For a highly available control plane to migrate from running only the KCM to running both the KCM and
the CCM requires that only one process in the control plane is running the cloud specific controllers.
This becomes non-trivial when introducing the CCM that runs overlapping controllers with the KCM.

For environments that can tolerate downtime, the control plane can be taken down in order to
reconfigure components to use the CCM, and then bring the control plane back up. This ensures that only
1 component can be running the set of cloud controllers. For environments that have stricter requirements
for uptime, some level of coordination is required between the two components to ensure that upgrading
control planes does not result in running the same controller in more than one place while also accounting for version skew.

In order to coordinate the cloud-specific controllers across the KCM and the CCM, this KEP proposes a
_primary_ and N configurable _secondary_ (a.k.a migration) leader election locks in the KCM and the CCM.
The primary lock represents the current leader election resource lock in the KCM and the CCM. The set of
secondary locks are defined by the cloud provider and run in parallel to the primary locks. For a migration
lock defined by the cloud provider, the cloud provider also determines the set of controllers run within the
migration lock and the controller manager it should run in - either the CCM or the KCM.

The properties of the migration lock are:
  * must have a unique name
  * the set of controllers in the lock is immutable.
  * no two migration locks should have overlapping controllers
  * the controller manager where the lock runs can change across releases.
  * for a minor release it should run exclusively in one type of controller manager - KCM or CCM.
  * given a minor release, all subsequent patch releases shouldn't change the migration locks
  
During migration, either the KCM or CCM may have multiple migration locks, though for performance reasons no more than 2 locks is recommended.

Let's say we are migrating the service, route, and nodeipam controllers from the KCM to the CCM across Kubernetes versions, say v1.17 to v1.18.
In v1.17, the cloud provider would define a new migration lock called `cloud-network-controller-migration` which specifies those controllers to run
inside the KCM (see Figure 1). As a result, in v1.17 those controllers would run in the KCM but under the `cloud-network-controller-migration` leader election.
To migrate to the CCM for v1.18, the cloud provider would update the `cloud-network-controller-migration` lock to now run in the CCM (see Figure 2).
During a control plane upgrade, the cloud network controllers may still run in one of the KCMs that are still on v1.17. A 1.17 KCM holding the lock
will prevent any of the v1.18 CCMs from claiming the lock. When the current holder of the lock goes down, one of the controller managers eligible will acquire lock.
Then in 1.19 the cloud provider would delete the `cloud-network-controller-migration` lock and run using the `cloud-controller-manager` lock for the migrated controllers.

<br/>
<br/>
<br/>

![example network controllers migration v1.17](/keps/sig-cloud-provider/images/migrating-cloud-controllers-v1-17.png)
<br/>
**Figure 1**: Example of migrating cloud network controllers in v1.17

<br/>
<br/>
<br/>

![example network controllers migration v1.18](/keps/sig-cloud-provider/images/migrating-cloud-controllers-v1-18.png)
<br/>
**Figure 2**: Example of migrating cloud network controllers in v1.18



### Implementation Details/Notes/Constraints [optional]

The cloud-provider needs to would need to implement an interface, that for a given controller returns the version of migration

### Risks and Mitigations

One of the risks related to this KEP is the KCM and the CCM running the same controllers if the implementation is not well tested.
To avoid this we should have e2e tests that covers the happy path and the edge cases

## Design Details

### Test Plan

test strategy will include:

- unit Testing
- integration test between KCM and KAS
- e2e testing with providers supporting the migration

### Graduation Criteria

##### Alpha -> Beta Graduation

the migration library is working and tested with at least one cloud provider

### Version Skew Strategy

The CCM's version shipping the migration (i.e. running under a new lock) should only support running
with a KCM that has the migration lock introduced for these controllers.

## Implementation History

- KEP merged as provisional
- KEP merged as implementable in 1.17
