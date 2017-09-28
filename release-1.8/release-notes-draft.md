## Introduction to v1.8.0

Kubernetes version 1.8 includes new features and enhancements, as well as fixes to identified issues. The release notes contain a brief overview of the important changes introduced in this release. The content is organized by Special Interest Groups ([SIGs][]).

For initial installations, see the [Setup topics][] in the Kubernetes
documentation.

To upgrade to this release from a previous version, take any actions required
[Before Upgrading](#before-upgrading).

For more information about the release and for the latest documentation,
see the [Kubernetes documentation](https://kubernetes.io/docs/home/).

[Setup topics]: https://kubernetes.io/docs/setup/pick-right-solution/
[SIGs]: https://github.com/kubernetes/community/blob/master/sig-list.md


## Major Themes

Kubernetes is developed by community members whose work is organized into
[Special Interest Groups][]. For the 1.8 release, each SIG provides the
themes that guided their work.

[Special Interest Groups]: https://github.com/kubernetes/community/blob/master/sig-list.md

### SIG API Machinery

[SIG API Machinery][] is responsible for all aspects of the API server: API registration and discovery, generic API CRUD semantics, admission control, encoding/decoding, conversion, defaulting, persistence layer (etcd), OpenAPI, third-party resources, garbage collection, and client libraries.

For the 1.8 release, SIG API Machinery focused on stability and on ecosystem enablement. Features include the ability to break large LIST calls into smaller chunks, improved support for API server customization with either custom API servers or Custom Resource Definitions, and client side event spam filtering.

[Sig API Machinery]: https://github.com/kubernetes/community/tree/master/sig-api-machinery

### SIG Apps

[SIG Apps][] focuses on the Kubernetes APIs and the external tools that are required to deploy and operate Kubernetes workloads.

For the 1.8 release, SIG Apps moved the Kubernetes workloads API to the new apps/v1beta2 group and version. The DaemonSet, Deployment, ReplicaSet, and StatefulSet objects are affected by this change. The new apps/v1beta2 group and version provide a stable and consistent API surface for building applications in Kubernetes. For details about deprecations and behavioral changes, see [Notable Features](#notable-features). SIG Apps intends to promote this version to GA in a future release.

[SIG Apps]: https://github.com/kubernetes/community/tree/master/sig-apps

### SIG Auth

[SIG Auth][] is responsible for Kubernetes authentication and authorization, and for
cluster security policies.

For the 1.8 release, SIG Auth focused on stablizing existing features that were introduced
in previous releases. RBAC was moved from beta to v1, and advanced auditing was moved from alpha
to beta. Encryption of resources stored on disk (resources at rest) remained in alpha, and the SIG began exploring integrations with external key management systems.

[SIG Auth]: https://github.com/kubernetes/community/tree/master/sig-auth

### SIG Autoscaling

[SIG Autoscaling][] is responsible for autoscaling-related components,
such as the Horizontal Pod Autoscaler and Cluster Autoscaler.

For the 1.8 release, SIG Autoscaling continued to focus on stabilizing
features introduced in previous releases: the new version of the
Horizontal Pod Autoscaler API, which supports custom metrics, and
the Cluster Autoscaler, which provides improved performance and error reporting.

[SIG Autoscaling]: https://github.com/kubernetes/community/tree/master/sig-autoscaling

### SIG Cluster Lifecycle

[SIG Cluster Lifecycle][] is responsible for the user experience of deploying,
upgrading, and deleting clusters.

For the 1.8 release, SIG Cluster Lifecycle continued to focus on expanding the
capabilities of kubeadm, which is both a user-facing tool to manage clusters
and a building block for higher-level provisioning systems. Starting
with the 1.8 release, kubeadm supports a new upgrade command and includes alpha
support for self hosting the cluster control plane.

[SIG Cluster Lifecycle]: https://github.com/kubernetes/community/tree/master/sig-cluster-lifecycle

### SIG Instrumentation

[SIG Instrumentation][] is responsible for metrics production and
collection.

For the 1.8 release, SIG Instrumentation focused on stabilizing the APIs
and components that are required to support the new version of the Horizontal Pod
Autoscaler API: the resource metrics API, custom metrics API, and
metrics-server, which is the new replacement for Heapster in the default monitoring
pipeline.

[SIG Instrumentation]: https://github.com/kubernetes/community/tree/master/sig-instrumentation

### SIG Multi-cluster (formerly known as SIG Federation)

[SIG Multi-cluster][] is responsible for infrastructure that supports
the efficient and reliable management of multiple Kubernetes clusters,
and applications that run in and across multiple clusters.

For the 1.8 release, SIG Multicluster focussed on expanding the set of
Kubernetes primitives that our Cluster Federation control plane
supports, expanding the number of approaches taken to multi-cluster
management (beyond our initial Federation approach), and preparing
to release Federation for general availability ('GA').

[SIG Multi-cluster]: https://github.com/kubernetes/community/tree/master/sig-federation

### SIG Node

[SIG Node][] is responsible for the components that support the controlled
interactions between pods and host resources, and manage the lifecycle
of pods scheduled on a node.

For the 1.8 release, SIG Node continued to focus
on a broad set of workload types, including hardware and performance
sensitive workloads such as data analytics and deep learning. The SIG also
delivered incremental improvements to node reliability.

[SIG Node]: https://github.com/kubernetes/community/tree/master/sig-node

### SIG Network

[SIG Network][] is responsible for networking components, APIs, and plugins in Kubernetes.

For the 1.8 release, SIG Network enhanced the NetworkPolicy API to support pod egress traffic policies.
The SIG also provided match criteria that allow policy rules to match a source or destination CIDR. Both features are in beta. SIG Network also improved the kube-proxy to include an alpha IPVS mode in addition to the current iptables and userspace modes.

[SIG Network]: https://github.com/kubernetes/community/tree/master/sig-network

### SIG Scalability

[SIG Scalability][] is responsible for scalability testing, measuring and
improving system performance, and answering questions related to scalability.

For the 1.8 release, SIG Scalability focused on automating large cluster
scalability testing in a continuous integration (CI) environment. The SIG
defined a concrete process for scalability testing, created
documentation for the current scalability thresholds, and defined a new set of
Service Level Indicators (SLIs) and Service Level Objectives (SLOs) for the system.
Here's the release [scalability validation report].

[SIG Scalability]: https://github.com/kubernetes/community/tree/master/sig-scalability
[scalability validation report]: https://github.com/kubernetes/features/tree/master/release-1.8/scalability_validation_report.md

### SIG Scheduling

[SIG Scheduling][] is responsible for generic scheduler and scheduling components.

For the 1.8 release, SIG Scheduling extended the concept of cluster sharing by introducing
pod priority and pod preemption. These features allow mixing various types of workloads in a single cluster, and help reach
higher levels of resource utilization and availability.
These features are in alpha. SIG Scheduling also improved the internal APIs for scheduling and made them easier for other components and external schedulers to use.

[SIG Scheduling]: https://github.com/kubernetes/community/tree/master/sig-scheduling

### SIG Storage

[SIG Storage][] is responsible for storage and volume plugin components.

For the 1.8 release, SIG Storage extended the Kubernetes storage API. In addition to providing simple
volume availability, the API now enables volume resizing and snapshotting. These features are in alpha.
The SIG also focused on providing more control over storage: the ability to set requests and
limits on ephemeral storage, the ability to specify mount options, more metrics, and improvements to Flex driver deployments.

[SIG Storage]: https://github.com/kubernetes/community/tree/master/sig-storage

## Before Upgrading

Consider the following changes, limitations, and guidelines before you upgrade:

* The kubelet now fails if swap is enabled on a node. To override the default and run with /proc/swaps on, set `--fail-swap-on=false`. The experimental flag `--experimental-fail-swap-on` is deprecated in this release, and will be removed in a future release.

* The `autoscaling/v2alpha1` API is now at `autoscaling/v2beta1`. However, the form of the API remains unchanged. Migrate the `HorizontalPodAutoscaler` resources to `autoscaling/v2beta1` to persist the `HorizontalPodAutoscaler` changes introduced in `autoscaling/v2alpha1`. The Horizontal Pod Autoscaler changes include support for status conditions, and autoscaling on memory and custom metrics.

* The metrics APIs, `custom-metrics.metrics.k8s.io` and `metrics`, were moved from `v1alpha1` to `v1beta1`, and renamed to `custom.metrics.k8s.io` and `metrics.k8s.io`, respectively. If you have deployed a custom metrics adapter, ensure that it supports the new API version. If you have deployed Heapster in aggregated API server mode, upgrade Heapster to support the latest API version.

* Advanced auditing is the default auditing mechanism at `v1beta1`. The new version introduces the following changes:

  * The `--audit-policy-file` option is required if the `AdvancedAudit` feature is not explicitly turned off (`--feature-gates=AdvancedAudit=false`) on the API server.
  * The audit log file defaults to JSON encoding when using the advanced auditing feature gate.
  * The `--audit-policy-file` option requires `kind` and `apiVersion` fields specifying what format version the `Policy` is using.
  * The webhook and log file now output the `v1beta1` event format.

    For more details, see [Advanced audit](https://kubernetes.io/docs/tasks/debug-application-cluster/audit/#advanced-audit).

* The deprecated `ThirdPartyResource` (TPR) API was removed.
  To avoid losing your TPR data, [migrate to CustomResourceDefinition](https://kubernetes.io/docs/tasks/access-kubernetes-api/migrate-third-party-resource/).

* The following deprecated flags were removed from `kube-controller-manager`:

  * `replication-controller-lookup-cache-size`
  * `replicaset-lookup-cache-size`
  * `daemonset-lookup-cache-size`

  Don't use these flags. Using deprecated flags causes the server to print a warning. Using a removed flag causes the server to abort the startup.

* StatefulSet: The deprecated `pod.alpha.kubernetes.io/initialized` annotation for interrupting the StatefulSet Pod management is now ignored. StatefulSets with this annotation set to `true` or with no value will behave just as they did in previous versions. Dormant StatefulSets with the annotation set to `false` will become active after upgrading.

* The CronJob object is now enabled by default at `v1beta1`. CronJob `v2alpha1` is still available, but it must be explicitly enabled. We recommend that you move any current CronJob objects to `batch/v1beta1.CronJob`. Be aware that if you specify the deprecated version, you may encounter Resource Not Found errors. These errors occur because the new controllers look for the new version during a rolling update.

* The `batch/v2alpha1.ScheduledJob` was removed. Migrate to `batch/v1beta.CronJob` to continue managing time based jobs.

* The `rbac/v1alpha1`, `settings/v1alpha1`, and `scheduling/v1alpha1` APIs are disabled by default.

* The `system:node` role is no longer automatically granted to the `system:nodes` group in new clusters. The role gives broad read access to resources, including secrets and configmaps. Use the `Node` authorization mode to authorize the nodes in new clusters. To continue providing the `system:node` role to the members of the `system:nodes` group, create an installation-specific `ClusterRoleBinding` in the installation. ([#49638](https://github.com/kubernetes/kubernetes/pull/49638))

## Known Issues

This section contains a list of known issues reported in Kubernetes 1.8 release. The content is populated via [v1.8.x known issues and FAQ accumulator](https://github.com/kubernetes/kubernetes/issues/53004).

* A performance issue was identified in large-scale clusters when deleting thousands of pods simultaneously across hundreds of nodes. Kubelets in this scenario can encounter temporarily increased latency of `delete pod` API calls -- above the target service level objective of 1 second. If you run clusters with this usage pattern and if pod deletion latency could be an issue for you, you might want to wait until the issue is resolved before you upgrade. 

For more information and for updates on resolution of this issue, see [#51899](https://issue.k8s.io/51899).

* Audit logs might impact the API server performance and the latency of large request and response calls. The issue is observed under the following conditions: if `AdvancedAuditing` feature gate is enabled, which is the default case, if audit logging uses the log backend in JSON format, or if the audit policy records large API calls for requests or responses.

For more information, see [#51899](https://github.com/kubernetes/kubernetes/issues/51899).

* Minikube version 0.22.2 or lower does not work with kubectl version 1.8 or higher. This issue is caused by the presence of an unregistered type in the minikube API server. New versions of kubectl force validate the OpenAPI schema, which is not registered with all known types in the minikube API server.

For more information, see [#1996](https://github.com/kubernetes/minikube/issues/1996).

* The `ENABLE_APISERVER_BASIC_AUDIT` configuration parameter for GCE deployments is broken, but deprecated.

For more information, see [#53154](https://github.com/kubernetes/kubernetes/issues/53154).

* `kubectl set` commands placed on ReplicaSet and DaemonSet occasionally return version errors. All set commands, including set image, set env, set resources, and set serviceaccounts, are affected.

For more information, see [#53040](https://github.com/kubernetes/kubernetes/issues/53040).

* Object quotas are not consistently charged or updated. Specifically, the object count quota does not reliably account for uninitialized objects. Some quotas are charged only when an object is initialized. Others are charged when an object is created, whether it is initialized or not. We plan to fix this issue in a future release.

For more information, see [#53109](https://github.com/kubernetes/kubernetes/issues/53109).

## Deprecations

This section provides an overview of deprecated API versions, options, flags, and arguments. Deprecated means that we intend to remove the capability from a future release. After removal, the capability will no longer work. The sections are organized by SIGs.

### Apps

- The `.spec.rollbackTo` field of the Deployment kind is deprecated in `extensions/v1beta1`.

- The `kubernetes.io/created-by` annotation is deprecated and will be removed in version 1.9.
  Use [ControllerRef](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/api-machinery/controller-ref.md) instead to determine which controller, if any, owns an object.

 - The `batch/v2alpha1.CronJob` is deprecated in favor of `batch/v1beta1`.

 - The `batch/v2alpha1.ScheduledJob` was removed. Use `batch/v1beta1.CronJob` instead.

### Auth

 - The RBAC v1alpha1 API group is deprecated in favor of RBAC v1.

 - The API server flag `--experimental-bootstrap-token-auth` is deprecated in favor of `--enable-bootstrap-token-auth`. The `--experimental-bootstrap-token-auth` flag will be removed in version 1.9.

### Autoscaling

 - Consuming metrics directly from Heapster is deprecated in favor of
   consuming metrics via an aggregated version of the resource metrics API.

   - In version 1.8, enable this behavior by setting the
     `--horizontal-pod-autoscaler-use-rest-clients` flag to `true`.

   - In version 1.9, this behavior will be enabled by default, and must be explicitly
     disabled by setting the `--horizontal-pod-autoscaler-use-rest-clients` flag to `false`.

### Cluster Lifecycle

- The `auto-detect` behavior of the kubelet's `--cloud-provider` flag is deprecated.

  - In version 1.8, the default value for the kubelet's `--cloud-provider` flag is `auto-detect`. Be aware that it works only on GCE, AWS and Azure.

  - In version 1.9, the default will be `""`, which means no built-in cloud provider extension will be enabled by default.

  - Enable an out-of-tree cloud provider with `--cloud-provider=external` in either version.

    For more information on deprecating auto-detecting cloud providers in kubelet, see [PR #51312](https://github.com/kubernetes/kubernetes/pull/51312) and [announcement](https://groups.google.com/forum/#!topic/kubernetes-dev/UAxwa2inbTA).

- The `PersistentVolumeLabel` admission controller in the API server is deprecated.

  - The replacement is running a cloud-specific controller-manager (often referred to as `cloud-controller-manager`) with the `PersistentVolumeLabel` controller enabled. This new controller loop operates as the `PersistentVolumeLabel` admission controller did in previous versions.

  - Do not use the `PersistentVolumeLabel` admission controller in the configuration files and scripts unless you are dependent on the in-tree GCE and AWS cloud providers.

  - The `PersistentVolumeLabel` admission controller will be removed in a future release, when the out-of-tree versions of the GCE and AWS cloud providers move to GA. The cloud providers are marked alpha in version 1.9.

### OpenStack

- The `openstack-heat` provider for `kube-up` is deprecated and will be removed
  in a future release. Refer to [Issue #49213](https://github.com/kubernetes/kubernetes/issues/49213)
  for background information.

### Scheduling

  - Opaque Integer Resources (OIRs) are deprecated and will be removed in
    version 1.9. Extended Resources (ERs) are a drop-in replacement for OIRs. You can use
    any domain name prefix outside of the `kubernetes.io/` domain instead of the
    `pod.alpha.kubernetes.io/opaque-int-resource-` prefix.

## Notable Features

### Workloads API (apps/v1beta2)

Kubernetes 1.8 adds the apps/v1beta2 group and version, which now consists of the
DaemonSet, Deployment, ReplicaSet and StatefulSet kinds. This group and version are part
of the Kubernetes Workloads API. We plan to move them to v1 in an upcoming release, so you might want to plan your migration accordingly.

For more information, see [the issue that describes this work in detail](https://github.com/kubernetes/features/issues/353)

#### API Object Additions and Migrations

- The DaemonSet, Deployment, ReplicaSet, and StatefulSet kinds
  are now in the apps/v1beta2 group and version.

- The apps/v1beta2 group version adds a Scale subresource for the StatefulSet
kind.

- All kinds in the apps/v1beta2 group version add a corresponding conditions
  kind.

#### Behavioral Changes

 - For all kinds in the API group version, a spec.selector default value is no longer
 available, because it's incompatible with `kubectl
 apply` and strategic merge patch. You must explicitly set the spec.selector value
 in your manifest. An object with a spec.selector value that does not match the labels in
 its spec.template is invalid.

 - Selector mutation is disabled for all kinds in the
 app/v1beta2 group version, because the controllers in the workloads API do not handle
 selector mutation in
 a consistent way. This restriction may be lifted in the future, but
 it is likely that that selectors will remain immutable after the move to v1.
 You can continue to use code that depends on mutable selectors by calling
 the apps/v1beta1 API in this release, but you should start planning for code
 that does not depend on mutable selectors.

 - Extended Resources are fully-qualified resource names outside the
 `kubernetes.io` domain. Extended Resource quantities must be integers.
 You can specify any resource name of the form `[aaa.]my-domain.bbb/ccc`
 in place of [Opaque Integer Resources](https://v1-6.docs.kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/#opaque-integer-resources-alpha-feature).
 Extended resources cannot be overcommitted, so make sure that request and limit are equal
 if both are present in a container spec.

 - The default Bootstrap Token created with `kubeadm init` v1.8 expires
 and is deleted after 24 hours by default to limit the exposure of the
 valuable credential. You can create a new Bootstrap Token with
 `kubeadm token create` or make the default token permanently valid by specifying
 `--token-ttl 0` to `kubeadm init`. The default token can later be deleted with
 `kubeadm token delete`.

 - `kubeadm join` now delegates TLS Bootstrapping to the kubelet itself, instead
 of reimplementing the process. `kubeadm join` writes the bootstrap kubeconfig
 file to `/etc/kubernetes/bootstrap-kubelet.conf`.

#### Defaults

 - The default spec.updateStrategy for the StatefulSet and DaemonSet kinds is
 RollingUpdate for the apps/v1beta2 group version. You can explicitly set
 the OnDelete strategy, and no strategy auto-conversion is applied to
 replace default values.

 - As mentioned in [Behavioral Changes](#behavioral-changes), selector
 defaults are disabled.

 - The default spec.revisionHistoryLimit for all applicable kinds in the
 apps/v1beta2 group version is 10.

 - In a CronJob object, the default spec.successfulJobsHistoryLimit is 3, and
 the default spec.failedJobsHistoryLimit is 1.

### Workloads API (batch)

- CronJob is now at `batch/v1beta1` ([#41039](https://github.com/kubernetes/kubernetes/issues/41039), [@soltysh](https://github.com/soltysh)).

- `batch/v2alpha.CronJob` is deprecated in favor of `batch/v1beta` and will be removed in a future release.

- Job can now set a failure policy using `.spec.backoffLimit`. The default value for this new field is 6. ([#30243](https://github.com/kubernetes/kubernetes/issues/30243), [@clamoriniere1A](https://github.com/clamoriniere1A)).

- `batch/v2alpha1.ScheduledJob` is removed.

- The Job controller now creates pods in batches instead of all at once. ([#49142](https://github.com/kubernetes/kubernetes/pull/49142), [@joelsmith](https://github.com/joelsmith)).

- Short `.spec.ActiveDeadlineSeconds` is properly applied to a Job. ([#48545]
(https://github.com/kubernetes/kubernetes/pull/48454), [@weiwei4](https://github.com/weiwei04)).


#### CLI Changes

- [alpha] `kubectl` plugins: `kubectl` now allows binary extensibility. You can extend the default set of `kubectl` commands by writing plugins
  that provide new subcommands. Refer to the documentation for more information.

- `kubectl rollout` and `rollback` now support StatefulSet.

- `kubectl scale` now uses the Scale subresource for kinds in the apps/v1beta2 group.

- `kubectl create configmap` and `kubectl create secret` subcommands now support
  the `--append-hash` flag, which enables unique but deterministic naming for
  objects generated from files, for example with `--from-file`.

- `kubectl run` can set a service account name in the generated pod
  spec with the `--serviceaccount` flag.

- `kubectl proxy` now correctly handles the `exec`, `attach`, and
  `portforward` commands.  You must pass `--disable-filter` to the command to allow these commands.

- Added `cronjobs.batch` to "all", so that `kubectl get all` returns them.

- Added flag `--include-uninitialized` to `kubectl annotate`, `apply`, `edit-last-applied`,
  `delete`, `describe`, `edit`, `get`, `label,` and `set`. `--include-uninitialized=true` makes
  kubectl commands apply to uninitialized objects, which by default are ignored
  if the names of the objects are not provided. `--all` also makes kubectl
  commands apply to uninitialized objects. See the
  [initializer documentation](https://kubernetes.io/docs/admin/extensible-admission-controllers/) for more details.

- Added RBAC reconcile commands with `kubectl auth reconcile -f FILE`. When
  passed a file which contains RBAC roles, rolebindings, clusterroles, or
  clusterrolebindings, this command computes covers and adds the missing rules.
  The logic required to properly apply RBAC permissions is more complicated
  than a JSON merge because you have to compute logical covers operations between
  rule sets. This means that we cannot use `kubectl apply` to update RBAC roles
  without risking breaking old clients, such as controllers.

- `kubectl delete` no longer scales down workload API objects before deletion.
  Users who depend on ordered termination for the Pods of their StatefulSets
  must use `kubectl scale` to scale down the StatefulSet before deletion.

- `kubectl run --env` no longer supports CSV parsing. To provide multiple environment
  variables, use the `--env` flag multiple times instead. Example: `--env ONE=1 --env TWO=2` instead of `--env ONE=1,TWO=2`.

- Removed deprecated command `kubectl stop`.

- Kubectl can now use http caching for the OpenAPI schema. The cache
  directory can be configured by passing the `--cache-dir` command line flag to kubectl.
  If set to an empty string, caching is disabled.

- Kubectl now performs validation against OpenAPI schema instead of Swagger 1.2. If
  OpenAPI is not available on the server, it falls back to the old Swagger 1.2.

- Added Italian translation for kubectl.

- Added German translation for kubectl.

#### Scheduling

* [alpha] This version now supports pod priority and creation of PriorityClasses ([user doc](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/))([design doc](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/pod-priority-api.md))

* [alpha] This version now supports priority-based preemption of pods ([user doc](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/))([design doc](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/pod-preemption.md))

* [alpha] Users can now add taints to nodes by condition ([design doc](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/scheduling/taint-node-by-condition.md))

#### Storage

* [stable] Mount options

  * The ability to specify mount options for volumes is moved from beta to stable.

  * A new `MountOptions` field in the `PersistentVolume` spec is available to specify mount options. This field replaces an annotation.

  * A new `MountOptions` field in the `StorageClass` spec allows configuration of mount options for dynamically provisioned volumes.

* [stable] Support Attach and Detach operations for ReadWriteOnce (RWO) volumes that use iSCSI and Fibre Channel plugins.

* [stable] Expose storage usage metrics

  * The available capacity of a given Persistent Volume (PV) is available by calling the Kubernetes metrics API.

* [stable] Volume plugin metrics

  * Success and latency metrics for all Kubernetes calls are available by calling the Kubernetes metrics API. You can request volume operations, including mount, unmount, attach, detach, provision, and delete.

* [stable] The PV spec for Azure File, CephFS, iSCSI, and Glusterfs is modified to reference namespaced resources.

* [stable] You can now customize the iSCSI initiator name per volume in the iSCSI volume plugin.

* [stable] You can now specify the World Wide Identifier (WWID) for the volume identifier in the Fibre Channel volume plugin.

* [beta] Reclaim policy in StorageClass

  * You can now configure the reclaim policy in StorageClass, instead of defaulting to `delete` for dynamically provisioned volumes.

* [alpha] Volume resizing

  * You can now increase the size of a volume by calling the Kubernetes API.

  * For alpha, this feature increases only the size of the underlying volume. It does not support resizing the file system.

  * For alpha, volume resizing supports only Gluster volumes.

* [alpha] Provide capacity isolation and resource management for local ephemeral storage

  * You can now set container requests, container limits, and node allocatable reservations for the new `ephemeral-storage` resource.

  * The `ephemeral-storage` resource includes all the disk space a container might consume with container overlay or scratch.

* [alpha] Mount namespace propagation

  * The `VolumeMount.Propagation` field for `VolumeMount` in pod containers is now available.

  * You can now set `VolumeMount.Propagation` to `Bidirectional` to enable a particular mount for a container to propagate itself to the host or other containers.

* [alpha] Improve Flex volume deployment

  * Flex volume driver deployment is simplified in the following ways:

    * New driver files can now be automatically discovered and initialized without requiring a kubelet or controller-manager restart.

    * A sample DaemonSet to deploy Flexvolume drivers is now available.

* [prototype] Volume snapshots

  * You can now create a volume snapshot by calling the Kubernetes API.

  * Note that the prototype does not support quiescing before snapshot, so snapshots might be inconsistent.

  * In the prototype phase, this feature is external to the core Kubernetes. It's available at https://github.com/kubernetes-incubator/external-storage/tree/master/snapshot.

### Cluster Federation

#### [alpha] Federated Jobs

Federated Jobs that are automatically deployed to multiple clusters
are now supported. Cluster selection and weighting determine how Job
parallelism and completions are spread across clusters. Federated Job
status reflects the aggregate status across all underlying cluster
jobs.

#### [alpha] Federated Horizontal Pod Autoscaling (HPA)

Federated HPAs are similar to the traditional Kubernetes HPAs, except
that they span multiple clusters. Creating a Federated HPA targeting
multiple clusters ensures that cluster-level autoscalers are
consistently deployed across those clusters, and dynamically managed
to ensure that autoscaling can occur optimially in all clusters,
within a set of global constraints on the the total number of replicas
permitted across all clusters.  If replicas are not
required in some clusters due to low system load or insufficient quota
or capacity in those clusters, additional replicas are made available
to the autoscalers in other clusters if required.

### Node Components

#### Autoscaling and Metrics

* Support for custom metrics in the Horizontal Pod Autoscaler is now at v1beta1. The associated metrics APIs (custom metrics and resource/master metrics) were also moved to v1beta1. For more information, see [Before Upgrading](#before-upgrading).

* `metrics-server` is now the recommended way to provide the resource
  metrics API. Deploy `metrics-server` as an add-on in the same way that you deploy Heapster.

##### Cluster Autoscaler

* Cluster autoscaler is now GA
* Cluster support size is increased to 1000 nodes
* Respect graceful pod termination of up to 10 minutes
* Handle zone stock-outs and failures
* Improve monitoring and error reporting

#### Container Runtime Interface (CRI)

* [alpha] Add a CRI validation test suite and CRI command-line tools. ([#292](https://github.com/kubernetes/features/issues/292), [@feiskyer](https://github.com/feiskyer))

* [stable] [cri-o](https://github.com/kubernetes-incubator/cri-o): CRI implementation for OCI-based runtimes [@mrunalp]

  * Passed all the Kubernetes 1.7 end-to-end conformance test suites.
  * Verification against Kubernetes 1.8 is planned soon after the release.

* [stable] [frakti](https://github.com/kubernetes/frakti): CRI implementation for hypervisor-based runtimes is now v1.1. [@feiskyer]

  * Enhance CNI plugin compatibility, supports flannel, calico, weave and so on.
  * Pass all CRI validation conformance tests and node end-to-end conformance tests.
  * Add experimental Unikernel support.

* [alpha] [cri-containerd](https://github.com/kubernetes-incubator/cri-containerd): CRI implementation for containerd is now v1.0.0-alpha.0, [@Random-Liu]

  * Feature complete. Support the full CRI API defined in v1.8.
  * Pass all the CRI validation tests and regular node end-to-end tests.
  * An ansible playbook is provided to configure a Kubernetes cri-containerd cluster with kubeadm.

* Add support in Kubelet to consume container metrics via CRI. [@yguo0905]
  * There are known bugs that result in errors when querying Kubelet's stats summary API. We expect to fix them in v1.8.1.

#### kubelet

* [alpha] Kubelet now supports alternative container-level CPU affinity policies by using the new CPU manager. ([#375](https://github.com/kubernetes/features/issues/375), [@sjenning](https://github.com/sjenning), [@ConnorDoyle](https://github.com/ConnorDoyle))

* [alpha] Applications may now request pre-allocated hugepages by using the new `hugepages` resource in the container resource requests. ([#275](https://github.com/kubernetes/features/issues/275), [@derekwaynecarr](https://github.com/derekwaynecarr))

* [alpha] Add support for dynamic Kubelet configuration. ([#281](https://github.com/kubernetes/features/issues/281), [@mtaufen](https://github.com/mtaufen))

* [alpha] Add the Hardware Device Plugins API. ([#368](https://github.com/kubernetes/features/issues/368), [@jiayingz], [@RenaudWasTaken])

* [stable] Upgrade cAdvisor to v0.27.1 with the enhancement for node monitoring. [@dashpole]

  * Fix journalctl leak
  * Fix container memory rss
  * Fix incorrect CPU usage with 4.7 kernel
  * OOM parser uses kmsg
  * Add hugepages support
  * Add CRI-O support

* Sharing a PID namespace between containers in a pod is disabled by default in version 1.8. To enable for a node, use the `--docker-disable-shared-pid=false` kubelet flag. Be aware that PID namespace sharing requires Docker version greater than or equal to 1.13.1.

* Fix issues related to the eviction manager.

* Fix inconsistent Prometheus cAdvisor metrics.

* Fix issues related to the local storage allocatable feature.

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

* Advanced audit allows logging failed login attempts.
 ([#51119](https://github.com/kubernetes/kubernetes/pull/51119), [@soltysh](https://github.com/soltysh))

* A `kubectl auth reconcile` subcommand has been added for applying RBAC resources. When passed a file which contains RBAC roles, rolebindings, clusterroles, or clusterrolebindings, it will compute covers and add the missing rules. ([#51636](https://github.com/kubernetes/kubernetes/pull/51636), [@deads2k](https://github.com/deads2k))

### Cluster Lifecycle

#### kubeadm

* [beta] A new `upgrade` subcommand allows you to automatically upgrade a self-hosted cluster created with kubeadm. ([#296](https://github.com/kubernetes/features/issues/296), [@luxas](https://github.com/luxas))

* [alpha] An experimental self-hosted cluster can now easily be created with `kubeadm init`. Enable the feature by setting the SelfHosting feature gate to true: `--feature-gates=SelfHosting=true` ([#296](https://github.com/kubernetes/features/issues/296), [@luxas](https://github.com/luxas))
   * **NOTE:** Self-hosting will be the default way to host the control plane in the next release, v1.9

* [alpha] A new `phase` subcommand supports performing only subtasks of the full `kubeadm init` flow. Combined with fine-grained configuration, kubeadm is now more easily consumable by higher-level provisioning tools like kops or GKE. ([#356](https://github.com/kubernetes/features/issues/356), [@luxas](https://github.com/luxas))
   * **NOTE:** This command is currently staged under `kubeadm alpha phase` and will be graduated to top level in a future release.

#### kops

* [alpha] Added support for targeting bare metal (or non-cloudprovider) machines. ([#360](https://github.com/kubernetes/features/issues/360), [@justinsb](https://github.com/justinsb)).

* [alpha] kops now supports [running as a server](https://github.com/kubernetes/kops/blob/master/docs/api-server/README.md). ([#359](https://github.com/kubernetes/features/issues/359), [@justinsb](https://github.com/justinsb)).

* [beta] GCE support is promoted from alpha to beta. ([#358](https://github.com/kubernetes/features/issues/358), [@justinsb](https://github.com/justinsb)).

#### Cluster Discovery/Bootstrap

* [beta] The authentication and verification mechanism called Bootstrap Tokens is improved. Use Bootstrap Tokens to easily add new node identities to a cluster. ([#130](https://github.com/kubernetes/features/issues/130), [@luxas](https://github.com/luxas), [@jbeda](https://github.com/jbeda)).

#### Multi-platform

* [alpha] The Conformance e2e test suite now passes on the arm, arm64, and ppc64le platforms. ([#288](https://github.com/kubernetes/features/issues/288), [@luxas](https://github.com/luxas), [@mkumatag](https://github.com/mkumatag), [@ixdy](https://github.com/ixdy))

#### Cloud Providers

* [alpha] Support is improved for the pluggable, out-of-tree and out-of-core cloud providers. ([#88](https://github.com/kubernetes/features/issues/88), [@wlan0](https://github.com/wlan0))

### Network

#### network-policy

* [beta] Apply NetworkPolicy based on CIDR ([#50033](https://github.com/kubernetes/kubernetes/pull/50033), [@cmluciano](https://github.com/cmluciano))

* [beta] Support EgressRules in NetworkPolicy ([#51351](https://github.com/kubernetes/kubernetes/pull/51351), [@cmluciano](https://github.com/cmluciano))

#### kube-proxy ipvs mode

[alpha] Support ipvs mode for kube-proxy([#46580](https://github.com/kubernetes/kubernetes/pull/46580), [@haibinxie](https://github.com/haibinxie))

### API Machinery

#### kube-apiserver
* Fixed an issue with `APIService` auto-registration. This issue affected rolling restarts of HA API servers that added or removed API groups being served.([#51921](https://github.com/kubernetes/kubernetes/pull/51921))

* [Alpha] The Kubernetes API server now supports the ability to break large LIST calls into multiple smaller chunks. A client can specify a limit to the number of results to return. If more results exist, a token is returned that allows the client to continue the previous list call repeatedly until all results are retrieved.  The resulting list is identical to a list call that does not perform chunking, thanks to capabilities provided by etcd3.  This allows the server to use less memory and CPU when very large lists are returned. This feature is gated as APIListChunking and is not enabled by default. The 1.9 release will begin using this by default.([#48921](https://github.com/kubernetes/kubernetes/pull/48921))

* Pods that are marked for deletion and have exceeded their grace period, but are not yet deleted, no longer count toward the resource quota.([#46542](https://github.com/kubernetes/kubernetes/pull/46542))


#### Dynamic Admission Control

* Pod spec is mutable when the pod is uninitialized. The API server requires the pod spec to be valid even if it's uninitialized. Updating the status field of uninitialized pods is invalid.([#51733](https://github.com/kubernetes/kubernetes/pull/51733))

* Use of the alpha initializers feature now requires enabling the `Initializers` feature gate. This feature gate is automatically enabled if the `Initializers` admission plugin is enabled.([#51436](https://github.com/kubernetes/kubernetes/pull/51436))

* [Action required] The validation rule for metadata.initializers.pending[x].name is tightened. The initializer name must contain at least three segments, separated by dots. You can create objects with pending initializers and not rely on the API server to add pending initializers according to `initializerConfiguration`. If you do so, update the initializer name in the existing objects and the configuration files to comply with the new validation rule.([#51283](https://github.com/kubernetes/kubernetes/pull/51283))

* The webhook admission plugin now works even if the API server and the nodes are in two separate networks,for example, in GKE.
The webhook admission plugin now lets the webhook author use the DNS name of the service as the CommonName when generating the server cert for the webhook.
Action required:
Regenerate the server cert for the admission webhooks. Previously, the CN value could be ignored while generating the server cert for the admission webhook. Now you must set it to the DNS name of the webhook service: `<service.Name>.<service.Namespace>.svc`.([#50476](https://github.com/kubernetes/kubernetes/pull/50476))


#### Custom Resource Definitions (CRDs)
* [alpha] The CustomResourceDefinition API can now optionally
  [validate custom objects](https://kubernetes.io/docs/tasks/access-kubernetes-api/extend-api-custom-resource-definitions/#validation)
  based on a JSON schema provided in the CRD spec.
  Enable this alpha feature with the `CustomResourceValidation` feature gate in `kube-apiserver`.

#### Garbage Collector
* The garbage collector now supports custom APIs added via Custom Resource Definitions
  or aggregated API servers. The garbage collector controller refreshes periodically.
  Therefore, expect a latency of about 30 seconds between when an API is added and when
  the garbage collector starts to manage it.


#### Monitoring/Prometheus
* [action required] The WATCHLIST calls are now reported as WATCH verbs in prometheus for the apiserver_request_* series.  A new "scope" label is added to all apiserver_request_* values that is either 'cluster', 'resource', or 'namespace' depending on which level the query is performed at.([#52237](https://github.com/kubernetes/kubernetes/pull/52237))


#### Go Client
* Add support for client-side spam filtering of events([#47367](https://github.com/kubernetes/kubernetes/pull/47367))


## External Dependencies

Continuous integration builds use Docker versions 1.11.2, 1.12.6, 1.13.1,
and 17.03.2. These versions were validated on Kubernetes 1.8. However,
consult an appropriate installation or upgrade guide before deciding what
versions of Docker to use.

- Docker 1.13.1 and 17.03.2

    - Shared PID namespace, live-restore, and overlay2 were validated.

    - **Known issues**

        - The default iptables FORWARD policy was changed from ACCEPT to
          DROP, which causes outbound container traffic to stop working by
          default. See
          [#40182](https://github.com/kubernetes/kubernetes/issues/40182) for
          the workaround.

        - The support for the v1 registries was removed.

- Docker 1.12.6

    - Overlay2 and live-restore are not validated.

    - **Known issues**

        - Shared PID namespace does not work properly.
          ([#207](https://github.com/kubernetes/community/pull/207#issuecomment-281870043))

        - Docker reports incorrect exit codes for containers.
          ([#41516](https://github.com/kubernetes/kubernetes/issues/41516))

- Docker 1.11.2

    - **Known issues**

        - Kernel crash with Aufs storage driver on Debian Jessie
          ([#27885](https://github.com/kubernetes/kubernetes/issues/27885)).
          The issue can be identified by using the node problem detector.

        - File descriptor leak on init/control.
          ([#275](https://github.com/containerd/containerd/issues/275))

        - Additional memory overhead per container.
          ([#21737](https://github.com/kubernetes/kubernetes/pull/21737))

        - Processes may be leaked when Docker is repeatedly terminated in a short
          time frame.
          ([#41450](https://github.com/kubernetes/kubernetes/issues/41450))
