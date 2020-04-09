---
title: Cloud Controller Manager Migration
authors:
  - "@andrewsykim"
owning-sig: sig-cloud-provider
participating-sigs:
  - sig-api-machinery
reviewers:
  - "@cheftako"
  - "@nckturner"
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

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [Migration Configuration](#migration-configuration)
    - [Component Flags](#component-flags)
    - [Example Walkthrough of Controller Migration](#example-walkthrough-of-controller-migration)
      - [Enable Leader Migration on Components](#enable-leader-migration-on-components)
      - [Deploy the CCM](#deploy-the-ccm)
      - [Update Leader Migration Config on Upgrade](#update-leader-migration-config-on-upgrade)
      - [Disable Leader Migration](#disable-leader-migration)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Graduation Criteria](#graduation-criteria)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

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

Support a migration process for large scale and highly available Kubernetes clusters using the in-tree cloud providers (via kube-controller-manager and kubelet) to their out-of-tree
equivalents (via cloud-controller-manager).

## Motivation

SIG Cloud Provider is in the process of migrating the cloud specific code from the core Kubernetes tree to external packages
and removing them from the kube-controller-manager, where they are today embedded. Once the extraction has been completed, existing users
running older versions of Kubernetes need a process to migrate their existing clusters to use the new cloud-controller-manager component
with minimal risk.

This KEP proposes a mechanism in which HA clusters can safely migrate “cloud specific” controllers between the
kube-controller-manager and the cloud-controller-manager via a shared resource lock between the two components. The pattern proposed
in this KEP should be reusable by other components in the future if desired.

The migration mechanism outlined in this KEP should only be used for Kubernetes clusters that have _very_ strict requirements on control plane availability.
If a cluster can tolerate short intervals of downtime, it is recommended to update your cluster with in-tree cloud providers disabled, and then deploy
the respective out-of-tree cloud-controller-manager.

### Goals

* Define migration process for large scale, highly available clusters to migrate from the in-tree cloud provider mechnaism, to their out-of-tree equivalents.

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

During migration, either the KCM or CCM may have multiple migration locks, though for performance reasons no more than 2 locks is recommended.

Let's say we are migrating the service, route, and nodeipam controllers from the KCM to the CCM across Kubernetes versions, say v1.17 to v1.18.
In v1.17, the cloud provider would define a new migration lock called `cloud-network-controller-migration` which specifies those controllers to run
inside the KCM (see Figure 1). As a result, in v1.17 those controllers would run in the KCM but under the `cloud-network-controller-migration` leader election.
To migrate to the CCM for v1.18, the cloud provider would update the `cloud-network-controller-migration` lock to now run in the CCM (see Figure 2).
During a control plane upgrade, the cloud network controllers may still run in one of the KCMs that are still on v1.17. A 1.17 KCM holding the lock
will prevent any of the v1.18 CCMs from claiming the lock. When the current holder of the lock goes down, one of the controller managers eligible will acquire lock.

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

#### Migration Configuration

The migration lock will be configured by defining new API types that will then be passed into the KCM and CCM.

```go
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// LeaderMigrationConfiguration provides versioned configuration for all migrating leader locks.
type LeaderMigrationConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// LeaderName is the name of the resource under which the controllers should be run.
	LeaderName string `json:"leaderName"`

	// ControllerLeaders contains a list of migrating leader lock configurations
	ControllerLeaders []ControllerLeaderConfiguration `json:"controllerLeaders"`
}

// ControllerLeaderConfiguration provides the configuration for a migrating leader lock.
type ControllerLeaderConfiguration struct {
	// Name is the name of the controller being migrated
	// E.g. service-controller, route-controller, cloud-node-controller, etc
	Name string `json:"name"`

	// Component is the name of the component in which the controller should be running.
	// E.g. kube-controller-manager, cloud-controller-manager, etc
	Component string `json:"component"`
}
```

#### Component Flags

The LeaderMigrationConfiguration type will be read by the `kube-controller-manager` and the `cloud-controller-manager` via a new flag `--cloud-migration-config` which
accepts a path to a file containing the LeaderMigrationConfiguration type in yaml.

#### Example Walkthrough of Controller Migration

This is an example of how you would migrate all cloud controllers from the CCM to the KCM during a typical cluster version upgrade.

##### Enable Leader Migration on Components

First, define a LeaderMigrationConfiguration resource in a yaml file containing all known cloud controllers. The component name for each controller should be set to
the component where the controllers are currently running. Almost always this is the `kube-controller-manager`. The configuration file should look something like this:
```yaml
kind: LeaderMigrationConfiguration
apiVersion: v1alpha1
leaderName: cloud-controllers-migration
controllerLeaders:
  - name: route-controller
    component: kube-controller-manager
  - name: service-controller
    component: kube-controller-manager
  - name: cloud-node-controller
    component: kube-controller-manager
  - name: cloud-nodelifecycle-controller
    component: kube-controller-manager
```

Save the leader migration configuration file somewhere, for this example we'll use `/etc/kubernetes/cloud-controller-migration.yaml`.
Now update the kube-controller-manager to set `--cloud-migration-config /etc/kubernetes/cloud-controller-migration.yaml`.

##### Deploy the CCM

Now deploy the CCM on your cluster but ensure it also has the `--cloud-migration-config` flag set, using the same config file you used for the KCM above.

How the CCM is deployed is out of scope for this KEP, refer to the cloud provider's documentation on how to do this.

#####  Update Leader Migration Config on Upgrade

To migrate controllers from the KCM to the CCM, update the component field from `kube-controller-manager` to `cloud-controller-manager` on every control plane node prior to
upgrading the node. If you are replacing nodes on upgrade, ensure new nodes set the `component` field to `cloud-controller-manager`. The new config file should look like this:
```yaml
kind: LeaderMigrationConfiguration
apiVersion: v1alpha1
leaderName: cloud-controllers-migration
controllerLeaders:
  - name: route-controller
    component: cloud-controller-manager
  - name: service-controller
    component: cloud-controller-manager
  - name: cloud-node-controller
    component: cloud-controller-manager
  - name: cloud-nodelifecycle-controller
    component: cloud-controller-manager
```

NOTE: During upgrade, it is acceptable for control plane nodes to specify different component names for each controller as long as the `leaderName` field is the same across nodes.

##### Disable Leader Migration

Once all controllers are migrated to the desired component:
* disable the cloud provider in the `kube-controller-manager` (set `--cloud-provider=external`)
* disable leader migration on the `kube-controller-manager` and `cloud-controller-manager` by unsetting the `--cloud-migration-config` field.

### Risks and Mitigations

* Increased apiserver load due to new leader election resource per migration configuration.
* User error could result in cloud controllers not running in any component at all.

### Graduation Criteria

##### Alpha -> Beta Graduation

Leader migration configuration is tested end-to-end on at least 2 cloud providers.

##### Beta -> GA Graduation

Leader migration configuration works on all in-tree cloud providers.

### Upgrade / Downgrade Strategy

See [Example Walkthrough of Controller Migration](#example-walkthrough-of-controller-migratoin) for upgrade strategy.
Clusters can be downgraded and migration can be disabled by reversing the steps in the upgrade strategy assuming the behavior of each controller
does not change incompatibly across those versions.

### Version Skew Strategy

Version skew is handled as long as the leader name is consistent across all control plane nodes during upgrade.

## Implementation History

- 07-25-2019 `Summary` and `Motivation` sections were merged signaling SIG acceptance
- 01-21-2019  Implementation details are proposed to move KEP to `implementable` state.
