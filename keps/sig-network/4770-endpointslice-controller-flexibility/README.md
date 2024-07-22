# KEP-4770: EndpointSlice Controller Flexibility

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [Well-Known Label](#well-known-label)
    - [Flexibility on the EndpointSlice Reconciler Module](#flexibility-on-the-endpointslice-reconciler-module)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Well-Known Label](#well-known-label-1)
  - [Flexibility on the EndpointSlice Reconciler Module](#flexibility-on-the-endpointslice-reconciler-module-1)
    - [Reconciler](#reconciler)
      - [Service/Pods](#servicepods)
      - [Endpoints](#endpoints)
    - [Metrics](#metrics)
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
  - [Well-known label: Use Annotation as Selector](#well-known-label-use-annotation-as-selector)
  - [Well-known label: Use Dummy Selector](#well-known-label-use-dummy-selector)
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

This proposal adds a new well-known label `service.kubernetes.io/endpoint-controller-name` to Kubernetes Services. This label disables the default Kubernetes EndpointSlice controller for the services where this label is applied and delegates the control of EndpointSlices to a custom EndpointSlice controller.

Additionally, this KEP aims to give more flexibility for the EndpointSlice Reconciler module to allow users to reconcile EndpointSlices with any type of resources (Currently only Service/Pods is supported). For example, Endpoints could be supported for the Kubernetes EndpointSlice Mirroring Controller.

## Motivation

As of now, a service can be delegated to a custom Service-Proxy/Gateway if the label `service.kubernetes.io/service-proxy-name` is set. Introduced in [KEP-2447](https://github.com/kubernetes/enhancements/issues/2447), this allows custom Service-Proxies/Gateways to implement services in different ways to address different purposes / use-cases. However, the EndpointSlices attached to this service will still be reconciled in the same way as any other service. Addressing more purposes / use-cases, for example, different pod IP addresses, is therefore not natively possible.

Delegating EndpointSlice control would allow custom controllers to define their own criteria for pod availability, selecting different pod IPs than the pod.status.PodIPs and more. As a reference implementation, and since [KEP-3685](https://github.com/kubernetes/enhancements/issues/3685), the reconciler logic used by Kubernetes can be reused by custom EndpointSlice controllers.

Providing a generic EndpointSlice Reconciler module would allow users to reuse the same code as the Kubernetes EndpointSlice Controller, thus ensuring consistency and reducing the effort needed to implement custom reconciliation logic. As highlighted in this [comment](https://github.com/kubernetes/kubernetes/pull/118953#discussion_r1245970845), the EndpointSlice Mirroring Controller would also benefit from these changes since the generic EndpointSlice Reconciler could be used from the EndpointSlice Mirroring Controller.

### Goals

* Provide the ability to disable the Kubernetes EndpointSlice controller for particular services.
* Extend and enhance the EndpointSlice Reconciler module to make it generic.

### Non-Goals

* Change / Replace / Deprecate the existing behavior of the Kubernetes EndpointSlice controller.
* Introduce additional supported types of the EndpointSlice controllers/Reconciler as part of Kubernetes.
* Modify the Service / EndpointSlice Specs.
* Add new features to the EndpointSlice Mirroring Controller.

## Proposal

#### Well-Known Label

`service.kubernetes.io/endpoint-controller-name` will be added as a well-known label applying on the Service object. When set on a service, no matter the service specs, the Endpoint, EndpointSlice, and EndpointSlice Mirroring controllers for that service will be disabled, thus Endpoints and EndpointSlices for this service will not be created by the Kubernetes Controller Manager. If the label is not set, the Endpoint, EndpointSlice, and EndpointSlice Mirroring controllers will be enabled for that service and the Endpoints and EndpointSlices will be handled as of today.

The Kubernetes Controller Manager will implement this label both at object creation and on dynamic addition/removal/updates of this label.

#### Flexibility on the EndpointSlice Reconciler Module

The reconciler structure will support more features to cover the requirements set by the current behavior of the EndpointSlice Mirroring Controller (e.g.: placeholder EndpointSlice does not exist in the Mirroring Controller).

The `Reconcile` function definition will be changed to accept general types of data (not specific to Services) and the list of features the endpointslices being reconciled will support (e.g. Traffic Distribution). Functions for specific types (Service/Pods for the EndpointSlice Controller and Endpoints for the EndpointSlice Mirroring Controller) generating data to pass to the `Reconcile` function will be provided by the EndpointSlice Reconciler module.

The Metrics and its cache between the EndpointSlice Reconciler and EndpointSlice Mirroring Reconciler are similar but not identical. They will be merged together to provide a re-usable and non-type specific metrics package.

The EndpointSlice controller will be adapted accordingly to support the new way of using the EndpointSlice Reconciler.

Finally, the EndpointSlice Mirroring reconciler will be discarded from `pkg/controller/endpointslicemirroring` and the EndpointSlice Mirroring Controller will use the EndpointSlice reconciler from staging with the appropriate configuration.

### User Stories (Optional)

#### Story 1

As a Cloud Native Network Function (CNF) vendor, some of my services are handled by custom Service-Proxies/Gateways over secondary networks provided by, for example, [Multus](https://github.com/k8snetworkplumbingwg/multus-cni). IPs configured in the service and registered by the EndpointSlice controller must be only the secondary IPs provided by the secondary network provider (e.g. [Multus](https://github.com/k8snetworkplumbingwg/multus-cni)).

Therefore, it must be possible to disable the default Kubernetes Endpoints and EndpointSlice Controller for certain services and re-use the Kubernetes EndpointSlice reconciler implementation to create a controller for secondary network providers.

### Notes/Constraints/Caveats (Optional)

N/A

### Risks and Mitigations

The existing behavior will be kept by default, and the Kubernetes EndpointSlice Controller will not manage the Services with the label. This ensures services without the label to continue to be managed as usual.

This will have no effect on other EndpointSlice controller implementations since they will not be influenced by the presence of this label.

To avoid potential new functionalities to leak into other controllers re-using the EndpointSlice Reconciler (for example, the EndpointSlice Mirroring Controller), new Kubernetes only functionnalities must be optional if possible. No functionalities outside of kubernetes/kubernetes will be added to this controller.

## Design Details

### Well-Known Label

The kube-controller-manager will pass to the Endpoints, EndpointSlice and EndpointSlice Mirroring Controllers an informer selecting services that are not labeled with `service.kubernetes.io/endpoint-controller-name`. Thus, if the label is added to an existing service (by updating the service), the service with the label will be considered as a deleted service for the controllers, and the Endpoints and EndpointSlices will be deleted. If a Service is created with the label, the controllers will not be informed about it, so the Endpoints and EndpointSlices will not be created. If the label is removed from an existing service (by updating the service), the service with the label will be considered as a newly created service for the controllers, and the Endpoints and EndpointSlices will be created.

In the Endpoints, EndpointSlice and EndpointSlice Mirroring Controllers, the behavior to create Endpoints/EndpointSlices on service creation and the behavior to delete the Endpoints/EndpointSlices on service deletion is already in place. Only the service informer passed to these controllers must be tweaked for the proposed well-known label (`service.kubernetes.io/endpoint-controller-name`) to work properly.

### Flexibility on the EndpointSlice Reconciler Module

The current behavior of the EndpointSlice Reconciler if a service exists but does not have any endpoint behind, is to create an placeholder EndpointSlice for that service. The placeholder EndpointSlice will have no endpoints and no ports. If the service is dualstack, then 2 placeholder EndpointSlices will be created, one for each IP Family (IPv4 and IPv6).

New fields will be added to the `Reconciler` struct to cover the needs of the EndpointSlice and EndpointSlice Mirroring Controllers. Unlike the EndpointSlice Mirroring Controller, the EndpointSlice Controller needs placeholders if a service does not contain any endpoint. So a boolean `placeholderEnabled` will be added. The EndpointSlice Mirroring Controller does not verify which resource owns the EndpointSlice before Updating/Deleting it, another boolean `ownershipEnforced` would control this.

#### Reconciler 

The type specific parameters like `pods` and `service` will be removed from the signature of the `Reconcile` function. The `ownerObjectRuntime`, `ownerObjectMeta`, `desiredEndpointSlices`, `supportedAddressesTypes`, `trafficDistribution` and `setEndpointSliceLabelsAnnotationsFunc` should now be provided. 

`ownerObjectRuntime` and `ownerObjectMeta` are used for setting the owner reference of the EndpointSlices, the name generation, namespace and the labels/annotations features (e.g.: Topology Mode).

`desiredEndpointSlices` and `supportedAddressesTypes` must now be pre-determined. Functions for the Pods/Service (EndpointSlice Controller) and Endpoints (EndpointSlice Mirroring Controller) will be provided as part of the module.

A function implementing the `SetEndpointSliceLabelsAnnotations` type will be passed as parameter of the `Reconcile` function to be called when a new EndpointSlice will be created and when the existing EndpointSlice(s) will be checked for updates. The reason to expose this is the different behavior to handle the labels and annotations between the different controllers handling the EndpointSlices in Kubernetes.

```golang
// SetEndpointSliceLabelsAnnotations returns the labels and annotations to be set for the endpointslice passed as parameter.
// The bool Changed returned indicates that the labels and annotations have changed and must be updated.
type SetEndpointSliceLabelsAnnotations func(
    logger klog.Logger, 
    epSlice *discovery.EndpointSlice, 
    controllerName string,
) (labels map[string]string, annotations map[string]string, changed bool)

// EndpointPortAddressType represents endpointslice(s) to be reconciled.
type EndpointPortAddressType struct {
	// List of endpoints to be included in the EndpointSlice(s).
	EndpointSet endpointsliceutil.EndpointSet
	// List of ports to be set for the EndpointSlice(s).
	Ports []discovery.EndpointPort
	// Address type of the EndpointSlice(s).
	AddressType discovery.AddressType
}

// Reconcile takes a set of desired endpointslices and
// compares them with the endpoints already present in any existing endpoint
// slices. It creates, updates, or deletes endpoint slices
// to ensure the desired set of endpoints are represented by endpoint slices.
func (r *Reconciler) Reconcile(
	logger klog.Logger,
	ownerObjectRuntime runtime.Object, // Required to get the GVK of the owner.
	ownerObjectMeta metav1.Object, // Required to get the name and namespace of the owner.
	desiredEndpointSlices []*EndpointPortAddressType,
	existingSlices []*discovery.EndpointSlice,
	supportedAddressesTypes sets.Set[discovery.AddressType],
	trafficDistribution *string, // Service feature.
    // Required to handle different behavior to set labels and annotations between the 
    // EndpointSlice Controller and the EndpointSlice Mirroring Controller.
	setEndpointSliceLabelsAnnotationsFunc SetEndpointSliceLabelsAnnotations,
	triggerTime time.Time, // Sets endpoints.kubernetes.io/last-change-trigger-time, removes the annotation if IsZero() is true.
) error
```

##### Service/Pods 

The `desiredEndpointSlices` and `supportedAddressesTypes` parameters to be passed to the `Reconcile` function will be returned by the `DesiredEndpointSlicesFromEndpoints`. The logic of this function will be taken from the current `reconcileByAddressType` function of EndpointSlice reconciler module here: [k8s.io/endpointslice/reconciler.go#L166-L212](https://github.com/kubernetes/endpointslice/blob/v0.30.2/reconciler.go#L166-L212).

The `SetLabels` function implements `SetEndpointSliceLabelsAnnotations`. The Pods/Service (EndpointSlice Controller) strategy is to clone the labels of the service (except "endpointslice.kubernetes.io/managed-by", "kubernetes.io/service-name" and "service.kubernetes.io/headless") and update the EndpointSlice if those got updated. In case of an update, the "service.kubernetes.io/headless", "kubernetes.io/service-name" and "endpointslice.kubernetes.io/managed-by" labels will also be set.

```golang
// DesiredEndpointSlicesFromServicePods returns the list of desired endpointslices for the given pods and services.
// It also return which address types can be handled by this service.
func DesiredEndpointSlicesFromServicePods(
	logger klog.Logger,
	pods []*corev1.Pod,
	service *corev1.Service,
	nodeLister corelisters.NodeLister,
) ([]*EndpointPortAddressType, sets.Set[discovery.AddressType], error)

type LabelsFromService struct {
	Service *corev1.Service
}

// SetLabels returns a map with the new endpoint slices labels and true if there was an update.
// Slices labels must be equivalent to the Service labels except for the reserved IsHeadlessService, LabelServiceName and LabelManagedBy labels
// Changes to IsHeadlessService, LabelServiceName and LabelManagedBy labels on the Service do not result in updates to EndpointSlice labels.
func (lfs *LabelsFromService) SetLabels(logger klog.Logger, epSlice *discovery.EndpointSlice, controllerName string) (map[string]string, map[string]string, bool)
```

Example of usage replacing the current `Reconcile` call from the EndpointSlice Controller ([pkg/controller/endpointslice/endpointslice_controller.go#L396](https://github.com/kubernetes/kubernetes/blob/v1.30.2/pkg/controller/endpointslice/endpointslice_controller.go#L396)):
```golang
desiredEndpointsByAddrTypePort, supportedAddressesTypes, _ := endpointslicerec.DesiredEndpointSlicesFromServicePods(
    logger, 
    pods, 
    service, 
    c.nodeLister,
)
labelsFromService := endpointslicerec.LabelsFromService{Service: service}
_ = c.reconciler.Reconcile(
    logger,
    service,
    service,
    desiredEndpointsByAddrTypePort,
    endpointSlices,
    supportedAddressesTypes,
    service.Spec.TrafficDistribution,
    labelsFromService.SetLabels,
    lastChangeTriggerTime,
)
```

##### Endpoints

The `desiredEndpointSlices` and `supportedAddressesTypes` parameters to be passed to the `Reconcile` function will be returned by the `DesiredEndpointSlicesFromEndpoints`. The logic of this function will be taken from the reconciler of EndpointSlice Mirroring Controller here: [pkg/controller/endpointslicemirroring/reconciler.go#L166-L212](https://github.com/kubernetes/kubernetes/blob/v1.30.2/pkg/controller/endpointslicemirroring/reconciler.go#L71-L110).

The `SetLabelsAnnotations` function implements `SetEndpointSliceLabelsAnnotations`. The Endpoints (EndpointSlice Mirroring Controller) strategy is to set the "kubernetes.io/service-name" and "endpointslice.kubernetes.io/managed-by" labels, clones the labels and annotations from the Endpoints and updates the EndpointSlice if there is any changes.

```golang
// DesiredEndpointSlicesFromServicePods returns the list of desired endpointslices for the given endpoints.
// It also return which address types are handled.
func DesiredEndpointSlicesFromEndpoints(
	endpoints *corev1.Endpoints,
	maxEndpointsPerSubset int32,
) ([]*EndpointPortAddressType, sets.Set[discovery.AddressType])

type LabelsAnnotationsFromEndpoints struct {
	Endpoints *corev1.Endpoints
}

func (lafe *LabelsAnnotationsFromEndpoints) SetLabelsAnnotations(logger klog.Logger, epSlice *discovery.EndpointSlice, controllerName string) (map[string]string, map[string]string, bool) 
```

Example of usage replacing the current `Reconcile` call from the EndpointSlice Mirroring Controller ([pkg/controller/endpointslicemirroring/endpointslicemirroring_controller.go#L334](https://github.com/kubernetes/kubernetes/blob/v1.30.2/pkg/controller/endpointslicemirroring/endpointslicemirroring_controller.go#L334)):
```golang
desiredEndpoints, supportedAddressesTypes := endpointslicerec.DesiredEndpointSlicesFromEndpoints(endpoints, r.maxEndpointsPerSubset)
return rec.Reconcile(
    logger,
    endpoints,
    endpoints,
    desiredEndpoints,
    existingSlices,
    supportedAddressesTypes,
    nil, // EndpointSlice Mirroring doesn't use traffic distribution.
    labelsAnnotationsFromEndpoints.SetLabelsAnnotations,
    time.Time{}, // So endpoints.kubernetes.io/last-change-trigger-time will be removed.
)
```

#### Metrics

Most of the metrics exposed by the EndpointSlice and EndpointSlice Mirroring Controllers are similar. The EndpointSlice Controller exposes `endpointslices_changed_per_sync`, `syncs` and `services_count_by_traffic_distribution` that are not exposed by the EndpointSlice Mirroring Controller. The EndpointSlice Mirroring Controller exposes `endpoints_updated_per_sync`, `addresses_skipped_per_sync` and `endpoints_sync_duration` that are not exposed by the EndpointSlice Controller. Metrics exposed will need to be merged together, and the metrics package should be changed to support any type of resources (pods/service, endpoints...).

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

* Tests in `pkg/controller/endpointslicemirroring/reconciler_test.go` will be moved to `staging/src/k8s.io/endpointslice`

##### Integration tests

- Usage of `service.kubernetes.io/endpoint-controller-name` on services
    * A service is created with the label and the service has then no endpoints neither endpointslices. Then service is updated removing the label and the service has now endpoints and endpointslices.
    * A service is created without the label, the service has endpoints and endpointslices. Then service is updated with the label and the service has no longer any endpoints nor endpointslices.

##### e2e tests

TDB

### Graduation Criteria

TDB

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism: setting the label `service.kubernetes.io/endpoint-controller-name` disables the Kubernetes endpointslice controller.
  - Will enabling / disabling the feature require downtime of the control
    plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No

###### Does enabling the feature change any default behavior?

N/A

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

N/A

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

N/A

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

- Initial proposal: 2024-06-01

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

## Infrastructure Needed (Optional)

N/A
