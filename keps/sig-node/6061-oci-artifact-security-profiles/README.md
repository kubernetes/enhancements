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
    - [Story 2: Vendor-Provided AppArmor Profiles](#story-2-vendor-provided-apparmor-profiles)
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
  - [Subset Validation Without Merge](#subset-validation-without-merge)
  - [Kubelet Allowlist for OCI Profile Sources](#kubelet-allowlist-for-oci-profile-sources)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

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
- [ ] Supporting documentation, e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes native support for pulling security profiles (seccomp and
AppArmor) from OCI-compatible registries. Today, Kubernetes requires security
profiles to be pre-installed on every node (`Localhost` type) or limits users
to the built-in `RuntimeDefault`. This creates operational burden for cluster
administrators who must distribute and synchronize profiles across all nodes
using external tooling such as DaemonSets, node image baking, or the
[Security Profiles Operator (SPO)][spo].

[spo]: https://github.com/kubernetes-sigs/security-profiles-operator

By extending the Kubernetes API and CRI to support OCI artifact references for
security profiles, users can store versioned, immutable profiles in container
registries alongside the container images they protect. The kubelet resolves
pull credentials and passes them to the CRI runtime, which pulls the artifacts
using the same registry infrastructure already in place for container images.

The design supports a layered profile model: pods can specify both a base
profile (Localhost or RuntimeDefault) and an OCI overlay profile. The CRI
runtime merges the OCI profile with the base profile and its own configured
baseline via intersection, producing an effective profile that is at least as
restrictive as every input. This guarantees that OCI-distributed profiles
cannot weaken node security regardless of their content, while allowing
administrators to enforce per-workload-class baselines without breaking
existing OCI profiles when the baseline changes.

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
first-class Kubernetes API feature, covering seccomp and AppArmor with a
uniform approach.

[oras]: https://oras.land/

Emerging use cases in AI agent sandboxing further motivate this work. Projects
like [NVIDIA OpenShell][openshell] use per-container security profiles for
fine-grained isolation of AI agent workloads. Native profile distribution via
OCI artifacts would let platform teams publish and version seccomp and AppArmor
profiles alongside the agent images they protect.

[openshell]: https://github.com/NVIDIA/OpenShell

### Goals

- Add an `OCI` profile type to the Kubernetes `SeccompProfile` and
  `AppArmorProfile` API types, allowing users to reference security profiles
  stored in OCI-compatible container registries.
- Extend the CRI API with a dedicated `PullSecurityProfileArtifact` RPC and an
  `OCI` profile type in the `SecurityProfile` message, enabling the kubelet to
  pull profiles via the runtime and pass resolved digests to sandbox/container
  creation calls.
- Reuse existing image pull infrastructure (pull secrets, credential providers,
  registry authentication) for profile pulls.
- Support pulling profiles by tag or digest.
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
  library (see [Profile Merging](#profile-merging)) are
  profile-type-agnostic and extend
  to landlock by adding a new `SecurityProfileKind` enum value, a new OCI
  config media type, and a landlock merge implementation in the library.
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
  Kyverno, OPA/Gatekeeper). However, the built-in PodSecurity admission
  controller must be updated to accept the `OCI` type (see
  [Design Details](#kubernetes-api-changes)).
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
to their existing container registry and reference them directly in pod specs:

```yaml
securityContext:
  seccompProfile:
    type: OCI
    oci:
      reference: "registry.example.com/security/profiles/api-server-seccomp:v2.1"
```

The CRI runtime pulls the profile using credentials resolved by the kubelet
from the pod's `imagePullSecrets`. Profile updates are decoupled from node
image updates.

#### Story 2: Vendor-Provided AppArmor Profiles

A database vendor ships a recommended AppArmor profile for their containerized
database. Without this feature, customers must manually install the profile on
every node. With OCI artifact support, the vendor publishes the profile to a
public registry and customers reference it directly:

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
      reference: "registry.example.com/agents/code-exec-seccomp:v1.0"
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

- **Immutability and digest pinning**: OCI profile references are immutable for
  the lifetime of a pod. Changing a security profile reference requires
  recreating the pod, the same as changing any other field in
  `securityContext`. The kubelet resolves each tag-based profile reference to a
  digest once via the `PullSecurityProfileArtifact` CRI RPC (see
  [CRI API Changes](#cri-api-changes)) and pins the resolved digest in
  container status for the pod's lifetime. All subsequent container creations and restarts
  within the pod use the pinned digest, not the original tag. This prevents
  profile version skew between the pod sandbox and its containers: even if a
  tag is updated in the registry or re-pulled by another pod, the pinned digest
  ensures every container in the pod uses the same profile content. Digest-based
  references in the pod spec are already immutable and bypass tag resolution
  entirely. New pods always resolve tags fresh, so deploying an updated profile
  only requires rolling pods, not any special re-pull mechanism.
- **Runtime profile caching**: The CRI runtime caches profile artifacts in its
  existing content store (the same store used for container image layers), keyed
  by digest. No separate storage is needed. Because profiles are stored as
  standard OCI artifacts, they share the content-addressable storage that the
  runtime already manages. The key requirement is that profile lookups use the
  digest, not the tag, independent of OCI image tag semantics. This matters
  because another pod referencing the same tag could trigger a re-pull that
  updates the tag-to-digest mapping in the runtime's image store. By caching
  profile content by digest rather than by tag, the runtime ensures that a previously resolved digest
  always returns the same profile content regardless of tag mutations. The
  kubelet only ever passes the pinned digest (not the tag) to
  `RunPodSandbox`/`CreateContainer`, so tag re-pulls by other pods cannot
  affect running pods.
- **Cache eviction with stale tags**: If the cached profile artifact is evicted
  from the runtime's content store (for example, by garbage collection) while
  the pod is still running, and the tag has since been updated in the registry
  to point to a different digest, the runtime re-pulls by the pinned digest,
  not the tag. If the registry no longer serves the pinned digest (for example,
  the old manifest was deleted), the `PullSecurityProfileArtifact` call fails
  and the container creation fails with a `SecurityProfilePullFailed` event.
  The pod is not silently left without a profile. This is a narrow case but is
  a defined error condition: the pod remains running (profiles are loaded into
  the kernel at creation time), but any new container creation within the pod
  (restart, new init container) will fail until the digest is available again.
- **Pre-pulling and content store sharing**: Because profile artifacts are
  stored as standard OCI artifacts in the runtime's content store, a profile
  that was pre-pulled via the regular `PullImage` API (for example, by a
  DaemonSet or node setup script) will already be present in the content store.
  The `PullSecurityProfileArtifact` call will find it cached and skip the
  network pull. The runtime still validates the artifact's media type, layer
  count, and size on the `PullSecurityProfileArtifact` path regardless of how
  the content arrived in the store.
- **Registry availability and retry semantics**: If the registry is unreachable,
  pods that reference artifact profiles will fail to start, similar to how image
  pull failures prevent pod startup. Profile caching mitigates this for
  steady-state operation. Profile pulls follow the kubelet's existing retry
  model: each pod sync attempt triggers one `PullSecurityProfileArtifact`
  call for uncached profiles, before sandbox or container creation begins.
  The kubelet's existing exponential backoff governs retry timing. There is
  no separate retry policy for profile pulls. If a profile remains
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
  cache resources.
- **No pull policy field**: Unlike container images, which have
  `imagePullPolicy`, security profile artifacts always use pull-if-not-present
  semantics. This is a deliberate simplification for alpha. Digest-pinned
  references (recommended for production) are immutable and never re-pulled.
  A pull policy field can be added in a future iteration if demand emerges
  (the `SecurityProfileOCIArtifact` struct allows this without API breakage).
- **Intersection with KEP-2535 (Ensure Secret Pulled Images)**: Because
  profile artifacts use pull-if-not-present semantics, the same credential
  reuse concern from [KEP-2535][kep-2535] applies: a profile pulled by one pod
  could be available in the runtime's content store for another pod that lacks
  credentials for that registry. Profile pulls reuse the kubelet's image pull
  credential path, so KEP-2535's credential verification policy
  (`imagePullCredentialsVerificationPolicy`) applies to profile artifacts
  automatically. When credential verification is enabled, the kubelet
  re-validates credentials against the registry before allowing a cached
  profile to be used by a new pod, preventing cross-tenant reuse of cached
  profiles. For alpha, this integration works without additional changes
  because `PullSecurityProfileArtifact` uses the same `AuthConfig` and
  credential resolution as `PullImage`. The kubelet tracks which credentials
  were used for each profile pull and applies the same verification policy
  configured for images.
- **Static pods**: Static pods do not have `imagePullSecrets` on the pod spec
  (there is no API server to resolve service account secrets). For static pods,
  credential resolution falls back to the kubelet's configured credential
  providers and any node-level registry auth configuration, the same behavior
  as container image pulls for static pods.
- **Privileged containers**: Containers running with `privileged: true` have
  their seccomp profile forced to `Unconfined` and their AppArmor profile set
  to `unconfined` by the runtime. Unlike `type: Localhost` (which predates this
  restriction), `type: OCI` is rejected at API validation on privileged
  containers. Since this is a new field, there is no backward compatibility
  concern, and rejecting early prevents confusion where a user specifies a
  profile that is silently ignored.
- **Layered profile semantics**: When the pod spec specifies a `baseProfile`
  alongside the OCI reference, the CRI runtime merges the OCI profile, the
  pod-spec base profile, and the runtime's own configured baseline via
  intersection. An operation is permitted in the effective profile only if all
  input profiles permit it. This means:
  - The runtime's configured baseline (RuntimeDefault or an admin-supplied
    profile on disk) is always the absolute floor. No pod spec can weaken it.
  - The pod-spec `baseProfile` adds per-workload-class restrictions on top of
    the runtime baseline. Different pods on the same node can have different
    base profiles, enabling workload-class-specific baselines (for example,
    a restrictive baseline for agentic workloads) without node-pool
    segregation.
  - The OCI profile adds per-workload restrictions from the registry.
  - If the OCI profile is more permissive than the baseline in some
    dimensions, those dimensions are constrained by the merge. The pod runs
    with the tighter-of-all profile. No profile is rejected for being "too
    permissive." When the merge constrains the OCI profile, the CRI runtime
    logs the constrained dimensions at the `warning` level (see
    [Observability](#observability)).
  - When the admin tightens the base profile or runtime baseline, existing
    OCI profiles continue to work because the merge automatically applies
    the new restriction. No OCI profiles need to be republished.
  When `baseProfile` is omitted, the runtime uses its configured default
  baseline only, producing a two-way merge (runtime baseline, OCI profile).
- **Linux-only**: Seccomp and AppArmor are Linux security mechanisms. This
  feature applies only to Linux containers. The CRI changes are scoped to
  `LinuxContainerSecurityContext` and `LinuxSandboxSecurityContext`. Windows
  containers are unaffected.

### Risks and Mitigations

**Risk**: Pulling profiles adds latency to pod startup.
**Mitigation**: The CRI runtime caches pulled profiles locally
(content-addressable by digest). After the first pull, subsequent pods using
the same profile reference hit the local cache. Digest-based references are
immutable and never re-pulled. Tag-based references are resolved to a digest
once via `PullSecurityProfileArtifact` and the kubelet pins the digest for the
pod's lifetime (see [Immutability and digest pinning](#notesconstraintscaveats)).
Other pods pulling the same tag do not affect already-pinned digests.

**Risk**: Malicious or corrupted profiles could compromise node security.
**Mitigation**: The CRI runtime merges every pulled OCI profile with the
node's baseline profile via intersection (see
[Profile Merging](#profile-merging)). The effective profile permits an
operation only if all input profiles (runtime baseline, pod-spec base profile,
and OCI profile) permit it. Even if a profile is tampered with in transit
(MitM) or a registry is compromised, the effective profile can only be as
permissive as the baseline, never more. A malicious OCI profile that attempts
to allow all syscalls is silently constrained to the baseline's restrictions
by the merge. The runtime also validates profile content before applying it
(e.g., valid JSON for seccomp). Additionally, profile artifacts can be
verified using standard OCI signature verification (sigstore/cosign) and
admission controllers can enforce that only signed profiles from trusted
registries are allowed.

**Risk**: Pods pulling arbitrary security profiles from the internet bypasses
node admin control. With `Localhost` profiles, node admins control which
profiles are available on disk. OCI profiles could shift that control to pod
authors, who can reference any artifact from any registry.
**Mitigation**: The CRI runtime's profile merging ensures that the effective
profile is always at least as restrictive as the node's baseline profile,
regardless of the OCI profile's source or content. The node admin controls
the baseline via the runtime's configuration (a file on disk or the
RuntimeDefault profile). This is a stronger guarantee than source-based
trust: rather than trusting that a registry only contains safe profiles, the
runtime constrains each profile to the baseline's permissions by
construction (intersection). Additionally, pods can specify a `baseProfile`
(Localhost or RuntimeDefault) that adds per-workload-class restrictions on
top of the runtime baseline; admission controllers can enforce that specific
workload classes use specific base profiles. The CRI runtime's signature
verification infrastructure (e.g., CRI-O's `/etc/containers/policy.json`)
provides an additional layer of control over profile origins, and admission
controllers can enforce reference patterns at the cluster level.

**Risk**: Garbage collection of cached profile artifacts could force
unnecessary re-pulls.
**Mitigation**: Profiles are loaded into the kernel at container creation time,
so GC of the cached artifact does not affect already-running containers.
However, if the cached artifact is removed while the pod is still running, any
new container creation (restart, new init container) would require a re-pull.
Both the kubelet and the CRI runtime track profile artifacts in use, mirroring
how both components already track container images in use. The kubelet maintains
the set of pinned profile digests from running pods and factors these into GC
decisions. The CRI runtime protects cached profile artifacts referenced by
running pods from garbage collection. For alpha, profile GC is runtime-managed
(the runtime evicts unreferenced profiles from its content store). An explicit
`RemoveSecurityProfileArtifact` RPC for kubelet-initiated removal can be added
in beta if needed, alongside the `ListSecurityProfileArtifacts` RPC for
enumeration.

**Risk**: Increased registry load from profile pulls.
**Mitigation**: Profiles are small (typically < 100 KB) and aggressively cached.
The additional registry load is negligible compared to container image pulls.

**Risk**: Registry as a single point of failure (denial of service).
**Mitigation**: Registry unavailability only affects pods that reference uncached
OCI profiles. Once a profile is cached locally by the CRI runtime, pods start
without registry access. Digest-pinned references (recommended for production)
are immutable and never need re-resolution. Operators can mitigate registry
dependency by using registry mirrors, pre-pulling profiles via DaemonSets or
node setup scripts, and monitoring
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

**Risk**: Pod status as an attack surface for profile substitution.
**Mitigation**: The kubelet stores the resolved profile digest in
the container status (e.g., `containerStatuses[*].seccompProfileArtifactDigest`)
so that it survives kubelet restarts and allows operators to audit which profile
content was applied. An actor with write access to pod status could modify the
resolved digest, causing the kubelet to apply a different profile on the next
container creation. This risk is mitigated by the fact that pod status write
access already implies significant cluster privileges (typically restricted to
the kubelet's node authorization or cluster administrators). The kubelet is the
only component that writes these status fields, and the API server can enforce
this via the NodeRestriction admission plugin, which limits kubelets to
updating the status of pods bound to their node. Additionally, on kubelet
restart the kubelet cross-checks the digest stored in status against the
original `oci.reference` from the pod spec: the repository portion of the digest
must match the repository in the spec reference. A tampered digest pointing to
a different repository is rejected, and the kubelet re-resolves from the
original spec reference. This prevents cross-pod profile substitution where an
attacker writes another pod's cached profile digest into this pod's status.

## Design Details

### Kubernetes API Changes

Extend `SeccompProfile` and `AppArmorProfile` in `k8s.io/api/core/v1` with a
new type and reference field:

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
    // The format is a standard OCI reference: registry/repository[:tag|@digest]
    // Must be a fully-qualified reference (no short names).
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
    // +unionDiscriminator
    Type SecurityProfileOCIBaseType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=SecurityProfileOCIBaseType"`

    // localhostProfile specifies the base profile path on the node.
    // For seccomp, this is a descending path relative to the kubelet's
    // configured seccomp profile location.
    // For AppArmor, this is the AppArmor profile name.
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

The same pattern applies to `AppArmorProfile`:

```go
type AppArmorProfile struct {
    // +unionDiscriminator
    Type AppArmorProfileType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=AppArmorProfileType"`

    // +optional
    LocalhostProfile *string `json:"localhostProfile,omitempty" protobuf:"bytes,2,opt,name=localhostProfile"`

    // +featureGate=SecurityProfileOCIArtifact
    // +optional
    OCI *SecurityProfileOCIArtifact `json:"oci,omitempty" protobuf:"bytes,3,opt,name=oci"`
}

const (
    AppArmorProfileTypeRuntimeDefault AppArmorProfileType = "RuntimeDefault"
    AppArmorProfileTypeLocalhost      AppArmorProfileType = "Localhost"
    AppArmorProfileTypeUnconfined     AppArmorProfileType = "Unconfined"
    AppArmorProfileTypeOCI            AppArmorProfileType = "OCI"
)
```

API validation:
- `oci` must be set when `type` is `OCI`, and must not be set for other types.
- `oci.reference` must be a fully-qualified OCI reference (no short names).
  Validation uses `distribution/reference` parsing to verify the reference
  format, consistent with how container image references are parsed elsewhere
  in the codebase (note: the existing `image` field on containers does not
  enforce strict validation, but the new `oci.reference` field validates
  format at admission time since it is a new field with no backward
  compatibility constraints).
- `oci.baseProfile` is optional. When set, `baseProfile.type` must be
  `RuntimeDefault` or `Localhost`. When `baseProfile.type` is `Localhost`,
  `baseProfile.localhostProfile` must be set. When `baseProfile.type` is
  `RuntimeDefault`, `baseProfile.localhostProfile` must not be set.
  `baseProfile.type` must not be `Unconfined` or `OCI` (no chaining OCI
  profiles, no unconfined base).
- `type: OCI` is rejected on privileged containers at API validation (see
  [Privileged containers](#notesconstraintscaveats)).
- Digest-pinned references (`@sha256:...`) are recommended for production use.

When the `SecurityProfileOCIArtifact` feature gate is disabled, two mechanisms
apply:
- **Type validation**: API validation rejects the `OCI` type value, preventing
  creation of pods that use OCI profile references.
- **Field stripping**: The `oci` field is stripped from new objects following
  the standard drop-disabled-fields pattern. Existing objects that already
  have the field set (created while the gate was enabled) retain the value on
  update to prevent data loss.
- **Status field stripping**: The `seccompProfileArtifactDigest` and
  `appArmorProfileArtifactDigest` fields in `ContainerStatus` follow the same
  pattern: stripped from status updates when the gate is disabled, retained on
  existing objects to avoid data loss.

Extend `ContainerStatus` with fields to report resolved artifact digests:

```go
type ContainerStatus struct {
    // ...existing fields...

    // seccompProfileArtifactDigest is the digest-pinned reference of the
    // OCI seccomp profile artifact applied to this container, if any
    // (e.g., "registry.example.com/profile@sha256:abc123...").
    // Set by the kubelet after a successful PullSecurityProfileArtifact call.
    // For pod-level profiles, the resolved digest is propagated to each
    // container's status. The registry and repository portions are preserved
    // from the original pull, ensuring that any re-pull uses the same
    // registry and credentials.
    // +featureGate=SecurityProfileOCIArtifact
    // +optional
    SeccompProfileArtifactDigest string `json:"seccompProfileArtifactDigest,omitempty" protobuf:"bytes,TBD,opt,name=seccompProfileArtifactDigest"`

    // appArmorProfileArtifactDigest is the digest-pinned reference of the
    // OCI AppArmor profile artifact applied to this container, if any.
    // Same semantics as seccompProfileArtifactDigest.
    // +featureGate=SecurityProfileOCIArtifact
    // +optional
    AppArmorProfileArtifactDigest string `json:"appArmorProfileArtifactDigest,omitempty" protobuf:"bytes,TBD,opt,name=appArmorProfileArtifactDigest"`
}
```

Using `ContainerStatus` rather than a separate status type follows the
existing pattern: each container already reports its own state, and the
resolved profile digest is per-container (even when inherited from a pod-level
profile). For pod-level profiles, the kubelet writes the same resolved digest
into every container's status.

**Pod Security Standards**: The built-in PodSecurity admission controller must
be updated to treat the `OCI` type the same as `Localhost` for the `restricted`
and `baseline` levels. Both represent user-selected profiles, and without this
update, pods using `type: OCI` would be rejected in namespaces enforcing
`restricted` Pod Security. This update is part of the kube-apiserver component
of the `SecurityProfileOCIArtifact` feature gate.

The trust model for `OCI` profiles is stronger than `Localhost`. With
`Localhost`, the node admin controls which profiles are available by placing
files on disk, but there is no validation that those profiles are
sufficiently restrictive. With `OCI`, the CRI runtime merges every pulled
profile with the node's baseline profile via intersection (see
[Profile Merging](#profile-merging)). The effective profile permits an
operation only if both the baseline and the OCI profile permit it, so the
result cannot weaken security below the baseline regardless of the OCI
profile's content. Different nodes can use different baseline profiles via
per-node runtime configuration, the same way different nodes can have
different files on disk.

PSA allows `Localhost` for `restricted` and `baseline` based on the
assumption that the node admin controls which profiles are available. PSA
does not verify file presence or profile content at admission time. The
trust model for `OCI` is stronger: PSA allows it because the CRI runtime
guarantees that the effective profile is no more permissive than the node's
baseline by construction (intersection). No coordination between PSA and
the CRI runtime is needed.

Unlike `Localhost`, where a profile on disk can be more permissive than the
runtime default (a seccomp profile can allow all syscalls; an AppArmor
profile can grant unrestricted access), `OCI` profiles are merged with the
baseline at apply time. An OCI profile that is effectively unconfined would
be silently constrained to the baseline's restrictions by the merge,
producing an effective profile equivalent to the baseline itself.

The CRI runtime's signature verification infrastructure (for example,
CRI-O's system-wide `/etc/containers/policy.json`) provides an additional
layer of control over profile artifact pulls. Cluster administrators can
also use admission webhooks (Kyverno, OPA/Gatekeeper) to restrict which
OCI references are allowed. The `oci.reference` field is part of the pod
spec, making it visible to all admission controllers. This allows policies
such as "only allow profiles from
`registry.internal.example.com/approved-profiles/`" or "require
digest-pinned references."

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
        // DefaultBaseType means the runtime uses its configured default
        // baseline. This is equivalent to not setting base_profile at all.
        DefaultBaseType = 0;
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

A dedicated RPC is used instead of reusing `PullImage` because profile
artifacts have different semantics: they require media type validation (only
seccomp or AppArmor config types are accepted), content validation (valid JSON
for seccomp, valid AppArmor policy language), size enforcement (the 1 MiB
default limit is far smaller than container images), and single-layer
verification. These constraints do not apply to container images and would
complicate `PullImage` if added there. The request includes a `profile_kind`
field so the runtime can validate the artifact's config media type matches the
expected security mechanism early, before extracting content. See
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
    // reference is the OCI reference
    // (e.g., "registry.example.com/profile:v1" or
    // "registry.example.com/profile@sha256:abc123...").
    string reference = 1;

    // auth contains registry authentication credentials, resolved by the
    // kubelet from imagePullSecrets, service account credentials, and
    // credential providers. Uses the same AuthConfig message as the PullImage
    // RPC.
    AuthConfig auth = 2;

    // profile_kind identifies the expected security mechanism (Seccomp or
    // AppArmor). The runtime uses this to validate that the pulled artifact's
    // config media type matches the expected kind and to reject mismatches
    // early (e.g., a seccomp artifact referenced in an apparmor field).
    // SecurityProfileKindUnspecified is rejected with InvalidArgument.
    SecurityProfileKind profile_kind = 3;

    // sandbox_config holds the pod sandbox configuration, including the
    // runtime handler. In containerd, the runtime handler determines which
    // snapshotter is used for image pulls, and snapshotters proxy
    // ImageService to obtain registry credentials. Including the sandbox
    // config ensures the pull uses the correct snapshotter and credential
    // flow for the pod's runtime handler, matching the behavior of
    // PullImage.
    PodSandboxConfig sandbox_config = 4;
}

enum SecurityProfileKind {
    SecurityProfileKindUnspecified = 0;
    Seccomp = 1;
    AppArmor = 2;
}

message PullSecurityProfileArtifactResponse {
    // resolved_digest is the digest-pinned reference that was pulled
    // (e.g., "registry.example.com/profile@sha256:abc123...").
    // For digest-pinned input references, this is the same as the input.
    string resolved_digest = 1;
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

1. **Resolve credentials**: The kubelet resolves pull credentials using the same
   code path as container image pulls: `imagePullSecrets` on the pod spec,
   service account image pull secrets, and any configured credential provider
   plugins. The credential resolution logic in `pkg/kubelet/images` and
   `pkg/kubelet/kubelet_pods.go` is reused directly.
2. **Pull the profile**: The kubelet calls `PullSecurityProfileArtifact` with
   the OCI reference and resolved credentials. This happens before preparing
   DRA resources, so a pull failure does not require
   cleaning up already-prepared resources. The CRI runtime pulls the artifact
   (if not cached) and validates its media type, content format, size, and
   layer count. No profile-vs-baseline comparison happens at pull time; the
   merge occurs later at apply time (see
   [Profile Merging](#profile-merging)). The runtime returns the resolved
   digest. The kubelet records the resolved digest in the container status and
   pins it for the pod's lifetime.
3. **Prepare resources**: After all profile pulls succeed, the kubelet proceeds
   with DRA resource preparation and other pod setup.
4. **Pass to CRI**: The kubelet constructs the `SecurityProfile` message with
   `profile_type = OCI` and `oci_ref` set to the pinned digest. If the pod
   spec includes a `baseProfile`, the kubelet populates the
   `SecurityProfile.base_profile` field with the corresponding type and
   localhost reference. For `RunPodSandbox`, the sandbox-level profile
   digest and base profile are included. For `CreateContainer`, the
   container-level profile digest and base profile are included.
5. **CRI runtime merges and applies the profile**: The runtime looks up the
   cached OCI profile by digest, loads the base profile (from
   `base_profile` if specified, or its configured default baseline), and
   merges all inputs via intersection (see
   [Profile Merging](#profile-merging)). The effective profile permits an
   operation only if all input profiles permit it. The runtime applies the
   merged profile to the container. No pull occurs at this stage.
6. **Container restarts**: On kubelet directed container restarts, the kubelet calls
   `PullSecurityProfileArtifact` again with the pinned digest (not the
   original tag). The runtime returns quickly from cache in the common case.
   If the cached artifact was garbage-collected, the runtime re-pulls by
   digest. This ensures the cache is warm before `CreateContainer` and
   maintains consistent profile content throughout the pod's lifetime.
7. **Kubelet restart**: On kubelet restart, the kubelet reads the previously
   pinned digest from the container status
   (`seccompProfileArtifactDigest` / `appArmorProfileArtifactDigest`). If a
   pinned digest exists, the kubelet uses it directly for subsequent
   `PullSecurityProfileArtifact` calls (ensuring cache warmth) and
   `RunPodSandbox`/`CreateContainer` calls, without re-resolving the tag.
   If no pinned digest is found in status (for example, the kubelet crashed
   before writing it), the kubelet re-resolves from the original spec
   reference. This is why the resolved digest is stored in container status
   rather than in kubelet-local state.

Profile pulls are independent of container image pulls and can run concurrently
with them. The kubelet pulls all unique profiles for a pod before proceeding to
DRA resource preparation. Integration with the parallel image pull feature
(KEP-3876) is a potential optimization for future work.

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
Ephemeral containers are added to a running pod after initial setup, so their
profile pulls happen inline at container creation time, not during the initial
pull-then-prepare sequence described above.

`Localhost` profiles are not validated at admission time either; the kubelet
constructs the profile path and passes it to the CRI runtime, which fails at
sandbox or container creation if the file is missing. `OCI` profiles follow
the same pattern: pull failures surface as sandbox or container creation
failures. The kubelet treats these the same as container image pull failures:
the pod remains in `Pending` and the kubelet retries with exponential backoff
(up to a maximum interval). The kubelet emits `SecurityProfilePulled` events
on successful pulls and `SecurityProfilePullFailed` events on failures.

If the CRI runtime permanently rejects a profile (for example, an invalid
media type, corrupt content, or an oversized artifact), the kubelet marks the
pod as terminally failed rather than retrying indefinitely. This follows the
precedent set by CSI errors during sandbox creation, where certain
deterministic failures cause the kubelet to stop retrying and surface a
terminal pod condition. Terminal failures are distinguished from transient
errors (registry unreachable, authentication timeout) by the CRI error code:
`InvalidArgument` and `Unimplemented` indicate permanent rejection, while
`Unavailable` or `DeadlineExceeded` trigger retries with backoff. This
prevents pods with bad profile references from being stuck in an infinite
retry loop.

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
three-way. The merge operation is commutative and associative, so the order
of inputs does not affect the result.

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
  `SCMP_ACT_ERRNO` > `SCMP_ACT_TRACE` > `SCMP_ACT_NOTIFY` >
  `SCMP_ACT_LOG` > `SCMP_ACT_ALLOW`. `SCMP_ACT_TRACE` ranks above
  `SCMP_ACT_NOTIFY` and `SCMP_ACT_LOG` because the ptrace tracer can
  modify or deny the syscall, whereas `SCMP_ACT_LOG` always allows it.
  Default actions, per-syscall rules,
  and architecture-specific entries are all merged. When argument filters
  are present and the intersection cannot be computed precisely (e.g.,
  overlapping but non-identical filter conditions), the merge is
  conservative: the syscall is denied. For architectures present in one
  profile but not another, the missing profile's default action applies for
  that architecture.
- **AppArmor**: Capabilities are intersected (the effective profile grants
  only capabilities present in all inputs). File access rules are
  intersected (the effective profile grants only file permissions present in
  all inputs). Network rules and other AppArmor permissions follow the same
  principle.
- **Landlock**: The merge library interface accommodates landlock, but
  implementation is deferred until the OCI runtime spec defines the landlock
  profile format (runtime-spec [PR #1241][landlock-pr]). Landlock profiles
  are inherently additive (they can only narrow access, never widen it),
  which makes the merge trivial: the merged profile is the union of
  restrictions from all inputs.

**Profile merge library**: Profile merging requires non-trivial
profile parsing and merge logic, especially for seccomp profiles with
architecture-specific syscall tables and argument filters. The
[security-profiles-merger][spm] library provides this functionality as a
standalone Go module (`github.com/saschagrunert/security-profiles-merger`),
used by CRI runtimes to merge profiles. The library exposes an `Intersect`
API per profile type (seccomp and AppArmor) and is designed to be extensible
to new profile types (such as landlock) without changes to the CRI protocol
or the kubelet. The module has no dependencies beyond the
[OCI runtime specification][oci-runtime-spec] (`runtime-spec v1.3.0`), so
both CRI-O and containerd can import it without circular dependencies.

The library also provides a `Union` API for combining profiles (used by SPO
to merge recorded profiles), but CRI runtimes only use `Intersect` for
baseline enforcement.

[spm]: https://github.com/saschagrunert/security-profiles-merger
[oci-runtime-spec]: https://github.com/opencontainers/runtime-spec
[crio-10037]: https://github.com/cri-o/cri-o/issues/10037

**Merge at apply time**: Profile merging happens at container or sandbox
creation time (`RunPodSandbox`/`CreateContainer`), not at pull time
(`PullSecurityProfileArtifact`). The pull RPC fetches and caches the raw OCI
artifact with content validation (valid JSON for seccomp, valid AppArmor
syntax, correct media type, size limits, single-layer enforcement). The
merge happens when the runtime applies the profile, using the cached OCI
artifact, the base profile from the CRI message, and the runtime's
configured baseline. This allows the same cached OCI artifact to be merged
with different base profiles for different pods, and allows baseline changes
to take effect without re-pulling artifacts.

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
| AppArmor | `application/vnd.cncf.apparmor-profile.config.v1+json` |

This KEP documents these media types but does not define them; media type
standardization is OCI-level work outside the scope of a Kubernetes KEP. The
seccomp media type is already in production use by CRI-O (since v1.30) and SPO.
The AppArmor media type follows the same naming convention. Both use the
`vnd.cncf.` vendor prefix, registered under the vendor tree as defined in
[RFC 6838, Section 3.2][rfc6838].

[rfc6838]: https://datatracker.ietf.org/doc/html/rfc6838#section-3.2

Profile content is stored as a single layer in the artifact. The artifact must
contain exactly one layer; runtimes must reject artifacts with zero layers or
more than one layer to ensure consistent behavior across CRI implementations.

The CRI runtime knows which security mechanism to apply from the CRI field
context (the `seccomp` or `apparmor` field on `LinuxContainerSecurityContext`).
The config media type serves as a validation check: if the media type does not
match the expected type for the CRI field (e.g., a seccomp artifact referenced
in the `apparmor` field), the runtime must reject the profile. The expected
content formats are:

- **Seccomp**: The layer contains a JSON seccomp profile as defined by the
  [OCI runtime spec][oci-seccomp].
- **AppArmor**: The layer contains an AppArmor profile in the standard
  AppArmor policy language.

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

Existing seccomp and AppArmor e2e tests provide a baseline. No prerequisite
updates are required.

##### Unit tests

- Seccomp's `fieldSeccompProfile` rejects unknown profile types with an
  explicit error (prerequisite fix, not gated by this feature)
- API validation for the new `OCI` profile type and `oci` field
  across seccomp and AppArmor
- PodSecurity admission accepts `OCI` at restricted and baseline levels
- Kubelet credential resolution for artifact pulls
- CRI message construction with OCI artifact references
- Profile merge library ([security-profiles-merger][spm]): seccomp merge
  across architectures, default actions, per-syscall rules, and argument
  filters (including conservative denial when intersection is unprovable);
  AppArmor merge across capabilities, filesystem permissions, and network
  rules (including nil vs empty field semantics)

##### Integration tests

- Kubelet integration tests exercising the full flow from pod spec to CRI call
  with OCI profile references
- Credential propagation from imagePullSecrets to CRI AuthConfig
- Feature gate enablement/disablement for the `OCI` type

##### CRI conformance tests (critest)

- Validate that `PullSecurityProfileArtifact` correctly pulls and caches OCI
  artifact profiles, returning the resolved digest
- Validate that the CRI runtime accepts `OCI` profile type in
  `SecurityProfile` messages for both seccomp and AppArmor, referencing a
  previously pulled digest
- Verify that the runtime rejects artifacts with invalid media types
- Verify that the runtime enforces the size limit and rejects oversized
  artifacts
- Verify that the runtime rejects artifacts with multiple layers
- Verify that the `profile_kind` field triggers rejection when the artifact's
  config media type does not match the expected security mechanism
- Verify that the runtime correctly merges an OCI profile with the runtime
  baseline via intersection, producing an effective profile that is at least
  as restrictive as the baseline
- Verify that the runtime correctly merges an OCI profile with a pod-spec
  base profile (Localhost) and the runtime baseline (three-way merge)
- Verify that the runtime returns appropriate errors for unreachable registries
  or invalid credentials

##### e2e tests

- Pull a seccomp profile from an OCI registry and apply it to a container
- Pull an AppArmor profile from an OCI registry and apply it to a container
- Verify that pull failures result in appropriate pod events
- Verify caching behavior (second pod using the same profile starts without
  re-pulling)
- Verify behavior with digest-pinned references
- Verify behavior with invalid or oversized artifacts
- Verify that an OCI profile more permissive than the baseline is silently
  constrained by the merge (effective profile equals the baseline)
- Verify layered profiles: OCI profile merged with a Localhost base profile
  produces the correct intersection

### Graduation Criteria

#### Alpha

- Fix seccomp's `fieldSeccompProfile` to reject unknown profile types with an
  explicit error instead of silently falling through to `Unconfined`
  (prerequisite, not gated by this feature)
- Feature implemented behind `SecurityProfileOCIArtifact` feature gate
- CRI API extended with `OCI` profile type and `PullSecurityProfileArtifact` RPC
- Kubelet calls `PullSecurityProfileArtifact` to pull profiles before
  DRA preparation, then passes the resolved digest to
  `RunPodSandbox`/`CreateContainer`
- Profile merge library ([security-profiles-merger][spm]) implemented with
  seccomp and AppArmor merge support, including architecture-specific syscall
  handling, argument filter intersection, capability intersection, filesystem
  permission intersection, and network rule intersection
- At least one CRI runtime (CRI-O, tracked in [cri-o#10037][crio-10037])
  implements the pull, merge, and apply flow for seccomp, including profile
  merging using the merge library. AppArmor API changes are included but
  runtime implementation may follow in beta.
- PodSecurity admission controller updated to accept the `OCI` profile type
- Initial e2e tests for seccomp OCI artifacts, including profile merge tests
  and layered profile tests

#### Beta

- Feature gate enabled by default, following the standard Kubernetes
  graduation pattern. Promotion to beta requires production support in at
  least one of CRI-O and containerd, and a release candidate available in the
  other.
- Seccomp fallthrough fix (reject unknown profile types) present in the `.0`
  release of the oldest supported kubelet minor version. This is a hard
  prerequisite for enabling the feature gate by default (see
  [Version Skew Strategy](#version-skew-strategy)).
- AppArmor merge support in the profile merge library (already implemented
  in [security-profiles-merger][spm])
- AppArmor OCI artifact support implemented and tested in at least one runtime
- Profile merge library adopted by both CRI-O and containerd
- Profile caching and garbage collection implemented
- Gather feedback from early adopters

#### GA

- At least two releases of beta usage
- Production support in both CRI-O and containerd
- Conformance tests in place
- Documentation published on kubernetes.io
- Media type registrations announced to IETF per [RFC 6838][rfc6838]

#### Deprecation

Not applicable. This KEP introduces a new `OCI` security profile type and does
not deprecate any existing functionality. The `RuntimeDefault`, `Localhost`, and
`Unconfined` profile types remain fully supported.

### Upgrade / Downgrade Strategy

The new `OCI` profile type is additive. Existing `RuntimeDefault`, `Localhost`,
and `Unconfined` profiles continue to work unchanged.

On **upgrade** with the feature gate enabled, pods can start using `OCI`
profile references. No migration is required for existing workloads.

On **downgrade** or feature gate disablement, pods with `OCI` profile
references will fail validation at the API server. Existing pods already running
with OCI profiles will continue running (profiles are applied at container
creation, not enforced continuously by the kubelet). New pods or restarted
containers will fail if they reference OCI profiles.

### Version Skew Strategy

This feature involves coordination between the API server and the kubelet.

- **New API server, old kubelet**: The API server accepts `OCI` profile types.
  An older kubelet that does not understand the `OCI` type will handle it
  differently depending on the security mechanism. **AppArmor** validation
  errors on unknown types and fails container creation (fail-closed).
  **Seccomp**, however, has a code path in `fieldSeccompProfile` that silently
  falls through to `Unconfined` for unrecognized types. This means an old
  kubelet would run OCI-referenced containers without any seccomp profile
  applied, which is a security gap. A prerequisite fix to the seccomp code path
  is required before this feature ships: the kubelet's seccomp handler must be
  updated to reject unknown profile types with an explicit error rather than
  falling through to `Unconfined`. This fix will be included in the same
  release as the alpha feature gate. **Beta promotion (feature gate on by
  default) requires that the seccomp fallthrough fix is present in the `.0`
  release of the oldest supported kubelet minor version.** Since Kubernetes
  does not require kubelet upgrades before API server upgrades, a cluster
  could be running
  kubelet `1.y-3.0` (the very first patch of the oldest supported minor
  version). A backport released in a later patch (e.g., `1.y-3.9`) does not
  help because that kubelet was never required to be updated. Practically,
  if the fix lands in `1.X.0`, beta can happen earliest when `1.X` is the
  oldest supported kubelet version (API server at `1.X+3`). This is a hard
  prerequisite, not a best-effort backport. The scheduler should avoid
  placing pods with OCI profiles on nodes with old kubelets;
  [Node Declared Features (KEP-5328)][ndf] can be used for this. If a pod
  lands on an unpatched old kubelet, the seccomp risk is that the container
  runs unconfined rather than failing.
- **Old API server, new kubelet**: The API server rejects `OCI` profile types
  at validation. Pods using this feature cannot be created. This is safe.
- **CRI version skew**: If the kubelet calls `PullSecurityProfileArtifact` on
  a CRI runtime that does not support the RPC, the runtime returns
  `Unimplemented`. The kubelet treats this as a terminal failure and marks the
  pod as failed (see [Kubelet Behavior](#kubelet-behavior)). This fails
  earlier and more clearly than if the `OCI` profile type were rejected inside
  `RunPodSandbox`/`CreateContainer`.

To improve scheduling in mixed clusters where some nodes support OCI security
profiles and others do not, [Node Declared Features (KEP-5328)][ndf] can be
used. Nodes can declare CRI runtime capabilities (including `OCI` profile type
support) as node features, enabling `nodeSelector` or `nodeAffinity` rules to
schedule pods with OCI profiles only on capable nodes. Runtime features
reported via the CRI `StatusRequest` can also signal OCI profile support,
complementing node declared features. Defining a standard declared feature for
this capability is out of scope for this KEP but is a natural follow-up.

Scheduler awareness of a node's baseline profile configuration is another
potential optimization: the scheduler could avoid placing pods on nodes whose
baseline profile is incompatible with the referenced OCI profile. This could
be expressed via node declared features or node labels derived from the
runtime configuration. This is out of scope for alpha but is a natural
extension of the node declared features integration.

[ndf]: https://github.com/kubernetes/enhancements/issues/5328

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
baseline via the runtime's configuration.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate causes the API server to reject new pods with
`OCI` profile references. Already-running pods continue running. On container
restart, pods with OCI profiles will fail because the kubelet rejects the
unrecognized `OCI` profile type with a clear error event (fail-closed),
provided the prerequisite fix to seccomp's `fieldSeccompProfile` fallthrough
has been applied (see [Version Skew Strategy](#version-skew-strategy)).

###### What happens if we reenable the feature if it was previously rolled back?

Pods with `OCI` profile references in their spec (created while the feature was
enabled, still present in etcd) will work again on their next container
creation. No data migration is needed.

###### Are there any tests for feature enablement/disablement?

Unit tests will verify that API validation correctly rejects/accepts `OCI`
profile types based on the feature gate state.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout cannot impact already running workloads because the feature is
opt-in per pod. A rollback (disabling the feature gate) will prevent new pods
with OCI profiles from starting but does not affect running pods.

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
- API: Pods with `seccompProfile.type: OCI` or `appArmorProfile.type: OCI`
  can be queried directly.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: `SecurityProfilePulled` (successful pull),
    `SecurityProfilePullFailed` (failed pull)
- [x] API .status
  - Other field: `containerStatuses[*].seccompProfileArtifactDigest` and
    `appArmorProfileArtifactDigest` report the resolved digest for each OCI
    profile, confirming which content was applied. Pull failures surface as
    error messages in container status.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The design target is that profile pulls should not add more than 2 seconds to
pod startup time (p99) when the profile is not cached and the registry is
reachable. This is not enforced by the kubelet but serves as a target for CRI
runtime implementations. Cached profile lookups should add less than 10 ms.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kubelet_security_profile_artifact_pull_duration_seconds`
  - Aggregation method: histogram
  - Components exposing the metric: kubelet
- [x] Metrics
  - Metric name: `kubelet_security_profile_artifact_pull_errors_total`
  - Aggregation method: counter
  - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A metric for cache hit/miss ratio would be useful for tuning cache
configuration but is not required for alpha. The kubelet emits pull duration
and error metrics from its `PullSecurityProfileArtifact` calls, which are
collected from the kubelet's metrics endpoint (not via the summary API). CRI
runtimes may also emit their own pull metrics at the runtime level; these
would need to be collected from the runtime's metrics endpoint directly.

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
(`github.com/saschagrunert/security-profiles-merger`) used by CRI runtimes
to merge OCI profiles with baseline profiles via intersection. It handles
profile-type-specific merge logic for both seccomp (including
architecture-specific syscall tables and argument filters) and AppArmor
(capabilities, filesystem permissions, network rules), with the interface
designed to accommodate landlock. CRI runtimes that do not integrate the
library cannot perform profile merging and cannot support the `OCI` profile
type. See [Graduation Criteria](#graduation-criteria) for phasing.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new Kubernetes API calls. The feature adds a `PullSecurityProfileArtifact`
CRI RPC and OCI registry pulls at the CRI runtime level, which are external to
the Kubernetes API.

###### Will enabling / using this feature result in introducing new API types?

No new API types. Existing `SeccompProfile` and `AppArmorProfile` types are
extended with new fields.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No, unless the OCI registry is a cloud-provider-managed registry (e.g., ECR,
GCR, ACR). In that case, credential provider plugins may make additional
calls, but this is the same mechanism used for container image pulls.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type: Pod
- Estimated increase in size: ~100 bytes per OCI profile reference in the spec
  (`oci` field), plus ~100 bytes per resolved digest in container status
  (`seccompProfileArtifactDigest` and/or `appArmorProfileArtifactDigest`
  fields per container).

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Pod startup latency may increase for pods using OCI profile references when
profiles are not cached. For uncached pulls, the expected latency is comparable
to small image pulls (under 2 seconds for typical profiles under 100 KB on
reasonable network conditions). This is not enforced by the kubelet but serves
as a design target for CRI runtime implementations. Cached pulls add negligible
latency.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

- Disk: Cached profiles consume disk space on the node. Profiles are small
  (typically < 100 KB). With a 1 MiB limit and reasonable caching, disk usage
  is negligible.
- Network: Profile pulls add network traffic, but profiles are much smaller
  than container images.
- Memory: The CRI runtime holds parsed profiles in memory. This is already the
  case for `Localhost` profiles.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Cached profiles consume inodes. With the default size limit (1 MiB) and
typical profile sizes, this is not a concern. The CRI runtime's garbage
collection should clean up unused cached profiles.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

This feature does not interact with the API server or etcd at runtime. Profile
pulls happen between the CRI runtime and the OCI registry. API server
unavailability does not affect already-scheduled pods.

###### What are other known failure modes?

- Registry unreachable or authentication failure
  - Detection: `kubelet_security_profile_artifact_pull_errors_total` metric
    increases. Pod events show `SecurityProfilePullFailed`.
  - Mitigations: Use digest-pinned references and ensure profiles are cached.
    Configure multiple registry mirrors.
  - Diagnostics: CRI runtime logs show pull attempts and errors.
  - Testing: e2e tests with unreachable registry endpoints.

- Invalid or corrupt profile content
  - Detection: Pod events show profile validation errors.
  - Mitigations: Use digest-pinned references to ensure immutability.
    Implement signature verification.
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

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check `kubelet_security_profile_artifact_pull_errors_total` for pull failures.
2. Check `kubelet_security_profile_artifact_pull_duration_seconds` for latency.
3. Verify registry connectivity from the node.
4. Check CRI runtime logs for credential resolution or pull issues.
5. Verify the artifact exists and has the correct media type.

## Implementation History

- 2026-05-06: Initial KEP draft

## Drawbacks

- **Adds complexity to the CRI API**: A new profile type, RPC
  (`PullSecurityProfileArtifact`), and message types increase the CRI surface
  area. However, the pattern mirrors the existing `PullImage`/`CreateContainer`
  separation and follows established conventions.
- **Registry dependency for pod startup**: Pods using OCI profiles cannot start
  if the registry is unreachable and profiles are not cached. This is the same
  trade-off that exists for container images.
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
- `PullImageRequest` includes image-specific metadata fields that do not
  apply to profiles. `PullImageResponse` returns an `image_ref` string, while
  profile pulls need to return a resolved digest.
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

### Subset Validation Without Merge

An earlier iteration of this KEP used subset-and-reject semantics: the CRI
runtime validated that every pulled OCI profile was equally or more restrictive
than the node's baseline profile and rejected profiles that exceeded it.
Profiles were validated at pull time (`PullSecurityProfileArtifact`), and pods
with over-permissive profiles failed to start with a
`SecurityProfileSubsetValidationFailed` event. There was no layered profile
model; each container had exactly one profile type.

This approach was replaced with merge (intersection) semantics for two reasons:

- **Operational fragility**: With subset-and-reject, tightening the baseline
  profile breaks all existing OCI profiles that do not already include the new
  restriction. For example, if the admin adds a new syscall to the deny list
  in the baseline, every OCI profile that allows that syscall suddenly fails
  validation. The admin must update every OCI profile in the registry before
  changing the baseline. With merge semantics, the admin simply tightens the
  baseline and the intersection automatically applies the new restriction.
  Existing OCI profiles continue to work.
- **Layered profile support**: The subset-and-reject model only supports a
  single profile per container. Merge semantics naturally support layered
  profiles where a pod specifies both a base profile (Localhost or
  RuntimeDefault) and an OCI overlay. The intersection produces an effective
  profile that combines restrictions from all inputs. This enables
  per-workload-class baselines (for example, a restrictive baseline for
  agentic workloads) without node-pool segregation, while preserving the node
  admin's control through the runtime-configured baseline floor.

### Kubelet Allowlist for OCI Profile Sources

An earlier iteration added a kubelet-level allowlist
(`securityProfileOCIArtifact.allowedRegistries` in `KubeletConfiguration`)
that restricted which registries OCI profiles could be pulled from. The
allowlist was deny-by-default: when empty, no OCI profiles were allowed.

This approach was replaced because it provides source-based trust rather than
content-based trust. The allowlist controls where a profile comes from but does
not guarantee what the profile does. A compromised allowed registry could serve
a profile that is more permissive than intended. Profile merging provides a
stronger guarantee: the runtime constrains each profile's effective permissions
to the baseline regardless of source. Additionally, the allowlist required
per-node configuration (similar to placing files on disk for Localhost
profiles), which does not improve operational burden over existing approaches.

## Infrastructure Needed (Optional)

- An OCI-compatible registry for e2e tests (can use the existing test
  infrastructure registry or a local registry).
- Sample security profile artifacts pushed to a test registry for e2e testing.
