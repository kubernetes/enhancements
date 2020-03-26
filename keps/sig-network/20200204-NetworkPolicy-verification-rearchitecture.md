---
title: Rearchitecting NetworkPolicy tests with a DSL for better upstream test coverage
authors:
  - "@jayunit100"
  - "@abhiraut"
  - "@sedefsavas"
  - "@McCodeman"
  - "@mattfenwick"
owning-sig: sig-network
reviewers:
  - "bowie@"
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2020-02-04
last-updated: 2020-03-05
status: implementable
---

Note that this approach of higher level DSLs for testing may be moved into sig-testing for a broader set of tests over time.

# Architecting NetworkPolicy tests with a DSL for better upstream test coverage of all CNIs

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-goals](#non-goals)
  - [Related issues](#related-issues)
    - [Also related but not directly addressed in this KEP](#also-related-but-not-directly-addressed-in-this-kep)
  - [Consequences of this problem](#consequences-of-this-problem)
- [A high level outline of Pod Traffic Pathways](#a-high-level-outline-of-pod-traffic-pathways)
- [Detailed examples of the problem statement](#detailed-examples-of-the-problem-statement)
  - [Incompleteness](#incompleteness)
    - [TODO Temporal tests](#todo-temporal-tests)
    - [Other concrete examples of incompleteness](#other-concrete-examples-of-incompleteness)
    - [List of missing functional test cases](#list-of-missing-functional-test-cases)
  - [Understandability](#understandability)
  - [Extensibility](#extensibility)
  - [Performance](#performance)
    - [Logging verbosity is worse for slow tests](#logging-verbosity-is-worse-for-slow-tests)
  - [Documentation](#documentation)
- [Goals](#goals-1)
- [Implementation History](#implementation-history)
  - [Part 1: Defining a static matrix of ns/pod combinations](#part-1-defining-a-static-matrix-of-nspod-combinations)
  - [Note on Acceptance and Backwards compatibility](#note-on-acceptance-and-backwards-compatibility)
- [Next steps](#next-steps)
- [Thoughts from initial research in this proposal (future KEPs)](#thoughts-from-initial-research-in-this-proposal-future-keps)
  - [Node local traffic: Should it be revisited in V2 ?](#node-local-traffic-should-it-be-revisited-in-v2-)
  - [Validation and 'type' : Should it be revisited ?](#validation-and-type--should-it-be-revisited-)
  - [Finding comprehensive test cases for policy validation](#finding-comprehensive-test-cases-for-policy-validation)
  - [Ensuring that large policy stacks evaluate correctly](#ensuring-that-large-policy-stacks-evaluate-correctly)
  - [Ensure NetworkPolicy evaluates correctly regardless of the order of events](#ensure-networkpolicy-evaluates-correctly-regardless-of-the-order-of-events)
  - [Generating the reachability matrix on the fly](#generating-the-reachability-matrix-on-the-fly)
  - [Consuming network policies as yaml files](#consuming-network-policies-as-yaml-files)
- [Alternative solutions to this proposal](#alternative-solutions-to-this-proposal)
    - [Keeping the tests as they are and fixing them one by one](#keeping-the-tests-as-they-are-and-fixing-them-one-by-one)
    - [Building a framework for NetworkPolicy evaluation](#building-a-framework-for-networkpolicy-evaluation)
    - [Have the CNI organization create such tests](#have-the-cni-organization-create-such-tests)
<!-- /toc -->

## Summary
This proposal suggests that we create and maintain a domain-specific language (DSL) for defining NetworkPolicies against connectivity truth tables, so we can automate positive and negative control tests to address the opportunities for improvement in the performance and adherence to Kubernetes network policy standards of CNI plugins.  We propose that the current NetworkPolicy test suite comprises 25 tests which can take 30 minutes to 1 hour to run, and this time period will be dramatically improved, while increasing test coverage dramatically as well, by following this approach - and initial tests corroborate the findings of this proposal.  In summary, this involves:

- Defining (redefining in some cases) the common set of test scenarios for all network policy tests and increasing performance by reusing a set of containers.
- Rearchitecting network policy tests to enhance readibility and reusability.
- Improve coverage for NetworkPolicy functional tests, and making them more hackable.
- Introduce time to conversion tests to measure performance against perturbed state at scale.

## Motivation 
The current network policy tests have a few issues which, without increasing technical debt, can be addressed architecturally.
 
- *Incompleteness*: Current tests do not confirm that a common set of negative scenarios for different policies are actually negative.  They also do not confirm a complete set of *positive* connectivity before starting tests (note: 4 out of the existing 25 tests actually do *some* positive control validation before applying policies, and all tests do positive validation *after* policy application).
- *Understandability*: They are difficult to reason about, due to lack of consistency, completeness, and code duplication.
- *Extensibility*: Extending them is a verbose process, which leads to more sprawl in terms of test implementation.
- *Performance*: They suffer from low performance due to the high number of pods created.  Network policy tests can take 30 minutes or longer.  The lack of completeness in positive controls, if fixed, could allow us to rapidly skip many tests destined for failure due to cluster health issues not related to network policy.
- *Dynamic scale*: In addition to increasing the performance of these tests, we also should expand their ability to evaluate CNI's with highly dynamic, realistic workloads, outputting summary metrics.
- *Documentation and Community*: The overall situation for these tests is that they are underdocumented and poorly understood by the community, and it's not clear how these tests are vetted when they are modified; this makes it difficult for CNI providers to compare and contrast compatibility and conformance to K8s standards for NetworkPolicys.
- *Continuous Integration*: As part of this overall effort, once this test suite is more reliable and proven to be faster, running a basic verification of it in CI with some collection of CNI providers which could feed back into upstream K8s test results would be ideal, so that we know the NetworkPolicy test and specifications, as defined, are implemented/implementable correctly by at least some CNI provider.

### Goals

- Rearchitect the way we write and define CNI NetworkPolicy test verifications
- Increase the visibility and quality of documentation available for network policies

#### Concrete goals

For a TLDR, see https://github.com/vmware-tanzu/antrea/blob/master/hack/netpol/pkg/main/main.go of roughly what the code looks like that we are proposing to integrate.

Conceptually we have 5 concrete changes that we are proposing:

1. Introduce a simple "DSL" (in golang) for defining network policy's and reachability matrices.
  - Currently, the word `DSL` might be a misnomer, but has taken hold.  Specifically, we are using a Builder API to concsiely define NetworkPolicies in a declarative way.
  - The DSL examples are later in this document, for example: 
    - DFL for defining test expectations:
    ```
	reachability := NewReachability(allPods, true).ExpectAllIngress(Pod("x/a"), false).Expect(Pod("x/a"), Pod("x/a"), true)
    ```
    - DSL for Defining network policies:
    ```
	builder := &NetworkPolicySpecBuilder{}
	builder = builder.SetName("x", policyName).SetPodSelector(map[string]string{"pod": "a"})
	builder.SetTypeIngress()

    ```
2. Create all namespaces and pods in the test matrix before tests start (exceptions for some tests if we want to test pod churn or label changes etc), as part of the testing library itsef.
3. Rewrite all existing network_policy.go tests using the above DSL using (mostly the same) ginkgo descriptions as current tests do.
4. Integrate tests with ginkgo by simply replacing existing network policy test declarations to use the new DSL
  - putting initialization into a BeforeAll block for scaffold pods/namespaces
5. Annotate these tests (in the code) with the CNIs they were run on, the last time they were running, and the test results (in lieu of waiting to decide on a canonical CNI)

There are many other concepts discussed after this, justifying and looking at the future of how this work might effect the way we think about network policy testing in the future, including how we might
use this work to make the e2e suite more decalarative in the future, but these concrete goals are the endpoint for this specific KEP.

### Non-goals

- Make tests specific to CNI providers
- Change NetworkPolicy APIs
 
### Related issues

As an overall improvement, this KEP will help to address the solutions for several existing issues in upstream Kubernetes.  Some of these issues have been duct-taped upstream, but our overarching goal is to reduce the amount of work required to verify that any such issues have been properly addressed and accounted for in the documentation, testing, and semantic aspects of how the API for NetworkPolicy itself is defined.

- https://github.com/kubernetes/kubernetes/issues/87857 (docs and understandability)
- https://github.com/kubernetes/kubernetes/issues/87893 (holes in our test coverage matrix)
- https://github.com/kubernetes/kubernetes/issues/85908 (failing tests, unclear semantics)
- https://github.com/kubernetes/kubernetes/issues/86578 (needs e2e coverage)
- https://github.com/kubernetes/kubernetes/issues/88375 (the test matrix for Egress is almost entirely empty, decreasing the verbosity of new tests will organically increase likelihood of new test submissions over time.)

#### Also related but not directly addressed in this KEP

- https://github.com/projectcalico/felix/issues/2008 (Not sure wether we should test this, might not be a true bug - but need to test postStart pods in networkpolicy upstream either way and be explicit)
- https://github.com/kubernetes/kubernetes/issues/87709 (Separate KEP, complimentary to this logging of netpol actions, will help describing states we reach) 


### Consequences of this problem
 
The consequences of this problem is that
 
- CNI providers cannot easily be compared for functionality.
- CNI providers implementing network policies must carry a lot of downstream test functionality.
- Testing a CNI provider for Kubernetes compatibility requires a lot of interpretation and time investment.
- Extending NetworkPolicy tests is time consuming and error prone, without a structured review process and acceptance standard.
- It is hard to debug tests, due to the performance characteristics - pods are deleted after each test, so we cannot reproduce the state of the cluster easily.

## A high level outline of Pod Traffic Pathways

Before diving into test details, we outline different types of pod traffic.  Each one of these types of traffic may pose different bugs to a CNI provider.  

Intranode
- pod -> pod
- pod -> host
- host -> pod

Internode
- pod -> pod
- pod -> host
- host -> pod
- `*`host -> host/hostNetwork pod (out-of-scope for K8s Network Policies API)

Traffic Transiting Service DNAT
- `*`Nodeport -> service (DNAT) -> pod (covered as a non-voting test case)
- pod -> service (DNAT) -> pod

Intranamespace
- pod -> pod
- hostNetwork pod -> pod
- pod -> hostNetwork pod

Internamespace
- pod -> pod
- hostNetwork pod -> pod
- pod -> hostNetwork pod

## Detailed examples of the problem statement
 
### Incompleteness

A few concrete missing tests are obvious incompleteness examples, such as https://github.com/kubernetes/kubernetes/issues/87893 and https://github.com/kubernetes/kubernetes/issues/46625

As mentioned in the pre-amble, there is sporadic validation of both positive and negative connectivity in all tests, and in many cases this validation is meaningful.  However, in none of the cases, it is complete.  That is, we do not have any tests which validate all obvious intra and inner namespace connectivity holes, both before and after application of policies.  

Examples which visualize this ensue:

For our first example, we will look at the incompleteness of one of the first tests
in the test suite for network_policy.go.  In this test, the following assertions are
made to verify that inter-namespace traffic can be blocked via NetworkPolicy.
 
The "X" lines denote communication which is blocked, whereas standard arrows denote
traffic that is allowed.

This is from the test "should enforce policy to allow traffic only from a different namespace, based on NamespaceSelector".
 
```
+-------------------------------------------------------------------+
| +------+    +-------+   Figure 1a: The NetworkPolicy Tests        | TODO: maybe include YAML examples side-by-side
| |      |    |       |   current logical structure only verifies   |       visual nomenclature (i.e., cA -> podA)
| |  a   |    |  b    |   one of many possible network connectivity |
| |      |    |       |   requirements. Pods and servers are both   |
| +--+---+    +--X----+   in the same node and namespace.           |
|    |           X                                                  |
|    |     ns A  X                                                  |
+----v-----------X+---+                                             |
||     server         |    Note that the server runs in the         |
||     80, 81         |    "framework" namespace, and so we don't   |
||                    |    draw that namespace specifically here,   |
||                    |    as that namespace is an e2e default.     |
|---------------------+                                             |
+-------------------------------------------------------------------+
```
 
A *complete* version of this test is suggested when we take the union of all
namespaces created in the entire network policy test suite. 
 
- namespaces B and C, in addition to the framework namespace
- each of these namespaces has 2 pods in them
- each of the pods in each of these namespaces attempts connecting to each port on the server
 
```
+-------------------------------------------------------------------------+
|  +------+              +------+  nsA                                    |
|  |      |              |      |                                         |
|  |   b  |              |  c   |     Figure 1b: The above test           |
|  +--+---+              +----X-+     is only complete if a permutation   |
|     |   +---------------+   X       of other test scenarios which       |
|     |   |    server     |   X       guarantee that (1) There is no      |
|     +--->    80,81      XXXXX       namespace that whitelists traffic   |
|         |               |           and that (2) there is no "pod"      | TODO: test "default" namespace
|         +----X--X-------+           which whitelists traffic.           |       check for dropped namespaces
| +------------X--X---------------+                                       |       make test instances bidirectional
| |            X  X               |   We limit the amount of namespaces   |          (client/servers)
| |   +------XXX  XXX-------+  nsB|   to test to 3 because 3 is the union |
| |   |      | X  X |       |     |   of all namespaces.                  |
| |   |  b   | X  X |   c   |     |                                       |
| |   |      | X  X |       |     |   By leveraging the union of all      |
| |   +------+ X  X +-------+     |   namespaces we make *all* network    |
| |            X  X               |   policy tests comparable,            |
| +-------------------------------+   to one another via a simple         |
|  +-----------X--X---------------+   truth table.                        |
|  |           X  X               |                                       |
|  |  +------XXX  XXX-------+  nsC|   This fulfills one of the core       |
|  |  |      |      |       |     |   requirements of this proposal:      |
|  |  |  c   |      |   b   |     |   comparing and reasoning about       |
|  |  |      |      |       |     |   network policy test completeness    |
|  |  +------+      +-------+     |   in a deterministic manner which     |
|  |                              |   doesn't require reading the code.   |
|  +------------------------------+                                       |
|                                      Note that the tests above are all  |
|                                      done in the "framework" namespace, |
|                                                  similar to Figure 1.   |
+-------------------------------------------------------------------------+
```

#### TODO Temporal tests

Note that we also don't explicitly test a few other scenarios that involve changes over time.

- Old pods obeying new policies
- New pods obeying old policies
- Extensive changing of pod labels is keeping up with policies

#### Other concrete examples of incompleteness

The above diagrams show that completeness is virtually impossible, the way the tests are written, because of the fact that each test is manually verifying bespoke cases.  More concretely, however, a look at `should enforce policy to allow traffic only from a different namespace, based on NamespaceSelector [Feature:NetworkPolicy]` reveals that some tests don't do positive controls (validation of preexisting connectivity), whereas others *do* do such controls.

#### List of missing functional test cases

TODO: use multiple pods in contiguous CIDR to validate CIDR traffic matching

- Stacked IPBlock case: Need to add a test case to verify the traffic when a
  CIDR (say 10.0.1.0/24) is used in an ``except`` clause in one NetworkPolicy,
  and the same CIDR is also used in an allow IPBlock rule in another
  NetworkPolicy, both targeting the same ``spec.PodSelector`` within the same
  Namespace.
- NamedPort resolving to multiple port numbers: Current test cases only test
  named port NetworkPolicies resolving to a single port number. Instead,
  improve the test case by testing that multiple Pods with the same name
  port backed by different port numbers are being allowed correctly by the
  NetworkPolicy rule.

### Understandability

TODO: test case names mean something, and each test case should have accompanying diagram
 
In this next case, we'll take another example test, which is meant to confirm that intra-namespace
traffic rules work properly.  This test has a misleading description, and an incomplete test matrix as well.
 
"Understandability" and "Completeness" are not entirely orthogonal - as illustrated here.  The fact that
we do not cover all communication scenarios (as we did in Figure 1b), means that we have to carefully
read the code for this test, to assert that it is testing the same scenario that its Ginkgo description
connotes.
 
We find that the Ginkgo description for this test isn't entirely correct, because
enforcing traffic *only* from a different namespace also means:
- Blocking traffic from the same namespace
- Confirming traffic from *any* pod in the whitelisted namespace
 
As an example of the pitfall in this test, a network policy provider which, by default
allowed *all internamespaced traffic as whitelisted*, would pass this test while violating
the semantics of it.
 
```
+----------------------------------------------------------------------------------------------+
|                                                                                              |
|           +------------------+       +-------------------+                Figure 2:          |
|           |                  |       | +---+      +---+  |                                   |
|   XXXXXXXXX      nsA         |       | | cA|  nsA | cB|  |                A more advanced    |
|   X    --->                  |       | +X--+      +---+  |                example. In these  |
|   X    |  |                  |       |  X             X  |                cases, we can      |
|   X    |  |     server       |       |  X   server    X  |                increase test      |
|   X    |  |      80,81       |     XXXXXXXXX 80,81 XXXX  |                coverage again     |
|   X    |  +------------------+     X +-------^-----------+                by testing an      |
|   X    |                           X         |                            entire truth       |
|   X    |  +------------------+     X +-------------------+                table (right).     |
|   X    |  |                  |     X |       |           |                                   |
|   X    |  |    +--+   +---+  |     X | +-----+----+---+  |                The "creating a    |
|   X    ------- +cA|   |cB |  |     X | |cA|       | cB|  |                network policy     |
|   X       |    +--+   +---+  |     X | +--+       +---+  |                for the server which
|   X       |   nsB            |     X |      nsB          |                allows traffic     |
|   X       +------------------+     X +-------------------+                from ns different  |
|   X                                X                                      then namespace-a   |
|   X       +------------------+     X  +------------------+                                   |
|   X       |                  |     X  |  +--+            |                test should confirm|
|   X       |   +--+    +--+   |     XXXXXX|cA|     +---+  |              positive connectivity|
|   +XXXXXXXXXXX|cA|    |cB|   |     X  |  +--+     | cB|  |                for both containers|
|           |   +--+    +-++   |     X  |           +---+  |                in nsB.  otherwise |
|           |                  |     X  |             X    |                a policy might not |
|           |     nsC          |     X  |    nsC      X    |                be whitelisting n+1|
|           +------------------+     X  +-------------X----+                pods.              |
|                                    X                X                                        |
|                                    XXXXXXXXXXXXXXXXXX                                        |
|                                                                                              |
|                                                                                              |
+----------------------------------------------------------------------------------------------+
```
 
### Extensibility

The previous scenarios look at logical issues with the current tests.  These issues can be mitigated by simply having more tests, which are as verbose as the existing tests.  However:
 
- Each test can be between 50 to 100 lines long.
- The network policies created in each test can be around 30 lines or so.
- There are 25 current tests.
 
Thus, in order to build a new test:
 
- We need to read the other tests, and attempt to capture their logic, for consistency's sake.
- The logic is different in each test, so, what positive and negative controls should be run
is not clear.
- Any given network policy test can take a minute or so to verify, because of namespace
deletion and pod startup times, meaning new tests of a simple network policy add a non-trivial
amount of time to the network policy tests, even though the time it takes to apply a network
policy is instantaneous, and the test itself is completely stateless.
- Comparing network policies between tests requires reading verbose Go structs, such as in the following example.
 
As an example of the cost of extensibility, we compare the subtle distinction between the:
 
`should enforce policy based on PodSelector or NamespaceSelector`
and
`should enforce policy based on PodSelector and NamespaceSelector`
 
tests.  These tests use almost identical harnesses, with a subtle `},{` clause
differentiating the stacked peers in network policy (an *or* selector) vs. a combined peer policy.
 
```
 
                    Ingress: []networkingv1.NetworkPolicyIngressRule{{
                        From: []networkingv1.NetworkPolicyPeer{
                            {
                                // TODO add these composably, so that can be disambiguated from combo networkpolicypeer
                                PodSelector: &metav1.LabelSelector{
                                    MatchLabels: map[string]string{
                                        "pod-name": "client-b",
                                    },
                                },
                            },
                            {
                                NamespaceSelector: &metav1.LabelSelector{
                                    MatchLabels: map[string]string{
                                        "ns-name": nsBName,
                                    },
                                },
                            },
                        },
                    }},
```
 
The AND test is obviously more selective, although it is tricky to tell from the struct that
it has been correctly written to be different from the OR test.
 
```
                    Ingress: []networkingv1.NetworkPolicyIngressRule{{
                        From: []networkingv1.NetworkPolicyPeer{
                            {
                                PodSelector: &metav1.LabelSelector{
                                    MatchLabels: map[string]string{
                                        "pod-name": "client-b",
                                    },
                                },
                                // because we lack },{ , this is a single Peer selecting
                                // pods, from namespaces selected by namespace selector.
                                // This is difficult to verify for correctness at a
                                // glance, due to the verbosity of the struct.
                                NamespaceSelector: &metav1.LabelSelector{
                                    MatchLabels: map[string]string{
                                        "ns-name": nsBName,
                                    },
                                },
                            },
                        },
```

In contrast, we can express the same and test using the API (including its entire connectivity matrix) in this proposal as follows:

```
	builder := &NetworkPolicySpecBuilder{}
	builder = builder.SetName("myns", "allow-podb-in-nsb").SetPodSelector(map[string]string{"pod": "b"})
	builder.SetTypeIngress()
	builder.AddIngress(nil, &p80, nil, nil, map[string]string{"pod-name": "b"}, nil, map[string]string{"ns-name": "b"}, nil)
	policy := builder.Get()
	reachability := NewReachability(allPods, false)
	reachability.ExpectAllIngress(Pod("myns/b"), false)
	reachability.Expect(Pod("b/b"), Pod("myns/b"), true)
	reachability.Expect(Pod("myns/b"), Pod("myns/b"), true)
```
We can of course make this much easier to reuse and reason about, as well as make it self-documenting, and will outline how in the solutions section.

### Performance
 
CURRENTLY: For each test, a few pods are spun up, and a polling process occurs where we wait for the pod to complete successfully and report it could connect to a target pod.  Because all clusters start pods at different rates, heuristics have to be relied on for timing a test out.  A large, slow cluster may not be capable of spinning pods up quickly, and thus may timeout one of the 25 tests, leading to a false negative result.
 
In some clusters, for example, namespace deletion is known to be slow - and in these cases the network policy tests may take more than an hour to complete.
 
- If network policies or pod CIDR's are not correct, it's likely all tests can fail, and thus the network policy suite may take an hour to finish, based on the estimate of 3 minutes, for each failed test, alongside 25 tests (in general, NetworkPolicy tests on a healthy EC2 cluster, with no traffic and broken network policies, take between 150 and 200 seconds complete).

PROPOSAL: (Implemented and working, as demo'd recently in sig-network) Using `Pod Exec` functionality, we've determined that 81 verifications can happen rapidly, within 30 seconds, when tests run inside of Kubernetes pods, compared with about the same time for a single test with up to 5 verifications, using Pod status indicators. In addition to this, after running the 81 verifications concurrently in go routines, the total execution time reduced drastically.



#### Logging verbosity is worse for slow tests
 
CURRENTLY: Slow running tests are also hard to understand, because logging and metadata is expanded over a larger period of time, increasing the amount
of information needed to be attended to diagnose an issue. For example, to test this, we have intentionally misconfigured my CIDR information for a calico CNI,
and found that the following verbose logging about is returned when running the `NetworkPolicy` suite:
```
Feb  4 16:01:16.747: INFO: Pod "client-a-swm8q": Phase="Pending", Reason="", readiness=false. Elapsed: 1.87729ms
... 26 more lines ...
Feb  4 16:02:04.808: INFO: Pod "client-a-swm8q": Phase="Failed", Reason="", readiness=false. Elapsed: 48.063517483s
```
Thus, the majority of the logging information from a CNI which may have an issue is actually related to the various polling operations
which occurred, rather then to the test itself.  Of course, this makes sense - since we always recreate pods, we have to potentially
wait many seconds for those pods to come up.
 
Thus, by increasing the performance of our tests, we also increase their understandability, because the amount of information needed to be
audited for inspecting a failure may be reduced by 50% (currently, 50% of the output for failing network policy tests is that of the polling
process for pods spinning up, which is easily avoided by a fixed server and client pod).
 
### Documentation
 
Documenting network states is very hard, in any scenario. Since the NetworkPolicy ginkgo tests are currently not documented outside of the code, no specific evidence is required here.  This proposal aims not to document these tests, but rather, to make the code more readable, and thus self-documenting.  However, formal documentation of how network policies, generally, are evaluated using a truth table approach, is part of this proposal.
This generic documentation will be insightful and concise for those needing to test their NetworkPolicy implementations, and likely to not become obsolete, due to the generic nature of the truth-table/matrix approach (compared to the highly specific nature of existing tests).

As a few examples of this:
- the test `Creating a network policy for the server which allows traffic from the pod 'client-a' in same namespace` actually needs to confirm that *no outside* namespace can communicate with the server.
- the test outlined in Figure2 is another example of a test which isn't described comprehensively.

In the solutions section, we will highlight how the proposal makes these tests, and thus the semantics of network policies, explicit and self documenting.
 
## Goals
 
In short, our solution to this problem follows

- *Increase performance* of tests by using persistent Deployments.
- *Increase understandability* by defining network scenario objects which can easily by modified and reused between tests, and outputting the entire contents of the truth table for each test, in some manner.
- *Increase completeness* by using a logical truth table which tests connectivity/disconnectivity for each scenario above.
- *Increase extensibility* by leveraging the scenario objects and the completeness checking functionality above.
- *Increase debuggability* by leveraging the performance changes above.
- *Audit all existing tests and upstream issues* For logical redundancy and consistency.  If a test is missing, we'll add it.  If redundant, we'll remove it if necessary.  All issues upstream such as - https://github.com/kubernetes/kubernetes/issues/46625 will also be audited and implement as needed as part of this enhancment.
 
## Implementation History
 
An architectural change to the current testing policies has been implemented and is described below.

*This implementation runs continuously as part of the Antrea CNI project.*

###  Part 1: Defining a static matrix of ns/pod combinations
  
1. Define a common set of namespaces, and pods, to be used to make a truth table that applies to all tests.  This is demonstrated in diagram 1b and 2.

- Namespaces: x,y,z
- Pods: a,b,c
- Services: All pods are *both* servers
- Clients: All pods *can* behave as clients 
 
These resources are *shared* across every test.
 
2. Define a structure for expressing the truth table of results. Since classically a truth table can be expressed as a 2D matrix, where
rows and columns are the lexically sorted list of all pod namespace pairs defined above, formatted as `namespace-pod`.  For example, a truth table defining a NetworkPolicy where only pods in the same namespace of the server can communicate to it, would look like this.  Capital letters are *namespaces*, and lower case letters are *pods* in those namespaces.  The tuple value represents connectivity to ports *80* and *81*, respectively.
 
Note this matrix is conceptual, the actual implementation of such a matrix is more easily done via other data structures, 
as demonstrated in the existing POC https://github.com/vmware-tanzu/antrea/tree/master/hack/netpol
                                                                                       


|    | As  | Aa  | Ab  | Ab  | Ba  | Bb  | Ca  | Cb  |
|----|-----|-----|-----|-----|-----|-----|-----|-----|
| As | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 |
| Aa | 1,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 |
| Ab | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 |
| Ba | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 |
| Bb | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 |
| Ca | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 |
| Cb | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 | 0,0 |

*Output suggestion*
- expected (t/f)
- result (T/F)
- success / failed (1/0), pass = 1

In the below matrix, both pods a and b could *successfully* connect to pod a (s), as expected (T), passing the test (1) 
```      x/a
x/a      sT1
x/b      sT1 
```
  
Most of the Matrices for this table will be permuting the first row and column, since the server pod currently always resides in the framework namespace.  However, tests might confirm two way connectivity
and other types of connectivity in the future, and such an expansion would work very cleanly with a matrix.
 
Part of this involves a pretty-print functionality for these tables, which can be output at the end of each Network policy test.  In failed
test scenarios, these tables can be compared, and one may easily parse out a logical inference such as "Everything outside the framework
namespace has connectivity, even when the truth table explicitly forbids it", which might, for example, point to a bug in a CNI provider
related to flagrantly allowing internamespace traffic.  Since it is obvious how such a matrix might be defined in Go, we don't provide a
code snippet or API example.
 
3. (about 80% done modulo ginkgo implementation details) Rewrite each individual test, reviewing semantics, to be precisely worded (and possibly verbose), and to simply define a specific policy and
set of 'whitelisted' communication associated with this policy. The whitelisting would be defined as a map of namespace->pods, since all other
information in the truth table is false.
 
Example:

Initially, to confirm the logical capacity of the builder mechanism for replacing existing tests, a prototype of inplace replacements of NetworkPolicy definitions was done here (prototype) https://gist.github.com/6a62266e0eec2b15e5250bd65daa4faa.  
Now, this underlying API has (essentially) been implemented in full, and the following repository https://github.com/vmware-tanzu/antrea/tree/master/hack/netpol, demonstrates a working implementation and port of the network policy tests (Most of the tests have been ported).  Each test follows a simple and easy to read pattern such as this:

 ```
 	// TODO, consider a SpecBuilder.New(...) style of invocation
	builder := &utils.NetworkPolicySpecBuilder{}
	builder = builder.SetName("allow-x-via-pod-and-ns-selector").SetPodSelector(map[string]string{"pod": "a"})
	builder.SetTypeIngress()
	builder.AddIngress(nil, &p80, nil, nil, map[string]string{"pod":"b"}, map[string]string{"ns":"y"}, nil, nil)

	reachability := utils.NewReachability(allPods, true)
	reachability.ExpectAllIngress(Pod("x/a"), false)
	reachability.Expect(Pod("y/b"), Pod("x/a"), true)
	reachability.Expect(Pod("x/a"), Pod("x/a"), true)
  ```
 This represents a significant reduction in code complexity, with the equivalent tests using the existing `network_policy.go` implementation being 3 to 4 times as long, mostly due to boilerplate around verification and go structures.

*Further improvements to the testing API*

- Make the function calls in network policy builder *even* more DSL-like, for example, 
```
Pod(...).InNamespace(...).CanAccess(...)
```
- Infer `Egress,Ingress` rules rather than force them to be specified, based on builder inputs.  They're redundant to begin with (i.e. calico doesn't even require them)
- Add `From` and `To` semantics to the struct API calls in reachability.
 
###  Note on Acceptance and Backwards compatibility

From discussion in the community, we've decided to manually verify that we haven't lost coverage and trust reviewers to be dilligent in final review of the new architecture, by comparing Ginkgo sentences and old test structs.

## Next steps

As of now, network policy tests are not run regularly against any CNI.  Although we should not endorse one CNI over another, we should regularly validate
that the NetworkPolicy tests *can* pass on *some* provider.  As part of this proposal we propose committing an annotation to the existing `network_policy.go` code which states...
- what environment the `network_policy.go` test suite was run in
- the last time which it was committed and passed.  

It's also acceptable to commit this as a Markdown file in the documentation.
 
There may be other, better ways of doing this.  Running an upstream validation job of these tests as a weekly PROW job, for example, would be a good way to make sure that these tests don't regress in the future.  This comes at the cost of coupling a job to an external CNI provider, so its not being explicitly suggested.

## Thoughts from initial research in this proposal (future KEPs)

These are not part of this KEP, but came as logical conclusions from doing the initial evaluation of the API testing current-state-of-affairs for discussion in future KEPS with the community.

### Node local traffic: Should it be revisited in V2 ?

There is an interesting caveat in the types.go API definition for NetworkPolicy.Ingress, wherein we mention that node local traffic is ALLOWED by default.  This is to enable health checks.  Although for now we can add this test, it's worth leveraging some of the thought process here to rethink wether this security hole is wanted long term in the network policy API.

### Validation and 'type' : Should it be revisited ?

There are some corner cases for validation (like type=Ingress, but an existing `egress:` stanza is present) which are not currently validated against.  Should we start rejecting such policies, since after all, they don't make any sense (i.e. sending an egress payload when the type is Ingress has no value, unless you're planning on toggling ingress/egress on and off over time, which seems like would be easier done by simply creating/deleting new policies.  And even if toggling were desired, you could do this with explicit API constructs "EgressEnabled:true", "IngressEnabled:true".

### Finding comprehensive test cases for policy validation

The idea of having a set of test scenarios, running the tests on those same test scenarios, and testing positive/negative cases are one of the ways this proposal aims to enhance the current testing.

To make the test cases more comprehensive, all different datapaths that packets follow should be explored. While packet datapaths are different with different CNIs, they may even differ within the same node depending on the subnets the pods belong. For example, some CNIs do not encapsulate packets that is destined to a pod in the same subnet, whereas the rest is encapsulated, which has huge impact on the datapath.

The reasoning behind the need to increase test coverage by adding these scenarios are that at different scenarios packets hit iptables/OVS/BPF rules at different points. To fully ensure that CNI solutions correctly enforce network policies for all cases the following test scenarios should be considered: 
- Have at least 2 nodes.
- Have pods that are on host network like kube-api-server.
- Have pods that are on different subnets.
- Have pods that are on different nodes.

An example test matrix would be (assume each pod has 2 containers, where each uses different ports for connection):
```
|      |                  Node-1            |                Node-2              |
|------|--------------|----------|----------|--------------|----------|----------|
|      | Host-network | Subnet-A | Subnet-B | Host-network | Subnet-A | Subnet-B |
|------|--------------|----------|----------|--------------|----------|----------|
| NS-A | Pod1         | Pod2     | Pod3     | Pod4         | Pod5     | Pod6     |
| NS-B | Pod7         | Pod8     | Pod9     | Pod10        | Pod11    | Pod12    |
|      |              |          |          |              |          |          |

```
One server should be picked from host-network pods and one server from a non-host-network pods (e.g., Pod1 and Pod2 are picked as servers), and all policies should be applied to each servers separately and all the remaining pods start testing their connection based on reachability matrix.
In summary, with the above matrix, inter/intra-namespace tests on the same node and inter-intra namespace tests on different nodes are demonstrated. Adding the third node and third namespace may provide additional coverage (such as server on ns A whitelisting ns B, but not whitelisting ns C) but having more namespace and nodes may not bring additional value in terms of coverage. 

### Ensuring that large policy stacks evaluate correctly

Right now the coverage of Policy stacks is rudimentary, we may want to test for a large number(i.e. 10) of policies, stacked, depending on whether we think this may be a bug source for providers.

### Ensure NetworkPolicy evaluates correctly regardless of the order of events

The order of Pod/Namespace/NetworkPolicy ADD/UPDATE/DELETE events should not matter while evaluating a NetworkPolicy. In general, CNIs may have different code paths for ADD Pod -> ADD NetworkPolicy order of events versus, ADD NetworkPolicy -> ADD Pod events. At least some tests should focus on the fact that the different order of events for different resources does not affect the evaluation of this NetworkPolicy.

### Generating the reachability matrix on the fly
Given a network policy and current pods in a cluster, reachability matrices can be generated automatically. This will make adding network policy test easier as they will no longer be defined manually for each test in the code. 

### Consuming network policies as yaml files
After the above improvement is implemented, since we can generate a reachability matrix on the fly, there will be no need to add network policies to the code. Instead, network policies can be added to a directory as yaml files. 

When the network policy tests are started: 
1) apply each policy one by one (by simply applying the policy yaml file), 
2) get all the pods in the cluster, 
3) generate a reachability matrix (which pods can connect with which other pods), 
4) test if the reachability matrix is correct.

Adopting this approach enable users to test any custom network policy without changing any code. Also, code maintenance will become much simpler as only thing to maintain will be the part to create reachability matrices as new enhancements added to network policies.

Another improvement can be to do combinatorial stacking of all the tests that exists, which will be very easy to do, if this approach is adopted.

## Alternative solutions to this proposal
 
#### Keeping the tests as they are and fixing them one by one
 
We could simply audit existing tests for completeness, and one-by-one, add new test coverage where it is lacking.  This may be feasible for the 23 tests we currently have, but it would be likely to bit-rot over time, and not solve the extensibility or debuggability problems.
 
#### Building a framework for NetworkPolicy evaluation
 
In this proposal, we've avoided suggesting a complex framework that could generate large numbers of services and pods, and large permutations of scenarios.
However, it should be noted that such a framework might be useful in testing performance at larger scales, and comparing CNI providers with one another. Such a framework could easily be adopted to cover the minimal needs of the NetworkPolicy implementation in core Kubernetes, so it might be an interesting initiative to work on.  Such an initiative might fall on the shoulders of another Sig, related to performance or scale.  Since NetworkPolicies have many easy-to-address
problems which are important as they stand, we avoid going down this rat-hole, for now.
 
That said, the work proposed here might be a first step toward a more generic CNI testing model.

#### Have the CNI organization create such tests

We cannot proxy this work to the individual CNI organization, because in large part, the semantics of how network policies are implemented and what we care about from an API perspective is defined by Kubernetes itself.  As we propose expansion of the Network Policy API, we need a way to express the effects of these new APIs in code, concisely, in a manner which is guaranteed to test robustly.

