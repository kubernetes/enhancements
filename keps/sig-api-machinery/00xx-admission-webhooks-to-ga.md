---
title: Graduate Admission Webhooks to GA
authors:
  - "@mbohlool"
owning-sig: sig-api-machinery
reviewers:
  - "@liggitt"
  - "@deads2k"
approvers:
  - "@liggitt"
  - "@deads2k"
editor: TBD
creation-date: 2019-01-27
last-updated: 2019-02-04
status: implemented
see-also:
  - "[Admission Control Webhook Beta Design Doc](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/admission-control-webhooks.md)"
---

# Admission Webhooks to GA

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Object selector](#object-selector)
  - [Scope](#scope)
  - [timeout configuration](#timeout-configuration)
  - [Port configuration](#port-configuration)
  - [Plumbing existing objects to delete admission](#plumbing-existing-objects-to-delete-admission)
  - [Mutating Plugin ordering](#mutating-plugin-ordering)
  - [Passing {Operation}Option to Webhook](#passing-operationoption-to-webhook)
    - [Connection Options](#connection-options)
  - [AdmissionReview v1](#admissionreview-v1)
  - [Convert to webhook-requested version](#convert-to-webhook-requested-version)
- [V1 API](#v1-api)
- [V1beta1 changes](#v1beta1-changes)
- [Validations](#validations)
- [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Post-GA tasks](#post-ga-tasks)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Admission webhook is a way to extend kubernetes by putting hook on object creation/modification/deletion.
Admission webhooks can mutate or validate the object. This feature has been Beta since Kubernetes 1.9.
This document outline required steps to graduate it to GA.

## Motivation

Admission webhooks are currently widely used for extending kubernetes but has been in beta for 3 releases.
Current set of feature requests and bug reports on the feature shows it is close to be stable and motivation
of this KEP is to address those feature requests and bug reports to move this feature toward general availability (GA).

### Goals

Based on the user feedback, These are the planned changes to current feature to graduate it to GA:

* Object Selector: Add a general selector for objects that a webhook can target.
* Scope: Define the scope of the targeted objects, e.g. cluster vs. namespaced.
* timeout configuration: Add a configuration field to shorten the timeout of the webhook call.
* port configuration: Add a configuration field to change the service port from the default 443.
* AdmissionReview version selection: Add a API field to set the AdmissionReview versions a webhook can understand.
* Pass existing objects to delete admission webhooks
* re-run mutating plugins if any webhook changed object to fix the plugin ordering problem
* pass OperationOption (such as CreateOption/DeleteOption) to the webhook
* make `Webhook.SideEffects` field required in `v1` API (look at [dryRun KEP(https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/0015-dry-run.md#admission-controllers)] for more information on this item)
* convert incoming objects to the webhook-requested group/version

### Non-Goals

These are the issues considered and rejected for GA:

* Can CA trust bundle being referenced via secret/configmap instead of inline? ([discussion issue](https://github.com/kubernetes/kubernetes/issues/72944))
* Should API servers have a dynamic way to authenticate to admission webhooks? ([Discussion issue](https://github.com/kubernetes/enhancements/pull/658))

## Proposal

This section describes each of the goals in the [Goals](#goals) section in more detail.

### Object selector

Currently Admission Webhook supports namespace selectors, but that may not be enough 
for some cases that admission webhook need to be skipped on some objects. For example 
if the Admission Webhook is running inside cluster and its rules includes wildcards 
which match required API objects for its own execution, it won't work. An object selector 
would be useful exclude those objects. 

Also in case of an optional webhooks, an object selector gives the end user ability to 
include or exclude an object without having access to admission webhook configuration 
which is normally restricted to cluster admins.

Note that namespaced objects must match both the object selector (if specified) and namespace selector to be sent to the Webhook.

The object selector applies to the labels of the Object and OldObject sent to the webhook.

An empty object selector matches all requests and objects.

A non-empty object selector is evaluated against the labels of both the old and new
objects that would be sent to the webhook. If the selector matches against either
set of labels, the selector is considered to match the request.

If one of the objects is nil (like `OldObject` is for create requests, or `Object` is for delete requests),
or cannot have labels (like a `DeploymentRollback` or `NodeProxyOptions` object), that object is not considered to match.

The proposed change is to add an ObjectSelector to the webhook API both in v1 and v1beta1:

```golang
type {Validating,Mutating}Webhook struct {
    ...
    // ObjectSelector decides whether to run the webhook based on if the
    // object has matching labels. objectSelector is evaluated against both
    // the oldObject and newObject that would be sent to the webhook, and
    // is considered to match if either object matches the selector. A null
    // object (oldObject in the case of create, or newObject in the case of
    // delete) or an object that cannot have labels (like a
    // DeploymentRollback or a PodProxyOptions object) is not considered to
    // match.
    // Use the object selector only if the webhook is opt-in, because end
    // users may skip the admission webhook by setting the labels.
    // Default to the empty LabelSelector, which matches everything.
    // +optional
    ObjectSelector *metav1.LabelSelector `json:"objectSelector,omitempty"
}
```

### Scope

Current webhook Rules applies to objects of all scopes. That means a Rule can use wildcards 
to target both namespaced and cluster-scoped objects. 

An evaluation of the targeting capabilities required by in-tree admission plugins showed that
some plugins (like NamespaceLifecycle and ResourceQuota) require the ability to intercept
all namespaced resources. This selection is currently inexpressible for webhook admission.

The proposal is to add a scope field to Admission Webhook configuration to limit webhook 
targeting to namespaced or cluster-scoped objects. This enables webhook developers to 
target only namespaced objects or cluster-scoped objects, just like in-tree admission plugins can.

The field will be added to both v1 and v1beta1.

The field is optional and defaults to "*", meaning no scope restriction.

```golang
type ScopeType string

const (
     // ClusterScope means that scope is limited to cluster-scoped objects.
     // Namespace objects are cluster-scoped.
     ClusterScope ScopeType = "Cluster"
     // NamespacedScope means that scope is limited to namespaced objects.
     NamespacedScope ScopeType = "Namespaced"
     // AllScopes means that all scopes are included.
     AllScopes ScopeType = "*"
)

type Rule struct {
    ...

     // Scope specifies the scope of this rule.
     // Valid values are "Cluster", "Namespaced", and "*"
     // "Cluster" means that only cluster-scoped resources will match this rule.
     // Namespace API objects are cluster-scoped.
     // "Namespaced" means that only namespaced resources will match this rule.
     // "*" means that there are no scope restrictions.
     // Default is "*".
     //
     // +optional
     Scope ScopeType `json:"scope,omitempty" protobuf:"bytes,3,opt,name=scope"`
}
```

### timeout configuration

Admission Webhook has a default timeout of 30 seconds today, but long running webhooks 
would result in a poor performance. By adding a timeout field to the configuration, 
the webhook author can limit the running time of the webhook that either result in 
failing the API call earlier or ignore the webhook call based on the failure policy. 
This feature, however, would not let the timeout to be extended more than 30 seconds 
and the timeout would be defaulted to a shorter value (e.g. 10 seconds) for v1 API while 
stays 30 second for v1beta API to keep backward compatibility.

```golang
type {Validating,Mutating}Webhook struct {
    ...
     // TimeoutSeconds specifies the timeout for this webhook. After the timeout passes,
     // the webhook call will be ignored or the API call will fail based on the
     // failure policy.
     // The timeout value must be between 1 and 30 seconds.
     // Default to 10 seconds for v1 API and 30 seconds for v1beta1 API.
     // +optional
     TimeoutSeconds *int32 `json:"timeoutSeconds,omitempty" protobuf:"varint,8,opt,name=timeoutSeconds"`
}
```

### Port configuration

Today Admission Webhook port is always expected to be 443 on service reference. But this 
limitation was arbitrary and there are cases that Admission Webhook cannot use this port. 
This feature will add a port field to service based webhooks and allows specifying a port 
other than 443 for service based webhooks. Specifying port should already be available for 
URL based webhooks.

The following field will be added to the `ServiceReference` types used by admission webhooks, `APIService`, and `AuditSink` configurations:

```golang
type ServiceReference struct {
    ...

     // If specified, the port on the service that hosting webhook.
     // Default to 443 for backward compatibility.
     // `Port` should be a valid port number (1-65535, inclusive).
     // +optional
     Port *int32 `json:"port,omitempty" protobuf:"varint,4,opt,name=port"`
}
```

### Plumbing existing objects to delete admission

Current admission webhooks can hook on `DELETE` events but they won't get any object in 
`oldObject` or `Object` field of `AdmissionRequest`. The proposal is to send them existing 
object(s) as the `oldObject`. This is also helpful for applying `ObjectSelector` to these 
webhooks. When deleting a collection, each matching object is sent as the `oldObject` to
to webhooks in individual request.

There is no API change for this feature, only documentation:

```golang
type AdmissionRequest struct {
  ...
  // OldObject is the existing object. Only populated for UPDATE and DELETE requests.
  // +optional
  OldObject runtime.RawExtension `json:"oldObject,omitempty" protobuf:"bytes,10,opt,name=oldObject"`
  ..
}
```

### Mutating Plugin ordering

A single ordering of mutating admissions plugins (including webhooks) does not work for all cases
(see https://issue.k8s.io/64333 as an example). A mutating webhook can add a new sub-structure 
to the object (like adding a `container` to a pod), and other mutating plugins which have already 
run may have opinions on those new structures (like setting an `imagePullPolicy` on all containers). 

To allow mutating admission plugins to observe changes made by other plugins, regardless of order,
admission will add the ability to re-run mutating plugins (including webhooks) a single time if 
there is any change made by a mutating webhook on the first pass.

1. Initial mutating admission pass
2. If any webhook plugins returned patches modifying the object, do a second mutating admission pass
   1. Run all in-tree mutating admission plugins
   2. Run all applicable webhook mutating admission plugins that indicate they want to be reinvoked

Mutating plugins that are reinvoked must be idempotent, able to successfully process an object they have already 
admitted and potentially modified. This will be clearly documented in admission webhook documentation,
examples, and test guidelines. Mutating admission webhooks *should* already support this (since any 
change they can make in an object could already exist in the user-provided object), and any webhooks
that do not are broken for some set of user input.

Note that idempotence (whether a webhook can successfully operate on its own output) is distinct from
the `sideEffects` indicator, which describes whether a webhook can safely be called in `dryRun` mode.

The reinvocation behavior will be opt-in, so `reinvocationPolicy` defaults to `"Never"`.

```golang
type MutatingWebhook struct {
    ...
    // reinvocationPolicy indicates whether this webhook should be called multiple times as part of a single admission evaluation.
    // Allowed values are "Never" and "IfNeeded".
    //
    // Never: the webhook will not be called more than once in a single admission evaluation.
    //
    // IfNeeded: the webhook will be called at least one additional time as part of the admission evaluation
    // if the object being admitted is modified by other admission plugins after the initial webhook call.
    // Webhooks that specify this option *must* be idempotent, able to process objects they previously admitted.
    // Note:
    // * the number of additional invocations is not guaranteed to be exactly one.
    // * if additional invocations result in further modifications to the object, webhooks are not guaranteed to be invoked again.
    // * webhooks that use this option may be reordered to minimize the number of additional invocations.
    // * to validate an object after all mutations are guaranteed complete, use a validating admission webhook instead (recommended for webhooks with side-effects).
    //
    // Defaults to "Never".
    // +optional
    ReinvocationPolicy *ReinvocationPolicyType `json:"reinvocationPolicy,omitempty"`
}

type ReinvocationPolicyType string

var (
  NeverReinvocationPolicy ReinvocationPolicyType = "Never"
  IfNeededReinvocationPolicy ReinvocationPolicyType = "IfNeeded"
)
```

Although this problem can go deeper than two levels (for example, if the second mutating webhook added
a new structure based on the new `container`), a single re-run is sufficient for the current set of 
in-tree plugins and webhook use-cases. If future use cases require additional or more targeted reinvocations,
the `AdmissionReview` response can be enhanced to provide information to target those reinvocations.

Test scenarios:

1. No reinvocation
   * in-tree -> mutation
   * webhook -> no mutation
   * complete (no reinvocation needed)

2. In-tree reinvocation only
   * in-tree -> mutation
   * webhook -> mutation
   * reinvoke in-tree -> no mutation
   * complete (in-tree didn't change anything, so no webhook reinvocation required)

3. Full reinvocation
   * in-tree -> mutation
   * webhook -> mutation
   * reinvoke in-tree -> mutation
   * reinvoke webhook -> mutation
   * complete (both reinvoked once, no additional loops)

4. Multiple webhooks, partial reinvocation
   * in-tree -> mutation
   * webhook A -> mutation
   * webhook B -> mutation
   * reinvoke in-tree -> no mutation
   * reinvoke webhook A -> no mutation
   * complete (no mutations occurred after webhook B was called, no reinvocation required)

5. Multiple webhooks, full reinvocation
   * in-tree -> mutation
   * webhook A -> mutation
   * webhook B -> mutation
   * reinvoke in-tree -> no mutation
   * reinvoke webhook A -> mutation
   * reinvoke webhook B -> mutation (webhook A mutated after webhook B was called, one reinvocation required)
   * complete (both reinvoked once, no additional loops)

### Passing {Operation}Option to Webhook

Each of the operations webhook can have an `Option` structure (e.g. `DeleteOption` or `CreateOption`) 
passed by the user. It is useful for some webhooks to know what are those options user requested to 
better modify or validate object. The proposal is to add those Options as an `Options` field to the 
`AdmissionRequest` API object.

```golang
type AdmissionRequest struct {
  ...
  // Options is the operation option structure user passed to the API call. e.g. `DeleteOption` or `CreateOption`.
  // +optional
  Options runtime.RawExtension `json:"options,omitempty" protobuf:"bytes,12,opt,name=options"`
  ..
}
```

#### Connection Options

Each connect operation declares its own options type, which is sent as the
`AdmissionRequest.Object` to admission webhooks. For example, the published
OpenAPI for node proxy declares its connect operation as:

```json
"/api/v1/nodes/{name}/proxy": {
  ...
  "x-kubernetes-action": "connect",
  "x-kubernetes-group-version-kind": {
    "group": "",
    "kind": "NodeProxyOptions",
    "version": "v1"
  }
}
```

Here, an admission webhook request for a node proxy connect operation will send
`NodeProxyOptions` as the `AdmissionRequest.Object` and since connect operations
have no common options, `AdmissionRequest.Options` will be absent.

This is consistent with how kubernetes operations send the type specified by the
OpenAPI `x-kubernetes-group-version-kind` property as the
`AdmissionRequest.Object` to admission webhooks.

### AdmissionReview v1

The payload API server sends to Admission webhooks is called AdmissionReview which is `v1beta1` today.
The proposal is to promote the API to v1 with no change to the API object definition. Because of different 
versions of Admission Webhooks, there must be a way for the webhook developer to specify which apiVersion 
of `AdmissionReview` they support. The field must be an ordered list which reflects the webhooks preference 
of `AdmissionReview` apiVersions.

A version list will be added to webhook configuration that would be defaulted to `['v1beta1']` in v1beta1 API 
and will be a required field in `v1`.

A webhook must respond with the same apiVersion of the AdmissionReview it received. 
For example, a webhook that registers admissionReviewVersions:["v1","v1beta1"] must be prepared to receive and respond with those versions. 

V1 API looks like this:

```golang
type {Validating,Mutating}Webhook struct {
    ...

     // AdmissionReviewVersions is an ordered list of preferred `AdmissionReview`
     // versions the Webhook expects. API server will try to use first version in
     // the list which it supports. If none of the versions specified in this list
     // supported by API server, validation will fail for this object.
     // If the webhook configuration has already been persisted, calls to the 
     // webhook will fail and be subject to the failure policy.
     // This field is required and cannot be empty.
     AdmissionReviewVersions []string `json:"admissionReviewVersions" protobuf:"bytes,9,rep,name=admissionReviewVersions"`
}
```

V1beta1 API looks like this:

```golang
type {Validating,Mutating}Webhook struct {
    ...

     // AdmissionReviewVersions is an ordered list of preferred `AdmissionReview`
     // versions the Webhook expects. API server will try to use first version in
     // the list which it supports. If none of the versions specified in this list
     // supported by API server, validation will fail for this object.
     // If the webhook configuration has already been persisted, calls to the 
     // webhook will fail and be subject to the failure policy.
     // Default to `['v1beta1']`.
     // +optional
     AdmissionReviewVersions []string `json:"admissionReviewVersions,omitempty" protobuf:"bytes,9,rep,name=admissionReviewVersions"`
}
```

### Convert to webhook-requested version

Webhooks currently register to intercept particular API group/version/resource combinations.

Some resources can be accessed via different versions, or even different API 
groups (for example, `apps/v1` and `extensions/v1beta1` Deployments). To 
intercept a resource effectively, all accessible groups/versions/resources
must be registered for and understood by the webhook.

When upgrading to a new version of the apiserver, existing resources can be 
made available via new versions (or even new groups). Ensuring all webhooks
(and registered webhook configurations) have been updated to be able to 
handle the new versions/groups in every upgrade is possible, but easy to 
forget to do, or to do incorrectly. In the case of webhooks not authored 
by the cluster-administrator, obtaining updated admission plugins that 
understand the new versions could require significant effort and time.

Since the apiserver can convert between all of the versions by which a resource 
is made available, this situation can be improved by having the apiserver 
convert resources to the group/versions a webhook registered for.

Because admission can be used for out-of-tree defaulting and field enforcement,
admission plugins may intentionally target specific versions of resources.
A `matchPolicy` field will be added to the webhook configuration object,
allowing a configuration to specify whether the apiserver should only route requests
which exactly match the specified rules to the webhook, or whether it should route
requests for equivalent resources via different API groups or versions as well.
For safety, this field defaults to `Exact` in `v1beta1`. In `v1`, we can default it to `Equivalent`.

```golang
// Webhook describes an admission webhook and the resources and operations it applies to.
type {Validating,Mutating}Webhook struct {
     ...
     // matchPolicy defines how the "rules" field is applied when a request is made 
     // to a different API group or version of a resource listed in "rules".
     // Allowed values are "Exact" or "Equivalent".
     // - Exact: match requests only if they exactly match a given rule. For example, if an object can be modified
     // via API versions v1 and v2, and "rules" only includes "v1", do not send a request to "v2" to the webhook.
     // - Equivalent: match requests if they modify a resource listed in rules via another API group or version.
     // For example, if an object can be modified via API versions v1 and v2, and "rules" only includes "v1",
     // a request to "v2" should be converted to "v1" and sent to the webhook.
     // Defaults to "Exact"
     // +optional
     MatchPolicy *MatchPolicyType `json:"matchPolicy,omitempty"`
```

The apiserver will do the following:

1. For each resource, compute the set of other resources that access or affect the same data, and the kind of the expected object. For example:
  * `apps,v1,deployments` (`apiVersion: apps/v1, kind: Deployment`) is also available via:
    * `apps,v1beta2,deployments` (`apiVersion: apps/v1beta2, kind: Deployment`)
    * `apps,v1beta1,deployments` (`apiVersion: apps/v1beta1, kind: Deployment`)
    * `extensions,v1beta1,deployments` (`apiVersion: extensions/v1beta1, kind: Deployment`)
  * `apps,v1,deployments/scale` (`apiVersion: autoscaling/v1, kind: Scale`) is also available via:
    * `apps,v1beta2,deployments/scale` (`apiVersion: apps/v1beta2, kind: Scale`)
    * `apps,v1beta1,deployments/scale` (`apiVersion: apps/v1beta1, kind: Scale`)
    * `extensions,v1beta1,deployments/scale` (`apiVersion: extensions/v1beta1, kind: Scale`)
2. When evaluating whether to dispatch an incoming request to a webhook with 
`matchPolicy: Equivalent`, check the request's resource *and* all equivalent 
resources against the ones the webhook had registered for. If needed, convert 
the incoming object to one the webhook indicated it understood.

The `AdmissionRequest` sent to a webhook includes the fully-qualified
kind (group/version/kind) and resource (group/version/resource):

```golang
type AdmissionRequest struct {
     ...
     // Kind is the type of object being manipulated.  For example: Pod
     Kind metav1.GroupVersionKind `json:"kind" protobuf:"bytes,2,opt,name=kind"`
     // Resource is the name of the resource being requested.  This is not the kind.  For example: pods
     Resource metav1.GroupVersionResource `json:"resource" protobuf:"bytes,3,opt,name=resource"`
     // SubResource is the name of the subresource being requested.  This is a different resource, scoped to the parent
     // resource, but it may have a different kind. For instance, /pods has the resource "pods" and the kind "Pod", while
     // /pods/foo/status has the resource "pods", the sub resource "status", and the kind "Pod" (because status operates on
     // pods). The binding resource for a pod though may be /pods/foo/binding, which has resource "pods", subresource
     // "binding", and kind "Binding".
     // +optional
     SubResource string `json:"subResource,omitempty" protobuf:"bytes,4,opt,name=subResource"`
```

Prior to this conversion feature, the resource and kind of the request made to the 
API server, and the resource and kind sent in the AdmissionRequest were identical.

When a conversion occurs and the object we send to the webhook is a different kind
than was sent to the API server, or the resource the webhook registered for is different
than the request made to the API server, we have three options for communicating that to the webhook:
1. Do not expose that fact to the webhook:
  * Set AdmissionRequest `kind` to the converted kind
  * Set AdmissionRequest `resource` to the registered-for resource
2. Expose that fact to the webhook using the existing fields:
  * Set AdmissionRequest `kind` to the API request's kind (not matching the object in the AdmissionRequest)
  * Set AdmissionRequest `resource` to the API request's resource (not matching the registered-for resource)
3. Expose that fact to the webhook using new AdmissionRequest fields:
  * Set AdmissionRequest `kind` to the converted kind
  * Set AdmissionRequest `requestKind` to the API request's kind
  * Set AdmissionRequest `resource` to the registered-for resource
  * Set AdmissionRequest `requestResource` to the API request's resource

Option 1 loses information the webhook could use (for example, to enforce different validation or defaulting rules for newer APIs).

Option 2 risks breaking webhook logic by sending it resources it did not register for, and kinds it did not expect.

Option 3 is preferred, and is the safest option that preserves information for use by the webhook.

To support this, three fields will be added to AdmissionRequest, and populated with the original request's kind, resource, and subResource:

```golang
type AdmissionRequest struct {
     ...
     // RequestKind is the type of object being manipulated by the the original API request.  For example: Pod
     // If this differs from the value in "kind", an equivalent match and conversion was performed.
     // See documentation for the "matchPolicy" field in the webhook configuration type.
     // +optional
     RequestKind *metav1.GroupVersionKind `json:"requestKind,omitempty"`
     // RequestResource is the name of the resource being requested by the the original API request.  This is not the kind.  For example: ""/v1/pods
     // If this differs from the value in "resource", an equivalent match and conversion was performed.
     // See documentation for the "matchPolicy" field in the webhook configuration type.
     // +optional
     RequestResource *metav1.GroupVersionResource `json:"requestResource,omitempty"`
     // RequestSubResource is the name of the subresource being requested by the the original API request.  This is a different resource, scoped to the parent
     // resource, but it may have a different kind. For instance, /pods has the resource "pods" and the kind "Pod", while
     // /pods/foo/status has the resource "pods", the sub resource "status", and the kind "Pod" (because status operates on
     // pods). The binding resource for a pod though may be /pods/foo/binding, which has resource "pods", subresource
     // "binding", and kind "Binding".
     // If this differs from the value in "subResource", an equivalent match and conversion was performed.
     // See documentation for the "matchPolicy" field in the webhook configuration type.
     // +optional
     RequestSubResource string `json:"requestSubResource,omitempty"`
```

## V1 API

The currently planned `v1` API is described in this section.
Please note that as long as there are open questions in the [Graduation Criteria](#graduation-criteria) section, this is not final.

```golang
package v1

type ScopeType string

const (
     // ClusterScope means that scope is limited to cluster-scoped objects.
     // Namespace API objects are cluster-scoped.
     ClusterScope ScopeType = "Cluster"
     // NamespacedScope means that scope is limited to namespaced objects.
     NamespacedScope ScopeType = "Namespaced"
     // AllScopes means that all scopes are included.
     AllScopes ScopeType = "*"
)

// Rule is a tuple of APIGroups, APIVersion, and Resources.It is recommended
// to make sure that all the tuple expansions are valid.
type Rule struct {
     // APIGroups is the API groups the resources belong to. '*' is all groups.
     // If '*' is present, the length of the slice must be one.
     // Required.
     APIGroups []string `json:"apiGroups,omitempty" protobuf:"bytes,1,rep,name=apiGroups"`

     // APIVersions is the API versions the resources belong to. '*' is all versions.
     // If '*' is present, the length of the slice must be one.
     // Required.
     APIVersions []string `json:"apiVersions,omitempty" protobuf:"bytes,2,rep,name=apiVersions"`

     // Resources is a list of resources this rule applies to.
     //
     // For example:
     // 'pods' means pods.
     // 'pods/log' means the log subresource of pods.
     // '*' means all resources, but not subresources.
     // 'pods/*' means all subresources of pods.
     // '*/scale' means all scale subresources.
     // '*/*' means all resources and their subresources.
     //
     // If wildcard is present, the validation rule will ensure resources do not
     // overlap with each other.
     //
     // Depending on the enclosing object, subresources might not be allowed.
     // Required.
     Resources []string `json:"resources,omitempty" protobuf:"bytes,3,rep,name=resources"`

     // Scope specifies the scope of this rule.
     // Valid values are "Cluster", "Namespaced", and "*"
     // "Cluster" means that only cluster-scoped resources will match this rule.
     // Namespace API objects are cluster-scoped.
     // "Namespaced" means that only namespaced resources will match this rule.
     // "*" means that there are no scope restrictions.
     // Default is "*".
     //
     // +optional
     Scope ScopeType `json:"scope,omitempty" protobuf:"bytes,3,opt,name=scope"`
}

type ConversionPolicyType string

const (
     // ConversionIgnore means that requests that do not match a webhook's rules but could be 
     // converted to a resource the webhook registered for, should be ignored.
     ConversionIgnore ConversionPolicyType = "Ignore"
     // ConversionConvert means that requests that do not match a webhook's rules but could be 
     // converted to a resource the webhook registered for, should be converted and sent to the webhook.
     ConversionConvert ConversionPolicyType = "Convert"
)

type FailurePolicyType string

const (
     // Ignore means that an error calling the webhook is ignored.
     Ignore FailurePolicyType = "Ignore"
     // Fail means that an error calling the webhook causes the admission to fail.
     Fail FailurePolicyType = "Fail"
)

type SideEffectClass string

const (
     // SideEffectClassUnknown means that no information is known about the side effects of calling the webhook.
     // If a request with the dry-run attribute would trigger a call to this webhook, the request will instead fail.
     SideEffectClassUnknown SideEffectClass = "Unknown"
     // SideEffectClassNone means that calling the webhook will have no side effects.
     SideEffectClassNone SideEffectClass = "None"
     // SideEffectClassSome means that calling the webhook will possibly have side effects.
     // If a request with the dry-run attribute would trigger a call to this webhook, the request will instead fail.
     SideEffectClassSome SideEffectClass = "Some"
     // SideEffectClassNoneOnDryRun means that calling the webhook will possibly have side effects, but if the
     // request being reviewed has the dry-run attribute, the side effects will be suppressed.
     SideEffectClassNoneOnDryRun SideEffectClass = "NoneOnDryRun"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ValidatingWebhookConfiguration describes the configuration of and admission webhook that accept or reject and object without changing it.
type ValidatingWebhookConfiguration struct {
     metav1.TypeMeta `json:",inline"`
     // Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata.
     // +optional
     metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
     // Webhooks is a list of webhooks and the affected resources and operations.
     // +optional
     // +patchMergeKey=name
     // +patchStrategy=merge
     Webhooks []ValidatingWebhook `json:"webhooks,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=Webhooks"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ValidatingWebhookConfigurationList is a list of ValidatingWebhookConfiguration.
type ValidatingWebhookConfigurationList struct {
     metav1.TypeMeta `json:",inline"`
     // Standard list metadata.
     // More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
     // +optional
     metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
     // List of ValidatingWebhookConfiguration.
     Items []ValidatingWebhookConfiguration `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// ValidatingWebhook describes an admission ValidatingWebhook and the resources and operations it applies to.
type ValidatingWebhook struct {
     // The name of the admission webhook.
     // Name should be fully qualified, e.g., imagepolicy.kubernetes.io, where
     // "imagepolicy" is the name of the webhook, and kubernetes.io is the name
     // of the organization.
     // Required.
     Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

     // ClientConfig defines how to communicate with the hook.
     // Required
     ClientConfig WebhookClientConfig `json:"clientConfig" protobuf:"bytes,2,opt,name=clientConfig"`

     // Rules describes what operations on what resources/subresources the webhook cares about.
     // The webhook cares about an operation if it matches _any_ Rule.
     // However, in order to prevent ValidatingAdmissionWebhooks and MutatingAdmissionWebhooks
     // from putting the cluster in a state which cannot be recovered from without completely
     // disabling the plugin, ValidatingAdmissionWebhooks and MutatingAdmissionWebhooks are never called
     // on admission requests for ValidatingWebhookConfiguration and MutatingWebhookConfiguration objects.
     Rules []RuleWithOperations `json:"rules,omitempty" protobuf:"bytes,3,rep,name=rules"`

     // matchPolicy defines how the "rules" field is applied when a request is made 
     // to a different API group or version of a resource listed in "rules".
     // Allowed values are "Exact" or "Equivalent".
     // - Exact: match requests only if they exactly match a given rule. For example, if an object can be modified
     // via API version v1 and v2, and "rules" only includes "v1", do not send a request to "v2" to the webhook.
     // - Equivalent: match requests if they modify a resource listed in rules via another API group or version.
     // For example, if an object can be modified via API version v1 and v2, and "rules" only includes "v1",
     // a request to "v2" should be converted to "v1" and sent to the webhook.
     // Defaults to "Equivalent"
     // +optional
     MatchPolicy *MatchPolicyType `json:"matchPolicy,omitempty"`

     // FailurePolicy defines how unrecognized errors from the admission endpoint are handled -
     // allowed values are Ignore or Fail. Defaults to Ignore.
     // +optional
     FailurePolicy *FailurePolicyType `json:"failurePolicy,omitempty" protobuf:"bytes,4,opt,name=failurePolicy,casttype=FailurePolicyType"`
    
     // NamespaceSelector decides whether to run the webhook on an object based
     // on whether the namespace for that object matches the selector. If the
     // object itself is a namespace, the matching is performed on
     // object.metadata.labels. If the object is another cluster scoped resource,
     // it never skips the webhook.
     //
     // For example, to run the webhook on any objects whose namespace is not
     // associated with "runlevel" of "0" or "1";  you will set the selector as
     // follows:
     // "namespaceSelector": {
     //   "matchExpressions": [
     //     {
     //       "key": "runlevel",
     //       "operator": "NotIn",
     //       "values": [
     //      "0",
     //      "1"
     //       ]
     //     }
     //   ]
     // }
     //
     // If instead you want to only run the webhook on any objects whose
     // namespace is associated with the "environment" of "prod" or "staging";
     // you will set the selector as follows:
     // "namespaceSelector": {
     //   "matchExpressions": [
     //     {
     //       "key": "environment",
     //       "operator": "In",
     //       "values": [
     //      "prod",
     //      "staging"
     //       ]
     //     }
     //   ]
     // }
     //
     // See
     // https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
     // for more examples of label selectors.
     //
     // Default to the empty LabelSelector, which matches everything.
     // +optional
     NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty" protobuf:"bytes,5,opt,name=namespaceSelector"`

     // SideEffects states whether this webhookk has side effects.
     // Acceptable values are: Unknown, None, Some, NoneOnDryRun
     // Webhooks with side effects MUST implement a reconciliation system, since a request may be
     // rejected by a future step in the admission change and the side effects therefore need to be undone.
     // Requests with the dryRun attribute will be auto-rejected if they match a webhook with
     // sideEffects == Unknown or Some.
     SideEffects SideEffectClass `json:"sideEffects" protobuf:"bytes,6,opt,name=sideEffects,casttype=SideEffectClass"`

     // ObjectSelector decides whether to run the webhook based on if the
     // object has matching labels. objectSelector is evaluated against both
     // the oldObject and newObject that would be sent to the webhook, and
     // is considered to match if either object matches the selector. A null
     // object (oldObject in the case of create, or newObject in the case of
     // delete) or an object that cannot have labels (like a
     // DeploymentRollback or a PodProxyOptions object) is not considered to
     // match.
     // Use the object selector only if the webhook is opt-in, because end
     // users may skip the admission webhook by setting the labels.
     // Default to the empty LabelSelector, which matches everything.
     // +optional
     ObjectSelector *metav1.LabelSelector `json:"objectSelector,omitempty"`

     // TimeoutSeconds specifies the timeout for this webhook. After the timeout passes,
     // the webhook call will be ignored or the API call will fail based on the
     // failure policy.
     // The timeout value must be between 1 and 30 seconds.
     // Default to 10 seconds.
     // +optional
     TimeoutSeconds *int32 `json:"timeout,omitempty" protobuf:"varint,8,opt,name=timeout"`

     // AdmissionReviewVersions is an ordered list of preferred `AdmissionReview`
     // versions the Webhook expects. API server will try to use first version in
     // the list which it supports. If none of the versions specified in this list
     // supported by API server, validation will fail for this object.
     // If the webhook configuration has already been persisted, calls to the 
     // webhook will fail and be subject to the failure policy.
     // This field is required and cannot be empty.
     AdmissionReviewVersions []string `json:"admissionReviewVersions" protobuf:"bytes,9,rep,name=admissionReviewVersions"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MutatingWebhookConfiguration describes the configuration of and admission webhook that accept or reject and may change the object.
type MutatingWebhookConfiguration struct {
     metav1.TypeMeta `json:",inline"`
     // Standard object metadata; More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata.
     // +optional
     metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
     // Webhooks is a list of webhooks and the affected resources and operations.
     // +optional
     // +patchMergeKey=name
     // +patchStrategy=merge
     Webhooks []MutatingWebhook `json:"webhooks,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=Webhooks"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MutatingWebhookConfigurationList is a list of MutatingWebhookConfiguration.
type MutatingWebhookConfigurationList struct {
     metav1.TypeMeta `json:",inline"`
     // Standard list metadata.
     // More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
     // +optional
     metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
     // List of MutatingWebhookConfiguration.
     Items []MutatingWebhookConfiguration `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// MutatingWebhook describes an admission MutatingWebhook and the resources and operations it applies to.
type MutatingWebhook struct {
     // The name of the admission webhook.
     // Name should be fully qualified, e.g., imagepolicy.kubernetes.io, where
     // "imagepolicy" is the name of the webhook, and kubernetes.io is the name
     // of the organization.
     // Required.
     Name string `json:"name" protobuf:"bytes,1,opt,name=name"`

     // ClientConfig defines how to communicate with the hook.
     // Required
     ClientConfig WebhookClientConfig `json:"clientConfig" protobuf:"bytes,2,opt,name=clientConfig"`

     // Rules describes what operations on what resources/subresources the webhook cares about.
     // The webhook cares about an operation if it matches _any_ Rule.
     // However, in order to prevent ValidatingAdmissionWebhooks and MutatingAdmissionWebhooks
     // from putting the cluster in a state which cannot be recovered from without completely
     // disabling the plugin, ValidatingAdmissionWebhooks and MutatingAdmissionWebhooks are never called
     // on admission requests for ValidatingWebhookConfiguration and MutatingWebhookConfiguration objects.
     Rules []RuleWithOperations `json:"rules,omitempty" protobuf:"bytes,3,rep,name=rules"`

     // matchPolicy defines how the "rules" field is applied when a request is made 
     // to a different API group or version of a resource listed in "rules".
     // Allowed values are "Exact" or "Equivalent".
     // - Exact: match requests only if they exactly match a given rule. For example, if an object can be modified
     // via API version v1 and v2, and "rules" only includes "v1", do not send a request to "v2" to the webhook.
     // - Equivalent: match requests if they modify a resource listed in rules via another API group or version.
     // For example, if an object can be modified via API version v1 and v2, and "rules" only includes "v1",
     // a request to "v2" should be converted to "v1" and sent to the webhook.
     // Defaults to "Equivalent"
     // +optional
     MatchPolicy *MatchPolicyType `json:"matchPolicy,omitempty"`

     // FailurePolicy defines how unrecognized errors from the admission endpoint are handled -
     // allowed values are Ignore or Fail. Defaults to Ignore.
     // +optional
     FailurePolicy *FailurePolicyType `json:"failurePolicy,omitempty" protobuf:"bytes,4,opt,name=failurePolicy,casttype=FailurePolicyType"`

     // reinvocationPolicy indicates whether this webhook should be called multiple times as part of a single admission evaluation.
     // Allowed values are "Never" and "IfNeeded".
     //
     // Never: the webhook will not be called more than once in a single admission evaluation.
     //
     // IfNeeded: the webhook will be called at least one additional time as part of the admission evaluation
     // if the object being admitted is modified by other admission plugins after the initial webhook call.
     // Webhooks that specify this option *must* be idempotent, able to process objects they previously admitted.
     // Note:
     // * the number of additional invocations is not guaranteed to be exactly one.
     // * if additional invocations result in further modifications to the object, webhooks are not guaranteed to be invoked again.
     // * webhooks that use this option may be reordered to minimize the number of additional invocations.
     // * to validate an object after all mutations are guaranteed complete, use a validating admission webhook instead.
     //
     // Defaults to "Never".
     // +optional
     ReinvocationPolicy *ReinvocationPolicyType `json:"reinvocationPolicy,omitempty"`
    
     // NamespaceSelector decides whether to run the webhook on an object based
     // on whether the namespace for that object matches the selector. If the
     // object itself is a namespace, the matching is performed on
     // object.metadata.labels. If the object is another cluster scoped resource,
     // it never skips the webhook.
     //
     // For example, to run the webhook on any objects whose namespace is not
     // associated with "runlevel" of "0" or "1";  you will set the selector as
     // follows:
     // "namespaceSelector": {
     //   "matchExpressions": [
     //     {
     //       "key": "runlevel",
     //       "operator": "NotIn",
     //       "values": [
     //      "0",
     //      "1"
     //       ]
     //     }
     //   ]
     // }
     //
     // If instead you want to only run the webhook on any objects whose
     // namespace is associated with the "environment" of "prod" or "staging";
     // you will set the selector as follows:
     // "namespaceSelector": {
     //   "matchExpressions": [
     //     {
     //       "key": "environment",
     //       "operator": "In",
     //       "values": [
     //      "prod",
     //      "staging"
     //       ]
     //     }
     //   ]
     // }
     //
     // See
     // https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
     // for more examples of label selectors.
     //
     // Default to the empty LabelSelector, which matches everything.
     // +optional
     NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty" protobuf:"bytes,5,opt,name=namespaceSelector"`

     // SideEffects states whether this webhookk has side effects.
     // Acceptable values are: Unknown, None, Some, NoneOnDryRun
     // Webhooks with side effects MUST implement a reconciliation system, since a request may be
     // rejected by a future step in the admission change and the side effects therefore need to be undone.
     // Requests with the dryRun attribute will be auto-rejected if they match a webhook with
     // sideEffects == Unknown or Some.
     SideEffects SideEffectClass `json:"sideEffects" protobuf:"bytes,6,opt,name=sideEffects,casttype=SideEffectClass"`

     // ObjectSelector decides whether to run the webhook based on if the
     // object has matching labels. objectSelector is evaluated against both
     // the oldObject and newObject that would be sent to the webhook, and
     // is considered to match if either object matches the selector. A null
     // object (oldObject in the case of create, or newObject in the case of
     // delete) or an object that cannot have labels (like a
     // DeploymentRollback or a PodProxyOptions object) is not considered to
     // match.
     // Use the object selector only if the webhook is opt-in, because end
     // users may skip the admission webhook by setting the labels.
     // Default to the empty LabelSelector, which matches everything.
     // +optional
     ObjectSelector *metav1.LabelSelector `json:"objectSelector,omitempty"`

     // TimeoutSeconds specifies the timeout for this webhook. After the timeout passes,
     // the webhook call will be ignored or the API call will fail based on the
     // failure policy.
     // The timeout value must be between 1 and 30 seconds.
     // Default to 10 seconds.
     // +optional
     TimeoutSeconds *int32 `json:"timeout,omitempty" protobuf:"varint,8,opt,name=timeout"`

     // AdmissionReviewVersions is an ordered list of preferred `AdmissionReview`
     // versions the Webhook expects. API server will try to use first version in
     // the list which it supports. If none of the versions specified in this list
     // supported by API server, validation will fail for this object.
     // If the webhook configuration has already been persisted, calls to the 
     // webhook will fail and be subject to the failure policy.
     // This field is required and cannot be empty.
     AdmissionReviewVersions []string `json:"admissionReviewVersions" protobuf:"bytes,9,rep,name=admissionReviewVersions"`
}

// RuleWithOperations is a tuple of Operations and Resources. It is recommended to make
// sure that all the tuple expansions are valid.
type RuleWithOperations struct {
     // Operations is the operations the admission hook cares about - CREATE, UPDATE, or *
     // for all operations.
     // If '*' is present, the length of the slice must be one.
     // Required.
     Operations []OperationType `json:"operations,omitempty" protobuf:"bytes,1,rep,name=operations,casttype=OperationType"`
     // Rule is embedded, it describes other criteria of the rule, like
     // APIGroups, APIVersions, Resources, etc.
     Rule `json:",inline" protobuf:"bytes,2,opt,name=rule"`
}

type OperationType string

// The constants should be kept in sync with those defined in k8s.io/kubernetes/pkg/admission/interface.go.
const (
    OperationAll OperationType = "*"
     Create       OperationType = "CREATE"
     Update       OperationType = "UPDATE"
     Delete       OperationType = "DELETE"
     Connect      OperationType = "CONNECT"
)

// WebhookClientConfig contains the information to make a TLS
// connection with the webhook
type WebhookClientConfig struct {
     // `url` gives the location of the webhook, in standard URL form
     // (`scheme://host:port/path`). Exactly one of `url` or `service`
     // must be specified.
     //
     // The `host` should not refer to a service running in the cluster; use
     // the `service` field instead. The host might be resolved via external
     // DNS in some apiservers (e.g., `kube-apiserver` cannot resolve
     // in-cluster DNS as that would be a layering violation). `host` may
     // also be an IP address.
     //
     // Please note that using `localhost` or `127.0.0.1` as a `host` is
     // risky unless you take great care to run this webhook on all hosts
     // which run an apiserver which might need to make calls to this
     // webhook. Such installs are likely to be non-portable, i.e., not easy
     // to turn up in a new cluster.
     //
     // A path is optional, and if present may be any string permissible in
     // a URL. You may use the path to pass an arbitrary string to the
     // webhook, for example, a cluster identifier.
     //
     // Attempting to use a user or basic auth e.g. "user:password@" is not
     // allowed. Fragments ("#...") and query parameters ("?...") are not
     // allowed, either.
     //
     // +optional
     URL *string `json:"url,omitempty" protobuf:"bytes,3,opt,name=url"`

     // `service` is a reference to the service for this webhook. Either
     // `service` or `url` must be specified.
     //
     // If the webhook is running within the cluster, then you should use `service`.
     //
     // +optional
     Service *ServiceReference `json:"service,omitempty" protobuf:"bytes,1,opt,name=service"`

     // `caBundle` is a PEM encoded CA bundle which will be used to validate the webhook's server certificate.
     // If unspecified, system trust roots on the apiserver are used.
     // +optional
     CABundle []byte `json:"caBundle,omitempty" protobuf:"bytes,2,opt,name=caBundle"`
}

// ServiceReference holds a reference to Service.legacy.k8s.io
type ServiceReference struct {
     // `namespace` is the namespace of the service.
     // Required
     Namespace string `json:"namespace" protobuf:"bytes,1,opt,name=namespace"`
     // `name` is the name of the service.
     // Required
     Name string `json:"name" protobuf:"bytes,2,opt,name=name"`

     // `path` is an optional URL path which will be sent in any request to
     // this service.
     // +optional
     Path *string `json:"path,omitempty" protobuf:"bytes,3,opt,name=path"`

     // If specified, the port on the service that hosting webhook.
     // Default to 443 for backward compatibility.
     // `Port` should be a valid port number (1-65535, inclusive).
     // +optional
     Port *int32 `json:"port,omitempty" protobuf:"varint,4,opt,name=port"`
}
```

## V1beta1 changes

All of the proposed changes will be added to `v1beta1` for backward compatibility 
and also to keep roundtrip-ability between `v1` and `v1beta1`. The only difference are:

* Default Value for `timeoutSeconds` field will be 30 seconds for `v1beta1`.
* `AdmissionReviewVersions` list is optional in v1beta1 and defaulted to `['v1beta1']` while required in `v1`.

## Validations

These set of new validation will be applied to both v1 and v1beta1:

* `Scope` field can only have `Cluster`, `Namespaced`, or `*` values (if empty, the field defaults to `*`).
* `Timeout` field must be between 1 and 30 seconds.
* `AdmissionReviewVersions` list must have at least one version supported by the API Server serving it. Note that for downgrade compatibility, Webhook authors should always support as many `AdmissionReview` versions as possible.

## Risks and Mitigations

The features proposed in this KEP are low risk and mostly bug fixes or new features that should have little to no risk on existing features.

## Graduation Criteria

To mark these as complete, all of the above features need to be implemented. 
An [umbrella issue](https://github.com/kubernetes/kubernetes/issues/73185) is tracking all of these changes.
Also there need to be sufficient tests for any of these new features and all existing features and documentation should be completed for all features.

There are still open questions that need to be addressed and updated in this KEP before graduation:

## Post-GA tasks

These are related Post-GA tasks:

* Allow mutating of existing objects in `DELETE` events
  * Should adding/removing existing finalizers in delete admission be allowed?
  * Should GC finalizers based on `DeleteOptions` happen before or after mutating admission?
* If we have enough evidence, consider a better solution for plugin ordering like letting a webhook indicate how many times it should be rerun.

## Implementation History

* First version of the KEP being merged: Jan 29th, 2019
* The set of features for GA approved, initial set of features marked implementable: Feb 4th, 2019
* Implementation start for all approved changes: Feb 4th, 2019
* Added details for auto-conversion: Apr 25th, 2019
* Added details for mutating admission re-running: May 6th, 2019
