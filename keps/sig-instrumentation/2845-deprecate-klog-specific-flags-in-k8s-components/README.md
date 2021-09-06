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
    - [Split stdout and stderr](#split-stdout-and-stderr)
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
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Continue supporting all klog features](#continue-supporting-all-klog-features)
  - [Release klog 3.0 with removed features](#release-klog-30-with-removed-features)
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
* Removing flags from klog itself

## Proposal

I propose to remove klog specific feature flags in Kubernetes core components
(kube-apiserver, kube-scheduler, kube-controller-manager, kubelet) and set them
to agreed good defaults. From klog flags we would remove all flags besides "-v"
and "-vmodule". With removal of flags to route logs based on type we want to
change the default routing that will work as better default. Changing the
defaults will be done in via multi release process, that will introduce some
temporary flags that will be removed at the same time as other klog flags.

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
directly impacting log volume and component performance.

### Logging defaults

With removal of configuration alternatives we need to make sure that defaults
make sense. List of logging features implemented by klog and proposed actions:
* Routing logs based on type/verbosity - Should be reconsidered.
* Writing logs to file - Feature removed.
* Log file rotation based on file size - Feature removed.
* Configuration of log headers - Use the current defaults.
* Adding backtrace - Feature removed.

For log routing I propose to adopt UNIX convention of writing info logs to
stdout and errors to stderr. For log headers I propose to use the current
default.

#### Split stdout and stderr

As logs should be treated as event streams I would propose that we separate two
main streams "info" and "error" based on log method called. As error logs should
usually be treated with higher priority, having two streams prevents single
pipeline from being clogged down (for example
[kubernetes/klog#209](https://github.com/kubernetes/klog/issues/209)).
For logging formats writing to standard streams, we should follow UNIX standard
of mapping "info" logs to stdout and "error" logs to stderr.

Splitting stdout from stderr would be a breaking change in both klog and
kubernetes components. However, we expect only minimal impact on users, as
redirecting both streams is a common practice. In rare cases that will be
impacted, adapting to this change should be a 1 line change. Still we will want
to give users a proper heads up before making this change, so we will hide the
change behind a new logging flag `--logtostdout`. This flag will be used avoid
introducing breaking change in klog.

With this flag we can follow multi release plan to minimize user impact (each
point should be done in a separate Kubernetes release):
1. Introduce the flag in disabled state and start using it in tests.
1. Announce flag availability and encourage users to adopt it.
1. Enable the flag by default and deprecate it (allows users to flip back to previous behavior)
1. Remove the flag following the deprecation policy.

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

- Klog can be configured without registering flags
- Kubernetes logging configuration drops global state
- Go-runner is feature complementary to klog flags planned for deprecation
- Projects in Kubernetes Org are migrated to go-runner
- Add --logtostdout flag to klog disabled by default
- Use --logtostdout in kubernetes tests

#### Beta

- Go-runner project is well maintained and documented
- Documentation on migrating off klog flags is publicly available
- Kubernetes klog flags are marked as deprecated
- Enable --logtostdout in Kubernetes components by default

#### GA

- Kubernetes klog specific flags are removed (including --logtostdout)

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Implementation History

- 20/06/2021 - Original proposal created in https://github.com/kubernetes/kubernetes/issues/99270
- 30/07/2021 - First KEP draft was created

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
