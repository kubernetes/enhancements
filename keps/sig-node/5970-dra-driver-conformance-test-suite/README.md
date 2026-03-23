# KEP-5970: DRA Driver Conformance Test Suite

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Device Vendor Self-Validation](#story-1-device-vendor-self-validation)
    - [Story 2: Distribution CI Gate](#story-2-distribution-ci-gate)
    - [Story 3: New Driver Development](#story-3-new-driver-development)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Conformance Test Scope](#conformance-test-scope)
  - [Test Framework and Invocation](#test-framework-and-invocation)
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
  - [Option A: Separate <code>kubernetes-sigs/dra-conformance</code> repository](#option-a-separate-kubernetes-sigsdra-conformance-repository)
  - [Option B: Extend existing e2e tests to accept external drivers](#option-b-extend-existing-e2e-tests-to-accept-external-drivers)
  - [Option C: Vendor-side test libraries (status quo)](#option-c-vendor-side-test-libraries-status-quo)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

DRA graduated to GA in v1.35, but there's no way for a device driver vendor
to verify their driver correctly implements the DRA plugin contract. The
existing conformance tests only cover API-level CRUD, and the e2e tests use
a fake driver that can't be swapped out for a real one.

This KEP adds a driver-agnostic conformance test suite that any DRA driver
can run against. It covers the basics—driver registration, publishing devices,
allocating claims, preparing/unpreparing resources, running a pod with a
device, and reusing resources after cleanup. The suite lives upstream, doesn't
depend on any specific distribution, and vendors can use it for
self-validation while distributions like OpenShift can wire it into CI
gating.

## Motivation

DRA went GA in v1.35, and we're already seeing multiple device vendors
(NVIDIA, Intel, AMD, and others) building DRA drivers. The problem is that
right now there's no common way for any of them to verify their driver
actually works correctly against the DRA plugin contract. Each vendor ends
up writing their own one-off tests—for example, NVIDIA has
[k8s-dra-driver-gpu#782](https://github.com/NVIDIA/k8s-dra-driver-gpu/issues/782)
just to test `NodePrepareResources` edge cases. That's duplicated effort
across every vendor, and there's no consistency in what gets tested.

There are e2e tests for DRA upstream, but they're all built around a fake
test driver that lives in the tree—you can't swap in a real one. The 4
conformance tests added in v1.35 don't help either; they just verify that
the API server accepts CRUD on the DRA types, without ever exercising the
driver. In practice, a vendor could pass every existing conformance test
and still not have a working driver.

The [dra-example-driver](https://github.com/kubernetes-sigs/dra-example-driver)
is what most vendors fork to start their driver, but it doesn't come with
any conformance tests they can run against their fork to know if they've
broken something.
There was the same problem in the CSI ecosystem, and it was solved with
[csi-test](https://github.com/kubernetes-csi/csi-test)—a shared test suite
any CSI driver can run against. I think DRA needs the same thing, and
now is the right time to do it before the driver ecosystem fragments
further.

### Goals

- Define the minimum set of behaviors that any conformant DRA driver must
  implement.
- Create an upstream, driver-agnostic conformance test suite that validates
  a DRA driver against these minimum behaviors.
- Enable driver vendors to self-validate during development by running the
  suite against their driver on a live cluster.
- Enable distributions (OpenShift, EKS, GKE, etc.) to run the upstream
  suite in their CI to verify driver conformance.
- Build on existing Kubernetes e2e test infrastructure (`test/e2e/dra/`,
  `drautils`, `e2econformance`, Ginkgo/Gomega).

### Non-Goals

- Testing driver-specific features (GPU memory allocation, network bandwidth,
  FPGA bitstreams, etc.). Each driver differentiates on features; this suite
  tests only the common contract.
- Performance or scale testing of drivers.
- Driver installation, upgrade, or lifecycle management testing.
- Testing optional DRA features such as admin access, multi-node allocation,
  partitionable devices, or device taints/tolerations.
- Replacing or duplicating existing DRA e2e tests that validate Kubernetes'
  own DRA implementation.
- Creating a formal certification program (this may be a future goal but is
  out of scope for this KEP).

## Proposal

Create a conformance test suite at `test/e2e/dra/conformance/` in the
`kubernetes/kubernetes` repository. The suite accepts a driver name and
DeviceClass as input and runs a set of tests against a live Kubernetes
cluster (v1.35+) where the vendor's DRA driver is already installed.

The tests validate the minimum DRA plugin contract: driver registration,
device publishing, claim allocation, node prepare/unprepare, a pod
actually running with the device, and resource reuse after cleanup. The
full list of behaviors and how each is verified is in the
[Conformance Test Scope](#conformance-test-scope) table below.

### User Stories

#### Story 1: Device Vendor Self-Validation

As a DRA device driver developer (e.g., building a GPU driver), I want to run
a standardized test suite against my driver to verify it correctly implements
the DRA plugin contract, so I can catch regressions early and ship with
confidence.

```bash
go test ./test/e2e/dra/conformance \
  --driver-name=nvidia.com/gpu \
  --device-class=gpu.nvidia.com \
  --kubeconfig=$KUBECONFIG
```

#### Story 2: Distribution CI Gate

As a Kubernetes distribution maintainer (e.g., OpenShift), I want to run an
upstream DRA driver conformance suite in my CI pipeline against real drivers
deployed on my distribution, so I can verify driver compatibility before
shipping to customers.

#### Story 3: New Driver Development

As a developer starting a new DRA driver by forking
[dra-example-driver](https://github.com/kubernetes-sigs/dra-example-driver),
I want a clear set of conformance tests that tell me exactly what behaviors
my driver must implement, so I have a concrete definition of "done" for the
minimum viable driver.

### Notes/Constraints/Caveats

- **Driver-agnostic device verification:** The suite must verify a container
  can access a device without knowing what the device is. The initial approach
  is to verify environment variables injected by the driver (all DRA drivers
  must support this via the standard CDI mechanism). Future iterations may
  support vendor-provided verification commands.

- **Single-node vs multi-node:** The minimum conformance suite requires only
  a single node with the driver installed. Multi-node scenarios (e.g.,
  network-attached resources, topology-aware scheduling) are out of scope
  for the initial version.

- **Dependency on a running driver:** Unlike the existing API CRUD conformance
  tests, these tests require a real DRA driver to be installed and running on
  the cluster. The test suite itself does not install drivers.

- **Version skew:** The DRA gRPC plugin interface has both `v1` and `v1beta1`
  versions. The conformance suite should test against the API version supported
  by the kubelet under test.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Tests may be flaky due to timing dependencies in driver registration and device preparation | Use appropriate timeouts and retry logic consistent with existing DRA e2e test patterns in `drautils` |
| Drivers may have non-standard environment variable injection | Require only the standard CDI-based env injection that all compliant drivers must support |
| Suite may become a bottleneck for driver releases if too strict | Keep the minimum scope small (7 behaviors) and allow vendors to skip optional tests via labels |
| Upstream community may not accept the suite | Engage with WG Device Management and SIG Node early; present at biweekly meetings before submitting code |

## Design Details

### Conformance Test Scope

The following table defines the minimum conformance behaviors:

| # | Test | Description | Verification |
|---|------|-------------|--------------|
| 1 | Driver Registration | Driver registers with kubelet via DRA plugin gRPC interface | Plugin is listed in kubelet's registered plugins |
| 2 | ResourceSlice Publishing | Driver publishes devices via ResourceSlice | ResourceSlice objects exist with matching driver name and at least one device |
| 3 | Device Allocation | ResourceClaim gets allocated | ResourceClaim status shows AllocationResult after scheduling |
| 4 | NodePrepareResources | Kubelet prepares devices | Pod transitions past the resource preparation phase without errors |
| 5 | Pod Runs Successfully | Pod starts with device access | Pod reaches Running state; container environment includes driver-injected variables |
| 6 | NodeUnprepareResources | Driver cleans up on pod deletion | Pod deletion completes; no FailedUnprepareDynamicResources events |
| 7 | Resource Reuse | Device is reusable after cleanup | A second pod with a new ResourceClaim for the same device class starts successfully |

### Test Framework and Invocation

The suite uses Ginkgo/Gomega, consistent with existing Kubernetes e2e tests.
Tests are tagged with `[Conformance]` and labeled with `DRADriverConformance`.

**Location:** `kubernetes/kubernetes/test/e2e/dra/conformance/`

**Invocation:**

```bash
# Run the full DRA driver conformance suite
go test ./test/e2e/dra/conformance \
  --driver-name=<driver-name> \
  --device-class=<device-class-name> \
  --kubeconfig=$KUBECONFIG

# Run with Ginkgo directly
ginkgo -v -focus="\[Conformance\]" \
  --label-filter="DRADriverConformance" \
  ./test/e2e/dra/conformance -- \
  --driver-name=<driver-name> \
  --device-class=<device-class-name>
```

**Prerequisites:**
- A running Kubernetes cluster (v1.35+)
- The vendor's DRA driver installed and running on at least one node
- At least one device available through the driver
- A DeviceClass object created for the driver

**Outputs:**
- Standard Ginkgo test output (pass/fail per test)
- JUnit XML report (for CI integration)

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

The existing DRA e2e tests in `test/e2e/dra/dra.go` and utilities in
`test/e2e/dra/utils/` serve as the foundation. No changes to existing tests
are required; the conformance suite builds on top of the existing
infrastructure.

##### Unit tests

The conformance suite is itself a test suite. Unit tests will cover:
- Test configuration parsing (driver name, device class flags)
- Helper functions for resource verification

##### Integration tests

Not applicable. The conformance suite is an e2e test suite that runs against
a live cluster.

##### e2e tests

The conformance suite *is* the e2e test deliverable. It will be validated by
running against:
- The existing upstream fake test driver (`test/e2e/dra/test-driver/`)
- The [dra-example-driver](https://github.com/kubernetes-sigs/dra-example-driver)

### Graduation Criteria

#### Alpha

- Conformance test suite implemented with all 7 minimum behaviors
- Suite runs successfully against the upstream fake test driver
- Suite runs successfully against `dra-example-driver`
- Documentation for vendors on how to run the suite
- Presented to and reviewed by WG Device Management

#### Beta

- At least 2 real-world driver vendors have run the suite and provided feedback
- All known flakiness issues resolved
- Suite integrated into at least one distribution's CI (e.g., OpenShift)
- Tests proven flake-free for at least 2 weeks on testgrid

#### GA

- At least 3 driver vendors regularly run the suite
- Suite has been stable for at least 2 releases
- Community consensus that the minimum conformance scope is correct

### Upgrade / Downgrade Strategy

Not applicable. This KEP introduces a test suite, not a runtime feature. The
test suite is versioned alongside the Kubernetes release and tests against the
DRA API version available in the cluster.

### Version Skew Strategy

The DRA plugin gRPC interface has multiple versions (`v1`, `v1beta1`). The
conformance suite tests against whatever version the kubelet under test
supports. The suite does not mandate a specific plugin API version; it
verifies behavior through the Kubernetes API (ResourceClaims, ResourceSlices,
Pod status) rather than calling the gRPC interface directly.

For clusters with version skew (e.g., kubelet n-1), the suite will still
work as long as the DRA feature is enabled and the driver is compatible with
the kubelet version.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Other
  - Describe the mechanism: This KEP introduces a test suite, not a runtime
    feature. There is nothing to enable or disable in a live cluster. The
    test suite is an external binary run against the cluster.
  - Will enabling / disabling the feature require downtime of the control
    plane? No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No.

###### Does enabling the feature change any default behavior?

No. This is a test suite, not a runtime feature.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Not applicable. The test suite is an external tool.

###### What happens if we reenable the feature if it was previously rolled back?

Not applicable.

###### Are there any tests for feature enablement/disablement?

Not applicable. The deliverable is itself a test suite.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Not applicable. The test suite does not modify cluster state beyond creating
temporary test namespaces and resources that are cleaned up after each test.

###### What specific metrics should inform a rollback?

Not applicable.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not applicable.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Not applicable. This is a test suite, not a runtime feature.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details: The test suite produces standard Ginkgo pass/fail output and
    JUnit XML reports. A passing suite indicates the driver is conformant.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Not applicable for a test suite.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Not applicable for a test suite.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. The test suite relies on existing DRA metrics already exposed by kubelet
and kube-controller-manager (e.g., `dra_operations_seconds`,
`dra_resource_claims_in_use`).

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- DRA-capable kubelet (v1.35+ with DynamicResourceAllocation enabled)
- A DRA driver installed and running on at least one node
- At least one device published by the driver via ResourceSlice

### Scalability

###### Will enabling / using this feature result in any new API calls?

The test suite creates temporary ResourceClaims, Pods, and reads
ResourceSlices during test execution. All resources are cleaned up after
each test. This is test-time behavior only and does not affect production
workloads.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No. Test resources are temporary and cleaned up.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The test suite is run externally and test resources are temporary.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. The test suite creates a small number of pods and claims per test and
cleans up after each test.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The test suite will fail with connection errors, same as any Kubernetes
client.

###### What are other known failure modes?

- Driver not installed: Tests fail at the ResourceSlice publishing check
  with a clear error indicating no ResourceSlices found for the given
  driver name.
- No devices available: Tests fail at device allocation with a timeout
  waiting for ResourceClaim allocation.
- Incorrect DeviceClass: Tests fail with a clear error when no devices
  match the specified DeviceClass.

###### What steps should be taken if SLOs are not being met to determine the problem?

Not applicable for a test suite.

## Implementation History

- 2026-02-25: Initial KEP draft created
- 2026-02-25: Proposal document shared with team for review

## Drawbacks

- **Maintenance burden:** The conformance suite has to be kept up to date
  with the DRA API—new features might mean new tests. That said, the suite
  is deliberately small and lives next to the existing DRA test code, so
  the overhead should be manageable.

- **False sense of security:** Passing conformance doesn't mean a driver
  is production-ready; it only covers the minimum contract. We'll need to
  be clear in the docs that this is a baseline, not a stamp of approval.

## Alternatives

### Option A: Separate `kubernetes-sigs/dra-conformance` repository

Instead of putting the suite in `kubernetes/kubernetes`, give it its own
repo under `kubernetes-sigs/`. That buys an independent release cycle but
means duplicating test infrastructure (`drautils`, `e2econformance`, etc.).

**Rejected because:** Starting in-tree means we get the existing test
infrastructure for free and the suite stays in sync with DRA API changes.
We can always extract it later if it outgrows its home.

### Option B: Extend existing e2e tests to accept external drivers

Modify the existing `test/e2e/dra/dra.go` tests to optionally use a
real driver instead of the fake test driver.

**Rejected because:** The existing tests are designed for testing
Kubernetes' DRA implementation and include behaviors specific to the
fake driver (failure injection, call counting). A separate conformance
suite with a clean interface is easier to maintain and use.

### Option C: Vendor-side test libraries (status quo)

Continue letting each vendor write their own validation tests.

**Rejected because:** This leads to fragmentation, duplication of effort,
inconsistent quality, and no common baseline for distributions.

## Infrastructure Needed (Optional)

- A CI job in testgrid to run the conformance suite against the upstream
  fake test driver on every PR that modifies `test/e2e/dra/`.
- Optional: A periodic CI job running the suite against
  `dra-example-driver` for additional validation.
