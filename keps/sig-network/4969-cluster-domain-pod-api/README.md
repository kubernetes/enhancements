# KEP-4969: Cluster Domain Pod API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Split Domains](#split-domains)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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
  - [ConfigMap a la k3s](#configmap-a-la-k3s)
  - [Dedicated API Resource](#dedicated-api-resource)
  - [kubelet <code>/configz</code>](#kubelet-configz)
  - [Parsing <code>resolv.conf</code>](#parsing-resolvconf)
  - [Exposing it directly over the Downward API](#exposing-it-directly-over-the-downward-api)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
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

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Almost all Kubernetes installations use DNS to access Services (and some Pods).
These Services (and Pods) have Fully Qualified Domain Names (FQDNs) that are
constructed using the format `{service}.{namespace}.svc.{clusterDomain}`,
where `{clusterDomain}` is _typically_ `cluster.local`, but can be reconfigured
by the cluster administrator.

Currently, there is no way for cluster workloads to query for this domain name,
leaving them to either use relative domain names or configure it manually.
Even the apiserver doesn't know the correct value, only each individual kubelet
(and the DNS server).

This KEP proposes adding a new field into the Pod's status with their cluster domain suffix.

## Motivation

Relative domain names are a source of ambiguity: does `get.app` refer to
the domain registry [get.app](https://get.app/) or the Service `get` in `app`?
This also becomes problematic for TLS, since there is no way to distinguish
which of these two cases a certificate applies to.

Fully Qualified Domain Names (FQDNs) can be used to resolve this, by always
specifying the full domain name. However, requiring each workload to configure
the cluster domain is tedious and error-prone, discouraging application 
developers from using them.

Many distributions already provide ways to query for the cluster domain (such as
kubeadm[^prior-art-kubeadm], k3s[^prior-art-k3s], and OpenShift[^prior-art-openshift]).
However, these are all inconsistent with each other, requiring applications to
provide special cases for each.

[^prior-art-kubeadm]: kubeadm (including kind) creates a ConfigMap `kube-system/kubeadm-config` that contains the full kubeadm config, including `networking.dnsDomain`.
[^prior-art-k3s]: k3s creates a ConfigMap `kube-system/clusterdns` that contains it as the `.data.clusterDomain` field. See <https://github.com/k3s-io/k3s/pull/1785>.
[^prior-art-openshift]: OpenShift defines a custom DNS CRD that contains it as the `.status.clusterDomain` field. See <https://docs.openshift.com/container-platform/4.17/networking/dns-operator.html#nw-dns-view_dns-operator>.

It can also be retrieved from the kubelet's `/configz` endpoint, however this is
[considered unstable](https://github.com/kubernetes/kubernetes/blob/9d967ff97332a024b8ae5ba89c83c239474f42fd/staging/src/k8s.io/component-base/configz/OWNERS#L3-L5).

### Goals

- Making it easier to use and generate FQDNs for internally-visible Services.
- Reducing the difference between Kubernetes distributions.

### Non-Goals

- Disallowing relative domain names.
- Modifying DNS resolution.
- Providing a central way to configure the cluster domain name setting across kubelets.
- Exposing all kubelet configuration.

## Proposal

Add a new field to the Pod status, containing the cluster domain suffix.
It should also be accessible via the downward API and `EnvVarSource.fieldRef`.

### User Stories

#### Story 1

The Pod `foo` needs to access its sibling Service `bar` in the same namespace
(`baz`). It adds two `env` bindings:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: foo
  namespace: baz
spec:
  containers:
    - name: foo
      env:
        - name: NAMESPACE
          valueFrom:
            fieldRef: metadata.namespace
        - name: CLUSTER_DOMAIN
          valueFrom:
            fieldRef: status.dns.clusterDomain
```

`foo` can now perform the query by running `curl https://bar.$NAMESPACE.svc.$CLUSTER_DOMAIN/`.

(Of course, in practice this would likely be integrated into the app itself, not
by shelling into bash, but the principle still applies.)

Kubernetes also configures the search domain and ndots, so that `bar` can be
accessed from within its namespace as simply `https://bar`, or from without as 
`https://bar.baz`. However, these shortnames are ambiguous with domain names
used on the public internet. This causes a few knock-on issues:

1. This ambiguity could cause clients to connect to the wrong target. Normally,
   this would be detected by TLS certificate failures, however:
2. This requires internal certificates to be issued for the shortname aliases
   which could potentially be valid internet hostnames, which could make it
   harder to detect such confusion (as well as enabling malicious impersonation).
3. The long search chain provided by Kubernetes to enable this increases DNS
   lookup times (and DNS server load), because it requires clients to try each
   option separately.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

#### Split Domains

Kubernetes currently does not prohibit different kubelets from specifying 
different cluster domains (`node-a` could set `cluster.local` while `node-b` 
specifies `cluster.remote`).
Advertising FQDNs generated using this API could cause issues in these mixed
environments, since `node-b` might not be able to resolve `cluster.local`
FQDNs correctly.

## Design Details

A new field `dns.clusterDomain` would be added to `PodStatus`:

```go
type PodStatus struct {
    // ..existing fields elided
    DNS *PodDNSStatus
}

type PodDNSStatus struct {
    ClusterDomain *string
}
```

The field would be populated by the Kubelet during pod initialization.

The field would also be accessible from the Downward API via `fieldRef`.

@thockin originally suggested that `clusterDomain` be called `zone`. This
proposal uses `clusterDomain` because it matches Kubelet's configuration field,
and because `zone` is somewhat ambiguous (are we talking about the cluster zone?
the namespace's zone?).

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

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

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

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

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

No changes are required to clients, this is a purely additive change.

When upgrading a cluster from a version that did not enable the feature, the
new status field will be set for new Pods launched from then on.

When downgrading a cluster, the status field will not be set anymore for new
Pods.

### Version Skew Strategy

Older kubelets (or kubelets with the feature disabled) will not set the attribute,
and fail to create Pods that request the attribute via the downward API.

Older apiservers will ignore attempts (by the kubelet) to set the attribute, and
reject attempts to create Pods that request the attribute via the downward API.

Applications that want to be compatible with both environments should have a
provision for the field being unset (such as a hard-coded default value, and/or
a manual override flag).

Additionally, such applications should retrieve the value from the Kubernetes
REST API, rather than using the downward API.

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

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PodClusterDomain
  - Components depending on the feature gate:
    - kubelet
    - kube-apiserver

###### Does enabling the feature change any default behavior?

No. Workloads that don't read the status field or request it via the downward
API would be unaffected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

Workloads that expect the feature to be available (and don't have a fallback in
place) would fail, but that has nothing to do with the rollout itself.

###### What happens if we reenable the feature if it was previously rolled back?

It should become available again. Existing Pods should be unaffected.

###### Are there any tests for feature enablement/disablement?

Not yet. The logic should be pretty trivial, but it probably makes sense to at
least test gate mismatches between apiserver and kubelet.

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

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

Pods that specify the downward API would fail to launch if it is no longer
available. This would also impact APIs that create Pods on behalf of users
(such as Deployments, StatefulSets, and third-party operators).

###### What specific metrics should inform a rollback?

Not applicable.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

Kubernetes doesn't (and can't) know who reads what status fields from the API.

Cluster administrators can query the cluster for Pods that request the field
via the downward API by running the following command:

```shell
kubectl get pods --all-namespaces --output=json | jq '.items[] | select(.spec.volumes[]? | [[{downwardAPI}], .projected.sources][][]?.downwardAPI.items[]?.fieldRef.fieldPath == "status.dns.clusterDomain") | .metadata | {name, namespace}'
```

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Other field: dns.clusterDomain is set

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Not applicable, as far as I can tell.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Not applicable, as far as I can tell.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Not applicable, as far as I can tell.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

Enabling the feature should not result in any new API calls (it should be able
to piggyback on the kubelet's existing status and DNS machinery).

Consuming the feature over the REST API may require some clients to request
their own Pod object during Pod startup.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

The new field `.status.dns.clusterDomain` would be added to all Pod objects.

For the default value of `cluster.local`, this would add 42 bytes to the 
minified JSON serialization of each Pod.

(`echo -n ',{"dns":{"clusterDomain":"cluster.local"}}' | wc -c`, assuming that
`.status` is already set, but `.status.dns` is not.)

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Beyond the size increase noted above, no.

The information is already stored locally, and required to generate the
`resolv.conf` file.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No, beyond the Pod size increase already noted.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

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

Not applicable, it should not require any new API or etcd calls.

###### What are other known failure modes?

- Different kubelets could be configured to use different cluster domains, which
  may confuse some workloads.
  - Detection: `kubectl get pods --all-namespaces --output=json | jq '.items[0] as $first | .items[] | select(.status.dns.clusterDomain != $first.status.dns.clusterDomain)'` 
    returns a non-empty result
  - Mitigations: Not much, beyond resolving the configuration issue and
    restarting the workloads. This KEP does not concern how cluster domains
    are managed (see the [non-goals](#non-goals)), nor does it declare this
    to be an *invalid* state, just one that may be surprising.
  - Diagnostics: Out of scope.
  - Testing: Out of scope.
- An existing kubelet may be reconfigured with a new cluster domain, leaving
  preexisting Pods with the old cluster domain configuration.
  - Detection: `kubectl get pods --all-namespaces --output=json | jq '(.items | map({key: .spec.nodeName, value: .status.dns.clusterDomain} | select(.key != null)) | from_entries) as $nodes | .items[] | select(.status.dns.clusterDomain != $nodes[.spec.nodeName]?)'` 
    returns a non-empty result
  - Mitigations: Restart the old workloads by deleting their Pods. Again, this
    state is not declared *invalid* by this KEP.
  - Diagnostics: Out of scope.
  - Testing: Out of scope.

###### What steps should be taken if SLOs are not being met to determine the problem?

Not applicable, as far as I can tell.

## Implementation History

- 2024-10-21: Discussion started in #sig-network: https://kubernetes.slack.com/archives/C09QYUH5W/p1729521715336479
- 2024-10-23: Summary of the current state of the art written by @lfrancke:  https://docs.google.com/document/d/11KO8UkLB8mg-fmUjzOYJNez_3NAsZ83gyC2hArYUCEU/edit?tab=t.gldt32oscsiw
- 2024-11-21: Initial KEP draft introduced
- 2025-06-09: KEP retargeted at introducing a status API, rather than focusing on the downward API

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

Trying to resolve the cluster domain from a different Pod could have confusing 
results, if running on a different kubelet that is configured to use a different
cluster domain (and its configured DNS server doesn't resolve the target pod's
cluster domain).

## Alternatives

### ConfigMap a la k3s

The ConfigMap written by k3s[^prior-art-k3s] could be blessed, requiring that 
all other distributions also provide it. However, this would require additional
migration effort from each distribution.

Additionally, this would be problematic to query for: users would have to query
it manually using the Kubernetes API (since ConfigMaps cannot be mounted across
Namespaces), and users would require RBAC permission to query wherever it is stored.

### Dedicated API Resource

This roughly shares the arguments for/against as [the ConfigMap alternative](#configmap-a-la-k3s),
although it would allow more precise RBAC policy targeting.

### kubelet `/configz`

The kubelet exposes a `/configz` endpoint which can be used to query its internal configuration.
This currently contains the cluster domain name.

However, this is a diagnostic utility, not a stable, documented, and versioned 
API. Encouraging users to rely on it also ossifies the idea that it is a part 
of the kubelet's static configuration.

### Parsing `resolv.conf`

The kubelet adds the cluster domain to the containers' `/etc/resolv.conf` file.
This could be parsed by users in order to guess the domain name.

However, this is an implementation detail that could be replaced by other 
mechanisms in the future, or disabled entirely.

It also requires clients to guess which domain is the correct one, which could
have false positives.

### Exposing it directly over the Downward API

A new type of Downward API could be added (alongside `fieldRef` and
`resourceFieldRef`), which allows "direct" access to properties provided by the
kubelet.

This would avoid increasing the size of Pods that don't use the feature.
However, it would also be less discoverable, and be inconsistent with the rest
of the platform.
