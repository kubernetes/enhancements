# Kubernetes Yearly Support Period

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Component changes](#component-changes)
- [Test Plan](#test-plan)
  - [Additional Version Test Coverage](#additional-version-test-coverage)
  - [Additional test/release process changes](#additional-testrelease-process-changes)
  - [Documentation](#documentation)
    - [Clarify “Support Policy” in community documentation](#clarify-support-policy-in-community-documentation)
    - [Document impact on external dependencies support windows](#document-impact-on-external-dependencies-support-windows)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Maturity levels (alpha, beta, stable)](#maturity-levels-alpha-beta-stable)
    - [Deprecation policy](#deprecation-policy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed](#infrastructure-needed)
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

The goal of this KEP is to increase patch-fix support for each Kubernetes release from the current 9 months to 1 year.
The primary motivation is to address an end-user request:  Business cycles (financial, certification, sales, vacation, HR and others) are overwhelmingly annual, leading to many users desiring at least annual support cycles in projects on which they depend.

Details of the expected tasks involved in implementing this policy are outlined below, the most major of which are:

- Review skew policies, identify which skews are derived from the number of supported versions, and identify new target skews (e.g. oldest kubelet against newest API server)
- Review deprecation policies, identify which deprecation periods are derived from the number of supported versions, and identify new target deprecation periods (e.g. deprecation period for GA features/APIs)
- Review support policies of external dependencies and identify/resolve gaps to ensure we do not claim longer support than the things we build on top of

## Motivation

Today the Kubernetes project delivers minor releases (e.g.: 1.13 or 1.14) every 3 months.
The project provides bugfix support via patch releases (e.g.: 1.13.Y), with each minor release (e.g.: 1.13) having this patch release stream of support for approximately 9 months.
This means a cluster operator must upgrade at least every 9 months to remain supported.
Nine months is an irregular period of time relative to the planning, operations, and financial cycles of most companies.

Beyond availability of patch support and an upgrade path, end-user also faces the complexity of managing usage of features versus [deprecation policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/) and the [version skew policy](https://kubernetes.io/docs/setup/release/version-skew-policy/).
With a new minor release coming every three months, there is a constant need to evaluate and plan and enact forward movement.

The survey conducted in early 2019 by the WG LTS showed that a significant subset of Kubernetes end-users fail to upgrade within the 9-month support period.
In the graph below, 1.13 was the current version, meaning that 1.9 and 1.10 were a few months out of support.
Yet about 1/3 of users had production machines on those versions:

![Kubernetes versions in production](./versions-in-production.png)

This, and other responses from the survey, suggest that this 30% of users would better be able to keep their deployments on supported versions if the patch support period were extended to 12-14 months.
This appears to be true regardless of whether the users are on DIY build or commercially vendored distributions.
An extension would thus lead to more than 80% of users being on supported versions, instead of the 50-60% we have now.

A yearly support period provides the cushion end-users appear to desire, and is more in harmony with familiar annual planning cycles.

We view elongating the patch support window toward 1 year as a relatively minor change, yet is believed viable from an implementation perspective and is a meaningful first response to users.

There are many unknowns about changing the support windows for a project with as many moving parts as Kubernetes, and keeping the change relatively small (relatively being the important word), gives us the chance to find out what those unknowns are in detail and address them.

### Goals

- Address end-user feedback regarding support time per release
- Define an annual support policy.
- Define any prerequisites for the policy.
- Set acceptance criteria for enacting the change in support policy. This should be supplemented by KEPs and PRs that address the implementation of the policy by the release team and Release Engineering (patch management).
- Clarify and confirm that the support period is defined based on time (e.g. 1 year) and not based on a specific number of releases (e.g. 3 or 4) given that the release cadence might change in the future.

### Non-Goals

- Define all details of “support”, which is somewhat imprecise in the project already today.
- Change the release cadence.  It is currently 3 months per release or 4 releases per year.  See “Alternatives” below for related work on release cadence.
- Establish an “LTS” release process in the sense perhaps known from other projects (e.g.: pick one release every year or two, give patch support on that release for multiple years).
- Declare that Kubernetes’ dependencies must also have a similar support lifetime per release.
- Align Kubernetes releases in time with those of its dependencies.

## Proposal

The proposal is to extend the Kubernetes support cadence from 9 months to 14 months.
The 14 months proposed here is essentially 12 months of support and 2 months of upgrade period.
NOTE: This follows and formalizes today’s practice of retaining patch branch CI for 6-7 weeks beyond the 9 months intended support phase as a failsafe option if any highly critical CVEs or blocking upgrade issues are observed by slower adopters of newer patch branches.

A visual example today vs. proposed for an adopter of 1.18 in April 2020:

![Visual example of upgrade cadences](./upgrade-cadence.png)

## Design Details

### Component changes

* Adjust behavior in the kube-apiserver that assumes a component lifetime of ~9 months
  * Lifetime of the certificate used for kube-apiserver loopback connections (#86552)
* Expand supported kubelet/apiserver skew to 12 months divided by release cadence minus 1, i.e. currently (12/3)-1=3 releases.
  * Currently, supported skew (2 versions) was chosen to allow the oldest kubelet to work against newest apiserver
  * Implementation should include adding a test to cover the added skew variations, have it release informing, watch for any errors, fix them, promote the test to release blocking.  This is already a gap today and can be treated as a project need orthogonal to this KEP.
* Expand supported kubectl/apiserver skew
  * Currently supported skew (+/- 1 minor version) theoretically allows using an n-1 kubectl to work against all supported apiservers (n-2/n-1/n).
  * Ideally, the latest kubectl would support speaking to all supported apiservers (+0/-3 minor versions).

## Test Plan

### Additional Version Test Coverage

In order to support the additional version, the following additional jobs will need to be added to testgrid.  These are all copies of existing jobs, spread for the additional versions.

* [SIG Release Jobs](https://testgrid.k8s.io/sig-release-master-blocking): Blocking and informing test job suites would need to be adjusted so that they are dropped after n-4 releases instead of n-3 as they are now.
* Add [skew tests/jobs](https://testgrid.k8s.io/sig-cli-master#skew-cluster-stable1-kubectl-latest-gke) to test furthest supported skew per above.  These are the only "new" tests that need to be added.
* [Conformance tests](https://testgrid.k8s.io/conformance-all) will theoretically also have to retire old versions 3 months later, thus running jobs for more versions.  In practice, our conformance test suite rarely retires jobs at all, and runs most jobs back to version 1.11, so no real change is needed here.

A few SIGs will want to expand their version test coverage as well.  [For example, SIG-Cluster-Lifecycle/Kubeadm tests](https://testgrid.k8s.io/sig-cluster-lifecycle-kubeadm) would need to cover an extra version and an extra upgrade version, as would [kubebuilder tests](https://testgrid.k8s.io/sig-api-machinery-kubebuilder).  Most SIGs, however, only run jobs against current and would not require changes.

### Additional test/release process changes

* Staff patch releases for one more branch for ~3-4 more months
  * Feedback from SIG Release received, see [SIG Release meeting discussion](https://docs.google.com/document/d/1Fu6HxXQu8wl6TwloGUEOXVzZ1rwZ72IAhglnaAMCPqA/edit#bookmark=id.mi11nk75iohl).
* Maintain CI for one more branch for ~3-4 more months
  * Modifying test job and infrastructure patterns may need to be deprecated on a slower schedule or backwards compatibility with older style tests in older branches may need maintained longer.  This is a problem already today, in that images and jobs need to be maintained for all branches already, but adding another version will make it worse.
* Formalize turn-down timing of CI/ for oldest release and patch acceptance criteria for release branch:
  * Historically this trailing period outside of support was squishy ("wait a little while, cut a final patch release, turn down jobs").  It is currently specified that oldest release branch CI support is turned off at the time when the newest, under-development release branch is created.  This happens at the first ‘beta’ release, e.g.: ‘1.16.0-beta.0’.  Due to an implementation detail where branches are referred to by “dev”, “beta”, “stable1”, “stable2”, “stable3” there is a rotation of marker variables at the point of new branch creation.  This is an automated process, run by the SIG Release subproject Release Engineering branch manager, and can be adjusted as needed.  But while CI is running if a CVE or upgrade issue is encountered and resolution is able to be backported the community has intended to do so.  In practice this has not been required.
  * This KEP proposes the rotation happen 2 months after the 1 year period, i.e.: 12+2 months patch ability.  This constitutes 12 months of normal patch releases, plus a two month grace period for end users to plan and upgrade their clusters to a supported version. During this special 2-month grace period only critical security and upgrade fixes are acceptable on the branch.  I.e.:
    * Support Policy during the first 12 months.
      * Same as the current 9 months policy.
    * Support Policy during the final +2 months.
      * Only critical security patches and upgrade blocking fixes, i.e.:
        * CVE assigned by Product Security Committee initiated to release branch by Product Security Committee
        * Cherry-pick of upgrade scenario bug fix approved by owning SIG and Patch Release Team
* Formalize 3rd party dependency update policy (e.g.: golang, but also others too)
    * Patch release updates: Need to track upstream golang patch releases continually.
    * Major release updates: Can we also formalize when to move to a new Golang major release.  This is needed as two of our annual releases today (and all of them if we move to a yearly lifecycle) run beyond the end of Golang’s patch release lifecycle.
    * Some dependencies in particular need calling out as needing effort for maintaining an extra release branch:
      * The Bazel testing dependency could add a lot of effort, because it may involve backporting bazel fixes.
      * golang version support policy could also create a lot of work.


### Documentation

Update the [version skew policy](https://kubernetes.io/docs/setup/release/version-skew-policy/), which is currently the only public place we document which versions are supported.  Also update internal documents with the same content such as [kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/create-cluster-kubeadm/#version-skew-policy) and [patch releases](https://github.com/kubernetes/sig-release/blob/master/releases/patch-releases.md)

#### Clarify “Support Policy” in community documentation
* Generally, e.g.:
https://git.k8s.io/community/contributors/devel/sig-release/cherry-picks.md#what-kind-of-prs-are-good-for-cherry-picks
* Specifically:
  * 12 months general bug fix patching and backports
  * 2 months critical security patches and upgrade blocking fixes

#### Document impact on external dependencies support windows

* Example: golang support for ~1 year (2 6-month releases)
* Options:
  * Upgrade dependencies more on release branches
  * Note where support gaps exist and create plans for more proactively managing deps

### Graduation Criteria

WG LTS proposes this KEP be implemented as of the 1.19.0 release.
We do not want to retroactively ask the community to add longer support for an already delivered release.
Starting with 1.19.0 means that the supported range would change to 1.19 - 1.22 when 1.22 comes out.

### Upgrade / Downgrade Strategy

Unchanged from existing project policy.
Upgrading an existing cluster in-place still requires step by step moving control plane components from version N to N+1.
It is recognized that this is currently a pain point in the project and this KEP incrementally worsens the user experience for folks who defer to the maximum their upgrades.

#### Maturity levels (alpha, beta, stable)

Criteria for promotion from alpha to beta and beta to stable remain unchanged.

#### Deprecation policy

Unchanged from existing Kubernetes project deprecation criteria.  Deprecations for GA user interfaces today are already at least 12 months.

### Version Skew Strategy

Incremental change elongating existing Kubernetes project version skew policy, i.e.: where “N+/-1” to “N+/-2” and where “N+/-2” to “N+/-3”.
This matters, especially for external clients programmatically interacting with a long-lived cluster, beyond the cluster control plane components, moved through the upgrade process.

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks

The proposed change may encourage end-users to provide less frequent experimental feedback (eg: alpha/beta feature feedback) to the community on new releases, upgrade-ability, and compatibility.
This is an existing problem in the project today for users who do not regularly update, it remains unaddressed by this KEP, but is not notably exacerbated.
Users who did not upgrade regularly do not give us early feedback.
Improving this remains as future work for the project.

There is a risk this KEP is a slippery slope to subsequent iterative support extensions.
There is a finite limit to the resources an open source community can give to supporting a given release and this KEP represents a pragmatic balance for our Kubernetes community.
Discussion has shown there is zero community support for further elongation of the community support window beyond that proposed in this KEP.
There is already active community resistance to the idea that one might attempt to further elongate support at the community level.
Distributors/vendors remain free to carve out their own support niche if they feel there is a market for that.

An increase to the support period of Kubernetes also increases the human burden of maintenance on the patch management (Release Engineering), test-infra, k8s-infra, sig-pm/release, and regular SIG functioning.
Some of these impacts have already been recognized and sub-projects have been set up to mitigate them.
Still, there will be an unavoidable increased cost in dollars for infrastructure resources kept online 3 additional months, and additional engineering costs patching critical bugs in more versions.

Other areas of possible concern are listed below, but WG LTS feels these are orthogonal, not blocked by this proposal, and may be addressed independently:

API Stability
* Definition: Core APIs have 1 GA (v1) version available (vs beta/alpha versions of the APIs).
* Refer to https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning for the existing policy.
* Deprecation Policy: https://kubernetes.io/docs/reference/using-api/deprecation-policy/
* Version Skew policy: https://kubernetes.io/docs/setup/release/version-skew-policy/
* Conformance without beta REST APIs (KEP): https://github.com/kubernetes/enhancements/pull/1332
* As the project matures in stability other KEPs may eventually aim to elongate:
  * Active test coverage for version skew and upgrade scenarios
  * Deprecation timelines

Improved test-infra, tests
* The project needs upgrade tests which cover version skew variations already today.  This is viewed as orthogonal to this KEP.

Tooling around patching and release is insufficient today.  The project could use for example:
* Upgrade tooling (e.g.: standard blue/green cluster control plane deployment tool?) to ease * migration from an old release (older than N-1) to a current release
* Compatibility checking tools: “k8s-compat-check -v 1.18 -config my_cluster” reports objects which will need conversion if my_cluster were to move to 1.18.
* Tooling to make cherry-pick PRs against multiple branches easier to submit and review as a set related to their common master branch ancestor PR.

Well defined and predictable path from alpha->beta->stable:
* Today the project does clearly define feature promotion and deprecation:
  * https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
  * https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-parts-of-the-api
* These timelines today are not out of harmony with a support lifecycle covering one year. Today a feature cannot become GA and also be deprecated away in a time shorter than one year.
* As the project matures in stability other KEPs may:
  * Modify alpha->beta->stable graduation criteria.  E.g.: A change to GA feature promotion such that is is accomplished in not less than 4 releases or 12 months may be beneficial.
  * SIG Arch is investigating this space, e.g.:
    * https://kcsna2019.sched.com/event/VvMK/raising-the-bar-production-readiness-in-k8s-john-belamaric
    * https://github.com/kubernetes/community/issues/4000

## Alternatives

  * No change: These continue to leave users unsatisfied.
  * Longer support change (e.g.: 2, 3, or 5 years):
    * This is hard to implement in the current context of the project’s practices.
    * Three years is a special number in that hardware leases are often on three-year cycles and that leads to a desire to pave systems on a three-year cadence.
    * All the work detailed in this KEP would be required anyway before this option could be implemented.
  * Change the time to create releases: This is orthogonal.  It may offer the ability to do software engineering in a way that creates a more stable output which is more supportable for a longer period, but it may also do the opposite.
  * Two proposals were introduced in the WG LTS namely Stable/Dev proposal by Tim St.Clair  and LTS proposal by Lubomir Ivanov.
    * Tim St Clair’s proposal can be read here: https://docs.google.com/document/d/1R31UmXJGsmb4fT3xZc5qQNf5LVOMOiNaQ0Sou7W6U50/edit - Tim proposes a devel-stable split, with even more frequent releases (about a month in between) in the devel branches, with a stable release once per year, after a hardening process.
    Stable releases would be supported for two years (that is, there would be two stable releases at any one time).
    This may be compatible and complementary with the change proposed in this KEP.  Additional changes in this direction remain a possibility in the future.
    * Lubomir’s proposal can be read here: https://docs.google.com/document/d/1o6OvJ1XAc-n9to-8YVYoSiEx95yqldQfdE3qplyy0CY/edit#heading=h.wlnbgu3qtn3i
    Lubomir proposes marking a release as stable about once every 18 months, and carefully backporting fixes to it for its support lifetime (which is also 18 months).
  Both of these proposals share as a base assumption that the current 9 months of support is not ideal, and should be extended, which we’ve tried to achieve with this proposal.
  As such Lubomir’s proposal may be compatible and complementary with the change proposed in this KEP.
  Additional changes in this direction remain a possibility in the future.

## Infrastructure Needed

No infrastructure is required to be added compared to today.  But…

The Release Engineering branch manager role would likely enact the “Remove the oldest release variant” on the release one version older than current, causing some infrastructure resources to remain in use three additional months compared to before.
The upper limit of resource consumption would be +25% on today, but this is an excessive estimate given master branch testing consumes a disproportionate amount of resources compared to patch release branch testing.

This estimate presumes that the implementation involves keeping four parallel releases under support and no changes to the cadence of creating a new release (currently 3 months).
It is possible that the cadence is slowed (e.g.: 4 months), leaving no net increase in infrastructure consumption.
It is also possible other alternatives in discussion speed the cadence (e.g.: less than 3 months), in which case resource consumption and the number of patch release branches in support concurrently across a year would need to be reevaluated (e.g.: Tim St. Clair’s proposal).
