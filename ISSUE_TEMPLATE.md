
BEFORE YOU CLICK CREATE ISSUE:

- Please first build consensus within the appropriate SIG or in github.com/kubernetes/kubernetes/issues
  that the problem you are trying to solve is worth solving at this time.
- Put a 2-3 sentence description of your feature under the **Description** heading below.
- Delete from the top of this text down to the **Description** heading.
- Read the text below.

Thanks for wanting to add a feature to Kubernetes!  You will be responsible for guiding
your feature through completion, and asking the right people for approvals.  

Large features typically go through three stages: [Alpha, Beta, and Stable](https://github.com/kubernetes/kubernetes/blob/master/docs/api.md#api-versioning)
Each stage requires various approvals from various teams.  Features require several releases
to reach Stable.

The following items have to be filled before the features freeze:

- Primary contact (assignee);
- SIG, responsible for the feature;
- Proposal link - do the public design proposal in the community repo;
- Reviewer(s) - recommend having 2+ reviewers (at least one from code-area OWNERS file) agreed to review. Reviewers from multiple companies preferred;
- Approver (for LGTM, likely from SIG to which feature belongs);
- Target - Alpha / beta / stable;
- Timeline / status;

Before the coding freeze:

- Release note - a short sentence, describing the feature;
- User documentation PR link (to http://github.com/kubernetes/kubernetes.github.io)


** Delete from here to top of input box before creating issue so that issue creation emails will be informative.**

# Description

**Add description here**


# Progress Tracker

Please, select the following checkboxes and fill with the necessary data after completing the desired steps: 

Before the features freeze:
- [ ] Primary contact
- [ ] SIG
- [ ] Proposal link 
- [ ] Reviewer(s)
- [ ] Approver
- [ ] Target
- [ ] Timeline / status

Before the coding freeze:
- [ ] Release note
- [ ] Documentation


More advices:

Design
   - Once you get LGTM from a *`@kubernetes/feature-reviewers`* member, you can check this checkbox, and the reviewer will apply the "design-complete" label.
 
Coding
  - Use as many PRs as you need.  Write tests in the same or different PRs, as is convenient for you.
  - As each PR is merged, add a comment to this issue referencing the PRs.  Code goes in the http://github.com/kubernetes/kubernetes repository,
        and sometimes http://github.com/kubernetes/contrib, or other repos.
  - When you are done with the code, apply the "code-complete" label.
  - When the feature has user docs, please add a comment mentioning `@kubernetes/feature-reviewers` and they will
        check that the code matches the proposed feature and design, and that everything is done, and that there is adequate
        testing.  They won't do detailed code review: that already happened when your PRs were reviewed.
        When that is done, you can check this box and the reviewer will apply the "code-complete" label.

Docs
  - Write user docs and get them merged in.
  - User docs go into http://github.com/kubernetes/kubernetes.github.io.
  - When the feature has user docs, please add a comment mentioning `@kubernetes/docs`.
  - When you get LGTM, you can check the `documentation` checkbox, and the reviewer will apply the "docs-complete" label.

