---
kep-number: 36
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
status: implementable
see-also:
  - [Admission Control Webhook Beta Design Doc](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/admission-control-webhooks.md)
---

# Admission Webhooks to GA

## Table of Contents

* [Summary](#summary)
* [Motivation](#motivation)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
* [Proposal](#proposal)
  * [Object selector](#object-selector)
  * [Scope](#scope)
  * [timeout configuration](#timeout-configuration)
  * [Port configuration](#port-configuration)
  * [Plumbing existing objects to delete admission](#plumbing-existing-objects-to-delete-admission)
  * [Mutating Plugin ordering](#mutating-plugin-ordering)
  * [Passing {Operation}Option to Webhook](#passing-operationoption-to-webhook)
  * [AdmissionReview v1](#admissionreview-v1)
  * [Convert to webhook-requested version](#convert-to-webhook-requested-version)
* [V1 API](#v1-api)
* [V1beta1 changes](#v1beta1-changes)
* [Validations](#validations)
* [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Post-GA tasks](#post-ga-tasks)
* [Implementation History](#implementation-history)

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

The labels for ObjectSelector will be extracted at the admission time before running 
any plugins and will be used for all Webhooks during the admission process.

The object selector applies to an Object's labels. For create and update, the object 
is the new object. For delete, it is existing object (passed as `oldObject` in the 
`admissionRequest`). For sub-resources, if the subresource accepts the whole object 
and apply the changes to object's labels (e.g. `pods/status`) the labels on the new 
object will be used otherwise (e.g. `pods/proxy`) the labels on the existing parent 
object will be used. If subresource does not have a parent object, the Webhook call 
may fail or skipped based on the failure policy.

When deleting a collection, the webhook will get a filtered list of objects if the 
ObjectSelector is applied. The webhook may be called multiple times with sub-lists.

The proposed change is to add an ObjectSelector to the webhook API both in v1 and v1beta1:

```golang
type Webhook struct {
    ...
     // ObjectSelector decides whether to run the webhook on an object based
     // on whether the object.metadata.labels matches the selector. A namespace object must
     // match both namespaceSelector (if present) and objectSelector (if present).
     // For sub-resources, if the labels on the newObject is effective (e.g. pods/status)
     // they will be used, otherwise the labels on the parent object will be used. If the
     // sub-resource does not have a parent object, the call may fail or skipped based on
     // the failure policy.
     //
     // Default to the empty LabelSelector, which matches everything.
     // +optional
     ObjectSelector *metav1.LabelSelector `json:"objectSelector,omitempty" protobuf:"bytes,7,opt,name=objectSelector"`
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
type Webhook struct {
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
webhooks. Note that `oldObject` when deleting a collection will be a `v1.List` of existing 
objects. API server may call the webhook multiple times with sub-lists. 

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

Current ordering of the plugins (including webhooks) is fixed but may not work for all cases 
(e.g. [this issue](https://github.com/kubernetes/kubernetes/issues/64333)). A mutating webhook 
can add a completely new structure to the object (e.g. a `Container`) and other mutating plugins 
may have opinion on those new structures (e.g. setting the image pull policy on all containers). 
Although this problem can go deeper than two level (if the second mutating webhook added a new 
structure based on the new `Container`), with current set of standard mutating plugins and use-cases, 
it look like a rerun of mutating plugins is a sufficient fix. The proposal is to rerun all mutating 
plugins (including webhooks) if there is any change by mutating webhooks. The assumption for this 
process is all mutating plugins are idempotent and this must be clearly documented to users.

Mutations from in-tree plugins will not trigger this process as they are well-ordered but if 
there is any mutation by webhooks, all of the plugins including in-tree ones will be run again.

This feature would be would be opt in and defaulted to false for `v1beta1`.

The API representation and behavior for this feature is still under design and will be updated/approved here prior to implementation.

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
type Webhook struct {
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
type Webhook struct {
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

A `conversionPolicy` field will be added to the webhook configuration object,
allowing a webhook to choose whether the apiserver should ignore these requests
or convert them to a kind corresponding to a resource the webhook registered for
and send them to the webhook. For safety, this field defaults to Convert.

```golang
// Webhook describes an admission webhook and the resources and operations it applies to.
type Webhook struct {
     ...
     // ConversionPolicy describes whether the webhook wants to be called when a request is made 
     // to a resource that does not match "rules", but which modifies the same data 
     // as a resource that does match "rules".
     // Allowed values are Ignore or Convert. 
     // "Ignore" means the request should be ignored.
     // "Convert" means the request should be converted to a kind corresponding to a resource matched by "rules", and sent to the webhook.
     // Defaults to Convert.
     // +optional
     ConversionPolicy *ConversionPolicyType `json:"conversionPolicy,omitempty"`
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
`conversionPolicy: Convert`, check the request's resource *and* all equivalent 
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
	// RequestKind is the type of object being manipulated in the original API request.  For example: Pod
     // If this differs from the value in "kind", conversion was performed.
     // +optional
	RequestKind *metav1.GroupVersionKind `json:"requestKind,omitempty"`
	// RequestResource is the name of the resource being requested in the original API request.  This is not the kind.  For example: ""/v1/pods
     // If this differs from the value in "resource", conversion was performed.
     // +optional
	RequestResource *metav1.GroupVersionResource `json:"requestResource,omitempty"`
	// RequestSubResource is the name of the subresource being requested in the original API request.  This is a different resource, scoped to the parent
	// resource, but it may have a different kind. For instance, /pods has the resource "pods" and the kind "Pod", while
	// /pods/foo/status has the resource "pods", the sub resource "status", and the kind "Pod" (because status operates on
	// pods). The binding resource for a pod though may be /pods/foo/binding, which has resource "pods", subresource
     // "binding", and kind "Binding".
     // If this differs from the value in "subResource", conversion was performed.
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
     Webhooks []Webhook `json:"webhooks,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=Webhooks"`
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
     Webhooks []Webhook `json:"webhooks,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=Webhooks"`
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

// Webhook describes an admission webhook and the resources and operations it applies to.
type Webhook struct {
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

     // ConversionPolicy describes whether the webhook wants to be called when a request is made 
     // to a resource that does not match "rules", but which modifies the same data 
     // as a resource that does match "rules".
     // Allowed values are Ignore or Convert. 
     // "Ignore" means the request should be ignored.
     // "Convert" means the request should be converted to a kind corresponding to a resource matched by "rules", and sent to the webhook.
     // Defaults to Convert.
     // +optional
     ConversionPolicy *ConversionPolicyType `json:"conversionPolicy,omitempty"`

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

     // ObjectSelector decides whether to run the webhook on an object based
     // on whether the object.metadata.labels matches the selector. A namespace object must
     // match both namespaceSelector (if present) and objectSelector (if present).
     // For sub-resources, if the labels on the newObject is effective (e.g. pods/status)
     // they will be used, otherwise the labels on the parent object will be used. If the
     // sub-resource does not have a parent object, the call may fail or skipped based on
     // the failure policy.
     //
     // See
     // https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
     // for more examples of label selectors.
     //
     // Default to the empty LabelSelector, which matches everything.
     // +optional
     ObjectSelector *metav1.LabelSelector `json:"objectSelector,omitempty" protobuf:"bytes,7,opt,name=objectSelector"`

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

* ConnectOptions is sent as the main object to the webhooks today (and it is mutable). Should we change that and send parent object as the main object?
* Update with design and test details for "convert to webhook-requested version"
* Update with design and test details for "mutating plugin ordering"

## Post-GA tasks

These are related Post-GA tasks:

* Allow mutating of existing objects in `DELETE` events
  * Interaction with Delete Collection List is ugly (patch items[0]...) and potentially problematic (cross-item patches)
  * Should adding/removing existing finalizers in delete admission be allowed?
  * Should GC finalizers based on `DeleteOptions` happen before or after mutating admission?
* If we have enough evidence, consider a better solution for plugin ordering like letting a webhook indicate how many times it should be rerun.

## Implementation History

* First version of the KEP being merged: Jan 29th, 2019
* The set of features for GA approved, initial set of features marked implementable: Feb 4th, 2019
* Implementation start for all approved changes: Feb 4th, 2019
* Target Kubernetes version for GA: TBD
