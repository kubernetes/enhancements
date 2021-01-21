# KEP-1295: Insecure Backend Proxy

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
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
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
2. There will not be e2e tests written because the scenario under which this API is effective is only in a mis-configured
   cluster where a kubelet has not refreshed its serving certs.  There is an existing positive and negative integration
   [test](https://github.com/kubernetes/kubernetes/blob/release-1.20/test/integration/apiserver/podlogs/podlogs_test.go#L141-L164)
   which the sig leads believe is sufficient.

### Graduation Criteria

The problem and solution are well understood and congruent to our past solutions.
This new API field will start at beta level.

### Upgrade / Downgrade Strategy

Because the change is isolated to non-persisted API contracts with the kube-apiserver, there are no skew or upgrade/downgrade considerations.

### Version Skew Strategy

Because the change is isolated to non-persisted API contracts with the kube-apiserver, there are no skew or upgrade/downgrade considerations.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: AllowInsecureBackendProxy
    - Components depending on the feature gate: kube-apiserver

* **Does enabling the feature change any default behavior?**
  No, all default behavior remains the same with the feature gate on or off.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, the feature can be disabled after enablement.
  Because no data is persisted via this API, there is no impact that lingers across kube-apiserver restarts.

* **What happens if we reenable the feature if it was previously rolled back?**
  Because no data is persisted via this API, there is no impact that lingers across kube-apiserver restarts.

* **Are there any tests for feature enablement/disablement?**
  Because no data is persisted via this API, there is no lingering memory in the system to check.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  This is contained to a single binary, with no persisted data.
  The worst failure mode is when an HA cluster has some members with the feature off and some members with the feature on.
  In such a case, the user observed behavior going through a load balancer is inconsistent until the cluster settles.

* **What specific metrics should inform a rollback?**
  If there is a notable increase in failed pod/logs calls, it may be indicative of the new code causing a problem.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Yes.  This was explicitly tested in the OpenShift distro when the feature went to beta.
  During HA cluster upgrades, the client observed behavior was inconsistent (as expected), but once all members had
  the feature gate consistent it was fine.
  Skew also worked correctly, with new clients sending the additional option simply not connecting as they wish, failing
  in the safe direction.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  No.

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
`pods_logs_insecure_backend_total` has a label `skip_tls_allowed` which will count how often this value is set by clients.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [ ] Metrics
    - Metric name: 
      `pods_logs_insecure_backend_total` indicates usage.
      `pods_logs_backend_tls_failure_total` indicates how often usage of the option may have allowed a connection to be established.
    - Components exposing the metric: kube-apiserver

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  pods/logs can suffer errors today based on user input because the kubelet cannot be verified.
  Because this is driven based on clients, different clusters may have different "reasonable" starting values.
  However, there should not be a marked increase the failure rate of pods/logs.

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  I don't think we need greater granularity here.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  No.
  This does not introduce any new calls from the kube-apiserver.

### Scalability

* **Will enabling / using this feature result in any new API calls?**
  no.
  It adds an option to an existing API call that would already have been called.

* **Will enabling / using this feature result in introducing new API types?**
  No.
  It adds a field to `PodLogOptions`, which is not a persisted API.

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
  No.

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  No

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  No.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  No impact because this feature only affects the kube-apiserver behavior.

* **What are other known failure modes?**
  There are no known failure modes.

* **What steps should be taken if SLOs are not being met to determine the problem?**
  The usual steps used to debug a pod/logs failure.
  This varies somewhat, but generally you gather.
    1. the kube-apiserver logs
    2. the pods you cannot connect to
    3. the node API running that pod
    4. the kubelet log for that node
    5. the crio log for that node
  From there you can decide how far the request is getting and whether you need to investigate the network connections.
  This is a fairly deep and rare thing to investigate today.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

Introduced as beta in 1.17.
Moving to stable in 1.21.

## Drawbacks [optional]

Why should this KEP _not_ be implemented.

## Alternatives [optional]

Similar to the `Drawbacks` section the `Alternatives` section is used to highlight and record other possible approaches to delivering the value proposed by a KEP.

## Infrastructure Needed [optional]

Use this section if you need things from the project/SIG.
Examples include a new subproject, repos requested, github details.
Listing these here allows a SIG to get the process for these resources started right away.
