# KEP-4753: Expose `ownerReferences` via `valueFrom` and  downward API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Implementation Details](#implementation-details)
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

Today when a pod wants to pass its `ownerReferences` to a new object it manage (a ConfigMap, a Secret, etc), it needs to call the API server to GET it's own ownerReferences and then pass it to the new object. 
The pod then needs GET access on all the intermediate resources which it may not have, and giving it that access is a security risk as it can access other resources it should not have access to.
This KEP proposes to expose the ownerReferences via `valueFrom` and downward API, so that the pod can directly pass its `ownerReferences` to the managed/created new object without having to call the API server.

This KEP is intentionally scoped small, and intends to keep the scope just to `ownerReferences` and not to other fields of the owner object to prevent security risks.

## Motivation

DaemonSets can dynamically orphan and adopt pods. Consider a scenario where, on day 1, a pod named foobar-deadbeef93-xyz76 on node XYZ creates an object (e.g a ConfigMap) associated with node XYZ, then  it would be incorrect to set the object owner to the pod foobar-deadbeef93-xyz76 instead, it should be set to damonset foobar, to prevent the object from being garbage collected if the pod cease to exist. This issue arises when pods create resources like ConfigMaps or CustomResources and these resources are not correctly reassigned to the new pod, leading to orphaned objects. Ensuring that daemonset pods can pass their ownerReferences to new objects they manage would prevent this issue, as the ownership hierarchy would be preserved even when pods are recreated.

For example, a CustomResource created by a pod managed by a DaemonSet may be unexpectedly garbage collected if the pod is deleted. This can disrupt system behavior, as the CustomResource is garbageCollected. By allowing the pod to inherit and pass down ownerReferences, the CustomResource would remain correctly managed by the DaemonSet(the owner), avoiding disruptions.

### Goals

* Gain access to pod `ownerReferences` on pods through volume mounts.
* Gain access to pod `ownerReferences` on pods through environmental variables.

### Non-Goals

- Not to expose additional pod info outside of `metadata.ownerReferences`.

## Proposal

The initial design includes:

* Being able to pass pod `ownerReferences` to volumes with fieldPath of `metadata.ownerReferences`.
* Being able to pass pod `ownerReferences` to environmental variables with fieldPath of `metadata.ownerReferences`.

### User Stories (Optional)

#### Story 1
xref https://github.com/kubernetes-sigs/node-feature-discovery/issues/1752

CustomResource object created by a pod owned by a DaemonSet is garbage collected when the pod is deleted by unexpected reasons. This triggers unwanted behavior in the system.
If the CustomResource object inherits the ownerReferences from the pod, it will be garbage collected only when the DaemonSet is deleted. This will add extra reliability to the system by projecting the ownerReferences from the Deployment/DaemonSet/StatefulSet to the CustomResource object.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

- Security Risk: Some environments may not want to expose the ownerReferences of a pod to the pod itself.
  - Mitigation: Install an admission check (webhook or Validating Admission Policy) to reject pods that try to use Downward API to access ownerReferences.

## Design Details

The intention behind this design is not to add extra structs or fields
but to extend the existing `downward API` to include the `metadata.ownerReferences` field.

### Implementation Details

* Will use pre-existing information from the Kubelet and VolumeHost structs
to extract pod metadata information. 

At `/pkg/apis/core/validation/validation.go` we will edit the following:
```go
var validVolumeDownwardAPIFieldPathExpressions = sets.New(
	"metadata.name",
	"metadata.namespace",
	"metadata.labels",
	"metadata.annotations",
	"metadata.uid",
  "metadata.ownerReferences")
```

At `pkg/apis/core/pods/helpers.go` we will edit the following function:

```go
func ConvertDownwardAPIFieldLabel(version, label, value string) (string, string, error) {
	if version != "v1" {
		return "", "", fmt.Errorf("unsupported pod version: %s", version)
	}

	if path, _, ok := fieldpath.SplitMaybeSubscriptedPath(label); ok {
		switch path {
		case "metadata.annotations", "metadata.labels":
			return label, value, nil
		default:
			return "", "", fmt.Errorf("field label does not support subscript: %s", label)
		}
	}

	switch label {
	case "metadata.annotations",
		"metadata.labels",
		"metadata.name",
		"metadata.namespace",
		"metadata.uid",
    "metadata.ownerReferences",
		"spec.nodeName",
		"spec.restartPolicy",
		"spec.serviceAccountName",
		"spec.schedulerName",
		"status.phase",
		"status.hostIP",
		"status.hostIPs",
		"status.podIP",
		"status.podIPs":
		return label, value, nil
	// This is for backwards compatibility with old v1 clients which send spec.host
	case "spec.host":
		return "spec.nodeName", value, nil
	default:
		return "", "", fmt.Errorf("field label not supported: %s", label)
	}
}
```
* Information will be passed to the pod via the `valueFrom` field in the volume.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kubernetes-downwardapi-volume-example
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: DaemonSet
    name: an-owned-pod-1722852739
    uid: 6be1683f-da9c-4f68-9440-82376231cfa6
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: Some-CRD
    name: an-owned-pod-1722852739
    uid: 6be1683f-da9c-4f68-9440-82376231cfa6
spec:
  containers:
    - name: client-container
      image: busybox
      command: ["sh", "-c"]
      args:
      - while true; do
          if [[ -e /etc/podinfo/ownerReferences ]]; then
            echo -en '\n\n'; cat /etc/podinfo/ownerReferences; fi;
          sleep 5;
        done;
      volumeMounts:
        - name: podinfo
          mountPath: /etc/podinfo
  volumes:
    - name: podinfo
      downwardAPI:
        items:
          - path: "ownerReferences"
            fieldRef:
              fieldPath: metadata.ownerReferences
```

Logs from the pod will show the ownerReferences of the pod:

The file representation of this fieldref is a JSON list of objects, each of which is a serialized `k8s.io/apimachinery/pkg/apis/meta/v1.OwnerReference`.

```bash
$ kubectl logs kubernetes-downwardapi-volume-example
{
    "kind": "OwnerReference",
    "apiVersion": "meta/v1",
    "items": [
        { .... }, { ... }
    ]
}
```

If `ownerReferences` is empty or not set, the file will be empty.

* Information will be passed to the pod via the `valueFrom` field in the environmental variables.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kubernetes-downwardapi-env-example
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: DaemonSet
    name: an-owned-pod-1722852739
    uid: 6be1683f-da9c-4f68-9440-82376231cfa6
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: Some-CRD
    name: an-owned-pod-1722852739
    uid: 6be1683f-da9c-4f68-9440-82376231cfa6
spec:
  containers:
    - name: client-container
      image: busybox
      command: ["sh", "-c"]
      args:
      - while true; do
          if [[ -n "${OWNER_REFERENCES}" ]]; then
            echo -en '\n\n'; echo $OWNER_REFERENCES; fi;
          sleep 5;
        done;
      env:
        - name: OWNER_REFERENCES
          valueFrom:
            fieldRef:
              fieldPath: metadata.ownerReferences
```

Logs from the pod will show the ownerReferences of the pod:

The environmental variable representation of this fieldref is a JSON list of objects, each of which is a serialized `k8s.io/apimachinery/pkg/apis/meta/v1.OwnerReference`.

```bash
$ kubectl logs kubernetes-downwardapi-env-example
{"kind": "OwnerReference", "apiVersion": "meta/v1","items": [{ .... }, { ... }]}
```

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

Unit and e2e testing will be added consistent with other resources in downward API.

##### Integration tests

There are no integration tests, only e2e tests.

##### e2e tests

e2e testing will consist of the following:
- Create a DaemonSet
- Verify if the pod created by the DaemonSet has the ownerReferences of the DaemonSet via VolumeMounts
- Verify the file mounted in the pod contains the ownerReferences of the DaemonSet

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

There is no issue with the version skew.

## Production Readiness Review Questionnaire


### Feature Enablement and Rollback

Simple change of a feature gate will either enable or disable this feature.

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `DownwardAPIOwnerReferences`
  - Components depending on the feature gate: `kubelet`

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

No, the missing Downward API field will be perceived as an unknown field.

###### What happens if we reenable the feature if it was previously rolled back?

Pods will gain access to the `ownerReferences` field again vioa the Downward API.

###### Are there any tests for feature enablement/disablement?

Yes, see in e2e tests section.

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

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

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

Not that we can think of.

## Alternatives

Today pods must use the API server to get their ownerReferences and pass them to the
new object they manage. This is a security risk as the pod needs GET access to all the intermediate resources.
And creates unnecessary load on the API server.

## Infrastructure Needed (Optional)

No
