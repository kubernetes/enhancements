# Controller Manager Leader Migration

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Migration Configuration](#migration-configuration)
    - [Default LeaderMigrationConfiguration](#default-leadermigrationconfiguration)
    - [Component Flags](#component-flags)
    - [Example Walkthrough of Controller Migration with Default Configuration](#example-walkthrough-of-controller-migration-with-default-configuration)
      - [Enable Leader Migration on Components](#enable-leader-migration-on-components)
      - [Upgrade the Control Plane](#upgrade-the-control-plane)
      - [Disable Leader Migration](#disable-leader-migration)
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
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

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

* Define migration process for large scale, highly available clusters to migrate from the in-tree cloud provider mechanism, to their out-of-tree equivalents.

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
migration lock and the controller manager it will run in - either the CCM or the KCM.

The properties of the migration lock are:
  * must have a unique name
  * the set of controllers in the lock is immutable.
  * no two migration locks should have overlapping controllers
  * the controller manager where the lock runs can change across releases.
  * for a minor release it must run exclusively in one type of controller manager - KCM or CCM.

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

![example network controllers migration v1.17](/keps/sig-cloud-provider/2436-controller-manager-leader-migration/migrating-cloud-controllers-v1-17.png)
<br/>
**Figure 1**: Example of migrating cloud network controllers in v1.17

<br/>
<br/>
<br/>

![example network controllers migration v1.18](/keps/sig-cloud-provider/2436-controller-manager-leader-migration/migrating-cloud-controllers-v1-18.png)
<br/>
**Figure 2**: Example of migrating cloud network controllers in v1.18



### Notes/Constraints/Caveats (Optional)

#### Migration Configuration

The migration lock will be configured by defining new API types that will then be passed into the KCM and CCM.

```go
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// LeaderMigrationConfiguration provides versioned configuration for all migrating leader locks.
type LeaderMigrationConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// LeaderName is the name of the resource under which the controllers will be run.
	LeaderName string `json:"leaderName"`

	// ResourceLock indicates the resource object type that will be used to lock
	// Must be either "leases" or "endpoints", defaults to 'leases'
	// No other types (e.g. "endpointsleases" or "configmapsleases") are allowed
	ResourceLock string

	// ControllerLeaders contains a list of migrating leader lock configurations
	ControllerLeaders []ControllerLeaderConfiguration `json:"controllerLeaders"`
}

// ControllerLeaderConfiguration provides the configuration for a migrating leader lock.
type ControllerLeaderConfiguration struct {
	// Name is the name of the controller being migrated
	// E.g. service-controller, route-controller, cloud-node-controller, etc
	Name string `json:"name"`

	// Component is the name of the component in which the controller will be running.
	// E.g. kube-controller-manager, cloud-controller-manager, etc
	// Or '*' meaning the controller can be run under any component that participates the migration
	Component string `json:"component"`
}

```

#### Default LeaderMigrationConfiguration

The `staging/controller-manager` package will provide a default `LeaderMigrationConfiguration` which has the
`component` filed of all controllers that need migrating set to `*`, indicating the controllers can be run under
either party of the migration.

Please refer to [an workthough](#example-walkthrough-of-controller-migration-with-default-configuration)
of an example cloud controllers migration from KCM to CCM that use the default configuration.

The default values must be only used when no configuration file is specified. If a custom configuration file is
specified to either controller manager, the specified configuration will completely replace default value for the
corresponding controller manager.

#### Component Flags

Both `kube-controller-manager` and `cloud-controller-manager` will get support for the following two flags for Leader
Migration. First, `--enable-leader-migration` is a boolean flag which defaults to `false` that indicates whether Leader
Migration is enabled. Second, `--leader-migration-config` is an optional flag that accepts a path to a file containing
the `LeaderMigrationConfiguration` type serialized in yaml.

If `--enable-leader-migration` is `true` but `--leader-migration-config` flag is empty or not set, the
default `LeaderMigrationConfiguration` will be used.

If `--enable-leader-migration` is not set or set to `false`, but `--leader-migration-config` is set and not empty, the
controller manager will print an error at `FATAL` level and exit immediately. Additionally,
if `--leader-migration-config` is set but the configuration file cannot be read or parsed, the controller manager will
log the failure at `FATAL` level and exit immediately.

#### Example Walkthrough of Controller Migration with Default Configuration

This is an example of migrating a KCM-only Kubernetes 1.21 control plane to KCM + CCM 1.22.

After the upgrade, all cloud controllers will be moved from the KCM to the KCM. We assume KCM and CCM are running with
default controller assignments, namely, in 1.21, KCM runs `route-controller`, `service-controller`
, `cloud-node-controller`, and `cloud-nodelifecycle-controller`, and in 1.22, CCM instead will run all the 4
controllers.

If KCM and CCM are not running with the default controller assignments, a custom configuration file can be specified
with `--leader-migration-config`. However, this example only covers the simple case of using default configuration.

At the beginning, KCM should not have `--enable-leader-migration` or `--leader-migration-config` set, but it should
have `--cloud-provider` already set to an existing cloud provider (e.g. `--cloud-provider=gce`). At this point, KCM
runs `route-controller`, `service-controller`, `cloud-node-controller`, and `cloud-nodelifecycle-controller`. CCM is not
yet deployed.

##### Enable Leader Migration on Components

The default LeaderMigrationConfiguration can be represented as follows:

```yaml
kind: LeaderMigrationConfiguration
apiVersion: controllermanager.config.k8s.io/v1
leaderName: cloud-provider-extraction-migration
resourceLock: leases
controllerLeaders:
  - name: route
    component: *
  - name: service
    component: *
  - name: cloud-node-lifecycle
    component: *
```

First, within 1.21 control plane, update the `kube-controller-manager` to set `--enable-leader-migration`
and `--feature-gate=ControllerManagerLeaderMigration` (this enables `ControllerManagerLeaderMigration` feature gate) but
not `--leader-migration-config`, this flag enables Leader Migration with default configuration, which prepares KCM to
participate in the migration.

##### Upgrade the Control Plane

Upgrade each node of the control plane to 1.22 with the following updates:

- KCM has neither `--enable-leader-migration` or `--leader-migration-config`
- KCM has no cloud provider enabled with`--cloud-provider=`
- CCM deployed with `--enable-leader-migration`
- CCM has its `--cloud-provider` set to the correct cloud provider

Starting from version 1.22, a proper default that represents the most recent migration will be provided.
`*` in the `component` field indicates that the controllers can be run under either `kube-controller-manager` or `cloud-controller-manager`. Because the migration happens between
the 1.21 `kube-controller-manager` with built-in cloud provider and 1.22 `cloud-controller-manager`, the controllers
can run under either controller manager, which is exactly as described in the default configuration.

During the upgrade, either KCM or CCM may hold the migration lease and thus run migrated controllers.
However, at any moment, there is one and only one of them running the migrated controllers.
After upgrade, only CCM will run the `route`, `service`, `cloud-node-lifecycle` controllers.

##### Disable Leader Migration

Once all nodes in the control plane are upgraded to 1.22, disable leader migration on the `cloud-controller-manager` by
unsetting the `--enable-leader-migration` flag.

### Risks and Mitigations

* Increased apiserver load due to new leader election resource per migration configuration.
* User error could result in cloud controllers not running in any component at all.

## Design Details

### Test Plan

- Unit Testing:
  - test resource reading, parsing, validation
  - test calculation of leader differences.
  - test all helpers
- Integration Testing
  - test resource registration, parsing, and validation against the Schema APIs
  - test interactions with the leader election APIs
- E2E Testing
  - In a replicated control plane, test control plane upgrade, assert controller managers
    become health and ready after upgrade
  - In a replicated control plane, test control plane upgrade, assert no controllers
    become active in both controller managers.

### Graduation Criteria

##### Alpha -> Beta Graduation

Leader migration configuration is tested end-to-end on at least 2 cloud providers.
The default migration configuration is implemented and tested.

##### Beta -> GA Graduation

- Leader Migration works on all in-tree cloud providers that require migration.
- Leader Migration has an automated upgrade test on a replicated control plane, with Leader Migration enabled, of the following cases
  - Upgrade from KCM only to KCM + CCM
  - Rollback from KCM + CCM to KCM only

### Upgrade / Downgrade Strategy

See [Example Walkthrough of Controller Migration](#example-walkthrough-of-controller-migration-with-default-configuration) for upgrade strategy.
Clusters can be downgraded and migration can be disabled by reversing the steps in the upgrade strategy assuming the behavior of each controller
does not change incompatibly across those versions.

### Version Skew Strategy

Version skew is handled as long as the leader name is consistent across all control plane nodes during upgrade.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ControllerManagerLeaderMigration`
  - Components depending on the feature gate: `cloud-controller-manager` and `kube-controller-manager`

###### Does enabling the feature change any default behavior?

No. The user must explicitly add `--enable-leader-migration` flag to enable this feature. If the user enables this
feature without providing a configuration, the default configuration will reflect default situation and "just works".

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Once the feature is enabled via feature gate, it can be disabled by unsetting `--enable-leader-migration` on KCM
and CCM.

###### What happens if we reenable the feature if it was previously rolled back?

This feature can be re-enabled without side effects.

###### Are there any tests for feature enablement/disablement?

Yes. Unit & integration tests include flag/configuration parsing. E2E test will have cases with the feature enabled and
disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The rollout may fail if the configuration file does not represent correct controller-to-manager assignment
or configurations mismatch between controller managers.
This can cause controllers referred in the configuration file to either be unavailable or run in multiple instances.

The rollback may fail if the leader election of the controller manager is not properly configured.
For example, multiple instances of the same controller manager are running without election, or none of the instances become the leader.
In these situations, all controllers will be either unavailable or conflict among multiple instances.

###### What specific metrics should inform a rollback?

If neither controller managers show `leader_active` for the main leader lock or the migration lock, Leader Migration may fail to activate and thus needs rollback.
If any of the controllers indicate they are unavailable through their per-controller metric, Leader Migration may need reconfiguration.
The metrics of each controller are specific to the implementation of each cloud provider and out of scope of this KEP,

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

The manual testing showed a clean takeover of the migration leader during both upgrade and downgrade process.
This process will be tested as part of the e2e suite, required by [Graduation Criteria](#graduation-criteria).

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?
None.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

N/A. This feature is never used by any user workloads.

###### How can someone using this feature know that it is working for their instance?

- [X] Other (treat as last resort)
  - The `Lease` resource used in the migration can be watched for transition of leadership and timing information. 
  - logs and metrics can directly indicate the status of migration.

Note that this feature is intended for cluster administrators, who should have access to metrics during the upgrade.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Leader Migration is designed to ensure availability of controller managers during upgrade,
and this feature will not affect SLOs of controller managers.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - per-controller health checks in both controller managers.
  - Components exposing the metric: kube-controller-manager, cloud-controller-manager

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Status of the migration lease, provided by the API server, can help observe the transition of holders
if exposed as resource metrics.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- API Server
  - needed for leader election
    - Impact of its outage on the feature: when leader election timeout, controller managers will lose the leader and exit, causing outage.
    - Impact of its degraded performance or high-error rates on the feature: delayed or retried operations of leader election.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Leader Migration uses exactly one more resource of `coordination.k8s.io/v1.Lease` using the standard leader election process.
Both `kube-controller-manager` and `cloud-controller-manager` will create, update, and watch on the lease.

If the service accounts are not granted access to the lease resources, the RBAC roles of each controller manager may need to modified before the upgrade.

###### Will enabling / using this feature result in introducing new API types?

Type: `controllermanager.config.k8s.io/v1.LeaderMigrationConfiguration`
This resource is only for configuration file parsing. The resource should never reach the API server.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

This feature uses exactly one more `coordination.k8s.io/v1.Lease` resource. The RBAC roles of both controller managers will 
gain additional ~50 bytes because of the new lease under `resourceName`.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Both `kube-controller-manager` and `cloud-controller-manager` runs another leader election process,
which cause negligible increases of CPU and memory usages, both during upgrade and under normal operations.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The existing implementation of controller managers will `klog.Fatal` once the leader times out, which eventually happens if API server is unavailable.
Leader Migration will not change such behavior.

###### What are other known failure modes?

None. Leader Migration is known to fail only because of misconfiguration or unavailability of the API server, both of which are discussed above.

###### What steps should be taken if SLOs are not being met to determine the problem?
N/A.

## Implementation History

- 07-25-2019 `Summary` and `Motivation` sections were merged signaling SIG acceptance
- 01-21-2019 Implementation details are proposed to move KEP to `implementable` state.
- 09-30-2020 `LeaderMigrationConfiguration` and `ControllerLeaderConfiguration` schemas merged as #94205.
- 11-04-2020 Registration of both types merged as #96133
- 12-28-2020 Parsing and validation merged as #96226
- 03-10-2021 Implementation for alpha state completed, released in 1.21.
- 03-30-2021 User guide published as kubernetes/website#26970
- 05-11-2021 KEP updated to target beta.
- 01-21-2022 KEP updated to target GA.
- 01-25-2022 Testing and monitoring revised for GA.

## Drawbacks

A single-node control plane does not need this feature. If downtime is allowed during control plane upgrade, KCM and CCM
can have no migration mechanism at all.

## Alternatives

Change all controllers so that they can handle a situation where two instances of the same controller are running in
both KCM and CCM. This requires a massive change to all controllers and potentially require other kinds of
synchronization. It would be better that the controller manager provides migration mechanism instead of relying on each
controller.
