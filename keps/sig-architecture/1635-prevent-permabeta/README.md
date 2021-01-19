
# KEP-1635: Require Transition from Beta

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Impacted APIs](#impacted-apis)
  - [sig-apps](#sig-apps)
  - [sig-auth](#sig-auth)
  - [sig-instrumentation](#sig-instrumentation)
  - [sig-network](#sig-network)
  - [sig-node](#sig-node)
  - [sig-scheduling](#sig-scheduling)
- [Drawbacks](#drawbacks)
- [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

- [ ] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

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
   This will likely be a future goal, but it isn't the goal of this KEP.

## Proposal

Once a REST API reaches beta, it has nine months to 
1. reach GA and deprecate the beta or 
2. have a new beta version and deprecate the previous beta.

If neither of those conditions met, the beta REST API is deprecated in the third release with a stated intent to remove the REST API entirely.
To avoid removal, the REST API must create a new beta version (it cannot go directly from deprecated to GA).

This means that every beta API will be deprecated in nine months and removed in 18 months.

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

### sig-apps
1. cronjobs.v1beta1.batch

### sig-auth
1. certificatesigningrequests.v1beta1.certificates.k8s.io
2. podsecuritypolicies.v1beta1.policy

### sig-instrumentation
1. events.v1beta1.events.k8s.io

### sig-network
1. endpointslices.v1beta1.discovery.k8s.io
2. ingresses.v1beta1.networking.k8s.io
3. ingressclasses.v1beta1.networking.k8s.io

### sig-node
1. runtimeclasses.v1beta1.node.k8s.io

### sig-scheduling
1. poddisruptionbudgets.v1beta1.policy

## Drawbacks

1. Consumers of beta APIs will be made aware of the status of the APIs and be given clear dates in documentation about
when they will have to update.  If the maintainers of these beta APIs do not graduate their API, a new beta version will
need to exist within 18-ish months and early adopters will have to update their manifests to the new version.  

## Upgrade / Downgrade Strategy

To ensure adherence, the kube-apiserver automatically stops serving expired beta APIs.
To avoid disruption to developers, there is a flow to handle removing these APIs.
 1. For alpha levels of a release, the expired beta APIs are served.
 2. The grace for an alpha level can be removed in a PR by setting 
    (strictRemovedHandlingInAlpha=true)[https://github.com/kubernetes/kubernetes/blob/73d4c245ef870390b052a070134f7c4751744037/pkg/controlplane/deleted_kinds.go#L72]
 3. The PR will highlight tests and code that need to be updated to react to the removed beta API.
 4. Updates to handle beta removal can be made before the first beta.0 is tagged.
 5. You know you're done when the PR from step 2 passes.
Following these steps will prevent any disruption to the kube development flow when expired APIs are automatically excluded
from the the kube-apiserver.

While the code is automatically enforcing, individual sigs can ease their transition by planning for the removal aspect
in their upgrade/downgrade strategies.
For instance, they could stop using the beta APIs in the second releases after the introduction of GA APIs.
This would maintain the +/-1 aspect of skew, while preventing a failure to communicate.
In addition, a sig can create issues targeted at the release removing the beta API to address any generation and verify
script behavior that needs to change.
