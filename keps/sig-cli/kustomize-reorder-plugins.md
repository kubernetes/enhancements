---
title: Kustomize Resource Plugins Based Reodering
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
editors:
 - "@jbrette"
creation-date: 2019-09-01
last-updated: 2019-09-01
status: provisional
see-also:
replaces:
superseded-by:
 - n/a
---

# Kustomize Resource Ordering

## Table of Contents
* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
* [Proposal](#proposal)
  * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Alternatives](#alternatives)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Summary

Kustomize orders Resource creation by sorting the Resources it emits based off their type
or the order used in the original files. The user needs to be able to update that order
to account for his CRDs and the kubectl operation he intends to perform (apply, delete, patch...)

See [kubernetes-sigs/kustomize#836] for an example.
See [kubernetes-sigs/kustomize#821] for an example.

## Motivation

Users may need direct control of the Resource create / update / delete ordering in cases were
sorting by Resource Type is insufficient.

### Goals

- Provide the ability for users to override the order that Kustomize emits Resources
  - Used by `kubectl apply`
  - Used by `kubectl delete`
  
### Non-Goals

- Ensure certain Resources are *Settled* or *Ready* before other Resources
- Ensure dependencies between Workloads

## Proposal

Provide a simple mechanism allowing users to override the order that
Resource operations are applied or deleted. 

The user can currently sort the resouces using the builtin legacyOrderTransformer
(itself based on the kustomize internal GroupVersionKind class) by invoking the following
operations

```bash
kustomize build <xxx> --reorder=legacy
```

or

```bash
kustomize build <xxx>
```

The user can also choose to preserve the resources in the order in which they were loaded:

```bash
kustomize build <xxx> --reorder=none
```

The new --reorder=kubectlappy and --reorder=kubectldelete options extend what was done 
for the legacyorder by using an external transformer. This brings flexibility to the configuration 
of the transformer (for instance kindorder, as described in the example bellow) and potentailly
the implementation of the transformer.

```bash
kustomize build <xxx> --reorder=kubectlapply | kubectl apply -f -
kustomize build <xxx> --reorder=kubectldelete | kubect delete -f -
```

The above commands are equivalent to something like:

```bash
kustomize build <xxxx> | someyamlorderingscript | kubectl apply -f -
kustomize build <xxxx> | someyamlreverseorderingscript | kubectl delete -f -
```

Ultimately the goal will be, during the next integration phase of kubectl and kustomize,
to get kubectl to indicate to kustomize what operation is beeing performed in order
to choose the appropriate sorting:

```bash
kubectl apply -k <xxx>
kubectl delete -k <xxx>
```


### kubectl apply example:

The `--reorder=kubectctlapply` option indicates to kustomize to load the `kubectlapplyordertransformer.yaml transformer config` ,
itself loading a kustomize external plugin in charge of sorting and filtering kustomize build output.

An example of configuration of such a kubectapplyordertransformer.yaml is provided bellow. It demonstrates how the KubectlApplyOrderTransfer loaded by invoking --reorder=kubectlapply would ouput the MyCRD before a RoleBinding resource.

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: KubectlApplyOrderTransformer
metadata:
  name: kubectlapplyordertransformer
kindorder:
- Namespace
- CustomResourceDefinition
- ServiceAccount
- ClusterRole
- RoleBinding
- MyCRD
- ClusterRoleBinding
- ConfigMap
- Service
- Deployment
- CronJob
- ValidatingWebhookConfiguration
- APIService
- Job
- Certificate
- ClusterIssuer
- Issuer
```

A new transformer could also be implemented to filter resources before invoking `kubectl apply`. In some use cases, users were trying to selectively get kustomize to output resources piped into kubectl. A transformer as shown bellow a user to only sent ServiceAccount and Role to kubectl

```bash
kustomize build <xxx> --reorder=kubectlappy | kubectl apply -f -
```

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: KubectlApplyFilterTransformer
metadata:
  name: kubectlapplyfiltertransformer
filter:
- ServiceAccount
- Role
```

Another example could be the one based on a previous KEP:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: KubectlApplyAnnotationBasedSorter
metadata:
  name: kubectlapplyannotationbasedsorter
sortOrder:
- matchAnnotations:
    some-annotation-name-1: some-annotation-value-1
    some-annotation-name-a: some-label-value-a
- matchAnnotations:
    some-annotation-name-1: some-annotation-value-1
- matchAnnotations:
    some-annotation-name-2: some-annotation-value-2
```


### kubectl delete example:

The `--reorder=kubectctldelete` option indicates to kustomize to load a `kubectlapplyordertransformer.yaml transformer config`,
itself loading a kustomize external plugin in charge of sorting and filtering kustomize build output.

An example of configuration of such a transformer.

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: KubectlDeleteOrderTransformer
metadata:
  name: kubectldeleteordertransformer
kindorder:
- Issuer
- ClusterIssuer
- Certificate
- Job
- APIService
- ValidatingWebhookConfiguration
- CronJob
- Deployment
- Service
- ConfigMap
- ClusterRoleBinding
- RoleBinding
- ClusterRole
- ServiceAccount
- CustomResourceDefinition
- Namespace
```

Note that the KubectlDeleteOrderTransformer could share the same implementation as the KubectlApplyOrderTransformer with a "reversed" kindorder. The key concept is the ability for kustomize to locate:
- a "kubectlapplyordertransformer.yaml" file when --reorder=kubectlappy is provived
- and a "kubectdeleteordertransformer.yaml" file when --reorder=kubectldelete is provided on the command line.

### Risks and Mitigations

- Risk: Current external plugins require the presence of the module in the XDG_CONFIG_HOME.
- Mitigation: Good documentation and recommendations about how to keep things simple.

- Note: Adding a kubectlapplyordertransformer.yaml to the "transfomers:" section of the kustomization.yaml is quite impracticle
since it would require the user to add it at the end of the list of transformers in his top level kustomization.yaml, and would
require him to edit editer the kustomization.yaml or that ordertransformer.yaml depending on the kubectl command (apply|delete) he is
about to invoke.

## Graduation Criteria

Use customer feedback to determine if we should support:

- Let the user specify the transformers configuration files in the "top" kustomization.yaml. Assuming we have the usual base,
  overlay setup, there no real need to order the resources during the processing of the base layer especially if the overlay
  layer is adding name prefix, suffix or namespace and the `kubectlapplyordertransformer` is sorting based on the name of the
  resources instead.
- Need more reorder option for instance when performing other kubectl operations such as patch
- Need to automatically add the --enable-alpha-plugins flag when reorder=kubectlapply or kubectldelete is specified.
- Create a default "kubectlapplyordertransformer.yaml" if the user did not specify any in his kustomization tree.
- Need to provided builtin version of those transformers.


## Doc

Update Kubectl Book and Kustomize Documentation

## Test plan

Unit Tests for:

- [ ] Ordering resources if no kindorder is specified in the transformerconfig.yaml
- [ ] Ordering of resources matches the --reorder=kubectlapply or --reorder=kubectldelete
- [ ] kustomize behavior is identical is --reorder is not specified or --reorder=legacy or --reorder=none is specified.


### Version Skew Tests

## Implementation History

## Alternatives
