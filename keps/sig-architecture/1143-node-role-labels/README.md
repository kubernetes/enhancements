# Appropriate use of node-role labels

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Use of <code>node-role.kubernetes.io/*</code> labels](#use-of-node-rolekubernetesio-labels)
  - [Current users of <code>node-role.kubernetes.io/*</code> within the project that must change](#current-users-of-node-rolekubernetesio-within-the-project-that-must-change)
    - [Service load-balancer](#service-load-balancer)
    - [Node controller excludes master nodes from consideration for eviction](#node-controller-excludes-master-nodes-from-consideration-for-eviction)
    - [Kubernetes e2e tests](#kubernetes-e2e-tests)
    - [Preventing accidental reintroduction](#preventing-accidental-reintroduction)
- [Design Details](#design-details)
  - [Migrating existing deployments](#migrating-existing-deployments)
    - [Instructions for deployers](#instructions-for-deployers)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Future work](#future-work)
- [Reference](#reference)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases) of the targeted release**.

These checklist items _must_ be updated for the enhancement to be released.

- [x] kubernetes/enhancements issue in release milestone, which links to KEP: https://github.com/kubernetes/enhancements/issues/1143
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Clarify that the `node-role.kubernetes.io/*` label is for use only by users and external projects and may not be used to vary Kubernetes behavior. Define migration process for all internal consumers of these labels.

## Motivation

The `node-role.kubernetes.io/master` and the broader `node-role.kubernetes.io` namespace for labels were introduced to provide a simple organizational and grouping convention for cluster users. The labels were reserved solely for organizing nodes via a convention that tools could recognize to display information to end users, and for use by opinionated external tooling that wished to simplify topology concepts. Use of the label by components within the Kubernetes project (those projects subject to API review) was restricted. Specifically, no project could mandate the use of those labels in a conformant distribution, since we anticipated that many deployments of Kubernetes would have more nuanced control-plane topologies than simply "a control plane node".

Over time, several changes to Kubernetes core and related projects were introduced that depended on the `node-role.kubernetes.io/master` label to vary their behavior in contravention to the guidance the label was approved under. This was unintentional and due to unclear reviewer guidelines that have since been more strictly enforced. Likewise, the complexity of Kubernetes deployments has increased and the simplistic mapping of control plane concepts to a node has proven to limit the ability of conformant Kubernetes distributions to self-host, as anticipated. The lack of clarity in how to use node-role and the disjoint mechanisms within the code has been a point of confusion for contributors that we wish to remove.

Finally, we wish to clarify that external components may use node-role tolerations and labels as they wish as long as they are cognizant that not all conformant distributions will expose or allow those tolerations or labels to be set.


### Goals

This KEP:

* Clarifies that the use of the `node-role.kubernetes/*` label namespace is reserved solely for end-user and external Kubernetes consumers, and:
  * Must not be used to vary behavior within Kubernetes projects that are subject to API review (kubernetes/kubernetes and all components that expose APIs under the `*.k8s.io` namespace)
  * Must not be required to be present for a cluster to be conformant
* Describes the locations within Kubernetes that must be changed to use an alternative mechanism for behavior
  * Suggests approaches for each location to migrate
* Describes the timeframe and migration process for Kubernetes distributions and deployments to update labels


## Proposal

### Use of `node-role.kubernetes.io/*` labels

* Kubernetes components MUST NOT set or alter behavior on any label within the `node-role.kubernetes.io/*` namespace.
* Kubernetes components (such as `kubectl`) MAY simplify the display of `node-role.kubernetes.io/*` labels to convey the node roles of a node
* Kubernetes examples and documentation MUST NOT leverage the node-role labels for node placement
* External users, administrators, conformant Kubernetes distributions, and extensions MAY use `node-role.kubernetes.io/*` without reservation
  * Extensions are recommended not to vary behavior based on node-role, but MAY do so as they wish
* First party components like `kubeadm` MAY use node-roles to simplify their own deployment mechanisms.
* Conformance tests MUST NOT depend on the node-role labels in any fashion
* Ecosystem controllers that desire to be placed on the masters MAY tolerate the node-role master taint or set nodeSelector to the master nodes in order to be placed, but SHOULD recognize that some deployment models will not have these node-roles, or may prohibit deployments that attempt to schedule to masters as unprivileged users. In general we recommend limiting this sort of placement rule to examples, docs, or simple deployment configurations rather than embedding the logic in code.


### Current users of `node-role.kubernetes.io/*` within the project that must change

The following components vary behavior based on the presence of the node-role labels:


#### Service load-balancer

The service load balancer implementation previously implemented a heuristic where `node-role.kubernetes.io/master` is used to exclude masters from the candidate nodes for a service. This is an implementation detail of the cluster and is not allowed. Since there is value in excluding nodes from service load balancer candidacy in some deployments, an alpha feature gated label `alpha.service-controller.kubernetes.io/exclude-balancer` was added in Kubernetes 1.9.

This label should be moved to beta in Kube 1.19 at its final name `node.kubernetes.io/exclude-from-external-load-balancers`, its feature gate `ServiceNodeExclusion` should default on in 1.19, the gate `ServiceNodeExclusion` should be declared GA in 1.21, and the gate will be removed in 1.22. The old alpha label should be honored in 1.21 and removed in 1.22.

Starting in 1.16 the legacy code block should be gated on `LegacyNodeRoleBehavior=true`


#### Node controller excludes master nodes from consideration for eviction

The `k8s.io/kubernetes/pkg/util/system/IsMasterNode(nodeName)` function is used by the NodeLifecycleController to exclude nodes with a node name that ends in `master` or starts with `master-` when considering whether to mark nodes as disrupted. A recent PR attempted to change this to use node-roles and was blocked. Instead, the controller should be updated to use a label `node.kubernetes.io/exclude-disruption` to decide whether to exclude nodes from being considered for disruption handling.


#### Kubernetes e2e tests

The e2e tests use a number of heuristics including the `IsMasterNode(nodeName)` function and the node-roles labels to select nodes. In order for conformant Kubernetes clusters to run the tests, the e2e suite must change to use individual user-provided label selectors to identify nodes to test, nodes that have special rules for testing unusual cases, and for other selection behaviors. The label selectors may be defaulted by the test code to their current values, as long as a conformant cluster operator can execute the e2e suite against an arbitrary cluster.

The `IsMasterNode()` method will be moved to be test specific, identified as deprecated, and will be removed as soon as possible.

QUESTION: Is a single label selector sufficient to identify nodes to test?


#### Preventing accidental reintroduction

In order to prevent reviewers from accidentally allowing code changes that leverage this functionality, we should clarify the Godoc of the constant to limit their use.  A lint process could be run as part of verify that requires approval of a small list to modify exclusions (currently only cmd/kubeadm will be allowed to use that constaint, with all test function being abstracted). The review doc should call out that labels must be scoped to a particular feature enablement vs being broad.

Some components like the external cloud provider controllers (considered to fall within these rules due to implementing k8s.io APIs) may be vulnerable to accidental assumptions about topology - code review and e2e tests are our primary mechanism to prevent regression.


## Design Details

### Migrating existing deployments

The proposed fixes will all require deployment-level changes. That must be staged across several releases, and it should be possible for deployers to move early and "fix" the issues that may be caused by their topology.

Therefore, for each change we recommend the following process to adopt the new labels in successive releases:

* Release 1 (1.16):
  * Introduce a feature gate for disabling node-role being honored. The gate defaults to on. `LegacyNodeRoleBehavior=true`
  * Define the new node label with an associated feature gate for each feature area. The gate defaults to off. `ServiceNodeExclusion=false` and `NodeDisruptionExclusion=false`
  * Behavior for each functional area is defined as `(LegacyNodeRoleBehavior == on && node_has_role) || (FeatureGate == on && node_has_label)`
  * No new components may leverage node-roles within Kubernetes projects.
  * Early adopters may label their nodes to opt in to the features, even in the absence of the gate.
* Release 2 (1.17):
  * The legacy alpha label `alpha.service-controller.kubernetes.io/exclude-balancer` is marked as deprecated
  * Deprecation of node role behavior in tree is announced for 1.21, with a detailed plan for cluster administrators and deployers
  * Gates are officially alpha
* Release 3 (1.19):
  * The old label `alpha.service-controller.kubernetes.io/exclude-balancer` is removed
  * For both labels, usage is reviewed and as appropriate the label is declared beta/GA and the feature gate is set on
  * All Kubernetes deployments should be updated to add node labels as appropriate: `kubectl label nodes -l node-role.kubernetes.io/master LABEL_A=VALUE_A`
  * Documentation will be provided on making the transition
  * Deployments may set `LegacyNodeRoleBehavior=false` after they have set the appropriate labels.
* Release 4 (1.21):
  * Default the legacy gate `LegacyNodeRoleBehavior` to off. Admins whose deployments still use the old labels may set `LegacyNodeRoleBehavior=true` during 1.19 to get the legacy behavior.
  * Deployments should stop setting `LegacyNodeRoleBehavior=false` if they opted out early.
* Release 5 (1.22):
  * The `LegacyNodeRoleBehavior` gate and all feature-level gates are removed, components that attempt to set these gates will fail to start.
  * Code that references node-roles within Kubernetes will be removed.

In Release 5 (which could be as early as 1.21) this KEP will be considered complete.

#### Instructions for deployers

The current behavior of the `node-role.kubernetes.io/master` label on nodes preventing them from being part of service load balancers or from being disrupted when NotReady is deprecated and will be fully removed in Kubernetes 1.20. Administrators and Kubernetes deployers should follow these steps.

If you are using the `alpha.service-controller.kubernetes.io/exclude-balancer` label in your deployments to exclude specific nodes from your deployment, the label has been replaced in 1.17 with `node.kubernetes.io/exclude-from-external-load-balancers`.  All administrators should run the following command before upgrading to Kubernetes 1.18 and set the feature gate `ServiceNodeExclusion=true`:

    kubectl label nodes --selector=alpha.service-controller.kubernetes.io/exclude-balancer \
        node.kubernetes.io/exclude-balancer=true

Cluster deployers that rely on the existing behavior where master nodes are not part of the service load balancer and master workloads will not be evicted if the master is NotReady for longer than the grace period should run the following command after upgrading to Kubernetes 1.18:

    kubectl label nodes --selector=node-role.kubernetes.io/master \
        node.kubernetes.io/exclude-from-external-load-balancers=true \
        node.kubernetes.io/exclude-disruption=true

After setting these labels in 1.18, administrators will need to take no further action.

Cluster deployers that wish to manage this migration during the 1.17 to 1.18 upgrade should label nodes and set feature gates before upgrading to 1.18. If `LegacyNodeRoleBehavior=false` is set, it must be removed prior to the 1.21 to 1.22 upgrade.


### Test Plan

* Unit tests to verify selection using feature gates

### Graduation Criteria

* New labels and feature flags become beta after one release, GA and defaulted on after two, and are removed after two releases after they are defaulted on (so 4 releases from when this is first implemented).
* Documentation for migrating to the new labels is available in 1.18.

### Upgrade / Downgrade Strategy

As described in the migration process, deployers and administrators have 2 releases to migrate their clusters.

### Version Skew Strategy

Controllers are updated after the control plane, so consumers must update the labels on their nodes before they update controller processes in 1.21.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `LegacyNodeRoleBehavior`, `ServiceNodeExclusion`
    - Components depending on the feature gate: `kube-apiserver`, `kube-controller-manager`, cloud controller managers
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**

Yes

* **What happens if we reenable the feature if it was previously rolled back?**

The old behavior is present.

* **Are there any tests for feature enablement/disablement?**

Yes

### Rollout, Upgrade and Rollback Planning

Covered in migration strategy.

### Monitoring requirements

* **How can an operator determine if the feature is in use by workloads?**

Not applicable to workloads

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**

Not applicable

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

Not applicable

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**

No

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**

No

### Scalability

* **Will enabling / using this feature result in any new API calls?**

No

* **Will enabling / using this feature result in introducing new API types?**

No

* **Will enabling / using this feature result in any new calls to cloud
  provider?**

No

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**

No

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**

No

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**

No

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**

Not applicable

* **What are other known failure modes?**

Not applicable

* **What steps should be taken if SLOs are not being met to determine the problem?**

Not applicable

## Implementation History

- 2019-07-16: Created
- 2020-04-15: Labels promoted to beta in 1.19 in https://github.com/kubernetes/kubernetes/pull/90126
- 2020-06-01: Updated for 1.19 with details of production readiness
- 2021-01-06: GA in 1.21 and marked to be removed in 1.22

## Future work

This proposal touches on the important topic of scheduling policy - the ability of clusters to restrict where arbitrary workloads may run - by noting that some conformant clusters may reject attempts to schedule onto masters. This is out of scope of this KEP except to indicate that node-role use by ecosystem components may conflict with future enhancements in this area.


## Reference

* https://groups.google.com/d/msg/kubernetes-sig-architecture/ZKUOPy2PNJ4/lDh4hs4HBQAJ
* https://github.com/kubernetes/kubernetes/pull/35975
* https://github.com/kubernetes/kubernetes/pull/39112
* https://github.com/kubernetes/kubernetes/pull/76654
* https://github.com/kubernetes/kubernetes/pull/80021
* https://github.com/kubernetes/kubernetes/pull/78500 - Work to remove master role label from e2e
