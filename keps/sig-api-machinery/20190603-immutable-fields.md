---
title: Immutable Fields
authors:
  - "@apelisse"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-api-machinery
reviewers:
approvers:
editor: TBD
creation-date: 2019-06-03
last-updated: 2019-06-03
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
currently no way to declaratively document that fields are immutable, and one
has to rely on either built-in validation for core types, or have to build a
validating webhooks for CRDs.

Providing a new `// +immutable` marker would make the API more descriptive to
users and help API developers by verifying these assertions automatically.

## Motivation

The kubernetes model tend to promote "immutable" objects rather than "delete and
recreate" on changes approach, and users have expressed the need for such a
feature through this issue for example:
https://github.com/kubernetes/kubernetes/issues/65973

### Goals

The goals are mostly twofold:
- Provide a mechanism to document APIs (both built-in types and CRDs) by having
  that additional marker provide information to users AND tools (including
  generated documentation, kubectl explain),
- Automatically provide logic for developers (of core-types and CRDs) to
  guarantee immutability of specific fields.

### Non-Goals

There are a few non-goals:
- OpenAPI has a notion of "readOnly", but this is only meant to simplify schema,
  and one is never supposed to send a readOnly value. This is different because
  we want "writeOnce", not "readOnly".

## Proposal

This proposal attempts to create a concept of immutable and to visit some of
what this can mean, but doesn't intend to propose all the possible
semantics. The proposal should not close the door to further improvements.

### Semantics

We'll define immutable as "writeOnce", which means that the fields can only be
set at creation time, and can never be updated after. Attempts to update an
immutable field will result in an error (as opposed to being ignored), though we
could potentially add that semantics later.

It's also important to note that this concept is orthogonal to field ownership
and list associativity. Ideally we would manage to keep the concepts entirely
separate. e.g. An atomic list is a different concept from an immutable list.

There are different semantics for a field to be immutable, especially if we
consider non-leaf fields like lists and maps.

#### Field selection

Scalar fields have an obvious "selection" mechanism: they are either immutable
or not. Things get more complicated for lists and maps, since we need to know if
the flag applies to the list/map itself, to its members, or both.

A few possible semantics are described here:

- Non-recursive: For a list or a map, one can not remove or add new items, but
  existing items can be modified.
- Recursive: means that none of their field can change, and new fields are not
  accepted.
- Recursive with addition and/or deletion:
  - New/deleted fields in maps and lists are tolerated.
  - The added/deleted type would probably be "Recursive" after the first level
    (addition/delete is not recursive).
  - For associative lists, keys are already "immutable" in a way (similar to how
    names are "immutables" for objects). Changing the keys would mean deleting
    and creating a new item in the list, which would be permitted by these
    rules.
- Mutable: A field set as mutable could cancel the recursive immutability.

#### On mutation of selected fields

- Error: mutation of an immutable field returns an error and the request is
  rejected. No mutation is performed at all.
- Ignored: mutation of the field (and/or sub-fields) are completely ignored, and
  not persisted. They do not trigger an error.

#### Proposal

The behavior on mutation of immutable field is orthogonal to selection of
immutable fields. If we accepted these two semantics at the same time, we would
have to either have two different markers (one for selection and one for
behavior) or have a single field with multiple options.

For the moment, we understand that "Recursive" is the most important and simple
selection mechanism, and "error" is the most natural way to handle
mutations. For this reason, this proposal focuses on providing those two options
only, with the opportunity to extend this functionality later on.

### Go IDL

The proposal is to add a new marker with the following (only) format:

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

That immutable tag would be extensible in the future by adding extra-parameters if needed:
```
// +immutable=ignore
```

### OpenAPI Extension

Since the semantics of the "readOnly" tag in OpenAPI is not the one we're trying
to have here.  We propose a new Kubernetes specific extensions, which has to
allow for further changes in the semantics of our immutable marker:
```
"x-kubernetes-immutable": ["recursive", "error"]
```

### Mutating admission chain

Mutating admission chain would have the exact same effects as user changes,
meaning that they wouldn't be able to change an object after creation. This is
very similar to what is done today since validation for update is run AFTER all
mutations. On creation, mutation are permitted.

### Where does this happen

This process is meant to happen right before the update validation and after
mutating and validating webhooks, and only run on updates. This will allow us to
keep the exact same behavior while removing the validation code that checks the
immutability of fields.

### Risks and Mitigations

One risk would have been to block updates to some metadata fields. But CRD
validation already guarantees that this is not possible for CRDs.  For
core-types, we would have to assert that `metadata` fields are not immutable.

## Design Details

### Test Plan

Not ready yet

### Graduation Criteria

This feature is mostly considered as a sub-feature of [Server-side
apply](https://github.com/kubernetes/enhancement/pull/555) and will probably
follow a very similar graduation line. It is expected to start directly as a
beta feature and will graduate to GA along with Server-side apply.

## Implementation History

N/A
