# KEP-3805: Server-Side Apply default in Kubectl

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
    - [Story 3](#story-3)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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

Server-Side Apply has been a major new feature of Kubernetes, and has
landed as GA in Kubernetes v1.22. Unfortunately, while the feature is
accessible through `kubectl apply –server-side=true`, new objects,
existing objects, and even previously server-side applied objects are
still client-side applied by default unless the proper flag value is
specified. The main reason is that Server-Side Apply is not entirely
backwards compatible with client-side, and breaking users of `kubectl
apply` now would come with a cost.

We’re proposing a way forward toward toggling the `--server-side` flag
by default.

## Motivation

While changing from client-side to server-side is difficult because it
will break some people's expectations and might break some scripts too,
but the feature is required none-the-less for its benefits, for users:
- Users are missing out on Server-Side Apply feature because they don't
  know they can use it
- Having both server-side and client-side feature is confusing for users
  who've tried the feature, and changing from one to the other can cause
  odd behaviors
- Some flags in kubectl are meant for this historical CSA while some
  flags are meant for SSA, causing even more confusion
- strategic-merge-patch (SMP) is not maintained and broken, a
  frustrating situation for users

For maintainers:
- Many workflows need to be considered for both paradigms, making
  maintenance of kubectl more difficult
- kubectl has a lot of code to maintain SMP and other client-side apply
  mechanisms. Removing the feature can greatly reduce the complexity of
  kubectl
- Removes some ugly server-side code meant to deal with transition from
  client-side apply to server-side apply

A lot of the benefits of using server-side apply has already been
discussed in blog-posts, see
https://kubernetes.io/blog/2022/10/20/advanced-server-side-apply/.

### Goals

The goal is to increase usage of Server-Side Apply so that users can
benefits from the feature, mostly by turning the feature on by default.
We also want to reach that goal by making the transition as smooth as
possible, and limit the risk of converting objects from client-side to
server-side mode.

### Non-Goals

N/A

## Proposal

The feature, in its alpha phase, consists of adding a new `auto` value
to the `--server-side` flag for `kubectl apply` (and corresponding
`kubectl diff`) while keeping `false` as the default value. All the
other values for the flag would continue to work as expected (for
reference, `false` continues to client-side apply, and `true` continues
to server-side apply).

The meaning of `auto` goes as follows:
- Resources continue to be fetched (GET) before-hand
- If the resource has a kubectl `last-applied` annotation, we infer that
  the resource is client-side applied, and we continue to client-side
  apply that resource
- If the resource is new (GET returns 404), the resource is server-side applied
- If the resource already exists but doesn't have the `last-applied`
  annotation, the resource is server-side applied. Note that this will
  treat previously "converted" objects as client-side apply.

For the alpha phase, the auto value will only be visible and usable if
the `KUBECTL_AUTO_SERVER_SIDE` is set. That variable will later be
removed once the flag is available for everyone, without breaking any
compatibility.

Our ultimate goal is to switch the `--server-side` flag to `true` as
early as permissible by Kubernetes deprecation policies. If the terms of
the change are finalized in time for code freeze, we will add a warning
and blog-post about this as part of the 1.27 release.

<<[UNRESOLVED What default value for --force-conflict]>>
We're not entirely sure what the value of `--force-conflict` should be
when we switch to `server-side=true`.

A few thoughts. The question can be summarized with the following
trade-off: Are conflicts worth breaking people? We would also have to
change the value of `--force-conflict`, which will break some people,
but arguably a much smaller set of people.

We don't know yet, but I'll assume we don't flip the force-conflict
switch (keep it to false) in the rest of the document since that
use-case is more complicated.
<<[/UNRESOLVED]>>

<<[UNRESOLVED Can we add the flag if we agree on terms before code-freeze?]>>
We know we want a warning, but since we don't know what value of
force-conflict we want yet, we don't know what the warning will look
like, we would still love to insert the warning in 1.27 if we can.
<<[/UNRESOLVED]>>

<<[UNRESOLVED Removal of CSA]>>
We considered removing the
`--server-side` flag altogether 2-3 releases after `true` becomes the
default. Curious if other folks think this is an option.
<<[/UNRESOLVED]>>

### User Stories (Optional)

#### Story 1

User 1 uses `--server-side=auto` and starts using `kubectl` for the
first time, creates a new project, everything will always be server-side
applied, they will never have any object to migrate from client-side to
server-side apply. When the flag eventually fips, they
already have the hard-coded auto value so they don't really see a
change, but they can decide to remove the flag if they want.

#### Story 2

User 2 uses `--server-side=auto` and periodically re-creates their
project, at first, their project might have a mix of server-side applied
and client-side applied resources, but they will eventually have only
server-side applied resources, without ever having to migrate any
resource from one to the other. When the flag eventually fips, they
already have the hard-coded auto value so they don't really see a
change, but they can decide to remove the flag if they want.

#### Story 3

Users 3 who use either the default value of `--server-side` or an
existing value will see no immediate change to the behavior. They will
get the warning if running manually and possibly try it out, or they may
miss the warning in scripts. In 3 releases, people who have completely
missed the warning will see their thing break, they can either fix it by
changing the value of `--server-side=false` or the value of
`--force-conflict=true`, a fairly easy change.

### Risks and Mitigations

This design has no impact on security, but some consequences on UX, since the
default UX of `kubectl apply` will change, 3 releases from now:
- Currently, it can never fail because of conflict, since they are
  overridden by default. The new default for Server-Side Apply is to
  fail on conflicts, unless the `--force-conflict` flag is used. While
  people can re-run the command with the flag, this might impact CI/CD
  scripts that use `kubectl apply` directly, since they may not have a
  break-glass way to address that.
- CSA injects a last-applied-annotation into the objects that it
  applies, but these don’t make sense in the context of SSA. An API or
  tool that would use this annotation to detect which resources have
  been applied would fail to find it for server-side applied objects.
  Note that doing that is heavily frowned upon.
- CSA users expect all excluded fields to be removed from the applied
  object on the server. SSA has more complicated semantics for removing
  fields (e.g. if another user manages fields)

This will be mitigated by laying out a plan to address these issues for
people who meet them, through a blog-post. Actual mitigation is to set
`--server-side=false` or `--force-conflict=true`.

## Design Details

As mentioned in the proposal, the new value of `auto` will detect if a
resource has been client-side applied before and will continue to
client-side apply these resources, while all other resources will be
server-side applied. This means that new resources will be server-side
applied by default.

The `last-applied` annotation is used to detect previous client-side
applied object, hence it is not inserted for server-side objects.

`kubectl apply` flags that are specific to client-side apply will
continue to work for client-side applied objects (prune, overwrite),
while server-side apply flags will apply to existing server-side applied
resources and new resources. Notably, the `--force-conflict` flag is
somewhat incompatible with `--server-side=false` today, it will be
possible to use the flag with `auto` (and we will entirely disallow it
with `--server-side=false`).

Some other commands might be impacted, especially when/if we remove
client-side apply altogether. This would deserve a full section once the
details of this are carved out. `kubectl create` (and family) notably
have a `--save-config` flag that create the last-applied annotation.
While I don't know how many people actually use the flag, the idea of
saving this as a config is confusing, since people don't actually have
the file and so the situation doesn't really fit well the `apply`
workflow. We suggest adding a warning when this flag is used, as well as
updating its documentation to suggest not using it, and possibly
deprecate it in the future.

<<[UNRESOLVED More details needed on exhaustive CSA clean-up]>>
Deprecating client-side apply from kubectl probably implies a LOT more
clean-up that is actually described here, and we would have to go
through the details when we decide the fate of client-side apply.
<<[/UNRESOLVED]>>

Because `kubectl diff` is supposed to map the behavior of `kubectl
apply` as closely as possible, the change will also be done for that
command.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
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

- `k8s.io/kubernetes/vendor/k8s.io/kubectl/pkg/cmd/apply`: `2023-01-24` - `76.5%`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

CLI tests will be added to both `test/cmd/diff.sh` and `test/cmd/apply.sh`.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

e2e tests will be added to check that:
- Migration from `auto` to `true` or `false` works as expected
- Auto behaves properly both on client-side applied and server-side applied objects
- As we add the deprecation, it is exposed given the right circumstances

### Graduation Criteria

#### Alpha

Alpha is the current level of the feature since server-side apply is
currently enabled by default in Kubernetes, but enabled on-demand by
kubectl users.

The feature already has a fair amount of usage since tools (sometimes
outside of kubectl) have used it both as "clients" and in controllers.

#### Beta

The environment variable to protect `auto` will be removed in Beta based
on feedback, so that the `auto` flag becomes available to all.

<!--
To be re-evaluated later:
Server-Side Apply has a very limited set of bugs or feature requests as
this point and is definitely mature. Enabling client-side will allow
increased usage and reduce burden cost for kubectl to maintain both
mechanisms, if we remove client-side apply. -->

#### GA

<!--
To be re-evaluated later:
Kubectl doesn't have real-time metrics for usage. The decision to move
to server-side entirely by default (if ever enabled) will be driven by
bug reports and complaints from customers. Also by the ability to
migrate existing client-side usage to server-side. -->

#### Deprecation

If we decide to remove the remove the `--server-side` flag, we would
have a deprecation warning at least two releases before. Same thing
applies for `--save-config` and other client-side related flags in
kubectl which we might remove.

### Upgrade / Downgrade Strategy

While upgrade / downgrade doesn't really apply to a kubectl feature, we
currently have a upgrade (and somewhat downgrade) feature in kubectl to
go from client-side to server-side apply. The upgrade and downgrade
works well in the nominal cases but fail with special cases. Enabling
server-side by default also intends to address that problem.

Going back from `--server-side=auto` to `--server-side=false` or
`--server-side=true` would trigger the same upgrade/downgrade strategy
mentioned above for each object that don't match the new mode.

### Version Skew Strategy

N/A.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Other
  - Describe the mechanism:

Feature is enabled on-demand, and can stop using it at any time. It also
requires the `KUBECTL_AUTO_SERVER_SIDE` environment variable to be set.

  - Will enabling / disabling the feature require downtime of the control
    plane?

Feature is client-side only, no.

  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

Feature is client-side only, no.

###### Does enabling the feature change any default behavior?

No, enabling the `auto` feature means actively selecting it.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

The feature can be disabled by unsetting the `KUBECTL_AUTO_SERVER_SIDE` environment variable.

###### What happens if we reenable the feature if it was previously rolled back?

Toggling the flag value can have impact, but that's already the case.
Changing the value to an automatic detection will actually help with
that.

###### Are there any tests for feature enablement/disablement?

Since enablement is not done through a feature gate and/or command-line
flags, tests are fairly easy to implement.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

In the context of this change, a rollback would possibly be someone
switching from server-side apply to client-side apply and vice-versa (or
from `auto` to another forced value) This problem isn't new and is one
of the reason that motivates this change. It mostly works well when the
default fieldmanager is being used.

<<[UNRESOLVED bad description of the problems here ]>>
We could certainly make a much better analysis of the problems we have
migrating from server-side to client-side and vice-versa here.
<<[/UNRESOLVED]>>

###### What specific metrics should inform a rollback?

People should inform their decision based on the direct error they get
from kubectl as they are trying to apply their resources.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

We have a series of tests for that.

<<[UNRESOLVED find the tests ]>>
I need to add the tests here.
<<[/UNRESOLVED]>>

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

This will eventually change the default behavior of `kubectl apply` in
possibly surprising way. `kubectl apply` can now fail because of a
conflict, which it wouldn't before on newly created objects. Another
surprising behavior is that since fields can be owned by multiple
actors, removing a field will not necessarily remove the field from the
cluster's resource.

### Monitoring Requirements

N/A. No monitoring in place.

###### How can an operator determine if the feature is in use by workloads?

`kubectl apply -V8` can help identify what type of apply is used, or the
output of the command when applying ("resource has been server-side
applied").

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details: Kubectl will fail with an error right away.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

Feature depends on server-side apply being enabled in the cluster, but
the feature has been GA since 1.22 and is now guaranteed to be present
in all clusters supported by the kubectl version (1 version schew).

### Scalability

###### Will enabling / using this feature result in any new API calls?

For now, we will continue to get resources before server-side applying,
which means that the number of requests will be similar (one GET, one
PATCH). This is done because we need to verify if the resource needs to
be client-side applied or server-side applied.

It's important to note that server-side apply ALWAYS sends a patch while
client-side could by-pass the send. This has an impact on the apiserver
since all the requests need to be processed, including by webhooks which
can increase the cluster load.

As the server-side field becomes auto, the initial get could absolutely
be removed saving an extra request.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

This should reduce the size of api objects since the last applied
annotation is not going to be introduced in new objects.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

PATCH requests are going to be more frequent since all the applied
objects will be sent to the server.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

API Server might have to process more requests, leading to increased CPU
and RAM usage. Non-changes should continue to be skipped by etcd, which
shouldn't result in a major disk/IO increase.

### Troubleshooting


###### How does this feature react if the API server and/or etcd is unavailable?

N/A.

###### What are other known failure modes?

Main failure mode is if kubectl fails to apply the resource, which will
lead to a direct error to the user.

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

N/A

## Drawbacks

Mainly two drawbacks to this KEP:
1. Resource usage will increase on apiservers since patch requests will
   have to be processed everytime.
2. This will change the default behavior of kubectl apply which might
   surprise some users.

## Alternatives

There are basically three different alternatives to the current design:
1. Status-quo: we keep client-side as the default for everything,
   offering the `--server-side=true` flag for who wants to use it.
   Drawback is that this situation is still confusing for users, and
   users are missing out on the new features of server-side apply.
2. Make `--server-side=auto` continue to client-side apply by default,
   but automatically server-side objects that have been previously
   server-side applied. The drawback of this approach is that the baby
   step doens't really get us anywhere close to where we want: having
   server-side apply enabled by default.
3. Make `--server-side=auto` the default, but that doesn't get us closer
   to what we want: have the feature enabled by default, or test the
   migration.


## Infrastructure Needed (Optional)

N/A
