<!--
Last template update: 2025-09-11

https://github.com/kubernetes/enhancements/commit/3ffc27b7413e285d429025a422dd79473d3e9b50
(See https://github.com/kubernetes/enhancements/commits/master/keps/NNNN-kep-template/README.md
to check if there are newer changes)
-->
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
# KEP-2033: Kubelet-in-UserNS (aka Rootless mode)

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
  - [FAQ: why not use admission controllers?](#faq-why-not-use-admission-controllers)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Production cluster](#story-1-production-cluster)
    - [Story 2: HPC cluster](#story-2-hpc-cluster)
    - [Story 3: <code>kind</code> with Rootless Docker/Podman](#story-3-kind-with-rootless-dockerpodman)
    - [Story 4: Temporary initial cluster for bootstrapping](#story-4-temporary-initial-cluster-for-bootstrapping)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Running Kubernetes inside Rootless Docker/Podman (kind, minikube)](#running-kubernetes-inside-rootless-dockerpodman-kind-minikube)
  - [Running Kubernetes directly on the host](#running-kubernetes-directly-on-the-host)
    - [Paths](#paths)
    - [Network](#network)
    - [RootlessKit network drivers](#rootlesskit-network-drivers)
    - [CNI plugins](#cni-plugins)
    - [cgroup](#cgroup)
  - [Required changes to Kubernetes](#required-changes-to-kubernetes)
    - [kubelet](#kubelet)
    - [kube-proxy](#kube-proxy)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

This KEP allows running the entire Kubernetes components (`kubelet`, CRI, OCI, CNI, and all `kube-*`) as a non-root user on the host,
by running them in a user namespace.
See [Notes/Constraints/Caveats](#notesconstraintscaveats) for the caveats.

**TLDR**: Most things do work without modifying Kubernetes. But we need to modify a just few lines of kubelet and kube-proxy to ignore errors during setting some sysctl and rlimit values.
See ["Required changes to Kubernetes"](#required-changes-to-kubernetes).

Resources:
- POC: [Usernetes](https://github.com/rootless-containers/usernetes)
- A presentation at KubeCon NA 2020: https://sched.co/fGWc
- Kubernetes PR: https://github.com/kubernetes/kubernetes/pull/92863
- Rootless k3s: https://github.com/k3s-io/k3s/blob/master/k3s-rootless.service
- kind with Rootless Docker/Rootless Podman: https://kind.sigs.k8s.io/docs/user/rootless/
  (It already works with unmodified Kubernetes, but contains dirty hack to fake procfs)
- Proposal in minikube repo, for running Kubernetes in Rootless Docker: https://github.com/kubernetes/minikube/issues/9495
- Proposal in minikube repo, for running Kubernetes in Rootless Podman: https://github.com/kubernetes/minikube/issues/8719

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

* Protect the host from potential container-breakout vulnerabilities. This is the main motivation.
* Allow users of shared machines (especially HPC) to run Kubernetes without the risk of accidentally breaking their colleagues' environments.
  Not recommended for real multi-tenancy where the users cannot be trusted.
  * Safe [`kind`](https://kind.sigs.k8s.io/docs/user/rootless/): Kubernetes inside Rootless Docker/Podman.
  * Safe Kubernetes-on-Kubernetes, to isolate workloads more strictly than Kubernetes API namespaces.

### FAQ: why not use admission controllers?
Admission controllers like PSP can restrict containers to use extra security options like AppArmor/SELinux, gVisor/Kata, and also potentially [Node-level UserNS](https://github.com/kubernetes/enhancements/issues/127) in the future.

However, these are not efficient to mitigate vulnerabilities of the node components themselves (kubelet, CRI, OCI...).

e.g.
- [CVE-2017-1002102](https://nvd.nist.gov/vuln/detail/CVE-2017-1002102): kubelet could delete files on the host during syncing secret/configMap/downwardAPI volumes
- [CVE-2019-11245](https://nvd.nist.gov/vuln/detail/CVE-2019-11245): Dockerfile USER instruction was ignored by kubelet
- [CVE-2018-11235](https://nvd.nist.gov/vuln/detail/CVE-2018-11235): kubelet could execute an arbitrary command as the root via gitRepo volumes
- Potential image extraction [zip-slip](https://snyk.io/research/zip-slip-vulnerability) vulnerabilities in CRI runtimes. Both containerd and CRI-O are working on implementing supports for new archive formats like zstd, imgcrypt, and stargz. Potentially these implementations have such vulnerabilities.
- And lots of CRI/OCI vulnerabilities in the past.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Allow `kubelet` and `kube-proxy` to be executed inside user namespaces create by a non-root user. See ["Required changes to Kubernetes"](#required-changes-to-kubernetes).

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

[The Node-level UserNS KEP](https://github.com/kubernetes/enhancements/issues/127) is similar to this KEP, but out of scope for this KEP.

While Node-level UserNS executes only containers inside UserNS. this KEP executes all the node components inside UserNS to mitigate vulnerabilities of all components,

Node-level UserNS and this KEP do not conflict and can be stacked together. (Node-level UserNS inside Kubelet's UserNS.)

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1: Production cluster

A user is scared of the past vulnerabilities of kubelet/CRI/OCI, and looking for a way to
mitigate such potential vulnerabilities.

So the user would want this KEP to be implemented.

The user may face difficulties for deploying stateful workloads because block-based and NFS-based persistent volumes
mostly do not work (see [Notes/Constraints/Caveats](#notesconstraintscaveats)), but this is not a huge deal,
when the user can use managed object storages such as Amazon S3, or managed RDBs such as Amazon RDS for storing persistent data.

If the user really needs to run an application that requires the root privileges, the user
would create a mixed cluster composed of rootful nodes and rootless nodes, and set
the node selector to ensure the privileged pods to be scheduled on rootful nodes.
However, it is more preferable to create another cluster for rootful nodes.

#### Story 2: HPC cluster

A user wants to deploy a Kubernetes cluster using shared HPC machines to run scientific research workloads.

However, the machine administrator does not want to allow the user to gain the root privileges,
because the admin thinks that the user may accidentally break other users' environments.

And yet, the admin hesitates to deploy a shared Kubernetes cluster and to create RBAC-restricted
accounts for users, because user management in Kubernetes is very difficult.

The user would want this KEP to be implemented so that he/she can deploy Kubernetes without
convincing the admin.

#### Story 3: `kind` with Rootless Docker/Podman

A user wants to run a test cluster inside Docker/Podman on his/her laptop using `kind`.

However, the user doesn't want Kubernetes/kind/Docker/Podman to gain the root privileges
because these components may accidentally break the host environment,
e.g. Docker may modify the host iptables in an unexpected way and break the user's VPN connectivity.

The user would want this KEP to be implemented so that he/she could run `kind` with Rootless Docker/Podman,
which won't break the host.

#### Story 4: Temporary initial cluster for bootstrapping

A user needs a temporary initial cluster to bootstrap an actual cluster with Cluster API.

The user wants to avoid having the root privileges.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

TL;DR: Things that work with Rootless Docker 20.10 and Rootless Podman 2.1 will work with Rootless Kubernetes as well. Other things will not.

cgroup:
- No support for cgroup v1.
- [Hugepages](../1539-hugepages/) cannot be supported because systemd doesn't support delegation of the hugetlb controller: https://github.com/systemd/systemd/issues/16325
- Device controller cannot be supported as well, but it is not a huge deal, because non-root users don't have permission to access insecure devices anyway.

Network:
- kube-proxy needs the following `KubeProxyConfiguration` to avoid hitting errors during setting `sysctl` values:
```yaml
conntrack:
# Skip setting sysctl value "net.netfilter.nf_conntrack_max"
  maxPerCore: 0
# Skip setting "net.netfilter.nf_conntrack_tcp_timeout_established"
  tcpEstablishedTimeout: 0s
# Skip setting "net.netfilter.nf_conntrack_tcp_timeout_close"
  tcpCloseWaitTimeout: 0s
```
- Some CNI plugins might not work. Flannel (VXLAN) is known to work.
- Limited network performance due to the slirp4netns overhead.
  **Mitigation:** Install [`lxc-user-nic` (SETUID binary)](https://github.com/rootless-containers/rootlesskit/blob/v0.14.2/docs/network.md#--netlxc-user-nic-experimental).
- NodePort less than 1024 cannot be exposed. This is not a problem with the default `--service-node-port-range` configuration (30000-32767).
  **Mitigation:** set `CAP_NET_BIND_SERVICE` file capability on `rootlesskit` binary.

Volumes:
- Block device volumes and (kernel-mode) NFS does not work, because user namespace only supports `tmpfs`, `bind`, and FUSE filesystems.
  `emptyDir`, `hostPath`, `local`, and API volumes (`configMap`, `secret`, `downwardAPI`, ...) are known to work without any issue.
  FUSE-based CSI volumes can be supported, but not recommended.
  **Mitigation:** Use managed object storage services such as Amazon S3/Google Cloud Storage/Azure Blob Storage, or use managed database services for storing persistent data.

SecurityContext:
- A container with `securityContext.privileged` cannot gain the real root privileges, obviously.
- `runAsUser`: supported, but the number of the UID is limited by `/etc/subuid`.
- `sysctls`: some sysctl parameters are supported, but some would fail in `EPERM`.
  Creating a Pod manifest with such sysctl parameters would fail.
  If this behavior is problematic, user should write a Mutating Admission Webhook to remove such sysctl parameters from Pod manifests.
- seccomp: supported
- AppArmor: unsupported. Creating a Pod with an AppArmor profile would fail.
- SELinux: Same as Rootless Podman. Applying an existing profile would be ok, but creating a new profile would not.
- [Node-level UserNS KEP](https://github.com/kubernetes/enhancements/issues/127): can be supported. This UserNS will be nested inside Kubelet's UserNS.


### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

If Linux kernel had vulnerabilities in its user namespace implementation, the root in the user namespace might be able to
escape from the user namespace, and take the real root privilege of the host.

So, it is still preferred to run pods with sandbox technologies like gVisor to mitigate potential kernel vulnerabilities.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Running Kubernetes inside Rootless Docker/Podman (kind, minikube)
When Kubernetes is being executed inside Rootless Docker/Podman, the namespaces and cgroups are already configured by Docker/Podman.
So, basically there is no additional task, but we still have to modify a few lines of kubelet and kube-proxy to ignore
minor sysctl & rlimit errors. See ["Required changes to Kubernetes"](#required-changes-to-kubernetes).


It should be noted that [kind already works with unmodified Kubernetes](https://kind.sigs.k8s.io/docs/user/rootless/),
but kind currently uses [very dirty hack to mount fake files under `/proc/sys` to avoid hitting sysctl errors.](https://github.com/kubernetes-sigs/kind/commit/85d51d898a5b642034e908d2869f0ad196c7d052#diff-3c55751d83af635109cece495ee2ff38206764a8b95f4cb8f11fc08a5c0ea8dcR105).


### Running Kubernetes directly on the host
The node components need to be executed inside a user namespace along with other namespaces (mount namespace, network namespace, etc.)
to gain fake-root privileges, mostly for mount and network operations.

To run Rootless Kubernetes directly on the host, [RootlessKit](https://github.com/rootless-containers/rootlesskit) can be used for creating namespaces.
In a nutshell, RootlessKit is an extended version of [`unshare`](http://man7.org/linux/man-pages/man1/unshare.1.html) for rootless containers.
RootlessKit has been already adopted by Docker, BuildKit, Usernetes, k3s, and partially by Podman.

All Kubernetes components including CRI runtime, kubelet, kube-proxy, and CNI daemon need to be executed in RootlessKit's namespaces.
```console
$ rootlesskit --net=slirp4netns --copy-up=/etc --copy-up=/run --copy-up=/var --pidns --cgroupns --ipcns --utsns -- containerd &
$ nsenter -t $ROOTLESSKIT_CHILD_PID -a kubelet ... &
$ nsenter -t $ROOTLESSKIT_CHILD_PID -a kube-proxy ... &
$ nsenter -t $ROOTLESSKIT_CHILD_PID -a flanneld ... &
```

#### Paths

Some paths like `/var/log/pods` are hardcoded in Kubernetes and hard to change.

Although these directories are not writable by unprivileged users, Kubernetes does NOT need to be changed to use unprivileged home directories,
because RootlessKit can bind-mount writable directories on these paths without the root privileges. (`rootlesskit --copy-up=/var`)

#### Network
The node components need to be executed in RootlessKit's network namespace, because an unprivileged user cannot do privileged operations in the host network namespace.
As the components are executed inside a network namespace, `NodePorts` are not directly accessible from other hosts.

An external controller should watch changes on `corev1.Service` resources and call [RootlessKit API](https://github.com/rootless-containers/rootlesskit/blob/v0.11.1/pkg/api/openapi.yaml) to set up port forwarding for the node ports.

k3s implementation: https://github.com/rancher/k3s/blob/v1.17.2+k3s1/pkg/rootlessports/controller.go#L92-L96

#### RootlessKit network drivers

RootlessKit supports two kinds of network stacks:
* TAP with pure usermode network stack (either `slirp4netns` or VPNKit)
* vEth with setuid binary `lxc-user-nic`

`slirp4netns` is preferred for security, `lxc-user-nic` is preferred for performance.

These stacks are used for the namespace where the node components are executed in, not for the containers' namespaces.
CNI plugins such as Flannel are expected to be used for the containers' namespace.

#### CNI plugins

Flannel (VXLAN) is known to work.

#### cgroup

[cgroup v2](../2254-cgroup-v2/) and systemd are required. cgroup v1 won't be supported due to security concerns.

containerd supports cgroup v2 for rootless mode since containerd v1.4.
[The master branch of CRI-O](https://github.com/cri-o/cri-o/commit/d3dbaec060e33870e5cb5c3f7ec4207837804b00) also supports cgroup v2 for rootless mode.
It will be included in CRI-O v1.22.

No code change is required on kubelet for managing cgroups, because we can use cgroup namespaces along with mount namespaces for creating writable `/sys/fs/cgroup` filesystem.

### Required changes to Kubernetes

Most things do work without modifying Kubernetes. But we need to modify a just few lines of kubelet and kube-proxy to ignore errors during setting some sysctl and rlimit values.

#### kubelet
Patch: ["kubelet/cm: ignore sysctl error when running in userns"](https://github.com/rootless-containers/usernetes/blob/v20210303.0/src/patches/kubernetes/0001-kubelet-cm-ignore-sysctl-error-when-running-in-usern.patch)

The patch modifies `kubelet` to ignore errors that happens during setting the following sysctl keys:
- `vm.overcommit_memory`
- `vm.panic_on_oom`
- `kernel.panic`
- `kernel.panic_on_oops`
- `kernel.keys.root_maxkeys`
- `kernel.keys.root_maxbytes`

> **Note**
> These sysctl parameters are set for `kubelet` itself. These are unrelated to `.spec.securityContext.sysctls` in Pod manifests.

#### kube-proxy
Patch: ["kube-proxy: allow running in userns"](https://github.com/rootless-containers/usernetes/blob/v20210303.0/src/patches/kubernetes/0002-kube-proxy-allow-running-in-userns.patch)

The patch modifies `kube-proxy` to ignore an error during setting `RLIMIT_NOFILE`.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

Tests are present in several subproject repos and third party repos:
- https://github.com/kubernetes-sigs/kind/blob/v0.17.0/.github/workflows/cgroup2.yaml#L24
- https://github.com/kubernetes/minikube/blob/v1.29.0/.github/workflows/pr.yml#L293-L410
- https://github.com/k3s-io/k3s/blob/v1.26.1+k3s1/.github/workflows/cgroup.yaml#L92-L99
- https://github.com/rootless-containers/usernetes/blob/v20221007.0/.cirrus.yml

Tests will be added to `kubernetes/test-infra` as well when the [`k8s-infra-prow-build`](https://github.com/kubernetes/k8s.io/blob/a071c4ed0823f193ee29e2f14e191be42dc1a1f0/infra/gcp/terraform/k8s-infra-prow-build/main.tf#L78) cluster
is upgraded to use cgroup v2.
This will probably automatically happen when [GKE bumps up their "regular" channel to Kubernetes v1.26 or later](https://cloud.google.com/kubernetes-engine/docs/how-to/node-system-config).

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

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

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

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
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->


- Alpha: Basic support for rootless mode on cgroups v2 hosts.

- Beta: e2e tests coverage.
  To move to beta, we need clarity if we intend to define two separate types of conformance suites:
  - kubernetes clusters that can run privileged workloads
  - kubernetes cluster that are restricted to run unprivileged workloads only
  Requirements:
  - [the cgroup v2 KEP](../2254-cgroup-v2/) to reach Beta or GA.

- GA: Assuming no negative user feedback based on production experience, promote after >= 2 releases in beta.
  Requirements:
  -  [the cgroup v2 KEP](../2254-cgroup-v2/) to reach GA.

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

This feature is new, there is no upgrade path from existing nodes.

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

N/A

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

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `KubeletInUserNamespace`
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

Enabling `KubeletInUsernamespace` feature gate does not automatically execute kubelet in a user namespace.
The user namespace has to be created by RootlessKit before running kubelet.
For `kind` usecase, the namespace is provided by Rootless Docker or Rootless Podman (they internally use RootlessKit).

Note that this feature gate does not support separating kubelet's user namespace from the user namespace of other
node components such as CRI.
All the node components must run in the same user namespace.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

During Alpha, we will document what workloads will work and what will not work.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

N/A, as switching back rootless to rootful requires redeploying the kubelet, and vice versa.

###### What happens if we reenable the feature if it was previously rolled back?

N/A.

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

CI will run `kind` (Kubernetes in Docker) tests with Rootless Docker/Podman.
Tests with a real cluster will be added later as well.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

This section will be fulfilled when targeting beta graduation to a release.

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

N/A

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

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

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [X] Other (treat as last resort)
  - Details: Use `systemctl --user is-system-running` to verify whether the processes (RootlessKit, kubelet, kube-proxy, and CRI) are running.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

N/A

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

- Kernel: 5.2 or later is recommended. At least 4.15 or later is required. ([Reason](https://github.com/opencontainers/runc/blob/master/docs/cgroup-v2.md#host-requirements))
- Systemd: 244 or later is recommended.
- CRI: containerd >= 1.4, or CRI-O >= 1.22 is required.
- OCI: runc >= 1.0-rc91 is required. runc >= 1.0-rc93 is recommended. crun works, too.

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

 - [RootlessKit]
    - Usage description: sets up namespaces, and forwards incoming TCP & UDP packets
      - Impact of its outage on the feature: kubelet, kube-proxy, CRI, and all container processes will crash, and will be restarted by systemd.
      - Impact of its degraded performance or high-error rates on the feature: Incoming packet forwarding will be slow.
 - [slirp4netns]
    - Usage description: forwards outgoing TCP & UDP packets via a virtual router
      - Impact of its outage on the feature: Outgoing packets will be dropped.
      - Impact of its degraded performance or high-error rates on the feature: Outgoing packet forwarding will be slow.

When a cluster is being created in a `kind` container with Rootless Docker/Rootless Podman provider,
the user namespace is already created by Rootless Docker/Rootless Podman, so RootlessKit and slirp4netns do not need to be installed 
in the `kind` container.

Both Docker and Podman use RootlessKit and slirp4netns (or VPNkit, optionally) internally.

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

No.

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

No.

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

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

RootlessKit and slirp4netns may face high CPU and memory consumption.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

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

Same as traditional rootful Kubernetes.

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

Same as traditional rootful Kubernetes.

###### What steps should be taken if SLOs are not being met to determine the problem?

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

- 2018-07-20: Early POC implementation in [Usernetes project](https://github.com/rootless-containers/usernetes)
- 2019-04-10: [k3s adopted the Usernetes patches (cgroupless version)](https://github.com/rancher/k3s/pull/195)
- 2019-06-04: [Presented KEP to SIG-node (cgroupless version)](https://github.com/kubernetes/enhancements/pull/1084)
- 2019-07-08: Withdrew the cgroupless KEP
- 2019-11-19: @giuseppe submitted [cgroup v2 KEP](https://github.com/kubernetes/enhancements/pull/1370)
- 2019-11-19: present KEP to SIG-node (cgroup v2 version)
- 2020-07-07: the cgroup v2 support is in `implementable` status

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

The primary drawback of this KEP is its complexity.
It also heavily relies on third-party, out-of-tree components.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

[The Node-level UserNS KEP](https://github.com/kubernetes/enhancements/issues/127) is often considered to be an alternative,
but it is actually not, because it can't mitigate vulnerabilities of kubelet, CRI, OCI, and their relevant components.
See [Non-goals](#non-goals) section.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
CI infra for cgroup v2 is needed
