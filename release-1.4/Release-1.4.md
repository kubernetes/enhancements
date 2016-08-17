#A Brief History
- Kubernetes 1.0 - July 21, 2015
- Kubernetes 1.1 - November 9, 2015 (+3mo19d or 15w6d)
- Kubernetes 1.2 - March 17, 2016 (+4mo8d or 18w3d - included holidays)
- Kubernetes 1.3 - July 1, 2016 (+3mo14d or 15w1d)

#Proposed timeline
Kubernetes 1.3 ships June 24 ([schedule](https://github.com/kubernetes/kubernetes/wiki/Release-1.3)) (actual was July 1).  To land 2 more releases in 2016, we should aim for 1.5 in early December (late December is a holiday period), and work back.  December 9 is a decent choice (and allows slip without landing during the holidays, if absolutely necessary).

A proposal so far has been to not immediately enter 1.4 after 1.3, and to actually build in a week for focusing on flakes and testing across our dev community.

To fit these milestones in, the best format appears to be 7w of coding, 4w of stabilization/release.  1.3 was 9w coding and 5w of stabilization.

Going from 14w to 11w per release seems feasible if we hold feature complete strongly (as we did for the first time in 1.3) and we make significant test/flake/queue investments (which we plan to do/are doing).  It also means if a given feature misses feature complete for 1.4, 1.5 isn't quite as far away, so it's a bit less painful.

###1.4 Overview
- June 27 flake week (1w)
- July 3 coding start (7w)
- Aug 19 feature complete, move to bugfix (4w) (Aug 17 update, FC moved one business day to Aug 22)
- Sept 16 release

###1.5 Overview
- Sept 19 coding start (7w)
- Nov 4 feature complete, move to bugfix (5w, includes US Thanksgiving holiday)
- Dec 9 release

##1.4 Details

###June 27 - July 1
- Flake and test fix-it week

###July 3 - Aug 19
- 7 week coding period
- Release 1.4 alphas every 2 weeks

###Aug 22 - Sept 2
- Enter code slush on head, no more features or major refactors
- Fix bugs and run tests
- Start Milestone Burndown meetings
- Branch and cut Beta release on Sept 2

###Sept 5 - Sept 16
- Open head for 1.5 work on Sept 5, after branch
- Fix bugs and run tests, update docs
- Release 1.4 on Sept 16
