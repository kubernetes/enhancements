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
- [Design Details](#design-details)
  - [Pod API](#pod-api)
  - [Env File](#env-file)
  - [Failure and Fallback Strategy](#failure-and-fallback-strategy)
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
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
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
containers using ConfigMap and Secret resources. However, these mechanisms have
a few drawbacks in following use-cases:

1. An `initContainer` creates some configuration that needs to be consumed as an
   environment variable by the main container. Existing mechanisms necessitate
   additional ConfigMap or Secret API calls.

2. when each k8s Node is initialized with an environment file that is used by
   Pods running on that node, k8s can provide a mechanism to instantiate env
   vars from this file.

3. The user is using a container offered by a vendor that requires configurating
   environment variables (for eg, license key, one-time secret tokens). They can
   use an `initContainer` to fetch this info into a file and the main container
   can consume.

The syntactic enhancement proposed in this KEP offers a simplified method for
populating environment variables directly from files. This change would
streamline the process for users.

### Goals

1. Support instantiating a container's environment variables from a file. This
file must be in emptyDir volume. The env file can be created by an initContainer
or sidecar in the emptyDir volume. kubelet will instantiate the env vars in the
container from the specified file in the emptyDir volume but it will not mount
the file.

2. All containers (container, initContainer, sidecar and ephemeral container)
will be able to load env vars from a file.

### Non-Goals

1. We do not plan to support expansion of environment variables, that is, when an
environment variable refers to another.

2. We do not intend to update env vars if the file content has changed after the
container consuming it has started.

## Proposal

### User Stories (Optional)

#### Story 1

The user is using a container offered by a vendors that requires configuring
env vars. [Some
vendors](https://developer.hashicorp.com/vault/docs/platform/k8s/injector/examples#environment-variable-example)
have to couple their app with Kubernetes to allow setting env vars in their
container.

Instead, k8s can provide a new mechanism that will be used to instantiate
env vars from the specified file. The user can generate an env file using an
`initContainer`. The container can then use the `fileRef` field to reference
the file from which it should initialize the env vars. Behind the scene, kubelet will
parse the env file and instantiate these env vars during container creation. It
does not mount this file onto the main container. In the example below,
the Pod's output includes `CONFIG_VAR=HELLO`.

```
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  initContainers:
  - name: setup-envfile
    image: registry.k8s.io/busybox
    command: ['sh', '-c', 'echo "CONFIG_VAR: HELLO" > /etc/config/config.yaml']
    volumeMounts:
    - name: data
      mountPath: /etc/config
  containers:
  - name: use-envfile
    image: registry.k8s.io/busybox
    command: [ "/bin/sh", "-c", "env" ]
    envFrom:
    - fileRef:
        path: config.yaml
        volumeName: data
  restartPolicy: Never
  volumes:
  - name: data
    emptyDir: {}
```

#### Story 2

Each k8s Node is instantiated with an env file that contains information about
the Node (for eg, instance type, failure zones). The user would like to
provide some of the information in this file to the vendor's container via
env vars. For example, consider that each node has the following env file:

```
CONFIG_VAR: HELLO
CONFIG_VAR_A: WORLD
...
```

The user can use the `fileKeyRef` field to select a subset of these env vars
that it wants to provide to the app. Behind the scene, kubelet will parse the
env file on the host and instantiate these env vars when creating the main
container. It does not mount this hostPath volume onto the main container.
In this case, the Pod's output includes `CONFIG_VAR=HELLO`

```
apiVersion: v1
kind: Pod
metadata:
  name: dapi-test-pod
spec:
  initContainers:
  - name: setup-envfile
    image: registry.k8s.io/busybox
    command: ['sh', '-c', 'cp /etc/config/config.yaml /data/config.yaml']
    volumeMounts:
    - name: config
      mountPath: /data
    - name: data
      mountPath: /etc/config
   containers:
  - name: use-envfile
    image: registry.k8s.io/busybox
    command: [ "/bin/sh", "-c", "env" ]
    env:
    - name: CONFIG_VAR
      valueFrom:
        fileKeyRef:
          path: config.yaml
          volumeName: config
          key: CONFIG_VAR
  restartPolicy: Never
  volumes:
  - name: config
    emptyDir: {}
  - name: data
    hostPath: /etc/config
```

## Design Details

Pods and PodTemplate will include `FileEnvSource` field under [`EnvFromSource`](https://pkg.go.dev/k8s.io/api/core/v1#EnvFromSource)
and `FileKeySelector` field under [`EnvVarSource`](https://pkg.go.dev/k8s.io/api/core/v1#EnvVarSource)
that can be set for an individual container.

### Pod API

`FileEnvSource` field will allow the users to populate environment variables
from the specified file. The user can specify a file using `VolumeName` and
`Path` fields. The `VolumeName` field specifies the volume mount that contains
the file and `Path` is the relative path in this volume mount filesystem.

```
type EnvFromSource struct {
    ...
    // The file to select from
    // +optional
    FileRef *FileEnvSource `json:"fileRef,omitempty" protobuf:"bytes,3,opt,name=fileRef"`
    ...
}

type FileEnvSource struct {
    // The name of the volume mount containing the env file.
    VolumeName string `json:",inline" protobuf:"bytes,1,opt,name=volumeName"`
    // The relative file path inside the volume mount to select from.
    Path string `json:",inline" protobuf:"bytes,2,opt,name=path"`
    // Specify whether the file must exist.
    // +optional
    Optional *bool `json:"optional,omitempty" protobuf:"varint,3,opt,name=optional"`
}
```

`FileKeySelector` field will allow the users to select the value for environment
variable from the specified key in the file.

```
type EnvVarSource struct {
    ...
    // Selects a key of the env file.
    // +optional
    FileKeyRef *FileKeySelector `json:"fileKeyRef,omitempty" protobuf:"bytes,5,opt,name=fileKeyRef"`
    ...
}

type FileKeySelector struct {
    // The name of the volume mount containing the env file.
    VolumeName string `json:",inline" protobuf:"bytes,1,opt,name=volumeName"`
    // The relative file path inside the volume mount to select from.
    Path string `json:",inline" protobuf:"bytes,2,opt,name=path"`
    // The key of the env file to select from.  Must be a valid key.
    Key string `json:"key" protobuf:"bytes,3,opt,name=key"`
    // Specify whether the file or its key must be defined
    // +optional
    Optional *bool `json:"optional,omitempty" protobuf:"varint,4,opt,name=optional"`
}
```

### Env File

The full specification of an env file:

1. **File Format**: The environment variable (env) file must adhere to valid YAML syntax to ensure correct parsing.

2. **Variable Naming**:
    
    a. **Standard**: Environment variable names should be composed of alphanumeric characters (letters and numbers), underscores, dots, or hyphens. The first character cannot be a digit.

    b. **Relaxed**: If the Kubernetes feature gate `RelaxedEnvironmentVariableValidation` is enabled, the naming restriction is less stringent, allowing any printable ASCII character except the equals sign (=).

3. **Duplicate Names**: If an environment variable is defined multiple times in the file, the last occurrence takes precedence and overrides any previous values.

4. **Size Limit**: The maximum allowed size for the env file is 1 MiB.

5. **File Location**: The env file must be placed within the `emptyDir` volume associated with the pod. If it is not found in the correct location, the Kubernetes API server will reject the pod creation request.

6. **Container Behavior**:
    
    a. **Startup**: At container startup, the kubelet (the Kubernetes node agent) will parse the env file from the emptyDir volume and inject the defined variables into the container's environment.
    
    b. **File Access**: The env file itself is not directly accessible within the container unless explicitly mounted by the container configuration.

7. **Dynamic Updates**: Currently, changes to the env file after container startup are not reflected in the container's environment until a restart. A potential future enhancement could involve dynamically updating environment variables when the file is modified.


### Failure and Fallback Strategy

There are different scenarios in which applying `FileEnvSource` and
`FileKeySelector` fields may fail. We plan to provide an erorr message that is
descriptive enough to troubleshoot the issue but it will not contain the env
file as it can contain sensitive information.

Below are the ones we mapped and their outcome once this KEP is implemented.

|Scenario| API Server Result | Kubelet Result |
|--------|-------------------|----------------|
|1. The volumeName specified in `FileEnvSource` or `FileKeySelector` field does not exist | Pod creation fails with an error |  |
|2. The volumeName specified in `FileEnvSource` or `FileKeySelector` is not an emptyDir | Pod creation fails with an error |  |
|3. The filepath specified in `FileEnvSource` field does not exist | Pod created | Container fails to start and error message in event.|
|4. The filepath specified in `FileEnvSource` field does not exist but `optional` field is set to true. | Pod created | Container starts and env vars are not populated.|
|5. Either the filepath or key specified in `FileKeySelector` field does not exist | Pod created | Container fails to start and error message in event.|
|6. Either the filepath or key specified in `FileKeySelector` field does not exist but `optional` field is set to true | Pod created | Container starts and env vars are not populated. |
|7. The specified file is not a parsable env file. | Pod created | Container fails to start and error message is reported in the events.|
|8. The specified file contains invalid env var names. | Pod created | Container fails to start and erorr message is reported in the events.|


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
this KEP' changes), therefore this will be covered of following new tests:

1. When the kubelet is upgraded, the env files will be instantiated in the
   container. On downgrade, the env files will be ignored but the pod will still
   run. On upgrade, the env files should be instantiated in the container again.

2. When the apiserver is upgraded, new envs will be written. But on downgrade,
   it cannot be written but the existing envs will continue to exist. On
   upgrade, the envs can again be written.

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
