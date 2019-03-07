---
kep-number: 0
title: RunAsGroup support in PodSpec and PodSecurityPolicy
authors:
  - "@krmayankk"
owning-sig: sig-node
participating-sigs:
  - sig-auth
reviewers:
  - "@tallclair"
  - "@mrunalp"
approvers:
  - "@liggitt"
  - "@derekwaynecarr"
editor: TBD
creation-date: 2017-06-21
last-updated: 2017-09-14
status: implementable
---

# RunAsGroup support in PodSpec and PodSecurityPolicy

https://github.com/kubernetes/community/blob/master/contributors/design-proposals/auth/runas-groupid.md

## Table of Contents

* [Graduation Criteria](#graduation-criteria)

* [Implementation History](#implementation-history)


## Graduation Criteria

- Publish Test Results from Master Branch of Cri-o To http://prow.k8s.io [#72253](https://github.com/kubernetes/kubernetes/issues/72253)
- Containerd and CRI-O tests included in k/k CI [#72287](https://github.com/kubernetes/kubernetes/issues/72287)
- Make CRI tests failures as release informing

## Implementation History
- Proposal merged on 9-18-2017
- Implementation merged as Alpha on 3-1-2018 and Release in 1.10
- Implementation for Containerd merged on 3-30-2018 
- Implementation for CRI-O merged on 6-8-2018
- Implemented RunAsGroup PodSecurityPolicy Strategy on 10-12-2018
- Planned Beta in v1.14
