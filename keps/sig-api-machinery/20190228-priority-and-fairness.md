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

TBD

### Implementation Details/Notes/Constraints

##### Definitions

- *BucketBinding*: a set of rules defining how we bind an inbound request w/ a bucket.
It works similar to \[Cluster\]RoleBinding in the RBAC world.
- *Quota*: a request must acquire one quota before its execution. Basically,
we have two kinds of quota: __reserved__ and __shared__. With reserved quota, the
request can be executed immediately. A bucket's reserved quota is exclusively
consumed by the request matches the bucket, while its shared quota can be consumed
by the request matching higher or the same priority.
- *Bucket*: a quota pool. The quota will be taken from the bucket when it's
acquired by any request and will be returned to the bucket when the request
finishes. A bucket has following attributes:
  - *Priority*: a fixed/preset set of priorities ranged 1...N. The lower
  number means higher priority. Each priority has one or more buckets providing
  quotas and buckets from higher priority can "borrow" shared quotas from
  the lowers.
  - *Reserved Quota*: an initial guaranteed concurrency can be provided by
  the bucket.
  - *Shared Quota*: an initial shared concurrency can be provided by the bucket.
  Shared quota cannot be consumed by requests queued at logically higher
  (or numerically lower) priorities.
  than the bucket's.
  - *Weight*: only works for shared quota. The higher weight a bucket has,
  the more chance the requests hitting the buckets can be executed from the
  queue. Note that a zero weight means that the requests matches the bucket
  cannot be executed by shared quotas, or rather, it can only be executed
  by reserved quotas.

- *Extra Shared Bucket / ESB*: a global logical bucket only providing shared
quotas at a logical "lowest" priority (which is unreachable via external
API requests). Requests doesn't match any priority will be queued in this
logical "lowest" priority until the extra bucket distributes its quota to it.
The extra shared quota can be consumed by requests from all priorities b/c
it lies at the "lowest". In a high level, ESB defines a default common concurrency
in the system and it's supposed to be auto-tunable by a moving average observed
by the system. Note that requests queued in this extra bandwidth are popped
strictly in a FIFO order.

##### Bandwidth Goal

We can define __Shared Bandwidth__ as a goal for priority band __p__, and
we have N fixed priority:

- *Shared Quota(p)*/ SQ: sum of shared quotas provided by
buckets at Priority p.
- *Total Shared Quota*/ TSQ: sum\[ SQ(i) over 1..N \] + ESB.sharedQuota
- *Maximum Shared Quota(p)* / MSQ: TSB - sum\[MSQ(1..p-1)\]

Generally, the system's total shared quota consists of a sum of shared
quotas defined by all buckets and an auto-tunable extra shared quota. A
request w/ higher priority can always be executed prior to the requests
w/ lower priority. Which is, at worst case, when requests from top priority
completely occupies the shared bandwidth, then requests from the rest lower
bands can only be executed under its reserved quota.

##### Overview

Within each priority band, we have a set of buckets matching the priority and
a bucket has its corresponding FIFO request queue. First we find a matched
bucket for the request and put the request into its FIFO queue. The request
will be finally executed until either one reserved quota from the bucket
or one shared quota from the pool is distributed. Note that the reserved quota
is distributed to bucket's FIFO queue straightly but the shared quota is
distributed by polling requests w/ WRR strategy for each priority from higher
to lower.

All requests will be pre-processed before it's actually executed by several stages:

0. We find a matched bucket by applying BucketBinding API.
1. *Enqueue*: The request will be blocked until it's signal'd by one quota.
2. *Distributing*: A daemon queue drainer finally assigns a reserved or
shared quota to the request. Note that reserved quota is assigned prior than
the shared to avoid competition, and this can be implemented by golang select
channels' order.
3. *Executing*: Actually run the delegation handlers.


```
Stages
----
                      Inbound Request
<0>                         |
                            v
----                  ( matched bkt2 )
<1>                         |
----     +------------------|-----------------------+
         | Priority P       |                       |
         |                  v                       |
         |   +-------+   +-------+   +-------+      |                            requests
         |   | Bkt1  |   | Bkt2  |   | Bkt3  |      |   WRR polling               quota
<2>      |   | FIFO  |   | FIFO  |   | FIFO  | .... | <------------- QueueDrainer -----> QuotaTracker
         |   | Queue |   | Queue |   | Queue |      |                                   (tracks usage)
         |   +-------+   +-------+   +-------+      |
         |                  |                       |
         |                  |                       |
----     +------------------|-----------------------+
                            |
<3>                         v
                        Executing
----
```


##### API Model and APIServer flags



```go
type Bucket struct {
    ...

    // ReservedQuota defines a max concurrency of requests which is consumed
    // by the bucket exclusively.
    ReservedQuota int

    // SharedQuota defines a shared concurrency which can be consumed by the
    // buckets at its higher or same priority band.
    SharedQuota int

    // Priority assigns the bucket into a certain priority band.
    Priority PriorityBand // Ranged in 1...N

    // Weight defines how much weight its matched requests have in the queue.
    // Note that a zero weight makes the buckets "unselectable" by WRR algorithm,
    // which is, the requests matching the very bucket can only be executed under
    // reserved quota. This is useful when we want those long-running requests
    // like WATCH, PROXY strictly limitted.
    Weight int
}

type BucketBinding struct {
    ...

    // Rules defines how we bind request to a bucket.
    Rules []BucketBindingRule // WIP

    // BucketRef references a bucket.
    BucketRef *Bucket
}
```

The extra bucket is defaulted by PostStartHook at the server's launching time
just like those RBAC default policies. The priority band of the extra bucket
is fixed to "Lowest" which is unreachable for other user-defined buckets.

##### Implementation

See a poc implementation at: [yue9944882/inflight-rate-limit](https://github.com/yue9944882/inflight-rate-limit)

This working system works as a filter handler in generic apiserver, so it
works both for kube-apiserver and aggregated apiservers:
- The filter list-watches *Bucket* and *BucketBinding* API models and reloads
 the buckets dynamically on events.
on notification of changes.
- The filter consists of one *quota tracker* and one *queue drainer*:
  - *quota tracker*: list-watches *Bucket* API models and records status
  of each bucket's reserved/shared quota in a map. The bucket's quota will
  decrease one if any request claim the quota and being executed/queued.
  And as a request completely finishes, its quota will be returned to the
  bucket by increasing one. quota tracker keeps reading latest status of
  each bucket and compare them w/ their last recorded status to distribute
  the quota as soon as any quota is returned.
  - *queue drainer*: works by polling requests from each priority and requesting
  quota from tracker. It polls queuing requests from queues in an order
  of "from priority 1 to priority p". Requests in the queue of priority
  p are also weighted as defined in their bucket, and queue drainer are
  taking request from the queue by applying WRR algorithm. Note that signal
  from ESQ will be regarded as an "always-lowest" priority, so we poll all
  queues in the system.
  - *ESQ auto-tuner*: observes system loads by recoding a moving average.
  It adjusts ESQ in a certain range to optimize the resource utilization.

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
