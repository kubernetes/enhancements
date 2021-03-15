<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->

# KEP-2572: Defining the Kubernetes Release Cadence

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [FIXME](#fixme)
    - [TODO Deterministic](#todo-deterministic)
    - [TODO Reduce risk](#todo-reduce-risk)
    - [Data](#data)
  - [Goals](#goals)
    - [FIXME Does that mandate a fixed frequency?](#fixme-does-that-mandate-a-fixed-frequency)
    - [FIXME Releases don’t necessarily have to be equally spaced](#fixme-releases-dont-necessarily-have-to-be-equally-spaced)
  - [Non-Goals](#non-goals)
    - [TODO Release Team](#todo-release-team)
    - [TODO Enhancement graduation](#todo-enhancement-graduation)
    - [FIXME Ideas](#fixme-ideas)
    - [FIXME Comment, without decision](#fixme-comment-without-decision)
    - [FIXME Needs response](#fixme-needs-response)
  - [FIXME Explanatory](#fixme-explanatory)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [TODO End User](#todo-end-user)
    - [TODO Distributors and downstream projects](#todo-distributors-and-downstream-projects)
    - [TODO Contributors](#todo-contributors)
    - [TODO SIG Release members](#todo-sig-release-members)
      - [TODO @neolit123](#todo-neolit123)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [TODO Schedule](#todo-schedule)
  - [FIXME Implementation Details](#fixme-implementation-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [FIXME](#fixme-1)
    - [LTS](#lts)
    - [Go faster](#go-faster)
    - [No](#no)
    - [Maintenance releases](#maintenance-releases)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
- [FIXME Cleanup](#fixme-cleanup)
  - [How do we make a decision?](#how-do-we-make-a-decision)
    - [Canonical](#canonical)
    - [Alternatives](#alternatives-1)
  - [Do we have any data?](#do-we-have-any-data)
  - [How do we implement?](#how-do-we-implement)
  - [Conversations](#conversations)
    - [Leads meeting feedback](#leads-meeting-feedback)
    - [From Jeremy](#from-jeremy)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### FIXME

What would you like to be added:

We should formally discuss whether or not it's a good idea to modify the kubernetes/kubernetes release cadence.
Why is this needed:

The extended release schedule for 1.19 will result in only three minor Kubernetes releases for 2020.

As a result, we've received several questions across a variety of platforms and DMs about whether the project is intending to only have three minor releases/year.

In an extremely scientific fashion, I took this question to a Twitter poll to get some initial feedback: https://twitter.com/stephenaugustus/status/1305902993095774210?s=20

Of the 709 votes, 59.1% preferred three releases over our current, non-2020 target of four.

There's quite a bit of feedback to distill from that thread, so let's start aggregating opinions here.

Strictly my personal opinion:
I'd prefer three releases/year.

    less churn for external consumers
    one less quarterly Release Team to recruit for
    for a lot of folx, there are usually multiple things happening at the quarterly boundary which a new Kubernetes release can steal focus from
    we can collectively use the ~3 months we'd gain for things like triage/roadmap planning, personal downtime, stability efforts

@kubernetes/sig-release @kubernetes/sig-architecture @kubernetes/sig-testing
/assign
/milestone v1.20
/priority important-longterm

#### TODO Deterministic

@akutz:

> I strongly believe a deterministic and known schedule is more important than the frequency itself. Slowing down to three releases, as @justaugustus said, will provide three additional months for triage and the addressing of existing issues. This should help us to better meet planned release dates as there, in theory, should be fewer unknown-unknowns. So a big +1 from me.

#### TODO Reduce risk

@Klaven:

> I see some people attributing drift to longer release cycles (we are only talking about extending them by a month, not 3 months), but I would argue that fast release cycles have caused their own amount of drift, never mind the burden on the release team.
>
> Look at GKE, for example. Versions 1.14 to 1.17 are supported. GKE is arguably one of the best Kubernetes providers and there is a LOT of drift because corporations don't like continuous rapid change and find it hard to support. I also think that as a project matures the rate of change of the increasingly stable and feature-complete core should decrease. At some point the plugins and the out-of-tree projects should be where more change happens. Projects like cluster-api and the like get the attention which used to be focused on maturing the core.
>
> I know that recently there has been a lot of focus on how many releases something needs in order to become GA. I think that honestly is the wrong approach.
>
> I do think it's valid to be concerned that the release of k8s is too much work. I would say this means this system is too laborious. Given that it is this much work, rushing it more would probably only hurt us more. It's obvious that the ecosystem has already felt this strain. If we want to be able to release frequently, we need the release process to become painless. If we don't fix that problem, I don't see any solution other then pushing releases to a manageable cadence.
>
> If we want to release quickly, we need to think not only about the release team, but also the downstream; the adopters. If we want to release quickly and frequently, then we need to focus on making the upgrade process even easier and similar things.

@ehashman:

> While some folks in the thread note that this increases the heft/risk of each release, I actually think less releases will reduce risk. I'm speaking from an operations perspective as opposed to a development perspective.
>
> The current Kubernetes release cadence is so high that most organizations cannot keep up with making a major version update every 3 months regularly, or going out of security support in less than a year. While in theory, releasing more frequently reduces the churn and risk of each release, this is only true if end users are actually able to apply the upgrades.
>
> In my experience, this is very challenging and I have not yet seen any organization consistently keep up with the 3 month major upgrade pace for production clusters, especially at a large scale. So, what effectively happens is that end users upgrade less frequently than 3 months, but since that isn't supported, they end up in the situation where they are required to jump multiple major releases at once, which effectively results in much higher risk.
>
> 4 vs. 3 releases is >30% more release work, but I do not believe it provides benefit proportional to that work, nor does a quarterly major release cadence match the vast majority of operation teams' upgrade cycles.

#### Data

@jberkus:

> All: I'm going to research what actual feature trajectory looks like through Kubernetes, because @johnbelamaric has identified that as a critical question. Stats to come.

@ehashman:

> @jberkus Any updates on the stats? :)

@jberkus:

> Nope, got bogged down with other issues, and the question of "what is a feature" in Kubernetes turns out to be a hard one to answer. We don't actually track features outside of a single release cycle; we track KEPs, which can either be part of a feature or the parent of several features, but don't match up 1:1 as features. So first I need to invent a way to identify "features" in a way that works for multiple release cycles.

@jberkus

> Sorry this has been forever, but answering the question of "how fast do our features advance" turns out to be really hard, because there is literally no object called a "feature" that persists reliably through multiple releases.
>
> To reduce the scope of the problem, I decided to limit this to tracked Enhancements introduced as alpha in 1.12 or 1.13, which were particularly fruitful releases for new features. Limiting it to Tracked kind of limits it to larger features, but I think these are the only ones required to go through alpha/beta/stable anyway (yes/no?). So, in 1.12 and 1.13:
>
> 20 new enhancements were introduced
> 7 did not follow a alpha/beta/stable path, mostly because the were removed or broken up into other features
> 2 are still beta
> 1 advanced in minimum time, that is 1 release alpha, 1 beta, then stable, in 9 months
> 4 advanced from alpha to beta in 1 release, but then took 2 or more releases to go to stable
> 7 advanced more slowly
>
> Our median number of releases for an enhancement to progress is:
>
> Alpha to Beta: 2 releases
> Beta to Stable: 3 releases
> Alpha to Stable: 6 releases
>
> Given this, it does not look like moving to 3 releases a year would slow down feature development  due to the alpha/beta/stable progression requirements.
>
> I will note, however, that for many enhancements nagging by the Release Team during each release cycle did provide a goad to resume stalled development. So I'm at least a little concerned that if we make 3/year and don't change how we monitor feature progression at all, it will still slow things down because devs are being nagged less often.

@ehashman

> I am a little less worried about the "nag factor" now that we've moved to push-driven enhancements this release (SIG leads track, enhancements team accepts vs. enhancements team tracks).

@jberkus

> I'm very worried about it, see new thread.

@johnbelamaric

> I am more concerned about the "nag factor" due to the move to push-driven
> development; I think the lack of nagging will slow some features down.
> However, there are new things at play that will help with it - namely, the
> "no more permabeta" where things can't linger in beta forever because their
> API gets turned off automatically. At least for things with APIs.
>
> I strongly believe our alpha-to-stable latency, already quite high, will
> get worse with 3 releases per year. But ultimately, it's up to the feature
> authors. If they want it to go fast enough, they'll have to push for it
> more. Missing a train will have higher cost.
>
> Anyway, I personally am, as I said before, ambivalent on this decision.
> Putting on my GKE hat, it makes my life easier. Putting on my OSS hat, I
> have concerns but nothing that would make me strongly oppose it. It's not a
> change I would push for, but I think it's reasonable to see what happens if
> we try it.
>
> And thank you Josh for the analysis. The median 6 releases goes from 18
> months to 24 months, which is not great but also something that is not
> forced on feature authors - they could push and get it done in less time if
> they need to. It's a rare feature that was making it in the 9 months
> before, so it would be a rare feature that is forced to have a longer cycle
> than they would have otherwise.

@jberkus

> I'm going to start a new thread on nag factors, because these are a bigger deal than I think folks want to address.
>
> There are two areas in the project, currently, that are almost entirely dependent on release team nagging (herafter RTN) for development:
>
>    Getting features to GA (and to a lesser degree, to beta)
>    Fixing test failures and flakes
>
> With the current 3-month cycle, this means that for 1 month of every cycle RTN doesn't happen, and as a result, these two activities don't happen. This is an extremely broken system. Contributors should not be dependent on nagging of any sort to take care of these project priorities, and the project as a whole shouldn't be depending on the RT for them except during Code Freeze.
>
> A 4-month cycle will make this problem worse, because we'll be looking a 2 months of every cycle where RTN won't happen, a doubling of the amount of time per year for tests to fail and alpha features to be forgotten.
>
> I am not saying that this is a reason NOT to do a 4-month cycle. I am saying that switching to a 4-month cycle makes fixing our broken RTN-based system an urgent priority. Fixing failing tests needs to happen year round. Reviewing features for promotion needs to happen at every SIG meeting.
>
> (FWIW, this is an issue that every large OSS project faces, which is why Code Freeze is to awful in so many projects)

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

#### FIXME Does that mandate a fixed frequency?

Thoughts:
Roughly, yes.

- Release cycle
- Planning / stability phase
- Repeat

As a consumer, I'd be looking for some predictability in the schedule.

#### FIXME Releases don’t necessarily have to be equally spaced

See point on predictability.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

#### TODO Release Team

@saschagrunert:

> On the other hand it will give less people a chance to participate in the shadowing program for example. Anyways, I think we will find appropriate solutions around those kind of new challenges.

@jeremyrickard:

> The one downside is that we will remove an opportunity for shadowing, and as we saw this time around we had >100 people apply, and this will remove ~24-ish opportunities. I think we can maybe identify some opportunities for folks that want to be involved though. takes off release lead hat

@wilsonehusin:

> as someone who started getting involved in this project through shadowing release team, I'd like to echo what @saschagrunert & @jeremyrickard raised above regarding shadow opportunities -- I'm glad we're acknowledging the downside and hope we can keep in mind to present other opportunities for folks to get involved!

@kcmartin:

> As to the potential for limiting shadow opportunities (mentioned by @jeremyrickard, @wilsonehusin, and others), I'm definitely tuned in to that being a downside, since I've served as a SIG-Release shadow three times, and I think it's a fantastic opportunity!
>
> One possible way to alleviate that downside would be to have 5 shadows, instead of three or four, per sub-team. I believe this is still a manageable number for the Leads, and could distribute the work more evenly.

@pires:

> On a more personal note, (@jeremyrickard wink, wink) I applied for release shadow believing I'd be picked given my past contributions to the project and my justification to be selected over others. Being rejected was a humbling experience and I'm happy to let you know I didn't lose any of the appetite to contribute. Others may feel differently but, then again, the project is maturing and so should the community.

#### TODO Enhancement graduation

@jberkus:

> @johnbelamaric everything you've said is valid. At the same time, though, my experience has been that the pressure goes the other way: features already linger in alpha or beta for way longer than they ought to. The push to get most features to GA -- or deprecate them -- really seems to be lacking. It's hard to pull stats for this, but most KEP-worthy features seem to take something like 2 years to get there. So from my perspective, more state changes per release would be a good thing (at least, more getting alpha features to beta/GA), even if we didn't change the number of releases per year.
>
> It's hard to tell whether or not switching to 3 releases a year would affect the slow pace of finishing features at all.

#### FIXME Ideas

@sftim:

> If there were an unsupported-but-tested Kubernetes release cut and published once a week - what would that mean?
>
> I'm imagining something that passes the conformance tests (little point otherwise) but comes with no guarantee. The Rust project has a model a bit like this with a daily unstable release which has nevertheless been through lots of automated testing.
>
> When I'm typing this I'm imagining that I could run minikube start --weekly-unstable and get a local test cluster based on the most recent release. If Kubernetes already had that built and working, would people pick different answers?

@jberkus:

> @sftim yeah, you've noticed that the reason, right now, we don't see a lot of community testing on alphas and betas is that we don't make them easy to consume.
>
> I'd say that it would need to go beyond that: we'd need images, minikube, and kubeadm releases for each weekly release.
>
> I don't know how that would affect our choice of major release cadence (isn't it orthagonal?) but it would be a great thing to do regardless. Also very hard.

#### FIXME Comment, without decision

@sftim:

> I look forward to using the mechanisms already in place (notably CustomResourceDefinition, but also things like scheduling plugins) to enhance the Kubernetes experience outside of the minor release cycle.
>
> A bit more decoupling, now that the investment is made to enable that, sounds good to me - and allows for minor releases of Kubernetes itself to become less frequent.

#### FIXME Needs response

@aojea:

> 3 releases is cool for development, but not for releasing something with a minimum level of quality.
> We barely keep with the tech debt we have in CI and testing, ie, how many jobs are failing for years that nobody noticed?, how many bugs are open for years? how many features are in alpha,beta for years?
> Each release cycle force people to FIX things if they want to release, the more time to release the more technical debt that you accumulate.
> At least in all my life I never see a project that reducing the release cycle you don't end rushing everything for last week and honestly, I gave up believing that will be real some time.

### FIXME Explanatory

@MIhirMishra:

> What is the need to decide in advance ? Release when it is ready for its level i.e. if it is ready for beta - release as beta and when ready for GA release as GA.
> More important is what is in the release than how frequently you are releasing.

@johnbelamaric:

>     What is the need to decide in advance ? Release when it is ready for its level i.e. if it is ready for beta - release as beta and when ready for GA release as GA.
>
> Not sure I understand the question. No one is suggesting deciding in advance - features will be advanced in their stage when they are ready. But the thing is, "when they are ready" depends on the release cadence. In order to go to beta, a feature has to have at least one alpha. In fact, realistically there will be more than one release with it in alpha, since it's really difficult to get meaningful feedback with a single cycle of alpha. Arguably, this becomes a little easier with longer time between releases, but realistically going from alpha release, to availability in downstream products, to real usage, to feedback and updated design and development is pretty hard to squeeze in before code freeze for the next release.
>
> Another way to think about this is that every feature goes through three state transitions:
>
>     inception -> alpha
>     alpha -> beta
>     beta -> GA
>
> Thus, the minimum number of releases to get from inception to GA is three - about 9 months now versus 12 with the proposed schedule. Now, it is the rare feature that would be able to do this in 9 months, because we general need more than one release of alpha, and probably for beta too to get a decent signal on quality.
>
> At the same time, a constant level of effort exerted on K8s would mean that the same number of features could have state transitions in the same amount of time. With fewer releases per year, that means more state transitions per release.
>
> That is, elongating the cycle creates: more content (transitions) per release, and longer time for a given feature to transition through all the states. Higher latency (in terms of time) with more throughput (in terms of releases).
>
> We don't want higher latency with more throughput. Because more throughput means riskier and more difficult upgrades and higher latency means more pain for developers and their customers.
>
> So, what are the mitigations? They amount to reducing the number of transitions per release (to address my (1) above) and reducing the number of transitions per feature (to address my (2) above).
>
> Reducing the transitions per release can be done by:
>
>     peeling features off of the monolithic release by pushing them out-of-tree (decomposition)
>     SIGs saying "no" more (which pushes things out-of-tree, most likely)
>     Requiring a higher bar for a state transition, thus making the effort involved to get to the next stage higher
>
> Reducing the transitions per feature can only be done by changing our policy. In order to do that, we would certainly need to raise the bar higher for state transition - this dovetails with our goal per release. Another possibility is to classify features into low and high risk features. Low risk features could go straight to beta and skip the alpha phase, for example.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

TODO: Add general note about state of the world and human challenges

#### TODO End User

- Hard to keep up with four releases / too much churn
  - TODO: Get data from SIG Cluster Lifecycle
  - One quarter where teams cannot focus on infra work
    - 16 weeks with 4 weeks for holiday buffer works well

@OmerKahani:

> 3 is the maximum upgrades that we can do in my company. Our customers are eCommerce merchants, so from September to December (include), we are in a freeze on all of the infrastructure.

#### TODO Distributors and downstream projects

https://www.cncf.io/certification/software-conformance/

- Keeping up for both installers and cluster addons
- Cloud provider parity
- Less upgrades helps complex workloads

@leodido:

> +1 for 3 releases.
>
> Making users able to catch-up is more important than keeping a pace so fast that can lead nowhere (we experienced the same with https://github.com/falcosecurity/falco and we switched to 6 releases per year from 12).

@afirth:

> I guess most end users are blocked by their upstream distro's ability to keep up with the K8s release. For example, GKE rapid channel is currently on 1.18, but 1.19 released in August. Somebody previously mentioned kops has similar issues (also currently on 1.18). I'm curious whether this is because those providers routinely find issues, or because it takes some fixed time to implement the new capabilities and changes. Either way, I don't think this change would impact end user's ability to get new features in a timely fashion much.

#### TODO Contributors

- Time for project enhancements
- Time for feature development
- Time for planning / KEPs
- Time for health and well-being of tests
- Time for mental health / curtailing burnout
- Time for KubeCon execution
- Further show of maturity with less churn

@pires:

> And as noted over Twitter, given someone's concerns on expecting same amount of changes over 25% less releases, I think it's of paramount importance for SIGs to step up and limit the things they include in a release, balancing what matters short/ long-term and kicking out all that can be done outside of the release cycle (we have CRDs, custom API servers, scheduling plug-ins, and so on). Now, I understand it's hard, sometimes even painful, to manage the enthusiasm some like me have on things close to them they want to see gaining traction but the early days are gone and this is now a solid OSS project that requires mature contributors.

#### TODO SIG Release members

- Reduce management overhead for SIG Release / Release Engineering
- With the yearly support KEP, we only have three 3 releases to maintain
- One less quarterly Release Team to recruit for

##### TODO @neolit123

>     Of the 709 votes, 59.1% preferred three releases over our current, non-2020 target of four.
>
> i find these results surprising. we had the same question in the latest SIG CL survey and the most picked answer was "not sure". this tells me that users possibly do not understand all the implications or it does not matter to them.
>
> a couple of benefits that i see from the developer side:
>
>     with the yearly support KEP we get 3 releases to maintain
>     less e2e test jobs
>
> as long as SIG Arch are on board we should just proceed with the change.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### TODO Schedule

@mpbarrett:

> Which months (ie April, August, Dec would be every 4 months from the beginning of a year) would be important for me as a user to know.

@OmerKahani:

> @tpepper for your question - the best month for us will be March, July, November-December.

### FIXME Implementation Details

@tpepper:

> Can folks comment on how they'd prefer this to look operationally?
>
> There are operational benefits to consumers in having to do less upgrades (even with the small risk that a longer cadence means each upgrade is bigger), but that's also somewhat depending on when those are presented. Should we maybe rotate the release points backwards or forwards to get away from having a release at a particular time when it's less consumable? We also need to consider how to make this beneficial or at least not super disruptive to community muscle memory.
>
>     symmetric four months active dev each, the ~3 months we'd gain for things like triage/roadmap planning, personal downtime, stability efforts spread across that. Are there specific benefits to picking any of:
>         releases in April, August, December?
>         releases in March, July, November?
>         releases in February, June, October?
>         releases in January, May, September?
>     asymmetry with some explicit downtimes using the ~3 months we'd gain for things like triage/roadmap planning, personal downtime, stability efforts? Eg: late November, December, early January and also August have typically been slower times already.
>         quiet December, dev activating more January through April release
>         dev May through July release
>         August stability scrubbing
>         dev September through early December release
>     some other explicit stability effort periods like taken in August 2020, formalized into the release cadence as a longer code freeze?
>
> Depending on how we lay out the annual calendar, we might also get some benefit (or complexity) relative to the golang release cycle and our need to update golang twice on average across the patch release lifecycle of each release. This may also compound relative to distributors release cadences, support lifecycles, their stabilization and lead time between content selection and release, and their balancing interoperability across a larger set of dependencies where those are tied to specific months on the annual calendar.
>
> There's not an easy answer here that's going to work right for everybody, but while lots of folks are +1'ing the abstract concept it would be good to capture additional constraints and ideas that are otherwise implicit when they make the +1.

@neolit123:

> Golang has a "symmetric" model, so i think k8s should do the same.
> the "symmetric" choice however, would require more discipline and availability from contributors, so my vote here is to try "symmetric" and if it fails (maybe after one year) go "asymmetric".

@khenidak:

> @tpepper would be great if we add to this post typical kubecon(s) schedule, since most the community (those who are reviewing, approving, building, releasing changes) is also heavily engaged in these events.

@vincepri:

> Trying to summarize general feedback I've gathered today here and there for visibility. This particular issue started with "slowing down", although it quickly became a reflection on why and what can we do about it:
>
>     Releases are hard, they need people to commit.
>     There is not enough automation (true for probably all our projects and repositories).
>     With each Kubernetes release, there is a world of clients that needs to be updated.
>     Changes are hard to keep up with, and sometimes important things are buried in release notes.
>     Some fear that less releases without policies (read: saying "no" more) isn't enough.
>     2020
>
> Taking a step back, a few people are suggesting to fix some of these problem from a technical perspective (which is good in itself) and we should prioritize these efforts. From the other side, there is a general sentiment that we need to slow down for the sake of this community's health.
>
> These are both valid, and agreeing on a slower cadence is just the first step; going forward we should normalize taking a few steps back, reflect, and course-correct when things are becoming unsustainable.

@cblecker:

>     Can folks comment on how they'd prefer this to look operationally?
>
> We should also talk about how this may impact things like code freeze (longer feature freeze with only bug/scale/stability fixes?).
>
> I'm +1 to 3 releases in general, but the details obviously matter. I'd also love to see this in a KEP if we come to consensus on making a change.

@jberkus:

> So, some pros/cons not previously mentioned on this thread:
>
> Additional Pros:
>
>     Easier to schedule releases because we can fudge dates more to avoid Kubecons and holidays
>     Reduced number of E2E, conformance, skew, and upgrade test jobs
>     Means that new 1+ year patch support doesn't result in additional patch releases
>     We'll get to 1.99 slower so we won't run out of digits
>
> Cons:
>
>     Increased pressure by feature authors to get their feature in this release as opposed to waiting.
>     Extra month for flakes/failures to get worse if nobody looks at them until Code Freeze
>     Extra problems with upstream dependency patch support if our timing is bad
>
> Regarding Tim's question of exact implementation:
>
> My vote is for symmetric releases in April, August, and December. While it's tempting to make December an "off month", development does happen all the time, and if it's not on a release, what is it on?
>
> That would be a reason for my 2nd choice, which would be Symmetric March, July, November, which puts Slow December at the beginning of the cycle instead of the end. However, that's mainly a benefit for working around Kubecon November, and there's no good reason to believe that Kubecon will be happening in November 2 years from now; it might be September or October instead.

@alculquicondor:

>     Extra month for flakes/failures to get worse if nobody looks at them until Code Freeze
>
> I think risk this can be reduced if we adjust (increase) the code freeze period.

@jberkus:

>         Extra month for flakes/failures to get worse if nobody looks at them until Code Freeze
> 
>     I think risk this can be reduced if we adjust (increase) the code freeze period.
>
> Better, how about we keep on top of flakes even if it's not Code Freeze?

@alculquicondor:

>     Better, how about we keep on top of flakes even if it's not Code Freeze?
>
> Absolutely, but it's not easy to enforce without hurting development of non-violators.

@jberkus:

> Shutting down merges is the nuclear option for preventing flakes and fails. We should be able to keep on top of them without resorting to that. But ... we're getting off topic here, unless folks think the "increased time to flake" is a blocker for this (I don't).

@johnbelamaric:

> I am tentatively in favor of 3 releases per year, primarily because I believe 4 releases per year is too hard for folks to consume. Even 3 releases per year is probably too much for most, but the downsides of fewer releases make anything less than 3 too risky in my mind.
>
> As I see it, those downsides are some things already mentioned above:
>
>     Build up of too much content in the release, and consequent potential for more painful upgrades.
>     Very long lead time to get a feature to GA through the alpha/beta/stable phases.
>
> Before making this decision, I think we need mitigations for these. Those mitigations have extensive ripples in how we do our development.
>
> For (1), some mitigations are:
> a) More development out-of-tree / decoupling more components
> b) SIGs saying "no" more
> c) Stricter admission criteria in the release (higher bar from SIG Release, SIG Testing, PRR, WG Reliability, SIG Scalability, etc.)
>
> Of course some of these mitigations might make (2) worse. Other ideas?
>
> For (2), we may want to thinking about how we do our feature graduation process. Having three stages to go through and one less release per year to do it will stretch out how long it takes to add a feature quite substantially. This is a big topic we wouldn't want to gate on, but we may want to have a plan for before moving forward. Some options others have mentioned to me:
> a) Features are in, or out. That is, straight to GA, but with features not admitted to the main build until they are ready. This means we need some alternate build and perhaps dev or feature branches.
> b) Two stages instead of three. I am very skeptical of this, but there is some support for this in how K8s is actually used today. We did an operator survey in the PRR subproject and we found that:
>
>     More than 90% of surveyed orgs allow (by policy) all Beta and GA features in prod.
>     Less than 10% of surveyed orgs have ever disabled a beta feature in prod. The caveat is that both operators with more than 10,000 nodes under management that answered the survey have done this.
>
> This would indicate that people already treat beat as GA, for the most part. That's not necessarily a good thing, but it is a fact. Of course, the big benefit for us as contributors is that betas can be changed if we have made a mistake. So again we probably wouldn't want to use the current bar for beta and just map it to GA. We would need to raise that quite a bit.
>
> With respect to alpha, the idea there is to gather feedback. I think we get some limited feedback with it, but is it enough? If alpha and beta are not really serving their intended purpose, are they really that useful, as currently defined?
>
> Another option for (2) is breaking up the monolith more and allowing components to release independently. However, this could make test coverage nearly impossible, as individual components would need to be tested in various versions. Given that K8s is operated independently by thousands of organizations, I don't think we can treat the core components as completely independent. Nonetheless there may be some opportunities for decomposition (like we did with CoreDNS, for example). @thockin mentioned kubelet and kube-proxy in this regard.
>
> See also (there are probably many more issues like these):
>
>     kubernetes/community#567
>     kubernetes/community#4000

@jberkus:

>     For (2), we may want to thinking about how we do our feature graduation process. Having three stages to go through and one less release per year to do it will stretch out how long it takes to add a feature quite substantially.
>
> Does it really, though? How many features actually went from alpha to GA in 9 months?
>
> Does anyone have hard data on this?

@johnbelamaric:

>     Does it really, though? How many features actually went from alpha to GA in 9 months?
> 
>     Does anyone have hard data on this?
>
> Ok, that's a fair point to challenge that assumption. I agree probably most don't do it in 9 months, but data would help. I am not sure if "months" is the right measure, though. My concern is that people will still take the same number of releases to make it happen, which means it will take longer.
>
> On a similar note, I am curious if there is any data on the amount of feedback alpha features actually get.

@jberkus:

> Yah, and "Is there some outstanding blocker that prevents new features from actually going from alpha to GA in 3 releases for any reason other than maturity?"
>
> That is, if a feature can go from alpha to GA in 3 releases, that's fine. That's a year, and do we really want to make the case that it should take less than a year to get a new feature to GA? BUT ... if something in our process means that features realistically can't be introduced in 1.23 and go to beta in 1.24, then we have a potential problem, because that timeline gets very long if it's gonna actually take you 5 releases.

@johnbelamaric:

> The two points I bring up are in tension with each other. That is, the same
features spread across fewer annual releases automatically means more per
release, or longer duration in the pipeline.

@bowei:

> We should make sure that there are sufficient improvements/metrics/goals to meet w/ any change (or no change).
> It wouldn't be great if 4 -> 3 didn't improve things and the same rationales would justify 3 -> 2.
> Is there a bar where we can comfortably go back from 3 -> 4?

@onlydole:

> +1 for three releases a year, and all of this discussion is fantastic!
>
> I agree with there being three releases a year. However, I do think that having more regular minor version releases would be helpful, so there isn’t any rush to get things into a specific release, nor a blocker around shipping bugfixes or improvements.
>
> I’d like to propose a strongly scoped path for Alpha, Beta, and GA features. I believe that allowing for a bit more leniency for Alpha and Beta code promotion and more stringent requirements for features before they make GA status.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include 
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### FIXME

#### LTS

@kellycampbell:

> +1 One thing I'm wondering is how this would affect vendors and other downstream integrators. For example, we build our clusters using kops. It is normally at least one release behind the k8s releases. Would having an extra month help them sync up with the release? I imagine other integrators and cloud providers would also benefit from extra time to update.
>
> Additional point: we really are only able to upgrade our clusters about twice a year for various reasons not related to the k8s or kops release schedule. I see the maturing k8s as similar to OS upgrades such as Ubuntu which releases twice a year and have LTS every 4 releases or so. They are able to patch incrementally and continuously though. If k8s had a similar ability to apply incremental patches in a standard way such that 1.19.1 -> 1.19.2 is more or less automatic and not up to each vendor, that would be amazing.

@chris-short:

> I'm in favor of three releases a year. I like @jberkus comment about a Q4 maintenance release too without so much hoopla and fanfare.
>
> I hope folks aren't driven to only fix stuff in Q4, "Oh that's a gnarly one, wait until the end of the year." Is something I could foresee someone thinking at some point, if we don't word things right about a maintenance release.
>
> One question I think has been lightly touched on is, "What about LTS releases?" (and I know this is out of scope but, I don't know where we stand on this atm)

@youngnick:

> The consensus on LTS (meaning multi-year support for a single version) is, in short, there's no consensus. We in the LTS WG worked for over two years, and we were able to get everyone to agree to extend the support window to one year (from nine months), which I think speaks to the passion that everyone has about this, the diversity of the use cases Kubernetes is supporting, and the community's determination to get it right.
>
> Speaking personally, I think that LTS is a long way away, if ever - it would require a lot more stability in the core than we have right now. With efforts like all the work to pull things out of tree, and the general movement towards adding new APIs outside of the core API group, I think it's plausible that one day, we may get to a place where we could consider it, but I don't think it's likely for some time, if ever. @tpepper, @jberkus, @LiGgit, and @dims among others may have thoughts here. :)

@jberkus:

> @youngnick you summed it up. Having 2 years of support for a specific API snapshot is unrealistic right now for all sorts of reasons, and it wasn't even clear that it was what people actually wanted.

#### Go faster

@sebgoa:

> Mostly a peanut gallery comment.
>
> The kubernetes releases have been a strong point of the software since its inception. The quality, testing and general care has been amazing and only improved (my point of reference is releases of some apache foundation software). With the increased usage, scrutiny and complexity of the software it feels like each release is a huge effort for the release team so naturally less releases could mean a bit less work.
>
> Users and even cloud providers seem to struggle to keep up with the releases (e.g 1.18 is not yet available on GKE for instance), so this also seems to indicate that less releases would ease the work of users and providers.
>
> But, generally speaking less releases (or less frequent minor releases) will also mean that each release will pack more weight, which means it will need even more testing and it will make upgrades tougher.
>
> With less releases developers will tend to rush their features at the last minute to "get it in" because the next one will be further apart.
>
> IMHO with more releases, developers don't need to rush their features, upgrades a more bite size and it necessarily pushes for even more automation.
>
> So at the risk of being down voted I would argue that we have worked over the last 15 years to agree that "release early, release often" was a good idea, that creating a tight feedback loop with devs, testers and users was a very good idea.
>
> Theoretically we should have processes in place to be able to automatically upgrade and be able to handle even a higher cadence of releases. I could see a future were people don't upgrade that often because there are less releases and then start to fall behind one year, then two...etc.
>
> PS: I understand this is almost a theoretical argument and that each release is a ton of work, I also know I am not helping the release team and I know 2020 is a very tough year.

@johnbelamaric:

>     I could see a future were people don't upgrade that often because there are less releases and then start to fall behind one year, then two...etc.
>
> Yes, this is a big fear of mine as well. We have worked hard to prevent vendor-based fragmentation (e.g., with conformance) and version-based fragmentation (with API round trip policies, etc). Bigger releases with riskier upgrades may undermine that work. We must avoid a Python2 -> 3 situation. This is also why we elected for a longer support cycle as opposed to an LTS. With the extensive ecosystem we have, fragmentation is extremely dangerous.
>
> I don't think going from 4->3 releases will create this problem, though I do think going to 2 or 1 release would. We need some plan around the mitigations I described earlier though, to ensure we avoid this fate.

#### No

@adrianotto:

> -1
>
> I acknowledge this proposed change will not slow the rate of change, but it does concentrate risk. It means that each release would carry more change, and more risk. It also means that adoption of those features will be slower, and that's bad for users.
>
> Release early and release often. This philosophy is a key reason k8s matured as quickly as it did. I accept that 2020 is a strange year, and should be handled as such. That is not a valid reason to change what is done in subsequent years. Each time you make a change like this, it has a range of unintended consequences, such as the risk packing I mentioned above. It would be tragic to slow overall slowdown in the promotion of GA features because they transition based on releases, not duration in use. If the release process is burdensome, we should be asking how we can apply our creativity to make it easier, and reducing the release frequency might be one of several options. But asking the question this way constrains us from looking at the bigger picture, and fully considering what will serve the community best.

@bowei:

> Echoing Adrian's comment:
>
> I think releases are a nice forcing function towards stabilization and having less releases will increase drift in the extra time.
> Are we coupling feature(s) stabilization to release cadence too much?
> One fear is that the work simply going to be pushed rather than decrease, but now there are fewer "stabilization" points in the year.

@spiffxp:

> I'm a net -1 on 3 releases per year, but I understand I'm in the minority. Reducing the frequency of a risky/painful process does not naturally lead to a net reduction of pain or risk, and usually incentivizes increased risk. "Stabilize the patient" can be a good first step, but is insufficient on its own.
>
> To @tpepper's question of implementation, if we go with 3 symmetric releases, I would suggest using the "extra time" as a tech debt / process debt paydown phase at the beginning of each release cycle. Somewhat like how we left the milestone restriction in place at the beginning of the 1.20 release cycle. This would provide opportunity to pay down tech debt / process debt that involves large refactoring or breaking changes, the sort of work that is actively discouraged during the code freeze leading up to a release.
>
> I may have too narrow a view, but I have concerns that an April / August / December cadence puts undue pressure to land in August. I'm thinking of industries that typically go through a seasonal freeze in Q4. Shifting forward by a month (January / May / September) or two (February, June, October) may relieve some of that pressure, though it does cause one release to straddle Q4/Q1 in an awkward way.
>
> Another option is to declare Q4 what it has been in practice, a quieter time during which we're not actually going to push hard on a release, but I don't think that works as well with 3 releases vs. 4.

#### Maintenance releases

@youngnick:

> I agree with @spiffxp that whatever we end up doing, we should acknowledge that calendar Q4 is substantially quieter than other quarters, with US Kubecon rolling into US Thanksgiving, rolling into the December festive season.
>
> I think that any plan to change the release cadence needs to take that as a prime consideration, whether it's keeping four releases a year and marking the Q4 one as minimal features, spreading three releases across the year, or some other solution.

@jberkus:

> @spiffxp we've talked about making Q4 a "maintenance" release endlessly, but we've never actually implemented that.

@jayunit100:

> Sounds like joshs comment is middle ground on the way to three : sure you get 4 releases but the fourth is only bug fixes, tests and stability .

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

## FIXME Cleanup

### How do we make a decision?

#### Canonical

Write a KEP

- Seek approval from:
  - SIG Release
  - SIG Architecture
  - SIG Testing
  - (maybe) Steering
- Set a lazy consensus timeout

#### Alternatives

- A survey
  - What would this need to contain to be effective?
- Vote to stakeholders (SIG leads + Steering)
  - Is there precedence for this outside of elections?
  - Should this include subproject owners?
  - 1 (Strong disagree) - 5 (Strong agree)

### Do we have any data?

Primarily anecdotal from SIG Release members, vendors, and end users.

AI:

- (to Elana) What kind of data specifically are we looking for?
  - Who's the audience? End users or principals?
  - Should we just do this all of the time post-release?
- (to Josh) What did we discover regarding feature trajectory?

Thoughts:
I would want requesters to be very explicit about the kind of data we're interested in.
SIG Release and others can work on collection, but we need to make sure this isn't a continually moving target.

We're also starting from a disadvantage trying to compare our status quo to something we haven't tried for a sustained period of time.

### How do we implement?

TBD
AI: Expand

Thoughts:
I feel like less is going to change in the process than people think.

- Make the decision
- Set the schedule

### Conversations

#### [Leads meeting](https://docs.google.com/document/d/1Jio9rEtYxlBbntF8mRGmj6Q1JAdzZ9fTDo3ru1HK_LI/edit#bookmark=id.val5alfdahlr) feedback

- Q: how are we making a decision?
- [comment]: 1.21 is the real EOY release as its scheduler covers december
- Do we have any data?
- [comment]: does sig-arch, sig-release, steering, etc own the final decision?
- [comment]: sep question of how we implement; does that mandate a fixed freq?
  - Stick to schedule and just not cut a 4th release?
- [comment]: releases don’t necessarily have to be equally spaced
- [comment]: cadence doesn't feel like things get the attention they deserve, things always feel rushed. Not a lot of space for people to take a step back.
- [comment]: should upgrade testing improve? Be blocking?

#### From Jeremy

John B:

> Ask in the meeting, how are we going to make the actual decision for 3 vs 4?
> are we going to vote? or will SIG Release just make the decision?

Elana:

> additional ask: can we send out a real survey to end users

Daniel:

> the concern about things "taking longer" to go stable because of # of releases in beta also came up again, can we think of a way to handle this?
> Do the three releases need to be evenly spaced?

Aaron C:

> Can we get more "implementation" details about how three releases would word?
> Can we make upgrade jobs / tests blocking to make the upgrade between versions better
