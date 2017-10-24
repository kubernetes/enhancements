# Release 1.9

The 1.9 release cycle begins on Monday, October 2, 2017.

* [Release Team](http://bit.ly/k8s19-team)
* [Meeting Minutes](http://bit.ly/k8s19-minutes)
* [Zoom](http://bit.ly/k8s19-zoom)
* [Slack](https://kubernetes.slack.com/messages/sig-release/)
* [Forum](https://groups.google.com/forum/#!forum/kubernetes-sig-release)
* [Feature Tracking Sheet](http://bit.ly/k8s19-features)
* [Milestone Process](https://github.com/kubernetes/community/blob/master/contributors/devel/release/issues.md)

## Notes

* There are a number of holidays plus KubeCon during Q4, so this release will
  need to have a reduced scope in order to meet the tight schedule.
* Features that don't have complete code and tests by [Code Freeze](#code-freeze)
  **may be disabled by the release team** before cutting the first beta.
* The release team will escalate [release-master-blocking](https://k8s-testgrid.appspot.com/sig-release-master-blocking)
  failures to SIGs throughout the cycle, not just near release cuts.
* Key deliverables (e.g. release cuts) tend to be scheduled on Wednesdays
  to maintain context while ramping up and then responding to any problems.

## Timeline

* Mon Oct 2: Start of release cycle
* **Fri Oct 27: [Feature Freeze](#feature-freeze)**
  * All features must have tracking issues.
* Wed Nov 1: v1.9.0-alpha.2
* Wed Nov 15: v1.9.0-alpha.3
* Thu Nov 16: v1.9.0-beta.0
  * Create `release-1.9` branch and start daily `branchff`.
  * Start setting up branch CI.
* **Mon Nov 20: [Code Slush](#code-slush)**
  * All PRs must be approved for the milestone to merge.
* **Wed Nov 22: [Code Freeze](#code-freeze)**
  * All features must be code-complete (*including tests*) and have docs PRs open.
  * Only release-blocking bug fixes allowed after this point.
* **Mon Nov 27: [Pruning](#pruning)**
  * The release team may begin **disabling incomplete features** unless they've
    been granted [exceptions](#exceptions).
* Wed Nov 29: v1.9.0-beta.1
  * Begin manual downgrade testing.
* Fri Dec 1: Docs Deadline
  * All docs PRs should be ready for review.
* Wed Dec 6: v1.9.0-beta.2 (week of KubeCon)
* **Mon Dec 11: End of Code Freeze**
  * Perform final `branchff`.
  * The `master` branch reopens for work targeting v1.10.
  * PRs for v1.9.0 must now be cherry-picked to the `release-1.9` branch.
* **Wed Dec 13: v1.9.0**
* Thu Dec 14: v1.10.0-alpha.1
* Thu Dec 21: Release retrospective (in community meeting slot)

## Details

### Feature Freeze

All features going into the release must have an associated issue in the
[features repo](https://github.com/kubernetes/features) by **Fri Oct 27**.

SIG PM will then review features and work with other SIGs to draft release notes
and themes.

### Code Slush

Starting on **Mon Nov 20**, only PRs that are [approved for the milestone](https://github.com/kubernetes/community/blob/master/contributors/devel/release/issues.md)
will be allowed to merge into the `master` branch.
All others will be deferred until the end of [Code Freeze](#code-freeze),
when `master` opens back up for the next release cycle.

Code Slush begins prior to Code Freeze to help reduce noise from miscellaneous
changes that aren't related to issues that SIGs have approved for the milestone.
Feature work is still allowed at this point, but it must follow the process to
get approved for the milestone.

#### Exceptions

Starting at Code Slush, the release team will solicit and rule on
[exception requests](https://github.com/kubernetes/features/blob/master/EXCEPTIONS.md)
for feature and test work that is unlikely to be done by Code Freeze.

### Code Freeze

All features going into the release must be code-complete (*including tests*)
and have [docs PRs](https://kubernetes.io/docs/home/contribute/create-pull-request/)
open by **Wed Nov 22**.

The docs PRs don't have to be ready to merge, but it should be clear what the
topic will be and who's responsible for writing it.

After this point, only release-blocking issues and PRs will be allowed in the
milestone. The milestone bot will remove anything that lacks the
`priority/critical-urgent` label.

### Pruning

Features that are partially implemented and/or lack sufficient tests may be
considered for pruning beginning on **Mon Nov 27**,
unless they've been granted [exceptions](#exceptions).

The release team will work with SIGs and feature owners to evaluate each case,
but for example, pruning could include actions such as:
* Disabling the use of a new API or field.
* Switching the default value of a flag or field.
* Moving a new API or field behind an Alpha feature gate.
* Reverting commits or deleting code.

This needs to occur before the first Beta so we have time to gather signal on
whether the system is stable in this state.

Pruning is intended to be a last resort that is rarely used.
The goal is just to make code freeze somewhat enforceable despite the lack of a
feature branch process.

### Burndown

Burndown meetings are held two or three times until the final release is near,
and then every business day until the release.

Join the [Kubernetes Milestone Burndown Group](https://groups.google.com/forum/#!forum/kubernetes-milestone-burndown)
to get the calendar invite.

* Focus on bugfix, test flakes and stabilization.
* Ensure docs and release notes are written.
* Identify all features going into the release, and make sure alpha, beta, ga is marked in features repo.

