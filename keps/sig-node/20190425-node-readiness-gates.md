---
title: Node Readiness Gates
authors:
  - "@andrewsykim"
  - "@vishh"
owning-sig: sig-node
participating-sigs:
  - sig-cloud-provider
  - sig-scheduling
reviewers:
  - "@dchen1107"
  - "@thockin"
  - "@vishh"
  - "@yastij"
approvers:
  - "@dchen1107"
  - "@thockin"
  - "@vishh"
editor: TBD
creation-date: 2019-04-25
last-updated: 2019-07-03
status: provisional
see-also:
  - "/keps/sig-network/0007-pod-ready++.md"
---

# Node Readiness Gates

## Table of Contents

   * [Node Readiness Gates](#node-readiness-gates)
      * [Table of Contents](#table-of-contents)
      * [Release Signoff Checklist](#release-signoff-checklist)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
      * [Proposal](#proposal)
      * [Design Details](#design-details)
         * [Determining Readiness via Pod Selectors](#determining-readiness-via-pod-selectors)
         * [Readiness Gates from External Sources](#readiness-gates-from-external-sources)
         * [Reporting Node Readiness via Node Status](#reporting-node-readiness-via-node-status)
         * [Limitations and Risks](#limitations-and-risks)
            * [Tolerating the Readiness Taint](#tolerating-the-readiness-taint)
            * [Unintended Pod Label Query for a Readiness Gate](#unintended-pod-label-query-for-a-readiness-gate)
         * [Graduation Criteria](#graduation-criteria)
            * [Alpha](#alpha)
            * [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
            * [Beta -&gt; GA Graduation](#beta---ga-graduation)
      * [Implementation History](#implementation-history)
      * [Alternatives](#alternatives)
         * [[Approach A] Tracking state of extensions via APIs](#approach-a-tracking-state-of-extensions-via-apis)
            * [Optional Extension - Track state of components along with Health](#optional-extension---track-state-of-components-along-with-health)
               * [Node Object extension](#node-object-extension)
               * [Sample Extension Status Object](#sample-extension-status-object)
         * [[Approach B] Using conditions](#approach-b-using-conditions)
         * [[Approach C] Support extensible health checks in the Kubelet](#approach-c-support-extensible-health-checks-in-the-kubelet)

Created by [gh-md-toc](https://github.com/ekalinin/github-markdown-toc)

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

As of Kubernetes v1.15, the health of extended node components like storage, network, device plugins, logging
or monitoring agents aren't taken into account while managing user pods on kubernetes nodes. This leads to a
broken contract between cluster administrators and end users where the latter ends up debugging issues like
lost metrics or logs, lack of network or storage connectivity, etc. On top of that lack of a generic abstraction
for node readiness leads to hacks across the k8s ecosystem to deal with the asynchronous & dynamic nature
of bootsrapping and managing k8s nodes.

This KEP presents an approach for solving the problem of extended node health or node readiness gates using existing APIs.

## Motivation

Since the early days of kubernetes, kubelet has been the primary source of readiness of a node where it queried
the health of critical system components like the docker daemon, network components, etc. Kubelet owns the node “Ready”
condition and the rest of Kubernetes cluster components rely on this (and a few other conditions) to infer that a node is “online”.

Kubernetes has evolved quite a bit since the early days and now a kubernetes “node” is comprised of many
system microservices that are all essential for a node to go “online” and start running user applications.
Examples include device & storage plugins, logging & monitoring agents, networking plugins and daemons like kube-proxy,
machine management daemons like Node Problem Detector, etc. Over the years, kubelet and the container runtimes
have evolved adequately to become highly reliable and so they become “ready” more consistently. But the other newer system
extensions may not be as mature yet, or may be slow to transition to “ready” for legitimate reasons.

The lack of a single unambiguous signal on when a “node” is “ready”, beyond just the kubelet or a container runtime
is causing several issues today.

* System components like Cluster Autoscaler and Scheduler have to combine several Node Conditions to evaluate a node’s
actual readiness. It is not straightforward to introduce new conditions into the policy.
* External systems (such as networking by the cloud provider) must often be configured before pods should be scheduled on a node.
* Cluster Autoscaler incorrectly assumes a GPU node to be “ready” prior to device plugins and/or device drivers
being installed which leads to spurious node additions and deletions thereby leading to custom logic in the autoscaler.
* Logs & metrics are lost if logging & monitoring agents are not online which can lead to sub-optimal (often unexpected)
user behavior.
* A node with offline kube-proxy can result in service outage in the event of an application update (via deployments for example).
* It is difficult for Node Lifecycle Controller and/or node repair systems to detect and remedy issues with extended system
components of the node since they lack a set of generic & easy to identify signals from those individual components on the node.

As Kubernetes becomes more and more customizable (or extensible) and gets deployed in mission critical systems,
it is imperative to provide fail safe mechanisms that are built into the system.

The remainder of this proposal introduces an approach for addressing the issue of node health extensibility using
node readiness gates.

### Goals

* introduce a mechanism that allows for customizable/extensible node readiness checks.

### Non-Goals

* improving time-to-readiness of nodes

## Proposal

Any solution that is chosen for addressing the set of problems stated above should meet the following design requirements:

* Be backwards compatible
* Be extensible to accommodate variations in kubernetes environments - the set of system microservices can vary across k8s deployments.
* Ideally, provide a single signal (not a policy) to cluster level components.
* Ideally, avoid introducing new API primitives.
* Support self-bootstrapping - scheduling system components (including the kubelet) even if the node is not logically ready.

## Design Details

### Determining Readiness via Pod Selectors

In this design, the indication of node readiness can be expressed by the readiness/health of specific pods running on that node. This approach
would leverage existing mechnanisms and well-known concepts to determine if a node is ready to receive pods.

A readiness taint will be used to track the overall readiness of a node. The kubelet, node lifecycle controller, other external controllers will
manage that taint.

The Node object will be extended to include a list of readiness gates like so:

```go
type NodeSpec struct {
  ...
  // Defines the list of readiness gates for a node
  // +optional
  ReadinessGates []NodeReadinessGate `json:"readinessGates,omitempty" protobuf:"bytes,7,opt,name=readinessGates"`
}

// NodeReadinessGate indicates a set of pods that must be ready on a given node before the
// node itself is considered ready. Pods listed under NodeReadinessGates should be system critical pods
// that should block other workloads from running on a node until it is ready.
type NodeReadinessGate struct {
   // The name of the readiness gate, mainly required as the patch merge key
   Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
   // A label query for pods representing this readiness gate.
   // Generally, labels here should point to a specific pod on a node.
   Selector LabelSelector `json:"selector,omitempty" protobuf:"bytes,2,opt,name=selector"`
   // The namespace query for the pods representing this readiness gate.
   Namespace string `json:"namespace,omitempty" protobuf:"bytes,3,opt,name=namespace"`
   // Indicates if this readiness gate should block the overall readiness of a node. Default: true
   // +optional
   BlocksReadiness bool `json:"blocksReadiness,omitempty" protobuf:"varint,4,opt,name=blocksReadiness"`
}
```

For each `NodeReadinessGate`, kubelet will fetch the state of all pods that match its label selector and take the readiness state of those pods into account.
If all pods represented by every `NodeReadinessGate` is ready (based on container state and readiness probes) then the node is also considered ready.
If no pods are found with the label selector then the node is consider to be "Not Ready".

The individual pods representing readiness gates are required to detect kubelet crash (or restart) quickly and reflect that in their own readiness states via probes.
The kubelet places a special meta readiness taint on a node object if 1 or more of those pods are not ready. If the Node Lifecycle Controller observes that all the pods selected
by `ReadinessGates` are ready, it will remove that taint immediately.

If `BlocksReadiness` is set to `false`, the overall readiness of a node is not blocked by the pods under this specific readiness gate. This is useful for non-critical pods
that should not block the readiness of a node but should be reported in the node status.

### Readiness Gates from External Sources

The `ReadinessGates` field can be used by external components to dictate node readiness since they can apply an arbitrary readiness gate with a reserved label selector
that should not be used by any pods on the cluster. A common usecase here is an external cloud provider that wants to ensure no pods are scheduled on a node until
networking for a given node is properly set up by the cloud provider. In this case, the cloud provider would apply any custom readiness gates to a node during registration
and a separate controller would remove the readiness gate when it sees that node's networking has been properly configured. No pods matching the label selector of that
readiness gate was required.

### Reporting Node Readiness via Node Status

The readiness of a node will be reported in the node's status subresource:

```go
type NodeStatus struct {
    ...
    // The status of each readiness gate by name
    // +optional
    Readiness []NodeReadiness `json:"readiness,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,12,rep,name=readiness"`
}

type NodeReadiness struct {
    // The name of the node readiness gate
    Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
    // Whether a node is ready based on its corresponding readiness gate
    Ready bool `json:"ready" protobuf:"varint,2,opt,name=ready"`
}
```

Updates for `node.Status.Readiness` will be done by the Node Lifecycle Controller.

### Limitations and Risks

#### Tolerating the Readiness Taint

One limitation in this design is that all system critical pods that should gate the readiness of a node must be updated to tolerate the readiness taint. Otherwise,
those pods cannot be scheduled on a node since the readiness taint would already be applied by the kubelet on registration. If users forget to apply this toleration,
they may accidentally gate the readiness of their nodes, preventing workloads from being scheduled there.

#### Unintended Pod Label Query for a Readiness Gate

If a Kubernetes user accidentally uses a label query that matches more pods than they thought, then node readiness may behave in a way they did not expect.
In the worse case scenario, if too many pods are selected for node readiness, the readiness for a node may flap as non-critical pods are rescheduled across the cluster.

### Graduation Criteria

#### Alpha

- the readiness taint is properly applied to a node if any of its readiness gate pods are not ready
- all functionality for readiness gates is gated by a feature gate (default off)

#### Alpha -> Beta Graduation

TODO: after alpha

#### Beta -> GA Graduation

TODO: after beta

## Implementation History

TODO: once accepted and KEP status is set to `implementable`.

## Alternatives

### [Approach A] Tracking state of extensions via APIs

For reasons similar to recent [node heartbeats revamp](/kep/sig-node/0009-node-heartbeat.md), individual node extensions will be expected to post their state via dedicated "[Lease](https://github.com/kubernetes/api/blob/master/coordination/v1beta1/types.go#L27)" API objects.

The Node API will include a new field that will track a list of node extensions that are expected on the node.
This list can be initially populated by the node controller when a node object is created or by the kubelet as it’s extensions register, or by individual extensions themselves (if they have sufficient privileges).

Here are some sample illustrations:

```yaml
type Node struct {
 …
 Spec:
  SystemExtensions []SystemExtension
...
}

type SystemExtension struct {
   Reference ObjectReference
   AffectsReadiness bool // Ignore non-critical node extensions from node readiness. Helps identify the overall set of extensions deterministically.
}
```

The following journey attempts to illustrate this solution a bit more in detail:

1. Kubelet starts up on a node and immediately taints the node with a special “kubernetes.io/MetaReadiness” taint with an effect of “NoSchedule” in addition to flipping it’s “Ready” condition to true (when appropriate).
2. Kubelet will not re-admit existing pods on the node that do not tolerate the “Readiness” taint. Kubelet will also not evict yet to be admitted pods to avoid causing disruptions. The node is still not fully functional yet.
3. The existing conditions that kubelet manages will continue to behave as-is (until they are transitioned to taints eventually).
4. Every node level extension component is expected to post it’s health continually via a dedicated Lease API that is understood by the the node controller.
5. The node controller will notice that the kubelet has restarted via the presence of the special taint. It will then wait for all expected node extension components to update their status prior to removing the taint.
6. Existing or newly deployed kubelet first class extensions (CSI or device plugins) on the node will register themselves with the kubelet and then update their state via a dedicated Lease object in a special (TBD) namespace.
7. Kubelet if newly registering will update the Node Object to include the device plugin as a required extension for determining node meta readiness (updates Node.Spec.SystemExtensions field as illustrated earlier) or the node controller can be updated to inject standard node extensions as part of Node object registration (possibly using webhooks).
8. Other non-kubelet extensions like kube-proxy, logging and monitoring agents will also register themselves (or get registered during Node object creation) as required addons by updating the Node object and periodically update their state.
9. For each node, the Node controller will keep track of the Lease objects that are included in Node.Spec.SystemExtensions field.
8. Once the node controller notices that all the required extensions have updated their health after the kubelet had placed a taint on the node, it will remove the taint. The node controller will continue to take into account the existing conditions as part of this design.
9. Once the taint is removed, kubelet will re-admit any existing pods and then transition to evicting any inadmissible pods.
10. If any of the system extensions fail to renew their lease (post health updates) for a TBD period (10s) of time, the node controller will taint the node with the same taint that the kubelet used originally. This prevents additional pods from being scheduled on to that node. If the node does not transition out of that state for a TBD duration (5 min?), the pods on the node will be evicted by the node controller.
11. Cluster Autoscaler (CA) will consider a new node to be logically booting until the special taint is removed.

Individual kubernetes vendors can easily customize their deployments and still use upstream Cluster Autoscaler and other ecosystem components that care about node health with this design.


#### Optional Extension - Track state of components along with Health

Each of the individual node extension will most likely want to expose domain specific status for instrospection purposes.
This status can be expressed as plugin specific CRDs which may be a good starting point.
Down the line though, having a generic abstraction(s) to represent the Status of different plugins or extensions will be necessary to empower the wider introspection & remediation ecosystem.

Assuming it is possible to define Status for some of the popular plugins like CSI, device plugins, kube proxy, logging & monitoring agents, the Lease API based approach can be extended as follows to easily track status of extensions in addition to Health.

1. Embed the "Lease" API object in an extension specific Status object. An alternative is to have a local Lease object with the same name as the Status object.
2. Similar to the workflow described above, the node controller, kubelet or extensions will register their status object as an extension via the Node.Spec.SystemExtensions field.
3. The Node Controller will be updated to extract the Lease object (or identifying a separate Lease object by name in the local namespace) and influence health of extensions similar to the workflow described above.
4. Ecosystem components like Monitoring, repair/remediation systems will be able to deterministically identify and expose or act on the status of the extensions.

Here are some API illustations.

##### Node Object extension

```yaml
type ExtensionFoo struct {
 TypeMeta
 ObjectMeta
 Spec ExtensionFooStatus
}

type ExtensionFooStatus {
  v1beta1.LeaseSpec
  OtherFooStatus interface{}
}

type Node struct {
 …
 Spec:
  SystemExtensions []SystemExtension
...
}

type SystemExtension struct {
   Reference ObjectReference
   AffectsReadiness bool // Ignore non-critical node extensions from node readiness
}
```

##### Sample Extension Status Object

```yaml
type DevicePluginExtension struct {
 TypeMeta
 ObjectMeta
 Status DevicePluginExtensionStatus
}

type DevicePluginExtensionStatus struct {
  LeaseSpec
  ComputeResource struct{ // embedded for brevity
    Name: “foo.com/bar”
    Quantity: x // Sample fields for illustration purposes
    UnHealthy: y
    DeviceIDs []string
  }
}

```

One of the goals for this proposal is to evaluate the viability and usefullness of combining Status and Health for extensions.

### [Approach B] Using conditions

This design continues the existing pattern of expressing health of various node features and components via Conditions.
For example the Kubelet uses conditions heavily and the Node Problem Detector also uses conditions. Conditions allow for recording heartbeats already.

This design is similar to the approach `A` where instead of recording state to a separate object, kubelet and other system components will record their state via Conditions.
The node controller will be extended to support a configurable policy on what Conditions influence the meta readiness of a node.

The kubelet and node controller will use a taint similar to the one illustrated in approach `A` to signal node meta readiness.

The scheduler needs to understand that certain special pods (system pods) can be scheduled to nodes even if the node isn’t logically ready.
There is no canonical construct today to differentiate system pods from regular pods.

The main drawback of this approach is the explosion on heartbeats (updates to Condition objects) to the Node object from various components which was the reason why a new heartbeat mechanism was introduced for the kubelet in the first place.
Another drawback is that Conditions are not extensible - it is not possible to include extension specific information as part of conditions leading to leaking the state of extensions to other sub-objects in the Node API object eventually.

### [Approach C] Support extensible health checks in the Kubelet

This approach assumes kubelet to be the central arbitrator of Node health.

Every node extension needs to register a canonical Readiness endpoint (similar to pod readiness/handlers) with the kubelet via the Node API object.

The kubelet will then poll health of individual extensions periodically and reflect the state of the individual extensions via a taint similar to the solutions mentioned above.
The main difference here is that the node controller doesn’t have to track the state of individual extensions since the kubelet is already doing it.

Optionally, the kubelet can reflect the state of each extension via a dedicated condition.

Upon restart, Kubelet will wait for one successful health check run across all registered extensions prior to removing the special taint.

This approach requires that all extensions are accessible to the kubelet via host network interfaces (barring the option of using exec binaries per addon that will suffer a distribution problem).

This approach also lacks the ability to include more state information about each extension in the API object - aka it’s limited to only health tracking. The load on the kubelet could be much higher due to the proliferation of system extensions. Every extension is required to build a canonical healthz endpoint (which is a best practice by itself).

