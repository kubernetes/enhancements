---
title: Component Bootstrap Token
authors:
  - "@answer1991"
  - "@dixudx"
  - "@zhangxiaoyu-zidif"
owning-sig: sig-auth
approvers:
  - TBD
creation-date: 2020-02-27
last-updated: 2020-02-27
status: provisional
---

# Component Bootstrap Token

## Table Of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Story](#user-story)
    - [Story 1](#story-1)
  - [Implementation Details](#implementation-details)
    - [Bootstrap Token Secret Fields Change](#bootstrap-token-secret-fields-change)
    - [CSR-Approver Change](#csr-approver-change)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Examples](#examples)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
<!-- /toc -->

## Summary

This KEP describes components(or Apps) running out of Kubernetes could use a bootstrap token to request their x509 
certificate through CSR API as its credentials to visit Kubernetes. 

## Motivation

Kubernetes already had an ability that enable Kubelet use a simple bearer token to bootstrap its x509 certificate as 
its credentials to visit Kubernetes, which is a simple and secure way to let a new Kubelet join into Kubernetes. 
And for the components(or Apps) running in the Kubernetes, they could use ServiceAccount Token to its credentials to 
visit Kubernetes.

However, for other components(or Apps) running out of Kubernetes, such as *kube-scheduler*, 
they now have poor user experience to join Kubernetes or visit Kubernetes. 
Kubernetes cluster admin should generate x509 certificate or any other credentials files manually for these components. 
With some command tools help, such as *kubeadm*,  we could simply the credentials file generate, but the security problems still exist. 
The generated credentials file may never expired or with a very long expire date, and we should rotate the credentials file manually.

### Goals

1. Provide a mechanism for components running out of Kubernetes using a bootstrap token to request their x509 certificate automatically.
2. Cluster admin could use [Bootstrap Token Secret](https://kubernetes.io/docs/reference/access-authn-authz/bootstrap-tokens/#bootstrap-token-secret-format) 
to configure *csr-approver* should auto approve specified identity credential CSR if it was created by some specified bootstrap token.

### Non-Goals

TBD

## Proposal

Components running out of Kubernetes could use a bearer token to bootstrap their x509 certificate as their credentials 
to visit Kubernetes, and could use their requested x509 certificate to renew another one.

### User Story

#### Story 1

*kubeadm* could use component bootstrap token to initialize *kube-scheduler* 's credentials file, and *kube-scheduler* will bootstrap its x509 certificate.

### Implementation Details

#### Bootstrap Token Secret Fields Change

TBD

#### CSR-Approver Change

TBD

### Risks and Mitigations

TBD

## Design Details

### Test Plan

TBD

### Graduation Criteria

TBD



#### Examples



##### Alpha -> Beta Graduation

TBD