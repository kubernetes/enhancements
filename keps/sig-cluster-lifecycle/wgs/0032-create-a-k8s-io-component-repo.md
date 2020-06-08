---
title: Create a `k8s.io/component-base` repo
authors:
  - "@luxas"
  - "@sttts"
owning-sig: sig-cluster-lifecycle
participating-sigs:
  - sig-api-machinery
  - sig-cloud-provider
reviewers:
  - "@thockin"
  - "@jbeda"
  - "@bgrant0607"
  - "@smarterclayton"
  - "@liggitt"
  - "@lavalamp"
  - "@andrewsykim"
  - "@cblecker"
approvers:
  - "@thockin"
  - "@jbeda"
  - "@bgrant0607"
  - "@smarterclayton"
editor: "@luxas"
creation-date: 2018-11-27
last-updated: 2018-12-10
status: implementable
---

# Create a `k8s.io/component-base` repo

**How we can consolidate the look and feel of core and non-core components with regards to ComponentConfiguration, flag handling, and common functionality with a new repository**

## Table of Contents

<!-- toc -->
- [Abstract](#abstract)
  - [History and Motivation](#history-and-motivation)
  - [&quot;Component&quot; definition](#component-definition)
  - [Goals](#goals)
  - [Success metrics](#success-metrics)
  - [Non-goals](#non-goals)
  - [Related proposals / references](#related-proposals--references)
- [Proposal](#proposal)
  - [Part 1: ComponentConfig](#part-1-componentconfig)
    - [Standardized encoding/decoding](#standardized-encodingdecoding)
    - [Testing helper methods](#testing-helper-methods)
    - [Generate OpenAPI specifications](#generate-openapi-specifications)
  - [Part 2: Command building / flag parsing for long-running daemons](#part-2-command-building--flag-parsing-for-long-running-daemons)
    - [Wrapper around cobra.Command](#wrapper-around-cobracommand)
    - [Flag precedence over config file](#flag-precedence-over-config-file)
    - [Standardized logging](#standardized-logging)
  - [Part 3: HTTPS serving](#part-3-https-serving)
    - [Common endpoints](#common-endpoints)
    - [Standardized authentication / authorization](#standardized-authentication--authorization)
  - [Part 4: Sample implementation in k8s.io/sample-component](#part-4-sample-implementation-in-k8siosample-component)
  - [Code structure](#code-structure)
  - [Timeframe and Implementation Order](#timeframe-and-implementation-order)
  - [OWNERS file for new packages](#owners-file-for-new-packages)
<!-- /toc -->

## Abstract

The proposal is about refactoring the Kubernetes core package structure in a way that all core component can share common code around

- ComponentConfig implementation
- flag and command handling
- HTTPS serving
- delegated authn/z
- logging.

Today this code is spread over the `k8s.io/kubernetes` repository, staging repository or pieces of code are in locations they don't belong to (example: `k8s.io/apiserver/pkg/util/logs` is
the for general logging, totally independent of API servers). We miss a repository far enough in the dependency hierarchy for code that is or should be common among core Kubernetes
component (neither `k8s.io/apiserver`, `k8s.io/apimachinery` or `k8s.io/client-go` are right for that).

This toolkit with this shared set of code can then be consumed by all the core Kubernetes components, higher-level frameworks like `kubebuilder` and `server-sdk` (which are more targeted
for a specific type of consumer), as well as any other ecosystem component that want to follow these patterns by vendoring this code as-is.

To implement this KEP in a timely manner and to start building a good foundation for Kubernetes component, we propose to create a Working Group, **WG Component Standard** to facilitate
this effort.

### History and Motivation

By this time in the Kubernetes development, we know pretty well how we want a Kubernetes component to work, function, and look. But achieving this requires a fair amount of more or less
advanced code. As we scale the ecosystem, and evolve Kubernetes to work more as a kernel, it's increasingly important to make writing extensions and custom Kubernetes-aware components
relatively easy. As it stands today, this is anything but straightforward. In fact, even the in-core components diverge in terms of configurability (Can it be declaratively configured? Do
flag names follow a consistent pattern? Are configuration sources consistently merged?), common functionality (Does it support the common "/version," "/healthz," "/configz," "/pprof," and
"/metrics" endpoints? Does it utilize Kubernetes' authentication/authorization mechanisms? Does it write logs in a consistent manner? Does it handle signals as others do?), and testability
(Do the internal configuration structs set up correctly to conform with the Kubernetes API machinery, and have roundtrip, defaulting, validation unit tests in place? Does it merge flags
and the config file correctly? Is the logging mechanism set up in a testable manner? Can it be verified that the HTTP server has the standard endpoints registered and working? Can it be
verified that authentication and authorization is set up correctly?).

![component architecture](component-arch.png)

This document proposes to create a new Kubernetes staging repository with minimal dependencies (_k8s.io/apimachinery_, _k8s.io/client-go_, and _k8s.io/api_) and good documentation on how
to write a Kubernetes-aware component that follows best practices. The code and best practices in this repo would be used by all the core components as well. Unifying the core components
would be great progress in terms of the internal code structure, capabilities, and test coverage. Most significantly, this would lead to an adoption of ComponentConfig for all internal
components as both a "side effect" and a desired outcome, which is long time overdue.

The current inconsistency is a headache for many Kubernetes developers, and confusing for end users. Implementing this proposal will lead to better code quality, higher test coverage in
these specific areas of the code, and better reusability possibilities as we grow the ecosystem (e.g. breaking out the _cloud provider_ code, building Cluster API controllers, etc.).
This work consists of three major pillars, and we hope to complete at least the ComponentConfig part of it—if not (ideally) all three pieces of work—in v1.14.

### "Component" definition

In this case, when talking about a _component_, we mean: "a CLI tool or a long-running server process that consumes configuration from a versioned configuration file and optionally
overriding flags". The component's implementation of ComponentConfig and command &amp; flag setup is well unit-tested. The component is to some extent Kubernetes-aware. The
component follows Kubernetes' conventions for config serialization and merging, logging, and common HTTPS endpoints.

When we say _core Kubernetes components_ we refer to: kube-apiserver, kube-controller-manager, kube-scheduler, kubelet, kube-proxy, and kubeadm.

_To begin with_, this proposal will focus on **factoring out the code needed for the core Kubernetes components**. As we go, however, this set of packages will become
generic enough to be usable by cloud provider and Cluster API controller extensions, as well as aggregated API servers.

### Goals

- Make it easy for a component to correctly adopt ComponentConfig (encoding/decoding/flag merging).
- Avoid moving code into _k8s.io/apiserver_ which does not strictly belong to an etcd-based, API group-serving apiserver. Corollary: remove etcd dependency from components.
- Factor out command- and flag-building code to a shared place.
- Factor out common HTTPS endpoints describing a component's status.
- Make the core Kubernetes components utilize these new packages.
- Have good documentation about how to build a component with a similar look and feel as core Kubernetes components.
- Increase test coverage for the configuration, command building, and HTTPS server areas of the component code.
- Break out OpenAPI definitions and violations for ComponentConfigs from the monorepo to a dedicated place per-component.
- Create a new Working Group, `WG Component Standard`, to facilitate and implement this and future related KEPs
- Integrate well with related higher-level framework projects,  e.g. `kubebuilder` or `controller-runtime`
  - This repo will host a set of packages needed by the core components in a generic manner, like a **generic toolkit**.
  - More higher-level and scoped projects like `kubebuilder` will focus specifically on one target consumer, vendor this code
    and wrap it in a way that makes sense for that specific consumer. That is what we like to refer to here as a **framework**.

### Success metrics

- All core Kubernetes components (kube-apiserver, kube-controller-manager, kube-scheduler, kubelet, kube-proxy, kubeadm) are using these shared packages in a consistent manner.
- Cloud providers can be moved out of core without having to depend on the core repository.
  - Related issue: [https://github.com/kubernetes/kubernetes/issues/69585](https://github.com/kubernetes/kubernetes/issues/69585)
- It's easier for _kubeadm_ to move out of the core repo when these component-related packages are in a "public" staging repository.
- `k8s.io/apiserver` doesn't have any code that isn't strictly related to serving API groups.

### Non-goals

- Graduate any ComponentConfig API versions (in this proposal).
- Make this library toolkit a "generic" cloud-native component builder. Such a framework, if ever created, could instead consume these packages.
  - We'll collaborate with higher-level projects like `kubebuilder` and `controller-runtime` though, and let them implement some of the higher-level code out of scope for this repo.
- Fixing _all the problems_ in the core components, and expanding this beyond what's really necessary.
  - Instead we'll work incrementally, and start with breaking out some basic stuff we _know_ every component must handle (e.g. configuration and flag parsing)
- Specifying (in this proposal) _exactly how a Kubernetes component should function_. That is to be followed up later by the new Working Group.

### Related proposals / references

- [Kubernetes Component Configuration](https://docs.google.com/document/d/1arP4T9Qkp2SovlJZ_y790sBeiWXDO6SG10pZ_UUU-Lc/edit) by [@mikedanese](https://github.com/mikedanese)
- [Versioned Component Configuration Files](https://docs.google.com/document/d/1FdaEJUEh091qf5B98HM6_8MS764iXrxxigNIdwHYW9c/edit#) by [@mtaufen](https://github.com/mtaufen)
- [Moving ComponentConfig API types to staging repos](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/component-config-conventions.md) by [@luxas](https://github.com/luxas) and [@sttts](https://github.com/sttts)
 - [Graduating KubeletFlags subfields to KubeletConfiguration](https://docs.google.com/document/d/18-MsChpTkrMGCSqAQN9QGgWuuFoK90SznBbwVkfZryo/edit) by [@mtaufen](https://github.com/mtaufen)
 - [Making /configz better](https://docs.google.com/document/d/1kNVSdw7H9EqyvI2BYH4EqwVtcg9bi9ZCPpJs8aKZFmM/edit) by [@mtaufen](https://github.com/mtaufen)
 - [Platform-Specific Component Configuration (draft)](https://docs.google.com/document/d/1rfSq9PGn_b7ILvWXjkgU9R6h5C84ywbKUrtJWo0IrYw/edit) by [@mtaufen](https://github.com/mtaufen)

## Proposal

This proposal contains three logical units of work. Each subsection is explained in more detail below.

### Part 1: ComponentConfig

#### Standardized encoding/decoding

- Encoding/decoding helper methods that would be referenced in every scheme package
  - To make it easy to encode/decode a type registered in a scheme, `Encode`/`Decode` helper funcs would be registered in the scheme package
  - Depending on the what these helper methods start to look like, they will be put in either `k8s.io/apimachinery` or `k8s.io/component-base`
- Warn or (if desired, error) on unknown fields by creating a new strict universal codec in `k8s.io/apimachinery`
  - This makes it possible for the component to spot e.g. config typos and notify the user instead of silently ignoring invalid config.
  - More high-level, this codec can be used for e.g. a `--validate-config` flag
- Support both JSON and YAML everywhere
  - With 100% coverage on supporting both. If the component implements its own code something might go wrong so e.g. only YAML is supported
- Support multiple YAML documents if needed
  - For instance, supporting to read a directory of JSON/YAML files, or reading one single file with multiple YAML files

#### Testing helper methods

- Conversion / roundtrip testing
  - Using YAML files in `testdata/` as well as roundtrip fuzzing testing
- API group testing
  - External types must have JSON tags
  - Internal types must not have any JSON tags
  - Verify expected API version name
  - Verify expected API Group name
  - Verify that the expected ComponentConfig type exists
- Defaulting testing using YAML files in `testdata/`
- Validation testing using YAML files in `testdata/`

#### Generate OpenAPI specifications

Provide a common way to generate OpenAPI specifications local to the component, so that external consumers can access it, and the component can expose it via e.g. a CLI flag or HTTPS
endpoint.

The API naming violations handling (right now done monolithically in the core k8s repo) will become local to the component's API group. In other words, if a Go `CapitalCase` field name
doesn't have a similar JSON `camelCase` name, today an OpenAPI exception will be stored in `k8s.io/kubernetes/api/api-rules/violations_exception.list`. This will be changed
for components though, so that they register their allowed exceptions in the `register_test.go` unit test file in the external API group directory.

### Part 2: Command building / flag parsing for long-running daemons

Getting this code centralized and standardized will make it way easier for a server/daemon/controller component to be implmented.
A more detailed document on _exactly how_ a Kubernetes server component should implement this is subject to a new KEP that will be created
later by the WG.

Please note that CLI tools are **not** targeted with this shared code here, only long-running daemons that basically only register flags to that one command.

#### Wrapper around cobra.Command

See the `cmd/kubelet` code for how much extra setup a Kubernetes component needs to do for building commands and flag sets. This code can be refactored into a generic wrapper around
_cobra_ for use with Kubernetes.

Note: As part of follow-up proposals from the new Working Group, we might also consider using only `pflag` for server/daemon components.

#### Flag precedence over config file

If the component supports both ComponentConfiguration and flags, flags should override fields set in the ComponentConfiguration. This is not straightforward to implement in code, and only
the kubelet does this at the moment. Refactoring this code in a generic helper library in this new repository will make adoption of the feature easy and testable.

The details of flag versus ComponentConfig semantics are _to be decided later in a different proposal_. Meanwhile, this flag precedence feature will be **opt-in**, so the kubelet and
kubeadm can directly adopt this code, until the details have been decided on for all components.

#### Standardized logging

Use the _k8s.io/klog_ package in a standardized way.

### Part 3: HTTPS serving

Many Kubernetes controllers are clients to the API server and run as daemons. In order to expose information on how the component is doing (e.g. profiling, metrics, current configuration,
etc.), an HTTPS server is run.

#### Common endpoints

In order to make it easy to expose this kind of information, a package is made in this new repo that hosts this common code. Initially targeted
endpoints are "/version," "/healthz," "/configz," "/pprof," and "/metrics."

#### Standardized authentication / authorization

In order to not expose this kind of information (e.g. metrics) to anyone that can talk to the component, it may utilize SubjectAccessReview requests to the API server, and hence delegate
authentication and authorization to the API server. It should be easy to add this functionality to your component.

### Part 4: Sample implementation in k8s.io/sample-component

Provides an example usage of the three main functions of the _k8s.io/component-base_ repo, implementing ComponentConfig, the CLI wrapper tooling and the common HTTPS endpoints with delegated
auth.

### Code structure

- k8s.io/component-base
  - config/
    - Would hold internal, shared ComponentConfig types across core components
    - {v1,v1beta1,v1alpha1}
      - Would hold external, shared ComponentConfig types across core components
    - serializer/
      - Would hold common methods for encoding/decoding ComponentConfig
    - testing/
      - Would hold common testing code for use in unit tests local to the implementation of ComponentConfig.
  - cli/
    - Would hold common methods and types for building a k8s component command (building on top of github.com/spf13/{pflag,cobra})
    - flags/
      - Would hold flag implementations for custom types like `map[string]string`, tri-state string/bool flags, TLS cipher/version flags, etc.
      - Code will be moved from `k8s.io/apiserver/pkg/util/{global,}flag`, `k8s.io/kubernetes/pkg/util/flag` and `k8s.io/kubernetes/pkg/version/verflag`.
    - testing/
      - Would hold common testing code for use in unit tests local to the implementation of the code
    - logging/
      - Would hold common code for using _k8s.io/klog_
      - Code will be moved from `k8s.io/apiserver/pkg/util/logs`
  - server/
    - auth/
      - Would hold code for implementing delegated authentication and authorization to Kubernetes
      - authorizer/
        - factory/
          - Code will be moved from `k8s.io/apiserver/pkg/authorization/authorizationfactory/delegating.go`
        - delegating/
          - Code will be moved from `k8s.io/apiserver/plugin/pkg/authorizer/webhook`
        - options/
          - Code will be moved from `k8s.io/apiserver/pkg/server/options/authorization.go`
      - authentication/
        - factory/
          - Code will be moved from `k8s.io/apiserver/pkg/authentication/authenticationfactory/delegating.go`
        - delegating/
          - Code will be moved from `k8s.io/apiserver/pkg/server/options/authentication.go`
        - options/
          - Code will be moved from `k8s.io/apiserver/pkg/server/options/authentication.go`
    - configz/
      - Would hold code for implementing a `/configz` endpoint in the component.
      - Code will be moved from `k8s.io/kubernetes/pkg/util/configz`.
      - Also consider moving the flag debugging endpoints in `k8s.io/apiserver/pkg/server/routes/flags.go` over here
    - healthz/
      - Would hold code for implementing a `/healthz` endpoint in the component
      - Code will be moved from `k8s.io/apiserver/pkg/server/healthz`
    - metrics/
      - Would hold code for implementing a `/metrics` endpoint in the component
      - Code will be moved from `k8s.io/kubernetes/pkg/util/metrics`,
        `k8s.io/apiserver/pkg/server/routes/metrics.go`, and `k8s.io/apiserver/pkg/endpoints/metrics`
    - openapi/
      - Would hold code for implementing a `/openapi/v2` endpoint in the component
      - Code will be moved from `k8s.io/apiserver/pkg/server/routes/openapi.go`
    - pprof/
      - Would hold code for implementing a `/pprof` endpoint in the component
      - Code will be moved from `k8s.io/apiserver/pkg/server/routes/profiling.go`
    - version/
      - Would hold code for implementing a `/version` endpoint in the component
      - Code will be moved from `k8s.io/apiserver/pkg/server/routes/version.go`
    - signal/
      - Would hold code to notify the command when SIGTERM or SIGINT happens, via a stop channel
      - Code will be moved from `k8s.io/apiserver/pkg/server/signal*.go`

### Timeframe and Implementation Order

**Objective:** The ComponentConfig part done for v1.14

**Stretch goal:** Get the CLI and HTTPS server parts done for v1.14.

**Implementation priorities:**

Here is a rough sketch of the priorities of the work for v1.14. Many of these items may be done in parallel.
We plan to "go broad over deep" here, in other words focus on implementing one step for each component before
proceeding with the next task vs. doing everything for a component before proceeding to the next one.

1. Create the `k8s.io/component-base` repo with the initial ComponentConfig shared code (e.g. packages earlier in `k8s.io/apiserver`)
2. Move shared `v1alpha1` ComponentConfig types and references from `k8s.io/api{server,machinery}/pkg/apis/config` to `k8s.io/component-base/config`
3. Set up good unit testing for all core ComponentConfig usage, by writing the `k8s.io/component-base/config/testing` package
4. Move server-related util packages from `k8s.io/kubernetes/pkg/util/` to `k8s.io/component-base/server`. e.g. delegated authn/authz "/configz", "/healthz", and "/metrics" packages are suitable
5. Move common flag parsing / cobra.Command setup code to `k8s.io/component-base/cli` from (mainly) the kubelet codebase.
6. Start using the command- and server-related code in all core components.

In parallel to all the steps above, a _k8s.io/sample-component_ repo is built up with an example and documentation how to consume the _k8s.io/component-base_ code

### OWNERS file for new packages

- Approvers for the `k8s.io/component-base/config/{v1,v1beta1,v1alpha1}` packages
  - @kubernetes/api-approvers
- Approvers for `staging/src/k8s.io/{sample-component,component-base}`
  - @sttts
  - @luxas
  - @jbeda
  - @lavalamp
- Approvers for moved subpackages:
  - those who owned packages before code move
- Reviewers for `staging/src/k8s.io/{sample-component,component-base}`:
  - @sttts
  - @luxas
  - @dixudx
  - @rosti
  - @stewart-yu
  - @dims
- Reviewers for moved subpackages:
  - those who owned packages before code move
