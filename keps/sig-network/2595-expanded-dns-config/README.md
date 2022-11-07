# KEP-2595: Expanded DNS Configuration

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
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
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Allow kubernetes to have expanded DNS(Domain Name System) configuration.

## Motivation

Kubernetes today limits DNS configuration according to [the obsolete
criteria](https://access.redhat.com/solutions/58028). As recent DNS resolvers
allow an arbitrary number of search paths, a new feature gate
`ExpandedDNSConfig` will be introduced. With this feature, kubernetes allows
more DNS search paths and longer list of DNS search paths to keep up with recent
DNS resolvers.

Confirmed that expanded DNS configuration is supported by
- `glibc 2.17-323`
- `glibc 2.28`
- `musl libc 1.22`
- `pure Go 1.10 resolver`
- `pure Go 1.16 resolver`

### Goals

- Make `kube-apiserver` allow expanded DNS configuration when validating Pod's
or PodTemplate's `DNSConfig`
- Make `kubelet` allow expanded DNS configuration when validating
[`resolvConf`](https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/#kubelet-config-k8s-io-v1beta1-KubeletConfiguration)
- Make `kubelet` allow expanded DNS configuration when validating actual DNS
resolver configuration composed of `cluster domain suffixes`(e.g.
    default.svc.cluster.local, svc.cluster.local, cluster.local), kubelet's
`resolvConf` and Pod's `DNSConfig`

### Non-Goals

- Remove limitation on DNS search paths completely
- Let cluster administrators limit the number of search paths or the length of
DNS search path list to an arbitrary number

## Proposal

- Expand `MaxDNSSearchPaths` to 32
- Expand `MaxDNSSearchListChars` to 2048

### User Stories (Optional)

#### Story 1

#### Story 2

### Notes/Constraints/Caveats (Optional)

Some container runtimes of older versions have their own restrictions on the
number of DNS search paths. For the container runtimes which are older than
- containerd v1.5.6
- CRI-O v1.22

, pods with expanded DNS configuration may get stuck in the pending state. (see
    [kubernetes#104352](https://github.com/kubernetes/kubernetes/issues/104352))

This enhancement relaxes the validation of `Pod` and `PodTemplate`. Once the
feature is activated, it must be carefully disabled. Otherwise, the objects left
over from the previous version which have the expanded DNS configuration can be
problematic.

### Risks and Mitigations

There may be some environments(DNS resolver or others) that break without
current limitations. At this point, it is fair to open a bug, so they can fix it.

## Design Details

- Declare and define `MaxDNSSearchPathsExpanded` to `32` and
`MaxDNSSearchListCharsExpanded` to `2048`
- Add the feature gate `ExpandedDNSConfig` (see [Feature Enablement and
Rollback](#feature-enablement-and-rollback))
- If the feature gate `ExpandedDNSConfig` is enabled, replace
`MaxDNSSearchPaths` with `MaxDNSSearchPathsExpanded` and replace
`MaxDNSSearchListChars` with `MaxDNSSearchListCharsExpanded` to allow expanded
DNS configuration

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

N/A

##### Unit tests

Verified that the API server accepts the pod or podTemplate with the expanded
DNS config and the kubelet accepts the resolv.conf or pod with the expanded DNS
config.

##### Integration tests

No integration tests are planned.

##### e2e tests

Will add an e2e test to ensure that the pod with the expanded DNS config can be
created and run successfully.

### Graduation Criteria

#### Alpha

- Implement the feature
- Add appropriate unit tests

#### Beta

- Address feedback from alpha
- Add e2e tests
- All major container runtimes supported versions allows this feature

#### GA

- Address feedback from beta
- Sufficient number of users using the feature
- Close on any remaining open issues & bugs

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

In clusters with older kubelets, old kubelets with `resolvConf` configured to
exceed bounds throw warnings but do not fail. Eventually, old kubelets truncate
the overage and apply the actual DNS resolver configuration.

In clusters with a replicated control plane, all kube-apiservers should enable
or disable the expanded DNS configuration feature.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

- **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `ExpandedDNSConfig`
    - Components depending on the feature gate:
      - `kubelet`
      - `kube-apiserver`
    - This feature is not compatible with some older container runtimes (see
        [Notes/Constraints/Caveats](#notesconstraintscaveats-optional))
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

- **Does enabling the feature change any default behavior?**

Enabling this feature allows kubernetes to have objects(Pod or PodTemplate) with
the expanded DNS configuration.

- **Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?**

Yes, the feature can be disabled by disabling the feature gate.

Before disabling the feature gate, is is recommended to remove objects
containing podsTemplate with the expanded DNS config as newly created pods will
be rejected by the apiserver.

```sh
$ cat << \EOF > get-expanded-dns-config-objects.tpl
{{- range $_, $objects := .items}}
  {{- with $searches := .spec.template.spec.dnsConfig}}
    {{- $length := len .searches }}
    {{- if gt $length 6 }}
      {{- $objects.metadata.name }}
      {{- printf " " }}
      {{- continue }}
    {{- end}}

    {{- $searchStr := "" }}
    {{- range $search := .searches}}
      {{- $searchStr = printf "%s %s" $searchStr $search }}
    {{- end}}
    {{- $searchLen := len $searchStr }}
    {{- if gt $searchLen 256}}
      {{- $objects.metadata.name }}
      {{- printf " " }}
      {{- continue }}
    {{- end }}
  {{- end}}
{{- end}}
EOF

# get deployments having the expanded DNS configuration
$ kubectl get deployments.apps --all-namespaces -o go-template-file=get-expanded-dns-config-objects.tpl
```

Once the feature is disabled, kube-apiserver will reject the newly requested pod
having expanded DNS configuration and kubelet will create a resolver
configuration excluding the overage.

If there is a problem with an object that already has expanded DNS
configuration, the object should be removed manually.

- **What happens if we reenable the feature if it was previously rolled back?**

New objects with expanded DNS configuration will be accepted by the apiserver
and new Pods with expanded configuration will be created by the kubelet.

- **Are there any tests for feature enablement/disablement?**

Yes.

We verified in unit tests that existing pods work with the feature enabled and
already created pods with the expanded DNS config work fine with the feature
disabled.

When this feature is disabled, objects containing podTemplate with the expanded
DNS config cannot create new pods until that podTemplate is fixed to have the
non-expanded DNS config.

### Rollout, Upgrade and Rollback Planning

- **How can a rollout fail? Can it impact already running workloads?**

If a kubelet starts with invalid `resolvConf`, new workloads will fail DNS
lookups.

- **What specific metrics should inform a rollback?**

If new workloads start to fail DNS lookups due to a corrupted resolv.conf, or
due to older resolver libraries, that would be an indication to rollback the
enablement.

- **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

Yes

- **Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?**

No

### Monitoring Requirements

- **How can an operator determine if the feature is in use by workloads?**

There is no metric to indicate the enablement. The operator has to check if
there are objects or DNS resolver configuration files with expanded
configuration to determine if the feature is in use.

- **What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [x] Other (treat as last resort)
    - Success of DNS lookups

- **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

DNS lookups should not fail as before the feature was enabled.

- **Are there any missing metrics that would be useful to have to improve observability of this feature?**

TBD

### Dependencies

- **Does this feature depend on any specific services running in the cluster?**

This feature requires container runtime support. See
[Notes/Constraints/Caveats](#notesconstraintscaveats-optional).

### Scalability

- **Will enabling / using this feature result in any new API calls?**

No

- **Will enabling / using this feature result in introducing new API types?**

No

- **Will enabling / using this feature result in any new calls to the cloud provider?**

No

- **Will enabling / using this feature result in increasing size or count of the existing API objects?**

The sum of the lengths of `PodSpec.DNSConfig.Searches` can be increased to 2048.

- **Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?**

The DNS lookup time can be increased, but it will be negligible.

- **Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?**

No

### Troubleshooting

- **How does this feature react if the API server and/or etcd is unavailable?**

N/A

- **What are other known failure modes?**

N/A

- **What steps should be taken if SLOs are not being met to determine the problem?**

If DNS lookups fail, you can check error messages. And then, validate the
kubelet's `resolvConf` if it is corrupted or use newer DNS resolver libraries if
they are too old.

## Implementation History

- 2021-03-26: [Initial discussion at
#100583](https://github.com/kubernetes/kubernetes/pull/100583)
- 2021-05-11: [Initial KEP
approved](https://github.com/kubernetes/enhancements/pull/2596)
- 2021-05-27: [Initial alpha implementations
merged](https://github.com/kubernetes/kubernetes/pull/100651)
- 2021-06-05: [Initial docs
merged](https://github.com/kubernetes/website/pull/28096)
- 2022-01-12: [Docs updated to add requirements for the
feature](https://github.com/kubernetes/website/pull/31305)

## Drawbacks

## Alternatives

- Remove the limitation of DNS search paths completely
- Make `MaxDNSSearchPaths` and `MaxDNSSearchListChars` configurable

## Infrastructure Needed (Optional)
