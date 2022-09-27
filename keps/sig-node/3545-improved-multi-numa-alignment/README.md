# KEP-3545: Improved multi-numa alignment in Topology Manager

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

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
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Proposed Change](#proposed-change)
  - [Implementation strategy](#implementation-strategy)
  - [Calculating average distance](#calculating-average-distance)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha to Beta Graduation](#alpha-to-beta-graduation)
    - [Beta to G.A Graduation](#beta-to-ga-graduation)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

We propose an enhancement to TopologyManager that enables it to favor sets of NUMA nodes with shorter distance between the nodes, when making admission decision. The proposed enhancement is only applicable when comparing sets of NUMA nodes equal in size.

## Motivation

To support latency-critical execution and high-throughput applications (including those running on Kubernetes), NUMA locality(topology) of resources assigned to the workloads is crucial. THe best NUMA locality can be achieved by minimizing number of NUMA nodes required to execute the workload, and minimizing the NUMA distance between the nodes. NUMA distance is a metric which exposes relative physical distance between NUMA cells which relates to latency that is introduced when a CPU needs to access resources from different NUMA nodes. The distance is always within range 10-254 and for a local access its value is 10.

As of now, TopologyManager provides policies to either provide resources from only single numa node or the smallest number of NUMA nodes. TopologyManager is not aware of the NUMA distances and does not take it into consideration during admission decisions. 

This limitation surfaces in multi socket, as well as single socket multi NUMA systems (commonly present in AMD hardware), and might cause a significant performance degradation for latency-critical execution and high-throughput applications, in a case when the Topology Manager schedules a pod on non-adjacent numa nodes.


### Goals

* Enable TopologyManager to prefer sets of NUMA nodes with shorter NUMA distances for all TopologyManager policies
* Preserve all other properties of all policies

## Proposal

### Risks and Mitigations

| Risk                                             | Impact | Mitigation                                                                                                                                                                               |
|--------------------------------------------------|--------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Bugs in the implementation lead to kubelet crash | High   | Disable the policy option and restart the kubelet. The workload will run but resources allocation can be spread among NUMA nodes which are non-adjacent what can lead to higher latency. |

## Design Details


### Proposed Change
 
We propose to
- add a new flag in Kubelet called `TopologyManagerPolicyOptions` in the kubelet config or command line argument called `topology-manager-policy-options` which allows the user to specify the Topology Manager policy option.
- add a new topology manager option called `prefer-closest-numa-nodes`; if present, this option will enable further refinements of the existing `restricted` and `best-effort` policies, this option has no effect for `none` and `single-numa-node` policies.

Best-effort or restricted policy chooses resources that will fit into the smallest number of NUMA nodes.
This enhancement wants to change how the best hint is chosen from choosing narrower `NUMANodeAffinity` bitmask to choosing the one that is the narrowest and has the shortest average distance between NUMA nodes.

To summarize properties of `prefer-closest-numa-nodes` policy:

* Preserve all properties of `best-effort` and `restricted` policies
* Choose `NUMANodeAffinity` bitmask which is the narrowest and if the bitmask width is the same prefer the bitmask with smaller average distance.

When `prefer-closest-numa-nodes` policy is enabled, we need to retrieve information regarding distances between NUMA nodes.
Right now Topology manager discovers Node layout using [CAdvisor API](https://github.com/google/cadvisor/blob/master/info/v1/machine.go#L40).

We would need to extend the `MachineInfo` struct with Distances field which would be an array of uint64 which will describe the distance between a given node and other nodes in the system.
This information can be read from sysfs:

```go 
const     distanceFile = "distance"

func (fs *realSysFs) GetDistance(nodePath string) (string, error) {
    distancePath := fmt.Sprintf("%s/%s", nodePath, distanceFile)
    distance, err := ioutil.ReadFile(distancePath)
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(distance)), err
}

func getDistance(sysFs sysfs.SysFs, nodeDir string) ([]uint64, error) {
    rawDistance, err := sysFs.GetDistance(nodeDir)
    if err != nil {
        //Ignore if per-node info is not available.
        klog.Warningf("Found node without distance information, nodeDir: %s", nodeDir)
        return nil, nil
    }

    distances := []uint64{}
    for _, distance := range strings.Split(rawDistance, " ") {
        distanceInt, err := strconv.ParseUint(distance, 10, 64)
        if err != nil {
            return nil, fmt.Errorf("cannot convert %s to int", distance)
        }
        distances = append(distances, distanceInt)
    }

    return distances, nil
}
```

### Implementation strategy

- Introduce new flag in Kubelet called `topology-manager-policy-options`, which when specified with `prefer-closest-numa-nodes` will modify the behavior of `best-effort` and `restricted` policy to pick NUMA nodes based on average distance between them.
- The `TopologyManagerPolicyOptions` flag is propagated to `ContainerManager` and later to `TopologyManager`.
- Enable `TopologyManager` NUMA distances discovery:
  - Temporarily add distances discovery logic into `kubelet` (similarly to [the introduction](https://github.com/kubernetes/kubernetes/commit/ecc14fe661c22f5da967a7ff50cfb3aead60905b) of `GetNUMANode()`).
  - Once `cadvisor` exposes the NUMA distance information through `MachineInfo`, remove the logic out of `kubelet` (similarly to [the removal](https://github.com/kubernetes/kubernetes/commit/a047e8aa1b705bb7e5be881fb63cf90a218b60d0) of `GetNUMANode()`).
  - The removal of the logic `kubelet` is [an Alpha to Beta graduation criteria](#alpha-to-beta-graduation).
- When `TopologyManager` is being created it discovers distances between NUMA nodes and stores them inside `manager` struct. This is temporary until `distance` information will land into `cadvisor`.
- Pass `TopologyManagerPolicyOptions` to best-effort and restricted policy. When this is specified best-hint is picked based on average distance between NUMA nodes. This would require modification to `compareHints` function to change how the best hint is calculated:

```go 

// NUMADistance is a matrix representing distances between NUMA nodes
type NUMADistance [][]uint64

func (n NUMADistance) CalculateAvgDistanceFor(bm bitmask.BitMask) int {
   // implementation
   return avgDistance
}

type PolicyOpts struct {
    PreferClosestNuma bool
    Distances NUMADistance
}

func compareHints(bestNonPreferredAffinityCount int, current *TopologyHint, candidate *TopologyHint, policyOpts *PolicyOpts) *TopologyHint {
    /*
    ...
    */

	if current.Preferred && candidate.Preferred {
        if candidate.NUMANodeAffinity.IsNarrowerThan(current.NUMANodeAffinity) {
            return candidate
        } 
        if policyOpts.PreferClosestNuma && candidate.NUMANodeAffinity.IsEqual(current.NUMANodeAffinity) {
            candidateDistance := policyOpts.Distances.CalculateAvgDistanceFor(candidate)
            currentDistance := policyOpts.Distances.CalculateAvgDistanceFor(current)
            // candidate avg distance is lower
            if candidateDistance < currentDistance {
                return candidate
            } 

            return current 
        }
	}

    /*
    ...
    */
}

```

### Calculating average distance

Let's consider following distance table:

|  node/node | node0 | node1 | node2 | node3 |
| ---------- | ----- | ----- | ----- | ----- |
| node0      | 10    | 11    | 12    | 12    |
| node1      | 11    | 10    | 12    | 12    |
| node2      | 12    | 12    | 10    | 11    |
| node3      | 12    | 12    | 11    | 10    |


If resources cannot be fitted into one NUMA node the new policy option will prefer hints with bitmasks that have lower average distance between NUMA nodes. Such bitmasks:

* 1100 -> (10 + 11 + 11 + 10) /4 = 10.5
* 1010 -> (10 + 12 + 10 + 12) /4 = 11
* 1001 -> (10 + 12 + 10 + 12) /4 = 11
* 0011 -> (10 + 11 + 10 + 11 ) /4 = 10.5
* 0110 -> (10 + 12 + 10 + 12 ) /4 = 11

So the bitmasks 1100 and 0011 has the lowest average distance between NUMA nodes.

If we consider system with 8 NUMA nodes:

|  node/node | node0 | node1 | node2 | node3 | node4 | node5 | node6 | node7 |
| ---------- | ----- | ----- | ----- | ----- | ----- | ----- | ----- | ----- |
| node0      | 10    | 11    | 12    | 12    | 30    | 30    | 30    | 30    |
| node1      | 11    | 10    | 12    | 12    | 30    | 30    | 30    | 30    |
| node2      | 12    | 12    | 10    | 11    | 30    | 30    | 30    | 30    |
| node3      | 12    | 12    | 11    | 10    | 30    | 30    | 30    | 30    |
| node4      | 30    | 30    | 30    | 30    | 10    | 11    | 12    | 12    |
| node5      | 30    | 30    | 30    | 30    | 11    | 10    | 12    | 12    |
| node6      | 30    | 30    | 30    | 30    | 12    | 12    | 10    | 11    |
| node7      | 30    | 30    | 30    | 30    | 12    | 12    | 11    | 10    |

And following bitmasks:

* 10001000 -> (10 + 30 + 10 +30) /4 = 20
* 11100000 -> (10 + 11 + 12 + 10 + 11 + 12 + 12 + 12 + 10) / 6 = 16.7

In second case even though the average distance is lower but the bitmask width is bigger and that is why the first case should be preffered.

### Test Plan

- [x] We understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates


##### Unit tests

- `k8s.io/kubernetes/pkg/kubelet/cm/topologymanager`: `09-23-2022` - `92.4`

##### Integration tests

 These cases will be added in the existing integration tests:
  - Feature gate enable/disable tests
  - `prefer-closest-numa-nodes` policy option works as expected. When policy option is enabled
     - `Merge` prefers hints with the lowest average distance between NUMA nodes.
  - Verify no significant performance degradation

##### e2e tests

These cases will be added in the existing e2e tests:
  - Feature gate enable/disable tests
  - `prefer-closest-numa-nodes` policy option works as expected.

### Graduation Criteria

#### Alpha
- [X] Implement the new policy option.
- [X] Temporarily include NUMA distances discovery logic in kubelet code.
- [X] Add proper e2e node tests.

#### Alpha to Beta Graduation
- [X] Gather feedback from the consumer of the `prefer-closest-numa-nodes` policy option.
- [X] Remove NUMA distances discovery logic from kubelet in favor of updated cAdvisor.
- [X] No major bugs reported in the previous cycle.

#### Beta to G.A Graduation
- [X] Allowing time for feedback (1 year).
- [X] Risks have been addressed.


### Upgrade / Downgrade Strategy

We expect no impact. The new policiy options are opt-in.

### Version Skew Strategy

No changes needed

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `TopologyManagerPolicyOptions`
  - Components depending on the feature gate: kubelet
- [x] Change the kubelet configuration to set a `TopologyManager` policy to `restricted` or `best-effort` and a `TopologyManagerPolicyOptions` to `prefer-closest-numa-nodes`
  - Will enabling / disabling the feature require downtime of the control
    plane? 
    No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
    Yes, a kubelet restart is required.

###### Does enabling the feature change any default behavior?

- Yes, it makes the behaviour of the TopologyManager restricted and best-effort policies to choose NUMA nodes based on distance.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

- Yes, disabling the feature gate shuts down the feature completely.
- Yes, through kubelet configuration - disable given policy option.

###### What happens if we reenable the feature if it was previously rolled back?

- No changes. The allocation of resources won't be changed for exiting containers. It will change for new containers only.

###### Are there any tests for feature enablement/disablement?

- There will be specific e2e test demonstrating that default behaviour is preserved when feature gate is disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Kubelet may fail to start. The kubelet may crash.

###### What specific metrics should inform a rollback?

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Inspect the kubelet configuration of the nodes: check feature gate and usage of the new option

###### How can someone using this feature know that it is working for their instance?

In order to verify this feature is working, one should:
Pick a node with at least 4 NUMA nodes and with at least 3 different distance values between them.
Ensure no other pods are running on given node.
Launch pod that requires resources from 2 NUMA nodes. For example if there are 8 CPUs per NUMA node, ask for 16 CPUs.
Verify that CPUs are assigned from NUMA nodes with lowest distance.
To verify the list of CPUs allocated to the container, their NUMA nodes and the distance between them:
- `exec` into uthe container and run `taskset -cp 1` to retrieve the list of CPUs assigned to PID 1 and then `numactl -H` to retrieve information which CPU is on which NUMA node and what is the distance between nodes. Assuming those commands are available inside the POD.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

No change.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A.

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

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?
N/A.

###### What are other known failure modes?

TBD.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A.

## Implementation History

- 2021-09-26: KEP created
