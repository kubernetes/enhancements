# KEP-2238: Liveness Probe Grace Periods

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Configuration example](#configuration-example)
  - [User Stories (Optional)](#user-stories-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [Graduation](#graduation)
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
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Liveness probes currently use the `terminationGracePeriodSeconds` on both
normal shutdown and when probes fail. Hence, if a long termination period is
set, and a liveness probe fails, a workload will not be promptly restarted
because it will wait for the full termination period.

I propose adding a new field to probes, `probe.terminationGracePeriodSeconds`.
When set, it will override the `terminationGracePeriodSeconds` for liveness or
startup termination, and will be ignored for readiness probes. This maintains
the current behaviour if desired while providing configuration to address this
unintended behaviour.

## Motivation

This change is important to give users the option to decouple these fields, as
the current behaviour is counter-intuitive and doesn't provide an option to
decouple.

[Other implementation options](#alternatives) might avoid an API change, but
would end up breaking backwards compatibility which some users may have come to
[expect/rely on](https://github.com/kubernetes/kubernetes/issues/64715#issuecomment-756100727),
even if we take a phased approach to introduce them.

An example of a scenario where this bug was apparent is a long outage reported
by an ingress controller:

> the controller uses a 3600s grace period so that it can drain long requests
> from end user app traffic (it listens on endpoints that can remain available
> for hours like host network, but also incoming tcp load balancers which have
> open connections), but when the controller wedges it takes an hour to
> resolve, which is the worst possible outcome (the process wedges immediately,
> but kube has to wait the full hour to kill it, defeating the goal of
> liveness). [comment](https://github.com/kubernetes/kubernetes/issues/64715#issuecomment-756201502)

### Goals

- Liveness probes can have a configurable timeout on failure.
- Existing behaviour is not changed.

### Non-Goals

Any changes to probe behaviour outside of the scope of addressing the
[bug](https://github.com/kubernetes/kubernetes/issues/64715) this is intended
to address.

## Proposal

We can add a `terminationGracePeriodSeconds` integer field to the Probe API.
This field would specify how long the kubelet should wait before forcibly
terminating the container when a probe fails, and override the pod-level
`terminationGracePeriodSeconds` value.

This change would apply to [liveness and startup probes][probes1], but **not**
readiness probes, which [use a different code path][probes2].

This change would be [API-compatible][compatible]. If the field is not
specified, we will default to the current behaviour, which uses the
`terminationGracePeriodSeconds`.

[compatible]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#on-compatibility
[probes1]: https://github.com/kubernetes/kubernetes/blob/e414d4e5c2ab85eada7eb8e80a363658e199968b/pkg/kubelet/kuberuntime/kuberuntime_manager.go#L631-L655
[probes2]: https://github.com/kubernetes/kubernetes/blob/e414d4e5c2ab85eada7eb8e80a363658e199968b/pkg/kubelet/prober/prober_manager.go#L55

### Configuration example

```yaml
spec:
  terminationGracePeriodSeconds: 3600
  containers:
  - name: test
    image: ...

    ports:
    - name: liveness-port
      containerPort: 8080
      hostPort: 8080

    livenessProbe:
      httpGet:
        path: /healthz
        port: liveness-port
      failureThreshold: 1
      periodSeconds: 60
      # New field #
      terminationGracePeriodSeconds: 60
```

### Risks and Mitigations

This should be a low-risk API change as it is backwards-compatible. If the
field is not set, the current behaviour will be maintained.

## Design Details

This field is not valid for readiness probes. We will add validation to ensure
`readinessProbe.terminationGracePeriodSeconds` remains unset.

We expect that the `livenessProbe.terminationGracePeriodSeconds` (or
`startupProbe.*`) will not be greater than the pod-level
`terminationGracePeriodSeconds`, but we will not explicitly validate this.

`initialDelaySeconds` is not relevant to this value; it indicates how long we
should wait before probing the container, but does not have any relation to how
long a probe should take or how long it will take to shut down the container
upon a failed probe.

`periodSeconds` and `timeoutSeconds` indicate how long we think a probe should
take, but not how long it will take to shut down the container; hence, they
also are not relevant to this value. It is suggested but not required that
`livenessProbe.terminationGracePeriodSeconds` be less than these values.

Finally, `failureThreshold` and `successThreshold` are not relevant to this
value; they indicate the number of failures required to mark a probe as
succeeded or failed, but not how long to give a container to gracefully shut
down upon failure.

### Test Plan

This change will be unit tested for the API changes and
backwards-compatibility.

If added to the Probe API, this will also need to be tested in the readiness
and startup probe cases. Readiness probes will not support the field. Startup
probes should be tested for correct behaviour.

E2E integration tests will verify the correct behaviour for the current broken
use case, where a `terminationGracePeriodSeconds` is set to a
large value, a probe fails, and with
`livenessProbe.terminationGracePeriodSeconds` set, the container terminates
quickly.

### Graduation Criteria

#### Alpha

- New probe field, `terminationGracePeriodSeconds`, is implemented and
  available behind a feature flag.
- Appropriate tests are written.

#### Beta

- Feature flag is defaulted on.
- Remove feature gate from [kubelet](https://github.com/kubernetes/kubernetes/pull/99375#issuecomment-794680869).
- Add validation to ensure `terminationGracePeriodSeconds` is non-negative.

_Below graduation criteria are tentative._

#### Graduation

- Feature flag is removed, feature is graduated.

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

On upgrade: feature flag/new field will become available for use.

On downgrade: no longer available. Any workloads with the Probe
`terminationGracePeriodSeconds` set will need to unset.

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

n-2 kubelet without this feature will default to the old behaviour, using the
pod-level `terminationGracePeriodSeconds`.

Only when feature gate is enabled for all components will we use the new field.

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
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `ProbeTerminationGracePeriod`
    - Components depending on the feature gate: Kubelet, API Server
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.

  No.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).

  Yes: disable feature flag.

  While feature flag is enabled, the feature can also be disabled by unsetting
  the field on the Probe specification, which will restore the default
  behaviour.

* **What happens if we reenable the feature if it was previously rolled back?**

  Field will be used instead of the `terminationGracePeriodSeconds` for
  shutting down a container upon failed liveness probe.

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.

  We will manually test that if a resource is created with the new field, and
  the feature flag is later disabled, that the field will no longer be used
  even though it is already filled in. We will then confirm that, when the flag
  is reenabled, the new value is used as expected.

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

  No.

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

  No.

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

  No.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)

  One new integer field on the Probe type.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

  No, this should not affect that.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

  No.

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

- 2021-01-07: Initial draft KEP

## Drawbacks

The main drawback of this proposal is that it requires a (compatible) API
change.

## Alternatives

From [#64715](https://github.com/kubernetes/kubernetes/issues/64715#issuecomment-756173979):

Infer the maximum grace period for liveness from `num failures * probe period`,
implying that a user with short liveness reaction wants a fast reaction from
their pod (@smarterclayton)

- PRO: Any user unaware of the coupling reacts to liveness failures faster,
  normal pods (including reason 2 above) are less likely to be impacted
- CON: If we later want to change the API to indicate this field, the default
  behavior makes less sense (i.e. complicates choosing the option above)
- CON: A user with a long grace period who wants liveness to take more than
  tens of seconds (not sure why they would) suddenly sees that behavior
  impacted
- CON: This sort of implicit calculation feels pretty arbitrary (we generally
  make defaults explicit vs calculated)

Introduce a new toleration, similar to execute toleration for node not ready
(how long the pod should be left on the node while unready before being
evicted), that controls how long liveness tolerates (@smarterclayton)

- PRO: avoids API change
- CON: basically a hack to avoid an API change, tolerations for execution don't
  really make sense for conditional stuff

Since liveness probe failures indicate that a container is unhealthy and should
be terminated, update the behaviour of liveness probes to terminate the pod
immediately, or with some default grace period (@sjenning)

- PRO: avoids API change
- CON: may break existing applications that rely on the grace period for crash
  safety
- CON: thresholds not user-configurable
- CON: even if we use a feature flag for this, it will eventually become the
  required behaviour

Introduce a similar field, but at the pod-level, next to the existing field.

- PRO: similar values are grouped together
- CON: logically, liveness probes operate on a per-container basis, and thus we
  may need different thresholds for different containers

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
