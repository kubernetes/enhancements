# KEP-3028: Retry during Server Initialization

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

While an apiserver instance is initializing, its response to incoming request(s) may be inconsistent. 
This KEP provides a way where clients can opt-in to receive a `Retry-After` response from the apiserver 
while it is initializing.

## Motivation

Today an apiserver instance starts responding to incoming request(s) before it has fully initialized (the first time `/readyz` reports `OK`). 
This may result in inconsistent or incorrect response from the apiserver, for example:
- RBAC has not fully initialized: it may result in unexpected `403` for some requests.
- Incomplete discovery: This may impact namespace lifecycle controller.
- CRDs potentially unavailable (404)

Such inconsistent response from the apiserver during the initialization phase may lead to some cluster 
component(s) performing unexpected actions, as stated above. Please note that some of these issues are being 
addressed independently. 

This KEP provides a more generic way for a component, so it can react to a response that was prepared by the
apiserver while initialization was in progress.
 

### Goals

- clients can opt-in to enable this feature (request scoped).

### Non-Goals

- The scope is confined to an apiserver instance, coordination between apiserver instances
  or with a load balancer is outside the scope.
- Any changes to how a subsystem initialization logic is wired to readiness check is outside
  the scope of this design document.


## Proposal

There are a couple of assumptions we are making:
- We deem the apiserver fully initialized when the `/reazy` endpoint of the apiserver returns
  a `200` status code for the first time.
- We assume all subsystem(s) of the apiserver are properly wired to the readiness check, 
  so when `/readyz` returns `200` for the first time it implies that the apiserver has fully initialized.

This KEP introduces an opt-in approach - when a client sends a request to the apiserver, it attaches the following 
conditional header `X-Kubernetes-If-Ready: `.

If the apiserver has not fully initialized yet and the incoming request has the header, it will respond with:
```
StatusCode = 503

Header:
 Retry-After: N
 X-Kubernetes-Ready: false
```

- A status code of `503` with a header `Retry-After: N` in the response instructs the client to
  retry the request after `N` seconds.
- The response header `X-Kubernetes-Ready: false` indicates to the caller that the server understood 
  the `X-Kubernetes-If-Ready` request header and took action appropriately.
  
If the server has fully initialized, the request will be processed and the following header will be injected to the response.

```
Header:
 X-Kubernetes-Ready: true
```

Upon receipt of a `503` status code with `Retry-After` response header, the K8s client-go logic will 
automatically retry the request up to maximum retry threshold.

The caller will receive the following result:
- After certain retries the request has succeeded, and the caller will see
  `X-Kubernetes-Ready: true` in the response header.
- The underlying client-go retry logic will give up after maximum retry threshold has been reached and will 
  propagate the following error to the caller:
```
StatusCode = 503

Header:
 Retry-After: N
 X-Kubernetes-Ready: false
```

With this, we can establish the following invariants:
- Presence of the header in the response indicates that the server understands the `X-Kubernetes-If-Ready` 
  header and has taken action appropriately.
- On the other hand, absence of the header in the response indicates that the server does not
  understand the `X-Kubernetes-If-Ready` header

This gives the client enough information to decide what to do in a situation where it sends `X-Kubernetes-If-Ready` 
to the server that does not understand the header due to a version mismatch.

#### client-go Changes

- Add a new boolean field `IfReadyHeader` in `rest.Config`. This allows a component author to
  opt-in for this feature for all requests associated with a `rest.Config` instance.

- Use a custom `http.RoundTripper` that adds the header to the request inside `RoundTrip`

#### No Automatic Retry Approach

Another option for the server is to always attach the header in the response:
```
Header:
 X-Kubernetes-Ready: {true|false}
```

when it sees the `X-Kubernetes-If-Ready` header in the request, and the request is processed as usual. The caller
checks the response status code and the value of the `X-Kubernetes-Ready` header and then react accordingly. With this 
approach, the retry rests with the caller since client-go will not automatically retry here.


#### Server Run Option
- We propose adding a new boolean server option `startup-send-retry-after-if-not-initialized`. The cluster operator can set 
  this option to `true` to enable this feature on the server side. By default, this option will be disabled.

Given that the conditionality is opt-in, what is the motivation for adding this server option? If the server logic
becomes a bottleneck for some reason, we have a way to disable it on the server.



### Notes/Constraints/Caveats (Optional)

We are going to add the following overhead in serving every request:
- check for the presence of the header `X-Kubernetes-If-Ready`.
- the server also needs to check if the apiserver has fully initialized. The cost could be 
  equivalent to checking if a go channel is closed.

Using a benchmarking test, We can measure the latency cost incurred.

## Design Details

At a high level, we can roughly translate this KEP into the following server filter:
```
func(w http.ResponseWriter, req *http.Request) {
 var sendRetryAfter bool
 if value, ok := req.Header.["X-Kubernetes-If-Ready"]; ok {    
  sendRetryAfter = !hasAPIServerFullyInitialized()
  if !sendRetryAfter {
    w.Header().Set("X-Kubernetes-Ready", "true")
  }
 }

 if !sendRetryAfter {
   handler.ServeHTTP(w, req)
   return
 }

 w.Header().Set("X-Kubernetes-Ready", "false")
 w.Header().Set("Retry-After", "5")
 http.Error(w, "The APIServer has not fully initialized yet, please try again later.",
   http.StatusServiceUnavailable)
}
```


### Test Plan

Unit tests will verify that the response is set appropriately for the following use cases:
- server has not fully initialized yet, request has the header
- server has not fully initialized yet, request does not have the header
- server has fully initialized, request has the header
- server has fully initialized, request does not have the header


In addition, we can have the following:
- integration tests to verify the above use cases
- benchmarking test to measure the latency overhead   


### Version Skew Strategy

- client talks to a server (older version) that does not understand the header: the caller will not see the header in the response.
- client talks to a server that understands the header

## Production Readiness Review Questionnaire

Enabling the feature on the server does not change the default behavior. The client must opt in via a request header to use it.

## Implementation History
