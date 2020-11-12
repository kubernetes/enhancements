# Anago to Krel Migration

<!-- toc -->

- [Objectives](#objectives)
- [Milestones](#milestones)
  - [First Milestone: Complete the Migration Effort](#first-milestone-complete-the-migration-effort)
    - [Open Issues](#open-issues)
    - [Acceptance Criteria](#acceptance-criteria)
  - [Second Milestone: Introduce krel stage/release](#second-milestone-introduce-krel-stagerelease)
    - [Open Issues](#open-issues-1)
- [Risks](#risks)
- [Quality/Test Plan](#qualitytest-plan)

<!-- /toc -->

_Moving away from running bash in production in k/release_

## Objectives

This roadmap defines a strategy for achieving two primary goals: migrating
exchangeable bits of bash code within anago to krel and creating a Golang native
replacement for anago.

## Milestones

1. Complete the code migration
1. Have a minimum working krel stage
1. Have a minimum working krel release
1. Remove/swap out Anago in a simple way, after completing the preceding steps

The scope and implementation details of Milestones 2-4 will become clearer as
work on Milestone 1 proceeds.

Creating new features for krel is out of scope.

### First Milestone: Complete the Migration Effort

Anago is still the main bash script running in GCB, which right now calls out to
krel if necessary. Many parts of the bash-based source code in k/release have
already been transferred to krel (golang), whereas we directly remove the
bash-based parts from the repository after each refactoring iteration.

This milestone focuses on reducing technical debt in k/release by migrating the
remaining bash code into refactored golang-based implementations. This effort
will lead to higher quality and provide a stable foundation for future feature
developments. By “stable,” we mean that making changes will not break the entire
system.

This migration will not interrupt our ability to cut releases.

#### Open Issues

The list of currently outlined issues, with assignees (release managers) where
established:

- Add krel anago subcommand to retrieve the build candidate (TBD)

  https://github.com/kubernetes/release/issues/1536

- Introduce krel anago subcommand to update GitHub release

  https://github.com/kubernetes/release/issues/1534 (@xmudrii)

- Finish-up krel push

  https://github.com/kubernetes/release/issues/1459 (@saschagrunert)

- Introduce krel subcommand for pushing git objects

  https://github.com/kubernetes/release/issues/1446 (TBD)

All four issues can be worked on in parallel. This is not a comprehensive list:
There are still parts in Anago that can be ported from bash and that are not
part of any issue yet.

#### Acceptance Criteria

- All issues currently open will be resolved
  ([#1534](https://github.com/kubernetes/release/issues/1534),
  [#1536](https://github.com/kubernetes/release/issues/1536),
  [#1446](https://github.com/kubernetes/release/issues/1446),
  [#1459](https://github.com/kubernetes/release/issues/1459))
- New code is unit-tested and code-reviewed (logical paths, not line coverage)
- Direct use of the new Golang source code in production

### Second Milestone: Introduce krel stage/release

In parallel to the ongoing migration (first milestone) we will introduce new
krel stage and krel release subcommands. The plan is to re-evaluate the current
functionality within anago and build a declarative approach of cutting releases.
We can re-use the already migrated parts as well as using the existing logic in
anago as guidance for the necessary feature set of krel stage/release.

#### Open Issues

The list of currently outlined issues, with assignees (release managers) where
established:

- Evaluate possible krel stage/release subcommands

  https://github.com/kubernetes/release/issues/1551

## Risks

The highest risk during the migration is that we end-up in a state where we
break the current functionality. This would mean that we cannot build releases
any more. Immediate fixing and incremental testing between the releases should
minimize this risk.

## Quality/Test Plan

Merge changes to the main branch from user fork/branch as per normal community
PR process. Feature branches will not be used.

Anago-replacement features must be behind a feature gate, initially ensuring
they are only run in ‘mock’ mode.

Merged features can be tested in production at any time so long as they are only
triggered from a mock stage or mock release or mock notify.

Non-mock testing will occur only during a release cycle’s alpha period. This
gives initial test ability for non-mock paths in Sep/Oct 2020 and again in
Jan/Feb 2021. Beyond Feb 2021, we will need to re-evaluate testing based on
future circumstances.

During alpha periods we can A/B test, eg: build alpha.1 with Anago and
immediately after build alpha.2 with krel. Compare the results.
