# KEP-1623: Standardize Conditions.

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Noteworthy choices](#noteworthy-choices)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

- [ ] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

While many Kuberentes APIs have `.status.conditions`, the schema of `condition` varies a lot between them.
There is very little commonality at the level of serialization, proto-encoding, and required vs optional.
Conditions are central enough to the API to make a common golang type with a fixed schema.
The schema can be a strong recommendation to all API authors.

## Motivation

Allow general consumers to expect a common schema for `.status.conditions` and share golang logic for common Get, Set, Is for `.status.conditions`.
The pattern is well-established and we have a good sense of the schema we now want.

### Goals

 1. For all new APIs, have a common type for `.status.conditions`.
 2. Provide common utility methods for `HasCondition`, `IsConditionTrue`, `SetCondition`, etc.
 3. Provide recommended defaulting functions that set required fields and can be embedded into conversion/default functions.

### Non-Goals

 1. Update all existing APIs to make use of the new condition type.

## Proposal

Introduce a type into k8s.io/apimachinery/pkg/apis/meta/v1 for `Condition` that looks like 
```go
type Condition struct {
	// Type of condition in CamelCase or in foo.example.com/CamelCase.
	// Many .condition.type values are consistent across resources like Available, but because arbitrary conditions can be
	// useful (see .node.status.conditions), the ability to deconflict is important.
	// +required
	Type string `json:"type" protobuf:"bytes,1,opt,name=type"`
	// Status of the condition, one of True, False, Unknown.
	// +required
	Status ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status"`
	// If set, this represents the .metadata.generation that the condition was set based upon.
	// For instance, if .metadata.generation is currently 12, but the .status.condition[x].observedGeneration is 9, the condition is out of date
	// with respect to the current state of the instance.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,3,opt,name=observedGeneration"`
	// Last time the condition transitioned from one status to another.
	// This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.
	// +required
	LastTransitionTime metav1.Time `json:"lastTransitionTime" protobuf:"bytes,4,opt,name=lastTransitionTime"`
	// The reason for the condition's last transition in CamelCase.
	// The specific API may choose whether or not this field is considered a guaranteed API.
	// This field may not be empty.
	// +required
	Reason string `json:"reason" protobuf:"bytes,5,opt,name=reason"`
	// A human readable message indicating details about the transition.
	// This field may be empty.
	// +required
	Message string `json:"message" protobuf:"bytes,6,opt,name=message"`
}
```

This is not strictly compatible with any of our existing conditions because of either proto ordinals,
required vs optional, or omitEmpty or not.
However, it encapsulates the best of what we've learned and will allow new APIs to have a unified type.

### Noteworthy choices
 1. `lastTransitionTime` is required.
    Some current implementations allow this to be missing, but this makes it difficult for consumers.
    By requiring it, the actor setting the field can set it to the best possible value instead of having clients try to guess.
 2. `reason` is required and must not be empty.
    The actor setting the value should always describe why the condition is the way it is, even if that value is "unknown unknowns".
    No other actor has the information to make a better choice.
 3. `lastHeartbeatTime` is removed.
    This field caused excessive write loads as we scaled.
    If an API needs this concept, it should codify it separately and possibly using a different resource.

### Graduation Criteria

Because meta/v1 APIs are necessarily v1, this would go direct to GA.
Using a meta/v1beta1 isn't a meaningful distinction since this type is embedded into other types which own their own versions.

### Upgrade / Downgrade Strategy

This KEP isn't proposing that existing types be changed.
This means that individual upgrade/downgrade situations will be handled discretely.
By providing recommended defaulting functions, individual APIs will be able to more easily transition to the new condition type.

### Version Skew Strategy

Standard defaulting and conversion will apply.
APIs which have extra values for this type may have to go through an intermediate version that drops them or accept
that certain optional fields of their conditions will be dropped.
Depending on the individual APIs and when their extra fields are deprecated, this could be acceptable choice.

## Implementation History

## Drawbacks

 1. There may be some one-time pain when new versions are created for APIs that wish to consume this common schema.
    Switching is not strictly required, but it is encouraged.

## Alternatives

 1. We could recommend a schema and not provide one.  This doesn't seem very nice to consumers.

