***if you can't edit this file please edit the [associated doc](https://docs.google.com/document/d/15iUswVe3gwJIR6JvFN7GSaYfd1fImblw261cZtInB-U/edit?usp=sharing) and comment on the [PR](https://github.com/kubernetes/features/pull/203) that changes have been made***

***automatically generated release notes [are at the bottom](#auto-generated-release-notes) please curate and move those notes into the correct SIG***

## API Machinery
### New Features
### Notable Changes
- The internal storage layer for kubernetes cluster state has been updated to use etcd v3 by default for new clusters.  Old clusters will have to plan for a data migration window.
### Breaking Changes

## AWS
### New Features
### Notable Changes
### Breaking Changes

## Apps
### New Features
- Introduce apps/v1beta1.Deployments resource with modified defaults compared to extensions/v1beta1.Deployments. ([#39683](https://github.com/kubernetes/kubernetes/pull/39683), [@soltysh](https://github.com/soltysh))
- Introduce new generator for apps/v1beta1 deployments ([#42362](https://github.com/kubernetes/kubernetes/pull/42362), [@soltysh](https://github.com/soltysh))
- Introduce the rolling update feature for DaemonSet. See [Performing a Rolling Update on a DaemonSet](https://deploy-preview-2878--kubernetes-io-master-staging.netlify.com/docs/tasks/manage-daemon/update-daemon-set/).
### Notable Changes
- Deployments that cannot make progress in rolling out the newest version will now indicate via the API they are blocked
- kubectl: respect deployment strategy parameters for rollout status ([#41809](https://github.com/kubernetes/kubernetes/pull/41809), [@kargakis](https://github.com/kargakis))
- kubectl logs allows getting logs directly from deployment, job and statefulset ([#40927](https://github.com/kubernetes/kubernetes/pull/40927), [@soltysh](https://github.com/soltysh))
- Deployment now fully respects ControllerRef to avoid fighting over Pods and ReplicaSets. At the time of upgrade, **you must not have Deployments with selectors that overlap**, or else [ownership of ReplicaSets may change](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/controller-ref.md#upgrading). ([#42175](https://github.com/kubernetes/kubernetes/pull/42175), [@enisoc](https://github.com/enisoc))
- StatefulSet now respects ControllerRef to avoid fighting over Pods. At the time of upgrade, **you must not have StatefulSets with selectors that overlap** with any other controllers (such as ReplicaSets), or else [ownership of Pods may change](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/controller-ref.md#upgrading). ([#42080](https://github.com/kubernetes/kubernetes/pull/42080), [@enisoc](https://github.com/enisoc))
- Add ready replicas in Deployments ([#37959](https://github.com/kubernetes/kubernetes/pull/37959), [@kargakis](https://github.com/kargakis))
- Add configurable limits to CronJob resource to specify how many successful and failed jobs are preserved. ([#40932](https://github.com/kubernetes/kubernetes/pull/40932), [@peay](https://github.com/peay))
- ReplicaSet has onwer ref of the Deployment that created it ([#35676](https://github.com/kubernetes/kubernetes/pull/35676), [@krmayankk](https://github.com/krmayankk))
### Breaking Changes
- Remove extensions/v1beta1 Jobs resource, and job/v1beta1 generator. ([#38614](https://github.com/kubernetes/kubernetes/pull/38614), [@soltysh](https://github.com/soltysh))
- 1.5 kubectl can't do `kubectl edit deployment` on 1.6 server ([#42392](https://github.com/kubernetes/kubernetes/issues/42392)). Current workaround is to either upgrade to 1.6 kubectl, or run `kubectl edit deployment.extensions` instead of `kubectl edit deployment`.

## Auth
### New Features
### Notable Changes
- RBAC API is promoted to v1beta1 (rbac.authorization.k8s.io/v1beta1), and defines default roles for control plane, node, and controller components.
### Breaking Changes

## Autoscaling
### New Features
### Notable Changes
- The Horizontal Pod Autoscaler now supports drawing metrics throug the API server aggregator.
- The Horizontal Pod Autoscaler now supports scaling on multiple, custom metrics.
### Breaking Changes

## Big Data
### New Features
### Notable Changes
### Breaking Changes

## CLI
### New Features
- Add new command "kubectl set selector" ([#38966](https://github.com/kubernetes/kubernetes/pull/38966), [@kargakis](https://github.com/kargakis))
- Add kubectl create poddisruptionbudget command ([#36646](https://github.com/kubernetes/kubernetes/pull/36646), [@kargakis](https://github.com/kargakis))
### Notable Changes
- `kubectl edit` now edits objects exactly as they were retrieved from the API. This allows using `kubectl edit` with third-party resources and extension API servers. Because client-side conversion is no longer done, the `--output-version` option is deprecated for `kubectl edit`. To edit using a particular API version, fully-qualify the resource, version, and group used to fetch the object (for example, `job.v1.batch/myjob`) ([#41304](https://github.com/kubernetes/kubernetes/pull/41304), [@liggitt](https://github.com/liggitt))
### Breaking Changes

## Cluster Lifecycle
### New Features
### Notable Changes
- New Bootstrap Token authentication and management method. Works well with kubeadm.
- Adds a new cloud-controller-manager binary that may be used for testing the new out-of-core cloudprovider flow
- Introduces an API for the Kubelet to request TLS certificates from the API server.
- kubeadm is enhanced and improved with a baseline feature set and command line flags that are now marked as beta.
### Breaking Changes

## Cluster Ops
### New Features
### Notable Changes
### Breaking Changes

## Contributor Experience
### New Features
### Notable Changes
### Breaking Changes

## Docs
### New Features
### Notable Changes
### Breaking Changes

## Federation
### New Features
### Notable Changes
- `kubefed` has graduated to beta: supports hosting federation on on-prem clusters, automatically configures `kube-dns` in joining clusters and allows passing arguments to federation components.
### Breaking Changes

## Instrumentation
### New Features
### Notable Changes
### Breaking Changes

## Network
### New Features
* Adds support for configurable DNS stub domains and upstream nameservers to kube-dns.
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
  
### Notable Changes
* An empty `kube-system:kube-dns` ConfigMap will be created for the cluster if one did not already exist.

### Breaking Changes

## Node
### New Features
### Notable Changes
- The Docker-CRI implementation is Beta and is enabled by default in kubelet.  You can disable it by --enable-cri=false. See [notes on the new implementation]( https://github.com/kubernetes/community/blob/master/contributors/devel/container-runtime-interface.md#kubernetes-v16-release-docker-cri-integration-beta) for more details.
- kubelet launches pods in a new cgroup hierarchy to better enforce quality of service.  Operators must drain all pods from their nodes prior to upgrade of the kubelet.  They must ensure the configured cgroup driver matches their associated container runtime cgroup driver.
### Breaking Changes

## On Prem
### New Features
### Notable Changes
### Breaking Changes

## OpenStack
### New Features
### Notable Changes
### Breaking Changes

## PM

[ Nothing for this release]

## Scalability
### New Features
### Notable Changes
### Breaking Changes

## Scheduling
### New Features

- **(Alpha feature)** Represent node problems "not ready" and "unreachable" using `NoExecute` taints. In combination with `tolerationSeconds` described below, this allows per-pod specification of how long to remain bound to a node that becomes unreachable or not ready, rather than using the default of 5 minutes. You can enable this alpha feature by including `TaintBasedEvictions=true` in `--feature-gates`, such as `--feature-gates=FooBar=true,TaintBasedEvictions=true`. Documentation is [here](https://kubernetes.io/docs/user-guide/node-selection/).

### Notable Changes

- The [multiple schedulers](https://kubernetes.io/docs/admin/multiple-schedulers/) feature is now in Beta. This feature allows you to run multiple schedulers in parallel, each responsible for different sets of pods. When using multiple schedulers, the scheduler name is now specified in a new-in-1.6 `schedulerName` field of the PodSpec rather than using the `scheduler.alpha.kubernetes.io/name` annotation on the Pod. When you upgrade to 1.6, the Kubernetes default scheduler will start using the `schedulerName` field of the PodSpec and will ignore the annotation. 

- [Node affinity/anti-affinity](https://kubernetes.io/docs/user-guide/node-selection/) and [pod affinity/anti-affinity](https://kubernetes.io/docs/user-guide/node-selection/) are now in Beta. Node affinity/anti-affinity allow you to specify rules for restricting which node(s) a pod can schedule onto, based on the labels on the node. Pod affinity/anti-affinity allow you to specify rules for spreading and packing pods relative to one another, across arbitrary topologies (node, zone, etc.) These affinity rules are now be specified in a new-in-1.6 `affinity` field of the PodSpec. Kubernetes 1.6 continues to support the alpha `scheduler.alpha.kubernetes.io/affinity` annotation on the Pod if you explicitly enable the alpha feature "AffinityInAnnotations", but it will be removed in a future release. When you upgrade to 1.6, if you have not enabled the alpha feature, then the scheduler will use the `affinity` field in PodSpec and will ignore the annotation. If you have enabled the alpha feature, then the scheduler will use the `affinity` field in PodSpec if it is present, and otherwise will use the annotation.

- [Taints and tolerations](https://kubernetes.io/docs/user-guide/node-selection/) are now in Beta. This feature allows you to specify rules for "repelling" pods from nodes by default, which enables use cases like dedicated nodes and reserving nodes with special features for pods that need those features. We've also added a `NoExecute` taint type that evicts already-running pods, and an associated `tolerationSeconds` field to tolerations to delay the eviction for a specified amount of time. As before, taints are created using `kubectl taint` (but internally they are now represented as a field `taints` in the NodeSpec rather than using the `scheduler.alpha.kubernetes.io/taints` annotation on Node). Tolerations are now specified in a new-in-1.6 `tolerations` field of the PodSpec rather than using the `scheduler.alpha.kubernetes.io/tolerations` annotation on the Pod. When you upgrade to 1.6, the scheduler will start using the fields and will ignore the annotations.

### Breaking Changes

The features described above are now specified using fields rather than annotations. As a result, you must take the following actions if you are already using any of the features (in their alpha form):

- **Multiple schedulers**
  - Modify your PodSpecs that currently use the `scheduler.alpha.kubernetes.io/name` annotation on Pod, to instead use the `schedulerName` field in the PodSpec.
  - Modify any custom scheduler(s) you have written so that they read the `schedulerName` field on Pod instead of the `scheduler.alpha.kubernetes.io/name` annotation. 
  - Note that you can only start using the `schedulerName` field **after** you upgrade to 1.6; it is not recognized in 1.5.

- **Node affinity/anti-affinity and pod affinity/anti-affinity**
  - You can continue to use the alpha version of this feature (with one caveat -- see below), in which you specify affinity requests using Pod annotations, in 1.6 by including `AffinityInAnnotations=true` in `--feature-gates`, such as `--feature-gates=FooBar=true,AffinityInAnnotations=true`. Otherwise, you must modify your PodSpecs that currently use the `scheduler.alpha.kubernetes.io/affinity` annotation on Pod, to instead use the `affinity` field in the PodSpec. Support for the annotation will be removed in a future release, so we encourage you to switch to using the field as soon as possible.
  - Caveat: The alpha version no longer supports, and the beta version does not support, the "empty `podAffinityTerm.namespaces` list means all namespaces" behavior. In both alpha and beta it now means "same namespace as the pod specifying this affinity rule."
  - Note that you can only start using the `affinity` field **after** you upgrade to 1.6; it is not recognized in 1.5.
  - The `--failure-domains` scheduler command line-argument is not supported in the beta vesion of the feature.

- **Taints**
  - You will need to use `kubectl taint` to re-create all of your taints after kubectl and the master are upgraded to 1.6. Between the time the master is upgraded to 1.6 and when you do this, your existing taints will have no effect. 
  - You can find out what taints you have in place on a node while you are still running Kubernetes 1.5 by doing `kubectl describe node <node name>`; the `Taints` section will show the taints you have in place. To see the taints that were created under 1.5 when you are running 1.6, do `kubectl get node <node name> -o yaml` and look for the "Annotation" section with the annotation key `scheduler.alpha.kubernetes.io/taints`
  - You can remove the "old" taints (stored internally as annotations) at any time after the upgrade by doing `kubectl annotate nodes <node name> scheduler.alpha.kubernetes.io/taints-`
(note the minus at the end, which means "delete the taint with this key")
  - Note that because any taints you might have created with Kubernetes 1.5 can only affect the scheduling of new pods (the `NoExecute` taint effect is introduced in 1.6), neither the master upgrade nor your running `kubectl taint` to re-create the taints will affect pods that are already running.

- **Tolerations**
  - After your master is upgraded to 1.6, you will need to update your PodSpecs to set the `tolerations` field of the PodSpec and remove the `scheduler.alpha.kubernetes.io/tolerations` annotation on the Pod. (It's not strictly necessary to remove the annotation, as it will have no effect once you upgrade to 1.6.) Between the time the master is upgraded to 1.6 and when you do this, tolerations attached to Pods that are created will have no effect. Pods that are already running will not be affected by the upgrade.
  - You can find the PodSpec tolerations that were created as annotations (if any) in a controller definition by doing `kubectl get <controller kind> <controller name> -o yaml` and looking for the "Annotation" section with the annotation key `scheduler.alpha.kubernetes.io/tolerations` (This will work whether you are running Kubernetes 1.5 or 1.6).
  - To update a controller's PodSpec to use the field instead of the annotation, use one of the kubectl commands that do update ("kubectl replace" or "kubectl apply" or "kubectl patch") or just delete the controller entirely and re-create it with a new pod template. Note that you will be able to do these things only after the upgrade.


## Service Catalog
- Adds a new API resource `PodPreset` and admission controller to enable defining cross-cutting injection of Volumes and Environment into Pods.
### New Features
### Notable Changes
### Breaking Changes

## Storage
### New Features
### Notable Changes
- A new volume driver capable of projecting secrets, configmaps, and downward API items into the same directory.
- Flex volume plugin is updated to support attach/detach interfaces. It broke backward compatibility. Please update your drivers and implement the new callouts.
- Added support for mount options in persistent volumes
- StorageClass API is promoted to v1 (storage.k8s.io/v1)
- Default storage classes are deployed during installation on Azure, AWS, GCE, OpenStack and vSphere
- Populate environment variables from a configmap or secret.
- Support for user-written/run dynamic PV provisioners. See github.com/kubernetes-incubator/external-storage for a golang library and examples
- ScaleIO Kubernetes Volume Plugin added enabling pods to seamlessly access and use data stored on ScaleIO volumes.
- Portworx Volume Plugin added capability to use [Portworx](http://www.portworx.com) as a storage provider for Kubernetes clusters. Portworx pools your servers capacity and turns your servers or cloud instances into converged, highly available compute and storage nodes.
- Add support to use NFSv3, NFSv4, and GlusterFS on GCI image cluster
### Breaking Changes

## Testing
### New Features
### Notable Changes
### Breaking Changes

## UI
### New Features
### Notable Changes
### Breaking Changes

## Windows
[Nothing for this release]

# Auto Generated Release Notes
## (please move information into the correct SIG)

**Release Note Preview - generated on Wed Mar 15 22:47:14 UTC 2017**

# Branch master

[Documentation](https://docs.k8s.io) & [Examples](https://releases.k8s.io/master/examples)

## Major Themes

* TBD

## Other notable improvements

* TBD

## Known Issues

* TBD

## Provider-specific Notes

* TBD

## Changelog since v1.5.0

### Action Required

* The --dns-provider argument of 'kubefed init' is now mandatory and does not default to `google-clouddns`. To initialize a Federation control plane with Google Cloud DNS, use the following invocation: 'kubefed init --dns-provider=google-clouddns' ([#42092](https://github.com/kubernetes/kubernetes/pull/42092), [@marun](https://github.com/marun))
* Change taints/tolerations to api fields ([#38957](https://github.com/kubernetes/kubernetes/pull/38957), [@aveshagarwal](https://github.com/aveshagarwal))
* Promote certificates.k8s.io to beta and enable it by default. Users using the alpha certificates API should delete v1alpha1 CSRs from the API before upgrading and recreate them as v1beta1 CSR after upgrading. ([#39772](https://github.com/kubernetes/kubernetes/pull/39772), [@mikedanese](https://github.com/mikedanese))
* include bootstrap admin in super-user group, ensure tokens file is correct on upgrades ([#39537](https://github.com/kubernetes/kubernetes/pull/39537), [@liggitt](https://github.com/liggitt))
* Switch default etcd version to 3.0.14. ([#36229](https://github.com/kubernetes/kubernetes/pull/36229), [@wojtek-t](https://github.com/wojtek-t))
    * Switch default storage backend flag in apiserver to `etcd3` mode.
* RBAC's special handling of the user "*" in RoleBinding and ClusterRoleBinding objects is deprecated and will be removed in v1beta1. To match all users, explicitly bind to the group "system:authenticated" and/or "system:unauthenticated". Existing v1alpha1 bindings to the user "*" will be automatically converted to the group "system:authenticated". ([#38981](https://github.com/kubernetes/kubernetes/pull/38981), [@liggitt](https://github.com/liggitt))
* The 'endpoints.beta.kubernetes.io/hostnames-map' annotation is no longer supported.  Users can use the 'Endpoints.subsets[].addresses[].hostname' field instead. ([#39284](https://github.com/kubernetes/kubernetes/pull/39284), [@bowei](https://github.com/bowei))
* `federation/deploy/deploy.sh` was an interim solution introduced in Kubernetes v1.4 to simplify the federation control plane deployment experience. Now that we have `kubefed`, we are deprecating `deploy.sh` scripts. ([#38902](https://github.com/kubernetes/kubernetes/pull/38902), [@madhusudancs](https://github.com/madhusudancs))
* Cluster federation servers have changed the location in etcd where federated services are stored, so existing federated services must be deleted and recreated. Before upgrading, export all federated services from the federation server and delete the services. After upgrading the cluster, recreate the federated services from the exported data. ([#37770](https://github.com/kubernetes/kubernetes/pull/37770), [@enj](https://github.com/enj))
* etcd2: watching from 0 returns all initial states as ADDED events ([#38079](https://github.com/kubernetes/kubernetes/pull/38079), [@hongchaodeng](https://github.com/hongchaodeng))
* Kubelet: The Docker-CRI implementation is not compatible with containers created by older Kubelets. It is recommended to drain your node before upgrade. If you choose to perform an in-place upgrade, the Kubelet will automatically restart all Kubernetes-managed containers on the node.
* Kubelet: The Docker-CRI implementation is not compatible with CNI plugins that do not conform to the [error handling behavior in the spec](https://github.com/containernetworking/cni/blob/master/SPEC.md#network-configuration-list-error-handling). The plugins are being updated to resolve this issue ([#43014](https://github.com/kubernetes/kubernetes/issues/43014)). You can disable CRI explicity (`--enable-cri=false`) as a temporary workaround.

### Other notable changes

* Add -p to mkdirs in gci-mounter function of gce configure.sh script ([#43134](https://github.com/kubernetes/kubernetes/pull/43134), [@shyamjvs](https://github.com/shyamjvs))
* Bumped rescheduler version to 0.3.0 ([#43106](https://github.com/kubernetes/kubernetes/pull/43106), [@piosz](https://github.com/piosz))
* kubeadm: `kubeadm reset` won't drain and remove the current node anymore ([#42713](https://github.com/kubernetes/kubernetes/pull/42713), [@luxas](https://github.com/luxas))
* hack/godep-restore.sh: use godep v79 which works ([#42965](https://github.com/kubernetes/kubernetes/pull/42965), [@sttts](https://github.com/sttts))
* Patch CVE-2016-8859 in gcr.io/google-containers/cluster-proportional-autoscaler-amd64 ([#42933](https://github.com/kubernetes/kubernetes/pull/42933), [@timstclair](https://github.com/timstclair))
* Disable devicemapper thin_ls due to excessive iops ([#42899](https://github.com/kubernetes/kubernetes/pull/42899), [@dashpole](https://github.com/dashpole))
* Use Prometheus instrumentation conventions ([#36704](https://github.com/kubernetes/kubernetes/pull/36704), [@fabxc](https://github.com/fabxc))
* Add new DaemonSet status fields to kubectl printer and describer.  ([#42843](https://github.com/kubernetes/kubernetes/pull/42843), [@janetkuo](https://github.com/janetkuo))
* Dropped the support for docker 1.9.x and the belows.  ([#42694](https://github.com/kubernetes/kubernetes/pull/42694), [@dchen1107](https://github.com/dchen1107))
* DaemonSet now respects ControllerRef to avoid fighting over Pods. ([#42173](https://github.com/kubernetes/kubernetes/pull/42173), [@enisoc](https://github.com/enisoc))
* restored normalization of custom `--etcd-prefix` when `--storage-backend` is set to etcd3 ([#42506](https://github.com/kubernetes/kubernetes/pull/42506), [@liggitt](https://github.com/liggitt))
* kubelet created cgroups follow lowercase naming conventions ([#42497](https://github.com/kubernetes/kubernetes/pull/42497), [@derekwaynecarr](https://github.com/derekwaynecarr))
* Support whitespace in command path for gcp auth plugin ([#41653](https://github.com/kubernetes/kubernetes/pull/41653), [@jlowdermilk](https://github.com/jlowdermilk))
* kubelet exports metrics for cgroup management ([#41988](https://github.com/kubernetes/kubernetes/pull/41988), [@sjenning](https://github.com/sjenning))
* Remove cmd/kube-discovery from the tree since it's not necessary anymore ([#42070](https://github.com/kubernetes/kubernetes/pull/42070), [@luxas](https://github.com/luxas))
* kubeadm: Hook up kubeadm against the BootstrapSigner ([#41417](https://github.com/kubernetes/kubernetes/pull/41417), [@luxas](https://github.com/luxas))
* Federated Ingress over GCE no longer requires separate firewall rules to be created for each cluster to circumvent flapping firewall health checks. ([#41942](https://github.com/kubernetes/kubernetes/pull/41942), [@csbell](https://github.com/csbell))
* ScaleIO Kubernetes Volume Plugin added enabling pods to seamlessly access and use data stored on ScaleIO volumes. ([#38924](https://github.com/kubernetes/kubernetes/pull/38924), [@vladimirvivien](https://github.com/vladimirvivien))
* Pods are launched in a separate cgroup hierarchy than system services. ([#42350](https://github.com/kubernetes/kubernetes/pull/42350), [@vishh](https://github.com/vishh))
* Experimental support to reserve a pod's memory request from being utilized by pods in lower QoS tiers. ([#41149](https://github.com/kubernetes/kubernetes/pull/41149), [@sjenning](https://github.com/sjenning))
* Juju: Disable anonymous auth on kubelet ([#41919](https://github.com/kubernetes/kubernetes/pull/41919), [@Cynerva](https://github.com/Cynerva))
* Remove support for debian masters in GCE kube-up. ([#41666](https://github.com/kubernetes/kubernetes/pull/41666), [@mikedanese](https://github.com/mikedanese))
* Implement bulk polling of volumes ([#41306](https://github.com/kubernetes/kubernetes/pull/41306), [@gnufied](https://github.com/gnufied))
* stop kubectl edit from updating the last-applied-configuration annotation when --save-config is unspecified or false. ([#41924](https://github.com/kubernetes/kubernetes/pull/41924), [@ymqytw](https://github.com/ymqytw))
* kubeadm: Rename some flags for beta UI and fixup some logic ([#42064](https://github.com/kubernetes/kubernetes/pull/42064), [@luxas](https://github.com/luxas))
* StorageClassName attribute has been added to PersistentVolume and PersistentVolumeClaim objects and should be used instead of annotation `volume.beta.kubernetes.io/storage-class`. The beta annotation is still working in this release, however it will be removed in a future release. ([#42128](https://github.com/kubernetes/kubernetes/pull/42128), [@jsafrane](https://github.com/jsafrane))
* Remove Azure kube-up as the Azure community has focused efforts elsewhere. ([#41672](https://github.com/kubernetes/kubernetes/pull/41672), [@mikedanese](https://github.com/mikedanese))
* Fluentd-gcp containers spawned by DaemonSet are now configured using ConfigMap ([#42126](https://github.com/kubernetes/kubernetes/pull/42126), [@crassirostris](https://github.com/crassirostris))
* Modified kubemark startup scripts to restore master on reboot ([#41980](https://github.com/kubernetes/kubernetes/pull/41980), [@shyamjvs](https://github.com/shyamjvs))
* Adds a new API resource `PodPreset` and admission controller to enable defining cross-cutting injection of Volumes and Environment into Pods. ([#41931](https://github.com/kubernetes/kubernetes/pull/41931), [@jessfraz](https://github.com/jessfraz))
* AWS cloud provider: allow to run the master with a different AWS account or even on a different cloud provider than the nodes. ([#39996](https://github.com/kubernetes/kubernetes/pull/39996), [@scheeles](https://github.com/scheeles))
* Update defaultbackend image to 1.3 ([#42212](https://github.com/kubernetes/kubernetes/pull/42212), [@timstclair](https://github.com/timstclair))
* Allow the Horizontal Pod Autoscaler controller to talk to the metrics API and custom metrics API as standard APIs. ([#41824](https://github.com/kubernetes/kubernetes/pull/41824), [@DirectXMan12](https://github.com/DirectXMan12))
* Implement support for mount options in PVs ([#41906](https://github.com/kubernetes/kubernetes/pull/41906), [@gnufied](https://github.com/gnufied))
* Add DNS suffix search list support in Windows kube-proxy. ([#41618](https://github.com/kubernetes/kubernetes/pull/41618), [@JiangtianLi](https://github.com/JiangtianLi))
* `--experimental-nvidia-gpus` flag is **replaced** by `Accelerators` alpha feature gate along with  support for multiple Nvidia GPUs.  ([#42116](https://github.com/kubernetes/kubernetes/pull/42116), [@vishh](https://github.com/vishh))
    * To use GPUs, pass `Accelerators=true` as part of `--feature-gates` flag.
    * Works only with Docker runtime.
* Clean up the kube-proxy container image by removing unnecessary packages and files. ([#42090](https://github.com/kubernetes/kubernetes/pull/42090), [@timstclair](https://github.com/timstclair))
* AWS: Support shared tag `kubernetes.io/cluster/<clusterid>` ([#41695](https://github.com/kubernetes/kubernetes/pull/41695), [@justinsb](https://github.com/justinsb))
* Insecure access to the API Server at localhost:8080 will be turned off in v1.6 when using kubeadm ([#42066](https://github.com/kubernetes/kubernetes/pull/42066), [@luxas](https://github.com/luxas))
* AWS: Do not consider master instance zones for dynamic volume creation ([#41702](https://github.com/kubernetes/kubernetes/pull/41702), [@justinsb](https://github.com/justinsb))
* Added foreground garbage collection: the owner object will not be deleted until all its dependents are deleted by the garbage collector. Please checkout the [user doc](https://kubernetes.io/docs/concepts/abstractions/controllers/garbage-collection/) for details. ([#38676](https://github.com/kubernetes/kubernetes/pull/38676), [@caesarxuchao](https://github.com/caesarxuchao))
    * deleteOptions.orphanDependents is going to be deprecated in 1.7. Please use deleteOptions.propagationPolicy instead.
* force unlock rbd image if the image is not used ([#41597](https://github.com/kubernetes/kubernetes/pull/41597), [@rootfs](https://github.com/rootfs))
* The kubernetes-master, kubernetes-worker and kubeapi-load-balancer charms have gained an nrpe-external-master relation, allowing the integration of their monitoring in an external Nagios server. ([#41923](https://github.com/kubernetes/kubernetes/pull/41923), [@Cynerva](https://github.com/Cynerva))
* make kubectl describe pod show tolerationSeconds ([#42162](https://github.com/kubernetes/kubernetes/pull/42162), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))
* Completed pods should not be hidden when requested by name via `kubectl get`. ([#42216](https://github.com/kubernetes/kubernetes/pull/42216), [@smarterclayton](https://github.com/smarterclayton))
* [Federation][Kubefed] Flag cleanup ([#41335](https://github.com/kubernetes/kubernetes/pull/41335), [@irfanurrehman](https://github.com/irfanurrehman))
* Add the support to the scheduler for spreading pods of StatefulSets. ([#41708](https://github.com/kubernetes/kubernetes/pull/41708), [@bsalamat](https://github.com/bsalamat))
* Portworx Volume Plugin added enabling [Portworx](http://www.portworx.com) to be used as a storage provider for Kubernetes clusters. Portworx pools your servers capacity and turns your servers or cloud instances into converged, highly available compute and storage nodes. ([#39535](https://github.com/kubernetes/kubernetes/pull/39535), [@adityadani](https://github.com/adityadani))
* Remove support for trusty in GCE kube-up. ([#41670](https://github.com/kubernetes/kubernetes/pull/41670), [@mikedanese](https://github.com/mikedanese))
* Import a natural sorting library and use it in the sorting printer. ([#40746](https://github.com/kubernetes/kubernetes/pull/40746), [@matthyx](https://github.com/matthyx))
* Parameter keys in a StorageClass `parameters` map may not use the `kubernetes.io` or `k8s.io` namespaces. ([#41837](https://github.com/kubernetes/kubernetes/pull/41837), [@liggitt](https://github.com/liggitt))
* Make DaemonSet respect critical pods annotation when scheduling.  ([#42028](https://github.com/kubernetes/kubernetes/pull/42028), [@janetkuo](https://github.com/janetkuo))
* New Kubelet flag `--enforce-node-allocatable` with a default value of `pods` is added which will make kubelet create a top level cgroup for all pods to enforce Node Allocatable. Optionally, `system-reserved` & `kube-reserved` values can also be specified separated by comma to enforce node allocatable on cgroups specified via `--system-reserved-cgroup` & `--kube-reserved-cgroup` respectively. Note the default value of the latter flags are "". ([#41234](https://github.com/kubernetes/kubernetes/pull/41234), [@vishh](https://github.com/vishh))
    * This feature requires a **Node Drain** prior to upgrade failing which pods will be restarted if possible or terminated if they have a `RestartNever` policy.
* Deployment of AWS Kubernetes clusters using the in-tree bash deployment (i.e. cluster/kube-up.sh or get-kube.sh) is obsolete. v1.5.x will be the last release to support cluster/kube-up.sh with AWS. For a list of viable alternatives, see: http://kubernetes.io/docs/getting-started-guides/aws/  ([#42196](https://github.com/kubernetes/kubernetes/pull/42196), [@zmerlynn](https://github.com/zmerlynn))
* make kubectl taint command respect effect NoExecute ([#42120](https://github.com/kubernetes/kubernetes/pull/42120), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))
* Flex volume plugin is updated to support attach/detach interfaces. It broke backward compatibility. Please update your drivers and implement the new callouts.  ([#41804](https://github.com/kubernetes/kubernetes/pull/41804), [@chakri-nelluri](https://github.com/chakri-nelluri))
* Implement the update feature for DaemonSet. ([#41116](https://github.com/kubernetes/kubernetes/pull/41116), [@lukaszo](https://github.com/lukaszo))
* [Federation] Create configmap for the cluster kube-dns when cluster joins and remove when it unjoins ([#39338](https://github.com/kubernetes/kubernetes/pull/39338), [@irfanurrehman](https://github.com/irfanurrehman))
* New GKE certificates controller. ([#41160](https://github.com/kubernetes/kubernetes/pull/41160), [@pipejakob](https://github.com/pipejakob))
* Juju: Fix shebangs in charm actions to use python3 ([#42058](https://github.com/kubernetes/kubernetes/pull/42058), [@Cynerva](https://github.com/Cynerva))
* Support kubectl apply set-last-applied command to update the applied-applied-configuration annotation ([#41694](https://github.com/kubernetes/kubernetes/pull/41694), [@shiywang](https://github.com/shiywang))
* On GCI by default logrotate is disabled for application containers in favor of rotation mechanism provided by docker logging driver. ([#40634](https://github.com/kubernetes/kubernetes/pull/40634), [@crassirostris](https://github.com/crassirostris))
* Cleanup fluentd-gcp image: rebase on debian-base, switch to upstream packages, remove fluent-ui & rails ([#41998](https://github.com/kubernetes/kubernetes/pull/41998), [@timstclair](https://github.com/timstclair))
* Updating apiserver to return http status code 202 for a delete request when the resource is not immediately deleted because of user requesting cascading deletion using DeleteOptions.OrphanDependents=false. ([#41165](https://github.com/kubernetes/kubernetes/pull/41165), [@nikhiljindal](https://github.com/nikhiljindal))
* [Federation][kubefed] Support configuring dns-provider ([#40528](https://github.com/kubernetes/kubernetes/pull/40528), [@shashidharatd](https://github.com/shashidharatd))
* Added support to minimize sending verbose node information to scheduler extender by sending only node names and expecting extenders to cache the rest of the node information ([#41119](https://github.com/kubernetes/kubernetes/pull/41119), [@sarat-k](https://github.com/sarat-k))
* Guaranteed admission for Critical Pods ([#40952](https://github.com/kubernetes/kubernetes/pull/40952), [@dashpole](https://github.com/dashpole))
* Switch to the `node-role.kubernetes.io/master` label for marking and tainting the master node in kubeadm ([#41835](https://github.com/kubernetes/kubernetes/pull/41835), [@luxas](https://github.com/luxas))
* Allow drain --force to remove pods whose managing resource is deleted. ([#41864](https://github.com/kubernetes/kubernetes/pull/41864), [@marun](https://github.com/marun))
* add kubectl can-i to see if you can perform an action ([#41077](https://github.com/kubernetes/kubernetes/pull/41077), [@deads2k](https://github.com/deads2k))
* enable DefaultTolerationSeconds admission controller by default ([#41815](https://github.com/kubernetes/kubernetes/pull/41815), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))
* Make DaemonSets survive taint-based evictions when nodes turn unreachable/notReady. ([#41896](https://github.com/kubernetes/kubernetes/pull/41896), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))
* Deprecate outofdisk-transition-frequency and low-diskspace-threshold-mb flags ([#41941](https://github.com/kubernetes/kubernetes/pull/41941), [@dashpole](https://github.com/dashpole))
* Add OWNERS for sample-apiserver in staging ([#42094](https://github.com/kubernetes/kubernetes/pull/42094), [@sttts](https://github.com/sttts))
* Update gcr.io/google-containers/rescheduler to v0.2.2, which uses busybox as a base image instead of ubuntu. ([#41911](https://github.com/kubernetes/kubernetes/pull/41911), [@ixdy](https://github.com/ixdy))
* Add storage.k8s.io/v1 API ([#40088](https://github.com/kubernetes/kubernetes/pull/40088), [@jsafrane](https://github.com/jsafrane))
* Juju - K8s master charm now properly keeps distributed master files in sync for an HA control plane. ([#41351](https://github.com/kubernetes/kubernetes/pull/41351), [@chuckbutler](https://github.com/chuckbutler))
* Fix zsh completion: unknown file attribute error ([#38104](https://github.com/kubernetes/kubernetes/pull/38104), [@elipapa](https://github.com/elipapa))
* kubelet config should ignore file start with dots ([#39196](https://github.com/kubernetes/kubernetes/pull/39196), [@resouer](https://github.com/resouer))
* Add an alpha feature that makes NodeController set Taints instead of deleting Pods from not Ready Nodes. ([#41133](https://github.com/kubernetes/kubernetes/pull/41133), [@gmarek](https://github.com/gmarek))
* Base etcd-empty-dir-cleanup on busybox, run as nobody, and update to etcdctl 3.0.14 ([#41674](https://github.com/kubernetes/kubernetes/pull/41674), [@ixdy](https://github.com/ixdy))
* Fix zone placement heuristics so that multiple mounts in a StatefulSet pod are created in the same zone ([#40910](https://github.com/kubernetes/kubernetes/pull/40910), [@justinsb](https://github.com/justinsb))
* Flag --use-kubernetes-version for kubeadm init renamed to --kubernetes-version ([#41820](https://github.com/kubernetes/kubernetes/pull/41820), [@kad](https://github.com/kad))
* `kube-dns` now runs using a separate `system:serviceaccount:kube-system:kube-dns` service account which is automatically bound to the correct RBAC permissions. ([#38816](https://github.com/kubernetes/kubernetes/pull/38816), [@deads2k](https://github.com/deads2k))
* [Kubemark] Fixed hollow-npd container command to log to file ([#41858](https://github.com/kubernetes/kubernetes/pull/41858), [@shyamjvs](https://github.com/shyamjvs))
* kubeadm: Remove the --cloud-provider flag for beta init UX ([#41710](https://github.com/kubernetes/kubernetes/pull/41710), [@luxas](https://github.com/luxas))
* The CertificateSigningRequest API added the `extra` field to persist all information about the requesting user. This mirrors the fields in the SubjectAccessReview API used to check authorization. ([#41755](https://github.com/kubernetes/kubernetes/pull/41755), [@liggitt](https://github.com/liggitt))
* Upgrade golang versions to 1.7.5 ([#41771](https://github.com/kubernetes/kubernetes/pull/41771), [@cblecker](https://github.com/cblecker))
* Added a new secret type "bootstrap.kubernetes.io/token" for dynamically creating TLS bootstrapping bearer tokens. ([#41281](https://github.com/kubernetes/kubernetes/pull/41281), [@ericchiang](https://github.com/ericchiang))
* Remove unnecessary metrics (http/process/go) from being exposed by etcd-version-monitor ([#41807](https://github.com/kubernetes/kubernetes/pull/41807), [@shyamjvs](https://github.com/shyamjvs))
* Added `kubectl create clusterrole` command. ([#41538](https://github.com/kubernetes/kubernetes/pull/41538), [@xingzhou](https://github.com/xingzhou))
* Support new kubectl apply view-last-applied command for viewing the last configuration file applied ([#41146](https://github.com/kubernetes/kubernetes/pull/41146), [@shiywang](https://github.com/shiywang))
* Bump GCI to gci-stable-56-9000-84-2 ([#41819](https://github.com/kubernetes/kubernetes/pull/41819), [@dchen1107](https://github.com/dchen1107))
* list-resources: don't fail if the grep fails to match any resources ([#41933](https://github.com/kubernetes/kubernetes/pull/41933), [@ixdy](https://github.com/ixdy))
* client-go no longer imports GCP OAuth2 and OpenID Connect packages by default. ([#41532](https://github.com/kubernetes/kubernetes/pull/41532), [@ericchiang](https://github.com/ericchiang))
* Each pod has its own associated cgroup by default. ([#41349](https://github.com/kubernetes/kubernetes/pull/41349), [@derekwaynecarr](https://github.com/derekwaynecarr))
* Whitelist kubemark in node_ssh_supported_providers for log dump ([#41800](https://github.com/kubernetes/kubernetes/pull/41800), [@shyamjvs](https://github.com/shyamjvs))
* Support KUBE_MAX_PD_VOLS on Azure ([#41398](https://github.com/kubernetes/kubernetes/pull/41398), [@codablock](https://github.com/codablock))
* Projected volume plugin ([#37237](https://github.com/kubernetes/kubernetes/pull/37237), [@jpeeler](https://github.com/jpeeler))
* `--output-version` is ignored for all commands except `kubectl convert`.  This is consistent with the generic nature of `kubectl` CRUD commands and the previous removal of `--api-version`.  Specific versions can be specified in the resource field: `resource.version.group`, `jobs.v1.batch`. ([#41576](https://github.com/kubernetes/kubernetes/pull/41576), [@deads2k](https://github.com/deads2k))
* Added bool type support for jsonpath. ([#39063](https://github.com/kubernetes/kubernetes/pull/39063), [@xingzhou](https://github.com/xingzhou))
* Nodes can now report two additional address types in their status: InternalDNS and ExternalDNS. The apiserver can use `--kubelet-preferred-address-types` to give priority to the type of address it uses to reach nodes. ([#34259](https://github.com/kubernetes/kubernetes/pull/34259), [@liggitt](https://github.com/liggitt))
* Clients now use the `?watch=true` parameter to make watch API calls, instead of the `/watch/` path prefix ([#41722](https://github.com/kubernetes/kubernetes/pull/41722), [@liggitt](https://github.com/liggitt))
* ResourceQuota ability to support default limited resources ([#36765](https://github.com/kubernetes/kubernetes/pull/36765), [@derekwaynecarr](https://github.com/derekwaynecarr))
* Fix kubemark default e2e test suite's name ([#41751](https://github.com/kubernetes/kubernetes/pull/41751), [@shyamjvs](https://github.com/shyamjvs))
* federation aws: add logging of route53 calls ([#39964](https://github.com/kubernetes/kubernetes/pull/39964), [@justinsb](https://github.com/justinsb))
* Fix ConfigMap for Windows Containers. ([#39373](https://github.com/kubernetes/kubernetes/pull/39373), [@jbhurat](https://github.com/jbhurat))
* add defaultTolerationSeconds admission controller ([#41414](https://github.com/kubernetes/kubernetes/pull/41414), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))
* Node Problem Detector is beta now. New features added: journald support, standalone mode and arbitrary system log monitoring. ([#40206](https://github.com/kubernetes/kubernetes/pull/40206), [@Random-Liu](https://github.com/Random-Liu))
* services of type loadbalancer consume both loadbalancer and nodeport quota. ([#39364](https://github.com/kubernetes/kubernetes/pull/39364), [@zhouhaibing089](https://github.com/zhouhaibing089))
* Fix the output of health-mointor.sh ([#41525](https://github.com/kubernetes/kubernetes/pull/41525), [@yujuhong](https://github.com/yujuhong))
* kubectl describe no longer prints the last-applied-configuration annotation for secrets. ([#34664](https://github.com/kubernetes/kubernetes/pull/34664), [@ymqytw](https://github.com/ymqytw))
* Report node not ready on failed PLEG health check ([#41569](https://github.com/kubernetes/kubernetes/pull/41569), [@yujuhong](https://github.com/yujuhong))
* Delay Deletion of a Pod until volumes are cleaned up ([#41456](https://github.com/kubernetes/kubernetes/pull/41456), [@dashpole](https://github.com/dashpole))
* Alpha version of dynamic volume provisioning is removed in this release. Annotation ([#40000](https://github.com/kubernetes/kubernetes/pull/40000), [@jsafrane](https://github.com/jsafrane))
    * "volume.alpha.kubernetes.io/storage-class" does not have any special meaning. A default storage class
    * and  DefaultStorageClass admission plugin can be used to preserve similar behavior of Kubernetes cluster,
    * see https://kubernetes.io/docs/user-guide/persistent-volumes/#class-1 for details.
* An `automountServiceAccountToken *bool` field was added to ServiceAccount and PodSpec objects. If set to `false` on a pod spec, no service account token is automounted in the pod. If set to `false` on a service account, no service account token is automounted for that service account unless explicitly overridden in the pod spec. ([#37953](https://github.com/kubernetes/kubernetes/pull/37953), [@liggitt](https://github.com/liggitt))
* Bump addon-manager version to v6.4-alpha.1 in kubemark ([#41506](https://github.com/kubernetes/kubernetes/pull/41506), [@shyamjvs](https://github.com/shyamjvs))
* Do not daemonize `salt-minion` for the openstack-heat provider. ([#40722](https://github.com/kubernetes/kubernetes/pull/40722), [@micmro](https://github.com/micmro))
* Move private key parsing from serviceaccount/jwt.go to client-go/util/cert ([#40907](https://github.com/kubernetes/kubernetes/pull/40907), [@cblecker](https://github.com/cblecker))
* Added configurable etcd initial-cluster-state to kube-up script. ([#41332](https://github.com/kubernetes/kubernetes/pull/41332), [@jszczepkowski](https://github.com/jszczepkowski))
* Fix AWS device allocator to only use valid device names ([#41455](https://github.com/kubernetes/kubernetes/pull/41455), [@gnufied](https://github.com/gnufied))
* [Federation][Kubefed] Bug fix relating kubeconfig path in kubefed init ([#41410](https://github.com/kubernetes/kubernetes/pull/41410), [@irfanurrehman](https://github.com/irfanurrehman))
* On GCE, the apiserver audit log (`/var/log/kube-apiserver-audit.log`) will be sent through fluentd if enabled. It will go to the same place as `kube-apiserver.log`, but tagged as its own stream. ([#41360](https://github.com/kubernetes/kubernetes/pull/41360), [@enisoc](https://github.com/enisoc))
* Bump GCE ContainerVM to container-vm-v20170214 to address CVE-2016-9962. ([#41449](https://github.com/kubernetes/kubernetes/pull/41449), [@zmerlynn](https://github.com/zmerlynn))
* Fixed issues [#39202](https://github.com/kubernetes/kubernetes/pull/39202), [#41041](https://github.com/kubernetes/kubernetes/pull/41041) and [#40941](https://github.com/kubernetes/kubernetes/pull/40941) that caused the iSCSI connections to be prematurely closed when deleting a pod with an iSCSI persistent volume attached and that prevented the use of newly created LUNs on targets with preestablished connections. ([#41196](https://github.com/kubernetes/kubernetes/pull/41196), [@CristianPop](https://github.com/CristianPop))
* The kube-apiserver [basic audit log](https://kubernetes.io/docs/admin/audit/) can be enabled in GCE by exporting the environment variable `ENABLE_APISERVER_BASIC_AUDIT=true` before running `cluster/kube-up.sh`. This will log to `/var/log/kube-apiserver-audit.log` and use the same `logrotate` settings as `/var/log/kube-apiserver.log`. ([#41211](https://github.com/kubernetes/kubernetes/pull/41211), [@enisoc](https://github.com/enisoc))
* On kube-up.sh clusters on GCE, kube-scheduler now contacts the API on the secured port. ([#41285](https://github.com/kubernetes/kubernetes/pull/41285), [@liggitt](https://github.com/liggitt))
* Default RBAC ClusterRole and ClusterRoleBinding objects are automatically updated at server start to add missing permissions and subjects (extra permissions and subjects are left in place). To prevent autoupdating a particular role or rolebinding, annotate it with `rbac.authorization.kubernetes.io/autoupdate=false`. ([#41155](https://github.com/kubernetes/kubernetes/pull/41155), [@liggitt](https://github.com/liggitt))
* Make EnableCRI default to true ([#41378](https://github.com/kubernetes/kubernetes/pull/41378), [@yujuhong](https://github.com/yujuhong))
* We change the default attach_detach_controller sync period to 1 minute to reduce the query frequency through cloud provider to check whether volumes are attached or not.  ([#41363](https://github.com/kubernetes/kubernetes/pull/41363), [@jingxu97](https://github.com/jingxu97))
* RBAC `v1beta1` RoleBinding/ClusterRoleBinding subjects changed `apiVersion` to `apiGroup` to fully-qualify a subject. ServiceAccount subjects default to an apiGroup of `""`, User and Group subjects default to an apiGroup of `"rbac.authorization.k8s.io"`. ([#41184](https://github.com/kubernetes/kubernetes/pull/41184), [@liggitt](https://github.com/liggitt))
* Add support for finalizers in federated configmaps (deletes configmaps from underlying clusters). ([#40464](https://github.com/kubernetes/kubernetes/pull/40464), [@csbell](https://github.com/csbell))
* Make DaemonSet controller respect node taints and pod tolerations.  ([#41172](https://github.com/kubernetes/kubernetes/pull/41172), [@janetkuo](https://github.com/janetkuo))
* Added kubectl create role command ([#39852](https://github.com/kubernetes/kubernetes/pull/39852), [@xingzhou](https://github.com/xingzhou))
* If `experimentalCriticalPodAnnotation` feature gate is set to true, fluentd pods will not be evicted by the kubelet. ([#41035](https://github.com/kubernetes/kubernetes/pull/41035), [@vishh](https://github.com/vishh))
* Align the hyperkube image to support running binaries at /usr/local/bin/ like the other server images ([#41017](https://github.com/kubernetes/kubernetes/pull/41017), [@luxas](https://github.com/luxas))
* Native support for token based bootstrap flow.  This includes signing a well known ConfigMap in the `kube-public` namespace and cleaning out expired tokens. ([#36101](https://github.com/kubernetes/kubernetes/pull/36101), [@jbeda](https://github.com/jbeda))
* Reverts to looking up the current VM in vSphere using the machine's UUID, either obtained via sysfs or via the `vm-uuid` parameter in the cloud configuration file. ([#40892](https://github.com/kubernetes/kubernetes/pull/40892), [@robdaemon](https://github.com/robdaemon))
* This PR adds a manager to NodeController that is responsible for removing Pods from Nodes tainted with NoExecute Taints. This feature is beta (as the rest of taints) and enabled by default. It's gated by controller-manager enable-taint-manager flag. ([#40355](https://github.com/kubernetes/kubernetes/pull/40355), [@gmarek](https://github.com/gmarek))
* The authentication.k8s.io API group was promoted to v1 ([#41058](https://github.com/kubernetes/kubernetes/pull/41058), [@liggitt](https://github.com/liggitt))
* Fixes issue [#38418](https://github.com/kubernetes/kubernetes/pull/38418) which, under circumstance, could cause StatefulSet to deadlock.  ([#40838](https://github.com/kubernetes/kubernetes/pull/40838), [@kow3ns](https://github.com/kow3ns))
    * Mediates issue [#36859](https://github.com/kubernetes/kubernetes/pull/36859). StatefulSet only acts on Pods whose identity matches the StatefulSet, providing a partial mediation for overlapping controllers.
* Introduces an new alpha version of the Horizontal Pod Autoscaler including expanded support for specifying metrics. ([#36033](https://github.com/kubernetes/kubernetes/pull/36033), [@DirectXMan12](https://github.com/DirectXMan12))
* Set all node conditions to Unknown when node is unreachable ([#36592](https://github.com/kubernetes/kubernetes/pull/36592), [@andrewsykim](https://github.com/andrewsykim))
* [Federation] Add override flags options to kubefed init ([#40917](https://github.com/kubernetes/kubernetes/pull/40917), [@irfanurrehman](https://github.com/irfanurrehman))
* Fix for detach volume when node is not present/ powered off ([#40118](https://github.com/kubernetes/kubernetes/pull/40118), [@BaluDontu](https://github.com/BaluDontu))
* Bump up GLBC version from 0.9.0-beta to 0.9.1 ([#41037](https://github.com/kubernetes/kubernetes/pull/41037), [@bprashanth](https://github.com/bprashanth))
* The deprecated flags --config, --auth-path, --resource-container, and --system-container were removed. ([#40048](https://github.com/kubernetes/kubernetes/pull/40048), [@mtaufen](https://github.com/mtaufen))
* Add `kubectl attach` support for multiple types ([#40365](https://github.com/kubernetes/kubernetes/pull/40365), [@shiywang](https://github.com/shiywang))
* [Kubelet] Delay deletion of pod from the API server until volumes are deleted ([#41095](https://github.com/kubernetes/kubernetes/pull/41095), [@dashpole](https://github.com/dashpole))
* remove the create-external-load-balancer flag in cmd/expose.go ([#38183](https://github.com/kubernetes/kubernetes/pull/38183), [@tianshapjq](https://github.com/tianshapjq))
* The authorization.k8s.io API group was promoted to v1 ([#40709](https://github.com/kubernetes/kubernetes/pull/40709), [@liggitt](https://github.com/liggitt))
* Bump GCI to gci-beta-56-9000-80-0 ([#41027](https://github.com/kubernetes/kubernetes/pull/41027), [@dchen1107](https://github.com/dchen1107))
* [Federation][kubefed] Add option to expose federation apiserver on nodeport service ([#40516](https://github.com/kubernetes/kubernetes/pull/40516), [@shashidharatd](https://github.com/shashidharatd))
* Rename --experiemental-cgroups-per-qos to --cgroups-per-qos ([#39972](https://github.com/kubernetes/kubernetes/pull/39972), [@derekwaynecarr](https://github.com/derekwaynecarr))
* PV E2E: provide each spec with a fresh nfs host ([#40879](https://github.com/kubernetes/kubernetes/pull/40879), [@copejon](https://github.com/copejon))
* Remove the temporary fix for pre-1.0 mirror pods ([#40877](https://github.com/kubernetes/kubernetes/pull/40877), [@yujuhong](https://github.com/yujuhong))
* fix --save-config in create subcommand ([#40289](https://github.com/kubernetes/kubernetes/pull/40289), [@xilabao](https://github.com/xilabao))
* We should mention the caveats of in-place upgrade in release note. ([#40727](https://github.com/kubernetes/kubernetes/pull/40727), [@Random-Liu](https://github.com/Random-Liu))
* The SubjectAccessReview API passes subresource and resource name information to the authorizer to answer authorization queries. ([#40935](https://github.com/kubernetes/kubernetes/pull/40935), [@liggitt](https://github.com/liggitt))
* When feature gate "ExperimentalCriticalPodAnnotation" is set, Kubelet will avoid evicting pods in "kube-system" namespace that contains a special annotation - `scheduler.alpha.kubernetes.io/critical-pod` ([#40655](https://github.com/kubernetes/kubernetes/pull/40655), [@vishh](https://github.com/vishh))
    * This feature should be used in conjunction with the rescheduler to guarantee availability for critical system pods - https://kubernetes.io/docs/admin/rescheduler/
* make tolerations respect wildcard key ([#39914](https://github.com/kubernetes/kubernetes/pull/39914), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))
* [Federation][kubefed] Add option to disable persistence storage for etcd ([#40862](https://github.com/kubernetes/kubernetes/pull/40862), [@shashidharatd](https://github.com/shashidharatd))
* Init containers have graduated to GA and now appear as a field.  The beta annotation value will still be respected and overrides the field value. ([#38382](https://github.com/kubernetes/kubernetes/pull/38382), [@hodovska](https://github.com/hodovska))
* apply falls back to generic 3-way JSON merge patch if no go struct is registered for the target GVK ([#40666](https://github.com/kubernetes/kubernetes/pull/40666), [@ymqytw](https://github.com/ymqytw))
* HorizontalPodAutoscaler is no longer supported in extensions/v1beta1 version. Use autoscaling/v1 instead. ([#35782](https://github.com/kubernetes/kubernetes/pull/35782), [@piosz](https://github.com/piosz))
* Port forwarding can forward over websockets or SPDY. ([#33684](https://github.com/kubernetes/kubernetes/pull/33684), [@fraenkel](https://github.com/fraenkel))
* Improve kubectl describe node output by adding closing paren ([#39217](https://github.com/kubernetes/kubernetes/pull/39217), [@luksa](https://github.com/luksa))
* Bump GCE ContainerVM to container-vm-v20170201 to address CVE-2016-9962. ([#40828](https://github.com/kubernetes/kubernetes/pull/40828), [@zmerlynn](https://github.com/zmerlynn))
* Use full package path for definition name in OpenAPI spec ([#40124](https://github.com/kubernetes/kubernetes/pull/40124), [@mbohlool](https://github.com/mbohlool))
* kubectl apply now supports explicitly clearing values not present in the config by setting them to null ([#40630](https://github.com/kubernetes/kubernetes/pull/40630), [@liggitt](https://github.com/liggitt))
* Fixed an issue where 'kubectl get --sort-by=' would return an error when the specified field were not present in at least one of the returned objects, even that being a valid field in the object model. ([#40541](https://github.com/kubernetes/kubernetes/pull/40541), [@fabianofranz](https://github.com/fabianofranz))
* Add initial french translations for kubectl ([#40645](https://github.com/kubernetes/kubernetes/pull/40645), [@brendandburns](https://github.com/brendandburns))
* OpenStack-Heat will now look for an image named "CentOS-7-x86_64-GenericCloud-1604". To restore the previous behavior set OPENSTACK_IMAGE_NAME="CentOS7" ([#40368](https://github.com/kubernetes/kubernetes/pull/40368), [@sc68cal](https://github.com/sc68cal))
* Preventing nil pointer reference in client_config ([#40508](https://github.com/kubernetes/kubernetes/pull/40508), [@vjsamuel](https://github.com/vjsamuel))
* The bash AWS deployment via kube-up.sh has been deprecated. See http://kubernetes.io/docs/getting-started-guides/aws/ for alternatives. ([#38772](https://github.com/kubernetes/kubernetes/pull/38772), [@zmerlynn](https://github.com/zmerlynn))
* Fix failing load balancers in Azure ([#40405](https://github.com/kubernetes/kubernetes/pull/40405), [@codablock](https://github.com/codablock))
* kubefed init creates a service account for federation controller manager in the federation-system namespace and binds that service account to the federation-system:federation-controller-manager role that has read and list access on secrets in the federation-system namespace.  ([#40392](https://github.com/kubernetes/kubernetes/pull/40392), [@madhusudancs](https://github.com/madhusudancs))
* Fixed an SELinux issue in kubeadm on Docker 1.12+ by moving etcd SELinux options from container to pod. ([#40682](https://github.com/kubernetes/kubernetes/pull/40682), [@dgoodwin](https://github.com/dgoodwin))
* Juju kubernetes-master charm: improve status messages ([#40691](https://github.com/kubernetes/kubernetes/pull/40691), [@Cynerva](https://github.com/Cynerva))
* genericapiserver: cut off more dependencies – episode 3 ([#40426](https://github.com/kubernetes/kubernetes/pull/40426), [@sttts](https://github.com/sttts))
* Adding vmdk file extension for vmDiskPath in vsphere DeleteVolume ([#40538](https://github.com/kubernetes/kubernetes/pull/40538), [@divyenpatel](https://github.com/divyenpatel))
* Remove outdated net.experimental.kubernetes.io/proxy-mode and net.beta.kubernetes.io/proxy-mode annotations from kube-proxy. ([#40585](https://github.com/kubernetes/kubernetes/pull/40585), [@cblecker](https://github.com/cblecker))
* Improve the ARM builds and make hyperkube on ARM working again by upgrading the Go version for ARM to go1.8beta2 ([#38926](https://github.com/kubernetes/kubernetes/pull/38926), [@luxas](https://github.com/luxas))
* Prevent hotloops on error conditions, which could fill up the disk faster than log rotation can free space. ([#40497](https://github.com/kubernetes/kubernetes/pull/40497), [@lavalamp](https://github.com/lavalamp))
* DaemonSet controller actively kills failed pods (to recreate them) ([#40330](https://github.com/kubernetes/kubernetes/pull/40330), [@janetkuo](https://github.com/janetkuo))
* forgiveness alpha version api definition ([#39469](https://github.com/kubernetes/kubernetes/pull/39469), [@kevin-wangzefeng](https://github.com/kevin-wangzefeng))
* Bump up glbc version to 0.9.0-beta.1 ([#40565](https://github.com/kubernetes/kubernetes/pull/40565), [@bprashanth](https://github.com/bprashanth))
* Improve formatting of EventSource in kubectl get and kubectl describe ([#40073](https://github.com/kubernetes/kubernetes/pull/40073), [@matthyx](https://github.com/matthyx))
* CRI: use more gogoprotobuf plugins ([#40397](https://github.com/kubernetes/kubernetes/pull/40397), [@yujuhong](https://github.com/yujuhong))
* Adds `shortNames` to the `APIResource` from discovery which is a list of recommended shortNames for clients like `kubectl`. ([#40312](https://github.com/kubernetes/kubernetes/pull/40312), [@p0lyn0mial](https://github.com/p0lyn0mial))
* Use existing ABAC policy file when upgrading GCE cluster ([#40172](https://github.com/kubernetes/kubernetes/pull/40172), [@liggitt](https://github.com/liggitt))
* Added support for creating HA clusters for centos using kube-up.sh. ([#39462](https://github.com/kubernetes/kubernetes/pull/39462), [@Shawyeok](https://github.com/Shawyeok))
* azure: fix Azure Container Registry integration ([#40142](https://github.com/kubernetes/kubernetes/pull/40142), [@colemickens](https://github.com/colemickens))
* - Splits Juju Charm layers into master/worker roles ([#40324](https://github.com/kubernetes/kubernetes/pull/40324), [@chuckbutler](https://github.com/chuckbutler))
    * - Adds support for 1.5.x series of Kubernetes
    * - Introduces a tactic for keeping templates in sync with upstream eliminating template drift
    * - Adds CNI support to the Juju Charms
    * - Adds durable storage support to the Juju Charms
    * - Introduces an e2e Charm layer for repeatable testing efforts and validation of clusters
* Move b.gcr.io/k8s_authenticated_test to gcr.io/k8s-authenticated-test ([#40335](https://github.com/kubernetes/kubernetes/pull/40335), [@zmerlynn](https://github.com/zmerlynn))
* genericapiserver: more dependency cutoffs ([#40216](https://github.com/kubernetes/kubernetes/pull/40216), [@sttts](https://github.com/sttts))
* AWS: trust region if found from AWS metadata ([#38880](https://github.com/kubernetes/kubernetes/pull/38880), [@justinsb](https://github.com/justinsb))
* Volumes and environment variables populated from ConfigMap and Secret objects can now tolerate the named source object or specific keys being missing, by adding `optional: true` to the volume or environment variable source specifications. ([#39981](https://github.com/kubernetes/kubernetes/pull/39981), [@fraenkel](https://github.com/fraenkel))
* kubectl create now accepts the label selector flag for filtering objects to create ([#40057](https://github.com/kubernetes/kubernetes/pull/40057), [@MrHohn](https://github.com/MrHohn))
* A new field `terminationMessagePolicy` has been added to containers that allows a user to request `FallbackToLogsOnError`, which will read from the container's logs to populate the termination message if the user does not write to the termination message log file.  The termination message file is now properly readable for end users and has a maximum size (4k bytes) to prevent abuse.  Each pod may have up to 12k bytes of termination messages before the contents of each will be truncated. ([#39341](https://github.com/kubernetes/kubernetes/pull/39341), [@smarterclayton](https://github.com/smarterclayton))
* Add a special purpose tool for editing individual fields in a ConfigMap with kubectl ([#38445](https://github.com/kubernetes/kubernetes/pull/38445), [@brendandburns](https://github.com/brendandburns))
* [Federation] Expose autoscaling apis through federation api server ([#38976](https://github.com/kubernetes/kubernetes/pull/38976), [@irfanurrehman](https://github.com/irfanurrehman))
* Powershell script to start kubelet and kube-proxy ([#36250](https://github.com/kubernetes/kubernetes/pull/36250), [@jbhurat](https://github.com/jbhurat))
* Reduce time needed to attach Azure disks ([#40066](https://github.com/kubernetes/kubernetes/pull/40066), [@codablock](https://github.com/codablock))
* fixing Cassandra shutdown example to avoid data corruption ([#39199](https://github.com/kubernetes/kubernetes/pull/39199), [@deimosfr](https://github.com/deimosfr))
* kubeadm: add optional self-hosted deployment for apiserver, controller-manager and scheduler. ([#40075](https://github.com/kubernetes/kubernetes/pull/40075), [@pires](https://github.com/pires))
* kubelet tears down pod volumes on pod termination rather than pod deletion ([#37228](https://github.com/kubernetes/kubernetes/pull/37228), [@sjenning](https://github.com/sjenning))
* The default client certificate generated by kube-up now contains the superuser `system:masters` group ([#39966](https://github.com/kubernetes/kubernetes/pull/39966), [@liggitt](https://github.com/liggitt))
* CRI: upgrade protobuf to v3 ([#39158](https://github.com/kubernetes/kubernetes/pull/39158), [@feiskyer](https://github.com/feiskyer))
* Add SIGCHLD handler to pause container ([#36853](https://github.com/kubernetes/kubernetes/pull/36853), [@verb](https://github.com/verb))
* Populate environment variables from a secrets. ([#39446](https://github.com/kubernetes/kubernetes/pull/39446), [@fraenkel](https://github.com/fraenkel))
* fluentd config for GKE clusters updated: detect exceptions in container log streams and forward them as one log entry. ([#39656](https://github.com/kubernetes/kubernetes/pull/39656), [@thomasschickinger](https://github.com/thomasschickinger))
* Made multi-scheduler graduated to Beta and then v1. ([#38871](https://github.com/kubernetes/kubernetes/pull/38871), [@k82cn](https://github.com/k82cn))
* Add authorization mode to kubeadm ([#39846](https://github.com/kubernetes/kubernetes/pull/39846), [@andrewrynhard](https://github.com/andrewrynhard))
* Update dependencies: aws-sdk-go to 1.6.10; also cadvisor ([#40095](https://github.com/kubernetes/kubernetes/pull/40095), [@dashpole](https://github.com/dashpole))
* Fixes a bug in the OpenStack-Heat kubernetes provider, in the handling of differences between the Identity v2 and Identity v3 APIs ([#40105](https://github.com/kubernetes/kubernetes/pull/40105), [@sc68cal](https://github.com/sc68cal))
* Update GCE ContainerVM deployment to container-vm-v20170117 to pick up CVE fixes in base image. ([#40094](https://github.com/kubernetes/kubernetes/pull/40094), [@zmerlynn](https://github.com/zmerlynn))
* AWS: Remove duplicate calls to DescribeInstance during volume operations ([#39842](https://github.com/kubernetes/kubernetes/pull/39842), [@gnufied](https://github.com/gnufied))
* The `attributeRestrictions` field has been removed from the PolicyRule type in the rbac.authorization.k8s.io/v1alpha1 API. The field was not used by the RBAC authorizer. ([#39625](https://github.com/kubernetes/kubernetes/pull/39625), [@deads2k](https://github.com/deads2k))
* Enable lazy inode table and journal initialization for ext3 and ext4 ([#38865](https://github.com/kubernetes/kubernetes/pull/38865), [@codablock](https://github.com/codablock))
* azure disk: restrict name length for Azure specifications ([#40030](https://github.com/kubernetes/kubernetes/pull/40030), [@colemickens](https://github.com/colemickens))
* Follow redirects for streaming requests (exec/attach/port-forward) in the apiserver by default (alpha -> beta). ([#40039](https://github.com/kubernetes/kubernetes/pull/40039), [@timstclair](https://github.com/timstclair))
* Use kube-dns:1.11.0 ([#39925](https://github.com/kubernetes/kubernetes/pull/39925), [@sadlil](https://github.com/sadlil))
* Anonymous authentication is now automatically disabled if the API server is started with the AlwaysAllow authorizer. ([#38706](https://github.com/kubernetes/kubernetes/pull/38706), [@deads2k](https://github.com/deads2k))
* genericapiserver: cut off kube pkg/version dependency ([#39943](https://github.com/kubernetes/kubernetes/pull/39943), [@sttts](https://github.com/sttts))
* genericapiserver: cut off pkg/serviceaccount dependency ([#39945](https://github.com/kubernetes/kubernetes/pull/39945), [@sttts](https://github.com/sttts))
* Move pkg/api/rest into genericapiserver ([#39948](https://github.com/kubernetes/kubernetes/pull/39948), [@sttts](https://github.com/sttts))
* genericapiserver: cut off pkg/apis/extensions and pkg/storage dependencies ([#39946](https://github.com/kubernetes/kubernetes/pull/39946), [@sttts](https://github.com/sttts))
* genericapiserver: cut off certificates api dependency ([#39947](https://github.com/kubernetes/kubernetes/pull/39947), [@sttts](https://github.com/sttts))
* Admission control support for versioned configuration files ([#39109](https://github.com/kubernetes/kubernetes/pull/39109), [@derekwaynecarr](https://github.com/derekwaynecarr))
* Fix issue around merging lists of primitives when using PATCH or kubectl apply. ([#38665](https://github.com/kubernetes/kubernetes/pull/38665), [@ymqytw](https://github.com/ymqytw))
* Fixes API compatibility issue with empty lists incorrectly returning a null `items` field instead of an empty array. ([#39834](https://github.com/kubernetes/kubernetes/pull/39834), [@liggitt](https://github.com/liggitt))
* kubelet: remove the pleg health check from healthz ([#37865](https://github.com/kubernetes/kubernetes/pull/37865), [@yujuhong](https://github.com/yujuhong))
* [scheduling] Moved pod affinity and anti-affinity from annotations to api fields [#25319](https://github.com/kubernetes/kubernetes/pull/25319) ([#39478](https://github.com/kubernetes/kubernetes/pull/39478), [@rrati](https://github.com/rrati))
* PodSecurityPolicy resource is now enabled by default in the extensions API group. ([#39743](https://github.com/kubernetes/kubernetes/pull/39743), [@pweil-](https://github.com/pweil-))
* add --controllers to controller manager ([#39740](https://github.com/kubernetes/kubernetes/pull/39740), [@deads2k](https://github.com/deads2k))
* proxy/iptables: don't sync proxy rules if services map didn't change ([#38996](https://github.com/kubernetes/kubernetes/pull/38996), [@dcbw](https://github.com/dcbw))
* NONE ([#39768](https://github.com/kubernetes/kubernetes/pull/39768), [@rkouj](https://github.com/rkouj))
* Update amd64 kube-proxy base image to debian-iptables-amd64:v5 ([#39725](https://github.com/kubernetes/kubernetes/pull/39725), [@ixdy](https://github.com/ixdy))
* Update dashboard version to v1.5.1 ([#39662](https://github.com/kubernetes/kubernetes/pull/39662), [@rf232](https://github.com/rf232))
* Fix kubectl get -f <file> -o <nondefault printer> so it prints all items in the file ([#39038](https://github.com/kubernetes/kubernetes/pull/39038), [@ncdc](https://github.com/ncdc))
* Scheduler treats StatefulSet pods as belonging to a single equivalence class. ([#39718](https://github.com/kubernetes/kubernetes/pull/39718), [@foxish](https://github.com/foxish))
* --basic-auth-file supports optionally specifying groups in the fourth column of the file ([#39651](https://github.com/kubernetes/kubernetes/pull/39651), [@liggitt](https://github.com/liggitt))
* To create or update an RBAC RoleBinding or ClusterRoleBinding object, a user must: ([#39383](https://github.com/kubernetes/kubernetes/pull/39383), [@liggitt](https://github.com/liggitt))
    * 1. Be authorized to make the create or update API request
    * 2. Be allowed to bind the referenced role, either by already having all of the permissions contained in the referenced role, or by having the "bind" permission on the referenced role.
* Fixes an HPA-related panic due to division-by-zero. ([#39694](https://github.com/kubernetes/kubernetes/pull/39694), [@DirectXMan12](https://github.com/DirectXMan12))
* federation: Adding support for DeleteOptions.OrphanDependents for federated services. Setting it to false while deleting a federated service also deletes the corresponding services from all registered clusters. ([#36390](https://github.com/kubernetes/kubernetes/pull/36390), [@nikhiljindal](https://github.com/nikhiljindal))
* Update kube-proxy image to be based off of Debian 8.6 base image. ([#39695](https://github.com/kubernetes/kubernetes/pull/39695), [@ixdy](https://github.com/ixdy))
* Update FitError as a message component into the PodConditionUpdater. ([#39491](https://github.com/kubernetes/kubernetes/pull/39491), [@jayunit100](https://github.com/jayunit100))
* rename kubernetes-discovery to kube-aggregator ([#39619](https://github.com/kubernetes/kubernetes/pull/39619), [@deads2k](https://github.com/deads2k))
* Allow missing keys in templates by default ([#39486](https://github.com/kubernetes/kubernetes/pull/39486), [@ncdc](https://github.com/ncdc))
* Caching added to the OIDC client auth plugin to fix races and reduce the time kubectl commands using this plugin take by several seconds. ([#38167](https://github.com/kubernetes/kubernetes/pull/38167), [@ericchiang](https://github.com/ericchiang))
* AWS: recognize eu-west-2 region ([#38746](https://github.com/kubernetes/kubernetes/pull/38746), [@justinsb](https://github.com/justinsb))
* Provide kubernetes-controller-manager flags to control volume attach/detach reconciler sync.  The duration of the syncs can be controlled, and the syncs can be shut off as well.  ([#39551](https://github.com/kubernetes/kubernetes/pull/39551), [@chrislovecnm](https://github.com/chrislovecnm))
* Generate OpenAPI definition for inlined types ([#39466](https://github.com/kubernetes/kubernetes/pull/39466), [@mbohlool](https://github.com/mbohlool))
* ShortcutExpander has been extended in a way that it will examine a ha… ([#38835](https://github.com/kubernetes/kubernetes/pull/38835), [@p0lyn0mial](https://github.com/p0lyn0mial))
* fixes nil dereference when doing a volume type check on persistent volumes ([#39493](https://github.com/kubernetes/kubernetes/pull/39493), [@sjenning](https://github.com/sjenning))
* Fix issue with PodDisruptionBudgets in which `minAvailable` specified as a percentage did not work with StatefulSet Pods. ([#39454](https://github.com/kubernetes/kubernetes/pull/39454), [@foxish](https://github.com/foxish))
* fix issue with kubectl proxy so that it will proxy an empty path - e.g. http://localhost:8001 ([#39226](https://github.com/kubernetes/kubernetes/pull/39226), [@luksa](https://github.com/luksa))
* Check if pathExists before performing Unmount ([#39311](https://github.com/kubernetes/kubernetes/pull/39311), [@rkouj](https://github.com/rkouj))
* Adding kubectl tests for federation ([#38844](https://github.com/kubernetes/kubernetes/pull/38844), [@nikhiljindal](https://github.com/nikhiljindal))
* Fix comment and optimize code ([#38084](https://github.com/kubernetes/kubernetes/pull/38084), [@tanshanshan](https://github.com/tanshanshan))
* When using OIDC authentication and specifying --oidc-username-claim=email, an `"email_verified":true` claim must be returned from the identity provider. ([#36087](https://github.com/kubernetes/kubernetes/pull/36087), [@ericchiang](https://github.com/ericchiang))
* Allow pods to define multiple environment variables from a whole ConfigMap ([#36245](https://github.com/kubernetes/kubernetes/pull/36245), [@fraenkel](https://github.com/fraenkel))
* Refactor the certificate and kubeconfig code in the kubeadm binary into two phases ([#39280](https://github.com/kubernetes/kubernetes/pull/39280), [@luxas](https://github.com/luxas))
* Added support for printing in all supported `--output` formats to `kubectl create ...` and `kubectl apply ...` ([#38112](https://github.com/kubernetes/kubernetes/pull/38112), [@juanvallejo](https://github.com/juanvallejo))
* genericapiserver: extract CA cert from server cert and SNI cert chains ([#39022](https://github.com/kubernetes/kubernetes/pull/39022), [@sttts](https://github.com/sttts))
* Endpoints, that tolerate unready Pods, are now listing Pods in state Terminating as well ([#37093](https://github.com/kubernetes/kubernetes/pull/37093), [@simonswine](https://github.com/simonswine))
* DaemonSet ObservedGeneration ([#39157](https://github.com/kubernetes/kubernetes/pull/39157), [@lukaszo](https://github.com/lukaszo))
* The `--reconcile-cidr` kubelet flag was removed since it had been deprecated since v1.5 ([#39322](https://github.com/kubernetes/kubernetes/pull/39322), [@luxas](https://github.com/luxas))
* Remove all MAINTAINER statements in Dockerfiles in the codebase as they are deprecated by docker ([#38927](https://github.com/kubernetes/kubernetes/pull/38927), [@luxas](https://github.com/luxas))
* Remove the deprecated vsphere kube-up. ([#39140](https://github.com/kubernetes/kubernetes/pull/39140), [@kerneltime](https://github.com/kerneltime))
* Kubectl top now also accepts short forms for "node" and "pod" ("no", "po") ([#39218](https://github.com/kubernetes/kubernetes/pull/39218), [@luksa](https://github.com/luksa))
* Remove 'exec' network plugin - use CNI instead ([#39254](https://github.com/kubernetes/kubernetes/pull/39254), [@freehan](https://github.com/freehan))
* Add three more columns to `kubectl get deploy -o wide` output. ([#39240](https://github.com/kubernetes/kubernetes/pull/39240), [@xingzhou](https://github.com/xingzhou))
* Add path exist check in getPodVolumePathListFromDisk ([#38909](https://github.com/kubernetes/kubernetes/pull/38909), [@jingxu97](https://github.com/jingxu97))
* Begin paths for internationalization in kubectl ([#36802](https://github.com/kubernetes/kubernetes/pull/36802), [@brendandburns](https://github.com/brendandburns))
* Fixes an issue where commas were not accepted in --from-literal flags when creating secrets. Passing multiple values separated by a comma in a single --from-literal flag is no longer supported. Please use multiple --from-literal flags to provide multiple values. ([#35191](https://github.com/kubernetes/kubernetes/pull/35191), [@SamiHiltunen](https://github.com/SamiHiltunen))
* Support loading UTF16 files if a byte-order-mark is present ([#39008](https://github.com/kubernetes/kubernetes/pull/39008), [@brendandburns](https://github.com/brendandburns))
* Fix fsGroup to vSphere ([#38655](https://github.com/kubernetes/kubernetes/pull/38655), [@abrarshivani](https://github.com/abrarshivani))
* Don't evict static pods ([#39059](https://github.com/kubernetes/kubernetes/pull/39059), [@bprashanth](https://github.com/bprashanth))
* Fixed a bug where the --server, --token, and --certificate-authority flags were not overriding the related in-cluster configs when provided in a `kubectl` call inside a cluster. ([#39006](https://github.com/kubernetes/kubernetes/pull/39006), [@fabianofranz](https://github.com/fabianofranz))
* Make fluentd pods critical ([#39146](https://github.com/kubernetes/kubernetes/pull/39146), [@crassirostris](https://github.com/crassirostris))
* assign -998 as the oom_score_adj for critical pods (e.g. kube-proxy) ([#39114](https://github.com/kubernetes/kubernetes/pull/39114), [@dchen1107](https://github.com/dchen1107))
* delete continue in monitorNodeStatus ([#38798](https://github.com/kubernetes/kubernetes/pull/38798), [@NickrenREN](https://github.com/NickrenREN))
* add create rolebinding ([#38991](https://github.com/kubernetes/kubernetes/pull/38991), [@deads2k](https://github.com/deads2k))
* Federation: Add `batch/jobs` API objects to federation-apiserver ([#35943](https://github.com/kubernetes/kubernetes/pull/35943), [@jianhuiz](https://github.com/jianhuiz))
* ABAC policies using "user":"*" or "group":"*" to match all users or groups will only match authenticated requests. To match unauthenticated requests, ABAC policies must explicitly specify "group":"system:unauthenticated" ([#38968](https://github.com/kubernetes/kubernetes/pull/38968), [@liggitt](https://github.com/liggitt))
* To add local registry to libvirt_coreos ([#36751](https://github.com/kubernetes/kubernetes/pull/36751), [@sdminonne](https://github.com/sdminonne))
* Add a KUBERNETES_NODE_* section to build kubelet/kube-proxy for windows ([#38919](https://github.com/kubernetes/kubernetes/pull/38919), [@brendandburns](https://github.com/brendandburns))
* Added kubeadm commands to manage bootstrap tokens and the duration they are valid for. ([#35805](https://github.com/kubernetes/kubernetes/pull/35805), [@dgoodwin](https://github.com/dgoodwin))
* AWS: Recognize ca-central-1 region ([#38410](https://github.com/kubernetes/kubernetes/pull/38410), [@justinsb](https://github.com/justinsb))
* Unmount operation should not fail if volume is already unmounted ([#38547](https://github.com/kubernetes/kubernetes/pull/38547), [@rkouj](https://github.com/rkouj))
* Changed default scsi controller type in vSphere Cloud Provider ([#38426](https://github.com/kubernetes/kubernetes/pull/38426), [@abrarshivani](https://github.com/abrarshivani))
* Move non-generic apiserver code out of the generic packages ([#38191](https://github.com/kubernetes/kubernetes/pull/38191), [@sttts](https://github.com/sttts))
* Admit critical pods in the kubelet ([#38836](https://github.com/kubernetes/kubernetes/pull/38836), [@bprashanth](https://github.com/bprashanth))
* Use daemonset in docker registry add on ([#35582](https://github.com/kubernetes/kubernetes/pull/35582), [@surajssd](https://github.com/surajssd))
* Node affinity has moved from annotations to api fields in the pod spec.  Node affinity that is defined in the annotations will be ignored. ([#37299](https://github.com/kubernetes/kubernetes/pull/37299), [@rrati](https://github.com/rrati))
* Since `kubernetes.tar.gz` no longer includes client or server binaries, `cluster/kube-{up,down,push}.sh` now automatically download released binaries if they are missing. ([#38730](https://github.com/kubernetes/kubernetes/pull/38730), [@ixdy](https://github.com/ixdy))
* genericapiserver: turn APIContainer.SecretRoutes into a real ServeMux ([#38826](https://github.com/kubernetes/kubernetes/pull/38826), [@sttts](https://github.com/sttts))
* Migrated fluentd addon to daemon set ([#32088](https://github.com/kubernetes/kubernetes/pull/32088), [@piosz](https://github.com/piosz))
* AWS: Add sequential allocator for device names. ([#38818](https://github.com/kubernetes/kubernetes/pull/38818), [@jsafrane](https://github.com/jsafrane))
* Add 'X-Content-Type-Options: nosniff" to some error messages ([#37190](https://github.com/kubernetes/kubernetes/pull/37190), [@brendandburns](https://github.com/brendandburns))
* Display pod node selectors with kubectl describe. ([#36396](https://github.com/kubernetes/kubernetes/pull/36396), [@aveshagarwal](https://github.com/aveshagarwal))
* The main repository does not keep multiple releases of clientsets anymore. Please find previous releases at https://github.com/kubernetes/client-go ([#38154](https://github.com/kubernetes/kubernetes/pull/38154), [@caesarxuchao](https://github.com/caesarxuchao))
* Remove a release-note on multiple OpenAPI specs ([#38732](https://github.com/kubernetes/kubernetes/pull/38732), [@mbohlool](https://github.com/mbohlool))
* genericapiserver: unify swagger and openapi in config ([#38690](https://github.com/kubernetes/kubernetes/pull/38690), [@sttts](https://github.com/sttts))
* fix connection upgrades through kuberentes-discovery ([#38724](https://github.com/kubernetes/kubernetes/pull/38724), [@deads2k](https://github.com/deads2k))
* Fixes bug in resolving client-requested API versions ([#38533](https://github.com/kubernetes/kubernetes/pull/38533), [@DirectXMan12](https://github.com/DirectXMan12))
* apiserver(s): Replace glog.Fatals with fmt.Errorfs ([#38175](https://github.com/kubernetes/kubernetes/pull/38175), [@sttts](https://github.com/sttts))
* Remove Azure Subnet RouteTable check ([#38334](https://github.com/kubernetes/kubernetes/pull/38334), [@mogthesprog](https://github.com/mogthesprog))
* Ensure the GCI metadata files do not have newline at the end ([#38727](https://github.com/kubernetes/kubernetes/pull/38727), [@Amey-D](https://github.com/Amey-D))
* Significantly speed-up make ([#38700](https://github.com/kubernetes/kubernetes/pull/38700), [@sttts](https://github.com/sttts))
* add QoS pod status field ([#37968](https://github.com/kubernetes/kubernetes/pull/37968), [@sjenning](https://github.com/sjenning))
* Fixed validation of multizone cluster for GCE ([#38695](https://github.com/kubernetes/kubernetes/pull/38695), [@jszczepkowski](https://github.com/jszczepkowski))
* Fixed detection of master during creation of multizone nodes cluster by kube-up. ([#38617](https://github.com/kubernetes/kubernetes/pull/38617), [@jszczepkowski](https://github.com/jszczepkowski))
* Update CHANGELOG.md to warn about anon auth flag ([#38675](https://github.com/kubernetes/kubernetes/pull/38675), [@erictune](https://github.com/erictune))
* Fixes NotAuthenticated errors that appear in the kubelet and kube-controller-manager due to never logging in to vSphere ([#36169](https://github.com/kubernetes/kubernetes/pull/36169), [@robdaemon](https://github.com/robdaemon))
* Fix an issue where AWS tear-down leaks an DHCP Option Set. ([#38645](https://github.com/kubernetes/kubernetes/pull/38645), [@zmerlynn](https://github.com/zmerlynn))
* Issue a warning when using `kubectl apply` on a resource lacking the `LastAppliedConfig` annotation ([#36672](https://github.com/kubernetes/kubernetes/pull/36672), [@ymqytw](https://github.com/ymqytw))
* Re-add /healthz/ping handler in genericapiserver ([#38603](https://github.com/kubernetes/kubernetes/pull/38603), [@sttts](https://github.com/sttts))
* Fail kubelet if runtime is unresponsive for 30 seconds ([#38527](https://github.com/kubernetes/kubernetes/pull/38527), [@derekwaynecarr](https://github.com/derekwaynecarr))
* Add support for Azure Container Registry, update Azure dependencies ([#37783](https://github.com/kubernetes/kubernetes/pull/37783), [@brendandburns](https://github.com/brendandburns))
* Fix panic in vSphere cloud provider ([#38423](https://github.com/kubernetes/kubernetes/pull/38423), [@BaluDontu](https://github.com/BaluDontu))
* fix broken cluster/centos and enhance the style ([#34002](https://github.com/kubernetes/kubernetes/pull/34002), [@xiaoping378](https://github.com/xiaoping378))
* Ability to quota storage by storage class ([#34554](https://github.com/kubernetes/kubernetes/pull/34554), [@derekwaynecarr](https://github.com/derekwaynecarr))
* [Part 2] Adding s390x cross-compilation support for gcr.io images in this repo ([#36050](https://github.com/kubernetes/kubernetes/pull/36050), [@gajju26](https://github.com/gajju26))
* kubectl run --rm no longer prints "pod xxx deleted" ([#38429](https://github.com/kubernetes/kubernetes/pull/38429), [@duglin](https://github.com/duglin))
* Bump GCE debian image to container-vm-v20161208 ([release notes](https://cloud.google.com/compute/docs/containers/container_vms#changelog)) ([#38432](https://github.com/kubernetes/kubernetes/pull/38432), [@timstclair](https://github.com/timstclair))
* Kubelet: Add image cache. ([#38375](https://github.com/kubernetes/kubernetes/pull/38375), [@Random-Liu](https://github.com/Random-Liu))
* Allow a selector when retrieving logs ([#32752](https://github.com/kubernetes/kubernetes/pull/32752), [@fraenkel](https://github.com/fraenkel))
* Fix unmountDevice issue caused by shared mount in GCI ([#38411](https://github.com/kubernetes/kubernetes/pull/38411), [@jingxu97](https://github.com/jingxu97))
* [Federation] Implement dry run support in kubefed init ([#36447](https://github.com/kubernetes/kubernetes/pull/36447), [@irfanurrehman](https://github.com/irfanurrehman))
* Fix space issue in volumePath with vSphere Cloud Provider ([#38338](https://github.com/kubernetes/kubernetes/pull/38338), [@BaluDontu](https://github.com/BaluDontu))
* [Federation] Make federation etcd PVC size configurable ([#36310](https://github.com/kubernetes/kubernetes/pull/36310), [@irfanurrehman](https://github.com/irfanurrehman))
* Allow no ports when exposing headless service ([#32811](https://github.com/kubernetes/kubernetes/pull/32811), [@fraenkel](https://github.com/fraenkel))
* Wait for the port to be ready before starting ([#38260](https://github.com/kubernetes/kubernetes/pull/38260), [@fraenkel](https://github.com/fraenkel))
* Add Version to the resource printer for 'get nodes' ([#37943](https://github.com/kubernetes/kubernetes/pull/37943), [@ailusazh](https://github.com/ailusazh))
* contribute deis/registry-proxy as a replacement for kube-registry-proxy ([#35797](https://github.com/kubernetes/kubernetes/pull/35797), [@bacongobbler](https://github.com/bacongobbler))
* [Part 1] Add support for cross-compiling s390x binaries ([#37092](https://github.com/kubernetes/kubernetes/pull/37092), [@gajju26](https://github.com/gajju26))
* kernel memcg notification enabled via experimental flag ([#38258](https://github.com/kubernetes/kubernetes/pull/38258), [@derekwaynecarr](https://github.com/derekwaynecarr))
* fix permissions when using fsGroup ([#37009](https://github.com/kubernetes/kubernetes/pull/37009), [@sjenning](https://github.com/sjenning))
* Pipe get options to storage ([#37693](https://github.com/kubernetes/kubernetes/pull/37693), [@wojtek-t](https://github.com/wojtek-t))
* add a configuration for kubelet to register as a node with taints ([#31647](https://github.com/kubernetes/kubernetes/pull/31647), [@mikedanese](https://github.com/mikedanese))
* Remove genericapiserver.Options.MasterServiceNamespace ([#38186](https://github.com/kubernetes/kubernetes/pull/38186), [@sttts](https://github.com/sttts))
* The --long-running-request-regexp flag to kube-apiserver is deprecated and will be removed in a future release. Long-running requests are now detected based on specific verbs (watch, proxy) or subresources (proxy, portforward, log, exec, attach). ([#38119](https://github.com/kubernetes/kubernetes/pull/38119), [@liggitt](https://github.com/liggitt))
* Better compat with very old iptables (e.g. CentOS 6) ([#37594](https://github.com/kubernetes/kubernetes/pull/37594), [@thockin](https://github.com/thockin))
* Fix GCI mounter issue ([#38124](https://github.com/kubernetes/kubernetes/pull/38124), [@jingxu97](https://github.com/jingxu97))
* Kubelet will no longer set hairpin mode on every interface on the machine when an error occurs in setting up hairpin for a specific interface. ([#36990](https://github.com/kubernetes/kubernetes/pull/36990), [@bboreham](https://github.com/bboreham))
* fix mesos unit tests ([#38196](https://github.com/kubernetes/kubernetes/pull/38196), [@deads2k](https://github.com/deads2k))
* Add `--controllers` flag to federation controller manager for enable/disable federation ingress controller ([#36643](https://github.com/kubernetes/kubernetes/pull/36643), [@kzwang](https://github.com/kzwang))
* Allow backendpools in Azure Load Balancers which are not owned by cloud provider ([#36882](https://github.com/kubernetes/kubernetes/pull/36882), [@codablock](https://github.com/codablock))
* remove rbac super user ([#38121](https://github.com/kubernetes/kubernetes/pull/38121), [@deads2k](https://github.com/deads2k))
* Fix resync goroutine leak in ListAndWatch ([#35672](https://github.com/kubernetes/kubernetes/pull/35672), [@tatsuhiro-t](https://github.com/tatsuhiro-t))
* API server have two separate limits for read-only and mutating inflight requests. ([#36064](https://github.com/kubernetes/kubernetes/pull/36064), [@gmarek](https://github.com/gmarek))
* check the value of min and max in kubectl ([#37789](https://github.com/kubernetes/kubernetes/pull/37789), [@yarntime](https://github.com/yarntime))
* The glusterfs dynamic volume provisioner will now choose a unique GID for new persistent volumes from a range that can be configured in the storage class with the "gidMin" and "gidMax" parameters. The default range is 2000 - 2147483647 (max int32). ([#37886](https://github.com/kubernetes/kubernetes/pull/37886), [@obnoxxx](https://github.com/obnoxxx))
* Set kernel.softlockup_panic =1 based on the flag. ([#38001](https://github.com/kubernetes/kubernetes/pull/38001), [@dchen1107](https://github.com/dchen1107))
* portfordwardtester: avoid data loss during send+close+exit ([#37103](https://github.com/kubernetes/kubernetes/pull/37103), [@sttts](https://github.com/sttts))
* Enable containerized mounter only for nfs and glusterfs types ([#37990](https://github.com/kubernetes/kubernetes/pull/37990), [@jingxu97](https://github.com/jingxu97))
* Add flag to enable contention profiling in scheduler. ([#37357](https://github.com/kubernetes/kubernetes/pull/37357), [@gmarek](https://github.com/gmarek))
* Add kubernetes-anywhere as a new e2e deployment option ([#37019](https://github.com/kubernetes/kubernetes/pull/37019), [@pipejakob](https://github.com/pipejakob))
* add create clusterrolebinding command ([#37098](https://github.com/kubernetes/kubernetes/pull/37098), [@deads2k](https://github.com/deads2k))
* kubectl create service externalname ([#34789](https://github.com/kubernetes/kubernetes/pull/34789), [@adohe](https://github.com/adohe))
* Fix logic error in graceful deletion ([#37721](https://github.com/kubernetes/kubernetes/pull/37721), [@derekwaynecarr](https://github.com/derekwaynecarr))
* Exit with error if <version number or publication> is not the final parameter. ([#37723](https://github.com/kubernetes/kubernetes/pull/37723), [@mtaufen](https://github.com/mtaufen))
* Fix Service Update on LoadBalancerSourceRanges Field ([#37720](https://github.com/kubernetes/kubernetes/pull/37720), [@freehan](https://github.com/freehan))
* Bug fix. Incoming UDP packets not reach newly deployed services ([#32561](https://github.com/kubernetes/kubernetes/pull/32561), [@zreigz](https://github.com/zreigz))
* GCI: Remove /var/lib/docker/network ([#37593](https://github.com/kubernetes/kubernetes/pull/37593), [@yujuhong](https://github.com/yujuhong))
* configure local-up-cluster.sh to handle auth proxies ([#36838](https://github.com/kubernetes/kubernetes/pull/36838), [@deads2k](https://github.com/deads2k))
* Add `clusterid`, an optional parameter to storageclass. ([#36437](https://github.com/kubernetes/kubernetes/pull/36437), [@humblec](https://github.com/humblec))
* local-up-cluster: avoid sudo for control plane ([#37443](https://github.com/kubernetes/kubernetes/pull/37443), [@sttts](https://github.com/sttts))
* Add error message when trying to use clusterrole with namespace in kubectl ([#36424](https://github.com/kubernetes/kubernetes/pull/36424), [@xilabao](https://github.com/xilabao))
* `kube-up.sh`/`kube-down.sh` no longer force update gcloud for provider=gce|gke. ([#36292](https://github.com/kubernetes/kubernetes/pull/36292), [@jlowdermilk](https://github.com/jlowdermilk))
* Fix issue when attempting to unmount a wrong vSphere volume ([#37413](https://github.com/kubernetes/kubernetes/pull/37413), [@BaluDontu](https://github.com/BaluDontu))
* Fix the equality checks for numeric values in cluster/gce/util.sh. ([#37638](https://github.com/kubernetes/kubernetes/pull/37638), [@roberthbailey](https://github.com/roberthbailey))
* kubelet: don't reject pods without adding them to the pod manager ([#37661](https://github.com/kubernetes/kubernetes/pull/37661), [@yujuhong](https://github.com/yujuhong))
* Modify GCI mounter to enable NFSv3 ([#37582](https://github.com/kubernetes/kubernetes/pull/37582), [@jingxu97](https://github.com/jingxu97))
* Fix photon controller plugin to construct with correct PdID ([#37167](https://github.com/kubernetes/kubernetes/pull/37167), [@luomiao](https://github.com/luomiao))
* Collect logs for dead kubelets too ([#37671](https://github.com/kubernetes/kubernetes/pull/37671), [@mtaufen](https://github.com/mtaufen))
* Set Dashboard UI version to v1.5.0 ([#37684](https://github.com/kubernetes/kubernetes/pull/37684), [@rf232](https://github.com/rf232))
* When deleting an object with `--grace-period=0`, the client will begin a graceful deletion and wait until the resource is fully deleted.  To force deletion, use the `--force` flag. ([#37263](https://github.com/kubernetes/kubernetes/pull/37263), [@smarterclayton](https://github.com/smarterclayton))
* federation service controller: stop deleting services from underlying clusters when federated service is deleted. ([#37353](https://github.com/kubernetes/kubernetes/pull/37353), [@nikhiljindal](https://github.com/nikhiljindal))
* Fix nil pointer dereference in test framework ([#37583](https://github.com/kubernetes/kubernetes/pull/37583), [@mtaufen](https://github.com/mtaufen))
* Kubernetes now automatically installs a StorageClass object when deployed on ([#31617](https://github.com/kubernetes/kubernetes/pull/31617), [@jsafrane](https://github.com/jsafrane))
    * AWS, Google Compute Engine, Google Container Engine, and OpenStack using
    * the default kube-up.sh scripts. This StorageClass is marked as default so that
    * a PersistentVolumeClaim without a StorageClass annotation now results in
    * automatic provisioning of storage (GCE PersistentDisk on Google Cloud, AWS
    * EBS on AWS, and Cinder on OpenStack). In previous versions of Kubernetes
    * a PersistentVolumeClaim without a StorageClass annotation on these cloud
    * platforms would be satisfied by manually-created PersistentVolume objects.
    * Administrators can choose to disable this behavior by deleting the automatically
    * installed StorageClass API object. Alternatively, administrators may choose to
    * keep the automatically installed StorageClass and only disable the defaulting
    * behavior by removing the "is-default-class" annotation from the StorageClass
    * API object.
* Fix TestServiceAlloc flakes ([#37487](https://github.com/kubernetes/kubernetes/pull/37487), [@wojtek-t](https://github.com/wojtek-t))
* Use gsed on the Mac ([#37562](https://github.com/kubernetes/kubernetes/pull/37562), [@roberthbailey](https://github.com/roberthbailey))
* Try self-repair scheduler cache or panic ([#37379](https://github.com/kubernetes/kubernetes/pull/37379), [@wojtek-t](https://github.com/wojtek-t))
* Fluentd/Elastisearch add-on: correctly parse and index kubernetes labels ([#36857](https://github.com/kubernetes/kubernetes/pull/36857), [@Shrugs](https://github.com/Shrugs))
* Mention overflows when mistakenly call function FromInt ([#36487](https://github.com/kubernetes/kubernetes/pull/36487), [@xialonglee](https://github.com/xialonglee))
* Update doc for kubectl apply ([#37397](https://github.com/kubernetes/kubernetes/pull/37397), [@ymqytw](https://github.com/ymqytw))
* Removes shorthand flag -w from kubectl apply ([#37345](https://github.com/kubernetes/kubernetes/pull/37345), [@MrHohn](https://github.com/MrHohn))
* Curating Owners: pkg/api ([#36525](https://github.com/kubernetes/kubernetes/pull/36525), [@apelisse](https://github.com/apelisse))
* Reduce verbosity of volume reconciler when attaching volumes ([#36900](https://github.com/kubernetes/kubernetes/pull/36900), [@codablock](https://github.com/codablock))
