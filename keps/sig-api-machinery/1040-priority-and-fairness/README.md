# KEP-1040: Priority and Fairness for API Server Requests

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
  - [Fair Queuing for Server Requests problem statement](#fair-queuing-for-server-requests-problem-statement)
  - [Fair Queuing for Server Requests, with equal allocations and serial virtual execution](#fair-queuing-for-server-requests-with-equal-allocations-and-serial-virtual-execution)
    - [Fair Queuing for Server Requests, with equal allocations and serial virtual execution, initial definition](#fair-queuing-for-server-requests-with-equal-allocations-and-serial-virtual-execution-initial-definition)
    - [Fair Queuing for Server Requests, with equal allocations and serial virtual execution, intended behavior](#fair-queuing-for-server-requests-with-equal-allocations-and-serial-virtual-execution-intended-behavior)
    - [Implementation of Fair Queuing for Server Requests with equal allocations and serial execution, technique and problems](#implementation-of-fair-queuing-for-server-requests-with-equal-allocations-and-serial-execution-technique-and-problems)
  - [Support for LIST requests](#support-for-list-requests)
    - [Width of the request](#width-of-the-request)
    - [Determining the width](#determining-the-width)
    - [Dispatching the request, as a modification to the initial Fair Queuing for Server Requests](#dispatching-the-request-as-a-modification-to-the-initial-fair-queuing-for-server-requests)
  - [Support for WATCH requests](#support-for-watch-requests)
    - [Watch initialization](#watch-initialization)
      - [Getting the initialization signal](#getting-the-initialization-signal)
      - [Passing the initialization signal to the filter](#passing-the-initialization-signal-to-the-filter)
      - [Width of the request](#width-of-the-request-1)
    - [Keeping the watch up-to-date](#keeping-the-watch-up-to-date)
      - [Estimating cost of the request](#estimating-cost-of-the-request)
      - [Multiple apiservers](#multiple-apiservers)
      - [Cost of the watch event](#cost-of-the-watch-event)
      - [Dispatching the request, as a modification to the initial Fair Queuing for Server Requests with LIST support](#dispatching-the-request-as-a-modification-to-the-initial-fair-queuing-for-server-requests-with-list-support)
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
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] KEP approvers have set the KEP status to `implementable`
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

Today the apiserver has a simple mechanism for protecting itself
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

This KEP introduces a new functionality that is supposed to address the
problems above. The goals/requirements for the solution (in the priority
order) are the following:
1. overload protection
1. fairness
1. throughput

So in other words, first of all we want to protect Kubernetes from overload,
within these boundaries make it fair across tenants and only with those
constraints optimize throughput.

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

- CONNECT requests are out of scope.  These are of a fairly
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

- Do something about CONNECT requests.

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
level; each priority level dispatches to its own concurrency pool and,
according to a configured limit, unused concurrency borrrowed from
lower priority levels; within each priority level queues compete with
even fairness.

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
examined, and the request is put in one of the queues holding the
least amount of work.  Originally this was just a matter of examining
queue length.  With the generalizations for width and extra latency,
the work in a queue is the sum of the work in its waiting requsts.
The work in a request is the product of its width and its total
estimated execution duration (including extra latency).

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

As mentioned [above](#non-goals), the functionality described here
operates independently in each apiserver.

The concurrency limit of an apiserver is divided among the non-exempt
priority levels, and they can do a limited amount of borrowing from
each other.

Two fields of `LimitedPriorityLevelConfiguration`, introduced in the
midst of the `v1beta2` lifetime, limit the borrowing.  The fields are
added in all the versions (`v1alpha1`, `v1beta1`, and `v1beta2`).  The
following display shows the new fields along with the updated
description for the `AssuredConcurrencyShares` field, in `v1beta2`.

```go
type LimitedPriorityLevelConfiguration struct {
  ...
  // `assuredConcurrencyShares` (ACS) contributes to the computation of the
  // NominalConcurrencyLimit (NominalCL) of this level.
  // This is the number of execution seats available at this priority level.
  // This is used both for requests dispatched from
  // this priority level as well as requests dispatched from other priority
  // levels borrowing seats from this level.
  // The server's concurrency limit (ServerCL) is divided among the
  // Limited priority levels in proportion to their ACS values:
  //
  // NominalCL(i)  = ceil( ServerCL * ACS(i) / sum_acs )
  // sum_acs = sum[limited priority level k] ACS(k)
  //
  // Bigger numbers mean a larger nominal concurrency limit, at the expense
  // of every other Limited priority level.
  // This field has a default value of 30.
  // +optional
  AssuredConcurrencyShares int32

  // `lendablePercent` prescribes the fraction of the level's NominalCL that
  // can be borrowed by other priority levels.  This value of this
  // field must be between 0 and 100, inclusive, and it defaults to 0.
  // The number of seats that other levels can borrow from this level, known
  // as this level's LendableConcurrencyLimit (LendableCL), is defined as follows.
  //
  // LendableCL(i) = round( NominalCL(i) * lendablePercent(i)/100.0 )
  //
  // +optional
  LendablePercent int32
  
  // `borrowingLimitPercent`, if present, specifies a limit on how many seats
  // this priority level can borrow from other priority levels.  The limit
  // is known as this level's BorrowingConcurrencyLimit (BorrowingCL) and
  // is a limit on the total number of seats that this level may borrow
  // at any one time.  When this field is non-nil, it must hold a non-negative
  // integer and the limit is calculated as follows.
  //
  // BorrowingCL(i) = round( NominalCL(i) * borrowingLimitPercent(i)/100.0 )
  //
  // When this field is left `nil`, the limit is effetively infinite.
  // +optional
  BorrowingLimitPercent *int32
}
```

Prior to the introduction of borrowing, the `assuredConcurrencyShares`
field had two meanings that amounted to the same thing: the total
shares of the level, and the non-lendable shares of the level.
While it is somewhat unnatural to keep the meaning of "total shares"
for a field named "assured" shares, rolling out the new behavior into
existing systems will be more continuous if we keep the meaning of
"total shares" for the existing field.  In the next version we should
rename the `AssuredConcurrencyShares` to `NominalConcurrencyShares`.

The following table shows the current default non-exempt priority
levels and a proposal for their new configuration.

| Name | Assured Shares | Proposed Lendable | Proposed Borrowing Limit |
| ---- | -------------: | ----------------: | -----------------------: |
| leader-election |  10 |   0% | none |
| node-high       |  40 |  25% | none |
| system          |  30 |  33% | none |
| workload-high   |  40 |  50% | none |
| workload-low    | 100 |  90% | none |
| global-default  |  20 |  50% | none |
| catch-all       |   5 |   0% | none |

Each non-exempt priority level `i` has two concurrency limits: its
NominalConcurrencyLimit (`NominalCL(i)`) as defined above by
configuration, and a CurrentConcurrencyLimit (`CurrentCL(i)`) that is
used in dispatching requests.  The CurrentCLs are adjusted
periodically, based on configuration, the current situation at
adjustment time, and recent observations.  The "borrowing" resides in
the differences between CurrentCL and NominalCL.  There are upper and lower
bound on each non-exempt priority level's CurrentCL, as follows.

```
MaxCL(i) = NominalCL(i) + BorrowingCL(i)
MinCL(i) = NominalCL(i) - LendableCL(i)
```

Naturally the CurrentCL values are also limited by how many seats are
available for borrowing from other priority levels.  The sum of the
CurrentCLs is always equal to the server's concurrency limit
(ServerCL) plus or minus a little for rounding in the adjustment
algorithm below.

Dispatching is done independently for each priority level.  Whenever
(1) a non-exempt priority level's number of occupied seats is zero or
below the level's CurrentCL and (2) that priority level has a
non-empty queue, it is time to consider dispatching another request
for service.  The Fair Queuing for Server Requests algorithm below is
used to pick a non-empty queue at that priority level.  Then the
request at the head of that queue is dispatched if possible.

Every 10 seconds, all the CurrentCLs are adjusted.  We do smoothing on
the inputs to the adjustment logic in order to dampen control
gyrations, in a way that lets a priority level reclaim lent seats at
the nearest adjustment time.  The adjustments take into account the
high watermark `HighSeatDemand(i)`, time-weighted average
`AvgSeatDemand(i)`, and time-weighted population standard deviation
`StDevSeatDemand(i)` of each priority level `i`'s seat demand over the
just-concluded adjustment period.  A priority level's seat demand at
any given moment is the sum of its occupied seats and the number of
seats in the queued requests.  We also define `EnvelopeSeatDemand(i) =
AvgSeatDemand(i) + StDevSeatDemand(i)`.  The adjustment logic is
driven by a quantity called smoothed seat demand
(`SmoothSeatDemand(i)`), which does an exponential averaging of
EnvelopeSeatDemand values using a coeficient A in the range (0,1) and
immediately tracks EnvelopeSeatDemand when it exceeds
SmoothSeatDemand.  The rule for updating priority level `i`'s
SmoothSeatDemand at the end of an adjustment period is
`SmoothSeatDemand(i) := max( EnvelopeSeatDemand(i),
A*SmoothSeatDemand(i) + (1-A)*EnvelopeSeatDemand(i) )`.  The command
line flag `--seat-demand-history-fraction` with a default value of 0.9
configures A.

Adjustment is also done on configuration change, when a priority level
is introduced or removed or its NominalCL, LendableCL, or BorrowingCL
changes.  At such a time, the current adjustment period comes to an
early end and the regular adjustment logic runs; the adjustment timer
is reset to next fire 10 seconds later.  For a newly introduced
priority level, we set HighSeatDemand, AvgSeatDemand, and
SmoothSeatDemand to NominalCL-LendableSD/2 and StDevSeatDemand to
zero.

For adjusting the CurrentCL values, each non-exempt priority level `i`
has a lower bound (`MinCurrentCL(i)`) for the new value.  It is simply
HighSeatDemand clipped by the configured concurrency limits:
`MinCurrentCL(i) = max( MinCL(i), min( NominalCL(i), HighSeatDemand(i)
) )`.

If `MinCurrentCL(i) = NominalCL(i)` for every non-exempt priority
level `i` then there is no wiggle room.  In this situation, no
priority level is willing to lend any seats.  The new CurrentCL values
must equal the NominalCL values.  Otherwise there is wiggle room and
the adjustment proceeds as follows.  For the following logic we let
the CurrentCL values be floating-point numbers, not necessarily
integers.

The priority levels would all be fairly happy if we set CurrentCL =
SmoothSeatDemand for each.  We clip that by the lower bound just shown
and define `Target(i)` as follows, taking it as a first-order target
for each non-exempt priority level `i`.

```
Target(i) = max( MinCurrentCL(i), SmoothSeatDemand(i) )
```

Sadly, the sum of the Target values --- let's name that TargetSum ---
is not necessarily equal to ServerCL.  However, if `TargetSum <
ServerCL` then all the Targets could be scaled up in the same
proportion `FairProp = ServerCL / TargetSum` (if that did not violate
any upper bound) to get the new concurrency limits `CurrentCL(i) :=
FairProp * Target(i)` for each non-exempt priority level `i`.
Similarly, if `TargetSum > ServerCL` then all the Targets could be
scaled down in the same proportion (if that did not violate any lower
bound) to get the new concurrency limits.  This shares the wealth or
the pain proportionally among the priority levels (but note: the upper
bound does not affect the target, lest the pain of not achieving a
high SmoothSeatDemand be distorted, while the lower bound _does_
affect the target, so that merely achieving the lower bound is not
considered a gain).  The following computation generalizes this idea
to respect the relevant bounds.

We can not necessarily scale all the Targets by the same factor ---
because that might violate some upper or lower bounds.  The problem is
to find a proportion `FairProp` that can be shared by all the priority
levels except those with a bound that forbids it.  This means to find
a value of `FairProp` that simultaneously solves all the following
conditions, for the non-exempt priority levels `i`, and also makes the
CurrentCL values sum to ServerCL.  In some cases there are many
satisfactory values of `FairProp` --- and that is OK, because they all
produce the same CurrentCL values.

```
CurrentCL(i) = min( MaxCL(i), max( MinCurrentCL(i), FairProp * Target(i) ))
```

This is similar to the max-min fairness problem and can be solved
using sorting and then a greedy algorithm, taking O(N log N) time and
O(N) space.

After finding the floating point CurrentCL solutions, each one is
rounded to the nearest integer to use in subsequent dispatching.

### Fair Queuing for Server Requests

The following subsections cover the problem statements and the current
solution.  This solution is dissatisfying in the following two ways.

1. By allocating the available seats equally rather than with max-min
fairness, the current solution sometimes pretends that a queue uses
more seats than it can.

2. By executing just one request at a time for a given queue in the
virtual world, the current solution has many circumstances in which a
fully accurate implementation would have to revise what happened in
the past in the virtual world.  That would require an expensive amount
of information retention and recalculation.  The current solution cuts
those corners.

### Fair Queuing for Server Requests problem statement

The Fair Queuing for Server Requests problem is as follows.

- Dispatch queued requests to a server that has a capacity of `C`
  concurrent seats, with max-min fairness in concurrency usage.

- Identify a request by the index `i` of its queue and the request's
  sequence number `j` in that queue.  Request `(i,j)` is the `j`th
  request to go into queue `i`.  For the sake of procedural
  regularity, every request is put into its queue upon arrival.
  Dispatch (that is, starting to execute the request) can happen
  immediately, if allowed by the regular constraints.

- Request `(i,j)` enters its queue at time `t_arrive(i,j)`.

- Each request `(i,j)` comes tagged with `width(i,j)`, the integer
  number of seats it occupies.  This is typically 1, but can be up to
  a configured limit `A` (typically a double-digit number) in some cases.

- At any given moment the sum of the widths of the executing requests
  should be as high as it can be without exceeding `C`, but if that
  would not allow any requests to be executing then instead there
  should be one executing.

- A request can be ejected before it has been dispatched.  This means
  to remove the request and never execute it.

- A request does _not_ come with an indication of how long it will
  take to execute.  That is only known when the execution finishes.

- A request _does_ come tagged with some extra time
  `extra_latency(i,j)` to tack onto the end its execution.  This extra
  time does not delay the return from the request handler, but _does_
  extend the time that the request's seats are considered to be
  occupied.

### Fair Queuing for Server Requests, with equal allocations and serial virtual execution

The is the technique currently used.  It occupies one of four
quadrants defined by two binary characteristics:
- whether the allocations of concurrency are equal or max-min fair,
  and
- whether a non-empty queue has exactly one or many requests executing
  at a time in the virtual world.

The following subsections cover: (a) the original KEP text used to
create the initial implementation (which lacked generalizations for
request width and extra latency), (b) equations that describe the
intended behavior, and (c) implementation difficulties (solved and
unsolved).

#### Fair Queuing for Server Requests, with equal allocations and serial virtual execution, initial definition

Following is the design that motivated the code as it existed just
before we started adding the concepts of width and extra latency.
There are problems with this design and the code, discussed in the
next section.

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

#### Fair Queuing for Server Requests, with equal allocations and serial virtual execution, intended behavior

Here is a succinct summary of behavior intended to solve the Fair
Queuing for Server Requests problem but with equal allocations instead
of max-min fair allocations.  This includes general width and extra
latency, although their implementation is in progress.

We imagine a virtual world in which request executions are scheduled
(queued and executed) differently than in the real world.  In the
virtual world, requests are generally executed with more or less
concurrency than in the real world and with infinitesimal
interleaving.

PLEASE NOTE VERY CAREFULLY: the virtual world uses the same clock as
the real world.  This is also true in the original Fair Queuing paper.
Some subsequent authors --- including the authors of the
implementation outline on Wikipedia and thus the authors of the
original APF work and thus the current code --- use the term "virtual
time" to refer to what the original paper and the discussion below
call `R(t)`.  Where the discussion below refers to "time in the
virtual world", it means `t` rather than `R(t)`.

Define the following.

- `len(i,j)` is the execution duration for request `(i,j)` in the real
  world, including `extra_latency(i,j)`.  At first `len` is only a
  guess.  When the real execution duration is eventually learned,
  `len` gets updated with that information.

- `t_dispatch_virtual(i,j)` is the time when request `(i,j)` begins
  execution in the virtual world.

- `t_finish_virtual(i,j)` is the time when request `(i,j)` ends
  execution in the virtual world (this includes completion of
  `extra_latency(i,j)`).  This is _not_ simply `t_dispatch_virtual`
  plus `len` because requests generally execute at a different rate in
  the virtual world.

- `NOS(i,t)` is the Number of Occupied Seats by queue `i` at time `t`;
  this is `Sum[over j such that t_dispatch_virtual(i,j) <= t <
  t_finish_virtual(i,j)] width(i,j)`.

- `SAQ(t)` is the Set of Active Queues at time `t`: those `i` for
  which there is a `j` such that `t_arrive(i,j) <= t <
  t_finish_virtual(i,j)`.

- `NEQ(t)` is the number of Non-Empty Queues at time `t`; it is the
  number of queues in `SAQ(t)`.

At time `t`, queue `i` is requesting `rho(i,t)` concurrency in the
virtual world.  This is the sum of the widths of the requests that
have arrived but not yet finished executing.

```
rho(i,t) =
    Sum[over j such that t_arrive(i,j) <= t < t_finish_virtual(i,j)]
        width(i,j)
```

The allocations of concurrency are written as `mu(i,t)` seats for
queue `i` at time `t`.  This design uses allocations that are equal
among non-empty queues, as follows.

```
mu(i,t) = mu_equal(t)  if rho(i,t) > 0
        = 0            otherwise

mu_equal(t) = min(C, Sum[over i] rho(i,t)) / NEQ(t)   if NEQ(t) > 0
            = 0                                       otherwise
```

In this design, a queue executes one request at a time in the virtual
world.  Thus, `NOS(i,t) = width(i,j)` for that relevant `j` whenever
there is one.

Each non-empty queue divides its allocated concurrency `mu` evenly
among the seats it occupies in the virtual world, so that the
aggregate rate work gets done on all the queue's seats is `mu`.

```
rate(i,t) = if NOS(i,t) > 0 then mu_equal(t) / NOS(i,t) else 0
```

Since `mu_equal(t)` can be greater or less than `width(i,j)`,
`rate(i,t)` can be greater or less than 1.

We use the above policy and rate to define the schedule in the virtual
world.  The scheduling is thus done with almost no interactions
between queues.  The interactions are limited to updating the `mu`
allocations whenever the `rho` requests change.

To make the scheduling technically easy to specify, we suppose that
no two requests arrive at the same time.  The implementation will be
serialized anyway, so this is no real restriction.

```
t_dispatch_virtual(i,j) = if j = 0 or t_finish_virtual(i,j-1) <= t_arrive(i,j)
         then t_arrive(i,j)
         else t_finish_virtual(i,j-1)
```

That is, a newly arrived request is dispatched immediately if the
queue had nothing executing otherwise the new request waits until all
of the queue's earlier requests finish.  Note that the concurrency
limit used here is different from the real world: a queue is allowed
to run one request at a time, regardless of how many it has waiting to
run and regardless of the server's concurrency limit `C`.  The
independent virtual-world scheduling for each queue is crucial for
fairness: a queue's virtual schedule depends only on the queue's
demand and allocated concurrency, not any detailed scheduling
interaction.  This also helps enable efficient implementations.

The end of a request's virtual execution (`t_finish_virtual(i,j)`) is
the solution to the following equation.

```
Integral[from tau=t_dispatch_virtual(i,j)
         to   tau=  t_finish_virtual(i,j)] rate(i,tau) dtau  =  len(i,j)
```

That is, each of a queue's requests is executed in the virtual world
at the aforementioned `rate`.  Before the real completion, `len` is
only a guess.  We use the same guess for every request, and use the
symbol `G` for that guess.

Once a request `(i,j)` finishes executing in the real world, its
actual execution duration is known and `len` gets set to that plus
`extra_latency(i,j)`.  This changes not only `t_finish_virtual(i,j)`
but also the `t_dispatch_virtual` and `t_finish_virtual` of the all
the queue's requests that arrived between `t_arrive(i,j)` and the next
of the queue's idle times.  Note that this can change `rho(i,t)` and
thus `mu_equal(t)` at those intervening times, and thus the subsequent
scheduling in other queues.  The computation of these changes can not
happen until `(i,j)` finishes executing in the real world --- but the
request might finish earlier in the virtual world (because of earlier
dispatch and/or `rate` being greater than 1).  An accurate
implementation would keep track of enough historical information to
revise all that scheduling.

The order of request dispatches in the real world is taken to be the
order of request completions in the virtual world.  In the case of
ties, round robin ordering is used starting from the queue that most
closely follows the one last dispatched from.  Requests are dispatched
as soon as allowed by that ordering and the concurrency bound in the
problem statement.

The following equation is an equivalent definition of
`t_finish_virtual(i,j)`.

```
Integral[from tau=t_dispatch_virtual(i,j)
         to   tau=  t_finish_virtual(i,j)] mu_equal(tau) dtau  =  width(i,j) * len(i,j)
```

For requests that have not yet begun executing in the real world, we
can simplify the above equation to the following.

```
Integral[from tau=t_dispatch_virtual(i,j)
         to   tau=  t_finish_virtual(i,j)] mu_equal(tau) dtau  =  width(i,j) * G
```

This means that when making real world dispatching decisions, the only
distinctions among waiting requests are their width estimates (plus
the previous history of their queue).  Originally we set `G` to a
value that is normally a gross over-estimate: one minute.  That meant
that differences in width estimates made grossly inordinate
differences in dispatching order.  Consider the example where all
requests actually take 100 ms to execute, and almost all have a width
of 1.  When a queue has a request of width 2 at its head, that queue's
estimated next completion time has a penalty compared to all the other
queues that is equal to the time it takes to execute 600 requests.
And once that width=2 request eventually is run and completes, that
queue's following estimated completion times jump forward by 1198
request's worth of time (as opposed to 599 in the usual case).

If requests all had the same width, we could set `G` to any value and
get the same behavior as any other value of `G`.  As far as
dispatching `width=1` requests is concerned, the problem at any time
is to pick a queue `i` for which the value of
`t_dispatch_virtual(i,oldest_waiting_j(i)) + G` is minimal.  Clearly,
the value of `G` does not matter to that ordering among queues.

To remove the inordinate impact of width estimates, we have changed
`G` to zero.  In other words, the differentiation among queues comes
only from requests that have completed in the real world.

We prefer the waiting requests to have _some_ effect, so plan to set
`G` to a small amount of time.  We plan to use 3 ms, as it is reported
to be a fairly characteristic execution duration.  Of course, no
number can be said to be truly characteristic of performance of
software so variously deployed and used; we settle for one number
based on some readily available experience.

Additionally, we could add `extra_latency(i,j)` when making the
initial guess at `len(i,j)`.  For now we avoid this, because it adds
another input that is only an estimate (and one with which we have
little experience).

An alternative way to avoid an inordinate impact from width estimates
is to guess not the execution time `len(i,j)` but rather the total
work `width(i,j) * len(i,j)`.  But a constant guess at that just
produces the same behavior as `G=0`, making the only differentiation
among queues come from their completed requests.

#### Implementation of Fair Queuing for Server Requests with equal allocations and serial execution, technique and problems

One of the key implementation ideas is taken from the original paper.
Define a global meter of progress named `R`, and characterize each
request's execution interval in terms of that meter, as follows.

```
R(t) = Integral[from tau=epoch to tau=t] mu_equal(tau) dtau
r_dispatch_virtual(i,j) = R(t_dispatch_virtual(i,j))
r_finish_virtual(i,j) =   R(t_finish_virtual(i,j))
```

In the current implementation, that global progress meter is called
"virtual time".

The value of working with `R` values rather than time is that they do
not vary with `mu` and `NOS`.  Compare the following two equations
(the first is derived from above, the second from the first and the
definitions of `R`, `r_dispatch_virtual`, and `r_finish_virtual`).

```
Integral[from tau=t_dispatch_virtual(i,j)
         to   tau=  t_finish_virtual(i,j)]
    mu_equal(tau) / width(i,j) dtau
    = len(i,j)

r_finish_virtual(i,j) = r_dispatch_virtual(i,j) + len(i,j) * width(i,j)
```

The next key idea is that when it comes time to dispatch the next
request in the real world, (a) it must be drawn from among those that
have arrived but not yet been dispatched in the real world and (b) it
must be the next one of those according to the ordering in the
real-world dispatch rule given in the problem statement --- recall
that is increasing `t_finish_virtual`, with ties broken by round-robin
ordering among queues.  The implementation maintains a cursor into
that order.  The cursor is a queue index (for resolving ties according
to round-robin) plus a bifurcated representation of each queue.  A
queue's representation consists of two parts: a set of requests that
are executing in the real world, and a FIFO of requests that are
waiting in the real world.  When it comes time to advance the cursor,
the problem is to find --- among the queues with requests waiting in
the real world --- the one whose oldest real-world-waiting request
(the one at head of the FIFO) has the lowest `t_finish_virtual`, with
ties broken appropriately.  This is the only place where the
`t_dispatch_virtual` and `t_finish_virtual` values have an effect on
the real world, and this fact is used to prune the implementation as
explained next.

It is not necessary to explicitly represent the `t_dispatch_virtual`
and `t_finish_virtual`, nor even the `r_dispatch_virtual` and
`r_finish_virtual`, of _every_ request that exists in the
implementation at a given moment.  For each queue with requests
waiting in the real world, all that is really needed (to support
finding the next request to dispatch) is the `r_finish_virtual` of the
oldest one of those requests.  The ordering constraint in the problem
statement is in terms of `t_finish_virtual`, but we can just as well
order by `r_finish_virtual` because `R(t)` is a monotonically
non-decreasing --- and strictly increasing where it matters ---
function of `t`.  The implementation calculates that request's
`r_finish_virtual` by adding its `len * width` to its
`r_dispatch_virtual`.  Rather than explicitly represent
`t_dispatch_virtual` and `t_finish_virtual`, or `r_dispatch_virtual`
and `r_finish_virtual`, on every request, the implementation simply
represents the `r_dispatch_virtual` of each queue's oldest
waiting-in-the-real-world request (if any).  In the implementation
this is a queue field named `virtualStart`.

A queue's `virtualStart` gets incrementally updated as follows.  The
regular case of virtual world dispatch is when one request `(i,j-1)`
finishes executing and the next one starts.  At that moment,
`virtualStart` was `r_finish_virtual(i,j-1) =
r_dispatch_virtual(i,j)`.  To account for the dispatch, the product
`len(i,j) * width(i,j)` is added to `virtualStart` because that
computes the initial guess at `r_finish_virtual(i,j) =
r_dispatch_virtual(i,j+1)`.  In the special case of a request arriving
to an empty queue, the arrival logic sets `virtualStart` to the
current `R` value --- because that is the `r_dispatch_virtual` of the
next request to dispatch --- and soon afterward the regular dispatch
logic does its thing.  When request `(i,j)` finishes execution and its
actual duration is learned, the queue's `virtualStart` is adjusted by
adding the product of `width(i,j)` and the correction to `len(i,j)`.

That is the only adjustment made when the correct value of `len(i,j)`
is learned.  This ignores other consequences that the equations above
call for.  Changing `len(i,j)` can change `rho(i,t)` for some times
`t`.  That can change `mu_equal(t)`, and thus the scheduling in all
the queues.

To advance the cursor, the implementation iterates through all the
queues to find, among those that have requests waiting in the real
world, the one with minimal next `r_finish_virtual` (with tie broken
appropriately).  This costs `O(N)` compute time, where `N` is the
number of queues.

Using an equal allocation of concurrency rather than a max-min fair
one is [issue
95979](https://github.com/kubernetes/kubernetes/issues/95979).  In
scenarios where the two allocations are different, equal allocation
gives some queues more concurrency than they can use and gives other
queues less than they should get.

One consequence of this mis-allocation is that while a queue uses less
than `mu_equal` but is non-empty at every moment when a request could
be dispatched from it, the equations above say that the queue gets
work done faster than it actually does.  The equations above assign
`t_dispatch_virtual` and `t_finish_virtual` values that reflect an
impossibly high rate of progress for such a queue.  That is, these
values get progressively more early.  The queue is effectively
building up "credit" in artificially low `t_dispatch_virtual` and
`t_finish_virtual` values, and can build up an arbitrary amount of
this credit.  Then an arbitrarily large amount of work could suddenly
arrive to that queue and crowd out other queues for an arbitrarily
long time.  To mitigate this problem, the implementation has a special
step that effectively prevents `t_dispatch_virtual` of the next
request to dispatch from dropping below the current time.  But that
solves only half of the problem.  Other queues may accumulate a
corresponding deficit (inappropriately large values for
`t_dispatch_virtual` and `t_finish_virtual`).  Such a queue can have
an arbitrarily long burst of inappropriate lossage to other queues.

### Support for LIST requests

Up until now, we were assuming that even though the requests aren't
necessarily equally expensive, their actual cost is actually greatly
reflected by the time it took to process them.  But while being processed
each of them is consuming the equal amount of resources.

It works well for requests that are touching only a single object.
However, given the fact that in practise the concurrency limits has to be
set much higher than number of available cores to achieve reasonable system
throughput, this no longer works that well for LIST requests that are orders
of magnitude more expensive.  There are two aspects of that:
- for CPU the hand-wavy way of rationalizing it is that he ratio of time
  the request is processed by the processor to the actual time of processing
  the request starts to visibly differ (e.g. due to I/O waiting time -
  there is communication with etcd in between for example).
- for memory the reasoning is more obvious as we simply keep all elements
  that we process in memory

As a result, kube-apiserver (and etcd) may be able to easily keep with N
simple in-flight requests (e.g. create or get a single Pod), but will explode
trying to process N requests listing all the pods in the system at the same
time.

#### Width of the request

In order to address this problem, we are introducing the concept of `width`
of the request.  Instead of saying that every request is consuming a single
unit of concurrency, we allow for a request to consume `<width>` units of
concurrency while being processed.

This basically means, that the cost of processing a given request is no
longer reflected by its `<processing latency>` and instead its cost is now
equal to `<width> x <processing latency>`.  The rationale behind it is that
the request is now consuming `<width>` concurrency units for the duration
of its processing.

While in theory the `width` can be an arbitrary non-integer number, for
practical reasons, we will assume it actually is an integer.  Given that
our estimations here are very rough anyway that seems a reasonable
simplification that makes dispatching the budget a bit simpler.

#### Determining the width

While one can imagine arbitrarily sophisticated algorithms for it (including
exposing defining the width of requests via FlowSchema API), we want to start
with something relatively simple to first get operational experience with it
before investing into sophisticated algorithms or exposing a knob to users.

In order to determine the function that will be approximating the `width` of
a request, we should first estimate how expensive a particular request is.
And we need to think about both dimensions that we're trying to protect from
overloading (CPU and RAM) and how many concurrency units a request can actually
consume.

Let's start with CPU.  The total cost of processing a LIST request should be
proportional to the number of processed objects.  However, given that in
practice processing a single request isn't parallelized (and the fact that
we generally scale the number of total concurrency units linearly with amount
of available resources), a single request should consume no more than A
concurrency units.  Fortunately that all compiles together because the
`processing latency` of the LIST request is actually proportional to the
number of processed objects, so the cost of the request (defined above as
`<width> x <processing latency>` really is proportional to the number of
processed objects as expected.

For RAM the situation is actually different.  In order to process a LIST
request we actually store all objects that we process in memory. Given that
memory is uncompressable resource, we effectively need to reserve all that
memory for the whole time of processing that request.  That suggests that
the `width` for the request from the RAM perspective should be proportional
to the number of processed items.

So what we get is that:
```
  width_cpu(N) = min(A, B * N)
  width_ram(N) = D * N
```
where N is the number of items a given LIST request is processing.

The question is how to combine them to a single number.  While the main goal
is to stay on the safe side and protect from the overload, we also want to
maximize the utilization of the available concurrency units.
Fortunately, when we normalize CPU and RAM to percentage of available capacity,
it appears that almost all requests are much more cpu-intensive.  Assuming
4GB:1CPU ratio and 10kB average object and the fact that processing larger
number of objects can utilize exactly 1 core, that means that we need to
process 400.000 objects to make the memory cost higher.
This means, that we can afford the potential minor efficiency that extremely
large requests would cause and just approximate it by protecting every resource
independently, which translates to the following function:
```
  width(n) = max(min(A, B * N), D * N)
```
We're going to better tune the function based on experiments, but based on the
above back-of-envelope calculations showing that memory should almost never be
a limiting factor we will approximate the width simply with:
```
width_approx(n) = min(A, ceil(N / E)), where E = 1 / B
```
Fortunately that logic will be well separated and purely in-memory so we
can decide to arbitrarily adjust it in future releases.

[TODO: describe how `N` is estimated.]

Given that the estimation is well separated piece of logic, we can decide
to replace with much more sophisticated logic later (e.g. whether it is
served from etcd or from cache, whether it is namespaced or not, etc.).

One more important aspect to resolve is what happens if a given priority
level doesn't have enough concurrency units assigned to it. To be on the
safe side we should probably implement borrowing across priority levels.
However, given we don't want to block introducing the `width` concept on
design and implementation of borrowing, until this is done we have two
main options:
- cap the `width` at the concurrency units assigned to the priority level
- reject requests for which we won't be able to allocate enough concurrency
  units

To avoid breaking some users, we will proceed with the first option (when
computing the cap we should also report requests that we believe are too
wide for a given priority level - it would allow operators to adjust configs).
That said, to accommodate for the inaccuracy here we will introduce a concept
of `additional latency` for a request.  This basically means that after the
request finishes in a real world, we still don't mark it as finished in
the virtual world for `additional latency`.
Adjusting virtual time of a queue to do that is trivial.  The other thing
to tweak is to ensure that the concurrency units will not get available
for other requests for that time (because currently all actions are
triggered by starting or finishing some request).  We will maintain that
possibility by wrapping the handler into another one that will be sleeping
for `additional latence` after the request is processed.

Note that given the estimation for duration of processing the requests is
automatically corrected (both up and down), there is no need to change that
in the initial version.

#### Dispatching the request, as a modification to the initial Fair Queuing for Server Requests

The hardest part of adding support for LIST requests is dispatching the
requests.  Now in order to start processing a request, it has to accumulate
`<width>` units of concurrency.

The important requirement to recast now is fairness.  As soon a single
request can consume more units of concurrency, the fairness is
no longer about the number of requests from a given queue, but rather
about number of consumed concurrency units.  This justifies the above
definition of adjusting the cost of the request to now be equal to
`<width> x <processing latency>` (instead of just `<processing latency>`).

At the same time, we want to maximally utilize the available capacity.
In other words, we want to minimize the time when some concurrency unit
is not used, but there are requests at a given PL that could use it.

In order to achieve the above goals, we are introducing the following
modification to the current dispatching algorithm:
- as soon as we choose the request to dispatch (i.e. the queue from which
  the first request should be dispatched), we start accumulating concurrency
  units until we accumulate `<width>` and only then dispatch the request.
  In other words, if the chosen request has width `<width>` and there
  are less then `<width>` available seats, we don't dispatch any other request
  (at a given priority level) until we will have `<width>` available seats
  at which point we dispatch this request.
  Such approach (as opposed to dispatching individual concurrency units
  independently one-by-one) allows us to not waste too many seats and avoid
  deadlocks if we would be dispatching seats to multiple LIST requests
  without having enough of them for a given priority level.
- however, to ensure fairness (especially over longer period of times)
  we need to change how virtual time is advanced too.  We will change the
  semantics of virtual time tracked by the queues to correspond to work,
  instead of just wall time.  That means when we estimate a request's
  virtual duration, we will use `estimated width x estimated latency` instead
  of just estimated latency.  And when a request finishes, we will update
  the virtual time for it with `seats x actual latency` (note that seats
  will always equal the estimated width, since we have no way to figure out
  if a request used less concurrency than we granted it).

However, now the queueing mechanism also requires adjustment.  So far,
when putting a request into the queue, we were choosing the shortest queue.
It worked, because it well proxied the total cost of processing all requests
in that queue.
After the above changes, the size of the queue is no longer correctly
approximating the cost of processing the queued items.  Given that the
total cost of processing a request is now `<width> x <processing latency>`,
the weight of the queue should now reflect that.

### Support for WATCH requests

The next thing to consider is support for long-running requests.  However,
solving it in a generic case is hard, because we don't have any way to
predict how expensive those requests will be.  Moreover, for requests like
port forwarding this is completely outside our control (being application
specific).  As a result, as the first step we're going to limit our focus
to just WATCH requests.

However, even for WATCH requests, there are effectively two separate problems
that has to be considered and addressed.

#### Watch initialization

The first problem is initialization of watch requests.  In this phase we
initialize all the structures, start goroutines, etc. and sync the watch
to "now".  Given that watch requests may be called with potentially quite
old resource version, we may need to go over many events, filter them out
and send the ones that are matching selectors.

We will solve this problem by also handling watch requests by our priority
and fairness kube-apiserver filter.  The queueing and admitting of watch
requests will be happening exactly the same as for all non-longrunning
requests.  However, as soon as watch is initialized, it will be sending
an artificial `finished` signal to the APF dispatcher - after receiving this
signal dispatcher will be treating the request as already finished (i.e.
the concurrency units it was occupying will be released and new requests
may potentially be immediately admitted), even though the request itself
will be still running.

However, there are still some aspects that require more detail discussion.

##### Getting the initialization signal

The first question to answer is how we will know that watch initialization
has actually been done.  However, the answer for this question is different
depending on whether the watchcache is on or off.

In watchcache, the initialization phase is clearly separated - we explicitly
compute `init events` and process them.  What we don't control at this level
is the process of serialization and sending out the events.
In the initial version we will ignore this and simply send the `initialization
done` signal as soon as all the init events are in the result channel.  In the
future, we can estimate it by assuming some constant (or proportional to
event size) time for an event and delay delivering the signal by the sum of
those across all init events.

If watchcache is disabled, the watch is just proxied to the etcd.  In this
case we don't have an easy way to answer the question whether the watch has
already catch up.  Given that watchcache is enabled by default for all
resources except from events, in the initial version we will simply deliver
the `initialization done` signal as soon as etcd watch is started.  We will
explore how to improve it in the future.

##### Passing the initialization signal to the filter

The second aspect is passing the `initialization done` signal from watchcache
or etcd3 storage layer to the priority and fairness filter.  We will do that
using a synchronization primitives (either channel or WaitGroup) and putting
that into request context.  Given that context is passed down with the request,
watchcache/etcd3 storage will have access to it and will be able to pass the
signal.

##### Width of the request

The performance characteristics of the watch initialization is very different
than processing a regular single-object request, we should (similarly as for
LIST requests) adjust the `width` of the request.  However, in the initial
version, we will just use `width=1` for all watch requests.  In the future,
we are going to evolve it towards a function that will be better estimating
the actual cost (potentially somewhat similarly to how LIST requests are done)
but we first need to a mechanism to allow us experiment and tune it better.

#### Keeping the watch up-to-date

Once the watch is initialized, we have to keep it up-to-date with incoming
changes.  That basically means that whenever some object that this particular
watch is interested in is added/updated/deleted, and appropriate event has
to be sent.

Similarly as in generic case of long-running requests, predicting this cost
up front is impossible. Fortunately, in this case we have a full control over
those events, because those are effectively a result of the mutating requests
that out mechanism is explicitly admitting.
So instead of associating the cost of sending watch events to the corresponding
WATCH request, we will reverse the situation and associate the cost of sending
watch events to the mutating request that triggering them.

In other words, we will be throttling the write requests to ensure that
apiserver would be able to keep up with the watch traffic instead of throttling
the watchers themselves.  The main reason for that is that throttling watchers
themselves isn't really effective: we either need to send all objects to them
anyway or in case we close them, they will try to resume from last received
event anyway.  Which means we don't get anything on throttling them.

##### Estimating cost of the request

Let's start with an assumption that sending every watch event is equally
expensive.  We will discuss how to generalize it below.

With the above assumption, a cost of a mutating request associated with
sending watch events triggered by it is proportional to the number of
watchers that has to process that event.   So let's describe how we can
estimate this number.

Obviously, we can't afford going over all watches to compute that - we need
to keep this information already precomputed. What if we would simply
store an in-memory map from (resource type, namespace, name) tuple into
number of opened watches that are interested in a given event.  The size of
that map won't be larger then the total number of watches, so that is
acceptable.
Note that each watch can also specify label and field selectors.  However,
in most of cases a particular object has to be processed for them anyway
to check if the selectors are satisfied.  So we can ignore those selectors
as the object contributes to the cost (even if it will not be send as not
satisfying the selector).
The only exception to this is caused by a few predefined selectors that
kube-apiserver is optimized for (this includes pods from a given node and
nodes/secrets/configmaps specifying metadata.name field selector).  Given
their simplicity, we can extend our mapping to handle those too.

Having such in-memory map, we can quickly estimate the cost of a request.
It's not as simple as taking a single map item as it requires adding watches
for the whole namespace and all objects of a given type.  But it be done in
O(1) map accesses.
Keeping such map up-to-date is also easy - whenever a new watch start we
increment a corresponding entry, when it ends we decrement it.

##### Multiple apiservers

All the above works well in the case of single kube-apiserver.  But if there
are N kube-apiservers, there is no guarantee that the number of watches are
evenly distributed across them.

To address the problem, individual kube-apiservers has to publish the
information about number of watches for other kube-apiservers.  We obviously
don't want to introduce a new communications channel, so that can be done
only by writing necessary information to the storage layer (etcd).
However, writing a map that can contain tens (or hundreds?) of thousands of
entries wouldn't be efficient. So we need to smartly hash that to a smaller
structure to avoid loosing too much information.
If we would have a hashing function that can combine only a similar buckets
(e.g. it won't combine "all Endpoints" bucket with "pods from node X") then
we can simply write maximum from all entries that are hashed to the same value.
This means that some costs may be overestimated, but if we reasonably hash
requests originating by system components, that seems acceptable.
The above can be achieved by hashing each resource type to a separate set of
buckets, and within a resource type hashing (namespace, name) as simple as:
```
  hash(ns, name) = 0                              if namespace == "" & name == ""
  hash(ns, name) = 1 + hash(namespace)%A          if name == ""
  hash(ns, name) = 1 + A + hash(namespace/name)%B otherwise
```
For small enough A and B (e.g. A=3, B=6), the representation should have less
than 1000 entries, so it would be small enough to make periodic updates in etcd
reasonable.

We can optimize amount of data written to etcd by frequently (say once per
second) checking what changed, but writing rarely (say once per minute) or
if values in some buckets significantly increased.
The above algorithm would allow us to avoid some more complicated time-smearing,
as whenever something quickly grows we report it, but we don't immediately
downscale which is a way to somehow incorporate a history.

However, we will treat the above as a feasibility proof.  We will just start
with the simplest approach of treating each kube-apiserver independently.
We will implement the above (i.e. knowledge sharing between kube-apiserver),
if the independence assumption will not work good enough.
The above description shows that it won't result in almost any wasted work
if the code will be well structured.

##### Cost of the watch event

We assumed above that the cost of processing every watch event is equal.
However, in practice the cost associated with sending an event consists
of two main parts:
- the cost of going through event change logic
- the cost of processing the event object (e.g. deserialization or sending data
  over network)
The first one is close to equal independently of the event, the second one
is more proportional to the size of the object.
However, estimating size of the object is hard to predict for PATCH or DELETE
requests. Additionally, even for POST or PUT requests where we could potentially
estimate it based on size of the body of the requests, we may not yet have
access to it when we need to make a decision.

One way to estimate it would be to keep a running average of watch event size
per bucket.  While it won't give us an accurate estimate, it should amortize
well over time.

Obviously some normalization will be needed here, but it's impossible to assess
it on paper, and we are leaving this for the implementation and tuning phase
to figure out the details.

##### Dispatching the request, as a modification to the initial Fair Queuing for Server Requests with LIST support

We described how we can estimate the cost of the request associated with
watch events triggerred by it.  But we didn't yet said how this translates
to dispatching the request.

First of all, we need to decide how to translate the estimated cost to the
`width` of the request and its latency.  Second, we need to introduce the
changes to our virtual world, as the fact that the request finished doesn't
mean that sending all associated watch events has also finished (as they are
send asynchronously).

Given that individual watch events are to significant extent processed
independently in individual goroutines, it actually makes sense to adjust the
`width` of the request based on the expected number of triggerred events.
However, we don't want to inflate the width of every single request that is
triggering some watch event (as described in the above sections, setting
the width to greater is reducing our ability to fully utilize our capacity).
The exact function to computing the width should be figured out during
further experiments, but the initial candidate for it would be:
```
  width(request) = min(floor(expected events / A), concurrency units in PL)
```

However, adjusting the width is not enough because, as mentioned above,
processing watch events happens asynchronously.  As a result, we will use
the mechanism of `additional latency` that is described in the section
about LIST requests to compensate for asynchronous cost of the request
(which in virtual world quals `<width> x <additional latency>`.

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
(`ServerCL`) and will settle down to the configured limit as requests
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
- `apiserver_flowcontrol_request_concurrency_limit` (gauge of NominalCL, broken down by priority)
- `apiserver_flowcontrol_request_min_concurrency_limit` (gauge of MinCL, broken down by priority)
- `apiserver_flowcontrol_request_max_concurrency_limit` (gauge of MaxCL, broken down by priority)
- `apiserver_flowcontrol_request_current_concurrency_limit` (gauge of CurrentCL, broken down by priority)
- `apiserver_flowcontrol_demand_seats` (timing ratio histogram of seat demand / NominalCL, broken down by priority)
- `apiserver_flowcontrol_demand_seats_high_water_mark` (gauge of HighSeatDemand, broken down by priority)
- `apiserver_flowcontrol_demand_seats_average` (gauge of AvgSeatDemand, broken down by priority)
- `apiserver_flowcontrol_demand_seats_stdev` (gauge of StDevSeatDemand, broken down by priority)
- `apiserver_flowcontrol_envelope_seats` (gauge of EnvelopeSeatDemand, broken down by priority)
- `apiserver_flowcontrol_smoothed_demand_seats` (gauge of SmoothSeatDemand, broken down by priority)
- `apiserver_flowcontrol_target_seats` (gauge of Target, brokwn down by priority)
- `apiserver_flowcontrol_seat_fair_frac` (gauge of FairProp)

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

GA:

- Satisfaction with LIST and WATCH support.
- API annotations properly support strategic merge patch.
- APF allows us to disable client-side rate limiting without causing the apiservers to wedge/crash.  Note that there is another level of concern that APF does not attempt to address, which is mismatch between the throughput that various controllers can sustain.
- Design and implement borrowing between priority levels.
- Satisfaction that the API is sufficient to support auto-tuning of capacity and resource costs.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?** To enable
  priority and fairness, all of the following must be enabled:
  - [x] Feature gate
    - Feature gate name: APIPriorityAndFairness
    - Components depending on the feature gate:
      - kube-apiserver
  - [x] Command-line flags
    - `--enable-priority-and-fairness`, and
    - `--runtime-config=flowcontrol.apiserver.k8s.io/v1alpha1=true`

* **Does enabling the feature change any default behavior?** Yes, requests that
  weren't rejected before could get rejected while requests that were rejected
  previously may be allowed. Performance of kube-apiserver under heavy load
  will likely be different too.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?** Yes.

* **What happens if we reenable the feature if it was previously rolled back?**
  The feature will be restored.

* **Are there any tests for feature enablement/disablement?** No. Manual tests
  will be run before switching feature gate to beta.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?** A
  misconfiguration could cause apiserver requests to be rejected, which could
  have widespread impact such as: (1) rejecting controller requests, thereby
  bringing a lot of things to a halt, (2) dropping node heartbeats, which may
  result in overloading other nodes, (3) rejecting kube-proxy requests to
  apiserver, thereby breaking existing workloads, (4) dropping leader election
  requests, resulting in HA failure, or any combination of the above.

* **What specific metrics should inform a rollback?** An abnormal spike in the
  `apiserver_flowcontrol_rejected_requests_total` metric should potentially be
  viewed as a sign that kube-apiserver is rejecting requests, potentially
  incorrectly. The `apiserver_flowcontrol_request_queue_length_after_enqueue`
  metric getting too close to the configured queue length could be a sign of
  insufficient queue size (or a system overload), which can be precursor to
  rejected requests.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  No. Manual tests will be run before switching feature gate to beta.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
  fields of API types, flags, etc.?** Yes, `--max-requests-inflights` will be
  deprecated in favor of APF.

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
  If the `apiserver_flowcontrol_dispatched_requests_total` metric is non-zero,
  this feature is in use. Note that this isn't a workload feature, but a
  control plane one.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [x] Metrics
    - Metric name: `apiserver_flowcontrol_request_queue_length_after_enqueue`
    - Components exposing the metric: kube-apiserver

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  No SLOs are proposed for the above SLI.

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?** No.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  No.

### Scalability

* **Will enabling / using this feature result in any new API calls?** Yes.
  Self-requests for new API objects will be introduced. In addition, the
  request execution order may change, which could occasionally increase the
  number of retries.

* **Will enabling / using this feature result in introducing new API types?**
  Yes, a new flowcontrol API group, configuration types, and status types are
  introduced. See `k8s.io/api/flowcontrol/v1alpha1/types.go` for a full list.

* **Will enabling / using this feature result in any new calls to the cloud
  provider?** No.

* **Will enabling / using this feature result in increasing size or count of
  the existing API objects?** No.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs]?** Yes, a non-negligible latency
  is added to API calls to kube-apiserver. While [preliminary tests](https://github.com/tkashem/graceful/blob/master/priority-fairness/filter-latency/readme.md)
  shows that the API server latency is still well within the existing SLOs,
  more thorough testing needs to be performed.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?** The proposed
  flowcontrol logic in request handling in kube-apiserver will increase the CPU
  and memory overheads involved in serving each request. Note that the resource
  usage will be configurable and may require the operator to fine-tune some
  parameters.

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**
  The feature is itself within the API server. Etcd being unavailable would
  likely cause kube-apiserver to fail at processing incoming requests.

* **What are other known failure modes?** A misconfiguration could reject
  requests incorrectly. See the rollout and monitoring sections for details on
  which metrics to watch to detect such failures (see the `kep.yaml` file for
  the full list of metrics). The following kube-apiserver log messages could
  also indicate potential issues:
  - "Unable to list PriorityLevelConfiguration objects"
  - "Unable to list FlowSchema objects"

* **What steps should be taken if SLOs are not being met to determine the
  problem?** No SLOs are proposed.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History


- v1.19: `Alpha` release
- v1.20: graduated to `Beta`
- v1.22: initial support for width concept and watch initialization
- v1.23: introduce `v1beta2` API
  - no changes compared to `v1beta1`
  - `v1beta1` remain as storage version
- v1.24: storage version changed to `v1beta2`

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
