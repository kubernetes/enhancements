# KEP-2012: Expose status.containerStatuses[*].imageID through Downward API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes](#notes)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
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
<!-- /toc -->

## Release Signoff Checklist

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
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubernetes currently offers [Downward API to expose pod information to
containers](https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/).

This KEP proposes to extend existing Kubernetes Downward API to support exposing
current imageID (image digest) to a running container through an environment variable.
This allows the running container to obtain information about what exactly is
being run inside the container, allowing easier debugging, logging, and reproducibility.

## Motivation

In service-oriented architectures, a container is the basic unit
of computation.
An image contains software which runs inside the container and
can potentially include multiple dependencies coming from various
sources (system packages, Python packages, compiled go programs, data
or fixture files, etc.).
As such the image can serve as way to exactly describe all those
dependencies and thus exactly the software which is or was running
inside the container.
Digest of the image (image ID) serves as a quick way to both describe
the image uniquely and to retrieve the exact version of it at a later time.
It is important to know what exactly is being run inside a container,
because this can help understand the results coming from the container
and to debug any issues.
Moreover, knowing exactly what is being run inside a container
allows easier reproducibility of that container.

Currently, it is possible to obtain this information from outside of the
container. But sometimes it is easier if this information is being
used inside the container: in logging messages originating from the container,
in records of computation recorded from the inside of the container,
or when communicating with the end user and informing them about which
exactly the version of the service are they communicating with.

### Goals

Goal of this KEP is to increase introspection a container can have into
itself and improve its understanding of what exactly is being run. This should
help with debugging, logging, and reproducibility of results and issues
related to the container. It should work even when image is not specified
with hash-based image name but with a name and tag which changes through
time to which exactly image it resolves to.

### Non-Goals

Providing a cryptographically secure attestation of the image running inside
the container is out of the scope of this proposal. This proposal addresses only
readily available imageID (image digest) information and does not provide
security guarantees for its validity. Feature described in this KEP is not
a security feature.

## Proposal

The proposal is to allow using `status.containerStatuses[*].imageID` as
the `fieldPath` of the `fieldRef` parameter in the `env` section of the container
configuration. `*` represents the index into the `containerStatuses` array.
The value should be the `imageID` (image digest) of the image
which is currently being used to run the container, allowing one
to retrieve exactly the same image at a later time. This should work
when `imagePullPolicy: Always` is used.

Moreover, for completeness also `status.containerStatuses[*].image` should
be allowed. It is easier available through alternative means, but having
both values be available through same Downward API allows one to obtain
both more user-friendly `image` and precise `imageID`.

### User Stories

#### Story 1

A web app wants to include in HTML footer a reference to the software version
which has served a particular request. Because app is packaged into a image
with many dependencies, exposing the digest of the image serves as such
version exactly describing what served the request. If a request goes
through multiple containers each can add additional digest information
to the header.

Alternatively, a container can log the digest of current image in logging
messages it produces.

Both of those approaches allows developer to have easier time debugging
and reproducing any issue reported by the user and/or discovered in logs.

#### Story 2

In science, reproducibility is one of important goals when doing
experiments. Many experiments in computer science use software
to conduct experiments. Images and containers are a great way to
achieve reproducibility because they pack all dependencies together
into one unit. When Kubernetes is used to run such jobs it is important
to record the exact version of the image being used so that the job
can be reproduced at a later time and results verified. This is complicated
when researchers prepare just inputs to the container, but the image and
its scheduling on the cluster is done by others, or automatically.
Thus, researchers might simply use a tag-based image name, e.g., `latest`.
Scheduling is done automatically with pod configuration which
contains tag-based image name. The image contains the software to
run the experiment on the inputs and record the results, including the
version of software used, i.e., Docker image version. Thus, the container
needs access to the digest of the image used for the container.

### Notes

Downward API already provides [various information](https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#capabilities-of-the-downward-api)
(pod and host's IPs, name, namespace, labels, annotations, resources requestxs and limits,
etc.). This proposal adds information about the image used to run a container.

### Risks and Mitigations

Additional risks because of this proposal are not anticipated.
Information provided to the containers are opt-in: downward API has to
be explicitly configured by the pod spec author. Supporting this
feature does not require obtaining any additional (sensitive) information
and this information is already readily available on the host.

## Design Details

To assure the correct information is provided to the container, the startup
of the container should do the following:

* Pull a new version of the image based on provided image name in the pod spec,
  when `imagePullPolicy` is `Always`.
* Obtain `imageID` of the image on the host, obtaining hash-based image name
  at the same time.
* Use hash-based image name to run the container, passing in `imageID` and `image`
  as environment variables if configured this way in pod's spec using
  Downward API.

Using hash-based image name in the last step assures that `imageID` matches
the image used to run the container.

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

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include 
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
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

Implementation of this is contained to only the executor component for containers,
which has to pass on the digest of the image it is using to run the container.
As such version skew is not anticipated to cause issues.

## Production Readiness Review Questionnaire

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

* **Does this feature depend on any specific services running in the cluster?**

No additional services besides what is already required to run pods in a cluster.

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

Docker provides a way to retrieve the image given hash-based Docker image name.
Some other container runtimes might not support that, so despite exposing imageID
this might not be enough to retrieve the same version of the image afterwards.
While this makes it harder to rerun the same image there might be other ways to
map imageID back to a image, e.g., enumerating all known images and checking their
imageID.

## Alternatives

One alternative is to use hash-based image name in the pod's spec and manually
provide it as well as an environment variable. This is error prone if done manually
and validates DRY principle (a templating engine can be used to mitigate that, but
then one has to use a templating engine, which adds to the complexity).
Moreover, this approach does not work if a tag-based image name is wanted
to be used in pod's spec, but container should know to which exactly image
this tag-based image name ended up resolving at the end by Kubernetes.

A container could call back into the Kubernetes API to obtain status of
itself (using Downward API to expose pod's `metadata.uid` first to the
container which can then be used when talking to the API), but this means
that API has to be opened to the container and that the image has to have
an API client, which means that the image has to be specifically tailored
to run on Kubernetes. Using environment variable is much easier and
easily provided in other running environments (e.g., when running the
image locally).

Docker in Docker could be used that a wrapper container is run by
Kubernetes, which then in turn runs the internal real job container providing
wanted environment variable. This approach requires that the wrapper
container is privileged which makes it not suitable for clusters with
external users.
