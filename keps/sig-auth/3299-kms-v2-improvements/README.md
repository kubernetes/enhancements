# KEP-3299: KMS v2 Improvements

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [v2 API](#v2-api)
  - [Key Hierarchy](#key-hierarchy)
  - [Metadata](#metadata)
  - [Status API](#status-api)
  - [Rotation](#rotation)
  - [Observability](#observability)
  - [Sequence Diagram](#sequence-diagram)
    - [Encrypt Request](#encrypt-request)
    - [Decrypt Request](#decrypt-request)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
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

This KEP proposes the new v2alpha1 `KeyManagementService` service contract to:
- enable fully automated key rotation for the latest key
- improve KMS plugin health check reliability
- improve observability of envelop operations between kube-apiserver, KMS plugins and KMS

It further proposes a SIG-Auth maintained KMS plugin reference implementation. This implementation will support a key hierarchy design that implements the v2alpha1 API and will serve as a baseline that provides:
- improve readiness times for clusters with a large number of encrypted resources
- reduce the likelihood of hitting the external KMS request rate limit
- metrics and tracing support

## Motivation

**Performance**: Today, when the kube-apiserver is restarted in a cluster and a LIST secret call is made (this applies to all resources encrypted at rest, which secrets tend to always be part of), due to the serial processing of LIST requests and the data encryption key (DEK) cache being empty, the initialization of informers may take significant time as a result of the large number of consecutive trips to the KMS plugin -> external KMS for all the DEKs that have been generated so far. This serial call can cause the KMS plugin to hit the external KMS rate limit and delay the overall readiness of the cluster. Currently, a DEK is generated for each object and is then encrypted using a KEK. This 1:1 mapping means if there is a burst of secret creation, then the KMS plugin can also hit the external KMS rate limit for encrypt operations.

**Rotation**: Currently, it requires lots of manual steps to [rotate a KMS key for Kubernetes](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/#rotating-a-decryption-key) and the process is error prone. It requires deployment of another instance of the KMS plugin with the new key running side by side with the old instance while adding a second entry of the new plugin to `EncryptionConfiguration`. Any change to the `EncryptionConfiguration` requires a kube-apiserver restart for the changes to take effect. For a single kube-apiserver configuration, this can lead to a brief period when the kube-apiserver is unavailable. The current rotation process requires multiple restarts of all kube-apiserver processes to ensure each server can decrypt and then encrypt using the new key. It requires multiple updates to the `EncryptionConfiguration`  to move the new key to the second and then first entry in the keys array so that it is used for encryption in the config. It also requires running storage migration (either via the storage version migrator or a manual invocation of `kubectl get secrets --all-namespaces -o json | kubectl replace -f - `) to encrypt all existing Secrets with the new key, which can timeout and leave the cluster in a state where it is still dependent on the old key.

**Health Check & Status**: Today, the health check from kube-apiserver to KMS plugin is an `Encrypt` operation followed by `Decrypt` operation. These operations cost money in cloud environments and do not allow the plugin to perform more holistic checks on if it is healthy. Furthermore, a plugin has no way to inform the API server if its underlying key encryption key (KEK) has been rotated. If we provide a separate status RPC call with its own `StatusRequest` and `StatusResponse`, the KMS plugin can indicate the change in KEK version as part of response. This could be an indication that the KEK is now rotated and storage migration is now required.

**Observability**: The only way to correlate a successful/failed envelope operation today is to use the approximate timestamp of the operation to check events in kube-apiserver, kms-plugin and KMS. There is no guarantee that the timestamp of the operation is the same as the timestamp of the corresponding event in KMS. This KEP proposes extending the signature of the kms-plugin interface to include the transaction ID (to be generated by the kube-apiserver), which kms-plugin could pass to KMS. This transaction ID will be logged in the kube-apiserver with additional metadata such as secret name and namespace for the envelope operation. Similarly, the transaction ID will be logged in the kms-plugin and optionally passed to KMS.

### Goals
- improve readiness times for clusters with a large number of encrypted resources
- reduce the likelihood of hitting the KMS rate limit
- enable fully automated key rotation for the latest key
- improve KMS plugin health check reliability
- improve observability of envelop operations between kube-apiserver, KMS plugins and KMS
- if this v2 API reaches GA in release N, the existing v1beta1 gRPC API will be deprecated at release N and removed at release N+3 (the existing key rotation dance of using multiple providers will be used to migrate from v1beta1 to v2)

### Non-Goals
- Prevent KMS rate limiting
- Recovery when KMS KEK is deleted
- Using the proposed transaction ID for audit logging

## Proposal

Performance, Health Check, Observability and Rotation:
- Support key hierarchy in KMS plugin that generates local KEK
- Expand `EncryptionConfiguration` to support a new KMSv2 configuration
- Add v2alpha1 `KeyManagementService` proto service contract in Kubernetes to include
    - `key_id` and additional metadata in `annotations` to support key rotation
    - `key_id`: the KMS Key ID, stable identifier, changed to trigger key rotation and storage migration
    - `annotations`: structured data, can contain the encrypted local KEK, can be used for debugging, recovery, opaque to API server, stored unencrypted, etc. Validation similar to how K8s labels are validated today. Labels have good size limits and restrictions today.
    - A status request and response periodically (order of minutes) returns `version`, `healthz`, and `key_id`
    - The `key_id` in status can be used on decrypt operations to compare and validate the key ID stored in the DEK cache and the latest `EncryptResponse` `key_id` to detect if an object is stale in terms of storage migration
    - Generate a new UID for each envelope operation in kube-apiserver
    - Add a new UID field to `EncryptRequest` and `DecryptRequest`
- Add support for hot reload of the `EncryptionConfiguration`:
    - Watch on the `EncryptionConfiguration`
    - When changes are detected, process the `EncryptionConfiguration` resource, and add new transformers and update existing ones atomically.
    - If there is an issue with creating or updating any of the transformers, retain the current configuration in the kube-apiserver and generate an error in logs.
- Enable fully automated rotation for `latest` key in KMS:
    > NOTE: Prerequisite: `EncryptionConfiguration` is set up to always use the `latest` key version in KMS and the values can be interpreted dynamically at runtime by the KMS plugin to hot reload the current write key. Rotation process sequence:
    - record initial key ID across all API servers
    - cause key rotation in KMS (user action in the remote KMS)
    - observe the change across the stack
    - storage migration (run storage migrator)

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### v2 API

`EncryptionConfiguration` will be expanded to support the new v2 API:

```diff
​​diff --git a/staging/src/k8s.io/apiserver/pkg/apis/config/v1/types.go b/staging/src/k8s.io/apiserver/pkg/apis/config/v1/types.go
index d7d68d2584d..84c1fa6546f 100644
--- a/staging/src/k8s.io/apiserver/pkg/apis/config/v1/types.go
+++ b/staging/src/k8s.io/apiserver/pkg/apis/config/v1/types.go
@@ -98,3 +99,10 @@ type KMSConfiguration struct {
+    // apiversion of KeyManagementService
+    APIVersion string `json:"apiVersion"`
```

Support key hierarchy in KMS plugin that generates local KEK and add v2alpha1 `KeyManagementService` proto service contract in Kubernetes to include `key_id`, `annotations`, and `status`. 

### Key Hierarchy

Key Hierarchy in KMS plugin (reference implementation):

1. No changes to the API server, keep 1:1 DEK mapping
    1. Assumption: A KMS plugin that was implemented using a local HSM would not need any changes because it would be able to handle the amount of encryption calls with ease since it would not need to perform network IO
    1. Assumption: local gRPC calls to the KMS plugin do not represent significant overhead
1. KMS plugin generates its own local KEK in-memory
1. External KMS is used to encrypt the local KEK
1. Local KEK is used for encryption of DEKs sent by API server
1. Local KEK is used for encryption based on policy (N events, X time, etc)

Since key hierarchy is implemented at the KMS plugin level, it should be seamless for the kube-apiserver. So whether the plugin is using a key hierarchy or not, the kube-apiserver should behave the same.

### Metadata

What is required of the kube-apiserver is to be able to tell the KMS plugin which KEK (local KEK or KMS KEK) it should use to decrypt the incoming DEK. To do so, upon encryption, the KMS plugin could provide the encrypted local KEK as part of the `annotations` field in the `EncryptResponse`. The kube-apiserver would then store it in etcd next to the DEK. Upon decryption, the kube-apiserver provides the encrypted local KEK in `annotations` and `key_id` from the last encryption when calling Decrypt. In case no encrypted local KEK is provided in the `annotations`, then we can assume key hierarchy is not used. The KMS plugin would query the external KMS to use the remote KEK to decrypt the DEK (same behavior as today). No state coordination is required between different instances of the KMS plugin.

For the reference KMS plugin, the encrypted local KEK is stored in etcd via the `annotations` field, and once decrypted, it can be stored in memory as part of the KMS plugin cache to be used for encryption and decryption of DEKs. The encrypted local KEK is used as the key and the decrypted local KEK is stored as the value.

```proto
message EncryptResponse {
    // The encrypted data.
    bytes ciphertext = 1;
    // The KMS key ID used to encrypt the data. This must always refer to the KMS KEK and not any local KEKs that may be in use.
    // This can be used to inform staleness of data updated via value.Transformer.TransformFromStorage.
    string key_id = 2;
    // Additional metadata to be stored with the encrypted data.
    // This metadata can contain the encrypted local KEK that was used to encrypt the DEK.
    // This data is stored in plaintext in etcd. KMS plugin implementations are responsible for pre-encrypting any sensitive data.
    map<string, bytes> annotations = 3;
}
```

The `DecryptRequest` passes the same `key_id` and `annotations` returned by the previous `EncryptResponse` of this data as its `key_id` and `annotations` for the decryption request.

```proto
message DecryptRequest {
    // The data to be decrypted.
    bytes ciphertext = 1;
    // UID is a unique identifier for the request.
    string uid = 2;
    // The keyID that was provided to the apiserver during encryption.
    // This represents the KMS KEK that was used to encrypt the data.
    string key_id = 3;
    // Additional metadata that was sent by the KMS plugin during encryption.
    map<string, bytes> annotations = 4;
}

message DecryptResponse {
    // The decrypted data.
    bytes plaintext = 1;
}

message EncryptRequest {
    // The data to be encrypted.
    bytes plaintext = 1;
    // UID is a unique identifier for the request.
    string uid = 2;
}
```

In terms of storage, a new structured protobuf format is proposed. The prefix for the new format is `k8s:enc:kms:v2:<config name>:`.

```go
// EncryptedObject is the representation of data stored in etcd after envelope encryption.
type EncryptedObject struct {
	// EncryptedData is the encrypted data.
	EncryptedData []byte `protobuf:"bytes,1,opt,name=encryptedData,proto3" json:"encryptedData,omitempty"`
	// KeyID is the KMS key ID used for encryption operations.
	KeyID string `protobuf:"bytes,2,opt,name=keyID,proto3" json:"keyID,omitempty"`
	// EncryptedDEK is the encrypted DEK.
	EncryptedDEK []byte `protobuf:"bytes,3,opt,name=encryptedDEK,proto3" json:"encryptedDEK,omitempty"`
	// Annotations is additional metadata that was provided by the KMS plugin.
	Annotations          map[string][]byte `protobuf:"bytes,4,rep,name=annotations,proto3" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}
```

This object simply provides a structured format to store the `EncryptResponse` data with the plugin name and encrypted object data. New fields can easily be added to this format.

### Status API

To improve health check reliability, the new StatusResponse provides version, healthz information, and can trigger key rotation via encryption state status updates.

```proto
message StatusRequest {}

message StatusResponse {
    // Version of the KMS plugin API.  Must match the configured .resources[].providers[].kms.apiVersion
    string version = 1;
    // Any value other than "ok" is failing healthz.  On failure, the associated API server healthz endpoint will contain this value as part of the error message.
    string healthz = 2;
    // the current write key, used to determine staleness of data updated via value.Transformer.TransformFromStorage.
    string key_id = 3;
}
```

### Rotation

The `key_id` will be funneled into the status of the new encryption state REST API.  Each API server will create this resource (via a controller that will likely run as a post start hook) with a random name on startup (stale objects will be garbage collected by a new controller that runs in the Kubernetes controller manager).  This resource provides the necessary information for a controller to automatically initiate storage migration when consensus has been reached among the different API server instances.  Opaque hashes are used to prevent leaking any specific information about the encryption configuration and the key IDs in use.  This is a cluster scoped REST API that is only meant to be read by the associated controllers and the cluster admin.

**Question: should writes to this resource be limited to the API server loopback identity?**

**Question: should we have a status field with the Kubernetes version to prevent migration during differences in API server version?**

```go
// metadata.name is the ID of the API server
type EncryptionState struct {
    metav1.TypeMeta `json:",inline"`
    // +optional
    metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

    Spec EncryptionStateSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`

    Status EncryptionStateStatus `json:"status" protobuf:"bytes,3,opt,name=status"`
}

// spec is empty
type EncryptionStateSpec struct{}

// status will be updated every 1 minute
// KCM garbage collector will delete objects that have not been updated within the last 15 minutes
type EncryptionStateStatus struct {
	// running a storage migration when the API servers do not agree on the
	// desired state will result in the etcd data being in an undefined state
    EncryptionConfigurationHash Hash `json:"encryptionConfigurationHash" protobuf:"bytes,1,opt,name=encryptionConfigurationHash"`

	// all resources that are encrypted are represented as idividual items in this list
	// note that this may get problamtic if we support */*
    Resources []EncryptionStateResource `json:"resources" protobuf:"bytes,2,opt,name=resources"`

	// the last time any status hash was updated
	LastUpdated metav1.Time `json:"lastUpdated" protobuf:"bytes,3,opt,name=lastUpdated"`
}

type EncryptionStateResource struct {
    Resource metav1.GroupResource `json:"resource" protobuf:"bytes,1,opt,name=resource"`

	// an empty string from the hash value with a non-empty updated time means that
	// a non-KMS v2 write key is in use.  This can be used with the `EncryptionConfigurationHash`
	// to determine when the API servers are at the same state.
    WriteKeyIDHash Hash `json:"writeKeyIDHash" protobuf:"bytes,2,opt,name=writeKeyIDHash"`

	// note that this field does not cover KMS v1 and other legacy encryption modes that
	// are set as read keys.  This is okay because those configurations are statically
	// covered by the `EncryptionConfigurationHash`.  KMS v2 can dynamically change keys
	// at runtime without any change to the `EncryptionConfigurationHash` and thus needs
	// its own hash tracking mechanism.
    ReadKeyIDHashes []Hash `json:"readKeyIDHashes" protobuf:"bytes,3,opt,name=readKeyIDHashes"`
}

type Hash struct {
	// opaque string, likely base64(SHA-256)
    Value       string      `json:"value" protobuf:"bytes,1,opt,name=value"`
    LastUpdated metav1.Time `json:"lastUpdated" protobuf:"bytes,2,opt,name=lastUpdated"`
}
```

A controller would run the following steps to perform a storage migration (in conjunction with `kube-storage-version-migrator`):

1. Check (likely hard-coded) config for expected number of API servers `N`
2. Wait for `N` `EncryptionState` objects to exist
3. Wait for all objects to have a `.status.lastUpdated` that is within `5` minutes of `time.Now()`
4. Wait for all status hashes to be in sync
5. Generate a hash `X` that is deterministically computed based on all status hashes
6. Create one `migration.k8s.io/v1alpha1` `StorageVersionMigration` object per group resource (if it does not exist)
	1. Set `.metadata.name` to `base64(X).group.resource`
	2. Set `.spec.resource.version` to the preferred version based on discovery
	3. (Unclear, maybe) Set `.metadata.ownerReferences` to include all `N` `EncryptionState` objects
7. Wait for `.status.conditions[.type = Succeeded].status` `==` `true`

### Observability

To improve observability, this design also generates a new `UID` for each envelope operation similar to `UID` generation in admission review requests here: https://github.com/kubernetes/kubernetes/blob/e9e669aa6037c380469b45200e59cff9b52d6d68/staging/src/k8s.io/apiserver/pkg/admission/plugin/webhook/request/admissionreview.go#L137.

This `UID` field is included in the `EncryptRequest` and `DecryptRequest` of the v2 API.  It will always be present. It is generated in the kube-apiserver and will be used:

1. For logging in the kube-apiserver. All envelope operations to the kms-plugin will be logged with the corresponding `UID`.
   1. The `UID` will be logged using a wrapper in the kube-apiserver to ensure that the `UID` is logged in the same format and is always logged.
   2. In addition to the `UID`, the kube-apiserver will also log non-sensitive metadata such as `name`, `namespace` and `GroupVersionResource` of the object that triggered the envelope operation.
2. Sent to the kms-plugin as part of the `EncryptRequest` and `DecryptRequest` structs.

### Sequence Diagram

#### Encrypt Request

```mermaid
sequenceDiagram
    participant etcd
    participant kubeapiserver
    participant kmsplugin
    participant externalkms
    kubeapiserver->>kmsplugin: encrypt request
    alt using key hierarchy
        kmsplugin->>kmsplugin: encrypt DEK with local KEK
        kmsplugin->>externalkms: encrypt local KEK with remote KEK
        externalkms->>kmsplugin: encrypted local KEK
        kmsplugin->>kmsplugin: cache encrypted local KEK
        kmsplugin->>kubeapiserver: return encrypt response <br/> {"ciphertext": "<encrypted DEK>", key_id: "<remote KEK ID>", <br/> "annotations": {"kms.kubernetes.io/local-kek": "<encrypted local KEK>"}}
    else not using key hierarchy
        %% current behavior
        kmsplugin->>externalkms: encrypt DEK with remote KEK
        externalkms->>kmsplugin: encrypted DEK
        kmsplugin->>kubeapiserver: return encrypt response <br/> {"ciphertext": "<encrypted DEK>", key_id: "<remote KEK ID>", "annotations": {}}
    end
    kubeapiserver->>etcd: store encrypt response and encrypted DEK
```

#### Decrypt Request

```mermaid
sequenceDiagram
    participant kubeapiserver
    participant kmsplugin
    participant externalkms
    %% if local KEK in annotations, then using hierarchy
    alt encrypted local KEK is in annotations
      kubeapiserver->>kmsplugin: decrypt request <br/> {"ciphertext": "<encrypted DEK>", key_id: "<key_id gotten as part of EncryptResponse>", <br/> "annotations": {"kms.kubernetes.io/local-kek": "<encrypted local KEK>"}}
        alt encrypted local KEK in cache
            kmsplugin->>kmsplugin: decrypt DEK with local KEK
        else encrypted local KEK not in cache
            kmsplugin->>externalkms: decrypt local KEK with remote KEK
            externalkms->>kmsplugin: decrypted local KEK
            kmsplugin->>kmsplugin: decrypt DEK with local KEK
            kmsplugin->>kmsplugin: cache decrypted local KEK
        end
        kmsplugin->>kubeapiserver: return decrypt response <br/> {"plaintext": "<decrypted DEK>", key_id: "<remote KEK ID>", <br/> "annotations": {"kms.kubernetes.io/local-kek": "<encrypted local KEK>"}}
    else encrypted local KEK is not in annotations
        kubeapiserver->>kmsplugin: decrypt request <br/> {"ciphertext": "<encrypted DEK>", key_id: "<key_id gotten as part of EncryptResponse>", <br/> "annotations": {}}
        kmsplugin->>externalkms: decrypt DEK with remote KEK (same behavior as today)
        externalkms->>kmsplugin: decrypted DEK
        kmsplugin->>kubeapiserver: return decrypt response <br/> {"plaintext": "<decrypted DEK>", key_id: "<remote KEK ID>", <br/> "annotations": {}}
    end
```

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

This section is incomplete and will be updated before the beta milestone.

##### Unit tests

This section is incomplete and will be updated before the beta milestone.

##### Integration tests

This section is incomplete and will be updated before the beta milestone.

##### e2e tests

This section is incomplete and will be updated before the beta milestone.


### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial unit and integration tests completed and enabled

#### Beta

TBD

#### GA

TBD

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->

- Feature gate
  - Feature gate name: `KMSv2`
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager

```go
FeatureSpec{
	Default: false,
	LockToDefault: false,
	PreRelease: featuregate.Alpha,
}
```

###### Does enabling the feature change any default behavior?

No. The v2 API is new in the v1.25 release.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, To disable encryption at rest using the v2 API:
1. Disable encryption at rest with KMS provider by running through these [steps](https://kubernetes.io/docs/tasks/administer-cluster/kms-provider/#disabling-encryption-at-rest)
   1. At the end of this step, all the data in etcd will be unencrypted.
2. Disable the `KMSv2` feature gate.

Disabling this gate without first doing a storage migration to use a different encryption at rest mechanism will result in data loss.

Once the feature gate is disabled, if the plan is to use a different encryption at rest mechanism instead of KMS, then unset the `--encryption-provider-config` flag on the kube-apiserver.

###### What happens if we reenable the feature if it was previously rolled back?

After the feature is reenabled, if a v2 KMS provider is still configured in the `EncryptionConfiguration`
- All new data will be encrypted with the external kms provider.
- Existing data can be decrypted if the key used for encryption before feature rollback still exists.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

- When the feature is disabled, data stored in etcd will no longer be encrypted using the external kms provider with v2 API.
- If the feature is disabled incorrectly (i.e without performing a storage migration), existing data that is encrypted with the external kms provider will be unable to be decrypted. This will cause list and get operations to fail for the resources that were encrypted.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

This section is incomplete and will be updated before the beta milestone.

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

- If a rollback of the feature is done without first doing a storage migration to use a different encryption at rest mechanism will result in data loss.
  - Workloads relying on existing data in etcd will no longer be able to access it.
  - The data can be retrieved by reenabling the feature gate or deleting and recreating the data.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

This section is incomplete and will be updated before the beta milestone.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

This section is incomplete and will be updated before the beta milestone.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

N/A

### Monitoring Requirements

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details:
    - Logs in kube-apiserver, kms-plugin and KMS will be logged with the corresponding `key_id`, `annotations`, and `UID`.
    - count of encryption/decryption requests by resource and version

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

There should be no impact on the SLO with this change.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Other (treat as last resort)
  - Details:
    - Logs in kube-apiserver, kms-plugin and KMS will be logged with the corresponding `key_id`, `annotations`, and `UID`.
    - Metrics for latency of encryption/decryption requests.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes, the new KMS v2 gRPC API.

###### Will enabling / using this feature result in introducing new API types?

Yes, the new KMS v2 gRPC types.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No, the v2 API is new.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

- This feature is part of API server. The feature is unavailable if API server is unavailable. ETCD data encryption with external kms-plugin will be unavailable.
- If the API server is unavailable, clients will be unable to create/get data that's stored in etcd. There will be no requests from the API server to the kms-plugin.
- If the `EncryptionConfiguration` file configured in the control plane node is not valid:
  - API server when restarted will fail at startup as it's unable to load the EncryptionConfig. This behavior is consistent with the KMS v1 API. The encryption configuration needs to be fixed to allow the API server to start properly.
- If the KMS plugin is unavailable:
  - API server when restarted will fail health check as it's unable to connect to the KMS plugin. The `/livez` and `/readyz` endpoints will show a `failed` health check for the kms provider. This behavior is consistent with the KMS v1 API. Refer to [docs](https://kubernetes.io/docs/reference/using-api/health-checks/) for the health API endpoints and how to exclude individual endpoints from causing the API server to fail health check.
  - To resolve the issue, the kms plugin must be fixed to be available. The logs in the kms-plugin should be indicative of the issue.

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
**Performance and rotation:**

We considered the follow approaches and each has its own drawbacks:
1. `cacheSize` field in `EncryptionConfiguration`. It is used by the API server to initialize a LRU cache of the given size with the encrypted ciphertext used as index. Having a higher value for the `cacheSize` will prevent calls to the plugin for decryption operations. However, this does not solve the issue with the number of calls to KMS plugin when encryption traffic is bursty.
2. Reduce the number of trips to KMS by caching DEKs by allowing one DEK to be used to encrypt multiple objects within the configured TTL period. One issue with this approach is it will be very hard to inform the API server to rotate the DEKs when a KEK has been rotated. 

**Observability**:

We considered using the `AuditID` from the kube-apiserver request that generated the envelope operation. This approach has the following drawbacks:

1. `AuditID` can be configured by the user with the `Audit-ID` header in the API server request. Multiple requests can be sent to the kube-apiserver with the same `Audit-ID`.
2. Not all API server requests will generate an envelope operation. The API server caches DEKs and for the DEK that's available in the cache, the kube-apiserver will not generate an envelope operation.
3. Since not all calls to the KMS correspond to an audit log, using audit ID is not complete for correlating calls from kube-apiserver->kms-plugin->KMS.

**Rotation**:

Since the storage format is structured and contains plaintext key ID information, the `EncryptionState` resource could include information about the current state of etcd:

```go
	// Reminder: this needs a way to indicate that non-kms v2 encryption methods are in use as well as when unencrypted data is present.
    // this field would be populated by scanning all KMS v2 encrypted data and recording all observed
    // key IDs.  this would be an expensive operation and is a bit weird to be per API server.
    InUseKeyIDHashes []Hash `json:"inUseKeyIDHashes" protobuf:"bytes,4,opt,name=inUseKeyIDHashes"`
```

This API has been deferred as it is not explicitly required for rotation.  It is still something that should be considered as it would provide an easy way for a user to confirm that storage migration was successful (for example, it allows a user to understand that an old KMS key can be safely deleted).

## Infrastructure Needed

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
We need a new git repo for the KMS plugin reference implementation. It will need to be synced from the k/k staging dir.
