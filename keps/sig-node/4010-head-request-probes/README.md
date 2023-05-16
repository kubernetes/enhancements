# KEP-4010: HEAD Request Probes

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
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
  - [Rollout, Upgrade, and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
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

This Kubernetes Enhancement Proposal (KEP) introduces the capability to use HTTP HEAD requests for defining readiness and liveness probes in Kubernetes pods. Currently, only GET requests are supported for probes, limiting the options for checking the health of containers. By allowing the use of HEAD requests, Kubernetes users will have more flexibility and finer-grained control over their probes, especially for customer-facing applications.

## Motivation

The motivation behind this proposal is to enhance Kubernetes' readiness and liveness probes by supporting HTTP HEAD requests. While GET requests are commonly used, they might not always be suitable for certain scenarios, such as checking the availability of resources without fetching their content. The ability to use HEAD requests will enable more efficient health checks for specific use cases, particularly those involving customer-facing services. By extending the probe functionality, Kubernetes aims to provide a more versatile and customizable platform for managing container health.

### Goals

- Allow Kubernetes users to utilize HTTP HEAD requests for defining readiness and liveness probes.
- Enhance the probe functionality to support more efficient and lightweight health checks.
- Provide additional flexibility for customer-facing applications that require specific health check behaviors.

### Non-Goals

- Modifying existing HTTP GET-based probes.
- Introducing new HTTP request methods beyond HEAD.

## Proposal

The proposal involves extending Kubernetes' readiness and liveness probes to support HTTP HEAD requests. This will involve modifying the Kubernetes API to allow users to specify the request method as HEAD for these probes. When a HEAD request is used, only the response headers will be retrieved, without the response body. This lightweight operation is ideal for checking the availability of resources without incurring unnecessary network traffic or consuming additional system resources.

For example, a pod's readiness probe configuration can be updated as follows:

```yaml
readinessProbe:
  httpHead:
    headers:
      - name: X-Custom-Header
        value: some-value
    queryParams:
      - name: token
        value: abc123
    path: /health
    port: 8080
    scheme: HTTPS
    timeoutSeconds: 3
    periodSeconds: 5
```

In this example, the readiness probe is configured to use an HTTP HEAD request with custom headers, query parameters, and a specific path, port, and scheme.

There is an [untested solution how this feature might be implemented](https://github.com/kubernetes/kubernetes/commit/e3d556f0920ff736fec5df73c42f6db93f436a37).  
Note: this solution is also missing a feature gate, which will be required for the final implementation.

### Risks and Mitigations

Introducing support for HTTP HEAD requests in probes may pose the following risks:

- Increased complexity: Adding support for HEAD requests may introduce additional complexity to the Kubernetes codebase. This risk can be mitigated through careful design and thorough testing.

## Design Details

1. Extend the existing readiness and liveness probe specifications to include an optional `httpHead` field.
2. When `httpHead` is specified, the probe will use an HTTP HEAD request instead of the default GET request.
3. The `httpHead` field will accept an object that allows specifying additional headers, query parameters, and any other relevant fields required for the HEAD request.
4. If `httpHead` is not specified, the probe will default to using a GET request, ensuring backward compatibility with existing probes.
5. Modify the Kubernetes health check mechanism to handle HEAD requests appropriately, including parsing the response and determining the pod's health based on the HEAD response status code.

### Test Plan

The following tests should be performed to ensure the proper functioning of the new HEAD request support:

#### Prerequisite testing updates

- Update existing tests for readiness and liveness probes to include scenarios using HEAD requests.

#### Unit tests

- Add unit tests to verify the correct handling of HEAD requests in the Kubernetes API server and Kubelet components.
- Test the behavior of the probes when HEAD requests are used.

#### Integration tests

- Perform integration tests to validate the interoperability of the updated components with other Kubernetes functionality.
- Verify the correct behavior of HEAD-based probes in various scenarios, including failures and recoveries.

#### e2e tests

- Create end-to-end tests that cover the entire lifecycle of pods with HEAD-based probes.
- Validate the behavior of probes in real-world scenarios, ensuring they function as expected.

### Graduation Criteria

#### Alpha

- [ ] Feature implemented behind a feature flag.
- [ ] e2e tests completed and enabled.

#### Beta

- [ ] No major bugs reported in the previous cycle.

#### GA

- [ ] Allowing time for feedback (1 year).
- [ ] Risks have been addressed.

### Upgrade / Downgrade Strategy

To upgrade to a version of Kubernetes that supports HEAD requests in probes, users will need to update their API server, Kubelet, and client libraries. Compatibility checks should be performed during the upgrade

### Version Skew Strategy

To manage version skew during the introduction of support for HTTP HEAD requests in readiness and liveness probes, the following strategy will be implemented:

- Documentation: Clear documentation will be provided that outlines the supported Kubernetes versions for the new feature. This documentation will include information on compatibility between different Kubernetes components.

- Compatibility Checks: During the upgrade process, compatibility checks will be performed to ensure that the Kubernetes cluster components are running compatible versions. These checks will help identify any version skew issues and provide guidance on resolving them before enabling the new feature.

## Production Readiness Review Questionnaire

Before enabling support for HTTP HEAD requests in readiness and liveness probes in a production environment, the following aspects will be assessed through a readiness review questionnaire:

### Feature Enablement and Rollback

- What flags or configuration settings are required to enable the new feature?

A feature flag `HeadRequestProbes` will be introducded.

- Can the feature be disabled or rolled back if issues arise?

Yes.

- Are there any specific considerations for enabling or disabling the feature on a live cluster?

No.

### Rollout, Upgrade, and Rollback Planning

- What is the recommended rollout strategy for enabling the new feature?

Update to the version which has the new feature implemented and enable the feature gate.

- Are there any specific considerations or steps to follow during the rollout process?

No.

- Are there any backward compatibility concerns to address during the rollout?

No.

### Monitoring Requirements

- Are there any additional monitoring requirements specific to HTTP HEAD-based probes?

Metrics need to be checked. If GET vs. HEAD differ, new metrics must be implemented.

### Dependencies

- Are there any dependencies on other Kubernetes features, components, or external systems?

No.

- Are there any specific version requirements for these dependencies?

No.

### Scalability

- How does the new feature impact the scalability of Kubernetes clusters?

It does not impact the scalability.

- Are there any scalability considerations or limitations to be aware of?

No.

### Troubleshooting

- What are the common issues or failure scenarios related to the new feature?

That the HEAD endpoint is not implemented at the application (obvious).

- What troubleshooting steps can be taken to diagnose and resolve these issues?

Request the health endpoint with a HEAD request directly.

- Are there any recommended logs or diagnostic information that should be collected for troubleshooting purposes?

No.

## Implementation History

- 16-05-2023: Proposal drafted.

## Drawbacks

The introduction of support for HTTP HEAD requests in readiness and liveness probes may have the following drawbacks:

- Increased Complexity: The new feature may add complexity to the Kubernetes codebase, potentially affecting maintenance and troubleshooting efforts.

## Alternatives

Some alternative approaches to consider instead of introducing support for HTTP HEAD requests in readiness and liveness probes include:

- Custom Handlers: Users could implement custom HTTP handlers within their containers to handle specific health checks instead of relying solely on standardized probes.
- Sidecar Containers: Users could deploy sidecar containers alongside their main application containers to perform custom health probes.
