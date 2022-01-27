# KEP-2845: Deprecate klog specific flags in Kubernetes Compnents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Removed klog flags](#removed-klog-flags)
  - [Logging defaults](#logging-defaults)
    - [Logging headers](#logging-headers)
  - [User Stories](#user-stories)
    - [Writing logs to files](#writing-logs-to-files)
  - [Caveats](#caveats)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Users don't want to use go-runner as replacement.](#users-dont-want-to-use-go-runner-as-replacement)
    - [Log processing in parent process causes performance problems](#log-processing-in-parent-process-causes-performance-problems)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Continue supporting all klog features](#continue-supporting-all-klog-features)
  - [Release klog 3.0 with removed features](#release-klog-30-with-removed-features)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
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

This KEP proposes to deprecate and in the future to remove a subset of the klog
command line flags from Kubernetes components, with goal of making logging of
k8s core components simpler, easier to maintain and extend by community.

## Motivation

Early on Kubernetes adopted glog logging library for logging. There was no
larger motivation for picking glog, as the Go ecosystem was in its infancy at
that time and there were no alternatives. As Kubernetes community needs grew
glog was not flexible enough, prompting creation of its fork klog. By forking we
inherited a lot of glog features that we never intended to support. Introduction
of alternative log formats like JSON created a conundrum, should we implement
all klog features for JSON? Most of them don't make sense and method for their
configuration leaves much to be desired. Klog features are controlled by set of
global flags that remain last bastion of global state in k/k repository. Those
flags don't have a single naming standard (some start with log prefix, some
not), don't comply to k8s flag naming (use underscore instead of hyphen) and
many other problems. We need to revisit how logging configuration is done in
klog, so it can work with alternative log formats and comply with current best
practices.

Lack of investment and growing number of klog features impacted project quality.
Klog has multiple problems, including:
* performance is much worse than alternatives, for example 7-8x than
  [JSON format](https://github.com/kubernetes/enhancements/tree/master/keps/sig-instrumentation/1602-structured-logging#logger-implementation-performance)
* doesn't support throughput to fulfill Kubernetes scalability requirements
  [kubernetes/kubernetes#90804](https://github.com/kubernetes/kubernetes/pull/90804)
* complexity and confusion caused by maintaining backward compatibility for
  legacy glog features and flags. For example
  [kuberrnetes/klog#54](https://github.com/kubernetes/klog/issues/54)

Fixing all those issues would require big investment into logging, but would not
solve the underlying problem of having to maintain a logging library. We have
already seen cases like [kubernetes/kubernetes#90804](https://github.com/kubernetes/kubernetes/pull/90804)
where it's easier to reimplement a klog feature in external project than fixing
the problem in klog. To conclude, we should drive to reduce maintenance cost and
improve quality by narrowing scope of logging library.

As for what configuration options should be standardized for all logging formats
I would look into 12 factor app standard (https://12factor.net/). It defines
logs as steams of events and discourages applications from taking on
responsibility for log file management, log rotation and any other processing
that can be done externally. This is something that Kubernetes already
encourages by collecting stdout and stderr logs and making them available via
kubectl logs. It's somewhat confusing that K8s components don't comply to K8s
best practices.

### Goals

* Unblock development of alternative logging formats
* Narrow scope of logging with more opinionated approach and smaller set of features
* Reduce complexity of logging configuration and follow standard component configuration mechanism.

### Non-Goals

* Change klog output format
* Remove flags from klog

## Proposal

We propose to remove klog specific feature flags in Kubernetes components
(including, but not limited to, kube-apiserver, kube-scheduler,
kube-controller-manager, kubelet) and leave them with defaults. From klog flags
we would remove all flags besides "-v" and "-vmodule". The component-base
"-log-flush-frequency" flag is also kept.

### Removed klog flags

To adopt 12 factor app standard for logging we would drop all flags that extend
logging over events streams. This change should be
scoped to only those components and not affect broader klog community.

Flags that should be deprecated:

* --log-dir, --log-file, --log-flush-frequency - responsible for writing to
  files and syncs to disk.
  Motivation: Not critical as there are easy to set up alternatives like:
  shell redirection, systemd service management or docker log driver. Removing
  them reduces complexity and allows development of non-text loggers like one
  writing to journal.
* --logtostderr, --alsologtostderr, --one-output, --stderrthreshold -
  responsible enabling/disabling writing to stderr (vs file).
  Motivation: Routing logs can be easily implemented by any log processors like:
  Fluentd, Fluentbit, Logstash.
* --log-file-max-size, --skip-log-headers - responsible configuration of file
  rotation.
  Motivation: Not needed if writing to files is removed.
* --add-dir-header, --skip-headers - klog format specific flags .
  Motivation: don't apply to other log formats
* --log-backtrace-at - A legacy glog feature.
  Motivation: No trace of anyone using this feature.

Flag deprecation should comply with standard k8s policy and require 3 releases before removal.

This leaves that two flags that should be implemented by all log formats

* -v - control global log verbosity of Info logs
* --vmodule - control log verbosity of Info logs on per file level

Those flags were chosen as they have effect of which logs are written,
directly impacting log volume and component performance. Flag `-v` will be
supported by all logging formats, however `-vmodule` will be optional for non
default "text" format.

### Logging defaults

With removal of configuration alternatives we need to make sure that defaults
make sense. List of logging features implemented by klog and proposed actions:
* Routing logs based on type/verbosity - Supported by alternative logging formats.
* Writing logs to file - Feature removed.
* Log file rotation based on file size - Feature removed.
* Configuration of log headers - Use the current defaults.
* Adding stacktrace - Feature removed.

Ability to route logs based on type/verbosity will be replaced with default
splitting info and errors logs to stdout and stderr. We will make this change
only in alternative logging formats (like JSON) as we don't want to introduce
breaking change in default configuration. Splitting stream will allow treating
info and errors with different priorities. It will unblock efforts like
[kubernetes/klog#209](https://github.com/kubernetes/klog/issues/209) to make
info logs non-blocking.

#### Logging headers

Default logging headers configuration results in klog writing information about
log type (error/info), timestamp when log was created and code line responsible
for generation it. All this information is useful and should be utilized by
modern logging solutions. Log type is useful for log filtering when looking for
an issue. Log generation timestamp is useful to preserve ordering of logs and
should be always preferred over time of injection which can be much later.
Source code location is important to identify how log line was generated.

Example:
```
I0605 22:03:07.224378 3228948 logger.go:59] "Log using InfoS" key="value"
```

### User Stories

#### Writing logs to files

We should use go-runner as a official fallback for users that want to retain
writing logs to files. go-runner runs as parent process to components binary
reading it's stdout/stderr and is able to route them to files. go-runner is
already released as part of official K8s images it should be as simple as changing:

```
/usr/local/bin/kube-apiserver --log-file=/var/log/kube-apiserver.log
```

to

```
/go-runner --log-file=/var/log/kube-apiserver.log /usr/local/bin/kube-apiserver
```

### Caveats

Is it ok for K8s components to drop support for subset of klog flags?

Technically K8s already doesn't support klog flags. Klog flags are renamed to
comply with K8s flag naming convention (underscores are replaced with hyphens).
Full klog support was never promised to users and removal of those flags should
be treated as removal of any other flag.

Is it ok for K8s components to drop support writing to files?
Writing directly to files is an important feature still used by users, but this
doesn't directly necessitates direct support in components. By providing a
external solution like go-runner we can allow community to develop more advanced
features while maintaining high quality implementation within components.
Having more extendable solution developed externally should be more beneficial
to community when compared to forcing closed list of features on everyone.

### Risks and Mitigations

#### Users don't want to use go-runner as replacement.

There are multiple alternatives that allow users to redirect logs to a file.
Exact solution depends on users preferred way to run the process with one shared
property, all of them supports consuming stdout/stderr. For example shell
redirection, systemd service management or
[docker logging driver](https://docs.docker.com/config/containers/logging/configure/).
Not all of them support log rotation, but it's users responsibility to know
complementary tooling that provides it. For example tools like
[logrotate](https://linux.die.net/man/8/logrotate).

#### Log processing in parent process causes performance problems

Passing logs through a parent process is a normal linux pattern used by
systemd-run, docker or containerd. For kubernetes we already use go-runner in
scalability testing to read apiserver logs and write them to file. Before we
reach Beta we should conduct detailed throughput testing of go-runner to
validate upper limit, but we don't expect any performance problem just based on
architecture.

## Design Details

### Test Plan

Go-runner is already used for scalability tests. We should ensure that we cover
all existing klog features.

### Graduation Criteria

#### Alpha

- The remaining supported klog options (`-v`, `--vmodule`) and
  `--log-flush-frequency` can be configured without registering flags.
- Kubernetes logging configuration is completely stored in one struct
  (`LoggingConfiguration`) before being applied to the process.
- Go-runner is feature complementary to klog flags planned for deprecation
- Projects in Kubernetes Org are migrated to go-runner
- JSON logs format splits stdout and stderr logs
- The klog flags which will be removed are marked as deprecated in command line
  help output and the deprecation is announced in the Kubernetes release notes.

#### Beta

- Go-runner project is well maintained and documented
- Documentation on migrating off klog flags is publicly available

#### GA

- Kubernetes klog specific flags are removed

## Implementation History

- 20/06/2021 - Original proposal created in https://github.com/kubernetes/kubernetes/issues/99270
- 30/07/2021 - KEP draft was created
- 26/08/2021 - Merged in provisional state
- 09/09/2021 - Merged as implementable
- Kubernetes 1.23 (tenatative): alpha, [deprecation
  period](https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-a-flag-or-cli)
  starts
- Kubernetes 1.24 (tentative): beta
- Kubernetes 1.26 (tentative): GA, deprecated flags get removed

## Drawbacks

Deprecating klog features outside klog might create confusion in community.
Large part of community doesn't know that klog was created from necessity and
is not the end goal for logging in Kubernetes. We should do due diligence to
let community know about our plans and their impact on external components
depending on klog.

## Alternatives

### Continue supporting all klog features
At some point we should migrate all logging
configuration to Options or Configuration. Doing so while supporting all klog
features makes their future removal much harder.

### Release klog 3.0 with removed features
Removal of those features cannot be done without whole k8s community instead of
just k8s core components

### Upgrade / Downgrade Strategy

For removal of klog specific flags we will be following K8s deprecation policy.
There will be 3 releases between informing users about deprecation and full removal.
During deprecation period there will not be any changes in behavior for clusters
using deprecated features, however after removal there will not be a way to
restore previous behavior. 3 releases should be enough heads up for users to
make necessary changes to avoid breakage.

### Version Skew Strategy

Proposed changes have no impact on cluster that would require coordination.
They only affect binary configuration and logs are written, which don't impact
other components in cluster. Users might be required to change flags passed to
k8s binaries, but this can be done one by one independently of other components.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [x] Other
  - Describe the mechanism: Passing command line flag to K8s component binaries.
  - Will enabling / disabling the feature require downtime of the control
    plane?
    **Yes, for apiserver it will require a restart, which can be considered a
    control plane downtime in non highly available clusters**
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
    **Yes, it will require restart of Kubelet**

###### Does enabling the feature change any default behavior?

No, we are not changing the default behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

After deprecation period, flags will be removed and users will not be able to re-enable them.
Only way to re-enable them would be to downgrade the cluster.

###### What happens if we reenable the feature if it was previously rolled back?

Flags cannot be reenabled without downgrading.

###### Are there any tests for feature enablement/disablement?

N/A, we are not introducing any new behavior.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

For removing klog flags, we don't have any escape hatch. Such breaking changes
will be properly announced, but users will need to make adjustments before
deprecation period finishes.

###### What specific metrics should inform a rollback?

Users could observe number of logs from K8s components that they ingest. If
there is a large drop in logs they get, whey should consider a rollback and
validate if their logging setup supports consuming binary stdout.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A, logging is stateless.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

Yes, as discussed above we will be removing klog flags.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?


###### How can someone using this feature know that it is working for their instance?

N/A

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

To detect if user logging system is consuming all logs generated by K8s
components it would be useful to have a metric to measure number of logs
generated. However, this is out of scope of this proposal, as topic of measuring
logging pipeline reliability heavily depends on third party logging systems that
are outside K8s scope.

### Dependencies

N/A

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

Scalability of logging pipeline is verified by existing scalability tests. We
don't plan to make any changes to existing tests.

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

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

Logs don't have a remote dependency on the API server or etcd.

###### What are other known failure modes?

No

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A
