# KEP-2845: Deprecate klog specific flags in Kubernetes Compnents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
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
command line flags from Kubernetes components to encourage more diverse 
approaches to logging in kubernetes ecosystem logging.

## Motivation

Early on Kubernetes adopted glog logging library for logging. Overtime the glog
was forked to klog and multiple improvements were implemented, but features put
into klog only piled up and were never removed. Introduction of alternative log
formats like JSON created a conundrum, should we implement all klog features for
JSON? Most of them don't make sense and method for their configuration leaves
much to be desired. Klog features are controlled by set of global flags that
remain last bastion of global state in k/k repository. Those flags don't have a
single naming standard (some start with log prefix, some not), don't comply to
k8s flag naming (use underscore instead of hyphen) and many other problems. We
need to revisit how logging configuration is done in klog so it can work with
alternative log formats and comply with current best practices.

Large number of features added to klog has lead to large drop in quality. 
[#90804](https://github.com/kubernetes/kubernetes/pull/90804) shows example 
where kubeup (canonical way to deploy kubernetes for testing) could not use 
klog feature to write log files due to scalability issues. The maintainers of 
klog decided it's easier to re-implementing a canonical klog feature in external
project than debugging the underlying problem.

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
* Improve quality of logging in k8s core components
* Logging should use configuration mechanism developed by WG component-standard

### Non-Goals

* Change klog output format

## Proposal

I propose that Kubernetes core components (kube-apiserver, kube-scheduler,
kube-controller-manager, kubelet) should drop flags that extend
logging over events streams or klog specific features. This change should be 
scoped to only those components and not affect broader klog community.

Flags that should be deprecated:

* --log-dir, --log-file, --log-flush-frequency - responsible for writing to 
  files and syncs to disk. 
  Motivation: Remove complexity to make alternative loggers easier to implement
  and reducing feature surface to improve quality.
* --logtostderr, --alsologtostderr, --one-output, --stderrthreshold - 
  responsible enabling/disabling writing to stderr (vs file). 
  Motivation: Not needed if writing to files is removed.
* --log-file-max-size, --skip-log-headers - responsible configuration of file 
  rotation. 
  Motivation: Not needed if writing to files is removed.
* --add-dir-header, --skip-headers - klog format specific flags . 
  Motivation: don't apply to other log formats
* --log-backtrace-at - an legacy glog feature. 
  Motivation: No trace of anyone using this feature.

Flag deprecation should comply with standard k8s policy and require 3 releases before removal.

This leaves that two flags that should be implemented by all log formats

* -v - control global log verbosity of Info logs
* --vmodule - control log verbosity of Info logs on per file level

Those flags were chosen as they have direct effect of which logs are written, 
directly impacting log volume and component performance.

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

TODO

#### Log processing in parent process causes performance problems

TODO

## Design Details

TODO

### Test Plan

Go-runner is already used for scalability tests. We should ensure that we cover
all existing klog features.

### Graduation Criteria

#### Alpha

- Klog can be configured without registering flags
- Kubernetes logging configuration drops global state
- Go-runner is feature complementary to klog flags planned for deprecation
- Projects in Kubernetes Org are migrated to go-runner

#### Beta

- Go-runner project is well maintained and documented 
- Documentation on switching go-runner is publicly available
- Kubernetes klog flags are marked as deprecated

#### GA

- Kubernetes klog flags are removed

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
