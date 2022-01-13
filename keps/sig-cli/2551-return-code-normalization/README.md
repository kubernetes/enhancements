# KEP-2551: kubectl exit code standardization

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Feature Gating](#feature-gating)
- [Design Details](#design-details)
  - [Error Codes](#error-codes)
  - [Changing error checker functions](#changing-error-checker-functions)
  - [Creating new error parser functions](#creating-new-error-parser-functions)
  - [Hybrid approach](#hybrid-approach)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
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

Standardize the exit codes of kubectl when the command is successful or when it fails.

## Motivation

kubectl is composed of several commands and subcommands, that can deal only with functions
as simple as getting the current version of server and client, to more complex functions
as getting differences between a local file and the current created resource on a Kubernetes
cluster, or doing some deployment roll out.

Additionally, some of those commands may call other external system commands (like an editor, 
or a diff command, or even a plugin). 

This brings a complexity about assessing:
* Was the command executed correctly
* If not, where was the problem? On the server side? Was the certificate expired? Or the resource does not exist?
* Was that a problem on the client side? Was that a problem of an external system command?

We have a number of issues across the repo, as [kubernetes #99354](https://github.com/kubernetes/kubernetes/issues/99354),
[kubectl #847](https://github.com/kubernetes/kubectl/issues/847), 
[kubernetes #73056](https://github.com/kubernetes/kubernetes/issues/73056), 
[kubernetes #39767](https://github.com/kubernetes/kubernetes/issues/39767) and 
[kubernetes #26424](https://github.com/kubernetes/kubernetes/issues/26424) that points
to this specific problem.

### Goals

* Document possible exit codes for kubectl 
* Implement the exit code common return function (maybe as a util function) 
* Gradually implement the exit code standardization for each command  

### Non-Goals

* Define a different return for each internal kubectl step or each APIServer condition/return
* Define a return code pattern for kubectl plugins or other external calls (we will recommend, but this is not a goal)

## Proposal

* Define the majority of the behaviors that a kubectl request can face:
  * Possible errors on the client side.
  * Possible errors on the server side.
* Define a table/list of numeric error codes for each of the main cases:
  * Allow external/subcommands (i.e. diff and exec) to return their exit codes unaltered as often as possible
  * Exit codes generated by kubectl itself should be distinct, so as not to be confused with codes from external/subcommands
* Implement a common way so commands can delegate the exit code normalization to a different function

### User Stories

#### Story 1
Joice, the SRE of a big company is automating the deployment of the infrastructure with a lot of kubectl commands.
Usually they use an external plugin to verify if the manifests are linted, and issue a warning if not. Joice wants to
safelly ignore those lint errors from the plugin on the pipeline, as the exit code for linting might be well known, but 
wants to warn users when the apply command fails because of differences between the manifests.

#### Story 2
Bruce Wayne, the security administrator of the Gotham Inc Company is following the development of a new product. Bruce asked
for the developers to warn when a new deployment fails because of the lack of some permission, so those permissions can be 
updated for the pipeline to work correctly.

The developers are making a lot of changes, and they keeps asking for Bruce to look for every pipeline execution, even those that 
fails because of wrong manifests and not because of authorization issues. So Bruce needs a new mechanism: That the pipeline knows
when it fails because of the lack of the authorization on Kubernetes API Server, and then the warning is sent to the security team
only when the pipeline breaks because of a specific error code that represents this authorization failure.

#### Story 3
Roberta works as the product manager of a big CI/CD SaaS provider. They want to have in their marketplace the
execution of kubernetes commands targeting a cluster, and fastly providing a feedback to the user if the error
was due to something on the client side (like a missing flag, an invalid yaml file because of...tabs...) or if this
is due to some invalid operation on the server side.


### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

* Users already relying on specific error codes might face migration problems / false positives or negatives 
because of some specific error code change. The proposal here is to decide if the old or new return code behavior
will be used based on a well defined environment variable. The old behavior will be kept and followed by the 
deprecation path will be defaulted after a number of releases.

#### Feature Gating
In an effort to roll out these changes slowly and prevent flag bloat, this feature will be enabled using the environment variable `KUBECTL_ERROR_CODES`, which will default to an empty value.
To enable the feature, the value must be non-empty, and not equal `0`, or `false`.

## Design Details

The majority of commands already are organized as the following:
* Run Complete to complete missing information with defaults. This runs on the client side
* Run Validation to check command syntax and missing arguments. This runs on the client side
* Execute command itself. This might run on the client side, be dry-run, run an external command or call the APIServer.

One thing that should be done is map all the error codes/ints as constants in some file, so they can be automatically
documented.

### Error Codes
The following table represents the proposed error codes and the condition describing its usage.

| Code  | Description                                                                                           |
| ----- | ----------------------------------------------------------------------------------------------------- |
| 1-200 | Reserved for exit codes from exec and external commands                                               |
| 201   | Catch-all for errors where the condition is unknown or no better codes exist to describe it           |
| 202   | Missing or improper use of keyword, command, or argument                                              |
| 203   | Client configuration error, invalid or missing configuration                                          |
| 204   | Network failure, API could not be reached                                                             |
| 205   | Authentication failure, identity could not be determined                                              |
| 206   | Authorization failure, identity was determined, but does not have access to requested resource(s)     |
| 207   | Unknown or invalid request to API                                                                     |
| 208   | Request timed out                                                                                     |
| 209   | Resource not found                                                                                    |
| 210   | Resource already exists                                                                               |
| 211   | Resource expired                                                                                      |
| 212   | Conflict while updating resource                                                                      |
| 213   | An underlying service was unavailable                                                                 |
| 214   | API internal error                                                                                    |
| 215   | Too many requests                                                                                     |
| 216   | Request entity too large                                                                              |
| 217   | Unexpected response from API                                                                          |
| 255   | An external command returned an exit code equal to or greater than 201, which is reserved for kubectl |

* The reserved exit codes are documented [here](https://tldp.org/LDP/abs/html/exitcodes.html) and should be used carefully in a way to not generate conflict with existing scripts.
* With consideration of exit codes from exec and other external commands (i.e. diff), starting kubectl codes at 201 would allow the original exit codes to pass through most of the time
* Starting kubectl error exit codes at 201 allows them to be more distinct and reduce confusion as to the origin of the error, whether kubectl itself or an external command
* The default kubectl error exit code would change from 1 to 201, with the hope that kubectl would never return the default error exit code, instead using a more appropriate code

### Changing error checker functions
All of the steps above uses [cmdutil.CheckErr](https://github.com/kubernetes/kubernetes/blob/2a26f276a8c8c13b2f45927ee5ece2063950dd1d/staging/src/k8s.io/kubectl/pkg/cmd/util/helpers.go#L114) 
function to delegate the error validation. This function might be slight changed 
(without changing its signature) to instead of use the default `fatalErrHandler` 
function, use some more specific function that might verify the error against a matrix of possible errors
and exit with the right code.

The function [CheckDiffErr](https://github.com/kubernetes/kubernetes/blob/2a26f276a8c8c13b2f45927ee5ece2063950dd1d/staging/src/k8s.io/kubectl/pkg/cmd/util/helpers.go#L124)
can be used as an example of how this can be implemented.

The pro of using this design approach is that almost no change will be needed in commands, as they already use the 
CheckErr function.

The con of using this design approach is that the Error Validation functions are pretty hard to change/evolve
in the manner they are designed, calling another checkErr inner function.

For the rollout of this approach, a new non-exported error checker function needs to be developed and make the 
call to the old checkErr or the new function according to the value of the Environment Variable described in
[Risks and Mitigations](#risks-and-mitigations) 

### Creating new error parser functions

Another design solution is to create helper functions for each steps:
* When running Complete, Validate or other client side steps, call cmdutil.CheckClientErr(err) and exits with some well defined client error code, mapped to ErrorCodeClient
* When running the Run* step, delegate the returning error to a new function (cmdutil.CheckRunErr) that
can assess if the error contains some APIError (like forbidden, not found) or Client Error and return the proper
error. Any new Return Code from Run step should be added to errors.go and the case predicted here. 

The pro of this approach is that we can re-develop everything controlling the behavior.

The con of this approach is that it takes much more time and code change to point every command to the 
right error checking function.

For the rollout of this approach, the new functions will call the old `CheckErr` 
according to the value of the Environment Variable described in [Risks and Mitigations](#risks-and-mitigations), 
or will follow the new flow.

### Hybrid approach

There's an Hybrid approach that can be used:

* For the steps that run on the client side, create a new function that does an early exit/return with 
an well known exit code that will be used for all client side operations (no difference between yaml
validation, missing flag, etc)
* For the Run* step, call CheckErr, that might delegate the error validation to a new function or follow
with the old behavior, depending on the Environment Variable described above

```
<<[UNRESOLVED external commands ]>>
For commands that call external commands (diff, plugins, edit) this needs to be discussed.
<<[/UNRESOLVED]>>
```


### Test Plan

* Add Unit tests for each specific error case mapped on the error matrix
* Add Unit tests on the commands to verify for specific cases (plugins, diff) if the right exit code is returned
* Add e2e tests to verify if the right exit code is returned


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


- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ENV VAR `KUBECTL_ERROR_CODES`
  
###### Does enabling the feature change any default behavior?

Yes, kubectl exit codes numbers will change after the FG is enabled

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, just remove the Env Var.

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?
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
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
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

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

<!--
At a high level, this usually will be in the form of "high percentile of SLI
per day <= X". It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code
-->

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

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

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

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
