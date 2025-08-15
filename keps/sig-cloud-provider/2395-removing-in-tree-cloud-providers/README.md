# KEP-2395: Removing In-Tree Cloud Provider Code

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Terms](#terms)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Phase 1 - Moving Cloud Provider Code to Staging](#phase-1---moving-cloud-provider-code-to-staging)
  - [Phase 2 - Building CCM from Provider Repos](#phase-2---building-ccm-from-provider-repos)
  - [Phase 3 - Migrating Provider Code to Provider Repos](#phase-3---migrating-provider-code-to-provider-repos)
  - [Phase 4 - Disabling In-Tree Providers](#phase-4---disabling-in-tree-providers)
  - [Staging Directory](#staging-directory)
    - [Cloud Provider Instances](#cloud-provider-instances)
  - [Test Plan](#test-plan)
      - [Prerequisite testing update](#prerequisite-testing-update)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Cloud Provider Specific Guidance](#cloud-provider-specific-guidance)
    - [General Guidance](#general-guidance)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Staging Alternatives](#staging-alternatives)
    - [Git Filter-Branch](#git-filter-branch)
  - [Build Location Alternatives](#build-location-alternatives)
    - [Build K8s/K8s from within K8s/Cloud-provider](#build-k8sk8s-from-within-k8scloud-provider)
    - [Build K8s/Cloud-provider within K8s/K8s](#build-k8scloud-provider-within-k8sk8s)
  - [Config Alternatives](#config-alternatives)
    - [Use component config to determine where controllers run](#use-component-config-to-determine-where-controllers-run)
<!-- /toc -->

## Release Signoff Checklist

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

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

The motivation behind this effort is to allow cloud providers to develop and make releases independent from the core
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

In order to remove cloud provider code from `k8s.io/kubernetes`. A 4 phase approach will be taken.

1. Move all code in `k8s.io/kubernetes/pkg/cloudprovider/providers/<provider>` to `k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers/<provider>/`. This requires removing all internal dependencies in each cloud provider to `k8s.io/kubernetes`.
2. Begin to build/release the CCM from external repos (`k8s.io/cloud-provider-<provider>`) with the option to import the legacy providers from `k8s.io/legacy-cloud-providers/<provider>`. This allows the cloud-controller-manager to opt into legacy behavior in-tree (for compatibility reasons) or build new implementations of the provider. Development for cloud providers in-tree is still done in `k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers/<provider>` during this phase.
3. Migrate all code in `k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers` and shift main development to `k8s.io/cloud-provider-<provider>`. External cloud provider repos can optionally still import `k8s.io/legacy-cloud-providers` but it will no longer be imported from core components in `k8s.io/kubernetes`. Changes to `k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers` are no longer accepted unless they are critical or security-related.
4. Disable in-tree providers and set the `DisableCloudProviders` and `DisableKubeletCloudCredentialProvider` feature gates to true by default. This will enable external CCM behavior by default in Kubernetes.

### Risks and Mitigations

* Kubernetes users will need to add CCM deployments to their clusters. Previously, users were able to enable the cloud controller loops of the kubernetes-controller-manager through command line flags. With the change to external CCMs users will be responsible for managing their own CCM deployments.
* Security for the core Kubernetes cloud provider interface will continue to reviewed by the Kubernetes SIG Security community.
* Security for the external CCMs will be reviewed by the project communities which own the specific CCM implementation, with supplemental reviews done by the SIG Security community.
* UX for the core Kubernetes cloud provider interface will continue to be reviewed by the Kubernetes SIG Cloud Provider, SIG Architecture, ans SIG API Machinery communities.
* UX for the external CCMs will be reviewed by the project communities which own the specific CCM implementation.

## Design Details

### Phase 1 - Moving Cloud Provider Code to Staging

In Phase 1, all cloud provider code in `k8s.io/kubernetes/pkg/cloudprovider/providers/<provider>` will be moved to `k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers/<provider>`. Reasons why we "stage" cloud providers as the first phase are:
* The staged legacy provider repos can be imported from the out-of-tree provider if they choose to opt into the in-tree cloud provider implementation. This allows for a smoother transition between in-tree and out-of-tree providers in cases where there are version incompatibilites between the two.
* Staging the cloud providers indicates to the community that they are slated for removal in the future.

The biggest challenge of this phase is to remove dependencies to `k8s.io/kubernetes` in all the providers. This is a requirement of staging a repository and a best practice for consuming external dependencies. All other repos "staged" (`client-go`, `apimachinery`, `api`, etc) in Kubernetes follow the same pattern. The full list of internal dependencies that need to be removed can be found in issue [69585](https://github.com/kubernetes/kubernetes/issues/69585).

### Phase 2 - Building CCM from Provider Repos

In Phase 2, cloud providers will be expected to build the cloud controller manager from their respective provider repos (`k8s.io/cloud-provider-<provider>`). Providers can choose to vendor in their legacy provider in `k8s.io/legacy-cloud-providers/<provider>`, build implementations from scratch or both. Development in-tree is still done in the staging directories under the `k8s.io/kubernetes` repo.

The kube-controller-manager will still import the cloud provider implementations in staging. The package location of the provider implementations will change because each staged directory will be "vendored" in from their respective staging directory. The only change in core components is how the cloud providers are imported and the behavior of each cloud provider should not change.

### Phase 3 - Migrating Provider Code to Provider Repos

In Phase 3, feature development is no longer accepted in `k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers/<provider>` and development of each cloud provider should be done in their respective external repos. Only bug and security fixes are accepted in-tree during this phase. It's important that by this phase, both in-tree and out-of-tree cloud providers are tested and production ready. Ideally most Kubernetes clusters in production should be using the out-of-tree provider before in-tree support is removed. A plan to migrate existing clusters from using the `kube-controller-manager` to the `cloud-controller-manager` is currently being developed. More details soon.

External cloud providers can optionally still import providers from `k8s.io/legacy-cloud-providers` but no core components in `k8s.io/kubernetes` will import the legacy provider and the respective staging directory will be removed along with all its dependencies.

### Phase 4 - Disabling In-Tree Providers

In Phase 4, two feature gates will be introduced to gradually disable and remove in-tree cloud providers:
1. `DisableCloudProviders` - this feature gate will disable any functionality in kube-apiserver, kube-controller-manager and kubelet related to the `--cloud-provider` component flag.
2. `DisableKubeletCloudCredentialProvider` - this feature gate will disable in-tree functionality in the kubelet to authenticate to the AWS, Azure and GCP container registries for image pull credentials.

Both of these features gates only impacts functionality tied to the `--cloud-provider` flag, specifically in-tree volume plugins are not covered. Users should refer to CSI migration efforts for these.

For alpha, the feature gates will be used for testing purposes. When enabled, tests will ensure that clusters with in-tree cloud providers disabled behaves as expected. This is targeted for v1.21 and will be
disabled by default.

For beta, the feature gates will be on by default, meaning core components will disallow use of in-tree cloud providers. This will act as a warning for users to migrate to external components. Users may
choose to continue using the in-tree provider by explicitly disabling the feature gates. Beta is targeted for v1.29 with the caveat that a majority of our CI signal jobs across providers should have converted to use CCM by then.

For GA, the feature gate will be enabled by default and locked. Users at this point MUST migrate to external components and use of the in-tree cloud providers will be disallowed. GA is targeted for v1.31. One release after GA, the in-tree cloud providers can be safely removed.

NOTE: the removal of the code will depend on when we can remove the in-tree storage plugins, so the actual removal may end up in a later release.

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

### Test Plan

This change will leverage the tests available in the kubernetes/kuberenetes repository
that exercise cloud controller behavior. The largest change to testing for this KEP is the
default enablement of external CCMs. Test workflows have been updated to exercise the
`DisableCloudProviders` and `DisableKubeletCloudCredentialProvider` feature gates while
also engaging the `--cloud-provider=external`, and related, command line flags to kubelet,
kube-apiserver, and kube-controller-manager.

In all other respects, the expected behavior of the cloud controller managers is not changing,
and as such the original tests for in-tree cloud controllers continue to be relevant and
necessary.

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing update

The behavior of the cloud controllers is not changing with respect to cluster functioning,
as such the prerequisite for this KEP is that all the current cloud controller related
tests are passing. When the external CCMs are enabled, these tests should continue to pass.

As described in the [non-goals](#non-goals) section, testing of individual cloud provider CCMs
is not in scope for this KEP. Each provider is expected to own the tests related to their
specific cloud platform.

##### Unit tests

This KEP describes a process for extracting and migrating current in-tree code
to external repositories. The notion of unit style testing takes on a different
meaning in this perspective as the focus for the unit testing will move to those
external repositories.

SIG Cloud Provider maintains a [framework for creating CCMs][ccmframework] which
contains unit testing for the core controller loops of a CCM implementation. See:

- https://github.com/kubernetes/cloud-provider/blob/master/controllers/node/node_controller_test.go
- https://github.com/kubernetes/cloud-provider/blob/master/controllers/nodelifecycle/node_lifecycle_controller_test.go
- https://github.com/kubernetes/cloud-provider/blob/master/controllers/route/route_controller_test.go
- https://github.com/kubernetes/cloud-provider/blob/master/controllers/service/controller_test.go

Cloud providers who create external CCMs will be responsible for providing unit
testing within their own repositories.

##### Integration tests

Integration testing on this change is complicated by the fact that any testing
with concrete CCM implementations requires cloud infrastructure from the same
provider. While it would be possible to create a mock provider for these type
of scenarios, the SIG considers this testing to be a low value when compared to
the e2e test which are running on concrete providers.

For the reasons stated above, this KEP focuses in e2e testing over integration.

##### e2e tests

This KEP will leverage the existing e2e test suite for the majority of testing.
Given that this KEP proposes no behavioral changes to the functioning the of
the cloud controllers, the existing tests will provide a valuable signal to
ensure that nothing has been broken during the migration from in-tree to
external CCM.

The following pull request is the first in a series (documented in the request)
which enable external CCMs by default for all GCE/GCP testing from the Kubernetes
repository:

- https://github.com/kubernetes/kubernetes/pull/117503

The following test grids show the e2e tests running with external CCMs on GCP
and AWS cloud providers respectively:

- https://testgrid.k8s.io/provider-gcp-periodics#E2E%20Full%20-%20Cloud%20Provider%20GCP%20-%20with%20latest%20k8s.io/kubernetes
- https://testgrid.k8s.io/provider-aws-periodics#ci-cloud-provider-aws-e2e-kubetest2

### Graduation Criteria

#### Alpha

- Feature gates added for `DisableCloudProviders` and `DisableKubeletCloudCredentialProviders`
- Working out-of-tree ccms for existing in-tree providers
- Unit and e2e tests to exercise the new out-of-tree CCMs

#### Beta

- Disable in-tree cloud providers, feature gates enabled by default.
- Most, if not all, testing in Kubernetes is using external CCMs. Exceptions
  are made for cases where no CCM is required.
- Multiple providers are being e2e tested against the latest Kubernetes
  libraries on master. This is to prevent regressions when pinning to known
  versions of external CCMs.
- Promotion of the migration progress and process through documentation,
  announcements on mailing lists, and delivered conference presentations.

#### GA

- Two releases have passed at beta status without incident or regression.

#### Deprecation

- Removal of all in-tree code no earlier than one release after GA release.
- Removal of option to set specific cloud providers through --cloud-provider,
  these options should be removed at in-tree providers are removed.

### Upgrade / Downgrade Strategy

The strategy for upgrading a Kubernetes cluster to use external CCMs can be
briefly described as updating the commnand line flags for the `kubelet`,
`kube-apiserver`, and `kube-controller-manager`, and then deploying the
external CCM pods into the cluster. More detailed information about the
command line flags and CCM operation can be found in the Kubernetes
documentation for [Cloud Controller Manager Administration][ccmadmin] and
[Migrate Replicated Control Plane To Use Cloud Controller Manager][ccmmigrate].

Similarly the strategy for downgrading a Kubernetes cluster to not use external
CCMs is a reverse of the previous description: change the command line flags
back to their original values, and then remove the external CCM deployments
from the cluster.

The strategy for upgrading and downgrading the external CCMs is not in scope
for this KEP as each cloud provider community will be responsible for the
maintenance of their CCM. Similarly, documenting the strategy for operating
specific cloud provider CCM implementations will be the responsibility of
those provider communities.

#### Cloud Provider Specific Guidance

The user is now responsible for running the external CCMs, this should be done
during cluster installation. Each provider may have different guidance for
operating their CCM and as such provider-specific documentation should be
consulted.

Some former in-tree cloud providers have created documentation to guide users
in operating the external CCMs for those platforms. Please see the following:

* AWS - [Getting Started with the External Cloud Controller Manager][awsccm]
* Azure - [Deploy Cloud Controller Manager][azureccm]
* OpenStack - [Get started with external openstack-cloud-controller-manager in Kubernetes][openstackccm]
* vSphere - [CPI - Cloud Provider Interface][vsphereccm]

An example of deploying an external CCM on GCE can be seen in the
`start-cloud-controller-manager` script function from the Kubernetes
repository:

* GCE - [kubernetes/cluster/gce/gci/configure-helper.sh][gce]

#### General Guidance

There are a few general notes to observe when migrating to external CCMs.
These notes are collected from
[Kubernetes issue 120411 - Collect tips for external cloud provider migration][issue120411].

* `kube-apiserver`
  * Ensure that the `DisableCloudProviders` feature gate is true, whether by default or explicitly.
* `kubelet`
  * Ensure that the `DisableKubeletCloudCredentialProviders` feature gate is true, whether by default or explicitly.
  * Set the `--image-credential-provider-config` and `--image-credential-provider-bin-dir` flags as appropriate for the cloud provider, see the [kubelet Synopsis documentation][kubelet] for more information.
* `kube-controller-manager`
  * Disable the `node-ipam-controller` using the `--controllers` command line flag, see the [kube-controller-manager Synopsis documentation][kcm] for more information.

### Version Skew Strategy

The external CCMs follow the [same version skew policy][skew] as the kube-controller-manager.
Providers, and communities, that publish their own CCM are responsible for
updating those CCMs to follow the version and skew policy.

Copied from the [Kubernetes Version Skew Policy documentation][skew]:

> kube-controller-manager, kube-scheduler, and cloud-controller-manager must not be newer than the kube-apiserver instances they communicate with. They are expected to match the kube-apiserver minor version, but may be up to one minor version older (to allow live upgrades).

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?
- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DisableCloudProviders
  - Components depending on the feature gate: kubelet, kube-apiserver, kube-controller-manager
- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DisableKubeletCloudCredentialProvider
  - Components depending on the feature gate: kubelet

###### Does enabling the feature change any default behavior?
Yes, enabling this feature will disable all capabilities enabled when `--cloud-provider` is set in core components.
Users need to ensure they have migrated to out-of-tree components prior to enabling this feature gate.
If appropriate extensions (CCM, credential provider, apiserver-network-proxy, etc) are in use, cloud provider capabilities
should remain the same at the very least.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?
Yes, the feature can be disabled once it is enabled. If disabled, users must ensure
that the CCM is no longer running in the cluster. Credential provider plugins and the
apiserver network proxy do not have to be stopped on rollback.

###### What happens if we reenable the feature if it was previously rolled back?
All capabilities from in-tree cloud providers will be re-disabled.

###### Are there any tests for feature enablement/disablement?
Yes, there are a large number of feature tests including:
* unit tests under [https://github.com/kubernetes/cloud-provider/blob/release-1.28](https://github.com/kubernetes/cloud-provider/blob/release-1.28)
* e2e test suites running with in-tree provider disabled:
  * [https://testgrid.k8s.io/provider-gcp-periodics#disable-cloud-provider](https://testgrid.k8s.io/provider-gcp-periodics#disable-cloud-provider)
  * [https://testgrid.k8s.io/provider-gcp-periodics#E2E%20Full%20-%20Cloud%20Provider%20GCP%20-%20with%20latest%20k8s.io/kubernetes](https://testgrid.k8s.io/provider-gcp-periodics#E2E%20Full%20-%20Cloud%20Provider%20GCP%20-%20with%20latest%20k8s.io/kubernetes)

For enablement/disablement tests, we are relying on manual tests that were
done downstream by at least two large cloud providers: GCP and AWS.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?
The primary method of rollout failure is due to configuration issues. When
migrating to use external CCMs there are several changes which must be made
to ensure proper functioning. When encountering a failure, users should review
the migration documentation to ensure that all command line flags, RBAC
requirements, taint tolerations, and manifests are following the best
practices.

It is possible that a failed rollout could affect running workloads. Normal
functioning of external CCMs involves activities around labeling new node
objects, removing node objects, and configuring loadbalancer type services.
If a rollout of external CCMs failed, users might see failures of workloads
to schedule, workloads might be evicted, and service-based traffic could be
interrupted.

For more information about rollouts, upgrades, and planning, please see the
Kubernetes documentation:
* [Cloud Controller Manager Administration][ccmadmin]
* [Migrate Replicated Control Plane To Use Cloud Controller Manager][ccmmigrate]

###### What specific metrics should inform a rollback?
There are a few observable behaviors that inform a rollback. If users start to
see nodes failing to register, load balancer calls failing or not happening,
deleted nodes not being removed, or components failing to start due to missing
taint tolerations, these could all be possible signs that a CCM rollout has
failed.

In addition there are several metrics exposed by the cloud-provider framework
that is maintained by SIG Cloud Provider. If the external CCM is using the
cloud-provider framework, the following metrics could inform about failing
behavior, users should be observant of these metric values increasing over
a short period of time, or after upgrade:

* `cloud_provider_taint_removal_delay_seconds`
* `initial_node_sync_delay_seconds`
* `nodesync_error_total`
* `nodesync_latency_seconds`
* `update_loadbalancer_host_latency_seconds`

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?
Yes, the upgrade and downgrade paths have been tested manually on AWS and GCE
by those provider owners. We do not have artifacts of those tests.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?
* The provider options (e.g. `aws`, `azure`, etc), with the exception of `external`,
  for the `--cloud-provider` flag of `kubelet`, `kube-apiserver`, and
  `kube-controller-manager` are deprecated with the beta release.
* The in-tree cloud controller loops for `kubelet`, `kube-apiserver`, and
  `kube-controller-manager` are deprecated with the beta release.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?
* On cloud providers that utilize an image credential provider, this feature
  is in use when container images are being pulled using those credentials.
* Workloads that explicitly use load balancer type services will depend on external CCMs
  on providers that support the load balancer service controller.
* Workloads that depend on well-known zone and region labels on nodes are
  also considered to be using this feature transitively.

###### How can someone using this feature know that it is working for their instance?
- [x] Other (treat as last resort)
  - Details: The `Uninitialized` taint should be removed from Node objects.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?
* 99% of `cloud_provider_taint_removal_delay_seconds` metric counts are less than 60 seconds.
* 99% of `initial_node_sync_delay_seconds` metric counts are less than 60 seconds.
* On providers that utilize load balancer type services:
  * 99% of `update_loadbalancer_host_latency_seconds` metric counts are less than 60 seconds.
  * 99% of `nodesync_latency_seconds` metric counts are less than 60 seconds.
  * Rate of increase for `nodesync_error_total` is less than 1 per 15 minutes.
* The existing SLOs for functionalities the `kube-controller-manager` should be used,
  this change is supposed to be no-op from the end-user perspective.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?
- [x] Metrics
  - Metric name: `cloud_provider_taint_removal_delay_seconds`
    - Components exposing the metric: CCM
  - Metric name: `initial_node_sync_delay_seconds`
    - Components exposing the metric: CCM
  - Metric name: `update_loadbalancer_host_latency_seconds`
    - Components exposing the metric: CCM
  - Metric name: `nodesync_error_total`
    - Components exposing the metric: CCM
  - Metric name: `nodesync_latency_seconds`
    - Components exposing the metric: CCM

###### Are there any missing metrics that would be useful to have to improve observability of this feature?
A metric to indicate failed connections with the cloud provider API
endpoints could be helpful in determining when a CCM is not able to
communicate with the underlying infrastructure.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?
CCM must be able to connect to cloud provider services and endpoints, this
will depend on the specifics of each provider

### Scalability

###### Will enabling / using this feature result in any new API calls?
Possibly. This change focuses on a migration of code from core Kubernetes
components into external user-managed components. During the alpha state of
this feature, the code from in-tree implementations was migrated into external
out-of-tree repositories. At that point in time, the feature code was the
same between in-tree and out-of-tree implementations. As the out-of-tree CCMs
have now become independent projects, it is out of scope for this KEP to
provide continuous updating of those project statuses with respect to API
calls.

###### Will enabling / using this feature result in introducing new API types?
No.

###### Will enabling / using this feature result in any new calls to the cloud provider?
Possibly. As this change focuses on a migration of previously in-tree code
to externally developed CCMs, it is out of scope to provide predictive
guidance about how those project teams will add or remove calls to the cloud
provider. In the alpha state of this feature, the in-tree code was migrated
into external out-of-tree repositories. At that point in time, no new
cloud provider calls were added.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?
No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?
No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?
No. In fact, it should reduce resource usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?
No, it should not. But, as with any change that requires running additional
pods it is possible that the new CCMs will place load on cluster
resources. The load from the new CCMs should not be significantly greater than
the previous in-tree implementations.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?
* CCMs will not be able to update node and service objects.
* In general, the expectations for the `kube-controller-manager` with respect
  to an unavailable API server and/or etcd will apply to this feature as
  well.

###### What are other known failure modes?
- CCM not communicating with provider infrastructure.
  - Detection: Nodes not having the `Uninitialized` taint removed
    automatically. The `nodesync_error_total` metric shows an increasing
    rate.
  - Mitigation: Check configuration of cloud credentials and CCM to ensure
    that the proper values, RBAC, and quotas are granted with the provider.
  - Diagnostics: Logs should be checked for failed calls to the infrastructure
    provider (these will looks different on each provider). The default log
    level should be sufficient to show these failures.
  - Testing: Check cloud credentials manually with provider. Additionally,
    watch nodes for removal of the `Uninitialized` taint and the application
    of `Ready` state.

###### What steps should be taken if SLOs are not being met to determine the problem?
* Checking the CCM logs for failures and errors.
* Review the RBAC requirements for the CCMs and related Kubernetes components.
* Confirm cloud provider configurations, settings, and quotas.
* Review requests and limits for CCMs to ensure proper resourcing.
* If bug or other failure, rollback to in-tree providers and open an issue
  with the appropriate cloud provider.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2019-01-28 - `Summary`, `Motivation`, and `Proposal` sections merged.
- 2021-02-09 - Add phase 4 to `Proposal` section with addition of `DisableCloudProviders` and `DisableKubeletCloudCredentialProvider` feature gates.
- 2021-02-09 - Add production readiness review for alpha stage.
- 2021-08-04 - First Kubernetes release (v1.22) with `DisableCloudProviders` and `DisableKubeletCloudCredentialProviders` feature gates.
- 2023-09-02 - Enable external CCMs by default for k/k CI.
- 2023-05-07 - All the in-tree cloud providers have been removed.

## Drawbacks

A drawback of this proposal is the complexity involved with doing this type
of change in the Kubernetes community. It will necessarily require an
increased level of turbulence for developers and users as they learn about
the changes to the `kubelet`, `kube-apiserver`, and `kube-controller-manager`
processes, and the changes necessary to operate the external CCMs. Users will
need to adapt their practices and automation to include the changes to
command line flags and manifests to enable external cloud providers.

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


[ccmadmin]: https://kubernetes.io/docs/tasks/administer-cluster/running-cloud-controller/#cloud-controller-manager
[ccmmigrate]: https://kubernetes.io/docs/tasks/administer-cluster/controller-manager-leader-migration/
[awsccm]: https://github.com/kubernetes/cloud-provider-aws/blob/master/docs/getting_started.md
[azureccm]: https://cloud-provider-azure.sigs.k8s.io/install/azure-ccm/
[vsphereccm]: https://cloud-provider-vsphere.sigs.k8s.io/cloud_provider_interface
[openstackccm]: https://github.com/kubernetes/cloud-provider-openstack/blob/master/docs/openstack-cloud-controller-manager/using-openstack-cloud-controller-manager.md
[issue120411]: https://github.com/kubernetes/kubernetes/issues/120411
[ccmframework]: https://github.com/kubernetes/cloud-provider
[kubelet]: https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/
[kcm]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/
[gce]: https://github.com/kubernetes/kubernetes/blob/release-1.28/cluster/gce/gci/configure-helper.sh#L2259-L2353
[skew]: https://kubernetes.io/releases/version-skew-policy/#kube-controller-manager-kube-scheduler-and-cloud-controller-manager
