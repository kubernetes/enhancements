# KEP-3744: Stay on supported go versions

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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
    - [Request the go team backport security fixes to more than two minor versions](#request-the-go-team-backport-security-fixes-to-more-than-two-minor-versions)
    - [Maintain a custom patched version of older go minor versions with security fix backports](#maintain-a-custom-patched-version-of-older-go-minor-versions-with-security-fix-backports)
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

- [ ] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] Production readiness review completed
- [ ] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes
  - https://github.com/kubernetes/kubernetes/issues/112408
  - [sig-release 2023-01-24](https://docs.google.com/document/d/1Fu6HxXQu8wl6TwloGUEOXVzZ1rwZ72IAhglnaAMCPqA/edit#bookmark=id.66qudq8j2af)
  - [sig-architecture 2023-01-26](https://docs.google.com/document/d/1BlmHq5uPyBUDlppYqAAzslVbAO8hilgjqZUTaNXUhKM/edit#bookmark=id.ab11sq962ek9)

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP describes a process for updating supported Kubernetes minor version
release-1.x branches to build and release with new minor versions of go,
with a focus on preserving behavior and compatibility of Kubernetes patch releases.

## Motivation

The go project has the following [release policies](https://github.com/golang/go/wiki/Go-Release-Cycle#release-maintenance):

* Release minor versions every ~6 months
* Support the last 2 minor versions with patch releases
  (each minor version of go has ~12 months of security fix support)

These policies are followed closely, resulting in [2 go minor versions per year since 2016](https://go.dev/doc/devel/release).

The Kubernetes project is built on go and has the following policies:

* Immediately test/adopt new minor versions of go on the main development branch.
  * This typically takes 0-2 months after a new go minor version is released.
    Sometimes we are able to adopt a release candidate for the go minor version,
    other times we have to wait for a regression to be fixed in a patch release (e.g. 
    [go1.10.1](https://github.com/golang/go/issues/23884),
    [go1.12.5](https://github.com/golang/go/issues/31679),
    [go1.13.4](https://github.com/golang/go/issues/35087),
    [go1.18.1](https://github.com/golang/go/issues/51852)).
* Release Kubernetes minor versions 3 times a year.
  * This can be misaligned with new go minor releases by 1-3 months.
* Support each Kubernetes minor version with patch releases for
  [up to 14 months](/keps/sig-release/1498-kubernetes-yearly-support-period/),
  including fixes for CVEs, dependency updates, and fixes for critical bugs.

The time to adopt new go minor versions and align with a Kubernetes release means
Kubernetes minor versions often start their 14-month support clock based
on a minor version of go with only 8-9 months of support left.

At the time this was written, exactly half of [all go patch releases](https://go.dev/doc/devel/release)
(84/168, 50%) contained fixes flagged as having possible security implications.
Even though many of these issues were not relevant to Kubernetes, many were,
so it remained important to use supported go versions that received those fixes.

An obvious solution would be to simply update Kubernetes release branches to new minor versions of go.
However, Kubernetes also [avoids destabilizing changes in patch releases](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-release/cherry-picks.md#what-kind-of-prs-are-good-for-cherry-picks),
and historically, this prevented updating release branches to new minor versions of go,
due to changes that were considered prohibitively complex, risky, or breaking. Examples include:

* go1.6: enabling http/2 by default
* go1.13: go module build and code generation changes
* go1.14: EINTR handling ([#92521](https://github.com/kubernetes/kubernetes/issues/92521))
* go1.17: dropping x509 CN support, ParseIP changes
* go1.18: disabling x509 SHA-1 certificate support by default
* go1.19: dropping current-dir LookPath behavior

Some of these changes could be easily mitigated in Kubernetes code,
some could only be opted out of via a user-specified GODEBUG envvar,
and others required invasive code changes or could not be avoided at all.

Because of this inconsistency, Kubernetes release branches historically remained on a single go minor version,
and risked being unable to pick up go security fixes for the last several months of each Kubernetes minor version.

Feedback to the go team about this situation prompted a [discussion](https://github.com/golang/go/discussions/55090),
[proposal](https://github.com/golang/go/issues/56986), [talk at GopherCon](https://www.youtube.com/watch?v=v24wrd3RwGo),
and a [design](https://go.dev/design/56986-godebug) for improving backward compatibility in go in ways that would allow
projects like Kubernetes to use supported go versions while retaining previous go version runtime behavior for a period of time.

With that proposal [accepted](https://github.com/golang/go/issues/56986#issuecomment-1387606939),
and a [request to clarify Kubernetes' approach to updating go versions on release branches](https://github.com/kubernetes/kubernetes/issues/112408),
it seemed like a good time to capture requirements and a process for updating Kubernetes release branches.

### Goals

* improve security of Kubernetes patch releases by building and releasing using a supported minor version of go
* preserve stability of Kubernetes patch releases by avoiding regressions or behavior changes due to updating go minor versions
* avoid end-user action-required items in patch releases due to updating go minor versions

### Non-Goals

* change the current approach of updating the default Kubernetes development branch to the latest go minor version as soon as possible
* change the cadence of Kubernetes patch releases
* change the duration of support for a given Kubernetes minor version

## Proposal

**1. Track prereq changes for each new minor go version**

Track changes made to the default Kubernetes development branch that were required to adopt a new go minor version (go 1.N).
This typically includes changes like:

* updates to static analysis tooling to support any go language changes
  (e.g. [#115129](https://github.com/kubernetes/kubernetes/pull/115129))
* updates to dependencies needed to work with go 1.N
  (e.g. [#114766](https://github.com/kubernetes/kubernetes/pull/114766))
* updates to Kubernetes code to fix issues identified by improved vet or lint checks
* updates to Kubernetes code to work with both go 1.N and 1.(N‑1)
  (e.g. [commit c31cc5ec](https://github.com/kubernetes/kubernetes/commit/c31cc5ec46315a02343ec6d6a2ef659e2cc8668e))

Prioritize making the prereq changes as minimal and low-risk as possible.
Merge those changes to the default Kubernetes development branch *prior* to updating to go 1.N.
This ensures those changes build and pass tests with both go 1.N and 1.(N‑1).
Here is an [example of tracking prereq changes for go1.20](https://github.com/kubernetes/release/issues/2815#issuecomment-1373891562).

**2. Backport prereq changes to release-1.x branches**

Backport prereq changes for go 1.N to release-1.x branches, keeping in mind the guidance to
[avoid introducing risk to release branches](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-release/cherry-picks.md#what-kind-of-prs-are-good-for-cherry-picks).

Considering the typical changes required to update go versions:

* Tooling and test changes carry minimal risk
* Dependency updates should be evaluated for extent / risk; if needed, and if possible, work with dependency maintainers to obtain the required fix in a more scoped way
* Updates to fix issues caught by improved vet or lint checks generally fix real problems and are reasonable to backport, or are very small / isolated and carry minimal risk
* Updates to make code work with both go 1.N and 1.(N‑1) should be evaluated for extent / risk; if needed, isolate the change in go-version-specific files.

Here is an [example of tracking backports of prereq changes for go 1.20](https://github.com/kubernetes/release/issues/2815#issuecomment-1373891562).

**3. Update release-1.x branches to new go minor versions**

Update release-1.x branches to build/release using go 1.N once all of these conditions are satisfied:

1. go 1.N has been released at least 3 months
   * this gives ~3 months for adoption of go 1.N by the go community
   * this gives ~3 months for completing the backports and cutting Kubernetes patch releases
     until go 1.(N+1) is released and go 1.(N‑1) previously used by Kubernetes patch releases
     goes out of support
2. go 1.N has been used in a released Kubernetes version for at least 1 month
   * this ensures all release-blocking and release-informing CI has run on go 1.N
   * this gives time for release candidates and early adoption by the Kubernetes community
3. Backported code and dependency changes build and pass unit and integration tests with both go 1.N 
   and the go minor version used for the .0 release of the Kubernetes release branch
   * this ensures consumers of patch releases of published Kubernetes
     libraries are not *forced* to update to go 1.N
4. There are no regressions relative to go 1.(N‑1) known to impact Kubernetes
5. Behavior changes in go 1.N are mitigated to preserve existing behavior for previous
   Kubernetes minor versions without requiring action by Kubernetes end-users.
   * In go1.21+, the go runtime is expected to match previous runtime behavior by default
     if we avoid leave the go version indicated in `go.mod` files in release branches unchanged,
     as described in https://github.com/golang/go/issues/56986
   * If necessary, other mitigation approaches can be used as long as they are transparent to end users. Examples:
     * defaulting GOGC in kube-apiserver `main()` to preserve go1.17 memory use 
       characteristics in Kubernetes 1.23.x updating from go1.17 to go1.18
     * defaulting GODEBUG in kube-apiserver `main()` to preserve SHA-1 x509 
       enablement in Kubernetes 1.23.x updating from go1.17 to go1.18
     * allowing kubectl plugin resolution to locate plugin binaries in the 
       current directory updating from go1.18 to go1.19

The [go update handbook](https://github.com/kubernetes/sig-release/blob/master/release-engineering/handbooks/go.md)
and [go update issue template](https://github.com/kubernetes/release/blob/master/.github/ISSUE_TEMPLATE/dep-golang.md)
would be updated or augmented to include this information.

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

### Test Plan

This KEP is not proposing specific code changes or additions to Kubernetes,
so it relies on the standard presubmit and post-submit release-blocking/release-informing test jobs.

This KEP does depend on the ability to test a specific change against two go versions.
Currently, this is accomplished using copies of presubmit jobs suffixed with
["go-canary"](https://grep.app/search?q=go-canary&filter[repo][0]=kubernetes/test-infra),
built using a proposed go version and manually triggered on a pull request.
This is the mechanism currently used to verify changes made to the default Kubernetes
development branch in preparation for a go minor version update.

To ensure release branches continue to remain compatible with the original
go minor version used with the .0 release for each release branch,
presubmit/periodic unit and integration test job variants would be created using using the original
go minor version. The [per-branch test job config fork](https://github.com/kubernetes/test-infra/tree/master/releng/config-forker)
tool could be updated to set up these unit and integration test jobs automatically.

##### Prerequisite testing updates

n/a

##### Unit tests

Addition of release branch periodic / presubmit unit test jobs
running with the original go minor version for the branch.

##### Integration tests

Addition of release branch periodic / presubmit integration test jobs
running with the original go minor version for the branch.

##### e2e tests

n/a

### Graduation Criteria

n/a

### Upgrade / Downgrade Strategy

n/a

### Version Skew Strategy

n/a

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

Updating to a new patch release built with a new go minor version.

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

End users can downgrade to the patch release prior to the go minor version update.

If a regression is reported, the go minor version update could be reverted and a new patch release cut.

###### What happens if we reenable the feature if it was previously rolled back?

n/a

###### Are there any tests for feature enablement/disablement?

n/a

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

n/a

###### What specific metrics should inform a rollback?

n/a

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

n/a

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

n/a

###### How can an operator determine if the feature is in use by workloads?

n/a

###### How can someone using this feature know that it is working for their instance?

n/a

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

n/a

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

n/a

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

n/a

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

n/a

###### What are other known failure modes?

n/a

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

- 2023-01-17: Created

## Drawbacks

* The primary drawback is risk of a user-facing regression in a patch release due to a go version change.
  Most of this KEP is focused on ways to detect and guard against that risk.
* This also creates some additional work for maintainers to track changes 
  related to go version updates, and shepherd those changes to more release branches.

## Alternatives

#### Request the go team backport security fixes to more than two minor versions

* The percentage of patch releases containing security-related content has risen in recent years,
  so "security fixes only" would not necessarily mean fewer patch releases on older minor versions
* Not all security fixes *can* be cleanly backported to older minor versions
* Adjusting go processes to make it easier for consumers like Kubernetes to update was strongly preferred

#### Maintain a custom patched version of older go minor versions with security fix backports

* This was alluded to in the discussion of https://github.com/kubernetes/kubernetes/issues/112408.
* It is not clear that all changes in every go minor and patch version with security implications
  would be consistently noticed and backported.
* It is not clear what would be done for fixes with security implications that do not backport cleanly / easily.
* Using a custom patched go distribution would lose some of the benefits of building/running the widely tested and used standard distribution
* Staffing maintenance of that fork (even potentially beyond the OSS Kubernetes minor version EOL date) would be an ongoing cost.
* This approach could confuse security scanners that would flag a binary built with an older go minor version as vulnerable
