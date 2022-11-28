<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

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
# KEP-3655: Kubectl Translation Improvement

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
  - [1. Make it simple for people to update translations.](#1-make-it-simple-for-people-to-update-translations)
  - [2. Identify and empower contributors who will be approvers and reviewers for each locale.](#2-identify-and-empower-contributors-who-will-be-approvers-and-reviewers-for-each-locale)
  - [3. Improve documentation.](#3-improve-documentation)
  - [4. Ensure all strings in code support translations.](#4-ensure-all-strings-in-code-support-translations)
  - [5. Establish metrics to track the state of translation completeness.](#5-establish-metrics-to-track-the-state-of-translation-completeness)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [BACKGROUND: How kubectl translations work.](#background-how-kubectl-translations-work)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [1. DETAILS: Make it simple for people to update translations.](#1-details-make-it-simple-for-people-to-update-translations)
  - [2. DETAILS: Identify and empower contributors who will be approvers and reviewers for each locale.](#2-details-identify-and-empower-contributors-who-will-be-approvers-and-reviewers-for-each-locale)
  - [3. DETAILS: Improve documentation.](#3-details-improve-documentation)
  - [4. DETAILS: Ensure all strings in code support translations.](#4-details-ensure-all-strings-in-code-support-translations)
  - [5. DETAILS: Establish metrics to track the state of translation completeness.](#5-details-establish-metrics-to-track-the-state-of-translation-completeness)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

This KEP proposes improvements to the process used to maintain kubectl's translations:
1. Make it simple for people to update translations.
2. Identify and empower contributors who will be approvers and reviewers for each locale.
3. Improve documentation.
4. Ensure all strings in code support translations.
5. Establish metrics to track the state of translation completeness.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Kubectl has built-in support for translating output based on the user's locale. Primarily this is used to translate command and flag help text, but also can include other text written by kubectl commands such as informational messages, warnings, and errors.

Over time, however, the translations have not been maintained, resulting in the output reverting to the default non-translated English text.

As a result, if you run a command in another language, most of the output is untranslated. Below is the output of `kubectl help` using the `ja_JP` (Japanese) locale as of kubectl 1.26.0-alpha. As you can see, there are some descriptions that are translated, but over 90% of the output remains in English.
```
$ LANG=ja_JP.UTF-8 kubectl help
kubectl controls the Kubernetes cluster manager.

 Find more information at: https://kubernetes.io/docs/reference/kubectl/

Basic Commands (Beginner):
  create          Create a resource from a file or from stdin
  expose          Take a replication controller, service, deployment or pod and expose it as a new Kubernetes service
  run             Run a particular image on the cluster
  set             Set specific features on objects

Basic Commands (Intermediate):
  explain         Get documentation for a resource
  get             1つまたは複数のリソースを表示する
  edit            Edit a resource on the server
  delete          Delete resources by file names, stdin, resources and names, or by resources and label selector

Deploy Commands:
  rollout         Manage the rollout of a resource
  scale           Set a new size for a deployment, replica set, or replication controller
  autoscale       Auto-scale a deployment, replica set, stateful set, or replication controller

Cluster Management Commands:
  certificate     Modify certificate resources.
  cluster-info    Display cluster information
  top             Display resource (CPU/memory) usage
  cordon          Mark node as unschedulable
  uncordon        Mark node as schedulable
  drain           Drain node in preparation for maintenance
  taint           Update the taints on one or more nodes

Troubleshooting and Debugging Commands:
  describe        Show details of a specific resource or group of resources
  logs            Print the logs for a container in a pod
  attach          Attach to a running container
  exec            Execute a command in a container
  port-forward    Forward one or more local ports to a pod
  proxy           Run a proxy to the Kubernetes API server
  cp              Copy files and directories to and from containers
  auth            Inspect authorization
  debug           Create debugging sessions for troubleshooting workloads and nodes

Advanced Commands:
  diff            Diff the live version against a would-be applied version
  apply           Apply a configuration to a resource by file name or stdin
  patch           Update fields of a resource
  replace         Replace a resource by file name or stdin
  wait            Experimental: Wait for a specific condition on one or many resources
  kustomize       Build a kustomization target from a directory or URL.

Settings Commands:
  label           リソースのラベルを更新する
  annotate        リソースのアノテーションを更新する
  completion      Output shell completion code for the specified shell (bash, zsh, fish, or powershell)

Other Commands:
  alpha           Commands for features in alpha
  api-resources   Print the supported API resources on the server
  api-versions    Print the supported API versions on the server, in the form of "group/version"
  config          kubeconfigを変更する
  plugin          Provides utilities for interacting with plugins
  version         Print the client and server version information

Usage:
  kubectl [flags] [options]

Use "kubectl <command> --help" for more information about a given command.
Use "kubectl options" for a list of global command-line options (applies to all commands).
```

The above example is for the `ja_JP` locale, but the result is similar for the other locales as well.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

The goal of this KEP is to define and coordinate the work that needs to be done in order to:

1. Improve translation completeness.
2. Ensure translations remain complete in the future.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

1. Adding or Removing locales from the supported list. 
   * While there may be merit to the argument that the list of supported locales should be changed, it is beyond the scope of this KEP and should be handled separately.
2. Rewriting kubectl's i18n code or making significant changes to how it works.
   * There may be some minor changes needed, but the goal of this KEP is to improve on what we already have in place.
3. Translating text that is generated from the API, controllers, plugins, or other sources outside kubectl. 
   * While some translation might be possible from these sources, it can be complex due to their dynamic nature and is out of scope for this KEP.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### 1. Make it simple for people to update translations.

To update translations today, you first need to install some prerequisites and run a script in the hack directory before you can do what you really want to do, which is to translate text.

This shouldn't be required.

This KEP will define how the translation files can be kept up-to-date with new and updated text placeholders from the code, so that translation contributors do not need to perform this unrelated task and the process becomes simpler for them.

### 2. Identify and empower contributors who will be approvers and reviewers for each locale.

Each locale should have trusted contributors who can function as reviewers and approvers for its translations. We can manage this through the use of directory-specific [OWNERS](https://www.kubernetes.dev/docs/guide/owners/) files.

This KEP will specify who should be added to the OWNERS file for each locale.

### 3. Improve documentation.

Currently, the documentation on how to contribute as a translator exists within the kubectl source code in a README.md. This document is not easy to discover and not friendly to first time contributors.

This KEP will call for the documentation to be updated and for it to be linked from a new, more prominent location where contributors can learn how to translate kubectl's text.

### 4. Ensure all strings in code support translations.

There are two factors contributing to the problem of incomplete translations:
1. Text that is available to be translated, but has not been.
2. Text that is not available for translation because it has not yet been wrapped in `i18n.T()`.

So far, this KEP has focused on the first aspect, but the second aspect must also be addressed.

As part of the effort to implement this KEP, we will wrap all translatable text in `i18n.T()` where possible within kubectl.

### 5. Establish metrics to track the state of translation completeness.

Without metrics, it will be difficult to know whether we have succeeded at the goal of improving translation completeness, but more importantly, metrics will provide a way for us to make sure we are aware of translation completeness for each locale in the future so that we can take action if it starts to diminish.

This KEP will define how metrics will be generated and used to ensure that it is easy to stay aware of the completeness of all locale translations.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

#### Story 2

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

#### BACKGROUND: How kubectl translations work.

The code to implement translations is located in kubectl's [`i18n`](https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io/kubectl/pkg/util/i18n) package. This package contains some utility functions to load and apply translations, and the locale-specific translation files, which are embedded in the kubectl executable.

To indicate that a specific string should be translated, the developer simply needs to wrap the English string in a call to the `i18n.T()` utility function.

For example, in [get.go](https://github.com/kubernetes/kubernetes/blob/8fb423bfabe0d53934cc94c154c7da2dc3ce1332/staging/src/k8s.io/kubectl/pkg/cmd/get/get.go#L163-L176), the following code wraps the `Short` field's value with a call to `i18n.T()`:

```
	cmd := &cobra.Command{
		Use:                   fmt.Sprintf("get [(-o|--output=)%s] (TYPE[.VERSION][.GROUP] [NAME | -l label] | TYPE[.VERSION][.GROUP]/NAME ...) [flags]", strings.Join(o.PrintFlags.AllowedFormats(), "|")),
		DisableFlagsInUseLine: true,
		Short:                 i18n.T("Display one or many resources"),
		Long:                  getLong + "\n\n" + cmdutil.SuggestAPIResources(parent),
		Example:               getExample,
		// ValidArgsFunction is set when this function is called so that we have access to the util package
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(o.Complete(f, cmd, args))
			cmdutil.CheckErr(o.Validate())
			cmdutil.CheckErr(o.Run(f, cmd, args))
		},
		SuggestFor: []string{"list", "ps"},
	}
```

The text passed to `i18n.T()` is not required to be single line of text. It can be multiple lines, like [this example](https://github.com/kubernetes/kubernetes/blob/8fb423bfabe0d53934cc94c154c7da2dc3ce1332/staging/src/k8s.io/kubectl/pkg/cmd/get/get.go#L90-L99):
```
	getLong = templates.LongDesc(i18n.T(`
		Display one or many resources.

		Prints a table of the most important information about the specified resources.
		You can filter the list using a label selector and the --selector flag. If the
		desired resource type is namespaced you will only see results in your current
		namespace unless you pass --all-namespaces.

		By specifying the output as 'template' and providing a Go template as the value
		of the --template flag, you can filter the attributes of the fetched resources.`))
```

Kubectl uses the [chai2010/gettext-go](https://github.com/chai2010/gettext-go), which is a Go library for [GNU GetText](https://www.gnu.org/software/gettext), to perform translations.

When kubectl starts, one of the first things it does is call [`i18n.LoadTranslations()`](https://github.com/kubernetes/kubernetes/blob/feca7983b77be3d7d578f3d5b64cbb1be6f327af/staging/src/k8s.io/kubectl/pkg/util/i18n/i18n.go#L94) to [determine your locale, load the translation files specific to your locale, and configure gettext-go to use your locale](https://github.com/kubernetes/kubernetes/blob/feca7983b77be3d7d578f3d5b64cbb1be6f327af/staging/src/k8s.io/kubectl/pkg/util/i18n/i18n.go#L99-L130). Kubectl determines your locale by looking at the [`LC_ALL`, `LC_MESSAGES`, and `LANG` environment variables, in that order of precedence](https://github.com/kubernetes/kubernetes/blob/feca7983b77be3d7d578f3d5b64cbb1be6f327af/staging/src/k8s.io/kubectl/pkg/util/i18n/i18n.go#L58-L64).

Once gettext-go has been configured to use your locale's translations, any call to `i18n.T()` will replace the original English text with the translated text for your locale. The translations for each locale are stored as subdirectories within kubectl's [pkg/util/i18n/translations/kubectl](https://github.com/kubernetes/kubernetes/tree/1c4387c78f0d48398efb0dcc3268fa156cdd8ffd/staging/src/k8s.io/kubectl/pkg/util/i18n/translations/kubectl) directory.

Each locale has two files: a `k8s.po` file and a `k8s.mo` file, so the directory structure looks like this:

```
kubectl
├── de_DE
│   └── LC_MESSAGES
│       ├── k8s.mo
│       └── k8s.po
├── default
│   └── LC_MESSAGES
│       ├── k8s.mo
│       └── k8s.po
├── en_US
│   └── LC_MESSAGES
│       ├── k8s.mo
│       └── k8s.po
├── fr_FR
│   └── LC_MESSAGES
│       ├── k8s.mo
│       └── k8s.po
├── it_IT
│   └── LC_MESSAGES
│       ├── k8s.mo
│       └── k8s.po
├── ja_JP
│   └── LC_MESSAGES
│       ├── k8s.mo
│       └── k8s.po
├── ko_KR
│   └── LC_MESSAGES
│       ├── k8s.mo
│       └── k8s.po
├── pt_BR
│   └── LC_MESSAGES
│       ├── k8s.mo
│       └── k8s.po
├── zh_CN
│   └── LC_MESSAGES
│       ├── k8s.mo
│       └── k8s.po
└── zh_TW
    └── LC_MESSAGES
        ├── k8s.mo
        └── k8s.po
```

The `.po` file and `.mo` files are [file types defined by GNU Gettext](https://www.gnu.org/software/gettext/manual/html_node/Files.html):
> The letters PO in .po files means Portable Object, to distinguish it from .mo files, where MO stands for Machine Object.

> PO files are meant to be read and edited by humans, and associate each original, translatable string of a given package with its translation in a particular target language.

> MO files are meant to be read by programs, and are binary in nature.

Contributors performing translations do so by editing the `k8s.po` file, either manually, or more likely using a tool like [`poedit`](https://poedit.net) which presents the source and translated text side-by-side, making it easy to see and update translations. If you use poedit, the `k8s.mo` file is updated automatically, or you can run `hack/update-translations.go` which will regenerate all of the `k8s.mo` files.

Here is an example excerpt from [`ja_JP/LC_MESSAGES/k8s.po`](https://github.com/kubernetes/kubernetes/blob/33e6ebc8f8df47864a77c867d78216adb70cd79d/staging/src/k8s.io/kubectl/pkg/util/i18n/translations/kubectl/ja_JP/LC_MESSAGES/k8s.po#L462-L465) showing the translation for the English text `Display one or many resources`:
```
# https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/get.go#L107
#: staging/src/k8s.io/kubectl/pkg/cmd/get/get.go:165
msgid "Display one or many resources"
msgstr "1つまたは複数のリソースを表示する"
```

When `i18n.T("Display one or many resources")` is called, `Display one or many resources` is replaced by `1つまたは複数のリソースを表示する`.

If the translation does not exist for a given English string, no translation is performed, and the original string is returned as-is. This is why translations can start out as fully complete, yet degrade over time as the English text is changed and new translations are not performed.

<<[UNRESOLVED TODO]>>

TODO: 
* Do both the `k8s.po` and `k8s.mo` files need to be embedded in kubectl, or can we only embed `k8s.mo`?  We should figure this out and not embed `k8s.po` unless necessary.

<</[UNRESOLVED]>>

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### 1. DETAILS: Make it simple for people to update translations.

Currently, translation contributors need to run `hack/update-translations.sh -x` to extract all strings wrapped in `i18n.T()` into a file called [`template.pot`](https://github.com/kubernetes/kubernetes/blob/33e6ebc8f8df47864a77c867d78216adb70cd79d/staging/src/k8s.io/kubectl/pkg/util/i18n/translations/kubectl/template.pot). This template can be synchronized with a locale's `k8s.po` file (using poedit for example) to bring in new text that needs to be translated.

The `hack/update-translations.sh` script has some prerequisites that must be installed and are unrelated to the job that the translation contributor is there to do. The usage of this script is documented, but not well enough for a first time contributor to be confident in running it.

Ideally however, translation contributors should not need to run any scripts or install any tools, other than a translation editor of their choice to update translations (like poedit).

To make sure that text is available to be translated without the translation contributor needing to run any additional processes, we will require developers to update the translation files as a presubmit condition of their pull request. This will be done through the use of a new script `hack/verify-translations.sh` which will fail if it detects out-of-date or missing translations. In this case, the developer will need to run `hack/update-translations.sh -x -k` to make sure the translation files are up-to-date.

As part of this effort, we should simplify `hack/update-translations.sh` so that you don't need to pass in flags and just make it work for the most common scenario by default (which I think is `-x -k`).

Alternative for consideration:
* Instead of having a presubmit job and requiring developers to keep the translation files up-to-date, we could run `hack/update-translations.sh` in a postsubmit job and commit any changes that occur back to the k/k repo (is that even possible?), or automatically open a pull request (more likely this will be the way).

### 2. DETAILS: Identify and empower contributors who will be approvers and reviewers for each locale.

Currently, the [OWNERS](https://github.com/kubernetes/kubernetes/blob/9682b7248fb69733c2a0ee53618856e87b067f16/staging/src/k8s.io/kubectl/pkg/util/i18n/translations/OWNERS) file under `i18n/translations` looks like this:

```
# See the OWNERS docs at https://go.k8s.io/owners

reviewers: []
approvers:
  - sig-cli-maintainers
emeritus_approvers:
  - brendandburns
```

Create an [OWNERS](https://www.kubernetes.dev/docs/guide/owners/) file under each locale's directory, enabling reviewers and approvers to be established specific to each locale, who are able to approve translations independent of the SIG-CLI reviewers and approvers, who may not have the ability to read the language being translated.

The new OWNERS file will use the [`no_parent_owners` option](https://www.kubernetes.dev/docs/guide/owners/#owners) to prevent SIG-CLI maintainers from being assigned to translation related PRs. If specific SIG-CLI maintainers are qualified to review and approve translation changes, they can be added separately. The purpose of this is to make sure that appropriate people are assigned to the language they are qualified to review/approve.

The new OWNERS file will use the [`labels` field](https://www.kubernetes.dev/docs/guide/owners/#owners) to automatically apply a new label called `area/translation`, which can be used to filter PRs to show only translation-related PRs.

Below will be the reviewers and approvers for each locale:

| Locale | Reviewers | Approvers |
|--------|-----------|-----------|
| de_DE  | @iamNoah1 | @iamNoah1 |
| fr_FR  |           |           |
| it_IT  |           |           |
| ja_JP  |           |           |
| ko_KR  |           |           |
| pt_BR  |           |           |
| zh_CN  |           |           |
| zh_TW  |           |           |

<<[UNRESOLVED TODO]>>

TODO: 
* Identify reviewers and approves for each locale and populate the able above

<</[UNRESOLVED]>>

### 3. DETAILS: Improve documentation.

There is a [`README.md`](https://github.com/kubernetes/kubernetes/blob/eb75b34394964fbc325ca08a5d0b126c5ea16b6f/staging/src/k8s.io/kubectl/pkg/util/i18n/translations/README.md) located within kubectl's i18 directory, which is helpful, but the process is not simple for a new or occasional contributor.

This document will be updated to remove any out-of-date information and add any new information.

We will also create a separate page on [k8s.dev](https://k8s.dev) and link to it from [Non-code Contributions](https://www.k8s.dev/docs/guide/non-code-contributions/), so that people who are interested in becoming a translator for kubectl can discover how to start.

### 4. DETAILS: Ensure all strings in code support translations.

Create an issue to track the work needed to wrap strings in `i18n.T()`.

Fix known issues affecting translation completeness:
* https://github.com/kubernetes/kubectl/issues/1326
* https://github.com/kubernetes/kubectl/issues/1327

<<[UNRESOLVED TODO]>>

TODO:
* There was already an issue for this, but it is closed now: https://github.com/kubernetes/kubectl/issues/910
* There is a [script called `extract.py` in the translations directory](https://github.com/kubernetes/kubernetes/blob/5ece28b77a284b24b674278378630373196789ac/staging/src/k8s.io/kubectl/pkg/util/i18n/translations/extract.py), but it doesn't seem to work anymore.
  * Should we try to get this back in working order?
  * How can it be used? Can we make a `hack/extract-translations.sh` or something like that to standardize its usage?

<</[UNRESOLVED]>>

### 5. DETAILS: Establish metrics to track the state of translation completeness.

In order to understand the current state of translations and track progress or regression over time, we need to establish a way to measure how complete the translations are for each locale.

Prototype utility to analyze `.po`/`.pot` files to determine translation coverage:  
https://github.com/brianpursley/gettext-report

<<[UNRESOLVED TODO]>>

TODO:
* How can the process be run automatically? 
* Should the data be stored over time or just be presented as a snapshot in time? 
  * If stored over time, where can this be stored? 
  * Are there other things in the k8s codebase that generate metrics that can be used as an example?
* Can we use periodic tests that run and fail if there are any incomplete translations?
  * Is the threshold 100% or some other specific %? 
  * Can (or should) they be part of the CI signal for k8s release?

<</[UNRESOLVED]>>

___
### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

* [x] I/we understand the owners of the involved components may require updates to
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

n/a

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

Currently, no integration tests exist to test translations.

However, we should be able to add some basic integration tests around translations, not to test specific text is translated, but to test the translation mechanism to make sure it is working properly.

Integration tests will be created to test kubectl help output for each of the supported locales, using each of the environment variables (`LC_ALL`, `LC_MESSAGES`, and `LANG`), and verifying that some chosen text (TBD) has been translated.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

n/a

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

In general, we try to use the same stages (alpha, beta, GA), regardless of how the
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

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

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

n/a

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

n/a

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

n/a

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

The KEP must have an approver from the
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

<!--
- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
-->

n/a

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

TODO:

###### What happens if we reenable the feature if it was previously rolled back?

TODO:

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

No

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

TODO:

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

TODO:

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

TODO:

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No 

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

n/a

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

<!--
- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:
-->

n/a

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

n/a

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

<!--
- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:
-->

n/a

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

No

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

n/a

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

No

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

No

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No

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

Translations are entirely contained within kubectl, so there is no API or etcd dependency.

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

None

###### What steps should be taken if SLOs are not being met to determine the problem?

n/a

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

TODO:

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

TODO: 

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

TODO: 
