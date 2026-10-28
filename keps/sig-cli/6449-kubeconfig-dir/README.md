<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-6449: Add `$HOME/.kube/config.d` support and automatic Kubeconfig Files Discovery

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 (Optional)](#story-1-optional)
    - [Story 2 (Optional)](#story-2-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Configuration Discovery](#configuration-discovery)
  - [Loading Precedence](#loading-precedence)
  - [Kubeconfig File Format](#kubeconfig-file-format)
  - [Existing Merge Machinery](#existing-merge-machinery)
  - [Tracking Configuration Origin](#tracking-configuration-origin)
  - [Read Operations](#read-operations)
  - [Write Operations](#write-operations)
    - [Modifying Existing Contexts](#modifying-existing-contexts)
    - [Modifying Existing Clusters](#modifying-existing-clusters)
  - [Delete Operations](#delete-operations)
  - [Rename Operations](#rename-operations)
  - [Current Context](#current-context)
  - [Creating New Objects](#creating-new-objects)
  - [Duplicate Contexts](#duplicate-contexts)
  - [Duplicate Clusters and Users](#duplicate-clusters-and-users)
  - [Explicit Disambiguation](#explicit-disambiguation)
  - [Conflict Detection and Merge Precedence](#conflict-detection-and-merge-precedence)
  - [Interaction with --kubeconfig](#interaction-with---kubeconfig)
- [Interaction with KUBECONFIG](#interaction-with-kubeconfig)
  - [External Tool-Generated Kubeconfigs](#external-tool-generated-kubeconfigs)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.
-->

Kubernetes clients currently support loading configuration from a default
kubeconfig file or from an explicit list of files, for example through the
`KUBECONFIG` env var. Managing an increasing number of independent
kubeconfig files requires users and tools to either modify a shared kubeconfig
file or maintain an explicit list of configuration files.

This proposal proposes support for discovering kubeconfig files from a well-known
configuration directory, initially proposed as `$HOME/.kube/config.d/`, in addition
to the existing default kubeconfig, thats mean `$HOME/.kube/config`.

This would allow users(or tools) to manage kubeconfig files independently while
making the clusters, users, and contexts defined by those files available to
Kubernetes clients without requiring every producer/users to modify the same
`~/.kube/config` file.

The proposal intends to build on the existing kubeconfig loading/merging and
`LocationOfOrigin` mechanisms(TBD) where possible. It also defines how configuration
origin, conflicting definitions, configuration writes and compatibility with
existing kubeconfig behavior should be handled.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Kubernetes users frequently interact with **multiple clusters** whose 
kubeconfig data originates from different sources. like:
- cluster provisioning tools generating kubeconfig files;
- cloud-provider CLIs generating or updating cluster credentials;
- users downloading kubeconfigs for independently managed clusters;
- and many more...

Although kubeconfig supports multiple clusters, users and contexts in a single file, 
combining independently generated configuration into `$HOME/.kube/config` is cumbersome.

Example like: a user may have:
```sh 
  $HOME/.kube/config
  $HOME/downloads/development.kubeconfig
  $HOME/downloads/production.kubeconfig
```

To use all of these configurations together today, the user needs to merge them or 
explicitly construct `KUBECONFIG`, for example:

`KUBECONFIG=$HOME/.kube/config:$HOME/downloads/development.kubeconfig:$HOME/downloads/production.kubeconfig <kubectl config commands>`
OR tools may instead modify `$HOME/.kube/config` directly. This requires multiple 
independent tools to coordinate writes to the same file and makes the lifecycle of 
individual cluster configurations harder to manage.

So the actual motivation is: according to this proposal- with directory-based 
discovery, a user(or tool) could instead manage:
```sh
$HOME/.kube/config
$HOME/.kube/config.d/development.yaml
$HOME/.kube/config.d/production.yaml
$HOME/.kube/config.d/provider.yaml
```
and kubectl could consume those files as a combined configuration.

This makes individual kubeconfig files easier to install, replace and manage 
independently. **This would make Kubernetes administrators life easy.**


### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
1. Allow kubectl/client-go to discover kubeconfig files from a well-known 
configuration directory(in our proposal: `$HOME/.kube/config.d/`)
2. Allow independently generated kubeconfig files to participate in the 
effective kubeconfig without requiring users or tools to modify `$HOME/.kube/config`.
3. Reuse the existing kubeconfig loading/merging implementation 
wherever suitable(proposal focused).
4. Preserve the source of clusters, users, and contexts loaded from individual 
files so that operations targeting existing objects can identify their originating files.
5. Define deterministic and safe behavior when the same named object appears in
 multiple discovered kubeconfig files.

6. Preserve backward compatibility with the existing default kubeconfig behavior 
as much as possible.
7. **The most important: `$HOME/.kube/config` will be the main config file.**

### Non-Goals

(As per the discussion from sig-cli meeting. Need more input)

1. Introduce ownership metadata for kubeconfig files.
2. Determine which external tool generated or owns a kubeconfig.
3. Introduce a new kubeconfig API or file format.
4. Require one context, cluster, or credential per file.
5. Replace KUBECONFIG.
6. Replace --kubeconfig.
7. Provide lifecycle management for kubeconfig files generated by external tools.
8. redesign authentication or credential storage.
9. require external tools to immediately adopt the new directory layout.(TBD)

NOTE: Shell prompt integrations or other mechanisms for displaying the active 
context/configuration may be useful work, but are not sure if this is required for the 
initial implementation. (need input)


<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

Introduce a default kubeconfig directory: `$HOME/.kube/config.d/`.
When using the default kubeconfig loading behavior, kubectl would consider the existing 
default file: `$HOME/.kube/config`, along with valid kubeconfig files discovered in:
`$HOME/.kube/config.d/`
The structre could be: 
```sh
$HOME/.kube/
├─ config
└─ config.d/
    ├─ development.yaml
    ├─ production.yaml
    ├─ provider.yaml
    └─ etc...
```

The resulting configuration would expose clusters, users, and contexts from those 
sources through the existing kubeconfig APIs.

For example: `kubectl config get-contexts`: could display contexts originating from both 
`$HOME/.kube/config` and files in `$HOME/.kube/config.d/`.

The implementation should make the discovered files participate in the existing 
kubeconfig loading/merging model instead of introducing a second merge 
implementation specifically for config.d.


### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1 (Optional)

#### Story 2 (Optional)

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

Risk: The same context, cluster, or user may be defined in multiple files.
Mitigation: Detect ambiguous definitions and return an error instead 
of silently choosing one. Users can disambiguate with `--kubeconfig`.

Risk: With multiple kubeconfigs, write operations may target the wrong configuration file.
Mitigation: Use the object's `LocationOfOrigin` to update, rename, or delete 
it in its source file.

Risk: Older kubectl versions will not discover files under config.d.
Mitigation: Keep the existing `$HOME/.kube/config` behavior unchanged 
and continue storing global state such as current-context there.

Risk: Automatic discovery could unexpectedly affect `KUBECONFIG` or `--kubeconfig`.
Mitigation: Preserve explicit kubeconfig selection semantics and apply config.d 
discovery only according to clearly defined default loading rules.

Risk: A discovered file may have been generated by another tool. Modifying it with
kubectl config could conflict with that tool's lifecycle.
Mitigation: This proposal does not attempt to establish file ownership. Existing
configuration modification commands remain explicit user actions.

Risk: An automatically discovered file may be invalid or inaccessible.
Mitigation: The discovery rules and failure behavior will be explicitly defined. Errors
should identify the physical source file so users can diagnose the problem.


<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

This enhancement extends the existing kubeconfig loading model to automatically 
discover kubeconfig files from a well-known directory while preserving the existing 
kubeconfig format, loading machinery and configuration commands 
as much as possible.

`$HOME/.kube/config` remains the primary/default/main kubeconfig. Files under 
`$HOME/.kube/config.d/` provide additional clusters, users, and contexts that participate 
in the effective configuration.

NOTE: The feature should be implemented as an extension of the existing client-go 
loading rules rather than as a separate loading or merging implementation.

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Configuration Discovery

When kubectl uses its default kubeconfig loading behavior, it currently 
considers `$HOME/.kube/config` as the default configuration source.
With this enhancement, the loading rules will additionally discover kubeconfig files from:
`$HOME/.kube/config.d/`

If config.d does not exist, kubectl continues with the existing behavior. 
The **directory is optional** and kubectl should not create it merely as a side 
effect of reading configuration.

The discovery step must produce a deterministic list of files. The exact 
ordering rules should be explicitly defined because ordering is 
observable through existing kubeconfig merge semantics even if ambiguous duplicate 
definitions are rejected separately.

Directory discovery must also distinguish kubeconfig candidates from unrelated files.

The implementation should define how other entries are handled, including directories, 
symbolic links, hidden files, editor backup files, and malformed files.

### Loading Precedence
The discovered files must become part of the actual client-go loading source list rather than 
being discovered only inside `ClientConfigLoadingRules.Load()`.

This distinction is important because kubeconfig is used by both read and write paths.

For example: adding files only during `Load()` may result in:

```sh
GetLoadingPrecedence()
    -> ~/.kube/config

Load()
    -> ~/.kube/config
    -> ~/.kube/config.d/development.yaml
    -> ~/.kube/config.d/production.yaml
```

In this situation, the effective configuration(read) and the configuration modification 
machinery(write) would have different understandings of which files participate in 
kubeconfig loading.

so, the source list should be consistently constructed before loading:
```sh
GetLoadingPrecedence()
        ->
        ~/.kube/config
        ~/.kube/config.d/development.yaml
        ~/.kube/config.d/production.yaml
ClientConfigLoadingRules
        |
       Load()
``` 

This allows normal kubectl operations and kubectl config operations to observe the same 
configuration sources.

The exact precedence between the main kubeconfig and discovered files must be explicitly 
specified. Similarly, the interaction with *KUBECONFIG* and an explicit **--kubeconfig* must preserve
their existing semantics.(need input)

NOTE: The behavior when KUBECONFIG is present needs to be defined separately as part of the 
compatibility contract.

### Kubeconfig File Format

Files in config.d are ordinary kubeconfig files.
No new fragment format is introduced.

**The enhancement does not require one context or one cluster per file.**

### Existing Merge Machinery

The implementation should continue to use the existing client-go kubeconfig loader and merge machinery.

### Tracking Configuration Origin

A central requirement for multi-file configuration is retaining the physical source of each named object.
Consider:

```sh
config.d/development.yaml
    cluster: development
    user: development-user
    context: development

config.d/production.yaml
    cluster: production
    user: production-user
    context: production
```

After loading, kubectl needs to know not only that a context named development
exists, but also that it originated from: `$HOME/.kube/config.d/development.yaml`

client-go already associates loaded clusters, contexts, and authentication 
information with `LocationOfOrigin`.

### Read Operations

Normal `kubectl` commands consume the effective configuration created from all applicable sources.

For example:
```sh
~/.kube/config
    context: minikube

config.d/development.yaml
    context: development

config.d/production.yaml
    context: production
```
should allow: `kubectl config get-contexts`

to expose:
```sh
minikube
development
production
```

same with other read operations.

### Write Operations

Write operations need to distinguish between modifications to existing named objects and 
modification of global/default selection state. (need input)

1. For existing named objects, the source file determines the write destination.
2. For global state such as `current-context`, the main kubeconfig remains the write destination.

#### Modifying Existing Contexts

Suppose: `$HOME/.kube/config.d/development.yaml`

contains:
```sh
contexts:
- name: development
  context:
    cluster: development
    user: development-user
```

Running: `kubectl config set-context development --namespace=otel`

should update the existing context in: `$HOME/.kube/config.d/development.yaml`
resulting conceptually in:
```sh
contexts:
- name: development
  context:
    cluster: development
    user: development-user
    namespace: otel
```

*The operation should not create a second development context in `$HOME/.kube/config`*.
The same rule applies to modifications of existing clusters and authentication information.

#### Modifying Existing Clusters

Given:
```sh
config.d/development.yaml
    cluster: development
```

running:
```sh
kubectl config set-cluster development \
    --server=https://new-development.example.com
```

should locate the existing cluster, determine its `LocationOfOrigin`, and modify that physical file.

### Delete Operations

Deletion follows the same provenance rule.

For example: `kubectl config delete-context development` 
should identify where development originated and remove it from that file.

### Rename Operations

Renaming an existing object also operates against its originating file.

For example: `kubectl config rename-context development dev`
should update the context in the file from which development was loaded.

The implementation must additionally validate that the new name does not 
introduce an ambiguous definition in the effective multi-file configuration.

### Current Context

`current-context` differs from named cluster, user, and context entries.

The selected context represents user-level default state and should remain 
stored in the main kubeconfig:
`$HOME/.kube/config`

### Creating New Objects

Creating a new object differs from modifying an existing object because no source 
file exists yet.

For example:
```sh
kubectl config set-cluster new-cluster \
    --server=https://new.example.com
```
cannot use `LocationOfOrigin` because `new-cluster` has not previously been loaded.

The initial design should define a predictable default destination for these operations. 
One compatibility-oriented option is to retain the existing behavior and 
create new objects in `$HOME/.kube/config`. (need input)

### Duplicate Contexts

Given:
```sh 
file-a.yaml:
    context: production

file-b.yaml:
    context: production
```

kubectl should retain enough information during loading/merging to know that 
production has multiple definitions.

**An operation requiring an unambiguous context must not silently choose one based only on directory ordering.**

For example:

`kubectl --context=production get pods`: should fail when the context cannot be uniquely resolved.

A diagnostic should identify the conflicting files, conceptually:
```sh
context "production" is defined in multiple kubeconfig files:
  ~/.kube/config.d/file-a.yaml
  ~/.kube/config.d/file-b.yaml
```
This prevents a user from unintentionally communicating with a different cluster than intended.

### Duplicate Clusters and Users

The same principle applies to named clusters and authentication information when an operation needs to identify a particular definition.

For example:
```sh
file-a.yaml:
    cluster: development

file-b.yaml:
    cluster: development
```

makes this operation ambiguous:
```sh
kubectl config set-cluster development \
    --server=https://new.example.com
```
Kubectl cannot safely determine which file should be modified.

**The command should fail instead of modifying whichever definition happens to win merge precedence.**

### Explicit Disambiguation

Users should be able to bypass automatic multi-file resolution by explicitly selecting a kubeconfig file.

For example:
```sh
kubectl --kubeconfig=$HOME/.kube/config.d/file-a.yaml \
    config set-cluster development \
    --server=https://new.example.com
```
provides an unambiguous configuration source.

In this case, normal explicit-file behavior should apply and automatic default
directory discovery should not introduce additional definitions into the operation.

### Conflict Detection and Merge Precedence

Conflict detection and merge precedence serve different purposes.
Loading still requires deterministic ordering:
```sh
config
config.d/a
config.d/b
config.d/c
```

However, deterministic ordering should not imply that duplicate independently 
discovered objects are always **intentional overrides.**

For example:
```sh
10-development.yaml
    context: development

90-development.yaml
    context: development
```

must not automatically establish an **/etc-style** override contract merely because 
lexical ordering is deterministic.

Where uniqueness is required to safely select or modify an object, ambiguity should be surfaced.

NOTE: The exact distinction between conflicts that are allowed to participate 
in ordinary merge semantics and conflicts that must produce errors should be 
finalized as part of the implementation design

### Interaction with --kubeconfig

An explicit `--kubeconfig` represents an explicit user selection of configuration input.

For example: `kubectl --kubeconfig=/tmp/test.yaml get pods`

should continue to operate against the explicitly selected configuration 
rather than unexpectedly including config.d

This is particularly important because explicit file selection 
is also the proposed mechanism for resolving ambiguous definitions.

## Interaction with KUBECONFIG

`KUBECONFIG` already provides an explicit ordered list of kubeconfig files.

For example: `KUBECONFIG=a.yaml:b.yaml:c.yaml kubectl get pods`

**The enhancement should preserve existing KUBECONFIG behavior.**

The design needs to explicitly determine whether setting `KUBECONFIG` suppresses 
default config.d discovery or whether discovered files participate in 
addition to the environment-provided files. (need input)

NOTE: A compatibility-preserving model would treat `KUBECONFIG` as an explicit source 
list and perform automatic `config.d` discovery only when default loading rules are being used.

### External Tool-Generated Kubeconfigs

**Kubectl should not attempt to determine which tool owns a discovered kubeconfig.**

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
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
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

### Graduation Criteria

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
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
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

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

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

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

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

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

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

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

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

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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

###### What steps should be taken if SLOs are not being met to determine the problem?

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

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
