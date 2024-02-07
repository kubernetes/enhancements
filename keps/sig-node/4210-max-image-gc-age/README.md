# KEP-4210: ImageMaximumGCAge in Kubelet

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
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

Add an option ImageMaximumGCAge, which allows an admin to specify a time after which unused images will be garbage collected
by the Kubelet, regardless of disk usage, as well as an associated feature gate to toggle the behavior.

## Motivation

Currently, all image garbage collection the Kubelet is triggered by disk usage going over a threshold (ImageGCLowThresholdPercent).
However, there are cases that additional conditions could be considered useful. One such condition is maximum age of an image.
If an image is unused for a long time (the exact amount of time will be decided, but on the order of weeks is what comes to mind),
then it is not likely to be used again.

One such condition that can be imagined are clusters with automatic upgrades. If a cluster goes through an upgrade process,
it will likely have images cached from the old release (old kube-apiserver/etcd/etc). While these images would eventually get removed
through the disk usage condition, they will needlessly occupy disk space before that.

### Goals

- Introduce an option to the Kubelet ImageMaximumGCAge and a feature gate ImageMaximumGCAge

### Non-Goals

- Introduce other conditions for image garbage collection to trigger.
    - A WG was put together in SIG-Node to collect other GC use cases. While some other cases were identified, all seemed to be covered by this (see Alternatives)


## Proposal

Kubelet has three different configuration fields for image garbage collection:
- ImageMinimumGCAge: the youngest an image can be to be qualified for garbage collection
- ImageGCLowThresholdPercent: The lowest disk usage will be before garbage collection begins
- ImageGCHighThresholdPercent: The highest disk usage will be before garbage collection runs each GC period.

Between each of these options, there is a common thread: image garbage collection only triggers when disk usage has reached a certain threshold.
In other words, there are no alternative conditions the Kubelet will begin triggering collection. Luckily, this condition is the most important:
the primary goal of garbage collection is to ensure images don't clutter the disk too much and cause it to fill up needlessly.
However, having the Kubelet be purely reactive means that images will clutter the disk and cause it to fill up. While there aren't reported cases
where this causes issues, it is inefficient with disk space and can cause the Kubelet to scramble to save disk space when the threshold is met.

An additional approach is to define a way for an admin to request images are cleaned up after they're unused for a certain period of time.
This would reduce the frequency of the disk usage hitting the level, and provide an admin more flexibility in how garbage collection is defined.

The proposal of this KEP is to add an option to the KubeletConfiguration object that looks like:
```
	// ImageMaximumGCAge is the maximum age an image can be unused before it is garbage collected.
    // The default of this field is 0, which disables it.
    // +optional
	ImageMaximumGCAge metav1.Duration
```

To begin, this option will be set to 0, which will be interpreted as "disabled". In the future, a more reasonable default may be chosen.

This option will only be adhered to if the feature gate ImageMaximumGCAge is configured for the Kubelet.

### User Stories (Optional)

#### Story 1

- As a cluster admin, I would like my unused images to be garbage collected in a timely manner, and not occupy disk space forever.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

- If set incorrectly the ImageMaximumGCAge option could cause unneeded image pulls. For instance, if a Cron job ran
once a week, but the ImageMaximumGCAge was set to less than a week, that image would get pulled every week, causing needless
traffic from the registry
    - Proper documentation on this is the best way to mitigate this risk.
    - Defining a good default for this value will be similarly tricky.
- Reliability of image age
    - Good testing will mitigate/fix any errors
- New, undiscovered races
    - If the max image gc age is set very low, will the kubelet race with itself and remove the image right after pulling it?
    - May need to define a minimum maximum gc age to prevent races like this.
- Runtime misbehavior
    - It's possible the runtime won't GC the image and kubelet will begin thrashing on the image.
    - Runtime maintainers should ensure to avoid this situation
    

## Design Details

Add an option to the Kubelet configuration:
```
	// ImageMaximumGCAge is the maximum age an image can be unused before it is garbage collected.
    // The default of this field is 0, which disables it.
    // +optional
	ImageMaximumGCAge metav1.Duration
```

This option will be wired down to the Kubelet's [image manager](https://github.com/kubernetes/kubernetes/blob/d5690f12b69a/pkg/kubelet/images/image_gc_manager.go),
similarly to the other garbage collection fields.

The Kubelet's image manager already keeps track of the last time an image was used through the `lastUsed` field in the
[imageRecord](https://github.com/kubernetes/kubernetes/blob/d5690f12b69a/pkg/kubelet/images/image_gc_manager.go#L153) structure.
So a comparison can be made in the realImageGCManager's function
[GarbageCollect](https://github.com/kubernetes/kubernetes/blob/d5690f12b69a/pkg/kubelet/images/image_gc_manager.go#L288) to garbage collect
the images that are older than the specified image age.

Since the Kubelet does not own images, and can only request images be cleaned up, this cleaning should be considered "best effort".

Further, since Kubelet's GC runs periodically every [5 minutes](https://github.com/kubernetes/kubernetes/blob/d5690f12b69a/pkg/kubelet/kubelet.go#L194)
the ImageMaximumGCAge may not be exactly precise. An image could be GC'd up to 5 minutes after it has aged out.

Finally, Kubelet restarts are a point that needs to be figured out. The easiest way to handle it would be waiting the full ImageGCMaximumAge for an image to be qualified for GC,
but that would essentially disable the feature if the Kubelet restarts more frequently than ImageGCMaximumAge.

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

- `pkg/kubelet/images`: `2023-09-14` - `84.2`

Additional tests will be added to pkg/kubelet/images to unit test the new field and verify it works along with the other GC options.

##### e2e tests

- `test/e2e_node/garbage_collector_test.go`

Additional tests will be added to this file to cover the garbage collection e2e.

### Graduation Criteria


#### Alpha

- Configuration field added to the Kubelet (disabled by default)
- Feature supported by Kubelet Image Manager
- Unit tests
- Add a metric `kubelet_image_garbage_collected_total` which tracks the number of images the kubelet is GC'ing through any mechanism.

#### Beta

- Add e2e tests
- Document `kubelet_image_garbage_collected_total` (a step missed in alpha)
- Add "reason" field to `kubelet_image_garbage_collected_total` to allow distinguishing between GC reasons (space based or time based).

#### GA

- Addition of conformance tests
- Some examples of real-world usage
- Allowing time for feedback

### Upgrade / Downgrade Strategy

This option is purely contained within the Kubelet, so the only concern is the flag is added to the configuration of the newer
Kubelet and then downgraded.

There's nothing the Kubernetes community can do to prevent this, and admins should ensure their configuration fields will function with
the processes they run.

### Version Skew Strategy

Version skew is not a worry assuming the internal Kubelet changes are synchronized with the configuration changes.

## Production Readiness Review Questionnaire


### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ImageGCMaximumAge
  - Components depending on the feature gate: kubelet
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, given a restart of the Kubelet.

###### What happens if we reenable the feature if it was previously rolled back?

- Nothing unexpected.

###### Are there any tests for feature enablement/disablement?

There will be a test to verify when the Kubelet configuration option is disabled that the image isn't GC'd early.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

- Invalid configuration configured.
- Even in the case where the ImageMaximumGCAge is set to 0, the Kubelet will only GC images when their corresponding containers are
removed, so no running workloads can be affected.

###### What specific metrics should inform a rollback?

- `kubelet_image_garbage_collected_total` metric drastically (100x) increasing, with the "reason" field being "age",
indicating thrashing of the GC manager and images being pulled.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

They will be, there should be no side effects.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- Verify the Kubelet Configuration with the Kubelet's configz endpoint
- Monitor the `kubelet_image_garbage_collected_total`, and expect some images are removed for reason "age"

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - `kubelet_image_garbage_collected_total` metric increases when an image ages out.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- The eventual default value should increase the average `kubelet_image_garbage_collected_total` by no more than 10x

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kubelet_image_garbage_collected_total`
  - Components exposing the metric: Kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

- A metric for each different GC trigger (disk usage vs time based).

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

Just Kubelet

### Scalability

###### Will enabling / using this feature result in any new API calls?

- Kubelet will call `RemoveImage` to the CRI implementation when an image should be garbage collected,
  which could happen more frequently/faster.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

KubeletConfiguration will gain an additional int64

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

- Potentially, depending on the age chosen, there could be more CPU used to do the image removal.
  - The frequency of the image removal will be a tradeoff for existing disk space

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

- Not likely, it's intended to prevent resource exhaustion of disk

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

- N/A

###### What are other known failure modes?

- The Kubelet could thrash with itself in a image pull/remove cycle if the value is set too low.

###### What steps should be taken if SLOs are not being met to determine the problem?

- Set a minimum value this field could be.

## Implementation History


2023-09-18: KEP opened, targeted at Alpha
2024-01-22: KEP updated to Beta

## Drawbacks

- It could be considered unnecessary, as the disk usage based garbage collection already covers this use case, albeit slower.

## Alternatives

- Add a Kubelet garbage collection plugin system
    - Too complicated, probably won't be needed.
    - The Image GC WG off of SIG-Node brainstormed use cases:
        - Additional conditions for GC:
            - Removing "older" tags of the same image.
            - do not keep images policy.
            - image GC based on pod priority
        - Only the last item is not covered here, and it was deemed not useful enough to warrant a generic solution.
- Delegate responsibility down to CRI
    - Would cause code duplication between CRI implementaions, out of scope for this.
- The image GC WG worked to identify other conditions for GC:
    - Both of these can be satisfied by this KEP, so we're not pursuing a more generic GC Plugin mechanism.

## Infrastructure Needed (Optional)

N/A
