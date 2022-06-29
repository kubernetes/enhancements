# KEP-2578: Windows Operational Readiness Specification

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Background](#background)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
    - [Story 4](#story-4)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Specification of Windows Operational Readiness (Kubernetes 1.24)](#specification-of-windows-operational-readiness-kubernetes-124)
    - [Basic Networking](#basic-networking)
    - [Basic Service accounts and local storage](#basic-service-accounts-and-local-storage)
    - [Basic Scheduling](#basic-scheduling)
    - [Basic concurrent functionality](#basic-concurrent-functionality)
  - [Sub-specifications of Windows Operational Readiness (Kubernetes 1.24)](#sub-specifications-of-windows-operational-readiness-kubernetes-124)
    - [Windows HostProcess Operational Readiness](#windows-hostprocess-operational-readiness)
    - [Active Directory](#active-directory)
    - [Network Policies](#network-policies)
    - [Windows Advanced Networking and Service Proxying](#windows-advanced-networking-and-service-proxying)
    - [Windows Worker Configuration](#windows-worker-configuration)
  - [Implementation](#implementation)
    - [Option 1 - Test tags](#option-1---test-tags)
    - [Option 2 - Sonobuoy implementation for convenience](#option-2---sonobuoy-implementation-for-convenience)
      - [Implementation details for Sonobuoy](#implementation-details-for-sonobuoy)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA graduation](#beta---ga-graduation)
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

As Kubernetes' e2e suite has expanded over the years, the 1000s of existing Ginkgo tests don't easily lend themselves to a meaningful interpretation of this operational readiness required for business-critical Windows applications. Thus, end-users of Windows clusters have a less stable experience due to differences in the installation and management of CNI, CRI, or other implementation and configuration details, compared with the more standardized Linux Kubernetes implementation.

To ameliorate this, we define an *operational readiness standard for Kubernetes clusters supporting Windows* that certifies the readiness of Windows clusters for running production workloads which:
- informs the creation and maintenance of existing e2e tests 
- informs the behavior and outputs of Windows conformance for K8s standardization tools such as Sonobuoy
- informs reviewers outside of the *sig-windows* community on the overall goals of the sig-windows tests inside of Kubernetes

In short, we assert that, to be "operationally ready": 

- Clusters capable of running Windows containers must be capable of passing Linux Conformance tests
- Clusters capable of running Windows containers must also pass the "core" Windows conformance tests defined in this document
- Clusters capable of running Windows containers, which support a specific feature (such as NetworkPolicies) must pass the tests for that feature defined in this KEP.
- Clusters capable of running Windows containers, to be operationally ready, must have 2 schedulable Linux nodes and 2 schedulable Windows nodes, to test properly verify pod to pod networking over services on different nodes.

One obvious ask here may be: Why not just define a set of Ginkgo tags? Ginkgo tags, which are commonly used to define test suites, can be subject to change over time (either in their implementation, their definition, or both).  Although they likely will be the canonical implementation detail of *how* this specification is verified in the real-world, we do not want to couple the definition of Windows "operational readiness" also aim to implement these definitions in a declarative, fully automated certification bundle via the Sonobuoy (https://sonobuoy.io/) tool, as a quick and easy way for end-users to verify Windows clusters.


This KEP: 
- abstractly defines the various categories of Windows "operational readiness", differentiating between core conformance from feature sets that are of common interest
- defines how these conformance tests can be run using *existing* Kubernetes ginkgo tags, without any additional tooling 
- prescribes a canonical implementation for a generic "Windows operational readiness" testing tool, which can be either hand-rolled (i.e. in a bash script) or, implemented as part of a broader initiative (i.e. inside of the sonobuoy.io, which is commonly used for Kubernetes verification)
- will be updated to reflect changes to Windows APIs and functionality over time (i.e. some features such as `HostProcess` aren't treated in this KEP as critical "operational readiness" concerns for the current release of Kubernetes, which is 1.22).

## Motivation

Running Windows containers (in small environments, or at scale) is a requirement for many mature enterprises migrating applications Kubernetes.  The requirements of a Windows Kubelet are comparable to a Linux Kubelet, however, because Windows does not support the same storage, security, and privileged container model that Linux does.  We thus need an unambiguous standard in place that is meaningful and actionable for comparing Windows Kubernetes support between vendors.

### Goals

- Define the verification requirements for "operational readiness" testing in Kubernetes from the implementation of tests themselves.
- Define how to test any given cluster for these requirements using ginkgo tags and the existing e2e tests as our first implementation for this verification.
- Prescribe a workflow for building automated Windows certification tools which leverage the above tests for a enterprise-ready, standardized mechanism for rapid and unambiguous Windows certification.

### Non-Goals

- Modify the existing Kubernetes Windows operational readiness specification, which has Linux specific tests.
- Modify existing tests inside of Kubernetes that would increase the bifurcation of testing implementation.
- Adding new tooling to Kubernetes core itself for testing Windows.

## Proposal

### Background

If we consider the various idiosyncrasies of verifying Kubernetes on Windows we are immediately confronted with several potential problem areas that can be difficult to wade through as an end user.

- The ability to run Linux containers on Windows (LCOW) is a potential feature which may conflate the definition of a "Windows Kubelet" significantly and could potentially result in false-positive test results where multi-arch images are able to pass a Conformance suite without actually running any Windows workloads. This specific issue can be solved by defining a specification and method of implementing operationally readiness conformance for Windows.
- There is currently a `LinuxOnly` tag in the Kubernetes e2e testing suite. This tag needs to be disambiguated along with this KEP defining explicit functional requirements for Windows nodes. This enables the tag to used as an implementation detail for how Windows clusters are verified.
- Some tests (including Networking, NetworkPolicy, and Storage related tests) behave differently on Windows because of differences in the feature sets and testing idioms that are supportable on Windows Server. We aim to distinguish the core requirements of such tests in terms of conformant Windows clusters regardless of existing implementation gaps which may exist between how Linux and Windows tests are implemented, as can be seen in the tests as they are currently defined  https://github.com/kubernetes/kubernetes/blob/master/test/e2e/network/netpol/network_policy.go#L1256. 
- Some critical Windows functionality doesn't have an analog in the Linux world, such as the ability to run activeDirectory as an identity provider for a pod, needs to be well-defined in a Kubernetes context for individuals planning on integrating production Windows workloads into their Kubernetes support model.
- The introduction of "Privileged containers" (formally referred to as `HostProcess` pods) further differentiates Linux from Windows in subtle ways which will allow Windows users to implement many of the common idioms in Kubernetes using a comparable runtime paradigm with slight differences.
- The Kubernetes "kube-proxy" implementation is not at parity with that of Linux and is potentially going to lag behind for the foreseeable future. For example, some features (such as EndpointSlices) aren't implemented in the Windows userspace proxy.  Yet other features such as DSR and terminating endpoints have specific Windows behaviors which aren't identical to that of the Linux service proxy https://github.com/kubernetes/kubernetes/issues/96514.
- The way users run pods on Windows, as a specific user, varies from the implementation on Linux, as illustrated here: https://github.com/openshift/windows-machine-config-operator/pull/638. 

We propose the following definitions as the Windows Operational Readiness for Kubernetes clusters that are released after Kubernetes 1.24 and onwards. This specification is to be updated as new features are added or removed from the Kubernetes API itself.

### User Stories (Optional)

#### Story 1

As a Windows Application developer, I want to verify that the features I rely on in Kubernetes are supported on my specific Windows Kubernetes cluster.

#### Story 2

As a Kubernetes vendor, I want to evaluate the completeness of my Kubernetes support matrix in the context of Windows supported features in the Kubernetes API.

#### Story 3

As a Kubernetes developer, I want to immediately discern whether the features I want to add will affect the Windows support matrix. If there are any confirmed or potential impact(s) I want to be able to rapidly convey them in a canonical way to the broader community.

#### Story 4

As an IT administrator, I want to know whether my current version of Kubernetes supports ActiveDirectory or other Windows features.

### Risks and Mitigations

## Design Details

### Specification of Windows Operational Readiness (Kubernetes 1.24)
- In order to qualify as a Windows test, the pods that demonstrate functionality for a given attribute of a cluster must be running on a `agnhost` image. 
- Additionally, this image must be scheduled to an operating system that meets the following requirements:
- Windows Build Versions:
  - Windows Server 2019 (LTSC)
  - Windows Server 2022 (LTSC)
- To avoid adding accidental complexity to the definition of operational readiness, we restrict the types of workload images used for tests like so:
  - All tests in the Core Windows Operational Readiness suite depend on `pause` as well as `agnhost`, and no other images (when it comes to windows workloads).
  - For non windows workloads that are transitive dependencies of windows workloads (i.e. a windows container's ability to access a linux busybox based pod), the above rule doesn't apply.
  - In cases where specific images are required for specific sorts of windows workloads that *explicitly* require non-agnhost programs (i.e. a GPU related application needing a certain Java version) exceptions are allowed, but these would be justified by specific workload requirements.
- If Pod.OS becomes the standard for scheduling in an OS specific manner, all tests should use the `Pod.OS` specification field as the *only* required scheduling directive for pods, once this field is made GA.  Up until this time, the way tests schedule Windows pods can use RuntimeClasses, Labels, or taints in any way which "works" to implement the other requirements.
- until this field is fully available in all tests, the use of `tolerations` and `labels` should be used for all tests to solidify the scheduling aspects of Windows containers. 
 
#### Basic Networking

- Ability to access Windows container IP *by pod IP*.
- Ability to expose windows pods *by creating the service ClusterIP*.
- Ability to expose windows pods *by creating the service NodePort*.
- Ability to schedule multiple containers, with distinct IP addresses, on the same node.
- Ability to delete and recreate services in such a way that load balancing rules for pods are recovered by whatever chosen service proxy is being utilized.
- Ability to delete and recreate pods for StatefulSets which preserve their ability to serve as routed endpoints for services.
- Ability to access internal pods by internal Kubernetes DNS service endpoints.
- Ability to access external services by Kubernetes DNS services endpoints (for non-airgapped clusters).
- Ability to access internal pods through custom DNS records, injected by the Kubernetes pod Specification.
- Ability to route services from pods from the EndpointSlice API.

#### Basic Service accounts and local storage

- Ability to reboot all of the Windows nodes in a cluster while running Windows pods, in such a way that recovery of these deleted pods occurs in the scope of a few minutes.
- Ability to delete and recreate services in such a way that load balancing rules for pods are recovered by whatever chosen service proxy is being utilized.
- Ability to mount a hostPath storage volume from a Kubelet into a pod and write to it. 
- Ability to mount a hostPath storage volume from a Kubelet into a pod and read from it. 
- Ability to mount host volumes
- Ability to rotate service accounts for running pods, with the APIServer access capacity of the pod remaining unchanged over the long term.
- Ability to read and write shared files on a single Kubernetes node between three running Windows containers, simultaneously.

#### Basic Scheduling

- Ability to schedule pods with CPU and Memory limits demonstrably honored over time.
- Ability to demonstrate the pods requesting more CPU and Memory then available are left in the Pending state over time.

#### Basic concurrent functionality

We intentionally do not define scale targets, as this KEP concerns itself with standardizing minimal functionality required for an operational Windows cluster.

That said, some level of concurrency and repetition is required to verify that a cluster can be functional, multi-tenant workloads.

Thus, we define the following simplistic "performance" tests.  Since Windows service load balancing is not as deeply hardened as that of Linux, we also propose that basic scalability requirements which limit node sizes for Windows clusters, be put in place, to ensure that "large clusters" are validated in a correct way.

- Ability to reschedule a deployment Pod, 50 times in a row, with continuous deletions in the background.
- Ability to route traffic to 10 pods behind a common ClusterIP service endpoint.
- Ability to route traffic to 10 pods behind a common NodePort service endpoint.

We do not include GMSA support in the basic definition of Windows OR because it is known to require other cluster components outside of Kubernetes's Control.

### Sub-specifications of Windows Operational Readiness (Kubernetes 1.24)

The following subsets of verification for Windows clusters further expand the way we define supportability for Windows clusters; however, they are not considered part of the "core" operational readiness specification. These features may be *heavily* reliant on functionality which resides well outside of Kubernetes's purview, for example: CNI Implementation, Container Runtime implementation, or Windows Server Edition and Configuration.

#### Windows HostProcess Operational Readiness

Note that, HostProcess Containers have not yet been verified to support all these features, so this is not "core" to the definition of operational readiness, but someday, we expect that it *will* be.
Nevertheless, we define this specification because it is an obvious starting point for the long-term, meaningful support of a full-fledged HostProcess feature in Kubernetes.

- Ability to access the APIServer using pod mounted service accounts from a hostProcess pod.
- Ability to create and manage host level networking (HCN) rules from a Windows hostProcess pod.
- Ability to launch hostProcess containers which share the IP address of a Windows node.
- Ability to launch hostProcess containers which can run other privileged Windows system API (to be specified further in the future).
- Ability for pods to bind to host network interfaces on windows (requires hostProcess pods for scheduling the pod itself).
- Ability for nodes to continue participating in a K8s cluster after rebooting (requires hostProcess pods for testing).

#### Active Directory

- Ability to run a pod as a GMSA User with only be able to access allowed resources
- Ability to read and write from local and remote storage using ActiveDirectory credentials
- Observed lack of Ability to read and write from local and remote storage when ActiveDirectory protected resources credentials are not present
- The behavior of the `RunAsUserName` field for Windows pods should be that it is supported (i.e. pods can use this), but that there are no guarantees around volume permissions and access when using this field.

#### Network Policies

Note that we do not include UDP, ipv6, in the initial definitions for NetworkPolicy conformance on Windows. Since NetworkPolicy testing support on Windows clusters was just recently added to Kubernetes core, we err on the side of a conservative definition for Windows NetworkPolicy which is sufficient to address the needs of *most* legacy network security applications.

- Ability to protect IPv4 pods from accessing other pods when TCP NetworkPolicies are present that block specific pod connectivity.
- Ability to protect IPv4 pods from accessing other pods when TCP NetworkPolicies are present that block specific namespace connectability.

#### Windows Advanced Networking and Service Proxying

These apply to Windows Server 2019 and up (we expect windows 2004 to be out of support once this KEP merges).

- Ability to support IPv6 and IPv4 services for accessing internal Linux pods from Windows pods over ClusterIP Endpoints.
- Ability to support IPv6 and IPv4 services for accessing internal Linux pods from Windows pods over NodePort Endpoints.
- Ability to support IPv6 and IPv4 services for accessing internal Windows pods from Linux pods over ClusterIP Endpoints.
- Ability to support IPv6 and IPv4 services for accessing internal Windows pods from Linux pods over NodePort Endpoints.
- Ability to run IPv4/IPv6 dual-stack networking (on supported OS Versions).

Note that, ipv4 and ipv6 are both known to be highly CNI (overlay vs non-overlay) and OS (2019 vs 2022) dependent.

#### Windows Worker Configuration

Workers node settings that must be compliant with basic operational readiness standards.  This set of checks
may or may not be automated, but provide administrators a way to escalate usability bugs related to their windows
user experience.  

These are not meant to be gating features in any test suite when it comes to defining Windows Operational Readiness for
the broader community.

We list these as an initial pass for the provisional KEP, as they are valuable as a guideline for vendors supporting Kubernetes on Windows.
- Ability to assess node's Microsoft Defender exclusion set for required processes (containerd).
- Ability for nodes to ping other nodes in the same cluster network
- Ability for nodes to access TCP and UDP services in the same cluster network through NodePorts on their resident kube proxy's.
- Ability for administrators to SSH or RDP into nodes to remotely run commands, mount storage assets, or restart Kubernetes servies.

### Implementation

This section details the implementation of a "tool" which verifies these clusters must allow for the following, differentiated set of verifications, specified above.

- Core Conformance
- HostProcess
- ActiveDirectory
- NetworkPolicy
- ServiceProxy

As an example, if a Windows verification run for 1.22 was occurring, the highest semver close to 1.22, could be used
for running NetworkPolicy verification.  In this case, 1.21.  This would allow for backward compatibility of this file
so that if a ginkgo tag changed, a new "test" with a new "version" and the SAME operationalReadinessDescription could be added.

As part of this KEP, we will finish the ongoing ginkgo tag curation at https://docs.google.com/spreadsheets/d/1Pz7-AUZ9uxMBwFx7ZC2U6dBfYpMUdOUi1UivBRVDw7I/edit#gid=0 - to live in the a `sig-windows` repository as an reference summarization and a convinient grouping of the tests.
This KEP implements a layer of abstraction mainly in a YAML/JSON, which defines each item described below containing
characteristics like filtering tags, kubernetes version, and a more generic description, this document can be parsed and processed,
by plugins or runtime dashboards, follows the proposed schema:

```
kubernetesVersions:
- "1.21"
- "1.22"

tests:
- ginkgoTag: 
  - NetworkPolicy
  ginkgoSkip:
  - UDP
  - SCTP
  operationalReadinessDescription: "Ability to protect IPv4 pods from acceessing other pods when TCP NetworkPolicys are present that block specific pod connectivity."
  # optional additional fields
  windowsImage: "agnhost:1.2.x"
  linuxImage: bool
  hasLinuxPods: bool
  hasWindowsPodssPods: "yes"
  kubernetesVersion: "1.21"
- ginkgoTag: 
  - HostProcess
  ginkgoSkip:
  - LinuxOnly
  operationalReadinessDescription: "Ability to access the APIServer using pod mounted service accounts from a hostProcess pod.
```

#### Option 1 - Test tags

One way to implement such a tool is to, specifically, target all available "ginkgo" tests in Kubernetes core itself, and run these one at a time (or concurrently).  Not we *do not* assert that these tests *must* run in parallel in order to comprise a conformance cluster (although this is often done in CI, its generally not a requirement of Kubernetes Conformance for linux clusters, either, since cluster capacity is not related to the functionality of a cluster). As an example, one such implementation of such a Conformance tool might be:

```
--ginkgo.focus=\\[Conformance\\]|\\[NodeConformance\\]|\\[sig-windows\\]|\\[sig-apps\\].CronJob --ginkgo.skip=\\[LinuxOnly\\]|\\[k8s.io\\].Pods.*should.cap.back-off.at.MaxContainerBackOff.\\\[NodeConformance\\]|\\[k8s.io\\].Pods.*should.have.their.auto-restart.back-off.timer.reset.on.image.update.\\[Slow\\]\\[NodeConformance\\]"
```

Currently, the de facto standard for Windows testing resides at https://github.com/kubernetes-sigs/windows-testing, and we suggest this tag to users, however it should be noted that this does *not* conform to the specification in this document:

- it triggers some tests which uses non `pause` images for validating functionality.
- it doesn't implement the pod churn or `emptyDir` sharing tests.
- it doesn't implement the *reboot* tests which are defined in this specification.

The ability to specify HostProcess, NetworkPolicy, and other tests in this document would follow a similar tagging strategy, and any such tool could just invoke the existing Kubernetes end-to-end test binary with appropriate `focus` and `skip` arguments.

#### Option 2 - Sonobuoy implementation for convenience

Given the complexity of the above tagging scheme, we propose a concrete example of a binary program for Windows verification. 
This was the option chosen for this KEP implementation.

```
sonobuoy run --mode=windows-conformance --windows.hostProcess --windows.activeDirectory ...
# which prints out the exact tests so that the actual test coverage is clear
```

In the above invocation, we verify

- Core Conformance
- HostProcess
- ActiveDirectory 

functionality for a specific Kubernetes release. The resulting output would be obtained using

```
sonobuoy status
```

Which would report a binary result of `Pass` or `Fail`. This tool could thus be used to establish a rigorous standard for portable verification of Kubernetes Windows based products across Vendors for both internal and external diagnostic purposes.


##### Implementation details for Sonobuoy

Below is an example of running the Windows Operational Readiness conformance tests with Sonobuoy

```
sonobuoy plugin install windows-operational-readiness <url>
sonobuoy run -p windows-operational-readiness
```

The URL can point to any custom URL. We have published a few examples of Windows e2e tests here: https://github.com/vmware-tanzu/sonobuoy-plugins/tree/master/windows-e2e.  The canonical version of these will be determined by sig-windows once the tooling convergence has evolved to the point that this KEP is ready for general consumption.  See the requirements for GA section for more details on this in the future.

The process will be the same whether running from a URL or local file, e.g. `sonobuoy run -p windows-op-readiness.yaml` or `sonobuoy run -p pluginURL`. We are also testing plugin installation which would make it easier to store, list, and use plugins.

> Question: In this scenario (or option 2), Sonobuoy still has to know which tests to invoke. In the end it is still just invoking e2e.test so neither of this really avoid option 1, it does make it easier though.

### Test Plan

This KEP defines the test plan for such a cluster, but in general, we will update all of our CI jobs to implement it.

### Graduation Criteria

The plan is to introduce the feature as alpha in the v1.24 time frame having the
initial framework implementation.

At this time, we'll expect that users can either:

1) Leverage a simple process to run specific tests validating a feature, such as...
```
	curl coreconformance.1.23.yaml
	skiptags=`cat coreconformance1.23.yaml` | yq .basicServices.reboot_all_nodes | grep skipTags
	skiptags=`cat coreconformance1.23.yaml` | yq .basicServices.reboot_all_nodes | grep focusTags
	e2e.test -skip="$skiptags" -focus="$focustags"
```

2) Run `sonobuoy` command line tooling, which consumes the above raw daya, to do the same via
command line flags:

```
	
```

#### Alpha -> Beta graduation

- The suite is graduated to beta when the core tests are implemented and running.

#### Beta -> GA graduation

- The suite is graduated to GA when the sub-specification tests are implemented and running.

### Upgrade / Downgrade Strategy

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

Not applicable because this KEP is focused on building Windows focused test suite.

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

