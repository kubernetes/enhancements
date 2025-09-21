# KEP-4427: Relaxed DNS search string validation

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
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
<!-- /toc -->

## Summary

Currently, Kubernetes validates search string in the `dnsConfig.searches` according to [RFC-1123](https://datatracker.ietf.org/doc/html/rfc1123)
which defines restrictions for hostnames. However, there are reasons why this validation is too strict for the use in `dnsConfig.searches`.

Firstly, while most DNS names identify hosts, there are record types (like SRV) that don't. For these, it's less clear
whether hostname restrictions apply, for example [RFC-1035 Section 2.3.1](https://datatracker.ietf.org/doc/html/rfc1035#section-2.3.1) points out
that it's better to stick with valid host names but also states that labels must meet the hostname requirements.

In practice, legacy workloads sometimes include an underscore (`_`) in DNS names and DNS servers will generally allow this.

Secondly, users may require setting `dnsConfig.searches` to a single dot character (`.`) should they wish to avoid unnessesary DNS lookup calls to internal Kubernetes domain names.

This KEP proposes relaxing the checks on DNS search strings only. Allowing these values in the `searches` field of `dnsConfig` allows pods to
resolve short names properly in cases where the search string contains an underscore or is a single dot character.

## Motivation

For workloads that resolve short DNS names where the full DNS name includes underscores,
it’s not possible to configure search strings using dnsConfig. For example, if a pod needs to look up an SRV record `_sip._tcp.abc_d.example.com`
using the short name of `_sip._tcp`, we would like to be able to add `abc_d.example.com` to the searches in the dnsConfig.

Here’s an example configuration which would support this case:

```
apiVersion: v1
kind: Pod
metadata:
  namespace: default
  name: dns-example
spec:
  containers:
    - name: test
      image: nginx
  dnsPolicy: "None"
  dnsConfig:
    nameservers:
      - 1.2.3.4
    searches:
      - abc_d.example.com
```

However, this returns an error:

```
The Pod "dns-example" is invalid: spec.dnsConfig.searches[0]: Invalid value: "abc_d.example.com": a DNS-1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example.com', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')
```

Allowing underscores in the search string allows integration with legacy workloads without allowing anyone to define
these names within Kubernetes. Since having underscores in a name creates other issues (such as inability to obtain a publicly trusted TLS certificate),
search strings seem like the only area where this is likely to occur.

Should a user require a DNS query to resolve to an external domain first (before the internal Kubernetes domain names) they would require adding a dot to the `dnsConfig.searches` list.

An example of this configuration could look like this:

```
apiVersion: v1
kind: Pod
metadata:
  namespace: default
  name: dns-example
spec:
  containers:
    - name: test
      image: nginx
  dnsPolicy: "None"
  dnsConfig:
    nameservers:
      - 1.2.3.4
    searches:
      - .
      - default.svc.cluster.local
      - svc.cluster.local
      - cluster.local
  ```

Applying the above Pod spec will result in the following error:

```
The Pod "dns-example" is invalid: spec.dnsConfig.searches[0]: Invalid value: "": a lowercase RFC 1123 subdomain must consist of lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character (e.g. 'example.com', regex used for validation is '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*')
```

### Goals

- Support workloads that need to resolve DNS short names where the full DNS name includes an underscore (`_`).
- Allow users to use a single dot character `.` as a search string

### Non-Goals

- Allowing support for any characters in the DNS search string

## Proposal

Introduce a RelaxedDNSSearchValidation feature gate which is disabled by default. When the feature gate is enabled,
a new DNS name validation function will be used, which keeps the existing validation logic but also allows an underscore (`_`) in any place
where a dash (`-`) would be allowed currently and allowing a single dot (`.`) character.

Since the relaxed check allows previously invalid values, care must be taken to support cluster downgrades safely. To accomplish this, the validation will distinguish between new resources and updates to existing resources:
- When the feature gate is disabled:
  - (a) New resources will use strict validation based on RFC-1123 (no change to current validation)
  - (b) Updates to existing resources will use relaxed validation if any search string in the existing list fails strict validation
- When the feature gate is enabled:
  - (c) New resources will use relaxed validation.
  - (d) Updates to existing resources will use relaxed validation.

This means that it is safe to downgrade a cluster with the feature gate enabled to a version where the feature gate is present (whether it’s enabled or disabled). It is not safe, in general, to downgrade from a cluster with the gate enabled to a version prior to the gate being introduced, since values may have been written to storage which will no longer pass validation. However, this scenario requires opting in through enabling the gate. In practice, the recommended approach would be to only enable to the gate after upgrading from a version with relaxed checking already present.

As long as the gate is disabled, there is no compatibility change, so cluster downgrades are not affected by the feature.

### Risks and Mitigations

The change is opt-in, since it requires configuring a search string with an underscore. So there is no risk beyond
the upgrade/downgrade risks which are addressed in the Proposal section.

## Design Details

See Proposal

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

Added validation will be covered by unit tests along with unit tests covering the behavior
when the gate is enabled or disabled.

##### Integration tests

We need to ensure that once there's an underscore, changes to the object will continue to pass validation even after the gate is off.
The test cases will cover behavior when the gate is on and when the gate is off, and will also cover transitioning from on to off
with a value that contains an underscore.

- Gate On
  - New value
    - Underscore and/or dot (expect validation to pass)
    - No Underscore and/or dot (expect validation to pass)
  - Existing value
    - Underscore and/or dot (expect validation to pass)
    - No Underscore and/or dot (expect validation to pass)
- Gate Off
  - New value
    - Underscore and/or dot (expect validation to fail)
    - No Underscore and/or dot (expect validation to pass)
  - Existing value
    - Underscore and/or dot (expect validation to pass)
    - No Underscore and/or dot (expect validation to pass)
- Ratcheting
   - Turn gate on, write search string with underscore and/or dot, turn gate off, change unrelated property on the object and verify that it passes validation, remove search value with the underscore and/or dot, verify that saving a search string with an underscore and/or dot is now prevented

In addition to the Pod itself, each integration test should be repeated with objects that contain a pod spec template:
-	Deployment
-	ReplicaSet
-	Job

##### e2e tests

- Add a test that verifies successful creation of a pod whose `dnsConfig.searches` contains an underscore and/or dot
- Add tests that verify successful creation of objects with a podTemplate whose `dnsConfig.searches`
  contains an underscore

### Graduation Criteria

#### Alpha
- [X] Feature implemented behind a gate
- [X] Initial e2e tests completed and enabled

#### Beta
- [X] No trouble reports from alpha release

#### GA
- [X] No trouble reports with the beta release, plus some anecdotal evidence of it being used successfully.

### Upgrade / Downgrade Strategy

See Proposal section.

### Version Skew Strategy

Kubelet only checks size limits but otherwise passes values through
[source](https://github.com/kubernetes/kubernetes/blob/f025a96d2f60984765731e01ad0de2c89e959b42/pkg/kubelet/network/dns/dns.go#L114).

Since the resolv.conf file is interpreted by the DNS resolver in the container image and not by the container runtime, the change
does not depend on the container runtime or its version.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: RelaxedDNSSearchValidation
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

Yes, there is a change to validation when the feature is enabled.
The change is managed through the racheting process described in the Proposal section.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?


Yes, the feature can be disabled. Proposal covers the validation logic in detail,
but briefly, existing values will be allowed with relaxed validation if the gate is disabled.

###### What happens if we reenable the feature if it was previously rolled back?

Then the relaxed validation will be allowed on new values in `dnsConfig.searches`.
Existing values (prior to the initial roll-back) will continue to pass validation regardless
of whether the gate is enabled or not.

###### Are there any tests for feature enablement/disablement?

Unit tests will cover cover the scenarios described in the Proposal section.

### Rollout, Upgrade and Rollback Planning


###### How can a rollout or rollback fail? Can it impact already running workloads?

Since this a change to validation behind a feature gate, rollout should pose no risk.

If a cluster needs to be rolled back for another reason, it's risky to enable this
feature unless the previous version also has the flag (whether it's enabled or disabled).

Since this feature allows previously invalid values in `dnsConfig.searches`, upgrading
from a version without the gate present (i.e. before introducing this feature) and then
enabling the gate is risky. In that scenario, if a search path is saved containing an
underscore and then the cluster is downgraded to a previous version with no knowledge
of the feature gate, then the downgrade may fail.

See the Proposal section for recommendation on avoding this scenario.

###### What specific metrics should inform a rollback?

N/A


###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Tested by hand.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Existence of an underscore the `dnsConfig.searches` array in any pod spec or pod spec template
would indicate the feature is in use.

###### How can someone using this feature know that it is working for their instance?

If they are able to save an object with a DNS string containing an underscore, then the feature is working.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A


###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. This is a change to API validation.

### Scalability

N/A

###### Will enabling / using this feature result in any new API calls?

No. This is a change to validation of existing API calls.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A. This is a change to validation within the API server.

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- [x] Alpha
  - [x] KEP (`k/enhancements`) update PR(s):
    - https://github.com/kubernetes/enhancements/pull/4428
    - https://github.com/kubernetes/enhancements/pull/4755
    - https://github.com/kubernetes/enhancements/pull/4884
  - [x] Code (`k/k`) update PR(s):
    - https://github.com/kubernetes/kubernetes/pull/127167
  - [ ] Docs (`k/website`) update PR(s):
- [x] Beta
  - [x] KEP (`k/enhancements`) update PR(s):
    - https://github.com/kubernetes/enhancements/pull/5045
    - https://github.com/kubernetes/enhancements/pull/5137
  - [x] Code (`k/k`) update PR(s):
    - https://github.com/kubernetes/kubernetes/pull/130128
  - [ ] Docs (`k/website`) update PR(s):
- [x] GA
  - [x] KEP (`k/enhancements`) update PR(s):
    - https://github.com/kubernetes/enhancements/pull/5352
  - [x] Code (`k/k`) update PR(s):
    - https://github.com/kubernetes/kubernetes/pull/132036
  - [ ] Docs (`k/website`) update PR(s):

## Drawbacks

Since it isn't possible to distinguish between record types a search string will be used for,
this also allows users to configure a pod that will use search string to from a hostname with
an underscore. The risk here is born by the user and the name is not defined within Kubernetes in
this case (instead it refers to a name configured outside the cluster).

## Alternatives

A workaround is to re-write the resolv.conf file from inside the pod. This typically requires running
the pod with higher privileges than the actual workload requires, however.
