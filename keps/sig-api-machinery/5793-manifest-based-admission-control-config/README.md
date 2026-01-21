# KEP-5793: Manifest Based Admission Control Config

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Supported Resource Types](#supported-resource-types)
  - [User Stories](#user-stories)
    - [Story 1: Platform Invariants](#story-1-platform-invariants)
    - [Story 2: Self-Protection and Preventing Name Collisions](#story-2-self-protection-and-preventing-name-collisions)
    - [Story 3: Bootstrapping Cluster Security](#story-3-bootstrapping-cluster-security)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [New AdmissionConfiguration Schema](#new-admissionconfiguration-schema)
  - [Manifest File Format](#manifest-file-format)
  - [Naming and Conflict Resolution](#naming-and-conflict-resolution)
  - [File Watching and Dynamic Reloading](#file-watching-and-dynamic-reloading)
  - [Decoding, Defaulting, and Validation](#decoding-defaulting-and-validation)
  - [Metrics and Audit Annotations](#metrics-and-audit-annotations)
  - [Implementation](#implementation)
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
  - [Deny policies in RBAC](#deny-policies-in-rbac)
  - [Static admission plugins](#static-admission-plugins)
  - [External configuration management](#external-configuration-management)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes adding file-based manifests to the kube-apiserver to configure admission webhooks
and policies on startup. These policies would exist outside of the Kubernetes API, enabling operators
and platforms to implement admission controls that:

1. Are guaranteed to be active before the API server begins processing requests
2. Cannot be bypassed or modified through the Kubernetes API
3. Can protect API-based admission control resources themselves (ValidatingAdmissionPolicy,
   MutatingAdmissionPolicy, ValidatingWebhookConfiguration, MutatingWebhookConfiguration, etc.)

This is achieved by augmenting the `AdmissionConfiguration` schema to include paths to manifest files
containing webhook and policy configurations that are loaded at API server startup and watched for
changes at runtime.

## Motivation

Today, most policy enforcement in Kubernetes is implemented through:
- `MutatingAdmissionWebhook` and `ValidatingAdmissionWebhook` plugins using webhook configurations
- `ValidatingAdmissionPolicy` (VAP) and `MutatingAdmissionPolicy` (MAP) for CEL-based policies

These admission controls are registered by creating API objects (`MutatingWebhookConfiguration`,
`ValidatingWebhookConfiguration`, `ValidatingAdmissionPolicy`, `MutatingAdmissionPolicy`, and their
binding resources). This creates several gaps:

1. Bootstrap gap: Policy enforcement is not active until these objects are created and picked up
   by the dynamic admission controller. This creates a window between API server startup and webhook
   registration where policies are not enforced.

2. Self-protection gap: Cluster administrators cannot protect webhook and policy configurations
   from deletion or modification, as these objects are not themselves subject to webhook admission
   (to prevent circular dependencies). A malicious or misconfigured actor with sufficient privileges
   can delete critical admission policies.

3. Etcd dependency: Current admission configurations depend on etcd availability. If etcd is
   unavailable or corrupted, admission policies may not be loaded correctly.

This KEP aims to address these issues by providing a file-based mechanism for configuring admission
controls that operates independently of the Kubernetes API.

### Goals

1. Guarantee enforcement from startup: File-configured admission policies and webhooks MUST be
   active before the API server begins processing requests. There must be no gap during startup
   when requests are handled but admission controls are not yet active.

2. Isolated universe: Manifest-based admission control exists in a tightly scoped and
   isolated universe. It may not reference API resources, nor vice-versa. This means no paramKind
   support, no service references, and no dynamic credentials. The manifest-based admission
   control objects will not be exposed as REST API visible API objects.

3. Enable platform-level protection: Manifest-based admission control can intercept and enforce
   policies on API-based admission control resources (VAP/MAP/VAPB/MAPB/VWC/MWC), providing a
   mechanism for platform operators to protect critical infrastructure.

4. Support dynamic updates: Manifest-based admission control files MAY be updated at runtime.
   The kube-apiserver will watch for file changes and reload configurations when files change and
   are validated successfully. Such changes are eventually consistent and observable via metrics.

5. Provide clear observability: Metrics and audit annotations MUST clearly distinguish between
   manifest-based and API-based admission decisions.

### Non-Goals

1. Cross-apiserver synchronization: Synchronization of file-based object information across
   API servers will not be implemented as part of this KEP.
   For control planes running multiple instances of the API server, each API server must be
   configured individually by external means.
   This is similar to how other file-based configurations (e.g., encryption configuration)
   work today.

2. API-dependent references: Manifest-based admission control objects may not depend on the rest API.
    1. Param objects for policies: No support for ValidatingAdmissionPolicy/MutatingAdmissionPolicy
    `paramKind` references. Policies configured via manifest cannot reference ConfigMaps or other
    cluster objects for parameters.
    2. Service references in webhooks: Only URL-based webhook endpoints are supported. Service
    references (`clientConfig.service`) are not supported because the service network may not be
    available at API server startup.
    3. Credentials for webhooks: Webhooks will only use statically configured credentials
   (e.g., `kubeConfigFile`). Service account credentials, cluster trust bundles, or other
   API-fetched credentials are not supported. Credentials that e.g. refer to an external
   OAuth endpoint are permitted.

3. API visibility: Manifest-based admission control objects are not visible through the
   Kubernetes API. These objects cannot be controlled through the API by design, may not be
   synchronized between API servers, and exposing them (similar to mirror pods) has proven
   error-prone in practice.

## Proposal

This proposal augments the `AdmissionConfiguration` resource (used with `--admission-control-config-file`)
to include paths to manifest files containing admission configurations. These manifests are loaded at
API server startup and watched for changes at runtime.

For prior art, see
[static pods](https://kubernetes.io/docs/tasks/configure-pod-container/static-pod/).
This proposal is both slightly simpler (no analogue to mirror pods) and slightly more complex
(more objects / configuration), see Design Details below.

### Supported Resource Types

The following resource types are supported in manifest files. Only the v1 API version is supported
for each type. Each admission plugin's manifest file/directory must only contain the types allowed
for that plugin (e.g., if you want to use manifests for all four admission plugins, you need four
separate manifest files or directories).

Webhooks:
- `admissionregistration.k8s.io/v1.ValidatingWebhookConfiguration`
- `admissionregistration.k8s.io/v1.MutatingWebhookConfiguration`
- `admissionregistration.k8s.io/v1.ValidatingWebhookConfigurationList`
- `admissionregistration.k8s.io/v1.MutatingWebhookConfigurationList`

CEL-based policies:
- `admissionregistration.k8s.io/v1.ValidatingAdmissionPolicy`
- `admissionregistration.k8s.io/v1.ValidatingAdmissionPolicyBinding`
- `admissionregistration.k8s.io/v1.MutatingAdmissionPolicy` (requires MAP to be at v1)
- `admissionregistration.k8s.io/v1.MutatingAdmissionPolicyBinding` (requires MAP to be at v1)

Note: [MutatingAdmissionPolicy] (MAP) is at v1beta1 as of Kubernetes 1.35 and is targeting GA in 1.36.

Generic lists:
- `v1.List` containing any of the above types

### User Stories

#### Story 1: Platform Invariants

As a platform administrator managing multiple Kubernetes clusters, I want to ensure that a baseline
set of security policies (e.g., "privileged containers are disallowed in non-system namespaces") is
enforced on all clusters, even if policy engines like OPA Gatekeeper or Kyverno are accidentally
deleted or misconfigured.

By placing a `ValidatingAdmissionPolicy` manifest in the API server's configuration directory
(or mounting it via a ConfigMap on the host), I can guarantee this policy is active the moment the
API server starts, before any other workloads can be created.

```yaml
# /etc/kubernetes/admission/no-privileged.yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "platform.deny-privileged-containers"
spec:
  failurePolicy: Fail
  matchConstraints:
    resourceRules:
    - apiGroups: [""]
      apiVersions: ["v1"]
      operations: ["CREATE", "UPDATE"]
      resources: ["pods"]
  validations:
  - expression: "!object.spec.containers.exists(c, c.securityContext.privileged == true)"
    message: "Privileged containers are not allowed"
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: "platform.deny-privileged-containers-binding"
spec:
  policyName: "platform.deny-privileged-containers"
  validationActions:
  - Deny
  matchResources:
    namespaceSelector:
      matchExpressions:
      - key: "kubernetes.io/metadata.name"
        operator: NotIn
        values: ["kube-system"]
```

#### Story 2: Self-Protection and Preventing Name Collisions

As a cluster operator, I want to prevent cluster administrators from accidentally or maliciously
deleting critical admission policies. I can define a manifest-based `ValidatingAdmissionPolicy` that
intercepts DELETE and UPDATE operations on admission-related resources and denies them if they match
specific criteria (e.g., have a `platform.example.com/protected: "true"` label).

Since this policy is defined on disk and not via the API, it cannot be removed or modified through
the API, providing a hard backstop against administrative errors.

**Recommended: Prevent name collisions with static configs**

Admins who want to avoid any possibility of confusion between manifest-based and REST-based
admission configurations should use a manifest-based policy to prevent creation of REST-based
objects whose names overlap with their static config. This ensures clear separation in audit
logs and metrics:

```yaml
# /etc/kubernetes/admission/prevent-name-collision.yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicy
metadata:
  name: "platform.prevent-admission-name-collision"
spec:
  failurePolicy: Fail
  matchConstraints:
    resourceRules:
    - apiGroups: ["admissionregistration.k8s.io"]
      apiVersions: ["*"]
      operations: ["CREATE", "UPDATE"]
      resources: ["validatingadmissionpolicies", "mutatingadmissionpolicies",
                  "validatingadmissionpolicybindings", "mutatingadmissionpolicybindings",
                  "validatingwebhookconfigurations", "mutatingwebhookconfigurations"]
  validations:
  - expression: "!object.metadata.name.startsWith('platform.')"
    message: "Names starting with 'platform.' are reserved for manifest-based admission configs"
---
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionPolicyBinding
metadata:
  name: "platform.prevent-admission-name-collision-binding"
spec:
  policyName: "platform.prevent-admission-name-collision"
  validationActions:
  - Deny
```

This pattern reserves a naming prefix (e.g., `platform.`) for manifest-based configurations,
preventing REST-based objects from using the same names and eliminating audit/metrics confusion.

#### Story 3: Bootstrapping Cluster Security

As a security engineer, I need to ensure that certain security-critical webhooks are active from the
very first moment the cluster accepts requests. This includes scenarios where:

- The cluster is being restored from backup
- etcd has been reset or is temporarily unavailable
- A new cluster is being bootstrapped

By using manifest-based webhook configuration, I can guarantee that my security webhook is called
for all relevant requests from API server startup, eliminating the bootstrap gap.

### Notes/Constraints/Caveats

1. URL-only webhooks: Webhooks must use `clientConfig.url` (not `clientConfig.service`) and
   be accessible via a static IP or external DNS name.

2. Per-API-server configuration: Each API server instance loads its own manifest files. In HA
   setups, operators must ensure consistency (e.g., via shared storage or configuration management).

3. Policy bindings must reference policies defined in the same manifest file set.

### Risks and Mitigations

| Risk | Description | Mitigation |
|------|-------------|------------|
| Silent failures | If a manifest file is malformed, policies might not load, leaving the cluster unprotected. | API server fails to start if initial manifest loading encounters validation errors. Runtime reload failures are logged and exposed via metrics; previous valid configuration is retained. |
| Name collisions | A manifest-based configuration might share a name with an API-based configuration, causing confusion. | Manifest-based and API-based configurations are treated as separate domains. Both will be invoked independently. Metrics and audit annotations clearly distinguish the source. Recommend using a naming convention (e.g., `platform.*` prefix) for manifest-based configurations. |
| Configuration drift | In HA setups, different API servers might have different manifest configurations. | This is documented as expected behavior (similar to other file-based configs). Operators must use external tooling to ensure consistency. |
| Versioning | Manifest format must match API server version. | Standard API machinery decoding is used, supporting version conversion where applicable. |
| Debugging difficulty | Manifest-based configurations are not visible via the API. | Dedicated metrics expose loaded configuration counts and health. Audit annotations indicate manifest-based sources. API server logs show loaded configurations at startup. |

## Design Details

### New AdmissionConfiguration Schema

The `AdmissionConfiguration` resource is extended with a `staticManifestsDir` field for the webhook
admission plugins:

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: AdmissionConfiguration
plugins:
- name: ValidatingAdmissionWebhook
  configuration:
    apiVersion: apiserver.config.k8s.io/v1
    kind: WebhookAdmissionConfiguration
    kubeConfigFile: "<path-to-kubeconfig>"
    staticManifestsDir: "/etc/kubernetes/admission/validating/"
- name: MutatingAdmissionWebhook
  configuration:
    apiVersion: apiserver.config.k8s.io/v1
    kind: WebhookAdmissionConfiguration
    kubeConfigFile: "<path-to-kubeconfig>"
    staticManifestsDir: "/etc/kubernetes/admission/mutating/"
- name: ValidatingAdmissionPolicy
  configuration:
    apiVersion: apiserver.config.k8s.io/v1
    kind: ValidatingAdmissionPolicyConfiguration
    staticManifestsDir: "/etc/kubernetes/admission/policies/"
- name: MutatingAdmissionPolicy
  configuration:
    apiVersion: apiserver.config.k8s.io/v1
    kind: MutatingAdmissionPolicyConfiguration
    staticManifestsDir: "/etc/kubernetes/admission/mutating-policies/"
```

The `staticManifestsDir` field accepts an absolute path to a directory. All direct-children
`.yaml`, `.yml`, and `.json` files in the directory are loaded.

Glob patterns are not supported. Relative paths are not supported.

Related objects (such as a ValidatingAdmissionPolicy and its associated ValidatingAdmissionPolicyBinding)
should be placed in the same file to ensure they are loaded and reloaded together atomically.

### Manifest File Format

Manifest files contain standard Kubernetes resource definitions. Multiple resources can be included
in a single file using YAML document separators (`---`).

Single resource example:
```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: "platform.security-webhook"
webhooks:
- name: "security.platform.example.com"
  clientConfig:
    url: "https://security-webhook.platform.svc:443/validate"
    caBundle: "<base64-encoded-ca-bundle>"
  rules:
  - apiGroups: [""]
    apiVersions: ["v1"]
    operations: ["CREATE", "UPDATE"]
    resources: ["pods"]
  admissionReviewVersions: ["v1"]
  sideEffects: None
  failurePolicy: Fail
```

List example:
```yaml
apiVersion: v1
kind: List
items:
- apiVersion: admissionregistration.k8s.io/v1
  kind: ValidatingAdmissionPolicy
  metadata:
    name: "platform.require-labels"
  spec:
    # ... policy spec
- apiVersion: admissionregistration.k8s.io/v1
  kind: ValidatingAdmissionPolicyBinding
  metadata:
    name: "platform.require-labels-binding"
  spec:
    # ... binding spec
```

### Naming and Conflict Resolution

All objects in manifest files must have unique names within their type. The naming rules are:

1. Uniqueness: If two manifest files define objects of the same type with
   the same name, the API server fails to start with a descriptive error.

2. Coexistence with API-based objects: Manifest-based and API-based objects are treated as
   belonging to separate domains. If both a manifest-based and API-based `ValidatingWebhookConfiguration`
   named "example" exist, both will be invoked for matching requests.

3. Recommended naming convention: To avoid confusion, use a distinctive prefix for manifest-based
   objects (e.g., `platform.`, `static.`, or `manifest.`).

### File Watching and Dynamic Reloading

The API server watches the configured manifest files/directories for changes:

1. Initial load: At startup, all configured paths are read and validated. The API server blocks
   until all manifests are successfully loaded. Invalid manifests cause startup failure.

2. Runtime reloading: Changes to manifest files trigger a reload:
   - File modifications are detected using filesystem watching (similar to other config file
     reloading in kube-apiserver such as authentication, authorization, encryption configs)
   - New configurations are validated before being applied
   - If validation fails, the error is logged, metrics are updated, and the previous valid
     configuration is retained
   - Successful reloads atomically replace the previous configuration
   - Changes are eventually consistent and observable via metrics

3. Atomic file updates: To avoid partial reads during file writes, changes to manifest files
   should be made atomically (e.g., write to a temporary file, then atomically rename/replace
   the actual file).

4. Error handling: If any error occurs during reload (missing file, permission errors, parse errors,
   validation errors), the previous configuration is retained and the error is logged. Successful
   reloads atomically replace the previous configuration. All reload attempts update the
   `automatic_reloads_total` and `automatic_reload_last_timestamp_seconds` metrics with the
   appropriate `status` label (`success` or `failure`) and `plugin` label to identify which
   admission plugin the reload was for.

### Decoding, Defaulting, and Validation

Manifest files are decoded using the strict decoder, which rejects manifests containing duplicate
fields or unknown fields. This matches the behavior of other configuration file loading in
kube-apiserver.

Each object loaded from manifest files undergoes the same versioned defaulting and validation that
the REST API applies. This includes:

- Version conversion where applicable (via standard API machinery decoding)
- Applying defaulting for the specified API version
- Running the same validation rules that the REST API would run on that version

In addition to standard validation, manifest-based configurations undergo additional restrictions:

- Webhooks: `clientConfig.url` and `caBundle` required; `clientConfig.service` not allowed
- Policies: `spec.paramKind` not allowed
- Bindings: `spec.paramRef` not allowed; referenced policy must exist in manifest file set

### Metrics and Audit Annotations

To distinguish manifest-based admission decisions from API-based ones:

Metrics:

Existing admission metrics (e.g., `apiserver_admission_webhook_admission_duration_seconds`) are STABLE
and cannot have new labels added. Instead, manifest-based admission uses a parallel set of metrics
that mirror the existing ones but are specific to manifest-based configurations. This ensures:
- No impact on clusters not using this feature
- No confusion in metrics between REST-based and manifest-based admission
- Clear separation for monitoring and alerting

New metrics for manifest-based webhooks (parallel to existing webhook metrics):
- `apiserver_admission_manifest_webhook_admission_duration_seconds{name, type, operation, rejected}` - latency histogram
- `apiserver_admission_manifest_webhook_rejection_count{name, type, operation, error_type, rejection_code}` - rejection counter
- `apiserver_admission_manifest_webhook_fail_open_count{name, type}` - fail open counter
- `apiserver_admission_manifest_webhook_request_total{name, type, operation, code, rejected}` - request counter

New metrics for manifest-based CEL policies (parallel to existing policy metrics):
- `apiserver_validating_admission_manifest_policy_check_total{policy, policy_binding, error_type, enforcement_action}`
- `apiserver_validating_admission_manifest_policy_check_duration_seconds{policy, policy_binding, error_type, enforcement_action}`
- `apiserver_mutating_admission_manifest_policy_check_total{policy, policy_binding, error_type}`
- `apiserver_mutating_admission_manifest_policy_check_duration_seconds{policy, policy_binding, error_type}`

New metrics for manifest loading health (following the existing reload metrics pattern):
- `apiserver_admission_manifest_config_automatic_reloads_total{plugin, status}` - reload counter
- `apiserver_admission_manifest_config_automatic_reload_last_timestamp_seconds{plugin, status}` - last reload timestamp


Audit annotations:

Manifest-based admission adds a new annotation to positively indicate that static configuration
was evaluated. This is a separate annotation from the existing webhook/policy annotations, not a
modification to their structure:

```json
{
  "source.admission.k8s.io/manifest-webhooks": "platform.mutating-webhook,platform.validating-webhook",
  "source.admission.k8s.io/manifest-policies": "require-labels,deny-privileged"
}
```

These annotations list the manifest-based configurations that were evaluated for the request.
Combined with the existing `mutation.webhook.admission.k8s.io/*` and `patch.webhook.admission.k8s.io/*`
annotations, operators can determine both what was evaluated and whether it came from static config.

**Evaluation order**: Manifest-based configurations are evaluated before REST-based configurations.
This ensures that platform-level policies enforced via static config take precedence and are always
recorded in audit logs before any API-based configurations.

### Implementation

1. Configuration types: Add `StaticManifestsDir string` to webhook and policy admission configs
2. Manifest loader: New package handling file reading, validation, watching, and atomic reload
3. Composite accessor: Merge manifest and API-based configurations; evaluate manifest-based first
4. Feature gate: `ManifestBasedAdmissionControlConfig`, defaulting to false for alpha
5. Metrics: Add parallel metrics for manifest-based admission (separate from existing stable metrics)

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

This feature will be primarily covered via integration tests. Since this feature is fully contained
within kube-apiserver and does not propose any additional user-facing REST APIs, e2e tests are not
necessary. Unit tests will cover individual components but are insufficient for testing the full
admission chain.

##### Unit tests

Coverage areas:
- Manifest file loading, parsing, and validation
- Error handling for missing/unreadable/malformed files
- Composite accessor merging manifest and API-based configurations

##### Integration tests

Test scenarios:
- Bootstrap enforcement: Manifest policy active immediately at startup
- Hot reload: Adding/removing manifest files updates enforcement
- Invalid config handling: Server continues with previous valid config on reload errors
- Coexistence: Both manifest and API-based configurations invoked
- Metrics: Parallel metrics correctly emitted for manifest-based admission

##### e2e tests

None required. This is an operator-facing feature best tested via integration tests that can
control API server startup flags. The feature does not expose new REST API endpoints.

### Graduation Criteria

#### Alpha

- Feature implemented behind `ManifestBasedAdmissionControlConfig` feature gate
- Integration tests completed and passing
- Manifest loading for webhooks (ValidatingWebhookConfiguration, MutatingWebhookConfiguration)
  implemented
- Manifest loading for CEL policies (ValidatingAdmissionPolicy, MutatingAdmissionPolicy, bindings)
  implemented
- Complete metrics implementation with parallel metrics for manifest-based admission
- Audit annotation support for manifest-based sources
- File watching and hot reload fully implemented and tested
- Documentation for alpha usage

#### Beta

- Feature gate defaults to enabled
- All known alpha issues resolved

#### GA

- At least two production users providing feedback
- Stable usage in production environments for at least two releases
- No regressions in API server startup time
- All feedback from beta users addressed

### Upgrade / Downgrade Strategy

Upgrade:
- Enabling the feature and providing manifest configuration is opt-in
- Existing clusters without manifest configuration see no change
- Clusters can gradually adopt by adding manifest files without disruption

Downgrade:
- Before downgrading to a version without this feature, operators must:
  1. Remove manifest file references from `AdmissionConfiguration`
  2. If relying on manifest-based policies, recreate them as API objects (where possible)
- Downgrading without removing configuration will cause API server startup failure (unknown
  configuration field)

### Version Skew Strategy

This is a purely API-server-internal feature. No other components (kubelet, kube-scheduler,
kube-controller-manager, etc.) are aware of the source of admission decisions. Therefore, version
skew between control plane components does not affect this feature.

In HA setups with multiple API servers:
- All API servers should be upgraded together (standard practice)
- During rolling upgrades, some API servers may have the feature while others don't
- Manifest files should only be deployed after all API servers support the feature

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ManifestBasedAdmissionControlConfig`
  - Components depending on the feature gate: `kube-apiserver`
- [x] Other
  - Mechanism: `--admission-control-config-file` pointing to an `AdmissionConfiguration` with
    `staticManifestsDir` configured
  - Enabling/disabling requires API server restart
  - No impact on nodes

###### Does enabling the feature change any default behavior?

No. Enabling the feature gate alone does not change behavior. Behavior changes only when manifest
files are explicitly configured in `AdmissionConfiguration`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disable the `ManifestBasedAdmissionControlConfig` feature gate or remove the
`staticManifestsDir` entries from `AdmissionConfiguration` and restart API server.
Manifest-based admission controls will no longer be enforced.

###### What happens if we reenable the feature if it was previously rolled back?

The manifest-based configurations will be loaded and enforced again. No state is persisted, so
re-enablement is clean.

###### Are there any tests for feature enablement/disablement?

Yes, integration tests will be added to verify correct behavior with feature gate enabled/disabled
and with/without manifest configuration.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout failures:
- Invalid manifest files cause API server startup failure
- Misconfigured webhooks (unreachable URLs) will reject requests if `failurePolicy: Fail`
- In HA setups, inconsistent manifest files across API servers cause inconsistent behavior

Impact on running workloads:
- Already running workloads are not affected (admission only applies to API requests)
- New requests may be rejected if policies are misconfigured

Mitigation:
- Validate manifest files before deployment
- Use `failurePolicy: Ignore` during initial rollout for webhooks
- Ensure consistent configuration across all API servers before enabling

###### What specific metrics should inform a rollback?

- `apiserver_admission_manifest_config_automatic_reloads_total{status="failure"}` increasing
- `apiserver_admission_manifest_webhook_rejection_count` unexpectedly high
- API server crash loops (check container restart count)
- Increased API request latency (webhook timeouts)

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested during alpha/beta, including upgrade→downgrade→upgrade path.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- Metric: `apiserver_admission_manifest_config_automatic_reloads_total > 0`
- Check `AdmissionConfiguration` for `staticManifestsDir` entries
- Check API server logs for manifest loading messages at startup

###### How can someone using this feature know that it is working for their instance?

- [x] Metrics
  - `apiserver_admission_manifest_config_automatic_reloads_total{status="success"}` shows successful reloads
  - `apiserver_admission_manifest_config_automatic_reload_last_timestamp_seconds` shows recent timestamp
  - `apiserver_admission_manifest_webhook_admission_duration_seconds` shows webhook activity
- [x] API server logs
  - Log message at startup: "Loaded N manifest-based webhook configurations"
  - Log message on reload: "Reloaded manifest-based configurations"
- [x] Admission behavior
  - Requests matching manifest-based policies are appropriately admitted/rejected

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- Startup time increase: < 1 second with typical configuration (< 100 policies)
- Manifest reload time: < 100ms
- No increase in p99 admission latency

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- `apiserver_admission_manifest_config_automatic_reloads_total{status="success"}` - rate of successful reloads
- `apiserver_admission_manifest_config_automatic_reloads_total{status="failure"}` - rate of failed reloads (should be 0)

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Potentially useful future additions (deferred to avoid cardinality issues):
- Per-file load status
- Configuration hash for drift detection

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No cluster services required. Configured webhook URLs must be reachable from API server.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new API calls. Manifest-based webhooks make HTTP calls same as API-based webhooks.

###### Will enabling / using this feature result in introducing new API types?

No new REST API types. `staticManifestsDir` field added to existing admission configuration types.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Potentially minimal increase in:
- API server startup time (reading and validating manifest files)
- Admission latency (additional webhooks/policies to evaluate)

Expected impact is negligible for typical configurations. Performance testing will validate.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Minimal increase:
- Memory: Proportional to number of configured policies/webhooks
- Disk I/O: Initial read at startup; periodic reads on file changes
- CPU: Negligible (parsing only on load/reload)

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Unlikely with reasonable configurations. Uses inotify watchers (one per directory) and shares
HTTP client pool with API-based webhooks.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

- API server unavailable: Feature is contained within API server; N/A
- etcd unavailable: Feature operates independently of etcd. Manifest-based policies continue
  to function even if etcd is unavailable, which is one of the motivating use cases.

###### What are other known failure modes?

| Failure Mode | Detection | Mitigation | Diagnostics |
|--------------|-----------|------------|-------------|
| Invalid manifest at startup | API server fails to start | Fix manifest file; Restart | API server logs show validation errors |
| Invalid manifest on reload  | Metrics and logs | Fix manifest file; Wait for reload or restart | API server logs show validation errors |
| Webhook endpoint unreachable | `apiserver_admission_webhook_fail_open_count` increases | Fix webhook endpoint; or change `failurePolicy` | Check webhook URL connectivity |
| File permission errors on startup | `apiserver_admission_manifest_config_automatic_reloads_total{status="failure"}` | Fix file permissions; Restart | API server logs show permission errors |
| File permission errors on reload | `apiserver_admission_manifest_config_automatic_reloads_total{status="failure"}` | Fix file permissions; Wait for reload or restart | API server logs show permission errors |
| Configuration drift across HA | Inconsistent admission decisions | Use configuration management | Compare manifest files across API servers |

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check `apiserver_admission_manifest_config_automatic_reloads_total{status="failure"}` for reload failures
2. Check API server logs for manifest-related errors
3. Verify webhook endpoints are reachable and responding quickly
4. Compare manifest configurations across API server instances
5. Temporarily switch webhooks to `failurePolicy: Ignore` to isolate issues
6. As last resort, remove manifest configuration to restore baseline behavior

## Implementation History

- 2020-04-21: Original KEP-1872 introduced for manifest-based admission webhooks
- 2026-01-15: KEP-5793 created, expanding scope to include CEL-based policies (VAP/MAP)

## Drawbacks

1. Reduced visibility: Users cannot list all active admission controls via the API. Manifest-based
   configurations require out-of-band inspection. This mirrors the visibility characteristics of
   compiled-in admission controllers.

2. Operational complexity: In HA setups, operators must ensure consistent configuration across
   all API server instances using external tooling.

3. Debugging difficulty: When admission is denied, users cannot easily determine if a manifest-based
   or API-based policy is responsible without access to metrics or logs.

4. Limited functionality: No support for paramKind, service references, or dynamic credentials
   limits the flexibility compared to API-based configurations.

## Alternatives

### Deny policies in RBAC

Adding deny policies to RBAC could allow protecting webhook configuration objects from deletion.
However:
- Would require significant RBAC redesign
- Far-reaching consequences for watchers and other components
- Doesn't address the bootstrap gap problem
- Overly broad solution for a specific use case

### Static admission plugins

Compiling custom admission logic into the API server binary would achieve similar goals but:
- Requires custom API server builds
- Much higher barrier to entry
- No runtime configurability
- Not practical for most operators

### External configuration management

Using external tools (Helm, Kustomize, GitOps) to ensure webhook configurations exist:
- Doesn't eliminate the bootstrap gap
- Configurations can still be deleted via API
- Relies on eventual consistency
- Doesn't provide hard protection guarantees

## Infrastructure Needed (Optional)

None.

[MutatingAdmissionPolicy]: https://github.com/kubernetes/enhancements/issues/3962
