# Proposed timeline for 1.9
We will begin the 1.9 release the first week of October 2017.

The proposal below is similar in layout previous releases:
- as before, key days aren't Fridays, since it can be hard to end milestones right up against weekends
- there are a number of North American holidays during Q4, so this release will
  have reduced scope and fewer features
- same as in 1.8
  * alpha's should not be cut unless all master-release-blocking tests are passing
  * release notes for planned features done 2 weeks out from feature freeze
  * docs PRs deadlines ahead of release

## 1.9 Release Schedule Key Dates
- Oct 4th (Wed) coding start

- Oct 25th (Weds): v1.9.0-alpha.2
- Oct 27th (Fri): Feature Freeze
- Nov 8th (Weds): v1.9.0-alpha.3
- Nov 22nd (Weds): v1.9.0-alpha.4
- Nov 22nd (Weds): Code Freeze 
- Nov 29th (Weds): v1.9.0-beta.0, v1.10.0-alpha.0
- Dec 6th (Fri): v1.9.0-beta.rc.1
- Dec 8th (Mon): v1.10.0-alpha.1
- Dec 13th (Weds):  v1.9.0 release!

## 1.9 Details

### Oct 5 - Nov 22
- 8 week coding period
- Release 1.9 alphas every 2 weeks
  - Alphas only cut if [release-master-blocking](https://k8s-testgrid.appspot.com/release-master-blocking) are passing
- Oct 11th (Weds): v1.9.0-alpha.1
- Oct 25th (Weds): v1.9.0-alpha.2
- Oct 27th (Fri): 
    * Feature Freeze: all features going into release should have an associated issue in the features repo
    * Release talking points draft due
    * Sig themes due in relnotes draft
- Oct 31st (Tues): 
    * First draft of feature related release notes due
    * Blog post draft for release team review
- Nov 8th (Weds): v1.9.0-alpha.3
- Nov 15th (Weds): 
    * Final draft of feature related release notes due
    * Finalize release talking points
- Nov 22nd (Weds): v1.9.0-alpha.4


### Nov 22 (Wed) - Dec 8 (Weds)
- Nov 22 Begin Code Slush
  * Hard deadline for feature-related PRs to be in submit queue with LGTM and approved label
  * Add milestone restriction on submit queue.
  * Community Feature Burndown Meetings held two or three times until release week (then every day). For those interested in joining please join [Kubernetes Milestone Burndown Group](https://groups.google.com/forum/#!forum/kubernetes-milestone-burndown)
  * Focus on bugfix, test flakes and stabilization
  * Ensure docs and release notes are written
  * Identify all features going into the release, and make sure alpha, beta, ga is marked in features repo
  * Release team to create release notes draft doc and share broadly
- Nov 29th (Weds): v1.9.0-beta.0
  * Branch Manager to create `release-1.9` branch and tests should be targeted
    to v1.9
  * v1.10.0-alpha.0 is cut
  * Test-infra lead to set up CI for release branch.
  * Begin regular fast-forwards (at least daily to keep CI up-to-date)
  * Manual downgrade work should begin
- Dec 1 (Fri)
  * All docs PRs must be open for tech review

### Dec 8th - Dec 13th
- Dec 8th (Weds): End of Code Freeze
  * v1.9.0-beta.rc.1 candidate cut
	  * RC means no known release-blockers outstanding.
	  * Only accept cherrypicks for release-blockers.
	  * Announce on mailing lists, Twitter, etc. to ask people to build and test the RC and submit feedback
	  * Perhaps managed k8s providers make rc.1 available on early access channels.
  * More RCs as needed to verify release-blocking fixes.
  * Final fast-forward of release branch.
  * sig-pm to make decisions regarding blog posts and marketing material
    releases and timeline
  * After this, all changes for the release must now be cherrypicked in batches by branch
  manager.
  * Remove milestone restriction on submit queue. 
  * Bot drains backlog.
  * Finalize blog post
- Dec 8th (Fri)
  * Docs PRs lgtm'd and ready for merge
- Dec 13th: Release 1.9!


### Post Release
- Dec 21st: Retrospective for 1.9

# Key features
[Feature tracking spreadsheet
link](https://docs.google.com/spreadsheets/d/1WmMJmqLvfIP8ERqgLtkKuE_Q2sVxX8ZrEcNxlVIJnNc/edit#gid=0)

## Operational Notes 
1. For the 1.9 release, the [release team](https://github.com/kubernetes/features/blob/master/release-1.9/release_team.md)
  will use the following procedure to identify release blocking issues
  1. Any issues listed in the [v1.9 milestone](https://github.com/kubernetes/kubernetes/issues?utf8=%E2%9C%93&q=is%3Aissue%20is%3Aopen%20milestone%3Av1.9)
     will be considered release blocking. It is everyone's responsibility to move non blocking issues out of the `v1.9` milestone. Items targeting 1.9 can be moved into the `v1.9` milestone.
     milestones **or the release will not ship**

# Contact us
- [via slack](https://kubernetes.slack.com/messages/sig-release/)
- [via email](mailto:kubernetes-release@googlegroups.com)
