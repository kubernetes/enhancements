# Support external signing of service account tokens

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Preserve existing behavior](#preserve-existing-behavior)
    - [New API](#new-api)
    - [Plugins](#plugins)
    - [Updates for token generation](#updates-for-token-generation)
    - [Updates for supported public keys](#updates-for-supported-public-keys)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
    - [Support for Legacy Tokens](#support-for-legacy-tokens)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [ExternalJWTSigner RPC](#externaljwtsigner-rpc)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade/Downgrade Strategy](#upgradedowngrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
    - [Different kube-apiserver versions running at the same time](#different-kube-apiserver-versions-running-at-the-same-time)
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
- [Possible Future Work](#possible-future-work)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] ~~e2e Tests for all Beta API Operations (endpoints)~~ no API endpoints
  - [ ] ~~(R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)~~ no API endpoints
  - [ ] ~~(R) Minimum Two Week Window for GA e2e tests to prove flake free~~ no API endpoints
- [x] (R) Graduation criteria is in place
  - [ ] ~~(R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)~~ no API endpoints
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Service account keys are used by kube-apiserver for JWT signing and authentication. Keys are stored on the disk and are loaded by kube-apiserver during process start time. Key management for service account keys is entirely within the cluster and is less flexible in that the signing keys remain the same for the lifetime of the process. This KEP proposes updates that will allow Kubernetes distributions to integrate with key management solutions of their choice (eg: HSMs, cloud KMSes).  

## Motivation

1. Ease of rotation.
  At present, kube-apiserver loads service account keys at process start time and the keys remain the same for the life-time of the process. Thus, rotating keys require a kube-apiserver to restart. External key management eliminates the need to restart.

2. Enhanced security.
  kube-apiserver loads service account keys from a file on the disk. This means anyone with privileged access to the kube-apiserver configuration / filesystem could exfiltrate signing material. An external signer that never returns signing material would mitigate this risk.

### Goals

- Enable integration with HSMs and cloud KMSes
- Support out-of-process JWT signing
- Support listing public verifying keys
- Ensure tokens signed by external signers remain consistent with tokens issued by kube-apiserver (consistent headers, algorithms, claims, etc)
- Preserve existing behavior and performance for keys not read over a socket

### Non-Goals

- Reading TLS serving certificates and key from a socket or reloading of kube-apiserver with new cert and key
- Reading any other certificates from a file

## Proposal

### User Stories

Allow kubernetes distributions to integrate with external JWT signers while preserving existing behavior. Choosing between existing and new behavior shall be configurable.

#### Preserve existing behavior

The kube-apiserver flags `--service-account-key-file` and `--service-account-signing-key-file` will continue to be used for reading from files (unless, `--service-account-signing-endpoint` is set; They are mutually exclusive ways of supporting JWT signing and authentication).

#### New API

A new versioned grpc API (ExternalJWTSigner) will be created under `k8s.io/kubernetes/pkg/serviceaccount`. This will be similar to how the KMS envelope encryption has an API at `k8s.io/kubernetes/staging/src/k8s.io/kms/apis/v2/api.proto`. It will have methods to sign JWTs and fetch supported keys (proto in [Design Details](#design-details) section). 

#### Plugins

- A signer plugin will be a client to the new grpc API thus providing support for external signing.
- A public key cache will integrate with the same grpc to fetch supported public keys.

#### Updates for token generation

- The Kubernetes Control Plane config accepts `ServiceAccountIssuer` which is an interface of type `serviceaccount.TokenGenerator`
- `serviceaccount.TokenGenerator` is defined in `pkg/serviceaccount/jwt.go`.
- Current implementation of `serviceaccount.TokenGenerator` is `serviceaccount.JWTTokenGenerator` which uses a static key for signing.
- This static implementation can be replaced by a Dynamic one which integrates with ExternalJWTSigner if external signing is enabled.

#### Updates for supported public keys

- kube-apiserver holds reference to a static implementation of `ServiceAccountPublicKeysGetter` interface called `StaticPublicKeysGetter`.
- For clusters configured with an external signer, this static implementation will be replaced by a Dynamic one backed by a key cache (let's call it `ExternalPublicKeysGetter`).
- OIDC Discovery docs are served by `OpenIDMetadataServer` and it gets supported keys from the same `ServiceAccountPublicKeysGetter` as configured for kube-apiserver. 
- Thus, the OIDC endpoint will automatically serve external keys if the kube-apiserver is configured to use `ExternalPublicKeysGetter`. 

### Notes/Constraints/Caveats

- A new flag `--service-account-signing-endpoint` will be added to kube-apiserver specifying a unix socket where the key service will be accessible.
- The flag `--service-account-signing-endpoint` can either be set to the location of a UDS on a filesystem, or be prefixed with an @ symbol and name a UDS in the abstract socket namespace.
   - Implementers running the kube-apiserver as a pod should note that using a filesystem based socket will weaken the pod isolation between kube-apiserver and external signer. That's because:
      - external has to run a root init container to set up the directory that will contain the socket on the host.
      - kube-apiserver has to have an external signer's group as a supplemental group in its security context.
    - In contrast, abstract sockets allow any unix user to connect to the socket, but the plugin must get the UID and GID of the calling user and enforce its own access control based on that.
- combining local and external keys will not be supported inside the kube-apiserver. This *could* be achieved inside an external signer implementation by combining local public keys with remote public keys and remote signing requests to accommodate a migration.
- Specifying both, `--service-account-key-file/--service-account-signing-key-file` and `--service-account-signing-endpoint` shall result in an error.

#### Support for Legacy Tokens

Implementers will have following options for legacy token support:
1. Turn off the loop (don't support legacy tokens) if external signing is enabled. (recommended to avoid non-expiring tokens)
2. Let the Controller loop run as it is with static signing keys. Stitch the public keys in external signer's JWKs.
3. Turn off the loop in kube-controller-manager and create a custom external signer for legacy tokens that obtains them via the external signer.

### Risks and Mitigations

- **Risk:** New token generation and validation could suffer a performance difference due to communication over a socket and a potential remote RPC call (will vary for different signer implementations).
  
  **Mitigation:** Performance overhead will be benchmarked and documented for integrators.

- **Risk:** Signing and verifying tokens over a grpc API carries the risk of a server side request forgery, where a malicious client could generate tokens.
  
  **Mitigation:** The API will only be accessible over a unix socket protected by standard file permissions; This is already done for other sensitive integrations like KMS. Documentation will be added to ensure that distributions limit the socket to privileged or specific uids.

- **Risk:** With an external signer, token headers and claims can diverge from that in existing JWTs as signed by kube-apiserver.
  
  **Mitigation:** External signers will only return signature and header while kube-apiserver will assemble the JWT. This will allow kube-apiserver to validate the header and control the claims.

## Design Details

### ExternalJWTSigner RPC

Will be served on a socket as configured via the `--service-account-signing-endpoint` flag.

```proto
syntax = "proto3";

// This service is served by a process on a local Unix Domain Socket.
service ExternalJWTSigner {
  // Sign takes a serialized JWT payload, and returns the serialized header and
  // signature.  The caller can then assemble the JWT from the header, payload,
  // and signature.
  //
  // The plugin MUST set a key id in the returned JWT header.
  rpc Sign(SignJWTRequest) returns (SignJWTResponse) {}

  // FetchKeys returns the set of public keys that are trusted to sign
  // Kubernetes service account tokens. Kube-apiserver will call this RPC:
  //
  // * Every time it tries to validate a JWT from the service account issuer with an unknown key ID, and
  //
  // * Periodically, so it can serve reasonably-up-to-date keys from the OIDC
  //   JWKs endpoint.
  rpc FetchKeys(FetchKeysRequest) returns (FetchKeysResponse) {}
  
  // Metadata is meant to be called once on startup.
  // Enables sharing metadata with kube-apiserver (eg: the max token lifetime that signer supports)
  rpc Metadata(MetadataRequest) returns (MetadataResponse) {}
}

message SignJWTRequest {
  // URL-safe base64 wrapped payload to be signed.
  // Exactly as it appears in the second segment of the JWT
  string claims = 1;
}

message SignJWTResponse {
  // header must contain only alg, kid, typ claims.
  // type must be “JWT”.
  // kid must be non-empty and its corresponding public key should not be excluded from OIDC discovery.
  // alg must be one of the algorithms supported by kube-apiserver (currently RS256, ES256, ES384, ES512).
  // header cannot have any additional data that kube-apiserver does not recognize.
  // Already wrapped in URL-safe base64, exactly as it appears in the first segment of the JWT.
  string header = 1;

  // The signature for the JWT.
  // Already wrapped in URL-safe base64, exactly as it appears in the final segment of the JWT.
  string signature = 2;
}

message FetchKeysRequest {}

message FetchKeysResponse {
  repeated Key keys = 1;

  // The timestamp when this data was pulled from the authoritative source of
  // truth for verification keys.
  // kube-apiserver can export this from metrics, to enable end-to-end SLOs.
  google.protobuf.Timestamp data_timestamp = 2;

  // refresh interval for verification keys to pick changes if any.
  // any value <= 0 is considered a misconfiguration.
  int64 refresh_hint_seconds = 3;
}

message Key {
  // A unique identifier for this key.
  string key_id = 1;

  // The public key, PKIX-serialized.
  // must be a public key supported by kube-apiserver (currently RSA 256 or ECDSA 256/384/521)
  bytes key = 2;

  // Set only for keys that are not used to sign bound tokens.
  // eg: supported keys for legacy tokens.
  // If set, key is used for verification but excluded from OIDC discovery docs.
  // if set, external signer should not use this key to sign a JWT.
  bool exclude_from_oidc_discovery = 3;
}

message MetadataRequest {}

message MetadataResponse {
  // used by kube-apiserver as the max token lifetime and for validation against configuration flag values:
  // 1. `--service-account-max-token-expiration`
  // 2. `--service-account-extend-token-expiration`
  //
  // * If `--service-account-max-token-expiration` is set while external-jwt-signer is configured, kube-apiserver treats that as misconfiguration and exits.
  // * If `--service-account-max-token-expiration` is not set, kube-apiserver uses `max_token_expiration_seconds` as max token lifetime.
  // * If `--service-account-extend-token-expiration` is true, the extended expiration is `min(1 year, max_token_expiration_seconds)`.
  //
  // `max_token_expiration_seconds` must be at least 600s.
  int64 max_token_expiration_seconds = 1;
}

```

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

<!-- 
  coverage at https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit
 -->

- `pkg/kubeapiserver/options` : `10-08-2024` - `67.2`
- `pkg/controlplane/apiserver/options` : `10-08-2024` - `57.1`
- `pkg/serviceaccount/` : `10-08-2024` - `74.4`

##### Integration tests

- Create a cluster with ExternalJWTSigner to configure an external signer and verify TokenRequest and TokenReview APIs work properly.
- Request a token for a service account principal.
- Use a token as bearer for making requests to kube-apiserver and ensure it succeeds.

- [TestExternalJWTSigningAndAuth](https://github.com/kubernetes/kubernetes/blob/8aae5398b3885dc271d407c4d661e19653daaf88/test/integration/serviceaccount/external_jwt_signer_test.go#L46C6-L46C35): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=serviceaccount), [triage search](https://storage.googleapis.com/k8s-triage/index.html?job=integration&test=serviceaccount)
- [TestDelayedStartForSigner](https://github.com/kubernetes/kubernetes/blob/8aae5398b3885dc271d407c4d661e19653daaf88/test/integration/serviceaccount/external_jwt_signer_test.go#L282): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=serviceaccount), [triage search](https://storage.googleapis.com/k8s-triage/index.html?job=integration&test=serviceaccount)

### Graduation Criteria

#### Alpha

- Unit tests are completed
- Integration tests are completed with a dummy ExternalSigner.

#### Beta

- All tests are completed.
- We have at least one ExternalSigner integration working with this change.
  - GKE integration is complete
- Decide whether to externalize legacy token controller code in a staging repo. Check [Support for Legacy Tokens](#support-for-legacy-tokens) for details.
  - Decided not to externalize legacy token controller code

#### GA

- More than one ExternalSigner integration are completed.
- Feature is tuned with feedback from distributions.

### Upgrade/Downgrade Strategy

Existing clusters preserving existing functionality: 
- Need no changes.

Existing clusters using enhanced functionality: 
- Need a control plane plugin that implements ExternalJWTSigner; To honor existing tokens during upgrade/migration until those tokens expire, it must be able to merge in the public keys for the previous static signing keys.
- Need to configure the kube-apiserver with the `--service-account-signing-endpoint` flag. 


### Version Skew Strategy

The change will completely live in the Control plane and thus, there is no question of a skew with component interaction. The only potential for a skew arises with different kube-apiserver versions running at the same time

#### Different kube-apiserver versions running at the same time
- Skew is likely to exist when externalSigning is enabled/disabled in a live cluster.
- Distributions can handle it by include old keys in new supported set
    - It will enable a smoother transition when enabling/disabling external signing on live clusters.
    - When disabling, distributions shall import keys from external signers to the on-disk key set.
    - When enabling, distributions shall implement an external signer such that it combines the external keys with on-disk keys. 

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate 
  - Feature gate name: ExternalServiceAccountTokenSigner
  - Components depending on the feature gate: kube-apiserver
- [x] Other
  - Describe the mechanism:
    - To enable:
      - kube-apiserver will need a restart with `--service-account-key-file`/`--service-account-signing-key-file` being un-set and `--service-account-signing-endpoint` being set.
      - former value of `--service-account-key-file` (a path on the file system) will need to be supplied to ExternalJWTSigner in some fashion(this is up to the owners of the distribution).
      - ExternalJWTSigner needs to combine the supported keys that are listed from external key management solutions and the keys that were on the cluster before enablement.
    - To disable:
      - kube-apiserver will need a restart with `--service-account-key-file`/`--service-account-signing-key-file` being set and `--service-account-signing-endpoint` being un-set.
      - Externally supported keys will need to get stitched into the set at `--service-account-key-file` (unless distributions want to stop supporting those keys completely).
  - Will enabling / disabling the feature require downtime of the control
    plane?
    - No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
    - No

###### Does enabling the feature change any default behavior?

Not from an end-user perspective. 

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

It will be possible but will need additional work from respective distributions. Thus, each distribution can decide for themselves.

Can be achieved by satisfying the following:
- kube-apiserver will need a restart with `--service-account-key-file`/`--service-account-signing-key-file` set and `--service-account-signing-endpoint` un-set.
- Distributions will require importing the externally supported keys to the file system path as configured in `--service-account-key-file`.
- If distributions do not intend to support tokens signed by external signers after the feature is disabled, then they can come up with a fair warning.

###### What happens if we re-enable the feature if it was previously rolled back?

experience would be the same as when enabling.

###### Are there any tests for feature enablement/disablement?

- Unit and integration tests will be added.
- The tests would **not** focus on continuing to support the same key set when enabling/disabling but would rather focus on viability of enabling/disabling.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

- Issues during rollout/rollback can disrupt jwt signing and authentication flow.
- During a rollout: 
  - If ExternalJWTSigner fails, kube-apiserver will never successfully start since it won't be able to get supported JWKs from ExternalJWTSigner.
  - All comms to kube-apiserver will be affected in this case.
- During a rollback:
  - A possible failure mode could be when externally supported keys are not imported and combined with supported on-disk keys.
  - This would cause any comms from workload to kube-apiserver using a jwt issued by ExternalJWTSigner to fail authentication.
  - Workloads will need to request a new jwt signed by on-disk keys to restore comms with kube-apiserver.

NOTE: Both, a rollout and a rollback needs kube-apiserver to restart with a config change. Any misconfiguration that prevents kube-apiserver from re-starting successfully will obviously effect all running workloads.

###### What specific metrics should inform a rollback?

1. High error rates on service account token creation.

   - Check for `apiserver_request_total`
   - Use following labels: resource="serviceaccounts", subresource="token"
   - Compute error rate:

     ````
      sum(rate(apiserver_request_total{job="kubernetes-apiservers",group="", version="v1",resource="serviceaccounts",subresource="token",code=~"5.."}[5m]))
     ````

2. Dramatic increase in reported 401 status code.

   - Check for `apiserver_request_total`
   - Use following labels: resource="serviceaccounts", subresource="token"
   - Compute error rate:

     ````
      sum(rate(apiserver_request_total{job="kubernetes-apiservers",group="", version="v1",code=~"401"}[5m]))
     ````


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be covered by integration tests.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The Feature would not be used by workload directly but will be used by kube-apiserver.

The usage should be visible to the operator via these metrics:

- apiserver_externaljwt_fetch_keys_data_timestamp
- apiserver_externaljwt_fetch_keys_request_total
- apiserver_externaljwt_fetch_keys_success_timestamp
- apiserver_externaljwt_request_duration_seconds
- apiserver_externaljwt_sign_request_total

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details:
    - The feature is not used by individual pods but the kube-apiserver itself. Simply successful Token requests and successful auth with kube-apiserver when using a bearer token in the request shall be indicators enough.
    - Initial read on kube-apiserver's `healthz` / `readyz` will be indicative of successful JWKs being fetched and thus, appropriately working ExternalSigner. 
    - Any subsequent issues on ExternalSigner will be observable via metrics.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

It is expected that external `Sign` request durations will be dominated by the external signer implementation.
Instrumenting processing time of your external signer implementation is recommended.

Experimentally, the gRPC overhead adds about 1ms to a TokenRequest, comparing the in-tree kube-apiserver
service account token signer with a stub external signer still doing local signing.

The `apiserver_externaljwt_request_duration_seconds{method=Sign,code=OK}` metrics
are expected to be within 1-10ms of the external signer processing time.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `apiserver_request_total` and `apiserver_request_duration_seconds`
    - Aggregation method:  aggregate over `job="kubernetes-apiservers",group="", version="v1",resource="serviceaccounts",subresource="token"`
    - Components exposing the metric: kube-apiserver
  - Metric name: `apiserver_externaljwt_sign_request_total`
    - Components exposing the metric: kube-apiserver
  - Metric name: `apiserver_externaljwt_request_duration_seconds`
    - Components exposing the metric: kube-apiserver

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

1. serviceaccount_jwks_freshness
2. serviceaccount_oidc_jwks_freshness
3. serviceaccount_jwks_fetch_requests_total

- 1 and 2 can both be gauge metrics. 
- Each will indicate how long it's been since the supported key set was last synced.
- 3 can be a cumulatively increasing metric with `status` that will also enable deriving error rate.

###### Other useful metrics that can assist with this feature

1. serviceaccount_stale_tokens_total
  - if `--service-account-extend-token-expiration` is set, then the token lifetimes in the cluster will be min(max token lifetime supported by ExternalJWTSigner, 1 year).
  - It is the integrator's responsibility to ensure that their ExternalJWTSigner implementation support signing tokens with 1 year validity i.e. if their clusters are relying on extended token lifetimes.
  - integrators can observe the `serviceaccount_stale_tokens_total` metric to confirm their cluster's reliance on `--service-account-extend-token-expiration`.

### Dependencies

One new dependency will be introduced and it will only be required for clusters configured/opted-in via the `--service-account-signing-endpoint` flag.

###### Does this feature depend on any specific services running in the cluster?

The feature depends on a cluster level service that will implement [ExternalJWTSigner RPC](#externaljwtsigner-rpc) and will serve on a UDS. Each distribution will have their own version, so let's just address it as `ExternalJWTSigner`.

- ExternalJWTSigner
    - Usage description:
      This service will act as a client to the external key management solution that a distribution will integrate with.

      NOTE: this being a configurable/opt-in feature, the impact will only be seen on clusters that are using the feature.

      - Impact of its outage on the feature:
        - kube-apiserver will not be able to sign new service account JWTs.
        - refreshing supported key sets will not be possible. Already held keys will continue to be supported unless kube-apiserver is restarted.
        - kube-apiserver is supposed to fetch JWKs everytime it sees a new kid that it does not recognize; Thus, every call that requires Auth will lead to an attempted refresh (single-flighted to prevent duplicate concurrent calls and rate-limited to no more than once a second).
      - Impact of its degraded performance or high-error rates on the feature:
        - service account token generation requests might require retries.
        - issues when syncing support key sets might cause intermediate auth failures only if there are changes in supported key sets.

### Scalability


###### Will enabling / using this feature result in any new API calls?

- Sign:
  - This will be a call from kube-apiserver to an external service account JWT signer.
  - The call will be made every time kube-apiserver receives a service account token request.
  - It's a critical path to create pods and keep them running. 
  - QPS will vary according to the cluster usage.
  - ExternalJWTSigner's throughput will vary from distribution to distribution.

- FetchKeys:
  - This is again a call from kube-apiserver to the external service account JWT signer.
  - It will be a periodic call to keep the OIDC provider in sync with the supported keys; Frequency will be decided by `refresh_hint_seconds` returned by an external signer. 
  - It will also be a dynamic call whenever kube-apiserver comes across a token signed by a key_id that it does not recognize.
  - QPS will be rather low since:
    - the signing keys shall not change very frequently.
    - even the smallest value of `refresh_hint_seconds` will result in at-most 1 qps.
  - ExternalJWTSigner's throughput will vary from distribution to distribution.

###### Will enabling / using this feature result in introducing new API types?

No new K8s api.

###### Will enabling / using this feature result in any new calls to the cloud provider?

- Create token
  - All service account token requests can result in a call to a cloud-provider.
  - It might be less often(or not at all) depending on the implementation of `ExternalJWTSigner` in each individual distribution.

- Listing keys
  - There can be periodic calls from implementation of `ExternalJWTSigner` to cloud providers for syncing supported signing keys.
  - The volume of calls will depend on implementation of `ExternalJWTSigner` in each individual distribution.  

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?


Service account JWT signing will be delegated to external signers. So, `create token` calls which shall otherwise be compliant with [api_call_latency](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/api_call_latency.md) SLO will now have a variable SLO that will be dictated by individual distribution's dependency(external signer).


###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

In the context of dimensions specified in [supported limits], there will be 1 additional process (or pod) per control plane node. Additional resource usage will be subject to respective implementation. The increase should be negligible.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

not likely.

### Troubleshooting

Symptom: kube-apiserver will not start with `--service-account-signing-endpoint` set

- check the kube-apiserver log for details about why startup failed
- ensure the socket `--service-account-signing-endpoint` points to is valid,
  the kube-apiserver user has permissions to access it, and the external signer is running
- ensure `--service-account-signing-key-file` and `--service-account-key-file` are not also set
- ensure the external signer supports the version of the externaljwt gRPC API kube-apiserver is using
- ensure the maximum supported token lifetime returned by the external signer does not conflict with any
  `--service-account-max-token-expiration` flag (the flag may not be longer than the max expiration supported by the external signer)

Symptom: token creation fails with `500` errors

- check `apiserver_externaljwt_sign_request_total` metrics for codes other than `OK` to determine if signing failures are the cause
- if signing requests are failing with `CANCELLED` or `DEADLINE_EXCEEDED` codes,
  check `apiserver_externaljwt_request_duration_seconds` metrics for timing distribution
  of external signing requests with `method=Sign`. If external signing is causing request timeouts,
  investigate improving the performance of your external signer integration.
- check the kube-apiserver log for details about other signing failures

Symptom: token use fails with authentication errors

- check the `apiserver_externaljwt_fetch_keys_request_total` metrics for codes other than `OK`
  to determine if verifying keys are failing to be fetched
- check the `apiserver_externaljwt_fetch_keys_success_timestamp` metric to determine the 
  last time public keys were successfully refreshed. If this exceeds the expected `refresh_hint_seconds`
  value for your particular external signer integration, check `kube-apiserver` logs for details on why
  the public key fetch is failing.
- check the `apiserver_externaljwt_fetch_keys_data_timestamp` metric to determine the `data_timestamp`
  reported by the external signer in the last successful fetch of public keys. Compare to the expected
  value for your particular external signer integration to determine if `kube-apiserver` is using current
  public keys. If this does not match, check your external signer for details on why it is not returning
  the expected public keys to the `FetchKeys` method.

###### How does this feature react if the API server and/or etcd is unavailable?

feature is only accessible via kube-apiserver. JWT signing and authentication will anyways not work without kube-apiserver.

###### What are other known failure modes?

Covered above in the troubleshooting section.

###### What steps should be taken if SLOs are not being met to determine the problem?

The improvement will likely need to happen on respective cloud-providers API. Confirming if it's the source of the problem can be done by observing cloud-provider's metrics.

## Implementation History

Initial PRs: 
- kubernetes/kubernetes#73110
- kubernetes/kubernetes#125177

1.32: Alpha release
- kubernetes/kubernetes#128190
- kubernetes/kubernetes#128192
- kubernetes/kubernetes#128953

1.34: Beta release
- kubernetes/kubernetes#131300

1.36: Stable release
- kubernetes/kubernetes#136118

## Drawbacks

Enabling the feature puts a remote service in the critical path of kube-apiserver. Thus, it can easily cause an outage. However, we have some relief in that it is an opt-in/configurable feature. 

## Alternatives

N/A

## Infrastructure Needed (Optional)

N/A

## Possible Future Work

- External signer, with its Sign() rcp, falls in the critical path for pod creation.
- Thus, external signers can impact the availability of JWT signing via kube-apiserver and thus pod creation.
- This can be remediated if we add x5c verification support.
  - In this scheme, a leaf cert(typically short lived) certified by previous cert in a chain of certificates is used to sign the JWT. 
  - The verification is done against the public key held in the Root cert.
  - However, it comes with tradeoffs:
    - With x5c support, the system will be better Available.
      - In our case, ExternalSigner will return a short lived cert that kube-apiserver will hold and use to mint tokens.
      - In case of an outage on ExternalSigner, kube-apiserver will continue to be able to mint tokens for the lifetime of the leaf cert it holds.
    - Without x5c support, the system will be more secure.
      - No leaf cert which can mint tokens.
      - Each token issue is auditable externally.
      - Signing is truly external.
      - Surface area of the token is identical to the existing implementation.
