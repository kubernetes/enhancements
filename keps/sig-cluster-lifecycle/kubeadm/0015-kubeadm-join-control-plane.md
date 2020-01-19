# kubeadm join --control-plane workflow

## Metadata

```yaml
---
title: "kubeadm join --control-plane workflow"
authors:
  - "@fabriziopandini"
owning-sig: sig-cluster-lifecycle
reviewers:
  - "@chuckha"
  - "@detiber"
  - "@luxas"
approvers:
  - "@luxas"
  - "@timothysc"
editor: "@fabriziopandini"
creation-date: 2018-01-28
last-updated: 2019-04-18
status: provisional
see-also:
  - KEP 0004
---
```

## Table of Contents

<!-- toc -->
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-goals](#non-goals)
  - [Challenges and Open Questions](#challenges-and-open-questions)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Create a cluster with more than one control plane instance (static workflow)](#create-a-cluster-with-more-than-one-control-plane-instance-static-workflow)
    - [Add a new control-plane instance (dynamic workflow)](#add-a-new-control-plane-instance-dynamic-workflow)
  - [Implementation Details](#implementation-details)
    - [Initialize the Kubernetes cluster](#initialize-the-kubernetes-cluster)
    - [Preparing for execution of kubeadm join --control-plane](#preparing-for-execution-of-kubeadm-join---control-plane)
    - [The kubeadm join --control-plane workflow](#the-kubeadm-join---control-plane-workflow)
    - [Dynamic workflow vs static workflow](#dynamic-workflow-vs-static-workflow)
    - [Strategies for deploying control plane components](#strategies-for-deploying-control-plane-components)
    - [Strategies for distributing cluster certificates](#strategies-for-distributing-cluster-certificates)
    - [<code>kubeadm upgrade</code> for HA clusters](#-for-ha-clusters)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Motivation

Support for high availability is one of the most requested features for kubeadm.

Even if, as of today, there is already the possibility to create an HA cluster
using kubeadm in combination with some scripts and/or automation tools (e.g.
[this](https://kubernetes.io/docs/setup/independent/high-availability/)), this KEP was
designed with the objective to introduce an upstream simple and reliable solution for
achieving the same goal.

Such solution will provide a consistent and repeatable base for implementing additional
capabilities like e.g. kubeadm upgrade for HA clusters.

### Goals

- "Divide and conquer”

  This proposal - at least in its initial release - does not address all the possible
  user stories for creating an highly available Kubernetes cluster, but instead
  focuses on:

  - Defining a generic and extensible flow for bootstrapping a cluster with multiple control plane instances,
    the `kubeadm join --control-plane` workflow.
  - Providing a solution *only* for well defined user stories. see
    [User Stories](#user-stories) and [Non-goals](#non-goals).

- Enable higher-level tools integration

  We expect higher-level and tooling will leverage on kubeadm for creating HA clusters;
  accordingly, the `kubeadm join --control-plane` workflow should provide support for
  the following operational practices used by higher level tools:

  - Parallel node creation

    Higher-level tools could create nodes in parallel (both nodes hosting control-plane instances and workers)
    for reducing the overall cluster startup time.
    `kubeadm join --control-plane` should support natively this practice without requiring
    the implementation of any synchronization mechanics by higher-level tools.

- Provide support both for dynamic and static bootstrap flow

  At the time a user is running `kubeadm init`, they might not know what
  the cluster setup will look like eventually. For instance, the user may start with
  only one control plane instance + n nodes, and then add further control plane instances with
  `kubeadm join --control-plane` or add more worker nodes with `kubeadm join` (in any order).
  This kind of workflow, where the user doesn’t know in advance the final layout of the control plane
  instances, into this document is referred as “dynamic bootstrap workflow”.

  Nevertheless, kubeadm should support also more “static bootstrap flow”, where a user knows
  in advance the target layout of the control plane instances (the number, the name and the IP
  of nodes hosting control plane instances).

- Support different etcd deployment scenarios, and more specifically run control plane components
  and the etcd cluster on the same machines (stacked control plane nodes) or run the etcd
  cluster on dedicated machines.

### Non-goals

- Installing a control-plane instance on an existing workers node.
  The nodes must be created as a control plane instance or as workers and then are supposed to stick to the
  assigned role for their entire life cycle.

- This proposal doesn't include a solution for API server load balancing (Nothing in this proposal
  should prevent users from choosing their preferred solution for API server load balancing).

- This proposal doesn't include support for self-hosted clusters (but nothing in this proposal should
  explicitly prevent us to reconsider this in the future as well).

- This proposal doesn't provide an automated solution for transferring the CA key and other required
  certs from one control-plane instance to the other. This is addressed in a separated KEP.
  (see KEP [Certificates copy for join --control-plane](20190122-Certificates-copy-for-kubeadm-join--control-plane.md))

- Nothing in this proposal should prevent practices that exist today.

### Challenges and Open Questions

- Keep the UX simple.

  - _What are the acceptable trade-offs between the need to have a clean and simple
    UX and the variety/complexity of possible kubernetes HA deployments?_

- Create a cluster without knowing its final layout

  Supporting a dynamic workflow implies that some information about the cluster are
  not available at init time, like e.g. the number of control plane instances, the IP of
  nodes candidates for hosting control-plane instances etc. etc.

  - _How to configure a Kubernetes cluster in order to easily adapt to future change
    of its own control plane layout like e.g. add a new control-plane instance, remove a
    control plane instance?_

  - _What are the "pivotal" cluster settings that must be defined before initializing
    the cluster?_

  - _How to combine into a single UX support for both static and dynamic bootstrap
    workflows?_

- Kubeadm limited scope of action

  - Kubeadm binary can execute actions _only_ on the machine where it is running
    e.g. it is not possible to execute actions on other nodes, to copy files across
    nodes etc.
  - During the join workflow, kubeadm can access the cluster _only_ using identities
    with limited grants, namely `system:unauthenticated` or `system:node-bootstrapper`.

- Upgradability

  - How to setup an high available cluster in order to simplify the execution
    of cluster version upgrades, both manually or with the support of `kubeadm upgrade`?_

## Proposal

### User Stories

#### Create a cluster with more than one control plane instance (static workflow)

As a kubernetes administrator, I want to create a Kubernetes cluster with more than one
control-plane instances, of which I know in advance the name and the IP.

\* A new "control plane instance" is a new kubernetes node with
`node-role.kubernetes.io/master=""` label and
`node-role.kubernetes.io/master:NoSchedule` taint; a new instance of control plane
components will be deployed on the new node; additionally, if the cluster uses local etcd mode,
and etcd is created and managed by kubeadm, a new etcd member will be
created on the joining machine as well.

#### Add a new control-plane instance (dynamic workflow)

As a kubernetes administrator, (_at any time_) I want to add a new control-plane instance* to
an existing Kubernetes cluster.

### Implementation Details

#### Initialize the Kubernetes cluster

As of today, a Kubernetes cluster should be initialized by running `kubeadm init` on a
first node, afterward referred as the bootstrap control plane.

in order to support the `kubeadm join --control-plane` workflow a new Kubernetes cluster is
expected to satisfy the following condition:

- The cluster must have a stable `controlplaneAddress` endpoint (aka the IP/DNS of the
  external load balancer)

The above condition/setting could be set by passing a configuration file to `kubeadm init`.

#### Preparing for execution of kubeadm join --control-plane

Before invoking `kubeadm join --control-plane`, the user/higher level tools
should copy control plane certificates from an existing control plane instance, e.g. the bootstrap control plane

> NB. kubeadm is limited to execute actions *only*
> in the machine where it is running, so it is not possible to copy automatically
> certificates from remote locations.
> NB. https://github.com/kubernetes/enhancements/pull/713 is porposing a possible approach
> for automatic copy of certificates across nodes

Please note that strictly speaking only ca, front-proxy-ca certificate and service account key pair
are required to be equal among all control plane instances. Accordingly:

- `kubeadm join --control-plane` will check for the mandatory certificates and fail fast if
  they are missing
- given the required certificates exists, if some/all of the other certificates are provided
  by the user as well, `kubeadm join --control-plane` will use them without further checks.
- If any other certificates are missing, `kubeadm join --control-plane` will create them.

> see "Strategies for distributing cluster certificates" paragraph for
> additional info about this step.

#### The kubeadm join --control-plane workflow

The `kubeadm join --control-plane` workflow will be implemented as an extension of the
existing `kubeadm join` flow.

`kubeadm join --control-plane` will accept an additional parameter, that is the apiserver advertise
address of the joining node; as detailed in following paragraphs, the value assigned to
this parameter depends on the user choice between a dynamic bootstrap workflow or a static
bootstrap workflow.

The updated join workflow will be the following:

1. Discovery cluster info [No changes to this step]

   > NB This step waits for a first instance of the kube-apiserver to become ready
   > (the bootstrap control plane); And thus it acts as embedded mechanism for handling the sequence
   > `kubeadm init` and `kubeadm join` actions in case of parallel node creation.

3. In case of `join --control-plane` [New step]

   1. Using the bootstrap token as identity, read the `kubeadm-config` configMap
      in `kube-system` namespace.

      > This requires to grant access to the above configMap for
      > `system:bootstrappers` group.

   2. Check if the cluster/the node is ready for joining a new control plane instance:

      a. Check if the cluster has a stable `controlplaneAddress`
      a. Checks if the mandatory certificates exists on the file system

   3. Prepare the node for hosting a control plane instance:

      a. Create missing certificates (in any).
         > please note that by creating missing certificates kubeadm can adapt seamlessly
         > to a dynamic workflow or to a static workflow (and to apiserver advertise address
         > of the joining node). see following paragraphs for additional info.

      a. Create static pod manifests for control-plane components and related kubeconfig files.

      > see "Strategies for deploying control plane components" paragraph
      > for additional info about this step.

   4. Create the admin.conf kubeconfig file

      > This operation creates an additional root certificate that enables management of the cluster
      > from the joining node and allows a simple and clean UX for the final steps of this workflow
      > (similar to the what happen for `kubeadm init`).
      > However, it is important to notice that this certificate should be treated securely
      > for avoiding to compromise the cluster.

3. Execute the kubelet TLS bootstrap process [No changes to this step]:

4. In case of `join --control-plane` [New step]

   1. In case of local etcd:

      a. Create static pod manifests for etcd

      b. Announce the new etcd member to the etcd cluster

      > Important! Those operations must be executed after kubelet is already started in order to minimize the time
      > between the new etcd member is announced and the start of the static pod running the new
      > etcd member, because during this time frame etcd gets temporary not available
      > (only when moving from 1 to 2 members in the etcd cluster).
      > From https://coreos.com/etcd/docs/latest/v2/runtime-configuration.html
      > "If you add a new member to a 1-node cluster, the cluster cannot make progress before
      > the new member starts because it needs two members as majority to agree on the consensus.
      > You will only see this behavior between the time etcdctl member add informs the cluster
      > about the new member and the new member successfully establishing a connection to the existing one."

      > Important! In order to make possible adding a new etcd member, kubeadm is changing how local etcd is deployed
      > using the `--apiserver-advertise-address` as additional `listen-client-urls`;
      > Please note that re-using the `--apiserver-advertise-address` for etcd is a trade-off that
      > allows to keep the kubeadm UX simple, considering the possible alternative require the user to
      > specify another IP address for each joining control-plane node.

      > This decision introduce also a limitation, because HA with local etcd won't be supported when the
      > user sets the `--apiserver-advertise-address` of each kube-apiserver instance, including the
      > instance on the bootstrap control plane, _equal to the `controlplaneAddress` endpoint_.
   
   2. Apply master taint and label to the node.

   3. Update the `kubeadm-config` configMap with the information about the new control plane instance.

#### Dynamic workflow vs static workflow

Defining how to manage the API server serving certificate is one of the most relevant decision
that impact how an Kubernetes cluster might change in future, because this certificate must include
the `--apiserver-advertise-address` for control plane instances.

In a static bootstrap workflow the final layout of the control plane - the number, the
name and the IP of control plane nodes - is known in advance. As a consequence a possible approach
is to add all the addresses of the control-plane nodes at `kubeadm init` time, and then
distribute the _same_ apiserver serving certificate among all the control plane instances.

This was the approach originally suggest in the kubeadm high availability guides, but this
prevents to add _unknown_ control-plane instances to the cluster.

Instead, the recommended approach suggested by this proposal is to let kubeadm take care of
the creation of _many_ API server serving certificates, one for on each node.

As described in the previous paragraph, if the apiserver serving certificate is missing,
`kubeadm join --control-plane` will generate a new certificate on the joining
control-plane; however this certificate will be “almost equal” to the certificate created on
the bootstrap control plane, because it should consider the name/address of the joining node.

As a consequence, the new apiserver serving certificate is specific to the current node (it cannot
be reused on other nodes) but this is considered an acceptable trade-offs as it allows kubeadm to
adapt seamlessly to a dynamic workflow or to a static workflow.

#### Strategies for deploying control plane components

As of today kubeadm supports two solutions for deploying control plane components:

1. Control plane deployed as static pods (current kubeadm default)
2. Self-hosted control plane (currently alpha)

As stated above, supporting for HA self-hosted control plane is non goal for this
proposal.

#### Strategies for distributing cluster certificates

As of today kubeadm supports two solutions for storing cluster certificates:

1. Cluster certificates stored on file system (current kubeadm default)
2. Cluster certificates stored in secrets (currently alpha and applicable only to
   self-hosted control plane)

There are two possible alternatives for case 1. "Cluster certificates stored on file system":

- delegate to the user/the higher level tools the responsibility to copy certificates
  from an existing node, e.g. the bootstrap control plane, to the joining node _before_
  invoking `kubeadm join --control-plane`.

- let kubeadm copy the certificates. This alternative is described in a separated KEP
  (see KEP [Certificates copy for join --control-plane](20190122-Certificates-copy-for-kubeadm-join--control-plane.md))

As stated above, supporting for Cluster certificates self-hosted control plane is a non goal
for this proposal, and the same apply to Cluster certificates stored in secrets.

#### `kubeadm upgrade` for HA clusters

The `kubeadm upgrade` workflow as of today is composed by two high level phases, upgrading the
control plane and upgrading nodes.

The above hig-level workflow will remain the same also in case of clusters with more than
one control plane instances, but with a new sub-step to be executed on secondary control-plane instances:

1. Upgrade the control plane

   1. Run `kubeadm upgrade apply` on a first control plane instance [No changes to this step]
   1. Run `kubeadm upgrade node` on secondary control-plane instances [modified to make additional
      steps in case of secondary control-plane nodes vs "simple" worker nodes]

      Nb. while this feature will be alpha, there will be a separated `kubeadm upgrade node experimental-control-plane`
      command.

1. Upgrade nodes/kubelet [No changes to this step]

## Graduation Criteria

- To create a periodic E2E tests for HA clusters creation
- To create a periodic E2E tests to ensure upgradability of HA clusters
- To document the kubeadm support for HA in kubernetes.io

## Implementation History

- original HA proposals [#1](https://goo.gl/QNtj5T) and [#2](https://goo.gl/C8V8PV)
- merged [Kubeadm HA design doc](https://goo.gl/QpD5h8)
- HA prototype [demo](https://goo.gl/2WLUUc) and [notes](https://goo.gl/NmTahy)
- [PR #58261](https://github.com/kubernetes/kubernetes/pull/58261) with the showcase implementation of the first release of this KEP
- v1.12 first implementation of `kubeadm join --control plane`
- v1.13 support for local/stacked etcd
- v1.14 implementation of automatic certificates copy
  (see KEP [Certificates copy for join --control-plane](20190122-Certificates-copy-for-kubeadm-join--control-plane.md)).

## Drawbacks

The `kubeadm join --control-plane` workflow requires that some condition are satisfied at `kubeadm init` time,
but the user will be informed about compliance/non compliance of such conditions only when running 
`kubeadm join --control plane`.

## Alternatives

1) Execute `kubeadm init` on many nodes

The approach based on execution of `kubeadm init` on each node candidate for hosting a control plane instance
was considered as well, but not chosen because it seems to have several drawbacks:

- There is no real control on parameters passed to `kubeadm init` executed on different nodes,
  and this might lead to unpredictable inconsistent configurations.
- The init sequence for above nodes won't go through the TLS bootstrap process,
  and this might be perceived as a security concern.
- The init sequence executes a lot of steps which are un-necessary (on an existing cluster); now those steps are
  mostly idempotent, so basically now no harm is done by executing them two or three times. Nevertheless, to
  maintain this contract in future could be complex.

Additionally, by having a separated `kubeadm join --control-plane` workflow instead of a single `kubeadm init`
workflow we can provide better support for:

- Steps that should be done in a slightly different way on a secondary control plane instances with respect
  to the bootstrap control plane (e.g. updating the kubeadm-config map adding info about the new control plane
  instance instead of creating a new configMap from scratch).
- Checking that the cluster/the kubeadm-config is properly configured for many control plane instances
- Blocking users trying to create secondary control plane instances on clusters with configurations
  we don't want to support as a SIG (e.g. HA with self-hosted control plane)
