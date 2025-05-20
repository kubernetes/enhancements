# KEP-3685: Move EndpointSlice Reconciler into Staging

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Dependencies](#dependencies)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

Move the EndpointSlice reconciler and it’s dependencies to staging so that the
reconciler logic be reused by out-of-tree EndpointSlice controllers.

Some changes are needed in `pkg/controller/endpointslice` to move the reconciler
code to `staging/src/k8s.io/endpointslice` and then import it as Go module.
These changes include making private methods public and updating the import
paths.

## Motivation

Moving the EndpointSlice reconciler to staging has several benefits:

1. The reconciler logic can be reused by custom EndpointSlice controllers. The
   fanout behavior is difficult to implement correctly and the current
   reconciler code has gone through several iterations to reach its currently
   level of robustness. There are cases where reconciler code has either been
   forked or rewritten, both of which are problematic from a supportability
   perspective.
2. It reduces the burden of fully migrating from Endpoints to EndpointSlices.
   It’s relatively easy to write a custom Endpoints controller, but less so with
   EndpointSlices. So, the migration path is unclear for pre-existing custom
   Endpoints controllers. Having a library that handles the more difficult
   aspects of managing EndpointSlices would mitigate some of the risk in
   migrating these controllers.
3. It would help to realize one of the design goals of EndpointSlices: that
   other controllers would exist to manage EndpointSlices, which is why the
   `endpointslice.kubernetes.io/managed-by` annotation was added.

### Goals

- Expose a simple, and highly maintained, API for EndpointSlice reconciling.
- Allow for the EndpointSlice reconciling code to easily be imported into other
  projects.

### Non-Goals

- A generic solution for reconciling fan-out style resources.
- Providing the ability for a Service to opt-out of the in-tree EndpointSlice
  controller. This was considered and determined to be out of scope.

## Proposal

1. Add the staging repository (`staging/src/k8s.io/endpointslice`) to
   `kubernetes/kubernetes` by following the steps in the [staging directory
   README][staging-readme].
2. Move the reconciler submodule and it’s [dependencies](###dependencies) to the
   staging directory and update imports in `pkg/controller/endpointslice`. This
   should be a single atomic change.
    - Some methods need to be made public to allow for reconciler code to be
      used as a library (at least `NewReconciler` and `Reconcile`).
3. Allow for the value of the `endpointslice.kubernetes.io/managed-by` label to
   be configurable by adding `managedBy` as an argument to `NewReconciler`.
4. Split computing the diff (EndpointSlices to create, update, and delete) and
   applying the diff into separate methods. This allows for controllers to
   easily add metadata to the managed EndpointSlices (e.g. adding an
   annotation).

[staging-readme]: https://github.com/kubernetes/kubernetes/tree/master/staging "External Repository Staging Area"

### Dependencies

The following dependencies would also have to be moved to the staging repository
(assume `staging_root = staging/src/k8s.io/endpointslice`):
- `pkg/controller/endpointslice/metrics       => $staging_root/metrics`
- `pkg/controller/endpointslice/topologycache => $staging_root/topologycache`
- `pkg/controller/util/endpoint               => $staging_root/util`
- `pkg/controller/util/endpointslice          => $staging_root/util`

The following dependencies don’t have to be moved as there’s only one trivial
function used from each:
- `pkg/api/v1/pod`
    - `IsPodReady` function can be moved to `component-helpers` as it’s also
      used by `kubectl`.
- `pkg/apis/core/v1/helper`
    - `IsServiceIPSet` function can be moved to staging or mirrored.
- `pkg/apis/discovery/validation`
    - `ValidateEndpointSliceName` is a re-export of `NameIsDNSSubdomain` from
      `k8s.io/apimachinery`, which is already in staging.

### Risks and Mitigations

Users might expect compatibility to be maintained for the library’s public API.
This is a risk for all staging repositories, and the current mitigation strategy
has been to clearly document that there are no compatibility guarantees in the
README:

> There are NO compatibility guarantees for this repository. It is in direct
> support of Kubernetes, so branches will track Kubernetes and be compatible
> with that repo. As we more cleanly separate the layers, we will review the
> compatibility guarantee.

## Design Details

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

## Production Readiness Review Questionnaire

N/A

## Implementation History

## Drawbacks

This change will add some maintenance burden to the EndpointSlice controller as
its code will be spread across `pkg/controller` and `staging`.

## Alternatives

Externalize the entire EndpointSlice controller and use alternate data sources
(informers) to customize behavior.
