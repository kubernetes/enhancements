# KEP-2579: Pod Security Admission Control

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
  - [Windows Support](#windows-support)
  - [Flexible Extension Support](#flexible-extension-support)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Monitoring](#monitoring)
  - [Audit Annotations](#audit-annotations)
  - [PodSecurityPolicy Migration](#podsecuritypolicy-migration)
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
- [Optional Future Extensions](#optional-future-extensions)
  - [Automated PSP migration tooling](#automated-psp-migration-tooling)
  - [Rollout of baseline-by-default for unlabeled namespaces](#rollout-of-baseline-by-default-for-unlabeled-namespaces)
  - [Custom Warning Messages](#custom-warning-messages)
  - [Windows restricted profile support](#windows-restricted-profile-support)
  - [Offline Policy Checking](#offline-policy-checking)
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
3. Feature parity with PodSecurityPolicy. In particular, support for providing default values or any
   other mutations will not be included.

## Proposal

The three profile levels (privileged, baseline, restricted) of the [Pod Security Standards] will
be hardcoded into the new admission plugin. Changes to the standards will be tied to the Kubernetes
version that they were introduced in.

### API

Policy application is controlled based on labels on the namespace. The following labels are supported:
```
pod-security.kubernetes.io/enforce: <policy level>
pod-security.kubernetes.io/enforce-version: <policy version>
pod-security.kubernetes.io/audit: <policy level>
pod-security.kubernetes.io/audit-version: <policy version>
pod-security.kubernetes.io/warn: <policy level>
pod-security.kubernetes.io/warn-version: <policy version>
```

These labels are considered part of the versioned Kubernetes API ([well-known
labels](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/api/core/v1/well_known_labels.go)),
with their maturity tracking the maturity level of the PodSecurity feature.

**Enforce:** Pods meeting the requirements of the enforced level are allowed. Violations are rejected
in admission.

**Audit:** Pods and [templated pods] meeting the requirements of the audit policy level are ignored.
Violations are recorded in a `pod-security.kubernetes.io/audit-violations: <violation>` [audit
annotation](#audit-annotations) on the audit event for the request. Audit annotations will **not**
be applied to the pod objects themselves, as doing so would violate the non-mutating requirement.

**Warn:** Pods and [templated pods] meeting the requirements of the warn level are ignored.
Violations are returned in a user-facing warning message. Warn & audit modes are independent; if the
functionality of both is desired, then both labels must be set.

[templated pods]: #podtemplate-resources

There are several reasons for controlling the policy directly through namespace labels, rather than
through a separate object:

- Using labels enables various workflows around policy management through kubectl, for example
  issuing queries like `kubectl get namespaces -l
  pod-security.kubernetes.io/enforce-version!=v1.22` to find namespaces where the enforcing
  policy isn't pinned to the most recent version.
- Keeping the options on namespaces allows atomic create-and-set-policy, as opposed to creating a
  namespace and then creating a second object inside the namespace.
- Policies settings are per-namespace singletons, and singleton objects are not well supported in
  Kubernetes.
- Labels are not part of the hardcoded namespace API, making it clear that this is a policy layer on
  top of namespaces, not inherent to namespaces itself.

### Validation

The following restrictions are placed (by the admission plugin) on the policy namespace labels:

1. Unknown labels with the `pod-security.kubernetes.io` prefix are rejected, e.g.
   `pod-security.kubernetes.io/foo-bar`
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
    - Allow pods and resources that were allowed under policy version X running on cluster version X
    - For new fields that are irrelevant, allow all values (e.g. pod overhead fields)
    - For new fields that are relevant to pod security, allow the default (explicit or implicit)
      value OR values allowed by newer profiles (e.g. a less privileged value).
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

Definitions of policy levels for previous versions will be kept in place indefinitely.

### PodTemplate Resources

Audit and Warn modes are also checked on resource types that embed a PodTemplate (enumerated below),
but enforce mode only applies to actual pod resources.

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

When an `enforce` policy (or version) label is added or changed, the admission plugin will test each pod
in the namespace against the new policy. Violations are returned to the user as warnings. These
checks have a timeout of 1 second and a limit of 3,000 pods, and will return a warning in the event
that not every pod was checked. User exemptions are ignored by these checks, but runtime class
exemptions and namespace exemptions still apply when determining whether to check the new `enforce` policy
against existing pods in the namespace. These checks only consider actual Pod resources, not [templated pods].

These checks are also performed when making a dry-run request, which can be an effective way of
checking for breakages before updating a policy, for example:

```
kubectl label --dry-run=server --overwrite ns --all pod-security.kubernetes.io/enforce=baseline
```

Evaluation of pods in a namespace is limited in the following dimensions, and a warning emitted if not all pods are checked:
* max of 3,000 pods ([documented](https://github.com/kubernetes/community/blob/master/sig-scalability/configs-and-limits/thresholds.md)
  scalability limit for per-namespace pod count)
* no more than 1 second or 50% of remaining request deadline (whichever is less).
* benchmarks show checking 3,000 pods takes ~0.01 second running with 100% of a 2.60GHz CPU

If multiple pods have identical warnings, the warnings are aggregated.

If there are multiple pods with an ownerReference pointing to the same controller,
controlled pods after the first one are checked only if sufficient pod count and time remain.
This prioritizes checking unique pods over checking many identical replicas.

### Admission Configuration

A number of options can be statically configured through the [Admission Configuration file][]:

```
apiVersion: apiserver.config.k8s.io/v1
kind: AdmissionConfiguration
plugins:
- name: PodSecurity
  configuration:
    defaults:  # Defaults applied when a mode label is not set.
      enforce:         <default enforce policy level>
      enforce-version: <default enforce policy version>
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

While a more restricted default would be preferable from a security perspective, the initial
priority for this feature is enabling broad adoption across clusters, and a restrictive default
would hinder this goal. See [Rollout of baseline-by-default for unlabeled
namespaces](#rollout-of-baseline-by-default-for-unlabeled-namespaces) for a potential path to a more
restrictive default post-GA.

#### Exemptions

Policy exemptions can be statically configured. Exemptions must be explicitly enumerated, and don’t
support indirection such as label or group selectors. Requests meeting criteria are ignored by the
admission controller (enforce, audit and warn) except for recording an [audit
annotation](#audit-annotations). Exemption dimensions include:

- Usernames: requests from users with an exempt authenticated (or impersonated) username are ignored.
- RuntimeClassNames: pods and [templated pods] with specifying an exempt runtime class name are ignored.
- Namespaces: pods and [templated pods] in an exempt namespace are ignored.

The username exemption is special in that the creating user is not persisted on the pod object, and
the pod may be modified by different non-exempt users in the future. See [Updates](#updates) for
details on how non-exempt updates of a previously exempted pod are handled. Use cases for username
exemptions include:

- Trusted controllers that create pods in tenant namespaces with additional 3rd party enforcement on
  the privileged pods.
- Break-glass operations roles, for example for [debugging workloads in a restricted
  namespace](#ephemeral-containers).

### Risks and Mitigations

**Future proofing:** The policy versioning aspects of this proposal are designed to anticipate
breaking changes to policies, either in light of new threats or new knobs added to pods. However, if
there is a new feature that needs to be restricted but doesn't have a sensible hardcoded
requirement, we would be put in a hard place. Hopefully the adoption of this proposal will
discourage such fields from being added.

**Scope creep:** How do we decide which fields are in-scope for policy restrictions? There are a
number of fields that are relevant to local-DoS prevention that are somewhat ambiguous. To mitigate
this, we are explicitly making the following out-of-scope:
  - Fields that only affect scheduling, and not runtime. E.g. `nodeSelector`, scheduling
    `tolerations`, `topologySpreadConstraints`.
  - Fields that only affect the control plane. E.g. `labels`, `finalizers`, `ownerReferences`.
  - Fields under the `PodStatus`, since users are not expected to have write permissions to the
    `status` subresource.
  - Fields that don't have a reasonable universal (unconfigurable) constraint. E.g. container image,
    resource requests, `runtimeClassName`.

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
extensibility use cases in mind.

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

Ephemeral containers will be subject to the same policy restrictions,
and adding or updating ephemeral containers will require a full policy check.
This means that an existing pod which is not valid according to the current
`enforce` policy will not be permitted to add or modify ephemeral containers.

#### Other Pod Subresources

The policy is not checked for the following Pod subresources:
- attach
- binding
- eviction
- exec
- log
- portforward
- proxy
- status

Although annotations can be updated through the status subresource, the apparmor annotations are
immutable and the seccomp annotations are validated to match the `seccompProfile` field present in the pod spec.

### Pod Security Standards

Policy level definitions are hardcoded and unconfigurable out of the box. However, the [Pod Security
Standards] leave open ended guidance on a few items, so we must make a static decision on how to
handle these elements:

_Note: all baseline policies also apply to restricted._

**HostPorts** - (baseline) HostPorts will be forbidden. This is a more niche feature, and violates
the container-host boundary.

**AppArmor** - (baseline) Allow anything except `unconfined`. Custom AppArmor profiles must be
installed by the cluster admin, and AppArmor fails closed when a profile is not present.

**SELinux** - (baseline) type may only be set to allowlisted values, level may be anything, user &
role must be unset. Spec:

- SELinuxOptions.Type
    - **Restricted Fields:**
        - `spec.securityContext.seLinuxOptions.type`
        - `spec.containers[*].securityContext.seLinuxOptions.type`
        - `spec.initContainers[*].securityContext.seLinuxOptions.type`
    - **Allowed Values:**
        - undefined/empty
        - `container_t`
        - `container_init_t`
        - `container_kvm_t`

- SELinuxOptions.User and SELinuxOptions.Role
    - **Restricted Fields:**
        - `spec.securityContext.seLinuxOptions.user`
        - `spec.containers[*].securityContext.seLinuxOptions.user`
        - `spec.initContainers[*].securityContext.seLinuxOptions.user`
        - `spec.securityContext.seLinuxOptions.role`
        - `spec.containers[*].securityContext.seLinuxOptions.role`
        - `spec.initContainers[*].securityContext.seLinuxOptions.role`
    - **Allowed Values:** undefined/empty

- SELinuxOptions.Level
    - Unrestricted.

**Non-root Groups** - (restricted) This optional constraint will be omitted from the initial
implementation.

**Seccomp**
- (restricted) Must be set to anything except `unconfined`. Same reasoning as AppArmor. If the default
  value changes from `unconfined` (see https://github.com/kubernetes/enhancements/issues/2413), then
  the requirement to set a profile will be lifted.
- (baseline) May be unset; any profile except `unconfined` allowed.

**Capabilities** - (baseline) Only the following capabilities may be added:
- AUDIT_WRITE
- CHOWN
- DAC_OVERRIDE
- FOWNER
- FSETID
- KILL
- MKNOD
- NET_BIND_SERVICE
- SETFCAP
- SETGID
- SETPCAP
- SETUID
- SYS_CHROOT

Notes:
- This set is equal to the [docker default capability
  set](https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities) minus
  `NET_RAW`.
- The [OpenShift default capability
  set](https://github.com/openshift/hypershift-toolkit/blob/148be6f31a365dbcdf1cbd647773f59ccbcc282a/assets/ignition/files/etc/crio/crio.conf#L87-L102)
  also drops AUDIT_WRITE, MKNOD, SETFCAP, and SYS_CHROOT._
- The allowed set for the restricted profile is implicitly empty (no capabilities), since
  [Kubernetes does not support ambient
  capabilities](https://github.com/kubernetes/kubernetes/issues/56374). If ambient capability
  support is ever added, we may want to consider a more conservative allowed set for the restricted
  profile.

**Volumes** - Inline CSI volumes will be allowed by the restricted profile. Justification:
  - Inline CSI volumes should only be used for ephemeral volumes
  - The CSIDriver object spec controls whether a driver can be used inline, and can be modified
    without binary changes to disable inline usage.
  - Risky inline drivers should already use a 3rd party admission controller, since they are usable
    by the baseline policy.
  - We should thoroughly document safe usage, both on the documentation for this (pod security
    admission) feature, as well as in the CSI driver documentation.

**Windows Host Process** - [Privileged Windows container
support](https://github.com/kubernetes/enhancements/tree/master/keps/sig-windows/1981-windows-privileged-container-support)
is targeting alpha in v1.22, and adds new fields for running privileged windows containers. These
fields will be restricted under the baseline policy.
- **Restricted Fields:**
  - `spec.securityContext.windowsOptions.hostProcess`
  - `spec.containers[*].securityContext.windowsOptions.hostProcess`
- **Allowed Values:** false, undefined/nil

_Note: These fields should be unconditionally restricted, regardless of targeted OS._

### Windows Support

The `privileged` and `baseline` levels do not require any OS-specific fields to be set.

The `restricted` level currently requires fields that are Linux-specific, which may prevent
Windows pods from running or require Windows kubelets to ignore those fields.

A mechanism for Windows-specific exemptions or requirements in the `restricted` profile is
described in the ["future work" section](#windows-restricted-profile-support) and addressed by
[KEP-2802](https://github.com/kubernetes/enhancements/issues/2802).

### Flexible Extension Support

In order to make it as easy as possible to extend and customize this policy controller, we will
publish the following tools:

- Library implementation - The admission plugin will be written using reusable library code.
- Webhook implementation - A standalone webhook implementation will be provided (using the same
  library code), enabling policy enforcement on older clusters.
- Thorough testing resources, which can be used to validate 3rd party implementations of the policy
  spec.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

- `k8s.io/pod-security-admission/admission`: `2022-05-12` - `80.7% of statements`
- `k8s.io/pod-security-admission/admission/api`: `2022-05-12` - `1.4% of statements` (mostly boilerplate & generated code)
- `k8s.io/pod-security-admission/admission/api/load`: `2022-05-12` - `88.5% of statements`
- `k8s.io/pod-security-admission/admission/api/scheme`: `2022-05-12` - `100.0% of statements`
- `k8s.io/pod-security-admission/admission/api/v1alpha1`: `2022-05-12` - `1.7% of statements` (generated API)
- `k8s.io/pod-security-admission/admission/api/v1beta1`: `2022-05-12` - `1.7% of statements` (generated API)
- `k8s.io/pod-security-admission/admission/api/validation`: `2022-05-12` - `100.0% of statements`
- `k8s.io/pod-security-admission/api`: `2022-05-12` - `9.3% of statements` **room for improvement**
- `k8s.io/pod-security-admission/cmd/webhook`: `2022-05-12` - `no unit tests` (mostly server setup, covered by integration)
- `k8s.io/pod-security-admission/cmd/webhook/server`: `2022-05-12` - `no unit tests` (mostly server setup, covered by integration)
- `k8s.io/pod-security-admission/cmd/webhook/server/options`: `2022-05-12` - `no unit tests` (mostly server setup, covered by integration)
- `k8s.io/pod-security-admission/metrics`: `2022-05-12` - `93.8% of statements`
- `k8s.io/pod-security-admission/policy`: `2022-05-12` - `88.3% of statements`
- `k8s.io/pod-security-admission/test`: `2022-05-12` - `73.7% of statements`

##### Integration tests

`k8s.io/kubernetes/test/integration/auth/podsecurity_test.go`
https://storage.googleapis.com/k8s-triage/index.html?test=TestPodSecurity

Pod Security admission has very thorough integration test coverage, including:
- Generated test fixtures for failing & passing pods across every type of check, version and level.
- Tests with only GA feature gates enabled, and the default set.
- Tests running as a built-in admission controller & webhook.
- Tests pods run directly & via a controller

##### e2e tests

There are no Pod Security specific E2E tests (we rely on integration test coverage instead), but the
Pod Security admission controller is enabled in E2E clusters, and all E2E test namespaces are
labeled with the enforcement label for Pod Security.

### Monitoring

Three metrics will be introduced:

```
pod_security_evaluations_total
```

This metric will be added to track policy evaluations against pods and [templated pods].
[Namespace evaluations](#namespace-policy-update-warnings) are not counted.
The metric will only be incremented when the policy check is actually performed. In other words,
this metric will not be incremented if any of the following are true:

- Ignored resource types, subresources, or workload resources without a pod template
- Update requests that are out of scope (see [Updates](#updates) above)
- Exempt requests (these are reported in the `pod_security_exemptions_total` metric instead)
- Errors that make policy evaluation impossible (these are reported in the `pod_security_exemptions_total` metric instead)

The metric will use the following labels:

1. `decision {allow, deny}` - The policy decision. `allow` is only recorded with `enforce` mode.
3. `policy_level {privileged, baseline, restricted}` - The policy level that the request was
   evaluated against.
4. `policy_version {v1.X, v1.Y, latest, future}` - The policy version that was used for the evaluation.
   Explicit versions less than or equal to the build of the API server or webhook are recorded in the form `v1.x` (e.g. `v1.22`).
   Explicit versions greater than the build of the API server or webhook (which are evaluated as `latest`) are recorded as `future`.
   Explicit use of the `latest` version or implicit use by omitting a version or specifying an unparseable version will be recorded as `latest`.
5. `mode {enforce, warn, audit}` - The type of evaluation mode being recorded. Note that a single
   request can increment this metric 3 times, once for each mode. `audit` and `warn` mode metrics
   are only incremented for violations. If this admission controller is enabled, every
   evaluated request will at least increment the `enforce` total.
6. `request_operation {create, update}` - The operation of the request being checked.
7. `resource {pod, controller}` - Whether the request object is a Pod, or a [templated
   pod](#podtemplate-resources) resource.
8. `subresource {ephemeralcontainers}` - The subresource, when relevant & in scope.

```
pod_security_exemptions_total
```

This metric will be added to track requests that are considered exempt. Ignored resources and out of
scope requests do not count towards the total. Errors encountered before the exemption logic will
not be counted as exempt.

The metric will use the following labels. The definitions match from the above label definitions.

1. `request_operation {create, update}`
2. `resource {pod, controller}`
3. `subresource {ephemeralcontainers}`

```
pod_security_errors_total
```

This metric will be added to track errors encountered during request evaluation.

The metric will use the following labels. The definitions match from the above label definitions.

1. `fatal {true, false}` - Whether the error prevented evaluation (short-circuit deny). If
   `fatal=false` then the latest restricted profile may be used to evaluate the pod.
2. `request_operation {create, update}`
3. `resource {pod, controller}`
4. `subresource {ephemeralcontainers}`

### Audit Annotations

The following audit annotations will be added:

1. `pod-security.kubernetes.io/enforce-policy = "<policy_level>:<version>"` - Record which policy was evaluated
   for enforcing mode.
    - version is `latest` or a specific version in the form `v1.x`
    - This annotation is only recorded when a policy is enforced. Specifically, it will not be
      recorded for irrelevant updates or exempt requests.
2. `pod-security.kubernetes.io/audit-violations = "<policy violations>"` - When an audit mode policy is violated, record
   the violation messages here.
3. `pod-security.kubernetes.io/exempt = "namespace" | "user" | "runtimeClass"` - For exempt requests, record the parameter
   that triggered the exemption here. If multiple parameters are exempt, the first in this ordered list will be returned:
   - namespace
   - user
   - runtimeClass
4. `pod-security.kubernetes.io/error = "<evaluation errors>"` - Errors evaluating policies are recorded here

Violation messages returned by enforcing policies are included in the `responseStatus` portion of audit events in the `ResponseComplete` stage.

### PodSecurityPolicy Migration

Migrating to the replacement policy from PodSecurityPolicies can be done effectively using a
combination of dry-run and audit/warn modes (although this becomes harder if mutating PSPs are
used).

Publish a step-by-step migration guide. A rough approach might look something like
this, with the items tagged (automated) having support from the PSP migration tool.

1. Enable the `PodSecurity` admission plugin, default everything to privileged.
2. Eliminate mutating PSPs:
    1. Clone all mutating PSPs to a non-mutating version
    2. Update all ClusterRoles authorizing use of the mutating PSPs to also authorize use of the
       non-mutating variant
    3. Watch for pods using the mutating PSPs (check via the `kubernetes.io/psp` annotation), and
       work with code owners to migrate to valid non-mutating resources.
    4. Delete mutating PSPs
3. Select a compatible pod security level for each namespace, based on the existing resources in the namespace
    1. Review the pod security level choices
    2. Evaluate the difference in privileges that would come from disabling the PSP controller.
4. Apply the pod security levels in `warn` and `audit` mode
5. Iterate on Pod and workload configurations until no warnings or audit violations exist
6. Apply the pod security levels in `enforce` mode
7. Disable `PodSecurityPolicy` admission plugin

This was published at https://kubernetes.io/docs/tasks/configure-pod-container/migrate-from-psp/ in 1.22.



### Graduation Criteria

Maturity level of this feature is defined by:
- `PodSecurity` feature gate
- Documented maturity of the feature repo (library & webhook implementations)

#### Alpha

The initial alpha implementation targeting v1.22 includes:

- Initial implementation is protected by the default-disabled feature gate `PodSecurity`
- The built-in PodSecurity admission controller is defalut-disabled.
- Initial set of E2E feature tests implemented and enabled in an alpha test job

#### Beta

We are targeting Beta in v1.23.

1. Resolve the following sections:
    - [x] [Restricted policy support for Windows pods](#windows-restricted-profile-support)
    - [x] [Deprecation / removal policy for old profile versions](#versioning)
    - [x] [Ephemeral containers support](#ephemeral-containers)
    - [x] [PSP migration workflow & support](#podsecuritypolicy-migration)
2. Collect feedback from the alpha, analyze usage of the webhook implementation. In particular, re-assess these API decisions:
    - Distinct `audit` and `warn` modes
    - Whether `enforce` should warn on templated pod resources
3. Thorough testing is already expected for alpha, but we will review our test coverage and fill any
   gaps prior to beta.
   - Feature tests are moved to the main test jobs (may postpone to GA)
4. Admission plugin included in the default enabled set (enforcement is still opt-in per-namespace).

#### GA

Targeting GA in v1.25.

**Conformance:**
- Enabling the admission controller with the "default-default" enforcing mode of privileged is
  essentially a no-op without adding namespace labels, so it doesn't have any impact on
  conformance.
- E2E framework has been updated to explicitly label test namespaces with the appropriate
  enforcement level, using the `NamespacePodSecurityEnforceLevel` framework value. For GA,
  conformance tests should be updated to use the most restrictive level possible.
- Pod Security Admission is *not* required for conformance.

**User Experience Improvements:**
- [Warn when labeling exempt namespaces](https://github.com/kubernetes/kubernetes/issues/109549)
- [Dedupe overlapping forbidden messages](https://github.com/kubernetes/kubernetes/issues/106129)
- [Aggregate identical warnings for multiple pods in a namespace](https://github.com/kubernetes/kubernetes/issues/103213)
- [Add context to failure messages](https://github.com/kubernetes/kubernetes/pull/105314)

**API Changes:**
- No changes to namespace label schema
- Add `pod-security.admission.config.k8s.io/v1` (admission configuration, not a REST API) with no
  changes from the `v1beta1` API.

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
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `PodSecurity`
    - Components depending on the feature gate: PodSecurity admission plugin
  - [x] Other
    - Describe the mechanism:
        - The new functionality is entirely encapsulated by the admission controller, so enabling or
          disabling the feature is a matter of adding or removing the admission plugin from the list
          of enabled admission plugins.
    - Will enabling / disabling the feature require downtime of the control
      plane?
        - Yes. Admission plugins are statically configured, so enabling or disabling will require an
          API server restart. This can be done in a rolling manner for HA clusters.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
        - No. This feature does not touch any node components.

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.
    - No.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).
  - Yes. Disabling it means that the policies will no longer be enforced, but there are no stateful
    changes that will be affected.

* **What happens if we reenable the feature if it was previously rolled back?**
  - There might be pods violating the policy in namespaces when it's turned on. The [Updates
    section](#updates) explains how this case is handled.

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.
  - There will be tests for updates to "violating" pods, but I do not think explicit feature
    enable/disablement tests are necessary.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**

  If `pod-security.kubernetes.io/enforce` labels are already present on namespaces,
  upgrading to enable the feature could prevent new pods violating the opted-into
  policy level from being created. Existing running pods would not be disrupted.

* **What specific metrics should inform a rollback?**

  On a cluster that has not yet opted into enforcement, non-zero counts for either 
  of the following metrics mean the feature is not working as expected:

  * `pod_security_evaluations_total{decision=deny,mode=enforce}`
  * `pod_security_errors_total`

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

  * Manual upgrade of the control plane to a version with the feature enabled was tested.
    Existing pods remained running. Creation of new pods in namespaces that did not opt into enforcement was unaffected.

  * Manual downgrade of the control plane to a version with the feature disabled was tested.
    Existing pods remained running. Creation of new pods in namespaces that had previously opted into enforcement was allowed once more.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?**
  
  No.

### Monitoring Requirements

* **How can an operator determine if the feature is in use by workloads?**
  - non-zero `pod_security_evaluations_total` metrics indicate the feature is in use

* **What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?**
  - [x] Metrics
    - Metric name: `pod_security_evaluations_total`, `pod_security_errors_total`
    - Components exposing the metric: `kube-apiserver`

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  - `pod_security_errors_total`
    - any rising count of these metrics indicates an unexpected problem evaluating the policy
  - `pod_security_errors_total{fatal=true}`
    - any rising count of these metrics indicates an unexpected problem evaluating the policy that
      is preventing pod write requests
  - `pod_security_errors_total{fatal=false}`,
    `pod_security_evaluations_total{decision=deny,mode=enforce,level=restricted,version=latest}`
    - a rising count of non-fatal errors indicates an error resolving namespace policies, which
      causes PodSecurity to default to enforcing `restricted:latest`
    - a corresponding rise in `restricted:latest` denials may indicate that these errors are
      preventing pod write requests
  - `pod_security_evaluations_total{decision=deny,mode=enforce}`
    - a rising count indicates that the policy is preventing pod creation as intended, but is
      preventing a user or controller from successfully writing pods

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  
  - An error rate other than 0 means invalid policy levels or versions were configured 
    on a namespace prior to the feature having been enabled. Until this is corrected, 
    that namespace will use the latest version of the "restricted" policy for the mode 
    that specified an invalid level/version.

* **Are there any missing metrics that would be useful to have to improve observability of this feature?**
  
  - None we are aware of

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**

  * It exists in the kube-apiserver process and makes use of pre-existing
    capabilities (etcd, namespace/pod informers) that are already inherent to the
    operation of the kube-apiserver.

### Scalability

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - Updating namespace enforcement labels will trigger a list of pods in that namespace.
    With the built-in admission plugin, this call will be local within the apiserver and will use the existing pod informer.
    There will be a hard cap on the number of pods analyzed, and a timeout for the review of those pods 
    that ensures evaluation does not exceed a percentage of the time allocated to the request.
    See [Namespace policy update warnings](#namespace-policy-update-warnings).
    - Timeout: minimum of 1 second or (remaining request deadline / 2)
    - Max pods to check: 3000 ([benchmarks](https://github.com/kubernetes/kubernetes/pull/104588) indicate that 3000 pods should evaluate in under 10ms)

* **Will enabling / using this feature result in introducing new API types?**
  - No.

* **Will enabling / using this feature result in any new calls to the cloud provider?**
  - No.

* **Will enabling / using this feature result in increasing size or count of the existing API objects?**
  Describe them, providing:
  - API type(s): Namespaces
  - Estimated increase in size: new labels, up to 300 bytes if all are provided
  - Estimated amount of new objects: 0

* **Will enabling / using this feature result in increasing time taken by any operations covered by [existing SLIs/SLOs]?**
  - This will require negligible additional work in Pod create/update admission.
  - Namespace label updates may heavier, but have limits in place.

* **Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?**
  - No. Resource usage will be negligible.
  - Initial benchmark cost of pod admission to a fully privileged namespace (default on feature enablement without explicit opt-in)
    - Time: 245.4 ns/op
    - Memory: 112 B/op
    - Allocs: 1 allocs/op
  - Initial benchmark cost of pod admission to a namespace requiring both baseline and restricted evaluation
    - Time: 4826 ns/op
    - Memory: 4616 B/op
    - Allocs: 22 allocs/op

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

* **How does this feature react if the API server and/or etcd is unavailable?**

  - It blocks creation/update of Pod objects, which would have been unavailable anyway.

* **What are other known failure modes?**

  - Invalid admission configuration
    - Detection: API server will not start / is unavailable
    - Mitigations: Disable the feature or fix the configuration
    - Diagnostics: API server error log
    - Testing: unit testing on configuration validation

  - Enforce mode rejects pods because invalid level/version defaulted to `restricted` level
    - Detection: rising `pod_security_errors_total{fatal=false}` metric counts
    - Mitigations: fix the malformed labels
    - Diagnostics:
      - Locate audit logs containing `pod-security.kubernetes.io/error` annotations on affected requests
      - Locate namespaces with malformed level labels:
        - `kubectl get ns --show-labels -l "pod-security.kubernetes.io/enforce,pod-security.kubernetes.io/enforce notin (privileged,baseline,restricted)"`
      - Locate namespaces with malformed version labels:
        - `kubectl get ns --show-labels -l pod-security.kubernetes.io/enforce-version | egrep -v 'pod-security.kubernetes.io/enforce-version=v1\.[0-9]+(,|$)'`

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Optional Future Extensions

The following features have been considered for future extensions of the this proposal. They are out
of scope for the initial proposal and/or implementation, but may be implemented in the future (in
which case they should be moved out of this section).

This whole section should be considered <<[UNRESOLVED]>>.

### Automated PSP migration tooling

We could also ship a standalone tool to assist with the steps identified above.
Here are some ideas for the sorts of things the tool could assist with:

- Analyze PSP resources
  - identify mutating PSPs
  - for non-mutating PSPs, identify the closest Pod Security Standards levels and highlight the differences
- Check the authorization mode for existing pods. For example, if a pod’s service account is not
  authorized to use the PSP that validated it (based on the `kubernetes.io/psp` annotation), then
  that should trigger a warning.
- Automate the dry-run and/or labeling process.
- Automatically select (and optionally apply) a policy level for each namespace.

### Rollout of baseline-by-default for unlabeled namespaces

If we wanted to change the default-default value from privileged to baseline, here is a possible
conservative rollout path, potentially with multiple releases between steps. Steps could be dropped
or combined for a more aggressive rollout:

1. Admission plugin goes to GA
2. Admission plugin enabled by default; Unlabeled namespaces treated as: `enforce=privileged`.
   Privileged pod (or PodTemplate controller) creation in an unlabeled namespace triggers the
   namespace to be labeled as privileged (`pod-security.kubernetes.io/enforce: privileged`).
3. Same as (2) but new namespaces are automatically labeled as unprivileged
   (`pod-security.kubernetes.io/enforce: baseline`).
4. Default unlabeled namespaces to enforce=baseline

Each step in the rollout could be overridden with a flag (e.g. force the admission plugin to step N)

### Custom Warning Messages

An optional `pod-security.kubernetes.io/warn-message` annotation can be used to return a custom
warning message (in addition to the standard message) whenever a policy warning is triggered. This
could be useful for announcing the date that new policy will take effect, or providing a point of
contact if you need to request an exception.

### Windows restricted profile support

Even without built-in support, enforcement for Windows pods can be delegated to a webhook admission
plugin by exempting the `windows` RuntimeClass. We should investigate built-in Windows support out
of the box though. Requirements include:

1. A standardized identifier for Windows pods
2. Write the Pod Security Standards for windows

Risk: If a Windows RuntimeClass uses a runtime handler that is also configured on linux nodes, then
a user can just create a linux pod with the Windows RuntimeClass and manually schedule it to a linux
node to bypass the policy checks. For example, this would be the case if the cluster was exclusively
using the dockershim runtime, which requires the hardcoded `docker` runtime handler to be set.

[KEP-2802](https://github.com/kubernetes/enhancements/issues/2802) proposes allowing a Pod to indicate its OS.
As part of that KEP:
* Pod validation will be adjusted to ensure values are not required
  for OS-specific fields that are irrelevant to the Pod's OS.
* Pod Security Standards will be reviewed and updated to indicate which Pod OSes they apply to
* The `restricted` Pod Security Standard will be reviewed to see if there are Windows-specific requirements that should be added
* The PodSecurity admission implementation will be updated to skip checks which do not apply to the Pod's OS.

### Offline Policy Checking

We could provide a standalone tool that is capable of checking the policies against resource files
or through stdin. It should be capable of evaluating `AdmissionReview` resources, but also pod and
templated pod resources. This could be useful in CI/CD pipelines and tests.

### Conformance

Clusters requiring baseline or restricted Pod Security levels should still be able to pass
conformance. This might require
[Conformance Profiles](https://github.com/kubernetes/enhancements/tree/master/keps/sig-architecture/1618-conformance-profiles)
to be feasible.

## Implementation History

- 2021-03-16: [Initial proposal](https://docs.google.com/document/d/1dpfDF3Dk4HhbQe74AyCpzUYMjp4ZhiEgGXSMpVWLlqQ/edit?ts=604b85df#)
              provisionally accepted.
- 2021-08-04: v1.22 Alpha version released
- 2021-08-24: v1.23 Beta KEP updates
- 2021-11-03: v1.23 Beta version released
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
