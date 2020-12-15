---
title: CustomResourceDefinition Conversion Webhook
authors:
  - "@mbohlool"
  - "@erictune"
owning-sig: sig-api-machinery
participating-sigs:
reviewers:
  - "@lavalamp"
  - "@deads2k"
  - "@sttts"
  - "@liggitt"
approvers:
  - "@lavalamp"
  - "@deads2k"
creation-date: 2019-04-25
last-updated: 2019-04-25
status: implemented
replaces:
  - "(https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/customresource-conversion-webhook.md)"
---

# CustomResourceDefinition Conversion Webhook

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Definitions](#definitions)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Detailed Design](#detailed-design)
  - [CRD API Changes](#crd-api-changes)
  - [Top level fields to Per-Version fields](#top-level-fields-to-per-version-fields)
  - [Support Level](#support-level)
  - [Rollback](#rollback)
  - [Webhook Request/Response](#webhook-requestresponse)
  - [Metadata](#metadata)
  - [Monitorability](#monitorability)
  - [Error Messages](#error-messages)
  - [Caching](#caching)
- [Examples](#examples)
  - [Example of Writing Conversion Webhook](#example-of-writing-conversion-webhook)
- [Example of Updating CRD from one to two versions](#example-of-updating-crd-from-one-to-two-versions)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary


This document proposes a detailed plan for adding support for version-conversion
of Kubernetes resources defined via Custom Resource Definitions (CRD).  The API
Server is extended to call out to a webhook at appropriate parts of the handler
stack for CRDs.

No new resources are added; the 
[CRD resource](https://github.com/kubernetes/kubernetes/blob/34383aa0a49ab916d74ea897cebc79ce0acfc9dd/staging/src/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/types.go#L187)
is extended to include conversion information as well as multiple schema
definitions, one for each apiVersion that is to be served.

## Definitions

**Webhook Resource**: a Kubernetes resource (or portion of a resource) that informs the API Server that it should call out to a Webhook Host for certain operations. 

**Webhook Host**: a process / binary which accepts HTTP connections, intended to be called by the Kubernetes API Server as part of a Webhook.

**Webhook**: In Kubernetes, refers to the idea of having the API server make an HTTP request to another service at a point in its request processing stack.  Examples are [Authentication webhooks](https://kubernetes.io/docs/reference/access-authn-authz/webhook/) and [Admission Webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/).  Usually refers to the system of Webhook Host and Webhook Resource together, but occasionally used to mean just Host or just Resource.

**Conversion Webhook**: Webhook that can convert an object from one version to another.

**Custom Resource**: In the context of this document, it refers to resources defined as Custom Resource Definition (in contrast with extension API server’s resources).

**CRD Package**: CRD definition, plus associated controller deployment, RBAC roles, etc, which is released by a developer who uses CRDs to create new APIs.

## Motivation

Version conversion is, in our experience, the most requested improvement to CRDs.  Prospective CRD users want to be certain they can evolve their API before they start down the path of developing a CRD + controller. 

### Goals

* As an existing author of a CRD, I can update my API's schema, without breaking existing clients.  To that end, I can write a CRD(s) that supports one kind with two (or more) versions.  Users of this API can access an object via either version (v1 or v2), and are accessing the same underlying storage (assuming that I have properly defined how to convert between v1 and v2.)

* As a prospective user of CRDs, I don't know what schema changes I may need in the future, but I want to know that they will be possible before I chose CRDs (over EAS, or over a non-Kubernetes API).

* As an author of a CRD Package, my users can upgrade to a new version of my package, and can downgrade to a prior version of my package (assuming that they follow proper upgrade and downgrade procedures; these should not require direct etcd access.)

* As a user, I should be able to request CR in any supported version defined by CRD and get an object has been properly converted to the requested version (assuming the CRD Package Author has properly defined how to convert).

* As an author of a CRD that does not use validation, I can still have different versions which undergo conversion.

* As a user, when I request an object, and webhook-conversion fails, I get an error message that helps me understand the problem.

* As an API machinery code maintainer, this change should not make the API machinery code harder to maintain

* As a cluster owner, when I upgrade to the version of Kubernetes that supports CRD multiple versions, but I don't use the new feature, my existing CRDs work fine.  I can roll back to the previous version without any special action.

### Non-Goals

## Proposal

1. A CRD object now represents a group/kind with one or more versions.

2. The CRD API (CustomResourceDefinitionSpec) is extended as follows:

    1. It has a place to register 1 webhook.

    2. it holds multiple "versions".

    3. Some fields which were part of the .spec are now per-version; namely Schema, Subresources, and AdditionalPrinterColumns.

3. A Webhook Host is used to do conversion for a CRD.

    4. CRD authors will need to write a Webhook Host that accepts any version and returns any version.

    5. Toolkits like kube-builder and operator-sdk are expected to provide flows to assist users to generate Webhook Hosts.

## Detailed Design

### CRD API Changes

The CustomResourceDefinitionSpec is extended to have a new section where webhooks are defined: 

```golang
// CustomResourceDefinitionSpec describes how a user wants their resource to appear
type CustomResourceDefinitionSpec struct {
  Group string
  Version string
  Names CustomResourceDefinitionNames
  Scope ResourceScope
  // Optional, can only be provided if per-version schema is not provided.
  Validation *CustomResourceValidation
  // Optional, can only be provided if per-version subresource is not provided.
  Subresources *CustomResourceSubresources
  Versions []CustomResourceDefinitionVersion
  // Optional, can only be provided if per-version additionalPrinterColumns is not provided.  
  AdditionalPrinterColumns []CustomResourceColumnDefinition

  Conversion *CustomResourceConversion
}

type CustomResourceDefinitionVersion struct {
  Name string
  Served Boolean
  Storage Boolean
  // Optional, can only be provided if top level validation is not provided.
  Schema *JSONSchemaProp
  // Optional, can only be provided if top level subresource is not provided.
  Subresources *CustomResourceSubresources
  // Optional, can only be provided if top level additionalPrinterColumns is not provided.  
  AdditionalPrinterColumns []CustomResourceColumnDefinition
}

Type CustomResourceConversion struct {
  // Conversion strategy, either "nop" or "webhook". If webhook is set, Webhook field is required.
  Strategy string

  // Additional information for external conversion if strategy is set to external
  // +optional
  Webhook *CustomResourceConversionWebhook
}

type CustomResourceConversionWebhook {
  // ClientConfig defines how to communicate with the webhook. This is the same config used for validating/mutating webhooks.
  ClientConfig WebhookClientConfig
}
```

### Top level fields to Per-Version fields

In *CRD v1beta1* (apiextensions.k8s.io/v1beta1) there are per-version schema, additionalPrinterColumns or subresources (called X in this section) defined and these validation rules will be applied to them:

* Either top level X or per-version X can be set, but not both. This rule applies to individual X’s not the whole set. E.g. top level schema can be set while per-version subresources are set.
* per-version X cannot be the same. E.g. if all per-version schema are the same, the CRD object will be rejected with an error message asking the user to use the top level schema.

In *CRD v1* (apiextensions.k8s.io/v1), there will be only version list with no top level X. The second validation guarantees a clean moving to v1. These are conversion rules:

*v1beta1->v1:*

* If top level X is set in v1beta1, then it will be copied to all versions in v1.
* If per-version X are set in v1beta1, then they will be used for per-version X in v1.

*v1->v1beta1:*

* If all per-version X are the same in v1, they will be copied to top level X in v1beta1
* Otherwise, they will be used as per-version X in v1beta1

### Support Level

The feature will be alpha in the first implementation and will have a feature gate that is defaulted to false. The roll-back story with a feature gate is much more clear. if we have the features as alpha in kubernetes release Y (>X where the feature is missing) and we make it beta in kubernetes release Z, it is not safe to use the feature and downgrade from Y to X but the feature is alpha in Y which is fine. It is safe to downgrade from Z to Y (given that we enable the feature gate in Y) and that is desirable as the feature is beta in Z.
On downgrading from a Z to Y, stored CRDs can have per-version fields set. While the feature gate can be off on Y (alpha cluster), it is dangerous to disable per-version Schema Validation or Status subresources as it makes the status field mutable and validation on CRs will be disabled. Thus the feature gate in Y only protects adding per-version fields not the actual behaviour. Thus if the feature gate is off in Y:

* Per-version X cannot be set on CRD create (per-version fields are auto-cleared).
* Per-version X can only be set/changed on CRD update *if* the existing CRD object already has per-version X set.

This way even if we downgrade from Z to Y, per-version validations and subresources will be honored. This will not be the case for webhook conversion itself. The feature gate will also protect the implementation of webhook conversion and alpha cluster with disabled feature gate will return error for CRDs with webhook conversion (that are created with a future version of the cluster).

### Rollback

Users that need to rollback to version X (but may currently be running version Y > X) of apiserver should not use CRD Webhook Conversion if X is not a version that supports these features.  If a user were to create a CRD that uses CRD Webhook Conversion and then rolls back to version X that does not support conversion then the following would happen:

1. The stored custom resources in etcd will not be deleted.

2. Any clients that try to get the custom resources will get a 500 (internal server error). this is distinguishable from a deleted object for get and the list operation will also fail. That means the CRD is not served at all and Clients that try to garbage collect related resources to missing CRs should be aware of this. 

3. Any client (e.g. controller) that tries to list the resource (in preparation for watching it) will get a 500 (this is distinguishable from an empty list or a 404).

4. If the user rolls forward again, then custom resources will be served again.

If a user does not use the webhook feature but uses the versioned schema, additionalPrinterColumns, and/or subresources and rollback to a version that does not support them per-version, any value set per-version will be ignored and only values in top level spec.* will be honor.

Please note that any of the fields added in this design that is not supported in previous kubernetes releases can be removed on an update operation (e.g. status update). The kubernetes release where defined the types but gate them with an alpha feature gate, however, can keep these fields but ignore there value.

### Webhook Request/Response

The Conversion request and response would be similar to [Admission webhooks](https://github.com/kubernetes/kubernetes/blob/951962512b9cfe15b25e9c715a5f33f088854f97/staging/src/k8s.io/api/admission/v1beta1/types.go#L29). The AdmissionReview seems to be redundant but used by other Webhook APIs and added here for consistency.

```golang
// ConversionReview describes a conversion request/response.
type ConversionReview struct {
  metav1.TypeMeta
  // Request describes the attributes for the conversion request.
  // +optional
  Request *ConversionRequest
  // Response describes the attributes for the conversion response.
  // +optional
  Response *ConversionResponse
}

type ConversionRequest struct {
  // UID is an identifier for the individual request/response. Useful for logging.
  UID types.UID
  // The version to convert given object to. E.g. "stable.example.com/v1"
  APIVersion string
  // Object is the CRD object to be converted.
  Object runtime.RawExtension
}

type ConversionResponse struct {
  // UID is an identifier for the individual request/response.
  // This should be copied over from the corresponding ConversionRequest.
  UID types.UID
  // ConvertedObject is the converted version of request.Object.
  ConvertedObject runtime.RawExtension
}
```

If the conversion is failed, the webhook should fail the HTTP request with a proper error code and message that will be used to create a status error for the original API caller.

### Metadata

To ensure consistent behaviour of converted CRDs following the established API machinery semantics, we will

* check that the returned list of converted objects has the same length and each object (at each index) has the same kind, name, namespace and UID as in the request object list (which ensures the same order), and fail the conversion otherwise
* restrict which fields of `ObjectMeta` a conversion webhook can change to labels and annotations.

  Mutations to other fields under `metadata` are ignored (i.e. restored to the values of the original object). This means that mutations to other fields also do not lead to an error (we do this because it is hard to define what a change to `ObjectMeta` actually is, with the known encoding issues of empty and undefined lists and maps in mind).
  
  Labels and annotations are validated with the usual API machinery ObjectMeta validation semantics.

### Monitorability

There should be prometheus variables to show:

* CRD conversion latency
    * Overall
    * By webhook name
    * By request (sum of all conversions in a request)
    * By CRD
* Conversion Failures count
    * Overall
    * By webhook name
    * By CRD
* Timeout failures count
    * Overall
    * By webhook name
    * By CRD

Adding a webhook dynamically adds a key to a map-valued prometheus metric. Webhook host process authors should consider how to make their webhook host monitorable: while eventually we hope to offer a set of best practices around this, for the initial release we won’t have requirements here.


### Error Messages

When a conversion webhook fails, e.g. for the GET operation, then the error message from the apiserver to its client should reflect that conversion failed and include additional information to help debug the problem. The error message and HTTP error code returned by the webhook should be included in the error message API server returns to the user.  For example:

```bash
$ kubectl get mykind somename
error on server: conversion from stored version v1 to requested version v2 for somename: "408 request timeout" while calling service "mywebhookhost.somens.cluster.local:443"
```


For operations that need more than one conversion (e.g. LIST), no partial result will be returned. Instead the whole operation will fail the same way with detailed error messages. To help debugging these kind of operations, the UID of the first failing conversion will also be included in the error message. 


### Caching

No new caching is planned as part of this work, but the API Server may in the future cache webhook POST responses.

Most API operations are reads.  The most common kind of read is a watch.  All watched objects are cached in memory. For CRDs, the cache
is per-version. That is the result of having one [REST store object](https://github.com/kubernetes/kubernetes/blob/3cb771a8662ae7d1f79580e0ea9861fd6ab4ecc0/staging/src/k8s.io/apiextensions-apiserver/pkg/registry/customresource/etcd.go#L72) per-version which
was an arbitrary design choice but would be required for better caching with webhook conversion. In this model, each GVK is cached, regardless of whether some GVKs share storage.  Thus, watches do not cause conversion.  So, conversion webhooks will not add overhead to the watch path.  Watch cache is per api server and eventually consistent.

Non-watch reads are also cached (if requested resourceVersion is 0 which is true for generated informers by default, but not for calls like `kubectl get ...`, namespace cleanup, etc). The cached objects are converted and per-version (TODO: fact check). So, conversion webhooks will not add overhead here too.

If in the future this proves to be a performance problem, we might need to add caching later.  The Authorization and Authentication webhooks already use a simple scheme with APIserver-side caching and a single TTL for expiration.  This has worked fine, so we can repeat this process.  It does not require Webhook hosts to be aware of the caching.


## Examples


### Example of Writing Conversion Webhook

Data model for v1:

|data model for v1|
|-----------------|
```yaml      
properties:
  spec:
    properties:
      cronSpec:
        type: string
      image:
        type: string
```

|data model for v2|
|-----------------|
```yaml
properties:
  spec:
    properties:
      min:
        type: string
      hour:
        type: string
      dayOfMonth:
        type: string
      month:
        type: string
      dayOfWeek:
        type: string
      image:
        type: string
```


Both schemas can hold the same data (assuming the string format for V1 was a valid format).

|crontab_conversion.go|
|---------------------|

```golang
import .../types/v1
import .../types/v2

// Actual conversion methods

func convertCronV1toV2(cronV1 *v1.Crontab) (*v2.Crontab, error) {
  items := strings.Split(cronV1.spec.cronSpec, " ")
  if len(items) != 5 {
     return nil, fmt.Errorf("invalid spec string, needs five parts: %s", cronV1.spec.cronSpec)
  }
  return &v2.Crontab{
     ObjectMeta: cronV1.ObjectMeta,
     TypeMeta: metav1.TypeMeta{
        APIVersion: "stable.example.com/v2",
        Kind: cronV1.Kind,
     },
     spec: v2.CrontabSpec{
        image: cronV1.spec.image,
        min: items[0],
        hour: items[1],
        dayOfMonth: items[2],
        month: items[3],
        dayOfWeek: items[4],
     },
  }, nil

}

func convertCronV2toV1(cronV2 *v2.Crontab) (*v1.Crontab, error) {
  cronspec := cronV2.spec.min + " "
  cronspec += cronV2.spec.hour + " "
  cronspec += cronV2.spec.dayOfMonth + " "
  cronspec += cronV2.spec.month + " "
  cronspec += cronV2.spec.dayOfWeek
  return &v1.Crontab{
     ObjectMeta: cronV2.ObjectMeta,
     TypeMeta: metav1.TypeMeta{
        APIVersion: "stable.example.com/v1",
        Kind: cronV2.Kind,
     },
     spec: v1.CrontabSpec{
        image: cronV2.spec.image,
        cronSpec: cronspec,
     },
  }, nil
}

// The rest of the file can go into an auto generated framework

func serveCronTabConversion(w http.ResponseWriter, r *http.Request) {
  request, err := readConversionRequest(r)
  if err != nil {
     reportError(w, err)
  }
  response := ConversionResponse{}
  response.UID = request.UID
  converted, err := convert(request.Object, request.APIVersion)
  if err != nil {
     reportError(w, err)
  }
  response.ConvertedObject = *converted
  writeConversionResponse(w, response)
}

func convert(in runtime.RawExtension, version string) (*runtime.RawExtension, error) {
  inApiVersion, err := extractAPIVersion(in)
  if err != nil {
     return nil, err
  }
  switch inApiVersion {
  case "stable.example.com/v1":
     var cronV1 v1Crontab
     if err := json.Unmarshal(in.Raw, &cronV1); err != nil {
        return nil, err
     }
     switch version {
     case "stable.example.com/v1":
        // This should not happened as API server will not call the webhook in this case
        return &in, nil
     case "stable.example.com/v2":
        cronV2, err := convertCronV1toV2(&cronV1)
        if err != nil {
           return nil, err
        }
        raw, err := json.Marshal(cronV2)
        if err != nil {
           return nil, err
        }
        return &runtime.RawExtension{Raw: raw}, nil
     }
  case "stable.example.com/v2":
     var cronV2 v2Crontab
     if err := json.Unmarshal(in.Raw, &cronV2); err != nil {
        return nil, err
     }
     switch version {
     case "stable.example.com/v2":
        // This should not happened as API server will not call the webhook in this case
        return &in, nil
     case "stable.example.com/v1":
        cronV1, err := convertCronV2toV1(&cronV2)
        if err != nil {
           return nil, err
        }
        raw, err := json.Marshal(cronV1)
        if err != nil {
           return nil, err
        }
        return &runtime.RawExtension{Raw: raw}, nil
     }
  default:
     return nil, fmt.Errorf("invalid conversion fromVersion requested: %s", inApiVersion)
  }
  return nil, fmt.Errorf("invalid conversion toVersion requested: %s", version)
}

func extractAPIVersion(in runtime.RawExtension) (string, error) {
  object := unstructured.Unstructured{}
  if err := object.UnmarshalJSON(in.Raw); err != nil {
     return "", err
  }
  return object.GetAPIVersion(), nil
}
```

Note: not all code is shown for running a web server.  

Note: some of this is boilerplate that we expect tools like Kubebuilder will handle for the user.

Also some appropriate tests, most importantly round trip test:

|crontab_conversion_test.go|
|-|

```golang
func TestRoundTripFromV1ToV2(t *testing.T) {
  testObj := v1.Crontab{
     ObjectMeta: metav1.ObjectMeta{
        Name: "my-new-cron-object",
     },
     TypeMeta: metav1.TypeMeta{
        APIVersion: "stable.example.com/v1",
        Kind: "CronTab",
     },
     spec: v1.CrontabSpec{
        image: "my-awesome-cron-image",
        cronSpec: "* * * * */5",
     },
  }
  testRoundTripFromV1(t, testObj)
}

func testRoundTripFromV1(t *testing.T, v1Object v1.CronTab) {
  v2Object, err := convertCronV1toV2(v1Object)
  if err != nil {
     t.Fatalf("failed to convert v1 crontab to v2: %v", err)
  }
  v1Object2, err := convertCronV2toV1(v2Object)
  if err != nil {
     t.Fatalf("failed to convert v2 crontab to v1: %v", err)
  }
  if !reflect.DeepEqual(v1Object, v1Object2) {
     t.Errorf("round tripping failed for v1 crontab. v1Object: %v, v2Object: %v, v1ObjectConverted: %v",
        v1Object, v2Object, v1Object2)
  }
}
```

## Example of Updating CRD from one to two versions 

This example uses some files from previous section.

**Step 1**: Start from a CRD with only one version  

|crd1.yaml|
|-|

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: crontabs.stable.example.com
spec:
  group: stable.example.com
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      properties:
        spec:
          properties:
            cronSpec:
              type: string
            image:
              type: string
  scope: Namespaced
  names:
    plural: crontabs
    singular: crontab
    kind: CronTab
    shortNames:
    - ct
```

And create it:

```bash
Kubectl create -f crd1.yaml
```

(If you have an existing CRD installed prior to the version of Kubernetes that supports the "versions" field, then you may need to move version field to a single item in the list of versions or just try to touch the CRD after upgrading to the new Kubernetes version which will result in the versions list being defaulted to a single item equal to the top level spec values)

**Step 2**: Create a CR within that one version:

|cr1.yaml|
|-|
```yaml

apiVersion: "stable.example.com/v1"
kind: CronTab
metadata:
  name: my-new-cron-object
spec:
  cronSpec: "* * * * */5"
  image: my-awesome-cron-image
```

And create it:

```bash
Kubectl create -f cr1.yaml
```

**Step 3**: Decide to introduce a new version of the API.

**Step 3a**: Write a new OpenAPI data model for the new version (see previous section).  Use of a data model is not required, but it is recommended.

**Step 3b**: Write conversion webhook and deploy it as a service named `crontab_conversion`

See the "crontab_conversion.go" file in the previous section.

**Step 3c**: Update the CRD to add the second version.

Do this by adding a new item to the "versions" list, containing the new data model:

|crd2.yaml|
|-|
```yaml

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: crontabs.stable.example.com
spec:
  group: stable.example.com
  versions:
  - name: v1
    served: true
    storage: false
    schema:
      properties:
        spec:
          properties:
            cronSpec:
              type: string
            image:
              type: string
  - name: v2
    served: true
    storage: true
    schema:
      properties:
        spec:
          properties:
            min:
              type: string
            hour:
              type: string
            dayOfMonth:
              type: string
            month:
              type: string
            dayOfWeek:
              type: string
            image:
              type: string
  scope: Namespaced
  names:
    plural: crontabs
    singular: crontab
    kind: CronTab
    shortNames:
    - ct
  conversion:
    strategy: external
    webhook:
      client_config:
        namespace: crontab
        service: crontab_conversion
        Path: /crontab_convert
```

And apply it:

```bash
Kubectl apply -f crd2.yaml
```

**Step 4**: add a new CR in v2:

|cr2.yaml|
|-|
```yaml

apiVersion: "stable.example.com/v2"
kind: CronTab
metadata:
  name: my-second-cron-object
spec:
  min: "*"
  hour: "*"
  day_of_month: "*"
  dayOfWeek: "*/5"
  month: "*"
  image: my-awesome-cron-image
```

And create it:

```bash
Kubectl create -f cr2.yaml
```

**Step 5**: storage now has two custom resources in two different versions. To downgrade to previous CRD, one can apply crd1.yaml but that will fail as the status.storedVersions has both v1 and v2 and those cannot be removed from the spec.versions list. To downgrade, first create a crd2-b.yaml file that sets v1 as storage version and apply it, then follow "*Upgrade existing objects to a new stored version*" in [this document](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definition-versioning/). After all CRs in the storage has v1 version, you can apply crd1.yaml.

**Step 5 alternative**: create a crd1-b.yaml that has v2 but not served.

|crd1-b.yaml|
|-|
```yaml

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: crontabs.stable.example.com
spec:
  group: stable.example.com
  versions:
  - name: v1
    served: true
    storage: true
    schema:
      properties:
        spec:
          properties:
            cronSpec:
              type: string
            image:
              type: string
  - name: v2
    served: false
    storage: false
  scope: Namespaced
  names:
    plural: crontabs
    singular: crontab
    kind: CronTab
    shortNames:
    - ct
  conversion:
    strategy: external
    webhook:
      client_config:
        namespace: crontab
        service: crontab_conversion
        Path: /crontab_convert
```

### Risks and Mitigations

What are the risks of this proposal and how do we mitigate.
Think broadly.
For example, consider both security and how this will impact the larger kubernetes ecosystem.

How will security be reviewed and by whom?
How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.

## Graduation Criteria

v1beta1:

- Finalize validation restrictions on what metadata fields conversion is able to mutate (https://github.com/kubernetes/kubernetes/issues/72160).
- Ensure the scenarios from (https://github.com/kubernetes/kubernetes/issues/64136) are tested:
  - Ensure what is persisted in etcd matches the storage version
  - Set up a CRD, persist some data, change the storage version, and ensure the previously persisted data is readable
  - Ensure discovery docs track a CRD through creation, version addition, version removal, and deletion
  - Ensure `spec.served` is respected
- Error-case handling when calling conversion results in an error during 
  - {get, list, create, update, patch, delete, deletecollection} calls to the primary resource
  - {get, update, patch} calls to the status subresource
  - {get, update, patch} calls to the scale subresource

v1:

- ConversionReview API v1 is stable (including fields that need to be updated for v1beta1 or v1)
- Documented step-by-step CRD version migration workflow that covers the entire
  process of migrating from one version to another (introduce a new CRD version,
  add the converter, migrate clients, migrate to new storage version, all the
  way to removing the old version)

## Implementation History

- Implemented in Kubernetes 1.13 release (https://github.com/kubernetes/kubernetes/pull/67006)

## Alternatives

First a defaulting approach is considered which per-version fields would be defaulted to top level fields. but that breaks backward incompatible change; Quoting from API [guidelines](/contributors/devel/sig-architecture/api_changes.md#backward-compatibility-gotchas):

> A single feature/property cannot be represented using multiple spec fields in the same API version simultaneously

Hence the defaulting either implicit or explicit has the potential to break backward compatibility as we have two sets of fields representing the same feature.

There are other solution considered that does not involved defaulting:

* Field Discriminator: Use `Spec.Conversion.Strategy` as discriminator to decide which set of fields to use. This approach would work but the proposed solution is keeping the mutual excusivity in a broader sense and is preferred.
* Per-version override: If a per-version X is specified, use it otherwise use the top level X if provided. While with careful validation and feature gating, this solution is also backward compatible, the overriding behaviour need to be kept in CRD v1 and that looks too complicated and not clean to keep for a v1 API.

Refer to [this document](http://bit.ly/k8s-crd-per-version-defaulting) for more details and discussions on those solutions.
