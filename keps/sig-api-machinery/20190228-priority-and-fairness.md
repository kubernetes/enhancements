---
title: Priority and Fairness for API Server Requests
authors:
  - "@MikeSpreitzer"
  - "@yue9944882"
owning-sig: sig-api-machinery
participating-sigs:
  - wg-multitenancy
reviewers:
  - "@deads2k"
  - "@lavalamp"
approvers:
  - "@deads2k"
  - "@lavalamp"
editor: TBD
creation-date: 2019-02-28
last-updated: 2019-02-28
status: implementable
see-also:
replaces:
superseded-by:
---

# Priority and Fairness for API Server Requests

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [Future Goals](#future-goals)
- [Proposal](#proposal)
  - [Request Categorization](#request-categorization)
  - [Assignment to a Queue](#assignment-to-a-queue)
    - [Queue Assignment Proof of Concept](#queue-assignment-proof-of-concept)
    - [Probability of Collisions](#probability-of-collisions)
  - [Resource Limits](#resource-limits)
    - [Primary CPU and Memory Protection](#primary-cpu-and-memory-protection)
    - [Secondary Memory Protection](#secondary-memory-protection)
    - [Latency Protection](#latency-protection)
  - [Queuing](#queuing)
  - [Dispatching](#dispatching)
    - [Fair Queuing for Server Requests](#fair-queuing-for-server-requests)
      - [The original story](#the-original-story)
      - [Re-casting the original story](#re-casting-the-original-story)
      - [From one to many](#from-one-to-many)
      - [From packets to requests](#from-packets-to-requests)
      - [Not knowing service duration up front](#not-knowing-service-duration-up-front)
  - [Example Configuration](#example-configuration)
  - [Reaction to Configuration Changes](#reaction-to-configuration-changes)
  - [Default Behavior](#default-behavior)
  - [Prometheus Metrics](#prometheus-metrics)
  - [Testing](#testing)
  - [Observed Requests](#observed-requests)
    - [Loopback](#loopback)
    - [TokenReview from kube-controller-manager](#tokenreview-from-kube-controller-manager)
    - [SubjectAccessReview by Aggregated API Server](#subjectaccessreview-by-aggregated-api-server)
    - [GET of Custom Resource by Administrator Using Kubectl](#get-of-custom-resource-by-administrator-using-kubectl)
    - [Node to Self](#node-to-self)
    - [Other Leases](#other-leases)
    - [Status Update From System Controller To System Object](#status-update-from-system-controller-to-system-object)
    - [Etcd Operator](#etcd-operator)
    - [Garbage Collectors](#garbage-collectors)
    - [Kube-Scheduler](#kube-scheduler)
    - [Kubelet Updates Pod Status](#kubelet-updates-pod-status)
    - [Controller in Hosted Control Plane](#controller-in-hosted-control-plane)
    - [LOG and EXEC on Workload Pod](#log-and-exec-on-workload-pod)
    - [Requests Over Insecure Port](#requests-over-insecure-port)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [References](#references)
  - [Design Considerations](#design-considerations)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed](#infrastructure-needed)
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
  necessarily be approximate, and we settle for that now.

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
- A matching precedence (default value is 1000);
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
logically highest matching precedence.  In case there are multiple of
those the implementation is free to pick any of those as best.  A
matching precedence is an integer, and a numerically lower number
indicates a logically higher precedence.

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

Since the assignment to queues is based on flows, a useful
configuration will be one in which flows are meaningful boundaries for
confinement/competition.  For bad example, if a particular
FlowSchema's flows are based on usernames and bad behavior correlates
with namespace then the bad behavior will be spread among all the
queues of that schema's priority.  Administrators need to make a good
choice for how flows are distinguished.


#### Queue Assignment Proof of Concept

The following golang code shows a simple recursive technique to
shuffle, deal, and pick.

```go
package main

import (
	"fmt"
	"math"
	"math/rand"
)

var numQueues uint64

func shuffleDealAndPick(v, nq uint64,
	lengthOfQueue func(int) int,
	mr func( /*in [0, nq-1]*/ int) /*in [0, numQueues-1] and excluding previously determined members of I*/ int,
	nRem, minLen, bestIdx int) int {
	if nRem < 1 {
		return bestIdx
	}
	vNext := v / nq
	ai := int(v - nq*vNext)
	ii := mr(ai)
	i := numQueues - nq // i is used only for debug printing
	mrNext := func(a /*in [0, nq-2]*/ int) /*in [0, numQueues-1] and excluding I[0], I[1], ... ii*/ int {
		if a < ai {
			fmt.Printf("mr[%v](%v) going low\n", i, a)
			return mr(a)
		}
		fmt.Printf("mr[%v](%v) going high\n", i, a)
		return mr(a + 1)
	}
	lenI := lengthOfQueue(ii)
	fmt.Printf("Considering A[%v]=%v, I[%v]=%v, qlen[%v]=%v\n\n", i, ai, i, ii, i, lenI)
	if lenI < minLen {
		minLen = lenI
		bestIdx = ii
	}
	return shuffleDealAndPick(vNext, nq-1, lengthOfQueue, mrNext, nRem-1, minLen, bestIdx)
}

func main() {
	numQueues = uint64(128)
	handSize := 6
	hashValue := rand.Uint64()
	queueIndex := shuffleDealAndPick(hashValue, numQueues, func (idx int) int {return idx % 10}, func(i int) int { return i }, handSize, math.MaxInt32, -1)
	fmt.Printf("For V=%v, numQueues=%v, handSize=%v, chosen queue is %v\n", hashValue, numQueues, handSize, queueIndex)
}
```

#### Probability of Collisions

The following code tabulates some probabilities of collisions.
Specifically, if there are `nHands` elephants, `probNextCovered` is
the probability that a random mouse entirely collides with the
elephants.  This is assuming fair dice and independent choices.  This
is not exactly what we have, but is close.

```go
package main

import (
	"fmt"
	"sort"
)

// sum computes the sum of the given slice of numbers
func sum(v []float64) float64 {
	c := append([]float64{}, v...)
	sort.Float64s(c) // to minimize loss of accuracy when summing
	var s float64
	for i := 0; i < len(c); i++ {
		s += c[i]
	}
	return s
}

// choose returns the number of subsets of size m of a set of size n
func choose(n, m int) float64 {
	if m == 0 || m == n {
		return 1
	}
	var ans = float64(n)
	for i := 1; i < m; i++ {
		ans = ans * float64(n-i) / float64(i+1)
	}
	return ans
}

// nthDeal analyzes the result of another shuffle and deal in a series of shuffles and deals.
// Each shuffle and deal randomly picks `handSize` distinct cards from a deck of size `deckSize`.
// Each successive shuffle and deal is independent of previous deals.
// `first` indicates that this is the first shuffle and deal.
// `prevDist[nUnique]` is the probability that the number of unique cards previously dealt is `nUnique`,
// and is unused when `first`.
// `dist[nUnique]` is the probability that the number of unique cards dealt up through this deal is `nUnique`.
// `distSum` is the sum of `dist`, and should be 1.
// `expectedUniques` is the expected value of nUniques at the end of this deal.
// `probNextCovered` is the probability that another shuffle and deal will deal only cards that have already been dealt.
func nthDeal(first bool, handSize, deckSize int, prevDist []float64) (dist []float64, distSum, expectedUniques, probNextCovered float64) {
	dist = make([]float64, deckSize+1)
	expects := make([]float64, deckSize+1)
	nexts := make([]float64, deckSize+1)
	if first {
		dist[handSize] = 1
		expects[handSize] = float64(handSize)
		nexts[handSize] = 1 / choose(deckSize, handSize)
	} else {
		for nUnique := handSize; nUnique <= deckSize; nUnique++ {
			conts := make([]float64, handSize+1)
			for news := 0; news <= handSize; news++ {
				// one way to get to nUnique is for `news` new uniques to appear in this deal,
				// and all the previous deals to have dealt nUnique-news unique cards.
				prevUnique := nUnique - news
				ways := choose(deckSize-prevUnique, news) * choose(prevUnique, handSize-news)
				conts[news] = ways * prevDist[prevUnique]
				//fmt.Printf("nUnique=%v, news=%v, ways=%v\n", nUnique, news, ways)
			}
			dist[nUnique] = sum(conts) / choose(deckSize, handSize)
			expects[nUnique] = dist[nUnique] * float64(nUnique)
			nexts[nUnique] = dist[nUnique] * choose(nUnique, handSize) / choose(deckSize, handSize)
		}

	}
	return dist, sum(dist), sum(expects), sum(nexts)
}

func main() {
	handSize := 7
	deckSize := 256
	fmt.Printf("choose(%v, %v) = %v\n", deckSize, handSize, choose(deckSize, handSize))
	var dist []float64
	var probNextCovered float64
	for nHands := 1; probNextCovered < 0.01; nHands++ {
		var distSum, expected float64
		dist, distSum, expected, probNextCovered = nthDeal(nHands == 1, handSize, deckSize, dist)
		fmt.Printf("After %v hands, distSum=%v, expected=%v, probNextCovered=%v, dist=%v\n", nHands, distSum, expected, probNextCovered, dist)
	}
}
```


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
Even later we intend to automatomatically tune the setting of an
apiserver's concurrency limit.

#### Secondary Memory Protection

A RequestPriority is also configured with a limit on the number of
requests that may be waiting in a given queue.

#### Latency Protection

An apiserver is also configured with a limit on the amount of time
that a request may wait in its queue.  If this time passes while a
request is still waiting for service then the request will be
rejected.

This may mean we need to revisit the scalability tests --- this
protection could keep us from violating latency SLOs even though we
are dropping many requests.

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
done being served.

To show how to cope with those differences, we start by re-casting the
original fair queuing story in a way that is more amenable to the
needed changes and has the same implementation.  Then we work through
a series of three changes.

##### The original story

We have a collection of FIFO queues of packets to be transmitted, one
packet at a time, over one link that goes at a fixed rate of
`mu_single` (quantified in bits/second).  We imagine a virtual world
called bit-by-bit round-robin (BR), in which the transmitter serves
the queues in a round-robin fashion but advances from one queue to the
next not on packet boundaries but rather on bit boundaries.  We define
a function of time R(t) that counts the number of "rounds" completed,
where a round is sending one bit from each non-empty queue.  Thus R(t)
is

```
Integral[from tau=start_of_story to tau=t] (
    if NAC(tau) > 0 then mu_single / NAC(tau) else 0) dtau
```

where `NAC(a time)` is the number of "active" or "non-empty" queues at
that time, defined precisely below.

We define start and finish R values for each packet:

```
S(i,j) = max(F(i,j-1), R(t_arrive(i,j)))
F(i,j) = S(i,j) + len(i,j) 
```

where:
- `S(i,j)` is the R value for the start of the j'th packet of queue i,
- `F(i,j)` is the R value for the finish of the j'th packet of queue i,
- `len(i,j)` is the number of bits in that packet, and
- `t_arrive(i,j)` is the time when that packet arrived.

We define `NAC(t)` to be the number of queues `i` for which `R(t) <=
F(i,j)` where `j` is the last packet of queue `i` to arrive at or
before `t`.

When it is time to start transmitting the next packet in the real
world, we pick the unsent packet with the smallest `F` value.  If
there is a tie then we pick the one whose queue follows most closely
in round-robin order the queue last picked from.

It is not necessary to explicitly represent the S and F values for
every unsent packet in the virtual world.  It suffices to track R(t)
and, for each queue, the contents of that queue and the S of the
queue's oldest unsent packet.  This is enough to make the cost of
reacting to a packet arrival or completion O(1) with the exception of
finding the next packet to send.  That costs O(num queues) in a
straightforward implementation and O(log(num queues) + (num ties)) if
the queues are kept in a data structure sorted by the F value of the
queue's oldest packet that has not started transmission.

##### Re-casting the original story

The original fair queuing technique can be understood in the following
way, which corresponds more directly to max-min fairness.  We have a
collection of FIFO queues of packets to be transmitted, one packet at
a time, over one link that goes at a fixed rate of `mu_single` (in
units of bits/second).  We imagine a virtual world called concurrent
service (CS), in which transmission is proceeding for all the
non-empty queues concurrently; queue `i` is transmitted at rate
`mu(i,t)` at time `t`.  At any given `t` the allocations are given by

```
mu(i,t) = min(rho(i,t), mu_fair(t))
```

where:
- `i` identifies a queue,
- `rho(i,t)` is the rate requested by queue `i` at time `t` and is defined to be
  the product of `mu_single` and the number of packets of that queue
  that are unsent in the virtual world at time `t`,
- `mu_fair(t)` is the smallest non-negative quantity that solves the equation
  ```
  min(mu_single, Sum[over i] rho(i,t)) = Sum[over i] min(rho(i,t), mu_fair(t))
  ```

That implies that `mu(i,t)` is zero for each empty queue and
`mu_single / (number of non-empty queues at time t)` for each
non-empty queue, where emptiness is judged in the virtual world.

In the virtual world bits are transmitted from queue `i` at rate
`mu(i,t)`, for every non-empty queue `i`.  Each time when a queue
transitions from empty to non-empty is when bits start being sent from
that queue in the virtual world; the solution `mu_fair(t)` gets
adjusted at this time, among others.  In this virtual world a queue's
packets are divided into three subsets: those that have been
completely sent, those that are in the process of being sent, and
those that have not yet started being sent.  That number being sent is
1 unless it is 0 and the queue has no unsent packets.  Unlike the
original fantasy, this virtual world uses the same clock as the real
world.  Whenever a packet finishes being sent in the real world, the
next packet to be transmitted is the unsent one that will finish being
sent soonest in the virtual world.  If there is a tie among several,
we pick the one whose queue is next in round-robin order (following
the queue last picked).

We can define beginning and end times (B and E) for transmission of
the j'th packet of queue i in the virtual world, with the following
equations.

```
B(i,j) = max(E(i,j-1), t_arrive(i,j))
Integral[from tau=B(i,j) to tau=E(i,j)] mu(i,tau) dtau = len(i,j)
```

This has a practical advantage over the original story: the integrals
are only over the lifetime of a single request's service --- rather
than over the lifetime of the server.  This makes it easier to use
floating point representations with sufficient precision.

Note that computing an E value before it has arrived requires
predicting the course of `mu(i,t)` from now until E arrives.  However,
because all the non-empty queues have the same value for `mu(i,t)`
(i.e., `mu_fair(t)`), we can safely make whatever assumption we want
without distorting the dispatching choice --- all non-empty queues are
affected equally, so even wildly wrong guesses don't change the ordering.

The correspondence with the original telling of the fair queuing story
goes as follows.  Equate

```
S(i,j) = R(B(i,j))
F(i,j) = R(E(i,j))
```

The recurrence relations for S and F correspond to the recurrence
relations for B and E.

```
S(i,j) = R(B(i,j))
       = R(max(E(i,j-1), t_arrive(i,j)))
       = max(R(E(i,j-1)), R(t_arrive(i,j)))
       = max(F(i,j-1), R(t_arrive(i,j)))
```

(R and max commute because both are monotonically non-decreasing).

Note that `mu_fair(t)` is exactly the same as `dR/dt` in the original
story.  So we can reason as follows.

```
Integral[tau=B(i,j) to tau=E(i,j)] mu(i,tau) dtau = len(i,j)

Integral[tau=start to tau=E(i,j)] (dR/dt)(tau) dtau -
Integral[tau=start to tau=B(i,j)] (dR/dt)(tau) dtau
= len(i,j)

R(E(i,j)) - R(B(i,j)) = len(i,j)

F(i,j) - S(i,j) = len(i,j)

F(i,j) = S(i,j) + len(i,j)
```

It is not necessary to track the B and E values for every unsent
packet in the virtual world.  It suffices to track, for each queue
`i`, the following things in the virtual world:
- the contents of the queue
- B of the packet currently being sent
- bits of that packet sent so far: `Integral[from tau=B to tau=now] mu(i,tau) dtau`

At any point in time the E of the packet being sent can be calculated,
under the assumption that `mu(i,t)` will henceforth be constant, by
adding to the current time the quotient of (remaining bits to send) /
`mu(i,t)`.  The E of the packet after that can be calculated by
further adding the quotient (size of packet) / `mu(i,t)`.  This E is
the value used to pick the next packet to send in the real world.

If we are satisfied with an O(num queues) cost to react to advancing
the clock or picking a request to dispatch then a direct
implementation of the above works.

It is possible to reduce those costs to O(log(num queues)) by
leveraging the above correspondence to work with R values and keep the
queues in a data structure sorted by F of the next packet to transmit
in the virtual world.  There will be two classes of queues: those that
do not and those that do have a packet waiting to start transmission
in the virtual world.  It is the latter class that is kept in a data
structure sorted by the F of that packet.  Such a packet's F does not
change as the system evolves over time, so an incremental step
requires only manipulating the queue and packet directly involved.

##### From one to many

The first change is to suppose that transmission is being done on
multiple parallel links.  Call the number of them `C`.  This allows
`mu_fair` to be higher.  Its constraint changes to

```
min(C*mu_single, Sum[over i] rho_i) = Sum[over i] min(rho_i, mu_fair)
```

Because we now have more possible values for `mu(i,t)` than 0 and
`mu_fair(t)`, it is more computationally complex to adjust the
`mu(i,t)` values when a packet arrives or completes virtual service.
That complexity is:
- O(n log n), where n is the number of queues,
  in a straightforward implementation;
- O(log n) if the queues are kept in a data structure sorted by `rho(i,t)`.

We can keep the same virtual transmission scheduling scheme as in the
single-link world --- that is, each queue sends one packet at a time
in the virtual world.  This looks a little unreal but that is
immaterial and keeps the logic simple; we can use the same B and E
equations.

However, the greater diversity of `mu(i,t)` values breaks the
correspondence with the original story.  We can still define `R(t) =
Integral[from tau=start to tau=t] mu_fair(tau) dtau`.  However,
because sometimes some queues get a rate that is less than
`mu_fair(t)`, it is not necessarily true that `R(E(i,j)) - R(B(i,j)) =
len(i,j)`.  Because all non-empty queues do not necessarily get the
rate `mu_fair(t)`, the prediction of that affects the dispatching
choice.  This ruins the simple story about how to get logarithmic cost
for dispatching.

We can partially recover by dividing queues into three classes rather
than two: empty queues, those for which `rho(i,t) <= mu_fair(t)`, and
those for which `mu_fair(t) < rho(i,t)`.  We can efficiently make a
dispatching choice from each of the two latter classes of queues,
under the assumption that `mu_fair(t)` will be henceforth constant,
and then efficiently choose between those two choices.  However, it is
also necessary to react to a stimulus that modifies `mu_fair(t)` or
some `rho(i,t)` so that some queues move between classes --- and this
costs O(m log n), where m is the number of queues moved and n is the
number of queues in the larger class.  The details are as follows.

For queues where `rho(i,t) <= mu_fair(t)` we can keep track of the
predicted E for the next packet to begin transmitting.  As long as
that queue's `rho` remains less than or equal to `mu_fair`, these E
predictions do not change.  We can keep these queues in a data
structure sorted by those E predictions.

For queues `i` where `mu_fair(t) <= rho(i,t)` we can keep track of the
F (that is, `R(E)`) of the next packet to begin transmitting.  As long
as `mu_fair` remains less than or equal to that queue's `rho`, that F
does not change.  We can keep these queues in a data structure sorted
by those F predictions.

When a stimulus --- that is, packet arrival or virtual completion ---
changes `mu_fair(t)` or some `rho(i,t)` in such a way that some queues
move between classes, those queues get removed from their old class
data structure and added to the new one.

When it comes time to choose a packet to begin transmitting in the
real world, we start by choosing the best packet from each non-empty
class of non-empty queues.  Supposing that gives us two packets, we
have to compare the E of one packet with the F of the other.  This is
done by assuming that `mu_fair` will not change henceforth.  Finally,
we may have to break a tie; that is done by round-robin ordering, as
usual.

##### From packets to requests

The next change is from transmitting packets to serving requests.  We
no longer have a collection of C links; instead we have a server
willing to serve up to C requests at once.  Instead of a packet with a
length measured in bits, we have a request with a service duration
measured in seconds.  The units change: `mu_single` and `mu_i` are no
longer in bits per second but rather are in service-seconds per
second; we call this unit "seats" for short.  We now say `mu_single`
is 1 seat.  As before, when it is time to dispatch the next request in
the real world we pick one that would finish soonest in the virtual
world, using round-robin ordering to break ties.

##### Not knowing service duration up front

The final change removes the up-front knowledge of the service
duration of a request.  Instead, we use a guess `G`.  When a request
finishes execution in the real world, we learn its actual service
duration `D`.  At this point we adjust the explicitly represented B
and E (and F, if using those) values of following requests in that
queue to account for the difference `D - G`.

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

### Reaction to Configuration Changes

We do not seek to make our life easy by making any configuration
object fields immutable.  Recall that objects can be deleted and
replacements (with the same name) created, so server-enforced
immutability of field values provides no useful guarantee to
downstream consumers (such as the request filter here).  We could try
to get useful guarantees by adding a finalizer per (config object,
apiserver), but at this level of development we will not attempt that.

Many details of the configuration are consumed only momentarily;
changes in these pose no difficulty.  Challenging changes include
changes in the number of queues, the queue length limit, or the
assured concurrency value (which is derived from several pieces of
config, as outlined elsewhere) of a request priority --- as well as
deletion of a priority level itself.

An increase in the number of queues of a priority level is handled by
simply adding queues.  A decrease is handled by making a distinction
between the desired and the actual number of queues.  When the desired
number drops below the actual number, the undesired queues are left in
place until they are naturally drained; new requests are put in only
the desired queues.  When an undesired queue becomes empty it is
deleted and the fair queuing round-robin pointer is advanced if it was
pointing to that queue.

When the assured concurrency value of a priority level increases,
additional requests are dispatched if possible.  When the assured
concurrency value decreases, there is no immediate reaction --- this
filter does not abort or suspend requests that are currently being
served.

When the queue length limit of a priority level increases, no
immediate reaction is required.  When the queue length limit
decreases, there is also no immediate reaction --- queues that are
longer than the new length limit are left to naturally shrink as they
are drained by dispatching and timeouts.

When a request priority configuration object is deleted, in a given
apiserver the corresponding implementation objects linger until all
the queues of that priority level are empty.  A FlowSchema associated
with one of these lingering undesired priority levels matches no
requests.

The [Dispatching](#Dispatching) section prescribes how the assured
concurrency value (`ACV`) is computed for each priority level, and the
sum there is over all the _desired_ priority levels (i.e., excluding
the lingering undesired priority levels).  For this reason and for
others, at any given time this may compute for some priority level(s)
an assured concurrency value that is lower than the number currently
executing.  In these situations the total number allowed to execute
will temporarily exceed the apiserver's configured concurrency limit
(`SCL`) and will settle down to the configured limit as requests
complete their service.

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
- apiserver_request_queue_length (histogram, broken down by
  RequestPriority name; buckets set at 0, 0.25, 0.5, 0.75, 0.9, 1.0
  times the relevant queue length limit)
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


### Observed Requests

To provide data about requests to use in designing the configuration,
here are some observations from a running system late in the
release 1.16 development cycle.  These are extracts from the
kube-apiserver log file, with linebreaks and indentation added for
readability.

The displayed data are a `RequestInfo` from
`k8s.io/apiserver/pkg/endpoints/request` and an `Info` from
`k8s.io/apiserver/pkg/authentication/user`.

#### Loopback

```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/apis/admissionregistration.k8s.io/v1beta1/mutatingwebhookconfigurations",
  Verb:"list", APIPrefix:"apis",
  APIGroup:"admissionregistration.k8s.io", APIVersion:"v1beta1",
  Namespace:"", Resource:"mutatingwebhookconfigurations",
  Subresource:"", Name:"",
  Parts:[]string{"mutatingwebhookconfigurations"}},
userInfo=&user.DefaultInfo{Name:"system:apiserver",
  UID:"388b748d-481c-4348-9c94-b7aab0c6efad",
  Groups:[]string{"system:masters"}, Extra:map[string][]string(nil)}
```

```
RequestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/api/v1/services", Verb:"watch", APIPrefix:"api", APIGroup:"",
  APIVersion:"v1", Namespace:"", Resource:"services", Subresource:"",
  Name:"", Parts:[]string{"services"}},
user.Info=&user.DefaultInfo{Name:"system:apiserver",
  UID:"388b748d-481c-4348-9c94-b7aab0c6efad",
  Groups:[]string{"system:masters"}, Extra:map[string][]string(nil)}
```

```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/api/v1/namespaces/default/services/kubernetes", Verb:"get",
  APIPrefix:"api", APIGroup:"", APIVersion:"v1", Namespace:"default",
  Resource:"services", Subresource:"", Name:"kubernetes",
  Parts:[]string{"services", "kubernetes"}},
userInfo=&user.DefaultInfo{Name:"system:apiserver",
  UID:"388b748d-481c-4348-9c94-b7aab0c6efad",
  Groups:[]string{"system:masters"}, Extra:map[string][]string(nil)}
  ```

#### TokenReview from kube-controller-manager

```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/apis/authentication.k8s.io/v1/tokenreviews", Verb:"create",
  APIPrefix:"apis", APIGroup:"authentication.k8s.io", APIVersion:"v1",
  Namespace:"", Resource:"tokenreviews", Subresource:"", Name:"",
  Parts:[]string{"tokenreviews"}},
userInfo=&user.DefaultInfo{Name:"system:kube-controller-manager",
  UID:"", Groups:[]string{"system:authenticated"},
  Extra:map[string][]string(nil)}
```

#### SubjectAccessReview by Aggregated API Server

```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/apis/authorization.k8s.io/v1beta1/subjectaccessreviews",
  Verb:"create", APIPrefix:"apis", APIGroup:"authorization.k8s.io",
  APIVersion:"v1beta1", Namespace:"", Resource:"subjectaccessreviews",
  Subresource:"", Name:"", Parts:[]string{"subjectaccessreviews"}},
userInfo=&user.DefaultInfo{Name:"system:serviceaccount:example-com:network-apiserver",
  UID:"55aa7599-67e7-4f29-80ae-d8c2cb5fd0c4",
  Groups:[]string{"system:serviceaccounts",
    "system:serviceaccounts:example-com", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

#### GET of Custom Resource by Administrator Using Kubectl

```
requestInfo=&request.RequestInfo{IsResourceRequest:false,
  Path:"/openapi/v2", Verb:"get", APIPrefix:"", APIGroup:"",
  APIVersion:"", Namespace:"", Resource:"", Subresource:"", Name:"",
  Parts:[]string(nil)},
userInfo=&user.DefaultInfo{Name:"system:admin", UID:"",
  Groups:[]string{"system:masters", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/apis/network.example.com/v1alpha1/namespaces/default/networkattachments",
  Verb:"list", APIPrefix:"apis", APIGroup:"network.example.com",
  APIVersion:"v1alpha1", Namespace:"default",
  Resource:"networkattachments", Subresource:"", Name:"",
  Parts:[]string{"networkattachments"}},
userInfo=&user.DefaultInfo{Name:"system:admin", UID:"",
  Groups:[]string{"system:masters", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

#### Node to Self

```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/api/v1/nodes/127.0.0.1/status", Verb:"patch", APIPrefix:"api",
  APIGroup:"", APIVersion:"v1", Namespace:"", Resource:"nodes",
  Subresource:"status", Name:"127.0.0.1", Parts:[]string{"nodes",
    "127.0.0.1", "status"}},
userInfo=&user.DefaultInfo{Name:"system:node:127.0.0.1", UID:"",
  Groups:[]string{"system:nodes", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/apis/coordination.k8s.io/v1/namespaces/kube-node-lease/leases/127.0.0.1",
  Verb:"update", APIPrefix:"apis", APIGroup:"coordination.k8s.io",
  APIVersion:"v1", Namespace:"kube-node-lease", Resource:"leases",
  Subresource:"", Name:"127.0.0.1", Parts:[]string{"leases",
    "127.0.0.1"}},
userInfo=&user.DefaultInfo{Name:"system:node:127.0.0.1", UID:"",
  Groups:[]string{"system:nodes", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

#### Other Leases

LIST by kube-controller-manager
```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/apis/coordination.k8s.io/v1/leases", Verb:"list",
  APIPrefix:"apis", APIGroup:"coordination.k8s.io", APIVersion:"v1",
  Namespace:"", Resource:"leases", Subresource:"", Name:"",
  Parts:[]string{"leases"}},
userInfo=&user.DefaultInfo{Name:"system:kube-controller-manager",
  UID:"", Groups:[]string{"system:authenticated"},
  Extra:map[string][]string(nil)}
```

WATCH by kube-controller-manager
```
RequestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/apis/coordination.k8s.io/v1beta1/leases", Verb:"watch",
  APIPrefix:"apis", APIGroup:"coordination.k8s.io",
  APIVersion:"v1beta1", Namespace:"", Resource:"leases", Subresource:"",
  Name:"", Parts:[]string{"leases"}},
user.Info=&user.DefaultInfo{Name:"system:kube-controller-manager",
  UID:"", Groups:[]string{"system:authenticated"},
  Extra:map[string][]string(nil)}
```

#### Status Update From System Controller To System Object

Deployment controller
```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/apis/apps/v1/namespaces/kube-system/deployments/kube-dns/status",
  Verb:"update", APIPrefix:"apis", APIGroup:"apps", APIVersion:"v1",
  Namespace:"kube-system", Resource:"deployments", Subresource:"status",
  Name:"kube-dns", Parts:[]string{"deployments", "kube-dns", "status"}},
userInfo=&user.DefaultInfo{Name:"system:serviceaccount:kube-system:deployment-controller",
  UID:"2b126368-be77-454d-8893-0384f9895f02",
  Groups:[]string{"system:serviceaccounts",
    "system:serviceaccounts:kube-system", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

#### Etcd Operator

List relevant pods
```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/api/v1/namespaces/example-com/pods", Verb:"list",
  APIPrefix:"api", APIGroup:"", APIVersion:"v1",
  Namespace:"example-com", Resource:"pods", Subresource:"", Name:"",
  Parts:[]string{"pods"}},
userInfo=&user.DefaultInfo{Name:"system:serviceaccount:example-com:default",
  UID:"6c906aa7-135c-4094-8611-fbdb1a8ea077",
  Groups:[]string{"system:serviceaccounts",
    "system:serviceaccounts:example-com", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

Update an etcd cluster
```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/apis/etcd.database.coreos.com/v1beta2/namespaces/example-com/etcdclusters/the-etcd-cluster",
  Verb:"update", APIPrefix:"apis", APIGroup:"etcd.database.coreos.com",
  APIVersion:"v1beta2", Namespace:"example-com",
  Resource:"etcdclusters", Subresource:"", Name:"the-etcd-cluster",
  Parts:[]string{"etcdclusters", "the-etcd-cluster"}},
userInfo=&user.DefaultInfo{Name:"system:serviceaccount:example-com:default",
  UID:"6c906aa7-135c-4094-8611-fbdb1a8ea077",
  Groups:[]string{"system:serviceaccounts",
    "system:serviceaccounts:example-com", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

Kube-scheduler places an etcd Pod
```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/api/v1/namespaces/example-com/pods/the-etcd-cluster-mxcxvgbcfg/binding",
  Verb:"create", APIPrefix:"api", APIGroup:"", APIVersion:"v1",
  Namespace:"example-com", Resource:"pods", Subresource:"binding",
  Name:"the-etcd-cluster-mxcxvgbcfg", Parts:[]string{"pods",
    "the-etcd-cluster-mxcxvgbcfg", "binding"}},
userInfo=&user.DefaultInfo{Name:"system:kube-scheduler", UID:"",
  Groups:[]string{"system:authenticated"},
  Extra:map[string][]string(nil)}
```

#### Garbage Collectors

Pod GC, list nodes.
```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/api/v1/nodes", Verb:"list", APIPrefix:"api", APIGroup:"",
  APIVersion:"v1", Namespace:"", Resource:"nodes", Subresource:"",
  Name:"", Parts:[]string{"nodes"}},
userInfo=&user.DefaultInfo{Name:"system:serviceaccount:kube-system:pod-garbage-collector",
  UID:"85b105c6-ca2f-42f0-8a75-1da13a8c1f6d",
  Groups:[]string{"system:serviceaccounts",
    "system:serviceaccounts:kube-system", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

Generic GC, `GET /api`
```
requestInfo=&request.RequestInfo{IsResourceRequest:false, Path:"/api",
  Verb:"get", APIPrefix:"", APIGroup:"", APIVersion:"", Namespace:"",
  Resource:"", Subresource:"", Name:"", Parts:[]string(nil)},
userInfo=&user.DefaultInfo{Name:"system:serviceaccount:kube-system:generic-garbage-collector",
  UID:"61271d97-a959-467a-afd5-892dfd30dec9",
  Groups:[]string{"system:serviceaccounts",
    "system:serviceaccounts:kube-system", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

Generic GC, `GET /apis/coordination.k8s.io/v1beta1`
```
requestInfo=&request.RequestInfo{IsResourceRequest:false,
  Path:"/apis/coordination.k8s.io/v1beta1", Verb:"get",
  APIPrefix:"apis", APIGroup:"", APIVersion:"", Namespace:"",
  Resource:"", Subresource:"", Name:"", Parts:[]string(nil)},
userInfo=&user.DefaultInfo{Name:"system:serviceaccount:kube-system:generic-garbage-collector",
  UID:"61271d97-a959-467a-afd5-892dfd30dec9",
  Groups:[]string{"system:serviceaccounts",
    "system:serviceaccounts:kube-system", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

#### Kube-Scheduler

LIST
```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/apis/storage.k8s.io/v1/storageclasses", Verb:"list",
  APIPrefix:"apis", APIGroup:"storage.k8s.io", APIVersion:"v1",
  Namespace:"", Resource:"storageclasses", Subresource:"", Name:"",
  Parts:[]string{"storageclasses"}},
userInfo=&user.DefaultInfo{Name:"system:kube-scheduler", UID:"",
  Groups:[]string{"system:authenticated"},
  Extra:map[string][]string(nil)}
```

WATCH
```
RequestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/api/v1/services", Verb:"watch", APIPrefix:"api", APIGroup:"",
  APIVersion:"v1", Namespace:"", Resource:"services", Subresource:"",
  Name:"", Parts:[]string{"services"}},
user.Info=&user.DefaultInfo{Name:"system:kube-scheduler", UID:"",
  Groups:[]string{"system:authenticated"},
  Extra:map[string][]string(nil)}
```

Bind Pod of Hosted Control Plane
```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/api/v1/namespaces/example-com/pods/the-etcd-cluster-mxcxvgbcfg/binding",
  Verb:"create", APIPrefix:"api", APIGroup:"", APIVersion:"v1",
  Namespace:"example-com", Resource:"pods", Subresource:"binding",
  Name:"the-etcd-cluster-mxcxvgbcfg", Parts:[]string{"pods",
    "the-etcd-cluster-mxcxvgbcfg", "binding"}},
userInfo=&user.DefaultInfo{Name:"system:kube-scheduler", UID:"",
  Groups:[]string{"system:authenticated"},
  Extra:map[string][]string(nil)}
```

Update Status of System Pod
```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/api/v1/namespaces/kube-system/pods/kube-dns-5f7bc9fd5c-2bsz8/status",
  Verb:"update", APIPrefix:"api", APIGroup:"", APIVersion:"v1",
  Namespace:"kube-system", Resource:"pods", Subresource:"status",
  Name:"kube-dns-5f7bc9fd5c-2bsz8", Parts:[]string{"pods",
    "kube-dns-5f7bc9fd5c-2bsz8", "status"}},
userInfo=&user.DefaultInfo{Name:"system:kube-scheduler", UID:"",
  Groups:[]string{"system:authenticated"},
  Extra:map[string][]string(nil)}
```

 Create Event About Hosted Control Plane
```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/apis/events.k8s.io/v1beta1/namespaces/example-com/events",
  Verb:"create", APIPrefix:"apis", APIGroup:"events.k8s.io",
  APIVersion:"v1beta1", Namespace:"example-com", Resource:"events",
  Subresource:"", Name:"", Parts:[]string{"events"}},
userInfo=&user.DefaultInfo{Name:"system:kube-scheduler", UID:"",
  Groups:[]string{"system:authenticated"},
  Extra:map[string][]string(nil)}
```

#### Kubelet Updates Pod Status

```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/api/v1/namespaces/default/pods/bb1-66bdc74b9c-bgm47/status",
  Verb:"patch", APIPrefix:"api", APIGroup:"", APIVersion:"v1",
  Namespace:"default", Resource:"pods", Subresource:"status",
  Name:"bb1-66bdc74b9c-bgm47", Parts:[]string{"pods",
    "bb1-66bdc74b9c-bgm47", "status"}},
userInfo=&user.DefaultInfo{Name:"system:node:127.0.0.1", UID:"",
  Groups:[]string{"system:nodes", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

#### Controller in Hosted Control Plane

LIST
```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/apis/network.example.com/v1alpha1/subnets", Verb:"list",
  APIPrefix:"apis", APIGroup:"network.example.com",
  APIVersion:"v1alpha1", Namespace:"", Resource:"subnets",
  Subresource:"", Name:"", Parts:[]string{"subnets"}},
userInfo=&user.DefaultInfo{Name:"system:serviceaccount:example-com:kos-controller-manager",
  UID:"cd95ccf8-f1ea-4398-9cc3-035331125442",
  Groups:[]string{"system:serviceaccounts",
   "system:serviceaccounts:example-com", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

WATCH
```
RequestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/apis/network.example.com/v1alpha1/subnets", Verb:"watch",
  APIPrefix:"apis", APIGroup:"network.example.com",
  APIVersion:"v1alpha1", Namespace:"", Resource:"subnets",
  Subresource:"", Name:"", Parts:[]string{"subnets"}},
user.Info=&user.DefaultInfo{Name:"system:serviceaccount:example-com:kos-controller-manager",
  UID:"cd95ccf8-f1ea-4398-9cc3-035331125442",
  Groups:[]string{"system:serviceaccounts",
    "system:serviceaccounts:example-com", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

UPDATE
```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/apis/network.example.com/v1alpha1/namespaces/default/subnets/sn-1",
  Verb:"update", APIPrefix:"apis", APIGroup:"network.example.com",
  APIVersion:"v1alpha1", Namespace:"default", Resource:"subnets",
  Subresource:"", Name:"sn-1", Parts:[]string{"subnets", "sn-1"}},
userInfo=&user.DefaultInfo{Name:"system:serviceaccount:example-com:kos-controller-manager",
  UID:"cd95ccf8-f1ea-4398-9cc3-035331125442",
  Groups:[]string{"system:serviceaccounts",
    "system:serviceaccounts:example-com", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

#### LOG and EXEC on Workload Pod

`kubectl logs bb1-66bdc74b9c-bgm47`

```
RequestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/api/v1/namespaces/default/pods/bb1-66bdc74b9c-bgm47/log",
  Verb:"get", APIPrefix:"api", APIGroup:"", APIVersion:"v1",
  Namespace:"default", Resource:"pods", Subresource:"log",
  Name:"bb1-66bdc74b9c-bgm47", Parts:[]string{"pods",
    "bb1-66bdc74b9c-bgm47", "log"}},
user.Info=&user.DefaultInfo{Name:"system:admin", UID:"",
  Groups:[]string{"system:masters", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

`kubectl logs -f bb1-66bdc74b9c-bgm47`

```
RequestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/api/v1/namespaces/default/pods/bb1-66bdc74b9c-bgm47/log",
  Verb:"get", APIPrefix:"api", APIGroup:"", APIVersion:"v1",
  Namespace:"default", Resource:"pods", Subresource:"log",
  Name:"bb1-66bdc74b9c-bgm47", Parts:[]string{"pods",
    "bb1-66bdc74b9c-bgm47", "log"}},
user.Info=&user.DefaultInfo{Name:"system:admin", UID:"",
  Groups:[]string{"system:masters", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

`kubectl exec bb1-66bdc74b9c-bgm47 date`

```
RequestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/api/v1/namespaces/default/pods/bb1-66bdc74b9c-bgm47/exec",
  Verb:"create", APIPrefix:"api", APIGroup:"", APIVersion:"v1",
  Namespace:"default", Resource:"pods", Subresource:"exec",
  Name:"bb1-66bdc74b9c-bgm47", Parts:[]string{"pods",
    "bb1-66bdc74b9c-bgm47", "exec"}},
user.Info=&user.DefaultInfo{Name:"system:admin", UID:"",
  Groups:[]string{"system:masters", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

#### Requests Over Insecure Port

Before fix
```
W0821 18:10:40.546166    5468 reqmgmt.go:56] No User found in context
&context.valueCtx{Context:(*context.valueCtx)(0xc007c89ad0), key:0,
  val:(*request.RequestInfo)(0xc0016afa20)}
of request &http.Request{Method:"GET",
  URL:(*url.URL)(0xc006f87d00), Proto:"HTTP/1.1", ProtoMajor:1,
  ProtoMinor:1,
  Header:http.Header{"Accept":[]string{"*/*"},
    "User-Agent":[]string{"curl/7.58.0"}},
  Body:http.noBody{}, GetBody:(func() (io.ReadCloser, error))(nil),
  ContentLength:0, TransferEncoding:[]string(nil), Close:false,
  Host:"localhost:8080", Form:url.Values(nil),
  PostForm:url.Values(nil),
  MultipartForm:(*multipart.Form)(nil), Trailer:http.Header(nil),
  RemoteAddr:"127.0.0.1:41628", RequestURI:"/healthz",
  TLS:(*tls.ConnectionState)(nil), Cancel:(<-chan struct {})(nil),
  Response:(*http.Response)(nil), ctx:(*context.valueCtx)(0xc007c89b00)}
```

After https://github.com/kubernetes/kubernetes/pull/81788
```
requestInfo=&request.RequestInfo{IsResourceRequest:false,
  Path:"/healthz/zzz", Verb:"get", APIPrefix:"", APIGroup:"",
  APIVersion:"", Namespace:"", Resource:"", Subresource:"", Name:"",
  Parts:[]string(nil)},
userInfo=&user.DefaultInfo{Name:"system:unsecured", UID:"",
  Groups:[]string{"system:masters", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

```
requestInfo=&request.RequestInfo{IsResourceRequest:true,
  Path:"/api/v1/namespaces/fooobar", Verb:"get", APIPrefix:"api",
  APIGroup:"", APIVersion:"v1", Namespace:"fooobar",
  Resource:"namespaces", Subresource:"", Name:"fooobar",
  Parts:[]string{"namespaces", "fooobar"}},
userInfo=&user.DefaultInfo{Name:"system:unsecured", UID:"",
  Groups:[]string{"system:masters", "system:authenticated"},
  Extra:map[string][]string(nil)}
```

### Implementation Details/Notes/Constraints

TBD

### Risks and Mitigations

Implementing this KEP will increase the overhead in serving each
request, perhaps to a degree that depends on some measure of system
and/or workload size.  The additional overhead must be practically
limited.

There are likely others.

## Design Details

We are still ironing out the high level goals and approach.  Several
earlier proposals have been floated, as listed next.  This section
contains a discussion of the issues.

### References

- [Min Kim's original proposal](https://docs.google.com/document/d/12xAkRcSq9hZVEpcO56EIiEYmd0ivybWo4YRXV0Nfq-8)

- [Mike Spreitzer's first proposal](https://docs.google.com/document/d/1YW_rYH6tvW0fvny5b7yEZXvwDZ1-qtA-uMNnHW9gNpQ)

- [Daniel Smith's proposal](https://docs.google.com/document/d/1BtwFyB6G3JgYOaTxPjQD-tKHXaIYTByKlwlqLE0RUrk)

- [Mike's second proposal](https://docs.google.com/document/d/1c5SkLHvA4H25sY0lihJtu5ESHm36h786vi9LaQ8xdtY)

- [Min's second proposal](https://github.com/kubernetes/enhancements/pull/864)

- [Daniel's brain dump](https://docs.google.com/document/d/1cwNqMDeJ_prthk_pOS17YkTS_54D8PbFj_XwfJNe4mE)

- [Mike's third proposal](https://github.com/kubernetes/enhancements/pull/930)

- [Mike's proposed first cut](https://github.com/kubernetes/enhancements/pull/933)

Also notable are the following notes from meetings on this subject.

- https://docs.google.com/document/d/1bEh2BqfSSr3jyh1isnXDdmfe6koKd_kMXCFj08uldf8
- https://docs.google.com/document/d/1P8NRaQaJBiBAP2Bb4qyunJpyQ-4JVzripfCi3UXG9zc

### Design Considerations

Following is an attempt to summarize the issues addressed in those
proposals and the current thinking on them; the current proposal
attempts to respond.

Despite the open issues, we seem to be roughly agreed on an outline
something like the following.

- When a request arrives at the handler, the request is categorized
  somehow.  The nature of the categories and the categorization
  process is one open issue. Some proposals allow for the request to
  be rejected upon arrival based on that categorization and some local
  state.  Unless rejected, the request is put into a FIFO queue.  That
  is one of many queues.  The queues are associated with the
  categories somehow.  Some proposals contemplate ejecting less
  desirable requests to make room for the newly queued request, if and
  when queue space is tight.

- A request might also be rejected at a later time, based on other
  criteria.  For example, as in the CoDel technique --- which will
  reject a request at the head of the queue if the latency is found to
  be too big at certain times.

- Based on some resource limit (e.g., QPS or concurrency) and with
  regards to priority and fairness criteria, requests are dispatched
  from the queues to be served (i.e., continue down the handler
  chain).

- We assume that when the requst-timeout handler aborts a request it
  is effective --- we assume the request stops consuming CPU and
  memory at that point.  We know that this is not actually true today,
  but is intended; we leave fixing that to be independent work, and
  for now this KEP simply ignores the gap.

One of the biggest questions is how to formulate the scheduling
parameters.  There are several related concepts in the state of the
art of scheduling, and we are trying to figure out what to adopt
and/or invent.  We would prefer to invent as little as possible; it is
a non-trivial thing to invent a new --- and sufficiently efficient ---
scheduling algorithm and prove it correct.  The vmware product line
uses scheduling parameters named "reservation", "limit", and "shares"
for each class of workload.  The first two are known elsewhere as
"assured" and "ceiling".  Finding a published algorithm that
implements all three has not proven easy.  Alternatively, priorities
are easy to implement and arguably more desirable --- provided there
is some form of fairness within each priority level.  The current
thinking is in that direction: use priorities, with simple equal
fairness among some categories of requests in each priority level.
There are published scheduling algorithms that provide fairness, and
we hope to use/adapt one of them to apply independently within the
confines of each priroity level.

Another issue is whether to manage QPS or concurrency or what.
Managing QPS leaps first to mind, perhaps because it is a simple
concept and perhaps because it is familiar from the self-restraint
that clients apply today.  But we want to also take service time into
account; a request flow with longer service times should get less QPS
because its requests are "heavier" --- they impose more load on the
apiserver.  A natural way to do this is with an inverse linear
relation.  For example, when two CPU-bound request flows are getting
equal CPU from the apiserver, and the first flow's requests have a
service time that is X times the service time of the second flow's
requests, the first flow's QPS is 1/X of the second's.  This is
exactly analogous to what happens in networking: if two flows are
getting the same bandwidth, and one flow's packets are X times as big
as the second's, then the first flow's packets per second rate is 1/X
that of the second flow.  This inverse linear relation amounts to
managing the product of QPS * service time.  That is equivalent to
managing concurrency.  Managing concurrency is an obvious choice for
memory, and we now see it is a good choice for CPU too.  This is also
a convenient choice because it is what the max-in-flight handler is
doing today, so we would be making a relatively modest extension to
that handler's conception.

Compared to traditional scheduling problems, ours is harder because of
the combination of these facts: (1) (unlike a router handling a
packet) the apiserver does not know beforehand how long a request will
take to serve nor how much memory it will consume, and (2) (unlike a
CPU scheduler) the apiserver can not suspend and resume requests.
Also, we are generally loathe to abort a request once it has started
being served.  We may some day consider doing this for low-priority
long-running requests, but are not addressing long-running requests at
first.  We are leaning towards adapting well known and studied
scheduling technique(s); but adaptation is a form of invention, and we
have not converged yet on what to do here.

Another issue is how to combine two goals: protection of CPU, and
protection of memory.  A related issue is the fact that there are two
stages of memory consumption: a request held in a queue holds some
memory, and a request being served may use a lot more.  The current
thinking seems to be focusing on using one QPS or concurrency limit on
requests being served, on the expectation that this limit can be set
to a value that provides reasonable protection for both CPU and memory
without being too low for either.

If we only limmit requests being served then the queues could cause
two problems: consuming a lot of apiserver memory, and introducing a
lot of latency.  For the latter we are aware of some solutions from
the world of networking, [CoDel](https://en.wikipedia.org/wiki/CoDel)
and
[fq_codel](https://tools.ietf.org/html/draft-ietf-aqm-fq-codel-06).
CoDel is a technique for ejecting requests from a queue for the
purpose of keeping latency low, and fq_codel applies the CoDel
technique in each of many queues.  CoDel is explicitly designed to
work in the context of TCP flows on the Internet.  This KEP should be
similarly explicit about the context, particularly including what is
the feedback given to clients and how do they react and what is the
net effect of all the interacting pieces.  No such analysis has yet
been done for any of the proposals.

The CoDel technique is described as parameterless but has two magic
numbers: an "initial interval" of 100 ms and a "target" of 5 ms.  The
initial interval is set based on round trip times in the Internet, and
the target is set based on a desired limit on the latency at each hop.
What are the analogous numbers for our scenario?  We do not have large
numbers of hops; typically at most two (client to main apiserver and
then main apiserver to aggregated apiserver).  What is analogous to
network round trip time?  We have a latency goal of 1 second, and a
request service time limit of 1 minute.  If we take 1 second as the
initial interval then note that the maximum service time is much
larger than the initial interval; by contrast, in networking, the
maximum service time (i.e., packet length / link speed) is much
smaller than the initial interval.  Even if we take 1 minute as our
initial interval, we still do not have the sort of relationship that
obtains in networking.  Note that in order to get good statistics on a
queue --- which is needed by the CoDel technique --- there have to be
many requests served during an interval.  Because of this mismatch,
and because equivalence of context has not been established, we are
not agreed that the CoDel technique can be used.

Note that the resource limit being applied is a distinct concept from
the fairness criteria.  For example, in CPU scheduling there may be 4
CPUs and 50 threads being scheduled onto those CPUs; we do not suppose
the goal is to have each thread to be using 0.08 CPUs at each instant;
a thread uses either 0 or 1 CPUs at a given instant.  Similarly, in
networking, a router may multiplex a thousand flows onto one link; the
goal is not to have each flow use 1/1000th of the link at each
instant; a packet uses 0 links while queued and 1 link while being
transmitted.  Each CPU or link is used for just one thing at a time;
this is the resource limit.  The fairness goal is about utilization
observed over time.  So it is in our scenario too.  For example, we
may have 5000 flows of requests and a concurrency limit of 600
requests at any one time.  That does not mean that our goal is for
each flow to have 0.12 requests running at each instant.  Our goal is
to limit the number of running requests to 600 at each instant and
provide some fairness in utilization averaged over time.

That average over time must not be over too much or too little time.
It would not make sense to average over all past time; that would
allow a flow to build up a huge amount of credit, enabling it to crowd
out other flows.  It also does not make sense for the average to cover
a small amount of time.  Because serving requests, like transmitting
packets, is lumpy we must average over many service times.  Approaches
to this include: using a sequence of intervals, using a sliding
window, and using an exponential decay.

Another open issue is the categorization: what are the categories and
how is a request assigned to a category?  We seem to be agreed on at
least one important point: each request is assigned to exactly one
category, and is handled by exactly one "bucket" or "queue".  We also
seem to be converging toward a two-level hierarchy of categories,
aligned with the handling outline discussed earlier: at the higher
level there are priorities, and within each priority level there is a
collection of flows that compete fairly.

It is desired to allow lesser priority traffic to get some service
even while higher priority traffic is arriving at a sufficient rate to
entirely occupy the server.  There needs to be a quantitative
definition of this relation between the priorities, and an
implementation that (at least roughly) implements the desired
quantitative relation.

For the higher level of categorization --- i.e., into priority levels
--- the idea is that this is based on a configured set of predicate =>
priority associations.  The predicate can test any authenticated
request attribute --- notably including both client identity and the
work being requested.  One issue to nail down is the question of what
happens if multiple predicates match a given request; the handler
should pick exactly one priority level for each request.

Within a given priority level we want to give a fair division of
capacity among several "flows"; the lower level of categorization is
how to compute a flow identifier from a request.

The handler may additionally hash the flows into queues, so that a
more manageable number of queues is involved.  Shuffle sharding can be
used to make it more likely that a mouse can avoid being squashed by
an elephant.

Some of the proposals draw inspiration from deficit (weighted or not)
round robin, and some from (weighted or not) fair queuing.  DRR has
the advantage of O(1) time to make a decision --- but ONLY if the
quantum is larger than the largest packet.  In our case that would be
quite large indeed, since the timeout for a request is typically 1
minute (it would be even worse if we wanted to handle WATCH requests).
The dispatching of the DRR technique is bursty, and the size of the
burst increases with the quantum.  The proposals based on DRR tend to
go with a small quantum, presumably to combat the burstiness.  The
logical extreme of this, in the unweighted case, is to use a quantum
of 1 bit (in the terms of the original networking setting).  That is
isomorphic to (unweighted) fair queuing!  The weighted versions, still
with miniscule quanta, are also isomorphic.

### Test Plan

- __Unit Tests__: All changes must be covered by unit tests. Additionally,
 we need to test the evenness of dispatching algorithm.
- __Integration Tests__: The use cases discussed in this KEP must be covered by integration tests.

### Graduation Criteria

Alpha:

- Necessary defaulting, validation
- Adequate documentation for the changes
- Minimum viable test cases mentioned in Test Plan section

Beta:

- Blocking Items:
  - Improving observability and robustness: adding debug endpoint dumping fine-grained states of the queues for priority-levels
  - Providing approaches to opt-out client-side rate-limitting: configurable client-side ratelimitting(QPS/Burst) via either kubeconfig or command-line flags 
  - Necessary e2e test: adding E2E tests which at least covers:
    - Basics of `flowcontrol.apiserver.k8s.io/v1beta1` API 
    - Reloading flowcontrol configurations upon `flowcontrol.apiserver.k8s.io/v1beta1` API resources changes
- Non-Blocking Items:
  - Supports concurrency limiting upon long-running requests
  - Allow constant concurrency/relative shares in the priority-level API model
  - Automatically manages versions of mandatory/suggested configuration
  - Discrimates paginated LIST requests


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
