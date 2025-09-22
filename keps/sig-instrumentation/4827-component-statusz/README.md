# KEP-4827: Component Statusz

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
    - [As a developer](#as-a-developer)
    - [As an operator](#as-an-operator)
    - [As support engineer](#as-support-engineer)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Data Format and versioning](#data-format-and-versioning)
  - [Authz and authn](#authz-and-authn)
  - [Endpoint Response Format](#endpoint-response-format)
    - [Data format: text](#data-format-text)
      - [Request](#request)
      - [Sample response](#sample-response)
    - [Data format: JSON (v1alpha1)](#data-format-json-v1alpha1)
      - [Request](#request-1)
      - [Response Body: <code>Statusz</code> object](#response-body-statusz-object)
      - [Sample Response](#sample-response-1)
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

We propose to enhance the observability, troubleshooting, and debugging capabilities of core Kubernetes components by implementing a statusz endpoint for all components. By exposing standardized, real-time data about a component's basic information, health status, and key metrics in a low-overhead manner, this enhancement empowers users with deeper insights into the behavior and performance of each component.

## Motivation

Originating at Google and embedded by default in many internal servers, z-pages have proven invaluable for real-time monitoring and debugging of complex distributed systems. Recognizing the power and utility of this aggregated interface, we aim to extend its benefits to the broader open source community. 

With statusz, we envision the following advantages

* Consolidated and precise information: Presenting critical build data in a clear, concise format, eliminates the need to dig through logs or configurations to gather this essential information. Z-pages offer a distinct advantage over external monitoring solutions by providing insights from within the server itself. This inherent integration eliminates the need for complex external setups like manual inspection of logs, configuration files, or running external scripts that may be subject to potential network latency issues
* Streamlined troubleshooting and debugging:  Immediate access to crucial build information like binary version, Go version and compatibility details enables better understanding of the component's state. We hope this interface makes it easier to identify version mismatches or incompatibilities that might contribute to issues, accelerating troubleshooting and debugging efforts


### Goals

* Enhance observability into component details by providing a concise overview of critical component information, such as build version, Go version, compatibility details, and other pertinent data points
* Set a precedent for presenting this basic but crucial component information in a consistent and accessible manner across the Kubernetes ecosystem

### Non-Goals

* Replace existing monitoring solutions: metrics, logs, traces
* To provide a general-purpose extension mechanism for other components (e.g., container runtime, CNI, device plugins) to expose their own information through z-pages
* Provide information about components that are not directly accessible due to network restrictions
* **Report the status of internal sub-components or external dependencies.** Many Kubernetes components, such as the `kube-controller-manager` (with its multiple controller loops) or the `kube-apiserver` (with its dependency on etcd), are complex. Surfacing the status of all these individual pieces is a significantly more complex problem that this KEP does not aim to solve. This endpoint is intentionally limited to reporting the status of the main serving binary itself. A separate KEP would be required to address aggregated or detailed dependency status.

## Proposal

We are proposing to add a new endpoint, /statusz on all core Kubernetes components which returns key details regarding a component's current state, including versioning, build information, and compatibility.

The scope of this endpoint is intentionally limited to the status of the primary serving process. It will not report on the status of internal sub-components (like individual controller loops in `kube-controller-manager`) or external dependencies (like etcd for `kube-apiserver`). This focused approach avoids the complexities that made the legacy `core/v1 ComponentStatus` API difficult to maintain and reason about. Surfacing the status of dependencies is a distinct problem that may be addressed in a future enhancement.

### User Stories

#### As a developer

- I want to quickly identify the exact version of a binary running in production so I can correlate it with known issues or recent code changes
- I want to quickly access the compatibility version information of a deployed binary so I can efficiently debug upgrade failures and identify potential compatibility issues with dependencies

#### As an operator

- I need to troubleshoot a performance issue and want to easily check for dependency conflicts or version mismatches without sifting through logs

#### As support engineer

- I want to access detailed build information to determine if a customer is running a known problematic version of a component

### Risks and Mitigations

1. **No sensitive data exposed**
    
  We will apply the  principle of least privilege by granting only the minimum necessary permissions for accessing /statusz, which is primarily a debugging tool for operators and maintainers.
  /statusz will not be accessible through general discovery mechanisms like system:discovery. This prevents unintended exposure of potentially sensitive debugging information.
  
  We'll leverage Kubernetes RBAC to grant access to /statusz only to specific users and service accounts that require it for operational monitoring and debugging. For this, we will use the existing system:monitoring group ([ref](https://github.com/kubernetes/kubernetes/blob/release-1.31/staging/src/k8s.io/apiserver/pkg/authentication/user/user.go#L73)), ensuring that only authorized personnel can access this endpoint.
  
1. **Performance impact**
    
  We will take care to carefully consider the computational overhead associated with the statusz endpoint to avoid negatively impacting the server's overall performance.

2. **Essential data only**
    
  We will prioritize inclusion of only essential diagnostic and monitoring data that aligns with the intended purpose of the z-page.

3. **API Stability and Evolution**

  The alpha release will support both a plain text format and a versioned JSON format (`v1alpha1`). The feature will be secured behind a feature gate that is disabled by default, ensuring users opt-in consciously. The `v1alpha1` JSON format is explicitly marked as alpha-quality, intended for early feedback, and is subject to change.

  For Beta, to provide a stable contract for consumers, the JSON response API will be promoted to `v1beta1` or `v1` based on feedback and will be considered stable. The plain text format remains the default for human consumption and backward compatibility.

## Design Details

### Data Format and versioning

The `/statusz` endpoint defaults to a `text/plain` response format, which is intended for human consumption and should not be parsed by automated tools.

For programmatic access, a structured, versioned JSON format is available with an initial alpha version of `v1alpha1`. This version is not stable and is intended for feedback and iteration during the Alpha phase.

- **Group**: `config.k8s.io`
- **Version**: `v1alpha1` (initial version, subject to change)
- **Kind**: `Statusz`

To receive the structured JSON response, a client **must** explicitly request it using an `Accept` header specifying these parameters. For example:

```
GET /statusz
Accept: application/json;as=Statusz;v=v1alpha1;g=config.k8s.io
```

This negotiation mechanism ensures clients are explicit about the exact API they want, preventing accidental dependencies on unstable or incorrect formats. If a client requests `application/json` without the required parameters, the server will respond with a `406 Not Acceptable` error.

### Authz and authn

Access to the endpoint will be limited to members of the `system:monitoring group` ([ref](https://github.com/kubernetes/kubernetes/blob/release-1.31/staging/src/k8s.io/apiserver/pkg/authentication/user/user.go#L73)), which is how paths like /healthz, /livez, /readyz are restricted today. 

// NOTE: Placeholder suggestion for handling kubelet auth. Subject to change based on https://github.com/kubernetes/kubernetes/issues/127990

To grant access to kubelet's /statusz endpoint, which is not typically accessible through the system:monitoring role, we can modify the `system:monitoring` role as follows
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: system:monitoring 
rules:
# existing rules
- apiGroups: [""]
  resources: ["nodes/statusz"]
  verbs: ["get"] 
```

### Endpoint Response Format

The `/statusz` endpoint supports both plain text and structured JSON formats. The default format is `text/plain`.

#### Data format: text

This format is the default and is intended for human readability.

##### Request
* Method: **GET**
* Endpoint: **/statusz**
* Header: `Accept: text/plain` (or no Accept header)
* Body: empty

##### Sample response

```
Started: Fri Sep  6 06:19:51 UTC 2024
Up: 0 hr 00 min 30 sec
Go version: go1.23.0
Binary version: 1.31.0-beta.0.981&#43;c6be932655a03b-dirty
Emulation version: 1.31.0-beta.0.981
Minimum Compatibility version: 1.30.0

List of paths
--------------
configz:/configz
healthz:/healthz
livez:/livez
metrics:/metrics
readyz:/readyz
sli metrics:/metrics/slis
```

#### Data format: JSON (v1alpha1)

This format is available in Alpha for programmatic access and must be explicitly requested. It is considered an alpha-level format.

##### Request
* Method: **GET**
* Endpoint: **/statusz**
* Header: `Accept: application/json;as=Statusz;v=v1alpha1;g=config.k8s.io`
* Body: empty

##### Response Body: `Statusz` object

The response is a `Statusz` object.

###### Go Struct Definition
```go
// Statusz is the structured response for the /statusz endpoint.
type Statusz struct {
	// Kind is "Statusz".
	Kind string `json:"kind"`
	// APIVersion is the version of the object, e.g., "config.k8s.io/v1alpha1".
	APIVersion string `json:"apiVersion"`

	// Standard object's metadata.
	// +optional
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`

	// StartTime is the time the component process was initiated.
	StartTime metav1.Time `json:"startTime"`
	// UptimeSeconds is the duration in seconds for which the component has been running continuously.
	UptimeSeconds int64 `json:"uptimeSeconds"`

	// GoVersion is the version of the Go programming language used to build the binary.
	// The format is not guaranteed to be consistent across different Go builds.
	// +optional
	GoVersion string `json:"goVersion,omitempty"`
	// BinaryVersion is the version of the component's binary.
	// The format is not guaranteed to be semantic versioning and may be an arbitrary string.
	BinaryVersion string `json:"binaryVersion"`
	// EmulationVersion is the Kubernetes API version which this component is emulating.
	// if present, formatted as "<major>.<minor>"
	// +optional
	EmulationVersion string `json:"emulationVersion,omitempty"`
	// MinimumCompatibilityVersion is the minimum Kubernetes API version with which the component is designed to work.
	// if present, formatted as "<major>.<minor>"
	// +optional
	MinimumCompatibilityVersion string `json:"minimumCompatibilityVersion,omitempty"`

	// Paths contains relative URLs to other essential read-only endpoints for debugging and troubleshooting.
	// +optional
	Paths []string `json:"paths,omitempty"`
}
```

###### JSON Structure
```json
{
  "kind": "Statusz",
  "apiVersion": "config.k8s.io/v1alpha1",
  "metadata": {
    "name": "<component-name>"
  },
  "startTime": "<start-time>",
  "uptimeSeconds": <uptime-in-seconds>,
  "goVersion": "<go-version>",
  "binaryVersion": "<binary-version>",
  "emulationVersion": "<emulation-version>",
  "minimumCompatibilityVersion": "<min-compatibility-version>",
  "paths": [
    "/configz",
    "/healthz",
    "/livez",
    "/metrics",
    "/readyz",
    "/metrics/slis"
  ]
}
```

##### Sample Response

```json
{
  "kind": "Statusz",
  "apiVersion": "config.k8s.io/v1alpha1",
  "metadata": {
    "name": "apiserver"
  },
  "startTime": "2024-09-06T06:19:51Z",
  "uptimeSeconds": 30,
  "goVersion": "go1.23.0",
  "binaryVersion": "1.31.0-beta.0.981+c6be932655a03b-dirty",
  "emulationVersion": "1.31",
  "minimumCompatibilityVersion": "1.30",
  "paths": [
    "/configz",
    "/healthz",
    "/livez",
    "/metrics",
    "/readyz",
    "/metrics/slis"
  ]
}
```
### API Versioning and Deprecation Policy

The versioned `/statusz` endpoint will follow the standard [Kubernetes API deprecation policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/).

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

N/A

##### Unit tests

- `staging/src/k8s.io/component-base/zpages/statusz`: Unit tests will be added to cover both the plain text and structured JSON output, including serialization and content negotiation logic.

##### Integration tests

- Integration tests will be added for each component to verify that the `/statusz` endpoint is correctly registered and serves both `text/plain` and the versioned `application/json` content types.

##### e2e tests

- E2E tests will be added to query the `/statusz` endpoint for each core component and validate the following:
  - The endpoint is reachable and returns a `200 OK` status.
  - Requesting with the `Accept` header for `application/json;as=Statusz;v=v1alpha1;g=config.k8s.io` returns a valid `Statusz` JSON object.
  - The JSON response can be successfully unmarshalled.
  - Requesting with `text/plain` or no `Accept` header returns the non-empty plain text format.

### Graduation Criteria

#### Alpha

- Feature gate `ComponentStatusz` is disabled by default.
- A structured JSON response (`config.k8s.io/v1alpha1`) is introduced for feedback, alongside the default `text/plain` format.
- Feature is implemented for at least one component (e.g., kube-apiserver).
- E2E tests are added for both plain text and the new JSON response format.
- Gather feedback from users and developers on the structured format.

#### Beta

- Feature gate `ComponentStatusz` is enabled by default.
- The JSON response API is promoted to `v1beta1` or `v1` based on feedback and is considered stable.
- Feature is implemented for all core Kubernetes components.

#### GA

- The JSON response API is promoted to a stable `v1` version after bake-in.
- Conformance tests are in place for the endpoint.

#### Deprecation

### Upgrade / Downgrade Strategy

In alpha, no changes are required to maintain previous behavior. And the feature gate can be turned on to make use of the enhancement.

### Version Skew Strategy

The primary purpose of the statusz page is to provide information about the individual component itself. While the underlying data format may be enhanced over time, the endpoint itself will remain a reliable source of status information for Kubernetes components.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

We will target this feature behind a flag `ComponentStatusz`

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ComponentStatusz
  - Components depending on the feature gate:
    - apiserver
    - kubelet
    - scheduler
    - controller-manager
    - kube-proxy

###### Does enabling the feature change any default behavior?

Yes it will expose a new statusz endpoint. 

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. But it will remove the statusz endpoint

###### What happens if we reenable the feature if it was previously rolled back?

It will expose the statusz endpoint again

###### Are there any tests for feature enablement/disablement?

Unit test will be introduced in alpha implementation.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

This feature should not cause rollout failures. If it does, we can disable the feature. In the worst
case, it is possible it could cause runtime failures, but it is highly unlikely we would not detect this
with existing tests.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

- apiserver_request_duration_seconds metric that will tell us if there's a spike in request latency of kube-apiserver that might indicate that the statusz endpoint is interfering with the component's core functionality or causing delays in processing other requests

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

They can curl a component's `statusz` endpoint. However, its important to note that this feature is not directly used by workloads, its a debug feature.

###### How can someone using this feature know that it is working for their instance?

- They can curl a component's `statusz` endpoint, which should return data.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This is a debugging feature and not something that workloads depend on. Therefore, any failure in calls to the /statusz endpoint is likely a bug or external issue like a network problem. So the SLO is essentially 100% success rate for the calls to statusz.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

This enhancement proposes data that can be used to determine the health of the component.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. We are open to input.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No, each component's statusz is independent.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Minimal to no impact expected, since the statusz endpoint is designed to provide lightweight, readily available information about a component's version and build details. Retrieving this data should not introduce significant delays or resource contention that would noticeably affect the performance of existing operations covered by SLIs/SLOs.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Enabling and using the /statusz feature is expected to result in a negligible increase in resource usage across the components where it is implemented. The primary reason for this is that the endpoint will mainly serve static, readily available information about the component's version and build details

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

statusz endpoint for apiserver will not be available if the API server itself is down.

###### What are other known failure modes?

Overreliance on statusz for critical monitoring. We will clearly document the intended use cases and limitations of the statusz endpoint, emphasizing that it's primarily for informational and troubleshooting purposes, not real-time monitoring or alerting.

###### What steps should be taken if SLOs are not being met to determine the problem?

The feature can be disabled by setting the feature-gate to false if the performance impact of it is not tolerable.

## Implementation History

## Drawbacks

## Alternatives

## Infrastructure Needed (Optional)
