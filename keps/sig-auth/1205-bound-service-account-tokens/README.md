# Bound Service Account Tokens

## Table Of Contents

<!-- toc -->
- [Summary](#summary)
- [Background](#background)
- [Motivation](#motivation)
- [Design Details](#design-details)
  - [TokenRequest](#tokenrequest)
    - [Token Attenuations](#token-attenuations)
      - [Audience binding](#audience-binding)
      - [Time Binding](#time-binding)
      - [Object Binding](#object-binding)
    - [API Changes](#api-changes)
      - [Add <code>tokenrequests.authentication.k8s.io</code>](#add-tokenrequestsauthenticationk8sio)
      - [Modify <code>tokenreviews.authentication.k8s.io</code>](#modify-tokenreviewsauthenticationk8sio)
      - [Example Flow](#example-flow)
    - [Service Account Authenticator Modification](#service-account-authenticator-modification)
    - [ACLs for TokenRequest](#acls-for-tokenrequest)
  - [TokenRequestProjection](#tokenrequestprojection)
    - [API Change](#api-change)
    - [File Permission](#file-permission)
      - [Proposed Heuristics](#proposed-heuristics)
      - [Alternatives Considered](#alternatives-considered)
  - [ServiceAccount Admission Controller Migration](#serviceaccount-admission-controller-migration)
    - [Prerequisites](#prerequisites)
    - [Safe Rollout of Time-bound Token](#safe-rollout-of-time-bound-token)
  - [Test Plan](#test-plan)
    - [TokenRequest/TokenRequestProjection](#tokenrequesttokenrequestprojection)
    - [RootCAConfigMap](#rootcaconfigmap)
    - [BoundServiceAccountTokenVolume](#boundserviceaccounttokenvolume)
  - [Graduation Criteria](#graduation-criteria)
    - [TokenRequest/TokenRequestProjection](#tokenrequesttokenrequestprojection-1)
      - [Beta-&gt;GA](#beta-ga)
    - [RootCAConfigMap](#rootcaconfigmap-1)
      - [Beta-&gt;GA](#beta-ga-1)
    - [BoundServiceAccountTokenVolume](#boundserviceaccounttokenvolume-1)
      - [Alpha-&gt;Beta](#alpha-beta)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
<!-- /toc -->

## Summary

This KEP describes an API that would allow workloads running on Kubernetes to
request JSON Web Tokens that are audience, time and eventually key bound. In
addition, this KEP introduces a new mechanism of distribution with support for
bound service account tokens and explores how to migrate from the existing
mechanism backwards compatibly.

## Background

Kubernetes already provisions JWTs to workloads. This functionality is on by
default and thus widely deployed. The current workload JWT system has serious
issues:

1.  Security: JWTs are not audience bound. Any recipient of a JWT can masquerade
    as the presenter to anyone else.
1.  Security: The current model of storing the service account token in a Secret
    and delivering it to nodes results in a broad attack surface for the
    Kubernetes control plane when powerful components are run - giving a service
    account a permission means that any component that can see that service
    account's secrets is at least as powerful as the component.
1.  Security: JWTs are not time bound. A JWT compromised via 1 or 2, is valid
    for as long as the service account exists. This may be mitigated with
    service account signing key rotation but is not supported by client-go and
    not automated by the control plane and thus is not widely deployed.
1.  Scalability: JWTs require a Kubernetes secret per service account.

## Motivation

We would like to introduce a new mechanism for provisioning Kubernetes service
account tokens that is compatible with our current security and scalability
requirements.

## Design Details

### TokenRequest

Infrastructure to support on demand token requests will be implemented in the
core apiserver. Once this API exists, a client of the apiserver will request an
attenuated token for its own use. The API will enforce required attenuations,
e.g. audience and time binding.

#### Token Attenuations

##### Audience binding

Tokens issued from this API will be audience bound. Audience of requested tokens
will be bound by the `aud` claim. The `aud` claim is an array of strings
(usually URLs) that correspond to the intended audience of the token. A
recipient of a token is responsible for verifying that it identifies as one of
the values in the audience claim, and should otherwise reject the token. The
TokenReview API will support this validation.

##### Time Binding

Tokens issued from this API will be time bound. Time validity of these tokens
will be claimed in the following fields:

- `exp`: expiration time
- `nbf`: not before
- `iat`: issued at

A recipient of a token should verify that the token is valid at the time that
the token is presented, and should otherwise reject the token. The TokenReview
API will support this validation.

Cluster administrators will be able to configure the maximum validity duration
for expiring tokens. During the migration off of the old service account tokens,
clients of this API may request tokens that are valid for many years. These
tokens will be drop in replacements for the current service account tokens.

##### Object Binding

Tokens issued from this API may be bound to a Kubernetes object in the same
namespace as the service account. The name, group, version, kind and uid of the
object will be embedded as claims in the issued token. A token bound to an
object will only be valid for as long as that object exists.

Only a subset of object kinds will support object binding. Initially the only
kinds that will be supported are:

- v1/Pod
- v1/Secret

The TokenRequest API will validate this binding.

#### API Changes

##### Add `tokenrequests.authentication.k8s.io`

We will add an imperative API (a la TokenReview) to the `authentication.k8s.io`
API group:

```golang
type TokenRequest struct {
  Spec   TokenRequestSpec
  Status TokenRequestStatus
}

type TokenRequestSpec struct {
  // Audiences are the intendend audiences of the token. A token issued
  // for multiple audiences may be used to authenticate against any of
  // the audiences listed. This implies a high degree of trust between
  // the target audiences.
  Audiences []string

  // ValidityDuration is the requested duration of validity of the request. The
  // token issuer may return a token with a different validity duration so a
  // client needs to check the 'expiration' field in a response.
  ValidityDuration metav1.Duration

  // BoundObjectRef is a reference to an object that the token will be bound to.
  // The token will only be valid for as long as the bound object exists.
  BoundObjectRef *BoundObjectReference
}

type BoundObjectReference struct {
  // Kind of the referent. Valid kinds are 'Pod' and 'Secret'.
  Kind string
  // API version of the referent.
  APIVersion string

  // Name of the referent.
  Name string
  // UID of the referent.
  UID types.UID
}

type TokenRequestStatus struct {
  // Token is the token data
  Token string

  // Expiration is the time of expiration of the returned token. Empty means the
  // token does not expire.
  Expiration metav1.Time
}

```

This API will be exposed as a subresource under a serviceaccount object. A
requestor for a token for a specific service account will `POST` a
`TokenRequest` to the `/token` subresource of that serviceaccount object.

##### Modify `tokenreviews.authentication.k8s.io`

The TokenReview API will be extended to support passing an additional audience
field which the service account authenticator will validate.

```golang
type TokenReviewSpec struct {
  // Token is the opaque bearer token.
  Token string
  // Audiences is the identifier that the client identifies as.
  Audiences []string
}
```

##### Example Flow

```
> POST /apis/v1/namespaces/default/serviceaccounts/default/token
> {
>   "kind": "TokenRequest",
>   "apiVersion": "authentication.k8s.io/v1",
>   "spec": {
>     "audience": [
>       "https://kubernetes.default.svc"
>     ],
>     "validityDuration": "99999h",
>     "boundObjectRef": {
>       "kind": "Pod",
>       "apiVersion": "v1",
>       "name": "pod-foo-346acf"
>     }
>   }
> }
{
  "kind": "TokenRequest",
  "apiVersion": "authentication.k8s.io/v1",
  "spec": {
    "audience": [
      "https://kubernetes.default.svc"
    ],
    "validityDuration": "99999h",
    "boundObjectRef": {
      "kind": "Pod",
      "apiVersion": "v1",
      "name": "pod-foo-346acf"
    }
  },
  "status": {
    "token":
    "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJz[payload omitted].EkN-[signature omitted]",
    "expiration": "Jan 24 16:36:00 PST 3018"
  }
}
```

The token payload will be:

```
{
  "iss": "https://example.com/some/path",
  "sub": "system:serviceaccount:default:default,
  "aud": [
    "https://kubernetes.default.svc"
  ],
  "exp": 24412841114,
  "iat": 1516841043,
  "nbf": 1516841043,
  "kubernetes.io": {
    "serviceAccountUID": "c0c98eab-0168-11e8-92e5-42010af00002",
    "boundObjectRef": {
      "kind": "Pod",
      "apiVersion": "v1",
      "uid": "a4bb8aa4-0168-11e8-92e5-42010af00002",
      "name": "pod-foo-346acf"
    }
  }
}
```

#### Service Account Authenticator Modification

The service account token authenticator will be extended to support validation
of time and audience binding claims.

#### ACLs for TokenRequest

The NodeAuthorizer will allow the kubelet to use its credentials to request a
service account token on behalf of pods running on that node. The
NodeRestriction admission controller will require that these tokens are pod
bound.

### TokenRequestProjection

A ServiceAccountToken volume projection that maintains a service account token
requested by the node from the TokenRequest API.

#### API Change

A new volume projection will be implemented with an API that closely matches the
TokenRequest API.

```go
type ProjectedVolumeSource struct {
  Sources []VolumeProjection
  DefaultMode *int32
}

type VolumeProjection struct {
  Secret *SecretProjection
  DownwardAPI *DownwardAPIProjection
  ConfigMap *ConfigMapProjection
  ServiceAccountToken *ServiceAccountTokenProjection
}

// ServiceAccountTokenProjection represents a projected service account token
// volume. This projection can be used to insert a service account token into
// the pods runtime filesystem for use against APIs (Kubernetes API Server or
// otherwise).
type ServiceAccountTokenProjection struct {
  // Audience is the intended audience of the token. A recipient of a token
  // must identify itself with an identifier specified in the audience of the
  // token, and otherwise should reject the token. The audience defaults to the
  // identifier of the apiserver.
  Audience string
  // ExpirationSeconds is the requested duration of validity of the service
  // account token. As the token approaches expiration, the kubelet volume
  // plugin will proactively rotate the service account token. The kubelet will
  // start trying to rotate the token if the token is older than 80 percent of
  // its time to live or if the token is older than 24 hours.Defaults to 1 hour
  // and must be at least 10 minutes.
  ExpirationSeconds int64
  // Path is the relative path of the file to project the token into.
  Path string
}
```

A volume plugin implemented in the kubelet will project a service account token
sourced from the TokenRequest API into volumes created from
ProjectedVolumeSources. As the token approaches expiration, the kubelet volume
plugin will proactively rotate the service account token. The kubelet will start
trying to rotate the token if the token is older than 80 percent of its time to
live or if the token is older than 24 hours.

To replace the current service account token secrets, we also need to inject the
clusters CA certificate bundle. We will deploy it as a configmap per-namespace
and reference it using a ConfigMapProjection.

A projected volume source that is equivalent to the current service account
secret:

```yaml
- name: kube-api-access-xxxxx
  projected:
    defaultMode: 420 # 0644
    sources:
      - serviceAccountToken:
          expirationSeconds: 3600
          path: token
      - configMap:
          items:
            - key: ca.crt
              path: ca.crt
          name: kube-root-ca.crt
      - downwardAPI:
          items:
            - fieldRef:
                apiVersion: v1
                fieldPath: metadata.namespace
              path: namespace
```

#### File Permission

The secret projections are currently written with world readable (0644,
effectively 444) file permissions. Given that file permissions are one of the
oldest and most hardened isolation mechanisms on unix, this is not ideal.
We would like to opportunistically restrict permissions for projected service
account tokens as long we can show that they won’t break users if we are to
migrate away from secrets to distribute service account credentials.

##### Proposed Heuristics

- _Case 1_: The pod has an fsGroup set. We can set the file permission on the
  token file to 0600 and let the fsGroup mechanism work as designed. It will
  set the permissions to 0640, chown the token file to the fsGroup and start
  the containers with a supplemental group that grants them access to the
  token file. This works today.
- _Case 2_: The pod’s containers declare the same runAsUser for all containers
  (ephemeral containers are excluded) in the pod. We chown the token file to
  the pod’s runAsUser to grant the containers access to the token. All
  containers must have UID either specified in container security context or
  inherited from pod security context. Preferred UIDs in container images are
  ignored.
- _Fallback_: We set the file permissions to world readable (0644) to match
  the behavior of secrets.

This gives users that run as non-root greater isolation between users without
breaking existing applications. We also may consider adding more cases in the
future as long as we can ensure that they won’t break users.

##### Alternatives Considered

- We can create a volume for each UserID and set the owner to be that UserID
  with mode 0400. If user doesn't specify runAsUser, fetching UserID in image
  requires a re-design of kubelet regarding volume mounts and image pulling.
  This has significant implementation complexity because:
  - We would have to reorder container creation to introspect images (that
    might declare USER or GROUP directives) to pass this information to the
    projected volume mounter.
  - Further, images are mutable so these directives may change over the
    lifetime of the pod.
  - Volumes are shared between all pods that mount them today. Mapping a
    single logical volume in a pod spec to distinct mount points is likely a
    significant architectural change.
- We pick a random group and set fsGroup on all pods in the service account
  admission controller. It’s unclear how we would do this without conflicting
  with usage of groups and potentially compromising security.
- We set token files to be world readable always. Problems with this are
  discussed above.

### ServiceAccount Admission Controller Migration

#### Prerequisites

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

    - Go: >= v0.15.7
    - Python: >= v12.0.0
    - Java: >= v9.0.0
    - Javascript: >= v0.10.3
    - Ruby: master branch
    - Haskell: v0.3.0.0

    For community-maintained client libraries, feel free to contribute to them
    if the reloading logic is missing.

    **Note**: If having trouble in finding places using in-cluster config
    completely, cluster operators can specify flag
    `--service-account-extend-token-expiration=true` to kube apiserver to allow
    tokens have longer expiration temporarily during the migration. Any usage of
    legacy token will be recorded in both metrics and audit logs. After fixing
    all the potentially broken workloads, turn off the flag so that the original
    expiration settings are honored. Note the
    `--service-account-extend-token-expiration` mitigation defaults to true, and
    that cluster administrators can set it to
    `--service-account-extend-token-expiration=false` to turn off the mitigation
    if desired.

    - Metrics: `serviceaccount_stale_tokens_total`
    - Audit: looking for `authentication.k8s.io/stale-token` annotation

    See next section for the details of how to discover the workloads that will
    suffer from expired tokens.

If anything goes wrong, please file a bug and CC @kubernetes/sig-auth-bugs. More
contact information
[here](https://github.com/kubernetes/community/tree/master/sig-auth#contact).

#### Safe Rollout of Time-bound Token

Legacy service account tokens distributed via secrets are not time-bound. Many
client libraries have come to depend on this behavior. After time-bound service
account token being used, if in-cluster clients do not periodically reload token
from projected volume, requests would be rejected once the initial token got
expired.

In order to allow guadual adoption of time-bound token, we would:

1.  Pick a constant period D between one and two hours. The value of D would be
    static across Kubernetes deployments, while avoiding collision with common
    duration.
1.  Modify service account admission control to inject token valid for D when
    the BoundServiceAccountTokenVolume feature is enabled.
1.  Modify kube apiserver TokenRequest API. When it receives TokenRequest with
    requested valid period D, extend the token lifetime to one year. At the same
    time, save the original requested D to `kubernetes.io/warnafter` field in
    minted token.
1.  In the TokenRequest status, tell clients that the token would be valid only
    for D, encouraging clients to reload token as if the token was valid for D.

This modification could be optionally enabled by providing a command line flag
to kube apiserver.

These extended tokens would not expire and continue to be accepted within one
year. At the same time, the authentication side could monitor whether clients
are properly reloading tokens by:

1.  Compare the `kubernetes.io/warnafter` field with current time. If current
    time is after `kubernetes.io/warnafter` field, it implies calling client is
    not reloading token regularly.
1.  Expose metrics to monitor number of legacy and stale token used.
1.  Add annotation to audit events for legacy and stale tokens including
    necessary information to locate problematic client.

### Test Plan

#### TokenRequest/TokenRequestProjection

- Unit tests
- E2E tests
  - Projected jwt tokens are correctly mounted. (conformance test)
  - The owner and mode of projected tokens are correctly set
  - In-cluster clients work with Token rotation

#### RootCAConfigMap

- Unit tests
- E2E tests
  - Every namespace has configmap `kube-root-ca.crt`

#### BoundServiceAccountTokenVolume

- Unit tests
- An upgrade test

1.  Create pod A with feature disabled where pod A is working and a secret
    volume is mounted
2.  Enable feature where pod A continue working
3.  Create pod B and it is working and projected volumes are mounted

### Graduation Criteria

#### TokenRequest/TokenRequestProjection

| Alpha | Beta | GA   |
| ----- | ---- | ---- |
| 1.10  | 1.12 | 1.20 |

##### Beta->GA

- [x] In use by multiple distributions
- [x] Approved by PRR and scalability
- [x] Any known bugs fixed
- [x] Tests passing
  - [x] E2E test [ServiceAccounts should mount projected service account
        token when requested](https://k8s-testgrid.appspot.com/sig-auth-gce#gce)
  - [x] E2E test [ServiceAccounts should set ownership and permission when
        RunAsUser or FsGroup is
        present](https://k8s-testgrid.appspot.com/sig-auth-gce#gce)
  - [x] E2E test
        [ServiceAccounts should support InClusterConfig with token rotation](https://k8s-testgrid.appspot.com/sig-auth-gce#gce-slow)

#### RootCAConfigMap

| Alpha | Beta | GA   |
| ----- | ---- | ---- |
| 1.13  | 1.20 | 1.21 |

##### Beta->GA

- [x] In use by multiple distributions
- [x] Approved by PRR and scalability
- [x] Any known bugs fixed

#### BoundServiceAccountTokenVolume

| Alpha | Beta | GA   |
| ----- | ---- | ---- |
| 1.13  | 1.21 | 1.22 |

##### Alpha->Beta

- [x] Any known bugs fixed

  - [x] PodSecurityPolicies that allow secrets but not projected volumes
        will prevent the use of token volumes.
    - Fixed in https://github.com/kubernetes/kubernetes/pull/92006
  - [x] In-cluster clients that don’t reload service account tokens will
        start failing an hour after deployment.
  - Mitigation added in
    https://github.com/kubernetes/kubernetes/issues/68164
  - [x] Pods running as non root may not access the service account token.
    - Fixed in https://github.com/kubernetes/kubernetes/pull/89193
  - [x] Dynamic clientbuilder does not invalidate token.
    - Fixed in https://github.com/kubernetes/kubernetes/pull/99324

* [x] Tests passing

  - [x] Upgrade test
        [sig-auth-serviceaccount-admission-controller-migration](https://k8s-testgrid.appspot.com/sig-auth-gce#upgrade-tests)

* [x] TokenRequest/TokenRequestProjection GA

* [x] RootCAConfigMap GA

##### Beta -> GA Graduation

- [x] Allow kube-apiserver to recognize multiple issuers to enable non
      disruptive issuer change.
  - Fixed in https://github.com/kubernetes/kubernetes/pull/101155
- [x] New `ServiceAccount` admission controller work as intended in Beta
      for >= 1 minor release without significant issues.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

- **How can this feature be enabled / disabled in a live cluster?**

  - Feature gate name: `BoundServiceAccountTokenVolume`
  - Components depending on the feature gate: kube-apiserver and
    kube-controller-manager
  - Will enabling / disabling the feature require downtime of the control
    plane? yes, need to restart kube-apiserver and kube-controller-manager.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? no.

- **Does enabling the feature change any default behavior?** yes, pods'
  service account tokens will expire after 1 year by default and are not
  stored as Secrets any more.

- **Can the feature be disabled once it has been enabled (i.e. can we roll
  back the enablement)?** yes.

- **What happens if we reenable the feature if it was previously rolled
  back?** the same as the first enablement.

- **Are there any tests for feature enablement/disablement?**

  - unit test: plugin/pkg/admission/serviceaccount/admission_test.go
  - upgrade test:
    test/e2e/upgrades/serviceaccount_admission_controller_migration.go

### Rollout, Upgrade and Rollback Planning

- **How can a rollout fail? Can it impact already running workloads?**

  1.  creation of CA configmap can fail due to permission / quota / admission
      errors.
  2.  newly issued tokens could fail to be recognized by skewed API servers
      not configured with the bound token signing key/issuer.

- **What specific metrics should inform a rollback?**

  1.  creation of CA configmap,
      - `root_ca_cert_publisher_rate_limiter_use`
  2.  authentication errors in (n-1) API servers,
      - `authentication_attempts`
      - `authentication_duration_seconds`

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path
  tested?**
  for upgrade, we have set up e2e test running here:
  https://k8s-testgrid.appspot.com/sig-auth-gce#upgrade-tests&width=5

  for downgrade, we have manually tested where a workload continues to
  authenticate successfully.

- **Is the rollout accompanied by any deprecations and/or removals of
  features, APIs, fields of API types, flags, etc.?** no

### Monitoring Requirements

- **How can an operator determine if the feature is in use by workloads?**

  Check TokenRequest in use:

  - `serviceaccount_valid_tokens_total`: cumulative valid projected service
    account tokens used
  - `serviceaccount_stale_tokens_total`: cumulative stale projected service
    account tokens used
  - `apiserver_request_total`: with labels `group="",version="v1",resource="serviceaccounts",subresource="token"`
  - `apiserver_request_duration_seconds`: with labels `group="",version="v1",resource="serviceaccounts",subresource="token"`

- **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**

  - [x] Metrics
    - Metric name: apiserver_request_total
    - Aggregation method: group="",version="v1",resource="serviceaccounts",subresource="token"
    - Components exposing the metric: kube-apiserver

- **What are the reasonable SLOs (Service Level Objectives) for the above
  SLIs?**

  - per-day percentage of API calls finishing with 5XX errors <= 1%

- **Are there any missing metrics that would be useful to have to improve
  observability of this feature?**

  - add granularity to `storage_operation_duration_seconds` to distinguish
    projected volumes: configmap, secret, token,..etc... or add new metrics
    so that we can know the usage of projected tokens.

### Dependencies

- **Does this feature depend on any specific services running in the
  cluster?** There are no new components required, but specific versions of
  kubelet and kube-controller-manager are required

  TokenRequest depends on kubelets >= 1.12

  BoundServiceAccountTokenVolume depends on kubelets >= 1.12 with TokenRequest
  enabled (default since 1.12) and kube-controller-manager >= 1.12 with
  RootCAConfigMap feature enabled (default since 1.20)

### Scalability

- **Will enabling / using this feature result in any new API calls?**

  - API call type: `TokenRequest`
  - estimated throughput: 1/pod every ~48 minutes.
  - originating component: kubelet
  - components listing and/or watching resources they didn't before: N/A.
  - API calls that may be triggered by changes of some Kubernetes resources:
    N/A.
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.): 1 call per pod every ~48 minutes.

- **Will enabling / using this feature result in introducing new API types?**
  no.

- **Will enabling / using this feature result in any new calls to the cloud
  provider?** no.

- **Will enabling / using this feature result in increasing size or count of
  the existing API objects?** controller creates one additional configmap per
  namespace.

- **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs]?** no.

- **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?** it adds a
  token minting operation in the API server every ~48 minutes for every pod.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may
consider splitting it into a dedicated `Playbook` document (potentially with
some monitoring details). For now, we leave it here.

- **How does this feature react if the API server and/or etcd is
  unavailable?**

  - TokenRequest API is unavailable
  - configmap containing API server CA bundle cannot be created or fetched

* **What are other known failure modes?**

  - failure to issue token via token subresource

    - Detection: check `apiserver_request_total` with labels
      `group="",version="v1",resource="serviceaccounts",subresource="token"`
    - Mitigations: disable the BoundServiceAccountTokenVolume feature gate in
      the kube-apiserver and recreate pods.
    - Diagnostics: "failed to generate token" in kube-apiserver log.
    - Testing: [e2e test](https://k8s-testgrid.appspot.com/sig-auth-gce#gce&width=5&include-filter-by-regex=ServiceAccounts%20should%20mount%20projected%20service%20account%20token)

  - failure to create root CA config map

    - Detection: check `root_ca_cert_publisher_sync_total` from
      kube-controller-manager. (available in 1.21+)
    - Mitigations: disable the BoundServiceAccountTokenVolume feature gate in
      the kube-apiserver and recreate pods.
    - Diagnostics: "syncing [namespace]/[configmap name] failed" in
      kube-controller-manager log.
    - Testing: [e2e test](https://k8s-testgrid.appspot.com/sig-auth-gce#gce&width=5&include-filter-by-regex=ServiceAccounts%20should%20guarantee%20kube-root-ca.crt%20exist%20in%20any%20namespace)

  - kubelet fails to renew token

    - Detection: check `apiserver_request_total` with labels
      `group="",version="v1",resource="serviceaccounts",subresource="token"` to
      see if failed in requesting a new token; check kubelet log.
    - Mitigations: disable the BoundServiceAccountTokenVolume feature gate in
      the kube-apiserver and recreate pods.
    - Diagnostics: "token [namespace]/[token name] expired and refresh failed"
      in kubelet log.
    - Testing: [e2e test](https://k8s-testgrid.appspot.com/sig-auth-gce#gce-slow&width=5)

  - workload fails to refresh token from disk

    - Detection: `serviceaccount_stale_tokens_total` emitted by kube-apiserver
    - Mitigations: update client library to newer version.
    - Diagnostics: look for `authentication.k8s.io/stale-token` in audit log if
      `--service-account-extend-token-expiration=true`, or check authentication
      error in kube-apiserver log.
    - Testing: covered in all client libraries' unittests.

* **What steps should be taken if SLOs are not being met to determine the
  problem?** Check kube-apiserver, kube-controller-managera and kubelet logs.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing slis/slos]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
