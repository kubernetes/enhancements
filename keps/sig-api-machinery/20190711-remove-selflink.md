---
title: Deprecate and remove SelfLink
authors:
  - "@wojtek-t"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-scalability
reviewers:
  - "@liggitt"
  - "@smarterclayton"
approvers:
  - "@lavalamp"
  - "@deads2k"
editor: "@wojtek-t"
creation-date: 2019-07-11
last-updated: 2019-07-24
status: implementable
---

# Deprecate and Remove SelfLink

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

`SelfLink` is a URL representing a given object. It is part of `ObjectMeta` and `ListMeta`
which means that it is part of every single Kubernetes object.

This KEP is proposing deprecating this field and removing it in an year according to our
`Deprecation policy`.

## Motivation

I haven't heard any really compelling reason for having `SelfLink` field. When modifying or
reading an object from kube-apiserver, its `Selflink` is set to exactly the URL that was
used to perform that operation, e.g.
```
apis/apps/v1/namespaces/default/deployments/deployment/status
```
So in order to get the object, client has to knew that URL anyway.

What is more, it leaves out exactly the thing that user can't tell from looking at a stored
object, which is what cluster and/or server it came from.

At the same time, setting this `SelfLink` field:
- is treated in a very special way in generic-apiserver - it is the only field that is being
set right before serializing the object (as this is the only place that has all the necessary
information to set it)
- has non-negligible performance impact - constructing the value performs couple memory
allocations (and memory allocations are things that have visible impact on Kubernetes
performance and scalability)

I propose to remove that field after necessary (long enough) deprecation period.

### Goals

- Eliminate performance impact caused by setting `SelfLink`
- Simplify the flow of generic apiserver by eliminating modifying objects late in the
processing path.

### Non-Goals

- Introduce location/source-cluster fields to ObjectMeta or ListMeta objects.

## Proposal

In v1.16, we will deprecate the `SelfLink` field in both `ObjectMeta` and `ListMeta`
objects by:
- documenting in field definition that it is deprecated and is going to be removed
- adding a release-note about field deprecation
We will also introduce a feature gate to allow disabling setting `SelfLink` fields
and opaque the logic setting it behind this feature gate.

In v1.20 (12 months and 4 release from v1.16) we will switch off the feature gate
which will automatically disable setting SelfLinks. However it will still be possible
to revert the behavior by changing value of a feature gate.

In v1.21, we will get rid of the whole code propagating those fields and fields themselves.
In the meantime, we will go over places referencing that field (see below) and get rid
of those too.

### Risks and Mitigations

The risk is that some users may significantly rely on this field in a way we are not aware of.
In that case, we rely on them start shouting loudly and 4 release before fields removal give
us time to revisit that decision.

## Design Details

I went through a k/k repo (including its staging repos) and all repos under [kubernetes-client][]
and this is the list of places that reference `SelfLink` fields (I excluded tests and all places
in apiserver responsible for setting it):

- [ ] https://github.com/kubernetes/kubernetes/blob/master/pkg/api/ref/ref.go
  Used for detecting version (which I believe should always be set?).
- [ ] https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/get/get.go
  Propagating SelfLink in kubectl get.
- [ ] https://github.com/kubernetes/kubernetes/blob/master/pkg/kubelet/config/common.go
  Doesn't seem to be really used anywhere.
- [ ] https://github.com/kubernetes/kubernetes/blob/master/pkg/printers/tablegenerator.go
  Setting SelfLink for table format.
- [ ] https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiextensions-apiserver/pkg/registry/customresource/tableconvertor/tableconvertor.go
  Setting SelfLink in conversion to table format for custom resources.
- [ ] https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/registry/rest/table.go
  Setting SelfLink in conversion to table format.
- [ ] staging/src/k8s.io/client-go/tools/reference/ref.go
  A copy of first item literally.

- [ ] https://github.com/kubernetes-client/java/blob/master/kubernetes/src/main/java/io/kubernetes/client/models/V1ListMeta.java
  Setter/getter and part of equals() and hashCode() methods.
- [ ] https://github.com/kubernetes-client/java/blob/master/kubernetes/src/main/java/io/kubernetes/client/models/V1ObjectMeta.java
  Setter/getter and part of equals() and hashCode() methods.
- [ ] https://github.com/kubernetes-client/csharp/blob/master/src/KubernetesClient/generated/Models/V1ListMeta.cs
  Setter/getter and constructor.
- [ ] https://github.com/kubernetes-client/csharp/blob/master/src/KubernetesClient/generated/Models/V1ObjectMeta.cs
  Setter/getter and constructor.
- [ ] https://github.com/kubernetes-client/go/blob/master/kubernetes/client/v1_list_meta.go
  Only part of type definition.
- [ ] https://github.com/kubernetes-client/go/blob/master/kubernetes/client/v1_object_meta.go
  Only part of type definition.
- [ ] https://github.com/kubernetes-client/ruby/blob/master/kubernetes/lib/kubernetes/models/v1_list_meta.rb
  Setter/getter.
- [ ] https://github.com/kubernetes-client/ruby/blob/master/kubernetes/lib/kubernetes/models/v1_object_meta.rb
  Setter/getter.
- [ ] https://github.com/kubernetes-client/perl/blob/master/lib/Kubernetes/Object/V1ListMeta.pm
  Seems like setter/getter to me.
- [ ] https://github.com/kubernetes-client/perl/blob/master/lib/Kubernetes/Object/V1ObjectMeta.pm
  Seems like setter/getter to me.
- [ ] https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_list_meta.py
  Setter/getter.
- [ ] https://github.com/kubernetes-client/python/blob/master/kubernetes/client/models/v1_object_meta.py
  Setter/getter.

[kubernetes-client]: https://github.com/kubernetes-client

### Test Plan

No new tests will be created - we expect all the tests to be passing at each phase of deprecation
and after removal of the fields.

### Graduation Criteria

The whole design is about meeting [Deprecation policy][deprecation-policy] - this doesn't
require more explanation.

[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

### Upgrade / Downgrade Strategy

No specific strategy is required.

### Version Skew Strategy

All the references to `SelfLink` should be removed early enough (2 releases before) the field
itself will be removed.

## Implementation History

2019-07-23: KEP merged.
2019-07-24: KEP move to implementable.
