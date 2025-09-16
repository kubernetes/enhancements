# KEP-4742: Node Topology Labels via Downward API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
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
  - [Expose all node labels via downward API](#expose-all-node-labels-via-downward-api)
  - [Helper controller](#helper-controller)
  - [Init container to retrieve Node topology](#init-container-to-retrieve-node-topology)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [X] e2e Tests for all Beta API Operations (endpoints)
  - [X] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [X] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [X] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Many workloads benefit significantly from being aware of the topology of the node they are scheduled on.
However, Kubernetes currently lacks a built-in mechanism to directly expose this node topology information to Pods.
This KEP proposes a built-in admission plugin to copy standard node topology labels to Pods.
Once copied, these labels can be consumed via the Downward API, just like any other Pod label.
This approach eliminates the need for workarounds, such as custom init containers with elevated privileges,
promoting a more secure and consistent solution.

## Motivation

Topology awareness is crucial for a growing number of applications and workloads on Kubernetes.
Knowing the Pods's location within the cluster's topology allows for significant performance
optimizations and improved resilience. End-user feedback has highlighted the following key use cases:
* High-bandwidth AI/ML workloads, especially those employing distributed training or inference,
  demonstrate a substantial need for topology awareness. The performance of these workloads is
  highly sensitive to communication latency between GPUs. By preferentially scheduling Pods
  requiring GPU-to-GPU communication within the same zone or rack, training and inference
  times can be dramatically reduced.  Importantly, while Kubernetes can facilitate this topology-aware
  placement, the actual orchestration of optimized GPU-to-GPU communication often relies on topology awareness
  within the machine learning framework itself (e.g., Ray, PyTorch Distributed. etc).  Kubernetes provides the
  necessary foundation for these frameworks to operate efficiently, but the frameworks are ultimately responsible
  for leveraging that foundation.
* CNI plugins can leverage topology information to establish more efficient network paths,
  reducing latency and increasing throughput. For instance, a CNI could prioritize connections
  within the same availability zone or rack.
* Distributed stateful applications, such as sharded databases, can use topology information to improve fault tolerance.
  By spreading replicas and traffic across different failure domains (e.g., zones, racks), these applications can achieve
  higher availability and resilience to failures in any one topology.

Today, applications obtain topology information by using an init container to query the Kubernetes API for the underlying node,
extracting the necessary data from the Node resource. As more workloads and use cases benefit from topology awareness,
we aim to simplify access to this information for Pods via the Downward API.

### Goals

* Values from Node labels `topology.kubernetes.io/zone`, `topology.kubernetes.io/region` and `kubernetes.io/hostname` are made
  available via downward API
* Additional node labels can be made available via downward API using admission webhooks that mutate `pods/binding`.

### Non-Goals

* Exposing non-standard node labels by default
* Enhnacements to topology-aware scheduling
* Changes to standard topology labels in Kubernetes
* Changes to downward API

## Proposal

Retreival of topology information will be achieved through the existing mechanism of passing down Pod labels using Downward API.
A new admission plugin, `PodTopologyLabels`, will be added to kube-apiserver. When enabled, it will mutate the `pods/binding` subresource,
adding topology labels that match those of the target Node. The Binding REST implementation will be updated to copy these labels
from the `pods/binding` subresource to the assigned Pod's labels.

Using the downward API to retrieve topology information will look similar to the following:
```
apiVersion: v1
kind: Pod
metadata:
  generateName: downwardapi-example-
spec:
  containers:
    - name: container
      image: registry.k8s.io/busybox
      command: ["sh", "-c"]
      args:
      - while true; do
          if [[ -e /etc/podinfo/zone ]]; then
            echo -en '\n\n'; cat /etc/podinfo/zone; fi;
          sleep 5;
        done;
      volumeMounts:
        - name: podinfo
          mountPath: /etc/podinfo
  volumes:
    - name: podinfo
      downwardAPI:
        items:
          - path: "zone"
            fieldRef:
              fieldPath: metadata.labels['topology.kubernetes.io/zone']
```

### User Stories

* As an ML engineer, I want to optimize GPU-to-GPU communication during training which requires topology-awareness in my training code.
* As a database developer, I want to leverage topology information to improve fault tolerance of sharded databases on Kubernetes.
* As a developer of a Kubernetes CNI plugin, I want to pass topology information down to the CNI plugin to optimize data paths for container networks.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

* Scope creep. Allowing additional node information or node label info could
create security issues. This is mitigated by limiting the node labels
to strictly those that are standardized through KEP-1659.

* Exposing sensitive data as node labels to pods. This is mitigated by ensuring
standard topology labels are available to Pods.

* Stale data. Information obtained through node labels is like information
attained through a configmap or secret mounted to a pod, being passed on
creation but not guaranteed to be immutable and thus should be treated as so.


## Design Details

* A built-in Kubernetes admission plugin, `PodTopologyLabels` will be introduced in kube-apiserver
* The `PodTopologyLabels` admission plugin is responsible for mutating `pods/binding` subresource, adding topology labels matching the target Node.
* `PodTopologyLabels` admission will overwrite `topology.kubernetes.io/*` labels on Pods.
* A feature gate, `PodTopologyLabelsAdmission` will be introduced in v1.33. Alpha and disabled by default.
The `PodTopologyLabels` admission plugin can only be set when this feature gate is enabled.
* The Binding REST implementation will be updated to copy all labels from `pods/binding` subresource into Pods.
At this point we will overwrite Pod labels in Binding that are allowed to be exposed via Downward API.
* For exposing additional node labels, at the discretion of the cluster admin, a mutating admission webhook can be used to mutate labels of the `pods/binding` subresource.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

Unit tests will be added for:
* New `PodTopologyLabels` admission plugin
* Binding REST implementation

Unit tests will also ensure behavior is exercised when the feature gate is enabled.

##### Integration tests

Integration tests will be added to test the following behavior:
* Pods contain topology labels when `PodTopologyLabels` admission plugin is enabled.
* Topology labels can be expressed in Pod Downward API
* Node labels outside standard topology labels are disallowed
* Topology labels are empty if underlying Node does not specify a topology

Integration tests will also ensure behavior is exercised when the feature gate is enabled.

##### e2e tests

E2E tests will be added to test the following scenarios:
* Pods contain topology labels when `PodTopologyLabels` admission plugin is enabled.
* A Pod using downward API to retrieve the underlying topology information about the Node
* A Pod attempting to use downward API to retrieve Node labels that are not the standard topology labels
* A Pod using downward API on a Node that does not contain any topology information.
* Use MutatingAdmissionPolicy to add a custom node label that can be used via downward API.

E2E tests will also ensure behavior is exercised when the feature gate is enabled.

### Graduation Criteria

#### Alpha

- All standard topology labels can be retrieved using downward API.
- Behavior is implemented behind a feature gate that is off by default.
- Initial unit, integration and e2e tests completed and enabled.
- Fix standard topology label used in PodTopologyLabels admission controller (topology.k8s.io -> topology.kubernetes.io)

#### Beta

- Unit, integration and e2e tests

#### GA

TODO after Beta.

### Upgrade / Downgrade Strategy

Upgrade should be relatively safe because there no behavior changes unless topology labels
are specified in the downward API.

Already running Pods will not be affected by enabling this feature in kube-apiserver.

There are some risks to downgrade, for example, if a Pod relies on the topology
information using downward API, that information could get lost on downgrade.

### Version Skew Strategy

There should be no risks with version skew between control plane and nodes since there's no changes made to the kubelet's downward API implementation.
All changes proposed in this KEP are control-plane only.

For n-3 kubelets, old kubelets still support consuming Pod labels via downward API. This capability should be supported as long as the control plane
has this feature enabled.

There may be some risks for version skew with other components (e.g CSI or CNI) if the downward API is used
to pass topology information to those components. Components should ensure they are using a kube-apiserver version
that supports copying topology labels to Pods.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

A feature gate `PodTopologyLabelsAdmission` will be added to the kube-apiserver to enable or disable this feature.

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `PodTopologyLabelsAdmission`
  - Components depending on the feature gate: `kube-apiserver`
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

Yes, all Pods will have topology labels copied over.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes it can. For already running Pods, node topology will continue to be available via downward API. For any new Pods created,
the downward API will return empty values for node topologies.

###### What happens if we reenable the feature if it was previously rolled back?

New Pods will have topology labels copied over. Older Pods created while the feature was disabled will be missing topology labels.

###### Are there any tests for feature enablement/disablement?

Tests will be added to ensure feature gate works as expected.

### Rollout, Upgrade and Rollback Planning

Manual testing will be exercised to ensure that PodTopologyLabelsAdmission can be enabled and then disabled.
When disabled, existing Pods with topology labels will continue to run with those labels and new Pods will no longer
container topology labels.

###### How can a rollout or rollback fail? Can it impact already running workloads?

Failure on rollout is unlikely since we do not override any existing topology labels on Pods.
Already running workloads that is already copying topology labels to Pods should be untouched by the new admission plugin.

###### What specific metrics should inform a rollback?

Metrics in the scheduler such as `pod_scheduling_attempts` can inform a rollback if there's a bug in the REST binding implementation.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Upgrade / rollback will be manually tested before graduation to Beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

They can check if new Pods contain the `topology.kubernetes.io/*` labels.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason:
- [X] API .status
  - Condition name:
  - Other field: check Pod labels
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?


- [X] Metrics
  - Metric name: `pod_scheduling_attempts`, `scheduler_scheduling_attempt_duration_seconds`
  - [Optional] Aggregation method:
  - Components exposing the metric: kube-scheduler
- [] Other (treat as last resort)
  - Details: SLI are not necessary for this admission plugin

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No, we can use `pod_scheduling_attempts` and `scheduler_scheduling_attempt_duration_seconds`.

### Dependencies

N/A

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Negligible increase to Pod size.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Little impact as this feature is only relevant when scheduling and running Pods, which would be unavailable anyways

###### What are other known failure modes?

Not all Kubernetes clusters have Nodes with topology labels. Requesting topology information
in these clusters will result in empty values returned via downward API and some
applications failing to start if they rely on this information.

###### What steps should be taken if SLOs are not being met to determine the problem?

Revert feature gate and stop consuming downward API.

## Implementation History

- `v1.33`: initial KEP is accepeted and alpha implementation is complete
- `v1.34`: fix topology labels from topology.k8s.io to topology.kubernetes.io.

## Drawbacks

Topology information is a Node-level construct. By implementing this KEP we are allowing information from Nodes
to trickel down to Pods. Expanding this pattern to additional node information in the future can lead to
weaker security boundaries between Pods and Nodes.

## Alternatives

### Expose all node labels via downward API

Allow retrieval of arbitrary Node labels from Pods. This has too many security risks as Nodes are often shared by many tenants.

### Helper controller

Implement a controller that copies selected node labels to Pods - either as the same label exactly, or by
writing a Pod annotation based on that subset of node labels. The value of the annotation could be a JSON document.
This implementation is not feasible since controllers cannot guarantee labels are added before Pods are running.

### Init container to retrieve Node topology

Many users have implemented workarounds like an init container that fetches Node topologies. While we can make an official
implementation of this, it requires handing down high privledges to Pods and unnecessary load to kube-apiserver.

## Infrastructure Needed (Optional)
