# KEP-4828: Component Flagz

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
  - [User Stories](#user-stories)
    - [User Story 1: Debugging Component Behavior](#user-story-1-debugging-component-behavior)
    - [User Story 2:  Verifying Flag Changes](#user-story-2--verifying-flag-changes)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Data Format and versioning](#data-format-and-versioning)
  - [Authz and authn](#authz-and-authn)
  - [Endpoint Response Format](#endpoint-response-format)
    - [Data format: text](#data-format-text)
      - [Request](#request)
      - [Response fields](#response-fields)
      - [Sample response](#sample-response)
    - [Structured API format (v1alpha1)](#structured-api-format-v1alpha1)
      - [Request](#request-1)
      - [Response Body: <code>Flagz</code> object](#response-body-flagz-object)
      - [Sample Response (JSON)](#sample-response-json)
  - [API Versioning and Deprecation Policy](#api-versioning-and-deprecation-policy)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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

We propose to enhance the observability, troubleshooting, and debugging capabilities of core Kubernetes components by implementing a flagz endpoint for all components. This will provide users with real-time visibility into the command-line flags each component was started with, aiding in the diagnosis of configuration issues and unexpected behavior

## Motivation

This proposal extends the ideas presented in the [Component Statusz KEP](https://github.com/kubernetes/enhancements/pull/4830), however, it proposes a distinct flagz endpoint to enhance visibility into component configurations. By providing real-time visibility into the active configuration of Kubernetes components, flagz enables users to quickly spot misconfigurations that could lead to system instability or outages

### Goals

* Provide users with the ability to dynamically inspect and understand the active flags of running Kubernetes components

### Non-Goals

* Replace existing monitoring solutions: metrics, logs, traces
* Provide information about components that are not directly accessible due to network restrictions

## Proposal

We are proposing to add a new endpoint, /flagz on all core Kubernetes components which returns the command-line flags each component was started with.

### User Stories

#### User Story 1: Debugging Component Behavior

- As a cluster administrator, I want to be able to query the /flagz endpoint on individual Kubernetes components so I can inspect their runtime flag values and diagnose unexpected behavior

#### User Story 2:  Verifying Flag Changes

- As a developer, I want to access the /flagz endpoint after deploying a component with modified flags so I can verify that my changes have been applied correctly

### Risks and Mitigations

1. **No sensitive data exposed**
    
    We will ensure that no sensitive data is exposed through flagz and that access to this endpoint is gated by using system-monitoring group.

2. **Performance impact**
    
    We will take care to carefully consider the computational overhead associated with the flagz endpoint to avoid negatively impacting the server's overall performance.

3. **Essential data only**
    
    We will only display the command-line flag values for a component in this page and nothing more.

4. **Premature Dependency on Unstable Format**

    The alpha release will explicitly support plain text format only, making it clear that the output is not intended for parsing or automated monitoring. The feature will be secured behind a feature gate that is disabled by default, ensuring users opt-in consciously. will support specifying a version as a query parameter to facilitate future transitions to structured schema changes without breaking existing integrations, once the usage patterns and requirements are better understood.

## Design Details

### Data Format and versioning

The `/flagz` endpoint defaults to a `text/plain` response format, which is intended for human consumption and should not be parsed by automated tools.

For programmatic access, a structured, versioned API format is available (supporting JSON, YAML, and CBOR) with an initial alpha version of `v1alpha1`. This version is not stable and is intended for feedback and iteration during the Alpha phase.

- **Group**: `config.k8s.io`
- **Version**: `v1alpha1` (initial version, subject to change)
- **Kind**: `Flagz`

To receive the structured JSON response, a client **must** explicitly request it using an `Accept` header specifying these parameters. For example:

```
GET /flagz
Accept: application/json;as=Flagz;v=v1alpha1;g=config.k8s.io
```

This negotiation mechanism ensures clients are explicit about the exact API they want, preventing accidental dependencies on unstable or incorrect formats. If a client requests `application/json` without the required parameters, the server will respond with a `406 Not Acceptable` error.

### Authz and authn

Access to the endpoint will be limited to members of the `system:monitoring group` ([ref](https://github.com/kubernetes/kubernetes/blob/release-1.31/staging/src/k8s.io/apiserver/pkg/authentication/user/user.go#L73)), which is how paths like /healthz, /livez, /readyz are restricted today. 

### Endpoint Response Format

The `/flagz` endpoint supports both plain text and structured API (JSON/YAML/CBOR) formats. The default format is `text/plain`.

#### Data format: text

This format is the default and is intended for human readability.

##### Request
* Method: **GET** 
* Endpoint: **/flagz**
* Header: `Accept: text/plain` (or omitted)
* Body: empty

##### Response fields
* **Flag Name**: The name of the command-line flag or configuration option
* **Value**: The current value assigned to the flag at runtime

##### Sample response

```
----------------------------
title: Kubernetes Flagz
description: Command line flags that Kubernetes component was started with.
----------------------------

default-watch-cache-size=100
delete-collection-workers=1
enable-garbage-collector=true
encryption-provider-config-automatic-reload=false
...
```


#### Structured API format (v1alpha1)

This format is available in Alpha for programmatic access and must be explicitly requested. It is considered an alpha-level format. The structured API supports JSON, YAML, and (if the CBORServingAndStorage feature gate is enabled) CBOR serialization.


##### Request
* Method: **GET** 
* Endpoint: **/flagz**
* Header: `Accept: application/json;as=Flagz;v=v1alpha1;g=config.k8s.io` (for JSON)
  or `Accept: application/yaml;as=Flagz;v=v1alpha1;g=config.k8s.io` (for YAML)
  or `Accept: application/cbor;as=Flagz;v=v1alpha1;g=config.k8s.io` (for CBOR, if enabled)
* Body: empty


##### Response Body: `Flagz` object

The response is a `Flagz` object, serialized in the requested format (JSON, YAML, or CBOR).

###### Go Struct Definition
```go
// Flagz is the structured response for the /flagz endpoint.
type Flagz struct {
	// Kind is "Flagz".
	Kind string `json:"kind"`
	// APIVersion is the version of the object, e.g., "config.k8s.io/v1alpha1".
	APIVersion string `json:"apiVersion"`

	// Standard object's metadata.
	// +optional
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`

	// Flags contains the command-line flags and their values.
	// The keys are the flag names and the values are the flag values,
	// possibly with confidential values redacted.
	// +optional
	Flags map[string]string `json:"flags,omitempty"`
}
```


###### Example Structure (JSON)
```json
{
  "kind": "Flagz",
  "apiVersion": "config.k8s.io/v1alpha1",
  "metadata": {
    "name": "<component-name>"
  },
  "flags": {
    "<flag-name>": "<flag-value>",
    ...
  }
}
```
YAML and CBOR follow the same structure, serialized in their respective formats.


##### Sample Response (JSON)

```json
{
  "kind": "Flagz",
  "apiVersion": "config.k8s.io/v1alpha1",
  "metadata": {
    "name": "apiserver"
  },
  "flags": {
    "default-watch-cache-size": "100",
    "delete-collection-workers": "1",
    "enable-garbage-collector": "true",
    "encryption-provider-config-automatic-reload": "false"
  }
}
```
YAML and CBOR responses are equivalent in structure.
### API Versioning and Deprecation Policy

The versioned `/flagz` endpoint will follow the standard [Kubernetes API deprecation policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/).

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `staging/src/k8s.io/component-base/zpages/flagz`: Unit tests have been added to cover both the plain text and structured output, including serialization and content negotiation logic.

##### Integration tests

- Integration tests have been added for each component to verify that the `/flagz` endpoint is correctly registered and serves both `text/plain` and the versioned `application/json`, `application/yaml`, `application/cbor` content types.

##### e2e tests

- E2E tests have been added to query the `/flagz` endpoint for each core component and validate the following:
  - The endpoint is reachable and returns a `200 OK` status.
  - Requesting with the `Accept` header for `application/json;as=Flagz;v=v1alpha1;g=config.k8s.io` returns a valid `Flagz` JSON object.
  - The JSON response can be successfully unmarshalled.
  - Requesting with `text/plain` or no `Accept` header returns the non-empty plain text format.

### Graduation Criteria

#### Alpha

- Feature gate `ComponentFlagz` is disabled by default.
- A structured JSON response (`config.k8s.io/v1alpha1`) is introduced for feedback, alongside the default `text/plain` format.
- Feature is implemented for at least one component (e.g., kube-apiserver).
- E2E tests are added for both plain text and the new structured response format.
- Gather feedback from users and developers on the structured format.

#### Beta

- Feature gate `ComponentFlagz` is enabled by default.
- The structured API is promoted to `v1beta1` or `v1` based on feedback and is considered stable.
- Feature is implemented for all core Kubernetes components.

#### GA

- The structured API is promoted to a stable `v1` version after bake-in.
- Conformance tests are in place for the endpoint.

#### Deprecation

### Upgrade / Downgrade Strategy

In alpha, no changes are required to maintain previous behavior. And the feature gate can be turned on to make use of the enhancement.

### Version Skew Strategy

The primary purpose of the flagz page is to provide information about the individual component itself. While the underlying data format may be enhanced over time, the endpoint itself will remain a reliable source of flag information across Kubernetes versions.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

We will target this feature behind a flag `ComponentFlagz`

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ComponentFlagz
  - Components depending on the feature gate:
    - apiserver
    - kubelet
    - scheduler
    - controller-manager
    - kube-proxy

###### Does enabling the feature change any default behavior?

Yes it will expose a new flagz endpoint. 

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. But it will remove the flagz endpoint

###### What happens if we reenable the feature if it was previously rolled back?

It will expose the flagz endpoint again

###### Are there any tests for feature enablement/disablement?

Unit test will be introduced in alpha implementation.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

This feature should not cause rollout failures. If it does, we can disable the feature. In the worst
case, it is possible it could cause runtime failures, but it is highly unlikely we would not detect this
with existing tests. The endpoint is intended to provide enhanced observability into component state.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

- apiserver_request_duration_seconds metric that will tell us if there's a spike in request latency of kube-apiserver that might indicate that the flagz endpoint is interfering with the component's core functionality or causing delays in processing other requests

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

This should not be necessary since we're adding a new endpoint with no dependencies. The rollback
simply removes the endpoint.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

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

They can curl a component's `flagz` endpoint. However, its important to note that this feature is not directly used by workloads, its a debug feature.

###### How can someone using this feature know that it is working for their instance?

- They can curl a component's `flagz` endpoint, which should return data.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This is a debugging feature and not something that workloads depend on. Therefore, any failure in calls to the /flagz endpoint is likely a bug or external issue like a network problem. So the SLO is essentially 100% success rate for the calls to flagz.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

This enhancement proposes data that can be used to determine the health of the component.
 (though this endpoint is not intended to be used for alerting.)

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Requests to `/flagz` will be tracked in the existing `apiserver_request_total` and `apiserver_request_duration_seconds` metrics.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No, each component's flagz is independent.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes, enabling this feature will result in a new HTTP endpoint (/flagz) being served by each component (including apiserver). However, this is not a Kubernetes API type or resource; it is a non-resource endpoint that provides component flag information for debugging and observability. No new Kubernetes API objects or resource types are introduced.

###### Will enabling / using this feature result in introducing new API types?

No, this feature does not introduce new Kubernetes API types or resources. While the flagz endpoint uses a structured JSON/YAML/CBOR response with Group/Version/Kind for content negotiation and consistency, it is not a Kubernetes API object and is not managed or persisted by the API server. The GVK is used solely to provide a predictable format for clients querying the endpoint.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Minimal to no impact expected, since the flagz endpoint is designed to provide lightweight, readily available information about a component's command-line flags. Retrieving this data should not introduce significant delays or resource contention that would noticeably affect the performance of existing operations covered by SLIs/SLOs.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Enabling and using the /flagz feature is expected to result in a negligible increase in resource usage across the components where it is implemented. The primary reason for this is that the endpoint will mainly serve static, readily available information about the component's command line flags.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

flagz endpoint for apiserver will not be available if the API server itself is down.

###### What are other known failure modes?

Overreliance on flagz for critical monitoring. We will clearly document the intended use cases and limitations of the flagz endpoint, emphasizing that it's primarily for informational and troubleshooting purposes.

###### What steps should be taken if SLOs are not being met to determine the problem?

The feature can be disabled by setting the feature-gate to false if the performance impact of it is not tolerable.

## Implementation History

- 1.32: New `flagz` endpoint introduced for [apiserver](https://github.com/kubernetes/kubernetes/pull/127581)
- 1.33: `/flagz` enablement extended to [kubelet](https://github.com/kubernetes/kubernetes/pull/128857), [scheduler](https://github.com/kubernetes/kubernetes/pull/128818), [controller-manager](https://github.com/kubernetes/kubernetes/pull/128824), and [kube-proxy](https://github.com/kubernetes/kubernetes/pull/128985)
- 1.35: Converted the `/flagz` endpoint to a structured API ([#134995](https://github.com/kubernetes/kubernetes/pull/134995)).
- 1.36: Added support for YAML and CBOR serialization for the `/flagz` endpoint ([#135309](https://github.com/kubernetes/kubernetes/pull/135309)). CBOR support is gated by the `CBORServingAndStorage` feature gate.

## Drawbacks

## Alternatives

## Infrastructure Needed (Optional)
