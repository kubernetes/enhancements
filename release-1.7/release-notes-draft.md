## Major Themes

TODO

## Features

Features for this release were tracked via the use of the [kubernetes/features](https://github.com/kubernetes/features) issues repo.  Each Feature issue is owned by a Special Interest Group from [kubernetes/community](https://github.com/kubernetes/community)

TODO:
- replace feature title with one-line release note

- **API Machinery**
  - [alpha] Kubernetes should be able to easily integrate external policy control  ([kubernetes/features#209](https://github.com/kubernetes/features/issues/209))
  - [beta] API Aggregation ([kubernetes/features#263](https://github.com/kubernetes/features/issues/263))
  - [beta] CustomResourceDefinitions, née ThirdPartyResources ([kubernetes/features#95](https://github.com/kubernetes/features/issues/95))
- **Apps**
  - [alpha] StatefulSets should support a burst mode for faster scale up / down ([kubernetes/features#272](https://github.com/kubernetes/features/issues/272))
  - [beta] Hashing collision avoidance mechanism for Deployments ([kubernetes/features#287](https://github.com/kubernetes/features/issues/287))
  - [beta] Add MaxUnavailable to PodDisruptionBudget ([kubernetes/features#285](https://github.com/kubernetes/features/issues/285))
  - [beta] StatefulSet Upgrades ([kubernetes/features#188](https://github.com/kubernetes/features/issues/188))
  - [beta] DaemonSet updates ([kubernetes/features#124](https://github.com/kubernetes/features/issues/124))
- **Auth**
  - [alpha] Kubelet Server TLS Certificate Rotation ([kubernetes/features#267](https://github.com/kubernetes/features/issues/267))
  - [alpha] Kubelet Client TLS Certificate Rotation ([kubernetes/features#266](https://github.com/kubernetes/features/issues/266))
  - [alpha] Encrypt secrets in etcd ([kubernetes/features#92](https://github.com/kubernetes/features/issues/92))
  - [beta] Limit node access to API ([kubernetes/features#279](https://github.com/kubernetes/features/issues/279))
- **Autoscaling**
  - [alpha] HPA Status Conditions ([kubernetes/features#264](https://github.com/kubernetes/features/issues/264))
- **Cluster Lifecycle**
  - [alpha] Support out-of-process and out-of-tree cloud providers ([kubernetes/features#88](https://github.com/kubernetes/features/issues/88))
- **Federation**
  - [alpha] Policy-based Federated Resource Placement ([kubernetes/features#250](https://github.com/kubernetes/features/issues/250))
- **Instrumentation**
  - [alpha] Metrics Server (for resource metrics API) ([kubernetes/features#271](https://github.com/kubernetes/features/issues/271))
- **Network**
  - [stable] Bring Network Policy to GA ([kubernetes/features#185](https://github.com/kubernetes/features/issues/185))
  - [stable] GCP Cloud Provider: Source IP preservation for Virtual IPs ([kubernetes/features#27](https://github.com/kubernetes/features/issues/27))
- **Node**
  - [alpha] CRI validation test suite ([kubernetes/features#292](https://github.com/kubernetes/features/issues/292))
  - [alpha] Enhance the Container Runtime Interface ([kubernetes/features#290](https://github.com/kubernetes/features/issues/290))
  - [alpha] Containerd CRI Integration ([kubernetes/features#286](https://github.com/kubernetes/features/issues/286))
- **Scheduling**
  - [alpha] Bind method in scheduler extender ([kubernetes/features#270](https://github.com/kubernetes/features/issues/270))
  - [alpha] Priority/preemption ([kubernetes/features#268](https://github.com/kubernetes/features/issues/268))
- **Storage**
  - [alpha] Capacity Isolation Resource Management ([kubernetes/features#245](https://github.com/kubernetes/features/issues/245))
  - [alpha] Durable (non-shared) local storage management ([kubernetes/features#121](https://github.com/kubernetes/features/issues/121))
  - [stable] StorageOS Volume Plugin ([kubernetes/features#190](https://github.com/kubernetes/features/issues/190))
  - [stable] Cloudprovider metrics for storage ([kubernetes/features#182](https://github.com/kubernetes/features/issues/182))

## Known Issues

Populated via [v1.7.0 known issues / FAQ accumulator](https://github.com/kubernetes/kubernetes/issues/TBD)

TODO

## Notable Changes to Existing Behavior

TODO

historically this has been done by human curation of PR's merged since 1.6.0, and responses to a call for comment

## Deprecations

TODO

flags, api resources, and behaviors
what is going away, what it is being replaced with

historically this has been done by human curation of PR's merged since 1.6.0, and responses to a call for comment

## Action Required Before Upgrading

historically this has been done by human curation of PR's merged since 1.6.0, and responses to a call for comment

## External Dependency Version Information

Continuous integration builds have used the following versions of external dependencies, however, this is not a strong recommendation and users should consult an appropriate installation or upgrade guide before deciding what versions of etcd, docker or rkt to use. 

TODO: this is copy-pasted from release-1.6/release-notes-draft.md, please update for 1.7

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

## Process

I started with the release-1.5/release-notes-draft.md file, and removed all content under the existing headings.

I use the following script to start populating the features section
```sh
for sig in $(hub issue -M 6 | sed -e 's|.*sig/\([^ ]*\).*|\1|g' | sort | uniq); do
  echo "- **$sig**"
  for stage in alpha beta stable; do 
    hub issue -M 6 -l sig/$sig,stage/$stage -f \
      "  - [$stage] %t ([kubernetes/features%i](%U))%n";
  done
done
```

I noticed the following sections in the release-1.6/release-notes-draft.md file, but have explicitly left them out here.  Please add back in if you feel they're necessary:
- WARNING: etcd backup strongly recommneded
- Changes to API Resources
- Changes to Major Components
- Changes to Cluster Provisioning Scripts
- Changes to Addons
