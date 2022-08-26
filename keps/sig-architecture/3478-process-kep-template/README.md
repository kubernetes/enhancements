# KEP-3478: Process KEP Template

<!-- toc -->
- [Signoff Checklist](#signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge a process enhancement as implemented, these
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
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

Quite a few process enhancements have been proposed using as KEPs.
Overall KEPs are good way to propose, communicate and ratify process
enhancements. But the main KEP template is a poor starting point.

This KEP proposed a KEP template written specifically for process enhancements.
Specifically, this template is intended any KEP that:

- Is exclusively a change to a process and does not require code changes of any kind.
- Is independent of the Kubernetes release cycle and enhancement freezes.
- Does not involve any changes that require production readiness review.
- Can be enacted immediately after it is approved.
- Requires only updates to process related documentation and communication with
  the appropriate process leads or groups to enact.

## Motivation

Process KEP authors are already individually trimming down the main KEP template
when authoring process KEPs. Some examples of this include:

- https://github.com/kubernetes/enhancements/tree/master/keps/sig-architecture/2527-clarify-status-observations-vs-rbac
- https://github.com/kubernetes/enhancements/tree/master/keps/sig-architecture/1635-prevent-permabeta
- https://github.com/kubernetes/enhancements/tree/master/keps/sig-architecture/1659-standard-topology-labels
- https://github.com/kubernetes/enhancements/tree/master/keps/sig-architecture/1194-prod-readiness
- https://github.com/kubernetes/enhancements/tree/master/keps/sig-architecture/3136-beta-apis-off-by-default

Determining which information in the main KEP template applies to a process KEP
is not entirely trivial. Does the KEP still need to be milestone bound to a
release? Which KEP sections are appropriate to omit? Does anything in Prod
Readiness apply? Do stability levels and graduation apply to a process
enhancement?

Without a process KEP template, each KEP author must attempt to answer these
questions individually. The goal of introducing a template it to make all those
decisions collectively so that process KEPs are less burdonsome to author, and
more consistent once written.

### Goals

Introduce a KEP template specifically for process enhancements.

### Non-Goals

Introduce mechanisms to keep the main KEP template and the new process KEP
template in sync. Except for general changes to KEPs that apply to all
templates, we expect the templates to diverge over time. We don't consider
this a problem. We believe that the vast majority of changes to the feature
enhancement KEP template will not apply to process enhancements and vis-versa.

## Proposal

Introduce a `NNNN-process-kep-template` that will be similar to `NNNN-kep-template`,
except:

- All instructions in the template will be re-written to be aimed at process
  KEP authors.
- Unnecessary sections will be dropped from both the `README.md` and
  `kep.yaml`.
- Documentation for the KEP process will call-out the existence of this
  separate template for process KEPs.

While the original KEP proposal did not explicitly suggest that multiple KEP
templates might be needed for the project as a whole, it does suggest that
[special purpose KEP templates may be useful](https://github.com/kubernetes/enhancements/blob/c7963986f074c2a712d483fdfd00e51b7c68c5d2/keps/sig-architecture/0000-kep-process/README.md?plain=1#L115-L120):

> SIGs also have the freedom to customize the KEP template
> according to their SIG-specific concerns. For example, the KEP template used to
> track API changes will likely have different subsections than the template for
> proposing governance changes.

### Risks and Mitigations

KEP authors might try to use the template for cases where it is not appropriate.
This can be mitigated by clearly listing out the conditions where this template
is appropriate. 

## Design Details

Process KEPs are not "graduated", they are "enacted". To minimize confusion
about this and leverage existing KEP concepts as much as possible,
"enacted" will be represented using the "implemented" KEP status. Also,
process KEPs will not graduate between the "alpha", "beta" and "stable"
stages, and will instead always be set to the "stable" stage.

The release process (and it's freezes) does not prevent process KEPs from
being merged or ratified. For this reason, process KEP tracking issues will
not use release milestone labels.

Structural differences from the main KEP template:

`README.md`:

- Include a list of conditions for when the process KEP template is appropriate.
- Remove "Test Plan", "Graduation Criteria", "Upgrade / Downgrade Strategy",
  "Version Skew Strategy", "Production Readiness Review Questionnaire" and
  "Implementation History" sections.
- Rewrite all instructions to be aimed at process enhancement authors.

`kep.yaml`:
- Add a `kep-type: process` field to make is easy to identify process KEPs.
- Remove `latest-milestone`, `milestone`, `feature-gates`, `disable-supported`
  and `metrics` fields.
- Add comments explaining how `status` should should be interprested for process
  KEPs.
- Continue to include `stage` but field but default it to `stable` and include
  comment explaining that process KEPs do transition through stability levels
  and that they are "enacted", not "graduated". Keeping this field is primarily
  intended to help keep `kep.yaml` files easy to process by tools.

## Drawbacks

See risks.

## Alternatives

- Interleave into the main KEP template instructions about how to handle
  process KEPs. This complicates the main template and makes it more
  burdonsome for feature enhancement authors.
- We could have limited this template to sig-architecture instead of making
  it a top level template. But this template seems generally useful to all
  SIGs.