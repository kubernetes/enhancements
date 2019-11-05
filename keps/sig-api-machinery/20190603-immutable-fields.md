---
title: Immutable Fields
authors:
  - "@apelisse"
  - "@sttts"
  - "@liggitt"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-api-machinery
reviewers:
  - "@erictune"
  - "@jpbetz"
  - "@liggitt"
  - "@logicalhan"
  - "@p0lyn0mial"
approvers:
  - "@liggitt"
  - "@deads2k"
editor: "@sttts"
creation-date: 2019-06-03
last-updated: 2019-10-01
status: provisional
see-also:
  - "/keps/sig-api-machinery/0006-apply.md"
---

# Immutable fields

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

A lot of fields in APIs tend to be "immutable", they can't be changed after
creation. This is true for example for many of the fields in pods. There is
currently no way to declaratively specify that fields are immutable, and one
has to rely on either built-in validation for core types, or have to build a
validating webhooks for CRDs.

Providing a new `// +immutable` marker would help 
- to make the API more descriptive to users
- to help API developers by verifying these assertions automatically
- and to publish this information via OpenAPI.

## Motivation

There are resources in Kubernetes which have immutable fields by design,
i.e. after creation of an object, those fields cannot be mutated anymore. E.g. a
pod's specification is mostly unchangeable once it is created. To change the
pod, it must be deleted, recreated and rescheduled. Users want to implement the
same kind of read-only semantics for CustomResources, for example:
https://github.com/kubernetes/kubernetes/issues/65973. Today this is only possible
with the unreasonable development overhead of a webhook.

### Goals

- extend the CRD API to be able to specify immutability for fields.
- publish the immutability field of CRDs via OpenAPI as vendor extension.
- verify immutability on CR update and patch requests.
- propose a source code marker to be consumed by kubebuilder (for CRDs) and openapi-gen (for native types).
- the semantics of immutability must be driven by:
  - we do not break/change old CRD persistence semantics.
  - the user-observed equality used for immutability checks must match the equality on
    persisted objects. I.e. if `StorageRoundtrip(object)` is the object returned by a
    create or update call, then we want that `StorageRoundtrip(a) == StorageRoundtrip(b)`
    is the equality used for comparing `a` and `b`. If that check fails, a request is rejected because of immutability conflict.
    
    The choice if this equality (vs. e.g. a semantic equality) is essential because both validation and defaulting depend on it for sane behaviour: an update to an immutable (sub-)object must not change validity or trigger defaulting to happen that did not happen before.
- the mechanism must extend to
  - the addition of protobuf or other encodings which unify values like empty, null and undefined.
  - the use for existing native types in order to replace complex validation code with a simple declarative marker on the types.
  - the restriction of the equality to only map keys, but not their values.
  - the allowance of addition and/or deletion of map keys in maps and in array of list-type `map` to support cases we have in Kubernetes API today like:
  
    - scheduler setting `spec.nodeName` (and it being immutable after that point)
    - populating `clusterIP` in services
    - append-only lists (like `ephemeralContainers`)
    - remove-only lists (like `finalizers` that may only shrink after deletion).

### Non-Goals

- The mechanism must be extensible to native types, but its implementation is optional.
- The mechanism must be extensible to future normalization behaviours which will be
  required to support protobuf for CRs. But this KEP does not aim at defining these
  and hence defining a custom equality which is compatible with normalization.
- The mechanism is not supposed to allow different orders in lists to be considered equal.

## Proposal

We propose

1. adding two optional vendor extensions to CRD OpenAPI schemas:
 
   1. `x-kubernetes-mutability: Immutable | AddOnly | RemoveOnly` and
   2. `x-kubernetes-key-mutability: Immutable | AddOnly | RemoveOnly`.

   For schemas of type `array` and for schemas of type `object` with `additionalProperties` (i.e. Golang `map[string]<type>`), we allow only `Immutable` for `x-kubernetes-mutability` and all three values of `x-kubernetes-key-mutability`.
   
   For all other schemas (especially of type `object` with `properties`, i.e. Golang structs; and all basic types), we allow all three values for `x-kubernetes-mutability` and we disallow `x-kubernetes-key-mutability`.
   
   For any other schema, 
   
   At the root-level and inside `.metadata` of an object, `x-kubernetes-mutability` and `x-kubernetes-key-mutability` are forbidden.
   
   Both extensions can only be set in `v1` CRDs on creation, but updated in `v1beta1` if preexisting.
   
2. to use **strict deep-equal** comparison of those fields marked as immutable during
   update validation in the request pipeline, with the only exception outlined below in the detailed semantics of the different values of the two vendor extensions.
   
We create another KEP to define custom normalization steps for CRs done during 
deserialization from etcd and when receiving a request (just after pruning and defaulting).

### Recursive Scope

If the `x-kubernetes-mutability` vendor extension is set (to `Immutable`, `AddOnly`, `RemoveOnly`), `x-kubernetes-mutability: Immutable` is implicitly applied recursively to all subschemas.

The `x-kubernetes-key-mutability` vendor extension does not apply recursively.

### Semantics of `x-kubernetes-mutability`

#### Immutable list, arbitrary list type

The whole list is immutable in the strict sense of `reflect.DeepEqual`.

#### Immutable map / object, arbitrary map type

The whole map is immutable in the strict sense of `reflect.DeepEqual`.

#### Mutable object, immutable fields

The value of a field is immutable, and it cannot be added (Immutable, RemoveOnly) or removed (Immutable, AddOnly).

*Example 1 - Immutable*:

```yaml
type: object
properties:
  foo:
    type: string
    x-kubernetes-mutability: Immutable | AddOnly | RemoveOnly
```

Allowed for AddOnly, disallowed otherwise:
- `{}` → `{foo:"a"}`

Allowed for RemoveOnly, disallowed otherwise:
- `{foo:"a"}` → `{}`

Disallowed:
- `{foo:"a"}` →	`{foo:"b"}`

#### Mutable array, immutable items, list type undefined / atomic

The items with their respective index in the array are immutable. Hence, appending and removal at the end of the array are allowed, the change of an item or the change of the order are not.

*Example 2*:

```yaml
type: object
properties:
  foo:
    type: array
    x-kubernetes-list-type: undefined | atomic
    items:
      type: object
      properties: ...
      x-kubernetes-mutability: Immutable
```

Allowed:
- `{}` → `{foo:...}`
- `{foo:...}` → `{}`
- `{foo:[a]}` → `{foo:[a,b]}` (value of key 0 is unchanged)
- `{foo:[a]}` →	`{foo:[]}`

Disallowed:
- `{foo:[a]}` → `{foo:[b]}`
- `{foo:[a]}` → `{foo:[b,a]}` (value of key 0 is changed)

#### Mutable array, immutable items, list type map

The key-value pairs in the array are immutable, the set of keys is not. Hence, addition and removal of key-values pairs

*Example 3*:

```yaml
type: object
properties:
  foo:
    type: array
    x-kubernetes-list-type: map
    x-kubernetes-list-map-keys: ["k"]
    items:
      type: object
      properties: ...
      x-kubernetes-mutability: Immutable
```

Allowed:
- `{}` → `{foo:...}`
- `{foo:...}` → `{}`
- `{foo:[{k:a}]}` → `{foo:[{k:a},{k:b}]}` (value of key a is unchanged)
- `{foo:[{k:a}]}` → `{foo:[]}`
- `{foo:[{k:a}]}` → `{foo:[{k:b}]}`
- `{foo:[{k:a},{k:b}]}`	→ `{foo:[{k:b},{k:a}]}`	(values of key a and b unchanged)

Disallowed:
- `{foo:[{k:a,v:1}]}` → `{foo:[{k:a,v:2}]}` (value of key a is changed)

**TODO:** removal+addition = change. The former is allowed, the latter disallowed. 

#### Mutable array, immutable items, list type set

Sets are maps with the whole value as key. Hence, addition and removal of any value in the set is allowed.

*Example 4*:

```yaml
type: object
properties:
  foo:
    type: array
    x-kubernetes-list-type: set
    items:
      type: string
      x-kubernetes-mutability: Immutable
```

Allowed:
- `{}` → `{foo:...}`
- `{foo:...}` → `{}`
- `{foo:[a]}` → `{foo:[a,b]}` (value of key a is unchanged)
- `{foo:[a]}` → `{foo:[]}` (key a is removed)
- `{foo:[a]}` → `{foo:[b]}` (key a is removed, key b is added)
- `{foo:[a]}` → `{foo:[b,a]}` (value of key a is unchanged, key b is added)

*Example 4b*: the values must be atomic. A non-trivial example is the following with map type atomic, i.e. a mutable set of atomic string maps.

```yaml
type: object
properties:
  foo:
    type: array
    x-kubernetes-list-type: set
    items:
      type: object
      x-kubernetes-mutability: Immutable
      x-kubernetes-map-type: atomic
      additionalProperties:
        type: string
```

#### Mutable map, immutable values, map type undefined / granular

Equivalently to the list type map, removal and addition of key-value pairs are allowed, while direct value change is not.

*Example 5*:

```yaml
type: object
properties:
  foo:
    type: array
    x-kubernetes-map-type: undefined | granular
    additionalProperties:
      type: string
      x-kubernetes-mutability: Immutable
```

Allowed:
- `{}` → `{foo:...}`
- `{foo:...}` → `{}`
- `{foo:{a:"1"}}` → `{foo:{a:"1", b:"2"}}` (value of key a is unchanged)
- `{foo:{a:"1"}}` → `{foo:{}}`
- `{foo:{a:"1"}}` → `{foo:{b:"1"}}`

Disallowed:

- `{foo:{a:"1"}` → `{foo:{a:"2"}}` (value of key a is changed)

#### Mutable map, immutable values, map type atomic

Same as map type map (the normal map case).

### Semantics of `x-kubernetes-key-mutability`

The vendor extension is `x-kubernetes-key-mutability` only applies to the keys, not the values of maps and arrays.

#### Immutable/AddOnly/RemoveOnly object keys, mutable values

*Example 6*:

```yaml
type: object
x-kubernetes-key-mutability: Immutable | AddOnly | RemoveOnly
properties:
  foo:
    type: string
```

Because object `properties` in Kubernetes-like APIs should not be considered as a known set, but each field individually we disallow the use of `x-kubernetes-key-mutability` in this case. CRD validation will reject it. 

#### Immutable array keys, mutable values, list type undefined / atomic

The set of indices is immutable/add-only/remove-only. Hence, appending (Immutable, RemoveOnly) or shrinking (Immutable, AddOnly) is disallowed, but changes that do not change the length are allowed.

*Example 7*:

```yaml
type: object
properties:
  foo:
    type: array
    x-kubernetes-list-type: undefined | atomic
    x-kubernetes-key-mutability: Immutable | AddOnly | RemoveOnly
    items:
      type: object
      properties: ...
```

Allowed:
- `{}` → `{foo:[]}`	(all keys are unchanged)
- `{foo:[]}` → `{}`	(all keys are unchanged)
- `{foo:[a]}` → `{foo:[b]}` (key 0 still exists)

Allowed with AddOnly, disallowed otherwise:
- `{}` → `{foo:[a]}` (key 0 is added)
- `{foo:[a]}` → `{foo:[a,b]}` (key 1 is added)
- `{foo:[a]}` → `{foo:[b,a]}` (key 1 is added)

Allowed with RemoveOnly, disabled otherwise:
- `{foo:[a]}` → `{}` (key 0 is removed)
- `{foo:[a]}` → `{foo:[]}` (key 0 is removed)

#### Immutable array keys, mutable values, list type map

The set of keys is immutable/add-only/remove-only. Hence, non-key values can be changed. New key-values cannot be added (Immutable, RemoveOnly) and old key-values cannot be removed (Immutable, AddOnly).

*Example 8*:

```yaml
type: object
properties:
  foo:
    type: array
    x-kubernetes-list-type: map
    x-kubernetes-list-map-keys: ["k"]
    x-kubernetes-key-mutability: Immutable | AddOnly | RemoveOnly
    items:
      type: object
      properties:
        k:
          type: string
```
       
Allowed:
- `{}` → `{foo:[]}` (all keys are unchanged)
- `{foo:[]}` → `{}` (all keys are unchanged)
- `{foo:[{k:a,v:1}]}` → `{foo:[{k:a,v:2}]}` (all keys are unchanged)
- `{foo:[{k:a},{k:b}]}` → `{foo:[{k:b},{k:a}]}` (all keys are unchanged)

Allowed for AddOnly, disallowed otherwise:
- `{}` → `{foo:[{k:a}]}` (key a is added)
- `{foo:[{k:a}]}` → `{foo:[{k:a},{k:b}]}` (key b is added)

Allowed for RemoveOnly, disallowed otherwise:
- `{foo:[{k:a}]}` → `{}` (key a is removed)
- `{foo:[{k:a}]}` → `{foo:[]}` (key a is removed)

Disallow:
- `{foo:[{k:a}]}` → `{foo:[{k:b}]}` (key a is removed, key b is added)

#### Immutable array keys, mutable value, list type set

The whole items are the keys. Hence, new items cannot be added (Immutable, RemoveOnly), old items cannot be removed (Immutable, AddOnly), but the order can.

Note: this is different from a set with `x-kubernetes-mutability: Immutable`. The latter does not allow order changes.

*Example 9*: 

```yaml
type: object
properties:
  foo:
    type: array
    x-kubernetes-list-type: set
    x-kubernetes-key-mutability: Immutable | AddOnly | RemoveOnly
    items:
      type: string
```

Allowed:
- `{}` → `{foo: []}` (all keys are unchanged)
- `{foo: []}` → `{}` (all keys are unchanged)
- `{foo: [a,b]}` → `{foo: [b,a]}` (all keys are unchanged)

Allowed for AddOnly, disallowed otherwise:
- `{foo:[a]}` → `{foo:[a,b]}` (key a is unchanged, key b is added)
- `{foo:[a]}` → `{foo:[b,a]}` (key a is unchanged, key b is added)

Disallowed for RemoveOnly, disallowed otherwise:
- `{foo:[a]}` → `{foo:[]}` (key a is removed)

Disallowed:
- `{foo:[a]}` → `{foo:[b]}` (key a is removed, key b is added)

#### Immutable map keys, mutable values, map type undefined / granular / atomic

Equivalently to the list type map, the set of keys is immutable|add-only|remove-only. Hence, values can be changed. New key-values cannot be added (Immutable, RemoveOnly) and old key-values cannot be removed (Immutable, AddOnly).

*Example 10*:

```yaml
type: object
properties:
  foo:
    type: object
    x-kubernetes-map-type: undefined | granular | atomic 
    x-kubernetes-key-mutability: Immutable | AddOnly | RemoveOnly
    additionalProperties:
      type: string
```

Allowed:

- `{}` → `{foo:{}}`	(keys are unchanged)
- `{foo:{}}` → `{}`	(keys are unchanged)
- `{foo:{a:"1"}` → `{foo:{a:"2"}}` (keys are unchanged)

Allowed for AddOnly, disallowed otherwise:
- `{}` → `{foo:{a:"1"}}` (key a is added)
- `{foo:{a:"1"}}` → `{foo:{a:"1", b:"2"}}` (key b is added)

Allowed for RemoveOnly, disallowed otherwise:
- `{foo:{a:"1"}}` → `{}` (key a is removed)

Disallowed:
- `{foo:{a:"1"}}` → `{foo:{b:"1"}}` (key a is removed, key b is added)
    
### Publishing

The `x-kubernetes-mutability` and `x-kubernetes-key-mutability` vendor extensions are published via `/openapi/v2` as is.

### Suggested marker syntax

In analogy to `+required`, `+optional` we propose to add a marker to kubebuilder's
controller-gen named `+immutable` meaning `x-kubernetes-immutability: Immutable` and `+immutable=AddOnly | RemoveOnly` for the other two values:

```
// The name can not be changed after creation.
// +immutable
Name string

// The list of containers can not change AT ALL after creation.
// No single field in existing containers can be changed, added or deleted,
// no new containers can be added, no existing container can be removed.
// +immutable
Containers []Containers
```

The `x-kubernetes-key-mutability` vendor extension is expressed via `// +immutable-keys` for `Immutable` and `// +immutable-keys=AddOnly | RemoveOnly` for the other two values.

### Future outline sketch: native resources

For native resources we add a pre-immutability-check normalization step for objects 
decoded from JSON which have normalizations defined:

1. versioned JSON blob comes in a request
2. unmarshalled into versioned Golang struct
3. defaulting
4. conversion to internal
5. if immutability is enabled for the resource:
   1. marshalling into JSON in-memory
   2. normalization creating a copy with shared data-structures
   3. strict immutability check against the old object, coming from proto assuming it is normalized.

### Mutating admission chain

Mutating admission chain would have the exact same effects as user changes,
meaning that they wouldn't be able to change an object after creation. This is
very similar to what is done today since validation for updates is run AFTER all
mutations.

### Where does this happen

This process is meant to happen right before the update validation and after
mutating, but before validating webhooks, and only run on updates. This will allow us to
keep the exact same behavior while removing the validation code that checks the
immutability of fields.

![Decoding steps which must apply defaults](20191001-crd-immutability-pipeline.png)

### Risks and Mitigations

- immutable metadata would break API machinery. We already forbid any specification/restriction of metadata fields other than `name` and `generatedName` today. We keep the same restriction for `x-kubernetes-mutability` and `x-kubernetes-key-mutability`, and
  at the root of the object (`x-kubernetes-mutability` would affect `metadata`, and `x-kubernetes-key-mutability` is forbidden as the root is of type `object` and `additionalProperties` is already disallowed). `kind` and `apiVersion` are
  immutable implicitly. We might publish immutable though for some of these fields.

## Design Details

### Test Plan

- exhaustive unit tests are added in apiextensions-apiserver for
  - CRD validation
    - for `x-kubernetes-mutability` and `x-kubernetes-key-mutability` at the root and in `metadata`
    - for `x-kubernetes-key-mutability` forbidden for object with `properties`
    - for `x-kubernetes-mutability: AddOnly | RemoveOnly` forbidden for type object with `additionalProperties` (maps) and arrays.
    - for `x-kubernetes-mutability` and `x-kubernetes-key-mutability` only for v1 CRDs, or during ratcheting updates.
  - immutability checking with all variants of `x-kubernetes-mutability`, `x-kubernetes-key-mutability`, and all variations of `x-kubernetes-list-type` and `x-kubernetes-map-type`.
- integration tests are added for
  - creation, updates, patches and server-side-apply of partially immutable CRs 
  - interaction of server-side-apply list-types, map-types and immutability
  - OpenAPI publishing of the vendor extensions
  - CRD updates of the immutability extensions and that the new immutability
    schemas are followed.
  - rejection of `x-kubernetes-mutability` and `x-kubernetes-key-mutability` for non-v1 CRDs on creation, ratcheting on update.
- e2e and conformance tests that
  - immutability is followed during updates, patches and server-side-apply. 
   
### Graduation Criteria

Because we must be able to downgrade from 1.17 to 1.16 without losing data, immutability must be introduced as alpha first.

For alpha:

- behaviour for all cases is implemented
- `v1` validation and ratcheting updates for `v1beta1` is implemented.
- root-level and metadata validation is implemented.
- restrictions of `x-kubernetes-mutability` and `x-kubernetes-key-mutability` values is implemented.
- integration tests exist with good coverage. 

For beta:

- API fields roundtrip through the previous version during downgrade. 
- normalization KEP (or something comparable) is merged.
- performance does not suffer for CRDs **which do not use** immutability vendor extensions.
- unit and integration tests are exhaustive for all list and map types.

For GA:

- performance is benchmarked with an upper bound overhead of 15% on CRDs with schemas.
- conformance tests are implement.

## Implementation History

N/A

## Alternative Considered

- OpenAPI has a notion of `readOnly`. This is meant to restrict fields to be set
  only in responses, not in a request payload. This does not match our 
  `never-change-after-creation` semantics.
- Allowing `x-kubernetes-mutability: Mutable` was considered to disable immutability imposed by a parent node. With it we could express `x-kubernetes-key-mutability: Immutable` as:
  ```yaml
  type: object
  properties:
    foo:
      type: object
      x-kubernetes-mutability: Immutable
      additionalProperties:
        type: string
        x-kubernetes-mutability: Mutable
  ```
  with the small difference that here these would be disallowed:
  - `{}` → `{foo: {}}`
  - `{foo: {}}` → `{}`.
  This API does not extend cleanly to  `AddOnly | RemoveOnly`:
  ```yaml
  type: object
  properties:
    foo:
      type: object
      x-kubernetes-mutability: AddOnly
      additionalProperties:
        type: string
        x-kubernetes-mutability: Mutable
  ```
  Does the `AddOnly` apply to the keys of the map or to the map itself as field `foo` in the parent object?
  
  If it `x-kubernetes-mutability: AddOnly | RemoveOnly` applies to both, we would get:
  
  Allowed:
  - `{foo:{a:"1"}` → `{foo:{a:"2"}}` (keys are unchanged)
  
  Allowed for AddOnly, disallowed otherwise:
  - `{}` → `{foo:{}}` (`foo` is added)
  - `{}` → `{foo:{a:"1"}}` (key a is added)
  - `{foo:{a:"1"}}` → `{foo:{a:"1", b:"2"}}` (key b is added)
  
  Allowed for RemoveOnly, disallowed otherwise:
  - `{foo:{}}` → `{}` (`foo` is removed)
  - `{foo:{a:"1"}}` → `{}` (key a is removed)
  
  Disallowed:
  - `{foo:{a:"1"}}` → `{foo:{b:"1"}}` (key a is removed, key b is added)
  
  **While this looks acceptable, there are reasons that we do not want to go with `x-kubernetes-mutability: Mutable`:**
  
  1. `x-kubernetes-mutability: Mutable` is a negative property which makes the schema weaker instead of stronger. With structural schemas we tried hard to avoid non-positive properties because it is hard to reason about: missing knowledge about that property leads to an inconsistent schema which forbids valid requests.
  
  2. A practical consequence of (1) is that of non-local, unexpected behaviour for shared types: imagine a `PodSpec` that marks certain fields mutable. If that type is embedded into an immutable object, that immutability is not applied to the whole `PodSpec` because it opts out for certain fields. This is counter-intuitive.
  
  As a fix for (2) one could ignore `x-kubernetes-mutability: Mutable` under more than one parent with `x-kubernetes-mutability: Immutable`. While this looks like a reasonable behaviour for the "shared type case", it makes the semantics complex.