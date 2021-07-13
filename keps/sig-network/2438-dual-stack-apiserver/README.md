# KEP-2438: Dual-Stack API Server

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1 - External Clients](#story-1---external-clients)
    - [Story 2 - APIServer Tooling](#story-2---apiserver-tooling)
    - [Story 3 - Internal Clients](#story-3---internal-clients)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [kube-apiserver Command-Line Arguments](#kube-apiserver-command-line-arguments)
  - [Service and EndpointSlice Reconciling](#service-and-endpointslice-reconciling)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Handling of the &quot;<code>kubernetes</code>&quot; Service](#handling-of-the--service)
    - [Handling of &quot;<code>kubernetes</code>&quot; Endpoints and EndpointSlices](#handling-of--endpoints-and-endpointslices)
    - [Service vs EndpointSlice Sync Issues](#service-vs-endpointslice-sync-issues)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
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

The initial implementation of [dual-stack networking] in Kubernetes
left the apiserver as a single-stack Service. This moves it to
dual-stack (in dual-stack clusters).

[dual-stack networking]: ../563-dual-stack/README.md

## Motivation

### Goals

- Ensure kube-apiserver can listen dual-stack (subject to appropriate
  feature gates):

    - Allow specifying dual-stack `--bind-address` on the command
      line.

    - Autodetect dual-stack bind addresses if possible when no
      explicit `--bind-address` is specified.

- Allow kube-apiserver to advertise dual-stack functionality to other
  components (subject to appropriate feature gates):

    - Allow specifying dual-stack `--advertise-address` on the command
      line (and likewise, autodetecting dual-stack advertise addresses
      from the bind address when `--advertise-address` is not
      explicitly specified).

    - Publish dual-stack `EndpointSlice`s for the "`kubernetes`"
      Service when dual-stack apiserver IPs are available.

    - When configured with both dual-stack bind/advertise addresses
      and dual-stack service CIDRs, create the "`kubernetes`" service as
      `ipFamilyPolicy: RequireDualStack`, and assign it the first IP
      address in both the IPv4 CIDR range and the IPv6 CIDR range.

### Non-Goals

- Trying to make `KUBERNETES_SERVICE_HOST` dual-stack.

## Proposal

### User Stories

#### Story 1 - External Clients

As a cluster administrator, I want my apiserver to be reachable by
both IPv4-only and IPv6-only clients outside the cluster, without them
needing to use a proxy or NAT mechanism.

#### Story 2 - APIServer Tooling

As a Kubernetes distributor or service provider, I want to be able to
reliably determine the IP addresses being used by the apiserver in a
user-created cluster, so that I can correctly create and configure
related infrastructure (eg, TLS certificates, endpoints for external
load balancers or DNS names, etc).

#### Story 3 - Internal Clients

As a developer, I want to run a pod in a dual-stack cluster that is
single-stack of the secondary IP family, and have it be able to
connect to the apiserver.

This is kind of an odd case, but [it has been seen in the wild].
(Although note that we are currently only proposing to fix access via
the `kubernetes.default.svc.cluster.local` name, not via
`KUBERNETES_SERVICE_HOST` / `rest.InClusterConfig()`.)

[it has been seen in the wild]: https://github.com/kubernetes/enhancements/issues/563#issuecomment-814560270

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

In theory, an administrator might be running an apiserver on a
dual-stack host but not want it to be exposed as a dual-stack service.
However, like most golang-based servers, the apiserver _listens_ on
both IP families by default unless explicitly configured to only
listen on a single IP, so the administrator in this case would already
be losing, and just not realizing it. Making the apiserver advertise
its secondary IP address in this case would arguably be a security
_improvement_, since it would make it easier for the administrator to
notice that it was not configured how they had wanted.

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

### kube-apiserver Command-Line Arguments

FIXME

### Service and EndpointSlice Reconciling

If the dual-stack apiserver changes take effect all at once when the
feature goes GA, then this could cause Service/EndpointSlice flapping
during the upgrade from the pre-GA release to the GA release.

(Note that the problem is only with `Service` and `EndpointSlice`, not
with `Endpoints` because `Endpoints` is inherently single-stack, so
both old and new apiservers will always agree on which IPs should be
listed there.)

To prevent this, we can make some of the dual-stack apiserver support
code be active even when the feature gate is disabled; the feature
gate will control whether an apiserver in a dual-stack cluster tries
to advertise *itself* as dual-stack, but not whether it allows *other*
apiservers to advertise themselves as dual-stack. Thus:

  - Current/old (eg, 1.21) apiserver behavior, regardless of single-
    or dual-stack configuration:

      - Advertises its own single-stack endpoint IP to etcd

      - Forces the `kubernetes` Service to be single-stack when it
        starts up.

      - Syncs Endpoints and the primary-IP-family EndpointSlice.
        Mistakenly ignores the secondary-IP-family EndpointSlice
        ([kubernetes #101070]) and so could leave behind a stale
        secondary-IP-family EndpointSlice on downgrade.

  - "Half-new" dual-stack apiserver (is running a version that has the
    new dual-stack apiserver code, and has a dual-stack
    `--service-cluster-ip-range`, but is not running with the
    `DualStackAPIServer` feature gate enabled):

      - Advertises its own *single-stack* endpoint IP to etcd (as with
        the old apiserver, because the feature gate is disabled)

      - Forces the `kubernetes` Service to be either single-stack or
        dual-stack, based on whether any other apiservers have
        advertised dual-stack endpoints.

      - Syncs Endpoints and single- or dual-stack EndpointSlices. (If
        any other apiservers have advertised dual-stack endpoints to
        etcd, then it will create a secondary-IP-family EndpointSlice
        with those endpoints (but not its own secondary-IP-family
        address, because the feature gate is disabled). Otherwise, it
        will create a primary-IP-family EndpointSlice only, and clean
        up a stale secondary-IP-family EndpointSlice if it sees one.)

  - New dual-stack apiserver (has the new code, has a dual-stack
    `--service-cluster-ip-range`, and has the `DualStackAPIServer`
    feature gate enabled)

      - Advertises its own dual-stack endpoint IPs to etcd

      - Forces the `kubernetes` Service to be dual-stack on startup

      - Syncs Endpoints and dual-stack EndpointSlices

As long as the "half-new" support goes in two releases before the
feature gate is enabled by default, then upgrades and downgrades
should proceed smoothly. People who enable the feature gate before
then may encounter problems with Service/EndpointSlice flapping during
upgrades, and may need to do some manual cleanup on downgrade.

The new code should also behave like the "half-new dual-stack" case
when an apiserver is configured to be single-stack but it sees that
other apiservers are advertising dual-stack. This will prevent
flapping during a roll-out of a single-stack to dual-stack
configuration change.

[kubernetes #101070]: https://github.com/kubernetes/kubernetes/issues/101070

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

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

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
-->

### Upgrade / Downgrade Strategy

As discussed under ["Service and EndpointSlice
Reconciling"](#service-and-endpointslice-reconciling), we will modify
the apiserver behavior when the feature gate is off to make it
cooperate better with apiservers where the feature gate is on (while
still having no change in behavior in a cluster where _all_ apiservers
have the feature gate off). Assuming that the feature remains alpha /
disabled-by-default for at least 2 releases, this should ensure that
all non-early-adopters get smooth upgrades and downgrades.

For early adopters, or if we want to only have the feature be alpha
for a single release, and not have completely smooth behavior for
users with N-2 version skew, there are a few potential problems:

#### Handling of the "`kubernetes`" Service

kube-apiserver attempts to force the "`kubernetes`" Service definition
to match its expectations on startup, but does not attempt to
reconcile it after that point. Old/current apiservers always expect
the Service to be single-stack.

Thus, during an upgrade in which some (new) apiservers are trying to
use the `DualStackAPIServer` feature gate, and some (old) apiservers
predate the new feature, whichever apiserver started last will
determine whether the Service is currently single or dual stack.

In a simple upgrade or downgrade, this would result in the Service
definition changing when the first apiserver was upgraded/downgraded,
and then not changing after that. In a more complicated
upgrade/downgrade scenario, there might be some flapping; eg, the
first master upgrades and changes the Service to dual-stack, but then
the second master reboots before upgrading, causing the old apiserver
to restart and revert the Service definition back to its single-stack
form, before then being upgraded and changing it back to dual-stack
again.

At no point would the Service be advertised as dual-stack when there
were no dual-stack apiservers, so the Service ought to always be
usable; it just may sometimes be single-stack when it _could_ have
been dual-stack.

(The one way that something could go wrong would be if a spurious
apiserver were to start up after the upgrade/downgrade completed. Eg,
after upgrading all apiservers to a dual-stack release, if an old
single-stack-only apiserver were to start up, but then exit, it might
spuriously change the Service back to single-stack before exiting.
It's not clear why this would happen, and at any rate, it would be
fixed as soon as any of the real apiservers restarted again.)

#### Handling of "`kubernetes`" Endpoints and EndpointSlices

For the "`kubernetes`" `Endpoints` (_not_ `EndpointSlice`), there
should be no flapping; both old and new apiservers will agree at all
times about what endpoints are available _in the cluster's primary IP
family_, and thus will always agree on the contents of the
single-stack `Endpoints` resource.

For `EndpointSlice`, current versions of kube-apiserver create a
single `EndpointSlice` object named "`kubernetes`", and assume that
other apiservers will do the same; they don't look at any other
`EndpointSlice` objects when reconciling. To create dual-stack
EndpointSlices, a new apiserver would need to create two objects with
different names. Even if new apiservers use the same name as the old
apiserver does for the primary-IP-family slice, old apiservers would
never see the slice it creates for the secondary IP family. This means
there would be no flapping, but also means that a stale
secondary-IP-family `EndpointSlice` could be left behind after a
downgrade if the new apiserver exited uncleanly. (However, in this
scenario the Service ought to end up as single-stack at the end as
well, so the secondary-IP-Family `EndpointSlice` ought to be ignored
anyway.)

#### Service vs EndpointSlice Sync Issues

Given that the Service is updated only on apiserver restart, but the
EndpointSlice is updated periodically by each apiserver, then whenever
there is a mix of single-stack and dual-stack apiservers, it is
possible to see any combination of service and endpoint
dual-stack-ness:

  - Single-stack Service, single-stack EndpointSlice: in this case,
    the service will work fine; everyone will just ignore the
    secondary IP family.

  - Single-stack Service, dual-stack EndpointSlices: kube-proxy will
    ignore the secondary IP family endpoints, since it has no
    secondary-family ClusterIP to match them up with. If other clients
    try to use the secondary-IP-family EndpointSlice values directly,
    they should succeed, except in the "stale EndpointSlice" case
    discussed above.

  - Dual-stack Service, single-stack EndpointSlice: kube-proxy will
    create working mappings for the primary ClusterIP, and a "broken"
    (ie, `-j REJECT`) mapping for the secondary ClusterIP. Clients
    explicitly attempting to connect to the secondary ClusterIP will
    fail, but clients attempting to connect to
    `kubernetes.default.svc.cluster.local` should succeed, even if
    they try the secondary ClusterIP first, because they should fall
    back to the primary ClusterIP after the secondary one fails.

  - Dual-stack Service, dual-stack EndpointSlices: works fine. Any
    still-single-stack apiservers will not be advertising an IP in the
    secondary family, so the secondary-family EndpointSlice may only
    point to a subset of masters.

### Version Skew Strategy

The feature should not be affected by the versions of other components
(assuming all of the other components are at least of a version that
fully supports dual-stack).

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

- Initial proposal: 2020-04-13
