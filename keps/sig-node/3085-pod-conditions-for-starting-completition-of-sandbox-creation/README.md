<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-3085: Pod Conditions for Starting and Completion of Sandbox Creation

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
    - [User Stories For Consuming PodHasNetwork Condition](#user-stories-for-consuming-podhasnetwork-condition)
      - [Story 1: Consuming PodHasNetwork Condition Per Pod In A Monitoring Service](#story-1-consuming-podhasnetwork-condition-per-pod-in-a-monitoring-service)
      - [Story 2: Consuming PodHasNetwork Condition In A Controller](#story-2-consuming-podhasnetwork-condition-in-a-controller)
    - [PodHasNetwork Condition Fields In Different User Scenarios](#podhasnetwork-condition-fields-in-different-user-scenarios)
      - [Scenario 1: Stateless pod scheduled on a healthy node and cluster](#scenario-1-stateless-pod-scheduled-on-a-healthy-node-and-cluster)
      - [Scenario 2: Pods with startup delays due to problems with CSI, CNI or Runtime Handler plugins](#scenario-2-pods-with-startup-delays-due-to-problems-with-csi-cni-or-runtime-handler-plugins)
      - [Story 3: Pod unable to start due to problems with CSI, CNI or Runtime Handler plugins](#story-3-pod-unable-to-start-due-to-problems-with-csi-cni-or-runtime-handler-plugins)
      - [Story 4: Pod Sandbox restart after a successful initial startup and crash](#story-4-pod-sandbox-restart-after-a-successful-initial-startup-and-crash)
      - [Story 5: Graceful pod sandbox termination](#story-5-graceful-pod-sandbox-termination)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Determining status of sandbox creation for a pod](#determining-status-of-sandbox-creation-for-a-pod)
  - [PodHasNetwork condition details](#podhasnetwork-condition-details)
  - [Enhancements in Kubelet Status Manager](#enhancements-in-kubelet-status-manager)
  - [Unavailability of API Server or etcd along with Kubelet Restart](#unavailability-of-api-server-or-etcd-along-with-kubelet-restart)
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
  - [Dedicated fields or annotations for the pod sandbox creation timestamps](#dedicated-fields-or-annotations-for-the-pod-sandbox-creation-timestamps)
  - [Surface pod sandbox creation latency instead of timestamps](#surface-pod-sandbox-creation-latency-instead-of-timestamps)
  - [Report sandbox creation latency as an aggregated metric](#report-sandbox-creation-latency-as-an-aggregated-metric)
  - [Report sandbox creation stages using Kubelet tracing](#report-sandbox-creation-stages-using-kubelet-tracing)
  - [Have CSI/CNI/CRI plugins mark their start and completion timestamps while setting up their respective portions for a pod](#have-csicnicri-plugins-mark-their-start-and-completion-timestamps-while-setting-up-their-respective-portions-for-a-pod)
  - [Use a dedicated service between Kubelet and CRI runtime to mark sandbox ready condition on a pod](#use-a-dedicated-service-between-kubelet-and-cri-runtime-to-mark-sandbox-ready-condition-on-a-pod)
  - [Have Kubelet mark sandbox ready condition on a pod using extended conditions](#have-kubelet-mark-sandbox-ready-condition-on-a-pod-using-extended-conditions)
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
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
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

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

Pod sandbox creation is a critical phase of a pod's lifecycle that the kubelet
orchestrates across multiple components: in-tree volume plugins (ConfigMap,
Secret, EmptyDir, etc), CSI plugins and container runtime (which in turn invokes
a runtime handler and CNI plugins). Completion of all these phases is marked by
the CRI runtime reporting a pod sandbox with networking configured. This KEP
proposes a `PodHasNetwork` pod condition in pod status to indicate successful
completion of pod sandbox creation (with networking configured) by Kubelet and
the CRI container runtime. The `PodHasNetwork` condition will mark an important
milestone in the pod's lifecycle similar to `ContainersReady` and the overall
`Ready` conditions in pod status today. In the future, a dedicated
`PodHasVolumes` condition may be considered to mark successful mounting and
setup of all volumes in the pod spec. An alternate name like `SandboxReady` is
avoided since Kubernetes does not directly surface low level sandbox related
concepts to all users.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Today, the scheduler surfaces a specific pod condition: `PodScheduled` that
clearly identifies whether a pod got scheduled by the scheduler and when
scheduling completed. However, no specific conditions around initialization of
successfully scheduled pods from the perspective of completion of pod sandbox
creation (marked by the presence of a pod sandbox with networking configured) is
surfaced to cluster administrators in a scoped and consumable fashion.

There is an existing pod condition: `Initialized` that tracks execution of init
containers. For pods without init containers, the `Initialized` condition is set
when the Kubelet starts to process a pod before any sandbox creation activities
start. For pods with init containers, the `Initialized` condition is set when
init containers have been pulled and executed to completion. Therefore, the
existing `Initialized` condition is insufficient and inaccurate for tracking
completion of sandbox creation of all pods in a cluster. This distinction
becomes especially relevant in multi-tenant clusters where individual tenants
own the pod specs (including the set of init containers) while the cluster
administrators are in charge of storage plugins, networking plugins and
container runtime handlers.

Conclusion of the creation of the pod sandbox is marked by the presence of a
sandbox with networking configured. A new dedicated condition marking this -
`PodHasNetwork` - will benefit cluster operators (especially of multi-tenant
clusters) who are responsible for configuration and operational aspects of the
various components that play a role in pod sandbox creation: CSI plugins, CRI
runtime and associated runtime handlers, CNI plugins, etc. The duration between
`lastTransitionTime` field of the `PodHasNetwork` condition (with `status` set
to `true` for a pod for the first time) and the existing `PodScheduled`
condition will allow metrics collection services to compute total latency of all
the components involved in pod sandbox creation as an SLI. Cluster operators can
use this to publish SLOs around pod initialization to their customers who launch
workloads on the cluster.

Custom pod controllers/operators can use a dedicated condition indicating
completion of pod sandbox creation to make better decisions around how to
reconcile a pod failing to become ready. As a specific example, a custom
controller for managing pods that refer to PVCs associated with node local
storage (e.g. Rook-Ceph) may decide to recreate PVCs (based on a specified PVC
template in the custom resource the controller is managing) if the sandbox
creation is repeatedly failing to complete, indicated by the new `PodHasNetwork`
condition reporting `false`. Such a controller can leave PVCs intact and only
recreate pods if sandbox creation completes successfully (indicated by the new
`PodHasNetwork` condition reporting `true`) but the pod's containers fail to
become ready. Further details of this is covered in a [user story](#story-2-consuming-podhasnetwork-condition-in-a-controller) below.

When a pod's sandbox no longer exists, the `status` of `PodHasNetwork` condition
will be set to `false`. The duration between a pod's `DeletionTimeStamp` and
subsequent `lastTransitionTime` of `PodHasNetwork` condition (with `status` set
to `false`) will indicate the latency of pod termination. This can also be
surfaced by metrics collection services as a SLI. Note that surfacing any
dedicated conditions around termination of pod sandbox is unnecessary and beyond
the scope of this KEP.

Individual container creation (including pulling images from a registry) takes
place after the successful completion of pod sandbox creation. Updates to pod
container status to report latencies associated with creation of individual
containers within a pod is beyond the scope of this KEP.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
- Surface a new pod condition `PodHasNetwork` to indicate the successful
completion of pod sandbox creation (that concludes with configuration of
networking for the pod) from Kubelet.
- Describe how the new pod condition can be consumed by external services to
determine state and duration of pod sandbox creation (that concludes with
successful configuration of networking for the pod).

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
- Modify the meaning of the existing `Initialized` condition
- Specify metrics collection based on the conditions around pod sandbox creation
- Specify additional conditions (beyond `PodHasNetwork` with `status` set to `false`) to indicate sandbox (and networking) teardown
- Surface beginning and completion of creation of individual containers
- Surface details around any intermediate networking configuration phases of the pod

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

This KEP proposes enhancements to the Kubelet to report the completion of pod
sandbox creation (that concludes with successful configuration of networking for
the pod) as a pod condition with type: `PodHasNetwork`. Metric collection
and monitoring services can use the fields associated with the `PodHasNetwork`
condition to report sandbox creation state and latency either at a per-pod
cardinality or aggregate the data based on various properties of the pod: number
of volumes, storage class of PVCs, runtime class, custom annotations for CNI and
IPAM plugins, arbitrary labels and annotations on pods, etc. Certain pod
controllers can use the pod sandbox conditions to determine an optimal
reconciliation strategy for pods and associated resources (like PVCs).

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### User Stories For Consuming PodHasNetwork Condition

Surfacing the completion of pod sandbox creation (that concludes with successful
configuration of networking for the pod) as a new pod condition -
`PodHasNetwork` - in pod status can be consumed in different ways:

##### Story 1: Consuming PodHasNetwork Condition Per Pod In A Monitoring Service

A cluster operator may already depend on a service like [Kube State
Metrics](https://github.com/kubernetes/kube-state-metrics) for monitoring the
state of their Kubernetes clusters. The cluster operator may want such a service
to surface pod sandbox creation state and latency at a granular level for each
pod (due to the ambiguity around `Initialized` state as described earlier). For
this story, we are assuming the service has been enhanced to [1] consume the new
`PodHasNetwork` pod condition as described in this KEP and [2] implement
informers and state to distinguish between the first time a pod sandbox becomes
ready (that concludes with successful configuration of networking for the pod)
and a subsequent instance of sandbox becoming ready (after sandbox destruction)
over the lifetime of a pod.

The operator can use PromQL queries to aggregate and analyze data (around pod
sandbox creation) based on custom pod labels and annotations (already surfaced
by a service like Kube State Metrics) indicating specific workload types across
different namespaces. For example, annotations and labels could be used to
differentiate pod sandbox creation state and latencies for "sensitive database"
workloads, "sensitive analysis" workloads and "untrusted build" workloads each
of which maps to pods mounting PVCs from different storage classes (depending on
the level of encryption desired), using a specific runtime class (depending on
the level of isolation desired - microvm vs runc based) and specific IPAM
characteristics around reachability of the pods. Access to the pod labels and
annotations along with the sandbox latency data at a per-pod cardinality is
essential to enable the aggregation based on factors that have special/custom
meaning for the operator's cluster and tenants. The values associated with such
labels and annotations may not map to distinct namespaces, existing pod fields
or other API object fields in a Kubernetes cluster.

Depending on the metrics and monitoring pipeline, as the cluster scales up,
cardinality of data at a per pod level (surfaced from a service like Kube State
Metrics) may lead to excessive load on the monitoring backend like Prometheus.
At such a point, the cluster operator may decide to create and deploy their own
custom monitoring service that uses a pod informer and aggregates (based on
custom pod labels and annotations) state and latency of pod sandbox creation
into a histogram which is ultimately reported to Prometheus. As with the
previous approach, access to the pod labels and annotations and the sandbox
latency data at a per-pod cardinality is essential to enable the aggregation
based on factors that have special/custom meaning for the operator's cluster and
tenants and may not map to distinct namespaces pod fields or other API object
fields in the cluster.

The data from the above monitoring services can be used as SLIs with associated
SLOs configured around sandbox creation state and latency (besides other metrics
like scheduling latency) for each specific workload type depending on specific
user requirements such as: desired encryption of persistent data (if any),
runtime isolation and network reachability (governed by different IPAM plugins).

##### Story 2: Consuming PodHasNetwork Condition In A Controller

A controller managing a set of pods along with associated resources like
networking configuration, storage or arbitrary dynamic resources (in the future)
can evaluate the `PodHasNetwork` condition to optimize the set of actions it
executes when bringing up pods and encountering failures. Depending on whether
the pod sandbox is ready (with networking configured), the controller may decide
to destroy and re-create the associated resources that are required for the
sandbox creation to complete or simply try to re-create the pod while keeping
the resources intact.

A specific example of the above would be a controller for stateful application
pods that mount PVCs that bind to node local PVs. Let's assume the stateful
application has built-in data replication capabilities and the controller
supports PVC templates to dynamically generate PVCs. When trying to bring up
fresh pods (after earlier pods got terminated), there could be a problem with
the CSI plugin that mounts the node local PV into the pod. In such a situation,
the sandbox creation will not complete. Based on the `PodHasNetwork` condition,
the controller may decided to create a fresh PVC. If sandbox creation does
complete successfully (marked by `PodHasNetwork` reporting true) but the pod
fails to enter a Ready state, the controller will retain the PVC (to avoid any
data replication) and only try to recreate the pod. Having access to the new
`PodHasNetwork` condition allows the controller to optimize it's reconciliation
strategy and realize the desired state more efficiently.

#### PodHasNetwork Condition Fields In Different User Scenarios

In each of the scenarios below, nearly identical `PodHasNetwork` conditions that
would result from different scenarios/problems are grouped together. The unique
scenarios are detailed after describing the values associated with the fields of
the `PodHasNetwork` condition. To make each scenario concrete, a specific set of
timestamps in the future is chosen. The `PodScheduled` condition is mentioned in
the stories but conditions after pod sandbox creation (e.g. `Initialized` and
`Ready`) are skipped. A service monitoring latency of initial pod sandbox
creation is assumed to implement a pod informer and appropriate state to
distinguish between the first time a pod sandbox becomes ready versus a
subsequent instance of readiness over the lifetime of the pod.

##### Scenario 1: Stateless pod scheduled on a healthy node and cluster

A user launches a simple, stateless runc based pod with no init containers in a
healthy cluster. The pod gets successfully scheduled at 2022-12-06T15:33:46Z and
pod sandbox is ready after three seconds at 2022-12-06T15:33:49Z.

The pod will report the following conditions in pod status at
2022-12-06T15:33:47Z (right after Kubelet worker starts processing the pod):
```
status:
  conditions:
  ...
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T15:33:47Z"
    status: "False"
    type: PodHasNetwork
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T15:33:46Z"
    status: "True"
    type: PodScheduled
```

The pod will report the following conditions in pod status at
2022-12-06T15:33:50Z (after pod sandbox creation is complete, marked by
successful configuration of pod networking):
```
status:
  conditions:
  ...
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T15:33:49Z"
    status: "True"
    type: PodHasNetwork
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T15:33:46Z"
    status: "True"
    type: PodScheduled
```

A service monitoring latency of initial pod sandbox creation will record a
latency of three seconds in this scenario based on the delta between
`lastTransitionTime` timestamp associated with `PodHasNetwork` and
`PodScheduled` conditions.

##### Scenario 2: Pods with startup delays due to problems with CSI, CNI or Runtime Handler plugins

In each of the scenarios under this section, problems or delays with
infrastructural plugins like CSI/CNI/CRI result in a ten second delay for pod
sandbox creation (marked by successful configuration of pod networking) to
complete. In each scenario, the pod gets successfully scheduled at
2022-12-06T15:33:46Z, pod sandbox is ready after ten seconds at
2022-12-06T15:33:56Z.

For each scenario below, the pod will report the following conditions in pod
status at 2022-12-06T15:33:47Z (right after Kubelet worker starts processing the
pod and the pod sandbox creation has started but not complete):
```
status:
  conditions:
  ...
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T15:33:47Z"
    status: "False"
    type: PodHasNetwork
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T15:33:46Z"
    status: "True"
    type: PodScheduled
```

For each scenario, the pod will report the following conditions in pod status at
2022-12-06T15:34:00Z (after pod sandbox is ready - marked by
successful configuration of pod networking - after ten seconds):
```
status:
  conditions:
  ...
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T15:33:56Z"
    status: "True"
    type: PodHasNetwork
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T15:33:46Z"
    status: "True"
    type: PodScheduled
```

A service monitoring duration of pod sandbox creation (marked by
successful configuration of pod networking) will record a latency of
ten seconds in these scenarios based on the delta between `lastTransitionTime`
timestamps associated with `PodHasNetwork` and `PodScheduled` conditions with
`status` set to `true`. For each observation associated with a scenario below,
the monitoring service also associates a label with the metric indicating
RuntimeClass of the pods and StorageClass of PVCs referred by the pod. This
enables further grouping of the data during analysis.

A cluster-wide SLO around initial pod sandbox creation latencies configured with
a threshold of less than ten seconds will record a breach in these scenarios.
Further analysis of the metrics based on labels indicating RuntimeClass of the
pods and StorageClass of PVCs referred by the pod will enable the cluster
administrators to isolate the cause of the breaches to specific infrastructure
plugins as detailed below.

###### Stateful pod encountering sandbox creation delays from attaching a PV backed by a CSI plugin

A Stateful pod refers to a PVC bound to a PV backed by a CSI plugin. After the
pod is scheduled on a node, the CSI plugin runs into problems in the storage
control plane when trying to attach the PV to the node. This results in several
retries that ultimately succeeds after nine seconds.

###### Stateless pod encountering sandbox creation delays from allocating IP from a CNI/IPAM plugin

A pod is scheduled on a node in an experimental pre-production cluster where the
operator has configured a new CNI plugin using a centralized IP allocation
mechanism. Due to a spike of load in the IP allocation service, the CNI plugin
times out several times but ultimately succeeds getting an IP address and
configuring the pod network after nine seconds.

###### Stateless pod encountering sandbox creation delays from microvm based sandbox initialization

A pod configured with a special microvm based runtime class is scheduled on a
node. The runtimeclass handler encounters crashes in the guest kernel multiple
times but ultimately initializes the virtual machine based sandbox environment
successfully after nine seconds.

##### Story 3: Pod unable to start due to problems with CSI, CNI or Runtime Handler plugins

In each of the scenarios under this section, problems or delays with
infrastructural plugins like CSI/CNI/CRI result in pod sandbox creation never
completing. In each scenario, the pod gets successfully scheduled at
2022-12-06T15:33:46Z, but pod sandbox creation runs into problems that do not
eventually resolve and results in repeated failures as kubelet tries to start
the pod.

For each scenario below, the pod will report the following conditions in pod
status at all times after 2022-12-06T15:33:47Z (after pod sandbox creation
started until the pod is deleted manually or by a controller):
```
status:
  conditions:
  ...
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T15:33:47Z"
    status: "False"
    type: PodHasNetwork
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T15:33:46Z"
    status: "True"
    type: PodScheduled
```

A service monitoring state of pod sandbox creation will record a metric
indicating failure to create pod sandbox beyond a configured duration.

A cluster-wide SLO around success rate of pod sandbox creation may record a
breach due to the pod sandbox creation failures. Further analysis of the metrics
aggregated based on labels (associated with the metrics) indicating RuntimeClass
of the pods and StorageClass of PVCs referred by the pod will enable the cluster
administrators to associate the failures to specific infrastructure plugins as
detailed below.

###### Stateful pod encountering sandbox creation failures when attaching a PV backed by a CSI plugin

A Stateful pod refers to a PVC bound to a PV backed by a CSI plugin. After the
pod is scheduled on a node, the CSI plugin runs into problems in the storage
control plane when trying to attach the PV to the node. The failure to attach
never resolves thus blocking pod sandbox creation.

###### Stateless pod encountering sandbox creation failures when allocating IP from a CNI/IPAM plugin

A pod is scheduled on a node in an experimental pre-production cluster where the
operator has configured a new CNI plugin using a centralized IP allocation
mechanism. Due to problems in the IP allocation service, the CNI plugin fails to
get an IP address and is unable to configure the pod network. This blocks pod
sandbox creation.

###### Stateless pod encountering sandbox creation failures from microvm based sandbox initialization

A pod configured with a special microvm based runtime class is scheduled on a
node. The runtimeclass handler encounters crashes in the guest kernel repeatedly
and is unable to initialize the virtual machine based sandbox environment.

##### Story 4: Pod Sandbox restart after a successful initial startup and crash

In each of the scenarios under this section, a pod sandbox is successfully
created but eventually gets destroyed due to problems in the host or the sandbox
environment. As a result, the pod sandbox has to be re-created (and pod
networking reconfigured) by Kubelet in coordination with CRI runtime. In each
scenario, the pod is successfully scheduled at 2022-12-06T15:33:46Z and pod
sandbox is ready after 5 seconds. The sandbox is destroyed after two hours.
Re-creation of the sandbox runs into problems but eventually succeed after nine
seconds.

The pod will report the following conditions in pod status at
2022-12-06T15:34:00Z (few seconds after initial pod sandbox is ready):
```
status:
  conditions:
  ...
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T15:33:52Z"
    status: "True"
    type: PodHasNetwork
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T15:33:46Z"
    status: "True"
    type: PodScheduled
```

The pod will report the following conditions in pod status at
2022-12-06T17:33:46Z (right after pod sandbox is destroyed):
```
status:
  conditions:
  ...
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T17:33:46Z"
    status: "False"
    type: PodHasNetwork
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T15:33:46Z"
    status: "True"
    type: PodScheduled
```

The pod will report the following conditions in pod status at
2022-12-06T17:34:00Z (few seconds after the new pod sandbox is ready):
```
status:
  conditions:
  ...
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T17:33:52Z"
    status: "True"
    type: PodHasNetwork
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T15:33:46Z"
    status: "True"
    type: PodScheduled
```

A service monitoring restarts associated with successfully created pod sandboxes
will record a restart in these scenarios. A service measuring initial pod
sandbox creation latency will need to implement logic (for example, using pod
informers and state) to differentiate the initial pod sandbox creation
from the latter pod sandbox creations resulting from node crashes/reboots or
sandbox crashes.

###### Node crash

A regular runc based pod is scheduled on a node whose kernel crashes after two
hours of the pod sandbox getting created successfully. The node restarts quickly
(resulting in no pod evictions) and kubelet has to re-create the pod sandbox.

###### Sandbox crash

A pod is configured with a microvm based runtime handler. The virtual machine
sandbox is created successfully but suffers a crash due to problems with the
guest kernel after two hours of the pod creation. As a result, kubelet has to
re-create the pod sandbox (and reconfigure pod networking).

##### Story 5: Graceful pod sandbox termination

A user launches a pod that runs successfully but eventually deleted by a
controller after several hours. The pod was scheduled at 2022-12-06T12:33:46Z
and the sandbox became ready at 2022-12-06T12:33:48Z. The delete request is
invoked at 2022-12-06T15:33:47Z and the pod is terminated by Kubelet at
2022-12-06T15:33:49Z

The pod will report the following conditions in pod status at
2022-12-06T15:33:46Z (right before the pod delete request is invoked):
```
status:
  conditions:
  ...
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T12:33:48Z"
    status: "True"
    type: PodHasNetwork
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T12:33:46Z"
    status: "True"
    type: PodScheduled
```

The pod will report the following conditions in pod status at
2022-12-06T15:33:49Z (right after the pod termination has been processed by Kubelet but the pod is yet to be completely deleted from API server):
```
status:
  conditions:
  ...
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T15:33:49Z"
    status: "False"
    type: PodHasNetwork
  - lastProbeTime: null
    lastTransitionTime: "2022-12-06T12:33:46Z"
    status: "True"
    type: PodScheduled
```

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

A monitoring service measuring duration of initial sandbox creation of a pod
(based on successful network configuration of the pod)
should differentiate between the initial and subsequent sandbox creations (if
any due to node crash/sandbox crash) and track them separately.  This can be
achieved using a pod informer whose event handler stores (in a persistent store
or as custom annotations on the pod) the `lastTransitionTime` field for
`PodHasNetwork` condition observed when it had `status` = `true` for the first
time. Later, if the pod sandbox is recreated, the `lastTransitionTime` for the
pod sandbox creation conditions can be differentiated from the data associated
with initial sandbox creation based on whether the initial data exists (either
in the persistent store or pod annotations).

Measuring duration of sandbox creation accurately beyond the initial sandbox
creation (marked by successful network configuration of the pod) is not
possible with the `PodHasNetwork` condition alone. This is
similar to other ready conditions like `ContainersReady` and overall pod `Ready`
which gets updated after containers are restarted without a specific marker of
when the process of restarting the containers or brining the pod back into a
ready state began following an event like a node crash.

When deriving SLOs based on SLIs around state and duration of sandbox creation,
user error scenarios should be filtered. In the context of pod sandbox creation,
such errors can surface due to:
- References to a secret or configmap that does not exist and never gets
created. As a result, a pod referencing a missing secret or configmap will never
go past the volume initialization phase.
- References to a secret or configmap that get created at a point of time after
the pod gets scheduled. In such scenarios, volume initialization phase of the
pod will be stuck until the referenced secrets/configmaps are created in the
cluster.
The metric collection service that generates SLIs can filter pods affected by
the above situations by evaluating `FailedMount` pod events associated with the
pod and matching a regular expression of the form `MountVolume.SetUp failed for
volume "(secret|config-map) .*" : (secret|config-map) ".*" not found"`.


### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

The main risk associated with `PodHasNetwork` is any potential confusion with
the existing `Initialized` condition. Both the existing `Initialized` conditions
and the new pod sandbox conditions refer to distinct stages in a pod's overall
initialization. Documentation will help mitigate this risk.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

The Kubelet will set a new condition on a pod: `PodHasNetwork` to surface the
successful completion of sandbox creation for a pod (which is marked by the
successful configuration of pod networking). A new `PodConditionType`
corresponding to `PodHasNetwork` will be added in `api/core/v1/types.go`. No
changes are required in the Pod Status API for this enhancement.

### Determining status of sandbox creation for a pod

Today, `syncPod()` in Kubelet is invoked with the `kubecontainer.PodStatus`
(distinct from the `v1.PodStatus` API) associated with a given pod.
`podSandboxChanged()` in `kubeGenericRuntimeManager` evaluates the
`SandboxStatuses` field in `PodStatus` to determine whether a new pod sandbox
will need to be created (and networking configured) for a pod. The same logic
will be used to determine whether a sandbox (with networking configured) is
ready for a pod in the Kubelet status manager.

### PodHasNetwork condition details

Kubelet will initially generate the `PodHasNetwork` condition as part of
existing calls to `generateAPIPodStatus()` early during `syncPod()`. The
`status` field will be set to `true` if a sandbox is ready (determined by
invoking `podSandboxChanged()` as described
[above](#determining-status-of-sandbox-creation-for-a-pod)). The `status` field
will be set to `false` if a sandbox is found to be not ready.

Kubelet will generate the `PodHasNetwork` condition for the final time (in the
life of a pod) as part of existing calls to `generateAPIPodStatus()` early
during `syncTerminatedPod()`. Prior invocations of `killPod()` (as part of
`syncTerminatingPod`) will result in the absence of a sandbox corresponding to
the pod. As a result, the `status` field of the `PodHasNetwork` condition will
be set to `false` (determined by invoking `podSandboxChanged()` as described
[above](#determining-status-of-sandbox-creation-for-a-pod)).

During periods of API server or etcd unavailability combined with a Kubelet
restart/crash (covered in more details
[below](#unavailability-of-api-server-or-etcd-along-with-kubelet-restart)),
the `lastTransitionTime` field of `PodHasNetwork` condition that
ultimately gets persisted upon Kubelet restarting and API server becoming
available again is as close as possible to an actual change in the condition
(that could not be persisted).

Changes of the `status` field will result in `lastTransitionTime` field getting
updated (by the Kubelet Status Manager).

### Enhancements in Kubelet Status Manager

Today, the Kubelet Status Manager surfaces APIs for other Kubelet components to
issue pod status updates. It caches the pod status and issues patches to the API
server when necessary. This infrastructure will be used for managing the new pod
conditions as well.

The Kubelet Status Manager will surface a new `GeneratePodHasNetworkCondition`
API. This will be invoked by Kubelet's `generateAPIPodStatus()` to populate the
pod status that is passed to `setPodStatus`. This is similar to the existing pod
conditions generator functions: `GeneratePodReadyCondition` and
`GeneratePodInitializedCondition`. If updates through `generateAPIPodStatus()`
is found to be inaccurate (for example if Kubelet is very busy), invocation of
`GeneratePodHasNetworkCondition` could also be added right after `createSandbox`
in `kubeGenericRuntimeManager` returns successfully.

`updateStatusInternal()` in the Kubelet Status Manager will be enhanced to mark
`updateLastTransitionTime` for the new `PodHasNetwork` condition when changes in
the `status` of the conditions are detected.

### Unavailability of API Server or etcd along with Kubelet Restart

If pod sandbox creation completed successfully on a node but API server became
unavailable, the Kubelet status manager will retry issuing the patches to the
API server. However, the Kubelet may get restarted (or crash) while the API
server is unavailable with the pod status updates not yet persisted. In such a
situation (expected to be quite rare), the timestamp associated with the
`lastTransitionTime` field in the new conditions will not be accurate due to
inability to persist or cache them. The `lastTransitionTime` field will get
updated on subsequent `generateAPIPodStatus()` calls based on the state of the
CRI sandbox and the corresponding timestamps will be persisted. This aligns with
handling of other Kubelet managed conditions (ContainersReady, (Pod) Ready) when
API server is unavailable and Kubelet restarts resulting in the status manager
cache getting dropped.

### Test Plan

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

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.
All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.
[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

A review of the existing E2E tests reveal that coverage of the basic, existing
pod conditions (populated by Kubelet) is sparse. While the existing pod
conditions are quite mature, we will consider adding explicit validation of some
of the subtle aspects of current behavior around the pod conditions (e.g.
ensuring `Initialized` condition of a pod without init containers is set very
early for a pod that will never reach the sandbox creation state due to
missing volume dependencies and will thus never actually initialize).

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

New unit tests will be mainly scoped to the Kubelet `status` package that a bulk
of the enhancements above will target.

- `k8s.io/kubernetes/pkg/kubelet/status`: `June 13, 2022` - `82.2`
- `k8s.io/kubernetes/pkg/kubelet/kubelet`: `June 13, 2022` - `64.5`
- `k8s.io/kubernetes/pkg/kubelet/kubelet_pods.go`: `June 13, 2022` - `71.3`

Note above that while the `Kubelet` package overall has low coverage, the
changes in the context of this KEP is scoped to the `generateAPIPodStatus`
method which is a tiny portion of the overall `Kubelet` package in the
kubelet_pods.go file.

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

N/A. See notes about e2e tests below.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

E2E tests will be introduced to cover the user scenarios mentioned above. Tests
will involve launching pods with characteristics mentioned below and
examining the pod status has the new `PodHasNetwork` condition with `status` and `reason` fields populated with expected values:
1. A basic pod that launches successfully without any problems.
2. A pod with references to a configmap (as a volume) that has not been created causing the pod sandbox creation to not complete until the configmap is created later.
3. A pod whose node is rebooted leading to the sandbox being recreated.

Tests for pod conditions in the `GracefulNodeShutdown` e2e_node test will be
enhanced to check the status of the new pod sandbox conditions are `false` after
graceful termination of a pod.

Testing updates of Pod conditions in the Conformance Test `Pods, completes the
lifecycle of a Pod and the PodStatus` will be enhanced to cover resetting the
new pod sandbox conditions.

### Graduation Criteria

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

#### Alpha

- Kubelet will report pod sandbox conditions if the feature flag `PodHasNetworkCondition` is enabled.
- Initial e2e tests completed and enabled.

#### Beta

- Gather feedback from cluster operators and developers of services or controllers that consume these conditions.
- Implement suggestions from feedback as feasible.
- Feature Flag defaults to enabled.
- Add more test cases and link to this KEP.

#### GA

- All tests are passing with no known flakiness.
- All feedback addressed around the new pod sandbox conditions.
- No open decision items around the new pod sandbox conditions.
- Feature Flag removed.

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

The new condition will be managed by the Kubelet. When upgrading a node to a
version of the Kubelet that can set the new condition, new pods launched on
that node will surface the new condition. If Kubelet on the node is later
downgraded, there may remain evicted pods that are not deleted. Foe such pods, a
node with a version of the Kubelet that does not support the new condition will
continue to report pods associated with it with the new conditions.

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

The new condition will be managed by the Kubelet. Since the control plane
components are not involved, handling of version skew is not applicable.

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
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PodHasNetworkCondition
  - Components depending on the feature gate: Kubelet

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No changes to any default behavior should result from enabling the feature.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, the feature can be disabled once it has been enabled. However the new pod
sandbox condition will get persisted in pods and would continue to be reported after the feature is disabled until those pods are deleted.

###### What happens if we reenable the feature if it was previously rolled back?

New pods created since re-enablement will report the new pod sandbox condition.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

Unit tests (as outlined in the [Unit tests](#unit-tests) section above) will be
used to confirm that the new pod condition introduced is being:
- evaluated and applied by the Kubelet Status manager when the feature is enabled.
- not evaluated nor applied when the feature is disabled.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

Skipping this section at the Alpha stage and will populate at Beta.

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

Skipping this section at the Alpha stage and will populate at Beta.

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

No, this feature does not have any dependencies. Other metric oriented services
in the cluster may depend on this.

### Scalability

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

Yes, the new pod condition will result in the Kubelet Status Manager making additional PATCH calls on the pod status fields.

The Kubelet Status Manager already has infrastructure to cache pod status updates (including pod conditions) and issue the PATCH es in a batch.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

Slight increase (a few bytes) of the Pod API object due to persistence of the additional condition in the pod status.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

If etcd/API server is unavailable, pod status cannot be updated. So the
`PodHasNetwork` condition associated with pod status cannot be updated either.
The pod status manager already retries the API server requests later (based on
data cached in the Kubelet) and that should help.

If pod sandbox creation completes for a pod on a node but API server becomes
unavailable (before the sandbox creation condition can be patched) and Kubelet
crashes or restarts (shortly after API server becoming and staying unavailable),
the `lastTransitionTime` field may be inaccurate. This is described in the
section [above](#unavailability-of-api-server-or-etcd-along-with-kubelet-restart).

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

None so far

###### What steps should be taken if SLOs are not being met to determine the problem?

SLOs are not applicable to pod status fields. Overall Kubernetes node level SLOs
may leverage this feature.

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

The main drawback associated with the new pod sandbox conditions involves a
slight potential increase in calls to the API Server from Kubelet to patch
`status` = `true` for the new `PodHasNetwork` condition in a pod's status.
Typically, this would involve an extra patch call for pod status in the lifetime
of most pods (if the status manager does not batch them with other pod status
updates): one when pod sandbox creation completes and another when the pod is
terminated. However, there could be a higher number of patch calls to API Server
if the pod sandbox environment (like a microvm) starts successfully and then
crashes in a re-start loop.

Caching of updates to pod status by the pod status manager and batching pod
status updates (which is already in place) can help mitigate frequent patch
calls to API server.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Dedicated fields or annotations for the pod sandbox creation timestamps

Timestamps around completion of pod sandbox creation may be surfaced as a
dedicated field in the pod status rather than a pod condition. However, since
the successful creation of pod sandbox is essentially a "milestones" in the life
of a pod (similar to Scheduled, Ready, etc), pod conditions is the ideal place
to surface these and aligns well with the existing conditions like
`ContainersReady` and overall `Ready`.

A dedicated annotation on the pod for surfacing this data is another potential
approach. However, usage of annotations for Kubelet managed data is typically
discouraged.

### Surface pod sandbox creation latency instead of timestamps

Surfacing the amount of time it took to successfully create a pod sandbox is an
alternative to surfacing the condition around completion of pod
sandbox creation (whose delta from pod scheduled condition reflects the
latency). The latency data would surface the same information from a pod
initialization SLI perspective as mentioned in the Motivations section.
Implementing this approach would require an API change on the pod status to
surface the latency data (as this no longer fits the structure of a pod
condition). This data cannot be consumed by other controllers as mentioned in
User Stories section.

### Report sandbox creation latency as an aggregated metric

The duration it took pod sandbox to become ready can be directly reported as a
prometheus metrics aggregated in a histogram. However, aggregating the data at
the Kubelet level prevents a metric collection service from classifying the data
based on interesting fields on a pod (runtime class, storage class of PVCs,
number of PVCs, etc) or using custom labels and annotations on pods that
indicate workload characteristics (that the cluster operator may wish to use as
a basis for aggregating the metrics).

This also prevents other controllers from acting on sandbox status as
mentioned in User Stories section.

### Report sandbox creation stages using Kubelet tracing

The Kubelet is being instrumented to emit traces based on OpenTelemetry around
sandbox creation stages (as well several other parts of the pod lifecycle).

To implement the pod sandbox creation latency SLI/SLO use cases, the tracing
infrastructure needs to be able to:
- Collect all traces around CRI sandbox creation for all pods with no sampling.
- Look-up pod fields from API server (associated with a pod's trace) like
labels/annotations/storage classes of PVCs referred by the pod/runtimeclass/etc.
that is of interest to cluster operators and their users for classifying and
aggregating the metrics.
- Look-up a pod's Scheduled condition fields to determine the beginning of pod
sandbox creation.

Since the lookup of the pod fields and existing conditions is necessary for SLIs
around pod sandbox creation latency, surfacing the `PodHasNetwork` condition in
pod status will allow a metric collection service to directly access the
relevant data without requiring the ability to collect and parse OpenTelemetry
traces. As mentioned in the User Stories, popular community managed services
like Kube State Metrics can consume the `PodHasNetwork` condition with a trivial
set of changes. Enhancing them to collect and parse OpenTelemetry traces with no
sampling and mapping the data to associated data from API server data will be
complex from an engineering and operational perspective.

For controllers using the pod sandbox conditions to determine reconciliation
strategy, access to the pod is typically necessary while collecting and parsing
traces would be unusual.

### Have CSI/CNI/CRI plugins mark their start and completion timestamps while setting up their respective portions for a pod

Each infrastructural plugin that Kubelet calls out to (in the process of setting
up a pod sandbox) can mark start and completion timestamps on the pod as
conditions. This approach would be similar to how readiness gates work today.
However, CSI and CRI plugins will need to be enlightened about fields in a pod
(like status conditions) and setup a client to the API server (to update the
conditions) which they may not implement to stay orchestrator agnostic.

### Use a dedicated service between Kubelet and CRI runtime to mark sandbox ready condition on a pod

An on-host binary that runs as a service and proxies CRI API calls between the
CRI runtime and Kubelet can intercept the successful creation of a pod sandbox
in response to CRI `RunPodSandbox`. Next, using an API server client, the binary
can mark extended conditions on a pod to indicate state of sandbox creation.
While this approach works, without requiring any additional changes to Kubelet,
it had a couple of disadvantages: First, this approach requires configuration
and management of a separate proxy binary between Kubelet and CRI runtime in the
cluster nodes. Second, the proxy binary will need to replicate the logic in
Kubelet status manager to efficiently interact with the API server (as well as
cache the status and retry in case of API server outages) regarding updates to
pod sandbox status. Therefore isolating the logic around pod sandbox conditions
to a separate binary intercepting API calls between kubelet and the CRI runtime
is not preferred.

### Have Kubelet mark sandbox ready condition on a pod using extended conditions

Instead of a "native" condition as proposed in this KEP, an "extended" condition
maybe used by Kubelet to mark the PodHasNetwork condition. Such a condition may
look like: `kubernetes.io/pod-has-network`. However, internal/core Kubernetes
components (like Kubelet) do not use "extended" conditions today. So this
approach may be unusual.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
