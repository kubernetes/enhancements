---
title: Extended NodeRestrictions for Pods
authors:
  - "tallclair"
owning-sig: sig-auth
participating-sigs:
  - sig-node
  - sig-cluster-lifecycle
reviewers:
  - derekwaynecarr
  - neolit123
  - deads2k
approvers:
  - liggitt
  - derekwaynecarr
  - neolit123
  - deads2k
editor: TBD
creation-date: 2019-09-16
status: implementable
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
  - [PodStatus Restrictions](#podstatus-restrictions)
  - [OwnerReferences](#ownerreferences)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Breaking Services](#breaking-services)
    - [Namespace Annotation Policy](#namespace-annotation-policy)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Munge Mirror Pods](#munge-mirror-pods)
  - [MVP mitigation of known threats](#mvp-mitigation-of-known-threats)
  - [Restrict namespaces](#restrict-namespaces)
  - [Weaker label restrictions](#weaker-label-restrictions)
  - [Annotation Restrictions](#annotation-restrictions)
  - [Alternative Label Modifications](#alternative-label-modifications)
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

1. Restrict labels to per-namespace whitelisted keys
2. Prevent nodes from modifying labels through a `pod/status` update.
3. Restrict mirror pod OwnerReferences to only allow a node reference.

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
extensions will be guarded by the `MirrorPodNodeRestriction` feature gate.

Nodes are not granted the `update` or `patch` permissions on pods, but may update `pod/status`. All
label and owner reference updates will be forbidden through `pod/status` updates, so restrictions
will not be checked on status updates. In other words, a node _can_ update the status on a pod that
has un-whitelisted labels.

### Label Restrictions

A new reserved annotation will be introduced for namespaces to whitelist mirror-pod labels:

```
node.kubernetes.io/mirror.allowed-label-keys = "key1,key2,..."
```

When the NodeRestriction controller receives a mirror pod create a request for a node, it will check
the pod for labels. If it is labeled, the pod's namespace is checked for the allowed-label-keys
annotation. If any of the mirror-pod's labels are not whitelisted, or the annotation is absent, the
create request will be rejected.

The Kubelet does not currently label pods, nor are there official label keys that apply to
pods. However, there are a few labels that are commonly applied to system addons & static pods:

- `component` (used by [kubeadm][kubeadm-labels])
- `tier` (used by [kubeadm][kubeadm-labels])
- `k8s-app` (common on [addons][addons-k8s-app])

The `k8s-app` label is used to match controllers for system components, and therefore should be
explicitly disallowed.

`kubeadm` should be modified to whitelist the `component` and `tier` labels, or potentially drop
them if they're not required.

[kubeadm-labels]: https://github.com/kubernetes/kubernetes/blob/e682310dcc5d805a408e0073e251d99b8fe5c06d/cmd/kubeadm/app/util/staticpod/utils.go#L60
[addons-k8s-app]: https://github.com/kubernetes/kubernetes/blob/e682310dcc5d805a408e0073e251d99b8fe5c06d/cluster/addons/kube-proxy/kube-proxy-ds.yaml#L23

### PodStatus Restrictions

Some metadata can be modified through a `pod/status` subresource update. OwnerRefrences are
restricted from pod/status updates, but labels & annotations can be updated. Going forward, nodes
will be restricted from making any label changes through `pod/status` updates.

As the kubelet doesn't make any label updates through this request, this change will be rolled out
without a feature gate in **v1.17**.

### OwnerReferences

OwnerReferences can be set on mirror pods today. With the new restrictions, mirror pods are only
allowed a single owner reference (or none), and it must refer to the node:

```go
 metav1.OwnerReference{
  APIVersion: "v1"
  Kind: "Node"
  Name: node.Name
  UID:  node.UID
  Controller: true  // Prevent other controllers from adopting the pod.
  BlockOwnerDeletion: false
}
```

The Kubelet will start injecting this OwnerReference into mirror-pods in **v1.17**, unguarded by a
feature gate.

The node owner reference will eventually be required, but due to apiserver-node version skew, this
must happen at least 2 releases after nodes start injecting this OwnerReference.

### Risks and Mitigations

#### Breaking Services

Some Kubernetes setups depend on statically serving services today. Applying these mitigations will
likely break these clusters. There is no way to apply these changes in a fully backwards compatible
way, so users or operators of such clusters will be required to whitelist the required labels prior
to enabling the `MirrorPodNodeRestriction` feature gate (or upgrading to a release with the feature
gate enabled).

Here is a kubectl monstrosity for listing the labels that need to be whitelisted on each namespcae:

```
kubectl get pods --all-namespaces -o=go-template='{{range .items}}{{if .metadata.annotations}}{{if (index .metadata.annotations "kubernetes.io/config.mirror") }}{{$ns := .metadata.namespace}}{{range $key, $value := .metadata.labels}}{{$ns}}{{": "}}{{$key}}{{"\n"}}{{end}}{{end}}{{end}}{{end}}' | sort -u
```

This command gives output as `namespace: label-key` pairs, for example:

```
$ kubectl get pods --all-namespaces -o=go-template='{{range .items}}{{if .metadata.annotations}}{{if (index .metadata.annotations "kubernetes.io/config.mirror") }}{{$ns := .metadata.namespace}}{{range $key, $value := .metadata.labels}}{{$ns}}{{": "}}{{$key}}{{"\n"}}{{end}}{{end}}{{end}}{{end}}' | sort -u
kube-system: component
kube-system: extra
kube-system: tier
```

Which should be translated into an annotation on the kube-system namespace:

```
node.kubernetes.io/mirror.allowed-label-keys: "component,extra,tier"
```

Like so:

```
$ kubectl annotate namespaces kube-system node.kubernetes.io/mirror.allowed-label-keys="component,extra,tier"
```

#### Namespace Annotation Policy

There is prior art for representing namespaced policy through annotations, such as with the
[PodNodeSelector][] or [PodTolerationRestriction][] admission controllers. However, this is a model
we're trying to move away from for general namespaced-policy specification, so adding another
namespace annotation is considered a risk.

In this case, I think an exception is warranted given that:

1. Use of the annotation is very niche
2. The annotation is narrowly-scoped. Even when required, it will probably only be needed on a small
   number (typically 1) of namespaces.
3. The annotation is expected to be static. New namespaces won't need to be annotated, and the
   annotation value is tied to the cluster setup.

[PodNodeSelector]: https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#configuration-annotation-format
[PodTolerationRestriction]: https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#podtolerationrestriction

## Design Details

### Test Plan

Our CI environment does not depend on static pods serving services, so we can enable the feature
gate in the standard Kubernetes E2E environment. The restrictions can be verified by impersonating a
node's identity and ensuring illegal mirror pods cannot be created while legal pods can be.

To test the Kubelet modifications, a privileged pod can write a static pod manifest to a node and
verify the expected changes are made.

### Graduation Criteria

The feature gate will initially be in a default-disabled alpha state. Graduating to beta will make
the feature enabled by default, and will break users who have not taken steps to whitelist
mirror-pod labels.

Here is an approximate graduation schedule, specific release numbers subject to change:

- v1.17
  - MirrorPodNodeRestriction: **alpha**
- v1.19
  - MirrorPodNodeRestriction: **beta**
- v1.20
  - MirrorPodNodeRestriction: **GA**

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

## Implementation History

- 2019-09-16 - KEP proposed

## Alternatives

### Munge Mirror Pods

An alternative to outright rejecting invalid mirror pods, the NodeRestriction controller could
modify the mirror pods to conform to the restrictions. For example, the controller could:

- Remove illegal labels, and dump them into an annotation for audit purposes
  (e.g. `kubernetes.io/config.removed-labels`)
- Remove illegal owner references, also dumping them into an annotation
- Add the node owner reference (and require it)

A problem with this approach is that the Kubelet will not attempt to recreate a mirror pod with
illegal labels once the labels are whitelisted. In contrast, if the pod is outright rejected, then
as soon as its labels are whitelisted the Kubelet would try to recreate it and succeed. This
argument does not apply to the owner references, but the benefit of only modifying the owner
reference is weaker.

### MVP mitigation of known threats

An MVP of this proposal to mitigate the [2 motivating examples](#motivation) must include:

1. Prevent nodes from modifying arbitrary labels through `pod/status` updates.
2. Prevent nodes from setting arbitrary labels on mirror pods.
3. Prevent nodes from setting arbitrary owner references on mirror pods.

An MVP would exclude the speculative annotation restrictions. It could optionally take a blacklist
approach to label restrictions rather than a whitelist approach, but doing so would force every
service label to use the `node-restriction.kubernetes.io/` prefix to prevent the MITM threat.

### Restrict namespaces

For additional defense-in-depth, mirror pods could be restricted to whitelisted namespaces, for
example only namespaces with a `node.kubernetes.io/mirror.allowed` annotation. We may consider
this change in a future release.

### Weaker label restrictions

Alternatives to the whitelist restriction approach were considered.

**Whitelist prefix**

The original version of this proposal suggested using a `unrestricted.node.kubernetes.io/*` prefix
for whitelisted labels. This approach requires migrating static pods to a new label through a
complicated rollout procedure coordinated between services and multiple feature gates.

**Explicitly opt mirror pods out controllers**

Requires the controllers to check for the `kubernetes.io/config.mirror` annotation before matching a
pod. While we could make this change for internal controllers, there is no way to enforce it for
third-party controllers, so this approach would be less-safe. It also still requires the
`pod/status` label update restriction and owner ref restrictions to be complete.

**Blacklist labels**

Rather than forbidding all labels except those under `unrestricted.node.kubernetes.io/`, we could
_allow_ all labels except those under `restricted.node.kubernetes.io/`.

This change is more consistent with the [self-labeling restrictions on
nodes](0000-20170814-bounding-self-labeling-kubelets.md), but has a much broader impact. Pod labels
& label selectors are much more widely used than node labels, and less cluster dependent. This means
the labels are often set through third party deployment tools, such as helm. In order to safely
match labels, ALL labels consumed by controllers or other security-sensitive operations would need
to be moved to the blacklisted domain. Doing so would be a disruptive change, and force all labels
to be under the same domain.

**Whitelist configuration**

We could provide a configurable option (flag / ComponentConfig) to the NodeRestriction admission
controller to explicitly whitelist specific labels. This would be in addition to the
`unrestricted.node.kubernetes.io/` prefix. Alternatively, it could optionally include prefixes, and
we could make `"unrestricted.node.kubernetes.io/*"` be the default value of the option.

Providing this option is tempting, but it increases the configurable surface area with a
security-sensitive option that is easy to misunderstand. For example, a system service matching
mirror pods should explicitly opt-in to using the insecure labels to make the implications
explicit. If any labels can be whitelisted, it becomes harder to audit the cluster.

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

### Alternative Label Modifications

Several alternative label modification schemes were discussed, including:

- Out right rejecting pods with illegal labels
- Munging the labels to fit the allowed schema

For more details, see https://github.com/kubernetes/enhancements/pull/1243#issuecomment-540758654
