---
title: declarative-field-manager
authors:
  - "@kwiesmueller"
owning-sig: sig-api-machinery
reviewers:
  - "@apelisse"
  - "@lavalamp"
approvers:
  - "@lavalamp"
creation-date: 2019-03-24
last-updated: yyyy-mm-dd
status: provisional
---

# declarative-field-manager

## Summary

Enable users to set the fieldManager, introduced with Server Side Apply, declaratively when sending the object to the apiserver.
The Server Side Apply feature introduced ownership of fields to improve merging and the object lifecycle when multiple actors interact with the same object.

While currently the identifier of the current actor (fieldManager) is set through an request option or the request user-agent,
this KEP aims to provide a declarative way of setting the current fieldManager through a field in the object itself.

## Motivation

It is a common practice to use Kubernetes manifests as stored configuration of an application. Either to just introduce version control to the cluster's state, or to automate the process of updating objects.

With the addition of Server Side Apply it could become necessary to change the way those manifests get applied, as the most benefit from this new feature requires every actor to define it's fieldManager. Doing this, currently is only possible through api options, the user-agent or as a kubectl option.

To keep the way of interacting with manifests as easy as possible, this KEP suggests to allow defining the fieldManager as part of the manifest itself. Therefor no further action is required when applying a set of manifests and users can easily benefit from Server Side Apply without having to change their apply workflow.

This would also assist existing solutions already interacting with manifest files, as the fieldManager information gets provided in a way they already know to handle which might improve compatibility.

### Goals

- Allow users to set their fieldManager information declaratively in their manifests

### Non-Goals

- Replace the existing ways to set the fieldManager information

## Proposal

To achieve the goals, we would add a field to ObjectMeta called `options`, this field would contain options for the api server being only `fieldManager` for now, but might get extended in the future.

This field should not get persisted to storage and is solely for sending request options information to the apiserver.
When the `options.fieldManager` field is not set, the apiserver would fallback onto current behavior and default the fieldManager (or fail).

This means, we add a field that is optional metadata, write-only and non-persisted.
The `options.fieldManager` should take a non empty string or be unset and should follow the same criteria as [ManagedFieldsEntry.Manager](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.14/#managedfieldsentry-v1-meta).

Setting or not setting the field will cause the following behavior for both apply and non-apply operations:

- If the fieldManager request option is set for the request (for example through kubectl), it will get used as manager
- If the newly introduced `options.fieldManager` field is set in the received request, it will get used as manager
- If both of the  two options are set and do not match, the request will get rejected.

An example of setting the field would be:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: example
  options:
    fieldManager: jenkins
data:
  k: v
```

### User Stories [optional]

#### Git Managed Configuration

Alice has a repo containing her current set of Kubernetes manifests as yaml in git.
The repo gets managed using Gitflow.
Master branch is automatically applied to the production Kubernetes cluster by a push triggered Jenkins job.
Staging branch is automatically applied to the staging Kubernetes cluster by a push triggered Jenkins job.
All members of her team file pull requests against the repo to bring changes into staging and production.

Aside from that, Alice and her team got services that directly manage their objects, like controllers and webhooks,
as well as services that updated labels for some objects.

They want to make use of the new Server Side Apply, so concurrent changes to objects from different actors are possible without unwanted overlap/conflicts.

To do so, assuming the changes in this KEP, Alice would add the newly introduced field to all manifests in their config management repo. The field would be set to e.g. `configManagement` (or their repo name).
Additionally, she would update the manifests sent by different actors (like controllers) to contain the field with a value like `controllerName`. Note that this might already be the case for some through the defaulting to the user-agent inside the apply and update pathway.

If now the team updates their repo with a change to their manifests, Jenkins again applies them as usual. On the apiserver side though, the fieldManager set in the applied manifest gets used and all fields set now get owned by the `configManagement` fieldManager.

Again as usual other actors step in to change the objects. They as well use their fieldManager value either set in the manifest file they apply, or based on their user-agent/configuration.
One of those actors is setting annotations for every service with external information.
As the actor now only updates annotations not already owned by the `configManagement` fieldManager, applying them will cause the actor to own them.

All actors now can coexist and manage their set of fields without interfering with each other.

#### Kustomize

In reference to the above workflow, it would be possible to override `options.fieldManager` for differen Kustomize overlays and therefor separate ownership depending on the actor currently active.

This means, updating single fields by using a Kustomize overlay (like updating images, labels or annotations) as part of a CI/CD pipeline could easily contain the fieldManager information without any changes to Kustomize (or similar tools) itself.

### Risks and Mitigations

One risk might be, that introducing this change would mean there is a field that does not get persisted to storage.
As a result, setting it causes different behavior on the apiserver, but is not reflected when reading the object back from the apiserver.

This can seem unintuitive but should be solved by proper documentation.

## Design Details

### Proposed API Change

```go
type ObjectMeta struct {
...
  // Options used by the apiserver when handling the object.
  // This field is write-only, non-persisted and optional.
  Options *MetaOptions `json:"options,omitempty" protobuf:"bytes,18,opt,name=options"`
...
}

// MetaOptions allow to set options used by the apiserver when handling objects.
// This field is write-only, non-persisted and optional.
// Options here are defined respective to their apiserver query parameters. Setting either one of them should cause the same effects. Setting MetaOptions and query parameters to different values, should fail the request.
// Setting options that are not known by the apiserver will have no effect and unknown fields get discarded.
type MetaOptions struct {
  // FieldManager is a name associated with the actor or entity that is responsible for the currently taking place
  // interaction with the object.
  // This field is write-only, non-persisted and optional.
  // It is only used by the apiserver on create, apply and update operations, to set the ManagedFields accordingly.
  // The value must be unset or less than or 128 characters long, and only contain printable characters, as defined by https://golang.org/pkg/unicode/#IsPrint.
  //
  // If the field is unset, the apiserver will default to the request user-agent.
  // If the request contains the fieldManager option, it acts like this field.
  // If both this field or the requests fieldManager option are set, but not equal the apply and non-apply operation will fail.
  //
  // This field is alpha and can be changed or removed without notice.
  //
  // +optional
  FieldManager string `json:"fieldManager,omitempty" protobuf:"bytes,19,opt,name=fieldManager"`
}
```

### Test Plan

e2e and integration tests should be added accordingly.

### Graduation Criteria

TBD

### Upgrade / Downgrade Strategy

No fields get persisted.
When the field is not set due to an old client, the current Server Side Apply defaulting is the fallback.
When the field is set by a new client, it will get ignored by an old apiserver.

### Version Skew Strategy

## Implementation History

- 30. Mar 2019: @kwiesmueller started implementing the [KEP](https://github.com/kubernetes/kubernetes/pull/75917)
- 05. July 2019: changes to the field name and location were made in response to PR feedback

## Drawbacks [optional]

See the risks outlined above.

## Alternatives [optional]

As an alternative to the unpersisted field, a custom annotation might be used.
This already got used for the previous `last-applied` annotation in kubectl.
However this approach seems wrong, as it would persist information in the object that is of no value and will probably change very often. Additionally using annotations for this seems like a workaround and for the apply to work we would have to strip the annotation from the managedFields object, which then would cause even more unintuitive behavior as a dedicated field, as one special annotation will not get persisted instead of a field that is explicitly declared to behave like that.
