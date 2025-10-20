<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

To get started with this template:

- [x] **Pick a hosting SIG.**
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
# KEP-NNNN: PDB for Multi-pod Replicas

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
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.
-->

A PodDisruptionBudget (PBD) ensures availability of a certain number of pods during voluntary disruptions (node drains). This proposal would allow PDBs to treat groups of pods (defined by the new `Workload` API from gang scheduling) as if they were individual pods for the purposes of measuring availability.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

The goal is to make PDBs usable for pod groups as defined by the `Workload` object, which are common for use cases like distributed ML workloads. Eviction or preemption of pods across multiple groups should be recognized as disrupting each of those groups, as opposed to evicting multiple pods from a single group (which only disrupts that one group). For these workloads, the health of the entire replica depends on the simultaneous availability of a certain number of pods within its group (as defined in the `Workload`'s `PodGroup`).

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Define availability for pod groups: allow application owners to define PDBs for multi-pod replicas (as defined by the `Workload` API) rather than individual pods.
- Update eviction logic to automatically detect group-based workloads. When a pod selected by a PDB is found to be part of a `Workload` (by checking for `pod.spec.workload.name`), the eviction logic will use the `Workload` and `PodGroup` definitions as the source of truth for grouping and availability.
- Maintain compatibility: ensure that common cluster operations that respect PDBs, such as `kubectl drain` and node drains initiated by `cluster-autoscaler`, follow the group-based disruption budgets.
- Preserve existing functionality: for backward compatibility, the behavior of PDBs selecting pods that are *not* part of a `Workload` (do not have `pod.spec.workload.name` set) should be unchanged.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

This change will only affect the Eviction API. The following are involuntary disruptions and do not use the Eviction API:
- Manual pod deletion
- Pods being deleted by their owning controller (e.g. during Deployment rollout)
- Node failure
- Pod cleanup due to a node being removed from the cluster
- Evictions by the Kubelet due to node pressure (e.g. memory shortage)
- Taint manager deleting NoExecute tainted pods

This proposal introduces no changes to the `PodDisruptionBudget` object, only the eviction logic.

This change will not affect the behavior of workload controllers for `Deployment`, `StatefulSet`, `Workload`, etc.
- The workload controller will be responsible for setting the `workload.name` and `workload.podGroup` on the pods it manages.
- The lifecycle and recovery of a disrupted replica is the responsibility of the workload controller, this will only handle evictions.

This change will not affect scheduling.
- It is out of scope to introduce any form of or change to gang scheduling. This only handles eviction of already-scheduled pods.

Partial replica health:
-  This KEP follows the definition of multi-pod replica health from the `Workload` API, using `minCount`. A replica is considered "available" if it meets `minCount`, and "unavailable" if it does not. We are not introducing any other definition of partial health (e.g. a percentage, or requiring all pods healthy).

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->


When the Eviction API evaluates a PDB, it will check the pods selected by the PDB's `selector`.

- If a selected pod does not have `spec.workload.name` set, it is treated as an individual pod, and availability is calculated based on pod counts, preserving existing behavior.
- If a selected pod has `spec.workload.name` and `spec.workload.podGroup` set, the eviction manager will treat *groups* of pods as the atomic unit for availability.

The logic will be as follows:
- Identify the `Workload` object referenced by `pod.spec.workload.name`.
- From the `workload.spec.podGroups`, find the entry matching `pod.spec.workload.podGroup`.
- This `PodGroup` object defines the group's policy. The `minCount` (from `policy.gang.minCount`) defines the minimum number of pods required for one replica of that group to be healthy. The `replicas` field defines how many such groups are expected.
- The PDB's `minAvailable` or `maxUnavailable` will be interpreted in terms of these `PodGroup` replicas, not individual pods.
- A `PodGroup` replica is considered "available" only if the number of healthy pods belonging to it meets its `minCount`.


### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->
*If the user is not using the `Workload` API, their process will be unaffected.*

#### Story 1: Distributed Workload

An engineer is running distributed ML training jobs using the `Workload` object. The `Workload` defines a `PodGroup` named `worker` with `replicas: 10` and `policy.gang.minCount: 8`. This means the job requires 10 replicas, and each replica consists of 8 pods.

To protect this long-running job from voluntary disruptions (like node drains), the user wants to ensure at least 9 of the 10 worker groups remain available.

The user would create a standard PDB targeting the worker pods:

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: my-training-job-workers-pdb
spec:
  minAvailable: 9
  selector:
    matchLabels:
      # Assuming pods are labeled by the workload controller
      workload: my-training-job
      pod-group: worker
```

Upon node drain, the Eviction API will:
1.  Select all pods matching the selector.
2.  Detect that these pods have `spec.workload.name: my-training-job` and `spec.workload.podGroup: worker`.
3.  Fetch the `Workload` object `my-training-job`.
4.  Identify that the PDB applies to the 'worker' `PodGroup`, which has 10 replicas.
5.  Interpret `minAvailable: 9` as "9 worker `PodGroup` replicas must remain available."
6.  A group is considered disrupted if evicting a pod would cause its healthy pod count to drop below its `minCount` (which is 8).
7.  The drain will proceed only if it does not cause the number of available worker groups to drop below 9.

This way, the job is protected to run with sufficient replicas during cluster maintenance, and the PDB definition is simple and intuitive.

#### Story 2: Cluster Maintenance

A cluster administrator frequently drains nodes for upgrades. The cluster has various workloads, including complex multi-pod applications defined by the `Workload` API.

To perform node drains safely, the administrator relies on application owners' PDBs. When the admin issues `kubectl drain <node>`, the Eviction API automatically identifies which pods belong to a `Workload`. It interprets the PDBs for those pods in terms of `PodGroup` replicas instead of individual pods, ensuring that the drain does not violate the application's group-based availability requirements.

This allows safe maintenance without causing outages, as the drain will pause if it cannot evict pods without violating a group-based PDB.

#### Setup Example

```mermaid
graph TD
    %% Define Styles for Setup Diagram
    classDef node_box fill:#ececff,stroke:#9696ff,stroke-width:2px,color:#1a1a1a
    classDef pod_box fill:#fff,stroke:#ccc,color:#1a1a1a
    classDef replica_label fill:none,stroke:none,font-weight:bold,color:#f0f0f0

    subgraph "Physical Node Setup"
        direction LR

        subgraph NodeA ["Node A"]
            G0P0("Group 0<br/>Pod 0")
            G0P1("Group 0<br/>Pod 1")
        end
        class NodeA node_box

        subgraph NodeB ["Node B"]
            G0P2("Group 0<br/>Pod 2")
            G1P0("Group 1<br/>Pod 0")
        end
        class NodeB node_box

        subgraph NodeC ["Node C"]
            G1P1("Group 1<br/>Pod 1")
            G1P2("Group 1<br/>Pod 2")
        end
        class NodeC node_box

        class G0P0,G0P1,G0P2,G1P0,G1P1,G1P2 pod_box
    end
    
    %% Logical Groupings (shown with links)
    subgraph LogicalGrouping ["Logical PodGroup Replicas"]
        direction TB
        style LogicalGrouping fill:none,stroke:none
        G0("Group 0")
        G1("Group 1")
        class G0,G1 replica_label
    end
    
    G0 -.-> G0P0
    G0 -.-> G0P1
    G0 -.-> G0P2
    
    G1 -.-> G1P0
    G1 -.-> G1P1
    G1 -.-> G1P2

    %% Style all links
    linkStyle 0,1,2,3,4,5 stroke:#888,stroke-dasharray: 5 5,stroke-width:2px
```

Assume the following setup: A `Workload` defines a `PodGroup` with `replicas: 2` and `minCount: 3`. This results in 2 logical groups (replicas) of 3 pods each. As in the diagram, Node A hosts Group 0 Pods 0 and 1, Node B hosts Group 0 Pod 2 and Group 1 Pod 0, and Node C hosts Group 1 Pods 1 and 2.

To protect at least one 3-pod group in the current system, a user could try a PDB with `minAvailable: 3` (pods). A node drain on Node B would see that there will still be 4 pods remaining afterwards, which satisfies `minAvailable: 3`, and proceed. Technically the PDB was honored, but now both groups have a failing pod (each is missing one pod and no longer meets `minCount: 3`), and the entire application fails.

After this change, a PDB with `minAvailable: 1` (interpreted as 1 group) would be evaluated. The eviction logic would identify the 2 `PodGroup` replicas. It would determine that evicting the pods on Node B would cause both replicas to become unavailable, violating the PDB (`minAvailable: 1`), and the node drain would safely stop before eviction.

```mermaid
graph TD
    %% Define Styles for Flowchart Diagram
    classDef action fill:#e6f3ff,stroke:#66b3ff,stroke-width:2px,color:#111
    classDef decision fill:#fff0e6,stroke:#ff9933,stroke-width:2px,color:#111
    classDef pdb_spec fill:#ffccff,stroke:#cc00cc,stroke-width:2px,color:#111
    classDef outcome_bad fill:#fff0f0,stroke:#ffaaaa,stroke-width:2px,color:#111
    classDef outcome_good fill:#f0fff0,stroke:#aaffaa,stroke-width:2px,color:#111
    classDef process fill:#f0f0f0,stroke:#ccc,color:#111

    StartDrain("kubectl drain<br/>node-b initiated")
    class StartDrain action

    StartDrain --> PDB_Type{Which PDB logic applies?}

    PDB_Type -- "Traditional PDB" --> PDB_Old(PDB Spec:<br/>minAvailable 3 pods)
    class PDB_Old pdb_spec

    PDB_Type -- "Workload-Aware PDB (with KEP)" --> PDB_New(PDB Spec:<br/>minAvailable 1 group)
    class PDB_New pdb_spec

    %% --- Traditional PDB Flow ---
    PDB_Old --> CalcPods(Calculate<br/>available pods)
    class CalcPods process

    CalcPods --> CheckPods{Are remaining pods<br/>>= 3?}
    class CheckPods decision

    CheckPods -- "Yes (4 >= 3)" --> DrainSuccess("Drain Proceeds:<br/>Node B pods evicted")
    class DrainSuccess action

    DrainSuccess --> AppDown("Application State:<br/><b>Both groups fail</b><br/>(Technically PDB honored,<br/>but intent violated)")
    class AppDown outcome_bad

    %% --- Multipod PDB Flow ---
    PDB_New --> DetectGroups(Detect pods belong<br/>to a Workload)
    class DetectGroups process

    DetectGroups --> CalcReplicas(Calculate<br/>available groups)
    class CalcReplicas process

    CalcReplicas --> CheckReplicas{Are remaining groups<br/>>= 1?}
    class CheckReplicas decision

    CheckReplicas -- "No (0 >= 1)" --> DrainBlocked("Drain Blocked:<br/>Eviction prevented")
    class DrainBlocked action

    DrainBlocked --> AppHealthy("Application State:<br/><b>Both groups healthy</b><br/>(PDB intent<br/>fully protected)")
    class AppHealthy outcome_good
```

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

#### Background on multi-pod replicas (LWS)

In this KEP, the LeaderWorkerSet (LWS) is used as the primary example of a multi-pod replica system. The LWS API allows users to manage a group of pods together as if they were a single pod, by specifying a template for a "leader" pod and each "worker" pod and the number of pods in the group (size). This is useful in cases where a leader process coordinates multiple worker processes, particularly in AI/ML distributed workloads for model training and inference.

All worker pods are treated the same: they are created from the same template, operated on in parallel, and if any workers fail the group is considered failing. A LeaderWorkerSet object will specify "replicas," which are not additional pods (group size), but additional leader+workers pod groups. For unique identification, each worker has an index, and each replica of the group has an index. The pods have various labels providing information as seen in the [docs](https://lws.sigs.k8s.io/docs/reference/labels-annotations-and-environment-variables/).

#### Background on the `Workload` API

In this KEP, the `Workload` object from the gang scheduling API is the source of truth for pod grouping. The `Workload` API allows users to define complex, multi-component applications.

A `Workload` object contains a list of `PodGroup`s. Each `PodGroup` defines:
* `name`: A unique identifier for the group within the `Workload`.
* `replicas`: The number of instances (replicas) of this group.
* `policy`: The scheduling policy, such as `Gang`.
* `policy.gang.minCount`: The minimum number of pods required for one replica of that group.

This KEP assumes that a pod controller (like the one managing `Workload` objects) will create pods and set `pod.spec.workload.name` and `pod.spec.workload.podGroup` on each pod it creates, linking it back to the `Workload` definition. The eviction logic uses this link to read the group's requirements.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

- This feature relies on the pod's `spec.workload.name` and `spec.workload.podGroup` fields being correctly set by its managing controller. If these fields are missing, point to a non-existent `Workload` object, or are set on some pods of a replica but not others, the eviction logic will fall back to per-pod counting, which may violate the application's true availability requirements.
- One failing pod in a large group will make that group "unavailable" if it drops below its `minCount`. A small number of failing pods spread across many groups could prevent all evictions and block a node drain. This is intended behavior (as the application is unhealthy), but may be surprising to operators.
- A PDB `selector` that matches pods from multiple different `PodGroup`s (or a mix of grouped and individual pods) may have complex or unintended behavior. Users should be advised to create separate PDBs for each distinct `PodGroup` they wish to protect.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

#### Eviction Logic Flow

The core logic change is in the PDB eviction controller.
1.  Get all pods matching the PDB's `selector`.
2.  Partition this pod list into two sets:
    - `individualPods`: Pods where `spec.workloadReference` is unset or `spec.workloadReference.Name` is empty.
    - `groupedPods`: Pods where `spec.workloadReference.Name` is set.
3.  If `groupedPods` is empty, proceed with the existing per-pod availability calculation on `individualPods`. The PDB's `minAvailable`/`maxUnavailable` are interpreted as pod counts.
4.  If `groupedPods` is not empty:
    - The controller will log a warning if `individualPods` is *also* not empty, as mixing pod types in one PDB is discouraged. The calculation will proceed based *only* on the `groupedPods`.
    - The PDB's `minAvailable`/`maxUnavailable` are now interpreted as `PodGroup` replica counts.
    - Group the `groupedPods` by their `(spec.workloadReference.Name, spec.workloadReference.PodGroup)` tuple.
    - For each unique `(workload, podGroup)` tuple found:
        - Fetch the `Workload` object.
        - Find the matching `PodGroup` in its `spec.podGroups`.
        - Read `totalReplicas = podGroup.replicas`.
        - Read `minSizePerReplica = podGroup.policy.gang.minCount`.
        - Identify all pods belonging to this `PodGroup` (across all its replicas).
        - Group *these* pods by their `spec.workloadReference.PodGroupReplicaIndex`.
        - Count the number of "available" replicas: A replica is "available" if its count of healthy, non-evicting pods is `>= minSizePerReplica`.
    - The "total" for the PDB calculation is the sum of `totalReplicas` for all `PodGroup`s matched by the selector.
    - The "available" count is the sum of "available" replicas across all matched `PodGroup`s.
5.  Compare this "available" group count against the PDB's `minAvailable` or `maxUnavailable` to decide if an eviction is allowed.

#### Group Health

A `PodGroup` replica is considered **available** if its number of healthy, non-evicting pods is greater than or equal to its `policy.gang.minCount`.

This logic inherently handles missing pods: the availability calculation is based *only* on the count of *currently healthy* pods. For example, if a replica expects 10 pods (`minCount` is 8) but only 9 pods exist (1 is missing), the replica is still considered **available** as long as 8 or 9 of those existing pods are healthy. If 3 pods are missing and only 7 healthy pods exist, the replica is **unavailable** (since 7 < 8).

If any pod in an available group is targeted for eviction, the availability of that group replica must be re-evaluated (i.e., "healthy pods - 1") to see if it would drop below `minCount`.

#### Handling `minAvailable` Percentages

The `Workload` object provides the total number of `replicas` for a `PodGroup`. This allows percentage-based `minAvailable` (e.g., "80%") and `maxUnavailable` to work correctly, as the total expected number of atomic units (the `PodGroup` replicas) is known.

#### Pods without `workloadReference`

If a PDB's `selector` matches a pod that is missing the `spec.workloadReference` field (or its `Name` is empty), it will be treated as an individual pod. If the PDB matches *only* individual pods, the standard per-pod logic applies. If it matches a mix, only the grouped-pod logic will apply (and the individual pods will be ignored for the PDB calculation, with a warning).

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
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

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [x] Other
  - Describe the mechanism: the new field in the PDB spec will be optional
  - Will enabling / disabling the feature require downtime of the control
    plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

It will change the behavior of the Eviction API and kube-scheduler, but should not affect any unrelated components.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, update the PDB to remove the field.

###### What happens if we reenable the feature if it was previously rolled back?

It will be enabled again, there should not be any disruptions.

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

No

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

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

No

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->
No

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

No different behavior

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

None that are not already part of the Eviction API

###### What steps should be taken if SLOs are not being met to determine the problem?

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

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
