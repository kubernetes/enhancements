# KEP-4460: Enable per-request Read/Write Deadline

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
      - [Client Hanging Indefinitely:](#client-hanging-indefinitely)
      - [Request Handler Running Indefinitely on the Server:](#request-handler-running-indefinitely-on-the-server)
      - [Impact When an HTTP/1x Connection is Hijacked:](#impact-when-an-http1x-connection-is-hijacked)
- [Design Details](#design-details)
  - [How We Manage Request Timeout Today:](#how-we-manage-request-timeout-today)
  - [Enabling Per-Request Read/Write Deadline](#enabling-per-request-readwrite-deadline)
      - [Write Deadline](#write-deadline)
      - [Read Deadline](#read-deadline)
      - [Integrate FlushError](#integrate-flusherror)
  - [Termination of the Request Handler](#termination-of-the-request-handler)
  - [Convert the Asynchronous Timeout Filter for Mutating-Request to Synchronous:](#convert-the-asynchronous-timeout-filter-for-mutating-request-to-synchronous)
  - [Post-Timeout Observability:](#post-timeout-observability)
      - [Monitor Post-Timeout Activity using context.AfterFunc:](#monitor-post-timeout-activity-using-contextafterfunc)
  - [Audit Log:](#audit-log)
  - [Impact on the Client](#impact-on-the-client)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist


Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

With this proposal, the Kubernetes API server will enable per-request read/write
timeouts using the `ResponseController` functionality provided by the golang net/http library.
With the enablement of per-request read/write timeout, the `timeout` filter in the
kube-apiserver will become redundant, and thus be removed from the server handler chain.


## Motivation

The `timeout` filter has been a source of many hard to debug data races in the past. 
With the removal of the `timeout` filter from the apiserver handler chain, the entire
request handler will be executed on a single goroutine. Thus it will eliminate 
the data race conditions that are inherent with the `timeout` filter.

### Goals

The primary goal is to reduce data race conditions by removing the `timeout`
filter from the apiserver request handler chain.

### Non-Goals

- WATCH requests are outside the scope of this proposal. Today, the `timeout`
handler does not handle any long-running requests including WATCH, it only 
handles short (non long-running) requests. The scope of this proposal is the 
same, short (non long-running) requests only. Today, for any WATCH, if we 
inspect the context of the request, it would not have any deadline. if we want 
to enable read/write deadline for WATCH, then an orthogonal effort to wire the
context of a WATCH request with a proper deadline is a prerequisite, and such
effort is outside the scope of this proposal.

- All other long-running requests are also outside the scope of this proposal.


## Proposal

A new feature flag `PerHandlerReadWriteTimeout` will be introduced, when enabled:
- On start-up, the kube-apiserver will not add the [timeout filter](https://github.com/kubernetes/kubernetes/blob/9791f0d1f39f3f1e0796add7833c1059325d5098/staging/src/k8s.io/apiserver/pkg/server/filters/timeout.go#L38-L62) 
  to the request handler chain.
- To every incoming non long-running request, the kube-apiserver will apply
  a request-scoped read/write deadline.
- Use a synchronous finisher in place of the the [asynchronous finisher](https://github.com/kubernetes/kubernetes/blob/3a4c35cc89c0ce132f8f5962ce4b9a48fae77873/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/finisher/finisher.go#L87-L122) 
  that is used today in order to asynchronously execute the admission and storage
  handling of mutating request(s). On the other hand, the synchronous finisher 
  will execute the request in the same goroutine.
  
We also need to maintain parity in terms of observability - the logs/metrics that
are generated for requests that time out today, should also be available when this 
feature is enabled.

### User Stories (Optional)

#### Story 1

#### Story 2

### Notes/Constraints/Caveats (Optional)


### Risks and Mitigations

##### Client Hanging Indefinitely:

With the `timeout` filter enabled, it sends a `504 GatewayTimeout` response
or an `error` to the client, if the request handler it is executing asynchronously
does not return within the allotted deadline.

If the client application does not enforce any client-side timeout, for example:
 - not setting a timeout to the underlying `http.Client` or `http.Transport` object
 - not setting a deadline to the `Context` object of an `http.Request`

> It's worth noting that `client-go` does not enforce any global timeout by default.

In this scenario, the client can potentially hang indefinitely, waiting for a
response from the server. 
It's worth noting that for HTTP/1x, [net/http does not execute the request handler in a new goroutine](https://github.com/golang/go/blob/b8ac61e6e64c92f23d8cf868a92a70d13e20a124/src/net/http/server.go#L2031-L2039).

In fact, we can reproduce a "forever" hanging client with HTTP/1x:
```
func TestClientHangingForeverWithHTTP1(t *testing.T) {
	clientDoneCh, handlerDoneCh := make(chan struct{}), make(chan error, 1)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer close(handlerDoneCh)

		ctrl := http.NewResponseController(w)
		if err := ctrl.SetWriteDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
			t.Errorf("expected no error from SetWriteDeadline, but got: %v", err)
			return
		}
		if err := ctrl.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
			t.Errorf("expected no error from SetReadDeadline, but got: %v", err)
			return
		}

		<-clientDoneCh
	}))


	defer server.Close()
	server.StartTLS()

	client := server.Client()
	func() {
		defer close(clientDoneCh)
		client.Get(server.URL + "/foo")
	}()

	select {
	case <-handlerDoneCh:
	case <-time.After(wait.ForeverTestTimeout):
		t.Errorf("expected the request handler to have terminated")
	}
}
```

Even though the read/write deadline is set on the server side for this request, the
client never receives any response or error from the server. We have opened an
[issue](https://github.com/golang/go/issues/65526) with the golang team to track this.


With the removal of the `timeout` filter, we need to ensure that a client, not 
enforcing any client-side timeout, never blocks indefinitely due to the corresponding
request handler being frozen on the server side. In order to maintain parity, the
client should receive a response/error after the allotted deadline for the 
request elapses on the server.


Mitigation Steps:
- a) If this is a bug, we can wait until the fix is in the net/http package.
- b) We can enable this feature for http/2.0 request(s) only, all
     http/1x request(s) will continue to be served using the `timeout` filter.
- c) We enable this feature for both http/1x and http/2.0; for http/1x requests
     we keep the `timeout` filter enabled.

Unit tests must be in place to verify that a frozen request handler
does not, as a consequence, block the corresponding client:
```
neverDoneCh := make(chan struct{})
func(w http.ResponseWriter, req *http.Request) {
	<-neverDoneCh
}
```


##### Request Handler Running Indefinitely on the Server:

Since there is no mechanism to stop a goroutine from the outside, we rely on the
request handler goroutine to self-terminate as soon as possible once the read/write 
deadline exceeds. We provide the following mechanisms to the handler so it knows 
when to terminate:
- The handler detects that the `context` associated with the request has exceeded
  its deadline, and so it terminates.
- In the course of serving a request, the handler invokes the `Write` or `FlushError`
  method of the `ResponseWriter` object. The invocation will result in either an error
  or a panic once the read/write deadline exceeds. The handler terminates as a consequence


The proposed feature will be in par with the `timeout` filter, if we decorate the
`ResponseWriter` object to throw a panic once the read/write deadline exceeds.


##### Impact When an HTTP/1x Connection is Hijacked:

The hijacker is responsible for managing the state of a hijacked connection. Add
a test to demonstrate that setting a read/write deadline before
the connection is hijacked does not interfere with the operation of the hijacker.

There are two code sites that invoke `Hijack`:
- apimachinery/pkg/util/proxy/upgradeaware.go
- apimachinery/pkg/util/httpstream/spdy/upgrade.go

Add test(s) to verify that with read/write timeout enabled, the above 
features work as expected with hijacked connections.

## Design Details

### How We Manage Request Timeout Today:

Let's revisit how the `timeout` filter works today. Currently, the apiserver
allocates a deadline to the `Context` of each non long-running request inside
the `WithRequestDeadline` filter:
```
		ctx, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()
		
		req = req.WithContext(ctx)
		handler.ServeHTTP(w, req)
```

The duration of this deadline is determined, as follows:

- The client specifies a valid timeout duration in the `timeout` parameter
  of a request, ie. `{path}?timeout=10s`
- In case the client does not specify a timeout, the apiserver resorts to
  a default value which is obtained from the server run option `--request-timeout`

It's also worth noting:
- The apiserver will clamp the user specified timeout duration to `min(user-specified-timeout,  --request-timeout)`
- If a client specifies a zero timeout duration `{path}?timeout=0s`, then the 
  value of `--request-timeout` is used instead.

The `timeout` filter wraps the remaining handler chain and executes it on a new goroutine. So there
are at least two goroutines that are involved:

- a) outer or client-facing: this is the [go routine that net/http creates](https://github.com/golang/go/blob/b8ac61e6e64c92f23d8cf868a92a70d13e20a124/src/net/http/h2_bundle.go#L5833)
     in order to run the request handler:
- b) inner: this is the goroutine that the `timeout` filter creates to
     execute the wrapped handler.

```
a) client <--- net/http (runHandler) <--- timeout filter
                                                 ^
                                                 |
                                                 |
                                       b) timeout filter <--- (receiving channel) <--- inner goroutine
```

The outer goroutine waits via a receiving channel for the inner goroutine to 
finish, and it will wait until the deadline associated with the request
`Context` exceeds. The following gist shows the essence of the `timeout` filter.
```
	receiveCh := make(chan interface{})
	go func() {
		defer func() {
			err := recover()
			receiveCh <- err
		}()
		handler.ServeHTTP(w, req)
	}()
	
	ctx := req.Context()
	select {
	case  <-receiveCh:
		return // the inner goroutine either panicked, or returned normally
	case <-ctx.Done():
		// the request has timed out, return a 504 Status Code to the client
	}
```


### Enabling Per-Request Read/Write Deadline

Primarily, there are three steps:

- a) Enablement must be backed by the feature gate `PerHandlerReadWriteTimeout`
  so that it can be toggled in its entirety.
- b) Enable per-request read/write deadline for each non long-running request.
- c) Do not add the `timeout` filter to the handler chain, since `b` makes it redundant.

> Setting PerHandlerReadWriteTimeout=false must revert both `b` and `c`


We can utilize the deadline associated with the request `Context` to set both 
the read and write timeout for a given non long-running request.

Inside the `WithRequestDeadline` filter, we can enable per-request timeout, as shown below:
```
if utilfeature.DefaultFeatureGate.Enabled(features.PerHandlerReadWriteTimeout) {
	ctrl := http.NewResponseController(w)
	if err := ctrl.SetReadDeadline(deadline); err != nil {
		handleError(w, req, http.StatusInternalServerError, "failed to set read deadline", err)
			return
		}
	if err := ctrl.SetWriteDeadline(deadline); err != nil {
		handleError(w, req, http.StatusInternalServerError, "failed to set write deadline", err)
		return
	}
}
```

Alternatively, we can encapsulate the per-request read/write timeout logic into
its own filter that is wrapped by `WithRequestDeadline`.
```
   handler = WithPerRequestTimeout(handler, ...)
   handler = WithRequestDeadline(handler, ...)
```
For the remainder of this proposal, we are going to refer to `WithPerRequestTimeout`.


> There is no known use case today where there is a need to manipulate the 
> read/write deadline for any given request, so for the initial implementation, we
> are setting the read and write deadline once inside the `WithPerRequestTimeout`
> filter, and not providing any knob for it to be manipulated further down.

This is the [issue](https://github.com/golang/go/issues/54136) that tracks the implementation of `ResponseController`

Let's delve into the inner working of read/write deadline for `http2`:

##### Write Deadline

`SetWriteDeadline` sets up a `time.AfterFunc` [1] with the given deadline, once
the write deadline exceeds the stream is closed [2] with a `stream reset` error,
and the client sees the stream error immediately, although the request handler
on the server might still be running after the timeout happens.

If the request handler on the server continues to write to the underlying
`ResponseWriter` object, the `Write` method will eventually return an
`i/o timeout` error (as soon as the internal buffer of the `ResponseWriter` 
object is full).

The `FlushError` method, on the other hand, is expected to return
an `i/o timeout` error immediately.

[1] https://github.com/golang/go/blob/b8ac61e6e64c92f23d8cf868a92a70d13e20a124/src/net/http/h2_bundle.go#L6570-L6594

[2] https://github.com/golang/go/blob/b8ac61e6e64c92f23d8cf868a92a70d13e20a124/src/net/http/h2_bundle.go#L5691-L5699


##### Read Deadline

`SetReadDeadline` sets up a `time.AfterFunc` [1] with the given deadline, once 
the read deadline exceeds, it closes the request `Body` with
an `i/o timeout` error [2].

> Note that in the case of read deadline exceeding, the stream is not closed with a `stream reset` error


[1] https://github.com/golang/go/blob/b8ac61e6e64c92f23d8cf868a92a70d13e20a124/src/net/http/h2_bundle.go#L6544-L6568

[2] https://github.com/golang/go/blob/b8ac61e6e64c92f23d8cf868a92a70d13e20a124/src/net/http/h2_bundle.go#L5681-L5689


##### Integrate FlushError

The `Flush` method of the underlying `ResponseWriter` object does not return an
`error`. With the introduction of per-request read/write deadline the golang team
has added a new version of `Flush` that returns an `error`, for both http/1x and http/2.0

```
FlushError() error
```

[1] https://github.com/golang/go/blob/b8ac61e6e64c92f23d8cf868a92a70d13e20a124/src/net/http/server.go#L1719C20-L1729

[2] https://github.com/golang/go/blob/b8ac61e6e64c92f23d8cf868a92a70d13e20a124/src/net/http/h2_bundle.go#L6600-L6623


We need to replace the use of `Flush` with `FlushError`:
```
// Flush
flusher, ok := w.(http.Flusher)
flusher.Flush()

// Replace with FlushError
flusher, ok := w.(interface{ FlushError() error })
if err := flusher.FlushError(); err != nil {
	return err
}
```

> With the use of `FlushError`, the request handler can terminate as soon as it 
> receives an error from `FlushError` after the timeout happens.


The following code sites use `Flush` today:
- https://github.com/kubernetes/kubernetes/blob/ba57653b676fa639f86345f2bca20bc9e1bdc8a9/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/watch.go#L194
- https://github.com/kubernetes/kubernetes/blob/3a4c35cc89c0ce132f8f5962ce4b9a48fae77873/staging/src/k8s.io/apiserver/pkg/util/flushwriter/writer.go#L50

> I't worth mentioning that integration of `FlushError` is technically not a
> pre-requisite for the proposed feature


### Termination of the Request Handler

Upon timeout, a request handler can self-terminate by returning the error it receives while:
- reading from the `Body` of the Request
- writing via the `Write` method of the `ResponseWriter` object, or
- flushing using the `FlushError` method


We can wrap the `ResponseWriter`, and the `Body` of the request to panic with an
`http.ErrAbortHandler` error on timeout.
```
type timeoutWriter struct {
	http.ResponseWriter
}

func (tw *timeoutWriter) Write(p []byte) (int, error) {
	n, err := tw.ResponseWriter.Write(p)
	if errors.IsTimeout(err) {
		panic(http.ErrAbortHandler) // alternate: return http.ErrHandlerTimeout
	}
	return n, err
}

func (tw *timeoutWriter) FlushError() error {
	err := tw.ResponseWriter.FlushError()
	if errors.IsTimeout(err) {
		panic(http.ErrAbortHandler) // alternate: return http.ErrHandlerTimeout
	}	
}


type timeoutReader struct {
	io.ReadCloser
}

func (tr *timeoutReader) Read(p []byte) (n int, err error) {
	n, err := tr.ReadCloser.Read(p)
	if errors.IsTimeout(err) {
		panic(http.ErrAbortHandler) // alternate: return http.ErrHandlerTimeout
	}	
	return n, err	
}


// wrap inside the WithPerRequestTimeout filter
func(w http.ResponseWriter, req *http.Request) {
	tr := &timeoutReader{ReadCloser: req.Body}
	req = req.Clone(req.Context())
	req.Body = tr
	w = &baseTimeoutWriter{ResponseWriter: w}
	
	handler.ServeHTTP(req, w)
}
```

We can also use a sentinel error `var ErrRequestTimeOut = errors.New("the request timed out")`
to uniquely identify the cases of request timeout from the other cases that use `http.ErrAbortHandler`.

We can also wrap the `context.Value' of the request context to panic:
```
type PanicAfterTimeoutContext struct {
	context.Context
}

func (c PanicAfterTimeoutContext) Value(key any) any {
	if c.Context.Err() != nil {
		panic(http.ErrAbortHandler)
	}
	return c.Context.Value(key)
}


// inside the handler
func(w http.ResponseWriter, req *http.Request) {
	req = req.WithContext(&PanicAfterTimeoutContext{Context: req.Context()})
	handler.ServeHTTP(w, req)
}
```

We need to mitigate the following issues that comes with wrapping `context.Value`:
- In order to perform post-timeout logging/metrics, we need access to the request
  scoped attributes using the context after the request deadline exceeds.
- A goroutine that is not request scoped (ie. a post-start hook) holds 
  the `context` of a request and invokes the `context.Value` method after the
  deadline exceeds, this would lead the kube-apiserver process to crash.


The panic behavior, if implemented, should have its own feature gate named `PerHandlerReadWriteTimeoutWithPanic`:
| PerHandlerReadWriteTimeout | PerHandlerReadWriteTimeoutWithPanic | |
| ------ | ------ | --------- |
| true | true | a) per request read/write deadline is enabled, and <br> b) inner handler `panics` when ResponseWriter or Context.Value is used after timeout elpases |
| true | false | a) per request read/write deadline is enabled, and <br> b) inner handler `returns error` when ResponseWriter is used after timeout elpases|
| false | true | No change, `PerHandlerReadWriteTimeout` must be enabled for the  proposed panic behavior |
| false | false | This is the default behavior for alpha |


### Convert the Asynchronous Timeout Filter for Mutating-Request to Synchronous:

In addition to the `timeout` filter, the admission and storage handling of 
mutating requests are executed asynchronously in separate goroutines by the
[asynchronous finisher](https://github.com/kubernetes/kubernetes/blob/3a4c35cc89c0ce132f8f5962ce4b9a48fae77873/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/finisher/finisher.go#L87-L122):

- [create](https://github.com/kubernetes/kubernetes/blob/3a4c35cc89c0ce132f8f5962ce4b9a48fae77873/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/create.go#L193-L218)
- [delete](https://github.com/kubernetes/kubernetes/blob/3a4c35cc89c0ce132f8f5962ce4b9a48fae77873/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/delete.go#L128-L132)
- [deletecollection](https://github.com/kubernetes/kubernetes/blob/3a4c35cc89c0ce132f8f5962ce4b9a48fae77873/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/delete.go#L270-L272)
- [patch](https://github.com/kubernetes/kubernetes/blob/3a4c35cc89c0ce132f8f5962ce4b9a48fae77873/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/patch.go#L667-L689)
- [update](https://github.com/kubernetes/kubernetes/blob/3a4c35cc89c0ce132f8f5962ce4b9a48fae77873/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/update.go#L225-L237)

We can replace the asynchronous finisher with one that executes the request
serially in the same goroutine as the invoker, as shown below:
```
func serialFinisher(ctx context.Context, fn ResultFunc) (runtime.Object, error) {
	result := &result{}
	func() {
		// capture the panic here to be rethrown later, this is in
		// keeping with the behavior of the asynchronous finisher
		defer func() {
			if reason := recover(); reason != nil {
				// store the panic reason into the result.
				result.reason = capture(reason)
			}
		}()
		result.object, result.err = fn()
	}()

	return result.Return()
}
```

If the feature gate `PerHandlerReadWriteTimeout` is enabled, we use the serial version.
```
func FinishRequest(ctx context.Context, fn ResultFunc) (runtime.Object, error) {
	if utilfeature.DefaultFeatureGate.Enabled(features.PerHandlerReadWriteTimeout) {
		return serialFinisher(ctx, fn)
	}
	return finishRequest(ctx, fn, postTimeoutLoggerWait, logPostTimeoutResult)
}
```


### Post-Timeout Observability:

Today, we provide the following metrics/logs to track request timeout activity:
||PerHandlerReadWriteTimeout = Disabled|
|-|----------------------------------|
|a)|`request_terminations_total`: this metric is incremented by 1 as soon as the `timeout` filter detects that the `context` associated with the request has [exceeded its deadline](https://github.com/kubernetes/kubernetes/blob/9791f0d1f39f3f1e0796add7833c1059325d5098/staging/src/k8s.io/apiserver/pkg/server/filters/timeout.go#L57).|
|b)|`request_aborts_total`: this metric tracks the number of requests which the apiserver has aborted, including request that has timed out, but the `timeout` filter was not able to write a `504` status code to the client due to the fact that the [inner handler had already written to the `ResponseWriter` object](https://github.com/kubernetes/kubernetes/blob/9791f0d1f39f3f1e0796add7833c1059325d5098/staging/src/k8s.io/apiserver/pkg/server/filters/timeout.go#L262-L276).|
|c)|`request_post_timeout_total`: this metric keeps track of the number of inner goroutines that have terminated after a request had timed out.|
|d)| We [print to the log](https://github.com/kubernetes/kubernetes/blob/9791f0d1f39f3f1e0796add7833c1059325d5098/staging/src/k8s.io/apiserver/pkg/server/filters/timeout.go#L140-L142) how much time it takes the inner goroutine to terminate after the request had timed out:<br>`post-timeout activity - time-elapsed: 2.453442ms, GET "/api/v1/namespaces/default" result: <nil>`|


After a request times out, the asynchronous finisher [spins up a new goroutine](https://github.com/kubernetes/kubernetes/blob/9791f0d1f39f3f1e0796add7833c1059325d5098/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/finisher/finisher.go#L127-L147)
and wait up to `5m` for its inner goroutine to return. Similarly, the timeout
filter [spins up a new goroutine](https://github.com/kubernetes/kubernetes/blob/9791f0d1f39f3f1e0796add7833c1059325d5098/staging/src/k8s.io/apiserver/pkg/server/filters/timeout.go#L124-L144)
, but waits indefinitely for its inner goroutine to return.


*Parity with the Proposed Feature*:
||PerHandlerReadWriteTimeout = Enabled|
|-|----------------------------------|
|a)| `request_terminations_total`: parity is maintained, all labels will exist as is without any change in semantics.|
|b)| `request_aborts_total`: parity is maintained, by throwing the panic `panic(http.ErrAbortHandler)` after a request times out; all labels will exist as is without any change in semantics.|
|c)| `request_post_timeout_total`: this metric will be available, but with the following changes:<ul><li>the label `source` will be dropped: today it has two possible values `{timeout-handler\|rest-handler}` in order to identify whether it is the `timeout` filter or the asynchronous finisher that is reporting this metric. With the proposed feature, this label is not required since the entire request handler will execute in a single goroutine.</li><li>the label `status` will be dropped: today it reports one of these values `{panic\|error\|ok\|pending}`. This label provides no value when the request handler executes on a single goroutine.</li></ul>|
|d)|parity is maintained.|

##### Monitor Post-Timeout Activity using context.AfterFunc:

We keep track of the following information of a request that times out.
```
type PostTimeoutRequestData struct {
	StartedAt      time.Time
	DeadlineAt     time.Time
	RequestURI     string
	AuditID        string
	Verb           string
	
	// the time the request handler returned after the request had timed out
	HandlerReturnedCh  <-chan time.Time
}
```

We maintain a global list of requests that time out, it is bound to a maximum
length allowed so it does not grow infinitely.
```
type PostTimeoutRequestList interface {
	Add(*PostTimeoutRequestData)
}
```

As soon as a request times out, and its handler has not returned yet the
`WithPerRequestTimeout` filter will add it to the globally synchronized list.
This will be accomplished using `context.AfterFunc`.
```
// WithPerRequestTimeout filter
func(w http.ResponseWriter, req *http.Request) {
	handlerDoneCh := make(chan time.Time, 1)
	defer func() {
		handlerDoneCh <- time.Now()
		close(handlerDoneCh)
	}()

	stopFn := context.AfterFunc(req.Context(), func() {
		// this executes in a separate goroutine, so copy to avoid race conditions
		data := &RequestData{StartedAt: started, DeadlineAt: deadline, RequestURI: req.RequestURI, 
			Verb: req.Verb, RequestAuditID: "",
			HandlerReturnedCh: handlerDoneCh}
		
		// add to the global PostTimeoutRequestList
		list.Add(data)
	})
	defer stopFn()
	
	handler.ServeHTTP(w, req)
```


We will setup a `PostStartHook` that will periodically scan the items in the 
list and determine how long a requst handler is active after the timeout had happened.
```
type PostTimeoutRequestListSweeper interface {
	// PostStartHook will call Sweep on a regular interval
	Sweep()
}
```

The `PostStartHook` goroutine will wake up every `5m`, and invoke `Sweep` to
perform the following operations on every item:

- if the `HandlerReturnedCh` channel is closed, it indicates that the request
  handler returned. The entry will be removed from the  list and post-timeout
  activity for this request will be reported in the logs. We will also receive
  the exact time the request handler returned irrespective of when the item is
  scanned, this is in par with what we have today. 
- If an entry is resident for more than `15m`, we report it as hanging "forever"
  and remove it from the list, so the list does not grow infinitely.

> These two intervals can be tweaked as needed

Given that the list is bound to a maximum length allowed, there is a chance that
the list can't accommodate new request that times out.

Add a test to benchmark the `Sweep` method with a list that is full to the maximum
length allowed, so we know the cost of lock contention for a request that is waiting
be to be added to the list. Please note that this applies to requests that timeout only,
for a request that finishes within the allotted time, it never gets added to the list.

Additionally, we can expose a debugging endpoint to the kube-apiserver that 
dumps a snapshot of the post-timeout list.


An alternate to using a global list and a PostStartHook, is to spin up a goroutine
from inside the `context.AfterFunc` function, as shown below:
```
// WithPerRequestTimeout filter
func(w http.ResponseWriter, req *http.Request) {
	handlerDoneCh := make(chan time.Time, 1)
	defer func() {
		handlerDoneCh <- time.Now()
		close(handlerDoneCh)
	}()

	stopFn := context.AfterFunc(req.Context(), func() {
		data := &RequestData{StartedAt: started, DeadlineAt: deadline, 
			RequestURI: req.RequestURI, Verb: req.Verb, RequestAuditID: ""}
		go func() {
			select {
			case returnedAt := <-handlerDoneCh:
				// report post-timeout activity
			case <-time.After(15*time.Minute):
				// report a "forever" hanging request
			}
		}()
	})
	defer stopFn()

	handler.ServeHTTP(w, req)
```


### Audit Log:

The audit filter follows the `timeout` filter in the handler chain.
```
... <-- timeout filter <--- ... <--- authentication <--- audit <--- ...
```

Today, in the course of a request being served, if it times out, we see a 
discrepancy between what the client sees in the response, and the audit log

|Discrepancy in Audit Log|
|----|
|Sequence `A`:<br> a) the inner goroutine had written to the `ResponseWriter` object   (http status code 200)<br> b) the request times out <br> c) the outer goroutine panicks due to `a`, the client sees a `stream reset` or `EOF` error depending on the HTTP protocol<br> d) the inner goroutine finishes its run <br> e) the audit filter stores `Status Code: 200` in the audit logs for this request|
|Sequence `B`:<br> a) the inner goroutine has not written to the `ResponseWriter` object yet <br> b) the request times out <br> c) the outer goroutine successfully writes `504` to the client <br> d) the inner goroutine writes to the `ResponseWriter` object <br> e) the audit filter stores `Status Code: 200` in the audit logs for this request|

With the proposed feature, we have a single goroutine executing the request 
handler, and we can remove the discrepancy in audit, as follows:
- Use a sentinel error `var ErrRequestTimeOut = errors.New("the request timed out")`
  instead of `http.ErrAbortHandler`
- From the delegated `ResponseWriter` object, panic with the sentinel error - `panic(ErrRequestTimeOut)`
- The audit filter can check for this error, and prepare the audit entry appropriately


```
	    defer func() {
			if r := recover(); r != nil {
				defer panic(r)

				if r == ErrRequestTimeOut {
				   event.ResponseStatus = &metav1.Status{
					  Message: ErrRequestTimeOut.Error(),
				}
				return
			}
			processAuditEvent(ctx, sink, event, ...)
		}()
		handler.ServeHTTP(respWriter, req)
```


### Impact on the Client

It's important to understand how the the client is impacted with this
feature enabled. Upon timeout of a request, what the client sees
depends on the following factors:

- a) Protocol of the request, HTTP/1x or HTTP/2.0.
- b) Whether the client specifies a timeout in the `timeout` parameter of the request.
- c) Whether the request handler writes to the `ResponseWriter` object by invoking
  `Write`, before the deadline exceeds.
- d) Whether the request handler invokes `Flush` on the `ResponseWriter` object, 
  preceded  by a `Write`, before the deadline exceeds.

If enabling this feature introduces a change in what the client observes, then
it should be noted and documented with a test. 

Let's outline the current behavior today, with the `timeout` filter enabled:

When a request times out, the client receives a `504 GatewayTimeout` only if the
request handler does not write to the underlying `ResponseWriter` 
object before timeout occurs.

Moreover, if the client sets a timeout value in the request URI
`{host/path}?timeout=10s`, it will yield the same deadline for the request on
both the client and the server. This is the case with `client-go` today.
In this scenario, if a request times out, the client is most likely to receive a
`context deadline exceeded` error due to the fact that the client is most likely
to observe the `Context` of the client request be expired well before the
`504 GatewayTimeout` response from the server makes the round-trip to the client.

Also, the client can not control when or whether the request handler writes to
the `ResponseWriter` object. So, today a client can not deterministically rely
on the `504 GatewayTimeout` error from the server when a request times out.

With the enablement of this feature, and the removal of the `timeout` filter the
apiserver is no longer sending a `504 GatewayTimeout` response when a request 
times out.

Below, for each scenario, we highlight how the server and client behave
differently depending on whether this feature is enabled.

- Feature(*Disabled*): timeout filter is enabled, what we have today
- Feature(*Enabled*): per-handler read/write deadline is in use, `timeout` filter
  is removed - what this KEP proposes

*Scenario `A`*:
- client specifies a timeout in the `timeout` parameter: **No**
- request handler writes to the `ResponseWriter` object before timeout happens: **No**

*Observation*:
|  |Feature(*Disabled*)|Feature(*Enabled*)|
| -- | -------- | ----------- |
| client | a) receives an `http.StatusGatewayTimeout` status code<br> b) reading the `Body` of the `Response` object does not yield any error| a) http/2.0 client receives a stream reset error, on the other hand, http/1x client receives a `local error: tls: bad record MAC` error.
| server | write to the `ResponseWriter` yields an `http: Handler timeout` error immediately | write to the `ResponseWriter` yields an `i/o timeout` error once the underlying buffer is full, so it may take a few `Write` invocations before the handler sees the timeout error.


> Same behavior for both http/1x and http/2.0

---

<br>Scenario `B`:
- client specifies a timeout in the `timeout` parameter: **No**
- request handler writes to the ResponseWriter object before timeout happens: **Yes**

*Observation (with http/2.0)*: 
| |Feature(*Disabled*)|Feature(*Enabled*)|
| -- | -------- | ----------- |
| client | receives an http2 stream reset error, since the outer goroutine in the `timeout` filter panics if the `ResponseWriter` object has already been written to. | same as above (since net/http directly resets the http2 stream)|
| server |write to the `ResponseWriter` yields an `http: Handler timeout` error immediately |write to the `ResponseWriter` yields an `i/o timeout` error once the underlying buffer is full.|

*Observation (with http/1x)*:
| |Feature(*Disabled*)|Feature(*Enabled*)|
| -- | -------- | ----------- |
| client | receives an `EOF` error | receives a `local error: tls: bad record MAC` error
| server | write to the `ResponseWriter` yields an `http: Handler timeout` error immediately| write to the `ResponseWriter` yields an `i/o timeout` error once the underlying buffer is full.

---

<br>Scenario `C`:
- client specifies a timeout in the `timeout` parameter: **No**
- request handler writes to and flushes the `ResponseWriter` object before timeout happens: **Yes**

*Observation (with http2)*:
| |Feature(*Disabled*)|Feature(*Enabled*)|
| -- | -------- | ----------- |
| client | a) receives a response with `http.StatusOK` code <br> b) receives a stream reset error while reading the `Body` of the `Response`| same behavior
| server | write to the `ResponseWriter` yields an `http: Handler timeout` error immediately | write to the `ResponseWriter` yields an `i/o timeout` error once the underlying buffer is full.

*Observation (with http/1x)*:
| |Feature(*Disabled*)|Feature(*Enabled*)|
| -- | -------- | ----------- |
| client | a) receives a response with `http.StatusOK` code <br> b) receives an `unexpected EOF` error while reading the `Body` of the `Response`| a) receives a response with `http.StatusOK` code <br> b) receives a `tls: bad record MAC` error while reading the `Body` of the `Response`
| server | write to the `ResponseWriter` yields an `http: Handler timeout` error immediately | write to the `ResponseWriter` yields an `i/o timeout` error once the underlying buffer is full.

---

<br>Scenario `D`:
- client specifies a timeout in the `timeout` parameter: **Yes**
- request handler writes to or flushes the `ResponseWriter` object before timeout happens: **Yes**

*Observation (with http/2.0)*:
| |Feature(*Disabled*)|Feature(*Enabled*)|
| -- | -------- | ----------- |
| client | a) receives a response with `http.StatusOK` code <br> b) receives an `context deadline exceeded` error while reading the `Body` of the `Response` | same behavior
| server | write to the `ResponseWriter` yields either a `stream closed` or `http: Handler timeout` error. | same behavior


### Test Plan

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.



##### Prerequisite testing updates

##### Unit tests

###### Document net/http Behavior

Add unit tests to document and assert on the behavior of the per-request 
read/write timeout using a bare net/http server. Each of the following
scenarios must be coded to an unit test for both http/1x, and http/2.0:

| | Test Cases|
|--|--|
|a) | write deadline is set, after timeout occurs, we expect the `Write` method of the `ResponseWriter` object to return an `i/o timeout` error once its internal buffer is full.
|b) | timeout occurs before the handler writes to the ResponseWriter object: write deadline is set, let the write timeout occur, invoke `Write` once with small data (`w.Write([]byte("a")`, it is not expected to return error, then invoke `FlushError`, it is expected to return an `i/o timeout` error immediately.
|c) | the handler writes to, but does not flush the `ResponseWriter` object before write timeout occurs: write deadline is set, timeout occurs after the request handler writes to the `ResponseWriter` object using `Write`, but `FlushError` has not been invoked yet.
|d) | write deadline is set, timeout occurs after the request handler writes to and flushes the `ResponseWriter` object, using `Write` and `FlushError` in order.
|e) | write deadline is set, the request handler keeps writing to the the `ResponseWriter` object in a hot loop.
|f) | the client sets up a request body, but never sends any content: read deadline is set, client does not send any content but keeps the stream of the request `Body` open, read timeout is expected.
|g) | read deadline elapses after the handler partially reads the Body of the request: read deadline is set, client sends some content and stops, but keeps the stream of the request `Body` open, read timeout expected.
|h) | does the read deadline have any impact if the request body is empty? read deadline is set, client request `Body` is empty, read timeout should have no effect
|i) | client request has the same deadline as the read/write deadline on the server.
|j) | simulate a slow network by adding delays while the client reads one small chunk at a time fom the `Body` of the response from the server
|k) | does the read/write deadline have any adverse impact on a hijacked connection?
|l) | document whether a connection is reused after a previous request riding on it fails with a write timeout


Each test should assert on the following:
- expected error from `Write` and `FlushError` method from the `ResponseWriter` object after timeout
- expected error from `Read` method of the `Body` of the request
- expected `response` and `err` from the server `client.Do(req)`
- expected error while reading the `Body` of the `http.Response` sent by the server.
- the request handler terminates.

> We need the above tests to understand and documet how per-request 
> read/write deadline behaves at the net/http level. This [PR](https://github.com/kubernetes/kubernetes/pull/122923)
> is tracking these tests

###### Delegated ResponseWriter

The kube-apiserver extends the `ResponseWriter` by delegating it, we have the following delegators:

- audit: https://github.com/kubernetes/kubernetes/blob/548d50da98f086714bebbf54b1cd578d594c7aa6/staging/src/k8s.io/apiserver/pkg/endpoints/filters/audit.go#L218
- latency tracker: https://github.com/kubernetes/kubernetes/blob/548d50da98f086714bebbf54b1cd578d594c7aa6/staging/src/k8s.io/apiserver/pkg/endpoints/filters/webhook_duration.go#L63
- timeout: https://github.com/kubernetes/kubernetes/blob/548d50da98f086714bebbf54b1cd578d594c7aa6/staging/src/k8s.io/apiserver/pkg/server/filters/timeout.go#L151
- httplog: https://github.com/kubernetes/kubernetes/blob/548d50da98f086714bebbf54b1cd578d594c7aa6/staging/src/k8s.io/apiserver/pkg/server/httplog/httplog.go#L58
- metrics: https://github.com/kubernetes/kubernetes/blob/548d50da98f086714bebbf54b1cd578d594c7aa6/staging/src/k8s.io/apiserver/pkg/endpoints/metrics/metrics.go#L795

We need to extend the tests for the delegated `ResponseWriter` objects to
accomplish the following goals:
- Each of the decorator listed above should use a shared test
- The test should verify that the wrapped `ResponseWriter` object produced by the
  decorator maintains interface compatibility with the given `ResponseWriter` it wraps.

This [PR](https://github.com/kubernetes/kubernetes/pull/125093) makes an attempt
to achieve this goal. Interface compatibility implies that the given and the
wrapped `ResponseWriter` objects are compatible in terms of implementation of
the following interfaces -
- http.Flusher
- FlusherError (`FlushError() error`)
- http.CloseNotifier
- http.Hijacker (applicable to http/1x only)

For example, if the given `ResponseWriter` object implements `http.Flusher`, the
wrapped `ResponseWriter` must also implement the said interface even
though it is not interested in overriding or delegating `http.Flusher`.


###### Test the WithPerRequestTimeout Unit

Next step is to write tests to verify whether the `WithPerRequestTimeout`
filter is setting the read/write timeout for the request as expected.
The handler chain should include the following filters:
```
	handler := WithPerRequestTimeout(...)
	handler = WithRequestDeadline(...)
	handler = WithRequestInfo(...)
	handler = withRequestReceivedTimestampWithClock(...)
	handler = WithAuditInit(...)
```	

The Tests must exercise the following combinations:

- request type: WATCH, long-running but not a WATCH, and non long-running 
- The feature `PerHandlerReadWriteTimeout` is enabled, or disabled

The tests should assert on:

- only non long running requests have `Context` deadline and read/write deadline
- the desired deadline is attached to the `Context` of the request
- the desired read/write deadline is attached to the request 
- verify that the request handler returns once the deadline exceeds

###### Test Exercising the Complete Server Handler Chain

Additionally, we must write new unit test(s) that exercise the complete handler
chain of the kube-apiserver:
```
	config := NewConfig(codecs)
	s, err := config.Complete(nil).New("test", NewEmptyDelegate())
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// our test
	}
	s.Handler.NonGoRestfulMux.Handle("/ping", handler)
	
	server := httptest.NewUnstartedServer(s.Handler)
	defer server.Close()
```

These tests must exercise all combinations of the following scenarios:

- a) feature gate: `PerHandlerReadWriteTimeout`: enabled or disabled
- b) protocol: http/1x or http/2.0
- c) client specifies a timeout in the request URI `{host/path}?timeout=10s`: yes, or no
- d) the state of the `ResponseWriter` object: 
  - the request handler neither does a `Write`, not `FlushError` before timeout occurs
  - the request handler does a `Write`, but not `FlushError` before timeout occurs
  - the request handler does a `Write`, followed by a `FlushError` before timeout occurs


Similarly, we should assert on the following, to cover the expectations on 
both the client and the server:

- server: 
  - expected error from `Write` and `FlushError` method from the
    `ResponseWriter` object after timeout
  - that the request handler terminates.
- client:
  - expected  `response` and `err` from `client.Do(req)`
  - expected error while reading the `Body` of the `http.Response` sent by the server.


> These tests will very closely resemble what we would see in production 
> environment since these tests exercise the entire handler chain of the
> kube-apiserver

> We can also use the tests where `PerHandlerReadWriteTimeout == false` to document
> what the client sees today with the timeout filter enabled, and compare with the 
> results from tests where `PerHandlerReadWriteTimeout == true` to understand how 
> differently the client is impacted with this new feature enabled.

> This [PR](https://github.com/kubernetes/kubernetes/pull/124730) documents the 
> current behavior.


###### Request Handler Should Terminate


Additionally, we should have test(s) that verify that a blocked handler does
not block the client, with both the feature enabled/disabled and 
for http/1x and http/2.0.
```
neverDoneCh := make(chan struct{})
func(w http.ResponseWriter, req *http.Request) {
	<-neverDoneCh
}
```

###### Fake ResponseController


Use a fake `ResponseController` to assert on whether the desired read/write
timeout duration is provided, see: https://github.com/golang/go/issues/60229
```
// WithFakeResponseController extends a given httptest.ResponseRecorder object
// with a fake implementation of http.ResonseController.
func WithFakeResponseController(w *httptest.ResponseRecorder) *FakeResponseRecorder {
	return &FakeResponseRecorder{ResponseRecorder: w}
}

type FakeResponseRecorder struct {
	*httptest.ResponseRecorder

	ReadDeadlines  []time.Time
	WriteDeadlines []time.Time
}

func (w *FakeResponseRecorder) SetReadDeadline(deadline time.Time) error {
	w.ReadDeadlines = append(w.ReadDeadlines, deadline)
	return nil
}

func (w *FakeResponseRecorder) SetWriteDeadline(deadline time.Time) error {
	w.WriteDeadlines = append(w.WriteDeadlines, deadline)
	return nil
}
```

##### Integration tests

- Scenario A: `PerHandlerReadWriteTimeout` is disabled, a request times out
- Scenario A: `PerHandlerReadWriteTimeout` is enabled, a request times out

We exercise the above scenarios with both http/1x and http/2.0 using client-go.

Add integration tests for components that use Hijack:
- apimachinery/pkg/util/proxy/upgradeaware.go
- apimachinery/pkg/util/httpstream/spdy/upgrade.go


##### e2e tests

Similar to Integration tests

### Graduation Criteria


#### Alpha

- Feature implemented behind a feature flag
- All new tests enumerated in "Test Plan" are implemented

#### Beta

TBD

#### GA

TBD

#### Deprecation


### Upgrade / Downgrade Strategy

Not applicable

### Version Skew Strategy

Not applicable

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback


###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PerHandlerReadWriteTimeout
  - Components depending on the feature gate:
	  - kube-apiserver

###### Does enabling the feature change any default behavior?

An API client will no longer receive the `504 GatewayTimeout` HTTP status code
when the server times out a request. Instead, the client will receive an `error` 

NOTE: the client can not deterministically rely on the receipt of the `504 GatewayTimeout`
HTTP status code as described in the [Impact on the Client](#impact-on-the-client) section.


###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, through the feature gate 

###### What happens if we reenable the feature if it was previously rolled back?

No additional considerations, except the feature is enabled.

###### Are there any tests for feature enablement/disablement?

There will be both unit tests and integration tests that asserts on:
- the behavior expected when the feature is enabled
- the behavior expected when the feature is disabled



### Rollout, Upgrade and Rollback Planning


###### How can a rollout or rollback fail? Can it impact already running workloads?


###### What specific metrics should inform a rollback?


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?


###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?


### Monitoring Requirements


###### How can an operator determine if the feature is in use by workloads?


###### How can someone using this feature know that it is working for their instance?


- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?


###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?


- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies


###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability


###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting


###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?


###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History


## Drawbacks


## Alternatives


## Infrastructure Needed (Optional)
