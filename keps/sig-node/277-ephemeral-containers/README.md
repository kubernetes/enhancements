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
  - [Restarting and Reattaching Ephemeral Containers](#restarting-and-reattaching-ephemeral-containers)
  - [Killing Ephemeral Containers](#killing-ephemeral-containers)
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
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
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

1.  A client constructs an `EphemeralContainer` based on command line and
    and appends it to `Pod.Spec.EphemeralContainers`. It updates the pod using
    the pod's `/ephemeralcontainers` subresource.
1.  The apiserver validates and performs the pod update.
    1.  Pod validation fails if container spec contains fields disallowed for
        Ephemeral Containers or the same name as a container in the spec or
        `EphemeralContainers`.
    1.  API resource versioning resolves update races.
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

### Restarting and Reattaching Ephemeral Containers

Ephemeral Containers will not be restarted.

We want to be more user friendly by allowing re-use of the name of an exited
ephemeral container, but this will be left for a future improvement.

One can reattach to a Ephemeral Container using `kubectl attach`. When supported
by a runtime, multiple clients can attach to a single debug container and share
the terminal. This is supported by Docker.

### Killing Ephemeral Containers

Ephemeral Containers will not be killed automatically unless the pod is
destroyed.  Ephemeral Containers will stop when their command exits, such as
exiting a shell.  Unlike `kubectl exec`, processes in Ephemeral Containers will
not receive an EOF if their connection is interrupted.

A future improvement could allow killing Ephemeral Containers when they're
removed from `EphemeralContainers`, but it's not clear that we want to allow
this. Removing an Ephemeral Container spec makes it unavailable for future
authorization decisions (e.g. whether to authorize exec in a pod that had a
privileged Ephemeral Container).

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
normal (using PUT, PATCH, etc) except that existing Ephemeral Containers may not
be modified or deleted. Deleting Ephemeral Containers is not supported in the
initial implementation to reduce complexity. It could be added in the future,
but see *Killing Ephemeral Containers* below for additional constraints.

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

- [ ] Ephemeral Containers API has been in alpha for at least 2 releases.
- [ ] Ephemeral Containers support namespace targeting.
- [ ] Tests are in Testgrid and linked in KEP.
- [ ] Metrics for Ephemeral Containers are added to existing contain creation
  metrics.
- [ ] CLI using Ephemeral Containers for debugging checked into a Kubernetes
  project repository (e.g. in `kubectl` or a `kubectl` plugin).
- [ ] A task on https://kubernetes.io/docs/tasks/ describes how to troubleshoot
  a running pod using Ephemeral Containers.
- [ ] A survey sent to early adopters doesn't reveal any major shortcomings.

#### Beta -> GA Graduation

- [ ] Ephemeral Containers have been in beta for at least 2 releases.
- [ ] Ephemeral Containers see use in 3 projects or articles.
- [ ] Ephemeral Container creation is covered by conformance tests

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

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
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).

* **What happens if we reenable the feature if it was previously rolled back?**

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
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

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
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


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
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

* **What steps should be taken if SLOs are not being met to determine the problem?**

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
