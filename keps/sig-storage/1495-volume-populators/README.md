# Volume Populators

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [VM Images](#vm-images)
    - [Backup/Restore](#backuprestore)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
- [Alternatives](#alternatives)
  - [Validating webhook](#validating-webhook)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

In Kubernetes 1.12, we added the `DataSource` field to the PVC spec. The field was implemented
as a `TypedLocalObjectReference` to give flexibility in the future about what objects could be
data sources for new volumes.

Since then, we have allowed only two things to be the source of a
new volume -- existing PVCs (indicating a user's intent to clone the volume) and snapshots
(indicating the user's intent to restore a snapshot). Implementation of these two data sources
relies on a CSI plugin to do the actual work, and no generic implementation exists for cloning
volumes or restoring snapshots.

Since the original design of the `DataSource` API field, we have been aware of the desire to
populate volumes with data from other sources, both CSI-specific and generic (compatible with
any CSI plugin). For new CSI-specific data sources, the path forward is clear, but for other
sources of data, which I call "Volume Populators" we don't have a mechanism. The main
problem is that current API validation logic uses a white list of object types, driven by
feature gates for each object type. This approach won't scale to generic populators, which
will by their nature be too numerous and varied.

For historical reasons (not intentional), the validation of the `DataSource` field caused
unsupported objects to be ignored, rather than rejected, and for the request to yield an
empty volume, as if no data source had been requested. It's not possible to update this
behavior in a backwards-compatible way, which forces us to use a different mechanism for
populators.

This proposal recommends that we replace the `DataSource` field with a new `DataSourceRef`
field with expanded semantics. The existing `DataSource` field will be deprecated and
maintained only for backwards compatibility. The new `DataSourceRef` field will allow
objects other than existing data sources (PVC and VolumeSnapshots), and crucially will
never ignore inputs.

Because volume populators will be implemented as new out-of-tree controllers, the in-tree
validation of data sources will be limited to rejecting definitely-invalid values. All
potentially-valid data sources will result in successful PVC creation, followed by either
the PVC binding to a volume with the desired contents, or events explaining why that hasn't
happened yet.

We will introduce a new CRD to register valid datasource kinds, and individual volume
populators will create a CR for each kind of data source that it understands. A new
`volume-data-source-validator` controller will generate events on PVCs with data sources
that don't match any registered volume populator.

## Motivation

### Goals

- Don't break any existing behavior, even obviously unintentional behavior that currently
  works.
- Enable users to create pre-populated volumes in a manner consistent with current practice
- Enable developers to innovate with new an interesting data sources
- Minimize change to existing workflows and breaking existing tools with new behavior
- Ensure that users continue to get accurate error messages and feedback about volume
  creation progress, whether using existing methods of volume creation or specifying a
  data source associated with a volume populator.
- Maintain existing pattern of allowing things to happen out of order. As long as there's
  a chance things will succeed in the future, just provide a status update and retry
  again later.

### Non-Goals

- This proposal recommends an implementation for volume populator but it DOES NOT
  require any specific approach for the actual volume populators. Alternative
  approaches that result in PVCs getting bound to volumes containing correct data
  are fine as long as they integrate with the data source validator by providing
  an instance of `VolumePopulator`.

## Proposal

Introduce a new field on PVCs called `DataSourceRef`. This new field operates like
`DataSource` except that it never ignores input -- contents are always accepted or the
entire PVC is rejected if the contents are deemed invalid.

Deprecate the `DataSource` field, but continue to support it in a backwards-compatible
way. In particular:

1. If `DataSource` and `DataSourceRef` are both unset; accept
1. If `DataSource` is set valid (PVC or VolumeSnapshot) AND `DataSourceRef` is not set;
   set `DataSourceRef` from `DataSource`
1. If `DataSource` is set valid AND `DataSourceRef` is set the same; accept
1. If `DataSource` is set valid AND `DataSourceRef` is set differently; reject
1. If `DataSource` is set invalid (anything but PVC or VolumeSnapshot) and `DataSourceRef`
   is not set; wipe `DataSource` (as today)
1. If `DataSource` is set invalid and `DataSourceRef` is set; reject
1. If `DataSource` is not set and `DataSourceRef` is set invalid (any core object other
   than PVC); reject
1. If `DataSource` is not set and `DataSourceRef` is set PVC or VolumeSnapshot; set
   `DataSource` from `DataSourceRef` and accept
1. If `DataSource` is not set and `DataSourceRef` is set some other valid value (not
   core object and not VolumeSnapshot); leave `DataSource` unset and accept
   
The last case is the one that enables using new types of data sources. Existing data
sources continue to work, either through the old interface or the new interface. New
data sources only work through the new interface, and additional validation is
performed after the PVC is created.

A new controller will be introduced called `data-source-validator` which will watch PVCs
objects and generate events on objects with data sources that are not considered valid. To
determine which data sources are valid a new CRD will be introduced valled `VolumePopulator`
which will allow actual data populator controllers to register the API `groups`/`kinds` that
they understand with the `data-source-validator` controller. If a PVC has a data source that
matches the `group`/`kind` of a registered `VolumePopulator` the controller will consider
it valid and emit no events, relying on the actual populator to give feedback to the user.

Instances of the VolumePopulator CRD will look like:

```
kind: VolumePopulator
apiVersion: populator.storage.k8s.io/v1alpha1
metadata:
  name: example-populator
sourceKind:
  group: example.storage.k8s.io
  kind: Example
```

In this model there will be 4 sources of feedback to the end user:
1. The API server will reject core objects other than PVCs. This is backwards compatible, and
   appropriate because we don't expect any existing core object (other than a PVC) to be a
   valid data source.
1. For data sources that refer to a unknown group/kind, the `volume-data-source-validator` will emit
   an event on the PVC informing the user that the PVC is currently not being acted on, so
   the user can fix any errors, and so the user doesn't wait forever wondering why the PVC
   is not binding. Alternatively, the event might remind the user that they forgot to
   install the actual populator, in which case they can do so, and the PVC will eventually
   bind if all of the installation steps succeed.
1. For every PVC under the responsibility of a CSI driver, the provisioner sidecar for that
   PVC will emit an event explaining that the sidecar is not taking action on the PVC,
   assuming the data source is something that a populator is responsible for and not a
   PVC or VolumeSnapshot, which the sidecar is responsible for.
1. For data source that refer to a known group/kind the populator controller itself should
   provide feedback on the PVC, including error messages if the data source is not found,
   is malformed, is inaccessible, or whatever. The populator controller may also provide
   status updates on the PVC if the population operation is long-running.

Populators themselves will work by responding to PVC objects with a data source they
understand, and producing a PV with the expected data, such that ordinary Kubernetes
workflows are not disrupted. In particular the PVC should be attachable to the end user's
pod the moment it is bound, similar to a PVC created from currently supported data sources.

### User Stories

There are a long list of possible use cases around volume populators, and I won't
try to detail them all here. I will detail a few that illustrate the challenges faced by
users and developers, but it's important to see these as a few examples among many.

#### VM Images

One use case was relevant to [KubeVirt](https://kubevirt.io/), where the Kubernetes
`Pod` object is actually a
VM running in a hypervisor, and the `PVC` is actually a virtual disk attached to a VM.
It's common for virtualization systems to allows VMs to boot from disks, and for disks
to be pre-populated with various OS images. OS images tend to be stored in external
repositories dedicated to that purpose, often with various mechanisms for retrieving
them efficiently that are external to Kubernetes.

One way to achieve this is to represent disk images as custom resources that point to
the image repository, and to allow creation of PVCs from these custom resources such
that the volumes come pre-populated with the correct data. Efficient population of the
data could be left up to a purpose-built controller that knows how to get the bits
where they need to be with minimal I/O.

#### Backup/Restore

Without getting into the details of how backup/restore should be implemented, it's
clear that whatever design one chooses, a necessary step is to have the user
(or higher level controller) create a PVC that points to the backup they want to
restore, and have the data appear in that volume somehow.

One can imagine backups simply being a special case of snapshots, in which case the
existing design is sufficient, but if you want anything more sophisticated, there
will inevitably be a new object type that represents a backup. While it's arguable
that backup should be something CSI plugins should be exclusively responsible for,
one can also argue that generic backup tools should also exist which can backup
and restore all kind of volumes. Those tools will be apart from CSI plugins and
yet need a way to populate volumes with restored volumes.

It's also likely that multiple backup/restore implementations will be developed,
and it's not a good idea to pick a winner at the Kubernetes API layer. It makes
more sense to enable developers to try different approaches by making the API allow
restoring from various kinds of things.

### Implementation Details/Notes/Constraints

As noted above, the proposal recommends a modest change to the PVC API which
creates a new more flexible field to specify data sources without breaking any
existing behaviors. Mostly the new `DataSourceRef` will never ignore user input,
so that arbitrary CRDs can be used as data sources.
This raises the question of WHAT will happen when users
put new things in that field, and HOW populators will actually work with so small
a change.

It's first important to note that only consumers of the `DataSource` field today
are the various dynamic provisioners, most notably the external-provisioner CSI
sidecar. Due to the API change, these consumers will need to switch to consume the
`DataSourceRef` field, but will otherwise continue to function mostly the same.
Currently, if the external-provisioner sidecar sees a data source it doesn't
understand, it simply ignores the request, which is both important for forward
compatibility, and also perfect for the purposes of a data populator. This allows
developers to add new types of data sources that the dynamic provisioners will
simply ignore, enabling a different controller to see these objects and respond
to them.

The recommended approach for implementing a volume populator is make a controller
that responds to new PVCs with data sources that it understands by creating
a second PVC (PVC') in a different namespace that has all of the same specs as
the original, minus the data source, and to attach a pod to that newly created
empty volume and to fill it with data. After PVC' has the correct data, the
associated pod can be removed, and the PV that PVC' was bound to can be rebound
to the original PVC.

A library is being developed to handle the common parts of the volume population
workflow, so that individual implementations of volume populators need only
include code to write data into the volume, and everything else can be handled
by this shared library. Ultimately the library may evolve into a sidecar
container for ease of bug fixing, but the important thing is that all of the
common controller logic should be reusable.

### Risks and Mitigations

The replacement of `DataSource` with `DataSourceRef` greatly reduces risk
relative to earlier designs. It leaves existing behavior the same and
requires all clients to understand and use the new fields to enable the
new functionality. The main risk is that if we eventually change our minds
about this functionality, we've introduced a new field that we might
need to carry into the future.

We are not concerned about backwards compatibility problems, as any existing
request should continue to behave the same, and existing controllers will
continue to work the same. Nevertheless we need to be careful about
implementing the admission control logic that ensures backwards compatibility
and we need good tests to cover all of the possible cases.

Security issues are hard to measure, because any security issues would be the
result of badly designed data populators that failed to put appropriate
limits on user's actions. Such security issues are going to be present with
any new controller, though, so they don't seem relevant to this change. The
main thing to realize is that the `DataSource` field is a "local" typed
object reference, so no matter what, the object in that field has to either
be in the same namespace as the PVC that references it, or it must be a
non-namespaced object. This seems like an appropriate and desirable
limitation for security reasons.

If we think about who can install populators, the RBAC required for a
populator to operate requires at minimum, the ability to either create or
modify PVs. Also the CRD for the data source type needs to be installed.
This means that populators will generally be installed by cluster admins
or similarly-powerful users, and those users can be expected to understand
the uses and implications of any populators they chose to install. 

## Design Details

### Test Plan

To test the API changes to PVC, we need test cases for each of the data source
situations outlined above, focusing especially on backwards compatibility.
- Create PVC with `DataSource` and `DataSourceRef` both unset; except success and both unset
- Create PVC with `DataSource` is set to PVC and `DataSourceRef` not set; expect success and `DataSourceRef` contains PVC
- Create PVC with `DataSource` is set to VolumeSnapshot and `DataSourceRef` not set; expect success and `DataSourceRef` contains VolumeSnapshot
- Create PVC with `DataSource` is set valid AND `DataSourceRef` is set the same; expect success
- Create PVC with `DataSource` is set valid AND `DataSourceRef` is set differently; expect error
- Create PVC with `DataSource` is set to Pod and `DataSourceRef` is not set; expect success and both unset
- Create PVC with `DataSource` is set to CRD and `DataSourceRef` is not set; expect success and both unset
- Create PVC with `DataSource` is set invalid and `DataSourceRef` valid; expect error
- Create PVC with `DataSource` is not set and `DataSourceRef` is set to Pod, expect error
- Create PVC with `DataSource` is not set and `DataSourceRef` is set to PVC; expect success and `DataSource` contains PVC
- Create PVC with `DataSource` is not set and `DataSourceRef` is set to VolumeSnapshot; expect success and `DataSource` contains VolumeSnapshot
- Create PVC with `DataSource` is not set and `DataSourceRef` is set to CRD; expect success and `DataSource` empty

To test the `volume-data-source-validator`, we need to check the following cases:
- Creation of a PVC with no `DataSourceRef` causes no events.
- Creation of a PVC with a VolumeSnapshot or PVC `DataSourceRef` causes no events.
- Creation of a PVC with a CRD `DataSourceRef` that's not registered by any
  volume populator causes UnrecognizedDataSourceKind events.
- Creation of a PVC with a CRD `DataSourceRef` that's registered by a
  volume populator causes no events.

### Graduation Criteria

#### Alpha -> Beta Graduation

- Before going to beta, we need a clear notion of how data populators should
  work.
- We will need a simple and lightweight implementation of a data populator
  that can be used by the E2E test suite to exercise the functionality.
- Automated tests that create a volume from a data source handled by a
  populator, to validate that the data source functionality works, and that
  any other required functionality for data population is working.
- We will need to see several implementations of working data populators that
  solve real world problems implemented in the community.
- `volume-data-source-validator` and associated VolumePopulator CRD should be released
  with beta versions available.

#### Beta -> GA Graduation

- Distributions including data populators as part of their distros (possibly
  a backup/restore implementation layered on top)
- Allowing time for feedback

### Upgrade / Downgrade Strategy

On upgrade, a new field is added, the contents of which can be reliably
determined for existing objects. New objects could put new contents into
that field, but the admission controllers will keep the old field in sync
with it for backwards compatibility.

On downgrade, simply dropping the new field would be safe because any
clients accessing the new alpha field would have caused the
backwards-compatible values to be set in the existing field.

### Version Skew Strategy

No issues

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: AnyVolumeDataSource
    - Components depending on the feature gate: kube-apiserver
  - [X] Other
    - Describe the mechanism: Also install the data-source-validator controller
      and its associated VolumePopulator CRD.
    - Will enabling / disabling the feature require downtime of the control
      plane? no
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? no

* **Does enabling the feature change any default behavior?**
  A new field is introduced on PVCs, but existing clients can safely ignore it.
  New clients that use the new field can cause PVCs to be created which can be
  used by volume populators.
  
* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, dropping or ignoring the new field returns the system to it's previous
  state. Worst case, some PVCs which were trying to use the new field might need
  to be deleted because they will never have anything happen to them after the
  feature is disaabled.
  
* **What happens if we reenable the feature if it was previously rolled back?**
  As long as the alpha field was not dropped, things will go back to how they
  were. If the alpha field is dropped, and then re-added, the same PVCs which
  were defunct after the disablement will need to be deleted, but nothing else
  bad will happen.

* **Are there any tests for feature enablement/disablement?**
  Not at this time.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
  None.

* **What specific metrics should inform a rollback?**
  None.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Yes, we are testing the upgrade and rollback paths on the prototype implementation
  as we go, and will manually test the finished version.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  No.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Simply look at the data source field of the PVCs. Non-empty data sources with
  a Kind other than snapshot and PVC indicate this feature is in use. Also the
  existence of any VolumePopulator.populator.storage.k8s.io CRs would indicate
  that a populator is installed and could be used.

  As the controller itself is out of tree, the administrator or distro must
  install it (similar to the volume snapshot controller) and we will use the
  same mechanisms as the volume snapshot controller for installation and
  health monitoring.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).
  * Counter for number of PVCs with no data sources
  * Counter for number of PVCs with valid data sources
  * Counter for number of PVCs with invalid data sources

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  This feature depends on the VolumePopulator CRD being installed, and the
  associated data-source-validator controller.

### Scalability

* **Will enabling / using this feature result in any new API calls?**
  No

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type: VolumePopulator.populator.storage.k8s.io
  - Supported number of objects per cluster: Small number
  - Supported number of objects per namespace: Not namespaced

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
  No
  
* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  No

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  No. PVC creation already takes an unbounded amount of time. It is valid to
  create a PVC with a data source that does not yet exist, and to create that
  data source later. This feature maintains that status quo ante, and in all
  cases, ensures that some controller is responsible for generating feedback
  on the progress of the PVC creation so users can understand why operations
  are not yet finished.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  The new data-source-validator controller will consist of a deployment with 1 or
  more replicas. It will watch PVCs and VolumePopulators and emit events on
  PVCs.

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**
  n/a
  
* **What are other known failure modes?**
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

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- The idea of data populators has been discussed abstractly in SIG-storage
  since 2018 at least.
- John Griffith did the original work to propose something like this, but
  that work got scaled down to just PVC clones.
- Ben Swartzlander picked up the populator proposal developed 2 prototypes
  in December 2019.
- New KEP proposed January 2020
- Feature gate merged as Alpha in v1.18
- No progress in v1.19
- Validation webhook proposed September 2020
- KEP updated to new format September 2020
- Webhook replaced with controller in December 2020
- KEP updated Feb 2021 for v1.21
- Resign with new `DataSourceRef` field in May 2021 for v1.22, still alpha

## Alternatives

### Validating webhook

One alternative that was considered was to use a validating webhook instead of
an async controller. This was prototyped first, and solved the problem of
preventing PVC with invalid data sources, but it solved that problem too well,
because it rejected PVCs which would be valid at some point in the future, if
they were created before the necessary VolumePopulator object was created.

The user experience of PVCs being rejected was too different from the existing
behavior, where, if a user creates a PVC referring to a volume snapshot data
source, the PVC is accepted even if none of the required controllers are
installed yet. Later, after the required controllers are installed, that PVC
can be bound as the user expected.

We wanted to preserve the general property of Kubernetes where it's okay to
specify your intent, and then satisfy the dependencies thereof out of order,
and still obtain the desired result. Programatic workflows that are dumb and
don't understand order of operations benefit from being able to create a
bunch of objects and wait for the desired result.
