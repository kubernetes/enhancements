# Glossary

- [API Review](#api-review)
- [Code Freeze](#code-freeze)
- [Exceptions](#exceptions)
- [Enhancements Freeze](#enhancements-freeze)
- [Kubernetes Enhancements Proposal (KEP)](#kubernetes-enhancement-proposal-kep)
- [Production Readiness Review (PRR)](#production-readiness-review-prr)

### API Review
The API review process is intended to maintain logical and functional integrity of the
 API over time, the consistency of user experience and the ability of previously
 written tools to function with new APIs.

If an enhancement is considered to be changing the Kubernetes core API in any way,
 a mandatory API review is required. An Enhancement owner should proactively request an
 API review for an enhancements PR by adding a `/label api-review` comment.
 The status of an API review can be found in the
 [API review backlog](https://github.com/orgs/kubernetes/projects/13).

More details can be found in [Kubernetes API Review Process](https://github.com/kubernetes/community/blob/master/sig-architecture/api-review-process.md).

### Code Freeze
All enhancements going into the release must be code-complete by the code freeze deadline.
 Any unmerged pull requests in k/k repo for the upcoming release milestone will be removed
 by the Release Team and will require an approved exception to have the milestone added back
 by the Release Team.

More details can be found in [Code Freeze](https://github.com/kubernetes/sig-release/blob/master/releases/release_phases.md#code-freeze).

### Exceptions
When an enhancement fails to meet the release deadline, Enhancement owner must file an
 [exception request](https://github.com/kubernetes/sig-release/blob/master/releases/EXCEPTIONS.md#requesting-an-exception)
 addressed to SIG sponsoring the enhancement, the Release Team, and SIG Release to gain approval.
 The Release Team will be responsible for approving or rejecting exceptions and the decision
 will be based on discussion with the sponsoring SIG's chairs/leads and any other participating
 SIG's chairs/leads.

More details can be found in [Exceptions to Milestone Enhancement Complete dates](https://github.com/kubernetes/sig-release/blob/master/releases/EXCEPTIONS.md).

### Enhancements Freeze
All enhancements going into the release must meet the [required criteria](https://github.com/kubernetes/sig-release/blob/master/releases/release_phases.md#enhancements-freeze) by the enhancements freeze deadline.

Any enhancements that do not meet all of the criteria will be removed from tracking for the upcoming release.
 Any unmerged pull requests in k/enhancements repo for the upcoming release milestone will be removed
 by the Release Team and will require an approved exception to have the milestone added back and the enhancement
 tracked again by the Release Team.

More details can be found in [Enhancements Freeze](https://github.com/kubernetes/sig-release/blob/master/releases/release_phases.md#enhancements-freeze).

### Kubernetes Enhancement Proposal (KEP)
A Kubernetes Enhancement Proposal is a way to propose, communicate and coordinate on new efforts
 for the Kubernetes project.

A KEP should be created after socializing an idea with the sponsoring and participating SIGs.
 To create a KEP, the [KEP template](https://github.com/kubernetes/enhancements/blob/master/keps/NNNN-kep-template/README.md)
 should be used and follow the process outlined in the template.

Use the [definition of what is an Enhancement](https://github.com/kubernetes/enhancements#is-my-thing-an-enhancement)
 to determine if a KEP is required.

More details can be found in [Kubernetes Enhancement Proposals](https://github.com/kubernetes/enhancements/blob/master/keps/README.md#kubernetes-enhancement-proposals-keps)
 and [Enhancements](https://github.com/kubernetes/enhancements/blob/master/README.md).

### Production Readiness Review (PRR)
Production Readiness Reviews are intended to ensure that features merging into Kubernetes
 are observable, scalable and supportable, can be safely operated in production environments,
 and can be disabled or rolled back in the event they cause increased failures in production.

PRR approval is a requirement for the enhancement to be part of the upcoming release and all 
 enhancements owners must [submit a KEP for production readiness approval](https://github.com/kubernetes/community/blob/master/sig-architecture/production-readiness.md#submitting-a-kep-for-production-readiness-approval)
 before the Enhancements Freeze deadline.

To request a Production Readiness Review, assign a PRR approver from the `prod-readiness-approvers`
 list in the [OWNERS_ALIASES](https://github.com/kubernetes/enhancements/blob/662e4553eee3939442c88e6cdaef4c776b564b22/OWNERS_ALIASES#L193)
 file and reach out to the production readiness approvers in slack channel #prod-readiness.

More details can be found in [Production Readiness Review Process](https://github.com/kubernetes/community/blob/master/sig-architecture/production-readiness.md#production-readiness-review-process)
 and [PRR KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-architecture/1194-prod-readiness).
