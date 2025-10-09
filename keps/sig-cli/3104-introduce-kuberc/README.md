<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [x] **Merge early and iterate.**
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
# KEP-3104: Introduce kuberc

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
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
    - [Story 4](#story-4)
    - [Story 5](#story-5)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [Open Questions](#open-questions)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Kubectl Kuberc Management Command (kubectl kuberc)](#kubectl-kuberc-management-command-kubectl-kuberc)
  - [kubectl kuberc view](#kubectl-kuberc-view)
  - [kubectl kuberc set --section defaults](#kubectl-kuberc-set---section-defaults)
    - [command](#command)
    - [option](#option)
    - [overwrite](#overwrite)
  - [kubectl kuberc set --section aliases](#kubectl-kuberc-set---section-aliases)
    - [name](#name)
    - [command](#command-1)
    - [option](#option-1)
    - [prependarg](#prependarg)
    - [appendarg](#appendarg)
  - [Allowlist Design Details](#allowlist-design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

This proposal introduces an optional `kuberc` file that is used to separate cluster credentials and server configuration from user preferences.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->
kubectl is one of the oldest parts of the Kubernetes project and has a strong guarantee on backwards compatibility. We want users to be able to opt in to new behaviors (like delete confirmation) that may be considered breaking changes for existing CI jobs and scripts.

[kubeconfig already has a field for preferences](https://github.com/kubernetes/kubernetes/blob/474fd40e38bc4e7f968f7f6dbb741b7783b0740b/staging/src/k8s.io/client-go/tools/clientcmd/api/types.go#L43) that is currently underutilized. The reason for not using this existing field is that creating a new cluster generally yields a new kubeconfig file which contains credentials and host information. While kubeconfigs can be merged and specified by path, we feel there should be a clear separation between server configuration and user preferences.

Additionally, users can split kubeconfig files into various locations, while maintaining a single preference file that will apply no matter which `--kubeconfig` flag or `$KUBECONFIG` env var is pointing to.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

* Introduce a `kuberc` file as a place for client preferences.
* Version and structure this file in a way that makes it easy to introduce new behaviors and settings for users.
* Enable users to define kubectl command aliases.
* Enable users to define kubectl default options.
* Deprecate [kubeconfig `Preferences` field](https://github.com/kubernetes/kubernetes/blob/4b024fc4eeb4a3eeb831e7fddec52b83d0b072df/staging/src/k8s.io/client-go/tools/clientcmd/api/v1/types.go#L40).

### Non-Goals

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

We propose introducing a new file that will be versioned and intended for user provided preferences. This new file is entirely opt-in.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a user I would like to specify preferences per client (device) without my kubeconfig overriding them when I access different clusters.

#### Story 2

As a user I would like to opt in to enforcing delete confirmation or colored output on my local client but not disrupt my CI pipeline with unexpected prompts or output.

https://groups.google.com/g/kubernetes-dev/c/y4Q20V3dyOk

https://github.com/kubernetes/kubectl/issues/524

#### Story 3

[UNRESOLVED] As a user I would like to use different preferences per context.

#### Story 4

As a user I would like to be able to opt out of deprecation warnings.

https://github.com/kubernetes/kubectl/issues/1317

#### Story 5

As a user I would like to be able to prevent the execution of untrusted binaries by the client-go credential plugin system.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

#### Open Questions

1. [How are subcommands indicated for command overrides?](https://github.com/kubernetes/enhancements/pull/3392#discussion_r896174406)
1. [How are subcommand aliases indicated?](https://github.com/kubernetes/enhancements/pull/3392#discussion_r896179267)
1. [How do we handle tying these settings to cluster contexts?](https://github.com/kubernetes/enhancements/pull/3392#discussion_r898239057)
1. [Do we want this file to live elsewhere i.e. XDG_CONFIG?](https://github.com/kubernetes/enhancements/pull/3392#discussion_r896177353)
1. [How do we exectue subcommands and do we want to support variable substitution i.e. `$1`](https://github.com/kubernetes/enhancements/pull/3392#discussion_r898227148)

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Confusing users with a new config file | Low | Documentation and education |

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

For beta this feature will be enabled by default. However, users can disable it
by setting the `KUBECTL_KUBERC` environment variable to `false`. Additionally,
users can always set the `KUBERC` environment variable to `off`, which disables
the preference mechanism at any point in time.

By default, the configuration file will be located in `~/.kube/kuberc`. A flag
will allow overriding this default location with a specific path, for example:
`kubectl --kuberc /var/kube/rc`.

The following top-level keys are proposed, alongside the kubernetes `metav1.TypeMeta`
fields (`apiVersion`, `kind`):

* `aliases` Allows users to declare their own command aliases, including options and values.
* `defaults` Enables users to set default options to be applied to commands.
* `credentialPluginPolicy` (available in **kubectl.config.k8s.io/v1beta1**)
  Allows users to deny all, alow all, or allow some client-go credential plugins.
* `credentialPluginAllowlist` (available in **kubectl.config.k8s.io/v1beta1**)
  Enables users to specify criteria for trusting binaries to be executed by
  the client-go credential plugin system.

`aliases` will not be permitted to override built-in commands but will take
precedence over plugins (builtins -> aliases -> plugins). Any additional options
and values will be appended to the end of the aliased command. Users are
responsible for defining aliases with this behavior in mind.

`defaults` is designed based on the principle that all configurable behavior is
initially implemented as options. This design decision was made after analyzing the
intended behavior and realizing that targeting options effectively addresses the
use cases. During command execution, a merge will be occur, with inline overrides
taking precedence over the defaults.


```
apiVersion: kubectl.config.k8s.io/v1beta1
kind: Preference

aliases:
  - name: getdbprod
    command: get
    prependArgs:
    - pods
    options:
    - name: labels
      default: what=database
    - name: namespace
      default: us-2-production

defaults:
  - command: apply
    options:
      - name: server-side
        default: "true"
  - command: delete
    options:
      - name: interactive
        default: "true"

credentialPluginPolicy: Allowlist
credentialPluginAllowlist:
    - command: cloudplatform-credential-helper
    - command: custom-credential-script
```

### Kubectl Kuberc Management Command (kubectl kuberc)

In alpha (and initially in beta), users are expected to generate their kuberc files manually. However, this operation is error-prone and cumbersome.
The lack of kubectl command which operates on kuberc file makes the adoption of this feature significantly difficult.
Therefore, this section proposes new kubectl command, namely `kubectl kuberc`.

`kubectl kuberc` is the main command serving as an entry point to the subcommands similar to how `kubectl create` is designed.
Invocation of `kubectl kuberc` prints the subcommands. 
Currently, there are two subcommands (but this can be extended in the future, when more functionality is added to kuberc).
All the subcommands accept `kuberc` flag to explicitly specify the kuberc file to be updated. File priority order is the same with
kuberc execution:

* If `--kuberc` flag is specified, operate on this file
* If `KUBERC` environment variable is specified, operate on this file
* If none, operate on default kuberc (i.e. `$HOME/.kube/kuberc`).

This command and subcommands are marked as alpha initially. They can be executed under `kubectl alpha`, until they are promoted to beta.

### kubectl kuberc view

`kubectl kuberc view` subcommand prints the defined kuberc file content in the given format via `--output` flag (default is yaml).

### kubectl kuberc set --section defaults

`kubectl kuberc set --section defaults` subcommand creates/updates the default values of commands. It has the following flags;

#### command

`kubectl kuberc set --section defaults` command validates the presence of the command given via flag `--command`.
This flag can contain subcommands as well. Examples might be `--command=apply`, `--command="create role"`.

#### option

`--option` flag accepts list of options. We may or may not validate the presence of the flag name in the given command. 
But it is up to user to set the correct default value in correct type. Therefore, default field of the options is arbitrary.
Examples might be `--option="server-side=true"`, `--option="namespace=test"`.

Although kuberc supports short versions of flags (e.g. `-n test`), 
this flag forces users to enter options in standardized format `--option=$flag_name=$flag_value`. 
This gives us the opportunity to standardize kuberc files. 

#### overwrite

By default, this command errors out, if it finds a section of same command and same flag that is executed. `--overwrite` flag
is used to update this section.

### kubectl kuberc set --section aliases

`kubectl kuberc set --section aliases` defines alias definitions of a command and a set of flag options. It has these flags;

#### name

This required field is to define the name of the alias. This is inherently arbitrary field based on the preferences of the user.

#### command

`kubectl kuberc set --section aliases` command validates the presence of the command given via flag `--command`.
This flag can contain subcommands as well. Examples might be `--command=apply`, `--command="create role"`.

#### option

`--option` flag accepts list of options. We may or may not validate the presence of the flag name in the given command.
But it is up to user setting the correct default value in correct type. Therefore, default field of the options is arbitrary.
Examples might be `--option="server-side=true"`, `--option="namespace=test"`.

Although kuberc supports short versions of flags (e.g. `-n test`),
this flag forces users to enter options in opinionated format `--option=$flag_name=$flag_value`.
This gives us the opportunity to standardize kuberc files.

#### prependarg

`--prependarg` is an arbitrary list of strings that accepts anything in a string array format.

#### appendarg

`--appendarg` is an arbitrary list of strings that accepts anything in string array format.

### Allowlist Design Details

`credentialPluginAllowlist` allows the end-user to provide an array of objects
describing required conditions for executing a credential plugin binary. The
overall result of a check against the allowlist will be the logical OR of the
individual checks against the allowlist's entries. Each allowlist entry MUST
have at least one nonempty field describing the conditions required for the
plugin's execution. If multiple fields are specified within an entry, the
binary in question must meet all of the required conditions in that entry in
order to be executed (i.e. they are combined with logical AND).

Each element in the allowlist is a set of criteria; if the binary in question
meets all of the criteria in at least one **set** of criteria, the plugin will
be allowed to execute.  If no criteria set succeeds after comparing the binary
to all sets of criteria, the operation will be immediately aborted and an error
returned.

At the outset, the entry object will have only one field, `command`. The path of
the binary specified in the kubeconfig will be compared against that named in
the `command` field. This field may contain a basename, or the full path of a
plugin. To ensure an exact match, `exec.LookPath` will be called on both the
`command` field and the binary named in the kubeconfig. The resulting absolute
paths must match. The following table illustrates this:


| Scenario | `PATH=` | Allowlist `command` | `exec.LookPath(allowlist.Command)` | kubeconfig `command` | `exec.LookPath(execConfig.Command)` | success? |
|----------|---------|---------------------|------------------------------------|----------------------|-------------------------------------|----------|
| kubeconfig lists full path; `my-binary` is in both `/usr/local/bin` and `/usr/bin` | `PATH=/usr/local/bin:/usr/bin:<...>`  | my-binary | /usr/local/bin/my-binary | /usr/bin/my-binary | /usr/bin/my-binary | false |
| kubeconfig lists full path; `my-binary` is only in `/usr/local/bin` | `PATH=/usr/local/bin:/usr/bin:<...>`  | my-binary | /usr/local/bin/my-binary | /usr/bin/my-binary | /usr/bin/my-binary | false |
| kubeconfig lists full path; `my-binary` is only in `/usr/bin` | `PATH=/usr/local/bin:/usr/bin:<...>`  | my-binary | /usr/bin/my-binary | /usr/bin/my-binary | /usr/bin/my-binary | true |
| kubeconfig lists full path; `my-binary` is only in `/usr/bin` | `PATH=/usr/local/bin:/usr/bin:<...>`  | /usr/bin/my-binary | /usr/bin/my-binary | /usr/bin/my-binary | /usr/bin/my-binary | true |
| kuberc lists full path; `my-binary` is only in `/usr/bin` | `PATH=/usr/local/bin:/usr/bin:<...>`  | /usr/bin/my-binary | /usr/bin/my-binary | my-binary | /usr/bin/my-binary | true |
| kuberc lists full path; `my-binary` is in `/usr/local/bin` | `PATH=/usr/local/bin:/usr/bin:<...>`  | /usr/bin/my-binary | /usr/bin/my-binary | my-binary | /usr/local/bin/my-binary | false |
| neither lists full path; `my-binary` is in `/usr/bin`; equivalent to basename match | `PATH=/usr/local/bin:/usr/bin:<...>`  | my-binary | /usr/bin/my-binary | my-binary | /usr/bin/my-binary | true |

If `credentialPluginPolicy` is set to `Allowlist`, but a
`credentialPluginAllowlist` is not provided, it will be considered an
configuration error. Rather than guess at what the user intended, the operation
will be aborted just before the `exec` call. An error describing the
misconfiguration will be returned. This is because the allowlist is a security
control, and it is likely the user has made a mistake. Since the output may be
long, it would be easy for a security warning to be lost at the beginning of
the output. An explicitly empty allowlist (i.e. `credentialPluginAllowlist: []`),
in combination with `credentialPluginPolicy: Allowlist` will be considered an
error for the same reason. The user should instead use `credentialPluginPolicy:
DisableAll` in this case.

Commands that don't create a client, such as `kubectl config view` will not be
affected by the allowlist. Additionally, commands that create but do not *use*
a client (such as commands run with `--dry-run`) will likewise remain
unaffected.

In future updates, other allowlist entry fields MAY be added. Specifically,
fields allowing for verification by digest or public key have been discussed.
The initial design MUST accommodate such future additions.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

We're planning to expand tests adding:
- config API fuzzy tests
- cross API config loading
- input validation and correctness
- simple e2e using kuberc

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

- `<package>`: `<date>` - `<test coverage>`

-->

- `k8s.io/kubectl/pkg/cmd/`: `2025-05-13` - `57.0%`
- `k8s.io/kubectl/pkg/config/`: `2025-05-13` - `0.0%`
- `k8s.io/kubectl/pkg/kuberc/`: `2025-05-13` - `64.5%`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

- <test>: <link to test coverage>

-->

- [test-cmd.run_kuberc_tests](https://github.com/kubernetes/kubernetes/blob/fd15e3fd5566fb0a65ded1883fbf51ce7a68fe28/test/cmd/kuberc.sh): [integration cmd-master](https://testgrid.k8s.io/sig-release-master-blocking#cmd-master)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

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
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
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

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

#### Alpha

- Initial implementation behind `KUBECTL_KUBERC` environment variable.

#### Beta

- Gather feedback from developers and shape API appropriately.
- Decide if we want to do support configs per context.

#### GA

- Address feedback.


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
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

This feature will follow the [version skew policy of kubectl](https://kubernetes.io/releases/version-skew-policy/#kubectl).

Furthermore, kubectl will be equipped with a mechanism which will allow it to
read all past versions of the kuberc file, and pick the latest known one.
This mechanism will ensure that users can continue using whatever version of
kuberc they started with, unless they are interested in newer feature available
only in newer releases.

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
- [x] Other
  - Describe the mechanism: The environment variable `KUBECTL_KUBERC=true`.
  - Will enabling / disabling the feature require downtime of the control
    plane? No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes the environment variable can be unset and `~/.kube/kuberc` can be removed.

###### What happens if we reenable the feature if it was previously rolled back?

The new feature is enabled.

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

Yes, there is a [kuberc CLI test](https://github.com/kubernetes/kubernetes/blob/06c196438acb771d26ff983ff0f18a611acba208/test/cmd/kuberc.sh#L153-L156),
which verifies that `--kuberc` is used when the `KUBECTL_KUBERC` is on, and is
ignored when it's turned off.

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

Not applicable.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

Not applicable.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Not applicable.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

Not applicable.

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

Not applicable.

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
- [x] Other (treat as last resort)
  - Details: The command will be logged with kubectl -v 2

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

Not applicable.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

Not applicable.

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

Not applicable.

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

No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

There will be a new type similar to a kubeconfig - not in the API server.

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

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No.

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

`kubectl` is not resilient to API server unavailability.

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

Not applicable.

###### What steps should be taken if SLOs are not being met to determine the problem?

Not applicable.

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

* 2021-06-02: [Proposal to add delete confirmation](https://github.com/kubernetes/enhancements/issues/2775)
* 2022-06-13: This KEP created.
* 2024-06-07: Update KEP with new env var name and template.
* 2025-05-13: Update KEP for beta promotion.

## Drawbacks

None considered.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

None considered.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

Not applicable.
