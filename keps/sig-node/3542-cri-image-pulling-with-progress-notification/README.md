# KEP-3542: CRI Image Pulling with Progress Notification

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Kubelet defaults](#kubelet-defaults)
  - [Kubelet config](#kubelet-config)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
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

Introduce new CRI API call for downloading a container image with possibility of pulling progress
reports being sent back to requestor and / or no-progress timeout. It should be possible to use
both the progress reporting and no-progress timeout together as well as separately. For instance,
the runtime should send back messages about image pulling progress with information
on how much data was downloaded and, if known, what is the current estimated total size of download, and
report a failure at any point after M consecutive seconds of no data being downloaded.

## Motivation

In High Performance Computing and other specialized environments container
images often get extremely big. Pulling several such images even over high speed
links can take a lot of time, tens of minutes or more. This often results
in a bad user experience when deploying a workload triggers pulling such
an image as a side-effect. For a long time the user has no visibility to
how creating the workload progresses, or if it progresses at all.

Introducing a new CRI ImageService RPC call for pulling images with a
streaming progress notification response and timing out once a timeout without
data transfer was reached would provide the low-level building blocks to improve this situation.

### Goals

- Extend CRI to provide image pulling progress API that can be utilized by client
tools (crictl pull), or machinery (kubelet pulling the image)
- Extend existing PullImage api to support a duration timeout with no progress feature
- Implement PoC / draft for CRI PullImageWithProgress call to runtime in Kubelet, hidden behind
FeatureGate, disable by default.

### Non-Goals

- Complete implementation of utilization of proposed interface additions in the client-tools and / or runtimes

## Proposal

We propose:

- extending existing ImagePull call with optional parameter:
  - no-progress timeout: number of seconds during which if completely no data transfer was ongoing, failure should be reported
- introducing new (additional, not replacing old one) API for requesting image pull,
that will return stream with periodic updates sent through it to the client until completion or
a timeout without download progress is reached. The image pull with progress request parameters will contain:
  - image pull request data structure, including new optional no-progress timeout field
  - the type of granularty based on which the client wants to receive updates about the progress:
    - time-based
    - size-based
    - none
  - interval of the updates (if any) respectively to the granularity type:
    - N seconds
    - N amount of data downloaded of the total image size, e.g. 1Gi
  - verbosity
    - summarized - single value, total current download progress against total estimated download size
    - verbose - summarized plus additionally per layer information of download progress against layer size

If / when it is possible to reliably determine percentage of the progress in the runtime,
percentage-based granularity type can be introduced then.

If the client did not specify the preferred notification granularity, default values should be used,
for instance every Gibibyte downloaded or every 60 seconds of time spent downloading an image.


### Kubelet defaults

For Kubelet implementation we propose:
- for alpha: disabling image pulling progress messages in request to Runtime
- default no-progress timeout to be 10 seconds

Example Kubelet image pull with progress request:
```
{
    request: {...},
    granularity_type: "none",
    interval: 0,
    no_progress_timeout: 10,
    verbosity: false,
}
```

### Kubelet config

The granularity type, interval, and no-progress timeout should be configurable through kubelet-config.

Suggested new Kubelet config fields are these:

       // NoProgressTimeout is a number of seconds after which stalled image download should be reported
       // as an error.
       // Supported values are:
       // - 0 for infinity, effectively no timeout
       // - positive number up to 4294967295 (uint32 max number, approx. 136 years)
       // Default: 10
       // +optional
       NoProgressTimeout int32 `json:"NoProgressTimeout,omitempty"`


For exact API objects structure, see [API design Details](#design-details) below.

### User Stories (Optional)

#### Story 1

The user is deploying an application to the k8s cluster, the Pod is being
scheduled and creation of the Pod begins. The only event occurring on the Pod
object events list is "Pulling image", with no progress indicator, or ETA.

If CRI had a possility to expose the progress and / or ETA for completing the
image pull, it would improve user experience greatly as well as debugging
experience in situations when the image source is not available and the timeout
will be inevitably reached, but not immediately: the progress events on the Pod
object would indicate if the image transfer is ongoing at all.

#### Story 2

The user is debugging an issue in k8s cluster by running `crictl pull` command
to pull the image - either to see if image is available or to check if connectivity
to the image source is in place. If the image is available and the pulling
is ongoing - no indication is possible to show based on CRI itself. At least
containerd runtime is already providing such information based on the
`ctr image pull docker.io/library/busybox:latest`.

#### Story 3

Particular registry is blocked by a firewall in the organization where the Kubernetes cluster is
running, and workload is trying to use an image that is impossible to download without explicit
network error. Pod in such situation will be pending the image download for a long time, which can
be shortened to a no-progress timeout set in ImagePullWithProgress call.

### Notes/Constraints/Caveats (Optional)

It is fairly impossible to get progress about unpacking the image layers, therefore for
- kubelet:  this operation can be silent without progress reports
- command line tools - stage of unpacking can be reported with verbose option in request

In case of lazy pulling, which allows starting the container while its image not yet fully downloaded,
the minimum amount of data the runtime needs to pull (as opposed to total / complete image size)
in order to start the container should be considered the total amount of download being in progress.
For instance, if the complete container image consists of 10 layers, 1 GiB per layer, and runtime
can start the container with only 4 layers pulled, then 4GiB should be considered the total download
amount and the progress message to Kubelet at hypothetical 1GiB progres point would look like:
```
{
  image_ref: "registryA.com/repo/imageB:tagC",
  offset: 1073741824,
  total: 4294967296,
}
```
When downloading the minimum needed layers to start the container is done, the progress reports
stream can be closed and pull operation can proceed silently in the runtime without notifications
sent to Kubelet for the rest of the image being lazily pulled as needed.

If ImagePullProgress is enabled for kubelet but runtime does not support it, the fallback should
use regular silent ImagePull API call.

### Risks and Mitigations

The risks are minimal, introduction of new API is not affecting any other APIs.

Misconfiguration of image pull request granularity for kubelet can result in big number
of progress responses published by runtime into return stream. This can be mitigated by kubelet
throttling the amount of events it publishes to Pod object (publishing less than received from
runtime), as well as in runtime by metering amount of responses being sent within time interval.
Intention of this KEP's CRI extension is to provide not-so-frequent updates for Kubelet (e.g. once
a minute), while having a possibility to have more-frequent updates for cli tools (e.g. once a
second).

Documentation should be updated to reflect new FeatureGates, Kubelet config options, and performance
overhead for reporting progress from image pulls.

Runtime can be such that does not support image pull progress reporting. In this case fallback to
regular image pulling call should happen on client side (kubelet, cli tool, other entities
requesting image to be pulled from runtime).

## Design Details

Following new CRI streaming API call is proposed:

    // PullImageWithProgress pulls an image with authentication config.
    // It returns periodically amount of image pulled so far.
    rpc PullImageWithProgress(PullImageWithProgressRequest) returns (stream PullImageWithProgressResponse) {}

The PullImageWithProgress() API call will connect and send a PullImageWithProgressRequest message to
runtime CRI image service server.

The PullImageWithProgressRequest contains base information needed to do the image pull (image name, auth config
and sandbox information), and it will also contain information how often the server should send progress reports.
The CRI client can restrict the progress reporting to be time-based (e.g. once every n. seconds),
or based on size (amount of bytes/KiB/MiB downloaded).

    message PullImageWithProgressRequest {
        // Include original non-progress request structure.
        PullImageRequest request = 1;
        // Granularity type of the progress reports. Supported values: time, size, none
        PullImageProgressGranularity granularity_type = 2;
        // The interval value of the chosen granularity.
        // For time based granularity, this is the number of seconds between reports. If time interval is 0, then runtime default report interval is used.
        // For size based granularity, this is the number of bytes received between reports. If set to 0, then runtime default report interval is used.
        UInt64Value interval = 3;
        UInt32Value no_progress_timeout = 4;
        // Summarized (false) or detailed (true) progress reports
        bool verbosity = 5;
    }

If the connection succeeds, the PullImageWithProgress() will return a gRPC stream to the caller and let it
to start to receive the progress messages via the stream. The image server will initiate an image pull, and
it will start to send progress reports to the CRI client. The progress reports will contain information how
much image has been downloaded so far.

    message PullImageWithProgressResponse {
        // Reference to the image in use.
        string image_ref = 1;
        // Amount of data received.
        UInt64Value offset = 2;
        // Total size of the image.
        UInt64Value total = 3;
        // Detailed per-layer download progress information, format to be defined later
        string details = 3;
    }


### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

The implementation PR adds a suite of unit and e2e tests.

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

- `k8s.io/kubernetes/pkg/kubelet>`: `2023-06-19` - `66.6`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- with fake runtime, image service to make sure failover to regular ImagePull happens
if runtime has ImagePullWithProgress not implemented
- with fake runtime, image service to ensure no-progress timeout error is handled


### Graduation Criteria

#### Alpha

- CRI extended with the new call and parameter for old call
- PoC feature implemented in kubelet behind a feature flag with only no-progress timeout handling, progress messages should not be requested from runtime
- PoC feature is implemented for either cri-o or containerd runtime, does not have to be released
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from community
- Full implementation of the suggested new CRI API call in kubelet is merged and released
- Implementation of the call for the crictl merged and released
- Implementation of the call for cri-o runtime merged and released
- Implementation of the call for containerd runtime merged and released
- Additional tests are in Testgrid and linked in KEP

#### GA

- TBD

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
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
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

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ImagePullProgress
  - Components depending on the feature gate: kubelet
- [x] Other
  - Describe the mechanism: the CRI-compliant runtime has to implement the call
  - Will enabling / disabling the feature require downtime of the control
    plane? - No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled). - Kubelet
    has to be restarted to pick up new configuration

###### Does enabling the feature change any default behavior?

If the container image is or has become impossible to download, pulling operation will fail faster.
<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the proposed new call is an alternative, not a replacement. The way kubelet is requesting to
pull an image should be possible to change through kubelet config.

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

Switching the feature on / off should only result in change of how much time it takes for image pull
operation to fail if there is no progress over defined period of time.

###### Are there any tests for feature enablement/disablement?

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

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout should not fail copmletely, it should fallback to existing stable PullImage call in case
the runtime does not support PullImageWithProgress call.

Rollback is disabling feature in kubelet config, no changes aside from kubelet config should be
needed.
<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

Runtime reporting callback is not implemented shold signify need for fallback to regular PullImage.

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

No.
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

This is not workload-related feature.
<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

`kubectl describe pods/pod` should in events section show that:

- [required] Image pulling has started, with progress reporting
- [optional] Image pulling is at particular point

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

In Alpha:

- image pulling operation should fail faster, when no progress in image pulling was observed over no-progress-timeout amount of time.

In Beta:

- given the default PullImageWithProgressRequest values and the maximum supported amount of Pods needing
to pull an image simultaneously and in parallel (worst case), kubelet is not under DoS attack by runtime.
- same scenario, but kubelet is not causing apiserver load.

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

No impact for alpha.

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

Yes, this is exactly the idea of this KEP.
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

No.
<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.
<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.
<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Using the no-progress timeout will shorten the time to fail when the target image cannot be downloaded and no
explicit network error is in place (when actual network error is in place, there's hardly any delay until failure).

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Not in alpha.

In Beta, Kubelet should be seeing insignificant increase in gRPC messages from Runtime.

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
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

In alpha: no behavior change.

###### What are other known failure modes?

Nothing should fail, if the feature is enabled, but not working (either kubelet or runtime does
not suppor proposed call) - the the fallback happens to the old functionality.

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

