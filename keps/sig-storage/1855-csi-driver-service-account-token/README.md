# Service Account Token for CSI Driver

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [User stories](#user-stories)
- [Proposal](#proposal)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [API Changes](#api-changes)
  - [Example Workflow](#example-workflow)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha-&gt;Beta](#alpha-beta)
    - [Beta-&gt;GA](#beta-ga)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Scalability](#scalability)
- [Alternatives](#alternatives)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

This KEP proposes a way to obtain service account token for pods that the CSI
drivers are mounting volumes for. Since these tokens are valid only for a
limited period, this KEP will also give the CSI drivers an option to re-execute
`NodePublishVolume` to mount volumes.

## Motivation

Currently, the only way that CSI drivers acquire service account tokens is to
directly read the token in the file system. However, this approach has
uncharming traits:

1.  It will not work for csi drivers which run as a different non-root user than
    the pods. See
    [file permission section for service account token](https://github.com/kubernetes/enhancements/blob/master/keps/sig-storage/20180515-svcacct-token-volumes.md#file-permission).
2.  CSI driver will have access to the secrets of pods that do not use it
    because the CSI driver should have a `hostPath` volume for the `pods`
    subdirectory to read the token.
3.  The audience of the token is defaulted to kube apiserver.
4.  The token is not guaranteed to be available (e.g.
    `automountServiceAccountToken=false`).

### User stories

- HashiCorp Vault provider for secret store CSI driver requires service
  account token of the pods they are mounting secrets at to authenticate to
  Vaults. The provisioned secrets also have given TTL in Vault, so it is
  necessary get tokens after the initial mount.
- Cert manager CSI dirver will create CertificateRequests on behalf of the
  pods.
- Amazon EFS CSI driver wants the service account tokens of pods to exchange
  for AWS credentials.

## Proposal

### Goals

- Allow CSI driver to request audience-bounded service account tokens of pods
  from kubelet to `NodePublishVolume`.
- Provide an option to re-execute `NodePublishVolume` in a best-effort manner.

### Non-Goals

- Other CSI calls e.g. `NodeStageVolume` may not acquire pods' service account
  tokens via this feature.
- Failed re-execution of `NodePublishVolume` will not unmount volumes.

### API Changes

```go
// CSIDriverSpec is the specification of a CSIDriver.
type CSIDriverSpec struct {
    ... // existing fields

    RequiresRemount *bool
    ServiceAccountTokens []ServiceAccountToken
}

// ServiceAccountToken contains parameters of a token.
type ServiceAccountToken struct {
    Audience *string
    ExpirationSeconds *int64
}
```

These three fields are all optional:

- **`ServiceAccountToken.Audience`**: will be set in `TokenRequestSpec`. This
- will default to `APIAudiences` of kube-apiserver if it is empty. The storage
  provider of the CSI driver is supposed to send a `TokenReview` with at least
  one of the audiences specified.

- **`ServiceAccountToken.ExpirationSeconds`**: will be set in
  `TokenRequestSpec`. The issued token may have a different duration, so the
  `ExpirationTimestamp` in `TokenRequestStatus` will be passed to CSI driver.

- **`RequiresRemount`**: should be only set when the mounted volumes by the
  CSI driver have TTL and require re-validation on the token.

  - **Note**: Remount means re-execution of `NodePublishVolume` in scope of
    CSI and there is no intervening unmounts. If use this option,
    `NodePublishVolume` should only change the contents rather than the
    mount because container will not be restarted to reflect the mount
    change. The period between remounts is 0.1s which is hardcoded as
    `reconcilerLoopSleepPeriod` in volume manager. However, the rate
    `TokenRequest` is not 10/s because it will be cached until expiration.

The token will be bounded to the pod that the CSI driver is mounting volumes for
and will be set in `VolumeContext`:

```go
"csi.storage.k8s.io/serviceAccount.tokens": {
  'audience': {
    'token': token,
    'expiry': expiry,
  },
  ...
}
```

### Example Workflow

Take the Vault provider for secret store CSI driver as an example:

1.  Create `CSIDriver` object with `ServiceAccountToken[0].Audience=['vault']`
    and `RequiresRemount=true`.
2.  When the volume manager of kubelet sees a new volume, the pod object in
    `mountedPods` will have `requiresRemound=true` after `MarkRemountRequired`
    is called. `MarkRemountRequired` will call into `RequiresRemount` of the
    in-tree csi plugin to fetch the `CSIDriver` object.
3.  Before `NodePublishVolume` call, kubelet will request token from
    `TokenRequest` api with `audiences=['vault']`.
4.  The token will be specified in `VolumeContext` to `NodePublishVolume` call.
5.  Every 0.1 second, the reconciler component of volume manager will remount
    the volume in case the vault secrets expire and re-login is required.

### Notes/Constraints/Caveats

The `RequiresRemount` is useful when the mounted volumes can expire and the
availability and validity of volumes are continuously required. Those volumes
are most likely credentials which rotates for the best security practice. There
are two options when the remount failed:

1.  Keep the container/pod running and use the old credentials.
    - The next `NodePublishVolume` may succeed if it was unlucky transient
      failure.
    - Given there are multiple of 0.1 second usage of stale credentials, it is
      critical for the credential provisioners to guarantee that the validity
      is revoked after expiry. In general, it is much harder to eliminate the
      sinks than source.
    - The container/pod will also have better observability in usage of the
      stale credentials.
2.  Kill the container/pod and hopefully the new container/pod has the refreshed
    credentials.
    - This will reduce the stale volume exposure by one sink.
    - More likely to overcome fatal errors.
    - Container start-up cost is high

Option 1 is adopted. See discussion
[here](https://github.com/kubernetes/enhancements/pull/1855#discussion_r443040359).

### Test Plan

- Unit tests around all the added logic in kubelet.
- E2E tests around remount and token passing.

### Graduation Criteria

#### Alpha

- Implemented the feature.
- Wrote all the unit and E2E tests.

#### Alpha->Beta

- Deployed the feature in production and went through at least minor k8s.
- Fixed any bugs.

#### Beta->GA

- Deployed the feature in production and went through at least minor k8s.
  version.
- Wrote stress/scale tests to make sure the feature is still working where
  large number of pods are running.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

- **How can this feature be enabled / disabled in a live cluster?**

  - Feature gate name: CSIServiceAccountToken
  - Components depending on the feature gate: kubelet, kube-apiserver
  - Will enabling / disabling the feature require downtime of the control
    plane? no.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? yes.

- **Does enabling the feature change any default behavior?** no.

- **Can the feature be disabled once it has been enabled (i.e. can we roll
  back the enablement)?** yes, as long as the new fields in `CSIDriverSpec` is
  not used.

- **What happens if we reenable the feature if it was previously rolled
  back?** nothing, as long as the new fields in `CSIDriverSpec` is not used.

- **Are there any tests for feature enablement/disablement?** yes, unit tests
  will cover this.

### Scalability

- **Will enabling / using this feature result in any new API calls?**

  - API call type: `TokenRequest`
  - estimated throughput: 1(`RequiresRemount=false`) or
    1/ExpirationSeconds/s(`RequiresRemount=true`) for each CSI driver using
    this feature.
  - originating component: kubelet
  - components listing and/or watching resources they didn't before: n/a.
  - API calls that may be triggered by changes of some Kubernetes resources:
    n/a.
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.): n/a.

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

1.  Instead of fetching tokens in kubelet, CSI drivers will be granted
    permission to `TokenRequest` api. This will require non-trivial admission
    plugin to do necessary validation and every csi driver needs to reimplement
    the same functionality.

## Implementation History
