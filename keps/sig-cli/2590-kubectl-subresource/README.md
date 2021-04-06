# KEP-2590: Kubectl Subresource Support

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Notes](#notes)
- [Examples](#examples)
    - [get](#get)
    - [patch](#patch)
- [Design Details](#design-details)
  - [Subresource support](#subresource-support)
  - [Table printer](#table-printer)
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
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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

This KEP proposes supporting a new flag `--subresource` to get, patch, edit and replace kubectl
commands to fetch and update `status` and `scale` subresources.

## Motivation

Today while testing or debugging, fetching subresources (like status) of API objects via kubectl
involves using `kubectl --raw`. Patching subresources using kubectl is not possible at all and 
requires using curl directly. This method is very cumbersome and not user-friendly.

This KEP adds subresources as a first class option in kubectl to allow users to work with the API
in a generic fashion.

### Goals

- Add a new flag `--subresource=[subresource-name]` to get, patch, edit
and replace kubectl commands to allow fetching and updating `status` and `scale`
subresources for all resources (built-in and CRs) that support these subresources.
- Display pretty printed table columns for the status (uses same columns as the main resource)
and scale subresources.

### Non-Goals

- Support subresources other than `status` and `scale`.
- Allow specifying `additionalPrinterColumns` for CRDs for the status subresource.

## Proposal

kubectl commands like get, patch, edit and replace will now contain a
new flag `--subresource=[subresource-name]` which will allow fetching and updating
`status` and `scale` subresources for all API resources.

Note that the API contract against the subresource is identical to a full resource. Therefore updating
the status subresource to hold new value which could protentially be reconciled by a controller 
to a different value is *expected behavior*.

If `--subresource` flag is used for a resource that doesn't support the subresource, 
a `NotFound` error will be returned.

## Notes

The alpha stage of this KEP does not change any behavior of the `apply` command.
The support for `--subresource` in this command will be added later.

## Examples

#### get

```shell
# for built-in types
# a `get` on a status subresource will return identical information
# to that of a full resource
$ kubectl get deployment nginx-deployment --subresource=status
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
nginx-deployment   3/3     3            3           43s

$ kubectl get deployment nginx-deployment --subresource=scale
NAME               DESIREDREPLICAS   AVAILABLEREPLICAS
nginx-deployment   3                 3

# for CRDS
# a `get` on a status subresource will return identical information
# to that of a full resource
$ kubectl get crontab cron--subresource=status
NAME   SPEC          REPLICAS   AGE
cron   * * * * */5   3          4m52s

$ kubectl get vmset vmset-1 --subresource=scale
NAME   DESIREDREPLICAS   AVAILABLEREPLICAS
cron   3                 0
```

Invalid subresources:

```shell
$ kubectl get pod nginx-deployment-66b6c48dd5-dv6gl --subresource=logs
error: --subresource must be one of [status scale], not "logs"

$ kubectl get pod nginx-deployment-66b6c48dd5-dv6gl --subresource=scale
Error from server (NotFound): the server could not find the requested resource
```

#### patch

```shell
# For built-in types
# update spec.replicas through scale subresource
$ kubectl patch deployment nginx-deployment --subresource='scale' --type='merge' -p '{"spec":{"replicas":2}}'
scale.autoscaling/nginx-deployment patched

# spec.replicas is updated for the main resource
$ k get deploy nginx-deployment
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
nginx-deployment   2/2     2            2           4m

# For CRDs
$ kubectl patch crontabs cron --subresource='status' --type='merge' -p '{"status":{"replicas":2}}'
crontab.stable.example.com/cron patched

$ kubectl get crontab cron --subresource=scale
NAME   DESIREDREPLICAS   AVAILABLEREPLICAS
cron   3                 2
```

## Design Details

### Subresource support

A new field `Subresource` and method `Subresource`/`WithSubresource` is added to
the [builder], [helper] and [visitor] code to use the API object at the subresource path.

[builder]: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/cli-runtime/pkg/resource/builder.go
[helper]: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/cli-runtime/pkg/resource/helper.go
[visitor]: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/cli-runtime/pkg/resource/visitor.go

For each relevant kubectl command, a new flag `--subresource` is added. If the flag is specified,
the value is also validated to ensure that it is a valid subresource.

If the subresource does not exist for an API resource, a `NotFound` error is returned.

### Table printer

To support table view for subresources using kubectl get, table convertor support is added to
the scale and status subresoruces for built-in and CRD types.

For built-in types, `StatusStore` and `ScaleStore` are updated to implement the `TableConvertor` interface.
`StatusStore` uses the same columns as the main resource object.

The following column definitions for the `Scale` object are added to [printers.go] to support the scale subresource:
- `Available Replicas` uses the json path `.status.replicas` of autoscalingv1.Scale object
- `Desired Replicas` uses the json path of `.spec.replicas` of autoscalingv1.Scale object

For custom resources:
- the status subresoruce uses the same columns as defined for the full resource, i.e., `additionalPrinterColumns` defined in the CRD.
- the scale subresource follows the same column definitions as the built-in types, and are defined in [helpers.go].

[printers.go]: https://github.com/kubernetes/kubernetes/blob/master/pkg/printers/internalversion/printers.go#L88
[helpers.go]: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/helpers.go

### Test Plan

- Unit tests, integration and e2e tests will be added.

### Graduation Criteria

#### Alpha -> Beta Graduation

- [ ] Collect user feedback on adding support of `--subresource` for `apply`

#### Beta -> GA Graduation

- [ ] User feedback gathered for atleast 1 cycle

### Upgrade / Downgrade Strategy

This functionality is contained entirely within kubectl and shares its strategy.
No configuration changes are required.

### Version Skew Strategy

Not applicable. There is nothing required of the API Server, so there
can be no version skew.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
- [x] Other
    - Describe the mechanism: A new flag for kubectl commands.
      For the alpha stage, description will be added to expicitly call
      out this flag as an alpha feature.
    - Will enabling / disabling the feature require downtime of the control
      plane? No, disabling the feature would be a client behaviour.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
      No, disabling the feature would be a client behaviour.

###### Does enabling the feature change any default behavior?

While the feature now updates kubectl's behavior to allow updating subresources,
it is gated by the `--subresource` flag so it does not change kubectl's default
behavior.

Subresources can also be updated using curl today so this feature only
provides a consistent way to use the API via kubectl, but does not allow additional
changes to the API that are not possible today.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, rolling back to a previous version of `kubectl` will remove support
for the `--subresource` flag and the ability to update subresources via kubectl.

However, this does not "lock" us to any changes to the subresources that were made
when the feature was enabled i.e. it does not remove the ability to update subresources
for existing API resources completely. If a subresource of an exisiting API resource needs
to be updated, this can be done via curl.

###### What happens if we reenable the feature if it was previously rolled back?

The `--subresource` flag can be used and subresources can be updated via kubectl again.

###### Are there any tests for feature enablement/disablement?

No, because it cannot be disabled or enabled in a single release.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout fail? Can it impact already running workloads?

The feature is encapsulated entirely within the kubectl binary, so rollout is
an atomic client binary update. Subresources can always be updated via curl,
so there are no version dependencies.

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This feature is completely with in the client. The upgrades and rollback of cluster will not be affected by this change.  
The update and downgrade of the kubectl version will only limit the availability of the `--subresource` flag and will not
change any API behavior.

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

- API call type (e.g. PATCH pods):
  GET, PATCH, PUT `/<subresource>`

- Estimated throughput:
  Negligible, because it's human initiated. At maximum, each command would involve
  two calls: 1 read and 1 mutate.

- Originating component(s):
  kubectl

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

`kubectl` is not resilient to API server unavailability.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

2021-03-01: Initial [POC PR] created  
2021-04-06: KEP proposed

[POC PR]: https://github.com/kubernetes/kubernetes/pull/99556

## Alternatives

Alternatives would be to use curl commands directly to update subresources.


