---
title: Priority and Fairness for API Server Requests
authors:
  - "@MikeSpreitzer"
  - "@yue9944882"
owning-sig: sig-api-machinery
participating-sigs:
  - wg-multitenancy
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-02-28
last-updated: 2019-02-28
status: provisional
see-also: []
replaces: []
superseded-by: []
---

# Priority and Fairness for API Server Requests

## Table of Contents

Table of Contents
=================

   * [Priority and Fairness for API Server Requests](#priority-and-fairness-for-api-server-requests)
      * [Table of Contents](#table-of-contents)
      * [Release Signoff Checklist](#release-signoff-checklist)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals) 
         * [Future Goals](#future-goals) 
      * [Proposal](#proposal)
         * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
         * [Risks and Mitigations](#risks-and-mitigations)
      * [Design Details](#design-details)
      * [Implementation History](#implementation-history)
      * [Drawbacks](#drawbacks)
      * [Alternatives](#alternatives)
      * [Infrastructure Needed](#infrastructure-needed)


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

This KEP generalizes the existing max-in-flight request handler in the
apiserver to make more distinctions among requests and provide
prioritization and fairness among the categories of requests.  An
outline of the request handling in an apiserver can be found at
https://speakerdeck.com/sttts/kubernetes-api-codebase-tour?slide=18 .

## Motivation

Today the apiserver has a simple mechanism for protectimg itself
against CPU and memory overloads: max-in-flight limits for mutating
and for readonly requests.  Apart from the distinction between
mutating and readonly, no other distinctions are made among requests;
consequently there can be undesirable scenarios where one subset of
the request load crowds out other parts of the request load.  Also,
the dispatching of these requests against two independent limits is
not
[work-conserving](https://en.wikipedia.org/wiki/Work-conserving_scheduler).

### Goals

Following are some bad scenarios that can happen today and which
should be preventable when this KEP is in place.

- Self-Maintenance crowded out.  Some requests are for system
  self-maintenance, such as: node heartbeats, kubelet and kube-proxy
  work on pods, services, secrets, etc involved in the system's
  self-hosting, and leader elections for system controllers.  In an
  overload scenario today there is no assurance of priority for these
  self-maintenance requests.

- Priority Inversions.  In the course of serving request A, there are
  some other requests B spawned --- directly or indirectly.  One
  example is requests B that arrive over a loopback connection ---
  such as requests issued by an admission plugin (e.g., ResourceQuota)
  or any client-ish code in a registry strategy.  Another example is
  requests issued by an external server that itself is serving
  call-outs from an apiserver (e.g., admission web-hooks).  Other
  examples include requests from an aggregated apiserver to create
  TokenReview and SubjectAccessReview objects.  Today it is possible
  that the very load imposed by request A crowds out requests B
  involved in serving A.

- Guaranteed capacity for Low Priority.  There can be thundering herds
  with higher priority running many minutes in the cluster. In order
  to prevent an outage for the normal users connecting the cluster,
  requests with higher priority will not completely starve out the
  whole capacity.

- Garbage Collector crowded out.  The garbage collector should keep up
  with the workload, but in an overload situation today this is not
  assured to happen.

- Deployment of Doom.  We had a situation where a bug in the
  Deployment controller caused it to run amuck under certain
  circumstances, issuing requests in a tight loop.  We would like
  controller bugs to not take the whole system down.

- Kubelet Amuck.  The controller that runs amuck might not be a
  central singleton, it could be a kubelet, kube-proxy, or other
  per-node or otherwise multiplied controller.  In such a situtation
  we would like only the guilty individual to suffer, not all its
  peers and the rest of the system.

- Overbearing or buggy tenants.  In a multi-tenant scenario, we would
  like to prevent some tenants from crowding out the others.  Various
  usage scenarios involve identifying the tenant in the following
  ways.
  
  - Each tenant corresponds with a kube API namespace.

  - Each tenant corresponds with a user name.

  - Each tenant corresponds with a prefix of the user name.

  - Each tenant corresponds with a user's group.  Other groups may
    exist.  There is a subset of the groups that serve to identify
    tenants.  Each user belongs to exactly one of the
    tenant-identifying groups.

This KEP introduces new functionality in apiservers, and it should be
possible to monitor this functionality through Prometheus metrics
available from the apiservers.

This KEP introduces new configuration objects, and they really
matter; it should be easy to apply suitable access controls.

There should be some reasonable defaults.

### Non-Goals

This will be our first cut at a significant area of functionality, and
our goals are deliberately modest.  We want to implement something
useful but limited and get some experience before going further.  Our
precise modesty has not been fully agreed.  Following is an initial
stake in the ground.

- No coordination between apiservers nor with a load balancer is
  attempted.  In this KEP each apiserver independently protects
  itself.  We imagine that later developments may add support for
  informing load balancers about the load state of the apiservers.

- The fairness does not have to be highly precise.  Any rough fairness
  will be good enough.

- WATCH and CONNECT requests are out of scope.  These are of a fairly
  different nature than the others, and their management will be more
  complex.  Also they are arguably less of an observed problem.

- We are only concerned with protection of the CPU and memory of the
  apiserver.  We are not concerned with etcd performance, nor output
  network bandwidth, nor the ability of clients to consume output.

- This KEP will not attempt auto-tuning the capacity limit(s).  Instead
  the administrator will configure each apiserver's capacity limit(s),
  analogously to how the max-in-flight limits are configured today.

- This KEP will not attempt to reproduce the functionality of the
  existing event rate limiting admission plugin.  Events are a
  somewhat special case.  For now we intend to simply leave the
  existing admission plugin in place.

- This KEP will not attempt to protect against denial-of-service
  attacks at lower levels in the stack; it is only about what can be
  done at the identified point in the handler chain.

- This KEP does not introduce threading of additional information
  through webhooks and/or along other paths to support avoidance of
  priority inversions.  While that is an attractive thing to consider
  in the future, this KEP is deliberately limited in its ambition.
  The intent for this KEP is to document that for the case of requests
  that are secondary to some other requests the configuration should
  identify those secondary requests and give them sufficiently high
  priority to avoid priority inversion problems.  That will
  necessarily be conservative, and we settle for that now.

### Future Goals

To recap, there are some issues that we have decided not to address
yet but we think may be interesting to consider in the future.

- Helping load balancers do a better job, considering each apiserver's
  current load state.

- Do something about WATCH and/or CONNECT requests.

- React somehow to etcd overloads.

- Generate information to help something respond to downstream
  congestion.

- Auto-tune the resource limit(s) and/or request cost(s).

- Be more useful for events.

- Thread additional information along the paths needed to enable more
  precisely targeted avoidance of priority inversions.


## Proposal

In short, this proposal is about generalizing the existing
max-in-flight request handler in apiservers to add more discriminating
handling of requests.  The overall approach is that each request is
categorized to a priority level and a queue within that priority
level; each priority level dispatches to its own isolated concurrency
pool; within each priority level queues compete with even fairness.

### Request Categorization

Upon arrival at the handler, a request is assigned to exactly one
_priority level_ and exactly one _flow_ within that priority level.
This is done by matching the request against a configured set of
FlowSchema objects.  This will pick exactly one best matching
FlowSchema, and that FlowSchema will identify a RequestPriority object
and the way to compute the request’s flow identifier.

A RequestPriority object defines a priority level.  Each one is either
_exempt_ or not.  There should be at most one exempt priority level.
Being exempt means that requests of that priority are not subject to
concurrency limits (and thus are never queued) and do not detract from
the concurrency available for non-exempt requests.  In a more
sophisticated system, the exempt priority level would be the highest
priority level.

It is expected that there will be only a few RequestPriority objects.
It is expected that there may be a few tens of FlowSchema objects.  At
one apiserver there may be tens of thousands of flow identifiers seen
close enough in time to have some interaction.

A flow is identified by a pair of strings: the name of the FlowSchema
and a "flow distinguisher" string.  The flow distinguisher is computed
from the request according to a rule that is configured in the
FlowSchema.

Each FlowSchema has:
- A boolean test of an authenticated request;
- A matching priority (default value is 1000);
- A reference to a RequestPriority object; and
- An optional rule for computing the request’s flow distinguisher; not
  allowed for a FlowSchema that refers to a RequestPriority that is
  exempt or has just one queue.

Each RequestPriority has:
- An `exempt` boolean (which defaults to `false`).
- A `catchAll` boolean (which defaults to `false`), which is relevant
  only to default behavior.

Each non-exempt RequestPriority also has:
- A non-negative integer AssuredConcurrencyShares;
- A number of queues; and
- A queue length limit.

Each non-exempt RequestPriority with more than one queue also has:
- A hand size (a small positive number).

The best matching FlowSchema for a given request is one of those whose
boolean test accepts the request.  It is a configuration error if
there is no FlowSchema that matches every request.  In case multiple
schemas accept the request, the best is one of those with the
logically highest matching priority.  In case there are multiple of
those the implementation is free to pick any of those as best.  A
matching priority is an integer, and a numerically lower number
indicates a logically higher priority.

A FlowSchema’s boolean test is constructed from atomic tests.  Each
atomic test compares an authenticated request attribute --- selected
from _either_ the client identity attributes or those that
characterize the work being requested --- with a literal value
(scalar, pattern, or set).  For every available atomic test, its
inverse is also available.  Atomic tests can be ANDed together.  Those
conjunctions can then be ORed together.  The predicate of a FlowSchema
is such a disjunction.

A FlowSchema’s rule for computing the request’s flow distinguisher
identifies a string attribute of the authenticated request and
optionally includes a transformation.  The available string attributes
are (1) namespace of a resource-oriented request (available only if
the predicate accepts only resource-oriented requests) and (2)
username.  If no transformation is indicated then the flow
distinguisher is simply the selected request attribute.  There is only
one transformation available, and it is based on a regex that is
configured in the flow schema and contains a capturing group.  The
transformation consists of doing a complete match against the regex
and extracting submatch number 1; if the selected string does not
match the regex then the transformation yields the empty string.

### Assignment to a Queue

A non-exempt RequestPriority object also has a number of queues (we
are talking about a number here, not the actual set of queues; the
queues exist independently at each apiserver).  If the
RequestPriority’s number of queues is more than one then the following
logic is used to assign a request to a queue.

For a given priority at a given apiserver, each queue is identified by
a numeric index (starting at zero).  A RequestPriority has a hand size
H (so called because the technique here is an application of shuffle
sharding), a small positive number.  When a request arrives at an
apiserver the request flow identifier’s string pair is hashed and the
hash value is used to shuffle the queue indices and deal a hand of
size H, as follows.  We use a hash function that produces at least 64
bits, and 64 of those bits are taken as an unsigned integer we will
call V.  The next step is finding the unique set of integers A[0] in
[0, numQueues), A[1] in [0, numQueues-1), … A[H-1] in
[0, numQueues-(H-1)), A[H] >= 0 such that V = sum[i=0, 1, ...H] A[i] *
ff(numQueues, i), where ff(N, M) is the falling factorial N!/(N-M)!.
The probability distributions of each of these A’s will not be
perfectly even, but we constrain the configuration such that
ff(numQueues, H) is less than 2^60 to keep the unevenness small.  Then
the coefficients A[0], … A[H-1] are converted into queue indices I[0],
… I[H-1] as follows.  I[0] = A[0].  I[1] is the A[1]’th entry in the
list of queue indices excluding I[0].  I[2] is the A[2]’th entry in
the list of queue indices excluding I[0] and I[1].  And so on.

The lengths of the queues identified by I[0], I[1], … I[H-1] are
examined, and the request is put in one of the shortest queues.

For example, if a RequestPriority has numQueues=128 and handSize=6,
the hash value V is converted into 6 unique queue indices plus
3905000064000*A[6].  There are 128 choose 6, which is about 5.4
billion, sets of 6 integers in the range [0,127].  Thus, if there is
one heavy flow and many light flows, the probability of a given light
flow hashing to the same set of 6 queues as the heavy flow is about
one in 5.4 billion.

It is the queues that compete fairly.

### Resource Limits

#### Primary CPU and Memory Protection

This proposal controls both CPU and memory consumption of running
requests by imposing a single concurrency limit per apiserver.  It is
expected that this concurrency limit can be set to a value that
provides effective protection of both CPU and memory while not being
too low for either.

The configuration of an apiserver includes a concurrency limit.  This
is a number, whose units is a number of readonly requests served
concurrently.  Unlike in today's max-in-flight handler, the mutating
and readonly requests are commingled without distinction.  The primary
resource limit applied is that at any moment in time the number of
running non-exempt requests should not exceed the concurrency limit.
Requests of an exempt priority are neither counted nor limited, as in
today's max-in-flight handler.  For the remainder, each server's
overall concurrency limit is divided among those non-exempt priority
levels and each enforces its own limit (independently of the other
levels).

At the first stage of development, an apiserver’s concurrency limit
will be derived from the existing configuration options for
max-mutating-in-flight and max-readonly-in-flight, by taking their
sum.  Later we may migrate to a single direct configuration option.

#### Secondary Memory Protection

A RequestPriority is also configured with a limit on the number of
requests that may be waiting in a given queue.

#### Latency Protection

An apiserver is also configured with a limit on the amount of time
that a request may wait in its queue.  If this time passes while a
request is still waiting for service then the request will be
rejected.

### Queuing

Once a request is categorized and assigned to a queue the next
decision is whether to reject or accept that request.

A request of an exempt priority is never rejected and never waits in a
queue; such a request is dispatched as soon as it arrives.

For queuing requests of non-exempt priority, the first step is to
reject all the requests that have been waiting longer than the
configured limit.  Once that is done, the newly arrived request is
considered.  This request is rejected if and only if the total number
of requests waiting in its queue is at least the configured limit on
that number.

A possible alternative would accept the request unconditionally and,
if that made the queue too long, reject the request at the head of the
queue.  That would be the preferred design if we were confident that
rejection will cause the client to slow down.  Lacking that
confidence, we choose to reject the youngest rather than the oldest
request of the queue, so that an investment in holding a request in a
queue has a chance of eventually getting useful work done.

### Dispatching

Requests of an exempt priority are never held up in a queue; they are
always dispatched immediately.  Following is how the other requests
are dispatched at a given apiserver.

The concurrency limit of an apiserver is divided among the non-exempt
priority levels in proportion to their assured concurrency shares.
This produces the assured concurrency value (ACV) for each non-exempt
priority level:

```
ACV(l) = ceil( SCL * ACS(l) / ( sum[priority levels k] ACS(k) ) )
```

where SCL is the apiserver's concurrency limit and ACS(l) is the
AssuredConcurrencyShares for priority level l.

Dispatching is done independently for each priority level.  Whenever
(1) a non-exempt priority level's number of running requests is below
the level's assured concurrency value and (2) that priority level has
a non-empty queue, it is time to dispatch another request for service.
The Fair Queuing for Server Requests algorithm below is used to pick a
non-empty queue at that priority level.  Then the request at the head
of that queue is dispatched.


#### Fair Queuing for Server Requests

This is based on fair queuing but is modified to deal with serving
requests in an apiserver instead of transmitting packets in a router.
You can find the original fair queuing paper at
[ACM](https://dl.acm.org/citation.cfm?doid=75247.75248) or
[MIT](http://people.csail.mit.edu/imcgraw/links/research/pubs/networks/WFQ.pdf),
and an
[implementation outline at Wikipedia](https://en.wikipedia.org/wiki/Fair_queuing).
Our problem differs from the normal fair queuing problem in three
ways.  One is that we are dispatching requests to be served rather
than packets to be transmitted.  Another difference is that multiple
requests may be served at once.  The third difference is that the
actual service time (i.e., duration) is not known until a request is
done being served.  The first two differences can easily be handled by
straightforward adaptation of the concept called "R(t)" in the
original paper and "virtual time" in the implementation outline.  In
that implementation outline, the notation `now()` is used to mean
reading the _virtual_ clock.  In the original paper’s terms, "R(t)" is
the number of "rounds" that have been completed at real time t, where
a round consists of virtually transmitting one bit from every
non-empty queue in the router (regardless of which queue holds the
packet that is really being transmitted at the moment); in this
conception, a packet is considered to be "in" its queue until the
packet’s transmission is finished.  For our problem, we can define a
round to be giving one nanosecond of CPU to every non-empty queue in
the apiserver (where emptiness is judged based on both queued and
executing requests from that queue), and define R(t) = (server start
time) + (1 ns) * (number of rounds since server start).  Let us write
NEQ(t) for that number of non-empty queues in the apiserver at time t.
For a given queue "q", let us also write "reqs(q, t)" for the number
of requests of that queue at that time.  Let us also write C for the
concurrency limit.  At a particular time t, the partial derivative of
R(t) with respect to t is

```
min(sum[over q] reqs(q, t), C) / NEQ(t) .
```

In terms of the implementation outline, this is the rate at which
virtual time (`now()`) is advancing at time t (in virtual nanoseconds
per real nanosecond).  Where the implementation outline adds packet
size to a virtual time, in our version this corresponds to adding a
service time (i.e., duration) to virtual time.

The third difference is handled by modifying the algorithm to dispatch
based on an initial guess at the request’s service time (duration) and
then make the corresponding adjustments once the request’s actual
service time is known.  This is similar, although not exactly
isomorphic, to the original paper’s adjustment by δ for the sake of
promptness.

For implementation simplicity (see below), let us use the same initial
service time guess for every request; call that duration G.  A good
choice might be the service time limit (1 minute).  Different guesses
will give slightly different dynamics, but any positive number can be
used for G without ruining the long-term behavior.

As in ordinary fair queuing, there is a bound on divergence from the
ideal.  In plain fair queuing the bound is one packet; in our version
it is C requests.

To support efficiently making the necessary adjustments once a
request’s actual service time is known, the virtual finish time of a
request and the last virtual finish time of a queue are not
represented directly but instead computed from queue length, request
position in the queue, and an alternate state variable that holds the
queue’s virtual start time.  While the queue is empty and has no
requests executing: the value of its virtual start time variable is
ignored and its last virtual finish time is considered to be in the
virtual past.  When a request arrives to an empty queue with no
requests executing, the queue’s virtual start time is set to `now()`.
The virtual finish time of request number J in the queue (counting
from J=1 for the head) is J * G + (virtual start time).  While the
queue is non-empty: the last virtual finish time of the queue is the
virtual finish time of the last request in the queue.  While the queue
is empty and has a request executing: the last virtual finish time is
the queue’s virtual start time.  When a request is dequeued for
service the queue’s virtual start time is advanced by G.  When a
request finishes being served, and the actual service time was S, the
queue’s virtual start time is decremented by G - S.

### Example Configuration


For requests from admins and requests in service of other, potentially
system, requests.
```yaml
kind: RequestPriority
meta:
  name: system-top
spec:
  exempt: true
```

For system self-maintenance requests.
```yaml
kind: RequestPriority
meta:
  name: system-high
spec:
  assuredConcurrencyShares: 100
  queues: 128
  handSize: 6
  queueLengthLimit: 100
```

For the garbage collector.
```yaml
kind: RequestPriority
meta:
  name: system-low
spec:
  assuredConcurrencyShares: 30
  queues: 1
  queueLengthLimit: 1000
```

For user requests from kubectl.
```yaml
kind: RequestPriority
meta:
  name: workload-high
spec:
  assuredConcurrencyShares: 30
  queues: 128
  handSize: 6
  queueLengthLimit: 100
```

For requests from controllers processing workload.
```yaml
kind: RequestPriority
meta:
  name: workload-low
spec:
  catchAll: true
  assuredConcurrencyShares: 100
  queues: 128
  handSize: 6
  queueLengthLimit: 100
```

Some flow schemata.

```yaml
kind: FlowSchema
meta:
  name: system-top
spec:
  requestPriority:
    name: system-top
  match:
  - and: # writes by admins (does this cover loopback too?)
    - superSet:
      field: groups
      set: [ "system:masters" ]
```

```yaml
kind: FlowSchema
meta:
  name: system-high
spec:
  requestPriority:
    name: system-high
  flowDistinguisher:
    source: user
    # no transformation in this case
  match:
  - and: # heartbeats by nodes
    - superSet:
      field: groups
      set: [ "system:nodes" ]
    - equals:
      field: resource
      value: nodes
  - and: # kubelet and kube-proxy ops on system objects
    - superSet:
      field: groups
      set: [ "system:nodes" ]
    - equals:
      field: namespace
      value: kube-system
  - and: # leader elections for system controllers
    - patternMatch:
      field: user
      pattern: system:controller:.*
    - inSet:
      field: resource
      set: [ "endpoints", "configmaps", "leases" ]
    - equals:
      field: namespace
      value: kube-system
```

```yaml
kind: FlowSchema
meta:
  name: system-low
spec:
  matchingPriority: 900
  requestPriority:
    name: system-low
  flowDistinguisher:
    source: user
    # no transformation in this case
  match:
  - and: # the garbage collector
    - equals:
      field: user
      value: system:controller:garbage-collector
```

```yaml
kind: FlowSchema
meta:
  name: workload-high
spec:
  requestPriority:
    name: workload-high
  flowDistinguisher:
    source: namespace
    # no transformation in this case
  match:
  - and: # users using kubectl
    - notPatternMatch:
      field: user
      pattern: system:serviceaccount:.*
```

```yaml
kind: FlowSchema
meta:
  name: workload-low
spec:
  matchingPriority: 9999
  requestPriority:
    name: workload-high
  flowDistinguisher:
    source: namespace
    # no transformation in this case
  match:
  - and: [ ] # match everything
```
  
Following is a FlowSchema that might be used for the requests by the
aggregated apiservers of
https://github.com/MikeSpreitzer/kube-examples/tree/add-kos/staging/kos
to create TokenReview and SubjectAccessReview objects.


```
kind: FlowSchema
meta:
  name: system-top
spec:
  matchingPriority: 900
  requestPriority:
    name: system-top
  flowDistinguisher:
    source: user
    # no transformation in this case
  match:
  - and:
    - inSet:
      field: resource
      set: [ "tokenreviews", "subjectaccessreviews" ]
    - superSet:
      field: user
      set: [ "system:serviceaccount:example-com:network-apiserver" ]
```

### Default Behavior

There must be reasonable behavior "out of the box", and it should be
at least a little difficult for an administrator to lock himself out
of this subsystem.  To accomplish these things there are two levels of
defaulting: one concerning behavior, and one concerning explicit API
objects.

The effective configuration is the union of (a) the actual API objects
that exist and (b) implicitly generated backstop objects.  The latter
are not actual API objects, and might not ever exist as identifiable
objects in the implementation, but are figments of our imagination
used to describe the behavior of this subsystem.  These backstop
objects are implicitly present and affecting behavior when needed.
There are two implicitly generated RequestPriority backstop objects.
One is equivalent to the `system-top` object exhibited above, and it
exists while there is no actual RequestPriority object with `exempt ==
true`.  The other is equivalent to the `workload-low` object exhibited
above, and exists while there is no RequestPriority object with
non-exempt priority.  There are also two implicitly generated
FlowSchema backup objects.  Whenever a request whose groups include
`system:masters` is not matched by any actual FlowSchema object, a
backstop equivalent to the `system-top` object exhibited above is
considered to exist.  Whenever a request whose groups do not include
`system:masters` is not matched by any actual FlowSchema object, the
following backstop object is considered to exist.

```yaml
kind: FlowSchema
meta:
  name: non-top-backstop
spec:
  matchingPriority: (doesn’t really matter)
  requestPriority:
    name: (name of an effectively existing RequestPriority, whether
	that is real or backstop, with catchAll==true)
  flowDistinguisher:
    source: user
    # no transformation in this case
  match:
  - and: [ ] # match everything
```

The other part of the defaulting story concerns making actual API
objects exist, and it goes as follows.  Whenever there is no actual
RequestPriority object with `exempt == true`, the RequestPriority
objects exhibited above are created --- except those with a name
already in use by an existing RequestPriority object.  Whenever there
is no actual FlowSchema object that refers to an exempt
RequestPriority object, the schema objects shown above as examples are
generated --- except those with a name already in use.

### Prometheus Metrics

Prior to this KEP, the relevant available metrics from an apiserver are:
- apiserver_current_inflight_requests (gauge, broken down by mutating or not)
- apiserver_longrunning_gauge
- apiserver_request_count (cumulative number served)
- apiserver_request_latencies (histogram)
- apiserver_request_latencies_summary

This KEP adds the following metrics.
- apiserver_rejected_requests (count, broken down by priority, FlowSchema, when (arrival vs timeout))
- apiserver_current_inqueue_requests (gauge, broken down by priority, FlowSchema)
- apiserver_current_executing_requests (gauge, broken down by priority, FlowSchema)
- apiserver_dispatched_requests (count, broken down by priority, FlowSchema)
- apiserver_wait_duration (histogram, broken down by priority, FlowSchema)
- apiserver_service_duration (histogram, broken down by priority, FlowSchema)

### Testing

There should be one or more end-to-end tests that exercise the
functionality introduced by this KEP.  Following are a couple of
suggestions.

One simple test would be to use a client like
https://github.com/MikeSpreitzer/k8api-scaletest/tree/master/cmdriverclosed
to drive workload with more concurrency than is configured to be
admitted, and see whether the amount admitted is as configured.

A similar but more sophisticated test would be like the ConfigMap
driver but would create/update/delete objects that have some
non-trivial behavior associated with them.  One possibility would be
ServiceAccount objects.  Creation of a ServiceAccount object implies
creation of a Secret, and deletion also has an implication.  Thrashing
such objects would test that the workload does not crowd out the
garbage collector.




### Implementation Details/Notes/Constraints

TBD

### Risks and Mitigations

Implementing this KEP will increase the overhead in serving each
request, perhaps to a degree that depends on some measure of system
and/or workload size.  The additional overhead must be practically
limited.

There are likely others.

## Design Details

TBD

## Implementation History

(none yet)

## Drawbacks

Increased burden on operator to provide good configuration.

Increase in stuff to consider when analyzing performance results.

Increased complexity of code to maintain.

Increased runtime costs.

## Alternatives

Once we have settled on a design there will be things to say about the
designs not chosen.

## Infrastructure Needed

The end-to-end test suite should exercise the functionality introduced
by this KEP.  This may require creating a special client to submit an
overload of low-priority work.
