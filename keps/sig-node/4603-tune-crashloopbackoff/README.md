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
    - [API opt in for max cap decay curve (<code>restartPolicy: Rapid</code>)](#api-opt-in-for-max-cap-decay-curve-restartpolicy-rapid)
  - [User Stories](#user-stories)
    - [Task isolation](#task-isolation)
    - [Fast restart on failure](#fast-restart-on-failure)
    - [Sidecar containers fast restart](#sidecar-containers-fast-restart)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
    - [On Success](#on-success)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Front loaded decay curve methodology](#front-loaded-decay-curve-methodology)
    - [New OneOf for <code>restartPolicy</code> -- <code>Rapid</code>](#new-oneof-for-restartpolicy----rapid)
  - [Kubelet overhead analysis](#kubelet-overhead-analysis)
    - [Observability](#observability)
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
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Global override](#global-override)
  - [Per exit code configuration](#per-exit-code-configuration)
    - [Flat-rate restarts for <code>Succeeded</code> Pods](#flat-rate-restarts-for-succeeded-pods)
      - [Related: API opt-in for flat rate/quick restarts when transitioning from <code>Succeeded</code> phase](#related-api-opt-in-for-flat-ratequick-restarts-when-transitioning-from-succeeded-phase)
      - [Related: <code>Succeeded</code> vs <code>Rapid</code>ly failing: who's getting the better deal?](#related-succeeded-vs-rapidly-failing-whos-getting-the-better-deal)
  - [Front loaded decay with interval](#front-loaded-decay-with-interval)
  - [More complex heuristics](#more-complex-heuristics)
  - [Expose podFailurePolicy to nonJob Pods](#expose-podfailurepolicy-to-nonjob-pods)
    - [Subsidize running time in backoff delay](#subsidize-running-time-in-backoff-delay)
    - [Detect anomalous workload crashes](#detect-anomalous-workload-crashes)
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
length of the pod run is less than 10 minutes. This KEP proposes a two-pronged
approach to revisiting the CrashLoopBackoff behaviors for common use cases:
1. modifying the standard backoff delay to start faster but decay to the same 5m
   threshold
2. allowing Pods to opt-in to an even faster backoff curve with a lower max cap

For each of these changes, the exact values are subject to modification in the
alpha period in order to empirically derive defaults intended to
maintain node stability.

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
   Always`)
2. Provide a `restartPolicy: Rapid` option to configure even faster restarts for
specific Pod/Container (particularly sidecar containers), regardless of exit
code, reducing the max wait to 1 minute

In addition, part of the alpha period will be dedicated entirely to
   systematically stress testing kubelet and API Server with different
   distributions of workloads utilizing the new backoff curves. Therefore, this
KEP requires instrumentation of enhanced visibility into pod restart behavior to
enable Kubernetes benchmarking during the alpha phase. During the benchmarking
period of alpha, kubelet memory and CPU, API server latency, and pod restart
latency will be observed and analyzed to define the maximum allowable
restart rate for fully saturated nodes.

Longer term, these
metrics will also supply cluster operators the data necessary to better analyze
and anticipate the change in load and node stability as a result of upgrading to
these changes.

Note that proposal will NOT change:
* backoff behavior for Pods transitioning from the "Success" state -- see
  [Notes/Constraints/Caveats](#on-success-and-the-10-minute-recovery-threshold)
* the time Kubernetes waits before resetting the backoff counter -- see the
  [Notes/Constraints/Caveats](#on-success-and-the-10-minute-recovery-threshold)
* the ImagePullBackoff -- out of scope, see [Design
  Details](#relationship-with-imagepullbackoff)
* changes that address 'late recovery', or modifications to backoff behavior
  once the max cap has been reached -- see
  [Alternatives](#more-complex-heuristics)


#### Existing backoff curve change: front loaded decay

This KEP proposes changing the existing backoff curve to load more restarts
earlier by changing the initial value of the exponential backoff. A number of
alternate initial values are modelled below, until the 5 minute cap would be
reached. This proposal suggests we start with a new initial value of 1s, and
analyze its impact on infrastructure during alpha.

![](todayvs1sbackoff.png)


#### API opt in for max cap decay curve (`restartPolicy: Rapid`)

Pods and restartable init (aka sidecar) containers will be able to set a new
OneOf value, `restartPolicy: Rapid`, to opt in to an exponential backoff decay
that starts at a lower initial value and maximizes to a lower cap. This proposal
suggests we start with a new initial value of 250ms and cap of 1 minute, and
analyze its impact on infrastructure during alpha.

!["A graph showing today's decay curve against a curve with an initial value of
250ms and a cap of 1 minute for a workload failing every 10 s"](todayvsrapid.png
"rapid vs todays' CrashLoopBackoff")


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

#### Rapid curve methodology

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
scope for the first alpha to provide users a way to opt workloads in to a even
faster restart behavior.

The finalization of the initial and max cap can only be done after benchmarking.
But as a conservative first estimate for alpha in line with maximums discussed
on [Kubernetes#57291](https://github.com/kubernetes/kubernetes/issues/57291),
the initial curve is selected at initial=250ms / cap=1 minute, but during
benchmarking this will be modelled against kubelet capacity, potentially
targeting something closer to an initial value near 0s, and a cap of 10-30s. To
further restrict the blast radius of this change before full and complete
benchmarking is worked up, this is gated by a separate alpha feature gate and is
opted in to per Pod using a new `restartPolicy: Rapid` value, described below.


#### New OneOf for `restartPolicy` -- `Rapid`
`restartPolicy` is an immutable field in podSpec and containerSpec. If set in
podSpec, each container in the Pod inherits the Pod's restart policy of either
`Never` (default), `OnFailure`, or `Always`; for a Job, the only valid options
are `Never` and `OnFailure`. In containerSpec, it is valid ONLY on init
containers and ONLY as `Always`, to configure a sidecar container that runs
continuously alongside the regular containers in the Pod.

This KEP will support a new value for this field, `Rapid`, which on feature flag
disablement will be interpreted as `Always`. If `restartPolicy: Rapid` is set or
inherited for a container and the feature flag for this feature is turned on for
a given cluster, that container will follow the new Rapid backoff curve.

Due to configuring this as another option to this field, this would make Rapid
backoff possible for restartable init (aka sidecar) containers, Pods,
Deployments, StatefulSets, ReplicaSets, DaemonSets, but NOT pure init
containers, Jobs or CronJobs.

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

### Observability

Again, let it be known that by definition this KEP will cause pods to restart
faster and more often than the current status quo and such a change is desired.
However, to do so as safely as possible, improved visibility into cluster
restart behavior is needed both for benchmarking this change and for cluster
operators to be able to quantify the risk posed to their specific clusters on
upgrade.

This KEP requires the ability to determine, for a given percentage of
heterogenity between "Succeeded" terminating pods, crashing pods whose
`restartPolicy: Always`, and crashing pods whose `restartPolicy: Rapid`, 
 * what is the load and rate of Pod restart related API requests to the API
   server?
 * what are the performance (memory, CPU, and latency) effects on the kubelet
   component?

In order to answer these questions, metrics tying together the number of
container restarts, the container restart policy (inherited or declared), and
the terminal state of a container before restarting must be tracked. For a more
complete picture, pod lifecycle duration in CrashLoopBackoff state as opposed to
Running state would also be useful.

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
available if the `restartPolicy` is set to `Never`. As a result, to expose the
new backoff curve Jobs using this feature, the updated backoff curve must also
be implemented in the Job controller.

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
compeltely different pattern, as unlike with CrashLoopBackoff the types of
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

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

This feature requires two levels of testing: the regular enhancement testing
(described in the template below) and the stress/benchmark testing required to
increase confidence in ongoing node stability given heterogeneous backoff timers
and timeouts.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

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

- `<package>`: `<date>` - `<test coverage>`

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

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

- Changes to existing backoff curve implemented behind a feature flag 
    1. Front-loaded decay curve for all workloads with
       `restartPolicy: Always`
    3. Front-loaded, low max cap backoff curve for workloads with `restartPolicy:
       Rapid`
- New OneOf option `Rapid` for `pod.spec.restartPolicy` and
  `pod.spec.container.restartPolicy` (restartable init aka sidecar containers
  only), converted to `Always` on downgrade or without feature flag
- Metrics implemented to expose pod and container restart policy statistics,
  exit states, and runtimes
- Initial e2e tests completed and enabled
  * Feature flag on or off
  * pod and container restart policy `Rapid` upgrade and downgrade path
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
- Conformance test added for pods/sidecar containers with `restartPolicy: Rapid`


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

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ReduceDefaultCrashLoopBackoffDecay` and `EnableRapidCrashLoopBackoffDecay`
  - Components depending on the feature gate: `kube-apiserver`, `kubelet`

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Yes, `ReduceDefaultCrashLoopBackoffDecay` changes the default backoff curve for
exiting Pods and sidecar containers when `restartPolicy` is either `OnFailure`
or `Always`.

Since we currently only have anecdotal benchmarking, the alpha will implement
the most conservative modeled initial value, 1s, resulting in 3 excess restarts
per backoff lifetime. (See [this section](#front-loaded-decay-curve-methodology)
for the source.]

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, disable is supported.

For `ReduceDefaultCrashLoopBackoffDecay`, if this is disabled, once kubelet is
restarted it will initialize the default backoff to the prior initial value of
10s, and all restart delays thereafter will be calculated against this equation.

For `EnableRapidCrashLoopBackoffDecay`, if this is disabled, once kube-apiserver
is restarted it will serve `restartPolicy` fields set to `Rapid` as `Always`.

###### What happens if we reenable the feature if it was previously rolled back?

Both features can also be reenabled.

For `ReduceDefaultCrashLoopBackoffDecay`, if this is reenabled, once kubelet is
restarted it will initialize the default backoff again to the new initial value
of 1s, and all restart delays thereafter will be calculated against this
equation.

For `EnableRapidCrashLoopBackoffDecay`, if this is disabled, once kube-apiserver
is restarted it will serve `restartPolicy` fields set to `Rapid` again as
`Rapid`, which kubelet will be able to interpret.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

Yes, this requires tests for
* switching `ReduceDefaultCrashLoopBackoffDecay` on or off
* switching `EnableRapidCrashLoopBackoffDecay` off when there are workloads with
  `restartPolicy: Rapid` set
* switching `EnableRapidCrashLoopBackoffDecay` off when there are no workloads
  with `restartPolicy: Rapid` set
* switching `EnableRapidCrashLoopBackoffDecay` off, and then back on again, when
  there are workloads with `restartPolicy: Rapid` set

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

It will not result in NEW API calls.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No, this KEP will not result in any new API types.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No, this KEP will not result in any new calls to the cloud provider.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No, this KEP will not result in increasing size or count of the existing API objects.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

Maybe! As containers will be restarting more, this may affect "Startup latency
of schedulable stateless pods", "Startup latency of schedule stateful pods".
This is directly the type of SLI impact that a) the split between the default
behavior change and the `Rapid` opt in is trying to mitigate, and b) one of the
targets of the benchmarking period during alpha.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

Yes! We expect more CPU usage of kubelet as it processes more restarts. During
the alpha benchmarking period, we will be quantifying that amount in fully and
partially saturated nodes with both the new default backoff curve and the
`Rapid` backoff curve.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

It's possible, and is why during this alpha period we must benchmark fully
saturated nodes with the most aggressive restart characteristics.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

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

### Global override

Allow an override of the global constant of a maximum backoff period (or other
settings) in kubelet configuration.

### Per exit code configuration

One alternative is for new contianer spec values that allow individual containers to
respect overrides on the global timeout behavior depending on the exit reason.
These overrides will exist for the following reasons:

* image download failures
* workload crash: any non-0 exit code from the workload itself
* infrastructure events: terminated by the system, e.g. exec probe failures,
  OOMKill, or other kubelet runtime errors
* success: a 0 exit code from the workload itself

These had been selected because there are known use cases where changed restart
behavior would benefit workloads epxeriencing these categories of failures.

#### Flat-rate restarts for `Succeeded` Pods

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

As discussed in
[Notes/Constraints/Caveats](#on-success-and-the-10-minute-recovery-threshold),
the first case is unlikely to be address by Kubernetes.

The latter two are considered out of scope for this KEP, as the most common use
cases are regarding the initial recovery period. If there is still sufficient
appetite after this KEP reaches beta to specifically address late recovery
scenarios, then that would be a good time to address them without the noise and
change of this KEP.

### More complex heuristics

The following alternatives are all considered by the author to be in the category of "more complex heuristics", meaning solutions predicated on kubelet making runtime decisions on a variety of system or workload states or trends. These approaches all share the common negatives of being:
1. harder to reason about
2. of unknown return on investment for use cases relative to the investment to implement
3. expensive to benchmark and test

That being said, after this initial KEP reaches beta and beyond, it is entirely possible that the community will desire more sophisticated behavior based on or inspired by some of these considered alternatives. As mentioned above, the observability and benchmarking work done within the scope of this KEP can help users provide empirical support for further enhancements, and the following review may be useful to such efforts in the future.

### Expose podFailurePolicy to nonJob Pods

TBD

#### Subsidize running time in backoff delay
FIXME: Subsidize latest succesful pod running time/readinessProbe/livenessProbe
into the CrashLoopBackOff backoff, potentially restarting the backoff counter as
a result 

#### Detect anomalous workload crashes

TBD


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
