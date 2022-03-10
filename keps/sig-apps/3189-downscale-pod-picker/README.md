# KEP-3189: Deployment/ReplicaSet Downscale Pod Picker

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
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

A new `downscalePodPicker` field will be added to `ReplicaSetSpec` and `DeploymentSpec`, this field specifies a user-created 
Pod-Picker REST API that is queried to determine which Pods are removed when replicas is decreased.

## Motivation

The idea of letting users influence how Deployments/ReplicaSets remove Pods when `replicas` are decreased has
been floating around since at least 2015, with the primary issues discussing it ([#4301](https://github.com/kubernetes/kubernetes/issues/4301),
[#45509](https://github.com/kubernetes/kubernetes/issues/45509)) having over 179 comments between them!

One of the main use-cases for this request is applications that have "pools of workers" along with a central system 
that assigns tasks to one these workers (e.g. [Apache Airflow](https://github.com/apache/airflow) and [Apache Spark](https://github.com/apache/spark)).
Naturally, if the number of tasks varies over time, the size of the pool should scale with the load.
However, Deployments/ReplicaSets are not currently application-aware when downscaling, so an active worker may be removed 
when there was an idle one (leading to wasted work).

As of Kubernetes 1.22, the `controller.kubernetes.io/pod-deletion-cost` annotation from
[KEP-2255](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/2255-pod-cost) is available in BETA.
However, this approach is not fit for purpose.

Firstly, the `pod-deletion-cost` annotation must be PATCHED on Pods BEFORE the `replicas` count is decreased, leading to the following:

1. it cannot be used with existing resources like [HorizontalPodAutoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/), 
   as this would require updating annotations before the autoscaler makes a decision to downscale
2. to use the feature, developers must create custom-autoscaler that updates annotations BEFORE decreasing `replicas`, 
   making it inaccessible to all but the most advanced Kubernetes users

Secondly, the `pod-deletion-cost` is an annotation, leading to the following:

1. a PATCH call to kube-apiserver is generated for EACH Pod annotation update, and this is not scalable for large numbers of Pods
2. annotations are permanent but `pod-deletion-cost` is transient, meaning annotations must be cleared after a scale-down (requiring yet more API calls)

These problems can be addressed if we flip the approach, and allow Deployments/ReplicaSets to "ask" an API which Pods 
are best to remove when the `replicas` count is decreased.

### Goals

- provide a scalable method for users to influence which Pods are removed when downscaling a Deployment/ReplicaSet
- improve support for autoscaling in apps that have "pools of workers"
- prevent a poorly designed (or faulty) `downscalePodPicker` from significantly impacting the Deployment/ReplicaSet behaviour
- support multiple protocols for the `downscalePodPicker`, starting with REST and expanding to gRPC

### Non-Goals

- guarantee that the Pods "chosen" by the `downscalePodPicker` will be removed (external factors may remove Pods before we do)
- guarantee that the Pods NOT "chosen" by the `downscalePodPicker` will be spared
- guarantee that `downscalePodPicker` will only be called once per downscale
- influence Node-removal decisions of systems like cluster-autoscaler
- cache results from `downscalePodPicker` (we will call the API every time `replicas` is decreased)


## Proposal

- A new `downscalePodPicker` field will be added to `ReplicaSetSpec` and `DeploymentSpec`.
- This field specifies a REST API that the kube-controller can access to inform which Pods it removes when `replicas` is decreased.

### User Stories (Optional)

#### Story 1

As a Data Platform Engineer, I want to run [Apache Airflow](https://github.com/apache/airflow) on Kubernetes and
autoscale the number of workers while killing the least active workers on downscale, this allows me to save money by
not under-utilizing Nodes, and reduce wasted time by minimising how many worker-tasks are impacted by scaling.

Solution:

- I can run my Airflow celery workers in a Deployment.
- I can use a [ScaledObject](https://keda.sh/docs/2.5/concepts/scaling-deployments/#scaledobject-spec) from
  [KEDA](https://github.com/kedacore/keda) to create a HorizontalPodAutoscaler that scales replicas based
  on current worker task load, using the [PostgreSQL Scaler](https://keda.sh/docs/2.5/scalers/postgresql/).
- I can create a REST API (with Python) for `downscalePodPicker` that runs in a Deployment, and answers a request to choose `N` Pods by:
   1. querying the Airflow Metadata DB to find how many tasks each worker is doing (weighting longer-running tasks higher)
   2. finding the `N` workers with the lowest weighting:
       - if multiple workers have the same weighting, return them as "tied pods"
       - if we find `N` workers doing nothing, we can exit early, and return those as "chosen pods"
   3. returning these lists of "chosen pods" and "tied pods"

#### Story 2

As a Platform Engineer, I want to run a sharded Minecraft server on Kubernetes and autoscale the number of shards
while impacting the fewest connected users on downscale, this allows me to save money by not under-utilizing Nodes,
and improve user-experience by minimising the number of users impacted by scaling.

Solution:

- I can run my Minecraft server shards in a Deployment.
- I can use my in-house solution to control how many `replicas` the Deployment has.
- I can create a REST API (with Java) for `downscalePodPicker` that runs in a Deployment, and answers a request to choose `N` Pods by:
   1. keeping an in-memory cache of how many users are on each shard (weighting "premium" users higher)
   2. finding the `N` shards with the lowest user-load:
       - if multiple shards have the same weighting, return them as "tied pods"
       - if we find `N` empty shards, we can exit early, and return those as "chosen pods"
   3. returning these lists of "chosen pods" and "tied pods"

#### Story 3

As a Data Engineer, I want to run an [Apache Spark](https://github.com/apache/spark) cluster and autoscale the number of workers
while impacting the fewest running tasks on downscale, this allows me to save money by not under-utilizing Nodes,
and reduce wasted time by minimising how many tasks are impacted by scaling.

Solution:

- (Similar to Story 1)

#### Story 4

As a Site Reliability Engineer, I want to ensure my NodeJS website maintains regional distribution when downscaling the
number of replicas, this allows me to ensure uptime when a region experiences an outage.

Solution:

- I can run my NodeJS application in a Deployment.
- I can use a HorizontalPodAutoscaler with CPU metrics to control how many `replicas` the Deployment has.
- I can create a REST API (with TypeScript) for `downscalePodPicker` that runs in a Deployment, and answers a request to choose `N` Pods by:
   1. keeping track of which region each Pod is in
   2. finding `N` Pods that we can remove without violating the regional distribution requirements:
       - if there are multiple acceptable options, we could choose the `N` Pods which are doing the least work
   1. return these Pods as the "chosen pods"

### Notes/Constraints/Caveats (Optional)

- The `downscalePodPicker` APIs are written by users to suite their specific needs, and are not part of Kubernetes.
- There are very few constraints on what a `downscalePodPicker` APIs can be, for example:
   - they may be fully external to the Kubernetes cluster (e.g. running on an external metrics system)
   - they may be part of the application which it controls (e.g. a REST API exposed by the application itself)
   - they may be an adjacent Deployment running on the same Kubernetes cluster (e.g. a Python flask app)

### Risks and Mitigations

- A slow `downscalePodPicker` API may increase the time to downscale a ReplicaSet
   - MITIGATION: we will provide a `timeoutSeconds` config, if the API has not successfully returned within this time, 
     we proceed as if all Pods were "tied", and use the existing process to resolve the tie
- The `downscalePodPicker` may not be reachable at all times, or may return an invalid response
   - MITIGATION: we will provide a `maxRetries` config, if the API continues to fail after this many attempts, 
     we proceed as if all Pods were "tied", and use the existing process to resolve the tie
- Users may attempt to use `downscalePodPicker` and `controller.kubernetes.io/pod-deletion-cost` at the same time, resulting in unexpected behaviour
   - MITIGATION: documentation will stress that only ONE of `downscalePodPicker` and `controller.kubernetes.io/pod-deletion-cost` should be used per ReplicaSet
- A malicious (or poorly designed) `downscalePodPicker` API may attempt to overwhelm the controller by returning an extremely large payload
   - MITIGATION: we will limit the maximum response size to something reasonable, like 1MB
- Users may find it difficult to build their own `downscalePodPicker` API
   - MITIGATION: documentation will provide example implementations in common programing languages
- Users may create non-idempotent `downscalePodPicker` APIs that have side effects (e.g. preventing chosen workers from accepting new tasks):
   - MITIGATION: documentation will stress that `downscalePodPicker` API must be idempotent, and may be called multiple times
- Malicious actors may call `downscalePodPicker` APIs from outside the controller loop
   - MITIGATION: documentation will stress that `downscalePodPicker` APIs should be secured by authentication at the network or request layer


## Design Details

#### K8S API Changes:

A new `downscalePodPicker` field will be added to `ReplicaSetSpec` and `DeploymentSpec`.

Here is an example ReplicaSet with thew new `downscalePodPicker` field:

```yaml
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: my-replicaset
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
        - name: my-container
          image: my-image

  #############################
  ## this is the new section ##
  #############################
  downscalePodPicker:
    http:
      host: my-app.my-namespace.svc.cluster.local
      httpHeaders:
        - name: authentication
          valueFrom:
            secretKeyRef:
              key: authentication-token
              name: my-secret
      path: "/downscale-pod-picker"
      port: 443
      scheme: "https"
    maxRetries: 3
    timeoutSeconds: 5
```

The `downscalePodPicker` field has type `DownscalePodPicker` with the following specification:

| Field            | Type                   | Description                                                   |
|------------------|------------------------|---------------------------------------------------------------|
| `http`           | HTTPDownscalePodPicker | Configs to access an HTTP API                                 |
| `maxRetries`     | int                    | Maximum number of retries (default: `3`)                      |
| `timeoutSeconds` | int                    | Maximum number of seconds the API call can run (default: `1`) |

The `http` field has type `HTTPDownscalePodPicker` with the following specification:

| Field         | Type             | Description                                                                                    |
|---------------|------------------|------------------------------------------------------------------------------------------------|
| `host`        | string           | Host name to connect to.                                                                       |
| `httpHeaders` | HTTPHeader array | Custom headers to set in the request.                                                          |
| `path`        | string           | Path to access on the HTTP server.                                                             |
| `port`        | string / int     | Name or number of the port to access on the container. Number must be in the range 1 to 65535. |
| `scheme`      | string           | Scheme to use for connecting to the host: `HTTP`, `HTTPS`. (default: `"HTTP"`)                 |

The `httpHeaders` field has type [`HTTPHeader array`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#httpheader-v1-core)
with the following specification:

(NOTE: We will need to evaluate how feasible it is to extend `HTTPHeader v1 core` with `valueFrom` to enable mounting headers from Secrets,
for authentication. This may need to be its own independent KEP.)

| Field       | Type             | Description                                                            |
|-------------|------------------|------------------------------------------------------------------------|
| `name`      | string           | The header field name                                                  |
| `value`     | string           | The header field value                                                 |
| `valueFrom` | HTTPHeaderSource | Source for the header's value. Cannot be used if `value` is non-empty. |

The `valueFrom` field has type `HTTPHeaderSource` with the following specification:

| Field          | Type              | Description                                             |
|----------------|-------------------|---------------------------------------------------------|
| `secretKeyRef` | SecretKeySelector | Selects a key of a secret in the ReplicaSet's namespace |

The `secretKeyRef` field has type [`SecretKeySelector`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#secretkeyselector-v1-core):

| Field  | Type   | Description                                                       |
|--------|--------|-------------------------------------------------------------------|
| `key`  | string | The key of the secret to select from. Must be a valid secret key. |
| `name` | string | Name of the referent.                                             |
| `key`  | string | Specify whether the Secret or its key must be defined.            |


#### Pod-Picker API contract:

The contract for a `downscalePodPicker` API will be as follows:

- Request payloads to the API will contain:
   - `number_of_pods_requested` (`int`): minimum number of Pods to return
   - `candidate_pods` (`list[string]`): list of Pod-names to choose from
- Response payloads from the API will contain:
   - `chosen_pods` (`list[string]`): list of Pod-names chosen to be removed
   - `tied_pods` (`list[string]`): list of Pod-names we can't decide between
- Other requirements:
   - both `chosen_pods` and `tied_pods` can be non-empty
   - total number of pods returned should aim to be AT LEAST `number_of_pods_requested`
   - only Pod-names contained in `candidate_pods` may be returned
- NOTES:
  - The response payload is split into two lists, because when `number_of_pods_requested > 1`, 
    it is possible to have some Pods which were definitely chosen and others which we cannot decide between, 
     - EX: if the Pods have the following metrics `[1,2,2,2]` and `number_of_pods_requested = 2`, we might return `chosen_pods = [pod-1]`, `tied_pods = [pod-2, pod-3, pod-4]`
  - This contract doesn't require that `downscalePodPicker` APIs make a decision about ALL `candidate_pods`, and allows them
    to be designed such that they exit early from their search if they find enough good candidates before considering all Pods.
     - EX: if the API is looking for the least-active Pods, it can exit early if enough fully-idle Pods are found to meet `number_of_pods_requested`
     - EX: if the Pod-Picker returns no pods, the downscale will continue as if the Pod-Picker was not defined
  - The controller will exclude Pods in Unassigned/PodPending/PodUnknown/Unready states from `candidate_pods`.
     - this is so that the Pod-Picker doesn't have to check pods which will always be killed first, regardless of the Pod-Picker's decision
  - When implemented as a REST API, the payload would be passed as JSON in a POST request.
     - in the future, we can allow Pod-Pickers which use other protocols, like gRPC

#### Controller Behaviour Changes:

On downscale, the `ReplicaSet` controller assigns all Pods a "rank" based on the return lists from Pod-Picker API:

- Returned `chosen_pods` have `rank = 0`
- Returned `tied_pods` have `rank = 1`
- All other Pods have `rank = 3`

This "rank" is used in addition to the existing downscale pod-ordering behaviour, for example, 
`Unassigned`/`PodPending`/`PodUnknown`/`Unready` Pods will always be killed first.
(NOTE: If there are enough of them to fulfill the downscale, NO calls will be made to the Pod-Picker API)

The new downscale pod-ordering behaviour is:

1. Pods not yet assigned to Node
2. Pods in `Pending` phase
3. Pods in `Unknown` phase
4. Pods without `Ready` condition
5. Pods with lower `controller.kubernetes.io/pod-deletion-cost` annotation
6. IF a Pod-Picker is specified on the ReplicaSet:
    - Pods with lower Pod-Picker "rank" 
    - `chosen_pods = rank 0`
    - `tied_pods = rank 1`
    - `all other pods = rank 3`
7. ELSE:
    - Pods with lower number of Pods on the same Node in a `Running` phase
8. Pods with less time having `Ready` condition:
    - times are `log2` binned if `LogarithmicScaleDown` feature-gate is enabled
9. Pods with higher `restart count`
10. Pods with newer `creation time`

#### Controller Code Changes:

To make the ReplicaSet controller preform the required `downscalePodPicker` API requests, we will be making the following changes:

1. `getPodsToDelete()` found at [pkg/controller/replicaset/replica_set.go#L801](https://github.com/kubernetes/kubernetes/blob/876d4e0ab029ac7c314bb0e033bdd036531fe426/pkg/controller/replicaset/replica_set.go#L801):
    - We will change how the `podWithRanks` instance of type [ActivePodsWithRanks](https://github.com/kubernetes/kubernetes/blob/876d4e0ab029ac7c314bb0e033bdd036531fe426/pkg/controller/controller_utils.go#L785-L797) is created.
    - If `downscalePodPicker` is specified on the ReplicaSet:
       - initialise a `number_of_pods_requested` integer with a value of `diff`
       - construct a `candidate_pods` list by starting from `filteredPods`:
          - remove Unassigned/PodPending/PodUnknown/Unready Pods:
             - decrement `number_of_pods_requested` by 1
             - if `number_of_pods_requested` is 0:
                - set rank of all Pods to 0
                - RETURN
          - populate a mapping from "pod name" -> "filteredPods list-index"
       - send `number_of_pods_requested` and `candidate_pods` to the `downscalePodPicker` API:
          - if timeout reached:
             - set rank of all Pods to 0
             - RETURN
          - if transport error:
             - set rank of all Pods to 0
             - RETURN
          - if response malformed or too large:
             - set rank of all Pods to 0
             - RETURN
       - get `chosen_pods` and `tied_pods` from the API response:
          - set rank of Pods in `chosen_pods` to 0
          - set rank of Pods in `tied_pods` to 1
          - set rank of remaining Pods to 3
          - RETURN
    - Else:
       - Use existing `getPodsRankedByRelatedPodsOnSameNode()` function
2. `manageReplicas()` found at [pkg/controller/replicaset/replica_set.go#L599](https://github.com/kubernetes/kubernetes/blob/876d4e0ab029ac7c314bb0e033bdd036531fe426/pkg/controller/replicaset/replica_set.go#L599):
    - Calculating `relatedPods` will only be necessary if `downscalePodPicker` is not specified.


### Test Plan

- ((TODO))

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

### Graduation Criteria

- ((TODO))

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

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

- ((TODO))

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

- ((TODO))

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

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DownscalePodPicker`
  - Components depending on the feature gate: `kube-controller-manager`

###### Does enabling the feature change any default behavior?

- If `downscalePodPicker` is specified on a ReplicaSet, then the current `"Pods on nodes with more
  replicas come before pods on nodes with fewer replicas"` behaviour is disabled.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

- Yes

###### What happens if we reenable the feature if it was previously rolled back?

- It should work as expected.

###### Are there any tests for feature enablement/disablement?

- ((TODO))

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->


### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

- ((TODO))

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

- ((TODO))

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

- ((TODO))

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

- ((TODO))

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

- ((TODO))

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

- ((TODO))

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

- ((TODO))

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

- ((TODO))

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

- ((TODO))

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

- ((TODO))

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

- ((TODO))

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

- ((TODO))

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- ((TODO))

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

- ((TODO))

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

- ((TODO))

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

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

- ((TODO))

###### What are other known failure modes?

- ((TODO))

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

- ((TODO))

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

- ((TODO))

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### ALTERNATIVE 1: Pod-Deletion-Cost annotation

Same as the current [`controller.kubernetes.io/pod-deletion-cost`](https://kubernetes.io/docs/concepts/workloads/controllers/replicaset/#pod-deletion-cost) annotation.

__USAGE PATTERN 1 - the annotation kept up-to-date with a controller:__

- PROBLEM 1: every update requires a PATCH call to the kube-apiserver (probably overloading it)
- PROBLEM 2: this is wasteful for deployments that rarely scale down
   - calculating and updating the cost could be expensive, and that work is wasted if not actually used to downscale

__USAGE PATTERN 2 - the annotation is updated only when downscaling:__

- PROBLEM 1: existing scaling tools like HorozontalPodAutoscaler can't be used
   - the annotation/status would need to be updated BEFORE downscaling, and we can't predict when HorozontalPodAutoscaler will downscale
   - therefore, users must write their own scalers that update the annotation/status before downscaling (making it inaccessible for most users)
- PROBLEM 2: after each downscale, any annotations added by the scaler will need to be cleared (they will become out of date)
   - related to this, we must wait to clear them before we can start the next downscale (or the old annotations will impact the new scale)
- PROBLEM 3: costs may be out-of-date by the time the controller picks which pod to downscale
   - some apps like airflow/spark may have quite rapidly changing "best" pods to kill

### ALTERNATIVE 2: Pod-Deletion-Cost http/exec probe

A new http/exec probe is created for Pods which returns their current pod-deletion-cost.

__USAGE PATTERN 1 - probes are used every X seconds to update a Pod status field:__

- (Suffers from the same problems as "USAGE PATTERN 1" of "ALTERNATIVE 1")

__USAGE PATTERN 2 - probes are ONLY used when downscaling:__

- PROBLEM 1: the controller cannot make a probe request to Pods (this must be done by the Node's kubelet)
  - to solve this you would need a complex system that has the controller "mark" the pods for probing by the kubelet
- PROBLEM 2: to find the "best" Pod to kill, we would need to probe every single Pod, this is not scalable
  - to solve this, you would need to use a heuristic approach, e.g. only checking a sample of Pods and returning the lowest cost from the sample

### ALTERNATIVE 3: Pod-Deletion-Cost API

A central user-managed API is queried by the controller and returns the pod-deletion-cost of each Pod.
(Rather than returning a list of `chosen_pods` and `tied_pods` like in this KEP)

- PROBLEM 1: the user-managed API can't exit-early
  - because a pod-deletion-cost must be returned for each Pod, the API can't have the concept of a "free" Pod, which if found can be immediately returned without checking the other Pods
- PROBLEM 2: the user-managed API may calculate the pod-deletion-cost of more Pods than necessary
  - the API is unaware of how many Pods are actually planned to be removed, so will calculate the pod-deletion-cost of more Pods than is necessary

## Infrastructure Needed (Optional)

- ((TODO))

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
