---
title: Apiserver discovery URL env variable
authors:
  - "@anguslees"
owning-sig: sig-network
participating-sigs:
  - sig-apimachinery
  - sig-architecture
reviewers:
  - "@thockin"
  - "@liggitt"
approvers:
  - TBD
editor: TBD
creation-date: 2019-12-20
last-updated: 2019-12-20
status: provisional
see-also:
  - "/keps/sig-apps/0028-20180925-optional-service-environment-variables.md"
replaces:
  - "/keps/sig-ccc/20181231-replaced-kep.md"
superseded-by:
  - "/keps/sig-xxx/20190104-superceding-kep.md"
---

# Title

Apiserver Discovery URL Environment Variable

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories [optional]](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Examples](#examples)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
      - [Removing a deprecated flag](#removing-a-deprecated-flag)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

Currently in-cluster API clients discover the apiserver endpoint using
the `KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT`
environment variables.  This KEP introduces a new
`KUBERNETES_SERVICE_URL` environment variable and proposes deprecating
the older environment variables.

## Motivation

`KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT` cannot be
naively concatenated into a URL when `KUBERNETES_SERVICE_HOST` is an
IPv6 literal address.  As introduced by [RFC 2732], IPv6 literal
addresses must be escaped with square brackets when forming a URL to
avoid ambiguity with URL port numbers - for example
`http://[1080::8:800:200C:417A]:6443`.  This is unlike IPv4 literals
or hostnames (even when hostnames resolve to IPv6 addresses), which
cannot themselves contain colon characters.

[RFC 2732]: https://tools.ietf.org/html/rfc2732

Concatenating the `KUBERNETES_SERVICE_HOST` and `..._PORT`
_incorrectly_ is a common error.  See for example the [java], [ruby],
[haskell], and [javascript] clients.  This is particularly difficult
in shell scripts, where there is no helper library available.

[java]: kubernetes-client/java#839
[ruby]: kubernetes-client/ruby#51
[haskell]: kubernetes-client/haskell#68
[javascript]: kubernetes-client/javascript#380

Secondarily, the general `FOO_SERVICE_HOST` and `FOO_SERVICE_PORT`
environment variables are a docker-compose-era legacy that works pooly
in Kubernetes, and we would like to [remove support for them one
day][servicevarskep].  Replacing the in-cluster Kubernetes API
discovery logic removes the last "need" for this family of environment
variables.

[servicevarskep]: /keps/sig-apps/0028-20180925-optional-service-environment-variables.md

### Goals

- Provide a pointer to the in-cluster API server endpoint that "just
  works" for IPv4, IPv6 and hostname forms.

### Non-Goals

- Actually remove `FOO_SERVICE_HOST`/`FOO_SERVICE_PORT` variables.
  This is covered by [other KEPs][servicevarskep].

## Proposal

1. Unconditionally inject a new environment variable into pods called
   `KUBERNETES_SERVICE_URL`.  The value would be the correctly-escaped
   equivalent of
   `https://$KUBERNETES_SERVICE_HOST:$KUBERNETES_SERVICE_PORT`. Note
   that this would continue to use a (single) IPv4 or v6 literal
   address in most cases.

   An open question is whether the URL could ever include a path
   component.  This KEP makes no statement about that, but notes the
   choice of URL format leaves this and alternate schemes as possible
   extension points in future (unlike just "host:port" format).

2. Announce the deprecation of the existing `KUBERNETES_SERVICE_HOST`
   and `KUBERNETES_SERVICE_PORT`.  Given the widespread client impact
   this KEP does not propose a timeline on actual removal, just that
   in-cluster clients SHOULD use the new `KUBERNETES_SERVICE_URL`
   mechanism.

### User Stories [optional]

#### Story 1

As a client developer, I would like to just use a discovery mechanism
without having to think about IPv6 syntax details.

#### Story 2

As a cluster admin, I would like to enable an apiserver IPv6 endpoint
without encountering widespread URL bugs.

### Implementation Details/Notes/Constraints [optional]

What are the caveats to the implementation?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they releate.

### Risks and Mitigations

What are the risks of this proposal and how do we mitigate.
Think broadly.
For example, consider both security and how this will impact the larger kubernetes ecosystem.

How will security be reviewed and by whom?
How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.

## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.
Anything that would count as tricky in the implementation and anything particularly challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage expectations).
Please adhere to the [Kubernetes testing guidelines][testing-guidelines] when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

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
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

#### Examples

These are generalized examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

##### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

##### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and GA/stable, since there's no opportunity for user feedback, or even bug reports, in back-to-back releases.

##### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md

### Upgrade / Downgrade Strategy

If applicable, how will the component be upgraded and downgraded? Make sure this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to make use of the enhancement?

### Version Skew Strategy

If applicable, how will the component handle version skew with other components? What are the guarantees? Make sure
this is in the test plan.

Consider the following in developing a version skew strategy for this enhancement:
- Does this enhancement involve coordinating behavior in the control plane and in the kubelet? How does an n-2 kubelet without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI, CRI or CNI may require updating that component before the kubelet.

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks

The biggest drawbacks of this proposal are that several
[alternatives](#alternatives) are also appealing.  It is not clear
that _this_ proposal is the right tradeoff since it is neither the
most minimal, nor the "best".

## Alternatives

There are several fine alternatives to this proposal:
- Continue to use KUBERNETES_SERVICE_HOST, but change it to
  `kubernetes.default` or other hostname instead of an IP (v4 or v6)
  address.

  Cons: This requires DNS, but is an even simpler change than this KEP.

- Declare that the endpoint is always `kubernetes.default:443` (ie:
  ignore environment variables)

  Cons: Declaring host is probably ok. Declaring port is probably not.

- Declare the endpoint is always `kubernetes.kube-system:443`
  (or some similar new/better clusterdns hostname)

- Declare the endpoint is always `_kubernetes._tcp` DNS `SRV`.
  Cool.  Unfortunately SRV is not trivial to use in a URL.

- Declare a well-known (anycast) IPv4/IPv6 address and port.
  Like some cloud providers did for 169.254.169.254 instance metadata.

  Cons: This has numerous issues for instances. We should not repeat
  these mistakes given the alternatives available.

- Write a file to a well-known path in the pod containing the
  apiserver endpoint information (perhaps in
  `/var/run/secrets/kubernetes.io/serviceaccount`).  If this was in
  kube/config format, and we set `KUBECONFIG` to the path then all the
  client in-cluster vs out-of-cluster bootstrap code goes away.

  Cons: Larger change.
