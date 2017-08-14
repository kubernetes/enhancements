# Checklist for SIGs and Release Team
As SIGs fill out their sections by component, please check off that
you are finished. For guidance about what should have a release note
please check out the [release notes guidance][] issue.

- [ ] sig-api-machinery
- [ ] sig-apps
- [ ] sig-architecture
- [ ] sig-auth
- [ ] sig-autoscaling
- [ ] sig-aws
- [ ] sig-azure
- [ ] sig-big-data
- [ ] sig-cli
- [ ] sig-cluster-lifecycle
- [ ] sig-cluster-ops
- [ ] sig-contributor-experience
- [ ] sig-docs
- [ ] sig-federation
- [ ] sig-governance.md
- [ ] sig-instrumentation
- [ ] sig-network
- [ ] sig-node
- [ ] sig-on-premise
- [ ] sig-openstack
- [ ] sig-product-management
- [ ] sig-release
- [ ] sig-scalability
- [ ] sig-scheduling
- [ ] sig-service-catalog
- [ ] sig-storage
- [ ] sig-testing
- [ ] sig-ui
- [ ] sig-windows

[release notes guidance]: https://github.com/kubernetes/community/issues/484

## **Major Themes**

## **Action Required Before Upgrading**

## **Known Issues**

## **Deprecations**

## **Notable Features**
#### Storage

* Capacity Isolation/Resource Management for Local Ephemeral Storage
* Block Volumes Support
* Enable containerization of mount dependencies
* Support Attach/Detach for RWO volumes such as iSCSI, Fibre Channel and RBD
* Volume Plugin Metrics
* Snapshots
* Resizing Volume Support
* Exposing StorageClass Params To End Users (aka Provisioning configuration in PVC)
* Mount Options to GA
* Allow configuration of reclaim policy in StorageClass
* Expose Storage Usage Metrics
* PV spec refactoring for plugins that reference namespaced resources: Azure File, CephFS, iSCSI, Glusterfs

### **Node Components**
#### kubelet
* [alpha] Add limited support for pod "checkpointing" in the kubelet to help enable self-hosting. ([#378](https://github.com/kubernetes/features/issues/378), [@timothysc](https://github.com/timothysc))


### **Cluster Lifecycle**
#### kubeadm
* [beta] A new `phase` subcommand supports performing only subtasks of the full `kubeadm init` flow. Combined with fine-grained configuration, kubeadm is now more easily consumable by higher-level provisioning tools like kops or GKE. ([#356](https://github.com/kubernetes/features/issues/356), [@luxas](https://github.com/luxas), [@justinsb](https://github.com/justinsb))

* [alpha] A new `upgrade` subcommand allows you to automatically upgrade a self-hosted cluster created with kubeadm. ([#296](https://github.com/kubernetes/features/issues/296), [@luxas](https://github.com/luxas))

#### kops
* [alpha] Added support for targeting bare metal (or non-cloudprovider) machines. ([#360](https://github.com/kubernetes/features/issues/360), [@justinsb](https://github.com/justinsb)).

* [alpha] kops now supports [running as a server](https://github.com/kubernetes/kops/blob/master/docs/api-server/README.md). ([#359](https://github.com/kubernetes/features/issues/359), [@justinsb](https://github.com/justinsb)).

* [beta] GCE support has been promoted from alpha to beta. ([#358](https://github.com/kubernetes/features/issues/358), [@justinsb](https://github.com/justinsb)).

#### Cluster Discovery/Bootstrap

* [beta] A new authentication and verification mechanism called Bootstrap Tokens has been added to the core API, which can be used to easily add new members to a cluster. ([#130](https://github.com/kubernetes/features/issues/130), [@luxas](https://github.com/luxas), [@jbeda](https://github.com/jbeda)).

#### Addons
* [alpha] A new system addon manager is available that is aiming to improve the downsides of the existing `kube-addons.sh` manager. ([#18](https://github.com/kubernetes/features/issues/18), [@justinsb](https://github.com/justinsb))

#### Multi-platform
* [beta] Kubernetes now has automated continuous-integration tests against all of our supported platforms (amd64, armhfp, aarch64, ppc64le, s390x), to ensure that it continues to work on these platforms. It's also possible to run clusters with nodes of mixed architectures. ([#288](https://github.com/kubernetes/features/issues/288), [@luxas](https://github.com/luxas), [@mkumatag](https://github.com/mkumatag), [@ixdy](https://github.com/ixdy))

#### Cloud Providers
* [beta] Support for out-of-tree and out-of-process cloud providers, a.k.a pluggable providers, has been promoted from alpha to beta. ([#88](https://github.com/kubernetes/features/issues/88), [@wlan0](https://github.com/wlan0))

#### DaemonSet
* [beta] DaemonSet upgrades can be achieved via a start-then-kill update strategy. ([#373](https://github.com/kubernetes/features/issues/373), [@aaronlevy](https://github.com/aaronlevy), [@diegs](https://github.com/diegs))
