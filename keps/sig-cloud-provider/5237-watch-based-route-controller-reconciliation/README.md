# KEP-5237: Convert route controller to watch-based reconciliation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Changed Field Filters](#changed-field-filters)
    - [Relying only on Events](#relying-only-on-events)
- [Design Details](#design-details)
    - [Full reconcile](#full-reconcile)
    - [Periodic reconcile](#periodic-reconcile)
    - [Workqueue singleton](#workqueue-singleton)
    - [Node update filters](#node-update-filters)
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
  - [Reconcile individual nodes and avoid filtering events](#reconcile-individual-nodes-and-avoid-filtering-events)
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

Switch the route controller (`k8s.io/cloud-provider`) from a fixed interval reconciliation loop to a watch-based solution, leveraging informers.

## Motivation

The route controller in the cloud-controller-manager is currently reconciling in a fixed interval (default: 10s). This leads to unnecessary API requests towards the infrastructure providers. Related controllers, such as the service or node controller are already using a watch-based mechanism. We propose to switch the route controller to a similar solution.

This will also reduce the average time spent waiting for the route reconciliation when a new Node is added, as the event immediately starts a reconciliation instead of having to wait for up to ten seconds.

### Goals

First, reduce the overall amount of API requests towards the infrastructure providers. Second, reduce average time for route reconciliation of new nodes added to the cluster.

### Non-Goals

Change the logic of the reconciliation process itself.

## Proposal

The current route controller runs a full reconciliation every ten seconds. We propose to instead reconcile whenever there is a new node added, removed, or certain fields are updated. These fields include the `node.status.addresses` field and the configured `PodCIDRs`. Additionally, the route controller will run a full reconciliation at a less frequent interval.

### User Stories (Optional)

#### Story 1

As an infrastructure provider I would like to reduce the amount of avoidable API requests. (For the environment and operational costs)

#### Story 2

As a cluster operator I would like to use my infrastructure providers routes instead of an overlay network. The routes should be added as soon as possible for new nodes.

#### Story 3

As a cluster operator I need to use the API rate limits from my infrastructure provider effectively. Sending frequent API requests even though nothing changed causes me to deplete the API rate limits faster.

### Risks and Mitigations

#### Changed Field Filters

We currently use the `Node.Status.Addresses` and `PodCIDRs` fields to trigger updates in the route reconciliation mechanism. However, relying solely on these fields may be insufficient, potentially causing missed route reconciliations when updates are necessary. This depends on the specific cloud-controller-manager implementations. Using these fields works for the CCM maintained by the authors, but we do not know the details of other providers.

This is mitigated by the feature gate `CloudControllerManagerWatchBasedRoutesReconciliation`, which allows infrastructure providers to test it and provide feedback on the fields.

#### Relying only on Events

Other controllers rely on finalizers to make sure that the resource is only deleted when the controller had the chance to run any cleanup. This is currently not implemented for any controller in [`k8s.io/cloud-provider`](http://k8s.io/cloud-provider). Because of this, Nodes may get deleted without the possibility to process the event in the route controller.

This can cause issues with limits on the number of routes in the infrastructure provider, as well as invalid routes being advertised as valid, causing possible networking reliability or confidentiality issues.

This is mitigated by:

- not-filtered node event causing a full reconcile of *all* routes
- The controller doing a periodic reconcile to clean up outdated routes, just at a lower frequency than before, even if no events came in.

## Design Details

We use a similar concept as already implemented in the node and service controller. Through node informers we register event handlers for the add, delete and update node events, where updates are filtered by certain criteria. To introduce the feature we establish a new feature gate called `CloudControllerManagerWatchBasedRoutesReconciliation`.

```
<<[UNRESOLVED @joelspeed @lukasmetzner]>>
We had a discussion about adding a new metric for A/B testing. To not miss the v1.35 milestone, we decided to postpone this metric until the beta stage, which is planned for milestone v1.36.
<<[/UNRESOLVED]>>

```
Additionally, we introduce a new non-gated metric `route_controller_route_sync_total`, which tracks the number of route reconciliations. This metric can be used for A/B testing to evaluate the impact of watch based route reconciliation.

#### Full reconcile

The current route controller always does a “full reconcile”, looking at all routes inside the configured `ClusterCIDR` and all nodes and then taking action to create/update/delete any routes to match the expectation.

This will be kept, as there is no way to know which routes to delete without this full perspective.

#### Periodic reconcile

As already established, the cloud-controller-manager does not utilize “Owner” references and therefore routes of already deleted nodes could remain in the infrastructure provider. To ensure a consistent state with the cluster an additional periodic reconcile should be implemented. This can be implemented via the already existing `--min-resync-period` flag, which is already used by the service and node controller.

#### Workqueue singleton

To ensure only a single reconciliation loop is run concurrently we use the [`k8s.io/client-go/util/workqueue`](http://k8s.io/client-go/util/workqueue) package. We only add a single key with the constant value “routes”.

#### Node update filters

Node updates are quite frequent. To reduce the number of requests sent to infrastructure providers, it’s important to filter these events and only trigger reconciliations when necessary.

Two fields are relevant for determining whether a reconcile should occur:

1. `Node.Status.Addresses` maps to the `TargetNodeAddresses` field in the `Route` struct. It determines where packets for a given IP range should be sent. Changes to this field must trigger a reconcile.
2. `Node.Spec.PodCIDRs` contains the IP ranges assigned to the node. These CIDRs are used as the destination in the created routes. Changes to this field must trigger a reconcile.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- Test filtering when updating a node by checking if reconcile is triggered through an item in the workqueue

##### Integration tests

- None, integration tests will be performed in the cloud providers implementing the controller.

##### e2e tests

- None, the current e2e tests should continue to work.

### Graduation Criteria

#### Alpha

- Implementation merged in [`k8s.io/cloud-provider`](http://k8s.io/cloud-provider)
- At least one cloud-provider has tested this in their CCM

#### Beta

- Multiple infrastructure provider have enabled the feature flag, and we are confident that the field selectors are correct.

#### GA

- The feature is enabled by default.

### Upgrade / Downgrade Strategy

- This is not an issue, as we do not touch the reconciliation logic itself
- This feature can be disabled/enabled at any time without implications

### Version Skew Strategy

- Not relevant, as the code is vendored in each CCM. It only relies on very basic features that are available in all currently supported Kubernetes versions.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
      - Feature gate name: `CloudControllerManagerWatchBasedRoutesReconciliation`
      - Components depending on the feature gate: [`k8s.io/cloud-provider`](http://k8s.io/cloud-provider) / cloud-controller-managers
- [x] Other
      - Describe the mechanism:
      - Will enabling / disabling the feature require downtime of the control plane?
        - no, the feature gate can be toggled at any time.
      - Will enabling / disabling the feature require downtime or reprovisioning of a node?
        - no, the feature gate can be toggled at any time.

###### Does enabling the feature change any default behavior?

Yes, the routes are no longer reconciled in a fixed interval of 10s. The feature still works the same.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, after disabling the feature, the route controller will switch back to the regular reconcile process.

###### What happens if we reenable the feature if it was previously rolled back?

Nothing different from the first enable.

###### Are there any tests for feature enablement/disablement?

No

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

In the worst case the route controller no longer runs. This means that any new Nodes will not have their routes setup and the Pod-to-Pod communicationis broken for talking to pods running on that new node.
This will not impact already running nodes/workloads, but they might be affected transitively because they rely on workloads scheduled to broken nodes.

###### What specific metrics should inform a rollback?

The metrics for the new workqueue can be used to determine whether the route controller still runs. If the following query is constant after adding a new node, it means that the new implementation does not reconcile the node routes.

```promql
workqueue_work_duration_seconds_count{name="Routes"}
```

Note that these metrics are only available from the CCM.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes, this did not cause any issues.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

The `--route-reconcile-period` flag can be deprecated, as it's no longer needed once we reach General Availability (GA).

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The feature is only used in the CCM if the `Route` controller is supported by the cloud-provider and enabled (enabled by default, can be disabled in). This is visible in existing clusters through the CCM metric `running_managed_controllers{name="route"}`.

After upgrading, they can look at `workqueue_work_duration_seconds_count{name="Routes"}` to see that the reconciler runs.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
      - There are no events for this.
- [x] API Node.status
      - Condition name: `NodeNetworkUnavailable=false` will be added to the Node if the controller works, same as before.
- [ ] Other (treat as last resort)
      - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

None

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
      - Metric name: `workqueue_work_duration_seconds_count{name="Routes"}`
      - [Optional] Aggregation method: `rate`
      - Components exposing the metric: CCMs
- [ ] Other (treat as last resort)
      - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A histogram similar to cloud_provider_taint_removal_delay_seconds for creation delay of Routes for new nodes.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

Cloud controller manager

### Scalability

###### Will enabling / using this feature result in any new API calls?

No, the node informer we rely on was already used in the route controller, through the Node Lister.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No new calls, the amount of calls may change depending on the specific configuration. Our goal is to reduce the number of calls.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Previously a reconcile would still happen every 10s based on cached node list. Now we only react to new events, so if the API server is unavailable nothing will happen.
The controller is only meant to do something when new nodes are added, and that can not happen anyway when the API server is unavailable.

###### What are other known failure modes?

- Continuous node events that match our field filters could trigger reconciliation in a tight loop. To prevent this, we apply a rate limiter to the workqueue. The limiter ensures that reconciliations never occur more frequently than every 10 seconds, preserving the behavior of the previous fixed interval while adding robustness against event storms.

###### What steps should be taken if SLOs are not being met to determine the problem?

The feature gate can be turned off at any time, as there is no migration process.

## Implementation History

## Drawbacks

A spike in events could trigger multiple reconciles successively.

## Alternatives

### Reconcile individual nodes and avoid filtering events

If the controller was changed to reconcile individual nodes, this filter could be possibly be removed and there would be less risk for missing fields in the filter.

We do not think this is a reasonable alternative because:

1) The cloud provider interface only allows a `ListRoutes`, so If we were to reconcile every node by itself we would need to call this many more times, countering the purpose of this enhancement.
2) The controller also does a regular “clean up” of routes that are no longer valid. If we only reconciled individual nodes we can not do this, as the node that this route belongs to may already be deleted.

## Infrastructure Needed (Optional)

None
