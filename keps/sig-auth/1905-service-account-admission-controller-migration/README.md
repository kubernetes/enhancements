# ServiceAccount Admission Controller Migration

## Table of Contents

<!-- toc -->

- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Example Walkthrough](#example-walkthrough)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha-&gt;Beta](#alpha-beta)
    - [Beta-&gt;GA](#beta-ga)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Scalability](#scalability)
- [Alternatives](#alternatives)
- [Implementation History](#implementation-history)

<!-- /toc -->

## Summary

This proposal presents a migration plan for transitioning ServiceAccount
admission controller to provisioning service account token as projected volume
using `TokenRequest` API.

## Motivation

When `AutomountServiceAccountToken` is turned on (default to true),
ServiceAccount admission controller will provision the token of the service
account accosicated with the pod as a secret volume automatically. This allows
in-cluster workloads to talk to the API server. However, the secret tokens are
not favored for the following reasons:

1.  They are long-lived and not rotated regularly and automatically, which makes
    them vulnerable to stealth exfiltration.
2.  They are persistent in etcd. Anyone who has access to etcd can potentially
    hijack the cluster.
3.  Manual rotation of them can be very complicated while keeping applications
    happy.
4.  For every service account in a pod, a secret of type
    `SecretTypeServiceAccountToken` will be created which contains a CA cert
    (usually large) as well as a token. This will cause scalability issue when a
    super cluster which has tons of service accounts. See
    [Issue#48408](https://github.com/kubernetes/kubernetes/issues/48408).
5.  The file permission for the token is `0644`, it is not ideal for highly
    privileged credentials being world readable.

### Goals

- Define safe migration instructions for cluster operators to switch to better
  service account tokens

### Non-Goals

- Deprecate legacy secret-backed tokens
- Deprecate token controller

## Proposal

We will enable feature `BoundServiceAccountVolume` by default to switch
`ServiceAccount` admission controller to provision service account tokens as
projected volumes for pods. Combining `TokenRequest` API with projected volume
gives us the following strenths:

1.  Tokens are short-lived and rotated regularly in an automatic manner.
2.  Tokens are stored as a special source of projected volume that will not
    persist in etcd like Secrets.
3.  Tokens are bound to the pods, which makes an urgent revocation easy by
    redeploying the workloads.
4.  The new volume type enables us to turn off `Token` controller to fix the
    scalability issue caused by large number of "super-sized" secrets.
5.  File permission on the the projected token volume is opportunistically
    restricted. See
    [KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-storage/20180515-svcacct-token-volumes.md#file-permission).
6.  Tokens are also audience bound, which reduces the risk of masquerade.

Before migration to a version with `BoundServiceAccountVolume=true`, cluster
operators should make sure:

1.  Set feature gate `TokenRequest=true`. (default to `true` since 1.12)

    - This feature requires the following flags to the API server:
      - `--service-account-issuer`
      - `--service-account-signing-key-file`
      - `--service-account-key-file`
      - `--api-audiences` (default to `--service-account-issuer`)

2.  Set feature gate `TokenRequestProjection=true`. (default to `true` since
    1.12)

3.  Update all workloads to newer version of officially supported Kubernetes
    client libraries to reload token:

    - Go: >= v11.0.0
    - Python: >= v12.0.0
    - Java: >= v9.0.0
    - Javascript: >= v0.10.3
    - Ruby: master branch
    - Haskell: master branch

    For community-maintained client libraries, feel free to contribute to them
    if the reloading logic is missing.

    **Note**: If having trouble in finding places using in-cluster config
    completely, cluster operators can specify flag
    `--service-account-extend-token-expiration` to kube apiserver to allow
    tokens have longer expiration temporarily during the migration. Any usage of
    legacy token will be recorded in both metrics and audit logs. After fixing
    all the potentially broken worklaods, don't forget to remove the flag so
    that the original expiration settings are honored.

    - Metrics: subsystem: serviceaccount, name: stale_tokens_total
    - Audit: looking for `authentication.k8s.io/stale-token` annotation More
      detail
      [here](https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/20190806-serviceaccount-tokens.md#safe-adoption)

If anything goes wrong, please file a bug and CC @kubernetes/sig-auth-bugs. More
contact information
[here](https://github.com/kubernetes/community/tree/master/sig-auth#contact).

### Notes/Constraints/Caveats

If manually turn on alpha `BoundServiceAccountVolume` in versions &lt; 1.19,
some issues might happen:

1.  PodSecurityPolicies that allow secrets but not projected volumes will
    handicap the provision of token volumes.
2.  Pods running as non root might not have permission to read the token.

### Test Plan

N/A

### Graduation Criteria

#### Alpha->Beta

All known migration frictions have been fixed.

#### Beta->GA

New `ServiceAccount` admission controller WAI in Beta for >= 1 minor without
significant issues.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

- **How can this feature be enabled / disabled in a live cluster?**

  - Feature gate name: `BoundServiceAccountVolume`
  - Components depending on the feature gate: kubelet, kube-apiserver
  - Will enabling / disabling the feature require downtime of the control
    plane? yes, need to restart kube apiserver.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? yes, need to restart kubelet.

- **Does enabling the feature change any default behavior?** yes, pods'
  service account tokens will not be long-lived and are not stored as Secrets
  any more.

- **Can the feature be disabled once it has been enabled (i.e. can we roll
  back the enablement)?** yes.

- **What happens if we reenable the feature if it was previously rolled
  back?** the same as the first enablement.

- **Are there any tests for feature enablement/disablement?** yes, unit tests
  will cover this.

### Scalability

- **Will enabling / using this feature result in any new API calls?**

  - API call type: `TokenRequest`
  - estimated throughput: 1/pod.
  - originating component: kubelet
  - components listing and/or watching resources they didn't before: N/A.
  - API calls that may be triggered by changes of some Kubernetes resources:
    N/A.
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.): N/A.

- **Will enabling / using this feature result in introducing new API types?**
  no.

- **Will enabling / using this feature result in any new calls to the cloud
  provider?** no.

- **Will enabling / using this feature result in increasing size or count of
  the existing API objects?** no.

- **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs]?** no.

- **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?** no.

## Alternatives

N/A

## Implementation History

N/A
