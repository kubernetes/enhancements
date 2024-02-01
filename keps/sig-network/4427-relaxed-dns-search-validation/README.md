# KEP-4427: Relaxed DNS search string validation

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
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
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

Currently, Kubernetes validates search string in the `dnsConfig.searches` according to [RFC-1123](https://datatracker.ietf.org/doc/html/rfc1123)
which defines restrictions for hostnames. While most DNS names identify hosts, there are record types (like SRV) that don't. For these, it's less clear
whether hostname restrictions apply, for example [RFC-1035 Section 2.3.1](https://datatracker.ietf.org/doc/html/rfc1035#section-2.3.1) points out
that it's better to stick with valid host names but also states that labels must meet the hostname requirements.

In practice, legcay workloads sometimes include an underscore (`_`) in DNS names and DNS servers will generally allow this.

This KEP proposes relaxing the checks on DNS search strings only. Allowing these values in the `searches` field of `dnsConfig` allows pods to
resolve short names properly in cases where the search string contains an underscore.

## Motivation

For workloads that resolve short DNS names where the full DNS name includes disallowed characters (like underscores),
it’s not possible to configure search strings using dnsConfig. For example, if a pod needs to look up an SRV record `_sip._tcp.abc_d.example.com`
using the short name of `_sip._tcp`, we would like to be able to add `abc_d.example.com` to the searches in the dnsConfig.

Here’s an example configuration which would support this case:

```
apiVersion: v1
kind: Pod
metadata:
  namespace: default
  name: dns-example
spec:
  containers:
    - name: test
      image: nginx
  dnsPolicy: "None"
  dnsConfig:
    nameservers:
      - 1.2.3.4
    searches:
      - abc_d.example.com
```

However, this returns an error:

```
The Pod "dns-example" is invalid: spec.dnsConfig.searches[0]: Invalid value: "abc_d.example.com": a DNS-1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example.com', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')
```

Allowing underscores in the search string allows integration with legacy workloads without allowing anyone to define
these names within Kubernetes. Since having underscores in a name creates other issues (such as inability to obtain a publicly trusted TLS certificate),
search strings seem like the only area where this is likely to occur.

### Goals

- Support workloads that need to resolve DNS short names where the full DNS name includes an underscore (`_`).

## Proposal

Introduce a RelaxedDNSSearchValidation feature gate which is disabled by default. When the feature gate is enabled,
a new DNS name validation function will be used, which keeps the existing check but also allows an underscore (`_`) in any place
where a dash (`-`) would be allowed currently.

Since the relaxed check allows previously invalid values, care must be taken to support cluster downgrades safely. To accomplish this, the validation will distinguish between new resources and updates to existing resources:
-	When the feature gate is disabled:
  - (a) New resources will use strict validation based on RFC-1123 (no change to current validation)
  - (b) Updates to existing resources will use relaxed validation if any search string in the existing list fails strict validation
- When the feature gate is enabled:
  - (c) New resources will use relaxed validation.
  - (d) Updates to existing resources will use relaxed validation.

This means that it is safe to downgrade a cluster with the feature gate enabled to a version where the feature gate is present (whether it’s enabled or disabled). It is not safe, in general, to downgrade from a cluster with the gate enabled to a version prior to the gate being introduced, since values may have been written to storage which will no longer pass validation. However, this scenario requires opting in through enabling the gate. In practice, the recommended approach would be to only enable to the gate after upgrading from a version with relaxed checking already present.

As long as the gate is disabled, there is no compatibility change, so cluster downgrades are not affected by the feature.

### Risks and Mitigations

The change is opt-in, since it requires configuring a search string with an underscore. So there is no risk beyond
the upgrade/downgrade risks which are addressed in the Proposal section.

## Design Details

See Proposal

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

Added validation will be covered by unit tests along with unit tests covering the behavior
when the gate is enabled or disabled.

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

##### e2e tests

Add a test that verifies when an underscore is present in the `dnsConfig.searches` that it appears in resolv.conf in the pod.

### Graduation Criteria

#### Alpha
- Feature implementd behind a gate
- Initial e2e tests completed and enabled

#### Beta
- No trouble reports from alpha release

#### GA
- No trouble reports with the beta release, plus some anecdotal evidence of it being used successfully.

### Upgrade / Downgrade Strategy

See Proposal section.

### Version Skew Strategy

Kubelet only checks size limits but otherwise passes values through
[source](https://github.com/kubernetes/kubernetes/blob/f025a96d2f60984765731e01ad0de2c89e959b42/pkg/kubelet/network/dns/dns.go#L114).

Since the resolv.conf file is interpreted by the DNS resolver in the container image and not by the container runtime, the change
does not depend on the container runtime or its version.

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
  - Feature gate name: RelaxedDNSSearchValidation
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

No, there is no change to default behavior.
This is a change to validation.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?


Yes, the feature can be disabled. Proposal covers the validation logic in detail,
but briefly, existing values will be allowed with relaxed validation if the gate is disabled.

###### What happens if we reenable the feature if it was previously rolled back?

Then the relaxed validation will be allowed on new values in `dnsConfig.searches`.
Existing values (prior to the initial roll-back) will continue to pass validation regardless
of whether the gate is enabled or not.

###### Are there any tests for feature enablement/disablement?

Unit tests will cover cover the scenarios described in the Proposal section.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

Since this a change to validation behind a feature gate, rollout should pose no risk.

If a cluster needs to be rolled back for another reason, it's risky to enable this
feature unless the previous version also has the flag (whether it's enabled or disabled).

Since this feature allows previously invalid values in `dnsConfig.searches`, upgrading
from a version without the gate present (i.e. before introducing this feature) and then
enabling the gate is risky. In that scenario, if a search path is saved containing an
underscore and then the cluster is downgraded to a previous version with no knowledge
of the feature gate, then the downgrade may fail.

See the Proposal section for recommendation on avoding this scenario.

###### What specific metrics should inform a rollback?

N/A


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

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
Existence of an underscore the `dnsConfig.searches` array in any pod spec or pod spec template
would indicate the feature is in use.

###### How can someone using this feature know that it is working for their instance?

N/A. This is a validation change.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A


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

No. This is a change to API validation.

### Scalability

N/A

###### Will enabling / using this feature result in any new API calls?

No. This is a change to validation of existing API calls.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

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

N/A. This is a change to validation within the API server.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

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

Since it isn't possible to distinguish between record types a search string will be used for,
this also allows users to configure a pod that will use search string to from a hostname with
an underscore. The risk here is born by the user and the name is not defined within Kubernetes in
this case (instead it refers to a name configured outside the cluster).

## Alternatives

A workaround is to re-write the resolv.conf file from inside the pod. This typically requires running
the pod with higher privileges than the actual workload requires, however.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
