<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

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
# KEP-5489: Default StorageClass per Volume Access Mode

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
    - [Story 1: The Cluster Administrator](#story-1-the-cluster-administrator)
    - [Story 2: The Application Developer](#story-2-the-application-developer)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Implementation Logic](#implementation-logic)
  - [Example Walkthrough](#example-walkthrough)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Downgrade Strategy](#downgrade-strategy)
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
    - [Priority + Field Selector](#priority--field-selector)
    - [External Mutating Admission Webhook](#external-mutating-admission-webhook)
  - [Rationale for Annotation-Based Approach](#rationale-for-annotation-based-approach)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

Currently, Kubernetes allows administrators to define a single, cluster-wide default `StorageClass`. This default is set using the `storageclass.kubernetes.io/is-default-class` annotation. This annotation-based mechanism is simple and non-disruptive, and has become the standard way to indicate the default `StorageClass` in a cluster. However, this single-default approach is insufficient for clusters that use diverse storage backends. For example, a `ReadWriteMany` (RWX) access mode typically requires a different storage provider (like NFS) than a `ReadWriteOnce` (RWO) access mode (like a standard block device). When the default `StorageClass` points to an RWO provider, users requesting RWX volumes without specifying a class will encounter provisioning failures.

This KEP proposes an enhancement that builds directly on the existing annotation-based design. It introduces a new annotation, modeled after `storageclass.kubernetes.io/is-default-class`, to allow cluster administrators to define a default `StorageClass` for each specific volume access mode (`ReadWriteOnce`, `ReadWriteMany`, and `ReadOnlyMany`). When a PVC is created without a `storageClassName`, the system will select the appropriate default based on the `accessModes` requested in the claim. This enables more intelligent and automated volume provisioning, reduces user error, and simplifies the user experience in clusters with heterogeneous storage systems.

## Motivation

The current mechanism for defining a default `StorageClass` in Kubernetes is limited to a single, cluster-wide default. A cluster administrator can only mark one `StorageClass` with the `storageclass.kubernetes.io/is-default-class` annotation. While simple, this model fails to address the needs of modern, heterogeneous clusters where different storage backends are required to serve different volume access modes.

A common scenario is a cluster that offers both block storage for `ReadWriteOnce` (RWO) volumes and a shared file system (like NFS) for `ReadWriteMany` (RWX) volumes. In this setup, an administrator is forced to choose one as the default. If the RWO `StorageClass` is the default, any user who creates a `PersistentVolumeClaim` (PVC) requesting RWX access without explicitly naming a `storageClassName` will experience a provisioning failure. This forces users to have specific knowledge of the underlying storage infrastructure, defeating the purpose of a seamless "default" experience.

This limitation has been a long-standing issue within the community, as evidenced by the extensive discussion in [Kubernetes Issue #89911](https://github.com/kubernetes/kubernetes/issues/89911), which was first opened in 2020. The sustained engagement on this issue demonstrates a clear and persistent demand from users and cluster administrators for a more flexible and intelligent defaulting mechanism.

### Goals

* To allow cluster administrators to specify a default `StorageClass` for each distinct volume access mode (`ReadWriteOnce`, `ReadWriteMany`, `ReadOnlyMany`, etc.).
* To improve the user experience by automatically provisioning volumes from the correct storage backend based on the access mode requested in a `PersistentVolumeClaim` (PVC), without requiring the user to specify a `storageClassName`.
* To reduce provisioning errors in clusters with multiple types of storage systems.
* To maintain backward compatibility with the existing single-default `StorageClass` mechanism, which will serve as a fallback.
* To define a deterministic defaulting behavior for PVCs that list multiple access modes. The system will use a fixed priority order (`ReadWriteMany` > `ReadOnlyMany` > `ReadWriteOnce` > `ReadWriteOncePod`) to select a default `StorageClass` if the PVC requests multiple modes for which defaults exist.

### Non-Goals

* This KEP will not introduce new `API` objects. The proposed solution will be implemented using annotations on the existing `StorageClass` object.
* This KEP will not change the behavior for `PersistentVolumeClaims` (PVCs) that explicitly set a `storageClassName`.
* This KEP will not alter the validation logic that ensures a provisioner supports the access modes it is being asked to provision for.

## Proposal

This KEP proposes enhancing the `PersistentVolumeClaim` (PVC) admission process to support defining a default `StorageClass` based on the volume access mode. **The design is intentionally based on the existing annotation mechanism (`storageclass.kubernetes.io/is-default-class`) that is already used to indicate the global default.** We will introduce a new annotation, following the same pattern, that can be applied to `StorageClass` objects to designate them as the default for a specific access mode.

**This design leverages the proven, annotation-based approach already familiar to Kubernetes administrators.** The proposed annotation is `storageclass.kubernetes.io/is-default-class-for-access-mode`. A cluster administrator can apply this annotation to multiple `StorageClass` objects, each with a different access mode value, such as `"ReadWriteOnce"`, `"ReadWriteMany"`, `"ReadOnlyMany"`, or `"ReadWriteOncePod"`.

The desired outcome is a more intelligent defaulting mechanism within the `kube-apiserver`. When a user creates a `PVC` without explicitly defining a `storageClassName`, the admission controller will perform the following logic:

1.  Inspect the `accessModes` field of the incoming `PVC`.
2.  For each access mode listed in the `PVC`, find any `StorageClass` designated as its default.
3.  **Tie-Breaking Logic:** If multiple `StorageClass` objects are found to be the default for the same access mode, the system will select the one with the most recent `metadata.creationTimestamp` as the sole candidate for that mode.
4.  **Priority Selection:** From the list of unique candidates (one per access mode), the system will apply the following fixed priority hierarchy to select the final `StorageClass`:
    1.  `ReadWriteMany`
    2.  `ReadOnlyMany`
    3.  `ReadWriteOnce`
    4.  `ReadWriteOncePod`
5.  If no mode-specific default is found, the system will fall back to the current behavior and look for a `StorageClass` with the global default annotation (`storageclass.kubernetes.io/is-default-class: "true"`).

### User Stories (Optional)

#### Story 1: The Cluster Administrator

A cluster administrator manages a `Kubernetes` environment that serves diverse workloads. Some applications require `ReadWriteOnce` (RWO) block storage for databases, while others need `ReadWriteMany` (RWX) shared file storage for collaborative tasks.

The administrator annotates their `ssd-block-storage` class as the default for `ReadWriteOnce` and their `nfs-shared-storage` class as the default for `ReadWriteMany`.

As a result, users of the cluster can now create `PersistentVolumeClaims` by specifying only the access mode they require. `Kubernetes` automatically provisions the correct type of storage, reducing support requests and simplifying the developer workflow.

#### Story 2: The Application Developer

An application developer is deploying a new service that requires a `ReadWriteOnce` volume for its data. The developer is not familiar with the specific names of the `StorageClasses` available in the cluster.

They create a simple `PersistentVolumeClaim` manifest, omitting the `storageClassName` field:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: app-data-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
```
When this manifest is applied, the `Kubernetes API server` sees the `ReadWriteOnce` request, finds the default `StorageClass` configured for that mode, and automatically provisions the correct volume. The application deploys successfully without the developer needing to know infrastructure-specific details.

### Notes/Constraints/Caveats (Optional)

1.  **Tie-Breaking with `creationTimestamp`**: In the event that multiple `StorageClass` objects are incorrectly annotated as the default for the exact same access mode, the system will break the tie by selecting the object that was created most recently. While this provides deterministic behavior, administrators should aim to have only one default per access mode to ensure clarity.

2.  **Hard-Coded Priority**: The priority order for resolving defaults among different access modes is part of the `Kubernetes control plane` logic and is not configurable by the cluster administrator.

3.  **Precedence Over Global Default**: The new, mode-specific default annotation will always be evaluated before the existing global default annotation. The global default is only used as a fallback.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

The implementation of this proposal will be confined to the existing DefaultStorageClass admission controller, which is part of the kube-apiserver. No new API kinds or fields will be introduced.

**New Annotation**

A new annotation will be used to designate a StorageClass as the default for a specific access mode. This is modeled after the existing `storageclass.kubernetes.io/is-default-class` annotation, ensuring consistency and ease of adoption.

* **Key**: `storageclass.kubernetes.io/is-default-class-for-access-mode`

* **Value**: A string representing a single access mode, e.g., `ReadWriteOnce`, `ReadWriteMany`, `ReadOnlyMany`, or `ReadWriteOncePod`.

**Example `StorageClass` Manifests:**

```yaml
# Default for ReadWriteOnce volumes
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: ssd-storage-rwo
  annotations:
    storageclass.kubernetes.io/is-default-class-for-access-mode: "ReadWriteOnce"
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-ssd
---
# Default for ReadWriteMany volumes
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: nfs-storage-rwx
  annotations:
    storageclass.kubernetes.io/is-default-class-for-access-mode: "ReadWriteMany"
provisioner: [example.com/nfs](https://example.com/nfs)
```

### Implementation Logic

The core sorting and selection algorithm will be implemented in a new, reusable function, **`GetDefaultClassByAccessModes`**, located in the utility package **`kubernetes/pkg/volume/util/storageclass.go`**.

This function will be called from two places to ensure consistent behavior:

1.  **`DefaultStorageClass` Admission Controller (`kube-apiserver`):** The primary path for defaulting.
2.  **`PersistentVolume` Controller (`kube-controller-manager`):** A fallback path to ensure defaulting occurs even if the admission controller is not run.

The **`GetDefaultClassByAccessModes`** function will perform the following steps:
1.  **Candidate Collection:** It takes the PVC's `accessModes` as input and creates a list of all `StorageClass` objects that are annotated as a default for **any** of the requested modes.
2.  **Sorting and Selection:** The list of candidates is sorted using two criteria:
    * **Primary Sort Key:** The hard-coded access mode priority order (`ReadWriteMany` > `ReadOnlyMany` > `ReadWriteOnce` > `ReadWriteOncePod`).
    * **Secondary Sort Key:** The `metadata.creationTimestamp` of the `StorageClass` (newer first).
3.  **Return Value or Fallback:**
    * If the sorted candidate list is **not empty**, the function returns the **top item** from the list.
    * If the sorted candidate list is **empty**, the function will then call the existing **`GetDefaultClass`** function and return its result.

### Example Walkthrough

Consider a cluster with the following three `StorageClass` objects:

* `sc-rwo`: Annotated as default for `ReadWriteOnce`, created at `2025-08-21T10:00:00Z`.
* `sc-rox`: Annotated as default for `ReadOnlyMany`, created at `2025-08-21T11:00:00Z`.
* `sc-global`: Annotated as the global default (`is-default-class: "true"`), created at `2025-08-21T09:00:00Z`.

A user submits the following `PersistentVolumeClaim`:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: multi-mode-pvc
spec:
  accessModes:
    - ReadWriteOnce
    - ReadOnlyMany
  resources:
    requests:
      storage: 10Gi
```
**The admission controller will perform these steps:**
1.  It finds two candidate `StorageClasses` matching the requested modes: `sc-rwo` and `sc-rox`.
2.  It sorts this list of two candidates. `sc-rox` (`ReadOnlyMany`) has a higher priority than `sc-rwo` (`ReadWriteOnce`), so it comes first in the sorted list.
3.  The controller picks the top item from the sorted list, which is `sc-rox`.
4.  The PVC is mutated to set `spec.storageClassName: "sc-rox"`. The global default (`sc-global`) is never considered.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

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

* **New Tests in `kubernetes/pkg/volume/util/storageclass.go`:** A new test suite will be created for the `GetDefaultClassByAccessModes` function. These tests will validate the full code path, including:
    * The core sorting logic based on access mode priority and `creationTimestamp`.
    * The fallback to the `GetDefaultClass` function when no mode-specific candidates are found. The real `GetDefaultClass` logic will be used, not a mock.

* **Updated Controller Tests:** Existing unit tests for the **`DefaultStorageClass` Admission Controller** and the **`PersistentVolume` Controller** will be updated to validate the complete defaulting behavior. They will call the `GetDefaultClassByAccessModes` utility function with various `StorageClass` configurations and assert the correct final outcome for the `PersistentVolumeClaim`.


- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

Integration tests will be added to verify the behavior of the `DefaultStorageClass` admission controller within a test instance of the `kube-apiserver`. These tests will create `StorageClass` and `PersistentVolumeClaim` objects and then assert that the `storageClassName` field is defaulted as expected.

The following scenarios will be covered:

* **Mode-Specific Defaulting:** A `PVC` requesting a single access mode (e.g., `ReadWriteOnce`) is correctly defaulted to the `StorageClass` annotated specifically for that mode.

* **Priority-Based Selection:** A `PVC` requesting multiple access modes (e.g., `ReadWriteOnce` and `ReadWriteMany`) is correctly defaulted to the `StorageClass` with the highest priority (`ReadWriteMany` in this case).

* **Timestamp Tie-Breaking:** When two `StorageClass` objects are annotated as the default for the exact same access mode, a new `PVC` is correctly defaulted to the one that was created more recently.

* **Fallback to Global Default:** When a `PVC` requests an access mode for which no mode-specific default exists, it correctly falls back and is defaulted to the `StorageClass` marked as the global default.

* **No Operation:** A `PVC` that already has its `spec.storageClassName` explicitly set is not modified by the admission controller.

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

For the **Alpha** release, a focused suite of end-to-end (e2e) tests will be added to `test/e2e/storage`. These tests are designed to validate the complete user journey, from `PersistentVolumeClaim` (PVC) creation to successful `PersistentVolume` (PV) provisioning and binding, ensuring the feature works correctly in a fully functional cluster.

While exhaustive permutation testing of the defaulting logic will be covered by unit and integration tests, the e2e tests will confirm the most critical user-facing scenarios:

* **Happy Path for a High-Priority Mode:** A test will configure a default `StorageClass` for the **`ReadWriteMany`** access mode. It will then create a PVC requesting `ReadWriteMany` and verify that the correct `StorageClass` is assigned and that the volume is successfully provisioned and bound. This validates the primary functionality from start to finish.

* **Backward Compatibility and Fallback:** A test will be added to ensure seamless interoperability with the existing default mechanism. It will configure only a legacy global default `StorageClass` (`storageclass.kubernetes.io/is-default-class: "true"`) and create a PVC. The test will verify that the PVC correctly **falls back** to using the global default and is provisioned successfully.

* **Priority Logic Verification:** A single test will be implemented to validate the priority selection mechanism in a real cluster. It will configure default `StorageClass` objects for both `ReadWriteOnce` and `ReadWriteMany`. A PVC requesting both modes will be created, and the test will assert that the higher-priority `ReadWriteMany` `StorageClass` is chosen for provisioning.

* **Non-Interference with Explicit Claims:** A critical test will verify that the new defaulting logic **does not interfere** with existing, standard behavior. It will create a PVC that explicitly sets a `storageClassName` and confirm that it is provisioned using that specific `StorageClass`, ignoring all configured defaults (both mode-specific and global).

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved 

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

This enhancement is implemented purely in the control plane (`kube-apiserver` and `kube-controller-manager`) and does not affect existing `PersistentVolumeClaims`, `PersistentVolumes`, or the data plane. The strategy is designed to be seamless and non-disruptive.

---
#### Upgrade Strategy

The new defaulting logic is additive and backward compatible.

* **To maintain previous behavior:** No action is required from a cluster administrator. Upon upgrading, the new logic will be active, but if no `StorageClass` objects are annotated with the new `storageclass.kubernetes.io/is-default-class-for-access-mode` key, the system will find no mode-specific candidates and will fall back to the existing global default mechanism. The cluster's defaulting behavior will remain unchanged.

* **To make use of the enhancement:** After upgrading, a cluster administrator can begin adding the `storageclass.kubernetes.io/is-default-class-for-access-mode` annotation to their `StorageClass` objects. The new defaulting behavior will take effect immediately for any new `PersistentVolumeClaims` created after the annotation is applied.

---
#### Downgrade Strategy

The downgrade process is safe and does not impact existing workloads or already-bound `PersistentVolumeClaims`.

When a cluster is downgraded to a version without this enhancement, the older `DefaultStorageClass` admission controller and `PersistentVolume` controller will simply ignore the new `storageclass.kubernetes.io/is-default-class-for-access-mode` annotation.

The only impact will be on the creation of **new** `PersistentVolumeClaims`: they will no longer benefit from the mode-specific defaulting logic. The system will revert entirely to the old behavior of using only the single global default `StorageClass`. No configuration changes are required on downgrade.

### Version Skew Strategy

This enhancement is confined entirely to the Kubernetes control plane (`kube-apiserver` and `kube-controller-manager`) and has no impact on node components.

* **Node Components (`kubelet`, `kube-proxy`):** These components are unaffected. They interact with `PersistentVolumeClaim` (PVC) and `PersistentVolume` (PV) objects after they have been processed by the control plane. The mechanism by which a `storageClassName` is defaulted onto a PVC is transparent to the `kubelet`. An `n-3` kubelet will see a valid PVC and will function correctly.

* **Control Plane Components:** The logic is self-contained within the `DefaultStorageClass` admission controller and the `PersistentVolume` controller.
    * During a cluster upgrade, there may be a mix of old and new `kube-apiserver` or `kube-controller-manager` instances.
    * If a PVC creation request is handled by a new controller, it will apply the new mode-specific defaulting logic.
    * If a PVC creation request is handled by an old controller, it will apply the old global-default logic.
    * In either case, a valid `PersistentVolumeClaim` object is persisted to `etcd`. The behavior will be inconsistent for the duration of the rolling upgrade, but this does not cause any component to fail or enter an error state. Once the upgrade is complete, the behavior becomes consistent across the cluster.

Because this feature only changes the defaulting logic for a field on an existing API object and does not introduce any new APIs or interactions between components, no special version skew strategy is required.

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

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: **PerAccessModeDefaultStorageClass**
  - Components depending on the feature gate:
    - **kube-apiserver**
    - **kube-controller-manager**

**Explanation:** The feature gate controls the defaulting logic in two places for maximum robustness.
1.  The **primary path** is within the `DefaultStorageClass` admission controller (`kube-apiserver`), which mutates a `PersistentVolumeClaim` (PVC) on creation.
2.  A **fallback path** exists in the `PersistentVolume` controller (`kube-controller-manager`) to handle cases where the admission controller might have been bypassed, ensuring consistent defaulting behavior.

###### Does enabling the feature change any default behavior?

Yes. Currently, a `PersistentVolumeClaim` (PVC) created without a `storageClassName` uses the single, cluster-wide default `StorageClass`.

When this feature is enabled, the behavior changes. The system will inspect the PVC's `accessModes` and assign a `StorageClass` that has been explicitly marked as the default for that specific access mode. If no mode-specific default is found, it will fall back to the old behavior of using the global default `StorageClass`. This change is intentional and is the core of the enhancement.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate and restarting the affected components (`kube-apiserver`, `kube-controller-manager`) will revert the system to the original behavior of only using the single global default `StorageClass`.

This change only affects the creation of *new* PVCs. PVCs that were already created while the feature was active will have their `storageClassName` field populated and will not be affected by the rollback. There is no impact on existing workloads or previously bound volumes.

###### What happens if we reenable the feature if it was previously rolled back?

Re-enabling the feature gate is safe. The `kube-apiserver` and `kube-controller-manager` will simply resume using the mode-specific defaulting logic for any new PVCs that are created. There is no persistent state that is affected by disabling and re-enabling the feature.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests will be added for the logic within the `PersistentVolumeClaim` admission controller and the `PersistentVolume` controller. These tests will simulate the feature gate being enabled and disabled and will assert that the correct defaulting logic is applied accordingly in each component.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout and rollback of this feature are safe and non-disruptive for existing workloads. The feature only affects the defaulting logic for newly created PersistentVolumeClaims (PVCs). Already running workloads and previously bound PVCs are not impacted. During a rolling upgrade or rollback, there may be a short period where different API servers or controllers apply different defaulting logic, leading to inconsistent assignment of `storageClassName` for new PVCs. However, this does not cause failures or impact existing objects.

###### What specific metrics should inform a rollback?

Operators should monitor for an unexpected increase in PVC provisioning failures, especially those related to missing or incompatible `StorageClass` assignments. Anomalies in PVC creation rates, error events, or user reports of unexpected storage backend selection may indicate misconfiguration or issues with the new defaulting logic and should prompt consideration of rollback.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No formal upgrade or rollback testing has been performed yet. Manual and automated tests will be planned prior to beta release to verify that enabling, disabling, and re-enabling the feature gate produces the expected defaulting behavior for new PersistentVolumeClaims, and that existing PVCs remain unaffected throughout upgrade, downgrade, and re-upgrade cycles.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No. This enhancement is additive and does not deprecate or remove any existing features, APIs, fields, or flags. The legacy global default annotation and behavior remain supported as a fallback.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

Operators can determine usage by querying for PersistentVolumeClaims (PVCs) that were created without an explicit `storageClassName` and have had the field defaulted based on access mode. This can be done by inspecting PVC objects and correlating their `storageClassName` with the configured mode-specific defaults.

###### How can someone using this feature know that it is working for their instance?

Users can verify the feature is working by creating a PVC without specifying `storageClassName` and checking that the field is automatically set to the expected StorageClass for the requested access mode. Successful provisioning and binding of the PVC further confirms correct operation.

- [x] Other
  - Details: Inspect the PVC object after creation to confirm `spec.storageClassName` is set as expected.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- 99.9% of PVCs created without a `storageClassName` should be defaulted correctly and provisioned successfully.
- Less than 1% of PVC creation requests should fail due to defaulting logic.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Other
  - Details: Monitor PVC provisioning success and failure rates using existing Kubernetes events and logs. There are no dedicated metrics for mode-specific defaulting at this time.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Metrics tracking the number of PVCs defaulted by access mode and the number of failures due to missing or ambiguous defaults would improve observability. These metrics are not currently implemented and may require future enhancements.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

No, this feature does not depend on any external or optional cluster-level services. It is implemented entirely within the Kubernetes control plane (`kube-apiserver` and `kube-controller-manager`). The feature relies only on standard Kubernetes components and resources, specifically `StorageClass` and `PersistentVolumeClaim` objects.

- [Kubernetes API server and controller manager]
  - Usage description: These components implement the defaulting logic for `storageClassName` based on access mode.
    - Impact of its outage on the feature: If either the API server or controller manager is unavailable, PVC creation and provisioning may fail or be delayed, as with any standard PVC workflow.
    - Impact of its degraded performance or high-error rates on the feature: Degraded performance may result in slower PVC defaulting and provisioning, but does not introduce new risks beyond those present in the existing PVC workflow.

No additional dependencies (such as metrics-server, cloud provider APIs, or external storage control planes) are required for the defaulting logic itself. However, successful PVC provisioning still depends on the availability and correct configuration of the underlying storage provisioners, as with standard Kubernetes storage workflows.

### Scalability

###### Will enabling / using this feature result in any new API calls?

**No**. This feature does not introduce new API calls.

###### Will enabling / using this feature result in introducing new API types?

**No**.

###### Will enabling / using this feature result in any new calls to the cloud provider?

**No**. This feature is internal to the Kubernetes control plane.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

**Yes, but trivially.**

* **API type(s):** `StorageClass`
* **Estimated increase in size:** The feature uses a new annotation (`storageclass.kubernetes.io/is-default-class-for-access-mode`). The size of this annotation is negligible (e.g., < 100 bytes) and is a one-time, administrator-driven change.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

**Yes, but the increase is negligible.** The latency for PVC creation will increase slightly, not due to new network calls, but due to a marginal increase in the computational complexity of the defaulting logic. Instead of just searching for one annotation, the logic now searches for mode-specific annotations and applies a priority sort. This additional CPU work is minimal and is not expected to impact API server SLOs.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

**No**. The `kube-apiserver` and `kube-controller-manager` will see a very small, temporary increase in CPU usage to process the more detailed defaulting logic. This increase is considered negligible.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

**No**. This feature's logic is contained entirely within the control plane and does not affect node-level resources.

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

If the API server is unavailable, new PersistentVolumeClaim (PVC) creation requests cannot be processed, so the defaulting logic will not be applied. Existing PVCs and workloads are unaffected, but new PVCs will be delayed until the API server is restored. If etcd is unavailable, the control plane cannot persist or retrieve PVC and StorageClass objects, resulting in failures for PVC creation and provisioning. Once etcd and the API server are restored, normal operation resumes.

###### What are other known failure modes?

- **No mode-specific default StorageClass configured**
  - Detection: PVCs created without a `storageClassName` and requesting an access mode with no default will fall back to the global default. If no global default exists, PVC provisioning will fail. This can be detected by monitoring PVC events for provisioning failures and by inspecting PVCs that remain in Pending state.
  - Mitigations: Ensure that at least one StorageClass is annotated as the global default or as the default for each required access mode.
  - Diagnostics: Look for events on the PVC such as `ProvisioningFailed` or messages indicating no suitable StorageClass found.
  - Testing: Unit and integration tests will cover scenarios where no mode-specific default is present.

- **Multiple StorageClasses annotated as default for the same access mode**
  - Detection: The system will select the most recently created StorageClass for that access mode. This may lead to unexpected defaulting. Detection is possible by inspecting StorageClass annotations and creation timestamps.
  - Mitigations: Administrators should ensure only one StorageClass is annotated as default for each access mode.
  - Diagnostics: Review StorageClass objects and their annotations.
  - Testing: Unit tests will verify tie-breaking logic.

- **Provisioner does not support requested access mode**
  - Detection: PVC provisioning fails, and events indicate incompatibility between the provisioner and requested access mode.
  - Mitigations: Update StorageClass configuration to use a compatible provisioner for each access mode.
  - Diagnostics: PVC events and controller logs will show errors.
  - Testing: Covered by existing e2e and integration tests.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Review PVC events and logs for provisioning failures or delays.
2. Check that StorageClass annotations are correctly configured for each access mode.
3. Verify that the API server and controller manager are healthy and running the expected version.
4. Confirm that underlying storage provisioners are available and correctly configured.
5. Inspect cluster resource usage and performance metrics to rule out control plane bottlenecks.
6. If issues persist, disable the feature gate to revert to the legacy defaulting logic and restore expected behavior for new PVCs.

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

## Drawbacks

The primary drawback of this proposal is the introduction of additional configuration complexity for cluster administrators.

* **Increased Mental Overhead:** Instead of a single, unambiguous global default `StorageClass`, administrators now have a more complex system to manage. They must be aware of mode-specific defaults, the global fallback, and the priority order used for tie-breaking. This could lead to misconfigurations where the resulting default `StorageClass` is not what was intended.

* **Non-Obvious Priority for Multiple Access Modes:** For a `PersistentVolumeClaim` that requests multiple access modes (e.g., `["ReadWriteOnce", "ReadWriteMany"]`), the defaulting logic relies on a fixed priority hierarchy (`ReadWriteMany` > `ReadOnlyMany` > `ReadWriteOnce`). This behavior, while deterministic, might not be immediately obvious to users, who might expect the first mode listed in the array to take precedence. This could lead to volumes being provisioned from an unexpected `StorageClass`.

## Alternatives

Several out-of-tree and alternative in-tree solutions were considered. While flexible, they introduce significant challenges related to ordering and compatibility with existing Kubernetes components, making them less ideal than the proposed annotation-based approach.

---

#### Standard Field on StorageClass

This approach would add a new field (e.g., `defaultForAccessMode`) directly to the `StorageClass` API, making the defaulting mechanism a first-class, versioned part of Kubernetes.

* **Pros:**
    * **Officially Supported and Discoverable:** The field would be part of the OpenAPI schema, making it visible and documented for users and tools. This improves discoverability and clarity.
    * **Easier for Tooling and Validation:** The API server could validate that only one StorageClass is default per access mode, reducing misconfiguration. Tools and IDEs could provide better support for official fields.

* **Cons:**
    * **Requires an API Change:** Adding a new field to a stable API requires a full review, versioning, and careful backward compatibility considerations. This process is slow and has a high bar for acceptance.
    * **Slower Iteration:** Changes to core APIs take longer to design, approve, and roll out. Any future improvements or fixes would also be slow.
    * **Upgrade/Downgrade Complexity:** Introducing new fields can complicate cluster upgrades and downgrades, especially if components run different versions.

---

#### Priority + Field Selector

This approach would add a `priority` field to StorageClass and allow the controller to select the highest-priority class matching the requested access mode, possibly using field selectors.

* **Pros:**
    * **Deterministic Selection:** Assigning priorities allows the system to deterministically select the highest-priority StorageClass when multiple candidates match, reducing ambiguity.
    * **Expresses Preferences:** Administrators can express nuanced preferences (e.g., prefer SSD over HDD) by adjusting priorities.

* **Cons:**
    * **Configuration Complexity:** Administrators must manage not just which StorageClasses exist, but also their relative priorities and matching logic, increasing cognitive load.
    * **Requires New API Fields and Logic:** This approach requires adding new fields and implementing logic to resolve ties and handle edge cases, increasing maintenance burden.
    * **Potential Overkill:** For most clusters, a single default per access mode is sufficient. Priority-based selection may be unnecessarily complex for common use cases.

---

#### Priority + Field Selector + Namespace Selector

This approach extends the above by also allowing a `namespaceSelector`, so that defaults can be scoped to specific namespaces, not just cluster-wide.

* **Pros:**
    * **Maximum Flexibility:** Allows administrators to define defaults that vary by namespace, supporting multi-tenant clusters where different teams or projects need different storage defaults.
    * **Supports Advanced Use Cases:** Enables sophisticated policies, such as different defaults for dev/test/prod namespaces or business units.

* **Cons:**
    * **Significant Complexity:** Configuration and implementation become much more complex. Administrators must manage priorities, selectors, and namespace scoping, increasing the risk of misconfiguration.
    * **Harder to Debug:** When defaults are determined by a combination of priority, field selectors, and namespace selectors, it can be difficult to understand or troubleshoot why a particular StorageClass was chosen.
    * **Requires More Sophisticated Admission Logic:** Implementing this logic in the API server or controllers is non-trivial and increases the risk of bugs or inconsistent behavior.

---

#### External Mutating Admission Webhook

This approach involves deploying a custom mutating admission webhook. This webhook would intercept `PersistentVolumeClaim` creation requests and inject a `storageClassName` based on the claim's `accessModes` if the field is not already set.

* **Pros:**
    * **Flexibility:** Administrators could implement any custom logic, extending beyond just access modes.
    * **Decoupling:** Keeps new defaulting logic out of the core Kubernetes codebase.

* **Cons:**
    * **Operational Overhead:** Requires administrators to deploy, manage, and monitor a critical cluster component, which could become a single point of failure for volume provisioning.
    * **Ordering Conflicts:** The primary challenge is the ordering with the built-in `DefaultStorageClass` admission controller. To solve the ordering problems reliably, we would have to migrate the existing admission controller.
    * **Incomplete Solution:** Even if admission-time ordering is resolved, this approach is incomplete because it wouldn't address the related logic in the **PersistentVolume (PV) controller**. The PV controller, which acts asynchronously to manage volume provisioning, has its own logic that is unaware of any external defaulting performed by a webhook. This separation can lead to inconsistent behavior and complicates future enhancements to the storage system.

---

#### Kubernetes Mutating Admission Policies (CEL)

This approach would leverage the built-in `MutatingAdmissionPolicy` feature, which uses the Common Expression Language (CEL) to define policies that can modify objects during admission. An administrator would create a `MutatingAdmissionPolicy` resource that inspects the `accessModes` of an incoming PVC and mutates the `storageClassName` field if it is empty.

* **Pros:**
    * **Native Kubernetes Feature:** This solution is in-tree and does not require installing third-party tools.
    * **Powerful:** CEL provides a highly flexible expression language to implement complex logic if needed.

* **Cons:**
    * **Complexity:** Writing and debugging CEL expressions is significantly more complex for administrators than applying a simple annotation. It raises the barrier for configuring what should be a straightforward behavior.
    * **Ordering Conflicts:** The challenges are the same as with a webhook, primarily the ordering with the built-in `DefaultStorageClass` admission controller.
    * **Incomplete Solution:** This approach also wouldn't address the logic in the **PV controller**. The controller's logic would remain unaware of the CEL-based mutation, creating a disconnect between the synchronous admission step and the asynchronous provisioning step. A fully integrated feature requires that all components in the storage lifecycle are aware of the defaulting mechanism.

---

### Rationale for Annotation-Based Approach

Annotations are used here to avoid introducing new API fields to a stable resource, which would require a full API review and a longer deprecation/support cycle. **This design is a direct extension of the existing `storageclass.kubernetes.io/is-default-class` annotation mechanism, which has proven effective and non-disruptive for indicating the global default.** The annotation-based approach is:

- **Non-disruptive:** It does not break existing clusters or require API version bumps.
- **Simple to implement and adopt:** Administrators can opt-in without waiting for API changes to propagate.
- **Consistent with current practice:** The existing global default mechanism also uses an annotation.

If the community prefers a first-class API field in the future, the logic and experience gained from the annotation-based approach can inform a smooth transition.
