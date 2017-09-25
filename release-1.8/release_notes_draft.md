# Checklist for SIGs and Release Team
As SIGs fill out their sections by component, please check off that
you are finished. For guidance about what should have a release note
please check out the [release notes guidance][] issue.

- [ ] sig-api-machinery
- [x] sig-apps
- [x] sig-architecture
- [x] sig-auth
- [x] sig-autoscaling
- [x] sig-aws
- [x] sig-azure
- [x] sig-big-data
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
- [x] sig-scalability
- [x] sig-scheduling
- [x] sig-service-catalog
- [x] sig-storage
- [x] sig-testing
- [x] sig-ui
- [x] sig-windows

[release notes guidance]: https://github.com/kubernetes/community/issues/484

## **Major Themes**

- The following Kubernetes objects are now part of the new apps/v1beta2 group and version: DaemonSet, Deployment, ReplicaSet, StatefulSet.

- The roles based access control (RBAC) API group for managing API authorization
has been promoted to v1. No changes were made to the API from v1beta1. This
promotion indicates RBAC's production readiness and adoption. Today, the
authorizer is turned on by default by many distributions of Kubernetes, and is a
fundamental aspect of a secure cluster.


### SIG Apps

[SIG Apps][] focuses on the Kubernetes APIs and external tools required to deploy and operate a wide variety of workloads.

For the 1.8 release, SIG Apps moved the kubernetes workloads API to the new apps/v1beta2 group and version. Though apps/v1beta2 group introduces several deprecations and behavioral changes, it provides developers with a stable and consistent API surface to build applications in Kubernetes. SIG Apps intends to promote this version to GA in a future release.


[SIG Apps]: https://github.com/kubernetes/community/tree/master/sig-apps

### SIG Auth

[SIG Auth][] is responsible for Kubernetes authentication, authorization, and
cluster security policies.

For the 1.8 release SIG Auth focused on stablizing existing features introduced
in previous releases. RBAC was promoted to v1 and advanced auditing was promoted
to beta. Encryption of resources at rest, which remained alpha, began exploring
integrations with external Key Management Systems.

[SIG Auth]: https://github.com/kubernetes/community/tree/master/sig-auth

### SIG Cluster Lifecycle

[SIG Cluster Lifecycle][] is responsible for the user experience of deploying,
upgrading, and deleting clusters.

For the 1.8 release SIG Cluster Lifecycle continued to focus on expanding the
capabilities of kubeadm, which is both a user-facing tool to manage clusters
and a building block for higher-level provisioning systems. Starting
with the 1.8 release kubeadm supports a new upgrade command and has alpha
support for self hosting the cluster control plane.

[SIG Cluster Lifecycle]: https://github.com/kubernetes/community/tree/master/sig-cluster-lifecycle


### SIG Node

[SIG Node][] is responsible for the components which support the controlled
interactions between pods and host resources as well as managing the lifecycle
of pods scheduled on a node.

For the 1.8 release SIG Node continued to focus
on supporting the broadest set of workload types, including hardware and performance
sensitive workloads such as data analytics and deep learning, while delivering
incremental improvements to node reliability.

[SIG Node]: https://github.com/kubernetes/community/tree/master/sig-node

### SIG Network

[SIG Network][] is responsible for networking components, APIs, and plugins in Kubernetes.

For the 1.8 release, SIG Network enhanced the NetworkPolicy API to include support for pod egress traffic policies, and a match criteria that allows policy rules to match source or destination CIDR. Both of these enhancements are designated as beta. SIG Network also focused on improving the kube-proxy to include an alpha IPVS mode in addition to the current iptables and userspace modes.

[SIG Network]: https://github.com/kubernetes/community/tree/master/sig-network

### SIG Storage

[SIG Storage][] is responsible for storage and volume plugin components.

For the 1.8 release, SIG Storage extends the Kubernetes storage API, beyond just
making volumes available, to enabling volume resizing and snapshotting. Beyond these
alpha/prototype features, the SIG, focused on providing users more control over their
storage: with features like the ability to set requests & limits on ephemeral storage,
the ability to specify mount options, more metrics, and improvements to Flex driver
deployments.

[SIG Storage]: https://github.com/kubernetes/community/tree/master/sig-storage

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

### SIG Scalability

[SIG Scalability][] is responsible for scalability testing, measuring and
improving system performance, and answering questions related to scalability.

For the 1.8 release, SIG Scalability focused on automating large cluster
scalability testing in a continuous integration (CI) environment. In addition
to defining a concrete process for scalability testing, SIG Scalability created
documentation for the current scalability thresholds and defined a new set of
Service Level Indicators (SLIs) and Service Level Objectives (SLOs) spanning
across the system. Here's the release [scalability validation report].

[SIG Scalability]: https://github.com/kubernetes/community/tree/master/sig-scalability
[scalability validation report]: https://github.com/kubernetes/features/tree/master/release-1.8/scalability_validation_report.md

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
  * The `--audit-policy-file` is now required unless the `AdvancedAudit`
    feature is explicitly turned off on the API server. (`--feature-gates=AdvancedAudit=false`)
  * The webhook and log file now output the `v1beta1` event format.
  * The audit log file defaults to JSON encoding when using the advanced
    auditing feature gate.
  * The `--audit-policy-file` requires `kind` and `apiVersion` fields
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

* The APIs `rbac/v1alpha1`, `settings/v1alpha1`, and `scheduling/v1alpha1` are
  disabled by default.

* The `system:node` role is no longer automatically granted to the `system:nodes`
  group in new clusters. It is recommended that nodes be authorized using the
  `Node` authorization mode instead. Installations that wish to continue giving
  all members of the `system:nodes` group the `system:node` role (which grants
  broad read access, including all secrets and configmaps) must create an
  installation-specific `ClusterRoleBinding`. ([#49638](https://github.com/kubernetes/kubernetes/pull/49638))

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
- The API server flag `--experimental-bootstrap-token-auth` is now deprecated in favor of `--enable-bootstrap-token-auth`. The `--experimental-bootstrap-token-auth` flag will be removed in 1.9.

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
 - The default spec.successfulJobsHistoryLimit is 3 and spec.failedJobsHistoryLimit
   is 1 on CronJobs.

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

* [stable] Mount Options
  * Promote the ability to specify mount options for volumes from beta to stable.
  * Introduce a new `MountOptions` field in the `PersistentVolume` spec to specify mount options (instead of an annotation).
  * Introduce a new `MountOptions` field in the `StorageClass` spec to allow configuration of mount options for dynamically provisioned volumes.
  enable k8s admins to control mount options being used in their clusters
* [stable] Support Attach/Detach for RWO volumes iSCSI and Fibre Channel.
* [stable] Expose Storage Usage Metrics
  * Expose how much available capacity a given PV has through the Kubernetes metrics API.
* [stable] Volume Plugin Metrics
  * Expose success and latency metrics for all the Kubernetes mount/unmount/attach/detach/provision/delete calls through the Kubernetes metrics API.
* [stable] Modify PV spec for Azure File, CephFS, iSCSI, Glusterfs to allow referencing namespaced resources.
* [stable] Support customization of iSCSI initiator name per volume in iSCSI volume plugin.
* [stable] Support WWID for volume identifier in Fibre Channel volume plugin.
* [beta] Reclaim policy in StorageClass
  * Allow configuration of reclaim policy in StorageClass, instead of always defaulting to `delete` for dynamically provisioned volumes.
* [alpha] Volume resizing
  * Enable increasing the size of a volume through the Kubernetes API.
  * For alpha, this feature only increases the size of the underlying volume and does not do filesystem resizing.
  * For alpha, this feature is only implmented for Gluster volumes.
* [alpha] Provide Capacity Isolation/Resource Management for Local Ephemeral Storage
  * Introduce ability to set container requests/limits, and node allocatable reservations for the new `ephemeral-storage` resource.
  * The `ephemeral-storage` resource includes all the disk space space a container may use (via container overlay or scratch).
* [alpha] Mount namespace propagation
  * Introduce new `VolumeMount.Propagation` field for `VolumeMount` in pod containers.
  * This field may be set to `Bidirectional` to enable a particular mount for a container to be propagated from the container to the host or other containers.
* [alpha] Improve Flexvolume Deployment
  * Simplify Flex volume driver deployment
    * Automatically discover and initialize new driver files instead of requiring kubelet/controller-manager restart.
    * Provide a sample DaemonSet that can be used to deploy Flexvolume drivers.
* [prototype] Volume Snapshots
  * Enable triggering the creation of a volume snapshot through the Kubernetes API.
  * The prototype does not support quiescing before snapshot, so snapshots might be inconsistent.
  * For the prototype phase, this feature is external to the core Kubernetes, and can be found at https://github.com/kubernetes-incubator/external-storage/tree/master/snapshot

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

* Websocket requests may now authenticate to the API server by passing a bearer token in a websocket subprotocol of the form `base64url.bearer.authorization.k8s.io.<base64url-encoded-bearer-token>`. ([#47740](https://github.com/kubernetes/kubernetes/pull/47740) [@liggitt](https://github.com/liggitt))
* Advanced audit now correctly reports impersonated user info. ([#48184], [@CaoShuFeng](https://github.com/CaoShuFeng))
* Advanced audit policy now supports matching subresources and resource names, but the top level resource no longer matches the subresouce. For example "pods" no longer matches requests to the logs subresource of pods. Use "pods/logs" to match subresources. ([#48836](https://github.com/kubernetes/kubernetes/pull/48836), [@ericchiang](https://github.com/ericchiang))
* Previously a deleted service account or bootstrapping token secret would be considered valid until it was reaped. It is now invalid as soon as the `deletionTimestamp` is set. ([#48343](https://github.com/kubernetes/kubernetes/pull/48343), [@deads2k](https://github.com/deads2k); [#49057](https://github.com/kubernetes/kubernetes/pull/49057), [@ericchiang](https://github.com/ericchiang))
* The `--insecure-allow-any-token` flag has been removed from the API server. Users of the flag should use impersonation headers instead for debugging. ([#49045](https://github.com/kubernetes/kubernetes/pull/49045), [@ericchiang](https://github.com/ericchiang))
* The NodeRestriction admission plugin now allows a node to evict pods bound to itself. ([#48707](https://github.com/kubernetes/kubernetes/pull/48707), [@danielfm](https://github.com/danielfm))
* The OwnerReferencesPermissionEnforcement admission plugin now requires `update` permission on the `finalizers` subresource of the referenced owner in order to set `blockOwnerDeletion` on an owner reference. ([#49133](https://github.com/kubernetes/kubernetes/pull/49133), [@deads2k](https://github.com/deads2k))
* The SubjectAccessReview API in the `authorization.k8s.io` API group now allows providing the user uid. ([#49677](https://github.com/kubernetes/kubernetes/pull/49677), [@dims](https://github.com/dims))
* After a kubelet rotates its client cert, it now closes its connections to the API server to force a handshake using the new cert. Previously, the kubelet could keep its existing connection open, even if the cert used for that connection was expired and rejected by the API server. ([#49899](https://github.com/kubernetes/kubernetes/pull/49899), [@ericchiang](https://github.com/ericchiang))
* PodSecurityPolicies can now specify a whitelist of allowed paths for host volumes. ([#50212](https://github.com/kubernetes/kubernetes/pull/50212), [@jhorwit2](https://github.com/jhorwit2))
* API server authentication now caches successful bearer token authentication results for a few seconds. ([#50258](https://github.com/kubernetes/kubernetes/pull/50258), [@liggitt](https://github.com/liggitt))
* The OpenID Connect authenticator can now use a custom prefix, or omit the default prefix, for username and groups claims through the --oidc-username-prefix and --oidc-groups-prefix flags. For example, the authenticator can map a user with the username "jane" to "google:jane" by supplying the "google:" username prefix. ([#50875](https://github.com/kubernetes/kubernetes/pull/50875), [@ericchiang](https://github.com/ericchiang))
* The bootstrap token authenticator can now configure tokens with a set of extra groups in addition to `system:bootstrappers`. ([#50933](https://github.com/kubernetes/kubernetes/pull/50933), [@mattmoyer](https://github.com/mattmoyer))
* Advanced audit allows logging failed login attempts. ([#51119](https://github.com/kubernetes/kubernetes/pull/51119), [@soltysh](https://github.com/soltysh))
* A `kubectl auth reconcile` subcommand has been added for applying RBAC resources. When passed a file which contains RBAC roles, rolebindings, clusterroles, or clusterrolebindings, it will compute covers and add the missing rules. ([#51636](https://github.com/kubernetes/kubernetes/pull/51636), [@deads2k](https://github.com/deads2k))

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
