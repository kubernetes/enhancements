# Building a Dockerless Kubelet

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [History](#history)
- [Returning to Motivation](#returning-to-motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [X] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [X] KEP approvers have set the KEP status to `implementable`
- [X] Design details are appropriately documented
- [X] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] Graduation criteria is in place
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This proposal outlines a plan to enable building a dockerless Kubelet. We
define a dockerless Kubelet as a Kubelet with no "Docker-specific" code and
no dependency on the `docker/docker` Golang package. We define "Docker-specific"
code as code which only serves a purpose when Docker is the container runtime.
"Docker-specific" code is never executed when the Kubelet uses a remote
container runtime (i.e. containerd or CRI-O).

Supporting building a dockerless Kubelet is a precursor to moving all
"Docker-specific" Kubelet code out-of-tree, in the name of treating Docker like
any other container runtime.

At a high level, this undertaking is similar to the efforts of
sig-cloud-provider to support [Building Kubernetes Without In-Tree Cloud Providers](/keps/sig-cloud-provider/20190729-building-without-in-tree-providers.md).
A big thanks to them for their leadership; much of this KEP is based off their great work :)

## Motivation

For this KEP to be worthwhile, we must believe the following two statements.

First, we must see a benefit to a dockerless Kubelet.
Second, we must believe supporting the ability to compile a dockerless Kubelet is
a useful first step towards a truly dockerless Kubelet.

A quick review of recent Kubernetes history provides context when considering
whether we agree with the statements in question.

### History

With 1.5, Kubernetes introduced the [Container Runtime Interface](https://kubernetes.io/blog/2016/12/container-runtime-interface-cri-in-kubernetes/)
(CRI). The CRI defines a standard interface for the Kubelet to communicate with container
runtimes. At the time of the CRI's release, Kubernetes supported only the Docker
and rkt container runtimes. Kubernetes introduced the CRI to avoid needing
provider specific code for each new container runtime.

Since the CRI's release, the Kubelet only interacts with container runtimes via
the CRI. For increasingly popular CRI implementations like containerd or CRI-O,
the singular focus on the CRI poses no obstacles, as these container runtimes
supported the CRI from the start. However, the Kubelet still needed a solution for
Docker and rkt, in-tree runtimes which did not support the CRI. In-tree support
for Rkt was deprecated, leaving only Docker as an issue.

Ultimately, the Kubelet introduced the `dockershim` to address Docker's lack of
CRI support. When a cluster operator chooses to use Docker as the container
runtime, the Kubelet starts running the `dockershim` in a separate go
routine within the main `kubelet` process; it is not currently possible to run
the `dockershim` as a standalone binary/process. The `dockershim` supports the
CRI, so the Kubelet communicates with `dockershim` as if it was any other remote
container runtime implementing the CRI. The `dockershim` makes the appropriate
calls to the Docker daemon via a heavy dependence on the `docker/docker` client library.

The `docker/docker` client library is a particularly painful dependency because
its pulls in code from many different open source libraries. For those managing
k8s dependencies, it can be extremely difficult to keep up with the changes to
all these dependencies. Additionally, all the open source libraries required by
`docker/docker` bloat the Kubelet binary.

## Returning to Motivation

We can anticipate the following benefits from the Kubelet having no in-tree
"Docker-specific" code and no dependency on the `docker/docker` Golang package.

First, a dockerless Kubelet would truly treat all container runtimes the same.
Second, the Kubelet's scope of responsibility would decrease: it would no longer
be responsible for making Docker conform to the CRI. Finally, the painful
`docker/docker` dependency would be gone.

While the aforementioned benefits are desirable, they do not outweigh the cons
of completely dropping support for Docker as a container runtime, as
Docker remains popular. In order for Kubernetes
to both support Docker as a container runtime, and for the Kubelet do have no
in-tree "Docker-specific" code and no dependency on `docker/docker`, one of two
following paths must be followed: either Docker must begin implementing the CRI natively
or the `dockershim` must be moved out-of-tree into a standalone component.

Clearly, both of these paths forward require significant work. Either effort would
require finding an owner, making non-trivial code changes, and updating patterns
of cluster management/operation.

Faced with a hefty chunk of work, we naturally try to break it up into smaller
components. This desire leads us to our second question: is supporting
compiling a dockerless Kubelet an appropriate first step?

We argue yes. First, the work to support compiling a dockerless Kubelet will
be useful regardless of which path forward we chose. First, to compile a dockerless
Kubelet we must consolidate all "Docker-specific" Kubelet code into specific
locations, which are easier to move out-of-tree or delete entirely when the time comes. Furthermore,
after this initial consolidation, we can create tooling to impose limitations on
where "Docker-specific" code can/cannot live. Second, allowing developers to
compile a dockerless Kubelet assists in testing either proposed solution.
Finally, allowing compiling a dockerless Kubelet allows projects/cluster
operators which already do not depend on Docker to obtain the dockerless
Kubelet's benefits (i.e. smaller binaries) without waiting for the
completion of the longer term projects to make Docker support the CRI or
move `dockershim` out-of-tree.

### Goals

Our goals follow from our motivation:

1. Support building Kubelet, from the `master` branch, without any "Docker-specific" code and without any
   dependency on `docker/docker`. As mentioned
   previously, we imagine the resulting binaries to be used to test the
   different paths for deleting/moving out-of-tree all "Docker-specific" code.
2. Draw clear delineations, with CI support, for what code in Kubelet can and
   cannot be "Docker-specific" and depend on `docker/docker`.

### Non-Goals

Our non-goals also follow from our motivation:

1. Either making Docker support the CRI or moving `dockershim` out-of-tree.
2. Removing uses of the `docker/docker` Golang library outside of the Kubelet.
3. Changing the official Kubernetes release builds.

## Proposal

We will undertake the following steps to obtain our goals. First, we will ensure
that all "Docker-specific" code in the Kubelet lives in `dockershim`. Then, we
will add a [build constraint](https://golang.org/pkg/go/build/#hdr-Build_Constraints)
to the Kubelet for a pseudo "build tag" specifying not to include
any in-tree "Docker-specific" code. If builds do not specify this build tag, the Go
compiler will compile the Kubelet as normal. If we do, the Go compiler will
compile the Kubelet without the "Docker-specific" code, and as a result, without
the dependency on `docker/docker`. In other words, it will simulate the aforementioned
code/dependency's removal.

A prototype is available in [kubernetes/kubernetes#87746](https://github.com/kubernetes/kubernetes/pull/87746).

To ensure that this dockerless Kubelet continues to function we will add CI building in this mode,
and CI running end to end tests against it (to be elaborated on in the test plan). Additionally, to
ensure the Kubelet doesn't introduce new dependencies on the `docker/docker`
Golang library, we will add automated tooling enforcing that only the
`dockershim` can depend on `docker/docker`.

One quick additional note - currently `cadvisor` also depends on the
`docker/docker` client library. Since `kubelet` depends on `cadvisor`, the
`kubelet` can not truly be rid of the `docker/docker` client library until `cadvisor` no
longer depends on `docker/docker`. Work to remove the `docker/docker` dependency
from `cadvisor` is being dealt with in separate workstreams.

This proposal follows the previous patterns for similar work, namely the efforts
of sig-cloud-provider to support [Building Without In-Tree Cloud Providers](/keps/sig-cloud-provider/20190729-building-without-in-tree-providers.md).

### User Stories

#### Story 1

As a developer working to make Docker function like any other remote container
runtime, I am attempting to validate that my proposed solution
functions correctly. Using this dockerless build ensures the Kubelet contains no
"Docker-specific" code, and any success/failures can be attributed to my
implementation.

### Implementation Details/Notes/Constraints

A couple high level notes (to be extended over time):

- We implement this functionality using a synthetic `dockerless` tag in go build
  constraints on relevant sources. If `GOFLAGS=-tags=dockerless` during build,
  then "Docker-specific" code will be excluded from the Kubelet build. With no
  "Docker-specific" code, we should also be excluding the dependency on
  `docker/docker`.

### Risks and Mitigations

This feature is only developer facing, which removes a large class of risks.

The largest remaining risk is that the build tags fall out of date, or are
burdensome to continue updating, which leads to the dockerless Kubelet build
breaking and/or being costly to maintain. This risk grows the longer the
dockerless Kubelet exists (i.e. the longer it takes to move `dockershim` out of
tree/have Docker support the CRI). Fortunately, these risks can be mitigated via
the CI tooling discussed earlier.

## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

We envision the following testing:

1. A verification, run during pre-submit, which ensures the Kubelet builds when the `dockerless` tag is
   enabled.
2. A verification, run during pre-submit, which ensures that only the imports of
   `github.com/docker/docker` occur within `pkg/kubelet/dockershim`.
3. Unit tests, run during pre-submit, which execute the standard `pkg/kubelet/...`
   unit tests with the `dockerless` tag enabled.
4. A e2e test, run during pre-submit, which executes a simple node e2e test w/ a
   Kubelet compiled with the `dockerless` tag enabled.

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

Since this is a developer facing change, I don't believe there is any graduation
criteria.

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Implementation History

- original KEP PR [kubernetes/kubernetes#87746](https://github.com/kubernetes/kubernetes/pull/87746) was merged on 2020-05-09.

## Drawbacks

One drawback is the opportunity cost of pursuing this workstream as opposed to
other possible workstreams.

Another drawback is the slight additional cost of the CI tooling we propose
adding.

## Alternatives

One alternative would be to do nothing.

Another alternative could be waiting to address "Docker-specific" code in the
Kubelet until we have more momentum around one of the longer-term solutions
discussed above. If we waited, we could delete "Docker-specific" code entirely,
instead of just compiling without it.

Finally, we could attempt to have a long-running branch in which all
"Docker-specific" code has been deleted, instead of attempting to support
compiling a dockerless Kubelet from master.
