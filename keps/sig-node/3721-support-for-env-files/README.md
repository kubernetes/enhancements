# KEP-3721: Support for env files

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
  - [Pod API](#pod-api)
  - [Failure and Fallback Strategy](#failure-and-fallback-strategy)
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

This is a proposal to add capability to populate environment variables in the
container from a file.

## Motivation

Kubernetes provides existing mechanisms for setting environment variables within
containers using ConfigMap and Secret resources. However, these methods
necessitate additional API calls. If a user wishes to populate environment
variables directly from a file, a workaround requiring the execution of the
`source <envfile>` command within the primary container is necessary. This
approach is unsuitable in cases where the user cannot modify the container image
or execute commands within it.

The syntactic enhancement proposed in this KEP offers a simplified method for
populating environment variables directly from files. This change would
streamline the process for users.

### Goals

- Define container environment variables from a file.

## Proposal

### User Stories (Optional)

#### Story 1

The user wants to define container environment variables from a file. It can
generate an env file using an `initContainer` or mount from the host. The
container can then use the env file to define its environment variables. In the
example below, the Pod's output includes `CONFIG_VAR=HELLO`.

```
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  initContainers:
  - name: setup-envfile
    image: registry.k8s.io/busybox
    command: ['sh', '-c', 'echo CONFIG_VAR="HELLO" > /config.env']
    volumeMounts:
    - name: data
      mountPath: /
  containers:
  - name: use-envfile
    image: registry.k8s.io/busybox
    command: [ "/bin/sh", "-c", "env" ]
    envFrom:
    - fileRef:
        path: /config.env
    volumeMounts:
    - name: data
      mountPath: /
  restartPolicy: Never
  volumes:
  - name: data
    emptyDir: {}
```

#### Story 2

The user wants to define container environment variables based on values from
an env file. It can generate an env file using an `initContainer` or mount from
the host. The container can then use the env file to define its environment
variables. In the example below, the Pod's output includes `SPECIAL_VAR=HELLO`.


```
apiVersion: v1
kind: Pod
metadata:
  name: dapi-test-pod
spec:
  initContainers:
  - name: setup-envfile
    image: registry.k8s.io/busybox
    command: ['sh', '-c', 'echo CONFIG_VAR="HELLO" > /config.env']
    volumeMounts:
    - name: data
      mountPath: /
  containers:
  - name: use-envfile
    image: registry.k8s.io/busybox
    command: [ "/bin/sh", "-c", "env" ]
    env:
    - name: SPECIAL_VAR
      valueFrom:
        fileKeyRef:
          path: /config.env
          key: CONFIG_VAR
    volumeMounts:
    - name: data
      mountPath: /
  restartPolicy: Never
  volumes:
  - name: data
    emptyDir: {}
```

## Design Details

Pods and PodTemplate will include `FileEnvSource` field under [`EnvFromSource`](https://pkg.go.dev/k8s.io/api/core/v1#EnvFromSource)
and `FileKeySelector` field under [`EnvVarSource`](https://pkg.go.dev/k8s.io/api/core/v1#EnvVarSource)
that can be set for an individual container.

### Pod API

`FileEnvSource` field will allow the users to populate environment variables
from the specified file. `Path` field should point to the file in key-value
format on the container filesystem.

```
type EnvFromSource struct {
    ...
    // The file to select from
    // +optional
    FileRef *FileEnvSource `json:"fileRef,omitempty" protobuf:"bytes,2,opt,name=fileRef"`
    ...
}

type FileEnvSource struct {
    // The file path to select from.
    Path string `json:",inline" protobuf:"bytes,1,opt,name=path"`
    // Specify whether the file must exist.
    // +optional
    Optional *bool `json:"optional,omitempty" protobuf:"varint,2,opt,name=optional"`
}
```

`FileKeySelector` field will allow the users to select the value for environment
variable from the specified key in the file. `Path`field should point to the
file in key-value format on the container filesystem. `Key` should be one of the
keys in the specified file.

```
type EnvVarSource struct {
    ...
    // Selects a key of the env file.
    // +optional
    FileKeyRef *FileKeySelector `json:"fileKeyRef,omitempty" protobuf:"bytes,4,opt,name=fileKeyRef"`
    ...
}

type FileKeySelector struct {
    // The file path to select from.
    Path string `json:",inline" protobuf:"bytes,1,opt,name=path"`
    // The key of the env file to select from.  Must be a valid key.
    Key string `json:"key" protobuf:"bytes,2,opt,name=key"`
    // Specify whether the file or its key must be defined
    // +optional
    Optional *bool `json:"optional,omitempty" protobuf:"varint,3,opt,name=optional"`
}
```

### Failure and Fallback Strategy

There are different scenarios in which applying `FileEnvSource` and
`FileKeySelector` fields may fail. Below are the ones we mapped and their
outcome once this KEP is implemented.

|Scenario| API Server Result | Kubelet Result |
|--------|-------------------|----------------|
|1. The filepath specified in `FileEnvSource` field does not exist | Pod created | Container fails to start and error message in event.|
|2. The filepath specified in `FileEnvSource` field does not exist but `optional` field is set to true. | Pod created | Container starts and env vars are not populated.|
|3. Either the filepath or key specified in `FileKeySleector` field does not exist | Pod created | Container fails to start and error message in event.|
|4. Either the filepath or key specified in `FileKeySleector` field exist but `optional` field is set to true | Pod created | Container starts and env vars are not populated. |


### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No

##### Unit tests

- [Pod Validation
  tests](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/core/validation/validation_test.go)
- TODO

##### Integration tests

- Pods tests: https://github.com/kubernetes/kubernetes/blob/91ee30074bee617d52fc24dc85132fe948aa5153/test/integration/pods/pods_test.go

##### e2e tests

The tests will be guarded by the `[Feature:EnvFiles]` tag.
- Pods e2e test: https://github.com/kubernetes/kubernetes/blob/91ee30074bee617d52fc24dc85132fe948aa5153/test/e2e/common/node/pods.go

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag.
- unit and integration tests completed.
- Initial e2e tests completed and enabled.

#### Beta

- Feature gate is enabled by default.
- Allowing time for feedback.

#### GA

- TBD

### Upgrade / Downgrade Strategy

kube-apiserver should be updated before kubelets in that order. Upgrade involves
draining all pods from the node, update to a matching kubelet and making the
node schedulable again. Downgrade involves doing the above in reverse.

### Version Skew Strategy

Since kubelet versions must not be ahead of kube-apiserver versions, the older
versions of kubelet or if the feature gate is disabled in kubelet, the fields
will be ignored.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: EnvFiles
  - Components depending on the feature gate:
    - kube-apiserver
    - kubelet

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, Containers already running will have their environment variables populated
from the specified file. But on restart, the fields will be stripped and the
feature will no longer work.

###### What happens if we reenable the feature if it was previously rolled back?

Newly started or restarted containers will have their environment variables
populated from the specified file.

###### Are there any tests for feature enablement/disablement?

No

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The [Version Skew Strategy](#version-skew-strategy) section covers this point.
It will not impact any already running workloads.


###### What specific metrics should inform a rollback?

An increase in pod validation errors can indicate issues with the field
translation. These would show up as `code=400` (Bad Request) errors in
`apiserver_request_total`.

The following errors could indicate problems with how kubelets are working
with this new fields:

- `started_containers_errors_total`
- `started_pods_errors_total`

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Automated tests will cover the scenarios with and without the changes proposed
on this KEP. As defined under [Version Skew Strategy](#version-skew-strategy),
we are assuming the cluster may have kubelets with older versions (without
this KEP' changes), therefore this will be covered as part of the new tests.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The feature is built into the kubelet and api server components. No metric is
planned at this moment. The way to determine usage is by checking whether the
pods/containers have a `FileEnvSource` or `FileKeySelector` fields set.

###### How can someone using this feature know that it is working for their instance?

The user can exec into the running container and check that the environment
variable is set:

```
$ kubectl exec -n $NAMESPACE $POD_NAME -- env
CONFIG_VAR=HELLO
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

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No impact to the running workloads

###### What are other known failure modes?

No impact is being foreseen to the running workloads based on the nature of
changes brought by this KEP.

Although some general errors and failures can be seen on [Failure and Fallback
Strategy](#failure-and-fallback-strategy).

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

* 2023/02/15: Initial proposal

## Drawbacks

No drawbacks identified yet.

## Alternatives

There are other ways to implement this feature. For eg, populating environment
variables from a ConfigMap or Secret resource. But this proposal allows
populating environment variables from a file in container filesystem and thus
reducing the need of an API call or extra permissions to kube-apiserver.

## Infrastructure Needed (Optional)

No new infrastructure needed.
