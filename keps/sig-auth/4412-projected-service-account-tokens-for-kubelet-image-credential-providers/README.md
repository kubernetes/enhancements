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
- [ ] **Fill out as much of the kep.yaml file as you can.
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
# KEP-4412: Projected service account tokens for Kubelet image credential providers

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
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
    - [Credential Provider Configuration](#credential-provider-configuration)
      - [Kubernetes API Server (KAS) Changes](#kubernetes-api-server-kas-changes)
    - [Credential Provider Request API](#credential-provider-request-api)
    - [Caching Credentials](#caching-credentials)
      - [Existing behavior](#existing-behavior)
      - [New behavior when the <code>serviceAccountTokenAudience</code> field is set](#new-behavior-when-the-serviceaccounttokenaudience-field-is-set)
    - [How will this work with the Ensure secret pull images KEP?](#how-will-this-work-with-the-ensure-secret-pull-images-kep)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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
- [Possible future work](#possible-future-work)
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
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

As Kubernetes has matured, we have taken strides towards limiting the need for long
lived credentials that are stored in the Kubernetes API.  The canonical example of this
change is the move of Kubernetes Service Account (KSA) tokens from ones that were
long lived, persisted in the Kubernetes API, and never rotated to tokens that are ephemeral,
short lived, and automatically rotated.  By hardening these tokens and stabilizing
their schema to match the semantics of OIDC ID tokens, we have enabled external consumers
to validate these tokens for use cases beyond Kubernetes.  For example, it is possible to
use a KSA token that is bound to a specific pod to fetch secrets from an external cloud
provider vault without the need for any long lived secrets in the cluster.

This KEP aims to make it possible for a similar 'secret-less' flow to be created for
image pulls, which happen before the pod is running.  Today, admins are limited to image pull
secrets that are stored directly in the Kubernetes API and thus are long lived and hard to rotate,
or secrets that are managed at the Kubelet level via a Kubelet credential provider (meaning
that any pod running on that node can access those images).  A pod should instead be able
to use its own identity to pull images, i.e. we should enable image pull authorization to be
tied to a particular workload while avoiding the need for long lived, persisted secrets.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

By moving to image pulls that are based on KSA tokens, we seek to:

- Reduce secret management overhead for developers and admins
- Reduce security risks associated with long lived secrets

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Allow workloads to pull images based on their own runtime identity without long lived / persisted secrets
- Avoid needing a kubelet/node based identity to pull images

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

We will expand the on-disk kubelet credential provider configuration to allow an
optional `tokenAttributes` field to be configured.  When this field is not set, no KSA
token will be sent to the plugin.  When it is set, the Kubelet will provision
a token with the given audience bound to the current pod and its service
account. This KSA token along with required annotations on the KSA defined in configuration
will be sent to the credential provider plugin via its standard input (along with the image
information that is already sent today). The KSA annotations to be sent will
be configurable in the kubelet credential provider configuration.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a Kubernetes administrator, I need to securely manage image pulls for my multi-tenant
cluster so that I can prevent unauthorized access to private container images and reduce
the security risks associated with static secrets.

#### Story 2

As a developer deploying applications in Kubernetes, I want my pods to use dynamic
service account tokens to pull images securely, ensuring that each pod only accesses images
that it is explicitly permitted to use, thereby maintaining proper segmentation of access
between services.

#### Story 3

As a security engineer, I need to eliminate the use of static secrets for image pulls
and leverage workload identity to enhance security through ephemeral, tightly scoped tokens.
This change reduces the attack surface and the risk of credential leaks, aligning with
best practices for security.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

To keep this KEP small both in terms of implementation and scope,
the design is written to meet the following criteria:

- No changes to any Kubernetes REST APIs
- No changes to the CRI API
- No changes to how the Kubelet interacts with the CRI API
- No changes to any registry
- Minimal changes to Kubelet
- Minimal changes to Kubelet credential providers
- Minimal changes to Kube API Server

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

#### Credential Provider Configuration

Adding a new field `TokenAttributes` to the `CredentialProvider` struct in the kubelet configuration.

1. `ServiceAccountTokenAudience` is the intended audience for the credentials. If set, the kubelet will generate a service account token for the audience and pass it to the plugin.
   1. Only a single audience can be configured.
   2. The credential provider can be set up multiple times for each audience.
      1. The name in the credential provider config must match the name of the provider executable as seen by the kubelet. To configure multiple instances of the same provider with different audiences, the name of the provider executable must be different and this can be done by creating a symlink to the same executable with a different name.
   3. By setting this field, the credential provider is opting into using service account tokens for image pull.
2. `RequireServiceAccount` indicates whether the plugin requires the pod to have a service account. If set to true, kubelet will only invoke the plugin if the pod has a service account. If set to false, kubelet will invoke the plugin even if the pod does not have a service account and will not include a token in the CredentialProviderRequest in that scenario. This is useful for plugins that are used to pull images for pods without service accounts (e.g., static pods).
3. `RequiredServiceAccountAnnotationKeys` is the list of annotation keys that the plugin is interested in and that are required to be present in the service account. The keys defined in this list will be extracted from the corresponding service account and passed to the plugin as part of `CredentialProviderRequest`. If any of the keys defined in this list are not present in the service account, kubelet will not invoke the plugin and will return an error. This field is optional and may be empty. Plugins may use this field to extract additional information required to fetch credentials or allow workloads to opt in to using service account tokens for image pull. If non-empty, `RequireServiceAccount` must be set to true.
4. `OptionalServiceAccountAnnotationKeys` is the list of annotation keys that the plugin is interested in and that are optional to be present in the service account. The keys defined in this list will be extracted from the corresponding service account and passed to the plugin as part of `CredentialProviderRequest`. The plugin is responsible for validating the existence of annotations and their values. This field is optional and may be empty. Plugins may use this field to extract additional information required to fetch credentials.

Regarding required and optional service account annotations keys, there are existing integrations like Azure and GCP that use the extra metadata to link the KSA name to a cloud identity, and this field will allow them to continue to do so. None of the existing integrations use this metadata as any form of security decision, it is simply to aid in doing token exchanges with the KSA token.
    1. Service account annotations are significantly more stable (and map naturally to the service account being the partitioning dimension).
    2. These fields makes the provider opt into getting the annotations it wants, avoids sending arbitrary annotation contents down to the plugin (including stuff like client-side apply annotations) and shrinks the set of annotations that could invalidate the cache.
    3. These fields can't be set without setting the `ServiceAccountTokenAudience` field.

The configuration will not support a plugin to function in 2 modes simultaneously (one with service account tokens and one without). If a plugin needs to function in both modes, it will need to be configured twice with different names. This can be done to facilitate a gradual migration to the new mode for the plugin on a per-image or per-registry basis.

```diff
--- a/pkg/kubelet/apis/config/types.go
+++ b/pkg/kubelet/apis/config/types.go
@@ -653,10 +653,19 @@ type CredentialProvider struct {
        // +optional
        Env []ExecEnvVar
 
+       // tokenAttributes is the configuration for the service account token that will be passed to the plugin.
+       // The credential provider opts in to using service account tokens for image pull by setting this field.
+       // When this field is set, kubelet will generate a service account token bound to the pod for which the
+       // image is being pulled and pass to the plugin as part of CredentialProviderRequest along with other
+       // attributes required by the plugin.
+       //
+       // The service account metadata and token attributes will be used as a dimension to cache
+       // the credentials in kubelet. The cache key is generated by combining the service account metadata
+       // (namespace, name, UID, and annotations key+value for the keys defined in
+       // serviceAccountTokenAttribute.requiredServiceAccountAnnotationKeys and serviceAccountTokenAttribute.optionalServiceAccountAnnotationKeys).
+       // The pod metadata (namespace, name, UID) that are in the service account token are not used as a dimension
+       // to cache the credentials in kubelet. This means workloads that are using the same service account
+       // could end up using the same credentials for image pull. For plugins that don't want this behavior, or
+       // plugins that operate in pass-through mode; i.e., they return the service account token as-is, they
+       // can set the credentialProviderResponse.cacheDuration to 0. This will disable the caching of
+       // credentials in kubelet and the plugin will be invoked for every image pull. This does result in
+       // token generation overhead for every image pull, but it is the only way to ensure that the
+       // credentials are not shared across pods (even if they are using the same service account).
+       // +optional
+       TokenAttributes *ServiceAccountTokenAttributes
+}
+
+// ServiceAccountTokenAttributes is the configuration for the service account token that will be passed to the plugin.
+type ServiceAccountTokenAttributes struct {
+       // serviceAccountTokenAudience is the intended audience for the projected service account token.
+       // +required
+       ServiceAccountTokenAudience string
+
+       // requireServiceAccount indicates whether the plugin requires the pod to have a service account.
+       // If set to true, kubelet will only invoke the plugin if the pod has a service account.
+       // If set to false, kubelet will invoke the plugin even if the pod does not have a service account
+       // and will not include a token in the CredentialProviderRequest in that scenario. This is useful for plugins that
+       // are used to pull images for pods without service accounts (e.g., static pods).
+       // +required
+       RequireServiceAccount *bool
+
+       // requiredServiceAccountAnnotationKeys is the list of annotation keys that the plugin is interested in
+       // and that are required to be present in the service account.
+       // The keys defined in this list will be extracted from the corresponding service account and passed
+       // to the plugin as part of the CredentialProviderRequest. If any of the keys defined in this list
+       // are not present in the service account, kubelet will not invoke the plugin and will return an error.
+       // This field is optional and may be empty. Plugins may use this field to extract
+       // additional information required to fetch credentials or allow workloads to opt in to
+       // using service account tokens for image pull.
+       // If non-empty, requireServiceAccount must be set to true.
+       // +optional
+       RequiredServiceAccountAnnotationKeys []string
+
+       // optionalServiceAccountAnnotationKeys is the list of annotation keys that the plugin is interested in
+       // and that are optional to be present in the service account.
+       // The keys defined in this list will be extracted from the corresponding service account and passed
+       // to the plugin as part of the CredentialProviderRequest. The plugin is responsible for validating
+       // the existence of annotations and their values.
+       // This field is optional and may be empty. Plugins may use this field to extract
+       // additional information required to fetch credentials.
+       // +optional
+       OptionalServiceAccountAnnotationKeys []string
 }

```

Example credential provider configuration:

```yaml
apiVersion: kubelet.config.k8s.io/v1
kind: CredentialProviderConfig
providers:
  - name: acr-credential-provider
    matchImages:
      - "*.registry.io/*"
    defaultCacheDuration: "10m"
    apiVersion: credentialprovider.kubelet.k8s.io/v1
    tokenAttributes:
      serviceAccountTokenAudience: my-audience
      # requireServiceAccount is set to true, so the plugin will only be invoked if the pod has a service account
      requireServiceAccount: true
      # requiredServiceAccountAnnotationKeys is the list of annotations that the plugin is interested in
      # the annotation key and corresponding value in the service account will be passed to the plugin. If
      # any of the keys defined in this list are not present in the service account, kubelet will not invoke
      # the plugin and will return an error.
      requiredServiceAccountAnnotationKeys:
      - domain.io/identity-id
      - domain.io/identity-type
      # optionalServiceAccountAnnotationKeys is the list of annotations that the plugin is interested in
      # the annotation key and corresponding value in the service account will be passed to the plugin. The
      # plugin is responsible for validating the existence of annotations and their values.
      optionalServiceAccountAnnotationKeys:
      - domain.io/some-optional-annotation
      - domain.io/annotation-that-does-not-exist
```

##### Kubernetes API Server (KAS) Changes

Today we only verify the SA in KAS and allow the kubelet to generate a token for any audience. We want to start verifying the audience as well, so we need to be explicit here about what audiences are allowed.
The other sources of audiences for the SA token can be observed via the Kubernetes API but this one can't, so we need a dynamic configuration for it.

KAS will be updated to allow for dynamically configuring service accounts and audiences for which the kubelet is allowed to generate a service account token for as part of the node audience restriction feature. This will be done by a synthetic authz check. The resulting subject access review (SAR) in KAS will look like this:

```yaml
Verb: request-serviceaccounts-token-audience
Resource: $audience
APIGroup: $request.apiGroup (which is the service account API group)
Name: $request.name (which is the service account name)
Namespace: $request.namespace (which is the service account namespace)
```

The node audience restriction will be enforced by the [NodeRestriction admission controller](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#noderestriction)
and kubelet will only be allowed to generate a service account token for the audience configured in the credential provider configuration if it is present in the list of audiences configured in KAS.

- Allow any audience for any service account

```yaml
rules:
- verbs: ["request-serviceaccounts-token-audience"]
  apiGroups: [""]
  # wildcard for audiences
  resources: ["*"]
  # unrestricted resourceNames allows all service account names
```

- Allow any audience for a specific service account `mysa`

```yaml
rules:
- verbs: ["request-serviceaccounts-token-audience"]
  apiGroups: [""]
  # wildcard for audiences
  resources: ["*"]
  resourceNames: ["mysa"]
```

- Allow a specific audience `myaudience` for any service account

```yaml
rules:
- verbs: ["request-serviceaccounts-token-audience"]
  apiGroups: [""]
  resources: ["myaudience"]
  # unrestricted resourceNames allows all service account names
```

- Allow a specific audience `myaudience` for a specific service account `mysa`

```yaml
rules:
- verbs: ["request-serviceaccounts-token-audience"]
  apiGroups: [""]
  resources: ["myaudience"]
  resourceNames: ["mysa"]
```

- Allow API server audience for all service accounts

```yaml
rules:
- verbs: ["request-serviceaccounts-token-audience"]
  apiGroups: [""]
  resources: [""]
  # unrestricted resourceNames allows all service account names
```

#### Credential Provider Request API

Add `ServiceAccountToken` and `ServiceAccountAnnotations` fields to `CredentialProviderRequest`.

1. `ServiceAccountToken` is the service account token bound to the pod for which the image is being pulled.
    If the `ServiceAccountTokenAudience` field is configured in the kubelet's credential provider configuration, the token will be sent to the plugin.
2. `ServiceAccountAnnotations` is a map of annotations on the KSA for which the image is being pulled. Only annotations defined in the `RequiredServiceAccountAnnotationKeys` and `OptionalServiceAccountAnnotationKeys` field in the credential provider configuration will be passed to the plugin. If the `RequiredServiceAccountAnnotationKeys` and `OptionalServiceAccountAnnotationKeys` fields are not set in the configuration, this field will be empty.

```diff
--- a/staging/src/k8s.io/kubelet/pkg/apis/credentialprovider/types.go
+++ b/staging/src/k8s.io/kubelet/pkg/apis/credentialprovider/types.go
@@ -32,6 +32,16 @@ type CredentialProviderRequest struct {
        // credential provider plugin request. Plugins may optionally parse the image
        // to extract any information required to fetch credentials.
        Image string
+
+       // serviceAccountToken is the service account token bound to the pod for which
+       // the image is being pulled. This token is only sent to the plugin if the
+       // tokenAttributes.serviceAccountTokenAudience field is configured in the kubelet's credential provider configuration.
+       ServiceAccountToken string
+
+       // serviceAccountAnnotations is a map of annotations on the service account bound to the
+       // pod for which the image is being pulled. The list of annotations in the service account
+       // that need to be passed to the plugin is configured in the kubelet's credential provider
+       // configuration.
+       ServiceAccountAnnotations map[string]string
 }
```

Example Kubernetes Service Account:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-service-account
  namespace: my-namespace
  annotations:
    domain.io/identity-id: "12345"
    domain.io/identity-type: "user"
    # this annotation will not be passed to the plugin because it's not in tokenAttributes.requiredServiceAccountAnnotationKeys or
    # tokenAttributes.optionalServiceAccountAnnotationKeys field in the credential provider configuration
    domain.io/annotation-that-will-not-be-passed: "value"
```

Example credential provider request:

```json
{
  "image": "my-image",
  "serviceAccountToken": "<service-account-token>",
  "serviceAccountAnnotations": {
    "domain.io/identity-id": "12345",
    "domain.io/identity-type": "user"
    // In the tokenAttributes.optionalServiceAccountAnnotationKeys field in the credential provider configuration above we have configured
    // domain.io/annotation-that-does-not-exist which is not present in the service account annotations. Because of this, this annotation
    // will not be passed to the plugin. The plugin is responsible for validating the existence of annotations and their values that's 
    // defined in the optionalServiceAccountAnnotationKeys field in the credential provider configuration.
  }
}
```

#### Caching Credentials

The cache key generation will be updated to support new caching modes while preserving the existing behavior when the audience field is not set.

##### Existing behavior

If the `serviceAccountTokenAudience` field is not set in the provider configuration for the plugin, the key generation should fall back to the current behavior: using only the image registry URL or other existing identifiers.

##### New behavior when the `serviceAccountTokenAudience` field is set

Based on the `PluginCacheKeyType` definitions (Image, Registry, Global), here’s a breakdown of how cache keys would be generated:

1. `GlobalPluginCacheKeyType` (Global)
   - The cache key will be generated based on the global cache key, service account metadata (namespace, name, UID), and hash of the service account annotations that are passed to the plugin in the `CredentialProviderRequest.ServiceAccountAnnotations` field.
2. `RegistryPluginCacheKeyType` (Registry)
   - The cache key will be generated based on the image registry URL, service account metadata (namespace, name, UID), and hash of the service account annotations that are passed to the plugin in the `CredentialProviderRequest.ServiceAccountAnnotations` field.
3. `ImagePluginCacheKeyType` (Image)
    - The cache key will be generated based on the image URL, service account metadata (namespace, name, UID), and hash of the service account annotations that are passed to the plugin in the `CredentialProviderRequest.ServiceAccountAnnotations` field.

#### How will this work with the Ensure secret pull images KEP?

[2535-ensure-secret-pull-images](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2535-ensure-secret-pulled-images) is a KEP that aims to give admin the ability to ensure pods that use a image are authorized to access the image. The KEP doesn't factor in the use of service account tokens for image pull. As part of the alpha implementation the 2 features together will not provide a desired outcome. Post alpha, we'll need to update the Ensure secret pull images KEP to factor in the use of service account tokens for image pull.

Notes from reviewing the current KEP and discussion with @stlaz:

1. Different KSA for same image should result in image pull from registry.
2. Same KSA for the image will result in allowing pod to use the image.
   1. How would expiry work in this scenario?
      1. I think it'll just be tied to the KSA and not so much to the expiry of the token because we're doing the same thing with image pull secrets (considered valid until deleted and recreated). Deletion and recreation of KSA will result in change in UID and that'll result in KSA not found in cache for the image (assuming the key used to store in cache is consistent with the cache key used in the credential provider cache that takes UID into consideration). Need to share the cache key generation logic to be consistent.
3. Need an update to `ImagePullCredentials` struct to also store coordinates of the KSA.

Until the two implementations are updated to work together, the alpha implementation of this KEP will use the KSA token based flow only when the pod is using image pull policy set to `Always`. This keeps the feature from misbehaving until we fix the implementations.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

- [x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

N/A

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

The unit test coverage is in k8s.io/kubernetes/pkg/credentialprovider package:

- k8s.io/kubernetes/pkg/credentialprovider: 09/11/2024 - 53.7

Unit tests will be added to cover:

1. Config validation
2. Credential provider logic when token attributes are not set
3. Credential provider logic when token attributes are set
4. Cache key generation

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

This kubelet feature is fully tested with unit and e2e tests.

For the node audience restriction changes in KAS, integration tests were added as part of the [implementation in v1.32 release](https://github.com/kubernetes/kubernetes/pull/128077).

- [test/integration/auth/node_test.go](https://github.com/kubernetes/kubernetes/blob/master/test/integration/auth/node_test.go)
- [triage history](https://storage.googleapis.com/k8s-triage/index.html?text=TestNodeRestrictionServiceAccountAudience&test=test%2Fintegration%2Fauth)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

There is an existing e2e test for kubelet credential providers using gcp credential provider.

- test/e2e_node/image_credential_provider.go: https://testgrid.k8s.io/sig-node-kubelet#kubelet-credential-provider

As part of alpha implementation, the [e2e test has been updated](https://github.com/kubernetes/kubernetes/commit/2090a01e0a495301432276216bbf9af102fc431c) to cover the new credential provider configuration and the new behavior of the kubelet when the `TokenAttributes` field is set.

We created a symlink to the existing gcp credential provider executable with a different name to use for testing service account token for credential provider. The credential provider has been updated to validate the following when plugin is run in service account token mode:

1. Check the required annotations are sent as part of the `CredentialProviderRequest.ServiceAccountAnnotations` field.
2. Check the service account token is sent as part of the `CredentialProviderRequest.ServiceAccountToken` field.
3. Extract the claims from the service account token and validate the audience claim matches the `ServiceAccountTokenAudience` field in the kubelet's credential provider configuration.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

#### Alpha

- Feature implemented behind a feature flag
- Unit tests for config validation
- Unit tests for current credential provider logic unchanged when token attributes are not set
- Unit tests for credential provider logic when token attributes are set
- Initial e2e tests completed and enabled
- `ServiceAccountNodeAudienceRestriction` feature gate implemented in KAS as a beta feature
  - Audience validation is enabled by default for service account tokens requested by the kubelet

#### Beta

- Make the feature compatible with the [Ensure secret pull images KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2535-ensure-secret-pulled-images).
- `ServiceAccountNodeAudienceRestriction` feature gate is beta in KAS and enabled by default. This feature needs to be beta/enabled by default at least one release before this KEP goes to beta. This is critical to support downgrade use cases.
- Caching KSA tokens per pod-sa to prevent generating tokens during hot loop/multiple containers with images.
- Some indication of whether the credentials are SA or SA+pod-scoped
  - whether that's indicated in the config or in the plugin-returned content, and what the default is if unspecified (defaulting to pod is less performance, defaulting to SA risks incorrect cross-pod caching)

#### GA

- Gather feedback
  - Cloudsmith has developed a [new credential provider plugin](https://github.com/cloudsmith-io/cloudsmith-kubernetes-credential-provider) that authenticates with cloudsmith registries using service account tokens. [Blog](https://github.com/cloudsmith-io/cloudsmith-kubernetes-credential-provider), [Demo video](https://github.com/cloudsmith-io/cloudsmith-kubernetes-credential-provider), [Feedback on slack](https://kubernetes.slack.com/archives/C0EN96KUY/p1750373833832959).

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

This feature is feature gated so explicit opt-in is required on upgrade and explicit opt-out is required on downgrade.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

To migrate to the new approach,

1. KAS will need to be updated to the new version to configure the audiences for which the kubelet is allowed to generate service account tokens for image pulls via `ClusterRole` or `Role` with the `request-serviceaccounts-token-audience` verb.
1. Kubelet on the node needs to be updated to enable the feature flag and configure the credential provider to use service account tokens for image pull via the `TokenAttributes` field in the kubelet credential provider configuration.
   1. The credential provider plugin will need to be updated to use the service account token for the audience configured in the kubelet credential provider configuration.

Migration of the workloads to the new approach can be done per image or per registry basis by configuring the credential provider multiple times with different names for the same plugin (w or w/o service account tokens, different audiences).

When things can fail:

If the kubelet is updated to enable the feature flag and the credential provider is configured with the `TokenAttributes` field set, but the KAS is not updated to the version that detects dynamic configuration of allowed audiences  via `ClusterRole` or `Role` with the `request-serviceaccounts-token-audience` verb, to allow the kubelet to generate service account tokens for the audience configured in the kubelet credential provider configuration, the image pull will fail. The old KAS will reject the token request from the kubelet.

Today we don't do any validation on the audience value that the kubelet requested -> we only check that the SA is in use on some pod scheduled to the kubelet. As part of this KEP, we're introducing a new feature gate `ServiceAccountNodeAudienceRestriction` in KAS to validate the audience value that the kubelet requests is either part of any API spec or is part of the audiences configured in the `ClusterRole` or `Role` with the `request-serviceaccounts-token-audience` verb. This feature gate will be beta/enabled by default at least one release before this KEP goes to beta.

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

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `KubeletServiceAccountTokenForCredentialProviders`
  - Components depending on the feature gate: kubelet

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ServiceAccountNodeAudienceRestriction`
  - Components depending on the feature gate: kube-apiserver

The purpose of the two feature gates is different, which is why they weren't named similarly.

The `KubeletServiceAccountTokenForCredentialProviders` feature gate is used to enable the kubelet to use service account tokens for image pull in the kubelet credential provider.

The `ServiceAccountNodeAudienceRestriction` feature gate is used to enable the kube-apiserver to validate the audience of the service account token requested by the kubelet. The feature gate in the Kubernetes API Server (KAS) was introduced to strictly enforce which audiences the kubelet can request tokens for. Before this change, the kubelet could request a token with any audience. With the feature gate enabled, the API server starts validating the requested audience.

The KAS feature gate doesn't need to be enabled for the kubelet feature to work. It graduated to beta in v1.32 and is enabled by default. The two are unrelated in functionality, but the KAS gate was necessary to ensure strict enforcement of the allowed audiences the kubelet can request tokens for.

If the KAS feature gate is not enabled, there will be no validation of the audience requested by the kubelet, and the kubelet will be able to request tokens for any audience. This is not recommended.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No. After enabling the feature flag in the kubelet, the credential provider still needs to opt into using service account tokens for image pull
by setting the `TokenAttributes` field in the kubelet credential provider configuration.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes. The feature flag needs to be disabled and the credential provider configuration for the provider that is using service account tokens for image pull
needs to be updated to not use the `TokenAttributes` field or the provider needs to be removed.

Kubelet needs to be restarted to invalidate the in-memory cache after removing the provider or updating the configuration.

Steps to disable the feature:

1. Update the kubelet credential provider configuration to remove providers that are using service account tokens for image pull.
2. Disable the feature flag in the kubelet.
3. Restart the kubelet.

These steps need to be performed on all nodes in the cluster.
After restarting the kubelet on all nodes, remove the allowed audiences for which the kubelet is allowed to generate service account tokens for image pulls in KAS by
removing the previous `ClusterRole` or `Role` with the `request-serviceaccounts-token-audience` verb, along with the corresponding `ClusterRoleBinding` or `RoleBinding` that binds the role to the kubelet.

###### What happens if we reenable the feature if it was previously rolled back?

No impact. The credential provider will continue to use the configuration set in the kubelet credential provider configuration.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

Feature enablement/disablement unit/integration tests will be added.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

Feature is enabled but exec plugin does not properly fetch and return credentials to the kubelet.
Impact is that kubelet cannot authenticate and pull credentials from those registries.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

High error rates from `kubelet_credential_provider_plugin_error` and long durations from `kubelet_credential_provider_plugin_duration`.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

No, upgrade->downgrade->upgrade were not tested. Manual validation will be done prior to promoting this feature to beta in v1.34.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

New metrics:

- `kubelet_credential_provider_config_hash` indicates the hash of the kubelet credential provider configuration file. This metric can be used by operators to determine if the kubelet credential provider configuration has changed.

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

Operators can use `kubelet_credential_provider_config_hash` metric to determine if the kubelet credential provider configuration has changed. If the hash of the configuration file changes, it indicates that the kubelet credential provider configuration has been updated.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

Users can observe events for successful image pulls that use the service account token for image pull.

- [x] Events
  - Event Reason: " Successfully pulled image "xxx" in 11.877s (11.877s including waiting). Image size: xxx bytes."

For registries or images configured to be pulled using a credential provider with a service account, a successful image pull seems to be the only way to confirm that it's working. If the credential provider is misbehaving, the kubelet will not be able to authenticate to the registry and pull images, which will result in image pull errors.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

On failure to fetch credentials from an exec plugin, the kubelet will retry after some period and invoke the plugin again.
The kubelet will retry whenever it attempts to pull an image, but until then, kubelet will not be able to authenticate to
the registry and pull images. The SLO for successfully invoking exec plugins should be based on the SLO for successfully
pulling images for the container registry in question.

The SLOs defined in [Pod startup latency SLI/SLO details](https://github.com/kubernetes/community/blob/master/sig-scalability/slos/pod_startup_latency.md) 
don't apply to this feature because image pull SLI is explicitly excluded from the pod startup latency SLI/SLO. However, if the kubelet is unable to
pull images due to misconfiguration of the credential provider plugin, it will result in pod startup failures.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Metrics
  - Metric name: `kubelet_credential_provider_plugin_errors` indicates number of errors from the credential provider plugin
  - Metric name: `kubelet_credential_provider_plugin_duration` indicates the duration of the execution in seconds for credential provider plugin
  - Components exposing the metric: kubelet

The metrics above will indicate if the credential provider plugin configured to use service account tokens for image pull is functioning properly.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

TBA.

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

This feature depends on the existence of a credential provider plugin binary on the host and a configuration file for the plugin to be read by the kubelet.

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

Yes. Kubelet will request a KSA token from KAS for pod + KSA for every provider (audience) that is configured to use service account tokens for image pull.
The load is small because the token generation happens only once during pod startup.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

This depends on the credential provider plugin implementation on how it fetches the credentials using the service account token.
The plugin will now be called for every unique KSA + (image/registry) combination that is being pulled if not already cached unlike
today where the cache is based on the image/registry URL. This could result in more calls to the cloud provider from the plugin to
exchange the KSA token for the credentials.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

SLIs/SLOs for image pull times might be impacted with the credential plugin using KSA token for fetching credentials. The credentials are
granular in scope (per service account) like image pull secrets, but the plugin must dynamically fetch the credentials using the KSA token
and these credentials could have shorter TTLs resulting in more frequent fetching of credentials.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

As part of the `ServiceAccountNodeAudienceRestriction` feature, KAS will need to watch PersistentVolumeClaims, PersistentVolumes and CSIDrivers
to determine the audiences that the kubelet is allowed to generate service account tokens for. These new informers (which are feature gated) will
result in additional resource usage in the KAS.

- Node authorizer is already watching persistent volumes via informers today.
- CSIDriver objects are expected to be ~few and ~slow-moving, so the impact is expected to be minimal.
- PersistentVolumeClaims are expected to be more numerous and more dynamic, so there could be more impact here.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
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

If the API server is unavailable, kubelet will not be able to fetch service account tokens for image pull. The kubelet will retry fetching the token after some period, but until then, kubelet will not be able to authenticate to the registry and pull images that rely on the credential provider plugin using service account tokens for image pull.

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

- check logs of kubelet
- check service availability of container registries used by the cluster

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

1.33: Alpha release
1.34: Beta release

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Possible future work

Passthrough mode where KSA tokens are sent directly to the CRI. This would allow the CRI to use the KSA token to pull images directly from the registry.

For example, openshift uses `imagePullSecrets` with the PSAT to pull images directly from the registry. If we support passthrough mode with this feature, kubelet can pass the PSAT in the docker config credentials (username: "<some static value>", password: "<PSAT>") to the CRI. The CRI can then use the PSAT to pull images directly from the registry. The kubelet can determine which registries it's ok to pass the PSAT to the CRI.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

always passthrough SA token to registry and rely on registry proxies to do sts

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

N/A
