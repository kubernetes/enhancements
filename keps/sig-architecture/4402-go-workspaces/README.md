<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
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
-->
# KEP-NNNN: Go workspaces for k/k

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Go and v2](#go-and-v2)
  - [Alternatives to gengo/v2](#alternatives-to-gengov2)
  - [Risks and Mitigations](#risks-and-mitigations)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The main Kubernetes git repo (github.com/kubernetes/kubernetes, colloquially
called "k/k") is the embodiment of evolution.  When it was created, the only
way to build Go applications was GOPATH.  Over time we built tooling which
understood GOPATH, and it leaked into many parts of our build, test, and
release systems.  Then Go added modules, in part because GOPATH was unpleasant
and had a tendency to leak into tools.  We adopted modules, but we also added
the idea of "staging" repositories - a way to eat our cake and have, too.  We
get the benefits of a monorepo (atomic commits, fast iteration across repos)
and then publish those to standalone repos for downstream consumption.  To make
this work with modules, we abused GOPATH even harder and wrote even more tools.

The Go project saw what we (and others) were doing and did not like it.  So
they created [workspaces](https://go.dev/doc/tutorial/workspaces).  They
basically created a solution that is purpose-built for us.

This KEP proposes that we adopt Go workspaces and bring our tooling up to
modern standards.  In fact, the author started this work in 2021 or 2022,
discovering issues along the way that the Go team has worked to resolve.

This is not a user-facing change.  No k8s cluster-user or cluster-admin should
know or care that this happened.  On the surface, it seems like something that
should not warrant a KEP, but:
  a) KEPs are how we communicate big changes;
  b) This is a big change;
  c) There's some ecosystem impact to developers

## Motivation

Anyone who has had to maintain the multitude of `verify-*` and `update-*`
scripts has felt this pain.  Anyone who has had to deal with how we build and
test Kubernetes has run into the mess.  It's a constant source of complexity
and friction for maintainers, and it leaks out into our developer ecosystem.

This is the sort of mess that wise people back away from.  The author is not so
wise.

### Goals

1) To get rid of our reliance on GOPATH hackery to build Kubernetes and to run
   verifier tools and code-generators.
2) To simplify and clarify our tooling with regards to multi-module operation.
3) To be able to use upstream tooling, such as `gopls` across multiple modules.

### Non-Goals

1) To rewrite our code-generators (though some could use it).
2) To make the build or tools faster.
3) To change the meaning or structure of staging or the publishing bot system.

## Proposal

This KEP proposes to add a `go.work` file (checked in) and fix all the tools
that break because of that.  At the end, there will be no reliance on GOPATH.
verify and update scripts can use normal Go conventions for building and
running code.  Abominations like `run-in-gopath.sh` will be deleted.

The implementation will come in several waves, which will have to be sequenced
across k8s.io/kubernetes, k8s.io/gengo, and k8s.io/kube-openapi repos.  In
order to not leave repos in a broken state, some PRs will need to be larger
than we might otherwise like, but commits can tell the story.

This is a proposed merge sequence:

1. Retool the core of gengo to handle workspaces.  This will be gengo/v2.  The
   hand-rolled parsing logic will be removed and replaced with
   `golang.org/x/tools/go/packages.Load`, which understands workspaces.
1. Update kube-openapi to use gengo/v2.  TBD as to whether this represents
   kube-openapi/v2 (that may be simplest).
1. Introduce workspaces to k/k.
   1. Add a `go.work` file (which breaks a ton of intra-repo tools).
   1. Convert non-gengo parts of our build (e.g. `make`, `make test`, many
      `verify-foo.sh and `update-foo.sh` scripts) to use workspaces (often just
      removing things like `GO111MODULE=off` or `-mod=mod` decorations).
   1. Update gengo and kube-openapi versions in k/k (vendored).
   1. Convert gengo-based tools (in staging/src/k8s.io/code-generator) to
      gengo/v2 and kube-openapi/v2.
   1. Fix any remaining tools which had dependencies on the gengo-based tools.
   1. Remove detritus, including (but not limited to) run-in-gopath.sh,
      references to GOPATH, workarounds for vendored packages, etc.

Along the way, some tools will need more touch than others.  At least one -
import-boss - seems best to retool away from gengo entirely.

Somewhere in there, we want to make deeper changes to gengo/v2 to simplify it.
Some of the framework seemed clean at the time, but nearly a decade later, just
makes the tools harder to understand.  This could happen after all of this
work (repeating the sequence) or at the same time, making for larger but more
complete reworking.

### Go and v2

There's been a lot written about Go's v2 problem.  It can be confusing for
producers and consumers.  That said, it seems to work for what we want, which
is to NOT make explicit releases of gengo, even though there's a "v2".

Once consumers switch to v2 (`go get k8s.io/gengo/v2`), they can simply use
normal Go unversioned updates (`go get -u k8s.io/gengo/v2`).  See [this
report](v2-experiment.md) for more details.
of this KEP for more.

### Alternatives to gengo/v2

We did consider some options before landing on gengo/v2.

  1. Put it in `gengo/v2` with a README that says "don't use this unless you
     are part of kubernetes".  Pro: simple.
  1. Put it in a new `k8s.io/<something>` repo with a README that says "don't
     use this unless you are part of kubernetes".
  1. Put it in `k8s.io/code-generator/<something>` with a README that says
     "don't use this unless you are part of kubernetes" (pro: that puts it into
     k/k so atomic commits are OK).
  1. Try to make a k8s.io/internal github org work, and put it as
     `k8s.io/internal/<something>`.  Pro: stronger than a README.

The general feeling was that `gengo/v2` is simplest and "good enough".

### Risks and Mitigations

There's not a clean way to do this.  Go workspaces DOES NOT work with our
symlinks-in-vendor hack, and all of our tools do not work without it.  We have
a LOT of tools and the hacks run DEEP.  We have a decade of legacy to scrub
off.  Worse, many of our tools make dubious assumptions about the
interchangeability of Go package paths (e.g. "example.com/foo/bar") and disk
paths.  Those assumptions held as long as we used GOPATH, BARELY held with
staging modules, and break totally in workspaces.

This means that some of our ecosystem-facing APIs (codegen tools, in
particular) *must* change.  We can mitigate some of the impact by creating "v2"
of things like k8s.io/gengo (which powers many tools).  We might choose to do
the same in kube-openapi, though its API is not changing, just the
implementation.

However, k8s.io/code-generator is more challenging.  It holds all of our tools
and some scripts for invoking them, and is used by a non-trivial number of
downstream projects (operators, custom API servers, etc).  Making a v2 of this
(which itself lives in staging, and is subject to this KEP!!) is something we
have not done before.  The simple answer is to rely on the fact that it was
never really versioned before, and just make the changes.  We could move it all
to a new "legacy-code-generator" staging repo and explain that it gets no
further support, or we could just tag the last legacy release of
k8s.io/code-generator and tell people to sync to that tag (or just rely on the
existing `v0.29.x` tag).  Same effect.

Another risk is that this depends on
[Go 1.22](https://tip.golang.org/doc/go1.22) and
[vendoring in workspace mode](https://github.com/golang/go/issues/60056), which
are not yet released. Go 1.22 is scheduled to release in February 2024, which
seems like good timing for k8s 1.30, but has not yet been confirmed as
plan-of-record.  As of Jan 13, 2024, people on the release team think it's
likely that we will release k8s 1.30 on Go 1.22, but
[that work](https://github.com/kubernetes/release/issues/3280) has not yet
begun.  The closer we get to the Kubernetes release, the more risk this
introduces, so if we can't get it in early, we should probably wait until 1.31.

We won't not merge things to master that require go 1.22 until master is
solidly on go 1.22 (1.22.0+, not a release candidate), with no known
regressions / release blockers for at least a couple weeks.

Another risk is that there are many tools in use which DO NOT HAVE TESTS.  They
seem to work today, but upon closer inspection it's not always clear that they
work correctly or completely, or what the exact semantics are intended to be.
Changing these is dangerous.  As we proceed with this work, we will have to add
tests to tools and verifiers, and look closely at the results of the various
code-generators.  Ideally there will be NO CHANGE in generated code.

### Test Plan

We will add some tests to untested tools, though complete functional coverage
is difficult, and would be a prohibitively high bar (e.g. it's unreasonable to
demand net-new tests for every codegen tool which exercises every path). That
said, we will look carefully at how each tool works, to mitigate risks.  At the
time of this writing, the author has spent upwards of 100 hours in the
debugger, single-stepping through each tool.

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

We will add tests for critical subsystems and tools being changed (e.g. gengo's
package loading).

##### Integration tests

N/A - see e2e

##### e2e tests

The best test for work like this is, more or less, "did it work?".

If the codegen tools emit the exact same code (or changes are manually
verifiable as correct) then "it works".  Part of this process must be to
exercise it on out-of-tree consumers, which will be difficult to do until it is
merged.  We expect some amount of ex post facto problem reports from
downstreams.

### Graduation Criteria

N/A - once this merges it is live.

### Upgrade / Downgrade Strategy

N/A - not user-facing.

### Version Skew Strategy

N/A - not user-facing.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

N/A for clusters.  Once we merge it, it is active for developers in k/k.  Once
we cut a release tag, it is active for consumers of k8s.io/code-generator.  If
we should need to back it out, it will be a complex set of git reverts, across
at least k/k and kube-openapi repos.

As such, we must make sure that each PR works and is self-contained - no
leaving the repo(s) in a broken state.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

N/A.

###### What happens if we reenable the feature if it was previously rolled back?

N/A.

###### Are there any tests for feature enablement/disablement?

N/A.

### Rollout, Upgrade and Rollback Planning

N/A.

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A.

###### What specific metrics should inform a rollback?

If there is massive downstream breakage, we may need to revert. If we adopt Go
1.22, merge this KEP, then decide to roll-back to Go 1.21 for some reason, we
WILL need to revert this KEP.  We should have pretty high confidence in Go 1.22
before we merge this.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

Yes.  The CLI of various code-gen tools has to change.  These tools are used by
downstream developers.

### Monitoring Requirements

N/A.

###### How can an operator determine if the feature is in use by workloads?

N/A.

###### How can someone using this feature know that it is working for their instance?

N/A.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A.

### Dependencies

Go 1.22

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

N/A.

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A.

###### What are other known failure modes?

None.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A.

## Implementation History

* Started work in 2022, needed more from Go team.
* Work went stale.
* Restarted in 2023.
* KEP.

## Drawbacks

None.

## Alternatives

1) Do nothing (keep our toos and dev-exp semi-broken)
2) Change how we manage our repos (abandon staging)
3) Abandon the mono-repo
4) Something more dramatic?

## Infrastructure Needed (Optional)

None.
