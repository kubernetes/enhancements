# Removing In-Tree Cloud Provider Code

## Table of Contents

<!-- toc -->
- [Terms](#terms)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Approach](#approach)
    - [Phase 1 - Moving Cloud Provider Code to Staging](#phase-1---moving-cloud-provider-code-to-staging)
    - [Phase 2 - Building CCM from Provider Repos](#phase-2---building-ccm-from-provider-repos)
    - [Phase 3 - Migrating Provider Code to Provider Repos](#phase-3---migrating-provider-code-to-provider-repos)
  - [Staging Directory](#staging-directory)
    - [Cloud Provider Instances](#cloud-provider-instances)
- [Alternatives](#alternatives)
  - [Staging Alternatives](#staging-alternatives)
    - [Git Filter-Branch](#git-filter-branch)
  - [Build Location Alternatives](#build-location-alternatives)
    - [Build K8s/K8s from within K8s/Cloud-provider](#build-k8sk8s-from-within-k8scloud-provider)
    - [Build K8s/Cloud-provider within K8s/K8s](#build-k8scloud-provider-within-k8sk8s)
  - [Config Alternatives](#config-alternatives)
    - [Use component config to determine where controllers run](#use-component-config-to-determine-where-controllers-run)
<!-- /toc -->

## Terms

- **CCM**: Cloud Controller Manager - The controller manager responsible for running cloud provider dependent logic,
such as the service and route controllers.
- **KCM**: Kubernetes Controller Manager - The controller manager responsible for running generic Kubernetes logic,
such as job and node_lifecycle controllers.
- **KAS**: Kubernetes API Server - The core api server responsible for handling all API requests for the Kubernetes
control plane. This includes things like namespace, node, pod and job resources.
- **K8s/K8s**: The core kubernetes github repository.
- **K8s/cloud-provider**: Any or all of the repos for each cloud provider. Examples include [cloud-provider-gcp](https://github.com/kubernetes/cloud-provider-gcp),
[cloud-provider-aws](https://github.com/kubernetes/cloud-provider-aws) and [cloud-provider-azure](https://github.com/kubernetes/cloud-provider-azure).
We have created these repos for each of the in-tree cloud providers. This document assumes in various places that the
cloud providers will place the relevant code in these repos. Whether this is a long-term solution to which additional
cloud providers will be added, or an incremental step toward moving out of the Kubernetes org is out of scope of this
document, and merits discussion in a broader forum and input from SIG-Architecture and Steering Committee.
- **K8s SIGs/library**: Any SIG owned repository.
- **Staging**: Staging: Separate repositories which are currently visible under the K8s/K8s repo, which contain code
considered to be safe to be vendored outside of the K8s/K8s repo and which should eventually be fully separated from
the K8s/K8s repo. Contents of Staging are prevented from depending on code in K8s/K8s which are not in Staging.
Controlled by [publishing kubernetes-rules-configmap](https://github.com/kubernetes/publishing-bot/blob/master/configs/kubernetes-rules-configmap.yaml)
- **In-tree**: code that lives in the core Kubernetes repository [k8s.io/kubernetes](https://github.com/kubernetes/kubernetes/).
- **Out-of-Tree**: code that lives in an external repository outside of [k8s.io/kubernetes](https://github.com/kubernetes/kubernetes/).

## Summary

This is a proposal outlining steps to remove "in-tree" cloud provider code from the k8s.io/kubernetes repo while being
as least disruptive to end users and other Kubernetes developers as possible.

## Motivation

Motiviation behind this effort is to allow cloud providers to develop and make releases independent from the core
Kubernetes release cycle. The de-coupling of cloud provider code allows for separation of concern between "Kubernetes core"
and the cloud providers within the ecosystem. In addition, this ensures all cloud providers in the ecosystem are integrating with
Kubernetes in a consistent and extendable way.

Having all cloud providers developed/released in their own external repos/modules will result in the following benefits:
* The core pieces of Kubernetes (kubelet, kube-apiserver, kube-controller-manager, etc) will no longer depend on cloud provider specific
APIs (and their dependencies). This results in smaller binaries and lowers the chance of security vulnerabilities via external dependencies.
* Each cloud provider is free to release features/bug fixes at their own schedule rather than relying on the core Kubernetes release cycle.

### Goals

- Remove all cloud provider specific code from the `k8s.io/kubernetes` repository with minimal disruption to end users and developers.

### Non-Goals

- Testing/validating out-of-tree provider code for all cloud providers, this should be done by each provider.

## Proposal

### Approach

In order to remove cloud provider code from `k8s.io/kubernetes`. A 3 phase approach will be taken.

1. Move all code in `k8s.io/kubernetes/pkg/cloudprovider/providers/<provider>` to `k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers/<provider>/`. This requires removing all internal dependencies in each cloud provider to `k8s.io/kubernetes`.
2. Begin to build/release the CCM from external repos (`k8s.io/cloud-provider-<provider>`) with the option to import the legacy providers from `k8s.io/legacy-cloud-providers/<provider>`. This allows the cloud-controller-manager to opt into legacy behavior in-tree (for compatibility reasons) or build new implementations of the provider. Development for cloud providers in-tree is still done in `k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers/<provider>` during this phase.
3. Delete all code in `k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers` and shift main development to `k8s.io/cloud-provider-<provider>`. External cloud provider repos can optionally still import `k8s.io/legacy-cloud-providers` but it will no longer be imported from core components in `k8s.io/kubernetes`.

#### Phase 1 - Moving Cloud Provider Code to Staging

In Phase 1, all cloud provider code in `k8s.io/kubernetes/pkg/cloudprovider/providers/<provider>` will be moved to `k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers/<provider>`. Reasons why we "stage" cloud providers as the first phase are:
* The staged legacy provider repos can be imported from the out-of-tree provider if they choose to opt into the in-tree cloud provider implementation. This allows for a smoother transition between in-tree and out-of-tree providers in cases where there are version incompatibilites between the two.
* Staging the cloud providers indicates to the community that they are slated for removal in the future.

The biggest challenge of this phase is to remove dependences to `k8s.io/kubernetes` in all the providers. This is a requirement of staging a repository and a best practice for consuming external dependencies. All other repos "staged" (`client-go`, `apimachinery`, `api`, etc) in Kubernetes follow the same pattern. The full list of internal dependencies that need to be removed can be found in issue [69585](https://github.com/kubernetes/kubernetes/issues/69585).

#### Phase 2 - Building CCM from Provider Repos

In Phase 2, cloud providers will be expected to build the cloud controller manager from their respective provider repos (`k8s.io/cloud-provider-<provider>`). Providers can choose to vendor in their legacy provider in `k8s.io/legacy-cloud-providers/<provider>`, build implementations from scratch or both. Development in-tree is still done in the staging directories under the `k8s.io/kubernetes` repo.

The kube-controller-manager will still import the cloud provider implementations in staging. The package location of the provider implementations will change because each staged directory will be "vendored" in from their respective staging directory. The only change in core components is how the cloud providers are imported and the behavior of each cloud provider should not change.

#### Phase 3 - Migrating Provider Code to Provider Repos

In Phase 3, feature development is no longer accepted in `k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers/<provider>` and development of each cloud provider should be done in their respective external repos. Only bug and security fixes are accepted in-tree during this phase. It's important that by this phase, both in-tree and out-of-tree cloud providers are tested and production ready. Ideally most Kubernetes clusters in production should be using the out-of-tree provider before in-tree support is removed. A plan to migrate existing clusters from using the `kube-controller-manager` to the `cloud-controller-manager` is currently being developed. More details soon.

External cloud providers can optionally still import providers from `k8s.io/legacy-cloud-providers` but no core components in `k8s.io/kubernetes` will import the legacy provider and the respective staging directory will be removed along with all its dependencies.

#### Phase 4 - Disabling In-Tree Providers

In Phase 4, two feature gates will be introduced to gradually disable and remove in-tree cloud providers:
1. `DisableCloudProviders` - this feature gate will disable any functionality in kube-apiserver, kube-controller-manager and kubelet related to the `--cloud-provider` component flag.
2. `DisableKubeletCloudCredentialProvider` - this feature gate will disable in-tree functionality in the kubelet to authenticate to the AWS, Azure and GCP container registries for image pull credentials.

Both of these features gates only impacts functionality tied to the `--cloud-provider` flag, specifically in-tree volume plugins are not covered. Users should refer to CSI migration efforts for these.

For alpha, the feature gates will be used for testing purposes. When enabled, tests will ensure that clusters with in-tree cloud providers disabled behaves as expected. This is targeted for v1.21 and will be
disabled by default.

For beta, the feature gates will be on by default, meaning core components will disallow use of in-tree cloud providers. This will act as a warning for users to migrate to external components. Users may
choose to continue using the in-tree provider by explicitly disabling the feature gates. Beta is targeted for v1.23 or v1.24.

For GA, the feature gate will be enabled by default and locked. Users at this point MUST migrate to external components and use of the in-tree cloud providers will be disallowed. One release after GA,
the in-tree cloud providers can be safely removed. GA is targeted for v1.25 or v1.26.

### Staging Directory

There are several sections of code which need to be shared between the K8s/K8s repo and the K8s/Cloud-provider repos.
The plan for doing that sharing is to move the relevant code into the Staging directory as that is where we share code
today. The current Staging repo has the following packages in it.
- Api
- Apiextensions-apiserver
- Apimachinery
- Apiserver
- Client-go
- Code-generator
- Kube-aggregator
- Metrics
- Sample-apiserver
- Sample-Controller

With the additions needed in the short term to make this work; the Staging area would now need to look as follows.
- Api
- Apiextensions-apiserver
- Apimachinery
- Apiserver
- Client-go
- **legacy-cloud-providers**
- Code-generator
- Kube-aggregator
- Metrics
- Sample-apiserver
- Sample-Controller

#### Cloud Provider Instances

Currently in K8s/K8s the cloud providers are actually included in the [providers.go](https://github.com/kubernetes/kubernetes/blob/master/pkg/cloudprovider/providers/providers.go)
file which then includes each of the in-tree cloud providers. In the short term, we would leave that file where it is
and adjust it to point at the new homes under Staging. For the K8s/cloud-provider repo, would have the following CCM
wrapper file. (Essentially a modified copy of cmd/cloud-controller-manager/controller-manager.go) The wrapper for each
cloud provider would import just their vendored cloud-provider implementation rather than providers.go file.

k8s/k8s: pkg/cloudprovider/providers/providers.go
```package cloudprovider

import (
  // Prior to cloud providers having been moved to Staging
  _ "k8s.io/kubernetes/pkg/cloudprovider/providers/aws"
  _ "k8s.io/kubernetes/pkg/cloudprovider/providers/azure"
  _ "k8s.io/kubernetes/pkg/cloudprovider/providers/cloudstack"
  _ "k8s.io/kubernetes/pkg/cloudprovider/providers/gce"
  _ "k8s.io/kubernetes/pkg/cloudprovider/providers/openstack"
  _ "k8s.io/kubernetes/pkg/cloudprovider/providers/ovirt"
  _ "k8s.io/kubernetes/pkg/cloudprovider/providers/photon"
  _ "k8s.io/kubernetes/pkg/cloudprovider/providers/vsphere"
)
```

k8s/cloud-provider-gcp: pkg/cloudprovider/providers/providers.go
```package cloudprovider

import (
  // Cloud providers
  _ "k8s.io/legacy-cloud-providers/aws"
  _ "k8s.io/legacy-cloud-providers/azure"
  _ "k8s.io/legacy-cloud-providers/gce"
  _ "k8s.io/legacy-cloud-providers/openstack"
  _ "k8s.io/legacy-cloud-providers/vsphere"
)
```

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: DisableCloudProviders
    - Components depending on the feature gate: kubelet, kube-apiserver, kube-controller-manager
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: DisableKubeletCloudCredentialProvider
    - Components depending on the feature gate: kubelet

* **Does enabling the feature change any default behavior?**
  Yes, enabling this feature will disable all capabilities enabled when `--cloud-provider` is set in core components.
  Users need to ensure they have migrated to out-of-tree components prior to enabling this feature gate.
  If appropriate extensions (CCM, credential provider, apiserver-network-proxy, etc) are in use, cloud provider capabilities
  should remain the same at the very least.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, the feature can be disabled once it is enabled. If disabled, users must ensure
  that the CCM is no longer running in the cluster. Credential provider plugins and the
  apiserver network proxy do not have to be stopped on rollback.

* **What happens if we reenable the feature if it was previously rolled back?**

  All capabilities from in-tree cloud providers will be re-disabled.

* **Are there any tests for feature enablement/disablement?**
  Adequate unit tests, component integration test and e2e tests will be added for this feature before
  it is goes beta and on by default.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**

  TBD for beta.

* **What specific metrics should inform a rollback?**

   TBD for beta.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

  TBD for beta.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**

  TBD for beta.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

  TBD for beta.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**

  TBD for beta.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

  TBD for beta.

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**

  TBD for beta.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**

  TBD for beta.


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**

  No, if anything it will result in reduced API calls in core components.

* **Will enabling / using this feature result in introducing new API types?**

  No.

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

  No, it will actually remove calls to the cloud provider in all core components.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**

  No.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**

  No.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**

  No. In fact, it should reduce resource usage.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

TBD for beta.

* **What are other known failure modes?**

TBD for beta.

* **What steps should be taken if SLOs are not being met to determine the problem?**

TBD for beta.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos


## Alternatives

### Staging Alternatives

#### Git Filter-Branch

One possible alternative is to make use of a Git Filter Branch to extract a sub-directory into a virtual repo. The repo
needs to be sync'd in an ongoing basis with K8s/K8s as we want one source of truth until K8s/K8s does not pull in the
code. This has issues such as not giving K8s/K8s developers any indications of what the dependencies various
K8s/Cloud-providers have. Without that information it becomes very easy to accidentally break various cloud providers
and time you change dependencies in the K8s/K8s repo. With staging the dependency line is simple and [automatically
enforced](https://github.com/kubernetes/kubernetes/blob/master/hack/verify-no-vendor-cycles.sh). Things in Staging are
not allowed to depend on things outside of Staging. If you want to add such a dependency you need to add the dependent
code to Staging. The act of doing this means that code should get synced and solve the problem. In addition the usage
of a second different library and repo movement mechanism will make things more difficult for everyone.

“Trying to share code through the git filter will not provide this protection. In addition it means that we now have
two sharing code mechanisms which increases complexity on the community and build tooling. As such I think it is better
to continue to use the Staging mechanisms. ”

### Build Location Alternatives

#### Build K8s/K8s from within K8s/Cloud-provider

The idea here is to avoid having to add a new build target to K8s/K8s. The K8s/Cloud-provider could have their own
custom targets for building things like KAS without other cloud-providers implementations linked in. It would also
allow other customizations of the standard binaries to be created. While a powerful tool, this mechanism seems to
encourage customization of these core binaries and as such to be discouraged. Providing the appropriate generic
binaries cuts down on the need to duplicate build logic for these core components and allow each optimization of build.
Download prebuilt images at a version and then just build the appropriate addons.

#### Build K8s/Cloud-provider within K8s/K8s

The idea here would be to treat the various K8s/Cloud-provider repos as libraries. You would specify a build flavor and
we would pull in the relevant code based on what you specified. This would put tight restrictions on how the
K8s/Cloud-provider repos would work as they would need to be consumed by the K8s/K8s build system. This seems less
extensible and removes the nice loose coupling which the other systems have. It also makes it difficult for the cloud
providers to control their release cadence.

### Config Alternatives

#### Use component config to determine where controllers run

Currently KCM and CCM have their configuration passed in as command line flags. If their configuration were obtained
from a configuration server (component config) then we could have a single source of truth about where each controller
should be run. This both solves the HA migration issue and other concerns about making sure that a controller only runs
in 1 controller manager. Rather than having the controllers as on or off, controllers would now be configured to state
where they should run, KCM, CCM, Nowhere, … If the KCM could handle this as a run-time change nothing would need to
change. Otherwise it becomes a slight variant of the proposed solution. This is probably the correct long term
solution. However for the timeline we are currently working with we should use the proposed solution.
