---
title: Component Config Validation Interface
authors:
  - "@rosti"
owning-sig: sig-cluster-lifecycle
participating-sigs:
  - sig-cluster-lifecycle
  - sig-api-machinery
reviewers:
  - "@mtaufen"
  - "@stealthybox"
  - "@sttts"
approvers:
  - "@timothysc"
editor: "@rosti"
creation-date: 2019-10-01
last-updated: 2019-10-01
status: provisional
---

# Component Config Validation Interface

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [The “experimental-config-validate“ sub-command](#the-experimental-config-validate-sub-command)
  - [Sub-command placement](#sub-command-placement)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Awkward UX for some components](#awkward-ux-for-some-components)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
  - [10/01/2019](#10012019)
- [Alternatives](#alternatives)
<!-- /toc -->


## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


## Summary

This document proposes a standard generic command line interface for config validation, that components can expose to users and tools.


## Motivation

With the advent of component configs, it became possible for cluster lifecycle tools (such as kubeadm) to generate and supply provisioned components with config files in YAML or JSON format. This is gradually increasing in popularity as maintaining a well structured and serialized in a text file config is far easier than maintaining a large set of command line arguments.

This, however, poses some problems. For example:
- The supplied config may be in an unsupported (newer or older) or unrecognized format.
- The config file may be corrupted due to disk, network, program or user error.
- A required system setting or module may not be setup properly or available, hence, prohibiting component operation with the current config.

Unfortunately, due to the nature of Kubernetes, such kinds of failures might not be detected by cluster lifecycle tools early enough and users might not be warned in time, that their configs are invalid.  
Currently, some cluster lifecycle tools vendor in and use Go code from the target components themselves to be able to verify component configs early enough during the provisioning operation, so that meaningful error messages are displayed in time directly to the user. This, however, poses its own problems as such pieces of code can deviate from the actual shipped component and can lead to false positives and false negatives.  
Hence, it’s best that validation is performed by the component itself in a one off operation.


### Goals

The main goal is to provide a stable standard command line interface for components that can be used to validate component configs and retrieve appropriate errors and warnings.


### Non-Goals

To impose the implementation of this proposal on all Kubernetes components, that provide component config. Implementing this interface is entirely optional.


## Proposal

The proposal is to introduce a new standardized sub-command for each component that implements the interface. This command can then be called by invoking the component’s executable or creating a container with it.


### The “experimental-config-validate“ sub-command

The new sub-command will be known as “experimental-config-validate”. Upon graduation to beta and GA, this sub-command will be renamed to just “config-validate”.

It must support a couple of mandatory flags:
- “--help” or “-h” - if specified, will dump usage information of the sub-command and exit without doing any config validation.
- “--file” or “-f” - supplies a YAML or JSON file containing a component config to be validated.

In all implementations of “experimental-config-validate”, the “--file” flag can be specified multiple times. If a component does not support multi-file config, supplying multiple “--file” values must return an appropriate error to the user without performing any validation.  
If no “--file” or “--help” switches are supplied, the command reads the config to be validated from the standard input.

When the “experimental-config-validate” is executed supplied with a config, it should validate it and:
- Exit with success if the supplied config is found to be valid. Each line of text printed on the standard output is considered to be a single warning message. Multiple warning messages can be printed.
- Exit with failure if an error is encountered and the component will be unable to operate normally with the config as is. Each line of text printed on the standard output is considered to be a single error message. Multiple error messages can be printed.


### Sub-command placement

The “experimental-config-validate” sub-command must be placed under the command that houses the component whose config needs to be validated.  
Below is an example that uses kubelet:
```
# kubelet experimental-config-validate --file=config.yaml
```

In case a component is housed in a binary along with other components (hyperkube for example), multiple “experimental-config-validate” commands can exist under different component sub-commands.  
Below is an example of such usage:
```
# hyperkube kubelet experimental-config-validate --file=kubelet-config.yaml
# hyperkube kube-proxy experimental-config-validate --file=kube-proxy-config.yaml
```

In case a component’s binary is housed in a container image, it is recommended that this image is able to take the “experimental-config-validate” command and it’s arguments as if a CMD instruction was used in a Dockerfile. The following example shows how this can be used with docker:
```
# docker run -it -v $(pwd)/kube-proxy-config.yaml:/kube-proxy-config.yaml k8s.gcr.io/kube-proxy:v1.16.0 experimental-config-validate --file=/kube-proxy-config.yaml
```

The same container image concept can be extended to images, that handle multiple components:
```
# docker run -it -v $(pwd)/kube-proxy-config.yaml:/kube-proxy-config.yaml k8s.gcr.io/hyperkube:v1.16.0 kube-proxy experimental-config-validate --file=/kube-proxy-config.yaml
```

It is up to the implementer’s judgment, to allow “experimental-config-validate” to be visible or hidden in the list of available commands in the component help messages. However, in all cases, implementers must document the implementation.


### Risks and Mitigations

#### Awkward UX for some components

The standard command line interface, proposed in this document, may not fit neatly in the CLI of every component. Hence, it may look awkward in the eyes of users. The mitigation to this is to allow implementers to choose if they want to hide the "experimental-config-validate" command or not.


## Design Details

### Test Plan

Every component is free to choose the best way to test the proposed features. However, integration tests to test the command line behavior are recommended.


### Graduation Criteria

This proposal is for an Alpha version only. Superseding or moving to Beta and GA will be decided upon in the future, based on user and developer feedback.


### Upgrade / Downgrade Strategy

As the proposal does not require storing any new state or changing any existing component state, upgrade and downgrade strategies are not required.


### Version Skew Strategy

Components are required to check for validity only on config formats and versions, which can be used for normal operation. In particular, older and newer config versions, that cannot be used with the current component version should not be validated, but rather reported as invalid along with an appropriate error message.


## Implementation History

### 10/01/2019

Initial proposal filed, containing Summary, Motivation, Proposal and other sections.


## Alternatives

Right now there are a couple of alternatives to this proposal:
- Have users to vendor in Go code to verify their component’s configs. This, however, is cumbersome, requires writing code and is prone to false positives and false negatives.
- Not provide component config validation interface. In this way users and tools won’t be able to detect bad or unsupported config format until the actual component is started with it. However, for many users that may be way too late and hard to detect.
