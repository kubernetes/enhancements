<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

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
# KEP-5752: ResourceQuota ObjectLabel Scope Label Management

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
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Label-Based Quota Selection](#story-1-label-based-quota-selection)
    - [Workload Exclusion Labels](#story-2-workload-exclusion-labels)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

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
-->

This KEP proposes enhancements to ResourceQuota management using new ObjectLabel scopes. The primary focus is on establishing proper label-based resource accounting and quota management in Kubernetes clusters for pods and PersistentVolumeClaims. 
This proposal provides a label selection mechanism, but it is up to the cluster administrator to ensure an appropriate level of security when using it.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

ResourceQuota with ObjectLabel scopes provides a mechanism for resource accounting and more flexible quota management. In this case ResourceQuota mostly used for limits and accounting — tracking resource usage and availability in the cluster.

This is especially acute in a multi-tenancy environment, when several teams or projects are work in the same namespace without the ability to adjust the workload for individual namespaces and, as a result individual quotas for group of application. 

Increasingly, we see the need to temporarily exclude a certain resource from the quota dynamically (for example, during the migration of a pod or pvc data), which does not allow us to do the current mechanism.

- [User scenario №1](https://github.com/kubernetes/kubernetes/issues/77508#issuecomment-492101342)
- [User scenario №2](https://github.com/kubernetes/kubernetes/issues/77508#issuecomment-501896087)
- [User scenario №3](https://github.com/kubernetes/kubernetes/issues/77508#issuecomment-924699249)

All these scenarios reflect the real needs of users, they can be seen in similar issues, [2019](https://github.com/kubernetes/kubernetes/issues/77508), [2022](https://github.com/kubernetes/kubernetes/issues/110065), [2025](https://github.com/kubernetes/kubernetes/issues/135718)

The `ObjectLabel` functionality is intended to be used only for PersistentVolumeClaims and Pods. This KEP aims to:

1. Enable administrators to reserve resource quota based on labels by organizing pods/pvc into groups (e.g., `env=stage`, `env=production`) in one namespace.
2. Based on this mechanisms it is possible to exclude certain workloads temporary from quota calculations (e.g., `workload-type=vm-migrating`, `workload-type=data-migrating`)
3. Using in-tree ValidatingAdmissionPolicy to enforce label management

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Provide ObjectLabel quota scope in Resource Quota for quota management through label-based workload grouping for PersistentVolumeClaims and Pods
- Specify examples included ValidatingAdmissionPolicy rules to control label assignment on pods on documentation.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Modifying eviction behavior based on new ResourceQuota scopes
- Implementing quota enforcement at the scheduler level
- Supporting ObjectLabel scopes for resources other than Pods and PVCs
- Adding an automatic mechanism for generating ValidatingAdmissionPolicy rules based on labels

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Label-Based Quota Selection

As a cluster administrator, I want to reserve specific labels (e.g., `app=*` for `app=backend`, `app=frontend`) for organizing pods into quota groups. Users should not be able to arbitrarily change these labels

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: backend-quota
  namespace: shared-ns
spec:
  hard:
    requests.cpu: "20"
    limits.cpu: "40"
    requests.memory: "200Gi"
    limits.memory: "400Gi"
    persistentvolumeclaims: "100"
    pods: "100"
  scopeSelector:
    matchExpressions:
    - scopeName: ObjectLabel
      operator: In
      values: ["app=backend"]
```
Only cluster administrator can manage ResourceQuotas:

```yaml
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingAdmissionPolicy
metadata:
  name: protect-all-resourcequotas
spec:
  failurePolicy: Fail
  matchConstraints:
    resourceRules:
    - apiGroups: [""]
      apiVersions: ["v1"]
      operations: ["CREATE", "UPDATE", "DELETE"]
      resources: ["resourcequotas"]
  validations:
  - expression: >
      "cluster-admin" in request.userInfo.groups
    message: >-
      Only cluster-admin group can CREATE/UPDATE/DELETE ResourceQuotas 
      in any namespace
```

I can create a policy that allows a user in a certain group to have only specific label values (e.g.,`app=backend`, `app=frontend` or `env=production`, `env=stage`), so as not to be able to enter the quota of another team or 'env' in namespace.

```yaml
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingAdmissionPolicy
metadata:
  name: restrict-backend-app-label
spec:
  failurePolicy: Fail
  matchConstraints:
    resourceRules:
    - apiGroups: [""]
      apiVersions: ["v1"]
      operations: ["CREATE", "UPDATE"]
      resources: ["pods"]
  validations:
    expression: >
      !(
        "backend-team" in request.userInfo.groups &&
        has(object.metadata.labels["app"]) &&
        object.metadata.labels["app"] != "backend"
      )
    message: "Users in group backend-team may only set label app=backend for 'app' label key"
```

Or using MutatingAdmissionPolicy automatically:

```yaml
apiVersion: admissionregistration.k8s.io/v1beta1
kind: MutatingAdmissionPolicy
metadata:
  name: auto-backend-app-label
spec:
  failurePolicy: Fail
  matchConstraints:
    resourceRules:
    - apiGroups: [""]
      apiVersions: ["v1"]
      operations: ["CREATE", "UPDATE"]
      resources: ["pods"]
  mutations:
  - patchType: "JSONPatch"
    jsonPatch: |
      [
        {
          "op": "replace",
          "path": "/metadata/labels/app",
          "value": "backend"
        }
      ]
    matchCondition: backend-team-condition
matchConditions:
- name: backend-team-force
  expression: '"backend-team" in request.userInfo.groups'
```

#### Workload Exclusion Labels

As a cluster administrator, I want to exclude certain workloads (e.g., migration jobs with `workload-type=migration` by third-party controller) from quota calculations. Only controllers or administrators should be able to set these exclusion labels.

ValidatingAdmissionPolicy example for restriction on a specific label: 

```yaml
apiVersion: admissionregistration.k8s.io/v1beta1
kind: ValidatingAdmissionPolicy
metadata:
  name: restrict-migration-label
spec:
  failurePolicy: Fail
  matchConstraints:
    resourceRules:
    - apiGroups: [""]
      apiVersions: ["v1"]
      operations: ["CREATE", "UPDATE"]
      resources: ["pods", "persistentvolumeclaims"]
  validations:
  - expression: >
      object.metadata.labels["workload-type"] != "migration" ||
      "cluster-admin" in request.userInfo.groups ||
      request.userInfo.username == "system:serviceaccount:controller-ns:controller-sa"
    message: >-
      Only cluster-admin group and controller service account can set 
      workload-type=migration label on Pods for exlude from quota
```
As in the previous use case, we must have rights for ResourceQuota resources only for cluster administrator.

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: shared-ns-quota
  namespace: shared-ns
spec:
  hard:
    requests.cpu: "20"
    limits.cpu: "40"
    requests.memory: "200Gi"
    limits.memory: "400Gi"
    persistentvolumeclaims: "100"
    pods: "100"
  scopeSelector:
    matchExpressions:
    - scopeName: ObjectLabel
      operator: In
      values: ["env=stage"]
    - scopeName: ObjectLabel
      operator: NotIn
      values: ["workload-type=migration"]
```

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

- **Risk**: Users might attempt to bypass label restrictions by using similar label names
  - **Mitigation**: ValidatingAdmissionPolicy should validate exact label keys and values, not just patterns

- **Risk**: MutatingAdmissionPolicy might conflict with user-specified labels
  - **Mitigation**: Policy should be designed to only set labels when they are missing, or use a clear precedence model

- **Risk**: ResourceQuota modifications by unauthorized users could disrupt cluster resource management
  - **Mitigation**: RBAC rules should restrict ResourceQuota modifications to administrators only

- **Risk**: Performance impact of ValidatingAdmissionPolicy evaluations
  - **Mitigation**: ValidatingAdmissionPolicy are compiled policies that run efficiently in the API server

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->
In this iteration, we would like to focus not so much on the technical implementation as on the design of the feature itself.
Need to add a new one in a certain places:

1. Added `ResourceQuotaScopeObjectLabel` constant to the core API types:
https://github.com/kubernetes/kubernetes/blob/15673d04e30c711a7bb0f0efe6abf4baead1463b/pkg/apis/core/types.go#L6456

```/pkg/apis/core/types.go
	// Match all objects based on their labels
	ResourceQuotaScopeObjectLabel ResourceQuotaScope = "ObjectLabel"
```

2. Added `ObjectLabel` to the standard resource quota scopes:
https://github.com/kubernetes/kubernetes/blob/15673d04e30c711a7bb0f0efe6abf4baead1463b/pkg/apis/core/helper/helpers.go#L116

```pkg/apis/core/helper/helpers.go
var standardResourceQuotaScopes = sets.New(
	core.ResourceQuotaScopeTerminating,
	core.ResourceQuotaScopeNotTerminating,
	core.ResourceQuotaScopeBestEffort,
	core.ResourceQuotaScopeNotBestEffort,
	core.ResourceQuotaScopePriorityClass,
	core.ResourceQuotaScopeObjectLabel,
)
```

3. Updated validation to enforce that `ObjectLabel` scope only accepts `In` and `NotIn` operators:
https://github.com/kubernetes/kubernetes/blob/15673d04e30c711a7bb0f0efe6abf4baead1463b/pkg/apis/core/validation/validation.go#L8026C6-L8026C47

```pkg/apis/core/validation/validation.go
		case core.ResourceQuotaScopeObjectLabel:
			if req.Operator != core.ScopeSelectorOpIn && req.Operator != core.ScopeSelectorOpNotIn {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("operator"), req.Operator,
					"must be 'In' or 'NotIn' when scope is ResourceQuotaScopeObjectLabel"))
```

4. Implemented the core label matching logic in the generic evaluator:
https://github.com/kubernetes/kubernetes/blob/15673d04e30c711a7bb0f0efe6abf4baead1463b/pkg/quota/v1/evaluator/core/pods.go#L350

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

- 2025-12-25: Initial KEP proposal

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Alternative 1: Scheduler-Based Quota Enforcement

Instead of using ResourceQuota with admission control, implement quota enforcement directly in the scheduler. This was rejected because it conflates scheduling decisions with resource accounting, which should be separate concerns.

### Alternative 2: Custom Resource for Label Rules

Create a custom resource (e.g., `LabelQuotaPolicy`) to define label assignment rules. This was rejected as it adds unnecessary API surface when ValidatingAdmissionPolicy and MutatingAdmissionPolicy provide the needed functionality.

### Alternative 3: Webhook-Based Solution

Use MutatingWebhook and ValidatingWebhook instead of compiled admission policies. This was rejected because compiled policies (ValidatingAdmissionPolicy/MutatingAdmissionPolicy) provide better performance and are easier to reason about.
