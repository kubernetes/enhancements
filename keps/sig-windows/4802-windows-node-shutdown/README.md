# KEP-4802: Graceful Node Shutdown for Windows Node

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
  - [Background on Windows Shutdown](#background-on-windows-shutdown)
  - [Implementation](#implementation)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha Graduation](#alpha-graduation)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

As specified in [kep2000](https://github.com/zylxjtu/enhancements/tree/master/keps/sig-node/2000-graceful-node-shutdown),
Kubelet should be aware of node shutdown and trigger graceful shutdown of pods
during a machine shutdown.

Node graceful shutdown has been available in Linux, this is to implement a comparable feature for Windows nodes

## Motivation

Users and cluster administrators expect that pods will adhere to expected pod
lifecycle including pod termination. Currently, when a node shuts down, pods do
not follow the expected pod termination lifecycle and are not terminated
gracefully which can cause issues for some workloads. This KEP aims to address
this problem by making the kubelet aware of the underlying Windows node shutdown.
Kubelet will initiate a graceful shutdown of pods, respecting lifecycle events such as 
pre-stop hooks, allowing pods to shut down as intended by pod authors

### Goals

*   Focus on handling shutdown on machines where kubelet itself runs as a Windows service, 
    on machines where kubelet is loaded through other loaders such as nssm.exe, this feature 
    will not be able to be enabled.
*   Make kubelet aware of underlying node shutdown event and trigger pod
    termination with sufficient grace period to shutdown properly
*   Handle node shutdown in cloud-provider agnostic way
*   Introduce minimal shutdown delay in order to shutdown node soon as possible
    (but not sooner)

### Non-Goals

*   Let users modify or change existing pod lifecycle or introduce new inner
    pod depencides / shutdown ordering
*   Support for this feature when the kubelet itself is not running as a Windows service (ex: loaded through nssm.exe), 
    under those scenarios, kublet will fail to start and log the corresponding error messages
*   Support Cancellation of Windows shutdown
*   Provide guarantee to handle all cases of graceful node shutdown, for
    example abrupt shutdown or sudden power cable pull can’t result in graceful
    shutdown

## Proposal


### User Stories (Optional)


#### Story 1

*   As a cluster administrator, I can configure the nodes in my cluster to
    allocate X seconds for my pods to terminate gracefully during a node
    shutdown

#### Story 2

*   As a developer I can expect that my pods will terminate gracefully during
    node shutdowns

### Background on Windows Shutdown

In the context of this KEP, shutdown is referred to as shutdown or restart of the
underlying Windows machine. A shutdown can be initiated via a
variety of methods for example:

1. `shutdown /t 0`
2. `shutdown /t 30` `#schedule a delayed shutdown in 30 seconds`
3. `shutdown /r /t 30` `#schedule a delayed restart in 30 seconds`
4. ACPI initiated shutdown/restart on the Windows machine
5. If a machine is a VM, the underlying hypervisor can press the “virtual”
   power button
6. For a cloud instance, stopping the instance via Cloud API, e.g. via `az vm stop -g xx -n xx`.
   Depending on the cloud provider, this may result in virtual power button press by the underlying hypervisor
7. Windows update (ex: Patch Tuesday) can also trigger a Windows node to be shutdown and rebooted

In the context of kubelet on Windows node, system is not
aware of the pods and containers running on the machine and will simply
kill them as regular Windows processes, that's the reason that we will need
to monitor the system for shutdown events and behave accordingly.

The high-level process of detecting a system shutdown on Windows is the same for all types of Windows applications and services. To detect a shutdown, 
we create a callback function and register it in the system. When a certain Windows event occurs, the system calls the callback function, transferring 
information about the event via input parameters. The callback function then analyzes the input parameter data, and executes code accordingly. In the 
case of this feature, the call back can detect the related (pre)shutdown event, and stop the pods gracefully before exiting

There’s no one way to make all types of applications (console/GUI and Windows services) detect a node shutdown, different types of applications differ 
in their syntax and functionality.

For general Windows console application, it can process the "CTRL_SHUTDOWN_EVENT", but according to [HandlerRoutine callback function](https://learn.microsoft.com/en-us/windows/console/handlerroutine#parameters)
and the [SetConsoleCtrlHandler function](https://learn.microsoft.com/windows/console/setconsolectrlhandler?redirectedfrom=MSDN) 
`Console functions, or any C run-time functions that call console functions, may not work reliably during processing of any of the three signals mentioned previously. 
The reason is that some or all of the internal console cleanup routines may have been called before executing the process signal handler`.

While running as a Windows service, programs will have more freedom to register a service control handler to monitoring various shutdown related events 
such as "SERVICE_CONTROL_PRESHUTDOWN", "SERVICE_CONTROL_SHUTDOWN". So in order to configure and process the (pre)shutdown event from Windows system, 
register the kubelet application as a Windows service program will be the most straightforward way.

Also from the same document [SetConsoleCtrlHandler function](https://learn.microsoft.com/windows/console/setconsolectrlhandler?redirectedfrom=MSDN),
It appears that "SERVICE_CONTROL_PRESHUTDOWN" is more flexible than "SERVICE_CONTROL_SHUTDOWN" for a Windows service to monitor and control

### Implementation
Similar with the linux case, "nodeshutdown_manager_windows.go" will be added to replace the current
fake/empty implementation for Windows, it will act as the main role in the node shutdown scenario and go through similar 
initialization/start process during kublet start up.

Also, we will update the Windows service package to "AcceptPreShutdown" and add a pre-shutdown handler to process
the "SERVICE_CONTROL_PRESHUTDOWN" event. This pre-shutdown handler will have an interface "ProcessShutdownEvent",
which will be implemented by the "nodeshutdown_manager_windows"

The entry point of running the kubelet as a Windows service is "InitService", in which, we can initialize and register 
the service control handler, the above "pre-shutdown handler" is part of this service control handler.

At this "InitService" stage, the kubelet has not read the kubelet configs/flags yet, so the "nodeshutdown_manager_windows", 
has not been instantiated, we will then set the "preshutdownhander" to be NIL, so will ignore the preshutdown event if any. 
This should be safe as the kubelet itself has not been started, let alone the pods it manages, so there is no 
need to do anything for shutting down at this stage.

When kubelet proceeds with starting up and initialize the "nodeshutdown_manager_window", it will set the service control handler’s 
pre-shutdownhandler to the "nodeshutdown_manager_window" and is now ready to monitor and process the preshutdown event

We will also add function of "UpdatePreShutdownInfo" and "QueryPreShutdownInfo" to Windows service package, these functions will be 
used by the "nodeshutdown_manager" to query and update the preshtudown timeout.

If we treat the 'pre-shutdown timeout value similar to the 'InhibitDelayMaxSec` in linux,
then the other parts of implementation will be pretty much similar with what linux has [implemented](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2000-graceful-node-shutdown#implementation)

The poc of the implementation can be found [here](https://github.com/zylxjtu/kubernetes/commit/854ea4bde88c0905241b43f5f80d470967bb909f#diff-8494875e9a1e884afd36f32fa90dceb2b827616f1e981ecb83a12901262214c7)

### Notes/Constraints/Caveats (Optional)

n/a

### Risks and Mitigations

Please refer to the the [part](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2000-graceful-node-shutdown#notesconstraintscaveats-optional) in KEP-2000

## Design Details

Use the same Kubelet Config as in linux.
```
type KubeletConfiguration struct {
    ...
    ShutdownGracePeriod metav1.Duration
    ShutdownGracePeriodCriticalPods metav1.Duration
}
```

Communication with service control manager will make use of golang.org/x/sys/windows/svc package,
which is already included in [vendor](https://github.com/kubernetes/kubernetes/tree/release-1.19/vendor/golang.org/x/sys/windows)


Termination of pods will make use of the existing
[killPod](https://github.com/kubernetes/kubernetes/blob/release-1.19/pkg/kubelet/pod_workers.go#L292) function
from the `kubelet` package and specify the appropriate `gracePeriodOverride` as
necessary.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates


##### Unit tests

The main logic of kill pod is the same as what linux did and there have been unit tests covered there.

For Windows specific logic such as update registry keys, setting Windows service control handler, 
It does not seem possible to cover with unit tests, so will cover with e2e tests.


##### Integration tests

There are efforts being worked on, but automated testing framework for Windows node level integration is not ready yet.
Until then, we will cover all the scenerios with e2e tests

##### e2e tests

*   New E2E tests to validate Windows node graceful shutdown.
    *   Shutdown grace period unspecified, feature is not active
    *   Pod’s ExecStop and SIGTERM handlers are given gracePeriodSeconds for
        case when gracePeriodSeconds <= kubeletConfig.ShutdownGracePeriod
    *   Pod’s ExecStop and SIGTERM handlers are given
        kubeletConfig.ShutdownGracePeriod for case when gracePeriodSeconds >
        kubeletConfig.ShutdownGracePeriod

### Graduation Criteria


#### Alpha Graduation

* Implemented the feature for Windows service only

* Investigate how e2e tests can be implemented (e.g. may need to create fake
  shutdown event)

#### Alpha -> Beta Graduation

* Sufficient E2E and unit testing
    *   Adding [Windows node level test](https://github.com/kubernetes/kubernetes/pull/129938) , which will include the gracefulshutdown case.
    *   [Enabling the test in CAPZ cluster](https://github.com/kubernetes-sigs/windows-testing/pull/506)

#### Beta -> GA Graduation

* Addressed feedback from beta
  * Beta feedback has been collected and incorporated
  * No major issues or API changes identified during beta period
* Sufficient number of users using the feature
  * Feature has been validated in production environments
  * Positive feedback from Windows node operators
* Confident that no further API / kubelet config configuration options changes are needed
  * Kubelet configuration options (`ShutdownGracePeriod`, `ShutdownGracePeriodCriticalPods`) are stable
* Close on any remaining open issues & bugs
  * All known bugs addressed
  * No blocking issues for GA graduation
* E2E tests are stable and flake-free for 2+ weeks
  * Tests enabled in CAPZ cluster via [PR #506](https://github.com/kubernetes-sigs/windows-testing/pull/506)
* User-facing documentation created for kubernetes.io

### Upgrade / Downgrade Strategy

Before GA graduation, feature gates will be used to control the enable/disable of this enhacement. During all phase, 
in order to enable this feature, the kubelet itself will need to run as a windows service.

### Version Skew Strategy

n/a

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

* **How can this feature be enabled / disabled in a live cluster?**

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `WindowsGracefulNodeShutdown`
  - Components depending on the feature gate: `kubelet`
  - **GA**: Feature gate is enabled by default. In 2-3 releases after GA, the feature gate may be deprecated and eventually removed.
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
    - No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
    - yes (will require restart of kubelet)

* **Does enabling the feature change any default behavior?**

  * The main behavior change is that during a node shutdown, pods running on the 
node will be terminated gracefully.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?**

  * Yes, the feature can be disabled by either disabling the feature gate, or
      setting `kubeletConfig.ShutdownGracePeriod` to 0 seconds.

* **What happens if we reenable the feature if it was previously rolled back?**

  * Kubelet will attempt to perform graceful termination of pods during a
    node shutdown.

* **Are there any tests for feature enablement/disablement?**

  * The e2e framework does not currently support enabling or disabling feature
  gates.  We have e2e tests to cover the feature when it is enabled and some predefined
  setting. Will add node level integration tests when the node level test framework is 
  available for Windows node

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

* **How can a rollout or rollback fail? Can it impact already running workloads?**

  * It wil not impact running workloads during rollout/rollback.

* **What specific metrics should inform a rollback?**

  * The failure of the roll out will behave like disbling this feature, operators can check the kubelet log to get more specific info.
ex: `The windows node graceful shutdown has not been enabled, the reasons are xxx`

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

  * The feature is part of kubelet config so updating kubelet config should enable/disable the feature; upgrade/downgrade is N/A

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?**

  * No

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

* **How can an operator determine if the feature is in use by workloads?**

  * Check if the feature gate and kubelet config settings are enabled on a node.

* **How can someone using this feature know that it is working for their instance?**

- [ ] Events
  - Event Reason: 
- [X] API .status
  - Condition name:  ContainersReady
  - Other field: 
- [X] Other (treat as last resort)
  - Details: Pod.Status.Message, Pod.Status.Reason

* **What are the reasonable SLOs (Service Level Objectives) for the enhancement?**

  * n/a

* **What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?**

<!--
Pick one more of these and delete the rest.
-->

- [x] Metrics
  - Metric name: GracefulShutdownStartTime, GracefulShutdownEndTime
  - [Optional] Aggregation method:
  - Components exposing the metric: Kubelet
- [x] Other (treat as last resort)
  - Details: The operator can get the service health information from the logs

* **Are there any missing metrics that would be useful to have to improve observability of this feature?**

  * n/a

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

* **Does this feature depend on any specific services running in the cluster?**

  * No, this feature doesn't depend on any specific services running the cluster.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

* **Will enabling / using this feature result in any new API calls?**

  * No

* **Will enabling / using this feature result in introducing new API types?**

  * No

* **Will enabling / using this feature result in any new calls to the cloud provider?**

  * No

* **Will enabling / using this feature result in increasing size or count of the existing API objects?**

  * No

* **Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?**

  * No

* **Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?**

  * No

* **Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?**

  * No

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

* **How does this feature react if the API server and/or etcd is unavailable?**

  * The feature does not depend on the API server / etcd.

* **What are other known failure modes?**

  - Kubelet does not detect the shutdown e.g. due to kubelet is not started as a Windows service. 
    - Detection: Kubelet logs
    - Mitigations: Workloads will not be affected, graceful node shutdown will not be enabled
    - Diagnostics: At default (v2) logging verbosity, kubelet will log if it is [running as a windows service](https://github.com/kubernetes/kubernetes/blob/b4e17418b340e161b8c6cc7f85a6e716abcb561a/pkg/windows/service/service.go#L130)
    - Testing: Working on adding SIG-Windows node level E2E tests check for graceful node shutdown including priority based shutdown

* **What steps should be taken if SLOs are not being met to determine the problem?**

  * n/a

## Implementation History

*   2024-08-31 - [Initial KEP approved](https://github.com/kubernetes/enhancements/pull/2001)
*   2026-02 - GA graduation in v1.36

## Drawbacks


## Alternatives

* No need to run kubelet itself as a Windows service but as a general Windows console application
    * General Windows application does not have much control on seting of the shutdown timeout value, and
      does not seem reliable to receive the shutdown event. A common practice of running kubelet on Windows
      node right now is through an external service control manager such as nssm. But no matter what the
      external service control manager will be, if the kubelet executable will not abel to get access to its
      service control hander, it will not be able to apply the corresponding logic to monitor and response to
      the corresponding shutdown event.
* Use RegKey WaitToKillServiceTimeout to control the shutdown time out value
    * As discussed in the Background part, Windows does not prefer to use this RegKey to update the shutdown
      timeout value, which will have a global affect on the services running on the host.


