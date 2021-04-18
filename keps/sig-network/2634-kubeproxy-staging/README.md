# KEP-2634: Move Kubeproxy package to staging
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Move the internal network dependencies that kube-proxy already has](#move-the-internal-network-dependencies-that-kube-proxy-already-has)
  - [Move other dependencies kube-proxy already has inside k/k](#move-other-dependencies-kube-proxy-already-has-inside-kk)
  - [Move feature gate package dependency](#move-feature-gate-package-dependency)
  - [Move pkg/proxy to staging](#move-pkgproxy-to-staging)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist
- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
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
kube-proxy packages actually are present inside Kubernetes repo, inside 
[pkg/proxy](https://github.com/kubernetes/kubernetes/tree/v1.21.0/pkg/proxy) and 
the idea is to move it its own repo, already existing in 
[kubernetes/kube-proxy](https://github.com/kubernetes/kube-proxy) repo


## Motivation

We are seeing the rise of a lot of kube-proxy implementations. Although each of them 
follow their own dataplane implementation (IPTables, IPVS, eBPF, open flow programming), 
every implementation needs to follow the same logic: watching Services, Endpoints and 
most recently EndpointSlices.

Everytime we change the logic from Services, each of those implementations need to make 
changes into their code base to be compliant with the new features. Some of them vendor
the whole kubernetes/kubernetes repo, while others copy all the code base to their own
repository.

Moving pkg/proxy logics to its own repo will allow a better decoupling of the code, 
the evolution of proxy features without being tighten to kubernetes main code and also 
to future kube-proxy implementations to import this logic without needing to import the 
whole Kubernetes code.

This was also motivated by [kubernetes #92369](https://github.com/kubernetes/kubernetes/issues/92369)

### Goals

* Move kube-proxy repo to `kubernetes/kube-proxy` repo using staging.
* Decouple kube-proxy from internal kubernetes dependencies.

### Non-Goals
* Implementing interfaces for kube-proxy new implementations (like virtual-kubelet, 
but for proxies). This can be addressed on a later KEP

## Proposal

kube-proxy already have its own repo. The proposal is to:

### Move the internal network dependencies that kube-proxy already has
kube-proxy have some internal dependencies, mostly notable in `kubernetes/pkg/utils`. We
can nominate some of them being `iptables`, `ipvs` and `conntrack` packages, but there are
others as well.

We need to move these dependencies to some repo that does not force kube-proxy to also
vendor the whole kubernetes repo.

<<[UNRESOLVED repository for network utils ]>>
Although the majority of utils used by kube-proxy can be moved to component-helpers,
some still vendors external dependencies (like `moby/ipvs`).
If we move every pkg/utils used by kube-proxy to component helpers, another project
already vendoring it (like kubectl) might vendor moby/ipvs because of transitive 
dependencies.
We need to define if network utils should be moved to its own repo (suggesting kubernetes/net-utils)
or if we can accept that other projects vendoring `component-helpers` are going to vendor the 
dependencies of kube-proxy as well
<<[/UNRESOLVED]>>

### Move other dependencies kube-proxy already has inside k/k
kube-proxy vendors some packages that are used by other parts of kubernetes codebase. 
From now, we can point:
* pkg/util/async (async bounder) which is used by pkg/controlplane
* pkg/util/sysctl which is used by pkg/kubelet and kubemark

### Move feature gate package dependency
Because kube-proxy relies on defined Feature gates, it vendors `pkg/features`.

This needs to be copied/moved the same way controller manager already does, moving this to
`kubernetes/kube-proxy/pkg/features` and changing on the code, and putting all the feature
gates referenced by kube-proxy there

### Move pkg/proxy to staging
The final step is to move pkg/proxy to `staging/src/k8s.io/kube-proxy/pkg` and let people willing
to vendor kube-proxy to change from kubernetes/kubernetes to `kubernetes/kube-proxy` repo.

### Risks and Mitigations
"If a project vendors Kubernetes to import kubeproxy code, this will break them. 
On the bright side, afterwards, these importers will have a much cleaner path to 
include kubeproxy code. Before moving forward with this plan, we will identify 
and communicate to these projects." - Shameless copied from 
[kubectl moving](https://github.com/kubernetes/enhancements/tree/master/keps/sig-cli/1020-kubectl-staging)

### Test Plan

All the existing kube-proxy tests will be migrated to point to the new repo location

### Graduation Criteria
As this is a file location move, the graduation criteria will be "every tests still passes"


## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

Not applicable

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: not applicable
  - Components depending on the feature gate:

###### Does enabling the feature change any default behavior?

Not applicable

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Not applicable

###### What happens if we reenable the feature if it was previously rolled back?

Not applicable

###### Are there any tests for feature enablement/disablement?

Not applicable

### Rollout, Upgrade and Rollback Planning

Not applicable

###### How can a rollout fail? Can it impact already running workloads?

Not applicable

###### What specific metrics should inform a rollback?

Not applicable

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not applicable

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

Not applicable

## Drawbacks

* This can cause some noise with people already implementing their own kube-proxy and 
currently happy about vendoring kubernetes/kubernetes repo
* Put some effort while there's already an alternative being developed (see below)

## Alternatives

There's already an effort to evolve kube-proxy, called [kpng](https://github.com/kubernetes-sigs/kpng)
but we believe that the effort of both can run in parallel
