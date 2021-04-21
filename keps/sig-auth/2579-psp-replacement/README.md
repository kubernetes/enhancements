# KEP-2579: PSP Replacement Policy (placeholder)

<<[UNRESOLVED]>>

**BLOCKING**

The name of this feature / policy is open to discussion. Options considered:
- Pod Isolation Policy
- Pod Policy Check
- Pod Security Standards
- ~~Namsepace Security Policy~~ (there is a lot more to namespace security that is out-of-scope for this policy)
- ~~Pod Configuration Constraints~~ (not all of pod configuration is in-scope)
- Pod Security Constraints
- Pod Isolation Constraints
- Pod Isolation Requirements
- Namespace Security Labels
- ~~Pod Security Defaults~~ (policy is non-mutating)
- Pod Security Policy v2

<<[/UNRESOLVED]>>

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
    - [Requirements](#requirements)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API](#api)
  - [Validation](#validation)
  - [Versioning](#versioning)
  - [PodTemplate Resources](#podtemplate-resources)
  - [Namespace policy update warnings](#namespace-policy-update-warnings)
  - [Admission Configuration](#admission-configuration)
    - [Defaulting](#defaulting)
    - [Exemptions](#exemptions)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Updates](#updates)
    - [Ephemeral Containers](#ephemeral-containers)
    - [Other Pod Subresources](#other-pod-subresources)
  - [Pod Security Standards](#pod-security-standards)
  - [Flexible Extension Support](#flexible-extension-support)
  - [Test Plan](#test-plan)
  - [Monitoring](#monitoring)
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
- [Optional Future Extensions](#optional-future-extensions)
  - [Rollout of baseline-by-default for unlabeled namespaces](#rollout-of-baseline-by-default-for-unlabeled-namespaces)
  - [PodSecurityPolicy Migration](#podsecuritypolicy-migration)
  - [Custom Profiles](#custom-profiles)
  - [Custom Warning Messages](#custom-warning-messages)
  - [Windows Support](#windows-support)
  - [Offline Policy Checking](#offline-policy-checking)
  - [Event recording](#event-recording)
  - [Conformance](#conformance)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Summary

Replace PodSecurityPolicy with a new built-in admission controller that enforces the
[Pod Security Standards].

- Policy enforcement is controlled at the namespace level through labels
- Policies can be applied in 3 modes. Multiple modes can apply to a single namespace.
    - Enforcing: policy violations cause the pod to be rejected
    - Audit: policy violations trigger an audit annotation, but are otherwise allowed
    - Warning: policy violations trigger a user-facing warning, but are otherwise allowed
- An optional per-mode version label can be used to pin the policy to the version that shipped
  with a given Kubernetes minor version (e.g. `v1.18`)
- Dry-run of namespace updates is supported to test enforcing policy changes against existing pods.
- Policy exemptions can be statically configured based on (requesting) user, RuntimeClass, or
  namespace. A request meeting exemption criteria is ignored by the admission plugin.

[Pod Security Standards]: https://kubernetes.io/docs/concepts/security/pod-security-standards/

## Motivation

Pod Security Policy is deprecated as of Kubernetes v1.21. There were numerous problems with
PSP that lead to the decision to deprecate it, rather than promote it to GA, including:

1. Policy authorization model - Policies are bound to the requesting user OR the pod’s service
   account
     - Granting permission to the user is intuitive, but breaks controllers
     - Dual model weakens security
2. Rollout challenges - PSP fails closed in the absence of policy
     - The feature can never be enabled by default
     - Need 100% coverage before rolling out (and no dry-run / audit mode)
     - Leads to insufficient test coverage
3. Inconsistent & unbounded API - Can lead to confusion and errors, and highlights lack of
   flexibility
     - API has grown organically and has many internal inconsistencies (usability challenge)
     - Unclear how to decide what should be part of PSP (e.g. fine-grained volume restrictions)
     - Doesn’t compose well
     - Mutation priority can be unexpected

However, we still feel that Kubernetes should include a mechanism to prevent privilege escalation
through the create-pod permission.

### Goals

Replace PodSecurityPolicy without compromising the ability for Kubernetes to limit privilege
escalation out of the box. Specifically, there should be a built-in way to limit create/update pod
permissions so they are not equivalent to root-on-node (or cluster).

#### Requirements

1. Validating only (i.e. no changing pods to make them comply with policy)
2. Safe to enable in new AND upgraded clusters
    - Dryrun policy changes and/or Audit-only mode
3. Built-in in-tree controller
4. Capable of supporting Windows in the future, if not in the initial release
    - Don’t automatically break windows pods
5. Must be responsive to Pod API evolution across versions
6. (fuzzy) Easy to use, don’t need to be a kubernetes/security/linux expert to meet the basic objective
7. Extensible: should work with custom policy implementations without whole-sale replacement
    - Enable custom policies keyed off of RuntimeClassNames

Nice to have:

1. Exceptions or policy bindings by requesting user
2. (fuzzy) Windows support in the initial release
3. Admission controller is enabled by default in beta or GA phase
4. Provide an easy migration path from PodSecurityPolicy
5. Enforcement on pod-controller resources (i.e. things embedding a PodTemplate)

### Non-Goals

1. Provide a configurable policy mechanism that meets the needs of all use-cases
2. Limit privilege escalation and other attacks beyond the pod to host boundary
   (e.g. policies on services, secrets, etc.)


## Proposal

The three profile levels (privileged, baseline, restricted) of the [Pod Security Standards] will
be hardcoded into the new admission plugin. Changes to the standards will be tied to the Kubernetes
version that they were introduced in.

### API

<<[UNRESOLVED]>>

_BLOCKING: depends on the name we choose._

The annotation prefix is a placeholder, and will depend on the final feature name. The proposed
prefix structure is `<featurename>.kubernetes.io`.

<<[/UNRESOLVED]>>

Policy application is controlled based on labels on the namespace. The following labels are supported:
```
<prefix>/allow: <policy level>
<prefix>/allow-version: <policy version>
<prefix>/audit: <policy level>
<prefix>/audit-version: <policy version>
<prefix>/warn: <policy level>
<prefix>/warn-version: <policy version>
```

**Allow:** Pods meeting the requirements of the allowed level are allowed. Violations are rejected
in admission.

**Audit:** Pods and [templated pods] meeting the requirements of the audit policy level are ignored.
Violations are recorded in a `<prefix>/audit: <violation>` annotation on the
audit event for the request. Audit annotations will **not** be applied to the pod objects
themselves, as doing so would violate the non-mutating requirement.

**Warn:** Pods and [templated pods] meeting the requirements of the warn level are ignored.
Violations are returned in a user-facing warning message. Warn & audit modes are independent; if the
functionality of both is desired, then both labels must be set.

[templated pods]: #podtemplate-resources

<<[UNRESOLVED]>>

**BLOCKING**

The mode key names are open to discussion.
@tallclair is not satisfied that `allow` and `audit/warn` carry slightly different meanings:

- `allow` means "allow pods meeting this profile level & below" (i.e. deny above)
- `audit` and `warn` mean "audit/warn on pods exceeding this profile level" (i.e. audit/warn above)

~~Using `deny` for the enforcing level would be more consistent with audit/warn (deny above), but is
a counter-intuitive user experience. One would expect `deny = privileged` to deny privileged pods,
but it really means allow privileged pods (hence using `allow` for the key).~~ (Rejected)

Another option is to change the `audit` and `warn` keys to something meaning "don't audit/warn at this
level and bellow", but we haven't thought of a good name. The best alternative we've come up with is
`audit-above` and `warn-above`, but that is cumbersome.

A final option is to change `allow` to `enforce`, which is more ambiguous. The ambiguity makes the
inconsistency with audit & warn less obvious, but doesn't help the user experience.

<<[/UNRESOLVED]>>

There are several reasons for controlling the policy directly through namespace labels, rather than
through a separate object:

- Using labels enables various workflows around policy management through kubectl, for example
  issuing queries like `kubectl get namespaces -l
  <prefix>/allow-version!=v1.22` to find namespaces where the enforcing
  policy isn't pinned to the most recent version.
- Keeping the options on namespaces allows atomic create-and-set-policy, as opposed to creating a
  namespace and then creating a second object inside the namespace.
- Policies settings are per-namespace singletons, and singleton objects are not well supported in
  Kubernetes.
- Labels are lighter-weight than namespace fields, making it clear that this is a policy layer on
  top of namespaces, not inherent to namespaces itself.

### Validation

The following restrictions are placed (by the admission plugin) on the policy namespace labels:

1. Unknown labels with the pod security prefix (TBD) are rejected, e.g. `<prefix>/foo-bar`
2. Policy level must be one of: `privileged`, `baseline`, `restricted`
3. Version values must be match `(latest|v[0-9]+\.[0-9]+`. That is, one of:
    1. `latest`
    2. `vMAJOR.MINOR` (e.g. `v1.21`)

Enforcement is best effort, and invalid labels that pre-existed the admission controller enablement
are ignored. Updates to a previously invalid label are only allowed if the new value is valid.

### Versioning

A specific version can be supplied for each enforcement mode. The version pins the policy to the
version that was defined at that kubernetes version. The default version is `latest`, which can be
provided to explicitly use the latest definition of the policy. There is some nuance to how versions
other than latest are applied:

- If the constrained pod field has not changed since the pinned version, the policy is applied as
  originally specified.
- The privileged profile always means fully unconstrained and is effectively unversioned (specifying
  a version is allowed but ignored).
- Specifying a version more recent than the current Kubernetes version is allowed (for rollback &
  version skew reasons), but is treated as `latest`.
- Under an older version X of a policy:
    - Allow pods that were allowed under policy version X running on cluster version X
    - Allow pods that set new fields that the policy level has no opinion about (e.g. pod overhead
      fields)
    - Allow pods that set new fields that the policy level has an opinion about if the value is the
      default (explicit or implicit) value OR the value is allowed by newer versions of the policy
      level (e.g. a less privileged value of a new field)
- Under the webhook implementation, policy versions are tied to the webhook version, not the cluster
  version. This means that it is recommended for the webhook to updated prior to updating the
  cluster. Note that policies are not guaranteed to be backwards compatible, and a newer restricted
  policy could require setting a field that doesn't exist in the current API version.

For example, the restricted policy level now requires `allowPrivilegeEscalation=false`, but this
field wasn't added until Kubernetes v1.8, and all containers prior to v1.8 implicitly ran as
`allowPrivilegeEscalation=true`. Under the **restricted v1.7** profile, the following
`allowPrivilegeEscalation` configurations would be allowed on a v1.8 cluster:
- `null` (allowed during the v1.7 release)
- `true` (equal in privilege to a v1.7 pod that didn't set the field)
- `false` (strictly less privileged than other allowed values)

<<[UNRESOLVED]>>

_Blocking for Beta._

How long will old profiles be kept for? What is the removal policy?

<<[/UNRESOLVED]>>

### PodTemplate Resources

Audit and Warn modes are also checked on resource types that embed a PodTemplate (enumerated below),
but Allow mode only applies to actual pod resources.

Since users do not create pods directly in the typical deployment model, the warning mechanism is
only effective if it can also warn on templated pod resources. Similarly, for audit it is useful to
tie the audited violation back to the requesting user, so audit will also apply to templated pod
resources. In the interest of supporting mutating admission controllers, policies will only
be enforced on actual pods.

Templated pod resources include:

- v1 ReplicationController
- v1 PodTemplate
- apps/v1 ReplicaSet
- apps/v1 Deployment
- apps/v1 StatefulSet
- apps/v1 DaemonSet
- batch/v1 CronJob
- batch/v1 Job

PodTemplate warnings & audit will only be applied to built-in types. CRDs that wish to take
advantage of this functionality can use an object reference to a v1/PodTemplate resource rather than
inlining a PodTemplate. We will publish a guide (documentation and/or examples) that demonstrate
this pattern. Alternatively, the functionality can be implemented in a 3rd party admission plugin
leveraging the library implementation.

### Namespace policy update warnings

When an allow policy (or version) label is added or changed, the admission plugin will test each pod
in the namespace against the new policy. Violations are returned to the user as warnings. These
checks have a timeout of XX seconds and a limit of YY pods, and will return a warning in the event
that not every pod was checked. User exemptions are ignored by these checks, but runtime class
exemptions still apply. Namespace exemptions are also ignored, but an additional warning will be
returned when updating the policy on an exempt namespace. These checks only consider actual Pod
resources, not [templated pods].

These checks are also performed when making a dry-run request, which can be an effective way of
checking for breakages before updating a policy, for example:

```
kubectl label --dry-run=server --overwrite ns --all <prefix>/allow=baseline
```

<<[UNRESOLVED]>>

_Non-blocking: can be decided on the implementing PR_

- What should the timeout be for pod update warnings?
- What should the pod limit be set to?

<<[/UNRESOLVED]>>


### Admission Configuration

A number of options can be statically configured through the [Admission Configuration file][]:

```
apiVersion: apiserver.config.k8s.io/v1
kind: AdmissionConfiguration
plugins:
- name: PSPReplacement # Placeholder
  configuration:
    defaults:  # Defaults applied when a mode label is not set.
      allow:         <default allow policy level>
      allow-version: <default allow policy version>
      audit:         <default audit policy level>
      audit-version: <default audit policy version>
      warn:          <default warn policy level>
      warn-version:  <default warn policy version>
    exemptions:
      usernames:         [ <array of authenticated usernames to exempt> ]
      runtimeClassNames: [ <array of runtime class names to exempt> ]
      namespaces:        [ <array of namespaces to exempt> ]
...
```

[Admission Configuration file]: https://github.com/kubernetes/kubernetes/blob/3d6026499b674020b4f8eec11f0b8a860a330d8a/staging/src/k8s.io/apiserver/pkg/apis/apiserver/v1/types.go#L27

#### Defaulting

The default policy level and version for each mode (when no label is present) can be statically
configured. The default for the static configuration is `privileged` and `latest`.

#### Exemptions

Policy exemptions can be statically configured. Exemptions must be explicitly enumerated, and don’t
support indirection such as label or group selectors. Requests meeting criteria are completely by
the admission controller (allow, audit and warn). Exemption dimensions include:

- Usernames: requests from users with an exempt authenticated (or impersonated) username are ignored.
- RuntimeClassNames: pods and [templated pods] with specifying an exempt runtime class name are ignored.
- Namespaces: pods and [templated pods] in an exempt namespace are ignored.

### Risks and Mitigations

**Future proofing:** The policy versioning aspects of this proposal are designed to anticipate
breaking changes to policies, either in light of new threats or new knobs added to pods. However, if
there is a new feature that needs to be restricted but doesn't have a sensible hardcoded
requirement, we would be put in a hard place. Hopefully the adoption of this proposal will
discourage such fields from being added.

**Scope creep:** How do we decide which fields are in-scope for policy restrictions? There are a
number of fields that are relevant to local-DoS prevention that are somewhat ambiguous.

**Ecosystem suppression:** In the discussions of PodSecurityPolicy replacements, there was a concern
that whatever we picked would become not only an ecosystem standard best-practice, but practically a
requirement (e.g. for compliance), and that in doing so we would prevent 3rd party policy
controllers from innovating and getting meaningful adoption. To mitigate these concerns, we have
tightly scoped this proposal, and hopefully struck an effective balance between configurability and
usefulness that still leaves plenty of room for 3rd party extensions. We are also providing a
library implementation and spec to encourage custom controller development.

**Exemptions creep:** We have already gotten requests to trigger exemptions on various different
fields. The more exemption knobs we add, the harder it becomes to comprehend the current state of
the policy, and the more room there is for error. To prevent this, we should be very conservative
about adding new exemption dimensions. The existing knobs were carefully chosen with specific
extensibility usecases in mind.

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

### Updates

Updates to the following pod fields are exempt from policy checks, meaning that if a pod update
request only changes these fields it will not be denied even if the pod is in violation of the
current policy level:

- Any metadata updates _EXCEPT_ changes to the seccomp or apparmor annotations:
    - `seccomp.security.alpha.kubernetes.io/pod` (deprecated)
    - `container.seccomp.security.alpha.kubernetes.io/*` (deprecated)
    - `container.apparmor.security.beta.kubernetes.io/*`
- Valid updates to `.spec.activeDeadlineSeconds`
- Valid updates to `.spec.tolerations`
- Valid updates to [Pod resources](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/1287-in-place-update-pod-resources)

Note that updates to container images WILL require a policy reevaluation.

Pod status & nodeName updates are handled by `status` and `binding` subresource requests
respectively, and are not checked against policies.

Update requests to Pods and [PodTemplate resources](#podtemplate-resources) will reevaluate the full object
against audit & warn policies, independent of which fields are being modified.

#### Ephemeral Containers

In the initial implementation, ephemeral containers will be subject to the same policy restrictions,
and adding or updating ephemeral containers will require a full policy check.

<<[UNRESOLVED]>>

_Non-blocking for alpha. This should be resolved for beta._

Once ephemeral containers allow [custom security contexts], it may be desirable to run an ephemeral
container with higher privileges for debugging purposes. For example, CAP_SYS_PTRACE is forbidden by
the baseline policy but can be useful in debugging. We could introduce yet-another-mode-label that
only applies enforcement to ephemeral containers (defaults to the allow policy).

[custom security contexts]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/277-ephemeral-containers#configurable-security-policy

One way this could be handled under the current model is:
1. Exempt a special username (not one that can be authenticated directly) from policy enforcement,
   e.g. `ops:privileged-debugger`
2. Grant the special user permission to ONLY operate on the ephemeral containers subresource (it is
   critical that they cannot create or update pods directly).
3. Grant (real) users that should have privileged debug capability the ability to impersonate the
   exempt user.

We could consider ways to streamline the user experience of this, for instance adding a special RBAC
binding that exempts users when operating on the ephemeral containers subresource (e.g. an
`escalate-privilege` verb on the ephemeral containers subresource).

<<[/UNRESOLVED]>>

#### Other Pod Subresources

Aside from ephemeral containers, the policy is not checked for any other Pod subresources (status,
bind, logs, exec, attach, port-forward).

Although annotations can be updated through the status subresource, the apparmor annotations are
immutable and the seccomp annotations are deprecated and slated for removal in v1.23.

### Pod Security Standards

Policy level definitions are hardcoded and unconfigurable out of the box. However, the [Pod Security
Standards] leave open ended guidance on a few items, so we must make a static decision on how to
handle these elements:

**HostPorts** - (baseline) HostPorts will be forbidden. This is a more niche feature, and violates
the container-host boundary.

**AppArmor** - (baseline) Allow anything except `unconfined`. Custom AppArmor profiles must be
installed by the cluster admin, and AppArmor fails closed when a profile is not present.

<<[UNRESOLVED]>>
**BLOCKING**

**SELinux** - (baseline)
- Unset `SELinuxOptions` is allowed.
- If options are set, they must match the following:
  - `type` is one of:
    - `container_init_t`
    - `container_kvm_t`
    - `container_logreader_t`
    - `container_t`
  - `level` can be anything
  - `user` is unset
  - `role` is unset

<<[/UNRESOLVED]>>

**Non-root Groups** - (restricted) This optional constraint will be omitted from the initial
implementation.

**Seccomp** - (restricted) Allow anything except `unconfined`. Same reasoning as AppArmor.

<<[UNRESOLVED]>>

**BLOCKING**

**Capabilities** - (baseline) Adding additional capabilities beyond the default set must be
disallowed.

Kubernetes does not define the default set, but Docker does. This leaves us with 2 options:

1. Hardcode the [docker default capability
   set](https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities), and
   allow adding a capability in that set. This enables workloads to use an allowlist approach where
   they drop all capabilities (`*`), and only explicitly add the ones they need. However, it means
   that operators cannot effectively customize the default set (e.g. drop CAP_NET_RAW) through
   runtime parameters.
2. Forbid any use of adding capabilities. This allows runtimes to customize the default set, but
   gives less flexibility to the pod definition.

@tallclair prefers option 2

<<[/UNRESOLVED]>>

### Flexible Extension Support

In order to make it as easy as possible to extend and customize this policy controller, we will
publish the following tools:

- Library implementation - The admission plugin will be written using reusable library code.
- Webhook implementation - A standalone webhook implementation will be provided (using the same
  library code), enabling policy enforcement on older clusters.
- Thorough testing resources, which can be used to validate 3rd party implementations of the policy
  spec.

### Test Plan

The admission controller can safely be enabled as a no-op with the default-defaults, i.e. everything
is privileged. This will let us run the admission controller in our standard E2E test jobs, by
relabeling specific test namespaces.

**E2E Tests:** The following tests should be added:

1. Allow mode tests:
    - Test all profile levels
    - Test profile version support
2. Warning mode tests:
    - Profile levels & version support
3. Namespace policy relabeling
    - Ensure labeling completes even when there are warnings
    - Test warning on violating pods
    - Test dry-run mode

Additionally, we should add tests to the upgrade test suite to ensure that version skew is properly
handled:

- A minimally specified pod (just a container image) should always be allowed by the baseline
  policy.
- A privileged pod should never be allowed by baseline or restricted
- A Fully specified pod within the bounds of baseline should be allowed by baseline, and rejected by
  restricted.
- A minimally specified restricted pod should be allowed at a pinned version.

**Integration Tests:** Audit mode tests should be added to integration testing, where we have
existing audit logging tests.

**Manual Testing Resources:** Pod resources will be provided covering all dimensions of the baseline
& restricted profiles, for validation of 3rd party policy implementations. These have been drafted
by @JimBugwadia: https://github.com/JimBugwadia/pod-security-tests

**Unit Tests:** Both the library and admission controller implementations will have thorough
coverage of unit tests.

### Monitoring

<<[UNRESOLVED]>>

**BLOCKING**

The following metrics could be useful:

1. Policy decision - record the policy decision for each mode, for each pod create/update request.
   Potential label dimensions:
    - decision (allow, deny, exempt)
    - policy mode (enforcing, warning, audit)
    - policy level (privileged, baseline, restricted)
    - policy version (latest, current, past, future), optionally replace current & past with the
      specific version
    - resource type (Pod, Deployment, ReplicaSet, etc.) - for audit & warnings applied to templated
      pods
    - request type (create, update)
2. Exemptions - record whenever a request is exempt (should this only increment when it would
   otherwise have been denied?)
    - exempt dimension (user, namespace, runtimeclass)
3. (maybe) Violated control - record the specific policy check that failed.
    - Potentially all the dimensions from the policy decision metric
    - control aspect (host network, privileged, capabilities, etc.)

<<[/UNRESOLVED]>>

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

Covered under [Versioning](#versioning) and [Test Plan](#test-plan).

### Version Skew Strategy

Covered under [Versioning](#versioning) and [Test Plan](#test-plan).

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md.

The production readiness review questionnaire must be completed for features in
v1.19 or later, but is non-blocking at this time. That is, approval is not
required in order to be in the release.

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

_This section must be completed when targeting alpha to a release._

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
  Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

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

## Optional Future Extensions

The following features have been considered for future extensions of the this proposal. They are out
of scope for the initial proposal and/or implementation, but may be implemented in the future (in
which case they should be moved out of this section).

This whole section should be considered <<[UNRESOLVED]>>.

### Rollout of baseline-by-default for unlabeled namespaces

If we wanted to change the default-default value from privileged to baseline, here is a possible
conservative rollout path, potentially with multiple releases between steps. Steps could be dropped
or combined for a more aggressive rollout:

1. Admission plugin goes to GA
2. Admission plugin enabled by default; Unlabeled namespaces treated as allow=privileged,
   audit=baseline, warn=privileged; Privileged pod (or PodTemplate controller) creation in an
   unlabeled namespace triggers the namespace to be labeled as privileged
3. Same as (2) but new namespaces are automatically labeled as unprivileged
   (`security.kubernetes.io/privileged: false`), warn=baseline
4. Default unlabeled namespaces to allow=baseline

Each step in the rollout could be overridden with a flag (e.g. force the admission plugin to step N)

### PodSecurityPolicy Migration

Migrating to the replacement policy from PodSecurityPolicies can be done effectively using a
combination of dry-run and audit/warn modes (although this becomes harder if mutating PSPs are
used).

We could also ship a standalone tool to assist with the PodSecurityPolicy migration. Here are some
ideas for the sorts of things the tool could assist with:

- Analyze PSP resources, identify the closest profile level, and highlight the differences
- Check the authorization mode for existing pods. For example, if a pod’s service account is not
  authorized to use the PSP that validated it (based on the `kubernetes.io/psp` annotation), then
  that should trigger a warning.
- Automate the dry-run and/or labeling process.
- Automatically select (and optionally apply) a policy level for each namespace.

We should also publish a step-by-step migration guide. A rough approach might look something like
this, with the items tagged (automated) having support from the PSP migration tool.

1. Enable 3-tier policy admission plugin, default everything to privileged.
2. Eliminate mutating PSPs:
    1. Clone all mutating PSPs to a non-mutating version (automated)
    2. Update all ClusterRoles authorizing use of the mutating PSPs to also authorize use of the
       non-mutating variant (automated)
    3. Watch for pods using the mutating PSPs (check via the `kubernetes.io/psp` annotation), and
       work with code owners to migrate to valid non-mutating resources.
    4. Delete mutating PSPs
3. Select a compatible profile for each namespace, based on the existing resources in the namespace
   (automated)
    1. Review the profile choices
    2. Evaluate the difference in privileges that would come from disabling the PSP controller
       (automated).
4. (optional) Apply the profiles in `warn` and `audit` mode (automated)
5. Apply the profiles in `allow` mode (automated)
6. Disable PodSecurityPolicy

### Custom Profiles

Allow custom profile levels to be statically configured. E.g.
`--extra-pod-security-levels=host-network`. Custom profiles are ignored by the built-in admission
plugin, and must be handled completely by a 3rd party webhook (including the dry-run implementation,
if desired).

### Custom Warning Messages

An optional `prefix/warn-message` annotation can be used to return a custom warning message (in
addition to the standard message) whenever a policy warning is triggered. This could be useful for
announcing the date that new policy will take effect, or providing a point of contact if you need to
request an exception.

### Windows Support

Even without built-in support, enforcement for Windows pods can be delegated to a webhook admission
plugin by exempting the `windows` RuntimeClass. We should investigate built-in Windows support out
of the box though. Requirements include:

1. A standardized identifier for Windows pods
2. Write the Pod Security Standards for windows

Risk: If a Windows RuntimeClass uses a runtime handler that is also configured on linux nodes, then
a user can just create a linux pod with the Windows RuntimeClass and manually schedule it to a linux
node to bypass the policy checks. For example, this would be the case if the cluster was exclusively
using the dockershim runtime, which requires the hardcoded `docker` runtime handler to be set.

### Offline Policy Checking

We could provide a standalone tool that is capable of checking the policies against resource files
or through stdin. It should be capable of evaluating `AdmissionReview` resources, but also pod and
templated pod resources. This could be useful in CI/CD pipelines and tests.

### Event recording

Allow recording an event in response to a pod creation attempt that exceeds a given level.

### Conformance

As this feature progresses towards GA, we should think more about how it interacts with conformance.

- Enabling the admission controller with the "default-default" enforcing mode of privileged is
  essentially a no-op without adding namespace labels, so it shouldn't have any impact on
  conformance.
- If we want a more restricted version to still be considered conformant, we might need to
  explicitly label namespaces in the conformance tests with the privilege level the tests require.

## Implementation History

- 2021-03-16: [Initial proposal](https://docs.google.com/document/d/1dpfDF3Dk4HhbQe74AyCpzUYMjp4ZhiEgGXSMpVWLlqQ/edit?ts=604b85df#)
              provisionally accepted.
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

See [Risks and Mitigations](#risks-and-mitigations)

## Alternatives

We have had extensive discussions of alternatives, several of which are captured in the following
documents:

- [Future of PodSecurityPolicy](https://docs.google.com/document/d/1VKqjUlpU888OYtIrBwidL43FOLhbmOD5tesYwmjzO4E/edit)
- [Bare Minimum Pod Security](https://docs.google.com/document/d/10dXwQ7hnf3-3uLqUuXuXGBo8z7eRM9ZjlTeS1y_7Bak/edit)
- [PSP++ Pre-KEP Proposal](https://docs.google.com/document/d/1F7flSlNTTb7YrzHof-n2JjXbRaOhbJTzvHjlhKTYaD4/edit)

## Infrastructure Needed (Optional)

We will need a new repo to host the code listed under [Flexible Extension
Support](#flexible-extension-support). This will need to be a staged under
https://github.com/kubernetes/kubernetes/tree/master/staging/src/k8s.io, since the library code will
be linked in-tree and needs to track the current API.
