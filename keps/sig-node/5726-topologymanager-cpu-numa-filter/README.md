# KEP-5726: TopologyManager CPU-attached NUMA filter option

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [User Stories](#user-stories)
- [Design Details](#design-details)
  - [API](#api)
  - [TopologyManager Behavior](#topologymanager-behavior)
  - [CPUManager and MemoryManager Behavior](#cpumanager-and-memorymanager-behavior)
  - [DeviceManager Behavior](#devicemanager-behavior)
  - [Failure Modes](#failure-modes)
- [Risks and Mitigations](#risks-and-mitigations)
- [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha](#alpha)
  - [Beta](#beta)
  - [Stable](#stable)
- [Production Readiness Review Questions](#production-readiness-review-questions)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Monitoring Requirements](#monitoring-requirements)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Add an alpha TopologyManager policy option, `restrict-to-cpu-numa-nodes`, to limit topology hint generation to NUMA nodes with CPUs attached.

## Motivation

Some large NUMA systems (typically coherent-system, lile NVIDIA GraceBlackwell/VeraRubin-class) expose many NUMA nodes to the OS even though only a small subset have CPUs attached. On systems such as NVIDIA GB200/300 platforms, kubelet topology hint generation can scale poorly when it considers the full NUMA-node set.

This proposal adds an explicit, opt-in TopologyManager behavior for those systems.

Related context:
- Root issue: `kubernetes/kubernetes#135541`
- Related broader discussion: `kubernetes/enhancements#5726`
- This reworks and supersedes `kubernetes/kubernetes#135581`

### Goals

- Provide an opt-in way to bound topology hint generation on large NUMA systems with many CPU-less NUMA nodes.
- Keep the effective NUMA-node set consistent across TopologyManager, CPUManager, MemoryManager, and DeviceManager.
- Preserve default behavior when the option is not enabled.

### Non-Goals

- Redesign NUMA topology discovery in cadvisor or kubelet.
- Change the semantics of `--reserved-memory`.
- Change default TopologyManager behavior for existing users.

## Proposal

Introduce a new TopologyManager alpha policy option:

- `restrict-to-cpu-numa-nodes`

When enabled:
- TopologyManager computes an effective NUMA-node set from NUMA nodes with CPUs attached.
- CPUManager and MemoryManager generate hints only across that effective NUMA-node set.
- MemoryManager generates hints only across that set.
- DeviceManager projects device NUMA topology onto that set using NUMA distance information.

When disabled:
- existing behavior is unchanged.

Example:

```yaml
topologyManagerPolicy: best-effort
topologyManagerScope: pod
topologyManagerPolicyOptions:
  max-allowable-numa-nodes: "34"
  prefer-closest-numa-nodes: "true"
  restrict-to-cpu-numa-nodes: "true"
```            

## User Stories

- As an operator of GB200 / Vera-class systems, I want kubelet topology hint generation to avoid exploring CPU-less NUMA nodes so kubelet can admit topology-aware pods reliably.
- As a Kubernetes user, I want this behavior to be opt-in so existing systems are unchanged unless I explicitly enable it.

## Design Details

### API

New TopologyManager policy option:
- `restrict-to-cpu-numa-nodes: "true|false"`

This is an alpha policy option under the existing TopologyManagerPolicyAlphaOptions feature gate.

Feature level:
- Alpha

### TopologyManager Behavior

When enabled, TopologyManager filters the discovered topology to CPU-attached NUMA nodes before constructing its effective NUMA view.
If filtering would produce an empty topology, kubelet falls back to the original topology.

### CPUManager and MemoryManager Behavior

Both managers consume the effective NUMA-node set from TopologyManager and generate hints only across that set.

### DeviceManager Behavior

Device plugins may report NUMA nodes outside the effective CPU-attached NUMA set.
When enabled, DeviceManager projects raw device NUMA-node IDs onto the effective NUMA-node set using NUMA distances, so hint generation and aligned allocation share the same reduced placement universe.

### Failure Modes

- If no CPU-attached NUMA nodes can be derived, fall back to the original topology.
- If device NUMA-distance data is unavailable, fall back conservatively.

## Risks and Mitigations

Risk:
- Some platforms may expect CPU-less NUMA nodes to remain first-class in hint generation.
Mitigation:
- option is alpha and opt-in.
Risk:
- Device projection may affect aligned allocation choices.
Mitigation:
- DeviceManager uses the same effective NUMA-node set as TopologyManager.
- Unit tests cover projected hint generation behavior.

## Test Plan

- Unit tests for TopologyManager option parsing and effective NUMA topology construction.
- Unit tests for CPUManager and MemoryManager hint generation with filtered NUMA sets.
- Unit tests for DeviceManager NUMA projection and aligned allocation behavior.
- Validation on affected large NUMA hardware such as GB200.

## Graduation Criteria

### Alpha

- KEP merged
- Pption implemented behind TopologyManagerPolicyAlphaOptions
- Unit tests added
- Validation on representative platform

### Beta

- Additional platform validation.
- No major correctness regressions reported.

### Stable

- Sufficient production confidence
- Finalized operator guidance

## Production Readiness Review Questions

### Feature Enablement and Rollback

- Feature gate: TopologyManagerPolicyAlphaOptions
- Component: kubelet
- Additional enablement: set restrict-to-cpu-numa-nodes
- Default behavior is unchanged
- Rollback is supported by removing the option

### Monitoring Requirements

No new metrics are proposed initially. Operators can use kubelet logs and admission behavior on affected nodes.

## Implementation History

- 2026-04-02: Initial draft
- 2026-04-02: Initial implementation PR opened in kubernetes/kubernetes#138172
