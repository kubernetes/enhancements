---
title: Seccomp to GA
authors:
  - "@tallclair"
owning-sig: sig-node
participating-sigs:
  - sig-api-machinery
  - sig-auth
reviewers:
  - "@liggitt"
  - "@derekwaynecarr"
  - "@dchen1107"
  - "@mrunalp"
approvers:
  - "@liggitt"
  - "@derekwaynecarr"
editor: TBD
creation-date: 2019-07-17
status: provisional
---

# Seccomp to GA

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API](#api)
    - [Pod API](#pod-api)
    - [PodSecurityPolicy API](#podsecuritypolicy-api)
- [Design Details](#design-details)
  - [Version Skew Strategy](#version-skew-strategy)
    - [Pod Creation](#pod-creation)
    - [Pod Update](#pod-update)
    - [PodSecurityPolicy Creation](#podsecuritypolicy-creation)
    - [PodSecurityPolicy Update](#podsecuritypolicy-update)
    - [PodSecurityPolicy Enforcement](#podsecuritypolicy-enforcement)
    - [PodTemplates](#podtemplates)
    - [Upgrade / Downgrade](#upgrade--downgrade)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [References](#references)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

This is a proposal to upgrade the seccomp annotation on pods & pod security policies to a field, and
mark the feature as GA. This proposal aims to do the _bare minimum_ to clean up the feature, without
blocking future enhancements.

## Motivation

Docker started enforcing a default seccomp profile in v1.10. At the time, Kubernetes (in v1.2)
didn't have a way to control the seccomp profile, so the profile was disabled (set to `unconfined`)
to prevent a regression (see https://github.com/kubernetes/kubernetes/pull/21790). In Kubernetes
v1.3, annotations were added to give users some control over the profile:

```
seccomp.security.alpha.kubernetes.io/pod: {unconfined,docker/default,localhost/<path>}
container.seccomp.security.alpha.kubernetes.io/<container_name>: ...
```

The feature has been more or less unchanged ever since. Also note that the addition predates feature
gates or our modern concept of feature lifecycle. So, even though the annotations include `alpha` in
the key, this is entirely useable on any production GA cluster.

There have been multiple attempts to [change the default
profile](https://github.com/kubernetes/kubernetes/issues/39845) or [formally spec the Kubernetes
seccomp profile](https://github.com/kubernetes/kubernetes/issues/39128), but both efforts were
abandoned due to friction and lack of investment.

Despite the `alpha` label, I think this feature needs to be treated as GA, and we're doing our users
a disservice by leaving it in this weird limbo state. As much as I would like to see seccomp support
fully fleshed out, if we block GA on those enhancements we will remain stuck in the current state
indefinitely. Therefore, I'm proposing we do the absolute minimum to clean up the current
implementation all accurately declare the feature "GA". Future enhancements can follow the standard
alpha -> beta -> GA feature process.

_NOTE: AppArmor is in a very similar state, but with some subtle differences. Promoting AppArmor to
GA will be covered by a separate KEP._

### Goals

- Declare seccomp GA
- Fully document and formally spec the feature support
- Add equivalent API fields to replace seccomp annotations
- Deprecate the seccomp annotations

### Non-Goals

This KEP proposes the absolute minimum to get seccomp to GA, therefore all functional enhancements
are out of scope, including:

- Changing the default seccomp profile from `unconfined`
- Defining any standard "Kubernetes branded" seccomp profiles
- Formally speccing the seccomp profile format in Kubernetes
- Providing mechanisms for loading profiles from outside the static seccomp node directory
- Changing the semantics around seccomp support
- Windows support (seccomp is very linux-specific)

## Proposal

### API

The seccomp API will be functionally equivalent to the current alpha API. This includes the Pod API,
which specifies what profile the pod & containers run with, and the PodSecurityPolicy API which
specifies allowed profiles & a default profile.

#### Pod API

The Pod Seccomp API is immutable, except in [`PodTemplates`](#podtemplates).

```go
type PodSecurityContext struct {
    ...
    // The seccomp options to use by the containers in this pod.
    // +optional
    Seccomp  *SeccompProfile
    ...
}

type SecurityContext struct {
    ...
    // The seccomp options to use by this container. If seccomp options are
    // provided at both the pod & container level, the container options
    // override the pod options.
    // +optional
    SeccompProfile  *SeccompProfile
    ...
}

// Only one profile source may be set.
// +union
type SeccompProfile struct {
    // +unionDescriminator
    Type SeccompProfileType
    // Use a predefined profile defined by the runtime.
    // Most runtimes only support "default"
    // +optional
    RuntimeProfile *string
    // Load a profile defined in static file on the node.
    // The profile must be preconfigured on the node to work.
    // +optional
    LocalhostProfile *string
}

type SeccompProfileType string

const (
    SeccompProfileUnconfined SeccompProfileType = "Unconfined"
    SeccompProfileRuntime    SeccompProfileType = "Runtime"
    SeccompProfileLocalhost  SeccompProfileType = "Localhost"
)
```

This API makes the options more explicit than the stringly-typed annotation values, and leaves room
for new profile sources to be added in the future (e.g. Kubernetes predefined profiles or ConfigMap
profiles). The seccomp options struct leaves room for future extensions, such as defining the
behavior when a profile cannot be set.

<<[UNRESOLVED]>>
What to do with the localhost profile type, given that we want to deprecate it? Alternative for
consideration:

Drop the LocalhostProfile *string field. Keep the SeccompProfileLocalhost SeccompProfileType, but
optionally change its value to LocalhostDeprecated.

When creating a pod, the profile type can only be set to "Localhost" if the annotation is also set
to localhost. When the kubelet goes to enforce the localhost profile, it fetches the path from the
annotation.

The one gotcha is how to handle annotation update in this case. I'm tempted to say allow the update,
and if the kubelet goes to enforce it and the annotation isn't set to localhost anymore, just treat
it as an invalid localhost path (fail the pod).
<<[/UNRESOLVED]>>

<<[UNRESOLVED]>>
What to do with RuntimeProfile field? There aren't any runtimes that support multiple built in
profiles, and this feature has never been requested.

If we dropped this field for now (just assume it's runtime/default), how bad would it be to add it
back at some point in the future if it was needed? As long as we defaulted the new field to
"default" and it's immutable, it doesn't seem like it would be that problematic? Or am I forgetting
something?
<<[/UNRESOLVED]>>

#### PodSecurityPolicy API

```go
type PodSecurityPolicySpec struct {
    ...
    // seccomp is the strategy that will dictate allowable and default seccomp
    // profiles for the pod.
    // +optional
    Seccomp *SeccompStrategyOptions
    ...
}

type SeccompStrategyOptions struct {
    // The default profile to set on the pod, if none is specified.
    // The default MUST be allowed by the allowedProfiles.
    // +optional
    DefaultProfile *v1.SeccompProfile

    // The set of profiles that may be set on the pod or containers.
    // If unspecified, seccomp profiles are unrestricted by this policy.
    // +optional
    AllowedProfiles *SeccompProfileSet
}

// A set of seccomp profiles. This struct should be a plural of v1.SeccompProfile.
// All values are optional, and an unspecified field excludes all profiles of
// that type from the set.
type SeccompProfileSet struct {
    // The allowed seccomp profile types.
    // +optional
    Types []SeccompProfileType
    // The allowed runtimeProfiles. A value of '*' allows all runtimeProfiles.
    // +optional
    RuntimeProfiles []string
    // The allowed localhostProfiles. Values may end in '*' to include all
    // localhostProfiles with a prefix.
    // +optional
    LocalhostProfiles []string
}
```

## Design Details

### Version Skew Strategy

Because the API is currently represented as (mutable) annotations, care must be taken for migrating
to the API fields. The cases to consider are: pod create, pod update, PSP create, PSP update.

All API skew is resolved in the API server. New Kubelets will only use the seccomp values specified
in the fields, and ignore the annotations.

#### Pod Creation

If no seccomp annotations or fields are specified, no action is necessary.

If _only_ seccomp fields are specified, add the corresponding annotations. This ensures that the
fields are enforced even if the node version trails the API version (see [Upgrade /
Downgrade](#upgrade--downgrade))

If _only_ seccomp annotations are specified, copy the values into the corresponding fields. This
ensures that existing applications continue to enforce seccomp, and prevents the kubelet from
needing to resolve annotations & fields.

If both seccomp annotations _and_ fields are specified, the values MUST match. This will be enforced
in API validation.

To raise awareness of annotation usage (in case of old automation), an additional warning annotation
will be added when a pod is created with a seccomp annotation:

```
warning.kubernetes.io/seccomp: "Seccomp set through annotations. Support will be dropped in v1.22"
```

#### Pod Update

The seccomp fields on a pod are immutable.

The behavior on annotation update is currently ill-defined: the annotation update is allowed, but
the new value will not be used until the container is restarted. There is no way to tell (from the
API) what value a container is using.

Therefore, seccomp annotation updates will be ignored. This maintains backwards API compatibility
(no tightening validation), and makes a small stabilizing change to behavior (new Kubelets will
ignore the update).

When an [Ephemeral Container](20190212-ephemeral-containers.md) is added, it will follow the same
rules for using or overriding the pod's seccomp profile. Ephemeral container's will never sync with
a seccomp annotation.

#### PodSecurityPolicy Creation

Unlike with pods, PodSecurityPolicy seccomp annotations and fields are _not_ synced.

If only seccomp annotations or fields are specified, no action is necessary. The set value is used
when applying the PodSecurityPolicy.

If both seccomp annotations _and_ fields are specified, the values MUST match. This will be enforced
in API validation.

#### PodSecurityPolicy Update

PodSecurityPolicy seccomp fields are mutable. On an update, the same rules are applied as for
creation, ignoring the old values.

If only seccomp annotations or fields are specified in the updated PSP, no action is necessary, and
the specified values are used.

If both seccomp annotations _and_ fields are specified in the updated PSP, the values MUST match.

#### PodSecurityPolicy Enforcement

The PodSecurityPolicy admission controller must continue to check the PSP object for annotations, as
well as for fields.

When setting default profiles, PSP only needs to set the field. The API machinery will handle
setting the annotation as necessary.

When enforcing allowed profiles, the PSP should check BOTH the annotations & fields. In most cases,
they should be consistent. On pod update, the seccomp annotations may differ from the fields. In
that case, the PSP enforcement should check both values as the effective value depends on the node
version running the pod.

#### PodTemplates

PodTemplates (e.g. ReplaceSets, Deployments, StatefulSets, etc.) will be ignored. The
field/annotation resolution will happen on template instantiation.

However, to raise awareness of existing controllers using the seccomp annotations that need to be
migrated, the same warning annotation will be added to the controller as for pods:

```
warning.kubernetes.io/seccomp: "Seccomp set through annotations. Support will be dropped in v1.22"
```

#### Upgrade / Downgrade

Nodes do not currently support in-place upgrades, so pods will be recreated on node upgrade and
downgrade. No special handling or consideration is needed to support this.

On the API server side, we've already taken version skew in HA clusters into account. The same
precautions make upgrade & downgrade handling a non-issue.

Since [we support](https://kubernetes.io/docs/setup/release/version-skew-policy/) up to 2 minor
releases of version skew between the master and node, annotations must continue to be supported and
backfilled for at least 2 versions passed the initial implementation. However, we can decide to
extend support farther to reduce breakage. If this feature is implemented in v1.18, I propose v1.22
as a target for removal of the old behavior.

### Test Plan

Seccomp already has [E2E tests][], but the tests are guarded by the `[Feature:Seccomp]` tag and not
run in the standard test suites.

Prior to being marked GA, the feature tag will be removed from the seccomp tests, and the tests will
be migrated to the new fields API. Tests will be tagged as `[LinuxOnly]`.

New tests will be added covering the annotation/field conflict cases described under
[Version Skew Strategy](#version-skew-strategy).

Test coverage for localhost profiles will be added, unless we decide to [keep localhost support in
alpha](#alternatives).

[E2E tests]: https://github.com/kubernetes/kubernetes/blob/5db091dde4d7de74283ca94870958acf63010c0a/test/e2e/node/security_context.go#L147

### Graduation Criteria

_This section is excluded, as it is the subject of the entire proposal._

## Implementation History

- 2019-07-17: Initial KEP

## Drawbacks

Promoting seccomp as-is to GA may be seen as "blessing" the current functionality, and make it
harder to make some of the enhancements listed under [Non-Goals](#non-goals). Since the current
behavior is unguarded, I think we already need to treat the behavior as GA (which is why it's been
so hard to change the default profile), so I do not think these changes will actually increase the
friction.

## Alternatives

The localhost feature currently depends on an alpha Kubelet flag. We could therefore label the
localhostProfile source as an alpha field, and keep it's functionality in an alpha state.

## References

- [Original seccomp proposal](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/node/seccomp.md)
