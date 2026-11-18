# KEP-5334: Image Pull Progress

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
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha 2](#alpha-2)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
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

This proposal introduces new CRI and Kubelet APIs to allow users to retrieve the image pull progress for a pod.
The kubelet tracks the image pulls of each pod and serves their progress to the client.
While a client is watching, the kubelet subscribes to the CRI component for the layer progress of the images that are being pulled.

## Motivation

In certain specialized environments, container images may be exceptionally large.
This can lead to a poor user experience, as deploying a workload may trigger the pulling of a large image,
causing unexpected delays and limited visibility into the progress of the operation.

### Goals

- Extend the CRI to include an API for reporting image pull progress.
- Implement support for the image pull progress API in CRI implementations (e.g. Containerd and CRI-O).
- Update `crictl` to support monitoring of image pull progress.
- Enhance the Kubelet to provide an API endpoint for image pull progress.
- Update `kubectl` to support monitoring of image pull progress.

### Non-Goals

None.

## Proposal

I propose that the kubelet tracks the image pulls of each pod and serves their progress as a stream, similar to how it serves container logs.
For the images that are being pulled, the kubelet subscribes to a new streaming CRI API and forwards the layer progress pushed by the CRI implementation.
The kubelet subscribes only while a client is watching.
This approach ensures that the CRI implementation only reports progress when a client actively requests it,
avoiding unnecessary overhead when the feature is not in use.

### User Stories (Optional)

#### Story 1

A user deploys an application to a Kubernetes cluster.
As the Pod is scheduled and created,
the only event visible in the Pod's event list is "Pulling image",
with no indication of progress or estimated time to completion.

#### Story 2

Allow kubectl and crictl to see image pull progress like other container tools.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

The introduction of image pull progress reporting carries minimal risk and does not impact any existing Kubernetes functionality.

If the underlying runtime does not support image pull progress reporting, the kubelet will still report the pull states that it tracks itself, only without layer progress. Image pulls are not affected.
FeatureGates and Kubelet configuration options allow operators to enable or disable the feature as needed, minimizing unintended exposure.
The runtime is only asked for progress while a client is watching; otherwise, the only overhead is the pull states that the kubelet keeps in memory for each pod.

Security and UX reviews should be conducted by SIG Node and SIG Architecture, with input from other relevant SIGs as needed.

Documentation will be updated to reflect new FeatureGates, Kubelet configuration options, and any performance considerations related to image pull progress reporting.

## Design Details

A container does not exist in the runtime while its image is being pulled,
and states such as waiting in the image pull queue or in back-off are only known to the kubelet.
The kubelet therefore tracks the image pulls of each pod and serves their progress itself, in the same way as it serves container logs.
The CRI implementation is only asked for the layer progress of an image that it is pulling.

1. `kubectl` sends a `GET` request to the `imagepullprogress` subresource of the pod.
2. The API server proxies the request to the kubelet of the node on which the pod runs.
3. The kubelet responds with the pull states of the images of the pod, which it tracks itself.
4. For each image that is being pulled, the kubelet subscribes to the CRI implementation and adds the layer progress to the response.

**Proposed changes to the cri:**

This API will stream the progress of an in-flight image pull.
It is part of the `ImageService` and is keyed by image; the kubelet maps images to pods and containers.

``` proto
// ImageService defines the public APIs for managing images.
service ImageService {
    ...

    // ImagePullProgress streams the progress of an in-flight pull of an image.
    rpc ImagePullProgress(ImagePullProgressRequest) returns (stream ImagePullProgressResponse) {}

    ...
}

message ImagePullProgressRequest {
    // Spec of the image, the same as in the PullImageRequest of the pull.
    ImageSpec image = 1;
}

message ImagePullProgressResponse {
    // The first response contains all layers known so far, later responses the layers that changed.
    repeated ImageLayerProgress layers = 1;
}

message ImageLayerProgress {
    // Digest of the layer (e.g., "sha256:abc123...").
    string digest = 1;
    // Phase of the layer.
    ImageLayerPhase phase = 2;
    // Number of bytes of the layer that have been downloaded.
    uint64 progress = 3;
    // Size of the layer in bytes, 0 if it is not known.
    uint64 total = 4;
}

enum ImageLayerPhase {
    // The download of the layer has not started.
    LAYER_WAITING     = 0;
    // The layer is being downloaded.
    LAYER_DOWNLOADING = 1;
    // The layer is downloaded and is being extracted. Implementations may skip this phase.
    LAYER_EXTRACTING  = 2;
    // The layer is available on the node, also if it was already there before the pull.
    LAYER_DONE        = 3;
}
```

- The request carries the same `ImageSpec` as the `PullImage` request, and the CRI implementation resolves it in the same way.
- The first response is a snapshot of all layers known at that time, so that a subscriber can join at any time during the pull.
  Each later response contains the layers that changed. All values are absolute.
- The stream ends when no pull of the image is in flight any more.
  If none is in flight when the request arrives, the stream ends without a response.
  The result of the pull is not part of the stream; `PullImage` returns it as before.
- A subscriber must not affect the pull: a slow or disconnected subscriber neither delays it nor makes it fail.
  The CRI implementation may skip intermediate updates for a slow subscriber.
- The frequency of progress updates is determined by the implementer, balancing the need for timely information with performance considerations.

**Proposed changes to the crictl:**

- Add a new subcommand `crictl image-pull-progress <image>` to allow users to monitor the progress of an in-flight pull of an image.
  - The subcommand will call `ImagePullProgress`, providing a streaming view of progress updates until the pull ends.

<!--
  The proposed `image-pull-progress` subcommand are preliminary and subject to further discussion and refinement.
 -->

**Proposed changes to the apiserver:**

Add a new `imagepullprogress` subresource for Pod objects.

- The API server will proxy requests to `/api/v1/namespaces/{namespace}/pods/{pod}/imagepullprogress` to the corresponding kubelet.
- This subresource will provide a streaming `GET` endpoint for image pull progress updates, in the same way as the existing `log` subresource.
- No changes to core Pod API types are required; this is implemented as a proxy subresource.
- RBAC and authorization will follow the same model as the `log` subresource.

**Proposed changes to the kubelet:**

It is expected to be implemented after the CRI implementation supporting the CRI API is released.

- Track the image pulls of each pod.
  For each container (init, regular and ephemeral) and each image volume of a pod, the kubelet records the state of its image:
  - `Waiting`: The pull has not started.
    The kubelet handles the images of a pod one after another, so it may not have reached this container or image volume yet (e.g., an earlier init container is still running),
    or the pull is waiting in the image pull queue of the kubelet.
  - `Pulling`: The runtime is pulling the image.
  - `Pulled`: The image was pulled.
  - `AlreadyPresent`: The image was already on the node and no pull was needed.
  - `Failed`: The pull failed, or the image cannot be used for another reason (e.g., it is not on the node and the pull policy is `Never`).
  - `BackOff`: The kubelet is waiting before it pulls again after a failed pull.

  The states are kept in memory and are removed together with the pod.
  After a kubelet restart, the kubelet reports `AlreadyPresent` for the images of the containers that were started before the restart, and for the image volumes of their pods.
- Subscribe to the CRI implementation on demand.
  While clients are watching, the kubelet calls `ImagePullProgress` for each watched image that is in the `Pulling` state.
  A subscription is shared by all clients, and the kubelet keeps the latest progress of each layer so that a client that connects later also gets a snapshot.
  If the stream ends while the image is still in the `Pulling` state (e.g., the CRI implementation has not registered the pull yet), the kubelet subscribes again after a short delay.
  When the last client disconnects, the kubelet closes the subscription.
  If the CRI implementation does not support `ImagePullProgress`, the kubelet reports the states without layer progress.
- Add new routes to the Kubelet server.
  - `/imagePullProgress/{podNamespace}/{podID}`
    This endpoint will function similarly to the logs endpoint:
    it accepts `GET` requests only and the kubelet writes the response itself,
    providing a server-push stream of image pull progress updates for all containers and image volumes of the specified pod.
  - `/imagePullProgress/{podNamespace}/{podID}/{containerName}`
    Same as the previous one, but for a single container.

  The response starts with the current states and continues with an update for each change.
  It ends when all images of the request are in the `Pulled` or `AlreadyPresent` state, or when the pod terminates or is deleted.

Unlike PortForward, which is limited to Pod-level operations, and Logs/exec, which are restricted to Container-level operations (with the client typically selecting a default container), this feature supports both Pod-level and Container-level image pull progress reporting.

**Data format served by the kubelet:**

The response is a stream of JSON objects, one per line.

``` golang
type Progress struct {
	// Name of the container, empty for an image volume.
	Container string `json:"container,omitempty"`
	// Name of the image volume, empty for a container.
	Volume string `json:"volume,omitempty"`
	// Image reference from the pod spec (e.g., "registry.example.com/app:v1").
	Image string `json:"image"`
	// State of the image: Waiting, Pulling, Pulled, AlreadyPresent, Failed or BackOff.
	State string `json:"state"`
	// Error message, only in the Failed and BackOff states.
	Error string `json:"error,omitempty"`

	// Digest of a layer of the image (e.g., "sha256:abc123...").
	Layer string `json:"layer,omitempty"`
	// Phase of the layer: Waiting, Downloading, Extracting or Done.
	Phase string `json:"phase,omitempty"`
	// Number of bytes of the layer that have been downloaded.
	Progress int64 `json:"progress,omitempty"`
	// Size of the layer in bytes, 0 if it is not known.
	Total int64 `json:"total,omitempty"`
}
```

- An object without `Layer` reports the state of the image of a container or image volume.
- An object with `Layer` reports the progress of one layer of an image in the `Pulling` state, as received from the CRI implementation.
- All values are absolute, so the kubelet may skip intermediate updates for a slow client.

For example, while the image of an image volume is being pulled:

``` json
{"volume":"model","image":"registry.example.com/model:v1","state":"Pulling"}
{"volume":"model","image":"registry.example.com/model:v1","state":"Pulling","layer":"sha256:abc123...","phase":"Downloading","progress":1048576,"total":52428800}
{"container":"app","image":"registry.example.com/app:v1","state":"Waiting"}
```

Clients maintain a progress map keyed by container or image volume and by layer, update the corresponding entry on each object, and render a progress indicator for each entry.

**Proposed changes to the kubectl:**

It is expected to be implemented on Alpha 2

- Add a new subcommand `kubectl alpha image-pull-progress <pod-name>` to allow users to monitor the image pull progress of a pod.
  - Include a `--container` flag to specify a particular container within the pod.
  - The subcommand will follow the conventions of existing `logs` and `exec` commands, providing a streaming view of progress updates.
  - By default, the image pull progress for all containers and image volumes in the pod will be displayed.

<!--
  The proposed `image-pull-progress` subcommand are preliminary and subject to further discussion and refinement.
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

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

The implementation PR will include unit test and e2e tests to ensure the correctness and reliability of the new functionality.

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

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

### Graduation Criteria

#### Alpha

- Update the CRI API to support image pull progress
- Add image pull progress functionality to containerd
- Add support for image pull progress in crictl

#### Alpha 2

- Implement image pull progress support in kubelet
- Add the image pull progress subcommand in kubectl

#### Beta

- CRI implementations (Containerd and CRI-O) support image pull progress reporting and have passed integration and e2e tests
- Sufficient e2e test coverage demonstrating stability

#### GA

- N examples of real-world usage showing adoption and value
- At least 2 releases in beta to gather feedback
- All feedback from beta phase addressed
- Documentation complete and up-to-date
- No outstanding high severity issues

### Upgrade / Downgrade Strategy

There are no specific upgrade/downgrade order requirements for this feature. 
The image pull progress functionality will only work when kubectl, the API server and the kubelet support it.
If any of them doesn't support the feature, the progress request will fail gracefully with an "unsupported" error,
but normal image pull operations will continue to work as before without the progress reporting capability.
If only the container runtime doesn't support the feature, the progress request will still work, but it will report the pull states without layer progress.

### Version Skew Strategy

- Older apierver: Will respond with "404 Not Found" for progress requests since they don't implement the new subresource
- Older kubelet: Will respond with "404 Not Found" for progress requests since they don't implement the new API
- Older runtime: Will respond with "Unimplemented" for `ImagePullProgress` since they don't implement the new CRI API; the kubelet will then report the pull states without layer progress
- Newer kubectl: Will gracefully handle "404 Not Found"/"405 Method Not Allowed" responses by treating them as feature unsupported on target kubelet
- Older kubectl: Versions without progress subcommand won't have this functionality available
- Mixed version clusters: Feature works where supported and gracefully degrades where unsupported

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

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ImagePullProgress
  - Components depending on the feature gate:
    - kubelet
- [x] KubeletConfiguration: `enableDebuggingHandlers: false`
  - Describe the mechanism:
    - This feature can be disabled via the kubelet configuration's `enableDebuggingHandlers` setting, similar to other debugging features like attach, exec, logs and port-forward
  - Will enabling / disabling the feature require downtime of the control
    plane?
    - No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
    - No.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

It should continue to work as expected.

###### Are there any tests for feature enablement/disablement?

Unit tests.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

No.

###### What specific metrics should inform a rollback?

No.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

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

No, the new interface will have no impact unless it is explicitly invoked.

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

No, enabling or using this feature will not introduce any new API calls beyond those explicitly invoked for image pull progress.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No.

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

No.

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
