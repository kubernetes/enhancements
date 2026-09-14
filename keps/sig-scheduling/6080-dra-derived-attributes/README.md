# KEP-6080: DRA Derived Attributes

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [End-User Rigidity](#end-user-rigidity)
  - [Attribute Standardization Bottleneck](#attribute-standardization-bottleneck)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Core Manifest Example: GPU &amp; NIC NUMA Alignment](#core-manifest-example-gpu--nic-numa-alignment)
  - [User Stories](#user-stories)
    - [Custom Multi-Device Grouping (GPU + NIC + CPU)](#custom-multi-device-grouping-gpu--nic--cpu)
    - [Substring / Regex Topology Extraction](#substring--regex-topology-extraction)
    - [Dynamic Capacity Tiering](#dynamic-capacity-tiering)
    - [Implicit Hardware Alignment via Naming Conventions](#implicit-hardware-alignment-via-naming-conventions)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
    - [KEP-5254 Comparison (DRA: Constraints with CEL)](#kep-5254-comparison-dra-constraints-with-cel)
    - [Interaction with KEP-5491 (List Types for Attributes)](#interaction-with-kep-5491-list-types-for-attributes)
    - [Naming and Override Semantics for Derived Attributes](#naming-and-override-semantics-for-derived-attributes)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [CEL Environment &amp; Validation](#cel-environment--validation)
  - [Scheduler Plugin Implementation](#scheduler-plugin-implementation)
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
- [Future Considerations](#future-considerations)
  - [Derived Attributes in Device Selectors](#derived-attributes-in-device-selectors)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

DRA currently relies on exact literal matching of device attributes across
requests (`matchAttributes`). While performant, literal matching is rigid
for end users. Workloads needing to pair devices across complex boundaries—
such as extracting a PCI or NUMA ID from a topology string—cannot express
these requirements inline.

This KEP introduces `derivedAttributes` to `.devices.requests`. It allows
users to synthesize virtual grouping keys on the fly using scoped, per-device
CEL expressions. The scheduler's constraint engine then evaluates these
derived keys exactly like static attributes.

## Motivation

### End-User Rigidity
Currently, DRA co-allocation requires exact literal matching of attributes via
`matchAttributes`. If a user wants to co-locate a GPU and a NIC based on a
shared PCIe locality, both device drivers must publish the exact same attribute
name and value. If one driver publishes `pcie_locality: 0` and the other
publishes `pcie_root: 0` (or embeds it in a topology string like `numa0-pcie1`),
the scheduler cannot match them. Users have no mechanism to bridge these schemas
inline.

### Attribute Standardization Bottleneck
To achieve cross-vendor or multi-driver device pairing today, hardware vendors
must agree on standardized attribute naming conventions (e.g.,
`resource.kubernetes.io/pcieRoot`).

However, core Kubernetes components like the scheduler treat these attribute
names as opaque strings during constraint matching. Forced standardization is
essentially a human-coordinated workaround for a missing API capability: **the
inability of the constraint engine to match different attribute names across
requests**. Relying on this workaround forces vendors into a slow pipeline for
API approval whenever a new pairing boundary is needed ([real example](https://github.com/kubernetes-sigs/dra-driver-nvidia-gpu/issues/1123)).
While standardized labels provide clear benefits for downstream observability
tools, **forcing their use as the sole mechanism for device alignment creates
rigid, human-coordinated dependencies rather than enabling flexible, API-driven
logic.**

While ongoing community efforts to standardize schemas are highly valuable for
long-term hardware representation, the rapidly evolving AI/ML ecosystem often
outstrips the turnaround time of formal standardization cycles. Frequently, the
physical topology information needed for co-allocation is already present on
device objects (embedded in vendor-specific strings, naming conventions, or
capacity metrics). The only bottleneck preventing end users from consuming this
data immediately is the strict requirement that attribute names match exactly
across requests.

### Goals
- Allow `.devices.requests` to define virtual grouping keys via scoped,
  per-device CEL expressions (`derivedAttributes`).
- Enable co-allocation of heterogeneous hardware across differing vendor
  attribute schemas without prior human coordination or schema standardization.
- Preserve the early-pruning performance of the scheduler's constraint engine
  by scoping CEL evaluation to individual candidate devices.

### Non-Goals
- Replacing existing static attribute matching (`matchAttributes`).
- Supporting arbitrary cross-device constraints in CEL (explored in KEP-5254
  but did not proceed due to scheduler scaling bottlenecks).

## Proposal

We propose extending `.devices.requests` with `derivedAttributes`. A derived
attribute defines a CEL expression evaluated against each candidate device
object. The resulting value acts as a virtual attribute that can be referenced
directly by existing `.devices.constraints[].matchAttribute` fields.

### Core Manifest Example: GPU & NIC NUMA Alignment

*(Note: The community is actively standardizing `numaNode` in [PR #6073](https://github.com/kubernetes/enhancements/pull/6073).
This example illustrates the API mechanics for bridging disparate schemas).*

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: gpu-numa-alignment-claim
spec:
  devices:
    requests:
    - name: gpu
      exactly:
        deviceClassName: gpu.nvidia.com
        count: 8
        # [NEW]: Compute virtual grouping key from GPU driver
        derivedAttributes:
        - name: shared-numa-node
          expression: "device.attributes['gpu.nvidia.com'].numa"
    - name: nic
      exactly:
        deviceClassName: dranet
        count: 1
        # [NEW]: Compute virtual grouping key from dranet driver
        derivedAttributes:
        - name: shared-numa-node
          expression: "device.attributes['dra.net'].numaNode"
    constraints:
    # Match the derived attribute across both requests using existing matchAttribute
    - matchAttribute: shared-numa-node
      requests: [gpu, nic]
```

### User Stories

#### Custom Multi-Device Grouping (GPU + NIC + CPU)
Users need to co-locate heterogeneous hardware (GPUs, high-speed NICs, host
CPUs) managed by independent drivers. Each driver publishes topology metadata
under different attribute names. Users can synthesize an identical virtual
grouping key across all three requests:
- **GPU**: `expression: "device.attributes['gpu.nvidia.com'].foo_key"`
- **NIC**: `expression: "device.attributes['dra.net'].bar_id"`
- **CPU**: `expression: "device.attributes['cpu.intel.com'].baz_domain"`

#### Substring / Regex Topology Extraction
Device drivers often publish monolithic topology strings (e.g.,
`topology: "numa0-pcieDomain1-nic0"`). Users can use CEL string manipulation
to extract the specific boundary needed inline:
- `expression: "device.attributes['vendor.com'].topology.split('-')[0]"`

#### Dynamic Capacity Tiering
Users want to group heterogeneous devices into matching performance tiers based
on capacity or numeric attributes. CEL comparisons can quantize continuous
ranges into discrete matching bins:
- `expression: "device.attributes['vendor.com'].memory_gb >= 80 ? 'tier-1' : 'tier-2'"`

#### Implicit Hardware Alignment via Naming Conventions
When physical topology alignment is known implicitly via device naming
conventions (e.g., `gpu0` aligns with `eth0`) rather than explicit topology
attributes, CEL can extract and match the underlying hardware index:
- **GPU**: `expression: "device.name.replace('gpu', '')"`
- **NIC**: `expression: "device.name.replace('eth', '')"`

### Notes/Constraints/Caveats

#### KEP-5254 Comparison (DRA: Constraints with CEL)
[KEP-5254 (PR #5391)](https://github.com/kubernetes/enhancements/pull/5391)
explored using CEL to express arbitrary constraints across entire device groups
(`cel.expression`). For example:

```yaml
    constraints:
    # KEP-5254 proposed evaluating CEL across an entire device group
    - cel:
        expression: "devices[0].attributes['gpu.nvidia.com'].numa == devices[1].attributes['dra.net'].numaNode"
      requests: [gpu, nic]
```

While highly flexible, evaluating expressions across entire device groups
prevented the scheduler from pruning invalid permutations early. Because the CEL
environment required candidate devices for both `gpu` and `nic` simultaneously
within the `devices` list, the scheduler had to generate combinatorial device
permutations before evaluating the constraint. This led to combinatorial
explosion during device filtering.

`derivedAttributes` resolves this by scoping CEL evaluation strictly to
individual candidate devices. The scheduler evaluates derived keys as individual
devices are filtered, maintaining the exact same early-pruning efficiency as
static attribute matching.

#### Interaction with KEP-5491 (List Types for Attributes)
[KEP-5491 (PR #5492)](https://github.com/kubernetes/enhancements/pull/5492)
introduces list-typed attributes to `ResourceSlice` (e.g., lists of strings or
integers) and redefines `matchAttribute` to evaluate as a non-empty set
intersection ($\cap v_k \neq \emptyset$), treating scalar values as single-
element lists.

`derivedAttributes` is fully forward-compatible with KEP-5491 and will inherit its
capabilities in three key areas:
1. **CEL Runtime Environment**: The CEL environment for `expression` will expose
   list-typed attributes exactly as defined in `ResourceSlice` by KEP-5491.
2. **List Return Types & Synthesis**: The allowed return types for `expression`
   will be expanded to include list types (`[]string`, `[]int64`, `[]bool`).
   Crucially, manifest authors can use CEL list literal syntax to dynamically
   synthesize a list from multiple individual scalar attributes inline (e.g.,
   `expression: "[device.attributes['v'].r1, device.attributes['v'].r2]"`).
   This allows bridging older scalar-only drivers with KEP-5491 list matching.
3. **Constraint Matching Semantics**: When `matchAttribute` evaluates a derived
   attribute that returns a list, it will adopt the exact same non-empty
   intersection matching semantics defined by KEP-5491. Scalar return values
   will be treated as single-element lists.

#### Naming and Override Semantics for Derived Attributes
We debate two design paths for validating `derivedAttributes` names:

1. **Option 1 (Allow FQDNs e.g., `vendo.com/attr`)**: Enables derived attributes
   to cleanly **override** static driver attributes of the same name, providing
   powerful inline flexibility at the slight risk of accidental shadowing.
2. **Option 2 (Strictly Bare Names e.g., `shared-numa`)**: Enforces an absolute
   syntactic boundary between static (FQDN) and derived (bare) attributes,
   eliminating shadowing risks but preventing direct attribute overrides.

Recommended: **Option 1 (Allowing FQDNs)**. Most manifest authors will naturally
choose simple bare names. Conversely, authoring an FQDN override requires
deliberate effort to duplicate a driver's schema, indicating clear user intent.
Enabling this pattern provides great flexibility.

### Risks and Mitigations
- **Risk**: A new CEL expression needs to be evaluated for each candidate device
  (in addition to any CEL expressions evaluated for device selectors).
- **Mitigation**: To prevent redundant evaluations of the same expression on
  a single device (which may be evaluated multiple times or against multiple
  requests), the scheduler plugin caches both the compiled CEL ASTs and the
  evaluated derived attribute values for each candidate device. Evaluation
  happens exactly once per candidate device per scheduling cycle, making the
  latency overhead strictly linear O(N) with the number of candidate devices.

## Design Details

### API Changes

We extend `DeviceRequest` in `resource.k8s.io/v1` with `DerivedAttributes`:

```go
// package resource

type DeviceRequest struct {
	// Existing fields...
	Name string `json:"name"`
	DeviceClassName string `json:"deviceClassName"`
	// ...

	// DerivedAttributes defines a set of virtual attributes computed via CEL expressions
	// for each candidate device.
	// +featureGate=DRADerivedAttributes
	// +listType=map
	// +listMapKey=name
	// +optional
	// +k8s:optional
	// +k8s:maxItems=8
	DerivedAttributes []DerivedAttribute `json:"derivedAttributes,omitempty"`
}

type DerivedAttribute struct {
	// Name is the identifier for this derived attribute, used in constraints.
	//
	// It has the same format as the name of attributes in a ResourceSlice.
	//
	// A domain prefix (e.g., "example.com/attribute-name") should be used if
	// the derived attribute is intended to override or shadow a static attribute
	// of the same name from a device driver. If the derived attribute is unique
	// and used solely for inline matching in constraints within the claim, a simple
	// bare name without a domain prefix (e.g., "my-derived-attribute") should
	// be used to prevent accidental shadowing and make the intent clear.
	//
	// +k8s:required
	Name QualifiedName `json:"name"`

	// Expression is a CEL expression evaluated against each candidate device.
	// The expression must evaluate to a primitive scalar (string, integer,
	// boolean, or semver) or a list of these scalars ([]string, []int64,
	// []bool, []semver) to act as a virtual grouping key. Any other return type
	// is an error and causes CEL evaluation for the device to fail.
	//
	// The expression's input is an object named "device", which carries the
	// same properties as in a CELDeviceSelector.
	//
	// When pod scheduling encounters CEL runtime errors (such as looking
	// up an attribute that isn't defined) for some devices, it will abort
	// allocation and fail scheduling for the Pod. Surfacing evaluation
	// errors immediately prevents silent topology matching failures that are
	// extremely hard to detect. A robust expression should, for example, check
	// for the existence of attributes before referencing them to avoid
	// runtime evaluation errors.
	//
	// The length of the expression must be smaller or equal to 10 Ki. The
	// cost of evaluating it is also limited based on the estimated number
	// of logical steps.
	//
	// +k8s:required
	Expression string `json:"expression"`
}
```

`DeviceConstraint` in `resource.k8s.io/v1` requires zero Go struct modifications.
Existing `MatchAttribute *string` fields will be reused to reference derived
attributes.

### CEL Environment & Validation
- **Environment**: The CEL environment for `Expression` is exactly the same
  as that for `CELDeviceSelector`, containing a single variable `device`.
- **Evaluation Order**: Derived attributes are evaluated **after** the device
  request's `CELDeviceSelector` has filtered candidate devices. They are not
  injected into the selector's CEL environment and are exclusively used for
  evaluating constraints like `matchAttribute` and `distinctAttribute`.
- **Return Type**: The CEL expression must evaluate to a scalar (string,
  integer, boolean, or semver) or a list of these scalars (`[]string`, `[]int64`,
  `[]bool`, `[]semver`).
- **Validation**: `kube-apiserver` validates the CEL syntax during
  `ResourceClaim` creation and update.
- **Runtime Error Handling**: If a CEL expression fails to evaluate on a
  candidate device at runtime (due to a missing attribute, null pointer
  reference, type mismatch, or other runtime error), the scheduler will abort
  the allocation and fail scheduling for the Pod immediately, even if other
  candidate devices or nodes evaluate successfully. This matches the behavior
  of CEL device selectors in the scheduler, where any runtime evaluation
  failure aborts allocation and fails scheduling for the Pod rather than
  silently filtering it out. Surfacing evaluation errors immediately prevents
  silent topology matching failures and ensures that broken expressions are
  detected and resolved.

### Scheduler Plugin Implementation
In `pkg/scheduler/framework/plugins/dynamicresources`:
1. The plugin compiles the CEL expressions defined in `derivedAttributes` for
   all requests in the Pod's `ResourceClaims`. Compiled ASTs are cached.
2. When evaluating candidate devices for a request, the plugin executes the
   cached CEL expressions against each `Device` object. The computed values are
   stored in a temporary map associated with the candidate device.
3. When evaluating constraints (like `matchAttribute` and `distinctAttribute`),
   the plugin implements a lookup precedence: it first checks if the attribute
   name matches a cached derived attribute on the candidate device's request;
   if not found, it falls back to looking up the static attribute on the
   `Device` object. If values do not match across the specified requests, the
   permutation is pruned.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates
None.

##### Unit tests
- `k8s.io/kubernetes/pkg/apis/resource/validation`: `2026-05-01` - `>90%`
  ([Coverage](https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit&include-filter-by-regex=k8s.io%2Fkubernetes%2Fpkg%2Fapis%2Fresource%2Fvalidation%7Ck8s.io%2Fkubernetes%2Fpkg%2Fscheduler%2Fframework%2Fplugins%2Fdynamicresources&show-stale-tests=))
  - Verify validation of `derivedAttributes` (valid/invalid CEL syntax).
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`:
  `2026-05-01` - `>80%`
  ([Coverage](https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit&include-filter-by-regex=k8s.io%2Fkubernetes%2Fpkg%2Fapis%2Fresource%2Fvalidation%7Ck8s.io%2Fkubernetes%2Fpkg%2Fscheduler%2Fframework%2Fplugins%2Fdynamicresources&show-stale-tests=))
  - Verify CEL compilation caching.
  - Verify correct CEL evaluation and constraint matching.

##### Integration tests
- `test/integration/scheduler_perf/dra/derived-attributes`:
  - Include a realistic scenario (number of attributes, complexity of the CEL
    expressions), then define a simple case for correctness checking and
    larger cases for performance measurement.

##### e2e tests
- `test/e2e/scheduling/dra.go`:
  - Add e2e tests verifying multi-request co-allocation using
    `derivedAttributes` across test device plugins.

### Graduation Criteria

#### Alpha
- Feature gate `DRADerivedAttributes` implemented.
- API validation and scheduler plugin implementation complete.
- Unit and integration tests passing.

#### Beta
- Gather feedback from DRA driver maintainers (SIG Node / SIG Network).
- Any additional e2e tests implemented and running in Testgrid canaries.
- Verify scheduler performance and latency overhead with large device counts
  using the `scheduler_perf` test cases.
- Revisit CEL compilation and evaluation caching strategy during performance
  benchmarking (e.g., evaluating whether using compiled CEL expression pointers
  as map keys offers benefits over exact expression strings; see
  [PR #140029 discussion](https://github.com/kubernetes/kubernetes/pull/140029#discussion_r3636503040)).
- Revisit `valToDeviceAttribute` conversion logic in CEL package with CEL
  experts to check type handling (e.g., considering `val.Value().(type)`,
  `traits.Lister`, or `ConvertToNative`; see
  [PR #140029 discussion](https://github.com/kubernetes/kubernetes/pull/140029#discussion_r3630068626)).
- Revisit sharing derived attribute evaluation cache across parallel node Filter
  operations to evaluate expressions once per claim rather than once per node
  (see [PR #140029 discussion](https://github.com/kubernetes/kubernetes/pull/140029#discussion_r3638735028)).
- Discuss CEL cost limit enforcement options for derived attributes with CEL
  maintainers (e.g., static cost validation vs. aggregate runtime evaluation vs.
  cost sampling; see
  [PR #140029 discussion](https://github.com/kubernetes/kubernetes/pull/140029#discussion_r3629395631)).

#### GA
- Proven adoption in deployment manifests and user documentation for real-world DRA drivers (e.g., dra-driver-cpu,
  dra-driver-nvidia-gpu, dra-driver-nvidia-tpus, dranet).
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

### Upgrade / Downgrade Strategy
- **Upgrade**: Enabling `DRADerivedAttributes` allows users to create
  `ResourceClaims` with derived attributes. Existing claims are unaffected.
- **Downgrade**: Disabling `DRADerivedAttributes` prevents creating or updating
  claims with derived attributes. Existing claims using derived attributes will
  fail validation on update, and the scheduler will ignore derived attributes
  during filtering.

### Version Skew Strategy
- `kube-apiserver` and `kube-scheduler` must both have `DRADerivedAttributes`
  enabled.
- If `kube-apiserver` is upgraded and has the feature gate enabled but
  `kube-scheduler` does not, the older scheduler will ignore `derivedAttributes`
  and treat non-FQDN `matchAttribute` strings as static attributes on the device
  objects, resulting in scheduling failures. Standard n-1 control plane version
  skew rules apply.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DRADerivedAttributes
  - Components depending on the feature gate: kube-apiserver,
    kube-controller-manager, kube-scheduler

###### Does enabling the feature change any default behavior?

No. Existing `ResourceClaims` without `derivedAttributes` are evaluated exactly
as before.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate prevents creating new `ResourceClaims` with
`derivedAttributes`. Existing claims with `derivedAttributes` will fail
validation on update, and the scheduler will ignore derived attributes during
filtering.

###### What happens if we reenable the feature if it was previously rolled back?

Existing `ResourceClaims` with `derivedAttributes` will resume being evaluated
correctly by the scheduler.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests in `pkg/apis/resource/validation` verify that
`derivedAttributes` and non-FQDN `matchAttribute` or `distinctAttribute` strings
are rejected when the feature gate is disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout failure or rollback does not impact running Pods, as DRA resource
allocation occurs during Pod scheduling. If a rollback occurs, pending Pods
referencing claims with `derivedAttributes` may fail to schedule.

###### What specific metrics should inform a rollback?

- `schedule_attempts_total` with `result="error"` or `result="unschedulable"` in
  `kube-scheduler`.
- `plugin_execution_duration_seconds` for the `dynamicresources` plugin in
  `kube-scheduler`.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Manual upgrade and rollback testing will be performed during Alpha by toggling
the feature gate on a local test cluster and verifying scheduling behavior.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

By checking for `ResourceClaim` objects where
`spec.devices.requests[*].derivedAttributes` is non-empty.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Condition name: `Allocation` condition on `ResourceClaim`. When successfully
    scheduled and allocated, the claim status reflects the allocated devices
    matching the derived constraints.
- [x] Events
  - Event Reason: `Scheduled` event on the Pod, indicating successful
    co-allocation by the scheduler.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Scheduling latency for Pods using `ResourceClaims` with `derivedAttributes`
should not increase by more than 5% compared to claims using literal
`matchAttributes`.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `plugin_execution_duration_seconds` (filter/reserve) for
    `dynamicresources` in `kube-scheduler`.
  - Components exposing the metric: `kube-scheduler`.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. Existing scheduler framework metrics and DRA controller metrics provide
sufficient observability.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. `ResourceClaim` objects will increase slightly in size when
`derivedAttributes` are defined (typically under 500 bytes per claim).

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

`kube-scheduler` latency will increase slightly to evaluate CEL expressions.
Because evaluation is scoped per device object rather than across device groups,
the overhead is linear with the number of candidate devices $O(N)$ rather than
combinatorial. Caching compiled CEL ASTs minimizes this overhead.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. `kube-scheduler` will experience a minor, negligible increase in CPU and
memory usage to compile and evaluate CEL expressions.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Standard Kubernetes behavior applies; scheduling and resource allocation cannot
proceed without the API server.

###### What are other known failure modes?

- **Failure mode**: Poorly optimized CEL expressions causing scheduler plugin
  timeouts.
  - **Detection**: Increase in `plugin_execution_duration_seconds` for
    `dynamicresources` and `schedule_attempts_total` with `result="error"`.
  - **Mitigations**: The scheduler enforces a bounded execution time for CEL
    evaluation. If an expression exceeds the limit or fails at runtime, the
    candidate device is pruned.

###### What steps should be taken if SLOs are not being met to determine the problem?

Examine `kube-scheduler` logs and metrics
(`plugin_execution_duration_seconds`) to identify if specific CEL expressions
in pending `ResourceClaims` are causing high evaluation latency.

## Implementation History

- 2026-05-15: Initial KEP draft created for Alpha in v1.37.

## Drawbacks

- Adds complexity to the `dynamicresources` scheduler plugin, which must manage
  additional CEL compilation, caching, and runtime evaluation environments
  (on top of existing ones.)

## Alternatives

- **(Status Quo) Forced Attribute Standardization**: Relying entirely on hardware
  vendors agreeing on standardized attribute naming conventions (e.g.,
  `resource.kubernetes.io/numa-node`). This is not ideal as it creates
  rigid, slow, human-coordinated dependencies rather than enabling flexible,
  API-driven co-allocation logic.

- **KEP-5254 (`MatchExpression`)**: Exploring the use of CEL to express
  arbitrary constraints across entire device groups. While offering incredible
  flexibility, evaluating expressions across entire device groups made it
  difficult for the scheduler to prune invalid permutations early, leading to
  combinatorial explosion during device filtering.

## Future Considerations

### Derived Attributes in Device Selectors

In the current design, `derivedAttributes` are evaluated **after** the device
request's `CELDeviceSelector` has filtered candidate devices. As a result,
derived attributes cannot be referenced within the selector's CEL expression
itself.

While making derived attributes available within selectors would improve
usability (by avoiding the need to duplicate complex mapping logic across the
selector and the derived attribute), it introduces significant complexities:

1. **Performance Overhead**: Evaluating derived attributes before selectors
   forces early evaluation on *all* candidate devices.
2. **CEL Environment Ambiguity**: Referencing derived attributes in the CEL
   environment requires resolving namespace collisions or expanding the
   standard schema (e.g., introducing `device.derivedAttributes`).

This functionality is excluded from the current KEP to prioritize scheduling
performance and simplify the initial implementation. It may be explored in a
future enhancement if the community identifies a strong need for this usability
improvement.
