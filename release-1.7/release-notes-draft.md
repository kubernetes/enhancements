## Major Themes

TODO:
- align with marketing plans

## Features

Features for this release were tracked via the use of the [kubernetes/features](https://github.com/kubernetes/features) issues repo.  Each Feature issue is owned by a Special Interest Group from [kubernetes/community](https://github.com/kubernetes/community)

TODO:
- replace docs PR links with links to actual docs

- **API Machinery**
  - [alpha] Add extensible external admission control ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4092)) ([kubernetes/features#209](https://github.com/kubernetes/features/issues/209))
  - [beta] User-provided apiservers can be aggregated (served along with) the rest of the Kubernetes API ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4173)) ([kubernetes/features#263](https://github.com/kubernetes/features/issues/263))
  - [beta] ThirdPartyResource is deprecated. Please migrate to the successor, CustomResourceDefinition. ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4071)) ([kubernetes/features#95](https://github.com/kubernetes/features/issues/95))
- **Apps**
  - [alpha] StatefulSet authors should be able to relax the ordering and parallelism policies for software that can safely support rapid, out-of-order changes. ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4162)) ([kubernetes/features#272](https://github.com/kubernetes/features/issues/272))
  - [beta] Hashing collision avoidance mechanism for Deployments ([kubernetes/features#287](https://github.com/kubernetes/features/issues/287))
  - [beta] Adds a MaxUnavailable field to PodDisruptionBudget ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4140)) ([kubernetes/features#285](https://github.com/kubernetes/features/issues/285))
  - [beta] StatefulSets currently do not support upgrades, which makes it limiting for lot of Enterprise use cases. This feature will track supporting Upgrade for StatefulSets on the server side declaratively. ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4174)) ([kubernetes/features#188](https://github.com/kubernetes/features/issues/188))
  - [beta] DaemonSet history and rollback feature is now supported. ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4098)) ([kubernetes/features#124](https://github.com/kubernetes/features/issues/124))
- **Auth**
  - [alpha] Rotation of the server TLS certificate on the kubelet ([kubernetes/features#267](https://github.com/kubernetes/features/issues/267))
  - [alpha] Rotation of the client TLS certificate on the kubelet ([kubernetes/features#266](https://github.com/kubernetes/features/issues/266))
  - [alpha] Encrypt secrets stored in etcd ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4181)) ([kubernetes/features#92](https://github.com/kubernetes/features/issues/92))
  - [beta] A new Node authorization mode and NodeRestriction admission plugin, when used in combination, limit nodes' access to specific APIs, so that they may only modify their own Node API object, only modify Pod objects bound to themselves, and only retrieve secrets and configmaps referenced by pods bound to themselves ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4077)) ([kubernetes/features#279](https://github.com/kubernetes/features/issues/279))
- **Autoscaling**
  - [alpha] HPA Status Conditions ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4118)) ([kubernetes/features#264](https://github.com/kubernetes/features/issues/264))
- **Cluster Lifecycle**
  - [alpha] Support out-of-tree and out-of-process cloud providers, a.k.a pluggable cloud providers ([docs PR](https://github.com/kubernetes/kubernetes/pull/47934)) ([kubernetes/features#88](https://github.com/kubernetes/features/issues/88))
- **Federation**
  - [alpha] The federation-apiserver now supports a SchedulingPolicy admission controller that enables policy-based control over placement of federated resources ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4075)) ([kubernetes/features#250](https://github.com/kubernetes/features/issues/250))
  - [alpha] Federation ClusterSelector annotation to direct objects to federated clusters with matching labels ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4214)) ([kubernetes/features#74](https://github.com/kubernetes/features/issues/74))
- **Instrumentation**
  - [alpha] Introduces a lightweight monitoring component for serving the core resource metrics API used by the Horizontal Pod Autoscaler and other components ([docs PR](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/metrics-server.md))([kubernetes/features#271](https://github.com/kubernetes/features/issues/271))
- **Network**
  - [stable] NetworkPolicy promoted to GA ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4003))([kubernetes/features#185](https://github.com/kubernetes/features/issues/185))
  - [stable] Source IP Preservation - change Cloud load-balancer strategy to health-checks and respond to health check only on nodes that host pods for the service ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4093)) ([kubernetes/features#27](https://github.com/kubernetes/features/issues/27))
- **Node**
  - [alpha] Provide a serial of common validation test suites for Kubelet CRI ([docs PR](https://github.com/kubernetes/community/pull/725)) ([kubernetes/features#292](https://github.com/kubernetes/features/issues/292))
  - [alpha] New RPC calls are added to the container runtime interface to retrieve container metrics from the runtime ([docs PR](https://github.com/kubernetes/kubernetes/pull/45614))([kubernetes/features#290](https://github.com/kubernetes/features/issues/290))
  - [alpha] Alpha integration with containerd 1.0, which supports basic pod lifecycle and image management. (TODO: A link to cri-containerd alpha release note)([docs PR](https://github.com/kubernetes-incubator/cri-containerd/blob/master/docs/proposal.md)) ([kubernetes/features#286](https://github.com/kubernetes/features/issues/286))
  - [stable] Add support for custom `/etc/hosts` entries through PodSpec's HostAliases ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4080)) ([kubernetes/kubernetes#43632](https://github.com/kubernetes/kubernetes/issues/43632))
- **Scheduling**
  - [alpha] Support for delegating pod binding to a scheduler extender([docs PR](https://github.com/kubernetes/kubernetes/pull/41447)) ([kubernetes/features#270](https://github.com/kubernetes/features/issues/270))
  - [alpha] A mechanism for imposing a total order on pods, that determines which pods run and which go pending when the cluster is overcomitted, implemented through a priority scheme, preemption of running pods by the default scheduler, having kubelet eviction order take the priority into account, and modifications to the quota mechanism to take priority into account ([docs PR](https://github.com/kubernetes/kubernetes/pull/41447))([kubernetes/features#268](https://github.com/kubernetes/features/issues/268))
- **Storage**
  - [alpha] This feature adds capacity isolation support for local storage at node, container, and volume levels ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4145)) ([kubernetes/features#245](https://github.com/kubernetes/features/issues/245))
  - [alpha] Make locally attached (non-network attached) storage available as a persistent volume source. ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4050)) ([kubernetes/features#121](https://github.com/kubernetes/features/issues/121))
  - [stable] Volume plugin for StorageOS provides highly-available cluster-wide persistent volumes from local or attached node storage ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4095)) ([kubernetes/features#190](https://github.com/kubernetes/features/issues/190))
  - [stable] Add support for cloudprovider metrics for storage API calls ([docs PR](https://github.com/kubernetes/kubernetes.github.io/pull/4138)) ([kubernetes/features#182](https://github.com/kubernetes/features/issues/182))

## Known Issues

Populated via [v1.7.x known issues / FAQ accumulator](https://github.com/kubernetes/kubernetes/issues/46733)

- Kubectl API discovery caching may be up to 10 minutes stale ([#47977](https://github.com/kubernetes/kubernetes/issues/47977))

TODO:
- populate this based on any further issues

## Notable Changes to Existing Behavior

TODO:
- historically this has been done by human curation of PR's merged since 1.6.0, and responses to a call for comment

## Deprecations

TODO:
- historically this has been done by human curation of PR's merged since 1.6.0, and responses to a call for comment
- flags, api resources, and behaviors
- what is going away, what it is being replaced with

## Action Required Before Upgrading

TODO:
- historically this has been done by human curation of PR's merged since 1.6.0, and responses to a call for comment

## External Dependency Version Information

TODO:
- this is copy-pasted from release-1.6/release-notes-draft.md, please update for 1.7

Continuous integration builds have used the following versions of external dependencies, however, this is not a strong recommendation and users should consult an appropriate installation or upgrade guide before deciding what versions of etcd, docker or rkt to use.

* Docker versions 1.10.3, 1.11.2, 1.12.6 have been validated
  * Docker version 1.12.6 known issues
    * overlay2 driver not fully supported
    * live-restore not fully supported
    * no shared pid namespace support
  * Docker version 1.11.2 known issues
    * Kernel crash with Aufs storage driver on Debian Jessie ([#27885](https://github.com/kubernetes/kubernetes/issues/27885))
      which can be identified by the [node problem detector](http://kubernetes.io/docs/admin/node-problem/)
    * Leaked File descriptors ([#275](https://github.com/docker/containerd/issues/275))
    * Additional memory overhead per container ([#21737](https://github.com/docker/docker/issues/21737))
  * Docker 1.10.3 contains [backports provided by RedHat](https://github.com/docker/docker/compare/v1.10.3...runcom:docker-1.10.3-stable) for known issues
  * Support for Docker version 1.9.x has been removed
* rkt version 1.23.0+
  * known issues with the rkt runtime are [listed in the Getting Started Guide](http://kubernetes.io/docs/getting-started-guides/rkt/notes/)
* etcd version 3.0.17
