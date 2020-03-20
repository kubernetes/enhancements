---
title: Require Transition from Beta
authors:
  - "@deads2k"
owning-sig: sig-architecture
participating-sigs:
  - sig-api-machinery
  - sig-apps
  - sig-architecture
  - sig-auth
  - sig-network
  - sig-node
  - sig-scheduling
reviewers:
  - "@bgrant0607"
  - "@liggitt"
  - "@smarterclayton"
approvers:
  - "@dims"
  - "@derekwaynecarr"
  - "@johnbelamaric"
creation-date: 2019-10-01
last-updated: 2020-03-19
status: implementable
see-also:
replaces:
superseded-by:
---

# Require Transition from Beta

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Impacted APIs](#impacted-apis)
  - [sig-apimachinery](#sig-apimachinery)
  - [sig-apps](#sig-apps)
  - [sig-auth](#sig-auth)
  - [sig-network](#sig-network)
  - [sig-node](#sig-node)
  - [sig-scheduling](#sig-scheduling)
- [Drawbacks](#drawbacks)
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

k8s.io REST APIs should not languish in beta.  They should take feedback and progress towards GA by either
1. meeting GA criteria and getting promoted, or
2. having a new beta and deprecating the previous beta
  
This must happen within nine months (three releases).  If it does not,
the REST API will be deprecated with an announced intent to remove the API per the [deprecation policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/).

## Motivation

When a REST API reaches beta, it is turned on by default.  This is great for getting feedback, but it can also lead to state
where users and vendors start building important infrastructure against APIs that are not considered stable.
In addition, once a REST API is on by default, the incentive to further stabilize appears to diminish.
See the REST API that have been beta for a long time: CSRs and Ingresses as examples.
If we're honest with ourselves, a single actor has been cleaning up behind a lot of the project to unstick perma-beta APIs.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports

### Goals

1. Prevent k8s.io REST APIs from being in a single beta version for more than nine months.
2. Prevent beta APIs from being treated as GA by users and vendors.

### Non-Goals

1. Promote APIs to GA before they are ready.
2. Control non-k8s.io REST APIs.
3. Control features that are not REST APIs.
4. Control fields on otherwise GA REST APIs.

## Proposal

Once a REST API reaches beta, it has nine months to 
1. reach GA and deprecate the beta or 
2. have a new beta version and deprecate the previous beta.

If neither of those conditions met, the beta REST API is deprecated in the second release with a stated intent to remove the REST API entirely.
To avoid removal, the REST API must create a new beta version (it cannot go directly from deprecated to GA).

For example, in v1.16, v1beta1 is released. Sample release note and API doc:
> * "The v1beta1 version of this API will be evaluated during v1.16, v1.17, and v1.18, then deprecated in v1.19 (in favor of a new beta version, a GA version, or with no replacement), then removed in v1.22"

Scenario A - progression to v1beta2 in v1.19. Sample release note and API doc:
> * "The v1beta1 version of this API is deprecated in favor of v1beta2, and will be removed in v1.22"
> * "The v1beta2 version of this API will be evaluated during v1.19, v1.20, and v1.21, then deprecated in v1.22 (in favor of a new beta version, a GA version, or with no replacement), then removed in v1.25"

Scenario B - progression to v1 in v1.19. Sample release note and API doc:
> * "The v1beta1 version of this API is deprecated in favor of v1, and will be removed in v1.22"

Scenario C - deprecation with no replacement. Sample release note and API doc:
> * "The v1beta1 version of this API is deprecated with no replacement, and will be removed in v1.22"

By regularly having new beta versions, we can ensure that consumers will not grow long running dependencies on particular betas which could pin design decisions.
It will also create an incentive for REST API authors to push their APIs to GA instead of letting them live in a permanent beta state.

## Impacted APIs
These sigs will need to announce in 1.19 that these APIs will be deprecated no later than 1.22 and removed no later than 1.25.
This is the same as the standard for new beta APIs introduced in 1.19.

### sig-apimachinery
1. events.v1beta1.events.k8s.io

### sig-apps
1. jobtemplates.v1beta1.batch
2. cronjobs.v1beta1.batch

### sig-auth
1. certificatesigningrequests.v1beta1.certificates.k8s.io
2. podsecuritypolicies.v1beta1.policy

### sig-network
1. endpointslices.v1beta1.discovery.k8s.io
2. ingresses.v1beta1.networking.k8s.io
3. ingressclasses.v1beta1.networking.k8s.io

### sig-node
1. runtimeclasses.v1beta1.node.k8s.io

### sig-scheduling
1. poddisruptionbudgets.v1beta1.policy
2. evictions.v1beta1.policy



## Drawbacks

1. Consumers of beta APIs will be made aware of the status of the APIs and be given clear dates in documentation about
when they will have to update.  If the maintainers of these beta APIs do not graduate their API, a new beta version will
need to exist within 18-ish months and early adopters will have to update their manifests to the new version.  
