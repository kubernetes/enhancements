<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-3241: Support services with same SRV name on different protocols

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Types, API server and controller behaviour](#types-api-server-and-controller-behaviour)
  - [kubernetes/dns updates](#kubernetesdns-updates)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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
  - [Don't rely on SRV records in this way](#dont-rely-on-srv-records-in-this-way)
  - [Manage DNS records outside Kubernetes](#manage-dns-records-outside-kubernetes)
  - [Use the <code>appProtocol</code> field](#use-the--field)
  - [Relax the ServicePort <code>name</code> uniqueness check](#relax-the-serviceport--uniqueness-check)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
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

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

Several network protocols operate over both TCP and UDP.  In cases
where SRV records are used to locate services, the SRV records
typically use the one IANA-registered service name for both the
`_tcp` and `_udp` variants.  Kubernetes' Service object schema and
DNS service discovery specification do not support this use case.
This KEP seeks to address this gap.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Several real-world protocols operate on either TCP or UDP, or a
combination of the two.  These include Kerberos, SIP, RADIUS and
SNMP.  In some cases, particularly Kerberos and SIP, SRV records are
used as a primary means of locating servers.  The SRV records use
the IANA-registered service name for the both the `_tcp` and `_udp`
protocols, for example:

```
_kerberos._tcp.<domain> IN SRV <priority> <weight> <port> <target>
_kerberos._udp.<domain> IN SRV <priority> <weight> <port> <target>
```

The Kubernetes Service object schema currently does not support the
creation of service objects that have `ports` with the same name but
a different protocol.  As a consequence, Kubernetes cannot produce
the SRV records required by such applications.

<!--
Endpoints and the
-->


### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Extend the ServicePort and EndpointPort types to allow the
  creation of Service objects that result in SRV records having the
  same service label but different protocols.

- Alter the behaviour of the Endpoint/EndpointSlice controller to
  recognise the new schema and propagate the required information to
  the subordinate Endpoints/EntpointSlice object.

- Coordinate updates to [kubernetes/dns][kubedns], other
  implementors of the [DNS-based Service Discovery
  specification][dns-spec] and the ExternalDNS system to recognise
  the new field and create the correct SRV records.

- If necessary, extend the [DNS-based Service Discovery
  specification][dns-spec] and other relevant documents to elaborate
  the requirements with respect to this KEP.

- Maintain backwards compatibility with no change in observable
  behaviour for Service objects that do not use the extended
  schema/semantics introduced by this KEP.

[kubedns]: https://github.com/kubernetes/dns
[dns-spec]: https://github.com/kubernetes/dns/blob/master/docs/specification.md

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Support for URI records or more advanced applications or SRV
  records such as DNS-SD ([RFC 6763][])

[RFC 6763]: https://www.rfc-editor.org/rfc/rfc6763.html

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a Kerberos administrator, I want to be able to deploy a Kerberos
KDC in Kubernetes and have Kubernetes create the required SRV
records automatically (either in CoreDNS and/or via ExternalDNS).

Specifically, the Service object:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: idm
spec:
  type: LoadBalancer
  selector:
    app: service-test
  ports:
  - name: kerberos-tcp
    srvServiceName: kerberos  # [field name subject to change]
    protocol: TCP
    port: 88
  - name: kerberos-udp
    srvServiceName: kerberos
    protocol: UDP
    port: 88
```

…should be accepted and should yield SRV records matching the
following pattern:

```
_kerberos._tcp.idm.<domain> IN SRV <priority> <weight> 88 <target>
_kerberos._udp.idm.<domain> IN SRV <priority> <weight> 88 <target>
```

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

A change to Service controller behaviour risks breaking backwards
compatibility for existing Service objects.  This is mitigated by
extending the ServicePort schema with the new `SRVServiceName`
field.  There should be no observable changes in behaviour when the
field is not set.


## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Types, API server and controller behaviour

A new, optional `SRVServiceName` field shall be added to the
`ServicePort` and `EndpointPort` types.  The field shall be
validated as a DNS\_LABEL.

**Note**: the field name `SRVServiceName` is subject to change.

```go
type ServicePort struct {
    ...
    // The service name that should be used in SRV records.  This must be a
    // DNS_LABEL, typically an IANA-registered service name (without
    // leading underscore).  This field can be used to create SRV records
    // with the same service name for different protocols.  If unspecified
    // If unspecified, the 'Name' field will be used for the service name
    // in SRV records.
    // +optional
    SRVServiceName *string
}
```

```go
type EndpointPort struct {
    ...
    // The service name that should be used in SRV records.  This must be a
    // DNS_LABEL, typically an IANA-registered service name (without
    // leading underscore).  This field can be used to create SRV records
    // with the same service name for different protocols.  If unspecified
    // If unspecified, the 'Name' field will be used for the service name
    // in SRV records.
    // +optional
    SRVServiceName *string
}
```

The relevant controllers shall be updated to propagate the
`SRVServiceName` field from Service objects to Endpoints and
EndpointSlice objects.


### kubernetes/dns updates

The CoreDNS service shall be updated to recognise the
`SRVServiceName` field.  If supplied, its value shall be used
instead of the `Name` field in the creation of SRV records.

Further implementation details will be provided later.


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

Existing tests do not use the `SRVServiceName` field.  Therefore
existing test suite is adequate to ensure the behaviour does not
change when when `SRVServiceName` field is unused.

To test the new functionality, new tests shall be introduced.  These
will create Service objects that use the `SRVServiceName` field and
check that the given data are reflected in the resulting Endpoints
and EndpointSlice objects.

Negative tests to ensure that invalid values are rejected shall be
implemented.

The test suites for other modified components (e.g. kubernetes/dns)
shall likewise be extended to test the new behaviour.

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

-->

This adds a new optional attribute to the stable Service object API.
We will follow the standard approach of initially guarding new
fields behind a feature gate.

The feature gate shall be called `SRVServiceName` (subject to
discussion).

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- CoreDNS recognises the new field and creates SRV records properly
- e2e tests extended to include CoreDNS SRV record generation

#### GA

- The `SRVServiceName` field is supported by other prominent
  implementations of the DNS service discovery and ExternalDNS
  systems.
- Where feasible, e2e tests for these other DNS implementations have
  been implemented

<!--
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

No specific upgrade/downgrade strategy is required.  All new
behaviour is contained in the API server and service controllers.

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

While the cluster is being upgraded, it is possible that failures
and normal leader election procedures may result in transient
reversion to an old controller version.  Thus, there may be
instability in the Endpoints objects and SRV records generated by
Kubernetes, until the upgrade is complete.  As the new field is
unlikely to be in use immediately, the risk of this occurring is
tolerable and does not need specific remediation.

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
  - Feature gate name: `SRVServiceName`
  - Components depending on the feature gate:
    - kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
    - **No**
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
    - **No**

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Enabling the feature does not change any default behaviour.  Service
objects must use the new `srvServiceName` ServicePort field to set
the same SRV service label with different protocols.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

The enhancement can be disabled.  Upon rollback, the
`srvServiceName` field will be ignored by the API server and the
Kubernetes will revert to the previous behaviour (i.e. creating SRV
records that do not satisfy the requirements of the applications for
which this feature was needed).

###### What happens if we reenable the feature if it was previously rolled back?

The enhanced behaviour for recognising and processing the
`srvServiceName` field returns.  No adverse effects.

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

No.  Using this feature only changes the information in Endpoints
objects and the SRV records created based on that information.

###### Will enabling / using this feature result in introducing new API types?

No.  This feature only extends the schema of the Service
(<!--TODO--> and Endpoints?) types

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

This enhancement introduces the new `srvServiceName` string field
in the ServicePort field.  The field is optional, not used by
default, and not required by most applications.  When used, the
field will typically contain an IANA-registered service name
(typically < 10 characters).  If the feature is used at all, for the
typical use case the field will be set in at least two ServicePort
values in the Service object.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

Enabling or using this feature is not expected to have an impact on
SLIs/SLOs.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

This feature introduces a trivial amount of additional behaviour in
the Service controller.  Performance impact is expected to be
negligible.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

No additional API calls are expected to be needed for this
enhancement.  The various controllers that deal with Service and
Endpoints objects, and the applications that make use of the DNS
records managed by Kubernetes, will fail in the same way as they
currently do.

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

This feature adds a new field to the ServicePort type—part of the
Service object.  This object type is often dealt with directly by
end users.  Yet another field to understand adds to the cognitive
burden of using Kubernetes.

The use case that this KEP addresses is relatively uncommon among
the kinds of applications typically *deployed* on Kubernetes (to
date).

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Don't rely on SRV records in this way

This is not a valid alternative.  The use case concerns the
operation of important, IETF standardised and widely used
applications/protocols such as Kerberos and SIP (among others) with
a diversity of client and server implementations.


### Manage DNS records outside Kubernetes

An alternative is to manage all DNS needs of the application outside
Kubernetes.  But this places a considerable burden upon the
application developer/operator.  Kubernetes already has a lot of
infrastructure in place for managing applications' DNS publication
requirements, alongside substantial ongoing efforts such as
ExternalDNS.  The changes in Kubernetes required to support this use
case are modest relative to the burden that *not* supporting it
places upon the application developer/operator.


### Use the `appProtocol` field

The ServicePort and EndpointPort objects have an `appProtocol` field
whose semantics overlap somewhat with the requirements driving this
KEP.  It would be possible to use this existing field to achieve
what this KEP seeks to achieve.  The following factors weigh against
this idea.

First, the purpose of `appProtocol` is to convey to
routing/balancing/ingress systems information about the transport or
other underlying application-layer protocols used by the service
(replacing the *ad hoc* annotations used in the past for this
purpose). For example, if a service uses HTTP, you could set
`appProtocol: HTTP`. But it would be inappropriate if this resulted
in the creation of an `_http._tcp.<...> SRV record`, in place of (or
in addition to) the SRV record(s) that would ordinarily be created
for the service.  See
[comment](https://github.com/kubernetes/kubernetes/issues/97149#issuecomment-742965305).

Second, the `appProtocol` field accepts non-standard "prefixed"
values (e.g.  `mycompany.com/my-custom-protocol`).  Such values do
not have a valid or reasonable interpretation as an SRV service
label.  See
[comment](https://github.com/kubernetes/kubernetes/issues/97149#issuecomment-764967090).


### Relax the ServicePort `name` uniqueness check

Conceptually, the requirement could be met by relaxing the
requirement that the ServicePort `name` is unique and instead
enforcing the uniqueness on the `name`+`protocol` pair.  However,
this breaks API semantics.  The `name` field is explicitly declared
to be unique within the Service object, and other components or
programs might rely on this property.

Additionally, relaxing the uniqueness requirement is likely to be
incompatible all existing client-side merge behaviour, and trigger
other issues with patch/merge behaviour (see
[comment](https://github.com/kubernetes/kubernetes/issues/97149#issuecomment-741911314)).

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

Nil.
