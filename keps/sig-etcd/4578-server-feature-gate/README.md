# KEP-4578: Server Feature Gate in etcd

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Is feature enablement state a server level or cluster level property?](#is-feature-enablement-state-a-server-level-or-cluster-level-property)
    - [Is feature gate a replacement of boolean flags?](#is-feature-gate-a-replacement-of-boolean-flags)
    - [Should we use feature gate for bug fixes?](#should-we-use-feature-gate-for-bug-fixes)
    - [Could the lifecycle of a feature change in patch versions?](#could-the-lifecycle-of-a-feature-change-in-patch-versions)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Feature Gate](#feature-gate)
  - [Ways to Query Feature Gate State](#ways-to-query-feature-gate-state)
  - [Feature Stages](#feature-stages)
  - [Path to Migrate Existing Experimental Features](#path-to-migrate-existing-experimental-features)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Milestone 1](#milestone-1)
    - [Milestone 2](#milestone-2)
    - [Milestone 3](#milestone-3)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
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
- [ ] (R) Test plan is in place
- [ ] (R) Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [etcd-io/website], for publication to [etcd.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[etcd.io]: https://etcd.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[etcd-io/website]: https://github.com/etcd-io/website

## Summary

We are introducing the Kubernetes style feature gate framework into etcd to gate future feature enhancement behind a sequence of feature lifecyles. Users would be able to turn features on or off and query feature enablement of an etcd server in a consistent way. 

## Motivation

Currently any new enhancements to the etcd are typically added as [an experimental feature](https://github.com/etcd-io/etcd/blob/main/Documentation/contributor-guide/features.md#adding-a-new-feature), with a configuration flag prefixed with "experimental", e.g. --experimental-feature-name. 

When it is time to [graduate an experimental feature to stable](https://github.com/etcd-io/etcd/blob/main/Documentation/contributor-guide/features.md#graduating-an-experimental-feature-to-stable), a new stable feature flag identical to the experimental feature flag but without the --experimental prefix is added to replace the old feature flag, which is a breaking change.

The following problems exist with the current `--experimental` flag mechanism: 
* hard to track when and why feature was introduced, and what stage it is in.
* no clear path to graduate flags, as removal of experimental prefix is a breaking change in command line.
* unclear, undocumented feature flag graduation criteria. So very often features get stuck or forgotten in experimental phase.
* no method for clients to query and make decisions based on feature enablement in a cluster.
* each `--experimental` flag will have to be piped through the code based separately. It is cumbersome, nothing stops them being modified somewhere along the path, and some code changes might slip through and not be guarded by the flag.

We are proposing to introduce Kubernetes style [feature gates](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/) into etcd to address these issues.

### Goals

* Introduce a single mechanism in the code base for enabling/disabling features with code changes guarded behind the mechanism without risking breaking the rest of the server if the feature is disabled.

* Introduce a way for a client to find out what features are enabled in the server.

* Codify lifecycle stages of a feature, and clear tracking and documentation of feature lifecycle progression.

* Pave the path to migrate all existing `--experimental` features to feature gate. 

### Non-Goals

* Handle the feature gate enablement on the cluster level. They require more careful consideration and thus warrants a  separate KEP.

* Use feature gate as a mechanism to fix bugs.

## Proposal

### User Stories (Optional)

#### Story 1

A developer adds a new feature, and finds a bug after its Alpha release. 

Now the developer can take their time to fix the bug, because they can be confident all the new changes related to the feature are guarded behind the feature gate, Alpha features are disabled by default and should be adpoted by a small set of users.

On the user side, how would the users know if they are using the buggy feature if they do not remember which flags were used to start the cluster? Before feature gate, they need to search through the long logs to find the flag value when the server initially started, if the log still exists. Now there is a way to find which features are enabled on a server directly, and restart the servers with the buggy feature enabled.

#### Story 2

A developer added a new feature, and after a few releases, they are ready to graduate the feature.

Before feature gate, to avoid breaking changes, they would need to add a stable flag `--my-feature-flag` along with `--experimental-my-feature-flag`. Now there are two duplicate flags doing exactly the same thing.
Now they need to wait till the next minor release to remove `--experimental-my-feature-flag`. If they forget, we are stuck with the two flags until someone bothers to look into verifying the two flags are identical and eventually removing `--experimental-my-feature-flag`.

Now, the developer just need to change stage of the feature to the next stage - single line change.

### Notes/Constraints/Caveats (Optional)

#### Is feature enablement state a server level or cluster level property?

There are simple features like `ExperimentalEnableDistributedTracing` (enables distributed tracing using OpenTelemetry protocol), which could work fine at the local server level.

There are also features like `ExperimentalEnableLeaseCheckpoint` (enables leader to send regular checkpoints to other members to prevent reset of remaining TTL on leader change), if different server nodes have different enablement values for `ExperimentalEnableLeaseCheckpoint`, the results would be inconsistent depending on which one is the leader and confusing.

There are also more critial features like changing the Apply logic of a raft message. All members have to agree on whether to use the new logic or old logic, otherwise it could break correctness.

So feature gate should be able to handle both server level and cluster level features. Since cluster level feature consensus is itself a complicated problem. We will defer it to its own KEP, and only focus on the server level feature gate in this KEP.

#### Is feature gate a replacement of boolean flags?
No. We should only consider feature gate flags for knobs that we plan to make default and force enabled for etcd instances. For optional knobs we should keep a command line flags.

Feature gate is meant to facilitate the safe development of a feature and guard any code changes behind the gate. It is not a feature configuration. A feature could have many other flags as its configuration, including boolean flags. If you think the user would still need to be able to toggle its configuration on/off when the feature graduates and becomes an integral part of etcd, then you would need a boolean configuration flag in addition to the feature gate when introducing the feature.

#### Should we use feature gate for bug fixes?

There are some use cases when some bugs are found and later fixed in a patch version. The client would need to know if the etcd version in use contains that fix to decide whether or not to use that feature. 

The question is: should a new feature gate be added to signal the bug fix? 

We think the answer is generally NO:
* the new feature would need to be enabled by default to always apply the bug fix for new releases.
* it changes the API which is not desirable in patch version releases.

The proper way of handling these bug fixes should be:
1. the feature should be gated by the feature gate from the beginning.
1. the feature should be disabled by default until it is widely tested in practice. 
1. when the bug is found, the feature should ideally be at a lifecycle in which it is disabled by default. If not, the admin should disable it by the `--feature-gates` flag.
1. when the client upgrades etcd to the patch version with the fix, the admin could enable it by the `--feature-gates` flag.

But if the bug fix is very critical, it could be discussed in a case-by-case manner.

#### Could the lifecycle of a feature change in patch versions?

Kubernetes have a minor release every 3 months, while the cadence of etcd minor releases is much less frequent. The question is: do we have to wait for years before graduating a new feature?

We think we should still stick to the common practice of not changing the lifecycle of a feature in patch versions. Because:
* changing the lifecycle of a feature is an API change. According to the [etcd Operations Guide](https://etcd.io/docs/v3.5/op-guide/versioning/), only new minor versions may add additional features to the API.
* bugs in etcd could be hard to detect, and the reliability and robustness of etcd is more important than speed. A long history of testing through practical adoption a new feature is beneficial.

With the feature gate in place, we could consider increasing the etcd release cadence because it would be easier to add new features and less risky to release new features.

### Risks and Mitigations

* As discussed before, we are deferring cluster level feature support to a follow-up KEP.

## Design Details

### Feature Gate

We will create a new `featuregate` module under `pkg/` (mostly copying code from [`k8s.io/component-base/featuregate`](https://github.com/kubernetes/component-base/tree/master/featuregate)) with the following interface:
```go
// FeatureGate indicates whether a given feature is enabled or not
type FeatureGate interface {
	// Enabled returns true if the key is enabled.
	Enabled(key Feature) bool
	// KnownFeatures returns a slice of strings describing the FeatureGate's known features.
	KnownFeatures() []string
	// DeepCopy returns a deep copy of the FeatureGate object, such that gates can be
	// set on the copy without mutating the original. This is useful for validating
	// config against potential feature gate changes before committing those changes.
	DeepCopy() MutableFeatureGate
}
```

We will use the new `VersionedSpecs`(introduced in [kep-4330](https://github.com/kubernetes/enhancements/tree/master/keps/sig-architecture/4330-compatibility-versions)) to register and track the features along with their lifecycle at different release versions. 

```go
defaultServerFeatureGates := map[Feature]VersionedSpecs {
		featureA: VersionedSpecs{
			{Version: mustParseVersion("3.6"), Default: false, PreRelease: Beta},
			{Version: mustParseVersion("3.7"), Default: true, PreRelease: GA},
		},
		featureB: VersionedSpecs{
			{Version: mustParseVersion("3.7"), Default: false, PreRelease: Alpha},
		},
}
```

A new `--feature-gates` command line argument would be added to start the etcd server, with format like `--feature-gates=featureA=true,featureB=false`. The flag can also be set in the `config-file`. The flag can only be set during startup. We do not support dynamically changing the feature gates when the server is running.

The `ServerConfig` struct will have a new `featuregate.FeatureGate`(immutable) field. `EtcdServer` would have an interface of `FeatureEnabled(key Feature) bool`, and it can be referenced throughout the server.

```go
type ServerConfig struct {
  ...
  // ServerFeatureGate server level feature gate
  ServerFeatureGate featuregate.FeatureGate
  ...
}

func (s *EtcdServer) FeatureEnabled(key Feature) bool
```

### Ways to Query Feature Gate State

New Prometheus gauge metrics would be added to monitor and query if a feature is enabled for the server. 

The metrics could look like:
* `etcd_feature_enabled{name="featureName",stage="Alpha"} 1` if the feature is enabled.

### Feature Stages

We propose etcd features to follow the convention of [Kubernetes feature stages](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/#feature-stages). Generally, a feature can go through a lifecycle of Alpha → Beta → GA → Deprecated. Each feature should be associated with a Kubernetes Enhancement Proposal ([KEP](https://www.kubernetes.dev/resources/keps/)), with high level graduation criteria defined in the KEP.

| Feature Stage | Properties | Minimum Graduation Criteria |
| --- | --- | --- |
| Alpha | Same as [Kubernetes feature stages - Alpha feature](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/#feature-stages): <ul><li>Disabled by default. </li><li>Might be buggy. Enabling the feature may expose bugs. </li><li>Support for feature may be dropped at any time without notice. </li><li>The API may change in incompatible ways in a later software release without notice. </li><li>Recommended for use only in short-lived testing clusters, due to increased risk of bugs and lack of long-term support.</li></ul> | Before moving a feature to Beta, it should have <ul><li> Full unit/integration/e2e/robustness test coverage.</li><li>Full performance benchmark/test if applicable.</li><li> No significant changes for at least 1 minor release.</li></ul> |
| Beta | Same as [Kubernetes feature stages - Beta feature](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/#feature-stages) except etcd Beta feature does not allow incompatible schema change: <ul><li>Enabled by default. </li><li>The feature is well tested. Enabling the feature is considered safe.</li><li>Support for the overall feature will not be dropped, though details may change.</li><li>Recommended for only non-business-critical uses because of potential for discovering new hard-to-spot bugs through wider adoption.</li></ul> | Before moving a feature to GA, it should have <ul><li> Widespread usage.</li><li>No bug reported for at least 1 minor release.</li></ul> |
| GA | Same as [Kubernetes feature stages - GA feature](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/#feature-stages): <ul><li>The feature is always enabled; you cannot disable it.</li><li>The corresponding feature gate is no longer needed.</li><li>Stable versions of features will appear in released software for many subsequent versions.</li></ul> | Before deprecating a GA feature, it should have <ul><li> Feature deprecation announcement.</li><li>No user impacting change for at least 1 minor release.</li></ul> |
| Deprecated | <ul><li>The feature gate is no longer in use. </li><li>The feature has graduated to GA or been removed.</li></ul> | <ul><li>If deprecating Beta feature, should set default to disabled first for at least 1 minor release.</li></ul> |

### Path to Migrate Existing Experimental Features

Feature gate is the replacement of future experimental features. We can also use it to establish a path to deprecate existing experimental features. 

To safely migrate an existing experimental feature `--experimental-feature-a`, we need to go through several stages:
1. the `--experimental-feature-a` and `--feature-gates=FeatureA=true|false` flags coexist for at least 1 minor release. Announce upcoming deprecation of the old flag.
    1. create a new feature `FeatureA` in feature gate. The lifecyle stage of `FeatureA` would be Alpha if the old flag is disabled by default, and Beta if it is enabled by default.
    1. both the `--experimental-feature-a` and `--feature-gates=FeatureA=true|false` can be used to set the enablement of `FeatureA`, and add checks to make sure the two flags are not both set at start-up. 
    1. all references to the `experimental-feature-a` value in the code would be replaced by `featureGate.Enabled(FeatureA)`
    1. print warning messages of flag deprecation in the help, and when the old flag is used.
1. depreate the old `--experimental-feature-a` flag in the next minor release. Keep the lifecycle of `FeatureA` the unchanged for at least 1 minor release.
1. normal lifecycle progression of `FeatureA` from now on.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

New feature gate unit tests will be added.

##### Integration tests

New feature gate integration tests will be added.

##### e2e tests

We will a couple of server level experimental features to the feature gate (without removing the original experimental flag), and add e2e tests to make sure the feature gate functions equivalently to their `--experimental-xxx` flags.

We will also add downgrade/upgrade e2e tests to make sure the feature gate does not break the cluster in mixed version cluster.

### Graduation Criteria

#### Milestone 1

* server level feature gate implemented.
  * `--feature-gates` flag added.
  * feature gate added to the server code, and used by a server level experimental feature.
  * feature metrics added.

#### Milestone 2

* server level feature gate thoroughly tested.
  * e2e tests added for the feature gate equivalent of the selected experimental feature(s).
  * robustness test scenarios added for the selected experimental feature(s).
  * documentation to track all feature gates added to [etcd-io/website].

#### Milestone 3

* migrate all existing `--experimental` feature flags to feature gate.
  * create equivalent feature gates for existing `--experimental` features without removing the old flags.
  * update references to the `--experimental` feature flags to the new feature gates in the code.

### Upgrade / Downgrade Strategy

The feature gate feature would available in 3.6+. 
Since server level features should not affect any cluster level properties, it should not impact Upgrade/Downgrade process.

### Version Skew Strategy

Since server level features should not affect any cluster level properties, it should not impact existing version skew policy.

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
