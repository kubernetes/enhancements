# KEP-2067: Rename the kubeadm "master" label and taint

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (optional)](#user-stories-optional)
    - [Administrator](#administrator)
    - [Administrator (single node)](#administrator-single-node)
    - [Developer](#developer)
    - [Developer (higher up the stack)](#developer-higher-up-the-stack)
  - [Notes/Constraints/Caveats (optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Risk 1](#risk-1)
- [Design Details](#design-details)
  - [Renaming the &quot;node-role.kubernetes.io/master&quot; Node label](#renaming-the-node-rolekubernetesiomaster-node-label)
  - [Renaming the &quot;node-role.kubernetes.io/master&quot; Node taint](#renaming-the-node-rolekubernetesiomaster-node-taint)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Removing a Deprecated Flag](#removing-a-deprecated-flag)
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
  - [Hard transition](#hard-transition)
  - [Using a feature gate](#using-a-feature-gate)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [ ] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubeadm applies a "node-role" label to its control plane Nodes.
Currently this label key is `node-role.kubernetes.io/master` and it should
be renamed to `node-role.kubernetes.io/control-plane`. Kubeadm also uses the
same "node-role" as key for a taint it applies on control plane Nodes.
This taint key should also be renamed to "node-role.kubernetes.io/control-plane".

## Motivation

The Kubernetes project is moving away from wording that is considered offensive.
A new working group [WG Naming](https://git.k8s.io/community/wg-naming) was created
to track this work, and the word "master" was declared as offensive.
This means it should be removed from source code, documentation, and user-facing
configuration from all Kubernetes projects.

### Goals

- Use "control-plane" instead of "master" in the key of the label
and taint set by kubeadm.
- Apply proper deprecation policies to minimize friction with users
facing the change.
- Notify users in release notes and on communication channels such as the
kubernetes-dev mailing list.

### Non-Goals

- Hard rename the kubeadm "master" label and taint keys to "control-plane"
within a single release.
- Leave users in the dark about the change.

## Proposal

### User Stories (optional)

#### Administrator

As an administrator of kubeadm-created clusters, I manage additional critical
workloads on Nodes labeled and tainted with "node-role.kubernetes.io/master".
This change affects me and I would like to transition my infrastructure within
a grace period.

#### Administrator (single node)

As an administrator, I deploy single-control-plane Node k8s clusters.
This means that I have to remove taints with the key "node-role.kubernetes.io/master".
This change affects me and I would like to transition my infrastructure within
a grace period.

#### Developer

As an application or addon developer, I create deployments that tolerate the
"node-role.kubernetes.io/master"-keyed taint or label-select the
"node-role.kubernetes.io/master"-labeled Nodes. This change affects me and I would
like to adapt my manifests within a grace period.

#### Developer (higher up the stack)

As a developer of a tool that sits on top of kubeadm in the stack, I have to adapt
my source code to handle the renaming of the "master" label / taint. Managed by my tool,
a version of kubeadm that no longer includes the "master" label / taint would require
special handling.

### Notes/Constraints/Caveats (optional)

None

### Risks and Mitigations

#### Risk 1

Users not having enough visibility about the change, which results in
breaking their setup.

_Mitigation_

Make sure the change is announced on all possible channels:
- Include "action-required" release notes in all appropriate stages
of the change.
- Notify #kubeadm and #sig-cluster-lifecycle channels on k8s slack.
- Notify the SIG Cluster Lifecycle and kubernetes-dev mailing lists.
- Ask Twitter / Reddit users to post about the change.

## Design Details

The process will be broken into multiple stages:
- Primary - 1.20
- Secondary - Minimum deprecation period for GA features is 1 year.
Estimated 1.24, but may depend on user feedback.
- Third - one release after Secondary
- Fourth - one release after Third

### Renaming the "node-role.kubernetes.io/master" Node label

Primary stage:
- Introduce the "node-role.kubernetes.io/control-plane" label in parallel to
the "master" label.
- Announce to users that they should adapt to use the new label.
Secondary stage:
- Remove the "master" label and announce it to the users.

### Renaming the "node-role.kubernetes.io/master" Node taint

Primary stage:
- Introduce the "node-role.kubernetes.io/control-plane:NoSchedule" toleration
in the CoreDNS Deployment of kubeadm.
- Announce to users that they should do that same for their workloads.
Secondary stage:
- Add the "node-role.kubernetes.io/control-plane:NoSchedule" taint to Nodes.
Third stage:
- Remove the "node-role.kubernetes.io/master:NoSchedule" taint from Nodes.
Fourth stage:
- Remove the "node-role.kubernetes.io/master:NoSchedule" toleration in the CoreDNS
Deployment of kubeadm
- Announce to users that they should remove tolerations for the "master" taint in
their workloads.

### Test Plan

Reuse the existing kubeadm upgrade tests (mutable upgrades) and Cluster API
(immutable upgrades) for testing the rollout of the change.

### Graduation Criteria

Not applicable.

#### Removing a Deprecated Flag

Not applicable.

### Upgrade / Downgrade Strategy

Downgrades are not supported by kubeadm.

For upgrades:
- During the primary stage new Nodes in the cluster will be added with
the "node-role.kubernetes.io/control-plane" label in parallel to the
"node-role.kubernetes.io/master" label.
The "node-role.kubernetes.io/control-plane:NoSchedule" toleration will
be added to the kubeadm CoreDNS Deployment of kubeadm so that it
tolerates both old and new nodes.
- During the secondary stage the "master" label will be removed from new
Nodes. User infrastructure must only manage the "control-plane" label
at that point. New nodes will also have the
"node-role.kubernetes.io/control-plane:NoSchedule" taint.
- During the third stage, new Nodes will only have the
"node-role.kubernetes.io/control-plane:NoSchedule" taint and the
"node-role.kubernetes.io/master:NoSchedule" taint will not be present.
- During the fourth stage the CoreDNS deployment of kubeadm will
have its toleration for the "node-role.kubernetes.io/master:NoSchedule"
taint removed.

### Version Skew Strategy

See above.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**

The new label / taint will be enabled in parallel to the old ones.
After a GA period the old label / taint will be removed.

* **Does enabling the feature change any default behavior?**

Removing the old label / taint can introduce changes in behavior.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

Users can opt-out of the new default label / taint that kubeadm places
on control-plane Nodes, and use the old "master" label / taint.

* **What happens if we reenable the feature if it was previously rolled back?**

There should be no problems around that, as long as the workloads
tolerate the right label / taints.

* **Are there any tests for feature enablement/disablement?**

The standard kubeadm upgrade tests can catch problems around the feature enablement.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**

The change can affect execution of workloads. Workloads must be adapted
to tolerate the eventual removal of the "master" label / taint and the
introduction of the "control-plane" label / taint. If a version of kubeadm
that upgrades a cluster to a new version that no longer has the old label / taint,
this would require patching the workload manifests.

* **What specific metrics should inform a rollback?**

Not applicable.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

This is not a supported scenario, as kubeadm does not support downgrade. However,
it can restore control-plane static Pod manifests in case the upgrade operation failed.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

The final stage of the change will apply the removal of the old label / taint,

### Monitoring Requirements

Not applicable.

### Dependencies

Not applicable.

### Scalability

Not applicable.

### Troubleshooting

Not applicable.

## Implementation History

- 2020-10-03 - initial provisional KEP created
- 2020-10-05 - KEP marked as implementable

## Drawbacks

None.

## Alternatives

### Hard transition

An alternative is to hard transition to the new key for the label and taint,
within one version,  which was already mentioned as a non-goal in this KEP.

### Using a feature gate

Using a feature gate was discussed, where:
- During ALPHA ("false" by default) the option to use the new label / taint
will be added, where users can switch to use the new keys for the label / taint.
- During BETA ("true" by default) the new label / taint will be used by default.
- For GA, the FG will be locked to true and deprecated. Then removed after
one more release.

The problem with this is that during BETA, the feature becoming "true" by default
will break the users in a similar fashion to a hard-transition. The feature gate
usage here would not introduce benefits over the main proposal, but would
introduce some maintenance overhead.

## Infrastructure Needed (Optional)

None.
