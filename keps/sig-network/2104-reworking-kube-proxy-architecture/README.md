<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-2104: rework kube-proxy architecture

# Index
<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [How we calculate deltas: The DiffStore](#how-we-calculate-deltas-the-diffstore)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
    - [Story 4](#story-4)
    - [Story 5](#story-5)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
  - [Server](#server)
  - [Client](#client)
  - [Backends](#backends)
    - [Fullstate Logic](#fullstate-logic)
    - [Filterreset logic](#filterreset-logic)
  - [Test Plan](#test-plan)
    - [Automation for the standard service proxy scenarios](#automation-for-the-standard-service-proxy-scenarios)
    - [Manual verification of complex scenarios](#manual-verification-of-complex-scenarios)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

At the beginning, `kube-proxy` was designed to handle the translation of Service objects to OS-level
resources.  Implementations included userspace, followed by iptables, and ipvs. With the growth of the
Kubernetes project, more implementations came to life, for instance with eBPF, and often in relation
to other goals (Calico to manage the network overlay, Cilium to manage app-level security, metallb
to provide an external LB for bare-metal clusters, etc).

Along this cambrian explosion of third-party software, the Service object itself received new
concepts to improve the abstraction, for instance to express topology. Thus, third-party
implementations are expected to update and become more complex over time, even if their core doesn't
change (ie, the eBPF translation layer is not affected).

This KEP is born from the conviction that more decoupling of the Service object and the actual
implementations is required, by introducing an intermediate, node-level abstraction provider. This
abstraction is expected to be the result of applying Kubernetes' `Service` semantics and business
logic to a simpler, more stable API.

## Motivation

<!--
This section is for explicitly listing the motivation, goals and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

Please note that the kube-proxy working group now has a [working implementation](https://github.com/kubernetes-sigs/kpng)
for most of the below goals. The implementation has CI, conformance testing,
and fine grained sig-network tests passing for the most critical use cases, including 
both linux and windows.

- Design a new architecture for service proxy implementations
  consisting of:

    - A "core" service proxy process that models the networking statespace
      of Kubernetes, and thus all the non-backend-technology-specific aspects of service proxying
      (e.g., determining which endpoints should be available on a
      given node, given traffic policy, topology, and pod readiness
      constraints)

    - A proper subset of "proxy backends" mapping to the current upstream proxy,
      which communicate with the "brain" to acquire it's local networking state space
      to implement technical details of writing backend routing rules
      from services to pods (eg. iptables, ipvs, Windows kernel).

    - A gRPC API for optimal communication between the core logic
      daemon and the backend implementations, which can be run in memory
      or externalized (so that the brain can run with a one-many relationship
      with "proxy backends"

    - A set of golang packages for working with the gRPC API, which   third parties
      can use to create their own proxy backend implementations out of tree.

- Provide an implementation of the core service proxy logic daemon.
  (This implementation will be used, unmodified, by all core and
  third-party proxy backend implementations.)

- Provide a golang library that consumes the gRPC API from the above
  daemon and exports the information to the proxy backend
  implementation in one of two forms:

    - a "full state" model, in which the library keeps track of the
      full state of Kubernetes network statespace on this node, and
      sends the proxy implementation a new copy of the full state on every
      update. The library will also include a package called
      "diffstore" that is designed to make it very easy for
      implementations to generate incremental updates based on the
      full state information.

    - an "incremental" model, in which the library simply passes on
      the updates from the gRPC API so the proxy implementation can
      update its own model, which allows for similar behavior to the
      current upstream kube-proxy.

- Provide additional reusable library elements for some shared backend logic,
  such as conntrack cleaning, which might be called from different
  proxy implementations.

- Provide new implementations of the existing "standard" proxy
  implementations (iptables, ipvs, and Windows kernel), based on the
  new daemon and client library. At least one of these will use the
  "full state" model and at least one will use the "incremental" model.

- Deprecate and eventually remove the existing proxy implementations
  in `k8s.io/kubernetes`, in favor of the new implementations. Also
  remove the associated support packages in `k8s.io/kubernetes` that
  are only used by kube-proxy (eg, `pkg/util/ipvs`,
  `pkg/util/conntrack`, `pkg/util/netsh`).

- Provide initial material that demonstrates how to run this decoupled proxy implementation on
separate nodes (i.e. with the "core service proxy brain" on *one* node, and a backend(s) on
other nodes, where all of the K8s networking state space is sent, remotely over GRPC).

### Non-Goals

- We Won't necessarily provide bulletproof NFT, eBPF, Userspace backends with parity to the core Windows kernel, IPTAbles, IPVS implementations.  
- We Won't Require KPNG to run in a mode where the kpng "core logic brain" is a separate process from the kpng "proxy backends", we state this as non goal because it's been a bit of a red herring debate in the past.  Although KPNG support this, it's not required. 
- We won't Require all proxiers to use the fullstate model.  However, this is ideal for new implementations, we think because it's easier to read and understand.

## Proposal

Decoupling of the KPNG core logic from networking backends, with the option to run these components in completely separate processes, or, together with the same shared memory. In cases where they are decoupled at the process level, one may, for example, use "localhost" as the gRPC API provider that will be accessible as usual via TCP (`127.0.0.1:12345`) and/or via a socket (`unix:///path/to/proxy.sock`).  In cases where they run together in memory, the same communication will happen, but the GRPC calls will just be local.

- it will connect to the API server and watch resources, like the current proxy;
- then, it will process them, applying Kubernetes specific business logic like topology computation
  relative to the local host;
- finally, provide the result of this computation to client via a gRPC watchable API.

We assume that some backend's will benefit from running the "core" Kubernetes logic in a way that is aligned to releases of Kubernetes , while rapidly upgrading their backend logic out-of-band from this.  This implementation allows that if one so chooses, although we expect most "stable" proxying implementations won't be upgrading at a more frequent clip then Kubernetes itself.

In this implementation, we either: 
- send the *full state*  of the kubernetes networking state-space to a client, every time anything needs to change.  Since this is done over GRPC or in memory, the bandwidth costs are low (anecdotally this has been measured, and it works for 1000s of services and pods - we can attach specific results to this KEP as needed).  An example of this can be seen in the ebpf and nft proxies in the KPNG project (https://github.com/kubernetes-sigs/kpng/blob/master/backends/nft/nft.go).  Some initial performance data is here https://github.com/kubernetes-sigs/kpng/blob/master/doc/proposal.md.
```
// the entire statespace of the Kubernetes networking model is embedded in this struct
type ServiceEndpoints struct {
	Service   *localnetv1.Service
	Endpoints []*localnetv1.Endpoint
}
```

- send the *incremental state* of the kubernetes networking state-space whenever an event occurs. We then allow backend clients to implement "SetService" and "SetEndpoint" methods, which allow them to use a similar API structure to that of upstream Kubernetes current IPTables Kube-proxy.  An example of this is in how KPNG currently implements the iptables proxy (https://github.com/kubernetes-sigs/kpng/blob/master/backends/iptables/sink.go).

```
func (s *Backend) SetService(svc *localnetv1.Service) {
	for _, impl := range IptablesImpl {
		// since iptables is a commonly used kubernetes proxying implementation
		// we kept the serviceChanges cache and just wrapped it under SetService
		impl.serviceChanges.Update(svc)
	}
}

```

Thus, we send the full state to a backend "client", such that the backend won't have to do
diff-processing and maintain a full cache of proxy data. This should provides simpler backend implementations,
reliable results and still be quite optimal, since many kernel network-level objects are
updated via atomic replace APIs. It's also a protection from slow readers, since no stream has to
be buffered.

Since the node-local state computed by the new proxy will be simpler and node-specific, it will
only change when the result for the current node is actually changed. Since there's less data in
the local state, change frequency is reduced compared to cluster state. Testing on actual clusters
showed a frequency reduction of change events by 2 orders of magnitude.

#### How we calculate deltas: The DiffStore

One fundamentally important part of building a service proxy in kubernetes is calculating "diffs", for example, if at time 1 we have

```
Service A -> Pod A1 , Pod A2
Service B -> Pod B1
```
and at time 2, we have
```
Service A -> Pod A1 , Pod A2
Service B -> Pod B1, Pod B2
```

We need to add *One* new networking rule: the fact that there is service B which can be loadbalanced to pod B2.  Any other networking rules already exist and need not be processed (this is more true for some backends then others, i.e. for IPVS or the windows kernel, which don't require rewriting of all rules every time there's a change).  

KPNG provides a "DiffStore" library, which allows arbitrary, generic go objects to be diffed in memory by a backend.  This can be viewed at https://github.com/kubernetes-sigs/kpng/tree/master/client/diffstore.  The overall usage of this store is relatively intuitive: Write to it continuously, and only register "differences" when looking at the Diffs.  The "Reset()" function causes the second "wave" in a series of writes to take place, such that a subsequent call to see the diff at a later time will reveal differences between the first and second series of writes.  Note that the `Get` call here will write a key if empty.

```
func ExampleStore() {
	store := NewBufferStore[string]()
	{
		fmt.Fprint(store.Get("a"), "hello a")
		store.Done()
		store.printDiff()
	}
	{
		store.Reset()
		fmt.Fprint(store.Get("a"), "hello a")
		store.Done()
		store.printDiff()
	}
```
The entire unit test for the diffstore which is used to cache and update the network state space on the backend side, is shown in the above `/diffstore/diffstore_test.go` file. 


### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a networking technology startup I want to make my own kube-proxy implementation but don't want to maintain the logic of talking to the APIServer, caching its data, or calculating an abbreviated/proxy-focused representation of the Kubernetes networking state space.  I'd like a wholesale framework I can simply plug my logic into. 

#### Story 2

As a Kubernetes maintainer, I don't want to have to understand the internals of a networking backend in order to simulate or write core updates to the logic of the kube-proxy locally.

#### Story 3

As a Kubernetes maintainer, I'd like to add new proxies to kubernetes-sigs repositories which aren't in-tree, but are community maintained and developed/licensed according to CNCF standards

#### Story 4

As an end user, I'd like to be able to easily test a Kubernetes backend's networking logic without plugging it into a real Kubernetes cluster, or maybe even use it to write networking rules that aren't directly provided by the Kubernetes API.

#### Story 5

As a developer I'd like to implement a backend proxy implementation without being dependent on the K8s API, and without creating any load on the Kubernetes API - either in edge networking scenarios, or in high scale scenarios.

### Notes/Constraints/Caveats (Optional)

- sending the full-state could be resource consuming on big clusters, but it should still be O(1) to
  the actual kernel definitions (the complexity of what the node has to handle cannot be reduced
  without losing functionality or correctness).

### Risks and Mitigations

- There's a performance risk when it comes to large scales, we've proposed a new issue https://github.com/kubernetes-sigs/kpng/issues/325 as a community wide, open scale testing session on a large cluster that we can run manually to inspect in real time and see any major deltas.

- There may be magic functionality that is unpublished in the kube-proxy that we don't know about which we lose when doing this.  

Mitigations are - falling back to the in-tree proxy, or simply titrating logic over piece by piece if we find holes .  We don't think there are many of these those because there are 100s of networking tests, many of which test specific items like udp proxying, avoiding blackholes, service updating, scaling of pods, local routing logic for things like service topologies, and so on.

- Story 5, while implementable from a development standpoint to make it easy to hack on new backends, hasn't been broadly tested in a production
context and might need tooling like mTLS and so on, in order to be production ready for clouds and other user facing environments.

## Design Details

A [draft implementation] exists and some [performance testing] has been done.

[draft implementation]: https://github.com/kubernetes-sigs/kpng/
[performance testing]: https://github.com/kubernetes-sigs/kpng/blob/master/doc/proposal.md

### API

The watchable API will be long polling, taking a "last known state info" and returning a stream of
objects. 

Proposed definition is found here: https://github.com/kubernetes-sigs/kpng/blob/master/api/localnetv1/services.proto

The main types composing the GRPC API are:
```
message Service
message IPFilter
message ServiceIPs
message Endpoint
message IPSet
message Port
message ClientIPAffinity
message ServiceInfo
message EndpointInfo
message EndpointConditions
message NodeInfo
message Node
```

### Server

The KPNG server is responsible for watching the Kubernetes API for 
changes to Service and Endpoint objects and translating to listening 
clients via the aforementioned API.

When the proxy server starts, it will generate a random InstanceID, and have Rev at 0. So, a client
(re)connecting will get the new state either after a proxy restart or when an actual change occurs.
The proxy will never send a partial state, only full states. This means it waits to have all its
Kubernetes watchers sync'ed before going to Rev 1.

The first OpItem in the stream will be the state info required for the next polling call, and any
subsequent item will be an actual state object. The stream is closed when the full state has been
sent.


### Client

The client library abstracts those details away and interprets the kpng api events following 
each state change. It includes a default Run function, sets up default flags, parses them and runs
the client, allowing very simple clients like this:

```golang
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mcluseau/kube-proxy2/pkg/api/localnetv1"
	"github.com/mcluseau/kube-proxy2/pkg/client"
)

func main() {
	client.Run(printState)
}

func printState(items []*localnetv1.ServiceEndpoints) {
	fmt.Fprintln(os.Stdout, "#", time.Now())
	for _, item := range items {
		fmt.Fprintln(os.Stdout, item)
	}
}
```

The currently proposed interface for the lower-level client is as follows:

```golang
package client // import "github.com/mcluseau/kube-proxy2/pkg/client"

type EndpointsClient struct {
	// Target is the gRPC dial target
	Target string

	// InstanceID and Rev are the latest known state (used to resume a watch)
	InstanceID uint64
	Rev        uint64

	// ErrorDelay is the delay before retrying after an error.
	ErrorDelay time.Duration

	// Has unexported fields.
}


// DefaultFlags registers this client's values to the standard flags.
func (epc *EndpointsClient) DefaultFlags(flags FlagSet) {
	flags.StringVar(&epc.Target, "api", "127.0.0.1:12090", "API to reach (can use multi:///1.0.0.1:1234,1.0.0.2:1234)")

	flags.DurationVar(&epc.ErrorDelay, "error-delay", 1*time.Second, "duration to wait before retrying after errors")

	flags.IntVar(&epc.MaxMsgSize, "max-msg-size", 4<<20, "max gRPC message size")

	epc.TLS.Bind(flags, "")
}

// Next sends the next diff to the sink, waiting for a new revision as needed.
// It's designed to never fail, unless canceled.
func (epc *EndpointsClient) Next() (canceled bool) {
	if epc.watch == nil {
		epc.dial()
	}

retry:
	if epc.ctx.Err() != nil {
		canceled = true
		return
	}

	// say we're ready
	nodeName, err := epc.Sink.WaitRequest()
	if err != nil { // errors are considered as cancel
		canceled = true
		return
	}

	err = epc.watch.Send(&localnetv1.WatchReq{
		NodeName: nodeName,
	})
	if err != nil {
		epc.postError()
		goto retry
	}

	for {
		op, err := epc.watch.Recv()

		if err != nil {
			// klog.Error("watch recv failed: ", err)
			epc.postError()
			goto retry
		}

		// pass the op to the sync
		epc.Sink.Send(op)

		// break on sync
		switch v := op.Op; v.(type) {
		case *localnetv1.OpItem_Sync:
			return
		}
	}
}

// Cancel will cancel this client, quickly closing any call to Next.
func (epc *EndpointsClient) Cancel() {
	epc.cancel()
}

// CancelOnSignals make the default termination signals to cancel this client.
func (epc *EndpointsClient) CancelOnSignals() {
	epc.CancelOn(os.Interrupt, os.Kill, syscall.SIGTERM)
}

// CancelOn make the given signals to cancel this client.
func (epc *EndpointsClient) CancelOn(signals ...os.Signal) {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, signals...)

		sig := <-c
		klog.Info("got signal ", sig, ", stopping")
		epc.Cancel()

		sig = <-c
		klog.Info("got signal ", sig, " again, forcing exit")
		os.Exit(1)
	}()
}

func (epc *EndpointsClient) Context() context.Context {
	return epc.ctx
}

func (epc *EndpointsClient) DialContext(ctx context.Context) (conn *grpc.ClientConn, err error) {
	klog.Info("connecting to ", epc.Target)

	opts := append(
		make([]grpc.DialOption, 0),
		grpc.WithMaxMsgSize(epc.MaxMsgSize),
	)

	tlsCfg := epc.TLS.Config()
	if tlsCfg == nil {
		opts = append(opts, grpc.WithInsecure())
	} else {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	}

	return grpc.DialContext(epc.ctx, epc.Target, opts...)
}

func (epc *EndpointsClient) Dial() (conn *grpc.ClientConn, err error) {
	if ctxErr := epc.ctx.Err(); ctxErr == context.Canceled {
		err = ctxErr
		return
	}

	return epc.DialContext(epc.ctx)
}
```

### Backends 

The backends make use of the provided client infrastructure to actually program 
networking rules into the kernel datapath. In KPNG the backends interact with the 
client library by first implementing the `backendcmd.Cmd` interface 

```golang
type Cmd interface {
	BindFlags(*pflag.FlagSet)
	Sink() localsink.Sink
}
```

This interface allows each backend to register their own specific set of CLI 
flags, and to define what type of sink they would like to use.  Usually 
the interface is implemented in a file called `register.go` and returned 
via an `init()` function within each backend. 

```golang
type Backend struct {
	localsink.Config
}

func init() {
	backendcmd.Register("to-iptables", func() backendcmd.Cmd { return &Backend{} })
}

func (s *Backend) BindFlags(flags *pflag.FlagSet) {
}

func (s *Backend) Sink() localsink.Sink {
	return filterreset.New(pipe.New(decoder.New(s), decoder.New(conntrack.NewSink())))
}
```

As shown above, a backend's methods define all the functionality (at a high 
level) it needs to function, however they all must implement the `Sink()`, 
which returns a `localsink.Sink` interface, and `BindFlags()` methods.

```golang
type Sink interface {
	// Setup is called once, when the job starts
	Setup()

	// WaitRequest waits for the next diff request, returning the requested node name. If an error is returned, it will cancel the job.
	WaitRequest() (nodeName string, err error)

	// Reset the state of the Sink (ie: when the client is disconnected and reconnects)
	Reset()

	localnetv1.OpSink
}
```

The sink interface is implemented by two different packages, the `filterreset` 
package, which provides methods to give incremental change data to the backends, or 
the `fullstate` package, which simply passes the full-state of the current Services 
and Endpoints to the backend.

In the scope of kubernetes it may be easier to think of the `fullstate` library 
as the package used by implementations who wish to follow level driven controller
constructs, and the `filterreset` library as the package used by implementations 
who wish to follow event driven controller constructs.  

#### Fullstate Logic 

![Alt text](kpng-fullstate-syncer.png?raw=true)

The fullstate library implements the `sink` interface via
a custom `Sink` struct: 

```golang
// EndpointsClient is a simple client to kube-proxy's Endpoints API.
type Sink struct {
	Config    *localsink.Config
	Callback  Callback
	SetupFunc Setup

	data *btree.BTree
}
```

The POC EBPF implementation is a good example of utilizing the fullstate package 
to interact with the KPNG client. Specifically the backend implements three main methods 
`Sink()` to actually create the fullstate sink, and the fullstate
sink's `Callback` and `Setup` functions.  The setup function is called once upon KPNG 
client startup, while the callback function is called anytime the state of 
kubernetes services and endpoints changes.

```golang 
func (s *backend) Setup() {
	ebc = ebpfSetup()
	klog.Infof("Loading ebpf maps and program %+v", ebc)
}

func (b *backend) Sink() localsink.Sink {
	sink := fullstate.New(&b.cfg)

	sink.Callback = fullstatepipe.New(fullstatepipe.ParallelSendSequenceClose,
		ebc.Callback,
	).Callback

	sink.SetupFunc = b.Setup

	return sink
}
```

The `Sink()` method shown above creates a new fullstate sinker via the fullstatepipe 
package.  The fullstatepipe can be configured to send events to the backend in 
three ways: 

```golang
const (
	// Sequence calls to each pipe stage in sequence. Implies storing the state in a buffer.
	Sequence = iota
	// Parallel calls each pipe stage in parallel. No buffering required, but
	// the stages are not really stages anymore.
	Parallel
	// ParallelSendSequenceClose calls each pipe entry in parallel but closes
	// the channel of a stage only after the previous has finished. No
	// buffering required but still a meaningful sequencing, especially when
	// using the diffstore.
	ParallelSendSequenceClose
)
```

The `ebc.Callback`(ebpf controller Callback) function resembles the following: 

```golang 
func (ebc *ebpfController) Callback(ch <-chan *client.ServiceEndpoints) {
	// Reset the diffstore before syncing
	ebc.svcMap.Reset(lightdiffstore.ItemDeleted)

	// Populate internal cache based on incoming fullstate information
	for serviceEndpoints := range ch {
		klog.V(5).Infof("Iterating fullstate channel, got: %+v", serviceEndpoints)

  ...
  // Abbrev. BUSINESS LOGIC
  ...

	// Reconcile what we have in ebc.svcInfo to internal cache and ebpf maps
	// The diffstore will let us know if anything changed or was deleted.
	if len(ebc.svcMap.Updated()) != 0 || len(ebc.svcMap.Deleted()) != 0 {
		ebc.Sync()
	}
}
```

And is what is responsible for programing the actual datapath rules to handle 
service proxying.

#### Filterreset logic 

![Alt text](kpng-filterreset-syncer.png?raw=true)


The `filterreset` library implements the `sink` interface as follows: 

```golang

type Sink struct {
	sink      localsink.Sink
	filtering bool
	memory    map[string]memItem
	seen      map[string]bool
}
```

This definition allows the implementations to construct their own custom sinks 
(see the `sink` field).  A great example of 
utilizing the `filterreset` library can be found in the iptables implementation.

```golang
func (s *Backend) Sink() localsink.Sink {
	return filterreset.New(pipe.New(decoder.New(s), decoder.New(conntrack.NewSink())))
}
```

Here the bakend initializes a new filterreset sink which receives events from the 
shared client via four main methods `SetService`, `DeletService`, `SetEndpoint`, 
`DeleteEndpoint`. 

```golang
func (s *Backend) SetService(svc *localnetv1.Service) {
	for _, impl := range IptablesImpl {
		impl.serviceChanges.Update(svc)
	}
}

func (s *Backend) DeleteService(namespace, name string) {
	for _, impl := range IptablesImpl {
		impl.serviceChanges.Delete(namespace, name)
	}
}

func (s *Backend) SetEndpoint(namespace, serviceName, key string, endpoint *localnetv1.Endpoint) {
	for _, impl := range IptablesImpl {
		impl.endpointsChanges.EndpointUpdate(namespace, serviceName, key, endpoint)
	}

}

func (s *Backend) DeleteEndpoint(namespace, serviceName, key string) {
	for _, impl := range IptablesImpl {
		impl.endpointsChanges.EndpointUpdate(namespace, serviceName, key, nil)
	}
}
```

These methods are ultimately where the iptables rules are programed by
the backend. The main use case for such a sink design was to more easly integrate  
with existing kube proxy backends (iptables, ipvs, etc) which already relied 
on such methods. 

### Test Plan

#### Automation for the standard service proxy scenarios

Upstream Kubernetes has a large set of 100s of tests which leverage service proxies, on different clouds, running
in prow. By "pring" into Kubernetes, we'll get these tests, for free... 

For each of our "completed" backends (iptables, ipvs, nft) KPNG currently runs

- all sig-network tests which involve service proxying
- all Conformance tests

We of course must ensure we pass all the scalability tests which run in PROW default CI,
and we must manually verify KPNG on all standard clouds, and especially, this is
important since cloud kube proxy configurations my leverage command line options/configurations
which arent needed in our CI/kind clusters.

#### Manual verification of complex scenarios 

We assert that some level of performance testing, manually, should be done since this is a
significant architectural change, but we will iterate the details of that later on.

[ x ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.

-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

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

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

*Having an all in one library to help people make kube proxys which are packaged outside of K/K* 

This solution would make it easier to build proxies without actually packaging an end to end solution.  It would however have the setback of *only* being a library, and also, it would force third parties to entirely implement caching solutions.  Finally, it would provide no usability at all to the fully generic "my backend is not in golang" use case, i.e. if someone made a C or java or python backend, they can use KPNG's "brain" as a separate process.  All-in-all, a client library that eases the burden of making proxies is an alternative to this approach but, it would throw away alot of the functionality, modularity, and the "marketplace" aspect of separate backends evolving rapidly with regard to a stable core KPNG "brain".

*Not re writing the kube-proxy and let it live on in pkg/ with new backends emerging over time* 

Because of configuration challenges, community growth challenges in the existing kube-proxy, we'd assert that the modular and easy to evolve/maintain model here is superior to a monolith in-tree.  We cite particularly the eBPF and Windows kube proxy user stories as "home run" scenarios for KPNG, because windows benefits from it's own approaches, and it shouldnt clutter the core K8s codebase, and similarly, eBPF is a bit of a niche' approach which shouldnt live in the Kubernetes core codebase.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
