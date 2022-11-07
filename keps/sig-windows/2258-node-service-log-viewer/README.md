# KEP-2258: Node service log viewer

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implement client for logs endpoint viewer (OS agnostic)](#implement-client-for-logs-endpoint-viewer-os-agnostic)
  - [Linux distros with systemd / journald](#linux-distros-with-systemd--journald)
  - [Linux distributions without systemd / journald](#linux-distributions-without-systemd--journald)
  - [Windows](#windows)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Large log files and events](#large-log-files-and-events)
    - [Wider access to all node level service logs](#wider-access-to-all-node-level-service-logs)
- [Design Details](#design-details)
    - [kubelet](#kubelet)
    - [kubectl](#kubectl)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

A Kubernetes cluster administrator has to log in to the relavant control-plane
or worker nodes to view the logs of the API server, kubelet etc. Or they would
have to implement a client side reader. A simpler and more elegant method would
be to allow them to use the kubectl CLI to also view these logs similar to
using it for other interactions with the cluster. Given the sensitive nature of
the information in node logs, this feature will only be available to cluster
administrators.

## Motivation

Troubleshooting issues with control-plane and worker nodes typically requires
a cluster administrator to SSH into the nodes for debugging. While certain
issues will require being on the node, issues with the kube-proxy or kubelet,
to name a couple, could be solved by perusing their logs. However this
too requires the administrator to SSH access into the nodes. Having a way for
them to view the logs using kubectl will significantly simplify their
troubleshooting.


### Goals
Provide a cluster administrator with a streaming view of logs using kubectl
without them having to implement a client side reader or logging into the node.
This would work for:
- Services on Linux worker and control plane nodes:
  - That have systemd / journald support.
  - That have services that log to `/var/log/`
- Windows worker nodes (all supported variants) that log to `C:\var\log`,
  System and Application logs, Windows Event Logs and Event Tracing (ETW).

### Non-Goals
- Providing support for non-systemd Linux distributions.
- Reporting logs for nodes that have config or connection issues with the
  cluster.
- Getting logs from services that do not use /var/log/.

## Proposal

### Implement client for logs endpoint viewer (OS agnostic)
- Implement a new `kubectl node-logs` to work with node objects.
- Implement a client for the `/var/log/` kubelet endpoint viewer.

### Linux distros with systemd / journald
Supplement the the `/var/log/` endpoint viewer on the kubelet with a thin shim
over the `journal` directory that shells out to journalctl. Then implement
`kubectl node-logs` to also work with node objects.

### Linux distributions without systemd / journald
Running the new "kubectl node-logs" command against services on nodes that do
not use systemd / journald should return "OS not supported". However getting
logs from `/var/log/` should work on all systems.

### Windows
Reuse the kubelet API for querying the Linux journal for invoking the
`Get-WinEvent` cmdlet in a PowerShell.

### User Stories

Consider a scenario where pods / containers are refusing to come up on certain
nodes. As mentioned in the motivation section, troubleshooting this scenario
involves the cluster administrator to SSH into nodes to scan the logs. Allowing
them to use `kubectl node-logs` to do the same as they would to debug issues with a
pod / container would greatly simply their debug workflow. This also opens up
opportunities for tooling and simplifying automated log gathering. The feature
can also be used to debug issues with Kubernetes services especially in Windows
nodes that run as native Windows services and not as DaemonSets or Deployments.

Here are some example of how a cluster administrator would use this feature:
```
# Show kubelet and crio journal logs from all masters
kubectl node-logs --role master -q kubelet -q crio

# Show kubelet log file (/var/log/kubelet/kubelet.log) from all Windows worker nodes
kubectl node-logs --label kubernetes.io/os=windows -q kubelet

# Display docker runtime WinEvent log entries from a specific Windows worker node
kubectl node-logs <node-name> --query docker
```

### Risks and Mitigations

#### Large log files and events
If the log that is attempted to be viewed is very large (GBs) there is
potential for the node performance to be degraded. To mitigate this we can
document that node logs should always be rotated in clusters that enable this
feature. We should also take into account nodes that don't take advantage of
journald's rate limiting options. We can then take real world feedback around
this for better mitigation when graduating the feature from alpha to beta.

#### Wider access to all node level service logs
The cluster administrator can now view all logs in /var/log/, systemd/journald
services and Windows services. Given that the cluster administrator can log
into the nodes and view the same information this should not be an issue.
However there is potential for scenarios where the cluster administrator does
not have access to the infrastructure. This again would benefit from real world
usage feedback.

## Design Details

#### kubelet

The kubelet already has a `/var/log/` [endpoint viewer](https://github.com/kubernetes/kubernetes/blob/b184272e278571d1e6650605dd4c39be897eaaa2/pkg/kubelet/kubelet.go#L1403)
that is lacking a client. Given its existence we can supplement that with a
wafer thin shim over the /journal directory that shells out to journalctl. This
allows us to extend the endpoint for getting logs from the system journal on
Linux systems that support systemd. To enable filtering of logs, we can reuse
the existing filters supported by journalctl. The `kubectl node-logs` will have
command line options for specifying these filters when interacting with node
objects.

On the Windows side viewing of logs from services that use `C:\var\log` will
be supported by the existing endpoint. For Windows services that log to the
the System and Application logs, Windows Event Logs and Event Tracing (ETW),
we can leverage the [Get-WinEvent cmdlet](https://docs.microsoft.com/en-us/powershell/module/microsoft.powershell.diagnostics/get-winevent?view=powershell-7.1)
that supports getting logs from all these sources. The cmdlet has filtering
options that can be leveraged to filter the logs in the same manner we do
with the journal logs.

Please note that filtering will not be available for logs in `/var/log/` or
`C:\var\log\`.

The feature now enables the cluster administrator to interrogate all services.
This could be prevented by having a whitelist of allowed services. But this
comes with severe disadvantages as there could be nodes (especially with
Windows) that have other services to support networking and monitoring.
These services are variable and will depend on how the nodes have been
configured. Here are some examples:
- [hybrid-overlay-node](https://github.com/ovn-org/ovn-kubernetes/tree/master/go-controller/hybrid-overlay)
- [windows-exporter](https://github.com/prometheus-community/windows_exporter).


The `/var/log/` endpoint is enabled using the `enableSystemLogHandler` kubelet
configuration options. To gain access to this new feature this option needs to
be enabled. In addition when introducing this feature it will be hidden behind a
`NodeLogViewer` feature gate in the kubelet that needs to be explicitly enabled. So
you need to enable both options to get access to this new feature and disabling
`enableSystemLogHandler` will disable the new feature irrespective of the
`NodeLogViewer` feature gate.

A reference implementation of this feature is available
[here](https://github.com/kubernetes/kubernetes/pull/96120).

#### kubectl

`kubectl` has an existing `logs` command that is used to view the logs for a
container in a pod or a specified resource. The sub-command looks at resource
types, so can be extended to work with node objects to view the logs of services
on the nodes. Given that the `logs` command depends on RBAC policies for access
to appropriate resource type and associated endpoints, it will allow us to
restrict node logs access to only cluster administrators as long as the cluster
is setup in that manner. Access to the `node/logs` sub-resource needs to be
explicitly granted as a user with access to `nodes` will not automatically have
access to `node/logs`. In the alpha phase the functionality will be behind
`kubectl alpha node-logs` sub-command. The functionality will be moved to
`kubectl node-logs` in the beta phase. However the examples will reference the
final destination i.e. `kubectl node-logs`.

The `logs --query` sub-command for node objects will follow a heuristics approach when
asked to query for logs from a Windows or Linux service. If asked to get the
logs from a service `foobar`, it will first assume `foobar` logs to the Linux
journal / Windows eventing mechanisms (Application, System, and ETW). If unable
to get logs from these, it will attempt to get logs from `/var/log/foobar.log`,
`/var/log/foobar/foobar.log`, `/var/log/foobar*INFO` or
`/var/log/foobar/foobar*INFO` in that order. Alternatively an explicit file
location can be passed to the `--query` option.
Here are some examples and explanation of the options that will be added.
```
Examples:
  # Show kubelet logs from all masters
  kubectl node-logs --role master -q kubelet

  # Show docker logs from Windows nodes
  kubectl node-logs -l kubernetes.io/os=windows -q docker

  # Show foo.log from Windows nodes
  kubectl node-logs -l kubernetes.io/os=windows -q /foo/foo.log

Options:
  -g, --grep='': Filter log entries by the provided regex pattern. Only applies to node journal logs.
  -o, --output='': Display journal logs in an alternate format (short, cat, json, short-unix). Only applies to node journal logs.
      --raw=false: Perform no transformation of the returned data.
      --role='': Set a label selector by node role.
  -l, --selector='': Selector (label query) to filter on.
      --since-time='': Return logs after a specific ISO timestamp.
      --tail=-1: Return up to this many lines (not more than 100k) from the end of the log.
      --sort=timestamp: Interleave logs by sorting the output. Defaults on when viewing node journal logs.
  -q, --query=[]: Return log entries that matches any of the specified service(s).
      --until-time='': Return logs before a specific ISO timestamp.
```

The `--sort=timestamp` feature will introduce log unification across node
objects by timestamps which can be extended to pod logs. This will allow users
to see logs across nodes from the same time. Similarly for pods, it will allow
seeing logs across containers aligned by time.

Given that the feature will be introduced behind a feature gate, by default
`kubectl node-logs` will return a functionality not available message. When the
feature is enabled in alpha phase, `kubectl node-logs` will display a
warning message that the feature is in alpha. When the `--query` option
is used against Linux nodes that do not support systemd/journald and the service
does not log to `/var/log`, the same functionality not available message will be
returned.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

Add unit tests to kubelet and kubectl that exercise the new arguments that
have been added. A reference implementation of the tests can be seen
[here](https://github.com/kubernetes/kubernetes/pull/96120/commits/253dbad91a3896680da74da32595f02120f56cfa#diff-1d703a87c6d6156adf2d0785ec0174bb365855d4883f5758c05fda1fee8f7f1b)

Given that a new kubelet package is introduced as part of this feature there is
no existing test coverage to link to.

##### Integration tests

Given that we need the kubelet running locally to test this feature, integration
tests will not be possible for this feature.

##### e2e tests

We will add a test that query the kubelet service logs on Windows and Linux nodes.
On Windows node, the same kubelet service logs will queried by explicitly
specifying the log file. In Linux the explicit log file query will be tested by
querying a random file in present in /var/log.

On the Linux side tests will be added to [kubelet node](https://github.com/kubernetes/kubernetes/blob/master/test/e2e/node/kubelet.go)
e2e tests. For Windows a new set of tests will be added to the existing
[e2e tests](https://github.com/kubernetes/kubernetes/tree/master/test/e2e/windows).

- node: https://storage.googleapis.com/k8s-triage/index.html?sig=node
- windows: https://storage.googleapis.com/k8s-triage/index.html?sig=windows

### Graduation Criteria

The plan is to introduce the feature as alpha in the v1.26 time frame behind the
`NodeLogViewer` kubelet feature gate and using the `kubectl alpha node-logs`
sub-command.

#### Alpha -> Beta Graduation

The plan is to graduate the feature to beta in the v1.27 time frame. At that
point we would have collected feedback from cluster administrators and
developers who have enabled the feature. Based on this feedback and issues
opened we should consider adding a kubelet side throttle for the viewing the
logs. In addition we will garner feedback on the heuristic approach and based on
that we will decide if we need introduce options to explicitly differentiate
between file vs journal / WinEvent logs.

The kubectl implementation will move from `kubectl alpha node-logs` to 
`kubectl node-logs`.
#### Beta -> GA Graduation

The plan is to graduate the feature to GA in the v1.28 time frame at which point
any major issues should have been surfaced and addressed during the alpha and
beta phases.

### Upgrade / Downgrade Strategy

### Version Skew Strategy

If a kubectl version that has the new `node-logs` option is used against a node
that is using a kubelet that does not have the extended `/var/log` endpoint
viewer, the result should be "feature not supported".

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate
    - Feature gate name: NodeLogViewer
    - Components depending on the feature gate: kubelet

* **Does enabling the feature change any default behavior?** No

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?** Yes. It can be disabled by disabling the `NodeLogViewer` feature
  gate in the kubelet.

* **What happens if we reenable the feature if it was previously rolled back?**
  There will be no adverse effects of enabling the feature gate after it was
  disabled.

* **Are there any tests for feature enablement/disablement?** No

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  - kubelet
    - Usage description:
      - Impact of its outage on the feature: If kubelet is not running on the
        node this feature will not work.
      - Impact of its degraded performance or high-error rates on the feature:
        If the kubelet is degraded this feature will also be degraded i.e. the
        node logs will not be returned.

### Scalability

* **Will enabling / using this feature result in any new API calls?**
  No

* **Will enabling / using this feature result in introducing new API types?**
  Yes. We will need to add a `NodeLogOptions` counterpart to
  [PodLogOptions](https://github.com/kubernetes/kubernetes/blob/548ad1b8d35d51e6d33ea21dcc75d60a789b00e6/pkg/apis/core/types.go#L4409)

* **Will enabling / using this feature result in any new calls to the cloud
provider?**
  No

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  No

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  No

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  In the case of large logs, there is potential for an increase in RAM and CPU
  usage on the node when an attempt is made to stream them. Feedback from the
  field during alpha will provide more clarity as we graduate from alpha to
  beta.

### Troubleshooting

## Implementation History

- Created on Jan 14, 2021
- Updated on May 5th, 2021

## Drawbacks

## Alternatives

Alternatively we could use a client side reader on the nodes to redirect the
logs. The Windows side would require privileged container support. However this
would not help scenarios where containers are not launching successfully on the
nodes.

For the kubectl changes an alternative to introducing `kubectl node-logs` would be to
introduce a plugin.
