# Production Readiness Review Process

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Phase 1 - Research and Pilot](#phase-1---research-and-pilot)
  - [Phase 2 - Implementation](#phase-2---implementation)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [n/a] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [n/a] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [n/a] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

This proposal introduces a production readiness review process for features
merging as alpha or graduating to beta or GA. The review process is intended to
ensure that features merging into Kubernetes are observable and supportable,
can be safely operated in production environments, and can be disabled or rolled
back in the event they cause increased failures in production.

## Motivation

Kubernetes has grown quickly and organically. The KEP process introduced a
mechanism to help build consensus and ensure the community supports the
direction and implementation of various features. It provides a way to document
release and graduation criteria, but it leaves that criteria up to the KEP
authors and approvers to define. Because these are normally the developers and
the SIGs associated with the KEP, there is not always a clear representation of
the view of cluster operators. This can result in operational and supportability
oversights in the graduation criteria.

This KEP proposes a process to ensure that production concerns are addressed in
all new features, at a level appropriate to the features' maturity levels.

### Goals

* Define production readiness criteria for alpha, beta, and GA features.
* Define a production readiness review gate and process for all features.
* Utilize existing tooling with prow to enforce the process.

### Non-Goals

* Building new tooling to enforce the process.
* Provide guidance for specific Kubernetes deployment models. That is,
  requirements for features should be generally applicable to Kubernetes
  deployments, not specific to use cases such as single or mult-tenant, cloud
  provider, on-prem, edge, or other modes.

## Proposal

* Document production readiness criteria in a process document in the
  kubernetes/community repository. Different levels of readiness may be
  specified for different feature maturity levels.

* Develop a production readiness questionnaire to ensure that the feature
  authors consider and document operational aspects of the feature. The results
  of this questionnaire will be included in playbook for the feature (the
  creation of this playbook should be one of the production readiness criteria).

  See [current questionnaire](https://github.com/kubernetes/community/blob/master/sig-architecture/production-readiness.md#questionnaire).

* Establish a production readiness review subproject under SIG Architecture,
  with subproject owners:
  - johnbelamaric

* Establish a production readiness review team, label, and CI check to prevent
  the merging of feature promotion PRs that lack production readiness.

### Risks and Mitigations

The primary risk is the slowing of feature merges. When this is due to the need
for the developers to improve the quality of the feature, that is appropriate.
When this is due to lack of bandwidth in the production readiness review team,
that is harmful. To mitigate this, the implementation of this process must
include a means of:
 * Identifying potential community members for participation in the team
 * A shadow program or other mechanism for preparing those individuals for
   membership on the team
 * Clear criteria for when one of these individuals is ready to become a full
   participant
 * Measurement of:
   - Review throughput for production readiness reviews
   - Team bench depth and size

## Design Details

### Phase 1 - Research and Pilot
* Targeted to the 1.17 cycle.
* Setup a pilot PRR team, which will:
  * Deliver an initial PRR questionnaire and pilot it with non-blocking PRRs for
    in-progress KEPs.
  * Deliver an interview/questionnaire form for operators and interview them on
    production issues that they experience, to ensure that the focus of this
    effort is on meaningful problems.
  * Deliver a postmortem summary of existing features that have stalled due to
    production-readiness issues (e.g., cron jobs).
* Resolve open questions, including:
  * ~~Should the scope of this expand to feature lifecycle?~~ No, not at this
    time.
  * How do we measure the effectiveness of this effort?

Status update on Phase 1:

* Pilot was conducted in 1.17 and 1.18. The questionnaire has been through several
  revisions and will be required in 1.19 as part of the KEP process.
* Research was done via a survey, a report on which will be produced, presented
  to SIG Architecture, and linked in here.
* Follow-up interviews are in the planning stage as of the beginning of the 1.19
  release cycle.
* Post-mortem of past enhancements has not been completed and is deferred until it
  is deemed necessary by the subproject team.
* Effectiveness measurement will come via a regular survey of users.

### Phase 2 - Implementation

The questionnaire was put into place in Phase 1, so what remains for Phase 2 is
to document the review process and implement the tooling supporting it.

The process will be enforced in the enhancements repository. A separate
`prod-readiness` directory will be maintained, with an OWNERS file with
PRR-specific reviewers and approvers. When targeting a release, the KEP author
must:
 * Update the `kep.yaml`, setting the `stage`, `latest-milestone`, and the
   `milestone` struct (which captures per-stage release versions).
 * Create a `prod-readiness/<sig>/<KEP number>.yaml` file, with the PRR
   approver's GitHub handle for the specific stage.

The `kepval` utility / enhancements CI will be updated to enforce this.

## Implementation History

- 2019-07-31: Created
- 2019-10-17: Review feedback, phase 1 implementable
- 2019-11-12: Add establishment of subproject
- 2020-04-21: Update for phase 2
- 2020-12-15: Mark implemented, update phase 2 details
