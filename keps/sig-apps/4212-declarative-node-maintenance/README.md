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
# KEP-4212: Declarative Node Maintenance

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
  - [Cluster Autoscaler](#cluster-autoscaler)
  - [kubelet](#kubelet)
  - [Motivation Summary](#motivation-summary)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Kubectl](#kubectl)
  - [NodeMaintenance API](#nodemaintenance-api)
  - [NodeMaintenance Admission](#nodemaintenance-admission)
  - [NodeMaintenance Controller](#nodemaintenance-controller)
    - [Idle](#idle)
    - [Finalizers and Deletion of the NodeMaintenance](#finalizers-and-deletion-of-the-nodemaintenance)
    - [Cordon](#cordon)
    - [Uncordon (Complete)](#uncordon-complete)
    - [Drain](#drain)
    - [Pod Selection](#pod-selection)
    - [Pod Selection and DrainTargets Example](#pod-selection-and-draintargets-example)
      - [PodTypes and Label Selectors Progression](#podtypes-and-label-selectors-progression)
    - [Status](#status)
    - [Supported Stage Transitions](#supported-stage-transitions)
  - [DaemonSet Controller](#daemonset-controller)
  - [kubelet: Graceful Node Shutdown](#kubelet-graceful-node-shutdown)
  - [kubelet: Static Pods](#kubelet-static-pods)
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
  - [Out-of-tree Implementation](#out-of-tree-implementation)
  - [Use a Node Object Instead of Introducing a New NodeMaintenance API](#use-a-node-object-instead-of-introducing-a-new-nodemaintenance-api)
  - [Use Taint Based Eviction for Node Maintenance](#use-taint-based-eviction-for-node-maintenance)
  - [Names considered for the new API](#names-considered-for-the-new-api)
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

This KEP proposes adding a declarative API to manage node maintenance. This API can be used to
implement additional capabilities around node draining.

## Motivation

The goal of this KEP is to analyze and improve node maintenance in Kubernetes.

Node maintenance is a request from a cluster administrator to remove all pods from a node(s) so
that it can be disconnected from the cluster to perform a software upgrade (OS, kubelet, etc.),
hardware or firmware upgrade, or simply to remove the node as it is no longer needed.

Kubernetes has existing support for this use case in the following way with `kubectl drain`:
1. There are running pods on node A, some of which are protected with PodDisruptionBudgets (PDB).
2. Set the node `Unschedulable` (cordon) to prevent new pods from being scheduled there.
3. Evict (default behavior) pods from node A by using the eviction API (see [kubectl drain worklflow](https://raw.githubusercontent.com/kubernetes/website/f2ef324ac22e5d9378f2824af463777182817ca6/static/images/docs/kubectl_drain.svg)).
4. Proceed with the maintenance and shut down or restart the node.
5. On platforms and nodes that support it, the kubelet will try to detect the imminent shutdown and
   then attempt to perform a [Graceful Node Shutdown](https://kubernetes.io/docs/concepts/architecture/nodes/#graceful-node-shutdown):
   - delay the shutdown pending graceful termination of remaining pods
   - terminate remaining pods in reverse priority order (see [pod-priority-graceful-node-shutdown](https://kubernetes.io/docs/concepts/architecture/nodes/#pod-priority-graceful-node-shutdown))

The main problem is that the current approach tries to solve this in an application agnostic way
and will simply attempt to remove all the pods currently running on the node. Since this approach
cannot be applied generically to all pods, the Kubernetes project has defined special
[drain filters](https://github.com/kubernetes/kubernetes/blob/56cc5e77a10ba156694309d9b6159d4cd42598e1/staging/src/k8s.io/kubectl/pkg/drain/filters.go#L153-L162)
that either skip groups of pods or an admin has to consent to override those groups to be either
skipped or deleted. This means that without knowledge of all the underlying applications on the
cluster, the admin has to make a potentially harmful decision.

From an application owner or developer perspective, the only standard tool they have is
a PodDisruptionBudget. This is sufficient in a basic scenario with a simple multi-replica
application. The edge case applications, where this does not work are very important to
the cluster admin, as they can block the node drain. And, in turn, very important to the
application owner, as the admin can then override the pod disruption budget and disrupt their
sensitive application anyway.

List of cases where the current solution is not optimal:

1. Without extra manual effort, an application running with a single replica has to settle for
   experiencing application downtime during the node drain. They cannot use PDBs with
   `minAvailable: 1` or `maxUnavailable: 0`, or they will block node maintenance. Not every user
   needs high availability either, due to a preference for a simpler deployment model, lack of
   application support for HA, or to minimize compute costs. Also, any automated solution needs
   to edit the PDB to account for the additional pod that needs to be spun to move the workload
   from one node to another. This has been discussed in the issue [kubernetes/kubernetes#66811](https://github.com/kubernetes/kubernetes/issues/66811)
   and in the issue [kubernetes/kubernetes#114877](https://github.com/kubernetes/kubernetes/issues/114877).
2. Similar to the first point, it is difficult to use PDBs for applications that can have a variable
   number of pods; for example applications with a configured horizontal pod autoscaler (HPA). These
   applications cannot be disrupted during a low load when they have only one pod. However, it is
   possible to disrupt the pods during a high load of the application (pods > 1) without
   experiencing application downtime. If the minimum number of pods is 1, PDBs cannot be used
   without blocking the node drain. This has been discussed in the issue [kubernetes/kubernetes#93476](https://github.com/kubernetes/kubernetes/issues/93476).
3. Graceful termination of DaemonSet pods is currently only supported on Linux as part of Graceful
   Node Shutdown feature. The length of the shutdown is again not application specific and is set
   cluster-wide (optionally by priority) by the cluster admin. This only partially
   [takes into account](https://github.com/kubernetes/kubernetes/blob/a31030543c47aac36cf323b885cfb6d8b0a2435f/pkg/kubelet/nodeshutdown/nodeshutdown_manager_linux.go#L368-L373)
   `.spec.terminationGracePeriodSeconds` of each pod and may cause premature termination of the
   application. This has been discussed in the issue [kubernetes/kubernetes#75482](https://github.com/kubernetes/kubernetes/issues/75482)
   and in the issue [kubernetes-sigs/cluster-api#6158](https://github.com/kubernetes-sigs/cluster-api/issues/6158).
4. There are cases during a node shutdown, when data corruption can occur due to premature node
   shutdown. It would be great if applications could perform data migration and synchronization of 
   cached writes to the underlying storage before the pod deletion occurs. This is not easy to
   quantify even with pod's `.spec.shutdownGracePeriod`, as the time depends on the size of the data
   and the speed of the storage. This has been discussed in the issue [kubernetes/kubernetes#116618](https://github.com/kubernetes/kubernetes/issues/116618)
   and in the issue [kubernetes/kubernetes#115148](https://github.com/kubernetes/kubernetes/issues/115148).
5. During the Graceful Node Shutdown the kubelet terminates the pods in order of their priority.
   The DaemonSet controller runs its own scheduling logic and creates the pods again. This causes a
   race. Such pods should be removed and not recreated, but higher priority pods that have not yet
   been terminated should be recreated if they are missing. This has been discussed in the issue
   [kubernetes/kubernetes#122912](https://github.com/kubernetes/kubernetes/issues/122912).
6. The Graceful Node Shutdown feature is not always reliable. If Dbus or kubelet is restarted
   during the shutdown, pods may be ungracefully terminated, leading to application disruption and
   data loss. New applications can get scheduled on such a node which can also be harmful.
   This has been discussed in issues [kubernetes/kubernetes#122674](https://github.com/kubernetes/kubernetes/issues/122674),
   [kubernetes/kubernetes#120613](https://github.com/kubernetes/kubernetes/issues/120613) and [kubernetes/kubernetes#122674](https://github.com/kubernetes/kubernetes/issues/112443).
7. There is no way to gracefully terminate static pods during a node shutdown
   [kubernetes/kubernetes#122674](https://github.com/kubernetes/kubernetes/issues/122674), and the
   lifecycle/termination is not clearly defined for static pods [kubernetes/kubernetes#16627](https://github.com/kubernetes/kubernetes/issues/16627).
8. Different pod termination mechanisms are not synchronized with each other. So for example, the
   taint manager may prematurely terminate pods that are currently under Node Graceful Shutdown.
   This can also happen with other mechanism (e.g., different types of evictions). This has been
   discussed in the issue [kubernetes/kubernetes#124448](https://github.com/kubernetes/kubernetes/issues/124448)
   and in the issue [kubernetes/kubernetes#72129](https://github.com/kubernetes/kubernetes/issues/72129).
9. There is not enough metadata about why the node drain was requested or why the pods are
   terminating. This has been discussed in the issue [kubernetes/kubernetes#30586](https://github.com/kubernetes/kubernetes/issues/30586)
   and in the issue [kubernetes/kubernetes#116965](https://github.com/kubernetes/kubernetes/issues/116965).

Approaches and workarounds used by other projects to deal with these shortcomings:
- https://github.com/medik8s/node-maintenance-operator uses a declarative approach that tries to
  mimic `kubectl drain` (and uses kubectl implementation under the hood).
- https://github.com/kubereboot/kured performs automatic node reboots and relies on `kubectl drain`
  implementation to achieve that.
- https://github.com/strimzi/drain-cleaner prevents Kafka or ZooKeeper pods from being drained
  until they are fully synchronized. Implemented by intercepting eviction requests with a
  validating admission webhook. The synchronization is also protected by a PDB with the
  `.spec.maxUnavailable` field set to 0. See the experience reports for more information.
- https://github.com/kubevirt/kubevirt intercepts eviction requests with a validating admission
  webhook to block eviction and to start a virtual machine live migration from one node to another.
  Normally, the workload is also guarded by a PDB with the `.spec.minAvailable` field set to 1.
  During the migration the value is increased to 2.
- https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler has an eviction process
  that takes inspiration from kubectl and build additional logic on top of it.
  See [Cluster Autoscaler](#cluster-autoscaler) for more details.
- https://github.com/kubernetes-sigs/karpenter taints the node during the node drain. It then
  attempts to evict all the pods on the node by calling the Eviction API. It prioritizes
  non-critical pods and non-DaemonSet pods
- https://github.com/aws/aws-node-termination-handler watches for a predefined set of events
  (spot instance termination, EC2 termination, etc.), then cordons and drains the node. It relies
  on the `kubectl` implementation.
- https://github.com/openshift/machine-config-operator updates/drains nodes by using a cordon and
  relies on the `kubectl drain` implementation.
- https://github.com/foriequal0/pod-graceful-drain intercepts eviction/deletion requests to
  gracefully and slowly terminate the pod.

Experience Reports:
- Federico Valeri, [Drain Cleaner: What's this?](https://strimzi.io/blog/2021/09/24/drain-cleaner/), Sep 24, 2021, description
  of the use case and implementation of drain cleaner
- Tommer Amber, [Solution!! Avoid Kubernetes/Openshift Node Drain Failure due to active PodDisruptionBudget](https://medium.com/@tamber/solution-avoid-kubernetes-openshift-node-drain-failure-due-to-active-poddisruptionbudget-df68efed2c4f), Apr 30, 2022 - user
  is unhappy about the manual intervention required to perform node maintenance and gives the
  unfortunate advice to cluster admins to simply override the PDBs. This can have negative
  consequences for user applications, including data loss. This also discourages the use of PDBs.
  We have also seen an interest in the issue [kubernetes/kubernetes#83307](https://github.com/kubernetes/kubernetes/issues/83307)
  for overriding evictions, which led to the addition of the `--disable-eviction` flag to
  `kubectl drain`. There are other examples of this approach on the web .
- Kevin Reeuwijk, [How to handle blocking PodDisruptionBudgets on K8s with distributed storage](https://www.spectrocloud.com/blog/how-to-handle-blocking-poddisruptionbudgets-on-kubernetes-with-distributed-storage), June 6, 2022 - a simple
  shell script example on how to drain the node in a safer way. It does a normal eviction, then
  looks for a pet application (Rook-Ceph in this case) and does hard delete if it does not see it.
  This approach is not plagued by the loss of data resiliency, but it does require maintenaning a
  list of pet applications, which can be prone to mistakes. In the end, the cluster admin has to do
  a job of the application maintainer.
- Artur Rodrigues, [Impossible Kubernetes node drains](https://www.artur-rodrigues.com/tech/2023/03/30/impossible-kubectl-drains.html), 30 Mar, 2023 - discusses
  the problem with node drains and offers a workaround to restart the application without the
  application owner's consents, but acknowledges that this may be problematic without the knowledge
  of the application
- Jack Roper, [How to Delete Pods from a Kubernetes Node with Examples](https://spacelift.io/blog/kubectl-delete-pod), 05 Jul, 2023 - also
  discusses the problem of blocking PDBs and offers several workarounds. Similar to others also
  offers a force deletion, but also a less destructive method of scaling up the application.
  However, this also interferes with application deployment and has to be supported by the
  application.

### Cluster Autoscaler

Accepts a `drain-priority-config` option, which is similar to Graceful Node Shutdown in that it
gives each priority a shutdown grace period. Also has a `max-graceful-termination-sec` option for
pod termination and a `max-pod-eviction-time` option after which the eviction is forfeited.

Each pod is first analyzed to see if it is drainable. Part of the logic is similar to kubectl and
its drain filters (see [Cluster Autoscaler rules](https://github.com/kubernetes/autoscaler/blob/554366f979b11aeb82df335a793e4d7a1acfadb4/cluster-autoscaler/simulator/drainability/rules/rules.go#L50-L77)):
- Mirror pods are skipped.
- Terminating pods are skipped.
- Pods and ReplicaSets/ReplicationControllers without owning controllers are blocking by default
  (the check can be modified with the `skip-nodes-with-custom-controller-pods` option).
- System pods (in the `kube-system` namespace) without a matching PDB are blocking by default
  (the check can be modified with the `skip-nodes-with-system-pods` option).
- Pods with `cluster-autoscaler.kubernetes.io/safe-to-evict: "false"` annotation are blocking.
- Pods with local storage are blocking unless they have a
  `cluster-autoscaler.kubernetes.io/safe-to-evict-local-volumes` annotation
  (the check can be modified with the `skip-nodes-with-local-storage` option, e.g., this check
  is skipped on [AKS](https://learn.microsoft.com/en-us/azure/aks/cluster-autoscaler?tabs=azure-cli#cluster-autoscaler-profile-settings)).
- Pods with PDBs that do not have `disruptionsAllowed` are blocking.

This can be enhanced with other rules and overrides.

It uses this logic to first check if all pods can be removed from a node. If not, it will report
those nodes. Then it will group all pods by a priority and evict them gradually from the lowest
to highest priority. This may include DaemonSet pods.

### kubelet

Graceful Node Shutdown is a part of the current solution for node maintenance. Unfortunately, it is
not possible to rely solely on this feature as a go-to solution for graceful node and workload
termination.

- The Graceful Node Shutdown feature is not application aware and may prematurely disrupt workloads 
  and lead to data loss.
- The kubelet controls the shutdown process using Dbus and systemd, and can delay (but not entirely
  block) it using the systemd inhibitor. However, if Dbus or the kubelet is restarted during the
  node shutdown, the shutdown might not be registered again, and pods might be terminated
  ungracefully. Also, new workloads can get scheduled on the node while the node is shutting down.
  Cluster admins should, therefore, plan the maintenance in advance and ensure that pods are
  gracefully removed before attempting to shut down or restart the machine.
- The kubelet has no way of reliably detecting ongoing maintenance if the node is restarted in the
  meantime.
- Graceful termination of static pods during a shutdown is not possible today. It is also not
  currently possible to prevent them from starting back up immediately after the machine has been
  restarted and the kubelet has started again, if the node is still under maintenance. 

### Motivation Summary

To sum up. Some applications solve the disruption problem by introducing validating admission
webhooks. This has some drawbacks. The webhooks are not easily discoverable by cluster admins. And
they can block evictions for other applications if they are misconfigured or misbehave. The
eviction API is not intended to be extensible in this way. The webhook approach is therefore not
recommended.

Some drainers solve the node drain by depending on the kubectl logic, or by extending/rewriting it
with additional rules and logic.

As seen in the experience reports and GitHub issues, some admins solve their problems by simply
ignoring PDBs which can cause unnecessary disruptions or data loss. Some solve this by playing
with the application deployment, but have to understand that the application supports this.

kubelet's Graceful Node Shutdown feature is a best-effort solution for unplanned shutdowns, but
it is not sufficient to ensure application and data safety.

### Goals
- Introduce NodeMaintenance API.
- Introduce a node maintenance controller that creates evacuations.
- Deprecate kubectl drain in favor of NodeMaintenance. Or at least print a warning.
- Make Graceful Node Shutdown prefer NodeMaintenance during a node shutdown as an opt-in feature
  for a better reliability and application safety.
- Implement NodeMaintenanceAwareKubelet feature to implement a lifecycle for static pods during a
  maintenance.
- Implement NodeMaintenanceAwareDaemonSet feature to prevent the scheduling of DaemonSet pods on
  nodes during a maintenance.

### Non-Goals
- Introduce a node maintenance period, nodeDrainTimeout (similar to [cluster-api](https://cluster-api.sigs.k8s.io/developer/architecture/controllers/control-plane)
  nodeDrainTimeout) or a TTL optional field as an upper bound on the duration of node maintenance.
  Then the node maintenance would be garbage collected and the node made schedulable again.
- Solve the node lifecycle management or automatic shutdown after the node drain is completed.
  Implementation of this is better suited for other cluster components and actors who can use the
  node maintenance as a building block to achieve their desired goals.
- Synchronize all pod termination mechanisms (see #8 in the [Motivation](#motivation) section), so that they do
  not terminate pods under NodeMaintenance/Evacuation.

## Proposal

Most of these issues stem from the lack of a standardized way of detecting a start of the node
drain. This KEP proposes the introduction of a NodeMaintenance object that would signal an intent
to gracefully remove pods from given nodes. The intent will be implemented by the newly proposed
[Evacuation API KEP](https://github.com/kubernetes/enhancements/issues/4563), which ensures
graceful pod removal or migration, an ability to measure the progress and a fallback to eviction if
progress is lost. The NodeMaintenance implementation should also utilize existing node's
`.spec.unschedulable` field, which prevents new pods from being scheduled on such a node.

We will deprecate the `kubectl drain` as the main mechanism for draining nodes and drive the whole
process via a declarative API. This API can be used either manually or programmatically by other
drain implementations (e.g. cluster autoscalers). 

To support workload migration, a new controller should be introduced to observe the NodeMaintenance
objects and then select pods for evacuation. The pods should be selected by node (`nodeSelector`)
and the pods should be gradually evacuated according to the workload they are running.
Controllers can then implement the migration/termination either by reacting to the Evacuation API
or by reacting to the NodeMaintenance API if they need more details.

### User Stories

#### Story 1

As a cluster admin I want to have a simple interface to initiate a node drain/maintenance without
any required manual interventions. I want to have an ability to manually switch between the
maintenance phases (Planning, Cordon, Drain, Drain Complete, Maintenance Complete). I also want to
observe the node drain via the API and check on its progress. I also want to be able to discover
workloads that are blocking the node drain.

#### Story 2

As an application owner, I want to run single replica applications without disruptions and have the
ability to easily migrate the workload pods from one node to another. This also applies to
applications with larger number of replicas that prefer to surge (upscale) pods first rather than
downscale.

#### Story 3

Cluster or node autoscalers that take on the role of `kubectl drain` want to signal the intent to
drain a node using the same API and provide a similar experience to the CLI counterpart.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

- This KEP depends on [Evacuation API KEP](https://github.com/kubernetes/enhancements/issues/4563).

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

A misconfigured .spec.nodeSelector could select all the nodes (or just all master nodes) in the
cluster. This can cause the cluster to get into a degraded and unrecoverable state.

An admission plugin ([NodeMaintenance Admission](#nodemaintenance-admission)) is introduced to
issue a warning in this scenario.

## Design Details

### Kubectl

`kubectl drain`: as we can see in the [Motivation](#motivation) section, kubectl is heavily used
either manually or as a library by other projects. It is safer to keep the old behavior of this
command. However, we will deprecate it along with all the library functions. We can print a
deprecation warning when this command is used, and promote the NodeMaintenance. Additionally, pods
that support evacuation and have `evacuation.coordination.k8s.io/priority_${EVACUATOR_CLASS}`
annotations will block eviction requests.

`kubectl cordon` and `kubectl uncordon` commands will be enhanced with a warning that will warn
the user if a node is made un/schedulable, and it collides with an existing NodeMaintenance object.
As a consequence the node maintenance controller will reconcile the node back to the old value.
Because of this we can make these commands noop when the node is under the NodeMaintenance.

### NodeMaintenance API

NodeMaintenance objects serve as an intent to remove or migrate pods from a set of nodes. We will
include Cordon and Drain toggles to support the following states/stages of the maintenance:
1. Planning: this is to let the users know that maintenance will be performed on a particular set
   of nodes in the future. Configured with `.spec.stage=Idle`.
2. Cordon: stop accepting (scheduling) new pods. Configured with `.spec.stage=Cordon`.
3. Drain: gives an intent to drain all selected nodes by creating `Evacuation` objects for the
   node's pods. Configured with `.spec.stage=Drain`.
4. Drain Complete: all targeted pods have been drained from all the selected nodes. The nodes can
   be upgraded, restarted, or shut down. The configuration is still kept at `.spec.stage=Drain` and
   `Drained` condition is set to `"True"` on the node maintenance object.
5. Maintenance Complete: make the nodes schedulable again once the node maintenance is done.
   Configured with `.spec.stage=Complete`.

```golang

// +enum
type NodeMaintenanceStage string

const (
// Idle does not interact with the cluster.
Idle NodeMaintenanceStage = "Idle"
// Cordon cordons all selected nodes by making them unschedulable.
Cordon NodeMaintenanceStage = "Cordon"
// Drain:
// 1. Cordons all selected nodes by making them unschedulable.
// 2. Gives an intent to drain all selected nodes by creating Evacuation objects for the
//    node's pods.
Drain NodeMaintenanceStage = "Drain"
// Complete:
// 1. Removes all Evacuation objects requested by this NodeMaintenance.
// 2. Uncordons all selected nodes by making them schedulable again, unless there is not another
//    maintenance in progress.
Complete NodeMaintenanceStage = "Complete"
)

type NodeMaintenance struct {
    ...
    Spec NodeMaintenanceSpec
    Status NodeMaintenanceStatus
}

type NodeMaintenanceSpec struct {
    // NodeSelector selects nodes for this node maintenance.
    // +required
    NodeSelector *v1.NodeSelector

    // The order of the stages is Idle -> Cordon -> Drain -> Complete.
    //
    // - The Cordon or Drain stage can be skipped by setting the stage to Complete.
    // - The NodeMaintenance object is moved to the Complete stage on deletion unless the Idle stage has been set.
    //
    // The default value is Idle.
    Stage NodeMaintenanceStage

    // DrainPlan is executed from the first entry to the last entry during the Drain stage.
    // DrainPlanEntry podType fields should be in the following order:
    // nil -> DaemonSet -> Static
    // DrainPlanEntry priority fields should be in ascending order for each podType.
    // If the priority and podType are the same, concrete selectors are executed first.
    //
    // The following entries are injected into the drainPlan on the NodeMaintenance admission:
    // - podPriority: 1000000000 // highest priority for user defined priority classes
    //   podType: "Default"
    // - podPriority: 2000000000 // system-cluster-critical priority class
    //   podType: "Default"
    // - podPriority: 2000001000 // system-node-critical priority class
    //   podType: "Default"
    // - podPriority: 2147483647 // maximum value
    //   podType: "Default"
    // - podPriority: 1000000000 // highest priority for user defined priority classes
    //   podType: "DaemonSet"
    // - podPriority: 2000000000 // system-cluster-critical priority class
    //   podType: "DaemonSet"
    // - podPriority: 2000001000 // system-node-critical priority class
    //   podType: "DaemonSet"
    // - podPriority: 2147483647 // maximum value
    //   podType: "DaemonSet"
    // - podPriority: 1000000000 // highest priority for user defined priority classes
    //   podType: "Static"
    // - podPriority: 2000000000 // system-cluster-critical priority class
    //   podType: "Static"
    // - podPriority: 2000001000 // system-node-critical priority class
    //   podType: "Static"
    // - podPriority: 2147483647 // maximum value
    //   podType: "Static"
    //
    // Duplicate entries are not allowed.
    // This field is immutable.
    DrainPlan []DrainPlanEntry

    // Reason for the maintenance.
    Reason string
}

const (
// Default selects all pods except DaemonSet and Static pods.	
Default PodType = "Default"
// DaemonSet selects DaemonSet pods.
DaemonSet PodType = "DaemonSet"
// Static selects static pods.
Static PodType = "Static"
)

type DrainPlanEntry struct {
    // PodSelector selects pods according to their labels.
    // This can help to select which pods of the same priority should be evacuated first.
    // +optional
    PodSelector *metav1.LabelSelector
    // PodPriority specifies a pod priority.
    // Pods with a priority less or equal to this value are selected.
    PodPriority int32
    // PodType selects pods according to the pod type:
    // - Default selects all pods except DaemonSet and Static pods.	
    // - DaemonSet selects DaemonSet pods.
    // - Static selects static pods.
    PodType PodType
}

type NodeMaintenanceStatus struct {
    // StageStatuses tracks the statuses of started stages.
    StageStatuses []StageStatus
    DrainStatus DrainStatus
    Conditions []metav1.Condition
}

type StageStatus struct {
    // Name of the Stage.
    Name NodeMaintenanceStage
    // StartTimestamp is the time that indicates the start of this stage.
    StartTimestamp *metav1.Time
}

type DrainStatus struct {
    // ReachedDrainTargets indicates which pods on all selected nodes are currently being targeted
    // for evacuation. Some of the nodes may have reached higher drain targets. This field tracks
    // only the lowest drain targets among all nodes. Consult the status of each node to observe
    // its current drain targets.
    //
    // Once evacuation of the Default PodType finishes, DaemonSet PodType entries appear.
    // Once the evacuation of DaemonSet PodType finishes, Static PodType entries appear.
    // The PodPriority for these entries is increased over time according to the .spec.DrainPlan
    // as the lower-priority pods finish evacuation.
    // The next entry in the .spec.DrainPlan is selected once all the nodes have reached their
    // DrainTargets.
    // If there are multiple NodeMaintenances for a node, the least powerful DrainTargets among
    // them are selected and set for that node. Thus, the DrainTargets do not have to correspond
    // to the entries in .spec.drainPlan for a single NodeMaintenance instance.
    // DrainTargets cannot backtrack and will target more pods with each update until all pods on
    // the node are targeted.
    ReachedDrainTargets  []DrainPlanEntry
    // Number of pods that have not yet started evacuation.
    PodsPendingEvacuation int32
    // Number of pods that have started evacuation and have a matching Evacuation object.
    PodsEvacuating int32
}

// NodeStatus is information about the current status of a node.
type NodeStatus struct {
    ...
    // MaintenanceStatus is present if a node is under a maintenance. This means there is at least
    // one active NodeMaintenance object targeting this node.
    MaintenanceStatus *MaintenanceStatus
}

type MaintenanceStatus struct {
    // DrainTargets specifies which pods on this node are currently being targeted for evacuation.
    // Once evacuation of the Default PodType finishes, DaemonSet PodType entries appear.
    // Once the evacuation of DaemonSet PodType finishes, Static PodType entries appear.
    // The PodPriority for these entries is increased over time according to the .spec.DrainPlan
    // as the lower-priority pods finish evacuation.
    // The next entry in the .spec.DrainPlan is selected once all the nodes have reached their
    // DrainTargets.
    // If there are multiple NodeMaintenances for a node, the least powerful DrainTargets among
    // them are selected and set for that node. Thus, the DrainTargets do not have to correspond
    // to the entries in .spec.drainPlan for a single NodeMaintenance instance.
    // DrainTargets cannot backtrack and will target more pods with each update until all pods on
    // the node are targeted.
    DrainTargets  []DrainPlanEntry
    // DrainMessage may specify a state of the drain on this node and a reason why the drain
    // targets are set to a particular values.
    DrainMessage string
    // Number of pods that have not yet started evacuation.
    PodsPendingEvacuation int32
    // Number of pods that have started evacuation and have a matching Evacuation object.
    PodsEvacuating int32
}

const (
    // DrainedCondition is a condition set by the node-maintenance controller that signals
    // whether all pods pending termination have terminated on all target nodes when drain is
    // requested by the maintenance object.
    DrainedCondition = "Drained"
}
```

### NodeMaintenance Admission

`nodemaintenance` admission plugin will be introduced.

It will validate all incoming requests for CREATE, UPDATE, and DELETE operations on the
NodeMaintenance objects. All nodes matching the `.spec.nodeSelector` must pass an authorization
check for the DELETE operation.

Also, if the `.spec.nodeSelector` matches all cluster nodes, a warning will be produced indicating
that the cluster may get into a degraded and unrecoverable state. The warning is non-blocking and
such NodeMaintenance is still valid and can proceed.

### NodeMaintenance Controller

Node maintenance controller will be introduced and added to `kube-controller-manager`. It will
observe NodeMaintenance objects and have the following main features:

#### Idle

The controller should not touch the pods or nodes that match the selector of the NodeMaintenance
object in any way in the `Idle` stage.

#### Finalizers and Deletion of the NodeMaintenance

When a stage is not `Idle`, `nodemaintenance.k8s.io/maintenance-completion` finalizer is placed on
the NodeMaintenance object to ensure uncordon and removal of Evacuations upon deletion.

When a deletion of the NodeMaintenance object is detected, its `.spec.stage` is set to `Complete`.
The finalizer is not removed until the `Complete` stage has been completed.

#### Cordon

When a `Cordon` or `Drain` stage is detected on the NodeMaintenance object, the controller
will set (and reconcile) `.spec.Unschedulable` to `true` on all nodes that satisfy
`.spec.nodeSelector`. It should alert via events if too many occur appear and a race to change
this field is detected.

An alternative to prevent raciness is to make the scheduler aware of active NodeMaintenances and
not schedule new pods there.

#### Uncordon (Complete)

When a `Complete` stage is detected on the NodeMaintenance object, the controller sets
`.spec.Unschedulable` back to `false`  on all nodes that satisfy `.spec.nodeSelector`, unless there
is no other maintenance in progress.

When the node maintenance is canceled (reaches the `Complete` stage without all of its pods
terminating), the controller will attempt to remove all Evacuations that match the node maintenance,
unless there is no other maintenance in progress.
- If there are foreign finalizers on the Evacuation, it should only remove its own instigator
  finalizer (see [Drain](#drain)).
- If the evacuator does not support a cancellation and it has set
  `.status.evacuationCancellationPolicy` to `Forbid`, deletion of the Evacuation object will not be
  attempted.

Consequences for pods:
1. Pods whose evacuators have not yet initiated evacuation will continue to run unchanged.
2. Pods whose evacuators have initiated evacuation and support cancellation
   (`.status.evacuationCancellationPolicy=Allow`) should cancel the evacuation and keep the pods
   available.
3. Pods whose evacuators have initiated evacuation and do not support cancellation
   (`.status.evacuationCancellationPolicy=Forbid`) should continue the evacuation and eventually
   terminate the pods.

#### Drain

When a `Drain` stage is detected on the NodeMaintenance object, Evacuation objects are created for
selected pods ([Pod Selection](#pod-selection)).

```yaml
apiVersion: v1alpha1
kind: Evacuation
metadata:
  finalizers:
    evacuation.coordination.k8s.io/instigator_nodemaintenance.k8s.io
  name: f5823a89-e03f-4752-b013-445643b8c7a0-muffin-orders-6b59d9cb88-ks7wb
  namespace: blue-deployment
spec:
  podRef:
    name: muffin-orders-6b59d9cb88-ks7wb
    uid:  f5823a89-e03f-4752-b013-445643b8c7a0
  progressDeadlineSeconds: 1800

```

This is resolved to the following Evacuation object according to the pod on admission:

```yaml

apiVersion: v1alpha1
kind: Evacuation
metadata:
  finalizers:
    evacuation.coordination.k8s.io/instigator_nodemaintenance.k8s.io
  labels:
    app: muffin-orders
  name: f5823a89-e03f-4752-b013-445643b8c7a0-muffin-orders-6b59d9cb88-ks7wb
  namespace: blue-deployment
spec:
  podRef:
    name: muffin-orders-6b59d9cb88-ks7wb
    uid:  f5823a89-e03f-4752-b013-445643b8c7a0
  progressDeadlineSeconds: 1800
  evacuators:
    - evacuatorClass: deployment.apps.k8s.io
      priority: 10000
      role: controller
```

The node maintenance controller requests the removal of a pod from a node by the presence of the
Evacuation. Setting `progressDeadlineSeconds` to  1800 (30m) should give potential evacuators
enough time to recover from a disruption and continue with the graceful evacuation. If the
evacuators are unable to evacuate the pod, or if there are no evacuators, the evacuation controller
will attempt to evict these pods, until they are deleted.

The only job of the node maintenance controller is to make sure that the Evacuation object exist
and has the `evacuation.coordination.k8s.io/instigator_nodemaintenance.k8s.io` finalizer. 

#### Pod Selection

The pods for evacuation would first be selected by node (`.spec.nodeSelector`). NodeMaintenance
should eventually remove all the pods from each node. To do this in a graceful manner, the
controller will first ensure that lower priority pods are evacuated first for the same pod type.
The user can also target some pods earlier than others with a label selector.

DaemonSet and static pods typically run critical workloads that should be scaled down last.

<<[UNRESOLVED Pod Selection Priority]>>
Should user daemon sets (priority up to 1000000000) be scaled down first?
<<[/UNRESOLVED]>>


To achieve this, we will ensure that the NodeMaintenance `.spec.drainPlan` always contains the
following entries:

```yaml
spec:
  drainPlan:
    - podPriority: 1000000000 # highest priority for user defined priority classes
      podType: "Default"
    - podPriority: 2000000000 # system-cluster-critical priority class
      podType: "Default"
    - podPriority: 2000001000 # system-node-critical priority class
      podType: "Default"
    - podPriority: 2147483647 # maximum value
      podType: "Default"
    - podPriority: 1000000000 # highest priority for user defined priority classes
      podType: "DaemonSet"
    - podPriority: 2000000000 # system-cluster-critical priority class
      podType: "DaemonSet"
    - podPriority: 2000001000 # system-node-critical priority class
      podType: "DaemonSet"
    - podPriority: 2147483647 # maximum value
      podType: "DaemonSet"
    - podPriority: 1000000000 # highest priority for user defined priority classes
      podType: "Static"
    - podPriority: 2000000000 # system-cluster-critical priority class
      podType: "Static"
    - podPriority: 2000001000 # system-node-critical priority class
      podType: "Static"
    - podPriority: 2147483647 # maximum value
      podType: "Static"
  ...
```

If not they will be added during the NodeMaintenance admission.

The node maintenance controller resolves this plan across intersecting NodeMaintenances. To
indicate which pods are being evacuated on which node, the controller populates
`.status.maintenanceStatus.drainTargets` on each node object. It also populates
`.status.drainStatus.reachedDrainTargets` of the NodeMaintenance to track the lowest drain targets among all
nodes (pods that are being evacuated/evicted everywhere). These status fields are updated during
the `Drain` stage to incrementally select pods with higher priority and pod type
(`Default` ->`DaemonSet` -> `Static`). It is also possible to partition the updates for the same
priorities according to the pod labels.

If there is only a single NodeMaintenance present, it selects the first entry from the
`.spec.drainPlan` and makes sure that all the targeted pods are evacuated/removed. It then selects
the next entry and repeats the process. If a new pod appears that matches the previous entries, it
will also be evacuated.

If there are multiple NodeMaintenances, we have to first resolve the lowest priority entry from the
`.spec.drainPlan` among them for the intersecting nodes. Non-intersecting nodes may have a higher
priority or pod type. The next entry in the plan can be selected once all the nodes of a
NodeMaintenance have finished evacuation and all the NodeMaintenances of intersecting nodes have
finished evacuation for the current drain targets. See the [Pod Selection and DrainTargets Example](#pod-selection-and-draintargets-example)
for additional details.

A similar kind of drain plan, albeit with fewer features is offered today by the
[Graceful Node Shutdown](https://kubernetes.io/docs/concepts/architecture/nodes/#graceful-node-shutdown)
feature and by the Cluster Autoscaler's [drain-priority-config](https://github.com/kubernetes/autoscaler/pull/6139).
The downside of these configurations is that they have `shutdownGracePeriodSeconds` which sets a
limit on how long the termination of pods should take. This is not application-aware and some
applications may require more time to gracefully shut down. Allowing such hard-coded timeouts may
result in unnecessary application disruptions or data corruption.

To support the evacuation of `DaemonSet` and `Static` pods, the daemon set controller and kubelet
should observe NodeMaintenance objects and Evacuations to coordinate the scale down of the pods on
the targeted nodes.

To ensure more streamlined experience we will not support the default kubectl [drain filters](https://github.com/kubernetes/kubernetes/blob/56cc5e77a10ba156694309d9b6159d4cd42598e1/staging/src/k8s.io/kubectl/pkg/drain/filters.go#L153-L162).
Instead, it should be possible to create the NodeMaintenance object with just a `spec.nodeSelector`.
The only thing that can be configured is which pods should be scaled down first.

NodeMaintenance alternatives to kubectl drain filters:
- `daemonSetFilter`: Removal of these pods should be supported by the DaemonSet controller.
- `mirrorPodFilter`: Removal of these pods should be supported by the kubelet.
- `skipDeletedFilter`: Creating evacuation of already terminating pods should have no downside and
  be informative for the user.
- `unreplicatedFilter`: Actors who own pods without a controller owner reference should have the
  opportunity to register an evacuator to evacuate their pods. Many drain solutions today evict
  these types of pods indiscriminately.
- `localStorageFilter`: Actors who own pods with local storage (having `EmptyDir` volumes) should
  have the opportunity to register an evacuator to evacuate their pods. Many drain solutions today
  evict these types of pods indiscriminately.

#### Pod Selection and DrainTargets Example

If two Node Maintenances are created at the same time for the same node. Then, for the intersecting
nodes, the entry with the lowest priority in the drainPlan is resolved first.

```yaml
apiVersion: v1alpha1
kind: NodeMaintenance
metadata:
  name: "maintenance-a"
spec:
  nodeSelector:
    # selects nodes one and two
  stage: Drain
  drainPlan:
    - podPriority: 5000
      podType: Default
    - podPriority: 15000
      podType: Default
    - podPriority: 3000
      podType: DaemonSet
  ...
status:
  drainStatus:
    podsPendingEvacuation: 130
    podsEvacuating: 17
    drainMessage: "Evacuating"
    reachedDrainTargets:
      - podPriority: 5000
        podType: Default
  ...
---
apiVersion: v1alpha1
kind: NodeMaintenance
metadata:
  name: "maintenance-b"
spec:
  nodeSelector:
    # selects nodes one and three
  stage: Drain
  drainPlan:
    - podPriority: 10000
      podType: Default
    - podPriority: 15000
      podType: Default
    - podPriority: 4000
      podType: DaemonSet
  ...
status:
  drainStatus:
    podsPendingEvacuation: 145
    podsEvacuating: 35
    drainMessage: "Evacuating (limited by maintenance-a)"
    reachedDrainTargets:
      - podPriority: 5000
        podType: Default
  ...
---
apiVersion: v1
kind: Node
metadata:
  name: "one"
status:
  maintenanceStatus:
    podsPendingEvacuation: 100
    podsEvacuating: 10
    drainMessage: "Evacuating"
    drainTargets:
      - podPriority: 5000
        podType: Default
  ...
---
apiVersion: v1
kind: Node
metadata:
  name: "two"
status:
  maintenanceStatus:
    podsPendingEvacuation: 30
    podsEvacuating: 7
    drainMessage: "Evacuating"
    drainTargets:
      - podPriority: 5000
        podType: Default
  ...
---
apiVersion: v1
kind: Node
metadata:
  name: "three"
status:
  maintenanceStatus:
    podsPendingEvacuation: 45
    podsEvacuating: 25
    drainMessage: "Evacuating"
    drainTargets:
      - podPriority: 10000
        podType: Default
  ...
```

If the node three is drained, then it has to wait for the node one, because the drain plan
specifies that all the pods with priority 10000 or lower should be evacuated first before moving on
to the next entry.

```yaml
apiVersion: v1alpha1
kind: NodeMaintenance
metadata:
  name: "maintenance-b"
spec:
  nodeSelector:
    # selects nodes one and three
  stage: Drain
  drainPlan:
    - podPriority: 10000
      podType: Default
    - podPriority: 15000
      podType: Default
    - podPriority: 4000
      podType: DaemonSet
  ...
status:
  drainStatus:
    podsPendingEvacuation: 145
    podsEvacuating: 5
    drainMessage: "Evacuating (limited by maintenance-a)"
    reachedDrainTargets:
      - podPriority: 5000
        podType: Default
---
apiVersion: v1
kind: Node
metadata:
  name: "three"
status:
  maintenanceStatus:
    podsPendingEvacuation: 45
    podsEvacuating: 0
    drainMessage: "Waiting for maintenance-a."
    drainTargets:
      - podPriority: 10000
        podType: Default
  ...
```

If the node one is drained, we still have to wait for the `maintenance-a` to drain node two. If we
were to start evacuating higher priority pods from node one earlier, we would not conform to the
drainPlan of `maintenance-a`. The plan specifies that all the pods with priority 5000 or lower
should be evacuated first before moving on to the next entry.


```yaml
apiVersion: v1alpha1
kind: NodeMaintenance
metadata:
  name: "maintenance-a"
spec:
  nodeSelector:
    # selects nodes one and two
  stage: Drain
  drainPlan:
    - podPriority: 5000
      podType: Default
    - podPriority: 15000
      podType: Default
    - podPriority: 3000
      podType: DaemonSet
  ...
status:
  drainStatus:
    podsPendingEvacuation: 130
    podsEvacuating: 2
    drainMessage: "Evacuating"
    reachedDrainTargets:
      - podPriority: 5000
        podType: Default
  ...
---
apiVersion: v1alpha1
kind: NodeMaintenance
metadata:
  name: "maintenance-b"
spec:
  nodeSelector:
    # selects nodes one and three
  stage: Drain
  drainPlan:
    - podPriority: 10000
      podType: Default
    - podPriority: 15000
      podType: Default
    - podPriority: 4000
      podType: DaemonSet
  ...
status:
  drainStatus:
    podsPendingEvacuation: 145
    podsEvacuating: 0
    drainMessage: "Waiting for maintenance-a."
    reachedDrainTargets:
      - podPriority: 5000
        podType: Default
  ...
---
apiVersion: v1
kind: Node
metadata:
  name: "one"
status:
  maintenanceStatus:
    podsPendingEvacuation: 100
    podsEvacuating: 0
    drainMessage: "Waiting for maintenance-a."
    drainTargets:
      - podPriority: 5000
        podType: Default
  ...
---
apiVersion: v1
kind: Node
metadata:
  name: "two"
status:
  maintenanceStatus:
    podsPendingEvacuation: 30
    podsEvacuating: 2
    drainMessage: "Evacuating"
    drainTargets:
      - podPriority: 5000
        podType: Default
  ...
---
apiVersion: v1
kind: Node
metadata:
  name: "three"
status:
  maintenanceStatus:
    podsPendingEvacuation: 45
    podsEvacuating: 0
    drainMessage: "Waiting for maintenance-a."
    drainTargets:
      - podPriority: 10000
        podType: Default
  ...
```

Once the node two drains, we can increment the drainTargets. 


```yaml
apiVersion: v1alpha1
kind: NodeMaintenance
metadata:
  name: "maintenance-a"
spec:
  nodeSelector:
    # selects nodes one and two
  stage: Drain
  drainPlan:
    - podPriority: 5000
      podType: Default
    - podPriority: 15000
      podType: Default
    - podPriority: 3000
      podType: DaemonSet
  ...
status:
  drainStatus:
    podsPendingEvacuation: 91
    podsEvacuating: 39
    drainMessage: "Evacuating"
    reachedDrainTargets:
      - podPriority: 10000
        podType: Default
  ...
---
apiVersion: v1alpha1
kind: NodeMaintenance
metadata:
  name: "maintenance-b"
spec:
  nodeSelector:
    # selects nodes one and three
  stage: Drain
  drainPlan:
    - podPriority: 10000
      podType: Default
    - podPriority: 15000
      podType: Default
    - podPriority: 4000
      podType: DaemonSet
  ...
status:
  drainStatus:
    podsPendingEvacuation: 115
    podsEvacuating: 30
    drainMessage: "Evacuating"
    reachedDrainTargets:
      - podPriority: 10000
        podType: Default
  ...
---
apiVersion: v1
kind: Node
metadata:
  name: "one"
status:
  maintenanceStatus:
    podsPendingEvacuation: 70
    podsEvacuating: 30
    drainMessage: "Evacuating (limited by maintenance-b)"
    drainTargets:
      - podPriority: 10000
        podType: Default
  ...
---
apiVersion: v1
kind: Node
metadata:
  name: "two"
status:
  maintenanceStatus:
    podsPendingEvacuation: 21
    podsEvacuating: 9
    drainMessage: "Evacuating"
    drainTargets:
      - podPriority: 15000
        podType: Default
  ...
---
apiVersion: v1
kind: Node
metadata:
  name: "three"
status:
  maintenanceStatus:
    podsPendingEvacuation: 45
    podsEvacuating: 0
    drainMessage: "Waiting for maintenance-b."
    drainTargets:
      - podPriority: 10000
        podType: Default
  ...
```

The progress of the drain should not be backtracked. If an intersecting `maintenance-c` is created,
the node one progress should stay the same regardless of the node maintenance drainPlan.


```yaml
apiVersion: v1alpha1
kind: NodeMaintenance
metadata:
  name: "maintenance-c"
spec:
  nodeSelector:
    # selects nodes one and four
  stage: Drain
  drainPlan:
    - podPriority: 2000
      podType: Default
    - podPriority: 15000
      podType: Default
  ...
status:
  drainStatus:
    podsPendingEvacuation: 90
    podsEvacuating: 35
    drainMessage: "Evacuating"
    reachedDrainTargets:
      - podPriority: 2000
        podType: Default
  ...
---
apiVersion: v1
kind: Node
metadata:
  name: "one"
status:
  maintenanceStatus:
    podsPendingEvacuation: 70
    podsEvacuating: 30
    drainMessage: "Evacuating (limited by maintenance-b, maintenance-c)"
    drainTargets:
      - podPriority: 10000
        podType: Default
  ...
---
apiVersion: v1
kind: Node
metadata:
  name: "four"
status:
  maintenanceStatus:
    podsPendingEvacuation: 20
    podsEvacuating: 5
    drainMessage: "Evacuating"
    drainTargets:
      - podPriority: 2000
        podType: Default
  ...
---
```

This is done to ensure that the pre-conditions of the older maintenances (`maintenance-a` and
`maintenance-b`) are not broken. When we remove workloads with priority 15000, our pre-condition is
that workloads with priority 5000 that might depend on these 15000 priority workloads are gone. If
we allow rescheduling of the lower priority pods, this assumption is broken.

Unfortunately, a similar precondition is broken for the `maintenance-c`, so we can at least emit an
event saying that we are fast-forwarding `maintenance-c` due to existing older maintenance(s). In
the extreme scenario, node one may already be turned off and creating a new maintenance that
assumes priority X pods are still running will not help to bring it back. Emitting an event would
help with observability and might help cluster admins better schedule node maintenances.

##### PodTypes and Label Selectors Progression

An example progression for the following drain plan might look as follows:


```yaml
spec:
  stage: Drain
  drainPlan:
    - podPriority: 1000
      podType: Default
    - podPriority: 2000
      podType: Default
      podSelector:
        matchLabels:
          app: postgres
    - podPriority: 2147483647
      podType: Default
    - podPriority: 1000
      podType: DaemonSet
    - podPriority: 2147483647
      podType: DaemonSet
    - podPriority: 2147483647
      podType: Static
status:
  nodeStatuses:
    - nodeRef:
        name: five
      drainTargets:
        - podPriority: 1000
          podType: Default
        - podPriority: 1000
          podType: Default
          podSelector:
            matchLabels:
              app: postgres 
  ...
```

```yaml
status:
  nodeStatuses:
    - nodeRef:
        name: five
      drainTargets:
        - podPriority: 1000
          podType: Default
        - podPriority: 2000
          podType: Default
          podSelector:
            matchLabels:
              app: postgres 
  ...
```

```yaml
status:
  nodeStatuses:
    - nodeRef:
        name: five
      drainTargets:
        - podPriority: 2147483647
          podType: Default
        - podPriority: 2147483647
          podType: Default
          podSelector:
            matchLabels:
              app: postgres 
  ...
```

```yaml
status:
  nodeStatuses:
    - nodeRef:
        name: five
      drainTargets:
        - podPriority: 2147483647
          podType: Default
        - podPriority: 2147483647
          podType: Default
          podSelector:
            matchLabels:
              app: postgres
        - podPriority: 1000
          podType: DaemonSet
  ...
```

```yaml
status:
  nodeStatuses:
    - nodeRef:
        name: five
      drainTargets:
        - podPriority: 2147483647
          podType: Default
        - podPriority: 2147483647
          podType: Default
          podSelector:
            matchLabels:
              app: postgres
        - podPriority: 2147483647
          podType: DaemonSet
  ...
```

```yaml
status:
  nodeStatuses:
    - nodeRef:
        name: five
      drainTargets:
        - podPriority: 2147483647
          podType: Default
        - podPriority: 2147483647
          podType: Default
          podSelector:
            matchLabels:
              app: postgres
        - podPriority: 2147483647
          podType: DaemonSet
        - podPriority: 2147483647
          podType: Static
  ...
```

#### Status

The controller can show progress by reconciling:
- `.status.stageStatuses` should be amended when a new stage is selected. This is used to track
  which stages have been started. Additional metadata can be added to this struct in the future.
-  `.status.drainStatus.drainTargets` should be updated during a `Drain` stage. The drain
  targets should be resolved according to the [Pod Selection](#pod-selection) and [Pod Selection and DrainTargets Example](#pod-selection-and-draintargets-example).
-  `.status.drainStatus.drainMessage` should be updated during a `Drain` stage. The message
   should be resolved according to [Pod Selection and DrainTargets Example](#pod-selection-and-draintargets-example).
- `.status.drainStatus.podsPendingEvacuation`, to indicate how many pods are left to start
  evacuation from the first node.
- `.status.drainStatus.podsEvacuating`, to indicate how many pods are being evacuated from the first node.
  These are the pods that have matching Evacuation objects. 
- To keep track of the entire maintenance the controller will reconcile a `Drained` condition and
  set it to true if all pods pending evacuation/termination have terminated on all target nodes
  when drain is requested by the maintenance object.
- NodeMaintenance condition or annotation can be set on the node object to advertise the current
  phase of the maintenance.

#### Supported Stage Transitions

The following transitions should be validated by the API server.

- Idle -> _Deletion_
  - Planning a maintenance in the future and canceling/deleting it without any consequence.
- (Idle) -> Cordon -> (Complete) -> _Deletion_.
  - Make a set of nodes unschedulable and then schedulable again.
  - The complete stage will always be run even without specifying it.
- (Idle) -> (Cordon) -> Drain -> (Complete) -> _Deletion_.
  - Make a set of nodes unschedulable, drain them, and then make them schedulable again.
  - Cordon and Complete stages will always be run, even without specifying them.
- (Idle) -> Complete -> _Deletion_.
  -  Make a set of nodes schedulable.

The stage transitions are invoked either manually by the cluster admin or by a higher-level
controller. For a simple drain, cluster admin can simply create the NodeMaintenance with
`stage: Drain` directly.

### DaemonSet Controller

The DaemonSet workloads should be tied to the node lifecycle because they typically run critical
workloads where availability is paramount. Therefore, the DaemonSet controller should respond to
the Evacuation only if there is a NodeMaintenance happening on that node and the DaemonSet is in
the `drainTargets`. For example, if we observe the following NodeMaintenance:

```yaml
apiVersion: v1alpha1
kind: NodeMaintenance
...
status:
  nodeStatuses:
    - nodeRef:
        name: six
      drainTargets:
        - podPriority: 2147483647
          podType: Default
        - podPriority: 5000
          podType: DaemonSet
  ...
```

To fulfil the Evacuation API, the DaemonSet controller should register itself as a controller
evacuator. To do this, it should ensure that the following annotation is present on its own pods.

```yaml
evacuation.coordination.k8s.io/priority_daemonset.apps.k8s.io: "10000/controller"
```

The controller should respond to the Evacuation object when it observes its own class
(`daemonset.apps.k8s.io`) in `.status.activeEvacuatorClass`.

For the above node maintenance, the controller should not react to Evacuations of DaemonSet pods
with a priority greater than 5000. This state should not normally occur, as Evacuation requests
should be coordinated with NodeMaintenance. If it does occur, we should not encourage this flow by
updating the `.status.ActiveEvacuatorCompleted` field, although it is required to update this field
for normal workloads.

If the DaemonSet pods have a priority equal to or less than 5000, the Evacuation status should be
updated appropriately as follows, and the targeted pod should be deleted by the DaemonSet
controller:

```yaml
apiVersion: v1alpha1
kind: Evacuation
metadata:
  finalizers:
    evacuation.coordination.k8s.io/instigator_nodemaintenance.k8s.io
  labels:
    app: critical-ds
  name: ae9b4bc6-e4ca-4f8e-962b-2d4459b1f684-critical-ds-5nxjs
  namespace: critical-workloads
spec:
  podRef:
    name: critical-ds-5nxjs
    uid:  ae9b4bc6-e4ca-4f8e-962b-2d4459b1f684
  progressDeadlineSeconds: 1800
  evacuators:
    - evacuatorClass: daemonset.apps.k8s.io
      priority: 10000
      role: controller
status:
  activeEvacuatorClass: daemonset.apps.k8s.io
  activeEvacuatorCompleted: false
  evacuationProgressTimestamp: "2024-04-22T21:40:32Z"
  expectedEvacuationFinishTime: "2024-04-22T21:41:32Z" # now + terminationGracePeriodSeconds:
  failedEvictionCounter: 0
  message: "critical-ds is terminating the pod due to node maintenance (OS upgrade)."
  conditions: []
```

Once the pod is terminated and removed from the node, it should not be re-scheduled on the node by
the DaemonSet controller until the node maintenance is complete.

### kubelet: Graceful Node Shutdown

The current Graceful Node Shutdown feature has a couple of drawbacks when compared to
NodeMaintenance:
- It is application agnostic as it only provides a static grace period before the shutdown based on
  priority. This does not always give the application enough time to react and can lead to data
  loss or application availability loss.
- The DaemonSet pods may be running important services (critical priority) that should be available
  even during part of the shutdown. The daemon set controller does not have the observability of
  the kubelet shutdown procedure and cannot infer which DaemonSets should stop running. The
  controller needs to know which DaemonSets should run on each node with which priorities and
  reconcile accordingly.

To support these use cases we could introduce a new configuration option to the kubelet called
`preferNodeMaintenanceDuringGracefulShutdown`.

This would result in the following behavior:

When a shutdown is detected, the kubelet would create a NodeMaintenance object for that node.
Then it would block the shutdown indefinitely, until all the pods are terminated. The kubelet
could pass the priorities from the `shutdownGracePeriodByPodPriority` to the NodeMaintenance,
just without the `shutdownGracePeriodSeconds`. This would give applications a chance to react and
gracefully leave the node without a timeout. [Pod Selection](#pod-selection) would ensure that user
workloads are terminated first and critical pods are terminated last. 

By default, all user workloads will be asked to terminate at once. The Evacuation API ensures that
an evacuator is selected or an eviction API is called. This should result in a fast start of a pod
termination. NodeMaintenance could then be used even by spot instances.

The NodeMaintenance object should survive kubelet restarts, and the kubelet would always know if
the node is under shutdown (maintenance). The cluster admin would have to remove the
NodeMaintenance object after the node restart to indicate that the node is healthy and can run pods
again. Admins are expected to deal with the lifecycle of planned NodeMaintenances, so reacting to
the unplanned one should not be a big issue.

If there is no connection to the apiserver (apiserver down, network issues, etc.) and the
NodeMaintenance object cannot be created, we would fall back to the original behavior of Graceful
Node Shutdown feature. If the connection is restored, we would stop the Graceful Node Shutdown and
proceed with the NodeMaintenance.

The NodeMaintenance would ensure that all pods are removed. This also includes the DaemonSet and
static pods.

### kubelet: Static Pods

Currently, there is no standard solution for terminating static pods. We can advertise what state
each node should be in, declaratively with NodeMaintenance. This can include static pods as well.

Since static pods usually run the most critical workloads, they should be terminated last according
to [Pod Selection](#pod-selection).

Similar to [DaemonSets](#daemonset-controller), static pods should be tied to the node lifecycle
because they typically run critical workloads where availability is paramount. Therefore, the
kubelet should respond to the Evacuation only if there is a NodeMaintenance happening on that node
and the `Static` pod is in the `drainTargets`. For example, if we observe the following
NodeMaintenance:

```yaml
apiVersion: v1alpha1
kind: NodeMaintenance
...
status:
  nodeStatuses:
    - nodeRef:
        name: six
      drainTargets:
        - podPriority: 2147483647
          podType: Default
        - podPriority: 2147483647
          podType: DaemonSet
        - podPriority: 7000
          podType: Static
  ...
```

To fulfil the Evacuation API, the DaemonSet controller should register itself as a controller
evacuator. To do this, it should ensure that the following annotation is present on its own pods.

```yaml
evacuation.coordination.k8s.io/priority_kubelet.k8s.io: "10000/controller"
```

The kubelet should respond to the Evacuation object when it observes its own class
(`kubelet.k8s.io`) in `.status.activeEvacuatorClass`.

For the above node maintenance, the kubelet should not react to Evacuations of static pods
with a priority greater than 7000. This state should not normally occur, as Evacuation requests
should be coordinated with NodeMaintenance. If it does occur, we should not encourage this flow by
updating the `.status.ActiveEvacuatorCompleted` field, although it is required to update this field
for normal workloads.

If the static pods have a priority equal to or less than 7000, the Evacuation status should be
updated appropriately as follows, and the targeted pod should be terminated by the kubelet:

```yaml
apiVersion: v1alpha1
kind: Evacuation
metadata:
  finalizers:
    evacuation.coordination.k8s.io/instigator_nodemaintenance.k8s.io
  labels:
    app: critical-static-workload
  name: 08deef1c-1838-42a5-a3a8-3a6d0558c7f9-critical-static-workload
  namespace: critical-workloads
spec:
  podRef:
    name: critical-static-workload
    uid:  08deef1c-1838-42a5-a3a8-3a6d0558c7f9
  progressDeadlineSeconds: 1800
  evacuators:
    - evacuatorClass: kubelet.k8s.io
      priority: 10000
      role: controller
status:
  activeEvacuatorClass: kubelet.k8s.io
  activeEvacuatorCompleted: false
  evacuationProgressTimestamp: "2024-04-22T22:10:05Z"
  expectedEvacuationFinishTime: "2024-04-22T22:11:05Z" # now + terminationGracePeriodSeconds:
  failedEvictionCounter: 0
  message: "critical-static-workload is terminating the pod due to node maintenance (OS upgrade)."
  conditions: []
```

Once the pod is terminated and removed from the node, it should not be started on the node by
the kubelet again until the node maintenance is complete.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
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

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

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

- [x] Feature gate
  - Feature gate name: DeclarativeNodeMaintenance - this feature gate enables the NodeMaintenance API and node
    maintenance controller which creates `Evacuation`
  - Components depending on the feature gate: kube-apiserver, kube-controller-manager

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### Out-of-tree Implementation

We could implement the NodeMaintenance or Evacuation API out-of-tree first as a CRD.

The KEP aims to solve graceful termination of any pod in the cluster. This is not possible with a
3rd party CRD as we need an integration with core components.

- We would like to solve the lifecycle of static pods during a node maintenance. This means that
  static pods should be terminated during the drain according to `drainPlan`, and they should stay
  terminated after the kubelet restart if the node is still under maintenance. This requires
  integration with kubelet. See [kubelet: Static Pods](#kubelet-static-pods) for more details.
- We would like to improve the Graceful Node Shutdown feature. Terminating pods via NodeMaintenance
  will improve application safety and availability. It will also improve the reliability of the
  Graceful Node Shutdown feature. However, this also requires the kubelet to interact with a
  NodeMaintenance. See [kubelet](#kubelet) and
  [kubelet: Graceful Node Shutdown](#kubelet-graceful-node-shutdown) for more details.
- We would like to also solve the lifecycle of DaemonSet pods during the NodeMaintenance. Usually
  these pods run important or critical services. These should be terminated at the right time
  during the node drain. To solve this, integration with NodeMaintenance is required. See
  [DaemonSet Controller](#daemonset-controller) for more details.

Also, one of the disadvantages of using a CRD is that it would be more difficult to get real-word
adoption and thus important feedback on this feature. This is mainly because the NodeMaintenance
feature coordinates the node drain and provides good observability of the whole process.
Third-party components that are both cluster admin and application developer facing can depend on
this feature, use it, and build on top of it.

### Use a Node Object Instead of Introducing a New NodeMaintenance API

As an alternative, it would be possible to signal the node maintenance by marking the node object
instead of introducing a new API. But, it is probably better to decouple this from the node for
reasons of extensibility and complexity.

Advantages of the NodeMaintenance API approach:
- It allows us to implement incremental scale down of pods by various attributes according to a
  drainPlan across multiple nodes.
- There may be additional features that can be added to the NodeMaintenance in the future.
- It helps to decouple RBAC permissions and general update responsibility from the node object.
- It is easier to manage a NodeMaintenance lifecycle compared to the node object.
- Two or more different actors may want to maintain the same node in two different overlapping time
  slots. Creating two different NodeMaintenance objects would help with tracking each maintenance
  along with the reason behind it.
- Observability is better achieved with an additional object.

### Use Taint Based Eviction for Node Maintenance

To signal the start of the eviction we could simply taint a node with the `NoExecute` taint. This
taint should be easily recognizable and have a standard name, such as
`node.kubernetes.io/maintenance`. Other actors could observe the creations of such a taint and
migrate or delete the pod. To ensure pods are not removed prematurely, application owners would
have to set a toleration on their pods for this maintenance taint. Such applications could also set
`.spec.tolerations[].tolerationSeconds`, which would give a deadline for the pods to be removed by
the NoExecuteTaintManager.

This approach has the following disadvantages:
- Taints and tolerations do not support PDBs, which is the main mechanism for preventing voluntary
  disruptions. People who want to avoid the disruptions caused by the maintenance taint would have
  to specify the toleration in the pod definition and ensure it is present at all times. This would
  also have an impact on the controllers, who would have to pollute the pod definitions with these
  tolerations, even though the users did not specify them in their pod template. The controllers
  could override users' tolerations, which the users might not be happy about. It is also hard to
  make such behaviors consistent across all the controllers.
- Taints are used as a mechanism for involuntary disruption; to get pods out of the node for some
  reason (e.g. node is not ready). Modifying the taint mechanism to be less harmful
  (e.g. by adding a PDB support) is not possible due to the original requirements.
- It is not possible to incrementally scale down according to pod priorities, labels, etc.

### Names considered for the new API

These names are considered as an alternative to NodeMaintenance:

- NodeIsolation
- NodeDetachment
- NodeClearance
- NodeQuarantine
- NodeDisengagement
- NodeVacation

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
