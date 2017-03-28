## WARNING: etcd backup strongly recommended

Before updating to 1.6, you are strongly recommended to back up your etcd data.
Please consult the installation procedure you are using (kargo, kops, kube-up,
kube-aws, kubeadm etc) for specific advice.

1.6 encourages etcd3, and switching from etcd2 to etcd3 involves a full
migration of data between different storage engines.  You must stop the API
from writing to etcd during an etcd2 -> etcd3 migration.  HA installations cannot
be migrated at the current time using the official kubernetes procedure.

1.6 will also default to protobuf encoding if using etcd3.  **This change is
irreversible.**  To rollback, you must restore from a backup made before the
protobuf/etcd3 switch, and any changes since the backup will be lost.  As 1.5
does not support protobuf encoding, if you roll back to 1.5 after upgrading to
protobuf you will be forced to restore from backup, and you will lose any changes
since you converted to protobuf.  After conversion to protobuf, you should
validate the correct operation of your cluster thoroughly before returning it
to normal operation.

Backups are always a good precaution, particularly around upgrades, but this
upgrade has multiple known issues where the only remedy is to restore from
backup.

Also, please note:
- using `application/vnd.kubernetes.protobuf` as the media storage type for 1.6 is default but not required
- the ability to rollback to etcd2 can be preserved by continuing to use `application/json` as the storage media type. This can be changed to `application/vnd.kubernetes.protobuf` at a later time.

## Major updates and release themes

* Kubernetes now supports up to 5,000 nodes via etcd v3, which is enabled by default.
* [Role-based access control (RBAC)](https://kubernetes.io//docs/admin/authorization/rbac) has graduated to beta, and defines secure default roles for control plane, node, and controller components.
* The [`kubeadm` cluster bootstrap tool](https://kubernetes.io/docs/getting-started-guides/kubeadm/) has graduated to beta. Some highlights:
  * All communication is now over TLS
  * Authorization plugins can be installed by kubeadm, including the new default of RBAC
  * The bootstrap token system now allows token management and expiration
* The [`kubefed` federation bootstrap tool](https://kubernetes.io/docs/tutorials/federation/set-up-cluster-federation-kubefed/) has also graduated to beta.
* Interaction with container runtimes is now through the CRI interface, enabling easier integration of runtimes with the kubelet. Docker remains the default runtime via Docker-CRI (which moves to beta).
* Various scheduling features have graduated to beta:
  * You can now use [multiple schedulers](https://kubernetes.io/docs/admin/multiple-schedulers/)
  * [Nodes](https://kubernetes.io/docs/user-guide/node-selection/#node-affinity-beta-feature) and [pods](https://kubernetes.io/docs/user-guide/node-selection/#inter-pod-affinity-and-anti-affinity-beta-feature) now support affinity and anti-affinity
  * Advanced scheduling can be performed with [taints and tolerations](https://kubernetes.io/docs/user-guide/node-selection/#taints-and-tolerations-beta-feature)
* You can now specify (per pod) how long a pod should stay bound to a node, when there is a node problem. 
* Various storage features have graduated to GA:
  * StorageClass pre-installed and set as default on Azure, AWS, GCE, OpenStack, and vSphere
  * Configurable [Dynamic Provisioning](https://kubernetes.io/docs/user-guide/persistent-volumes/#dynamic) and [StorageClass](https://kubernetes.io/docs/user-guide/persistent-volumes/#storageclasses)
* DaemonSets [can now be updated by a rolling update](https://kubernetes.io/docs/tasks/manage-daemon/update-daemon-set).

## Action Required

### Certificates API
* Users of the alpha certificates API should delete v1alpha1 CSRs from the API before upgrading and recreate them as v1beta1 CSR after upgrading. ([#39772](https://github.com/kubernetes/kubernetes/pull/39772), [@mikedanese](https://github.com/mikedanese))

### Cluster Autoscaler
* If you are using (or planning to use) Cluster Autoscaler please wait for Kubernetes 1.6.1. In 1.6.0 Cluster Autoscaler 
may ocassionally increase the size of the cluster a bit more than it is actually needed, when there are 
unschedulable pods, scale up is required and cloud provider is slow to set up networking for new nodes. 
Anyway, the cluster should get back to the proper size after 10 min.

### Deployment
* Deployment now fully respects ControllerRef to avoid fighting over Pods and ReplicaSets. At the time of upgrade, **you must not have Deployments with selectors that overlap**, or else [ownership of ReplicaSets may change](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/controller-ref.md#upgrading). ([#42175](https://github.com/kubernetes/kubernetes/pull/42175), [@enisoc](https://github.com/enisoc))

### Federation
* The --dns-provider argument of 'kubefed init' is now mandatory and does not default to `google-clouddns`. To initialize a Federation control plane with Google Cloud DNS, use the following invocation: 'kubefed init --dns-provider=google-clouddns' ([#42092](https://github.com/kubernetes/kubernetes/pull/42092), [@marun](https://github.com/marun))
* Cluster federation servers have changed the location in etcd where federated services are stored, so existing federated services must be deleted and recreated. Before upgrading, export all federated services from the federation server and delete the services. After upgrading the cluster, recreate the federated services from the exported data. ([#37770](https://github.com/kubernetes/kubernetes/pull/37770), [@enj](https://github.com/enj))

### Internal Storage Layer
* upgrade to etcd3 prior to upgrading to 1.6 **OR** explicitly specify `--storage-type=etcd2 --storage-media-type=application/json` when starting the apiserver

### kubelet
* Hard eviction thresholds will be subtracted from the Allocatable capacity on nodes to improve node reliability. This *may* break existing clusters since the overall schedulable capacity would reduce after upgrading to v1.6. You can opt-out of this feature by specifying `--experimental-allocatable-ignore-eviction=true`.
* Fluentd was migrated to Daemon Set, which targets nodes with beta.kubernetes.io/fluentd-ds-ready=true label. If you use fluentd in your cluster please make sure that the nodes with version 1.6+ contains this label. ([#42931](https://github.com/kubernetes/kubernetes/pull/42931), [@piosz](https://github.com/piosz))

### kubectl
* Running `kubectl taint` (alpha in 1.5) against a 1.6 server requires upgrading kubectl to version 1.6
* Running `kubectl taint` (alpha in 1.5) against a 1.5 server requires a kubectl version of 1.5
* Running `kubectl create secret` no longer accepts passing multiple values to a single --from-literal flag using comma separation
  * Update command invocations to pass separate --from-literal flags for each value

### RBAC
* Default ClusterRole and ClusterRoleBinding objects are automatically updated at server start to add missing permissions and subjects (extra permissions and subjects are left in place). To prevent autoupdating a particular role or rolebinding, annotate it with `rbac.authorization.kubernetes.io/autoupdate=false`. ([#41155](https://github.com/kubernetes/kubernetes/pull/41155), [@liggitt](https://github.com/liggitt))
* `v1beta1` RoleBinding/ClusterRoleBinding subjects changed `apiVersion` to `apiGroup` to fully-qualify a subject. ServiceAccount subjects default to an apiGroup of `""`, User and Group subjects default to an apiGroup of `"rbac.authorization.k8s.io"`. ([#41184](https://github.com/kubernetes/kubernetes/pull/41184), [@liggitt](https://github.com/liggitt))
* To create or update an RBAC RoleBinding or ClusterRoleBinding object, a user must: ([#39383](https://github.com/kubernetes/kubernetes/pull/39383), [@liggitt](https://github.com/liggitt))
    * 1. Be authorized to make the create or update API request
    * 2. Be allowed to bind the referenced role, either by already having all of the permissions contained in the referenced role, or by having the "bind" permission on the referenced role.
* The `--authorization-rbac-super-user` flag (alpha in 1.5) is deprecated; the `system:masters` group has privileged access ([#38121](https://github.com/kubernetes/kubernetes/pull/38121), [@deads2k](https://github.com/deads2k))
* special handling of the user `*` in RoleBinding and ClusterRoleBinding objects is removed in v1beta1. To match all users, explicitly bind to the group `system:authenticated` and/or `system:unauthenticated`. Existing v1alpha1 bindings to the user `*` are automatically converted to the group `system:authenticated`. ([#38981](https://github.com/kubernetes/kubernetes/pull/38981), [@liggitt](https://github.com/liggitt))

### Scheduling
* **Multiple schedulers**
  * Modify your PodSpecs that currently use the `scheduler.alpha.kubernetes.io/name` annotation on Pod, to instead use the `schedulerName` field in the PodSpec.
  * Modify any custom scheduler(s) you have written so that they read the `schedulerName` field on Pod instead of the `scheduler.alpha.kubernetes.io/name` annotation. 
  * Note that you can only start using the `schedulerName` field **after** you upgrade to 1.6; it is not recognized in 1.5.

* **Node affinity/anti-affinity and pod affinity/anti-affinity**
  * You can continue to use the alpha version of this feature (with one caveat -- see below), in which you specify affinity requests using Pod annotations, in 1.6 by including `AffinityInAnnotations=true` in `--feature-gates`, such as `--feature-gates=FooBar=true,AffinityInAnnotations=true`. Otherwise, you must modify your PodSpecs that currently use the `scheduler.alpha.kubernetes.io/affinity` annotation on Pod, to instead use the `affinity` field in the PodSpec. Support for the annotation will be removed in a future release, so we encourage you to switch to using the field as soon as possible.
  * Caveat: The alpha version no longer supports, and the beta version does not support, the "empty `podAffinityTerm.namespaces` list means all namespaces" behavior. In both alpha and beta it now means "same namespace as the pod specifying this affinity rule."
  * Note that you can only start using the `affinity` field **after** you upgrade to 1.6; it is not recognized in 1.5.
  * The `--failure-domains` scheduler command line-argument is not supported in the beta vesion of the feature.

* **Taints**
  * You will need to use `kubectl taint` to re-create all of your taints after kubectl and the master are upgraded to 1.6. Between the time the master is upgraded to 1.6 and when you do this, your existing taints will have no effect. 
  * You can find out what taints you have in place on a node while you are still running Kubernetes 1.5 by doing `kubectl describe node <node name>`; the `Taints` section will show the taints you have in place. To see the taints that were created under 1.5 when you are running 1.6, do `kubectl get node <node name> -o yaml` and look for the "Annotation" section with the annotation key `scheduler.alpha.kubernetes.io/taints`
  * You can remove the "old" taints (stored internally as annotations) at any time after the upgrade by doing `kubectl annotate nodes <node name> scheduler.alpha.kubernetes.io/taints-` (note the minus at the end, which means "delete the taint with this key")
  * Note that because any taints you might have created with Kubernetes 1.5 can only affect the scheduling of new pods (the `NoExecute` taint effect is introduced in 1.6), neither the master upgrade nor your running `kubectl taint` to re-create the taints will affect pods that are already running.
  * Rescheduler relies on taints, which were changed in a backward incompatible way. Rescheduler 0.3 shipped together with Kubernetes 1.6 understands the new taints and will clean up the old annotations, but Rescheduler 0.2 shipped together with Kubernetes 1.5 doesn't understand the new taints. In order to avoid eviction loop during 1.5->1.6 upgrade you need to either upgrade the master node atomically (for example by using `upgrade.sh` script for GCE) or ensure that the rescheduler is upgraded first, then the scheduler and then all the other master components.

* **Tolerations**
  * After your master is upgraded to 1.6, you will need to update your PodSpecs to set the `tolerations` field of the PodSpec and remove the `scheduler.alpha.kubernetes.io/tolerations` annotation on the Pod. (It's not strictly necessary to remove the annotation, as it will have no effect once you upgrade to 1.6.) Between the time the master is upgraded to 1.6 and when you do this, tolerations attached to Pods that are created will have no effect. Pods that are already running will not be affected by the upgrade.
  * You can find the PodSpec tolerations that were created as annotations (if any) in a controller definition by doing `kubectl get <controller kind> <controller name> -o yaml` and looking for the "Annotation" section with the annotation key `scheduler.alpha.kubernetes.io/tolerations` (This will work whether you are running Kubernetes 1.5 or 1.6).
  * To update a controller's PodSpec to use the field instead of the annotation, use one of the kubectl commands that do update ("kubectl replace" or "kubectl apply" or "kubectl patch") or just delete the controller entirely and re-create it with a new pod template. Note that you will be able to do these things only after the upgrade.

### Service
* The 'endpoints.beta.kubernetes.io/hostnames-map' annotation is no longer supported.  Users can use the 'Endpoints.subsets[].addresses[].hostname' field instead. ([#39284](https://github.com/kubernetes/kubernetes/pull/39284), [@bowei](https://github.com/bowei))

### StatefulSet
* StatefulSet now respects ControllerRef to avoid fighting over Pods. At the time of upgrade, **you must not have StatefulSets with selectors that overlap** with any other controllers (such as ReplicaSets), or else [ownership of Pods may change](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/controller-ref.md#upgrading). ([#42080](https://github.com/kubernetes/kubernetes/pull/42080), [@enisoc](https://github.com/enisoc))

### Volume
* StorageClass pre-installed and set as default on Azure, AWS, GCE, OpenStack, and vSphere.
  * This is something to pay close attention to if you’ve been using Kubernetes for a while, because it changes the default behavior of PersistentVolumeClaim objects on these clouds.
  * Marking a StorageClass as default makes it so that even a PersistentVolumeClaim without a StorageClass specified will trigger dynamic provisioning (instead of binding to an existing pool of PVs).
  * If you depend on the old behavior of volumes binding to existing pool of PersistentVolume objects then modify the StorageClass object and set `storageclass.beta.kubernetes.io/is-default-class` to `false`.
* Flex volume plugin is updated to support attach/detach interfaces. It broke backward compatibility. Please update your drivers and implement the new callouts.  ([#41804](https://github.com/kubernetes/kubernetes/pull/41804), [@chakri-nelluri](https://github.com/chakri-nelluri))

## Notable Features
Features for this release were tracked via the use of the [kubernetes/features](https://github.com/kubernetes/features) issues repo.  Each Feature issue is owned by a Special Interest Group from the [kubernetes/community](https://github.com/kubernetes/community/blob/master/sig-list.md).

### Autoscaling
* **[alpha]** The Horizontal Pod Autoscaler now supports drawing metrics throug the API server aggregator.
* **[alpha]** The Horizontal Pod Autoscaler now supports scaling on multiple, custom metrics.
* Cluster Autoscaler publishes its status to kube-system/cluster-autoscaler-status ConfigMap.
* Cluster Autoscaler can continue operations while some nodes are broken or unready.
* Cluster Autoscaler respects Pod Disruption Budgets when removing a node.

### DaemonSets
* **[beta]** Introduce the rolling update feature for DaemonSet. See [Performing a Rolling Update on a DaemonSet](https://deploy-preview-2878--kubernetes-io-master-staging.netlify.com/docs/tasks/manage-daemon/update-daemon-set/).

### Deployments
* **[beta]** Deployments that cannot make progress in rolling out the newest version will now indicate via the API they are blocked ([docs](https://kubernetes.io/docs/user-guide/deployments/#deployment-status))

### Federation
* **[beta]** `kubefed` has graduated to beta: supports hosting federation on on-prem clusters, automatically configures `kube-dns` in joining clusters and allows passing arguments to federation components.

### Internal Storage Layer
* **[stable]** The internal storage layer for kubernetes cluster state has been updated to use etcd v3 by default for new clusters.  Old clusters will have to plan for a data migration window. ([docs](https://github.com/kubernetes/kubernetes.github.io/pull/2763))([kubernetes/features#44](https://github.com/kubernetes/features/issues/44))

### kubeadm
* **[beta]** Introduces an API for clients to request TLS certificates from the API server. See the [tutorial](https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster).
* **[beta]** kubeadm is enhanced and improved with a baseline feature set and command line flags that are now marked as beta. Other parts of kubeadm, including subcommands under `kubeadm alpha`, are [still in alpha](https://kubernetes.io/docs/admin/kubeadm/).  Using it is considered safe, although note that upgrades and HA are not yet supported. Please [try it out](https://kubernetes.io/docs/getting-started-guides/kubeadm/) and [give us feedback](https://kubernetes.io/docs/getting-started-guides/kubeadm/#feedback)!
* **[alpha]** New Bootstrap Token authentication and management method. Works well with kubeadm. kubeadm now supports managing tokens, including time based expiration, after the cluster is launched.  See [kubeadm reference docs](https://kubernetes.io/docs/admin/kubeadm/#manage-tokens) for details.
* **[alpha]** Adds a new cloud-controller-manager binary that may be used for testing the new out-of-core cloudprovider flow.

### kubelet
* **[stable]** kubelet launches pods in a new cgroup hierarchy to better enforce quality of service.  Operators must drain all pods from their nodes prior to upgrade of the kubelet.  They must ensure the configured cgroup driver matches their associated container runtime cgroup driver.

### RBAC
* **[beta]** RBAC API is promoted to v1beta1 (rbac.authorization.k8s.io/v1beta1), and defines default roles for control plane, node, and controller components.
* **[beta]** The Docker-CRI implementation is Beta and is enabled by default in kubelet.  You can disable it by `--enable-cri=false`. See [notes on the new implementation]( https://github.com/kubernetes/community/blob/master/contributors/devel/container-runtime-interface.md#kubernetes-v16-release-docker-cri-integration-beta) for more details.

### Scheduling
- **[beta]** The [multiple schedulers](https://kubernetes.io/docs/admin/multiple-schedulers/). This feature allows you to run multiple schedulers in parallel, each responsible for different sets of pods. When using multiple schedulers, the scheduler name is now specified in a new-in-1.6 `schedulerName` field of the PodSpec rather than using the `scheduler.alpha.kubernetes.io/name` annotation on the Pod. When you upgrade to 1.6, the Kubernetes default scheduler will start using the `schedulerName` field of the PodSpec and will ignore the annotation. 
- **[beta]** [Node affinity/anti-affinity](https://kubernetes.io/docs/user-guide/node-selection/) and **[beta]** [pod affinity/anti-affinity](https://kubernetes.io/docs/user-guide/node-selection/). Node affinity/anti-affinity allow you to specify rules for restricting which node(s) a pod can schedule onto, based on the labels on the node. Pod affinity/anti-affinity allow you to specify rules for spreading and packing pods relative to one another, across arbitrary topologies (node, zone, etc.) These affinity rules are now be specified in a new-in-1.6 `affinity` field of the PodSpec. Kubernetes 1.6 continues to support the alpha `scheduler.alpha.kubernetes.io/affinity` annotation on the Pod if you explicitly enable the alpha feature "AffinityInAnnotations", but it will be removed in a future release. When you upgrade to 1.6, if you have not enabled the alpha feature, then the scheduler will use the `affinity` field in PodSpec and will ignore the annotation. If you have enabled the alpha feature, then the scheduler will use the `affinity` field in PodSpec if it is present, and otherwise will use the annotation.
- **[beta]** [Taints and tolerations](https://kubernetes.io/docs/user-guide/node-selection/). This feature allows you to specify rules for "repelling" pods from nodes by default, which enables use cases like dedicated nodes and reserving nodes with special features for pods that need those features. We've also added a `NoExecute` taint type that evicts already-running pods, and an associated `tolerationSeconds` field to tolerations to delay the eviction for a specified amount of time. As before, taints are created using `kubectl taint` (but internally they are now represented as a field `taints` in the NodeSpec rather than using the `scheduler.alpha.kubernetes.io/taints` annotation on Node). Tolerations are now specified in a new-in-1.6 `tolerations` field of the PodSpec rather than using the `scheduler.alpha.kubernetes.io/tolerations` annotation on the Pod. When you upgrade to 1.6, the scheduler will start using the fields and will ignore the annotations.
- **[alpha]** Represent node problems "not ready" and "unreachable" using `NoExecute` taints. In combination with `tolerationSeconds` described below, this allows per-pod specification of how long to remain bound to a node that becomes unreachable or not ready, rather than using the default of 5 minutes. You can enable this alpha feature by including `TaintBasedEvictions=true` in `--feature-gates`, such as `--feature-gates=FooBar=true,TaintBasedEvictions=true`. Documentation is [here](https://kubernetes.io/docs/user-guide/node-selection/).

### Service Catalog
- **[alpha]** Adds a new API resource `PodPreset` and admission controller to enable defining cross-cutting injection of Volumes and Environment into Pods.

### Volumes
* **[stable]** StorageClass API is promoted to v1 (storage.k8s.io/v1).
* **[stable]** Default storage classes are deployed during installation on Azure, AWS, GCE, OpenStack and vSphere.
* **[stable]** Added ability to populate environment variables from a configmap or secret.
* **[stable]** Support for user-written/run dynamic PV provisioners. The Kubernetes Incubator contains [a golang library and examples](https://github.com/kubernetes-incubator/external-storage).
* **[stable]** Volume plugin for ScaleIO enabling pods to seamlessly access and use data stored on Dell EMC ScaleIO volumes.
* **[stable]** Volume plugin for Portworx added capability to use [Portworx](http://www.portworx.com) as a storage provider for Kubernetes clusters. Portworx pools server capacity and turns servers or cloud instances into converged, highly available compute and storage nodes.
* **[stable]** Add support to use NFSv3, NFSv4, and GlusterFS on GCE/GKE GCI image based clusters.
* **[beta]** Added support for mount options in persistent volumes.
* **[alpha]** All in one volume proposal - a new volume driver capable of projecting secrets, configmaps, and downward API items into the same directory.
* **[alpha]** Flex volume API and Improved lifecycle (flexvolume).

## Deprecations
* Remove extensions/v1beta1 Jobs resource, and job/v1beta1 generator. ([#38614](https://github.com/kubernetes/kubernetes/pull/38614), [@soltysh](https://github.com/soltysh))
* `federation/deploy/deploy.sh` was an interim solution introduced in Kubernetes v1.4 to simplify the federation control plane deployment experience. Now that we have `kubefed`, we are deprecating `deploy.sh` scripts. ([#38902](https://github.com/kubernetes/kubernetes/pull/38902), [@madhusudancs](https://github.com/madhusudancs))


### Cluster Provisioning Scripts
* The bash AWS deployment via kube-up.sh has been deprecated. See http://kubernetes.io/docs/getting-started-guides/aws/ for alternatives. ([#38772](https://github.com/kubernetes/kubernetes/pull/38772), [@zmerlynn](https://github.com/zmerlynn))
* Remove Azure kube-up as the Azure community has focused efforts elsewhere. ([#41672](https://github.com/kubernetes/kubernetes/pull/41672), [@mikedanese](https://github.com/mikedanese))
* Remove the deprecated vsphere kube-up. ([#39140](https://github.com/kubernetes/kubernetes/pull/39140), [@kerneltime](https://github.com/kerneltime))

### Other Deprecations
* Remove cmd/kube-discovery from the tree since it's not necessary anymore ([#42070](https://github.com/kubernetes/kubernetes/pull/42070), [@luxas](https://github.com/luxas))

#### kubeadm
* Quite a few flags been renamed or removed.  Those options that are removed as flags can still be accessed via the config file.  Most noteably this includes external etcd settings and the option for setting the cloud provider on the API server.  The [kubeadm reference documentation](https://kubernetes.io/docs/admin/kubeadm/) is up to date with the new flags.

## Changes to API Resources
### ABAC
* ABAC policies using `"user":"*"` or `"group":"*"` to match all users or groups will only match authenticated requests. To match unauthenticated requests, ABAC policies must explicitly specify `"group":"system:unauthenticated"` ([#38968](https://github.com/kubernetes/kubernetes/pull/38968), [@liggitt](https://github.com/liggitt))

### Admission Control
* Adds a new API resource `PodPreset` and admission controller to enable defining cross-cutting injection of Volumes and Environment into Pods. ([#41931](https://github.com/kubernetes/kubernetes/pull/41931), [@jessfraz](https://github.com/jessfraz))
* Enable DefaultTolerationSeconds admission controller by default ([#41815](https://github.com/kubernetes/kubernetes/pull/41815), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))
* ResourceQuota ability to support default limited resources ([#36765](https://github.com/kubernetes/kubernetes/pull/36765), [@derekwaynecarr](https://github.com/derekwaynecarr))
* Add defaultTolerationSeconds admission controller ([#41414](https://github.com/kubernetes/kubernetes/pull/41414), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))
* An `automountServiceAccountToken` field was added to ServiceAccount and PodSpec objects. If set to `false` on a pod spec, no service account token is automounted in the pod. If set to `false` on a service account, no service account token is automounted for that service account unless explicitly overridden in the pod spec. ([#37953](https://github.com/kubernetes/kubernetes/pull/37953), [@liggitt](https://github.com/liggitt))
* Admission control support for versioned configuration files ([#39109](https://github.com/kubernetes/kubernetes/pull/39109), [@derekwaynecarr](https://github.com/derekwaynecarr))
* Ability to quota storage by storage class ([#34554](https://github.com/kubernetes/kubernetes/pull/34554), [@derekwaynecarr](https://github.com/derekwaynecarr))

### Authentication
* The authentication.k8s.io API group was promoted to v1([#41058](https://github.com/kubernetes/kubernetes/pull/41058), [@liggitt](https://github.com/liggitt))

### Authorization
* The authorization.k8s.io API group was promoted to v1 ([#40709](https://github.com/kubernetes/kubernetes/pull/40709), [@liggitt](https://github.com/liggitt))
* The SubjectAccessReview API now passes subresource and resource name information to the authorizer to answer authorization queries. ([#40935](https://github.com/kubernetes/kubernetes/pull/40935), [@liggitt](https://github.com/liggitt))

### Autoscaling
* Introduces an new alpha version of the Horizontal Pod Autoscaler including expanded support for specifying metrics. ([#36033](https://github.com/kubernetes/kubernetes/pull/36033), [@DirectXMan12](https://github.com/DirectXMan12))
* HorizontalPodAutoscaler is no longer supported in extensions/v1beta1 version. Use autoscaling/v1 instead. ([#35782](https://github.com/kubernetes/kubernetes/pull/35782), [@piosz](https://github.com/piosz))
* Fixes an HPA-related panic due to division-by-zero. ([#39694](https://github.com/kubernetes/kubernetes/pull/39694), [@DirectXMan12](https://github.com/DirectXMan12))

### Certificates
* The CertificateSigningRequest API added the `extra` field to persist all information about the requesting user. This mirrors the fields in the SubjectAccessReview API used to check authorization. ([#41755](https://github.com/kubernetes/kubernetes/pull/41755), [@liggitt](https://github.com/liggitt))
* Native support for token based bootstrap flow.  This includes signing a well known ConfigMap in the `kube-public` namespace and cleaning out expired tokens. ([#36101](https://github.com/kubernetes/kubernetes/pull/36101), [@jbeda](https://github.com/jbeda))

### Config Map
* Volumes and environment variables populated from ConfigMap and Secret objects can now tolerate the named source object or specific keys being missing, by adding `optional: true` to the volume or environment variable source specifications. ([#39981](https://github.com/kubernetes/kubernetes/pull/39981), [@fraenkel](https://github.com/fraenkel))
* Allow pods to define multiple environment variables from a whole ConfigMap ([#36245](https://github.com/kubernetes/kubernetes/pull/36245), [@fraenkel](https://github.com/fraenkel))

### CronJob
* Add configurable limits to CronJob resource to specify how many successful and failed jobs are preserved. ([#40932](https://github.com/kubernetes/kubernetes/pull/40932), [@peay](https://github.com/peay))

### Daemon Set
* DaemonSet now respects ControllerRef to avoid fighting over Pods. ([#42173](https://github.com/kubernetes/kubernetes/pull/42173), [@enisoc](https://github.com/enisoc))
* Make DaemonSet respect critical pods annotation when scheduling.  ([#42028](https://github.com/kubernetes/kubernetes/pull/42028), [@janetkuo](https://github.com/janetkuo))
* Implement the update feature for DaemonSet. ([#41116](https://github.com/kubernetes/kubernetes/pull/41116), [@lukaszo](https://github.com/lukaszo))
* Make DaemonSets survive taint-based evictions when nodes turn unreachable/notReady. ([#41896](https://github.com/kubernetes/kubernetes/pull/41896), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))
* Make DaemonSet controller respect node taints and pod tolerations.  ([#41172](https://github.com/kubernetes/kubernetes/pull/41172), [@janetkuo](https://github.com/janetkuo))
* DaemonSet controller actively kills failed pods (to recreate them) ([#40330](https://github.com/kubernetes/kubernetes/pull/40330), [@janetkuo](https://github.com/janetkuo))
* DaemonSet ObservedGeneration ([#39157](https://github.com/kubernetes/kubernetes/pull/39157), [@lukaszo](https://github.com/lukaszo))

### Deployment
* Add ready replicas in Deployments ([#37959](https://github.com/kubernetes/kubernetes/pull/37959), [@kargakis](https://github.com/kargakis))
* Deployments that cannot make progress in rolling out the newest version will now indicate via the API they are blocked
* Introduce apps/v1beta1.Deployments resource with modified defaults compared to extensions/v1beta1.Deployments. ([#39683](https://github.com/kubernetes/kubernetes/pull/39683), [@soltysh](https://github.com/soltysh))
* Introduce new generator for apps/v1beta1 deployments ([#42362](https://github.com/kubernetes/kubernetes/pull/42362), [@soltysh](https://github.com/soltysh))

### Node
* Set all node conditions to Unknown when node is unreachable ([#36592](https://github.com/kubernetes/kubernetes/pull/36592), [@andrewsykim](https://github.com/andrewsykim))

### Pod
* Init containers have graduated to GA and now appear as a field.  The beta annotation value will still be respected and overrides the field value. ([#38382](https://github.com/kubernetes/kubernetes/pull/38382), [@hodovska](https://github.com/hodovska))
* A new field `terminationMessagePolicy` has been added to containers that allows a user to request `FallbackToLogsOnError`, which will read from the container's logs to populate the termination message if the user does not write to the termination message log file.  The termination message file is now properly readable for end users and has a maximum size (4k bytes) to prevent abuse.  Each pod may have up to 12k bytes of termination messages before the contents of each will be truncated. ([#39341](https://github.com/kubernetes/kubernetes/pull/39341), [@smarterclayton](https://github.com/smarterclayton))
* Fix issue with PodDisruptionBudgets in which `minAvailable` specified as a percentage did not work with StatefulSet Pods. ([#39454](https://github.com/kubernetes/kubernetes/pull/39454), [@foxish](https://github.com/foxish))
* Node affinity has moved from annotations to api fields in the pod spec.  Node affinity that is defined in the annotations will be ignored. ([#37299](https://github.com/kubernetes/kubernetes/pull/37299), [@rrati](https://github.com/rrati))
* Moved pod affinity and anti-affinity from annotations to api fields [#25319](https://github.com/kubernetes/kubernetes/pull/25319) ([#39478](https://github.com/kubernetes/kubernetes/pull/39478), [@rrati](https://github.com/rrati))
* Add QoS pod status field ([#37968](https://github.com/kubernetes/kubernetes/pull/37968), [@sjenning](https://github.com/sjenning))

### Pod Security Policy
* PodSecurityPolicy resource is now enabled by default in the extensions API group. ([#39743](https://github.com/kubernetes/kubernetes/pull/39743), [@pweil-](https://github.com/pweil-))

### RBAC
* the `attributeRestrictions` field has been removed from the PolicyRule type in the rbac.authorization.k8s.io/v1alpha1 API. The field was not used by the RBAC authorizer. ([#39625](https://github.com/kubernetes/kubernetes/pull/39625), [@deads2k](https://github.com/deads2k))
* A user can now be authorized to bind a particular role by having permission to perform the `bind` verb on the referenced role ([#39383](https://github.com/kubernetes/kubernetes/pull/39383), [@liggitt](https://github.com/liggitt))

### Replica Set
* ReplicaSet has onwer ref of the Deployment that created it ([#35676](https://github.com/kubernetes/kubernetes/pull/35676), [@krmayankk](https://github.com/krmayankk))

### Secrets
* Populate environment variables from a secrets. ([#39446](https://github.com/kubernetes/kubernetes/pull/39446), [@fraenkel](https://github.com/fraenkel))
* Added a new secret type "bootstrap.kubernetes.io/token" for dynamically creating TLS bootstrapping bearer tokens. ([#41281](https://github.com/kubernetes/kubernetes/pull/41281), [@ericchiang](https://github.com/ericchiang))

### Service
* Endpoints, that tolerate unready Pods, are now listing Pods in state Terminating as well ([#37093](https://github.com/kubernetes/kubernetes/pull/37093), [@simonswine](https://github.com/simonswine))
* Fix Service Update on LoadBalancerSourceRanges Field ([#37720](https://github.com/kubernetes/kubernetes/pull/37720), [@freehan](https://github.com/freehan))
* Bug fix. Incoming UDP packets not reach newly deployed services ([#32561](https://github.com/kubernetes/kubernetes/pull/32561), [@zreigz](https://github.com/zreigz))
* Services of type loadbalancer consume both loadbalancer and nodeport quota. ([#39364](https://github.com/kubernetes/kubernetes/pull/39364), [@zhouhaibing089](https://github.com/zhouhaibing089))

### Stateful Set

* Fix zone placement heuristics so that multiple mounts in a StatefulSet pod are created in the same zone ([#40910](https://github.com/kubernetes/kubernetes/pull/40910), [@justinsb](https://github.com/justinsb))
* Fixes issue [#38418](https://github.com/kubernetes/kubernetes/pull/38418) which, under circumstance, could cause StatefulSet to deadlock.  ([#40838](https://github.com/kubernetes/kubernetes/pull/40838), [@kow3ns](https://github.com/kow3ns))
  * Mediates issue [#36859](https://github.com/kubernetes/kubernetes/pull/36859). StatefulSet only acts on Pods whose identity matches the StatefulSet, providing a partial mediation for overlapping controllers.

### Taints and Tolerations
* Add a manager to NodeController that is responsible for removing Pods from Nodes tainted with NoExecute Taints. This feature is beta (as the rest of taints) and enabled by default. It's gated by controller-manager enable-taint-manager flag. ([#40355](https://github.com/kubernetes/kubernetes/pull/40355), [@gmarek](https://github.com/gmarek))
* Add an alpha feature that makes NodeController set Taints instead of deleting Pods from not Ready Nodes. ([#41133](https://github.com/kubernetes/kubernetes/pull/41133), [@gmarek](https://github.com/gmarek))
* Make tolerations respect wildcard key ([#39914](https://github.com/kubernetes/kubernetes/pull/39914), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))
* Forgiveness alpha version api definition ([#39469](https://github.com/kubernetes/kubernetes/pull/39469), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))
* Rescheduler uses taints in v1beta1 and will remove old ones (in version v1alpha1) right after its start. ([#43106](https://github.com/kubernetes/kubernetes/pull/43106), [@piosz](https://github.com/piosz))

### Volumes
* StorageClassName attribute has been added to PersistentVolume and PersistentVolumeClaim objects and should be used instead of annotation `volume.beta.kubernetes.io/storage-class`. The beta annotation is still working in this release, however it will be removed in a future release. ([#42128](https://github.com/kubernetes/kubernetes/pull/42128), [@jsafrane](https://github.com/jsafrane))
* Parameter keys in a StorageClass `parameters` map may not use the `kubernetes.io` or `k8s.io` namespaces. ([#41837](https://github.com/kubernetes/kubernetes/pull/41837), [@liggitt](https://github.com/liggitt))
* Add storage.k8s.io/v1 API ([#40088](https://github.com/kubernetes/kubernetes/pull/40088), [@jsafrane](https://github.com/jsafrane))
* Alpha version of dynamic volume provisioning is removed in this release. Annotation ([#40000](https://github.com/kubernetes/kubernetes/pull/40000), [@jsafrane](https://github.com/jsafrane))
    * "volume.alpha.kubernetes.io/storage-class" does not have any special meaning. A default storage class
    * and  DefaultStorageClass admission plugin can be used to preserve similar behavior of Kubernetes cluster,
    * see https://kubernetes.io/docs/user-guide/persistent-volumes/#class-1 for details.
* Reduce verbosity of volume reconciler when attaching volumes ([#36900](https://github.com/kubernetes/kubernetes/pull/36900), [@codablock](https://github.com/codablock))
* We change the default attach_detach_controller sync period to 1 minute to reduce the query frequency through cloud provider to check whether volumes are attached or not.  ([#41363](https://github.com/kubernetes/kubernetes/pull/41363), [@jingxu97](https://github.com/jingxu97))

## Changes to Major Components
### API Server
* **`--anonymous-auth` is enabled by default, unless the API server is started with the `AlwaysAllow` authorizer. ([#38706](https://github.com/kubernetes/kubernetes/pull/38706), [@deads2k](https://github.com/deads2k))**
* **when using OIDC authentication and specifying --oidc-username-claim=email, an `"email_verified":true` claim must be returned from the identity provider. ([#36087](https://github.com/kubernetes/kubernetes/pull/36087), [@ericchiang](https://github.com/ericchiang))**
  * `--basic-auth-file` supports optionally specifying groups in the fourth column of the file ([#39651](https://github.com/kubernetes/kubernetes/pull/39651), [@liggitt](https://github.com/liggitt))
* API server now has two separate limits for read-only and mutating inflight requests. ([#36064](https://github.com/kubernetes/kubernetes/pull/36064), [@gmarek](https://github.com/gmarek))
* Restored normalization of custom `--etcd-prefix` when `--storage-backend` is set to etcd3 ([#42506](https://github.com/kubernetes/kubernetes/pull/42506), [@liggitt](https://github.com/liggitt))
* Updating apiserver to return http status code 202 for a delete request when the resource is not immediately deleted because of user requesting cascading deletion using DeleteOptions.OrphanDependents=false. ([#41165](https://github.com/kubernetes/kubernetes/pull/41165), [@nikhiljindal](https://github.com/nikhiljindal))
* Use full package path for definition name in OpenAPI spec ([#40124](https://github.com/kubernetes/kubernetes/pull/40124), [@mbohlool](https://github.com/mbohlool))
* Follow redirects for streaming requests (exec/attach/port-forward) in the apiserver by default (alpha -> beta). ([#40039](https://github.com/kubernetes/kubernetes/pull/40039), [@timstclair](https://github.com/timstclair))
* Add 'X-Content-Type-Options: nosniff" to some error messages ([#37190](https://github.com/kubernetes/kubernetes/pull/37190), [@brendandburns](https://github.com/brendandburns))
* Fixes bug in resolving client-requested API versions ([#38533](https://github.com/kubernetes/kubernetes/pull/38533), [@DirectXMan12](https://github.com/DirectXMan12))
* Replace glog.Fatals with fmt.Errorfs ([#38175](https://github.com/kubernetes/kubernetes/pull/38175), [@sttts](https://github.com/sttts))
* Pipe get options to storage ([#37693](https://github.com/kubernetes/kubernetes/pull/37693), [@wojtek-t](https://github.com/wojtek-t))
* The --long-running-request-regexp flag to kube-apiserver is deprecated and will be removed in a future release. Long-running requests are now detected based on specific verbs (watch, proxy) or subresources (proxy, portforward, log, exec, attach). ([#38119](https://github.com/kubernetes/kubernetes/pull/38119), [@liggitt](https://github.com/liggitt))
* if kube-apiserver is started with `--storage-backend=etcd2`, the media type `application/json` is used. ([#43122](https://github.com/kubernetes/kubernetes/pull/43122), [@liggitt](https://github.com/liggitt))
* API fields that previously serialized null arrays as `null` and empty arrays as `[]` no longer distinguish between those values and always output `[]` when serializing to JSON. ([#43422](https://github.com/kubernetes/kubernetes/pull/43422), [@liggitt](https://github.com/liggitt))
* Generate OpenAPI definition for inlined types ([#39466](https://github.com/kubernetes/kubernetes/pull/39466), [@mbohlool](https://github.com/mbohlool))

### API Server Aggregator
* Rename kubernetes-discovery to kube-aggregator ([#39619](https://github.com/kubernetes/kubernetes/pull/39619), [@deads2k](https://github.com/deads2k))
* Fix connection upgrades through kuberentes-discovery ([#38724](https://github.com/kubernetes/kubernetes/pull/38724), [@deads2k](https://github.com/deads2k))

#### Generic API Server
* Move pkg/api/rest into genericapiserver ([#39948](https://github.com/kubernetes/kubernetes/pull/39948), [@sttts](https://github.com/sttts))
* Move non-generic apiserver code out of the generic packages ([#38191](https://github.com/kubernetes/kubernetes/pull/38191), [@sttts](https://github.com/sttts))
* Fixes API compatibility issue with empty lists incorrectly returning a null `items` field instead of an empty array. ([#39834](https://github.com/kubernetes/kubernetes/pull/39834), [@liggitt](https://github.com/liggitt))
* Re-add /healthz/ping handler in genericapiserver ([#38603](https://github.com/kubernetes/kubernetes/pull/38603), [@sttts](https://github.com/sttts))
* Remove genericapiserver.Options.MasterServiceNamespace ([#38186](https://github.com/kubernetes/kubernetes/pull/38186), [@sttts](https://github.com/sttts))
* genericapiserver: cut off more dependencies – episode 3 ([#40426](https://github.com/kubernetes/kubernetes/pull/40426), [@sttts](https://github.com/sttts))
* genericapiserver: more dependency cutoffs ([#40216](https://github.com/kubernetes/kubernetes/pull/40216), [@sttts](https://github.com/sttts))
* genericapiserver: cut off kube pkg/version dependency ([#39943](https://github.com/kubernetes/kubernetes/pull/39943), [@sttts](https://github.com/sttts))
* genericapiserver: cut off pkg/serviceaccount dependency ([#39945](https://github.com/kubernetes/kubernetes/pull/39945), [@sttts](https://github.com/sttts))
* genericapiserver: cut off pkg/apis/extensions and pkg/storage dependencies ([#39946](https://github.com/kubernetes/kubernetes/pull/39946), [@sttts](https://github.com/sttts))
* genericapiserver: cut off certificates api dependency ([#39947](https://github.com/kubernetes/kubernetes/pull/39947), [@sttts](https://github.com/sttts))
* genericapiserver: extract CA cert from server cert and SNI cert chains ([#39022](https://github.com/kubernetes/kubernetes/pull/39022), [@sttts](https://github.com/sttts))
* genericapiserver: turn APIContainer.SecretRoutes into a real ServeMux ([#38826](https://github.com/kubernetes/kubernetes/pull/38826), [@sttts](https://github.com/sttts))
* genericapiserver: unify swagger and openapi in config ([#38690](https://github.com/kubernetes/kubernetes/pull/38690), [@sttts](https://github.com/sttts))

### Client
* Use Prometheus instrumentation conventions ([#36704](https://github.com/kubernetes/kubernetes/pull/36704), [@fabxc](https://github.com/fabxc))
* Clients now use the `?watch=true` parameter to make watch API calls, instead of the `/watch/` path prefix ([#41722](https://github.com/kubernetes/kubernetes/pull/41722), [@liggitt](https://github.com/liggitt))
* Move private key parsing from serviceaccount/jwt.go to client-go/util/cert ([#40907](https://github.com/kubernetes/kubernetes/pull/40907), [@cblecker](https://github.com/cblecker))
* Caching added to the OIDC client auth plugin to fix races and reduce the time kubectl commands using this plugin take by several seconds. ([#38167](https://github.com/kubernetes/kubernetes/pull/38167), [@ericchiang](https://github.com/ericchiang))
* Fix resync goroutine leak in ListAndWatch ([#35672](https://github.com/kubernetes/kubernetes/pull/35672), [@tatsuhiro-t](https://github.com/tatsuhiro-t))

#### client-go
* The main repository does not keep multiple releases of clientsets anymore. Please find previous releases at https://github.com/kubernetes/client-go ([#38154](https://github.com/kubernetes/kubernetes/pull/38154), [@caesarxuchao](https://github.com/caesarxuchao))
* Support whitespace in command path for gcp auth plugin ([#41653](https://github.com/kubernetes/kubernetes/pull/41653), [@jlowdermilk](https://github.com/jlowdermilk))
* client-go no longer imports GCP OAuth2 and OpenID Connect packages by default. ([#41532](https://github.com/kubernetes/kubernetes/pull/41532), [@ericchiang](https://github.com/ericchiang))
* Added bool type support for jsonpath. ([#39063](https://github.com/kubernetes/kubernetes/pull/39063), [@xingzhou](https://github.com/xingzhou))
* Preventing nil pointer reference in client_config ([#40508](https://github.com/kubernetes/kubernetes/pull/40508), [@vjsamuel](https://github.com/vjsamuel))
* Prevent hotloops on error conditions, which could fill up the disk faster than log rotation can free space. ([#40497](https://github.com/kubernetes/kubernetes/pull/40497), [@lavalamp](https://github.com/lavalamp))

### Cloud Provider

#### AWS
* Allow to running the master with a different AWS account or even on a different cloud provider than the nodes. ([#39996](https://github.com/kubernetes/kubernetes/pull/39996), [@scheeles](https://github.com/scheeles))
* Support shared tag `kubernetes.io/cluster/<clusterid>` ([#41695](https://github.com/kubernetes/kubernetes/pull/41695), [@justinsb](https://github.com/justinsb))
* Do not consider master instance zones for dynamic volume creation ([#41702](https://github.com/kubernetes/kubernetes/pull/41702), [@justinsb](https://github.com/justinsb))
* Fix AWS device allocator to only use valid device names ([#41455](https://github.com/kubernetes/kubernetes/pull/41455), [@gnufied](https://github.com/gnufied))
* Trust region if found from AWS metadata ([#38880](https://github.com/kubernetes/kubernetes/pull/38880), [@justinsb](https://github.com/justinsb))
* Remove duplicate calls to DescribeInstance during volume operations ([#39842](https://github.com/kubernetes/kubernetes/pull/39842), [@gnufied](https://github.com/gnufied))
* Recognize eu-west-2 region ([#38746](https://github.com/kubernetes/kubernetes/pull/38746), [@justinsb](https://github.com/justinsb))
* Recognize ca-central-1 region ([#38410](https://github.com/kubernetes/kubernetes/pull/38410), [@justinsb](https://github.com/justinsb))
* Add sequential allocator for device names. ([#38818](https://github.com/kubernetes/kubernetes/pull/38818), [@jsafrane](https://github.com/jsafrane))

#### Azure
* Fix failing load balancers in Azure ([#40405](https://github.com/kubernetes/kubernetes/pull/40405), [@codablock](https://github.com/codablock))
* Reduce time needed to attach Azure disks ([#40066](https://github.com/kubernetes/kubernetes/pull/40066), [@codablock](https://github.com/codablock))
* Remove Azure Subnet RouteTable check ([#38334](https://github.com/kubernetes/kubernetes/pull/38334), [@mogthesprog](https://github.com/mogthesprog))
* Add support for Azure Container Registry, update Azure dependencies ([#37783](https://github.com/kubernetes/kubernetes/pull/37783), [@brendandburns](https://github.com/brendandburns))
* Allow backendpools in Azure Load Balancers which are not owned by cloud provider ([#36882](https://github.com/kubernetes/kubernetes/pull/36882), [@codablock](https://github.com/codablock))

#### GCE
* On GCI by default logrotate is disabled for application containers in favor of rotation mechanism provided by docker logging driver. ([#40634](https://github.com/kubernetes/kubernetes/pull/40634), [@crassirostris](https://github.com/crassirostris))

#### GKE
* New GKE certificates controller. ([#41160](https://github.com/kubernetes/kubernetes/pull/41160), [@pipejakob](https://github.com/pipejakob))

#### vSphere
* Reverts to looking up the current VM in vSphere using the machine's UUID, either obtained via sysfs or via the `vm-uuid` parameter in the cloud configuration file. ([#40892](https://github.com/kubernetes/kubernetes/pull/40892), [@robdaemon](https://github.com/robdaemon))
* Fix for detach volume when node is not present/ powered off ([#40118](https://github.com/kubernetes/kubernetes/pull/40118), [@BaluDontu](https://github.com/BaluDontu))
* Adding vmdk file extension for vmDiskPath in vsphere DeleteVolume ([#40538](https://github.com/kubernetes/kubernetes/pull/40538), [@divyenpatel](https://github.com/divyenpatel))
* Changed default scsi controller type in vSphere Cloud Provider ([#38426](https://github.com/kubernetes/kubernetes/pull/38426), [@abrarshivani](https://github.com/abrarshivani))
* Fixes NotAuthenticated errors that appear in the kubelet and kube-controller-manager due to never logging in to vSphere ([#36169](https://github.com/kubernetes/kubernetes/pull/36169), [@robdaemon](https://github.com/robdaemon))
* Fix panic in vSphere cloud provider ([#38423](https://github.com/kubernetes/kubernetes/pull/38423), [@BaluDontu](https://github.com/BaluDontu))
* Fix space issue in volumePath with vSphere Cloud Provider ([#38338](https://github.com/kubernetes/kubernetes/pull/38338), [@BaluDontu](https://github.com/BaluDontu))

### Federation
#### Federated DNS
* Add logging of route53 calls ([#39964](https://github.com/kubernetes/kubernetes/pull/39964), [@justinsb](https://github.com/justinsb))

#### Kubefed
* Flag cleanup ([#41335](https://github.com/kubernetes/kubernetes/pull/41335), [@irfanurrehman](https://github.com/irfanurrehman))
* Create configmap for the cluster kube-dns when cluster joins and remove when it unjoins ([#39338](https://github.com/kubernetes/kubernetes/pull/39338), [@irfanurrehman](https://github.com/irfanurrehman))
* Support configuring dns-provider ([#40528](https://github.com/kubernetes/kubernetes/pull/40528), [@shashidharatd](https://github.com/shashidharatd))
* Bug fix relating kubeconfig path in kubefed init ([#41410](https://github.com/kubernetes/kubernetes/pull/41410), [@irfanurrehman](https://github.com/irfanurrehman))
* Add override flags options to kubefed init ([#40917](https://github.com/kubernetes/kubernetes/pull/40917), [@irfanurrehman](https://github.com/irfanurrehman))
* Add option to expose federation apiserver on nodeport service ([#40516](https://github.com/kubernetes/kubernetes/pull/40516), [@shashidharatd](https://github.com/shashidharatd))
* Add option to disable persistence storage for etcd ([#40862](https://github.com/kubernetes/kubernetes/pull/40862), [@shashidharatd](https://github.com/shashidharatd))
* kubefed init creates a service account for federation controller manager in the federation-system namespace and binds that service account to the federation-system:federation-controller-manager role that has read and list access on secrets in the federation-system namespace.  ([#40392](https://github.com/kubernetes/kubernetes/pull/40392), [@madhusudancs](https://github.com/madhusudancs))
* Implement dry run support in kubefed init ([#36447](https://github.com/kubernetes/kubernetes/pull/36447), [@irfanurrehman](https://github.com/irfanurrehman))
* Make federation etcd PVC size configurable ([#36310](https://github.com/kubernetes/kubernetes/pull/36310), [@irfanurrehman](https://github.com/irfanurrehman))

#### Other Notable Changes
* Federated Ingress over GCE no longer requires separate firewall rules to be created for each cluster to circumvent flapping firewall health checks. ([#41942](https://github.com/kubernetes/kubernetes/pull/41942), [@csbell](https://github.com/csbell))
* Add support for finalizers in federated configmaps (deletes configmaps from underlying clusters). ([#40464](https://github.com/kubernetes/kubernetes/pull/40464), [@csbell](https://github.com/csbell))
* Add support for DeleteOptions.OrphanDependents for federated services. Setting it to false while deleting a federated service also deletes the corresponding services from all registered clusters. ([#36390](https://github.com/kubernetes/kubernetes/pull/36390), [@nikhiljindal](https://github.com/nikhiljindal))
* Add `--controllers` flag to federation controller manager for enable/disable federation ingress controller ([#36643](https://github.com/kubernetes/kubernetes/pull/36643), [@kzwang](https://github.com/kzwang))
* Stop deleting services from underlying clusters when federated service is deleted. ([#37353](https://github.com/kubernetes/kubernetes/pull/37353), [@nikhiljindal](https://github.com/nikhiljindal))
* Expose autoscaling apis through federation api server ([#38976](https://github.com/kubernetes/kubernetes/pull/38976), [@irfanurrehman](https://github.com/irfanurrehman))
* Federation: Add `batch/jobs` API objects to federation-apiserver ([#35943](https://github.com/kubernetes/kubernetes/pull/35943), [@jianhuiz](https://github.com/jianhuiz))

### Garbage Collector
* Added foreground garbage collection: the owner object will not be deleted until all its dependents are deleted by the garbage collector. Please checkout the [user doc](https://kubernetes.io/docs/concepts/abstractions/controllers/garbage-collection/) for details. ([#38676](https://github.com/kubernetes/kubernetes/pull/38676), [@caesarxuchao](https://github.com/caesarxuchao))
  * deleteOptions.orphanDependents is going to be deprecated in 1.7. Please use deleteOptions.propagationPolicy instead.

### kubeadm
* A new label and taint is used for marking the master. The label is `node-role.kubernetes.io/master=""` and the taint has the effect `NoSchedule`. If you want to schedule a workload one the master (a networking DaemonSet for example), you must tolerate the `node-role.kubernetes.io/master="":NoSchedule` taint
* The kubelet API is now secured, only cluster admins are allowed to access it.
* Insecure access to the API Server over `localhost:8080` is now disabled.
* The control plane components now talk securely to each other. The API Server talks securely to the kubelets in the cluster.
* kubeadm creates RBAC-enabled clusters. This means that some add-ons which worked previously won't work without having their YAML configuration updated. Please see each vendor's own documentation to check that the add-ons you are using will work with Kubernetes 1.6.
* The kube-discovery Deployment isn't used anymore when creating a kubeadm cluster and shouldn't be used anywhere else either due to its insecure nature.
* The Certificates API has graduated from alpha to beta. This is a backwards-incompatible change since the alpha support is dropped, and therefore kubeadm v1.5 and v1.6 don't work together (for example kubeadm v1.5 can't join a kubeadm v1.6 cluster).
* Supporting only etcd3, with 3.0.14 as the minimum version.
* `kubeadm reset` will no longer drain nodes automatically.  This is because the credentials on nodes do not have permission to perform this operation.  We have documented an [alternate procedure](https://kubernetes.io/docs/getting-started-guides/kubeadm/#tear-down), driven from the API server/master.
* Hook up kubeadm against the BootstrapSigner ([#41417](https://github.com/kubernetes/kubernetes/pull/41417), [@luxas](https://github.com/luxas))
* Rename some flags for beta UI and fixup some logic ([#42064](https://github.com/kubernetes/kubernetes/pull/42064), [@luxas](https://github.com/luxas))
* Insecure access to the API Server at localhost:8080 will be turned off in v1.6 when using kubeadm ([#42066](https://github.com/kubernetes/kubernetes/pull/42066), [@luxas](https://github.com/luxas))
* Flag --use-kubernetes-version for kubeadm init renamed to --kubernetes-version ([#41820](https://github.com/kubernetes/kubernetes/pull/41820), [@kad](https://github.com/kad))
* Remove the --cloud-provider flag for beta init UX ([#41710](https://github.com/kubernetes/kubernetes/pull/41710), [@luxas](https://github.com/luxas))
* Fixed an SELinux issue in kubeadm on Docker 1.12+ by moving etcd SELinux options from container to pod. ([#40682](https://github.com/kubernetes/kubernetes/pull/40682), [@dgoodwin](https://github.com/dgoodwin))
* Add authorization mode to kubeadm ([#39846](https://github.com/kubernetes/kubernetes/pull/39846), [@andrewrynhard](https://github.com/andrewrynhard))
* Refactor the certificate and kubeconfig code in the kubeadm binary into two phases ([#39280](https://github.com/kubernetes/kubernetes/pull/39280), [@luxas](https://github.com/luxas))
* Added kubeadm commands to manage bootstrap tokens and the duration they are valid for. ([#35805](https://github.com/kubernetes/kubernetes/pull/35805), [@dgoodwin](https://github.com/dgoodwin))

### kubectl

#### New Commands
- `apply set-last-applied` *updates the applied-applied-configuration annotation* ([#41694](https://github.com/kubernetes/kubernetes/pull/41694), [@shiywang](https://github.com/shiywang))
- `kubectl apply view-last-applied` *viewing the last configuration file applied* ([#41146](https://github.com/kubernetes/kubernetes/pull/41146), [@shiywang](https://github.com/shiywang))

#### Create subcommands
  - `create poddisruptionbudget` ([#36646](https://github.com/kubernetes/kubernetes/pull/36646), [@kargakis](https://github.com/kargakis))
  - `create clusterrole` ([#41538](https://github.com/kubernetes/kubernetes/pull/41538), [@xingzhou](https://github.com/xingzhou))
  - `create role` ([#39852](https://github.com/kubernetes/kubernetes/pull/39852), [@xingzhou](https://github.com/xingzhou))
  - `create clusterrolebinding` ([#37098](https://github.com/kubernetes/kubernetes/pull/37098), [@deads2k](https://github.com/deads2k))
  - `create service externalname` ([#34789](https://github.com/kubernetes/kubernetes/pull/34789), [@adohe](https://github.com/adohe))
- `set selector` - update selector labels ([#38966](https://github.com/kubernetes/kubernetes/pull/38966), [@kargakis](https://github.com/kargakis))
- `can-i` to see if you can perform an action ([#41077](https://github.com/kubernetes/kubernetes/pull/41077), [@deads2k](https://github.com/deads2k))

#### Updates to existing commands
* Printing and output
  * Import a numeric ordering sorting library and use it in the sorting printer. ([#40746](https://github.com/kubernetes/kubernetes/pull/40746), [@matthyx](https://github.com/matthyx))
  * DaemonSet get and describe show status fields.  ([#42843](https://github.com/kubernetes/kubernetes/pull/42843), [@janetkuo](https://github.com/janetkuo))
  * Pods describe shows tolerationSeconds ([#42162](https://github.com/kubernetes/kubernetes/pull/42162), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))
  * Node describe contains closing paren ([#39217](https://github.com/kubernetes/kubernetes/pull/39217), [@luksa](https://github.com/luksa))
  * Display pod node selectors with kubectl describe. ([#36396](https://github.com/kubernetes/kubernetes/pull/36396), [@aveshagarwal](https://github.com/aveshagarwal))
  * Add Version to the resource printer for 'get nodes' ([#37943](https://github.com/kubernetes/kubernetes/pull/37943), [@ailusazh](https://github.com/ailusazh))
  * Added support for printing in all supported `--output` formats to `kubectl create ...` and `kubectl apply ...` ([#38112](https://github.com/kubernetes/kubernetes/pull/38112), [@juanvallejo](https://github.com/juanvallejo))
  * Add three more columns to `kubectl get deploy -o wide` output. ([#39240](https://github.com/kubernetes/kubernetes/pull/39240), [@xingzhou](https://github.com/xingzhou))
  * Fix kubectl get -f <file> -o <nondefault printer> so it prints all items in the file ([#39038](https://github.com/kubernetes/kubernetes/pull/39038), [@ncdc](https://github.com/ncdc))
  * kubectl describe no longer prints the last-applied-configuration annotation for secrets. ([#34664](https://github.com/kubernetes/kubernetes/pull/34664), [@ymqytw](https://github.com/ymqytw))
  * Completed pods should not be hidden when requested by name via `kubectl get`. ([#42216](https://github.com/kubernetes/kubernetes/pull/42216), [@smarterclayton](https://github.com/smarterclayton))
  * Improve formatting of EventSource in kubectl get and kubectl describe ([#40073](https://github.com/kubernetes/kubernetes/pull/40073), [@matthyx](https://github.com/matthyx))
* Attach now supports multiple types ([#40365](https://github.com/kubernetes/kubernetes/pull/40365), [@shiywang](https://github.com/shiywang))
* Create now accepts the label selector flag for filtering objects to create ([#40057](https://github.com/kubernetes/kubernetes/pull/40057), [@MrHohn](https://github.com/MrHohn))
* Top now accepts short forms for "node" and "pod" ("no", "po") ([#39218](https://github.com/kubernetes/kubernetes/pull/39218), [@luksa](https://github.com/luksa))
* Begin paths for internationalization in kubectl ([#36802](https://github.com/kubernetes/kubernetes/pull/36802), [@brendandburns](https://github.com/brendandburns))
  * Add initial french translations for kubectl ([#40645](https://github.com/kubernetes/kubernetes/pull/40645), [@brendandburns](https://github.com/brendandburns))

#### Updates to apply 
* New command `apply set-last-applied` *updates the applied-applied-configuration annotation* ([#41694](https://github.com/kubernetes/kubernetes/pull/41694), [@shiywang](https://github.com/shiywang))
* New command `apply view-last-applied` *command for viewing the last configuration file applied* ([#41146](https://github.com/kubernetes/kubernetes/pull/41146), [@shiywang](https://github.com/shiywang))
* `apply` now supports explicitly clearing values by setting them to null ([#40630](https://github.com/kubernetes/kubernetes/pull/40630), [@liggitt](https://github.com/liggitt))
* Warn user when using `apply` on a resource lacking the `LastAppliedConfig` annotation ([#36672](https://github.com/kubernetes/kubernetes/pull/36672), [@ymqytw](https://github.com/ymqytw))
* PATCH (i.e. apply and edit) now supports merging lists of primitives ([#38665](https://github.com/kubernetes/kubernetes/pull/38665), [@ymqytw](https://github.com/ymqytw))
* `apply` falls back to generic 3-way JSON merge patch for Third Party Resource or unregistered types ([#40666](https://github.com/kubernetes/kubernetes/pull/40666), [@ymqytw](https://github.com/ymqytw))

#### Updates to edit
* `edit` now supports Third party resources and extension API servers. ([#41304](https://github.com/kubernetes/kubernetes/pull/41304), [@liggitt](https://github.com/liggitt))
  * Now to edit a particular API version, provide the fully-qualify the resource, version, and group used to fetch the object (for example, `job.v1.batch/myjob`)
  * Client-side conversion is no longer done, and the `--output-version` option is deprecated for `kubectl edit`.
* `edit` works as intended with apply and will not change the annotation
  * No longer updates the last-applied-configuration annotation when --save-config is unspecified or false. ([#41924](https://github.com/kubernetes/kubernetes/pull/41924), [@ymqytw](https://github.com/ymqytw))
  * Fixes issue that caused apply to revert changes made by edit

#### Bug fixes
* Fixed --save-config in create subcommand to save the annotation ([#40289](https://github.com/kubernetes/kubernetes/pull/40289), [@xilabao](https://github.com/xilabao))
* Fixed an issue where 'kubectl get --sort-by=' would return an error if the specified fields in sort were not specified in one or more of the returned objects. ([#40541](https://github.com/kubernetes/kubernetes/pull/40541), [@fabianofranz](https://github.com/fabianofranz))
  * Previously this would cause the command to fail regardless of whether or not the field was present in the object model
  * Now the command will succeed even if the sort-by field is missing from one or more of the objects
* Fixed issue with kubectl proxy so it will now proxy an empty path - e.g. http://localhost:8001 ([#39226](https://github.com/kubernetes/kubernetes/pull/39226), [@luksa](https://github.com/luksa))
* Fixed an issue where commas were not accepted in --from-literal flags for the creation of secrets. ([#35191](https://github.com/kubernetes/kubernetes/pull/35191), [@SamiHiltunen](https://github.com/SamiHiltunen))
  * Passing multiple values separated by a comma in a single --from-literal flag is no longer supported. Please use multiple --from-literal flags to provide multiple values. 
* Fixed a bug where the --server, --token, and --certificate-authority flags were not overriding the related in-cluster configs when provided in a `kubectl` call inside a cluster. ([#39006](https://github.com/kubernetes/kubernetes/pull/39006), [@fabianofranz](https://github.com/fabianofranz))

#### Other Notable Changes
* The api server will publish the extensions/Deployments API as preferred over the apps/Deployment (introduced in 1.6). ([#43553](https://github.com/kubernetes/kubernetes/pull/43553), [@liggitt](https://github.com/liggitt))
  * This will ensure certain commands in 1.5 versions of kubectl continue function when run against a 1.6 server. (e.g. `kubectl edit deployment`) 
* Taint
  * The `taint` command will not function in a skewed 1.5 / 1.6 environment - client and server versions must match (See Action required section above)
  * Change taints/tolerations to api fields ([#38957](https://github.com/kubernetes/kubernetes/pull/38957), [@aveshagarwal](https://github.com/aveshagarwal))
  * Make kubectl taint command respect effect NoExecute ([#42120](https://github.com/kubernetes/kubernetes/pull/42120), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))
* Allow drain --force to remove pods whose managing resource is deleted. ([#41864](https://github.com/kubernetes/kubernetes/pull/41864), [@marun](https://github.com/marun))
* `--output-version` is ignored for all commands except `kubectl convert`.  This is consistent with the generic nature of `kubectl` CRUD commands and the previous removal of `--api-version`.  Specific versions can be specified in the resource field: `resource.version.group`, `jobs.v1.batch`. ([#41576](https://github.com/kubernetes/kubernetes/pull/41576), [@deads2k](https://github.com/deads2k))
* Allow missing keys in templates by default ([#39486](https://github.com/kubernetes/kubernetes/pull/39486), [@ncdc](https://github.com/ncdc))
* Add error message when trying to use clusterrole with namespace in kubectl ([#36424](https://github.com/kubernetes/kubernetes/pull/36424), [@xilabao](https://github.com/xilabao))
* When deleting an object with `--grace-period=0`, the client will begin a graceful deletion and wait until the resource is fully deleted.  To force deletion, use the `--force` flag. ([#37263](https://github.com/kubernetes/kubernetes/pull/37263), [@smarterclayton](https://github.com/smarterclayton))

### kubelet
* Apply taint tolerations for NoExecute for all static pods. ([#43116](https://github.com/kubernetes/kubernetes/pull/43116), [@dchen1107](https://github.com/dchen1107))
* Disable devicemapper thin_ls due to excessive iops ([#42899](https://github.com/kubernetes/kubernetes/pull/42899), [@dashpole](https://github.com/dashpole))
* Dropped the support for docker 1.9.x and the belows.  ([#42694](https://github.com/kubernetes/kubernetes/pull/42694), [@dchen1107](https://github.com/dchen1107))
* kubelet created cgroups follow lowercase naming conventions ([#42497](https://github.com/kubernetes/kubernetes/pull/42497), [@derekwaynecarr](https://github.com/derekwaynecarr))
* kubelet exports metrics for cgroup management ([#41988](https://github.com/kubernetes/kubernetes/pull/41988), [@sjenning](https://github.com/sjenning))
* Pods are launched in a separate cgroup hierarchy than system services. ([#42350](https://github.com/kubernetes/kubernetes/pull/42350), [@vishh](https://github.com/vishh))
* Experimental support to reserve a pod's memory request from being utilized by pods in lower QoS tiers. ([#41149](https://github.com/kubernetes/kubernetes/pull/41149), [@sjenning](https://github.com/sjenning))
* `--experimental-nvidia-gpus` flag is **replaced** by `Accelerators` alpha feature gate along with  support for multiple Nvidia GPUs.  ([#42116](https://github.com/kubernetes/kubernetes/pull/42116), [@vishh](https://github.com/vishh))
  * To use GPUs, pass `Accelerators=true` as part of `--feature-gates` flag.
  * Works only with Docker runtime.
* New Kubelet flag `--enforce-node-allocatable` with a default value of `pods` is added which will make kubelet create a top level cgroup for all pods to enforce Node Allocatable. Optionally, `system-reserved` & `kube-reserved` values can also be specified separated by comma to enforce node allocatable on cgroups specified via `--system-reserved-cgroup` & `--kube-reserved-cgroup` respectively. Note the default value of the latter flags are "". ([#41234](https://github.com/kubernetes/kubernetes/pull/41234), [@vishh](https://github.com/vishh))
  * This feature requires a **Node Drain** prior to upgrade failing which pods will be restarted if possible or terminated if they have a `RestartNever` policy.
* Guaranteed admission for Critical Pods ([#40952](https://github.com/kubernetes/kubernetes/pull/40952), [@dashpole](https://github.com/dashpole))
* Deprecate outofdisk-transition-frequency and low-diskspace-threshold-mb flags ([#41941](https://github.com/kubernetes/kubernetes/pull/41941), [@dashpole](https://github.com/dashpole))
* kubelet config should ignore file start with dots ([#39196](https://github.com/kubernetes/kubernetes/pull/39196), [@resouer](https://github.com/resouer))
* Each pod has its own associated cgroup by default. ([#41349](https://github.com/kubernetes/kubernetes/pull/41349), [@derekwaynecarr](https://github.com/derekwaynecarr))
* Nodes can now report two additional address types in their status: InternalDNS and ExternalDNS. The apiserver can use `--kubelet-preferred-address-types` to give priority to the type of address it uses to reach nodes. ([#34259](https://github.com/kubernetes/kubernetes/pull/34259), [@liggitt](https://github.com/liggitt))
* Fix ConfigMap for Windows Containers. ([#39373](https://github.com/kubernetes/kubernetes/pull/39373), [@jbhurat](https://github.com/jbhurat))
* Node Problem Detector is beta now. New features added: journald support, standalone mode and arbitrary system log monitoring. ([#40206](https://github.com/kubernetes/kubernetes/pull/40206), [@Random-Liu](https://github.com/Random-Liu))
* Report node not ready on failed PLEG health check ([#41569](https://github.com/kubernetes/kubernetes/pull/41569), [@yujuhong](https://github.com/yujuhong))
* Delay Deletion of a Pod until volumes are cleaned up ([#41456](https://github.com/kubernetes/kubernetes/pull/41456), [@dashpole](https://github.com/dashpole))
* Make EnableCRI default to true ([#41378](https://github.com/kubernetes/kubernetes/pull/41378), [@yujuhong](https://github.com/yujuhong))
* The deprecated flags --config, --auth-path, --resource-container, and --system-container were removed. ([#40048](https://github.com/kubernetes/kubernetes/pull/40048), [@mtaufen](https://github.com/mtaufen))
* We should mention the caveats of in-place upgrade in release note. ([#40727](https://github.com/kubernetes/kubernetes/pull/40727), [@Random-Liu](https://github.com/Random-Liu))
* Delay deletion of pod from the API server until volumes are deleted ([#41095](https://github.com/kubernetes/kubernetes/pull/41095), [@dashpole](https://github.com/dashpole))
* Rename --experiemental-cgroups-per-qos to --cgroups-per-qos ([#39972](https://github.com/kubernetes/kubernetes/pull/39972), [@derekwaynecarr](https://github.com/derekwaynecarr))
* Remove the temporary fix for pre-1.0 mirror pods ([#40877](https://github.com/kubernetes/kubernetes/pull/40877), [@yujuhong](https://github.com/yujuhong))
* When feature gate "ExperimentalCriticalPodAnnotation" is set, Kubelet will avoid evicting pods in "kube-system" namespace that contains a special annotation - `scheduler.alpha.kubernetes.io/critical-pod` ([#40655](https://github.com/kubernetes/kubernetes/pull/40655), [@vishh](https://github.com/vishh))
* Port forwarding can forward over websockets or SPDY. ([#33684](https://github.com/kubernetes/kubernetes/pull/33684), [@fraenkel](https://github.com/fraenkel))
* CRI: use more gogoprotobuf plugins ([#40397](https://github.com/kubernetes/kubernetes/pull/40397), [@yujuhong](https://github.com/yujuhong))
* kubelet tears down pod volumes on pod termination rather than pod deletion ([#37228](https://github.com/kubernetes/kubernetes/pull/37228), [@sjenning](https://github.com/sjenning))
* CRI: upgrade protobuf to v3 ([#39158](https://github.com/kubernetes/kubernetes/pull/39158), [@feiskyer](https://github.com/feiskyer))
* Add SIGCHLD handler to pause container ([#36853](https://github.com/kubernetes/kubernetes/pull/36853), [@verb](https://github.com/verb))
* kubelet: remove the pleg health check from healthz ([#37865](https://github.com/kubernetes/kubernetes/pull/37865), [@yujuhong](https://github.com/yujuhong))
* Fixes nil dereference when doing a volume type check on persistent volumes ([#39493](https://github.com/kubernetes/kubernetes/pull/39493), [@sjenning](https://github.com/sjenning))
* The `--reconcile-cidr` kubelet flag was removed since it had been deprecated since v1.5 ([#39322](https://github.com/kubernetes/kubernetes/pull/39322), [@luxas](https://github.com/luxas))
* Remove 'exec' network plugin - use CNI instead ([#39254](https://github.com/kubernetes/kubernetes/pull/39254), [@freehan](https://github.com/freehan))
* Add path exist check in getPodVolumePathListFromDisk ([#38909](https://github.com/kubernetes/kubernetes/pull/38909), [@jingxu97](https://github.com/jingxu97))
* Don't evict static pods ([#39059](https://github.com/kubernetes/kubernetes/pull/39059), [@bprashanth](https://github.com/bprashanth))
* Assign -998 as the oom_score_adj for critical pods (e.g. kube-proxy) ([#39114](https://github.com/kubernetes/kubernetes/pull/39114), [@dchen1107](https://github.com/dchen1107))
* Admit critical pods in the kubelet ([#38836](https://github.com/kubernetes/kubernetes/pull/38836), [@bprashanth](https://github.com/bprashanth))
* Fail kubelet if runtime is unresponsive for 30 seconds ([#38527](https://github.com/kubernetes/kubernetes/pull/38527), [@derekwaynecarr](https://github.com/derekwaynecarr))
* Add image cache. ([#38375](https://github.com/kubernetes/kubernetes/pull/38375), [@Random-Liu](https://github.com/Random-Liu))
* kernel memcg notification enabled via experimental flag ([#38258](https://github.com/kubernetes/kubernetes/pull/38258), [@derekwaynecarr](https://github.com/derekwaynecarr))
* kubelet will no longer set hairpin mode on every interface on the machine when an error occurs in setting up hairpin for a specific interface. ([#36990](https://github.com/kubernetes/kubernetes/pull/36990), [@bboreham](https://github.com/bboreham))
* kubelet: don't reject pods without adding them to the pod manager ([#37661](https://github.com/kubernetes/kubernetes/pull/37661), [@yujuhong](https://github.com/yujuhong))

### kube-controller-manager
* add --controllers to controller manager ([#39740](https://github.com/kubernetes/kubernetes/pull/39740), [@deads2k](https://github.com/deads2k))

### kube-dns
* Adds support for configurable DNS stub domains and upstream nameservers.
  The following configuration options have been added to the `kube-system:kube-dns` ConfigMap:
  ```
  "stubDomains": {
    "acme.local": ["1.2.3.4"]
  },
  ```
  is a map of domain to list of nameservers for the domain. This is used
  to inject private DNS domains into the kube-dns namespace. In the above
  example, any DNS requests for *.acme.local will be served by the
  nameserver 1.2.3.4.
  ```
  "upstreamNameservers": ["8.8.8.8", "8.8.4.4"]
  ```
  is a list of upstreamNameservers to use, overriding the configuration
  specified in /etc/resolv.conf.

* <a name="#CNIcompatdetails"></a>Container Network Interface ([CNI](https://github.com/containernetworking/cni)) integration with Container
  Runtime Interface (CRI) is enabled by default.
  * The standard `bridge` plugin have been validated to interoperate with the new CRI + CNI code path.
  * If you are using plugins other than `bridge`, make sure you have updated custom plugins to the 
    latest version that is compatible.
  * If you encounter any issues involving CNI plugins, CRI can be disabled with setting the `--enable-cri`
    flag to `false`.
  * [Associated action required notes](#CNIcompat).
  
* An empty `kube-system:kube-dns` ConfigMap will be created for the cluster if one did not already exist.

### kube-proxy
* **- Add tcp/udp userspace proxy support for Windows. ([#41487](https://github.com/kubernetes/kubernetes/pull/41487), [@anhowe](https://github.com/anhowe))**
* Add DNS suffix search list support in Windows kube-proxy. ([#41618](https://github.com/kubernetes/kubernetes/pull/41618), [@JiangtianLi](https://github.com/JiangtianLi))
* Add a KUBERNETES_NODE_* section to build kubelet/kube-proxy for windows ([#38919](https://github.com/kubernetes/kubernetes/pull/38919), [@brendandburns](https://github.com/brendandburns))
* Remove outdated net.experimental.kubernetes.io/proxy-mode and net.beta.kubernetes.io/proxy-mode annotations from kube-proxy. ([#40585](https://github.com/kubernetes/kubernetes/pull/40585), [@cblecker](https://github.com/cblecker))
* proxy/iptables: don't sync proxy rules if services map didn't change ([#38996](https://github.com/kubernetes/kubernetes/pull/38996), [@dcbw](https://github.com/dcbw))
* Update kube-proxy image to be based off of Debian 8.6 base image. ([#39695](https://github.com/kubernetes/kubernetes/pull/39695), [@ixdy](https://github.com/ixdy))
* Update amd64 kube-proxy base image to debian-iptables-amd64:v5 ([#39725](https://github.com/kubernetes/kubernetes/pull/39725), [@ixdy](https://github.com/ixdy))
* Clean up the kube-proxy container image by removing unnecessary packages and files. ([#42090](https://github.com/kubernetes/kubernetes/pull/42090), [@timstclair](https://github.com/timstclair))
* Better compat with very old iptables (e.g. CentOS 6) ([#37594](https://github.com/kubernetes/kubernetes/pull/37594), [@thockin](https://github.com/thockin))

### Scheduler
* Add the support to the scheduler for spreading pods of StatefulSets. ([#41708](https://github.com/kubernetes/kubernetes/pull/41708), [@bsalamat](https://github.com/bsalamat))
* Added support to minimize sending verbose node information to scheduler extender by sending only node names and expecting extenders to cache the rest of the node information ([#41119](https://github.com/kubernetes/kubernetes/pull/41119), [@sarat-k](https://github.com/sarat-k))
* Support KUBE_MAX_PD_VOLS on Azure ([#41398](https://github.com/kubernetes/kubernetes/pull/41398), [@codablock](https://github.com/codablock))
* Mark multi-scheduler graduation to beta and then v1. ([#38871](https://github.com/kubernetes/kubernetes/pull/38871), [@k82cn](https://github.com/k82cn))
* Scheduler treats StatefulSet pods as belonging to a single equivalence class. ([#39718](https://github.com/kubernetes/kubernetes/pull/39718), [@foxish](https://github.com/foxish))
* Update FitError as a message component into the PodConditionUpdater. ([#39491](https://github.com/kubernetes/kubernetes/pull/39491), [@jayunit100](https://github.com/jayunit100))
* Fix comment and optimize code ([#38084](https://github.com/kubernetes/kubernetes/pull/38084), [@tanshanshan](https://github.com/tanshanshan))
* Add flag to enable contention profiling in scheduler. ([#37357](https://github.com/kubernetes/kubernetes/pull/37357), [@gmarek](https://github.com/gmarek))
* Try self-repair scheduler cache or panic ([#37379](https://github.com/kubernetes/kubernetes/pull/37379), [@wojtek-t](https://github.com/wojtek-t))

### Volume Plugins

#### Azure Disk
* restrict name length for Azure specifications ([#40030](https://github.com/kubernetes/kubernetes/pull/40030), [@colemickens](https://github.com/colemickens))

#### GlusterFS
* The glusterfs dynamic volume provisioner will now choose a unique GID for new persistent volumes from a range that can be configured in the storage class with the "gidMin" and "gidMax" parameters. The default range is 2000 - 2147483647 (max int32). ([#37886](https://github.com/kubernetes/kubernetes/pull/37886), [@obnoxxx](https://github.com/obnoxxx))

#### Photon
* Fix photon controller plugin to construct with correct PdID ([#37167](https://github.com/kubernetes/kubernetes/pull/37167), [@luomiao](https://github.com/luomiao))

#### rbd
* force unlock rbd image if the image is not used ([#41597](https://github.com/kubernetes/kubernetes/pull/41597), [@rootfs](https://github.com/rootfs))

#### vSphere
* Fix fsGroup to vSphere ([#38655](https://github.com/kubernetes/kubernetes/pull/38655), [@abrarshivani](https://github.com/abrarshivani))
* Fix issue when attempting to unmount a wrong vSphere volume ([#37413](https://github.com/kubernetes/kubernetes/pull/37413), [@BaluDontu](https://github.com/BaluDontu))

#### Other Notable Changes
* Implement bulk polling of volumes ([#41306](https://github.com/kubernetes/kubernetes/pull/41306), [@gnufied](https://github.com/gnufied))
* Check if pathExists before performing Unmount ([#39311](https://github.com/kubernetes/kubernetes/pull/39311), [@rkouj](https://github.com/rkouj))
* Unmount operation should not fail if volume is already unmounted ([#38547](https://github.com/kubernetes/kubernetes/pull/38547), [@rkouj](https://github.com/rkouj))
* Provide kubernetes-controller-manager flags to control volume attach/detach reconciler sync.  The duration of the syncs can be controlled, and the syncs can be shut off as well.  ([#39551](https://github.com/kubernetes/kubernetes/pull/39551), [@chrislovecnm](https://github.com/chrislovecnm))
* Fix unmountDevice issue caused by shared mount in GCI ([#38411](https://github.com/kubernetes/kubernetes/pull/38411), [@jingxu97](https://github.com/jingxu97))
* Fix permissions when using fsGroup ([#37009](https://github.com/kubernetes/kubernetes/pull/37009), [@sjenning](https://github.com/sjenning))
* Fixed issues [#39202](https://github.com/kubernetes/kubernetes/pull/39202), [#41041](https://github.com/kubernetes/kubernetes/pull/41041) and [#40941](https://github.com/kubernetes/kubernetes/pull/40941) that caused the iSCSI connections to be prematurely closed when deleting a pod with an iSCSI persistent volume attached and that prevented the use of newly created LUNs on targets with preestablished connections. ([#41196](https://github.com/kubernetes/kubernetes/pull/41196), [@CristianPop](https://github.com/CristianPop))

## Changes to Cluster Provisioning Scripts

### AWS
* Deployment of AWS Kubernetes clusters using the in-tree bash deployment (i.e. cluster/kube-up.sh or get-kube.sh) is obsolete. v1.5.x will be the last release to support cluster/kube-up.sh with AWS. For a list of viable alternatives, see: http://kubernetes.io/docs/getting-started-guides/aws/  ([#42196](https://github.com/kubernetes/kubernetes/pull/42196), [@zmerlynn](https://github.com/zmerlynn))
* Fix an issue where AWS tear-down leaks an DHCP Option Set. ([#38645](https://github.com/kubernetes/kubernetes/pull/38645), [@zmerlynn](https://github.com/zmerlynn))

### Juju
* The kubernetes-master, kubernetes-worker and kubeapi-load-balancer charms have gained an nrpe-external-master relation, allowing the integration of their monitoring in an external Nagios server. ([#41923](https://github.com/kubernetes/kubernetes/pull/41923), [@Cynerva](https://github.com/Cynerva))
* Disable anonymous auth on kubelet ([#41919](https://github.com/kubernetes/kubernetes/pull/41919), [@Cynerva](https://github.com/Cynerva))
* Fix shebangs in charm actions to use python3 ([#42058](https://github.com/kubernetes/kubernetes/pull/42058), [@Cynerva](https://github.com/Cynerva))
* K8s master charm now properly keeps distributed master files in sync for an HA control plane. ([#41351](https://github.com/kubernetes/kubernetes/pull/41351), [@chuckbutler](https://github.com/chuckbutler))
* Improve status messages ([#40691](https://github.com/kubernetes/kubernetes/pull/40691), [@Cynerva](https://github.com/Cynerva))
* Splits Juju Charm layers into master/worker roles ([#40324](https://github.com/kubernetes/kubernetes/pull/40324), [@chuckbutler](https://github.com/chuckbutler))
    * Adds support for 1.5.x series of Kubernetes
    * Introduces a tactic for keeping templates in sync with upstream eliminating template drift
    * Adds CNI support to the Juju Charms
    * Adds durable storage support to the Juju Charms
    * Introduces an e2e Charm layer for repeatable testing efforts and validation of clusters

### libvirt CoreOS
* To add local registry to libvirt_coreos ([#36751](https://github.com/kubernetes/kubernetes/pull/36751), [@sdminonne](https://github.com/sdminonne))

### GCE
* **the `gce` provider enables both RBAC authorization and the permissive legacy ABAC policy that makes all service accounts superusers. To opt out of the permissive ABAC policy, export the environment variable `ENABLE_LEGACY_ABAC=false` before running `cluster/kube-up.sh`. ([#43544](https://github.com/kubernetes/kubernetes/pull/43544), [@liggitt](https://github.com/liggitt))**
* **the `gce` provider ensures the bootstrap admin token user is included in the super-user group ([#39537](https://github.com/kubernetes/kubernetes/pull/39537), [@liggitt](https://github.com/liggitt))**
* Remove support for debian masters in GCE kube-up. ([#41666](https://github.com/kubernetes/kubernetes/pull/41666), [@mikedanese](https://github.com/mikedanese))
* Remove support for trusty in GCE kube-up. ([#41670](https://github.com/kubernetes/kubernetes/pull/41670), [@mikedanese](https://github.com/mikedanese))
* Don't fail if the grep fails to match any resources ([#41933](https://github.com/kubernetes/kubernetes/pull/41933), [@ixdy](https://github.com/ixdy))
* Fix the output of health-mointor.sh ([#41525](https://github.com/kubernetes/kubernetes/pull/41525), [@yujuhong](https://github.com/yujuhong))
* Added configurable etcd initial-cluster-state to kube-up script. ([#41332](https://github.com/kubernetes/kubernetes/pull/41332), [@jszczepkowski](https://github.com/jszczepkowski))
* The kube-apiserver [basic audit log](https://kubernetes.io/docs/admin/audit/) can be enabled in GCE by exporting the environment variable `ENABLE_APISERVER_BASIC_AUDIT=true` before running `cluster/kube-up.sh`. This will log to `/var/log/kube-apiserver-audit.log` and use the same `logrotate` settings as `/var/log/kube-apiserver.log`. ([#41211](https://github.com/kubernetes/kubernetes/pull/41211), [@enisoc](https://github.com/enisoc))
* On kube-up.sh clusters on GCE, kube-scheduler now contacts the API on the secured port. ([#41285](https://github.com/kubernetes/kubernetes/pull/41285), [@liggitt](https://github.com/liggitt))
* Use existing ABAC policy file when upgrading GCE cluster ([#40172](https://github.com/kubernetes/kubernetes/pull/40172), [@liggitt](https://github.com/liggitt))
* Ensure the GCI metadata files do not have newline at the end ([#38727](https://github.com/kubernetes/kubernetes/pull/38727), [@Amey-D](https://github.com/Amey-D))
* Fixed detection of master during creation of multizone nodes cluster by kube-up. ([#38617](https://github.com/kubernetes/kubernetes/pull/38617), [@jszczepkowski](https://github.com/jszczepkowski))
* Fixed validation of multizone cluster for GCE ([#38695](https://github.com/kubernetes/kubernetes/pull/38695), [@jszczepkowski](https://github.com/jszczepkowski))
* Fix GCI mounter issue ([#38124](https://github.com/kubernetes/kubernetes/pull/38124), [@jingxu97](https://github.com/jingxu97))
* Exit with error if <version number or publication> is not the final parameter. ([#37723](https://github.com/kubernetes/kubernetes/pull/37723), [@mtaufen](https://github.com/mtaufen))
* GCI: Remove /var/lib/docker/network ([#37593](https://github.com/kubernetes/kubernetes/pull/37593), [@yujuhong](https://github.com/yujuhong))
* Fix the equality checks for numeric values in cluster/gce/util.sh. ([#37638](https://github.com/kubernetes/kubernetes/pull/37638), [@roberthbailey](https://github.com/roberthbailey))
* Modify GCI mounter to enable NFSv3 ([#37582](https://github.com/kubernetes/kubernetes/pull/37582), [@jingxu97](https://github.com/jingxu97))
* Use gsed on the Mac ([#37562](https://github.com/kubernetes/kubernetes/pull/37562), [@roberthbailey](https://github.com/roberthbailey))
* Bump GCI 
* to gci-beta-56-9000-80-0 ([#41027](https://github.com/kubernetes/kubernetes/pull/41027), [@dchen1107](https://github.com/dchen1107))
* to gci-stable-56-9000-84-2 ([#41819](https://github.com/kubernetes/kubernetes/pull/41819), [@dchen1107](https://github.com/dchen1107))
* Bump GCE ContainerVM
  * to container-vm-v20161208 ([release notes](https://cloud.google.com/compute/docs/containers/container_vms#changelog)) ([#38432](https://github.com/kubernetes/kubernetes/pull/38432), [@timstclair](https://github.com/timstclair))
  * to container-vm-v20170201 to address CVE-2016-9962. ([#40828](https://github.com/kubernetes/kubernetes/pull/40828), [@zmerlynn](https://github.com/zmerlynn))
  * to container-vm-v20170117 to pick up CVE fixes in base image. ([#40094](https://github.com/kubernetes/kubernetes/pull/40094), [@zmerlynn](https://github.com/zmerlynn))
  * to container-vm-v20170214 to address CVE-2016-9962. ([#41449](https://github.com/kubernetes/kubernetes/pull/41449), [@zmerlynn](https://github.com/zmerlynn))

### OpenStack
* Do not daemonize `salt-minion` for the openstack-heat provider. ([#40722](https://github.com/kubernetes/kubernetes/pull/40722), [@micmro](https://github.com/micmro))
* OpenStack-Heat will now look for an image named "CentOS-7-x86_64-GenericCloud-1604". To restore the previous behavior set OPENSTACK_IMAGE_NAME="CentOS7" ([#40368](https://github.com/kubernetes/kubernetes/pull/40368), [@sc68cal](https://github.com/sc68cal))
* Fixes a bug in the OpenStack-Heat kubernetes provider, in the handling of differences between the Identity v2 and Identity v3 APIs ([#40105](https://github.com/kubernetes/kubernetes/pull/40105), [@sc68cal](https://github.com/sc68cal))

### Container Images
* Update gcr.io/google-containers/rescheduler to v0.2.2, which uses busybox as a base image instead of ubuntu. ([#41911](https://github.com/kubernetes/kubernetes/pull/41911), [@ixdy](https://github.com/ixdy))
* Remove unnecessary metrics (http/process/go) from being exposed by etcd-version-monitor ([#41807](https://github.com/kubernetes/kubernetes/pull/41807), [@shyamjvs](https://github.com/shyamjvs))
* Align the hyperkube image to support running binaries at /usr/local/bin/ like the other server images ([#41017](https://github.com/kubernetes/kubernetes/pull/41017), [@luxas](https://github.com/luxas))
* Bump up GLBC version from 0.9.0-beta to 0.9.1 ([#41037](https://github.com/kubernetes/kubernetes/pull/41037), [@bprashanth](https://github.com/bprashanth))

### Other Notable Changes
* The default client certificate generated by kube-up now contains the superuser `system:masters` group ([#39966](https://github.com/kubernetes/kubernetes/pull/39966), [@liggitt](https://github.com/liggitt))
* Added support for creating HA clusters for centos using kube-up.sh. ([#39462](https://github.com/kubernetes/kubernetes/pull/39462), [@Shawyeok](https://github.com/Shawyeok))
* Enable lazy inode table and journal initialization for ext3 and ext4 ([#38865](https://github.com/kubernetes/kubernetes/pull/38865), [@codablock](https://github.com/codablock))
* Since `kubernetes.tar.gz` no longer includes client or server binaries, `cluster/kube-{up,down,push}.sh` now automatically download released binaries if they are missing. ([#38730](https://github.com/kubernetes/kubernetes/pull/38730), [@ixdy](https://github.com/ixdy))
* fix broken cluster/centos and enhance the style ([#34002](https://github.com/kubernetes/kubernetes/pull/34002), [@xiaoping378](https://github.com/xiaoping378))
* Set kernel.softlockup_panic =1 based on the flag. ([#38001](https://github.com/kubernetes/kubernetes/pull/38001), [@dchen1107](https://github.com/dchen1107))
* configure local-up-cluster.sh to handle auth proxies ([#36838](https://github.com/kubernetes/kubernetes/pull/36838), [@deads2k](https://github.com/deads2k))
* `kube-up.sh`/`kube-down.sh` no longer force update gcloud for provider=gce|gke. ([#36292](https://github.com/kubernetes/kubernetes/pull/36292), [@jlowdermilk](https://github.com/jlowdermilk))
* Collect logs for dead kubelets too ([#37671](https://github.com/kubernetes/kubernetes/pull/37671), [@mtaufen](https://github.com/mtaufen))

## Changes to Addons

### Dashboard
* Update dashboard version to v1.6.0 ([#43210](https://github.com/kubernetes/kubernetes/pull/43210), [@floreks](https://github.com/floreks))

### DNS
* Updates the dnsmasq cache/mux layer to be managed by dnsmasq-nanny. ([#41826](https://github.com/kubernetes/kubernetes/pull/41826), [@bowei](https://github.com/bowei))
  dnsmasq-nanny manages dnsmasq based on values from the
  kube-system:kube-dns configmap:
  ```
  "stubDomains": {
  "acme.local": ["1.2.3.4"]
   },
  ```
  
  is a map of domain to list of nameservers for the domain. This is used
  to inject private DNS domains into the kube-dns namespace. In the above
  example, any DNS requests for `*.acme.local` will be served by the
  ```
  nameserver 1.2.3.4.
  ```
  
  ```
  upstreamNameservers": ["8.8.8.8", "8.8.4.4"]
  ```
  is a list of upstreamNameservers to use, overriding the configuration
  specified in `/etc/resolv.conf`.
* `kube-dns` now runs using a separate `system:serviceaccount:kube-system:kube-dns` service account which is automatically bound to the correct RBAC permissions. ([#38816](https://github.com/kubernetes/kubernetes/pull/38816), [@deads2k](https://github.com/deads2k))
* Use kube-dns:1.11.0 ([#39925](https://github.com/kubernetes/kubernetes/pull/39925), [@sadlil](https://github.com/sadlil))

### DNS Autoscaler
* Patch CVE-2016-8859 in gcr.io/google-containers/cluster-proportional-autoscaler-amd64 ([#42933](https://github.com/kubernetes/kubernetes/pull/42933), [@timstclair](https://github.com/timstclair))

### Cluster Autoscaler
* Allow the Horizontal Pod Autoscaler controller to talk to the metrics API and custom metrics API as standard APIs. ([#41824](https://github.com/kubernetes/kubernetes/pull/41824), [@DirectXMan12](https://github.com/DirectXMan12))

### Cluster Load Balancing
* Update defaultbackend image to 1.3 ([#42212](https://github.com/kubernetes/kubernetes/pull/42212), [@timstclair](https://github.com/timstclair))

### etcd Empty Dir Cleanup
* Base etcd-empty-dir-cleanup on busybox, run as nobody, and update to etcdctl 3.0.14 ([#41674](https://github.com/kubernetes/kubernetes/pull/41674), [@ixdy](https://github.com/ixdy))

### Fluentd
* Migrated fluentd addon to daemon set ([#32088](https://github.com/kubernetes/kubernetes/pull/32088), [@piosz](https://github.com/piosz))
* Fluentd was migrated to Daemon Set, which targets nodes with beta.kubernetes.io/fluentd-ds-ready=true label. If you use fluentd in your cluster please make sure that the nodes with version 1.6+ contains this label. ([#42931](https://github.com/kubernetes/kubernetes/pull/42931), [@piosz](https://github.com/piosz))
* Fluentd-gcp containers spawned by DaemonSet are now configured using ConfigMap ([#42126](https://github.com/kubernetes/kubernetes/pull/42126), [@crassirostris](https://github.com/crassirostris))
* Cleanup fluentd-gcp image: rebase on debian-base, switch to upstream packages, remove fluent-ui & rails ([#41998](https://github.com/kubernetes/kubernetes/pull/41998), [@timstclair](https://github.com/timstclair))
* On GCE, the apiserver audit log (`/var/log/kube-apiserver-audit.log`) will be sent through fluentd if enabled. It will go to the same place as `kube-apiserver.log`, but tagged as its own stream. ([#41360](https://github.com/kubernetes/kubernetes/pull/41360), [@enisoc](https://github.com/enisoc))
* If `experimentalCriticalPodAnnotation` feature gate is set to true, fluentd pods will not be evicted by the kubelet. ([#41035](https://github.com/kubernetes/kubernetes/pull/41035), [@vishh](https://github.com/vishh))
* fluentd config for GKE clusters updated: detect exceptions in container log streams and forward them as one log entry. ([#39656](https://github.com/kubernetes/kubernetes/pull/39656), [@thomasschickinger](https://github.com/thomasschickinger))
* Make fluentd pods critical ([#39146](https://github.com/kubernetes/kubernetes/pull/39146), [@crassirostris](https://github.com/crassirostris))
* Fluentd/Elastisearch add-on: correctly parse and index kubernetes labels ([#36857](https://github.com/kubernetes/kubernetes/pull/36857), [@Shrugs](https://github.com/Shrugs))

### Heapster
* Bumped Heapster to v1.3.0. ([#43298](https://github.com/kubernetes/kubernetes/pull/43298), [@piosz](https://github.com/piosz))
  * More details about the release https://github.com/kubernetes/heapster/releases/tag/v1.3.0

### Registry
* Use daemonset in docker registry add on ([#35582](https://github.com/kubernetes/kubernetes/pull/35582), [@surajssd](https://github.com/surajssd))
* contribute deis/registry-proxy as a replacement for kube-registry-proxy ([#35797](https://github.com/kubernetes/kubernetes/pull/35797), [@bacongobbler](https://github.com/bacongobbler))
