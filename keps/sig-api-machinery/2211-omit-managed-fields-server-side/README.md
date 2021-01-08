# KEP-2211: Omit ManagedFields on the Server-side

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Server-side changes](#server-side-changes)
  - [Client-side changes](#client-side-changes)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Server-side](#server-side)
  - [Client-side](#client-side)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

ManagedFields is used to keep track of who has changed a field of an object, and Server-side Apply relies on such information.
While the managedFields is beneficial to resolving conflicts on updates, it is a bit verbose to many users and controllers
because this field could be quite large and some users and controllers are not interested in such information. 
In addition, the bandwidth usage and size of internal caches could be significantly affected due to the additional managedFields.

This KEP is proposing omitting the managedFields on the server-side when clients request objects to make the response less verbose
and save the amount of data that needs to be transferred.

## Motivation

Many users have complained in [kubernetes#90066][90066] that they got impacted by the addition of managedFields
(and the issue itself has received more than 100 thumb-up reactions), and we want to improve the user experience.

[90066]: https://github.com/kubernetes/kubernetes/issues/90066

At the same time, it is not ideal to only drop the managedFields on the client-side, because that requires the same
bandwidth usage compared to simply returning the original object, and every client
(such as kubectl or CLI tools written by users using client-go) needs to drop the managedFields by themselves.

With this KEP, the API server could return objects without managedFields on demand, so that least client-side changes is needed
to make the response less verbose. 

### Goals

* Omit managedFields from responses, when requested by clients.

### Non-Goals

* Turn off managedFields. Server-side Apply relies on managedFields, we could not simply disable it.
* Clear managedFields only on the client-side.

## Proposal

### Server-side changes

Implement a new content type modifier, say `mf`, which leverages the existing content type negotiation mechanism,
so that clients could ask the server to omit managedFields with an `Accept` header like `Accept: application/json;mf=none`.

The new modifier `mf` is the abbreviation for `Managed Fields`. It represents whether clients want to omit or request a 
specific version of managed fields of objects. The available values are `none` or a version name, such as `v1` or `v2`.
Only `none` is supported at the moment.

### Client-side changes

Implement a flag in `kubectl` to let users decide whether to omit the managedFields.

### Risks and Mitigations

Risk:
* The API server might not be up-to-date and cannot recognize the new content type modifier.

Mitigations:
1. The new content type modifier is ignored by the old server, so it has no effect and will not break any existing things.
2. Make `kubectl get` to drop managedFields if user wants to omit managedFields, but the API server doesn't realize that.

## Design Details

### Server-side

Add field `ManagedFields string` to [`MediaTypeOptions`](https://github.com/kubernetes/kubernetes/blob/9fb1aa92f2206c04ce77c0f9767e235f94d5f7ba/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/negotiation/negotiate.go#L145):
```go
package negotiation

// MediaTypeOptions describes information for a given media type that may alter
// the server response
type MediaTypeOptions struct {
...
	// managedFields represents whether clients want to omit or request specific version of managed fields of objects.
	// The available values are "none" or a specific version, such as "v1" or "v2".
	ManagedFields string
}
```

When we [`doTransformObject`](https://github.com/kubernetes/kubernetes/blob/9fb1aa92f2206c04ce77c0f9767e235f94d5f7ba/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/response.go#L58), 
we check if the value of `MediaTypeOptions.ManagedFields` is `none`, if yes, we omit the managedFields from the object:
```go
package handlers

func doTransformObject(ctx context.Context, obj runtime.Object, opts interface{}, mediaType negotiation.MediaTypeOptions, scope *RequestScope, req *http.Request) (runtime.Object, error) {
	if _, ok := obj.(*metav1.Status); ok {
		return obj, nil
	}
	if err := setObjectSelfLink(ctx, obj, req, scope.Namer); err != nil {
		return nil, err
	}

	if mediaType.ManagedFields == "none" {
		omitManagedFields(obj)
	}
	...
}

func omitManagedFields(obj runtime.Object) {
	if _, err := meta.Accessor(obj); err == nil {
		a, _ := meta.Accessor(obj)
		a.SetManagedFields(nil)
	} else if meta.IsListType(obj) {
		_ = meta.EachListItem(obj, func(item runtime.Object) error {
			a, err := meta.Accessor(item)
			if err != nil {
				// not implement `metav1.Object`, ignore
				return nil
			}
			a.SetManagedFields(nil)
			return nil
		})
	}
}
```
### Client-side

Implement a flag `--show-managed-fields` for `kubectl get`. Its value could be empty or a version name(in case 
users want to request a specific version of managed fields), such as `v1`, `v2` etc, and the value defaults to `none`
which means the managedFields are omitted.

In summary, the behavior of `kubectl get` is shown as below:

| Command | Display ManagedFields |
| ------- | --------------------- |
| kubectl get foo | No |
| kubectl get --show-managed-fields foo | Yes(Display all managedFields) |
| kubectl get --show-managed-fields=v1 foo | Yes(Only display "v1" managedFields) |


### Test Plan

* Unit tests covering all corner cases of logic of newly introduced content type modifier.
* Regular e2e tests are passing.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include 
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

TBD

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

TBD

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

TBD

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

TBD

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

TBD
