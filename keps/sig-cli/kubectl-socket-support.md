---
title: Socket Support for Kubectl
authors:
  - "@choo-stripe"
  - "@dixudx"
  - "@answer1991"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-cli
reviewers:
  - "@pwittrock"
  - "@soltysh"
  - "@seans3"
approvers:
  - "@pwittrock"
  - "@soltysh"
  - "@liggitt"
editor: "@dixudx"
creation-date: 2020-03-16
last-updated: 2020-03-16
status: implementable
---

# Socket Support for Kubectl

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location
in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary
This proposal adds a new feature that would allow sockets (`unix` and `tcp`) as a new protocol for the server to be connected.

## Motivation

Currently several kinds of credentials (passwords, tokens, x509 certificates and etc) are stored in a kubeconfig file and
persisted on disk. It is not safe and controllable, especially when this kubeconfig file with credentials gets spread to everywhere.
Moreover, it will become difficult to revoke the access to the apiserver.

Sockets are used as a secured proxy into some infrastructure. For now kubectl does not expose a way to send the request
over a unix socket. Currently we've only got `http` and `https`. `Docker` client does provide such supports via different types of Socket: `unix` and `tcp`.

It would be great to be able to send requests from kubectl over sockets.

### Goals

* kubectl users can use sockets to connect to the apiserver:
    * Allow passing socket file to `--server=unix:///path/to/socket` for kubectl, much like `-H unix:///var/run/docker.sock -H tcp://192.168.59.106` in `docker` client;
    * Allow specifying sockets as server addresses in kubeconfig file;

### Non-Goals

* Convey credentials through unix sockets;
* Adopting [SGX] to secure the memory access;

[SGX]: https://www.intel.com/content/www/us/en/architecture-and-technology/software-guard-extensions.html

## Proposal

### User Stories

To access a Kubernetes cluster, users will issue `kubectl` using an appropriate kubeconfig with scoped roles. However it is
hard to avoid the kubeconfig files get spread, which will bring risks for production clusters. Moreover, it is not easy to revoke
those kubeconfig files unless they get expired, especially TLS certificates.

From the perspective of security, a more dynamic kubeconfig is preferred, such as using unix sockets, which could be revoked and closed
as needed. As well this will bring good isolation when socket file is read-only by specific users.

```bash
export TOKEN=USER1_TOKEN
kubectl proxy --unix-socket=/home/user1/.kube/kube.sock --token=${TOKEN}
```

Such of proxies could be setup automatically by administrators on demand. There is no need to expose tokens, certificates to users.

Then user `user1` can issue below command to the cluster, while other users have no access to such socket file. It is a good way to keep it
confidential.

```bash
kubectl --server=/home/user1/.kube/kube.sock get pods
```

### Implementation Details/Notes/Constraints

Also socket addresses can be specified as `Cluster.Server`.

```go
// staging/src/k8s.io/client-go/tools/clientcmd/api/v1/types.go
// Cluster contains information about how to communicate with a kubernetes cluster
type Cluster struct {
    // Server is the address of the kubernetes cluster (https://hostname:port).
    // Socket address is still supported in experimental, such as unix:///var/run/kube.sock, tcp://192.168.59.106
    Server string `json:"server"`
    ...
}
```

To connect to apiserver via sockets, we also need to modify current `ClientConfig()` to convey unix sockets over HTTP.

```go
// staging/src/k8s.io/client-go/tools/clientcmd/client_config.go
// ClientConfig is used to make it easy to get an api server client
type ClientConfig interface {
	...
	// ClientConfig returns a complete client config
	ClientConfig() (*restclient.Config, error)
	...
}
```

Taking `DirectClientConfig` for an example,

```go
// ClientConfig implements ClientConfig
func (config *DirectClientConfig) ClientConfig() (*restclient.Config, error) {
	...

	clientConfig := &restclient.Config{}
	clientConfig.Host = configClusterInfo.Server
    
    if scheme, sockpath, err := isSocketAddr(clientConfig.Host); err == nil {
        clientConfig.Dial = func(_ context.Context, _, _ string) (net.Conn, error) {
            return net.DialTimeout(scheme, sockpath, defaultTimeout)
        }
    }
    ...

}
```

### Risks and Mitigations

There should be no compatibility issues. New clients will still be compatible with old servers and vice-versa.
And it will bring no change for in-cluster accessing since such unix socket cannot be speficied
by `KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT`.

This proposal will need to modify `k8s.io/client-go` to allow `Cluster.Server` to be a unix socket. So technically unix sockets can be used
by webhooks and operators as well. For cluster deployers and administrators, please be aware to keep unix socket file confidential by setting right file modes
to avoid being misused.

Sending HTTP request over unix socket is not a new thing, which is supported by other languages as well, such as Python, Java.
Please refer to [docker client sdk](https://docs.docker.com/engine/api/sdk/#unofficial-libraries) for references.

## Design Details

### Test Plan

### Graduation Criteria

Once the experimental kubectl socket flag is implemented, this can be rolled out in multiple phases.

#### Alpha -> Beta Graduation
- [ ] Gather the feedback, which will help improve the command
- [ ] Extend with the new features based on feedback

#### Beta -> GA Graduation
- [ ] Address all major issues and bugs raised by community members

### Upgrade / Downgrade Strategy

This section is not relevant because this is a client-side component only.

## Implementation History

* 2020-03: Added KEP
