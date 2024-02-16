# KEP-3221: Structured Authorization Configuration

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Protecting installed CRDs](#story-1-protecting-installed-crds)
    - [Story 2: Preventing unnecessarily nested webhooks](#story-2-preventing-unnecessarily-nested-webhooks)
    - [Story 3: Denying requests on certain scenarios](#story-3-denying-requests-on-certain-scenarios)
    - [Story 4: Controlling access of a privileged RBAC role](#story-4-controlling-access-of-a-privileged-rbac-role)
    - [Story 5: Varying defaults across versions of the API](#story-5-varying-defaults-across-versions-of-the-api)
    - [Story 6: Conditionally filtering requests to webhooks](#story-6-conditionally-filtering-requests-to-webhooks)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Monitoring](#monitoring)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (1.29)](#alpha-129)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Today, `kube-apiserver` allows one to configure the authorization chain using
a set of command line flags of the format `--authorization-*`. And cluster
administrators can only enable one webhook as a part of the authorization
chain using the `--authorization-modes` flag. This limits admins from creating
authorization chains with multiple webhooks that validate requests in a certain
order. This proposal makes the case for a more structured config to define the
authorization chain for `kube-apiserver` and allowing multiple webhooks with
well-defined parameters and enabling fine grained control like explicit Deny
authorizer.

## Motivation

Cluster administrators should be able to specify more than one authorization
webhook in the API Server handler chain. They also need to be able to
declaratively configure the authorizer chain using a configuration file. It
should also be easy to say when to deny requests, for example, when a webhook
is unreachable.

### Goals

- Define a configuration file format for configuring Kubernetes API Server
Authorization chain.
- Allow ordered definition of authorization modes.
- Allow definition of multiple webhooks in the authorization chain while all
other types of authorizers should at most be specified once.
- Allow resource/user based pre-filtering of webhooks using CEL to prevent unnecessary
invocations.
- Enable user to define the policy when a webhook can't be reached due to
network errors etc.
- Enable automatic reload of the configuration.

### Non-Goals

- Replace path to kubeconfig file for webhooks by allowing auth configuration
within the new file structure
- Allow ordering of super user authorizer (always allow user group;
`system:masters`) so that it can be moved from the beginning of the auth chain.
- Deciding whether ABAC should be continued to be used by the users.

> Note: The above may be revisited in future alpha iterations of this KEP or as a separate KEP (for e.g., the superuser authorizer)

## Proposal

Add a configuration format having specific precedence order and defined failure modes for configuring authorizer chain. See [Design Details](#design-details) for more details.

```yaml
apiVersion: apiserver.config.k8s.io/v1alpha1
kind: AuthorizationConfiguration
authorizers:
  - name: system-webhook
    type: Webhook
    webhook:
      unauthorizedTTL: 30s
      timeout: 3s
      subjectAccessReviewVersion: v1
      matchConditionSubjectAccessReviewVersion: v1
      failurePolicy: Deny
      connectionInfo:
        type: KubeConfig
        kubeConfigFile: /kube-system-authz-webhook.yaml
      matchConditions:
      # only send resource requests to the webhook
      - expression: has(request.resourceAttributes)
      # only intercept requests to kube-system
      - expression: request.resourceAttributes.namespace == 'kube-system'
      # don't intercept requests from kube-system service accounts
      - expression: !('system:serviceaccounts:kube-system' in request.user.groups)
  - type: Node
  - type: RBAC
  - name: internal
    type: Webhook
    webhook:
      authorizedTTL: 5m
      unauthorizedTTL: 30s
      timeout: 3s
      subjectAccessReviewVersion: v1
      failurePolicy: NoOpinion
      connectionInfo:
        type: InClusterConfig
```

### User Stories

#### Story 1: Protecting installed CRDs

Having certain Custom Resource Definitions available at cluster startup has
been a long requested feature. One of the blockers for having a controller
reconciling those CRDs is to have a protection mechanism for them, which can
be achieved through Authorization Webhooks. However since `kube-apiserver`
only allows specifying a single webhook, this would result in cluster
administrators being not able to specify their own. Support for multiple
webhooks would make this possible. Moreover, RBAC rules can't be used here
since RBAC allows one to add permissions but not deny them. We won't be able
to add restrictions on 'non-system' users only for a set of resources using
RBAC; 'non-system' users here refers to anyone who shouldn't be able to
update/delete the protected set of CRDs.

Relevant Discussion Thread is sig-apimachinery: https://groups.google.com/g/kubernetes-sig-api-machinery/c/MBa19WTETMQ

A relevant configuration for this scenario with the assumptions:
1. The "protected" CRDs are installed in the kube-system namespace.
2. They can only be modified by users in the group "system:serviceaccounts:kube-superuser"

Note: The above are hypothetical for now since there has been no decision on
protection rules for system CRDs. The below example is only for demonstration
purposes.
```yaml
apiVersion: apiserver.config.k8s.io/v1alpha1
kind: AuthorizationConfiguration
authorizers:
  - type: Webhook
    name: system-crd-protector
    webhook:
      unauthorizedTTL: 30s
      timeout: 3s
      subjectAccessReviewVersion: v1
      matchConditionSubjectAccessReviewVersion: v1
      failurePolicy: Deny
      connectionInfo:
        type: KubeConfig
        kubeConfigFile: /kube-system-authz-webhook.yaml
      matchConditions:
      # only send resource requests to the webhook
      - expression: has(request.resourceAttributes)
      # only intercept requests to kube-system (assumption i)
      - expression: request.resourceAttributes.namespace == 'kube-system'
      # don't intercept requests from kube-system service accounts
      - expression: !('system:serviceaccounts:kube-system' in request.user.groups)
      # only intercept update, delete or deletecollection requests
      - expression: request.resourceAttributes.verb in ['update', 'delete','deletecollection']
  - type: Node
  - type: RBAC
```

#### Story 2: Preventing unnecessarily nested webhooks

Consider a system administrator who would like to apply a set of validations
to certain requests before handing it off to webhooks defined using frameworks
like Open Policy Agent.

They would have to run nested webhooks within the one added to the auth chain to
have the intended effect. This enhancement allows the administrator to configure
this behaviour via a structured API and invoke the additional webhook only when
relevant. This also allows administrators to define `failurePolicy` behaviours for
separate webhooks, leading to more predictable outcomes.

The below example is only for demonstration purposes.
```yaml
apiVersion: apiserver.config.k8s.io/v1alpha1
kind: AuthorizationConfiguration
authorizers:
  - name: system-crd-protector
    type: Webhook
    webhook:
      unauthorizedTTL: 30s
      timeout: 3s
      subjectAccessReviewVersion: v1
      matchConditionSubjectAccessReviewVersion: v1
      failurePolicy: Deny
      connectionInfo:
        type: KubeConfig
        kubeConfigFile: /kube-system-authz-webhook.yaml
      matchConditions:
      # only send resource requests to the webhook
      - expression: has(request.resourceAttributes)
      # only intercept requests to kube-system
      - expression: request.resourceAttributes.namespace == 'kube-system'
      # don't intercept requests from kube-system service accounts
      - expression: !('system:serviceaccounts:kube-system' in request.user.groups)
  - type: Node
  - type: RBAC
  - name: opa
    type: Webhook
    webhook:
      unauthorizedTTL: 30s
      timeout: 3s
      subjectAccessReviewVersion: v1
      matchConditionSubjectAccessReviewVersion: v1
      failurePolicy: Deny
      connectionInfo:
        type: KubeConfig
        kubeConfigFile: /opa-kube-system-authz-webhook.yaml
      matchConditions:
      # only send resource requests to the webhook
      - expression: has(request.resourceAttributes)
      # only intercept requests to kube-system
      - expression: request.resourceAttributes.namespace == 'kube-system'
      # don't intercept requests from kube-system service accounts
      - expression: !('system:serviceaccounts:kube-system' in request.user.groups)
```

#### Story 3: Denying requests on certain scenarios

The authorizer chain should be powerful enough to deny anyone making a request
if certain conditions are satisfied, except for the `system:masters` user group.

#### Story 4: Controlling access of a privileged RBAC role

Certain users associated with a privileged role might need to have their access
scoped to certain namespaces. Having ordered authorization modes allows the
administrator to add a webhook restricting certain user tokens before RBAC
grants access to the user.

#### Story 5: Varying defaults across versions of the API

If the default values for configuration evolve over time, affected users might
have to override these values in case they are affected by supplying the flags
with updated defaults. A configuration format like the one proposed can be
versioned over time with changing defaults, mitigating this risk by only
affecting users who knowingly bump the version.

#### Story 6: Conditionally filtering requests to webhooks

Let's say if an API request is being made by an user in a specific group, the
webhook request can be skipped.


### Risks and Mitigations

- In HA clusters, there may be a skew in how the `kube-apiserver` processes in
each are configured. This may create inconsistencies. Mitigation is to have the
cluster administrator handle such scenarios.
- In case an administrator enables this feature and the webhook kubeconfig file
is invalid or doesn't exist at the specified path, `kube-apiserver` on that node
will not be able to start. This can be mitigated by fixing the malformed values.


## Design Details

We would like to introduce a structured file format which allows authorization
to be configured using a flag (`--authorization-config`) which accepts a
path to a file on the disk. Setting both `--authorization-config` and
configuring an authorization webhook using the `--authorization-webhook-*`
command line flags will not be allowed. If the user does that,
there will be an error and API Server would exit right away.

For HA Clusters, the cluster administrator needs to be careful about the
migrating from using the old flags to the config file format. Here is a
proposed way:
1. Translate existing CLI flags to the structured config in each servers. Ensure
that it is exactly the same across servers.
2. Change the flags on kube-apiserver to use the config.
3. Restart on kube-apiserver at a time.
4. Parellely, update the config files to the final desired config. The automatic
reloader would pick up the changes. There would be a minutes order of delay.

The configuration would be validated at startup and the API server will fail to
start if the configuration is invalid.

The API server will periodically reload the configuration at a specific time
interval. If it or any of referenced files change, and the new configuration
passes validation, then the new configuration will be used for the Authorizer
chain. The reloader will also check if the webhook is healthy, thereby
preventing any typo/misconfiguration with the Webhook resulting in bad
Authorizer config. If healthcheck on the webhook failed, the last known good
config will be used. In the next iteration of reload, if webhook is found to be
healthy, the new config will be used. Logging and metrics would be used to
signal success/failure of a config reload so that cluster admins can have
observability over this process. Reload must not add or remove Node or RBAC
authorizers. They can be reordered, but cannot be added or removed.

The proposed structure is illustrated below:

> The sample configuration describes all the fields, their defaults and possible
values.

```yaml
apiVersion: apiserver.config.k8s.io/v1alpha1
kind: AuthorizationConfiguration
authorizers:
  - type: Webhook
    # Name used to describe the authorizer
    # This is explicitly used in monitoring machinery for metrics
    # Note:
    #   - Validation for this field is similar to how K8s labels are validated today.
    # Required, with no default
    name: webhook
    webhook:
      # The duration to cache 'authorized' responses from the webhook
      # authorizer.
      # Same as setting `--authorization-webhook-cache-authorized-ttl` flag
      # Default: 5m0s
      authorizedTTL: 30s
      # The duration to cache 'authorized' responses from the webhook
      # authorizer.
      # Same as setting `--authorization-webhook-cache-unauthorized-ttl` flag
      # Default: 30s
      unauthorizedTTL: 30s
      # Timeout for the webhook request
      # Maximum allowed is 30s.
      # Required, with no default.
      timeout: 3s
      # The API version of the authorization.k8s.io SubjectAccessReview to
      # send to and expect from the webhook.
      # Same as setting `--authorization-webhook-version` flag
      # Required, with no default
      # Valid values: v1beta1, v1
      subjectAccessReviewVersion: v1
      # MatchConditionSubjectAccessReviewVersion specifies the SubjectAccessReview
      # version the CEL expressions are evaluated against
      # Valid values: v1
      # Required, no default value
      matchConditionSubjectAccessReviewVersion: v1
      # Controls the authorization decision when a webhook request fails to
      # complete or returns a malformed response or errors evaluating
      # matchConditions.
      # Valid values:
      #   - NoOpinion: continue to subsequent authorizers to see if one of
      #     them allows the request
      #   - Deny: reject the request without consulting subsequent authorizers
      # Required, with no default.
      failurePolicy: Deny
      connectionInfo:
        # Controls how the webhook should communicate with the server.
        # Valid values:
        # - KubeConfig: use the file specified in kubeConfigFile to locate the
        #   server.
        # - InClusterConfig: use the in-cluster configuration to call the
        #   SubjectAccessReview API hosted by kube-apiserver. This mode is not
        #   allowed for kube-apiserver.
        type: KubeConfig
        # Path to KubeConfigFile for connection info
        # Required, if connectionInfo.Type is KubeConfig
        kubeConfigFile: /kube-system-authz-webhook.yaml
        # matchConditions is a list of conditions that must be met for a request to be sent to this
        # webhook. An empty list of matchConditions matches all requests.
        # There are a maximum of 64 match conditions allowed.
        #
        # The exact matching logic is (in order):
        #   1. If at least one matchCondition evaluates to FALSE, then the webhook is skipped.
        #   2. If ALL matchConditions evaluate to TRUE, then the webhook is called.
        #   3. If at least one matchCondition evaluates to an error (but none are FALSE):
        #      - If failurePolicy=Deny, then the webhook rejects the request
        #      - If failurePolicy=NoOpinion, then the error is ignored and the webhook is skipped
      matchConditions:
      # expression represents the expression which will be evaluated by CEL. Must evaluate to bool.
      # CEL expressions have access to the contents of the SubjectAccessReview in v1 version.
      # If version specified by subjectAccessReviewVersion in the request variable is v1beta1,
      # the contents would be converted to the v1 version before evaluating the CEL expression.
      #
      # Documentation on CEL: https://kubernetes.io/docs/reference/using-api/cel/
      #
      # only send resource requests to the webhook
      - expression: has(request.resourceAttributes)
      # only intercept requests to kube-system
      - expression: request.resourceAttributes.namespace == 'kube-system'
      # don't intercept requests from kube-system service accounts
      - expression: !('system:serviceaccounts:kube-system' in request.user.groups)
  - type: Node
    name: node
  - type: RBAC
    name: rbac
  - type: Webhook
    name: in-cluster-authorizer
    webhook:
      authorizedTTL: 5m
      unauthorizedTTL: 30s
      timeout: 3s
      subjectAccessReviewVersion: v1
      failurePolicy: NoOpinion
      connectionInfo:
        type: InClusterConfig
```

Validation will allow multiple authorizers of type "Webhook" to be added to the
config, but at most one authorizer each for other types. The ordering of this chain will
be decided by the order specified in the file.

The keys `kubeConfigFile`, `authorizedTTL`, `unauthorizedTTL` and
`subjectAccessReviewVersion` accept values corresponding to flags
`--authorization-webhook-config-file`, `--authorization-webhook-cache-authorized-ttl`,
`--authorization-webhook-cache-unauthorized-ttl` and `--authorization-webhook-version`
respectively.

Today, the `SubjectAccessReview` version defaults to `v1beta1` if the corresponding
flag is not supplied. While configuring authorization modes using the file config,
the version supported by a webhook has to be mentioned using a required field
`subjectAccessReviewVersion`.

The user can define a CEL expression to determine whether a request needs to dispatched
to the authz webhook for which the expression has been defined. The user would have access
to a `request` variable containing a `SubjectAccessReview` object in the version specified
by the `matchConditionSubjectAccessReviewVersion` field. If the version specified by
`subjectAccessReviewVersion` in the request variable is `v1beta1`, the contents would be
converted to the version specified in `matchConditionSubjectAccessReviewVersion` before
evaluating the CEL expression.

When no matchConditions are satisfied for a request, the webhook would be skipped. In such
situations, the decision is logged in the audit log with the `authorization.k8s.io/webhook-skipped`
annotation. Benefit of this is that resource and user info will also be logged.

The code path for enabling the above will only be triggered if the feature flag is enabled
while the feature is in alpha and beta.

### Monitoring

We will add the following metrics:

1. `apiserver_authorization_decisions_total`

This will be incremented when an authorizer makes a terminal decision (allow or deny).
It will track total authorization decision invocations across the following labels.

Labels {along with possible values}:
- `type` {<authorizer_type>}
  - value matches the configuration `type` field, e.g. `RBAC`, `ABAC`, `Node`, `Webhook`
- `name` {<authorizer_name>}
  - value matches the configuration `name` field, e.g. `rbac`, `node`, `abac`, `<webhook name>`
- `decision` {allowed, denied}

2. `apiserver_authorization_webhook_evaluations_total`

This will be incremented on round-trip of an authorization webhook.
It will track total invocation counts across the following labels.

- `name` {<authorizer_name>}
  - value matches the configuration `name` field
- `result` {canceled, timeout, error, success}
  - `canceled`: the call invoking the webhook request was canceled
  - `timeout`: the webhook request timed out
  - `error`: the webhook response completed and was invalid
  - `success`: the webhook response completed and was well-formed

3. `apiserver_authorization_webhook_duration_seconds`

This is a Histogram metric that will track the total round trip time of the requests to the webhook.

Labels {along with possible values}:
- `name` {<authorizer_name>}
  - value matches the configuration `name` field
- `result` {canceled, timeout, error, success}
  - `canceled`: the call invoking the webhook request was canceled
  - `timeout`: the webhook request timed out
  - `error`: the webhook response completed and was invalid
  - `success`: the webhook response completed and was well-formed

4. `apiserver_authorization_webhook_evaluations_fail_open_total`

This metric will be incremented when a webhook request times out or
returns an invalid response, and the failurePolicy is set to `NoOpinion`.

Labels {along with possible values}:

- `name` {<authorizer_name>}
  - value matches the configuration `name` field
- `result` {timeout, error}
  - `timeout`: the webhook request timed out
  - `error`: the webhook response completed and was invalid

5. `apiserver_authorization_config_controller_automatic_reload_last_timestamp_seconds`

This Gauge metric will record last time in seconds when an authorization reload was performed, partitioned by apiserver_id_hash and status.
- `apiserver_id_hash`
- `status` (`success` or `failure`)

6. `apiserver_authorization_config_controller_automatic_reloads_total`

This Counter metric records the total number of reload successes and failures, partitioned by API server apiserver_id_hash and status.
- `apiserver_id_hash`
- `status` (`success` or `failure`)

7. `apiserver_authorization_match_condition_evaluation_errors_total`

This will be incremented when an authorization webhook encounters a match condition error.

Labels {along with possible values}:
- `type` {<authorizer_type>}
  - Currently only `Webhook` authorizers support match conditions
- `name` {<authorizer_name>}
  - value matches the configuration `name` field

8. `apiserver_authorization_match_condition_exclusions_total`

This will be incremented when an authorization webhook is skipped because match conditions exclude it.

Labels {along with possible values}:
- `type` {<authorizer_type>}
  - Currently only `Webhook` authorizers support match conditions
- `name` {<authorizer_name>}
  - value matches the configuration `name` field

9. `apiserver_authorization_match_condition_evaluation_seconds`

Authorization match condition evaluation time in seconds.

Labels {along with possible values}:
- `type` {<authorizer_type>}
  - Currently only `Webhook` authorizers support match conditions
- `name` {<authorizer_name>}
  - value matches the configuration `name` field

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

> Note: We will add more packages to the list during implementation.

**Benchmarks**

We should benchmark the cost of some common CEL expressions inside
`matchConditions`. Some examples being:
- scoping to a particular namespace/group/resource
- excluding a particular set of users/groups

##### Integration tests

Integration tests would be added to ensure the following:
- Authorization of requests work in the existing command line flag
based mode (feature flag turned off)
- Authorization of requests work with an apiserver bootstrapped with
authorization configuration file (feature flag turned on)
    - without a webhook
    - with a webhook - successful request
    - with a webhook - error on request with `failurePolicy: Deny`
    - with a webhook - error on request with `failurePolicy: NoOpinion`
- Automatic reload periodically checks for changes in the configuration
and validates the new configuration and ensures the webhook is healthy
before the new config can be used.

There will be a mix and match of various authorization mechanisms to ensure all
desired functionality works.

##### e2e tests

End-to-end tests won't be needed as unit and integration tests would cover all
the scenarios.

### Graduation Criteria

#### Alpha (1.29)

- Add file based authorizer chain configuration
- Add feature flag for gating usage
- Unit tests and Integration tests to be written

#### Beta

- Observability and metrics complete
- File reloading functionality complete
- Address user reviews and iterate (if needed, keep in Alpha until changes stabilize)
- Feature flag will be turned on by default

#### GA

- Feature flag removed

### Upgrade / Downgrade Strategy

There is no change if cluster administrators want to keep on using command line flags.

If the cluster administrators wants to configure authorizers using a config file,
they need to make sure the config file exists before upgrading the cluster.
When downgrading clusters, they would need to switch their invocation back to use flags.

In clusters with multiple API servers, rippling out authorization configuration changes
using a rolling strategy is recommended, verifying the change is effective and functional
on one API server before proceeding to the next API server.

The recommended strategy to switch from command line flags to a config file is to:

1. Switch from command line flags to a config file that expresses an identical configuration
2. Once all servers are successfully operating with the config file, roll out config modifications

### Version Skew Strategy

Not applicable, authorizers are configured per API server.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `StructuredAuthorizationConfiguration`
  - Components depending on the feature gate:
        - kube-apiserver
- [x] Other
  - `kube-apiserver` command-line flag: `--authorization-config`

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, the feature can be disabled once enabled.

###### What happens if we reenable the feature if it was previously rolled back?

When the feature is re-enabled, `--authorization-config` flag should be present.
The behaviour is the same as when the feature is enabled for the first time.

###### Are there any tests for feature enablement/disablement?

We will have extensive unit tests during feature implementation. There would be tests
for the Authorizer chain in both the old and new configuration scenarios.

We will add integration tests to validate the enablement/disablement flow.
  - When `--authorization-config` flags is defined, the feature flag must be turned on (when feature is in Alpha).
  - Setting `--authorization-config` along `--authorization-modes` and `--authorization-webhook-*`
command line flags should return an error.
  - Configuring the authorizer using legacy flags will continue to be allowed

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout can fail when the authorization configuration file being passed doesn't
exist or is invalid.

A rollback can fail if the admins failed to set the existing command line flag
`--authorization-webhook-*`.

Already running workloads won't be impacted but cluster users won't be able to
access the control plane if the cluster is single-node.

###### What specific metrics should inform a rollback?

Not applicable.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No. There is no data stored for this feature which persists between upgrade / downgrade,
or between enable / disable. The feature is purely an API server configuration option.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The cluster administrators can check the flags passed to the `kube-apiserver` if
they have access to the control plane nodes. If the `--authorization-config`
is set to a valid authorization configuration file, the feature is being used.
Or, they can look at the metrics exposed by `kube-apiserver`.

###### How can someone using this feature know that it is working for their instance?

- [x] Other
  - Details: Since this feature introduced the `name` field to the webhook authorizer,
  users can first specify a value in the `name` field of the AuthorizationConfiguration.
  Then look at the `apiserver_authorization_webhook_evaluations_total` metrics to ensure the
  count for the named webhook authorizer is increasing.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The total number of `apiserver_authorization_webhook_evaluations_fail_open_total` metric with failure code
is within reasonable limits. A rising value indicates issues with either the
authorizer chain or the webhook itself.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `apiserver_authorization_decisions_total`
  - Components exposing the metric: kube-apiserver

If the cluster administrator has defined an authorizer chain and the above metric
doesn't show an increasing trend even if there are requests made to `kube-apiserver`
that should be evaluated by the authorizer chain, this will indicate a problem.

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

No. No additional calls will be made to the Kubernetes API Server.

###### Will enabling / using this feature result in introducing new API types?

Yes, but these are types for defining configuration and are not served.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Maybe, the only case where API Server requests would be impacted is when the
cluster administrator defines multiple webhooks.

**Note**: This is a result of the intended feature.
If multiple webhooks are defined and one or more of them are unreachable, the
request latency would get a hit but this is up to the configuration made by the
user. The feature implementation itself doesn't introduce any change to the
existing SLIs/SLOs.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

No. There would be very minimal addition to the memory used by the API Server and
number of log entries written to the disk. During the Alpha implementation,
the small impact will be measured and rationalized to keep the addition
minimal. The addition would be well within the scalability limits and
thresholds.

For use-cases where the CEL filters would pre-filter requests even before the need to
be dispatched to a webhook, there would be a performance improvement due to lower
number of network calls.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature exists only in the API server.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No effect.

###### What are other known failure modes?

- Configuration file cannot be loaded at server start
  - Detection: API server process exits
  - Mitigation: Revert to previous success invocation or configuration
  - Diagnostics: Configuration validation errors are logged at default verbosity.
  - Testing: Configuration file loading and validation is unit tested
- Configuration file cannot be reloaded while server is running
  - Detection: `apiserver_authorization_config_controller_automatic_reload_last_timestamp_seconds` metric
    indicates the `failure` status timestamp is most recent.
  - Mitigation: Revert to previous success invocation or configuration
  - Diagnostics: Configuration validation errors are logged at default verbosity.

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- [x] 2022-06-10 - Provisional KEP introduced
- [x] 2023-05-08 - Provisional KEP re-introduced
- [x] 2023-06-15 - KEP Accepted as implementable
- [x] 2023-07-05 - Implementation started
- [x] 2023-09-27 - Update KEP according to actual state
- [x] 2023-12-15 - First release (1.29) when feature available
- [x] 2024-01-29 - Targeting beta in 1.30

## Drawbacks

- Having multiple webhooks adds more complexity.

## Alternatives

- Multiple flags to define additional webhooks.

## Infrastructure Needed (Optional)

None
