
# Kubelet CRI support

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Identified Work Items](#identified-work-items)
    - [Changes from v1alpha2 to v1beta1](#changes-from-v1alpha2-to-v1beta1)
    - [Clean Up](#clean-up)
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
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary
Identify remaining gaps to promote CRI to Beta and GA to reflect its practical use in production for many years.

## Motivation
CRI based runtimes such as CRI-O and containerd have been in use in production for over a year now with the current CRI API.
We want to signal to the users that the CRI API is production ready and they should feel comfortable moving away dockershim
as it is slated to be deprecated.

### Goals
- Graduate the CRI API to stable.
- Identify any fields that need to made more type safe such as Seccomp.
- Address and cleanup the notes/todos in the CRI.

### Non-Goals
- Block on any big new features.

## Proposal
Evolve the CRI API version as we address feedback in each milestone towards stable.
- v1alpha2 (alpha, current state)
- v1beta (beta, proposed 1.20)
- v1 (stable, TBD)

### Risks and Mitigations
| Risk  | Detail  | Mitigation  |
|---|---|---|
| CRI stats performance  | CRI stats performance may be worse compared to cadvisor  |  Measure performance and share report with community |

## Design Details

### Identified Work Items
- No longer map the `container-runtime-endpoint` flag as experimental.
- Keep the `image-service-endpoint` flag as experimental and evaluate if it makes sense to keep
  as a configurable or remove it.

#### Changes from v1alpha2 to v1beta1

- kubenet: There exists an open TODO in the specification to remove support for setting PodCidr for kubenet networking. However for CRI
implementations CNI is the existing standard and is primarily the only solution being tested with the CRI container runtime integrations.
Need Sig-Node and Sig-Networking to help validate if / when kubenet is being deprecated and if we should deprecate this before beta. If not when. 
   https://github.com/kubernetes/kubernetes/issues/62288
   type NetworkConfig struct {
	   // CIDR to use for pod IP addresses. If the CIDR is empty, runtimes
	   // should omit it.
	   PodCidr              string   `protobuf:"bytes,1,opt,name=pod_cidr,json=podCidr,proto3" json:"pod_cidr,omitempty"`
	   XXX_NoUnkeyedLiteral struct{} `json:"-"`
	   XXX_sizecache        int32    `json:"-"`
   }

#### Clean Up
- Removal of TODOs that are no longer valid should be done before v1beta. We have scraped the api specification once and have a small
list of commits to file.

### Test Plan
   - Review of the existing test cases in critest and adding more if we find any gaps.
   - Make sure we have e2e node (and possibly selected e2e conformance) tests running on more than one CRI implementation.

### Graduation Criteria

#### Alpha -> Beta Graduation

- Passes all existing CRI tests on at least two container runtimes (sig-node(e2e-node) and cri-tools(critest)).
- Is in production on numerous clouds. (Note: this reflects the urgency of the signal to move off non CRI solutions.)
- Documentation is updated to reflect beta status. 
- Update the CI with containerd and CRI-O versions that support the v1 proto.
- Ensure that the required CRI stats changes are included. See https://github.com/kubernetes/enhancements/issues/2371.

#### Beta -> GA Graduation
- TBD

### Upgrade / Downgrade Strategy
Kubelet and the runtime versions should use the same CRI version in lock-step.
Upgrade involves draining all pods from a node, installing a CRI runtime with this version of the API and update to a matching kubelet
and then make the node schedulable again.

### Version Skew Strategy
Kubelet and the CRI runtime versions are expected to match so we don't have to worry about it for v1beta1.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback
* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [x] Other
    - Describe the mechanism:
      Install, configure, and run a CRI runtime on a node. Change the kubelet configuration to point to the CRI runtime socket and restart the kubelet.
    - Will enabling / disabling the feature require downtime of the control
      plane?
      No. The control plane nodes could be modified one at a time to switch to CRI runtimes.
    - Will enabling / disabling the feature require downtime or reprovisioning of a node?
      Yes. One could re-provision an existing nodes or provision new nodes with a CRI runtime and kubelet configured to talk to that runtime and then
      migrate your existing workloads to the new nodes.

* **Does enabling the feature change any default behavior?**
  - It changes the default container runtime from dockershim, but the container workloads are expected to work the same way as they do with dockershim.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?**
  Yes, the users could switch back to dockershim on a node reversing the process of installing CRI based runtime.

* **What happens if we reenable the feature if it was previously rolled back?**
  No impact per existing kubernetes policy for draining nodes for node lifecyle. IOW container runtime being used CRI or internal docker-shim is tied
  to node lifecycle operations.

* **Are there any tests for feature enablement/disablement?**
   No impact for v1 vs v2 or v1 point release extensions for the CRI api. Instead container runtimes will expose grpc service endpoints on a single socket
   for the CRI services as separate v1/v2 service types. A container runtime would have to provide two endpoints for each service by type if it wants to
   support two different versions of kubelet/runtime integrations.
  
### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Workloads scheduled on nodes with a CRI runtime may fail due to some misconfiguration of that node. Yes, it could impact running workloads
  since we depend upon draining and moving workloads around to switch to CRI runtimes on a node.

* **What specific metrics should inform a rollback?**
  - Nodes that have been switched to CRI runtime are not ready.
  - Workloads are failing to come up on a CRI configured node.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  We don't expect to do any automated upgrade or rollback from and to dockershim so this doesn't apply. 

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  - TODO

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  The Node object returns the configured CRI runtime.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [x] Other (treat as last resort)
    - Details:
      - Node Ready

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  TBD

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  TBD
  
### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  No.

### Scalability

_For beta, this section is required: reviewers must answer these questions._

* **Will enabling / using this feature result in any new API calls?**

 Exec/attach/port forwarding go through the API server.
 
* **Will enabling / using this feature result in introducing new API types?**

  No, new k8s API types besides seccomp changes in the CRI.

* **Will enabling / using this feature result in any new calls to the cloud provider?**

  No.

* **Will enabling / using this feature result in increasing size or count of the existing API objects?**

  No.

* **Will enabling / using this feature result in increasing time taken by any operations covered by [existing SLIs/SLOs]?**

  No. 

* **Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?**

  We have an open item to dive deeper into perf comparison of CRI stats vs. cadvisor and will update here once we have more data.

### Troubleshooting

- Troubleshooting for CRI integrations requires a good set of documentation be provided by kubernetes for interactions with pods,
containers, kubelet, security profiles, crictl (cri-tools client), developer and test (e2e, node, and critest) guides, and each
of the container runtimes. As such, while in v1Beta there should be workgroup(s) formed or issues tracked for reaching GA criteria
over the beta to GA period.  

* **How does this feature react if the API server, kubelet and/or etcd is unavailable?**
 - Open streams for Exec/Attach/Port forward that are forwarded by kubelet to API Server will likely timeout and close if
   API Server becomes unavailable.
 - CRI Runtimes are resiliant to kubelet loosing connection over GRPC CRI calls.
 - CRI Runtimes Integrations are not known to checkpoint using etcd and thus are not directly affected by etcd at the node.  
 

* **What are other known failure modes?**
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

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- First version with v1alpha1 was released in k8s 1.5.  See https://github.com/kubernetes/community/blob/ee783a18a34ef16da07f8d16d42782a6f78a9253/contributors/devel/sig-node/container-runtime-interface.md
- v1alpha was released with k8s 1.10. See https://github.com/kubernetes/kubernetes/pull/58973
- v1 proto was introduced in k8s 1.20. See https://github.com/kubernetes/kubernetes/pull/96387
