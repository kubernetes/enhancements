## **Major Themes**

Kubernetes 1.7 is a milestone release that adds security, stateful application, and extensibility features motivated by widespread production use of Kubernetes.

Security enhancements in this release include encrypted secrets (alpha), network policy for pod-to-pod communication, the node authorizer to limit Kubelet access to API resources, and Kubelet client / server TLS certificate rotation (alpha).  

Major features for stateful applications include automated updates to StatefulSets, enhanced updates for DaemonSets, a burst mode for faster StatefulSets scaling, and (alpha) support for local storage.

Extensibility features include API aggregation (beta), CustomResourceDefinitions (beta) in favor of ThirdPartyResources, support for extensible admission controllers (alpha), pluggable cloud providers (alpha), and container runtime interface (CRI) enhancements.

## **Action Required Before Upgrading**

* NetworkPolicy has been promoted from extensions/v1beta1 to the new networking.k8s.io/v1 API group. The structure remains unchanged from the v1beta1 API. The net.beta.kubernetes.io/network-policy annotation on Namespaces (used to opt in to isolation) has been removed. Instead, isolation is now determined on a per-pod basis. A NetworkPolicy may target a pod for isolation by including the pod in its spec.podSelector. Targeted Pods accept the traffic specified in the respective NetworkPolicy (and nothing else). Pods not targeted by any NetworkPolicy accept all traffic by default. ([#39164](https://github.com/kubernetes/kubernetes/pull/39164), [@danwinship](https://github.com/danwinship))

	**Action Required:** When upgrading to Kubernetes 1.7 (and a [network plugin](https://kubernetes.io/docs/tasks/administer-cluster/declare-network-policy/) that supports the new NetworkPolicy v1 semantics), you should consider the following.
	
	The v1beta1 API used an annotation on Namespaces to activate the DefaultDeny policy for an entire Namespace. To activate default deny in the v1 API, you can create a NetworkPolicy that matches all Pods but does not allow any traffic:

    ```yaml
    kind: NetworkPolicy
    apiVersion: networking.k8s.io/v1
    metadata:
      name: default-deny
    spec:
      podSelector:
    ```

	This will ensure that Pods that aren't matched by any other NetworkPolicy will continue to be fully-isolated, as they were in v1beta1. 
 
	In Namespaces that previously did not have the "DefaultDeny" annotation, you should delete any existing NetworkPolicy objects. These had no effect in the v1beta1 API, but with v1 semantics they might cause some traffic to be unintentionally blocked.

* `cluster/update-storage-objects.sh` now supports updating StorageClasses in etcd to storage.k8s.io/v1. You must do this prior to upgrading to 1.8. ([#46116](https://github.com/kubernetes/kubernetes/pull/46116), [@ncdc](https://github.com/ncdc))

* PodSpecs containing parent directory references such as `..` (for example, `../bar`) in hostPath volume path or in volumeMount subpaths must be changed to the simple absolute path. Backsteps `..` are no longer allowed.([#47290](https://github.com/kubernetes/kubernetes/pull/47290), [@jhorwit2](https://github.com/jhorwit2)). 

* Azure: Container permissions for provisioned volumes have changed to private. If you have existing Azure volumes that were created by Kubernetes v1.6.0-v1.6.5, you should change the permissions on them manually. ([#47605](https://github.com/kubernetes/kubernetes/pull/47605), [@brendandburns](https://github.com/brendandburns))

* GKE/GCE: New and upgraded 1.7 GCE/GKE clusters no longer have an RBAC ClusterRoleBinding that grants the cluster-admin ClusterRole to the default service account in the kube-system Namespace. ([#46750](https://github.com/kubernetes/kubernetes/pull/46750), [@cjcullen](https://github.com/cjcullen)). If this permission is still desired, run the following command to explicitly grant it, either before or after upgrading to 1.7:
    ```
    kubectl create clusterrolebinding kube-system-default --serviceaccount=kube-system:default --clusterrole=cluster-admin
    ```

* kube-controller-manager has dropped support for the `--insecure-experimental-approve-all-kubelet-csrs-for-group` flag. It is accepted in 1.7, but ignored. Instead, the csrapproving controller uses authorization checks to determine whether to approve certificate signing requests: ([#45619](https://github.com/kubernetes/kubernetes/pull/45619), [@mikedanese](https://github.com/mikedanese))

    * Before upgrading, users must ensure their controller manager will enable the csrapproving controller, create an RBAC ClusterRole and ClusterRoleBinding to approve CSRs for the same group, then upgrade. Example roles to enable the equivalent behavior can be found in the TLS bootstrapping documentation ([https://github.com/kubernetes/kubernetes.github.io/pull/4208](https://github.com/kubernetes/kubernetes.github.io/pull/4208)).

* The Namespace API object no longer supports the deletecollection operation. ([#46407](https://github.com/kubernetes/kubernetes/pull/46407), [@liggitt](https://github.com/liggitt))

* `kubectl create role` and  `kubectl create clusterrole`  invocations must be updated to specify multiple resource names as repeated  `--resource-name` arguments instead of comma-separated arguments to a single `--resource-name` argument. E.g. `--resource-name=x,y` must become `--resource-name x --resource-name y` ([#44950](https://github.com/kubernetes/kubernetes/pull/44950), [@xilabao](https://github.com/xilabao))

* `kubectl create rolebinding` and `kubectl create clusterrolebinding` invocations must be updated to  specify multiple subjects as repeated  `--user`, `--group`, or `--serviceaccount` arguments instead of comma-separated arguments to a single `--user`, `--group`, or `--serviceaccount`.  E.g. `--user=x,y` must become `--user x --user y`  ([#43903](https://github.com/kubernetes/kubernetes/pull/43903), [@xilabao](https://github.com/xilabao))

* The following alpha API groups were unintentionally enabled by default in previous releases, and will no longer be enabled by default in v1.8: ([#47690](https://github.com/kubernetes/kubernetes/pull/47690), [@caesarxuchao](https://github.com/caesarxuchao))

    * rbac.authorization.k8s.io/v1alpha1

    * settings.k8s.io/v1alpha1

    * If you wish to continue using them in v1.8, please enable them explicitly using the `--runtime-config` flag on the apiserver (for example, `--runtime-config="rbac.authorization.k8s.io/v1alpha1,settings.k8s.io/v1alpha1"`)

## **Known Issues**

Populated via [v1.7.x known issues / FAQ accumulator](https://github.com/kubernetes/kubernetes/issues/46733)

* The kube-apiserver discovery APIs (for example, `/apis`) return information about the API groups being served, and can change dynamically.
During server startup, prior to the server reporting healthy (via `/healthz`), not all API groups may be reported.
Wait for the server to report healthy (via `/healthz`) before depending on the information provided by the discovery APIs.
Additionally, since the information returned from the discovery APIs may change dynamically, a cache of the results should not be considered authoritative.
ETag support is planned in a future version to facilitate client caching.
([#47977](https://github.com/kubernetes/kubernetes/issues/47977), [#44957](https://github.com/kubernetes/kubernetes/issues/44957))

* The DaemonSet controller will evict running Pods that do not tolerate the NoSchedule taint if the taint is added to a Node.  There is an open PR ([#48189](https://github.com/kubernetes/kubernetes/pull/48189)) to resolve this issue, but as this issue also exists in 1.6, and as we do not wish to risk release stability by merging it directly prior to a release without sufficient testing, we have decided to defer merging the PR until the next point release for each minor version ([#48190](https://github.com/kubernetes/kubernetes/issues/48190)).

* Protobuf serialization does not distinguish between `[]` and `null`.
API fields previously capable of storing and returning either `[]` and `null` via JSON API requests (for example, the Endpoints `subsets` field)
can now store only `null` when created using the protobuf content-type or stored in etcd using protobuf serialization (the default in 1.6).
JSON API clients should tolerate `null` values for such fields, and treat `null` and `[]` as equivalent in meaning unless specifically documented otherwise for a particular field. ([#44593](https://github.com/kubernetes/kubernetes/issues/44593))

## **Deprecations**

* ThirdPartyResource (Alpha) is deprecated. Please migrate to the successor, CustomResourceDefinition (Beta). ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4071)) ([kubernetes/features#95](https://github.com/kubernetes/features/issues/95))

* Swagger 1.2 spec (/swaggerapi/*) is deprecated. Please use OpenAPI instead.

* The namespace API object no longer supports the deletecollection operation. ([#46407](https://github.com/kubernetes/kubernetes/pull/46407), [@liggitt](https://github.com/liggitt))

* DaemonSet’s .spec.templateGeneration has been deprecated.  ([#45924](https://github.com/kubernetes/kubernetes/pull/45924), [@janetkuo](https://github.com/janetkuo))

* In 1.7, the kube-proxy component has been converted to use a configuration file. The old flags still work in 1.7, but they are being deprecated and will be removed in a future release. Cluster administrators are advised to switch to using the configuration file, but no action is strictly necessary in 1.7. ([#34727](https://github.com/kubernetes/kubernetes/pull/34727), [@ncdc](https://github.com/ncdc))

## **Features**

Features for this release were tracked via the use of the [kubernetes/features](https://github.com/kubernetes/features) issues repo. Each Feature issue is owned by a Special Interest Group from [kubernetes/community](https://github.com/kubernetes/community)

TODO: replace docs PR links with links to actual docs

* API Machinery

    * [alpha] Add extensible external admission control ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4092)) ([kubernetes/features#209](https://github.com/kubernetes/features/issues/209))

    * [beta] User-provided apiservers can be aggregated (served along with) the rest of the Kubernetes API ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4173)) ([kubernetes/features#263](https://github.com/kubernetes/features/issues/263))

    * [beta] ThirdPartyResource is deprecated. Please migrate to the successor, CustomResourceDefinition. ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4071)) ([kubernetes/features#95](https://github.com/kubernetes/features/issues/95))

* Apps

    * [beta] StatefulSet supports RollingUpdate and OnDelete update strategies. ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4174)) ([kubernetes/features#188](https://github.com/kubernetes/features/issues/188))

    * [beta] DaemonSet supports history and rollback. ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4098)) ([kubernetes/features#124](https://github.com/kubernetes/features/issues/124))

    * [alpha] StatefulSet authors should be able to relax the ordering and parallelism policies for software that can safely support rapid, out-of-order changes. ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4162)) ([kubernetes/features#272](https://github.com/kubernetes/features/issues/272))

    * [beta] PodDisruptionBudget has a new field MaxUnavailable, which allows users to specify the maximum number of disruptions that can be tolerated during eviction. ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4140)) ([kubernetes/features#285](https://github.com/kubernetes/features/issues/285))

    * [beta] Deployments uses a hashing collision avoidance mechanism that ensures new rollouts will not block on hashing collisions anymore. ([kubernetes/features#287](https://github.com/kubernetes/features/issues/287))

* Auth

    * [alpha] Rotation of the server TLS certificate on the kubelet ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4208)) ([kubernetes/features#267](https://github.com/kubernetes/features/issues/267))

    * [alpha] Rotation of the client TLS certificate on the kubelet ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4208)) ([kubernetes/features#266](https://github.com/kubernetes/features/issues/266))

    * [alpha] Encrypt secrets stored in etcd ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4181)) ([kubernetes/features#92](https://github.com/kubernetes/features/issues/92))

    * [beta] A new Node authorization mode and NodeRestriction admission plugin, when used in combination, limit nodes' access to specific APIs, so that they may only modify their own Node API object, only modify Pod objects bound to themselves, and only retrieve secrets and configmaps referenced by pods bound to themselves ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4077)) ([kubernetes/features#279](https://github.com/kubernetes/features/issues/279))

    * [alpha] Advanced Auditing enhances the Kubernetes API audit logging capabilities through a customizable policy, pluggable audit backends, and richer audit data ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4101)) ([kubernetes/features#22](https://github.com/kubernetes/features/issues/22))

* Autoscaling

    * [alpha] HPA Status Conditions ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4118)) ([kubernetes/features#264](https://github.com/kubernetes/features/issues/264))

* Cluster Lifecycle

    * [alpha] Improved support for out-of-tree and out-of-process cloud providers, a.k.a pluggable cloud providers ([docs](https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller)) ([kubernetes/features#88](https://github.com/kubernetes/features/issues/88))

    * [alpha] Manual upgrades for kubeadm from v1.6 to v1.7. Automated upgrades ([kubernetes/features#296](https://github.com/kubernetes/features/issues/296)) are targeted for v1.8. ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/3999)) (part of [kubernetes/features#11](https://github.com/kubernetes/features/issues/11))

* Federation

    * [alpha] The federation-apiserver now supports a SchedulingPolicy admission controller that enables policy-based control over placement of federated resources ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4075)) ([kubernetes/features#250](https://github.com/kubernetes/features/issues/250))

    * [alpha] Federation ClusterSelector annotation to direct objects to federated clusters with matching labels ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4214)) ([kubernetes/features#74](https://github.com/kubernetes/features/issues/74))

* Instrumentation

    * [alpha] Introduces a lightweight monitoring component for serving the core resource metrics API used by the Horizontal Pod Autoscaler and other components ([kubernetes/features#271](https://github.com/kubernetes/features/issues/271))

* Network

    * [stable] NetworkPolicy promoted to GA ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4003)([kubernetes/features#185](https://github.com/kubernetes/features/issues/185))

    * [stable] Source IP Preservation - change Cloud load-balancer strategy to health-checks and respond to health check only on nodes that host pods for the service ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4093)) ([kubernetes/features#27](https://github.com/kubernetes/features/issues/27))

* Node

    * [alpha] CRI validation testing, which provides a test framework and a suite of tests to validate that the CRI server implementation meets all the requirements. This allows the CRI runtime developers to verify that their runtime conforms to CRI, without needing to set up Kubernetes components or run Kubernetes end-to-end tests. ([docs](https://github.com/kubernetes/community/blob/master/contributors/devel/cri-validation.md) and [release notes](https://github.com/kubernetes-incubator/cri-tools/releases/tag/v0.1)) ([kubernetes/features#292](https://github.com/kubernetes/features/issues/292))

    * [alpha] Adds support of container metrics in CRI ([docs PR](https://github.com/kubernetes/community/pull/742)) ([kubernetes/features#290](https://github.com/kubernetes/features/issues/290))

    * [alpha] Alpha integration with containerd 1.0, which supports basic pod lifecycle and image management. [docs](https://github.com/kubernetes-incubator/cri-containerd/blob/master/README.md) and [release notes](https://github.com/kubernetes-incubator/cri-containerd/releases/tag/v0.1.0) ([kubernetes/features#286](https://github.com/kubernetes/features/issues/286))

* Scheduling

    * [alpha] Support for delegating pod binding to a scheduler extender ([kubernetes/features#270](https://github.com/kubernetes/features/issues/270))

* Storage

    * [alpha] This feature adds capacity isolation support for local storage at node, container, and volume levels ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4145)) ([kubernetes/features#245](https://github.com/kubernetes/features/issues/245))

    * [alpha] Make locally attached (non-network attached) storage available as a persistent volume source. ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4050)) ([kubernetes/features#121](https://github.com/kubernetes/features/issues/121))

    * [stable] Volume plugin for StorageOS provides highly-available cluster-wide persistent volumes from local or attached node storage ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4095)) ([kubernetes/features#190](https://github.com/kubernetes/features/issues/190))

    * [stable] Add support for cloudprovider metrics for storage API calls ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4138)) ([kubernetes/features#182](https://github.com/kubernetes/features/issues/182))

## **External Dependency Version Information**

Continuous integration builds have used the following versions of external dependencies, however, this is not a strong recommendation and users should consult an appropriate installation or upgrade guide before deciding what versions of etcd, docker or rkt to use.

* Docker versions 1.10.3, 1.11.2, 1.12.6 have been validated

    * Docker version 1.12.6 known issues

        * overlay2 driver not fully supported

        * live-restore not fully supported

        * no shared pid namespace support

    * Docker version 1.11.2 known issues

        * Kernel crash with Aufs storage driver on Debian Jessie ([#27885](https://github.com/kubernetes/kubernetes/issues/27885)) which can be identified by the [node problem detector](http://kubernetes.io/docs/admin/node-problem/)

        * Leaked File descriptors ([#275](https://github.com/docker/containerd/issues/275))

        * Additional memory overhead per container ([#21737](https://github.com/docker/docker/issues/21737))

    * Docker 1.10.3 contains [backports provided by RedHat](https://github.com/docker/docker/compare/v1.10.3...runcom:docker-1.10.3-stable) for known issues

* rkt version 1.23.0+

    * known issues with the rkt runtime are [listed in the Getting Started Guide](http://kubernetes.io/docs/getting-started-guides/rkt/notes/)

* etcd version 3.0.17

* Go version: 1.8.3. [Link to announcement](https://groups.google.com/d/msg/kubernetes-dev/0XRRz6UhhTM/YODWVnuDBQAJ)

    * Kubernetes can only be compiled with Go 1.8. Support for all other versions is dropped.

### **Other notable changes**

* API Machinery

    * The protobuf serialization of API objects has been updated to store maps in a predictable order to ensure that the representation of that object does not change when saved into etcd. This prevents the same object from being seen as being modified, even when no values have changed. ([#47701](https://github.com/kubernetes/kubernetes/pull/47701), [@smarterclayton](https://github.com/smarterclayton))

    * API resource discovery now includes the singularName used to refer to the resource. ([#43312](https://github.com/kubernetes/kubernetes/pull/43312), [@deads2k](https://github.com/deads2k))

    * Enhance the garbage collection admission plugin so that a user who doesn't have delete permission of the owning object cannot modify the blockOwnerDeletion field of existing ownerReferences, or add new ownerReferences with blockOwnerDeletion=true ([#43876](https://github.com/kubernetes/kubernetes/pull/43876), [@caesarxuchao](https://github.com/caesarxuchao))

    * The Kubernetes API server now exits if it encounters a networking failure (e.g. the networking interface hosting its address goes away) to allow a process manager (systemd/kubelet/etc) to react to the problem. Previously the server would log the failure and try again to bind to its configured address:port. ([#42272](https://github.com/kubernetes/kubernetes/pull/42272), [@marun](https://github.com/marun))

    * Exec and portforward actions over SPDY now properly handle redirects sent by the Kubelet ([#44451](https://github.com/kubernetes/kubernetes/pull/44451), [@ncdc](https://github.com/ncdc))

    * The proxy subresource APIs for nodes, services, and pods now support the HTTP PATCH method. ([#44929](https://github.com/kubernetes/kubernetes/pull/44929), [@liggitt](https://github.com/liggitt))

    * The Categories []string field on discovered API resources represents the list of group aliases (e.g. "all") that each resource belongs to. ([#43338](https://github.com/kubernetes/kubernetes/pull/43338), [@fabianofranz](https://github.com/fabianofranz))

    * [alpha] The Kubernetes API supports retrieving tabular output for API resources via a new mime-type application/json;as=Table;v=v1alpha1;g=meta.k8s.io. The returned object (if the server supports it) will be of type meta.k8s.io/v1alpha1 with Table, and contain column and row information related to the resource. Each row will contain information about the resource - by default it will be the object metadata, but callers can add the ?includeObject=Object query parameter and receive the full object. In the future kubectl will use this to retrieve the results of kubectl get. ([#40848](https://github.com/kubernetes/kubernetes/pull/40848), [@smarterclayton](https://github.com/smarterclayton))

    * The Prometheus metrics for the kube-apiserver for tracking incoming API requests and latencies now return the subresource label for correctly attributing the type of API call. ([#46354](https://github.com/kubernetes/kubernetes/pull/46354), [@smarterclayton](https://github.com/smarterclayton))

    * The behavior of some watch calls to the server when filtering on fields was incorrect. If watching objects with a filter, when an update was made that no longer matched the filter a DELETE event was correctly sent. However, the object that was returned by that delete was not the (correct) version before the update, but instead, the newer version. That meant the new object was not matched by the filter. This was a regression from behavior between cached watches on the server side and uncached watches, and thus broke downstream API clients. ([#46223](https://github.com/kubernetes/kubernetes/pull/46223), [@smarterclayton](https://github.com/smarterclayton))

    * OpenAPI spec is now available in protobuf binary and gzip format (with ETag support) ([#45836](https://github.com/kubernetes/kubernetes/pull/45836), [@mbohlool](https://github.com/mbohlool))

    * Updating apiserver to return UID of the deleted resource. Clients can use this UID to verify that the resource was deleted or waiting for finalizers. ([#45600](https://github.com/kubernetes/kubernetes/pull/45600), [@nikhiljindal](https://github.com/nikhiljindal))

    * Fix incorrect conflict errors applying strategic merge patches to resources. ([#43871](https://github.com/kubernetes/kubernetes/pull/43871), [@liggitt](https://github.com/liggitt))

* Apps

    * StatefulSet has a configurable update strategy. Updates can be performed using the RollingUpdate or OnDelete strategies. OnDelete is the default behavior, and it implements the same behavior as StatefulSets in 1.6. RollingUpdate performs automated, ordered updates to Pods in a StatefulSet. ([#46669](https://github.com/kubernetes/kubernetes/pull/46669), [@kow3ns](https://github.com/kow3ns))([kubernetes/features#287](https://github.com/kubernetes/features/issues/287)).

    * StatefulSetStatus now tracks replicas, readyReplicas, currentReplicas, and updatedReplicas. The semantics of replicas is now consistent with DaemonSet and ReplicaSet, and readyReplicas has the semantics that replicas did prior to 1.7 ([#46669](https://github.com/kubernetes/kubernetes/pull/46669), [@kow3ns](https://github.com/kow3ns)).

    * DaemonSets supports history and rollback ([#45924](https://github.com/kubernetes/kubernetes/pull/45924), [@janetkuo](https://github.com/janetkuo))([kubernetes/features#124](https://github.com/kubernetes/features/issues/124)).

    * ControllerRevision type has been added for StatefulSet and DaemonSet history. Clients should not depend on the stability of this type as it may change, as necessary, in future releases to support StatefulSet and DaemonSet update and rollback. We enable this type as we do with beta features, because StatefulSet update and DaemonSet update are enabled. ([#45867](https://github.com/kubernetes/kubernetes/pull/45867), [@kow3ns](https://github.com/kow3ns))

    * PodDisruptionBudget now uses ControllerRef to decide which controller owns a given Pod, so it doesn't get confused by controllers with overlapping selectors. ([#45003](https://github.com/kubernetes/kubernetes/pull/45003), [@krmayankk](https://github.com/krmayankk))

    * Deployments are updated to use (1) a more stable hashing algorithm (fnv) than the previous one (adler) and (2) a hashing collision avoidance mechanism that will ensure new rollouts will not block on hashing collisions anymore. ([#44774](https://github.com/kubernetes/kubernetes/pull/44774), [@kargakis](https://github.com/kargakis))([kubernetes/features#287](https://github.com/kubernetes/features/issues/287))

    * PodDisruptionBudget now has a MaxUnavailable field. This allows users to specify the maximum number of disruptions that can be tolerated during eviction. ([#45587](https://github.com/kubernetes/kubernetes/pull/45587), [@foxish](https://github.com/foxish))([kubernetes/features#285](https://github.com/kubernetes/features/issues/285))

    * Deployments and DaemonSets rollouts are considered complete when all of the desired replicas are updated and available. This change affects kubectl rollout status and Deployment condition. ([#44672](https://github.com/kubernetes/kubernetes/pull/44672), [@kargakis](https://github.com/kargakis))

    * Job controller now respects ControllerRef to avoid fighting over Pods. ([#42176](https://github.com/kubernetes/kubernetes/pull/42176), [@enisoc](https://github.com/enisoc))

    * CronJob controller now respects ControllerRef to avoid fighting with other controllers. ([#42177](https://github.com/kubernetes/kubernetes/pull/42177), [@enisoc](https://github.com/enisoc))

* Auth

    * Features:

        * Azure plugin for client auth ([#43987](https://github.com/kubernetes/kubernetes/pull/43987), [@cosmincojocar](https://github.com/cosmincojocar))

        * Permission to use a PodSecurityPolicy can now be granted within a single namespace by allowing the use verb on the podsecuritypolicies resource within the namespace. ([#42360](https://github.com/kubernetes/kubernetes/pull/42360), [@liggitt](https://github.com/liggitt))

        * Break the 'certificatesigningrequests' controller into a 'csrapprover' controller and 'csrsigner' controller. ([#45514](https://github.com/kubernetes/kubernetes/pull/45514), [@mikedanese](https://github.com/mikedanese))

        * kube-apiserver: --service-account-lookup now defaults to true, requiring the Secret API object containing the token to exist in order for a service account token to be valid. This enables service account tokens to be revoked by deleting the Secret object containing the token. ([#44071](https://github.com/kubernetes/kubernetes/pull/44071), [@liggitt](https://github.com/liggitt))

        * kubectl auth cani now supports non-resource URLs ([#46432](https://github.com/kubernetes/kubernetes/pull/46432), [@CaoShuFeng](https://github.com/CaoShuFeng))

        * Promote kubelet tls bootstrap to beta. Add a non-experimental flag to use it and deprecate the old flag. ([#46799](https://github.com/kubernetes/kubernetes/pull/46799), [@mikedanese](https://github.com/mikedanese))

        * Add the alpha.image-policy.k8s.io/failed-open=true annotation when the image policy webhook encounters an error and fails open. ([#46264](https://github.com/kubernetes/kubernetes/pull/46264), [@Q-Lee](https://github.com/Q-Lee))

    * Bug fixes:

        * Make gcp auth provider not to override the Auth header if it's already exits ([#45575](https://github.com/kubernetes/kubernetes/pull/45575), [@wanghaoran1988](https://github.com/wanghaoran1988))

        * The oidc client plugin has reduce round trips and fix scopes requested ([#45317](https://github.com/kubernetes/kubernetes/pull/45317), [@ericchiang](https://github.com/ericchiang))

        * API requests using impersonation now include the system:authenticated group in the impersonated user automatically. ([#44076](https://github.com/kubernetes/kubernetes/pull/44076), [@liggitt](https://github.com/liggitt))

        * RBAC role and rolebinding auto-reconciliation is now performed only when the RBAC authorization mode is enabled. ([#43813](https://github.com/kubernetes/kubernetes/pull/43813), [@liggitt](https://github.com/liggitt))

        * PodSecurityPolicy now recognizes pods that specify runAsNonRoot: false in their security context and does not overwrite the specified value ([#47073](https://github.com/kubernetes/kubernetes/pull/47073), [@Q-Lee](https://github.com/Q-Lee))

        * Tokens retrieved from Google Cloud with application default credentials will not be cached if the client fails authorization ([#46694](https://github.com/kubernetes/kubernetes/pull/46694), [@matt-tyler](https://github.com/matt-tyler))

* Autoscaling

    * Cluster Autoscaler 0.6. More information available [here](https://github.com/kubernetes/autoscaler/blob/master/cluster-autoscaler/README.md).

* CLI

    * Features

        * kubectl logs supports specifying a container name when using label selectors ([#44282](https://github.com/kubernetes/kubernetes/pull/44282), [@derekwaynecarr](https://github.com/derekwaynecarr))

        * kubectl rollout supports undo and history for DaemonSet ([#46144](https://github.com/kubernetes/kubernetes/pull/46144), [@janetkuo](https://github.com/janetkuo))

        * kubectl rollout supports status and history for StatefulSet  ([#46669](https://github.com/kubernetes/kubernetes/pull/46669), [@kow3ns](https://github.com/kow3ns)).

        * Implement kubectl get controllerrevisions ([#46655](https://github.com/kubernetes/kubernetes/pull/46655), [@janetkuo](https://github.com/janetkuo))

        * kubectl create clusterrole supports --non-resource-url ([#45809](https://github.com/kubernetes/kubernetes/pull/45809), [@CaoShuFeng](https://github.com/CaoShuFeng))

        * kubectl logs/attach supports specifying a wait timeout with --pod-running-timeout

        *  ([#41813](https://github.com/kubernetes/kubernetes/pull/41813), [@shiywang](https://github.com/shiywang))

    * New commands

        * Add kubectl config rename-context ([#46114](https://github.com/kubernetes/kubernetes/pull/46114), [@arthur0](https://github.com/arthur0))

        * Add kubectl apply edit-last-applied subcommand ([#42256](https://github.com/kubernetes/kubernetes/pull/42256), [@shiywang](https://github.com/shiywang))

    * Strategic Merge Patch

        * Reference docs now display the patch type and patch merge key used by `kubectl apply` to merge and identify unique elements in arrays.

        * kubectl edit and kubectl apply will keep the ordering of elements in merged lists ([#45980](https://github.com/kubernetes/kubernetes/pull/45980), [@mengqiy](https://github.com/mengqiy))

        * New patch directive (retainKeys) to specifying clearing fields missing from the request ([#44597](https://github.com/kubernetes/kubernetes/pull/44597), [@mengqiy](https://github.com/mengqiy))

        * Open API now includes strategic merge patch tags (previously only in go struct tags) ([#44121](https://github.com/kubernetes/kubernetes/pull/44121), [@mbohlool](https://github.com/mbohlool))

    * Plugins

        * Introduces the ability to extend kubectl by adding third-party plugins. Developer preview, please refer to the documentation for instructions about how to use it. ([#37499](https://github.com/kubernetes/kubernetes/pull/37499), [@fabianofranz](https://github.com/fabianofranz))

        * Added support for a hierarchy of kubectl plugins (a tree of plugins as children of other plugins). ([#45981](https://github.com/kubernetes/kubernetes/pull/45981), [@fabianofranz](https://github.com/fabianofranz))

            * Added exported env vars to kubectl plugins so that plugin developers have access to global flags, namespace, the plugin descriptor and the full path to the caller binary.

    * Enhancement

        * kubectl auth cani now supports non-resource URLs ([#46432](https://github.com/kubernetes/kubernetes/pull/46432), [@CaoShuFeng](https://github.com/CaoShuFeng))

        * Kubectl set selector and set subject no longer print "running in local/dry-run mode..." at the top.  The output can now be piped and interpretted as yaml or json ([#46507](https://github.com/kubernetes/kubernetes/pull/46507), [@bboreham](https://github.com/bboreham))

        * When using an in-cluster client with an empty configuration, the --namespace flag is now honored ([#46299](https://github.com/kubernetes/kubernetes/pull/46299), [@ncdc](https://github.com/ncdc))

        * The help message for missingResourceError is now generic ([#45582](https://github.com/kubernetes/kubernetes/pull/45582), [@CaoShuFeng](https://github.com/CaoShuFeng))

        * Kubectl taint node now supports label selectors ([#44740](https://github.com/kubernetes/kubernetes/pull/44740), [@ravisantoshgudimetla](https://github.com/ravisantoshgudimetla))

        * Kubectl proxy --www now logs warning when the dir is invalid  ([#44952](https://github.com/kubernetes/kubernetes/pull/44952), [@CaoShuFeng](https://github.com/CaoShuFeng))

        * Kubectl taint output has been enhanced with the operation ([#43171](https://github.com/kubernetes/kubernetes/pull/43171), [@ravisantoshgudimetla](https://github.com/ravisantoshgudimetla))

        * Kubectl --user and --cluster now support completion ([#44251](https://github.com/kubernetes/kubernetes/pull/44251), [@superbrothers](https://github.com/superbrothers))

        * Kubectl config use-context now supports completion ([#42336](https://github.com/kubernetes/kubernetes/pull/42336), [@superbrothers](https://github.com/superbrothers))

        * Kubectl version now supports --output ([#39858](https://github.com/kubernetes/kubernetes/pull/39858), [@alejandroEsc](https://github.com/alejandroEsc))

    * Printing/describe

        * Print conditions of RC/RS in 'kubectl describe' command. ([#44710](https://github.com/kubernetes/kubernetes/pull/44710), [@xiangpengzhao](https://github.com/xiangpengzhao))

        * Improved output on 'kubectl get' and 'kubectl describe' for generic objects. ([#44222](https://github.com/kubernetes/kubernetes/pull/44222), [@fabianofranz](https://github.com/fabianofranz))

        * In 'kubectl describe', find controllers with ControllerRef, instead of showing the original creator. ([#42849](https://github.com/kubernetes/kubernetes/pull/42849), [@janetkuo](https://github.com/janetkuo))

    * Bug fixes

        * Fix some false negatives in detection of meaningful conflicts during strategic merge patch with maps and lists. ([#43469](https://github.com/kubernetes/kubernetes/pull/43469), [@enisoc](https://github.com/enisoc))

        * Restored the ability of kubectl running inside a pod to consume resource files specifying a different namespace than the one the pod is running in. ([#44862](https://github.com/kubernetes/kubernetes/pull/44862), [@liggitt](https://github.com/liggitt))

        * Kubectl commands run inside a pod using a kubeconfig file now use the namespace specified in the kubeconfig file, instead of using the pod namespace. If no kubeconfig file is used, or the kubeconfig does not specify a namespace, the pod namespace is still used as a fallback. ([#44570](https://github.com/kubernetes/kubernetes/pull/44570), [@liggitt](https://github.com/liggitt))

        * Fixed kubectl cluster-info dump to support multi-container pod. ([#44088](https://github.com/kubernetes/kubernetes/pull/44088), [@xingzhou](https://github.com/xingzhou))

        * Kubectl will print a warning when deleting the current context ([#42538](https://github.com/kubernetes/kubernetes/pull/42538), [@adohe](https://github.com/adohe))

        * Fix VolumeClaims/capacity in "kubectl describe statefulsets" output. ([#47573](https://github.com/kubernetes/kubernetes/pull/47573), [@k82cn](https://github.com/k82cn))

    * I18n

        * Add Traditional Chinese translation for kubectl ([#46559](https://github.com/kubernetes/kubernetes/pull/46559), [@warmchang](https://github.com/warmchang))

        * Add Japanese translation for kubectl ([#46756](https://github.com/kubernetes/kubernetes/pull/46756), [@girikuncoro](https://github.com/girikuncoro))

        * Add Simplified Chinese translation for kubectl ([#45573](https://github.com/kubernetes/kubernetes/pull/45573), [@shiywang](https://github.com/shiywang))

* Cluster Lifecycle

    * Features:

        * kubeadm: Enable the Node Authorizer/Admission plugin in v1.7 ([#46879](https://github.com/kubernetes/kubernetes/pull/46879), [@luxas](https://github.com/luxas))

        * kubeadm: Users can now pass extra parameters to etcd in a kubeadm cluster ([#42246](https://github.com/kubernetes/kubernetes/pull/42246), [@jamiehannaford](https://github.com/jamiehannaford))

        * kubeadm: Make kubeadm use the new CSR approver in v1.7 ([#46864](https://github.com/kubernetes/kubernetes/pull/46864), [@luxas](https://github.com/luxas))

        * kubeadm: Allow enabling multiple authorization modes at the same time ([#42557](https://github.com/kubernetes/kubernetes/pull/42557), [@xilabao](https://github.com/xilabao))

        * kubeadm: add proxy client-certs to kube-apiserver to allow it to proxy aggregated api servers ([#43715](https://github.com/kubernetes/kubernetes/pull/43715), [@deads2k](https://github.com/deads2k))

    * Action required when upgrading:

        * kubeadm: Modifications to cluster-internal resources installed by kubeadm will be overwritten when upgrading from v1.6 to v1.7. ([#47081](https://github.com/kubernetes/kubernetes/pull/47081), [@luxas](https://github.com/luxas))

        * kubeadm deb/rpm packages: cAdvisor doesn't listen on `0.0.0.0:4194` without authentication/authorization because of the possible information leakage. The cAdvisor API can still be accessed via `https://{node-ip}:10250/stats/`, though. ([kubernetes/release#356](https://github.com/kubernetes/release/pull/356), [@luxas](https://github.com/luxas))

    * Deletions:

        * cluster/ubuntu: Removed due to [deprecation](https://github.com/kubernetes/kubernetes/tree/master/cluster#cluster-configuration) and lack of maintenance. ([#44344](https://github.com/kubernetes/kubernetes/pull/44344), [@mikedanese](https://github.com/mikedanese))

        * cluster/aws: Removed due to [deprecation](https://github.com/kubernetes/kubernetes/pull/38772) and lack of maintenance. ([#42196](https://github.com/kubernetes/kubernetes/pull/42196), [@zmerlynn](https://github.com/zmerlynn))

* Federation

    * TODO: Sig-federation please find notable changes (feature & bug fix) below for your SIG and pull into this section

* Instrumentation

    * Bumped Heapster to v1.4.0. More details about the release https://github.com/kubernetes/heapster/releases/tag/v1.4.0

    * Fluentd manifest pod is no longer created on non-registered master when creating clusters using kube-up.sh. ([#44721](https://github.com/kubernetes/kubernetes/pull/44721), [@piosz](https://github.com/piosz))

    * Stackdriver cluster logging now deploys a new component to export Kubernetes events. ([#46700](https://github.com/kubernetes/kubernetes/pull/46700), [@crassirostris](https://github.com/crassirostris))

    * Stackdriver Logging deployment exposes metrics on node port 31337 when enabled. ([#47402](https://github.com/kubernetes/kubernetes/pull/47402), [@crassirostris](https://github.com/crassirostris))

* Network

    * DNS and name resolution

        * Updates kube-dns to 1.14.2 ([#45684](https://github.com/kubernetes/kubernetes/pull/45684), [@bowei](https://github.com/bowei))

            * Support kube-master-url flag without kubeconfig

            * Fix concurrent R/Ws in dns.go

            * Fix confusing logging when initializing server

            * Fix printf in cmd/kube-dns/app/server.go

            * Fix version on startup and --version flag

            * Support specifying port number for nameserver in stubDomains

        * A new field hostAliases has been added to pod.spec to support adding entries to a Pod's /etc/hosts file. ([#44641](https://github.com/kubernetes/kubernetes/pull/44641), [@rickypai](https://github.com/rickypai))

        * Fix DNS suffix search list support in Windows kube-proxy. ([#45642](https://github.com/kubernetes/kubernetes/pull/45642), [@JiangtianLi](https://github.com/JiangtianLi))

    * Kube Proxy

        * kube-proxy: ratelimit runs of iptables by sync-period flags ([#46266](https://github.com/kubernetes/kubernetes/pull/46266), [@thockin](https://github.com/thockin))

        * Fixes a bug in the kubernetes-worker Juju charm code that attempted to give kube-proxy more than one api endpoint. ([#44677](https://github.com/kubernetes/kubernetes/pull/44677), [@wwwtyro](https://github.com/wwwtyro))

    * IP masquerade

        * ip-masq-agent is now on by default for GCE ([#47794](https://github.com/kubernetes/kubernetes/pull/47794), [@dnardo](https://github.com/dnardo))

        * Bump the memory request/limit for ip-masq-daemon. ([#47887](https://github.com/kubernetes/kubernetes/pull/47887), [@dnardo](https://github.com/dnardo))

    * Cloud provider: AWS

        * New 'service.beta.kubernetes.io/aws-load-balancer-extra-security-groups' Service annotation to specify extra Security Groups to be added to ELB created by AWS cloudprovider ([#45268](https://github.com/kubernetes/kubernetes/pull/45268), [@redbaron](https://github.com/redbaron))

        * Clean up blackhole routes when using kubenet ([#47572](https://github.com/kubernetes/kubernetes/pull/47572), [@justinsb](https://github.com/justinsb))

        * Maintain a cache of all instances, to fix problem with > 200 nodes with ELBs ([#47410](https://github.com/kubernetes/kubernetes/pull/47410), [@justinsb](https://github.com/justinsb))

        * Avoid spurious ELB listener recreation - ignore case when matching protocol ([#47391](https://github.com/kubernetes/kubernetes/pull/47391), [@justinsb](https://github.com/justinsb))

        * Allow configuration of a single security group for ELBs ([#45500](https://github.com/kubernetes/kubernetes/pull/45500), [@nbutton23](https://github.com/nbutton23))

        * Remove check that forces loadBalancerSourceRanges to be 0.0.0.0/0. ([#38636](https://github.com/kubernetes/kubernetes/pull/38636), [@dhawal55](https://github.com/dhawal55))

    * Cloud provider: Azure

        * Add support for UDP ports ([#45523](https://github.com/kubernetes/kubernetes/pull/45523), [@colemickens](https://github.com/colemickens))

        * Fix support for multiple loadBalancerSourceRanges ([#45523](https://github.com/kubernetes/kubernetes/pull/45523), [@colemickens](https://github.com/colemickens))

        * Support the Service spec's sessionAffinity ([#45523](https://github.com/kubernetes/kubernetes/pull/45523), [@colemickens](https://github.com/colemickens))

    * Cloud provider: GCE

        * Bump GLBC version to 0.9.5 - fixes [loss of manually modified GCLB health check settings](https://github.com/kubernetes/kubernetes/issues/47559) upon upgrade from pre-1.6.4 to either 1.6.4 or 1.6.5. ([#47567](https://github.com/kubernetes/kubernetes/pull/47567), [@nicksardo](https://github.com/nicksardo))

        * [beta] Support creation of GCP Internal Load Balancers from Service objects ([#46663](https://github.com/kubernetes/kubernetes/pull/46663), [@nicksardo](https://github.com/nicksardo))

        * GCE installs will now avoid IP masquerade for all RFC-1918 IP blocks, rather than just 10.0.0.0/8. This means that clusters can be created in 192.168.0.0./16 and 172.16.0.0/12 while preserving the container IPs (which would be lost before). ([#46473](https://github.com/kubernetes/kubernetes/pull/46473), [@thockin](https://github.com/thockin))

        * The Calico version included in kube-up for GCE has been updated to v2.2. ([#38169](https://github.com/kubernetes/kubernetes/pull/38169), [@caseydavenport](https://github.com/caseydavenport))

        * Add ip-masq-agent addon to the addons folder which is used in GCE if --non-masquerade-cidr is set to 0/0 ([#46038](https://github.com/kubernetes/kubernetes/pull/46038), [@dnardo](https://github.com/dnardo))

        * Adds support for allocation of pod IPs via IP aliases. ([#42147](https://github.com/kubernetes/kubernetes/pull/42147), [@bowei](https://github.com/bowei))

    * Cloud provider vSphere:

        *  cloud provider: Filter out IPV6 node addresses. ([#45181](https://github.com/kubernetes/kubernetes/pull/45181), [@BaluDontu](https://github.com/BaluDontu))

    * Cloud provider OpenStack:

        * Fixes issue during LB creation where ports were incorrectly assigned to a floating IP ([#44387](https://github.com/kubernetes/kubernetes/pull/44387), [@jamiehannaford](https://github.com/jamiehannaford))

    * Network policy

        * Add short name "netpol" for networkpolicies ([#42241](https://github.com/kubernetes/kubernetes/pull/42241), [@xiangpengzhao](https://github.com/xiangpengzhao))

    * ServiceController

        * Exclude nodes labeled as master from LoadBalancer / NodePort; restores documented behaviour. ([#44745](https://github.com/kubernetes/kubernetes/pull/44745), [@justinsb](https://github.com/justinsb))

        * Fix node selection logic on initial LB creation ([#45773](https://github.com/kubernetes/kubernetes/pull/45773), [@justinsb](https://github.com/justinsb))

    * Security updates

        * Update kube-dns and ip-masq-agent addons with new base images containing fixes for CVE-2016-4448, CVE-2016-9841, CVE-2016-9843, CVE-2017-1000366, CVE-2017-2616, and CVE-2017-9526. ([#47877](https://github.com/kubernetes/kubernetes/pull/47877), [@ixdy](https://github.com/ixdy))

    * Source IP preservation

        * Promotes Source IP preservation for Virtual IPs to GA. ([#41162](https://github.com/kubernetes/kubernetes/pull/41162), [@MrHohn](https://github.com/MrHohn)). Two annotations have been promoted to API fields:

            * Service.Spec.ExternalTrafficPolicy was 'service.beta.kubernetes.io/external-traffic' annotation.

            * Service.Spec.HealthCheckNodePort was 'service.beta.kubernetes.io/healthcheck-nodeport' annotation.

        * When switching from the service.beta.kubernetes.io/external-traffic annotation to the new externalTrafficPolicy field, the values change as follows: * "OnlyLocal" becomes "Local" * "Global" becomes "Cluster". ([#46716](https://github.com/kubernetes/kubernetes/pull/46716), [@thockin](https://github.com/thockin))

    * Documentation

        * Documented NodePort networking for CDK. ([#44863](https://github.com/kubernetes/kubernetes/pull/44863), [@chuckbutler](https://github.com/chuckbutler))

* Node

    * Features

        * Kubelet Container Runtime Interface (CRI) support

            * [GA] The Docker-CRI implementation is GA. The legacy, non-CRI Docker integration has been completely removed.

            * [beta] [CRI-O](https://github.com/kubernetes-incubator/cri-o) v1.0.0-alpha.0. It has passed all e2e tests. ([release notes](https://github.com/kubernetes-incubator/cri-o/releases/tag/v1.0.0-alpha.0))

            * [beta] [Frakti](https://github.com/kubernetes/frakti) v1.0. It has passed all node conformance tests. ([release notes](https://github.com/kubernetes/frakti/releases/tag/v1.0))

            * [alpha] [cri-containerd](https://github.com/kubernetes-incubator/cri-containerd) v0.1.0. It supports basic sandbox/container/image management and passes 30/36 CRI validation test. ([release notes](https://github.com/kubernetes-incubator/cri-containerd/releases/tag/v0.1.0))

        * Support running Ubuntu image on GCE node ([#44744](https://github.com/kubernetes/kubernetes/pull/44744), [@yguo0905](https://github.com/yguo0905))

        * Kubernetes now shares a single PID namespace among all containers in a pod when running with docker >= 1.13.1. This means processes can now signal processes in other containers in a pod, but it also means that the kubectl exec {pod} kill 1 pattern will cause the pod to be restarted rather than a single container. ([#45236](https://github.com/kubernetes/kubernetes/pull/45236), [@verb](https://github.com/verb))

        * A new field hostAliases has been added to the pod spec to support adding entries to a Pod's /etc/hosts file. ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4080))([#44641](https://github.com/kubernetes/kubernetes/pull/44641), [@rickypai](https://github.com/rickypai))

        * With --feature-gates=RotateKubeletClientCertificate=true set, the Kubelet will ([#41912](https://github.com/kubernetes/kubernetes/pull/41912), [@jcbsmpsn](https://github.com/jcbsmpsn))

            * request a client certificate from the API server during the boot cycle and pause

            * waiting for the request to be satisfied. It will continually refresh the certificate

        * Create clusters with GPUs in GCE by specifying `type=<gpu-type>,count=<gpu-count>` to NODE_ACCELERATORS environment variable. ([#45130](https://github.com/kubernetes/kubernetes/pull/45130), [@vishh](https://github.com/vishh))

            * List of available GPUs - [https://cloud.google.com/compute/docs/gpus/#introduction](https://cloud.google.com/compute/docs/gpus/#introduction)

        * Disk Pressure triggers the deletion of terminated containers on the node. ([#45896](https://github.com/kubernetes/kubernetes/pull/45896), [@dashpole](https://github.com/dashpole))

        * Support status.hostIP in downward API ([#42717](https://github.com/kubernetes/kubernetes/pull/42717), [@andrewsykim](https://github.com/andrewsykim))

        * Upgrade Node Problem Detector to v0.4.1. New features added:

            * Add /dev/kmsg support for kernel log parsing. ([#112](https://github.com/kubernetes/node-problem-detector/pull/112), [@euank](https://github.com/euank))

            * Add ABRT support. ([#105](https://github.com/kubernetes/node-problem-detector/pull/105), [@juliusmilan](https://github.com/juliusmilan))

            * Add a docker image corruption problem detection in the default docker monitor config. ([#117](https://github.com/kubernetes/node-problem-detector/pull/117), [@ajitak](https://github.com/ajitak))

        * Upgrade CAdvisor to v0.26.1. New features added:

            * Add Docker overlay2 storage driver support.

            * Add ZFS support.

            * Add UDP metrics (collection disabled by default).

    * Bug fixes

        * Changed Kubelet default image-gc-high-threshold to 85% to resolve a conflict with default settings in docker that prevented image garbage collection from resolving low disk space situations when using devicemapper storage. ([#40432](https://github.com/kubernetes/kubernetes/pull/40432), [@sjenning](https://github.com/sjenning))

        * Mark all static pods on the Master node as critical to prevent preemption ([#47356](https://github.com/kubernetes/kubernetes/pull/47356), [@dashpole](https://github.com/dashpole))

        * Restrict active deadline seconds max allowed value to be maximum uint32 to avoid overflow ([#46640](https://github.com/kubernetes/kubernetes/pull/46640), [@derekwaynecarr](https://github.com/derekwaynecarr))

        * Fix a bug with cAdvisorPort in the KubeletConfiguration that prevented setting it to 0, which is in fact a valid option, as noted in issue [#11710](https://github.com/kubernetes/kubernetes/pull/11710). ([#46876](https://github.com/kubernetes/kubernetes/pull/46876), [@mtaufen](https://github.com/mtaufen))

        * Fix a bug where container cannot run as root when SecurityContext.RunAsNonRoot is false. ([#47009](https://github.com/kubernetes/kubernetes/pull/47009), [@yujuhong](https://github.com/yujuhong))

        * Fix the Kubelet PLEG update timestamp to better reflect the health of the component when the container runtime request hangs. ([#45496](https://github.com/kubernetes/kubernetes/pull/45496), [@andyxning](https://github.com/andyxning))

        * Avoid failing sync loop health check on container runtime errors ([#47124](https://github.com/kubernetes/kubernetes/pull/47124), [@andyxning](https://github.com/andyxning))

        * Fix a bug where Kubelet does not ignore pod manifest files starting with dots ([#45111](https://github.com/kubernetes/kubernetes/pull/45111), [@dwradcliffe](https://github.com/dwradcliffe))

    * Changes/deprecations

        * Marks the Kubelet's `--master-service-namespace` flag deprecated ([#44250](https://github.com/kubernetes/kubernetes/pull/44250), [@mtaufen](https://github.com/mtaufen))

        * Remove PodSandboxStatus.Linux.Namespaces.Network from CRI since it is not used/needed. ([#45166](https://github.com/kubernetes/kubernetes/pull/45166), [@feiskyer](https://github.com/feiskyer))

        * Remove the `--enable-cri` flag. CRI is now the default, and the only way to integrate with Kubelet for the container runtimes.([#45194](https://github.com/kubernetes/kubernetes/pull/45194), [@yujuhong](https://github.com/yujuhong))

        * CRI has been moved to package pkg/kubelet/apis/cri/v1alpha1/runtime as part of Kubelet API path cleanup. ([#47113](https://github.com/kubernetes/kubernetes/pull/47113), [@feiskyer](https://github.com/feiskyer))

        * Add generic Toleration for NoExecute Taints to NodeProblemDetector, so that NPD can be scheduled to nodes with NoExecute taints by default. ([#45883](https://github.com/kubernetes/kubernetes/pull/45883), [@gmarek](https://github.com/gmarek))

* Scheduling

    * NOTE:  If you are using AffinityInAnnotations=true in --feature-gates, then the 1.7 release is your last opportunity to convert from specifying affinity/anti-affinity using the scheduler.alpha.kubernetes.io/affinity annotation on Pods, to using the Affinity field of PodSpec. Support for the alpha version of node and pod affinity (which uses the scheduler.alpha.kubernetes.io/affinity annotations on Pods) is going away **in Kubernetes 1.8 **(not this release, but the next release). If you have not enabled AffinityInAnnotations=true in --feature-gates, then this change does not affect you.

    * Move hardPodAffinitySymmetricWeight to scheduler policy config ([#44159](https://github.com/kubernetes/kubernetes/pull/44159), [@wanghaoran1988](https://github.com/wanghaoran1988))

* Storage

    * Features

        * The options passed to a Flexvolume plugin's mount command now contains the pod name (kubernetes.io/pod.name), namespace (kubernetes.io/pod.namespace), uid (kubernetes.io/pod.uid), and service account name (kubernetes.io/serviceAccount.name). ([#39488](https://github.com/kubernetes/kubernetes/pull/39488), [@liggitt](https://github.com/liggitt))

        * GCE and AWS dynamic provisioners extension: admins can configure zone(s) in which a persistent volume shall be created. ([#38505](https://github.com/kubernetes/kubernetes/pull/38505), [@pospispa](https://github.com/pospispa))

        * Implement API usage metrics for GCE storage. ([#40338](https://github.com/kubernetes/kubernetes/pull/40338), [@gnufied](https://github.com/gnufied))

        * Add support for emitting metrics from openstack cloudprovider about storage operations. ([#46008](https://github.com/kubernetes/kubernetes/pull/46008), [@NickrenREN](https://github.com/NickrenREN))

        * vSphere cloud provider: vSphere storage policy support for dynamic volume provisioning. ([#46176](https://github.com/kubernetes/kubernetes/pull/46176), [@BaluDontu](https://github.com/BaluDontu))

        * Support StorageClass in Azure file volume ([#42170](https://github.com/kubernetes/kubernetes/pull/42170), [@rootfs](https://github.com/rootfs))

        * Start recording cloud provider metrics for AWS ([#43477](https://github.com/kubernetes/kubernetes/pull/43477), [@gnufied](https://github.com/gnufied))

        * Support iSCSI CHAP authentication ([#43396](https://github.com/kubernetes/kubernetes/pull/43396), [@rootfs](https://github.com/rootfs))

        * Openstack cinder v1/v2/auto API support ([#40423](https://github.com/kubernetes/kubernetes/pull/40423), [@mkutsevol](https://github.com/mkutsevol))

    * Bug-Fixes

        * Fixes issue with Flexvolume, introduced in 1.6.0, where drivers without an attacher would fail (node indefinitely waiting for attach). A driver API addition is introduced: drivers that don't implement attach should return attach: false on init. ([#47503](https://github.com/kubernetes/kubernetes/pull/47503), [@chakri-nelluri](https://github.com/chakri-nelluri))

        * Fix dynamic provisioning of PVs with inaccurate AccessModes by refusing to provision when PVCs ask for AccessModes that can't be satisfied by the PVs' underlying volume plugin. ([#47274](https://github.com/kubernetes/kubernetes/pull/47274), [@wongma7](https://github.com/wongma7))

        * Fix pods failing to start if they specify a file as a volume subPath to mount. ([#45623](https://github.com/kubernetes/kubernetes/pull/45623), [@wongma7](https://github.com/wongma7))

        * Fix erroneous FailedSync and FailedMount events being periodically and indefinitely posted on Pods after kubelet is restarted. ([#44781](https://github.com/kubernetes/kubernetes/pull/44781), [@wongma7](https://github.com/wongma7))

        * Fix AWS EBS volumes not getting detached from node if routine to verify volumes are attached runs while the node is down ([#46463](https://github.com/kubernetes/kubernetes/pull/46463), [@wongma7](https://github.com/wongma7))

        * vSphere cloud provider: Fix volume detach on node failure. ([#45569](https://github.com/kubernetes/kubernetes/pull/45569), [@divyenpatel](https://github.com/divyenpatel))

        * Improves performance of Cinder volume attach/detach operations. ([#41785](https://github.com/kubernetes/kubernetes/pull/41785), [@jamiehannaford](https://github.com/jamiehannaford))

        * AWS: Process disk attachments even with duplicate NodeNames ([#47406](https://github.com/kubernetes/kubernetes/pull/47406), [@justinsb](https://github.com/justinsb))

        * Fix iSCSI iSER mounting. ([#47281](https://github.com/kubernetes/kubernetes/pull/47281), [@mtanino](https://github.com/mtanino))

        * iscsi storage plugin: Fix dangling session when using multiple target portal addresses. ([#46239](https://github.com/kubernetes/kubernetes/pull/46239), [@mtanino](https://github.com/mtanino))

        * Fix log spam due to unnecessary status update when node is deleted. ([#45923](https://github.com/kubernetes/kubernetes/pull/45923), [@verult](https://github.com/verult))

        * Don't try to attach volume to new node if it is already attached to another node and the volume does not support multi-attach. ([#45346](https://github.com/kubernetes/kubernetes/pull/45346), [@codablock](https://github.com/codablock))

        * vSphere cloud provider: Report same Node IP as both internal and external. ([#45201](https://github.com/kubernetes/kubernetes/pull/45201), [@abrarshivani](https://github.com/abrarshivani))

        * detach the volume when pod is terminated ([#45286](https://github.com/kubernetes/kubernetes/pull/45286), [@gnufied](https://github.com/gnufied))

        * vSphere cloud provider: Filter out IPV6 node addresses. ([#45181](https://github.com/kubernetes/kubernetes/pull/45181), [@BaluDontu](https://github.com/BaluDontu))

        * vSphere cloud provider: Fix fetching of VM UUID on Ubuntu 16.04 and Fedora. ([#45311](https://github.com/kubernetes/kubernetes/pull/45311), [@divyenpatel](https://github.com/divyenpatel))

        * Roll up volume error messages in the kubelet sync loop. ([#44938](https://github.com/kubernetes/kubernetes/pull/44938), [@jayunit100](https://github.com/jayunit100))

        * Catch error when failed to make directory in NFS volume plugin ([#38801](https://github.com/kubernetes/kubernetes/pull/38801), [@nak3](https://github.com/nak3))

    * Attention-Required when Upgrading

        * Alpha volume provisioning is removed and default storage class should be used instead. ([#44090](https://github.com/kubernetes/kubernetes/pull/44090), [@NickrenREN](https://github.com/NickrenREN))

        * Portworx volume driver no longer has to run on the master. ([#45518](https://github.com/kubernetes/kubernetes/pull/45518), [@harsh-px](https://github.com/harsh-px))

        * Default behaviour in Cinder storageclass is changed. If availability is not specified, the zone is chosen by algorithm. It makes possible to spread stateful pods across many zones. ([#44798](https://github.com/kubernetes/kubernetes/pull/44798), [@zetaab](https://github.com/zetaab))

# Unsorted

* Update kube-dns, metadata-proxy, and fluentd-gcp, event-exporter, prometheus-to-sd, and ip-masq-agent addons with new base images containing fixes for CVE-2016-4448, CVE-2016-9841, CVE-2016-9843, CVE-2017-1000366, CVE-2017-2616, and CVE-2017-9526. ([#47877](https://github.com/kubernetes/kubernetes/pull/47877), [@ixdy](https://github.com/ixdy))

* Bump the memory request/limit for ip-masq-daemon. ([#47887](https://github.com/kubernetes/kubernetes/pull/47887), [@dnardo](https://github.com/dnardo))

* HostAliases is now parsed with hostAliases json keys to be in line with the feature's name. ([#47512](https://github.com/kubernetes/kubernetes/pull/47512), [@rickypai](https://github.com/rickypai))

* ip-masq-agent is now the default for GCE ([#47794](https://github.com/kubernetes/kubernetes/pull/47794), [@dnardo](https://github.com/dnardo))

* New 'service.beta.kubernetes.io/aws-load-balancer-extra-security-groups' Service annotation to specify extra Security Groups to be added to ELB created by AWS cloudprovider ([#45268](https://github.com/kubernetes/kubernetes/pull/45268), [@redbaron](https://github.com/redbaron))

* AWS: clean up blackhole routes when using kubenet ([#47572](https://github.com/kubernetes/kubernetes/pull/47572), [@justinsb](https://github.com/justinsb))

* Tokens retrieved from Google Cloud with application default credentials will not be cached if the client fails authorization ([#46694](https://github.com/kubernetes/kubernetes/pull/46694), [@matt-tyler](https://github.com/matt-tyler))

* Taints support in gce/salt startup scripts. ([#47632](https://github.com/kubernetes/kubernetes/pull/47632), [@mwielgus](https://github.com/mwielgus))

* Fix VolumeClaims/capacity in "kubectl describe statefulsets" output. ([#47573](https://github.com/kubernetes/kubernetes/pull/47573), [@k82cn](https://github.com/k82cn))

* The protobuf serialization of API objects has been updated to store maps in a predictable order to ensure that the representation of that object does not change when saved into etcd. This prevents the same object from being seen as being modified, even when no values have changed. ([#47701](https://github.com/kubernetes/kubernetes/pull/47701), [@smarterclayton](https://github.com/smarterclayton))

* kubectl logs with label selector supports specifying a container name ([#44282](https://github.com/kubernetes/kubernetes/pull/44282), [@derekwaynecarr](https://github.com/derekwaynecarr))

* Adds an approval work flow to the the certificate approver that will approve certificate signing requests from kubelets that meet all the criteria of kubelet server certificates. ([#46884](https://github.com/kubernetes/kubernetes/pull/46884), [@jcbsmpsn](https://github.com/jcbsmpsn))

* AWS: Maintain a cache of all instances, to fix problem with > 200 nodes with ELBs ([#47410](https://github.com/kubernetes/kubernetes/pull/47410), [@justinsb](https://github.com/justinsb))

* Bump GLBC version to 0.9.5 - fixes [loss of manually modified GCLB health check settings](https://github.com/kubernetes/kubernetes/issues/47559) upon upgrade from pre-1.6.4 to either 1.6.4 or 1.6.5. ([#47567](https://github.com/kubernetes/kubernetes/pull/47567), [@nicksardo](https://github.com/nicksardo))

* Update cluster-proportional-autoscaler, metadata-proxy, and fluentd-gcp addons with fixes for CVE-2016-4448, CVE-2016-8859, CVE-2016-9841, CVE-2016-9843, and CVE-2017-9526. ([#47545](https://github.com/kubernetes/kubernetes/pull/47545), [@ixdy](https://github.com/ixdy))

* AWS: Batch DescribeInstance calls with nodeNames to 150 limit, to stay within AWS filter limits. ([#47516](https://github.com/kubernetes/kubernetes/pull/47516), [@gnufied](https://github.com/gnufied))

* AWS: Process disk attachments even with duplicate NodeNames ([#47406](https://github.com/kubernetes/kubernetes/pull/47406), [@justinsb](https://github.com/justinsb))

* kubefed will now configure NodeInternalIP as the federation API server endpoint when NodeExternalIP is unavailable for federation API servers exposed as NodePort services ([#46960](https://github.com/kubernetes/kubernetes/pull/46960), [@lukaszo](https://github.com/lukaszo))

* PodSecurityPolicy now recognizes pods that specify runAsNonRoot: false in their security context and does not overwrite the specified value ([#47073](https://github.com/kubernetes/kubernetes/pull/47073), [@Q-Lee](https://github.com/Q-Lee))

* Update to kube-addon-manager:v6.4-beta.2: kubectl v1.6.4 and refreshed base images ([#47389](https://github.com/kubernetes/kubernetes/pull/47389), [@ixdy](https://github.com/ixdy))

* Azure plugin for client auth ([#43987](https://github.com/kubernetes/kubernetes/pull/43987), [@cosmincojocar](https://github.com/cosmincojocar))

* AWS: Avoid spurious ELB listener recreation - ignore case when matching protocol ([#47391](https://github.com/kubernetes/kubernetes/pull/47391), [@justinsb](https://github.com/justinsb))

* gce kube-up: The Node authorization mode and NodeRestriction admission controller are now enabled ([#46796](https://github.com/kubernetes/kubernetes/pull/46796), [@mikedanese](https://github.com/mikedanese))

* update gophercloud/gophercloud dependency for reauthentication fixes ([#45545](https://github.com/kubernetes/kubernetes/pull/45545), [@stuart-warren](https://github.com/stuart-warren))

* servicecontroller: Fix node selection logic on initial LB creation ([#45773](https://github.com/kubernetes/kubernetes/pull/45773), [@justinsb](https://github.com/justinsb))

* Fix iSCSI iSER mounting. ([#47281](https://github.com/kubernetes/kubernetes/pull/47281), [@mtanino](https://github.com/mtanino))

* StorageOS Volume Driver ([#42156](https://github.com/kubernetes/kubernetes/pull/42156), [@croomes](https://github.com/croomes))

    * [StorageOS](http://www.storageos.com/) can be used as a storage provider for Kubernetes. With StorageOS, capacity from local or attached storage is pooled across the cluster, providing converged infrastructure for cloud-native applications.

* Make gcp auth provider not to override the Auth header if it's already exits ([#45575](https://github.com/kubernetes/kubernetes/pull/45575), [@wanghaoran1988](https://github.com/wanghaoran1988))

* Allow pods to opt out of PodPreset mutation via an annotation on the pod. ([#44965](https://github.com/kubernetes/kubernetes/pull/44965), [@jpeeler](https://github.com/jpeeler))

* Add Traditional Chinese translation for kubectl ([#46559](https://github.com/kubernetes/kubernetes/pull/46559), [@warmchang](https://github.com/warmchang))

* Remove Initializers from admission-control in kubernetes-master charm for pre-1.7 ([#46987](https://github.com/kubernetes/kubernetes/pull/46987), [@Cynerva](https://github.com/Cynerva))

* Added state guards to the idle_status messaging in the kubernetes-master charm to make deployment faster on initial deployment. ([#47183](https://github.com/kubernetes/kubernetes/pull/47183), [@chuckbutler](https://github.com/chuckbutler))

* Bump up Node Problem Detector version to v0.4.0, which added support of parsing log from /dev/kmsg and ABRT. ([#46743](https://github.com/kubernetes/kubernetes/pull/46743), [@Random-Liu](https://github.com/Random-Liu))

* Deprecated Binding objects in 1.7. ([#47041](https://github.com/kubernetes/kubernetes/pull/47041), [@k82cn](https://github.com/k82cn))

* Add secretbox and AES-CBC encryption modes to at rest encryption. AES-CBC is considered superior to AES-GCM because it is resistant to nonce-reuse attacks, and secretbox uses Poly1305 and XSalsa20. ([#46916](https://github.com/kubernetes/kubernetes/pull/46916), [@smarterclayton](https://github.com/smarterclayton))

* The HorizontalPodAutoscaler controller will now only send updates when it has new status information, reducing the number of writes caused by the controller. ([#47078](https://github.com/kubernetes/kubernetes/pull/47078), [@DirectXMan12](https://github.com/DirectXMan12))

* Added exponential backoff to Azure cloudprovider ([#46660](https://github.com/kubernetes/kubernetes/pull/46660), [@jackfrancis](https://github.com/jackfrancis))

* fixed HostAlias in PodSpec to allow foo.bar hostnames instead of just foo DNS labels. ([#46809](https://github.com/kubernetes/kubernetes/pull/46809), [@rickypai](https://github.com/rickypai))

* Implements rolling update for StatefulSets. Updates can be performed using the RollingUpdate, Paritioned, or OnDelete strategies. OnDelete implements the manual behavior from 1.6. status now tracks ([#46669](https://github.com/kubernetes/kubernetes/pull/46669), [@kow3ns](https://github.com/kow3ns))

    * replicas, readyReplicas, currentReplicas, and updatedReplicas. The semantics of replicas is now consistent with DaemonSet and ReplicaSet, and readyReplicas has the semantics that replicas did prior to this release.

* Add Japanese translation for kubectl ([#46756](https://github.com/kubernetes/kubernetes/pull/46756), [@girikuncoro](https://github.com/girikuncoro))

* federation: Add admission controller for policy-based placement ([#44786](https://github.com/kubernetes/kubernetes/pull/44786), [@tsandall](https://github.com/tsandall))

* Get command uses OpenAPI schema to enhance display for a resource if run with flag 'experimental-use-openapi-print-columns'. ([#46235](https://github.com/kubernetes/kubernetes/pull/46235), [@droot](https://github.com/droot))

    * An example command:

    * kubectl get pods --experimental-use-openapi-print-columns 

* add gzip compression to GET and LIST requests ([#45666](https://github.com/kubernetes/kubernetes/pull/45666), [@ilackarms](https://github.com/ilackarms))

* Alpha feature: allows users to set storage limit to isolate EmptyDir volumes. It enforces the limit by evicting pods that exceed their storage limits ([#45686](https://github.com/kubernetes/kubernetes/pull/45686), [@jingxu97](https://github.com/jingxu97))

* Adds the Categories []string field to API resources, which represents the list of group aliases (e.g. "all") that every resource belongs to. ([#43338](https://github.com/kubernetes/kubernetes/pull/43338), [@fabianofranz](https://github.com/fabianofranz))

* Promote kubelet tls bootstrap to beta. Add a non-experimental flag to use it and deprecate the old flag. ([#46799](https://github.com/kubernetes/kubernetes/pull/46799), [@mikedanese](https://github.com/mikedanese))

* Fix disk partition discovery for brtfs ([#46816](https://github.com/kubernetes/kubernetes/pull/46816), [@dashpole](https://github.com/dashpole))

    * Add ZFS support

    * Add overlay2 storage driver support

* [beta] Support creation of GCP Internal Load Balancers from Service objects ([#46663](https://github.com/kubernetes/kubernetes/pull/46663), [@nicksardo](https://github.com/nicksardo))

* Introduces status conditions to the HorizontalPodAutoscaler in autoscaling/v2alpha1, indicating the current status of a given HorizontalPodAutoscaler, and why it is or is not scaling. ([#46550](https://github.com/kubernetes/kubernetes/pull/46550), [@DirectXMan12](https://github.com/DirectXMan12))

* Support OpenAPI spec aggregation for kube-aggregator ([#46734](https://github.com/kubernetes/kubernetes/pull/46734), [@mbohlool](https://github.com/mbohlool))

* Implement kubectl rollout undo and history for DaemonSet ([#46144](https://github.com/kubernetes/kubernetes/pull/46144), [@janetkuo](https://github.com/janetkuo))

* Respect PDBs during node upgrades and add test coverage to the ServiceTest upgrade test. ([#45748](https://github.com/kubernetes/kubernetes/pull/45748), [@mml](https://github.com/mml))

* Add the alpha.image-policy.k8s.io/failed-open=true annotation when the image policy webhook encounters an error and fails open. ([#46264](https://github.com/kubernetes/kubernetes/pull/46264), [@Q-Lee](https://github.com/Q-Lee))

* Enable kubelet csr bootstrap in GCE/GKE ([#40760](https://github.com/kubernetes/kubernetes/pull/40760), [@mikedanese](https://github.com/mikedanese))

* Implement Daemonset history ([#45924](https://github.com/kubernetes/kubernetes/pull/45924), [@janetkuo](https://github.com/janetkuo))

* When switching from the service.beta.kubernetes.io/external-traffic annotation to the new ([#46716](https://github.com/kubernetes/kubernetes/pull/46716), [@thockin](https://github.com/thockin))

    * externalTrafficPolicy field, the values chnag as follows: * "OnlyLocal" becomes "Local" * "Global" becomes "Cluster".

* Fix kubelet reset liveness probe failure count across pod restart boundaries ([#46371](https://github.com/kubernetes/kubernetes/pull/46371), [@sjenning](https://github.com/sjenning))

* The gce metadata server can be hidden behind a proxy, hiding the kubelet's token. ([#45565](https://github.com/kubernetes/kubernetes/pull/45565), [@Q-Lee](https://github.com/Q-Lee))

* AWS: Allow configuration of a single security group for ELBs ([#45500](https://github.com/kubernetes/kubernetes/pull/45500), [@nbutton23](https://github.com/nbutton23))

* Allow remote admission controllers to be dynamically added and removed by administrators. External admission controllers make an HTTP POST containing details of the requested action which the service can approve or reject. ([#46388](https://github.com/kubernetes/kubernetes/pull/46388), [@lavalamp](https://github.com/lavalamp))

* iscsi storage plugin: Fix dangling session when using multiple target portal addresses. ([#46239](https://github.com/kubernetes/kubernetes/pull/46239), [@mtanino](https://github.com/mtanino))

* Duplicate recurring Events now include the latest event's Message string ([#46034](https://github.com/kubernetes/kubernetes/pull/46034), [@kensimon](https://github.com/kensimon))

* With --feature-gates=RotateKubeletClientCertificate=true set, the kubelet will ([#41912](https://github.com/kubernetes/kubernetes/pull/41912), [@jcbsmpsn](https://github.com/jcbsmpsn))

    * request a client certificate from the API server during the boot cycle and pause

    * waiting for the request to be satisfied. It will continually refresh the certificate

    * as the certificates expiration approaches.

* The Kubernetes API supports retrieving tabular output for API resources via a new mime-type application/json;as=Table;v=v1alpha1;g=meta.k8s.io. The returned object (if the server supports it) will be of type meta.k8s.io/v1alpha1 with Table, and contain column and row information related to the resource. Each row will contain information about the resource - by default it will be the object metadata, but callers can add the ?includeObject=Object query parameter and receive the full object. In the future kubectl will use this to retrieve the results of kubectl get. ([#40848](https://github.com/kubernetes/kubernetes/pull/40848), [@smarterclayton](https://github.com/smarterclayton))

* kubectl auth cani now supports non-resource URLs ([#46432](https://github.com/kubernetes/kubernetes/pull/46432), [@CaoShuFeng](https://github.com/CaoShuFeng))

* Webhook added to the API server which emits structured audit log events. ([#45919](https://github.com/kubernetes/kubernetes/pull/45919), [@ericchiang](https://github.com/ericchiang))

* kubectl edit and kubectl apply will keep the ordering of elements in merged lists ([#45980](https://github.com/kubernetes/kubernetes/pull/45980), [@mengqiy](https://github.com/mengqiy))

* [Federation][kubefed]: Use StorageClassName for etcd pvc ([#46323](https://github.com/kubernetes/kubernetes/pull/46323), [@marun](https://github.com/marun))

* Restrict active deadline seconds max allowed value to be maximum uint32 ([#46640](https://github.com/kubernetes/kubernetes/pull/46640), [@derekwaynecarr](https://github.com/derekwaynecarr))

* Implement kubectl get controllerrevisions ([#46655](https://github.com/kubernetes/kubernetes/pull/46655), [@janetkuo](https://github.com/janetkuo))

* Local storage plugin ([#44897](https://github.com/kubernetes/kubernetes/pull/44897), [@msau42](https://github.com/msau42))

* With --feature-gates=RotateKubeletServerCertificate=true set, the kubelet will ([#45059](https://github.com/kubernetes/kubernetes/pull/45059), [@jcbsmpsn](https://github.com/jcbsmpsn))

    * request a server certificate from the API server during the boot cycle and pause

    * waiting for the request to be satisfied. It will continually refresh the certificate as

    * the certificates expiration approaches.

* Add kubectl config rename-context ([#46114](https://github.com/kubernetes/kubernetes/pull/46114), [@arthur0](https://github.com/arthur0))

* Move hardPodAffinitySymmetricWeight to scheduler policy config ([#44159](https://github.com/kubernetes/kubernetes/pull/44159), [@wanghaoran1988](https://github.com/wanghaoran1988))

* AWS: support node port health check ([#43585](https://github.com/kubernetes/kubernetes/pull/43585), [@foolusion](https://github.com/foolusion))

* support replaceKeys patch strategy and directive for strategic merge patch ([#44597](https://github.com/kubernetes/kubernetes/pull/44597), [@mengqiy](https://github.com/mengqiy))

* Prevent kubelet from setting allocatable < 0 for a resource upon initial creation. ([#46516](https://github.com/kubernetes/kubernetes/pull/46516), [@derekwaynecarr](https://github.com/derekwaynecarr))

* add --non-resource-url to kubectl create clusterrole ([#45809](https://github.com/kubernetes/kubernetes/pull/45809), [@CaoShuFeng](https://github.com/CaoShuFeng))

* Add kubectl apply edit-last-applied subcommand ([#42256](https://github.com/kubernetes/kubernetes/pull/42256), [@shiywang](https://github.com/shiywang))

* Adding admissionregistration API group which enables dynamic registration of initializers and external admission webhooks. It is an alpha feature. ([#46294](https://github.com/kubernetes/kubernetes/pull/46294), [@caesarxuchao](https://github.com/caesarxuchao))

* Fix log spam due to unnecessary status update when node is deleted. ([#45923](https://github.com/kubernetes/kubernetes/pull/45923), [@verult](https://github.com/verult))

* GCE installs will now avoid IP masquerade for all RFC-1918 IP blocks, rather than just 10.0.0.0/8. This means that clusters can ([#46473](https://github.com/kubernetes/kubernetes/pull/46473), [@thockin](https://github.com/thockin))

    * be created in 192.168.0.0./16 and 172.16.0.0/12 while preserving the container IPs (which would be lost before).

* set selector and set subject no longer print "running in local/dry-run mode..." at the top, so their output can be piped as valid yaml or json ([#46507](https://github.com/kubernetes/kubernetes/pull/46507), [@bboreham](https://github.com/bboreham))

* ControllerRevision type added for StatefulSet and DaemonSet history. ([#45867](https://github.com/kubernetes/kubernetes/pull/45867), [@kow3ns](https://github.com/kow3ns))

* Bump Go version to 1.8.3 ([#46429](https://github.com/kubernetes/kubernetes/pull/46429), [@wojtek-t](https://github.com/wojtek-t))

* Upgrade Elasticsearch Addon to v5.4.0 ([#45589](https://github.com/kubernetes/kubernetes/pull/45589), [@it-svit](https://github.com/it-svit))

* PodDisruptionBudget now uses ControllerRef to decide which controller owns a given Pod, so it doesn't get confused by controllers with overlapping selectors. ([#45003](https://github.com/kubernetes/kubernetes/pull/45003), [@krmayankk](https://github.com/krmayankk))

* aws: Support for ELB tagging by users ([#45932](https://github.com/kubernetes/kubernetes/pull/45932), [@lpabon](https://github.com/lpabon))

* kube-proxy: ratelimit runs of iptables by sync-period flags ([#46266](https://github.com/kubernetes/kubernetes/pull/46266), [@thockin](https://github.com/thockin))

* Deployments are updated to use (1) a more stable hashing algorithm (fnv) than the previous one (adler) and (2) a hashing collision avoidance mechanism that will ensure new rollouts will not block on hashing collisions anymore. ([#44774](https://github.com/kubernetes/kubernetes/pull/44774), [@kargakis](https://github.com/kargakis))

* The Prometheus metrics for the kube-apiserver for tracking incoming API requests and latencies now return the subresource label for correctly attributing the type of API call. ([#46354](https://github.com/kubernetes/kubernetes/pull/46354), [@smarterclayton](https://github.com/smarterclayton))

* Add Simplified Chinese translation for kubectl ([#45573](https://github.com/kubernetes/kubernetes/pull/45573), [@shiywang](https://github.com/shiywang))

* The --namespace flag is now honored for in-cluster clients that have an empty configuration. ([#46299](https://github.com/kubernetes/kubernetes/pull/46299), [@ncdc](https://github.com/ncdc))

* oidc client plugin: reduce round trips and fix scopes requested ([#45317](https://github.com/kubernetes/kubernetes/pull/45317), [@ericchiang](https://github.com/ericchiang))

* Fix init container status reporting when active deadline is exceeded. ([#46305](https://github.com/kubernetes/kubernetes/pull/46305), [@sjenning](https://github.com/sjenning))

* Break the 'certificatesigningrequests' controller into a 'csrapprover' controller and 'csrsigner' controller. ([#45514](https://github.com/kubernetes/kubernetes/pull/45514), [@mikedanese](https://github.com/mikedanese))

* Modifies kubefed to create and the federation controller manager to use credentials associated with a service account rather than the user's credentials. ([#42042](https://github.com/kubernetes/kubernetes/pull/42042), [@perotinus](https://github.com/perotinus))

* Adds a MaxUnavailable field to PodDisruptionBudget ([#45587](https://github.com/kubernetes/kubernetes/pull/45587), [@foxish](https://github.com/foxish))

* The behavior of some watch calls to the server when filtering on fields was incorrect. If watching objects with a filter, when an update was made that no longer matched the filter a DELETE event was correctly sent. However, the object that was returned by that delete was not the (correct) version before the update, but instead, the newer version. That meant the new object was not matched by the filter. This was a regression from behavior between cached watches on the server side and uncached watches, and thus broke downstream API clients. ([#46223](https://github.com/kubernetes/kubernetes/pull/46223), [@smarterclayton](https://github.com/smarterclayton))

* 'kubefed init' now supports overriding the default etcd image name with the --etcd-image parameter. ([#46247](https://github.com/kubernetes/kubernetes/pull/46247), [@marun](https://github.com/marun))

* remove the elasticsearch template ([#45952](https://github.com/kubernetes/kubernetes/pull/45952), [@harryge00](https://github.com/harryge00))

* Adds the CustomResourceDefinition (crd) types to the kube-apiserver. These are the successors to ThirdPartyResource. See [https://github.com/kubernetes/community/blob/master/contributors/design-proposals/thirdpartyresources.md](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/thirdpartyresources.md) for more details. ([#46055](https://github.com/kubernetes/kubernetes/pull/46055), [@deads2k](https://github.com/deads2k))

* StatefulSets now include an alpha scaling feature accessible by setting the spec.podManagementPolicy field to Parallel. The controller will not wait for pods to be ready before adding the other pods, and will replace deleted pods as needed. Since parallel scaling creates pods out of order, you cannot depend on predictable membership changes within your set. ([#44899](https://github.com/kubernetes/kubernetes/pull/44899), [@smarterclayton](https://github.com/smarterclayton))

* fix kubelet event recording for selected events. ([#46246](https://github.com/kubernetes/kubernetes/pull/46246), [@derekwaynecarr](https://github.com/derekwaynecarr))

* Moved qos to api.helpers. ([#44906](https://github.com/kubernetes/kubernetes/pull/44906), [@k82cn](https://github.com/k82cn))

* OpenAPI spec is now available in protobuf binary and gzip format (with ETag support) ([#45836](https://github.com/kubernetes/kubernetes/pull/45836), [@mbohlool](https://github.com/mbohlool))

* Added support to a hierarchy of kubectl plugins (a tree of plugins as children of other plugins). ([#45981](https://github.com/kubernetes/kubernetes/pull/45981), [@fabianofranz](https://github.com/fabianofranz))

    * Added exported env vars to kubectl plugins so that plugin developers have access to global flags, namespace, the plugin descriptor and the full path to the caller binary.

* Ignored mirror pods in PodPreset admission plugin. ([#45958](https://github.com/kubernetes/kubernetes/pull/45958), [@k82cn](https://github.com/k82cn))

* Don't try to attach volume to new node if it is already attached to another node and the volume does not support multi-attach. ([#45346](https://github.com/kubernetes/kubernetes/pull/45346), [@codablock](https://github.com/codablock))

* The Calico version included in kube-up for GCE has been updated to v2.2. ([#38169](https://github.com/kubernetes/kubernetes/pull/38169), [@caseydavenport](https://github.com/caseydavenport))

* Kubelet: Fix image garbage collector attempting to remove in-use images. ([#46121](https://github.com/kubernetes/kubernetes/pull/46121), [@Random-Liu](https://github.com/Random-Liu))

* Add ip-masq-agent addon to the addons folder which is used in GCE if --non-masquerade-cidr is set to 0/0 ([#46038](https://github.com/kubernetes/kubernetes/pull/46038), [@dnardo](https://github.com/dnardo))

* Fix serialization of EnforceNodeAllocatable ([#44606](https://github.com/kubernetes/kubernetes/pull/44606), [@ivan4th](https://github.com/ivan4th))

* Add --write-config-to flag to kube-proxy to allow users to write the default configuration settings to a file. ([#45908](https://github.com/kubernetes/kubernetes/pull/45908), [@ncdc](https://github.com/ncdc))

* The NodeRestriction admission plugin limits the Node and Pod objects a kubelet can modify. In order to be limited by this admission plugin, kubelets must use credentials in the system:nodes group, with a username in the form system:node:<nodeName>. Such kubelets will only be allowed to modify their own Node API object, and only modify Pod API objects that are bound to their node. ([#45929](https://github.com/kubernetes/kubernetes/pull/45929), [@liggitt](https://github.com/liggitt))

* vSphere cloud provider: Report same Node IP as both internal and external. ([#45201](https://github.com/kubernetes/kubernetes/pull/45201), [@abrarshivani](https://github.com/abrarshivani))

* avoid concrete examples for missingResourceError ([#45582](https://github.com/kubernetes/kubernetes/pull/45582), [@CaoShuFeng](https://github.com/CaoShuFeng))

* Fix DNS suffix search list support in Windows kube-proxy. ([#45642](https://github.com/kubernetes/kubernetes/pull/45642), [@JiangtianLi](https://github.com/JiangtianLi))

* Fix the bug where StartedAt time is not reported for exited containers. ([#45977](https://github.com/kubernetes/kubernetes/pull/45977), [@yujuhong](https://github.com/yujuhong))

* Update Dashboard version to 1.6.1 ([#45953](https://github.com/kubernetes/kubernetes/pull/45953), [@maciaszczykm](https://github.com/maciaszczykm))

* Examples: fixed cassandra mirror detection that assumes an FTP site will always be presented ([#45965](https://github.com/kubernetes/kubernetes/pull/45965), [@pompomJuice](https://github.com/pompomJuice))

* Removes the deprecated kubelet flag --babysit-daemons ([#44230](https://github.com/kubernetes/kubernetes/pull/44230), [@mtaufen](https://github.com/mtaufen))

* [Federation] Automate configuring nameserver in cluster-dns for CoreDNS provider ([#42895](https://github.com/kubernetes/kubernetes/pull/42895), [@shashidharatd](https://github.com/shashidharatd))

* Add an AEAD encrypting transformer for storing secrets encrypted at rest ([#41939](https://github.com/kubernetes/kubernetes/pull/41939), [@smarterclayton](https://github.com/smarterclayton))

* Update Minio example ([#45444](https://github.com/kubernetes/kubernetes/pull/45444), [@NitishT](https://github.com/NitishT))

* [Federation] Segregate DNS related code to separate controller ([#45034](https://github.com/kubernetes/kubernetes/pull/45034), [@shashidharatd](https://github.com/shashidharatd))

* API Registration is now in beta. ([#45247](https://github.com/kubernetes/kubernetes/pull/45247), [@mbohlool](https://github.com/mbohlool))

* Allow kcm and scheduler to lock on ConfigMaps. ([#45739](https://github.com/kubernetes/kubernetes/pull/45739), [@timothysc](https://github.com/timothysc))

* Fix lint failures on kubernetes-e2e charm ([#45832](https://github.com/kubernetes/kubernetes/pull/45832), [@Cynerva](https://github.com/Cynerva))

* Mirror pods must now indicate the nodeName they are bound to on creation. The mirror pod annotation is now treated as immutable and cannot be added to an existing pod, removed from a pod, or modified. ([#45775](https://github.com/kubernetes/kubernetes/pull/45775), [@liggitt](https://github.com/liggitt))

* Updating apiserver to return UID of the deleted resource. Clients can use this UID to verify that the resource was deleted or waiting for finalizers. ([#45600](https://github.com/kubernetes/kubernetes/pull/45600), [@nikhiljindal](https://github.com/nikhiljindal))

* OwnerReferencesPermissionEnforcement admission plugin ignores pods/status. ([#45747](https://github.com/kubernetes/kubernetes/pull/45747), [@derekwaynecarr](https://github.com/derekwaynecarr))

* prevent pods/status from touching ownerreferences ([#45826](https://github.com/kubernetes/kubernetes/pull/45826), [@deads2k](https://github.com/deads2k))

* Fix lint errors in juju kubernetes master and e2e charms ([#45494](https://github.com/kubernetes/kubernetes/pull/45494), [@ktsakalozos](https://github.com/ktsakalozos))

* Ensure that autoscaling/v1 is the preferred version for API discovery when autoscaling/v2alpha1 is enabled. ([#45741](https://github.com/kubernetes/kubernetes/pull/45741), [@DirectXMan12](https://github.com/DirectXMan12))

* Promotes Source IP preservation for Virtual IPs to GA. ([#41162](https://github.com/kubernetes/kubernetes/pull/41162), [@MrHohn](https://github.com/MrHohn))

    * Two api fields are defined correspondingly:

        * Service.Spec.ExternalTrafficPolicy <- 'service.beta.kubernetes.io/external-traffic' annotation.

        * Service.Spec.HealthCheckNodePort <- 'service.beta.kubernetes.io/healthcheck-nodeport' annotation.

* the resource quota controller was not adding quota to be resynced at proper interval ([#45685](https://github.com/kubernetes/kubernetes/pull/45685), [@derekwaynecarr](https://github.com/derekwaynecarr))

* fluentd will tolerate all NoExecute Taints when run in gcp configuration. ([#45715](https://github.com/kubernetes/kubernetes/pull/45715), [@gmarek](https://github.com/gmarek))

* Added Group/Version/Kind and Action extension to OpenAPI Operations ([#44787](https://github.com/kubernetes/kubernetes/pull/44787), [@mbohlool](https://github.com/mbohlool))

* Updates kube-dns to 1.14.2 ([#45684](https://github.com/kubernetes/kubernetes/pull/45684), [@bowei](https://github.com/bowei))

        * Support kube-master-url flag without kubeconfig

        * Fix concurrent R/Ws in dns.go

        * Fix confusing logging when initialize server

        * Fix printf in cmd/kube-dns/app/server.go

        * Fix version on startup and --version flag

        * Support specifying port number for nameserver in stubDomains

* detach the volume when pod is terminated ([#45286](https://github.com/kubernetes/kubernetes/pull/45286), [@gnufied](https://github.com/gnufied))

* Don't append :443 to registry domain in the kubernetes-worker layer registry action ([#45550](https://github.com/kubernetes/kubernetes/pull/45550), [@jacekn](https://github.com/jacekn))

* AWS: Remove check that forces loadBalancerSourceRanges to be 0.0.0.0/0. ([#38636](https://github.com/kubernetes/kubernetes/pull/38636), [@dhawal55](https://github.com/dhawal55))

* azure: add support for UDP ports ([#45523](https://github.com/kubernetes/kubernetes/pull/45523), [@colemickens](https://github.com/colemickens))

    * azure: fix support for multiple loadBalancerSourceRanges

    * azure: support the Service spec's sessionAffinity

* The fix makes scheduling go routine waiting for cache (e.g. Pod) to be synced. ([#45453](https://github.com/kubernetes/kubernetes/pull/45453), [@k82cn](https://github.com/k82cn))

* vSphere cloud provider: Filter out IPV6 node addresses. ([#45181](https://github.com/kubernetes/kubernetes/pull/45181), [@BaluDontu](https://github.com/BaluDontu))

* A small clean up to remove unnecessary functions. ([#45018](https://github.com/kubernetes/kubernetes/pull/45018), [@ravisantoshgudimetla](https://github.com/ravisantoshgudimetla))

* Removed old scheduler constructor. ([#45472](https://github.com/kubernetes/kubernetes/pull/45472), [@k82cn](https://github.com/k82cn))

* vSphere cloud provider: Fix fetching of VM UUID on Ubuntu 16.04 and Fedora. ([#45311](https://github.com/kubernetes/kubernetes/pull/45311), [@divyenpatel](https://github.com/divyenpatel))

* This fixes the overflow for priorityconfig- valid range {1, 9223372036854775806}. ([#45122](https://github.com/kubernetes/kubernetes/pull/45122), [@ravisantoshgudimetla](https://github.com/ravisantoshgudimetla))

* Bump cluster autoscaler to v0.5.4, which fixes scale down issues with pods ignoring SIGTERM. ([#45483](https://github.com/kubernetes/kubernetes/pull/45483), [@mwielgus](https://github.com/mwielgus))

* Remove deprecated node address type NodeLegacyHostIP. ([#44830](https://github.com/kubernetes/kubernetes/pull/44830), [@NickrenREN](https://github.com/NickrenREN))

* UIDs and GIDs now use apimachinery types ([#44714](https://github.com/kubernetes/kubernetes/pull/44714), [@jamiehannaford](https://github.com/jamiehannaford))

* Enable basic auth username rotation for GCI ([#44590](https://github.com/kubernetes/kubernetes/pull/44590), [@ihmccreery](https://github.com/ihmccreery))

* Kubectl taint node based on label selector ([#44740](https://github.com/kubernetes/kubernetes/pull/44740), [@ravisantoshgudimetla](https://github.com/ravisantoshgudimetla))

* Scheduler perf modular extensions. ([#44770](https://github.com/kubernetes/kubernetes/pull/44770), [@ravisantoshgudimetla](https://github.com/ravisantoshgudimetla))

* Use Docker API Version instead of docker version ([#44068](https://github.com/kubernetes/kubernetes/pull/44068), [@mkumatag](https://github.com/mkumatag))

* bump(golang.org/x/oauth2): a6bd8cefa1811bd24b86f8902872e4e8225f74c4 ([#45056](https://github.com/kubernetes/kubernetes/pull/45056), [@ericchiang](https://github.com/ericchiang))

* apimachinery: make explicit that meta.KindToResource is only a guess ([#45272](https://github.com/kubernetes/kubernetes/pull/45272), [@sttts](https://github.com/sttts))

* Fixed misspelled http URL in the cluster-dns example ([#45246](https://github.com/kubernetes/kubernetes/pull/45246), [@psiwczak](https://github.com/psiwczak))

* separate discovery from the apiserver ([#43003](https://github.com/kubernetes/kubernetes/pull/43003), [@deads2k](https://github.com/deads2k))

* Remove the --secret-name flag from kubefed join, instead generating the secret name arbitrarily. ([#42513](https://github.com/kubernetes/kubernetes/pull/42513), [@perotinus](https://github.com/perotinus))

* Added InterPodAffinity unit test case with Namespace. ([#45152](https://github.com/kubernetes/kubernetes/pull/45152), [@k82cn](https://github.com/k82cn))

* Use munged semantic version for side-loaded docker tag ([#44981](https://github.com/kubernetes/kubernetes/pull/44981), [@ixdy](https://github.com/ixdy))

* Increase Dashboard's memory requests and limits ([#44712](https://github.com/kubernetes/kubernetes/pull/44712), [@maciaszczykm](https://github.com/maciaszczykm))

* Create and push a docker image for the cloud-controller-manager ([#45154](https://github.com/kubernetes/kubernetes/pull/45154), [@luxas](https://github.com/luxas))

* Align Extender's validation with prioritizers. ([#45091](https://github.com/kubernetes/kubernetes/pull/45091), [@k82cn](https://github.com/k82cn))

* Retry calls we report config changes quickly. ([#44959](https://github.com/kubernetes/kubernetes/pull/44959), [@ktsakalozos](https://github.com/ktsakalozos))

* Added CIFS PV support for Juju Charms ([#45117](https://github.com/kubernetes/kubernetes/pull/45117), [@chuckbutler](https://github.com/chuckbutler))

* add APIService conditions ([#43301](https://github.com/kubernetes/kubernetes/pull/43301), [@deads2k](https://github.com/deads2k))

* Log warning when invalid dir passed to kubectl proxy --www ([#44952](https://github.com/kubernetes/kubernetes/pull/44952), [@CaoShuFeng](https://github.com/CaoShuFeng))

* Roll up volume error messages in the kubelet sync loop. ([#44938](https://github.com/kubernetes/kubernetes/pull/44938), [@jayunit100](https://github.com/jayunit100))

* Introduces the ability to extend kubectl by adding third-party plugins. Developer preview, please refer to the documentation for instructions about how to use it. ([#37499](https://github.com/kubernetes/kubernetes/pull/37499), [@fabianofranz](https://github.com/fabianofranz))

* Fixes juju kubernetes master: 1. Get certs from a dead leader. 2. Append tokens. ([#43620](https://github.com/kubernetes/kubernetes/pull/43620), [@ktsakalozos](https://github.com/ktsakalozos))

* Use correct option name in the kubernetes-worker layer registry action ([#44921](https://github.com/kubernetes/kubernetes/pull/44921), [@jacekn](https://github.com/jacekn))

* Start recording cloud provider metrics for AWS ([#43477](https://github.com/kubernetes/kubernetes/pull/43477), [@gnufied](https://github.com/gnufied))

* Add metrics to all major gce operations {latency, errors} ([#44510](https://github.com/kubernetes/kubernetes/pull/44510), [@bowei](https://github.com/bowei))

    * The new metrics are:

    * cloudprovider_gce_api_request_duration_seconds{request, region, zone}

    * cloudprovider_gce_api_request_errors{request, region, zone}

    * request is the specific function that is used.

    * region is the target region (Will be "<n/a>" if not applicable)

    * zone is the target zone (Will be "<n/a>" if not applicable)

    * Note: this fixes some issues with the previous implementation of

    * metrics for disks:

        * Time duration tracked was of the initial API call, not the entire

    * operation.

        * Metrics label tuple would have resulted in many independent

    * histograms stored, one for each disk. (Did not aggregate well).

* Update kubernetes-e2e charm to use snaps ([#45044](https://github.com/kubernetes/kubernetes/pull/45044), [@Cynerva](https://github.com/Cynerva))

* Log the error (if any) in e2e metrics gathering step ([#45039](https://github.com/kubernetes/kubernetes/pull/45039), [@shyamjvs](https://github.com/shyamjvs))

* The proxy subresource APIs for nodes, services, and pods now support the HTTP PATCH method. ([#44929](https://github.com/kubernetes/kubernetes/pull/44929), [@liggitt](https://github.com/liggitt))

* cluster-autoscaler: Fix duplicate writing of logs. ([#45017](https://github.com/kubernetes/kubernetes/pull/45017), [@MaciekPytel](https://github.com/MaciekPytel))

* CRI: Fix StopContainer timeout ([#44970](https://github.com/kubernetes/kubernetes/pull/44970), [@Random-Liu](https://github.com/Random-Liu))

* Fixes a bug where pods were evicted even after images are successfully deleted. ([#44986](https://github.com/kubernetes/kubernetes/pull/44986), [@dashpole](https://github.com/dashpole))

* Fix some false negatives in detection of meaningful conflicts during strategic merge patch with maps and lists. ([#43469](https://github.com/kubernetes/kubernetes/pull/43469), [@enisoc](https://github.com/enisoc))

* kubernetes-master juju charm properly detects etcd-scale events and reconfigures appropriately. ([#44967](https://github.com/kubernetes/kubernetes/pull/44967), [@chuckbutler](https://github.com/chuckbutler))

* Add redirect support to SpdyRoundTripper ([#44451](https://github.com/kubernetes/kubernetes/pull/44451), [@ncdc](https://github.com/ncdc))

* Support running Ubuntu image on GCE node ([#44744](https://github.com/kubernetes/kubernetes/pull/44744), [@yguo0905](https://github.com/yguo0905))

* Send dns details only after cdk-addons are configured ([#44945](https://github.com/kubernetes/kubernetes/pull/44945), [@ktsakalozos](https://github.com/ktsakalozos))

* Added support to the pause action in the kubernetes-worker charm for new flag --delete-local-data ([#44931](https://github.com/kubernetes/kubernetes/pull/44931), [@chuckbutler](https://github.com/chuckbutler))

* Upgrade go version to v1.8 ([#41636](https://github.com/kubernetes/kubernetes/pull/41636), [@luxas](https://github.com/luxas))

* Add namespace-{list, create, delete} actions to the kubernetes-master layer ([#44277](https://github.com/kubernetes/kubernetes/pull/44277), [@jacekn](https://github.com/jacekn))

* Fix problems with scaling up the cluster when unschedulable pods have some persistent volume claims. ([#44860](https://github.com/kubernetes/kubernetes/pull/44860), [@mwielgus](https://github.com/mwielgus))

* Feature/hpa upscale downscale delay configurable ([#42101](https://github.com/kubernetes/kubernetes/pull/42101), [@Dmitry1987](https://github.com/Dmitry1987))

* Add short name "netpol" for networkpolicies ([#42241](https://github.com/kubernetes/kubernetes/pull/42241), [@xiangpengzhao](https://github.com/xiangpengzhao))

* Restored the ability of kubectl running inside a pod to consume resource files specifying a different namespace than the one the pod is running in. ([#44862](https://github.com/kubernetes/kubernetes/pull/44862), [@liggitt](https://github.com/liggitt))

* e2e: handle nil ReplicaSet in checkDeploymentRevision ([#44859](https://github.com/kubernetes/kubernetes/pull/44859), [@sttts](https://github.com/sttts))

* Fix false positive "meaningful conflict" detection for strategic merge patch with integer values. ([#44788](https://github.com/kubernetes/kubernetes/pull/44788), [@enisoc](https://github.com/enisoc))

* Documented NodePort networking for CDK. ([#44863](https://github.com/kubernetes/kubernetes/pull/44863), [@chuckbutler](https://github.com/chuckbutler))

* Deployments and DaemonSets are now considered complete once all of the new pods are up and running - affects kubectl rollout status (and ProgressDeadlineSeconds for Deployments) ([#44672](https://github.com/kubernetes/kubernetes/pull/44672), [@kargakis](https://github.com/kargakis))

* Exclude nodes labeled as master from LoadBalancer / NodePort; restores documented behaviour. ([#44745](https://github.com/kubernetes/kubernetes/pull/44745), [@justinsb](https://github.com/justinsb))

* Fixes issue during LB creation where ports where incorrectly assigned to a floating IP ([#44387](https://github.com/kubernetes/kubernetes/pull/44387), [@jamiehannaford](https://github.com/jamiehannaford))

* Remove redis-proxy.yaml sample, as the image is nowhere to be found. ([#44801](https://github.com/kubernetes/kubernetes/pull/44801), [@klausenbusk](https://github.com/klausenbusk))

* Resolves juju vsphere hostname bug showing only a single node in a scaled node-pool. ([#44780](https://github.com/kubernetes/kubernetes/pull/44780), [@chuckbutler](https://github.com/chuckbutler))

* kubectl commands run inside a pod using a kubeconfig file now use the namespace specified in the kubeconfig file, instead of using the pod namespace. If no kubeconfig file is used, or the kubeconfig does not specify a namespace, the pod namespace is still used as a fallback. ([#44570](https://github.com/kubernetes/kubernetes/pull/44570), [@liggitt](https://github.com/liggitt))

* This adds support for CNI ConfigLists, which permit plugin chaining. ([#42202](https://github.com/kubernetes/kubernetes/pull/42202), [@squeed](https://github.com/squeed))

* API requests using impersonation now include the system:authenticated group in the impersonated user automatically. ([#44076](https://github.com/kubernetes/kubernetes/pull/44076), [@liggitt](https://github.com/liggitt))

* Print conditions of RC/RS in 'kubectl describe' command. ([#44710](https://github.com/kubernetes/kubernetes/pull/44710), [@xiangpengzhao](https://github.com/xiangpengzhao))

* NONE ([#44487](https://github.com/kubernetes/kubernetes/pull/44487), [@resouer](https://github.com/resouer))

* cinder: Add support for the KVM virtio-scsi driver ([#41498](https://github.com/kubernetes/kubernetes/pull/41498), [@mikebryant](https://github.com/mikebryant))

* Disallows installation of upstream docker from PPA in the Juju kubernetes-worker charm. ([#44681](https://github.com/kubernetes/kubernetes/pull/44681), [@wwwtyro](https://github.com/wwwtyro))

* Job controller now respects ControllerRef to avoid fighting over Pods. ([#42176](https://github.com/kubernetes/kubernetes/pull/42176), [@enisoc](https://github.com/enisoc))

* CronJob controller now respects ControllerRef to avoid fighting with other controllers. ([#42177](https://github.com/kubernetes/kubernetes/pull/42177), [@enisoc](https://github.com/enisoc))

* The hyperkube image has been slimmed down and no longer includes addon manifests and other various scripts. These were introduced for the now removed docker-multinode setup system. ([#44555](https://github.com/kubernetes/kubernetes/pull/44555), [@luxas](https://github.com/luxas))

* Refactoring reorganize taints function in kubectl to expose operations ([#43171](https://github.com/kubernetes/kubernetes/pull/43171), [@ravisantoshgudimetla](https://github.com/ravisantoshgudimetla))

* The Kubernetes API server now exits if it encounters a networking failure (e.g. the networking interface hosting its address goes away) to allow a process manager (systemd/kubelet/etc) to react to the problem. Previously the server would log the failure and try again to bind to its configured address:port. ([#42272](https://github.com/kubernetes/kubernetes/pull/42272), [@marun](https://github.com/marun))

* Fixes a bug in the kubernetes-worker Juju charm code that attempted to give kube-proxy more than one api endpoint. ([#44677](https://github.com/kubernetes/kubernetes/pull/44677), [@wwwtyro](https://github.com/wwwtyro))

* Fixes a missing comma in a list of strings. ([#44678](https://github.com/kubernetes/kubernetes/pull/44678), [@wwwtyro](https://github.com/wwwtyro))

* Fix ceph-secret type to kubernetes.io/rbd in kubernetes-master charm ([#44635](https://github.com/kubernetes/kubernetes/pull/44635), [@Cynerva](https://github.com/Cynerva))

* Add support for Azure internal load balancer ([#43510](https://github.com/kubernetes/kubernetes/pull/43510), [@karataliu](https://github.com/karataliu))

* Improved output on 'kubectl get' and 'kubectl describe' for generic objects. ([#44222](https://github.com/kubernetes/kubernetes/pull/44222), [@fabianofranz](https://github.com/fabianofranz))

* Add Kubernetes 1.6 support to Juju charms ([#44500](https://github.com/kubernetes/kubernetes/pull/44500), [@Cynerva](https://github.com/Cynerva))

    * Add metric collection to charms for autoscaling

    * Update kubernetes-e2e charm to fail when test suite fails

    * Update Juju charms to use snaps

    * Add registry action to the kubernetes-worker charm

    * Add support for kube-proxy cluster-cidr option to kubernetes-worker charm

    * Fix kubernetes-master charm starting services before TLS certs are saved

    * Fix kubernetes-worker charm failures in LXD

    * Fix stop hook failure on kubernetes-worker charm

    * Fix handling of juju kubernetes-worker.restart-needed state

    * Fix nagios checks in charms

* Fixed kubectl cluster-info dump to support multi-container pod. ([#44088](https://github.com/kubernetes/kubernetes/pull/44088), [@xingzhou](https://github.com/xingzhou))

* Prints out status updates when running kubefed init ([#41849](https://github.com/kubernetes/kubernetes/pull/41849), [@perotinus](https://github.com/perotinus))

* CRI: Fix kubelet failing to start when using rkt. ([#44569](https://github.com/kubernetes/kubernetes/pull/44569), [@yujuhong](https://github.com/yujuhong))

* Remove deprecatedPublicIPs field ([#44519](https://github.com/kubernetes/kubernetes/pull/44519), [@thockin](https://github.com/thockin))

* Remove deprecated ubuntu kube-up deployment. ([#44344](https://github.com/kubernetes/kubernetes/pull/44344), [@mikedanese](https://github.com/mikedanese))

* Use OS-specific libs when computing client User-Agent in kubectl, etc. ([#44423](https://github.com/kubernetes/kubernetes/pull/44423), [@monopole](https://github.com/monopole))

* kube-apiserver now drops unneeded path information if an older version of Windows kubectl sends it. ([#44421](https://github.com/kubernetes/kubernetes/pull/44421), [@mml](https://github.com/mml))

* Extending the gc admission plugin so that a user who doesn't have delete permission of the *owner* cannot modify blockOwnerDeletion field of existing ownerReferences, or add new ownerReference with blockOwnerDeletion=true ([#43876](https://github.com/kubernetes/kubernetes/pull/43876), [@caesarxuchao](https://github.com/caesarxuchao))

* kube-apiserver: --service-account-lookup now defaults to true, requiring the Secret API object containing the token to exist in order for a service account token to be valid. This enables service account tokens to be revoked by deleting the Secret object containing the token. ([#44071](https://github.com/kubernetes/kubernetes/pull/44071), [@liggitt](https://github.com/liggitt))

* CRI: kubectl logs -f now stops following when container stops, as it did pre-CRI. ([#44406](https://github.com/kubernetes/kubernetes/pull/44406), [@Random-Liu](https://github.com/Random-Liu))

* Add completion support for --user and --cluster to kubectl ([#44251](https://github.com/kubernetes/kubernetes/pull/44251), [@superbrothers](https://github.com/superbrothers))

* dnsprovider: avoid panic if route53 fields are nil ([#44380](https://github.com/kubernetes/kubernetes/pull/44380), [@justinsb](https://github.com/justinsb))

* In 'kubectl describe', find controllers with ControllerRef, instead of showing the original creator. ([#42849](https://github.com/kubernetes/kubernetes/pull/42849), [@janetkuo](https://github.com/janetkuo))

* Heat cluster operations now support environments that have multiple Swift URLs ([#41561](https://github.com/kubernetes/kubernetes/pull/41561), [@jamiehannaford](https://github.com/jamiehannaford))

* Adds support for allocation of pod IPs via IP aliases. ([#42147](https://github.com/kubernetes/kubernetes/pull/42147), [@bowei](https://github.com/bowei))

* validateClusterInfo: use clientcmdapi.NewCluster() ([#44221](https://github.com/kubernetes/kubernetes/pull/44221), [@ncdc](https://github.com/ncdc))

* Fix corner-case with OnlyLocal Service healthchecks. ([#44313](https://github.com/kubernetes/kubernetes/pull/44313), [@thockin](https://github.com/thockin))

* Adds annotations to all Federation objects created by kubefed. ([#42683](https://github.com/kubernetes/kubernetes/pull/42683), [@perotinus](https://github.com/perotinus))

* [Federation][Kubefed] Bug fix to enable disabling federation controllers through override args ([#44209](https://github.com/kubernetes/kubernetes/pull/44209), [@irfanurrehman](https://github.com/irfanurrehman))

* [Federation] Remove deprecated federation-apiserver-kubeconfig secret ([#44287](https://github.com/kubernetes/kubernetes/pull/44287), [@shashidharatd](https://github.com/shashidharatd))

* Scheduler can recieve its policy configuration from a ConfigMap ([#43892](https://github.com/kubernetes/kubernetes/pull/43892), [@bsalamat](https://github.com/bsalamat))

* AWS cloud provider: fix support running the master with a different AWS account or even on a different cloud provider than the nodes. ([#44235](https://github.com/kubernetes/kubernetes/pull/44235), [@mrIncompetent](https://github.com/mrIncompetent))

* add rancher credential provider ([#40160](https://github.com/kubernetes/kubernetes/pull/40160), [@wlan0](https://github.com/wlan0))

* Support generating Open API extensions for strategic merge patch tags in go struct tags ([#44121](https://github.com/kubernetes/kubernetes/pull/44121), [@mbohlool](https://github.com/mbohlool))

* Aggregated used ports at the NodeInfo level for PodFitsHostPorts predicate. ([#42524](https://github.com/kubernetes/kubernetes/pull/42524), [@k82cn](https://github.com/k82cn))

* Catch error when failed to make directory in NFS volume plugin ([#38801](https://github.com/kubernetes/kubernetes/pull/38801), [@nak3](https://github.com/nak3))

* Support iSCSI CHAP authentication ([#43396](https://github.com/kubernetes/kubernetes/pull/43396), [@rootfs](https://github.com/rootfs))

* Support context completion for kubectl config use-context ([#42336](https://github.com/kubernetes/kubernetes/pull/42336), [@superbrothers](https://github.com/superbrothers))

* print warning when delete current context ([#42538](https://github.com/kubernetes/kubernetes/pull/42538), [@adohe](https://github.com/adohe))

* Add node e2e tests for hostPid ([#44119](https://github.com/kubernetes/kubernetes/pull/44119), [@feiskyer](https://github.com/feiskyer))

* kubelet: make dockershim.sock configurable ([#43914](https://github.com/kubernetes/kubernetes/pull/43914), [@ncdc](https://github.com/ncdc))

* Fix [broken service accounts when using dedicated service account key](https://github.com/kubernetes/kubernetes/issues/44285). ([#44169](https://github.com/kubernetes/kubernetes/pull/44169), [@mikedanese](https://github.com/mikedanese))

* Fix incorrect conflict errors applying strategic merge patches to resources. ([#43871](https://github.com/kubernetes/kubernetes/pull/43871), [@liggitt](https://github.com/liggitt))

* Fix [transition between NotReady and Unreachable taints](https://github.com/kubernetes/kubernetes/issues/43444). ([#44042](https://github.com/kubernetes/kubernetes/pull/44042), [@gmarek](https://github.com/gmarek))

* leader election lock based on scheduler name ([#42961](https://github.com/kubernetes/kubernetes/pull/42961), [@wanghaoran1988](https://github.com/wanghaoran1988))

* [Federation] Remove FEDERATIONS_DOMAIN_MAP references ([#43137](https://github.com/kubernetes/kubernetes/pull/43137), [@shashidharatd](https://github.com/shashidharatd))

* Fix for [federation failing to propagate cascading deletion](https://github.com/kubernetes/kubernetes/issues/44304). ([#44108](https://github.com/kubernetes/kubernetes/pull/44108), [@csbell](https://github.com/csbell))

* Fix bug with service nodeports that have no backends not being rejected, when they should be. This is not a regression vs v1.5 - it's a fix that didn't quite fix hard enough. ([#43972](https://github.com/kubernetes/kubernetes/pull/43972), [@thockin](https://github.com/thockin))

* Fix for [failure to delete federation controllers with finalizers](https://github.com/kubernetes/kubernetes/issues/43828). ([#44084](https://github.com/kubernetes/kubernetes/pull/44084), [@nikhiljindal](https://github.com/nikhiljindal))

* Fix container hostPid settings. ([#44097](https://github.com/kubernetes/kubernetes/pull/44097), [@feiskyer](https://github.com/feiskyer))

* Fixed an issue mounting the wrong secret into pods as a service account token. ([#44102](https://github.com/kubernetes/kubernetes/pull/44102), [@ncdc](https://github.com/ncdc))

* Juju: Enable GPU mode if GPU hardware detected ([#43467](https://github.com/kubernetes/kubernetes/pull/43467), [@tvansteenburgh](https://github.com/tvansteenburgh))

* get-kube-local.sh checks pods with option "--namespace=kube-system" ([#42518](https://github.com/kubernetes/kubernetes/pull/42518), [@mtanino](https://github.com/mtanino))

* Using http2 in kubeapi-load-balancer to fix kubectl exec uses ([#43625](https://github.com/kubernetes/kubernetes/pull/43625), [@mbruzek](https://github.com/mbruzek))

* AWS cloud provider: allow to set KubernetesClusterID or KubernetesClusterTag in combination with VPC. ([#42512](https://github.com/kubernetes/kubernetes/pull/42512), [@scheeles](https://github.com/scheeles))

* changed kubelet default image-gc-high-threshold to 85% to resolve a conflict with default settings in docker that prevented image garbage collection from resolving low disk space situations when using devicemapper storage. ([#40432](https://github.com/kubernetes/kubernetes/pull/40432), [@sjenning](https://github.com/sjenning))

* When creating a container using envFrom, ([#42083](https://github.com/kubernetes/kubernetes/pull/42083), [@fraenkel](https://github.com/fraenkel))

        * validate the name of the ConfigMap in a ConfigMapRef

        * validate the name of the Secret in a SecretRef

* RBAC role and rolebinding auto-reconciliation is now performed only when the RBAC authorization mode is enabled. ([#43813](https://github.com/kubernetes/kubernetes/pull/43813), [@liggitt](https://github.com/liggitt))

* Permission to use a PodSecurityPolicy can now be granted within a single namespace by allowing the use verb on the podsecuritypolicies resource within the namespace. ([#42360](https://github.com/kubernetes/kubernetes/pull/42360), [@liggitt](https://github.com/liggitt))

* Enable audit log in local cluster ([#42379](https://github.com/kubernetes/kubernetes/pull/42379), [@xilabao](https://github.com/xilabao))

* ActiveDeadlineSeconds is validated in workload controllers now, make sure it's not set anywhere (it shouldn't be set by default and having it set means your controller will restart the Pods at some point) ([#38741](https://github.com/kubernetes/kubernetes/pull/38741), [@sandflee](https://github.com/sandflee))

* azure: all clients poll duration is now 5 seconds ([#43699](https://github.com/kubernetes/kubernetes/pull/43699), [@colemickens](https://github.com/colemickens))

* addressing issue [#39427](https://github.com/kubernetes/kubernetes/pull/39427) adding a flag --output to 'kubectl version' ([#39858](https://github.com/kubernetes/kubernetes/pull/39858), [@alejandroEsc](https://github.com/alejandroEsc))

* Support secure etcd cluster for centos provider. ([#42994](https://github.com/kubernetes/kubernetes/pull/42994), [@Shawyeok](https://github.com/Shawyeok))

* Use Cluster Autoscaler 0.5.1, which fixes an issue in Cluster Autoscaler 0.5 where the cluster may be scaled up unnecessarily. Also the status of Cluster Autoscaler is now exposed in kube-system/cluster-autoscaler-status config map. ([#43745](https://github.com/kubernetes/kubernetes/pull/43745), [@mwielgus](https://github.com/mwielgus))

* Use ProviderID to address nodes in the cloudprovider ([#42604](https://github.com/kubernetes/kubernetes/pull/42604), [@wlan0](https://github.com/wlan0))

* Openstack cinder v1/v2/auto API support ([#40423](https://github.com/kubernetes/kubernetes/pull/40423), [@mkutsevol](https://github.com/mkutsevol))

* API resource discovery now includes the singularName used to refer to the resource. ([#43312](https://github.com/kubernetes/kubernetes/pull/43312), [@deads2k](https://github.com/deads2k))

* Add the ability to lock on ConfigMaps to support HA for self hosted components ([#42666](https://github.com/kubernetes/kubernetes/pull/42666), [@timothysc](https://github.com/timothysc))

* OpenStack clusters can now specify whether worker nodes are assigned a floating IP ([#42638](https://github.com/kubernetes/kubernetes/pull/42638), [@jamiehannaford](https://github.com/jamiehannaford))

* Add Host field to TCPSocketAction ([#42902](https://github.com/kubernetes/kubernetes/pull/42902), [@louyihua](https://github.com/louyihua))

* Be able to specify the timeout to wait for pod for kubectl logs/attach ([#41813](https://github.com/kubernetes/kubernetes/pull/41813), [@shiywang](https://github.com/shiywang))

* Add support for bring-your-own ip address for Services on Azure ([#42034](https://github.com/kubernetes/kubernetes/pull/42034), [@brendandburns](https://github.com/brendandburns))

    * create configmap has a new option --from-env-file that populates a configmap from file which follows a key=val format for each line. ([#38882](https://github.com/kubernetes/kubernetes/pull/38882), [@fraenkel](https://github.com/fraenkel))

        * create secret has a new option --from-env-file that populates a configmap from file which follows a key=val format for each line.

* update the signing key for percona debian and ubuntu packages ([#41186](https://github.com/kubernetes/kubernetes/pull/41186), [@dixudx](https://github.com/dixudx))

* fc: Drop multipath.conf snippet ([#36698](https://github.com/kubernetes/kubernetes/pull/36698), [@fabiand](https://github.com/fabiand))
