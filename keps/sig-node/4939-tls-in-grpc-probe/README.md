<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [x] **Merge early and iterate.**
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
# KEP-4939: Support TLS in GRPC Probe

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

The new gRPC health probe enables developers to probe
[gRPC health servers](https://github.com/grpc-ecosystem/grpc-health-probe)
from the node.
This allows them to stop using workarounds such as this
[grpc-health-probe](https://github.com/grpc-ecosystem/grpc-health-probe)
paired with `exec` probes.

It allows natively running health checks on gRPC services
without deploying additional binaries as well as other benefits
outlined in [the announcement](https://kubernetes.io/blog/2022/05/13/grpc-probes-now-in-beta/).

A limitation in the current implementation is that it only supports
gRPC servers that do not leverage TLS connections.
Even if they are not concerned about certificate verification for the health check,
a connection cannot be established at all if the server is expecting TLS and the client
is not.

This enhancement aims to add configuration options to enable TLS on the gRPC probe.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

We often deploy internal gRPC services on our cluster.
These deployments provide internal services and it's simple to add
the health server to them so they can be verified through a single interface.

It's also worth noting we have and internal CA that signs certs for communicating
with these servers and all of them use TLS.

Currently, we are using the `exec` probe for readiness and liveness configured as:

```
liveness_probe {
  exec {
    command = [
      "/bin/grpc_health_probe",
      "-addr=:8443",
      "-tls",
      "-tls-no-verify",
    ]
  }
}
```

We would really like to switch to the gRPC probes introduced in 1.24
but are unable to do so since there is no way to configure it to use a TLS connection
when reaching out to the health server.

Instead we must continue to rely on the `exec` probe and cannot reap the benefits
described in [the announcement](https://kubernetes.io/blog/2022/05/13/grpc-probes-now-in-beta/).

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

The primary goal is to support TLS connections when using the `grpc` probe.
The probe will use TLS but not verify the certificate.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

It is not a goal of this KEP to support providing a certificate to verify the TLS
connection.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

I would like to add new configuration fields alongside `port` and `service` in the
[Probe GRPCAction](https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#Probe).

They can be used to indicate whether or not TLS should be used
and can serve as a basis for future TLS related functionality if desired.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

1. Adds more code to Kubelet and surface area to `Pod.Spec`

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

We should add a new struct to
[GRPCAction](https://github.com/kubernetes/kubernetes/blob/f422a58398d457a88ac7b05afca85a0e6b805605/pkg/apis/core/types.go#L2665-L2676)
named `tls` which is optional.
It would have a single field `mode` which only will have one value currently `NoVerify`.

The presence of the `tls` object would be enough to indicate a desire to use TLS.

For example, these configurations would enable TLS with no verification:

```
grpcAction:
    port: 12345
    tls:
        mode: NoVerify
```
```
grpcAction:
    port: 12345
    tls:
```

This configuration would disable TLS:

```
grpcAction:
    port: 12345
```

I have identified the relevant code changes once the configuration
structure is hammered out.

Currently, the gRPC probe uses `insecure.NewCredentials()` when
[establishing the connection](https://github.com/kubernetes/kubernetes/blob/d9441212d3e11dc13198f9d4df273c3555ecad11/pkg/probe/grpc/grpc.go#L59).

If configured to use TLS, that `DialOption` should be replaced with:

```go
tlsConfig := &tls.Config{
    InsecureSkipVerify: true,
}
tlsCredentials := credentials.NewTLS(tlsConfig)

opts := []grpc.DialOption{
    // ...
    grpc.WithTransportCredentials(tlsCredentials),
    // ...
}
```

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

Plan is to add unit tests to `pkg/probe/grpc/grpc_test.go`
that can verify the flags are interpreted correctly
and that a TLS (with no verify) and non-TLS configurations work as expected.

- `k8s.io/kubernetes/pkg/probe/grpc`: TBD

##### Integration tests

N/A, only unit tests and e2e coverage.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

Tests in test/e2e/common/node/container_probe.go:
- should not be restarted with a GRPC liveness probe: TBD
- should be restarted with a GRPC liveness probe: TBD

### Graduation Criteria

#### Alpha

- Implement the feature.
- Add unit and e2e tests for the feature.

#### Beta

- Solicit feedback from the Alpha.
- Ensure tests are stable and passing.

#### GA
- [ ] Address feedback from beta usage
- [ ] Validate that API is appropriate for users
- [ ] Close on any remaining open issues & bugs

### Upgrade / Downgrade Strategy

Upgrade: default values for new configurables should default to the current state so upgrade should not require anything.

Downgrade: gRPC probes will only support TLS in a downgrade from alpha

### Version Skew Strategy

Ths issue here is when nodes are behind the control plane,
the older nodes will receive the `tls` config but ignore it which would
cause probes to fail if TLS is required.

We may not be able to graduate this widely until all kubelet version
skew supports the new `tls` configuration.

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `GRPCContainerProbeTLS`
  - Components depending on the feature gate:
    - `kubelet` (probing)
    - API server (API changes)

###### Does enabling the feature change any default behavior?

No, it should be designed so that omitting the `tls`
configuration causes it to not use TLS which is the default behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Removing the `tls` configuration should cause the probe to work as it did before.

###### What happens if we reenable the feature if it was previously rolled back?

Re-applying the `tls` configuration would cause the probe to start using TLS again.

###### Are there any tests for feature enablement/disablement?

Unit tests can be implemented to verify the gRPC probe
behavees correctly with the feature enabled or disabled.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

Enabling or disabling the `tls` feature should only affect the next run of the probe.
It wouldn't "fail" but it would revert to not using TLS which would be an issue if the running gRPC server is expecting TLS.

###### What specific metrics should inform a rollback?

Rollback wouldn't address issues. Pods will need to stop using the new probe
type.

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

When gRPC probe is configured, Pod must be scheduled and, the metric
`probe_total` can be observed to see the result of probe execution.

Event will be emitted for the failed probe and logs available in `kubelet.log`
to troubleshoot the failing probes.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Probe must leverage TLS as configured and succeed 
whenever service has returned the correct response
in defined timeout, and fail otherwise.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

The metric `probe_total` can be used to check for the probe result. Event and
`kubelet.log` log entries can be observed to troubleshoot issues.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

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

A new type for the `tls` config will be added to the `grpcAction` type.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

If configured, they will increase the config for the `grpcProbe` to include the `tls` option.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

There could be a marginal increase due to TLS vs. the default Non-TLS

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

There could be a marginal increase due to TLS vs. the default Non-TLS

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Taken from the 
[original KEP](https://github.com/kubernetes/enhancements/blob/4855713ab20b9653b3b715ded82d772bb38a8108/keps/sig-node/2727-grpc-probe/README.md#can-enabling--using-this-feature-result-in-resource-exhaustion-of-some-node-resources-pids-sockets-inodes-etc)
for the gRPC probe since it's still applicable:

Yes, gRPC probes use node resources to establish connection.
This may lead to issue like [kubernetes/kubernetes#89898](https://github.com/kubernetes/kubernetes/issues/89898).

The node resources for gRPC probes can be exhausted by a Pod with HostPort
making many connections to different destinations or any other process on a node.
This problem cannot be addressed generically.

However, the design where node resources are being used for gRPC probes works
for the most setups. The default pods maximum is `110`. There are currently
no limits on number of containers. The number of containers is limited by the
amount of resources requested by these containers. With the fix limiting
the `TIME_WAIT` for the socket to 1 second,
[this calculation](https://github.com/kubernetes/kubernetes/issues/89898#issuecomment-1383207322)
demonstrates it will be hard to reach the limits on sockets.

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

No dependency on etcd availability.

###### What are other known failure modes?

None

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

* 2021-11-17: [Original gRPC Probe implementation](https://github.com/kubernetes/kubernetes/commit/b7affcced15923b8a45510301a90542eec232c49)

## Drawbacks

N/A

## Alternatives

[Discussed](https://github.com/kubernetes/enhancements/pull/5029#discussion_r1936341743)
using a boolean parameter instead of the `tls` struct.
Decided to use the approach in order to leave fexibility for 
future TLS related configuration such as certificate verification.

