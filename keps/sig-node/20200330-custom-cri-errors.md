---
title: Custom errors in the CRI
authors:
  - "@mrunalp"
owning-sig: sig-node
reviewers:
  - "@derekwaynecarr"
  - "@dchen1107"
approvers:
  - "@derekwaynecarr"
  - "@dchen1107"
editor: "@mrunalp"
creation-date: 2020-03-16
last-updated: 2020-03-30
status: 
---
# Custom errors in the CRI
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Options](#options)
    - [Option 1: Introduce an error package for the CRI](#option-1-introduce-an-error-package-for-the-cri)
    - [Option 2: Add additional states for pods and containers](#option-2-add-additional-states-for-pods-and-containers)
    - [Option 3: Use grpc rpc status and add helper package](#option-3-use-grpc-rpc-status-and-add-helper-package)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist
- [ ] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


## Summary
Add custom errors to the CRI for common error scenarios to make the kubelet 
more efficient.

## Motivation
Kubelet spends a lot of time during termination determining whether a pod or a 
container is still running. Implementing some basic custom errors types for
atleast the 'not found' case should improve the time it takes for the kubelet
to tear down.

### Goals
Introduce custom errors in the CRI to cover the most common error scenarios
that the kubelet will benefit from.

### Non-Goals
Introduce a custom error for every possible error that we return from
the container runtimes.

## Proposal

### User Stories

#### Story 1
Detect containers and pods not found by the container runtimes, so the kubelet could exit
early out of some loops.

#### Story 2
Inform the kubelet to retry an operation after some time if we are waiting for
some condition to be satisfied.

### Options

#### Option 1: Introduce an error package for the CRI
```go
k8s.io/cri-api/pkg/errors

const (
    ErrNotFound = errors.New("not found")
    ...
)
```

Pros:
- Easy to add support first class errors for golang.

Cons:
- No support for container runtime servers implemented in other languages.

#### Option 2: Add additional states for pods and containers
```
-- a/staging/src/k8s.io/cri-api/pkg/apis/runtime/v1alpha2/api.proto
+++ b/staging/src/k8s.io/cri-api/pkg/apis/runtime/v1alpha2/api.proto
@@ -435,6 +435,7 @@ message LinuxPodSandboxStatus {
 enum PodSandboxState {
     SANDBOX_READY    = 0;
     SANDBOX_NOTREADY = 1;
+    SANDBOX_NOTFOUND = 2;
 }
 
 // PodSandboxStatus contains the status of the PodSandbox.
@@ -828,10 +829,11 @@ message RemoveContainerRequest {
 message RemoveContainerResponse {}
 
 enum ContainerState {
-    CONTAINER_CREATED = 0;
-    CONTAINER_RUNNING = 1;
-    CONTAINER_EXITED  = 2;
-    CONTAINER_UNKNOWN = 3;
+    CONTAINER_CREATED  = 0;
+    CONTAINER_RUNNING  = 1;
+    CONTAINER_EXITED   = 2;
+    CONTAINER_UNKNOWN  = 3;
+    CONTAINER_NOTFOUND = 4;
 }
```

Pros:
- Easier to support different programming languages for container runtimes.

Cons:
- Needs more changes to support errors for the APIs. 
- Requires inspecting objects for errors on the client side which doesn't
  look idiomatic.

#### Option 3: Use grpc rpc status and add helper package
GRPC defines [error codes](https://github.com/grpc/grpc/blob/master/doc/statuscodes.md)
that we can make use of with a helper package.

```
c, err := myRuntimeGetContainerById(id)
if err != nil {
    return nil, status.Errorf(codes.NotFound, "container not found")
}

k8s.io/cri-api/pkg/errors

func IsNotFound(err error) bool {

}

```

Pros:
- Uses the error infrastructure from grpc and protobufs.
- Works across programming langauges.
- The helper package allows us to change implementation in the future
  if needed.
- Enables us to optimize kubelet further without needing to do major code
  sweep given that kubelet always has to handle the error.


## Design Details
Implement Option 3 as described above.

### Test Plan
Add tests to cri-tests to ensure that runtimes return the right error codes.
It will be difficult to add cri-tests for the dockershim till it is moved
out of tree.


### Graduation Criteria

- Container runtimes like CRI-O and containerd adopt the custom errors.
- Add tests to cri-test test to check that the custom errors are returned correctly.
- Add e2e test to check for the custom errors.

## Implementation History
TBD
