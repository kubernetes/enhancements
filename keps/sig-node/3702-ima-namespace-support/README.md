# KEP-3702: IMA namespace support inside containers

- [KEP-XXXXX: IMA namespace support inside containers](#kep-xxxxx-ima-namespace-support-inside-containers)
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
    - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [Linux kernel](#linux-kernel)
    - [Runtime specification](#runtime-specification)
    - [CRI API](#cri-api)
    - [Kubernetes pod resource](#kubernetes-pod-resource)
    - [Monitoring and alerting](#monitoring-and-alerting)
    - [Test Plan](#test-plan)
    - [Graduation Criteria](#graduation-criteria)
      - [GA](#ga)
    - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Version Skew Strategy](#version-skew-strategy)
  - [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

  

[kubernetes.io]:  https://kubernetes.io/

[kubernetes/enhancements]:  https://git.k8s.io/enhancements

[kubernetes/kubernetes]:  https://git.k8s.io/kubernetes

[kubernetes/website]:  https://git.k8s.io/website


## Summary

IMA namespaces allow to check the file integrity. This proposal adds regular file integrity inside containers deployed in kubernetes.


## Motivation

File integrity is a way to improve security in systems allowing to:

* Detect illicit activity
* Detect unintended changes
* Verify the status and health of the system
* Comply with access rules

This can be achieved using IMA (Integrity Measurement Architecture) and EVM (Extended Verification Module), which can use a TPM chip as Hardware Root of Trust for high security environments.

### Goals

* Check integrity of regular files inside containers, and hence check POD integrity
* Using remote attestation mechanism, check the integrity of a given POD or all PODS in the cluster deployed with IMA namespace enabled periodically
* Alert about corrupted or compromised pods

### Non-Goals
.
  

## Proposal
<!-- We propose to enable IMA linux namespaces in pods.
Since IMA namespaces can be created when a container is launched, we can provide transparent integrity verification on any linux container.
IMA and EVM can use a TPM chip as a hardware root of trust. Hence we can verify images against a set of golden hash values, as well as avoiding any further changes to the overlayfs to intercept calls and check the integrity of files. -->

### User Stories (Optional)

#### Story 1
As a cluster admin, I want to detect undesired file changes, so that I can take out pods that have been compromised.

An intruder perform some malicious changes inside a certain pod's files. The system should be able to detect those changed and alert about the inconsistent pod. The remote attestation framework could keep this pod running or make a copy (for forensic analysis) and delete it.

#### Story 2
As a cluster admin, I want to deploy a only proven and certified pods, so that I can comply with internal policies as well as security regulations.

Let's say that we are working in a high security environment where only approved images can be deployed. In this scenario we can make sure that the pod deploy used an imaged that hasn't been tampered.

#### Story 3
As a cluster admin, I want to deny access to certain files inside the pod, so that a potential intruder can't access sensitive information.

In some cases, we should not even allow root to modify certain files inside the container.

### Notes/Constraints/Caveats (Optional)
We need to enable IMA in the kernel and container runtimes (runC, CRI-O, docker, containerd, etc.). 

### Risks and Mitigations

  

## Design Details
  

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

In order for the this feature to work, the nodes where the pods will be deploy should have IMA enabled and a recent kernel that supports IMA namespaces (WIP, it should be merged soon). The nodes should also have a TPM chip. We could use nodAffinity of labels and annotations in nodes in order to select where to deploy the pods.

The linux kernel IMA namespace support is based on user namespaces. Therefore, the container runtime should first create a user namespace and then create an IMA namespaces. In order to use IMA namespaces it is necessary to enable user namespaces as well.

Should we enable IMA namespaces by default when enabling user namespaces?

### Linux kernel

IMA is only available in Linux hosts and Linux containers. Unfortunately, IMA is not a separate namespace, which is needed in order to isolate it and be used inside containers. Upcoming kernel patches should add support for IMA namespaces.


### Runtime specification

There is an ongoing discussion regarding the runtime changes.

https://github.com/opencontainers/runtime-spec/pull/1164


### CRI API

We propose to add the following message.

  

```protobuf

message LinuxSandboxSecurityContext {
    NamespaceOption namespace_options = 1;
    SELinuxOption selinux_options = 2;
    Int64Value run_as_user = 3;
    Int64Value run_as_group = 8;
    bool readonly_rootfs = 4;
    repeated int64 supplemental_groups = 5;
    bool privileged = 6;
    SecurityProfile seccomp = 9;
    SecurityProfile apparmor = 10;
    string seccomp_profile_path = 7 [deprecated=true];
    // new field
    bool ima = 11;
}

```
### Kubernetes pod resource
We propose the following change in the podTemplate to enable IMA namespaces.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
spec:
  securityContext:
  # New field

    ima: true
  containers:
  - name: nginx
    image: nginx:1.14.2
    ports:
    - containerPort: 80
```

The item pod.spec.ima.policy will automatically enable IMA for all the container in the pod with a given policy. Since all containers in a pod share the same namespaces, we need to have this policy in advance when creating the pod and its infrastructure container.


### Monitoring and alerting

This features will integrate with a future remote attestation procedure, which will monitor pods and in case of a violation take some actions like pod revocation, alerting, etc.

### Test Plan



Which unit tests should we include?



### Graduation Criteria


#### GA


### Upgrade / Downgrade Strategy


### Version Skew Strategy

## Production Readiness Review Questionnaire

  

Not applicable because this is a policy KEP.

  

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