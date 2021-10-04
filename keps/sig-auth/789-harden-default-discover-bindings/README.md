# Harden Default RBAC Discovery ClusterRole(Binding)s

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Existing customization of <code>system:discovery</code>](#existing-customization-of-systemdiscovery)
    - [Dependence on existing unauthenticated behavior](#dependence-on-existing-unauthenticated-behavior)
- [Graduation Criteria](#graduation-criteria)
  - [Testing](#testing)
  - [Documentation](#documentation)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

The aim of this change is to remove the `system:unauthenticated` subjects from the `system:discovery` and `system:basic-user` ClusterRoleBindings, while preserving unauthenticated access to a genuinely non-sensitive subset of the current APIs (e.g. `GET /healthz`, `GET /version`). This will improve the default privacy and security posture for new clusters.

## Motivation

One of the work items resulting from the [CVE-2018-1002105](https://github.com/kubernetes/kubernetes/issues/71411) post-mortem was to [investigate hardening the default RBAC discovery ClusterRoleBindings](https://github.com/kubernetes/kubernetes/issues/72115) (i.e. `system:discovery` and `system:basic-user`) to limit potential avenues for similar attack. Additionally, the fact that API extensions are exposed by the default discovery bindings is surprising to some and represents a potential privacy concern (e.g. `GET /apis/self-flying-cars.unicorn.vc/v1/`).

### Goals

* Remove discovery from the set of APIs which allow for unauthenticated access by default, improving privacy for CRDs and the default security posture of default clusters in general.

### Non-Goals

* To protect default clusters from unauthenticated access entirely (already achievable via `--anonymous-auth=false`), or to add support for more granular schema discovery.
  * There are several non-sensitive and legitimate use-cases for unauthenticated calls, such as `/healthz` liveliness checks and [returning useful information about the cluster](https://github.com/kubernetes/kubernetes/issues/45366#issuecomment-299275002) to `kubectl version`, that we don't wish to break.
* To prevent _namespace_ admins from granting namespace-level permissions to anonymous users.
* To protect the CRD names and details from _authenticated_ users.
* To address other, more fundamental, discovery-related bugs uncovered in [CVE-2018-1002105](https://github.com/kubernetes/kubernetes/issues/71411), e.g.:
  * https://github.com/kubernetes/kubernetes/issues/72113
  * https://github.com/kubernetes/kubernetes/issues/72117

## Proposal

* Remove the `system:unauthenticated` subject group from the default [`system:discovery` and `system:basic-user` ClusterRoleBindings](https://github.com/kubernetes/kubernetes/blob/release-1.13/plugin/pkg/auth/authorizer/rbac/bootstrappolicy/policy.go#L531-L532)
* Create a new default ClusterRole which prominently, explicitly documents its intent to freely disclose information (e.g. `system:public-info-viewer`) and contains the `GET /healthz`, `/version` and `/version/` PolicyRules.
* Create a new default ClusterRoleBinding for the new ClusterRole, which grants the `system:(un)authenticated` subject groups access.

### Risks and Mitigations

#### Existing customization of `system:discovery`

If a user upgrades a cluster which has modified the `system:discovery` ClusterRoleBinding, these changes could either be trampled or restricted endpoints could end up being re-exposed via the new `system:public-info-viewer` binding.

Mitigation: During the API server's RBAC auto-reconciliation, if `system:discovery` exists and `system:public-info-viewer` does not (i.e. the state after an upgrade) we'll copy `system:discovery`'s subjects and reconciliation annotation value. This will preserve permissions as-is during API server upgrades, even with the addition of `system:public-info-viewer`, since existing customization will be preserved.

#### Dependence on existing unauthenticated behavior

Some use-cases might require the existing permissions to be preserved for unauthenticated calls, and some currently-working configurations might be broken for new installs. 

Disambiguating between accidental and necessary dependence on the current behavior will have to be determined by the user on a case-by-case basis. However, in the release notes, we can include easy 'escape hatches' to re-enable unauthenticated access to the discovery APIs, such as the ones recommended to cluster admins below.

From [`system:basic-user`](https://github.com/kubernetes/kubernetes/blob/8b98e802eddb9f478ff7d991a2f72f60c165388a/plugin/pkg/auth/authorizer/rbac/bootstrappolicy/policy.go#L209-L215):
* create `selfsubjectaccessreviews`
* create `selfsubjectrulesreviews`

pro: unauthenticated users can't list actions available to them
con: this doesn't actually forbid actions, and can create a potentially confusing mismatch for users which rely on these APIs as a sanity check

From [`system:discovery`](https://github.com/kubernetes/kubernetes/blob/8b98e802eddb9f478ff7d991a2f72f60c165388a/plugin/pkg/auth/authorizer/rbac/bootstrappolicy/policy.go#L198-L208):
* get `/api`, `/api/*`
* get `/apis`, `/apis/*`
* get `/openapi`, `/openapi/*`

pro: can't see potentially sensitive custom resource names/schemas by default
con: consumers that need schema info, such as editors and users of namespaced resources w/ default tools (e.g. `kubectl get pods -n foo`) must allow authentication to the discovery APIs to fetch it (though, this is already the case with clusters which have disabled anonymous auth). More concretely, **cluster admins that wish anonymous users to have API access should grant these permissions as part of cluster setup**, for example:
```
kubectl create clusterrolebinding anonymous-discovery --clusterrole=system:discovery --group=system:unauthenticated
kubectl create clusterrolebinding anonymous-access-review --clusterrole=system:basic-user --group=system:unauthenticated
```
A potential future feature could automatically grant discovery permissions to anonymous users in the event that they're granted access to another API.

## Graduation Criteria

This proposal will have 'graduated' once the unauthenticated API surface has been minimized without excessive user impact. Excessive user impact includes issues that can't be mitigated with a single `kubectl` invocation or fixed by enabling request authentication.

### Testing 

To address the addition of `system:public-info-viewer` and the modification of `system:discovery` and `system:basic-user`:
* update testdata for [ClusterRoles](https://github.com/kubernetes/kubernetes/blob/master/plugin/pkg/auth/authorizer/rbac/bootstrappolicy/testdata/cluster-roles.yaml) and [ClusterRoleBindings](https://github.com/kubernetes/kubernetes/blob/master/plugin/pkg/auth/authorizer/rbac/bootstrappolicy/testdata/cluster-role-bindings.yaml)
* add/modify cases in [reconcile_role_test.go](https://github.com/kubernetes/kubernetes/blob/master/pkg/registry/rbac/reconciliation/reconcile_role_test.go) and [reconcile_rolebindings_test.go](https://github.com/kubernetes/kubernetes/blob/master/pkg/registry/rbac/reconciliation/reconcile_rolebindings_test.go)

### Documentation

Documentation regarding the [default RBAC discovery ClusterRole(Bindings)](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#discovery-roles) will have to be updated to reflect the new realities.

## Implementation History

2019-01-31: KEP submitted
2019-02-06: [implementation PR opened](https://github.com/kubernetes/kubernetes/pull/73807)
2019-02-27: [documentation PR opened](https://github.com/kubernetes/website/pull/12888)
2019-03-01: implementation PR merged
2019-03-11: documentation PR closed
