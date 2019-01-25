---
title: Kubectl Apply Prune Enhancement
authors:
  - "@Liujingfang1"
owning-sig: sig-cli
participating-sigs:
  - sig-cli
  - sig-apps
  - sig-apimachinery
reviewers:
  - "@pwittrock"
  - "@seans3"
  - "@soltysh"
approvers:
  - "@pwittrock"
  - "@seans3"
  - "@soltysh"
editor:
  - "Liujingfang1"
  - "crimsonfaith91"
creation-date: 2019-02-04
last-updated: 2019-02-14
status: provisional
see-also:
  - n/a
replaces:
  - n/a
superseded-by:
  - n/a
---

# Kubectl Apply Prune Enhancement

## Table of Contents
* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [Challenges](#challenges)
    * [Limitation of Alpha Prune](#limitation-of-alpha-prune)
    * [A Suggested Solution](#a-suggested-solution)
        * [Apply Config Object](#apply-config-object)
        * [Child Objects](#child-objects)
        * [How It Works](#how-it-works)
        * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Drawbacks](#drawbacks)
* [Alternatives](#alternatives)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

Link to tracking issue: [kubernetes/kubectl#572](https://github.com/kubernetes/kubectl/issues/572)

## Summary

`kubectl apply --prune` has been in Alpha for more than a year. This design proposes a better approach of handling `--prune` and move it to a second round of Alpha. This feature targets 1.15 release.

## Motivation

`kubectl apply` operates an aggregation of Kubernetes Objects. While `kubectl apply` is widely used in different workflows, users expect it to be able to sync the objects on the cluster with configuration saved in files, directories or repos. That means, when `kubectl apply` is run on those configurations, the corresponding objects are expected to be correctly created, updated and pruned. 

Here is an example workflow to demonstrate the meaning of syncing objects. In the directory `dir`, there are files declaring Kubernetes objects `foo`, `bar`. When the first time the directory is applied to the cluster by
```
kubectl apply -f dir
```
the objects `foo` and `bar` are created on cluster. Later, those resource files could be updated by users. Correspondingly the objects declared by those files could be updated, removed or even new objects being added. Assume the objects declared in the same directory `dir` becomes `foo`, `baz`. `bar` is deleted. After this modification, when 
```
kubectl apply --prune -f dir
```
is run, the object `foo` in the cluster is expected to be updated to match the state defined in the resource file; the object `bar` in the cluster is expected to be removed and `baz` is expected to be created.

  Currently `kubectl apply --prune` is still in Alpha. It is based on label matching. Users need to be really careful to use this command, so that they don't delete objects unintentionally. There is a list of [issues](https://github.com/kubernetes/kubectl/issues/572) observed by users. The most noticeable issue happens when changing directories or working with directories with the same labels. When two directories share the same label. `kubectl apply --prune` from one directory will delete objects from the other directory.

 In this design, we take an overview of the pruning challenges and propose a better pruning solution. This new approach is able to work with or to be utilized by generic configuration management tools.

### Goals

Propose a solution for better pruning in `kubectl apply` so that
- it can find the correct set of objects to delete
- it can avoid mess up when changing directories
- it can avoid selecting matching objects across all resources and namespaces

### Non-Goals

- Make `--prune` default on
- Improvement on Garbage Collections

## Proposal

### Challenges
The challenges of pruning are mainly from two aspects.
- Find out correct objects to delete. When a directory is applied after some updates, it loses the track of objects that are previously applied from the same directory.
- Delete objects at proper time. Assume the set of objects to be deleted is available, when to delete them is another challenge. Those objects could be still needed during rolling out other objects. One example is a ConfigMap object being referred in a Deployment object. When rolling update happens, new pods refer to new ConfigMap and old pods are still running and referring to old ConfigMap. The old ConfigMap can't be deleted until the rolling up successfully finishes.

In this design, we focus on a solution for the first challenge. For the second challenge, we can blacklist certain types such as ConfigMaps and Secrets for deletion. Generally deleting objects at proper time can be better addressed by Garbage Collection.

### limitation of Alpha prune

The Alpha prune works by label selector:
```
  # Apply the configuration in manifest.yaml that matches label app=nginx and delete all the other resources that are not in the file and match label app=nginx.
  kubectl apply --prune -f manifest.yaml -l app=nginx
```
Internally, `kubectl apply` takes following logic to perform pruning.
- For one namespace, it queries all types of resources except [some hard coded ones](https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/apply/apply.go#L632)
to find objects that matches the label. The number of queries within one namespace is about the number of API resources installed in the cluster.
- Among the matched objects, skip the ones without [last-applied-configuration annotation](https://kubernetes.io/docs/concepts/overview/object-management-kubectl/declarative-config/#how-apply-calculates-differences-and-merges-changes).
- Among the rest objects, delete any object that is not present in the new apply set.
- Repeat this for all visited namespaces. Overall, the number of queries is propotional to the number of API resources installed in the cluster.

The limitations are mainly from three aspects:
- Objects might be deleted unintentionally when different directories use the same labels
- Some types are hard-coded to avoid being pruned.
- The query number is neither constant nor linear to the number of resources being applied/pruned. 

### A suggested solution

#### Apply-Config Object
To solve the first challenge, a concept of apply-config object is proposed. An apply-config object is an object that contains the GKN information for all the objects that are to be applied. The apply-config object is not generated by kubectl when applying is running. Instead, it should be generated by other tools such as kustomize, Helm. When generating resources to be consumed by `kubectl apply`, those tools are responsible to generate the associated apply-config object to utilize better pruning.

An object is identified as an apply-config object when it has a special label `kubectl.kubernetes.io/apply-cfg: "true"`. The GKN information for all other objects are stored in annotations. Here is an apply-config object example.
```
apiVersion: v1
kind: ConfigMap
metadata:
 name: name-foo
 labels:
   kubectl.kubernetes.io/apply-cfg: "true"
 annotations:
   kubectl.kubernetes.io/last-applied-objects: |
     namespace1:group1:kind1:somename
     namespace1:group2:kind2:secondname
     namespace3:group3:kind3:thirdname
```
Note that the version is not necessary to find the object. The version of the resource can be selected by the client and available versions can be discovered dynamically

The apply-config object is not limited to ConfigMap type. Any type of Object can be an apply-config object as long as it has the `apply-cfg` label. That means a custom resource object, such as an [Application](https://github.com/kubernetes-sigs/application) custom resource object, can also be used as an apply-config object when needed.

The apply-config object could have an optional hash label `kubectl.kubernetes.io/apply-hash`, which is based on the content of annotation `kubectl.kubernetes.io/last-applied-objects`. This hash label is used for validation in kubectl. More details are in [How It Works](#how-it-works).

#### Child Objects
The objects passed to apply will have annotations pointing back to the apply-config object for the same apply.
```
 annotations:
    kubectl.kubernetes.io/apply-owner: uuid_of_the_apply_cfg_object 
```
The presence of this annotation is to confirm that a live object is safe to delete when it is to be pruned. A child object may be updated or touched by other users or apply owner, in which case the annotation will be missing or different. In this scenario, it is safe to not delete the child object. This annotation is used to prevent deleting child objects in this scenario.

#### How it works
Since the apply-config object is not generated by `kubectl`, other tools or users need to generate an apply-config object configuration before piping them into `kubectl apply`. For example, in following workflows, `kustomize` or `helm` need to generate the apply-config object. Moreover, every time those tools run, it should generate the apply-config object with the same name.
```
kustomize build . | kubectl apply -f -
# or
helm template mychart | kubectl apply -f -
```

When `kubectl apply` is run, it looks for an apply-config object by label `kubectl.kubernetes.io/apply-cfg: "true"`. If no apply-config object is found, `kubectl` will give a warning that no apply-config object is found. When an apply-config object is found, the pruning action is triggered. If the label `kubectl.kubernetes.io/apply-hash` is present in the apply-config object, `kubectl` will validate this hash against the content stored in annotation. This validation is to prevent unintentional modification of the apply-config object annotations from users. Once the validation passes, the next step is to query the live apply-config object from the cluster. Assume the apply-config object has kind `K` and name `N`, `kubectl` looks for a live object from the cluster with the same kind and name.
- If a live apply-config object is not found, Kubectl will create the apply-config object and the corresponding set of objects
- If a live apply-config object is found, then the set of live objects will be found by reading the annotation field of the live apply-config object. Kubectl diffs the live set with new set of objects and takes following steps: 
  - set the apply-config object annotation as the union of two sets and update the apply-config object
  - create new objects
  - prune old objects
  - remove the pruned objects from apply-config object’s annotation field
  - validate the `apply-hash` in apply-config object as a signal that pruning has successfully finished

  
pros:
  - The apply-config object can be any type of objects
  - No need to selecting matching objects by labels. More efficient
  - Number of queries is linear to the number of resources being applied/pruned
  
cons:
  - Users need be careful when modifying resource files so that the label `kubectl.kubernetes.io/apply-cfg: "true"` is not
    removed unintentionally.
  - Users/tools need to maintain the correct list of objects in annotations


#### Risks and Mitigations

Deciding the correct set of objects to change is hard. `--prune` Alpha tended to solve that problem in kubectl itself. In this new approach, kubectl is not going to decide such set. Instead it delegates the job to users or tools. Those who need to use `--prune` will be responsible to create an apply-config object as a record of the correct set of objects. Kubectl consumes this apply-config object and triggers the pruning action.

This approach requires users/tools to do more work than just specifying a label. which might discourage the usage of `--prune`. On the other side, existing tools such as Kustomize, Helm can utilize this approach by generating an apply-config object when expanding or generating the resources.

## Graduation Criteria
`kubectl apply` with pruning can be moved from Alpha to Beta when

- There is no severe issue with the new approach.
 
- Pruning is widely used in CI workflow with Kubernetes.

## Implementation History

The major implementation includes following steps:

- add functions to identify and validate apply-config object
- add a different code path to handle the case when apply-config object is present
- add unit test
- add e2e tests
- update `kubectl apply` document and examples

As a convenient way for users to view the intention of apply, `--dry-run` will show categorized output: 
```
$ kubectl apply -f <dir> --dry-run
Resources to add
  pod1...
  pod2...
Resources modified
  pod3...
  pod4...
Resources unmodified
  pod5...
  pod6...
Resources to delete
  pod7...     
```

## Drawbacks

- This approach is not backward compilable with Alpha `--prune`
- The apply-config object can't be auto pruned. It need manual clean up.

## Alternatives

### Using a unique identifier as label
Instead of listing all the child objects in the apply-config object, a label could be set in the apply-config object. 
```
apiVersion: v1
kind: ConfigMap
metadata:
 name: predictable-name-foo
 labels:
   kubectl.kubernetes.io/apply-confg: "true"
   kubectl.kubernetes.io/apply-id: some_uuid
```

All child objects have this label. Then `kubectl apply --prune` is run, the live set of child objects is found by label selecting and annotation matching.
```
 labels:
   kubectl.kubernetes.io/apply-id: some_uuid
 annotations:
    kubectl.kubernetes.io/apply-owner: uuid_of_the_apply_config_object
```
Any object in the live set that is not in the new apply set will be deleted.

The label set in the apply-config object should not have collisions with resources outside of the aggregation identified by the apply-config objects. This will require users or tools create a unique identifier.
- pros:
  - This extends the idea of Alpha `--prune` with an apply-config object; existing `--prune` users can easily adopt this new approach.
  - This could be used by other functionality of kubectl, such as checking if the applied objects are settled
  - This can work for Application CRD or other CRDs that with an apply-config object concept
  - No need to maintain a list of child objects and store it in the apply-config object
- cons:
  - need a unique label that doesn't collision with other resources; may need other tool to mechanism to generate a unique label
  - selecting matching objects across namespaces and types is not efficient
  - The unique identifier may not be human-readable

### Handling prune on the server side
Currently, some logic in `kubectl apply` has started to be moved to the server side. `kubectl apply` performs 3-way merge for resolving state differences before sending the state to API server. With alpha release of [server-side `apply`](https://github.com/kubernetes/enhancements/issues/555), this 3-way merge is going to be replaced. Similar to that, the pruning logic in `kubectl apply` can also be moved to the server side.

-pros:
  - All other CLIs such as Helm client to enjoy the benefits of the `prune` feature on server side
  - The server side can assign owner-reference, which helps cleaning up
  - With introduction of server-side apply for single object, it makes sense to host logic of applying operations for
    a batch of objects in the API server. Then the server side will have a complete story for applying objects.
  
-cons:
  - No capability to support working on a batch of objects. Need to have an API to represent an aggregation of objects.
  - longer development cycle than pure client side pruning. This is required due to necessity of API design and approval,
    subsequent controller design, implementation and maintenance, and possibly convoluting RBAC setup. This may take up to 2 years. On the contrary, client-side pruning is achievable within one release cycle(The proposed solution is targeting 1.15).
  - Will increase the loads of API-server
  