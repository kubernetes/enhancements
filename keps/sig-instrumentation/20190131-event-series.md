---
title: Event series API
authors:
  - "@gmarek"
owning-sig: sig-instrumentation
participating-sigs:
  - sig-instrumentation
  - sig-scalability
  - sig-architecture
reviewers:
  - "@wojtekt"
  - "@bgrant0607"
approvers:
  - "@wojtekt"
  - "@bgrant0607"
editor: TBD
creation-date: 2019-01-31
last-updated: 2019-01-31
status: implementable
see-also:
  - n/a
replaces:
  - n/a
superseded-by:
  - n/a
---

# Title

Event series API

## Table of Contents

<!-- toc -->
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Graduation Criteria

Beta:

- Ensure we do not have any performance regression compared to the orignal API
- test coverage for edge-cases


GA:

- Update Event semantics such that they'll be considered useful by app developers
- Reduce impact that Events have on the system's performance and stability
- Switch all the controllers to use the new Event API

## Implementation History

- 2017-10-7 design proposal merged under [kubernetes/community](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/instrumentation/events-redesign.md)
- 2017-11-23 Event API group is [merged](https://github.com/kubernetes/kubernetes/pull/49112)
