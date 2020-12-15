---
title: future-of-kubectl-cp
authors:
  - "@sallyom"
owning-sig: sig-cli
participating-sigs:
  - sig-usability
reviewers:
  - "@liggitt"
  - "@brendandburns"
approvers:
  - "@pwittrock"
  - "@soltysh"
editor: TBD
creation-date: 2019-09-20
last-updated: 2019-09-20
status: provisional
---

# future-of-kubectl-cp

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals for kubectl cp](#goals-for-kubectl-cp)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [UserA wants to copy a file or a directory from a local host to a pod (or a container in a pod)](#usera-wants-to-copy-a-file-or-a-directory-from-a-local-host-to-a-pod-or-a-container-in-a-pod)
    - [UserB wants to copy a file or a directory from a pod to a local host](#userb-wants-to-copy-a-file-or-a-directory-from-a-pod-to-a-local-host)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

This document summarizes and originates from this email thread, 
[Proposal to drop kubectl cp](https://groups.google.com/forum/?utm_medium=email&utm_source=footer#!msg/kubernetes-sig-cli/_zUy67lK49k/aE6vncYiAgAJ).   

This document aims to solidify the future of `kubectl cp` as a tool that provides basic function of copying files between local environments and pods.  Any advanced use cases
such as those involving symlinks or modifying file permissions should be performed outside of `kubectl cp` through `kubectl exec`, addons, or shell commands.    

Over the past few releases, there have been numerous security issues with `kubectl cp` that have resulted in release updates in all supported versions of kubectl.
At the same time,any new PR that extends `kubectl cp` must undergo extra reviews to evaluate security threats that may arise [1][2].  Over the past few months,
security fixes have required dropping edge cases and function of the command.  It is increasingly difficult to maintain a cp command that is both
useful and secure.  There are alternative approaches that provide the same function as `kubectl cp` [3].  Using `kubectl exec ...| tar`
provides transparency when copying files as well as mitigations for path traversals, symlink directory escapes, tar bombs, and other exploits.
Use of tar is more featureful, in that it can preserve file permissions and copy pod-to-pod.  Also, `kubectl cp` is dependent on the tar binary
in a container.  A malicious tar binary is outside of what `kubectl cp` can control.    

With all of this in mind the cost and risk of maintaining the cp command should be weighed against what is considered crucial functionality in kubectl. 
It's better to address 80% of use cases with a simple tool than trying to address the remaining 20% at the cost of risking those 80%.     

[1] https://github.com/kubernetes/kubernetes/pull/78622   
[2] https://github.com/kubernetes/kubernetes/pull/73053   
[3] https://gist.github.com/tallclair/9217e2694b5fdf27b55d6bd1fda01b53   

## Motivation

- The `kubectl cp` command has been the subject of multiple security vulnerability reports.
    * [CVE-2018-1002100](http://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2018-1002100)
    * [CVE-2019-1002101](http://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2019-1002101)
    * [CVE-2019-11246](https://cve.mitre.org/cgi-bin/cvename.cgi?name=2019-11246)
    * [CVE-2019-11249](https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2019-11249)
- To use `kubectl cp`, container images are required to have the tar binary. `kubectl cp` is not available when running containers from the minimal [scratch image](https://hub.docker.com/_/scratch/).    
  Running from scratch is by itself a tactic to securing containers, as it encourages the best practice of limiting the tools packaged in an image to only what's required by a workload.   

This proposal is that `kubectl cp` should perform only basic copying of files.  Advanced features of file copying should be out of scope for `kubectl cp`.  

### Goals for kubectl cp

- Provide simple function to copy a single file or directory, without advanced features such as symlinks or file permission changes
- Ensure, through extra review process tbd, that features added to `kubectl cp` are well understood and easy to secure.
- Offer users example `kubectl exec`/ shell commands to address advanced options of copying files. (There are a few already in --help)
  - As an alternative, problematic `kubectl cp` code (handling symlinks) can be replaced with code to shell out to `kubectl exec ... | tar`

### Non-Goals

For either of these, a separate proposal weighing the cost/benefit would be required.  These are out of scope of this proposal to simplify `kubectl cp`:
- Rewrite `kubectl cp` to not use tar, by modifying CRI as outlined partially [here](https://github.com/kubernetes/kubernetes/issues/58512). 
- Rewrite `kubectl cp` to be functional in scratch based containers through use of ephemeral containers as outlined [here](https://github.com/kubernetes/kubernetes/issues/58512#issuecomment-528384746)

## Proposal

- `kubectl cp` should provide simple function of copying single file or directory between local environments and pods.
- Identify and document `kubectl exec` commands to address more advanced options for copying files.  
- Provide users attempting to use `kubectl cp + symlinks/etc` with output showing comparable `kubectl exec ...| tar` cmds.
- It is up for a decision in this proposal whether the community prefers to implement the `shelling out to tar from within kubectl cp` 
or leave as suggestions in error output. 
- Barring decision of the above, only the user stories listed below should be supported by `kubectl cp`.  If additional user stories are added via shelling out to tar from kubectl, 
  those will be outlined below.    

### User Stories

#### UserA wants to copy a file or a directory from a local host to a pod (or a container in a pod)

```console
 $ kubectl cp localdir somens/somepod:/poddir (-c acontainer)
 $ kubectl cp localdir somens/somepod:/poddir (-c acontainer)
 $ kubectl cp localdir/filea somens/somepod:/poddir (-c acontainer)
```

#### UserB wants to copy a file or a directory from a pod to a local host

```console
 $ kubectl cp somens/somepod:/poddir localdir
 $ kubectl cp somens/somepod:/poddir/filea localdir/filea
```

### Implementation Details/Notes/Constraints [optional]

### Risks and Mitigations

Any scripts or automation that currently rely on advanced features of `kubectl cp` will be broken.
To mitigate, detailed information about why the command now fails as well as example `kubectl exec ...| tar` alternatives will be output. 

## Design Details

### Test Plan

Include test automation script that calls all `kubectl cp` flags or options that are removed.
Ensure that failure includes example alternative approach, plus information about the failure, skipped files, etc.

### Graduation Criteria

### Upgrade / Downgrade Strategy

`kubectl cp` function removed as a result of a CVE fix or other will be documented clearly.
Information about why subcommand/option is no longer supported, what files are skipped, and also alternative `kubectl exec ...| tar` commands 
will be included in failed command output.  This output will then always be given (not just for a deprecation period). 

### Version Skew Strategy

## Implementation History

## Drawbacks

Automation scripts that include `kubectl cp` will be broken if options and features are removed from the command.
The motivation of improving security is weighed against this potential drawback.  
