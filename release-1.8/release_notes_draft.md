# Checklist for SIGs and Release Team
As SIGs fill out their sections by component, please check off that
you are finished. For guidance about what should have a release note
please check out the [release notes guidance][] issue.

- [ ] sig-api-machinery
- [ ] sig-apps
- [ ] sig-architecture
- [x] sig-auth
- [ ] sig-autoscaling
- [ ] sig-aws
- [ ] sig-azure
- [ ] sig-big-data
- [ ] sig-cli
- [x] sig-cluster-lifecycle
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

- The kubernetes workloads API (the DaemonSet, Deployment, ReplicaSet, and
StatefulSet kinds) have been moved to the new apps/v1beta2 group version. This
is the current version of the API, and the version we intend to promote to 
GA in future releases. This version of the API introduces several deprecations 
and behavioral changes, but its intention is to provide a stable, consistent 
API surface for promotion.

- The roles based access control (RBAC) API group for managing API authorization
has been promoted to v1. No changes were made to the API from v1beta1. This
promotion indicates RBAC's production readiness and adoption. Today, the
authorizer is turned on by default by many distributions of Kubernetes, and is a
fundamental aspect of a secure cluster.

## **Action Required Before Upgrading**

* The autoscaling/v2alpha1 API has graduated to autoscaling/v2beta1.  The
  form remains unchanged.  HorizontalPodAutoscalers making use of features
  from the autoscaling/v2alpha1 API will need to be migrated to
  autoscaling/v2beta1 to ensure that the new features are properly
  persisted.

* The metrics APIs (`custom.metrics.metrics.k8s.io` and `metrics`) have
  graduated from `v1alpha1` to `v1beta1`.  If you have deployed a custom
  metrics adapter, ensure that it supports the new API version.  If you
  have deployed Heapster in aggregated API server mode, ensure that you
  upgrade Heapster as well.

* Advanced auditing has graduated from `v1alpha1` to `v1beta1` with the
  following changes to the default behavior.
  * The webhook and log file now output the `v1beta1` event format.
  * The audit log file defaults to JSON encoding when using the advanced
    auditing feature gate.
  * The`--audit-policy-file` requires `kind` and `apiVersion` fields
    specifying what format version the `Policy` is using.

## **Known Issues**

## **Deprecations**

### Apps
 - The .spec.rollbackTo field of the Deployment kind is deprecated in the
 extensions/v1beta1 group version.
 - The pod.alpha.kubernetes.io/initialized has been removed.

### Scheduling
- Opaque Integer Resources (OIRs) are deprecated and will be removed in
  v1.9. Extended Resources (ERs) are a drop-in replacement for OIRs. Users can use
  any domain name prefix outside of the `kubernetes.io/` domain instead of the
  previous `pod.alpha.kubernetes.io/opaque-int-resource-` prefix.

### Auth
- With the introduction of RBAC v1, the RBAC v1alpha1 API group has been deprecated.

### Cluster Lifecycle

- The `auto-detect` behavior of the kubelet's `--cloud-provider` flag is deprecated.
  - In v1.8, the default value for the kubelet's `--cloud-provider` flag is `auto-detect`. It only works on a few cloud providers though.
  - In v1.9, the default will be `""`, which means no built-in cloud provider extension will be enabled by default.
  - If you want to use an out-of-tree cloud provider in either version, you should use `--cloud-provider=external`
  - [PR #51312](https://github.com/kubernetes/kubernetes/pull/51312) and [announcement](https://groups.google.com/forum/#!topic/kubernetes-dev/UAxwa2inbTA)

## **Notable Features**

### [Workload API (apps/v1beta2)](https://github.com/kubernetes/features/issues/353)

Kubernetes 1.8 adds the apps/v1beta2 group version. This group version contains
the Kubernetes workload API which consists of the DaemonSet, Deployment,
ReplicaSet and StatefulSet kinds. It is the current version of the API, and we
intend to promote it to GA in upcoming releases

#### API Object Additions and Migrations

- The current version DaemonSet, Deployment, ReplicaSet, and StatefulSet kinds
  are now in the apps/v1beta2 group version.
- The apps/v1beta2 group version adds a Scale subresource for the StatefulSet
kind.
- All kinds in the apps/v1beta2 group version add a corresponding conditions
  kind.

#### Behavioral Changes

 - For all kinds in the API group version, as it is incompatible with kubectl
 apply and strategic merge patch, spec.selector defaulting is disabled. Users
 must set the spec.selector in their manifests, and the creation of an object
 with a spec.selector that does not match the labels in its spec.template is
 considered to be invalid.
 - As none of the controllers in the workloads API handle selector mutation in
 a consistent way, selector mutation is disabled in for all kinds in the
 app/v1beta2 group version. This restriction may be lifted in the future, but
 it is likely that that selectors will remain immutable after GA promotion.
 Users that have any code that depends on mutable selectors may continue to use
 the apps/v1beta1 API for this release, but they should begin migration to code
 that does depend on mutable selectors.
 - Extended Resources are fully-qualified resource names outside the
 `kubernetes.io` domain. Extended Resource quantities must be integers.
 Users can use any resource name of the form `[aaa.]my-domain.bbb/ccc`
 in place of [Opaque Integer Resources](https://v1-6.docs.kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#opaque-integer-resources-alpha-feature).
 Extended resources cannot be overcommitted, so request and limit must be equal
 if both are present in a container spec.
 - The default Bootstrap Token created with `kubeadm init` v1.8 expires
 and is deleted after 24 hours by default to limit the exposure of the
 valuable credential. You can create a new Bootstrap Token with
 `kubeadm token create` or make the default token permanently valid by specifying
 `--token-ttl 0` to `kubeadm init`. The default token can later be deleted with
 `kubeadm token delete`.
 - `kubeadm join` now delegates TLS Bootstrapping to the kubelet itself, instead
 of reimplementing that process. `kubeadm join` writes the bootstrap KubeConfig
 file to `/etc/kubernetes/bootstrap-kubelet.conf`.

 #### Defaults

 - The default spec.updateStrategy for the StatefulSet and DaemonSet kinds is
 RollingUpdate for the apps/v1beta2 group version. Users may specifically set
 the OnDelete strategy, and no strategy auto-conversion will be applied to
 replace defaulted values.
 - As mentioned in [Behavioral Changes](#behavioral-changes), selector
 defaulting is disabled.
 - The default spec.revisionHistoryLimit for all applicable kinds in the
 apps/v1beta2 group version has set to 10.

#### CLI Changes

- The kubectl rollout and rollback implementation is complete for StatefulSet.
- The kubectl scale command will uses the Scale subresource for kinds in the
  apps/v1beta2 group.
- kubectl delete will no longer scale down workload API objects prior to
  deletion. Users who depend on ordered termination for the Pods of their
  StatefulSet’s must use kubectl scale to scale down the StatefulSet prior to
  deletion.
- `kubectl create configmap` and `kubectl create secret` subcommands now support the `--append-hash` flag, which enables unique yet deterministic naming for objects generated from files, e.g. via `--from-file`.

#### Scheduling
* [alpha] Support pod priority and creation of PriorityClasses ([design doc](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/pod-priority-api.md))
* [alpha] Support priority-based preemption of pods ([design doc](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/pod-preemption.md))
* [alpha] Taint nodes by condition ([design doc](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/taint-node-by-condition.md))

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
* [alpha] Kubelet now supports alternative container-level CPU affinity policies using the new CPU manager. ([#375](https://github.com/kubernetes/features/issues/375), [@sjenning](https://github.com/sjenning), [@ConnorDoyle](https://github.com/ConnorDoyle))

* [alpha] Applications may now request pre-allocated hugepages by using the new `hugepages` resource in the container resource requests. ([#275](https://github.com/kubernetes/features/issues/275), [@derekwaynecarr](https://github.com/derekwaynecarr))

* [alpha] Add support for dynamic Kubelet configuration ([#281](https://github.com/kubernetes/features/issues/281), [@mtaufen](https://github.com/mtaufen))

#### Autoscaling and Metrics

* Support for custom metrics in the Horizontal Pod Autoscaler is moving to
  beta.  The associated metrics APIs (custom metrics and resource/master
  metrics) are graduating to v1beta1.  See [Action Required Before
  Upgrading](#action-required-before-upgrading).

### Auth

* [GA] The RBAC API group has been promoted from v1beta1 to v1. No API changes were introduced.
* [beta] Advanced auditing has been promoted from alpha to beta. The webhook and logging policy formats have changed since alpha, and may require modification.
* [beta] Kubelet certificate rotation through the certificates API has been promoted from alpha to beta. RBAC cluster roles for the certificates controller have been added for common uses of the certificates API, such as the kubelet's.
* [beta] SelfSubjectRulesReview, an API that lets a user see what actions they can perform with a namespace, has been added to the authorization.k8s.io API group. This bulk query is intended to enable UIs to show/hide actions based on the end user, and for users to quickly reason about their own permissions.
* [alpha] Building on the 1.7 work to allow encryption of resources such as secrets, a mechanism to store resource encryption keys in external Key Management Systems (KMS) was introduced. This complements the original file-based storage and allows integration with multiple KMS. A Google Cloud KMS plugin was added and will be usable once the Google side of the integration is complete.

### **Cluster Lifecycle**

#### kubeadm

* [beta] A new `upgrade` subcommand allows you to automatically upgrade a self-hosted cluster created with kubeadm. ([#296](https://github.com/kubernetes/features/issues/296), [@luxas](https://github.com/luxas))

* [alpha] An experimental self-hosted cluster can now easily be created with `kubeadm init`. Enable the feature by setting the SelfHosting feature gate to true: `--feature-gates=SelfHosting=true` ([#296](https://github.com/kubernetes/features/issues/296), [@luxas](https://github.com/luxas))
   * **NOTE:** Self-hosting will be the default way to host the control plane in the next release, v1.9

* [alpha] A new `phase` subcommand supports performing only subtasks of the full `kubeadm init` flow. Combined with fine-grained configuration, kubeadm is now more easily consumable by higher-level provisioning tools like kops or GKE. ([#356](https://github.com/kubernetes/features/issues/356), [@luxas](https://github.com/luxas))
   * **NOTE:** This command is currently staged under `kubeadm alpha phase` and will be graduated to top level in a future release.

#### kops

* [alpha] Added support for targeting bare metal (or non-cloudprovider) machines. ([#360](https://github.com/kubernetes/features/issues/360), [@justinsb](https://github.com/justinsb)).

* [alpha] kops now supports [running as a server](https://github.com/kubernetes/kops/blob/master/docs/api-server/README.md). ([#359](https://github.com/kubernetes/features/issues/359), [@justinsb](https://github.com/justinsb)).

* [beta] GCE support has been promoted from alpha to beta. ([#358](https://github.com/kubernetes/features/issues/358), [@justinsb](https://github.com/justinsb)).

#### Cluster Discovery/Bootstrap

* [beta] The authentication and verification mechanism called Bootstrap Tokens has been improved. Use Bootstrap Tokens to add new node identities to a cluster easily. ([#130](https://github.com/kubernetes/features/issues/130), [@luxas](https://github.com/luxas), [@jbeda](https://github.com/jbeda)).

#### Multi-platform

* [alpha] The Conformance e2e test suite now passes on the arm, arm64, and ppc64le platforms. ([#288](https://github.com/kubernetes/features/issues/288), [@luxas](https://github.com/luxas), [@mkumatag](https://github.com/mkumatag), [@ixdy](https://github.com/ixdy))

#### Cloud Providers

* [alpha] Support for the pluggable, out-of-tree and out-of-core cloud providers, has been significantly improved. ([#88](https://github.com/kubernetes/features/issues/88), [@wlan0](https://github.com/wlan0))

### **Network**
#### network-policy
* [beta] Apply NetworkPolicy based on CIDR ([#50033](https://github.com/kubernetes/kubernetes/pull/50033), [@cmluciano](https://github.com/cmluciano))
#### kube-proxy ipvs mode
* [alpha] Support ipvs mode for kube-proxy([#46580](https://github.com/kubernetes/kubernetes/pull/46580), [@haibinxie](https://github.com/haibinxie))

## External Dependencies
Continuous integration builds have used Docker versions 1.11.2, 1.12.6, 1.13.1,
and 17.03.2. These versions have been validated on Kubernetes 1.8. However,
consult an appropriate installation or upgrade guide before deciding what
versions of Docker to use.

- Docker 1.13.1 and 17.03.2
    - Shared PID namespace, live-restore, and overlay2 have been validated.
    - **Known issues**
        - The default iptables FORWARD policy has been changed from ACCEPT to
          DROP, which causes outbound container traffic to stop working by
          default. See
          [#40182](https://github.com/kubernetes/kubernetes/issues/40182) for
          the workaround.
        - The support for the v1 registries has been removed.
- Docker 1.12.6
    - Overlay2 and live-restore have *not* been validated.
    - **Known issues**
        - Shared PID namespace does not work properly.
          ([#207](https://github.com/kubernetes/community/pull/207#issuecomment-281870043))
        - Docker reports incorrect exit codes for containers.
          ([#41516](https://github.com/kubernetes/kubernetes/issues/41516))
- Docker 1.11.2
    - **Known issues**
        - Kernel crash with Aufs storage driver on Debian Jessie
          ([#27885](https://github.com/kubernetes/kubernetes/issues/27885)),
          which can be identified by the node problem detector.
        - File descriptor leak on init/control.
          ([#275](https://github.com/containerd/containerd/issues/275))
        - Additional memory overhead per container.
          ([#21737](https://github.com/kubernetes/kubernetes/pull/21737))
        - Processes may be leaked when Docker is killed repeatedly in a short
          time frame.
          ([#41450](https://github.com/kubernetes/kubernetes/issues/41450))
