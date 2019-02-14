---
kep-number: 0
title: Custom Resource Definitions
authors:
  - "@brendandburns"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-architecture
reviewers:
  - "@bgrant0607"
  - "@smarterclayton"
  - "@lavalamp"
  - "@deads2k"
approvers:
  - "@lavalamp"
  - "@smarterclayton"
  - TBD
editor: TBD
creation-date: "2013-02-13"
last-updated: "2013-02-13"
status: implemented
---

# Custom Resource Definitions

## Summary

This KEP covers the addition and usage of Custom Resource Definitions. This KEP is different than most KEPs
because it is an attempt to retrospectively describe the development of a feature in Kubernetes that 
already exists. We are doing this because custom resource definitions were created and evolved before the
KEP process was put in place. Because of this, there is no single place that describes the reasons for
custom resource definitions and how they should be used in Kubernetes.

## Motivation

The primary motivation for custom resources was to provide a simple easy to use extension mechanism
for Kubernetes clusters to enable customization by Kubernetes users and to facilitate a rich
ecosystem of plugins for Kubernetes clusters. If Kubernetes is the kernel, CustomResourceDefinitions
are intended to be loadable kernel modules.

### Goals

   * Provide an easy to use mechanism for the dynamic addition of new API types to Kubernetes.
   * Enable API extensibility without significant additional load on operators of Kubernetes clusters.
   * Enable an ecosystem of plugin providers who can produce and distribute extensions that add value to end-user clusters.

### Non-Goals
   * Replace all existing built-in API objects with Custom Resources.
   * Enable every possible extension API. Unsupported patterns should use aggregated API servers.
   * Develop an arbitrary key-value store for applications. Custom resources are control-plane objects.
   * Develop tooling for managing the installation ordering or dependencies of custom resources. This is deferred to other KEPs.
   * Expressing or managing inheritance, referential integrity, or other inter-object relationships.
   * Document how to create a custom resource definition. This is done [elsewhere](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/)


## Implementation History

Third-party resources were originally [proposed](https://github.com/kubernetes/kubernetes/pull/11781) 
shortly after Kubernetes 1.0. At the time it was clear that making the Kubernetes API server
extensible was critical to the long-term health of the project and there were two different
approaches for extension proposed:
   * Plugin resources (initial "ThirdPartyResource" eventually "CustomResourceDefinitions")
   * Delgation to an "aggregated API server"

ThirdPartyResources were added to the API server, but the API server was actually really not ready
for them, much of the dynamic discovery and grouping mechanisms were primitive, or non-existent.
As a result, and because there was little initial support from the core API machinery maintainers
for ThirdPartyResources, the feature languished with bugs, inconsistencies and missing functionality.

Despite this, ThirdPartyResources gained popularity, especially with the development of the Operator
pattern by CoreOS. This usage lead to many bug fixes and proving out the model as a viable method for
developing a rich ecosystem of APIs.

When it became clear that ThirdPartyResources were not only viable, but actually the predominant way
to extend the Kubernetes API Server. Core API Machinery maintainers renamed ThirdPartyResources to
CustomResourceDefinitions and rewrote most of the core code to improve it's compatability and
integration with a significantly more mature API server.

At this point, the core CustomResource mechanisms have matured to a point where there is discussion
about how to build out core required APIs as custom resources (see below).

## Proposal

### User Stories

#### Story 1: Easy extensibility

Alice is a developer who runs her own Kubernetes cluster. Alice mostly deploys Java applications. One
day Alice thinks: "Instead of always using Kubernetes `Deployment` objects, I'd like to create a `JavaDeployment` object that abstracts lots of the boilerplate of the `PodTemplate` inside the `Deployment`
object into a `JavaPodTemplate` that knows how to run Java applications.

Alice uses CustomResourceDefinitions and a simple controller and extends her cluster to have 
`JavaDeployments` in a single afternoon.

#### Story 2: Lightweight cluster add-ons

Jane is a cluster adminstrator. Here users are hopeless at deploying and managing certificates as
Secrets in Kubernetes. Jane wants to add a `RotatingSecret` which uses the API Server's signing 
capabilities to automatically rotate certificates for her users. Jane fines an open source project
that has this capability, and Jane installs the extension into the clusters she administers without
worrying about extra storage components that she needs to monitor and manage.

#### Story 3: An ecosystem of plugin providers.

Julia is running a startup. She wants to make it easy for the world to perform XSS testing against
their websites. Julia builds a custom piece of software that does XSS attacks, and also builds a
`XSSDetector` custom resource. Julia can build her business based on people installing her software
as an extension inside of their Kubernetes clusters no matter where they are running.

### Implementation details

For details of how to use `CustomResourceDefinitions` there is excellent documentation [elsewhere](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/).

### Patterns for Custom Resources

There are three main patterns for custom resources, ordered in complexity from simplest to most complex:
   * Dumb data objects
   * Compiler patterns
   * Operator patterns

#### Dumb data objects

Dumb data objects are the simplest use case for CustomResourceDefinitions. In this case, the resource
is simply used for CRUD operations. A user might create an object with `kubectl` and an application
might read that object in when it starts and periodically resynchronize that data during it's operation.
The object is used only for data storage, there is no control-plane machinery running in the cluster.

#### Compiler Patterns

The next, slightly more complicated use case for CustomResourceDefinitions is a "compiler" pattern.
In this usage, the CustomResource is an abstraction that "compiles" down to lower level Kubernetes
primitives. In this pattern there is a "compiler" process that is running in the cluster, watching
for new resources to be created and "compiling" them into Kubernetes primitives, and likewise
looking for modifications and deletes and reacting accordingly, but beyond this compiler, there is
no control loop running to dynamically manage the objects created from the CustomResourceDefinition.
It is assumed that lower level Kubernetes objects (e.g. `ReplicaSet`) are used to achieve self-healing
behaviors if necessary.

#### Operator Patterns

The operator pattern is well documented in a variety of places. It extends from the compiler pattern
to include online maintainence and repair of the resources created by the CustomResource. Operators are processes that watch their respective custom resources and take online actions to ensure the health of the custom resource. An example
of such repair might be a `Database` custom resource. It follows the compiler pattern by producing a
`StatefulSet` to deploy the database, but it extends that pattern to become an operator, because it
also performs periodic backups of the database to cloud storage, and it can recover from those
backups if all Pods in the StatefulSet fail. 

### Open Questions

#### Dependency management

If truly core APIs are developed as custom resources, then there can be dependency issues when
bootstrapping a Kubernetes cluster. If core components (e.g. the controller manager) needs to manage
a custom resource, how do you ensure that the resource gets created before the controller-manager
starts, when the controller-manager itself is in the lifecycle for custom resources. We need to untangle
these circular dependencies. And ensure that people who write such controllers, write them in such a
way that they can temporarily handle the resources they manage not existing.

Another significant concern in dependency management is dependencies between custom resource definitions.
As the ecosystem of plugins becomes richer, the chances that one plugin will depend on another increases.
Currently there is no good way to express that in order for CustomResource `A` to work correctly, 
CustomResource `B` needs to be installed as well.

#### Version upgrades

Another open question for CustomResources is version management. Currently there are no built-in practices
for moving from one version of a CustomResource to another. To do such a migration, the tool itself would
have to perform the schema migration by reading all of the data objects in the old version and rewriting
them in the new version. To make this successful , at the very least we will need to provide the ability to
mark a CustomResource as "read-only" to ensure that there isn't skew between the two representations.

We may need to invest significantly more heavily in this space to make it easy for users to upgrade or 
update their objects. For now, most users have simply chosen not to upgrade the version.

### Risks and Mitigations

#### DDOS or other attacks on storage

Because custom resources share the same storage as the core API types, there is risk that either via 
requests/second or storage space usage, custom resources will become deliberate or inadvertent denial
of service attacks on a cluster. Currently there is a maximum size cap for custom resources, but it is
a pretty limited defense against these sorts of concerns. This makes the installation of a custom
resource a very privileged action for a cluster.

#### Custom resource adminstration

For many reasons, installing a new custom resource is a privileged action for a cluster adminstrator.
However, this significantly increases the complexity of developing a Custom Resource. Ideally the
resource itself would install itself into the cluster when it's controller is run as a Pod, but
because you don't want to grant the controller the general ability to create arbitrary CustomResources,
it is generally the case that the controller won't have proper permissions to create a resource. 

In the future we may want to develop some sort of "single use token" or some equivalent such that custom resources are easy to install, and yet don't require cluster adminstrator privileges to bootstrap themselves.

## Graduation Criteria

This work has already graduated.

## Drawbacks

To become full-fledged API objects CustomResources require extensive usage of webhooks for various API 
object lifecycle events (e.g. validation, defaulting, etc). This pattern means that the code for an custom 
resource can be splattered around in various callbacks and as such can be harder to understand.

Likewise, all types of APIs may not be implementable (or easily implementable) as custom resources because
of this limitation.

## Alternatives

Aggregated API Servers where requests are delegated from the API server to an delegate API server
are a more flexible (though more complicated) alternative to custom resource definitions.

