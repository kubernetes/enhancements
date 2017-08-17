# Checklist for SIGs and Release Team
As SIGs fill out their sections by component, please check off that
you are finished. For guidance about what should have a release note
please check out the [release notes guidance][] issue.

- [ ] sig-api-machinery
- [ ] sig-apps
- [ ] sig-architecture
- [ ] sig-auth
- [ ] sig-autoscaling
- [ ] sig-aws
- [ ] sig-azure
- [ ] sig-big-data
- [ ] sig-cli
- [ ] sig-cluster-lifecycle
- [ ] sig-cluster-ops
- [ ] sig-contributor-experience
- [ ] sig-docs
- [ ] sig-federation
- [ ] sig-governance.md
- [ ] sig-instrumentation
- [ ] sig-network
- [ ] sig-node
- [ ] sig-on-premise
- [ ] sig-openstack
- [ ] sig-product-management
- [ ] sig-release
- [ ] sig-scalability
- [ ] sig-scheduling
- [ ] sig-service-catalog
- [ ] sig-storage
- [ ] sig-testing
- [ ] sig-ui
- [ ] sig-windows

[release notes guidance]: https://github.com/kubernetes/community/issues/484

## **Major Themes**

## **Action Required Before Upgrading**

* The autoscaling/v2alpha1 API has graduated to autoscaling/v2beta1.  The
  form remains unchanged.  HorizontalPodAutoscalers making use of features
  from the autoscaling/v2alpha1 API will need to be migrated to
  autoscaling/v2beta1 to ensure that the new features are properly
  persisted.

* The metrics APIs (`custom.metrics.metrics.k8s.io` and `metrics`) have
  graduated from `v1alpha1` to `v1beta1`.  If you have deployed a custom
  metrics adapter, ensure that it supports the new API version.  If you
  have deployed Heapster in aggregated API server mode, ensure that you
  upgrade Heapster as well. 

## **Known Issues**

## **Deprecations**

## **Notable Features**
#### Storage

* Capacity Isolation/Resource Management for Local Ephemeral Storage
* Block Volumes Support
* Enable containerization of mount dependencies
* Support Attach/Detach for RWO volumes such as iSCSI, Fibre Channel and RBD
* Volume Plugin Metrics
* Snapshots
* Resizing Volume Support
* Exposing StorageClass Params To End Users (aka Provisioning configuration in PVC)
* Mount Options to GA
* Allow configuration of reclaim policy in StorageClass
* Expose Storage Usage Metrics
* PV spec refactoring for plugins that reference namespaced resources: Azure File, CephFS, iSCSI, Glusterfs

#### Autoscaling and Metrics

* Support for custom metrics in the Horizontal Pod Autoscaler is moving to
  beta.  The associated metrics APIs (custom metrics and resource/master
  metrics) are graduating to v1beta1.  See [Action Required Before
  Upgrading](#action-required-before-upgrading).
