# KEP-5708: Standard Namespace Role Label

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: ClusterNetworkPolicy for User Namespaces](#story-1-clusternetworkpolicy-for-user-namespaces)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Label Specification](#label-specification)
  - [System Namespaces](#system-namespaces)
  - [Test Plan](#test-plan)
      - [Unit tests](#unit-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

Introduce a standard label `kubernetes.io/namespace-role` to distinguish "system" namespaces (e.g., `kube-system`, `kube-public`, `default`) from "user" namespaces. This enables distribution-independent policy targeting, particularly for ClusterNetworkPolicy.

## Motivation

Real-world use of ClusterNetworkPolicy requires distinguishing "system" namespaces from "user" namespaces. Administrators need to write policies that apply to user workloads without accidentally breaking cluster infrastructure.

For example, [Securing a Cluster](https://kubernetes.io/docs/tasks/administer-cluster/securing-a-cluster/) documentation suggests restricting access to cloud metadata APIs (169.254.169.254). However, system components like cloud providers, ingress controllers, and CoreDNS may require access. Without a standard label, there is no distribution-independent way to write such policies.

### Goals

- Define a standard label key `kubernetes.io/namespace-role` for namespace classification
- Automatically label system namespaces (`kube-system`, `kube-public`, `default`) with `kubernetes.io/namespace-role=system`
- Enable ClusterNetworkPolicy to target namespaces by role

### Non-Goals

- Define additional role values beyond `system` (future KEP)
- Automatically label user-created namespaces
- Change existing namespace admission or validation

## Proposal

Add the label `kubernetes.io/namespace-role=system` to system namespaces when they are created by the system namespaces controller.

### User Stories

#### Story 1: ClusterNetworkPolicy for User Namespaces

As a cluster administrator, I want to block user namespace pods from accessing cloud metadata APIs while allowing system components to access them.

```yaml
apiVersion: policy.networking.k8s.io/v1alpha2
kind: ClusterNetworkPolicy
metadata:
  name: block-user-namespaces-from-metadata-api
spec:
  tier: Admin
  priority: 100
  subject:
    namespaces:
      matchExpressions:
      - key: kubernetes.io/namespace-role
        operator: NotIn
        values:
        - system
  egress:
  - action: "Deny"
    to:
    - networks:
      - 169.254.169.254/32
```

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Existing namespaces won't have the label | Document that label only applies to newly created system namespaces; provide migration guidance |
| Users may expect automatic labeling of their namespaces | Clearly document that only system namespaces are auto-labeled |

## Design Details

### Label Specification

- **Key**: `kubernetes.io/namespace-role`
- **Value**: `system`

The label uses the `kubernetes.io` prefix as it is a core Kubernetes concept.

### System Namespaces

The following namespaces receive the `system` role label when created:
- `kube-system`
- `kube-public`
- `default`

### Test Plan

##### Unit tests

- `pkg/controlplane/controller/systemnamespaces`: Tests verify system namespaces are created with correct labels

### Graduation Criteria

#### Alpha

- Label applied to system namespaces on creation
- Unit tests passing
- Feature gate `NamespaceRoleLabel` available

#### Beta

- Feedback from ClusterNetworkPolicy users incorporated
- E2E tests added

#### GA

- Two releases of beta stability
- Documentation complete

### Upgrade / Downgrade Strategy

- **Upgrade**: New system namespaces created after upgrade will have the label. Existing namespaces are unchanged.
- **Downgrade**: The label remains on namespaces but has no effect.

### Version Skew Strategy

The label is passive metadata. No coordination required between components.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `NamespaceRoleLabel`
  - Components depending on the feature gate: `kube-apiserver`

###### Does enabling the feature change any default behavior?

Yes. System namespaces created after enablement will have the `kubernetes.io/namespace-role=system` label.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate stops new namespaces from receiving the label. Existing labels remain.

###### What happens if we reenable the feature if it was previously rolled back?

New system namespaces will receive the label. Existing namespaces are unchanged.

###### Are there any tests for feature enablement/disablement?

Unit tests cover label application behavior.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

No impact on running workloads. The label is passive metadata.

###### What specific metrics should inform a rollback?

None. The label has no runtime behavior.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

To be tested manually before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check for namespaces with label `kubernetes.io/namespace-role=system`:
```
kubectl get ns -l kubernetes.io/namespace-role=system
```

###### How can someone using this feature know that it is working for their instance?

Verify system namespaces have the expected label after cluster creation.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A - passive metadata.

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

N/A.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No additional API calls. Label is added during existing namespace creation.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Minimal increase: ~40 bytes per system namespace for the label.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No different from baseline behavior.

###### What are other known failure modes?

None.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A.

## Implementation History

- 2026-03-07: Initial KEP created

## Alternatives

1. **Annotation instead of label**: Labels are preferred as they support `matchExpressions` in selectors.
2. **Namespace-scoped field**: Would require API changes; labels are more flexible and backward-compatible.
3. **Admission webhook**: Adds operational complexity; built-in labeling is simpler.
