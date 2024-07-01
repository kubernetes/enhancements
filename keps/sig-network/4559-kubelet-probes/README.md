# KEP-4559: Pod ProbeHandler/LifecycleHandler v2

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
    - [RESOLVED Summary discussions (preserving for history)](#resolved-summary-discussions-preserving-for-history)
    - [UNRESOLVED Summary discussions](#unresolved-summary-discussions)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API Changes to Fix the SSRF](#api-changes-to-fix-the-ssrf)
  - [Kubelet/Runtime Changes to Fix the NetworkPolicy Hole](#kubeletruntime-changes-to-fix-the-networkpolicy-hole)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Goroutines vs. Network Namespaces](#goroutines-vs-network-namespaces)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

There are several problems with the existing Pod liveness, readiness,
and startup probes (listed based on severity):

1. **Probe-based Blind SSRF Attacks**: The current definition of TCP and
  HTTP probes allows the user to specify an alternative hostname/IP to
  connect to rather than the pod IP. (The expected use is for sending a
  probe via a HostPort, NodePort, or LoadBalancer IP.) But this allows a
  "blind SSRF" (Server-Side Request Forgery) attack, in which a pod can
  trick kubelet into sending an HTTP GET request to an arbitrary URL (or
  portscanning TCP ports on arbitrary hosts). ([kubernetes #99425])

2. **Network Probe Performance (?)**: In the past, there have been problems
  with probes consuming unexpectedly large amounts of kubelet resources
  ([kubernetes #89898], "Sometime Liveness/Readiness Probes fail because
  of net/http: request canceled while waiting for connection
  (Client.Timeout exceeded while awaiting headers)"). This has been
  mitigated to some extent ([kubernetes #115143])... it's not clear if
  this is still considered a problem.

3. **Dual-Stack Pods**: Probes are currently defined to use the pod's
  primary IP family, but there are cases where you might want to do an
  IPv6 probe in an IPv4-primary dual-stack cluster. ([kubernetes
  #101324])

4. **The NetworkPolicy Hole**: TCP, HTTP, and GRPC probes are expected
  to travel over the network from kubelet to the pod. This, of course,
  only works if there is nothing in the pod network blocking kubelet's
  access to those pods. When we added NetworkPolicy to Kubernetes, we
  didn't want users to have to manually add rules allowing access from
  Kubelet (and didn't provide any way for them to write such a rule
  anyway), so we just said that kubelet probes are not subject to
  NetworkPolicy, and the NetworkPolicy implementation needs to make
  that work. In almost all cases, this is implemented by allowing
  _all_ node-level processes and host-network pods on the node to
  reach _all_ ports on _all_ pod-network pods on the node, which users
  probably do not want (and may be unaware of). SIG Network would like
  to avoid continuing to have this problem in [AdminNetworkPolicy] and
  any [future successor to NetworkPolicy].

5. **Constraints on Network Architecture Imposed by Pod Probes**: The
  fact that kubelet must be able to reach every pod on its node
  implies that every pod on a given node must have a distinct IP
  address. While that is certainly required anyway if every pod in the
  cluster is expected to be able to be able to talk to every other
  pod, there are clusters where that is actually an "anti-requirement"
  (e.g., [multi-tenant scenarios]), and in some of these cases, people
  have wanted to reuse the same pod IPs for different pods. Although
  it would be possible to deploy a Kubernetes cluster with a CNI
  plugin, service proxy and NetworkPolicy implementation that properly
  handle non-unique pod IPs, there is currently no way to make
  network-based pod probes work correctly in such an environment.

  The proposed [Multi-Network feature] raises similar issues, in that
  in addition to the possibility of networks with overlapping IPs, it
  also potentially involves networks that cannot be reached from the
  host-network namespace.

- **Unclear Semantics for Network Probes**: The intention behind
  introducing liveness probe was to answer "is the server process
  in the pod alive"? and readiness probe was to answer "is the server
  ready to accept traffic"? However since this is achieved by checking
  connectivity to the container, they get mis-interpreted semantically
  to mean "is the pod reachable over the network?" This has
  turned out to not really work in many cases. (In particular, the
  fact that kubelet probes are exempt from NetworkPolicy means that a
  probe may succeed even when no other pods would be able to reach the
  pod because [the NetworkPolicy has not yet been fully programmed].)
  This led to the creation of [readiness gates] to provide better pod
  readiness information.

- **Exec Probe Performance (?)**: [Exec probes are much, much slower than
  network probes]. Apparently this slowness is a somewhat inevitable
  side effect of how exec probes work in an OCI runtime context?
  Possibly because it requires making multiple synchronous calls? No known
  issues were opened outside of the blogpost but anyway, it's possible
  this could be improved.

- **Defining localhost probes doesn't work as expected**: When defining
  `.Host` to be `127.0.0.1` the user might be expecting this to be confined
  to the pod's own localhost but it is not. The entire node can be exposed
  by doing so which can be surprising to the users.

See further discussion in [kubernetes #102613].

#### RESOLVED Summary discussions (preserving for history)
- Figure out if there are any concerns with non-namespace-based Linux
  runtimes (kata, kubevirt, etc): There are no concerns here because both kata containers and [kubevirt] use network namespaces. They expect the CNI to set up the pod networking. Then their agent creates the required networking infrastructure inside this pod network namespace to connect to the VM interface

#### UNRESOLVED Summary discussions

- Figure out details of exec probe performance problem.

- Figure out if network probe performance is still a problem.

- Figure out if there are any Windows-specific concerns (e.g, that our
  proposal may be more difficult on Windows). Alternatively, it may
  turn out that removing the constraint that kubelet needs to be able
  to talk to all pods would simplify things on Windows?

[kubernetes #99425]: https://github.com/kubernetes/kubernetes/issues/99425
[AdminNetworkPolicy]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/2091-admin-network-policy/README.md
[future successor to NetworkPolicy]: https://github.com/kubernetes-sigs/network-policy-api/issues/136
[multi-tenant scenarios]: https://github.com/kubernetes/kubernetes/issues/31824
[Multi-network feature]: https://github.com/kubernetes/enhancements/issues/3698
[kubernetes #101324]: https://github.com/kubernetes/kubernetes/issues/101324
[the NetworkPolicy has not yet been fully programmed]: https://kubernetes.io/docs/concepts/services-networking/network-policies/#pod-lifecycle
[readiness gates]: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#pod-readiness-gate
[Exec probes are much, much slower than network probes]: https://medium.com/netcracker/exec-probes-a-story-about-experiment-and-relevant-problems-12de616c0a76
[kubernetes #89898]: https://github.com/kubernetes/kubernetes/issues/89898
[kubernetes #115143]: https://github.com/kubernetes/kubernetes/pull/115143
[kubernetes #102613]: https://github.com/kubernetes/kubernetes/issues/102613
[kubevirt]: https://github.com/kubevirt/kubevirt/blob/main/docs/devel/networking.md#vmi-networking

## Motivation

### Goals

```
<<[UNRESOLVED goals ]>>

The exact details of the goals depend on whether we decide to
deprecate the existing probe types, or just change their semantics.

Also, there may be additional goals depending on the resolution of the
UNRESOLVED section in the Summary.

<<[/UNRESOLVED]>>
```

- Continue to backward-compatibly support all existing probe handlers
  and lifecycle handlers by default (at least for a while).

- Deprecate the [`TCPSocketAction.Host`] and [`HTTPGetAction.Host`]
  fields in pod probes (since they allow the SSRF attack and have no
  other compelling use case). Also deprecate the [`HTTPGetAction.Host`]
  field in pod lifecycle hooks.

    - Allow administrators to use [Pod Security admission] to block
      Pods that have probes and lifecycle hooks using those fields.

- Move to a system where TCP, HTTP, GRPC pod probes and HTTP lifecycle
  hooks are run inside the pod network namespace rather than being sent
  over the pod network, so that (among other things) kubelet does not
  need to have any particular pod-network security privileges.

    - (For pods that really have a strong use case for checking
      connectivity in some non-standard way, such as via a load
      balancer IP, it would still be possible for a user to write an
      exec probe that uses `curl` or the like to connect to that IP
      from the pod itself.)

    - If we implement this by creating a new set of probe/lifecycle handler
      types, rather than by modifying the semantics of the existing probes,
      then it should be possible to block the old probe types via Pod
      Security admission, so that administrators can create
      NetworkPolicies that don't include the kubelet probe hole.

- Allow dual-stack / wrong-single-stack probing, either based on
  information in the probe definition (e.g. `ipFamily: IPv6` or
  `ipFamilyPolicy: RequireDualStack`) or by always doing "[Happy
  Eyeballs]" (probing IPv4 and IPv6 in parallel and only requiring one
  to succeed).

- Make it easy for users who aren't doing "weird" things (eg,
  readiness probe via LoadBalancer) to transition from the old probe
  system to the new one.

    - This might mean that they don't need to do anything, or it might
      mean that there's an easy transition, like just changing
      `tcpSocket:` to `tcp:` in their pod's probe definition.

[`TCPSocketAction.Host`]: https://github.com/kubernetes/kubernetes/blob/v1.28.0/staging/src/k8s.io/api/core/v1/types.go#L2248
[`HTTPGetAction.Host`]: https://github.com/kubernetes/kubernetes/blob/v1.28.0/staging/src/k8s.io/api/core/v1/types.go#L2219
[Pod Security admission]: https://kubernetes.io/docs/concepts/security/pod-security-admission/
[Happy Eyeballs]: https://tools.ietf.org/html/rfc8305

### Non-Goals

- Adding entirely-new probe types.

- Changing the definition of NetworkPolicy v1 to remove the "kubelet
  can access all local pods" hole, even in the case where the cluster
  is configured in a way to forbid the use of probes that require the
  hole. Even if _kubelet_ no longer needs to access pods, some users
  likely depend on that rule to allow access to some pods from other
  host-level processes, and there is no other way in NetworkPolicy v1
  to write such a rule if we remove the built-in one. (However,
  AdminNetworkPolicy and any future successor to NetworkPolicy should
  be designed to not have this rule by default.)

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### API Changes to Fix the SSRF

Regardless of what else we do, we will deprecate the
`TCPSocketAction.Host` and `HTTPGetAction.Host` fields, and allow them
to be blocked by Pod Security admission. Deprecation here will include
documenting the field in the API as deprecated and added a warning to
users when they attempt to use these fields.

Beyond that, there are two (and a half) possible plans:

1. Change the semantics of the existing `TCPSocketAction`,
   `HTTPGetAction`, and `GRPCAction` probe types and say that
   henceforth the probes will be done in the pod network namespace
   rather than being sent over the pod network (unless you use the
   deprecated `Host` field to request an alternate probe endpoint; we
   probably shouldn't just move those kinds of probes into the pod
   network namespace since that might require a "hairpin" connection
   to work when it wasn't required before).

2. Leave the existing probe types with their existing semantics, but
   deprecate them and add new probe types with the
   inside-the-network-namespace semantics. (`v1.ProbeHandler`
   currently has `httpGet` and `tcpSocket` fields; these could
   potentially become just `http` and `tcp`. It's less obvious what
   `grpc` could be renamed to.)

2½. Deprecate-and-replace `httpGet` and `tcpSocket` as in (2), but
    change the semantics of `grpc` as in (1), since GRPC probes are
    already more constrained than the TCP and HTTP probes anyway (and
    I don't have a good suggestion for renaming them).

### Kubelet/Runtime Changes to Fix the NetworkPolicy Hole

Again, a few possible plans. Some of these ideas are pretty bad, but
may have good ideas within them that could be remixed into other
plans...

1. Rather than performing pod probes from the node IP, each kubelet
   could launch a "probing pod", and perform probes from there. A
   created-by-default AdminNetworkPolicy could force all pods to
   accept connections from the kubelet probe pods. This is perhaps
   _slightly_ better than the current situation since allowing
   connections from the probe pods would not allow any unintended
   additional traffic from anyone else the way that allowing
   connections from the node IP does.

     - The simplest / most portable way to do this would be to have
       the "probing pod" be an ordinary pod-network pod, though (given
       current AdminNetworkPolicy rules) this would means that all
       pods on all nodes would have to accept connections from all
       probing pods, meaning an attacker who was able to compromise
       any probing pod would then be able to connect to any pod in the
       cluster.

     - Alternatively, the "probing pod" could be a static pod with a
       special node-local IP, allocated independently of the normal
       pod network. (Handwave handwave handwave...)

2. Kubelet could perform probes inside pods without needing any
   changes to the runtime, by attaching an ephemeral container to each
   pod, containing some `kubelet-probe` binary that would run the
   probes and report the results back to kubelet (by some means). Most
   plausibly, this would be a single container that would persist
   through the life of the pod, periodically running probes, rather
   than being a new ephemeral container attached each time kubelet
   needed to perform a probe. Kubelet could perhaps attach such a
   container to the pod at construction time, via CRI calls, without
   exposing the container to the kubernetes API, like it does with the
   pause / sandbox / infrastructure container. (In fact, the probes
   could even be done as part of the existing pause / sandbox /
   infrastructure container.)

3. If we can improve the performance of exec probes, then kubelet
   could simply implement the network probes as exec probes, though
   this would again require behind-the-scenes modifications to the pod
   via CRI, to mount a volume containing the `kubelet-probe` binary so
   that it's available to the exec probe.

4. We could add a new method to CRI to allow kubelet to do the
   equivalent of `nsenter -n` / `ip netns exec` without any of the
   other overhead of an exec probe. It is not clear that this concept
   would map well to Windows containers or even to
   virtualization-based container runtimes on Linux. In the worst
   case, it could fall back to something like (3).

5. We could add a new method to CRI to allow kubelet to request that
   the runtime perform a probe inside the container via some means
   appropriate to the runtime. If this is not simply a wrapper around
   exec probes, then this implies that CRI will need to understand
   each type of probe supported by Kubernetes (i.e., it will need to
   support every feature of `HTTPGetAction`, `TCPSocketAction` and
   `GRPCAction`.) In the future, adding new probe types (for example,
   a UDP probe) would require adding support to both `Pod`/kubelet and
   to CRI.

     - If CRI is going to need to understand that much of the
       semantics of Kubernetes probes, then it might make sense to
       just _completely_ move probes into CRI, and have CRI simply
       report back as part of the pod status whether the pod is live,
       ready, etc, according to its probes. This might allow for
       better optimization of probes on the CRI side.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

#### Goroutines vs. Network Namespaces

Moving pod probes into the pod network namespace will run into the
standard difficulties with using kernel namespaces from golang code.

The `setns()` syscall in Linux changes the namespace of a thread; this
does not always interact well with goroutines in golang. In
particular, you can ensure that the current goroutine won't be moved
to another thread, and you can ([since go 1.10]) ensure that other
goroutines won't be moved onto your thread, but you cannot (and in the
general case would not want to) guarantee that new goroutines spawned
from your goroutine will run in the current thread. This means that it
is unsafe for a golang function which has called `setns()` to invoke
any code which it is not _certain_ will perform all of its work in the
current goroutine. In particular, [`net.Dial` does not guarantee that
it will establish the connection from the same goroutine it was called
from] so it is unlikely that a golang-based CRI implementation would
be able to run probes directly from its main process.

[since go 1.10]: https://github.com/golang/go/commit/2595fe7fb6
[`net.Dial` does not guarantee that it will establish the connection from the same goroutine it was called from]: https://github.com/golang/go/issues/44922

### Risks and Mitigations

If we change the semantics of the existing probe types (i.e., just say
that henceforth if you don't specify `TCPSocketAction.Host` or
`HTTPGetAction.Host` then the probe is done in the pod network
namespace rather than going across the network) then we risk breaking
people who are doing sufficiently weird things. (I'm not even sure
what these "sufficiently weird things" would be. One example would be
having the pod recognize the node IP as a source IP so that it can
respond differently to kubelet probes than it would to a "real"
request.) It would be safer (though more annoying) to instead
deprecate the existing probe types and create new ones with the new
semantics.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

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

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

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

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

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
