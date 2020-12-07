# KEP-2155: Apply for client-go's typed client

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Poor adoption](#poor-adoption)
- [Design Details](#design-details)
  - [Apply functions](#apply-functions)
  - [Generated apply configuration types](#generated-apply-configuration-types)
    - [Alternative 1: Genreated structs where all fields are pointers](#alternative-1-genreated-structs-where-all-fields-are-pointers)
    - [Alternative 2: Generated &quot;builders&quot;](#alternative-2-generated-builders)
    - [Comparison of alternatives](#comparison-of-alternatives)
    - [DeepCopy support](#deepcopy-support)
    - [Code Generator Changes](#code-generator-changes)
      - [Addition of applyconfiguration-gen](#addition-of-applyconfiguration-gen)
      - [client-gen changes](#client-gen-changes)
  - [Interoperability with structured and unstructured types](#interoperability-with-structured-and-unstructured-types)
  - [Test Plan](#test-plan)
    - [Fuzz-based round-trip testing](#fuzz-based-round-trip-testing)
  - [Integration testing](#integration-testing)
  - [e2e testing](#e2e-testing)
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
  - [Alternative: Use YAML directly](#alternative-use-yaml-directly)
  - [Alternative: Combine go structs with fieldset mask](#alternative-combine-go-structs-with-fieldset-mask)
  - [Alternative: Use varadic function based builders](#alternative-use-varadic-function-based-builders)
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
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
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

client-go's typed clients need a typesafe, programmatic way to make apply
requests.

## Motivation

Currently, the only way to invoke server side apply from client-go is to call
`Patch` with `PatchType.ApplyPatchType` and provide a `[]byte` containing the
YAML or JSON of the apply configuration. This has a couple important
deficiencies:

- It is a gap completeness of the type client, which provides typesafe APIs for
  all other major methods.
- It makes it to easy for developers to make a major, but non-obvious mistake:
  Use the existing go structs to construct an apply configuration, serialize
  it to JSON, and pass it to `Patch`. This can cause zero valued required
  fields being accientally included in the apply configuration resulting
  in fields being accidentally set to incorrect values and/or fields accidentally
  being accidentally being clamed as owned.

Both sig-api-machinery and wg-api-expression agree that this enhancement is
required before server side apply to be promoted to GA.

### Goals

Introduce a typesafe, programmatic way to call server side apply using the typed
client in client-go.

Express all apply configurations in Go that can be expressed in
YAML. Specifically, an apply request must include only field that are set by
applier and exclude those not set by applier.

Validate this enhancement meets the needs of developers:

- An developer not directly involved in this enhancement successfully converts
  a 1st party controller (one in github.com/kubernetes/kubernetes) to us this
  enhancement.
- A representative group of the developer community is made aware of this
  proposed enhancement, is given early access to it via a fork of
  controller-tools with the requisite generators, and is given the opportunity
  to try it out and provide feedback.

### Non-Goals

Enhancements to client-go's dynamic client. The client-go dynamic client already
supports Apply via Patch, which is adequate for the dynamic client.

Protobuf support. Apply does not support protobuf, and it will not be added with
this enhancement.

## Proposal

`Apply` functions should be included in the typed clients generated for
client-go and should accept the apply configuration using a strongly typed
representation, which will need to be generated for this purpose.

### Risks and Mitigations

#### Poor adoption

Risk: Developers adoption is poor, either because the asthetics/ergonomics are
not to their liking or the functinality is insufficient to do what they need to
do with it. This could lead to (a) poor server side apply adoption, and/or (b)
developers building alternate solutions.

Mitigation: We are working with the kubebuilder community to
get hands on feedback from developers to guide our design decisions around
asthetics/egronomics with a goal of having both client-go and kubebuilder
take an aligned approach to adding apply to clients in a typesafe way.

## Design Details

### Apply functions

The client-go typed clients will be extended to include Apply functions, e.g.:

```go
func (c *deployments) Apply(ctx Context, deployment *appsv1apply.Deployment, fieldManager string, metav1.ApplyOptions) (*Deployment, error)
```

`ApplyOptions` will be added to metav1 even though `PatchOptions` will continue
to be used over the wire. This will make it obvious in the Apply function
signature that `fieldManager` is required.

```go
type ApplyOptions struct {
  DryRun []string `json:"dryRun,omitempty" protobuf:"bytes,1,rep,name=dryRun"`
  Force *bool `json:"force,omitempty" protobuf:"varint,2,opt,name=force"`
}

func (ApplyOptions) ToPatchOptions(fieldManager string) PatchOptions
```

Apply will combine the `fieldManager` argument with `ApplyOptions` to create the
`PatchOptions`.

Each apply call will be required to provide a fieldmanager name. We will not
provide a a way for the fieldmanager name to be set for the entire
clientset. There are a couple reasons for this:

- If a client has multiple code paths where it makes apply requests to the same
  object, but with different field sets, they must use different field manager
  names. If they use the same field manager name they will cause fields to be
  accidentally removed or disowned. This is a potential foot gun we would like to
  avoid.

- Apply requests always conflict with update requests, even if they were made by
  the same client with the same field manager name. This is by design. So when a
  controller migrates from update to apply, it will need to deal with conflicts
  regardless of what field manager name is used.

### Generated apply configuration types

All fields present in an apply configuration become owned by the applier after
when the apply request succeeds. Go structs contain zero valued fields which are
included even if the user never explicitly sets the field. Required boolean
fields are a good example of fields that would be applied incorrectly using go
structs, e.g. `ContainerStatus.Ready` (required, not omitempty). Because of this
we cannot use the existing go structs to represent apply configurations.

<<[UNRESOLVED @jpbetz @jennybuckley ]>> 
Finalize which alternative to use based on developer feedback. See the
[Alternatives](#alternatives) for a complete list, but are currently focusing on
the two below alternatives. We are working with the Kubebuilder community to
gather feedback on what developers prefer.
<<[/UNRESOLVED]>>

#### Alternative 1: Genreated structs where all fields are pointers

Example usage:

```go
&appsv1apply.Deployment{
  Name: &pointer.StringPtr("nginx-deployment"),
  Spec: &appsv1apply.DeploymentSpec{
    Replicas: &pointer.Int32Ptr(0),
    Template: &v1apply.PodTemplate{
      Spec: &v1apply.PodSpec{
        Containers: []v1.Containers{
          {
            Name: &pointer.StringPtr("nginx"),
            Image: &pointer.StringPtr("nginx:latest"),
          },
        }
      },
    },
  },
}
```

For this approach, developers need to use a library like
https://github.com/kubernetes/utils/blob/master/pointer/pointer.go
to inline primitive literals.

#### Alternative 2: Generated "builders"

Example usage:

```go
&appsv1apply.Deployment().
  SetObjectMeta(&metav1apply.ObjectMeta().
    SetName("nginx-deployment")).
  SetSpec(&appsv1apply.DeploymentSpec().
    SetReplicas(0).
    SetTemplate(
      &v1apply.PodTemplate().
        SetSpec(&v1apply.SetPodSpec().
          SetContainers(v1apply.ContainerList{
            v1apply.Container().
              SetName("nginx").
              SetImage("nginx:1.14.2")
            v1apply.Container().
              SetName("sidecar").
          })
        )
      )
    )
  )
```

#### Comparison of alternatives

See https://github.com/kubernetes/kubernetes/pull/95988 for a working implementation
of alterative 1 and https://github.com/jpbetz/kubernetes/tree/apply-client-go-builders
for a working implementation of alternative 2.

Of the two leading alternatives--"builders" and "structs with pointers"--we implemented
prototypes of both. They had roughly equivalent performance, and no differences
in their capabilities. The choice between the two is primarily one of asthetics/ergonomics.

Some of the feedback we have heard from the community:

- "structs with pointers" feels more go idiomatic and more closely aligned with
  the go structs used for Kubernetes types both for builtin types and by
  Kubebuilder.
- It's nice how "builders" are clearly visually distinct from the main go types.
- Having to use a utility library to wrap literal values as pointers for the
  "structs with pointers" is not a big deal. I'm already familiar
  with having to do this in go and once I learn use a utility library for it
  I'm all set.
- The "builders" are awkward to use in an IDE. I felt like I was fighting with
  my IDE to get chain function calls and organize them hierarchally as expected
  by this approach.

TODO: We are providing the developer community with a fork of controller-tools
that will allow them to generate apply configuration types that have both the
alternatives. Our goal is to get feedback from developers after they try out the
generated apply configurations and use that to inform our decision.

While it is possible to generate types that have both the pointer fields exposed
and the builder functions, we would prefer to provide clear guidance to the
community on how apply configurations should be represented in go and encourage
consistent use of only one of these approaches.

#### DeepCopy support

If "structs with pointers" approach is used, the existing deepcopy-gen
can be used to generate deep copy impelemntations for the generated apply
configuration types.

#### Code Generator Changes

hack/update-codegen.sh and hack/verify-codegen.sh will be updated to generate
the apply functions and apply configuration types.

##### Addition of applyconfiguration-gen

- Add staging/src/k8s.io/code-generator/cmd/applyconfigurations-gen
- Generates into staging/vendor/k8s.io/client-go/applyconfigurations/
- Only generate builders for struct types reachable from the types that have the +clientgen annotation
- Don't generate builders for MarshalJSON types (Quantity, IntOrString)
- Don't generate builders for RawExtension or Unknown

##### client-gen changes

Since client-gen is available for use with 3rd party project, we must ensure all
changes to it are backward compatible. The Apply functions will only be generated
by client-gen if a optional flag is set.

The Apply functions will be included for all built-in types. Strictly speaking
this can be considered a breaking change to the generated client interface, but
adding functions to interfaces is a change we have made in the past, and developers
that have alternate implementations of the interface will usually get a compiler
error in this case, which is relatively trivial to resolve


### Interoperability with structured and unstructured types

For "structs with pointers", json.Marshal, json.Unmarshal and conversions to and
from unstructured work the same as with go structs.

For "builders", each could implement `MarshalJSON`, `UnmarshalJSON`,
`ToUnstructured` and `FromUnstructured`.

### Test Plan

#### Fuzz-based round-trip testing

All generated types will be populated using the existing Kubernetes type fuzzer
(see pkg/api/testing) and round tripped to/from the go types. This ensures that
all the generated apply configuration types are able to be correctly populated
with the full state of the go types they mirror.

### Integration testing

The Apply function and the generated apply configuration types will be tested
together in test/integration/client/client_test.go.

### e2e testing

We will migrate the cluste rrole aggregation controller to use apply and verify
it is correct using the existing e2e tests, expanding coverage as needed.

### Graduation Criteria

This enhancement will graduate to GA as part of Server Side Apply. It does
not make sense to graduate it independently.

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness/README.md.

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

Use of apply is opt-in by clients. Clients transitioning from update to apply
may choose to a transition strategy appropriate for their use case. Typically
test coverage should be sufficient, but in some cases involving more complex
logic it might be appropriate to put the new apply logic behind a feature
flag so it is possible to rollback to update if there is an unexpected issue.


### Rollout, Upgrade and Rollback Planning

See above.

### Monitoring Requirements

Server side apply monitoring is already in place and is sufficient.

### Dependencies

Depends on server side apply which has been in beta since 1.16.

### Scalability

This is a client feature and does not directly impact system scalability, other
than the potential to increase adoption of server side apply, which has already
been validated to have minor additional server side processing compared with
update.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
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

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

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

- Increases hack/update-codegen.sh by roughly 5 seconds.
- Increases client-go API surface area.

## Alternatives

### Alternative: Use YAML directly

For fields that need to be set programmatically, use templating.

Limitations:

- Not typesafe, so arguably should be part of a dynamic client only (which can already do apply)
- Templating doesn't work well for some cases. E.g. a variable number of containers


### Alternative: Combine go structs with fieldset mask

User directly provides the go structs as they exist today and also provides a fieldset "mask" that enumerates all the fields included in the apply configuration. A custom serializer would be required to combine the object and the mask together.

```
obj := &appsv1.Deployment{ …}
mask := TODO
tombstoned := TODO: is another fieldset required for tombstones?
Apply(..., obj, mask, tombstoned, …)
```

Limitations:

- Error prone. No way to ensure that the mask and the object have the same set of fields directly set by the caller (e.g. if the user directly sets a field to its zero value, there is no way to warn them that they forgot to add it to the mask)
- Even if there was some typesafe way to define masks and tombstones, constructing them is going to add to the work required by client-go apply users.

### Alternative: Use varadic function based builders

```
appsv1apply.Deployment(
  metav1apply.ObjectMeta(
    appsv1apply.Name("nginx-deployment"),
  ),
  appsv1apply.DeploymentSpec(
    appsv1apply.Replicas(0),
    appsv1apply.PodTemplate(
      appsv1apply.PodSpec(
        appsv1apply.TombStoned("hostname"),
        appsv1apply.PodContainer(
          appsv1apply.Name("nginx"),
          appsv1apply.Image("nginx:1.14.2"),
        ),
        appsv1apply.TombStoned(
          appsv1apply.PodContainer(
            appsv1apply.Name("sidecar"),
          ),
        ),
      ),
    ),
  ),
)

```

This could be implemented by generating varadic functions, e.g.:

```
func Deployment(fields ...DeploymentField{}) {
   var object map[string]interface{} // This is the underlying data structure
   for field := range fields {
     switch f := fields.(type) {
     case NameField:
       object["name"] = f.value
     // other types
     }
   }
}

func Name(value string) DeploymentField { … }
```

Limitations:

- Lots of identifier collision issues to deal with. For example, we can't have multiple "Name" functions in the same package. This can probably be mitigated by either generating more unique names or by allowing a common field like Name, which is typically a string, to be shared across multiple structs that have name fields.

