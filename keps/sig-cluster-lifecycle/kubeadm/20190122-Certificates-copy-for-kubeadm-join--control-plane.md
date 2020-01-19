---
title: "Certificates copy for join --control-plane"
authors:
  - "@fabriziopandini"
owning-sig: sig-cluster-lifecycle
participating-sigs:
reviewers:
  - "@neolit123"
  - "@detiber"
  - "@mattmoyer"
  - "@chuckha"
  - "@liztio"
approvers:
  - "@timothysc"
  - "@luxas"
editor: TBD
creation-date: 2019-01-22
last-updated: 2019-04-18
status: implementable
see-also:
  - KEP-0015
replaces:
superseded-by:
---

# Certificates copy for join --control-plane

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non Goals](#non-goals)
- [Proposal](#proposal)
  - [Constraints](#constraints)
  - [Key element of the proposal](#key-element-of-the-proposal)
  - [Implementation Details](#implementation-details)
    - [The encryption key](#the-encryption-key)
    - [The kubeadm-certs secret](#the-kubeadm-certs-secret)
    - [The TTL-token](#the-ttl-token)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [<strong>Test</strong> Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [x] k/enhancements [issue 357](https://github.com/kubernetes/enhancements/issues/357) in release
      milestone and linked to KEP
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG
      meetings, relevant PRs/issues, release notes

## Summary

Automatic certificates copy makes easier to create HA clusters with the kubeadm tool using exactly
the same `kubeadm init` and `kubeadm join` commands the users are familiar with.

## Motivation

As confirmed by the recent [kubeadm survey](https://drive.google.com/file/d/1eN9sGsdXWpurmplbEVn9UX5NiseQqlzO/view?usp=sharing),
support for high availability cluster is one of the most requested features for kubeadm.

A lot of effort was already done in kubeadm for achieving this goal, among them the redesign
of the kubeadm config file and its graduation to beta and the implementation of the
[`kubeadm join --control-plane workflow (KEP0015)`](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cluster-lifecycle/kubeadm/0015-kubeadm-join-control-plane.md),
but the solution currently in place stills requires the manual copy of cluster certificates from
the bootstrap control-plane node to secondary control-plane nodes.

This KEP introduces automatic certificates copy, eliminating the manual operation described
above and completing the kubeadm solution for creating HA clusters.

### Goals

- Usability: use exactly the same `kubeadm init` and `kubeadm join` commands the users are
  familiar with.
- Security: provide a solution that enables the secure copy of cluster certificates between
  control-plane nodes

### Non Goals

- Handle certificate copy in case of External CA.
  If users decide to use an external CA with kubeadm without providing the CA keys, they will be required to create
  signed certificates and all the kubeconfig files (including X.509 certificates) for the control plane nodes.
  This is needed, because kubeadm cannot generate any certificates without the required CA keys.

## Proposal

### Constraints

The solution described in this proposal is deeply influenced by following constraints:

1. Kubeadm can execute actions only on the machine where it is running e.g. it is not
   possible to execute actions on other nodes.
2. `kubeadm init` and `kubeadm join` are separated actions (executed on separated machines/at
   different times).
3. During the join workflow, kubeadm can access the cluster only using identities with
   limited grants, namely `system:unauthenticated` or `system:node-bootstrapper`.

### Key element of the proposal

At a high level, the proposed solution can be summarized by the following workflow:

**On control plane node 1:**

```bash
kubeadm init --upload-certs
```

The new `--upload-certs` flag will trigger a new kubeadm init phase executing following
actions:

1. An encryption key (32bytes for key SHA-256) will be generated; such key **will never be**
   **stored on cluster**

2. Cluster certificates will be encrypted using the above key (using AES-256/GCM as method)

3. Encrypted certificates will be stored in a new Kubernetes secret named `kubeadm-certs`;
   it is important to notice that:

   - This secret is the technical solution that provides a temporary bridge between
     `kubeadm init` and `kubeadm join` commands/between different nodes.

   - Without the encryption key, this secret contains a harmless bag of bytes.

4. A second bootstrap token will be generated, with a shorter duration than the
   join token (the TTL-token)

5. The `ownerReference` field in the `kubeadm-certs` secret will be linked to the
   TTL token, thus ensuring automatic deletion of the secret when the TTL-token gets
   deleted by the `TokenCleaner` controller

6. RBAC rules will be created to ensure access to the above config map to users
   in the `system:node-bootstrapper` group (the bootstrap tokens).

**On control plane node 2:**

```bash
kubeadm kubeadm join --control-plane --certificate-key={key from step above}
```

The new `--certificate-key` will trigger following actions:

1. The `kubeadm-certs` config will be retrieved from the cluster
2. Cluster certificates will be decrypted using the provided key, and
   then stored on the disk

### Implementation Details

#### The encryption key

The default lifecycle of the encryption key will be the following:

- a random encryption-key (32bytes for key SHA-256) will be created
  by `kubeadm init --upload-certs`
- the encryption-key will be printed in the output of the `kubeadm init --upload-certs`
  command, but never stored in the cluster
- after some TTL the `kubeadm-certs` encrypted with the above key will be automatically
  deleted, thus limiting the risks related to a possible theft of encryption-key (see
  risks and mitigations for more detail).

The following variant of the default lifecycle will be supported:

1. The user will be allowed to pre-generate an encryption-key and pass it to
   `kubeadm init --upload-certs` command using the kubeadm config file
2. The user will be allowed to hide the encryption-key from the kubeadm output
   using the `skip-token-print` or (a similar flag)
3. The user will be allowed to pass the encryption-key to `kubeadm join --control-plane`
   command using the kubeadm config file instead of the `--certificate-key` flag
4. The user will be allowed to generate a new encryption-key after the first one expires by
   invoking the `kubeadm init phases upload-certs` command (the command will
  re-create/override the existing `kubeadm-certs` secret as well)

#### The kubeadm-certs secret

The `kubeadm-certs` secret will be stored in the `kube-system` namespace; RBAC rules
for ensuring access to the users in the `system:node-bootstrapper` group
(the bootstrap tokens) will be created.

The `kubeadm-certs` secret `ownerReference` attribute will be set to the TTL-token
UID as described in the example below; See [Garbage Collection](https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/)
for more details.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kubeadm-certs
  namespace: kube-system
  ownerReferences:
  - apiVersion: v1
    controller: true
    blockOwnerDeletion: true
    kind: Secret
    name: bootstrap-token-abcdef
    uid: 84e913dc-191a-11e9-96ae-0242f550e41f
type: Opaque
data:
  ...
```

The `kubeadm-certs` secrets will contain the cluster certificates encrypted
using the encryption key (32bytes, SHA-256) with with AES-256/GCM as method, and then
base64 encoded as usual.

Please note that the upload certs process defined in this KEP will always
upload all the necessary certificates (regardless of the cluster architecture e.g.
external-CA or external etcd), because the assumption is that, when executed with
the `--upload-certs` flag, kubeadm is delegated by the user to copy all the certificates
required for joining a new control plane node.
In other words, it won't be possible to use kubeadm `--upload-certs` for copying a subset
of certificates only.

For example, in a cluster with local etcd the following certs/keys will be copied:
- cluster CA cert and key (`/etc/kubernetes/pki/ca.crt` and `/etc/kubernetes/pki/ca.key`)
- Front proxy CA cert and key (`/etc/kubernetes/pki/front-proxy-ca.crt` and `/etc/kubernetes/pki/front-proxy-ca.key`)
- service account signing key (`/etc/kubernetes/pki/sa.key` and `/etc/kubernetes/pki/sa.pub`)
- Etcd CA ert and key (`/etc/kubernetes/pki/etcd/ca.crt` and `/etc/kubernetes/pki/etcd/ca.key`)

Please note that client certificates are not part of the above list of certificates
because `kubeadm join --control-plane` workoflow generates new client certificates taking
care of adding SANS specifically for the joining node.

#### The TTL-token

The TTL-token is a regular bootstrap token, with a short duration, but without
any assigned usage or groups (see [Authenticating with Bootstrap Tokens](https://kubernetes.io/docs/reference/access-authn-authz/bootstrap-tokens/#bootstrap-token-secret-format)
for more details).

This token is created with the only purpose of triggering automatic deletion of
the `kubeadm-certs` secret when deleted (the two objects will be linked via
`ownerReference` attributes).

The `TokenCleaner` controller will ensure automatic deletion of bootstrap tokens
and therefore also deletion of the TTL-token.

### Risks and Mitigations

In case the cluster administrator uses the `kubeadm init --upload-certs` flag,
the `kubeadm-certs` secret will store sensitive cluster certificates. The following
security measures apply to the `kubeadm-certs` secret:

- the `kubeadm-certs` secret is only valid for a short period of time (with a duration
  shorter than bootstrap tokens)
- if an attacker get access to the cluster when the `kubeadm-certs` secret still
  exists, the cluster certs contained in the secret are protected by RBAC rules
- if an attacker get access to the cluster when the `kubeadm-certs` secret still
  exists with an identity that is authorized to read the `kubeadm-certs` secret,
  the cluster certificates contained in the secret will be protected by the
  encryption algorithm (without the encryption key, the secret contains a harmless
  bag of bytes)

The following security measures apply to the encryption key:

- the encryption key will never be stored on the cluster by kubeadm
- the encryption key will never be transmitted on the wire by kubeadm
- the encryption key is implicitly valid only for a short period of time (because
  when the `kubeadm-certs` secret is automatically deleted, the encryption key
  became useless). Please note that this does not apply if the content of the
  kubeadm-certs was already obtained, the encryption-key could later decode it
  (see residual risk below).
- it will be possible to hide the encryption key from the `kubeadm init` output
  by using `--skip-token-print` or similar flag (but this requires the user
  passing an encryption key to kubeadm using the kubeadm config file), thus
  avoiding the key improperly registered in logs
- it will be possible to use a config file for passing the encryption key to
  `kubeadm join` command, thus avoiding the key improperly registered in
  shell history

Another possible risk derives from kubeadm allowing the user to fully customize
control plane components, including also the possibility to remove the TokenCleaner
from the list of enabled controllers in the kube-scheduler configuration, and, as
a consequence, disable the mechanisms that manages temporarily valid tokens and
`kubeadm-certs` secret.

A possible mitigation action for this risk is to enforce the TokenCleaner in case
when the user opts in to the upload-certs workflow defined in this document.

Despite all the above measures, there is still a residual risk in case an attacker
manages to steal the encryption key from the cluster administrator and to get access
to the cluster with adequate rights before the `kubeadm-certs` secret is
automatically deleted.

However, we assume this residual risk could be accepted by the cluster administrator
in favor of the simplified UX offered by the automatic certificates copy during
the `kubeadm init` / `kubeadm join --control-plane` operations.

Nevertheless, as an additional mitigation action, the potential risk will be clearly
stated both in the documentation and in the kubeadm output.

## Design Details

### **Test** Plan

This feature is going to be tested:

- with unit tests
- with e2e tests, if kubeadm E2E tests are implemented during the v1.14 cycle
- with integration tests (using Kind, recently extended for supporting HA/multi node)

### Graduation Criteria

- gather feedback from users
- examples of real world usages
- properly document the new function and the residual risk
- have CI/CD tests validating the `kubeadm init` / `kubeadm join --control-plane`
  workflow with automatic certificates copy

### Upgrade / Downgrade Strategy

Not applicable (this KEP simplify the cluster bootstrap process, but does not affect
how upgrades/downgrades are managed)

### Version Skew Strategy

Not applicable (this KEP simplify the cluster bootstrap process, but does not
affect component version or version skew policy)

## Implementation History

- 22 Jan 2019 - first release of this KEP
- v1.14 implementation as alpha feature without
  - Extension of the kubeadm config file for allowing usage of pre-generated certificate keys
  - TokenCleaner enforcement
  - E2E tests

## Alternatives

Several alternatives were considered:

- a simpler version of this proposal was considered (without TTL), but discarded
  in favor of a more secure approach
- alternative solutions - without the need of the `kubeadm-certs` secret - were
  considered as well:
  - Client/server architecture, with a service component on a bootstrap control-plane;
    this was discarded due to the higher complexity
  - Creation of kubernetes job with the responsibility of reading certs from
    a bootstrap control-plane; this was discarded due to to the limited authorization
    of the identity used by the kubeadm join workflow
