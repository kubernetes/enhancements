# KEP-4231: Metrics Anomaly Detector

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Monitoring Requirements](#monitoring-requirements)
- [Implementation History](#implementation-history)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in 
  [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]

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

The [Metrics Anomaly Detector] (MAD) is a valuable tool for maintaining the
health and stability of a Kubernetes environment. It monitors the health of
specified endpoints and stores aggregated health check events in a circular
buffer. This data can be queried to identify patterns or anomalies over a
specified time range, providing valuable insights into the system's health. The
buffer size, monitored endpoints, and query interval are all configurable,
allowing for customization based on the specific needs of the user. This
flexibility makes MAD a versatile tool for a variety of Kubernetes environments.

- [Metrics Anomaly Detector]: https://github.com/rexagod/mad/blob/master/README.md

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

 This tool is beneficial for maintaining system stability and performance,
 offering valuable insights into the system's health, and allowing for proactive
 troubleshooting and issue resolution. Please refer to the project [`README.md`]
 for a more detailed insight.

- [`README.md`]: https://github.com/rexagod/mad/blob/master/README.md

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

* To monitor the health of specified endpoints in a Kubernetes environment,
  reducing the need for manual monitoring.
* To store aggregated health check events in a circular buffer, which can be
  queried to identify patterns or anomalies over a specified time range.
* To provide a configurable tool, allowing users to set the buffer size,
  monitored endpoints, and query interval according to their specific needs.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

* The project does not aim to replace comprehensive monitoring and alerting
  systems. It is designed to supplement these systems by providing additional
  insights into the health of specific Kubernetes components.
* The project does not provide a user interface for visualizing the health check
  data. It focuses on the backend functionality of collecting and analyzing the
  data.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

The desired outcome of this project is to enhance the stability and performance
of Kubernetes environments by providing valuable insights into the system's
health. This will be achieved by storing aggregated health check events in a
circular buffer, which can be queried to identify patterns or anomalies over a
specified time range. The buffer size, monitored endpoints, and query interval
are all configurable, providing flexibility for different Kubernetes
environments. The success of this project can be measured by its ability to
accurately monitor the health of specified endpoints and identify patterns or
anomalies in the health check data. This can be quantified by the number of
anomalies detected and the accuracy of these detections. Additionally, user
feedback on the tool's usability and effectiveness can also be a measure of
success.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

SIG-Instrumentation was recently requested to have a solution in place for
analyzing the [component SLI metrics] for anomalies w.r.t. the health (`healthz`
and `readyz` status) for various components these health-check metrics were
exposed at `/metrics/slis`. This is different from a change-point where the
system is just transiently "down" than a scenario where it is mostly "healthy"
to a time when it is frequently "unhealthy".

[component SLI metrics]: https://kubernetes.io/docs/reference/instrumentation/slis/#sli-metrics

#### Story 2

As a user, I want to have a solution in place that reduces the manual cost of
making certain observations and writing queries against each one of them. Such a
tool should enable me to monitor the health of specified endpoints in a
Kubernetes environment, corresponding to Kubernetes or non-Kubernetes
components, and provide a way of querying the data to identify patterns or
anomalies over a specified time range.

#### Story 3

As a cluster admin, I want to automate the process of ananlyzing the health of
critical components before and after performing an upgrade, so I can infer the
state of the cluster based on environmental changes conveniently and make
appropriate decisions.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

The project forks `sample-controller`, and builds a controller that monitors the
health of specified endpoints and stores aggregated health check events in a
circular buffer. This is done using the `spec.BufferSize`,
`spec.HealthcheckEndpoints`, and `spec.QueryInterval` fields of the
`MetricsAnomalyDetector` CRD. The `container/ring` package is used to implement
the circular buffer, which is updated at every `queryInterval` seconds, using
the healthcheck events from the specified endpoints.

A query interface is also provided, that allows users to identify patterns or
anomalies in the health check data. This is essentially a rate-limited server
that listens for queries, extracts the parameters (time-intervals to report the
overall health status for, and the resource key, in order to identify it), and
returns the results.

The project has already released `v0.0.1`, and available
[here](https://github.com/rexagod/mad/tree/master).

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

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

- `https://github.com/rexagod/mad/internal/`: `27th February, 2024` - `5.0% of statements`
- WIP.

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

- WIP.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- WIP.

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

#### Alpha

- More test coverage.
- Deploy and reconcile manifests using the controller.
- Use [isolation-forests](https://ars.els-cdn.com/content/image/1-s2.0-S0952197622004936-fx1_lrg.jpg) as the underlying algorithm.

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

`N/A` (not merging into Kubernetes)

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

- (27th February, 2024) `v0.0.1` rolled out.
- WIP.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

A dedicated repository for the `metrics-anomaly-detector` subproject under 
the `kubernetes-sigs`.
