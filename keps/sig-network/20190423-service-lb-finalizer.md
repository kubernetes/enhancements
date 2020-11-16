---
title: Finalizer Protection for Service LoadBalancers
authors:
  - "@MrHohn"
owning-sig: sig-network
participating-groups:
  - sig-cloud-provider
reviewers:
  - "@andrewsykim"
  - "@bowei"
  - "@jhorwit2"
  - "@jiatongw"
approvers:
  - "@andrewsykim"
  - "@bowei"
  - "@thockin"
editor: TBD
creation-date: 2019-04-23
last-updated: 2020-01-06
status: implemented
see-also:
replaces:
superseded-by:
---

# Finalizer Protection for Service LoadBalancers

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [n+2 upgrade/downgrade is not supported](#n2-upgradedowngrade-is-not-supported)
  - [Other notes](#other-notes)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

We will be adding finalizer protection to ensure the Service resource is not
fully deleted until the correlating load balancer resources are deleted. Any
service that has `type=LoadBalancer` (both existing and newly created ones)
will be attached a service LoadBalancer finalizer, which should be removed by
service controller upon the cleanup of related load balancer resources. Such
finalizer protection mechanism will be released with phases to ensure downgrades
can happen safely.

## Motivation

There are various cases where service controller can leave orphaned load
balancer resources after services are deleted (ref discussion on
https://github.com/kubernetes/kubernetes/issues/32157,
https://github.com/kubernetes/kubernetes/issues/53451). We are periodically
getting bug reports and customer issues that replicated such problem, which
seems to be common enough and is worth to have a better mechanism for ensuring
the cleanup of load balancer resources.

### Goals

Ensure the Service resource is not fully deleted until the correlating load
balancer resources are deleted.

## Proposal

We are going to define a finalizer for service LoadBalancers with name
`service.kubernetes.io/load-balancer-cleanup`. This finalizer will be attached
to any service that has `type=LoadBalancer` if the cluster has the cloud
provider integration enabled. Upon the deletion of such service, the actual
deletion of the resource will be blocked until this finalizer is removed.
This finalizer will not be removed until cleanup of the correlating load
balancer resources are considered finished by service controller.

Note that the removal of this finalizer might also happen when service type
changes from `LoadBalancer` to another. This however doesn't change the
implication that the resources cleanup must be fulfilled before fully deleting
the service.

The lifecyle of a `LoadBalancer` type service with finalizer would look like:
- Creation
  1. User creates a service.
  2. Service controller observes the creation and attaches finalizer to the service.
  3. Provision of load balancer resources.
- Deletion
  1. User issues a deletion for the service.
  2. Service resource deletion is blocked due to the finalizer.
  3. Service controller observed the deletion timestamp is added.
  4. Cleanup of load balancer resources.
  5. Service controller removes finalizer from the service.
  6. Service resource deleted.
- Update to another type
  1. User update service from `type=LoadBalancer` to another.
  2. Service controller observed the update.
  3. Cleanup of load balancer resources.
  4. Service controller removes finalizer from the service.

The expected cluster upgrade/downgrade path for service with finalizer would be:
- Upgrade from pre-finalizer version
  - All existing `LoadBalancer` services will be attached a finalzer upon startup
  of the new version of service controller.
  - The newly created `LoadBalancer` services will have finalizer attached upon
  creation.
- Downgrade from with-finailzer version
  - All existing `LoadBalancer` service will have the attached finalizer removed
  upon the cleanup of load balancer resources.
  - The newly created `LoadBalancer` services will not have finailzer attached.

To ensures that downgrades can happen safely, the first release will include the
"remove finalizer" logic with the "add finalizer" logic behind a gate. Then in a
later release we will remove the feature gate and enable both the "remove" and
"add" logic by default.

As such, we are proposing Alpha/Beta/GA phase for this enhancement as below:
- Alpha: Finalizer cleanup will always be on. Finalizer addition will be off by
default but can be enabled via a feature gate.
- Beta: Finalizer cleanup will always be on. Finalizer addition will be on by
default but can be disabled via a feature gate.
- GA: Service LoadBalancers Finalizer Protection will always be on.

### Risks and Mitigations

#### n+2 upgrade/downgrade is not supported

If user does n+2 upgrade from v1.14 -> v1.16 and then does a downgrade back to v1.14.
They would have added finalizers to the Service but then lose the removal logic on
the downgrade. And hence Service with `type=LoadBalancer` can't be deleted until the
finalizer on it is manually removed.

To keep the upgrade/downgrade safe a user would always do n+1 upgrade/downgrade as
stated on https://kubernetes.io/docs/setup/version-skew-policy/#supported-component-upgrade-order.

### Other notes

If the cloud provider opts-out of [LoadBalancer](https://github.com/kubernetes/cloud-provider/blob/402566916174f020983cb0bd467daeae6206ae02/cloud.go#L48-L49)
support, service controller won't be run at all (see [here](https://github.com/kubernetes/kubernetes/blob/3e52ea8081abc13398de6283c31056cd6aecf6b4/pkg/controller/service/service_controller.go#L229-L232)).
Hence finalizer won't be added/removed by service controller.

If any other custom controller that watches Service with `type=LoadBalancer`, it
should implement its own finalizer protection.

### Test Plan

We will implement e2e test cases to ensure:
- Service finalizer protection works with various service lifecycles on a cluster
that enables it.

In addition to above, we should have upgrade/downgrade tests that:
- Verify the downgrade path and ensure service finalizer removal works.
- Verify the upgrade path and ensure finalizer protection works with existing LB
services. 

### Graduation Criteria

Beta: Allow Alpha ("remove finalizer") to soak for at least one release, then
switch the "add finalizer" logic to be on by default.

GA: Allow Beta to soak for at least one release. (There is no behavioral
differences from the Beta phase.)

## Implementation History

- 2017-10-25 - First attempt of adding finalizer to service
(https://github.com/kubernetes/kubernetes/pull/54569)
- 2018-07-06 - Split finalizer cleanup logic to a separate PR
(https://github.com/kubernetes/kubernetes/pull/65912)
- 2019-04-23 - Creation of the KEP
- 2019-05-23 - PR merged for adding finalizer support in LoadBalancer services (https://github.com/kubernetes/kubernetes/pull/78262)
