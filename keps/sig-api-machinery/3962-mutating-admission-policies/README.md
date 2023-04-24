# KEP-3962: Mutating Admission Policies

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [Bindings](#bindings)
    - [Unsetting values](#unsetting-values)
    - [Old object state](#old-object-state)
    - [Complex mutations](#complex-mutations)
    - [Multiple mutations](#multiple-mutations)
      - [Order](#order)
      - [Safety](#safety)
      - [Reinvocation](#reinvocation)
  - [User Stories](#user-stories)
    - [Use case: Set a label](#use-case-set-a-label)
    - [Use case: Set a field](#use-case-set-a-field)
    - [Use case: Sidecar injection](#use-case-sidecar-injection)
    - [Use case: Clear an annotation](#use-case-clear-an-annotation)
    - [Use case: If an annotation is set, set a field instead](#use-case-if-an-annotation-is-set-set-a-field-instead)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Object type names](#object-type-names)
  - [SSA Merge algorithm reuse](#ssa-merge-algorithm-reuse)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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

This enhancement adds a `MutatingAdmissionPolicy` API to allow mutating
admission control rules to be declared using CEL expressions. This continues the
work started by the [`ValidatingAdmissionPolicy`
API](/keps/sig-api-machinery/3488-cel-admission-control/README.md).

Mutations can be declared in CEL by combining CEL's object instantiation
and Server Side Apply's merge algorithms.

## Motivation

A large proportion of mutations perform relatively simple changes such as setting
a label or annotation, or adding a sidecar container to a pod. Such mutations
can be expressed conveniently in CEL, eliminating the need for the developmental
and operational complexity of a webhook.

Offering CEL based mutation also has other fundamental advantages over webhooks.
CEL mutations can be declared in a way that allows the kube-apiserver to
introspect the mutation and extract useful information about which fields the
mutation reads and writes. This information can be useful when ordering
mutations.

### Goals

- Provide an alternative to mutating webhooks for the vast majority of mutating
  admission use cases.
- Provide the in-tree extensions needed to build policy frameworks for
  Kubernetes, again without requiring webhooks for the vast majority of use
  cases.
- Provide an out-of-tree implementation of this enhancement (using a webhook)
  that is supported by the Kubernetes org to provide this enhancement
  functionality to Kubernetes versions where this enhancement is not available.
- Provide core functionality as a library so that use cases like GitOps,
  CI/CD pipelines, and auditing can run the same CEL validation checks
  that the API server does.

### Non-Goals

- Build a comprehensive in-tree policy framework. We believe the ecosystem is
  best equipped to explore and develop policy frameworks.
- Full feature parity with mutating admission webhooks. For example, this
  enhancement is not expected to ever support making requests to external
  systems.
- Replace the admission controllers compiled into the API server. 
- Static or on-initialization specification of admission config. This is a
  needed feature but should be solved in a general way and not in this KEP
  (xref: https://github.com/kubernetes/enhancements/issues/1872).

## Proposal

Introduce `MutatingAdmissionPolicy`, e.g.

```yaml
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: MutatingAdmissionPolicy
metadata:
  name: "sidecar-policy.example.com"
spec:
  paramKind:
    group: mutations.example.com
    kind: Sidecar
    version: v1
  matchConstraints:
    resourceRules:
    - apiGroups:   ["apps"]
      apiVersions: ["v1"]
      operations:  ["CREATE"]
      resources:   ["pods"]
  mutations:
    - name: inject-sidecar
      apply: >
        Object{
          spec: Object.spec{
            initContainers: [
              Object.spec.initContainers{
                name: params.name,
                image: params.image,
                args: params.args,
                restartPolicy: params.restartPolicy
                // ... other container fields injected here ...
              }
            ] + oldObject.spec.initContainers
          }
        }
```

The `apply` field contains a CEL expression that evalutes to a partially
populated object representing an Server Side Apply "apply configuration". The
apply configuration is then merged into the request object.

By using Server Side Apply merge algorithms, schema declarations like
`x-kubernetes-list-type: map`, that control how a merge is performed, will be
respected.

However, unlike with server side apply, these mutations will not have a field
manager. This has important implications in how the merge is performed that will
be discussed in more detail in the below "Unsetting values" section.

In this example, note that:

- `Object{}`, `Object.spec{}` and similar are CEL object instantiations, and are
  used to create a subset of the fields of a `Pod`.
- `oldObject` refers to the state of the object before the mutation is applied.

To use this `MutatingAdmissionPolicy` we first must create a policy binding
and `Sidecar` parameter resource:

```yaml
# Policy Binding
apiVersion: admissionregistration.k8s.io/v1alpha1
kind: MutatingAdmissionPolicyBinding
metadata:
  name: "sidecar-binding-test.example.com"
spec:
  policyName: "sidecar-policy.example.com"
  paramRef:
   name: "meshproxy-test.example.com"
   namespace: "default"
```

```yaml
# Sidecar parameter resource
apiVersion: mutations.example.com
kind: Sidecar
metadata: meshproxy-test.example.com
  name: 
spec:
  name: mesh-proxy
  image: mesh/proxy:v1.0.0
  args: ["proxy", "sidecar"]
  restartPolicy: Always
```

Next, a pod may be created:

```yaml
kind: Pod
spec:
  initContainers:
  - name: myapp-initializer
    image: example/initializer:v1.0.0
  containers:
  - name: myapp
    image: example/myapp:v1.0.0
```

Since the API request to create the pod matches the `MutatingAdmissionPolicy`
the pod will be mutated, resulting in:

```yaml
kind: Pod
spec:
  initContainers:
  - name: mesh-proxy
    image: mesh/proxy:v1.0.0
    args: ["proxy", "sidecar"]
    restartPolicy: Always
  - name: myapp-initializer
    image: example/initializer:v1.0.0
  containers:
  - name: myapp
    image: example/myapp:v1.0.0
```

#### Bindings

Bindings will be almost the same as `ValidatingAdmissionPolicyBinding`, but with
the following difference:

- No `validationActions` field (unless anyone can think of any useful way to offer a dry-run type option)

#### Unsetting values

Since there is no field manager used for the merge, the server side apply merge
algorithm will only add and replace values. This is because server side apply is
designed to only unset values that were previously owned by the field manager
but excluded from the apply configuration.

To work around this limitation, we will take advantage of CEL's optional type
feature to make it possible to express that a value should be unset. For example:

```cel
Object{
  spec: Object.spec{
    ?fieldToRemove: optional.none() # plz remove
  }
}
```

We will track which fields are unset in this way and remove them after the
server side apply merge algorithm is run.

This solves the vast majority of value removal needs. Specifically:

| Schema type | Merge type | Example of how to unset a value                                      |
| ----------- | -----------| -------------------------------------------------------------------- |
| struct      | atomic     | See below                                                            |
| struct      | granular   | `?fieldToRemove: optional.none()`                                    |
| map         | atomic     | `mapField: oldObject.spec.mapField.filter(k, k != "keyToRemove")`    |
| map         | granular   | `mapField: {?"keyToRemove": optional.none()}`                        |
| list        | atomic     | `listField: oldObject.spec.listField.filter(e, e != "itemToRemove")` |
| list        | set        | `setField: oldObject.spec.setField.filter(e, e != "itemToRemove")`   |
| list        | map        | See below                                                            |

<<[UNRESOLVED jpbetz ]>>
Decide on how to handle struct, type=atomic
<<[/UNRESOLVED]>>

struct, type=atomic Options:
  - Recreate the struct, e.g.: `Object.spec.structField: Object.spec.structField{field1: oldObject.spec.structField.field1}`
  - TODO: The entire OpenAPIv3 stanza of a CRD is an atomic struct, can we support mutations of it?

<<[UNRESOLVED jpbetz ]>>
Decide on how to handle list, type=map
<<[/UNRESOLVED]>>

list, type=map Options:
  - Add a `objects.remove()` that makes it possible to remove "list,type=map"
    entries.
  - Use a synthetic "_remove: true" marker field to all "list,type=map" object
    schemas so that a user could do `[Object.spec.listMapField{keyField: "key",
    _remove: true}]`.
  - Offer way to declare which fields are "owned" by the mutation, and then remove
    any fields that are owned but where no replacement value is set.

#### Old object state

Sometimes the current state of the object will be needed. This is available via
the `oldObject` variable. For example, to update all containers in a pod to use
the "Always" imagePullPolicy:

```cel
Object{
  spec: Object.spec{
    containers: oldObject.spec.containers.map(c,
      Object.spec.containers.item{
          name: c.name,
          imagePullPolicy: "Always"
      }
    )
  }
}
```

#### Complex mutations

<<[UNRESOLVED jpbetz ]>>
Expose merge functions directly? I'm concerned with the API complexity of doing this, but
it does solve for some otherwise difficult cases.
<<[/UNRESOLVED]>>

On rare occasions, it may be useful to perform apply directly on part of an
object. For example, imagine that the field "widgets" is a list of objects, but
the field is not a listType=map, and so there is no way to merge in an added
field to each widget using server side apply directly like was done with the
above imagePullPolicy example.

A workaround is to recreate all the widgets list, but use apply() on each widget to merge
in a single field change:

```cel
Object{
    spec: Object.spec{
        containers: oldObject.spec.widgets.map(oldWidget,
            objects.apply(oldWidget, Object.spec.widgets.item{
                part: "xyz"
            })
        )
    }
}
```

Note that such fields are very rare in the Kubernetes API since they don't work
well with server side apply. But this may come in handy with CRDs when the CRD
author fails to use `x-kubernetes-list-type: map`.

This can also be avoided if we allow mutations to be scoped, e.g. `scope:
spec.widgets[*]`.

#### Multiple mutations

##### Order

<<[UNRESOLVED jpbetz ]>>
Should we offer a separate `scope` field for mutations or just compute scopes
from apply configurations applied to the root?
<<[/UNRESOLVED]>>

We will order mutations primarily by scope. Mutations scoped nearer to the root
are ordered before deeper leveled scopes. 

How will scopes be determined?

- Option 1: explicitly as an API field (e.g. `spec.containers[*]`
- Option 2: computed by inspecting the CEL expression to identify which parts of
  the apply configuration are just path information. E.g. `Object{ spec: Object.spec{ containers: oldObject.spec.containers.filter(...) }}` is clearly
  scoped to `spec.containers`
- Option 3: Offer Option 1 for convenience but also further narrow the actual
  scopes used for computing order using Option 2.

Once scopes are known, they could be written to the status to make it easier for
cluster administrators to understand the total order of mutations.

Alternative: Do what mutating webhooks do: order mutations using the
lexographical ordering of the resource names. Just like with mutating webhooks,
reinvocation will be required.

Alternative: Order by constructing a DAG of which fields each mutation reads and
writes. CEL expressions can be statically analyzed to build a list of the schema
fields that are read/written. Potential cycles could be detected and a warning
written to status.. This is significantly more coplex than the other alternatives.
For this alternative to offer any advantage, we would need to be able to use
the data to ban cycles from being introduced so that the DAG can be used
to determine a total order of mutations.

##### Safety

<<[UNRESOLVED jpbetz ]>>
Define the API for this.
<<[/UNRESOLVED]>>

To ensure mutations are not "broken" by other mutations (overwritten, undone, or
otherwise invalidated), we will validate mutations after they are applied to
ensure that they applied in a reasonable way. For basic single field mutations a
validation can be automatic (e.g. run the mutation again and making sure nothing
changes). Some mutations will have more complex validation needs. For example, a
sidecar injector might want to allow for the fields of the injected sidecar
container to be modified (`imagePullPolicy` is a prime example), but still
verify that the sidecar container exists in the list of containers and has some
core set of fields set to the expected values.

We intend make it convenient in the API for mutations to declare validation
requirements. A set of validation options like this might be sufficient:

- `Exact`: Validate that the mutation is fully applied (replaying the
  mutation doesn't change anything)
- `Overwritable`: No validation (allow other mutators to overwrite)
- `conditional` (probably not an enum value though): Validate that some CEL
  expressions results in true (can be used to validate that some subset of
  fields, or that an int increase remains monotomic, ...)

The safest way to ensure that admission authors pick a reasonable validation option
is to use a required field without a default.

By putting the validation of each mutation in the MutatingAdmissionResource we
can enable both the validating and mutating changes atomically in an API server.

##### Reinvocation

Like mutating webhooks, we will reinvoke mutations if there is any possiblity
that two mutations are interacting with each other. Any mutation that could
possibly have read fields written by another mutator, or that might overwrite
fields written by another mutator will trigger reinvocation. 

We can be more intelligent about reinvocation that mutating webhooks since we
can compute scopes of both fields read and written for CEL mutations, which is
not something that is possible with mutating webhooks.

Like mutating webhooks, we will set a limit on the number of reinvocations that
can occur. Unlike mutating webhooks, CEL invocation is fast so we more than two
invocation passes might be possible. However, since CEL mutation also has more
information about what fields each mutator reads and writes, we also do not
expect reinvocation to be as often needed, and so will likely use the same
reinvocation limit of 2 that is used by webhooks, but may also reinvoke
for purposes of validating mutations (see above "Safety" section).

We may also be able to detect that mutations have not "converged" (settled
on a stable output) within the two reinvocations and report this information
into the status of all mutators that are "fighting with each other".

### User Stories

#### Use case: Set a label

```cel
Object.spec{
  Object.metadata{
    labels:
      "label-to-set": "label-value"
  }
}
```

#### Use case: Set a field

```cel
Object{
    spec: Object.spec{
        containers: oldObject.spec.containers.map(c,
            Object.spec.containers.item{
                name: c.name,
                imagePullPolicy: "Always"
            })
        )
        // ... same for initContainers and ephemeralContainers ...
    }
}
```

#### Use case: Sidecar injection

```cel
Object{
  spec: Object.spec{
    initContainers: [
      Object.spec.initContainers{
        name: params.name,
        image: params.image,
        args: params.args,
        restartPolicy: params.restartPolicy
        // ... other container fields injected here ...
      }
    ] + oldObject.spec.initContainers
  }
}
```

#### Use case: Clear an annotation

```cel
Object{
  metadata: Object.metadata{
    annotations:
      ?"annotation-to-unset": optional.none()
  }
}
```

#### Use case: If an annotation is set, set a field instead

```cel
Object{
  metadata: Object.metadata{
    annotations:
      ?"some-annotation": optional.none()
  }
  spec: Object.spec{
    someField: oldObject.annotations["some-annotation"]
  }
}
```

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

Risk: Enabling CEL object instantiation enables users to allocate memory in
a more directly way than previously available.

- Mitigation/Justification: List and map literals were already possible, so this
  doesn't fundamentally change the memory allocation situation. We could further
  mitigate by statically estimating the "memory cost" of CEL expressions that
  include any form of data literal (list, map, object or scalar).

Risk: Ordering muations based on scope of the change instead of lexographical
order of resource name make it more difficult to reason about mutation order.

- Mitigation/Justification: The order will still be deterministic, which is the
  main benefit lexographical order provided.

## Design Details

### Object type names

As part of this enhancement, we are enabling CEL object instantiation, which
we have left disabled in previous CEL features.

When enabling CEL object instantiation we need to decide:

- How object type names will represented in CEL. This KEP shows a
  "Object.spec.container" naming system. Is this what we will use? Or will be
  use actual schema type names, e.g. "v1.Pod.spec.container"?
- Will validating admission features also gain the ability to instantiate CEL
  types? This has memory consumption implications.

### SSA Merge algorithm reuse

Reusing Server Side Apply merge algorithms is complicated by presence of
numerous different representations of schema types in Kubernetes (Structural
schemas, multiple OpenAPI schema representations, SMD schemas, ...). For an
initial alpha, we may simply perform the needed conversions, but longer term
memory consumption and runtime performance may demand that we minimize the
conversions needed.

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
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

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
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

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

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

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
