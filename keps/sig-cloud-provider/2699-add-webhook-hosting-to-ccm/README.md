<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.** (SIG Cloud Provider)
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements** (https://github.com/kubernetes/enhancements/issues/2699)
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.** (Done)
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
# KEP-2699: Add webhook hosting capability to CCM framework

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
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 - Full control, separation of concerns](#story-1---full-control-separation-of-concerns)
    - [Story 2 - Fast and simple](#story-2---fast-and-simple)
    - [Story 3 - Immediate Cloud Provider Extraction effort](#story-3---immediate-cloud-provider-extraction-effort)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
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

This KEP will detail enhancing the CCM framework to support cloud provider
specific webhooks. The intent is to make it easy to either generate a binary
or enhance the existing CCM binary to host such webhooks. We also intend to
allow for easily linking in "standard" webhooks needed by other SIGs which
need to be customized for particular cloud providers.

## Motivation

The Cloud Controller Manager (CCM) is the binary into which the Cloud
Provider places all the controllers needed to make a Kubernetes cluster work
correctly on their Cloud. There are also occasions when it makes sense for a Cloud
Provider to want these customizations to be applied synchronously, during API 
server request handling, rather than asynchronously after a change has already 
been applied. 

Our initial example of this is from SIG Storage. These would like the functionality 
from the PersistentVolumeLabel (PVL) admission controller 
(https://github.com/kubernetes/kubernetes/issues/52617).
This needs to be completed for cloud provider extraction to complete. Several
Cloud Providers have indicated that this should be done in-line, especially as the
existing deprecated solution is implemented in an API server admission plugin, 
which is in-line in the request path.

### Goals

Our immediate goal is to allow in tree Cloud Providers to stop using the 
existing PVL admission controller and do so using the framework. However we want to
build a framework which wil be usable by similar solutions to problems. This KEP is
about the framework needed to support the PVL webhook and not the webhook itself.
The frameworks default listener for the webhook should use existing Kubernetes 
mechanism (secure serving, authz, authn) to secure itself and validate the client.
It should be possible to then easily change that configuration to any other Kubernetes
supported options for a webhook.

### Non-Goals

We are not intending to create a [general admission webhook solution](https://book.kubebuilder.io/cronjob-tutorial/webhook-implementation.html). This is just
intended to host Cloud Provider specific webhooks as part of the Control Plane.

## Proposal

We will start by adding extension hooks which can be registered in the 
cmd/cloud-controller-manager/main.go. This would be similar to the mechanism
we already use to register new controllers. The existing sample shows this with
a sample of registering the nodeipamcontroller which is not a normally installed
controller in the cloud controller manager. In a similar way we will have a 
sample of integrating a PVL mutating webhook into the sample CCM. We will also 
have the system automatically detect if there are both controllers and webhooks
registered in the binary. If both are registered it will automatically add 
command line flags allowing webhooks and/or controllers to be disabled. There will
be two separate flags, the controller flag and the webhook flag. The 
controller flag will default to the controllers being enabled. The webhook flag
will default to the webhooks being disabled. We would also like to provide a 
builder pattern for registering both the controller and webhook extensions.

Another issue to consider is how the mutating/admission webhook configuration is
written into the cluster. This may be somewhat dependent on if the Cloud Provider
intends to run the webhook server on the Control Plane or on the Cluster. We would 
recommend running the webhook server on the Control Plane. However for some Cloud 
Providers that can lead to special issues with the configuration. As such we will 
provide a flag which enables the service to automatically register the webhooks as 
part of startup.  However that functionality can be disabled, allowing the Cloud 
Provider to do their own custom registration, as part of cluster setup.

There are a few parts to this issue. If the webhook server is run on the control 
plane it may be possible to do things like assume it will be collocated with the
KAS, hence allowing the webhook server to be reference via localhost. It also means
that the webhook server could be instantiated from a static manifest. If the 
webhook server is run in the cluster, then resources such as the Node, Pod, PVs, 
and AdmissionController which are needed to start the webhook server, must all be
creatable prior to the webhook server coming up. In addition the template code for
the webhook server, should not be written such that having all the webhook servers
crash will not wedge the cluster from being able to get the webhook server started
again.

### User Stories (Optional)

The users of this KEP are Cloud Providers and feature developers whose code 
impacts Cloud Providers. The intent is to make it easy for them both develop 
features and to maintain the CCM controllers and webhooks across multiple versions. 
At the same time we are attempting to make it easy for the SIGs to make 
controllers or webhooks which can do what they know needs to be done and integrated 
into Cloud Provider specific processes. We would like to do that in a way which 
makes merging upgrades relatively painless.

#### Story 1 - Full control, separation of concerns

Some Cloud Providers would prefer to keep controllers and webhooks in different 
processses. They have concerns about attempting to run batch controllers in the 
same process as webhooks which are "in-line" and time sensitive. For these users
it is easy to either build two different binaries or have the same binary act
as two different binaries based on command line flags.

#### Story 2 - Fast and simple

For Cloud Providers who would like to keep things simple, it is easy to create
a single process which handles both controllers and webhooks. While this KEP 
does not deployment, this is a simple deployment, being fewer processes. It 
does not stop the Cloud Provider from converting to Story 1 later. This system
should make our part of this simple. Obviously the Cloud Provider would have to
change their deployment setup.

#### Story 3 - Immediate Cloud Provider Extraction effort

PVL use case. Cloud Providers want to allow customers to migrate an existing
workload to Kubernetes. That workload uses an existing persistent volume. To
get that workload migrated the end user needs to be able to link the existing
PV into the cluster. However this requires an association which requires calls
out to the cloud provider for certain kinds of storage. Ideally the lookup and
label of the PV to that pre-existing storage happens in-line when the PV is 
written. That ensures the write volume is attached to the Node/Pod when it is
scheduled and there are no race conditions.

### Notes/Constraints/Caveats (Optional)


### Risks and Mitigations

Potentially there could be problems running webhooks and controller in the same
process. Delays of 10 seconds or more can cause webhooks to fail. It is important 
to understand that irrespective of failure mode on the webhook coniguration, 
timeouts will always turn a webhook call into a FAIL. As such we are making it
easy to easily turn the CCM into two processses to mitigate this. It will be upto
the Cloud Provider to determine if they want the webhook policy to be FAIL or 
IGNORE. We will have the sample set the configuration to IGNORE as its the safe option. 
Incorrectly setting FAIL can quickly lead to a non functional cluster. Having a FAIL 
policy on Pods for example can prevent the system from allocating the webhook service, 
which prevents the webhook from ever passing.

Webhooks are configured by a runtime resource. As a consequence this configuration
can be modified to deleted at runtime. That means that an admin on the cluster can
disable or alter the functionality. This potentially makes it harder for a cloud 
provider to enforce that this logic is being applied. It also means that there 
needs to be a deployment mechanism for the webhook. It is left to the Cloud 
Provider to determine if the need for an in-line request is sufficient to override
these concerns. The Cloud Provider can alternatively use the controller route which 
is not in-line or use [an admission controller](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/), 
built into the APIServer.

We are actually changing the framework which generates the CCM and not the CCM
itself. It has been pointed out that it is not the role of the controller
manager to run webhooks. Controller managers should run controllers and webhooks
are not controllers. As we are modifying the framework, we should consider this
as we can create two processes. The CCM which just has controllers in it. We
can also create a Cloud Webhook Manager. That is being left as homework for the 
Cloud Provider. However the sample CCM which demonstrates how this will be done
will have both in the same sample to make it easy.

It is wroth being aware that the CCM derives from the KCM. The KCM (and the CCM)
predate efforts like controller runtime. The controller runtime is a good 
reference as it is demonstrates that operators and webhooks can be successfully
run inside the same binary. It further demonstrates that this is a pattern 
which is understood and followed by a significant portion of the Kubernetes
community. Having said that, we consider it more important to unify the KCM
and CCM code bases, then to build on top controller runtime. We are not saying
not to use anything from controller runtime. We are saying that if need to choose
between unifying the KCM & CCM code and building with controller runtime, we will
choose unifying the KCM & CCM code base.

## Design Details

A sample  of how the Builder pattern might look is:

```
cmOptions, err := options.NewCloudManagerOptions()
if err != nil {
klog.Fatalf("unable to initialize command options: %v", err)
}
fss := cliflag.NamedFlagSets{}
cloudManagerBuilder := app.NewCloudManagerBuilder("name")
cloudManagerBuilder.setOptions(cmOptions)
cloudManagerBuilder.setFlags(fss)
cloudManagerBuilder.registerWebhook(gvkList, handler)
cloudManagerBuilder.registerWebhook(gvkSecondList, secondHandler)
manager, err := cloudManagerBuilder(wait.NeverStop)
if err != nil {
klog.Fatalf("unable to construct cloud manager: %v", err)
}
err := command.start()
```

This will not alter the existing extension hooks in the controller manager 
framework, as they are critical for backward compatibility. The builders
are meant to be an abstraction layer on top to make the extensions easier to 
use. So for the existing controller manager code you might see changes like:

```
cloudControllerManagerBuilder.registerController("nodeipamcontroller", handler)
cloudControllerManagerBuilder.deregisterController("servicecontroller")
```

The handler in this case is likely to be of type ControllerInitFuncConstructor.

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

There will be no e2e tests in K/K as we do not build a deployable CCM in K/K.
There will be integration tests for the builder patterns.
In addition we will rely on K/cloud-provider-gcp to demonstrate that the PVL mutating webhook comes up.

### Graduation Criteria

#### Alpha

- Have the sample CCM come up and able to serve PVL mutating webhook.
- Requires that we get a PVL mutating webhook written for at least 1 Cloud Provider.

#### Beta

- TBD

#### GA

- TBD

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Not deprecated

### Upgrade / Downgrade Strategy

- Upgrade is not believed to be an issue at this point.
- Currently we are leaving upgrade as an issue for the Cloud Provider

### Version Skew Strategy

- We are currently assuming that this will be deployed as part of the control
plane. We assume it will be upgraded with the KAS, KCM and CCM.

## Production Readiness Review Questionnaire

- TBD

### Feature Enablement and Rollback

- TBD

###### How can this feature be enabled / disabled in a live cluster?

This will be built into the CCM by the Cloud Provider. Code must be written
specifically by the Cloud Provider to enable this feature.

There will be a feature gate which will be used to track the stage of the feature.
Principally this is to make users aware of the support level of the feature.
It will control if the listener can be started.
Please note however this is a library and we expect users to vendor this into their own code base.
As such we cannot control if they will remove the check rather than setting the flag.
- Feature gate name: CloudWebhookServer
- Components depending on the feature gate: cloud-controller-manager

###### Does enabling the feature change any default behavior?

This cannot just be "enabled".

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

If you build using our framework, then you will be able to disable using a command line flag.
It can also be disabled by changing the admission webhook configuration.

###### What happens if we reenable the feature if it was previously rolled back?

For new update requests it will work. However it will not change any persisted 
resources, unless they are rewritten.

###### Are there any tests for feature enablement/disablement?

- TBD

### Rollout, Upgrade and Rollback Planning

- TBD

###### How can a rollout or rollback fail? Can it impact already running workloads?

- TBD

###### What specific metrics should inform a rollback?

- TBD

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

- TBD

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

- TBD

### Monitoring Requirements

- TBD

###### How can an operator determine if the feature is in use by workloads?

By examining the admission webhook configuration.

###### How can someone using this feature know that it is working for their instance?

- TBD

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- TBD

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

- TBD

### Dependencies

- TBD

###### Does this feature depend on any specific services running in the cluster?

- It requires on mutating/validating admission webhooks.

### Scalability

- The webhooks have an advantage that they can be more easily scaled than controllers.

###### Will enabling / using this feature result in any new API calls?

It requires a new call admission webhook call.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

Depends on the Cloud Providers implementation.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes, in the same way that any additional admission webhook call does.
It is worth noting that the Cloud Provider has the option of instead
using a controller, at least for the PVL case. However that is not
the preferred mechanism. These is an optional extension mechanism.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

- TBD

### Troubleshooting

- This is a admission webhook server. Those already exist and those troubleshooting
mechanism should apply here as well.

###### How does this feature react if the API server and/or etcd is unavailable?

- This feature does not apply unless the API server is functional.

###### What are other known failure modes?

Timeouts on webhooks act as failures, so any resource sent to the CCM will fail
if it times out.

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- TBD

## Drawbacks

- TBD

## Alternatives

The primary alternative is to use controllers to solve all the problems. 
This has an issue for things which need to be done in-line. If it is not 
ok for state to be missing from a resource between creation and usage,
the controllers are a problem

Initializers solve the problem between creation and usage, however this
solution has been deprecated.

## Infrastructure Needed (Optional)

- TBD
