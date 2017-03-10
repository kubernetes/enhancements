#Proposed timeline for 1.6
We will begin the 1.6 release the first week of 2017.
We will use the time between releases for planning, designing and bugfixes.

The proposal below is identical in layout to the 1.4 and 1.5 plans with some
modifications:
- as before, key days aren't Fridays, since it can be hard to end milestones right up against weekends
- some sigs will focus on stabilization for 1.6, and there may be fewer new
  feature proposals
- there are fewer major holidays and vacation breaks this quarter 

##1.6 Release Schedule
- Dec 8 - Jan 3: planning/design/bugfix period
- Jan 3 (Tues) coding start (7w)
- Jan 24 (Tues) features repo freeze
- Jan 30th (Monday) v1.6.0-alpha.1
- Feb 13th (Monday) v1.6.0-alpha.2
- Feb 20th (Monday) Cut release-1.6 branch and v1.6.0-beta.0; Feature Burndown Meetings begin
- Feb 27th (Monday) Start code freeze
- Feb 28th (Tuesday) v1.6.0-beta.1
- March 7th (Tuesday) - v1.6.0-beta.2
- March 14th (Tuesday) Lift code freeze and v1.6.0-rc.1
- March 22nd (Wednesday) - v1.6.0 (release!)

##1.6 Details

###Jan 3 - Feb 27
- 7 week coding period
- Release 1.6 alphas every 2 weeks

###Feb 20 - Feb 24
- Community Feature Burndown Meetings held two or three times this week. For those interested in joining please
  join [the Google Group](https://groups.google.com/forum/#!forum/kubernetes-milestone-burndown)
- Identify all features going into the release, and make sure alpha, beta, ga is
  marked in features repo

###Feb 27 - Mar 22
- Enter code slush on head, no new features or major refactors
- Fix bugs, and test flakes
- Start Milestone Burndown meetings (2x per week until the week leading up the
  release, then every day)

###Mar 8 - Mar 22
- Open head for 1.7 work on Mar 14, after 1.6 release branch has been fast-forwarded.
- Fix bugs and run tests, update docs
- Branch and cut second beta release on Mar 7 (Tuesday)
- Branch and cut release candidate on Mar 14 (Tuesday)
- Release 1.6 on March 22 (Wednesday)


#Key features
[Feature tracking spreadsheet
link](https://docs.google.com/spreadsheets/d/1nspIeRVNjAQHRslHQD1-6gPv99OcYZLMezrBe3Pfhhg/edit#gid=0)

## Notable operational changes

1. Starting in the 1.6 release the [release team](https://github.com/kubernetes/features/blob/master/release-1.6/release_team.md)
  will use the following procedure to identify release blocking issues
  1. Any issues listed in the [v1.6 milestone](https://github.com/kubernetes/kubernetes/issues?utf8=%E2%9C%93&q=is%3Aissue%20is%3Aopen%20milestone%3Av1.6)
     will be considered release blocking. It is everyone's responsibility to move non blocking issues out of the `v1.6` milestone. Items targeting 1.7 can be moved into the `v1.7` milestone.
     milestones **or the release will not ship**
  1. The release team *will not* use or consider the `release-blocking` or `non-release-blocking` labels although they may have meaning
     for other members of the community
  1. The release team *will not* user or consider `priority/*` labels of any kind. There is an ongoing migration away from `priority/p[0-9]` labels
     in favor of `priority/[a-z]` labels and usage is inconsistent across the project.
  1. `priority/p*` labels will continue to be used by the submit queue

# Contact us
- [via slack](https://kubernetes.slack.com/messages/k8s-release/)
- [via email](kubernetes-release@googlegroups.com)
