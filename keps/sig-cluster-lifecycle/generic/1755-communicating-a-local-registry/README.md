<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up.  KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please ensure to complete all
  fields in that template.  One of the fields asks for a link to the KEP.  You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "title", "authors", "owning-sig",
  "status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary", and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
  Assign it to people in the SIG that are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly.  The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved.  Any KEP
marked as a `provisional` is a working document and subject to change.  You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused.  If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement", for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example.  If there are
new details that belong in the KEP, edit the KEP.  Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable` or significant changes once
it is marked `implementable` must be approved by each of the KEP approvers.
If any of those approvers is no longer appropriate than changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross cutting KEPs).
-->
# KEP-1755: Standard for communicating a local registry

<!--
This is the title of your KEP.  Keep it short, simple, and descriptive.  A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Pushing images to an outdated URL](#pushing-images-to-an-outdated-url)
    - [Confusion due to multiple versions in the ConfigMap](#confusion-due-to-multiple-versions-in-the-configmap)
- [Design Details](#design-details)
  - [The <code>local-registry-hosting</code> ConfigMap](#the-local-registry-hosting-configmap)
  - [LocalRegistryHosting](#localregistryhosting)
    - [Specification for LocalRegistryHosting v1](#specification-for-localregistryhosting-v1)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [How the configuration is stored](#how-the-configuration-is-stored)
  - [How the configuration is updated](#how-the-configuration-is-updated)
  - [Support for future registry topologies](#support-for-future-registry-topologies)
  - [Support for secure registries](#support-for-secure-registries)
  - [Support for other local image loaders](#support-for-other-local-image-loaders)
- [Infrastructure Needed (optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

- [x] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] KEP approvers have approved the KEP status as `implementable`
- [x] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

<!--
This section is incredibly important for producing high quality user-focused
documentation such as release notes or a development roadmap.  It should be
possible to collect this information before implementation begins in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself.  KEP editors, SIG Docs, and SIG PM
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.
-->

Local clusters like Kind, K3d, Minikube, and Microk8s let users iterate on Kubernetes
quickly in a hermetic environment. To avoid network round-trip latency, these
clusters can be configured to pull from a local, insecure registry. 

This KEP proposes a standard for how these clusters should expose their support
for this feature, so that tooling can interoperate with them without redundant
configuration.

## Motivation

<!--
This section is for explicitly listing the motivation, goals and non-goals of
this KEP.  Describe why the change is important and the benefits to users.  The
motivation section can optionally provide links to [experience reports][] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Many local clusters currently support local registries. But they put the onus of
configuration on the user. First, the user has to follow instructions to setup
the registry in the cluster.

- https://kind.sigs.k8s.io/docs/user/local-registry/
- https://microk8s.io/docs/registry-built-in
- https://github.com/rancher/k3d/blob/master/docs/registries.md#using-a-local-registry
- https://minikube.sigs.k8s.io/docs/handbook/registry/#enabling-insecure-registries

When the cluster has been set up, the user has to manually go through each
individual tool that needs to push images to the cluster, and configure it with
the hostname and port of the new registry.

The motivation of this KEP is to remove this configuration redundancy, and
reduce the burden on the user to configure all their image tools.

### Goals

<!--
List the specific goals of the KEP.  What is it trying to achieve?  How will we
know that this has succeeded?
-->

- Agree on a standard way for cluster configuration tools to record how
  developer tools should interact with the local registry.
  
- Agree on a standard way for developer tools to read that information
  when pushing images to the cluster.

### Non-Goals

<!--
What is out of scope for this KEP?  Listing non-goals helps to focus discussion
and make progress.
-->

- Modifying how local registries are currently implemented on these clusters.

- How this might be extended to support multiple registries. This proposal
  assumes clusters have at most one registry, because that's how all existing
  implementations work.

- Any API for configuring a local registry in a cluster. If there was a standard
  CRD that configured a local registry, and all implementations agreed to
  support that CRD, this KEP would become moot. That approach would have
  substantial technical challenges. OpenShift supports [a
  CRD](https://docs.openshift.com/container-platform/4.4/openshift_images/image-configuration.html)
  for this, which they use to configure registries.conf inside the cluster.
  
- A general-purpose mechanism for publicly exposing cluster configuration.

- A general-purpose mechanism for cluster capability detection for developer
  tooling. With the best practices around cluster capability detection moving
  fast, local registries are the only best practice that's mature enough where
  this makes sense.
  
- The creation of Git repositories that can host the API proposed in this KEP.

- Many local clusters expose other methods to load container images into the
  cluster, independent of the local registry. This proposal doesn't address how
  developer tools should detect them.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation.  The "Design Details" section below is for the real
nitty-gritty.
-->

Tools that configure a local registry should also apply a ConfigMap that
communicates "LocalRegistryHosting".

The ConfigMap specifies everything a tool might need to know about how to
interact with the local registry.

Any tool that pushes an image to a registry should be able to read this
ConfigMap, and decide to push to the local registry instead.

### User Stories (optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system.  The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

Alice is setting up a development environment on Kubernetes.

Alice creates a local Kind cluster. She follows the Kind-specific instructions
for setting up a local registry during cluster creation.

Alice will have some tool she interacts with that builds images and pushes them
to a registry the cluster can access. That tool might be an IDE like VSCode,
an infra-as-code toolchain like Pulumi, or a multi-service dev environment like Tilt.

On startup, the tool should connect to the cluster and read a ConfigMap.

If the config specifies a registry location, the tool may automatically adjust
image pushes to push to the specified registry.

If the config specifies a `help` URL, the tool may prompt the user to set up a
registry for faster development.

### Notes/Constraints/Caveats (optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

- Users may see the ConfigMap and draw mistaken conclusions about how to
  interact with it. They think deleting the ConfigMap would delete the local
  registry (which it does not).

- This KEP does not specify any automatic mechanism for keeping the
  LocalRegistryHosting up-to-date with how the cluster is configured.
  For example, the user might delete the registry. The cluster might
  not have a way of knowing that the registry has died.

### Risks and Mitigations

<!--
What are the risks of this proposal and how do we mitigate.  Think broadly.
For example, consider both security and how this will impact the larger
kubernetes ecosystem.

How will security be reviewed and by whom?

How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.
-->

This KEP only defines the specification for a ConfigMap that includes versioned
structures. There are potential, minimal risks around the usage of this
ConfigMap, but mitigation is delegated to the cluster configuration tools and
their documentation.

#### Pushing images to an outdated URL

**Risk**: Tool X reads a local registry host from the ConfigMap and tries to
push images to it, but the URL is out of date.

*Mitigation*: It is the responsibility of the cluster admin / configuration
tooling to keep the ConfigMap up-to-date either by manual adjustment or via a
controller.

#### Confusion due to multiple versions in the ConfigMap

**Risk**: By definition, the ConfigMap can include multiple versions of the
structures defined in this KEP. `localRegistryHosting.v1` and
`localRegistryHosting.v2` can be present at the same time. Readers might get
confused what version they are supposed to use.

*Mitigation*: Cluster configuration tools should document that they are
consuming the specification defined in this KEP and briefly outline the best
practices - i.e. using the latest `vX` when possible.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable.  This may include API specs (though not always
required) or even code snippets.  If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### The `local-registry-hosting` ConfigMap

Tools that configure a local registry should apply a ConfigMap to the cluster.

Documentation that educates people on how to set up a registry manually should
include instructions on what the ConfigMap should be.

The name of the ConfigMap must be `local-registry-hosting`.

The namespace must be `kube-public`.

Under `data`, the ConfigMap should contain structures in the format
`localRegistryHosting.vX`.  `vX` is the major version of the structure. The
contents of each field are YAML.

Example:

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
```

### LocalRegistryHosting

The `localRegistryHosting.vX` field of the ConfigMap describes how tools should
communicate with a local registry. Tools should use this registry to load images
into the cluster.

The contents of a `localRegistryHosting.vX` specification are frozen and will not
change.

When adding, removing or renaming fields, future proposals should increment the
MAJOR version of an API hosted in the ConfigMap - e.g. `localRegistryHosting.vX`
where X is incremented.

Writers of this ConfigMap (i.e., cluster configuration tooling) should write as
many top-level fields as the cluster supports.  The most recent version is the
source of truth.

Readers of this ConfigMap should start with the most recent version they support
and work backwards. Readers are responsible for doing any defaulting of fields
themselves without the assistance of any common API machinery.

#### Specification for LocalRegistryHosting v1

Example ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: local-registry-hosting
  namespace: kube-public
data:
  localRegistryHosting.v1: |
    host: "localhost:5000"
    hostFromContainerRuntime: "registry:5000"
    hostFromClusterNetwork: "kind-registry:5000"
    help: "https://kind.sigs.k8s.io/docs/user/local-registry/"
```

Golang implementation:

```go
// LocalRegistryHostingV1 describes a local registry that developer tools can
// connect to. A local registry allows clients to load images into the local
// cluster by pushing to this registry.
type LocalRegistryHostingV1 struct {
	// Host documents the host (hostname and port) of the registry, as seen from
	// outside the cluster.
	//
	// This is the registry host that tools outside the cluster should push images
	// to.
	Host string `yaml:"host,omitempty"`

	// HostFromClusterNetwork documents the host (hostname and port) of the
	// registry, as seen from networking inside the container pods.
	//
	// This is the registry host that tools running on pods inside the cluster
	// should push images to. If not set, then tools inside the cluster should
	// assume the local registry is not available to them.
	HostFromClusterNetwork string `yaml:"hostFromClusterNetwork,omitempty"`

	// HostFromContainerRuntime documents the host (hostname and port) of the
	// registry, as seen from the cluster's container runtime.
	//
	// When tools apply Kubernetes objects to the cluster, this host should be
	// used for image name fields. If not set, users of this field should use the
	// value of Host instead.
	//
	// Note that it doesn't make sense semantically to define this field, but not
	// define Host or HostFromClusterNetwork. That would imply a way to pull
	// images without a way to push images.
	HostFromContainerRuntime string `yaml:"hostFromContainerRuntime,omitempty"`

	// Help contains a URL pointing to documentation for users on how to set
	// up and configure a local registry.
	//
	// Tools can use this to nudge users to enable the registry. When possible,
	// the writer should use as permanent a URL as possible to prevent drift
	// (e.g., a version control SHA).
	//
	// When image pushes to a registry host specified in one of the other fields
	// fail, the tool should display this help URL to the user. The help URL
	// should contain instructions on how to diagnose broken or misconfigured
	// registries.
	Help string `yaml:"help,omitempty"`
}
```

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.  Anything
that would count as tricky in the implementation and anything particularly
challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations).  Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

Writers of the ConfigMap are responsible for validating
the config against the specifications in this proposal.

It is out of scope for this proposal to create shared API machinery that can be
used for this purpose.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and
GA/stable, since there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

Not directly applicable. 

The proposed API in this KEP will iterate in MAJOR increments - e.g. v1, v2, v3.

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to make use of the enhancement?
-->

Cluster configuration tools are free to decide if they wish to also
upgrade/downgrade the proposed ConfigMap structures as part of their
upgrade/downgrade process. It is out of scope for this proposal to define or
host the API machinery for that.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

N/A

## Implementation History

<!--
Major milestones in the life cycle of a KEP should be tracked in this section.
Major milestones might include
- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

2020-05-08: initial KEP draft
2020-05-20: addressed initial reviewer comments
2020-06-10: KEP marked as implementable

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

This KEP is contingent on the maintainers of local clusters agreeing to it
(specifically: Kind, K3d, Minikube, Microk8s, and others), since this is really
about better documenting what they're already doing.

This KEP is very narrow in scope and lightweight in implementation. It wouldn't
make sense if there was a more ambitious local registry proposal. It also
wouldn't make sense if there was a more ambitious proposal for cluster feature
discovery.

## Alternatives

<!--
What other approaches did you consider and why did you rule them out?  These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### How the configuration is stored

SIG Cluster Lifecycle have discussed many alternative proposals for where this
data would be stored, including:

- Using annotations on a Namespace
- Using a plain ConfigMap
- Using a ConfigMap with an embedded, versioned component config
- Using a Custom Resource

This proposal attempts to strike the right balance between a format that
follows existing conventions, and a format that doesn't require too much
Kubernetes API machinery to implement.

This proposal also doesn't use alpha/beta versioning, to avoid the common
graduation expectations that come with Kubernetes-core hosted features.

### How the configuration is updated

The group had a longer discussion about whether this should be "inert" data, or
whether there should be mechanisms for keeping it up to date. In some cases,
there is a one-to-one mapping between the cluster config and the
LocalRegistryHosting.  For example, on Kind, the containerd config has the
registry host in it.  The group also discussed whether there should be a
solution for reflecting parts of the container configuration outside the
cluster, and let tools outside the cluster read that to infer the existence of a
local registry.

But ultimately, the semantics of how these values align changes a
lot, and relying on it might be unwise.

### Support for future registry topologies

In the future, this config might apply to remote clusters. For example, the user
might have a remote managed development cluster with an in-cluster registry. Or
each developer might have their own namespace, with an in-cluster registry
per-namespace. The current proposal could potentially expand to include more
registry topologies. But to avoid over-engineering, this proposal doesn't
explore deeply what that might look like.

### Support for secure registries

Remote clusters sometimes support a registry secured by a self-signed
CA. (e.g., [OpenShift's image
controller](https://docs.openshift.com/container-platform/4.4/openshift_images/image-configuration.html#images-configuration-parameters_image-configuration)).
A future version of this config might contain a specification for how to share
the appropriate certificates and auto-load them into image push tooling. But
there's currently no common standard for how to configure image push tools with
these certificates.

This proposal is focused on local, insecure registries. If a cluster offers a
secure registry, they can use the `help` field to instruct the user how to
configure their tools to use it.

### Support for other local image loaders

Many local clusters support multiple methods for loading images, with different
trade-offs (e.g., `kind load`, exposing the Docker socket directly, etc).  Local
cluster maintainers have expressed interest in advertising these other loaders,
and have often written lengthy documentation on how to use them.

An earlier version of this proposal considered a single ConfigMap with a field
for each image loading method. Tools would read them all at once and pick
between them.

But it seemed too early to add other image loaders. The big concern is
interoperability. If two clusters expose the same image loading mechanism, tools
will interact with them in the same way.

Local registries are the one mechanism that seem to have critical mass right now and
we can guarantee interoperability for, because registry behavior is well-specified.

In the future, there might be other mechanisms to gather all the image loading methods 
(e.g., creating a ConfigMap for each method and applying a common label.)

## Infrastructure Needed (optional)

<!--
Use this section if you need things from the project/SIG.  Examples include a
new subproject, repos requested, github details.  Listing these here allows a
SIG to get the process for these resources started right away.
-->

None
