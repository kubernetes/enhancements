---
title: declarative-field-manager
authors:
  - "@kwiesmueller"
owning-sig: sig-api-machinery
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-03-24
last-updated: yyyy-mm-dd
status: implemented
---

# declarative-field-manager

## Summary

Enable users to set the fieldManager, introduced with Server Side Apply, declaratively when sending the object to the apiserver.
The Server Side Apply feature introduced ownership of fields to improve merging and improve the object lifecycle when multiple actors interact with the same object.

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

To realize the goals, we would add a field to ObjectMeta called `fieldManager`, right next to the existing `managedFields`.
This field should not get persisted to storage and is solely for sending fieldManager information to the apiserver.
When the field is not set, the apiserver would fallback onto current behavior and default the fieldManager (or fail).

This means, we add a field that is optional metadata, write-only and non-persisted.
The field should take a non empty string or be unset and should follow the same criteria as [ManagedFieldsEntry.Manager](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.14/#managedfieldsentry-v1-meta).

Setting or not setting the field will cause the following behavior for both apply and non-apply operations:

- If the fieldManager option is set for the request (for example through kubectl), it will get used as manager
- If the newly introduced `fieldManager` field is set in the received request, it will get used as manager
- If both of the  two options are set and do not match, the request will get rejected.

An example of setting the field would be:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: example
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

### Risks and Mitigations

One risk might be, that introducing this change would mean there is a field that does not get persisted to storage.
As a result, setting it causes different behavior on the apiserver, but is not reflected when reading the object back from the apiserver.

This can seem unintuitive and we need to make sure documentation is right on this.

## Design Details

### Proposed API Change

```go
type ObjectMeta struct {
...
  // ManagedFields maps workflow-id and version to the set of fields
  // that are managed by that workflow. This is mostly for internal
  // housekeeping, and users typically shouldn't need to set or
  // understand this field. A workflow can be the user's name, a
  // controller's name, or the name of a specific apply path like
  // "ci-cd". The set of fields is always in the version that the
  // workflow used when modifying the object.
  //
  // This field is alpha and can be changed or removed without notice.
  //
  // +optional
  ManagedFields []ManagedFieldsEntry `json:"managedFields,omitempty" protobuf:"bytes,17,rep,name=managedFields"`
...
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

## Drawbacks [optional]

See the risks outlined above.

## Alternatives [optional]

As an alternative to the unpersisted field, a custom annotation might be used.
This already got used for the previous `last-applied` annotation in kubectl.
However this approach seems wrong, as it would persist information in the object that is of no value and will probably change very often. Additionally using annotations for this seems like a workaround and for the apply to work we would have to strip the annotation from the managedFields object, which then would cause even more unintuitive behavior as a dedicated field, as one special annotation will not get persisted instead of a field that is explicitly declared to behave like that.
