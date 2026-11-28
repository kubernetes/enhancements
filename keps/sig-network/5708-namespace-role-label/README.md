# KEP-5708: Standard Namespace Role Label

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: ClusterNetworkPolicy for user namespaces](#story-1-clusternetworkpolicy-for-user-namespaces)
    - [Story 2: A controller that skips infrastructure namespaces](#story-2-a-controller-that-skips-infrastructure-namespaces)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Label Specification](#label-specification)
  - [Reconciliation](#reconciliation)
  - [Namespaces Kubernetes Does Not Label](#namespaces-kubernetes-does-not-label)
  - [Misuse of the Reserved Label](#misuse-of-the-reserved-label)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

Reserve the label key `kubernetes.io/namespace-role` and define one value, `system`, meaning *this namespace is not intended to hold user workloads*. kube-apiserver reconciles that label onto `kube-system`, `kube-public` and `kube-node-lease`; distributions, operators and administrators may apply the same label to their own infrastructure namespaces. Consumers can then select on it portably.

This KEP does not settle what "a system namespace" means in general — there is no single accepted definition today. It defines the meaning of one label value and the namespaces for which kube-apiserver guarantees it.

## Motivation

Cluster-scoped policy needs to distinguish namespaces holding user workloads from namespaces holding the cluster's own components. Today that is done by hardcoding namespace names, which is not portable: infrastructure lives in different namespaces on different distributions, so a policy written against `kube-system` is wrong on a cluster whose DNS is in `openshift-dns`.

For example, [Securing a Cluster](https://kubernetes.io/docs/tasks/administer-cluster/securing-a-cluster/#restricting-cloud-metadata-api-access) recommends restricting access to the cloud metadata API (169.254.169.254), but cloud controllers, ingress controllers and CoreDNS may need it. The same need appears outside networking: workload-management controllers want to skip infrastructure namespaces, and each one invents its own label for the purpose.

### Goals

- Define the reserved key `kubernetes.io/namespace-role` and the value `system`, with a meaning that is not networking-specific.
- Have kube-apiserver **reconcile** the label onto the namespaces it owns, so it is present on existing clusters after upgrade and restored if removed.
- Document that other parties may apply the same label to their own infrastructure namespaces.

### Non-Goals

- Defining further values (for example `addon`). See [Notes/Constraints/Caveats](#notesconstraintscaveats).
- Labelling namespaces kube-apiserver does not own; Kubernetes cannot know which of them are infrastructure.
- Labelling user namespaces as `user`. Absence of the label is not an assertion.
- Using the label as an authorization boundary. It is metadata for policy to select on, not a permission check.

## Proposal

Extend the system namespaces controller in kube-apiserver so that on each sync it ensures the namespaces it owns carry `kubernetes.io/namespace-role=system`.

Reconciling rather than labelling at creation is what makes the label usable as a selector: `kube-system`, `kube-public` and `kube-node-lease` are created once at bootstrap and never recreated, so a create-time-only implementation would never label an existing cluster — never labelling the clusters that need the policy.

### User Stories

#### Story 1: ClusterNetworkPolicy for user namespaces

As a cluster administrator, I want to block user-namespace pods from reaching the cloud metadata API while allowing system components to reach it. Illustrative of one consumer, **not normative**; this KEP does not depend on that API.

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

#### Story 2: A controller that skips infrastructure namespaces

As the author of a workload-management controller, I want to act only on user namespaces. Today I define my own label and ask every administrator to apply it, so each installation is configured differently. A standard label lets me skip `kubernetes.io/namespace-role=system` and be portable without configuration.

### Notes/Constraints/Caveats

**Is a two-way split the right one?** A finer taxonomy has been suggested — core components, add-ons, user workloads. This KEP starts with one value because "core" versus "add-on" varies by distribution (CoreDNS may be either), so the distinction would re-create the non-portability, and because both break the cluster if restricted, so one bucket avoids multi-exclusion selectors. A keyed value leaves room to add values later without reserving a second key. **This remains open for SIG Network and SIG Architecture and should be settled before `implementable`.**

**Function, not origin.** A DNS namespace is infrastructure whether it is called `kube-system` or something distribution-specific. Kubernetes cannot determine that, so it guarantees the label only on namespaces it creates; see [Namespaces Kubernetes Does Not Label](#namespaces-kubernetes-does-not-label).

**ClusterNetworkPolicy maturity.** A concern has been raised about building a convention around an API that is out-of-tree and pre-GA. Nothing here depends on that API: the key, the value and the reconciliation stand alone, Story 2 is a non-networking consumer, and Story 1 is non-normative. This KEP can remain `provisional` until that API stabilises without the design changing.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| A `NotIn [system]` policy is written while infrastructure namespaces Kubernetes does not own are still unlabelled, so the policy applies to them | Document that only the three labelled namespaces are guaranteed, and that others must be labelled first. `kubectl get ns -L kubernetes.io/namespace-role` audits this. Still better than the status quo, where the exclusion list lives in each policy rather than on the namespace |
| Absence of the label is read as "user namespace" | Document that the label is a positive assertion only; consumers needing a closed-world answer combine it with their own selectors |
| The label is applied to a namespace that is not infrastructure | Default RBAC limits namespace label changes to cluster administrators, so this is admin error, not escalation. Beta adds an admission warning; see [Misuse of the Reserved Label](#misuse-of-the-reserved-label) |
| Reconciliation fights an administrator who removes the label deliberately | Scoped to three namespaces where the assertion is unambiguous. Disable the feature gate to opt out |

## Design Details

### Label Specification

- **Key**: `kubernetes.io/namespace-role` — the `kubernetes.io` prefix reserves it, so the meaning is set by Kubernetes rather than per-cluster.
- **Value**: `system` — *this namespace is not intended to hold user workloads*. Deliberately not networking-specific.

The system namespaces controller already manages four namespaces in kube-apiserver. `Options.SystemNamespaces` defaults to `kube-system`, `kube-public` and `default` ([`pkg/controlplane/apiserver/options/options.go`][opts]), and kube-apiserver appends `kube-node-lease` ([`cmd/kube-apiserver/app/options/options.go`][cmd]). There is no CLI flag for the list, so it is not administrator-configurable; an administrator marks additional namespaces by applying the label.

This KEP labels **three** of those four: `kube-system`, `kube-public` and `kube-node-lease`. Their contents are not user workloads on any conformant cluster.

`default` is managed by the same controller but is deliberately **not** labelled. It is a special namespace rather than a system namespace: its purpose is to hold user workloads when none is specified, the opposite of what `system` asserts. Labelling it would make a `NotIn [system]` policy silently skip the namespace where unqualified user workloads land — the most dangerous gap such a policy can have. Whether it deserves a distinct value is tied to the taxonomy question; leaving it unlabelled is the conservative choice.

[opts]: https://github.com/kubernetes/kubernetes/blob/master/pkg/controlplane/apiserver/options/options.go
[cmd]: https://github.com/kubernetes/kubernetes/blob/master/cmd/kube-apiserver/app/options/options.go

### Reconciliation

The controller in `pkg/controlplane/controller/systemnamespaces` already runs `sync` every minute (`interval` is hardcoded to `1 * time.Minute`, driven by `wait.Until`), reading namespaces from an informer lister. Today its per-namespace helper `createNamespaceIfNeeded` returns immediately when the namespace exists, so it never touches an existing namespace.

That helper is extended to ensure the label after the existence check:

- Label **absent** — patch it on.
- Label **present and `system`** — no write, so no patching every minute.
- Label **present with another value** — overwrite and log; for these three the value is not a matter of opinion.
- The patch is limited to this key; other labels and annotations are untouched.
- `default` is skipped, so the labelled set is not simply `SystemNamespaces`.

Reconciliation means the label survives accidental removal — restored within about a minute — which a selector in an Admin-tier deny policy depends on.

There is precedent for a reserved `kubernetes.io/` label on namespaces being maintained by the apiserver rather than the user: `pkg/registry/core/namespace/strategy.go` force-sets `kubernetes.io/metadata.name` on every namespace create and update. See [Alternatives](#alternatives) for why that hook alone is not sufficient here.

### Namespaces Kubernetes Does Not Label

Kubernetes does not label, and does not try to infer, anything outside the three above. The expectation this KEP sets is that distributions, operators and administrators apply `kubernetes.io/namespace-role=system` to their own infrastructure namespaces, documented in kubernetes/website alongside the key.

This is a real limitation: on a cluster with many infrastructure namespaces, a `NotIn [system]` selector matches them until they are labelled. The value added is one agreed key instead of a different convention per tool.

### Misuse of the Reserved Label

Default RBAC already limits namespace label changes to cluster administrators: the namespace-scoped `admin` and `edit` roles are granted only read verbs on `namespaces` in `plugin/pkg/auth/authorizer/rbac/bootstrappolicy/policy.go`. A namespace user therefore cannot label their own namespace to escape policy, and the remaining case is administrator error.

At **beta**, a **warning** — not a rejection — is returned when the label is set on a namespace kube-apiserver does not manage. `namespaceStrategy` already has the hooks for this: `WarningsOnCreate` and `WarningsOnUpdate` in `pkg/registry/core/namespace/strategy.go`, the latter currently returning `nil`. Warning is the right strength: labelling a genuine distribution infrastructure namespace is the intended use, so the apiserver cannot tell correct use from mistake. Not in alpha, because the wording depends on how the convention is taken up and getting it wrong is noisy for every legitimate user.

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

Existing `systemnamespaces` tests assert the namespaces are created; they need to tolerate label patches on sync and be reviewed for assumptions that the controller writes nothing once the namespaces exist.

##### Unit tests

Coverage to be filled in from https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit before targeting a milestone.

- `k8s.io/kubernetes/pkg/controlplane/controller/systemnamespaces`: `TBD` - `TBD`

Cases: label added when absent; no write when already `system`; corrected when set to another value; other labels and annotations preserved; `default` not labelled; no write when the gate is disabled.

##### Integration tests

In `test/integration/controlplane`, covering what unit tests cannot — that the label appears on a real apiserver. Cases: the three namespaces labelled after startup; namespaces that already exist without the label get it (the upgrade path); a label deleted by a client is restored; gate disabled writes nothing and removes nothing; disable then re-enable re-adds. Names and permalinks when targeting a milestone.

##### e2e tests

None for alpha: the behaviour is control-plane-internal and integration tests can control the gate. For beta, an e2e test for label presence and for the admission warning, since the warning is user-visible.

### Graduation Criteria

#### Alpha

- Key and `system` value documented, including that absence is not an assertion.
- Reconciliation implemented for the three labelled namespaces, skipping `default`.
- Feature gate `NamespaceRoleLabel`, defaulting to off.
- Unit and integration tests as above, including gate-disabled cases.

#### Beta

- Taxonomy question resolved — whether `system` remains the only value.
- Admission warning for the label on namespaces kube-apiserver does not own.
- Guidance in kubernetes/website for distributions and operators, including that unlabelled infrastructure namespaces must be labelled before exclusion policies.
- Feedback from at least one consumer using the label as a selector.
- e2e test for label presence and the admission warning.
- Metric for reconciliation failures.

#### GA

- Two releases at beta with no change to the label's meaning.
- At least one distribution outside kubernetes/kubernetes labelling its own infrastructure namespaces, showing the convention is portable in practice.
- Documentation complete.

### Upgrade / Downgrade Strategy

- **Upgrade**: the first sync after the gate is enabled labels the three owned namespaces, new cluster or pre-existing. No administrator action, no migration.
- **Downgrade**: reconciliation stops; labels already written remain, since removing them could break a policy relying on them at that moment. Remove deliberately with `kubectl label ns <name> kubernetes.io/namespace-role-`; with the gate off they are not restored. With the gate on, such a removal is undone within about a minute.

### Version Skew Strategy

The label carries no runtime behaviour, so no coordination is required. In an HA control plane with the gate enabled on some apiservers, those write the label and the others do not remove it, so the outcome matches having it enabled everywhere. Reconciliation is idempotent, so apiservers racing to add it converge.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `NamespaceRoleLabel`
  - Components depending on the feature gate: `kube-apiserver`

###### Does enabling the feature change any default behavior?

Yes: `kube-system`, `kube-public` and `kube-node-lease` gain the label. Nothing in Kubernetes acts on it, so the change is observable but has no behavioural effect on its own. A cluster with policy already selecting on the key would see it start to match those namespaces.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes — disabling the gate stops reconciliation. Labels already written are not removed; see [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).

###### What happens if we reenable the feature if it was previously rolled back?

The next sync re-adds the label where missing. Reconciliation is idempotent, so enable/disable cycles converge on the same state.

###### Are there any tests for feature enablement/disablement?

Yes: unit tests for the gate-disabled path, integration tests with the gate on and off, including namespaces that already exist unlabelled and disable-then-reenable.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Running workloads are unaffected; nothing in Kubernetes reads the label. The realistic failure is consumer-side: a policy selecting on the key before the labels exist changes behaviour when they appear. Mitigation is ordering — apply the labels, audit with `kubectl get ns -L kubernetes.io/namespace-role`, then write the policy.

Reconciliation itself can fail if the patch is rejected or storage is unreachable. The label is then absent, which is the pre-feature state, and a `NotIn [system]` policy would apply to those namespaces — hence the beta metric.

###### What specific metrics should inform a rollback?

A non-zero rate of label reconciliation failures (metric added at beta). Absent that, `kubectl get ns -L kubernetes.io/namespace-role` showing the label missing while the gate is enabled.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet. The "namespaces already exist unlabelled" integration test is the upgrade path and disable-then-reenable covers upgrade->downgrade->upgrade; both before alpha ships.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

`kubectl get ns -L kubernetes.io/namespace-role`. In the stronger sense of a consumer selecting on the key, by inspecting the policies on the cluster.

###### How can someone using this feature know that it is working for their instance?

The three labelled namespaces carry the label, and it reappears within about a minute if removed:

```
kubectl label ns kube-public kubernetes.io/namespace-role-
kubectl get ns kube-public -L kubernetes.io/namespace-role
```

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Other (treat as last resort)
  - Details: presence of the label on the three namespaces; at beta, a reconciliation-error counter.

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

The label is present on all three labelled namespaces whenever the gate is enabled, and is restored within about a minute of removal.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Yes — a counter of failed label reconciliations, so a cluster where the label is silently absent is detectable rather than discovered when a policy misbehaves. Beta graduation criterion.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No — only the system namespaces controller, part of kube-apiserver. Not ClusterNetworkPolicy or any other consumer; see [Notes/Constraints/Caveats](#notesconstraintscaveats).

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes, bounded. The controller already reads these namespaces from its cache; it issues a PATCH only on mismatch. Steady state: zero writes. On enablement or upgrade: at most three PATCH calls, once. Administrator removes a label: one PATCH.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Three Namespace objects grow by 40 bytes plus serialization overhead. No change in object count.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No — a label comparison on three cached objects per sync, writing only on mismatch.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The controller runs inside kube-apiserver, so the feature is not running if the apiserver is down. If etcd is unavailable the patch fails and retries next sync; the intermediate state is the label absent, matching pre-feature behaviour.

###### What are other known failure modes?

- **Label missing from one of the three while the gate is enabled.** Detect with the audit command, or the beta metric; diagnose from apiserver logs for patch failures; apply by hand, which is idempotent with the controller.
- **An unlabelled infrastructure namespace is caught by a policy.** Not a bug — the limitation in [Namespaces Kubernetes Does Not Label](#namespaces-kubernetes-does-not-label). Detect by comparing the audit output against where the cluster actually runs infrastructure; label them.
- **The label set on a namespace that does hold user workloads**, so policy skips it. Detect with the audit command; remove the label. The beta admission warning is meant to catch this as it happens.

###### What steps should be taken if SLOs are not being met to determine the problem?

Check kube-apiserver logs for namespace label patch failures and confirm the gate is enabled on all apiservers.

## Implementation History

- 2026-03-07: Initial KEP created.
- 2026-11-28: Revised following review — reconcile instead of labelling on create; `default` excluded from labelling and `kube-node-lease` added; dropped the general definition of "system namespace"; removed the non-goal excluding admission changes and added a beta warning; recorded the taxonomy question as open; decoupled from ClusterNetworkPolicy. Design details checked against kubernetes/kubernetes at `ce9710e5109`.

## Drawbacks

The label is only as complete as the parties who apply it. Kubernetes can guarantee three namespaces; the rest depends on distributions and administrators adopting the convention, so until adoption is broad a policy author cannot treat "no `system` label" as "safe to restrict". That is weaker than the motivating use case ideally wants, and is the main argument against doing this at all; the alternative of Kubernetes guessing which namespaces are infrastructure is worse. Reserving a `kubernetes.io/` key also fixes its meaning more or less permanently.

## Alternatives

1. **Annotation instead of a label**: annotations cannot be used in `matchExpressions`, and selecting namespaces is the point.
2. **A field on Namespace**: self-documenting and validatable, but needs an API change and a new validation surface, is less freely applied by third-party components, and cannot be selected on by existing label-selector machinery.
3. **Admission webhook or distro-provided labels**: the status quo — each tool defines its own label and the administrator wires it up. Works, but is exactly the non-portability this KEP removes, and adds a component to every cluster.
4. **Reuse `kubernetes.io/metadata.name`**: name-matching with extra steps. It is the hardcoding this KEP exists to replace and is no more portable than writing the namespace name into the policy.
5. **Set the label in the registry strategy, as `metadata.name` is.** `namespaceStrategy.Canonicalize` stamps `kubernetes.io/metadata.name` on every create and update, with no controller, no patch and no delay. Rejected as the sole mechanism because it only fires when a namespace is written, and `kube-system`, `kube-public` and `kube-node-lease` are written once at bootstrap and effectively never again — so on an existing cluster the label would never appear, which is the same flaw as labelling at creation time. The controller is what backfills. Doing both is possible but adds a second code path for no gain over a one-minute convergence.
6. **A boolean key such as `kubernetes.io/system-namespace=true`**: a keyed value allows further values later without reserving another key, and a boolean named for "system" pre-commits to the two-way split that is still open.

## Infrastructure Needed (Optional)

None.
