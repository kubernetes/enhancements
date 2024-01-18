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
    - [Pod Annotations](#pod-annotations)
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
    - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
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

This is a (retroactive) KEP to add AppArmor support to the Kubernetes API.

## Motivation

AppArmor can enable users to run a more secure deployment, and/or provide better auditing and
monitoring of their systems. AppArmor should be supported to provide users an alternative to
SELinux, and provide an interface for users that are already maintaining a set of AppArmor profiles.

### Background

Kubernetes AppArmor support predates most of our current feature lifecycle practices, including the
KEP process. This KEP is backfilling for current AppArmor support. For the original AppArmor
proposal, see https://github.com/kubernetes/design-proposals-archive/blob/main/auth/apparmor.md.

### Goals

- Allow running Pods with AppArmor confinement

### Non-Goals

- Defining any standard "Kubernetes branded" AppArmor profiles
- Formally specifying the AppArmor profile format in Kubernetes
- Providing mechanisms for defining custom profiles using the Kubernetes API, or for
  loading profiles from outside of the node.
- Windows support

## Proposal

Add an immutable pod annotation to configure the AppArmor profile on a per-container basis.

### API

The beta API is defined through annotations on pods.

#### Pod Annotations

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

The following validations are applied to the AppArmor annotations on pods:
- Pod annotations are immutable (cannot be added, modified, or removed on pod update)
- Annotation value must have a `localhost/` prefix, or be one of: `""`, `runtime/default`, `unconfined`.

#### Node Status

The Kubelet appends "AppArmor enabled" to the message on the node ready condition to indicate if a
node is AppArmor enabled.

## Design Details

When an AppArmor profile is set on a container, the kubelet will pass the option on to the
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

##### Integration tests

- Pod Security tests: https://github.com/kubernetes/kubernetes/blob/1ded677b2a77a764a0a0adfa58180c3705242c49/test/integration/auth/podsecurity_test.go

##### e2e tests

[AppArmor node E2E][https://github.com/kubernetes/kubernetes/blob/2f6c4f5eab85d3f15cd80d21f4a0c353a8ceb10b/test/e2e_node/apparmor_test.go]
  - These tests are guarded by the `[Feature:AppArmor]` tag and run as part of the
    [containerd E2E features](https://testgrid.k8s.io/sig-node-containerd#node-e2e-features)
    test suite.

### Failure and Fallback Strategy

There are different scenarios in which applying an AppArmor profile may fail,
below are the ones we mapped and their outcome once this KEP is implemented:

| Scenario                                                                                           | API Server Result    | Kubelet Result                                                                                                                                        |
| -------------------------------------------------------------------------------------------------- | -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1) Using localhost or explicit `runtime/default` profile when container runtime does not support AppArmor.  | Pod created          | The outcome is container runtime dependent. In this scenario containers may 1) fail to start or 2) run normally without having its policies enforced. |
| 2) Using custom or `runtime/default` profile that restricts actions a container is trying to make. | Pod created          | The outcome is workload and AppArmor dependent. In this scenario containers may 1) fail to start, 2) misbehave or 3) log violations.                  |
| 3) Using a localhost profile that does not exist on the node.                                      | Pod created          | Container runtime dependent: containers fail to start. Retry respecting RestartPolicy and back-off delay.  Error message in event.                |
| 4) Using an unsupported runtime profile (i.e. `runtime/default-audit`).                            | Fails validation: pod **not** created. | N/A                                                                                                                                                   |
| 5) Using localhost or explicit `runtime/default` profile when AppArmor is disabled by the host or build | Pod created. | Kubelet puts Pod in blocked ssstate.                                                                                                                    |
| 6) Using implicit (default) `runtime/default` profile when AppArmor is disabled by the host or build. | Pod created | Container created without AppArmor enforcement. |

Scenario 2 is the expected behavior of using AppArmor and it is included here
for completeness.

### Version Skew Strategy

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
annotations stripped.

###### What happens if we reenable the feature if it was previously rolled back?

Newly started or restarted containers in pods that still have the AppArmor annotations will
have the specified AppArmor profile applied, rather than the runtime default.

###### Are there any tests for feature enablement/disablement?

No.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

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

Yes.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring requirements

###### How can an operator determine if the feature is in use by workloads?

The feature is built into the kubelet and api server components. No metric is
planned at this moment. The way to determine usage is by checking whether the
pods/containers have a AppArmorProfile set.

###### How can someone using this feature know that it is working for their instance?

The AppArmor enforcement status is not directly surfaced by Kubernetes, but is visible through the
linux proc API:

```sh
$ cat /proc/1/attr/current
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
