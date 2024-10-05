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

2. The user is using a container offered by a vendor that requires configurating
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
container consuming it has started, unless the container is restarted for some
other reason

3. We do not intend to use a change in the file content to trigger a container
restart.

4. We do not intend to support instantiating all variables from an env file,
   without knowing their names a priori.

## Proposal

### User Stories (Optional)

#### Story 1

Distroless containers, known for their minimal footprint, lack a shell, making
traditional environment variable configuration via env files impossible. While
Kubernetes ConfigMaps and Secrets can be used to set env vars, they fall short
the user needs to set per-pod replica env vars.

To address this, Kubernetes can introduce a new mechanism allowing env vars to
be populated directly from a file. Users can employ an `initContainer` to
generate this file, then utilize the `fileKeyRef` field within their main
container definition to specify its location. Kubernetes will then handle parsing
the file and setting the env vars during container creation, eliminating the need
to mount the file.


This approach provides a streamlined solution for env var configuration in
distroless containers. In the example below, the Pod's distroless container will
be instantiated with `CONFIG_VAR=HELLO`.

```
apiVersion: v1
kind: Pod
metadata:
  name: dapi-test-pod
spec:
  initContainers:
  - name: setup-envfile
    image: registry.k8s.io/busybox
    command: ['sh', '-c', 'echo "CONFIG_VAR=HELLO" > /data/config.env']
    volumeMounts:
    - name: config
      mountPath: /data
   containers:
  - name: use-envfile
    image: registry.k8s.io/distroless-app
    env:
    - name: CONFIG_VAR
      valueFrom:
        fileKeyRef:
          path: config.env
          volumeName: config
          key: CONFIG_VAR
  restartPolicy: Never
  volumes:
  - name: config
    emptyDir: {}
```

#### Story 2

Managing unique configs for each Pod in Kubernetes can be cumbersome. Traditionally,
Kubernetes ConfigMap are used to store configs and populate env vars,
but this approach becomes less efficient when dealing with per-pod configs.

To simplify this process, Kubernetes can introduce a new mechanism where an `initContainer`
can build a config file. The main application container then utilizes
the `fileKeyRef` field to reference this file and populate its env vars. Kubernetes
automatically parses the file and sets the env vars during container creation without
mounting the file directly.

This enhancement offers a more manageable way to handle per-pod config in Kubernetes.
In the example below, the Pod's output includes `CONFIG_MAIN=hello`.

```
apiVersion: v1
kind: Pod
metadata:
  name: dapi-test-pod
spec:
  initContainers:
  - name: setup-envfile
    image: registry.k8s.io/busybox
    command: ['sh', '-c', 'echo "CONFIG_INIT=hello" > /data/config.env']
    volumeMounts:
    - name: config
      mountPath: /data
   containers:
  - name: use-envfile
    image: registry.k8s.io/busybox
    command: [ "/bin/sh", "-c", "env" ]
    env:
    - name: CONFIG_MAIN
      valueFrom:
        fileKeyRef:
          path: config.env
          volumeName: config
          key: CONFIG_INIT
  restartPolicy: Never
  volumes:
  - name: config
    emptyDir: {}
```



## Design Details

Pods and PodTemplate will include `FileKeySelector` field under [`EnvVarSource`](https://pkg.go.dev/k8s.io/api/core/v1#EnvVarSource)
that can be set for an individual container.

### Pod API

`FileKeySelector` field will allow the users to select the value for environment
variable from the specified key in the file. The user can specify a file using `VolumeName` and
`Path` fields. The `VolumeName` field specifies the volume mount that contains
the file and `Path` is the relative path in this volume mount filesystem.


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
    // The key of the env file to select from. It must be a valid key within the
    // env file. An invalid key will prevent the pod from starting, even if it's
    // successfully created.
    Key string `json:"key" protobuf:"bytes,3,opt,name=key"`
    // Specify whether the file or its key must be defined. If the file or key
    // does not exist, then the env var is not published.
    // +optional
    Optional *bool `json:"optional,omitempty" protobuf:"varint,4,opt,name=optional"`
}
```

### Env File

The full specification of an env file:

1. **File Format**: The environment variable (env) file must adhere to valid [.env syntax](https://smartmob-rfc.readthedocs.io/en/latest/2-dotenv.html) to ensure correct parsing. An example:

```
KEY1=VALUE1
KEY2=VALUE2
...
```

2. **Variable Naming**: We will apply the same variable name [restrictions](https://github.com/kubernetes/kubernetes/blob/a7ca13ea29ba5b3c91fd293cdbaec8fb5b30cee2/pkg/apis/core/validation/validation.go#L2583-L2596) as other API-defined env vars.

3. **Duplicate Names**: If an environment variable is defined multiple times in the file, the last occurrence takes precedence and overrides any previous values.

4. **Size Limit**: To start with, the maximum allowed size for the env file will be 64KiB. Limits for key-value length will be added as a part of implementation after additional investigation.

5. **File Location**: The env file must be placed within the `emptyDir` volume associated with the pod. If it is not found in the correct location, the Kubernetes API server will reject the pod creation request.

6. **Container Behavior**:
    
    a. **Startup**: At container startup, the kubelet will parse the env file from the emptyDir volume and inject the defined variables into the container's environment. To avoid race condition with another container updating the env file, we will restrict mounting the emptyDir volume (containing the env file) in initContainer only. The env file can either be mounted or consumed in a container.
    
    b. **File Access**: The env file itself is not directly accessible within the container unless explicitly mounted by the container configuration. Since the env file must be in the `emptyDir` volume, it should be safe to instantiate the pod container with the specified env vars from this file.

### Failure and Fallback Strategy

There are different scenarios in which applying `FileKeySelector` fields may fail.
We plan to provide an erorr message that is descriptive enough to troubleshoot the
issue but it will not contain the env file as it can contain sensitive information.

Below are the ones we mapped and their outcome once this KEP is implemented.

|Scenario| API Server Result | Kubelet Result |
|--------|-------------------|----------------|
|1. The volumeName specified in `FileKeySelector` field does not exist | Pod creation fails with an error |  |
|2. The volumeName specified in `FileKeySelector` is not an emptyDir | Pod creation fails with an error |  |
|3. Either the filepath or key specified in `FileKeySelector` field does not exist | Pod created | Container fails to start and error message in event.|
|4. Either the filepath or key specified in `FileKeySelector` field does not exist but `optional` field is set to true | Pod created | Container starts and env vars are not populated. |
|5. The specified file is not a parsable env file. | Pod created | Container fails to start and error message is reported in the events.|
|6. The specified file contains invalid env var names. | Pod created | Container fails to start and erorr message is reported in the events.|
|7. The container's UID does not have permission to read the env file. | Pod created | Container fails to start and erorr message is reported in the events.|


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
from the specified file. But on restart, the fields will be skipped and the
feature will no longer work.

###### What happens if we reenable the feature if it was previously rolled back?

Newly started or restarted containers will have their environment variables
populated from the specified file.

###### Are there any tests for feature enablement/disablement?

Yes, the unit tests will be added along with alpha implementation.

1. Validate that the `fileKeyRef` field is dropped when FeatureGate is disabled.
2. Validate that the `fileKeyRef` field is not dropped when FeatureGate is
   enabled.
3. Validate that the kubelet instantiates the env vars from `fileKeyRef` when
   the FeatureGate is enabled.


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
