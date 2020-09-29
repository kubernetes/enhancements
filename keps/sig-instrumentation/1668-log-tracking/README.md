<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
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
# Log tracking for K8s component log

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
  - [New three unique logging meta-data](#new-three-unique-logging-meta-data)
  - [Note](#note)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Summary of Cases](#summary-of-cases)
  - [Logging metadata](#logging-metadata)
- [Design Details](#design-details)
  - [Prerequisite](#prerequisite)
  - [Design of ID propagation (incoming request to webhook)](#design-of-id-propagation-incoming-request-to-webhook)
  - [Design of Mutating webhook](#design-of-mutating-webhook)
  - [Design of ID propagation (controller)](#design-of-id-propagation-controller)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
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
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
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

This KEP proposes a method for adding new three unique logging meta-data into K8s component logs.
It makes us more easy to identify specific logs related to an user request (such as `kubectl apply`) and object (such as Pod, Deployment).
It is expected to reduce investigation cost greatly when trouble shoothing.
### New three unique logging meta-data

We use three meta-data. These meta-data have different features and are used for troubleshooting from different perspectives.

| meta-data name | feature |
| ------ | ------ |
| trace-id | spans an user request. unique for user's request |
| span-id | spans a controller action. unique for controller action |
| initial-trace-id | spans the entire object lifecycle. unique for related objects |

### Note

This KEP is **how** a component could add meta-data to logs. To actually add meta-data to K8s component logs, the following procedure is necessary in addition.
- Open issues for each component, and discuss them with the SIGs that own that component.
- After get agreement, utilize this KEP's feature to change the source code that outputs log to add meta-data into these logs.
Please note that this KEP alone does not change the log format(does not add meta-data to logs).
## Motivation

<!--
This section is for explicitly listing the motivation, goals and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Tracking logs among each Kubernetes component related to specific an user operation and objects is very tough work.
It is necessary to match logs by basically using timestamps and object's name as hints.
If multiple users throw many API requests at the same time, it is very difficult to track logs across each Kubernetes component log.


### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
 - Implement method which propagates new logging meta-data among each K8s component
 - Design and implement so as not to interfere with [Tracing KEP](https://github.com/kubernetes/enhancements/pull/1458)
   - e.g. implement of initial-trace-id, adding trace-id to object annotation executed in mutating webhook, etc.
### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
 - Add new logging metadata into actual K8s component logs
   - This task will be done by opening  issues after completing this KEP
 - To centrally manage the logs of each Kubernetes component with Request-ID (This can be realized with existing OSS such as Kibana, so no need to implement into Kubernetes components).

## Proposal

### User Stories (Optional)

#### Story 1

Suspicious user operation(e.g. unknown pod operations) or cluster processing(e.g. unexpected pod migration to another node) is detected.
Users want to get their mind around the whole picture and root cause.
As part of the investigation, it may be necessary to scrutinize the relevant logs of each component in order to figure out the series of cluster processing.
It takes long time to scrutinize the relevant logs without this log tracking feature, because component logs are independent of each other, and it is difficult to find related logs and link them.

This is similar to the [Auditing](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/), except for the following points.

 - Audit only collects information about http request sending and receiving in kube-apiserver, so it can't track internal work of each component.
 - Audit logs can't be associated to logs related to user operation (kubectl operation), because auditID is different for each http request.

#### Story 2

Failed to attach PV to pod
Prerequisite: It has been confirmed that the PV has been created successfully.
In this case, the volume generation on the storage side is OK, and there is a possibility that the mount process to the container in the pod is NG.
In order to identify the cause, it is necessary to look for the problem area while checking the component (kubelet) log as well as the system side syslog and mount related settings.

This log tracking feature is useful to identify the logs related to specific user operation  and cluster processing, and can reduce investigation cost in such cases.
### Summary of Cases

 - Given a component log(such as error log), find the API request that caused this (error) log.
 - Given an API Request(such as suspicious API request), find the resulting component logs.

### Logging metadata

We use three logging meta-data, and propagate them each K8s component by using OpenTelemetry.
OpenTelemetry has SpanContext which is used for propagation of K8s component.

| meta-data name | feature |
| ------ | ------ |
| trace-id | We use SpanContext.TraceID as trace-id<br>trace-id spans an user request.<br>trace-id is unique for user's request |
| span-id | We use SpanContext.SpanID as span-id<br>span-id spans a controller action.<br>span-id is unique for controller action |
| initial-trace-id | We implement new id(InitialTraceID) to SpanContext<br>We use SpanContext.InitialTraceID as initial-trace-id<br>initial-trace-id spans the entire object lifecycle. <br> initial-trace-id is unique for related objects |

All of three id's inception is from object creation and it dies with object deletion



## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->
### Prerequisite
We need to consider three cases:
- Case1: Requests from kubectl that creating an object
- Case2: Requests from kubectl other than creating (e.g. updating, deleting) an object
- Case3: Requests from controllers

The design below is based on the above three cases

### Design of ID propagation (incoming request to webhook)

**1. Incoming request to apiserver from kubectl or controller**
- For request from kubectl, request's header does not have trace-id, span-id or initial-trace-id
- For request from controller, request's header has trace-id, span-id and initial-trace-id

**2. Preprocessing handler (othttp handler)**  
2.1 Do othttp's original Extract(), and get SpanContext
- For request from kubectl, result is null (no trace-id, span-id, initial-trace-id)
- For request from controller we can get trace-id, span-id and initial-trace-id  
2.2 Create/Update SpanContext
- For request from kubectl
  - Since we don't get any SpanContext, do StartSpan() to start new trace (new trace-id and span-id)
  - the new SpanContext will be saved in the request's context "r.ctx"
- For request from controller
  - Since we get SpanContext, do StartSpanWithRemoteParent() to update the SpanContext (new span-id)
  - the updated SpanContext will be saved in the request's context "r.ctx"

**3. Creation handler**  
3.1 do our new Extract() to get initial-trace-id from request header to a golang ctx
- For request from kubectl we can't get initial-trace-id
- For request from controller we can get initial-trace-id  
3.2 get SpanContext from r.ctx to golang ctx

Notice that in this creation handler, the request will be consumed, so we need golang ctx to carry our information for propagation in apiserver.

**4. Make new request for sending to webhook**  
4.1 call othttp's original Inject() to inject the trace-id and span-id from golang ctx to header
4.2 call our new Inject() to inject the initial-trace-id from golang ctx to header  
- For request from kubectl we don't have initial-trace-id, so do nothing
- For request from controller we can do this

the order above(4.1 and 4.2) does not matter

### Design of Mutating webhook
check the request's header
- if there is initial-trace-id, add trace-id, span-id and initial-trace-id to annotation (This is the case for requests from controller.)
- if there is no initial-trace-id, check the request's operation
  - if operation is create, copy the trace-id as initial-trace-id, and add trace-id, span-id and initial-trace-id to annotation (This is the case for requests from kubectl create.)
  - if operation is not create, add trace-id, span-id to annotation (This is the case for requests from kubectl other than create.)

### Design of ID propagation (controller)
When controllers create/update/delete an object A based on another B, we propagate context from B to A. E.g.:
```
    ctx = traceutil.WithObject(ctx, objB)
    err = r.KubeClient.CoreV1().Create(ctx, objA...)
```
We do propagation across objects without adding traces to that components.

### Risks and Mitigations

TBD

### Test Plan

All added code will be covered by unit tests.

### Graduation Criteria

Alpha should provide basic functionality covered with tests described in this KEP.

#### Alpha -> Beta Graduation

- Feature covers all important workload objects
- All necessary tests are completed

#### Beta -> GA Graduation

- Feedback about this feature is collected and addressed
- Enabled in Beta for at least two releases without complaints

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: ObjectTraceIDs
    - Components depending on the feature gate: kube-apiserver, kube-controller-manager
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes.

* **What happens if we reenable the feature if it was previously rolled back?**
  Objects created during the rollback will have no initial-trace-id until they
  are recreated.

* **Are there any tests for feature enablement/disablement?**
  Unit test can ensure that the feature enablement/disablement is valid.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  The proposed ids will not be added or updated. It will not have impact
  on running workloads.

* **What specific metrics should inform a rollback?**
  N/A

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  N/A

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  N/A

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  It can be determined by checking objects annotation with proposed ids.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  N/A

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  N/A

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  N/A

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  N/A

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  N/A

* **Will enabling / using this feature result in introducing new API types?**
  N/A

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
  N/A

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  N/A

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  N/A

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  TBD

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  The feature will be unavailable.

* **What are other known failure modes?**
  TBD

* **What steps should be taken if SLOs are not being met to determine the problem?**
  N/A

## Implementation History

* 2020-09-01: KEP proposed
* 2020-09-28: PRR questionnaire updated
