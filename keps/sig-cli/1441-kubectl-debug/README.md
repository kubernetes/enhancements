# KEP-1441: kubectl debug

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Use Cases](#use-cases)
    - [Distroless Containers](#distroless-containers)
    - [Kubernetes System Images](#kubernetes-system-images)
    - [Operations and Support](#operations-and-support)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Pod Troubleshooting with Ephemeral Debug Container](#pod-troubleshooting-with-ephemeral-debug-container)
    - [Debug Container Naming](#debug-container-naming)
    - [Container Namespace Targeting](#container-namespace-targeting)
    - [Interactive Troubleshooting and Automatic Attaching](#interactive-troubleshooting-and-automatic-attaching)
  - [Pod Troubleshooting by Copy](#pod-troubleshooting-by-copy)
    - [Creating a Debug Container by copy](#creating-a-debug-container-by-copy)
    - [Modify Application Image by Copy](#modify-application-image-by-copy)
  - [Node Troubleshooting with Privileged Containers](#node-troubleshooting-with-privileged-containers)
  - [Debugging Profiles](#debugging-profiles)
    - [Profile: general](#profile-general)
    - [Profile: baseline](#profile-baseline)
    - [Profile: restricted](#profile-restricted)
    - [Profile: sysadmin](#profile-sysadmin)
    - [Profile: netadmin](#profile-netadmin)
    - [Default Profile](#default-profile)
    - [Future Improvements](#future-improvements)
  - [User Stories](#user-stories)
    - [Operations](#operations)
    - [Debugging](#debugging)
    - [Automation](#automation)
    - [Technical Support](#technical-support)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Full <code>kubectl debug</code> Arguments](#full-kubectl-debug-arguments)
  - [Test Plan](#test-plan)
    - [Alpha milestones](#alpha-milestones)
    - [Beta milestones](#beta-milestones)
    - [GA milestones](#ga-milestones)
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
- [Alternatives](#alternatives)
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

This proposal adds a command to `kubectl` to improve the user experience of
troubleshooting. Current `kubectl` commands such as `exec` and `port-forward`
allow troubleshooting at the container and network level. `kubectl debug`
extends these capabilities to include Kubernetes abstractions such as Pod, Node
and [Ephemeral Containers].

User journeys supported by the initial release of `kubectl debug` are:

1. Create an Ephemeral Container in a running Pod to attach debugging tools
   to a distroless container image. (See [*Pod Troubleshooting with Ephemeral
   Containers*](#pod-troubleshooting-with-ephemeral-debug-container))
2. Restart a pod with a modified `PodSpec`, to allow in-place troubleshooting
   using different container images or permissions. (See [Pod Troubleshooting by
   Copy](#pod-troubleshooting-by-copy))
3. Start and attach to a privileged container in the host namespace. (See [*Node
   Troubleshooting with Privileged Containers*](
   #node-troubleshooting-with-privileged-containers))

[Ephemeral Containers]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/277-ephemeral-containers/README.md

## Motivation

### Use Cases

#### Distroless Containers

Many developers of native Kubernetes applications wish to treat Kubernetes as an
execution platform for custom binaries produced by a build system. These users
can forgo the scripted OS install of traditional Dockerfiles and instead `COPY`
the output of their build system into a container image built `FROM scratch` or
a [distroless container image]. This confers several advantages:

1.  **Minimal images** lower operational burden and reduce attack vectors.
2.  **Immutable images** improve correctness and reliability.
3.  **Smaller image size** reduces resource usage and speeds deployments.

The disadvantage of using containers built `FROM scratch` is the lack of system
binaries provided by a Linux distro image makes it difficult to
troubleshoot running containers. Kubernetes should enable one to troubleshoot
pods regardless of the contents of the container images.

#### Kubernetes System Images

Kubernetes itself is migrating to [distroless for k8s system images] such as
`kube-apiserver` and `kube-dns`. This led to the creation of a [scratch
debugger] to copy debugging tools into the running container, but this script
has some downsides:

1. Since it's not possible to install the debugging tools using only Kubernetes,
   the script issues docker commands directly and so isn't portable to other
   runtimes.
2. The script requires broad administrative access to the node to run docker
   commands.
3. Installing tools into the running container requires modifying the image
   being debugged.

`kubectl debug` would replace the scratch debugger script with a method
idiomatic to Kubernetes.

#### Operations and Support

As Kubernetes gains in popularity, it's becoming the case that a person
troubleshooting an application is not necessarily the person who built it.
Operations staff and Support organizations want the ability to attach a "known
good" or automated debugging environment to a pod.

### Goals

Make available to users of Kubernetes a troubleshooting facility that:

1. works out of the box as a first-class feature of the platform.
2. does not depend on tools having already been included in container images.
3. does not require administrative access to the node. (Administrative access
   via the Kubernetes API is acceptable.)

New `kubectl` concepts increase cognitive burden for all users of Kubernetes.
This KEP seeks to minimize this burden by mirroring the existing `kubectl`
workflows where possible.

### Non-Goals

Ephemeral containers are supported on Windows, but it's not the recommended
debugging facility. The other troubleshooting workflows described here are
equally as useful to Windows containers. We will not attempt to create
separate debugging facilities for Windows containers.

[distroless container image]: https://github.com/GoogleCloudPlatform/distroless
[distroless for k8s system images]: https://git.k8s.io/enhancements/keps/sig-release/20190316-rebase-images-to-distroless.md
[scratch debugger]: https://github.com/kubernetes-retired/contrib/tree/master/scratch-debugger

## Proposal

`kubectl debug` supports a number of debugging modes, described in the
following sections.

### Pod Troubleshooting with Ephemeral Debug Container

We will add a new command `kubectl debug` that will:

1. Construct a `v1.Container` for the debug container based on command line
   arguments. This will include an optional container for namespace targeting
   (described below).
2. Fetch the specified pod's existing ephemeral containers using
   `GetEphemeralContainers()` in the generated pod client.
3. Append the new debug container to the pod's ephemeral containers and call
   `UpdateEphemeralContainers()`.
4. Watch pod for updates to the debug container's `ContainerStatus` and
   automatically attach once the container is running. *(optional based on
   command line flag)*

We will attempt to make `kubectl debug` useful with a minimal of arguments
by using reasonable defaults where possible.

```
Examples:
  # Create an interactive debugging session in pod mypod and immediately attach to it.
  # (requires the EphemeralContainers feature to be enabled in the cluster)
  kubectl debug mypod -it --image=busybox

  # Create a debug container named debugger using a custom automated debugging image.
  # (requires the EphemeralContainers feature to be enabled in the cluster)
  kubectl debug --image=myproj/debug-tools -c debugger mypod
```

#### Debug Container Naming

Currently, there is no support for deleting or recreating ephemeral containers.
In cases where the user does not specify a name, `kubectl` should generate a
unique name and display it to the user.

#### Container Namespace Targeting

In order to see processes running in other containers in the pod, [Process
Namespace Sharing] should be enabled for the pod. In cases where process
namespace sharing isn't enabled for the pod, `kubectl` will set
`TargetContainer` in the `EphemeralContainer`. This will cause the ephemeral
container to be created in the namespaces of the target container in runtimes
that support container namespace targeting.

[Process Namespace Sharing]: https://kubernetes.io/docs/tasks/configure-pod-container/share-process-namespace

#### Interactive Troubleshooting and Automatic Attaching

Since the primary use case for `kubectl debug` is interactive troubleshooting,
`kubectl debug` supports automatically attaching to the ephemeral container
using the same conventions as `kubectl run`.

### Pod Troubleshooting by Copy

Pod troubleshooting via Ephemeral Container relies on an alpha feature which is
unlikely to be enabled on production clusters. In order to support these
clusters, and because it would be generally useful, we will support a mode of
Pod troubleshooting that behaves similar to Pod Troubleshooting with Ephemeral
Debug Container but operates instead on a copy of the target pod.

The following additional options will cause a copy of the target pod to be
created:

```
Options:
      --copy-to='': Create a copy of the target Pod with this name.
      --replace=false: When used with '--copy-to', delete the original Pod
      --same-node=false: When used with '--copy-to', schedule the copy of target Pod on the same node.
      --share-processes=true: When used with '--copy-to', enable process namespace sharing in the copy.
```

The modification `kubectl debug` makes to `Pod.Spec.Containers` depends on the
value of the `--container` flag. Unlike with debugging with ephemeral
containers, debugging by copy does not support generating a name for new
containers.

#### Creating a Debug Container by copy

If a user specifies a container name with `--container` that does not exist,
then the user is instructing `kubectl debug` to create a new Debug Container in
the Pod copy. In this case `--image` is the container image to use for the new
container.

```
Examples:
  # Create a debug container as a copy of the original Pod and attach to it
  kubectl debug mypod -it --container=debug --image=busybox --copy-to=my-debugger
```

#### Modify Application Image by Copy

If a user specifies a container name with `--container` that exists, then the
user is instructing `kubectl debug` to change the image for this one container
to the image specified in `--image`.

If a user does not specify a `--container`, then they are instructing `kubectl
debug` to change the image for one or more containers in the pod. In this case,
`--image` is the image mutation language defined by `kubectl set image`.

```
Examples:
  # Create a copy of mypod named my-debugger with my-container's image changed to busybox
  kubectl debug mypod --image=busybox --container=my-container --copy-to=my-debugger -- sleep 1d

  # Create a copy of mypod with the image of all container changed to busybox
  kubectl debug mypod --image=*=busybox --copy-to=my-debugger

  # Create a copy of mypod with the specified image changes
  kubectl debug mypod --image=main=busybox,sidecar=debian --copy-to=my-debugger
```

### Node Troubleshooting with Privileged Containers

When invoked with a node as a target, `kubectl debug` will create a new
pod with the following fields set:

* `nodeName: $target_node`
* `hostIPC: true`
* `hostNetwork: true`
* `hostPID: true`
* `restartPolicy: Never`

Additionally, the node's '/' will be mounted at `/host`.

```
Examples:
  # Create an interactive debugging session on a node and immediately attach to it.
  # The container will run in the host namespaces and the host's filesystem will be mounted at /host
  kubectl debug node/mynode -it --image=busybox
```

### Debugging Profiles

Since launching `kubectl debug` we've received feedback that more configurability
is needed for generated pods and containers.

* [kubernetes/kubernetes#97103](https://issues.k8s.io/97103): ability to set capability `SYS_PTRACE`
* [kubernetes/kubectl#1051](https://github.com/kubernetes/kubectl/issues/1051): ability to set privileged
* [kubernetes/kubectl#1070](https://github.com/kubernetes/kubectl/issues/1070): strip probes on pod copy
* (various): ability to set `SYS_ADMIN` and `NET_ADMIN` capabilities

These requests are relevant for all debugging journeys. That is, a user may want to
set `SYS_ADMIN` while debugging a node, a pod by ephemeral container, or a pod by copy.
`kubectl debug` is intended to guide the user through a debugging scenario, and
requiring the user to specify a series of flags on the command line is a poor experience.

Instead, we'll introduce "Debugging profiles" which are configurable via a single command
line flag, `--profile`.  A user may then use, for example, `--profile=netadmin` when
debugging a node to create a pod with the `NET_ADMIN` capaibility.

The available profiles will be:

| Profile    | Description                                                     |
|------------|-----------------------------------------------------------------|
| general    | A reasonable set of defaults tailored for each debuging journey |
| baseline   | Compatible with baseline [Pod Security Standard]                |
| restricted | Compatible with restricted [Pod Security Standard]              |
| sysadmin   | System Administrator (root) privileges                          |
| netadmin   | Network Administrator privileges.                               |
| legacy     | Backwards compatibility with 1.22 behavior                      |

Debugging profiles are intended to work seamlessly with the [Pod Security Standard]
enforced by the [PodSecurity] admission controller. The baseline and restricted
profiles will generate configuration compatible with the corresponding security
level.

[Pod Security Standards]: https://kubernetes.io/docs/concepts/security/pod-security-standards/
[PodSecurity]: http://kep.k8s.io/2579

#### Profile: general

| Journey             | Debug Container Behavior                                             |
|---------------------|----------------------------------------------------------------------|
| Node                | empty securityContext; uses host namespaces, mounts root partition   |
| Pod Copy            | sets `SYS_PTRACE` in debugging container, sets shareProcessNamespace |
| Ephemeral Container | sets `SYS_PTRACE` in ephemeral container                             |

This profile prioritizes the debugging experience for the general case. For pod debugging it sets
`SYS_PTRACE` and uses pod-scoped namespaces. Probes and labels are stripped from Pod copies to
ensure the copy isn't killed and doesn't receive traffic during debugging.

Node debugging uses host-scoped namespaces but doesn't otherwise request escalated privileges.

#### Profile: baseline

| Journey             | Debug Container Behavior                          |
|---------------------|---------------------------------------------------|
| Node                | empty securityContext; uses isolated namespaces   |
| Pod Copy            | empty securityContext; sets shareProcessNamespace |
| Ephemeral Container | empty securityContext                             |

This profile is identical to "general" but eliminates privileges that are disallowed under the
baseline security profile, such as host namespaces, host volume, mounts and `SYS_PTRACE`.

Probes and labels continue to be stripped from Pod copies.

#### Profile: restricted

| Journey             | Debug Container Behavior                          |
|---------------------|---------------------------------------------------|
| Node                | empty securityContext; uses private namespaces    |
| Pod Copy            | empty securityContext; sets shareProcessNamespace |
| Ephemeral Container | empty securityContext                             |

This profile is identical to "baseline" but adds configuration that's required under the restricted
security profile, such as requiring a non-root user and dropping all capabilities.

Probes and labels continue to be stripped from Pod copies.

#### Profile: sysadmin

| Journey             | Debug Container Behavior               |
|---------------------|----------------------------------------|
| Node                | sets privileged; uses host namespaces  |
| Pod Copy            | sets privileged on debugging container |
| Ephemeral Container | sets privileged on ephemeral container |

This profile offers elevated privileges for system debugging.

Probes and labels are be stripped from Pod copies.

#### Profile: netadmin

| Journey             | Debug Container Behavior                                                          |
|---------------------|-----------------------------------------------------------------------------------|
| Node                | sets `NET_ADMIN` and `NET_RAW`; uses host namespaces                              |
| Pod Copy            | sets `NET_ADMIN` and `NET_RAW` on debugging container; sets shareProcessNamespace |
| Ephemeral Container | sets `NET_ADMIN` and `NET_RAW` on ephemeral container                             |

This profile offers elevated privileges for network debugging.

Probes and labels are be stripped from Pod copies.

#### Default Profile

In order to maintain backwards compatibility the `legacy` profile will be the default profile until 1.35.
When `--profile` is not specified `kubectl debug` will print a warning about the upcoming change in behavior.

From 1.36, `general` will be the default profile. `legacy` profile will entirely be removed in 1.39.

#### Future Improvements

It might be possible to support user-configurable profiles, but it's not a goal of
this KEP, and we have no plans to implement it.

The [PodSecurity] KEP mentions a couple of options for "break glass" functionality to allow
bypassing security policy for debugging purposes. If a standard emerges for break glass, `kubectl
debug` should be updated to support it.

### User Stories

#### Operations

Alice runs a service "neato" that consists of a statically compiled Go binary
running in a minimal container image. One of its pods is suddenly having
trouble connecting to an internal service. Being in operations, Alice wants to
be able to inspect the running pod without restarting it, but she doesn't
necessarily need to enter the container itself. She wants to:

1.  Inspect the filesystem of target container
2.  Execute debugging utilities not included in the container image
3.  Initiate network requests from the pod network namespace

This is achieved by running a new "debug" container in the pod namespaces. Her
troubleshooting session might resemble:

```
% kubectl debug -it --image debian neato-5thn0 -- bash
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

Alice discovers that the cluster's DNS service isn't responding.

#### Debugging

Bob is debugging a tricky issue that's difficult to reproduce. He can't
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

Carol is a security engineer tasked with running security audits across all of
her company's running containers. Even though her company has no standard base
image, she's able to audit all containers using:

```
% for pod in $(kubectl get -o name pod); do
    kubectl debug --image gcr.io/neato/security-audit -p $pod /security-audit.sh
  done
```

#### Technical Support

Dan's team provides support for his company's multi-tenant cluster. He can
access the Kubernetes API (as a viewer) on behalf of the users he's supporting,
but he does not have administrative access to nodes or a say in how the
application image is constructed. When someone asks for help, Dan's first step
is to run his team's autodiagnose script:

```
% kubectl debug --image=k8s.gcr.io/autodiagnose nginx-pod-1234
```

### Notes/Constraints/Caveats

1.  There's an output stream race inherent to creating then attaching a
    container which causes output generated between the start and attach to go
    to the log rather than the client. This is not specific to Ephemeral
    Containers and exists because Kubernetes has no mechanism to attach a
    container prior to starting it. This larger issue will not be addressed by
    Ephemeral Containers, but Ephemeral Containers would benefit from future
    improvements or work arounds.

### Risks and Mitigations

1.  There are no guaranteed resources for ad-hoc troubleshooting. If
    troubleshooting causes a pod to exceed its resource limit it may be evicted.
    This risk can be removed once support for pod resizing has been implemented.

## Design Details

### Full `kubectl debug` Arguments

```
Debug cluster resources using interactive debugging containers.

 'debug' provides automation for common debugging tasks for cluster objects identified by resource and name. Pods will
be used by default if resource is not specified.

 The action taken by 'debug' varies depending on what resource is specified. Supported actions include:

  *  Workload: Create a copy of an existing pod with certain attributes changed, for example changing the image tag to a new version.
  *  Workload: Add an ephemeral container to an already running pod, for example to add debugging utilities without restarting the pod.
  *  Node: Create a new pod that runs in the node's host namespaces and can access the node's filesystem.

 Alpha disclaimer: command line flags may change

Examples:
  # Create an interactive debugging session in pod mypod and immediately attach to it.
  # (requires the EphemeralContainers feature to be enabled in the cluster)
  kubectl debug mypod -it --image=busybox

  # Create a debug container named debugger using a custom automated debugging image.
  # (requires the EphemeralContainers feature to be enabled in the cluster)
  kubectl debug --image=myproj/debug-tools -c debugger mypod

  # Create a debug container as a copy of the original Pod and attach to it
  kubectl debug mypod -it --image=busybox --copy-to=my-debugger

  # Create a copy of mypod named my-debugger with my-container's image changed to busybox
  kubectl debug mypod --image=busybox --container=my-container --copy-to=my-debugger -- sleep 1d

  # Create an interactive debugging session on a node and immediately attach to it.
  # The container will run in the host namespaces and the host's filesystem will be mounted at /host
  kubectl debug node/mynode -it --image=busybox

Options:
      --arguments-only=false: If specified, everything after -- will be passed to the new container as Args instead of
Command.
      --attach=false: If true, wait for the container to start running, and then attach as if 'kubectl attach ...' were
called.  Default false, unless '-i/--stdin' is set, in which case the default is true.
  -c, --container='': Container name to use for debug container.
      --copy-to='': Create a copy of the target Pod with this name.
      --env=[]: Environment variables to set in the container.
      --image='': Container image to use for debug container.
      --image-pull-policy='': The image pull policy for the container.
      --quiet=false: If true, suppress informational messages.
      --replace=false: When used with '--copy-to', delete the original Pod
      --same-node=false: When used with '--copy-to', schedule the copy of target Pod on the same node.
      --share-processes=true: When used with '--copy-to', enable process namespace sharing in the copy.
  -i, --stdin=false: Keep stdin open on the container(s) in the pod, even if nothing is attached.
      --target='': When debugging a pod, target processes in this container name.
  -t, --tty=false: Allocate a TTY for the debugging container.

Usage:
  kubectl debug (POD | KIND '/' NAME) --image=image [ -- COMMAND [args...] ]

Use "kubectl options" for a list of global command-line options (applies to all commands).
```

### Test Plan

#### Alpha milestones

In addition to standard unit tests for `kubectl`, the `debug` command will be
released as a `kubectl alpha` subcommand, signaling users to expect instability.
During the alpha phase we will gather feedback from users that we expect will
improve the design of `debug` and identify the Critical User Journeys we should
test prior to Alpha -> Beta graduation.

#### Beta milestones

For Beta release, the following user journeys will have integration tests in the
`test/cmd` package:

  - [Pod Troubleshooting by Copy](#pod-troubleshooting-by-copy)
  - [Node Troubleshooting with Privileged Containers](#node-troubleshooting-with-privileged-containers)

Additionally we'll review unit tests to ensure they're complete before
graduation.

#### GA milestones

If the `EphemeralContainers` feature has reached beta, we will add an
integration test for [Pod Troubleshooting with Ephemeral Debug Container
](#pod-troubleshooting-with-ephemeral-debug-container).

### Graduation Criteria

#### Alpha -> Beta Graduation

- [x] A task on https://kubernetes.io/docs/tasks/ describes how to troubleshoot
  a running pod using Ephemeral Containers.
- [ ] A survey sent to early adopters doesn't reveal any major shortcomings.
- [x] Test plan is amended to address the most common user journeys.
- [ ] Test plan Beta milestones reached.

#### Beta -> GA Graduation

- [x] Test plan GA milestones reached
- [x] User feedback gathered over 2 release cycles.
- [x] 3 external articles suggest using `kubectl debug`
  - https://collabnix.com/debug-your-kubernetes-applications-made-easy-with-kubectl-debug-tool/
  - https://www.linkedin.com/pulse/debug-kubernetes-applications-kubectl-cuong-nguyen-duc
  - https://medium.com/datamindedbe/debugging-running-pods-on-kubernetes-2ba160c47ef5

### Upgrade / Downgrade Strategy

This functionality is contained entirely within `kubectl` and shares its
strategy. No configuration changes are required.

### Version Skew Strategy

`kubectl debug` makes use the following recent features:

- Process namespace sharing (alpha: 1.10, beta: 1.12, GA: 1.17)
- Ephemeral containers (alpha: 1.16)

Not all functionality in `kubectl debug` requires these features. If an
invocation requires a feature that is not enabled in the cluster, the api server
will reject the pod creation request.

Special consideration is given to the Ephemeral Containers feature since this
feature will not be enabled on most clusters while it is in alpha. The `kubectl
debug -h` displays `(requires the EphemeralContainers feature to be enabled in
the cluster)` for examples that require ephemeral containers.

Since ephemeral containers use a dedicated subresource, the api server will
return a 404 when the feature is disabled. When this happens for a target that
exists, kubectl prints `ephemeral containers are disabled for this cluster`.

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
- [x] Other
  - Describe the mechanism: A new command in `kubectl alpha`
  - Will enabling / disabling the feature require downtime of the control
    plane? no
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? no

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

It's a new command so there's no default behavior in kubectl. If a user
has installed a plugin named "debug", that plugin will be masked by the
new `kubectl debug` command. This is a known issue with kubectl plugins,
and it's being addressed separately by sig-cli, likely by detecting this
condition and printing a warning.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, you could roll back to a previous release of `kubectl` and any pods
created by `kubectl debug` would still be accessible via other `kubectl`
commands.

###### What happens if we reenable the feature if it was previously rolled back?

You can create pods again

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

No, because it cannot be disabled or enabled in a single release

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

The feature is encapsulated entirely within the kubectl binary, so rollout is
an atomic client binary update. Pods created by the new command may be
manipulated by any version of `kubectl`, so there are no version dependencies.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

There's no need for a rollback unless `kubectl` is not working at all.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

No, there's no need.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No

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

Since it uses the standard core API, there's no way to determine whether a
pod or ephemeral container was created by `kubectl debug` or manually by a
user.

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
- [x] Other (treat as last resort)
  - Details: There's no running service.

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

There's no running service.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [x] Other (treat as last resort)
  - Details: There's no running service.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

It would be easy to add a "created by kubectl debug" annotation to a newly
created pod, but we don't want to preemptively add features that are only
theoretically useful.

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

No.

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

- API call type (e.g. PATCH pods):
  GET/CREATE/PATCH pods, GET nodes

- estimated throughput:
  Negligible, because it's human initiated. There will be 1 read + 1 mutate
  per `kubectl debug` command. At that point there's a new pod for the system
  to manage.

- originating component(s):
  kubectl

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

One new Pod or EphemeralContainer when initiated by a user.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

Creating a pod will use more resources, but only when initiated by a user.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No.

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

`kubectl` is not resilient to API server unavailability.

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

This command creates objects using the core API. Writing a Playbook of how to
respond when the system is not creating core Kinds is outside the scope of
this KEP.

###### What steps should be taken if SLOs are not being met to determine the problem?

Definitely stop running `kubectl debug`.

## Implementation History

- *2019-08-06*: Initial KEP draft
- *2019-12-05*: Updated KEP for expanded debug targets.
- *2020-01-09*: Updated KEP for debugging nodes and mark implementable.
- *2020-01-15*: Added test plan.
- *1.18*: Features released in `kubectl alpha`
  - [Pod Troubleshooting with Ephemeral Debug Container](#pod-troubleshooting-with-ephemeral-debug-container)
- *1.19*: Features released in `kubectl alpha`
  - [Pod Troubleshooting by Copy](#pod-troubleshooting-by-copy)
  - [Node Troubleshooting with Privileged Containers](#node-troubleshooting-with-privileged-containers)
- *2020-09-20*: Updated KEP to reflect actual implementation details.
- *2020-09-23*: Update KEP for mutating multiple container images in debug-by-copy.
- *2020-09-24*: Update KEP for Production Readiness and beta graduation.
- *2024-01-16*: Promote kubectl debug to GA
- *2025-10-02*: Update KEP to drop auto profile and default general

## Alternatives

An exhaustive list of alternatives to ephemeral containers is included in the
[Ephemeral Containers KEP].

[Ephemeral Containers KEP]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/277-ephemeral-containers/README.md
