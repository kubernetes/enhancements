_(approved; last update 5/19)_

#Learnings from 1.2
The k8s 1.2 release was laid out approximately as:
- 8w of coding, then declare "feature complete" (no major features or refactors allowed after here)
- 2w of bug fixes and testing, then release last Alpha (head in code slush these two weeks)
- 2w of bug fixes and testing, then declare Beta, release and branch (head in code slush these two weeks)
- 2w of bug fixes and docs work post Beta, then release 1.2 Final

None of these dates were considered "hard dates", but more of a guideline towards a 3 month release cycle.

In reality, many features were not finished in time for the feature complete date, which slipped about 2w (and even then still had some features finishing up after).  The branch/beta date ended up slipping 2w in line.  The final release ended up slipping 1.5w, so was brought in by a couple of days.  We ended up slipping many docs updates past the binary release.

In 1.3, we should probably make a few adjustments:
- check back on 1.3 blocking feature status more often (and get help from others as needed)
- minimize the list of 1.3 blocking features (nail those, and ship 1.3 with them + whatever else is done, but don't hold the release for anything not deemed blocking)
- add a week to feature coding at the expense of one bugfix week (as we continue to improve testing and stay on top of test flakes during the milestone, this should be ok)

#Proposed timeline
Kubernetes 1.2 shipped mid-March, so to maintain our 3m minor revision trend, we should aim for 1.3 in mid to late June.

That would look like:
- 9w of coding, then declare "feature complete" (no major features or refactors allowed after here)
- 1w of bug fixes and testing, then release last Alpha (head in code slush this week)
- 2w of bug fixes and testing, then declare Beta, release and branch (head in code slush these two weeks)
- 2w of bug fixes and docs work post Beta, then release 1.3 Final

###Mar 21 - Mar 25
- Feature coding week 1

###Mar 28 - Apr 1
- Feature coding week 2

###Apr 4 - Apr 8
- Feature coding week 3
- check in on status all 1.3 blocking features in Community Meeting

###Apr 11 - Apr 15 
- Feature coding week 4

###Apr 18 - Apr 22
- Feature coding week 5

###Apr 25 - Apr 29
- Feature coding week 6
- check in on status all 1.3 blocking features in Community Meeting, get help for some if needed

###May 2 - May 6
- Feature coding week 7

###May 9 - May 13
- Feature coding week 8
- check in on status all 1.3 blocking features in Community Meeting, get help for some if needed

###May 16 - May 20
- Feature coding week 9
- Feature Complete due end of day May 20 (slipped by one business day to May 23)

###May 23 - May 27
- Feature Complete due end of day May 23
- Bugfix week 1
- enter code slush
- cut final Alpha end of week

###May 30 - Jun 3
- Bugfix week 2
- US holiday on Mon May 30

###Jun 6 - Jun 10
- Bugfix week 3
- branch, cut Beta end of week, remove code slush on head

###Jun 13 - Jun 17
- Bugfix week 4

###Jun 20 - Jun 24
- release 1.3 Final

#Key features
[Feature tracking spreadsheet](https://docs.google.com/spreadsheets/d/1rrt179VjClAfMYnh8NwC_RLJbdI5j6zcVkW1z14vPR0)
