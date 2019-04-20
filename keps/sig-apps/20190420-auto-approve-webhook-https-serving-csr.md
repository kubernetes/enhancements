---
title: Auto Approve Webhook HTTPs serving CertificateSignRequest
authors:
  - "@answer1991"
owning-sig: sig-apps
reviewers:
approvers:
creation-date: 2019-04-20
last-updated: 2019-04-20
status: provisional
---

# Table of Contents

   * [Table of Contents](#table-of-contents)
   * [Summary](#summary)
   * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
   * [Proposal](#proposal)
         * [User Stories](#user-stories)
            * [Story 1](#story-1)
         * [Risks and Mitigations](#risks-and-mitigations)
   * [Design Details](#design-details)
   * [Implementation History](#implementation-history)

# Summary

This feature will enable *kube-controller-manager* to auto approve `CertificateSignReuqest` from Webhook server, and the certificate declare in `CertificateSignRequest` can only be used for Webhook HTTPs serving.

# Motivation

For now, develop and deploy a Webhook server to Kubernetes is a little complicated:

 	1. Generate a self-certificated SSL pem pair
 	2. Store the SSL pair as a Kubernetes Secret
 	3. Deploy the Webhook and mount the Secret to the Webhook Pod, and Webhook application use the mounted SSL Secret to serve HTTPs service
 	4. Declare the `MutatingWebhookConfiguration` or `ValidatingWebhookConfiguration` using the self-certificated CA

If Kubernetes can auto-approve the webhook HTTPs serving `CertificateSignRequest`, the deploy process will be simplified.

### Goals

1. Develop a admission-plugin called `AdmissionWebhookCAInjector`, which will inject CA for the `MutatingWebhookConfiguration` or `ValidatingWebhookConfiguration` if its webhook CA is empty
2.  *kube-controller-manager* auto-approve the Webhook HTTPs Serving `CertificateSignRequest(CSR)`, the CSR has the following features:
   1. Without CommonName and Groups
   2. Usages must be:
      1. digital signature
      2. key enciphermen
      3. server auth
3. Webhook application can create `CertificateSignRequest(CSR)` at the entrypoint main function. After CSR is approved, Webhook application use the CSR generated certificate to serve HTTPs service.

### Non-Goals

* Disable Webhook serving HTTPs using self-certificated SSL is NOT a goal

# Proposal

### User Stories

#### Story 1

TODO

### Risks and Mitigations

TODO

# Design Details

# Implementation History

* 2019-04-07: Initial KEP document.