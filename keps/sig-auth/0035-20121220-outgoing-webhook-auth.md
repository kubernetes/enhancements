---
kep-number: 35
title: Outgoing Webhook Authentication
authors:
  - "@pbarker"
  - "@mattmoyer"
  - "@xstevens"
owning-sig: sig-auth
participating-sigs:
  - sig-api-machinery
reviewers:
  - "@liggitt"
  - "@tallclair"
  - "@sttts"
  - "@deads2k"
approvers:
  - "@sttts"
  - "@liggitt"
editor: TBD
creation-date: 2018-12-20
last-updated: 2018-12-20
status: provisional
see-also:
replaces:
superseded-by:
---

# Outgoing Webhook Authentication

## Table of Contents

* [Outgoing Webhook Authentication](#outgoing-webhook-authentication)
  * [Table of Contents](#table-of-contents)
  * [Summary](#summary)
  * [Motivation](#motivation)
      * [Goals](#goals)
      * [Non-Goals](#non-goals)
  * [Proposal](#proposal)
      * [User Stories](#user-stories)
        * [Story 1](#story-1)
        * [Story 2](#story-2)
      * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
      * [Risks and Mitigations](#risks-and-mitigations)
  * [Graduation Criteria](#graduation-criteria)
  * [Implementation History](#implementation-history)
  * [Alternatives](#alternatives)

## Summary

We want to provide a means of authenticating outgoing webhooks from the apiserver and its aggregates.

## Motivation

Outgoing webhooks from the apiserver such as the ones found in [dynamic admission control](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers) and [dynamic audit](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#dynamic-backend) 
suffer from a lack of authentication. The intention of this KEP is to provide a means for the receiving server 
to authenticate the apiserver and its aggregates.

### Goals
* A simple means of authenticating apiserver clients.

### Non-Goals
* Providing all the authentication schemes found in the current kubeconfig.

## Proposal

We propose a simple mechanism for authenticating webhooks using the token [Authentication API](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/authentication/types.go). The shared 
[webhook client](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/util/webhook/client.go) will be parameterized to optionally enhance every outgoing request with a token obtained from a  
[Token Request](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/authentication/types.go#L112). The receiving server can then check that token using a [Token Review](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/authentication/types.go#L45).

### User Stories

#### Story 1
I am a cluster administrator using the dynamic auditing feature and want to make sure the logs I receive are from the apiserver.

#### Story 2
I am a cluster administrator using dynamic admission control and want to verify that only verified parties are using my token review endpoint.

### Implementation Details/Notes/Constraints

A new struct will be added to the [client config](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/util/webhook/client.go#L40) for outgoing webhooks:

```go
type AuthInfo struct {
  TokenRequest  *authenticationv1.TokenRequestSpec
}

type ClientConfig struct {
  Name     string
  URL      string
  CABundle []byte
  AuthInfo *AuthInfo
  Service  *ClientConfigService
}
```
If provided the client will provision a token and enhance the outgoing request with an an auth header:
```
Authorization: bearer <token>
```

The client will check and refresh the token when necessary. The server can now check the token using a 
[TokenReview](https://github.com/kubernetes/kubernetes/blob/master/pkg/apis/authentication/types.go#L45).


### Risks and Mitigations

Prevent server from becoming a confused deputy (making attacker-controlled call with apiserver creds).

## Graduation Criteria

We will know if this has succeeded by telling whether it solves the majority of auth concerns around outgoing webhooks.

## Implementation History

- initial draft: 12/20/2018

## Alternatives

We alternatively explored allowing authentication info to be provided in a secret. However, this task becomes  
difficult for the aggregate servers, many of which run in separate namespaces. It wasn't clear how a secret 
could be shared reasonably between them. We haven't abandoned this idea entirely but feel the solution above 
solves the majority of use cases.