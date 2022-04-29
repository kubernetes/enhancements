# KEP-3285: Subresource finalizers

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Supported API calls to the endpoint](#supported-api-calls-to-the-endpoint)
    - [GET](#get)
    - [PUT](#put)
  - [Implementation](#implementation)
  - [CLI support](#cli-support)
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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes a way to set different RBACs to `finalizers` field and its entire resource.

## Motivation

`Finalizers` is used by controllers as a pre-deletion hook.
There is no subresource for `finalizers`, so the only way to grant an actor access to them is to grant access to the whole object.
On the other hand, some controllers only need to update `finalizers` field and some of its subresources, and don't need to update its entire resource.
To implement least privilege for such controllers, there should be a way to only allow updating `finalizers`.

### Goals

- Provide a way to set a different RBAC to `finalizers` from its entire resource
- Provide a way to audit and update only `metadata.finalizers` field

### Non-Goals

- Provide a way to set a different RBAC to other than `finalizers` from its entire resource
- Provide a way to allow updating its entire resource  but disallow updating `finalizers` field

## Proposal

A new REST endpoint `<resource>/finalizers` is added automatically to all resources as an alternative way to access the `metadata.finalizers` fielld, which allows separate RBAC.
The existing method to acceess `metadata.finalizers` field (through the main `<resource>` endpoint) remains unchanged.

### User Stories (Optional)

#### Story 1

There is a controller that only updates `status` field and `finalizers` field of a CRD. Administrators of the controller can allow the controller to update only these particular fields.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

- Name collision of the endpoints is needed to be considered.
- Other existing fields may require the same alternative way to access.
For example, [candidates](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go) in metadata fields are:
  - `labels`
  - `annotations`
  - `ownerReferences`
  - `status/conditions`

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

Risks:
- Bugs could make unintended fields accessible.

Mitigations:
- The new endpoint needs to be negative-tested to prove that the other fields are NOT modifiable.

## Design Details

Add `<resource>/finalizers` endpoint to all resources and allow access to `metadata.finalizers` for the resources through the endpoints.
The actual field in each resource isn't added, but the request to the endpoint is handled as the operation to `metadata.finalizers`.
`Role` and `ClusterRole` can be set to the endpoint, therefore users can set a separate RBAC to `<resource>/finalizers`.
A PoC implementation can be found [here](https://github.com/mkimuram/kubernetes/commit/751a67e03d62397c1d21947e9e2e3a81f279a29f) (Please also see [here](https://github.com/kubernetes/enhancements/pull/2840#discussion_r723464012) for how it works. Note that API definition in this KEP will differ from the one in the PoC).

```
<<[UNRESOLVED Endpoint name]>>
Endpoint name for the subresource is still being discussed.
- `/finalize`: already used by namespace
- `/finalizers`: some debate about whether sub-resources should be nouns of verbs
<<[/UNRESOLVED]>>
```

### API Changes

A new `Metadata` resource is introduced in `subresources.k8s.io` API group:

```golang
// Metadata represents a metadata field to be accessed via an endpoint for the subresource.
type Metadata struct {
	metav1.TypeMeta `json:",inline"`

	// Only following fields should be set:
	// - Namespace
	// - Name
	// - UID
	// - ResourceVersion
	// - The field for the endpoint, like Finalizers field for the finalizers endpoint
	// If other fields are set, they are ignored.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
}
```

```
<<[UNRESOLVED Resource name]>>
Resource name is still being discussed.
`Metadata` seems good if the same type can be used for all of the metadata subresources.
If resource should be defined for each metadata subresource, `Finalize` or `Finalizers` will be candidates.
<<[/UNRESOLVED]>>
```

### Supported API calls to the endpoint

The `<resource>/finalizers` endpoint supports 2 verbs, GET and PUT.

```
<<[UNRESOLVED Whether to implement PATCH]>>
There is no good way to remove a specific item from a list by using existing PATCH request.
Server-side apply is trying to resolve this, but it can't still be used for this purpose, currently.
Therefore, whether to implement PATCH is decided later.
<<[/UNRESOLVED]>>
```

#### GET

GET verb is used to read the value in `metadata.finalizers` field.

- Request format:
    Empty string should be passed as a request body. Any strings passed in the request body are ignored.

- Response format:
  - On success, the `Metadata` resource is returned. `Namespace`, `Name`, `UID`, `ResourceVersion`, and `Finalizers` fields are copied from the original resources.

         ex)
         ```json
         {
             "kind": "Metadata",
             "apiVersion": "v1alpha1",
             "metadata": {
                 "namespace": "default",
                 "name": "my-config",
                 "uid": "1af42023-9303-4dd3-9383-f117fbd25296",
                 "resourceVersion": "486",
                 "finalizers": ["example.com/my-finalizer"],
             }
         }
         ```

  - On failure, error message is stored in `Status` resource.

        ex)
        ```json
        {
            "kind": "Status",
            "apiVersion": "v1",
            "metadata": {},
            "status": "Failure",
            "message": "\"finalizers\" is forbidden: User \"example-user\" cannot get subresource \"finalizers\" of \"ConfigMap\" in API group \"\" in the namespace \"default\"",
            "reason": "Forbidden",
            "details": {},
            "code": 403
        }
        ```

- Status code:
  - 200 "OK": Returned on success without change
  - 403 "Forbidden": Returned when GET verb is not allowed for `<resource>/finalizers` endpoint (RBAC admission handles this)

#### PUT

PUT verb is used to set the value in `metadata.finalizers` field.

- Request format:
    `Metadata` resource is used for PUT request. `Namespace`, `Name`, `UID`, and `ResourceVersion` fields need to be copied from the original resource (or `Metadata` resource returned from the GET method) and `Finalizers` field should be set to the intended value.

    ex)
    ```json
    {
        "kind": "Metadata",
        "apiVersion": "v1alpha1",
        "metadata": {
            "namespace": "default",
            "name": "my-config",
            "uid": "1af42023-9303-4dd3-9383-f117fbd25296",
            "resourceVersion": "486",
            "finalizers": ["example.com/my-new-finalizer"],
        }
    }
    ```

- Response format:
  - On success, the `Metadata` resource is returned (the same format as the one for GET).
  - On failure, error message is stored in `Status` resource (the same format as the one for GET)

- Status code:
  - 200 "OK": Returned on success without change
  - 201 "Created": Returned on success with change
  - 401 "Unauthorized": Returned on failure in setting the value
  - 403 "Forbidden": Returned when PUT verb is not allowed for `<resource>/finalizers` endpoint (RBAC admission handles this)
  - 409 "Conflict": Returned when `ResourceVersion` conflicts

### Implementation

This kind of subresource has already been added as [`scale`](https://github.com/kubernetes/kubernetes/pull/21966/commits/9e99f9fa0e45a8f56c399f9cbb8047a4f39dde4e) to some of the resources, like `Replicaset` and `Deployment`.
Therefore, similar implementation can be done by adding endpoint for `finalizers` to all the resources.

### CLI support

[KEP-2590: Kubectl Subresource Support](https://github.com/kubernetes/enhancements/tree/master/keps/sig-cli/2590-kubectl-subresource) provides a way to access subresource by `kubectl patch --subresource=` command.
Currently, it only supports `status` and `scale` subresources and it only supports GET, PATCH, EDIT and REPLACE verbs.
Therefore, it needs to be extended to support GET and PUT verbs for `finalizers` subresource.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

- [x] I/we understand the owners of the involved components may require updates to
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

API:
- `pkg/registry/core/rest` : `2022/6/20` - 6.0%
- `pkg/registry/apps/daemonset/storage` : `2022/6/20` - 58.8
- `pkg/registry/apps/deployment/storage` : `2022/6/20` - 67.4%
- `pkg/registry/apps/replicaset/storage` : `2022/6/20` - 63.4%
- `pkg/registry/apps/statefulset/storage` : `2022/6/20` - 64.0%
- `pkg/registry/autoscaling/horizontalpodautoscaler/storage` : `2022/6/20` - 70.6%
- `pkg/registry/batch/cronjob/storage` : `2022/6/20` - 52.9%
- `pkg/registry/batch/job/storage` : `2022/6/20` - 73.1%
- `pkg/registry/certificates/certificates/storage` : `2022/6/20` - 63.1%
- `pkg/registry/coordination/lease/storage` : `2022/6/20` -
- `pkg/registry/core/configmap/storage` : `2022/6/20` - 87.5%
- `pkg/registry/core/endpoint/storage` : `2022/6/20` - 75.0%
- `pkg/registry/core/event/storage` : `2022/6/20` - 77.8%
- `pkg/registry/core/limitrange/storage` : `2022/6/20` - 87.5%
- `pkg/registry/core/namespace/storage` : `2022/6/20` - 69.4%
- `pkg/registry/core/node/storage` : `2022/6/20` - 48.6%
- `pkg/registry/core/persistentvolume/storage` : `2022/6/20` - 68.8%
- `pkg/registry/core/persistentvolumeclaim/storage` : `2022/6/20` - 75.0%
- `pkg/registry/core/pod/storage` : `2022/6/20` - 75.7%
- `pkg/registry/core/podtemplate/storage` : `2022/6/20` - 85.7%
- `pkg/registry/core/replicationcontroller/storage` : `2022/6/20` - 65.8%
- `pkg/registry/core/resourcequota/storage` : `2022/6/20` - 68.8%
- `pkg/registry/core/secret/storage` : `2022/6/20` - 85.7%
- `pkg/registry/core/service/allocator/storage` : `2022/6/20` - 35.6%
- `pkg/registry/core/service/ipallocator/storage` : `2022/6/20` -
- `pkg/registry/core/service/portallocator/storage` : `2022/6/20` -
- `pkg/registry/core/service/storage` : `2022/6/20` - 90.7%
- `pkg/registry/core/serviceaccount/storage` : `2022/6/20` - 10.7%
- `pkg/registry/discovery/endpointslice/storage` : `2022/6/20` -
- `pkg/registry/flowcontrol/flowschema/storage` : `2022/6/20` -
- `pkg/registry/flowcontrol/prioritylevelconfiguration/storage` : `2022/6/20` -
- `pkg/registry/networking/ingress/storage` : `2022/6/20` - 62.5%
- `pkg/registry/networking/ingressclass/storage` : `2022/6/20` -
- `pkg/registry/networking/networkpolicy/storage` : `2022/6/20` - 73.3%
- `pkg/registry/node/runtimeclass/storage` : `2022/6/20` -
- `pkg/registry/policy/poddisruptionbudget/storage` : `2022/6/20` - 62.5%
- `pkg/registry/policy/podsecuritypolicy/storage` : `2022/6/20` - 75.0%
- `pkg/registry/rbac/clusterrole/storage` : `2022/6/20` -
- `pkg/registry/rbac/clusterrolebinding/storage` : `2022/6/20` -
- `pkg/registry/rbac/role/storage` : `2022/6/20` -
- `pkg/registry/rbac/rolebinding/storage` : `2022/6/20` -
- `pkg/registry/scheduling/priorityclass/storage` : `2022/6/20` - 83.3%
- `pkg/registry/storage/csidriver/storage` : `2022/6/20` - 85.7%
- `pkg/registry/storage/csinode/storage` : `2022/6/20` - 85.7%
- `pkg/registry/storage/csistoragecapacity/storage` : `2022/6/20` - 85.7%
- `pkg/registry/storage/storageclass/storage` : `2022/6/20` - 87.5%
- `pkg/registry/storage/volumeattachment/storage` : `2022/6/20` - 66.7%
- `staging/src/k8s.io/apiextensions-apiserver/pkg/registry/customresource`: 2022/6/20` - 72.6%
- `staging/src/k8s.io/apiextensions-apiserver/pkg/registry/customresourcedefinition`: 2022/6/20` - 13.1%

CLI:
- `staging/src/k8s.io/kubectl/pkg/cmd/get`: 2022/6/20` - 81.0%
- `staging/src/k8s.io/kubectl/pkg/cmd/patch`: 2022/6/20` - 57.1%

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- TestFinalizeAllResources (similar tests to those of [scale](https://github.com/kubernetes/kubernetes/blob/master/test/integration/apiserver/apply/scale_test.go)): <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

GET:
- Verify finalizers can be read through GET call to finalizers if getting finalizers is allowed:  <link to test coverage>
    (make sure UID and ResourceVersion is properly set in the response).
- Verify finalizers can't be read through GET call to finalizers if getting finalizers isn't allowed:  <link to test coverage>
- Verify resource can't be read through GET call to the resource if only getting finalizers is allowed:  <link to test coverage>

PUT:
- Verify finalizers can be updated through PUT call to finalizers if updating finalizers is allowed:  <link to test coverage>
- Verify finalizers can't be updated through PUT call to finalizers if updating finalizers isn't allowed:  <link to test coverage>
- Verify finalizers can't be updated through PUT call to finalizers if UID is invalid:  <link to test coverage>
- Verify finalizers can't be updated through PUT call to finalizers if ResourceVersion is invalid:  <link to test coverage>
- Verify resource can't be updated through PUT call to the resource if only updating finalizers is allowed:  <link to test coverage>

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Additional tests are in Testgrid and linked in KEP

#### GA

- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

### Upgrade / Downgrade Strategy

- Upgrade:
  - Method: Enable the FinalizeSubresource feature gate
  - Behavior: `metadata.finalizers` field can be accessed through `<resource>/finalizers` API endpoint
- Downgrade:
  - Method: Disable the FinalizeSubresource feature gate
  - Behavior: `metadata.finalizers` field can't be accessed through `<resource>/finalizers` API endpoint

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
  - Feature gate name: FinalizeSubresource
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

No.
Existing behaviors remain unchanged. `metadata.finalizers` field can be accessed through `<resource>/finalizers` API endpoint, if it is allowed by RBAC.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by disabling the feature gates.

###### What happens if we reenable the feature if it was previously rolled back?

`metadata.finalizers` field can be accessed through `<resource>/finalizers` API endpoint, if it is allowed by RBAC , again.

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

No. Tests will be added as described in the Test Plan section.

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
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

Existing apiserver metrics, like apiserver_request_total, can be used to determine if the feature is in use.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [x] API .status
  - Other field: By accessing `<resource>/finalizers` API endpoint, and check if there are any responses.

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

API calls to `<resource>` may be replaced with API calls to `<resource>/finalizers`.
In most cases, number of API calls won't be changed, but in some cases, one existing API call
may be separated into two API calls (one for a `subresource` and the other for `finalizers`).

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

The change in number of the API calls described above might affect.
However, if it matters, users can choose not to use this feature, by using the existing API calls.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

This feature just adds additional endpoints, therefore it behaves the same way as the existing endpoints.

###### What are other known failure modes?

N/A

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

N/A

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

- Allow access to entire resource, which breaks the principle of least privilege
- Implement v2 API for all resources, which aren't backward compatible

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
