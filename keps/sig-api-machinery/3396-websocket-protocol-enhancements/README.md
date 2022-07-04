# KEP-3396: WebSocket protocol enhancements

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
- [Design Details](#design-details)
  - [Protocol changes](#protocol-changes)
  - [Protocol negotiation](#protocol-negotiation)
  - [Test Plan](#test-plan)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

kubectl commands `exec`, `cp`, `attach`, and `port-forward` currently use [SPDY](https://en.wikipedia.org/wiki/SPDY)
to communicate with API server.
[SPDY is deprecated and its use is causing issues](https://github.com/kubernetes/kubernetes/issues/7452).
API server already supports WebSocket for those API endpoints and the plan is to [migrate the commands to
WebSocket protocol](https://github.com/kubernetes/kubernetes/issues/89163). Unfortunately, the application-level
protocol that API server implements on top of WebSocket has
[deficiencies](https://github.com/kubernetes/kubernetes/issues/89899) that block the transition.
This KEP aims to resolve them to unblock the migration and to allow all API consumers to use WebSocket rather than SPDY.

## Motivation

API server defines an application-level protocol on top of WebSocket connection. It's a framing protocol
that allows to multiplex several logical bidirectional channels on top of a single WebSocket connection.

Deficiencies in the current (v4) WebSocket protocol:

- No mechanism to do a half-close of a logical channel. Neither client nor server can indicate that no
  more data is coming on a particular channel. This makes it impossible to implement certain use-cases in those
  kubectl commands. For example, `kubectl cp file pod` spawns `tar` in the target pod and streams the contents of a file
  as a tar stream via STDIN to the `tar` command. It needs to close STDIN to indicate that all the data has been sent.
  If it cannot do it, `tar` gets stuck forever, waiting for more data.

- No mechanism to do a reset of a logical channel to stop the other party from sending more data. This is needed for
  parity with SPDY and is used mostly in error handling situations.

### Goals

- Define a new WebSocket protocol version (v5) without the deficiencies described above.
- Change kubectl to use new WebSocket protocol for `exec`, `cp`, and `attach`. It should fall back to SPDY when
  new protocol is not supported by the server.

### Non-Goals

- Removal of SPDY in any part of the system. In particular, API server will still have to support SPDY for some time
  for backwards compatibility with older clients.
- Moving `port-forward` to WebSocket. It's not clear how `port-forward` via WebSocket should work. This can be done
  separately later.

## Proposal

- Define and implement a protocol command to half-close a logical channel. Only for new protocol version (v5).
- Define and implement a protocol command to reset a logical channel. Only for new protocol version (v5).
- Change kubectl to use the new WebSocket protocol when it is supported, falling back to SPDY when it's not. 

### User Stories (Optional)

#### Story 1

#### Story 2

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

## Design Details

### Protocol changes

Currently, API server supports [binary and base64 WebSocket protocols](https://github.com/kubernetes/kubernetes/blob/3ddd0f45aa91e2f30c70734b175631bec5b5825a/staging/src/k8s.io/apiserver/pkg/util/wsstream/conn.go#L34-L64). 

Binary protocol: channel id `0xFF` can be used as a control channel. First byte of the message contents can be used
to encode the command and the rest of the message can be used to encode the payload for the command.

Binary protocol control commands:

- `0x00`: half-close the channel. Next byte is the channel id.
- `0x01`: reset the channel. Next byte is the channel id.

Base64-based protocol: can use character `c` (for `control`) as the channel id for the control channel.

Base64-based protocol control commands:

- `h`: half-close the channel. Next character is the channel id.
- `r`: reset the channel. Next character is the channel id.

### Protocol negotiation

If we have a new protocol, then older servers will not have support for it. In that case a newer client will try
v5 but an older server will not accept that. To be backwards compatible, ideally client should use the intended
mechanism that HTTP `Upgrade` provides - send a list of protocols. I.e. client would send a header with
something like `Upgrade: v5.channel.k8s.io, SPDY/3.1` and server would pick a protocol it supports,
taking into account client's preference (the list is ordered).

The difficulty with the above is that both Gorilla WebSocket and SPDY libraries are not built to allow to
use them this way. Both encapsulate negotiation, and you cannot compose them to get the above behavior. 
Refactoring them is not an option since SPDY is dead and nobody will spend time on the library and Gorilla
WebSocket is [looking for maintainers](https://github.com/gorilla/websocket/issues/370) and hence
a PR is unlikely to get merged quickly even if at all (since this is a big API addition/change).

An alternative is to try to establish `v5.channel.k8s.io` and fall back to `SPDY/3.1` but that is 2x
the number of requests when newer client is used with an older server. Perhaps this is acceptable
since `exec`, `attach`, and `cp` are I/O-heavy anyway and are not used all the time. This is a temporary
overhead, eventually this situation will become less and less frequent as servers upgrade to a version with
the new protocol. If the overhead is not acceptable, user can almost always use a client of a matching version.
At some point SPDY support should be removed and this problem will go away completely.

### Test Plan

kubectl commands that this KEP proposes to update should already be covered by unit, and end-to-end tests.
They should all pass, perhaps with minor updates to unit tests. Additional unit tests should be added for new code.

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

- <test>: <link to test coverage>

##### e2e tests

- <test>: <link to test coverage>

### Graduation Criteria

### Upgrade / Downgrade Strategy

### Version Skew Strategy

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

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

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

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

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

### Scalability

###### Will enabling / using this feature result in any new API calls?

###### Will enabling / using this feature result in introducing new API types?

###### Will enabling / using this feature result in any new calls to the cloud provider?

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

## Drawbacks

## Alternatives

## Infrastructure Needed (Optional)
