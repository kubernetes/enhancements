---
title: http probe enhancement
authors:
  - "@konghui"
owning-sig: sig-apimachinery
participating-sigs:
  - sig-node
reviewers:
  - TBD
approvers:
  - TBD
editor: "@konghui"
creation-date: 2019-07-30
last-updated: 2019-07-30
status: provisional
see-also:
replaces:
superseded-by: 
---

# Explain HTTP Probe enhancement

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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
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

Current HTTP probe is not sufficent for check http service running correct or not. In some
stuation user need to add accuracy status code and the http response content to judge
the webservice is work well or not.

HTTP Probe is a mechanism of probe pod is running correct or not. It is part of probe.
Usually HTTP response is between 200 and 400 is correct

This KEP is proposing add some extra info to check HTTP service was work correct or not.

## Motivation
Sometimes people need a accurate statusCode check for the HTTP probe. Such as I run the
elasticsearch on the kubernetes. When thereis a special request, which has a large result.
It can triger the the circuit breaker. It cause all request to the elasticsearch return
429 temporary.

Another is some special progame when it enconter a problem it will return 302 instead of 
another code.

There is also have some webprograme has it's owner healthy check page. It return success
or false in the http response content. So we need a mechanism to check the http probe 
response is same as I expect or not.

So propose enhance kubect http probe feature.

### Goals

- Some user can use accurate statusCode to identi there webservice is work correct or not
- user can judge the webservice is correct or not through response content

### Non-Goals

- some user can't judge the web service is work well through HTTP status code

## Proposal
In v1.17, we will add two condition assit us to judge the http probe result is success
or not:
- add http probe enhancement on the release-note
- add a feature gate on the kubernetes user can enable http probe enhance feature.

In v1.18, we will switch off the feature gate which will automatically enable http 
probe enhancement. However it will still be possible to revert the behavior by changing
value of the feature gate
### User Stories [optional]

#### Story 1
Some people use a health checker page. It can report the current status of the webservice.
It usually report some content like `ok` to indicate current service is ok. But for kubernetes
it hardlly to probe this webservice, because HTTP probe on the kubelet only check the HTTP
status return code. In this scenario the probe result always is success, because if web service 
was down health check page will return `not ok`, but the HTTP status code always to be 200.
#### Story 2
We running `elasticsearch` on the kubernetes. In some case if it has insufficent heap to handle
a special request will trigger circuit breaker. All of the request will be return 429 temporarily.
So I need to ignore 429 status code from the HTTP probe.

### Risks and Mitigations

The risk is if has bug, pod HTTP probe feature will not work well. User can enable extra
HTTP probe check by enable the feature gate.

## Design Details

I going to add two field on the `HTTP Probe`.
Add two field called `ExpectHTTPCodes` and `ExpectHTTPContent` on the struct `HTTPGetAction` in k8s.io/api/core/v1
```
// HTTPGetAction describes an action based on HTTP Get requests.
type HTTPGetAction struct {
	// Path to access on the HTTP server.
	// +optional
	Path string `json:"path,omitempty" protobuf:"bytes,1,opt,name=path"`
	// Name or number of the port to access on the container.
	// Number must be in the range 1 to 65535.
	// Name must be an IANA_SVC_NAME.
	Port intstr.IntOrString `json:"port" protobuf:"bytes,2,opt,name=port"`
	// Host name to connect to, defaults to the pod IP. You probably want to set
	// "Host" in httpHeaders instead.
	// +optional
	Host string `json:"host,omitempty" protobuf:"bytes,3,opt,name=host"`
	// Scheme to use for connecting to the host.
	// Defaults to HTTP.
	// +optional
	Scheme URIScheme `json:"scheme,omitempty" protobuf:"bytes,4,opt,name=scheme,casttype=URIScheme"`
	// Custom headers to set in the request. HTTP allows repeated headers.
	// +optional
	HTTPHeaders []HTTPHeader `json:"httpHeaders,omitempty" protobuf:"bytes,5,rep,name=httpHeaders"`
	// Expect status code. If return code in the ExpectStatusCode and response match EexpectHTTPContent(will ignore if ExpectHTTPContent is empty). it treat as success.
	// +optional
	ExpectHTTPCodes []int `json:"expectHTTPCodes,omitempty" protobuf:"bytes,6,rep,name=expectHTTPCodes"`
        // Expect HTTP conetnt. If http response content match expectHTTPContent and result HTTPcode in ExpectHTTPCodes (will ignore if ExpectHTTPCodes is empty). It treat as success
        // +optional
        ExpectHTTPContent `json:"expectHTTPContent,omitempty" protobuf:"bytes,7,rep,name=expectHTTPContent"`

}
```

How to match ExpectHTTPCodes:
```
If probe result HTTP code in the list of ExpectHTTPCodes. We say it match the ExpectHTTPCodes, else not.
```
How to match ExpectHTTPContent:
We support use globbing patterns test probe result content is match ExpectHTTPContent or not, for example:

|probe content	|   ExpectHTTPContent	| resuilt|
      -:    	|         :-:         	|   :-
|  helloworld	|      helloworld 	| match	|
|  helloworld   |      helloworl\*	| match	|
|  helloworld   |       helleworld	| mismatch|
|  hellworld	|      hellawor\*	| mismatch|


We introduce a new feature gate named `HTTPProbeEnhancement`. It will do flow the condition.
1. If user not enable feature gate. It works as normal. If HTTP status code between 200 and 400. return Probe.Success.
2. If user enable feature gate. Both ExpectHTTPContent and ExpectHTTPCodes is empty. We treat it as condition 1.
3. If user enable feature gate. `ExpectHTTPCodes` is not empty and `ExpectHTTPContent` is empty. If probe result HTTP status code in the ExpectHTTPCodes return Probe.Success else return Probe.Failure.
4. If user eanble feature gate. `ExpectHTTPCodes` is empty and `ExpectHTTPContent` is not empty. If probe result HTTP content match ExpectHTTPContent return Probe.Success else return Probe.Failure.
5. If user enable feature gate. `ExpectHTTPCodes` and `ExpectHTTPContent` is not empty. If both probe result HTTP code match ExpectHTTPCodes and HTTP content match ExpectHTTPContent return Probe.Success else return Probe.Failure.

### Test Plan
### Graduation Criteria
### Upgrade / Downgrade Strategy
use `HTTPProbeEnhancement` feture gate, default is disabled, If there is some problem, user can disable this fetaure gate.
## Implementation History
