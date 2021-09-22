
# KEP-2907: Secrets Store CSI Provider

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
    - [Directory traversal vulnerabilities](#directory-traversal-vulnerabilities)
    - [Authenticating to external secret APIs](#authenticating-to-external-secret-apis)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [GA](#ga)
    - [Deprecation](#deprecation)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
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

The [secrets-store-csi-driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver) project provides a portable method for applications to consume secrets from external secret APIs through the filesystem. This effort was added to the `sig-auth` subproject in February 2020 and currently there are providers for Azure, AWS, GCP, and HashiCorp Vault. All the providers for the driver are out-of-tree. This KEP intends to cover making the core functionality of the driver GA.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

- Signal the stability of the driver interface and implementation for the core task of making secrets available to pod filesystem.

### Non-Goals

- Extending the Kubernetes Secret object
- Introduce a new Kubernetes type

## Proposal

This project introduces a new Container Storage Interface (CSI) driver for fetching secrets and writing to a `tmpfs` mount in the Pod filesystem. The driver is deployed as a `DaemonSet`. A new Custom Resource Definition (CRD) called a `SecretProviderClass` is introduced that informs the driver of which external secret storage API to contact and how to map the secrets from those APIs to file paths. The driver communicates with the external secret provider processes through a gRPC interface over a Unix Domain Socket.

### User Stories (Optional)

#### Story 1

1. Application reads secret from filesystem on startup
2. Application watches secret for rotation
3. Application Pod YAML remains unchanged and works across secret providers

### Notes/Constraints/Caveats (Optional)

Since the proposal is a storage driver, native support for presenting secrets to a process through environment variables is not possible. In addition to the default mount, the driver also supports syncing the mounted content as Kubernetes secret. This is an optional feature and isn't enabled by default.

### Risks and Mitigations

#### Directory traversal vulnerabilities

The driver<->provider interface has been expanded to allow the driver to be the only process that actually writes files to the pod filesystem. The only hostpath providers need are now the one for creating the unix socket used for communication with the driver process.

The driver protects against directory traversal vulnerabilities by re-using the `atomic_writer` used by Kubernetes Secrets and ConfigMaps which includes protections against writing to unintended paths.

#### Authenticating to external secret APIs

Authentication and authorization is largely up to the external API and its provider process, but the driver itself does include a few features that enable scoping access to secrets to individual pods.

[Pod info](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/603-csi-pod-info) is propagated to the external provider.

Additionally [KEP 1855](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/1855-csi-driver-service-account-token) allows the driver to propagate a token to the provider. This enabled the providers to impersonate the pod for fetching secrets without the need granting large RBAC scopes.

## Design Details

### Test Plan

- Automated pre and post submit end-to-end integration tests
- Periodic end-to-end integration tests
- Supported providers each have their own integration test suites along with a reference provider

### Graduation Criteria

#### GA

- Month+ soak of minor release
- Completion of milestone requirements
- Agreement of stability documented on community call from 3+ provider maintainers

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

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

This will be deployed to clusters as a standalone separate component.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

- non-zero `total_node_publish` metrics indicate the CSI driver is used by the workloads.
- `total_sync_k8s_secret` metrics indicate the optional Sync as Kubernetes secret feature is used by the workloads.
- `total_rotation_reconcile` metrics indicate the optional rotation reconciliation feature is used by the workloads.

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- `total_node_publish_error`
  - any rising count of this metric indicates a problem with mounting the volume for pod.
- `total_node_unpublish_error`
  - any rising count of this metric indicates a problem with unmounting the volume for pod.

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

- [x] Metrics
  - Metric name: `total_node_publish`
  - Components exposing the metric: `secrets-store-csi-driver`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

- [Kubernetes Container Storage Interface](https://github.com/kubernetes/community/blob/98b3d97d2e7f91bb62b8e88710c29c1675efb689/contributors/design-proposals/storage/container-storage-interface.md)
- [KEP 596](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/596-csi-inline-volumes)
- Supports windows containers (Kubernetes version v1.18+)
- [KEP 1855: Service Account Token for CSI Driver](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/1855-csi-driver-service-account-token)

The driver uses CSI Inline Volumes to mount the external secrets-store objects in the pod. The CSI Inline Volumes feature is enabled by default in Kubernetes 1.16+. For windows containers, the CSI Inline Volumes feature is enabled by default in Kubernetes 1.18+.

The minimum supported Kubernetes version is 1.16 for Linux and 1.18 for Windows.

###### Does this feature depend on any specific services running in the cluster?

- Kubelet
  - If kubelet service is not running, the pods referencing the csi driver for volume will fail to start.

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

Load test results: https://secrets-store-csi-driver.sigs.k8s.io/load-tests.html

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

### Troubleshooting

https://secrets-store-csi-driver.sigs.k8s.io/troubleshooting.html

## Implementation History

- Dec 2018 - [First Commit](https://github.com/kubernetes-sigs/secrets-store-csi-driver/commit/56fb54bfdb2058ef043ff36363b90787a52c51b7#diff-bc37d034bad564583790a46f19d807abfe519c5671395fd494d8cce506c42947)
- Feb 2020 - [Incorporated into sig-auth with pluggable provider model](https://github.com/kubernetes/org/issues/1245)
- May 2020 - [v0.0.10](https://github.com/kubernetes-sigs/secrets-store-csi-driver/releases/tag/v0.0.10) (First release)
- July 2021 - [v0.1.0](https://github.com/kubernetes-sigs/secrets-store-csi-driver/releases/tag/v0.1.0) (First minor release)

## Drawbacks

- Environment Variables: There appears to be a strong desire to consume secrets using environment variables but the only way for this to work currently is through syncing secrets to Kubernetes Secrets.
- Consume only: The proposed CRD and implementation does not provide a way to add/write/edit secrets in the external stores.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
