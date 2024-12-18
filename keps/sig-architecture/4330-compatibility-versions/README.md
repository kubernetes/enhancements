<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-4330: Compatibility Versions in Kubernetes

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Component Flags](#component-flags)
    - [--emulation-version](#--emulation-version)
    - [--min-compatibility-version](#--min-compatibility-version)
  - [Skew](#skew)
  - [Changes to Feature Gates](#changes-to-feature-gates)
    - [Feature Gate Lifecycles](#feature-gate-lifecycles)
    - [Feature gating changes](#feature-gating-changes)
  - [Validation ratcheting](#validation-ratcheting)
    - [CEL Environment Compatibility Versioning](#cel-environment-compatibility-versioning)
  - [StorageVersion Compatibility Versioning](#storageversion-compatibility-versioning)
  - [API availability](#api-availability)
  - [API Field availability](#api-field-availability)
  - [Discovery](#discovery)
  - [Version introspection](#version-introspection)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Risk: Increased cost in test due to the need to test changes against the e2e tests of multiple release branches](#risk-increased-cost-in-test-due-to-the-need-to-test-changes-against-the-e2e-tests-of-multiple-release-branches)
    - [Risk: Increased maintenance burden on Kubernetes maintainers](#risk-increased-maintenance-burden-on-kubernetes-maintainers)
    - [Risk: Unintended and out-of-allowance version skew](#risk-unintended-and-out-of-allowance-version-skew)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
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

We intend to introduce version compatibility and emulation options to
Kubernetes control plane components to make upgrades safer by increasing the
granularity of steps available to cluster administrators. 

We will introduce a `--emulation-version` flag to emulate the capabiliites
(APIs, features, ...) of a prior Kubernetes release version. When used, the
capabilities available will match the emulated version. Any capabilities present
in the binary version that were introduced after the emulation version will be
unavailable and any capabilities removed after the emulation version will be
available. This enables a binary version to emulate the behavior of a previous
version with sufficient fidelity that interoperability with other system
components can be defined in terms of the emulated version. 

We will also introduce a `--min-compatibility-version` flag to control the
minimum version a control plane component is compatible with (in terms of
storage versions, validation rules, ...). When used, the component tolerates
workloads that expect the behavior of the specified minimum compatibility
version, component skew ranges extend based on the minimum compatibility
version, and rollbacks can be performed back to the specified minimum
compatibility version.

## Motivation

The notion of more granular steps in Kubernetes upgrades is attractive because
it is more rigorous about how we step through a Kubernetes control-plane
upgrade, introducing potentially corrupting data (i.e. data only present in N+1
and not in N) only in later stages of the upgrade process.

For example, upgrading from Kubernetes 1.30 to 1.31 while keeping the emulation
version at 1.30 would enable a cluster administrator to validate that the new
Kubernetes binary version is working as desired before exposing any feature changes
introduced in 1.31 to cluster users, and without writing and data to storage at
newer API versions.

This extra step increases the granularity of our upgrade sequence so that 
(1) failures are more easily diagnosed (since we have more granular steps, we
have slightly more data sheerly from which point of the upgrade lifecycle we
achieved failure) and (2) failures are more easily auto-reverted by upgrade
orchestration as we are taking smaller and more incremental steps forward,
which means there is less to “undo” on a failure condition.

It also becomes possible to skip binary versions while still performing a
stepwise upgrade of Kubernetes control-plane. For example:

- (starting point) binary-version 1.28 (compat-version 1.28)
- upgrade binary-version to 1.31 (emulation-version stays at 1.28 - this is our skip-level binary upgrade)
- keep binary-version 1.31 while upgrading emulation-version to 1.29 (stepwise upgrade of emulation version)
- keep binary-version 1.31 while upgrading emulation-version to 1.30 (stepwise upgrade of emulation version)
- keep binary-version 1.31 while upgrading emulation-version to 1.31 (stepwise upgrade of emulation version)

Benefits to upgrading binary version independent of emulation version:

- A skip-version binary upgrade that transitions through emulation versions has
  higher probability of bugs in those intermediate binary versions already being
  fixed.
- During an upgrade, it is possible upgrade the new binary version while the
  emulation version is fixed (e.g. `binaryVersion: 1.28` -> `{binaryVersion: 1.29, emulationVersion: 1.28}`).
  This allow differences in the binary (bugs fixed or introduced, performance
  changes, ...) to be introduced into a cluster and verified safe before
  allowing access to the new APIs and features new version. Once the binary
  version has been deamed safe, the emulation version can then be upgraded.
- Any upgrade system that successfully detects failures between upgrade steps
  can report which upgrade step failed. This makes it easier to diagnose the
  failures, because there are fewer possible causes of the failure. (There's a
  huge difference between "A cluster upgrade failed" and "A cluster upgrade
  failed when only the apiserver's binary version changed").
- For upgrades where multiple failures occur, this increases the odds those
  failures are split across steps. An upgrade system that is able to pause after
  a step where failures are detected allow for failures at that step to be
  addressed before proceeding to subsequent steps. These failures can be
  addressed without the disruption and "noise" from failures in subsequent
  steps.
- An emulation version rollback can be performed without changing binary version.

A dedicated `--min-compatibility-version` flag provides direct control of when
deprecated features are removed from the API.  If the `--min-compatibility-version`
is kept at a fixed version number during a binary version or emulation version
upgrade, the cluster admin is guaranteed no features will be removed, reducing
the risk of the upgrade step.

Also, `--min-compatibility-version` can be used to provide a wider skew range
between components.

Lastly, a `--min-compatibility-version` can be set to the binary version
for new clusters being used for green field projects, providing immediate
access to the latest Kubernetes features without the need to wait an additional
release for features to settle in as is typically needed for rollback support.

### Goals

- Introduce the metadata necessary to configure features/APIs/storage-versions/validation rules
  to match the behavior of an older Kubernetes release version
- A Kubernetes binary with emulation version set to N, will pass the
  conformance and e2e tests from Kubernetes release version N.
- A Kubernetes binary with emulation version set to N does not enable any
  changes (storage versions, CEL feature, features) that would prevent it
  from being rolled back to N-1.
- The most recent Kubernetes version supports emulation version being set to
  the full range of supported versions.
  - In alpha we intend to support:
    - `--emulation-version` range of `binaryMinorVersion`..`binaryMinorVersion-1`
  - In beta, we intend to extend support to:
    - `--emulation-version` range of `binaryMinorVersion`..`binaryMinorVersion-3`
    - `--min-compatibility-version` to `emulationVersion`..`binaryMinorVersion-3`

### Non-Goals

- Support `--emulation-version` for Alpha features.  Alpha feature are not
  designed to be upgradable, so we will not allow alpha features to be enabled when
  `--emulation-version` is set.
- `--min-compatibility-version` will only apply to Beta and GA features. Only
  Alpha features available in the current binary version will be available for enablement
  and are allowed to change in behavior across releases in ways that are incompatible
  with previous versions.
- Changes to Cluster API/kubeadm/KIND/minikube to absorb the compatibility versions
  will be addressed separate from this KEP

## Proposal

### Component Flags

#### --emulation-version

- Defaults to `binaryVersion` (matching current behavior)
- Must be <= `binaryVersion`
- Must not be lower than the supported range of minor versions (see graduation
  criteria for ranges). If below the supported version range the binary will
  exit and report an invalid flag value error telling the user what versions are
  allowed.
- Is not a bug-for-bug compatibility model. It specifically controls which APIs,
  feature gates, configs and flags are enabled to match that of a previous
  release version.

Adding `--emulation-version` to kubelet is out of scope for this enhancement.
But we do need to define how kubelet skew behaves when the kube-apiserver has
`--emulation-version` set. Our general rule is that we want to define skew
using emulation versions when they are in use. So not only must a kubelet
version be <= the kube-apiserver binary version, it must also be <= the
`--emulation-version` of the kube-apiserver.

#### --min-compatibility-version

- Defaults to `emulationVersion-1` if `emulationVersion.GreaterThan(binaryVersion-3)` (matching current behavior in emulation mode), 
and defaults to `emulationVersion` if `emulationVersion.EqualTo(binaryVersion-3)` (because of the max version range we are supporting) 
- Must be <= `--emulation-version`
- Must not be lower than the supported range of minor versions (see graduation
  criteria for ranges). If below the supported version range the binary will
  exit and report an invalid flag value error telling the user what versions are
  allowed.

### Skew

The [Version skew policy](https://kubernetes.io/releases/version-skew-policy/)
rules be defined in terms of compatibility and emulation versions:

- kube-controller-manager, kube-scheduler, and cloud-controller-manager:
  - Previously: `1.{binaryMinorVersion-1}`..`{binaryVersion}`
  - With this enhancement: `{minCompatibilityVersion}..{emulationVersion}`

- kube-proxy, kubelet:
  - Previously: `1.{binaryMinorVersion-3}`..`{binaryVersion}`
  - With this enhancement: `{minCompatibilityVersion-2}..{emulationVersion}`

- kubectl:
  - Previously: `1.{binaryMinorVersion-1}`..`{binaryVersion+1}`
  - With this enhancement: `{minCompatibilityVersion}..{emulationVersion+1}`

### Changes to Feature Gates

Features will track version information, i.e.:

```go
map[Feature]VersionedSpecs{
		featureA: VersionedSpecs{
			{Version: mustParseVersion("1.27"), Default: false, PreRelease: Beta},
			{Version: mustParseVersion("1.28"), Default: true, PreRelease: GA},
		},
		featureB: VersionedSpecs{
			{Version: mustParseVersion("1.28"), Default: false, PreRelease: Alpha},
		},
		featureC: VersionedSpecs{
			{Version: mustParseVersion("1.28"), Default: false, PreRelease: Beta},
		},
		featureD: VersionedSpecs{
			{Version: mustParseVersion("1.26"), Default: false, PreRelease: Alpha},
			{Version: mustParseVersion("1.28"), Default: true, PreRelease: Deprecated},
		}
```

Features with compatibility implications (like a new API field or relaxing
validation to allow a new enum value) could include that in their feature-gate
spec:

```go
map[Feature]VersionedSpecs{
		featureA: VersionedSpecs{
			  {Version: mustParseVersion("1.28"), MinCompatibilityVersion: mustParseVersion("1.28"), ...},Alpha},
			  {Version: mustParseVersion("1.29"), MinCompatibilityVersion: mustParseVersion("1.28"), ...},Beta},
			  {Version: mustParseVersion("1.30"), MinCompatibilityVersion: mustParseVersion("1.28"), ...},
		},
```

And features using a gate to guard a behavior change with compatibility
implications that isn't really going through the feature lifecycle could set the
feature version and the min compatibility version to the same thing:

```go
relaxValidationFeatureA: VersionedSpecs{
    {Version: mustParseVersion("1.30"), MinCompatibilityVersion: mustParseVersion("1.30"), Default: true, ...},
},
```

When a component starts, feature gates `VersionedSpecs` will be compared against
the emulation and compatibility version to determine which features to enable.

#### Feature Gate Lifecycles

`--feature-gates` must behave the same as it did for the emulation version. For
example, it must be possible to use `--feature-gates` to disable features that
were beta at the emulation version. One important implication of this
requirement is that feature gating must be kept in the Kubenetes codebase after
a feature has reached GA (or been removed) to support the emulation and
compatibility version ranges.

A feature that is promoted once per release would look something like:

```go
map[Feature]VersionedSpecs{
		featureA: VersionedSpecs{
			{Version: mustParseVersion("1.26"), Default: false, PreRelease: Alpha},
			{Version: mustParseVersion("1.27"), Default: true, PreRelease: Beta},
			{Version: mustParseVersion("1.28"), Default: true, PreRelease: GA},
		},
}
```

The lifecycle of the feature would be:

| Release | Stage | Feature tracking information                      |
| ------- | ----- | ------------------------------------------------- |
| 1.26    | alpha | Alpha: 1.26                                       |
| 1.27    | beta  | Alpha: 1.26, Beta: 1.27 (on-by-default)           |
| 1.28    | GA    | Alpha: 1.26, Beta: 1.27 (on-by-default), GA: 1.28 |
| 1.29    | GA    | Alpha: 1.26, Beta: 1.27 (on-by-default), GA: 1.28 |
| 1.30    | GA    | Alpha: 1.26, Beta: 1.27 (on-by-default), GA: 1.28 |
| 1.31    | GA    | **Feature implementation becomes part of normal code. `if featureGate enabled { // implement feature }` code may be removed at this step** |

All feature gating and tracking must remain in code through 1.30 for
emulation version support range (see graduation criteria for ranges we plan to support).

For a Beta feature that is removed, e.g.:

```go
map[Feature]VersionedSpecs{
		featureA: VersionedSpecs{
			{Version: mustParseVersion("1.26"), Default: false, PreRelease: Beta},
			{Version: mustParseVersion("1.27"), Default: false, PreRelease: Deprecated},
			{Version: mustParseVersion("1.31"), Default: false, PreRelease: Removed},
		},
}
```

The steps to remove the Beta feature would be:

| Release | Stage | Feature tracking information                      |
| ------- | ----- | ------------------------------------------------- |
| 1.26    | beta  | Beta: 1.26                                        |
| 1.27    | beta  | Beta: 1.26, Deprecated: 1.27                      |
| 1.28    | beta  | Beta: 1.26, Deprecated: 1.27                      |
| 1.29    | beta  | Beta: 1.26, Deprecated: 1.27                      |
| 1.30    | -     | Beta: 1.26, Deprecated: 1.27, Removed: 1.31       |
| 1.31    | -     | Beta: 1.26, Deprecated: 1.27, Removed: 1.31       |
| 1.32    | -     | Beta: 1.26, Deprecated: 1.27, Removed: 1.31       |
| 1.33    | -     | **`if featureGate enabled { // implement feature }` code may be removed at this step** |

(Features that are deleted before reaching Beta do not require emulation version
support since we don't support emulation version for alpha features)

Note that this respects a 1yr deprecation policy.

All feature gating and tracking must remain in code through 1.32 for
emulation version support range see (see graduation criteria for ranges we plan to support).

#### Feature gating changes

In order to preserve the behavior of in-development features across multiple releases,
feature implementation history should also be preserved in the code base instead of in place modifications.

Only sigificant and observable changes in feature capabilities should be across
releases. We do not want to impose a unreasonable burdon on feature authors. 
The main criteria to make the decision is: 
**Does this change break the contract with existing users of the feature?** i.e. would this change break the workloads of existing feature users if the user does not change the compatibility version?

Here are some common change scenarios and whether the change needs to be preserved or not:
* API change [Yes]
* Change of supported systems [Yes]
* Bug fix [No]
* Performance optimizations [No]
* Unstable metrics change [No]
* Code refactoring [No]

Listed below are some concrete examples of feature changes:
**Feature**|**Changes That Should Be Preserved**|**Changes That Do Not Need To Be Preserved**
-----|-----|-----
APIPriorityAndFairness | [add v1beta3 for Priority And Fairness](https://github.com/kubernetes/kubernetes/pull/112306) | [More seat metrics for APF](https://github.com/kubernetes/kubernetes/pull/105873)
ValidatingAdmissionPolicy | | [Drop AvailableResources from controller context](https://github.com/kubernetes/kubernetes/pull/117977), [Encapsulate KCM controllers with their metadata](https://github.com/kubernetes/kubernetes/pull/120371)
APIServerTracing | | [Revert "Graduate API Server tracing to beta"](https://github.com/kubernetes/kubernetes/pull/113803)
MemoryManager | | [Don't reuse memory of a restartable init container](https://github.com/kubernetes/kubernetes/pull/120715)
NodeSwap | if done after promoting to beta: [Add full cgroup v2 swap support and remove cgroup v1 support](https://github.com/kubernetes/kubernetes/pull/118764) | [only configure swap if swap is enabled](https://github.com/kubernetes/kubernetes/pull/120784)

To preserve the behavior, naively the feature implementations can be gated by version number. 
For example, if `FeatureA` is partially implemented in 1.28 and additional functionality
is added in 1.29, the feature developer is expected to gate the functionality by version.
E.g.:

```go
if feature_gate.Enabled(FeatureA) && feature_gate.EmulationVersion() <= "1.28" {implementation 1}
if feature_gate.Enabled(FeatureA) && feature_gate.EmulationVersion() >= "1.29" {implementation 2}
```

A better way might be to define a `featureOptions` struct constructed based on the the feature gate, and have the `featureOptions` control the main code flow, so that the main code is version agnostic. 
E.g.:

```go
// in kube_features.go
const (
  FeatureA featuregate.Feature = "FeatureA"
  FeatureB featuregate.Feature = "FeatureB"
)

func init() {
	utilfeature.DefaultMutableFeatureGate.AddVersioned(defaultKubernetesFeatureGates)
}

var defaultKubernetesFeatureGates = map[Feature]VersionedSpecs{
		featureA: VersionedSpecs{
			{Version: mustParseVersion("1.27"), Default: false, PreRelease: Alpha},
		},
    featureB: VersionedSpecs{
			{Version: mustParseVersion("1.28"), Default: false, PreRelease: Alpha},
			{Version: mustParseVersion("1.30"), Default: true, PreRelease: Beta},
		},
}

type featureOptions struct {
  AEnabled        bool
  AHasCapabilityZ bool
  BEnabled        bool
  BHandler        func()
}

func newFeatureOptions(featureGate FeatureGate) featureOptions {
  opts := featureOptions{}
  if featureGate.Enabled(FeatureA) {
    opts.AEnabled = true
  }
  if featureGate.EmulationVersion() > "1.29" {
    opts.AHasCapabilityZ = true
  }

  if featureGate.Enabled(FeatureB) {
    opts.BEnabled = true
  }
  if featureGate.EmulationVersion() > "1.28" {
    opts.BHandler = newHandler
  } else {
    opts.BHandler = oldHandler
  }
  return opts
}

// in client.go
func ClientFunction() {
  // ...
  featureOpts := newFeatureOptions(utilfeature.DefaultFeatureGate)
  if featureOpts.AEnabled {
    // ...
    if featureOpts.AHasCapabilityZ {
      // run CapabilityZ
    }
  }
  if featureOpts.BEnabled {
    featureOpts.BHandler()
  }
  // ...
}

```

### Validation ratcheting

All validationg ratcheting needs to account for compatibility version.

If code to support ratcheting is introduced in 1.30, then new values needing the
ratcheting may only be written if the compatibility version >= 1.30.
Since we require emulation version >= compatibility version, the emulation version
must also be 1.30 or greater.

#### CEL Environment Compatibility Versioning

CEL compatibility versioning is a special case of validation ratcheting.

CEL environments already [support a compatibility
version](https://github.com/kubernetes/kubernetes/blob/7fe31be11fbe9b44af262d5f5cffb1e73648aa96/staging/src/k8s.io/apiserver/pkg/cel/environment/base.go#L45).
The CEL compatibility version is used to ensure when a kubeneretes contol plane
component reads a CEL expression from storage written by a (N+1) newer version
(either due to version skew or a rollback), that a compatible CEL environment
can still be constructed and the expression can still be evaluated.  This is
achieved by making any CEL environment changes (language settings, libraries,
variables) available for [stored expressions one version before they are allowed
to be written by new
expressions](https://github.com/kubernetes/kubernetes/blob/7fe31be11fbe9b44af262d5f5cffb1e73648aa96/staging/src/k8s.io/apiserver/pkg/cel/environment/environment.go#L38).

The only change we must make for this enhancement is to remove the
[compatibility version
constant](https://github.com/kubernetes/kubernetes/blob/7fe31be11fbe9b44af262d5f5cffb1e73648aa96/staging/src/k8s.io/apiserver/pkg/cel/environment/base.go#L45)
and instead always pass in N-1 of the compatibility version introduced by this
enhancement as the CEL compatibility version.

### StorageVersion Compatibility Versioning

StorageVersions specify what version an apiserver uses to write resources to etcd
for each API group. The StorageVersion changes across releases as API groups
graduate through stability levels.

During upgrades and downgrades, the storage version is particularly important.
To enable upgrades and rollbacks, pre compatibility version, the version selected for storage in etcd in 
version N must be (en/de)codable for k8s versions N-1 through N+1. With compatibility version, the version selected for storage in etcd for the combination of `EmulationVersion` and `MinCompatibilityVersion` must be (en/de)codable for k8s versions `MinCompatibilityVersion` through `EmulationVersion+1`.

Thus, to determine the storage version to use at compatibility version N, we 
will find the set of all supported GVRs for each version in the range of `MinCompatibilityVersion` and `EmulationVersion+1` and intersect 
them to find a list of all GVRs supported by every binary version in the window. 
The storage version of each group-resource is the newest 
(using kube-aware version sorting) version found in that list for that group-resource.

### API availability

Similar to feature flags, all APIs group-versions declarations will be modified
to track which Kubernetes version the API group-versions are introduced (or
removed) at.

GA APIs should match the exact set of APIs enabled in GA for the Kubernetes version
the emulation version is set to.

All Beta APIs (both off-by-default and on-by-default, if any of those
still exist) need to behave exactly as they did for the Kubernetes version
the emulation version is set to. I.e. `--runtime-config=api/<version>` needs
to be able to turn on APIs exactly like it did for each Kubernetes version that
emulation version can be set to.

Alpha APIs may not be enabled in conjunction with emulation version.

### API Field availability

API fields that were introduced after the emulation version will **not** be
pruned. Ideally they would be, but we already show information about unavailable
fields in the API today like disabled-by-default features (Alphas mostly) and
make no attempt to hide those fields in discovery.

We consider pruning fields based on emulation version useful future work that
would benefit multiple aspects of how APIs are served, so while we're not taking
on the effort in this KEP, we are interested in seeing this improved.

### Discovery

Discovery will [enable](https://github.com/kubernetes/kubernetes/blob/7080b51ee92f67623757534f3462d8ae862ef6fe/staging/src/k8s.io/apiserver/pkg/util/openapi/enablement.go#L32) the group versions matching the emulation version.

The API fields include will match what is described in the "API Fields" section.

### Version introspection

The `/version` endpoint will be augmented to report binary version when this feature
is enabled. Note that this changes default behavior by always including a new field
in `/version` responses.  E.g.

```json
{
  "major": "1",
  "minor": "30",
  "binaryMajor": "1",
  "binaryMinor": "32",
  "compatibility": "29",
  "gitVersion": "v1.30.0",
  "gitCommit": "<something>",
  "gitTreeState": "clean",
  "buildDate": "2024-03-30T06:36:32Z",
  "goVersion": "go1.21.something",
  "compiler": "gc",
  "platform": "linux/arm64"
}
```

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

A cluster administrator is running Kubernetes 1.30.12 and wishes to perform a
cautious upgrade to 1.31.5 using the smallest upgrade steps possible, validating
the health of the cluster between each step.

- For each control plane component, in the [recommended
  order](https://kubernetes.io/releases/version-skew-policy/):
  - Upgrades binary to `kubernetes-1.31.5` but sets `--emulation-version=1.30`
  - Verifies that the cluster is healthy
- Next, for each control plane component:
  - Sets `--emulation-version=1.31`
  - Verifies that the cluster is healthy

#### Story 2

A cluster administrator is running Kubernetes 1.30.12 and wishes to perform a cautious
upgrade to 1.31.5, but after upgrading realizes that a feature the workload depends on
had been removed and needs to rollback until the workload can be modified to not
depend on the feature.

- For each control plane component, in the [recommended
  order](https://kubernetes.io/releases/version-skew-policy/):
  - Cluster admin restarts the component with `--emulation-version=1.30` set

This avoids having to rollback the binary version.  Once the workload is fixed, the
cluster administrator can remove the `--emulation-version` to roll the cluster
forward again.

#### Story 3

A cluster administrator creating a new Kubernetes cluster for a development of a
new project and wishes to make use of the latest available features.

- Cluster admin starts all cluster components with a `1.30` binary version and
  sets `--min-compatibility-version=1.30` as well.

Because the cluster admin has no need to rollback, setting
`--min-compatibility-version=1.30` can be used to indicate that they do not
require any feature availability delay to support a compatibility range
and benefit from access to all the latest available features.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

### Risk: Increased cost in test due to the need to test changes against the e2e tests of multiple release branches

TODO: Establish a plan for this for Alpha. Implement for Beta.

#### Risk: Increased maintenance burden on Kubernetes maintainers

Why we think this is manageable:

- We already author features to be gated. The only change here is include
  enough information about features so that they can be selectively enabled/disabled
  based on emulation version.
- We already manually deprecate/remove features. This change will instead
  leave features in code longer, and require feature gates to track at which
  verion a feature is deprecated/removed.  The total maintenance work is
  about the same.
- Some maintenance becomes simpler as the additional version data about
  features makes them easier to reason about and keep track of.

#### Risk: Unintended and out-of-allowance version skew

From @deads2k: "I see an additional risk of unintended and out-of-allowance version skew between binaries. A kube-apiserver and kube-controller-manager contract is still +/-1 (as far as I see here). Compatibility and emulation versions, especially across three versions, makes it more likely for accidental mismatches.

While a hard shutdown of a process is likely worse than the disease, exposing some sort of externally trackable signal for cluster-admins and describing how to use it could significantly mitigate the problem."

Possible mitigations:

- Clients send version numbers in request headers. Servers use this to detect
  out-of-allowance skew. Servers then surface this to cluster administrators
  through a metric.
- Components register identity leases (apiserver already does this)
  https://github.com/kubernetes/enhancements/pull/4356 proposes doing it for
  controller managers. Components include their version information in the
  identity leases. A separate controller inspects all the leases for skew and
  surafces it to cluster administrators.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

For Alpha, we will fill this out as we implement.

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

For Alpha we will add integration test to ensure `--emulation-version` behaves expected according to the following grid:

**Transition**|**N-1 Behavior**|**N Behavior**|**Expect when emulation-version=N-1**|**Expect when emulation-version=N (or is unset)**
-----|-----|-----|-----|-----
Alpha feature introduced|-|off-by-default|feature does not exist, feature gate may not be set|alpha features may not be used in conjunction with emulation version
Alpha feature graduated to Beta|off-by-default|on-by-default|feature enabled only when `--feature-gates=<feature>=true`|feature enabled unless `--feature-gates=<feature>=false`
Beta feature graduated to GA|on-by-default|on|feature enabled unless `--feature-gates=<feature>=false`|feature always enabled, feature gate may not be set
Beta feature removed|on-by-default|-|feature enabled unless `--feature-gates=<feature>=false`|feature does not exist
Alpha API introduced|-|off-by-default|API does not exist|alpha APIs may not be used in conjunction with emulation version
Beta API graduated to GA|off-by-default|on|API available only when `--runtime-config=api/v1beta1=true`|API `api/v1` available
Beta API removed|off-by-default|-|API available only when `--runtime-config=api/v1beta1=true`|API `api/v1beta1` does not exist
on-by-default Beta API removed|on-by-default|-|API available unless `--runtime-config=api/v1beta1=false`|API `api/v1beta1` does not exist
API Storage version changed|v1beta1|v1|Resources stored as v1beta1|Resources stored as v1
new CEL function|-|function in StoredExpressions CEL environment|CEL function does not exist|Resources already containing CEL expression can be evaluated
introduced CEL function|function in StoredExpressions CEL environment|function in NewExpressions CEL environment|Resources already containing CEL expression can be evaluated|CEL expression can be written to resources and can be evaluted from storage

- Other cases we will test are:
  - `--emulation-version=<N-2>` - fails flag validation, binary exits
  - `--emulation-version=<N+1>` - fails flag validation, binary exits
  - we only allow data into new API fields once they existed in the previous release, this needs to account for emulation version
  - we only relax validation after the previous release tolerates it, this needs to account for emulation version

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

For e2e testing, we intend to run e2e tests from the N-1 minor version of kubernetes
against the version being tested with --emulation-version set to the N-1 minor versions.

This is a new kind of testing-- it requires running the tests from a release branch against
the the branch being tested (either master or another release branch).

We intend to have this up and running for Beta

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Emulation version support for N-1 minor versions
- Integration tests completed and enabled

#### Beta

- Initial cross-branch e2e tests completed and enabled
- Emulation version support for N-3 minor versions
- Min compatibility version support for N-3 minor versions
- Clients send version number and servers report out-of-allowance skew to a metric
  (Leveraging work from KEP-4355 if possible)
- All existing features migrated to versioned feature gate - [kubernetes #125031](https://github.com/kubernetes/kubernetes/issues/125031)
- Verification machinery added - [kubernetes #125032](https://github.com/kubernetes/kubernetes/issues/125032) 
- Integrate [test/featuregate_linter](https://github.com/kubernetes/kubernetes/blob/35488ef5c7212a3d491b86e02b1ba05dbbc4b894/test/featuregates_linter/README.md) into golangci-lint

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

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

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

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: CompatibilityVersions
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager
    - kube-scheduler


###### Does enabling the feature change any default behavior?

Yes, `/version` will respond with `binary{Major,Minor}` and `minCompatibility{Major,Minor}` fields.
This addition of fields should be handled by clients in a backward compatible
way, and is a relatively safe change.

No other default behavior is changed.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Note that when the `--emulation-version` or `--min-compatibility-version` flags is
set, the feature flag must be turned on (when feature is in Alpha). So to
disable the feature, the flag must also be removed.

###### What happens if we reenable the feature if it was previously rolled back?

Behavior is as expected, `--emulation-version` or `--min-compatibility-version`
may be set again.

###### Are there any tests for feature enablement/disablement?

Yes, feature enablement/disablement will be fully tested in Alpha.

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

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

