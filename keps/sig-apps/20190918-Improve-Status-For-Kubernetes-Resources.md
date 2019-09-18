---
title: improve-status-for-k8s-resources  
authors:
  - @barney-s

owning-sig: sig-apps  
participating-sigs:
  - sig-apps
  - sig-api

reviewers:
  - @liggitt
  - @kow3ns
  - @janetkuo
  - @mortent

approvers:
  - @kow3ns
  - @liggitt

editor: @barney-s  
creation-date: 2019-09-16  
last-updated: 2019-04-16  
status: implementable  
see-also:  
replaces:  
superseded-by:  

---

# Improve .status of kubernetes resources

## Table of Contents

- [Summary](#summary)
- [Motivation](#motivation)
- [Goals](#goals)
- [Understanding status](#understanding-status)
- [Proposal for standardizing status](#proposal-for-standardizing-status)
- [Proposal to improve status sources](#proposal-to-improve-status-sources)
- [Proposal for implementation](#proposal-for-implementation)
- [Existing Tooling](#existing-tooling)
- [FAQs](#faqs)
- [Implementation Notes](#implementation-notes)
- [Constraints](#constraints)
- [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)

## Summary

Propose standardizing sections of [`.status`](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md) similar to [`.metadata`](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#metadata) thereby allowing tooling to work with any kubernetes

## Motivation

Dynamic tooling works with all KRM (kubernetes resource model) types sych as core-types and [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/). Mostly the tools operate on the `.spec` and standardized `.metadata` of the resources. Some examples of dynamic tooling include:  
- apiserver - store any KRM types, update their metadata
- RBAC - create roles, permissions to access any KRM type
- kubectl - CRUD, diff any KRM type
- helm - create package that contain any KRM type
- Dynamic client - CRUD any KRM type
- Application CRD - define a grouping of resources of any KRM type.
- UIs -  view, edit any KRM object (YAML/JSON).
- CI/CD/workflow systems - manage any KRM resources  

Meanwhile the `.status` of KRM types are largely inconsistent and with no common interface to deal with them. When inferring `.status`, automation tools resort to per-type logic for well known core-types and CRDs. There exists no conventions, standardized fields in `.status` that dynamic tools can take advantage of.

Some of the use cases which can benefit from standardizing .status are:  
- After using a client to create resources (kubectl create -f), we want to wait until the resources are ready to use.
- After using a client to update a resource (apply), wait until those updates have taken effect.  
- In a UI, or the CLI we would like to show the status of any KRM resource: is it "working as expected".  
- We also want to aggregate the status of multiple resources into a one or two summary metrics.
- CI/CD systems working with KRM resources can use standardized status to determine when it is safe to move to the next step
- A manual operator can answer the question  "can I go home now" in a consistent and with higher confidence after actuating a bunch of changes to their application in a k8s cluster.

## Goals

* Understand the state of things wrt to [`.status`](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md) of Kubernetes objects
* Propose a set of standardized conditions/fields and conventions for `.status`
* Discuss how tooling can make use of them


## Understanding status

`.status` of a KRM types has many kinds of fields, some of which:



*   are Informational (eg: IP lists)
*   reflect the current state (eg: readyreplicas). 
*   reflect conditions (conditions array)

`.status` section of KRM types have very few conventions and no standardized recommendations. In some cases the resources even lack the status section. 


### .status as a Guarantee

It would help to think of some fields of `.status` as describing a form of guarantee. Conversely the guarantee desired determines the semantics of the fields in `.status`. Behavioural guarantees allows us to reason the state of the intent being fulfilled. And wherever stronger or more appropriate guarantees are desired, we could add additional status fields or conditions. 


### Point in time snapshot

Typically the guarantee provided by status is a point in time snapshot and doesn’t promise the same in future nor any indication of past. Capturing status trend across time need to be explicitly computed and added. 


### Depends on the resource

What is guaranteed by the status depends on the resource.  As an example for a Pod, `Ready `condition is a guarantee that the containers are running and kubelet health checks have passed. For a Statefulset, `ReadyReplicas `is a guarantee on how many pods have `Ready `condition set but nothing beyond that. It doesn’t guarantee the version of the Pods during a rolling update. In steady state all Pods are expected to have converged to the CurrentRevision.


### Depends on source of Guarantee

There are different sources of guarantees in a cluster which mutate the status of objects. 


<table>
  <tr>
   <td><strong>Source</strong>
   </td>
   <td><strong>Status fields</strong>
   </td>
   <td><strong>Guarantee</strong>
   </td>
  </tr>
  <tr>
   <td>Kubelet
   </td>
   <td>Pod Conditions
   </td>
   <td>
<ul>

<li>Pod/Container liveliness

<li>NW reachability within the node (using http health checks)

<li>Failure conditions with respect to storage
</li>
</ul>
   </td>
  </tr>
  <tr>
   <td>Core Controllers
   </td>
   <td>Core resource status and conditions
   </td>
   <td>
<ul>

<li>Readiness of workload (aggregate of Pod readiness)

<li>Version of workload (under certain conditions)
Some Core controllers that don’t populate enough status for the object say Service. Some objects use phase string in the status which their controller updates, say PVC. Workload controllers update status by aggregating underlying Pod status. 
</li>
</ul>
   </td>
  </tr>
  <tr>
   <td>Custom controllers
   </td>
   <td>Custom resource status and conditions
   </td>
   <td>Depending on how it is implemented, custom controllers for certain workloads can provide stronger guarantees of the workload readiness than just using core workload controllers. Eg: ElasticSearch cluster status vs statefulset Pod status
   </td>
  </tr>
</table>



### Depends on who is asking


*   Cluster admin may be interested in guarantees that enough resources are created and ready. 
*   A DevOps may be interested in guarantees of application readiness.
*   An end user may be interested in application behaviour and performance.
*   A CxO may be interested in optimal cluster usage guarantees.

Different people are interested in different guarantee attributes.


### Aggregation

Aggregation of resource status is a common pattern to reduce information entropy. 

Characteristics:



*   May lead to loss of information
*   Does not provide tighter guarantees on underlying resources than what the underlying resources provide.
*   Additional guarantees may be added

For example, core workload controllers aggregate Pod status to determine their status fields. But it does not improve on the Pod guarantees. But In addition to this, version guarantees are added.


## Proposal for standardizing status

To simplify the interpretation, aggregation and propagation of `.status` of KRM objects, we need to standardize a few fields and conventions for  `.status`. We can come up with recommendations for fields as well as conditions. 


### Conditions vs Fields

Conditions have been debated widely [here](https://github.com/kubernetes/kubernetes/issues/50798), [here](https://github.com/kubernetes/kubernetes/issues/7856#issuecomment-335687733) and documented [here](https://github.com/kubernetes/community/blob/930ce655/contributors/devel/api-conventions.md#typical-status-properties). For objects that have `.conditions`, It is easy to add new condition types. We recommend using conditions in the all CRDs. The dis-advantage of conditions vs fields is that client tooling needs to handle condition array instead of simple fields. There are no conventions on condition types and how to deprecate them unlike fields which are handled via resource versioning. Fields are simple to implement and use but not flexible when changing across versions. We feel that conditions give a good start and some condition types can be promoted to fields later on.


### Standardizing Conditions

*   Standardize presence of  `.conditions` in all KRM types.
*   Standardize a few conditions types

<table>
  <tr>
   <td>
<strong>Condition</strong>
   </td>
   <td><strong>Description</strong>
   </td>
   <td><strong>Aggregation</strong>
   </td>
  </tr>
  <tr>
   <td>Ready
   </td>
   <td>Qualifies a resource’s readiness as defined by the controller. Please note that this is ready from the control plane perspective. It may not be ready from external users, due to dependant or wrapping services not being ready.
   </td>
   <td>If all children that have Ready true (ANDing), then the set  of children is ready.  If the parent has other internal readiness checks and those are True as well, then the Parent should show Ready.
   </td>
  </tr>
  <tr>
   <td>Settled
   </td>
   <td>Captures a notion of convergence with the intent (.spec). This is useful for automation systems which rely on a guarantee of desired intent being fulfilled.  For resources with a completion semantic, this could be used to indicate completion. This is not a monotonic condition. Meaning it can bounce between True and False in response to changes to the intent. 
   </td>
   <td> If all children that set Settled are Settled, then the set of children is settled.  If the Parent does not plan to create/delete/modify children, and has acted on the observed generation, then the Parent is settled.
   </td>
  </tr>
  <tr>
   <td>Health (optional)
   </td>
   <td>False could imply that the intent is not fulfilled and/or the resources are unavailable or degraded. True implies that the intent is fulfilled and the resources are available completely. This field greatly benefits from custom application controllers that have intimate knowledge of the workload they are managing.
   </td>
   <td>
   </td>
  </tr>
  <tr>
   <td>Progress (optional)
   </td>
   <td>
   </td>
   <td>
   </td>
  </tr>
  <tr>
   <td>...
   </td>
   <td>
   </td>
   <td>
   </td>
  </tr>
</table>



### Standardizing Fields

If promoting conditions to fields we lose certain information such as time, reason, message etc.


<table>
  <tr>
   <td>Field
   </td>
   <td>Description
   </td>
  </tr>
  <tr>
   <td>isReady
   </td>
   <td>Binary field that tracks <code>Ready</code> condition. 
   </td>
  </tr>
  <tr>
   <td>isSettled
   </td>
   <td>Binary field that tracks <code>Settled</code> condition. 
   </td>
  </tr>
  <tr>
   <td>Health (optional)
   </td>
   <td>Trianry RYG field. 
<p>
Health can be used as a single flag that captures the state of the resource. It could derive its state from underlying conditions like <code>Ready, TrafficReady, Converged </code>etc. 
<p>
A value of:
<ul>

<li>Red could imply that the resource may not be functional and/or the resource intent is not fulfilled.

<li>Yellow could imply that the intent is fulfilled, but the resource is available in an acceptable but degraded state.

<li>Green could imply that the intent is fulfilled and the resources are available completely.

<p>
This field greatly benefits from custom application controllers that have intimate knowledge of the workload they are managing. For core controllers, this field only represents the controller view based on intent and current state of things.
</li>
</ul>
   </td>
  </tr>
  <tr>
   <td>...
   </td>
   <td>
   </td>
  </tr>
</table>



### .status .spec Consistency

Intent specified in `.spec` can be changed and is tracked by <code>.metadata.[generation](https://stackoverflow.com/questions/47100389/what-is-the-difference-between-a-resourceversion-and-a-generation)</code>. By convention, <code>.status.observedGeneration</code> is used to indicate that the controller has [observed but not yet completed acting](https://github.com/knative/serving/issues/4149) on a certain version of the intent. This implies that tooling can’t reason which generation of the intent the <code>.status</code> reflects. 


#### .status.reconciledGeneration

We propose adding `.status.reconciledGeneration` to indicate the generation of the intent that the control loop has acted upon. This gives the tooling an ability to reason what generation of the intent is being reflected in the status. Something along the lines of “settled at <generation>”.  Tooling can also implement wait till it observes last applied changes by checking if the reconciledGeneration is greater than or equal to the last applied generation.


#### Atomic flipping of status by API server

Another proposal is to flip the standardized conditions/fields whenever the spec changes. This needs to be done in the API server. This simplifies tooling and controller implementations since we don't need to handle `reconciledGeneration`. But we lose the ability to reason or wait for last applied changes to take effect. Any subsequent changes would flip the status and the tooling would have to wait till the status finally converges at a later generation.


### Location

Standardized `.status` fields can be located in any of these locations: 
- `.metadata.status`
- `.status.meta`
- `.status` (Preferred)


<table>
  <tr>
   <td><strong>Location</strong>
   </td>
   <td><strong>Pros</strong>
   </td>
   <td><strong>Cons</strong>
   </td>
  </tr>
  <tr>
   <td>.metadata.status 
<p>
<code>metadata:</code>
<p>
<code>  status:</code>
<p>
<code>    reconciledGeneration: </code>
<p>
<code>    conditions:</code>
<p>
<code>      - type: Ready</code>
<p>
<code>        reason: </code>
<p>
<code>        … </code>
<p>
<code>    ready: True</code>
<p>
<code>    … </code>
<p>
<code>spec: …</code>
<p>
<code>status: …</code>
   </td>
   <td>Available for all resources by default.
<p>
Explicit contract between API and tooling.
   </td>
   <td>Extensive changes in API signature, tooling.
<p>
Scope of project is large and affects almost every part of the k8s system.
<p>
Overloads metadata with status which is related to live object state.
   </td>
  </tr>
  <tr>
   <td>.status.meta
<p>
<code>metadata: …</code>
<p>
<code>spec: … </code>
<p>
<code>status:</code>
<p>
<code>  meta:</code>
<p>
<code>    reconciledGeneration: </code>
<p>
<code>    conditions:</code>
<p>
<code>      - type: Ready</code>
<p>
<code>        reason: </code>
<p>
<code>        …</code>
<p>
<code>    ready: True</code>
<p>
<code>    …</code>
   </td>
   <td>Clean namespace within status section.
<p>
Can be standardized by API as explicit contract for tooling.
<p>
Not overloading metadata.
   </td>
   <td>If k8s core and tooling defines this, extensive changes in API signature, tooling.
   </td>
  </tr>
  <tr>
   <td>.status
<p>
<code>metadata: …</code>
<p>
<code>spec: … </code>
<p>
<code>status:</code>
<p>
<code>  ready: True</code>
<p>
<code>  reconciledGeneration: </code>
<p>
<code>  conditions:</code>
<p>
<code>    - type: Ready</code>
<p>
<code>      reason: </code>
<p>
<code>      …</code>
   </td>
   <td>No extensive changes in API signature, tooling.
<p>
Contract between API and  tooling is by convention.
   </td>
   <td>Since it is by convention, we need to rely on community adoption of conventions for CRDs. For core resources we can add the conventions. 
<p>
Tooling needs to handle types that don't conform to conventions.
   </td>
  </tr>
</table>



## Proposal to improve status sources

In addition to core controllers and custom controllers updating `.status`, additional sources could be used to improve status guarantees.


### Augment `.status` using external controllers, probes

There are certain status guarantees that the existing sources cannot surface. For such guarantees we would need external probes or additional custom controllers to augment the resource status. To avoid conflicts different sources work with non-overlapping fields, condition types.

An existing [PodReady ++](https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/0007-pod-ready%2B%2B.md) proposal describes something similar. The subtle difference from that proposal is that the conditions set by external probes are not expected to have a bearing on Ready condition of the resource implicitly. A standardized set of conditions across resources that could be used as signals for status guarantees by the aggregating controllers. 

 

An example:

An external controller that uses network probes to determine if a Pod is serving traffic via a service to set a condition `TrafficReady`. An external probe would test a multi tier application end to end and set a condition `ApplicationQualified` . (_This is different from the podready++ proposal. In podready++ proposal these additional conditions affect Ready condition_)


### Augment status using external dependant resources

Status from external dependant services could also be monitored and fed into the status/condition of the aggregated object (Custom resource) or core resource. This would need custom controllers which operate in decorator mode for the resources. An example is a controller that manages CloudSQL using CRDs.


### Status trend

Stability of status across time and rate of perturbation could itself be a meta-status that could feed into theSLOs. For example a Pod flipping between ready/unready.


## Proposal for implementation


### Core types standardized conditions


*   Add standardized conditions (and possibly fields) for all core objects. This would mean opening a slew of PRs across all core types.
*   Till all the core types have conditions added we need to do the status computation on client side
    *   [Cli-utils](https://github.com/kubernetes-sigs/cli-utils/blob/master/internal/pkg/status/legacy_status.go) has an implementation that is being tested out.
    *   [Application CRD](https://github.com/kubernetes-sigs/application/blob/master/pkg/apis/app/v1beta1/status.go) also does status computation using a controller and summarize it into the Application object.


#### Ready condition

The table below describes when an object of a specific Kind is considered to be "Ready":


<table>
  <tr>
   <td><strong>Kind</strong>
   </td>
   <td><strong>Is Ready Condition(s)</strong>
   </td>
  </tr>
  <tr>
   <td>Deployment
   </td>
   <td>
<ul>

<li>status.observedGeneration == metadata.generation

<li>status.replicas == spec.replicas

<li>status.readyReplicas == spec.replicas

<li>status.availableReplicas == spec.replicas

<li>status.conditions is not empty

<li>All items in status.conditions matches any:

<li>type == "Progressing" AND status == "True" AND reason == "NewReplicaSetAvailable"

<li>type == "Available" AND status == "True"

<li>All items in status.conditions do not match any:

<li>type == "ReplicaFailure" AND status == "True"
</li>
</ul>
   </td>
  </tr>
  <tr>
   <td>StatefulSet
   </td>
   <td>
<ul>

<li>status.observedGeneration == metadata.generation

<li>status.replicas == spec.replicas

<li>status.readyReplicas == spec.replicas

<li>status.currentReplicas == spec.replicas
</li>
</ul>
   </td>
  </tr>
  <tr>
   <td>DaemonSet
   </td>
   <td>
<ul>

<li>status.observedGeneration == metadata.generation

<li>status.desiredNumberScheduled == status.numberAvailable

<li>status.desiredNumberScheduled == status.numberReady
</li>
</ul>
   </td>
  </tr>
  <tr>
   <td>ReplicaSet
   </td>
   <td>
<ul>

<li>status.observedGeneration == metadata.generation

<li>status.replicas == spec.replicas

<li>status.readyReplicas == spec.readyReplicas

<li>status.availableReplicas == spec.availableReplicas

<li>All items in status.conditions do not match any:

<li>type == "ReplicaFailure" AND status == "True"
</li>
</ul>
   </td>
  </tr>
  <tr>
   <td>Pod
   </td>
   <td>
<ul>

<li>status.conditions contains at least one item that matches any: 
<ul>
 
<li>type == "Ready" AND status == "True"
 
<li>type == "Ready" AND reason == "PodCompleted"
</li> 
</ul>
</li> 
</ul>
   </td>
  </tr>
  <tr>
   <td>Service
   </td>
   <td>
<ul>

<li>Any of the following are true: 
<ul>
 
<li>type == "ClusterIP" (default)
 
<li>type == "NodePort"
 
<li>type == "ExternalName"
 
<li>type == "LoadBalancer" AND "spec.clusterIP" is not empty AND "status.loadBalancer.ingress" is not empty AND all objects in "status.loadBalancer.ingress" has an "ip" that is not empty
</li> 
</ul>
</li> 
</ul>
   </td>
  </tr>
  <tr>
   <td>PersistentVolumeClaim
   </td>
   <td>
<ul>

<li>status.phase == "Bound"
</li>
</ul>
   </td>
  </tr>
  <tr>
   <td>PodDisruptionBudget
   </td>
   <td>
<ul>

<li>status.observedGeneration == metadata.generation

<li>status.desiredHealthy == spec.minAvailable

<li>status.currentHealthy >= "status.desiredHealthy"
</li>
</ul>
   </td>
  </tr>
  <tr>
   <td>ReplicationController
   </td>
   <td>
<ul>

<li>status.observedGeneration == metadata.generation

<li>status.replicas == spec.replicas

<li>status.readyReplicas == spec.readyReplicas

<li>status.availableReplicas == spec.availableReplicas
</li>
</ul>
   </td>
  </tr>
</table>



### CRDs Standardized conditions

Once we agree on a standardized set of fields and conditions (as listed above), we can enhance [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) to make it easy to inject such fields in the Custom Resources.


## Existing Tooling



*   [ArgoCD](https://argoproj.github.io/argo-cd/)
    *   Implements per-type logic for [core types](https://argoproj.github.io/argo-cd/operator-manual/health/)
    *   Supports extending argo-cd to implement per-crd type health checks using lua scripting. This change can be bundled into argo-cd or specified via a config map.
*   [Kapp](https://github.com/k14s/kapp/blob/master/docs/apply-waiting.md)
    *   [Per type logic](https://github.com/k14s/kapp/blob/master/docs/apply-waiting.md)
    *   Supports custom type waiting using annotations. Controllers need to update annotations in addition to status to signal kapp. 
*   [Keptn](https://keptn.sh/)
    *   A CD tool from dynatrace
    *   Uses [pitometer](https://www.dynatrace.com/news/blog/automated-deployment-and-architectural-validation-with-pitometer-and-keptn/) to [score](https://www.slideshare.net/grabnerandi/shipping-code-like-a-keptn-continuous-delivery-automated-operations-on-k8s) deployments and drive decisions
    *   Pitometer could be used as a source to augment k8s object status 
*   Istio
    *   Request for [config status](https://github.com/istio/istio/issues/882)
    *   Request for [Exposing status](https://github.com/istio/istio/issues/6082)
*   Programming Status waiting
    *   [GCP deployment manager](https://github.com/GoogleCloudPlatform/deploymentmanager-samples/blob/master/examples/v2/waiter/waiter.jinja)
    *   [AWS CFN wait](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waitcondition.html)
*   Spinnaker
*   [Knative](https://github.com/knative/docs/blob/master/docs/serving/spec/knative-api-specification-1.0.md)
    *   Ready condition for indicating operational correctness
    *   Non-ready conditions feed into Ready condition


## FAQs

Q: How do clients wait for stability after a Change?   
A: Wait for Ready condition, May be Health as well.

Q: How to filter out other editors (HPA)?  
A: Wait for Ready and check if reconciledGeneration >= applied generation

Q: Does generation change on when annotations or labels are changed ?  
A: No

Q: How would this work safely with Apply v2 ?  
A: ToDo Explore


## Implementation Notes

- Open status change PRs for workload core types and track them to closure.
- Decide the list of other core types for which status needs to be standardized.
- Open PRs for any non-workload core types and track themn to closure.


## Constraints
- The success of the standardization effort rests on community adoption.

## Test Plan
- Each of the core-types would be tested as and when they are fixed. 

## Graduation Criteria

- [ ] Implement for core types (workloads)
- [ ] Update documents to reflect the changes

## Implementation History

- `.status` has been part of most core-types since begining
