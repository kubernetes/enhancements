# KEP-4974: Deprecate v1.Endpoints

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API Server Warnings](#api-server-warnings)
  - [Additional Warnings?](#additional-warnings)
    - [Warnings via annotations](#warnings-via-annotations)
    - [Warnings in <code>kubectl get endpoints</code>](#warnings-in-kubectl-get-endpoints)
    - [Warnings via logs/metrics/Events](#warnings-via-logsmetricsevents)
  - [<code>kubernetes.default</code> Endpoints](#kubernetesdefault-endpoints)
  - [Endpoints Cleanup](#endpoints-cleanup)
  - [E2E Test Updates](#e2e-test-updates)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Deprecation - Stage 1](#deprecation---stage-1)
    - [Deprecation - Stage 2](#deprecation---stage-2)
    - [Deprecation - Stage 3](#deprecation---stage-3)
    - [Deprecation - Stage 4](#deprecation---stage-4)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
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

The `v1.Endpoints` API has been essentially (though not _actually_)
deprecated since EndpointSlices became GA in 1.21. Several new Service
features (such as dual-stack and topology, not to mention "services
with more than 1000 endpoints") are implemented only for
EndpointSlice, not for Endpoints. Kube-proxy no longer uses Endpoints
ever, for anything, and the Gateway API conformance tests also require
implementations to use EndpointSlices, so Gateway implementations
don't use Endpoints either.

Despite this, kube-controller-manager still does all of the work of
managing Endpoints objects for all Services, and a cluster cannot pass
the conformance test suite unless the Endpoints and EndpointSlice
Mirroring controllers are running, even though in many cases nothing
will ever look at the output of the Endpoints controller (and the
EndpointSlice Mirroring controller will never output anything).

While Kubernetes's API guarantees make it essentially impossible to
ever actually fully remove Endpoints, we should at least move toward a
world where most users run Kubernetes with the Endpoints and
EndpointSlice Mirroring controllers disabled.

## Motivation

### Goals

- Officially declare v1.Endpoints to be deprecated. Update
  documentation and put out appropriate communications (blog posts,
  etc) to ensure that end users are aware of this deprecation.

- Add warnings (e.g., `Warning:` headers on Endpoints create/update)
  to alert users of the fact that Endpoints is deprecated.

- Update the e2e test suite to make it possible to run it in a "no
  Endpoints controller" configuration, by rewriting some tests and
  adding feature tags to others.

- Update the conformance test suite to not require the Endpoints and
  EndpointSlice Mirroring controllers to be running, by rewriting some
  tests and demoting others from conformance.

- Explicitly document that disabling `endpoints-controller` and/or
  `endpointslice-mirroring-controller` via kube-controller-manager's
  `--controllers` flag is a supported and conforming configuration.

- Come up with a plan to clean up stale Endpoints in clusters that
  aren't running the Endpoints controller.

```
<<[UNRESOLVED kubernetes.default ]>>

- MAYBE change kube-apiserver to optionally not generate Endpoints for
  kubernetes.default, though this would require adding a
  kube-apiserver configuration option, and the benefit from removing
  just that 1 Endpoints object is small. Perhaps instead it could just
  add an annotation to the object pointing out the fact that Endpoints
  is deprecated.

<<[/UNRESOLVED]>>
```

```
<<[UNRESOLVED disable-by-default ]>>

- MAYBE eventually move `endpoints-controller` and
  `endpointslice-mirroring-controller` out of the "on-by-default"
  controller list so that they become opt-in rather than opt-out. If
  this was going to happen, it might make sense to create a separate
  KEP for that when we eventually do get around to it, since it will
  have additional PRR-ish concerns around upgrades, etc.

- MAYBE eventually switch our e2e test jobs to disable those
  controllers by default, and only run a single periodic/optional job
  to run the Endpoints-specific tests.

<<[/UNRESOLVED]>>
```

### Non-Goals

- Deleting or modifying the `v1.Endpoints` API.

- Removing or demoting the conformance tests that test the
  `v1.Endpoints` API independently of the Endpoints controller.

## Proposal

Overall, this KEP is mostly just about documentation and tests; it is
already possible to disable the Endpoints and EndpointSlice Mirroring
controllers via kube-controller-manager's `--controllers` option, and
we believe that this will have no ill effects in a vanilla Kubernetes
cluster (though, currently, it will cause the e2e tests to fail).

### API Server Warnings

Whenever any Endpoints object is created or updated (except by the
Endpoints controller), the apiserver will return the warning:

```
The Endpoints API is deprecated, and all users should use the EndpointSlice API instead.
```

### Additional Warnings?

There are other ways we could warn users about Endpoints being deprecated.

```
<<[UNRESOLVED additional-warnings ]>>

All of these ideas are UNRESOLVED.

<<[/UNRESOLVED]>>
```

#### Warnings via annotations

The Endpoints controller could potentially add an annotation to every
Endpoints object it creates:

```yaml
metadata:
  annotations:
    endpoints.kubernetes.io/deprecated: >
      The Endpoints API is deprecated, and all users should use the
      EndpointSlice API instead. Endpoints objects are not guaranteed
      to be generated in all clusters.
```

Likewise, the EndpointSlice Mirroring controller could add such an
annotation to every Endpoints object it mirrors. (It would not make
sense to annotate the EndpointSlice, since presumably anyone still
generating Endpoints objects rather than EndpointSlices is unaware of
EndpointSlices and would never look at the mirrored slice.)

The obvious problem with this idea is that part of the motivation for
this is to reduce resource usage, and adding a largish annotation to
every Endpoints object is taking one step back. (Or does etcd compress
its data in some way that would mitigate this?)

We could potentially try to have a "virtual" annotation that gets
added by the apiserver on read, and stripped back out on write, though
this might be fragile.

Having the EndpointSlice Mirroring controller modify the Endpoints
objects that it mirrors could cause also problems with controllers
that are reconciling Endpoints objects and not expecting anyone else
to modify them. (This might be problem with either "real" or "virtual"
annotations.)

#### Warnings in `kubectl get endpoints`

We could potentially have `kubectl` print warnings (to stderr) when
anyone gets or lists endpoints. There doesn't seem to be precedent for
this, but it seems like it should work.

#### Warnings via logs/metrics/Events

We could potentially warn within the apiserver about clients Getting,
Listing, or Watching Endpoints objects. (This would presumably be
generic and based on the fact that the v1.Endpoints type is marked as
deprecated. Maybe there is already something like this?)

### `kubernetes.default` Endpoints

The Endpoints for the `kubernetes.default` Service are created by the
apiserver rather than by kube-controller-manager (and are thus
independent of whether the Endpoints controller is running or not).

We have a few options here:

  1. Do nothing, continue to always generate both an Endpoints and
     an EndpointSlice for `kubernetes.default` in all clusters.

  2. As with #1, but add a deprecation warning annotation to the
     Endpoints object.

  3. Add a config option to kube-apiserver to optionally disable
     `kubernetes.default` Endpoints generation (possibly with the
     warning as in #2 in the case where it's not disabled).

  4. Do #1, #2, or #3, but eventually (maybe a long time in the
     future) unconditionally stop generating the Endpoints object.

Whether #4 is allowed or not depends on whether we think there is an
"API"-type guarantee that a `kubernetes.default` Endpoints object
exists (or if the "API" is merely that the `kubernetes.default`
_Service_ exists and "works", without reference to _how_ it works).

```
<<[UNRESOLVED kubernetes.default ]>>

Pick an option.

<<[/UNRESOLVED]>>
```

### Endpoints Cleanup

We do not want to leave stale `Endpoints` objects around forever if
the Endpoints controllers are not running. (This is both a waste of
disk space and a potential source of confusion since the Endpoints
objects would quickly become out-of-date and incorrect.)

One possibility would be to just document that administrators should
delete all existing Endpoints themselves if they are going to disable
the controllers.

Another possibility would be to have kube-controller-manager do this
automatically. (Though the idea of having it automatically take action
because a controller _isn't_ enabled seems slightly magical? But
perhaps there is precedent somewhere else?)

Another possibility would be to create an `endpoints-cleanup`
controller, that could be enabled explicitly, and document that admins
should (probably) enable that controller if they disable the others.

Or perhaps the EndpointSlice controller could delete Endpoints objects
that were more than 24 hours out of date with respect to their
EndpointSlices?

In all cases, we should probably not automatically delete Endpoints
that were not originally created by the Endpoints controller.

```
<<[UNRESOLVED endpoints-cleanup ]>>

Decide what to do here.

<<[/UNRESOLVED]>>
```

### E2E Test Updates

There are a surprising number of e2e tests that still make use of
Endpoints, mostly because there was never any active effort to port
old tests away. These will need to be updated. See the [e2e
tests](#e2e-tests) section for more details.

### Risks and Mitigations

Obviously if a cluster contains components that read Endpoints
objects, then disabling the Endpoints controllers would break those
clusters. Given that the `v1.Endpoints` type would still exist in
these clusters, the failure mode would not be the component would fail
entirely with errors; it would just think that the Endpoints it was
looking for didn't exist ("yet"?).

We would need to mitigate this by helping users to figure out if
anything in their cluster depends on Endpoints.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

N/A

##### Unit tests

We will have unit tests for any new code (e.g. apiserver warnings).

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

TBD

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

We will want to add a new periodic e2e job that confirms that the e2e
suite passes in a cluster with the Endpoints controllers disabled.
This will require also adding a Feature tag to allow skipping the
Endpoints-specific tests.

There are quite a few places in the e2e tests that currently use
`v1.Endpoints`:

  - Many of the tests in `test/e2e/network/service.go` do various
    checks on both Endpoints and EndpointSlices. This will need to be
    split into separate tests, with the Endpoints tests
    feature-tagged.

  - Some of the tests in `test/e2e/network/endpointslice.go` will need
    to be split up, to separately test "EndpointSlices are created
    correctly" and "EndpointSlices match Endpoints" in separate tests,
    with the Endpoints tests feature-tagged.

  - The conformance tests in
    `test/e2e/network/endpointslicemirroring.go` should be demoted
    from conformance, and all of the tests should be feature-tagged,
    but should otherwise be unchanged.

  - Several tests in `test/e2e/network/dual_stack.go` check that the
    Endpoints controller does the right thing but _do not_ check that
    the EndpointSlice controller does the right thing (which means
    that we do not actually have any proper e2e testing of dual-stack
    Services). These should be updated to test EndpointSlices, with
    the Endpoints tests split out into separate feature-tagged tests.

  - `[It] [sig-network] Services should test the lifecycle of an
    Endpoint [Conformance]`: This just tests that the API works and
    does not test the behavior of the controllers, so it just needs to
    be feature-tagged, but not changed.

  - `[It] [sig-network] Service endpoints latency should not be very
    high [Conformance]`: This tests the latency of the Endpoints
    controller, and should be fixed to test the latency of the
    EndpointSlice controller instead, since the latency of the
    Endpoints controller has no impact on the functioning of a
    cluster.

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

This is a deprecation, not a new feature, so there are no
Alpha/Beta/GA stages. However, the deprecation will take place over
multiple releases.

```
<<[UNRESOLVED ordering ]>>

Confirm that we want to change the conformance criteria late in the
process rather than early.

<<[/UNRESOLVED]>>
```

#### Deprecation - Stage 1

- Mark v1.Endpoints as deprecated in the API.

- Ensure that all official Kubernetes documentation primarily
  discusses EndpointSlices, and mentions Endpoints only as a
  deprecated API. Except where we are explicitly documenting
  Endpoints, no examples should use `kubectl get endpoints` or involve
  creating an Endpoints object.

- Write a blog post about the deprecation and the overall KEP plan,
  and mention it in the "mid point comms" blog post prior to the
  release.

#### Deprecation - Stage 2

Stage 2 can begin as soon as Stage 1 is complete, and does not need to
be completed all at once.

- Warnings are emitted when users create Endpoints.

- If we are going to add logs/metrics/Events/etc, then they are added
  and have tests.

- e2e tests have been updated/reorganized so that all tests that
  depend on the Endpoints controller are in a single test suite in
  `test/e2e/network/endpoints.go` and all tests that depend on the
  EndpointSlice controller are in a single test suite in
  `test/e2e/network/endpointslicemirroring.go`. (The latter may
  already be true.)

- We create a periodic e2e job that runs with the Endpoints and
  EndpointSlice Mirroring controllers disabled, and with the
  associated tests skipped, and confirm that it passes.

#### Deprecation - Stage 3

Stage 3 will not happen until the relevant SIGs feel that enough time
has passed since the initial deprecation announcement.

- The tests depending on the Endpoints and EndpointSlice Mirroring
  controllers are demoted from conformance, and there is additional
  communication (e.g. blog post) about this.

- The documentation is updated to explain how to disable the Endpoints
  and EndpointSlice Mirroring controllers, but warns that third-party
  components may still depend on them.

#### Deprecation - Stage 4

Stage 4 will not happen until the relevant SIGs feel reasonably
confident that the Kubernetes ecosystem has mostly moved away from
Endpoints.

- The documentation becomes more bullish on the idea of disabling the
  controllers, implying that it's a reasonable default.

### Upgrade / Downgrade Strategy

The KEP does not propose any automatic change to behavior; behavior
would only be changed when the administrator chose to disable the
controllers, which could happen at any time.

### Version Skew Strategy

There are no skew issues because there are no new or modified APIs,
and because no (recent) versions of Kubernetes ever look at Endpoints.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

The KEP does not define a new feature, it just proposes that we
document that users are allowed to disable the Endpoints and
EndpointSlice Mirroring controllers.

###### Does enabling the feature change any default behavior?

The KEP does not define a new feature that can be enabled.

(Obviously disabling the endpoints controllers changes default
behavior, but this would be because the administrator chose to do
that, not something that would happen automatically.)

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

The KEP does not define a new feature that can be enabled/disabled.

(If an administrator chooses to disable the endpoints controllers, and
then decides this was a bad idea, they can re-enable them, and even if
something had previously deleted all of the autogenerated Endpoints
objects, re-enabling the endpoints controller would regenerate them.)

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

N/A: The KEP does not define a new feature that can be enabled/disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Disabling the endpoints controllers in a cluster where some
third-party components depend on them could have arbitrarily bad
effects.

###### What specific metrics should inform a rollback?

There are currently none. There could perhaps be metrics counting
Gets, Lists, and Watches of v1.Endpoints, which might suggest the
existence of components that had not been updated to use
EndpointSlices.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The feature would be enabled by the operator.

###### How can someone using this feature know that it is working for their instance?

- [X] Other (treat as last resort)
  - Details: 

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

There are no specific SLOs, but kube-controller-manager,
kube-apiserver, and etcd should use less CPU, and etcd should use less
disk space, if the controllers are disabled.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

(See above about metrics informing a rollback.)

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No; the goal of this KEP is to drastically reduce the overall size of
the API database.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No; the goal of this KEP is to drastically reduce the overall size of
the API database, and to somewhat reduce the CPU usage of
kube-controller-manager, kube-apiserver, and etcd.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

None

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- Initial proposal: 2024-11-21
