---
kep-number: 0
title: Remove kube-proxy's automatic clean up logic
authors:
  - "@bowei"
owning-sig: sig-network
participating-sigs:
reviewers:
  - "@andrewsykim"
approvers:
  - "@bowei"
  - "@thockin"
editor:
creation-date: 2018-03-24
last-updated: 2018-03-24
status: implementable
see-also:
replaces:
superseded-by:
---

# Remove kube-proxy's automatic clean up logic

## Summary

Remove the automatic network rule cleanup from kube-proxy.
Only clean up rules on boot if the `--cleanup` flag is present.

## Motivation

kube-proxy's rule cleanup has a substantial performance impact,
and is not normally necessary.

### Goals

* Reduce the typical kube-proxy startup time,
by avoiding unnecessary cleanup.

## Proposal

Add the flag `--cleanup` to kube-proxy,
and only clean up iptables/ipvs rules if `--cleanup` is set.
If cleanup is not set,
start kube-proxy without cleaning up old rules.

`--cleanup` should run cleanup for both iptables and ipvs.
There is overlap between modes and the kind of network rules used -
for example,
ipvs mode also creates some iptables rules.

## Graduation Criteria

Beta: Stable without notable bugs for at least one release.

## Implementation History

`--cleanup` is [already implemented](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/#options) in 1.13.

## Alternatives

The existing recommendation (citation?) for switching proxy mode is to restart the node.

`--cleanup`
(or a new argument, for backwards compatibility)
could be instead only run cleanup for the corresponding proxy mode
(EG `--cleanup` only cleans up iptables rules when running with `--proxy-mode iptables`) 

## Appendix

### Non-options

Continuing to automatically clean up rules on-start.
Cleaning up rules is slow,
during which time traffic is interrupted (EG [k/k 75360](https://github.com/kubernetes/kubernetes/issues/75360)).
