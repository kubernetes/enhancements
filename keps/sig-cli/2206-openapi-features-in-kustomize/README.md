# KEP-2206: OpenAPI Features in Kustomize

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Story](#user-story)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
- [Implementation History](#implementation-history)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
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
OpenAPI is an open specification for defining types and operations on those 
types.  The Kubernetes API server has supported it fully since version 1.15,
meaning one can query an API server to get a detailed list of the types it 
understands, including any Custom Resources that it knows about.  Clients can 
use this information to correctly instantiate those types and make calls on 
the server.

Kustomize uses a hard-coded snapshot of the kubernetes API taken at version 
1.19 to help it understand field types and most importantly, merge keys to 
use when patching type instances.  This is how, for example, kustomize can 
change the `image` name used in a particular container in a particular 
Deployment instance.

Two immediate problems with this setup are 1), to upgrade the OpenAPI version, 
once must recompile kustomize with a new snapshot, 2) there’s no simple way to 
teach kustomize about a custom resource (field types, field validation rules, 
merge keys), despite the availability of OpenAPI schema data from custom 
resource tooling like kubebuilder.

If a kustomize user wants to specify their own OpenAPI schema files to use. 
Because the Kubernetes OpenAPI schema is currently built into the kustomize 
binary, that strategic merging for custom resources is not supported well. 
If a kustomize user were able to specify their own custom OpenAPI schema via 
an `openapi` field in a kustomization file, they would be able to specify 
patch strategy and merge key information for their custom resources. This 
proposal describes what that openapi field would look like and how the user 
would be able to get an OpenAPI schema containing their custom resource 
information. 

## Motivation

Allowing users to specify their own OpenAPI schema files will give them more 
flexibility in how kustomize does strategic merging, especially for custom 
resources. Users will be able to choose the patch strategy and merge keys for 
their resources. 

For a short presentation, see https://docs.google.com/presentation/d/1p3ASgdGvLYoUVLmx60_rjLuwuz4ypzwgj2ZWp7-kCjQ/preview?slide=id.gb19ebdc14e_0_0

### Goals

- In the `openapi` field of a kustomization file, a user may specify a path 
to an OpenAPI schema file to use instead of whatever version was compiled in 
to kustomize.
- Add a `kustomize openapi fetch` command that will fetch the OpenAPI schema 
from the user’s current cluster
- Create the necessary conditions to merge open API data - e.g. read one 
large file containing the full kubernetes native type set, and one or more 
relatively small files containing custom resource data (this would eliminate 
the need to first load CRDs into an API server then fetch everything back as 
one large blow).

### Non-Goals

- Kustomize will not require the ability to contact a live API server in 
order to perform a build.  The act of acquiring an openAPI spec and using 
that spec in a build will be completely independent.

## Proposal

The kustomization file would have a new `openapi` field:
```
openapi:
  path: myschema.json
```

Kustomize would also have a new command to fetch the OpenAPI schema from 
the user’s current cluster. 
```
$ kustomize openapi fetch <outputFileName>
Fetching OpenAPI document using the current context in kubeconfig
```


### User Story

This user story is based on kubernetes-sigs/kustomize#2825. 

Suppose a user has a custom resource definition: 

```

apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: mycrds.example.com
spec:
  group: example.com
  versions:
    - name: v1alpha1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                template:
                  type: object
                  properties:
                    spec:
                      type: object
                      properties:
                        containers:
                          type: array
                          items:
                            type: object
                            properties:
                              name:
                                type: string
                              image: 
                                type: string
                              command: 
                                type: string
                              ports:
                                type: array
                                items:
                                  type: object
                                  properties:
                                    name:
                                      type: string
                                    protocol:
                                      type: string
                                    containerPort:
                                      type: integer

  scope: Namespaced
  names:
    plural: mycrds
    singular: mycrd
    kind: MyCRD
    shortNames:
    - mc

```


They can apply this custom resource definition to their cluster, and 
define their custom resource: 

```
apiVersion: example.com/v1alpha1
kind: MyCRD
metadata:
  name: service
spec:
  template:
    spec:
      containers:
      - name: server
        image: server
        command: example
        ports:
        - name: grpc
          protocol: TCP
          containerPort: 8080
```


Say they want to update this resource via a patch in a kustomization 
file, to change the image to `nginx`:

```
resources:
- mycrd.yaml

patchesStrategicMerge:
- |-
  apiVersion: example.com/v1alpha1
  kind: MyCRD
  metadata:
    name: service
  spec:
    template:
      spec:
        containers:
        - name: server
          image: nginx
```


The output of `kustomize build` currently would be:

```
apiVersion: example.com/v1alpha1
kind: MyCRD
metadata:
  name: service
spec:
  template:
    spec:
      containers:
      - image: nginx
        name: server
```

Instead of merging the patch with the resource, the entire containers 
field is overwritten. To fix this, they can fetch their cluster’s 
OpenAPI schema using `kustomize openapi fetch myschema.json`, which 
will include OpenAPI data for their custom resource.  They will have 
to edit that OpenAPI schema if they want to have a different patch 
strategy and merge key for kustomize to use. For example, they may 
edit their custom resource OpenAPI data as follows, to include patch 
strategy and merge key information from the PodTemplateSpec:

```
{
  "definitions": {
    "v1alpha1.MyCRD": {
      "description": "MyCRD is the Schema for the mycrd API",
      "properties": {
        "apiVersion": {
          "description": "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources",
          "type": "string"
        },
        "kind": {
          "description": "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds",
          "type": "string"
        },
        "metadata": {
          "type": "object"
        },
        "spec": {
          "description": "MyCRDSpec defines the desired state of MyCRD",
          "properties": {
            "template": {
              "$ref": "#/definitions/io.k8s.api.core.v1.PodTemplateSpec",
              "description": "Template describes the pods that will be created."
            }
          },
          "required": [
            "template"
          ],
          "type": "object"
        },
        "status": {
          "description": "MyCRDStatus defines the observed state of MyCRD",
          "properties": {
            "success": {
              "type": "boolean"
            }
          },
          "type": "object"
        }
      },
      "type": "object",
      "x-kubernetes-group-version-kind": [
        {
          "group": "example.com",
          "kind": "MyCRD",
          "version": "v1alpha1"
        }
      ]
    }
  }
}
```

Then, they can write a kustomization file:

```
resources:
- mycrd.yaml

openapi:
  path: mycrd_schema.json

patchesStrategicMerge:
- |-
  apiVersion: example.com/v1alpha1
  kind: MyCRD
  metadata:
    name: service
  spec:
    template:
      spec:
        containers:
        - name: server
          image: nginx
```

The result will merge the patch with the resource correctly: 

```
apiVersion: example.com/v1alpha1
kind: MyCRD
metadata:
  name: service
spec:
  template:
    spec:
      containers:
      - command: example
        image: nginx
        name: server
        ports:
        - containerPort: 8080
          name: grpc
          protocol: TCP
```


### Notes/Constraints/Caveats (Optional)

The schema file that the user specifies in the openapi field of the 
kustomization file must include the entire OpenAPI schema document 
that kustomize needs. This may confuse users who expect this to be a 
supplemental schema file that gets merged with the builtin one. 

### Risks and Mitigations

This proposal allows the user to replace the builtin OpenAPI schema with 
their own, so the risks involved are no different than what already exists. 

## Design Details

When kustomize applies a strategic merge patch, it makes use of the kyaml 
merging libraries. First, it uses the kyaml walker code, which utilizes 
the kustomize/kyaml/openapi library. With the openapi library, it does 
two things:

1) Makes a call to `openapi.initSchema()`, which loads the builtin kubernetes
OpenAPI schema into a global schema variable for all other functions to use
2) Retrieves the patch strategy and merge keys from the schema (`openapi.GetPatchStrategyAndKeyList()`) 

Instead of loading the builtin schema, `openapi.initSchema()` should 
read the schema from a file if specified (and default to the builtin 
schema if there is no file specified). The kustomize/kyaml/openapi library 
already has a function `openapi.SchemaFromFile(path string)` that can load 
in an OpenAPI schema from a file. 

Kustomize would have to make the kustomization’s openapi/path field available 
to `openapi.initSchema()` so that it can load the schema from the specified 
file.

Additional logic would be added to handle overlays; as kustomize recursively 
processes kustomizations, the schema would have to be reinitialized at each 
step to account for potentially different schemas in each kustomization. 

The `kustomize openapi fetch` command (to fetch OpenAPI data from the user’s 
current cluster) should be implemented mostly in cli-utils so that any other 
programs that want to have a similar command can share the underlying code. 
 

### Test Plan

Unit tests will be added to demonstrate different merge behavior with 
different schema files provided via the kustomization. 

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels]. -->

#### Alpha -> Beta Graduation

- Complete features required for this functionality
- read schema from openapi field of kustomization
- kustomize openapi fetch command
- Write appropriate unit tests for the above features
- Add documentation for these features 


<!--
#### Beta -> GA Graduation

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases. -->


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


