---
title: Add apply command to kustomize
authors:
  - "@monopole"
owning-sig: sig-cli
participating-sigs:
  - sig-cli
  - sig-api
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-03-08
last-updated: 2019-03-08
status: provisional
see-also:
  - "/keps/sig-api-machinery/0006-apply.md"
---


[DAM]: https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/declarative-application-management.md
[SSA-KEP]: /keps/sig-api-machinery/0006-apply.md
[SSA-PR]: https://github.com/kubernetes/kubernetes/pull/72947
[`KUBECONFIG`]: https://kubernetes.io/docs/tasks/access-application-cluster/configure-access-multiple-clusters
[deprecation policy]: https://github.com/kubernetes-sigs/kustomize/blob/master/docs/versioningPolicy.md
[`kustomize`]: https://github.com/kubernetes-sigs/kustomize

# Add `apply` command to kustomize

## Table of Contents

 * [Summary](#summary)
 * [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
 * [Proposal](#proposal)
    * [Risks and Mitigations](#risks-and-mitigations)
 * [Design Details](#design-details)
    * [Test Plan](#test-plan)
    * [Graduation Criteria](#graduation-criteria)
 * [Implementation History](#implementation-history)
 * [Alternatives](#alternatives)
 * [Infrastructure Needed](#infrastructure-needed)

## Summary

To the [`kustomize`] CLI add the command

```
kustomize apply $target
```

to send customized resources directly to the
_apply_ endpoint of the user's preferred
cluster.

## Motivation

Kubernetes is a level-based system to which one
_applies_ a declaration of the state one wants the
system to reach.  The system performs a three-way merge
between its current (actual) state, the previously
declared state (which may or may not have already been
reached), and the newly applied declaration of state.

The logic for performing an _apply_ operation has
    historically lived behind the `kubectl apply` command,
meaning a _client_ had to query the server for state,
perform complex merge logic, and send an appropriate
patch to the server.

In k8s v1.14, a new _apply_ API resource was introduced
with _alpha_ status, backed by three-way merge logic
[housed in the server][SSA-KEP] ([PR][SSA-PR]).  Moving this logic
to the server has many benefits, the most obvious being
that _apply_ logic can now be maintained in one place,
and accessed, via API calls, by any number of other
clients.

[`kustomize`] is a tool that gathers kubernetes
resource data from various sources, customizes
these resources for various purposes, and emits YAML
intended for application to a k8s cluster.

Until v1.14, `kubectl` housed the only means to perform
an apply, thus entire kustomization operation had to be
expressed as

```
kustomize build $target | kubectl apply -f -
```

The existence of the new endpoint allows this
to be simplified to

```
kustomize apply $target
```

leveraging the same cluster context data used by
`kubectl` (the [`KUBECONFIG`] file,
e.g. `$HOME/.kube/config`).


### Goals


 * Provide an end-to-end reference implementation
   [declarative application management][DAM]
   involving only one client, simplifying
   continuous deployment schemes that involve
   customized resources.

 * Validate the [goals of the apply KEP][SSA-KEP],
   proving that the _apply_ endpoint doesn't require
   `kubectl` logic, and supports clients other than
   `kubectl`, giving the endpoint firm grounds to move
   to _beta_ then _GA_.

 * [stretch - as needed] Move `kubectl` logic to
   supporting CLI libraries to allow more clients to
   exploit [`KUBECONFIG`].
 
 
### Non-Goals

 * No new logic in `kustomize` beyond that needed
   to send an HTTP request containing data
   `kustomize` already computes.

## Proposal

1) Add a new `apply` command which takes the same
   arguments as the existing `build` command (i.e. a
   kustomization target).

2) Build a request aimed at the correct API server per
   [`KUBECONFIG`].
   
3) In the request body, place the YAML that would have
   been emitted to `stdout` by `kustomize build
   $target`.
   
4) Send the request.

5) Report the response to `stdout` in some
   palatable form.


### Risks and Mitigations

The API server's endpoints require the proper
credentials.

`kustomize` should leverage the same client-side
credential management code currently used by `kubectl`.

There could be a review from _sig-auth_ to confirm.

## Design Details

Design covered in the [proposal section](#proposal).

### Test Plan

A set of unit tests confirms that the `apply` command
can compose a well formed request for the _apply_
endpoint.

The payload should be consistent with the chosen
kustomization.yaml file, and the target server should
correspond to the data in the local [`KUBECONFIG`].

### Graduation Criteria

The command will require use of an _--alpha_ flag, as
it targets a feature that is _alpha_ in API server
1.14.

The command can go _beta_ and _GA_ in sequence with the
endpoint it hits, as the client side logic is by
definition just request composition.

If needed, deprecation of the `apply` command could be
performed per `kustomize`'s [deprecation policy].


## Implementation History

TBD

## Alternatives

Blacklist `kustomize` from making a request to the _apply_ API endpoint.

## Infrastructure Needed

None.

