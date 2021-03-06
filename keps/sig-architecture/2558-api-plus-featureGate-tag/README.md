# KEP-2558: Add a +featureGate tag to API files

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
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
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
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

As an API reviewer and consumer, I find myself looking for information about
which featureGate covers a given field.  Many API producers include it in their
comments but that is free-form, and not amenable to automation.  This KEP
proposes a new tag, `+featureGate=` which can be specified.

## Motivation

As an API reviewer and consumer, I find myself looking for information about
which featureGate covers a given field.  Many API producers include it in their
comments but that is free-form, and not amenable to automation.

### Goals

* Make it clearer when a field is governed by a feature gate.
* Enable the next step of automation and tooling to use these tags.
* Expose this info in openapi and/or other API documents.

### Non-Goals

* To design the automation or documentation
* To use these tags to filter API discovery

## Proposal

This KEP proposes a new API comment tag: `+featureGate=<gate name>`.  This can
be specified on API fields which are optionally filtered out by that gate
(alpha and beta).  When a gate goes GA (locked on) this tag can be removed.

<<[UNRESOLVED]>>
Open questions:

1) Should we also add a +alpha/+beta tag (redundant with the tag struct, but
more accessible), or should tooling derive that by looking up the gate name
(one place to assert).

2) Should we also tag structs or even new Kinds when they are gated?  This is
less common and less well-defined.

3) Should we tag *all* fields, even those that are not gated, so that tooling
can know that the absence of the tag is an error, rather than indicating GA.
<<[/UNRESOLVED]>>

### Notes/Constraints/Caveats (Optional)

NOTE: This KEP is only proposing to normalize the tag, not yet defining the
tooling.  As such, no code change is expected.

### Risks and Mitigations

Once we introduce this it is YET ANOTHER checkbox during API review.  Then we
have to deal with the implications of forgetting it on a field.  Tooling might
incorrectly assume that the absence of this tag means a field is GA.

## Design Details

Use and document `+featureGate=<gate name>`.

### Test Plan

N/A

### Graduation Criteria

This KEP does not really have an alpha or beta stage.  It is "done" when the
preponderance of existing gated fields use this tag.

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

N/A

###### Does enabling the feature change any default behavior?

N/A

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

N/A

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

N/A

### Rollout, Upgrade and Rollback Planning

N/A

###### How can a rollout fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A

### Monitoring Requirements

N/A

###### How can an operator determine if the feature is in use by workloads?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

N/A

###### Does this feature depend on any specific services running in the cluster?

N/A

### Scalability

N/A

###### Will enabling / using this feature result in any new API calls?

N/A

###### Will enabling / using this feature result in introducing new API types?

N/A

###### Will enabling / using this feature result in any new calls to the cloud provider?

N/A

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

N/A

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

N/A

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

N/A

### Troubleshooting

N/A

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

March 6, 2021: First draft

## Drawbacks

It's one more hoop to jump through that can be forgotten.

## Alternatives

Use a proper IDL.
