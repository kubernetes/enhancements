---
title: Out-of-Tree Credential Providers
authors:
  - "@mcrute"
owning-sig: sig-cloud-provider
participating-sigs:
  - sig-node
  - sig-auth
reviewers:
  - "@andrewsykim"
  - "@cheftako"
  - "@nckturner"
approvers:
  - TBD
editor: TBD
creation-date: 2019-10-04
last-updated: 2019-10-16
status: provisional
---

# Out-of-Tree Credential Providers

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API Server Proxy](#api-server-proxy)
  - [Sidecar Credential Daemon](#sidecar-credential-daemon)
  - [External Non-Daemon Executable](#external-non-daemon-executable)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Replace the existing in-tree container image registry (registry) credential providers with an external and pluggable credential provider mechanism and remove in-tree credential providers.

## Motivation

kubelet uses cloud provider specific SDKs to obtain credentials when pulling container images from cloud provider specific registries. The use of cloud provider specific SDKs from within the main Kubernetes tree is deprecated by [KEP-0002](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/20180530-cloud-controller-manager.md) and all existing uses need to be migrated out-of-tree. This KEP supports that migration process by removing this SDK usage.

In addition to supporting cloud provider migration this KEP supports allowing users to support multiple container registries across different cloud providers by introducing a pluggable interface to cloud specific credential providers such that a single user could run multiple credential providers for different clouds at the same time. Currently this functionality is gated by having only a single, active cloud provider within the API server and kubelet.

### Goals

* Develop/test/release an API for kubelet to obtain registry credentials from the API server
* Update/test/release the credential acquisition logic within kubelet
* Build user documentation for out-of-tree credential providers
* Migrate existing in-tree credential providers to new credential provider interface
* Remove in-tree credential provider code from kuberentes core

### Non-Goals

* Broad removal of cloud SDK usage, this effort falls under the [KEP for removing in-tree providers](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/2019-01-25-removing-in-tree-providers.md).

## Proposal

### API Server Proxy

The API server will act as a proxy to an external container registry credential provider that may support multiple cloud providers. The credential provider service will return container runtime compatible responses of the type currently used by the credential provider infrastructure within the kubelet along with credential expiration information to allow the API server to cache credential responses for a period of time.

This limits the cloud-specific privileges required for each node for the purpose of fetching credentials. Centralized caching helps to avoid cloud-specific rate limits for credential acquisition by consolidating that credential acquisition within the API server.

### Sidecar Credential Daemon

Each node will run a sidecar credential daemon that can obtain cloud-specific container registry credentials and may support multiple cloud providers. This service will be available to the kubelet on the local host and will return container runtime responses compatible with those currently used by the credential provider infrastructure within kubelet. Each daemon will perform its own caching of credentials for the node on which it runs.

This architecture is similar to the approach taken by CSI with node plugins and is a well understood deployment pattern.It limits fleet-wide cacheability of credentials.

### External Non-Daemon Executable

An executable capable of providing container registry credentials will be installed on each node. This binary will be executed by the kubelet to obtain container registry credentials in a format compatible with container runtimes. Credential responses may be cached within the kubelet.

This Architecture is similar to the approach taken by CNI and is a well understood deployment pattern. It limits fleet-wide cacheability of credentials.

### Risks and Mitigations

TODO

## Design Details

### Test Plan

TODO

### Graduation Criteria

TODO

### Upgrade / Downgrade Strategy

TODO

### Version Skew Strategy

TODO

## Implementation History

TODO

## Infrastructure Needed

* New GitHub repos for existing credential providers (AWS, Azure, GCP)
