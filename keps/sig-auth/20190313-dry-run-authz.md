---
title: Dry-run authz
authors:
  - "@apelisse"
owning-sig: sig-auth
participating-sigs:
  - sig-api-machinery
reviewers:
  - TBD
approvers:
  - "@liggitt"
  - "@lavalamp"
editor: TBD
creation-date: 2019-03-13
last-updated: 2019-03-13
status: provisional
see-also:
  - "/keps/sig-api-machinery/0015-dry-run.md"
replaces:
superseded-by:
---

# Dry-run authz

## Table of Contents

## Summary

Kubernetes server-side dry-run is a feature that lets people validate their
changes without having to persist them. It allows them to verify that the
validation (including built-in admission chain and validating webhooks) for
their object would be passing, and to know what changes would be made by
defaulter and mutating webhooks. It's also used as a foundation to build
`kubectl diff` which lets people visualize their changes in a more natural
way. It's a powerful tool for CI/CD. Unfortunately, dry-run doesn't have a
specific authorization model, which means that users need write-access to
perform these non-persisting requests.

## Motivation

Proper CI and CD user isolation requires that the user used for CI will not be the
same as the CD user, mostly because CI shouldn't require write access. Also,
it's hard for arbitrary users of the system to use dry-run since administator
don't necessarily want to permit all users to make modifying requests.

Having a "dry-run" type of authorization would allow administrator to allow some
users to be able to run dry-run without giving them write-access to the cluster.

### Goals

- Allow dry-run requests without requiring write access

### Non-Goals

## Proposal

We're proposing the following changes:
- "populate dry run into the request attributes strict parsed from query params
prior to the authz filter" (I have no idea what this means, I copied what
@liggitt said)
- Add a new dry-run attribute in the SubjectAccessReview API: to be defined
- Give proper semantic to RBAC so that one can do a dry-run CREATE but not CREATE

### Risks and Mitigations

A bug in the dry-run machinery could cause escalation for a user that has
dry-run permission to be allowed to write on the server.

## Design Details

## Implementation History

N/A

## Alternatives [optional]

None yet.
