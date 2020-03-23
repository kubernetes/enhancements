# KEP-1630: Expand kubeconfig to configure client-side rate limits

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [in-cluster kubeconfig overrides](#in-cluster-kubeconfig-overrides)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

- [ ] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubernetes REST clients use a simple token bucket rate limiter to control the pace of requests from a client.
We will expose the values already present in the [REST config](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/client-go/rest/config.go#L114-L120), via a kubeconfig file inside the client stanza.
We will wire the in-cluster kubeconfig file to honor the presence of a new file to allow customization inside a cluster at the discretion of a different, user provided, controller.

## Motivation

We've always had reasons to override these values, so we created one-off arguments (`--kube-api-burst`, `--kube-api-qps`).
With widespread adoption of operators and other in-cluster workloads, having a standard way to provide these values makes sense.
In addition, with the introduction of API priority and fairness, being able to configure and even disable this client-side rate limiting in compatible environments is important.

### Goals

1. Allow easy and consistent configuration for the existing client-side rate limiting.

### Non-Goals

1. Allow customization of client-side rate limiting strategies.

## Proposal

Add to the `AuthInfo` section of a kubeconfig.
This section identifies a particular client identity.
This identity spans all the contexts with different namespaces, so it is a closer logical match.
You don't generally want different QPS characteristics based on namespace.

```go
type AuthInfo struct {
	// ...
	// QPS indicates the maximum QPS to the master from this client.
	// If it's zero, the created RESTClient will use DefaultQPS: 5
	QPS float32

	// Maximum burst for throttle.
	// If it's zero, the created RESTClient will use DefaultBurst: 10.
	Burst int32
}
```

### in-cluster kubeconfig overrides
The in-cluster config is the automatic configuration that exists within pods.
The client code itself can be written to consume an overrides.kubeconfig file, which will be merged with the standard
kubeconfig merging rules that we use the env var path today.
The controller code will *not* be updated.
If a cluster-admin wants to control this file, he can provide his own controller to write content into this file in 
serviceaccount token secrets.

### Test Plan

This is heavily unit testable.

### Graduation Criteria

The API itself is well-known since near the beginning of kube.
We are simply exposing existing values, it will enter as GA.

### Upgrade / Downgrade Strategy

Old clients will ignore the unknown field.
Old `kubectl` will rewrite the stanza and remove the new values.

### Version Skew Strategy

Old clients will ignore the unknown field.
Old `kubectl` will rewrite the stanza and remove the new values.

## Implementation History

## Drawbacks

## Alternatives

1. Connecting to the kube-apiserver to retrieve a value for default QPS and burst.
   No default value is going to be good across all workloads.
   It's also mechanicaly weird to create a client to get a client.

