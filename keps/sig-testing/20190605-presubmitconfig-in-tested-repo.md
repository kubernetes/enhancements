---
title: Presubmit config inside the tested repo
authors:
  - "@alvaroaleman"
owning-sig: sig-testing
participating-sigs:
  - sig-testing
reviewers:
  - "@stevekuznetsov"
  - "@cjwagner"
approvers:
  - "@stevekuznetsov"
  - "@cjwagner"
editor: TBD
creation-date: 2019-06-04
last-updated: 2019-07-24
status: implementable
---

# Presubmit config inside the tested repo


## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Security](#security)
    - [Components that need the <code>Presubmit</code> configuration but do not have a <code>git ref</code> to work on](#components-that-need-the--configuration-but-do-not-have-a--to-work-on)
- [Implementation History](#implementation-history)
<!-- /toc -->

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
* If the `test-infra` repository is not public, outside collaborators are unable to change job configs. This
  may happen if an organization that uses Prow has a mixture of public and private repositories and chooses
  not to bear the maintenance overhead of multiple Prow instances.


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
* Pull Requests on which an error occurred during parsing, defaulting or validation of presubmits that
  are managed in the code repository are not considered as merge candidates by Tide
* Tide executes the presubmits that are defined inside the code repository when it re-tests
* Renamed blocking presubmits added via pull request trigger a migration on all in-flight PRs
* Removed blocking presubmits via pull request trigger a status retire on all in-flight PRs
* All existing functionality except for what is listed in the [Risks and Mitigations](#Risks-and-Mitigations) section will continue to work when `inrepoconfig` is enabled
* All existing functionality will continue to work when `inrepoconfig` is not enabled

### Non-Goals

* The option to configure Postsubmits or Periodics inside the tested repository. This may be
  done in a future iteration.

## Proposal

It is proposed to introduce a the option to configure additional presubmits inside
the code repositories that are tested by prow via a file named `prow.yaml`.

This requires to change the existing `Config` struct to not expose a `Presubmits`
property anymore, but instead getter functions to get all Presubmits with the
ones from the `prow.yaml` added, if applicable.

Additionally, all components that need to access the `Presubmit` configuration need
to be changed to use the new getters and  to contain a git client which can be used
to fetch the `Presubmit` config from inside the repo.

### Risks and Mitigations

#### Security

The current attack vector to get credentials out of Prow is a rogue pull requestor
changing the scripts that are being executed during a test run to print or upload
credentials that are passed into the job.

With `inrepoconfig` a pull requestor could additionally create new Jobs that use
credentials that are previously not passed into any job of the repo or that are
executed on a different Kubernetes cluster which contains higher-privilege credentials.

Both the exiting attack vector and the changes introduced via `inrepoconfig` require
the pull requestor to be an org member or to get an org member to approve the pull
request for testing.

There are several possible approaches to mitigate the added security risk, for
example:

* Extend the configuration for `inrepoconfig` to allow/deny specific values for
	various job properties. Easy to setup, but requires code for every possible
	property
* Maintain an allow/deny list for users that are allowed to change job configs
* Allow operators to configure a webhook, which will then receive all pull request
	events and their changes to `prow.yaml`. The webhook can then allow or deny that.
	This is the most flexible solution and would even allow to connect the permission
	management for `inrepoconfig` to a third-party system like LDAP. It has the drawback
	that its more complicated to set up and introduces a new SPOF into Prow

Finding the best solution to mitigate the security risk added by `inrepoconfig` will
not be part of its first iteration, because that problem is considered to be much
easier to solve than finding an agreeable solution on how to implement `inrepoconfig`
itself.

#### Components that need the `Presubmit` configuration but do not have a `git ref` to work on

Components that need the `Presubmit` config but do not have a git reference at hand
can not work as before with `inrepoconfig` because the list of Presubmits depends on
the `ref`. This limitation will be documented.

## Implementation History

* A basic but functioning [prototype](https://github.com/kubernetes/test-infra/pull/12836)
  for this feature was created that served as initial basis for this KEP.
* A non-working [sketch pull request](https://github.com/kubernetes/test-infra/pull/13342) that shows which parts of Prow need to be touched
	and how the signatures for the newly-added functions look like was created to
	be the basis for a discussion on how exactly an implementation could look like
* Current work is being tracked via a [GitHub tracking issue](https://github.com/kubernetes/test-infra/issues/13370)
