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
    - [Attributes JSON Generation](#attributes-json-generation)
    - [Cleanup (NodeUnprepareResources)](#cleanup-nodeunprepareresources)
    - [API Changes](#api-changes)
  - [Driver Integration](#driver-integration)
  - [Workload Discovery](#workload-discovery)
  - [Schema version handling](#schema-version-handling)
  - [Metadata Lifecycle](#metadata-lifecycle)
  - [Usage Examples](#usage-examples)
    - [Example 1: Physical GPU Passthrough (KubeVirt)](#example-1-physical-gpu-passthrough-kubevirt)
    - [Example 2: Network Device (SR-IOV / DPDK)](#example-2-network-device-sr-iov--dpdk)
  - [Feature Gate](#feature-gate)
  - [Feature Maturity and Rollout](#feature-maturity-and-rollout)
    - [Alpha (v1.36)](#alpha-v136)
    - [Beta](#beta)
    - [GA](#ga)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (v1.36)](#alpha-v136-1)
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
  - [Alternative 3: Add Method to DRAPlugin Interface](#alternative-3-add-method-to-draplugin-interface)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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
Interface) mounts. Drivers provide device metadata by populating a `Metadata` field in `PrepareResult` when returning
from `PrepareResourceClaims`. The framework writes this metadata to per-request JSON files, which are mounted into
containers via CDI. Containers only see metadata for the requests they reference (via `resources.claims[].request`),
providing proper isolation when a claim has multiple requests used by different containers. Workloads like KubeVirt can
read device metadata like PCIe bus address, or mediated device UUID, from standardized file paths without requiring
custom controllers or Downward API changes.

## Motivation

Workloads that need to interact with DRA-allocated devices (like KubeVirt virtual machines) require access to device
specific metadata such as PCIe bus addresses or mediated device UUIDs. Currently, to fetch attributes from allocated
devices, users must:
1. Go to `ResourceClaimStatus` to find the request and device name
2. Look up the `ResourceSlice` with the device name to get attribute values

This complexity forces ecosystem projects like KubeVirt to build custom controllers that watch these objects and inject
attributes via annotations/labels, leading to fragile, error-prone, and racy designs.

### Goals

- Provide an easy way for DRA driver authors to make the attributes and other metadata discoverable inside the 
pods (specifically containers requesting devices).
- Minimize complexity and avoid modifications to core components like scheduler and kubelet to maintain system
  reliability and scalability
- Maintain full backward compatibility with existing DRA drivers and workloads
- Define a versioned JSON schema to ensure compatibility within versions and clear migration paths across versions
- Support metadata updates after NodePrepareResources so that drivers (e.g. network DRA) can update metadata from NRI
 hooks such as RunPodSandbox.

### Non-Goals

- Expose the entirety of `ResourceClaim`/`ResourceSlice` objects
- Provide an additional hook in DRA path to update after metadata NodePrepareResources call for metadata updates

## Proposal

This proposal introduces framework-assisted attribute exposure via CDI mounting in the DRA kubelet plugin
framework (`k8s.io/dynamic-resource-allocation/kubeletplugin`). The framework provides a command-line
flag (e.g. `--enable-device-metadata`) that drivers integrate into their CLI.
Once integrated, operators enable or disable the metadata feature via the flag when starting the plugin
process (no driver image change required). When the flag is set, the framework enables the metadata code
path; when unset, the feature is fully disabled — no CDI specs or metadata files are generated. Metadata
files are always written under the kubelet device plugin directory for that driver
(`{kubeletDir}/plugins/{driverName}/dra-device-metadata/...`).

When enabled, the framework **always generates a CDI spec** (bind-mount) for every claim the driver
prepares, ensuring the container mount point exists regardless of when the metadata content arrives.
The metadata file itself is provided in one of two ways:

**Immediate metadata** (e.g. GPU drivers): `PrepareResourceClaims` returns with `Device.Metadata`
populated. The framework writes the metadata file immediately alongside the CDI spec.

**Deferred metadata** (e.g. network drivers like DRANet): `PrepareResourceClaims` returns without
`Device.Metadata` (or with an empty result) because device details are not yet available — network
configuration (IP addresses, interface names, MAC addresses) only becomes known after CNI runs during
`RunPodSandbox`. The CDI mount is still done at prepare time with empty metadata. The driver writes
the file via `MetadataUpdater.UpdateRequestMetadata()` from its NRI `RunPodSandbox` hook, before the
pod starts. See [Metadata Lifecycle](#metadata-lifecycle) for the full sequence.

In both cases the framework:
1. Generates CDI specs that bind-mount the driver's metadata file into containers at a well-known container path
2. Cleans up files and CDI specs during NodeUnprepareResources

On the **host**, metadata files are written under the driver's plugin directory:
`{kubeletDir}/plugins/{driverName}/dra-device-metadata/{claimNs}_{claimName}/{requestName}/metadata.json`

Within the **container**, CDI bind-mounts expose the metadata at a standardized path:
`/var/run/dra-device-attributes/{claimName}/{requestName}/{driverName}-metadata.json`

Metadata is organized per-request, ensuring containers only see metadata for the specific requests they use (via `resources.claims[].request`).

### User Stories (Optional)

#### Story 1

As a workload developer (e.g., KubeVirt), I want to automatically discover device attributes (like PCIe addresses) by
reading a JSON file at a known path, so my application can configure devices without parsing ResourceClaim/ResourceSlice
objects, calling the Kubernetes API, or requiring custom controllers.

#### Story 2

As a DRA driver author, I want to populate a `Metadata` field in `PrepareResult` to expose device attributes
in a standardized format, with the framework handling file writing, directory structure, CDI mounting, and
cleanup, so I only need to provide the metadata content.

#### Story 3

As a telco CNF developer, I want network device metadata (PCI address, interface name, IPs, MTU) to be available
inside my container, so my DPDK application can discover and bind to the correct devices without custom controllers.

### Notes/Constraints/Caveats (Optional)

- **File-based, not env vars**: Attributes are exposed as JSON files mounted via CDI, not environment variables. This
  allows for complex structured data and dynamic attribute sets.
- **Enable/disable**: Drivers wire up the framework's flag config (e.g. `--enable-device-metadata`) in their CLI; operators then control enable/disable via the flag without driver image changes. See [Feature Gate](#feature-gate).
- **No API changes**: Zero modifications to Kubernetes API types; framework/driver-side only.
- **File lifecycle**: Created during NodePrepareResources, deleted during NodeUnprepareResources; no new shared host directory.

### Risks and Mitigations

**Risk**: Exposing device attributes might leak sensitive information.

**Mitigation**: Drivers control which attributes are published via the `Device.Metadata` field. Files are created
   with 0644 permissions (readable but not writable by container). Drivers should only expose non-sensitive metadata.

**Risk**: File system clutter from orphaned attribute files.

**Mitigation**: Framework implements cleanup in NodeUnprepareResources. On driver restart, framework can perform
   best-effort cleanup by globbing and removing stale files.


**Risk**: JSON schema changes could break workloads.

**Mitigation**: The schema is versioned based on well known kubernetes CRD conventions. Applications in the pod should 
be able to read the json based on the discovered version.

## Design Details

### Framework Implementation

#### Attributes JSON Generation

Drivers provide device metadata by populating the `Metadata` field in `PrepareResult.Devices[]` when returning
from `PrepareResourceClaims`. This ensures drivers explicitly provide accurate, current device information at the
time of preparation - no auto-generation from ResourceSlice data.

**Per-Request Metadata Design**: Metadata is organized per-request, not per-claim. This ensures containers
only see metadata for the specific requests they use (via `resources.claims[].request`), providing proper
isolation when a claim has multiple requests used by different containers.

**Multi-Driver Safety**: Each driver writes metadata under its own plugin directory on the host. When a
single request with `count > 1` allocates devices from multiple DRA drivers (e.g., a `DeviceClass`
whose selectors match devices from different drivers), each driver independently writes only its own
file — no coordination or shared directory is needed. CDI bind-mounts expose each driver's file in the
container as `{driverName}-metadata.json`. Applications enumerate `*-metadata.json` files in the
container's request directory to discover all devices.

**Metadata file schema (Go types)**

   The following Go structs define the schema of the per-request `metadata.json` file written by the
   framework. The framework builds these from `PrepareResult` (claim metadata plus `Devices[].Metadata`
   and device name/pool/driver) and marshals them to JSON.

   **Staging location**: These types will be introduced in the Kubernetes staging tree under the
   DRA component. The canonical definitions will live in
   `kubernetes/kubernetes` at `staging/src/k8s.io/dynamic-resource-allocation/api/metadata`
   (and a versioned subpackage such as `api/metadata/v1alpha1` for the serialized schema). The
   framework in `staging/src/k8s.io/dynamic-resource-allocation/kubeletplugin` will reference
   these types for encoding the metadata JSON and for the driver-facing `Device.Metadata` contract.
   This keeps the metadata schema versioned and co-located with the DRA framework without adding
   new Kubernetes API types.

   ```go
   // DeviceMetadata contains metadata about devices allocated to a ResourceClaim.
   // It is serialized to versioned JSON files that can be mounted into containers.
   type DeviceMetadata struct {
       metav1.TypeMeta   `json:",inline"`
       metav1.ObjectMeta `json:"metadata,omitempty"`

       // Requests contains the device allocation information for each request
       // in the ResourceClaim.
       // +optional
       Requests []DeviceMetadataRequest `json:"requests,omitempty"`
   }
   ```

   The framework populates only `metadata.name`, `metadata.namespace`, `metadata.uid`, and
   `metadata.generation`.

   ```go
   // DeviceMetadataRequest contains metadata for a single request within a ResourceClaim.
   type DeviceMetadataRequest struct {
       // Name is the name of the request (from the ResourceClaim spec).
       Name string `json:"name"`

       // Devices contains metadata for each device allocated to this request.
       // +optional
       Devices []Device `json:"devices,omitempty"`
   }

   // Device contains metadata about a single allocated device.
   type Device struct {
       // Name is the name of the device within the pool.
       Name string `json:"name"`

       // Driver is the name of the DRA driver that manages this device.
       Driver string `json:"driver"`

       // Pool is the name of the resource pool this device belongs to.
       Pool string `json:"pool"`

       // Attributes contains the device attributes from the ResourceSlice.
       // Keys are qualified attribute names (e.g., "model", "resource.k8s.io/pciBusID").
       // Values use the Kubernetes DeviceAttribute type for consistency with the
       // resource.k8s.io API.
       // +optional
       Attributes map[resourcev1.QualifiedName]resourcev1.DeviceAttribute `json:"attributes,omitempty"`

       // NetworkData contains network-specific device data (e.g., interface name,
       // addresses, hardware address). This is populated for network devices,
       // typically during the CRI RPC RunPodSandbox, before the containers are
       // started and after the network namespace is created.
       // +optional
       NetworkData *resourcev1.NetworkDeviceData `json:"networkData,omitempty"`
   }
   ```

   `Attributes` and `NetworkData` use the same types as in the resource.k8s.io API: `resourcev1.DeviceAttribute`
   for attribute values (see ResourceSlice device attributes) and `resourcev1.NetworkDeviceData` for network
   device data (e.g. `interfaceName`, `addresses`, `hwAddress`).

   The driver API uses the same attribute and network types: `kubeletplugin.DeviceMetadata` and the
   `Device` slice in `UpdateRequestMetadata` use `resourcev1.DeviceAttribute` and `resourcev1.NetworkDeviceData`,
   so the framework can write the metadata file without type conversion.

   **Example serialized JSON** (GPU device at `{claimNs}_{claimName}/gpu-request/example.com-metadata.json`).
   Attribute values use resource.k8s.io `DeviceAttribute` serialization (typed value wrappers: `string`, `int`, `version`).

   ```json
   {
     "apiVersion": "metadata.resource.k8s.io/v1alpha1",
     "kind": "DeviceMetadata",
     "metadata": {
       "name": "my-claim",
       "namespace": "default",
       "uid": "abc-123-def-456",
       "generation": 1
     },
     "requests": [
       {
         "name": "gpu-request",
         "devices": [
           {
             "name": "gpu-0",
             "driver": "example.com",
             "pool": "node-1-gpus",
             "attributes": {
               "driverVersion": {
                 "version": "1.0.0"
               },
               "index": {
                 "int": 1
               },
               "model": {
                 "string": "LATEST-GPU-MODEL"
               },
               "uuid": {
                 "string": "gpu-93d37703-997c-c46f-a531-755e3e0dc2ac"
               },
               "resource.k8s.io/pciBusID": {
                 "string": "0000:00:01.0"
               }
             }
           }
         ]
       }
     ]
   }
   ```

**Directory structure on host** (per-driver plugin directory):

   Each driver writes metadata under its own plugin directory. No shared host directory is introduced.
   ```
   {kubeletDir}/plugins/{driverName}/dra-device-metadata/
   └── {claimNamespace}_{claimName}/
       ├── {requestName1}/
       │   └── metadata.json
       └── {requestName2}/
           └── metadata.json
   ```

   When a single request has devices from multiple drivers (e.g., `count: 2` with a cross-driver
   `DeviceClass`), each driver writes its own file in its own plugin directory:
   ```
   {kubeletDir}/plugins/example.com/dra-device-metadata/
   └── default_my-claim/
       └── accel/
           └── metadata.json

   {kubeletDir}/plugins/bar.com/dra-device-metadata/
   └── default_my-claim/
       └── accel/
           └── metadata.json
   ```

**Why per-driver plugin directory (not a shared host directory)**:

   An earlier design used a shared `/var/run/dra-device-attributes/` directory on the host. Using each
   driver's existing plugin directory instead has several advantages:

   - **Portability**: The path inherits kubelet's `--root-dir` configuration, which already handles
     differences across Linux distributions and potentially Windows. A hardcoded `/var/run/...` path
     is not portable.
   - **No duplicate configuration**: Drivers already know their plugin directory. A shared directory
     would require every driver (and the framework) to duplicate a separate configuration option.
   - **Scoped cleanup**: When a driver is uninstalled, its entire plugin directory can be removed.
     With a shared directory, cleanup must identify and remove individual files belonging to a
     specific driver — the same problem CDI files already have, but simpler is better.
   - **No cross-driver coordination**: Each driver writes only in its own directory. No risk of
     races, permission conflicts, or naming collisions between drivers.

**Generate CDI spec** (one per driver per request): Each driver creates a CDI spec that bind-mounts
   its metadata file from the driver's plugin directory into the container at the well-known container
   path. This avoids overlapping directory mounts when multiple drivers serve the same request.
   ```json
   {
     "cdiVersion": "0.3.0",
     "kind": "{driverName}/metadata",
     "devices": [
       {
         "name": "{claimUID}_{requestName}",
         "containerEdits": {
           "mounts": [
             {
               "hostPath": "{kubeletDir}/plugins/{driverName}/dra-device-metadata/{claimNamespace}_{claimName}/{requestName}/metadata.json",
               "containerPath": "/var/run/dra-device-attributes/{claimName}/{requestName}/{driverName}-metadata.json",
               "options": ["ro", "bind"]
             }
           ]
         }
       }
     ]
   }
   ```
   Note: Each driver mounts only its own metadata file. The host path is under the driver's own
   plugin directory; the container path uses a standardized convention. When multiple drivers serve
   the same request, each creates a separate CDI spec targeting a different container file path —
   no mount conflicts. Containers only see metadata files for requests they reference via
   `resources.claims[].request`.

**CDI device ID**: `{driverName}/metadata={claimUID}_{requestName}`

**Container visibility**: A container with `resources.claims[].request: gpu-request` sees
`/var/run/dra-device-attributes/my-claim/gpu-request/{driverName}-metadata.json` for each driver
that allocated devices for that request. In the common single-driver case, the container sees one file
(e.g., `example.com-metadata.json`). In the multi-driver case, it sees one file per driver.

#### Cleanup (NodeUnprepareResources)

The framework removes metadata files for the unprepared claims during cleanup

#### API Changes

**Command-line flag**: A boolean flag (e.g. `--enable-device-metadata`) is the only way to enable the feature. When the flag is enabled, the CDI mounts are done; if the driver does not provide enough data at prepare time, the mounted file will be empty, and the driver can use `MetadataUpdater` to write it later before the pod starts. When unset, the feature is off. Host path is always the kubelet device plugin directory (see [Directory structure on host](#directory-structure-on-host-per-driver-plugin-directory)); no Start() option.

**Device.Metadata field in PrepareResult**: Drivers that have metadata available at prepare time
populate this field. The framework writes the metadata file immediately. Drivers that do not have
metadata yet (e.g. network drivers) leave this field nil and write later via `MetadataUpdater`.

```go
// Device provides the CDI device IDs for one request in a ResourceClaim.
// Existing struct in k8s.io/dynamic-resource-allocation/kubeletplugin
type Device struct {
    Requests     []string
    PoolName     string
    DeviceName   string
    CDIDeviceIDs []string
    ShareID      *types.UID
    
    // Metadata contains device attributes to expose to workloads.
    // When set, the framework writes this to a JSON file mounted into containers
    // immediately after PrepareResourceClaims returns.
    // When nil, the CDI mount is still done (when feature is enabled) but the metadata field will be empty
    // the driver writes it via MetadataUpdater.UpdateRequestMetadata() before the pod starts.
    Metadata *DeviceMetadata
}

// DeviceMetadata contains device attributes to expose to workloads.
// Uses the same types as resource.k8s.io (ResourceSlice device attributes, etc.).
type DeviceMetadata struct {
    // Attributes contains device attributes. Keys should follow Kubernetes naming conventions
    // (e.g., "resource.kubernetes.io/pciBusID"). Values use resourcev1.DeviceAttribute.
    Attributes map[string]resourcev1.DeviceAttribute `json:"attributes,omitempty"`
    
    // NetworkData contains network-specific device information.
    // Populated by network DRA plugins (e.g., SR-IOV, DPDK).
    NetworkData *resourcev1.NetworkDeviceData `json:"networkData,omitempty"`
}
```

### Driver Integration

When the flag is set, drivers provide metadata in one of two ways:

- **At prepare time**: Driver populates `Device.Metadata` in `PrepareResult`. Framework writes the
  metadata file immediately. Suitable for drivers that have all device information available during
  `PrepareResourceClaims` (e.g. GPU drivers).
- **After prepare time**: Driver returns without `Device.Metadata`. The CDI mount is done but the
  metadata field is empty. Driver writes it via `MetadataUpdater.UpdateRequestMetadata()` before the pod
  starts (e.g. from an NRI hook after CNI). Suitable for network drivers like DRANet where device
  details only become available after CNI runs.

**Key benefits**: CDI spec always generated when flag is set (mount point exists); driver can provide metadata immediately or via MetadataUpdater; framework handles file writing, CDI spec generation, and cleanup.

For network DRA drivers that write metadata in two phases (initial attributes during
`PrepareResourceClaims`, then network info via NRI after CNI), see the
[Metadata Lifecycle](#metadata-lifecycle) section.

### Workload Discovery

The presence of `{driverName}-metadata.json` indicates metadata from that driver is available for this request. Absence may mean deferred metadata not yet written (e.g. network driver writing from NRI hook). Applications enumerate `*-metadata.json` in the request directory; if required metadata is missing, error or wait as appropriate.

### Schema version handling

The metadata JSON includes `apiVersion` and `kind` (e.g. `metadata.resource.k8s.io/v1alpha1`,
`DeviceMetadata`). Schema versions may evolve over time (e.g. new optional fields, or a future
v1beta1). Consumers need a consistent strategy for reading files whose version may differ from
what they were compiled against.

**Options considered:**

**Option A: Workload conditional reading (raw JSON).** The workload reads the metadata file,
inspects `apiVersion` (and optionally `kind`), and only parses if it supports that version;
otherwise it errors or skips.

- *Pros:* No library dependency. Works with any language. Single source of truth (the file itself).
- *Cons:* Every workload must implement version checks and parsing by hand. No automatic conversion
  between schema versions — if the framework starts writing v1beta1, existing workloads that only
  understand v1alpha1 break unless they are updated.

**Option B: Pod declares desired version.** The Pod specifies the metadata schema version it
supports, for example via an annotation like `resources.kubernetes.io/dra-metadata-version: v1alpha1`
or a field in the resource claim template. The framework or driver could refuse to run, or write to
a different path, if the requested version is unsupported.

- *Pros:* Explicit contract between Pod and framework. Scheduler/kubelet could theoretically validate.
- *Cons:* Requires new API or annotation design, cross-component coordination, and a migration path
  for existing Pods. Adds complexity for a file that is already self-describing via `apiVersion`.

**Option C: Go consumers use the metadata package with internal types.** The metadata package
provides internal (unversioned) types and a `runtime.Scheme` with registered conversions for each
supported version (v1alpha1, future v1beta1, etc.). Go consumers decode the JSON through the scheme
and always work with the stable internal types. The scheme handles version detection and conversion
automatically.

- *Pros:* No manual version checks. Automatic forward/backward conversion. Follows the standard
  Kubernetes API machinery pattern (`k8s.io/api` / `k8s.io/apimachinery`).
- *Cons:* Requires a Go dependency on the metadata package. Non-Go consumers must handle versioning
  themselves (though they can still benefit from the self-describing `apiVersion` field in the JSON).

**Decision:** **Option C (internal types with scheme-based decoding).** The metadata package
(`k8s.io/dynamic-resource-allocation/pkg/metadata`) provides:

- **Internal types** (`pkg/metadata`): Unversioned, canonical in-memory representation (`DeviceMetadata`,
  `DeviceMetadataRequest`, `Device`). These are the types Go consumers program against.
- **Versioned types** (`pkg/metadata/v1alpha1`, future `pkg/metadata/v1beta1`, etc.): External types
  with JSON tags, used for serialization. Each version package registers itself with the scheme and
  provides auto-generated conversion functions to/from the internal types.
- **Scheme and codec**: A `runtime.Scheme` with all versioned and internal types registered, plus
  conversion and defaulting functions. Consumers call `runtime.Decode(codec, data)` to get the
  internal `*metadata.DeviceMetadata` regardless of which version was serialized to disk.

This matches the standard Kubernetes API machinery pattern (the same approach used by
`k8s.io/api` / `k8s.io/apimachinery`). When a new version is introduced:
1. A new versioned subpackage is added (e.g. `pkg/metadata/v1beta1`).
2. Conversion functions between v1beta1 and internal types are generated.
3. Both versions are registered in the scheme.
4. Existing Go consumers continue to work unchanged — the scheme decodes either version into
   the same internal types.

**Non-Go consumers** (e.g. shell scripts, Python) can still read `apiVersion` from the JSON
and branch accordingly, but the primary versioning mechanism is the Go package with
scheme-based conversion.

**Good practice for Go consumers:**

```go
import (
    "os"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/runtime/serializer/json"
    "k8s.io/dynamic-resource-allocation/pkg/metadata"
    _ "k8s.io/dynamic-resource-allocation/pkg/metadata/v1alpha1" // register version
)

func ReadMetadata(path string) (*metadata.DeviceMetadata, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    scheme := metadata.NewScheme() // scheme with all registered versions
    codec := json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme,
        json.SerializerOptions{Yaml: false, Pretty: false, Strict: true})
    obj, _, err := codec.Decode(data, nil, nil)
    if err != nil {
        return nil, err // unknown version or malformed JSON
    }
    dm, ok := obj.(*metadata.DeviceMetadata)
    if !ok {
        return nil, fmt.Errorf("unexpected type %T", obj)
    }
    return dm, nil
}
```

The consumer always works with `*metadata.DeviceMetadata` (internal type). If the file on
disk is `v1alpha1` today or `v1beta1` tomorrow, the scheme converts automatically.

### Metadata Lifecycle

**Immediate metadata (e.g. GPU drivers)**: The metadata file is written when `PrepareResourceClaims`
returns with `Device.Metadata` populated. The file remains unchanged for the lifetime of the prepared
claim. `metadata.generation` is set to `1`.

**Deferred metadata (e.g. network drivers)**: Network DRA drivers like DRANet return an empty
`PrepareResult` from `PrepareResourceClaims` — no devices, no metadata — because the actual
device configuration (IP addresses, interface names, MAC addresses) is only available after CNI
runs during pod sandbox creation. In this case:

1. During `PrepareResourceClaims`, the framework generates the CDI spec (to set up the bind-mount
   target), but does **not** write a metadata file (there is nothing to write yet).
2. During the NRI `RunPodSandbox` hook (after CNI), the driver calls
   `MetadataUpdater.UpdateRequestMetadata()` to write the metadata file for the first time
   (`generation=1`).

```
┌─────────┐    ┌──────────────────┐    ┌────────────┐    ┌─────┐
│ Kubelet │    │ Network DRA      │    │ Containerd │    │ CNI │
│         │    │ Driver           │    │            │    │     │
└────┬────┘    └────────┬─────────┘    └──────┬─────┘    └──┬──┘
     │                  │                     │             │
     │ PrepareResources │                     │             │
     │─────────────────>│                     │             │
     │                  │ (return empty       │             │
     │                  │  result; framework  │             │
     │                  │  creates CDI spec   │             │
     │                  │  but no metadata    │             │
     │                  │  file yet)          │             │
     │<─────────────────│                     │             │
     │                  │                     │             │
     │ RunPodSandbox()  │                     │             │
     │─────────────────────────────────────-->│             │
     │                  │                     │ CNI ADD     │
     │                  │                     │────────────>│
     │                  │                     │   PodIPs    │
     │                  │                     │<────────────│
     │                  │ NRI RunPodSandbox() │             │
     │                  │<────────────────────│             │
     │                  │ (write metadata     │             │
     │                  │  with IPs, iface,   │             │
     │                  │  generation=1)      │             │
     │                  │────────────────────>│             │
     │<───────────────────────────────────────│             │
     │                  │                     │             │
     │ CreateContainers │                     │             │
     │─────────────────────────────────────-->│             │
```

The `metadata.generation` field is incremented each time the metadata is updated. For the deferred
case, the first write during the NRI hook sets `generation=1`. Subsequent updates (if any) increment
it further.

**Framework support for updates**: The framework provides a `MetadataUpdater` that drivers can use
during NRI hooks to update metadata for already-prepared claims:

```go
// MetadataUpdater allows drivers to update metadata after initial preparation.
// Used by network DRA drivers during NRI hooks when network info becomes available.
type MetadataUpdater interface {
    // UpdateRequestMetadata updates the metadata for a specific request.
    // The framework validates that this request's devices belong to the calling driver
    // (based on what was returned in PrepareResourceClaims).
    // The generation number is automatically incremented.
    UpdateRequestMetadata(
        ctx context.Context,
        claimNamespace, claimName string,
        requestName string,
        devices []Device,
    ) error
}
```

The `kubeletplugin.Helper` implements `MetadataUpdater` since it already owns the metadata writer,
CDI cache, and device state. The expected deployment model is **same-process**: the driver binary
hosts both the kubelet plugin and the NRI plugin, and passes the `Helper` directly to the NRI
plugin at construction time:

```go
func NewDriver(ctx context.Context, config *Config) (*driver, error) {
    helper, err := kubeletplugin.Start(ctx, plugin, opts...)
    // helper implements MetadataUpdater

    // NRI plugin receives helper as a direct Go reference (same process)
    nriPlugin, err := nri.StartPlugin(ctx, &nriHandler{
        metadataUpdater: helper,
    })
    return &driver{helper: helper, nriPlugin: nriPlugin}, nil
}
```

This design ensures:
- Drivers can only update metadata for requests they prepared (framework validates ownership)
- Per-driver files (`{driverName}-metadata.json`) mean no cross-driver conflicts
- Each driver's metadata file is independent
- No IPC or service discovery is needed — the NRI plugin receives the `MetadataUpdater` at creation time

Since the NRI `RunPodSandbox` hook runs after CNI but before containers start, the updated
`{driverName}-metadata.json` (with network info) is guaranteed to be present by the time the application reads it.

The files are removed during `UnprepareResourceClaims`.

**Availability guarantee**: Metadata files are available to **all** containers in the Pod
(init containers, regular containers, and sidecar containers) for the entire duration of the
container lifecycle. This is because:

1. `NodePrepareResources` runs before `RunPodSandbox`, and `RunPodSandbox` completes (including
   NRI hooks) before any init container starts. Therefore, metadata files are fully written and
   up-to-date before the first init container runs.
2. `NodeUnprepareResources` is called only after all containers have terminated, including
   sidecar containers. Therefore, metadata files remain on disk and readable throughout the
   entire Pod lifetime — no container will observe missing or partially cleaned-up files.

Any user code at any point in a container's lifecycle can read the metadata files.

### Usage Examples

#### Example 1: Physical GPU Passthrough (KubeVirt)

> **Note:** The shell/jq approach shown below is for illustration only and is **not** the preferred
> way to consume metadata. It bypasses schema version handling and will break if the serialized
> version changes. The preferred approach is to use the Go metadata package with internal types and
> scheme-based decoding (see [Schema version handling](#schema-version-handling)). A full Go example
> will be provided in the
> [dra-example-driver](https://github.com/kubernetes-sigs/dra-example-driver) repository.

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
    resources:
      claims:
      - name: pgpu
        request: gpu-request
    command:
      - /bin/sh
      - -c
      - |
        METADATA=$(cat /var/run/dra-device-attributes/physical-gpu-claim/gpu-request/gpu.example.com-metadata.json)
        PCI_BUS_ID=$(echo $METADATA | jq -r '.requests[0].devices[0].attributes["resource.kubernetes.io/pciBusID"].string')
        echo "Binding GPU at PCI $PCI_BUS_ID"
```

#### Example 2: Network Device (SR-IOV / DPDK)

> **Note:** Same caveat as above — shell/jq is shown for illustration only.
> Use the Go metadata package for production workloads.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: dpdk-app
spec:
  resourceClaims:
  - name: sriov-nic
    resourceClaimName: sriov-vf-claim
  containers:
  - name: dpdk
    image: dpdk-app:latest
    resources:
      claims:
      - name: sriov-nic
        request: network-request
    command:
      - /bin/sh
      - -c
      - |
        METADATA=$(cat /var/run/dra-device-attributes/sriov-vf-claim/network-request/sriov.example.com-metadata.json)
        PCI_BUS_ID=$(echo $METADATA | jq -r '.requests[0].devices[0].attributes["resource.kubernetes.io/pciBusID"].string')
        IFACE=$(echo $METADATA | jq -r '.requests[0].devices[0].networkData.interfaceName')
        MAC=$(echo $METADATA | jq -r '.requests[0].devices[0].networkData.hwAddress')
        IP=$(echo $METADATA | jq -r '.requests[0].devices[0].networkData.addresses[0]')
        echo "Binding DPDK to PCI $PCI_BUS_ID, interface $IFACE, MAC $MAC, IP $IP"
```

### Feature Gate

No Kubernetes feature gate. The framework provides a boolean flag (e.g. `--enable-device-metadata`) that drivers integrate into their CLI. Once integrated, operators enable/disable via the flag in deployment configuration (e.g. DaemonSet args) without a new driver image.

### Feature Maturity and Rollout

#### Alpha (v1.36)

- Framework implementation in `k8s.io/dynamic-resource-allocation/kubeletplugin`: `Metadata` field on `Device`; command line flags for driver integration
- Drivers integrate framework's flag into their CLI flags; operators control enable/disable via flag
- Unit tests (JSON generation, CDI spec, file lifecycle); integration tests with test driver; E2E for file mount and content
- Documentation for driver authors on flag integration and metadata usage

#### Beta

- Consider making metadata required (drivers must explicitly opt-out)
- Standardize JSON schema with versioning (`"schemaVersion": "v1beta1"`)
- Production-ready error handling and edge cases
- Performance benchmarks for prepare latency
- Documentation for workload developers
- Real-world validation from KubeVirt and other consumers

#### GA
- At least one stable consumer (e.g., KubeVirt) using attributes in production
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
- **Multiple requests**: Pod with claim containing multiple requests, verify separate directories per request
- **Container isolation**: Verify container only sees metadata for requests it references via `resources.claims[].request`
- **Empty metadata**: Driver writes no attributes, verify file is created with request metadata only
- **Attribute types**: Test string, bool, int, version attributes are correctly written
- **Generation number**: Verify generation increments on metadata updates
- **Cleanup**: Verify files are removed after unprepare
- **Enable/disable**: Verify feature off when flag unset; no files when `Device.Metadata` is nil

Tests will be added to `test/integration/dra/`.

#### e2e tests

E2E tests will validate real-world scenarios:

- **Metadata file mounted**: Pod can read `/var/run/dra-device-attributes/{claimName}/{requestName}/{driverName}-metadata.json`
- **Per-request isolation**: Container only sees directory for requests it references
- **Correct content**: Verify JSON contains expected apiVersion, kind, metadata, and device list
- **Multi-device request**: Verify attributes from all allocated devices in the request are included
- **CDI integration**: Verify CRI runtime correctly processes CDI device ID and mounts directory
- **Cleanup on delete**: Delete Pod, verify attribute files are removed from host

Tests will be added to `test/e2e/dra/dra.go`.

### Graduation Criteria

#### Alpha (v1.36)

- [ ] Framework implementation complete with `Device.Metadata` field in `PrepareResult`
- [ ] Framework writes metadata file when `Device.Metadata` is populated
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

**Upgrade:** No API changes; control plane unchanged. Framework is backward compatible (drivers that don't populate `Device.Metadata` unchanged). When the flag is set, drivers that provide metadata expose it; workloads without DRA or not reading the files are unaffected.

**Downgrade:** Implemented in the driver plugin framework, not kubelet. Disabling the flag or downgrading the driver: new pods won't get metadata files; existing pods keep theirs until termination.

**Rolling upgrade:** Toggle the flag per node/deployment; no cluster-wide coordination. Workloads should handle missing metadata files gracefully.

### Version Skew Strategy

No control-plane/node coordination (driver-side only). Newer driver with flag set: pods get metadata files. Older driver or flag unset: no metadata files. Test in non-production first.

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

- [x] Set or unset the framework flag (e.g. `--enable-device-metadata`) in the DRA plugin process args (e.g. DaemonSet). No driver image change required.

###### Does enabling the feature change any default behavior?

No. When enabled, new CDI mount points expose DRA device attributes as JSON files.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Unset the flag (or omit it) in the plugin deployment. New pods won't get metadata files; existing pods keep theirs until they terminate. Ensure attribute consumers have a fallback if needed.

###### What happens if we reenable the feature if it was previously rolled back?

Restores full functionality for new pods; no data migration or special handling.

###### Are there any tests for feature enablement/disablement?

Yes
- unit tests (no files when feature off or `Device.Metadata` nil)
- integration tests (flag toggle)
- E2E (files present when on, absent when off)

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

- Pod startup latency: Drivers write metadata files during NodePrepareResources, adding a small I/O overhead.
  The framework's schema validation and file writes are lightweight operations.

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
- 2026-01-12: Simplified directory structure based on wg-device-management feedback:
  removed `driverNames` file and `.supported/` markers to avoid race conditions
  between drivers and simplify workload discovery
- 2026-01-25: Redesigned driver integration based on review feedback:
  - Removed `MetadataWriter` helper approach (relied on timing assumptions)
  - Added `Metadata` field to `PrepareResult.Device` struct
  - Drivers now explicitly provide metadata when returning from `PrepareResourceClaims`
  - Added Alternative 3 documenting the interface method approach
  - Added `MetadataUpdater` for network DRA drivers to update metadata via NRI hooks
    (two-phase metadata with generation number for late-arriving network info)
- 2026-01-25: Changed from per-claim to per-request metadata organization:
  - Each request gets its own `{driverName}-metadata.json` file in `{claimNs}_{claimName}/{requestName}/`
  - CDI mounts are per-file (bind-mounting the driver's specific metadata file), so containers only see
    metadata for requests they use and there are no mount conflicts between drivers
  - Enables proper isolation when a claim has multiple requests used by different containers
  - Prevents races when a single request allocates devices from multiple drivers (`count > 1`)

## Drawbacks

1. **Filesystem dependency**: Unlike Downward API environment variables (which are managed by kubelet), this approach
   requires reliable filesystem access to `/var/run/`. Failures in file writes block Pod startup.
2. **CDI runtime requirement**: Not all CRI runtimes support CDI (or support different CDI versions). This limits
   compatibility to newer runtimes and requires clear documentation.
3. **Opaque file paths**: Workloads must discover filenames via globbing or parse JSON to match claim names. The
   Downward API approach with env vars would have been more ergonomic.
4. **No schema standardization in Alpha**: The JSON structure is subject to change. Early adopters may need to update
   their parsers between versions.
5. **Driver implementation required**: Drivers must populate `Device.Metadata` in `PrepareResult` to provide metadata.
   The Downward API approach would have been transparent to drivers.
6. **Limited discoverability**: Workloads can't easily enumerate all claims or requests; they must know the claim name
  or glob for files. Env vars would provide named variables.

## Alternatives

### Alternative 1: Downward API with ResourceSliceAttributeSelector (Original Design)

**Description**: Add `resourceSliceAttributeRef` selector to `core/v1.EnvVarSource` allowing environment variables to reference DRA device attributes. Kubelet would run a local controller watching ResourceClaims and ResourceSlices to resolve attributes at container start.

**Example**:
```yaml
env:
- name: PGPU_PCI_BUS_ID
  valueFrom:
    resourceSliceAttributeRef:
      claimName: pgpu-claim
      requestName: pgpu-request
      attribute: resource.kubernetes.io/pciBusID
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
        "PGPU_PCI_BUS_ID=0000:00:1e.0",
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

### Alternative 3: Add Method to DRAPlugin Interface

**Description**: Add a new method `GetDeviceMetadata` to the `DRAPlugin` interface that drivers must implement.
The framework would call this method after `PrepareResourceClaims` succeeds.

**Example**:
```go
type DRAPlugin interface {
    PrepareResourceClaims(ctx context.Context, claims []*resourceapi.ResourceClaim) (map[types.UID]PrepareResult, error)
    UnprepareResourceClaims(ctx context.Context, claims []NamespacedObject) (map[types.UID]error, error)
    HandleError(ctx context.Context, err error, msg string)
    
    // GetDeviceMetadata is called by framework after PrepareResourceClaims succeeds
    GetDeviceMetadata(ctx context.Context, claimUID types.UID) (*DeviceMetadata, error)
}
```

**Pros**:
- Causes compile error for existing drivers - forces awareness of the new feature
- Explicit opt-out by returning `nil, nil`
- Clear separation of concerns

**Cons**:
- Requires drivers to maintain state across two method calls (`PrepareResourceClaims` returns, then
  `GetDeviceMetadata` is called separately)
- Redundant method - driver has all the information during `PrepareResourceClaims` already
- Less elegant than returning metadata in `PrepareResult`

**Why not chosen**:
- Adding a field to `PrepareResult` is more natural since the driver has accurate device information
  at the time of preparation
- No need for drivers to maintain state across methods
- While this approach guarantees compile errors, the benefit doesn't outweigh the complexity

## Infrastructure Needed (Optional)

None. This feature will be developed within existing Kubernetes repositories:
- Metadata types (DeviceMetadata, DeviceMetadataRequest, Device) in `kubernetes/kubernetes` (staging/src/k8s.io/dynamic-resource-allocation/api/metadata and api/metadata/v1alpha1)
- Framework implementation in `kubernetes/kubernetes` (staging/src/k8s.io/dynamic-resource-allocation/kubeletplugin)
- Tests in `kubernetes/kubernetes` (test/integration/dra, test/e2e/dra, test/e2e_node)
- Documentation in `kubernetes/website` (concepts/scheduling-eviction/dynamic-resource-allocation)

Ecosystem integration (future):
- KubeVirt will consume attributes from JSON files (separate KEP in kubevirt/kubevirt)
- DRA driver examples will be updated to demonstrate `Device.Metadata` usage