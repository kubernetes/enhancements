# KEP-2411: CRI Container Log Rotation

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
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The CRIContainerLogRotation feature gate was implemented in v1.10 and has been in Beta stage since v1.11. We would like to identify any gaps in the implementation of this feature so that we can promote it to stable as it has already been in production use for quite some time now. With this feature gate, the kubelet is in charge of managing the container log directory structure as well as rotating the logs when a certain (user configurable) limit is reached.

## Motivation

Container runtimes that communicate with the kubelet via the Container Runtime Interface (CRI) needed a container log management system. The kubelet was already in charge of determining the container log file path and passing that down to the container runtime so that it can write the container logs there. Thus making the kubelet in charge of rotating the container logs allows the kubelet to manage and access the logs directly without having to call the container runtime. An added advantage of this is that logging agents can ingest files directly without any further integrations with the container runtime.

### Goals

- The kubelet assigns the log path for a container and runtime writes the container output to that path
- The kubelet periodically checks the disk space occupied by the container logs and rotates them if necessary
- After the logs are rotated, the kubelet sends a signal to the container runtime to re-open the log file
- The kubelet exposes a consistent log directory structure with metadata so that any logging agent can integrate with it

### Non-Goals

- Shipping the logs directly to a remote storage
- Supporting container runtimes that run on a different virtual/physical machine from the kubelet
- Allow kubelet to manage the lifecycle of the logs to pave the way for better disk management in the future. This implies that the lifecycle of containers and their logs need to be decoupled.

## Proposal

Graduate the CRIContainerLogRotation feature gate from Beta to Stable. The kubelet already decides the container log directory structure and passes that down to the container runtime. The container runtime then writes the container's logs to this location. This makes the kubelet the best candidate to manage the rotation of the container logs as it already know the log directory structure and has access to it. The CRILogRotation feature gate implementation adds a container log manager package, which manages and rotates the container logs. It also adds 2 flags to the kubelet that allows the user to configure the maximum size of each log file and the maximum number of log files to retain. These flags are **--container-log-max-size** and **--container-log-max-files**.

### User Stories (Optional)

#### Story 1

As a kubernetes user, I want to use a CRI container runtime and want the container logs to be managed and rotated by the kubelet, so I don't have to worry about logs filling up my disk space and can access older longs when needed. I also want to be able to configure the size of my log file and how many log files to retain.

#### Story 2

As a kubernetes user, I want to integrate a logging agent that aggregates the logs to a remote storage, so I can easily access my logs in a centralized location without needing to access each node on my cluster to view the logs.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

- Loss of some logs during log rotation. There is an open issue on this with a suggested fix https://github.com/kubernetes/kubernetes/issues/64760. Have added this to the graduation criteria as well.

## Design Details

This implementation adds container log manager package, which the kubelet uses to manage and rotate the logs. The container log manager will only start up when the container runtime being used is one that communicates with the kubelet via the CRI i.e CRI-O, containerd, etc.

The rotated logs are compressed with gzip. The latest rotated log is not compressed as a logging agent, such as fluentd, might still be reading it right after rotation and/or the container runtime might still be writing to it shortly after getting the path to the new log file. The kubelet periodically checks the amount of disk space being used by the container logs and rotates them if the max value has been reached. After the logs are rotated, the kubelet sends a signal to the container runtime to re-open the log file.

The user can configure the maximum size of a log file and the maximum number of log files to retain with the **--container-log-max-size** and **--container-log-max-files** flags. The default values are **10Mi** for the max file size and **5** for the max number of log files. These parameters will only be applied if a CRI container runtime is being used, it will be ignored for dockershim.

The kubelet exposes a consistent log directory structure with embedded metadata so that logging agents can integrate with it is. Since the kubelet is in charge of setting the log directory structure and can directly access and manage the log files, the logging agents can directly work with the kubelet without having to make any further integrations with the container runtime in use.

### Test Plan

- There are currently unit tests and node E2E integration tests for container log rotation
- Get feedback on performance and stability of the CRI log format on other products aside from OpenShift and GKE

### Graduation Criteria

#### Alpha -> Beta Graduation

- Unit and node E2E tests are consistently passing
- Logging agents can easily integrate with kubernetes and push rotated logs to a remote storage
- Successful log rotations by the kubelet

#### Beta -> GA Graduation

- Successfully run in production
- Solicit feedback in SIG Node community that there are no issues with individual distributions production usage (OpenShift and GKE both report no major issue)

### Upgrade / Downgrade Strategy

On Upgrade: feature will be available to use as it already is, but will be promoted to GA.

On downgrade: feature will be available to use when the feature gate is set, but will be moved back to Beta.

### Version Skew Strategy

Since this feature was promoted to Beta in v1.11, it will still be available with a n-2 kubelet. No coordination with the control plane is required. Changes to any other components on the node are not needed.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

- **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: CRIContainerLogRotation
    - Components depending on the feature gate: Kubelet
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

- **Does enabling the feature change any default behavior?**

  With the dockershim, the docker daemon was in charge of managing and rotating the logs. With a CRI container runtime, the kubelet is in charge of managing and rotating logs. There is no real change to default behavior apart form the fact that the log rotation will depend on which container runtime is being used.

- **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

  Yes, but if disabled the container logs will not be rotated when using a CRI container runtime.

- **What happens if we reenable the feature if it was previously rolled back?**

  No impact, container log rotation will work for CRI container runtimes.

- **Are there any tests for feature enablement/disablement?**

  There are already unit and node e2e tests in place for this feature.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

- **How can a rollout fail? Can it impact already running workloads?**

   When the container log manager doesn't start up and rotate the logs as expected. Restarts shouldn't affect ths as it is the container runtime that will be writing the logs. The kubelet is in charge of checking log size and rotating when needed.

- **What specific metrics should inform a rollback?**

  - There is major loss of logs on nodes that use a CRI runtime.
  - Logging agents are unable to integrate with k8s and aggregate logs to a remote storage.

- **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  
  Any manual testing was done when the feature was initially implemented in https://github.com/kubernetes/kubernetes/pull/59898.

- **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  
  No

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

- **How can an operator determine if the feature is in use by workloads?**

  When a CRI container runtime is used, the logs are being rotated and stored in the log directory structure with a gzip format.

- **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [x] Other (treat as last resort)
    - Details: Error messages logged in the kubelet journal when there is a failure to rotate the logs or delete old log files.

- **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  
  100% of logs are rotated according the configured max size and files without any loss.

- **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**

  N/A

### Dependencies

_This section must be completed when targeting beta graduation to a release._

- **Does this feature depend on any specific services running in the cluster?**

  - [Kubelet]
    - Usage description: Responsible for managing the log directory structure and rotating the logs.
      - Impact of its outage on the feature: Logs will not be rotated. Container runtime may continue to write logs to the file even after the max size has been reached. This could bring the cluster down, if the logs continue to grow uncontrolled without pod eviction enabled.
      - Impact of its degraded performance or high-error rates on the feature: Loss of logs during rotation. Logging agents may have issues aggregating the logs.

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

- **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - Re-open container log file after logs are rotated
  - Not much, only the container ID
  - Kubelet
  - None
  - This will be triggered after the max file size for logs has been reached and the kubelet has rotated the logs


- **Will enabling / using this feature result in introducing new API types?**

  No

- **Will enabling / using this feature result in any new calls to the cloud 
provider?**

  No

- **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**

  No

- **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**

  No

- **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**

  No

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

- **How does this feature react if the API server and/or etcd is unavailable?**

Container logs are written to a path on disk that the kubelet directly manages, so there should be no impact if the etcd and/or API server is unavailable

- **What are other known failure modes?**
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

- **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

Original Issue: https://github.com/kubernetes/kubernetes/issues/58823
First PR with implementation: https://github.com/kubernetes/kubernetes/pull/59898
Original design doc with solutions considered: https://docs.google.com/document/d/1oQe8dFiLln7cGyrRdholMsgogliOtpAzq6-K3068Ncg/edit#
Follow up PR: https://github.com/kubernetes/kubernetes/pull/58899
Graduation to Beta: https://github.com/kubernetes/kubernetes/pull/64046
