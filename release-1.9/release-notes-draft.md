## 1.9 Release Notes

## WARNING: etcd backup strongly recommended

Before updating to 1.9, you are strongly recommended to back up your etcd data. Consult the installation procedure you are using (kargo, kops, kube-up, kube-aws, kubeadm etc) for specific advice.

Kubernetes 1.9 now defaults to etcd 3.1. Some upgrade methods might upgrade etcd from 3.0 to 3.1 automatically when you upgrade from Kubernetes 1.8, unless you specify otherwise. Because [etcd does not support downgrading](https://coreos.com/etcd/docs/latest/upgrades/upgrade_3_1.html), you'll need to either remain on etcd 3.1 or restore from a backup if you want to downgrade back to Kubernetes 1.8.

## Introduction to 1.9.0

Kubernetes version 1.9 includes new features and enhancements, as well as fixes to identified issues. The release notes contain a brief overview of the important changes introduced in this release. The content is organized by Special Interest Group ([SIG](https://github.com/kubernetes/community/blob/master/sig-list.md)).

For initial installations, see the [Setup topics](https://kubernetes.io/docs/setup/pick-right-solution/) in the Kubernetes documentation.

To upgrade to this release from a previous version, first take any actions required [Before Upgrading](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.9.md#before-upgrading).

For more information about this release and for the latest documentation, see the [Kubernetes documentation](https://kubernetes.io/docs/home/).

## Major themes

Kubernetes is developed by community members whose work is organized into
[Special Interest Groups][]. For the 1.9 release, SIGs provide the
themes that guided their work.

[Special Interest Groups]: https://github.com/kubernetes/community/blob/master/sig-list.md

### API Machinery

Extensibility. SIG API Machinery added a new class of admission control webhooks (mutating), and brought the admission control webhooks to beta.

### Apps

The core workloads API, which is composed of the DaemonSet, Deployment, ReplicaSet, and StatefulSet kinds, has been promoted to GA stability in the apps/v1 group version. The apps/v1beta2 group version is deprecated. All new code should use the kinds in the apps/v1 group version. 

### Auth

SIG Auth focused on extension-related authorization improvements. Permissions can now be added to the built-in RBAC admin/edit/view roles using [cluster role aggregation](https://kubernetes.io/docs/admin/authorization/rbac/#aggregated-clusterroles). [Webhook authorizers](https://kubernetes.io/docs/admin/authorization/webhook/) can now deny requests and short-circuit checking subsequent authorizers. Performance and usability of the beta [PodSecurityPolicy](https://kubernetes.io/docs/concepts/policy/pod-security-policy/) feature was also improved.

### Azure

SIG Azure worked on improvements in the cloud provider, including significant work on the Azure Load Balancer implementation.

### Cluster Lifecycle

SIG Cluster Lifecycle has been focusing on improving kubeadm in order to bring it to GA in a future release and developing the [Cluster API](https://github.com/kubernetes/kube-deploy/tree/master/cluster-api). For kubeadm, most new features have gone in as alpha features, for instance support for CoreDNS, IPv6 and Dynamic Kubelet Configuration. We expect to graduate these features to beta and beyond in the next release. The initial Cluster API spec and GCE sample implementation were developed from scratch during this cycle, and we look forward to stabilizing them into something production-grade during 2018.

### Network

In v1.9 SIG Network has implemented alpha support for IPv6, and alpha support for CoreDNS as a drop-in replacement for kube-dns. Additionally, SIG Network has begun the deprecation process for the extensions/v1beta1 NetworkPolicy API in favor of the networking.k8s.io/v1 equivalent.

### Node

SIG Node iterated on the ability to support more workloads with better performance and improved reliability.  Alpha features were improved around hardware accelerator support, device plugins enablement, and cpu pinning policies to enable us to graduate these features to beta in a future release.  In addition a number of reliability and performance enhancements were made across the node to help operators in production. 

### OpenStack

Configuration simplification through smarter defaults and the use of auto-detection wherever feasible (Block Storage API versions, Security Groups) as well as updating API support:

*   Block Storage (Cinder) V3 is now supported.
*   Load Balancer (Octavia) V2 is now supported, in addition to Neutron LBaaS V2.
*   Neutron LBaas V1 support has been removed.

This allows Kubernetes to take full advantage of the relevant services as exposed by OpenStack clouds. Refer to the [Cloud Providers](https://kubernetes.io/docs/concepts/cluster-administration/cloud-providers/#openstack) documentation for more information.

### Storage

[SIG Storage](https://github.com/kubernetes/community/tree/master/sig-storage) is responsible for storage and volume plugin components.

For the 1.9 release, SIG Storage made Kubernetes more pluggable and modular by introducing an alpha implementation of Container Storage Interface (CSI). CSI will make installing new volume plugins as easy as deploying a pod, and enable third-party storage providers to develop their plugins without the need to add code to the core Kubernetes codebase.

The SIG also focused on adding functionality to the Kubernetes volume subsystem: alpha support for exposing volumes as block devices inside containers, extending the alpha volume-resizing support to more volume plugins, and topology aware volume scheduling.

### Windows

We are advancing support for Windows Server and Windows Server Containers to beta along with continued feature and functional advancements on both the Kubernetes and Windows platforms. This opens the door for many Windows-specific applications and workloads to run on Kubernetes, significantly expanding the implementation scenarios and the enterprise reach of Kubernetes.

## Before Upgrading 

Consider the following changes, limitations, and guidelines before you upgrade:

#### **API Machinery**

*   The admission API, which is used when the API server calls admission control webhooks, is moved from `admission.v1alpha1` to `admission.v1beta1`. You must **delete any existing webhooks before you upgrade** your cluster, and update them to use the latest API. This change is not backward compatible.
*   The admission webhook configurations API, part of the admissionregistration API, is now at v1beta1. Delete any existing webhook configurations before you upgrade, and update your configuration files to use the latest API. For this and the previous change, see also [the documentation]([https://kubernetes.io/docs/admin/extensible-admission-controllers/#external-admission-webhooks](https://kubernetes.io/docs/admin/extensible-admission-controllers/#external-admission-webhooks)).
*   A new `ValidatingAdmissionWebhook` is added (replacing `GenericAdmissionWebhook`) and is available in the generic API server. You must update your API server configuration file to pass the webhook to the `--admission-control` flag. ([#55988](https://github.com/kubernetes/kubernetes/pull/55988),[ @caesarxuchao](https://github.com/caesarxuchao))  ([#54513](https://github.com/kubernetes/kubernetes/pull/54513),[ @deads2k](https://github.com/deads2k))
*   The deprecated options `--portal-net` and `--service-node-ports` for the API server are removed. ([#52547](https://github.com/kubernetes/kubernetes/pull/52547),[ @xiangpengzhao](https://github.com/xiangpengzhao))

#### **Auth**

*   PodSecurityPolicy: A compatibility issue with the allowPrivilegeEscalation field that caused policies to start denying pods they previously allowed was fixed. If you defined PodSecurityPolicy objects using a 1.8.0 client or server and set allowPrivilegeEscalation to false, these objects must be reapplied after you upgrade. ([#53443](https://github.com/kubernetes/kubernetes/pull/53443),[ @liggitt](https://github.com/liggitt))
*   KMS: Alpha integration with GCP KMS was removed in favor of a future out-of-process extension point. Discontinue use of the GCP KMS integration and ensure [data has been decrypted](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/#decrypting-all-data) (or reencrypted with a different provider) before upgrading ([#54759](https://github.com/kubernetes/kubernetes/pull/54759),[ @sakshamsharma](https://github.com/sakshamsharma))

#### **CLI**

*   Swagger 1.2 validation is removed for kubectl. The options `--use-openapi` and `--schema-cache-dir` are also removed because they are no longer needed. ([#53232](https://github.com/kubernetes/kubernetes/pull/53232),[ @apelisse](https://github.com/apelisse))

#### **Cluster Lifecycle**

*   You must either specify the `--discovery-token-ca-cert-hash` flag to `kubeadm join`, or opt out of the CA pinning feature using `--discovery-token-unsafe-skip-ca-verification`.
*   The default `auto-detect` behavior of the kubelet's `--cloud-provider` flag is removed.
    *   You can manually set `--cloud-provider=auto-detect`, but be aware that this behavior will be removed completely in a future version.
    *   Best practice for version 1.9 and future versions is to explicitly set a cloud-provider. See [the documentation](https://kubernetes.io/docs/getting-started-guides/scratch/#cloud-providers)
*   The kubeadm `--skip-preflight-checks` flag is now deprecated and will be removed in a future release.
*   If you are using the cloud provider API to determine the external host address of the apiserver, set `--external-hostname` explicitly instead. The cloud provider detection has been deprecated and will be removed in the future ([#54516](https://github.com/kubernetes/kubernetes/pull/54516),[ @dims](https://github.com/dims))

#### **Multicluster**

*   Development of Kubernetes Federation has moved to [github.com/kubernetes/federation](https://github.com/kubernetes/federation). This move out of tree also means that Federation will begin releasing separately from Kubernetes. Impact: 
    *   Federation-specific behavior will no longer be included in kubectl
    *   kubefed will no longer be released as part of Kubernetes
    *   The Federation servers will no longer be included in the hyperkube binary and image. ([#53816](https://github.com/kubernetes/kubernetes/pull/53816),[ @marun](https://github.com/marun))

#### **Node**

*   The kubelet `--network-plugin-dir` flag is removed. This flag was deprecated in version 1.7, and is replaced with `--cni-bin-dir`. ([#53564](https://github.com/kubernetes/kubernetes/pull/53564),[ @supereagle](https://github.com/supereagle))
*   kubelet's `--cloud-provider` flag no longer defaults to "auto-detect". If you want cloud-provider support in kubelet, you must set a specific cloud-provider explicitly. ([#53573](https://github.com/kubernetes/kubernetes/pull/53573),[ @dims](https://github.com/dims))

#### **Network**

*   NetworkPolicy objects are now stored in etcd in v1 format. After you upgrade to version 1.9, make sure that all NetworkPolicy objects are migrated to v1. ([#51955](https://github.com/kubernetes/kubernetes/pull/51955), [@danwinship](https://github.com/danwinship))
*   The API group/version for the kube-proxy configuration has changed from `componentconfig/v1alpha1` to `kubeproxy.config.k8s.io/v1alpha1`. If you are using a config file for kube-proxy instead of the command line flags, you must change its apiVersion to `kubeproxy.config.k8s.io/v1alpha1`. ([#53645](https://github.com/kubernetes/kubernetes/pull/53645), [@xiangpengzhao](https://github.com/xiangpengzhao))
*   The "ServiceNodeExclusion" feature gate must now be enabled in order for the `alpha.service-controller.kubernetes.io/exclude-balancer` annotation on nodes to be honored. ([#54644](https://github.com/kubernetes/kubernetes/pull/54644),[ @brendandburns](https://github.com/brendandburns))

#### **Scheduling**

*   Taint key `unreachable` is now in GA.
*   Taint key `notReady` is changed to `not-ready`, and is also now in GA.
*   These changes are automatically updated for taints. Tolerations for these taints must be updated manually. Specifically, you must:
    *   Change `node.alpha.kubernetes.io/notReady` to `node.kubernetes.io/not-ready`
    *   Change `node.alpha.kubernetes.io/unreachable` to  `node.kubernetes.io/unreachable`
*   The `node.kubernetes.io/memory-pressure` taint now respects the configured whitelist.  To use it, you must add it to the whitelist.([#55251](https://github.com/kubernetes/kubernetes/pull/55251),[ @deads2k](https://github.com/deads2k))
*   Refactor kube-scheduler configuration ([#52428](https://github.com/kubernetes/kubernetes/pull/52428))
    *   The kube-scheduler command now supports a --config flag which is the location of a file containing a serialized scheduler configuration. Most other kube-scheduler flags are now deprecated. ([#52562](https://github.com/kubernetes/kubernetes/pull/52562),[ @ironcladlou](https://github.com/ironcladlou))
*   Opaque integer resources (OIR), which were (deprecated in v1.8.), have been removed. ([#55103](https://github.com/kubernetes/kubernetes/pull/55103),[ @ConnorDoyle](https://github.com/ConnorDoyle))

#### **Storage**

*   [alpha] The LocalPersistentVolumes alpha feature now also requires the VolumeScheduling alpha feature.  This is a breaking change, and the following changes are required: 
    *   The VolumeScheduling feature gate must also be enabled on kube-scheduler and kube-controller-manager components.
    *   The NoVolumeNodeConflict predicate has been removed.  For non-default schedulers, update your scheduler policy.
    *   The CheckVolumeBinding predicate must be enabled in non-default schedulers. ([#55039](https://github.com/kubernetes/kubernetes/pull/55039),[ @msau42](https://github.com/msau42))

#### **OpenStack**

*   Remove the LbaasV1 of OpenStack cloud provider, currently only support LbaasV2. ([#52717](https://github.com/kubernetes/kubernetes/pull/52717),[ @FengyunPan](https://github.com/FengyunPan))

## Known Issues

This section contains a list of known issues reported in Kubernetes 1.9 release. The content is populated from the [v1.9.x known issues and FAQ accumulator](https://github.com/kubernetes/kubernetes/issues/57159).

*   If you are adding Windows Server Virtual Machines as nodes to your Kubernetes environment, there is a compatibility issue with certain virtualization products. Specifically the Windows version of the kubelet.exe calls `GetPhysicallyInstalledSystemMemory` to get the physical memory installed on Windows machines and reports it as part of node metrics to heapster. This API call fails for VMware and VirtualBox virtualization environments. This issue is not present in bare metal Windows deployments, in Hyper-V, or on some of the popular public cloud providers.

## Deprecations

This section provides an overview of deprecated API versions, options, flags, and arguments. Deprecated means that we intend to remove the capability from a future release. After removal, the capability will no longer work. The sections are organized by SIGs.


##### **API Machinery**



*   The kube-apiserver `--etcd-quorum-read` flag is deprecated and the ability to switch off quorum read will be removed in a future release. ([#53795](https://github.com/kubernetes/kubernetes/pull/53795),[ @xiangpengzhao](https://github.com/xiangpengzhao))
*   The `/ui` redirect in kube-apiserver is deprecated and will be removed in Kubernetes 1.10. ([#53046](https://github.com/kubernetes/kubernetes/pull/53046), [@maciaszczykm](https://github.com/maciaszczykm))
*   `etcd2` as a backend is deprecated and support will be removed in Kubernetes 1.13.


##### **Auth**



*   Default controller-manager options for `--cluster-signing-cert-file` and `--cluster-signing-key-file` are deprecated and will be removed in a future release. ([#54495](https://github.com/kubernetes/kubernetes/pull/54495),[ @mikedanese](https://github.com/mikedanese))
*   RBAC objects are now stored in etcd in v1 format. After upgrading to 1.9, ensure all RBAC objects (Roles, RoleBindings, ClusterRoles, ClusterRoleBindings) are at v1. v1alpha1 support is deprecated and will be removed in a future release. ([#52950](https://github.com/kubernetes/kubernetes/pull/52950),[ @liggitt](https://github.com/liggitt))


##### **Cluster Lifecycle**



*   kube-apiserver: `--ssh-user` and `--ssh-keyfile` are now deprecated and will be removed in a future release. Users of SSH tunnel functionality in Google Container Engine for the Master -> Cluster communication should plan alternate methods for bridging master and node networks. ([#54433](https://github.com/kubernetes/kubernetes/pull/54433),[ @dims](https://github.com/dims))
*   The kubeadm `--skip-preflight-checks` flag is now deprecated and will be removed in a future release.
*   If you are using the cloud provider API to determine the external host address of the apiserver, set `--external-hostname` explicitly instead. The cloud provider detection has been deprecated and will be removed in the future ([#54516](https://github.com/kubernetes/kubernetes/pull/54516),[ @dims](https://github.com/dims))


##### **Network**



*   The NetworkPolicy extensions/v1beta1 API is now deprecated and will be removed in a future release. This functionality has been migrated to a dedicated v1 API - networking.k8s.io/v1. v1beta1 Network Policies can be upgraded to the v1 API with the [cluster/update-storage-objects.sh script](https://github.com/danwinship/kubernetes/blob/master/cluster/update-storage-objects.sh). Documentation can be found [here](https://kubernetes.io/docs/concepts/services-networking/network-policies/). ([#56425](https://github.com/kubernetes/kubernetes/pull/56425), [@cmluciano](https://github.com/cmluciano))


##### **Storage**



*   The `volume.beta.kubernetes.io/storage-class` annotation is deprecated. It will be removed in a future release. For the StorageClass API object, use v1, and in place of the annotation use `v1.PersistentVolumeClaim.Spec.StorageClassName` and `v1.PersistentVolume.Spec.StorageClassName` instead. ([#53580](https://github.com/kubernetes/kubernetes/pull/53580),[ @xiangpengzhao](https://github.com/xiangpengzhao))


##### **Scheduling**



*   The kube-scheduler command now supports a --config flag, which is the location of a file containing a serialized scheduler configuration. Most other kube-scheduler flags are now deprecated. ([#52562](https://github.com/kubernetes/kubernetes/pull/52562),[ @ironcladlou](https://github.com/ironcladlou))


##### **Node**



*   The kubelet's `--enable-custom-metrics` flag is now deprecated. ([#54154](https://github.com/kubernetes/kubernetes/pull/54154),[ @mtaufen](https://github.com/mtaufen))


## Notable Changes


#### **Workloads API (apps/v1)**

As announced with the release of version 1.8, the Kubernetes Workloads API is at v1 in version 1.9. This API consists of the DaemonSet, Deployment, ReplicaSet and StatefulSet kinds.


#### **API Machinery**


##### **Admission Control**



*   Admission webhooks are now in beta, and include the following:
    *   Mutation support for admission webhooks. ([#54892](https://github.com/kubernetes/kubernetes/pull/54892),[ @caesarxuchao](https://github.com/caesarxuchao))
    *   Webhook admission now takes a config file that describes how to authenticate to webhook servers ([#54414](https://github.com/kubernetes/kubernetes/pull/54414),[ @deads2k](https://github.com/deads2k))
    *   The dynamic admission webhook now supports a URL in addition to a service reference, to accommodate out-of-cluster webhooks. ([#54889](https://github.com/kubernetes/kubernetes/pull/54889),[ @lavalamp](https://github.com/lavalamp))
    *   Added `namespaceSelector` to `externalAdmissionWebhook` configuration to allow applying webhooks only to objects in the namespaces that have matching labels. ([#54727](https://github.com/kubernetes/kubernetes/pull/54727),[ @caesarxuchao](https://github.com/caesarxuchao))
*   Metrics are added for monitoring admission plugins, including the new dynamic (webhook-based) ones. ([#55183](https://github.com/kubernetes/kubernetes/pull/55183),[ @jpbetz](https://github.com/jpbetz))
*   The PodSecurityPolicy annotation kubernetes.io/psp on pods is set only once on create. ([#55486](https://github.com/kubernetes/kubernetes/pull/55486),[ @sttts](https://github.com/sttts))


##### **API & API server**



*   Fixed a bug related to discovery information for scale subresources in the apps API group ([#54683](https://github.com/kubernetes/kubernetes/pull/54683),[ @liggitt](https://github.com/liggitt))
*   Fixed a bug that prevented client-go metrics from being registered in Prometheus. This bug affected multiple components. ([#53434](https://github.com/kubernetes/kubernetes/pull/53434),[ @crassirostris](https://github.com/crassirostris))


##### **Audit**



*   Fixed a bug so that `kube-apiserver` now waits for open connections to finish before exiting. This fix provides graceful shutdown and ensures that the audit backend no longer drops events on shutdown. ([#53695](https://github.com/kubernetes/kubernetes/pull/53695),[ @hzxuzhonghu](https://github.com/hzxuzhonghu))
*   Webhooks now always retry sending if a connection reset error is returned. ([#53947](https://github.com/kubernetes/kubernetes/pull/53947),[ @crassirostris](https://github.com/crassirostris))


##### **Custom Resources**



*   Validation of resources defined by a Custom Resource Definition (CRD) is now in beta ([#54647](https://github.com/kubernetes/kubernetes/pull/54647),[ @colemickens](https://github.com/colemickens))
*   An example CRD controller has been added, at [github.com/kubernetes/sample-controller](https://github.com/kubernetes/sample-controller). ([#52753](https://github.com/kubernetes/kubernetes/pull/52753),[ @munnerz](https://github.com/munnerz))
*   Custom resources served by CustomResourceDefinition objects now support field selectors for `metadata.name` and `metadata.namespace`. Also fixed an issue with watching a single object; earlier versions could watch only a collection, and so a watch on an instance would fail.  ([#53345](https://github.com/kubernetes/kubernetes/pull/53345),[ @ncdc](https://github.com/ncdc))


##### **Other**



*   kube-apiserver now runs with the default value for `service-cluster-ip-range` ([#52870](https://github.com/kubernetes/kubernetes/pull/52870),[ @jennybuckley](https://github.com/jennybuckley))
*   Add `--etcd-compaction-interval` to apiserver for controlling request of compaction to etcd3 from apiserver. ([#51765](https://github.com/kubernetes/kubernetes/pull/51765),[ @mitake](https://github.com/mitake))
*   The httpstream/spdy calls now support CIDR notation for NO_PROXY ([#54413](https://github.com/kubernetes/kubernetes/pull/54413),[ @kad](https://github.com/kad))
*   Code generation for CRD and User API server types is improved with the addition of two new scripts to k8s.io/code-generator: `generate-groups.sh` and `generate-internal-groups.sh`. ([#52186](https://github.com/kubernetes/kubernetes/pull/52186),[ @sttts](https://github.com/sttts))
*   [beta] Flag `--chunk-size={SIZE}` is added to `kubectl get` to customize the number of results returned in large lists of resources. This reduces the perceived latency of managing large clusters because the server returns the first set of results to the client much more quickly. Pass 0 to disable this feature.([#53768](https://github.com/kubernetes/kubernetes/pull/53768),[ @smarterclayton](https://github.com/smarterclayton))
*   [beta] API chunking via the limit and continue request parameters is promoted to beta in this release. Client libraries using the Informer or ListWatch types will automatically opt in to chunking. ([#52949](https://github.com/kubernetes/kubernetes/pull/52949),[ @smarterclayton](https://github.com/smarterclayton))
*   The `--etcd-quorum-read flag` now defaults to true to ensure correct operation with HA etcd clusters. This flag is deprecated and the flag, as well as the ability to turn off this functionality, will be removed in future versions. ([#53717](https://github.com/kubernetes/kubernetes/pull/53717),[ @liggitt](https://github.com/liggitt))
*   Add `events.k8s.io` api group with v1beta1 API containing redesigned event type. ([#49112](https://github.com/kubernetes/kubernetes/pull/49112),[ @gmarek](https://github.com/gmarek))
*   Fixed a bug where API discovery failures were crashing the kube controller manager via the garbage collector. ([#55259](https://github.com/kubernetes/kubernetes/pull/55259),[ @ironcladlou](https://github.com/ironcladlou))
*   `conversion-gen` is now usable in a context without a vendored k8s.io/kubernetes. The Kubernetes core API is removed from `default extra-peer-dirs`. ([#54394](https://github.com/kubernetes/kubernetes/pull/54394),[ @sttts](https://github.com/sttts))
*   Fixed a bug where the `client-gen` tag for code-generator required a newline between a comment block and a statement.  tag shortcomings when newline is omitted ([#53893](https://github.com/kubernetes/kubernetes/pull/53893)) ([#55233](https://github.com/kubernetes/kubernetes/pull/55233),[ @sttts](https://github.com/sttts))
*   The Apiserver proxy now rewrites the URL when a service returns an absolute path with the request's host. ([#52556](https://github.com/kubernetes/kubernetes/pull/52556),[ @roycaihw](https://github.com/roycaihw))
*   The gRPC library is updated to pick up data race fix ([#53124](https://github.com/kubernetes/kubernetes/pull/53124)) ([#53128](https://github.com/kubernetes/kubernetes/pull/53128),[ @dixudx](https://github.com/dixudx))
*   Fixed server name verification of aggregated API servers and webhook admission endpoints ([#56415](https://github.com/kubernetes/kubernetes/pull/56415),[ @liggitt](https://github.com/liggitt))


#### **Apps**



*   The `kubernetes.io/created-by` annotation is no longer added to controller-created objects. Use the `metadata.ownerReferences` item with controller set to `true` to determine which controller, if any, owns an object. ([#54445](https://github.com/kubernetes/kubernetes/pull/54445),[ @crimsonfaith91](https://github.com/crimsonfaith91))
*   StatefulSet controller now creates a label for each Pod in a StatefulSet. The label is `statefulset.kubernetes.io/pod-name`, where `pod-name` = the name of the Pod. This allows users to create a Service per Pod to expose a connection to individual Pods. ([#55329](https://github.com/kubernetes/kubernetes/pull/55329),[ @kow3ns](https://github.com/kow3ns))
*   DaemonSet status includes a new field named `conditions`, making it consistent with other workloads controllers. ([#55272](https://github.com/kubernetes/kubernetes/pull/55272),[ @janetkuo](https://github.com/janetkuo))
*   StatefulSet status now supports conditions, making it consistent with other core controllers in v1 ([#55268](https://github.com/kubernetes/kubernetes/pull/55268),[ @foxish](https://github.com/foxish))
*   The default garbage collection policy for Deployment, DaemonSet, StatefulSet, and ReplicaSet has changed from OrphanDependents to DeleteDependents when the deletion is requested through an `apps/v1` endpoint.  ([#55148](https://github.com/kubernetes/kubernetes/pull/55148),[ @dixudx](https://github.com/dixudx))
    *   Clients using older endpoints will be unaffected. This change is only at the REST API level and is independent of the default behavior of particular clients (e.g. this does not affect the default for the kubectl `--cascade` flag).
    *   If you upgrade your client-go libs and use the `AppsV1()` interface, please note that the default garbage collection behavior is changed.


#### **Auth**


##### **Audit**



*   RequestReceivedTimestamp and StageTimestamp are added to audit events ([#52981](https://github.com/kubernetes/kubernetes/pull/52981),[ @CaoShuFeng](https://github.com/CaoShuFeng))
*   Advanced audit policy now supports a policy wide omitStage ([#54634](https://github.com/kubernetes/kubernetes/pull/54634),[ @CaoShuFeng](https://github.com/CaoShuFeng))


##### **RBAC**



*   New permissions have been added to default RBAC roles ([#52654](https://github.com/kubernetes/kubernetes/pull/52654),[ @liggitt](https://github.com/liggitt)):
    *   The default admin and edit roles now include read/write permissions
    *   The view role includes read permissions on poddisruptionbudget.policy resources. 
*   RBAC rules can now match the same subresource on any resource using the form `*/(subresource)`. For example, `*/scale` matches requests to `replicationcontroller/scale`. ([#53722](https://github.com/kubernetes/kubernetes/pull/53722),[ @deads2k](https://github.com/deads2k))
*   The RBAC bootstrapping policy now allows authenticated users to create selfsubjectrulesreviews. ([#56095](https://github.com/kubernetes/kubernetes/pull/56095),[ @ericchiang](https://github.com/ericchiang))
*   RBAC ClusterRoles can now select other roles to aggregate. ([#54005](https://github.com/kubernetes/kubernetes/pull/54005),[ @deads2k](https://github.com/deads2k))
*   Fixed an issue with RBAC reconciliation that caused duplicated subjects in some bootstrapped RoleBinding objects on each restart of the API server. ([#53239](https://github.com/kubernetes/kubernetes/pull/53239),[ @enj](https://github.com/enj))


##### **Other**



*   Pod Security Policy can now manage access to specific FlexVolume drivers ([#53179](https://github.com/kubernetes/kubernetes/pull/53179),[ @wanghaoran1988](https://github.com/wanghaoran1988))
*   Audit policy files without apiVersion and kind are treated as invalid. ([#54267](https://github.com/kubernetes/kubernetes/pull/54267),[ @ericchiang](https://github.com/ericchiang))
*   Fixed a bug that where forbidden errors were encountered when accessing ReplicaSet and DaemonSets objects via the apps API group. ([#54309](https://github.com/kubernetes/kubernetes/pull/54309),[ @liggitt](https://github.com/liggitt))
*   Improved PodSecurityPolicy admission latency. ([#55643](https://github.com/kubernetes/kubernetes/pull/55643),[ @tallclair](https://github.com/tallclair))
*   kube-apiserver: `--oidc-username-prefix` and `--oidc-group-prefix` flags are now correctly enabled. ([#56175](https://github.com/kubernetes/kubernetes/pull/56175),[ @ericchiang](https://github.com/ericchiang))
*   If multiple PodSecurityPolicy objects allow a submitted pod, priority is given to policies that do not require default values for any fields in the pod spec. If default values are required, the first policy ordered by name that allows the pod is used. ([#52849](https://github.com/kubernetes/kubernetes/pull/52849),[ @liggitt](https://github.com/liggitt))
*   A new controller automatically cleans up Certificate Signing Requests that are Approved and Issued, or Denied. ([#51840](https://github.com/kubernetes/kubernetes/pull/51840),[ @jcbsmpsn](https://github.com/jcbsmpsn))
*   PodSecurityPolicies have been added for all in-tree cluster addons ([#55509](https://github.com/kubernetes/kubernetes/pull/55509),[ @tallclair](https://github.com/tallclair))


##### **GCE**



*   Added support for PodSecurityPolicy on GCE: `ENABLE_POD_SECURITY_POLICY=true` enables the admission controller, and installs policies for default addons. ([#52367](https://github.com/kubernetes/kubernetes/pull/52367),[ @tallclair](https://github.com/tallclair))


#### **Autoscaling**



*   HorizontalPodAutoscaler objects now properly functions on scalable resources in any API group. Fixed by adding a polymorphic scale client. ([#53743](https://github.com/kubernetes/kubernetes/pull/53743),[ @DirectXMan12](https://github.com/DirectXMan12))
*   Fixed a set of minor issues with Cluster Autoscaler 1.0.1 ([#54298](https://github.com/kubernetes/kubernetes/pull/54298),[ @mwielgus](https://github.com/mwielgus))
*   HPA tolerance is now configurable by setting the `horizontal-pod-autoscaler-tolerance` flag. ([#52275](https://github.com/kubernetes/kubernetes/pull/52275),[ @mattjmcnaughton](https://github.com/mattjmcnaughton))
*   Fixed a bug that allowed the horizontal pod autoscaler to allocate more `desiredReplica` objects than `maxReplica` objects in certain instances. ([#53690](https://github.com/kubernetes/kubernetes/pull/53690),[ @mattjmcnaughton](https://github.com/mattjmcnaughton))


#### **AWS**



*   Nodes can now use instance types (such as C5) that use NVMe. ([#56607](https://github.com/kubernetes/kubernetes/pull/56607), [@justinsb](https://github.com/justinsb))
*   Nodes are now unreachable if volumes are stuck in the attaching state. Implemented by applying a taint to the node. ([#55558](https://github.com/kubernetes/kubernetes/pull/55558),[ @gnufied](https://github.com/gnufied))
*   Volumes are now checked for available state before attempting to attach or delete a volume in EBS. ([#55008](https://github.com/kubernetes/kubernetes/pull/55008),[ @gnufied](https://github.com/gnufied))
*   Fixed a bug where error log messages were breaking into two lines. ([#49826](https://github.com/kubernetes/kubernetes/pull/49826),[ @dixudx](https://github.com/dixudx))
*   Fixed a bug so that volumes are now detached from stopped nodes. ([#55893](https://github.com/kubernetes/kubernetes/pull/55893),[ @gnufied](https://github.com/gnufied))
*   You can now override the health check parameters for AWS ELBs by specifying annotations on the corresponding service. The new annotations are: `healthy-threshold`, `unhealthy-threshold`, `timeout`, `interval`. The prefix for all annotations is  `service.beta.kubernetes.io/aws-load-balancer-healthcheck-`. ([#56024](https://github.com/kubernetes/kubernetes/pull/56024),[ @dimpavloff](https://github.com/dimpavloff))
*   Fixed a bug so that AWS ECR credentials are now supported in the China region. ([#50108](https://github.com/kubernetes/kubernetes/pull/50108),[ @zzq889](https://github.com/zzq889))
*   Added Amazon NLB support ([#53400](https://github.com/kubernetes/kubernetes/pull/53400),[ @micahhausler](https://github.com/micahhausler))
*   Additional annotations are now properly set or updated  for AWS load balancers ([#55731](https://github.com/kubernetes/kubernetes/pull/55731),[ @georgebuckerfield](https://github.com/georgebuckerfield))
*   AWS SDK is updated to version 1.12.7 ([#53561](https://github.com/kubernetes/kubernetes/pull/53561),[ @justinsb](https://github.com/justinsb))


#### **Azure**



*   Fixed several issues with properly provisioning Azure disk storage ([#55927](https://github.com/kubernetes/kubernetes/pull/55927),[ @andyzhangx](https://github.com/andyzhangx))
*   A new service annotation `service.beta.kubernetes.io/azure-dns-label-name` now sets the Azure DNS label for a public IP address. ([#47849](https://github.com/kubernetes/kubernetes/pull/47849),[ @tomerf](https://github.com/tomerf))
*   Support for GetMountRefs function added; warning messages no longer displayed. ([#54670](https://github.com/kubernetes/kubernetes/pull/54670), [#52401](https://github.com/kubernetes/kubernetes/pull/52401),[ @andyzhangx](https://github.com/andyzhangx))
*   Fixed an issue where an Azure PersistentVolume object would crash because the value of `volumeSource.ReadOnly` was set to nil. ([#54607](https://github.com/kubernetes/kubernetes/pull/54607),[ @andyzhangx](https://github.com/andyzhangx))
*   Fixed an issue with Azure disk mount failures on CoreOS and some other distros ([#54334](https://github.com/kubernetes/kubernetes/pull/54334),[ @andyzhangx](https://github.com/andyzhangx))
*   GRS, RAGRS storage account types are now supported for Azure disks. ([#55931](https://github.com/kubernetes/kubernetes/pull/55931),[ @andyzhangx](https://github.com/andyzhangx))
*   Azure NSG rules are now restricted so that external access is allowed only to the load balancer IP. ([#54177](https://github.com/kubernetes/kubernetes/pull/54177),[ @itowlson](https://github.com/itowlson))
*   Azure NSG rules can be consolidated to reduce the likelihood of hitting Azure resource limits (available only in regions where the Augmented Security Groups preview is available).  ([#55740](https://github.com/kubernetes/kubernetes/pull/55740), [@itowlson](https://github.com/itowlson))
*   The Azure SDK is upgraded to v11.1.1. ([#54971](https://github.com/kubernetes/kubernetes/pull/54971),[ @itowlson](https://github.com/itowlson))
*   You can now create Windows mount paths  ([#51240](https://github.com/kubernetes/kubernetes/pull/51240),[ @andyzhangx](https://github.com/andyzhangx))
*   Fixed a controller manager crash issue on a manually created k8s cluster. ([#53694](https://github.com/kubernetes/kubernetes/pull/53694),[ @andyzhangx](https://github.com/andyzhangx))
*   Azure-based clusters now support unlimited mount points. ([#54668](https://github.com/kubernetes/kubernetes/pull/54668)) ([#53629](https://github.com/kubernetes/kubernetes/pull/53629),[ @andyzhangx](https://github.com/andyzhangx))
*   Load balancer reconciliation now considers NSG rules based not only on Name, but also on Protocol, SourcePortRange, DestinationPortRange, SourceAddressPrefix, DestinationAddressPrefix, Access, and Direction. This change makes it possible to update NSG rules under more conditions.  ([#55752](https://github.com/kubernetes/kubernetes/pull/55752),[ @kevinkim9264](https://github.com/kevinkim9264))
*   Custom mountOptions for the azurefile StorageClass object are now respected. Specifically, `dir_mode` and `file_mode` can now be customized. ([#54674](https://github.com/kubernetes/kubernetes/pull/54674),[ @andyzhangx](https://github.com/andyzhangx))
*   Azure Load Balancer Auto Mode: Services can be annotated to allow auto selection of available load balancers and to provide specific availability sets that host the load balancers. (e.g. `service.beta.kubernetes.io/azure-load-balancer-mode=auto|as1,as2...`)


#### **CLI**


##### **Kubectl**



*   `kubectl cp` can now copy a remote file into a local directory. ([#46762](https://github.com/kubernetes/kubernetes/pull/46762),[ @bruceauyeung](https://github.com/bruceauyeung))
*   `kubectl cp` now honors destination names for directories. A complete directory is now copied; in previous versions only the file contents were copied. ([#51215](https://github.com/kubernetes/kubernetes/pull/51215),[ @juanvallejo](https://github.com/juanvallejo))
*   You can now use `kubectl get` with a fieldSelector. ([#50140](https://github.com/kubernetes/kubernetes/pull/50140),[ @dixudx](https://github.com/dixudx))
*   Secret data containing Docker registry auth objects is now generated using the config.json format ([#53916](https://github.com/kubernetes/kubernetes/pull/53916),[ @juanvallejo](https://github.com/juanvallejo))
*   `kubectl apply` now calculates the diff between the current and new configurations based on the OpenAPI spec. If the OpenAPI spec is not available, it falls back to baked-in types. ([#51321](https://github.com/kubernetes/kubernetes/pull/51321),[ @mengqiy](https://github.com/mengqiy))
*   `kubectl explain` now explains `apiservices` and `customresourcedefinition`. (Updated to use OpenAPI instead of Swagger 1.2.) ([#53228](https://github.com/kubernetes/kubernetes/pull/53228),[ @apelisse](https://github.com/apelisse))
*   `kubectl get` now uses OpenAPI schema extensions by default to select columns for custom types. ([#53483](https://github.com/kubernetes/kubernetes/pull/53483),[ @apelisse](https://github.com/apelisse))
*   kubectl `top node` now sorts by name and `top pod` sorts by namespace. Fixed a bug where results were inconsistently sorted. ([#53560](https://github.com/kubernetes/kubernetes/pull/53560),[ @dixudx](https://github.com/dixudx))
*   Added --dry-run option to kubectl drain. ([#52440](https://github.com/kubernetes/kubernetes/pull/52440),[ @juanvallejo](https://github.com/juanvallejo))
*   Kubectl now outputs <none> for columns specified by -o custom-columns but not found in object, rather than "xxx is not found" ([#51750](https://github.com/kubernetes/kubernetes/pull/51750),[ @jianhuiz](https://github.com/jianhuiz))
*   `kubectl create pdb` no longer sets the min-available field by default. ([#53047](https://github.com/kubernetes/kubernetes/pull/53047),[ @yuexiao-wang](https://github.com/yuexiao-wang))
*   The canonical pronunciation of kubectl is "cube control".
*   Added --raw to kubectl create to POST using the normal transport. ([#54245](https://github.com/kubernetes/kubernetes/pull/54245),[ @deads2k](https://github.com/deads2k))
*   Added kubectl `create priorityclass` subcommand ([#54858](https://github.com/kubernetes/kubernetes/pull/54858),[ @wackxu](https://github.com/wackxu))
*   Fixed an issue where `kubectl set` commands occasionally encountered conversion errors for ReplicaSet and DaemonSet objects ([#53158](https://github.com/kubernetes/kubernetes/pull/53158),[ @liggitt](https://github.com/liggitt))


#### **Cluster Lifecycle**


##### **API Server**



*   [alpha] Added an `--endpoint-reconciler-type` command-line argument to select the endpoint reconciler to use. The default is to use the 'master-count' reconciler which is the default for 1.9 and in use prior to 1.9. The 'lease' reconciler stores endpoints within the storage api for better cleanup of deleted (or removed) API servers. The 'none' reconciler is a no-op reconciler, which can be used in self-hosted environments.   ([#51698](https://github.com/kubernetes/kubernetes/pull/51698), [@rphillips](https://github.com/rphillips))


##### **Cloud Provider Integration**



*   Added `cloud-controller-manager` to `hyperkube`. This is useful as a number of deployment tools run all of the kubernetes components from the `hyperkube `image/binary. It also makes testing easier as a single binary/image can be built and pushed quickly.  ([#54197](https://github.com/kubernetes/kubernetes/pull/54197),[ @colemickens](https://github.com/colemickens))
*   Added the concurrent service sync flag to the Cloud Controller Manager to allow changing the number of workers. (`--concurrent-service-syncs`) ([#55561](https://github.com/kubernetes/kubernetes/pull/55561),[ @jhorwit2](https://github.com/jhorwit2))
*   kubelet's --cloud-provider flag no longer defaults to "auto-detect". If you want cloud-provider support in kubelet, you must set a specific cloud-provider explicitly. ([#53573](https://github.com/kubernetes/kubernetes/pull/53573),[ @dims](https://github.com/dims))


##### **Kubeadm**



*   kubeadm health checks can now be skipped with `--ignore-preflight-errors`; the `--skip-preflight-checks` flag is now deprecated and will be removed in a future release. ([#56130](https://github.com/kubernetes/kubernetes/pull/56130),[ @anguslees](https://github.com/anguslees)) ([#56072](https://github.com/kubernetes/kubernetes/pull/56072),[ @kad](https://github.com/kad))
*   You now have the option to use CoreDNS instead of KubeDNS. To install CoreDNS instead of kube-dns, set CLUSTER_DNS_CORE_DNS to 'true'. This support is experimental. ([#52501](https://github.com/kubernetes/kubernetes/pull/52501),[ @rajansandeep](https://github.com/rajansandeep)) ([#55728](https://github.com/kubernetes/kubernetes/pull/55728),[ @rajansandeep](https://github.com/rajansandeep))
*   Added --print-join-command flag for kubeadm token create. ([#56185](https://github.com/kubernetes/kubernetes/pull/56185),[ @mattmoyer](https://github.com/mattmoyer))
*   Added a new --etcd-upgrade keyword to kubeadm upgrade apply. When this keyword is specified, etcd's static pod gets upgraded to the etcd version officially recommended for a target kubernetes release. ([#55010](https://github.com/kubernetes/kubernetes/pull/55010),[ @sbezverk](https://github.com/sbezverk))
*   Kubeadm now supports Kubelet Dynamic Configuration on an alpha level. ([#55803](https://github.com/kubernetes/kubernetes/pull/55803),[ @xiangpengzhao](https://github.com/xiangpengzhao))
*   Added support for adding a Windows node ([#53553](https://github.com/kubernetes/kubernetes/pull/53553),[ @bsteciuk](https://github.com/bsteciuk))


##### **Juju**



*   Added support for SAN entries in the master node certificate. ([#54234](https://github.com/kubernetes/kubernetes/pull/54234),[ @hyperbolic2346](https://github.com/hyperbolic2346))
*   Add extra-args configs for scheduler and controller-manager to kubernetes-master charm ([#55185](https://github.com/kubernetes/kubernetes/pull/55185),[ @Cynerva](https://github.com/Cynerva))
*   Add support for RBAC ([#53820](https://github.com/kubernetes/kubernetes/pull/53820),[ @ktsakalozos](https://github.com/ktsakalozos))
*   Fixed iptables FORWARD policy for Docker 1.13 in kubernetes-worker charm ([#54796](https://github.com/kubernetes/kubernetes/pull/54796),[ @Cynerva](https://github.com/Cynerva))
*   Upgrading the kubernetes-master units now results in staged upgrades just like the kubernetes-worker nodes. Use the upgrade action in order to continue the upgrade process on each unit such as juju run-action kubernetes-master/0 upgrade ([#55990](https://github.com/kubernetes/kubernetes/pull/55990),[ @hyperbolic2346](https://github.com/hyperbolic2346))
*   Added extra_sans config option to kubeapi-load-balancer charm. This allows the user to specify extra SAN entries on the certificate generated for the load balancer. ([#54947](https://github.com/kubernetes/kubernetes/pull/54947),[ @hyperbolic2346](https://github.com/hyperbolic2346))
*   Added extra-args configs to kubernetes-worker charm ([#55334](https://github.com/kubernetes/kubernetes/pull/55334),[ @Cynerva](https://github.com/Cynerva))


##### **Other**



*   Base images have been bumped to Debian Stretch (9) ([#52744](https://github.com/kubernetes/kubernetes/pull/52744),[ @rphillips](https://github.com/rphillips))
*   Upgraded to go1.9. ([#51375](https://github.com/kubernetes/kubernetes/pull/51375),[ @cblecker](https://github.com/cblecker))
*   Add-on manager now supports HA masters. ([#55466](https://github.com/kubernetes/kubernetes/pull/55466),[ #55782](https://github.com/x13n),[ @x13n](https://github.com/x13n))
*   Hyperkube can now run from a non-standard path. ([#54570](https://github.com/kubernetes/kubernetes/pull/54570))


#### **GCP**



*   The service account made available on your nodes is now configurable. ([#52868](https://github.com/kubernetes/kubernetes/pull/52868),[ @ihmccreery](https://github.com/ihmccreery))
*   GCE nodes with NVIDIA GPUs attached now expose nvidia.com/gpu as a resource instead of alpha.kubernetes.io/nvidia-gpu. ([#54826](https://github.com/kubernetes/kubernetes/pull/54826),[ @mindprince](https://github.com/mindprince))
*   Docker's live-restore on COS/ubuntu can now be disabled ([#55260](https://github.com/kubernetes/kubernetes/pull/55260),[ @yujuhong](https://github.com/yujuhong))
*   Metadata concealment is now controlled by the ENABLE_METADATA_CONCEALMENT env var. See cluster/gce/config-default.sh for more info. ([#54150](https://github.com/kubernetes/kubernetes/pull/54150),[ @ihmccreery](https://github.com/ihmccreery))
*   Masquerading rules are now added by default to GCE/GKE ([#55178](https://github.com/kubernetes/kubernetes/pull/55178),[ @dnardo](https://github.com/dnardo))
*   Fixed master startup issues with concurrent iptables invocations. ([#55945](https://github.com/kubernetes/kubernetes/pull/55945),[ @x13n](https://github.com/x13n))
*   Fixed issue deleting internal load balancers when the firewall resource may not exist. ([#53450](https://github.com/kubernetes/kubernetes/pull/53450),[ @nicksardo](https://github.com/nicksardo))


#### **Instrumentation**


##### **Audit**



*   Adjust batching audit webhook default parameters: increase queue size, batch size, and initial backoff. Add throttling to the batching audit webhook. Default rate limit is 10 QPS. ([#53417](https://github.com/kubernetes/kubernetes/pull/53417),[ @crassirostris](https://github.com/crassirostris))
    *   These parameters are also now configurable. ([#56638](https://github.com/kubernetes/kubernetes/pull/56638), [@crassirostris](https://github.com/crassirostris))


##### **Other**



*   Fix a typo in prometheus-to-sd configuration, that drops some stackdriver metrics. ([#56473](https://github.com/kubernetes/kubernetes/pull/56473),[ @loburm](https://github.com/loburm))
*   [fluentd-elasticsearch addon] Elasticsearch and Kibana are updated to version 5.6.4 ([#55400](https://github.com/kubernetes/kubernetes/pull/55400),[ @mrahbar](https://github.com/mrahbar))
*   A new field is added to CRI container log format to support splitting a long log line into multiple lines. ([#55922](https://github.com/kubernetes/kubernetes/pull/55922),[ @Random-Liu](https://github.com/Random-Liu))
*   fluentd now supports CRI log format. ([#54777](https://github.com/kubernetes/kubernetes/pull/54777),[ @Random-Liu](https://github.com/Random-Liu))
*   Bring all prom-to-sd container to the same image version ([#54583](https://github.com/kubernetes/kubernetes/pull/54583))
    *   Reduce log noise produced by prometheus-to-sd, by bumping it to version 0.2.2. ([#54635](https://github.com/kubernetes/kubernetes/pull/54635),[ @loburm](https://github.com/loburm))
*   [fluentd-elasticsearch addon] Elasticsearch service name can be overridden via env variable ELASTICSEARCH_SERVICE_NAME ([#54215](https://github.com/kubernetes/kubernetes/pull/54215),[ @mrahbar](https://github.com/mrahbar))


#### **Multicluster**


##### **Federation**



*   Kubefed init now supports --imagePullSecrets and --imagePullPolicy, making it possible to use private registries. ([#50740](https://github.com/kubernetes/kubernetes/pull/50740),[ @dixudx](https://github.com/dixudx))
*   Updated cluster printer to enable --show-labels ([#53771](https://github.com/kubernetes/kubernetes/pull/53771),[ @dixudx](https://github.com/dixudx))
*   Kubefed init now supports --nodeSelector, enabling you to determine on what node the controller will be installed. ([#50749](https://github.com/kubernetes/kubernetes/pull/50749),[ @dixudx](https://github.com/dixudx))


#### **Network**


##### **IPv6**



*   [alpha] IPv6 support has been added. Notable IPv6 support details include:
    *   Support for IPv6-only Kubernetes cluster deployments. **<span style="text-decoration:underline;">Note:</span>** This feature does not provide dual-stack support.
    *   Support for IPv6 Kubernetes control and data planes.
    *   Support for Kubernetes IPv6 cluster deployments using kubeadm.
    *   Support for the iptables kube-proxy backend using ip6tables.
    *   Relies on CNI 0.6.0 binaries for IPv6 pod networking.
    *   Adds IPv6 support for kube-dns using SRV records.
    *   Caveats
        *   Only the CNI bridge and local-ipam plugins have been tested for the alpha release, although other CNI plugins do support IPv6.
        *   HostPorts are not supported.
*   An IPv6 network mask for pod or cluster cidr network must be /66 or longer. For example: 2001:db1::/66, 2001:dead:beef::/76, 2001:cafe::/118 are supported. 2001:db1::/64 is not supported
*   For details, see [the complete list of merged pull requests for IPv6 support](https://github.com/kubernetes/kubernetes/pulls?utf8=%E2%9C%93&q=is%3Apr+is%3Amerged+label%3Aarea%2Fipv6).


##### **IPVS**



*   You can now use the --cleanup-ipvs flag to tell kube-proxy whether to flush all existing ipvs rules in on startup ([#56036](https://github.com/kubernetes/kubernetes/pull/56036),[ @m1093782566](https://github.com/m1093782566))
*   Graduate kube-proxy IPVS mode to beta. ([#56623](https://github.com/kubernetes/kubernetes/pull/56623), [@m1093782566](https://github.com/m1093782566))


##### **Kube-Proxy**



*   Added iptables rules to allow Pod traffic even when default iptables policy is to reject. ([#52569](https://github.com/kubernetes/kubernetes/pull/52569),[ @tmjd](https://github.com/tmjd))
*   You can once again use 0 values for conntrack min, max, max per core, tcp close wait timeout, and tcp established timeout; this functionality was broken in 1.8. ([#55261](https://github.com/kubernetes/kubernetes/pull/55261),[ @ncdc](https://github.com/ncdc))
*   Session affinity will now work properly when using an external load balancer along with when ExternalTrafficPolicy=Local. ([#55519](https://github.com/kubernetes/kubernetes/pull/55519),[ @MrHohn](https://github.com/MrHohn))


##### **CoreDNS**



*   You now have the option to use CoreDNS instead of KubeDNS. To install CoreDNS instead of kube-dns, set CLUSTER_DNS_CORE_DNS to 'true'. This support is experimental. ([#52501](https://github.com/kubernetes/kubernetes/pull/52501),[ @rajansandeep](https://github.com/rajansandeep)) ([#55728](https://github.com/kubernetes/kubernetes/pull/55728),[ @rajansandeep](https://github.com/rajansandeep))


##### **Other**



*   Pod addresses will now be removed from the list of endpoints when the pod is in graceful termination. ([#54828](https://github.com/kubernetes/kubernetes/pull/54828),[ @freehan](https://github.com/freehan))
*   You can now use a new supported service annotation for AWS clusters, `service.beta.kubernetes.io/aws-load-balancer-ssl-negotiation-policy`, which lets you specify which [predefined AWS SSL policy](http://docs.aws.amazon.com/elasticloadbalancing/latest/classic/elb-security-policy-table.html) you would like to use. ([#54507](https://github.com/kubernetes/kubernetes/pull/54507),[ @micahhausler](https://github.com/micahhausler))
*   Termination grace period for the calico/node add-on DaemonSet has been eliminated, reducing downtime during a rolling upgrade or deletion. ([#55015](https://github.com/kubernetes/kubernetes/pull/55015),[ @fasaxc](https://github.com/fasaxc))
*   Fixed bad conversion in host port chain name generating func which led to some unreachable host ports. ([#55153](https://github.com/kubernetes/kubernetes/pull/55153),[ @chenchun](https://github.com/chenchun))
*   Fixed IPVS availability check ([#51874](https://github.com/kubernetes/kubernetes/pull/51874),[ @vfreex](https://github.com/vfreex))
*   The output for kubectl describe networkpolicy * has been enhanced to be more useful. ([#46951](https://github.com/kubernetes/kubernetes/pull/46951),[ @aanm](https://github.com/aanm))
*   Kernel modules are now loaded automatically inside a kube-proxy pod ([#52003](https://github.com/kubernetes/kubernetes/pull/52003),[ @vfreex](https://github.com/vfreex))
*   Improve resilience by annotating kube-dns addon with podAntiAffinity to prefer scheduling on different nodes. ([#52193](https://github.com/kubernetes/kubernetes/pull/52193),[ @StevenACoffman](https://github.com/StevenACoffman))
*   [alpha] Added DNSConfig field to PodSpec. "None" mode for DNSPolicy is now supported. ([#55848](https://github.com/kubernetes/kubernetes/pull/55848),[ @MrHohn](https://github.com/MrHohn))
*   You can now add "options" to the host's /etc/resolv.conf (or --resolv-conf), and they will be copied into pod's resolv.conf when dnsPolicy is Default. Being able to customize options is important because it is common to leverage options to fine-tune the behavior of DNS client. ([#54773](https://github.com/kubernetes/kubernetes/pull/54773),[ @phsiao](https://github.com/phsiao))
*   Fixed a bug so that the service controller no longer retries if doNotRetry service update fails. ([#54184](https://github.com/kubernetes/kubernetes/pull/54184),[ @MrHohn](https://github.com/MrHohn))
*   Added --no-negcache flag to kube-dns to prevent caching of NXDOMAIN responses. ([#53604](https://github.com/kubernetes/kubernetes/pull/53604),[ @cblecker](https://github.com/cblecker))


#### **Node**


##### **Pod API**



*   A single value in metadata.annotations/metadata.labels can now be passed into the containers via the Downward API. ([#55902](https://github.com/kubernetes/kubernetes/pull/55902),[ @yguo0905](https://github.com/yguo0905))
*   Pods will no longer briefly transition to a "Pending" state during the deletion process. ([#54593](https://github.com/kubernetes/kubernetes/pull/54593),[ @dashpole](https://github.com/dashpole))
*   Added pod-level local ephemeral storage metric to the Summary API. Pod-level ephemeral storage reports the total filesystem usage for the containers and emptyDir volumes in the measured Pod. ([#55447](https://github.com/kubernetes/kubernetes/pull/55447),[ @jingxu97](https://github.com/jingxu97))


##### **Hardware Accelerators**



*   Kubelet now exposes metrics for NVIDIA GPUs attached to the containers. ([#55188](https://github.com/kubernetes/kubernetes/pull/55188),[ @mindprince](https://github.com/mindprince))
*   The device plugin Alpha API no longer supports returning artifacts per device as part of AllocateResponse. ([#53031](https://github.com/kubernetes/kubernetes/pull/53031),[ @vishh](https://github.com/vishh))
*   Fix to ignore extended resources that are not registered with kubelet during container resource allocation. ([#53547](https://github.com/kubernetes/kubernetes/pull/53547),[ @jiayingz](https://github.com/jiayingz))


##### **Kubelet**



*   The EvictionHard, EvictionSoft, EvictionSoftGracePeriod, EvictionMinimumReclaim, SystemReserved, and KubeReserved fields in the KubeletConfiguration object (`kubeletconfig/v1alpha1`) are now of type map[string]string, which facilitates writing JSON and YAML files. ([#54823](https://github.com/kubernetes/kubernetes/pull/54823),[ @mtaufen](https://github.com/mtaufen))
*   Relative paths in the Kubelet's local config files (`--init-config-dir`) will now be resolved relative to the location of the containing files. ([#55648](https://github.com/kubernetes/kubernetes/pull/55648),[ @mtaufen](https://github.com/mtaufen))
*   It is now possible to set multiple manifest url headers via the Kubelet's `--manifest-url-header` flag. Multiple headers for the same key will be added in the order provided. The ManifestURLHeader field in KubeletConfiguration object (kubeletconfig/v1alpha1) is now a map[string][]string, which facilitates writing JSON and YAML files. ([#54643](https://github.com/kubernetes/kubernetes/pull/54643),[ @mtaufen](https://github.com/mtaufen))
*   The Kubelet's feature gates are now specified as a map when provided via a JSON or YAML KubeletConfiguration, rather than as a string of key-value pairs, making them less awkward for users. ([#53025](https://github.com/kubernetes/kubernetes/pull/53025),[ @mtaufen](https://github.com/mtaufen))
*   CRI now uses the correct localhost seccomp path when provided with input in the format of localhost//profileRoot/profileName. ([#55450](https://github.com/kubernetes/kubernetes/pull/55450),[ @feiskyer](https://github.com/feiskyer))


##### **Other**



*   Fixed a performance issue ([#51899](https://github.com/kubernetes/kubernetes/pull/51899)) identified in large-scale clusters when deleting thousands of pods simultaneously across hundreds of nodes, by actively removing containers of deleted pods, rather than waiting for periodic garbage collection and batching resulting pod API deletion requests. ([#53233](https://github.com/kubernetes/kubernetes/pull/53233),[ @dashpole](https://github.com/dashpole))
*   Problems deleting local static pods have been resolved. ([#48339](https://github.com/kubernetes/kubernetes/pull/48339),[ @dixudx](https://github.com/dixudx))
*   CRI now only calls UpdateContainerResources when cpuset is set. ([#53122](https://github.com/kubernetes/kubernetes/pull/53122),[ @resouer](https://github.com/resouer))
*   Containerd monitoring is now supported. ([#56109](https://github.com/kubernetes/kubernetes/pull/56109),[ @dashpole](https://github.com/dashpole))
*   deviceplugin has been extended to more gracefully handle the full device plugin lifecycle, including: ([#55088](https://github.com/kubernetes/kubernetes/pull/55088),[ @jiayingz](https://github.com/jiayingz))
    *   Kubelet now uses an explicit cm.GetDevicePluginResourceCapacity() function that makes it possible to more accurately determine what resources are inactive and return a more accurate view of available resources.
    *   Extends the device plugin checkpoint data to record registered resources so that we can finish resource removing devices even upon kubelet restarts.
    *   Passes sourcesReady from kubelet to the device plugin to avoid removing inactive pods during the grace period of kubelet restart.
    *   Extends the gpu_device_plugin e2e_node test to verify that scheduled pods can continue to run even after a device plugin deletion and kubelet restart.
*   The NodeController no longer supports kubelet 1.2. ([#48996](https://github.com/kubernetes/kubernetes/pull/48996),[ @k82cn](https://github.com/k82cn))
*   Kubelet now provides more specific events via FailedSync when unable to sync a pod. ([#53857](https://github.com/kubernetes/kubernetes/pull/53857),[ @derekwaynecarr](https://github.com/derekwaynecarr))
*   You can now disable AppArmor by setting the AppArmor profile to unconfined. ([#52395](https://github.com/kubernetes/kubernetes/pull/52395),[ @dixudx](https://github.com/dixudx))
*   ImageGCManage now consumes ImageFS stats from StatsProvider rather than cadvisor. ([#53094](https://github.com/kubernetes/kubernetes/pull/53094),[ @yguo0905](https://github.com/yguo0905))
*   Hyperkube now supports the support --experimental-dockershim kubelet flag. ([#54508](https://github.com/kubernetes/kubernetes/pull/54508),[ @ivan4th](https://github.com/ivan4th))
*   CRI now supports debugging via a verbose option for status functions. ([#53965](https://github.com/kubernetes/kubernetes/pull/53965),[ @Random-Liu](https://github.com/Random-Liu))
*   Kubelet no longer removes default labels from Node API objects on startup ([#54073](https://github.com/kubernetes/kubernetes/pull/54073),[ @liggitt](https://github.com/liggitt))
*   The overlay2 container disk metrics for Docker and CRI-O now work properly. ([#54827](https://github.com/kubernetes/kubernetes/pull/54827),[ @dashpole](https://github.com/dashpole))
*   Removed docker dependency during kubelet start up. ([#54405](https://github.com/kubernetes/kubernetes/pull/54405),[ @resouer](https://github.com/resouer))
*   Added Windows support to the system verification check. ([#53730](https://github.com/kubernetes/kubernetes/pull/53730),[ @bsteciuk](https://github.com/bsteciuk))
*   Kubelet no longer removes unregistered extended resource capacities from node status; cluster admins will have to manually remove extended resources exposed via device plugins when they the remove plugins themselves. ([#53353](https://github.com/kubernetes/kubernetes/pull/53353),[ @jiayingz](https://github.com/jiayingz))
*   The stats summary network value now takes into account multiple network interfaces, and not just eth0. ([#52144](https://github.com/kubernetes/kubernetes/pull/52144),[ @andyxning](https://github.com/andyxning))
*   Kubelet can now provide full summary api support for the CRI container runtime, with the exception of container log stats. ([#55810](https://github.com/kubernetes/kubernetes/pull/55810),[ @abhi](https://github.com/abhi))
*   Base images have been bumped to Debian Stretch (9). ([#52744](https://github.com/kubernetes/kubernetes/pull/52744),[ @rphillips](https://github.com/rphillips))


#### **OpenStack**



*   OpenStack Cinder support has been improved:
    *   Cinder version detection now works properly. ([#53115](https://github.com/kubernetes/kubernetes/pull/53115),[ @FengyunPan](https://github.com/FengyunPan))
    *   The OpenStack cloud provider now supports Cinder v3 API. ([#52910](https://github.com/kubernetes/kubernetes/pull/52910),[ @FengyunPan](https://github.com/FengyunPan))
*   Load balancing is now more flexible:
    *   The OpenStack LBaaS v2 Provider is now [configurable](https://kubernetes.io/docs/concepts/cluster-administration/cloud-providers/#openstack). ([#54176](https://github.com/kubernetes/kubernetes/pull/54176),[ @gonzolino](https://github.com/gonzolino))
    *   OpenStack Octavia v2 is now supported as a load balancer provider in addition to the existing support for the Neutron LBaaS V2 implementation. Neutron LBaaS V1 support has been removed. ([#55393](https://github.com/kubernetes/kubernetes/pull/55393),[ @jamiehannaford](https://github.com/jamiehannaford))
*   OpenStack security group support has been beefed up  ([#50836](https://github.com/kubernetes/kubernetes/pull/50836),[ @FengyunPan](https://github.com/FengyunPan)): 
    *   Kubernetes will now automatically determine the security group for the node
    *   Nodes can now belong to multiple security groups


#### **Scheduling**


##### **Hardware Accelerators**



*   Add ExtendedResourceToleration admission controller. This facilitates creation of dedicated nodes with extended resources. If operators want to create dedicated nodes with extended resources (such as GPUs, FPGAs, and so on), they are expected to taint the node with extended resource name as the key. This admission controller, if enabled, automatically adds tolerations for such taints to pods requesting extended resources, so users don't have to manually add these tolerations. ([#55839](https://github.com/kubernetes/kubernetes/pull/55839),[ @mindprince](https://github.com/mindprince))


##### **Other**



*   Scheduler cache ignores updates to an assumed pod if updates are limited to pod annotations.  ([#54008](https://github.com/kubernetes/kubernetes/pull/54008),[ @yguo0905](https://github.com/yguo0905))
*   Issues with namespace deletion have been resolved. ([#53720](https://github.com/kubernetes/kubernetes/pull/53720),[ @shyamjvs](https://github.com/shyamjvs)) ([#53793](https://github.com/kubernetes/kubernetes/pull/53793),[ @wojtek-t](https://github.com/wojtek-t))
*   Pod preemption has been improved.  
    *   Now takes PodDisruptionBudget into account. ([#56178](https://github.com/kubernetes/kubernetes/pull/56178),[ @bsalamat](https://github.com/bsalamat))
    *   Nominated pods are taken into account during scheduling to avoid starvation of higher priority pods. ([#55933](https://github.com/kubernetes/kubernetes/pull/55933),[ @bsalamat](https://github.com/bsalamat))
*   Fixed 'Schedulercache is corrupted' error in kube-scheduler ([#55262](https://github.com/kubernetes/kubernetes/pull/55262),[ @liggitt](https://github.com/liggitt))
*   The kube-scheduler command now supports a --config flag which is the location of a file containing a serialized scheduler configuration. Most other kube-scheduler flags are now deprecated. ([#52562](https://github.com/kubernetes/kubernetes/pull/52562),[ @ironcladlou](https://github.com/ironcladlou))
*   A new scheduling queue helps schedule the highest priority pending pod first. ([#55109](https://github.com/kubernetes/kubernetes/pull/55109),[ @bsalamat](https://github.com/bsalamat))
*   A Pod can now listen to the same port on multiple IP addresses.  ([#52421](https://github.com/kubernetes/kubernetes/pull/52421),[ @WIZARD-CXY](https://github.com/WIZARD-CXY))
*   Object count quotas supported on all standard resources using count/<resource>.<group> syntax ([#54320](https://github.com/kubernetes/kubernetes/pull/54320),[ @derekwaynecarr](https://github.com/derekwaynecarr))
*   Apply algorithm in scheduler by feature gates. ([#52723](https://github.com/kubernetes/kubernetes/pull/52723),[ @k82cn](https://github.com/k82cn))
*   A new priority function ResourceLimitsPriorityMap (disabled by default and behind alpha feature gate and not part of the scheduler's default priority functions list) that assigns a lowest possible score of 1 to a node that satisfies one or both of input pod's cpu and memory limits, mainly to break ties between nodes with same scores. ([#55906](https://github.com/kubernetes/kubernetes/pull/55906),[ @aveshagarwal](https://github.com/aveshagarwal))
*   Kubelet evictions now take pod priority into account ([#53542](https://github.com/kubernetes/kubernetes/pull/53542),[ @dashpole](https://github.com/dashpole))
*   PodTolerationRestriction admisson plugin: if namespace level tolerations are empty, now they override cluster level tolerations. ([#54812](https://github.com/kubernetes/kubernetes/pull/54812),[ @aveshagarwal](https://github.com/aveshagarwal))


#### **Storage**



*   [stable] `PersistentVolume` and `PersistentVolumeClaim` objects must now have a capacity greater than zero.
*   [stable] Mutation of `PersistentVolumeSource` after creation is no longer allowed
*   [alpha] Deletion of `PersistentVolumeClaim` objects that are in use by a pod no longer permitted (if alpha feature is enabled).
*   [alpha] Container Storage Interface
    *   New CSIVolumeSource enables Kubernetes to use external CSI drivers to provision, attach, and mount volumes.
*   [alpha] Raw block volumes 
    *   Support for surfacing volumes as raw block devices added to Kubernetes storage system.
    *   Only Fibre Channel volume plugin supports exposes this functionality, in this release.
*   [alpha] Volume resizing
    *   Added file system resizing for the following volume plugins: GCE PD, Ceph RBD, AWS EBS, OpenStack Cinder
*   [alpha] Topology Aware Volume Scheduling 
    *   Improved volume scheduling for Local PersistentVolumes, by allowing the scheduler to make PersistentVolume binding decisions while respecting the Pod's scheduling requirements.
    *   Dynamic provisioning is not supported with this feature yet.
*   [alpha] Containerized mount utilities
    *   Allow mount utilities, used to mount volumes, to run inside a container instead of on the host.
*   Bug Fixes
    *   ScaleIO volume plugin is no longer dependent on the drv_cfg binary, so a Kubernetes cluster can easily run a containerized kubelet. ([#54956](https://github.com/kubernetes/kubernetes/pull/54956),[ @vladimirvivien](https://github.com/vladimirvivien))
    *   AWS EBS Volumes are detached from stopped AWS nodes. ([#55893](https://github.com/kubernetes/kubernetes/pull/55893),[ @gnufied](https://github.com/gnufied))
    *   AWS EBS volumes are detached if attached to a different node than expected. ([#55491](https://github.com/kubernetes/kubernetes/pull/55491),[ @gnufied](https://github.com/gnufied))
    *   PV Recycle now works in environments that use architectures other than x86. ([#53958](https://github.com/kubernetes/kubernetes/pull/53958),[ @dixudx](https://github.com/dixudx))
    *   Pod Security Policy can now manage access to specific FlexVolume drivers.([#53179](https://github.com/kubernetes/kubernetes/pull/53179),[ @wanghaoran1988](https://github.com/wanghaoran1988))
    *   To prevent unauthorized access to CHAP Secrets, you can now set the secretNamespace storage class parameters for the following volume types:
        *   ScaleIO; StoragePool and ProtectionDomain attributes no longer default to the value default.  ([#54013](https://github.com/kubernetes/kubernetes/pull/54013),[ @vladimirvivien](https://github.com/vladimirvivien))
        *   RBD Persistent Volume Sources ([#54302](https://github.com/kubernetes/kubernetes/pull/54302),[ @sbezverk](https://github.com/sbezverk))
        *   iSCSI Persistent Volume Sources ([#51530](https://github.com/kubernetes/kubernetes/pull/51530),[ @rootfs](https://github.com/rootfs))
    *   In GCE multizonal clusters, `PersistentVolume` objects will no longer be dynamically provisioned in zones without nodes. ([#52322](https://github.com/kubernetes/kubernetes/pull/52322),[ @davidz627](https://github.com/davidz627))
    *   Multi Attach PVC errors and events are now more useful and less noisy. ([#53401](https://github.com/kubernetes/kubernetes/pull/53401),[ @gnufied](https://github.com/gnufied))
    *   The compute-rw scope has been removed from GCE nodes ([#53266](https://github.com/kubernetes/kubernetes/pull/53266),[ @mikedanese](https://github.com/mikedanese))
    *   Updated vSphere cloud provider to support k8s cluster spread across multiple vCenters ([#55845](https://github.com/kubernetes/kubernetes/pull/55845),[ @rohitjogvmw](https://github.com/rohitjogvmw))
    *   vSphere: Fix disk is not getting detached when PV is provisioned on clustered datastore. ([#54438](https://github.com/kubernetes/kubernetes/pull/54438),[ @pshahzeb](https://github.com/pshahzeb))
    *   If a non-absolute mountPath is passed to the kubelet, it must now be prefixed with the appropriate root path. ([#55665](https://github.com/kubernetes/kubernetes/pull/55665),[ @brendandburns](https://github.com/brendandburns))


## External Dependencies



*   The supported etcd server version is **3.1.10**, as compared to 3.0.17 in v1.8 ([#49393](https://github.com/kubernetes/kubernetes/pull/49393),[ @hongchaodeng](https://github.com/hongchaodeng))
*   The validated docker versions are the same as for v1.8: **1.11.2 to 1.13.1 and 17.03.x**
*   The Go version was upgraded from go1.8.3 to **go1.9.2** ([#51375](https://github.com/kubernetes/kubernetes/pull/51375),[ @cblecker](https://github.com/cblecker))
    *   The minimum supported go version bumps to 1.9.1. ([#55301](https://github.com/kubernetes/kubernetes/pull/55301),[ @xiangpengzhao](https://github.com/xiangpengzhao))
    *   Kubernetes has been upgraded to go1.9.2 ([#55420](https://github.com/kubernetes/kubernetes/pull/55420),[ @cblecker](https://github.com/cblecker))
*   CNI was upgraded to **v0.6.0** ([#51250](https://github.com/kubernetes/kubernetes/pull/51250),[ @dixudx](https://github.com/dixudx))
*   The dashboard add-on has been updated to [v1.8.0](https://github.com/kubernetes/dashboard/releases/tag/v1.8.0). ([#53046](https://github.com/kubernetes/kubernetes/pull/53046), [@maciaszczykm](https://github.com/maciaszczykm))
*   Heapster has been updated to [v1.5.0](https://github.com/kubernetes/heapster/releases/tag/v1.5.0). ([#57046](https://github.com/kubernetes/kubernetes/pull/57046), [@piosz](https://github.com/piosz))
*   Cluster Autoscaler has been updated to [v1.1.0](https://github.com/kubernetes/autoscaler/releases/tag/cluster-autoscaler-1.1.0). ([#56969](https://github.com/kubernetes/kubernetes/pull/56969), [@mwielgus](https://github.com/mwielgus))
*   Update kube-dns 1.14.7 ([#54443](https://github.com/kubernetes/kubernetes/pull/54443),[ @bowei](https://github.com/bowei))
*   Update influxdb to v1.3.3 and grafana to v4.4.3 ([#53319](https://github.com/kubernetes/kubernetes/pull/53319),[ @kairen](https://github.com/kairen))