---
title: Extended NodeRestrictions for Pods
authors:
  - "tallclair"
owning-sig: sig-auth
participating-sigs:
  - sig-node
  - sig-cluster-lifecycle
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-09-16
status: provisional
---

# Extended NodeRestrictions for Pods

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Background](#background)
  - [Threat Model](#threat-model)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Label Restrictions](#label-restrictions)
  - [Annotation Restrictions](#annotation-restrictions)
  - [OwnerReferences](#ownerreferences)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [MVP mitigation of known threats](#mvp-mitigation-of-known-threats)
  - [Restrict namespaces](#restrict-namespaces)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in
[kubernetes/enhancements] referencing this KEP and targeting a release milestone **before
[Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases) of the targeted
release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e.,
[kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be
updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link
      to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to
      [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list
      discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked
`implementable` should be approved by each of the KEP approvers. If any of those approvers is no
longer appropriate than changes to that list should be approved by the remaining approvers and/or
the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement
is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Extend the [NodeRestriction][] admission controller to add further limitations on a node's effect on
pods:

1. Restrict labels to a whitelisted prefix: `unrestricted.node.kubernetes.io/`
2. Restrict mirror pod OwnerReferences to only allow a node reference.

## Motivation

The [Node Authorizer][] and associated [NodeRestriction][] controllers introduced the concept of
node isolation: making it possible to prevent a compromised node from compromising the whole
cluster. A key step is limiting the node's access to resources to only those required by pods
running on the node. For example, a node can only read secrets referenced by pods on that node.

However, there are other controllers in the cluster interacting with pods on a node, and under some
circumstances those controllers can be manipulated to do attacker's will. This is known as a
["confused deputy attack"](https://en.wikipedia.org/wiki/Confused_deputy_problem).

Examples include:

- Making a pod match a service selector in order to [man-in-the-middle][] (MITM) the service
  traffic.
- Making a pod match a {ReplicaSet,StatefulSet,etc.} controller so the controller deletes
  legitimate replicas, thereby DoSing the application.

There are likely other 3rd party controllers that could also be manipulated.

In order to mitigate these attack scenarios, the node (and all pods running on the node) must not be
able to manipulate pods in a way that they're matched by controllers.

### Background

The Kubelet has two mechanisms that can be used to manipulate pods. The first is through the
pod/status subresource. Despite the name, an update to pod/status can also update (some) of the
pod's metadata. Of particular interest are updates to labels and annotations. Note that pod/status
updates _cannot_ modify the OwnerReferences [[1][]].

The second mechanism is through the creation of "mirror pods". The Kubelet can run pods from other
sources than the API server, such as a static manifest directory or pulled from an HTTP
server. These pods are referred to as "static pods". Since static pods don't come from the API
server, components that read pods from the API wouldn't know about them. To compensate, the Kubelet
creates a "mirror pod", which reflects the state of the static pod it's running.

Mirror pods have some special properties. They are identified with a special annotation,
`kubernetes.io/config.mirror`, and the Kubelet is only authorized to create mirror pods, and only on
the same node (itself). The Kubelet won't run a mirror pod (since it's actually running a static
pod). Mirror pods are also restricted from using a service account, or secrets, configmaps,
persistent volumes, and other resources restricted by the node authorizer, in order to prevent an
attacker from bypassing the authorization by creating mirror pods.

[1]: https://github.com/kubernetes/kubernetes/blob/ab73a018de51bddf9d03d6fed6e867b60196c796/pkg/registry/core/pod/strategy.go#L162-L171

[Node Authorizer]: https://kubernetes.io/docs/reference/access-authn-authz/node/
[NodeRestriction]: https://kubernetes.io/docs/reference/access-authn-authz/node/
[man-in-the-middle]: https://en.wikipedia.org/wiki/Man-in-the-middle_attack

### Threat Model

At a high level, this proposal targets clusters making effective use of node isolation to separate
sensitive workloads or limit the blast radius of a successful node compromise. More specifically, it
makes the following assumptions:

- The cluster uses scheduling constraints such as node selectors & taints / tolerations to separate
  workloads of different trust or privilege levels.
- An attacker has compromised a low-privilege node meaning they have code execution as root in the
  host namespaces (i.e. a container escape). Low-privilege in this case means the node is not
  hosting any privileged workloads with permissions to trivially take over the cluster
  (e.g. unrestricted pod creation, read secrets, etc).

### Goals

- Prevent a compromised node from manipulating controllers to execute confused deputy attacks.

### Non-Goals

- Solving all node isolation issues.
- Solving all possible man-in-the-middle attacks.

## Proposal

All restrictions will be enforced through the [NodeRestriction][] admission controller. These
extensions will be guarded by the `NodeRestrictionPods` feature gate.

On update, only the delta will be restricted. In other words, the Kubelet can modify pods with
restricted labels / owner references as long as it's not modifying (or adding or
deleting) the restricted entries.

### Label Restrictions

The Kubelet will be prevented from updating pod labels or creating mirror pods with labels, except
for whitelisted keys:

- Any labels starting with `unrestricted.node.kubernetes.io/` are allowed.

The Kubelet does not currently label pods, nor are there official label keys that apply to
pods. However, there are a few labels that are commonly applied to system addons & static pods:

- `component` (used by [kubeadm][kubeadm-labels])
- `tier` (used by [kubeadm][kubeadm-labels])
- `k8s-app` (common on [addons][addons-k8s-app])

The `k8s-app` label is used to match controllers for system components, and therefore should be
explicitly disallowed.

It is not clear how the `tier` and `component` labels are consumed, but I recommend uses be migrated
to `unrestricted.node.kubernetes.io/component` and `unrestricted.node.kubernetes.io/tier`, rather
than special casing the existing labels.

[kubeadm-labels]: https://github.com/kubernetes/kubernetes/blob/e682310dcc5d805a408e0073e251d99b8fe5c06d/cmd/kubeadm/app/util/staticpod/utils.go#L60
[addons-k8s-app]: https://github.com/kubernetes/kubernetes/blob/e682310dcc5d805a408e0073e251d99b8fe5c06d/cluster/addons/kube-proxy/kube-proxy-ds.yaml#L23

### OwnerReferences

OwnerReferences cannot be updated through the `pod/status` subresource, but they can be set on
mirror pods. With the new restrictions, mirror pods are only allowed a single owner reference (or
none), and it must refer to the node:

```go
 metav1.OwnerReference{
  APIVersion: "v1"
  Kind: "Node"
  Name: node.Name
  UID:  node.UID
  Controller: nil // or false
  BlockOwnerDeletion: nil // all values allowed
}
```

### Risks and Mitigations

Some Kubernetes setups depend on statically serving services today. Applying these mitigations will
likely break these clusters. There is no way to apply these changes in a fully backwards compatible
way, so instead we will rely on a staged rollout through the `NodeRestrictionPods` feature gate, and
call out the actions required.

Clusters currently depending on label-matching static pods will need to migrate the
labels to the new whitelisted key prefix. This can be done in a non-disruptive way,
but requires a multistep process. For example, to migrate static pods providing a Service:

1. Update the static pods (by deploying updated static manifests) with _both_ the old & new labels.
2. Update the service selector to match the new labels.
3. Update the static pods to remove the old labels.

## Design Details

### Test Plan

Our CI environment does not depend on static pods serving services, so we can enable the feature
gate in the standard Kubernetes E2E environment. The restrictions can be verified by impersonating a
node's identity and ensuring illegal mirror pods cannot be created.

### Graduation Criteria

The feature gate will initially be in a default-disabled alpha state. Graduating to beta will make
the feature enabled by default, but users that have not yet updated existing label
usage to the unrestricted keys can still disable it. We will allow at least 2 releases before
migrating to GA and removing the feature gate entirely.

### Upgrade / Downgrade Strategy

Upgrade / downgrade is only meaningful for enabling / disabling the feature gate. If no explicit
action is taken, this will happen on upgrade when the feature graduates to beta.

Enabling the feature will not affect pods that are already running. If a new static pod is deployed,
or a node needs to (re)create a static pod with an illegal label, that mirror pod will be
rejected. The Kubelet will still run the pod locally, but it will not be exposed through the
Kubernetes API, controllers won't be able to find it, and the scheduler may not account for its
resources.

Rolling back / disabling the feature will not affect existing pods. If a mirror pod was previously
rejected, the Kubelet will attempt to recreate it and it will now be allowed.

### Version Skew Strategy

In an HA environment, it's possible for some apiservers to have the feature enabled and others
disabled. In this case, violating may or may not be allowed. Since the feature doesn't affect
existing pods, a violating pod will continue to be allowed by another server that does have the
restrictions enabled.

## Implementation History

- 2019-09-16 - KEP proposed

## Alternatives

### MVP mitigation of known threats

An MVP of this proposal to mitigate the [2 motivating examples](#motivation) must include:

1. Prevent nodes from modifying arbitrary labels through `pod/status` updates.
2. Prevent nodes from setting arbitrary labels on mirror pods.
3. Prevent nodes from setting arbitrary owner references on mirror pods.

An MVP would exclude the speculative annotation restrictions. It could optionally take a blacklist
approach to label restrictions rather than a whitelist approach, but doing so would force every
service label to use the `node-restriction.kubernetes.io/` prefix to prevent the MITM threat.

### Restrict namespaces

For additional defense-in-depth, mirror pods could be restricted to whitelisted namespaces. Doing so
would be a more disruptive change, but is something we could consider in the future.

### Annotation Restrictions

In addition to label & owner restrictions, annotation keys could be restricted too. I am still open
to adding these restrictions in a future extension, but doing so is contingent on concrete use
cases.

Under these restrictions, the Kubelet would be prevented from updating pod annotations or creating
mirror pods with annotations, except for whitelisted keys:

1. Any annotations starting with `unrestricted.node.kubernetes.io/` are allowed.
2. Annotations the Kubelet currently uses are allowed:
- `ConfigMirrorAnnotationKey = "kubernetes.io/config.mirror"`
- `ConfigHashAnnotationKey = "kubernetes.io/config.hash"`
- `ConfigFirstSeenAnnotationKey = "kubernetes.io/config.seen"`
- `ConfigSourceAnnotationKey = "kubernetes.io/config.source"`
3. Well-known annotations that may be used on static pods are allowed:
- `PodPresetOptOutAnnotationKey = "podpreset.admission.kubernetes.io/exclude"
- `SeccompPodAnnotationKey = "seccomp.security.alpha.kubernetes.io/pod"`
- `SeccompContainerAnnotationKeyPrefix = "container.seccomp.security.alpha.kubernetes.io/"` (prefix
  match)
- `ContainerAnnotationKeyPrefix = "container.apparmor.security.beta.kubernetes.io/"` (prefix match)
- `PreferAvoidPodsAnnotationKey = "scheduler.alpha.kubernetes.io/preferAvoidPods"`
- `BootstrapCheckpointAnnotationKey = "node.kubernetes.io/bootstrap-checkpoint"`

