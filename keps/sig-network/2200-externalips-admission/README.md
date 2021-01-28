# KEP-2200: Deny use of ExternalIPs via admission control

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [X] Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This proposal is in response to CVE-2020-8554: "Man in the middle using
LoadBalancer or ExternalIPs".

Fundamentally the `Service.spec.externalIPs[]` feature is bad.  It predates
`Service.spec.type=LoadBalancer` and, now that we have that, has very few
use-cases.  In short an unprivileged user can hijack an IP address via a
Service spec.  In contrast, `type=LoadBalancer` uses Service status, which most
normal users should not be allowed to write.

This KEP proposes to block the use of ExternalIPs via a built-in admission
controller.  The justification for this, as opposed to a webhook, is that 99%
of users will never use this feature, and making them ALL run a webhook seems
terrible.

## Motivation

https://github.com/kubernetes/kubernetes/issues/97110

### Goals

Make it possible to disable an insecure feature for the vast majority of users
very quickly.

### Non-Goals

* Make this the default (breaking change)
* Make the feature safe to use.

## Proposal

This KEP proposes to add a built-in admission controller
"DenyServiceExternalIPs", which rejects any CREATE or UPDATE operation which
adds a new value to `Service.spec.externalIPs`.  Existing values will be
tolerated and may be removed.

The number of rejected operations will be exposed by the standard admission
metrics (`apiserver_admission_controller_admission_duration_seconds_bucket{name="DenyServiceExternalIPs",rejected="true", ...}`).

### User Stories (Optional)

Alice the admin does not want her users using this insecure feature.  She
enabled this admission controller and knows no user can use it.  She can then
audit existing users and make them stop.

### Risks and Mitigations

Some installations may want to use this feature in a more controlled way.  They
can use a custom webhook admission controller or a policy controller to enforce
their own rules.

This is a precedent we should not set lightly.  In this case the VAST majority
of users do not need this feature and this proposal is very surgical in nature.
As far as we know, there are few other unprivileged fields with this much
power anywhere in our API, and most of those already have some form of controls
on them.

## Design Details

One simple admission controller should be enough to disable this misfeature.
Unfortunately it can not be on by default (that would be breaking).

This means that platform-providers may need to expose an option to control
this.  While we generally try to avoid mixing knobs that cluster-users would
set with knobs that cluster-providers own, it seems reasonable to close this as
soon as possible and consider better answers when we have more cases to
generalize from.  See "Alternatives" below for more.

See "Proposal" above.

### Test Plan

* Unit tests to ensure CREATE and UPDATE operations are rejected when adding
  new `externalIPs`.
* Unit tests to ensure UPDATE operations allow existing `externalIPs`.

### Graduation Criteria

This feature will debut as "GA", bypassing alpha and beta.  It's already opt-in
and very small scope.

### Upgrade / Downgrade Strategy

Cluster upgrades/downgrades should not be an issue.

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Other flag
    - Flag name: --enable-admission-plugins (existing)

* **Does enabling the feature change any default behavior?**
  Yes.  The `externalIPs` field will not be allowed to mutate, except to remove
  existing values.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?**
  Yes.

* **What happens if we reenable the feature if it was previously rolled back?**
  No problem.

* **Are there any tests for feature enablement/disablement?**
  Unit tests should suffice.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
  It could start disallowing all Service operations, if the controller was
  buggy.

* **What specific metrics should inform a rollback?**
  `apiserver_admission_controller_admission_duration_seconds_bucket{name="DenyServiceExternalIPs",rejected="true", ...}`

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Manual testing:
  * Create a service "extip" with 2 `externalIPs` values
  * Upgrade to new apiserver and enable new admission controller
  * Try to create a new service using `externalIPs` -> fail
  * Try to change the "extip" service in an unrelated way -> OK
  * Try to change the value of one `externalIPs` value in extip -> fail
  * Try to remove the [0] value of `externalIPs` -> OK
  * Try to add the removed value back -> fail
  * Remove the last `externalIPs` value -> OK
  * Try to add the removed value back -> fail
  * Revert to "standard" apiserver
  * Try to add the removed value back -> OK

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?**
  No.

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
  There are two possible facets of this: 1) Is the admission control enabled?
  and 2) Are any users using externalIPs?

  To point 1, admins can look at their admission control config
  (--enable-admission-plugins) and look for `DenyServiceExternalIPs` in that
  list.

  To point 2, admins can look at all services in the cluster for use of
  the `externalIPs` field.  Via kubectl:

  ```
  kubectl get svc --all-namespaces -o go-template='
  {{- range .items -}}
    {{if .spec.externalIPs -}}
      {{.metadata.namespace}}/{{.metadata.name}}: {{.spec.externalIPs}}{{"\n"}}
    {{- end}}
  {{- end -}}
  '
  ```

* **What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?**
  N/A

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  N/A

* **Are there any missing metrics that would be useful to have to improve observability of this feature?**
  This proposes to use the existing
  `apiserver_admission_controller_admission_duration_seconds_bucket{name="DenyServiceExternalIPs", ...}` metrics.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  No.

### Scalability

* **Will enabling / using this feature result in any new API calls?**
 No.

* **Will enabling / using this feature result in introducing new API types?**
 No.

* **Will enabling / using this feature result in any new calls to the cloud provider?**
 No.

* **Will enabling / using this feature result in increasing size or count of the existing API objects?**
 No.

* **Will enabling / using this feature result in increasing time taken by any operations covered by [existing SLIs/SLOs]?**
 No.

* **Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?**
 No.

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**
  It is part of apiserver REST path.

* **What are other known failure modes?**
  None.

* **What steps should be taken if SLOs are not being met to determine the problem?**
  N/A

## Implementation History

* 2020-12-07: First draft
* 2021-01-04: Edits to PRR section.
* 2021-01-15: Edits from feedback.

## Drawbacks

It is a slippery-slope to other ad hoc policies.  Counter: this is very
surgical and overwhelmingly not a useful feature.

Users who REALLY need this feature can enable it and apply whatever bespoke
admission policies they need (or not).

## Alternatives

* Force users to use policy controllers as webhooks. Forever.
* Make a breaking API change and disable or rip-out the feature.
* Add a new flag telling validation logic to dissallow this field.
* Make a more complex API to define which namespaces can use this feature
  and/or which IPs they can use.
* Make a new API that allows cluster-users to enable this sort of field-block
  without changing admission-control flags on apiserver.
