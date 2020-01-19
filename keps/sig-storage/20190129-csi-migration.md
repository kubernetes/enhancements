---
title: In-tree Storage Plugin to CSI Migration
authors:
  - "@davidz627"
  - "@jsafrane"
owning-sig: sig-storage
participating-sigs:
  - sig-architecture
  - sig-cluster-lifecycle
reviewers:
  - "@saadali"
  - "@msau42"
approvers:
  - "@saadali"
editor: "@davidz627"
creation-date: 2019-01-29
last-updated: 2019-01-29
status: implementable
see-also:
  - "https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/csi-migration.md"
---

# In-tree Storage Plugin to CSI Migration Design Doc


## Table of Contents

<!-- toc -->
- [Summary](#summary)
  - [Glossary](#glossary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha -&gt; Beta](#alpha---beta)
  - [Beta -&gt; GA](#beta---ga)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

This document presents a detailed design for migrating in-tree storage plugins
to CSI. This will be an opt-in feature turned on at cluster creation time that
will redirect in-tree plugin operations to a corresponding CSI Driver.

### Glossary

* ADC (Attach Detach Controller): Controller binary that handles Attach and Detach portion of a volume lifecycle
* Kubelet: Kubernetes component that runs on each node, it handles the Mounting and Unmounting portion of volume lifecycle
* CSI (Container Storage Interface): An RPC interface that Kubernetes uses to interface with arbitrary 3rd party storage drivers
* In-tree: Code that is compiled into native Kubernetes binaries
* Out-of-tree: Code that is not compiled into Kubernetes binaries, but can be run as Deployments on Kubernetes


## Motivation

The Kubernetes volume plugins are currently in-tree meaning all logic and
handling for each plugin lives in the Kubernetes codebase itself. With the
Container Storage Interface (CSI) the goal is to move those plugins out-of-tree.
CSI defines a standard interface for communication between the Container
Orchestrator (CO), Kubernetes in our case, and the storage plugins.

As the CSI Spec moves towards GA and more storage plugins are being created and
becoming production ready, we will want to migrate our in-tree plugin logic to
use CSI plugins instead. This is motivated by the fact that we are currently
supporting two versions of each plugin (one in-tree and one CSI), and that we
want to eventually transition all storage users to CSI.

In order to do this we need to migrate the internals of the in-tree plugins to
call out to CSI Plugins because we will be unable to deprecate the current
internal plugin API’s due to Kubernetes API deprecation policies. This will
lower cost of development as we only have to maintain one version of each
plugin, as well as ease the transition to CSI when we are able to deprecate the
internal APIs.

### Goals

* Compile all requirements for a successful transition of the in-tree plugins to
  CSI
    * As little code as possible remains in the Kubernetes Repo
    * In-tree plugin API is untouched, user Pods and PVs continue working after
      upgrades
    * Minimize user visible changes
* Design a robust mechanism for redirecting in-tree plugin usage to appropriate
  CSI drivers, while supporting seamless upgrade and downgrade between new
  Kubernetes version that uses CSI drivers for in-tree volume plugins to an old
  Kubernetes version that uses old-fashioned volume plugins without CSI.
* Design framework for migration that allows for easy interface extension by
  in-tree plugin authors to “migrate” their plugins.
    * Migration must be modular so that each plugin can have migration turned on
      and off separately

### Non-Goals

* Design a mechanism for deploying  CSI drivers on all systems so that users can
  use the current storage system the same way they do today without having to do
  extra set up.
* Implementing CSI Drivers for existing plugins
* Define set of volume plugins that should be migrated to CSI

## Proposal

### Implementation Details/Notes/Constraints
The detailed design was originally implemented as a [design proposal](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/csi-migration.md)

### Risks and Mitigations

* Performance risks as outlined in design proposal
* ADC and Kubelet synchronization fairly complicated, upgrade path non-trivial - mitigation discussed in design proposal

## Graduation Criteria

### Alpha -> Beta

* All volume operation paths covered by Migration Shim in Alpha for >= 1 quarter
* Tests outlined in design proposal implemented
* Required CRD and driver installation solved generally

### Beta -> GA

* All volume operation paths covered by Migration Shim in Beta for >= 1 quarter without significant issues

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- 2019-01-29 KEP Created
- 2019-01-05 Implementation started
