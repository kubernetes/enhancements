<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
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
# KEP-4604: Tune CrashLoopBackoff

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Existing backoff curve change: front loaded decay](#existing-backoff-curve-change-front-loaded-decay)
  - [User Stories](#user-stories)
    - [Task isolation](#task-isolation)
    - [Fast restart on failure](#fast-restart-on-failure)
    - [Sidecar containers fast restart](#sidecar-containers-fast-restart)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Front loaded decay curve methodology](#front-loaded-decay-curve-methodology)
  - [Rapid curve methodology](#rapid-curve-methodology)
  - [Kubelet overhead analysis](#kubelet-overhead-analysis)
  - [Benchmarking](#benchmarking)
  - [Relationship with Job API podFailurePolicy and backoffLimit](#relationship-with-job-api-podfailurepolicy-and-backofflimit)
  - [Relationship with ImagePullBackOff](#relationship-with-imagepullbackoff)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Global override](#global-override)
  - [Per exit code configuration](#per-exit-code-configuration)
  - [<code>RestartPolicy: Rapid</code>](#restartpolicy-rapid)
  - [Flat-rate restarts for <code>Succeeded</code> Pods](#flat-rate-restarts-for-succeeded-pods)
    - [On Success and the 10 minute recovery threshold](#on-success-and-the-10-minute-recovery-threshold)
      - [Related: API opt-in for flat rate/quick restarts when transitioning from <code>Succeeded</code> phase](#related-api-opt-in-for-flat-ratequick-restarts-when-transitioning-from-succeeded-phase)
      - [Related: <code>Succeeded</code> vs <code>Rapid</code>ly failing: who's getting the better deal?](#related-succeeded-vs-rapidly-failing-whos-getting-the-better-deal)
  - [Front loaded decay with interval](#front-loaded-decay-with-interval)
  - [Late recovery](#late-recovery)
  - [More complex heuristics](#more-complex-heuristics)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

CrashLoopBackoff is designed to slow the speed at which failing containers
restart, preventing the starvation of kubelet by a misbehaving container.
Currently it is a subjectively conservative, fixed behavior regardless of
container failure type: when a Pod has a restart policy besides `Never`, after
containers in a Pod exit, the kubelet restarts them with an exponential back-off
delay (10s, 20s, 40s, …), that is capped at five minutes. The delay for restarts
will stay at 5 minutes until a container has executed for 10 minutes without any
problems, in which case the kubelet resets the restart backoff timer for that
container and further crash loops start again at the beginning of the delay
curve
([ref](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/)). 

Both the decay to 5 minute back-off delay, and the 10 minute recovery threshold,
are considered too conservative, especially in cases where the exit code was 0
(Success) and the pod is transitioned into a "Completed" state or the expected
length of the pod run is less than 10 minutes.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

[Kubernetes#57291](https://github.com/kubernetes/kubernetes/issues/57291), with
over 250 positive reactions and in the top five upvoted issues in k/k, covers a
range of suggestions to change the rate of decay for the backoff delay or the
criteria to restart the backoff counter, in some cases requesting to make this
behavior tunable per node, per container, and/or per exit code. Anecdotally,
there are use cases representative of some of Kubernetes' most rapidly growing
workload types like
[gaming](https://github.com/googleforgames/agones/issues/2781) and
[AI/ML](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/3329-retriable-and-non-retriable-failures)
that would benefit from this behavior being different for varying types of
containers. Application-based workarounds using init containers or startup
wrapper scripts, or custom operators like
[kube_remediator](https://github.com/ankilosaurus/kube_remediator) and
[descheduler](https://github.com/kubernetes-sigs/descheduler) are used by the
community to anticipate crashloopbackoff behavior, prune pods with nonzero
backoff counters, or otherwise "pretend" the pod did not exit recently to force
a faster restart from kubelet. Discussions with early Kubernetes contributors
indicate that the current behavior was not designed beyond the motivation to
throttle misbehaving containers, and is open to reintepretation in light of user
experiences, empirical evidence, and emerging workloads.

By definition this KEP will cause pods to restart faster and more often than the
current status quo; let it be known that such a change is desired. It is also
the intention of the author that, to some degree, this change happens without
the need to reconfigure workloads or expose extensive API surfaces, as
experience shows that makes changes difficult to adopt, increases the risk for
misconfiguration, and can make the system overly complex to reason about. That
being said, this KEP recognizes the need of backoff times to protect node
stability, avoiding too big of sweeping changes to backoff behavior, or setting
unreasonable or unpredictable defaults. Therefore the approach manages the risk
of different decay curves to node stability by providing an API surface to
opt-in to the riskiest option, and ultimately also provides the observability
instrumentation needed by cluster operators to model the impact of this change
on their existing deployments.

A large number of alternatives have been discussed over the 5+ years the
canonical tracking issue has been open, some of which imply high levels of
sophistication for kubelet to make different decisions based on system state,
workload specific factors, or by detecting anomalous workload behavior. While
this KEP does not rule out those directions in the future, the proposal herein
focuses on the simpler, easily modeled changes that are designed to address the
most common issues observed today. It is the desire of the author that the
observability and benchmarking improvements instrumented as part of this
proposal can serve as a basis for pursuing those types of improvements in the
future, including as signals for end users to represent such solutions as
necessary. Some analysis of these alternatives are included in the Alternatives
Considered section below for future reference.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

* Improve pod restart backoff logic to better match the actual load it creates
  and meet emerging use cases
* Quantify and expose risks to node stability resulting from decreased and/or
  hetereogeneous backoff configurations
* Empirically derive defaults and allowable thresholds that approximate current
  node stability
* Provide a simple UX that does not require changes for the majority of
  workloads
* <<[UNRESOLVED]>> Must work for Jobs and sidecar containers <<[/UNRESOLVED]>>

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

* This effort is NOT intending to support fully user-specified configuration, to
  cap risk to node stability
* This effort is purposefully NOT implementing more complex heuristics by
  kubelet (e.g. system state, workload specific factors, or by detecting
anomalous workload behavior) to focus on better benchmarking and observability
  and address common use cases with easily modelled changes first
* This effort is NOT changing the independent backoff curve for image pulls

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

This design seeks to incorporate a two-pronged approach:

1. Change the existing initial value for the backoff curve to stack more retries
   earlier for all restarts (`restartPolicy: OnFailure` and `restartPolicy:
   Always`) <<[UNRESOLVED]>>
2. Provide a option to configure even faster restarts for specific Pod/Container
(particularly sidecar containers)/Node/Cluster, regardless of exit code,
reducing the max wait to 1 minute <<[/UNRESOLVED]>>

In addition, part of the alpha period will be dedicated entirely to
   systematically stress testing kubelet and API Server with different
   distributions of workloads utilizing the new backoff curves. Therefore, this
KEP requires instrumentation of enhanced visibility into pod restart behavior to
enable Kubernetes benchmarking during the alpha phase. During the benchmarking
period of alpha, kubelet memory and CPU, API server latency, and pod restart
latency will be observed and analyzed to define the maximum allowable restart
rate for fully saturated nodes.

Longer term, these
metrics will also supply cluster operators the data necessary to better analyze
and anticipate the change in load and node stability as a result of upgrading to
these changes.

Note that proposal will NOT change:
* backoff behavior for Pods transitioning from the "Success" state -- see [here
  in Alternatives Considered](#on-success-and-the-10-minute-recovery-threshold)
* the time Kubernetes waits before resetting the backoff counter -- see the
  [here inAlternatives
  Considered](#on-success-and-the-10-minute-recovery-threshold)
* the ImagePullBackoff -- out of scope, see [Design
  Details](#relationship-with-imagepullbackoff)
* changes that address 'late recovery', or modifications to backoff behavior
  once the max cap has been reached -- see
  [Alternatives](#more-complex-heuristics)


### Existing backoff curve change: front loaded decay

This KEP proposes changing the existing backoff curve to load more restarts
earlier by changing the initial value of the exponential backoff. A number of
alternate initial values are modelled below, until the 5 minute cap would be
reached. This proposal suggests we start with a new initial value of 1s, and
analyze its impact on infrastructure during alpha.

![](todayvs1sbackoff.png)


### User Stories

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Task isolation
By design the container wants to exit so it will be recreated and restarted to
start a new “session” or “task”. It may be a new gaming session or a new
isolated task processing an item from the queue.

This is not possible to do by creating the wrapper of a container that will
restart the process in a loop because those tasks or gaming sessions desire to
start from scratch. In some cases - they may even want to download a new
container image.

For these cases, it is important that the startup latency of the full
rescheduling of a new pod is avoided.

#### Fast restart on failure

There are AI/ML scenarios where an entire workload is delayed if one of the Pods
is not running. Some Pods in this scenario may fail with the recoverable error -
for example some dependency failure or be killed by infrastructure (e.g. exec
probe failures on containerd slow restart). In such cases, it is desirable to
restart the container as fast as possible and not wait for 5 minutes of the max
crashloop backoff timeout. With the existing max crashloop backoff timeout, a
failure of a single pod in a highly-coupled workload can cause a cascade in the
workload leading to an overall delay of (much) greater than the 5 minute
backoff.

The typical pattern here today is to be quick to restart container, but with the
intermittently failed dependency (e.g. network is down for some time, some
intermittent issue with a GPU), this causes the container to fail repeatedly and
the backoff timeout to eventually be reached. However when the dependency goes
back to green, the container is not restarted immediately and has already
reached the maximum crash loop backoff duration of 5 minutes.

#### Sidecar containers fast restart

There are cases when the Pod consists of a user container implementing business
logic and a sidecar providing networking access (Istio), logging (opentelemetry
collector), or orchestration features. In some cases sidecar containers are
critical for the user container functioning as they provide a basic
infrastructure for the user container to run in. In such cases it is beneficial
to not apply the exponential backoff to the sidecar container restarts and keep
it at a low constant value.

This is especially true for cases when the sidecar is killed by infrastructure
(e.g. OOMKill) as it may have happened for reasons independent from the sidecar
functionality.


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

The biggest risk of this proposal is reducing the decay of the _default_
CrashLoopBackoff: to do so too severely compromises node stability, risking the
kubelet component to become too slow to respond and the pod lifecycle to
increase in latency, or worse, causing entire nodes to crash if kubelet takes up
too much CPU or memory. Since each phase transition for a Pod also has an
accompanying API request, if the requests become rapid enough due to fast enough
churn of Pods through CrashLoopBackoff phases, the central API server could
become unresponsive, effectively taking down an entire cluster.

The same risk exists for the <<[UNRESOLVED]>> per Node <<[/UNRESOLVED]>>
feature, which, while not default, is by design a more severe reduction in the
decay behavior. It can abused by <<[UNRESOLVED]>> cluster operators
<<[/UNRESOLVED]>>, and in the worst case cause nodes to fully saturate with
<<[UNRESOLVED]>> instantly <<[/UNRESOLVED]>> restarting pods that will never
recover, risking similar issues as above: taking down nodes or at least
nontrivially slowing kubelet, or increasing the API requests to store backoff
state so significantly that the central API server is unresponsive and the
cluster fails.

During alpha, naturally the first line of defense is that the enhancements, even
the reduced "default" baseline curve for CrashLoopBackoff, are not usable by
default and must be opted into. In this specific case they are opted into
separately as kubelet flags, so clusters will only be affected by each risk if
the cluster operator enables the new features during the alpha period.

Beyond this, there are two main mitigations during alpha: conservativism in
changes to the default behavior, and <<[UNRESOLVED]>> per-node <<[/UNRESOLVED]>>
opt-in and redeployment required for the more aggressive behavior.

The alpha changes to the default backoff curve were chosen because they are
minimal -- <<[ UNRESOLVED]>>the proposal maintains the existing rate and max
cap, and reduces the initial value to the point that only introduces 3 excess
restarts per pod, the first 2 excess in the first 10 seconds and the last excess
following in the next 30 seconds (see [Design
Details](#front-loaded-decay-curve-methodology)). For a hypothetical node with
the max 110 pods all stuck in a simultaneous CrashLoopBackoff, API requests to
change the state transition would increase at its fastest period from ~110
requests/10s to 330 requests/10s. <<[/UNRESOLVED]>> By passing this minimal
change through the existing SIG-scalability tests, while pursuing manual and
more detailed periodic benchmarking during the alpha period, we can increase the
confidence in this change and in the possibility of reducing further in the
future.

For the <<[UNRESOLVED]>> per node <<[/UNRESOLVED]>> case, because the change is
more significant, including lowering the max cap, there is more risk to node
stability expected. This change is of interest to be tested in the alpha period
by end users, and is why it is still included with opt-in even though the risks
are higher. That being said it is still a relatively conservative change in an
effort to minimize the unknown changes for fast feedback during alpha, while
improved benchmarking and testing occurs. <<[UNRESOLVED update with real stress
test results]>> For a hypothetical node with the max 110 pods all stuck in a
simultaneous `Rapid` CrashLoopBackoff, API requests to change the state
transition would increase from ~110 requests/10s to 440 requests/10s, and since
the max cap would be lowered, would exhibit up to 440 requests in excess every
300s (5 minutes), or an extra 1.4 requests per second once all pods reached
their max cap backoff. It also should be noted that due to the specifics of the
configuration required in the Pod manifest, being against an immutable field,
will require the Pods in question to be redeployed. This means it is unlikely
that all Pods will be in a simultaneous CrashLoopBackoff even if they are
designed to quickly crash, since they will all need to be redeployed and
rescheduled. <<[/UNRESOLVED]>>

## Design Details 

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Front loaded decay curve methodology
As mentioned above, today the standard backoff curve is a 2x exponential decay
starting at 10s and capping at 5 minutes, resulting in a composite of the
standard hockey-stick exponential decay graph followed by a linear rise until
the heat death of the universe as depicted below:

![A graph showing the backoff decay for a Kubernetes pod in
CrashLoopBackoff](./crashloopbackoff-succeedingcontainer.png "CrashLoopBackoff
decay")

Remember that the backoff counter is reset if containers run longer than 10
minutes, so in the worst case where a container always exits after 9:59:59, this
means in the first 30 minute period, the container will restart twice. In a more
easily digestible example used in models below, for a fast exiting container
crashing every 10 seconds, in the first 30 minutes the container will restart
about 10 times, with the first four restarts in the first 5 minutes.

Why change the initial value of the backoff curve instead of its rate, or why
not change the decay function entirely to other well known equations (like
functions resulting in curves that are lienar, parabolic, sinusoidal, etc)?

Exponential decay, particularly at a rate of 2x, is commonly used for software
retry backoff as it has the nice properties of starting restarts at a low value,
but penalizing repeated crashes harshly, to protect primarily against
unrecoverable failures. In contrast, we can interpret linear curves as
penalizing every failure the same, or parabolic and sinusoidal curves as giving
our software a "second chance" and forgiving later failures more. For a default
restart decay curve, where the cause of the restart cannot be known, 2x
exponential decay still models the desired properties more, as the biggest risk
is unrecoverable failures causing "runaway" containers to overload kubelet.

To determine the effect in abstract of changing the initial value on current
behavior, we modeled the change in the starting value of the decay from 10s to
1s, 250ms, or even 25ms. 

!["A graph showing the decay curves for different initial values"](differentinitialvalues.png
"Alternate CrashLoopBackoff initial values")

For today's decay rate, the first restart is within the
first 10s, the second within the first 30s, the third within the first 70s.
Using those same time windows to compare alternate initial values, for example
changing the initial rate to 1s, we would instead have 3 restarts in the first
time window, 1  restart within the time window, and two more restarts within the
third time window. As seen below, this type of change gives us more restarts
earlier, but even at 250ms or 25ms initial values, each approach a similar rate
of restarts after the third time window.

![A graph showing different exponential backoff decays for initial values of
10s, 1s, 250ms and 25ms](initialvaluesandnumberofrestarts.png "Changes to decay
with different initial values")

Among these modeled initial values, we would get between 3-7 excess restarts per
backoff lifetime, mostly within the first three time windows matching today's
restart behavior.

<<[UNRESOLVED include stress test data now]>> <<[/UNRESOLVED]>>

### Rapid curve methodology

For some users in
[Kubernetes#57291](https://github.com/kubernetes/kubernetes/issues/57291), any
delay over 1 minute at any point is just too slow, even if it is legitimately
crashing. A common refrain is that for independently recoverable errors,
especially system infrastructure events or recovered external dependencies, or
for absolutely nonnegotiably critical sidecar pods, users would rather poll more
often or more intelligently to reduce the amount of time a workload has to wait
to try again after a failure. In the extreme cases, users want to be able to
configure (by container, node, or exit code) the backoff to close to 0 seconds.
This KEP considers it out of scope to implement fully user-customizable
behavior, and too risky without full and complete benchmarking to node stability
to allow legitimately crashing workloads to have a backoff of 0, but it is in
scope for the first alpha to provide users a way to <<[UNRESOLVED]>> opt nodes
in <<[/UNRESOLVED]>> to a even faster restart behavior.

The finalization of the initial and max cap can only be done after benchmarking.
But as a conservative first estimate for alpha in line with maximums discussed
on [Kubernetes#57291](https://github.com/kubernetes/kubernetes/issues/57291),
the initial curve is selected at <<[UNRESOLVED]>>initial=250ms / cap=1 minute,
<<[/UNRESOLVED]>> but during benchmarking this will be modelled against kubelet
capacity, potentially targeting something closer to an initial value near 0s,
and a cap of 10-30s. To further restrict the blast radius of this change before
full and complete benchmarking is worked up, this is gated by a separate alpha
feature gate and is opted in to per <<[UNRESOLVED]>> node <<[/UNRESOLVED]>>.

### Kubelet overhead analysis

As it's likely that in some deployments pods will restart more often that in
current Kubernetes, for this KEP, it's important to understand what the kubelet
does during pod restarts. 

* Potentially re-downloads the image (utilizing network + IO and blocks other
  image downloads)
* Clears up old container using Containerd
* Redownloads secrets and configs if needed
* Recreates the container using Containerd
* Runs startup probes until container started (startup probes may be more
  expensive than the readiness probes as they often configured to run more
  frequently)
* Application runs thru initialization logic (typically utilizing more IO)
* Logs information about all those container operations (utilizing disk IO and
  “spamming” logs)

```
 <<[UNRESOLVED add the rest of the analysis since 1.31 and answer these questions from original PR]>> 
 > What conditions lead to a re-download of an image? I wonder if we can eliminate this, or if that's too much of a behavior change.
 > Similar question for image downloads. Although in this case, I think the kubelet should have an informer for any secrets or configmaps used, so it should just pull from cache. Is that true for EnvVarFrom values?
 >Does this [old container cleanup using containerd] include cleaning up the image filesystem? There might be room for some optimization here, if we can reuse the RO layers.
 
  <<[/UNRESOLVED]>>
```

```
 <<[UNRESOLVED>>
It's because of the way we handle backoff:
https://github.com/kubernetes/kubernetes/blob/a7ca13ea29ba5b3c91fd293cdbaec8fb5b30cee2/pkg/kubelet/kuberuntime/kuberuntime_manager.go#L1336-L1349

So the first time the container exits, there is no backoff delay recorded, but then it adds a backoff key at line 1348.

So the actual (current) backoff implementation is:

0 seconds delay for first restart
10 seconds for second restart
10 * 2^(restart_count - 2) for subsequent restarts
But those numbers are all delayed up to 10s due to kubernetes/kubernetes#123602
 <<[UNRESOLVED>>
```

### Benchmarking

Again, let it be known that by definition this KEP will cause pods to restart
faster and more often than the current status quo and such a change is desired.
However, to do so as safely as possible, it is required that during the alpha
period, we reevaluate the SLIs and SLOs and benchmarks related to this change and
expose clearly the methodology needed for cluster operators to be able to quantify the
risk posed to their specific clusters on upgrade.

To best reason about the changes in this KEP, we requires the ability to
determine, for a given percentage of heterogenity between "Succeeded"
terminating pods, and crashing pods whose `restartPolicy: Always`:
 * what is the load and rate of Pod restart related API requests to the API
   server?
 * what are the performance (memory, CPU, and pod start latency) effects on the
   kubelet component? Considering the effects of different plugins (e.g. CSI,
   CNI)

Today there are alpha SLIs in Kubernetes that can observe that impact in
aggregate:
* Kubelet component CPU and memory
* `kubelet_http_inflight_requests`
* `kubelet_http_requests_duration_seconds`
* `kubelet_http_requests_total`
* `kubelet_pod_worker_duration_seconds`
* `kubelet_runtime_operations_duration_seconds`
* `kubelet_pod_start_duration_seconds`
* `kubelet_pod_start_sli_duration_seconds`

In addition, estimates given the currently suggested changes in API requests are
included in [Risks and Mitigations](#risks-and-mitigations) and were deciding
factors in specific changes to the backoff curves. Since the changes in this
proposal are deterministic, this is pre-calculatable for a given heterogenity of
quantity and rate of restarting pods.

In addition, the `kube-state-metrics`, project already implements
restart-specific metadata for metrics that can be used to observe pod restart
latency in more detail, including:
* `kube_pod_container_status_restarts_total`
* `kube_pod_restart_policy`
* `kube_pod_start_time`
* `kube_pod_created`

During the alpha period, these metrics, the SIG-Scalability benchmarking tests,
added kubelet performance tests, and manual benchmarking by the author against
`kube-state-metrics` will be used to answer the above questions, tying together the
container restart policy (inherited or declared), the terminal state of a
container before restarting, and the number of container restarts, to articulate
the rate and load of restart related API requests and the performance effects on
kubelet.

<<[UNRESOLVED add initial benchmarking test data]>> <<[/UNRESOLVED]>>

### Relationship with Job API podFailurePolicy and backoffLimit

Job API provides its own API surface for describing alterntive restart
behaviors, from [KEP-3329: Retriable and non-retriable Pod failures for
Jobs](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/3329-retriable-and-non-retriable-failures),
in beta as of Kubernetes 1.30. The following example from that KEP shows the new
configuration options: `backoffLimit`, which controls for number of retries on
failure, and `podFailurePolicy`, which controls for types of workload exit codes
or kube system events to ignore against that `backoffLimit`.

```yaml
apiVersion: v1
kind: Job
spec:
  [ . . . ]
  backoffLimit: 3
  podFailurePolicy:
    rules:
    - action: FailJob
      onExitCodes:
        containerName: main-job-container
        operator: In
        values: [1,2,3]
    - action: Ignore
      onPodConditions:
      - type: DisruptionTarget
```

The implementation of KEP-3329 is entirely in the Job controller, and the
restarts are not handled by kubelet at all; in fact, use of this API is only
available if the `restartPolicy` is set to `Never` (though
[kubernetes#125677](https://github.com/kubernetes/kubernetes/issues/125677)
wants to relax this validation to allow it to be used with other `restartPolicy`
values). As a result, to expose the new backoff curve Jobs using this feature,
the updated backoff curve must also be implemented in the Job controller.

### Relationship with ImagePullBackOff

ImagePullBackoff is used, as the name suggests, only when a container needs to
pull a new image. If the iamge pull fails, a backoff decay is used to make later
retries on the image download wait longer and longer. This is configured
internally independently
([here](https://github.com/kubernetes/kubernetes/blob/release-1.30/pkg/kubelet/kubelet.go#L606))
from the backoff for container restarts
([here](https://github.com/kubernetes/kubernetes/blob/release-1.30/pkg/kubelet/kubelet.go#L855)).

This KEP considers changes to ImagePullBackoff as out of scope, so during
implementation this will keep the same backoff. This is both to reduce the
number of variables during the benchmarking period for the restart counter, and
because the problem space of ImagePullBackoff could likely be handled by a
completely different pattern, as unlike with CrashLoopBackoff the types of
errors with ImagePullBackoff are less variable and better interpretable by the
infrastructure as recovereable or non-recoverable (i.e. 404s).


### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

This feature requires two levels of testing: the regular enhancement testing
(described in the template below) and the stress/benchmark testing required to
increase confidence in ongoing node stability given heterogeneous backoff timers
and timeouts.

Some stress/benchmark testing will still be developed as part of this enhancement,
including the kubelet_perf tests indicated in the e2e section below.

Some of the benefit of pursuing this change in alpha is to also have the
opportunity to run against the existing SIG-Scalability performance and
benchmarking tests within an alpha candidate. In addition, manual benchmark
testing with GKE clusters can be performed by the author and evaluated as
candidates for formal, periodic benchmark testing in the Kubernetes testgrid.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

* Test coverage of proper requeue behavior; see
  https://github.com/kubernetes/kubernetes/issues/123602

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `kubelet/kuberuntime/kuberuntime_manager_test`: **could not find a successful
  coverage run on
  [prow](https://prow.k8s.io/view/gs/kubernetes-jenkins/logs/ci-kubernetes-coverage-unit/1800947623675301888)**

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- k8s.io/kubernetes/test/integration/kubelet:
  https://prow.k8s.io/view/gs/kubernetes-jenkins/logs/ci-kubernetes-integration-master/1800944856244162560
  * test with and without feature flags enabled

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- k8s.io/kubernetes/test/e2e/node/kubelet_perf: for a given percentage of
heterogenity between "Succeeded" terminating pods, and crashing pods whose
`restartPolicy: Always``, 
 * what is the load and rate of Pod restart related API requests to the API
   server?
 * what are the performance (memory, CPU, and pod start latency) effects on the
   kubelet component? Considering the effects of different plugins (e.g. CSI,
   CNI)

### Graduation Criteria

#### Alpha

- Changes to existing backoff curve implemented behind a feature flag 
    1. Front-loaded decay curve for all workloads with `restartPolicy: Always`
       <<[UNRESOLVED]>>
    2. Front-loaded, low max cap backoff curve for nodes workloads with XYZ
       config
- New XYZ kubelet config convertable to ABC on downgrade or without feature flag
<<[/UNRESOLVED]>>
- Metrics implemented to expose pod and container restart policy statistics,
  exit states, and runtimes
- Initial e2e tests completed and enabled
  * Feature flag on or off
  * <<[UNRESOLVED]>>node upgrade and downgrade path <<[/UNRESOLVED]>>
- Fix https://github.com/kubernetes/kubernetes/issues/123602 if this blocks the
  implementation, otherwise beta criteria
- Low confidence in the specific numbers/decay rate


#### Beta

- Gather feedback from developers and surveys
- High confidence in the specific numbers/decay rate
- Benchmark restart load methodology and analysis published and discussed with
  SIG-Node
- Additional e2e and benchmark tests, as identified during alpha period, are in
  Testgrid and linked in KEP

#### GA

- 2 Kubernetes releases soak in beta
- Completely finalize the decay curves and document them thoroughly
- Remove the feature flag code
- Confirm the exponential backoff decay curve related tests and code are still
  in use elsewhere and do not need to be removed
- Conformance test added for <<[UNRESOLVED]>> configured nodes <<[/UNRESOLVED]>>


<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
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

For `ReduceDefaultCrashLoopBackoffDecay`:

For an existing cluster, no changes are required to configuration, invocations
or API objects to make an upgrade.

To use the enhancement, the alpha feature gate is turned on. In the future when
(/if) the feature gate is removed, no configurations would be required to be
made, and the default behavior of the baseline backoff curve would -- by design
-- be changed.

For `EnableRapidCrashLoopBackoffDecay`:

For an existing cluster, no changes are required to configuration, invocations
or API objects to make an upgrade.

To make use of this enhancement, on upgrade, the feature gate must first be
turned on. Then, if any <<[UNRESOLVED]>>nodes want to use a different backoff
curve, their kubelet must be completely redeployed with XYZ kubelet config.
<<[/UNRESOLVED]>>

To stop use of this enhancement, there are two options. 

On a <<[UNRESOLVED]>> per-node basis, nodes can be completely redeployed with
XYZ kubelet config set to ABC. Since the Pods have been completely redeployed,
they will lose their prior backoff counter anyways and, if restarted, will start
from the beginning of their backoff curve (either the original one with initial
value 10s, or the new baseline with initial value 1s, depending on whether
they've turned on the `ReduceDefaultCrashLoopBackoffDecay` feature gate).
<<[/UNRESOLVED]>>

Or, the entire cluster can be restarted with the
`EnableRapidCrashLoopBackoffDecay` feature gate turned off. In this case, any
<<[UNRESOLVED]>>Node configured with a different backoff curve will instead use
the default backoff curve. Again, since the cluster was restarted and Pods were
redeployed, they will not maintain prior state and will start at the beginning
of their backoff curve.<<[/UNRESOLVED]>>

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

For the default backoff curve, no coordination must be done between the control
plane and the nodes; all behavior changes are local to the kubelet component and
its start up configuration.

For the  <<[UNRESOLVED]>> node case, since it is local to each kubelet and the
restart logic is within the responsibility of a node's local kubelet, no
coordination must b e done between the control plane and the nodes.
<<[/UNRESOLVED]>>

## Production Readiness Review Questionnaire

<<[UNRESOLVED removed when changing from 1.31 proposal to 1.32 proposal,
incoming in a separate PR]>> <<[/UNRESOLVED]>>

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

* 04-23-2024: Problem lead opted in by SIG-Node for 1.31 target
  ([enhancements#4603](https://github.com/kubernetes/enhancements/issues/4603))
* 06-04-2024: KEP proposed to SIG-Node focused on providing limited alpha
  changes to baseline backoff curve, addition of opt-in `Rapid` curve, and
  change to constant backoff for `Succeeded` Pods
* 06-06-2024: Removal of constant backoff for `Succeeded` Pods
* 09-09-2024: Removal of `RestartPolicy: Rapid` in proposal, removal of PRR, in
  order to merge a provisional and address the new 1.32 design in a cleaner PR

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

CrashLoopBackoff behavior has been stable and untouched for most of the
Kubernetes lifetime. It could be argued that it "isn't broken", that most people
are ok with it or have sufficient and architecturally well placed workarounds
using third party reaper processes or application code based solutions, and
changing it just invites high risk to the platform as a whole instead of
individual end user deployments. However, per the [Motivation](#motivation)
section, there are emerging workload use cases and a long history of a vocal
minority in favor of changes to this behavior, so trying to change it now is
timely. Obviously we could still decide not to graduate the change out of alpha
if the risks are determined to be too high or the feedback is not positive.

Though the issue is highly upvoted, on an analysis of the comments presented in
the canonical tracking issue
[Kubernetes#57291](https://github.com/kubernetes/kubernetes/issues/57291), 22
unique commenters were requesting a constant or instant backoff for `Succeeded`
Pods, 19 for earlier recovery tries, and 6 for better late recovery behavior;
the latter is arguably even more highly requested when also considering related
issue [Kubernetes#50375](https://github.com/kubernetes/kubernetes/issues/50375).
Though an early version of this KEP also addressed the `Success` case, in its
current version this KEP really only addresses the early recovery case, which by
our quantitative data is actually the least requested option. That being said,
other use cases described in [User Stories](#user-stories) that don't have
quantitative counts are also driving forces on why we should address the early
recovery cases now. On top of that, compared to the late recovery cases, early
recovery is more approachable and easily modelable and improving benchmarking
and insight can help us improve late recovery later on (see also the related
discussion in Alternatives [here](#more-complex-heuristics) and
[here](#late-recovery)).

CrashLoopBackoff behavior has been stable and untouched for most of the
Kubernetes lifetime. It could be argued that it "isn't broken", that most people
are ok with it or have sufficient and architecturally well placed workarounds
using third party reaper processes or application code based solutions, and
changing it just invites high risk to the platform as a whole instead of
individual end user deployments. However, per the [Motivation](#motivation)
section, there are emerging workload use cases and a long history of a vocal
minority in favor of changes to this behavior, so trying to change it now is
timely. Obviously we could still decide not to graduate the change out of alpha
if the risks are determined to be too high or the feedback is not positive.

Though the issue is highly upvoted, on an analysis of the comments presented in
the canonical tracking issue
[Kubernetes#57291](https://github.com/kubernetes/kubernetes/issues/57291), 22
unique commenters were requesting a constant or instant backoff for `Succeeded`
Pods, 19 for earlier recovery tries, and 6 for better late recovery behavior;
the latter is arguably even more highly requested when also considering related
issue [Kubernetes#50375](https://github.com/kubernetes/kubernetes/issues/50375).
Though an early version of this KEP also addressed the `Success` case, in its
current version this KEP really only addresses the early recovery case, which by
our quantitative data is actually the least requested option. That being said,
other use cases described in [User Stories](#user-stories) that don't have
quantitative counts are also driving forces on why we should address the early
recovery cases now. On top of that, compared to the late recovery cases, early
recovery is more approachable and easily modelable and improving benchmarking
and insight can help us improve late recovery later on (see also the related
discussion in Alternatives [here](#more-complex-heuristics) and
[here](#late-recovery)).

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Global override

Allow an override of the global constant of a maximum backoff period (or other
settings) in kubelet configuration.

### Per exit code configuration

One alternative is for new container spec values that allow individual containers to
respect overrides on the global timeout behavior depending on the exit reason.
These overrides will exist for the following reasons:

* image download failures
* workload crash: any non-0 exit code from the workload itself
* infrastructure events: terminated by the system, e.g. exec probe failures,
  OOMKill, or other kubelet runtime errors
* success: a 0 exit code from the workload itself

These had been selected because there are known use cases where changed restart
behavior would benefit workloads epxeriencing these categories of failures.

### `RestartPolicy: Rapid`

In the 1.31 version of this proposal, this KEP proposed a two-pronged approach
to revisiting the CrashLoopBackoff behaviors for common use cases:
1. modifying the standard backoff delay to start faster but decay to the same 5m
   threshold
2. allowing Pods to opt-in to an even faster backoff curve with a lower max cap

For step (2), the method to allow the Pods to opt-in was by a new enum value,
`Rapid`, for a Pod's `RestartPolicy`. In this case, Pods and restartable init
(aka sidecar) containers will be able to set a new OneOf value, `restartPolicy:
Rapid`, to opt in to an exponential backoff decay that starts at a lower initial
value and maximizes to a lower cap. This proposal suggests we start with a new
initial value of 250ms and cap of 1 minute, and analyze its impact on
infrastructure during alpha.

!["A graph showing today's decay curve against a curve with an initial value of
250ms and a cap of 1 minute for a workload failing every 10 s"](todayvsrapid.png
"rapid vs todays' CrashLoopBackoff")

### Flat-rate restarts for `Succeeded` Pods

We start from the assumption that the "Succeeded" phase of a Pod in Kubernetes
means that all workloads completed as expected. Most often this is colloquially
referred to as an exit code 0, as this exit code is what is caught by Kuerbenetes
for linux containers.

The wording of the public documentation
([ref](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-restarts))
and the naming of the `CrashLoopBackOff` state itself implies that it is a
remedy for a container not exiting as intended, but the restart delay decay
curve is applied to even successful pods if their `restartPolicy = Always`. On the
canonical tracking issue for this problem, a significant number of requests
focus on how an exponential decay curve is inappropriate for workloads
completing as expected, and unnecessarily delays healthy workloads from being
rescheduled.

This alternative would vastly simplify and cut down on the restart delays for
workloads completing as expected, as detectable by their transition through the
"Succeeded" phase in Kubernetes. The target is to get as close to the capacity
of kubelet to instantly restart as possible, anticipated to be somewhere within
0-10s flat rate + jitter delay for each restart, pending benchmarking in alpha.

Fundamentally, this change is taking a stand that a successful exit of a
workload is intentional by the end user -- and by extension, if it has been
configured with `restartPolicy = Always`, that its impact on the Kubernetes
infrastructure when restarting is by end user design. This is in contrast to the
prevailing Kubernetes assumption that that on its own, the Pod API best models
long-running containers that rarely or never exit themselves with "Success";
features like autoscaling, rolling updates, and enhanced workload types like
StatefulSets assume this, while other workload types like those implemented with
the Job and CronJob API better model workloads that do exit themselves, running
until Success or at predictable intervals. If this alternative was pursued, we
would instead interpret an end user's choice to run a relatively fast exiting
Pod (under 10 minutes) with both a successful exit code and configured to
`restartPolicy: Always`, as their intention to restart the pod indefinitely
without penalty.

!["A graph comparing today's CrashLoopBackOff decay curve with a linear 5s delay for a container that exits successfully every 10s"](flatratesuccessvstoday.png "linear vs today's CrashLoopBackoff")

**Why not?**: This provides a workaround (and therefore, opportunity for abuse),
where application developers could catch any number of internal errors of their
workload in their application code, but exit successfully, forcing extra fast
restart behavior in a way that is opaque to kubelet or the cluster operator.
Something similar is already being taken advantage of by application developers
via wrapper scripts, but this causes no extra strain on kubelet as it simply
causes the container to run indefinitely and uses no kubelet overhead for
restarts.

#### On Success and the 10 minute recovery threshold

The original version of this proposal included a change specific to Pods
transitioning through the "Succeeded" phase to have flat rate restarts. On
further discussion, this was determined to be both too risky and a non-goal for
Kubernetes architecturally, and moved into the Alternatives section. The risk
for bad actors overloading the kubelet is described in the Alternatives section
and is somewhat obvious. The larger point of it being a non-goal within the
design framework of Kubernetes as a whole is less transparent and discussed
here.

After discussion with early Kubernetes contributors and members of SIG-Node,
it's become more clear to the author that the prevailing Kubernetes assumption
is that that on its own, the Pod API best models long-running containers that
rarely or never exit themselves with "Success"; features like autoscaling,
rolling updates, and enhanced workload types like StatefulSets assume this,
while other workload types like those implemented with the Job and CronJob API
better model workloads that do exit themselves, running until Success or at
predictable intervals. In line with this assumption, Pods that run "for a while"
(longer than 10 minutes) are the ones that are "rewarded" with a reset backoff
counter -- not Pods that exit with Success. Ultimately, non-Job Pods are not
intended to exit Successfully in any meaningful way to the infrastructure, and
quick rerun behavior of any application code is considered to be an application
level concern instead.

Therefore, even though it is widely desired by commenters on
[Kubernetes#57291](https://github.com/kubernetes/kubernetes/issues/57291), this
KEP is not pursuing a different backoff curve for Pods exiting with Success any
longer.

For Pods that are today intended to rerun after Success, it is instead suggested
to 

1. exec the application logic with an init script or shell that reruns it
   indefinitely, like that described in
   [Kubernetes#57291#issuecomment-377505620](https://github.com/kubernetes/kubernetes/issues/57291#issuecomment-377505620):
  ```
  #!/bin/bash

  while true; do
      python /code/app.py
  done
  ```
2. or, if a shell in particular is not desired, implement the application such
   that it starts and monitors the restarting process inline, or as a
   subprocess/separate thread/routine

The author is aware that these solutions still do not address use cases where
users have taken advantage of the "cleaner" state "guarantees" of a restarted
pod to alleviate security or privacy concerns between sequenced Pod runs. In
these cases, during alpha, it is recommended to take advantage of the
`restartPolicy: Rapid` option, with expectations that on further infrastructure
analysis this behavior may become even faster.

This decision here does not disallow the possibility that this is solved in
other ways, for example:
1. the Job API, which better models applications with meaningful Success states,
   introducing a variant that models fast-restarting apps by infrastructure
   configuration instead of by their code, i.e. Jobs with `restartPolicy:
   Always` and/or with no completion count target
2. support restart on exit 0 as a directive in the container runtime or as a
   common independent tool, e.g. `RESTARTABLE CMD mycommand` or
   `restart-on-exit-0 -- mycommand -arg -arg -arg`
3. formalized reaper behavior such as discussed in
   [Kubernetes#50375](https://github.com/kubernetes/kubernetes/issues/50375)

However, there will always need to be some throttling or quota for restarts to
protect node stability, so even if these alternatives are pursued separately,
they will depend on the analysis and benchmarking implementation during this
KEP's alpha stage to stay within node stability boundaries. 


##### Related: API opt-in for flat rate/quick restarts when transitioning from `Succeeded` phase

Workloads must opt-in with `restartPolicy: FastOnSuccess`, as a
foil to `restartPolicy: OnFailure`. In this case, existing workloads with
`restartPolicy: Always` or ones not determined to be in the critical path would
use the new, yet still relatively slower, front-loaded decay curve and only
those updated with `FastOnSuccess` would get truer fast restart behavior.
However, then it becomes impossible for a workload to opt into both
`restartPolicy: FastOnSuccess` and `restartPolicy: Rapid`.

##### Related: `Succeeded` vs `Rapid`ly failing: who's getting the better deal?

When both a flat rate `Succeeded` and a `Rapid` implementation were combined in
this proposal, depending on the variation of the initial value, the first few
restarts of a failed container would be faster than a successful container,
which at first look seems backwards.

!["A graph showing the intersection of delay curves between a linear rate for
success and an exponential rate for rapid
failures"](successvsrapidwhenfailed.png "success vs rapid CrashLoopBackoff
dcay")

However, based on the use cases, this is still correct because the goal of
restarting failed containers is to take maximum advantage of quickly recoverable
situations, while the goal of restarting successful containers is only to get
them to run again sometime and not penalize them with longer waits later when
they've behaving as expected.

### Front loaded decay with interval
In an effort
to anticipate API server stability ahead of the experiential data we can collect
during alpha, the proposed changes are to both reduce the initial value, and include a
step function to a higher delay cap once the decay curve triggers the same
number of total restarts as experienced today in a 10 minute time horizon, in
order to approximate load (though not rate) of pod restart API server requests.

In short, the current proposal is to implement a new initial
value of 1s, and a catch-up delay of 569 seconds (almost 9.5 minutes) on the 6th
retry.

!["A graph showing the delay interval function needed to maintain restart
number"](controlfornumberofrestarts.png
"Alternate CrashLoopBackoff decay")

**Why not?**: If we keep the same decay rate as today (2x), no matter what the
initial value is, the majority of the added restarts are in the beginning. Even
if we "catch up" the delay to the total number of restarts, we expect problem
with kubelet to happen more as a result of the faster restarts in the beginning,
not because we spaced out later ones longer. In addition, we are only talking
about 3-7 more restarts per backoff, even in the fastest modeled case (25ms
initial value), which is not anticipated to be a sufficient enough hit to the
infrastructure to warrant implementing such a contrived backoff curve.

!["A graph showing the changes to restarts depending on some initial values"](initialvaluesandnumberofrestarts.png "Different CrashLoopBackoff initial values")

### Late recovery

There are many use cases not covered in this KEP's target [User
Stories](#user-stories) that share the common properties of being concerned with
the recovery timeline of Pods that have already reached their max cap for their
backoff. Today, some of these Pods will have their backoff counters reset once
they have run successfully for 10 minutes. However, user stories exist where

1. the Pod will never successfully run for 10 minutes by design
2. the user wants to be able to force the decay curve to restart
   ([Kubernetes#50375](https://github.com/kubernetes/kubernetes/issues/50375))
3. the application knows what to wait for and could communicate that to the
   system (like a restart probe)

As discussed
[here in Alternatives Considered](#on-success-and-the-10-minute-recovery-threshold),
the first case is unlikely to be address by Kubernetes.

The latter two are considered out of scope for this KEP, as the most common use
cases are regarding the initial recovery period. If there is still sufficient
appetite after this KEP reaches beta to specifically address late recovery
scenarios, then that would be a good time to address them without the noise and
change of this KEP.

### More complex heuristics

The following alternatives are all considered by the author to be in the
category of "more complex heuristics", meaning solutions predicated on kubelet
making runtime decisions on a variety of system or workload states or trends.
These approaches all share the common negatives of being:
1. harder to reason about
2. of unknown return on investment for use cases relative to the investment to
   implement
3. expensive to benchmark and test

That being said, after this initial KEP reaches beta and beyond, it is entirely
possible that the community will desire more sophisticated behavior based on or
inspired by some of these considered alternatives. As mentioned above, the
observability and benchmarking work done within the scope of this KEP can help
users provide empirical support for further enhancements, and the following
review may be useful to such efforts in the future.

  * Expose podFailurePolicy to nonJob Pods
  * Subsidize successful running time/readinessProbe/livenessProbe seconds in
    current backoff delay
  * Detect anomalous workload crashes


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
