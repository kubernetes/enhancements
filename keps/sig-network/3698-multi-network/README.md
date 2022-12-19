# KEP-3698: Multi-Network

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Personas](#personas)
    - [Cluster operator](#cluster-operator)
    - [Application developer](#application-developer)
  - [Terminology](#terminology)
  - [User Stories](#user-stories)
    - [Story #1](#story-1)
    - [Story #2](#story-2)
    - [Story #3](#story-3)
    - [Story #4](#story-4)
    - [Story #5](#story-5)
    - [Story #6](#story-6)
    - [Story #7](#story-7)
    - [Story #8](#story-8)
  - [Requirements](#requirements)
    - [Phase I (base API and reference in Pod)](#phase-i-base-api-and-reference-in-pod)
    - [Phase II (access control, downward API)](#phase-ii-access-control-downward-api)
    - [Phase III (basic Kubernetes features integration)](#phase-iii-basic-kubernetes-features-integration)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Background](#background)
  - [Network](#network)
  - [CNI usage models](#cni-usage-models)
    - [Standalone](#standalone)
    - [Agent-based](#agent-based)
- [Phase I Design Details](#phase-i-design-details)
  - [Overview](#overview)
  - [New Resources](#new-resources)
    - [PodNetwork](#podnetwork)
      - [Status Conditions](#status-conditions)
      - [Provider](#provider)
      - [Enabled](#enabled)
      - [IPAM](#ipam)
      - [InUse state](#inuse-state)
      - [Mutability](#mutability)
      - [Lifecycle](#lifecycle)
      - [Validations](#validations)
    - [PodNetworkAttachment](#podnetworkattachment)
      - [Status Conditions](#status-conditions-1)
      - [InUse indicator](#inuse-indicator)
      - [Mutability](#mutability-1)
      - [Lifecycle](#lifecycle-1)
      - [Validations](#validations-1)
    - [Resources relations](#resources-relations)
  - [Default PodNetwork](#default-podnetwork)
    - [Availability](#availability)
    - [Automatic creation](#automatic-creation)
    - [Manual creation](#manual-creation)
    - [Network Migration](#network-migration)
  - [PodNetwork Controller](#podnetwork-controller)
  - [Feature gate](#feature-gate)
  - [Attaching PodNetwork to a Pod](#attaching-podnetwork-to-a-pod)
    - [Static validations](#static-validations)
    - [Active validations](#active-validations)
    - [Auto-population](#auto-population)
    - [Status](#status)
    - [DRA integration (alternative)](#dra-integration-alternative)
  - [API server changes](#api-server-changes)
  - [Scheduler changes](#scheduler-changes)
  - [Endpointslice controller changes](#endpointslice-controller-changes)
  - [Kubelet changes](#kubelet-changes)
  - [CRI changes](#cri-changes)
    - [Pod Creation](#pod-creation)
    - [Pod Status](#pod-status)
    - [CNI API](#cni-api)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

One of the main requirements for Kubernetes networking today is to enable
connectivity between `Pods` in a cluster. This facility satisfies a large number
of Kubernetes users, but it isn't sufficient for some important cases:

* applications leveraging different isolated networks exposed through different
  interfaces
* applications leveraging performance-oriented interfaces (e.g. `AF_XDP`,
  `memif`, `SR-IOV`)
* applications requiring support for protocols not yet supported by Kubernetes

Here we propose a solution that enables multiple network interfaces to be
defined in the `Pod` specification to provide a standard and clear mechanism to
implement these more complex configurations.

## Motivation

We want to have a common API allowing us to define a catalog of different
networks in the Kubernetes cluster. It would allow attaching a pod to one or
several networks via a given type of interface depending on its connectivity
or performance needs.

### Goals

* Define user stories and requirements for the Multi-Network effort in Kubernetes.
* Introduce API object to kubernetes describing networks Pod can attach to.
* Evolve current Kubernetes networking model to support multiple networks,
defining the new model in a backwards compatible way.
* Integrate this API with the existing core objects.
* Define “reference implementation” that can be used in Kubernetes CI.
* Define conformance tests.


### Non-Goals

Define the CNI implementation.

## Proposal

### Personas

#### Cluster operator

This role will be responsible for defining and managing PodNetworks (see below)
that properly describe the infrastructure available for the cluster, namespace
and workloads. This persona can define which users can “attach” to a specific
PodNetwork.

#### Application developer

**Application developer** is the consumer of PodNetwork via referencing them in
their workloads. Application developers usually will not create or remove the
PodNetwork on their own.

### Terminology

* **PodNetwork** - name of the API object that will represent a network in
  Kubernetes cluster
* **Cluster Default PodNetwork** - This is the PodNetwork attached to the Pod
  when no additional networking configuration is provided in Pod spec.
* **Primary PodNetwork** - This is the PodNetwork inside the Pod which interface
  is used for the default gateway.
* **KCM** - kube-controller-manager is a component containing most core kubernetes 
  controllers


### User Stories

All user stories represent the type of use cases the multi-networking API should
be able to support. References to technologies or exact products does not
indicate that this API will directly support them. User stories can be
overlapping in some areas. We want to document the concrete use cases from an
end-user standpoint -- they may (and should!) boil down to a few common
primitives listed in the below requirements section.


#### Story #1
As a Cloud Native Network Function (CNF) vendor my workloads deals with 2 types
of network traffic: control plane and dataplane. For regulatory compliance I have
to isolate these 2 traffic types. To achieve this I require 2 separate network
interfaces inside my Pod. The underlying implementation (outside the Pod) has to
ensure isolation on either Layer-2 or Layer-3.

<p align="center">
  <img src="mn-story-1.png?raw=true" alt="multi-network story 1 network L2 isolation"/>
</p>

#### Story #2
As a Cloud Native Network Function (CNF) vendor I require a HW-based interface
(e.g. SRIOV VF) to be provisioned to my workload Pod. I need to leverage that HW
for performance purposes (high bandwidth, low latency), that my user-space
application (e.g. DPDK-based) can use. The VF will not use the standard netdev
kernel module. The Pod’s scheduling to the nodes should be based on hardware
availability (e.g. devicePlugin or some other way). This interface might not
support some Kubernetes functionality like e.g. Services, I am willing to give
that up for performance.

<p align="center">
  <img src="mn-story-2.png?raw=true" alt="multi-network story 2 HW interface"/>
</p>

#### Story #3
I have implemented my Kubernetes cluster networking using a virtual switch. In
this implementation I am capable of creating isolated Networks. I need a means
to express to which Network my workloads connect to.

<p align="center">
  <img src="mn-story-3.png?raw=true" alt="multi-network story 3 virtual switch isolation"/>
</p>

#### Story #4
As a Virtual Machine -based compute platform provider that I run on top of
Kubernetes and Kubevirt I require multi-tenancy. The isolation has to be
achieved on Layer-2 for security reasons.

<p align="center">
  <img src="mn-story-4.png?raw=true" alt="multi-network story 4 Kubevirt VMs"/>
</p>

#### Story #5
As a platform operator I need to connect my on-premise networks to my workload
Pods. I need to have the ability to represent these networks in my Kubernetes
cluster in such a way that I can easily use them in my workloads.

<p align="center">
  <img src="mn-story-5.png?raw=true" alt="multi-network story 5 On-Premise Network representation"/>
</p>

#### Story #6
As a Kubernetes cluster operator I wish to isolate workloads based on
namespaces and network access by connecting to only a non- Cluster Default PodNetwork.
Those workloads should have the same level of Kubernetes functionality:
Services, NetworkPolicies, access to Kubernetes API.

<p align="center">
  <img src="mn-story-6.png?raw=true" alt="multi-network story 6 namespace networks profiles"/>
</p>

#### Story #7
As a “Power User” with admin privileges I wish to have the ability to modify my
Pod network namespace without any restrictions. I am aware that by doing this I
might break established contracts for the kubernetes features.

#### Story #8
As Cluster operator I wish to manage my cluster’s bandwidth usage on a per-pod
basis, simultaneously preserving all other Pod networking configuration the same
across those Pods.

<p align="center">
  <img src="mn-story-10.png?raw=true" alt="multi-network story 10 per-pod connection differentiation"/>
</p>

### Requirements

Below requirements are divided into phases. Each phase will be in its own design
section of this KEP that will be defined and implemented incrementally.

#### Phase I (base API and reference in Pod)
1. Multi-network should not change the behavior of existing clusters where no 
multi-network constructs are configured/used.
2. Multi-network needs to be represented explicitly as an API construct in
Kubernetes as it fundamentally ties in with other APIs in the Kubernetes
networking model, extending their functionality.
   * We will introduce an PodNetwork API object that represents the existing
     infrastructure’s networking.
3. PodNetwork functions as a handle that decouples the implementation
(i.e. infrastructure/network admins) from the consumer (cluster user). This
abstraction follows the Kubernetes model of decoupling intent from implementation.
4. PodNetwork shall not define any implementation specific parameters in its
specification.
5. PodNetwork shall provide cluster generic options only.
6. PodNetwork can reference to implementation-specific parameters.
7. PodNetwork is the object Application developer will reference in their
workflows (e.g. Pod, Service).
8. Cluster Default PodNetwork is the PodNetwork the cluster has been created
with initially.
9. Cluster Default PodNetwork cannot be removed, but can be replaced.
10. Pods can reference one or more PodNetworks when trying to attach to them.
    * This list is explicit and all the PodNetworks that has to be available in
      the Pod has to be listed.
11. When Pods does not specify any reference to PodNetwork it connects to Cluster
Default PodNetwork (network the cluster has been created with).
12. Pod shall be able to provide additional configuration on how it attaches to
a PodNetwork.
13. Every Pod connected to specific PodNetwork must have connectivity within
that network across the Cluster.
14. A Pod connected to specific PodNetwork may or may not have cross 
connectivity between different PodNetworks.
15. Pods attached to PodNetworks are connected to each other in a manner 
defined by the PodNetwork implementation.

#### Phase II (access control, downward API)
16. Pods access to PodNetwork is controlled via RBAC configuration.
17. Implementation-agnostic PodNetwork Interface information (e.g. PodNetowrk
name, IP address, etc.) for each attachment will be exposed to runtime Pod (via 
e.g. environment variables, downward API etc.).

#### Phase III (basic Kubernetes features integration)
18. Kubelet network-based probing can be optionally enabled for the non- Cluster
    Default PodNetwork attachments to Pod.
19. Kubernetes API can be optionally accessible via the non- Cluster Default
    PodNetwork attachments to Pod.
20. Kubernetes API can optionally reach the Pod via the non- Cluster Default
    PodNetwork attachments.
21. All PodNetwork attachments to Pod are optionally able to provide Service
and NetworkPolicy functionality.

### Risks and Mitigations

N/A

## Background

### Network
Network is a very overloaded term, and for many might have different meaning: it
might be represented as a VLAN, or a specific interface in a Node, or identified
as a unique IP subnet (a CIDR). In this design we do not want to limit to one
definition, and ensure that we are flexible enough to satisfy anyone's definition.

### CNI usage models
For the purpose of this design we want to call out the 2 models on how CNI API
is used by various implementations.

#### Standalone
This is the simplest and direct mode. Here we leverage in full the CNI API and
all its configuration capabilities. All parameters required to set up a network
inside a Pod network namespace are provided from the default kubelet API and the
parameters of the local conflist file present in CNI configuration path (default
/etc/cni/net.d/). The binary configures everything based on just these parameters
and does not require any additional connection with Kubernetes API.

<p align="center">
  <img src="cni-stand.png?raw=true" alt="standalone CNI mode"/>
</p>

#### Agent-based
In this model, the CNI implementation is based on a fully fledged agent that
usually runs as a hostNetwork Pod on each of the cluster Nodes. That agent has
access to the Kubernetes API with RBAC defined permissions. The CNI binary is as
simple as possible and communicates with the agent via local connections (e.g.
socket file). Most of the logic is inside the agent. This agent does not rely on
the CNI conflist, but gathers all the required data from Kubernetes API on its
own (e.g. pulls the whole Pod object).

<p align="center">
  <img src="cni-agent.png?raw=true" alt="agent-based CNI mode"/>
</p>

## Phase I Design Details
### Overview
This design adds information to Pods to which “networks” (plural) it attaches to.
Then the Pod scheduler will be able to understand which “network” is available
and which is not in the cluster.

For that purpose this design introduces 2 new objects: *PodNetwork* and
*PodNetworkAttachment*. *PodNetwork* is a core component that functions as a
representation of a specific network in Kubernetes cluster, as well as a future
handle for other objects to reference.
The *PodNetworkAttachment* is used as an additional Pod-level means to parameterize
Pods’ network attachments.

Lastly we wish to introduce a new controller for these objects, to support the
proposed life cycle.

### New Resources
#### PodNetwork
This KEP will add a *PodNetwork* object. This will be a core API object so that
we can reference it in other Kubernetes core objects, like Pod. The core design
principles for this object are:
1. *PodNetwork*’s form is generic and further specification is
implementation-specific - is the network L2 or L3, or is it handled by CRI, CNI
or a controller, shall the network respect isolation or not, should all be the
decision of implementation. We want to provide an API handle for the rest of the
core or extended objects in Kubernetes.
1. *PodNetworks* may have overlapping subnets across them.
1. This enhancement is fully backward compatible and does not require any
additional configuration from existing deployments to continue functioning.

*PodNetwork* object is described as follows:
```go
// +genclient
// +genclient:nonNamespaced
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// PodNetwork represents a logical network in Kubernetes Cluster.
type PodNetwork struct {
        metav1.TypeMeta   `json:",inline"`
        metav1.ObjectMeta `json:"metadata,omitempty"`

        Spec   PodNetworkSpec   `json:"spec"`
        Status PodNetworkStatus `json:"status,omitempty"`
}

// PodNetworkSpec contains the specifications for podNetwork object
type PodNetworkSpec struct {

        // Enabled is used to administratively enable/disable a PodNetwork.
        // When set to false, PodNetwork Ready condition will be set to False.
        // Defaults to True.
        //
        // +optional
        // +kubebuilder:default=true
        Enabled bool `json:"enabled,omitempty"`

        // ParametersRefs points to the vendor or implementation specific parameters
        // objects for the PodNetwork.
        //
        // +optional
        ParametersRefs []ParametersRef `json:"parametersRefs,omitempty"`

        // Provider specifies the provider implementing this PodNetwork.
        //
        // +kubebuilder:validation:MinLength=1
        // +kubebuilder:validation:MaxLength=253
        // +optional
        Provider string `json:"provider,omitempty"`
}

// ParametersRef points to a custom resource containing additional
// parameters for thePodNetwork.
type ParametersRef struct {
        // Group is the API group of k8s resource e.g. k8s.cni.cncf.io
        Group string `json:"group"`

        // Kind is the API name of k8s resource e.g. network-attachment-definitions
        Kind string `json:"kind"`

        // Name of the resource.
        Name string `json:"name"`

        // Namespace of the resource.
        // +optional
        Namespace string `json:"namespace,omitempty"`
}

// PodNetworkStatus contains the status information related to the PodNetwork.
type PodNetworkStatus struct{
        // Conditions describe the current conditions of the PodNetwork.
        //
        // Known condition types are:
        // * "Ready"
        // * "ParamsReady"
        //
        // +optional
        // +listType=map
        // +listMapKey=type
        // +kubebuilder:validation:MaxItems=5
        Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```
Generic example:
```yaml
apiVersion: v1
kind: PodNetwork
metadata:
  name: dataplane
spec:
  enabled: true
  provider: "foo.io/bar"
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: "2022-11-17T18:38:01Z"
    status: "True"
    type: Ready
```
Example with implementation-specific parameters:
```yaml
apiVersion: v1
kind: PodNetwork
metadata:
  name: dataplane
spec:
  enabled: true
  provider: "k8s.cni.cncf.io/multus"
  parametersRefs:
  - group: k8s.cni.cncf.io
    kind: network-attachment-definitions
    name: parametersA
    namespace: default
  - group: k8s.cni.cncf.io
    kind: network-attachment-definitions
    name: complementaryParametersB
    namespace: default
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: "2022-11-17T18:38:01Z"
    status: "True"
    type: Ready
  - lastProbeTime: null
    lastTransitionTime: "2022-11-17T18:38:01Z"
    status: "True"
    type: ParamsReady
```
##### Status Conditions
The PodNetwork object will use following conditions:
* **Ready** - indicates that the PodNetwork object is correct (validated) and
all other conditions are set to “true”. This condition will switch back to
“false” if ParamsReady condition is “false”. Pods cannot be attached to a
PodNetwork that is not Ready. This condition does not indicate readiness of
specific PodNetwork on a per Node-basis. Following are the error reasons for
this condition:

| Reason name              | Description                                                                                                                      |
|--------------------------|----------------------------------------------------------------------------------------------------------------------------------|
| ParamsNotReady           | The ParamsReady condition is not present or has “false” value. This can only happen when the “parametersRefs” field has a value. |
| AdministrativelyDisabled | The PodNetwork’s Enabled field is set to false.                                                                                  |

* **ParamsReady** - indicates that the objects specified in the “parametersRefs”
field are ready for use. The implementation (effectively the owner of the
specified objects) is responsible for setting this condition to “true” after
having performed whatever checks the implementation requires (such as validating
the CRs and checking that the network they reference is ready). The “Ready”
condition is dependent on the value of this condition when the “parametersRefs”
field is not empty. The available “reasons” for this condition are implementation
specific. When multiple references are provided in the “parametersRefs” field,
it is implementation responsibility to provide accurate status for all the listed
objects using this one condition.

The conditions life-cycle will be handled by the PodNetwork controller described
below.

##### Provider
The provider field gives the ability to uniquely identify what implementation
(provider) is going to handle specific instances of PodNetwork objects. The value
has to be in the form of an url. It is the implementer’s decision on how this field
is going to be leveraged. They will decide how they behave when this field is
empty and what specific value they are going to respect. This will dictate if a
specific implementation can co-exist with other ones in the same cluster.

We have considered using classes (similar to GatewayClass etc.), but we do not
expect any extra action for a PodNetwork to take place for specific implementation.
PodNetwork already points to a custom resource, which implementation can hook on
for any specific extra configuration. At this point of time we consider using
classes as overkill, and in future if such need arises, we can reuse the provider
field for that purpose.

##### Enabled
The Enabled field is created to allow proper migration from an existing PodNetwork.
When set to false no new Pods can be attached to that PodNetwork.

##### IPAM
In this design we will not provide any specification for IPAM handling for
PodNetworks. This will be specified in the following phases.

##### InUse state
The PodNetwork object can be referenced by at least one Pod or
PodNetworkAttachment. When this is the case, the PodNetwork cannot be deleted.
This will be maintained by the PodNetwork Controller and enforced via a finalizer.\
When identifying Pods using PodNetwork for this purpose, the controller will
filter out Pods that are in Succeeded or Failed state.

##### Mutability
The PodNetwork object will be immutable, except for the *Enabled* field.

##### Lifecycle
A PodNetwork will be in following phases:
1. **Created** - when the user just created the object and it does not have any
conditions.
2. **NotReady** - when PodNetwork’s Ready condition is false. Pods can reference
such a PodNetwork, but will be in Pending state until the PodNetwork becomes Ready.
3. **Ready** - when validation of the PodNetwork succeeded and Ready condition
is set to true. Here Pods can start attaching to a PodNetwork.
4. **InUse** - when there is a Pod or PodNetworkAttachment that references a
given PodNetwork. PodNetwork deletion is blocked by a finalizer when InUse.
5. **Disabled** - when the user sets the Enabled field to false. We will mark
such PodNetwork NotReady, and no new Pods will be able to attach to such PodNetwork.

##### Validations
We will introduce following validations for this object:
* Prevent mutation
* Ensure provider is a string
* Ensure listed parametersRef object fields are strings

#### PodNetworkAttachment
The other object this KEP will add is PodNetworkAttachment object. This object
will give providers the ability of a more detailed or per-Pod oriented
configuration of the Pod attachment. In comparison, the PodNetwork object is a
global representation of the network, and PodNetworkAttachment provides optional,
more detailed configuration on a per-Pod level.
PodNetworkAttachment object will have 2 functions:
* Provides ability to configure individual Interface of a Pod and still reference
same PodNetwork
* Can contain status for an individual interface of a Pod

This is a namespaced object. It is described as follows:
```go
// +genclient
// +genclient:Namespaced
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced

// PodNetworkAttachment provides optional pod-level configuration of PodNetwork.
type PodNetworkAttachment struct {
        metav1.TypeMeta   `json:",inline"`
        metav1.ObjectMeta `json:"metadata,omitempty"`

        Spec   PodNetworkAttachmentSpec   `json:"spec,omitempty"`
        Status PodNetworkAttachmentStatus `json:"status,omitempty"`
}

// PodNetworkAttachmentSpec is the specification for the PodNetworkAttachment resource.
type PodNetworkAttachmentSpec struct {
        // PodNetworkName refers to a PodNetwork object that this PodNetworkAttachment is
        // connected to.
        //
        // +required
        PodNetworkName string `json:"podNetworkName"`

        // ParametersRefs points to the vendor or implementation specific parameters
        // object for the PodNetworkAttachment.
        //
        // +optional
        ParametersRefs []ParametersRef `json:"parametersRefs,omitempty"`
}

// PodNetworkAttachmentStatus is the status for the PodNetworkAttachment resource.
type PodNetworkAttachmentStatus struct {
        // Conditions describe the current conditions of the PodNetworkAttachment.
        //
        // Known condition types are:
        // * "Ready"
        // * "ParamsReady"
        //
        // +optional
        // +listType=map
        // +listMapKey=type
        // +kubebuilder:validation:MaxItems=5
        Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```
Example:
```yaml
apiVersion: v1
kind: PodNetworkAttachment
metadata:
  name: pod1
  namespace: default
spec:
  podNetworkName: "dataplane"
  parametersRefs:
  - group: k8s.cni.cncf.io
    kind: podParams
    name: parametersA
    namespace: default
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: "2022-11-17T18:38:01Z"
    status: "True"
    type: Ready
  - lastProbeTime: null
    lastTransitionTime: "2022-11-17T18:38:01Z"
    status: "True"
    type: ParamsReady
```
##### Status Conditions
The PodNetworkAttachment will follow a similar life cycle as the PodNetwork object. One change will be a new error reason for Ready condition:
* **Ready** - indicates that the PodNetworkAttachment object is correct (validated) and ParamsReady condition is set to “true”, including the referenced PodNetwork’s Ready condition. This condition will switch back to “false” if any of the above conditions change to “false”. Pods cannot be attached to a PodNetworkAttachment that is not Ready. This condition does not indicate readiness of specific PodNetworkAttachment on a specific Node. Following are the error reasons for this condition:

| Reason name        | Description                                                                                                                            |
|--------------------|----------------------------------------------------------------------------------------------------------------------------------------|
| ParamsNotReady     | The ParamsReady condition is not present or has “false” value. This error can only happen when the “parametersRefs” field has a value. |
| PodNetworkNotReady | The referenced PodNetwork object Ready condition has “false” value.                                                                    |

The conditions life-cycle will be handled by the PodNetwork controller described below.

##### InUse indicator
The PodNetworkAttachment object can be referenced by at least one Pod. When this
is the case, the PodNetworkAttachment cannot be deleted. This will be maintained
by the PodNetwork Controller and enforced via a finalizer.\
When identifying Pods using PodNetworkAttachment for this purpose, the controller
will filter out Pods that are in Succeeded or Failed state.

##### Mutability
The PodNetworkAttachment object will be immutable.
The API server will provide the admission control for that.

##### Lifecycle
The PodNetworkAttachment will be in following phases:
1. Created - when the user just created the object and it does not have any
conditions. 
2. NotReady - when PodNetworkAttachment’s Ready condition is false. Pods can
reference such PodNetworkAttachment, but will be in Pending state until the
PodNetwork becomes Ready. 
3. Ready - when validation of the PodNetworkAttachment succeeded and Ready
condition is set to true, here Pod can start attaching to a PodNetworkAttachment. 
4. InUse - when there is a Pod that references a given PodNetworkAttachment.
PodNetworkAttachment deletion is blocked by a finalizer when InUse.

##### Validations
We will introduce following validations for this object:
* Prevent mutation
* Ensure podNetworkName is a string
* Ensure listed parametersRef object fields are strings
This validation will be performed in the API server.

#### Resources relations
The relation of the new objects to Pod and between each other is described in
this diagram:
<p align="center">
  <img src="obj-reletion.png?raw=true" alt="introduced objects relationship"/>
</p>

The arrows define which object references what object. Additionally:
* PodNetworkAttachment must reference exactly 1 PodNetwork
* PodNetwork can be referenced by multiple PodNetworkAttachments
* Pod can reference multiple PodNetworks or PodNetworkAttachments
* Pod has to reference at least 1 PodNetwork or PodNetworkAttachment (for
backward compatibility, when nothing specified, “default” PodNetwork is
auto-populated, see details below)
* For specific PodNetwork a Pod can either reference that PodNetwork or
PodNetworkAttachment (referencing that PodNetwork), but not both at the same time
* PodNetwork can be referenced by multiple Pods
* PodNetworkAttachment can be referenced by multiple Pods
* PodNetwork can reference none, one or multiple parameters CRs
* PodNetworkAttachment can reference none, one or multiple parameters CRs
* Specific parameter CR can be referenced by one or more PodNetwork or
PodNetworkAttachment, this is implementation specific decision

### Default PodNetwork
We will introduce a “default” PodNetwork. This will be the Cluster Default
PodNetwork. This PodNetwork will represent today's kubernetes networking done
for Pods. This PodNetwork is characterized as:
* Will always be created during cluster creation (similarly to “default” kubernetes
Namespace)
* All Pods not referencing any PodNetwork will connect to the “default” PodNetwork
* This PodNetwork will be named “default”
* The “default” PodNetwork must be available on all nodes
* Until it is created, all kubelets will report “Default PodNetwork not found”
for their respective Nodes.

#### Availability
Considering “default” PodNetwork is critical for cluster functionality, we will
provide special handling for it when it is being deleted. On deletion events,
we will recreate it, so that it never has the deletionTimestamp field set. This
is going to be handled by the API server.

In case “default” PodNetwork references any additional params in ParametersRef
field, the implementer is responsible for those objects' availability.
“default” PodNetwork will become not Ready if ParamsReady condition is “false”.

#### Automatic creation
The PodNetwork controller will automatically create the “default” PodNetwork.
The values set in it will be determined by the arguments passed to KCM.

#### Manual creation
We will introduce a new flag to KCM named **disable-default-podnetwork-creation**
that will disable the automatic creation of the “default” PodNetwork. When
specified, the Cluster Operator will be required to create the “default”
PodNetwork. Until it is created, all kubelets will report “Default PodNetwork
not found” for their respective Nodes.

#### Network Migration
This Phase will not implement the “default” PodNetwork migration procedure. It
will arrive in a later time with a next phase. Following is the initial idea how
it could work.\
To change the configuration of Default PodNetwork, we will expose a new field in
the Node spec object called overrideDefaultPodNetwork. This field will allow you
to change what is the default PodNetwork on a per-Node basis. This field will be
mutable. When kubelet is going to report the presence of the Default PodNetwork,
it will first look at this field, and then fallback to the “default” name.\
For the migration process, the installer will have to set that field to the new
PodNetwork, and when the “default” PodNetwork is no longer “InUse”, it can be
deleted and replaced with a new version. At this point the installer will have
to clear up the Node’s overrideDefaultPodNetwork field.

### PodNetwork Controller
This KEP will introduce a new controller in KCM. Its main function will be:
* Handle PodNetwork and PodNetworkAttachment conditions
* Handle PodNetwork and PodNetworkAttachment finalizer to prevent deletion when
InUse
* Automatically create the “default” PodNetwork

### Feature gate
This feature will introduce a new feature gate to the --feature-gates argument
and will be named **MultiNetwork**. All changes proposed in this design will be
behind this gate.

### Attaching PodNetwork to a Pod
We will extend the Pod spec with a new field Pod.PodSpec.Networks. It will be a
list allowing attaching PodNetworks to a Pod. This list is explicit, and only
the listed PodNetworks will be attached to the Pod. When the Networks field is
not specified, Cluster Default PodNetwork will be attached (and set in that field)
to the Pod (to keep backward compatibility). Proposed changes are as follows:
```go
// PodSpec is a description of a pod.
type PodSpec struct {
[...]
        // Networks is a list of PodNetworks that will be attached to the Pod.
        //
        // +kubebuilder:default=[{podNetworkName: “default”}]
        // +optional
        Networks []Network `json:"networks,omitempty"`
}

// Network defines what PodNetwork to attach to the Pod.
type Network struct {
        // PodNetworkName is name of PodNetwork to attach
        // Only one of: [PodNetworkName, PodNetworkAttachmentName] can be set
        //
        // +optional
        PodNetworkName string `json:"podNetworkName,omitempty"`

        // PodNetworkAttachmentName is name of PodNetwork to attach
        // Only one of: [PodNetworkName, PodNetworkAttachmentName] can be set
        //
        // +optional
        PodNetworkAttachmentName string `json:"podNetworkAttachmentName,omitempty"`

        // InterfaceName is the network interface name inside the Pod for this attachment.
        // This field functionality is dependent on the implementation and its support
        // for it.
        // Examples: eth1 or net1
        //
        // +optional
        InterfaceName string `json:"interfaceName,omitempty"`

        // IsDefaultGW4 is a flag indicating this PodNetwork will hold the IPv4 Default
        // Gateway inside the Pod. Only one Network can have this flag set to True.
        // This field functionality is dependent on the implementation and its support
        // for it.
        //
        // +optional
        IsDefaultGW4 bool `json:"isDefaultGW4,omitempty"`

        // IsDefaultGW6 is a flag indicating this PodNetwork will hold the IPv6 Default
        // Gateway inside the Pod. Only one Network can have this flag set to True.
        // This field functionality is dependent on the implementation and its support
        // for it.
        //
        // +optional
        IsDefaultGW6 bool `json:"isDefaultGW6,omitempty"`
}
```

#### Static validations
We will perform static validation of the provided spec in the API Server,
alongside the current Pod spec checks. These errors will be provided to the user
immediately when they try to create Pod.
This will include:
* Ensure a single Pod references a given PodNetwork only 1 time
* IsDefaultGW4 and IsDefaultGW6 uniqueness for “true” value across multiple
“Network” objects
* InterfaceName uniqueness across multiple “Network” objects
* InterfaceName naming constraints for Linux and Windows
* Ensure Network objects are not specified when hostNetwork field is set

The 1 PodNetwork per Pod restriction is coming from our current lack of details
for future Multi-Networking requirements (e.g. Service support), and can be
changed in future when we will discuss them.

#### Active validations
Beside the above (static validations), we will perform additional active
validations, inside the  scheduler, that require queries for other objects
(e.g. PodNetwork). These errors will be presented as Pod Events, and the Pod
will be kept in Pending state until the issues are resolved. We will do the
following validation:
* Referenced PodNetwork or PodNetworkAttachment is present
* Referenced PodNetwork or PodNetworkAttachment is Ready
* Referenced PodNetworkAttachment, that is referencing a PodNetwork, follows the
rule of: single Pod references a given PodNetwork only 1 time

#### Auto-population
When networks field is not set, and hostNetwork is not set, we will
auto-populate this field with following values:
```yaml
networks:
- podNetworkName: default
```

This is to ensure backward compatibility for clusters that will not use
PodNetwork explicitly.

#### Status
All the IP addresses for attached PodNetworks will be present in the
Pod.PodStatus.PodIPs list. To properly identify which IP belongs to what
PodNetwork, we will expand the PodIP struct to include the name of PodNetwork
the specific IP belongs to. This list will allow only 1 IP address per family
(v4, v6) per PodNetwork. The Pod.PodStatus.PodIP behavior will not change.
Proposed changes below:
```go
// IP address information for entries in the (plural) PodIPs field.
// Each entry includes:
//
//      IP: An IP address allocated to the pod.
//      PodNetworkName: Name of the PodNetwork the IP belongs to.
//      InterfaceName: Name of the network interface inside the Pod.
type PodIP struct {
        // ip is an IP address (IPv4 or IPv6) assigned to the pod
        IP string `json:"ip,omitempty" protobuf:"bytes,1,opt,name=ip"`

        // PodNetworkName is name of the PodNetwork the IP belongs to
        //
        // +optional
        PodNetworkName string `json:"podNetwork"`

        // InterfaceName is name of the network interface used for this attachment
        //
        // +optional
        InterfaceName string `json:"interfaceName",omitempty`
}
```
Example:
```yaml
kind: Pod
metadata:
  name: pod1
  namespace: default
spec:
[...]
  networks:
  - podNetwork: default
    interfaceName: eth0
    isDefaultGW4: true
  - podNetwork: dataplane1
    interfaceName: net1
  - podNetworkAttachmentName: my-interface-fast
    interfaceName: net2
status:
[...]
  podIP: 192.168.5.54
  podIPs:
  - ip: 192.168.5.54
    podNetwork: default
    interfaceName: eth0
  - ip: 10.0.0.20
    podNetwork: dataplane1
    interfaceName: net1
  - ip: 2011::233
    podNetwork: my-interface-fast
    interfaceName: net2
```
The above status is expected to be populated by kubelet, but this can only happen
after CRI provides support for the new Pod API. Because of that, initially
kubelet will behave as it does today, without updating the additional fields.
Until CRI catches up, the PodNetwork providers will be able to update that field
on their own.

#### DRA integration (alternative)
We have discussed, as an alternative model, usage of the DRA API, and concluded
that using it will be less clear for the user, compared to the explicit
PodNetwork model, proposed above.

### API server changes
These are the changes for the API server covered by this design:
* Provide validation webhook for PodNetwork or PodNetworkAttachment objects
* Handle “default” PodNetwork deletion
* Extend static Pod spec validation

### Scheduler changes
These are the changes we will do in Pod scheduler:
* Provide active Pod spec validation

When one of the multi-network validation fails, scheduler will follow the current
“failure” path for Pod:
* set PodScheduled condition (of the Pod) to False with appropriate error message
* send Pod Event with same error message

### Endpointslice controller changes
Considering changes to Pod.PodStatus.PodIPs list, we must ensure that the
controller is using the correct IPs when creating the endpoints. We will ensure
that only IPs for "defautl" PodNetwork will be used.

### Kubelet changes
We will introduce an additional check for kubelet readiness for networking.
Today kubelet does this verification via CRI that checks CNI config presence.
We will add a “default” PodNetwork presence check to this flow. Until such a
PodNetwork is not created, kubelet will keep the Ready condition in “False”
with “default PodNetwork not found in API” error message.

### CRI changes
Considering the main input argument for kubelet when it interacts with CRI is
the v1.Pod object, the above changes cover the kubelet-side part of providing
the required data for multi-network. Next what is required are the changes to
CRI API and CNI API which include
* Pod creation flow update in RunPodSandbox
* Pod status flow update in PodSandboxStatus
* CNI input and output values

Considering all above changes are in the direct scope of CRI, this KEP will not
propose complete changes for them, and a separate KEP will be created to cover it.
Below are some suggestions on what the changes could look like.

#### Pod Creation
The Pod creation is handled by the SyncPod function, which calls RunPodSandbox
([code](https://github.com/kubernetes/cri-api/blob/release-1.28/pkg/apis/runtime/v1/api.proto#L40))
CRI API. The parameters for that function are defined by PodSandboxConfig ([code](https://github.com/kubernetes/cri-api/blob/release-1.28/pkg/apis/runtime/v1/api.pb.go#L1343)).
We propose change that API message in following way:
```go
type PodSandboxConfig struct {
[...]
        // Optional configuration for PodNetworks.
        PodNetworks []*PodNetworkConfig `protobuf:"bytes,10,opt,name=podNetworks,proto3" json:"podNetworks,omitempty"`
}

// PodNetworkConfig specifies the PodNetwork configuration.
type PodNetworkConfig struct {
        // Name of the podNetwork.
        Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`

        // Provider is the name of the implementer.
        Provider string `protobuf:"bytes,2,opt,name=provider,proto3" json:"provider,omitempty"`

        // InterfaceName name of the network interface inside the Pod namespace. Default: 0 (not specified).
        InterfaceName string `protobuf:"bytes,3,opt,name=interfaceName,proto3" json:"interfaceName,omitempty"`

        XXX_NoUnkeyedLiteral struct{} json:"-"
        XXX_sizecache        int32    json:"-"
}
```

#### Pod Status
This part is as well using the same SyncPod function, that gets all data from
the PodSandboxStatus ([code](https://github.com/kubernetes/cri-api/blob/release-1.28/pkg/apis/runtime/v1/api.proto#L58))
CRI API. Internally there is PodIP structure to which we would like to add new
fields PodNetworkName new InterfaceName:
```go
// PodIP represents an ip of a Pod
type PodIP struct {
[...]
        // PodNetworkName name of the PodNetwork this IP belongs to
        PodNetworkName string   `protobuf:"bytes,2,opt,name=podNetworkName,proto3" json:"podNetworkName,omitempty"`

        // InterfaceName is name of the network interface inside Pod for this Interface/PodNetwork
        InterfaceName string   `protobuf:"bytes,3,opt,name=interfaceName,proto3" json:"interfaceName,omitempty"`
}
```

#### CNI API
Lastly, we would like for the CNI API to be able to handle Multi-Network for the
agent-based CNI model (see above). It should have the ability to pass the list
of PodNetworks requested by the Pod to the CNI binary, as well as be able to
receive a list of IPs from the CNI with mapped PodNetworks.

### Graduation Criteria

#### Alpha

#### Beta

N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

N/A

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

N/A

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

N/A

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

N/A

###### How can someone using this feature know that it is working for their instance?

N/A

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

## Implementation History

N/A

## Drawbacks

N/A

## Alternatives

We will not define a unified API, and this capability will live on as just an addon
to Kubernetes.
