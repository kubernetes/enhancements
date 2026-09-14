# KEP-6061: OCI Artifact-Based Security Profile Distribution

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Distributing Seccomp Profiles Across a Fleet](#story-1-distributing-seccomp-profiles-across-a-fleet)
    - [Story 2: Vendor-Provided AppArmor Profiles (beta)](#story-2-vendor-provided-apparmor-profiles-beta)
    - [Story 3: Agentic Workloads with Admin-Controlled Baselines](#story-3-agentic-workloads-with-admin-controlled-baselines)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Kubernetes API Changes](#kubernetes-api-changes)
  - [CRI API Changes](#cri-api-changes)
  - [Kubelet Behavior](#kubelet-behavior)
  - [Profile Merging](#profile-merging)
    - [Observability](#observability)
  - [OCI Artifact Format](#oci-artifact-format)
  - [Profile Verification](#profile-verification)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [CRI conformance tests (critest)](#cri-conformance-tests-critest)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
  - [Security Profiles Operator (SPO)](#security-profiles-operator-spo)
  - [CRI-Runtime-Only Pull](#cri-runtime-only-pull)
  - [Dynamic Resource Allocation (DRA)](#dynamic-resource-allocation-dra)
  - [Node Resource Interface (NRI)](#node-resource-interface-nri)
  - [Extending PullImage with Media Type](#extending-pullimage-with-media-type)
  - [Kubernetes API Object (ConfigMap with OCI Source)](#kubernetes-api-object-configmap-with-oci-source)
  - [Annotation-Based Approach](#annotation-based-approach)
  - [Kubelet-Managed Pull into the Localhost Profile Directory](#kubelet-managed-pull-into-the-localhost-profile-directory)
  - [Init Container Writing the Profile to a Volume](#init-container-writing-the-profile-to-a-volume)
  - [Earlier Iterations of This KEP](#earlier-iterations-of-this-kep)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation, e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes native support for pulling security profiles from
OCI-compatible registries, starting with seccomp in alpha and adding AppArmor
in beta. Today, Kubernetes requires security profiles to be pre-installed on
every node (`Localhost` type) or limits users to the built-in
`RuntimeDefault`. This creates operational burden for cluster administrators
who must distribute and synchronize profiles across all nodes using external
tooling such as DaemonSets, node image baking, or the
[Security Profiles Operator (SPO)][spo].

[spo]: https://github.com/kubernetes-sigs/security-profiles-operator

By extending the Kubernetes API and CRI to support OCI artifact references for
security profiles, users can store versioned, immutable profiles in container
registries alongside the container images they protect and reference them by
digest. The kubelet resolves pull credentials and passes them to the CRI
runtime, which pulls the artifacts using the same registry infrastructure
already in place for container images.

The design supports a layered profile model: pods can specify both a base
profile (Localhost or RuntimeDefault) and an OCI overlay profile. The CRI
runtime merges the OCI profile with the base profile and its own configured
baseline via intersection, producing an effective profile that is at least as
restrictive as every input. This guarantees that OCI-distributed profiles
cannot weaken node security regardless of their content, while allowing
administrators to enforce per-workload-class baselines without breaking
existing OCI profiles when the baseline changes. The converse also holds: an
OCI profile can never grant an operation that the node's baseline denies.
Workloads that need a profile looser than the baseline continue to use
`Localhost`.

## Motivation

Security profiles are critical for defense-in-depth in container environments.
Seccomp filters restrict syscall access and AppArmor confines filesystem and
network operations. Despite their importance, adopting custom security profiles
in Kubernetes remains difficult because of the distribution problem: profiles
must exist on every node before pods can reference them.

The current options are:

- **Bake profiles into node images**: Tightly couples profile versions to node
  image releases. Updating a profile requires rolling all nodes.
- **DaemonSet-based distribution**: Fragile, race-prone (pods may start before
  profiles are distributed), and difficult to version.
- **Security Profiles Operator (SPO)**: Solves distribution well but requires
  installing a full operator with CRDs, RBAC, and a webhook. This is
  significant overhead for users who just want to reference a profile.

OCI artifacts are the natural distribution mechanism. They are versioned,
content-addressable, and signable. Container registries are already part of
every Kubernetes deployment. CRI-O has shipped OCI artifact support for seccomp
profiles since v1.30 (via pod annotations), proving the concept works in
production. SPO has supported OCI artifact distribution for seccomp profiles
since v0.8.0 using [ORAS][oras]. This KEP promotes that pattern to a
first-class Kubernetes API feature, covering seccomp in alpha and AppArmor in
beta with a uniform approach.

[oras]: https://oras.land/

Emerging use cases in AI agent sandboxing further motivate this work. Projects
like [NVIDIA OpenShell][openshell] use per-container security profiles for
fine-grained isolation of AI agent workloads. Native profile distribution via
OCI artifacts would let platform teams publish and version seccomp and AppArmor
profiles alongside the agent images they protect.

[openshell]: https://github.com/NVIDIA/OpenShell

### Goals

- Add an `OCI` profile type to the Kubernetes `SeccompProfile` API type
  (alpha) and, once the artifact format is settled, to `AppArmorProfile`
  (beta), allowing users to reference security profiles stored in
  OCI-compatible container registries.
- Extend the CRI API with a dedicated `PullSecurityProfileArtifact` RPC and an
  `OCI` profile type in the `SecurityProfile` message, enabling the kubelet to
  pull profiles via the runtime and pass digest-pinned references to
  sandbox/container creation calls.
- Integrate with [Node Declared Features (KEP-5328)][ndf] so that pods using
  OCI profiles are only scheduled to nodes whose kubelet supports them, and
  report runtime support via CRI `RuntimeFeatures` so that the kubelet
  rejects such pods at admission when the CRI runtime does not.
- Reuse existing image pull infrastructure (pull secrets, credential providers,
  registry authentication) for profile pulls.
- Reference profiles by immutable digest. Mutable tag references are deferred
  to beta (see [Notes/Constraints/Caveats](#notesconstraintscaveats)).
- Support layered profiles where a pod specifies both a base profile
  (Localhost or RuntimeDefault) and an OCI overlay. The CRI runtime merges
  the profiles via intersection, so the effective profile is at least as
  restrictive as every input.

### Non-Goals

- **Landlock profile support in the Kubernetes API or CRI**: Landlock does not
  yet have a finalized profile format in the OCI runtime specification
  (runtime-spec [PR #1241][landlock-pr]). The architecture proposed here is
  designed to accommodate landlock once the OCI runtime spec and runc add
  support, but this KEP does not define the landlock profile format or API
  fields. The `PullSecurityProfileArtifact` RPC and the profile merge
  library (see [Profile Merging](#profile-merging)) are profile-type-agnostic
  and extend to landlock by adding a new `SecurityProfileKind` enum value, a
  new OCI config media type, and a landlock merge implementation in the
  library.
  Landlock profiles are always additive (they can only narrow access, never
  widen it), which simplifies merging compared to seccomp or AppArmor.
- **SELinux profile distribution**: SELinux uses a fundamentally different
  model from seccomp and AppArmor. Policy modules are compiled and installed
  system-wide via `semodule`, not applied per-container. The Kubernetes API
  uses `SELinuxOptions` (labels) rather than a `SecurityProfile` type, and
  neither the CRI nor existing tooling (SPO, CRI-O) supports OCI artifact
  distribution for SELinux profiles. SELinux profile distribution may be
  addressed in a follow-up KEP if demand emerges.
- **Profile recording or generation**: SPO provides profile recording via its
  log enricher and BPF recorder. This KEP focuses on distribution only.
- **Profile admission or policy enforcement**: Deciding which profiles are
  allowed is the domain of admission controllers (e.g., Pod Security Standards,
  Kyverno, OPA/Gatekeeper). The built-in PodSecurity admission controller
  is updated to accept the `OCI` type in `restricted` namespaces only in
  beta, once the seccomp fallthrough fix has reached the oldest supported
  kubelet (see [Design Details](#kubernetes-api-changes)).
- **Replacing SPO**: SPO provides a broader feature set including profile
  recording, base profile composition, and webhook-based binding. This KEP
  addresses the core distribution primitive that SPO and other tools can build
  upon.

[landlock-pr]: https://github.com/opencontainers/runtime-spec/pull/1241
[kep-2535]: https://github.com/kubernetes/enhancements/issues/2535

## Proposal

### User Stories

#### Story 1: Distributing Seccomp Profiles Across a Fleet

A platform team maintains a set of hardened seccomp profiles for their
microservices. Today they bake profiles into node images, causing tight coupling
between profile and node release cycles. With this feature, they push profiles
to their existing container registry and reference them by digest in pod
specs:

```yaml
securityContext:
  seccompProfile:
    type: OCI
    oci:
      reference: "registry.example.com/security/profiles/api-server-seccomp@sha256:3b1c..."
```

The CRI runtime pulls the profile using credentials resolved by the kubelet
from the pod's `imagePullSecrets`. Profile updates are decoupled from node
image updates: rolling out a new profile version means updating the digest in
the pod template, the same workflow as updating an image digest.

#### Story 2: Vendor-Provided AppArmor Profiles (beta)

A database vendor ships a recommended AppArmor profile for their containerized
database. Without this feature, customers must manually install the profile on
every node. Once AppArmor support lands in beta, the vendor publishes the
profile (in the structured JSON format described in
[OCI Artifact Format](#oci-artifact-format)) to a public registry and
customers reference it directly:

```yaml
securityContext:
  appArmorProfile:
    type: OCI
    oci:
      reference: "vendor-registry.io/database/apparmor-profile@sha256:abc123..."
```

#### Story 3: Agentic Workloads with Admin-Controlled Baselines

A platform team runs AI agent workloads that need fine-grained seccomp profiles
per agent type. The cluster admin places a restrictive baseline seccomp profile
on each node and uses admission policy to require all pods in the `agentic`
namespace to reference it as their base profile. Individual agent pods layer an
OCI profile on top for workload-specific restrictions:

```yaml
securityContext:
  seccompProfile:
    type: OCI
    oci:
      reference: "registry.example.com/agents/code-exec-seccomp@sha256:9d4e..."
      baseProfile:
        type: Localhost
        localhostProfile: "agentic-baseline.json"
```

The CRI runtime merges the OCI profile with the Localhost base profile and the
node's runtime baseline via intersection. The effective profile is at least as
restrictive as all three inputs. When the admin tightens the Localhost baseline
(for example, blocking a newly discovered dangerous syscall), all existing OCI
profiles continue to work because the merge automatically applies the new
restriction. No OCI profiles need to be updated.

### Notes/Constraints/Caveats

- **Digest-only references**: In alpha, `oci.reference` must be digest-pinned
  (`registry/repository@sha256:...`); API validation rejects tag references.
  A security profile reference should be immutable, and digest-only
  references keep the alpha design small: there is no tag resolution, no
  pinned state to persist across kubelet restarts, and no status field to
  report which content was applied, because the spec already says so.
  Rolling out a new profile version means updating the digest in the pod
  template and rolling pods, the same workflow as updating an image digest.
  Tag references are planned for beta. They require the kubelet to resolve
  each tag once per pod sandbox, persist the resolved reference in the
  sandbox metadata so that every container in the sandbox uses the same
  content across restarts, and report the resolved reference in container
  status for auditing. Deferring tags lets alpha validate the pull, merge,
  and apply path first.
- **Runtime profile caching**: The CRI runtime caches profile artifacts in
  its content-addressable storage, keyed by digest. Whether this is the same
  store as container image layers (containerd's content store) or a dedicated
  artifact store (CRI-O's `artifacts` store) is a runtime implementation
  detail. The runtime must not evict an artifact that is referenced by an
  existing pod sandbox or container. If the artifact is nevertheless missing
  at container creation time (for example, after a manual cleanup of the
  runtime's storage), the kubelet re-pulls it by digest on the next sync. If
  the registry no longer serves that digest, the `PullSecurityProfileArtifact`
  call fails and the container creation fails with a
  `SecurityProfilePullFailed` event. The pod is not silently left without a
  profile: it remains running (profiles are loaded into the kernel at
  creation time), but any new container creation within the pod (restart, new
  init container) fails until the digest is available again.
- **Pre-pulling**: Whether an artifact pulled via the regular `PullImage`
  API is visible to `PullSecurityProfileArtifact` depends on the runtime's
  storage layout. CRI-O, for example, keeps OCI artifacts in a store that is
  separate from the container image store, so a `PullImage` of a profile does
  not warm the profile cache. The portable way to pre-pull profiles is a pod
  (for example, from a DaemonSet) that references the profile in its own
  `securityContext`, which drives the pull through the same
  `PullSecurityProfileArtifact` path that regular workloads use. The runtime
  validates the artifact's media type, layer count, and size on every
  `PullSecurityProfileArtifact` call regardless of how the content arrived in
  the store.
- **Registry availability and retry semantics**: If the registry is unreachable,
  pods that reference artifact profiles will fail to start, similar to how image
  pull failures prevent pod startup. Profile caching mitigates this for
  steady-state operation. Profile pulls follow the kubelet's existing retry
  model: each pod sync attempt triggers one `PullSecurityProfileArtifact`
  call per unique reference, which returns immediately once the artifact is
  present. The kubelet's existing exponential backoff governs retry timing.
  There is no separate retry policy for profile pulls. If a profile remains
  unavailable, the pod stays in `Pending` state while the kubelet retries
  sandbox creation with backoff. For Jobs, setting `activeDeadlineSeconds`
  bounds the total time before the Job is marked as failed. Without it, the
  pod remains pending indefinitely, the same behavior as an unresolvable
  container image reference.
- **Profile size limits**: Security profiles are typically small (under 100 KB
  for seccomp, under 50 KB for AppArmor). The CRI runtime must enforce a
  maximum artifact size to prevent abuse (CRI-O defaults to 1 MiB,
  configurable). This limit is an implicit contract for compliant CRI
  implementations, not communicated via the CRI protocol itself. The CRI spec
  will document the recommended default limit (1 MiB) and the requirement that
  runtimes reject oversized artifacts. Additionally, runtimes must validate that
  the artifact contains only the expected profile layer and reject artifacts
  with unrelated layers. This prevents abuse scenarios where an attacker
  pre-pulls large layers unrelated to profiles to consume node disk space or
  cache resources. The size limit alone does not bound merge cost: the merge
  is linear in the number of distinct syscalls but quadratic in the number
  of entries that name the same syscall, and a 1 MiB profile can hold
  several thousand entries for one syscall. The merge library's artifact
  validation therefore caps entries per syscall
  (`MaxArtifactEntriesPerSyscall`, 128), which keeps the worst case at a few
  milliseconds per syscall; its benchmarks cover both the 1 MiB limit and
  the cap.
- **No pull policy field**: Unlike container images, which have
  `imagePullPolicy`, security profile artifacts always use pull-if-not-present
  semantics. Because references are digest-pinned, the content behind a
  reference never changes, so a pull policy would only control whether the
  registry is contacted when the artifact is already present. A pull policy
  field can be added in a future iteration if demand emerges (the
  `SecurityProfileOCIArtifact` struct allows this without API breakage).
- **Intersection with KEP-2535 (Ensure Secret Pulled Images)**: Because
  profile artifacts use pull-if-not-present semantics, the same credential
  reuse concern from [KEP-2535][kep-2535] applies: a profile pulled by one pod
  is available in the runtime's storage for another pod on the same node that
  lacks credentials for that registry. KEP-2535 closes this gap for images by
  having the kubelet record which credentials pulled each image and by
  re-issuing `PullImage` (which always resolves the reference against the
  registry) when a pod's credentials have not been verified for that image.
  For alpha, profile artifacts are **not** covered by this verification: the
  kubelet's image pull manager is not consulted for profile pulls, and
  `PullSecurityProfileArtifact` skips the registry entirely when the
  digest-pinned reference is already present. The exposure is bounded. Profile
  content is never exposed to the container, and the effective profile is the
  intersection with the node baseline, so the worst case is that a pod applies
  a restriction set it lacks credentials to read, comparable to a `Localhost`
  profile being available to every pod on the node. A pod author who already
  knows a private profile's digest can also learn whether it is cached on a
  node by observing whether a pod referencing it starts without credentials;
  the same probe exists for images pulled with `IfNotPresent` today and is
  what KEP-2535 closes. Beta integrates profile
  pulls with the pull manager: the kubelet records the credentials used per
  resolved reference and, when `imagePullCredentialsVerificationPolicy`
  requires verification for a pod, calls `PullSecurityProfileArtifact` in a
  mode that forces the runtime to resolve the manifest against the registry
  before reusing local content (a request field to be defined for beta). This
  is a beta graduation criterion.
- **Static pods**: Static pods do not have `imagePullSecrets` on the pod spec
  (there is no API server to resolve service account secrets). For static pods,
  credential resolution falls back to the kubelet's configured credential
  providers and any node-level registry auth configuration, the same behavior
  as container image pulls for static pods. The mirror pod is created from
  the same manifest and carries the `oci` field; if the API server rejects
  it because its feature gate is disabled, the static pod still runs on the
  node without a mirror pod, as with any static pod whose mirror fails
  validation.
- **Privileged containers**: CRI runtimes do not apply seccomp profiles to
  containers running with `privileged: true`, and CRI-O also skips AppArmor
  for them (containerd applies a `Localhost` AppArmor profile even to
  privileged containers). Existing profile types are accepted on privileged
  containers for backward compatibility even though they may be ignored.
  `type: OCI` is a new value, so API validation rejects any privileged
  container whose effective seccomp profile (and, from beta, AppArmor
  profile) is `OCI`. The effective profile is the container-level profile if
  set, otherwise the pod-level profile. A pod-level `OCI` profile is
  therefore allowed only if every privileged container in the pod (including
  init and ephemeral containers) overrides it with a non-OCI container-level
  profile. Ephemeral containers are added through the `ephemeralcontainers`
  subresource after pod creation, so the same rule is enforced in that
  update's validation path. Rejecting early prevents confusion where a user
  specifies a profile that is silently ignored.
- **Layered profile semantics**: When the pod spec specifies a `baseProfile`
  alongside the OCI reference, the CRI runtime intersects the runtime's
  configured baseline, the pod-spec base profile, and the OCI profile. The
  runtime baseline is always the floor, the base profile adds
  per-workload-class restrictions, and the OCI profile adds per-workload
  restrictions. No profile is rejected for being too permissive; the merge
  constrains it and logs what was constrained. See
  [Profile Merging](#profile-merging) for the full semantics.
- **OCI profiles cannot loosen the baseline**: Because the merge is an
  intersection, an OCI profile can only remove permissions relative to the
  runtime's configured baseline (`RuntimeDefault` unless the node admin
  configures otherwise). Workloads that need syscalls or accesses the baseline
  denies (for example `bpf`, `perf_event_open`, or `unshare` without extra
  capabilities under the default seccomp profiles of containerd and CRI-O)
  cannot obtain them through an OCI profile. They continue to use `Localhost`
  profiles, or the node admin loosens the runtime baseline for the node pool
  that hosts them. This is a deliberate trade-off: distribution convenience
  for profiles that tighten, node admin control for anything that loosens.
- **Linux-only**: Seccomp and AppArmor are Linux security mechanisms. This
  feature applies only to Linux containers. The CRI changes are scoped to
  `LinuxContainerSecurityContext` and `LinuxSandboxSecurityContext`. Windows
  containers are unaffected.

### Risks and Mitigations

**Risk**: Pulling profiles adds latency to pod startup.
**Mitigation**: The CRI runtime caches pulled profiles locally
(content-addressable by digest). After the first pull, subsequent pods using
the same profile reference hit the local cache. References are digest-pinned,
so cached content is never re-pulled.

**Risk**: Malicious, tampered, or over-permissive profiles could weaken node
security. With `Localhost` profiles, node admins control which profiles exist
on disk; OCI profiles let pod authors reference any artifact from any
registry.
**Mitigation**: The CRI runtime merges every pulled OCI profile with the
node's baseline via intersection (see [Profile Merging](#profile-merging)),
so the effective profile is never more permissive than the baseline
regardless of the profile's source or content. Even if a profile is tampered
with in transit or a registry is compromised, the worst case is an effective
profile equal to the baseline. The node admin controls the baseline through
the runtime's configuration, and pods can add a `baseProfile` for
per-workload-class restrictions that admission policy can require. The
runtime also validates profile content before applying it. Signature
verification through the runtime's existing policy (for example CRI-O's
`/etc/containers/policy.json`) and admission webhooks that restrict allowed
references add control over provenance on top of the content guarantee.

**Risk**: A pod with `type: OCI` that reaches a kubelet older than v1.38
runs without any seccomp profile, because `fieldSeccompProfile` in those
kubelets silently falls through to `Unconfined` for unknown types.
**Mitigation**: Node Declared Features keeps the scheduler from placing such
pods on kubelets that do not declare the feature, and the fallthrough fix
([kubernetes/kubernetes#141958][k8s-141958]) makes v1.38 and later kubelets
fail closed. What neither covers is a pod created with `spec.nodeName`
pointing at a pre-1.38 kubelet, or an ephemeral container with an `OCI`
profile added to a pod already running on one, so enabling the gate by
default waits until
the fix is in the `.0` release of the oldest supported kubelet (v1.41), the
PodSecurity `restricted` check does not accept `OCI` before that floor, and
operators enabling the alpha gate are advised to do so only on clusters
whose kubelets are all v1.38 or later (see
[Version Skew Strategy](#version-skew-strategy)).

**Risk**: Garbage collection of cached profile artifacts could force
unnecessary re-pulls.
**Mitigation**: Profiles are loaded into the kernel at container creation time,
so GC of the cached artifact does not affect already-running containers.
However, if the cached artifact is removed while the pod is still running, any
new container creation (restart, new init container) would require a re-pull.
Runtimes must not report profile artifacts in `ListImages`; otherwise the
kubelet's image garbage collector would treat them as unused images and remove
them under disk pressure. Profile artifacts are therefore invisible to the
kubelet's garbage collector, and for alpha, profile GC is entirely
runtime-managed: the runtime must protect artifacts referenced by an existing
pod sandbox or container and may evict unreferenced artifacts according to
its own policy. Given the 1 MiB size limit and the small number of distinct
profiles per node, never evicting is an acceptable alpha policy. The number
of distinct artifacts is not bounded, however, so a node that runs a stream
of short-lived pods with distinct profiles accumulates them; operators
enabling the alpha should monitor the runtime's artifact store usage (for
CRI-O, the `artifacts` directory under the storage root), and runtimes
should expose its count and byte size as metrics, until kubelet-driven
garbage collection lands in beta. Beta adds
`ListSecurityProfileArtifacts` and `RemoveSecurityProfileArtifact` RPCs so
that the kubelet can enumerate and remove artifacts in the same way it
manages images today.

**Risk**: Increased registry load from profile pulls.
**Mitigation**: Profiles are small (typically < 100 KB) and aggressively cached.
The additional registry load is negligible compared to container image pulls.

**Risk**: Registry as a single point of failure (denial of service).
**Mitigation**: Registry unavailability only affects pods that reference uncached
OCI profiles. Once a profile is cached locally by the CRI runtime, pods start
without registry access. References are digest-pinned and never need
re-resolution. Operators can mitigate registry dependency by using registry
mirrors, pre-pulling profiles via DaemonSet pods that reference them (see
[Pre-pulling](#notesconstraintscaveats)), and monitoring
`kubelet_security_profile_artifact_pull_errors_total` for early warning of
registry issues. The blast radius is limited to new pods referencing uncached
profiles; already-running pods are unaffected.

**Risk**: Lateral access to other pods' cached profiles.
**Mitigation**: Profile artifacts cached by the CRI runtime are stored in the
runtime's content store, which is not directly accessible from within
containers. Profile content is applied to the kernel (seccomp BPF filters,
AppArmor policy) at container creation and is not exposed as a file inside the
container's filesystem. A container cannot read or modify the profile applied
to another container on the same node. This is the same trust model as
`Localhost` profiles: a pod that references another pod's profile (whether by
`Localhost` path or OCI reference) does not gain any new access. The profile is
applied by the runtime, not exposed to the container. An attacker with root
access on the node could read the runtime's content store, but node-level root
access already implies full control over all containers. For multi-tenant
clusters, ensure that container breakout mitigations (seccomp, AppArmor, user
namespaces) are in place and that registry credentials do not grant
cross-tenant access to profile artifacts.

## Design Details

### Kubernetes API Changes

Extend `SeccompProfile` in `k8s.io/api/core/v1` with a new type and reference
field:

```go
type SeccompProfile struct {
    // type indicates which kind of seccomp profile will be applied.
    // Valid options are:
    //   RuntimeDefault, Localhost, Unconfined, OCI
    // +unionDiscriminator
    Type SeccompProfileType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=SeccompProfileType"`

    // localhostProfile indicates a profile defined in a file on the node.
    // Must be a descending path, relative to the kubelet's configured seccomp
    // profile location. Must be set if type is "Localhost". Must NOT be set
    // for any other type.
    // +optional
    LocalhostProfile *string `json:"localhostProfile,omitempty" protobuf:"bytes,2,opt,name=localhostProfile"`

    // oci specifies an OCI artifact containing the security profile.
    // Must be set if type is "OCI". Must NOT be set for any other type.
    // +featureGate=SecurityProfileOCIArtifact
    // +optional
    OCI *SecurityProfileOCIArtifact `json:"oci,omitempty" protobuf:"bytes,3,opt,name=oci"`
}

// SecurityProfileOCIArtifact specifies an OCI artifact reference for a
// security profile, with an optional base profile for layered enforcement.
type SecurityProfileOCIArtifact struct {
    // reference is the OCI artifact reference.
    // The format is a fully-qualified, digest-pinned OCI reference:
    // registry/repository@sha256:<digest>. Short names and tag references
    // are rejected.
    Reference string `json:"reference" protobuf:"bytes,1,opt,name=reference"`

    // baseProfile optionally specifies a base profile that the OCI profile
    // is merged with via intersection. The CRI runtime merges the OCI
    // profile, this base profile, and the runtime's own configured baseline;
    // the effective profile permits an operation only if all inputs permit it.
    // This allows administrators to enforce per-workload-class baselines
    // (e.g., a restrictive Localhost profile for agentic workloads) while
    // letting pod authors layer additional restrictions via OCI profiles.
    // When omitted, the CRI runtime uses its configured default baseline
    // (RuntimeDefault or an admin-supplied profile) as the only base.
    // +optional
    BaseProfile *SecurityProfileOCIBase `json:"baseProfile,omitempty" protobuf:"bytes,2,opt,name=baseProfile"`
}

// SecurityProfileOCIBase specifies the base profile for layered OCI profile
// enforcement. The CRI runtime merges the OCI profile with this base profile
// (and its own configured baseline) via intersection: the effective profile
// permits an operation only if all inputs permit it.
type SecurityProfileOCIBase struct {
    // type specifies the kind of base profile.
    // Valid options are: RuntimeDefault, Localhost.
    // Localhost is valid for seccomp profiles. AppArmor profiles (beta)
    // accept RuntimeDefault only, because a kernel-loaded Localhost
    // AppArmor profile has no text that could be merged.
    // +unionDiscriminator
    Type SecurityProfileOCIBaseType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=SecurityProfileOCIBaseType"`

    // localhostProfile specifies the base profile path on the node.
    // For seccomp, this is a descending path relative to the kubelet's
    // configured seccomp profile location.
    // Must be set when type is "Localhost". Must NOT be set for other types.
    // +optional
    LocalhostProfile *string `json:"localhostProfile,omitempty" protobuf:"bytes,2,opt,name=localhostProfile"`
}

type SecurityProfileOCIBaseType string

const (
    SecurityProfileOCIBaseTypeRuntimeDefault SecurityProfileOCIBaseType = "RuntimeDefault"
    SecurityProfileOCIBaseTypeLocalhost      SecurityProfileOCIBaseType = "Localhost"
)

const (
    SeccompProfileTypeRuntimeDefault SeccompProfileType = "RuntimeDefault"
    SeccompProfileTypeLocalhost      SeccompProfileType = "Localhost"
    SeccompProfileTypeUnconfined     SeccompProfileType = "Unconfined"
    SeccompProfileTypeOCI            SeccompProfileType = "OCI"
)
```

`baseProfile` is part of alpha rather than deferred because it is the API
shape that resolved the trust-model discussion during review: the runtime
baseline alone gives node admins a floor, but per-workload-class baselines
(Story 3) require a pod-visible base that admission policy can mandate. The
three-way merge reuses the same intersection as the two-way case, so the
added implementation surface is one CRI message and its validation.

**AppArmor (beta)**: `AppArmorProfile` gains the same `OCI` type and `oci`
field in beta, once two open points are settled: the artifact content format
(the merge library operates on a structured profile, not on the AppArmor
policy language, see [OCI Artifact Format](#oci-artifact-format)) and the
runtime path for rendering, loading, and unloading merged profiles in the
kernel (see [Profile Merging](#profile-merging)). Shipping the field in
alpha without a runtime that can exercise it would leave it untested, so it
is deferred. The `SecurityProfileKind` CRI enum reserves the `AppArmor`
value now so that no CRI change is needed later. The kubelet's AppArmor
handler already fails closed on unknown profile types, so no prerequisite
fix analogous to the seccomp one is needed. AppArmor adds only
`AppArmorProfile.OCI *SecurityProfileOCIArtifact` and the
`AppArmorProfileTypeOCI` constant; `SecurityProfileOCIArtifact` and
`SecurityProfileOCIBase` are shared unchanged, with validation restricting
the AppArmor `baseProfile` to `RuntimeDefault` because a kernel-loaded
`Localhost` AppArmor profile has no retrievable text to merge.

API validation:
- `oci` must be set when `type` is `OCI`, and must not be set for other types.
- `oci.reference` must be a fully-qualified, digest-pinned OCI reference
  (`registry/repository@sha256:...`), parsed with `distribution/reference`.
  The parser bounds the repository name at 255 characters and the digest is
  fixed-length, so a valid reference is under 330 characters and no separate
  length limit is needed. Short names and tag references are rejected in
  alpha (see [Digest-only references](#notesconstraintscaveats)). The
  existing `image` field on containers does not enforce strict format
  validation; the new field can, because it has no backward compatibility
  constraints.
- `oci.baseProfile` is optional. When set, `baseProfile.type` must be
  `RuntimeDefault` or `Localhost`. When `baseProfile.type` is `Localhost`,
  `baseProfile.localhostProfile` must be set and passes the same
  descending-path validation as `seccompProfile.localhostProfile`. When
  `baseProfile.type` is `RuntimeDefault`, `baseProfile.localhostProfile` must
  not be set. `baseProfile.type` must not be `Unconfined` or `OCI` (no
  chaining OCI profiles, no unconfined base).
- Any privileged container whose effective seccomp profile (container-level,
  falling back to pod-level) is `OCI` is rejected at API validation (see
  [Privileged containers](#notesconstraintscaveats)).

When the `SecurityProfileOCIArtifact` feature gate is disabled, two mechanisms
apply:
- **Type validation**: API validation rejects the `OCI` type value, preventing
  creation of pods that use OCI profile references.
- **Field stripping**: The `oci` field is stripped from new objects following
  the standard drop-disabled-fields pattern. Existing objects that already
  have the field set (created while the gate was enabled) retain the value on
  update to prevent data loss.

No status fields are added in alpha. The digest in `oci.reference` identifies
the applied content exactly, so there is nothing to report that the spec does
not already say. When tag references are added in beta, a `ContainerStatus`
field reporting the resolved, digest-pinned reference accompanies them so
that users can audit which content a tag resolved to.

**Pod Security Standards**: The built-in PodSecurity admission controller
keeps rejecting `type: OCI` in `restricted` namespaces for all of alpha. Today
the `restricted` seccomp check (`seccompProfileRestricted_1_25`) only accepts
`RuntimeDefault` and `Localhost`, and it stays that way until the seccomp
fallthrough fix ([kubernetes/kubernetes#141958][k8s-141958]) is present in
the `.0` release of the oldest supported kubelet minor version, which is the
same v1.41 floor that gates beta (see
[Version Skew Strategy](#version-skew-strategy)). The reason is skew. The
PodSecurity policy library (`k8s.io/pod-security-admission`) is versioned
rather than feature gated: it is consumed outside kube-apiserver and does not
consult feature gates. And the API server cannot rely on a pre-1.38 kubelet
rejecting a profile type it does not know: before the fix, such a kubelet
silently fell through to `Unconfined` for any unrecognized
`SeccompProfileType`. A versioned check that admitted `OCI` in v1.38 would
therefore let a `restricted` namespace user obtain an unconfined container on
a skewed kubelet, which is exactly what `restricted` exists to prevent. Node
Declared Features keeps such pods off old kubelets when they go through the
scheduler, but not when they are bound directly via `nodeName` or gain an
ephemeral container after scheduling, so it is not sufficient on its own.

Once the floor is reached, the change lands as a new versioned check
(`seccompProfileRestricted_1_41`, numbered after the release that ships it)
that adds `OCI` to the accepted set. Namespaces that pin an older policy
version via `pod-security.kubernetes.io/<mode>-version` keep rejecting `OCI`,
which is the expected behavior of versioned checks. No gating is needed on the
PodSecurity side: when the `SecurityProfileOCIArtifact` feature gate is
disabled, API validation rejects the `OCI` type before PodSecurity evaluates
the pod. The `baseline` AppArmor check (`appArmorProfile_1_0`) receives the
same treatment when AppArmor support lands in beta.

The `baseline` seccomp check only forbids an explicit `Unconfined`, so it
admits `type: OCI` today without any change, and the same skew exposure
applies to it by omission: a pod bound via `nodeName` to a pre-1.38 kubelet
runs unconfined there. During alpha the feature gate is off by default, and
an administrator who enables it on a cluster that still runs pre-1.38
kubelets accepts that exposure for `baseline` namespaces. The recommendation
is to enable the alpha gate only on clusters whose kubelets are all at v1.38
or later. The v1.41 floor closes the gap for `baseline` without a code
change, because by then no supported kubelet can fall through.

The relaxation itself is sound once skew no longer undermines it. PSA allows
`Localhost` for `restricted` and `baseline` on the assumption that
the node admin controls which profiles are available on disk; it verifies
neither file presence nor content. The trust model for `OCI` is stronger: the
CRI runtime merges every pulled profile with the node's baseline via
intersection (see [Profile Merging](#profile-merging)), so the effective
profile cannot be more permissive than the baseline regardless of the OCI
profile's content. A `Localhost` profile can be more permissive than the
runtime default; an `OCI` profile that is effectively unconfined is
constrained to the baseline. Different nodes can use different baselines via
per-node runtime configuration, the same way different nodes can have
different files on disk. No coordination between PSA and the CRI runtime is
needed.

The CRI runtime's signature verification infrastructure (for example,
CRI-O's system-wide `/etc/containers/policy.json`) provides an additional
layer of control over profile artifact pulls. Cluster administrators can
also use admission webhooks (Kyverno, OPA/Gatekeeper) to restrict which
OCI references are allowed. The `oci.reference` field is part of the pod
spec, making it visible to all admission controllers. This allows policies
such as "only allow profiles from
`registry.internal.example.com/approved-profiles/`" or "require the
`agentic-baseline.json` base profile in the `agentic` namespace."

### CRI API Changes

Extend the CRI `SecurityProfile` message in `runtime/v1/api.proto` with a new
`OCI` profile type:

```protobuf
message SecurityProfile {
    enum ProfileType {
        RuntimeDefault = 0;
        Unconfined = 1;
        Localhost = 2;
        OCI = 3;
    }
    ProfileType profile_type = 1;

    // localhost_ref is the profile path on the node when profile_type is
    // Localhost.
    // For seccomp, it must be an absolute path to the seccomp profile.
    // For AppArmor, this field is the AppArmor profile name.
    string localhost_ref = 2;

    // oci_ref is the digest-pinned reference of a previously pulled
    // OCI security profile artifact (from PullSecurityProfileArtifact).
    // The runtime uses this to look up the cached profile content.
    // Must be set when profile_type is OCI.
    string oci_ref = 3;

    // base_profile specifies an optional base profile to merge with
    // the OCI profile via intersection. Only meaningful when profile_type
    // is OCI; runtimes must ignore this field for other profile types.
    // The runtime merges the OCI profile, the base_profile, and its own
    // configured baseline, producing an effective profile that permits an
    // operation only if all inputs permit it.
    // When unset, the runtime uses its configured default baseline only.
    SecurityProfileBase base_profile = 4;
}

// SecurityProfileBase specifies a base profile for layered OCI profile
// enforcement. Used in SecurityProfile.base_profile when the pod spec
// includes a baseProfile for the OCI artifact.
message SecurityProfileBase {
    enum BaseType {
        // BaseTypeUnspecified means the runtime uses its configured default
        // baseline. This is equivalent to not setting base_profile at all;
        // the kubelet never sends it and omits base_profile instead, so
        // runtimes only need to treat it as unset.
        BaseTypeUnspecified = 0;
        RuntimeDefault = 1;
        Localhost = 2;
    }
    BaseType type = 1;

    // localhost_ref is the profile path on the node when type is Localhost.
    // Same semantics as SecurityProfile.localhost_ref.
    string localhost_ref = 2;
}
```

Add a new `PullSecurityProfileArtifact` RPC to `ImageService`, alongside
`PullImage`. In containerd, snapshotters proxy `ImageService` for registry
credential handling. Placing profile pulls on a different service (such as
`RuntimeService`) would bypass that credential flow, causing pulls to fail
when a snapshotter is configured. Snapshotters that proxy `ImageService` will
need to forward the new RPC, but this is a pass-through addition: the
snapshotter forwards credentials without needing any profile-specific logic.
This is simpler than extending `PullImage`, which would require snapshotters
to handle profile-specific validation and response semantics inline.

Until such a proxy is updated, it answers the new method with
`Unimplemented`. Because `RuntimeService.Status` bypasses the proxy, the
runtime's `RuntimeFeatures` can report support while the pull path does not;
the kubelet then treats the `Unimplemented` response as permanent and the pod
fails with a clear reason (see [Kubelet Behavior](#kubelet-behavior)) rather
than hanging. Proxy implementations are encouraged to forward unknown
`ImageService` methods transparently so that new RPCs do not require a proxy
release. The containerd side, including the known proxies, is tracked in
[containerd#13546][containerd-13546].

A dedicated RPC is used instead of reusing `PullImage` because profile
artifacts have different semantics: they require media type validation (only
seccomp or AppArmor config types are accepted), content validation (valid
seccomp JSON), size enforcement (the 1 MiB default limit is far smaller than
container images), and single-layer verification. These constraints do not
apply to container images and would complicate `PullImage` if added there.
The request includes a `profile_kind` field so the runtime can validate the
artifact's config media type matches the expected security mechanism early,
before extracting content. See
[Extending PullImage with Media Type](#extending-pullimage-with-media-type) in
the Alternatives section for a detailed comparison.

Separating the pull from `RunPodSandbox`/`CreateContainer` gives the kubelet
control over retry timing, lets it fail before preparing DRA resources, and
avoids overloading the sandbox/container lifecycle calls with unrelated pull
logic.

If additional OCI artifact types beyond security profiles emerge in the future
(for example, configuration bundles or policy documents), the CRI API could
evolve toward a more general `PullArtifact` RPC that accepts a media type or
artifact kind discriminator. For alpha, a security-profile-specific RPC is
preferred because it encodes the validation semantics (size limits,
single-layer enforcement, profile kind matching) directly in the contract
rather than relying on callers to pass the right parameters to a generic
endpoint. Generalizing the RPC is a backward-compatible change that can happen
in a later CRI version if demand materializes.

```protobuf
service ImageService {
    // ...existing RPCs...

    // PullSecurityProfileArtifact pulls a security profile OCI artifact and
    // caches it locally. The returned digest-pinned reference is passed to
    // RunPodSandbox or CreateContainer via SecurityProfile.oci_ref.
    rpc PullSecurityProfileArtifact(PullSecurityProfileArtifactRequest)
        returns (PullSecurityProfileArtifactResponse) {}
}

message PullSecurityProfileArtifactRequest {
    // image is the OCI reference of the profile artifact, using the same
    // ImageSpec message as PullImage. image.image holds the digest-pinned
    // reference (e.g., "registry.example.com/profile@sha256:abc123..."),
    // image.user_specified_image holds the reference as written in the pod
    // spec, and image.runtime_handler selects the runtime handler. In
    // containerd, the runtime handler determines which snapshotter is used
    // for pulls, and snapshotters proxy ImageService to obtain registry
    // credentials, so carrying the handler here keeps profile pulls on the
    // same credential path as image pulls.
    ImageSpec image = 1;

    // auth contains registry authentication credentials, resolved by the
    // kubelet from imagePullSecrets, service account credentials, and
    // credential providers. Uses the same AuthConfig message as the PullImage
    // RPC.
    AuthConfig auth = 2;

    // sandbox_config holds the pod sandbox configuration, which is used to
    // pull the artifact in the context of the pod sandbox, matching PullImage.
    PodSandboxConfig sandbox_config = 3;

    // profile_kind identifies the expected security mechanism. Alpha uses
    // Seccomp only; AppArmor is reserved for beta. The runtime uses this to
    // validate that the pulled artifact's config media type matches the
    // expected kind and to reject mismatches early (e.g., an AppArmor
    // artifact referenced from a seccomp field).
    // SecurityProfileKindUnspecified is rejected with InvalidArgument.
    SecurityProfileKind profile_kind = 4;
}

enum SecurityProfileKind {
    SecurityProfileKindUnspecified = 0;
    Seccomp = 1;
    // AppArmor is reserved for beta; runtimes reject it with
    // InvalidArgument until AppArmor support is implemented.
    AppArmor = 2;
}

message PullSecurityProfileArtifactResponse {
    // resolved_ref is the digest-pinned reference that was pulled
    // (e.g., "registry.example.com/profile@sha256:abc123..."). Runtimes
    // must preserve the registry and repository from the request reference
    // even when a mirror served the content, and must accept this value in
    // SecurityProfile.oci_ref. In alpha the kubelet only sends digest-pinned
    // references, so this equals image.image and the kubelet passes the spec
    // reference to oci_ref. The field exists so that tag resolution can be
    // added in beta without changing the message; the kubelet then passes
    // resolved_ref unchanged to oci_ref.
    string resolved_ref = 1;

    // cached is true if the artifact was already present in the runtime's
    // storage and no registry request was made. The kubelet uses it to
    // label its pull duration metric.
    bool cached = 2;
}
```

`PullSecurityProfileArtifact` has pull-if-not-present semantics. If the
digest-pinned reference is already present in the runtime's storage, the
runtime returns it without contacting the registry. See the
[KEP-2535 note](#notesconstraintscaveats) for the credential verification
consequences of this choice.

Runtime support is reported through the existing `RuntimeFeatures` message
returned by the `Status` RPC, so that the kubelet can reject pods at
admission on nodes whose runtime lacks support (see
[Kubelet Behavior](#kubelet-behavior)):

```protobuf
message RuntimeFeatures {
    // ...existing fields...

    // security_profile_oci_artifact is set to true if the runtime supports
    // the PullSecurityProfileArtifact RPC and the OCI profile type in
    // SecurityProfile.
    bool security_profile_oci_artifact = 4;
}
```

The CRI runtime is responsible for pulling the artifact, caching it by digest,
validating its content, and merging it with the base profile and runtime
baseline when applying the profile in `RunPodSandbox` or `CreateContainer`.
The kubelet resolves image pull secrets and passes credentials via `AuthConfig`,
the same way it does for `PullImage`. See [Kubelet Behavior](#kubelet-behavior)
for the full pull-then-prepare sequencing.

### Kubelet Behavior

When the kubelet encounters a pod with an `OCI` type security profile:

1. **Admission**: Two kubelet admission checks apply. The Node Declared
   Features admission check rejects the pod if the kubelet's feature gate is
   disabled, because the node then does not declare
   `SecurityProfileOCIArtifact` (see
   [Version Skew Strategy](#version-skew-strategy)). A dedicated admission
   handler (the same mechanism as the existing AppArmor admit handler)
   rejects the pod with reason `SecurityProfileOCIArtifactUnsupported` if the
   CRI runtime does not report `security_profile_oci_artifact` in
   `RuntimeFeatures`. In both cases the pod transitions to `Failed` and is
   not retried on this node.
2. **Resolve credentials**: The kubelet resolves pull credentials using the
   same code path as container image pulls: `imagePullSecrets` on the pod
   spec, service account image pull secrets, and any configured credential
   provider plugins (`getPullSecretsForPod` in `pkg/kubelet/kubelet_pods.go`
   and the keyring logic in `pkg/credentialprovider`). Credentials are
   resolved on every `SyncPod`, so re-pulls and ephemeral container pulls use
   current credentials even if secrets were rotated after pod creation.
3. **Pull the profiles**: In `Kubelet.SyncPod`, after pull secrets are
   resolved and before the runtime manager's `SyncPod` is invoked, the
   kubelet calls `PullSecurityProfileArtifact` once for every unique OCI
   reference in the pod spec. Uniqueness is determined by the full
   `oci.reference` string, not by digest alone, because different references
   may resolve through different registries and require different
   credentials. This is before DRA resource preparation and
   sandbox creation, which both happen inside the runtime manager, so a pull
   failure does not require cleaning up already-prepared resources. The call
   happens on every `SyncPod`, not only before the sandbox exists; when the
   artifact is already present the runtime returns immediately without
   touching the registry, so the steady-state cost is one local RPC per
   reference per sync. Pulls are subject to the kubelet's image pull
   throttling (`serializeImagePulls` and `maxParallelImagePulls`) and to
   `runtimeRequestTimeout`, the standard CRI call timeout, which is
   appropriate for artifacts capped at 1 MiB. Sharing the throttle means a
   profile pull occupies an image pull slot; with the default serialized
   pulls this delays other pulls by the few milliseconds a cached call takes
   or by one small download. If alpha shows contention, profile pulls can
   move to a dedicated limit without an API change. The CRI runtime pulls the
   artifact (if not present), checks its media type, size, and layer count,
   and validates the profile content with the merge library's
   `ValidateArtifact`. No profile-vs-baseline comparison happens at pull
   time; the merge occurs later at apply time (see
   [Profile Merging](#profile-merging)).
4. **Pass to CRI**: The kubelet constructs the `SecurityProfile` message with
   `profile_type = OCI` and `oci_ref` set to the spec reference. If the pod
   spec includes a `baseProfile`, the kubelet populates the
   `SecurityProfile.base_profile` field with the corresponding type and
   localhost reference. For `RunPodSandbox`, the pod-level profile and base
   profile are included. For `CreateContainer`, the container-level profile
   (or the inherited pod-level profile) and base profile are included.
5. **CRI runtime merges and applies the profile**: The runtime looks up the
   cached OCI profile by digest, loads the base profile (from
   `base_profile` if specified, or its configured default baseline), and
   merges all inputs via intersection (see
   [Profile Merging](#profile-merging)). The effective profile permits an
   operation only if all input profiles permit it. The runtime applies the
   merged profile to the container. No pull occurs at this stage.
6. **Container restarts and kubelet restarts**: The spec reference is the
   only state, so nothing needs to be persisted or recovered. Because step 3
   runs on every `SyncPod`, a restart is preceded by the same
   `PullSecurityProfileArtifact` call, which returns immediately while the
   artifact is present; the runtime must keep artifacts referenced by an
   existing sandbox or container available. If the artifact is missing
   anyway (for example, after a manual cleanup of runtime storage), that
   call re-pulls it by digest before `CreateContainer` runs. Ephemeral
   containers are added through a later `SyncPod`, so the same step covers
   them.

Profile pulls happen before container image pulls, which occur inside the
runtime manager's `SyncPod`. Pulling the profiles of a pod concurrently with
each other and with image pulls, for example by integrating with the parallel
image pull feature (KEP-3876), is a potential optimization for future work.

The kubelet does not need to understand the profile content. This maintains the
existing separation of concerns where the kubelet orchestrates and the runtime
enforces. See [CRI API Changes](#cri-api-changes) for the rationale behind
using a dedicated RPC on `ImageService`.

Each container's security context is handled independently, so different
containers in the same pod can mix profile types (e.g., an init container using
`Localhost` and an app container using `OCI`). OCI profile references work at
both the pod-level `securityContext` (applying to all containers) and the
container-level `securityContext` (overriding the pod default). This includes
init containers, sidecar containers, and ephemeral containers, all of which
already support seccomp and AppArmor profiles. The kubelet resolves credentials
and calls `PullSecurityProfileArtifact` for each unique OCI reference.

`Localhost` profiles are not validated at admission time either; the kubelet
constructs the profile path and passes it to the CRI runtime, which fails at
sandbox or container creation if the file is missing. `OCI` profiles follow
the same pattern: pull failures surface during pod sync, before sandbox
creation. The kubelet treats these the same as container image pull failures:
the pod remains in `Pending` and the kubelet retries with exponential backoff
(up to a maximum interval). The kubelet emits `SecurityProfilePulled` events
on successful pulls and `SecurityProfilePullFailed` events on failures.

If the CRI runtime permanently rejects a profile (for example, an invalid
media type, corrupt content, or an oversized artifact), the kubelet marks the
pod as `Failed` with reason `SecurityProfileRejected` rather than retrying
indefinitely. The precedent is the kubelet's handling of
`VolumeAttachmentLimitExceeded`: `Kubelet.SyncPod` calls `rejectPod` when
volume setup fails deterministically, before the runtime is involved. Profile
pulls sit at the same point in `SyncPod`, so the same mechanism applies.
Terminal and transient failures are distinguished the same way the kubelet
already classifies image pull errors: by the well-known error prefixes
defined in `k8s.io/cri-api/pkg/errors`. `RegistryUnavailable` and
`SignatureValidationFailed` are reused with their existing kubelet handling
(both are transient and retried with backoff; a signature can be attached to
an already published digest, and the signature policy can change), and a new
`SecurityProfileInvalid` well-known error marks permanent
rejection (invalid media type, oversized artifact, wrong layer count, or
content that fails validation). Every other error, including authentication
and permission failures (credentials may be rotated or updated on the
service account) and gRPC `Unavailable` or `DeadlineExceeded`, is transient
and retried with backoff. `Unimplemented` should not occur after admission
because the admission handler already checks `RuntimeFeatures`; if it does
occur (a runtime downgraded underneath a running kubelet), it is treated as
permanent. This prevents pods with bad profile references from being stuck
in an infinite retry loop while keeping recoverable failures retryable.

Because credential resolution reuses the pod's `imagePullSecrets`, the registry
hosting the profile artifacts must be covered by the same pull secrets used for
container images. If profiles are stored in a different registry than the pod's
images, that registry's credentials must be added to the pod's
`imagePullSecrets`. This is the same model used for image volumes.

### Profile Merging

To ensure that OCI-pulled profiles cannot weaken node security, the CRI
runtime merges every OCI profile with the node's baseline via intersection.
The effective profile applied to a container permits an operation only if
all input profiles permit it. This is the core security guarantee of OCI
profile distribution: even if a profile is tampered with in transit or a
registry is compromised, the effective profile is always at least as
restrictive as the baseline.

**Three-way merge**: The CRI runtime merges up to three profile inputs:
1. The runtime's configured baseline (RuntimeDefault or an admin-supplied
   profile on disk, configured via the runtime's own configuration such as
   CRI-O's `crio.conf` or containerd's `config.toml`)
2. The pod-spec base profile (optional, specified via
   `oci.baseProfile` in the pod spec; can be `Localhost` or
   `RuntimeDefault`)
3. The OCI profile pulled from the registry

The merge is an intersection: for each security-relevant dimension, the
effective profile uses the most restrictive setting from any input. The
result is at least as restrictive as every individual input.

When the pod spec omits `baseProfile`, the merge is two-way (runtime
baseline and OCI profile). When `baseProfile` is specified, the merge is
three-way. The inputs are merged in a fixed order: runtime baseline, then
pod-spec base profile, then OCI profile. The resulting actions do not depend
on that order (intersection is commutative and associative), but tie-breaks
for values that do not affect restrictiveness, such as `errnoRet` and the
listener settings, resolve in favor of the earlier input, so the runtime
baseline wins.

**Why merge instead of subset-and-reject**: With subset validation, the
runtime rejects any OCI profile that is more permissive than the baseline
in any dimension. This creates an operational problem: when an admin adds
a new restriction to the baseline (blocking a newly discovered dangerous
syscall), all existing OCI profiles that did not already block that syscall
suddenly fail validation and pods break. The admin must update every OCI
profile in the registry before tightening the baseline.

With merge semantics, the admin simply tightens the baseline and the merge
automatically applies the new restriction to all OCI profiles. Existing OCI
profiles continue to work because the merge constrains them. If an OCI
profile permits an operation the baseline does not, that operation is
silently blocked in the effective profile. No OCI profiles need to be
republished.

**Merge semantics per profile type**:

- **Seccomp**: For each syscall, the merge takes the more restrictive action
  from all input profiles. Action restrictiveness is ordered:
  `SCMP_ACT_KILL_PROCESS` > `SCMP_ACT_KILL_THREAD` > `SCMP_ACT_TRAP` >
  `SCMP_ACT_ERRNO` > `SCMP_ACT_NOTIFY` > `SCMP_ACT_TRACE` >
  `SCMP_ACT_LOG` > `SCMP_ACT_ALLOW`. `SCMP_ACT_NOTIFY` ranks above
  `SCMP_ACT_TRACE` because it blocks the syscall until a supervisor decides,
  whereas `SCMP_ACT_TRACE` traps to a ptrace tracer that may let it proceed.
  Both fail closed without a supervisor: `SCMP_ACT_TRACE` returns `ENOSYS`
  when no tracer is attached, and runc refuses to start a container that
  uses `SCMP_ACT_NOTIFY` without a listener. Because OCI profiles cannot
  supply a listener, `SCMP_ACT_NOTIFY` is rejected in OCI profiles at pull
  validation; a runtime baseline that uses it keeps its own listener. The
  top-level `defaultAction` is merged by the same restrictiveness ordering as
  per-syscall actions. When argument filters are present and the intersection
  cannot be expressed in the OCI format, the merge is conservative and falls
  back to the more restrictive surrounding action. For example, if one input
  allows `write` only when argument 0 is greater than 2 and another only when
  argument 0 is less than 100, the exact intersection (a range) is not
  expressible because several conditions on the same argument index within
  one entry are alternatives, so `write` is denied. Filters on different
  argument indices are conjoined and identical filters are kept.
  Architecture lists are intersected: an empty list means "unspecified" and
  defers to the other inputs, and an architecture present in only some
  non-empty inputs is dropped from the effective filter, so syscalls made
  through it hit libseccomp's bad-architecture action and kill the process.
  Two non-empty lists with no architecture in common yield an empty list,
  which the runtime-spec reads as "native architecture only". Runtimes
  populate the native architecture on every input before merging (the
  library's `PopulateNativeArchitecture`), which makes the intersection
  exact. Argument conditions are compared as runtimes evaluate them:
  `valueTwo` is only read for `SCMP_CMP_MASKED_EQ` and is ignored for every
  other operator. When multiple
  inputs specify the same action for a syscall (for example, both use
  `SCMP_ACT_ERRNO`), the `errnoRet` value comes from the earliest input in
  merge order: the runtime baseline, then the pod-spec base profile, then
  the OCI profile. `defaultErrnoRet` follows the same rule. Top-level
  `flags` are merged by what they do, so that the effective profile never
  loosens the baseline: `SECCOMP_FILTER_FLAG_SPEC_ALLOW`, which disables the
  Speculative Store Bypass mitigation, is present only if every input sets
  it, while `SECCOMP_FILTER_FLAG_LOG`, which adds audit logging, is present
  if any input sets it. An empty list means "no flags". `listenerPath`,
  `listenerMetadata` and `SECCOMP_FILTER_FLAG_WAIT_KILLABLE_RECV` belong to
  the node-local listener and are taken from the runtime baseline only; OCI
  profiles that set them are rejected at pull validation, as are `errnoRet`
  values above the kernel's maximum errno.
- **AppArmor (beta)**: The merge operates on the structured AppArmor profile
  format described in [OCI Artifact Format](#oci-artifact-format), not on
  the AppArmor policy language: capabilities, filesystem paths per access
  mode, and network rules are intersected, with `nil` meaning "unspecified"
  and an empty set meaning "nothing permitted". Rule classes the structured
  format cannot express (for example `signal`, `ptrace`, or `mount`) always
  come from the runtime's default profile template. Unlike seccomp, a merged
  AppArmor profile must be rendered to policy language and loaded into the
  kernel (`apparmor_parser --replace`) under a runtime-generated name before
  a container can use it, and unloaded when unreferenced. This path, and the
  `AppArmorProfile.oci` API field, land in beta.
- **Landlock**: Out of scope until the OCI runtime spec defines the format
  (see [Non-Goals](#non-goals)); the library interface accommodates it.

**Profile merge library**: Profile merging requires non-trivial
profile parsing and merge logic, especially for seccomp profiles with
architecture-specific syscall tables and argument filters. The
[security-profiles-merger][spm] library provides this functionality as a
standalone Go module (`sigs.k8s.io/security-profiles-merger`), used by CRI
runtimes to merge profiles. The library exposes `Intersect`,
`Union`, `Validate`, and `Diff` APIs per profile type; CRI runtimes only use
`Intersect` (SPO uses `Union` to combine recorded profiles). Seccomp profiles
are represented as `specs.LinuxSeccomp` from the OCI runtime specification;
AppArmor profiles use the library's structured `apparmor.Profile` type (the
same abstract format used by the Security Profiles Operator). The library
does not parse the AppArmor policy language. It is designed to be extensible
to new profile types (such as landlock) without changes to the CRI protocol
or the kubelet. The module has no dependencies beyond the
[OCI runtime specification][oci-runtime-spec] (`runtime-spec v1.3.0`), so
both CRI-O and containerd can import it without circular dependencies. The
module started in the author's personal namespace; moving it to a
community-owned organization with SIG Node owners before any runtime
integrates it is an alpha criterion, so that runtimes never take a
dependency on a single-maintainer repository. That move completed on
2026-09-10: the repository is a SIG Node subproject under
[kubernetes-sigs][spm], and the module path changed accordingly.

[spm]: https://github.com/kubernetes-sigs/security-profiles-merger
[oci-runtime-spec]: https://github.com/opencontainers/runtime-spec
[crio-10037]: https://github.com/cri-o/cri-o/issues/10037
[containerd-13546]: https://github.com/containerd/containerd/issues/13546
[spo-3421]: https://github.com/kubernetes-sigs/security-profiles-operator/pull/3421

**Merge at apply time**: Profile merging happens at container or sandbox
creation time (`RunPodSandbox`/`CreateContainer`), not at pull time
(`PullSecurityProfileArtifact`). The pull RPC fetches and caches the raw OCI
artifact with content validation (valid seccomp JSON, correct media type,
size limits, single-layer enforcement). The merge happens when the runtime
applies the profile, using the cached OCI artifact, the base profile from the
CRI message, and the runtime's configured baseline. This allows the same
cached OCI artifact to be merged with different base profiles for different
pods, and allows baseline changes to take effect without re-pulling
artifacts. Because the pull path runs the merge library's validation on
every artifact and runtimes validate their configured baseline when loading
configuration, the merge at apply time cannot fail on profile content. A
failure there indicates a runtime bug and surfaces through the existing
sandbox or container creation error path (`FailedCreatePodSandBox` or
`Failed` events, retry with backoff).

**Per-node control**: Each node's CRI runtime has its own baseline profile
configuration. Different nodes can use different baselines, the same way
different nodes can have different `Localhost` profile files on disk.
The runtime's configured baseline is the absolute floor; no pod spec can
weaken it. The pod-spec `baseProfile` adds per-workload-class restrictions
on top of the runtime baseline.

#### Observability

When the merge constrains the OCI profile (the effective
profile is more restrictive than the OCI profile alone), the CRI runtime
logs the constrained dimensions at the `warning` level. The log entry
includes the container name, the OCI profile reference, and a summary of
which operations were constrained (e.g., "syscall X denied by runtime
baseline", "capability Y removed by base profile"). This gives
administrators a clear signal when pods experience unexpected EPERM errors
due to merge constraints. Conservative argument filter denials (where the
intersection cannot be computed precisely) are also logged. CRI runtimes
should use structured logging with a well-known message identifier so that
log aggregation tools can surface these events.

These logs are the only signal in alpha. The CRI has no channel through
which a runtime can attach warnings to a `RunPodSandbox` or
`CreateContainer` response, so the kubelet cannot emit a pod event for merge
constraints without a CRI change. CRI runtimes should additionally expose a
counter of constrained merges on their own metrics endpoint. Beta evaluates
surfacing constraint information through the CRI container status so that
the kubelet can emit a `SecurityProfileMergeConstrained` event and make the
condition visible in `kubectl describe pod`.

**Relationship to CRI runtime policy**: Profile merging and the CRI
runtime's signature verification (such as CRI-O's
`/etc/containers/policy.json`) are complementary. Profile merging
controls what a profile is allowed to do. Signature verification controls
who published the profile. Both are configured by the node admin.

### OCI Artifact Format

Security profiles are stored as OCI artifacts following the
[OCI Image Specification](https://github.com/opencontainers/image-spec/blob/main/manifest.md).

Each profile type uses a distinct config media type for identification:

| Profile Type | Config Media Type |
|-------------|-------------------|
| Seccomp | `application/vnd.cncf.seccomp-profile.config.v1+json` |
| AppArmor (beta) | `application/vnd.cncf.apparmor-profile.config.v1+json` |

This KEP documents these media types but does not define them. Both use the
`vnd.cncf.` vendor prefix, registered under the vendor tree as defined in
[RFC 6838, Section 3.2][rfc6838]. OCI does not standardize artifact media
types; the `vnd.cncf.` names are owned by the publishing projects (SPO and
CRI-O), and their registration under the IANA vendor tree is a beta
criterion so that the strings are fixed before the feature is on by
default. The list of accepted config media types is part of the CRI contract
documented in `api.proto`. SPO publishes seccomp artifacts with the typed
config media type since [security-profiles-operator#3421][spo-3421] (merged
2026-09-10); before that it used the generic
`application/vnd.unknown.config.v1+json`, and CRI-O's annotation-based
seccomp artifact support does not validate the config media type at all.
CRI-O accepts the typed media type as part of [cri-o#10037][crio-10037], and
the e2e tests for this feature use artifacts published with it (see
[Infrastructure Needed](#infrastructure-needed-optional)). Artifacts
published with the generic config media type are rejected by
`PullSecurityProfileArtifact` and must be republished; this is a one-time
migration for early adopters of the annotation-based mechanism.

Runtimes identify the profile kind from the manifest's config media type. If
the manifest uses the OCI 1.1 empty config descriptor
(`application/vnd.oci.empty.v1+json`), the manifest's `artifactType` field
is used instead and must carry the same value.

[rfc6838]: https://datatracker.ietf.org/doc/html/rfc6838#section-3.2

Profile content is stored as a single layer in the artifact. The artifact must
contain exactly one layer; runtimes must reject artifacts with zero layers or
more than one layer to ensure consistent behavior across CRI implementations.
The layer blob is the raw profile document, not a tar archive. Runtimes
identify the profile kind from the config media type (or `artifactType`) and
must not require a particular layer media type: tooling such as ORAS labels
file layers `application/vnd.oci.image.layer.v1.tar` by default even though
the blob is raw JSON.

The CRI runtime knows which security mechanism to apply from the CRI field
context (the `seccomp` or `apparmor` field on `LinuxContainerSecurityContext`).
The config media type serves as a validation check: if the media type does not
match the expected type for the CRI field (e.g., a seccomp artifact referenced
in the `apparmor` field), the runtime must reject the profile. The expected
content formats are:

- **Seccomp**: The layer contains a JSON seccomp profile as defined by the
  [OCI runtime spec][oci-seccomp] (`linux.seccomp`).
- **AppArmor (beta)**: The layer contains a JSON document in the structured
  AppArmor profile format of the [security-profiles-merger][spm] library,
  which is the abstract format that SPO's `AppArmorProfile` API uses:
  `executable`, `filesystem`, `network`, and `capabilities` rule sets. Raw
  AppArmor policy language is not accepted because it cannot be merged with
  the baseline (see [Profile Merging](#profile-merging)). The format is
  expressive enough for beta if it can represent the default AppArmor
  profiles shipped by containerd and CRI-O and the example profiles in the
  SPO repository. If it cannot, AppArmor support stays deferred rather than
  shipping a merge that silently drops rule classes.

[oci-seccomp]: https://github.com/opencontainers/runtime-spec/blob/main/config-linux.md#seccomp

### Profile Verification

Signature verification for security profile artifacts is handled by the CRI
runtime using its existing image signature verification infrastructure, the
same approach used for [image volumes][image-volumes]. CRI-O, for example,
verifies signatures using the system-wide signature policy
(`/etc/containers/policy.json`) with optional namespace-specific overrides via
`SignaturePolicyDir`. This provides consistent trust policy across container
images, image volumes, and security profile artifacts without introducing a new
verification mechanism. If verification fails, the CRI call returns an error
and the kubelet surfaces it as a pod event.

[image-volumes]: https://kubernetes.io/docs/concepts/storage/volumes/#image

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

Existing seccomp e2e tests provide a baseline. No prerequisite updates are
required.

##### Unit tests

- `pkg/kubelet/kuberuntime`: seccomp's `fieldSeccompProfile` rejects unknown
  profile types with an explicit error (prerequisite fix, not gated by this
  feature); CRI `SecurityProfile` message construction with OCI references
  and base profiles
- `pkg/apis/core/validation`: the new `OCI` profile type, `oci` field,
  digest-only reference rule, `baseProfile` rules, and the privileged
  container rule
- `pkg/api/pod`: drop-disabled-fields behavior for `oci` when the feature
  gate is off
- `k8s.io/pod-security-admission/policy` (beta): the new versioned
  `restricted` seccomp check accepts `OCI`, and older check versions keep
  rejecting it
- `pkg/kubelet`: credential resolution for artifact pulls, the admission
  handler for runtime feature support, and terminal versus transient error
  classification
- `k8s.io/component-helpers/nodedeclaredfeatures/features/securityprofileociartifact`:
  feature discovery from the kubelet's gate, `MaxVersion`, and inference from
  pods with `OCI` profiles
- Profile merge library ([security-profiles-merger][spm]): seccomp merge
  across architectures, default actions, per-syscall rules, and argument
  filters (including conservative denial when intersection is unprovable)

##### Integration tests

- API server integration tests (`test/integration`) for feature gate
  enablement and disablement: `OCI` is rejected with the gate off, accepted
  with the gate on, and existing objects retain the field across a gate flip
- API validation rejects tag references and short names in `oci.reference`
- PodSecurity integration tests: `restricted` keeps rejecting `OCI` during
  alpha; the new check version that accepts it is covered when it lands in
  beta
- Scheduler integration tests verifying that pods with `OCI` profiles are only
  placed on nodes declaring the feature

##### CRI conformance tests (critest)

- Validate that `RuntimeFeatures` reports `security_profile_oci_artifact`
- Validate that `PullSecurityProfileArtifact` correctly pulls and caches OCI
  artifact profiles, returning the resolved reference
- Validate that the CRI runtime accepts the `OCI` profile type in
  `SecurityProfile` messages for seccomp, referencing a previously pulled
  digest
- Verify that the runtime rejects artifacts with invalid media types
- Verify that the runtime enforces the size limit and rejects oversized
  artifacts
- Verify that the runtime rejects artifacts with multiple layers
- Verify that the runtime rejects OCI profiles that use `SCMP_ACT_NOTIFY`,
  set listener fields, or exceed the per-syscall entry cap
- Verify that a second pull of the same reference reports `cached`
- Verify that the `profile_kind` field triggers rejection when the artifact's
  config media type does not match the expected security mechanism
- Verify that the runtime correctly merges an OCI profile with the runtime
  baseline via intersection, producing an effective profile that is at least
  as restrictive as the baseline
- Verify that the runtime correctly merges an OCI profile with a pod-spec
  base profile (Localhost) and the runtime baseline (three-way merge)
- Verify that the runtime returns the well-known `RegistryUnavailable` error
  for unreachable registries, `SecurityProfileInvalid` for rejected
  artifacts, and a retryable error for invalid credentials

##### e2e tests

The kubelet-to-runtime flow is covered by `e2e_node` tests labeled
`[Feature:SecurityProfileOCIArtifact]` and
`[NodeFeature:SecurityProfileOCIArtifact]`, running in a dedicated CRI-O job
(the first runtime to implement the RPC). Test artifacts are published to
`registry.k8s.io` through the image promoter, alongside the existing e2e test
images. The tests cover:

- Pull a seccomp profile from an OCI registry and apply it to a container
- Verify that a static pod with an `OCI` profile pulls using node-level
  credential providers
- Verify that pull failures result in appropriate pod events
- Verify caching behavior (second pod using the same profile starts without
  re-pulling)
- Verify behavior with invalid or oversized artifacts
- Verify that an OCI profile more permissive than the baseline is silently
  constrained by the merge (effective profile equals the baseline)
- Verify layered profiles: OCI profile merged with a Localhost base profile
  produces the correct intersection

### Graduation Criteria

#### Alpha

- Fix seccomp's `fieldSeccompProfile` to reject unknown profile types with an
  explicit error instead of silently falling through to `Unconfined`
  (prerequisite, not gated by this feature; merged for v1.38 as
  [kubernetes/kubernetes#141958][k8s-141958])
- Feature implemented behind `SecurityProfileOCIArtifact` feature gate
- CRI API extended with `OCI` profile type, `PullSecurityProfileArtifact`
  RPC, the `security_profile_oci_artifact` runtime feature, and the
  `SecurityProfileInvalid` well-known error
- Kubelet calls `PullSecurityProfileArtifact` to pull profiles before DRA
  preparation and passes the references to `RunPodSandbox`/`CreateContainer`
- Kubelet admission handler rejects pods with `OCI` profiles when the runtime
  does not report `security_profile_oci_artifact`
- Node Declared Features integration: the kubelet declares the feature when
  its gate is enabled, the scheduler infers the requirement from pods with
  `OCI` profiles, and the framework's kubelet admission check rejects such
  pods when the gate is off
- Profile merge library ([security-profiles-merger][spm]) implemented with
  seccomp merge support, including architecture-specific syscall handling
  and argument filter intersection, artifact validation
  (`ValidateArtifact`), and worst-case merge benchmarks
- The [security-profiles-merger][spm] module moved to a community-owned
  organization with SIG Node owners before any runtime integrates it (done
  2026-09-10, `kubernetes-sigs/security-profiles-merger`)
- At least one CRI runtime (CRI-O, tracked in [cri-o#10037][crio-10037])
  implements the pull, merge, and apply flow for seccomp using the merge
  library
- Initial e2e tests for seccomp OCI artifacts, including profile merge tests
  and layered profile tests

#### Beta

Required before the feature gate is enabled by default:

- Production support in at least one of CRI-O and containerd, and a release
  candidate available in the other.
- Seccomp fallthrough fix (reject unknown profile types) present in the `.0`
  release of the oldest supported kubelet minor version. This is a hard
  prerequisite for enabling the feature gate by default (see
  [Version Skew Strategy](#version-skew-strategy)). The fix shipped in
  `1.38.0`, so the earliest release that can satisfy this is v1.41.
- PodSecurity `restricted` check updated to accept the `OCI` profile type as
  a new versioned check. This depends on the fallthrough prerequisite above,
  because the API server cannot rely on an older kubelet rejecting an
  unknown profile type (see
  [Kubernetes API Changes](#kubernetes-api-changes)).
- `ListSecurityProfileArtifacts` and `RemoveSecurityProfileArtifact` RPCs
  added and kubelet garbage collection of profile artifacts implemented, so
  that unreferenced artifacts do not accumulate on nodes.
- KEP-2535 credential verification extended to profile artifacts (see
  [Notes/Constraints/Caveats](#notesconstraintscaveats)).
- Config media types registered under the IANA vendor tree by the
  publishing projects (see [OCI Artifact Format](#oci-artifact-format)).
- Feedback from early adopters gathered and acted on.

Planned during beta, each gated on its own readiness:

- Tag references: API validation accepts tags, the kubelet resolves each
  tag once per pod sandbox and persists the resolved reference in the
  sandbox metadata, and a `ContainerStatus` field reports the resolved
  reference (see [Digest-only references](#notesconstraintscaveats)).
- AppArmor: `AppArmorProfile.oci` API field, structured artifact format
  settled against the criteria in
  [OCI Artifact Format](#oci-artifact-format), PodSecurity `baseline`
  AppArmor check updated, and pull, merge, render, load, and unload
  implemented and tested in at least one runtime.
- Profile merge library adopted by both CRI-O and containerd.
- Merge constraints surfaced through the CRI so that the kubelet can emit a
  pod event (see [Observability](#observability)).

#### GA

- At least two releases of beta usage
- Production support in both CRI-O and containerd
- Conformance tests in place
- Documentation published on kubernetes.io

#### Deprecation

Not applicable. This KEP introduces a new `OCI` security profile type and does
not deprecate any existing functionality. The `RuntimeDefault`, `Localhost`, and
`Unconfined` profile types remain fully supported.

### Upgrade / Downgrade Strategy

The new `OCI` profile type is additive. Existing `RuntimeDefault`, `Localhost`,
and `Unconfined` profiles continue to work unchanged.

On **upgrade** with the feature gate enabled, pods can start using `OCI`
profile references. No migration is required for existing workloads.

On **downgrade** or feature gate disablement, new pods with `OCI` profile
references fail validation at the API server. Existing pods already running
with OCI profiles continue running (profiles are applied at container
creation, not enforced continuously by the kubelet) until the kubelet
restarts: the kubelet re-runs pod admission on restart, and the Node Declared
Features admission check fails pods whose required feature the node no longer
declares, so a kubelet restarted with the gate disabled marks running OCI
pods `Failed`. Operators disabling the gate should expect those pods to be
replaced. Before such a restart, container restarts within those pods behave
differently depending on the kubelet:

- A kubelet from the release that ships this feature or later, with the
  feature gate disabled, rejects the disabled `OCI` type with an explicit
  error, so the container fails to restart (fail-closed).
- A kubelet downgraded to a release **without** the fallthrough fix (which
  also predates the admission check) restarts containers with an `OCI`
  seccomp profile as `Unconfined`, because `fieldSeccompProfile` silently
  falls through for unknown types. Operators downgrading the kubelet across
  that boundary should delete pods that use `OCI` profiles first.

### Version Skew Strategy

This feature involves coordination between the API server and the kubelet.

- **New API server, old kubelet**: The API server accepts `OCI` profile types.
  An older kubelet that does not understand the `OCI` type has a code path
  in `fieldSeccompProfile` that silently falls through to `Unconfined` for
  unrecognized types. This means an old kubelet would run OCI-referenced
  containers without any seccomp profile applied, which is a security gap. A
  prerequisite fix to the seccomp code path is required before this feature
  ships: the kubelet's seccomp handler must be updated to reject unknown
  profile types with an explicit error rather than falling through to
  `Unconfined`. This fix merged as [kubernetes/kubernetes#141958][k8s-141958]
  and ships in v1.38, the same release as the alpha feature gate.
  **Beta promotion (feature gate on by default) requires that
  the seccomp fallthrough fix is present in the `.0` release of the oldest
  supported kubelet minor version.** Since Kubernetes does not require
  kubelet upgrades before API server upgrades, a cluster could be running
  kubelet `1.y-3.0` (the very first patch of the oldest supported minor
  version). A backport released in a later patch (e.g., `1.y-3.9`) does not
  help because that kubelet was never required to be updated. Practically,
  if the fix lands in `1.X.0`, beta can happen earliest when `1.X` is the
  oldest supported kubelet version (API server at `1.X+3`). Since the fix
  landed in `1.38.0`, beta is therefore possible no earlier than v1.41. This
  is a hard prerequisite, not a best-effort backport. In addition, the feature
  integrates with [Node Declared Features (KEP-5328)][ndf] from alpha (see
  below), which keeps the scheduler from placing pods with OCI profiles on
  nodes with old kubelets. The two mechanisms are complementary: declared
  features cover every pod that goes through the scheduler, and the `.0`
  requirement covers the two paths that do not: pods bound directly via
  `nodeName`, and ephemeral containers with an `OCI` profile added through
  the `ephemeralcontainers` subresource to a pod already running on an old
  kubelet, which the `NodeDeclaredFeatureValidator` admission plugin does
  not validate. If a pod lands on an unpatched old kubelet, the seccomp risk
  is that the container runs unconfined rather than failing. For the same
  reason the PodSecurity `restricted` check does not accept `OCI` before that
  floor (see [Kubernetes API Changes](#kubernetes-api-changes)). During
  alpha the gate is off by default, and an operator who enables it on a
  cluster that still runs pre-1.38 kubelets accepts that a pod bound
  directly to one of them may run unconfined; the recommendation is to
  enable the alpha gate only once every kubelet is at v1.38 or later.
- **Old API server, new kubelet**: The API server rejects `OCI` profile types
  at validation. Pods using this feature cannot be created. This is safe.
- **CRI version skew**: A CRI runtime that does not support the RPC does not
  report `security_profile_oci_artifact` in `RuntimeFeatures`, so the
  kubelet's admission handler rejects pods that use it (see
  [Kubelet Behavior](#kubelet-behavior)). If a runtime nevertheless returns
  `Unimplemented` (for example, a runtime that reported the feature but was
  downgraded underneath a running kubelet), the kubelet treats this as a
  terminal failure and marks the pod as `Failed`. Both paths fail earlier and
  more clearly than if the `OCI` profile type were rejected inside
  `RunPodSandbox`/`CreateContainer`.

[Node Declared Features (KEP-5328)][ndf] is GA since v1.37 and this feature
uses it from alpha. The kubelet declares `SecurityProfileOCIArtifact` in
`node.status.declaredFeatures` when its feature gate is enabled. Declared
features must be derivable from feature gates and static kubelet
configuration alone, so that autoscalers can predict them for nodes that do
not exist yet; CRI runtime support is therefore deliberately not part of the
declaration. The feature sets `MaxVersion` to the GA release plus the
supported kubelet skew, as the framework requires. The scheduler infers the
requirement from any `seccompProfile.type: OCI` in the pod spec and filters
out nodes that do not declare the feature; a pod with no matching node stays
`Pending` with an event naming the missing feature. The framework's kubelet
admission check enforces the same rule for pods that bypass the scheduler,
but only on kubelets that know this feature (v1.38 or later with the gate
disabled): the check infers a pod's requirements through the shared feature
library, and a kubelet whose library predates the feature infers nothing and
admits the pod. Old kubelets are therefore never selected by the scheduler,
but a pod created with `spec.nodeName` pointing at one is caught by nothing:
the API server's `NodeDeclaredFeatureValidator` validates pod updates only
(the main spec and the `resize` subresource), not creates. Nor does it
validate the `ephemeralcontainers` subresource, so an ephemeral container
with an `OCI` profile added to a pod already running on an old kubelet is
not caught either. That is why the fallthrough fix, not Node Declared
Features, is what makes those paths safe (see
[Version Skew Strategy](#version-skew-strategy)). Runtime
support is checked separately at kubelet admission via `RuntimeFeatures` (see
[Kubelet Behavior](#kubelet-behavior)): a node whose kubelet has the gate
enabled but whose runtime lacks support is schedulable and fails such pods at
admission, the same as the existing AppArmor admit handler on nodes without
AppArmor.

Scheduler awareness of a node's baseline profile configuration is another
potential optimization: the scheduler could avoid placing pods on nodes whose
baseline profile is incompatible with the referenced OCI profile. This could
be expressed via node declared features or node labels derived from the
runtime configuration. This is out of scope for alpha but is a natural
extension of the node declared features integration.

[ndf]: https://github.com/kubernetes/enhancements/issues/5328
[k8s-141958]: https://github.com/kubernetes/kubernetes/pull/141958

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `SecurityProfileOCIArtifact`
  - Components depending on the feature gate: kubelet, kube-apiserver

###### Does enabling the feature change any default behavior?

No. The feature adds a new profile type (`OCI`). Existing profile types and
their behavior are unchanged. Pods that do not use OCI profile references
are unaffected. The CRI runtime uses RuntimeDefault as the implicit baseline
for profile merging. Node admins can optionally configure a different
baseline via the runtime's configuration. The only visible change on nodes
that do not run OCI-profile pods is that kubelets with the gate enabled add
`SecurityProfileOCIArtifact` to `node.status.declaredFeatures`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate causes the API server to reject new pods with
`OCI` profile references. Already-running pods continue running until the
kubelet restarts with the gate disabled; at that point the kubelet's
admission re-check (Node Declared Features) marks them `Failed`, because the
node no longer declares the feature. Before such a restart, container
restarts in those pods fail because the kubelet rejects the disabled `OCI`
profile type with a clear error event (fail-closed). A kubelet that predates
the feature fails closed only if it includes the prerequisite fix to
seccomp's `fieldSeccompProfile` fallthrough (see
[Version Skew Strategy](#version-skew-strategy)).

###### What happens if we reenable the feature if it was previously rolled back?

Pods with `OCI` profile references in their spec (created while the feature was
enabled, still present in etcd) will work again on their next container
creation. No data migration is needed.

###### Are there any tests for feature enablement/disablement?

Unit tests verify that API validation rejects or accepts `OCI` profile types
based on the feature gate state and that disabled fields are dropped from new
objects but retained on existing ones. API server integration tests exercise
the gate flip sequence (enabled, disabled, re-enabled) against stored pods.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout cannot impact already running workloads because the feature is
opt-in per pod. A rollback (disabling the feature gate) prevents new pods with
OCI profiles from being created and causes container restarts within existing
OCI-profile pods to fail closed on the kubelet. Running containers are not
affected until the kubelet restarts with the gate disabled, at which point
its admission re-check marks the OCI-profile pods `Failed` and their
controllers replace them. See
[Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy) for the kubelet
downgrade case.

###### What specific metrics should inform a rollback?

- `kubelet_security_profile_artifact_pull_errors_total` increasing, indicating
  registry connectivity or authentication problems.
- `kubelet_security_profile_artifact_pull_duration_seconds` showing high
  latency, indicating registry performance issues affecting pod startup.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TBD for beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- Metric: `kubelet_security_profile_artifact_pull_duration_seconds` (histogram)
  is emitted by the kubelet whenever it calls `PullSecurityProfileArtifact`.
  A non-zero count indicates active use.
- API: Pods with `seccompProfile.type: OCI` can be queried directly.
- Node status: nodes whose kubelet supports the feature list
  `SecurityProfileOCIArtifact` in `status.declaredFeatures`.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: `SecurityProfilePulled` (successful pull),
    `SecurityProfilePullFailed` (failed pull)
- [x] Other (treat as last resort)
  - Details: The digest in `oci.reference` identifies the applied content
    exactly. Pull failures surface as error messages in container status,
    and the CRI runtime's verbose container status (`crictl inspect`) shows
    the effective profile in the runtime spec.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The design target is that profile pulls should not add more than 2 seconds to
pod startup time (p99) when the profile is not cached and the registry is
reachable. This is not enforced by the kubelet but serves as a target for CRI
runtime implementations. Cached profile lookups should add less than 10 ms.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kubelet_security_profile_artifact_pull_duration_seconds`
  - Labels: `cached` (`true` when the runtime served the artifact from local
    storage, as reported in the CRI response)
  - Aggregation method: histogram
  - Components exposing the metric: kubelet
- [x] Metrics
  - Metric name: `kubelet_security_profile_artifact_pull_errors_total`
  - Labels: `reason` with a bounded set of values
    (`registry_unavailable`, `signature_validation_failed`,
    `security_profile_invalid`, `unauthenticated`, `other`)
  - Aggregation method: counter
  - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Cache hits and misses are distinguished by the `cached` label on the pull
duration histogram, so no separate cache metric is needed. The kubelet emits
pull duration and error metrics from its `PullSecurityProfileArtifact` calls,
which are collected from the kubelet's metrics endpoint (not via the summary
API). CRI runtimes may also emit their own metrics at the runtime level, for
example a counter of merge constraints (see
[Observability](#observability)); these would need to be collected from the
runtime's metrics endpoint directly.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- OCI-compatible container registry
  - Usage description: Stores and serves security profile artifacts.
  - Impact of its outage on the feature: Pods referencing uncached OCI profiles
    will fail to start. Pods referencing cached profiles or using other profile
    types are unaffected.
  - Impact of its degraded performance or high-error rates on the feature:
    Increased pod startup latency. The kubelet retries container and sandbox
    creation with backoff, which triggers re-pull attempts.

This feature also depends on the [security-profiles-merger][spm] library at
build time (not a runtime service). The library is a standalone Go module
(`sigs.k8s.io/security-profiles-merger`, a SIG Node subproject) used by CRI
runtimes to merge OCI profiles with baseline profiles via intersection. It handles
profile-type-specific merge logic for seccomp (including
architecture-specific syscall tables and argument filters) and, from beta,
AppArmor (capabilities, filesystem permissions, network rules), with the
interface designed to accommodate landlock. CRI runtimes that do not
integrate the library cannot perform profile merging and cannot support the
`OCI` profile type. See [Graduation Criteria](#graduation-criteria) for
phasing.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new Kubernetes API calls. The feature adds a `PullSecurityProfileArtifact`
CRI RPC and OCI registry pulls at the CRI runtime level, which are external to
the Kubernetes API.

###### Will enabling / using this feature result in introducing new API types?

No new API resources. `SeccompProfile` gains an `oci` field backed by two new
embedded struct types (`SecurityProfileOCIArtifact` and
`SecurityProfileOCIBase`).

###### Will enabling / using this feature result in any new calls to the cloud provider?

No, unless the OCI registry is a cloud-provider-managed registry (e.g., ECR,
GCR, ACR). In that case, credential provider plugins may make additional
calls, but this is the same mechanism used for container image pulls.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type: Pod
- Estimated increase in size: ~150 bytes per OCI profile reference in the spec
  (`oci` field with a digest-pinned reference and optional `baseProfile`).

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Pod startup latency may increase for pods using OCI profile references when
profiles are not cached. For uncached pulls, the expected latency is comparable
to small image pulls (under 2 seconds for typical profiles under 100 KB on
reasonable network conditions). This is not enforced by the kubelet but serves
as a design target for CRI runtime implementations. Cached pulls add negligible
latency.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

- Disk: Cached profiles consume disk space on the node. Profiles are small
  (typically < 100 KB) and bounded by the 1 MiB limit each; the number of
  cached artifacts is not bounded in alpha (see the resource exhaustion
  answer below).
- Network: Profile pulls add network traffic, but profiles are much smaller
  than container images.
- Memory: The CRI runtime holds parsed profiles in memory. This is already the
  case for `Localhost` profiles.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Cached profiles consume disk space and inodes. Each artifact is bounded by
the size limit (1 MiB), but alpha has no kubelet-driven garbage collection
and the number of distinct artifacts on a node is not bounded, so a stream
of short-lived pods with distinct profile digests grows the runtime's
artifact store until beta adds the `ListSecurityProfileArtifacts` and
`RemoveSecurityProfileArtifact` RPCs (see
[Risks and Mitigations](#risks-and-mitigations)). Runtimes should expose
the artifact store's count and byte size as metrics so operators can watch
it, and may cap total artifact storage by evicting unreferenced artifacts
in least-recently-used order. This gap is a consequence of the dedicated
RPC: image volumes (KEP-4639) reuse `PullImage` and are therefore covered
by the existing image garbage collector from the start.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

This feature does not interact with the API server or etcd at runtime. Profile
pulls happen between the CRI runtime and the OCI registry. API server
unavailability does not affect already-scheduled pods.

###### What are other known failure modes?

- Registry unreachable or authentication failure
  - Detection: `kubelet_security_profile_artifact_pull_errors_total` metric
    increases. Pod events show `SecurityProfilePullFailed`.
  - Mitigations: Pre-pull profiles so that they are cached (see
    [Pre-pulling](#notesconstraintscaveats)). Configure registry mirrors.
  - Diagnostics: CRI runtime logs show pull attempts and errors.
  - Testing: e2e tests with unreachable registry endpoints.

- Invalid or corrupt profile content
  - Detection: Pod events show profile validation errors.
  - Mitigations: Republish the artifact and update the digest in the pod
    template. Enable signature verification.
  - Diagnostics: CRI runtime logs show validation errors with profile details.
  - Testing: e2e tests with invalid profile content.

- Profile merge produces unexpected effective profile
  - Detection: Container behavior differs from expectations because the
    effective profile (intersection of OCI profile, base profile, and runtime
    baseline) is more restrictive than the OCI profile alone. CRI runtime
    warning logs identify which dimensions were constrained and by which
    input (see [Observability](#observability)).
  - Mitigations: Review the OCI profile, base profile, and runtime baseline
    to understand which restrictions each input contributes. The effective
    profile is the intersection: an operation is only permitted if all
    inputs permit it.
  - Diagnostics: CRI runtime warning logs show the per-dimension merge
    result (e.g., "syscall X denied by runtime baseline", "capability Y
    removed by base profile"). Conservative argument filter denials are
    also logged.
  - Testing: e2e tests with profiles that are more permissive than the
    baseline, verifying the merge constrains them and emits warning logs.

- Pod `Failed` at kubelet admission with reason
  `SecurityProfileOCIArtifactUnsupported`
  - Detection: Pod phase `Failed` with that reason; the node's runtime does
    not report `security_profile_oci_artifact` in `RuntimeFeatures`.
  - Mitigations: Upgrade the CRI runtime (and any `ImageService` proxy in
    front of it), or keep the kubelet feature gate disabled on nodes whose
    runtime lacks support so that they do not declare the feature.
  - Diagnostics: `crictl info` shows the runtime's `Status` response,
    including its features; kubelet logs show the admission rejection.
  - Testing: Unit tests for the admission handler with `RuntimeFeatures`
    lacking the flag.

- Pod stays `Pending` because no node declares the feature
  - Detection: Scheduler event naming the missing
    `SecurityProfileOCIArtifact` declared feature.
  - Mitigations: Enable the feature gate on kubelets whose runtime supports
    the feature, or remove the `OCI` profile from the pod.
  - Diagnostics: `kubectl get node <name> -o jsonpath='{.status.declaredFeatures}'`.
  - Testing: Scheduler integration tests.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check `kubelet_security_profile_artifact_pull_errors_total` for pull failures.
2. Check `kubelet_security_profile_artifact_pull_duration_seconds` for latency.
3. Verify registry connectivity from the node.
4. Check CRI runtime logs for credential resolution or pull issues.
5. Verify the artifact exists and has the correct media type.

## Implementation History

- 2026-05-06: Initial KEP draft
- 2026-06-16: Replaced the kubelet registry allowlist with runtime-side
  profile comparison
- 2026-06-17: Switched from subset validation to merge (intersection)
  semantics and added the layered `baseProfile` model
- 2026-06-22: Referenced the security-profiles-merger library
- 2026-09-09: Alpha scoped to seccomp with digest-only references (tags and
  AppArmor deferred to beta), Node Declared Features integration, KEP-2535
  alpha gap documented, merge input order and CRI error classification
  defined, beta criteria split into gate-on prerequisites and in-beta work
- 2026-09-09: Seccomp fallthrough fix proposed in
  [kubernetes/kubernetes#141958][k8s-141958]
- 2026-09-09: `ValidateArtifact` and artifact-size benchmarks added to the
  merge library; per-syscall entry cap documented
- 2026-09-09: SPO publishes runtime-spec seccomp profiles as KEP-6061
  artifacts ([security-profiles-operator#3421][spo-3421])
- 2026-09-10: Seccomp fallthrough fix merged for v1.38, which places the
  earliest possible beta at v1.41
- 2026-09-10: [security-profiles-operator#3421][spo-3421] merged
- 2026-09-10: Merge library moved to `kubernetes-sigs/security-profiles-merger`
  as a SIG Node subproject; e2e test artifacts promoted to `registry.k8s.io`
- 2026-09-11: Merge library released as `sigs.k8s.io/security-profiles-merger`
  v0.4.1 with the flag, `valueTwo`, and single-profile merge semantics fixed
  after a review against this KEP; the merge description above follows
- 2026-09-11: PodSecurity `restricted` relaxation moved from alpha to the
  v1.41 beta floor after review, because the API server cannot rely on a
  pre-1.38 kubelet rejecting an unknown profile type; `restricted` keeps
  rejecting `OCI` during alpha, the `baseline` skew exposure and the
  `nodeName` and ephemeral container gaps in Node Declared Features are
  documented, and the skew risk is listed under Risks and Mitigations

## Drawbacks

- **Adds complexity to the CRI API**: A new profile type, RPC
  (`PullSecurityProfileArtifact`), and message types increase the CRI surface
  area. However, the pattern mirrors the existing `PullImage`/`CreateContainer`
  separation and follows established conventions.
- **Registry dependency for pod startup**: Pods using OCI profiles cannot start
  if the registry is unreachable and profiles are not cached. This is the same
  trade-off that exists for container images.
- **Artifacts sit outside the kubelet's image garbage collector until beta**:
  Because profile artifacts are pulled through a dedicated RPC and must not
  appear in `ListImages`, the kubelet cannot garbage collect them in alpha.
  Image volumes (KEP-4639) avoided this by reusing `PullImage`. The cost is
  the unbounded artifact count described in the resource exhaustion answer;
  the benefit is profile-specific validation in the CRI contract.
- **CRI runtime implementation burden**: Each CRI runtime must implement the
  pull, cache, merge, and apply logic. Profile merging requires integrating
  the [security-profiles-merger][spm] library, which handles the complexity
  of seccomp architecture-specific syscall merging, argument filter
  intersection, and AppArmor rule intersection. CRI-O has already
  demonstrated the pull and apply flow for seccomp, and the merge library is
  shared across runtimes to avoid duplicating this logic.

## Alternatives

### Security Profiles Operator (SPO)

SPO already supports pulling seccomp profiles from OCI artifacts and
distributing them to nodes. However, SPO is a full operator with CRDs, RBAC,
webhooks, and a controller. For users who only need profile distribution
without recording, composition, or policy features, SPO is significant
overhead. This KEP provides the distribution primitive natively, which SPO (and
other tools) can build upon rather than re-implementing.

### CRI-Runtime-Only Pull

CRI-O's existing approach uses pod annotations to trigger OCI artifact pulls
entirely within the runtime, without Kubernetes API or CRI changes. This
works but has limitations:

- Annotations are not validated by the API server.
- Limited integration with Kubernetes-level credential management
  (imagePullSecrets). CRI-O uses its own registry auth configuration, which
  must be managed separately.
- Runtime-specific (other runtimes must independently implement the same
  annotation scheme).
- Not visible in `kubectl describe pod` or standard tooling.

The CRI-based approach proposed in this KEP addresses all of these limitations.

### Dynamic Resource Allocation (DRA)

DRA is designed to manage access to hardware resources and vendor-specific
devices. While DRA plugins can technically run arbitrary logic during resource
preparation, using DRA for security profile distribution would be a misuse of
the abstraction. Security profiles are not resources to be allocated; they are
configuration that modifies container behavior. DRA does not have a mechanism
to inject security context settings into the container spec, and overloading it
for this purpose would create confusing semantics. The CRI-based approach
proposed in this KEP is a better fit because security profile application is
already a CRI runtime responsibility.

### Node Resource Interface (NRI)

NRI plugins can modify container configurations at creation time, including
security context fields. An NRI plugin could theoretically pull OCI artifacts
and inject profile paths. However, this approach has several drawbacks: NRI
plugins operate outside the Kubernetes API (no validation, no status
reporting), credential management must be re-implemented in the plugin, and the
behavior is invisible to standard Kubernetes tooling. NRI is better suited for
node-level policy adjustments than for implementing a first-class distribution
mechanism. The proposed CRI approach provides end-to-end integration with the
Kubernetes API, credential management, and status reporting.

### Extending PullImage with Media Type

Instead of a dedicated `PullSecurityProfileArtifact` RPC, the `PullImage` RPC
could be extended with an optional media type field. The runtime would interpret
the media type to decide whether to apply profile-specific validation (size
limits, single-layer enforcement, content validation). This reduces CRI API
surface and reuses the existing pull path.

This alternative was considered but rejected for several reasons:

- Snapshotters in containerd proxy `ImageService` for credential handling.
  Extending `PullImage` would require snapshotters to handle profile-specific
  validation and response semantics inline. A dedicated
  `PullSecurityProfileArtifact` RPC only requires pass-through forwarding in
  the snapshotter proxy, with no profile-specific logic.
- Profile pulls have different validation requirements (size limits, single-layer
  enforcement, content validation, config media type matching) that would need
  to be conditional on the media type inside `PullImage`, making the pull path
  more complex.
- `PullSecurityProfileArtifactRequest` deliberately mirrors
  `PullImageRequest` (`ImageSpec`, `AuthConfig`, `PodSandboxConfig`) so that
  the credential and runtime handler flow is identical, but adds
  `profile_kind` for early media type matching. Encoding that in a media type
  field on `PullImage` would make `PullImage` behave differently depending on
  the field's value, which is harder to reason about for both runtimes and
  snapshotter proxies.
- A dedicated RPC allows profile-specific fields (`profile_kind` for media type
  validation) and future extensions (landlock support) without touching the
  container image pull path.

### Kubernetes API Object (ConfigMap with OCI Source)

Instead of a new `OCI` profile type, profiles could be modeled as Kubernetes
API objects, for example by extending ConfigMap to support an OCI artifact as
its data source. Pods would reference profiles via a `configMapRef`-style
field, and a controller would pull the OCI artifact and populate the ConfigMap.
This would make profiles manageable via `kubectl create|get|delete` and
visible as first-class cluster objects.

This alternative was considered but not chosen for several reasons:

- It introduces an indirection layer (OCI artifact to ConfigMap to pod) that
  adds latency, failure modes, and a controller dependency. The direct
  CRI-based approach pulls profiles on-demand at the node level, avoiding
  a cluster-level synchronization step.
- ConfigMaps have a 1 MiB size limit, which is sufficient for profiles but
  would store profile content in etcd, adding load to the control plane for
  data that is better cached at the node level.
- The CRI runtime already handles profile application and is the natural
  place to also handle profile retrieval. Routing through the API server
  adds a hop that does not improve security or reliability.
- The existing `Localhost` profile type already establishes the pattern of
  the kubelet and CRI runtime resolving profile content without API server
  involvement. The `OCI` type extends this pattern to registry-hosted
  profiles rather than introducing a fundamentally different object model.

### Annotation-Based Approach

Instead of extending the SecurityProfile types, profiles could be referenced
via standardized annotations (e.g., `security-profiles.kubernetes.io/seccomp`).
This avoids API changes but loses type safety, validation, and discoverability.
Given that SecurityProfile types already exist with a well-defined enum, adding
a new enum value is cleaner than introducing a parallel annotation scheme.

### Kubelet-Managed Pull into the Localhost Profile Directory

The kubelet could pull profile artifacts itself and place them under its
seccomp profile root, then treat the reference as a `Localhost` profile.
This would give the kubelet a registry client and content store of its own,
which it deliberately does not have (all pulls go through the CRI), would
duplicate credential and mirror handling that runtimes already implement,
and would produce plain `Localhost` semantics: a file on disk that any pod
on the node can reference and that is never merged with the baseline. The
CRI-based design keeps a single pull path and keeps the baseline guarantee.

### Init Container Writing the Profile to a Volume

An init container could fetch a profile and write it to an `emptyDir` for
the application container to use. Seccomp `Localhost` profiles must live
under the kubelet's configured seccomp root on the host, not in a pod
volume, and a container cannot apply a seccomp or AppArmor profile to a
sibling container in any case. This pattern therefore cannot work without
host access, at which point it is a DaemonSet distributing files, which the
Motivation section already covers.

### Earlier Iterations of This KEP

Two earlier designs were replaced during review:

- **Subset validation without merge**: The CRI runtime validated at pull time
  that every OCI profile was at least as restrictive as the node's baseline
  and rejected profiles that exceeded it. This is operationally fragile:
  tightening the baseline breaks every existing OCI profile that does not
  already include the new restriction, so the admin must republish all
  profiles before changing the baseline. It also cannot express layered
  profiles. Merge (intersection) semantics apply new baseline restrictions
  automatically and support a per-workload-class `baseProfile`.
- **Kubelet registry allowlist**: A deny-by-default
  `securityProfileOCIArtifact.allowedRegistries` list in
  `KubeletConfiguration` restricted where profiles could be pulled from.
  This is source-based trust: it controls where a profile comes from, not
  what it does, and a compromised allowed registry could still serve an
  over-permissive profile. It also required per-node configuration, which
  does not improve on distributing `Localhost` files. Profile merging gives
  a content-based guarantee regardless of source.

## Infrastructure Needed (Optional)

- Sample seccomp profile artifacts published to `registry.k8s.io` through the
  image promoter, alongside the existing e2e test images, using the typed
  config media type. Done on 2026-09-10:
  `registry.k8s.io/security-profiles-operator/seccomp-test-profiles` carries
  the `deny-chmod`, `permissive`, `invalid`, and `oversized` fixtures,
  promoted from SPO's staging build; the promoter preserves `artifactType`
  and the config media type and signs the promoted copies.
- A CRI-O based `e2e_node` job with the `SecurityProfileOCIArtifact` feature
  gate enabled.
