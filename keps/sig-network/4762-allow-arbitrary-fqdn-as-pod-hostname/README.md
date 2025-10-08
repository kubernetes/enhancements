# KEP-4762: Allows setting arbitrary FQDN as the pod's hostname
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
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
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This proposal allows users to set arbitrary Fully Qualified Domain Name (FQDN) as the hostname of a pod, introduces a new field `hostnameOverride` for the podSpec, which, if set, once the API is GA will always be respected by the Kubelet (otherwise it will fall back to legacy behavior), and no longer cares about the `hostname` as well as the `subdomain` values.

## Motivation

This feature will allow some traditional applications to join kubernetes in a more friendly way. Some older services may use hostname to determine permissions or service operations. When migrating services to k8s, the migration path will become confusing due to the hostname restrictions of the pod itself, because when we try to add a Fully Qualified Domain Name (FQDN) hostname to the pod, it will inevitably always carry the `cluster-suffix`, which will never be possible for services that expect to use DNS to match the hostname.

### Goals

* Allow users to set any arbitrary FQDN as pod hostname.
* Write the FQDN set by the user to `/etc/hosts` in the pod.

### Non-Goals

* Add DNS records for the FQDN set by the user.

## Proposal

We add a new field called `hostnameOverride` to `podSpec`, of type string. When the value of the `hostnameOverride` field is not an empty string, it always overrides the values of the `setHostnameAsFQDN`, `subdomain`, and `hostname` fields in `podSpec` to become the hostname of the pod, and only allow the value of setHostnameAsFQDN to be nil.

### User Stories (Optional)

#### Story 1

As a Kubernetes administrator, I want the Kerberos replication daemon (kpropd) to accurately handle hostname resolution for authentication.

In a Kubernetes environment, kpropd on the receiving end uses the hostname to determine the appropriate service credential for authentication purposes (e.g., foo-0.default.pod.cluster-local). However, on the sending side, kpropd uses the hostname it is connecting to (e.g., kdc1.example.com) to generate the cryptographic secret for secure communication. These hostnames must match to ensure that the cryptographic process can generate consistent data on both ends. Any discrepancy between these hostnames can result in authentication failure due to mismatched cryptographic data.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

The Linux kernel limits the hostname field to 64 bytes (see [sethostname(2)](http://man7.org/linux/man-pages/man2/sethostname.2.html)). If a hostname reaches this 64 byte kernel hostname limit, Kubernetes will fail to create the Pod Sandbox, causing the Pod to remain indefinitely in the `ContainerCreating` state.

To mitigate this issue, we will implement a validation during resource creation to check whether the value of hostnameOverride exceeds 64 bytes. Creation requests exceeding this limit will be denied.

After enabling this feature, if users utilize it to create a group of Pods via Deployment or StatefulSet, multiple Pods with identical names may concentrate on a single node. This could lead to unintended consequences, though we haven't identified specific potential issues at this time.

## Design Details

We are introducing a new feature gate called `HostnameOverride`. When this feature gate is enabled, users can add the `hostnameOverride` field in the podSpec.

The `hostnameOverride` field has a length limitation of 64 characters and must adhere to the DNS subdomain names standard defined in [RFC 1123](https://datatracker.ietf.org/doc/html/rfc1123).

Additionally, in the `generatePodSandboxConfig` method of kubelet, the pod's hostname will always be overridden with the value of `hostnameOverride`, and it will be written in the pod's `/etc/hosts`.

For Windows containers, we only set the container's hostname and do not create an `/etc/hosts` file for it (as we have previously made it clear that we do not create an `/etc/hosts` file for Windows containers).

If both `setHostnameAsFQDN` and `hostnameOverride` fields are set, or if both `hostNetwork` and `hostnameOverride` fields are set, we will reject the creation of the resource and return an error indicating that these fields are mutually exclusive with the `hostnameOverride` field.

Based on the above design, after the KEP is implemented, we can achieve the following results.

|  # | `.hostname` | `.subdomain` | `.setHostnameAsFQDN` | `.hostnameOverride` | `.hostNetwork` | `$(hostname)`                   | `$(hostname -f)`                | DNS (assuming service exists)   |
| -- | ----------- | ------------ | -------------------- | ------------------- | -------------- | ------------------------------- | ------------------------------- | ------------------------------- |
|  0 |             |              |                      |                     |                | `<pod-name>`                    | `<pod-name>`                    |                                 |
|  1 | `aa`        |              |                      |                     |                | `aa`                            | `aa`                            |                                 |
|  2 |             | `bb`         |                      |                     |                | `<pod-name>`                    | `<pod-name>.bb.<ns>.svc.<zone>` | `<pod-name>.bb.<ns>.svc.<zone>` |
|  3 | `aa`        | `bb`         |                      |                     |                | `aa`                            | `aa.bb.<ns>.svc.<zone>`         | `aa.bb.<ns>.svc.<zone>`         |
|  4 |             |              | true                 |                     |                | `<pod-name>`                    | `<pod-name>`                    |                                 |
|  5 | `aa`        |              | true                 |                     |                | `aa`                            | `aa`                            |                                 |
|  6 |             | `bb`         | true                 |                     |                | `<pod-name>.bb.<ns>.svc.<zone>` | `<pod-name>.bb.<ns>.svc.<zone>` | `<pod-name>.bb.<ns>.svc.<zone>` |
|  7 | `aa`        | `bb`         | true                 |                     |                | `aa.bb.<ns>.svc.<zone>`         | `aa.bb.<ns>.svc.<zone>`         | `aa.bb.<ns>.svc.<zone>`         |
|  8 |             |              |                      | `xx.yy.zz`          |                | `xx.yy.zz`                      | `xx.yy.zz`                      |                                 |
|  9 | `aa`        |              |                      | `xx.yy.zz`          |                | `xx.yy.zz`                      | `xx.yy.zz`                      |                                 |
| 10 |             | `bb`         |                      | `xx.yy.zz`          |                | `xx.yy.zz`                      | `xx.yy.zz`                      | `<pod-name>.bb.<ns>.svc.<zone>` |
| 11 | `aa`        | `bb`         |                      | `xx.yy.zz`          |                | `xx.yy.zz`                      | `xx.yy.zz`                      | `aa.bb.<ns>.svc.<zone>`         |
| 12 |             |              | true                 | `xx.yy.zz`          |                | INVALID                         | INVALID                         | INVALID                         |
| 13 | `aa`        |              | true                 | `xx.yy.zz`          |                | INVALID                         | INVALID                         | INVALID                         |
| 14 |             | `bb`         | true                 | `xx.yy.zz`          |                | INVALID                         | INVALID                         | INVALID                         |
| 15 | `aa`        | `bb`         | true                 | `xx.yy.zz`          |                | INVALID                         | INVALID                         | INVALID                         |
| 16 |             |              |                      |                     | true           | `<same-as-node>`                | `<same-as-node>`                |                                 |
| 17 | `aa`        |              |                      |                     | true           | `<same-as-node>`                | `<same-as-node>`                |                                 |
| 18 |             | `bb`         |                      |                     | true           | `<same-as-node>`                | `<same-as-node>                 | `<pod-name>.bb.<ns>.svc.<zone>` |
| 19 | `aa`        | `bb`         |                      |                     | true           | `>same-as-node>`                | `>same-as-node>`                | `aa.bb.<ns>.svc.<zone>`         |
| 20 |             |              | true                 |                     | true           | `<same-as-node>`                | `<same-as-node>`                |                                 |
| 21 | `aa`        |              | true                 |                     | true           | `<same-as-node>`                | `<same-as-node>`                |                                 |
| 22 |             | `bb`         | true                 |                     | true           | `<same-as-node>`                | `<same-as-node>`                | `<pod-name>.bb.<ns>.svc.<zone>` |
| 23 | `aa`        | `bb`         | true                 |                     | true           | `<same-as-node>`                | `<same-as-node>`                | `aa.bb.<ns>.svc.<zone>`         |
| 24 |             |              |                      | `xx.yy.zz`          | true           | INVALID                         | INVALID                         | INVALID                         |
| 25 | `aa`        |              |                      | `xx.yy.zz`          | true           | INVALID                         | INVALID                         | INVALID                         |
| 26 |             | `bb`         |                      | `xx.yy.zz`          | true           | INVALID                         | INVALID                         | INVALID                         |
| 27 | `aa`        | `bb`         |                      | `xx.yy.zz`          | true           | INVALID                         | INVALID                         | INVALID                         |
| 28 |             |              | true                 | `xx.yy.zz`          | true           | INVALID                         | INVALID                         | INVALID                         |
| 29 | `aa`        |              | true                 | `xx.yy.zz`          | true           | INVALID                         | INVALID                         | INVALID                         |
| 30 |             | `bb`         | true                 | `xx.yy.zz`          | true           | INVALID                         | INVALID                         | INVALID                         |
| 31 | `aa`        | `bb`         | true                 | `xx.yy.zz`          | true           | INVALID                         | INVALID                         | INVALID                         |                    |



As shown in the table, setting `hostnameOverride` will only change the hostname inside the pod and will not modify the DNS records in Kubernetes.

### Test Plan

##### Prerequisite testing updates

##### Unit tests

- Add kubelet unit tests to verify that container hostnames are correctly generated:  `k8s.io/kubernetes/pkg/kubelet/kuberuntime`: `2025-06-06` - `69.0%`
- Add API validation unit tests to ensure all field combinations yield correct results: `k8s.io/kubernetes/pkg/apis/core/validation` : `2025-06-06` - `84.7%`

##### Integration tests

- N/A

##### e2e tests

- Add a conformance test to `test/e2e` that verifies our implementation conforms to the expectation defined in the table within the #Design Details section.

### Graduation Criteria

#### Alpha

- Use the `HostnameOverride` feature gate to implement this feature.
- Initial e2e tests completed and enabled.
  - The link to the added e2e test: https://github.com/kubernetes/kubernetes/blob/master/test/e2e/common/node/pod_hostnameoverride.go
- Add documentation for feature gates.
- Add a detailed table to the docs illustrating the mappings between pod hostnames and DNS records under different configurations.

#### Beta

- Make feature gate to be enabled by default.
- Update the feature gate documentation.

#### GA

- No issues reported during two releases.

### Upgrade / Downgrade Strategy

API server should be upgraded before Kubelets. Kubelets should be downgraded before the API server.

### Version Skew Strategy

The core implementation resides in kubelet.

Older kubelet versions will ignore the pod's hostnameOverride field:
• Newly created Pods will retain previous behavior

Older apiserver versions will similarly ignore the hostnameOverride field:
• The apiserver doesn't populate the hostnameOverride value, so newer kubelet versions will maintain legacy behavior

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: HostnameOverride
  - Components depending on the feature gate: kubelet, kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Using the feature gate is the only way to enable/disable this feature.

###### What happens if we reenable the feature if it was previously rolled back?

There will be no impact on running Pods in the cluster. This change solely affects newly created Pods. Once enabled, you can set pod hostnames by configuring the `podSpec.hostnameOverride` field.


###### Are there any tests for feature enablement/disablement?

We have added unit tests for enabling and disabling the feature gate in: `pkg/kubelet/kubelet_pods_test.go#TestGeneratePodHostNameAndDomain`

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

No known failure modes.

###### What specific metrics should inform a rollback?

The `kubelet_started_pods_total` metrics helps determine whether enabling/disabling this feature causes abnormal pod restarts in the cluster.

`kubelet_started_pods_errors_total` metrics tracks if feature toggling results in pod startup failures.

`kubelet_restarted_pods_total` metrics monitors whether enabling/disabling triggers restarts of Static Pods.

`run_podsandbox_errors_total` metric helps detect if enabling the feature gate and using this functionality would cause sandbox container creation failures.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

I use `FEATURE_GATES=HostnameOverride=true ./hack/local-up-cluster.sh` to create a new cluster.

Check the cluster version:
```
$GOPATH/src/k8s.io/kubernetes/_output/bin/kubectl version
Client Version: v1.35.0
Kustomize Version: v5.7.1
Server Version: v1.35.0
```
Run a pod that uses HostnameOverride:
```
cat <<EOF | $GOPATH/src/k8s.io/kubernetes/_output/bin/kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: data-writer-pod
spec:
  hostUsers: false
  hostNetwork: true
  priorityClassName: system-node-critical
  containers:
  - name: writer-container
    image: busybox
    command: ["/bin/sh", "-c", "sleep 3600"]
EOF
```
Confirm the pod is running normally and the HostnameOverride feature is working correctly:
```
➜  opt $GOPATH/src/k8s.io/kubernetes/_output/bin/kubectl get pods
NAME       READY   STATUS    RESTARTS   AGE
test-pod   1/1     Running   0          5s
➜  opt $GOPATH/src/k8s.io/kubernetes/_output/bin/kubectl exec test-pod -- hostname
test-hostname
```

Checkout the `release-1.34` branch, add a tag using `git tag v1.34.0`, rebuild the `kubelet` and `kube-apiserver` binaries, and run kubelet and kube-apiserver using the same local command.

Check the cluster version:
```
$GOPATH/src/k8s.io/kubernetes/_output/bin/kubectl version
Client Version: v1.35.0
Kustomize Version: v5.7.1
Server Version: v1.34.0
```
Confirm the pod is still running and the HostnameOverride feature is still working correctly:

```
➜  opt $GOPATH/src/k8s.io/kubernetes/_output/bin/kubectl get pods
NAME       READY   STATUS    RESTARTS   AGE
test-pod   1/1     Running   0          3m56s
➜  opt $GOPATH/src/k8s.io/kubernetes/_output/bin/kubectl exec test-pod -- hostname
test-hostname
```
Delete the pod and recreate it:
```
opt $GOPATH/src/k8s.io/kubernetes/_output/bin/kubectl delete test-pod
pod "test-pod" deleted from default namespace

cat <<EOF | $GOPATH/src/k8s.io/kubernetes/_output/bin/kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: data-writer-pod
spec:
  hostUsers: false
  hostNetwork: true
  priorityClassName: system-node-critical
  containers:
  - name: writer-container
    image: busybox
    command: ["/bin/sh", "-c", "sleep 3600"]
EOF
```
Confirm the pod is running and the HostnameOverride feature is still working correctly:
```
➜  opt $GOPATH/src/k8s.io/kubernetes/_output/bin/kubectl get pods
NAME       READY   STATUS    RESTARTS   AGE
test-pod   1/1     Running   0          17s
➜  opt $GOPATH/src/k8s.io/kubernetes/_output/bin/kubectl exec test-pod -- hostname
test-hostname
```
Checkout to the master branch, add a tag using `git tag v1.35.0`, rebuild the `kubelet` and `kube-apiserver` binaries, and run `kubelet` and `kube-apiserver` using the same local command.

Check the cluster version:
```
$GOPATH/src/k8s.io/kubernetes/_output/bin/kubectl version
Client Version: v1.35.0
Kustomize Version: v5.7.1
Server Version: v1.35.0
```
Confirm the pod is still running and the HostnameOverride feature is still working correctly:
```
$GOPATH/src/k8s.io/kubernetes/_output/bin/kubectl get pods
NAME       READY   STATUS    RESTARTS   AGE
test-pod   1/1     Running   0          98s
➜  opt $GOPATH/src/k8s.io/kubernetes/_output/bin/kubectl exec test-pod -- hostname
test-hostname
```

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Users can check which workloads are utilizing this feature with the following command:
```
kubectl get pods -A -o json | jq -r '.items[] | select(.spec.hostnameOverride != null) | "\(.metadata.namespace) \(.metadata.name) \(.spec.hostnameOverride)"'
```

###### How can someone using this feature know that it is working for their instance?

Users can use the following command to identify which workloads are using this feature and verify whether it is functioning as expected.
```
kubectl get pods -A -o json | jq -r '.items[] | select(.spec.hostnameOverride != null) | "\(.metadata.namespace) \(.metadata.name) \(.spec.hostnameOverride)"' | while IFS=' ' read -r ns pod ho; do actual=$(kubectl exec -n "$ns" "$pod" -- hostname 2>/dev/null); [ "$actual" = "$ho" ] && echo "$ns $pod $actual $ho"; done
```

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

If the `kubelet_started_pods_errors_total` metric in a cluster remains consistently at 0, then after introducing this feature, the value of `kubelet_started_pods_errors_total` should similarly remain at 0.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?


- [x] Metrics
  - Metric name: `run_podsandbox_errors_total`, `kubelet_started_pods_total`, `kubelet_started_pods_errors_total`, `kubelet_restarted_pods_total`
  - [Optional] Aggregation method: A sharp increase in these metric values would indicate abnormal pod restarts or creation errors in the cluster caused by toggling the feature gate.
  - Components exposing the metric: Kubelet
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Implementing this feature requires adding a new field to the Pod object, which will increase its size. However, we'll limit the new field's length to 64 bytes.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No impact to the running workloads

###### What are other known failure modes?

No known failure modes.

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2024-07-18: Initial draft KEP
- 2025-08-13: Align KEPs with implemented PRs and documentation.
- 2025-10-10: Promote to beta stage

## Drawbacks

This is not a standard Kubernetes use case; it is undoubtedly in conflict with the current pod's potential DNS records, and using it will bring more confusion to users. Moreover, we are not sure how much it can help traditional services that can benefit from being migrated to Kubernetes.

## Alternatives

* Configure hostnameOverride via kube-apiserver:
  * If the `hostnameOverride` field is set, Kubelet will always respect this field (otherwise it will revert to the old behavior). In the default or REST logic, we can see if `hostnameOverride` is not set, then we check the `hostname`, `setHostnameAsFQDN`, and the `cluster-suffix`, and write the result into `hostnameOverride`. If the user sets it themselves, we will retain it and treat it as an override, this can ultimately simplify `Kubelet` as it can remove legacy behavior, but it means teaching the `kube-apiserver` about the `cluster-suffix`, however, it is challenging to find an existing or grace way to pass the `kube-apiserver`’s configuration options in the REST or default logic.
* Migrate Legacy Projects:
  * Repair the traditional projects that cannot be migrated to Kubernetes, or find alternatives.
* Relax hostname Validation:
  * Do not add new fields, relax the validation of the `hostname` field in `podSpec` to allow it to accept strings in FQDN format, and when the `hostname` is set to FQDN, we will unconditionally ignore the `subdomain` and `setHostnameAsFQDN` fields, or to keep the current `hostname` and be able to override or omit the `default.svc.cluster.local` part. However, doing so will cause us to lose the DNS resolution records for the pod.
* Custom setHostnameAsFQDN:
  * Do not add new fields, allowing the value of `setHostnameAsFQDN` to be set to `Custom`, the pod's hostname can still meet our expectations. However, since `setHostnameAsFQDN` is currently a boolean type, modifying it would be disruptive to the existing API.
* Init Container Hostname
  * We can start an init container with privileged mode and run the command hostname mypod.fqdn.com within the init container to set the Pod's hostname to mypod.fqdn.com. This can achieve the same goal.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
