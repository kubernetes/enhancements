---
title: KEP Template
authors:
  - "@jennybuckley"
  - "@apelisse"
owning-sig: sig-api-machinery
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-06-21
last-updated: 2019-06-21
status: provisional
see-also:
  - "/keps/sig-api-machinery/0006-apply.md"
---

# Apply Scalability

## Table of Contents

- TODO

## Summary

Serverside Apply (SSA) in its current implementation causes some serious scalability issues when the feature is enabled. Specifically, list operation duration was highly dependent on the number of updates happening. From our testing it seemed to be because of a large number of small object allocations.

## Motivation

### The Problems

From our testing, the problems seem to be because of a large number of small object allocations.

#### On list, update, and apply: 

Deserializing managedFields as a deeply nested field in the apiserver allocates many small objects. Objects are deserialized when fetched from etcd.
The size of managedFields is similar to the size of the object. It increases etcd size, and cache size of the apiserver and clients.

#### On update and apply: 

Converting managedFields from the api representation into the internal representation causes many small object allocations.
Some structured-merge-diff operations (like compare) also create many small objects that are not needed. Most “field set” operations are not done in place.

### Experiments

The following experiments convinced us that the problem is mostly the allocations of small objects. Latency in etcd because of read/write with bigger objects, and/or data transfer was not a significant issue.

For all these scenarios, we’ve created a deployment with 5,000 pods. Pods that have the managedFields (when server-side apply is enabled) are obviously larger. Here are some numbers for the comparison, with the objects in protobuf format:

- Server-side apply disabled:
  - A pod is about 8,137 bytes
  - 5,000 pods is 40,506,271 bytes
- Server-side apply enabled:
  - A pod is about 13,164 bytes (~62% larger)
  - 5,000 pods is 67,182,690 bytes

All the following scenarios have been tested by LISTing all the pods, then running 250 patches per second while still LISTing the pods at the same time.

1. Server-side apply disabled, 
2. Server-side apply enabled, 
3. Server-side apply disabled, but with an extra annotation. The annotation is about 5,000 bytes to compensate for the difference in object size with server-side apply.
4. Server-side apply disabled, with 320 small annotations. Those annotations total about 5,000 bytes to simulate the small object allocation.

The average latency are as follows:

| Scenario | LIST | PATCH | LIST while PATCH |
|---|---|---|---|
| 1. |360.4 ms | 8.0 ms | 361.0 ms |
| 2. | 1294.5 ms | 38.5 ms | 2529.6 ms |
| 3. | 480.8 ms | 11.3 ms | 500.2 ms |
| 4. | 1260.1 ms | 9.7 ms | 1503.2 ms |

### Goals

TODO

### Non-Goals

TODO

## Proposal

We have considered a few overlapping solutions:

### Only send managedFields if requested:

Add a query parameter (or some other mechanism) to allow users to request to see the managedFields field in metadata. If the parameter is missing,  strip the field on reads and writes (in responses). The benefit here is we can skip serializing the field most of the time. Most clients won’t and shouldn’t have to understand how to read this field. So making them receive it is a waste of resources, and confusing.

### Store managedFields as a blob

Instead of a highly structured nested map, with lots of small fields, we could store it as a blob. The blob would save on allocation/GC of small objects when deserializing. We could also gzip the entire thing to save space. The downside is that it would make the field harder/impossible for a user to understand. This is mitigated by combining with solution 1, or by client-go/kubectl unzipping the field automatically.

### Convert managedFields directly to internal fieldpath

On updates and applies, the apiserver needs to convert managedFields into an internal representation. We originally decoupled the managedFields API from the internal representation of a fieldset, to allow arbitrary changes to the internal representation without breaking the API. However converting from a serialized config to a gostruct and then into a different struct causes a lot of allocations of small objects and stress on the garbage collector. Instead, we can directly serialize as the internal representation.

The internal format could be stored in protobuf, which would make it much smaller than json (json has a lot of repeated fields that make it extremely large).

### Only update managedFields if previously applied to

Most objects created with post or put are never applied to, so tracking managedFields on them is a waste of resources. If an object without managedFields does get applied to, we can still generate a single managedFields entry for all the existing fields. The generated managedFields conflicts set will be the same, but we won’t have any way of knowing know who is being conflicted with. CPU and memory cost on the apiserver will be dependent on the number of objects managed by server-side apply. It would increase more gradually as controllers decided to adopt SSA. Adoption could be decided on a controller by controller basis, weighing the cost vs. benefits each time. If someone applies to an object created by post or put, the conflicts they see would not contain information about the timestamp or field manager, so conflicts become less useful.

### Storing in one format, serving in another

Another possible solution would be to combine many of these solutions:

Some combination of storing the field managers as one big blob in etcd, using the internal fieldpath representation, as gzipped protobuf, would give us a much lower overhead in etcd and very fast serialization/deserialization. This is especially true if deserialization doesn’t need to know about the specifics of managedfields (hence just deserializing as a blob).

Serving the API as it currently is (or maybe changing it if we believe we need to change what we serve), but only on demand so that the price of deserializing would only be paid by clients that specifically request it.

Based on initial numbers, this solution has the most advantages:
etcd objects (based on pods above) only grow by 30%
listing objects has nearly zero cost since we don’t send unless requested,
patches and updates can deserialize the data very quickly directly in protobuf, and process the data
API is still “usable”

This could be implemented as a subresource similar to scale. (something like “/api/v1/pods/managedfields”) The subresource would decode the stored form of managedFields and exposes an object of type meta/v1.ManagedFields which would be highly structured. The form would be similar to or the same as how managedFields is stored today. This part could be implemented at a future date.

### Add a default specifier syntax to managed fields

Define a new syntax in manage fields, where exactly one manager can put: managedFields[].fields.allOtherFields:true

When an object is created by apply or post or put, with no other managers, then it looks like this:

```
metadata:
  managedFields:
  - manager: kubectl
    operation: Apply
    apiVersion: v1
    fields:
      default
```

Taking advantage of the fact that the field names are already listed in the object. This allows us to skip mentioning the field names in the managedFields.  This is the common case, and so there is minimal nesting and minimal small objects created.   In the case that something, such as an HPA in this example, changes the object, then it looks like this:

```
metadata:
  managedFields:
  - manager: kubectl
    operation: Apply
    apiVersion: v1
    fields:
      default
  - manager: hpa
    operation: Patch
    apiVersion: v1
    fields:
      f:spec:
        f:replicas:
```

This should improve performance if it is true that one manager owns a preponderance of the fields, and other managers own few.  This would need to be verified somehow.
