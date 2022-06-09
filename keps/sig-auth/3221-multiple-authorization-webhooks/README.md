# KEP-3221: Multiple Authorization Webhooks

<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
- [x] **Create an issue in kubernetes/enhancements**
- [x] **Make a copy of this template directory.**
- [x] **Fill out as much of the kep.yaml file as you can.**
- [x] **Fill out this file as best you can.**
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
-->


<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Protecting installed CRDs](#story-1-protecting-installed-crds)
    - [Story 2: Preventing unnecessarily nested webhooks](#story-2-preventing-unnecessarily-nested-webhooks)
    - [Story 3: Denying requests on certain scenarios](#story-3-denying-requests-on-certain-scenarios)
    - [Story 4: Varying defaults across versions of the API](#story-4-varying-defaults-across-versions-of-the-api)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Monitoring](#monitoring)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (1.25)](#alpha-125)
    - [Beta](#beta)
    - [GA](#ga)
    - [GA + 3 cycles](#ga--3-cycles)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Today, `kube-apiserver` allows one to configure the authorization chain using a set of command line flags of the format `--authorization-*`.
And cluster administrators can only enable one webhook as a part of the authorization chain using the `--authorization-modes` flag.
This limits admins from creating authorization chains with multiple webhooks that validate requests in a certain order.
This proposal makes the case for a more structured config to define the authorization chain for `kube-apiserver` and allowing multiple webhooks with well-defined parameters and enabling fine grained control like explicit Deny authorizer and more definitive `SubjectAccessReviews`.

## Motivation

Cluster administrators should be able to specify more than one authorization webhook in the API Server handler chain.
They also need to be able to declaratively configure the authorizer chain using a configuration file.

It should also be easy to say when to Deny requests, for example, when a webhook is unreachable.

### Goals

- Define a configuration file format for configuring Kubernetes API Server Authorization chain.
- Allow ordered definition of authorization modes.
- Allow definition of multiple webhooks in the authorization chain.
- Enable user to define the policy when a webhook can't be reached due to network errors etc.
- Enable more defnitive results for a `SubjectAccessReview` by allowing config to distinguish webhooks which might deny access.

### Non-Goals

- Allow resource/user based pre-filtering of webhooks to prevent unnecessary invocations.
- Hot reload of the configuration.

## Proposal

Add a configuration format having specific precedence order and defined failure modes for configuring authorizer chain.

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: AuthorizationConfiguration
authorizers:
- type: Webhook
  webhook:
    server: https://<path-to-webserver>
    authentication:
        certificateAuthority: </path/to/ca.pem> # either this or certificateAuthorityData
        clientCertificate: </path/to/cert.pem> # either this or clientCertificateData
        clientKey: </path/to/key.pem> # either this or clientKeyData
    authorizedTTL: 5m
    unauthorizedTTL: 30s
    subjectAccessReviewVersion: v1
    onError: Deny
- type: Node
- type: RBAC
- type: Webhook
  webhook:
    kubeConfigFile: <path-to-allow-webhook-config>
        authentication:
        certificateAuthority: </path/to/ca.pem> # either this or certificateAuthorityData
        clientCertificate: </path/to/cert.pem> # either this or clientCertificateData
        clientKey: </path/to/key.pem> # either this or clientKeyData
    authorizedTTL: 5m
    unauthorizedTTL: 30s
    subjectAccessReviewVersion: v1
    onError: NoOpinion
```

### User Stories (Optional)

#### Story 1: Protecting installed CRDs

Having certain Custom Resource Definitions available at cluster startup has been a long requested feature.
One of the blockers for having a controller reconciling those CRDs is to have a protection mechanism for them, which can be achieved through Authorization Webhooks.
However since `kube-apiserver` only allows specifying a single webhook, this would result in cluster administrators being not able to specify their own.
Support for multiple webhooks would make this possible.
Moreover, RBAC rules can't be used here since RBAC allows to one add permissions but not deny them.
We won't be able to add restrictions on 'non-system' users only for a set of resources using RBAC; 'non-system' users here refers to anyone who shouldn't be able to update/delete the protected set of CRDs.

Relevant Discussion Thread is sig-apimachinery: https://groups.google.com/g/kubernetes-sig-api-machinery/c/MBa19WTETMQ

#### Story 2: Preventing unnecessarily nested webhooks

Consider a system administrator who would like to apply a set of validations to certain requests before handing it off to webhooks defined using frameworks like Open Policy Agent.

They would have to nested webhooks within the one added to the auth chain to have the intended effect.
This enhancement allows the administrator to configure this behavior via a structured API and invoke the additional webhook only when relevant.
This also allows administrators to define `onError` behavior for separate webhooks, leading to more predictable outcomes.

#### Story 3: Denying requests on certain scenarios

The authorizer chain should be powerful enough to deny anyone making a request if certain conditions are satisfied.

#### Story 4: Varying defaults across versions of the API

If the default values for configuration evolve over time, affected users might have to override these values in case they are affected by supplying the flags with updated defaults.
A configuration format like the one proposed can be versioned over time with changing defaults, mitigating this risk by only affecting users who knowingly bump the version.

### Notes/Constraints/Caveats (Optional)

TBD

### Risks and Mitigations

- In HA clusters, there may be a skew in how the `kube-apiserver` processes in each are configured. This may create inconsistencies. Mitigation is to have the cluster bootstrapper handle such scenarios.
- In case an administrator enables this feature and the webhook kubeconfig file is invalid or doesn't exist at the specified path, `kube-apiserver` on that node will not be able to start. This can be mitigated by fixing the malformed values.


## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

We would like to introduce a structured file format which allows authorization to be configured using a flag (`--authorization-config-file`) which accepts a path to a file on the disk.
This feature can be enabled or disabled by the explicit feature flag `--authorization-config-from-file`.
The proposed structure is illustrated below:

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: AuthorizationConfiguration
authorizers:
- type: Webhook
  webhook:
    server: https://<path-to-webserver>
    authentication:
        certificateAuthority: </path/to/ca.pem> # either this or certificateAuthorityData
        clientCertificate: </path/to/cert.pem> # either this or clientCertificateData
        clientKey: </path/to/key.pem> # either this or clientKeyData
    authorizedTTL: 5m
    unauthorizedTTL: 30s
    subjectAccessReviewVersion: v1
    onError: Deny | NoOpinion
- type: Node
- type: RBAC
- type: Webhook
  webhook:
    kubeConfigFile: <path-to-allow-webhook-config>
    authorizedTTL: 5m
    unauthorizedTTL: 30s
    subjectAccessReviewVersion: v1
    onError: NoOpinion
```

Validation will allow multiple authorizers of type "Webhook" to be added to the config, but one authorizer each for other types.
The ordering of this chain will be decided by the order specified in the file.

The keys `kubeConfigFile`, `authorizedTTL`, `unauthorizedTTL` and `subjectAccessReviewVersion` version accept values corresponding to flags `--authorization-webhook-config-file`, `--authorization-webhook-cache-authorized-ttl`, `--authorization-webhook-cache-unauthorized-ttl` and `--authorization-webhook-version` respectively.

This diagram shows how the configuration flags map to configuration file keys for a webhook:
![](https://i.imgur.com/PJSmgOq.jpg)

The `onError` will be a required field which allow users to specify whether or not request is denied if the webhook errors out or is unreachable.
This allows `SubjectAccessReviews` to be more definitive.

Today, the `SubjectAccessReview` version defaults to `v1beta1` if the corresponding flag is not supplied.
While configuring authorization modes using the file config, the version supported by a webhook has to be mentioned using a required field `subjectAccessReviewVersion`.

Authentication of webhooks will move to the configuration file and would be mapped as shown in the diagram above.

> Note: ABAC will not be supported by the new file format as we discourage the usage of this mode.

The code path for enabling the above will only be triggered if the feature flag will be enabled until the time the feature flag is removed and configuring authorizer through a file becomes GA.

> TODO: More details on the implementation will be charted once the summary, motivation, goals, non-goal and proposal are approved.

### Monitoring

We will add the following 4 metrics:

1. `apiserver_authorization_step_invocations_total`

This will be incremented on round-trip of an authorizer. It will track total authorization decision invocations across the following labels.

- `mode` {RBAC, Node, Webhook}
- `decision` {Allow, Deny, NoOpinion}

2. `apiserver_authorization_step_webhook_invocations_total`

This will be incremented on round-trip of an authorization webhook. It will track total invocation counts across the following labels.

- `server`
- `code` {2xx, 4xx, 5xx}
- `decision` {Allow, Deny, NoOpinion}
- `subject_access_review_version` {v1beta1, v1}

3. `apiserver_authorization_step_webhook_duration_seconds`

This metric will track the average latency

Labels {along with possible values}:
- `server`
- `code` {2xx, 4xx, 5xx}
- `decision` {Allow, Deny, NoOpinion}
- `subject_access_review_version` {v1beta1, v1}

4. `apiserver_authorization_step_webhook_error_total`

This metric will be incremented when a webhook returns a 4xx or 5xx (erroneous) response.

Labels {along with possible values}:

- server
- `code` {4xx, 5xx}
- `decision` {Deny, NoOpinion}
- `subject_access_review_version` {v1beta1, v1}

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

Unit tests would be added along with any new code introduced.

Existing test coverage:
- cmd/kube-apiserver/app/options: 65
- staging/src/k8s.io/apiserver/pkg/authorization: NA
- staging/src/k8s.io/apiserver/apis/config/v1: 54.5

> Will add more packages to the list during implementation.

##### Integration tests

Integration tests would be added to ensure the following:
- Authorization of requests work in the existing flag based mode (feature flag turned off)
- Authorization of requests work with an apiserver bootstrapped with authorization configuration file (feature flag turned on)
    - without an webhook
    - with an webhook - successful request
    - with an webhook - error on request with `onError: Deny`
    - with an webhook - error on request with `onError: NoOpinion`

##### e2e tests

End-to-end tests won't be needed as unit and integration tests would cover all the scenarios.

### Graduation Criteria

#### Alpha (1.25)

- Add file based authorizer chain configuration
- Add feature flag for gating usage
- Unit tests and Integration tests to be written

#### Beta

- Address user reviews and iterate (if needed keep in Alpha until changes stabilize)
- Feature flag will be turned on by default

#### GA

- Feature flag removed
- Existing command line flags will have no effect

#### GA + 3 cycles

- Remove the existing command line flags

### Upgrade / Downgrade Strategy

While the feature is in Alpha, there is no change if cluster administrators want to keep on using command line flags.

When the feature goes to Beta/GA or the cluster administrators want to configure authorizers using a config file, they need to make sure the config file exists before upgrading the cluster.
Similarly when downgrading clusters, they would need to add the flags back to their bootstrap mechanism.

### Version Skew Strategy

Not applicable.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `AuthorizationConfigFromFile`
  - Components depending on the feature gate:
        - kube-apiserver
###### Does enabling the feature change any default behavior?

Yes, kube-apiserver will not use the `--authorization-*` to configure the authorizer chain.
`kube-apiserver` will need the `--authorization-config-file` flag to be set to a valid config file on disk.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled once enabled. If the `--authorization-*` flags are not removed while enabling the feature, past behavior would be restored.

###### What happens if we reenable the feature if it was previously rolled back?

When the feature is re-enabled, `--authorization-config-file` flag should be present. The behaviour is the same as when the feature is enabled for the first time.

###### Are there any tests for feature enablement/disablement?

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout can fail when the authorization configuration file being passed doesn't exist or is invalid.

Already running workloads won't be impacted but cluster users won't be able to access the control plane if the cluster is single-node.

###### What specific metrics should inform a rollback?

Not applicable.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TBD.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

None until the feature reaches GA and at a future point in time, `--authorization-*` flags other than `--authorization-config-file` are removed.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

> //TODO: To be elaborated more during Beta graduation.

###### How can an operator determine if the feature is in use by workloads?

The cluster administrators can check the flags passed to the `kube-apiserver` if they have access to the control plane nodes. If the `--authorization-config-file` is set to a valid authorization configuration file, the feature is being used.
Or, they can look at the metrics exposed by `kube-apiserver`.

###### How can someone using this feature know that it is working for their instance?

- [x] Other
  - Details: They can look at the metrics if `apiserver_authorization_step_invocations_total` is increasing.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The amount of errors denoted by `apiserver_authorization_step_webhook_error_total` is within reasonable limits.
A rising value indicate issues with either the authorizer chain or the webhook itself.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `apiserver_authorization_step_invocations_total`
  - Components exposing the metric: kube-apiserver

If the cluster administrator has defined an authorizer chain and the above metric doesn't show an increasing trend even if there are requests made to `kube-apiserver` that should be evaluated by the authorizer chain, this will indicate a problem.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

None other than what we are planning to add as part of the feature.

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

None

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Maybe, the only case where API Server requests woud be impacted is when the cluster administrator defines multiple webhooks.

**Note**: This is a result of the intended feature.
If multiple webhooks are defined and one or more of them are unreachable, the request latency would get a hit but this is upto the configuration made by the user.
The feature implementation itself doesn't introduce any change to the existing SLIs/SLOs.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No.

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

No effect.

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

- [x] Provisional KEP introduced
- [ ] KEP Accepted as implementable
- [ ] Implementation started
- [ ] First release (1.YY) when feature available

## Drawbacks

- Having multiple webhooks adds more complexity.

## Alternatives

- Multiple flags to define additional webhooks

## Infrastructure Needed (Optional)

None
