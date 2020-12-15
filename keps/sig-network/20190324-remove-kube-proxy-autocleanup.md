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
last-updated: 2018-04-02
status: implementable
see-also:
replaces:
superseded-by:
---

# Remove kube-proxy's automatic clean up logic

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-goals](#non-goals)
- [Proposal](#proposal)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

Remove any clean up functionality in kube-proxy that is done automatically/implicitly.
More concretely, kube-proxy should only attempt to clean up its proxy rules when the `--cleanup` flag is set.

## Motivation

Historically, kube-proxy has been built with a "clean up" functionality that allows it to find and remove any proxy
rule that it created. This functionality runs in two steps. The first step requires an explicit request from
the user by setting the `--cleanup` flag in which case kube-proxy would attempt to remove proxy rules associated with
all its proxy modes and then exit. The second step is done automatically by kube-proxy on start-up in which kube-proxy
would detect and remove any proxy rules associated with the proxy modes it was not currently using.

We've learned by now that trying to automatically clean up proxy rules is prone to bugs due to the complex dependency
between all the proxy modes and the Linux kernel. There are overlapping proxy rules between the iptables
and IPVS proxy modes which can result in connectivity issues during start-up as kube-proxy attempts to clean up
these overlapping rules. In the worse case scenario, kube-proxy will flush active iptable chains every time it is restarted.
This KEP aims to simplify this clean up logic, and as a result, make kube-proxy's behavior more predictable and safe.

### Goals

* Simplify the clean up code path for kube-proxy by removing any logic that is done implicitly by kube-proxy.
* Remove bugs due to overlapping proxier rules, causing user-visible connectivity issues during kube-proxy startup.

### Non-goals

* Re-defining the set of proxy rules that should be cleaned up.
* Improving the performance of kube-proxy startup.

## Proposal

Only clean up iptables and IPVS proxy rules if `--cleanup` is set.
If `--cleanup` is not set, start kube-proxy without cleaning up any existing rules.

For example, remove code like [this](https://github.com/kubernetes/kubernetes/blob/e7eb742c1907eb4f1c9e5412f6cd1d4e06f3c277/cmd/kube-proxy/app/server_others.go#L180-L187) where kube-proxy attempts to clean up proxy rules in a proxy mode it is not using.

```go
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

With proxy clean up always requiring an explicit request from the user, we will also recommend users to reboot their nodes if they choose to
switch between proxy modes. This was always expected but not documented/vocalized well enough. The `--cleanup` flag should only be used in the event that a
node reboot is not an option. Accompanying a proxy mode switch with a node reboot ensures that any state in the kernel associated with the previous
proxy mode is cleared. This expectation should be well documented in the release notes, the Kubernetes docs, and in kube-proxy's help command.

## Graduation Criteria

* kube-proxy does not attempt to remove proxy rules in any proxy mode unless the `--cleanup` flag is set.
* the expectations around kube-proxy's clean up behavior is stated clearly in the release notes, docs and in kube-proxy's help command.
* there is documentation strongly suggesting that a node should be rebooted along with any proxy mode switch

## Implementation History

* 2019-03-14: initial [bug report](https://github.com/kubernetes/kubernetes/issues/75360) was created where kube-proxy would flush iptable chains even when the proxy mode was iptables

## Alternatives

* Removing support for cleaning up other proxy modes has been suggested. For example, if a user wished to change proxy modes,
rather than running `kube-proxy --cleanup`, users would be advised to restart the worker node.
* Continue to support automatic/implicit clean up of proxy rules by fixing any existing bugs where overlapping rules are removed.
Though we can continue to support this, doing so correctly is not trivial. The overlapping rulesets between the existing
proxy modes makes it difficult to fix this without adding more complexity to kube-proxy or changing the proxy rules in an incompatible way.
