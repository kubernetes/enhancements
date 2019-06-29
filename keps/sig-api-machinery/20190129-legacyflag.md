---
title: legacyflags
authors:
  - "@mtaufen"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-architecture
  - sig-cluster-lifecycle
  - wg-component-standard
reviewers:
  - "@kubernetes/wg-component-standard"
approvers:
  - "@luxas"
  - "@sttts"
editor: TBD
creation-date: 2019-01-29
last-updated: 2019-04-02
status: provisional
see-also:
  - KEP-32
---

# kflag

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Requirements](#requirements)
    - [General](#general)
    - [Flag Registration](#flag-registration)
    - [Flag Parsing](#flag-parsing)
    - [Application of Parsed Flag Values](#application-of-parsed-flag-values)
  - [Implementation Details](#implementation-details)
    - [Code Location](#code-location)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Maintenance](#maintenance)
    - [Migrating existing components](#migrating-existing-components)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Create a _legacyflags_ package, which wraps pflag and addresses the pain-points of backwards-compatible
ComponentConfig migration, and selective registration of third-party library flags.


## Motivation

Make it easier for component owners to implement
[Versioned Component Configuration Files](https://docs.google.com/document/d/1FdaEJUEh091qf5B98HM6_8MS764iXrxxigNIdwHYW9c/edit#)
(a.k.a. ComponentConfig).

While a number of problems with existing machinery are highlighted in the above doc, this
proposal is focused on the following two use-cases:
* Enforcing [flag precedence](https://github.com/kubernetes/kubernetes/issues/56171) between values
  specified on the command line and values from a config file, which is required to incrementally
  migrate flags to a config file with backwards compatibility.
* Preventing third-party code from
  [implicitly registering flags](https://github.com/kubernetes/kubernetes/pull/57613)
  against a component's command-line interface, which is required for components to maintain
  explicit control over their command-line interface.

Rather than require other components to copy and customize the Kubelet's relatively complex
solutions to these problems, we should put the common functionality in a library, legacyflags, to reduce
the burden on component owners.

### Goals

Provide core Kubernetes components with common building blocks that simplify the following:
* Application of parsed flag values to arbitrary structs (e.g. config structs).
* Selective inclusion of globally-registered flags in third-party code.

### Non-Goals

* This KEP is not concerned with fixing various problems with Cobra, such as third-party flags
  leaking into the default usage or help text. We may write a future KEP to cover these issues and
  command building in general.
* This KEP is not concerned with the structure of flag and configuration types or their composition
  into aggregates. We may write a future KEP to cover this.
* This KEP is not concerned with loading and parsing config files, or resolving references contained
  in those files (e.g. relative paths).
* This KEP is not concerned with dynamically configuring components, or otherwise managing
  component lifecycle.
* This KEP is not concerned with flag or config validation.
* This KEP is not an attempt to fork pflag.

## Proposal

This proposal recommends writing a wrapper package around the
[pflag](https://godoc.org/github.com/spf13/pflag) library.
This wrapper package is called _legacyflags_ in this proposal.
The intent is _not_ to fork pflag.

This KEP is mostly about pinning down requirements and is accompanied by an example PR, linked
in the below [Implementation Details](#implementation-details) section, which demonstrates _one_ way
of accomplishing these goals.

### Requirements

#### General
* Components that consume legacyflags should require less maintenance than components that implement
  ComponentConfig without legacyflags.
* legacyflags should be as compile-time type-safe as possible. There are many possible combinations of
  config, so if runtime casts from interface{} are required, it may be difficult to test all
  paths for safety.
* Unless a component needs to access the underlying pflag FlagSet to implement additional custom
  behavior, it should not need to use pflag directly.

#### Flag Registration
* Flags registered by third-party libraries must not appear in a component's command-line interface
  unless the component owners explicitly allow it.
* legacyflags should provide helpers for importing globally-registered third-party flags into a FlagSet.

#### Flag Parsing
* Components should only have to parse their command-line once, even if they have to apply the flag
  values multiple times, or to multiple structs. The Kubelet's approach to enforcing flag-precedence
  involves re-parsing the command-line, which led to significant complexity. This is especially
  important if components allow dynamic config reloading at runtime, in which case flags need
  to be applied on top of each new config.

#### Application of Parsed Flag Values
* Components must be able to apply the parsed values an arbitrary number of times without
  accumulating side-effects. Components may apply values once for at least as many config sources
  they can consume (at least once to access the initial flag values, and afterward to enforce flag
  precedence on each config). Each stage of loading configuration (flags, config file, dynamic
  config) is a potential decision point, prior to which flag precedence must be enforced.
* Components should be able to apply the parsed flag values to structs that only represent a subset
  of all flag values. For instance, if a component has a Flags struct and a Config struct,
  it may need to apply values to each independently.
* Components must be able to specify custom behavior for merging values during application.
  Feature gates, for example, are merged piecemeal between the command-line and config files, with
  gates specified on the command-line taking precedence.
* Application of flag values should only consider the flags specified on the command line. It should
  be careful, for example, to not overwrite values loaded from config files with the default values
  of omitted flags.


### Implementation Details

Fundamentally, separating parsing from application requires that we have a scratch space to parse
the initial values into, and a way to map that scratch space onto target objects.

To maintain type safety, this proposal recommends a two-layered approach, where legacyflags handles the
common types, and components handle the component-specific types:
* legacyflags Implements:
  * Statically typed flag registration helpers that allocate scratch space.
  * Statically typed helpers to apply the values of parsed flags to arbitrary targets.
* Components Implement:
  * Aggregate flag registration functions, which use legacyflags helpers to register groups of flags the
    component cares about.
  * Aggregate flag application functions, which use legacyflags helpers to apply a subset of flag values
    to an arbitrary structure determined by the component.

This proposal recommends the approach in this example PR:
* https://github.com/kubernetes/kubernetes/pull/73494

See also @sttts's prototype for an alternative approach that predates this KEP:
* https://github.com/kubernetes/kubernetes/pull/72037

#### Code Location

We will implement legacyflags in a new repo: `k8s.io/legacyflags`.

During the implementation, we will maintain an open PR against `k8s.io/kubernetes` that vendors
the latest version of the library and demonstrates its use in a real component.

### Risks and Mitigations

#### Maintenance
The legacyflags package is a wrapper around an existing package, pflag.
The pflag package has a stable API, so we would at most expect to extend legacyflags when pflag is
extended, but only in the event that we really need that extension. We expect this to be infrequent.
Given that legacyflags should make components easier to maintain (it reduces the complexity of the
bootstrap process by making the flag precedence implementation easier to use and reason about),
the maintenance of legacyflags itself is likely time well spent.

#### Migrating existing components
This should amount to an internal refactoring for existing components, with no external changes in
behavior (if we do change external-facing behavior, it's a bug). Some components may have better
test coverage than others, or unique features (like dynamic Kubelet config), so we will have to be
careful to test properly on a component-by-component basis as we migrate components to use legacyflags.

## Graduation Criteria

* Functional completeness:
  * legacyflags implements helpers for the same set of types as pflag (or at least all the types used
    by core Kubernetes components).
  * We are satisfied that the example conversion PR demonstrates legacyflags's utility, measured by
    whether the conversion PR can be merged.
  * (optional) legacyflags absorbs the helpers that currently exist in `k8s.io/apiserver/pkg/util/flag`,
    such as `MapStringBool`.
* Migration completeness:
  * All core Kubernetes components (excluding kubectl) use legacyflags to implement their
    legacy command-line interface as they transition to ComponentConfig.

## Implementation History

* 2019-01-29: Initial KEP PR.