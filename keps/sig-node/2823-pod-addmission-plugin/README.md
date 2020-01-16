# KEP-2823: Add node-level plugin support for pod admission handler

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
  - [Design](#design)
    - [Configuration file](#configuration-file)
    - [Feature gate](#feature-gate)
    - [Kubelet to pod admission plugin communication](#kubelet-to-pod-admission-plugin-communication)
    - [Implementation detail](#implementation-detail)
    - [Handling of rejected pods](#handling-of-rejected-pods)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir
  in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and
  SIG Testing input (including test refactors)
    - [ ] e2e Tests for all Beta API Operations (endpoints)
    - [ ] (R) Ensure GA e2e tests for meet requirements
      for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
    - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
    - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806)
      must be hit
      by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
  publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to
  mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Today, kubelet is responsible for determining if a Pod can execute on the node.
Kubelet compares the required capabilities of the Pod against the discovered
capabilities of both the worker node and the container runtime.

Kubelet will reject the Pod if any required capabilities in the Pod.Spec are not
supported by the container engine running on the node. Such capabilities might
include the ability to set sysctl parameters, use of elevated system privileges
or use of a non-default process mount. Likewise, kubelet checks the Pod against
node capabilities; for example, the presence of a specific apparmor profile or
host kernel.

These validations represent final, last-minute checks immediately before the Pod
is started by the container runtime. These node-local checks differ from
API-layer validations like Pod Security Policies or Validating Admission
webhooks. Whereas the latter may be deactivated or removed by Kubernetes cluster
administrators, the former node-local checks cannot be disabled.

It is not currently possible to add additional validations before admitting the
Pod. This document proposes a framework for enabling additional node-local Pod
admission checks.

## Motivation

Amazon Elastic Kubernetes Service (EKS) provides users a managed Kubernetes
control plane. EKS users are provisioned a Kubernetes cluster running on AWS
cloud infrastructure. While the EKS user does not have host-level administrative
access to the master nodes, it is important to point out that they do have
administrative rights on that Kubernetes cluster.

The EKS user’s worker node administrative access depends on the type of worker
node the EKS user chooses. EKS users have three options. The first option is to
bring their own EC2 instances as worker nodes. The second option is for EKS
users to launch a managed worker node group. These first two options both result
in the EKS user maintaining full host-level administrative rights on the worker
nodes. The final option — the option that motivated this proposal — is for the
EKS user to forego worker node management entirely using AWS Fargate, a
serverless computing environment. With AWS Fargate, the EKS user does not have
host-level administrative access to their worker node; in fact, the worker node
runs on a serverless computing platform that abstracts away the entire notion of
a host.

In building the AWS EKS support for AWS Fargate, the AWS Kubernetes engineering
team faced a dilemma: how could they prevent Pods destined to run on Fargate
nodes from using host networking or assuming elevated host user privileges? This
is required to restrict pods which are not supported on these worker nodes and
also to prevent pods from accessing some portion of file system on the node or
tampering other process on the host.

The team initially investigated using a webhook or Pod Security Policy (PSP)
that would prevent Pods with a Fargate scheduler type from having an elevated
security context or using host networking. However, because the EKS user has
administrative rights on the Kubernetes cluster, API-layer constructs such as a
Pod Security Policy or validation webhook may be deleted, which would
effectively disable the effect of that PSP. Likewise, the second solution the
team landed on — using Node taints and tolerations — was similarly bound to the
Kubernetes API layer, which meant EKS users could modify those Node taints and
tolerations, effectively disabling the effects. A third potential solution
involving OCI hooks was then investigated. OCI hooks are separate executables
that an OCI-compatible container runtime invokes that can modify the behaviour
of the containers in a sandbox. While this solution would have solved the
API-layer problem, it introduced other issues, such as the inefficiency of
downloading the container image to the Node before the OCI hook was run. Since
pods can be scheduled directly on the node using nodename parameter in pod spec,
its required to add such validation in Kubelet itself.

The final solution the EKS team settled on involved changing kubelet itself to
support additional node-local Pod admission checks. This KEP outlines the EKS
team’s approach and proposes upstreaming these changes to kubelet in order to
allow extensible node-local last-minute validation checks. This functionality
will enable cloud providers who support nodeless/serverless worker nodes to
restrict Pods based on fields other than those already being validated by
kubelet.

Also, Kubernetes scheduler is extensible but isn’t omniscient about everything
on the node hence there can be cases where a scheduling decision cannot be
fulfilled by the node based on the current node conditions such as node-level
resource allocation. The in-tree PodAdmitHandlers mentioned below are good
examples for this.

* Eviction manager rejects pods if the node is under high pressure (e.g., PIDs)
* Topology manager rejects pods if the NUMA affinity cannot be satisfied

There can be additional resource allocation conditions and policies handled
externally to the kubelet, e.g.,

* Alternative resource allocator implemented by the CRI which needs to reject
  pods in conditions similar to the built-in resource managers.

These can also benefit from a pluggable admission handler to reject pods before
they are admitted by the kubelet. This is hard to be replicated by exposing more
information to the scheduler which may be too costly and subject to race
conditions.

### Goals

- Allow deployers of fully managed worker nodes to have control over Pods
  running on those nodes.

- Enable node-local Pod admission checks without requiring changes to kubelet.

### Non-Goals

- Move existing validations to “out of tree” plugins.

- Change existing API-layer validation solutions such as Pod Security Policies
  and validating admission webhooks.

## Proposal

The approach taken is similar to the container networking interface (CNI) plugin
architecture. With CNI, kubelet invokes one or more CNI plugin binaries on the
host to set up a Pod’s networking. kubelet discovers available CNI plugins
by [examining](https://github.com/kubernetes/kubernetes/blob/dd5272b76f07bea60628af0bb793f3cca385bf5e/pkg/kubelet/dockershim/docker_service.go#L242)
a well-known directory (`/etc/cni/net.d`) for configuration files
and [loading](https://github.com/kubernetes/kubernetes/blob/dd5272b76f07bea60628af0bb793f3cca385bf5e/pkg/kubelet/dockershim/docker_service.go#L248)
plugin [descriptors](https://github.com/kubernetes/kubernetes/blob/f4db8212be53c69a27d893d6a4111422fbce8008/pkg/kubelet/dockershim/network/plugins.go#L52)
upon startup.

To support pluggable validation for pod admission on the worker node, we propose
to have kubelet similarly discover node-local Pod admission plugins listed in a
new PodAdmissionPluginDir flag.

Other option is to enable this functionality through feature flag
“enablePodAdmissionPlugin” and have the directory path defined inside the
kubelet itself.

### User Stories (Optional)

#### Story 1

#### Story 2

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

These out-of-tree plugins should be provisioned and managed along with kubelet.

## Design Details

### Design

#### Configuration file

Node-local Pod admission plugins will be listed in a configuration file. The
Plugins field indicates the list of plugins to invoke before admitting the Pod.

```
{
  "name": "admission-plugin",
  "version": "0.1",
  "plugins": [
    {
      "name": "sysctlcheck",
      "type": "shell"
    },
    {
      "name": "fargatecheck",
      "type": "shell"
    }
  ]
}
```

A node-local Pod admission plugin has the following structure:

```

// PluginType indicates type of the admission plugin
type PluginType string

const (
    PluginTypeShell  PluginType = "shell"  // binary to execute.
    PluginTypeGRPC   PluginType = "grpc"   // Local port on the host.
    PluginTypeSocket PluginType = "socket" // fd to connect to.
)

// AdmissionPluginManager is the podAdmitHandler shim for external plugins.
type AdmissionPluginManager struct {
    confDir      string
    binDir       string
    pluginConfig *PluginConfig
}

// PluginConfig represents the plugin configuration file
type PluginConfig struct {
    Name         string              `json:"name,omitempty"`
    Version      string              `json:"version,omitempty"`
    Plugins      []*AdmissionPlugin  `json:"plugins"`
}

// AdmissionPlugin represents individual plugins specified in the configuration.
type AdmissionPlugin struct {
    Name     string     `json:"name"`
    Type     PluginType `json:"type"`
}

```

#### Feature gate

This functionality adds a new feature gate named “PodAdmissionPlugin” which
decides whether to invoke admission plugin or not.

#### Kubelet to pod admission plugin communication

Kubelet will encode the pod spec and invoke each admission plugin. After
decoding the pod spec, plugins can perform additional validations and return the
encoded form of the struct mentioned below to kubelet to decide whether to admit
the pod or not. For shell plugin type, request will be sent over stdin and
response will be received over stdout and stderr.

```
AdmitResult { 
   Admit bool 
   Reason string 
   Message string 
}
```

#### Implementation detail

As part of this implementation, new sub package will be added to
pkg/kubelet/lifecycle. In-tree admission handler shim will be included in this
package, which will be responsible for discovering and invoking the pod
admission plugins available on the host to decide on whether to admit the pod or
not.

If the plugin does not respond or if it's crashing, then Kubelet will fail-close
and pod will not be accepted by the kubelet. Because if kubelet ignores the
response and schedules the pod then intention of not executing pod with specific
needs on these worker nodes will be violated. These plugins doesn't mutate the
Pod object and can be invoked in parallel since there is no dependency between
them.

These validations will be applied to both static pods which are managed by
Kubelet and pods scheduled by control plane.

#### Handling of rejected pods

The pod will enter a Terminated state and by default won’t be deleted or
rescheduled but this behavior is consistent with in-tree admission handler
rejections and it’s well documented (e.g.,
in [topology-manager-docs](https://kubernetes.io/docs/tasks/administer-cluster/topology-manager/#policy-single-numa-node)).

One benefit of associating this rejection with a specific Reason provided by the
plugins (and in-tree handlers) is that the cluster admin can choose to run an
operator (e.g., the descheduler) that watches over pod rejections and delete the
pods based on the Reason field.

### Test Plan

### Graduation Criteria

### Upgrade / Downgrade Strategy

### Version Skew Strategy

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: PluggablePodAdmitHandler
    - Components depending on the feature gate: Kubelet

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes

###### What happens if we reenable the feature if it was previously rolled back?

Kubelet should start invoking the out-of-tree plugin for pod admission
validation

###### Are there any tests for feature enablement/disablement?

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

###### How can someone using this feature know that it is working for their instance?

- [X] Events
    - Event Reason: Pod admit handlers should emit events if pod cannot be
      placed on the node.
- [ ] API .status
    - Condition name:
    - Other field:
- [ ] Other (treat as last resort)
    - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
- [ ] Other (treat as last resort)
    - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

### Scalability

###### Will enabling / using this feature result in any new API calls?

###### Will enabling / using this feature result in introducing new API types?

###### Will enabling / using this feature result in any new calls to the cloud provider?

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2021-07-06: Initial KEP sent out for initial review, including Summary,
  Motivation and Proposal

## Drawbacks

## Alternatives

Why we didn’t go over CRI shim or using OCI hook approach?

1. Even before Kubelet invokes
   container [runtime](https://github.com/kubernetes/kubernetes/blob/v1.14.6/pkg/kubelet/kubelet.go#L1665)
   it sets up few things for pod,
   including [cgroup](https://github.com/kubernetes/kubernetes/blob/v1.14.6/pkg/kubelet/kubelet.go#L1605)
   , volume
   mount, [pulling secrets](https://github.com/kubernetes/kubernetes/blob/v1.14.6/pkg/kubelet/kubelet.go#L1644)
   for pods etc.

2. OCI hook is invoked just before running the container, therefore Kubelet
   would have already downloaded the image as well. Even if hook rejects the Pod
   object, there is no good way to emit events on why hook failed Pod creation.

## Infrastructure Needed (Optional)

