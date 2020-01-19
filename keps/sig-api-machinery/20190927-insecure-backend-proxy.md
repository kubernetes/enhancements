---
title: Insecure Backend Proxy
authors:
  - "@deads2k"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-api-machinery
  - sig-auth
  - sig-cli
reviewers:
  - "@sttts"
  - "@cheftako"
  - "@liggitt"
  - "@soltysh"
approvers:
  - "@lavalamp"
  - "@mikedanese"
editor: TBD
creation-date: 2019-09-27
last-updated: 2019-09-27
status: implementable
see-also:
replaces:
superseded-by:
---

# Insecure Backend Proxy

When trying to get logs for a pod, it is possible for a kubelet to have an expired serving certificate.
If a client chooses, it should be possible to bypass the default behavior of the kube-apiserver and allow the kube-apiserver
to skip TLS verification of the kubelet to allow gathering logs.  This is especially important for debugging
misbehaving self-hosted clusters.

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories [optional]](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [x] kubernetes/enhancements issue in release milestone, which links to KEP https://github.com/kubernetes/enhancements/issues/1295
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

When trying to get logs for a pod, it is possible for a kubelet to have an expired serving certificate.
If a client chooses, it should be possible to bypass the default behavior of the kube-apiserver and allow the kube-apiserver
to skip TLS verification of the kubelet to allow gathering logs.  This is safe because the kube-apiserver's credentials
are always client certificates which cannot be replayed by an evil-kubelet and risk is contained to an evil-kubelet
returning false log data.  If the user has chosen to accept this risk, we should allow it for the same reason we
have an option for `--insecure-skip-tls-verify`.

## Motivation

On self-hosted clusters it is possible to end up in a state where a kubelet's serving certificate has expired so a kube-apiserver
cannot verify the kubelet identity, *but* the kube-apiserver's client certificate is still valid so the kubelet can still
verify the kube-apiserver.  In this condition, a cluster-admin may need to get pod logs to debug his cluster.

### Goals

1. Allow cluster-admins to get pod logs from kubelets with expired serving certificates.  This will include an API change
and an addition argument to `kubectl log`

### Non-Goals

1. Allow any bidirectional traffic proxied to kubelets.  This may be a future objective, but is not in scope for the current KEP.

## Proposal

In [PodLogOptions](https://github.com/kubernetes/api/blob/d58b53da08f5430bb0f4e1154a73314e82b5b3aa/core/v1/types.go), 
add a `InsecureSkipTLSVerifyBackend bool`
```go
// PodLogOptions is the query options for a Pod's logs REST call.
type PodLogOptions struct {
	// ... existing fields snipped
	
	// insecureSkipTLSVerifyBackend indicates that the apiserver should not confirm the validity of the 
	// serving certificate of the backend it is connecting to.  This will make the HTTPS connection between the apiserver
	// and the backend insecure. This means the apiserver cannot verify the log data it is receiving came from the real
	// kubelet.  If the kubelet is configured to verify the apiserver's TLS credentials, it does not mean the 
	// connection to the real kubelet is vulnerable to a man in the middle attack (e.g. an attacker could not intercept
	// the actual log data coming from the real kubelet).
	// +optional
	InsecureSkipTLSVerifyBackend bool `json:"insecureSkipTLSVerifyBackend,omitempty" protobuf:"varint,9,opt,name=insecureSkipTLSVerifyBackend"`
}
```
The streamer for logs already prevents redirects (see https://github.com/kubernetes/kubernetes/blob/4ee9f007cbc88cca5fa3e8576ff951a52a248e3c/pkg/registry/core/pod/rest/log.go#L83) , so an evil kubelet intercepting this traffic cannot redirect the kube-apiserver
to use its high powered credentials for a nefarious purpose.
The `LocationStreamer` can take an additional argument and the `LogLocation`'s `NodeTransport` can actually produce a purpose
built transport for insecure connections.

To make this easier to use, we can add an `--insecure-skip-tls-verify-backend` flag to `kubectl log` which plumbs the option.
This part of the KEP is a nice to have, since the kube-apiserver owns the backing capability which has general utility.

### User Stories [optional]

#### Story 1

#### Story 2

### Implementation Details/Notes/Constraints [optional]

This design is safe based on the following conditions:

1. kube-apiservers only authenticate to kubelets using certificates.  See `--kubelet-client-certificate`.  Certificate based
authentication does not send replayable credentials to backends.
2. Clients must opt-in to the functionality and the documentation must include the impact in plain english.
3. Evil kubelets cannot trick kube-apiservers into using their credentials for any other purpose.
In order to use the kube-apiserver creds, the target of any proxy
must terminate the connection.  Since the URL is chosen by the kube-apiserver the evil kubelet cannot rewrite that destination URL.  The destination is the URL for
getting logs for one particular pod.


### Risks and Mitigations

A super user with write permissions to Node and Pod API objects and read permissions to Pod logs to make the API
server exfiltrate data at `https://<nodeName>:<nodePort>/containerLogs/<podNamespace>/<podName>/<containerName>`, where
all the bracketed parameters were under their control.  
This is ability is already present for someone with full API control via an APIService configured to point to a service
with `insecureSkipTLSVerify:true`, with a similar restriction on the path of requests sent to it (must have a leading 
`/apis/<group>/<version>` path prefix).
This could affect 
the only scenarios I can really think of are:
1. unsecured kubelets from another cluster (or kubelets from another cluster using the same CA)
2. a non-kubelet endpoint that ignored the specified path and served something confidential (and was unsecured or
honored the CA that signed the apiserver's cert)
Both scenarios are pretty unlikely.

Trying to restrict the insecureSkipTLSVerify option to ignoring only the expiry date requires reimplementing the TLS 
handshake with a custom method to 
1. get the serving cert
2. pull the notAfter to find a time when the cert was valid
3. construct a set of custom verify options at that time to verify the signature and hostname
The risk in doing that is greater than the additional benefit.


## Design Details

### Test Plan

1. Positive and negative tests for this are fairly easy to write and the changes are narrow in scope.

### Graduation Criteria

The problem and solution are well understood and congruent to our past solutions.
This new API field will start at beta level.

### Upgrade / Downgrade Strategy

Because the change is isolated to non-persisted API contracts with the kube-apiserver, there are no skew or upgrade/downgrade considerations.

### Version Skew Strategy

Because the change is isolated to non-persisted API contracts with the kube-apiserver, there are no skew or upgrade/downgrade considerations.

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks [optional]

Why should this KEP _not_ be implemented.

## Alternatives [optional]

Similar to the `Drawbacks` section the `Alternatives` section is used to highlight and record other possible approaches to delivering the value proposed by a KEP.

## Infrastructure Needed [optional]

Use this section if you need things from the project/SIG.
Examples include a new subproject, repos requested, github details.
Listing these here allows a SIG to get the process for these resources started right away.
