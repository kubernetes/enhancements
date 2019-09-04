---
title: Kustomize CRD Integration
authors:
 - "@jbrette"
owning-sig: sig-cli
participating-sigs:
 - sig-apps
 - sig-api-machinery
reviewers:
 - "@monopole"
 - "@Liujingfang1"
approvers:
 - "@monopole"
 - "@Liujingfang1"
editors:
 - "@jbrette"
creation-date: 2019-09-15
last-updated: 2019-09-15
status: provisional
see-also:
replaces:
superseded-by:
 - n/a
---

# Kustomize CRD Integration

## Table of Contents
- [Kustomize CRD Integration](#kustomize-crd-integration)
  - [Table of Contents](#table-of-contents)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [Merge Patch handling](#merge-patch-handling)
    - [Simple transformers handling](#simple-transformers-handling)
    - [Complex transformers handling](#complex-transformers-handling)
    - [Variable Transformer Integration](#variable-transformer-integration)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Graduation Criteria](#graduation-criteria)
  - [Doc](#doc)
  - [Test plan](#test-plan)
    - [Version Skew Tests](#version-skew-tests)
  - [Implementation History](#implementation-history)
  - [Alternatives](#alternatives)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Summary

CRDs are increasly beeing used in Kubernetes projects. Istio, OpenShift, Argo resources, to name a few, are more and more present. Also most of builtin transformers can be configure (even if it can be quite tedious) to support CRDs, handling of the patches by the strategicMergePatch transformer is different between K8s native objects and CRDs. The first are using the SMP, the later are using JsonMergePatch.

We are aiming at addressing those issues:
- CRD system for OpenShift resources not interpreted as expected [#1531](https://github.com/kubernetes-sigs/kustomize/issues/1531)
- kustomize handling of CRD inconsistent with k8s handling starting in 1.16 [#1510](https://github.com/kubernetes-sigs/kustomize/issues/1510)

## Motivation

### Goals

Provide the ability for the user to integrate new CRD into kustomize flows:

- Help to register new CRDs into kustomize scheme to leverage SMP instead of just JMP.
- Create/Adjust built-in transformers config such as prefix, suffix, namespaces, annotations, namereference and varreference transformers.
- Adjust kustomize configuration to account for Cluster scoped CRDs.
- Help the user to leverage syntax check transformers such as kubeval.
  
### Non-Goals

- Account for improvments done in kubernetes in CRDs handling, such as go lang annotations to improve patching procedures.
- Keep kustomize build process independant of the CRDs go modules.
  
## Proposal

The proposal is to try to align the behavior of the kustomize with the behavior of kubernetes/kubectl
- The proper handling of patching is done by the ability of a kubernetes operator to register the schema.
- The registration of the CRDs into the system, including the syntax is done through a yaml or json file.

### Merge Patch handling

The POC of solution is using the kustomize v3 plugin concept. The following dummy transformer/register allows
the user to build the following kustomize external plugin (go module) and get the kustomize to load it.

```go
package main

import (
        "github.com/keleustes/armada-crd/pkg/apis/armada/v1alpha1"
        "k8s.io/client-go/kubernetes/scheme"
        "sigs.k8s.io/kustomize/v3/pkg/ifc"
        "sigs.k8s.io/kustomize/v3/pkg/resmap"
)

// plugin loads the ArmadaChart CRD scheme into kustomize
type plugin struct {
        ldr ifc.Loader
        rf  *resmap.Factory
}

//nolint: golint
//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(
        ldr ifc.Loader, rf *resmap.Factory, _ []byte) (err error) {
        p.ldr = ldr
        p.rf = rf

        // Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
        return v1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme)
}

func (p *plugin) Transform(m resmap.ResMap) error {
        return nil
}

```

The corresponding transformer.yaml:

```yaml
apiVersion: armada.airshipit.org/v1alpha1
kind: ArmadaCRDRegister
metadata:
  name: armadacrdregister
```

The correspoding kustomization.yaml:

```yaml
resources:
- ./resources.yaml

transformers:
- ./transformer.yaml
```

This scheme registration is what drive kustomize to use [SMP vs JMP](https://github.com/kubernetes-sigs/kustomize/blob/master/k8sdeps/transformer/patch/conflictdetector.go#L154)

```go
		versionedObj, err := scheme.Scheme.New(toSchemaGvk(id.Gvk))
		if err != nil && !runtime.IsNotRegisteredError(err) {
			return nil, err
		}
		var cd conflictDetector
		if err != nil {
			cd = newJMPConflictDetector(rf)
		} else {
			cd, err = newSMPConflictDetector(versionedObj, rf)
			if err != nil {
				return nil, err
			}
```

This way of loading the scheme should improved and streamlined to be more user friendly. 

### Simple transformers handling

When creating a new CRD, generating the yaml file used for kubectl and kustomize is necessary for deployment and quite [straightforward](https://github.com/keleustes/armada-crd/blob/master/Makefile#L45):
```bash
controller-gen crd paths=./pkg/apis/armada/... crd:trivialVersions=true output:crd:dir=./kubectl output:none
```

Using the yaml definition of the CustomResourceDefinition, could be done dynamically in the same way kustomize is currently loading the JSON version of those CRDs. This would:
- allow to register the Kind, Group and Version of the new CRD
- allow to register the scope of the CRD (Cluster/Namespaced).
- allow to have access to some part of the syntax (so the fieldSpecs)
- Kustomize could automatically detect those CRD.yaml among the list of files provided in the "resources:" field.

An offline tool should also be able to scan the CRD.yaml and create the skeleton of configuration for each builtin transformer. User intervention would most likely be needed to trim the transformer config.
This tool would help create most of the fieldSpecs path: 

This would address two issues:
- Creating fieldSpec is quite error prone because some fields such as labels are often located deep inside the specs of the CRD (Path for Deployment is really deep). 
- The mutableField method at the core the transformer needs to follow with specific rules to handle slice of maps. 

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  name: armadachartgroups.armada.airshipit.org
spec:
  additionalPrinterColumns:
  - JSONPath: .status.actualState
    description: State
    name: State
    type: string
  - JSONPath: .spec.targetState
    description: Target State
    name: Target State
    type: string
  - JSONPath: .status.satisfied
    description: Satisfied
    name: Satisfied
    type: boolean
  group: armada.airshipit.org
  names:
    kind: ArmadaChartGroup
    listKind: ArmadaChartGroupList
    plural: armadachartgroups
    shortNames:
    - acg
    singular: armadachartgroup
  scope: Namespaced
  subresources:
    status: {}
  validation:
    openAPIV3Schema:
      description: ArmadaChartGroup is the Schema for the armadachartgroups API
      properties:
        apiVersion:
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        spec:
          description: ArmadaChartGroupSpec defines the desired state of ArmadaChartGroup
          properties:
            chartRefs:
              description: reference to chart document
              items:
                type: string
              type: array
            sequenced:
              description: enables sequenced chart deployment in a group
              type: boolean
            targetState:
              description: Target state of the ChartGroups
              type: string
          required:
          - charts
          - targetState
          type: object
        status:
          description: defines the observed state of ArmadaChartGroup
          properties:
            actualState:
              description: Actual state of the  ChartGroups
              type: string
            conditions:
              description: List of conditions and states related to the resource.
              items:
                description: ResourceCondition represents one current condition of
                  a resource.
                properties:
                  lastTransitionTime:
                    format: date-time
                    type: string
                  message:
                    type: string
                  reason:
                    type: string
                  resourceName:
                    type: string
                  resourceVersion:
                    format: int32
                    type: integer
                  status:
                    description: ResourceConditionStatus represents the current status
                      of a Condition
                    type: string
                  type:
                    type: string
                required:
                - status
                - type
                type: object
              type: array
            reason:
              description: Reason indicates the reason for any related failures.
              type: string
            satisfied:
              description: Satisfied indicates if the actual state satisfies its target
                state
              type: boolean
          required:
          - actualState
          - satisfied
          type: object
      type: object
  version: v1alpha1
  versions:
  - name: v1alpha1
    served: true
    storage: true
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: []
  storedVersions: []
```

### Complex transformers handling

The above simple CRD definition example also highlights the complexity of dealing with:
- NameReferences Configuration: The `chartRefs` is an array of reference to a 'chart' CRD. The names of those charts are impacted by prefix/suffix. It means the nameReference transformer configuration has to be be updated.

Note: If you take the example of ArgoRollout CRD which syntax is very similar to the Deployment syntax, most of the kustomize default configuration related to Deployment resources have to be duplicated to create the ArgoRollout entries.

Kustomize supports creation of some of the internal transformer config by ingesting json matching the CRD,
assuming that the following annotations have been addded to the [JSON file](https://github.com/kubernetes-sigs/kustomize/blob/master/pkg/transformers/config/factorycrd_test.go#L120):
- "x-kubernetes-annotation"
- "x-kubernetes-label-selector"
- "x-kubernetes-identity"
- "x-kubernetes-object-ref-api-version"
- "x-kubernetes-object-ref-kind"
- "x-kubernetes-object-ref-name-key"

The complexity relies in:
- accessing/generating the swagger.json equivalent which goes through the usage of kube-openapi (openapi-gen). 
  - [airshipit](https://github.com/keleustes/armada-crd/blob/master/Makefile#L65)
  - [argoproj](https://github.com/argoproj/argo/blob/master/pkg/apis/workflow/v1alpha1/openapi_generated.go)
- adding the annotations (x-kubernetes-object-ref-kind) to the generated swagger.json seems to be manual.
- the keys important for merging are currently not beeing used by kustomize [x-kubernetes-patch-merge-key](https://github.com/argoproj/argo/blob/master/api/openapi-spec/swagger.json#L754)

The swagger.json needs to be further refined to be usable by kustomize or by the very useful `kubeval` transformer (https://github.com/keleustes/armada-crd/blob/master/Makefile#L81)

We need to find a way to streamline all that work, so that kustomize can handle are CRD in the same simple way it can handle a native K8s object.


### Variable Transformer Integration

Almost every single field of a CR can potentially be the target of the variable replacement. 
Forcing the user to manually translate the CRD.yaml into the VariableResolutionTransformer configuration is not pratical. In the case, kustomize could leverage the [Automatic Creation of 'vars:' and 'varReferences:' sections #1217](https://github.com/kubernetes-sigs/kustomize/pull/1217) internally add the following entries in the kustomization.yaml

If the user creates the following entries:

```yaml
kind: MyCRD1
metadata:
   name: mycrd1
spec:
   somefield: $(MyCRD2.mycrd2.spec.someotherfield)
```

```yaml
kind: MyCRD2
metadata:
   name: mycrd2
spec:
   someotherfield: automatic variable resolution
```

Kustomize, when using the PR, internally automatically creates the following entries:

in the kustomization.yaml:

```yaml
var:
- name: MyCRD2.mycrd2.spec.someotherfield
  objref:
    kind: MyCRD2
    name: mycrd2
  fieldref:
    fieldpath: spec.someotherfield
```

in the transformer configuration kustomizeconfig.yaml:

```yaml
varReference:
- kind: MyCRD1
  path: spec/somefield
```

### Risks and Mitigations

- Risk: Using go modules/plugins is quite tricky. If a plugin requires a version of module different from the one used when
  kustomize was build (from instance yaml...), the plugin will not load.
- Risk: The "registration" of the new types need to happen before the first merge/mergeconflictdetection is invoked.
- Risk: During the SMP invocation, some of the syntax check is forced before the other transformers and generators are invoked.
  This may force the user to write update his patches to be compliant with new rules (similar to be forced to specify the name: attribute when patching the container slice of a Deployment)

## Graduation Criteria

Use customer feedback to determine if we should support:

- Have a utilitary tool used to be build a plugin/transformers that would excapsulate the transformer configurations for that CRD. 
- Ability to get kustomize to load that plugin at runtime.

## Doc

Update Kubectl Book and Kustomize Documentation

## Test plan

Unit Tests for:

- [ ] Ability to load AirshipIt CRDs.
- [ ] Ability to load OpenShift CRDs
- [ ] Ability to load ArgoProj CRDs.
- [ ] Ability to load Istio CRDs.

### Version Skew Tests

## Implementation History

The following POC PR have been proposed:

- [Get Argo, Istio, OpenShift and Airship CRD to use SMP instead of JMP #1539](https://github.com/kubernetes-sigs/kustomize/pull/1539)
- [Automatic Creation of 'vars:' and 'varReferences:' sections #1217](https://github.com/kubernetes-sigs/kustomize/pull/1217)

## Alternatives

A workaround to not using the 
[scheme](https://github.com/kubernetes-sigs/kustomize/blob/master/k8sdeps/transformer/patch/conflictdetector.go#L154) but would force kustomize to reimplement the content of [k8s.io/apimachinery/pkg/util/strategicpatch
](https://github.com/kubernetes-sigs/kustomize/blob/master/k8sdeps/transformer/patch/conflictdetector.go#L133), but would still be possible.
