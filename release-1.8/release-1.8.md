# Proposed timeline for 1.8
We will begin the 1.8 release the first week of July 2017.

The proposal below is similar in layout to the 1.7 plans with some
modifications:
- as before, key days aren't Fridays, since it can be hard to end milestones right up against weekends
- new in 1.8
  * alpha's should not be cut unless all master-release-blocking tests are passing
  * release notes for planned features done 2 weeks out from feature freeze
  * docs PRs deadlines ahead of release

## 1.8 Release Schedule Key Dates
- July 5 (Weds) coding start (7w)
- July 12th (Weds): v1.8.0-alpha.1
- July 26th (Weds): v1.8.0-alpha.2
- Aug 1 (Tues): Feature Freeze
- Aug 9 (Weds): v1.8.0-alpha.3
- Aug 23 (Weds): v1.8.0-alpha.4
- Sept 1 (Fri): Code Freeze 
- Sept 6 (Weds): v1.8.0-beta.0, v1.9.0-alpha.0
- Sept 13 (Weds): v1.8.0-beta.1
- Sept 18th (Mon): v1.9.0-alpha.1
- Sept 20th (Weds): v1.8.release-candidate
- Sept 27th (Weds):  v1.8.0 release!

## 1.8 Details

### July 5 - Sept 1
- 8 week coding period
- Release 1.8 alphas every 2 weeks
  - Alphas only cut if [release-master-blocking](https://k8s-testgrid.appspot.com/release-master-blocking) are passing
- July 12th (Weds): v1.8.0-alpha.1
- July 26th (Weds): v1.8.0-alpha.2
- Aug 1st (Tues): Feature Freeze: all features going into release should
  have an associated issue in the features repo
- Aug 8th (Tues): First draft of feature related release notes due
- Aug 9th (Weds): v1.8.0-alpha.3
- Aug 15th (Tues): Final draft of feature related release notes due
- Aug 23rd (Weds): v1.8.0-alpha.4

### Sept 1 (Fri) - Sept 14 (Thurs)
- Sept 1st Begin Code Freeze (Code Slush)
  * Hard deadline for feature-related PRs to be in submit queue
  * Add milestone restriction on submit queue.
  * Community Feature Burndown Meetings held two or three times until release week (then every day). For those interested in joining please join [Kubernetes Milestone Burndown Group](https://groups.google.com/forum/#!forum/kubernetes-milestone-burndown)
  * Focus on bugfix, test flakes and stabilization
  * Ensure docs and release notes are written
  * Identify all features going into the release, and make sure alpha, beta, ga is marked in features repo
  * Release team to create release notes draft doc and share broadly
- Sept 6th (Weds): v1.8.0-beta.0
  * Branch Manager to create `release-1.8` branch.
  * v1.9.0-alpha.0 is cut
  * Test-infra lead to set up CI for branch.
  * Begin regular fast-forwards (at least daily to keep CI up-to-date).
- Sept 8th
  * Docs PRs must be open for tech review
- Sept 13th (Weds): End of Code Freeze
  * v1.8.0-beta.1 cut
  * Final fast-forward of release branch.
  * After this, all changes for the release must now be cherrypicked in batches by branch
  manager.
  * Remove milestone restriction on submit queue. Bot drains backlog over the
  weekend.

### Sept 15 - Sept 27
- Sept 15th (Fri)
  * Docs PRs lgtm'd and ready for merge
- Sept 18th (Mon): v1.9.0-alpha.1
- Sept 20th (Weds): v1.8.release-candidate
  * RC means no known release-blockers outstanding.
  * Only accept cherrypicks for release-blockers.
  * Announce on mailing lists, Twitter, etc. to ask people to build and test the RC and submit feedback
  * Perhaps managed k8s providers make rc.1 available on early access channels.
  * More RCs as needed to verify release-blocking fixes.
- Release 1.8 on Sept 27 (Wednesday)


# Key features
[Feature tracking spreadsheet
link](https://docs.google.com/spreadsheets/d/1AFksRDgAt6BGA3OjRNIiO3IyKmA-GU7CXaxbihy48ns/edit#gid=0)

## Operational Changes 
1. For the 1.8 release, the [release team](https://github.com/kubernetes/features/blob/master/release-1.8/release_team.md)
  will use the following procedure to identify release blocking issues
  1. Any issues listed in the [v1.8 milestone](https://github.com/kubernetes/kubernetes/issues?utf8=%E2%9C%93&q=is%3Aissue%20is%3Aopen%20milestone%3Av1.8)
     will be considered release blocking. It is everyone's responsibility to move non blocking issues out of the `v1.8` milestone. Items targeting 1.8 can be moved into the `v1.8` milestone.
     milestones **or the release will not ship**

# Contact us
- [via slack](https://kubernetes.slack.com/messages/sig-release/)
- [via email](mailto:kubernetes-release@googlegroups.com)
