# KEP-2763: Support Ambient Capabilities in Kubernetes.

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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Changes to kubernetes API (<a href="https://pkg.go.dev/k8s.io/api/core/v1">https://pkg.go.dev/k8s.io/api/core/v1</a>)](#changes-to-kubernetes-api-httpspkggodevk8sioapicorev1)
    - [Restricted ambient capabilities.](#restricted-ambient-capabilities)
  - [Changes to runtime API (<a href="https://pkg.go.dev/k8s.io/cri-api/pkg/apis/runtime/v1alpha2">https://pkg.go.dev/k8s.io/cri-api/pkg/apis/runtime/v1alpha2</a>)](#changes-to-runtime-api-httpspkggodevk8siocri-apipkgapisruntimev1alpha2)
  - [Changes to containerd/containerd (<a href="https://github.com/containerd/containerd">https://github.com/containerd/containerd</a>)](#changes-to-containerdcontainerd-httpsgithubcomcontainerdcontainerd)
  - [Order of changes](#order-of-changes)
  - [Test Plan](#test-plan)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

[Ambient capabilities](https://man7.org/linux/man-pages/man7/capabilities.7.html) is a set of capabilities that are preserved across an execve(2) of a program that is not privileged. This KEP proposes that kubernetes provide a way to set ambient capabilities for containers through the Pod manifest. It also proposes changes that must be made to `containerd` and `CRI-O` to enable ambient capabilities end-to-end.

## Motivation

Running containers as non-root has been a long recommended best-practice in kubernetes and we have published [blogs](https://kubernetes.io/blog/2018/07/18/11-ways-not-to-get-hacked/#8-run-containers-as-a-non-root-user) recommending this best practice. In addition to running as non-root it is also recommended that all capabilities other than the ones required are dropped from the container. Since most containers don’t require any capabilities this guidance becomes easy to follow.

NOTE: In the example below net_bind_service is only userd for demonstration purposes, in no way are we scoping this KEP just to net_bind_service. See the [User Stories (Optional)](#user-stories-optional) section below for other cases where ambient capabilities can help.

**The following works:-**
```Dockerfile
FROM ubuntu

COPY main /bin/simpleserver
```

```
docker build -t simpleserver:nofilecaps .
```

```yaml
# pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: static-web
  labels:
    role: myrole
spec:
  containers:
    - name: web
      securityContext:
              runAsUser: 1000
              runAsGroup: 1000
              allowPrivilegeEscalation: false
              capabilities:
                      drop: ["All"]
      image: simpleserver:nofilecaps
      command: ["simpleserver", "--port", "8080"]
```

```
kubectl apply -f pod.yaml
pod/static-web created

kubectl get pods                                                        
NAME         READY   STATUS    RESTARTS   AGE
static-web   1/1     Running   0          10s
```

**The following does not work:** because the container is running as non-root user `1000` and since it is mounting a privileged port it will require `CAP_NET_BIND_SERVICE` linux capability.

```yaml
# pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: static-web
  labels:
    role: myrole
spec:
  containers:
    - name: web
      securityContext:
              runAsUser: 1000
              runAsGroup: 1000
              allowPrivilegeEscalation: false
              capabilities:
                      drop: ["All"]
                      add: ["NET_BIND_SERVICE"]
      image: simpleserver:nofilecaps
      command: ["simpleserver", "--port", "80"]
```

```
kubectl apply -f pod.yaml
pod/static-web created

kubectl get pods
NAME         READY   STATUS             RESTARTS   AGE
static-web   0/1     CrashLoopBackOff   1          6s

kubectl logs static-web
2021/06/05 18:21:31 About to listen on port: 80
2021/06/05 18:21:31 http.ListenAndServe(:80, nil) failed with err: listen tcp :80: bind: permission denied
```

Capabilities that are either added explicitly in the manifest or by default to a non-root container do not get added to its effective and permitted set because effective and permitted sets get cleared when you transition from UID 0 to UID !0. To get around this today users have to apply the capabilities to the binary in their image build phase.

**A user could apply file capabilities as shown below :-**

```Dockerfile
FROM ubuntu

COPY simpleserver /bin/simpleserver

RUN apt-get update && apt-get -y --no-install-recommends install libcap2-bin

RUN setcap cap_net_bind_service=+ep /bin/simpleserver
```

```
docker build -t simpleserver:filecaps .
```

**Now the if we update the image and create the pod:-**

```yaml
# pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: static-web
  labels:
    role: myrole
spec:
  containers:
    - name: web
      securityContext:
              runAsUser: 1000
              runAsGroup: 1000
              allowPrivilegeEscalation: false
              capabilities:
                      drop: ["All"]
                      add: ["NET_BIND_SERVICE"]
      image: simpleserver:filecaps
      command: ["simpleserver", "--port", "80"]
```

```
kubectl apply -f pod.yaml
pod/static-web created

kubectl get pods
NAME         READY   STATUS             RESTARTS   AGE
static-web   0/1     CrashLoopBackOff   1          17s

kubectl logs static-web
2021/06/05 18:25:09 About to listen on port: 80
2021/06/05 18:25:09 http.ListenAndServe(:80, nil) failed with err: listen tcp :80: bind: permission denied
```

**Note** The above does not work because we cannot set `allowPrivilegeEscalation: false` anymore because `allowPrivilegeEscalation` directly controls the `no_new_privs` flag. With `no_new_privs` set, file capabilities are not added to the permitted set.

**Now updating the Pod by removing `allowPrivilegeEscalation: false`:-**

```yaml
# pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: static-web
  labels:
    role: myrole
spec:
  containers:
    - name: web
      securityContext:
              runAsUser: 1000
              runAsGroup: 1000
              capabilities:
                      drop: ["All"]
                      add: ["NET_BIND_SERVICE"]
      image: simpleserver:filecaps
      command: ["simpleserver", "--port", "80"]
```

```
kubectl apply -f pod.yaml
pod/static-web created

kubectl get pods
NAME         READY   STATUS    RESTARTS   AGE
static-web   1/1     Running   0          9s
```

**Note:** While applying the capabilities to the binary during the image build is a work-around, it should be noted that now even if we switched back to port 8080 in the above example we would have to add the capability in the container's `SecurityContext`. 

**If we update the port to non-privileged but use the setcap image:-**

```yaml
# pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: static-web
  labels:
    role: myrole
spec:
  containers:
    - name: web
      securityContext:
              runAsUser: 1000
              runAsGroup: 1000
              allowPrivilegeEscalation: false
              capabilities:
                      drop: ["All"]
      image: simpleserver:filecaps
      command: ["simpleserver", "--port", "8080"]
```

```
kubectl apply -f pod.yaml
pod/static-web created

kubectl get pods
NAME         READY   STATUS             RESTARTS   AGE
static-web   0/1     CrashLoopBackOff   1          11s

kubectl logs static-web
standard_init_linux.go:211: exec user process caused "operation not permitted"
```

**Now if we add cap_net_bind_service to the container even though it is not needed, it works**

```yaml
# pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: static-web
  labels:
    role: myrole
spec:
  containers:
    - name: web
      securityContext:
              runAsUser: 1000
              runAsGroup: 1000
              allowPrivilegeEscalation: false
              capabilities:
                      drop: ["All"]
                      add: ["NET_BIND_SERVICE"]
      image: simpleserver:filecaps
      command: ["simpleserver", "--port", "8080"]
```

```
kubectl apply -f pod.yaml
pod/static-web created

kubectl get pods
NAME         READY   STATUS    RESTARTS   AGE
static-web   1/1     Running   0          10s
```

While applying capabilities to the binary during image build is a work-around it is not always feasible to do so as a lot of users use 3rd party images, so in order to run containers as non-root in an image whose build you do not control you would essentially have to do something like:

```Dockerfile
FROM debian-base:buster-v1.4.0
COPY --from=imageThatIWantToRunAsNonRoot /binaryThatIWantToRunAsNonRoot /binaryThatIWantToRunAsNonRoot
RUN apt-get update \
    && apt-get -y --no-install-recommends install libcap2-bin
RUN setacp cap_my_binary_needs=+ep /binaryThatIWantToRunAsNonRoot

FROM imageThatIWantToRunAsNonRoot
# override the binary in the image
COPY --from=0 /binaryThatIWantToRunAsNonRoot /binaryThatIWantToRunAsNonRoot
```

Here we override the binary in the original image with a setcaped binary and push the image to our private repository. While this is a workaround to the problem if you don’t control the build, you are now forced to effectively maintain another copy of the image.

The way capabilities are today designed is that they are effective in limiting the capabilities of a user but we lack the ability to grant capabilities to a user unless we also control the build of an image. 

### Goals

- Enable support for ambient capabilities for Kubernetes and the `containerd` and `CRI-O` runtime.
- Define a safe set of capabilities that should be allowed to be added to the ambient set.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
- Runtimes other than `containerd` and `CRI-O` are not in scope at the moment.
- Windows is not covered in this design.

## Proposal

- We propose updating the kubernetes core and CRI API's to allow users to configure adding capabilities to the ambient set. This would allow users to run non-root containers without having to apply file capabilities to the binary during image build time. Details on how and where these changes will be made are explained in the [Changes to kubernetes API (<a href="https://pkg.go.dev/k8s.io/api/core/v1">https://pkg.go.dev/k8s.io/api/core/v1</a>)](#changes-to-kubernetes-api-httpspkggodevk8sioapicorev1) and [Changes to runtime API (<a href="https://pkg.go.dev/k8s.io/cri-api/pkg/apis/runtime/v1alpha2">https://pkg.go.dev/k8s.io/cri-api/pkg/apis/runtime/v1alpha2</a>)](#changes-to-runtime-api-httpspkggodevk8siocri-apipkgapisruntimev1alpha2) sections below.

- We also propose that container runtimes like `containerd` and `CRI-O` be updated to account for these changes to the CRI apis and add the requested capabilities to the ambient set when they create the containers. Details on how and where these changes will be made are explained in the [Changes to containerd/containerd (<a href="https://github.com/containerd/containerd">https://github.com/containerd/containerd</a>)](#changes-to-containerdcontainerd-httpsgithubcomcontainerdcontainerd) section below.

### User Stories (Optional)

<<[UNRESOLVED] How much demand is there for this feature outside of NET_BIND_SERVICE>>

_Blocking for Alpha._

Is there enough demand for this feature today?

<<[/UNRESOLVED]>>

#### Story 1
As a security conscious user I should be able to run my container as non-root and add the minimum set of capabilities required for it to function even when I do not control how the image is built.

#### Story 2
@Jc2K [commented](https://github.com/kubernetes/enhancements/pull/2757#issuecomment-906729690)
If you were running one of the popular IDS's or some other network monitoring tool like moloch/arike on your cluster as a `DaemonSet` then i'd expect you could run it as a non root user with CAP_NET_RAW (and possibly CAP_NET_ADMIN).

I personally have run `DaemonSet`'s that would use `CAP_NET_ADMIN` to configure some environment specific networking. I would have run this as non-root if I had this KEP.

Yes, `CAP_NET_ADMIN` is a pretty elevated permission. Most of the use cases I can think are things i'd expect the team deploying the CNI to be interested in, rather than the person deploying something on port 443 (where they are different people). I'd still prefer to deploy them not as root, where that actually works.

#### Story 3
@Jc2K [commented](https://github.com/kubernetes/enhancements/pull/2757#discussion_r697016426)
The most useful use of this is where you have to run something that does have to have pretty elevated permissions. Like an IDS, or a network visibility tool, or some daemon that configures some weird aspect of your network stack. You can't get away from the fact that these things are running and they are running with NET_ADMIN or that they are in the host network namespace. Let's set as many restrictions on them as we can and run them as non-root.

### Risks and Mitigations
One risk of adding this feature is that it could be used to give non-root users elevated permissions on the host. For example one could add `cap_sys_admin` to the ambient capabilities set of a container making it essentially root on host. To mitigate this we propose that for alpha we only support a carefully curated set of "safe" capabilities that are most commonly required by workload.

## Design Details

### Changes to kubernetes API (https://pkg.go.dev/k8s.io/api/core/v1)

<<[UNRESOLVED pick how we want to update the K8S APIs]>>

_Blocking for Alpha._

There are 2 options here:-
- Option 1: Reuse Add field in [Capabilities](https://pkg.go.dev/k8s.io/api/core/v1#Capabilities)

  When a capability gets added explicitly to a non-root container it also gets added to the ambient set in addition to getting added to inheritable, permitted, bounding and effective sets. The default capabilities are not added to the ambient set.

  Pros and Cons

  :+1: Simple add and drop API is easy to use from a user perspective.

  :+1: No new field needs to added to the K8S API Capabilities object.

  :-1: How would we tell the difference at API validation time? I assume we'd have to look at mustRunAsNonRoot, because the apiserver has no idea whether a textual runAsUser is root or not. mustRunAsNonRoot it designed to fail at the last minute, in the Kubelet, which is bad UX. This makes this approach very awkward. Difrrent behavior for root vs non-root might be confusing for users.

  :-1: Since we are changing the default behavior for non-root containers we might break existing user containers. Issues with linux capabilities have proven themselves to be very hard to debug in the past.

- Option 2: Add new field to [Capabilities](https://pkg.go.dev/k8s.io/api/core/v1#Capabilities)

  When a capability is added using this field only then does it get added to the ambient set in addition to inheritable, permitted, bounding and effective sets. The behavior is consistent for root and non-root containers.

  ```diff
  type Capabilities struct {
    // Added capabilities
    // +optional
    Add []Capability `json:"add,omitempty" protobuf:"bytes,1,rep,name=add,casttype=Capability"`
    // Removed capabilities
    // +optional
    Drop []Capability `json:"drop,omitempty" protobuf:"bytes,2,rep,name=drop,casttype=Capability"`
  + // Ambient capabilities to add
  + // +optional
  + Ambient []Capability `json:"ambient,omitempty" protobuf:"bytes,3,rep,name=ambient,casttype=Capability"`
  }
  ```

  Pros and Cons

  :+1: Since this is a new field existing user workloads are unaffected by this change.

  :+1: The capabilities struct only really translates to Linux Capabilities, adding a new field does make it easier for users who understand Linux capabilities to confiure them.

  :+1: Ambient caps are useful for root containers as well.A root process can control what caps it gives to child processes that are running as non-root. The capabilities list list for these non-root processes may not be the same as the root caps in Add.

  :-1: Requires changes to the K8S API Capabilities object.

<<[/UNRESOLVED]>>

`Ambient` capabilities will adhere to the following rules:

1. Ambient capabilities can only be added.
2. Default capabilities are not added to the ambient capabilities set. By default the container has an empty ambient capability set,
3. Only capability that are explicitly added in the manifest will be added to the ambient set.
4. Since the ambient capability set obeys the invariant that no capability can ever be ambient if it is not both permitted and inheritable, adding a capability to the ambient set will also add it to the permitted and inheritable set.

#### Restricted ambient capabilities.

<<[UNRESOLVED what is the set of capabilities that we should allow to be made ambient]>>

_Target Beta, non-blocking for Alpha._

Some capabilities like `CAP_SYS_ADMIN` and `CAP_DAC_OVERRIDE` make a non-root user very powerful and we should restrict them from being added to the ambient capabilities set.

<<[/UNRESOLVED]>>

### Changes to runtime API (https://pkg.go.dev/k8s.io/cri-api/pkg/apis/runtime/v1alpha2)

We propose adding a new field to the [Capability](https://pkg.go.dev/k8s.io/cri-api/pkg/apis/runtime/v1alpha2#Capability) struct

```diff
type Capability struct {
  // List of capabilities to add.
  AddCapabilities []string `protobuf:"bytes,1,rep,name=add_capabilities,json=addCapabilities,proto3" json:"add_capabilities,omitempty"`
  // List of capabilities to drop.
  DropCapabilities     []string `protobuf:"bytes,2,rep,name=drop_capabilities,json=dropCapabilities,proto3" json:"drop_capabilities,omitempty"`
  XXX_NoUnkeyedLiteral struct{} `json:"-"`
  XXX_sizecache        int32    `json:"-"`
+ // List of ambient capabilities to add.
+ AddAmbientCapabilities []string `protobuf:"bytes,3,rep,name=add_ambient_capabilities,json=addAmbientCapabilities,proto3" json:"add_ambient_capabilities,omitempty"`
}
```

Adding apabilities to the `ambient` field in the `capabilities` field of the containers `securityContext` will directly update this field, and these changes would live in the [convertToRuntimeCapabilities](https://github.com/kubernetes/kubernetes/blob/ea0764452222146c47ec826977f49d7001b0ea8c/pkg/kubelet/kuberuntime/security_context.go#L121:6) function.


### Changes to containerd/containerd (https://github.com/containerd/containerd)

`containerd` clears all ambient capabilities on container creation. See [this](https://github.com/containerd/containerd/blob/055c801ededcb7a5e82f47bdeed555cdf6c64bd8/pkg/cri/server/container_create_linux.go#L233) link for details. 

```go
// Clear all ambient capabilities. The implication of non-root + caps
// is not clearly defined in Kubernetes.
// See https://github.com/kubernetes/kubernetes/issues/56374
// Keep docker's behavior for now.
specOpts = append(specOpts,
	customopts.WithoutAmbientCaps,
	customopts.WithSelinuxLabels(processLabel, mountLabel),
)
```

We would need to update code at that link to add the ambient capabilities by calling the [WithAmbientCapabilities](https://github.com/containerd/containerd/blob/ab963e1cc16a845567a0e3e971775c29c701fcf8/oci/spec_opts.go#L858) function instead of the [WithoutAmbientCapabilities](https://github.com/containerd/containerd/blob/04f73e3f8a097d95111f8419fa136d196b3a8725/pkg/cri/opts/spec_linux.go#L354) function. 

### Order of changes
We propose that the changes be made in the following order:

1. Changes described in the [Changes to runtime API (<a href="https://pkg.go.dev/k8s.io/cri-api/pkg/apis/runtime/v1alpha2">https://pkg.go.dev/k8s.io/cri-api/pkg/apis/runtime/v1alpha2</a>)](#changes-to-runtime-api-httpspkggodevk8siocri-apipkgapisruntimev1alpha2) section are made first as both containerd and kubernetes rely on these.
2. Changes described in the [Changes to containerd/containerd (<a href="https://github.com/containerd/containerd">https://github.com/containerd/containerd</a>)](#changes-to-containerdcontainerd-httpsgithubcomcontainerdcontainerd) are made next.
3. Once a version of `containerd` that supports ambient capabilities is available we make the changes described in the [Changes to kubernetes API (<a href="https://pkg.go.dev/k8s.io/api/core/v1">https://pkg.go.dev/k8s.io/api/core/v1</a>)](#changes-to-kubernetes-api-httpspkggodevk8sioapicorev1) section.


### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->


### Graduation Criteria
Maturity level of this feature is defined by the AmbientCapabilities feature-gate.

#### Alpha

The alpha of this feature is about making sure that all the infrastructure
required to enable ambient capabilities is in place. It includes the following work:-

1. The following sections need to be resolved before Alpha:-
    - [ ] [User Stories (Optional)](#user-stories-optional)
    - [ ] [Changes to kubernetes API (<a href="https://pkg.go.dev/k8s.io/api/core/v1">https://pkg.go.dev/k8s.io/api/core/v1</a>)](#changes-to-kubernetes-api-httpspkggodevk8sioapicorev1)

2. AmbientCapabilities feature-gate added in default off configuration(feature is opt-in by default)
3. Changes to CRI API as described in section [Changes to runtime API (<a href="https://pkg.go.dev/k8s.io/cri-api/pkg/apis/runtime/v1alpha2">https://pkg.go.dev/k8s.io/cri-api/pkg/apis/runtime/v1alpha2</a>)](#changes-to-runtime-api-httpspkggodevk8siocri-apipkgapisruntimev1alpha2) implemented.
4. A version of containerd and CRI-O that properly sets the ambient capabilities as described in section [Changes to containerd/containerd (<a href="https://github.com/containerd/containerd">https://github.com/containerd/containerd</a>)](#changes-to-containerdcontainerd-httpsgithubcomcontainerdcontainerd) available.
5. Changes to k8s API as described in [Changes to kubernetes API (<a href="https://pkg.go.dev/k8s.io/api/core/v1">https://pkg.go.dev/k8s.io/api/core/v1</a>)](#changes-to-kubernetes-api-httpspkggodevk8sioapicorev1) implemented.
6. Update to kubelet code to sets ambient capabilities and these changes are hidden behind the feature-gate.
7. e2e tests with the version of containerd that supports ambient capabilities with the feature gate enabled in kubelet passing in TestGrid.

#### Beta

While alpha was focusd on setting up the infra for this feature, beta will focus on the safe enablement.

1. Collect feedback from Alpha and use information to guide the following sections need to be resolved before Beta.
    - [] [Restricted ambient capabilities.](#restricted-ambient-capabilities)
2. Update Pod validation to prevent restricted capabilities from being added based on how the section above is resolved.
3. Enabled `AmbientCapabilites` feature-gate by default. (feature is opt-out by default)
4. Thorough testing is already expected for alpha, but we will review our test coverage and fill any gaps prior to beta.

#### GA

<<[UNRESOLVED]>>
_Blocking for GA, non-blocking for Alpha and Beta._

- Examples of real world usage and positive user feedback.

<<[/UNRESOLVED]>>


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
There is no toil expected on upgrade of existing workloads. But, if a `container` has capabilities in the `ambient` field in  the `capabilities` field of the `securityContext` and the cluster is downgraded to n-1 then the workload with the new field wouldn't get the ambient capabilities and may fail.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->
If a kubelet that is not aware of the `ambient` field in `capabilities` field of the `securityContext` then it will simply ignore the field. We propose adding a taint/toleration/well known label to mark nodes that can support ambient capability requests.

If the node is running a version of `containerd`/`CRI-O` that does not support ambient capabilities then the workload may fail even if it is supported by kubelet and the `AmbientCapabilities` feature-gate is enabled. To prevent this from happening we plan to add documentation to specify the version of containerd and CRI-O that support ambient capabilities.

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

<!--
Pick one of these and delete the rest.
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: AmbientCapabilities
  - Components depending on the feature gate: `kubelet`, `kube-apiserver`

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
Yes

###### What happens if we reenable the feature if it was previously rolled back?

If there are no containers which specify the `ambient` capabilities to add in their `securityContext` then nothing will change.
If there are containers which specify `ambient` capabilities in their `securityContext` then these containers will have the capability or capabilities in their ambient capability set.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->
The necessary unit tests in kubelet dealing with creation of Pods with ambient capabilities with the feature gate on and off will be added.

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
No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
No,

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
No.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

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
N/A.

## Alternatives
- Including `NET_BIND_SERVICE` in the ambient set by default.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
