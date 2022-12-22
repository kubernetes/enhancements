
  

<!--

**Note:** When your KEP is complete, all of these comment blocks should be removed.

  

To get started with this template:

  

- [ ] **Pick a hosting SIG.**

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

# KEP-3702: IMA namespace support inside containers

  

<!--

This is the title of your KEP. Keep it short, simple, and descriptive. A good

title can help communicate what the KEP is and should be considered as part of
7
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

  

IMA namespaces allow to check the file integrity. This proposal adds file integrity inside containers deployed in kubernetes.

  
  

## Motivation

  

File integrity is a way to improve security in systems allowing to:

* Detect illicit activity

* Detect unintended changes

* Verify the status and health of the system

* Comply with access rules

  

This can be achieved using IMA (Integrity Measurement Architecture) and EVM (Extended Verification Module), which can use a TPM chip as Hardware Root of Trust for high security environments.

  

### Goals

  

* Allow IMA to work inside a container with remote attestation

  

### Non-Goals

  

## Proposal
  
We propose to enable IMA linux namespaces in pods.

Since IMA namespaces can be created when a container is launched, we can provide transparent integrity verification on any linux container.

IMA and EVM can use a TPM chip as a hardware root of trust. Hence we can verify images against a set of golden hash values, as well as avoiding any further changes to the overlayfs to intercept calls and check the integrity of files.


### User Stories (Optional)

  

#### Story 1

As a cluster admin, I want to detect undesired file changes, so that I can take out pods that have been compromised.

  

#### Story 2

  
As a cluster admin, I want to deploy a only proven and certified pods, so that I can comply with internal policies as well as security regulations.

#### Story 3

  
As a cluster admin, I want to deny access to certain files inside the pod, so that a potential intruder can't access sensitive information.

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


There will be a CRI API change which will allow the pod to use IMA namespaces and specify the namespace policy.

  

### Linux kernel

  

IMA is only available in Linux hosts and Linux containers. Unfortunately, IMA is not a separate namespace, which is needed in order to isolate it and be used inside containers. Upcoming kernel patches should add support for IMA namespaces.

  

### Runtime specification

In order to run a container we need a bundle that contains a config.json file with all the configuration. There is a Linux specific configuration section where the namespaces are listed. According to the standard, the following namespace types SHOULD be supported.

  

* pid

* network

* mount

* ipc

* uts

* user

* cgroup

  

We suggest to add a new namespaces, initially as a OPTIONAL type, in order to keep the backward compatibility.

  

### CRI API

  

We propose to add the following message.

  

```protobuf

message NamespaceOptions {
  bool  ima = 6;
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

  

This KEP is a policy KEP, not a feature KEP. It will start as GA.

  

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
