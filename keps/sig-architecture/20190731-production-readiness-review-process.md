---
title: Production Readiness Review Process
authors:
  - "@johnbelamaric"
owning-sig: sig-architecture
participating-sigs:
  - sig-release
reviewers:
  - "@derekwaynecarr"
  - "@vishh"
approvers:
  - TBD
editor: TBD
creation-date: 2019-07-31
last-updated: 2019-08-01
status: provisional
---

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
- [Implementation History](#implementation-history)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
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
ensure that features merging into Kubernetes are supportable, can be safely
operated in production environments, and can be disabled or rolled back in the
event they cause increased failures in production.

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

* TBD: Document some experiences that motivate the need for this.

### Goals

* Define production readiness criteria for alpha, beta, and GA features.
* Define a production readiness review gate and process for all features.
* Utilize existing tooling with prow to enforce the process.

### Non-Goals

* Building new tooling to enforce the process.

## Proposal

* Document production readiness criteria in a process document in the
  kubernetes/community repository. Different levels of readiness may be
  specified for different feature maturity levels.

* Develop a production readiness questionnaire to ensure that the feature
  authors consider and document operational aspects of the feature. The results
  of this questionnaire will be included in playbook for the feature (the
  creation of this playbook should be one of the production readiness criteria).

  The actual questionnaire will be part of the implementation of this KEP, but
  some example items include:
  * Feature enablement and rollback
    - How can this feature be enabled / disabled in a live cluster?
    - Can the feature be disabled once it has been enabled (i.e., can we roll
      back the enablement)?
    - Will enabling / disabling the feature require downtime for the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node?
    - What happens if a cluster with this feature enabled is rolled back? What
      happens if it is subsequently upgraded again?
    - Are there tests for this?
  * Dependencies
    - Does this feature depend on any specific services running in the cluster
      (e.g., a metrics service)?
    - How does this feature respond to complete failures of the services on
      which it depends?
    - How does this feature respond to degraded performance or high error rates
      from services on which it depends?
  * Monitoring requirements
    - How can an operator determine if the feature is in use by workloads?
    - How can an operator determine if the feature is functioning properly?
    - What are the service level indicators an operator can use to determine the
      health of the service?
    - What are reasonable service level objectives for the feature?
  * Troubleshooting
    - What are the known failure modes?
    - How can those be detected via metrics or logs?
    - What are the most useful log messages and what logging levels do they require?
  * etc.

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

TBD - this section should include the label name and the description of prow
configuration that is needed

## Implementation History

- 2019-07-31: Created

## Infrastructure Needed

TBD
