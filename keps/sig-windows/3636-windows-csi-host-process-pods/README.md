<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [x] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-3636: CSI Drivers in Windows as HostProcess Pods

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
  - [Glossary](#glossary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Prerequisite: Make CSI Proxy an embedded library without a server component](#prerequisite-make-csi-proxy-an-embedded-library-without-a-server-component)
  - [Preferred option: Update the CSI Drivers to use the server code directly](#preferred-option-update-the-csi-drivers-to-use-the-server-code-directly)
  - [Alternative: Update the translation layer to use the server code gRPC](#alternative-update-the-translation-layer-to-use-the-server-code-grpc)
  - [Alternative: Convert CSI Proxy to a Library of Functions](#alternative-convert-csi-proxy-to-a-library-of-functions)
  - [Comparison Matrix](#comparison-matrix)
  - [Maintenance of the new model and existing client/server model of CSI Proxy](#maintenance-of-the-new-model-and-existing-clientserver-model-of-csi-proxy)
  - [Security analysis](#security-analysis)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
-   ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.
-->

In Kubernetes, CSI enables third-party storage providers to write and deploy plugins without needing
to alter the core Kubernetes codebase. 

A CSI Driver in Kubernetes has two main components: a controller plugin that runs in
the control plane and a node plugin that runs on every node.

The node plugin requires direct access to the host to make block devices and/or filesystems
available to the kubelet. In Linux-based nodes, CSI Drivers use the [mkfs(8)](https://man7.org/linux/man-pages/man8/mkfs.8.html)
and the [mount(8)](https://man7.org/linux/man-pages/man8/mount.8.html) commands to format and mount filesystems.

In Windows-based nodes, a node plugin cannot execute similar Windows commands due to the missing capability
of running privileged operations from a container. To solve this issue, the CSI community created a proxy binary
called [CSI Proxy](https://kubernetes.io/blog/2020/04/03/kubernetes-1-18-feature-windows-csi-support-alpha/) which
performs privileged storage operations on behalf of the CSI Driver. First, cluster administrators run the CSI Proxy
on the node as a service. Next, CSI Drivers connect to named pipes set up by CSI Proxy and issue commands through a gRPC API.
CSI Proxy then runs privileged PowerShell commands to mount and format filesystems. This strategy was adopted by
several CSI Drivers that want to support Windows nodes and eventually led to
[CSI Proxy becoming stable and GA in Kubernetes 1.22](https://kubernetes.io/blog/2021/08/09/csi-windows-support-with-csi-proxy-reaches-ga/).

In 2021, the SIG Windows community introduced a feature called [HostProcess containers](https://kubernetes.io/blog/2021/08/16/windows-hostprocess-containers/).
This feature enables running Windows Process-isolated containers (hence the name HostProcess container).

With this feature, a CSI Driver node plugin can run as a HostProcess container and issue the privileged
storage operation directly without a proxy binary. This KEP explains the implementation details
of how Windows-based node plugins can adopt HostProcess containers and the evolution of
CSI Proxy from a client/server-based proxy to a library of privileged storage operation functions similar to
[kubernetes/mount-utils](https://github.com/kubernetes/mount-utils).

### Glossary
 
Terms used in this document:

* API Group - A grouping of APIs in CSI Proxy by purpose. For example, the Volume API Group has API Methods related to volume interaction.
  [There are 4 API groups (Disk, Filesystem, Volume, SMB) in v1 status and 2 API Groups in v1beta status](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/README.md#feature-status).
* API Version - An API Group can have multiple versions.
  [The versions include v1alpha1, v1beta1, v1beta2, v1beta3, v1, v2alpha1](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/README.md#feature-status).
* Translation layer - Generated Go code in CSI Proxy that transforms client versioned requests to server "version-agnostic" requests.
* CSI Proxy server - The CSI Proxy binary running in the host node.
* CSI Proxy client - The Go module client used by CSI Drivers and addons to connect to the CSI Proxy server.
* CSI Proxy v1 - The CSI Proxy implementation using the client/server model.
* CSI Proxy v2 - The CSI Proxy implementation using a Go module imported by CSI drivers.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

The client/server model of CSI Proxy enabled running privileged storage operations from CSI Node plugins
in Windows nodes. The server is the CSI Proxy binary running as a Windows service in the node,
and the client is the CSI Driver node plugin, which makes an RPC request to CSI Proxy on
most node CSI operations. While this model works, it has a few drawbacks:

- **Different deployment model than Linux** - Linux privileged containers perform the privileged storage
  operations (format/mount). However, Windows containers aren't privileged. To work around the problem, the CSI Driver runs as non-privileged containers,
  and privileged operations are relayed to CSI Proxy. In deployment manifests, the Windows component needs an
  additional section to mount the named pipes exposed by CSI Proxy as a hostpath.
- **Additional component in the host to maintain** - The cluster administrator needs to install and run CSI Proxy
  during node bootstrap. The cluster administrator also needs to think about the upgrade workflow in addition to
  upgrading the CSI Driver.
- **Difficult releases of bugfixes & features** - After a bugfix, we create a new version of the CSI Proxy to be
  redeployed in the cluster. After a feature is merged, in addition to redeploying a new version of CSI Proxy,
  the client needs to be updated with a new version of the CSI Proxy client and connect to the new version of the named pipes.
  This workflow is not as simple as the Linux counterpart, which only needs to update Go dependencies.
- **Multiple API versions to maintain** - As part of the original design of CSI Proxy, it was decided to have different
  protobuf versions whenever there were breaking changes (like updates in the protobuf services & messages). This led
  to having multiple versions of the API (v1alphaX, v1betaX, v1). In addition, if we want to add a new feature, we'd need
  to create a new API version e.g. v2alpha1 ([see this PR as an example of adding methods to the Volume API Group](https://github.com/kubernetes-csi/csi-proxy/pull/186)).
  it includes an API group for handling the SMB protocol, which a CSI Driver might not use.

In 1.22, SIG Windows introduced [HostProcess containers](https://kubernetes.io/blog/2021/08/16/windows-hostprocess-containers/)
as an alternative way to run containers. HostProcess containers run directly in the host
and behave like to a regular process. The HostProcess containers feature became stable in 1.26.

Using HostProcess containers in CSI Drivers enables CSI Drivers to perform the privileged storage operations
directly. Most of the drawbacks in the client/server model are no longer present in the new model.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Define the implementation details to transform CSI Proxy from its client/server model to
  a go module that can be directly imported by CSI Drivers.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
- Improve the performance of CSI Drivers in Windows - There should be an improvement in the performance by
  removing the communication aspects between the CSI Driver and CSI Proxy (the protobuf serialization/deserialization,
  the gRPC call through named pipes). However, this improvement might be negligible, as most of the latency
  comes from running powershell commands, which is outside the scope of this change.
- Deprecate the client/server model - This model is still used by the majority of CSI Driver implementations,
  adopting the new go module model will take time and in the meantime we still plan to maintain
  both models.
- Define strict security implementation details - A goal is to understand the security implications of enabling HostProcess
  containers. We aim to provide guidelines but not implementation details about the components that need to be installed
  in the cluster.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

This proposal advocates for evolving CSI Proxy from a standalone binary into a Go library that CSI drivers can directly import.
By leveraging HostProcess containers, CSI drivers for Windows can bundle the necessary privileged functionality,
eliminating the need for a separate proxy component running on each node.
This shift simplifies the deployment and maintenance model for both driver developers and cluster administrators.

The core of this proposal is to:

- Refactor the CSI Proxy codebase to expose its API groups as importable Go packages, removing the client/server gRPC architecture.
- Provide clear migration guidelines for CSI driver developers to adopt the new library model and transition their
  drivers to use HostProcess containers. This will include code examples, manifest changes, and security best practices.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a CSI driver developer, I want to consume CSI Proxy as a Go library so that I can simplify my driver's architecture,
remove the runtime dependency on an external CSI Proxy binary, and align the Windows deployment model more closely with the Linux model,
reducing maintenance overhead.

#### Story 2

As a Kubernetes cluster administrator, I want CSI drivers for Windows to be self-contained without requiring me to
separately install and manage the CSI Proxy lifecycle on my nodes.
This will simplify node bootstrapping, reduce operational complexity,
and make driver upgrades more straightforward.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->
HostProcess containers run as processes in the host. One of the differences with a privileged Linux container
is that there's no filesystem isolation. This means that enabling HostProcess containers should be done for
system components only. This point will be expanded on in the detailed design.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

Security implications of HostProcess containers will be reviewed by SIG Windows, SIG Storage
and SIG Security.

One risk about enabling the HostProcess containers feature is not having enough security policies in the cluster
for workloads, if workloads can be deployed as HostProcess containers or if there's an escalation that allow
non-privileged pods to become HostProcess containers then workloads have complete access to the host filesystem,
this allows access to the tokens in `/var/lib/kubelet` as well as the volumes of other pods inside `/var/lib/kubelet/`.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

CSI Proxy has a client/server design with two main components:

* a binary that runs in the host (the CSI Proxy server). This binary can execute privileged storage operations on the
  host. Once configured to run as a Windows service, it creates named pipes on startup for all the versions of the API
  Groups defined on the codebase.
* client Go libraries that CSI Drivers and Addons import to connect to the CSI Proxy server. The methods and objects
  available in the library are defined with [protobuf](https://github.com/kubernetes-csi/csi-proxy#feature-status). On
  startup, the CSI Driver initializes a client for each version of the API Groups required, which will connect and issue
  requests through gRPC to their pre-configured named pipes on the host.

CSI Driver implementers can write a Windows-specific implementation of the node component of the CSI Driver. In the
implementation, a CSI Driver will make use of the imported CSI Proxy client libraries to issue privileged storage
operations. Assuming that a volume was created and attached to a node by the controller component of the CSI Driver,
the following CSI calls will be done by the kubelet to the CSI Driver.

**Volume set up**

* NodeStageVolume - Create a Windows volume, format it to NTFS, and create a partition access path in the node (global mount).
* NodePublishVolume - Create a symlink from the kubelet Pod-PVC path to the global path (pod mount).

**Volume tear down**

* NodeUnpublishVolume - Remove the symlink created above.
* NodeUnstageVolume - Remove the partition access path.

CSI Proxy is designed to be backwards compatible, and a single binary running in the Windows node can serve requests from
multiple CSI Proxy client versions. We're able to do this, because the CSI Proxy binary will create named
pipes on startup for all the versions available in every API Group (e.g. the Volume, Disk, Filesystem, SMB groups). In addition,
there's a translation layer in the CSI Proxy binary that transforms client version specific requests to server "version
agnostic" requests, which are then processed by the CSI Proxy binary. The following diagram shows the conversion process
(from the [CSI Proxy development docs](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/docs/DEVELOPMENT.md)):

![CSI Proxy client/server model](./csi-proxy-client-server.jpg)

Understanding the translation layer will help in the transition to HostProcess containers, as most of the code that the
clients use to communicate with the CSI Proxy server is generated. The translation layer's objective is to generate Go code
that maps versioned client requests to server agnostic requests. It does so by analyzing the generated `api.pb.go`
files (generated through `protoc` from the protobuf files) for each version of the API Groups and generating multiple
files for different purposes (taking as example the Volume API Group):


* [\<version\>/server_generated.go](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/pkg/server/volume/impl/v1beta3/server_generated.go)
  - The gRPC server implementation of the methods of a versioned API Group. Each method receives a versioned request and
    expects a versioned response. The code generated follows this pattern:

```
func v1Foo(v1Request v1FooRequest) v1FooResponse {

  // convert versioned request to server request (version agnostic)
  fooRequest = convertV1FooRequestToFooRequest(v1Request)

  // process request (server handler)
  fooResponse = server.Foo(fooRequest)

  // convert server response (version agnostic) to versioned response
  v1Response = convertFooResponseToV1FooResponse(fooResponse)

  return v1Response
}
```


* [types_generated.go](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/pkg/server/volume/impl/types_generated.go)
  The idea is to collect all the methods available across all the versions of an API Group so that the server has a
  corresponding implementation for it. The generator reads all the methods found across the
  `volume/<version>/api.pb.go` files and generates an interface with all the methods found that the server must
  implement, in the example above the server interface will have the `Foo` method
* [\<version\>/conversion_generated.go](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/pkg/server/volume/impl/v1/conversion_generated.go)
  The generated implementation of the conversion functions shown above (e.g. `convertV1FooRequestToFooRequest`,
  `convertFooResponseToV1FooResponse`). In some cases, it's possible that the conversion code generator generates a nested
  data structure that's not built correctly. There's an additional file with overrides for the functions that were
  generated incorrectly.
* Client [\<API Group\>/\<version\>/client_generated.go](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/client/groups/volume/v1/client_generated.go)
  Generated in the client libraries to be used by users of the CSI Proxy client. It creates proxy methods corresponding
  to the `api.pb.go` methods of the versioned API Group. This file defines the logic to create a connection to the
  corresponding named pipe, creating a gRPC client out of it and storing it for later usage. As a result, the proxy
  methods don't need a reference to the gRPC client.


### Prerequisite: Make CSI Proxy an embedded library without a server component

If we configure the Windows node component of a CSI Driver/Addon to be a Windows HostProcess pod, then it'll be able to
use the same powershell commands that we use in the server code of CSI Proxy. The idea is to use the server code of CSI
Proxy as a library in CSI Drivers/Addons. With this, we also remove the server component.

As described in the [Windows HostProcess Pod](https://kubernetes.io/docs/tasks/configure-pod-container/create-hostprocess-pod/)
guide, we'd need to configure the PodSpec of node component of the CSI Driver/Addon that runs in Windows nodes with:


```yaml
spec:
  securityContext:
    windowsOptions:
      hostProcess: true
      runAsUserName: "NT AUTHORITY\\SYSTEM"
```

### Preferred option: Update the CSI Drivers to use the server code directly

Modify the client code to use the server API handlers directly which would call the server implementation next, this
means that the concept of an "API version" is also removed from the codebase, the clients instead would import and use
the internal server structs (request and response objects).

Currently, GCE PD CSI driver uses the v1 Filesystem API group as follows:


```go
// note the API version in the imports
import (
  fsapi "github.com/kubernetes-csi/csi-proxy/client/api/filesystem/v1"
  fsclient "github.com/kubernetes-csi/csi-proxy/client/groups/filesystem/v1"
)

func NewCSIProxyMounterV1() (*CSIProxyMounterV1, error) {
  fsClient, err := fsclient.NewClient()
  if err != nil {
    return nil, err
  }
  return &CSIProxyMounterV1{
    FsClient:     fsClient,
  }, nil
}

// ExistsPath - Checks if a path exists. Unlike util ExistsPath, this call does not perform follow link.
func (mounter *CSIProxyMounterV1) PathExists(path string) (bool, error) {
  isExistsResponse, err := mounter.FsClient.PathExists(context.Background(),
    &fsapi.PathExistsRequest{
      Path: mount.NormalizeWindowsPath(path),
    })
  if err != nil {
    return false, err
  }
  return isExistsResponse.Exists, err
}

// usage
csiProxyV1, _ := NewCSIProxyMounterV1()
csiProxyV1.PathExists(path)
```


Internally the `PathExists` call is in the file [\<API Group\>/\<version\>/client_generated.go](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/client/groups/volume/v1/client_generated.go)
described above, which performs the execution through gRPC. In the proposal we'd need to use the server implementation
instead:

```go
// note that there is no version in the import
import (
  fsapi "github.com/kubernetes-csi/csi-proxy/pkg/os/filesystem"
  fsserver "github.com/kubernetes-csi/csi-proxy/pkg/server/filesystem"
  fsserverimpl "github.com/kubernetes-csi/csi-proxy/pkg/server/filesystem/impl"
)

// no need to initialize a gRPC client, however the server handler impl is initialized instead
// no need for a versioned client

func NewCSIProxyMounter() (*CSIProxyMounter, error) {
  fsServer, err := fsserver.NewServer(fsapi.New())
  if err != nil {
    return nil, err
  }
  return &CSIProxyMounter{
    FsServer: fsServer,
  }, nil
}

// ExistsPath - Checks if a path exists. Unlike util ExistsPath, this call does not perform follow link.
func (mounter *CSIProxyMounter) PathExists(path string) (bool, error) {
  isExistsResponse, err := mounter.FsServer.PathExists(context.Background(),
    &fsserverimpl.PathExistsRequest{
      Path: mount.NormalizeWindowsPath(path),
    },
    // 3rd arg is the version, remove the version here too!
    )
  if err != nil {
    return false, err
  }
  return isExistsResponse.Exists, err
}

// usage
csiProxy, _ := NewCSIProxyMounter()
csiProxy.PathExists(path)
```

![csi-proxy-library](./csi-proxy-library.jpg)

Pros:

* We remove the concept of API Version & the translation layer and instead consider the go mod version as the API
  version. This is how other libraries like [k8s.io/mount-utils](https://github.com/kubernetes/mount-utils) work.
    * Version dependent server validation in the API handler layer is removed.
    * Legacy structs for older API versions are removed.
* New APIs are easier to add. Only the server handler & impl code is modified, so there’s no need for the code
  generation tool anymore.

Cons:

* The client goes through a bigger diff. Every occurrence of a call to a CSI Proxy method needs to be modified to use
  the server handler & impl code, but this penalty is paid only once.
    * Legacy interface implementations for the v1beta API in the CSI Drivers are removed.
* As we no longer use protobuf to define the API and use internal structs instead, we'd need to update the API docs to
  be directly generated from source code (including the comments around server handler methods and internal server
  structs).

It is worth noting that at this point, the notion of a server is no longer valid, as CSI Proxy has become a
library. We can take this opportunity to reorganize the packages by

1. Moving `/pkg/server/<API Group>` and `/pkg/server/<API Group>/impl` to `/pkg/<API Group>`
2. Moving `/pkg/os/<API Group>` to `/pkg/<API Group>/api`

The new structure looks like:


```
pkg
├── disk
│   ├── api
│   │   ├── api.go
│   │   └── types.go
│   ├── disk.go
│   └── types.go
├── iscsi
│   ├── api
│   │   ├── api.go
│   │   └── types.go
│   ├── disk.go
│   └── types.go
```

There are also three minor details we can take care of while we’re migrating:

1. The two structs under `pkg/shared/disk/types.go` are only ever referenced by `pkg/os/disk`, so they can be safely added
   to `pkg/disk/api/types.go`.
2. The FS server receives `workingDirs` as an input, in addition to the OS API. It’s only used to sandbox what directories
   the CSI Proxy is enabled to operate on. Now that control is part of the CSI Driver, we can safely remove it.
3. `pkg/os/filesystem` is no longer necessary, as the implementation just calls out to the Golang standard library os
   package. We can deprecate it in release notes and remove it in a future release.

### Alternative: Update the translation layer to use the server code gRPC

Modify the implementation of [\<API Group\>/\<version\>/client_generated.go](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/client/groups/volume/v1/client_generated.go)
so that it calls the server implementation directly (which should be part of the imported go module). The current
implementation uses `w.client` which is the gRPC client:


```go
func (w *Client) GetVolumeStats(
  context context.Context,
  request *v1.GetVolumeStatsRequest,
  opts ...grpc.CallOption
) (*v1.GetVolumeStatsResponse, error) {
  return w.client.GetVolumeStats(context, request, opts...)
}
```


The new implementation should use the server code instead. In the server code, `volumeserver` is the implementation agnostic server that's instantiated by every versioned client `volumeservervX`. E.g.,


```go
import (
  v1 "github.com/kubernetes-csi/csi-proxy/client/api/volume/v1"
  volumeserver "github.com/kubernetes-csi/csi-proxy/pkg/server/volume"
  volumeserverv1 "github.com/kubernetes-csi/csi-proxy/pkg/server/volume/v1"
)

// initialize all the versioned volume servers i.e. do what cmd/csi-proxy does but on the client
serverImpl := volumeserver.NewServer()

// shim that would need to be auto generated for every version
serverv1 := volumeserverv1.NewVersionedServer(serverImpl)

// client still calls the conversion handler code
func (w *Client) GetVolumeStats(
  context context.Context,
  request *v1.GetVolumeStatsRequest
) (*v1.GetVolumeStatsResponse, error) {
  return serverv1.GetVolumeStats(context, request)
}
```

![csi-proxy-reuse-client-server-pod](./csi-proxy-reuse-client-server-pod.jpg)

Pros:

* We get to reuse the protobuf code.
* We would still support the client/server model, as this is a new method that clients would use.
* We only need to change the client import paths to use the alternative version that doesn't connect to the server with
  gRPC, which minimizes the changes necessary in the client code.

Cons:

* New APIs would need to be added to the protobuf file, and we would need to run the code generation tool again, with
  the rule of not modifying already released API Groups. This means that we would also need to create another API Group
  version for a new API.
* We still have two distinct concepts of version: the Go module version and the API version. Given that we want to use
  CSI Proxy as a library, it makes sense to use the Go module version as the source of truth and implement a single API
  version in each Go version.

### Alternative: Convert CSI Proxy to a Library of Functions

With the new changes, CSI Proxy is effectively just a library of Go functions mapping to Windows commands. The notion of
servers and clients is no longer relevant, so it makes sense to restructure the package into a library of functions,
with each API Group’s interfacing functions and types provided under `pkg/<API Group>` (right now, these files sit at
`pkg/server/<API Group>/server.go` and `pkg/server/<API Group>/impl/types.go`). The OS-facing API at `/pkg/os` is kept
is, and the corresponding OS API struct is initialized globally inside each `pkg/<API Group>` (to allow for subbing
during testing). All other code can be safely deleted.

```go
// there is now only one single import
import fs "github.com/kubernetes-csi/csi-proxy/pkg/fs"

// there is no longer a need to initialize a server
func NewCSIProxyMounter() *CSIProxyMounter {
  return &CSIProxyMounter{
  }
}

// ExistsPath - Checks if a path exists. Unlike util ExistsPath, this call does not perform follow link.
func (*CSIProxyMounter) PathExists(path string) (bool, error) {
  // both mounter.FsServer and fsserverimpl are changed to just fs
  isExistsResponse, err := fs.PathExists(context.Background(),
    &fs.PathExistsRequest{
      Path: mount.NormalizeWindowsPath(path),
    })
  if err != nil {
    return false, err
  }
  return isExistsResponse.Exists, err
}

// usage
csiProxy := NewCSIProxyMounter()
csiProxy.PathExists(path)

// at test time
fs.UseAPI(mockAPI)
// run tests...
fs.ResetAPI()
```

This is the most invasive option of all three. Specifically, we combine the two imports into one and move to a pure
function paradigm. However, the method implementation sees very minimal changes, requiring only import path updates.

Pros:

* Like implementation idea 2, we switch to a single notion of version via Go modules.
* The pure function paradigm more accurately reflects the nature of the new design, which simplifies how clients use the
  library.
* Like implementation idea 2, new APIs are easier to add by moving away from code generation.

Cons:

* There is now an implicit dependency on the os API package-level variable. Testing can still be done by subbing out the
  variable with a mock implementation during test time.
* More work (2 imports -> 1, remove server initialization, replace function call and request type package names) needs
  to be done by clients to adapt to the new change, though it’s not that much more than implementation idea 2. Again,
  the price is only paid once.
* Like impl idea 2, we also need to transition our API doc generation to generate from Go source.


### Comparison Matrix

| ||Preferred option: Update the CSI Drivers to use the server code directly | Alternative: Update the translation layer to use the server code gRPC| Alternative: Convert CSI Proxy to a Library of Functions|
| --- |--- |--- |--- |
| Adoption cost||Considerate (imports and API calls)| Minimal (only changing imports) | Considerate (imports, API calls, and initialization)|
| Future development|Directly add methods to Go code, but leaves legacy notion of “server”| Still need code generation and and protobuf | Directly add functions to Go code. Code base cleaned up|
| Versioning||Go mod version only| Both Go mod version and API version are maintained | Go mod version only|
| Testing|Current tests should still work.| Current tests should still work. | OS API mocking needs to be subbed in, as we have an implicit dependency|
| Support for legacy client/server model|Not supported| Still supported | Not supported|


### Maintenance of the new model and existing client/server model of CSI Proxy

We plan to maintain both versions (the client/server model and the library model)
while the majority of CSI Drivers are in the client/server model.

The `library-development` branch will be used for the development of this model.
We will create release artifacts from the `library-development` branch and use it in CSI Drivers.
Once the library reaches GA, we will create a `v2` from the `library-development` branch and make it the new default.
For compatibility purposes, `master` will still point to the client/server model.

### Security analysis

- Install the Pod Security Admissions controller and use Pod Security Standards
  - Embrace the least privilege principle, quoting [Enforcing Pod Security Standards | Kubernetes](https://kubernetes.io/docs/setup/best-practices/enforcing-pod-security-standards/#embrace-the-principle-of-least-privilege)
    - Namespaces that lack any configuration at all should be considered significant gaps in your cluster security model.
      We recommend taking the time to analyze the types of workloads occurring in each namespace, and by referencing the Pod Security Standards,
      decide on an appropriate level for each of them. Unlabeled namespaces should only indicate that they've yet to be evaluated.
    - Namespaces allowing privileged workloads should establish and enforce appropriate access controls.
    - For workloads running in those permissive namespaces, maintain documentation about their unique security requirements.
      If at all possible, consider how those requirements could be further constrained.
  - In namespaces without privileged workloads:
    - Follow the guidelines in https://kubernetes.io/docs/tasks/configure-pod-container/enforce-standards-namespace-labels/#applying-to-a-single-namespace,
      for example, add the following labels to a namespace:

  ```plain
  kubectl label --overwrite ns my-existing-namespace \
    pod-security.kubernetes.io/enforce=restricted \
    pod-security.kubernetes.io/enforce-version=v1.25
  ```

    - Both the baseline and restricted Pod Security Standards disallows the creation of HPC pods (docs).
- Create a Windows group with limited permissions to create files under the kubelet controlled path `C:\var\lib\kubelet` and set the `runAsUserName` field in the PodSpec to that group.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

The CSI Proxy is implemented out of tree, as such, its testing is not
tied strictly to kubernetes testing.

- **unit tests and integration tests** - Unit and integration tests will be set up
  in the repository through a combination of tests to run in Github Action Windows workers
  and in Kubernetes clusters with Windows nodes created through `kubernetes/test-infra`.
- **e2e tests** - Because CSI Proxy v2 is a library to be used by CSI Drivers it cannot
  be e2e tested on its own. Instead, CSI Driver authors need to ensure that the kubernetes
  external storage test suite passes after integrating their CSI Drivers with CSI Proxy v2.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

For CSI Proxy we already have unit tests inside `pkg/<API Group>`. These tests are run on presubmit for every PR.

Examples:

- [volume tests](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/pkg/volume/volume_test.go)
- [filesystem tests](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/pkg/filesystem/filesystem_test.go)

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

For CSI Proxy, we already have integration tests inside `integrationtests`. These tests are run on presubmit for every PR.

Examples:

- [volume integration tests](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/integrationtests/volume_test.go)
- [filesystem integration tests](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/integrationtests/filesystem_test.go)
- [iscsi integration tests](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/integrationtests/iscsi_test.go)
- [system integration tests](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/integrationtests/system_test.go)
- [smb integration tests](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/integrationtests/smb_test.go)
- [disk integration tests](https://github.com/kubernetes-csi/csi-proxy/blob/c0c6293490fd8aec269685bb4089be56d69921b1/integrationtests/disk_test.go)

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

Because CSI Proxy v2 is a library to be used by CSI Drivers it cannot be e2e tested on its own.
Instead, CSI Driver authors need to ensure that the kubernetes external storage test suite passes after integrating
their CSI Drivers with CSI Proxy v2.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].
-->
Most of the code used by CSI Drivers through CSI Proxy is already GA. This KEP defines a new mechanism to run
the same code that the CSI Driver executes through CSI Proxy directly inside the CSI Driver.
To verify that the new mechanism is mature, we define the following graduation criteria:

#### Alpha

At least 1 CSI Driver uses an alpha release of CSI proxy v2

#### Beta

At least 2 CSI Drivers use a release of CSI Proxy v2

#### GA

At least 2 CSI Drivers use a release of CSI Proxy v2 for at least 1 Kubernetes release (to check on issues during cluster upgrades)

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

We plan to maintain both versions (the client/server model and the go library model)
because the majority of CSI Drivers use the client/server model.
There is no deprecation of the CSI Proxy v1 model.

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->
During the development of a new minor version of the CSI Driver we suggest the following changes:

**CSI Proxy**

- Start a development branch for the upcoming work (`library-development`).
- Refactor the filesystem, disk, volume, system, iSCSI, SMB API Groups out of the current client/server.
- Remove the client/server code from the codebase.
- Update the unit and integration tests to work with the refactored code.
- Run the integration tests in a HostProcess container.
- Update the README and DEVELOPMENT docs.
- Once the above items are completed, we can create an alpha tag in the `library-development` branch to import in CSI Drivers.

**CSI Driver**

- Update the CSI Proxy library to the alpha v2 tag from the `library-development` branch.
- Update the codebase import to use the server implementation directly instead of the client library.
- Update the CSI Driver deployment manifest with the HostProcess container fields in the `PodSpec`.
- Run the e2e tests.

When the CSI Driver is upgraded to the next minor version it'd include the imported CSI Proxy library.

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
CSI Proxy v1 has a different release cycle than the CSI Driver, each CSI Proxy binary has its own version and
supports different CSI Proxy client versions. CSI Proxy v2 is a go library imported by the CSI Drivers
so the responsibility of handling version skew is owned by the CSI Driver.

This component is a dataplane component only and it doesn't need to handle API server version skews,
management of possible version skew for CSI features implemented in the CSI Driver is handled
by the CSI Driver and not CSI Proxy v2.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [x] Other
  - Describe the mechanism:
    - Since this component is out of tree, it's up
      to CSI Driver authors to integrate CSI Proxy v2 with their CSI Drivers.
  - Will enabling / disabling the feature require downtime of the control
    plane?
    - No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
    - Yes, after a node plugin is integrated with CSI Proxy v2
      it needs to be installed in the node. Since the node plugin is a container
      there needs to be a new rollout.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

The CSI Proxy v2 change is for CSI Driver authors and their CSI Drivers,
there is no default behavior changed that affects cluster users.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, by rolling back to an old implementation of the CSI Driver node plugin using the CSI Proxy v1 client.
Since it needs a rollout, there would be downtime of the CSI Driver node plugin.

###### What happens if we reenable the feature if it was previously rolled back?

Nothing, the behavior is constrained to the CSI Driver version only so
the CSI Driver would use the go mod library instead.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

No, because this component is out of tree, CSI Driver authors need to verify
that their CSI Driver handles feature enablement/disablement.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

