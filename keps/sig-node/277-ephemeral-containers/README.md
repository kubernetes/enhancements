# KEP-277: Ephemeral Containers

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Development](#development)
  - [Operations and Support](#operations-and-support)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Creating Ephemeral Containers](#creating-ephemeral-containers)
  - [Reattaching Ephemeral Containers](#reattaching-ephemeral-containers)
  - [Ephemeral Container Lifecycle](#ephemeral-container-lifecycle)
  - [Removing Ephemeral Containers](#removing-ephemeral-containers)
  - [Configurable Security Policy](#configurable-security-policy)
    - [Specifying Security Context](#specifying-security-context)
    - [Compatibility with existing Admission Controllers](#compatibility-with-existing-admission-controllers)
  - [User Stories](#user-stories)
    - [Operations](#operations)
    - [Debugging](#debugging)
    - [Automation](#automation)
    - [Technical Support](#technical-support)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Security Considerations](#security-considerations)
    - [Requiring a Subresource](#requiring-a-subresource)
    - [Creative New Uses of Ephemeral Containers](#creative-new-uses-of-ephemeral-containers)
- [Design Details](#design-details)
  - [Kubernetes API Changes](#kubernetes-api-changes)
    - [Pod Changes](#pod-changes)
      - [Alternative Considered: Omitting TargetContainerName](#alternative-considered-omitting-targetcontainername)
    - [Updating a Pod](#updating-a-pod)
  - [Container Runtime Interface (CRI) changes](#container-runtime-interface-cri-changes)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
  - [Container Spec in PodStatus](#container-spec-in-podstatus)
  - [Extend the Existing Exec API (&quot;exec++&quot;)](#extend-the-existing-exec-api-exec)
  - [Ephemeral Container Controller](#ephemeral-container-controller)
  - [Mutable Pod Spec Containers](#mutable-pod-spec-containers)
  - [Image Exec](#image-exec)
  - [Attaching Container Type Volume](#attaching-container-type-volume)
  - [Using docker cp and exec](#using-docker-cp-and-exec)
  - [Inactive container](#inactive-container)
  - [Implicit Empty Volume](#implicit-empty-volume)
  - [Standalone Pod in Shared Namespace (&quot;Debug Pod&quot;)](#standalone-pod-in-shared-namespace-debug-pod)
  - [Exec from Node](#exec-from-node)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This proposal adds to Kubernetes a mechanism to run a container with a
temporary duration that executes within namespaces of an existing pod.
Ephemeral Containers are initiated by a user and intended to observe the state
of other pods and containers for troubleshooting and debugging purposes.

Ephemeral Containers unlock the possibility for a new command, `kubectl debug`,
which parallels the existing, `kubectl exec`.  Whereas `kubectl exec` runs a
_process_ in a _container_, `kubectl debug` could run a _container_ in a _pod_.

For example, the following command would attach to a newly created container in
a pod:

```
kubectl debug -it --image=debian target-pod -- bash
```

## Motivation

### Development

Many developers of native Kubernetes applications wish to treat Kubernetes as an
execution platform for custom binaries produced by a build system. These users
can forgo the scripted OS install of traditional Dockerfiles and instead `COPY`
the output of their build system into a container image built `FROM scratch` or
a
[distroless container image](https://github.com/GoogleCloudPlatform/distroless).
This confers several advantages:

1.  **Minimal images** lower operational burden and reduce attack vectors.
1.  **Immutable images** improve correctness and reliability.
1.  **Smaller image size** reduces resource usage and speeds deployments.

The disadvantage of using containers built `FROM scratch` is the lack of system
binaries provided by a Linux distro image makes it difficult to
troubleshoot running containers. Kubernetes should enable one to troubleshoot
pods regardless of the contents of the container images.

On Windows, the minimal [Nano Server](https://hub.docker.com/_/microsoft-windows-nanoserver)
image is the smallest available, which still retains the `cmd` shell and some
common tools such as `curl.exe`. This makes downloading debugger scripts and
tools feasible today during a `kubectl exec` session without the need for a
separate ephemeral container. Windows cannot build containers `FROM scratch`.

### Operations and Support

As Kubernetes gains in popularity, it's becoming the case that a person
troubleshooting an application is not necessarily the person who built it.
Operations staff and Support organizations want the ability to attach a "known
good" or automated debugging environment to a pod.

### Goals

In order to support the debugging use case, Ephemeral Containers must:

*   allow access to namespaces and the file systems of individual containers
*   fetch container images at run time rather than at the time of pod or image
    creation
*   respect admission controllers and audit logging
*   be discoverable via the API
*   support arbitrary runtimes via the CRI (possibly with reduced feature set)
*   require no administrative access to the node
*   have no _inherent_ side effects to the running container image
*   define a v1.Container available for inspection by admission controllers

### Non-Goals

Even though this proposal makes reference to a `kubectl debug`, implementation
of this user-level command is out of scope. This KEP focuses on the API and
kubelet changes required to enable such a debugging experience.

A method for debugging using Ephemeral Containers should be proposed in a
separate KEP or implemented via `kubectl` plugins.

Pods running on Windows Server 2019 will not have feature parity and support
all the user stories described here. Only the network troubleshooting user
story detailed under [Operations](#operations) would be feasible.

## Proposal

In order to execute binaries that may not have been included at pod creation
type, we will introduce a new type of container, the Ephemeral Container, which
may be added to a pod that is already running. Ephemeral containers are not the
building blocks of services: they're an alternative to copying binaries to
pods or building large container images that may have every binary you might
need.

Because they don't fit within the normal pod lifecycle, and since they're not
intended for building services, ephemeral containers have a number of
restrictions:

* They may only be added a pod that has already been created.
* They will not be restarted.
* No resources are reserved for processes in ephemeral containers, and resource
  configuration may not be specified.
* Fields used for building services, such as ports, may not be specified.

### Creating Ephemeral Containers

Ephemeral containers are described in the `EphemeralContainers` field of
`Pod.Spec`. This must be updated using the `/ephemeralcontainers` subresource,
similarly to updating `Pod.Status` via `/status`.

The end-to-end process for creating an ephemeral container is:

1.  Fetch a `Pod` object from the `/pods` resource.
1.  Modify `spec.ephemeralContainers` and write it back to the Pod's
    `/ephemeralcontainers` subresource, for example using `UpdateEphemeralContainers`
    in the generated client. (Patching is also supported on `/ephemeralcontainers`.)
1.  The apiserver discards all changes except those to `spec.ephemeralContainers`.
    That is, only `spec.ephemeralContainers` may be changed via `/ephemeralcontainers`.
1.  The apiserver validates the update.
    1.  Pod validation fails if container spec contains fields disallowed for
        Ephemeral Containers or the same name as a container in the spec or
        `EphemeralContainers`.
    1.  Registered admission controllers receive an `AdmissionReview` request
        containing the entire `Pod`.
1.  The kubelet's pod watcher notices the update and triggers a `syncPod()`.
    During the sync, the kubelet calls `kuberuntime.StartEphemeralContainer()`
    for any new Ephemeral Container.
    1.  `StartEphemeralContainer()` uses the existing `startContainer()` to
        start the Ephemeral Container.
    1.  After initial creation, future invocations of `syncPod()` will publish
        its ContainerStatus but otherwise ignore the Ephemeral Container. It
        will exist for the life of the pod sandbox or it exits. In no event will
        it be restarted.
1.  `syncPod()` finishes a regular sync, publishing an updated PodStatus (which
    includes the new `EphemeralContainer`) by its normal, existing means.
1.  The client performs an attach to the debug container's console.

There are no limits on the number of Ephemeral Containers that can be created in
a pod, but exceeding a pod's resource allocation may cause the pod to be
evicted.

### Reattaching Ephemeral Containers

One may reattach to a Ephemeral Container using `kubectl attach`. When supported
by a runtime, multiple clients can attach to a single debug container and share
the terminal. This is supported by the Docker runtime.

### Ephemeral Container Lifecycle

Ephemeral containers will stop when their command exits, such as exiting a
shell, and they will not be restarted.  Unlike `kubectl exec`, processes in
Ephemeral Containers will not receive an EOF if their connections are
interrupted, so shells won't automatically exit on disconnect.

There is no API support for killing or restarting an ephemeral container.
The only way to exit the container is to send it an OS signal.

### Removing Ephemeral Containers

Ephemeral containers may not be removed from a Pod once added, but
we've received feedback during the alpha period that users would like
the possibility of removing ephemeral containers (see
[#84764](https://issues.k8s.io/84764)).

Removal is out of scope for the initial graduation of ephemeral containers,
but it may be added by a future KEP.

### Configurable Security Policy

The ability to add containers to a pod implies security trade offs. We've
received the following requirements and feedback on the alpha implementation:

*   Admission controllers should be able to enforce policy based on the
    cumulative pod specification, so operations that prune information,
    such as removing Ephemeral Containers should not be allowed.
*   Restarting a pod is disruptive, so for reasons of operation, security,
    and resource hygiene it should be possible to delete Ephemeral Containers
    via the API. ([#84764]).
*   Ephemeral Containers could allow privilege escalation greater than that
    of the initial pod, so setting a custom security context should not be
    allowed.
*   Ephemeral Containers, which are initiated by humans for debugging purposes,
    should be allowed a more permissive security context regular containers.
    ([#53188])

Security policy is a problem better solved through the existing extension
mechanism for applying custom policy: [Admission Controllers].

Cluster administrators will be expected to choose from one of the following
mechanisms for restricting usage of ephemeral containers:

*   Use RBAC to control which users are allowed to access the
    `/ephemeralcontainers` subresource.
*   Write or use a third-party admission controller to allow or reject
    Pod updates that modify ephemeral containers based on the content of
    the update.
*   Disable the feature using the `EphemeralContainers` feature gate.

This means that all ephemeral container features will be allowed in a default
Kubernetes install.

[Admission Controllers]: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/

#### Specifying Security Context

The initial implementation of Ephemeral Containers prohibited setting a
`securityContext` for an Ephemeral Container. This is now explicitly allowed.
Cluster administrators wishing to limit this feature should prevent this using
admission control or RBAC.

#### Compatibility with existing Admission Controllers

Existing Admission Controllers concerned with security will need to be updated
to examine the `securityContext` of `ephemeralContainers`. Admission Controllers
configured to fail open (for example, by ignoring updates using the
`/ephemeralcontainers` subresource or not checking ephemeral containers for
a security context) are at risk of no longer protecting against privilege
escalation.

Because the initial implementation of the Ephemeral Containers API specified
that `securityContext` in ephemeral containers is not allowed, some Admission
Controllers may have chosen to ignore this field.

Since it's not feasible to discover how many admission controllers are affected
by this, the best way to move forward is to make the change sooner rather than
later and emphasize the change in release notes. We'll stress that cluster
administrators should ensure that their admission controllers support ephemeral
containers prior to upgrading and provide instructions for how to disable
ephemeral container creation in a cluster.

### User Stories

#### Operations

Jonas runs a service "neato" that consists of a statically compiled Go binary
running in a minimal container image. One of the its pods is suddenly having
trouble connecting to an internal service. Being in operations, Jonas wants to
be able to inspect the running pod without restarting it, but he doesn't
necessarily need to enter the container itself. He wants to:

1.  Inspect the filesystem of target container
1.  Execute debugging utilities not included in the container image
1.  Initiate network requests from the pod network namespace

This is achieved by running a new "debug" container in the pod namespaces. His
troubleshooting session might resemble:

```
% kubectl debug -it -m debian neato-5thn0 -- bash
root@debug-image:~# ps x
  PID TTY      STAT   TIME COMMAND
    1 ?        Ss     0:00 /pause
   13 ?        Ss     0:00 bash
   26 ?        Ss+    0:00 /neato
  107 ?        R+     0:00 ps x
root@debug-image:~# cat /proc/26/root/etc/resolv.conf
search default.svc.cluster.local svc.cluster.local cluster.local
nameserver 10.155.240.10
options ndots:5
root@debug-image:~# dig @10.155.240.10 neato.svc.cluster.local.

; <<>> DiG 9.9.5-9+deb8u6-Debian <<>> @10.155.240.10 neato.svc.cluster.local.
; (1 server found)
;; global options: +cmd
;; connection timed out; no servers could be reached
```

Jonas discovers that the cluster's DNS service isn't responding.

#### Debugging

Thurston is debugging a tricky issue that's difficult to reproduce. He can't
reproduce the issue with the debug build, so he attaches a debug container to
one of the pods exhibiting the problem:

```
% kubectl debug -it --image=gcr.io/neato/debugger neato-5x9k3 -- sh
Defaulting container name to debug.
/ # ps x
PID   USER     TIME   COMMAND
    1 root       0:00 /pause
   13 root       0:00 /neato
   26 root       0:00 sh
   32 root       0:00 ps x
/ # gdb -p 13
...
```

He discovers that he needs access to the actual container, which he can achieve
by installing busybox into the target container:

```
root@debug-image:~# cp /bin/busybox /proc/13/root
root@debug-image:~# nsenter -t 13 -m -u -p -n -r /busybox sh


BusyBox v1.22.1 (Debian 1:1.22.0-9+deb8u1) built-in shell (ash)
Enter 'help' for a list of built-in commands.

/ # ls -l /neato
-rwxr-xr-x    2 0        0           746888 May  4  2016 /neato
```

Note that running the commands referenced above requires `CAP_SYS_ADMIN` and
`CAP_SYS_PTRACE`.

This scenario also requires process namespace sharing which is not available
on Windows.

#### Automation

Ginger is a security engineer tasked with running security audits across all of
her company's running containers. Even though her company has no standard base
image, she's able to audit all containers using:

```
% for pod in $(kubectl get -o name pod); do
    kubectl debug -m gcr.io/neato/security-audit -p $pod /security-audit.sh
  done
```

#### Technical Support

Roy's team provides support for his company's multi-tenant cluster. He can
access the Kubernetes API (as a viewer) on behalf of the users he's supporting,
but he does not have administrative access to nodes or a say in how the
application image is constructed. When someone asks for help, Roy's first step
is to run his team's autodiagnose script:

```
% kubectl debug --image=k8s.gcr.io/autodiagnose nginx-pod-1234
```

### Notes/Constraints/Caveats

1.  There are no guaranteed resources for ad-hoc troubleshooting. If
    troubleshooting causes a pod to exceed its resource limit it may be evicted.
1.  There's an output stream race inherent to creating then attaching a
    container which causes output generated between the start and attach to go
    to the log rather than the client. This is not specific to Ephemeral
    Containers and exists because Kubernetes has no mechanism to attach a
    container prior to starting it. This larger issue will not be addressed by
    Ephemeral Containers, but Ephemeral Containers would benefit from future
    improvements or work arounds.
1.  Ephemeral Containers should not be used to build services, which we've
    attempted to reflect in the API.

### Risks and Mitigations

#### Security Considerations

Ephemeral Containers have no additional privileges above what is available to
any `v1.Container`. It's the equivalent of configuring an shell container in a
pod spec except that it is created on demand.

Admission plugins must be updated to guard `/ephemeralcontainers`. They should
apply the same container image and security policy as for regular containers.

We designed the API to be compatible with the existing Kubernetes RBAC
mechanism. Cluster Administrators are able to authorize Ephemeral Containers
independent of other pod operations.

We've worked with the sig-auth leads to review these changes.

#### Requiring a Subresource

It would simplify initial implementation if we updated `EphemeralContainers`
with a standard pod update, but we've received clear feedback that cluster
administrators want close control over this feature. This requires a separate
subresource.

This feature will have a long alpha, and we can re-examine this decision prior
to exiting alpha.

#### Creative New Uses of Ephemeral Containers

Though this KEP focuses on debugging, Ephemeral Containers are a general
addition to Kubernetes, and we should expect that the community will use them to
solve other problems. This is good and intentional, but Ephemeral Containers
have inherent limitations which can lead to pitfalls.

For example, it might be tempting to use Ephemeral Containers to perform
critical but asynchronous functions like backing up a production database, but
this would be dangerous because Ephemeral Containers have no execution
guarantees and could even cause the database pod to be evicted by exceeding its
resource allocation.

As much as possible we've attempted to make it clear in the API these
limitations, and we've restricted the use of fields that imply a container
should be part of `Spec.Containers`.

## Design Details

### Kubernetes API Changes

Ephemeral Containers are implemented in the Core API to avoid new dependencies
in the kubelet.  The API doesn't require an Ephemeral Container to be used for
debugging. It's intended as a general purpose construct for running a
short-lived container in a pod.

#### Pod Changes

Ephemeral Containers are represented in `PodSpec` and `PodStatus`:

```
type PodSpec struct {
	...
	// List of user-initiated ephemeral containers to run in this pod.
	// This field is alpha-level and is only honored by servers that enable the EphemeralContainers feature.
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	EphemeralContainers []EphemeralContainer `json:"ephemeralContainers,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,34,rep,name=ephemeralContainers"`
}

type PodStatus struct {
	...
	// Status for any Ephemeral Containers that running in this pod.
	// This field is alpha-level and is only honored by servers that enable the EphemeralContainers feature.
	// +optional
	EphemeralContainerStatuses []ContainerStatus `json:"ephemeralContainerStatuses,omitempty" protobuf:"bytes,13,rep,name=ephemeralContainerStatuses"`
}
```

`EphemeralContainerStatuses` resembles the existing `ContainerStatuses` and
`InitContainerStatuses`, but `EphemeralContainers` introduces a new type:

```
// An EphemeralContainer is a container that may be added temporarily to an existing pod for
// user-initiated activities such as debugging. Ephemeral containers have no resource or
// scheduling guarantees, and they will not be restarted when they exit or when a pod is
// removed or restarted. If an ephemeral container causes a pod to exceed its resource
// allocation, the pod may be evicted.
// Ephemeral containers may not be added by directly updating the pod spec. They must be added
// via the pod's ephemeralcontainers subresource, and they will appear in the pod spec
// once added.
// This is an alpha feature enabled by the EphemeralContainers feature flag.
type EphemeralContainer struct {
	// Ephemeral containers have all of the fields of Container, plus additional fields
	// specific to ephemeral containers. Fields in common with Container are in the
	// following inlined struct so than an EphemeralContainer may easily be converted
	// to a Container.
	EphemeralContainerCommon `json:",inline" protobuf:"bytes,1,req"`

	// If set, the name of the container from PodSpec that this ephemeral container targets.
	// The ephemeral container will be run in the namespaces (IPC, PID, etc) of this container.
	// If not set then the ephemeral container is run in whatever namespaces are shared
	// for the pod. Note that the container runtime must support this feature.
	// +optional
	TargetContainerName string `json:"targetContainerName,omitempty" protobuf:"bytes,2,opt,name=targetContainerName"`
}
```

Much of the utility of Ephemeral Containers comes from the ability to run a
container within the PID namespace of another container. `TargetContainerName`
allows targeting a container that doesn't share its PID namespace with the rest
of the pod. We must modify the CRI to enable this functionality (see below).

`EphemeralContainerCommon` is an inline copy of `Container` that resolves the
following contradictory requirements:

1. Ephemeral containers should be represented by a type that is easily
   convertible to `Container` so that code that operations on `Container` can
   also operate on ephemeral containers.
1. Fields of `Container` that have different behavior for ephemeral containers
   should be separately and clearly documented. Since many fields of ephemeral
   containers have different behavior, this requires a separate type.

`EphemeralContainerCommon` contains fields that ephemeral containers have in
common with `Container`. It's field-for-field copy of `Container`, which is
enforced by the compiler:

```
// EphemeralContainerCommon converts to Container. All fields must be kept in sync between
// these two types.
var _ = Container(EphemeralContainerCommon{})
```

Since `EphemeralContainerCommon` is inlined, the API machinery hides this
complexity from the end user, who sees a type, `EphemeralContainer` which has
all of the fields of `Container` plus an additional field `targetContainerName`.

##### Alternative Considered: Omitting TargetContainerName

It would be simpler for the API, kubelet and kubectl if `EphemeralContainers`
was a `[]Container`, but as isolated PID namespaces will be the default for some
time, being able to target a container will provide a better user experience.

#### Updating a Pod

Most fields of `Pod.Spec` are immutable once created. There is a short allow
list of fields which may be updated, and we will extend this to include
`EphemeralContainers`. The ability to add new containers is a large change for
Pod, however, and we'd like to begin conservatively by enforcing the following
best practices:

1.  Ephemeral Containers lack guarantees for resources or execution, and they
    will never be automatically restarted. To avoid pods that depend on
    Ephemeral Containers, we allow their addition only in pod updates and
    disallow them during pod create.
1.  Some fields of `v1.Container` imply a fundamental role in a pod. We will
    disallow the following fields in Ephemeral Containers: `ports`,
    `livenessProbe`, `readinessProbe`, and `lifecycle.`
1.  Some fields of `v1.Container` imply consequences for the entire pod. For
    example, one would expect setting `resources` to increase resources
    allocated to the pod, but this is not yet supported. We will disallow
    `resources` in Ephemeral Containers.
1.  Cluster administrators may want to restrict access to Ephemeral Containers
    independent of other pod updates.

To enforce these restrictions and enable RBAC, we will introduce a new Pod
subresource, `/ephemeralcontainers`. `EphemeralContainers` can only be modified
via this subresource. `EphemeralContainerStatuses` is updated in the same manner
as everything else in `Pod.Status` via `/status`.

`Pod.Spec.EphemeralContainers` may be updated via `/ephemeralcontainers` as per
normal (using PUT, PATCH, etc) except that existing Ephemeral Containers may
not be modified.

The subresources `attach`, `exec`, `log`, and `portforward` are available for
Ephemeral Containers and will be forwarded by the apiserver. This means `kubectl
attach`, `kubelet exec`, `kubectl log`, and `kubectl port-forward` will work for
Ephemeral Containers.

Once the pod is updated, the kubelet worker watching this pod will launch the
Ephemeral Container and update its status. A client creating a new Ephemeral
Container is expected to watch for the creation of the container status before
attaching to the console using the existing attach endpoint,
`/api/v1/namespaces/$NS/pods/$POD_NAME/attach`. Note that any output of the new
container occurring between its creation and attach will not be replayed, but it
can be viewed using `kubectl log`.

### Container Runtime Interface (CRI) changes

Since Ephemeral Containers use the Container Runtime Interface, Ephemeral
Containers will work for any runtime implementing the CRI, including Windows
containers. It's worth noting that Ephemeral Containers are significantly more
useful when the runtime implements [Process Namespace Sharing].
Windows Server 2019 does not support process namespace sharing
(see [doc](https://kubernetes.io/docs/setup/windows/intro-windows-in-kubernetes/#v1-pod)).

The CRI requires no changes for basic functionality, but it will need to be
updated to support container namespace targeting, described fully in
[Targeting a Namespace].

[Process Namespace Sharing]: https://git.k8s.io/enhancements/keps/sig-node/495-pod-pid-namespace
[Targeting a Namespace]: https://git.k8s.io/enhancements/keps/sig-node/495-pod-pid-namespace#targeting-a-specific-containers-namespace

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->
This feature will be tested with a combination of unit, integration and e2e
tests. In particular:

* Field validation (e.g. of Container fields disallowed in Ephemeral Containers)
  will be tested in unit tests.
* Pod update semantics will be tested in integration tests.
* Ephemeral Container creation will be tested in e2e-node.

None of the tests for this feature are unusual or tricky.

### Graduation Criteria

#### Alpha -> Beta Graduation

- [x] Ephemeral Containers API has been in alpha for at least 2 releases.
- [x] Ephemeral Containers support namespace targeting.
- [ ] Metrics for Ephemeral Containers are added to existing contain creation
  metrics.
- [x] CLI using Ephemeral Containers for debugging checked into a Kubernetes
  project repository (e.g. in `kubectl` or a `kubectl` plugin).
- [x] A task on https://kubernetes.io/docs/tasks/ describes how to troubleshoot
  a running pod using Ephemeral Containers.
- [ ] Ephemeral Container creation is covered by e2e-node tests.
- [ ] Update via `/ephemeralcontainers` validates entire PodSpec to protect against future bugs.

#### Beta -> GA Graduation

- [ ] Ephemeral Containers have been in beta for at least 2 releases.
- [ ] Ephemeral Containers see use in 3 projects or articles.
- [ ] Ephemeral Container creation is covered by [conformance tests].
- [ ] The following cosmetic codebase TODOs are resolved:
  - [ ] kubectl incorrectly suggests a debug container can be reattached after exit
  - [ ] `validateEphemeralContainers` adds a superfluous index to ephemeral container spec paths

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

### Upgrade / Downgrade Strategy

No action is required when upgrading/downgrading between versions of this feature.

### Version Skew Strategy

For API compatibility, we rely on the [Adding Unstable Features to Stable
Versions] API Changes recommendations. An n-2 kubelet won't recognize the new
fields, so the API should remain in alpha for at least 2 releases.

Namespace targeting requires adding an enum value to the CRI. This will present
an unknown value to old CRIs. Ideally, [CRI Optional Runtime Features] would
allow us to query for this feature, but this is unlikely to be implemented.
Instead, we will update the CRI and add a conformance test. (As of this KEP the
CRI is still in alpha.) Runtimes will be expected to handle an unknown
`NamespaceMode` gracefully.

[Adding Unstable Features to Stable Versions]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#adding-unstable-features-to-stable-versions
[CRI Optional Runtime Features]: https://issues.k8s.io/32803

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md.

The production readiness review questionnaire must be completed for features in
v1.19 or later, but is non-blocking at this time. That is, approval is not
required in order to be in the release.

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

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: EphemeralContainers
    - Components depending on the feature gate: kube-apiserver, kubelet
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**

  No, this feature does not change existing behavior.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

  Yes. Any running ephemeral containers will continue to run, but they will
  become inaccessible and exit when the Pod is deleted.

* **What happens if we reenable the feature if it was previously rolled back?**

  This behaves as expected: the feature will begin working again.

* **Are there any tests for feature enablement/disablement?**

  Some unit tests are exercised with the feature both enabled and disabled to
  verify proper behavior in both cases. Integration test verify that the API
  server accepts/rejects requests when the feature is enabled/disabled.

  This feature is implemented in the apiserver and kubelet. The change to the
  core API has had several releases to soak. For the apiserver, the main risk
  is described by [Adding Unstable Features to Stable Versions]. Specifically:

  > Ensuring existing data is preserved is needed so that when the feature is
  > enabled by default in a future version n and data is unconditionally allowed
  > to be persisted in the field, an n-1 API server (with the feature still
  > disabled by default) will not drop the data on update.

  We've followed the instructions in this doc for how to persist unstable fields
  during an update. The case for this feature is slightly more complicated
  because the field may not be set by the default update resource. This logic
  has been in place for several releases. It has unit tests in
  [TestDropEphemeralContainers](https://github.com/kubernetes/kubernetes/blob/v1.20.2/pkg/api/pod/util_test.go#L1450).

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**

  This feature allows setting a new field, `ephemeralContainers` in a Pod spec.
  Enabling the feature won't affect existing workloads since they were not
  previously allowed to set this field.

  Component restarts won't affect this feature.

* **What specific metrics should inform a rollback?**

  A rollback is only indicated if there's a catastrophic failure that prevents
  the cluster from functioning normally, for example if pod or container
  creation begins to fail.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

  Since this feature is not critical to production workloads, the main risk
  is that enabling the feature by default will adversely affect other components.

  I've tried to simulate this manually by running (using `local-cluster-up.sh`
  with `PRESERVE_ETCD=true`):

  1. (Cluster version 1.20.2, `FEATURE_GATES=EphemeralContainers=false`):
     1. Create pod
     1. Attempt to create ephemeral container (expect fail)

  1. (Cluster version 1.21+, `FEATURE_GATES=EphemeralContainers=true`):
     1. describe pod
     1. exec in pod
     1. Attempt to create ephemeral container (expect success)

  1. (Cluster version 1.20.2, `FEATURE_GATES=EphemeralContainers=false`):
     1. describe pod
     1. exec in pod
     1. Attempt to create ephemeral container (expect fail)

  1. (Cluster version 1.21+, `FEATURE_GATES=EphemeralContainers=true`):
     1. describe pod
     1. exec in pod
     1. Attempt to create ephemeral container (expect success)

  The apiserver and kubelet have automated upgrade tests
  (https://testgrid.k8s.io/google-gce-upgrade), but these likely don't exercise
  ephemeral containers. We'll investigate whether it's possible to add ephemeral
  containers to these existing tests.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**

  No.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

  This information is available by examining pod objects in the API server
  for the field `pod.spec.ephemeralContainers`. Additionally, the kubelet surfaces
  the following metrics, added in [#99000](https://issues.k8s.io/99000):

  - `kubelet_managed_ephemeral_containers`: The number of ephemeral containers
    in pods managed by this kubelet.
  - `kubelet_started_containers_total`: Counter of all containers started by
    this kubelet, indexed by `container_type`. Ephemeral containers have a
    `container_type` of `ephemeral_container`.
  - `kubelet_started_containers_errors_total `: Counter of errors encountered
    when this kubelet starts containers, idnexed by `container_type`.
    Ephemeral containers have a `container_type` of `ephemeral_container`.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [x] Metrics
    - Metric name: `apiserver_request_total{component="apiserver",resource="pods",subresource="ephemeralcontainers"}` (apiserver), `kubelet_started_containers_errors_total{container_type="ephemeral_container"}`
    - [Optional] Aggregation method: Aggregate by container type
    - Components exposing the metric: apiserver, kubelet
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

  Ephemeral containers are, by design, best effort. We are unable to offer an SLO
  for ephemeral containers until the kubelet supports some sort of dynamic resource
  reallocation.

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**

  No.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**

  - Runtime support for [Namespace targeting].
    - Usage description: One feature of Ephemeral containers, namespace
      targeting, uses a feature of the CRI that is often overlooked by runtime
      implementors.
      - Impact of its outage on the feature: Degraded operation. Ephemeral
        containers will work, but will receive an isolated namespace.
      - Impact of its degraded performance or high-error rates on the feature: N/A

[Namespace targeting]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/495-pod-pid-namespace#targeting-a-specific-containers-namespace

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**

  Not in a meaningful way. Any additional calls would fall within existing
  usage patterns of humans interactive with Pods.

* **Will enabling / using this feature result in introducing new API types?**

  There an no new Kinds for storage, but new types are used in `v1.Pod`.
  Ephemeral containers are added by writing a `v1.Pod` containing
  `pod.spec.ephemeralContainers` to the pod's `/ephemeralcontainers`
  subresource, similar to how the kubelet updates `pod.status`.

  - API type: 
    - v1.Pod (with `/ephemeralcontainers` subresource)
  - Supported number of objects per cluster: same as Pods
  - Supported number of objects per namespace: same as Pods

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**

  No.

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**

  - API type(s): v1.Pod
  - Estimated increase in size: Additional `Container` for each Ephemeral
    container. This is expected to be negligible since these are created
    manually by humans.
  - Estimated amount of new objects: N/A

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**

  When users add additional containers to a Pod, the pod will have additional
  containers to shut down and garbage collect when the Pod exits.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**

  Not automatically. Use of this feature will result in additional containers
  running on kubelets, but it does not change the amount of resources allocated
  to pods.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

  Identical to other (non-ephemeral) containers.

* **What are other known failure modes?**
  - Addition of ephemeral container is prohibited by API server
    - Detection: API server metric described in monitoring section
    - Mitigations: None. This doesn't affect user workloads.
    - Diagnostics: API error returned to user.
    - Testing: Yes, integration tests.

  - Ephemeral container is added to Pod, but not created by kubelet for unknown reason.
    - Detection: kubelet metric described in monitoring section.
    - Mitigations: None. This doesn't affect user workloads.
    - Diagnostics: Error message added to Pod event log.
    - Testing: No. There aren't any known failure reasons to test for.

  - Feature flag is enabled on apiserver but not kubelet.
    - Detection: This is not specific to this feature. I'm not sure of the
      recommended way to detect out-of-sync feature flags between components.
    - Mitigations: None. This doesn't affect user workloads.
    - Diagnostics: This one is tough because the code to print error messages is
      hidden behind the feature flag...
    - Testing: No, testing for cluster misconfiguration at dev time doesn't
      prevent cluster misconfiguration at run time.

  One may completely disable the feature using the `EphemeralContainers` feature
  flag, but it's also possible to prevent the creation of new ephemeral containers
  without a restart by removing authorization to `ephemeralcontainers` subresource
  via [RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/).

* **What steps should be taken if SLOs are not being met to determine the problem?**

  Troubleshoot using apiserver and kubelet error logs.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- *2016-06-09*: Opened [#27140](https://issues.k8s.io/27140) to explore
  solutions for debugging minimal container images.
- *2017-09-27*: Merged first version of proposal for troubleshooting running
  pods [kubernetes/community#649](https://github.com/kubernetes/community/pull/649)
- *2018-08-23*: Merged update to use `Container` in `Pod.Spec`
  [kubernetes/community#1269](https://github.com/kubernetes/community/pull/1269)
- *2019-02-12*: Ported design proposal to KEP.
- *2019-04-24*: Added notes on Windows feature compatibility
- *2020-09-29*: Ported KEP to directory-based template.
- *2021-01-07*: Updated KEP for beta release in 1.21 and completed PRR section.
- *2021-04-12*: Switched `/ephemeralcontainers` API to use `Pod`.
- *2021-05-14*: Add additional graduation criteria
- *2021-07-09*: Revert KEP to alpha because of the new API introduced in 1.22.
- *2021-08-23*: Updated KEP for beta release in 1.23.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

We've explored many alternatives to Ephemeral Containers for the purposes of
debugging, so this section is quite long.

### Container Spec in PodStatus

Originally there was a desire to keep the pod spec immutable, so we explored
modifying only the pod status. An `EphemeralContainer` would contain a Spec, a
Status and a Target:

```
// EphemeralContainer describes a container to attach to a running pod for troubleshooting.
type EphemeralContainer struct {
        metav1.TypeMeta `json:",inline"`

        // Spec describes the Ephemeral Container to be created.
        Spec *Container `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

        // Most recently observed status of the container.
        // This data may not be up to date.
        // Populated by the system.
        // Read-only.
        // +optional
        Status *ContainerStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`

        // If set, the name of the container from PodSpec that this ephemeral container targets.
        // If not set then the ephemeral container is run in whatever namespaces are shared
        // for the pod.
        TargetContainerName string `json:"targetContainerName,omitempty" protobuf:"bytes,4,opt,name=targetContainerName"`
}
```

Ephemeral Containers for a pod would be listed in the pod's status:

```
type PodStatus struct {
        ...
        // List of user-initiated ephemeral containers that have been run in this pod.
        // +optional
        EphemeralContainers []EphemeralContainer `json:"ephemeralContainers,omitempty" protobuf:"bytes,11,rep,name=ephemeralContainers"`

}
```

To create a new Ephemeral Container, one would append a new `EphemeralContainer`
with the desired `v1.Container` as `Spec` in `Pod.Status` and updates the `Pod`
in the API. Users cannot normally modify the pod status, so we'd create a new
subresource `/ephemeralcontainers` that allows an update of solely
`EphemeralContainers` and enforces append-only semantics.

Since we have a requirement to describe the Ephemeral Container with a
`v1.Container`, this lead to a "spec in status" that seemed to violate API best
practices. It was confusing, and it required added complexity in the kubelet to
persist and publish user intent, which is rightfully the job of the apiserver.

### Extend the Existing Exec API ("exec++")

A simpler change is to extend `v1.Pod`'s `/exec` subresource to support
"executing" container images. The current `/exec` endpoint must implement `GET`
to support streaming for all clients. We don't want to encode a (potentially
large) `v1.Container` into a query string, so we must extend `v1.PodExecOptions`
with the specific fields required for creating a Debug Container:

```
// PodExecOptions is the query options to a Pod's remote exec call
type PodExecOptions struct {
        ...
        // EphemeralContainerName is the name of an ephemeral container in which the
        // command ought to be run. Either both EphemeralContainerName and
        // EphemeralContainerImage fields must be set, or neither.
        EphemeralContainerName *string `json:"ephemeralContainerName,omitempty" ...`

        // EphemeralContainerImage is the image of an ephemeral container in which the command
        // ought to be run. Either both EphemeralContainerName and EphemeralContainerImage
        // fields must be set, or neither.
        EphemeralContainerImage *string `json:"ephemeralContainerImage,omitempty" ...`
}
```

After creating the Ephemeral Container, the kubelet would upgrade the connection
to streaming and perform an attach to the container's console. If disconnected,
the Ephemeral Container could be reattached using the pod's `/attach` endpoint
with `EphemeralContainerName`.

Ephemeral Containers could not be removed via the API and instead the process
must terminate. While not ideal, this parallels existing behavior of `kubectl
exec`. To kill an Ephemeral Container one would `attach` and exit the process
interactively or create a new Ephemeral Container to send a signal with
`kill(1)` to the original process.

Since the user cannot specify the `v1.Container`, this approach sacrifices a
great deal of flexibility. This solution still requires the kubelet to publish a
`Container` spec in the `PodStatus` that can be examined for future admission
decisions and so retains many of the downsides of the Container Spec in
PodStatus approach.

### Ephemeral Container Controller

Kubernetes prefers declarative APIs where the client declares a state for
Kubernetes to enact. We could implement this in a declarative manner by creating
a new `EphemeralContainer` type:

```
type EphemeralContainer struct {
        metav1.TypeMeta
        metav1.ObjectMeta

        Spec v1.Container
        Status v1.ContainerStatus
}
```

A new controller in the kubelet would watch for EphemeralContainers and
create/delete debug containers. `EphemeralContainer.Status` would be updated by
the kubelet at the same time it updates `ContainerStatus` for regular and init
containers. Clients would create a new `EphemeralContainer` object, wait for it
to be started and then attach using the pod's attach subresource and the name of
the `EphemeralContainer`.

A new controller is a significant amount of complexity to add to the kubelet,
especially considering that the kubelet is already watching for changes to pods.
The kubelet would have to be modified to create containers in a pod from
multiple config sources. SIG Node strongly prefers to minimize kubelet
complexity.

### Mutable Pod Spec Containers

Rather than adding to the pod API, we could instead make the pod spec mutable so
the client can generate an update adding a container. `SyncPod()` has no issues
adding the container to the pod at that point, but an immutable pod spec has
been a basic assumption and best practice in Kubernetes. Changing this
assumption complicates the requirements of the kubelet state machine. Since the
kubelet was not written with this in mind, we should expect such a change would
create bugs we cannot predict.

### Image Exec

An earlier version of this proposal suggested simply adding `Image` parameter to
the exec API. This would run an ephemeral container in the pod namespaces
without adding it to the pod spec or status. This container would exist only as
long as the process it ran. This parallels the current kubectl exec, including
its lack of transparency. We could add constructs to track and report on both
traditional exec process and exec containers. In the end this failed to meet our
transparency requirements.

### Attaching Container Type Volume

Combining container volumes ([#831](https://issues.k8s.io/831)) with the ability
to add volumes to the pod spec would get us most of the way there. One could
mount a volume of debug utilities at debug time. Docker does not allow adding a
volume to a running container, however, so this would require a container
restart. A restart doesn't meet our requirements for troubleshooting.

Rather than attaching the container at debug time, kubernetes could always
attach a volume at a random path at run time, just in case it's needed. Though
this simplifies the solution by working within the existing constraints of
`kubectl exec`, it has a sufficient list of minor limitations (detailed in
[#10834](https://issues.k8s.io/10834)) to result in a poor user experience.

### Using docker cp and exec

Instead of creating an additional container with a different image, `docker cp`
could be used to add binaries into a running container before calling `exec` on
the process. This approach would be feasible on Windows as it doesn't require
process namespace sharing. It also doesn't involve the complexities with adding
mounts as described in [Attaching Container Type Volume](#attaching-container-type-volume).
However, it doesn't provide a convenient way to package or distribute binaries
as described in this KEP or the alternate [Image Exec](#image-exec) proposal.
`docker cp` also doesn't have a CRI equivalent, so that would need to be
addressed in an alternate proposal.

### Inactive container

If Kubernetes supported the concept of an "inactive" container, we could
configure it as part of a pod and activate it at debug time. In order to avoid
coupling the debug tool versions with those of the running containers, we would
want to ensure the debug image was pulled at debug time. The container could
then be run with a TTY and attached using kubectl.

The downside of this approach is that it requires prior configuration. In
addition to requiring prior consideration, it would increase boilerplate config.
A requirement for prior configuration makes it feel like a workaround rather
than a feature of the platform.

### Implicit Empty Volume

Kubernetes could implicitly create an EmptyDir volume for every pod which would
then be available as a target for either the kubelet or a sidecar to extract a
package of binaries.

Users would have to be responsible for hosting a package build and distribution
infrastructure or rely on a public one. The complexity of this solution makes it
undesirable.

### Standalone Pod in Shared Namespace ("Debug Pod")

Rather than inserting a new container into a pod namespace, Kubernetes could
instead support creating a new pod with container namespaces shared with
another, target pod. This would be a simpler change to the Kubernetes API, which
would only need a new field in the pod spec to specify the target pod. To be
useful, the containers in this "Debug Pod" should be run inside the namespaces
(network, pid, etc) of the target pod but remain in a separate resource group
(e.g. cgroup for container-based runtimes).

This would be a rather large change for pod, which is currently treated as an
atomic unit. The Container Runtime Interface has no provisions for sharing
outside of a pod sandbox and would need a refactor. This could be a complicated
change for non-container runtimes (e.g. hypervisor runtimes) which have more
rigid boundaries between pods.

This is pushing the complexity of the solution from the kubelet to the runtimes.
Minimizing change to the Kubernetes API is not worth the increased complexity
for the kubelet and runtimes.

It could also be possible to implement a Debug Pod as a privileged pod that runs
in the host namespace and interacts with the runtime directly to run a new
container in the appropriate namespace. This solution would be runtime-specific
and pushes the complexity of debugging to the user. Additionally, requiring
node-level access to debug a pod does not meet our requirements.

### Exec from Node

The kubelet could support executing a troubleshooting binary from the node in
the namespaces of the container. Once executed this binary would lose access to
other binaries from the node, making it of limited utility and a confusing user
experience.

This couples the debug tools with the lifecycle of the node, which is worse than
coupling it with container images.
