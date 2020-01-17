---
title: Defaulting for Custom Resources
authors:
  - "@sttts"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-api-machinery
reviewers:
  - "@deads2k"
  - "@lavalamp"
  - "@liggitt"
  - "@mbohlool"
  - "@apelisse"
approvers:
  - "@deads2k"
  - "@lavalamp"
editor: "@sttts"
creation-date: 2019-04-26
last-updated: 2019-07-29
status: implemented
see-also:
  - "/keps/sig-api-machinery/20180731-crd-pruning.md"
  - "/keps/sig-api-machinery/20190425-structural-openapi.md"
---

# Defaulting for Custom Resources

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Validation](#validation)
  - [Behaviour of Embedded Resource ObjectMeta Defaults](#behaviour-of-embedded-resource-objectmeta-defaults)
  - [Examples](#examples)
- [References](#references)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Alternatives considered](#alternatives-considered)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Defaulting is a fundamental step in the processing of API objects in the request pipeline of the kube-apiserver. Defaulting happens during deserialization, i.e. after decoding of a versioned object, **but before** conversion to a hub type.

Defaulting is implemented for most native Kubernetes API types and plays a crucial role for API compatibility when adding new fields. CustomResources do not support this natively. 

Mutating admission webhooks can be used to partially replicate defaulting for incoming request payloads. But mutating admission webhooks do not run when reading from etcd.

This KEP is about adding support for specifying default values for fields via OpenAPI v3 validation schemas in the CRD manifest. OpenAPI v3 has native support for a `default` field with arbitrary JSON values, for example: 

```yaml
properties:
  foo:
    type: string
    default: "abc" 
```

This KEP proposes to apply these default values during deserialization, in the same way as native resources do. We assume _structural schemas_ as defined in [KEP Vanilla OpenAPI Subset: Structural Schema](/keps/sig-api-machinery/20190425-structural-openapi.md).

This feature starts behind a feature gate `CustomResourceDefaulting`, disabled by default as alpha in 1.15.

It will graduate to GA in `apiextensions.k8s.io/v1` for 1.16. Defaults can only be set on creation via the v1 API.

## Motivation

* By far most native Golang based resources implement defaulting. CRDs do not allow that, leading to unnatural, not Kubernetes-like APIs. This is bad for the ecosystem.
* Mutating Admission Webhooks can be used for defaulting, but:
  * they are not adequate as it is not possible to set default values of newly added fields on GET because admission is not run in the storage layer when reading from etcd.
  * webhooks are complex, both from the development/maintenance point of view and due to their non-trivial operational overhead. This makes them a no-go for many "not so ambitiously, professionally developed CRDs", e.g. in in-house enterprise environments.
* _Structural schemas_ as defined in [KEP Vanilla OpenAPI Subset: Structural Schema](/keps/sig-api-machinery/20190425-structural-openapi.md) make defaulting a low hanging fruit.
 
### Goals

* add CustomResource defaulting at the correct position in the request pipeline 
* allow definition of defaults via the OpenAPI v3 `default` field.

### Non-Goals

* allow non-constant defaults: native Golang code can of course set defaults which depends on other fields of a JSON object. This is out of scope here and would need some kind of defaulting webhook.
* native-type declarative defaulting: this KEP is about CRDs. Though it might be desirable to support the same `// +default=<some-json-value>` tag and a mechanism in `k8s.io/apiserver` to evaluate defaults in native, generic registries, this is out-of-scope of the KEP though.

## Proposal

We assume the CRD has _structural schemas_ (as defined in [KEP Vanilla OpenAPI Subset: Structural Schema](/keps/sig-api-machinery/20190425-structural-openapi.md)).

We propose to

1. derive the value-validation-less variant of the structural schema (trivial by definition of _structural schema_) and 
2. recursively follow the given CustomResource instance and the structural schema, applying specified defaults where an object field is undefined (`if _, ok := obj[field]; !ok` => default).

This means that we do not default 

* explicit JSON `null` values (it might be rejected during validation depending on the `nullable` setting)
* nor empty lists or maps
* nor zero numbers/integers or empty strings (`omitempty` during marshalling indirectly controls whether these values are defaulted or not)

This corresponds to the [official OpenAPI v3 `default` semantics](https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md#properties).

We do defaulting in the serializer by passing a real defaulter to [`versioningserializer.NewCodec`](https://github.com/kubernetes/apimachinery/blob/master/pkg/runtime/serializer/versioning/versioning.go#L49) such that defaulting is done natively just after the binary payload has been unmarshalled into an `map[string]interface{}` and pruning of [KEP: Pruning for CustomResources](/keps/sig-api-machinery/20180731-crd-pruning.md) was done.

Like for native resources, we do defaulting

* during request payload deserialization
* after mutating webhook admission
* during read from storage.
     
Note: like for native resources, we do not default after webhook conversions. Hence, webhook conversions should be complete in the sense that they return defaulted objects in order for the API user to see defaulted objects. Technically we could do additional defaulting, but to match native resources, we do not.

Compare the yellow boxes in the following figure:

![Decoding steps which must apply defaults](20190426-crd-defaulting-pipeline.png)

We rely on the validation steps in the request pipeline to verify that the default value validates value validation. We will check the types in default values using the _structural schema_ during CRD creation and update though. We will also reject defaults which contain values which will be pruned.

Defaulting happens top-down, i.e. we apply defaults for an object first, then dive into the fields (including, the new one).

The `default` field in the CRD types is considered alpha quality. We will add a `CustomResourceDefaulting` feature gate. Values for `default` will be rejected if the gate is not enabled and there have not been `default` values set before (ratcheting validation). 

[Kubebuilder's crd-gen](https://github.com/kubernetes-sigs/controller-tools/tree/master/pkg/crd) can make use of this feature by adding another tag, e.g. `// +default=<arbitrary-json-value>`. Defaults are arbitrary JSON values, which must also validate (types are checked during CRD creation and update, value validation is checked for requests, but not for etcd reads) and are not subject to pruning (defaulting happens after pruning).

### Validation

CRDs with defaults can only be created via `apiextensions.k8s.io/v1`. They are rejected for `v1beta1` on creation (updates keep working). 

Default values must be pruned, i.e. must not have fields which are not specified by the given structural schema (including support for `x-kubernetes-preserve-unknown-fields` to exclude nodes from being pruned), with the exception of
 
* defaults inside of `.metadata` of an embedded resource (`x-kubernetes-embedded-resource: true`)
* defaults which span `.metadata` of an embedded resource.

Values conflicting with this are rejected when creating or updating a CRD. 

Defaults are validated against the schema (including embedded `ObjectMeta` validation).

Defaults inside `.metadata` at the root are not allowed.

### Behaviour of Embedded Resource ObjectMeta Defaults

While defaults for `ObjectMeta` fields are not checked for being pruned during CRD creation and update (previous section), we do pruning of default values on CR storage creation. That way no user-provided default values are lost, but the pruning step during storage creation ensures that no unknown or invalid defaulted values are persisted during CR creation or update.

### Examples

1. default in the undefined case

   ```yaml
   type: object
   properties:
     foo:
       type: string
       default: "abc"
   ```

   Then
   
   ```json
   {}
   ```
   
   is defaulted to:
   
   ```json
   {
     "foo": "abc"
   }
   ```
   
2. no defaulting

   ```yaml
   type: object
   properties:
     foo:
       type: string
       default: "abc"
   ```

   Then
   
   ```json
   {
     "foo": "def"
   }
   ```
   
   is defaulted to:
   
   ```json
   {
     "foo": "def"
   }
   ```
   
3. array default in the undefined case

   ```yaml
   type: object
   properties:
     foo:
       type: array
       items:
         type: integer
       default: [1]
   ```

   Then
   
   ```json
   {}
   ```
   
   is defaulted to:
   
   ```json
   {
     "foo": [1]
   }
   ```
   
   In contrast
   
   ```json
   {
     "foo": null  
   }
   ```
   
   is defaulting to
   
   ```json
   {
     "foo": null  
   }
   ```
   
   and then rejected by validation because `foo` has no `nullable: true`.
   
   Similarly, empty lists stay empty lists:
   
    ```json
    {
      "foo": [] 
    }
    ```
         
    is defaulted to:
         
    ```json
    {
       "foo": []
    }
    ```
   
4. top-down defaulting
      
   ```yaml
   type: object
   properties:
     foo:
       type: object
       properties:
         a:
           type: string
           default: "abc"
         b:
           type: string
       default: {"b":"def"}
   ```

   Then
   
   ```json
   {}
   ```
   
   is defaulted to:
   
   ```json
   {
     "foo": {"a":"abc", "b":"def"}
   }
   ```

## References

* Old pruning implementation PR https://github.com/kubernetes/kubernetes/pull/64558, to be adapted. With structural schemas it will become much simpler.
* [OpenAPI v3 specification](https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md)
* [JSON Schema](http://json-schema.org/)

### Test Plan

**blockers for alpha:**

* we add unit tests for the general defaulting algorithm
* we add apiextensions-apiserver integration tests to
  * verify that CRDs with default, but without structural schemas are rejected by validation.
  * verify that CRDs without defaults keep working (probably nothing new needed)
  * verify that CRDs with defaults are defaulted if the feature gate is enabled:
    * during request payload deserialization
    * during mutating webhook admission
    * during read from storage
  * verify with feature gate closed that CRDs with defaults are rejected on create and on updating for the first default value, but accepted if defaults existed before (ratcheting validation).

**blockers for beta:**

* add tests for default value validation:
  * CRD schema with defaults containing unknown fields inside metadata
      * new CRD (allowed)
      * updated CRD where existing CRD did contain the unknown field (allowed)
      * updated CRD where existing CRD did not contain the unknown field (allowed)
      * building CR storage from persisted CRD discards unknown fields
  * CRD schema with defaults containing schema-invalid fields inside metadata (e.g. finalizers: 1)
    * new CRD (forbidden)
    * updated CRD where existing CRD did contain the invalid field (allowed)
    * updated CRD where existing CRD did not contain the invalid field (forbidden)
    * building CR storage from persisted CRD discards schema invalid fields from defaults.
  * CRD schema with defaults containing unknown fields outside of metadata is
    * accepted if they fall under the scope of `x-kubernetes-preserve-unknown-fields: true`
    * rejected otherwise.

* we are happy with the API and its behaviour

**blockers for GA:**
* add tests for default value validation:
  * CRD schema with defaults is rejected on creation via the `v1beta1` endpoints, but updates via `v1beta1` are still possible.
* we verified that performance of defaulting is adequate and not considerably reducing throughput. As a rule of thumb, we expect defaulting to be considerably faster than a deep copy.

### Graduation Criteria

* the test plan is fully implemented for the respective quality level

### Upgrade / Downgrade Strategy

* defaults cannot be set in 1.14 CRDs.
* CRDs created in 1.15 will keep defaults when downgrading to 1.14 (because we have them in our `v1beta1` types already). They won't be effective and the CRD will not validate anymore. This is acceptable for an alpha feature.
* CRDs created in 1.15 with the feature gate enabled will keep working the same way when upgrading to 1.16, or conversely during downgrade from 1.16 to 1.15 as we do ratcheting validation.
* Creation of CRDs with defaults via `v1beta1` API will be disabled, even if the feature gate is explicitly enabled.

### Version Skew Strategy

* kubectl is not aware of defaulting in any relevant way. 

## Alternatives considered

* we considered following the behaviour of native resources regarding `null` values and empty lists/maps and zero values for string, number and integer, i.e. this semantics:

  For atomic types:
     
  * `if v, ok := obj[fld]; !ok` => default
  * `else if !nullable && v == nil` => default
     
  and for `array` type in the schema one of these:
     
  * `if v, ok := obj[fld]; !ok` => default
  * `else if !nullable && v == nil` => default
  * `else if array, ok := v.([]interface{}); !ok` => return deserialization error
  * `else if !nullable && len(array) == 0` => default
    
  and for `object` type in the schema:
    
  * `if v, ok := obj[fld]; !ok` => default
  * `else if !nullable && v == nil` => default
  * `else if object, ok := v.(map[string]interface{}); !ok` => return deserialization error
  * `else if !nullable && len(object) == 0` => default.

  We decided against that because
  
  1. the native-type defaulting semantics inherited
  
     * the Golang unmarshalling behaviour which identifies undefined and `null` values (if one does not use additional pointer types to fight against it)
     * the Protobuf inability to distinguish undefined and empty lists and maps.
     
     For CRDs we can distinguish undefined and `null`. We control Protobuf for CRDs, when we add support in the future, and hence can use [some kind of value packaging](https://github.com/protocolbuffers/protobuf/issues/1606#issuecomment-281832148) to represent undefined lists/maps and other values faithfully.
     
     This way we avoid cargo culting an accidental API convention.
     
  2. the semantics of "default if undefined" is much simpler than any variant with (potentially conditional) defaulting of `null`  and empty slices/maps.
         
  3. it conflicts with [official OpenAPI v3 semantics for `default`](https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.0.md#properties)
  
     If we need more native-type like defaulting in addition, we can add an alternative
      
     ```yaml
     x-kubernetes-legacy-default: <arbitrary-json>
     ``` 

     which is mutual exclusive with `default`. Using another field name avoids this conflict, but gives us the legacy behaviour of defaulting `null` and empty lists/maps.
     
     This might potentially be relevant when embedding upstream Kubernetes types into CRDs.

## Implementation History
