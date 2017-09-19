# Checklist for SIGs and Release Team
As SIGs fill out their sections by component, please check off that
you are finished. For guidance about what should have a release note
please check out the [release notes guidance][] issue.

- [ ] sig-api-machinery
- [ ] sig-apps
- [ ] sig-architecture
- [x] sig-auth
- [x] sig-autoscaling
- [ ] sig-aws
- [ ] sig-azure
- [ ] sig-big-data
- [x] sig-cli
- [x] sig-cluster-lifecycle
- [x] sig-cluster-ops
- [x] sig-contributor-experience
- [x] sig-docs
- [ ] sig-federation
- [x] sig-governance.md
- [x] sig-instrumentation
- [x] sig-network
- [ ] sig-node
- [x] sig-on-premise
- [x] sig-openstack
- [x] sig-product-management
- [x] sig-release
- [ ] sig-scalability
- [x] sig-scheduling
- [x] sig-service-catalog
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

### SIG Node

[SIG Node][] is responsible for the components which support the controlled 
interactions between pods and host resources as well as managing the lifecycle
of pods scheduled on a node. For the 1.8 release SIG Node continued to focus
on supporting the broadest set of workload types, including hardware and performance
sensitive workloads such as data analytics and deep learning, while delivering
incremental improvements to node reliability.



[SIG Node]: https://github.com/kubernetes/community/tree/master/sig-node

### SIG Autoscaling

[SIG Autoscaling][] is responsible for autoscaling-related components,
such as the Horizontal Pod Autoscaler and Cluster Autoscaler.

For the 1.8 release, SIG Autoscaling continued focused on stabilizing
features introduced in previous releases, such as the new version of the
Horizontal Pod Autoscaler API (with support for custom metrics), as well
as the Cluster Autoscaler (with improved performance and error reporting).


[SIG Autoscaling]: https://github.com/kubernetes/community/tree/master/sig-autoscaling

### SIG Instrumentation

[SIG Instrumentation][] is responsible for metrics production and
collection.

For the 1.8 release, SIG Instrumentation focused on stabilizing the APIs
and components required to support the new version of the Horizontal Pod
Autoscaler API: the resource metrics API, custom metrics API, and
metrics-server, the new replacement for Heapster in the default monitoring
pipeline.


[SIG Instrumentation]: https://github.com/kubernetes/community/tree/master/sig-instrumentation

## **Action Required Before Upgrading**

* The autoscaling/v2alpha1 API has graduated to autoscaling/v2beta1.  The
  form remains unchanged.  HorizontalPodAutoscalers making use of features
  from the autoscaling/v2alpha1 API will need to be migrated to
  autoscaling/v2beta1 to ensure that the new features are properly
  persisted.

* The metrics APIs, `custom-metrics.metrics.k8s.io` and `metrics`, have
  graduated from `v1alpha1` to `v1beta1`, and been renamed to
  `custom.metrics.k8s.io` and `metrics.k8s.io`, respectively. If you have
  deployed a custom metrics adapter, ensure that it supports the new API
  version. If you have deployed Heapster in aggregated API server mode,
  ensure that you upgrade Heapster as well.

* Advanced auditing has graduated from `v1alpha1` to `v1beta1` with the
  following changes to the default behavior.
  * Advanced auditing is enabled by default.
  * The webhook and log file now output the `v1beta1` event format.
  * The audit log file defaults to JSON encoding when using the advanced
    auditing feature gate.
  * The`--audit-policy-file` requires `kind` and `apiVersion` fields
    specifying what format version the `Policy` is using.

* The deprecated ThirdPartyResource (TPR) API has been removed.
  To avoid losing your TPR data, you must
  [migrate to CustomResourceDefinition](https://kubernetes.io/docs/tasks/access-kubernetes-api/migrate-third-party-resource/)
  **before upgrading to 1.8**.

* The following deprecated flags have been removed from `kube-controller-manager`:

  * `replication-controller-lookup-cache-size`
  * `replicaset-lookup-cache-size`
  * `daemonset-lookup-cache-size`

  Do not use deprecated flags.

* StatefulSet: The deprecated `pod.alpha.kubernetes.io/initialized` annotation for
  interrupting StatefulSet Pod management is now ignored. If the annotation is set
  to `true` or left blank, you won't see any change in behavior. If it's set to
  `false`, previously dormant StatefulSets might become active after upgrading.

* CronJobs has been promoted to `v1beta1` which is now turned on by default.
  Although version `v2alpah1` is still available, it is deprecated.  Migrate to
  `batch/v1beta1.CronJobs`.  Additionally, upgrading cluster in high-availability
  configuration might return errors. The new controllers rely on the latest
  version of the resources.  If the expected version is not found during rolling
  upgrade, the system throws resource not found errors.
* The `batch/v2alpha1.ScheduledJobs` has been removed.  Migrate to `batch/v1beta.CronJobs`
  to continue managing time based jobs.

## **Known Issues**

## **Deprecations**

### Apps

- The `.spec.rollbackTo` field of the Deployment kind is deprecated in the
  extensions/v1beta1 group version.

- The `kubernetes.io/created-by` annotation is now deprecated and will be removed in v1.9.
  Use [ControllerRef](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/controller-ref.md)
  instead to determine which controller, if any, owns an object.

 - The `batch/v2alpha1.CronJob` has been deprecated in favor of `batch/v1beta1`.

 - The `batch/v2alpha1.ScheduledJobs` has been removed, use `batch/v1beta1.CronJobs` instead.


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

### Autoscaling

- Consuming metrics directly from Heapster is now deprecated in favor of
  consuming metrics via an aggregated version of the resource metrics API.
  - In v1.8, this behavior can be enabled by setting the
    `--horizontal-pod-autoscaler-use-rest-clients` flag to `true`.
  - In v1.9, this behavior will be on by default, and must by explicitly
    disabled by setting the above flag to `false`.

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

### [Workload API (batch)]
- CronJob has been promoted to `batch/v1beta1` ([#41039](https://github.com/kubernetes/kubernetes/issues/41039), [@soltysh](https://github.com/soltysh)).
- `batch/v2alpha.CronJob` has been deprecated in favor of `batch/v1beta` and will be removed in future releases.
- Job has now the ability to set a failure policy using `.spec.backoffLimit`.  The default value for this new field is set to 6. ([#30243](https://github.com/kubernetes/kubernetes/issues/30243), [@clamoriniere1A](https://github.com/clamoriniere1A)).
- `batch/v2alpha1.ScheduledJobs` has been removed.
- Job controller creates pods in batches instead of all at once ([#49142](https://github.com/kubernetes/kubernetes/pull/49142), [@joelsmith](https://github.com/joelsmith)).
- Short `.spec.ActiveDeadlineSeconds` is properly applied to a job ([#48545](https://github.com/kubernetes/kubernetes/pull/48454), [@weiwei4](https://github.com/weiwei04)).


#### CLI Changes

- [alpha] `kubectl` plugins: `kubectl` now allows binary extensibility as an alpha
  feature. Users can extend the default set of `kubectl` commands by writing plugins
  that provide new subcommands. Please refer to the documentation for more information.
- `kubectl rollout` and `rollback` now support StatefulSet.
- `kubectl scale` now uses the Scale subresource for kinds in the apps/v1beta2 group.
- `kubectl create configmap` and `kubectl create secret` subcommands now support
  the `--append-hash` flag, which enables unique yet deterministic naming for
  objects generated from files, e.g. via `--from-file`.
- `kubectl run` learned how to set a service account name in the generated pod
  spec with the `--serviceaccount` flag.
- `kubectl proxy` will now correctly handle the `exec`, `attach`, and
  `portforward` commands.  You must pass `--disable-filter` to the command in
  order to allow these endpoints.
- Added `cronjobs.batch` to "all", so `kubectl get all` returns them.
- Add flag `--include-uninitialized` to kubectl annotate, apply, edit-last-applied,
  delete, describe, edit, get, label, set. "--include-uninitialized=true" makes
  kubectl commands apply to uninitialized objects, which by default are ignored
  if the names of the objects are not provided. "--all" also makes kubectl
  commands apply to uninitialized objects. Please see the
  [initializer](https://kubernetes.io/docs/admin/extensible-admission-controllers/)
  doc for more details.
- Add RBAC reconcile commands through `kubectl auth reconcile -f FILE`. When
  passed a file which contains RBAC roles, rolebindings, clusterroles, or
  clusterrolebindings, it will compute covers and add the missing rules.
  The logic required to properly "apply" RBAC permissions is more complicated
  that a json merge since you have to compute logical covers operations between
  rule sets. This means that we cannot use kubectl apply to update RBAC roles
  without risking breaking old clients (like controllers).
- `kubectl delete` no longer scales down workload API objects prior to deletion.
  Users who depend on ordered termination for the Pods of their StatefulSet’s
  must use kubectl scale to scale down the StatefulSet prior to deletion.
- `kubectl run --env` no longer supports CSV parsing. To provide multiple env
  vars, use the `--env` flag multiple times instead of having env vars separated
  by commas. E.g. `--env ONE=1 --env TWO=2` instead of `--env ONE=1,TWO=2`.
- Remove deprecated command `kubectl stop`.
- Allows kubectl to use http caching mechanism for the OpenAPI schema. The cache
  directory can be configured through `--cache-dir` command line flag to kubectl.
  If set to empty string, caching will be disabled.
- Kubectl performs validation against OpenAPI schema rather than Swagger 1.2. If
  OpenAPI is not available on the server, it falls back to the old Swagger 1.2.
- Add Italian translation for kubectl.
- Add German translation for kubectl.

#### Scheduling
* [alpha] Support pod priority and creation of PriorityClasses ([user doc](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/))([design doc](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/pod-priority-api.md))
* [alpha] Support priority-based preemption of pods ([user doc](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/))([design doc](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/pod-preemption.md))
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

* [stable] CRI-O support, it has passed all e2es. [@mrunalp]

#### Autoscaling and Metrics

* Support for custom metrics in the Horizontal Pod Autoscaler is moving to
  beta.  The associated metrics APIs (custom metrics and resource/master
  metrics) are graduating to v1beta1.  See [Action Required Before
  Upgrading](#action-required-before-upgrading).

* metrics-server is now the reccomended way to provide the resource
  metrics API. It is deployable as an addon, similarly to how Heapster is
  deployed.

##### Cluster Autoscaler

* Cluster autoscaler is now GA
* Incresed cluster support size to 1000 nodes
* Respect graceful pod termination of up to 10 minutes
* Handle zone stock-outs and failures
* Improved monitoring and error reporting

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
* [beta] Support EgressRules in NetworkPolicy ([#51351](https://github.com/kubernetes/kubernetes/pull/51351), [@cmluciano](https://github.com/cmluciano))
#### kube-proxy ipvs mode
* [alpha] Support ipvs mode for kube-proxy([#46580](https://github.com/kubernetes/kubernetes/pull/46580), [@haibinxie](https://github.com/haibinxie))

### API Machinery

* [alpha] The CustomResourceDefinition API can now optionally
  [validate custom objects](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/#validation)
  based on a JSON schema provided in the CRD spec.
  Enable this alpha feature with the `CustomResourceValidation` feature gate in `kube-apiserver`.

* The garbage collector now supports custom APIs added via CustomResourceDefinition
  or aggregated API servers. The garbage collector controller refreshes periodically.
  Therefore, expect a latency of about 30 seconds between when the API is added and when
  the garbage collector starts to manage it.

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
