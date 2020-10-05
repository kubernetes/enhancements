# Manifest based registration of Admission webhooks

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Naming](#naming)
  - [New AdmissionConfig schema](#new-admissionconfig-schema)
  - [Reconfiguring manifest file](#reconfiguring-manifest-file)
    - [Behavioral details of reconfiguration](#behavioral-details-of-reconfiguration)
  - [Metrics and audit annotations](#metrics-and-audit-annotations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
    - [Removing a deprecated flag](#removing-a-deprecated-flag)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Manifest based webhook configuration allows registering admission webhooks
during kube-apiserver start up allowing for no delays in policy enforcement
between policy addition and kube-apiserver startup. 

## Motivation

Today most policy enforcement is implemented through MutatingAdmissionWebhooks
and/or ValidatingAdmissionWebhooks. These admission webhooks are registered
through creating MutatingWebhookConfiguration or ValidatingWebhookConfiguration
objects. Any policy enforcement is not in place until these webhook
configurations are created, thereby registering the webhook. This creates a gap
in enforcement spanning from when the kube-apiserver is started to until the
webhook configuration objects are created and picked up by the dynamic admission
controller. Another gap is the inability of the cluster administrator to protect
against deletion of these webhook configuration objects as
MutatingWebhookConfiguration and ValidatingWebhookConfiguration objects are not
subject to webhook admission policy.

This KEP aims to address these issues.  

### Goals

- A robust admission webhook registration process where there is no period of
  time between the kube-apiserver coming up and the registration of an admission
  webhook.

- The registration of webhooks registered through this process should be
  protected from alteration by API requests made by cluster users.

- It should be possible to alter webhooks registered through this new process
  without restarting the kube-apiserver.
 
## Proposal

In short, this proposal is about augmenting `AdmissionConfig`'s plugin
`configuration` [specification](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#authenticate-apiservers)
to include a path to a configuration file containing one of the following:
- `admissionregistration.k8s.io/v1.ValidatingWebhookConfigurationList` 
- `admissionregistration.k8s.io/v1.MutatingWebhookConfigurationList` 
- v1.List with items of admissionregistration.k8s.io/v1.ValidatingWebhookConfiguration 
- v1.List  with items of admissionregistration.k8s.io/v1.MutatingWebhookConfiguration.
This configuration is loaded as manifest based webhooks in api servers. These
webhooks are called by validating and mutating admission plugins along with
dynamically loaded webhooks for relevant admission requests. This also means
that these webhooks are not API visible objects and hence cannot be modified by
an API user.

### Naming
All webhooks in the manifest file need to have unique names. If a new webhook
configuration API object with a same name as a webhook in the manifest is added,
both webhook would be invoked. Essentially, webhooks in the manifest file will
be treated as belonging to a different domain from the webhooks registered
through the API. 

### New AdmissionConfig schema

A `webhooksFile` field is added to `configuration` field of a plugin object in
`AdmissionConfig`.

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: AdmissionConfiguration
plugins:
- name: ValidatingAdmissionWebhook
  configuration:
    ....
    webhooksFile: "<path-to-manifest-file>"
- name: MutatingAdmissionWebhook
  configuration:
    ....
    webhooksFile: "<path-to-manifest-file>"
```

### Reconfiguring manifest file

The manifest file is watched so that the webhook configuration can be
dynamically changed by editing the contents of the file. In addition to watching
the file for changes, we would also read the file periodically checking for
changes.

#### Behavioral details of reconfiguration
This section details the behavior of the mechanism when encountering non-optimal
situations.

In cases where:
* the given manifest file can no longer be found or
* attempts at reading the manifest file result in an error or
* the contents of the manifest file can no longer be parsed or
* validation errors are encountered when parsing the webhook configuration of
the manifest file,

the previously successfully loaded set of webhooks will continue to be invoked
when necessary. An error would be surfaced through a metric (See the metrics
section). If the file or a valid version of it re-appears, the new file will to
loaded and if valid, the webhooks in this file would be used for future
invocations. Also, the error metric would be unset.

### Metrics and audit annotations
For the administrator to uniquely monitor manifest based webhooks, two
additional metrics would be added at [stability level `ALPHA`](https://github.com/kubernetes/enhancements/blob/master/keps/sig-instrumentation/20190404-kubernetes-control-plane-metrics-stability.md#stability-classes)
to the existing set of metrics.

a. Webhook metadata: This metric would contain metadata of each webhook invoked,
both API based and manifest based. The metadata includes the webhook type
(mutating/validating) and if it is manifest based.

b. Manifest errors: Since the manifest file can be reconfigured post API server
startup, we would add a metric informing users if there were errors
loading/parsing the current manifest file.

Audit annotations for mutating webhooks would also carry an additional field
`manifestBased` indicating the invocation of a manifest based webhook. 

## Design Details

### Test Plan

Unit tests for:

- Loading, parsing, defaulting, and validation of manifest based webhoooks for
  both Mutating and Validating webhooks.
- Reconfiguration of the manifest for both Mutating and Validating webhooks.
- Additional metrics being set.

Integration tests added to test/integration/apiserver/admissionwebhook
covering the following scenarios:

- Standard expected behavior for both Mutating and Validating webhooks including
  API server not starting on an invalid version of the manifest file.
- Standard expected behavior with client auth for both Mutating and Validating
  webhooks.
- Successful reconfiguration of the manifest for both Mutating and Validating
  webhooks including metric checks.
- Failed reconfiguration of the manifest for both Mutating and Validating
  webhooks including metric checks.

### Graduation Criteria

#### Alpha -> Beta Graduation

#### Beta -> GA Graduation

#### Removing a deprecated flag

### Upgrade / Downgrade Strategy

### Version Skew Strategy

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
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
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).

* **What happens if we reenable the feature if it was previously rolled back?**

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
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


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  No.

* **Will enabling / using this feature result in introducing new API types?**
  No.

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**
  No.

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  No.

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  No.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  No.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
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

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2020-04-21: KEP introduced
- 2020-10-05: Unresolved issues addressed. Testing plan, Scalability section,
updated.

## Drawbacks

- Reduced visibility for users wanting to list all active admission webhooks.
  This KEP has similar visibility characteristics as compiled in admission
  controllers.

## Alternatives

Adding Deny policies to RBAC allowing a cluster administrator to create roles
that deny access to certain webhook configuration objects was considered. But,
adding Deny policies to RBAC has far reaching consequences like
redesigning/changing the implementation of object watchers.
