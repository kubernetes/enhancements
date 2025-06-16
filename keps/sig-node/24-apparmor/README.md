# Add AppArmor Support

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Background](#background)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API](#api)
    - [Pod Annotations (beta API)](#pod-annotations-beta-api)
    - [Pod API](#pod-api)
      - [RuntimeDefault Profile](#runtimedefault-profile)
      - [Localhost Profile](#localhost-profile)
    - [Validation](#validation)
    - [Node Status](#node-status)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Failure and Fallback Strategy](#failure-and-fallback-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
    - [Pod Creation](#pod-creation)
    - [Pod Security Admission](#pod-security-admission)
    - [Pod Update](#pod-update)
    - [PodTemplates](#podtemplates)
    - [Warnings](#warnings)
    - [Kubelet fallback](#kubelet-fallback)
    - [Runtime Profiles](#runtime-profiles)
    - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
      - [Kubelet Backwards compatibility](#kubelet-backwards-compatibility)
    - [Removing annotation support](#removing-annotation-support)
  - [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Syncing fields &amp; annotations on workload resources](#syncing-fields--annotations-on-workload-resources)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required _prior to targeting to a milestone / release_.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in
      [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and
      SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [ ] Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for
      publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to
      mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This is a proposal to add AppArmor support to the Kubernetes API.

For GA graduation, this proposal aims to do the _bare minimum_ to clean up the feature from its beta
release, without blocking future enhancements.

## Motivation

AppArmor can enable users to run a more secure deployment, and/or provide better auditing and
monitoring of their systems. AppArmor should be supported to provide users an alternative to
SELinux, and provide an interface for users that are already maintaining a set of AppArmor profiles.

### Background

Kubernetes AppArmor support predates most of our current feature lifecycle practices, including the
KEP process. This KEP is backfilling for current AppArmor support. For the original AppArmor
proposal, see https://github.com/kubernetes/design-proposals-archive/blob/main/auth/apparmor.md.

This KEP is proposing a minimal path to GA, per the
[no perma-Beta requirement](/keps/sig-architecture/1635-prevent-permabeta/README.md).
This feature graduation closely parallels that of [Seccomp](/keps/sig-node/135-seccomp/README.md).
The notable exceptions are that the AppArmor annotations are immutable on pods, which simplifies the
migration. AppArmor is also feature gated, via the `AppArmor` gate.

### Goals

- Allow running Pods with AppArmor confinement

### Non-Goals

This KEP proposes the absolute minimum to provide generally available AppArmor
confinement for Pods and their containers. Further functional enhancements are out of scope,
including:

- Defining any standard "Kubernetes branded" AppArmor profiles
- Formally specifying the AppArmor profile format in Kubernetes
- Providing mechanisms for defining custom profiles using the Kubernetes API, or for
  loading profiles from outside of the node.
- Windows support

## Proposal

Add a new field to the Pod API that allows defining the AppArmor profile. The new field should be
part of the security context.

### API

Pods and PodTemplate will include an `appArmorProfile` field that you can set either for a Pod's
security context or for an individual container. If AppArmor options are defined at both the pod and
container level, the container-level options override the pod options.

#### Pod Annotations (beta API)

The beta API was defined through annotations on pods.

The `container.apparmor.security.beta.kubernetes.io/<container_name>` annotation will be used to
configure the AppArmor profile that the container named `<container_name>` is run with. The
annotation is immutable on Pods.

Possible annotation values are:

1. `runtime/default` - This explicitly selects the default profile configured by the container
   runtime. Absent this annotation, containerd and CRI-O will run non-privileged containers with
   this profile by default on AppArmor-enabled (LSM loaded) hosts.
2. `unconfined` - Run without any AppArmor profile. This is the default for privileged pods.
3. `localhost/<profile_name>` - Run the container using the `<profile_name>` AppArmor profile. The
   profile must be pre-loaded into the kernel (typically via `apparmor_parser` utility), otherwise
   the container will not be started.

#### Pod API

The Pod AppArmor API is generally immutable, except in `PodTemplates`.

```go
type PodSecurityContext struct {
    ...
    // The AppArmor options to use by the containers in this pod.
    // Note that this field cannot be set when spec.os.name is windows.
    // +optional
    AppArmorProfile  *AppArmorProfile
    ...
}

type SecurityContext struct {
    ...
    // The AppArmor options to use by this container. If AppArmor options are
    // provided at both the pod & container level, the container options
    // override the pod options.
    // Note that this field cannot be set when spec.os.name is windows.
    // +optional
    AppArmorProfile  *AppArmorProfile
    ...
}

// AppArmorProfile defines a pod or container's AppArmor settings.
// Only one profile source may be set.
// +union
type AppArmorProfile struct {
    // type indicates which kind of AppArmor profile will be applied.
    // Valid options are:
    //   Localhost - a profile pre-loaded on the node.
    //   RuntimeDefault - the container runtime's default profile.
    //   Unconfined - no AppArmor enforcement.
    // +unionDescriminator
    Type AppArmorProfileType

    // LocalhostProfile indicates a loaded profile on the node that should be used.
    // The profile must be preconfigured on the node to work.
    // Must match the loaded name of the profile.
    // Must only be set if type is "Localhost".
    // +optional
    LocalhostProfile *string
}

type AppArmorProfileType string

const (
    AppArmorProfileTypeUnconfined     AppArmorProfileType = "Unconfined"
    AppArmorProfileTypeRuntimeDefault AppArmorProfileType = "RuntimeDefault"
    AppArmorProfileTypeLocalhost      AppArmorProfileType = "Localhost"
)
```

This API makes the options more explicit and leaves room for new profile sources
to be added in the future (e.g. Kubernetes predefined profiles or ConfigMap
profiles) and for future extensions, such as defining the behavior when a
profile cannot be set.

##### RuntimeDefault Profile

We propose maintaining the support to a single runtime profile, which will be
defined by using the `AppArmorProfileTypeRuntimeDefault`. The reasons being:

- No changes to the current behavior. Users are currently not allowed to specify
  other runtime profiles. The existing API server rejects runtime profile names
  that are different than `runtime/default`.
- Most runtimes only support the default profile, although the CRI is flexible
  enough to allow the kubelet to send other profile names.
- Multiple runtime profiles has never been requested as a feature.

If built-in support for multiple runtime profiles is needed in the future, a new
KEP will be created to cover its details.

##### Localhost Profile

This KEP proposes LocalhostProfile as the only source of user-defined
profiles. User-defined profiles are essential for users to realize
the full benefits out of AppArmor, allowing them to decrease their attack
surface based on their own workloads.

###### Updating localhost AppArmor profiles

AppArmor profiles are applied at container creation time. The underlying
container runtime only references already loaded profiles by its name.
Therefore, updating the profiles content requires a manual reload (typically via
`apparmor_parser`).

Note that changing profiles is not recommended and may cause containers to fail
on next restart, in the case of the new profile being more restrictive, invalid
or the file no longer available on the host.

Currently, users have no way to tell whether their physical profiles have been
deleted or modified. This KEP proposes no changes to the existing functionality.

The recommended approach for rolling out changes to AppArmor profiles is to
always create _new profiles_ instead of updating existing ones. Create and
deploy a new version of the existing Pod Template, changing the profile name to
the newly created profile. Redeploy, once working delete the former Pod
Template. This will avoid disruption on in-flight workloads.

The current behavior lacks features to facilitate the maintenance of AppArmor
profiles across the cluster. Two examples being: 1) the lack of profile
synchronization across nodes and 2) how difficult it can be to identify that
profiles have been changed on disk/memory, after pods started using it. However,
Kubernetes managed profiles are out of scope for this KEP.
Out of tree enhancements like the
[security-profiles-operator](https://github.com/kubernetes-sigs/security-profiles-operator) can
provide such enhanced functionality on top.

###### Profiles managed by the cluster admins

The current support relies on profiles being loaded on all cluster nodes
where the pods using them may be scheduled. It is also the cluster admin's
responsibility to ensure the profiles are correctly saved and synchronized
across the all nodes. Existing mechanisms like node `labels` and `nodeSelectors`
can be used to ensure that pods are scheduled on nodes supporting their desired
profiles.

#### Validation

The following validations were applied to the AppArmor annotations on pods:
- Pod annotations are immutable (cannot be added, modified, or removed on pod update)
- Annotation value must have a `localhost/` prefix, or be one of: `""`, `runtime/default`, `unconfined`.

The annotation validations will be carried over to the field API, and the following additional
validations are proposed:
1. Fields must match the corresponding annotations when both are present, except for ephemeral containers.
2. AppArmor profile must be unset on Windows pods (`spec.os.name == "windows"`). Only enforced on fields.
3. Localhost profile must not be empty, and must not be padded with whitespace. Only enforced on creation.
   This was previously enforced by the [Kubelet](https://github.com/kubernetes/kubernetes/blob/2624e93d55375a9642977d4d5795841ab7463b1d/pkg/security/apparmor/validate.go#L70-L77).

*Note on localhost profile validation:* AppArmor profile naming is flexible, but both of the leading
CRI implementations (containerd & cri-o) require a profile with a matching name to be loaded. This
prevents the special `unconfined` profile, or various wildcard and variable profile names from being
used in practice. This validation is deferred to the runtime, rather than being enforced by the API
for backwards compatibility.

#### Node Status

The Kubelet SHOULD NOT append the AppArmor status to the node ready condition message.

The ready condition is certainly not the right place for this message, but more generally the
kubelet does not broadcast the status of every optional feature. (A beta implementation of
this feature, added before the Kubernetes enhancement process was formalized, did customize
the node ready condition message).

## Design Details

When an AppArmor profile is set on a container (or pod), the kubelet will pass the option on to the
container runtime, which is responsible for running the container with the desired profile. Profiles
must be loaded into the kernel before the container is started (profile loading is out of scope for
this KEP). For more details, see https://kubernetes.io/docs/tutorials/security/apparmor/.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None

##### Unit tests

- [`TestDropAppArmor`](https://github.com/kubernetes/kubernetes/blob/0ff6a00fafee08467d946ab18c7839d9704d27d5/pkg/api/pod/util_test.go#L706)
- [Pod validation tests](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/validation/validation_test.go)
- [`TestReadyCondition`](https://github.com/kubernetes/kubernetes/blob/0ff6a00fafee08467d946ab18c7839d9704d27d5/pkg/kubelet/nodestatus/setters_test.go#L1480)
- [Host validation tests](https://github.com/kubernetes/kubernetes/blob/master/pkg/security/apparmor/validate_test.go)
- [Pod Security Admission policy](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/pod-security-admission/policy/check_appArmorProfile_test.go)

New tests will be added covering the annotation/field conflict cases described
under [Version Skew Strategy](#version-skew-strategy).

##### Integration tests

- Pod Security tests: https://github.com/kubernetes/kubernetes/blob/1ded677b2a77a764a0a0adfa58180c3705242c49/test/integration/auth/podsecurity_test.go

##### e2e tests

[AppArmor node E2E][https://github.com/kubernetes/kubernetes/blob/2f6c4f5eab85d3f15cd80d21f4a0c353a8ceb10b/test/e2e_node/apparmor_test.go]
  - These tests are guarded by the `[Feature:AppArmor]` tag and run as part of the
    [containerd E2E features](https://testgrid.k8s.io/sig-node-containerd#node-e2e-features)
    test suite.

The E2E tests will be migrated to the field-based API.

### Failure and Fallback Strategy

There are different scenarios in which applying an AppArmor profile may fail,
below are the ones we mapped and their outcome once this KEP is implemented:

| Scenario                                                                                           | API Server Result    | Kubelet Result                                                                                                                                        |
| -------------------------------------------------------------------------------------------------- | -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1) Using localhost or explicit `runtime/default` profile when container runtime does not support AppArmor.  | Pod created          | The outcome is container runtime dependent. In this scenario containers may 1) fail to start or 2) run normally without having its policies enforced. |
| 2) Using custom or `runtime/default` profile that restricts actions a container is trying to make. | Pod created          | The outcome is workload and AppArmor dependent. In this scenario containers may 1) fail to start, 2) misbehave or 3) log violations.                  |
| 3) Using a localhost profile that does not exist on the node.                                      | Pod created          | Container runtime dependent: containers fail to start. Retry respecting RestartPolicy and back-off delay.  Error message in event.                |
| 4) Using an unsupported runtime profile (i.e. `runtime/default-audit`).                            | Fails validation: pod **not** created. | N/A                                                                                                                                                   |
| 5) Using localhost or explicit `runtime/default` profile when AppArmor is disabled by the host or build | Pod created. | Kubelet puts Pod in blocked state.                                                                                                                    |
| 6) Using implicit (default) `runtime/default` profile when AppArmor is disabled by the host or build. | Pod created | Container created without AppArmor enforcement. |
| 7) Using localhost profile with invalid (empty) name                                               | Fails validation: pod **not** created. | N/A                                                                                                                                                   |

Scenario 2 is the expected behavior of using AppArmor and it is included here
for completeness.

Scenario 7 represents the case of failing the existing validation, which is
defined at [Pod API](#pod-api).

### Version Skew Strategy

All API skew is resolved in the API server.

#### Pod Creation

If no AppArmor annotations or fields are specified, no action is necessary.

If the `AppArmor` feature is disabled per feature gate, then the annotations and
fields are cleared ([current behavior](https://github.com/kubernetes/kubernetes/blob/f58f70bd5730658505042cd9baa80f72d3b6e31e/pkg/api/pod/util.go#L526-L532)).

If the pod's OS is `windows`, fields are forbidden to be set and annotations
are not copied to the corresponding fields.

If _only_ AppArmor fields are specified, add the corresponding annotations. If these
are specified at the Pod level, copy the annotations to each container that does
not have annotations already specified. This ensures that the fields are enforced
even if the node version trails the API version (see [Version Skew Strategy](##version-skew-strategy)).

If _only_ AppArmor annotations are specified, copy the values into the
corresponding fields. This ensures that existing applications continue to
enforce AppArmor, and prevents the kubelet from needing to resolve annotations &
fields. If the annotation is empty, then the `runtime/default` profile will be
used by the CRI container runtime. If a localhost profile is specified, then
container runtimes will strip the `localhost/` prefix, too. This will be covered
by e2e tests during the GA promotion.

If both AppArmor annotations _and_ fields are specified, the values MUST match.
This will be enforced in API validation.

Container-level AppArmor profiles override anything set at the pod-level.

#### Pod Security Admission

The Pod Security admission plugin will be updated to evaluate AppArmorProfile fields in addition to
annotations.

The [policy for the **baseline** Pod security standard](https://kubernetes.io/docs/concepts/security/pod-security-standards/#baseline)
forbids setting an `Unconfined` profile, but allows unset, `RuntimeDefault` and `Localhost`
profiles. In the case of localhost profiles, this can include OS profiles intended for other system
daemons, so additional profile restrictions are encouraged (e.g. via
[ValidatingAdmissionPolicy](https://kubernetes.io/docs/reference/access-authn-authz/validating-admission-policy/)).


#### Pod Update

The AppArmor fields on a pod are immutable, which also applies to the
[annotation](https://github.com/kubernetes/kubernetes/blob/b46612a74224b0871a97dae819f5fb3a1763d0b9/pkg/apis/core/validation/validation.go#L177-L182).

When an [Ephemeral Container](/keps/sig-node/277-ephemeral-containers/README.md) is added, it
will follow the same rules for using or overriding the pod's AppArmor profile.
Ephemeral container's will never sync with an AppArmor annotation.

#### PodTemplates

PodTemplates (and their embeddings within e.g. ReplicaSets, Deployments, StatefulSets, etc.) will be
ignored. The field/annotation resolution will happen on template instantiation.

#### Warnings

To raise awareness of workloads using the beta AppArmor annotations that need to be migrated, a
warning will be emitted when only AppArmor annotations are set (no fields) on pod creation, or pod
template (including workload resources with an embedded pod template) create & update.

#### Kubelet fallback

Since Kubelet versions must not be ahead of API versions, Kubelets can defer annotation/field
resolution to the API server, and only consider the AppArmor fields.

The exception to this is static pods. In this case, Kubelet will copy annotation values to fields in the
[`applyDefaults`](https://github.com/kubernetes/kubernetes/blob/2363cdcc399cbf428210efb2c51575ddcad2b84a/pkg/kubelet/config/common.go#L57C6-L57C19)
function. In this case, Kubelet will also log a warning.

#### Runtime Profiles

The API Server will continue to reject annotations with runtime profiles
different than `runtime/default`, to maintain the existing behavior.

Violations would lead to the error message:

```
Invalid value: "runtime/profile-name": must be a valid AppArmor profile
```

#### Upgrade / Downgrade Strategy

Nodes do not currently support in-place upgrades, so pods will be recreated on
node upgrade and downgrade. No special handling or consideration is needed to
support this.

On the API server side, we've already taken version skew in HA clusters into
account. The same precautions make upgrade & downgrade handling a non-issue.

Since
[we support](https://kubernetes.io/docs/setup/release/version-skew-policy/) up
to 2 minor releases of version skew between the master and node, annotations
must continue to be supported and backfilled for at least 2 versions passed the
initial implementation. Specifically, fields will no longer be copied to annotations for older kubelet
versions. However, annotations submitted to the API server will continue to be copied to fields at the
kubelet indefinitely, as was done with Seccomp.

##### Kubelet Backwards compatibility

Since we don't support running newer Kubelets than API server, new Kubelets only need to handle
AppArmor fields. All the version skew resolution happens within the API server.

#### Removing annotation support

_(Assuming field support merges in 1.30, otherwise adjust all versions a constant amount)_

Phase 1 (v1.30): AppArmor field support merged
- Sync annotations & fields on Pod create (version skew strategy described above)
- Warn on annotation use, if field isn't set
- Kubelet copies static pod annotations to fields

Phase 2 (v1.34):
- API server stops copying fields to annotations
- Warn on annotation use if there is no corresponding _container_ field (including on workload resources)
- **Risk:** policy controllers that don't consider field values

Phase 3 (v1.36): End state
- API server stops copying annotations to fields
- Kubelet stops copying annotations to fields for static pods
- Validation that annotations & fields match persists indefinitely
- **Risk:** workloads that haven't migrated

### Graduation Criteria

**General Availability:**
- Field-based API

## Production Readiness Review Questionnaire

### Feature enablement and rollback

###### How can this feature be enabled / disabled in a live cluster?

AppArmor is controlled by the `AppArmor` feature gate (already beta by the time this KEP was
formally opened).

- [X] Feature gate
  - Feature gate name: `AppArmor`
  - Components depending on the feature gate:
      - kube-apiserver
      - kubelet

###### Does enabling the feature change any default behavior?

No - AppArmor has been enabled by default since Kubernetes v1.4.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Containers already running with AppArmor enforcement will continue to do so, but on restart
will fallback to the container runtime default. Pods created with AppArmor disabled will have their
fields & annotations stripped.

###### What happens if we reenable the feature if it was previously rolled back?

Newly started or restarted containers in pods that still have the AppArmor field/annotations will
have the specified AppArmor profile applied, rather than the runtime default.

###### Are there any tests for feature enablement/disablement?

No.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The [Version Skew Strategy](#version-skew-strategy) section covers this point.
Running workloads should have no impact as the Kubelet will support either the
existing annotations or the new fields introduced by this KEP.

Disabling the AppArmor feature will cause the container runtimes to apply the runtime default
profile (except for privileged pods). In cases where a user was expecting to apply a custom profile
(or explicitly unconfined), this could break the workload.

###### What specific metrics should inform a rollback?

An increase in pod validation errors can indicate issues with the field translation. These would
show up as `code=400` (Bad Request) errors in `apiserver_request_total`.

The following errors could indicate problems with how kubelets are interpreting AppArmor profiles.
* `started_containers_errors_total`
* `started_pods_errors_total`

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Automated tests will cover the scenarios with and without the changes proposed
on this KEP. As defined under [Version Skew Strategy](#version-skew-strategy),
we are assuming the cluster may have kubelets with older versions (without
this KEP' changes), therefore this will be covered as part of the new tests.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

The promotion of AppArmor to GA would deprecate the beta annotations as described in the
[Version Skew Strategy](#version-skew-strategy).

### Monitoring requirements

###### How can an operator determine if the feature is in use by workloads?

The feature is built into the kubelet and api server components. No metric is
planned at this moment. The way to determine usage is by checking whether the
pods/containers have a AppArmorProfile set.

###### How can someone using this feature know that it is working for their instance?

The AppArmor enforcement status is not directly surfaced by Kubernetes, but is visible through the
linux proc API. For example, you can check what profile a container is running with by execing into it:

```sh
$ kubectl exec -n $NAMESPACE $POD_NAME -- cat /proc/1/attr/current
k8s-apparmor-example-deny-write (enforce)
```

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Negligible increase in Pod object size, and any objects embedding a PodSpec.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. AppArmor profiles are managed outside of Kubernetes, and without this feature enabled the
runtime default AppArmor profile is still enforced on non-privileged containers (for AppArmor
enabled hosts).

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No impact to running workloads.

###### What are other known failure modes?

No impact is being foreseen to running workloads based on the nature of
changes brought by this KEP.

Although some general errors and failures can be seen on [Failure and Fallback
Strategy](#failure-and-fallback-strategy).

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- 2016-07-25: [AppArmor design proposal](https://github.com/kubernetes/design-proposals-archive/blob/main/auth/apparmor.md)
- 2016-09-26: AppArmor beta release with v1.4
- 2020-01-10: Initial (retrospective) KEP

## Drawbacks

- Custom AppArmor profiles are not fully managed by Kubernetes
- AppArmor support adds a dimension to the feature compatibility matrix, as support is not
  guaranteed in linux

## Alternatives

### Syncing fields & annotations on workload resources

AppArmor fields & annotations on Pods are immutable, which means that syncing fields & annotations
is a one-time operation. This is not true for workload resources (ReplicaSets, Deployments, etc).

In order to support syncing fields on workload resources, we need to account for clients that only
pay attention to one of the field/annotation settings. When combined with the validation requirement
that fields & annotations match, getting this right in both the patch & update cases adds
significant complexity.
