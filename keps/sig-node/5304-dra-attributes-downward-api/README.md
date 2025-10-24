<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-5304: DRA Device Attributes Downward API

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Framework Implementation](#framework-implementation)
    - [Attributes JSON Generation (NodePrepareResources)](#attributes-json-generation-nodeprepareresources)
    - [Cleanup (NodeUnprepareResources)](#cleanup-nodeunprepareresources)
    - [Helper Functions](#helper-functions)
  - [Driver Integration](#driver-integration)
  - [Workload Consumption](#workload-consumption)
  - [Usage Examples](#usage-examples)
    - [Example 1: Physical GPU Passthrough (KubeVirt)](#example-1-physical-gpu-passthrough-kubevirt)
    - [Example 2: vGPU with Mediated Device](#example-2-vgpu-with-mediated-device)
  - [Feature Gate](#feature-gate)
  - [Feature Maturity and Rollout](#feature-maturity-and-rollout)
    - [Alpha (v1.35)](#alpha-v135)
    - [Beta](#beta)
    - [GA](#ga)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (v1.35)](#alpha-v135-1)
    - [Alpha (v1.35)](#alpha-v135-2)
    - [Beta](#beta-1)
    - [GA](#ga-1)
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
  - [Alternative 1: Downward API with ResourceSliceAttributeSelector (Original Design)](#alternative-1-downward-api-with-resourcesliceattributeselector-original-design)
  - [Alternative 2: DRA Driver Extends CDI with Attributes (Driver-Specific)](#alternative-2-dra-driver-extends-cdi-with-attributes-driver-specific)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

This KEP proposes exposing Dynamic Resource Allocation (DRA) device attributes to workloads via CDI (Container Device
Interface) mounts. The DRA framework will provide helper functions enabling drivers to automatically generate per-claim
attribute JSON files and mount them into containers via CDI. Workloads like KubeVirt can read device metadata like
PCIe bus address, or mediated device UUID, from standardized file paths without requiring custom controllers or Downward
API changes.

## Motivation

Workloads that need to interact with DRA-allocated devices (like KubeVirt virtual machines) require access to device
specific metadata such as PCIe bus addresses or mediated device UUIDs. Currently, to fetch attributes from allocated
devices, users must:
1. Go to `ResourceClaimStatus` to find the request and device name
2. Look up the `ResourceSlice` with the device name to get attribute values

This complexity forces ecosystem projects like KubeVirt to build custom controllers that watch these objects and inject
attributes via annotations/labels, leading to fragile, error-prone, and racy designs.

### Goals

- Provide a mechanism for workloads to discover DRA device metadata within workload pods.
- Minimize complexity and avoid modifications to core components like scheduler and kubelet to maintain system
  reliability and scalability
- Provide an easy way for DRA device authors to make the attributes discoverable inside the pod.
- Maintain full backward compatibility with existing DRA drivers and workloads

### Non-Goals

- Require changes to existing DRA driver implementations
- Expose the entirety of `ResourceClaim`/`ResourceSlice` objects
- Support dynamic updates to attributes after container start
- Standardize attribute names or JSON schema in Alpha

## Proposal

This proposal introduces **framework-managed attribute JSON generation and CDI mounting** in the DRA kubelet plugin
framework (`k8s.io/dynamic-resource-allocation/kubeletplugin`). Drivers opt-in by setting the `AttributesJSON(true)`
option when starting their plugin.

When enabled, the framework automatically:
1. Generates a JSON file per claim+request containing device attributes
2. Creates a corresponding CDI spec that mounts the attributes file into containers
3. Appends the CDI device ID to the NodePrepareResources response
4. Cleans up files during NodeUnprepareResources

The workload reads attributes from the standardized path: `/var/run/dra-device-attributes/{driverName}-{claimUID}-{requestName}.json`

### User Stories (Optional)

#### Story 1

As a KubeVirt developer, I want the virt-launcher Pod to automatically discover the PCIe address of an allocated
physical GPU by reading a JSON file at a known path, so that it can construct the libvirt domain XML to pass through the
device to the virtual machine guest without requiring a custom controller.

#### Story 2

As a DRA driver author, I want to enable attribute exposure with a single configuration option (`AttributesJSON(true)`)
and let the framework handle all file generation, CDI mounting, and cleanup, so I don't need to write custom logic for
every driver.

#### Story 3

As a workload developer, I want to automatically discover device attributes inside the pod without parsing
ResourceClaim/ResourceSlice objects or calling the Kubernetes API, so my application can remain simple and portable.

### Notes/Constraints/Caveats (Optional)

- **File-based, not env vars**: Attributes are exposed as JSON files mounted via CDI, not environment variables. This
  allows for complex structured data and dynamic attribute sets.
- **Opt-in in Alpha**: Drivers must explicitly enable `AttributesJSON(true)` the framework doesn't enable iy by default.
- **No API changes**: Zero modifications to Kubernetes API types. This is purely a framework/driver-side implementation.
- **File lifecycle**: Files are created during NodePrepareResources and deleted during NodeUnprepareResources.

### Risks and Mitigations

**Risk**: Exposing device attributes might leak sensitive information.
**Mitigation**: Attributes originate from `ResourceSlice`, which is cluster-scoped. Drivers control which attributes are
   published. NodeAuthorizer ensures kubelet only accesses resources for scheduled Pods. Files are created with 0644
   permissions (readable but not writable by container).

**Risk**: File system clutter from orphaned attribute files.
**Mitigation**: Framework implements cleanup in NodeUnprepareResources. On driver restart, framework can perform
   best-effort cleanup by globbing and removing stale files.

**Risk**: CRI runtime compatibility (not all runtimes support CDI).
**Mitigation**: Document CDI runtime requirements clearly. For Alpha, target containerd 1.7+ and CRI-O 1.23+ which have
   stable CDI support. Fail gracefully if CDI is not supported.

**Risk**: JSON schema changes could break workloads.
**Mitigation**: In Alpha, document that schema is subject to change. In Beta, the JSON schema could potentially be
   standardized and versioned.

## Design Details

### Framework Implementation

#### Attributes JSON Generation (NodePrepareResources)

When `AttributesJSON` is enabled, the framework intercepts NodePrepareResources and for each claim+request:

1. **Lookup attributes**: There is already a resourceslice controller running in the plugin which has an informer/lister
   and cache. Using this cache, attributes will be looked up for a device.
2. **Generate attributes JSON**:
   ```json
   {
     "claims": [
       {
         "claimName": "my-claim",
         "requests": [
           {
             "requestName": "my-request",
             "attributes": {
               "foo": "bar",
               "resource.kubernetes.io/pciBusID": "0000:00:1e.0"
             }
           }
         ]
       }
     ]
   }
   ```
3. **Write attributes file**: `{attributesDir}/{driverName}-{claimUID}-{requestName}.json`
4. **Generate CDI spec**:
   ```json
   {
     "cdiVersion": "0.3.0",
     "kind": "{driverName}/test",
     "devices": [
       {
         "name": "claim-{claimUID}-{requestName}-attrs",
         "containerEdits": {
           "env": [],
           "mounts": [
             {
               "hostPath": "/var/run/dra-device-attributes/{driverName}-{claimUID}-{requestName}.json",
               "containerPath": "/var/run/dra-device-attributes/{driverName}-{claimUID}-{requestName}.json",
               "options": ["ro", "bind"]
             }
           ]
         }
       }
     ]
   }
   ```

5. **Write CDI spec**: `{cdiDir}/{driverName}-{claimUID}-{requestName}-attrs.json`
6. **Append CDI device ID**: Adds `{driverName}/test=claim-{claimUID}-{requestName}-attrs` to the device's
   `CdiDeviceIds` in the response

#### Cleanup (NodeUnprepareResources)

When `AttributesJSON` is enabled, the framework removes files for the unprepared claims

#### Helper Functions

The `resourceslice.Controller` gains a new method:

```go
// LookupDeviceAttributes returns device attributes (stringified) from the controller's
// cached ResourceSlices, filtered by pool and device name.
func (c *Controller) LookupDeviceAttributes(poolName, deviceName string) map[string]string
```

### Driver Integration

Drivers enable the feature by passing options to `kubeletplugin.Start()`:

```go
plugin, err := kubeletplugin.Start(ctx, driverPlugin,
    kubeletplugin.AttributesJSON(true),
    kubeletplugin.CDIDirectoryPath("/var/run/cdi"),
    kubeletplugin.AttributesDirectoryPath("/var/run/dra-device-attributes"),
)
```

### Workload Consumption

Workloads read attributes from the mounted file:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: virt-launcher-gpu
spec:
  resourceClaims:
  - name: my-gpu-claim
    resourceClaimName: physical-gpu-claim
  containers:
  - name: virt-launcher
    image: kubevirt/virt-launcher:latest
    command:
      - /bin/sh
      - -c
      - |
        # Read attributes from mounted JSON
        ATTRS_FILE="/var/run/dra-device-attributes/"$(ls /var/run/dra-device-attributes/*.json | head -1)
        PCI_ROOT=$(jq -r '.claims[0].requests[0].attributes["resource.kubernetes.io/pcieRoot"]' $ATTRS_FILE)
        echo "PCI Root: $PCI_ROOT"
        # Use PCI_ROOT to configure libvirt domain XML...
```

**File Path Convention**: `/var/run/dra-device-attributes/{driverName}-{claimUID}-{requestName}.json`

Since workloads typically know their claim name but not the UID, they can:
- Use shell globbing: `ls /var/run/dra-device-attributes/{driverName}-*.json`
- Parse the filename to extract UID and request name
- Or read all JSON files and match by `claimName` field

### Usage Examples

#### Example 1: Physical GPU Passthrough (KubeVirt)

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: vm-with-gpu
spec:
  resourceClaims:
  - name: pgpu
    resourceClaimName: physical-gpu-claim
  containers:
  - name: compute
    image: kubevirt/virt-launcher:latest
    command:
      - /bin/sh
      - -c
      - |
        ATTRS=$(cat /var/run/dra-device-attributes/gpu.example.com-*-pgpu.json)
        PCI_ROOT=$(echo $ATTRS | jq -r '.claims[0].requests[0].attributes["resource.kubernetes.io/pcieRoot"]')
        # Generate libvirt XML with PCI passthrough using $PCI_ROOT
        echo "<hostdev mode='subsystem' type='pci'><source><address domain='$PCI_ROOT' .../></source></hostdev>"
```

#### Example 2: vGPU with Mediated Device

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: vm-with-vgpu
spec:
  resourceClaims:
  - name: vgpu
    resourceClaimName: virtual-gpu-claim
  containers:
  - name: compute
    image: kubevirt/virt-launcher:latest
    command:
      - /bin/sh
      - -c
      - |
        ATTRS=$(cat /var/run/dra-device-attributes/vgpu.example.com-*-vgpu.json)
        MDEV_UUID=$(echo $ATTRS | jq -r '.claims[0].requests[0].attributes["dra.kubevirt.io/mdevUUID"]')
        # Use MDEV_UUID to configure mediated device passthrough
        echo "<hostdev mode='subsystem' type='mdev'><source><address uuid='$MDEV_UUID'/></source></hostdev>"
```

### Feature Gate

<TODO>

### Feature Maturity and Rollout

#### Alpha (v1.35)

- Opt-in only (drivers must explicitly enable `AttributesJSON(true)`)
- Framework implementation in `k8s.io/dynamic-resource-allocation/kubeletplugin`
- Helper functions for attribute lookup
- Unit tests for JSON generation, CDI spec creation, file lifecycle
- Integration tests with test driver
- E2E test validating file mounting and content
- Documentation for driver authors
- **No feature gate**: This is a framework-level opt-in, not a Kubernetes API change

#### Beta

- Opt-out (drivers must explicitly disable `AttributesJSON(false)`, otherwise it will be enabled by default)
- Standardize JSON schema with versioning (`"schemaVersion": "v1beta1"`)
- Production-ready error handling and edge cases
- Performance benchmarks for prepare latency
- Documentation for workload developers
- Real-world validation from KubeVirt and other consumers

#### GA
- Always enabled
- At least one stable consumer (e.g., KubeVirt) using attributes in production
- Schema versioning and backward compatibility guarantees
- Comprehensive e2e coverage including failure scenarios

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

#### Prerequisite testing updates

No additional prerequisite testing updates are required. Existing DRA test infrastructure will be leveraged.

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `<package>`: `<date>` - `<test coverage>`

#### Integration tests

Integration tests will cover:

- **End-to-end attribute exposure**: Create Pod with resourceClaims, verify attributes JSON is generated and mounted
- **Multiple claims**: Pod with multiple resource claims, verify separate files for each claim+request
- **Missing attributes**: ResourceSlice with no attributes, verify empty map is written
- **Attribute types**: Test string, bool, int, version attributes are correctly stringified
- **Cleanup**: Verify files are removed after unprepare
- **Opt-in behavior**: Verify files are NOT created when `AttributesJSON(false)`

Tests will be added to `test/integration/dra/`.

#### e2e tests

E2E tests will validate real-world scenarios:

- **Attributes file mounted**: Pod can read `/var/run/dra-device-attributes/{driver}-{uid}-{request}.json`
- **Correct content**: Verify JSON contains expected claim name, request name, and attributes
- **Multi-device request**: Verify attributes from all allocated devices are included
- **CDI integration**: Verify CRI runtime correctly processes CDI device ID and mounts file
- **Cleanup on delete**: Delete Pod, verify attribute files are removed from host

Tests will be added to `test/e2e/dra/dra.go`.

### Graduation Criteria

#### Alpha (v1.35)

#### Alpha (v1.35)

- [ ] Framework implementation complete with opt-in via `AttributesJSON(true)`
- [ ] Helper functions for attribute lookup implemented
- [ ] Unit tests for core logic (JSON generation, CDI spec creation, file lifecycle)
- [ ] Integration tests with test driver
- [ ] E2E test validating file mounting and content
- [ ] Documentation for driver authors published
- [ ] Known limitations documented (no schema standardization yet)


#### Beta

TBD

#### GA

TBD

### Upgrade / Downgrade Strategy

**Upgrade:**
- No Kubernetes API changes, so upgrade is transparent to control plane
- Framework changes are backward compatible: existing drivers without `AttributesJSON(true)` continue to work unchanged
- Drivers can opt-in at their own pace by adding `AttributesJSON(true)` option
- Workloads without DRA claims are unaffected
- Workloads with DRA claims but not reading attribute files are unaffected

**Downgrade:**
- Kubelet downgrade is NOT problematic: This feature is implemented entirely in the driver plugin framework, not in
  kubelet
- If downgrading the *driver* existing pods with mounted attribute files will continue to run but new pods will not have
  attribute files mounted

**Rolling upgrade:**
- Drivers can be upgraded one at a time without cluster-wide coordination
- Pods using upgraded drivers (with `AttributesJSON(true)`) get attribute files; pods using old drivers don't
- Node/kubelet upgrades do not affect this feature (it's driver-side only)
- Workloads should handle missing files gracefully

### Version Skew Strategy

**Control Plane and Node Coordination:**
- This feature primarily involves changes in driver hence no coordination needed between control plane and node

**Version Skew Scenarios:**

1. **Newer Driver**: pods created after this update will have attributes file
2. **Older Driver**: pods created by this driver will not have the attributes file

**Recommendation:**
- Test in a non-production environment first

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

- [x] Rollout a driver with `AttributesJSON(true)`

###### Does enabling the feature change any default behavior?

No. Enabling the feature adds new CDI mount points containing attributes of DRA devices in JSON format

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. The feature can be disabled by rolling back to previoud driver version or a new version with `AttributesJSON(false)`

**Consequences:**
- New Pods will not have the attributes available inside the pod
- Existing running Pods will continue to run

**Recommendation:** Before disabling, make sure attributes consumers have an alternative mechanism to lookup the attributes

###### What happens if we reenable the feature if it was previously rolled back?

Re-enabling the feature restores full functionality.

- New Pods will work correctly
- Existing Pods (created while feature was disabled) are unaffected.

No data migration or special handling is required.

###### Are there any tests for feature enablement/disablement?

Yes:
- Unit tests verify files are NOT created when `AttributesJSON(false)`
- Integration tests verify opt-in behavior with framework flag toggle
- E2E tests validate files are present with feature on, absent with feature off

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

NO

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes, but the impact should be minimal:

- Pod startup latency: Drivers must lookup attribute values before starting containers, but the impact of this is
  minimized by local informer based lookup

- The feature does not affect existing SLIs/SLOs for clusters not using DRA or for drivers not opting-in on this feature

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No significant risk of resource exhaustion.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2025-10-02: KEP created and initial proposal drafted
- 2025-10-03: KEP updated with complete PRR questionnaire responses

## Drawbacks

1. **Filesystem dependency**: Unlike Downward API environment variables (which are managed by kubelet), this approach
   requires reliable filesystem access to `/var/run/`. Failures in file writes block Pod startup.
2. **CDI runtime requirement**: Not all CRI runtimes support CDI (or support different CDI versions). This limits
   compatibility to newer runtimes and requires clear documentation.
3. **Opaque file paths**: Workloads must discover filenames via globbing or parse JSON to match claim names. The
   Downward API approach with env vars would have been more ergonomic.
4. **No schema standardization in Alpha**: The JSON structure is subject to change. Early adopters may need to update
   their parsers between versions.
5. **Driver opt-in complexity**: Drivers must understand and configure multiple framework options (`AttributesJSON`,
   `CDIDirectoryPath`, `AttributesDirectoryPath`, `ResourceSliceLister`). The Downward API approach would have been
   transparent to drivers.
6. **Limited discoverability**: Workloads can't easily enumerate all claims or requests; they must know the claim name
  or glob for files. Env vars would provide named variables.

## Alternatives

### Alternative 1: Downward API with ResourceSliceAttributeSelector (Original Design)

**Description**: Add `resourceSliceAttributeRef` selector to `core/v1.EnvVarSource` allowing environment variables to reference DRA device attributes. Kubelet would run a local controller watching ResourceClaims and ResourceSlices to resolve attributes at container start.

**Example**:
```yaml
env:
- name: PGPU_PCI_ROOT
  valueFrom:
    resourceSliceAttributeRef:
      claimName: pgpu-claim
      requestName: pgpu-request
      attribute: resource.kubernetes.io/pcieRoot
```

**Pros**:
- Native Kubernetes API integration
- Familiar pattern for users (consistent with Downward API)
- Transparent to drivers (no driver changes required)
- Type-safe API validation
- Named environment variables (no globbing required)

**Cons**:
- Requires core API changes (longer review/approval cycle)
- Adds complexity to kubelet (new controller, watches, caching)
- Performance impact on API server (kubelet watches ResourceClaims/ResourceSlices cluster-wide or per-node)
- Limited to environment variables (harder to expose complex structured data)
- Single attribute per reference (multiple env vars needed for multiple attributes)

**Why not chosen**:
- Too invasive for Alpha; requires API review and PRR approval
- Kubelet performance concerns with additional watches
- Ecosystem requested CDI-based approach for flexibility and faster iteration

### Alternative 2: DRA Driver Extends CDI with Attributes (Driver-Specific)

**Description**: Each driver generates CDI specs with custom environment variables containing attributes. No framework involvement.

**Example** (driver-generated CDI):
```json
{
  "devices": [{
    "name": "gpu-0",
    "containerEdits": {
      "env": [
        "PGPU_PCI_ROOT=0000:00:1e.0",
        "PGPU_DEVICE_ID=device-00"
      ]
    }
  }]
}
```

**Pros**:
- No framework changes
- Maximum driver flexibility
- Works today with existing DRA

**Cons**:
- Every driver must implement attribute exposure independently (duplication)
- No standardization across drivers (KubeVirt must support N different drivers)
- Error-prone (drivers may forget to expose attributes or use inconsistent formats)
- Hard to discover (workloads must know each driver's conventions)

**Why not chosen**:
- Poor user experience (no standard path or format)
- High maintenance burden for ecosystem (KubeVirt, etc.)
- Missed opportunity for framework to provide common functionality

## Infrastructure Needed (Optional)

None. This feature will be developed within existing Kubernetes repositories:
- Framework implementation in `kubernetes/kubernetes` (staging/src/k8s.io/dynamic-resource-allocation/kubeletplugin)
- Helper functions in `kubernetes/kubernetes` (staging/src/k8s.io/dynamic-resource-allocation/resourceslice)
- Tests in `kubernetes/kubernetes` (test/integration/dra, test/e2e/dra, test/e2e_node)
- Documentation in `kubernetes/website` (concepts/scheduling-eviction/dynamic-resource-allocation)

Ecosystem integration (future):
- KubeVirt will consume attributes from JSON files (separate KEP in kubevirt/kubevirt)
- DRA driver examples will be updated to demonstrate `AttributesJSON(true)` usage