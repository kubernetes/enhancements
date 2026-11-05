<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-5709: Add a well-known pod network readiness gate

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Definition of "network ready"](#definition-of-network-ready)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Condition type](#condition-type)
  - [Approach A: Network plugin webhook (no core changes)](#approach-a-network-plugin-webhook-no-core-changes)
  - [Approach B: Extend kubelet readiness logic (kubelet change)](#approach-b-extend-kubelet-readiness-logic-kubelet-change)
  - [Approach C: API server injects built-in readiness gate (API server change)](#approach-c-api-server-injects-built-in-readiness-gate-api-server-change)
  - [User Stories](#user-stories)
    - [Story 1: Preventing traffic black-holes during pod startup](#story-1-preventing-traffic-black-holes-during-pod-startup)
    - [Story 2: NetworkPolicy deny rules before traffic arrives](#story-2-networkpolicy-deny-rules-before-traffic-arrives)
    - [Story 3: Pods with multiple network devices via DRA](#story-3-pods-with-multiple-network-devices-via-dra)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Webhook behaviour](#webhook-behaviour)
  - [Node-agent PATCH flow](#node-agent-patch-flow)
  - [RBAC requirements](#rbac-requirements)
  - [Interaction with existing conditions](#interaction-with-existing-conditions)
  - [Worked example](#worked-example)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubernetes currently has no explicit signal for whether a pod
has been fully attached to the pod network and is ready to receive traffic.
The closest existing condition, [`PodReadyToStartContainers`][KEP-3085],
indicates that the pod sandbox has been created and CNI `ADD` has
returned — but not that the network datapath is fully programmed.
This KEP introduces a well-known [pod readiness gate][KEP-580]
condition that the network plugin sets to indicate network readiness,
cleanly separating application readiness (answered by readiness probes) from
network readiness (answered by the network plugin). This becomes
especially important as [KEP-4559] proposes to move kubelet probes to run
inside the pod network namespace, removing the implicit network
reachability signal that today's over-the-network probes
also provide.

[KEP-3085]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/3085-pod-conditions-for-starting-completition-of-sandbox-creation/README.md
[KEP-4559]: https://github.com/kubernetes/enhancements/issues/4559
[KEP-580]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/580-pod-readiness-gates/README.md

## Motivation

Today, kubelet readiness probes are executed from the host network
namespace, sending traffic over the pod network to reach the pod.
Because the probe traverses the pod network, a successful
readiness probe implicitly suggests both that the application is
ready to serve traffic and that the network plugin may have
assigned an IP and programmed the necessary routes and rules —
i.e., that the pod is reachable over the network. This behaviour pre-dates
Kubernetes 1.0 and is well established, but coupling these two
concerns in a single probe becomes a limitation as the network
stack evolves.

This implicit signal is also unreliable for two reasons. First,
because kubelet runs on the same node as the pod, a probe can
succeed over the local network path before routes or flows are
programmed on remote nodes — so the pod may not yet be reachable
from other nodes. Second, NetworkPolicy implementations
[explicitly exempt kubelet probes][np-pod-lifecycle] from policy
enforcement, so a probe can succeed before NetworkPolicy rules are
fully installed — meaning the pod may not yet be reachable by
peers on same node other than kubelet. The CNI plugin's `ADD` call typically
returns before the pod is fully plumbed into the network, because
blocking on full network programming would serialize pod creation
and significantly degrade bulk pod startup performance. The
existing [`PodReadyToStartContainers`][KEP-3085] condition
(originally named `PodHasNetwork`, renamed because the old name
was misleading) captures the moment CNI `ADD` returns and the
sandbox is ready — but this does not mean the network datapath is
fully programmed (e.g., OVS flows installed, nftables rules in
place, routes propagated to remote nodes). The readiness probe
happens to paper over this gap most of the time, but not always.

[KEP-4559] makes this problem explicit. It proposes moving TCP,
HTTP, and gRPC probes to run inside the pod network namespace using
CRI `PortForward()`, connecting to `localhost` rather than the pod
IP. This solves critical security and architectural problems (the
blind SSRF attack, the NetworkPolicy hole that exempts kubelet
probes, and constraints on network architectures with overlapping
pod IPs), but it means the probe no longer traverses the pod
network at all. Without a replacement signal, the following failure
scenario becomes possible:

1. A pod starts and its application begins listening on a port.
2. The new localhost-based readiness probe succeeds.
3. The pod is marked `Ready` and added to Service endpoints.
4. But the network plugin has not yet finished programming the network.
5. Traffic is routed to the pod and fails because it is not yet
   reachable.

Rather than trying to preserve the historical coupling between
probes and network reachability, this KEP proposes making network
readiness an explicit, first-class signal. Kubernetes already has a
mechanism for external controllers to participate in pod readiness
decisions: [pod readiness gates][KEP-580]. Network plugins can use
this mechanism to indicate when a pod's network is fully
programmed. The readiness probe then only needs to answer "is the
application processing connections?", while the network readiness
gate answers "can traffic reach this pod?" — a clean separation
that is more accurate and reliable than the implicit signal we
depend on today.

[np-pod-lifecycle]: https://kubernetes.io/docs/concepts/services-networking/network-policies/#pod-lifecycle

### Definition of "network ready"

What constitutes "network ready" is intentionally left to the network
plugin, because the details vary by implementation. As a general
guideline, the condition should be set to `True` when the pod's
datapath is fully programmed and the pod can receive traffic from
other pods in the cluster. Examples of what a plugin might wait for
include OVS flows or eBPF programs installed, nftables/iptables
rules for NetworkPolicy applied, or routes propagated to remote
nodes.

Plugins should make a best effort to ensure global reachability
before signaling readiness. If the plugin must update an API object
or other shared state to alert other nodes about the pod, it should
not mark the pod ready until that update has been made. However,
the plugin does not need to verify that every other node has
actually processed that update — it is sufficient that the
information has been published and every node *could* have
programmed access to the pod.

The plugin SHOULD NOT wait for external reachability from
outside the cluster (e.g., Ingress or cloud load balancers), as those
are separate concerns with their own readiness signals.

### Goals

- Define a well-known pod readiness gate condition type that network
  plugins set to signal that a pod's network datapath is fully
  programmed and the pod is reachable from other pods.
- Scope is pod-networked pods, where the network plugin must program
  the datapath after the sandbox is created.
- Provide a standard convention that all network plugins can adopt,
  so that the ecosystem converges on a single condition type rather
  than each plugin inventing its own.
- Cleanly separate application readiness (answered by readiness
  probes) from network readiness (answered by the network plugin),
  making the overall readiness model explicit rather than relying on
  the accidental coupling between over-the-network probes and
  network programming.

### Non-Goals

- Implementing this readiness gate in every network plugin
  (OVN-Kubernetes, Cilium, Calico, etc.). This KEP defines the
  convention and provides a reference implementation in `kindnet`;
  adoption by other network plugins is up to each project.
- Modifying kubelet's built-in readiness evaluation logic. The
  proposal builds on the existing [pod readiness gates][KEP-580]
  mechanism without changing how kubelet computes the `Ready`
  condition.
- Obligatory distributed multi-node network probing (e.g., verifying
  that a pod is reachable from every other node in the cluster). The
  network plugin is allowed to signal that the network is ready as
  soon as every node *could* have programmed access to it; it does
  not need to verify that every node actually has done so.
- Host-network pods (`hostNetwork: true`). These pods use the node's
  existing network namespace; there is no plugin-managed datapath to
  program.
- Replacing or changing existing readiness probes. Readiness probes
  continue to serve their current purpose of indicating application
  readiness; this KEP adds a complementary signal for network
  readiness.

## Proposal

This KEP defines a well-known pod condition type that network plugins
set to indicate that a pod's network datapath is fully programmed and
the pod is reachable from other pods. All three approaches below
share one thing:

- **Signaling.** The network plugin's node agent PATCHes
  `status.conditions` to set the condition to `True` when the
  datapath is ready.

The approaches differ in how the readiness gate enters
`spec.readinessGates` — which is the prerequisite for kubelet to
account for the condition when computing the pod's `Ready` status.

<<[UNRESOLVED which-approach ]>>

Reviewers: please comment on which approach you prefer. After
agreement, the chosen approach becomes the proposal and the other
two move to Alternatives.

<<[/UNRESOLVED]>>

### Condition type

<<[UNRESOLVED condition-type-naming ]>>

Three naming options are under consideration:

1. **`<plugin-domain>/network-ready`** (e.g., `cilium.io/network-ready`,
   `ovn-kubernetes.io/network-ready`) — each network plugin uses its
   own domain prefix. Ownership is unambiguous, and in multi-plugin
   clusters (e.g., Multus + DRA) each plugin can independently signal
   its own condition. Follows the custom pod condition naming
   convention described in [KEP-580]. Trade-off: there is no single
   condition name that operators and tooling can rely on across
   clusters, and Approach B (kubelet hardcodes the condition) would
   not work with per-plugin names. Additionally, without a single
   standard name there is no way to know for sure whether the feature
   is in use in a given cluster — though the well-known suffix
   (`/network-ready`) could be used by tooling to discover the
   condition regardless of the domain prefix.

2. **`networking.kubernetes.io/pod-network-ready`** — a single
   domain-qualified name under the `networking.kubernetes.io` prefix.
   Gives operators and tooling a consistent name to query across any
   cluster while still following the [KEP-580] naming convention.
   Trade-off: in multi-plugin clusters only one plugin can own the
   condition, or the plugins must coordinate who sets it. This also
   affects clusters where the pod network implementation consists of
   multiple unrelated pieces (e.g., `flannel` plus
   `kube-network-policies`) — both components affect whether the pod
   is fully reachable, but they may not coordinate with each other
   enough to produce a single condition.

3. **`PodNetworkReady`** — a short, unqualified name that mirrors
   the style of built-in conditions like `PodReadyToStartContainers`
   and `ContainersReady`. Same single-name benefits as option 2, but
   the unqualified form signals this is a core ecosystem convention
   rather than an extension. Same multi-plugin and multi-component
   trade-offs apply.

For the remainder of this KEP the placeholder `<NetworkReady>` is used
wherever the condition type appears. Reviewers: please comment on which
naming option you prefer.

<<[/UNRESOLVED]>>

### Approach A: Network plugin webhook (no core changes)

The network plugin deploys a mutating admission webhook that
intercepts pod `CREATE` requests and appends `<NetworkReady>` to
`spec.readinessGates`. Because readiness gates are immutable after
creation ([KEP-580]), the webhook must fire at pod creation time.
If the pod spec already contains the readiness gate (for example,
added by the user or a higher-level controller), the webhook is a
no-op. The plugin's node agent then PATCHes `status.conditions` to
set the condition to `True` when the datapath is ready.

- **Pro:** Follows [KEP-580]'s design exactly; no core Kubernetes
  changes required.
- **Con:** Every network plugin must independently implement the
  webhook.
- **Con:** The webhook sits in the pod creation path, adding
  latency to every pod create.
- **Con:** If the webhook is unavailable and `failurePolicy` is
  `Ignore`, pods are created without the gate and silently lose
  protection. Conversely, if `failurePolicy` is `Fail`, pod
  creation is blocked entirely while the webhook is down.
- **Con:** The network plugin needs pod `/status` PATCH permission
  in addition to the webhook, increasing the RBAC surface.

### Approach B: Kubelet natively checks a well-known condition (kubelet change)

Kubelet is modified to natively factor the well-known
`<NetworkReady>` condition from `status.conditions` into its `Ready`
computation — the same way it already factors in `ContainersReady`.
No readiness gate in `spec.readinessGates` is needed and no webhook
or spec mutation is involved. The network plugin only needs to PATCH
the status condition; kubelet does the rest.

- **Pro:** No `spec` mutation involved — no readiness gates, no
  webhook. The plugin only PATCHes a `status` condition, which is
  the simplest possible contract for plugin authors. There is
  precedent for this pattern: `ContainersReady` is already hardcoded
  into kubelet's `Ready` formula without being a readiness gate, and
  [KEP-3085] added `PodReadyToStartContainers` as another
  kubelet-managed well-known condition.
- **Con:** Requires a kubelet code change, increasing scope.
- **Con:** Unlike `ContainersReady` (which kubelet itself sets),
  `<NetworkReady>` would be set by an external agent — making
  kubelet's `Ready` computation depend on an out-of-tree component
  for the first time.
- **Con:** Kubelet has no way of knowing ahead of time whether
  the network plugin supports the condition. The plugin would need
  to set the condition to `False` during CNI ADD to signal that it
  will set it to `True` later, so kubelet knows to wait.
  Alternatively, kubelet could set it to `Unknown` before setting
  up the pod sandbox.

### Approach C: API server injects built-in readiness gate (API server change)

The API server automatically injects `<NetworkReady>` into
`spec.readinessGates` for every pod at creation time, making the
readiness gate built-in. The network plugin only needs to PATCH the
status condition to `True` when the datapath is ready.

- **Pro:** Every pod gets the readiness gate automatically,
  with no webhook needed, so no latency during pod create.
- **Con:** First-ever built-in readiness gate; steers away from
  [KEP-580]'s original design, which explicitly delegated readiness
  gate injection to external controllers via webhooks.
- **Con:** Backward-compatibility risk — if a network plugin does
  not set the condition, pods are stuck not-Ready forever. Requires
  a feature gate and careful rollout.

### User Stories

#### Story 1: Preventing traffic black-holes during pod startup

A platform team runs a large cluster with an overlay network plugin.
They observe occasional HTTP 5xx errors immediately after a
Deployment rolls out new pods, because the pods are marked `Ready`
and added to Service endpoints before the network plugin has finished
programming routes on remote nodes. After the network plugin adopts
this KEP's readiness gate, new pods are held out of endpoints until
the plugin confirms network readiness, eliminating the transient
errors.

#### Story 2: NetworkPolicy deny rules before traffic arrives

A compliance team requires that NetworkPolicy deny rules are
enforced before a pod receives any traffic. The
[Kubernetes docs][np-pod-lifecycle] already require that
implementations must not allow traffic that should be denied, but
may temporarily deny traffic that should be allowed during startup.
With this readiness gate, the network plugin can defer setting the
condition to `True` until the datapath is fully programmed,
providing an additional layer of assurance that the pod is not
added to Service endpoints prematurely. Note that this KEP does
not change the existing NetworkPolicy contract — it only provides
a signal that the cluster-default network datapath is ready.

#### Story 3: Pods with multiple network devices via DRA

<<[UNRESOLVED multi-network-scope ]>>

An HPC team uses Dynamic Resource Allocation (DRA) to attach
multiple network devices to a single pod — for example, a primary
cluster network interface plus a high-speed RDMA interface. Each
device may be programmed by a different plugin or driver, and each
has its own readiness timeline.

It is an open question whether the network readiness condition
should cover only the cluster-default pod network (since Services
today are always reached over that network) or all attached network
interfaces. [danwinship notes][dra-discussion] that secondary-network
readiness may be an application-level concern rather than a
network-readiness-gate concern, because there is no guarantee that
remote resources on the
secondary network are available even if the interface is plumbed.
How this interacts with future multi-network Service models is TBD.

[dra-discussion]: https://github.com/kubernetes/enhancements/pull/5995#discussion_r3039987684

<<[/UNRESOLVED]>>

### Notes/Constraints/Caveats

- **Added latency to `Ready`.** By design, the pod's `Ready`
  condition will not become `True` until the network plugin confirms
  the datapath is programmed. This adds time to the pod startup
  path relative to today's behaviour. The trade-off is intentional —
  a pod that is `Ready` before its network is functional causes
  worse problems (traffic black-holes, 5xx errors) than a pod that
  takes slightly longer to become `Ready`.

- **Interaction with `PodReadyToStartContainers`.** The
  `PodReadyToStartContainers` condition (from [KEP-3085]) indicates
  that the sandbox is created and CNI `ADD` has returned. The
  `<NetworkReady>` condition is a strictly later signal — it indicates
  that the full datapath is programmed. Both conditions can coexist;
  they answer different questions.

- **Multiple network plugins.** In clusters with more than one
  network plugin (e.g., Multus + DRA), the behaviour depends on the
  naming option chosen. With per-plugin names (option 1), each
  plugin signals its own condition independently. With a single
  well-known name (options 2 or 3), the plugins must coordinate who
  sets the condition, or a meta-controller must aggregate their
  signals.

- **(Approach A only) Webhook availability.** If the mutating
  webhook is unavailable and `failurePolicy` is `Ignore`, pods are
  created without the readiness gate and silently lose protection.
  If `failurePolicy` is `Fail`, pod creation is blocked while the
  webhook is down. Plugin authors should choose the failure policy
  that matches their users' risk tolerance.

- **(Approach A only) Existing pods.** Readiness gates are immutable
  after pod creation, so pods created before the webhook was deployed
  will not have the readiness gate. This is expected and safe — those
  pods continue to behave as they always have.

### Risks and Mitigations

| Risk | Applies to | Mitigation |
|------|-----------|------------|
| Plugin bug causes the condition to never be set to `True`, leaving pods stuck not-Ready. Especially severe for Approach C where every pod gets the gate automatically. | All | Plugin authors should implement a timeout or fallback. Operators can detect the issue by querying for pods where `ContainersReady` is `True` but `Ready` is `False` for an extended period. Approach C would additionally require a feature gate for safe rollout. |
| Extra API call (PATCH to pod status) per pod increases API server load. | All | The PATCH is a single, small write per pod startup — the same pattern already used by other controllers. Negligible compared to existing pod lifecycle writes. |
| Adoption is fragmented — some plugins adopt the convention, others don't, leading to inconsistent behaviour across clusters. | All | This KEP provides a clear, minimal convention. SIG Network can encourage adoption by listing compliant plugins in the KEP's implementation history, adding conformance tests, and documenting the convention on kubernetes.io. |
| RBAC misconfiguration prevents the plugin's node agent from PATCHing pod status. | All | Document the required RBAC rules (see Design Details). Plugin installation manifests should include the necessary ClusterRole / ClusterRoleBinding. |
| Webhook outage causes pods to be created without the readiness gate, silently losing protection. | A | Plugin authors should monitor webhook availability and alert on failures. Clusters that require the guarantee can use `failurePolicy: Fail`. |
| Webhook adds latency to the pod creation path. | A | The webhook performs a small, deterministic mutation (appending one entry to a list). Latency should be comparable to other mutating webhooks in the cluster. |
| Bug in kubelet readiness logic could affect all pods, not just those using network readiness. | B | The change should be small and well-tested. Kubelet already evaluates readiness gates; this adds one more condition to the same code path. |

## Design Details

### Webhook behaviour

The network plugin deploys a `MutatingWebhookConfiguration` that
targets pod `CREATE` operations. The webhook appends the well-known
readiness gate to `spec.readinessGates`:

```yaml
spec:
  readinessGates:
  - conditionType: "<NetworkReady>"
```

If the readiness gate is already present (injected by the user, a
Helm chart, or another controller), the webhook leaves the list
unchanged. The webhook should target pods that use CNI networking
(i.e., skip pods with `hostNetwork: true`).

### Node-agent PATCH flow

The plugin's node agent watches for pods on its node. When the agent
has finished programming the datapath for a pod, it issues a
strategic-merge PATCH against the pod's `/status` subresource:

```http
PATCH /api/v1/namespaces/<ns>/pods/<name>/status
Content-Type: application/strategic-merge-patch+json
```

```json
{
  "status": {
    "conditions": [
      {
        "type": "<NetworkReady>",
        "status": "True",
        "lastTransitionTime": "2025-07-01T12:00:00Z"
      }
    ]
  }
}
```

### RBAC requirements

The node agent needs permission to PATCH the `/status` subresource
of pods. A minimal ClusterRole looks like:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: network-readiness-agent
rules:
- apiGroups: [""]
  resources: ["pods/status"]
  verbs: ["patch"]
```

### Interaction with existing conditions

The `<NetworkReady>` condition complements the conditions kubelet
already manages:

| Condition | Set by | Meaning |
|-----------|--------|---------|
| `PodReadyToStartContainers` | kubelet | Sandbox created, CNI `ADD` returned |
| `ContainersReady` | kubelet | All containers passed their readiness probes (application is ready to serve) |
| `<NetworkReady>` | network plugin | Datapath fully programmed, pod is reachable from other pods |
| `Ready` | kubelet | All of the above are `True` (including all readiness gates) |

For pods that back a Service, the `Ready` condition determines
whether the pod is added to endpoints. Both the application
(readiness probes / `ContainersReady`) and the network
(`<NetworkReady>`) must be ready before traffic is routed to the
pod.

The timeline during pod startup is:

1. `PodReadyToStartContainers` becomes `True` — containers begin
   starting.
2. Readiness probes begin running. When all containers pass,
   kubelet sets `ContainersReady` to `True` — the application is
   ready to serve traffic. However, `<NetworkReady>` does not yet
   exist in `status.conditions`, so kubelet evaluates it as `False`
   per [KEP-580] semantics. `Ready` remains `False`.
3. The network plugin sets `<NetworkReady>` to `True`. Kubelet
   re-evaluates and sets `Ready` to `True`.
4. The endpoints controller adds the pod to the Service. Traffic
   flows only after both the application and the network are ready.

### Worked example

Consider a cluster running a network plugin that adopts this
convention. A user creates the following Deployment:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 2
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
      - name: web
        image: registry.k8s.io/e2e-test-images/agnhost:2.43
        ports:
        - containerPort: 80
        readinessProbe:
          httpGet:
            path: /healthz
            port: 80
          periodSeconds: 5
```

The user's manifest contains no mention of readiness gates — that
detail is handled transparently by the network plugin. After the
mutating webhook fires, the pod spec stored in etcd looks like:

```yaml
spec:
  readinessGates:
  - conditionType: "<NetworkReady>"
  containers:
  - name: web
    # ... same as above ...
```

Once the pod is fully started and the network plugin has signaled
readiness, the resulting pod status looks like:

```yaml
status:
  conditions:
  - type: PodReadyToStartContainers
    status: "True"
  - type: ContainersReady
    status: "True"
  - type: "<NetworkReady>"
    status: "True"
  - type: Ready
    status: "True"
```

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

The existing readiness gate mechanism ([KEP-580]) is already well
tested in kubelet. The tests below focus on validating the specific
interaction pattern this KEP defines: a network-readiness condition
set by an external agent gating the pod's overall `Ready` status
and Service endpoint membership.

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

The packages touched depend on the chosen approach. Coverage data
will be collected before implementation begins.

**Approach A (webhook):** No production code changes in
kubernetes/kubernetes. SIG Network will add mock-based unit tests
in k/k that simulate a network plugin injecting the readiness gate
and PATCHing the condition, to validate the convention works
correctly against kubelet's readiness evaluation logic.

- `pkg/kubelet/status`: `<date>` - `<test coverage>`

**Approach B (kubelet change):**

- `pkg/kubelet/status`: `<date>` - `<test coverage>`
  - Pod with `<NetworkReady>` condition absent remains not-Ready
    even when `ContainersReady` is `True`.
  - Pod with `<NetworkReady>` set to `True` and `ContainersReady`
    `True` becomes `Ready`.
  - Pod with `<NetworkReady>` set to `False` remains not-Ready.
  - Feature-gate disabled: kubelet ignores `<NetworkReady>` and
    computes `Ready` as before.

**Approach C (API server change):**

- `pkg/registry/core/pod`: `<date>` - `<test coverage>`
  - Readiness gate is injected into `spec.readinessGates` for new
    non-host-network pods.
  - Readiness gate is not injected for host-network pods.
  - Existing pods without the gate are not affected on update.
  - Feature-gate disabled: no readiness gate is injected.

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

Integration tests will verify the end-to-end readiness gate
lifecycle in a controlled environment:

- Create a pod with the `<NetworkReady>` readiness gate in
  `spec.readinessGates`. Verify that the pod's `Ready` condition
  remains `False` even after all containers are running and passing
  their readiness probes.
- PATCH the pod's `/status` subresource to set `<NetworkReady>` to
  `True`. Verify that the pod's `Ready` condition transitions to
  `True`.
- Create a Service selecting the pod. Verify that the pod's IP is
  NOT present in the EndpointSlice while `<NetworkReady>` is
  absent, and IS present after the condition is set to `True`.
- (Approach B/C) Verify feature-gate enable/disable behaviour:
  with the gate disabled, pods should become `Ready` without
  waiting for the `<NetworkReady>` condition.

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

E2e tests will validate the pattern in a real cluster:

- **Network readiness blocks endpoint membership:** Deploy a pod
  behind a Service with a readiness gate for `<NetworkReady>`.
  Verify the pod is not added to EndpointSlice until an external
  agent sets the condition to `True`. Verify traffic reaches the
  pod only after the condition is set.
- **Host-network pods are unaffected:** Deploy a host-network pod
  and verify it becomes `Ready` without needing a `<NetworkReady>`
  condition (the webhook should skip it, or the plugin should
  immediately set the condition to `True`).
- **Rollout behaviour:** Perform a Deployment rolling update where
  new pods have the readiness gate. Verify the rollout does not
  proceed until each new pod has both `ContainersReady` and
  `<NetworkReady>` set to `True`.

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
