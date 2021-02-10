# RunAsGroup support in PodSpec and PodSecurityPolicy

## Table of Contents

<!-- toc -->
- [Abstract](#abstract)
- [Motivation](#motivation)
  - [What is the significance of Primary Group Id?](#what-is-the-significance-of-primary-group-id)
- [Goals](#goals)
- [Use Cases](#use-cases)
  - [Use Case 1:](#use-case-1)
  - [Use Case 2:](#use-case-2)
- [Design](#design)
  - [Model](#model)
    - [SecurityContext](#securitycontext)
    - [PodSecurityContext](#podsecuritycontext)
    - [PodSecurityPolicy](#podsecuritypolicy)
- [Behavior](#behavior)
  - [Note About RunAsNonRoot field](#note-about-runasnonroot-field)
- [Summary of Changes needed](#summary-of-changes-needed)
- [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Abstract
As a Kubernetes User, we should be able to specify both user id and group id for the containers running 
inside a pod on a per Container basis, similar to how docker allows that using docker run options `-u`, 
```
-u, --user="" Username or UID (format: <name|uid>[:<group|gid>]) format
```

PodSecurityContext allows Kubernetes users to specify RunAsUser which can be overridden by RunAsUser
in SecurityContext on a per Container basis. There is no equivalent field for specifying the primary
Group of the running container.

## Motivation
Enterprise Kubernetes users want to run containers as non root. This means running containers with a 
non zero user id and non zero primary group id. This gives Enterprises, confidence that their customer code
is running with least privilege and if it escapes the container boundary, will still cause least harm
by decreasing the attack surface.

### What is the significance of Primary Group Id?
Primary Group Id is the group id used when creating files and directories. It is also the default group 
associated with a user, when he/she logins. All groups are defined in `/etc/group` file and are created
with the `groupadd` command. A Process/Container runs with uid/primary gid of the calling user. If no
primary group is specified for a user, 0(root) group is assumed. This means, any files/directories created
by a process running as user with no primary group associated with it, will be owned by group id 0(root).

## Goals

1. Provide the ability to specify the Primary Group id for a container inside a Pod
2. Bring launching of containers using Kubernetes at par with Dockers by supporting the same features.


## Use Cases

### Use Case 1:
As a Kubernetes User, I should be able to control both user id and primary group id of containers 
launched using Kubernetes at runtime, so that i can run the container as non root with least possible
privilege.

### Use Case 2:
As a Kubernetes User, I should be able to control both user id and primary group id of containers 
launched using Kubernetes at runtime, so that i can override the user id and primary group id specified
in the Dockerfile of the container image, without having to create a new Docker image.

## Design

### Model

Introduce a new API field in SecurityContext and PodSecurityContext called `RunAsGroup`.

#### SecurityContext

```
// SecurityContext holds security configuration that will be applied to a container.
// Some fields are present in both SecurityContext and PodSecurityContext.  When both
// are set, the values in SecurityContext take precedence.
type SecurityContext struct {
     //Other fields not shown for brevity
    ..... 

     // The UID to run the entrypoint of the container process.
     // Defaults to user specified in image metadata if unspecified.
     // May also be set in PodSecurityContext.  If set in both SecurityContext and
     // PodSecurityContext, the value specified in SecurityContext takes precedence.
     // +optional
     RunAsUser *int64
     // The GID to run the entrypoint of the container process.
     // Defaults to group specified in image metadata if unspecified.
     // May also be set in PodSecurityContext.  If set in both SecurityContext and
     // PodSecurityContext, the value specified in SecurityContext takes precedence.
     // +optional
     RunAsGroup *int64
     // Indicates that the container must run as a non-root user.
     // If true, the Kubelet will validate the image at runtime to ensure that it
     // does not run as UID 0 (root) and fail to start the container if it does.
     // If unset or false, no such validation will be performed.
     // May also be set in SecurityContext.  If set in both SecurityContext and
     // PodSecurityContext, the value specified in SecurityContext takes precedence.
     // +optional
     RunAsNonRoot *bool

    .....
 }
```

#### PodSecurityContext 

```
type PodSecurityContext struct {
     //Other fields not shown for brevity
    ..... 

     // The UID to run the entrypoint of the container process.
     // Defaults to user specified in image metadata if unspecified.
     // May also be set in SecurityContext.  If set in both SecurityContext and
     // PodSecurityContext, the value specified in SecurityContext takes precedence
     // for that container.
     // +optional
     RunAsUser *int64
     // The GID to run the entrypoint of the container process.
     // Defaults to group specified in image metadata if unspecified.
     // May also be set in PodSecurityContext.  If set in both SecurityContext and
     // PodSecurityContext, the value specified in SecurityContext takes precedence.
     // +optional
     RunAsGroup *int64
     // Indicates that the container must run as a non-root user.
     // If true, the Kubelet will validate the image at runtime to ensure that it
     // does not run as UID 0 (root) and fail to start the container if it does.
     // If unset or false, no such validation will be performed.
     // May also be set in SecurityContext.  If set in both SecurityContext and
     // PodSecurityContext, the value specified in SecurityContext takes precedence.
     // +optional
     RunAsNonRoot *bool

    .....
 }
```

#### PodSecurityPolicy

PodSecurityPolicy defines strategies or conditions that a pod must run with in order to be accepted
into the system. Two of the relevant strategies are RunAsUser and SupplementalGroups. We introduce 
a new strategy called RunAsGroup which will support the following options:
- MustRunAs
- RunAsAny

```
// PodSecurityPolicySpec defines the policy enforced.
 type PodSecurityPolicySpec struct {
     //Other fields not shown for brevity
    ..... 
  // RunAsUser is the strategy that will dictate the allowable RunAsUser values that may be set.
  RunAsUser RunAsUserStrategyOptions
  // SupplementalGroups is the strategy that will dictate what supplemental groups are used by the SecurityContext.
  SupplementalGroups SupplementalGroupsStrategyOptions


  // RunAsGroup is the strategy that will dictate the allowable RunAsGroup values that may be set.
  RunAsGroup RunAsGroupStrategyOptions
   .....
}

// RunAsGroupStrategyOptions defines the strategy type and any options used to create the strategy.
 type RunAsUserStrategyOptions struct {
     // Rule is the strategy that will dictate the allowable RunAsGroup values that may be set.
     Rule RunAsGroupStrategy
     // Ranges are the allowed ranges of gids that may be used.
     // +optional
     Ranges []GroupIDRange
 }

// RunAsGroupStrategy denotes strategy types for generating RunAsGroup values for a
 // SecurityContext.
 type RunAsGroupStrategy string
 
 const (
     // container must run as a particular gid.
     RunAsGroupStrategyMustRunAs RunAsGroupStrategy = "MustRunAs"
     // container may make requests for any gid.
     RunAsGroupStrategyRunAsAny RunAsGroupStrategy = "RunAsAny"
 )
```

## Behavior

Following points should be noted:

- `FSGroup` and `SupplementalGroups` will continue to have their old meanings and would be untouched.  
- The `RunAsGroup` In the SecurityContext will override the `RunAsGroup` in the PodSecurityContext.
- If both `RunAsUser` and `RunAsGroup` are NOT provided, the USER field in Dockerfile is used
- If both `RunAsUser` and `RunAsGroup` are specified, that is passed directly as User.
- If only one of `RunAsUser` or `RunAsGroup` is specified, the remaining value is decided by the Runtime,
  where the Runtime behavior is to make it run with uid or gid as 0.

Basically, we guarantee to set the values provided by user, and the runtime dictates the rest.

Here is an example of what gets passed to docker User
- runAsUser set to 9999, runAsGroup set to 9999 -> Config.User set to 9999:9999
- runAsUser set to 9999, runAsGroup unset -> Config.User set to 9999 -> docker runs you with 9999:0
- runAsUser unset, runAsGroup set to 9999 -> Config.User set to :9999 -> docker runs you with 0:9999 
- runAsUser unset, runAsGroup unset -> Config.User set to whatever is present in Dockerfile
This is to keep the behavior backward compatible and as expected.

### Note About RunAsNonRoot field

Note that this change does not introduce an equivalent field called runAsNonRootGroup in both SecurityContext
and PodSecurityContext. There was ongoing discussion about this field at PR [#62216](https://github.com/kubernetes/kubernetes/pull/62217)
The summary of this discussion seems as follows:-
- Use PSP MustRunAs Group strategy to guarantee that Pod never runs with 0 as Primary Group ID.
- Using the PSP MustRunAs Group strategy forces Pod to always specify a RunAsGroup
- RunAsGroup field when specified in PodSpec, will always override USER field in Dockerfile

There are other potentially unresolved discussions in that PR which need a followup.

## Summary of Changes needed
- https://github.com/kubernetes/kubernetes/pull/52077
- https://github.com/kubernetes/kubernetes/pull/67802
- https://github.com/kubernetes/kubernetes/pull/61030
- https://github.com/kubernetes/kubernetes/pull/72230
- https://github.com/kubernetes/kubernetes/pull/70465
- https://github.com/kubernetes/website/pull/12297
- https://github.com/kubernetes/kubernetes/pull/73007

## Test Plan
For `Alpha`, unit tests and e2e tests were added to test functionality at both
container and pod level for dockershim.

For `Beta`, tests were added to other CRI's like cri-o, containerd and Docker.

For `GA`, the introduced e2e tests will be promoted to conformance. It was also
verified that all e2e coverage was proper and CRI's had tests in their respective
repos testing this feature.

## Graduation Criteria

Beta
- RunAsGroup is tested for containerd and CRI-O in cri-tools repo using critest
  --  [Tests](https://github.com/kubernetes-sigs/cri-tools/blob/16911795a3c33833fa0ec83dac1ade3172f6989e/pkg/validate/security_context_linux.go#L357)
- critests are executed in cri-tools for all merges as GitHub Action
  --  [CRI-O](https://github.com/kubernetes-sigs/cri-tools/actions?query=workflow%3A%22critest+CRI-O%22)
  --  [containerd](https://github.com/kubernetes-sigs/cri-tools/actions?query=workflow%3A%22critest+containerd%22) 

GA
- assuming no negative user feedback, promote after 1 release at beta.
- verify test coverage for CRI's

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback
This feature is enabled in alpha releases using the feature flag `RunAsGroup`.


### Rollout, Upgrade and Rollback Planning


* **How can a rollout fail? Can it impact already running workloads?**
Its possible in an incorrect configuration. For e.g. lets say the init container writes some 
data using runAsGroup of 234, but the main container comes up as 436 and tries to read the 
data written by the initcontainer. If that fails, the pod will not be ready and the deployment
wont proceed. This should not impact already running workloads. One way, this can affect 
already running workloads is when data is shared between all pods and the access of the files
is changed by the initContainer due to misconfigured runAsGroup.


* **What specific metrics should inform a rollback?**
Metrics will be specific to application. Generic metrics like pod not being healthy and running
should generally inform rollback in this case. More specific checks will involve intrusive testing
like exec into a pod to determine the gid.


* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested? **
Yes, manually

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.? **
Moving from Beta to GA, is accompanied by the removal of the feature flag `RunAsGroup`. No other deprecations or removals 
are in scope or part of this process.

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
By inspecting the pod spec of any workload using kubectl or client-go libraries. If the pod spec
has RunAsGroup present either at the container or pod level, then the feature is in use.
```
kubectl get pods --all-namespaces  -o json | jq -r '.items[] | select(.spec.securityContext.runAsGroup != null or .spec.containers[].securityContext.runAsGroup != null)|[.metadata.name, .metadata.namespace]'
```

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
If a pod with this feature is enabled, and the pod is running , it's healthy.
If the pod doesn't have the expected runAsGroup id as determined by the below command,
the feature is not supported in that container runtime. Dont know if this caught earlier
somewhere.

```
id -g
```

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
N/A

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
N/A


### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
This feature only depends on the container runtime(CRI) supporting this feature.

### Scalability

* **Will enabling / using this feature result in any new API calls?**
  No
* **Will enabling / using this feature result in introducing new API types?**
  No

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
  No

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
This feature adds two new fields on at the pod level and one in each and every container this field is used in.


* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  No

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No


### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**
After a pod is deployed, this feature will continue to work even if etcd or api server is unavailable.
The functions not available when apiserver or etcd is unavailable is not specific to this feature.


* **What are other known failure modes?**
N/A

* **What steps should be taken if SLOs are not being met to determine the problem?**
  N/A




## Implementation History
- Proposal merged on 9-18-2017
- Implementation merged as Alpha on 3-1-2018 and Release in 1.10
- Implementation for Containerd merged on 3-30-2018 
- Implementation for CRI-O merged on 6-8-2018
- Implemented RunAsGroup PodSecurityPolicy Strategy on 10-12-2018
- Beta in 1.14
- GA in 1.21
