---
title: Install Addons via kubeadm
authors:
  - "@stealthybox"
  - "@fabriziopandini"
owning-sig: sig-cluster-lifecycle
participating-sigs:
reviewers:
  - TBD
  - "@dholbach"
  - "@fabriziopandini"
  - "@justinsb"
  - "@neolit123"
  - "@rosti"
approvers:
  - TBD
  - "@timothysc"
editor: TBD
creation-date: 2019-10-13
last-updated: 2019-10-13
status: provisional #|implementable|implemented|deferred|rejected|withdrawn|replaced
see-also:
  - "/keps/sig-cluster-lifecycle/addons/0035-20190128-addons-via-operators.md"
  - "/keps/sig-cluster-lifecycle/wgs/0032-create-a-k8s-io-component-repo.md"
replaces:
superseded-by:
---

# Install Addons via kubeadm

## Table of Contents
<!-- [Tool for generating](https://github.com/ekalinin/github-markdown-toc) -->

- [Install Addons via kubeadm](#install-addons-via-kubeadm)
  - [Table of Contents](#table-of-contents)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
      - [Goals](#goals)
      - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
      - [User Stories](#user-stories)
        - [Links](#links)
      - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
      - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
      - [Test Plan](#test-plan)
      - [Graduation Criteria](#graduation-criteria)
        - [Examples](#examples)
            - [Alpha](#alpha)
            - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
            - [Beta -&gt; GA Graduation](#beta---ga-graduation)
            - [Removing a deprecated flag](#removing-a-deprecated-flag)
      - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
      - [Version Skew Strategy](#version-skew-strategy)
  - [Implementation History](#implementation-history)
  - [Drawbacks](#drawbacks)
  - [Alternatives](#alternatives)
  - [Infrastructure Needed](#infrastructure-needed)


## Release Signoff Checklist

<!-- **ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released. -->

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!-- **Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website -->

## Summary

<!-- The `Summary` section is incredibly important for producing high quality user-focused documentation such as release notes or a development roadmap.
It should be possible to collect this information before implementation begins in order to avoid requiring implementors to split their attention between writing release notes and implementing the feature itself.
KEP editors, SIG Docs, and SIG PM should help to ensure that the tone and content of the `Summary` section is useful for a wide audience.

A good summary is probably at least a paragraph in length. -->

The addons subproject hosted by SIG Cluster Lifecycle has been working on implementation details regarding addon installation, user-experience, maintenance, and operation.
The [addon-installer library POC](https://github.com/kubernetes-sigs/addon-operators/pull/25) functions as a small, vendorable implementation for applying addons to the cluster using kubectl and Kustomize. The `AddonInstallerConfiguration` API kind is intended to be a user-facing ComponentConfig.
This KEP proposes an additional phase to kubeadm that invokes the addon-installer library.

## Motivation

<!-- This section is for explicitly listing the motivation, goals and non-goals of this KEP.
Describe why the change is important and the benefits to users.
The motivation section can optionally provide links to [experience reports][] to demonstrate the interest in a KEP within the wider Kubernetes community. -->

Kubeadm currently installs two "core" addons into the cluster once the control-plane is running:  kube-proxy and CoreDNS.
Optionally, it also supports installing kube-dns in place of CoreDNS.
Installation of these addons occurs via application of manifests for which the strings are built into and versioned with the kubeadm binary.
It is not currently possible to disable installation of these addons or modify them with a formal kubeadm init.

Upgrades are accomplished with modifications to kubeadm's upgrade logic, often by introspecting which GVK's exist in the current cluster. 

Kubeadm installs these "core" addons leaving an almost functioning cluster available for the user, but a CNI implementation is not installed. That is left to the cluster operator.

Maintaining these manifests and this logic with kubeadm has to this point has allowed users to receive a useful, default setup.
As the ecosystem grows, this is proving to now be too inflexible for users and vendors.

Use of ClusterAPI means less operators will be manually mutating clusters before they are expected to function.
Kubeadm does not need to maintain CNI installation, but users should be able to use kubeadm to produce a functioning cluster, complete with with operational CNI. 

### Goals

<!-- List the specific goals of the KEP.
How will we know that this has succeeded? -->

Integrating the addon-installer library and API with kubeadm will allow users to opt out of installing kube-proxy or CoreDNS/kube-dns.
Some reasons for this may be:
- The user does not desire pod networking and/or service-discovery in their cluster.
- CNI implementations such as kube-router and Cilium can actually replace kube-proxy.

The "core" addons can be referenced and installed from a central, extensible, shared implementation.
In the case of CoreDNS, this may even make sense to be managed by the upstream project instead of maintaining copies of manifests in kubernetes/kubernetes.

This integration will also allow users to extend these "core" addons using their own kustomize driven patches.
Patches are expected to be supplied via the API, filesystem, or git with additional potential for OCI images.

This will also enable users to specify additional addons to install such as CNI implementations, drivers, webhook-servers, or other APIs and operators.

Simple upgrades to newer manifests for a given addon should be supported.
Removal should also be supported.

The mechanisms and APIs for doing this are independently factored from kubeadm meaning other installers such as kops and eksctl as well as cloud and vendor solutions may implement their own defaulting logic and user-interfaces.

### Non-Goals

Addons are currently expected to be applied in order with no success condition beyond API validity.
No dependency system or DAG is currently proposed or in-scope.

Discussions of how addons may validate ComponentConfigs and other files and how the AddonInstaller API can enable that are welcome. There are existing efforts in this area already (currently championed by @rosti) that may need to be cross-referenced.

Usable operators for addons such as the "CoreDNS Operator" are for consideration regarding the motivation, UX, and vision of this KEP's features. However, implementation, packaging, and delivery of these operators is out of scope for the technical implementation of the AddonInstaller in kubeadm.

Providing an "uninstall" signal, hook, or mechanism for addons is currently out of scope but is open for discussion.

## Proposal

An alpha feature gate "`AddonInstaller`" will be added to kubeadm; it will default to false may be user-enabled using the existing flag and `ClusterConfiguration`.
When this feature gate is enabled, the current `kube-proxy` and `coredns` phases under `init.addon` will noop.
A new phase called `installer` will activate creating a sensible logger and runtime for `sigs.k8s.io/addon-operators/installer` library.
This phase is expected to operate in an idempotent manner, run properly during `kubeadm init` and `kubeadm upgrade`, and produce helpful output in a dry-run.

An `AddonInstallerConfiguration` will be defaulted but may be overridden by the user.
It is interpreted to be the authoritative declaration of addons in the cluster.

The phase can roughly operate in this manner:
- Parse the kubeadm global `--dry-run` flag.
- Determine if a KUBECONFIG is available for the cluster.
- Load an `AddonInstallerConfiguration` from the multi-YAML-document kubeadm `--config`.
  If missing, default to one that installs the version appropriate manifests for kube-proxy and the user-indicated DNS solution.
- If possible, pull the previous `AddonInstallerConfiguration` from the cluster ConfigMap.
- Create an installer library runtime with: dryRun, kubeConfig, oldInstallCfg, and newInstallCfg.
- Prune addons from the cluster if names from the previous configuration are missing in the new configuration.
  The library should remove addons serially in reverse order.
- Apply all addons in the new configuration.
  The library should respect the declared, serial order.
- Apply the new `AddonInstallerConfiguration` to the cluster ConfigMap.

Removing resources from the kubernetes API currently does not propagate any "delete reason" to processes in Pods.
This can make it difficult (even for smart addon-operators with separately installed CustomResources) to determine when to clean up with definitive convergence. One use-case is removal of iptables rules. Another is removal of an operator + cleanup of operator-managed external resources in a single declaration.
Uninstall Support Proposal A [not currently in scope]:
- An uninstall kustomize dir can be optionally provided with an addon (along with `name` + `ref`) when applied.
- When the addon is removed, either explicitly or as a consequence of prune via declaration, these manifests could be applied.
- Any uninstall Jobs can be polled for completion.
- If successful:
  - All addon manifests are removed.
  - All uninstall manifests are removed.

### User Stories

Users may enable this feature in `kubeadm init` and `kubeadm upgrade` using the `AddonInstaller` feature gate:
```shell
kubeadm init --feature-gates AddonInstaller=true
kubeadm init --config
```
```yaml
apiVersion: kubeadm.k8s.io/v1beta2
kind: InitConfiguration
---
apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
featureGates:
  AddonInstaller: true
```

Users may invoke the phase similarly:
```shell
kubeadm init phase addon installer --feature-gates AddonInstaller=true
kubeadm init phase addon installer --config
```
The feature gate is still expected to be enabled/true when invoking the phase directly.

Users may print the default `AddonInstallerConfiguration` using `kubeadm config`:
```shell
kubeadm config print init-defaults --feature-gates AddonInstaller=true
# this requires adding a feature-gates flag to this command
```
TODO: Discuss this alternative:
```shell
kubeadm config print addon-defaults
```
Similar U/X can be expected for `kubeadm config migrate` and `kubeadm config view`

The user may extend the default kubeadm config using the `AddonInstallerConfiguration` kind.
The API details are still being finalized. Usage may look like this:
```yaml
---
apiVersion: addons.config.x-k8s.io/v1alpha1
kind: AddonInstallerConfiguration
addons:
- name: kube-proxy
  kustomizeRef: github.com/kubernetes/kubernetes//cluster/addons/kustomize/kube-proxy/?ref=v1.17.0
- name: coredns-operator
  kustomizeRef: github.com/kubernetes-sigs/addon-operators//kustomize/coredns-operator/?ref=v0.1.0
- name: my-addon1
  manifestRef: ../my-local-dir/addon1/
- name: weavenet
  ref: oci+kustomize://weaveworks/kustomize-weavenet:v2.5.2
    # kustomize OCI backend currently unimplemented
    # POC packaging:  https://github.com/ecordell/kpg
```

It may be in scope for kubeadm to internally patch the `AddonInstallerConfiguration` with the contents of user-supplied ComponentConfigs such as `KubeProxyConfiguration`.
Various patchList fields may need to be added to the `Addon` type of the library's API to support this specific U/X as well as more lightweight user-extension of addons.

The AddonInstaller feature composes with kubeadm's global `--dry-run` flag:
- on `init`, the intended addons (and perhaps manifests) can simply be printed.
- on `upgrade diff` and `upgrade apply`, the expected difference relative to the cluster can be output.
  `kubectl` already does this currently.

The difference in behavior between a dryRun init and upgrade is caused by the presence of a working KUBECONFIG and control-plane.

Air-gap support may be accomplished through the mirroring of git repositories or images containing the kustomize dir.
Mirroring git repositories, cloning git repositories to a local filesystem, and hosting OCI images would seem to all be a reasonable capabilities for a user in an air-gapped environment. The API allows a user to reference addon sources for all of these use-cases.

#### Links

@fabriziopandini began a design doc for kubeadm integration:
- https://docs.google.com/document/d/1FZA873LBXf-aG4UULOk96MhT8xm_nPHq5Vu0z61S8pk

More generic User Stories have been collected by @dholbach:
- doc: https://docs.google.com/document/d/17NH4xcFdeh4NVcIBjMQNK2P27j9xrsm4sCFTXdZqopg
- sheet: https://docs.google.com/spreadsheets/d/1Np0aOQYyiRqRQYOUoMcUwP_2Tt9b39HUwJXQIgW9rYA

### Implementation Details/Notes/Constraints

We intend for the installer library and API to live in `sigs.k8s.io/addon-operators/installer`.
Core kustomize addons and images may be hosted in a number of git repositories or image registries.

Internally, the addon-installer library execs `kubectl -k`.
This introduces a binary and PATH dependency for `kubeadm` when using this feature.
Furthermore, certain sensible usage of this feature may depend on `git` by proxy of kustomize.
While `kubectl` may commonly be available in the cluster-installation environment, `git` may not be available.
ex: The current kinder images do not have `git`.

It may be more appropriate to exec `kustomize` directly or vendor it in for manifest generation so that dryRun and validation of the kustomize addons will work in the absence of a working kube-apiserver. (This is important before `kubeadm init`)

The kubeconfig may be passed to the library as a file-path.
This works for kubeadm and means less work when executing `kubectl` for the apply logic.
Should we also support a formal internal type for this?

A more mature addon-installer library may:
- Vendor kustomize for manifest generation.
- Use an apiClient type for applying the manifests.
  ( This may be compatible with some shareable client type supporting dryRun? )
- Have an go-native git clone implementation

Kustomize currently supports constructing manifests from local directories and git.
OCI image support in kustomize is currently unimplemented.
A POC of some of this mechanism is implemented here: https://github.com/ecordell/kpg
Implementation of this feature could be delegated to another KEP.

### Risks and Mitigations

<!-- What are the risks of this proposal and how do we mitigate.
Think broadly.
For example, consider both security and how this will impact the larger kubernetes ecosystem.

How will security be reviewed and by whom?
How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject. -->

The core of this proposal is that it expands on the current packaging of addons within the kubeadm binary.
This binary is currently signed, released, and distributed by kubernetes release machinery.

Since this proposal necessitates new artifacts and distribution channels for default/"core" components, SIG Release will ultimately be interested in providing methods of verification and integrity of those artifacts for users.

Git is a strong tool for versioning manifests and patches but may be questionable for asserting source provenance without instrumenting non-default behavior.
This attack surface may become more evident when mirroring in an airgap environment.
When fetching/cloning from GitHub the transport will either be verified by HTTPS or an SSH host key.

Distributing kustomize dirs via signed OCI images may be a solution for this.

Proposing additional support for raw manifest lists (with or without kustomization) may also be effective for certain use-cases when sourced over HTTPS from cloud-storage such as GitHub-raw, S3, GCS, or an internal solution like NGINX or Minio.

## Design Details

### Test Plan

<!-- **Note:** *Section not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.
Anything that would count as tricky in the implementation and anything particularly challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage expectations).
Please adhere to the [Kubernetes testing guidelines][testing-guidelines] when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md -->

### Graduation Criteria

<!-- **Note:** *Section not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. Initial KEP should keep
this high-level with a focus on what signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/ -->

#### Examples

<!-- These are generalized examples to consider, in addition to the aforementioned [maturity levels][maturity-levels]. -->

##### Alpha

- Support for API-driven ComponentConfig declaration of addons is working in kubeadm for init and upgrade.
- ClusterAPI implementations may begin experimenting with producing more functional clusters as a result.
- Users and vendors can provide feedback.

##### Alpha -> Beta Graduation

<!-- - Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP -->

TODO

##### Beta -> GA Graduation

<!-- - N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and GA/stable, since there's no opportunity for user feedback, or even bug reports, in back-to-back releases. -->

The current user-experience for installing the three current "core" addons with kubeadm, either by default or with user specification becomes equivalent or improved in safety and U/X when kubeadm uses the `AddonInstallerConfiguration`.

##### Removing a deprecated flag

<!-- - Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/conformance-tests.md -->

The Alpha `AddonInstaller=true|false` key-value pair will eventually be removed from kubeadm's featureGates flag and API.

### Upgrade / Downgrade Strategy

<!-- If applicable, how will the component be upgraded and downgraded? Make sure this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to make use of the enhancement? -->

### Version Skew Strategy

<!-- If applicable, how will the component handle version skew with other components? What are the guarantees? Make sure
this is in the test plan.

Consider the following in developing a version skew strategy for this enhancement:
- Does this enhancement involve coordinating behavior in the control plane and in the kubelet? How does an n-2 kubelet without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI, CRI or CNI may require updating that component before the kubelet. -->

`kubeadm` default addon refs will be generated according to the target kubernetes version.

Warnings or overridable Preflight-checks may be invoked when "core" addon refs are detected in the `AddonInstallerConfiguration` that do not match or are lower than the target Kubernetes control-plane version.

## Implementation History

<!-- Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded -->

- Overarching Addon-Management design:
  https://github.com/kubernetes-sigs/addon-operators/issues/10
- addon-installer lib POC:
  https://github.com/kubernetes-sigs/addon-operators/pull/25
- kubeadm integration POC:
  https://github.com/kubernetes/kubernetes/compare/master...stealthybox:kubeadm-addon-installer

## Drawbacks

<!-- Why should this KEP _not_ be implemented. -->

## Alternatives

<!-- Similar to the `Drawbacks` section the `Alternatives` section is used to highlight and record other possible approaches to delivering the value proposed by a KEP. -->

## Infrastructure Needed

<!-- Use this section if you need things from the project/SIG.
Examples include a new subproject, repos requested, github details.
Listing these here allows a SIG to get the process for these resources started right away. -->
