---
title: KEP Template
authors:
- "@mvladev"
owning-sig: sig-api-machinery
participating-sigs:
- sig-node
- sig-network
reviewers:
- "@deads2k"
- "@thockin"
approvers:
- "@deads2k"
editor: "@mvladev"
creation-date: 2019-08-20
last-updated: 2020-01-21
status: provisional
see-also: []
replaces: []
superseded-by: []
---

# Title

Master service of type ExternalName

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Kube-apiserver](#kube-apiserver)
  - [kubelet](#kubelet)
  - [Implementation Notes](#implementation-notes)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
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

The master kubernetes service created in the `default` namespace is called `kubernetes`. It's used by Pods running in the cluster to discover the kube-apiserver and communicate with it.

kube-apiserver manages the master `Service` of type `ClusterIP` and associated `Endpoints` objects. Using this service, kubelet injects two environment variables called `KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT` into Pod's containers. Those environment variables can be detected by Kubernetes API clients such as `client-go` and used to determine if they run as a Pod in the cluster.

This KEP is about adding support in Kubernetes for the master service to be of type `ExternalName` in addition to `ClusterIP`. This allows for kube-apiservers exposed via external FQDN (fully qualified domain name) to be directly consumed by Pods running in the cluster via that FQDN.

## Motivation

When exposing the kube-apiserver in HA mode, typically, a Layer-4 load-balancer is used. On different public/private cloud-providers, the endpoint of that load-balancer is either an IPv4/IPv6 address or a hostname (AWS being most prominent example for such offering). When consuming the latter offering, kubernetes cluster operators cannot use the hostname directly and have to disable the master endpoint reconciler in kube-apiserver with `--endpoint-reconciler-type=none` and write [custom controllers](https://github.com/gardener/aws-lb-readvertiser) which convert the hostname to IP addresses.

It might be more suitable to advertise kube-apiserver's endpoint as FQDN, such as `api.my-cluster.example.com` for disaster recovery / prevention - in case of accidental deletion of said load-balancer and/or it's associated IP, the only thing required for recovery is to reconfigure the NS records to point to the newly created load-balancer.

Historically there are several issues related to this problem:

- https://github.com/kubernetes/kubernetes/issues/34043
- https://github.com/kubernetes/kubernetes/issues/47588
- https://github.com/coreos/tectonic-installer/issues/384

### Goals

- Make kube-servers exposed and advertised as FQDN, first class citizen in Kubernetes.
- The master service reconciler in kube-apiserver should support services of type `ExternalName`.
- Support in kubelet for injecting the required environment variables `KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT` for [in-cluster discovery](https://github.com/kubernetes/client-go/blob/c8dc69f8a8bf8d8640493ce26688b26c7bfde8e6/rest/config.go#L399-L411) for master kubernetes service of type `ExternalName`.

### Non-Goals

- Environment variable support in kubelet for services of type `ExternalName` other than the master service. Related [comment](https://github.com/kubernetes/kubernetes/issues/60535#issuecomment-490757630).
- In-cluster API server detection/auto-configuration other than environment variables named `KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT`.
- Network connectivity from Nodes/Pods to the kube-server is out of scope and is assumed to be properly configured and working. Pods using the hostnetwork or CNI network are able to initiate connections to kube-apiserver.

## Proposal

In order to support this feature in Kubernetes the following changes are proposed:

### Kube-apiserver

The master service reconciler in kube-apiserver is extended to support services of type `ExternalName`.

Example:

Given a kube-apiserver advertised with host `api.127.0.0.1.nip.io` and port `443`, it will reconcile the master service with the following content:

```yaml
apiVersion: v1
kind: Service
metadata:
  labels:
    component: apiserver
    provider: kubernetes
  name: kubernetes
  namespace: default
spec:
  clusterIP: ""
  type: ExternalName
  externalName: api.127.0.0.1.nip.io
  ports:
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
```

It'll also delete the corresponding `Endpoints` called `kubernetes` from the `default` namespace as it's not needed for `Services` of type `ExternalName`.

To provide a backwards-compatible way for cluster operators to enable/disable/configure this feature a new flag is going to be introduced:

- `--kubernetes-service-external-name` with the following description:

  ```text
  The hostname on which to advertise the apiserver to members of the cluster.
  This address must be reachable by the rest of the cluster.
  The master service will be of type ExternalName, using this as the value of the externalName.
  If blank, --advertise-address is going to be used and the master service will be of type ClusterIP.
  Useful if the apiserver is running behind a load balancer which does not have a static ip.
  ```

### kubelet

No changes to kubelet flags are needed.

The environment variable injection mechanism is going to be changed to allow the master service to be of type `ExternalName`.

Example:

Given a kube-apiserver advertised with host `api.127.0.0.1.nip.io` and port `443` when executing `env` command in a Pod's container, the following should be returned:

```text
KUBERNETES_SERVICE_PORT=443
KUBERNETES_PORT=tcp://api.127.0.0.1.nip.io:443
KUBERNETES_PORT_443_TCP_ADDR=api.127.0.0.1.nip.io
KUBERNETES_PORT_443_TCP_PORT=443
KUBERNETES_PORT_443_TCP_PROTO=tcp
KUBERNETES_SERVICE_PORT_HTTPS=443
KUBERNETES_PORT_443_TCP=tcp://api.127.0.0.1.nip.io443
KUBERNETES_SERVICE_HOST=api.127.0.0.1.nip.io

# container-specific variables
HOSTNAME=busybox
SHLVL=1
HOME=/root
TERM=xterm
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
```

### Implementation Notes

While the `--external-hostname` flag already exists in kube-apiserver, it's documented as targeting mainly external (not in-cluster) clients:

```text
--external-hostname string
  The hostname to use when generating externalized URLs for this master (e.g. Swagger API Docs).
```

and in some cases / network configurations, the API server might be advertised as `api.external.example.com` to (human) end-users, but in the same time it can be only reachable via `api.internal.example.com` from in-cluster clients / Pods.

### Risks and Mitigations

This feature requires a running DNS solution such as CoreDNS for connectivity to the API server.

Adjustments to CPU and Memory resources of said DNS service might be needed to cope with extra queries.

## Design Details

TBD

### Test Plan

- Unit Tests: All changes to kube-apiserver and kubelet must be covered by unit tests.
- Integration Tests: The use cases discussed in this KEP must be covered by integration tests.

### Graduation Criteria

(none yet)

**Note:** *Section not required until targeted at a release.*

<!--
Define graduation milestones.

These may be defined in terms of API maturity, or as something else. Initial KEP should keep
this high-level with a focus on what signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/
-->

### Upgrade / Downgrade Strategy

- As this feature needs to be explicitly enabled via a the `--kubernetes-service-external-name` flag in kube-apiserver, no steps are needed when upgrading with kube-apiserver or kubelet version which support it.
- For downgrade to versions which support this feature - no steps are needed.
- For downgrade to versions which do not support this feature, `--kubernetes-service-external-name` must be removed from kube-apserver.

### Version Skew Strategy

Following the kubernetes version skew [policy](https://kubernetes.io/docs/setup/release/version-skew-policy/):

- older kubelet versions (n-2 max) which do not support this functionality cannot host Pods which use in-cluster configuration. This can be mitigated by:
  - (manually) tainting those Nodes, so kube-scheduler does not assign those Pods to them.
  - having this feature behind a feature gate (disabled by default) which can be enabled once kubelet is upgraded.
  - custom mutating webhook admission controller which manually adds `KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT` environment variables to each Pod container scheduled on such Nodes.

## Implementation History

Historically, an implementation for this feature was proposed on a two occasions:

- https://github.com/kubernetes/kubernetes/pull/47588
- https://github.com/kubernetes/kubernetes/pull/79312

<!--
Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

- enabling / disabling this functionality in kube-apiserver will require restart of all Pod containers which use the environment variables for in-cluster configuration.
- in-cluster DNS solutions such as CoreDNS is a requirement for this functionality.
- in-cluster DNS solutions such as CoreDNS must have `spec.dnsPolicy: Default` to avoid chicken and egg problem. This is already the case for [CoreDNS](https://github.com/kubernetes/kubernetes/blob/1719ce7883db57d24d70605606f189469fd50b60/cluster/addons/dns/coredns/coredns.yaml.base#L164) and [KubeDNS](https://github.com/kubernetes/kubernetes/blob/1719ce7883db57d24d70605606f189469fd50b60/cluster/addons/dns/kube-dns/kube-dns.yaml.base#L217).

## Alternatives

- continue using IP advertised apiservers.
- an alternative would be to use TLS passthrough proxies (such as HAProxy or Envoy) and update the master service to include a selector to those workloads:

  ```yaml
  apiVersion: v1
  kind: Service
  metadata:
    labels:
      component: apiserver
      provider: kubernetes
    name: kubernetes
    namespace: default
  spec:
    clusterIP: "10.0.0.1"
    type: ClusterIP
    selector:
      app: haproxy
    ports:
    - name: https
      port: 443
      protocol: TCP
      targetPort: 443
  ```

- in-cluster configuration detection could be changed in clients and kubelet - instead of using environment variables, a additional files could be mounted in containers by kubelet:
  - `/var/run/secrets/kubernetes.io/serviceaccount/kubernetes-service-host` could be the file alternative of `KUBERNETES_SERVICE_HOST` environment variable.
  - `/var/run/secrets/kubernetes.io/serviceaccount/kubernetes-service-port` could be the file alternative of `KUBERNETES_SERVICE_PORT` environment variable.
