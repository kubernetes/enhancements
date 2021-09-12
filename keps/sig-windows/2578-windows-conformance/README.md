# KEP-2578: Windows Conformance

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non Goals](#non-goals)
- [Proposal](#proposal)
  - [Background](#background)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
    - [Story 4](#story-4)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Specification of Windows Core Conformance (Kubernetes 1.23)](#specification-of-windows-core-conformance-kubernetes-123)
    - [Basic Networking](#basic-networking)
    - [Basic Service accounts and local storage](#basic-service-accounts-and-local-storage)
    - [Basic Scheduling](#basic-scheduling)
    - [Basic performance](#basic-performance)
  - [Sub-specifications of Windows Conformance (Kubernetes 1.23)](#sub-specifications-of-windows-conformance-kubernetes-123)
    - [Windows HostProcess Conformance](#windows-hostprocess-conformance)
    - [Active Directory](#active-directory)
    - [Network Policies](#network-policies)
    - [Windows Service Proxying](#windows-service-proxying)
  - [Implementation](#implementation)
    - [Option 1 - Test tags](#option-1---test-tags)
    - [Option 2 - Sonobuoy implementation for convenience](#option-2---sonobuoy-implementation-for-convenience)
      - [Implementation details for sonobuoy](#implementation-details-for-sonobuoy)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta graduation](#alpha---beta-graduation)
    - [Beta-&gt;GA graduation](#beta-ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Alternatives](#alternatives)
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

In general, the ad-hoc usage of bespoke ginkgo tags is cubmbersome and makes adoption of Windows workloads on Kubernetes difficult to certify.  In addition to defining these specific definitional constructs for Windows conformance in this KEP, we also aim to implement these defintitions in a declarative, fully automated certification bundle via the sonobuoy (https://sonobuoy.io/) tool, as a quick and easy way for end-users to verify windows clusters.

This KEP first abstractly defines the various categories of Windows "conformance", differentiating between core conformance from feature sets that are of common interest.

Next, we define how these conformance tests can be run using *existing* Kubernetes ginkgo tags, without any additional tooling.

Finally, we prescribe the implementation of a generic "windows conformance" testing tool, which can be either hand-rolled (i.e. in a bash script) or, implemented as part of a broader initiative (i.e. inside of the sonobuoy.io, which is commonly used for Kubernetes verification).

The initial implementation of this KEP is mainly focused on standardizing Windows Core Conformance definitions.  Other features, such as `HostProcess`, will be refined and updated with new releases, as these APIs and their standardization evolve in the community.

## Motivation

Running Windows containers at scale is a requirement for many mature enterprises migrating applications Kubernetes.  The requirements of a Windows Kubelet are not comparable to a Linux Kubelet, however, because of the fact that Windows does not support the same storage, security, and priveliged container model that Linux does.  We thus need an unambiguous standard in place that is meaningfull and actionable for comparing Windows Kubernetes support between vendors.

### Goals

- Disambiguate the verification requirements for Windows Conformance testing in Kubernetes from the implementation of tests themselves.
- Specify a canonical mechanism for validating Windows Conformance which leverages the exsiting e2e test suite in a declarative way.
- Standardize the definitions for how we verify support of new Windows features in Kubernetes, and standardize the features *not* fully supported by Windows.

### Non Goals

- Modify the existing Kubernetes Conformance specification, which has linux specific tests.
- Modify existing tests inside of Kubernetes that would increase the bifurcation of testing implementation.
- Adding new tooling to Kubernetes core itself for testing Windows.

## Proposal

### Background

Currently, tools such as the Kubernetes end-to-end testing suite and Sonobuoy can be used to verify Kubernetes clusters, but the filters and, more importantly, the meaning behind certain tests can be ambiguous.

If we consider the various idiosyncracies of verifying Kubernetes on Windows we are immediately confronted with several potential hotspots that can be difficult to wade through as an end user.

- The ability to run linux containers on windows (LCOW) is a potential feature which may conflate the definition of a "Windows Kubelet" significantly, and may potentially result in false-positive test results where in multi-arch images are able to pass a Conformance suite without actually running any windows workloads.  This of course is an absurd outcome which can be solved by having a specification and implementation for Windows Conformance in place that goes beyond just specifying a set of tests. 
- Currently, there exists a `LinuxOnly` tag in the Kubernetes e2e testing suite, but this tag needs to be disambiguated.  We need to make explicit functional requirements for Windows nodes in this KEP so that this tag can be used as an implementation detail for how Windows clusters are verified.
- Some tests (including Networking, NetworkPolicy, and Storage related tests) behave differently on Windows by necessity, because of differences in the feature sets and testing idioms that are supportable on Windows Server.  We aim to distinguish the core requirements of such tests in terms of conformant Windows clusters, irrespective of implementation gaps which may exist between how Linux and Windows implement certain tests.
- Some critical windows functionality, such as the ability to run activeDirectory as an identity provider for a pod, need to be well-defined in a Kubernetes context for individuals planning on integrating production Windows workloads into their Kubernetes support model.
- The introduction of "Priveliged containers" (formally referred to as `HostProcess` pods), further differentiates Linux from Windows in subtle ways which specifically will, eventually, allow Windows users to implement many of the common idioms in Kubernetes using a comparable, but different runtime paradigm.
- The Kubernetes ServiceProxy implementation is not at parity with that of Linux, and likely never will be.  This needs to be made explicit so that attempts to reach parity between Kuberentes service API implementations on Windows and Linux can be approached in a systematic manner in terms of core requirements and supportability.  Features such as DSR (direct-server-return), EndpointSlices, and so on generally require special treatment for Windows clusters, some of which may have a complex interplay with the underlying CNI provider.

We propose the following definitions as the Windows conformance standard for Kubernetes clusters that are released after Kuebrentes 1.23 and onwards.  This specification is to be updated as new features are added, or removed, from the Kubernetes API itself.

### User Stories (Optional)

#### Story 1

As a Windows Application developer I want to verify that the features i rely on in Kubernetes are supported on my specific K8s Windows cluster.

#### Story 2

As a Kubenetes vendor I want to evalaute the completeness of my Kubernetes support matrix in context of Windows supported features in the Kubernetes API.

#### Story 3

As a Kubernetes developer I want to immediately know wether or not the features I want to add will effect the Windows support matrix, and if so, I want to be able to rapidly convey this in a canonical way to the broader community.

#### Story 4

As an IT administrator, I want to know wether my current version of Kubernetes supports ActiveDirectory

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

### Specification of Windows Core Conformance (Kubernetes 1.23)

- In order to qualify as a Windows test, the pods that demonstrate functionality for a given attribute of a cluster much be running on a `pause` image, and this image must be scheduled to an operating system that:

    - 1809
    - 1909
    - 2004
    - 2022

- All tests in the Core Conformance suite depend on `pause`, and no other images.
- All tests should use the `Pod.OS` specification field as the *only* required scheduling directive for pods, once this field is made GA.  Up until this time, the way tests schedule Windows pods can use RuntimeClasses, Labels, or taints in any way which "works" to implement the other requirements.

#### Basic Networking

- Ability to mount host volumes
- Ability to access linux container IPs *by service IP* from windows containers
- Ability to access windows container IPs *by service IP* from linux containers
- Ability to access linux container IPs *by NodePort IP* from windows containers
- Ability to access windows container IPs *by NodePort IP* from linux containers
- Ability to access linux container IPs *by pod IP* from windows containers
- Ability to access windows container IPs *by pod IP* from linux containers
- Ability to mount a hostPath storage volume from a Kubelet into a pod and write to it. 
- Ability to mount a hostPath storage volume from a Kubelet into a pod and read from it. 
- Ability to schedule multiple containers, with distinct IP addresses, on the same node.
- Ability to reboot all of the Windows nodes in a cluster while running windows pods, in such a way that recovery of these deleted pods occurs in the scope of a few minutes.
- Ability to delete and recreate services in such a way that load balancing rules for pods are recovered by whatever chosen service proxy is being utilized.
- Ability to delete and recreate pods for StatefulSets which preserve their ability to serve as routed endpoints for services, while also having unchanging IP addresses.
- Ability to access internal pods by internal Kubernetes DNS service endpoints.
- Ability to access external services by Kubernetes DNS services endpoints (for non-airgapped clusters).
- Ability to access internal pods through custom DNS records, injected by the Kubernetes pod Specification.
- Ability to route services from pods from the EndpointSlice API.

#### Basic Service accounts and local storage

- Ability to rotate service accounts for running pods, with the APIServer access capacity of the pod remaining unchanged over the long term.
- Ability to read and write shared files on a single Kubernetes node between three running Windows containers, simultaneously.

#### Basic Scheduling

- Ability to schedule pods with CPU and Memory limits demonstrably honored over time.
- Ability to demonstrate the pods requesting more CPU and Memory then available are left in the Pending state over time.

#### Basic performance

- Ability to reschedule the a deployment Pod, 50 times in a row, with continous deletions in the background.
- Ability to route traffic to 10 pods behind a common ClusterIP service endpoint.
- Ability to route traffic to 10 pods behind a common NodePort service endpoint.

We do not include GMSA support in the basic definition of WindowsConformance, because it is known to require other cluster components outside of Kubernete's Control.

### Sub-specifications of Windows Conformance (Kubernetes 1.23)

The following subsets of verifications for Windows clusters further expand the way we define supportability for Windows clusters, but they are not considered part of the "core" Conformance specification.  These features may be *heavily* reliant on functionality which resides well outside of Kubernete's perview, for example: CNI Implementation, Container Runtime implementation, or Windows Server Edition + Configuration.

#### Windows HostProcess Conformance

Note that, HostProcess Containers have not yet been verified to support all of these features.
Nevertheless, we define this specification because it is an obvious starting point for the long-term, meaningfull support of a full-fledged HostProcess feature in Kubernetes.

- Ability to access the APIServer using pod mounted service accounts from a hostProcess pod.
- Ability to create and manage host level networking (hcn) rules from a Windows hostProcess pod.
- Ability to launch hostProcess containers which share the IP address of a Windows node.
- Ability to launch hostProcess containers which can run other priveliged Windows system API (to be specified further in the future).
- Ability to utilize NT Authority/Network Service should be able to interact with host network objects

#### Active Directory

- Ability to run a pod as a GSMA User with only be able to access allowed resources
- Ability to read and write from local storage using ActiveDirectory credentials
- Observed lack of Ability to read and write from local storage when ActiveDirectory protected resources credentials are not present

#### Network Policies

Note that we do not include UDP, ipv6, in the initial definitions for NetworkPolicy conformance on Windows.  Since NetworkPolicy testing support on
Windows clusters was just recently added to Kuberentes Core, we err on the side of a conservative definition for Windows NetworkPolicy which
is sufficient to address the needs of *most* legacy network security applications.

- Ability to protect IPv4 pods from acceessing other pods when TCP NetworkPolicys are present that block specific pod connectivity.
- Ability to protect IPv4 pods from acceessing other pods when TCP NetworkPolicys are present that block specific namespace connectibity.

#### Windows Service Proxying

- Ability to support IPv6 and IPv4 services for accessing internal Linux pods from Windows pods over ClusterIP Endpoints.
- Ability to support IPv6 and IPv4 services for accessing internal Linux pods from Windows pods over NodePort Endpoints.
- Ability to support IPv6 and IPv4 services for accessing internal Windows pods from Linux pods over ClusterIP Endpoints.
- Ability to support IPv6 and IPv4 services for accessing internal Windows pods from Linux pods over NodePort Endpoints.

### Implementation

This section details the implementation of a "tool" which verifies these clusters must allow for the following, differentiated set of verifications, specified above.

- core conformance
- hostProcess
- activeDirectory
- networkPolicy
- serviceProxy

#### Option 1 - Test tags

One way to implement such a tool is to, specifically, target all available "ginkgo" tests in Kubernetes core itself, and run these one at a time (or concurrently).  Not we *do not* assert that these tests *must* run in parallel in order to comprise a conformance clusters (although this is often done in CI, its generally not a requirement of Kubernetes Conformance for linux clusters, either, since cluster capacity is not related to the functionality of a cluster). As an example, one such implementation of such a Conformance tool might be:

```
--ginkgo.focus=\\[Conformance\\]|\\[NodeConformance\\]|\\[sig-windows\\]|\\[sig-apps\\].CronJob --ginkgo.skip=\\[LinuxOnly\\]|\\[k8s.io\\].Pods.*should.cap.back-off.at.MaxContainerBackOff.\\[Slow\\]\\[NodeConformance\\]|\\[k8s.io\\].Pods.*should.have.their.auto-restart.back-off.timer.reset.on.image.update.\\[Slow\\]\\[NodeConformance\\]"
```

Currently, the defacto standard for windows testing resides at https://github.com/kubernetes-sigs/windows-testing, and we suggest this tag to users, however it should be noted that this does *not* conform to the specification in this document:

- it triggers some tests which uses non `pause` images for validating functionality.
- it doesn't implement the pod churn or emptyDir sharing tests.
- it doesn't implement the *reboot* tests which are defined in this specification.

The ability to specify HostProcess, NetworkPolicy, and other tests in this document would follow a similar tagging strategy, and any such tool could just invoke the existing Kuberentes end-to-end test binary with appropriate `focus` and `skip` arguments.

#### Option 2 - Sonobuoy implementation for convenience

Given the complexity of the above tagging scheme, we propose a concrete example of a binary program for Windows verification.

```
sonobuoy run --mode=windows-conformance --windows.hostProcess --windows.activeDirectory ...
```

In the above invocation, we verify

- core conformance
- hostProcess
- activeDirectory 

functionality for a specific Kubernetes release.  The resulting output would be obtained using

```
sonobuoy status
```

Which would report a binary result of `Pass` or `Fail`.  This tool could thus be used to
establish a rigorous standard for portable verificatoin of Kubernetes Windows based products across Vendors for both internal and external diagnostic purposes.

##### Implementation details for sonobuoy

So you'd have something like this:

```
sonobuoy plugin install windows-conformance <url>
sonobuoy run -p windows-conformance
```

The URL can point to any custom URL, but I have a few already made here: https://github.com/vmware-tanzu/sonobuoy-plugins/tree/master/windows-e2e

This is the same as just running from a URL or local file, e.g. `sonobuoy run -p windows-conformance.yaml` or `sonobuoy run -p pluginURL` but I was also showing that we are currently testing out plugin "installation" (caching?) which makes it easier to store, list, use plugins.

> Question: In this scenario (or option 2), Sonobuoy still has to know which tests to invoke. In the end it is still just invoking e2e.test so neither of this really avoid option 1, it does make it easier though.

### Test Plan

This KEP contemplate a full conformance suite for Windows environments already.

### Graduation Criteria

The plan is to introduct the feature as alpha in the v1.23 time frame having the
initial framework implementation.

#### Alpha -> Beta graduation

- The suite is graduated to beta when the core tests are implemented and running.

#### Beta->GA graduation

- The suite is graduated to GA when the sub-specifiction tests are implemented and running.

### Upgrade / Downgrade Strategy

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Other
  - Describe the mechanism: Based on tool option implemented a tag can be added or 
  - Will enabling / disabling the feature require downtime of the control
    plane? yes
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled). yes

###### Does enabling the feature change any default behavior?

No

###### Are there any tests for feature enablement/disablement?

None

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The CI must fix the tags or uninstall the Sonobuoy plugin in case of rollback.

###### What specific metrics should inform a rollback?

None

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

None

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

None

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

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

The system running the suite can be affected by cases where resources can be saturated without
proper Pod limits.

### Troubleshooting

## Alternatives

