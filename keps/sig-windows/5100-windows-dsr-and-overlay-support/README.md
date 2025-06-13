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
# KEP-5100: [RETROACTIVE] DSR and Overlay support in Windows kube-proxy

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
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [DSR Enablement](#dsr-enablement)
  - [Overlay support](#overlay-support)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [X] e2e Tests for all Beta API Operations (endpoints)
  - [X] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [X] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [X] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

Add support for DSR (Direct Server Return) and Overlay networking mode support for Windows kube-proxy.

Support for both of these features was added in K8s v1.14 without a KEP.
This KEP is to retroactively document the changes made to Windows kube-proxy to support these features and provide a path for promoting these features to GA.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

DSR support was added to Windows Server 2019 as part of the May 2020 update.
DSR provides performance optimizations by allowing the return traffic routed through load balancers to bypass the load balancer and respond directly to the client; reducing load on the load balancer and also reducing overall latency.
More information on DSR on Windows can be found [here](https://techcommunity.microsoft.com/blog/networkingblog/direct-server-return-dsr-in-a-nutshell/693710).

Overlay networking mode is a common networking mode used in Kubernetes clusters and is required by some for some important scenarios like network policy support with Calico CNI.
Adding support for overlay networking mode in Windows kube-proxy will allow users to use more CNI solutions with Windows nodes.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

Enable DSR and overlay networking on Windows nodes running kube-proxy in Kubernetes clusters.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

DSR and Overlay networking mode support is already implemented in Windows kube-proxy and has been extensively tested in the Windows CI pipeline.
This proposal is to promote the existing implementations to GA.

Additionally, DSR support on Windows is supported on both EKS and AKS.
Both DSR and overlay networking support have been used in the Windows CI pipelines running release-informing jobs since K8s v1.20.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a cluster administrator, I want to enable DSR functionality on Windows nodes in order to reduce load in the Host Network Service and reduce latency for client requests.

#### Story 2

As a cluster administrator, I want to be able to enable network policy on Windows nodes which requires overlay networking mode support in kube-proxy for some CNI solutions.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

Overlay networking mode is not compatible with dualstack networking on Windows.

If kube-proxy is started with both overlay networking mode and dualstack networking enabled, a warning message will be added and ip address space with be downgraded to ipv4 only. This is existing behavior and has not caused any reported issues.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

Enabling DSR and overlay networking mode support in Windows kube-proxy both have very little risk.

For DSR, the Windows Host Network Service handles all of the logic for managing network traffic; kube-proxy only needs to specify if DSR should be used when creating/sycing load balancer rules.
Additionally, DSR must be enabled with a kube-proxy command switch (--enable-dsr=true) disabling DSR is can be performed by redeploying kube-proxy on Windows nodes.

Overlay networking support in Windows has been used in the Windows CI pipelines running release-informing [capz-windows-master](https://testgrid.k8s.io/sig-windows-signal#capz-windows-master) jobs since K8s v1.20.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

Since the functionality is already implemented, the design details section will cover the current implementation.

### DSR Enablement

DSR is enabled by passing `--enable-dsr=true` as a command line switch to the Windows kube-proxy.
Prior to GA, kube-proxy will ensure that `WinDSR=true` is specified in the feature-gates and will fail to start if DSR is enabled without that.

Checks for terminating and service endpoints handle DSR traffic differently than non-DSR traffic to adhere to behavior defined in [KEP-1669: Proxy Terminating Endpoints](https://github.com/kubernetes/enhancements/issues/1669)
- Local endpoints will be skipped when determining if all endpoints for a service are terminated if DSR is enabled and service type is load balancer.
- Non-local endpoints will be skipped when considering if all endpoints for a service are non-serving if DSR is enabled and service type is load balancer.

Flags passed to HNS (Host Networking Service) calls used for the following operators will be updated to include a flag indicating if DSR is enabled for all get, create, and update loadbalancer HNS calls.


### Overlay support

To enable overlay networking on Windows nodes, HNS network created on the node prior to starting kube-proxy and specified by `$KUBE_NETWORK` should be of type `Overlay`.
Prior to GA `WinOverlay=true` must be specified in the kube-proxy feature gates.
If the specified network is of type `Overlay` and the the feature gate is not set, kube-proxy will log an error and fail to start.

Addintionally, in overlay networking node, kube-proxy needs to know the source IP address of the traffic it is proxying by setting `--source-vip=$sourceVIP` on the kube-proxy command line.

Creating the endpoint varries by CNI implementation and here are two examples:

- For Flannel, the endpoint is created prior to starting kube-proxy like in this [example](https://github.com/kubernetes-sigs/sig-windows-tools/blob/3018559a4f396972a6c89b588f6b5fab030b72f6/hostprocess/flannel/kube-proxy/start.ps1#L6-L46)
- For Calico, the endpoint is crated by the node agent and queried by name prior to starting kube-proxy like in this [example](https://github.com/kubernetes-sigs/sig-windows-tools/blob/3018559a4f396972a6c89b588f6b5fab030b72f6/hostprocess/calico/kube-proxy/start.ps1#L76C1-L90C2)

Once kube-proxy is running in overlay networking mode, the specified source VIP will sometimes be used on in load balancer policy rules based on the backend endpoints using the following logic:

a) Backend endpoints are any IP's outside the cluster ==> Choose Node's IP as the source VIP
b) Backend endpoints are IP addresses of a remote node => Choose Node's IP as the source VIP
c) Everything else (Local POD's, Remote POD's, Node IP of current Node) ==> Choose the specified source VIP

Everything else is handled by the Windows HNS.

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

Kube-proxy for Windows must run on Windows machines so coverage is not reported in ci-kubernetes-coverage-unit. 
This coverage data was run manually on a Windows Server 2022 machine:

- k8s.io/kubernetes/pkg/proxy/winkernel: 2025-02-11 - 58.8% of statements


##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

Functionality described in this KEP require Windows nodes and are primarily validated with unit and e2e tests.
The Kubernetes project does not currently have support for running integration tests for Windows specific code-paths.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

All Windows nodes running kube-proxy in https://testgrid.k8s.io/sig-windows-master-release#capz-windows-master have DSR and overlay networking configured.


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

-->
#### Alpha

N/A - This feature is already implemented.

#### Beta

- Test passes on testgrid with WinDSR and Winoverlay enabled on Windows nodes are running regularly.
- Unit tests validating expected behavior for both DSR and overlay networking mode are added.
  - For DSR, unit tests validating feature gate is set correctly and that the correct flags are passed to HNS calls will also be added.

#### GA

- 2 or more CNI solutions support overlay networking mode for Windows nodes.

    1. [Calico networking](https://github.com/projectcalico/calico/blob/cf9455706a1fc6e9d5b11e4556f3d087007124e7/node/windows-packaging/CalicoWindows/kubernetes/kube-proxy-service.ps1#L65-L79) on Windows enables WinOverlay feature if the underlying HNS network is of type overlay.
    2. [Flannel](https://github.com/microsoft/SDN/blob/master/Kubernetes/flannel/start-kubeproxy.ps1#L19-L28) networking does the same.

<!--

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

For DSR `--enable-dsr=true` must be passed as a kube-proxy command line switch to enable the functionality.
This means that the upgrade/downgrade strategy is the redeploy kube-proxy with the appropriate configuration.

For overlay networking mode the entire cluster must be configured for overlay networking so cluster it is not possible for upgrade / downgrade this functionality on a per-node basis.

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

N/A - As long as the all nodes are configured for overlay networking mode, there is no version skew strategy required since networking APIs are not changing.

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

For DSR support:

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: WinDSR
  - Components depending on the feature gate: kube-proxy
- [x] Other
  - Describe the mechanism: DSR is enabled by passing `--enable-dsr=true` as a command line switch to the Windows kube-proxy.
  - Will enabling / disabling the feature require downtime of the control
    plane? no
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? Yes, there will be a slight period where network traffic might not be routed correctly while kube-proxy is restarted.
    Kube-proxy will rules will be re-synced with/without DSR support when kube-proxy is starting up.
    Nodes that handle network traffic show be drained before toggling DSR support.

For overlay networking mode:

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: WinOverlay
  - Components depending on the feature gate: kube-proxy
- [x] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane? Yes and no - The HNS network used by kube-proxy must be re-created with the correct type before starting kube-proxy which can disrupt network traffic but also all nodes in a cluster must use the same network type so it is not possible to switch between overlay and bridge networking on a per-node basis.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? See above.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No.
For DSR, `--enable-dsr=true` must be passed as a kube-proxy command line switch to enable the functionality.
For overlay networking supprt, behavior changes only occur if the HNS network used by kube-proxy is of type `Overlay` which would only be done intentionally as part of joining nodes to a cluster.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

For DSR, yes, DSR can be disabled by passing `--enable-dsr=false` as a kube-proxy command line switch and restarting kube-proxy.

For Overlay, no, overlay networking mode cannot be disabled on a per-node basis. All nodes in a cluster must use the same network type so it is not possible to switch between overlay and bridge networking on a per-node basis.

###### What happens if we reenable the feature if it was previously rolled back?

For DSR, kube-proxy should resync HNS rules and start using DSR again.

###### Are there any tests for feature enablement/disablement?

We have periodic test passes running in prow that use both of these configurations

- [capz-windows-master-containerd2](https://testgrid.k8s.io/sig-windows-signal#capz-windows-master-containerd2) all of the Windows CAPZ tests use calico by default.
- [ltsc2025-containerd-flannel-sdnoverlay-master](https://testgrid.k8s.io/sig-windows-networking#ltsc2025-containerd-flannel-sdnoverlay-master) for flannel with overlay networking mode.

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

For overlay, no, because the feature requires the cluster to be configured for overlay networking mode and cannot be enabled on a per-node basis.

For DSR, unit tests will be added to validate that DSR is enabled and disabled correctly and that the correct flags are passed to HNS calls for each case.
These will be required for the feature to move to beta.

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

For DSR a rollout or rollback should not fail. Nodes can operate with DSR enabled or disabled per node in a cluster.

For overlay networking mode support, a rollout can fail if the CNI configuration for the node and kube-proxy configuration are not in sync. This would cause nodes to never go into the Ready state.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

Node ready state should be monitored to ensure nodes join the cluster and are properly configured to start running pods.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

For DSR support yes, manual verification was done to ensure that DSR can be enabled and disabled on a node.

The steps for the manual validation went as followed:

- Create a cluster with 1 Linux control plane node and 2 Windows worker nodes.
- Deploy a kube-proxy deamonSet with `--feature-gates=WinDSR=true` and `--enable-dsr=true` to Windows worker nodes.
- Deploy IIS (Internet Information Services) on both Windows worker nodes and expose the service with a LoadBalancer service.
- Once the service IP became available, test that the service is from the each Windows node and outside of the cluster.
- Redeploy the kube-proxy deamonSet with `--enable-dsr=false` to Windows worker nodes.
- Wait for Kube-proxy to start and test that the service is still reachable from each Windows node and outside of the cluster.
- Redeploy the kube-proxy deamonSet with `--enable-dsr=true` to Windows worker nodes.
- Wait for Kube-proxy to start and test that the service is still reachable from each Windows node and outside of the cluster.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No

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

If configured for use, both DSR and overlay networking will be used by any workloads that communicate with other pods/services in the cluster.

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
- [x] Other (treat as last resort)
  - Details: Pod-to-Pod and Pod-to-Service traffic will not route correctly if the feature is not working.

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

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [X] Other (treat as last resort)
  - Details: Monitoring of workload-specific network traffic to ensure that traffic is being routed correctly.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

No

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

DNS and CNI solutions must be deployed in the cluster.

Both DSR and overlay networking modes are supported for all patch versions of Windows Server 2022 and Windows Server 2025.
DSR requires Windows Server 2019 with May 2020 updates (or later).

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

No

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

Enabling DSR will increase the number of IP addresses in use on each node by 1 for the VIP used to route return traffic.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

A troubleshooting guide for general Windows networking issues can be found at https://learn.microsoft.com/en-us/troubleshoot/windows-server/software-defined-networking/troubleshoot-windows-server-software-defined-networking-stack

https://github.com/microsoft/SDN/ contains some additional troubleshooting scripts to collect detailed information and can help in troubleshooting
- https://github.com/microsoft/SDN/blob/master/Kubernetes/windows/hns.v2.psm1 is a powershell module with cmdlets for inspecting HNS policies and endpoints
- https://github.com/microsoft/SDN/blob/master/Kubernetes/windows/helper.psm1 contains useful helper functions for troubleshooting
- https://github.com/microsoft/SDN/tree/master/Kubernetes/windows/debug contains various powershell scripts for enabling tracing, collectings stats and perf counters, starting packet captures, etc

Troubleshooting issues with Direct Server Return (DSR) on Windows:

- Ensure that the kube-proxy command line switch `--enable-dsr=true` is set and `--feature-gates=WinDSR=true` is set.
- Inspect kube-proxy logs for any warnings or errors
- If everything looks correct, log onto the node and inspect the HNS rules to ensure DSR is enabled for the load balancer rules.
  - Log onto the node and use `hnsdiag.exe list loadbalancers -d` to list all the load balancers and details about their rules.
    You should see `IsDSR:true` for load balancer policies proxied by kube-proxy.
  - You can use `hnsdiag.exe` to get detailed information about local networks and endpoints in addition to loadbalancers.
- If you are still having issues create an issue at https://github.com/microsoft/windows-containers

Troubleshooting issues with overlay networking mode on Windows:

- Ensure that the CNI solution has either created a HNS network of type `Overlay` or that instructions provided by the CNI solution have been followed to create the network.
- Ensure that the name of the network created above is passed to kube-proxy with the `$Env:KUBE_NETWORK` environment variable.
- Check kube-proxy logs for any warnings or errors.
- If everything looks correct, log onto the node and inspect the HNS rules to ensure that the source VIP is being used correctly.
  - Log onto the node and use `hnsdiag.exe list loadbalancers -d` to list all the load balancers and details about their rules.
    You should see the source VIP being used for load balancer policies proxied by kube-proxy.
  - You can use `hnsdiag.exe` to get detailed information about local networks and endpoints in addition to loadbalancers.
- If you are still having issues create an issue at https://github.com/microsoft/windows-containers 

###### How does this feature react if the API server and/or etcd is unavailable?

This feature does not change the functionality of kube-proxy or other Kubernetes components if the API server or etcd is unavailable. Kube-proxy would retain the existing behavior if the API server or etcd is unavailable, which would result in new Pod and Service endpoints not routing correctly on the nodes.

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

We have not observed any additional failure modes with DSR or overlay networking mode support on Windows nodes.

###### What steps should be taken if SLOs are not being met to determine the problem?

See [Troubleshooting](#troubleshooting)

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

- **2019-02-20** - DSR and overlay networking mode support added to Windows kube-proxy (k/k PR [#70896](https://github.com/kubernetes/kubernetes/pull/70896)
- **2025-01-28** - [KEP #5100](https://github.com/kubernetes/enhancements/issues/5100) created to document the changes made to Windows kube-proxy to support DSR and overlay networking mode support and provide a path for promoting these features to GA.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

The functionally described in this KEP is already implemented and used by various cloud providers so there are no drawbacks to not implementing it.
The drawbacks for not progressing the features to GA are that this functionality may get removed from kube-proxy in the future which would result in Windows not being able to support some CNI solutions (Calico networking with network policy support) and not being able to take advantage of DSR performance optimizations.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

This functionality has already merged into k/k so other alternatives have not been considered.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
