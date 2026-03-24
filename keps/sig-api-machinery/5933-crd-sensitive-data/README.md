# KEP-5933: Sensitive Data Marker for CRD Fields

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API Changes](#api-changes)
    - [OpenAPI Extension](#openapi-extension)
    - [Structural Schema](#structural-schema)
    - [Validation Rules](#validation-rules)
  - [RBAC Model](#rbac-model)
  - [Field Stripping on get/list/watch](#field-stripping-on-getlistwatch)
  - [Audit Log Masking](#audit-log-masking)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Feature Gate](#feature-gate)
  - [Schema Layer](#schema-layer)
  - [Sensitive Field Utilities](#sensitive-field-utilities)
  - [CRD Handler Integration](#crd-handler-integration)
  - [Watch Filtering](#watch-filtering)
  - [Audit Integration](#audit-integration)
  - [CRD Update and Deletion](#crd-update-and-deletion)
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
- [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
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

This KEP proposes a mechanism to mark individual fields in Custom Resource Definition (CRD) OpenAPI v3 schemas as sensitive using a new vendor extension `x-kubernetes-sensitive-data`. When at least one field in a CRD schema carries this marker, the API server will:

1. **Strip** marked fields from get/list/watch responses unless the caller has explicit RBAC permission on a new `sensitive` subresource.
2. **Mask** marked fields unconditionally in audit log entries, regardless of the audit level or the caller's permissions.

This brings Secret-grade protection to arbitrary CRD fields without requiring CRD authors to split sensitive values into separate Secret objects.

## Motivation

Operators and platform teams frequently store sensitive values — passwords, TLS private keys, API tokens — directly in CRD specs because the alternative (referencing a separate Secret and wiring it through controllers) adds significant complexity. Today, those values are:

- Returned **in full** to every authenticated caller that can `get` the resource, even if the caller only needs non-sensitive fields.
- Logged **in cleartext** in audit log entries, creating a compliance risk.

There is no first-class Kubernetes primitive that lets a CRD author say "this field is sensitive — treat it accordingly".

### Goals

- Introduce `CRDSensitiveData` feature gate for a new schema-level marker `x-kubernetes-sensitive-data: true` for CRD fields.
- Gate access to sensitive field values behind a dedicated RBAC subresource (`<resource>/sensitive`), stripping them from API responses when the caller lacks permission.
- Mask sensitive field values unconditionally in audit log request and response bodies.

### Non-Goals

- Changing the behavior of built-in resources (e.g., Secret, ConfigMap).
- Encryption at rest in etcd (neither field-level nor resource-level encryption is provided by this KEP).
- External secret management integration (e.g., Vault, cloud KMS) — this KEP focuses on API-server-native mechanisms only.
- Protecting fields in non-CRD aggregated API servers.

## Proposal

### API Changes

#### OpenAPI Extension

A new boolean vendor extension `x-kubernetes-sensitive-data` is introduced in CRD OpenAPI v3 validation schemas:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: databases.example.com
spec:
  group: example.com
  names:
    kind: Database
    plural: databases
  scope: Namespaced
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Validation:
          type: object
          properties:
            spec:
              type: object
              properties:
                host:
                  type: string
                port:
                  type: integer
                password:
                  type: string
                  x-kubernetes-sensitive-data: true
                tls:
                  type: object
                  x-kubernetes-sensitive-data: true
                  properties:
                    privateKey:
                      type: string
                    certificate:
                      type: string
```

#### Structural Schema

A new field `XSensitiveData bool` is added to `JSONSchemaProps` in internal, v1, and v1beta1 API types, and to the `Extensions` struct in the structural schema package. 
The field is propagated through conversion, deep-copy, and OpenAPI export.

#### Validation Rules

Added to `pkg/apis/apiextensions/validation/validation.go`:

- `x-kubernetes-sensitive-data: true` is allowed only on **leaf** scalar fields (`string`, `integer`, `number`, `boolean`) and on `object`/`array` nodes (marking the entire sub-tree).
- It is **forbidden** on the root schema node (you cannot mark the entire resource as sensitive).
- When the `CRDSensitiveData` feature gate is disabled, setting the marker to `true` is rejected during CRD validation.

### RBAC Model

A new virtual subresource `sensitive` is defined for each CRD that contains sensitive fields. To read sensitive field values, a caller must have `get`, `list`, or `watch` permission on `<resource>/sensitive`.

Example `ClusterRole`:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: database-sensitive-reader
rules:
  - apiGroups: ["example.com"]
    resources: ["databases/sensitive"]
    verbs: ["get", "list", "watch"]
```

Callers **without** this permission receive the resource with sensitive fields removed (set to `null` / omitted). Callers **with** this permission receive the full resource.

Write access to sensitive fields is controlled by standard `create`/`update`/`patch` verbs on the main resource — no additional write-side subresource is required.

For resources with sensitive fields, full-object updates require authorization to read `<resource>/sensitive`. If a caller can update the main resource but cannot read its sensitive content, the API server rejects the request. This prevents accidental removal of sensitive fields during read-modify-write flows based on redacted responses.

### Field Stripping on get/list/watch

In the CRD handler (`customresource_handler.go`), after deserializing the response object:

1. Check whether the CRD has any sensitive fields (cached in `crdInfo`).
2. If yes, perform an RBAC SubjectAccessReview for `<resource>/sensitive`.
3. If the caller lacks permission, deep-copy the object and call `StripSensitiveFields()` before returning it.

For `watch`, a `watch.Filter` wrapper strips sensitive fields from every event object delivered to callers without the `sensitive` subresource permission.

### Audit Log Masking

Sensitive fields are **never** written to the audit log in cleartext. Before serialization:

- `audit.LogRequestObject()` and `audit.LogResponseObject()` invoke an `ObjectTransformer` registered by the CRD handler.
- The transformer deep-copies the object and calls `MaskSensitiveFields()`, replacing each sensitive value with `"******"`.
- PATCH request bodies containing sensitive paths are masked in the same way.

This masking applies regardless of the audit policy level and regardless of whether the caller has the `sensitive` subresource permission.

### User Stories

#### Story 1

As a CRD author, I want to store passwords and private keys in my custom resource `spec` without exposing them via `kubectl get -o yaml` to every cluster user, and without them appearing in cleartext in audit logs.

I add `x-kubernetes-sensitive-data: true` to the relevant fields in my CRD schema. The API server automatically strips those fields for callers without `myresource/sensitive` permission and masks the values in audit entries.

#### Story 2

As a platform team, I want to propose a standardized mechanism for marking sensitive CRD fields to the Kubernetes community, so that the ecosystem converges on a single approach instead of ad-hoc solutions in each operator.

### Notes/Constraints/Caveats (Optional)

- The marker applies at the API server level only. It does not prevent a controller with full RBAC from reading and leaking sensitive values through its own logs or status fields. Operators should apply the principle of least privilege.
- PATCH masking in audit is more complex than GET/PUT masking because the patch body must be parsed and individual values replaced by path.

### Risks and Mitigations

**Risk: Performance overhead of per-request RBAC checks for sensitive fields.**

Every get/list/watch for a CRD with sensitive fields triggers an additional authorization check. This can be mitigated by caching authorization decisions (similar to how the API server caches RBAC decisions today) and by skipping the check entirely when the CRD has no sensitive fields.

**Risk: Complexity in CRD handler and interaction with audit.**

The CRD handler gains new responsibilities. Thorough unit and integration tests, as well as a clear separation into utility packages (`sensitive/sensitive.go`), will mitigate correctness risks.

**Risk: Data loss from full-object updates by callers without sensitive read access.**  

A caller that has permission to update the resource, but is not authorized to read `<resource>/sensitive`, receives a redacted form of the object. If that caller performs a full-object update (`PUT`) using the returned representation, sensitive fields omitted from the response may be unintentionally removed from the persisted object. To mitigate this risk, for resources with sensitive fields, the API server rejects full-object updates from callers that are not authorized to read the sensitive content.

## Design Details

### Feature Gate

A new alpha feature gate `CRDSensitiveData` is added to `staging/src/k8s.io/apiextensions-apiserver/pkg/features/kube_features.go`. All new logic is gated behind this flag.

### Schema Layer

The `XSensitiveData bool` field is added at every level of the CRD schema representation: internal types, external v1 and v1beta1 API types (with the corresponding JSON and protobuf tags), the structural schema `Extensions` struct used at runtime, and the OpenAPI export path. Conversion, deep-copy, and code-generation outputs are updated accordingly.

### Sensitive Field Utilities

A new package within the apiextensions-apiserver provides utilities for checking whether a schema has any sensitive fields, collecting their JSON paths, stripping sensitive values from unstructured objects (for API responses), and masking them with `"******"` (for audit logs). Both strip and mask functions traverse the unstructured object tree alongside the structural schema, following the same pattern as `pruning.Prune()`.

### CRD Handler Integration

The CRD handler caches whether each served resource has sensitive fields. On every get/list/watch request, if sensitive fields are present, the handler performs an RBAC check on the `<resource>/sensitive` subresource. Callers that lack permission receive a deep copy of the object with sensitive fields stripped before the response is sent.

### Watch Filtering

Watch streams are wrapped in a filter that strips sensitive fields from every event object before delivering it to the client. The filter applies the same RBAC-based logic as get/list: callers with `<resource>/sensitive` permission receive unmodified events.

### Audit Integration

In `AuditContext` (`staging/src/k8s.io/apiserver/pkg/audit/context.go`):

```go
type ObjectTransformer func(obj runtime.Object) runtime.Object

func (ac *AuditContext) SetObjectTransformer(fn ObjectTransformer)
```

Before encoding audit request/response bodies, if an `ObjectTransformer` is set, the audit subsystem invokes it. The CRD handler registers a transformer that deep-copies the object and calls `MaskSensitiveFields()`.

For PATCH requests, the transformer parses the JSON patch body and masks values at sensitive paths, or replaces the entire body with `{"redacted": "patch contains sensitive fields"}`.

### CRD Update and Deletion

When a CRD is updated to remove all sensitive fields, or deleted entirely, the handler clears the cached sensitive-field metadata for that resource. In-flight requests complete normally.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

- `schema/sensitive`: `HasSensitiveFields`, `CollectSensitivePaths`, `StripSensitiveFields`, `MaskSensitiveFields` on various schemas (nested objects, arrays, leaf scalars).
- `validation`: marker on root (rejected), on leaf (accepted), on object/array (accepted), with feature gate disabled (rejected).

Target packages and current coverage:

- `staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/schema`: TBD
- `staging/src/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation`: TBD

##### Integration tests

- Create a CRD with `x-kubernetes-sensitive-data` fields, create a CR, and verify:
  - `GET` without `sensitive` subresource permission returns the object with sensitive fields stripped.
  - `GET` with `sensitive` subresource permission returns the full object.
  - `WATCH` events strip sensitive fields for unauthorized callers.

##### e2e tests

- End-to-end test creating a CRD with sensitive fields, creating CRs, and validating field stripping and audit masking behavior.
- Test file: `test/e2e/apimachinery/crd_sensitive_data.go` (new).

### Graduation Criteria

#### Alpha

- Feature implemented behind `CRDSensitiveData` feature gate.
- Schema marker validation, field stripping, and audit masking are implemented.
- Unit and integration tests pass.

#### Beta

- Feedback collected from early adopters.
- e2e tests added and passing in CI.
- Performance impact of per-request RBAC checks for sensitive fields is measured and acceptable.
- Documentation published.

#### GA

- Feature enabled by default.
- Conformance tests added.
- At least two releases of soak time at Beta.

### Upgrade / Downgrade Strategy

**Upgrade:** Enabling the `CRDSensitiveData` feature gate and applying a CRD with `x-kubernetes-sensitive-data` fields activates the feature. No data migration is required.

**Downgrade:** Disabling the feature gate causes the API server to reject new CRDs with the marker. Existing CRDs with the marker continue to function (the field is preserved in storage), but the sensitive-field logic is not enforced: fields are returned in full and audit entries are not masked.

### Version Skew Strategy

The feature is entirely server-side (kube-apiserver / apiextensions-apiserver). There is no client-side component. In a multi-apiserver HA configuration, if some API servers have the feature gate enabled and others do not, callers may see inconsistent behavior (fields stripped on one server but not another). Therefore, the feature gate should be enabled or disabled uniformly across all API servers before serving traffic.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `CRDSensitiveData`
  - Components depending on the feature gate:
    - kube-apiserver (apiextensions-apiserver)

###### Does enabling the feature change any default behavior?

No. The feature only activates when a CRD is created or updated with at least one field marked `x-kubernetes-sensitive-data: true`. Existing CRDs without the marker are unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate stops enforcing sensitive-field stripping and audit masking. The `x-kubernetes-sensitive-data` field values are preserved in stored CRDs but ignored at runtime.

###### What happens if we reenable the feature if it was previously rolled back?

Safe. The CRD handler re-reads the stored schemas and rebuilds the sensitive-field cache. Field stripping and audit masking resume immediately.

###### Are there any tests for feature enablement/disablement?

Yes. Integration tests verify behavior with the feature gate enabled and disabled, including the case where a CRD with sensitive markers exists while the gate is off.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout failure could occur if the API server fails to start with the new code. This would be caught by standard API server health checks. Running workloads (CRs) are unaffected because the feature only changes how responses are constructed and how audit entries are recorded, not how data is stored.

A rollback removes field stripping and audit masking. This means sensitive values become visible again in API responses and audit logs. Cluster admins should be aware of this when planning rollbacks.

No data migration is required on rollback.

###### What specific metrics should inform a rollback?

- Elevated error rates on CRD get/list/watch requests.
- Elevated API server latency on CRD endpoints.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Integration tests cover enable → disable → enable transitions.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. The feature is self-contained within the API server.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes. For each get/list/watch request on a CRD with sensitive fields, an additional authorization check is performed for the `<resource>/sensitive` subresource. This leverages the existing authorizer cache and is expected to have negligible overhead.

###### Will enabling / using this feature result in introducing new API types?

No new API types. A new field is added to the existing `JSONSchemaProps` type, and a virtual subresource `sensitive` is introduced.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

CRDs with the marker will have a slightly larger schema (one additional boolean per marked field). CR objects themselves are unchanged in size.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

A small increase in get/list/watch latency for CRDs with sensitive fields is expected due to the additional RBAC check and potential deep-copy + field stripping. This is expected to be within noise for typical workloads.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Deep-copying objects for field stripping increases transient memory allocation. For large objects with many sensitive fields, this could be measurable. We will benchmark during alpha and optimize if needed.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This is a control-plane-only feature.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature is part of the API server request path. If the API server is unavailable, no requests are served. If etcd is unavailable, CRD storage operations fail as they do today — the sensitive data feature does not change this behavior.

###### What are other known failure modes?

- **Sensitive fields not stripped from responses**
  - Detection: Perform a `GET` on a CR with sensitive fields as a user without `<resource>/sensitive` permission and verify the fields are absent.
  - Mitigation: Verify the feature gate is enabled and the CRD schema contains `x-kubernetes-sensitive-data: true`.
  - Diagnostics: Check API server logs for errors during schema processing.

- **Audit log contains cleartext sensitive values**
  - Detection: Manual inspection of audit log entries.
  - Mitigation: Verify the `ObjectTransformer` is registered by the CRD handler. Check API server logs.
  - Diagnostics: Enable verbose logging on the audit subsystem.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check whether the `CRDSensitiveData` feature gate is enabled on all API servers.
2. Inspect the CRD schema to confirm `x-kubernetes-sensitive-data` markers are present.
3. Review API server logs (`kube-apiserver.log`) for errors in schema processing or authorization.
4. If performance is degraded, consider whether the number of sensitive CRDs or the size of CRs is unusually large, and profile the API server.

## Implementation History

- 2025-02-19: Initial KEP draft.

## Drawbacks

- Requires upstream Kubernetes changes in apiextensions-apiserver and kube-apiserver, which implies a long review and acceptance cycle.
- Adds complexity to the CRD handler, which must now integrate with RBAC subresources and the audit subsystem.
- PATCH request audit masking is technically challenging, requiring JSON path analysis of patch bodies.

## Alternatives

**Move all secrets to an external vault (e.g., HashiCorp Vault).**
This is a valid complementary approach but does not address the need for in-cluster API-level protection of CRD fields. It also requires significant operational overhead for every cluster.

**Create a separate resource type for each "secret" object.**
This leads to schema duplication and API proliferation. The per-field marker approach is more flexible and does not require additional resources.

**Store sensitive values in a native Kubernetes Secret and reference it from the CRD.**  
This is the primary alternative used today and remains broadly viable. However, it requires splitting the API across multiple objects and adds controller and lifecycle management complexity. This KEP aims to provide Secret-like protection for selected CRD fields without requiring an additional resource.
