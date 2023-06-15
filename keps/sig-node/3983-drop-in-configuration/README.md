# KEP-3983: Add support for a drop-in kubelet configuration directory

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Add support for a drop-in configuration directory for the Kubelet. This directory can be specified via a "--config-dir" flag, and configuration files will be processed in alphanumeric order. The flag will be empty by default and if not specified, drop-in support will not be enabled. Establishment of conventions for configuration processing will be done, and further work can be done to add this support for other components.


## Motivation

A common pattern for software configuration in linux is support for a drop-in configuration directory. The location of this directory is often based on a corresponding configuration file. For instance, `/etc/security/limits` can be overridden by files in `/etc/security/limits.d`. This pattern is useful for a number of reasons, though a large motivation here is to allow files to be owned by a single owner. If multiple processes are vying for changing the same file, then they could stamp over each other's changes and possibly race against each other, creating TOCTOU problems.

Components in Kubernetes can similarly be configured by multiple entities and preventing races between them is cumbersome. There has been past work in the Kubelet to have a Dynamic Configuration, but resolving between multiple entities and a last known good state was also complicated. Since the Kubelet is the node agent, and is often distributed as a package on the host operating system along with the container runtime, configuring it similarly to other host processes seems clear. This paves the path for continuing the pattern of drop-in configuration for the Kubelet.


### Goals

* Add support for a "--config-dir" flag to the kubelet to allow users to specify a drop-in directory, which will override the configuration for the Kubelet located at `/etc/kubernetes/kubelet.conf`
* Extend kubelet configuration parsing code to handle files in the drop-in directory.
* Define Kubernetes best-practices for configuration definitions, similarly to [API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md). This is intended for other maintainers who would wish to setup a configuration object that works well with drop-in directories.
* As a goal for beta, Add ability to easily view the effective configuration that is being used by kubelet.

### Non-Goals

* Add support for drop-in configuration for Kubernetes components other than the Kubelet.
* Dynamically reconfiguring running kubelets if drop-in contents change.

## Proposal

This proposal aims to add support for a drop-in configuration directory for the kubelet via specifying a "--config-dir" flag (for example, `/etc/kubernetes/kubelet.conf.d`). Users are able to specify individually configurable kubelet config snippets in files, formatted in the same way as the existing kubelet.conf file. The kubelet will process the configuration provided in the drop-in directory in alphanumeric order:


1. If no other configuration for the subfield(s) exist, append to the base configuration
2. If the subfield(s) exists in the base configuration at `/etc/kubernetes/kubelet.conf` file or another file in the drop-in directory with lesser alphanumeric ordering, overwrite it

    * If the subfield(s) exist as a list, overwrite instead of attempting to merge. This makes it easier to delete items from lists defined in the base kubelet.conf or other drop-ins without having to modify other files. See example below


If there are any issues with the drop-ins (e.g. formatting errors), the error will be reported in the same way as a misconfigured kubelet.conf file. Only files with a `.conf` extension will be parsed. All other files found will be skipped and logged.

This drop-in directory is purely optional and if empty, the base configuration is used and no behavior changes will be introduced. The "--config-dir" flag will be empty by default and if not specified, drop-in support will not be enabled. This aims to align with "--config" flag defaults.

Example:

Base configuration:
```
authentication:
  anonymous:
    enabled: false
  webhook:
    enabled: true
  x509:
    clientCAFile: /etc/kubernetes/pki/ca.crt
clusterDNS:
 - 1.2.3.4
 - 1.2.3.5
```

Drop-in 1:
```
authentication:
  x509
    clientCAFile: /some/new/location
```

Drop-in 2:
```
clusterDNS:
 - 1.2.3.6
```

Final result:
```
authentication:
  anonymous:
    enabled: false
  webhook:
    enabled: true
  x509:
    clientCAFile: some/new/location
clusterDNS:
 - 1.2.3.6
```

### User Stories


#### Story 1

As a cluster admin, I would like to be able to easily customize the Kubelet configuration for different node types, while still sharing a base configuration. For instance, I would like to have customized system reserved allocations for the control plane and workers.


#### Story 2

As a Kubernetes distribution author, I would like to enable users to customize fields on the Kubelet while leaving a sensible and secure default in an easy way.


#### Story 3

As a cluster admin, I would like to have cgroup management and log size management in different files, so I can automate per-node management of those configurations performed via different components without cross-interference.


### Risks and Mitigations

* Handling of zeroed fields
    * It’s possible the configuration of the Kubelet does not handle not specified fields well. Special testing will need to be done for different types to define and ensure conformance of that behavior.
* Handling of lists
    * Contention could be found with how lists should be handled (append or overwrite, also see proposal). Consensus should be found and testing performed. 

## Design Details


### Test Plan

[X] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.


##### Prerequisite testing updates


##### Unit tests

* Unit tests will be added, details to be added here

##### Integration tests

* :

##### e2e tests

* :

### Graduation Criteria

#### Alpha

Add ability to support drop-in configuration directory.

#### Beta

Add ability to easily view the effective configuration that is being used by kubelet. Details to be added during beta phase.

### Upgrade / Downgrade Strategy

Upgrades and downgrades are safe as far as Kubelet stability is concerned. It’s possible a vendor may ship vital pieces of configuration within a drop-in directory. If the Kubelet downgrades to a version that doesn’t support reading the drop-in directory, the kubelet will not recognize the "--config-dir" flag and risk failing. However, assuming that vendor left that the original `/etc/kubelet/kubelet.conf` is in a valid state, and the flag isn't specified, there should be no risk to the system. Any configuration that exists in a drop-in dir won't be applied, but that would not affect kubelet stability.


### Version Skew Strategy

All behavior change is encapsulated within the Kubelet, so there is no version skew possible within core Kubernetes. It is possible third party tools may attempt to utilize the Kubelet’s drop-in directory before the Kubelet is upgraded to support it, which would cause silent failures. It is the responsibility of these third party tools to ensure the Kubelet is new enough to support this.


## Production Readiness Review Questionnaire


### Feature Enablement and Rollback


###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: KubeletDropInConfig
  - Components depending on the feature gate: Kubelet

Aside from the featuregate, admins will also have to explicitly enable the kubelet flag to enable this.


###### Does enabling the feature change any default behavior?

No, upgrading to a version of the Kubelet with this feature will not enable the Kubelet to be configured with the drop-in directory if no flag is specified.


###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Roll back and remove the `--config-dir` flag from the kubelet's CLI, as well as the KubeletDropInConfig featuregate.


###### What happens if we reenable the feature if it was previously rolled back?

This feature will be re-enabled via adding back the `--config-dir` flag to the CLI, as well as the KubeletDropInConfig featuregate, as mentioned above.


###### Are there any tests for feature enablement/disablement?

A test will be added to assemble a single, functional kubelet configuration object from various individual drop-in config files.

A test will be added to check that if a user attempts to use the `--config-dir` flag without the featuregate, the kubelet will properly error.


### Rollout, Upgrade and Rollback Planning


###### How can a rollout or rollback fail? Can it impact already running workloads?

The feature can cause the Kubelet to fail if the configuration in the drop-in directory is invalid. A rollback could fail if the original configuration also has an invalid configuration. This situation would cause workloads to not appear on that node. Neither of these cases are expected.


###### What specific metrics should inform a rollback?

The Kubelet not starting, which will cause nodes to be NotReady.


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

The feature does not persist any data and so the upgrade->downgrade->upgrade path is not special.


###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No


### Monitoring Requirements


###### How can an operator determine if the feature is in use by workloads?

Workloads do not directly consume this feature, it is for cluster admins during kubelet configuration. Administrators can check the kubelet feature flag metric `kubernetes_feature_enabled` to see if this is enabled.


###### How can someone using this feature know that it is working for their instance?

In alpha, the user can query their active kubeletconfiguration to see if their drop-ins have taken effect.
In beta and onwards, the user will be able to read this off logs or the API, to be determined as described above.


###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?


###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?


###### Are there any missing metrics that would be useful to have to improve observability of this feature?


### Dependencies


###### Does this feature depend on any specific services running in the cluster?

No


### Scalability


###### Will enabling / using this feature result in any new API calls?

No


###### Will enabling / using this feature result in introducing new API types?

No, though there may be changes to the Kubelet configuration required.


###### Will enabling / using this feature result in any new calls to the cloud provider?

No


###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No, though metadata on the fields may need to be changed.


###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

It will take slightly longer for the Kubelet to start, but it should not be noticeable unless there are very many (hundreds?) of configurations.


###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Likely negligible amounts of CPU.


###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Not likely


### Troubleshooting


###### How does this feature react if the API server and/or etcd is unavailable?

This feature is enabled in Kubelet alone.


###### What are other known failure modes?

Invalid configuration, same as exists today with `/etc/kubernetes/kubelet.conf`


###### What steps should be taken if SLOs are not being met to determine the problem?

Fix the invalid configuration, or remove configurations.


## Implementation History

- 2023-05-04: KEP initialized.


## Drawbacks


## Alternatives

Reinstate the now deprecated Dynamic Kubelet Configuration

Continue to rely on CLI flags or systemd drop-in files.
