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
    - [GA](#ga)
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

Add support for a drop-in configuration directory for the Kubelet. This directory can be specified via a `--config-dir` flag, and configuration files will be processed in alphanumeric order. The flag will be empty by default and if not specified, drop-in support will not be enabled. During the alpha phase, we introduced an environment variable called `KUBELET_CONFIG_DROPIN_DIR_ALPHA` to control the drop-in configuration directory for testing purposes. In the beta phase, we plan to leave the `--config-dir` flag unset by default, which aligns with the behavior of the `--config` flag. Users are encouraged to opt in by specifying their desired configuration directory. Additionally, we will enhance the feature with E2E testing and streamline the configuration process. As part of this optimization, we will remove the `KUBELET_CONFIG_DROPIN_DIR_ALPHA` environment variable, simplifying configuration management. The feature will be enabled by default during the beta phase, and we will evaluate its status in future releases.


## Motivation

A common pattern for software configuration in linux is support for a drop-in configuration directory. The location of this directory is often based on a corresponding configuration file. For instance, `/etc/security/limits` can be overridden by files in `/etc/security/limits.d`. This pattern is useful for a number of reasons, though a large motivation here is to allow files to be owned by a single owner. If multiple processes are vying for changing the same file, then they could stamp over each other's changes and possibly race against each other, creating TOCTOU problems.

Components in Kubernetes can similarly be configured by multiple entities and preventing races between them is cumbersome. There has been past work in the Kubelet to have a Dynamic Configuration, but resolving between multiple entities and a last known good state was also complicated. Since the Kubelet is the node agent, and is often distributed as a package on the host operating system along with the container runtime, configuring it similarly to other host processes seems clear. This paves the path for continuing the pattern of drop-in configuration for the Kubelet.


### Goals

* Add support for a `--config-dir` flag to the kubelet to allow users to specify a drop-in directory, which will override the configuration for the Kubelet located at `/etc/kubernetes/kubelet.conf`
* Extend kubelet configuration parsing code to handle files in the drop-in directory.
* Define Kubernetes best-practices for configuration definitions, similarly to [API conventions](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md). This is intended for other maintainers who would wish to setup a configuration object that works well with drop-in directories.
* Add ability to easily view the effective configuration that is being used by kubelet.

### Non-Goals

* Add support for drop-in configuration for Kubernetes components other than the Kubelet.
* Dynamically reconfiguring running kubelets if drop-in contents change.

## Proposal

This proposal aims to add support for a drop-in configuration directory for the kubelet via specifying a `--config-dir` flag (for example, `/etc/kubernetes/kubelet.conf.d`). Users are able to specify individually configurable kubelet config snippets in files, formatted in the same way as the existing kubelet.conf file. The kubelet will process the configuration provided in the drop-in directory in alphanumeric order:


1. If no other configuration for the subfield(s) exist, append to the base configuration
2. If the subfield(s) exists in the base configuration at `/etc/kubernetes/kubelet.conf` file or another file in the drop-in directory with lesser alphanumeric ordering, overwrite it

    * If the subfield(s) exist as a list, overwrite instead of attempting to merge. This makes it easier to delete items from lists defined in the base kubelet.conf or other drop-ins without having to modify other files. See example below


If there are any issues with the drop-ins (e.g. formatting errors), the error will be reported in the same way as a misconfigured kubelet.conf file. Only files with a `.conf` extension will be parsed. All other files found will be skipped and logged.

This drop-in directory is purely optional and if empty, the base configuration is used and no behavior changes will be introduced. The `--config-dir` flag, along with the `KUBELET_CONFIG_DROPIN_DIR_ALPHA` environment variable, allows users to specify a drop-in configuration directory for the Kubelet. This directory is empty by default, ensuring that drop-in support is not enabled unless explicitly configured. This aims to align with `--config` flag defaults.

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
    * During the beta phase, we will conduct additional testing to address risks and refine the feature.

## Design Details


### Test Plan

[X] I/we understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.


##### Prerequisite testing updates


##### Unit tests

* cmd/kubelet/app: 07-17-2023  27.6

##### Integration tests

* N/A

##### e2e tests

* A [test](https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/kubelet_config_dir_test.go) should confirm that the kubelet.conf.d directory is correctly processed, and its contents are accurately reported in the configz endpoint.

### Graduation Criteria

#### Alpha

Add ability to support drop-in configuration directory.

#### Beta

Add ability to augment the feature's capabilities with a focus on robustness and testing, which includes:
  - Ensure the correct kubelet configuration is displayed when queried using the `kubectl get --raw "/api/v1/nodes/{node-name}/proxy/configz"` command, particularly verifying the contents of the kubelet.conf.d directory.
  - Remove the environment variable `KUBELET_CONFIG_DROPIN_DIR_ALPHA`, introduced during the Alpha phase, to streamline the user experience by simplifying configuration management.
  - Leave the `--config-dir` flag empty by default. Users can configure it by specifying a path, with `/etc/kubernetes/kubelet.conf.d` as the recommended directory.
  - Add a version compatibility check for drop-in files to ensure alignment with the expected Kubelet configuration API version and catch discrepancies when future versions are introduced.
  - Provide official guidance on the Kubernetes website for merging lists and structures in the kubelet configuration file, including documentation for the `/configz` endpoint.

#### GA

Collect user feedback and gather information about real-world use cases for this feature.

### Upgrade / Downgrade Strategy

Upgrades and downgrades are safe as far as Kubelet stability is concerned. It’s possible a vendor may ship vital pieces of configuration within a drop-in directory. If the Kubelet downgrades to a version that doesn’t support reading the drop-in directory, the kubelet will not recognize the "--config-dir" flag and risk failing. However, assuming that vendor left that the original `/etc/kubelet/kubelet.conf` is in a valid state, and the flag isn't specified, there should be no risk to the system. Any configuration that exists in a drop-in dir won't be applied, but that would not affect kubelet stability.


### Version Skew Strategy

All behavior change is encapsulated within the Kubelet, so there is no version skew possible within core Kubernetes. It is possible third party tools may attempt to utilize the Kubelet’s drop-in directory before the Kubelet is upgraded to support it, which would cause silent failures. It is the responsibility of these third party tools to ensure the Kubelet is new enough to support this.


## Production Readiness Review Questionnaire


### Feature Enablement and Rollback


###### How can this feature be enabled / disabled in a live cluster?

- [] Feature gate
N/A

In addition to configuring the KUBELET_CONFIG_DROPIN_DIR_ALPHA environment variable, administrators must explicitly set the --config-dir flag in the kubelet's command-line interface (CLI) to enable this feature. Starting from the beta phase, specifying the --config-dir flag is the only way to enable this feature. The default value for `--config-dir` is an empty string, which means the feature is disabled by default.

The decision to use an environment variable (KUBELET_CONFIG_DROPIN_DIR_ALPHA) over a feature gate was made to avoid potential conflicts in configuration settings. With the current configuration flow, feature gates could lead to unexpected behavior when CLI settings conflict with the kubelet.conf.d directory. The potential issue arises when the CLI initially sets the feature gate to "off," but the kubelet configuration specifies it as "on." In this scenario, the kubelet would start with the feature gate "off," switch it to "on" during configuration rendering, and then have conflicting settings when reading the kubelet.conf.d directory, leading to unexpected behavior. By using an environment variable during the alpha phase, we provided a simpler and more predictable way to control the drop-in configuration directory for testing. In the beta phase, we are removing this environment variable to streamline configuration management and enhance the user experience.


###### Does enabling the feature change any default behavior?

No, upgrading to a version of the Kubelet with this feature will not enable the Kubelet to be configured with the drop-in directory if no flag is specified.


###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. To disable the feature, roll back by removing the --config-dir flag from the kubelet's CLI.

###### What happens if we reenable the feature if it was previously rolled back?

This feature will be re-enabled via adding back the `--config-dir` flag to the CLI.

###### Are there any tests for feature enablement/disablement?

A test will be added to assemble a single, functional kubelet configuration object from various individual drop-in config files.


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

Workloads do not directly consume this feature, it is for cluster admins during kubelet configuration.
To check if the feature is enabled, users can query the merged configuration. One way to do this is by hitting the configz endpoint using kubectl or a standalone kubelet mode.


###### How can someone using this feature know that it is working for their instance?

In alpha, the user can query their active kubeletconfiguration to see if their drop-ins have taken effect.
In beta and onwards, the user will be able to read this off logs or the API, to be determined as described above.


###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The node bootstrap time should be minimal so kubelet doesn't take too long to reconcile the configuration.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

No noticeable increase in the kubelet startup time.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

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

Invalid configuration, including issues like incorrect file permissions or misconfigured settings for the drop-in directory and files, falls under known failure modes, same as exists today with `/etc/kubernetes/kubelet.conf`


###### What steps should be taken if SLOs are not being met to determine the problem?

Fix the invalid configuration, or remove configurations.


## Implementation History

- 2023-05-04: KEP initialized.
- 2023-07-17: Alpha is implemented in 1.28
- 2023-09-25: KEP retargeted to Alpha in 1.29
- 2024-01-19: Added an [e2e](https://testgrid.k8s.io/sig-node-release-blocking#node-kubelet-serial-containerd&include-filter-by-regex=KubeletConfigDropInDir) test and set KEP target to Beta in 1.30
- 2024-09-30: Update Beta requirements
- 2025-10-02: Update to stable


## Drawbacks


## Alternatives

Reinstate the now deprecated Dynamic Kubelet Configuration

Continue to rely on CLI flags or systemd drop-in files.
