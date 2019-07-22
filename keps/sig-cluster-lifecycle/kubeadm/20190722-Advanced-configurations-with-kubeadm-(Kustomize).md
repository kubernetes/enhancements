---
title: Advanced configurations with kubeadm (Kustomize)
authors:
  - "@fabriziopandini"
owning-sig: sig-cluster-lifecycle
participating-sigs:
  - sig-cluster-lifecycle
reviewers:
  - "@neolit123"
  - "@rosti"
  - "@ereslibre"
  - "@detiber"
  - "@vincepri"
approvers:
  - "@timothysc"
  - "@luxas"
editor: "@fabriziopandini"
creation-date: 2019-07-22
last-updated: 2019-07-22
status: implementable
---

# Advanced configurations with kubeadm (Kustomize)

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Implementation Details](#implementation-details)
    - [Kustomize integration with kubeadm](#kustomize-integration-with-kubeadm)
    - [Providing and storing Kustomize patches to kubeadm](#providing-and-storing-kustomize-patches-to-kubeadm)
    - [Storing and retrieving Kustomize patches for kubeadm](#storing-and-retrieving-kustomize-patches-for-kubeadm)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP is aimed at defining a new kubeadm feature that will allow users to bootstrap 
a Kubernetes cluster with configuration options - control-plane or kubelet settings -
not supported by the Kubeadm config API.

## Motivation

Kubeadm currently allows you to define a limited set of configuration options for a
Kubernetes cluster via the Kubeadm config API or the corresponding CLI flags.
More specifically it allows:

1. To define configurations settings at cluster level using the `ClusterConfiguration`
   config object 
2. To define a limited set of configurations at the node level using the
   `NodeRegistrationOptions` object or the `localAPIEndpoint` object

The above set of configurations covers the most common use cases, but there are other
use cases that cannot be achieved with kubeadm config API as of today. Some examples:

- It is not possible to set/alter timeouts for liveness probes in control plane components.
- It is not possible to set/alter kubelet eviction policy at the node level. 

A common workaround for those limitations is to manually alter static pod manifests
or kubelet configuration after kubeadm init/join, but this is error-prone and changes
are not preserved during upgrades.

This KEP aims to overcome the limitations of the kubeadm config API by introducing a
new feature for defining “advanced configuration” for:

- control-plane static pod manifests stored into `/etc/kubernetes/manifests`.
- kubelet component config stored in `/var/lib/kubelet/config.yaml`.

### Goals

Considering the complexity of this topic, this document is expected to be subject
to some iterations. The goal of the current iteration is to:

- Get initial approval on Summary and Motivation paragraphs
- To identify a semantic for defining “advanced configurations” for control-plane
  or kubelet settings.
- To define UX for passing “advanced configurations” to kubeadm init and to kubeadm join.
- To define mechanics, limitations, and constraints for preserving “advanced configurations”
  during cluster lifecycle and more specifically for supporting the kubeadm upgrade workflow.
- To ensure the proper functioning of “advanced configurations” with kubeadm phases.
- To clarify what is in the scope of kubeadm and what instead should be the responsibility
  of the users/of higher-level tools in the stack like e.g. cluster API 

### Non-Goals

- To provide any validation or guarantee about the security, conformance, 
  consistency, of “advanced configurations” for control-plane or kubelet settings. 
  As it is already for `extraArgs` fields in the kubeadm config API or in the
  Kubelet/KubeProxy component config, the responsibility of proper usage of those 
  advanced configuration options belongs to higher-level tools/user's.
- To deprecate the Kubeadm config API because:
  - The config API defines a clear contract with kubeadm and the users.
  - The config API is well suited for most common use cases and it will remain the
    recommended way to define core settings for a cluster.
  - The config API implicitly defines the main cluster variants the kubeadm team 
    is committed to support and monitor in the Kubernetes test grid.
- To define how to manage “advanced configurations” for the etcd static pod manifest 
  (this is postponed until kubeadm - [`etcdadm`](https://github.com/kubernetes-sigs/etcdadm)
  project integration).
- To define how to manage “advanced configurations” for the addons 
  (this is postponed until kubeadm - [`addons`](https://github.com/kubernetes-sigs/addon-operators) 
  project integration).
- To define a solution for “advanced configurations” for the controller-manager
  component config (still WIP upstream; not yet integrated with kubeadm)

## Proposal

### User Stories 

#### Story 1
As a cluster administrator, I want to set kubelet eviction policy for a joining node
according to the node hardware features.

#### Story 2
As a cluster administrator, I want to set timeouts for the kube-apiserver liveness
probes for edge clusters.

#### Story 3
As a cluster administrator, I want to upgrade my cluster preserving all the
“advanced configuration” already in place

### Implementation Details

This proposal explores as a first option for implementing Kubeadm 
“advance configrations“ the usage of Kustomize; please refer to
[Declarative application management in Kubernetes](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/declarative-application-management.md)
and [Kustomize KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cli/0008-kustomize.md)
for background information about Kustomize.

Support for "advanced configrations“/kustomize in kubeadm will be available under
a new feature-flag named `kustomize`, initially alpha and disabled by default.

#### Kustomize integration with kubeadm

By adopting Kustomize in the context of Kubeadm, this proposal assumes to:

- Let kubeadm generate static pod manifests or kubelet configuration as usual.
- Use kubeadm generated artifacts as a starting point for applying patches
  containing “advanced configurations”.

This has some implications:

1. The higher-level tools/users have to express “advanced configurations” using
   one of the two alternative techniques supported by Kustomize - the [strategic
   merge patch](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/glossary.md#patchstrategicmerge)
   and the [JSON patch](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/glossary.md#patchjson6902) -.
2. The higher-level tools/users have to provide patches before running kubeadm;
   this point is going to be further discussed in the following paragraphs.
3. Kubeadm is responsible for coordinating the execution of Kustomize within the
   init/join/upgrade workflows
4. as a consequence of the previous point, higher-level tools/users are not
   requested to take care of defining `kustomization.yaml` files nor to define
   a local folder structure.

Additionally, in order to simplify the first implementation of this KEP, this 
proposal is going to assume that Kustomize patches for kubeadm are always defined
specifically for the node where kubeadm is going to be executed.

This point could be reconsidered in the future, by e.g. introducing cluster-wide
patches and/or patches for a subset of nodes.

#### Providing and storing Kustomize patches to kubeadm

Before kubeadm init, Kustomize patches should be eventually provided to kubeadm
by higher-level tools/users; patches should be provided in a well know location
like e.g. `/etc/kubernetes/kustomize` or in a custom location specified with a CLI flag. 
Eventually, in the future, this flag will be added to the kubeadm config API as well.

In order to simplify the first implementation of this KEP, this proposal is assuming
to use the same approach also for kubeadm join; this point could be reconsidered
in the future, by e.g. defining a method for allowing higher-level tools/users to
define Kustomize patches using a new CRD.

#### Storing and retrieving Kustomize patches for kubeadm

Kustomize patches, should be preserved during the whole cluster lifecycle, mainly
for allowing kubeadm to preserve changes during the kubeadm upgrade workflow.

In order to simplify the first implementation of this KEP, this proposal is assuming
that Kustomize patches will remain stored in the `/etc/kubernetes/kustomize` or in a
custom location specified with a CLI flag for the necessary time; this point could
be reconsidered in the future, by e.g. defining a method for allowing higher-level
tools/users to define Kustomize patches using a new CRD.

### Risks and Mitigations

_Confusion between kubeadm config API and kustomize_
Kubeadm already offers a way to implement cluster settings, that is the kubeadm API
and component configs. Adding a new feature for supporting “advanced configurations”
can create confusion in the users.

kubeadm maintainers should take care of making differences cristal clear in release notes
and feature announcement:

- The config API is well suited for most common use cases and it will remain the
  recommended way to define core settings for a cluster, while “advanced configurations”
  are designed for less common use cases.
- The config API implicitly defines the main cluster variants the kubeadm team is
  committed to support and monitor in the Kubernetes test grid, while instead higher-level
  tools/user are responsible for the security, conformance, consistency, of 
  “advanced configurations” for control-plane or kubelet settings. 

_To not provide expected flexibility_
In order to provide guarantee about kubeadm respecting “advanced configurations” during
init, join, upgrades or single-phase execution, it is necessary to define some trade-offs
around _what_ can be customized and _how_.

Even if the proposed solution is based on the user feedback/issues, the kubeadm maintainers
want to be sure the implemented is providing the expected level of flexibility and, in
order to ensure that, we will wait for at least one K8s release cycle for the users to provide 
feedback before moving forward in graduating the feature. 

Similarly, the kubeadm maintainers should work with [`etcdadm`](https://github.com/kubernetes-sigs/etcdadm)
project and [`addons`](https://github.com/kubernetes-sigs/addon-operators) project
to ensure a consistent approach across different components.

## Design Details

### Test Plan

Add at least a new E2E periodic test exercising  “advanced configurations” during init,
join and upgrades.

### Graduation Criteria

This proposal in its initial version covers only the creation of a new alpha features.
Graduation criteria will be defined in the following iterations on this proposal and
consider user feedback as well.

### Upgrade / Downgrade Strategy

As stated in goals, kubeadm will preserve “advanced configurations” during upgrades,
and more specifically, it will re-apply patches after each upgrade.

Downgrades are not supported by kubeadm, and not considered by this proposal.

### Version Skew Strategy

This proposal does not impact kubeadm compliance with official K8s version skew policy;
higher-level tools/user are responsible for the security, conformance, consistency of
“advanced configurations” that can impact or the aforementioned point.

## Implementation History

Tuesday, July 30, 2019
- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started

## Drawbacks

Kubeadm already offers a way to implement cluster settings, that is the kubeadm API
and the support for providing component configs. Adding a new feature for supporting
“advanced configurations” can create confusion in the users.

See risks and mitigations.

## Alternatives

There are many alternatives to “Kustomize” in the ecosystem; see [Declarative application management in Kubernetes](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/declarative-application-management.md).

While there is great value in several different approaches “Kustomize” was selected as
the first choice for this proposal because it already has first-class supported in
kubectl (starting from v1.14).
