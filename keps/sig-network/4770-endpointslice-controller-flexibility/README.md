# KEP-4770: EndpointSlice Controller Flexibility

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [Well-Known Label](#well-known-label)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Services Over Secondary Networks](#story-1-services-over-secondary-networks)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Well-Known Label](#well-known-label-1)
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
  - [Well-known label: Use Annotation as Selector](#well-known-label-use-annotation-as-selector)
  - [Well-known label: Use Dummy Selector](#well-known-label-use-dummy-selector)
  - [Well-known label: Disable the Kube-Controller-Manager Controllers](#well-known-label-disable-the-kube-controller-manager-controllers)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This proposal adds a new well-known label `service.kubernetes.io/endpoint-controller-name` to Kubernetes Services. This label disables the default Kubernetes EndpointSlice, EndpointSlice Mirroring and Endpoints controllers for the services where this label is applied and delegates the control of EndpointSlices to a custom EndpointSlice controller.

## Motivation

As of now, a service can be delegated to a custom Service-Proxy/Gateway if the label `service.kubernetes.io/service-proxy-name` is set. Introduced in [KEP-2447](https://github.com/kubernetes/enhancements/issues/2447), this allows custom Service-Proxies/Gateways to implement services in different ways to address different purposes / use-cases. However, the EndpointSlices attached to this service will still be reconciled in the same way as any other service. Addressing more purposes / use-cases, for example, different pod IP addresses, is therefore not natively possible.

Delegating EndpointSlice control would allow custom controllers to define their own criteria for pod availability, selecting different pod IPs than the pod.status.PodIPs and more. As a reference implementation, and since the EndpointSlice Reconciler has been moved into Staging in [KEP-3685](https://github.com/kubernetes/enhancements/issues/3685), the reconciler logic used by Kubernetes can be reused by custom EndpointSlice controllers.

### Goals

* Provide the ability to disable the Kubernetes EndpointSlice, EndpointSlice Mirroring and Endpoints controllers for particular services.
* Allow custom EndpointSlice Controllers to re-use the Service Selector field.
* Offer an explicit and standard way to indicate which component is managing the EndpointSlices for a particular service.

### Non-Goals

* Change / Replace / Deprecate the existing behavior of the Kubernetes EndpointSlice, EndpointSlice Mirroring and Endpoints controllers.
* Introduce additional supported types of the EndpointSlice controllers/Reconciler as part of Kubernetes.
* Modify the Service / EndpointSlice / Endpoints Specs.
* Changing the behavior of kube-proxy.
* Provide a new way to select backends for a Service.

## Proposal

#### Well-Known Label

`service.kubernetes.io/endpoint-controller-name` will be added as a well-known label applying on the Service object. When set on a service, no matter the service specs, the Endpoint, EndpointSlice, and EndpointSlice Mirroring controllers for that service will be disabled, thus Endpoints and EndpointSlices for this service will not be created by the Kubernetes Controller Manager. If the label is not set, the Endpoint, EndpointSlice, and EndpointSlice Mirroring controllers will be enabled for that service and the Endpoints and EndpointSlices will be handled as of today.

The EndpointSlice, EndpointSlice Mirroring and Endpoints controllers will obey this label both at object creation and on dynamic addition/removal/updates of this label.

### User Stories (Optional)

#### Story 1: Services Over Secondary Networks

As a Cloud Native Network Function (CNF) vendor, some of my Kubernetes services are handled by custom Service-Proxies/Gateways (using `service.kubernetes.io/service-proxy-name`) over secondary networks provided by, for example, [Multus](https://github.com/k8snetworkplumbingwg/multus-cni). IPs configured in the service and registered by the EndpointSlice controller must be only the secondary IPs provided by the secondary network provider.

Therefore, it must be possible to disable the default Kubernetes Endpoints and EndpointSlice Controller for certain services and use a specialized EndpointSlice reconciler implementation to create a controller for secondary network providers.

### Notes/Constraints/Caveats (Optional)

N/A

### Risks and Mitigations

The existing behavior will be kept by default, and the Kubernetes EndpointSlice Controller will not manage the Services with the label. This ensures services without the label to continue to be managed as usual.

This will have no effect on other EndpointSlice controller implementations since they will not be influenced by the presence of this label.

## Design Details

### Well-Known Label

The kube-controller-manager will pass to the Endpoints, EndpointSlice and EndpointSlice Mirroring Controllers an informer selecting services that are not labeled with `service.kubernetes.io/endpoint-controller-name`. Thus, if the label is added to an existing service (by updating the service), the service with the label will be considered as a deleted service for the controllers, and the Endpoints and EndpointSlices will be deleted. If a Service is created with the label, the controllers will not be informed about it, so the Endpoints and EndpointSlices will not be created. If the label is removed from an existing service (by updating the service), the service with the label will be considered as a newly created service for the controllers, and the Endpoints and EndpointSlices will be created.

In the Endpoints, EndpointSlice and EndpointSlice Mirroring Controllers, the behavior to create Endpoints/EndpointSlices on service creation and the behavior to delete the Endpoints/EndpointSlices on service deletion is already in place. Only the service informer passed to these controllers must be tweaked for the proposed well-known label (`service.kubernetes.io/endpoint-controller-name`) to work properly.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

TDB

##### Integration tests

- Usage of `service.kubernetes.io/endpoint-controller-name` on services
    * With the feature gate enable, a service is created with the label and the service has then no Endpoints neither EndpointSlices. Then service is updated removing the label and the service has now Endpoints and EndpointSlices.
    * With the feature gate enable, a service is created without the label, the service has Endpoints and EndpointSlices. Then service is updated with the label and the service has no longer any Endpoints nor EndpointSlices.
    * With the feature gate disabled, a service is created with the label and the service has EndpointSlices as it would have had without the label.

##### e2e tests

- Usage of `service.kubernetes.io/endpoint-controller-name` on services
    * A service is created with the label, an EndpointSlice is manually created with an endpoint (simulating an external controller). The test verifies the service is reachable. 

### Graduation Criteria

#### Alpha

- Feature implemented behind feature gates (`ExternalEndpointController`). Feature Gates are disabled by default.
- Documentation provided.
- Initial unit, integration and e2e tests completed and enabled.

#### Beta

- Feature Gates are enabled by default.
- No major outstanding bugs.
- Feedback collected from the community (developers and users) with adjustment provided, implemented and tested.

#### GA

- 2 examples of real-world usage.
- Allowing time for feedback from developers and users.

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ExternalEndpointController`
  - Components depending on the feature gate: kube-controller-manager
- [ ] Other
  - Describe the mechanism: 
  - Will enabling / disabling the feature require downtime of the control
    plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No

###### Does enabling the feature change any default behavior?

When the feature-gate `ExternalEndpointController` is enabled, the label `service.kubernetes.io/endpoint-controller-name` will work as described in this KEP. Otherwise, no, for the existing services without the `service.kubernetes.io/endpoint-controller-name` label, the EndpointSlice, EndpointSlice Mirroring and Endpoints controllers will continue to generate Endpoints and EndpointSlices for all services.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

If a service is labeled with `service.kubernetes.io/endpoint-controller-name`, and the feature is disabled, then the Kubernetes Controller Manager will start reconciling the Endpoints and EndpointSlices for this service. This could potentially cause traffic disturbance for the service as unexpected IPs (Pod.Status.PodIPs) will be registered to the EndpointSlices/Endpoints.

###### What happens if we reenable the feature if it was previously rolled back?

If a service is labeled with `service.kubernetes.io/endpoint-controller-name`, and the feature is re-enabled, then the Kubernetes Controller Manager (KCM) will stop reconciling the Endpoints and EndpointSlices for this service and will delete the existing KCM managed ones.

###### Are there any tests for feature enablement/disablement?

Enablement/disablement of this feature is tested as part of the integration tests.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

N/A

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [x] API .status
  - Condition name: 
  - Other field: When the `service.kubernetes.io/endpoint-controller-name` label is set on a service, no Endpointslice and no Endpoint will be created but the Kubernetes Controller Manager.
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
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

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- Initial proposal: 2024-07-19

## Drawbacks

TBD

## Alternatives

### Well-known label: Use Annotation as Selector 

Services without selectors will not get any EndpointSlice objects. Therefore, selecting pods can be done in different ways, for example, via annotation. An annotation will be used in the service to select which pods will be used as backend for this service. For example, [nokia/danm](https://github.com/nokia/danm) uses `danm.k8s.io/selector` (e.g. [DANM service declaration](https://github.com/nokia/danm/blob/v4.3.0/example/svcwatcher_demo/services/internal_lb_svc.yaml#L7)), and [projectcalico/vpp-dataplane](https://github.com/projectcalico/vpp-dataplane) uses `extensions.projectcalico.org/selector` (e.g. [Calico-VPP Multinet services](https://github.com/projectcalico/vpp-dataplane/blob/v3.25.1/docs/multinet.md#multinet-services)). To simplify the user experience, a mutating webhook could read the selector, add them to the annotation and clear them from the specs when the type of service is detected.

The custom EndpointSlice Controller will then read the annotation to select the pods targeted by the service.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: service
  annotations:
    selector: "app=a"
spec: {}
```

This alternative potentially leads to confusion among users and inconsistency in how services are managed as each implementation is using its own annotation (see the nokia/danm and projectcalico/vpp-dataplane examples), leading to a fragmented approach.

### Well-known label: Use Dummy Selector

The set of Pods targeted by a Service is determined by a selector, the labels in the selector must be included as part of the pod labels. If a dummy selector is added to the service, Kubernetes will not select any pod, the endpointslices created by Kubernetes will then be empty. To simplify the user experience, a mutating webhook could add the dummy selector when the type of service is detected.

The custom EndpointSlice Controller could read the service.spec.selector and ignore the dummy label to select pods targeted by the service.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: service
spec:
  selector:
    app: a
    dummy-selector: "true"
```

This alternative fails to prevent the placeholder (empty) EndpointSlice(s) to be created by Kube-Controller-Manager. This also potentially causes confusion among users as every implementation could use a different dummy-selector key. Additionally, a miss-configuration with the missing dummy label will lead to unintended EndpointSlices being created with Pod.Status.PodIPs.

### Well-known label: Disable the Kube-Controller-Manager Controllers

The list of controllers to enable in the Kube-Controller-Manager can be set using the `--controllers` flag ([kube-controller-manager documentation](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/)). The EndpointSlice can then be disabled in the Kube-Controller-Manager and implemented as an external one that will support the label feature described in this KEP.

This alternative requires significant changes to the cluster management as the cluster level configuration must be modified and a new EndpointSlice controller for the primary network must be developed and deployed to replace the disabled one in the Kube-Controller-Manager.

## Infrastructure Needed (Optional)

N/A
