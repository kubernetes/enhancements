---
title: Rebase Kubernetes Main Master and Node Images to Distroless/static
authors:
  - "@yuwenma"
owning-sig: sig-release
participating-sigs:
  - sig-release
  - sig-cloud-provider
reviewers:
  - "@tallclair"
approvers:
  - "@tallclair"
editor: yuwenma
creation-date: 2019-03-16
last-updated: 2019-03-21
status: implementable
---

# Rebase Kubernetes Main Master and Node Images to Distroless/static

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
  - [Kubernetes Images](#kubernetes-images)
    - [Type 1 FROM Scratch](#type-1-from-scratch)
    - [Type 2 Debian Based Images](#type-2-debian-based-images)
    - [Type 3 Alpine Based Images](#type-3-alpine-based-images)
  - [Distroless and Previous Work](#distroless-and-previous-work)
- [Proposal](#proposal)
  - [For Core Master Images](#for-core-master-images)
    - [<a href="https://github.com/kubernetes/kubernetes/blob/caf9d94d697ce327e0c1c3dee71a1f06a6fc918e/build/root/Makefile#L419">Bash Release</a>](#bash-release)
    - [<a href="https://github.com/kubernetes/kubernetes/blob/caf9d94d697ce327e0c1c3dee71a1f06a6fc918e/build/root/Makefile#L604">Bazel Release</a>](#bazel-release)
    - [<a href="https://github.com/kubernetes/test-infra/tree/master/kubetest">Test Release</a>](#test-release)
  - [Solution](#solution)
    - [Notifications to Cloud Providers](#notifications-to-cloud-providers)
  - [For Generic Add-On Images](#for-generic-add-on-images)
    - [Example](#example)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
  - [Status Updates](#status-updates)
  - [Further work](#further-work)
  - [Rebased Images](#rebased-images)
<!-- /toc -->

## Summary

The effort of rebasing the k8s images to distroless/static is aimed at making the k8s images thinner, safer and less vulnerable. The scope is not only improving the core containers but will cover the master and node addons which have their own release process. As for the core containers, this effort is targeting the v1.15 release.

## Motivation

Rebasing the k8s images to distroless/static can make the images thinner, safer and less vulnerable.

Meanwhile, it will drastically reduce churn on the total number of k8s images versions. Due to the fact that many images are based on debian base and a vulnerability in debian base (a couple times a month) will result in rebuilding every image, changing the image from debian base to distroless/static can reduce the total number of k8s image versions.

What's more, it reduces the burden of managing and maintaining multiple k8s images from the security (e.g. CVE), compatibility and build process concerns.

### Goals

Use image gcr.io/distroless/static:latest as the only base image for the following kubernetes images

- Images based FROM scratch
- Images based on debian/alpine and only for the purpose of redirecting logs with shell.
- Images based on k8s.gcr.io/debian-base due to previous rebasing from busybox.

Help the community and contributors better understanding and maintaining the images.

- Set up the policy that only `distroless/static` and `k8s.gcr.io/debian-base` are used (as the base image) for the images hosted in the official k8s.gcr.io image repository. And if the image is based on debian-base, it should be documented in the exception list.
- Improve the presubmit prow test to guarantee that the upcoming k8s/kubernetes PRs won't introduce dependencies that distroless/static doesn't support.
- Document the base image list for important kubernetes components, including both core containers and important add-ons. Also, document the exception list (unable to base on distroless).


### Non-Goals

- Do not change Images based on debian/alpine that requires fluentd (e.g. hyperkube).
- Do not change images that have hard dependencies on non-static binaries.

## Background

This section discusses how the goal and scope are determined due to the reality. It also contains the real use cases.

### Kubernetes Images

Kubernetes not only runs images in the containers, but its components themselves are running and deploying as images. Each component image can be built from different base images.

Currently, kubernetes uses three main types of base images to build their components.

#### Type 1 FROM Scratch

This docker image is based “FROM scratch” and doesn’t have external dependencies. The original motivation of using “FROM scratch” is to keep the image extremely thin and only contain what a static binaries need. However, caveats are found when running the go static binaries due to some missing non-binary dependencies like ca-certificates, resolv.conf, hosts, and nsswitch (see [issue/69195](https://github.com/kubernetes/kubernetes/issues/69195)).

#### Type 2 Debian Based Images

An image can be based from Debian due to different reasons.

- One big reason is that the image needs to use shell to redirect the glog. This base image now is an overkill because K8s 1.13 can support using klog which accepts a --log-file flag to point to the log path directly. (Historically images doing this mostly relied on busybox or alpine. Some recent change has migrated off those to debian-base. [PR/70245](https://github.com/kubernetes/kubernetes/pull/70245/files))
- Another reason of using debian is from the CVE concerns. Those images are originally rebased from busybox to debian for better CVE feeds and management (See [PR/70245](https://github.com/kubernetes/kubernetes/pull/70245)).
- A third type of images uses debian for certain external dependencies.

#### Type 3 Alpine Based Images

The reasons for images based on alpine are similar to the ones on debian. Debian is more widely used due to previous “Establish base image policy for system containers” effort (see [issue/40248](https://github.com/kubernetes/kubernetes/issues/40248)).

### Distroless and Previous Work

"Distroless images contain only your application and its runtime dependencies. They do not contain package managers, shells or any other programs you would expect to find in a standard Linux distribution.”  (See [distroless/README](https://github.com/GoogleContainerTools/distroless))
Distroless supports the dependencies where “FROM scratch” misses and more light-weighted than debian or alpine. Meanwhile, distroless “improves the signal to noise of scanners (e.g. CVE) and reduces the burden of establishing provenance to just what you need.”(from [distroless/README](https://github.com/GoogleContainerTools/distroless))

Using Distroless/static as a common image base is originally proposed as an exploration area in [the base image policy for system containers](https://github.com/kubernetes/kubernetes/issues/40248). Tim(tallclair@) has driven the effort on defining and establishing the base image policy (main changes):

* Add Alpine iptable as base image for kube-proxy. Previously kube-proxy is based on debian iptable image.  (This direction is scrapped, see [issue/39696](https://github.com/kubernetes/kubernetes/issues/39696) for details)
* Rebase busybox images to debian-base.
* Rebase certain alpine images to debian-base

The distroless/static solution is filed separately in [issue/70249](https://github.com/kubernetes/kubernetes/issues/70249). This kep, as a more up-to-date version, is slightly different than the original issue.

## Proposal

The approaches to rebase different containers can vary significantly due to the function of the containers, the cloud-providers’ release workflows, and legacy reasons (repo migration plans, retirement plans, etc). This section will discuss 4 main types of image rebasing strategies, and this should cover the majority of the kubernetes containers.

1. The images are built via bazel. In such case, we will update the bazel BUILD rule to switch to the base image to distroless/static. This method applies for the core containers like kube-apiserver, kube-controller-manager, cloud-controller-manager, kube-scheduler. (See detailed solution in [Core Master Images](##for-core-master-images))
2. The images have dependencies that are not supported by distroless/static. One typical example is the usage of shell. Previously, shell is widely used for redirecting glog output to a certain directory. This use case is no longer needed since we've switched from glog to klog which can accept a flag to specify the log output path. A generic approach is: Remove the dependencies that distroless/static doesn't support and then rebase the images to distroless (e.g. [issue/1787](https://github.com/kubernetes/autoscaler/issues/1787). Meanwhile, we limit the introduction of new dependencies. (e.g. [pr/74690](https://github.com/kubernetes/kubernetes/pull/74690#discussion_r266037189))
3. Images based "FROM scratch" is safe to switch to distroless/static directly.
4. Images from kubernetes incubator won't be changed directly by this KEP and the release plan is not estimated here. We notify the project OWNERs and we defer to the OWNERs on whether or not those images should be updated.

### For Core Master Images

The core master images includes `kube-apiserver`, `kube-controller-manager`, `kube-scheduler` and `kube-proxy`. For `kube-proxy`, it is based on debian-iptable which distroless/static doesn't support iptables yet. Thus, kube-proxy won't be changed.

Currently, there are **three** different workflows to build the core master images and which workflow to use  is determined by each **cloud provider**.

#### [Bash Release](https://github.com/kubernetes/kubernetes/blob/caf9d94d697ce327e0c1c3dee71a1f06a6fc918e/build/root/Makefile#L419)

Run `make release` under kubernetes repo. This approach is most commonly used and it uses the bash scripts (See [build/release.sh](https://github.com/kubernetes/kubernetes/blob/caf9d94d697ce327e0c1c3dee71a1f06a6fc918e/build/release.sh#L36) for details) to build the images. In this workflow, the base image is specified in the [`build/common.sh`](https://github.com/kubernetes/kubernetes/blob/f26048ecb1c7b6fb67c2e7c7c96070d7a1743d86/build/common.sh#L96).

#### [Bazel Release](https://github.com/kubernetes/kubernetes/blob/caf9d94d697ce327e0c1c3dee71a1f06a6fc918e/build/root/Makefile#L604)

Run `make bazel-release` under kubernetes repo. This approach uses bazel to build the image artifact based on this [BUILD](https://github.com/kubernetes/kubernetes/blob/caf9d94d697ce327e0c1c3dee71a1f06a6fc918e/build/release-tars/BUILD) rule (More details in [bazel.bzl/release-filegroup](https://github.com/kubernetes/repo-infra/blob/4528e18f5d62a2a5172f76d198738d85d4d04734/defs/build.bzl#L134)) . In this workflow, the base image is specified in the [build/BUILD](https://github.com/kubernetes/kubernetes/blob/3fd6f97f55d51c01df3c01c7ffbb2834c25d9900/build/BUILD#L31) rule.

#### [Test Release](https://github.com/kubernetes/test-infra/tree/master/kubetest)

Run `kubetest` or `hack/e2e`. See details in the [test-infra repo](https://github.com/kubernetes/test-infra/tree/master/kubetest). This approach is recommended for development testing and is broadly used by contributors. However, this approach is under a test env and it uses different config than the two official workflows as described above. In this workflow, the base image is specified to use k8s.gcr.io/pause:3.1.

### Solution

This KEP is expected to rebase images for all three workflows. This requires each cloud provider team to be involved in the manifest updates and release workflow testing part (See the graph below). Before we switch the base images to `distroless/static`, each cloud provider team should make sure their manifest config is updated so that the **command doesn’t require shell to run the executable binaries and no log redirection is involved in the command**. Otherwise rebasing images to distroless will **break** the core containers running in the cluster master VMs. The test release should also be updated to `distroless/static` so as we can guarantee further changes wouldn’t be able to add unexpected dependencies (otherwise, they will fail the e2e tests in the github prow test stage).

![Rebase Core Master](RebaseCoreMaster.png?raw=true "Rebase Core Master")

#### Notifications to Cloud Providers

- For log redirection, please use flag [`log-file`](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver) (e.g. `--log-file=/var/log/kube-controller-manager.log`) and also disable standard output (e.g. `--logtostderr=false`)
- When removing the shell from manifest command, please also update the parameter format to exec form.` [“executable”, “param1, “param2”]`
- See example in [PR/75624](https://github.com/kubernetes/kubernetes/pull/75624)
- Detailed timelines about switching to the distroless/static will be announced later on. Please make sure manifest change is well tested in the release workflow (as shown in the right blue part).

### For Generic Add-On Images

It more or less depends on add-on OWNERs’ judge on whether/how the add-on images should be rebased. The below progress is what we proposed to the OWNERs. This should apply for most use cases.

1. (If the images depends on a k8s version that is earlier than v1.13) Sync up with current k8s head. Since kubernetes 1.13,oss kubernetes no longer uses glog which requires shell to redirect the log file. Instead, k8s is using klog which accepts a log path flag. This sync-up is necessary to remove log redirection.
2. (If the images use glog) Replace the glog to klog inside the add-on files.
3. Update the base image to distroless and remove distroless-preinstalled packages like ca-certificate.
4. (If necessary) Update the container upstart command to avoid using bash command. (For log redirection, see examples in the *For Core Master Images* section).
5. If bash scripts can’t be easily removed, document the container as exception in **[this list](https://github.com/kubernetes/sig-release/blob/master/release-engineering/baseimage-exception-list.md)**
6. After the above steps are done, require release engineers' help on monitoring the performance.

#### Example

[ingress-gce/fuzzer](https://github.com/kubernetes/ingress-gce/blob/64eee7e3521680057b071d5e9bebaa215086a4bc/Dockerfile.fuzzer) was based on alpine and can't be switched to distroless directly due to the fact that it needs shell to redirect the glog file. To allow the images to be based on distroless (which doesn't contain shell), we firstly need to remove the dependency on the shell (use klog instead of glog), and then rebase the image. (Related PR [pr/682](https://github.com/kubernetes/ingress-gce/pull/682), [pr/666](https://github.com/kubernetes/ingress-gce/pull/666))


## Graduation Criteria

This KEP is targeted at v1.15 release. The full list of images switched to distroless/static will be updated later on.

## Implementation History

### Status Updates
- Rebased the [following images](#rebased-images) to `gcr.io/distroless/static:latest` or `k8s.gcr.io/debian-base:v1.0.0`.
- Investigated [these images](https://github.com/kubernetes/sig-release/blob/master/release-engineering/baseimage-exception-list.md) as exceptions (can't based on distroless).
- Triaged and fixed the following issues which blocked rebasing images:
  * [Avoid log truncation due to log-rotation delays](https://github.com/kubernetes/klog/issues/55#issuecomment-481032528)
  * [Log duplication with --log-file](https://github.com/kubernetes/klog/pull/65)
  * [Fix MakeFile push workflow for metrics-server](https://github.com/kubernetes-incubator/metrics-server/pull/259)

### Further work
- Triage klog for the performance regression on core master containers:
  * Affected images: kube-controller-manager, kube-scheduler, kube-apiserver
  * Blocked PRs:
     - [Update manifests to use klog --log-file](https://github.com/kubernetes/kubernetes/pull/78466)
     - [Rebase core master images for both bazel and bash release](https://github.com/kubernetes/kubernetes/pull/75306)
- Avoid using exec in kube-controller-manager for flexvolume.

### Rebased Images

| Component Name        |   on Master/Node         |  Previous Image --> Current Image   |      Image      |   Code Complete  |   Release Complete |    Contact        |
| --------------------- | :-----------------------:|:-----------------------------------:|:---------------:|:----------------:|:------------------:|:------------------|
|     addon-resize      |  Master + Node           |        Busybox  --> distroless      |  k8s.gcr.io/addon-resizer:1.8.5 |      Done        |   Done           |    @bskiba @yuwenma |
|     cluster-proportional-autoscaler  |  Master + Node   |        scratch  --> distroless      |  k8s.gcr.io/cluster-proportional-autoscaler-arm:v1.6.0 |  Done | Done |  @yuwenma @MrHohn |
|     cluster-proportional-vertical-autoscaler  |  Master + Node   |        scratch  --> distroless      |  k8s.gcr.io/cpvpa-amd64:v0.7.1 |  Done | Done |  @yuwenma @MrHohn |
|     event-exporter  |  Master + Node   |        debian-base  --> distroless      |  k8s.gcr.io/event-exporter:v0.2.5 |  Done | Done |  @x13n @yuwenma |
|     node-termination-handler  |  Master + Node   |        alpine  --> distroless      |  [k8s.gcr.io/gke-node-termination-handler](https://pantheon.corp.google.com/gcr/images/google-containers/GLOBAL/gke-node-termination-handler@sha256:aca12d17b222dfed755e28a44d92721e477915fb73211d0a0f8925a1fa847cca/details?tab=info) |  Done | Done |  @yuwenma |
|     metadata-proxy  |  Master + Node   |        scratch  --> distroless       |  k8s.gcr.io/metadata-proxy:v0.1.12 |  Done | Done |  @dekkagaijin @yuwenma |
|     metrics-server  |  Master + Node   |        busybox  --> distroless       |  k8s.gcr.io/metrics-server:v0.3.3 |  Done | Done |  @yuwenma @kawych  |
|     prometheus-to-sd  |  Master + Node   |        debian-base  --> distroless       |  k8s.gcr.io/metrics-server:v0.5.2 |  Done | Done |  @loburm   |
|     ip-masq-agent  |  Master + Node   |         busybox  --> debian-iptables       |  k8s.gcr.io/ip-masq-agent:v2.4.1 |  Done | Done |  @BenTheElder @yuwenma   |
|     slo-monitor  |  Master   |        alpine  --> distroless       |  k8s.gcr.io/slo-monitor:0.11.2 |  Done | Done |  @yuwenma   |
|     kubelet-to-gcm   |  Master   |        scratch  --> distroless       |  k8s.gcr.io/kubelet-to-gcm:v1.2.11 |  Done | wait for next release |  @yuwenma |
|     etcd-version-monitor   |  Master   |        scratch  --> distroless       |  k8s.gcr.io/etcd-version-monitor:v0.1.3 |  Done | Done |  @yuwenma |
|     etcd-empty-dir-cleanup   |  Master   |        busybox  --> distroless       |  k8s.gcr.io/etcd-empty-dir-cleanup:3.3.10.1 |  Done | Done |  @yuwenma |
|     etcd |  Master   |        busybox  --> distroless       |  k8s.gcr.io/etcd:3.3.10-1 |  Done | Done |  @yuwenma |
|     defaultbackend  |  Master + Node   | scratch  --> distroless  | Wait for next release  |  Done | targeting v1.16 | @rramkumar1 @yuwenma  |
|     fuzzer   |  Master + Node   | alpine  --> distroless  | Wait for next release  |  Done | targeting v1.16 |  @rramkumar1 @yuwenma   |
|     ingress-gce-glbc  |  Master + Node   | alpine  --> distroless  | Wait for next release |  Done | targeting v1.16 | @rramkumar1 @yuwenma    |
|     k8s-dns-kube-dns  |  Master + Node   | alpine  --> debian-base  | k8s.gcr.io/k8s-dns-kube-dns:1.15.3 |  Done | Done | @yuwenma @prameshj  |
|     k8s-dns-sidecar  |  Master + Node   | alpine  --> debian-base  | k8s.gcr.io/k8s-dns-sidecar:1.15.3 |  Done |Done  | @yuwenma @prameshj  |
|     k8s-dns-dnsmasq-nanny |  Master + Node   | alpine  --> debian-base  | k8s.gcr.io/k8s-dns-dnsmasq-nanny:1.15.3  |  Done | Done   | @yuwenma @prameshj  |
|     k8s-dns-node-cache |  Node   | debian:stable-slim  --> debian-base  | k8s.gcr.io/k8s-dns-node-cache:1.15.3 |  Done | Done | @yuwenma @prameshj  |
|     cluster-autoscaler |  Master   | debian-base  --> distroless  | k8s.gcr.io/cluster-autoscaler:v1.16.0 |  Done | Done | @losipiuk  |
