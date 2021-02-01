# KEP-2398: Protect critical cluster requests from badly configured admission webhooks

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
  To be fulfilled
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  To be fulfilled
- [Implementation History](#implementation-history)
  To be fulfilled
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
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

Badly configured admission webhooks could reject/fail critical cluster requests, i.e. `kube-controller-manager`(KCM) 
and `kube-scheduler`(KSH) lease renewal, which will put the KCM or KSH in a bad state. 

This proposal covers an approach protecting cluster critical requests from badly configured admission webhook. 
 

## Motivation

When an admission webhook with `failurePolicy` set to `fail` goes down, the admission of all matching requests defined 
by the webhook will be rejected by `kube-apiserver`. Similarly, a webhook with `timeoutSeconds` set to too long could 
potentially cause the client to give up on the requests.

Components like KCM or KSH relies on updates on lease object for 
leader election, and will break if failing to renew leadership. This is especially true if certain percentage,
say 50%, of the requests against the bad admission webhook fail, which could cause all KCM replicas to spin.

There are many admission webhooks that intercept requests against resources within all namespaces, i.e. 
[opa](https://github.com/open-policy-agent/opa), [Gatekeeper](https://github.com/open-policy-agent/gatekeeper), 
and countless customer created ones, which will interfere with KCM/KSH leases.

This is especially painful if the maintainer of KCM and admission webhook are different parties. For example, 
in a managed kubernetes cluster, KCM and KSH are maintained by the cloud provider while the admission webhook is owned 
by the customer. It is undesirable that actions from customer could break components owned by the cloud provider.


### Goals

* Even with a badly configured admission webhook, the KCM/KSH leader election leases can still be renewed.

### Non-Goals

* Invent another leader election mechanism, i.e. by talking to a database.
* Protect objects other than KCM or KSH leases from bad webhooks.


## Proposal

Ideally, we want to bypass webhook for requests to a hard-coded list of KCM/KSH lease objects. However, since the lease 
object name, namespace or type are configurable in KCM/KSH, the proposal is to have `kube-apiserver` take the KCM/KSH 
object name, namespaces and type as flags. If nothing is set, existing behavior is preserved (mutating requests against 
KCM/KSH lease will still go through admission webhooks).  


### Risks and Mitigations


## Design Details

To be fulfilled once agreed upon on the right approach.


## Production Readiness Review Questionnaire

## Implementation History


## Drawbacks

It might surprise certain people that requests against certain objects bypass admission webhooks (today the only exceptions are
`MutatingWebhookConfiguration` and `ValidatingWebhookConfiguration`).

## Alternatives

Rather than completely bypassing admission webhooks, we could alternatively create a client side `failPolicy` and 
`timeoutSeconds` which overrides the ones defined by webhook configuration for KCM/KSH leases. The downside of this 
approach is that a healthy webhook could still intentionally or unintentionally (i.e. with a bug) reject to admit 
mutating requests against KCM/KSH leases.
