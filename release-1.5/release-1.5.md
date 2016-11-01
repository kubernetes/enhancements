(approved at Aug 25 community meeting)

#1.5 Tenative Timeline

###September 19 - Monday, November 7, 2016
- [x] **Monday, September 19, 2016**
  - 7 week coding period begins
  - 1.5 alpha releases are cut every 2 weeks during this period.
- [x] **Monday, October 10, 2016**
  - *Feature freeze* begins
    - All features that planned for v1.5 must be defined in the [features repository with the 1.5 milestone label](https://github.com/kubernetes/features/issues?q=is%3Aopen+is%3Aissue+milestone%3Av1.5) by this date.
- **Monday, November 7, 2016**
  - Feature Complete Date
    - Final day to merge non-bug related code changes for the v1.5 release.
    - All feature code must be LGTMed with tests written and in the submit queue by 6 PM PST.

###November 8 - November 18, 2016
- **Tuesday, November 8, 2016**
  - *Code freeze* begins
    - Only bug fixes with the `v1.5` milestone will be merged to `master` branch after this date.
    - All other changes must go through the [exceptions process](https://github.com/kubernetes/features/blob/master/EXCEPTIONS.md)
      - All requests for exception must be submitted by Nov 8, 6 PM PST.
- **Wednesday, November 9, 2016**
  - [Milestone Burndown](https://groups.google.com/forum/#!forum/kubernetes-milestone-burndown) meetings begin
    - All requests for exception will be reviewed and either approved or rejected during the first meeting.
    - Requesters will be notified within 24 hours.
- **Friday, November 18, 2016**
  - 1.5 release branch fast-forwarded to match `master` branch (picking up all changes merged since code freeze).
  - 1.5 Beta released

###November 18, 2016 - December 8, 2016
- **Friday, November 18, 2016**
  - `master` branch is opened for 1.6 work after 1.5 release branch has been fast-forwarded.
  - All bug fixes after this point must be manually cherry-picked to the 1.5 release branch.
- **Friday, November 25, 2016**
  - Docs for all [1.5 features](https://github.com/kubernetes/features/issues?q=is%3Aopen+is%3Aissue+milestone%3Av1.5) should have PRs out for review.
  - Include a link to the relevant 1.5 feature in the Docs PR.
- **Friday, December 2, 2016**
  - Docs for all [1.5 features](https://github.com/kubernetes/features/issues?q=is%3Aopen+is%3Aissue+milestone%3Av1.5) must be merged.
- **Thursday, December 8, 2016**
  - Release 1.5

#Key features
[Feature tracking spreadsheet (draft)](https://docs.google.com/spreadsheets/d/1g9JU-67ncE4MHMeKnmslm-JO_aKeltv2kg_Dd6VFmKs/edit?usp=sharing)

#Why?
Kubernetes 1.4 is set to release on Sept 20.  We want to have another release of Kubernetes in the 2016 calendar year, so that means in December.

December tends to have a lot of vacation time towards the end, and we want to have a little buffer time in case of slips.  Late November also has the US Thanksgiving holiday, when many people will be on vacation.

The proposal below is identical in layout to the 1.4 plan, with the exceptions of:
- key days aren't Fridays, since it can be hard to end milestones right up against weekends
- a week is added for the bugfix period due to the Thanksgiving holiday
- KubeCon is Nov 8-9, Kubernetes Dev Summit - Nov 10.
