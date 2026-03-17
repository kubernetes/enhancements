# KEP-34146: kubectl example

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Precedent Analysis](#precedent-analysis)
  - [Successful kubectl UX Additions](#successful-kubectl-ux-additions)
  - [Failed Precedent: KEP-2380 Data-Driven Commands](#failed-precedent-kep-2380-data-driven-commands)
  - [Key Takeaway](#key-takeaway)
- [Why Not a Plugin?](#why-not-a-plugin)
- [Proposal](#proposal)
  - [Basic Usage](#basic-usage)
  - [Advanced Usage](#advanced-usage)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Architecture: Struct-Based Generation](#architecture-struct-based-generation)
  - [Builder Registry](#builder-registry)
  - [Adding New Examples](#adding-new-examples)
  - [Default Values](#default-values)
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
- [Release Timing Strategy](#release-timing-strategy)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Future Work](#future-work)
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

This KEP proposes adding a new kubectl subcommand `kubectl example` that generates production-ready seed YAML for Kubernetes resources using typed Go structs. It complements `kubectl explain` by providing practical, immediately applicable manifests rather than schema documentation.

The workflow is: `kubectl explain pod` (understand the schema) → `kubectl example pod` (get a working manifest) → `kubectl apply -f -` (try it).

## Motivation

Users often need practical, applicable YAML examples for Kubernetes resources. While `kubectl explain` provides detailed schema information, it doesn't give users ready-to-use YAML snippets. This creates a gap where users must manually construct YAML from documentation, which can be error-prone and time-consuming, especially for beginners.

This KEP addresses that gap by introducing `kubectl example`, which outputs seed YAML for resources. The examples are designed to be:

- **Immediately applicable**: Can be piped directly to `kubectl apply` for testing
- **Best-practice oriented**: Include resource limits, recommended labels, and common configurations
- **Educational**: Serve as starting points that users can modify for their needs
- **Offline-capable**: Generated entirely from in-binary Go structs with no API server dependency

For instance:

```shell
# Understand the schema
kubectl explain pod

# Get a working manifest
kubectl example pod

# Try it immediately
kubectl example pod | kubectl apply -f -
```

### Goals

1. Provide a new `kubectl example` subcommand that outputs seed YAML for Kubernetes resources
2. Complement `kubectl explain` by offering practical, applicable examples
3. Support common resources with sensible defaults and customization flags (`--name`, `--image`, `--replicas`)
4. Work fully offline using struct-based generation (no API server required)

### Non-Goals

1. Replace or modify `kubectl explain`
2. Provide exhaustive examples for all possible configurations
3. Generate examples dynamically from cluster state
4. Cover every Kubernetes resource kind — focus on the most commonly used resources

## Precedent Analysis

Several kubectl UX commands have successfully navigated the KEP process. Their history provides a roadmap for `kubectl example`.

### Successful kubectl UX Additions

| Command | KEP | Time to Alpha | Key Factor |
|---------|-----|---------------|------------|
| `kubectl debug` | [KEP-1441](https://github.com/kubernetes/enhancements/tree/master/keps/sig-cli/1441-kubectl-debug) | ~8 months | Clear user pain point (debugging pods), sig-cli sponsor early |
| `kubectl diff` | [KEP-491](https://github.com/kubernetes/enhancements/tree/master/keps/sig-cli/491-kubectl-diff) | ~2 months | Small, focused scope — one command, one purpose |
| `kubectl events` | [KEP-1440](https://github.com/kubernetes/enhancements/tree/master/keps/sig-cli/1440-kubectl-events) | ~2 years | Broader scope, required more iteration on API surface |
| `kubectl wait` | N/A (pre-KEP) | Graduated alpha→beta→GA | Utility command, no feature gate needed |

**Common success factors**: (1) clear user pain point, (2) small surface area, (3) client-only with no server changes, (4) early sig-cli sponsor engagement.

`kubectl example` shares all four factors — it is a single read-only command that generates YAML locally.

### Failed Precedent: KEP-2380 Data-Driven Commands

[KEP-2380](https://github.com/kubernetes/enhancements/tree/master/keps/sig-cli/2380-scalable-kubectl-commands) attempted to solve a related problem — making kubectl commands data-driven so they could adapt to new resource types. It was ultimately abandoned because it **required server-side metadata changes** (adding command hints to CRD schemas), which created cross-SIG coordination challenges and coupling between client and server.

### Key Takeaway

`kubectl example` succeeds where KEP-2380 failed by being **entirely client-side**. No new API types, no server-side metadata, no CRD schema changes. Builders are compiled into the kubectl binary and work offline. This eliminates the cross-SIG coordination burden that stalled KEP-2380.

## Why Not a Plugin?

A natural question is whether `kubectl example` should be a [kubectl plugin](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/) (e.g., `kubectl-example` distributed via [krew](https://krew.sigs.k8s.io/)) rather than a built-in command. We considered this and believe built-in is the right choice:

| Concern | Built-in command | Plugin |
|---------|-----------------|--------|
| **Discovery** | Appears in `kubectl --help` and shell completion | Invisible unless user knows to search krew |
| **Distribution** | Available to every kubectl user immediately | Requires separate install step |
| **CI testing** | Tested in Kubernetes CI on every release | Maintained separately, may drift from API types |
| **Educational flow** | `kubectl explain` → `kubectl example` is a natural, discoverable pair | Users must learn about krew, find the plugin, install it |
| **Type safety** | Uses in-tree API types (`corev1`, `appsv1`), compile-time guarantees | Must vendor or copy types, no compile-time guarantees against kubectl's tree |

Additionally, **`kubectl example` is not a niche tool** — it targets the same audience as `kubectl explain`, which is every kubectl user. The `explain` → `example` flow is most valuable when both commands are first-class and discoverable together.

**Comparison with `kubectl create`**: `kubectl create` generates minimal imperative manifests for quick resource creation. `kubectl example` generates educational, best-practice manifests designed as starting points for real workloads — including resource limits, recommended labels, and production-oriented defaults.

## Proposal

### Basic Usage

```shell
kubectl example pod
```

Outputs a complete, valid Pod manifest:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: example-pod
  labels:
    app.kubernetes.io/name: example-pod
spec:
  containers:
  - name: example-container
    image: alpine:latest
    command: ["sleep", "3600"]
    resources:
      requests:
        memory: "64Mi"
        cpu: "250m"
      limits:
        memory: "128Mi"
        cpu: "500m"
```

### Advanced Usage

- `kubectl example deployment --replicas=3 --image=myapp:v1` — Generate with custom parameters
- `kubectl example --list` — List all available example resources
- `kubectl example pod --name=my-pod | kubectl apply -f -` — Customize and apply directly

### Risks and Mitigations

#### No Examples Available for a Resource

**Risk**: The requested resource does not have a predefined example.

**Mitigation**: Return a clear error message suggesting `kubectl explain` for schema information and `kubectl example --list` for available examples. The `--list` flag makes coverage explicit.

#### Outdated Examples

**Risk**: Examples may not reflect the latest best practices or API changes.

**Mitigation**: Examples are generated from canonical Go API types (`corev1.Pod`, `appsv1.Deployment`, etc.), so they are always structurally correct for the kubectl version. Best-practice defaults (resource limits, labels) are maintained as part of kubectl releases. Struct-based generation ensures output stays in sync with API types at compile time.

## Design Details

### Architecture: Struct-Based Generation

Examples are generated using **typed Go structs** from the Kubernetes API, not embedded YAML templates. Each resource kind has a dedicated builder function that constructs a fully-typed API object and marshals it to YAML via `sigs.k8s.io/yaml`.

High-level flow:

1. User types `kubectl example <resource>` (with optional `--name`, `--image`, `--replicas` flags)
2. kubectl resolves the resource kind, including aliases (e.g., `po` → `pod`, `deploy` → `deployment`)
3. kubectl looks up the builder function in a `buildersByKind` registry
4. The builder constructs a typed Go struct (e.g., `corev1.Pod`, `appsv1.Deployment`) with the user's flag values applied
5. `sigs.k8s.io/yaml` marshals the struct to YAML
6. kubectl outputs the YAML to stdout

This approach provides:

- **Type safety**: Builders use `corev1`, `appsv1`, `batchv1`, `networkingv1`, and `metav1` API types, so invalid field names or structures are caught at compile time
- **Determinism**: Same inputs always produce identical YAML output — no template rendering, string interpolation, or conditional logic
- **Parameterization**: `--name`, `--image`, and `--replicas` flags modify the struct fields before marshaling, providing real customization
- **API consistency**: Output automatically follows Kubernetes API field ordering conventions since it is marshaled from the canonical Go types

### Builder Registry

The `buildersByKind` map routes resource kind strings (and their aliases) to builder functions:

```go
buildersByKind map[string]func(name, image string, replicas int) ([]byte, error)
```

Supported resources and aliases:

| Kind | Aliases | Builder | API Types Used |
|------|---------|---------|----------------|
| pod | pods, po | `buildPod` | `corev1.Pod` |
| deployment | deployments, deploy | `buildDeployment` | `appsv1.Deployment` |
| service | services, svc | `buildService` | `corev1.Service` |
| persistentvolumeclaim | persistentvolumeclaims, pvc | `buildPVC` | `corev1.PersistentVolumeClaim` |
| secret | secrets | `buildSecret` | `corev1.Secret` |
| customresourcedefinition | customresourcedefinitions, crd | `buildCRD` | `map[string]interface{}` (unstructured) |
| configmap | configmaps, cm | `buildConfigMap` | `corev1.ConfigMap` |
| job | jobs | `buildJob` | `batchv1.Job` |
| cronjob | cronjobs | `buildCronJob` | `batchv1.CronJob` |
| ingress | ingresses, ing | `buildIngress` | `networkingv1.Ingress` |
| networkpolicy | networkpolicies, netpol | `buildNetworkPolicy` | `networkingv1.NetworkPolicy` |

Note: CRD uses an unstructured map because `k8s.io/apiextensions-apiserver` is not in kubectl's `go.mod`. All other resources use their canonical typed API objects.

### Adding New Examples

To add a new resource example:

1. Create a builder function in `resources.go` that returns the typed API object
2. Register the kind and its aliases in the `buildersByKind` map in `example.go`
3. Add a fallback alias entry in `fallbackResolve()` for offline resolution
4. Add unit tests that unmarshal the output back into the typed object and assert field values
5. Update `--list` output (automatic from `buildersByKind` keys)

### Default Values

Each builder applies sensible, production-oriented defaults:

| Resource | Image | Key Defaults |
|----------|-------|-------------|
| **Pod** | `alpine:latest` | `sleep 3600` command, resource requests (250m CPU, 64Mi memory) and limits (500m CPU, 128Mi memory) |
| **Deployment** | `nginx:stable` | 1 replica, port 80, resource limits |
| **Service** | — | ClusterIP type, port 80→80 |
| **PVC** | — | ReadWriteOnce, 1Gi storage |
| **Secret** | — | Opaque type with placeholder `stringData` |
| **CRD** | — | Complete apiextensions/v1 structure with OpenAPI validation schema |
| **ConfigMap** | — | `config.yaml` file key + `LOG_LEVEL` environment variable key |
| **Job** | `perl:5.40` | Pi calculation example, BackoffLimit=4, RestartPolicy=Never |
| **CronJob** | `busybox:1.36` | `*/5 * * * *` schedule, date command |
| **Ingress** | — | nginx rewrite annotation, `example.com` host, PathTypePrefix, port 80 |
| **NetworkPolicy** | — | Frontend→App→Database flow, Ingress+Egress policy types |

All resources include `app.kubernetes.io/name` labels following Kubernetes recommended labels convention.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None required. The command is purely additive with no changes to existing kubectl behavior.

##### Unit tests

- Verify correct YAML output for all 11 supported resources
- Verify `--name`, `--image`, and `--replicas` flag overrides work correctly
- Verify alias resolution for all registered aliases (po, deploy, svc, pvc, crd, cm, ing, netpol)
- Verify error handling for unsupported resource kinds
- Verify `--list` output includes all registered resources
- Tests unmarshal YAML back into typed Go objects and assert specific field values (not string matching)
- **Current coverage**: 15 test functions, all passing

##### Integration tests

Integration tests will ensure the command integrates well with kubectl's existing infrastructure, including resource discovery when a kubeconfig is available, and that the command falls back gracefully to offline alias resolution when no API server is reachable.

##### e2e tests

E2E tests will validate that the output YAML can be applied to a cluster successfully:

- `kubectl example pod | kubectl apply -f -` creates a running pod
- `kubectl example deployment | kubectl apply -f -` creates a deployment with correct replica count
- Output works across different cluster configurations and Kubernetes versions

### Graduation Criteria

#### Alpha

- `kubectl example` command implemented with 11 resource builders (pod, deployment, service, pvc, secret, crd, configmap, job, cronjob, ingress, networkpolicy)
- Unit tests in place with structured assertions (15 test functions)
- `--name`, `--image`, `--replicas` customization flags working
- `--list` flag for discoverability
- Offline-first: works without API server via `fallbackResolve()`
- Command available in kubectl builds

#### Beta

- User feedback incorporated from alpha usage
- E2E tests passing in CI
- Documentation updated on kubernetes.io with examples
- Additional resource builders based on community feedback (e.g., StatefulSet, DaemonSet)

#### GA

- Comprehensive examples for all commonly used resources
- Examples validated against multiple Kubernetes versions
- No breaking changes in output format
- Feature promoted as stable in kubectl documentation
- At least two releases between beta and GA for feedback collection

### Upgrade / Downgrade Strategy

Not applicable. This is a new, purely additive kubectl subcommand. Upgrading kubectl adds the command; downgrading removes it. No cluster state, configuration, or existing behavior is affected.

### Version Skew Strategy

The command generates YAML from in-binary Go struct builders with no API server dependency. The output uses stable API versions (`v1`, `apps/v1`, `batch/v1`, `networking.k8s.io/v1`) that are available across all supported Kubernetes versions. When a kubeconfig is available, the command may optionally attempt discovery-based kind resolution, but falls back to a local alias map if the API server is unreachable. No version skew issues arise because the output is self-contained YAML.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Other
  - Describe the mechanism: This is a new kubectl subcommand. It is enabled by building kubectl with the new code. No feature gate is required — kubectl CLI commands (like `kubectl debug`, `kubectl diff`, `kubectl events`) graduate through alpha→beta→GA without feature gates.
  - Will enabling / disabling the feature require downtime of the control plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning of a node? No

###### Does enabling the feature change any default behavior?

No. It adds a new command; no existing commands or behaviors are modified.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by using an older version of kubectl that does not include the command.

###### What happens if we reenable the feature if it was previously rolled back?

Normal operation. The command is stateless.

###### Are there any tests for feature enablement/disablement?

Unit tests verify the command is registered and functional. Since there is no feature gate, enablement/disablement is controlled by kubectl binary version.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

It cannot. This is a purely additive CLI command that generates YAML to stdout. It does not modify cluster state, running workloads, or any existing kubectl behavior.

###### What specific metrics should inform a rollback?

Not applicable. The command is a local CLI tool with no server-side component.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not applicable. The command is stateless and has no persistent state to migrate.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Not applicable. This is a local CLI command.

###### How can someone using this feature know that it is working for their instance?

Run `kubectl example pod` and verify valid YAML output is printed to stdout.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Not applicable. This is a local CLI command with no service component.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Other (treat as last resort)
  - Details: Not applicable — local CLI command.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Not applicable.

### Dependencies

None. The command uses only packages already in kubectl's dependency tree: `corev1`, `appsv1`, `batchv1`, `networkingv1`, `metav1`, and `sigs.k8s.io/yaml`.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. Examples are generated entirely from in-binary Go struct builders. No API server contact is needed. If a kubeconfig is available, the command may optionally attempt discovery-based kind resolution, but falls back to a local alias map if the API server is unreachable.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The struct builders add negligible binary size to kubectl (a few KB of compiled Go code).

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The command works fully offline. Examples are generated from in-binary struct builders. Discovery-based kind resolution gracefully falls back to a hardcoded alias map when the API server is unreachable.

## Implementation History

- **2024-12**: Initial KEP draft and PR opened ([kubernetes/enhancements#5576](https://github.com/kubernetes/enhancements/pull/5576))
- **2024-12**: Initial implementation PR opened with embedded YAML templates ([kubernetes/kubernetes#134529](https://github.com/kubernetes/kubernetes/pull/134529))
- **2026-03**: Rearchitected from YAML templates to struct-based generation using typed Kubernetes API objects (`corev1`, `appsv1`, `metav1`) with `sigs.k8s.io/yaml` marshaling. Added working `--name`, `--image`, `--replicas` flags. Rewrote tests with structured assertions (15 test functions).
- **2026-03**: Expanded resource coverage from 6 to 11 builders: added ConfigMap (`corev1`), Job (`batchv1`), CronJob (`batchv1`), Ingress (`networkingv1`), NetworkPolicy (`networkingv1`). Updated KEP with precedent analysis, release timing strategy, and plugin rationale.

## Release Timing Strategy

### Target: v1.37 Alpha

The v1.36 Enhancements Freeze has already passed, so the earliest realistic target is **v1.37** (estimated July–October 2026).

### Action Items

1. **Attend sig-cli biweekly meeting** (Wednesdays 09:00 PT) to present the KEP and request a sponsor
2. **Request KEP review** from sig-cli tech leads and chairs:
   - Chairs: @ardaguclu, @mpuckett159
   - Tech Leads: @eddiezane, @soltysh
   - Primary sponsor target: @soltysh (extensive kubectl experience, tech lead)
3. **Post to sig-cli mailing list** with KEP summary before meeting presentation
4. **Target v1.37 Enhancements Freeze** — submit enhancement issue linking to this KEP directory before the freeze date
5. **Iterate on KEP feedback** — address reviewer comments promptly to maintain momentum

### Timeline

| Milestone | Target Date | Action |
|-----------|------------|--------|
| KEP review requested | March 2026 | Post to sig-cli mailing list, attend meeting |
| KEP sponsor assigned | April–May 2026 | Work with sponsor to refine KEP |
| KEP marked `implementable` | Before v1.37 Enhancements Freeze | Get KEP approver sign-off |
| Alpha implementation merged | v1.37 code freeze | PR already open, iterate on review feedback |
| Beta (expanded resources, e2e) | v1.38 | Incorporate alpha feedback |
| GA | v1.39 | Stable after two release cycles of feedback |

## Drawbacks

- Adds a new top-level kubectl subcommand, increasing the command surface area
- Examples are opinionated and may not cover every user's specific use case
- Struct-based builders require Go code changes to add new resources (vs. dropping in a YAML file), though this is offset by compile-time type safety and API consistency

## Alternatives

1. **Embedded YAML templates**: The original implementation used `//go:embed` with `.yaml` files. This was simpler but produced static output with no real parameterization, no type safety, and risked template drift from the actual API types. Abandoned in favor of struct-based generation.

2. **Dynamic generation from OpenAPI schema**: Generate examples by walking the cluster's OpenAPI spec. More flexible but requires API server access, produces verbose output, and cannot provide sensible default values without heuristics. This is essentially what KEP-2380 attempted and it failed due to server-side complexity.

3. **External example repository**: Host examples in a separate repo and fetch them at runtime. Avoids binary size growth but introduces a network dependency and versioning complexity.

4. **kubectl plugin via krew**: Distribute as `kubectl-example` plugin. Rejected because plugins don't appear in `kubectl --help`, require separate installation, aren't tested in Kubernetes CI, and break the natural `explain` → `example` discoverability flow. See [Why Not a Plugin?](#why-not-a-plugin) for full analysis.

5. **Subcommand of explain**: `kubectl explain --example pod` instead of `kubectl example pod`. Considered but rejected to keep the UX simple and the commands orthogonal — `explain` is for schema documentation, `example` is for working manifests.

## Future Work

- Expand resource coverage: StatefulSet, DaemonSet, HorizontalPodAutoscaler, ServiceAccount
- Support `--output=json` flag for JSON output (trivial with struct-based approach since `sigs.k8s.io/yaml` supports both)
- Community-contributed examples via a plugin mechanism for custom resource types
- Integration with `kubectl explain` to show examples inline with field documentation
- Version-aware examples that adapt to the target cluster's API capabilities
