# KEP-785: Scheduler Component Config API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [x] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] KEP approvers have approved the KEP status as `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The kube-scheduler configuration API `kubescheduler.config.k8s.io` is currently
in version `v1alpha2`. We propose its graduation to `v1beta1` in order to
promote its wider use.

## Motivation

The `kubescheduler.config.k8s.io` API has been in alpha stage for several
releases. In release 1.18, we introduced `v1alpha2`, including important
changes such as:

- The removal of the old Policy API in favor of plugin configurations, that
  align with the new scheduler framework.
- The introduction of scheduling profiles, that allow a scheduler to appear
  as multiple schedulers under different configurations.
  
A configuration API allows cluster administrators to build, validate and
version their configurations in a more robust way than using command line flags.

Graduating this API to Beta is a sign of its maturity that would encourage wider
usage.

### Goals

- Introduce `kubescheduler.config.k8s.io/v1beta1` as a copy of
`kubescheduler.config.k8s.io/v1alpha2` with minimal cleanup changes.
- Use the newly created API objects to build the default configuration for kube-scheduler.

### Non-Goals

- Remove support for `kubescheduler.config.k8s.io/v1alpha2`

## Proposal

For the most part, `kubescheduler.config.k8s.io/v1beta1` will be a copy of
`kubescheduler.config.k8s.io/v1alpha2`, with the following differences:

- [ ] `.bindTimeoutSeconds` will be an argument for `VolumeBinding` plugin.
- [ ] `.profiles[*].plugins.unreserve` will be removed. We will preserve
  the extension in the internal type as long as `v1alpha2` is not removed.
  During conversion of `v1beta1`, all the plugins from `reserve` will be
  copied into `unreserve`.
- [ ] Embedded types of `RequestedToCapacityRatio` will include missing json tags
  and will be decoded with a case-sensitive decoder.

### Risks and Mitigations

The major risk is around the removal of the `unreserve` extension point.
However, this is mitigated for the following reasons:

- The interfaces for `Reserve` and `Unreserve` in the scheduler framework won't
  be merged until `v1alpha2` is removed.
- There are no in-tree Reserve or Unreserve plugins prior to 1.19.
  The `VolumeBinding` plugin is now implementing both interfaces.
  
The caveat is that out-of-tree plugins that want to work for v1beta1 need to
implement both interfaces. Otherwise scheduler startup will fail. Plugins can
choose to provide empty implementations.
This will be documented in https://kubernetes.io/docs/reference/scheduling/profiles/

### Test Plan

- [ ] Compatibility tests for `v1alpha2` that show no internal configuration
  changes.
- [ ] Test for `v1alpha2` configurations that exercise plugins that use
  `Reserve` but not `Unreserve` (and viceversa) are executed accordingly.
- [ ] Test that `.bindTimeoutSeconds` set in a `v1alpha2` propagates to the
  volume binding plugin.
- [ ] Test that args for `RequestedToCapacityRatio` with both casing pass
  in `v1alpha2`, but not in `v1beta1`.

### Graduation Criteria

#### Alpha -> Beta Graduation

- Complete features listed in [proposal][#proposal].
- Tests in [test plan](#test-plan)

## Implementation History

- 2020-05-08: KEP for beta graduation sent for review, including motivation,
  proposal, risks, test plan and graduation criteria.
