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

We have a number of issues across the repo, as 
* kubectl diff returns 1 which means "Differences were found" [kubernetes #99354](https://github.com/kubernetes/kubernetes/issues/99354)
* Wrong error code thrown when list of a certain resource is empty [kubectl #847](https://github.com/kubernetes/kubectl/issues/847)
* kubectl exec can unexpectedly exit 0 [kubernetes #73056](https://github.com/kubernetes/kubernetes/issues/73056) 
* kubectl should warn user if they attempt to authenticate to the cluster using an expired certificate[kubernetes #39767]* (https://github.com/kubernetes/kubernetes/issues/39767)  
* kubectl exec doesn't return command exit code [kubernetes #26424](https://github.com/kubernetes/kubernetes/issues/26424) 

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
Bruce Wayne, the security administrator of the Gotham Inc company is following the development of a new product. Bruce asked
for the developers to warn when a new deployment fails because of the lack of some permission, so those permissions can be 
updated for the pipeline to work correctly.

The developers are making a lot of changes, and they keep asking for Bruce to look for every pipeline execution, even those that 
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
* Validation: check `Options` and set defaults. This runs on the client side.
* Execute based on `Options`. This might run on the client side, be dry-run, run an external command or call the APIServer.

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

```
<<[UNRESOLVED error codes ]>>
Some error codes are hard to distinguish due to server returning 404 Not Found in both cases, due to security reasons.
We may end dropping some of those as they are not implementable 
<<[/UNRESOLVED]>>
```

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
* When running validation, call cmdutil.CheckValidationErr(err) and exits with some well defined client error code, mapped to ErrorCodeClient
* When executing, delegate the returning error to a new function (cmdutil.CheckExecutionErr) that
can assess if the error contains some APIError (like forbidden, not found) or Client Error and return the proper
error. Any new Return Code from execution step should be added to errors.go and the case predicted here. 

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
* For the execution step, call CheckErr, that might delegate the error validation to a new function or follow
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

#### Alpha 

- Feature implemented and enabled only with Environment Variable

#### Beta
- Feature enabled by default, but enablement available via Environment Variable
- Maximum of 2 issues regarding kubectl exit code opened while feature was in Alpha (last release)

#### GA 
- Feature enabled by default and enablement not configurable
- No issues regarding kubectl exit code opened on the last 2 releases while feature was in Beta

### Upgrade / Downgrade Strategy

This behavior is feature gated through an Environment Variable and can be enabled or 
disabled without impact to upgrade or downgrade between versions 

### Version Skew Strategy

Not applicable, as this is just a kubectl change

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?


- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ENV VAR `KUBECTL_ERROR_CODES`
  
###### Does enabling the feature change any default behavior?

Yes, kubectl exit codes numbers will change after the FG is enabled

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, just remove the Env Var.

###### What happens if we reenable the feature if it was previously rolled back?

kubectl will start issuing different and standardized exit codes again when errors occurs.

###### Are there any tests for feature enablement/disablement?

No.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout fail? Can it impact already running workloads?

As it's related only to client side change (kubectl) it wont impact running workloads unless
they rely internally on kubectl command/binary

###### What specific metrics should inform a rollback?

Different / unexpected error/exit codes when using this feature must be watched

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not for this stage of implementation, should be tested when targeting Beta

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Not applicable

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Not applicable

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

Not applicable

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Not applicable

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

Not applicable

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

As this feature runs on client side, if API server and/or etcd is unavailable what should happen
is the feature issuing the correct exit code.

###### What are other known failure modes?

Not applicable

###### What steps should be taken if SLOs are not being met to determine the problem?
Not applicable
## Implementation History

* 2022-01-28 KEP ready to merge to Alpha
* 2021-03-16 Initial KEP PR

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
