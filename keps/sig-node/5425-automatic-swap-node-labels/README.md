# KEP-5425: Automatic Swap Node Labels

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
  - [NodeFeatureDiscovery Limitations](#nodefeaturediscovery-limitations)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

This KEP proposes that kubelet automatically set well-known node labels based on swap configuration to enable workload isolation and scheduling decisions for swap-enabled nodes.

## Motivation

### Goals

- Enable workload isolation between swap-enabled and swap-disabled nodes.
- Support nodeSelector and nodeAffinity based node selection for swap use-cases without requiring external tools.

### Non-Goals

- Implementing API-based swap limits (separate KEP).
- Modifying existing Swap modes or functionality at node.
- Providing swap capacity information for quantitative scheduling.

## Background

Currently, Kubernetes supports swap through node-level configuration (`NoSwap`, `LimitedSwap`), but workloads have no way to discover or request specific swap behavior from nodes. This KEP proposes Kubelet adding a new node-label for swap-enabled nodes during bootstrap.

This will help with below scenarios:

1. Enable targeted scheduling for swap demanding pods to swap enabled nodes.
2. Guard workloads that will have concerns about performance inference from accidentally placed on a swap enabled node.
2. Users missing to manually label nodes for swap-based scheduling.
3. Avoid external tooling dependency (NodeFeatureDiscovery) for swap feature discovery.

### NodeFeatureDiscovery Limitations

While NFD can detect swap devices on nodes, it has some limitations:
- NFD labels reflect hardware state (swap device presence), not Kubernetes configuration state 
- It cannot distinguish between `NoSwap`, `LimitedSwap`, or future swap modes
- NFD is not an in-tree solution, requiring additional deployments for working with swap.

## Proposal

### API Changes

Kubelet will automatically set the following well-known node labels based on swap configuration:

```yaml
metadata:
  labels:
    # Indicates the kubelet swap behavior mode  
    node.kubernetes.io/swap-behavior: "NoSwap" | "LimitedSwap"
```

### Label Semantics

#### `node.kubernetes.io/swap-behavior` 
- **`"NoSwap"`**: Workloads will not use swap (disabled by configuration).
- **`"LimitedSwap"`**: Node-level swap with implicit calculation

#### Integration Point

There is precedent in Kubernetes for auto-labeling of nodes based on hardware / os features / cloud-topology.

eg: below labels are set by Kubelet
```bash
kubernetes.io/arch - CPU architecture
kubernetes.io/hostname - Node hostname
kubernetes.io/os - Operating system
node.kubernetes.io/instance-type - Instance type from cloud provider
topology.kubernetes.io/region - Cloud region
topology.kubernetes.io/zone - Cloud availability zone
```

Swap labels will also be set during initial node status update, following the same pattern as existing kubelet-managed labels.

### Implementation

#### Kubelet Changes

```go
// api/core/v1/well_known_labels.go
const LabelLinuxSwapBehavior = "node.kubernetes.io/swap-behavior"

// pkg/kubelet/kubelet_node_status_others.go
func (kl *Kubelet) getNodeSwapLabels(node *v1.Node) (map[string]string, error) {
    swapBehavior := kl.kubeletConfiguration.MemorySwap.SwapBehavior
    
    // Set node label based on swap-presence and configuration
    found, err := swapDetected()
    if err != nil {
      return nil, err
    }

    if swapBehavior == "NoSwap" {
        if found {
          klog.Warning("Swap is detected at node, but swap disabled for workloads because configuration is NoSwap.")
        }
    } else {
        if !found {
          klog.Warningf("Swap configured(%v) but swap device not detected at node.", swapBehavior)
        }
    }

    return map[string]string{v1.LabelLinuxSwapBehavior: string(swapBehavior)}, nil
}
```

### User Stories

#### Story 1: Workload Isolation
As a cluster administrator, I want to isolate swap-sensitive workloads from swap-enabled nodes to prevent performance degradation.

```yaml
# Non-swap workload
spec:
  nodeSelector:
    node.kubernetes.io/swap-behavior: "NoSwap"
```

#### Story 2: Swap-Required Workloads  
As a developer, I want my memory-intensive batch jobs to run only on swap-enabled nodes for better resource utilization.

```yaml
# workload desiring swap
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: node.kubernetes.io/swap-behavior
            operator: NotIn
            values: ["NoSwap"]
```

#### Story 3: Migration Planning
As a cluster operator, I want to identify which nodes have which swap configuration during cluster migration.

```bash
kubectl get nodes -l node.kubernetes.io/swap-behavior=LimitedSwap
```

## Design Details

### Label Lifecycle

- Labels are set during kubelet startup and initial node registration / reconciliation.
- Labels are updated when kubelet configuration changes
- Labels persist through node restarts

### Backwards Compatibility

- New labels are additive. There will be no breaking changes
- Existing swap functionality is unchanged with the new label proposed
- Nodes without explicit swap configuration will get a new `node.kubernetes.io/swap-behavior: "NoSwap"` label on kubelet / node restart

### Validation

- Kubelet validates that swap configuration matches detected system state. This will raise kubelet warning logs if swap behavior is `LimitedSwap` but no swap is detected.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

- kubelet

  - Test `getNodeSwapLabels()` function with different swap configurations
  - Test label updates when swap configuration changes
  - Test feature gate enabled/disabled scenarios

##### Integration tests

- `TestKubeletSwapLabels` - kubelet startup with swap configuration
  - Test kubelet sets correct labels on node registration
  - Test label updates when kubelet configuration is updated
- `TestSwapLabelsPersistence` - Validate label persistence across kubelet restarts
- `TestSwapLabelsValidation` - Test validation of swap configuration vs system state

##### e2e tests

- E2E test to validate workload scheduling based on swap labels
- In a cluster with swap-enabled nodes, node-query with labels should filter the right set of nodes.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved 

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

**Alpha**

- Feature is implemented behind the feature gate `AutomaticSwapNodeLabels`
- Basic unit tests are implemented and passing
- Initial integration tests are completed
- Swap Documentation is updated with automatic node-label enhancement details.
- Manual verification of swap node labels for different swap configurations is completed.

**Beta**
- Feature gate is enabled by default
- Added comprehensive test coverage (unit, integration, e2e)

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

**Upgrade**

Users can enable the `AutomaticSwapNodeLables` feature gate on Kubelet after upgrading to a Kubernetes version that supports it. On Kubelet startup, new swap labels will be automatically added to nodes 

**Downgrade**

Downgrading kubelet to a version without `AutomaticSwapNodeLabels` feature will not remove the pre-created labels; this is also desirable to ensure scheduling decisions will continue to respect the node-isolation. For a clean downgrade, users will have to manually delete the labels if they are no longer desired after downgrade as below:

```bash
kubectl label nodes --all node.kubernetes.io/swap-behavior-
```

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

Nodes with older kubelets won't have swap labels, but functionality is not impacted. Node selection with swap behavior labels will only include swap-enabled nodes in newer versions. This would have implication that workloads using swap labels may not find suitable nodes during mixed-version periods.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `AutomaticSwapNodeLabels`
  - Components depending on the feature gate: `kubelet`
- [X] Other
  - Describe the mechanism: Kubelet add preset labels for swap detection and mode on startup.
  - Will enabling / disabling the feature require downtime of the control
    plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Yes, but minimally. New swap relevant node-labels will appear on the node objects / `kubectl describe node` output after upgrade. No functional behavior changes to existing workloads or scheduling.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes. Users could set feature gate `AutomaticSwapNodeLabels=false`. During rollback, the nodes with the feature disabled would stop reporting the node label `node.kubernetes.io/swap-behavior`. Manual action will be required if user intended to remove the labels. If swap configuration at the node is changed, labels that were not cleaned-up will reflect stale configuration.  If labels were removed, workloads using the label-based node-swap filtering  will no longer follow the swap-node-isolation.

###### What happens if we reenable the feature if it was previously rolled back?

If the feature is re-enabled, swap-behavior labels will be re-added based on current swap configuration.

###### Are there any tests for feature enablement/disablement?

Yes, feature enablement/disablement tests will be added along with alpha implementation.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

* 2025-06-19: [Proposal](https://github.com/kubernetes/kubernetes/issues/132416) and discussion.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### Manual Labeling Only
**Not Preferred**: Requires manual cluster management and doesn't scale. Since swap is considered a high-risk configuratio, it is preferrable to include this node metadata as a built-in solution. 

### Use NodeFeatureDiscovery
**Not Preferred**: 
- NFD detects swap presence in the machine, not kubelet configuration
- External dependency for core functionality
- Cannot distinguish between swap modes

### Automatic Swap Taints
**Not Preferred**: Taints are more disruptive for swap enabled nodes to be added at bootstrap. This will require admins add tolerations to node-management pods and other critical daemon-sets for swap nodes, requiring to manage a second set of yamls which is undesirable. Auto-Labels will help users intend to manage swap with taints with selection queries such as:

`kubectl taint nodes -l node.kubernetes.io/swap-behavior=LimitedSwap swap-enabled=true:NoSchedule`

### Extending NodeCapabilities API for swap
**Early In Design**: Labels provide better integration with existing node-selection mechanisms (nodeSelector, affinity, etc.), and applicable for more use-cases such as monitoring. NodeCapabilities API (as proposed) in early design stages and is considered only for Kubelet feature-discovery currently. NodeCapabilities for swap could help with complex swap-aware scheduling needs, if integrated in the future. If swap-capability is exported by node in the future, it can co-exist with labels which are generic node metadata that can help with other filtering needs.

### Include Swap Capacity Information
**Not Preffered**: 
- Conflicts with API / adds complexity without clear use case
- Capacity can change dynamically, making labels stale

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
