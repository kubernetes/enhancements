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

- The kubernetes workloads API (the DaemonSet, Deployment, ReplicaSet, and 
StatefulSet kinds) have been moved to the new apps/v1beta2 group version. This
is the current version of the API, and the version we intend to promote to 
GA in future releases. This version of the API introduces several deprecations 
and behavioral changes, but its intention is to provide a stable, consistent 
API surface for promotion. 

## **Action Required Before Upgrading**

## **Known Issues**

## **Deprecations**

### Apps 
 - The rollbackTo field of the Deployment kind is depreatcted in the 
 apps/v1beta2 group version. 
 - The templateGeneration field of the DaemonSet kinds is deprecated in the 
 apps/v1beta2 group.
 - The pod.alpha.kubernetes.io/initialized has been removed.


## **Notable Features**

### [Workload API (apps/v1beta2)](https://github.com/kubernetes/features/issues/353)

Kubernetes 1.8 adds the apps/v1beta2 group version. This group version contains 
the Kubernetes workload API which consists of the DaemonSet, Deployment, 
ReplicaSet and StatefulSet kinds. It is the current version of the API, and we 
intend to promote it to GA in upcoming releases 

#### API Object Additions and Migrations

- The current version DaemonSet, Deployment, ReplicaSet, and StatefulSet kinds 
  are now in the apps/v1beta2 group version.
- The apps/v1beta2 group version adds a Scale subresource for the StatefulSet 
kind.
- All kinds in the apps/v1beta2 group version add a corresponding conditions 
  kind.

#### Behavioral Changes

 - For all kinds in the API group version, as it is incompatible with kubectl 
 apply and strategic merge patch, spec.selector defaulting is disabled. Users
 must set the spec.selector in their manifests, and the creation of an object 
 with a spec.selector that does not match the labels in its spec.template is 
 considered to be invalid.
 - As none of the controllers in the workloads API handle selector mutation in 
 a consistent way, selector mutation is disabled in for all kinds in the 
 app/v1beta2 group version. This restriction may be lifted in the future, but 
 it is likely that that selectors will remain immutable after GA promotion. 
 Users that have any code that depends on mutable selectors may continue to use 
 the apps/v1beta1 API for this release, but they should begin migration to code 
 that does depend on mutable selectors.

 #### Defaults

 - The default spec.updateStrategy for the StatefulSet and DaemonSet kinds is 
 RollingUpdate for the apps/v1beta2 group version. Users may specifically set 
 the OnDelete strategy, and no strategy auto-conversion will be applied to 
 replace defaulted values.
 - As mentioned in [Behavioral Changes](#behavioral-changes), selector 
 defaulting is disabled.
 - The default spec.revisionHistoryLimit for all applicable kinds in the 
 apps/v1beta2 group version has set to 10.

#### CLI Changes

- The kubectl rollout and rollback implementation is complete for StatefulSet.
- The kubectl scale command will uses the Scale subresource for kinds in the 
  apps/v1beta2 group.
- kubectl delete will no longer scale down workload API objects prior to 
  deletion. Users who depend on ordered termination for the Pods of their 
  StatefulSet’s must use kubectl scale to scale down the StatefulSet prior to 
  deletion.




