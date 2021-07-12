---
title: Generate and Mount x509 Certificate for ServiceAccount
authors:
  - "@answer1991"
  - "@aijingyc"
owning-sig: sig-auth
participating-sigs:
  - sig-node
  - sig-api-machinery
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-12-04
last-updated: 2019-12-04
status: provisional


---

# Generate and Mount x509 Certificate for ServiceAccount

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Usage ClientAuth: Request Kubernetes](#usage-clientauth-request-kubernetes)
    - [Usage ServerAuth: Setup Webhook Server](#usage-serverauth-setup-webhook-server)
    - [Dual Client/Serving Certificate Auth: TLS Connection between Ping and Pong](#dual-clientserving-certificate-auth-tls-connection-between-ping-and-pong)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Change](#api-change)
    - [Extend ServiceAccountTLSProjection to VolumeProjection](#extend-serviceaccounttlsprojection-to-volumeprojection)
    - [Extend ServeHosts to CertificateSigningRequestSpec](#extend-servehosts-to-certificatesigningrequestspec)
  - [Encoding Identity to ServiceAccount x509 Certificate](#encoding-identity-to-serviceaccount-x509-certificate)
  - [Kubelet Setup ProjectedVolumeSource Modification](#kubelet-setup-projectedvolumesource-modification)
  - [x509 Authenticator Modification](#x509-authenticator-modification)
  - [CSR ACLs Modification](#csr-acls-modification)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
- [Implementation History](#implementation-history)
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

This KEP describes Kubernetes generates and mounts ServiceAccounts' x509 certificates for Pods.
ServiceAccount's x509 certificate has multiple usage, the most common is ClientAuth and ServerAuth, 
which will help the Pods in Kubernetes reduce complexity of mutual authentication and encrypted connection.

## Motivation

More and more system or components use x509 certificate as identify and encrypt method between Client and Server.
For example, Kubelet and Kubernetes API Server use x509 certificate to identify each other, and the connection between them is encrypted with the TLS. 
Using x509 certificate as identity had been proved efficient and safe.

For now, Kubernetes does not provide a option to generate x509 certificate for ServiceAccount, 
and Pods can not use ServiceAccount x509 certificate as their identity against APIs(Kubernetes API Server or otherwise),
what's more, we had lots of extra work to do to setup a TLS server(HTTPs) in a Pod, such as generate server TLS certificates and mount them as Secret.

Obviously, it's not easy enough when we want to setup mutual authentication between Services, especially the connection should be TLS encrypted.

### Goals

1. Kubernetes generate and mount x509 certificate for ServiceAccount as well as ServiceAccount Token.
2. Pod could request and mount several ServiceAccount x509 certificates for different usage.
3. Provide a option to specify `client-go` using ServiceAccount x509 certificate as identity to request Kubernetes as an alternative plan of ServiceAccount Token.

### Non-Goals

1. Deprecated and remove ServiceAccount Token.
2. Deprecated and remove `TokenRequest` and `TokenReview`.

## Proposal

Kubernetes generate and mount ServiceAccount x509 certificates for Pods.
The container in a Pod could request and mount several ServiceAccount x509 certificates, 
and could specified the usage and some extra information for each ServiceAccount x509 certificate.

### User Stories

#### Usage ClientAuth: Request Kubernetes

The Pod could request and mount ServiceAccount x509 certificate with the usage of ClientAuth.
With this certificate auto mounted, Pod could use it as identity against APIs, such as using it to request Kubernetes.

For example: a Pod named `foo`, and its namespace is `default`. The Pod reference a ServiceAccount named `foo-sa`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  namespace: default
  name: foo
spec:
  serviceAccountName: foo-sa
  ...
```

Then 4 files will be auto mounted to Pod path `/var/run/secrets/kubernetes.io/serviceaccount`:

1. `ca.crt`: the Kubernetes Server CA file.
2. `token`:  the ServiceAccount JWT.
3. `tls.crt`: the ServiceAccount x509 certificate.
4. `tls.key`: the ServiceAccount x509 key.

The application in this Pod is build with `clien-go`, and it visit Kubernetes(`https://kubernetes.default.svc`) could use ServiceAccount x509 certificate(`tls.crt` and `tls.key`). 
Kubernetes could get the ServiceAccount identity information from its requests.

#### Usage ServerAuth: Setup Webhook Server

When we extend Kubernetes, Webhook Server is suggested. 
With ServiceAccount x509 certificate, we could make setting up Webhook Server simple as the HTTPs TLS certificate generated and CA is trusted.

For example, we can setup Webhook Server Pod like that:

```yaml
apiVersion: v1
kind: Pod
metadata:
  namespace: default
  name: webhook
spec:
  serviceAccountName: foo-sa
  containes:
  - name: webhook
    image: webhook:latest
    volumeMounts:
    - name: server-tls
      path: /server-tls
  volumes:
    - name: server-tls
      projected:
        serviceAccountTLS: 
          usage:
          - ServerAuth
          serveHosts:
          - webhook
          - webhook.default
          - webhook.default.svc
          certificatePath: tls.crt
          certificateKey: tls.key
  ...
```

The Webhook Server Pod could find `tls.crt` and `tls.key` in path `/server-tls`, 
and which can be used as Webhoo HTTPs TLS certificate, the SANs could be specified too. 

#### Dual Client/Serving Certificate Auth: TLS Connection between Ping and Pong

There are two Service(Ping and Pong) need visit each other and Auth the requests, the scene like the issue [#62747](https://github.com/kubernetes/kubernetes/issues/62747) described.

With the ServiceAccount x509 certificates implemented, 
Ping and Pong could use ServiceAccount x509 certificate as their ServerAuth and ClientAuth.

For example, the Ping Pod may like that:

```yaml
apiVersion: v1
kind: Pod
metadata:
  namespace: default
  name: ping
spec:
  serviceAccountName: ping-sa
  containes:
  - name: ping
    image: ping:latest
    volumeMounts:
    - name: tls-client-k8s
      path: /var/run/secrets/kubernetes.io/serviceaccount
    - name: tls-client-pong
      path: /tls-client-pong
    - name: tls-server
      path: /tls-cerver
  volumes:
    - name: tls-client-pong
      projected:
        configMap:
          name: kube-root-ca.crt
          items:
            - key: ca.crt
              path: ca.crt
        serviceAccountTLS: 
          usage:
          - ClientAuth
          extensions:
          - "client-name=ping"
          certificatePath: tls.crt
          certificateKey: tls.key
    - name: tls-server
      projected:
        serviceAccountTLS: 
          usage:
          - ServerAuth
          serveHosts:
          - ping
          - ping.default
          - ping.default.svc
          certificatePath: tls.crt
          certificateKey: tls.key
  ...
```

There are 3 ServiceAccount x509 certificates exist in the Ping Pod:

1. *tls-client-k8s*: mount in `/var/run/secrets/kubernetes.io/serviceaccount`, 
which Ping used as ClientAuth to visit Kubernetes, Kubernetes could get ServiceAccount namespace and name from Ping's request.
2. *tls-client-pong*: mount in `/tls-client-pong`, which Ping used as ClientAuth to visit Pong Service. 
The Pong could get these information from Ping's request certificate and do Authz then:
   1. Client ServiceAccount namespace and name.
   2. Extensions, which is `client-name=ping` in this example.
3. *tls-server*: mount in `/tls-cerver` which Ping used as its HTTPs TLS certificate. 
When Pong requests to Ping, Pong will check Ping's TLS certificate is trusted.

The Pong example omitted, as this just like a mirror of Ping.

### Risks and Mitigations

TBD, let's talk about ServiceAccount x509 certificate rotate and CA/CA-Key leak.

## Design Details

### API Change

#### Extend ServiceAccountTLSProjection to VolumeProjection

`ServiceAccountTLSProjection` is used by a Pod to request and mount the ServiceAccount x509 certificates:

```go
// ServiceAccountTLSProjection represents a projected service account x509 certificate
// volume. This projection can be used to insert a service account certificate into
// the pods runtime filesystem for use against APIs (Kubernetes API Server or
// otherwise).
type ServiceAccountTLSProjection struct {
  // Usages is the usage of TLS certificate.
  Usages []ServiceAccountTLSUsage
  
  // ServeHosts is the SANs for DNS or IPs.
  ServeHosts []string

  // ExpirationSeconds is the requested duration of validity of the service
  // account TLS certificate.
  ExpirationSeconds int64

  // Extensions are the extensions in the Subject of TLS certificate.
  Extensions []string

  // CertificatePath is the path relative to the mount point of the file to project the
  // TLS certificate into.
  CertificatePath string

  // KeyPath is the path relative to the mount point of the file to project the
  // TLS key into.
  KeyPath string
}

// Projection that may be projected along with other supported volume types
type VolumeProjection struct {
  // all types below are the supported types for projection into the same volume

    ...

  // information about the serviceAccount TLS data to project
  // +optional
  ServiceAccountTLS *ServiceAccountTLSProjection
}
```

### Encoding Identity to ServiceAccount x509 Certificate

There are 3 identity information should be encoding into the ServiceAccount x509 certificate Subject:

* Encoding ServiceAccount Identity

Username of ServiceAccount will be encoded into Subject CommonName, 
the format will be `system:serviceaccount:${sa-namespace}:${sa-name}`.
Two Groups of ServiceAccount will be encoded into Subject Organization, 
the format will be `system:serviceaccounts` and `system:serviceaccounts:${sa-namespace}`.

1. ${sa-namespace}: the namespace of ServiceAccount.
3. ${sa-name}: the name of ServiceAccount.

Kubernetes API Server Authz the requests according to this information if client use ServiceAccount x509 certificate as their identity.

* Encoding Pod Identity

The namespace and name the Pod which request and mount the ServiceAccount x509 certificate will be encoded into Subject OrganizationalUnit,
the format will be `system:pod-namespace=${pod-namespace}` and `system:pod-name=${pod-name}`.

1. ${pod-namespace}: the namespace of Pod which request and mount this x509 certificate.
3. ${pod-name}: the name of Pod which request and mount this x509 certificate.

Kubernetes API Server will decode these information and add them to UserInfo.Extensions. 
And it's useful information when two Service implement dual Client/Serving certificate Authn. 
    
* Encoding Extensions in ServiceAccountTLSProjection

The `Extensions` field values in `ServiceAccountTLSProjection` will be encoded into certificate's Subject OrganizationalUnit.

Kubernetes API Server will decode these information and add them to UserInfo.Extensions. 
And it's useful information when two Service implement dual Client/Serving certificate Authn.

### Kubelet Setup ProjectedVolumeSource Modification

As `ServiceAccountTLSProjection` extended to `VolumeProjection`, Kubelet should handle to setup `ServiceAccountTLSProjection`.
Kubelet will create a CSR for the `ServiceAccountTLSProjection` and get the certificate data from CSR status. 
Also, Kubelet will handle the ServiceAccount x509 certificate rotate just like the `ServiceAccountTokenProjection`.

### x509 Authenticator Modification

The x509 authenticator will be extended to support getting the ServiceAccount UserInfo from the
requests.

### CSR ACLs Modification

The NodeAuthorizer will allow the Kubelet to use its credentials to create a ServiceAccount x509 certificate CSR 
on behalf of pods running on that node.
The NodeRestriction admission controller will require that these certificates are pod
bound.

### Test Plan

TBD

### Graduation Criteria

##### Alpha -> Beta Graduation

TBD

## Implementation History

- 2019-12-04: KEP opened

## Alternatives

TODO