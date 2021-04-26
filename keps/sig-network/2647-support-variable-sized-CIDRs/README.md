# KEP-2647: NodeIPAM controller support variable sized CIDRs

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Current Behaviors](#current-behaviors)
  - [User Stories](#user-stories)
  - [Proposal Details](#proposal-details)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
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

Make NodeIPAM controller support variable sized CIDRs.

## Motivation

Some user want to change `--node-cidr-mask-size` after create kubernetes cluster by some reason, 
but current kubernetes doesn't allow `--node-cidr-mask-size` to change, because NodeIPAM controller
doesn't support variable sized CIDRs.

### Goals

- Make NodeIPAM controller support variable sized CIDRs.
- handle multi-control plane upgradation case

### Non-Goals


## Proposal

Use interval tree instead of bitmap to allocate podCIDR.

### Current Behaviors

After the cidr is changed, new allocated node cidr may have overlay with the old ones. More can be found in [kubernetes#90922](https://github.com/kubernetes/kubernetes/issues/90922)

### User Stories

For example, we have a kubernetes cluster, the `clusterCIDR` is 192.168.0.0/16 and `--node-cidr-mask-size` is 24
1. add two nodes to cluster, podCIDR will be `192.168.0.0/24` and `192.168.1.0/24`;
2. change `--node-cidr-mask-size` to 23 and delete node `192.168.1.0/24`;
3. add a node pod CIDR will be `192.168.0.0/23`, `192.168.0.0/24` and `192.168.0.0/23` overlap.

### Notes/Constraints/Caveats (optional)


### Risks and Mitigations

1. Cluster has more than 1 apiserver and can it works correctly when apiserver is upgrading one by one?
2. How can we revert/retore the change?

## Design Details

We need use a new data structure instead of bitmap to allocate podCIDR, it should support variable sized CIDRs, 
interval tree is suitable. I found etcd already implement an interval tree 
(https://github.com/etcd-io/etcd/blob/master/pkg/adt/interval_tree.go).

If CIDR can convert to Interval, then we can use `IntervalTree.Insert`, `IntervalTree.Intersects` and `IntervalTree.Delete`.
First get begin IP and end IP in CIDR, because IPv6 length is 128 bits, `uint64` is not enough, 
so convert an IP to `big.Int`, then convert big.Int to string, we got `adt.StringInterval`.

```go
func (s *CidrSet) cidrToInterval(cidr *net.IPNet) (adt.Interval, error) {
	if cidr == nil {
		return adt.Interval{}, fmt.Errorf("cidr is nil")
	}
	if !s.clusterCIDR.Contains(cidr.IP.Mask(s.clusterCIDR.Mask)) {
		return adt.Interval{}, fmt.Errorf("cidr %v is out the range of cluster cidr %v", cidr, s.clusterCIDR)
	}
	begin := big.NewInt(0)
	end := big.NewInt(1)
	maskSize, length := cidr.Mask.Size()
	if s.clusterMaskSize < maskSize {
		if cidr.IP.To4() != nil {
			begin = begin.SetBytes(cidr.IP.To4())
		} else {
			begin = begin.SetBytes(cidr.IP.To16())
		}
		ones, bits := cidr.Mask.Size()
		end.Lsh(end, uint(bits-ones)).Add(begin, end)
	} else {
		if s.clusterCIDR.IP.To4() != nil {
			begin = begin.SetBytes(s.clusterCIDR.IP.To4())
		} else {
			begin = begin.SetBytes(s.clusterCIDR.IP.To16())
		}
		ones, bits := s.clusterCIDR.Mask.Size()
		end.Lsh(end, uint(bits-ones)).Add(begin, end)
	}
	return adt.NewStringInterval(
		fmt.Sprintf("%0"+strconv.Itoa(length)+"s", begin),
		fmt.Sprintf("%0"+strconv.Itoa(length)+"s", end),
	), nil
}
```

### Test Plan

Unit test for the new data structure `pkg/controller/nodeipam/ipam/cidrset/cidr_set_test.go` and exercising the new functionality `pkg/controller/nodeipam/ipam/range_allocator_test.go`


### Graduation Criteria


### Upgrade / Downgrade Strategy

A designed upgrade case that should be taken into account.
- Have 3-controller-manager instances
- Upgrade 1 of them to this new model
- Allocate node with the new CID size
- Leader election changes which controller-manager is in charge

### Version Skew Strategy



## Benchmarking

```go
func BenchmarkIPv4AllocateNet(b *testing.B) {
	_, clusterCIDR, _ := net.ParseCIDR("10.0.0.0/8")
	a, err := NewCIDRSet(clusterCIDR, 24)
	if err != nil {
		b.Fatalf("unexpected error: %v", err)
	}
	for i := 0; i < a.maxCIDRs; i++ {
		if _, err := a.AllocateNext(); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkIPv6AllocateNext(b *testing.B) {
	_, clusterCIDR, _ := net.ParseCIDR("2001:0db8:1234:3::/48")
	a, err := NewCIDRSet(clusterCIDR, 64)
	if err != nil {
		b.Fatalf("unexpected error: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < a.maxCIDRs; i++ {
		if _, err := a.AllocateNext(); err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
```

![memprofile](profile001.png)


## Production Readiness Review Questionnaire


### Feature enablement and rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  No.

* **Does enabling the feature change any default behavior?**
  No.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  No.

* **What happens if we reenable the feature if it was previously rolled back?**

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling and disabling feature
  gates. However, unit tests in each component dealing with managing data created
  with and without the feature are necessary. At the very least, think about
  conversion tests if API types are being modified.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g. what if some components will restart
  in the middle of rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metrics. Operations against Kubernetes API (e.g.
  checking if there are objects with field X set) may be last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At the high-level this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide a comprehensive guidance, but at the very
  high level (they needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  Describe the metrics themselves and the reason they weren't added (e.g. cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

  For each of the fill in the following, thinking both about running user workloads
  and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high error rates on the feature:


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirms the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

* **Will enabling / using this feature result in introducing new API types?**
  Describe them providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

* **Will enabling / using this feature result in any new calls to cloud
  provider?**

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  Describe them providing:
  - API type(s):
  - Estimated increase in size: (e.g. new annotation of size 32B)
  - Estimated amount of new objects: (e.g. new Object X for every existing Pod)

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data send and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits][].

### Troubleshooting

Troubleshooting section serves the `Playbook` role as of now. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now we leave it here though.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
  For each of them fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without loogging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debugging the issue?
      Not required until feature graduated to Beta.
    - Testing: Are there any tests for failure mode? If not describe why.

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History



## Drawbacks



## Alternatives

Use map record a CIDR used by node count, but it cost too many memories,
see https://github.com/kubernetes/kubernetes/pull/90926. 

## Infrastructure Needed (optional)
