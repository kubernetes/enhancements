# Proposed timeline for 1.7
We will begin the 1.7 release the first week of April 2017.

The proposal below is similar in layout to the 1.5 and 1.6 plans with some
modifications:
- as before, key days aren't Fridays, since it can be hard to end milestones right up against weekends

## 1.7 Release Schedule
- April 3 (Mon) coding start (7w)
- April 6th (Thurs): v1.7.0-alpha.1
- April 19th (Weds): v1.7.0-alpha.2
- April 26 (Weds) features repo freeze
- May 3 (Weds): v1.7.0-alpha.3
- May 17(Weds): v1.7.0-alpha.4
- May 31 (Weds): v1.7.0-beta.0
- Jun 1 (Thurs): Code Freeze 
- Jun 7 (Weds): v1.7.0-beta.1
- June 14 (Weds): v1.7.0-beta.2
- June 19th (Mon): v1.8.0-alpha.1
- June 21th (Weds): v1.7.release-candidate
- June 28th (Weds):  v1.7.0 (release!)

## 1.7 Details

### April 3 - May 31
- 8 week coding period
- Release 1.7 alphas every 2 weeks
- April 6th (Thurs) v1.7.0-alpha.1
- April 19th (Weds): v1.7.0-alpha.2
- April 26 (Weds) features repo freeze: all features going into release should
  have an associated issue in the features repo
- May 3 (Weds): v1.7.0-alpha.3
- May 17(Weds): v1.7.0-alpha.4
- May 31 (Weds): v1.7.0-beta.0
  * Create release branch.
  * Set up CI for branch.
  * Begin regular fast-forwards (at least daily to keep CI up-to-date).

### June 1 (Thurs) - June 15(Thurs)
- June 1 Begin Feature Freeze
  * Deadline for feature-related PRs to be in submit queue
  * Add milestone restriction on submit queue.
  * Community Feature Burndown Meetings held two or three times until release week (then every day). For those interested in joining please join [the Google Group](https://groups.google.com/forum/#!forum/kubernetes-milestone-burndown)
  * Focus on bugfix, test flakes and stabilization
  * Ensure docs and release notes are written
  * Identify all features going into the release, and make sure alpha, beta, ga is marked in features repo
- Jun 7 (Weds): v1.7.0-beta.1 cut
- June 14: End of Code Slush
  * v1.7.0-beta.2 cut
  * Final fast-forward of release branch.
  * All changes for the release must now be cherrypicked in batches by branch
  manager.
  * Remove milestone restriction on submit queue. Bot drains backlog over the
  weekend.

### June 19 - June 28
- June 19th (Mon): v1.8.0-alpha.1
- June 21th (Weds): v1.7.release-candidate
  * RC means no known release-blockers outstanding.
  * Only accept cherrypicks for release-blockers.
  * Announce on mailing lists, Twitter, etc. to beg people to try the RC for real.
  * Perhaps managed k8s providers make rc.1 available on early access channels.
  * More RCs as needed to verify release-blocking fixes.
- Release 1.7 on June 28 (Wednesday)


# Key features
[Feature tracking spreadsheet
link](https://docs.google.com/spreadsheets/d/1IJSTd3MHorwUt8i492GQaKKuAFsZppauT4v1LJ91WHY/edit#gid=0)

## Notable operational changes

1. Starting in the 1.7 release the [release team](https://github.com/kubernetes/features/blob/master/release-1.7/release_team.md)
  will use the following procedure to identify release blocking issues
  1. Any issues listed in the [v1.7 milestone](https://github.com/kubernetes/kubernetes/issues?utf8=%E2%9C%93&q=is%3Aissue%20is%3Aopen%20milestone%3Av1.7)
     will be considered release blocking. It is everyone's responsibility to move non blocking issues out of the `v1.7` milestone. Items targeting 1.7 can be moved into the `v1.7` milestone.
     milestones **or the release will not ship**

# Contact us
- [via slack](https://kubernetes.slack.com/messages/k8s-release/)
- [via email](mailto:kubernetes-release@googlegroups.com)
