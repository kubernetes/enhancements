# KEP-614: SCTP Support

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Service with SCTP and Virtual IP](#service-with-sctp-and-virtual-ip)
    - [Headless Service with SCTP](#headless-service-with-sctp)
    - [Service with SCTP without selector](#service-with-sctp-without-selector)
    - [SCTP as container port protocol in Pod definition](#sctp-as-container-port-protocol-in-pod-definition)
    - [SCTP port accessible from outside the cluster](#sctp-port-accessible-from-outside-the-cluster)
    - [NetworkPolicy with SCTP](#networkpolicy-with-sctp)
    - [Userspace SCTP stack](#userspace-sctp-stack)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [SCTP in Services](#sctp-in-services)
      - [Kubernetes API modification](#kubernetes-api-modification)
      - [Services with host level ports](#services-with-host-level-ports)
      - [Services with type=LoadBalancer](#services-with-typeloadbalancer)
    - [SCTP support in Kube DNS](#sctp-support-in-kube-dns)
    - [SCTP in the Pod's ContainerPort](#sctp-in-the-pods-containerport)
    - [SCTP in NetworkPolicy](#sctp-in-networkpolicy)
    - [Interworking with applications that use a user space SCTP stack](#interworking-with-applications-that-use-a-user-space-sctp-stack)
      - [Problem definition](#problem-definition)
      - [The solution in the Kubernetes SCTP support implementation](#the-solution-in-the-kubernetes-sctp-support-implementation)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
    - [Basic tests](#basic-tests)
    - [SCTP Connectivity Tests](#sctp-connectivity-tests)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The goal of the SCTP support feature is to enable the usage of the
SCTP protocol in Kubernetes [Service], [NetworkPolicy], and
[ContainerPort] as an additional protocol value option beside the
current TCP and UDP options. SCTP is an IETF protocol specified in
[RFC4960], and it is used widely in telecommunications network stacks.
Once SCTP support is added as a new protocol option those applications
that require SCTP as L4 protocol on their interfaces can be deployed
on Kubernetes clusters on a more straightforward way. For example they
can use the native kube-dns based service discovery, and their
communication can be controlled in the native NetworkPolicy way.

[Service]: https://kubernetes.io/docs/concepts/services-networking/service/
[NetworkPolicy]: 
https://kubernetes.io/docs/concepts/services-networking/network-policies/
[ContainerPort]:https://kubernetes.io/docs/concepts/services-networking/connect-applications-service/#exposing-pods-to-the-cluster
[RFC4960]: https://tools.ietf.org/html/rfc4960

## Motivation

SCTP is a widely used protocol in telecommunications. It would ease
the management and execution of telecommunication applications on
Kubernetes if SCTP were added as a protocol option to Kubernetes.

### Goals

Add SCTP support to Kubernetes ContainerPort, Service and
NetworkPolicy, so applications running in pods can use the native
kube-dns based service discovery for SCTP based services, and their
communication can be controlled via the native NetworkPolicy way.

It is also a goal to enable ingress SCTP connections from clients
outside the Kubernetes cluster, and to enable egress SCTP connections
to servers outside the Kubernetes cluster.

### Non-Goals

It is not a goal here to add SCTP support to load balancers that are
provided by cloud providers. The Kubernetes side implementation will
not restrict the usage of SCTP as the protocol for the Services with
type=LoadBalancer, but we do not implement the support of SCTP into
the cloud specific load balancer implementations.

It is not a goal to support multi-homed SCTP associations. Such a
support also depends on the ability to manage multiple IP addresses
for a pod, and in the case of Services with ClusterIP or NodePort the
support of multi-homed associations would also require the support of
NAT for multihomed associations in the SCTP related NF conntrack
modules.

## Proposal

### User Stories

#### Service with SCTP and Virtual IP

As a user of Kubernetes I want to define Services with Virtual IPs for
my applications that use SCTP as L4 protocol on their interfaces,so
client applications can use the services of my applications on top of
SCTP via that Virtual IP.

Example:
```
kind: Service
apiVersion: v1
metadata:
  name: my-service
spec:
  selector:
    app: MyApp
  ports:
  - protocol: SCTP
    port: 80
    targetPort: 9376
```

#### Headless Service with SCTP

As a user of Kubernetes I want to define headless Services for my
applications that use SCTP as L4 protocol on their interfaces, so
client applications can discover my applications in kube-dns, or via
any other service discovery methods that get information about
endpoints via the Kubernetes API.

Example:
```
kind: Service
apiVersion: v1
metadata:
  name: my-service
spec:
  selector:
    app: MyApp
  ClusterIP: "None"
  ports:
  - protocol: SCTP
    port: 80
    targetPort: 9376
```
#### Service with SCTP without selector

As a user of Kubernetes I want to define Services without selector for
my applications that use SCTP as L4 protocol on their interfaces, so I
can implement my own service controllers if I want to extend the basic
functionality of Kubernetes.

Example:
```
kind: Service
apiVersion: v1
metadata:
  name: my-service
spec:
  ClusterIP: "None"
  ports:
  - protocol: SCTP
    port: 80
    targetPort: 9376
```

#### SCTP as container port protocol in Pod definition

As a user of Kubernetes I want to define hostPort for the SCTP based
interfaces of my applications

Example:
```
apiVersion: v1
kind: Pod
metadata:
  name: mypod
spec:
  containers:
  - name: container-1
    image: mycontainerimg
    ports:
      - name: diameter
        protocol: SCTP
        containerPort: 80
        hostPort: 80
```

#### SCTP port accessible from outside the cluster

As a user of Kubernetes I want to have the option that client
applications that reside outside of the cluster can access my SCTP
based services that run in the cluster.

Example:
```
kind: Service
apiVersion: v1
metadata:
  name: my-service
spec:
  type: NodePort
  selector:
    app: MyApp
  ports:
  - protocol: SCTP
    port: 80
    targetPort: 9376
```

Example:
```
kind: Service
apiVersion: v1
metadata:
  name: my-service
spec:
  selector:
    app: MyApp
  ports:
  - protocol: SCTP
    port: 80
    targetPort: 9376
  externalIPs:
  - 80.11.12.10
```

#### NetworkPolicy with SCTP

As a user of Kubernetes I want to define NetworkPolicies for my
applications that use SCTP as L4 protocol on their interfaces, so the
network plugins that support SCTP can control the accessibility of my
applications on the SCTP based interfaces, too.

Example:
```
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: myservice-network-policy
  namespace: myapp
spec:
  podSelector:
    matchLabels:
      role: myservice
  policyTypes:
  - Ingress
  ingress:
  - from:
    - ipBlock:
        cidr: 172.17.0.0/16
        except:
        - 172.17.1.0/24
    - namespaceSelector:
        matchLabels:
          project: myproject
    - podSelector:
        matchLabels:
          role: myclient
    ports:
    - protocol: SCTP
      port: 7777
```

#### Userspace SCTP stack

As a user of Kubernetes I want to deploy and run my applications that
use a userspace SCTP stack, and at the same time I want to define SCTP
Services in the same cluster. I use a userspace SCTP stack because of
the limitations of the kernel's SCTP support. For example: it's not
possible to write an SCTP server that proxies/filters arbitrary SCTP
streams using the sockets APIs and kernel SCTP.

### Notes/Constraints/Caveats (Optional)

#### SCTP in Services

##### Kubernetes API modification

The Kubernetes API modification for Services to support SCTP is obvious.

##### Services with host level ports

The kube-proxy and the kubelet starts listening on the defined TCP or
UDP port in case of Servies with ClusterIP or NodePort or externalIP,
and in case of containers with HostPort defined. The goal of this is
to reserve the port in question so no other host level process can use
that by accident. When it comes to SCTP the agreement is that we do
not follow this pattern. That is, Kubernetes will not listen on host
level ports with SCTP as protocol. The reason for this decision is,
that the current TCP and UDP related implementation is not perfect
either, it has known gaps in some use cases, and in those cases this
listening is not started. But no one complained about those gaps so
most probably this port reservation via listening logic is not needed
at all.

##### Services with type=LoadBalancer

For Services with type=LoadBalancer we expect that the cloud
provider's load balancer API client in Kubernetes rejects the requests
with unsupported protocol.

#### SCTP support in Kube DNS

Kube DNS shall support SRV records with "_sctp" as "proto" value.
According to our investigations, the DNS controller is very flexible
from this perspective, and it can create SRV records with any protocol
name. I.e. there is no need for additional implementation to achieve
this goal.

Example:

```
_diameter._sctp.my-service.default.svc.cluster.local. 30 IN SRV 10 100 1234 my-service.default.svc.cluster.local.
```

#### SCTP in the Pod's ContainerPort

The Kubernetes API modification for the Pod is obvious.

We support SCTP as protocol for any combinations of containerPort and hostPort.

#### SCTP in NetworkPolicy

The Kubernetes API modification for the NetworkPolicy is obvious.

In order to utilize the new protocol value the network plugin must support it.

#### Interworking with applications that use a user space SCTP stack

##### Problem definition

A userpace SCTP stack usually creates raw sockets with IPPROTO_SCTP.
And as it is clearly highlighted in the [documentation of raw
sockets]:

> Raw sockets may tap all IP protocols in Linux, even protocols like
> ICMP or TCP which have a protocol module in the kernel. In this
> case, the packets are passed to both the kernel module and the raw
> socket(s).

i.e. if both the kernel module (lksctp) and a userspace SCTP stack are
active on the same node both receive the incoming SCTP packets
according to the current [kernel] logic.

But in turn the SCTP kernel module will handle those packets that are
actually destined to the raw socket as Out of the blue (OOTB) packets
according to the rules defined in [RFC4960]. I.e. the SCTP kernel
module sends SCTP ABORT to the sender, and on that way it aborts the
connections of the userspace SCTP stack.

As we can see, a userspace SCTP stack cannot co-exist with the SCTP
kernel module (lksctp) on the same node. That is, the loading of the
SCTP kernel module must be avoided on nodes where such applications
that use userspace SCTP stack are planned to be run. The SCTP kernel
module loading is triggered when an application starts managing SCTP
sockets via the standard socket API or via syscalls.

In order to resolve this problem the solution was to dedicate nodes to
userspace SCTP applications in the past. Such applications that would
trigger the loading of the SCTP kernel module were not deployed on
those nodes.

##### The solution in the Kubernetes SCTP support implementation

Our main task here is to provide the same node level isolation
possibility that was used in the past: i.e. to provide the option to
dedicate some nodes to userspace SCTP applications, and ensure that
the actions performed by Kubernetes (kubelet, kube-proxy) do not load
the SCTP kernel modules on those dedicated nodes.

On the Kubernetes side we solve this problem so, that we do not start
listening on the SCTP ports defined for Servies with ClusterIP or
NodePort or externalIP, neither in the case when containers with SCTP
HostPort are defined. On this way we avoid the loading of the kernel
module due to Kubernetes actions.

On application side it is pretty easy to separate application pods
that use a userspace SCTP stack from those application pods that use
the kernel space SCTP stack: the usual nodeselector label based
mechanism, or taints are there for this very purpose.

NOTE! The handling of TCP and UDP Services does not change on those
dedicated nodes.

We propose the following solution:

We describe in the Kubernetes documentation the mutually exclusive
nature of userspace and kernel space SCTP stacks, and we would
highlight, that the required separation of the userspace SCTP stack
applications and the kernel module users shall be achieved with the
usual nodeselector or taint based mechanisms.

[documentation of raw sockets]: http://man7.org/linux/man-pages/man7/raw.7.html
[kernel]: https://github.com/torvalds/linux/blob/0fbc4aeabc91f2e39e0dffebe8f81a0eb3648d97/net/ipv4/ip_input.c#L191

### Risks and Mitigations

## Design Details

### Test Plan

Unlike with UDP and TCP, we can't necessarily just test that SCTP
connections work, because some machines won't have the SCTP kernel
module installed (or will have it blocked from being loaded). So we
want to have some tests that are just "make sure kube-proxy, etc are
doing what we expect them to" that can run everywhere, and then
another set of "SCTP connections actually work" tests that are behind
a separate feature tag.

#### Basic tests

These will be tagged `[Feature:SCTP]`.

- `"Pods, Services, and NetworkPolicies can declare SCTP ports"`

  - Just checks that resources can be created using `SCTP` as a
    `Protocol` value.

- `"Pods can declare SCTP HostPorts when using kubenet"`

  - Confirms that adding an SCTP HostPort creates an appropriate
    iptables rule.

  - Will be Skipped if SSH is not available or kubenet is not in use.

- `"Services can declare SCTP ServicePorts when using the iptables proxier"`

  - Confirms that appropriate iptables rules are created for SCTP
    services.

  - Will be Skipped if SSH is not available or the `iptables` proxier
    is not in use.

- `"A NetworkPolicy matching an SCTP port does not allow TCP or UDP
  traffic [Feature:NetworkPolicy]"`

If SSH is available, then each test will also ensure that if `sctp.ko`
is not loaded before the test, then it is also not loaded after the
test.

Since the tests will initially require the `SCTPSupport` feature gate
to be enabled, we will create a periodic job to run just the SCTP
tests (and perhaps a few other network conformance tests, just to
confirm that enabling SCTP does not break other things.) After SCTP
reaches GA, the tests could run as part of the normal presubmit job
and the periodic job could be retired.

#### SCTP Connectivity Tests

These will be tagged "`[Feature:SCTPConnectivity]`". They can be run
by network plugin developers but will not run as part of any of the
standard Kubernetes test runs. They should all be marked
`[Disruptive]` since they may cause the SCTP kernel module to be
loaded, which may interfere with the Basic Tests' "make sure the SCTP
kernel module doesn't get loaded" checks.

We will need to vendor
[github.com/ishidawataru/sctp](https://github.com/ishidawataru/sctp)
and use it from `agnhost` and `e2e.test` to create SCTP client and
server sockets.

- `"A pod can connect directly to another pod via SCTP"`

- `"A pod can connect to another pod via SCTP via a Service IP"`

- `"A pod can connect to another pod via SCTP via a Load Balancer"`,

  - Will be Skipped on bare metal, and clouds that don't support SCTP
    LoadBalancers.

- `"NetworkPolicy can be used to allow or deny traffic to SCTP ports
  [Feature:NetworkPolicy]"`

### Graduation Criteria

#### Alpha -> Beta Graduation

Graduation criteria:

- The e2e tests described in the Test Plan have been written.

- A periodic job has been created to run the "Basic Tests", and it
  passes.

#### Beta -> GA Graduation

Graduation criteria:

- At least 2 out-of-tree network plugins can pass the
  `[Feature:SCTPConnectivity]` tests.

- Both `iptables` and `ipvs` proxiers have been demonstrated to work
  with SCTP.

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

- 2018-06-11 Initial [code PR](https://github.com/kubernetes/kubernetes/pull/64973)
- 2018-06-16 Initial [KEP PR](https://github.com/kubernetes/community/pull/2276)
- 2018-08-24 Initial KEP merged
- 2018-08-28 Initial code merged
- 2018-09-11 [Feature proposal](https://github.com/kubernetes/enhancements/issues/614) filed
- 2018-09-27 Kubernetes 1.12.0 released with Alpha SCTP support
- 2019-10-02 Test Plan and Graduation Criteria added

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
