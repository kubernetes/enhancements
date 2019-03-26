---
title: Remove kube-proxy's automatic clean up logic
authors:
  - "@vllry"
owning-sig: sig-network
participating-sigs:
reviewers:
  - "@andrewsykim"
approvers:
  - "@bowei"
  - "@thockin"
editor: TBD
creation-date: 2018-03-24
last-updated: 2018-03-24
status: provisional
see-also:
replaces:
superseded-by:
---

# Remove kube-proxy's automatic clean up logic

## Summary

Remove the automatic network rule cleanup from kube-proxy.
Only clean up rules from running in other proxy modes when using the `--cleanup` command.

## Motivation

kube-proxy's rule cleanup can substantially delay kube-proxy reboot time,
and is not normally necessary.

### Goals

* Avoid bugs where different proxier modules' rules overlap,
causing user-visible impact between cleanup and full sync

### Non-goals

Continuing to automatically clean up rules on-start.
Cleaning up rules is slow,
during which time traffic is interrupted (EG [k/k 75360](https://github.com/kubernetes/kubernetes/issues/75360)).

## Proposal

Only clean up leftover network rules (iptables rules etc) if `--cleanup` is set.

If `--cleanup` is not set,
start kube-proxy without cleaning up old rules.

EG remove code like [this](https://github.com/kubernetes/kubernetes/blob/e7eb742c1907eb4f1c9e5412f6cd1d4e06f3c277/cmd/kube-proxy/app/server_others.go#L180-L187):

```
	if proxyMode == proxyModeIPTables {
		...
		userspace.CleanupLeftovers(iptInterface)
		// IPVS Proxier will generate some iptables rules, need to clean them before switching to other proxy mode.
		// Besides, ipvs proxier will create some ipvs rules as well.  Because there is no way to tell if a given
		// ipvs rule is created by IPVS proxier or not.  Users should explicitly specify `--clean-ipvs=true` to flush
		// all ipvs rules when kube-proxy start up.  Users do this operation should be with caution.
		if canUseIPVS {
			ipvs.CleanupLeftovers(ipvsInterface, iptInterface, ipsetInterface, cleanupIPVS)
		}
		...
```
## Graduation Criteria

Default behavior should GA immediately,
with a strong release note.

## Implementation History

## Alternatives

Removing support for cleaning up other proxy modes has been suggested.
EG if a user wished to change proxy modes,
rather than running `kube-proxy --cleanup`,
users would be advised to restart the worker node.
