---
title: Building Kubernetes Without In-Tree Cloud Providers
authors:
  - "@BenTheElder"
owning-sig: sig-cloud-provider
participating-groups:
  - sig-release
  - sig-testing
reviewers:
  - "@spiffxp"
  - "@cheftako"
  - "@andrewsykim"
  - "@stephenaugustus"
approvers:
  - "@cheftako"
  - "@andrewsykim"
  - "@spiffxp"
  - "@stephenaugustus"
editor: TBD
creation-date: 2019-07-29
last-updated: 2020-05-08
status: implemented
see-also:
  - "/keps/sig-cloud-provider/20190125-removing-in-tree-providers.md"
  - "/keps/sig-cloud-provider/20180530-cloud-controller-manager.md"
---

# Building Kubernetes Without In-Tree Cloud Providers

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories [optional]](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This proposal outlines a plan to enable building Kubernetes without the in-tree
cloud providers in preparation for [removing them entirely](keps/sig-cloud-provider/20190125-removing-in-tree-providers.md).

## Motivation

The in tree cloud-provider implementations are being [removed](keps/sig-cloud-provider/20190125-removing-in-tree-providers.md) in the future, this involves a large amount
of code that is used in many places in tree. In order to prepare for this eventuality
it would be helpful to see what that removal entails exactly and verify that Kubernetes
will continue to function correctly. Doing so is a bit tricky without ensuring
that the in-tree provider code is not being used via some unexpected side-channel
(such as `init()` methods). Building binaries without the in-tree cloud provider
packages would allow us to verify this and additionally provide experimentally
smaller / cheaper binaries for parties interested in out of tree provider or 
no provider based clusters.

### Goals

- Enable building Kubernetes without in-tree cloud providers and without forking
  - Enable testing out of tree providers with a simulation of the future removal of the in-tree code.
  - Enable experimentation with cloud-provider-less clusters

### Non-Goals

- Building the out of tree providers
- Changing the official Kubernetes release builds
- Building the e2e tests 
  - Decoupling cloud providers is a larger problem there and not necessary to test out-of-tree providers or build smaller binaries
- Mechanisms for migrating to out of tree providers
  - CSI Migration for in-tree Volumes is already underway in SIG Storage
  - External Credential Providers is being written / solved in another KEP ([#541](https://github.com/kubernetes/enhancements/issues/541))
  - CCMs and overall scope for moving out of tree is in [removing-in-tree-providers](keps/sig-cloud-provider/20190125-removing-in-tree-providers.md)

## Proposal

We will add a [build constraints](https://golang.org/pkg/go/build/#hdr-Build_Constraints)
to the cloud provider code for a pseudo "build tag" specifying not to include
any in-tree provider code. This will allow compiling the binaries as normal today
and simulating the removal of this code by specifying the tag at build time and
triggering the build constraints on the files in these packages.

Some small adjustments may be necessary to the code base to ensure that the
other packages can build without depending on these packages.

A prototype is available in [kubernetes/kubernetes#80353](https://github.com/kubernetes/kubernetes/pull/80353).

To ensure that this continues to function we will add CI building in this mode,
and CI running end to end tests against it (see the test plan).

### User Stories [optional]

#### Story 1

As an out of tree cloud provider implementer, I want to develop and test against
Kubernetes without the in tree providers.

Kubernetes out of tree cloud provider developers will be able to build Kubernetes
in this mode and build & test their cloud-controller-manager implementations and
associated tooling against this build in preparation for the actual hard removal
of the in-tree providers.

#### Story 2

As a developer working to replace an in-tree provider with an out-of-tree provider,
I am attempting to validate that I work with KAS/KCM/Kubelet which do not have 
(my) in-tree cloud-provider compiled in and have successfully migrated all the 
functionality I need to CCM/CSI/... Using this build ensures the relevant 
functionality is not in KAS/KCM/Kubelet. It also allows me to work with 
smaller binaries.

#### Story 3

As a [kind](https://github.com/kubernetes-sigs/kind) developer / user I want to 
use Kubernetes binaries without cloud providers for local clusters.

Developers and users will be able to build local clusters leveraging this mode
to not pay for cloud providers they are unable to use.

### Implementation Details/Notes/Constraints [optional]

This is implemented using a synthetic `nolegacyproviders` tag in go build
constraints on the relevant sources. If `GOFLAGS=-tags=nolegacyproviders` is set
then the legacy cloud provider pacakges will be excluded from the build.

In order to make this work the following additional changes are made:

- Packages that we fully exclude (the legacy provider packages) _must_ contain
a `doc.go` file or any other file that does NOT contain any code or build 
constraints. Go will not allow "building" a package without any files passing
the constraints, however it will happily build a package with no actual methods
/ variables / ...

- A few locations in the code do not properly use the cloud provider interface
(instead, importing the cloud provider packages directly),
some of these must be updated with both a "with provider" version and a
"without provider" version broken out of the existing code. In particular this 
includes the in-tree volumes until CSI migration is standard, and the GCE IPAM
logic in kube-controller-manager.
  - Note that the nodeIpamController GCE IPAM logic is slated for removal (see [the cloud controller manager KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/20180530-cloud-controller-manager.md))

In particular this adds tags / constraints to:
- `staging/src/k8s.io/legacy-cloud-providers/*` (constraints on all the providers)
- `pkg/cloudprovider` (constraints on importing the providers)
- `pkg/volumes/*`, `cmd/kubelet` (versions with and without the imported providers for in-tree volumes)
- `pkg/controller/nodeipam`, `cmd/kube-apiserver`, `cmd/kube-controller-manager` (with and without GCE IPAM)

`test/*` is punted to a future follow up, and credential providers are punted
to [the external credential provider KEP](https://github.com/kubernetes/enhancements/pull/1137).

### Risks and Mitigations

This is only developer facing, however we will need to ensure that these tags
stay up to date if we want this build mode to continue to work (the normal
build mode should work by default without any additional maintenance).

To ensure this continues to work we can mitigate by:

- verify in CI that the cloud provider packages have boilerplate including the
build constraints
- build in this mode in CI to ensure that the build succeeds

## Design Details

### Test Plan

We will add CI to ensure that we can build with this mode enabled.

Additionally, we can add CI to ensure that clusters can actually be started in
this mode.

Initially, [kind](https://github.com/kubernetes-sigs/kind) can be used to ensure
that Kubernetes works without the providers, in the future we can extend this
CI to out-of-tree providers combined with this build mode as their CI is spun up.

### Graduation Criteria

##### Alpha -> Beta Graduation

Likely unnecessary, as we will eventually remove the in-tree provider code entirely for [removing-in-tree-providers](keps/sig-cloud-provider/20190125-removing-in-tree-providers.md).
This is also not a user facing change.

##### Beta -> GA Graduation

Likely unnecessary, as we will eventually remove the in-tree provider code entirely for  [removing-in-tree-providers](keps/sig-cloud-provider/20190125-removing-in-tree-providers.md).
This is also not a user facing change.

Final graduation can be considered to be when the cloud provider code is actually
removed from the Kubernetes source tree, at which point this work will be complete.

### Upgrade / Downgrade Strategy

N/A ?

### Version Skew Strategy

N/A ?

## Implementation History

- original prototype [kubernetes/kubernetes#80353](https://github.com/kubernetes/kubernetes/pull/80353)
- original KEP PR [kubernetes/enhancements#1180](https://github.com/kubernetes/enhancements/pull/1180)
- typechecking CI [kubernetes/kubernetes#85457](https://github.com/kubernetes/kubernetes/pull/85457)

## Drawbacks [optional]

This does require maintaining these tags / constraints for the providerless build,
however in the default mode without our pseudo-tag the code will build as today
and require zero additional maintenance to function. As in-tree providers are
relatively stable and expected not to gain new features, this should require
minimal effort and can be automated to a limited extent.

## Alternatives [optional]

We could simply wait for the in-tree providers to be removed entirely, however
this may not provide sufficient tools to adequately prepare.

There is also a risk that cloud providers would each need to duplicate this
work to test cloud-provider free Kubernetes for their out of tree provider.

We could attempt to create a branch/PR with those changes in them. 
However the in-tree providers are not guaranteed to exit at the same time. 
So the branch/PR might have to be kept for a long period of time. 
In addition to being expensive the maintain such a PR/branch, it would obfuscate
the effort. So developers would end up changing CP related code and have little
/ no visibility that their changes were CP related.

## Infrastructure Needed [optional]

None?

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

