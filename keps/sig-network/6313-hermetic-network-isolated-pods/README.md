# KEP-6313: Hermetic Pods: network-isolated pod sandboxes

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: DRA-provisioned accelerator fabric](#story-1-dra-provisioned-accelerator-fabric)
    - [Story 2: Disconnected batch job](#story-2-disconnected-batch-job)
    - [Story 3: VM sandboxes and device passthrough](#story-3-vm-sandboxes-and-device-passthrough)
    - [Story 4: Local-only multi-container pods](#story-4-local-only-multi-container-pods)
    - [Story 5: Agentic orchestrators](#story-5-agentic-orchestrators)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Defaulting and <code>hostNetwork</code> mirroring](#defaulting-and-hostnetwork-mirroring)
  - [Validation](#validation)
  - [CRI Integration](#cri-integration)
  - [Kubelet Behavior](#kubelet-behavior)
  - [Control Plane Behavior](#control-plane-behavior)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
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
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP adds **Hermetic (Network Isolated) Pods**, Pods that opt out of the cluster network.
A hermetic pod gets a network namespace with only a loopback interface and
is not reachable from the cluster.

## Motivation

Kubernetes was built on the early assumption that every Pod is attached to a
single flat cluster network and is routable from every other Pod. This
assumption has always generated friction with workloads that need advanced
routing, multi-tenancy or meshes. Those workloads get around the "default
network" with out-of-band mechanisms: per-node runtime configuration, custom
runtime classes, chained plugins, or annotations.

SIG Network has already tried to solve this inside the project. The
[Multi-Network effort (KEP-3698)] concluded it was too complex: problems like
network probes or DNS across multiple networks had no clear solution, portability
depended on very specific (often private) plugin implementations, and the
user stories were fragmented — some wanted multiple NICs, others VPC-like
networks — each backed by a relatively small number of users. A follow-up
proposal to simplify the low level API ([KEP-4410]) required offloading the
network to another component that will make it incompatible with existing
deployments and hard to roll out.

What has worked is composing on top of stable core primitives instead of
extending the core network model:

- [Kubernetes Network Drivers][knd-paper] defined a new architecture, based
  on DRA, to provide a native mechanism to plumb additional network devices.
  [DRANET] is a project that demonstrated the successful use of this
  architecture to solve the challenges of the new AI/ML workloads and RDMA
  networks.
- [KEP-4962] (GA) added a standard way to report network data in DRA. All
  drivers report their network devices with a common language, and
  integrators can use that data to build network integrations out of core
  (for example, their own EndpointSlice controllers), keeping core stable.
- [NRI] showed how network applications can integrate directly with the
  runtime. This removes the limitations of the CNI model, which composes
  plugins only through serial Unix-style chaining and file system based configuration
  that breaks the separation of concerns (see the [service mesh CNI discussion]).

This friction is more critical these days. New workloads, like the agentic
ones, require strict isolation, and their density and scale make the traditional
out-of-band mechanisms obsolete. At the same time, DRA network drivers, VM sandboxes
that provide their own networking (e.g. Kata/Firecracker/... with VFIO passthrough),
disconnected jobs, ... have been waiting for a solution for years.

The missing piece is a core primitive to NOT attach the default network.
Decomposing the Pod network attachment solves this problem for both
maintainers and integrators: integrators get an isolated sandbox they can
build on with the mechanisms above, and the project stops carrying the toil
of workarounds and requirements that the core network model was never meant
to express.

[Multi-Network effort (KEP-3698)]: https://github.com/kubernetes/enhancements/issues/3698
[KEP-4410]: https://github.com/kubernetes/enhancements/issues/4410
[knd-paper]: https://ieeexplore.ieee.org/document/11146291/
[DRANET]: https://github.com/google/dranet
[KEP-4962]: https://github.com/kubernetes/enhancements/issues/4962
[NRI]: https://github.com/containerd/nri
[service mesh CNI discussion]: https://github.com/kubernetes/kubernetes/issues/130594

### Goals

- Allow users to declare that a Pod must not be attached to the cluster
  network.
- Provide an API that is backwards compatible with the current network model.
- Fail predictably: reject network-dependent features (cluster DNS, service
  links, network probes, host ports) at admission time instead of failing at
  runtime, and never run an isolated pod attached to the network on nodes
  that do not support the feature.
- Define the contract between the kubelet and the container runtimes.
- Keep isolated pods out of service discovery: Services, Endpoints,
  EndpointSlices and service environment variables.

### Non-Goals

- Designing a multi-network or secondary-interface attachment framework.
  `networkMode` is intentionally about the absence of automatic attachment;
  it provides a clean surface for delegation (e.g., DRA drivers), but how
  interfaces are attached afterwards and its lifecycle is out of scope.
- Isolating anything other than networking. Storage, IPC, PID, devices and
  host access are governed by existing fields.
- Deprecating or removing `hostNetwork`. It is a GA v1 field and remains
  supported forever; `networkMode: Host` is its enum alias.
- Changing how kube-proxy, NetworkPolicy or DNS work for non-hermetic pods.
- Skipping service account token mounting for hermetic pods. Tokens may still
  be consumed by non-network means.

## Proposal

Add a new optional enum field to `PodSpec`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hermetic-example
spec:
  networkMode: None
  containers:
  - name: worker
    image: registry.k8s.io/e2e-test-images/agnhost:2.53
    command: ["sleep", "3600"]
```

Semantics of `networkMode: None`:

1. The API server defaults `dnsPolicy` to `None` (without requiring
   `dnsConfig.nameservers`) and `enableServiceLinks` to `false`, and validates
   that no network-dependent feature is requested (see [Validation](#validation)).
2. The kubelet uses a new CRI field to tell the runtime to create the sandbox
   network namespace with only the loopback interface and to skip all the
   network plumbing (CNI is never invoked on Linux runtimes).
3. The pod runs with empty `status.podIP`/`status.podIPs` and becomes `Ready`
   without waiting for an IP. `status.hostIP` is still reported.
4. The endpoints and endpointslice controllers never select the pod into
   Endpoints/EndpointSlices, even if a Service selector matches it.
5. The kubelet does not inject the `KUBERNETES_SERVICE_HOST`/`KUBERNETES_PORT`
   environment variables. It also maps the pod hostname to `127.0.0.1` and
   `::1` in the managed `/etc/hosts`, so `localhost` and `$(hostname)`
   resolve inside the sandbox.
6. If the container runtime does not report support for isolated sandboxes,
   the kubelet rejects the pod at admission (**fail closed**) instead of
   silently attaching it to the cluster network.

`networkMode: Host` and `networkMode: Cluster` do not change any behavior:
`Host` is the explicit form of `hostNetwork: true` and `Cluster` is the
current default.

### User Stories

#### Story 1: DRA-provisioned accelerator fabric

As an operator of an AI training cluster, I allocate RDMA/accelerator NICs to
pods through DRA drivers. I want the pod sandbox to start empty: no
interfaces, routes or iptables rules from the default network plugin. The DRA
driver is then the only owner of the pod connectivity. Today I have to
coordinate teardown and priority with the CNI configuration on every node.

#### Story 2: Disconnected batch job

As a security engineer, I run jobs that process sensitive data mounted from
volumes, and I must prove to auditors that the workload has no network path
in or out. A deny-all NetworkPolicy depends on the network plugin enforcing
it, and the pod still gets a routable IP. With `networkMode: None` there is
nothing to enforce: the only interface is loopback.

#### Story 3: VM sandboxes and device passthrough

As a confidential-computing platform owner using VM sandboxes, networking
is provided inside the VM (e.g., via VFIO passthrough of a physical function).
The node-level network plumbing is useless and potentially harmful. I want
the runtime to skip it entirely, driven by a core API field rather
than a runtime-specific annotation.

#### Story 4: Local-only multi-container pods

As a CI system author, I run pods whose containers only talk over `localhost`
(hermetic builds and tests). I want to guarantee that no external network
dependency can leak into the build, and I do not want these pods to consume
cluster IP space or appear in service discovery.

#### Story 5: Agentic orchestrators

As an operator of an agent orchestrator (e.g. Substrate), I run untrusted,
often machine-generated code in sandboxed pods at very high density and
churn. Each agent must start with zero network access by default; the
orchestrator then grants connectivity explicitly through its own channels
(a local proxy over `localhost`, or interfaces injected by a driver). I
cannot depend on NetworkPolicy enforcement for this guarantee, and at this
scale the per-pod network plumbing and IPAM are too much overhead: the pods never
use the cluster network, but they still consume IP space, endpoints
processing, and setup/teardown time on every sandbox.

### Notes/Constraints/Caveats

**Naming.** This was extensively discussed during the initial reviews
([kubernetes/kubernetes#141604]):

- "Hermetic" is only the colloquial name of the feature. It describes the
  contract well: a sealed environment with no external network attachments,
  like hermetic builds in Bazel or Nix. It is deliberately **not** an API
  value: reviewers pointed out that "hermetic" can also imply storage, host
  access or compute isolation, and this KEP does not provide those.
- "CNI" never appears in the API. CNI is an implementation detail of Linux
  container runtimes; it does not exist in the Kubernetes API vocabulary.
- `None` follows existing ecosystem semantics (`docker run --network
  none`) and the existing `dnsPolicy: None`.

**The pod still has a network namespace.** The pod keeps loopback
(`127.0.0.1`, `::1`) and the containers in the pod communicate over
`localhost`. This is why names like `noNetwork` were rejected: they are
technically inaccurate.

**Kubernetes API access.** A hermetic pod cannot reach the kube-apiserver (or
anything else). The documentation in [services-networking concepts] assumes
that every pod can reach the API and other pods. Those documents must be
updated to describe `networkMode: None` as an explicit, opt-in exception.
Pods that do not set the field keep all the documented guarantees.

**Downward API.** `status.podIP`/`status.podIPs` resolve to empty values for
hermetic pods. `status.hostIP` remains populated.

[kubernetes/kubernetes#141604]: https://github.com/kubernetes/kubernetes/pull/141604
[services-networking concepts]: https://kubernetes.io/docs/concepts/services-networking/

### Risks and Mitigations

**Fail-open on version skew (security risk).** The scheduler keeps hermetic
pods away from unsupported nodes using [Node Declared Features (KEP-5328)],
but pods can bypass the scheduler (`spec.nodeName`, DaemonSets, static pods).
An old runtime or kubelet that does not understand the field could silently
attach an isolated pod to the cluster network. The kubelet **fails closed**
instead: if `networkMode` is `None` and the runtime does not report
support for isolated sandboxes (via CRI `RuntimeFeatures`), the kubelet
rejects the pod with `PodAdmitFailed`. See
[Version Skew Strategy](#version-skew-strategy).

[Node Declared Features (KEP-5328)]: https://github.com/kubernetes/enhancements/issues/5328

**Ecosystem assumptions that every pod has an IP.** Controllers, meshes and
operators that read `status.podIP` will see an empty value. This is the same
state as a pod whose sandbox is still being created, so well-behaved
consumers already tolerate it. The endpoints controllers get an explicit
check anyway.

**API confusion with `hostNetwork`.** Two fields describing the same thing is
not great, but `hostNetwork` is v1 and cannot go away. Defaulting keeps both in
sync and validation rejects conflicts; a reader of either field gets the
same answer.

**Scope creep towards multi-network.** An enum invites new values
(`bridge`-like modes, named networks). Not in this KEP. Any future value
needs its own KEP and cannot weaken the conformance guarantees of the
default mode.

## Design Details

### API Changes

New enum type and `PodSpec` field:

```go
// PodNetworkMode describes how the pod sandbox network is provisioned.
// +enum
type PodNetworkMode string

const (
	// PodNetworkModeCluster attaches the pod to the cluster-wide pod
	// network. The container runtime performs its configured network
	// plumbing and the pod is assigned routable pod IPs. This is the
	// default and matches the historical behavior of Kubernetes.
	PodNetworkModeCluster PodNetworkMode = "Cluster"

	// PodNetworkModeHost shares the host's network namespace with the pod.
	// It is equivalent to setting hostNetwork: true; the two fields are
	// kept in sync by API defaulting.
	PodNetworkModeHost PodNetworkMode = "Host"

	// PodNetworkModeNone provides the pod with a dedicated network
	// namespace containing only the loopback interface. The container
	// runtime performs no network plumbing and the pod is assigned no IP.
	// The pod is never selected into Services and receives no cluster DNS
	// configuration or service environment variables.
	PodNetworkModeNone PodNetworkMode = "None"
)
```

```go
type PodSpec struct {
	// ...

	// networkMode selects how the pod sandbox network is provisioned.
	// "Cluster" (default) attaches the pod to the cluster network,
	// "Host" shares the host network namespace (equivalent to
	// hostNetwork: true and kept in sync with it), and "None" gives the
	// pod an isolated network namespace with only a loopback interface
	// and no automatic network plumbing.
	// When "None" is selected, dnsPolicy must be "None",
	// enableServiceLinks must be false, and probes, lifecycle handlers
	// and hostPorts that require networking are forbidden.
	// This field is immutable.
	// +featureGate=PodNetworkMode
	// +optional
	NetworkMode *PodNetworkMode `json:"networkMode,omitempty" protobuf:"bytes,45,opt,name=networkMode,casttype=PodNetworkMode"`
}
```

**Why an enum and not a boolean.** The first proposal was `hermetic *bool`
(see [Alternatives](#alternatives)). It got a lot of pushback and it makes
it hard to think about in conjunction with `hostNetwork`, since they will be
two mutually exclusive booleans that will need cross-field validation
in both directions (`if hermetic { hostNetwork must be false }`, and the
reverse). The [API conventions] discourage exactly this. With a single enum
a pod has exactly one network mode, the invalid combinations cannot be
written, and `hostNetwork` is absorbed into it as the `Host` value.

[API conventions]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#primitive-types

### Defaulting and `hostNetwork` mirroring

`hostNetwork` is a GA v1 field and cannot be removed. Mirroring an enum onto
a legacy boolean is the best we can do without breaking v1 compatibility. Defaulting
in `pkg/apis/core/v1/defaults.go` keeps both fields in sync, for Pods and
for every workload `PodTemplateSpec`:

```go
func SetDefaults_PodSpec(obj *v1.PodSpec) {
	// NetworkMode <-> HostNetwork mirroring.
	if utilfeature.DefaultFeatureGate.Enabled(features.PodNetworkMode) {
		if obj.NetworkMode == nil {
			// Old clients: derive the mode from the legacy field.
			if obj.HostNetwork {
				obj.NetworkMode = ptr.To(v1.PodNetworkModeHost)
			} else {
				obj.NetworkMode = ptr.To(v1.PodNetworkModeCluster)
			}
		} else if *obj.NetworkMode == v1.PodNetworkModeHost {
			// New clients: keep the legacy field in sync for older
			// controllers and clients that read hostNetwork directly.
			obj.HostNetwork = true
		}
	}

	if obj.DNSPolicy == "" {
		if obj.NetworkMode != nil && *obj.NetworkMode == v1.PodNetworkModeNone {
			obj.DNSPolicy = v1.DNSNone
		} else {
			obj.DNSPolicy = v1.DNSClusterFirst
		}
	}
	if obj.NetworkMode != nil && *obj.NetworkMode == v1.PodNetworkModeNone &&
		obj.EnableServiceLinks == nil {
		obj.EnableServiceLinks = ptr.To(false)
	}
	// ... existing defaulting ...
}
```

Properties of this strategy:

- An old client submitting `hostNetwork: true` observes `networkMode: Host`
  after defaulting; behavior is unchanged.
- A new client submitting `networkMode: Host` produces `hostNetwork: true` in
  the stored object, so an n-1 controller reading only the raw boolean makes
  the correct decision.
- A client cannot send the conflict "`networkMode: Host` with
  `hostNetwork: false`". `hostNetwork` is a plain boolean, so the API server
  cannot tell an explicit `false` apart from an unset field. In both cases
  defaulting sets `hostNetwork` to `true` to match `networkMode: Host`.
- The remaining conflicts (`networkMode: Cluster` or `None` combined with an
  explicit `hostNetwork: true`) are rejected by validation.
- When the `PodNetworkMode` feature gate is disabled, the field is stripped
  from new objects (standard `dropDisabledFields` handling), no mirroring
  occurs, and behavior is exactly as today. Objects that already carry the
  field keep it, following the API compatibility rules for disabled gates.

### Validation

Implemented in `pkg/apis/core/validation/validation.go` and invoked from
`ValidatePodSpec` (covering Pods and all workload templates):

```go
// validatePodNetworkMode validates spec.networkMode and its interactions
// with network-dependent fields.
func validatePodNetworkMode(spec *core.PodSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if spec.NetworkMode == nil {
		return allErrs
	}
	modePath := fldPath.Child("networkMode")

	switch *spec.NetworkMode {
	case core.PodNetworkModeCluster:
		if spec.HostNetwork {
			allErrs = append(allErrs, field.Invalid(modePath, *spec.NetworkMode,
				`must be "Host" when hostNetwork is true`))
		}
	case core.PodNetworkModeHost:
		if !spec.HostNetwork {
			// Defaulting keeps these in sync; reaching this branch means
			// an internal client bypassed defaulting.
			allErrs = append(allErrs, field.Invalid(modePath, *spec.NetworkMode,
				"hostNetwork must be true when networkMode is \"Host\""))
		}
	case core.PodNetworkModeNone:
		allErrs = append(allErrs, validateIsolatedPodSpec(spec, fldPath)...)
	default:
		allErrs = append(allErrs, field.NotSupported(modePath, *spec.NetworkMode,
			[]core.PodNetworkMode{core.PodNetworkModeCluster, core.PodNetworkModeHost, core.PodNetworkModeNone}))
	}
	return allErrs
}

// validateIsolatedPodSpec enforces that a networkMode "None" pod requests no
// feature that depends on pod or cluster networking.
func validateIsolatedPodSpec(spec *core.PodSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if spec.HostNetwork {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("networkMode"),
			core.PodNetworkModeNone, "must not be \"None\" when hostNetwork is true"))
	}
	if spec.DNSPolicy != core.DNSNone {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("dnsPolicy"),
			spec.DNSPolicy, `must be "None" when networkMode is "None"`))
	}
	if spec.EnableServiceLinks == nil || *spec.EnableServiceLinks {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("enableServiceLinks"),
			spec.EnableServiceLinks, `must be false when networkMode is "None"`))
	}

	podshelper.VisitContainersWithPath(spec, fldPath, func(c *core.Container, cFldPath *field.Path) bool {
		// Network probes (httpGet, tcpSocket, grpc) cannot succeed without a
		// network interface; only exec probes are permitted.
		checkProbe := func(probe *core.Probe, probePath *field.Path) {
			if probe == nil {
				return
			}
			if probe.HTTPGet != nil {
				allErrs = append(allErrs, field.Forbidden(probePath.Child("httpGet"),
					`may not be set when networkMode is "None"`))
			}
			if probe.TCPSocket != nil {
				allErrs = append(allErrs, field.Forbidden(probePath.Child("tcpSocket"),
					`may not be set when networkMode is "None"`))
			}
			if probe.GRPC != nil {
				allErrs = append(allErrs, field.Forbidden(probePath.Child("grpc"),
					`may not be set when networkMode is "None"`))
			}
		}
		checkProbe(c.LivenessProbe, cFldPath.Child("livenessProbe"))
		checkProbe(c.ReadinessProbe, cFldPath.Child("readinessProbe"))
		checkProbe(c.StartupProbe, cFldPath.Child("startupProbe"))

		if c.Lifecycle != nil {
			checkHandler := func(h *core.LifecycleHandler, hPath *field.Path) {
				if h == nil {
					return
				}
				if h.HTTPGet != nil {
					allErrs = append(allErrs, field.Forbidden(hPath.Child("httpGet"),
						`may not be set when networkMode is "None"`))
				}
				if h.TCPSocket != nil {
					allErrs = append(allErrs, field.Forbidden(hPath.Child("tcpSocket"),
						`may not be set when networkMode is "None"`))
				}
			}
			checkHandler(c.Lifecycle.PostStart, cFldPath.Child("lifecycle", "postStart"))
			checkHandler(c.Lifecycle.PreStop, cFldPath.Child("lifecycle", "preStop"))
		}

		// hostPort mappings require an attached network interface.
		portsPath := cFldPath.Child("ports")
		for i, port := range c.Ports {
			if port.HostPort > 0 {
				allErrs = append(allErrs, field.Forbidden(portsPath.Index(i).Child("hostPort"),
					`may not be set when networkMode is "None"`))
			}
		}
		return true
	})

	return allErrs
}
```

Additional validation adjustments:

- Only `exec` probes and `exec` lifecycle handlers pass validation.
  `httpGet`, `tcpSocket` and `grpc` are rejected at admission.
- Today, `dnsPolicy: None` requires the user to provide
  `dnsConfig.nameservers`. Pods with `networkMode: None` are an exception: they can
  have no nameservers at all. The kubelet then writes an empty
  `/etc/resolv.conf`, unless the user provides a `dnsConfig` (for example,
  for a resolver listening on loopback).
- `networkMode` cannot be changed after the pod is created, like the other
  fields that define the sandbox.

### CRI Integration

A new typed field in `PodSandboxConfig` (`k8s.io/cri-api`) carries the
sandbox network mode to the runtime:

```proto
enum PodSandboxNetworkMode {
    // POD attaches the sandbox to the pod network. The runtime performs its
    // configured network plumbing (e.g., invokes CNI plugins on Linux).
    // This is the default and matches historical behavior.
    POD = 0;
    // NONE creates the sandbox network namespace with only the loopback
    // interface configured and skips all network plumbing. The runtime MUST
    // NOT attach any external interface and MUST report an empty IP in
    // PodSandboxStatus.
    NONE = 1;
}

message PodSandboxConfig {
    // ... existing fields ...

    // network_mode instructs the runtime how to provision networking for
    // the sandbox network namespace. Host-network sandboxes continue to be
    // requested via NamespaceOption (NamespaceMode NODE) and use POD here.
    PodSandboxNetworkMode network_mode = 10;
}
```

Capability discovery, so the kubelet can fail closed:

```proto
message RuntimeFeatures {
    // ... existing fields ...

    // network_mode_none is set to true if the runtime supports creating
    // sandboxes with PodSandboxNetworkMode NONE.
    bool network_mode_none = 4;
}
```

Runtime obligations for `NONE` sandboxes:

- create the netns according to `NamespaceOption` as usual;
- bring up `lo` (`127.0.0.1/8`, `::1/128`), the equivalent of
  `ip link set lo up`;
- no plumbing on setup and none on teardown;
- report no IPs in `PodSandboxStatus.network`.

A reference containerd implementation exists ([aojea/containerd `cniless`
branch]). Upstream containerd and CRI-O implementations are a beta graduation
requirement.

[aojea/containerd `cniless` branch]: https://github.com/aojea/containerd/tree/cniless

### Kubelet Behavior

- Admission is fail-closed. If the pod has `networkMode: None` and the
  runtime does not report `network_mode_none` in `RuntimeFeatures`, the
  kubelet rejects the pod with `PodAdmitFailed`, reason
  `UnsupportedNetworkMode`. Same rejection if the kubelet's `PodNetworkMode`
  gate is off. Attaching an intentionally isolated pod to the cluster
  network is worse than not running it.
- `generatePodSandboxConfig` sets `network_mode: NONE`.
  `determinePodSandboxIPs` returns no IPs, and `PodSandboxChanged` must not
  recreate the sandbox because the IP is missing.
- `status.podIP`/`status.podIPs` stay empty. Readiness is computed from
  container state only. `status.hostIP` is reported as usual.
- The managed `/etc/hosts` is still mounted, with the pod hostname (and
  `hostname.subdomain` if set) mapped to `127.0.0.1` and `::1` to avoid
  breaking applications that depend on that.
- No `KUBERNETES_SERVICE_HOST`/`KUBERNETES_PORT`, no service links
  (`enableServiceLinks` is already forced to `false`). The kubelet also
  skips the service-informer sync wait for these pods: one less cluster
  dependency in the startup path, and it shows in startup latency.
- `dnsPolicy: None` with no `dnsConfig` produces an empty `resolv.conf`. A
  user-supplied `dnsConfig` is honored: a resolver on loopback inside the
  pod, or nameservers reachable over driver-injected interfaces.

### Control Plane Behavior

- `ShouldPodBeInEndpoints` (in `k8s.io/endpointslice/util`, shared by the
  Endpoints and EndpointSlice controllers) returns `false` for
  `networkMode: None` pods, so they are never published even if a Service
  selector matches.
- The kubelet publishes support for `networkMode: None` through
  [Node Declared Features (KEP-5328)] (GA since v1.37), so the scheduler
  keeps hermetic pods away from nodes without a supporting kubelet and
  runtime. This KEP only registers a new feature in that existing mechanism.
  The kubelet fail-closed admission remains the final guarantee for pods
  that bypass the scheduler.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None identified.

##### Unit tests

Planned coverage:

- `pkg/apis/core/validation`: `networkMode` cross-field validation (all
  enum values × `hostNetwork` × `dnsPolicy` × `enableServiceLinks` × probes ×
  lifecycle handlers × `hostPort`), feature gate on/off, update immutability.
- `pkg/apis/core/v1`: defaulting of `networkMode`, `hostNetwork` mirroring in
  both directions, `dnsPolicy`/`enableServiceLinks` defaulting for `None`,
  across Pod and all workload template kinds.
- `pkg/api/pod`: `dropDisabledFields` behavior with the gate disabled.
- `pkg/kubelet/kuberuntime`: sandbox config generation, IP determination and
  `PodSandboxChanged` for `NONE` sandboxes; admission rejection on
  unsupported runtimes.
- `pkg/kubelet`: managed hosts file content and environment variable
  construction for hermetic pods.
- `pkg/kubelet/status`: readiness generation without pod IPs.
- `staging/src/k8s.io/endpointslice/util`: `ShouldPodBeInEndpoints`.
- `pkg/controller/endpoint`, `pkg/controller/endpointslice`: exclusion of
  hermetic pods.

- `pkg/apis/core/validation`: `2026-09-02` - `84.9%`
- `pkg/apis/core/v1`: `2026-09-02` - `81.2%`
- `pkg/kubelet/kuberuntime`: `2026-09-02` - `68.3%`

##### Integration tests

- `test/integration/pods` (`TestPodNetworkMode*`): create/update of Pods and of Deployment,
  StatefulSet, DaemonSet, Job and CronJob templates using every enum value;
  verification of defaulting, mirroring, validation rejections (including the
  explicit `networkMode: None` + `hostNetwork: true` conflict) and behavior
  with the feature gate disabled.

- [test name](https://github.com/kubernetes/kubernetes/blob/master/test/integration/pods/pods_test.go): links to [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=PodNetworkMode) and [triage](https://storage.googleapis.com/k8s-triage/index.html?test=PodNetworkMode) to be added once merged.

##### e2e tests

- `test/e2e/network/hermetic.go` (cluster e2e, `Feature:PodNetworkMode`,
  requires a runtime with support): pod runs and becomes `Ready` with empty
  `status.podIP`; only `lo` exists in the sandbox; hostname resolves to
  loopback via `/etc/hosts`; no `KUBERNETES_SERVICE_HOST`; downward API
  `status.podIP` is empty; pod is excluded from EndpointSlices of a matching
  Service.
- `test/e2e_node/hermetic_test.go` (node e2e): sandbox creation without
  network plumbing, fail-closed admission against a runtime without support,
  kubelet restart idempotency (no sandbox recreation due to missing IP).
- Until upstream runtime releases exist, CI runs against kind with the
  patched containerd reference implementation; a periodic job will be added
  under sig-network.

- [test name](https://github.com/kubernetes/kubernetes/blob/master/test/e2e/network/hermetic.go): links to [SIG Network testgrid](https://testgrid.k8s.io/sig-network-misc?include-filter-by-regex=PodNetworkMode) and [triage](https://storage.googleapis.com/k8s-triage/index.html?test=PodNetworkMode) to be added once the job exists.

### Graduation Criteria

#### Alpha

- Feature implemented behind the `PodNetworkMode` feature gate (default off)
  in kube-apiserver, kube-controller-manager and kubelet.
- new Pod field `networkMode` enum, defaulting/mirroring and validation complete.
- CRI `PodSandboxConfig.network_mode` and `RuntimeFeatures.network_mode_none`
  merged in `cri-api`; kubelet fail-closed admission implemented.
- Kubelet/runtime support surfaced via [Node Declared Features (KEP-5328)]
  so the scheduler avoids nodes without support.
- Unit and integration tests listed above; initial e2e tests running against
  a reference runtime implementation.

#### Beta

- At least one released upstream runtime (containerd and/or CRI-O) supports
  `PodSandboxNetworkMode NONE` and reports the capability.
- Scheduling based on node declared features validated in heterogeneous
  clusters (mixed node versions and runtimes).
- Feedback gathered from DRA driver authors and early adopters.
- E2e tests running in Testgrid and linked in this KEP; no flakes for the
  gate-enabled jobs.
- Documentation updated, including the [services-networking concepts] pages
  that currently assume universal pod connectivity.
- Metrics for fail-closed admission rejections available.

#### GA

- Container runtime support on containerd and CRI-O.
- Real-world usage of `networkMode: None` by at least two distinct classes of
  adopters (e.g., a DRA networking driver and a batch/security platform).
- At least two releases between beta and GA to allow user feedback and bug
  reports.
- Conformance impact settled: `networkMode: None` is an optional capability
  gated on runtime support; conformance behavior for the default mode is
  unchanged. `Cluster`/`Host` mirroring semantics are covered by conformance
  tests.
- All issues and gaps identified during beta resolved.

#### Deprecation

Not applicable

### Upgrade / Downgrade Strategy

- **Upgrade.** Enabling the feature requires enabling the `PodNetworkMode`
  gate on kube-apiserver, kube-controller-manager and kubelets, plus a
  runtime version that reports the capability. Existing workloads require no
  changes: `networkMode` defaults to values consistent with their current
  `hostNetwork` setting and behavior is identical.
- **Downgrade / gate disablement.** New writes drop the field
  (`dropDisabledFields`); stored objects keep it, per the standard API
  compatibility rules. Kubelets with the gate disabled (including older
  kubelets that already know the field) refuse to admit `networkMode: None`
  pods instead of running them attached to the network. Hermetic pods that
  are already running keep their isolated sandboxes until they are deleted.
- The mirroring rules guarantee that `hostNetwork` remains a correct source
  of truth for any component that has not been upgraded.

### Version Skew Strategy

- **kube-apiserver (HA, mixed versions).** During a rolling control-plane
  upgrade, an n-1 apiserver drops the unknown field on write. This is the
  standard alpha-field skew behavior; users should not rely on the field
  until all apiservers are upgraded and the gate is enabled everywhere.
- **Old controllers (n-1 kube-controller-manager).** Hermetic pods never have
  IPs, and pods without IPs are already excluded from Endpoints and
  EndpointSlices by existing controller logic.
- **Old kubelet (n-1..n-3, field unknown).** The most important skew: an old
  kubelet drops the unknown field and would run the pod attached to the
  cluster network (fail-open). All the mitigations work from alpha:
  1. old kubelets do not declare the node feature, so the scheduler never
     places `networkMode: None` pods on them;
  2. on upgraded nodes, the kubelet fail-closed admission covers the pods
     that bypass the scheduler (`spec.nodeName`, DaemonSets, static pods);
  3. the gate is off by default. The residual risk is a pod that both
     bypasses the scheduler and targets a non-upgraded node; this is
     documented, and operators can cordon or upgrade those nodes.
- **New kubelet + old runtime.** The runtime does not report
  `network_mode_none`; the kubelet rejects the pod at admission
  (fail-closed, `PodAdmitFailed`).
- **hostNetwork consumers.** Any component of any version reading
  `spec.hostNetwork` observes correct values thanks to mirroring.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `PodNetworkMode`
  - Components depending on the feature gate: kube-apiserver,
    kube-controller-manager, kubelet

###### Does enabling the feature change any default behavior?

No. With the gate enabled, pods that do not set `networkMode` receive a
defaulted value (`Cluster`, or `Host` when `hostNetwork: true`) that exactly
describes their existing behavior. Only pods that explicitly set
`networkMode: None` behave differently, and that value cannot be set today.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the gate on the apiserver causes the field to be dropped from
new objects; stored objects keep the field and remain valid. Kubelets with
the gate disabled refuse to admit `networkMode: None` pods (fail closed);
running hermetic pods keep their isolated sandboxes until deleted. No
existing (non-hermetic) workload is affected in any way by enabling or
disabling the gate.

###### What happens if we reenable the feature if it was previously rolled back?

The field becomes writable and enforceable again. Stored objects that
retained `networkMode` values resume full semantics. There is no state to
reconcile beyond the field itself.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests cover `dropDisabledFields` and defaulting with the gate on
and off, including objects created with the field and then updated with the
gate disabled (field retention), and kubelet admission behavior for a
`networkMode: None` pod with the gate disabled. Integration tests in
`test/integration/pods` exercise the same transitions against a real
apiserver.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Running workloads are not touched: the feature only affects pods explicitly
setting `networkMode: None`, and sandboxes are never reconfigured in place.
Failure modes during rollout are limited to hermetic pods themselves:
scheduling onto a node whose kubelet or runtime lacks support results in
`PodAdmitFailed` (visible, fail-closed) on upgraded kubelets, or — on
non-upgraded kubelets that drop the field — a pod wired to the cluster
network (see Version Skew). Mixed-version HA control planes may
intermittently drop the field on write until all apiservers are upgraded.

###### What specific metrics should inform a rollback?

- Rate of pod admission rejections on kubelets with reason
  `UnsupportedNetworkMode` (unschedulable/failing hermetic pods).
- Pods with `spec.networkMode: None` reporting a non-empty `status.podIP`
  (indicates a fail-open node that must be upgraded or cordoned).
- Standard rollout signals: `kubelet_pod_start_duration_seconds`,
  apiserver validation error rates on pod writes.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet (provisional).

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Query pods with `spec.networkMode=None` (e.g., via kube-state-metrics or a
field query). On nodes, kubelet admission rejection metrics/events with
reason `UnsupportedNetworkMode` indicate attempted use on unsupported nodes.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Other field: the pod is `Running`/`Ready` with empty `status.podIP` and
    `status.podIPs`; `kubectl exec` into the pod shows only the loopback
    interface. A hermetic pod that shows a pod IP indicates a fail-open node.
- [x] Events
  - Event Reason: `PodAdmitFailed` / `UnsupportedNetworkMode` when the node
    cannot honor the isolation request (fail closed).

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Hermetic pod sandbox creation should be at least as fast and reliable as
regular pods on the same node (it strictly removes the network plumbing step
from the startup path). No regression in startup latency SLOs for
non-hermetic pods.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kubelet_pod_start_duration_seconds`,
    `kubelet_started_pods_errors_total`, kubelet admission rejection counts
    (exact metric for `UnsupportedNetworkMode` rejections to be finalized for
    beta, and listed in `kep.yaml`).
  - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A kubelet counter for admission rejections labeled by reason (including
`UnsupportedNetworkMode`) will be added for beta if no suitable metric
exists.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- Container runtime with CRI `PodSandboxNetworkMode NONE` support
  (containerd/CRI-O version supporting this KEP).

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. Hermetic pod startup removes the network plumbing step entirely so it will improve pod startup.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. Resource usage decreases using hermetic pods (no CNI execution,
no IPAM, no endpoints, no service env var construction).

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. Hermetic pods consume fewer node resources than equivalent
cluster-networked pods (no veth pairs, no IPs, ...).

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Hermetic pods are less affected than regular pods: they do not depend on
service environment variables or cluster DNS at startup, and the kubelet
skips the service-informer sync wait for them. Otherwise standard kubelet
behavior applies.

###### What are other known failure modes?

- Hermetic pod stuck rejected on a node.
  - Detection: pod events show `PodAdmitFailed` with reason
    `UnsupportedNetworkMode`; admission rejection metrics.
  - Mitigations: upgrade the node's runtime/kubelet or reschedule the pod to
    a supporting node; this is the intended fail-closed behavior.
  - Diagnostics: kubelet log line naming the runtime and the missing
    `RuntimeFeatures.network_mode_none` capability.
  - Testing: covered by node e2e against a runtime without support.
- Hermetic pod attached to the cluster network (fail-open) on a non-upgraded
  kubelet.
  - Detection: pod with `spec.networkMode: None` reports non-empty
    `status.podIP`.
  - Mitigations: cordon/upgrade the node; keep the gate off until all nodes
    are upgraded.
  - Diagnostics: comparison of pod spec vs. status; node kubelet version.
  - Testing: node declared features (from alpha) keep the scheduler away
    from these nodes; the fail-open path itself is not testable on the old
    kubelet.

###### What steps should be taken if SLOs are not being met to determine the problem?

Inspect kubelet admission events/metrics for rejection reasons, verify
runtime capability via the node's CRI status, and check for version skew
between apiserver, kubelet and runtime.

## Implementation History

- 2026-08-26: Proposal socialized on the SIG Network [mailing list] and in a
  [design doc], gathering initial feedback.
- 2026-08-26: POC published as [kubernetes/kubernetes#141604] (API bool +
  validation/defaulting, CRI plumbing, kubelet behavior, endpoints exclusion,
  e2e/node tests, patched containerd reference implementation).
- 2026-09-02: KEP created as `provisional`, adopting the `networkMode` enum
  design based on POC and design-doc review feedback.

[mailing list]: https://groups.google.com/a/kubernetes.io/g/sig-network/c/dWqf4h4Dz8s/m/6MNo-2w2EgAJ
[design doc]: https://docs.google.com/document/d/1vVZ2zazJnDaj3z0Ahi6fknGJo_SIYXoteQZ3Q3BGYBI/

## Drawbacks

- For opted-in pods only, it breaks the long-standing documented model where
  every pod has an IP and can reach every other pod and the API server. The
  [services-networking concepts] documentation must be updated. Tooling that
  assumes every pod has an IP will see empty values (the same state as a
  sandbox that has not started yet).
- Container runtimes must implement and test a third sandbox networking mode.

## Alternatives

**A boolean field (`hermetic: *bool`).** The first design, implemented in
the POC ([kubernetes/kubernetes#141604]). It got a lot of pushback, a
boolean mutually exclusive with `hostNetwork` needs cross-field validation
in both directions and cannot evolve. The enum expresses the same intent,
makes the invalid combinations unwritable, and absorbs `hostNetwork`.

**Alternative field names.** Considered and rejected:

| Candidate | Assessment |
| --- | --- |
| `hermetic: true` | Accurately describes the sealed contract, but reviewers noted it implies isolation beyond networking (storage, host access, compute). Kept as the colloquial feature name only. |
| `loopbackOnly: true` | Too mechanical; describes the Linux namespace state rather than workload intent, and is unclear about DNS/service links/probes. |
| `isolated: true` | Overloaded: confused with compute isolation (CPU pinning, NUMA), security sandboxes (gVisor/Kata) and NetworkPolicy namespace isolation. |
| `airgapped: true` | Misleading: commonly refers to offline clusters/infrastructure, not individual sandboxes. |
| `disconnected: true` | Sounds like a transient operational state (edge node offline), not an intentional configuration. |
| `noNetwork: true` | Technically inaccurate: the pod does have a network namespace and a loopback interface. |
| `cniDisabled: true` | Leaks a runtime implementation detail; CNI is not Kubernetes API vocabulary and does not apply to Windows or VM runtimes. |

`networkMode: None` avoids all of the above: it describes intent, matches the
`docker run --network none` and `dnsPolicy: None` precedents, and names the
axis (`networkMode`) that also covers `hostNetwork`.

**Pod annotation (e.g., `io.kubernetes.cri.sandbox.hermetic`).** Annotations
are not versioned, not validated, and fail open: an unaware kubelet or
runtime silently ignores them, and the dependent fields (probes, DNS, service
links) cannot be validated.

**Multi-network attachment API.** The multi-network effort models an explicit
list of networks that a pod attaches to; an empty list could mean "no
networks". That design is much larger and has unresolved conformance
questions (Services, NetworkPolicy, probes and DNS across networks). This KEP
only covers the absence of automatic attachment — a binary intent — and does
not preclude any future multi-network work.

**Deny-all NetworkPolicy.** It depends on a plugin that enforces the policy.
The pod still gets an IP and node CIDR space, still attaches interfaces,
still publishes endpoints, and there is no guarantee about the sandbox
contents.

**RuntimeClass / per-node runtime configuration that skips CNI.** It is not
declarative per pod and it is invisible to the control plane: validation,
endpoints and status handling remain wrong. It is not portable across
runtimes and it requires dedicated node pools.
