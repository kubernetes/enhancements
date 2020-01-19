---
title: graduate-resourcequotascopeselectors-to-stable
authors:
  - "@ravisantoshgudimetla"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-scheduling
  - sig-apimachinery
reviewers:
  - "@bsalamat"
  - "@k82cn"
  - "@derekwaynecarr"
  - “@sjenning”
  - "@vikaschoudhary16"
approvers:
  - "@bsalamat"
  - "@derekwaynecarr"
editor: TBD
creation-date: 2019-04-23
last-updated: 2019-04-23
status: implementable
see-also:
replaces:
superseded-by:
---

# Graduate ResourceQuotaScopeSelector to stable

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Notes](#implementation-notes)
  - [Constraints](#constraints)
  - [Test Plan](#test-plan)
    - [Existing Tests](#existing-tests)
    - [Needed Tests](#needed-tests)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

[ResourceQuotaScopeSelectors](https://kubernetes.io/docs/concepts/policy/resource-quotas/#resource-quota-per-priorityclass) has been created in the past to expand scopes to represent priorityClass names and their corresponding behaviour. This helps in removing the restriction of allowing critical pods to be created in `kube-system` namespace.

## Motivation

[Priority and Preemption](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/) has been GA'ed in 1.14 but with a caveat that critical pods could be created only in `kube-system` namespace. We wish to graduate `ResourceQuotaScopeSelectors` feature in order to overcome this limitation.

### Goals

* Plan to promote ResourceQuotaScopeSelectors to stable version.
* Remove the limitation of creating critical pods only in `kube-system` namespace.

### Non-Goals

* Changing API field or meaning

## Proposal

### Implementation Notes

In the current implementation: 

1. Priority admission plugin is blocking [creation of critical pods](https://github.com/kubernetes/kubernetes/blob/90fbbee12950f336db2da94dda7beb87846f94e0/plugin/pkg/admission/priority/admission.go#L150) in namespaces other than `kube-system`.

2. We don't have a default quota at bootstrap phase with scope selector to restrict critical pods to be created in `kube-system`.

The current implementation can be changed to relax the restriction of creating critical pod within `kube-system` namespace and let this restriction be created as a default quota at cluster bootstrap phase automatically.

This ensures:

1. We are backwards compatible.
2. System is not being abused where any regular user can create a critical pod in namespace of his/her own choice with those pods having capability to  displace control-plane or other critical pods.
2. Cluster-admin can create quota in whatever the namespace he/she wants instead of limiting critical pods creation to `kube-system` namespace. The default quota with scope selectors is used in `kube-system` namespace.

### Constraints

We should verify the automatic creation of quota and see if it causes any problems with quotas created in other namespaces. 

### Test Plan

#### Existing Tests
- [Run or Not](https://github.com/kubernetes/kubernetes/blob/90fbbee12950f336db2da94dda7beb87846f94e0/test/e2e/apimachinery/resource_quota.go#L799) tests the resource quota under different scenarios to check if the creation/deletion of resource quota with scope selectors is working or not.

#### Needed Tests

- Conformance tests need to be added for default quota with ResourceQuotaScopeSelectors.

### Graduation Criteria

- [ ] Remove limitation of critical pod creation in `kube-system` namespace in pod priority admission plugin
- [ ] Create a `AdmissionConfiguration` object with `limitedResources` to prevent creation of system critical pods in all namespaces
- [ ] Add a default quota with scope selector to allow critical pods to be created in `kube-system` namespace only
- [ ] Graduate ResourceQuotaScopeSelectors API to GA
- [ ] Needs a conformance test
- [ ] Update documents to reflect the changes

## Implementation History

- ResourceQuotaScopeSelectors was introduced as alpha in kubernetes 1.11
- In Kuberenetes 1.12 this feature was promoted to Beta
