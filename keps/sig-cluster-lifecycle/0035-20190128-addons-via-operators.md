KEP: Addons via Operators

---
kep-number: 35
title: Addons via Operators
authors:
  - "@justinsb"
owning-sig: sig-cluster-lifecycle
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-01-28
last-updated: 2019-01-28
status: provisional
---

# Addons via Operators

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Infrastructure Needed](#infrastructure-needed)


## Summary

We propose to use operators for managing cluster addons.  Each addon will have
its own CRD, and users will be able to perform limited tailoring of the addon
(install/don’t install, choose version, primary feature selection) by modifying
the CR.  The operator encodes any special logic (e.g. dependencies) needed to
install the addon.

We will create tooling to make it easy to build addon operators that follow the
best practices we identify as part of this work.  For example, we expect that
most addons will be declarative, and likely be specified as part of a “cluster
bundle”, so we will make it easy to build basic addon operators that follow
these patterns.

We hope that components will choose to maintain their own operators, encoding
their knowledge of how best to operate their addon.


## Motivation

Addons are components that are managed alongside the lifecycle of the cluster.
They are often tied to or dependent on the configuration of other cluster
components.  Management of these components has proved complicated.  Our
existing solution in the form of the bash addon-manager has many known
shortcomings and is not widely adopted.  As we focus more development outside of
the kubernetes/kubernetes repo, we expect more addon components of greater
complexity.  This is one of the long-standing backlog items for
sig-cluster-lifecycle.

Use of operators is now generally accepted, and the benefits to other
applications are generally recognized.  We aim to bring the benefits of
operators to addons also.

### Goals

* Explore the use of operators for managing addons
* Create patterns, libraries & tooling so that addons are of high quality,
  consistent in their API surface (common fields on CRDs, use of Application
  CRD, consistent labeling of created resources), yet are easy to build.
* Build addons for the basic set of components, acting as a quality reference
  implementation suitable for production use.  We aim also to demonstrate the
  utility and explore any challenges, and to verify that the tooling does make
  addon-development easy.


### Non-Goals

* We do not intend to mandate that all installation tools use addon operators;
  installation tools are free to choose their own path.
* Management of non-addons is out of scope (for example installation of end-user
  applications, or of packaged software that is not an addon)


## Proposal

This is the current plan of action; it is based on experience gathered and work
done for Google’s GKE-on-prem product.  However we don’t expect this will
necessarily be directly applicable in the OSS world and we are open to change as
we discover new requirements.

* Extend kubebuilder & controller-runtime to make it easy to build operators for
  addons
* Build addons for the primary addons currently in the cluster/ directory
* Plug in those addons operators into kube-up / cluster-api / kubeadm / kops /
  others (subject to those projects being interested)
* Develop at least one addon operator outside of kubernetes/kubernetes
  (LocalDNS-Cache?) and figure out how it can be used despite being out-of-tree
* Investigate use of webhooks to prevent accidental mutation of child objects
* Investigate the RBAC story for addons - currently the operator must itself
  have all the permissions that the addon needs, which is not really
  least-privilege.  But it is not clear how to side-step this, nor that any of
  the alternatives would be better or more secure.
* Investigate use of patching mechanisms (as seen in `kubectl patch` and
  `kustomize`) to support advanced tailoring of addons.  The goal here is to
  make sure that everyone can use the addon operators, even if they “love it but
  just need to change one thing”.  This ensures that the addon operators
  themselves can remain bounded in scope and complexity.


We expect the following functionality to be common to all operators for addons:

* A CRD per addon
* Common fields in spec that define the version and/or channel
* Common fields in status that expose the current health & version information
  of the addon
* Addons follow a common structure, with the CR as root object, an Application
  CR, consistent labels of all objects
* Some form of protection or rapid reconciliation to prevent accidental
  modification of child objects
* Operators are declaratively driven, and can source manifests via https
  (including mirrors), or from data stored in the cluster itself
  (e.g. configmaps or cluster-bundle CRD, useful for airgapped)
* Operators are able to expose different update behaviours: automatic immediate
  updates; notification of update-available in status; purely manual updates
* Operators are able to observe other CRs to perform basic sequencing
* Addon manifests are able express an operator minimum version requirement, so
  that an addon with new requirements can require that the operator be updated
  first


### Risks and Mitigations

This will involve running a large number of new controllers.  This will require
more resources; we can mitigate this by combining them into a single binary
(similar to kube-controller-manager).

Automatically updating addons could result in new SPOFs, we can mitigate this
through mirroring (including support for air-gapped mirrors).

Providing a good set of addons could result in a monoculture where mistakes
affect most/all kubernetes clusters (even if we don’t mandate adoption, if we
succeed we hope for widespread adoption).  We can continue with our strategies
that we use for core components such as kube-apiserver: primarily we must keep
the notion of stable vs less-stable releases, to stagger the risk of a bad
rollout.  We must also consider this a trade-off against the risk that without
coordination each piece of tooling must reinvent the wheel; we expect more
mistakes (even measured per cluster) in that scenario.

## Graduation Criteria

We will succeed if addon operators are:

* Used: addon operators are adopted by the majority of cluster installation
tooling
* Useful: users are generally satisfied with the functionality of addon
operators and are not trying to work around them, or making lots of proposals /
PRs to extend them
* Ubiquitous: the majority of components include an operator
* Federated: the components maintain their own operators, encoding their
knowledge of how best to run their addon.


## Implementation History

Addon Operator session given by jrjohnson & justinsb at Kubecon NA - Dec 2018
KEP created - Jan 29 2019

## Infrastructure Needed

Initial development of the tooling can probably take place as part of
kubebuilder

We should likely create a repo for holding the operators themselves.  Eventually
we would hope these would migrate to the various addon components, so we could
also just store these under e.g. cluster-api.

Unclear whether this should be a subproject?
