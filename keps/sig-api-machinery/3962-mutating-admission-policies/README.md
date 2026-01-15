# KEP-3962: Mutating Admission Policies

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Summary](#summary-1)
  - [Phase 1](#phase-1)
    - [API Shape](#api-shape)
    - [Old object state](#old-object-state)
    - [Construct Typed Object](#construct-typed-object)
    - [Bindings](#bindings)
    - [Parameterization](#parameterization)
    - [Reinvocation](#reinvocation)
    - [Metrics](#metrics)
  - [Phase 2](#phase-2)
    - [Construct Type Enforcement](#construct-type-enforcement)
    - [Unsetting values](#unsetting-values)
      - [Safety](#safety)
    - [CEL Library Change](#cel-library-change)
    - [Share Bindings](#share-bindings)
    - [Type Handling](#type-handling)
    - [Composition variables](#composition-variables)
  - [Risk](#risk)
  - [User Stories](#user-stories)
    - [Use case: Set a label](#use-case-set-a-label)
    - [Use case: AlwaysPullImages](#use-case-alwayspullimages)
    - [Use case: DefaultIngressClass](#use-case-defaultingressclass)
    - [Use case: DefaultStorageClass](#use-case-defaultstorageclass)
    - [Use case: DefaultTolerationSeconds](#use-case-defaulttolerationseconds)
    - [Use case: if-conditional based on value contained in nested map-list](#use-case-if-conditional-based-on-value-contained-in-nested-map-list)
    - [Use case: LimitRanger](#use-case-limitranger)
    - [Use case: priority class](#use-case-priority-class)
    - [Use case: Sidecar injection](#use-case-sidecar-injection)
    - [Use case: Remove an annotation](#use-case-remove-an-annotation)
    - [Use case: If an annotation is set, set a field instead](#use-case-if-an-annotation-is-set-set-a-field-instead)
    - [Use case: modify deprecated field under CRD versions](#use-case-modify-deprecated-field-under-crd-versions)
    - [Use Case - mutation VS controller fight](#use-case---mutation-vs-controller-fight)
    - [Use Case - limitation](#use-case---limitation)
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
  - [Alternative 2: Introduce new syntax](#alternative-2-introduce-new-syntax)
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

This enhancement adds mutating admission policies, declared using CEL
expressions, as an alternative to mutating admission webhooks. This continues
the work started by [`KEP-3488`
API](/keps/sig-api-machinery/3488-cel-admission-control/README.md) for
validating admission policies.

This enhancement proposes an approach where mutations are declared in CEL by
combining CEL's object instantiation, JSON Patch, and Server Side Apply's merge algorithms.

## Motivation

A large proportion of mutating admission needs are for relatively simple
operations such as setting a label, setting a field, or adding a sidecar
container to a pod. These mutations can be expressed trivially in only a few
lines of CEL, eliminating the developmental and operational complexity of a
webhook.

Offering CEL based mutation also has other fundamental advantages over webhooks.
CEL mutations can be declared in a way that allows the kube-apiserver to
introspect the mutation and extract useful information about which fields the
mutation operation reads and writes. This information can be leveraged to do
things like finding order for mutating admission policies that minimizes the
need for reinvocation. Also, in-process mutation is sufficiently fast,
especially when compared with webhooks, that it is reasonable to re-run
mutations to do things like validate that multiple muation policy is still
applied after all other mutating admission operations have been applied.

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

### Summary

Before getting into all the individual fields and capabilities, let's look at the general "shape" of the API.
Very similar to what we have in ValidatingAdmissionPolicy, this API separates policy definition from policy configuration by splitting responsibilities across resources. The resources involved are:
- Policy definitions (MutatingAdmissionPolicy)
- Policy bindings (MutatingAdmissionPolicyBinding)
- Policy param resources (custom resources or config maps)

The idea is to leverage the CEL power of the object construction and allow users to define how they want to mutate the admission request through CEL expression.
This proposal aims to allow mutations to be expressed using JSON Patch, or the "apply configuration" introduced by Server Side Apply. 
And users would be able to define only the fields they care about inside MutatingAdmissionPolicy, the object will be constructed using CEL which would be similar to a Server Side Apply configuration patch and then be merged into the request object using the structural merge strategy. 
See sigs.k8s.io/structured-merge-diff for more details.

Note: See the alternative consideration section for the alternatives.

Pros:
- JSON Patch provides a migration path from mutating admission webhooks, which must use JSON Patch.
- Also build on Server Side Apply so that we will continue investing SSA as the best way to do patch updates to resources;
  - Does not require the users to learn a new syntax;
  - Inherit the declarative nature;
  - Leverages existing merging strategy, markers and openapi extensions.

Cons:
- Lack of deletion support (see the unsetting values section for the details and workaround);
- Migration effort from Mutation Webhook
  

### Phase 1

#### API Shape
Similar to the `validations` field in ValidatingAdmissionPolicy, a `mutations` field will be defined inside MutatingAdmissionPolicy which allows users to define a list of mutations that apply to the specific resources.
Each mutation field contains a CEL expression which evaluates to a partially populated object representing a Server Side Apply "apply configuration". The apply configuration is then merged into the request object.

Here is an example of injecting an initContainer.

```yaml
apiVersion: admissionregistration.k8s.io/v1
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
  matchConditions:
    - name: does-not-already-have-sidecar
      expression: "!object.spec.initContainers.exists(ic, ic.name == params.name)"
  failurePolicy: Fail
  reinvocationPolicy: IfNeeded
  mutations:
    - patchType: "ApplyConfiguration" // "ApplyConfiguration", "JSONPatch" supported. 
      expression: >
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
            ]
          }
        }
```
The field `patchType` is used to specify which strategy is used for the mutation.
Supported values include "ApplyConfiguration", "JSONPatch". 
The "ApplyConfiguration" strategy will prevent user from performing ambiguous action like manipulating atomic list. 
The detailed definition of ambiguous action should be reviewed before beta.
For any mutation requires modification regarding with ambiguous action, "JSONPatch" strategy is needed.

The "JSONPatch" strategy will use JSON Patch like what is done in Mutating Webhook. 

Example JSON Patch:

```yaml
apiVersion: admissionregistration.k8s.io/v1
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
  matchConditions:
    - name: does-not-already-have-sidecar
      expression: "!object.spec.initContainers.exists(ic, ic.name == params.name)"
  failurePolicy: Fail
  reinvocationPolicy: IfNeeded
  mutations:
    - patchType: "JSONPatch"
      expression: >
        JSONPatch{op: "add", path: "/spec/initContainers/-", value: Object.spec.initContainers{
                name: params.name,
                image: params.image,
                args: params.args,
                restartPolicy: params.restartPolicy
                // ... other container fields injected here ...
        }
```

When "ApplyConfiguration" specified, the expression evaluates to an object that has the same type as the incoming object, and the type alias Object refers to the type (see Type Handling for details).

By using Server Side Apply merge algorithms, schema declarations like
`x-kubernetes-list-type: map`, that control how a merge is performed, will be
respected.

However, unlike with server side apply, these mutations will not have a field
manager specified. This has important implications in how the merge is performed that will
be discussed in more detail in the below "Unsetting values" section.

Note: Mutation policy will generally follow the way how mutation webhook deals with field manager.

In this example, note that:

- `Object{}`, `Object.spec{}` and similar are CEL object instantiations, and are
  used to create a subset of the fields of a `Pod`.
- `object` refers to the state of the object before the mutation policy is applied.
- `oldObject` refers to the state of object currently in etcd.
- `params` refers to the param resource.

To use this `MutatingAdmissionPolicy` we first must create a policy binding
and `Sidecar` parameter resource:

```yaml
# Policy Binding
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingAdmissionPolicyBinding
metadata:
  name: "sidecar-binding-test.example.com"
spec:
  policyName: "sidecar-policy.example.com"
  paramRef:
   name: "meshproxy-test.example.com"
   namespace: "default"
---
# Sidecar parameter resource
apiVersion: mutations.example.com
kind: Sidecar
metadata: 
  name: meshproxy-test.example.com
spec:
  name: mesh-proxy
  image: mesh/proxy:v1.0.0
  args: ["proxy", "sidecar"]
  restartPolicy: Always
```

Next, we can test the policy with a pod:

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

#### Old object state

Sometimes the current state of the object will be needed. This is available via
the `object` variable. For example, to update all containers in a pod to use
the "Always" imagePullPolicy:

```cel
Object{
  spec: Object.spec{
    containers: object.spec.containers.map(c,
      Object.spec.containers.item{
          name: c.name,
          imagePullPolicy: "Always"
      }
    )
  }
}
```

#### Construct Typed Object
In the mutation expressions, CEL supports constructing the object with a named type. 
At the language level, the named type can be anything that the CEL library registers with the environment. 
For MutatingAdmissionPolicy, Object is the alias of the type that the incoming object confirms.

With the object construction syntax, field names are no longer quoted because they are no longer map keys. 
If any of the object construction violates the defined schema, the expression compilation will error and the user can retrieve the error from the rejection message.
For list types, i.e. with the OpenAPI type of “array”, the special item field resolves the type of its items.
In Alpha 1, the CEL environment and its type providers compile the constructed object, but make no effort to check if the field names and types match these of the schemas (i.e. everything is still Dyn). 
See Construct Type Enforcement in Alpha 2 for future plans.

#### Bindings

Bindings will be almost the same as `ValidatingAdmissionPolicyBinding`, but with
the following difference:

- No `validationActions` field (unless anyone can think of any useful way to offer a dry-run type option)

#### Parameterization
Similar to ValidatingAdmissionPolicy, the mutation admission policy can refer to a param, and the param object can be specified per-namespace. 
We expect to fully reuse existing params handling logic from ValidatingAdmissionPolicy.

#### Reinvocation

The existing reinvocation policy established between webhooks and admission controllers will be extended to also handle admission policies.
Ref: [the current re-invocation mechanism for webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#reinvocation-policy).
Admission policies will be reinvoked after admission controllers and before webhooks. 

With mutating admission policies added, the the mutating admission plugin order will become:
- Mutating admission controllers(e.g. DefaultIngressClass, DefaultStorageClass, etc)
- Mutating admission policies (introduced within this enhancement and the order will be discussed below)
- Mutating admission webhooks (ordered lexicographically by webhook name)

To allow mutating admission plugins to observe changes made by other plugins, built-in mutating admission plugins are re-run if a mutating webhook modifies an object, 
same will apply with mutating policy. The mutating policies are rerun if a mutating webhook or mutation policy modifies an object.

For the running order within mutating admission policies, there are a couple options proposed:
- option 1(suggested by @deads2k): ordered randomly but keep the same random order while reinvocation.
  - Pros: 
    - Encourage user to write order-independent mutations
  - Cons: 
    - The final state of request is not deterministic
    - The mutation should not have dependencies in between
- option 2: the lexicographical ordering of the resource names
  - Pros:
    - Align with the behavior with mutating webhook
  - Cons:
    - User has to be mindful on the order if there is dependency existing
    - User has a hacky way to enforce the order

Considering it would be easier to go with random order and then switch to a particular order, 

Notes: If the mutations run in random order, a concern would be if people didn't write idempotence mutations,
the result might be different between two admission request. Please refer to Safety section for ways to check idempotence.

#### Metrics
Goals:
- Parity with validating admission policy metrics 
  - Should include counter of deny, success violations 
  - Label by {policy, policy binding, mutation expression} identifiers
- Counters for number of policy definitions and policy bindings in cluster 
  - Label by state (active vs. error), enforcement action (deny, warn)
- Counters for Variable Composition 
  - Should include a counter of variable resolutions to measure time saved. 
  - Label by policy identifier


### Phase 2

All these capabilities are required and should be discussed thoughtfully before Beta, but will not be implemented in the first alpha release of this enhancement due to the size and complexity of this enhancement.

#### Construct Type Enforcement

The type alias Object and its descendants, in Phase 2, are now real types that derive from the resolved OpenAPI schemas. 
If any type violations happen in the constructed object, the CEL checker will raise the errors before the expressions evaluate.

For bigger schemas, the construction of CEL types can be expensive. It is recorded to take ~100ms to resolve and parse apps/v1.Deployment. 
Optimizations like caching or lazy sub-schema resolution can be candidates of beta/GA graduation criteria.

#### Unsetting values

Since there is no field manager used for the merge, the server side apply merge
algorithm will only add and replace values. To unset values, JSON Patch mutations must be used.

##### Safety

To ensure mutations are not "broken" by other mutations (overwritten, undone, or
otherwise invalidated) and ensure the deterministic final state due to the random running order,
we provide an option to check if rerun certain mutation policy leads to object change.
It also helps to ensure the mutation is written in a idempotent way in consideration of the random running ordering. 

These validation checks will be declared using a `mutationValidationPolicy` field, which is an enum of the following values:
- Fail - Replaying the mutation on the mutated object should result in an identical object,  if not, fail the request.
- Warn - Replaying the mutation on the mutated object should result in an identical object,  if not, pass the request with a warning message.
- Skip - Don't replay the mutation.

For example:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingAdmissionPolicy
metadata:
  name: "sidecar-policy.example.com"
spec:
  # ...
  mutations:
    expression: >
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
            ] + object.spec.initContainers
          }
        }
  mutationValidationPolicy: Fail
```
If rerun the mutation policy caused object change, the request should be failed

To validate an object after all mutations are guaranteed complete, we highly recommend to use a validating admission policy to validate the final state of object. 

#### CEL Library Change
We expect this feature to require minimal changes to the core or the Kubernetes-specific CEL library. 
However, this feature uses the optional library in a way that the library was not designed to. We acknowledge the risk where not all current or future features of the optional library will be available.

We will be evaluating the existing CEL library to see if any specific func should be added for mutation use case.
A potential candidate would be the hashing function which might be helpful in the recent discussion of controller sharding.
Ref: https://github.com/timebertt/kubernetes-controller-sharding/blob/main/docs/design.md#the-clusterring-resource-and-sharder-webhook (the kubernetes-controller-sharding project could also eliminate their webhook if we supported this). 

In consideration of written expression for deep nested list/map, library which could help with flatten the list or accumulation alike functions might be useful to add.

#### Share Bindings

The suggested best practice of using MutatingAdmissionPolicy would be having ValidatingAdmissionPolicy also set to validate if the request matches the desired result. 
However, having both MutatingAdmissionPolicy and ValidatingAdmissionPolicy with parameterization would result in 6 new resources(policy, binding and params for both Mutating and validating).
A possible path would be allowing one binding to bind both MutatingAdmissionPolicy and ValidatingAdmissionPolicy which could be further discussed before going to Beta.

#### Type Handling

The type system works differently than ValidatingAdmissionPolicy in the following aspects.

1. Schema Enforcement of Structural Merge Diff

  The resulting object will be converted into typed objects as understood by SMD, with existing SMD schema validations still effective. 
  Should SMD return an error during the conversion, the error handling follows the failure policy. 
  Note that it is possible to pass CEL expression compilation but still fail the schema validation if Variable Composition is used, see Variable Composition below.
2. Type Checking Before Runtime

  Similar to ValidatingAdmissionPolicy, a controller, running in kube-controller-manager, compiles the expressions against the types defined in matchConstraints. 
  The number of types to check are also heavily limited to prevent the checks taking up too much computing time from the KCM.

#### Composition variables
To control the size of mutation expression, and to better reuse parts of the expressions, sub-expressions can be extracted into a separate variables section, similar to variables of ValidatingAdmissionPolicy.
```yaml
variables:
  - name: targetContainers
    expression: >-
      object.spec.template.containers.filter(c,
      c.image.contains("example.com"))
  - name: transformedContainers
    expression: >
      variables.targetContainers.map(c, {"name": c.name, "env": {"name": "FOO",
      "value": "foo"}})
mutations:
  - patchType: "ApplyConfiguration"
    expression: |
      Object{
          spec: Object.spec{
              template: Object.spec.template{
                  containers: variables.transformedContainers
              }
          }
      }

```

With variable composition, it is possible to escape from compile-time type checking. For example
```yaml
variables:
  - name: definitelyNotAContainer # resulting type is Dyn
    expression: >-
      params.foo == "bar" ? true : "true"
mutations:
  - patchType: "ApplyConfiguration"
    expression: |
        Object{
          spec: Object.spec{
              template: Object.spec.template{
                  containers: [variables.definitelyNotAContainer] # will pass, but error at runtime.
              }
          }
      }

```

### Risk

1. Ensure the final state match expectation.
There might be multiple mutating admission policies, mutating webhooks, other controllers trying to mutate the incoming request and each happens separately, 
and they might mutate the same part of the object. 
It might be hard to ensure that the final state matches expectations. 
For best practice, the validation process is highly recommended whenever there is a mutation process set up. The validating admission policy is recommended to be set up whenever a mutation admission policy is set to verify the final state of the data matches the expectation. Also refer to the Safety section for further details.

2. Failures in MutatingAdmissionPolicy will fail request in admission chain.
If the failure policy is set to fail and the mutation admission policy matches all resources, the failure/error in MAP might infect the control plane availability.

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

#### Use case: AlwaysPullImages
Force all image pull policy under containers to Always
```cel
Object{
    spec: Object.spec{
        containers: object.spec.containers.map(c,
            Object.spec.containers.item{
                name: c.name,
                imagePullPolicy: "Always"
            }
        )
        // ... same for initContainers and ephemeralContainers ...
    }
}
```

#### Use case: DefaultIngressClass
While creation of Ingress objects that do not request any specific ingress class,
adds a default ingress class to them.

```yaml
matchConditions:
  - name: 'need-default-ingress-class'
    expression: '!has(object.spec.ingressClassName)'
mutations:
  - patchType: "ApplyConfiguration"
    expression: |
      Object{
        spec: Object.spec{
          ingressClassName: "defaultIngressClass"
        }
      }
```

#### Use case: DefaultStorageClass
While creation of PersistentVolumeClaim objects that do not request any specific storage class,
adds a default storage class to them.
```yaml
matchConditions:
  - name: 'need-default-storage-class'
    expression: '!has(object.spec.storageClassName)'
mutations:
  - patchType: "ApplyConfiguration"
    expression: |
      Object{
        spec: Object.spec{
          storageClassName: "defaultStorageClass"
        }
      }
```

#### Use case: DefaultTolerationSeconds
Sets the default forgiveness toleration for pods to tolerate the taints `notready:NoExecute`.

Should be supported through `JSONPatch`.

#### Use case: if-conditional based on value contained in nested map-list
If the volumemount specified in containers does not have a volume associated, add a volume.
```yaml
variables:
  - name: volumeMountsList
    expression: "object.spec.containers.map(c, c.volumeMounts.map(v, v.name))"
  - name: volumesList
    expression: "object.spec.volumes.map(v, v.name)"
mutations:
  - patchType: "ApplyConfiguration"
    expression: |
      Object{
        spec: Object.spec{
          volumes: volumeMountsList.filter(n, !(n in volumesList)).map(v, {
              name: v,
              configMap: params.addFields
          })
        }
      }
```
It could be simplified with composition variables.
I have a [gist example](https://gist.github.com/cici37/e8181e53069435a307cd7822305b217c) here.

#### Use case: LimitRanger
Apply default resource requests to Pods that don't specify any.
```yaml
mutations:
  - patchType: "ApplyConfiguration"
    expression: |
      Object{
        spec: Object.spec{
          containers: object.spec.containers.filter(c, !has(c.resources)).map(c, 
            {
                name: c.name,
                resources: {#default resources settings}
            }
        }
      }
```

#### Use case: priority class 

Add a default priority class if it is not set in pod
```yaml
matchConditions:
  - name: 'no-priority-class'
    expression: '!has(object.spec.priorityClassName)'
mutations:
  - patchType: "ApplyConfiguration"
    expression: |
      Object{
        spec: Object.spec{
          priorityClassName: params.defaultPriorityClass
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
    ] + object.spec.initContainers
  }
}
```


#### Use case: Remove an annotation

```cel
JSONPatch{
    op: "remove",
    path: "/metadata/annotations/annotation-to-unset"
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
    someField: object.annotations["some-annotation"]
  }
}
```

#### Use case: modify deprecated field under CRD versions
Support atomic list modification through JSON Patch


#### Use Case - mutation VS controller fight
https://github.com/open-policy-agent/gatekeeper/issues/2963#issuecomment-1683971371
Out of scope. The proposed feature is going to be added as an admission plugin which has no control over other controllers potentially being added.

#### Use Case - limitation

The current design will not support the following use cases
- Involves creation of additional resources
- Reference additional resources which is not fixed
- It is tricky to write expression in deeply nested list/map with conditional check 

For 1, additional resources creation is not supported since the current design focuses on updating the incoming request.

For 2, parameter resources could potentially be used for any operation requires additional resource involved.
However, if the additional resource involved is based on querying in incoming request, it will not be supported.

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

Risk: Final state of the object might not match the output of mutation policies.
There might be multiple mutating admission policies, mutating webhooks, other controllers trying to mutate the incoming request and each happens separately, 
and they might mutate the same part of the object. It might be hard to ensure that the final state matches expectations. 

- Mitigation/Justification: For best practice, the validation process is highly recommended whenever there is a mutation process set up. 
The validating admission policy is recommended to be set up whenever a mutation admission policy is set to verify the final state of the data matches the expectation. Also refer to the Safety section for further details.



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

[ x ] I/we understand the owners of the involved components may require updates to
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

In the alpha phase, the integration tests are expected to be added for:

- The behavior with feature gate and API turned on/off and mix match
- The happy path with everything configured and mutation proceeded successfully
- Mutation with different failure policies
- Mutation with different Match Criteria
- Mutation violations for different reasons including type checking failures, misconfiguration, failed mutation, etc and formatted messages
- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->
We will test the edge cases mostly in integration test and unit test. We may add e2e test for spot check of the feature presence.
- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Support both `JSONType` and `ApplyConfiguration` for `patchType`
- Composition variable support is needed before going to beta
- Initial e2e tests completed and enabled

#### Beta

- Have proper monitoring for MAP admission plugin
- Fix any blocking issues/bugs surfaced before code freeze
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Including all function needed with performance and security in consideration

#### GA

- N examples of real-world usage
- N installs
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
- Support both `JSONType` and `ApplyConfiguration` for `patchType`
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

No changes are required for a cluster to make an upgrade and maintain existing behavior.
There is new API that does not effect the cluster during upgrade. It only has effects
if it is used after the upgrade.

If a cluster is downgraded, no changes are required. The cluster continues to work as expected since
the alpha version will have functionality compatible with beta and stable release, any downgrade
will be to a version that also contains the feature.

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

This feature is implemented in the kube-apiserver component, skew with other
kubernetes components do not require coordinated behavior.

Clients should ensure the kube-apiserver is fully rolled out before using the
feature.

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

- [ x ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: MutatingAdmissionPolicy
  - Components depending on the feature gate: kube-apiserver


###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
No, default behavior is the same.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
Yes, disabling the feature will result in mutation expressions being ignored.

###### What happens if we reenable the feature if it was previously rolled back?
The MutatingAdmissionPolicy will be enforced again.

###### Are there any tests for feature enablement/disablement?
Unit test and integration test will be introduced in alpha implementation.

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
The existing workload could potentially be mutated and cause unexpected data stored if the mutation is misconfigured.

While rollout, the cluster administrator could use `mutationValidationPolicy` to reduce the risk of unexpected mutation.
The failurePolicy could be configured to decide if a failure should reject the admission request.
In this way it will minimize the effect on the running workloads.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
On a cluster that has not yet opted into MutatingAdmissionPolicy, non-zero counts for either of the following metrics mean the feature is not working as expected:

- cel_admission_mutation_total
- cel_admission_mutation_errors

On a cluster that opt into MutatingAdmissionPolicy, consider rollout if observed elevated API server errors or excessive `apiserver_cel_evaluation_duration_seconds` / `apiserver_cel_compilation_duration_seconds`.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
Upgrade and rollback will be tested manually in a kind:

- Enabled feature gate, created a MutatingAdmissionPolicy and MutatingAdmissionPolicyBinding with mutation to add a label to a pod.

- Disabled feature gate, restarted apiserver, confirmed that the
  MutatingAdmissionPolicy and MutatingAdmissionPolicyBinding still exist. Added another Pod
  to verify that the mutation would not happen.

- Re-enabled the feature gate, restarted apiserver, confirmed that
  the mutation will occur for new incoming pod creation request.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
No.

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
The following metrics could be used to see if the feature is in use:

- mutating_admission_policy/check_total
- mutating_admission_policy/definition_total

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->
Metrics like mutating_admission_policy/check_total can be used to check how many mutations applied in total

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
No impact on latency for admission request when MutatingAdmissionPolicy are absent.
Performance when MutatingAdmissionPolicy are in use will need to be measured and optimized before GA.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
    The Metrics below could be used:
    mutating_admission_policy/check_total
    mutating_admission_policy/definition_total
    mutating_admission_policy/check_duration_seconds

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
No. 

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->
No.

###### Does this feature depend on any specific services running in the cluster?
No.
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
Yes. A new API group is introduced which will be used for this feature.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
Yes. We introduced two new kinds for this feature: MutatingAdmissionPolicy and MutatingAdmissionPolicyBinding as described in this doc.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
The existing admission request latency might be affected when the feature is used. We expect this to be negligible and will measure it before GA.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
We don't expect it to. Especially comparing to the existing method to achieve the same goal, using this feature will not result in non-negligible increase of resource usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->
No.

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
No change from existing behavior. The feature will serve same as if it's disabled.

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
Same as without this feature.

###### What are other known failure modes?
N/A

###### What steps should be taken if SLOs are not being met to determine the problem?
The feature can be disabled by disabling the API or setting the feature-gate to false if the performance impact of it is not tolerable.
Try to run the validations separately to see which rule is slow
Remove the problematic rules or update the rules to meet the requirement

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

- v1.32: Alpha
- v1.34: Beta
- v1.36: Stable

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
Here are the alternative considerations compared to using JSON Patch and the apply configurations introduced by Server Side Apply.

### Alternative 2: Introduce new syntax
  
Another alternative consideration would be rewriting your own merge algorithm which is a lot of duplicated effort.
- Pros:
  - More flexibility on how merging works
  - Support most of the existing use cases
- Cons:
  - Duplicated effort
  - Introducing a new language model into k8s which increase the maintenance effort


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
