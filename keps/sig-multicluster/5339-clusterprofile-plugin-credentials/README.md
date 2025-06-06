<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

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
- [ ] **Merge early and iterate.**
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
# KEP-5339: Plugin for Credentials in ClusterProfile

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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

To manage an Inventory of Clusters, a platform admin can rely on having the cluster manager
output [ClusterProfile CRs](https://github.com/kubernetes-sigs/cluster-inventory-api/blob/main/apis/v1alpha1/clusterprofile_types.go) that point to the clusters. Those CRs are key for multicluster controllers
that want to operate on the clusters. However, there isn't a single way to obtain credentials
to reach those clusters. This KEP provides a standardized way to obtain credentials for Clusters
when using ClusterProfile and makes it pluggable to allow the diverse ecosystem to support the
multitude of ways to obtain credentials. It also reuses part of the Kubeconfig external provider
semantics to make implementation easier.

## Motivation

ClusterInventory is unfinished without an ability to use the clusters and controller
writers have been very explicit that credentials are needed. Previous attempts at writing credentials have
failed and we believe that a plugin model, also reusing known flows, will help solve the "credentials" need
for ClusterProfiles.

See [introduction slides](https://docs.google.com/presentation/d/1v5-J-kFJ3TSpKqSraHcYkCz2NG7cNnYpq0ISF85wNMU/edit)

### Goals

* Provide a library for controllers to obtain credentials for a cluster represented by a ClusterProfile
* Allow cluster managers to provide a method to obtain credentials that doesn't require to be embedded into the controller code
  and recompiling.
* Be a secure mechanism for credential obtention and storage.

### Non-Goals

* Define the mechanism for shipping plugins to be used by the controllers and their delivery in the controller image/pod.
* Design plugin or a library for plugins
* Mandate Federated workload identity / OIDC frameworks (though they are recommended)

## Proposal

The proposed approach to obtaining credentials is to leverage plugins for retrieving the credentials from an issuer recognized by the target cluster. The controller using ClusterProfile
would use a library running a local executable which would retrieve the credentials for the current controller and a given clusterprofile.
It is expected that plugins would leverage elements local to the controller to help assert the identity of the controller (environment variables, config files, KSA, the local IP, etc...)
to retrieve credentials that are valid on the target cluster. Plugins would be exec'ed by the controller so that they don't need be built-in the binary, allowing flexibility into writing their
own credential plugins and still leveraging multicluster controllers written by the community. In addition, we propose to reuse the exec approach and protocol used for external credentials in
Kubeconfig (but not the configuration part of kubeconfig). Finally, in order to retrieve the endpoint for the cluster, we standardize the property names that are used in ClusterProfile.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

Because of its interaction with authentication and credentials, particular attention in this design must be paid to security:

* credentials leak: ClusterProfile, Controller configuration and Plugin configuration should never contain sensitive information
* Plugin poisoning: supporting credentials provider plugins in a controller relies on trusting the plugin code itself. Particular attention must
be provided by the user deploying a controller to make sure the plugin that they install are from trusted source as they will have access to the controller's identity.

Another risk is around AuthZ. This design doesn't cover the distribution of RBAC to multiple clusters and identifying what principal a controller can be identified as. This
setup is currently left to the responsibility of the platform admin setting up the different clusters and controllers.

## Design Details

The proposal's implementation would be done via a Library in https://github.com/kubernetes-sigs/cluster-inventory-api.
The library would be in golang. The library is provided as the community shared implementation for golang and
it is possible that other implementations would be created, and would work with the same plugin mechanism defined here,
allowing for reuse of the external providers that cluster managers write.

The expected prototype for a controller is expected to be the following:

`func (c *ClusterProfileExternalProviders) GetConfig(cp *ClusterProfile) (*rest.Config, error)`

The library implementation flow is expected to be as follows:

1. Build the endpoint details of the cluster by reading properties of the ClusterProfile
2. Call the CredentialsExternalProviders, following the same flow defined in [KEP 541](https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/541-external-credential-providers/README.md)
  (giving the ability to reuse the code in [client-go's exec package](https://github.com/kubernetes/client-go/blob/master/plugin/pkg/client/auth/exec/exec.go#L159))
3. Build the rest.Config and return it to the caller

#### External credentials Provider plugin mechanism

In order to call the plugin, the library execs the plugin defined in the configuration. It passes the Cluster information that was obtained from the ClusterProfile.
The library then calls the plugin following the protocol defined in [KEP 541](https://github.com/kubernetes/enhancements/blob/master/keps/sig-auth/541-external-credential-providers/README.md).
The library provided in https://github.com/kubernetes-sigs/cluster-inventory-api can leverage the [original code that is kept in client-go](https://github.com/kubernetes/client-go/blob/master/plugin/pkg/client/auth/exec/exec.go#L159).

### Standardizing the Provider definition

In order to populate the Cluster object that the exec provider requires, we standardize a new field in ClusterProfile called `credentials` that is stored in the Spec of the ClusterProfile.
All the data from this structure is specific to the clusterProfile and does not contain any Controller-specific information. It must be usable by different
controller, applications or consumers without requiring changes. It also cannot contain any data considered a secret; and we consider that reachability information
is not sensitive.

The definition is as follows:
```
type Credentials struct {
  credentialProviders map[string]CredentialsConfig // mapping of credentials types to their config. In some cases the cluster may recognize different identity types and they may have different endpoints or TLS config.
}

// CredentialsTypes defines the type of credentials that are accepted by the cluster. For example, GCP credentials (tokens that are understood by GCP's IAM) are designated by the string `google`.
type CredentialsType string

// CredentialsConfig gives more details on data that is necessary to reach out the cluster for this kind of Credentials
type CredentialsConfig struct {
  Cluster *Cluster // Configuration to reach the cluster (endpoints, proxy, etc) // See following section for details.
}
```

#### Cluster Data

The Cluster structure for the exec defined in KEP 541, [implemented in k/k](https://github.com/kubernetes/kubernetes/blob/a34c07971b610eb33908a743eadb4c61beeecc50/staging/src/k8s.io/client-go/pkg/apis/clientauthentication/types.go#L73-L80) assumes the following:
```
type Cluster struct {
	// Server is the address of the kubernetes cluster (https://hostname:port).
	Server string
	// TLSServerName is passed to the server for SNI and is used in the client to
	// check server certificates against. If ServerName is empty, the hostname
	// used to contact the server is used.
	// +optional
	TLSServerName string
	// InsecureSkipTLSVerify skips the validity check for the server's certificate.
	// This will make your HTTPS connections insecure.
	// +optional
	InsecureSkipTLSVerify bool
	// CAData contains PEM-encoded certificate authority certificates.
	// If empty, system roots should be used.
	// +listType=atomic
	// +optional
	CertificateAuthorityData []byte
	// ProxyURL is the URL to the proxy to be used for all requests to this
	// cluster.
	// +optional
	ProxyURL string
	// DisableCompression allows client to opt-out of response compression for all requests to the server. This is useful
	// to speed up requests (specifically lists) when client-server network bandwidth is ample, by saving time on
	// compression (server-side) and decompression (client-side): https://github.com/kubernetes/kubernetes/issues/112296.
	// +optional
	DisableCompression bool
	// Config holds additional config data that is specific to the exec
	// plugin with regards to the cluster being authenticated to.
	//
	// This data is sourced from the clientcmd Cluster object's
	// extensions[client.authentication.k8s.io/exec] field:
	//
	// clusters:
	// - name: my-cluster
	//   cluster:
	//     ...
	//     extensions:
	//     - name: client.authentication.k8s.io/exec  # reserved extension name for per cluster exec config
	//       extension:
	//         audience: 06e3fbd18de8  # arbitrary config
	//
	// In some environments, the user config may be exactly the same across many clusters
	// (i.e. call this exec plugin) minus some details that are specific to each cluster
	// such as the audience.  This field allows the per cluster config to be directly
	// specified with the cluster info.  Using this field to store secret data is not
	// recommended as one of the prime benefits of exec plugins is that no secrets need
	// to be stored directly in the kubeconfig.
	// +optional
	Config runtime.Object
}
```


#### ClusterProfile Example

Example of a GKE ClusterProfile, which would map to a plugin providing credentials of type `google`:
```
apiVersion: multicluster.x-k8s.io/v1alpha1
kind: ClusterProfile
metadata:
 name: my-cluster-1
spec:
  displayName: my-cluster-1
  clusterManager:
    name: GKE-Fleet
  credentials:
    google:
      cluster:
        server: https://connectgateway.googleapis.com/v1/projects/123456789/locations/us-central1/gkeMemberships/my-cluster-1
status:
  version:
    kubernetes: 1.28.0
  properties:
   - name: clusterset.k8s.io
     value: some-clusterset
   - name: location
     value: us-central1
```


### Configuring plugins in the controller

Plugins are selected by a hardcoded string which represents the type of credentials that is expected by the cluster, for example, "google" for GKE Clusters.
Thish allows the controller to attach a different binary name or path for the the binary.

It is expected that the library will have a mapping from its supported type of credentials to the expected binary to call. The library would be fed via a repeated flag `clusterprofile-creds-provider` for ease of use.
The flag maps a credentials type to the associated binary and potential flags that should be passed. It cannot contain cluster-specific information (which is not known at that time).
```
./controller ... --clusterprofile-creds-provider "google='/usr/bin/gke-gcloud-auth-plugin --flag1 value1 --flag2 value2'"
```

Despite being a flag, we can express the equivalent structure for each Plugin:
```
type Provider struct {
  CredentialsType string
  ExecutablePath string
  args []string
}
```

Given the plugin is executed directly by the controller, it may expect to have access to the same environment as the controller itself, inclusive of envvars, filesystem and network.
It is expected that the identity of the plugin is the same as the controller itself.

### Plugin Examples

As an example, we provide pseudocode for plugins that could easily be implemented with the protocol. They are ultrasimplified
version of the code and structures to convey the idea and not be an implementation example.

#### Secret Reader plugin

This plugin assumes the controller is aware of the list of clusters ahead of time and has created secrets for them.
It simply reads the token from the secret mapped to the cluster.

```
func GetToken(clusterName string) string {
  // query secrets local to this controller (same cluster, same namespace)
  secret := secrets.Get(clusterName)
  return secret.Data.token
}
```

#### GKE with Workload Identity Federation

This plugin uses Workload Identity Federation to call the other clusters that are GKE clusters and therefore understanding google-issued credentials.

```
func GetToken() string {
  // This library calls looks at the standard envvar called GOOGLE_CREDENTIALS and if not found, calls the Metadata Server IP (169.254.169.254)
  creds := google.GetDefaultCredentials()
  return creds.Token()
}
```

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
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

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

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

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

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

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

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

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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

N/A - Out of tree library

### Rollout, Upgrade and Rollback Planning

N/A - Out of tree library

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->


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

N/A - just a library/protocol

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

N/A no use of etcd

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
