---
title: Presubmit config inside the tested repo
authors:
  - "@alvaroaleman"
owning-sig: sig-testing
participating-sigs:
  - sig-testing
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-06-04
last-updated: 2019-06-05
status: provisional
---

# Presubmit config inside the tested repo


## Table of Contents


* [Presubmit config inside the tested repo](#presubmit-config-inside-the-tested-repo)
* [Table of Contents](#table-of-contents)
* [Release Signoff Checklist](#release-signoff-checklist)
* [Summary](#summary)
* [Motivation](#motivation)
   * [Goals](#goals)
   * [Non-Goals](#non-goals)
* [Proposal](#proposal)
   * [prow/config adjustments](#prowconfig-adjustments)
   * [New prow/inrepoconfig/api package](#new-prowinrepoconfigapi-package)
   * [New prow/inrepoconfig package](#new-prowinrepoconfig-package)
   * [Extend prow/config with configuration to opt-in to this feature](#extend-prowconfig-with-configuration-to-opt-in-to-this-feature)
   * [Add code to tide and the trigger plugin so they will use presubmits from a prow.yaml if the feature is activated](#add-code-to-tide-and-the-trigger-plugin-so-they-will-use-presubmits-from-a-prowyaml-if-the-feature-is-activated)
* [Risks and Mitigations](#risks-and-mitigations)
   * [Plugins may have an inconsitent view on what presubmits exist for a repo](#plugins-may-have-an-inconsitent-view-on-what-presubmits-exist-for-a-repo)
   * [Someone could accidentally delete all presubmits in prow.yaml and have their PR instantly merged](#someone-could-accidentally-delete-all-presubmits-in-prowyaml-and-have-their-pr-instantly-merged)
* [Implementation History](#implementation-history)

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

This document proposes to change the Prow presubmit handling to optionally version some or
all presubmits in the same repository that also contains the code that is being tested.

## Motivation

Currently, all jobs for Prow are configured in a central `infra-config` repository that is
in most cases distinct from the repositories whose code is being tested in presubmits. This
poses severall challenges:

* It is not possible to see in a Pull Request that introduces a new presubmit if that presubmit will
  actually pass. The two workarounds for this are to either have both the job author and the reviewer
  use two CLIs to create a Pod from the job or to make the Job initially optional, verify it with a
  test Pull Request to the code repository, then make it mandatory. Both of these workarounds are
  cumbersome.
* The same issues apply when doing changes to the config of a presubmit
* When a project maintains multiple branches, e.G. because there are release branches, the
  maintainers must remember to create a copy of the presubmit in the `infra-config` repository.
  Otherwise it is easely possible that the presubmit becomes incompatible with one of the branches
  it is used in, because somone makes a change to its config and forgets to test against all branches.
  This is an additional step maintainers have to remember when branching off.


### Goals

* It is possible to version some or all presubmits of a given repository inside that repository in a
  `yaml` file
* This feature is optional and opt-in
* The triggering of presubmits on pull request creation or updates continues to work and includes the
  jobs that are managed inside the code repository
* Re-Running tests via the `/retest` command continues to work and includes the jobs that are
  managed inside the code repository
* Explicitly executing optional tests via the `/test <<myjob>>` command continues to work and includes
  all jobs that are managed inside the code repository
* All the existing defaulting and validation for Presubmit jobs is being used to default and validate
  jobs that are managed inside the code repository
* If there is an error parsing, defaulting or validating the presubmits that are managed inside the
  code repository, a comment will be posted to GitHub stating the error
* Pull Requests on which an error occurred during parsing, defaulting or validation the presubmits that
  are managed in the code repository are not consideres as merge candidates by Tide
* Tide executes the presubmits that are defined inside the code repository when it re-tests
* Renamed blocking presubmits added via pull request trigger a migration on all in-flight PRs
* Removed blocking presubmits via pull request trigger a status retire on all in-flight PRs

### Non-Goals

* The option to configure Postsubmits or Periodics inside the tested repository. This may be
  done in a future iteration.

## Proposal

It is proposed to introduce a the option to configure additional presubmits inside the code
repositories that are tested by prow via a file named `prow.yaml`

The minimum part of Prow this has to be integrated with is:

* The `trigger` plugin as it manages the triggering of Jobs upon Pull Request Events or commands
* `Tide`, as it re-excutes jobs when the base branch for a pull request changed after it was
  tested

The following changes will be required:

#### `prow/config` adjustments

The `prow/config` package needs to unite its various defaulting and
validation funcs for presubmits so that in the end a func can be
exported that will be used both in case of config reload and
when additional presubmits are loaded from a `prow.yaml`.

*Open question:* What would be the best signature for such a function?
It must be able to optionally accept additional presubmits for one repo

#### New `prow/inrepoconfig/api` package

A new package `prow/inrepoconfig/api` will be added which contains all
the functionality needed by everything that wants to make use of the
presubmits in `prow.yaml`. It will

* Export a `type InRepoConfig` that will be used to unmarshal the contents of `prow.yaml` into
* Export a `func New(log *logrus.Entry, c *config.Config, gc *git.Client, org, repo, baseRef string, headRefs []string) (*InRepoConfig, error)`

The `func New` uses the passed in `*git.Client` to fetch the `baseRef`
and all `headRefs` of the repo and merges them together using the
appropriate strategy. The appropriate strategy can be found out by leveraging `*Config.Tide.MergeMethod(org, repo)`.
Using the `*git.Client` has the advantage that it can do caching, which
is very important for bigger repositories.
After the merge was done, it is checked if a `prow.yaml` exists.
Its absence will not be considered an error and an empty `InrepoConfig`
will get returned.
If `prow.yaml` exists, it will be unmarshalled using `yaml.UnmarshalStrict`
to avoid human errors like typos in map keys. If that was successful,
all presubmits found will be defaulted and validated using the exported
logic from the `prow/config` package.

*Alternative:* The whole logic for fetching and parsing the `prow.yaml`
could be put into the `prow/config` package

#### New `prow/inrepoconfig` package

A new package `prow/inrepoconfig` will be added that exports a
`func HandlePullRequest(log *logrus.Entry, c *config.Config, ghc githubClient, gc *git.Client, pr github.PullRequest)`. It:
* Is intended to be used by the `trigger` plugin only
* Initially creates a github status context to prevent merges while
  fetching and parsing of `prow.yaml` is still in progress
* Fetches and parses the `prow.yaml` using the `prow/inrepoconfig/api`
  package
* Posts a comment to GitHub if an error occurs
* Updates the status context to `failure` or `success` after the
  `prow.yaml` parsing is done
* Has functionality to remove status contexts that existed in an older
  `baseRef` and got removed in a newer one. This could be done by:
    * Recognizing status contexts created by Prow  using a combination
      of a static list (e.G. `tide`) and for the remaining contexts by
      comparing their `targetURL` with what is configured in Prow as
      `jobURLPrefix`. If the `targetURL` starts with `jobURlPrefix`
      and the context does not belong to one of the known presubmits,
      the status context will be retired.
    * Leveraging the `status-reconciler`. The `status-reconciler`
      is edge-driven, meaning whenever there is a configuration change
      it will use the delta to find presubmits that got removed, then
      check on all pull requests if they contain a status context for
      a removed presubmit. To make this work for `prow.yaml`, we would
      have to extend the `status-reconciler` to get an event from GitHub
      whenever there is a pushevent to the base branch of any open
      pull request, so that it can find out if a presubmit defined in
      `prow.yaml` got removed


*Alternative:* This could be put into the `prow/plugins/trigger` package
 as that is the only consumer for this functionality

#### Extend `prow/config` with configuration to opt-in to this feature

This feature should be opt-in to avoid surprises. To achieve that, the
`ProwConfig` struct in the `prow/config` package has to be extend with
a `InRepoConfig     map[string]InRepoConfig` property. Also, a
`func (pc *ProwConfig) InRepoConfigFor(org, repo string) InRepoConfig`
has to be added that can be used to determine if this functionality is
enabled. Initially, the `InRepoConfig` struct will be defined as follows:

```
type InRepoConfig struct {
  Enabled bool `json:"enabled,omitempty"`
}
```

This allows to add further configuration options to this feature in the
future if needed.

#### Add code to `tide` and the `trigger` plugin so they will use presubmits from a `prow.yaml` if the feature is activated

Both `tide` and the `trigger` plugin will need some code that checks
if the `prow.yaml` feature is activated and if yes, amend the presubmits
from it to the presubmits that are configured in the `infra-config` repo.

In `tide` this is very simple, we can just use the `prow/inrepoconfig/api` package to figure out the additional presubmits.

*Open question:* What happens if the merged `prow.yaml` becomes invalid?
Is tide able to recover from that?

In the `trigger` plugin, we have to check if the `prow.yaml` feature is
enabled and if yes, use the functionality described from the new `prow/inrepoconfig`
package to fetch additional presubmits from it.


### Risks and Mitigations

#### Plugins may have an inconsitent view on what presubmits exist for a repo

Currently, the presubmit config is exported as a `map[string][]Presubmit`. With the approach outlined above, plugins have to opt-in
to see presubmits defined in `prow.yaml`.

A possible mitigation for this could be to unexport the `map[string][]presubmit`
and replace it with a `func (c* Config) Presubmits(org, repo, baseRef string, headRefs []string) ([]Presubmit, error)`.
This would require adjusments in all plugins that use the presubmit config.

*Open question with this approach:* What do we do about Prow components that
require the presubmit config but do not have a revision they work on,
like e.G `branchprotector`?

#### Someone could accidentally delete all presubmits in `prow.yaml` and have their PR instantly merged

Pull requests are expected to require a review and a reviewer is expected
to catch this situation. If there is reason to assume this may not always
work out, a presubmit could be added in the `infra-config` repo that checks
if the content of `prow.yaml` matches expectiations.

#### Security implications

Today its already possible for someone that can get their pull request tested to hijack credentials by
changing one of the scripts that is executed by a job that has access to credentials.

On top of that, these additional attack vectors would be introduced by a `prow.yaml` feature:
* It is possible to add additional credentials to a job
* It is possible to set very high resource requests on a job
* It is possible to create a high number of jobs, resulting in DOS on the Prow build cluster
* It is possible to configure arbitrary `build` clusters on a job

All of these scenarios still require to be an org member or having an org member
using `/ok-to-test`, thought.

In a future iteration we can think about configuration options to limit the credentials, resources,
number of jobs or build clusters that users can allocate via `prow.yaml`

## Implementation History

A [prototype](https://github.com/kubernetes/test-infra/pull/12836) for this feature was created that
served as basis for this KEP.
