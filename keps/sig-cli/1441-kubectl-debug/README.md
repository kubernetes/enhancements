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
    - [Proposed Ephemeral Debug Arguments](#proposed-ephemeral-debug-arguments)
  - [Pod Troubleshooting by Copy](#pod-troubleshooting-by-copy)
    - [Creating a Debug Container by copy](#creating-a-debug-container-by-copy)
    - [Modify Application Image by Copy](#modify-application-image-by-copy)
  - [Node Troubleshooting with Privileged Containers](#node-troubleshooting-with-privileged-containers)
  - [User Stories](#user-stories)
    - [Operations](#operations)
    - [Debugging](#debugging)
    - [Automation](#automation)
    - [Technical Support](#technical-support)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
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

[Ephemeral Containers]: https://git.k8s.io/enhancements/keps/sig-node/20190212-ephemeral-containers.md

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
`kubectl debug` will automatically attach to the console of the newly created
ephemeral container and will default to creating containers with `Stdin` and
`TTY` enabled.

These may be disabled via command line flags.

#### Proposed Ephemeral Debug Arguments

```
% kubectl help debug
Execute a container in a pod.

Examples:
  # Start an interactive debugging session with a debian image
  kubectl debug mypod --image=debian

  # Run a debugging session in the same namespace as target container 'myapp'
  # (Useful for debugging other containers when shareProcessNamespace is false)
  kubectl debug mypod --target=myapp

Options:
  -a, --attach=true: Automatically attach to container once created
  -c, --container='': Container name.
  -i, --stdin=true: Pass stdin to the container
  --image='': Required. Container image to use for debug container.
  --target='': Target processes in this container name.
  -t, --tty=true: Stdin is a TTY

Usage:
  kubectl debug (POD | TYPE/NAME) [-c CONTAINER] [flags] -- COMMAND [args...] [options]

Use "kubectl options" for a list of global command-line options (applies to all commands).
```

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
  --copy-labels=false: When used with `--copy-to`, specifies whether labels
                       should also be copied. Note that copying labels may cause
                       the copy to receive traffic from a service or a replicaset
                       to kill other Pods.
  --delete-old=false: When used with `--copy-to`, delete the original Pod.
  --edit=false: Open an editor to modify the generated Pod prior to creation.
  --same-node=false: Schedule the copy of target Pod on the same node.
  --share-processes=true: When used with `--copy-to`, enable process namespace
                          sharing in the copy.
```

The modification `kubectl debug` makes to `Pod.Spec.Containers` depends on the
value of the `--container` flag.

#### Creating a Debug Container by copy

If a user does not specify a `--container` or specifies one that does not exist,
then the user is instructing `kubectl debug` to create a new Debug Container in
the Pod copy.

```
Examples:
  # Create a copy of 'mypod' with a new debugging container and attach to it
  kubectl debug mypod --copy-to=mypod-debug --image=debian --attach -- bash
```

#### Modify Application Image by Copy

If a user specifies a `--container`, then they are instructing `kubectl debug` to
create a copy of the target pod with a new image for one of the containers.

```
Examples:
  # Create a copy of 'mypod' with the debugging image for container 'app'
  kubectl debug mypod --copy-to=mypod-debug --image=myapp-image:debug --container=myapp -- myapp --debug=5
```

Note that the Pod API allows updates of container images in-place, so
`--copy-to` is not necessary for this operation. `kubectl debug` isn't necessary
to achieve this -- it can be done today with patch -- but `kubectl debug` could
implement it as well for completeness.

### Node Troubleshooting with Privileged Containers

When invoked with a node as a target, `kubectl debug` will create a new
pod with the following fields set:

* `nodeName: $target_node`
* `hostIPC: true`
* `hostNetwork: true`
* `hostPID: true`
* `restartPolicy: Never`

Additionally, `/` on the node will be mounted as a HostPath volume.

```
Examples:
  # Start an interactive debugging session on mynode with a debian image
  kubectl debug node/mynode --image=debian

Options:
  -a, --attach=true: Automatically attach to container once created
  -c, --container='': Container name.
  -i, --stdin=true: Pass stdin to the container
  --image='': Required. Container image to use for debug container.
  -t, --tty=true: Stdin is a TTY
```

### User Stories

#### Operations

Alice runs a service "neato" that consists of a statically compiled Go binary
running in a minimal container image. One of its pods is suddenly having
trouble connecting to an internal service. Being in operations, Alice wants to
be able to inspect the running pod without restarting it, but she doesn't
necessarily need to enter the container itself. She wants to:

1.  Inspect the filesystem of target container
1.  Execute debugging utilities not included in the container image
1.  Initiate network requests from the pod network namespace

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

### Test Plan

In addition to standard unit tests for `kubectl`, the `debug` command will be
released as a `kubectl alpha` subcommand, signaling users to expect instability.
During the alpha phase we will gather feedback from users that we expect will
improve the design of `debug` and identify the Critical User Journeys we should
test prior to Alpha -> Beta graduation.

### Graduation Criteria

#### Alpha -> Beta Graduation

- [ ] Ephemeral Containers API has graduated to Beta
- [ ] A task on https://kubernetes.io/docs/tasks/ describes how to troubleshoot
  a running pod using Ephemeral Containers.
- [ ] A survey sent to early adopters doesn't reveal any major shortcomings.
- [ ] Test plan is amended to address the most common user journeys.

#### Beta -> GA Graduation

- [ ] Ephemeral Containers are GA

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
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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

```
<<[UNRESOLVED copied over from template and needs to be filled. ]>>

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

<<[/UNRESOLVED]>>
```

## Implementation History

- *2019-08-06*: Initial KEP draft
- *2019-12-05*: Updated KEP for expanded debug targets.
- *2020-01-09*: Updated KEP for debugging nodes and mark implementable.
- *2020-01-15*: Added test plan.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

An exhaustive list of alternatives to ephemeral containers is included in the
[Ephemeral Containers KEP].

[Ephemeral Containers KEP]: https://git.k8s.io/enhancements/keps/sig-node/20190212-ephemeral-containers.md
