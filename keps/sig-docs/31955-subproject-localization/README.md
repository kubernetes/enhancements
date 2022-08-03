# KEP-31955: SIG Docs Localization Subproject

- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Localizations across our Kubernetes documentation have been spinning up and making great progress, ensuring greater inclusivity across the global users of our open source work. These localizations have also made an impact on how we make decisions about large changes to our documentation, and to this end, we want to empower our localization groups to have more autonomy in these processes. We'd also like to formally recognize their efforts as a way to reward, and further encourage, localization efforts across multiple languages, locations, and timezones.

## Motivation

It’s important to make sure that folks who are working on different localizations also have a say in how their work gets prioritized and trickled down from the main English language documentation. This is why SIG Docs wants to formally create a Localization subproject within SIG Docs. The localization community, via an official subproject, will be able to have a larger say in the goings-on of the SIG, as well as be involved in larger decisions and changes. We would also expect this recognition to help fuel the contributing pipeline across languages, which could eventually trickle down into more contributors for the entire SIG.

### Goals

* Formalize localization efforts and processes in a subproject
* Increase resources for current and new localizations
* Create leadership roles that are involved in SIG-wide decision-making

### Non-Goals

* Overcomplicate the localization process
* Gatekeep efforts to spin up new languages for Kubernetes documentation

## Proposal

SIG Docs is proposing to formalize localization efforts by creating an official localization subproject. Owners of this subproject have already been identified in @bradtopol and @Abbie. Setting up an official localization subproject has the following goals:

* Recognize localization efforts across multiple languages
* Involve localization contributors in larger decisions across SIG Docs
* Increase and/or create new resources for current and new localizations
* Increase our localization contributor pool
* Spin down inactive localizations as needed

Governance for this subproject wouldn't diverge from the already established governance of SIG Docs.

One measurement of success via subproject creation would be the stability of our current localizations, and their ability to implement downstream changes coming from our English documentation, alongside decisions from SIG Docs. An example of this is the current Dockershim deprecation and how the removal of Dockershim references in our English documentation trickles down successfully to other languages. Larger navigation or design changes to our docs also need localization involvement and support, hence, the subproject's role in ensuring this support will be a crucial success metric. Possible metrics that illustrate this support are:

* Number of PRs open on average (indicating progress being made to implement updates and changes)
* Number of reviewers for each localization (having a minimum, with onboarding possibilities from the subproject)
* Resources existing in several languages to increase the contributor pool

Open source is truly global and multilingual, so its important for the Kubernetes project to facilitate contributions from technologists around the world in their own language. Spinning up an official subproject to solidify localization efforts will mean greater resources to achieve this aim, uniting folks around the common goal of making Kubernetes accessible to people across multiple languages, locations, and timezones.

### Notes/Constraints/Caveats (Optional)

A monthly localization meeting is already on the SIG Docs calendar and attended regularly by some language representatives.

### Risks and Mitigations

The main risks are abandonment of localization efforts and staleness of localizations as the English site changes. We've already seen larger changes to English documentation be harder to take on and implement down the localization funnel, hence, having leads of an official subproject involved in those discussions and decisions will hopefully mitigate those issues. We see this occurring via regular communication in SIG Docs bi-weekly meetings, alongside the regularly monthly meetings that localization contributors are already having.

## Drawbacks

Localization work already exists without an official subproject, with approximately 10 languages noted as current in terms of reviewers, Slack channels, and PRs. It could be argued that creating an official subproject isn't needed to facilitate localization efforts, however, we're choosing to concentrate on the growth, scalability, and maintainability of our localizations, hence this subproject proposal.

## Alternatives

Unofficial efforts to standardize on localizations already exist and are the current alternative measure in place. However, we're seeing a lack of decision-making power effect the progress of updates and changes trickling down from our English documentation. Hence, we are ruling out continuing down an unofficial path so we can empower localization contributors, alongside growing our contributor base via subproject creation and recognition.
