# KEP-3031: Signing release artifacts

<!-- toc -->

- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required _prior to targeting to a milestone / release_.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Target of this enhancement is to define which technology the Kubernetes
community is using to signs release artifacts.

## Motivation

Signing artifacts provides end users a chance to verify the integrity of the
downloaded resource. It allows to mitigate man-in-the-middle attacks directly on
the client side and therefore ensures the trustfulness of the remote serving the
artifacts.

### Goals

- Defining the used tooling for signing all Kubernetes related artifacts
- Providing a standard signing process for related projects (like k/release)

### Non-Goals

- Discussing not user-facing internal technical implementation details

## Proposal

Every Kubernetes release produces a set of artifacts. We define artifacts as
something consumable by end users. Artifacts can be binaries, container images,
checksum files, documentation, provenance metadata, or the software bill of
materials. None of those end-user resources are signed right now.

The overall goal of SIG Release is to unify the way how to sign artifacts. This
will be done by relying on the tools of the Linux Foundations digital signing
project [sigstore](https://www.sigstore.dev). This goal aligns with the
[Roadmap and Vision](https://github.com/kubernetes/sig-release/blob/f62149/roadmap.md)
of SIG Release to provide a secure software supply chain for Kubernetes. It also
joins the effort of gaining full SLSA Compliance in the Kubernetes Release
Process ([KEP-3027](https://github.com/kubernetes/enhancements/issues/3027)).
Because of that, the future [SLSA](https://slsa.dev) compliance of artifacts
produced by SIG release will require signing artifacts starting from level 2.

[cosign](https://github.com/sigstore/cosign) will be the tool of our choice when
speaking about the technical aspects of the solution. How we integrate the
projects into our build process in k/release is out of scope of this KEP and
will be discussed in the Release Engineering subproject of SIG Release. A
pre-evaluation of the tool has been done already to ensure that it meets the
requirements.

An [ongoing discussion](https://github.com/kubernetes/release/issues/2227) about
using cosign already exists in k/release. This issue contains technical
discussions about how to utilize the existing Google infrastructure as well as
consider utilizing keyless signing via workload identities. Nevertheless, this
KEP focuses more on the "What" aspects rather than the "How".

### User Stories (Optional)

- As an end user, I would like to be able to verify the Kubernetes release
  artifacts, so that I can mitigate possible resource modifications by the
  network.

### Risks and Mitigations

- **Risk:** Unauthorized access to the signing key or its infrastructure

  **Mitigations:**

  - Storing the credentials in a secure Google Cloud Project with
    limited access for SIG Release.
  - Enabling the cosign [transparency log
    (Rekor)](https://github.com/sigstore/cosign#rekor-support) to make the key
    usage publicly auditable.
  - Working towards [keyless
    signing](https://github.com/sigstore/cosign/blob/3f83940/KEYLESS.md) to
    minimize the attack surface of the supply chain.

### Test Plan

Testing of the lower-level signing implementation will be done by writing unit tests
as well as integration tests within the
[release-sdk](https://github.com/kubernetes-sigs/release-sdk) repository. This
implementation is going to be used by
[krel](https://github.com/kubernetes/release/blob/master/docs/krel/README.md)
during the release creation process, which is tested separately. The overall
integration into krel can be tested manually by the Release Managers as well,
while we use the pre-releases of v1.24 as first instance for full end-to-end
feedback.

### Graduation Criteria

#### Alpha

- Outline and integrate an example process for signing Kubernetes release
  artifacts.

  Tracking issue: https://github.com/kubernetes/release/issues/2383

#### Beta

- Standard Kubernetes release artifacts (binaries and container images) are
  signed.

#### GA

- All Kubernetes artifacts are signed. This does exclude everything which gets
  build outside of the main Kubernetes repository.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

Signed images have not to be verified, so they do not interfere with a running
cluster at all. They can be verified manually or by using the tooling provided
by our documentation.

###### Does enabling the feature change any default behavior?

Not when a manual verification will be done. If the cluster will change its
configuration to only accept signed images, then invalid signatures will cause
the container runtime to refuse the image pull. The same behavior could be
achieved by using an admission webhook which verifies the signature.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, depending on how the signature verification will be done.

###### What happens if we reenable the feature if it was previously rolled back?

It will behave in the same way as enabled initially.

###### Are there any tests for feature enablement/disablement?

No, not on a cluster level. We test the signatures during the release process.

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

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

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

## Drawbacks

- The initial implementation effort from the release engineering perspective
  requires adding an additional layer of complexity to the Kubernetes build
  pipeline.

## Alternatives

- Using the [OCI Registry As Storage (ORAS) project](https://github.com/oras-project/oras)

## Implementation History

- 2022-01-27 Updated to contain test plan and correct milestones
- 2021-11-29 Initial Draft
