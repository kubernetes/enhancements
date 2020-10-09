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
# KEP-1900: Add additional validation to volume snapshot objects

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
  - [Background on Admission webhooks](#background-on-admission-webhooks)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Validating Scenarios](#validating-scenarios)
    - [VolumeSnapshot](#volumesnapshot)
    - [VolumeSnapshotContent](#volumesnapshotcontent)
  - [Authentication](#authentication)
  - [Timeout](#timeout)
  - [Idempotency/Deadlock](#idempotencydeadlock)
  - [Automatic Labelling of Invalid Objects](#automatic-labelling-of-invalid-objects)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Backwards compatibility](#backwards-compatibility)
    - [Rollback](#rollback)
    - [Current Controller validation of OneOf semantic](#current-controller-validation-of-oneof-semantic)
      - [Handling VolumeSnapshot.](#handling-volumesnapshot)
      - [Handling VolumeSnapshotContent](#handling-volumesnapshotcontent)
- [Design Details](#design-details)
  - [Deployment](#deployment)
  - [Kubernetes API Server Configuration](#kubernetes-api-server-configuration)
  - [Webhook Server Deployment](#webhook-server-deployment)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
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

Tighten validation on `VolumeSnapshot` and `VolumeSnapshotContent` by updating the CRD validation schema and providing a webhook server to enforce immutability.

This KEP will list the new validation rules. It will also provide the release plan to ensure backwards compatibility. As well, it will outline the deployment plan of the webhook server. The webhook server is deployed separately from the snapshot controller.

This tightening of the validation on volume snapshot objects is considered a change to the volume snapshot API. Choosing not to install the webhook server and participate in the 2-phase release process can cause future problems when upgrading from v1beta1 to V1 volumesnapshot API if there are currently persisted objects which fail the new stricter validation. Potential impacts include being unable to delete invalid snapshot objects. It should be possible to downgrade the CRD definition as a workaround.

## Motivation

VolumeSnapshot feature has been on the BETA stage in Kubernetes OSS community since Kubernetes version 1.17. The community has identified a gap in lacking validation when CR(custom resource), i.e, VolumeSnapshot, VolumeSnapshotContent, are created [issue](https://github.com/kubernetes-csi/external-snapshotter/issues/187). This gap will need to be resolved before the feature can be brought to GA.

### Background on Admission webhooks

Admission webhooks are HTTP callbacks to intercept requests to the API server. They could be validating webhooks and mutating webhooks(details). Admission webhooks have been released in BETA since K8s v1.9 and GA in v1.16. Following prerequisites are needed to be able to use this feature:

- K8s version, v1.9+ to use admissionregistration.k8s.io/v1beta1 or v1.16+ to use admissionregistration.k8s.io/v1 (Note that volume snapshot moved to BETA in v1.17)
- Corresponding admission controllers(MutatingAdmissionWebhook, ValidatingAdmissionWebhook) is enabled. (in v1.18+, both will be enabled by default, with mutating precedes validating)
- API admissionregistration.k8s.io/v1beta1 or admissionregistration.k8s.io/v1 is enabled.(Prefer v1 over v1beta1)

Admission controllers have been in common use in kubernetes for a long time. Admission webhooks are the new, preferred way to control admission, especially for external (out-of-tree) components like the CSI external snapshotter.

A webhook server receives [AdmissionReview](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#request) requests from API server, and responses with a response of the same type to either admit/deny the request. Following simplified diagram shows the workflow.
(Note that the mutating webhooks will be invoked BEFORE validating).

![Webhook workflow diagram](./webhook-workflow.png)

The webhook server will expose an HTTP endpoint such that to allow the API server to send AdmissionReview requests. Webhook server providers can [dynamically](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#configure-admission-webhooks-on-the-fly) configure what type of resources and what type of admission webhooks via creating CRs of type ValidatingWebhookConfiguration and/or MutatingWebhookConfiguration.

CRD validation is preferred over webhook validation due to their lower complexity, however CRD validation schema is unable to enforce immutability or provide ratcheting validation.

### Goals

- Provide an updated CRD schema to validate fields
- Prevent:
  - Invalid VolumeSnapshot/VolumeSnapshotContent from creation and update
  - Invalid updates on immutable fields, i.e., VolumeSnapshot.Spec.Source
- Provide a pre-built image which can be used to deploy the webhook server
- Provide a way to deploy the webhook server in cluster
- Provide a way to authenticate the webhook server to the API server via TLS
- Provide a release process to safely tighten the validation and move towards the ideal state of using builtin CRD validation while maintaining backwards compatibility

### Non-Goals

- Provide a way to authenticate the API server to the webhook server

## Proposal

Tighten the validation on Volume Snapshot objects. Please see the tables below for detailed information.

Due to backwards compatibility concerns, the tightening will occur in three phases.

1. The first phase is webhook-only, and will use [ratcheting validation](#backwards-compatibility). It will be the user's responsibility to clean up invalid objects which already existed before the webhook was enabled. Invalid objects are those which fail the new, stricter validation. The controller will not be able to automatically fix invalid objects, however it will apply a [label](#automatic-labelling-of-invalid-objects) to invalid objects so that users can easily locate them.
2. The second phase can occur once all invalid objects are cleared from the cluster. It will be the cluster admin's responsibility to check and detect when it is safe to move to the second phase. The CRD schema validation will be tightened and the webhook will stick around to enforce immutability until immutable fields come to CRDs (Custom Resource Definition). This will be accompanied by a version change (from `v1beta1` to `v1`) to make it clear the CRD is using different validation. however the storage version will be kept as `v1beta1` to ensure a [rollback](#rollback) is possible at phase 2.
3. The storage version of the CRD will be changed from `v1beta1` to `v1`

The phases come in separate releases to allow users / cluster admin the opportunity to clean their cluster of any invalid objects. More details are in the Risks and Mitigations section.

The webhook server will perform validation on Volume Snapshot objects when CREATE and UPDATE requests are made to the api server for `VolumeSnapshot` and `VolumeSnapshotContent` objects. The webhooks will only use validating webhooks, which are read-only.  An image will be built and example `Deployment` and `Service` yaml files will be provided. Example configuration files for the `ValidatingWebhookConfiguration` will be provided, to be used to register the webhooks on the API server.

The webhook will be developed inside the [external-snapshotter](https://github.com/kubernetes-csi/external-snapshotter) repository.

### Validating Scenarios

The following is a list of fields which will get checked when a CREATE or UPDATE operation is sent to the API server. Some validation is already enforced by the CRD schema definition, for example some required fields and enums.

All of the validation desired can be achieved by updating the CRDs to take advantage of the OpenApi v3 schema validation. In particular, the `oneOf` and `minLength` fields can be used.

There is a desire for some fields to be immutable, which is not yet supported by CRDs. See the immutable fields [KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/20190603-immutable-fields.md) for the latest updates. As of August 2020, the KEP is provisional and has no clear timeline for when immutable fields will come to CRDs.

#### VolumeSnapshot

| Operation |            Field             |                                                                                                                                               Reason                                                                                                                                                | HTTP RCode |
| :-------: | :--------------------------: | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------: | :--------: |
|  CREATE   |         spec.Source          |                                                                                       Exactly one of PersistentVolumeClaimName (Dynamic) or VolumeSnapshotContentName (Pre-provisioned) should be specified.                                                                                        |    400     |
|  UPDATE   |         spec.Source          |                             Immutable, no updates allowed. If the user has specified an incorrect source, they must delete and remake the snapshot. The webhook validation server will not be able to guarantee that only incorrect sources are allowed to be updated.                              |    400     |
|  CREATE   | spec.VolumeSnapshotClassName | Must not be the empty string. Can be unset (to use the default snapshot class, if it is set. If the default snapshot class is not set or there is more than 1 default class, then the hook will allow the creation but the snapshot will fail.), or set to a non-empty string (the snapshot class). |    400     |
|  UPDATE   | spec.VolumeSnapshotClassName |                                    Same restrictions as CREATE. We won’t restrict updating by making this field immutable (only applying the same restrictions as creation) but this field should only be changed by those who know exactly what they are doing.                                    |    400     |

#### VolumeSnapshotContent
| Operation |         Field          |                                                                                                                                                                                                                                                            Reason                                                                                                                                                                                                                                                             | HTTP RCode |
| :-------: | :--------------------: | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------: | :--------: |
|  CREATE   |      spec.Source       |                                                                                                                                                                                Exactly one of VolumeHandle (dynamic snapshot created by controller) or SnapshotHandle (pre-provisioned snapshot created by cluster admin) should be specified                                                                                                                                                                                 |    400     |
|  UPDATE   |      spec.Source       |                                                                                                                                                                                                                                                Immutable, no updates allowed.                                                                                                                                                                                                                                                 |    400     |
|  CREATE   | spec.VolumeSnapshotRef | Must have both name and namespace fields set. Preprovisioned: This is the reference to the yet to be created VolumeSnapshot object which should bind to this VolumeSnapshotContent. https://github.com/kubernetes-csi/external-snapshotter/blob/097b1fc7d7cd6576182ca34512c14de1c84b2127/pkg/apis/volumesnapshot/v1beta1/types.go#L270. Dynamic: This is the reference to the VolumeSnapshot object which triggered the creation of this VolumeSnapshotContent. It also has the UID field, but this is set by the controller. |    400     |
|  UPDATE   | spec.VolumeSnapshotRef |                                                                                                                                                                                                                                  Immutable, no updates allowed, once it's UID has been set.                                                                                                                                                                                                                                   |    400     |

### Authentication

There are two directions to authentication. Authenticating the identity of the webhook server, and authenticating the identitiy of the kubernetes api server.

The API server authenticates the webhook server through TLS certificates and HTTPS. This is required, and an example method of deploying the webhook server with HTTPS will be provided.

Authentication on incoming requests to the webhook server is configurable however out of scope of this document. It’s the user’s responsibility in general to configure the webhook service and the API server if authentication is required ([details](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#authenticate-apiservers)). The web server implementation, however, should allow users to configure whether authentication is required or not. If no authentication config is specified, the webhook server should default to “NoClientCert”, which effectively will not authenticate the identity of the clients.

### Timeout

Webhooks add latency to each API server call configured in the ValidationWebhookConfig/MutatingWebhookConfig, for this KEP it should only affect `CREATE` and `UPDATE` requests on snapshot resources. Thus setting up a reasonable timeout for each AdmissionReview request from the webhook server side is critical. The default timeout is 10 seconds if not specified. When an AdmissionReview request sent to the webhook server timed out, `failurePolicy`(default to `Fail` which is equivalent to disallow) will be triggered.

In the ValidatingWebhookConfiguration yaml [example](#kubernetes-api-server-configuration), a default timeout of two seconds is provided, cluster admins who wish to change the timeout may change the value of `timeoutSeconds`.

To avoid migration pain it is recommended to start with a `failurePolicy` value of `Ignore`, changing it to `Fail` only after the webhook is confirmed to have been installed successfully. Choosing `Ignore` means that it would be possible invalid objects can get created/updated in the system.

### Idempotency/Deadlock

Since only validating webhooks will be introduced in this version, idempotency/deadlock are not relevant.

### Automatic Labelling of Invalid Objects

The controller will apply a label called `snapshot.storage.sigs.k8s.io/invalid-snapshot-resource` to `VolumeSnapshot` and `snapshot.storage.sigs.k8s.io/invalid-snapshot-content-resource` to `VolumeSnapshotContent` objects which fail strict validation. For valid objects the label will not be present, and for invalid objects it will be present. The value of the label does not matter, and is set to the empty string by default. The controller will use the same validation logic in the webhook.

For example here's the yaml for an invalid `VolumeSnapshot`:

```yaml
apiVersion: snapshot.storage.k8s.io/v1beta1
kind: VolumeSnapshot
metadata:
  name: snapshot-label-example
  labels:
    snapshot.storage.kubernetes.io/invalid-snapshot-resource: "" # Label applied for invalid VolumeSnapshot objects
...
```

Here's an example for the yaml for an invalid `VolumeSnapshotContent`:
```yaml
apiVersion: snapshot.storage.k8s.io/v1beta1
kind: VolumeSnapshotContent
metadata:
  name: snapcontent-72d9a349-aacd-42d2-a240-d775650d2455
  labels:
    snapshot.storage.kubernetes.io/invalid-snapshot-content-resource: "" # Label applied for invalid VolumeSnapshotContent objects
...
```

Users and cluster admins MUST ensure there are NO objects with the labels applied before upgrading to phase 2. The labels are added by the controller, and there may be a delay after deployment. It is recommended to wait 48 hours after installing the webhook and new controller, as the controller does a full resync of each snapshot resource every 24 hours.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1
Cluster admin can deploy the webhook server.
Users can create and update snapshot objects with confidence invalid updates will be rejected.

Following are some typical scenarios we are aiming to prevent:
- Creation of invalid CRs
  - Reject if a VolumeSnapshot CR does not have a legit VolumeSnapshotSource, i.e., missing both PersistentVolumeName and VolumeSnapshotContentName.
  - Reject if a VolumeSnapshotContent CR does not have a legit VolumeSnapshotContentSource, i.e., both VolumeHandle and SnapshotHandle have been specified
- Updating immutable fields
  - Reject updates to VolumeSnapshot’s VolumeSnapshotSource
  - Reject updates to VolumeSnapshotContent’s VolumeSnapshotContentSource
  - Reject updates to VolumeSnapshotContent’s volume snapshot ref after binding

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

#### Backwards compatibility

There is a backwards compatibility issue involved when tightening the validation on snapshot objects. Since the feature is already in beta we are committed to more backward compatibility guarantee than alpha.

Backward compatibility violations which are okay:
- create: users can no longer create objects which fail strict validation
- update: users can no longer update objects to fail strict validation

Backwards compatibility violations which are not okay:
- delete: users can no longer delete objects which fail strict validation

If the validation tightening would prevent a no-op update, it would prevent deletion of that object because deletions require removing finalizers, which are done via update. Therefore we must support a no-op update, where previously invalid objects can be updated without changes. Making a previously optional field required in the schema blocks update of previously persisted data that omitted the field (unless the update populates the newly required field, or you specify a schema default)

To tackle the backwards compatibility problem, this KEP proposes the following release process.

Begin with validating webhook only enforcement. The webhook will perform the following validation
- one release with ratcheting validation using the webhook server
   - webhook is strict on create
   - webhook is strict on updates where the existing object passes strict validation
   - webhook is relaxed on updates where the existing object fails strict validation (allows finalizer removal, status update, deletion, etc)
 - The user will need to delete or fix all invalid objects. The webhook and controllers will not take any automatic action to reconcile invalid objects. However, the controller will add a label.

For `UPDATE` operations, the webhook server will receive the existing object and the new, proposed object. We will use this feature to check when the existing objects passes or fails strict validation.

Once we are sure no invalid data is persisted, we can switch to CRD schema-enforced validation with validating webhooks for immutability in a subsequent release.

#### Rollback 

If users do not completely remove their invalid objects before upgrading their CRD definition, it should be possible to downgrade the CRD definition to allow invalid objects to get deleted.

The rollback procedure would look like this:
1. New v1 api with v1beta1 storage is causing problems and user wants to rollback
2. Control plane (including CRD + snapshot controller) is rolled back and the v1 version is dropped. Objects still persist though since it's stored as v1beta1
3. User fixes their problems with invalid objects
4. User upgrades the control plane again.
5. In an n+2 release, once all the invalid objects are purged, we can switch the storage version to v1.

In phase 2, the storage version will be kept at v1beta1 in order to ensure the rollback is possible.

In phase 3, the storage version will be changed to v1.

```yaml
v1 (served=true, storage=false)
v1beta1 (served=false, storage=true)
```

#### Current Controller validation of OneOf semantic

##### Handling VolumeSnapshot.

See code [here](https://github.com/kubernetes-csi/external-snapshotter/blob/v2.1.1/pkg/common-controller/snapshot_controller.go#L192).

If the object violates oneOf semantic: Update the VolumeSnapshot status to “SnapshotValidationError” and issue an event.

Note:
- If the VolumeSnapshot object has been updated AFTER binding to a VSC, binding from VolumeSnapshot->VSC will be lost.
- Deletion of an invalid resource is not blocked by that check as the deletion workflow happens before validation(code). This is to ensure that a user can delete an invalid VolumeSnapshot resource.

##### Handling VolumeSnapshotContent

See code [here](https://github.com/kubernetes-csi/external-snapshotter/blob/v2.1.1/pkg/common-controller/snapshot_controller.go#L91).
If the object violates oneOf semantic: Update the VSC status to “ContentValidationError” and issue an event.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Deployment

There are two main steps to setup validation for the snapshot objects. The kubernetes API server must be configured to connect to the webhook server, and the webhook server must be deployed and reachable. Make sure to take a look at the [prerequisites](#background-on-admission-webhooks) before deploying.

A sample script will be provided which will handle the deployment of TLS certificates. It is not considered production ready and users are encouraged to use their own certificate management process. The demo will create certificates as a secret in the cluster and mount them as a volume. The `ValidatingWebhookConfiguration` will need to be updated with the cluster admin bundle.

### Kubernetes API Server Configuration

The API server must be configured to connect to the webhook server for certain API requests. This is done by creating a ValidatingWebhookConfiguration object. For a more thorough explanation of each field refer to the documentation. An example yaml file is provided below. The value of timeoutSeconds will affect the latency of snapshot creation, and must be considered carefully as it might affect the time the application is frozen for.

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: "webhook-validation.storage.sigs.k8s.io"
webhooks:
- name: "snapshot.webhook-validation.storage.sigs.k8s.io"
  rules:
  - apiGroups:   ["snapshot.storage.k8s.io"]
    apiVersions: ["v1beta1"]
    operations:  ["CREATE", "UPDATE"]
    resources:   ["volumesnapshots", "volumesnapshotcontents"]
    scope:       "*"
  clientConfig:
    service:
      namespace: "default"
      name: "snapshot-validation-service"
      path: "/volumesnapshots"
    caBundle: "LS0tLS...base64 encoded of public key...LS0K"
  admissionReviewVersions: ["v1", "v1beta1"]
  sideEffects: None
  failurePolicy: Ignore # We recommend switching to Fail only after successful installation of the server and webhook.
  timeoutSeconds: 2 # This will affect the latency and performance. Finetune this value based on your application's tolerance.
```

### Webhook Server Deployment

The recommended deployment mode for the webhook server is within the same node as the api server to minimize network latency. For high-availability we recommend using a Deployment and Service to deploy the validation server. Some example yaml files are provided, and should be changed to suit the Cluster Admin’s needs.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: snapshot-validation-deployment
  labels:
    app: snapshot-validation
spec:
  replicas: 3
  selector:
    matchLabels:
      app: snapshot-validation
  template:
    metadata:
      labels:
        app: snapshot-validation
    spec:
      containers:
      - name: snapshot-validation
        image: image:xxx # change the image to released image or if you wish to use your own custom validation server image
        args: ['webhook', '--tls-cert-file=/etc/webhook/certs/cert.pem', '--tls-private-key-file=/etc/webhook/certs/key.pem'] # Change args as needed
        ports:
        - containerPort: 443 # change the port as needed
        volumeMounts:
          - name: webhook-certs
            mountPath: /etc/webhook/certs
            readOnly: true
      volumes:
        - name: webhook-certs
          secret:
            secretName: snapshot-validation-secret
```

```yaml
apiVersion: v1
kind: Service
metadata:
  name: snapshot-validation-service
  namespace: default # Don't use the default namespace. Choose an appropriate one.
spec:
  selector:
    app: snapshot-validation
  ports:
    - protocol: TCP
      port: 443 # Change if needed
      targetPort: 443 # Change if the webserver image expects a different port
```

### Test Plan
There will be unit testing on the webserver in the same repository to ensure that the correct policy gets enforced.

Since the webhook is developed in the external-snapshotter repository, and does not test any csi driver, it would not be a good fit for e2e tests to go under the kubernetes core repository. Hence the plan for e2e tests is to add a new test job in external-snapshotter repo that brings up a [kind](https://kind.sigs.k8s.io/) cluster, installs crds and the webhook, and then runs validation tests.

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

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: VolumeSnapshotDataSource (overall feature gate)
    - Components depending on the feature gate:
  - [ ] Other
    - Describe the mechanism: Create or delete the `validatingwebhookconfiguration` object. Once we reach phase two of the release with validating via CRDs, the feature cannot be disabled.
    - Will enabling / disabling the feature require downtime of the control
      plane? No (Phase 1)
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled). No

* **Does enabling the feature change any default behavior?**
  Currently some validation is not fully enforced. This will tighten the validation to be in line with what is intended.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  In phase one, the feature can be disabled by removing the webhook. However once we update the CRDs users will not easily be able to disable it once they have upgraded.

* **What happens if we reenable the feature if it was previously rolled back?** Nothing special.

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

Sig-storage wide webhook design. This was not accepted because the scope would be too big.

Wait until immutable fields for crds are implemented.
This [KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/20190603-immutable-fields.md) tracks the feature.


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
