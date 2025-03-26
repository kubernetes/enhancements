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
# KEP-4762: Allows setting arbitrary FQDN as the pod's hostname

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
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

This proposal allows users to set arbitrary Fully Qualified Domain Name (FQDN) as the hostname of a pod, introduces a new field `actualPodHostname` for the podSpec, which, if set, once the API is GA will always be respected by the Kubelet (otherwise it will fall back to legacy behavior), and no longer cares about the `hostname` as well as the `subdomain` values.

## Motivation

This feature will allow some traditional applications to join kubernetes in a more friendly way. Some older services may use hostname to determine permissions or service operations. When migrating services to k8s, the migration path will become confusing due to the hostname restrictions of the pod itself, because when we try to add a Fully Qualified Domain Name (FQDN) hostname to the pod, it will inevitably always carry the `cluster-suffix`, which will never be possible for services that expect to use DNS to match the hostname.

### Goals

* Allow users to set any arbitrary FQDN as pod hostname.
* Write the FQDN set by the user to `/etc/hosts` in the pod.

### Non-Goals

* Add DNS records for the FQDN set by the user.

## Proposal

We add a new field called `actualPodHostname` to `podSpec`, of type string. When the value of the `actualPodHostname` field is not an empty string, it always overrides the values of the `setHostnameAsFQDN`, `subdomain`, and `hostname` fields in `podSpec` to become the hostname of the pod, and only allow the value of setHostnameAsFQDN to be nil.

### User Stories (Optional)

#### Story 1

As a Kubernetes administrator, I want the Kerberos replication daemon (kpropd) to accurately handle hostname resolution for authentication.

In a Kubernetes environment, kpropd on the receiving end uses the hostname to determine the appropriate service credential for authentication purposes (e.g., foo-0.default.pod.cluster-local). However, on the sending side, kpropd uses the hostname it is connecting to (e.g., kdc1.example.com) to generate the cryptographic secret for secure communication. These hostnames must match to ensure that the cryptographic process can generate consistent data on both ends. Any discrepancy between these hostnames can result in authentication failure due to mismatched cryptographic data.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

The hostname field of the Linux Kernel is limited to 64 bytes (see [sethostname(2)](http://man7.org/linux/man-pages/man2/sethostname.2.html)), while most Kubernetes resource types require a name as defined in [RFC 1123](https://tools.ietf.org/html/rfc1123), which limits them to 63 bytes. Kubernetes attempts to include the Pod name as hostname, unless this limit is reached. When the limit is reached, Kubernetes has a series of mechanisms to deal with the issue. These include, truncating Pod hostname when a “Naked” Pod name is longer than 63 bytes, and having an alternative way of generating Pod names when they are part of a Controller, like a Deployment. Without any remediation, users might hit the 64 bytes kernel hostname limit, and Kubernetes will fail to create the Pod Sandbox and the pod will remain in `ContainerCreating` (Pending status) forever. 

To alleviate this problem, we will determine whether the value of `actualPodHostname` is greater than 63 bytes during the creation process and give a warning.

## Design Details

We are introducing a new feature gate called `SetAnyFQDNAsPodHostname`. When this feature gate is enabled, users can add the `SetAnyFQDNAsPodHostname` field in the podSpec.

During the pod creation process, we will validate whether the value of `SetAnyFQDNAsPodHostname` exceeds 63 bytes. If it does, a warning message will be output.

Additionally, in the `generatePodSandboxConfig` method of kubelet, the pod's hostname will always be overridden with the value of `SetAnyFQDNAsPodHostname`, and it will be written in the pod's `/etc/hosts`.

Based on the above design, after the KEP is implemented, we can achieve the following results.

| Behavior                                                     | result for `hostname`                                | result for `hostname -f`                             | DNS Record                                           |
| ------------------------------------------------------------ | ---------------------------------------------------- | ---------------------------------------------------- | ---------------------------------------------------- |
| hostname: busybox1                                           | busybox1                                             | busybox1                                             | None                                                 |
| hostname: busybox1<br />subdomain: busybox-subdomain         | busybox1                                             | busybox1.busybox-subdomain.default.svc.cluster.local | busybox1.busybox-subdomain.default.svc.cluster.local |
| hostname: busybox1<br />subdomain: busybox-subdomain<br />setHostnameAsFQDN: true | busybox1.busybox-subdomain.default.svc.cluster.local | busybox1.busybox-subdomain.default.svc.cluster.local | busybox1.busybox-subdomain.default.svc.cluster.local |
| actualPodHostname: www.example.com                           | www.example.com                                      | www.example.com                                      | None                                                 |
| hostname: busybox1<br />actualPodHostname: www.example.com   | www.example.com                                      | www.example.com                                      | None                                                 |
| hostname: busybox1<br />actualPodHostname: www.example.com<br />subdomain: busybox-subdomain | www.example.com                                      | www.example.com                                      | busybox1.busybox-subdomain.default.svc.cluster.local |
| hostname: busybox1<br />actualPodHostname: www.example.com<br />subdomain: busybox-subdomain<br />setHostnameAsFQDN: true | www.example.com                                      | www.example.com                                      | busybox1.busybox-subdomain.default.svc.cluster.local |



### Test Plan

##### Prerequisite testing updates

##### Unit tests

- `k8s.io/kubernetes/pkg/kubelet/kuberuntime`: `2024-07017` - `67.2%`
- `k8s.io/kubernetes/pkg/registry/core/pod` : `2024-07017` - `64.8%`

##### Integration tests

- N/A

##### e2e tests

- Add test in `test/e2e_node/` to verify if the `actualPodHostname` field is effective.

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

- Use the `SetAnyFQDNAsPodHostname` feature gate to implement this feature.
- Initial e2e tests completed and enabled.
- Add documentation for feature gates.

#### Beta

- Gather feedback from developers and surveys
- Make feature gate to be enabled by default.
- Update the feature gate documentation.

#### GA

- No issues reported during two releases.

### Upgrade / Downgrade Strategy

**Alpha**: In version 1.32, we enter Alpha with the `SetAnyFQDNAsPodHostname` feature gate disabled by default. Users can manually enable `SetAnyFQDNAsPodHostname` to use this feature. 

**Beta**: The `SetAnyFQDNAsPodHostname` feature gate is enabled by default. Users can manually disable `SetAnyFQDNAsPodHostname` to turn off this feature. 

**GA**: The `SetAnyFQDNAsPodHostname` feature is locked and enabled by default.

### Version Skew Strategy

No need to consider version skew, as older versions of kubelet will not be concerned with the `actualPodHostname` field in the podSpec.

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

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: SetAnyFQDNAsPodHostname
  - Components depending on the feature gate: kubelet, kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Using the feature gate is the only way to enable/disable this feature.

###### What happens if we reenable the feature if it was previously rolled back?

The feature should continue to work just fine.

###### Are there any tests for feature enablement/disablement?

We will manually test enabling and disabling the feature gate.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

No known failure modes.

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

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

- [x] Metrics
  - Metric name: `run_podsandbox_errors_total`
  - [Optional] Aggregation method: When the value of `actualPodHostname` exceeds 63 characters, the Pod Sandbox creation will fail, and this will be reflected in the metrics.
  - Components exposing the metric: Kubelet
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

No

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Using this feature requires adding a new field to the pod object, which will inevitably increase its size.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

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

- 2024-07-18: Initial draft KEP

## Drawbacks

This is not a standard Kubernetes use case; it is undoubtedly in conflict with the current pod's potential DNS records, and using it will bring more confusion to users. Moreover, we are not sure how much it can help traditional services that can benefit from being migrated to Kubernetes.

## Alternatives

* If the `actualPodHostname` field is set, Kubelet will always respect this field (otherwise it will revert to the old behavior). In the default or REST logic, we can see if `actualPodHostname` is not set, then we check the `hostname`, `setHostnameAsFQDN`, and the `cluster-suffix`, and write the result into `actualPodHostname`. If the user sets it themselves, we will retain it and treat it as an override, this can ultimately simplify `Kubelet` as it can remove legacy behavior, but it means teaching the `kube-apiserver` about the `cluster-suffix`, however, it is challenging to find an existing or grace way to pass the `kube-apiserver`’s configuration options in the REST or default logic.
* Repair the traditional projects that cannot be migrated to Kubernetes, or find alternatives.
* Do not add new fields, relax the validation of the `hostname` field in `podSpec` to allow it to accept strings in FQDN format, and when the `hostname` is set to FQDN, we will unconditionally ignore the `subdomain` and `setHostnameAsFQDN` fields, or to keep the current `hostname` and be able to override or omit the `default.svc.cluster.local` part. However, doing so will cause us to lose the DNS resolution records for the pod.
* Do not add new fields, allowing the value of `setHostnameAsFQDN` to be set to `Custom`, the pod's hostname can still meet our expectations. However, since `setHostnameAsFQDN` is currently a boolean type, modifying it would be disruptive to the existing API.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
