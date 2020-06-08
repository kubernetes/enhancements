<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up.  KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please ensure to complete all
  fields in that template.  One of the fields asks for a link to the KEP.  You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "title", "authors", "owning-sig",
  "status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary", and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG that are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly.  The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved.  Any KEP
marked as a `provisional` is a working document and subject to change.  You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused.  If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement", for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example.  If there are
new details that belong in the KEP, edit the KEP.  Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable` or significant changes once
it is marked `implementable` must be approved by each of the KEP approvers.
If any of those approvers is no longer appropriate than changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross cutting KEPs).
-->

# KEP-1792: Configure FQDN as Hostname for Pods

<!--
This is the title of your KEP.  Keep it short, simple, and descriptive.  A good
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
  - [User Stories](#user-stories)
    - [Story 1: User does not Configure Pod to have FQDN](#story-1-user-does-not-configure-pod-to-have-fqdn)
    - [Story 2: User Configures Pod to have FQDN](#story-2-user-configures-pod-to-have-fqdn)
    - [Story 3: User Configures Pod to have FQDN and it would like the pod hostname to be the FQDN](#story-3-user-configures-pod-to-have-fqdn-and-it-would-like-the-pod-hostname-to-be-the-fqdn)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

<!--
This section is incredibly important for producing high quality user-focused
documentation such as release notes or a development roadmap.  It should be
possible to collect this information before implementation begins in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself.  KEP editors, SIG Docs, and SIG PM
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.
-->

This proposal gives users the ability to set a pod’s hostname to its Fully Qualified Domain Name (FQDN).
A new PodSpec field `hostnameFQDN` will be introduced. When a user sets this field to true, its Linux
kernel hostname field ([the nodename field of struct utsname](http://man7.org/linux/man-pages/man2/uname.2.html))
will be set to its fully qualified domain name (FQDN). Hence, both uname -n and hostname --fqdn will return
the pod’s FQDN. The new PodSpec field `hostnameFQDN` will default to `false` to preserve current behavior, i.e.,
setting the hostname field of the kernel to the pod's shortname.

Related Kubernetes issue [#1791](https://github.com/kubernetes/enhancements/issues/1797).

## Motivation

<!--
This section is for explicitly listing the motivation, goals and non-goals of
this KEP.  Describe why the change is important and the benefits to users.  The
motivation section can optionally provide links to [experience reports][] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

This feature would increase the interoperability of Kubernetes with legacy applications.
Traditionally, Unix and certain Linux distributions, such as
[RedHat ](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/7/html/networking_guide/ch-configure_host_names)
and CentOS, have recommended setting the kernel hostname field to the host FQDN. As a consequence,
many applications created before Kubernetes rely on this behavior. Having this feature would help
containerize existing applications without deep, risky code changes.

### Goals

<!--<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up.  KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please ensure to complete all
  fields in that template.  One of the fields asks for a link to the KEP.  You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "title", "authors", "owning-sig",
  "status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary", and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG that are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly.  The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved.  Any KEP
marked as a `provisional` is a working document and subject to change.  You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused.  If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement", for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example.  If there are
new details that belong in the KEP, edit the KEP.  Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable` or significant changes once
it is marked `implementable` must be approved by each of the KEP approvers.
If any of those approvers is no longer appropriate than changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross cutting KEPs).
-->
<!--
List the specific goals of the KEP.  What is it trying to achieve?  How will we
know that this has succeeded?
-->

Giving users the ability to set the hostname field of the kernel to the FQDN of a Pod.



### Non-Goals

<!--
What is out of scope for this KEP?  Listing non-goals helps to focus discussion
and make progress.
-->
Giving users a way to configure or enforce this feature cluster-wide.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation.  The "Design Details" section below is for the real
nitty-gritty.
-->
This proposal gives users the ability to set a pod’s hostname to its FQDN. A new PodSpec field named `hostnameFQDN`
will be introduced, with type `*bool`.

The values of `hostnameFQDN` are:
* `nil` (default): The Linux kernel hostname field ([the nodename field of struct utsname](http://man7.org/linux/man-pages/man2/uname.2.html))
of a pod will be set to its shortname. This is the current behavior.
* `False`: Same as `nil`
* `True`: The Linux kernel hostname field ([the nodename field of struct utsname](http://man7.org/linux/man-pages/man2/uname.2.html))
of a pod will be set to its fully qualified domain name (FQDN). FQDN is determined as described [here](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service))


### User Stories

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system.  The goal here is to make this feel real for users without getting
bogged down.
-->
#### Story 1: User does not Configure Pod to have FQDN
Assume we have a pod named `foo` in a namespace `bar`. The PodSpec `subdomain` is not set. This pod does not have FQDN, so the value of `hostnameFQDN` does not have an impact. The Pod spec for this example would be:

```yaml
# Pod spec
apiVersion: v1
kind: Pod
metadata: {"name": "foo", "namespace": "bar"}
spec:
  ...

```

If we `exec` into the Pod:
* `uname -n` returns `foo`
* `hostname --fqdn` returns `foo`

#### Story 2: User Configures Pod to have FQDN

Assume we have a pod named `foo` in a namespace `bar`. The PodSpec `subdomain` is set to `test`. We also assume the cluster-domain is set to its default, i.e. `cluster.local`. The FQDN of this pod is defined as `foo.test.bar.svc.cluster.local` (see details [here](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service)). The user does not set `hostnameFQDN`. The Pod spec for this example would be:

```yaml
# Pod spec
apiVersion: v1
kind: Pod
metadata: {"name": "foo", "namespace": "bar"}
spec:
  ...
  hostname: "foo"  # Optional for this example
  subdomain: "test"
```

If we `exec` into the Pod:
* `uname -n` returns `foo`
* `hostname --fqdn` returns `foo.test.bar.svc.cluster.local`

#### Story 3: User Configures Pod to have FQDN and it would like the pod hostname to be the FQDN

Assume we have a pod named `foo` in a namespace `bar`. The PodSpec `subdomain` is set to `test`. We also assume the cluster-domain is set to its default, i.e. `cluster.local`. The FQDN of this pod is defined as `foo.test.bar.svc.cluster.local` (see details in [here](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service)). Additionally, the user sets `hostnameFQDN`: `true`. The Pod spec for this example would be:

```yaml
# Pod spec
apiVersion: v1
kind: Pod
metadata: {"namespace": "bar", "name": "foo"}
spec:
  ...
  hostname: "foo"  # Optional for this example
  subdomain: "test"
  hostnameFQDN: "true"
```

If we `exec` into the Pod:
* `uname -n` returns `foo.test.bar.svc.cluster.local`
* `hostname --fqdn` returns `foo.test.bar.svc.cluster.local`


### Notes/Constraints/Caveats

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

The hostname field of the Linux Kernel is limited to 63 bytes
(see [sethostname(2)](http://man7.org/linux/man-pages/man2/sethostname.2.html)). Kubernetes attempts to include the
Pod name as hostname, unless this limit is reached. When the limit is reached, Kubernetes has a series of mechanisms
to deal with the issue. These include, truncating Pod hostname when a “Naked” Pod name is longer than 63 bytes, and
having an alternative way of generating Pod names when they are part of a Controller, like a Deployment. The proposed
feature might still hit the 63 Bytes limit unless we create or adapt similar remediation techniques. Without any
remediation, Kubernetes will fail to create the Pod Sandbox and the pod will remain in “ContainerCreating” (Pending status)
forever. The feature proposed here will make this issue occur more frequently, as now the whole FQDN would be limited to 63
bytes. Next we illustrate the issue with an example of a potential error message, based on an initial draft of this
feature (PR [#91035](https://github.com/kubernetes/kubernetes/pull/91035)):

```bash
$ kubectl get pod
NAME                                                  READY   STATUS              RESTARTS   AGE
longpodnametestsaoitfail23423423432wer-547cc5-st6dd   0/1     ContainerCreating   0          52s
```
```bash
$ kubectl describe pod longpodnametestsaoitfail23423423432wer-547cc5-st6dd
Name:           longpodnametestsaoitfail23423423432wer-547cc5-st6dd
Namespace:      foo
...
...
Events:
  Type     Reason                  Age               From                                Message
  ----     ------                  ----              ----                                -------
  Normal   Scheduled               16s               default-scheduler                   Successfully assigned foo/longpodnametestsaoitfail23423423432wer-547cc5-st6dd to host.company.com
  Warning  FailedCreatePodSandBox  1s (x2 over 16s)  kubelet, host.company.com  Failed create pod sandbox: Failed to set FQDN in hostname, Pod hostname longpodnametestsaoitfail23423423432wer-547cc5-st6dd.p1324234234234.foo.svc.test q.company.com is too long (63 characters is the limit).
```

This failure mode is not great because it might not be apparent to users that their pods are failing. To improve the UX of this failure mode we will create an example Admission Controller that people can take and customize to apply their own policies. For example, if users care only about Deployments, they can make sure this Admission Controller account for the size of FQDN when the `hostnameFQDN` and `subdomain` flags are set in the PodSpec template.


```
<<[UNRESOLVED Will this work on Windows? ]>>

We are not certain that this will work on Windows as we could not do full
Kubernetes tests on Windows. We did a test some basic test with Docker on
a Windows machine. This test did "docker run -h <FQDN>  -it container",
and Docker just set the FQDN in the Windows COMPUTENAME environment
variable, so hostname returned the FQDN string. I could not find any
specific Windows Kubelet Pod runtime class. I guess it might just work as
Kubernetes simply relies on underlying runtime, e.g., Docker?

<<[/UNRESOLVED]>>
```


### Risks and Mitigations

<!--
What are the risks of this proposal and how do we mitigate.  Think broadly.
For example, consider both security and how this will impact the larger
kubernetes ecosystem.

How will security be reviewed and by whom?

How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.
-->

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable.  This may include API specs (though not always
required) or even code snippets.  If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.  Anything
that would count as tricky in the implementation and anything particularly
challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations).  Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->
The following end-to-end test is implemented in addition to unit tests:
-   Create e2e cases for the 3 User scenarios we described and check value returned by `hostname`/`uname -n` versus `hostname --fqdn`


### Graduation Criteria
-   Compatible with major systems (e.g. linux, windows)

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
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and
GA/stable, since there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to make use of the enhancement?
-->

We will gate off this feature for one release (1.19), then we enable it as GA in the next release (1.20)

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

Old kubelets that do not have support for this feature will just ignore the PodSpec `hostnameFQDN` field.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable, can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md

Production readiness review questionnaire must be completed for features in
v1.19 or later, but is non-blocking at this time. That is, approval is not
required in order to be in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.

-->

### Feature enablement and rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: hostnameFQDN
    - Components depending on the feature gate: Kubelet
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  No

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Yes, it can be disabled. However, it will only have effect on newly created
  pods. Existing pods will keep having their FQDN as hostname, if they were
  configured for it.

* **What happens if we reenable the feature if it was previously rolled back?**
  New pods, configured to have FQDN in hostname, will start getting FQDN in the
  hostname field of kernel.

* **Are there any tests for feature enablement/disablement?**
  We will have unit tests and integration tests. Not sure if we need conversion tests.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  It is not clear that the rollout can fail due to this feature. The scope of this feature
  is very limited and it is disabled by default.

* **What specific metrics should inform a rollback?**
  We could have a metric in Kubelet that records number of failed pods that use this feature. If that
  metric spikes we could trigger a rollback.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  We tested introducing and removing this feature. Running pods are not affected by
  either introducing nor removing the feature. When disabling the feature, Pods
  using this feature that are "stuck" due to having long FQDNs will go into
  running.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  N/A

### Monitoring requirements

_This section must be completed when targeting beta graduation to a release._

TODO

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metrics. Operations against Kubernetes API (e.g.
  checking if there are objects with field X set) may be last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At the high-level this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide a comprehensive guidance, but at the very
  high level (they needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  Describe the metrics themselves and the reason they weren't added (e.g. cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  No


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirms the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  No

* **Will enabling / using this feature result in introducing new API types?**
  No

* **Will enabling / using this feature result in any new calls to cloud
  provider?**
  No

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  No

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  No

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No

### Troubleshooting

Troubleshooting section serves the `Playbook` role as of now. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now we leave it here though.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  It is not affected.

* **What are other known failure modes?**
  TODO

  For each of them fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without loogging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debugging the issue?
      Not required until feature graduated to Beta.
    - Testing: Are there any tests for failure mode? If not describe why.

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

<!--
Major milestones in the life cycle of a KEP should be tracked in this section.
Major milestones might include
- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

Setting the FQDN in the hostname field of the Kernel is not the standard in applications that have been developed to run in orchestration platforms such as Kubernetes. Additionally, the fact that the Kernel hostname field is limited to 63 bytes causes pretty poor failure modes, where users might not immediately know that something went wrong.


## Alternatives

<!--
What other approaches did you consider and why did you rule them out?  These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
Alternative to creating this feature:
* Make users fix all their own legacy code to not assume the FQDN is the hostname, which does not seem practical.

Alternative to controlling the feature:
* We could also control the use of this feature using a Kubelet configuration flag. Configuration flags are harder to maintain and it requires from platforms, such as GKE, to include support for them. Additionally, using a PodSpec flag we ensure that the behavior of controllers, like Deployments, is consistent on all its pods. For example, if we were to use a Kubelet config flag we might end up on a situation where different pods of the same deployment behave differently.

Alternatives for improving UX of failure mode:
* Create an admission plugin that calculates the length of the FQDN of the Pod. The problem of this approach is that it might
not cover all scenarios, there are many entry points to generate a pod, i.e., deployments, replicasets, CRD, etc. Another problem
is that it breaks Kubernetes abstraction layers as we have to make assumptions from the top layer.
* Create non-retriable errors for pods. Currently failures like the one generated by this kernel hostname limit retry forever.
It would be nice if we can define that an error is fatal, then the pod changes to Failed state.


## Infrastructure Needed (optional)

<!--
Use this section if you need things from the project/SIG.  Examples include a
new subproject, repos requested, github details.  Listing these here allows a
SIG to get the process for these resources started right away.
-->
