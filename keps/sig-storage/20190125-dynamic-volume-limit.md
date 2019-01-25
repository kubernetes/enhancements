---
kep-number: 0
title: Dynamic attached volume limits
authors:
  - "@gnufied
owning-sig: sig-storage
participating-sigs:
  - sig-storage
  - sig-scheduling
reviewers:
  - @bsalamat
  - @saad-ali
approvers:
  - @bsalamat
  - @saad-ali
editor: TBD
creation-date: 2018-08-15
last-updated: 2019-01-25
status: implemented
see-also:
  - n/a
replaces:
  - n/a
superseded-by:
  - n/a
---

# Title

Dynamic attached volume limits

## Table of Contents


## Summary

Number of volumes of certain type that can be attached to a node should be configurable easily and should be based on node type.

This proposal implements dynamic attachable volume limits on a per-node basis rather than cluster global defaults that exist today. This
proposal also implements a way of configuring volume limits for CSI volumes.


## Motivation

Currently the number of volumes attachable to a node is either hard-coded or only configurable via an environment variable. Also existing limits only apply to well known volume types like EBS, GCE and is not available to all volume plugins.

This proposal enables any volume plugin to specify those limits and also allows same volume type to have different volume limits depending on type of node.

### Goals

List the specific goals of the KEP.
How will we know that this has succeeded?

### Non-Goals

What is out of scope for his KEP?
Listing non-goals helps to focus discussion and make progress.

## Proposal

#### Prerequisite

* 1.11 This feature will be protected by an alpha feature gate, so as API and CLI changes needed for it. We are planning to call
  the feature `AttachVolumeLimit`.
* 1.12 This feature will be behind a beta feature gate and enabled by default.

#### API Changes

There is no API change needed for this feature. However existing `node.Status.Capacity` and `node.Status.Allocatable` will
be extended to cover volume limits available on the node too.

The key name that will store volume will be start with prefix `attachable-volumes-`. The volume limit key will respect
format restrictions applied to Kubernetes Resource names. Volume limit key for existing plugins might look like:


* `attachable-volumes-aws-ebs`
* `attachable-volumes-gce-pd`

`IsScalarResourceName` check will be extended to cover storage limits:

```go
func IsStorageAttachLimit(name v1.ResourceName) bool {
    return strings.HasPrefix(string(name), v1.ResourceStoragePrefix)
}

// Extended and Hugepages resources
func IsScalarResourceName(name v1.ResourceName) bool {
    return IsExtendedResourceName(name) || IsHugePageResourceName(name) ||
        IsPrefixedNativeResource(name) || IsStorageAttachLimit(name)
}
```

The prefix `storage-attach-limits-*` can not be used as a resource in pods, because it does not adhere to specs defined in following function:


```go
func IsStandardContainerResourceName(str string) bool {
    return standardContainerResources.Has(str) || IsHugePageResourceName(core.ResourceName(str))
}
```

Additional validation tests will be added to make sure we don't accidentally break this.

#### Alternative to using "storage-" prefix
We also considered using currently defined `GetPluginName` interface(of Volume Plugins) for using as key in the `node.Status.Capacity`. Ultimately
we decided against using it, because most in-tree plugins start with `kubernetes.io/` and we needed a uniform way to identify storage
related capacity limits in `node.Status`.

#### Changes to scheduler

Scheduler will retrieve available attachable limit on a node from `node.Status.Allocatable` and store it in `nodeInfo` cache. Volume
limits will be treated like any other scalar resource.

For `AWS-EBS`, `AzureDisk` and `GCE-PD` volume types, existing `MaxPD*` predicates will be updated to use volume attach limits available
from node's allocatable property. To be backward compatible - the scheduler will fallback to older logic, if no limit is set in `node.Status.Allocatable` for AWS, GCE and Azure volume types.

#### Setting of limit for existing in-tree volume plugins

The volume limit for existing volume plugins will be set by querying the volume plugin. Following function
will be added to volume plugin interface:

```go
type VolumePluginWithAttachLimits interface {
    // Return key name that is used for storing volume limits inside node Capacity
    // must start with storage- prefix
    VolumeLimitKey(spec *Spec) string
    // Return volume limits for plugin
    GetVolumeLimits() (map[string]int64, error)
}
```

When querying the plugin - plugin will use `ProviderName` function of CloudProvider to check
if plugin is usable on the node. For example - querying for `GetVolumeLimits` from `aws-ebs` plugin with `gce` cloudprovider
will result in error.

Kubelet will query the volume plugins inside `kubelet.initialNode` function and populate `node.Status` with returned values.

For GCE and AWS - `GetVolumeLimits` will return limits depending on node type. Plugin already has node name accessible
via `VolumeHost` interface and hence it will check the node type and return the volume limits.

We do not aim to cover all in-tree volume types. We will support dynamic volume limits proposed here for following volume types:

* GCE-PD
* AWS-EBS
* AzureDisk

We expect to add incremental support for other volume types.

#### Changes for Kubernetes 1.12

For Kubernetes 1.12, we are adding support for CSI and moving the feature to beta.

#### CSI support

A new function will be added to `pkg/volume/util/attach_limit.go` which will return CSI attach limit
resource name.

The interface of function will be:

```go
const (
    // CSI attach prefix
    CSIAttachLimitPrefix = "attachable-volumes-csi-"

    // Resource Name length
    ResourceNameLengthLimit = 63
)

func GetCSIAttachLimitKey(driverName string) string {
    csiPrefixLength := len(CSIAttachLimitPrefix)
    totalkeyLength := csiPrefixLength + len(driverName)
    if totalkeyLength >= ResourceNameLengthLimit {
        charsFromDriverName := driverName[:23]
        // compute SHA1 of driverName and get first 16 chars
        return CSIAttachLimitPrefix + charsFromDriverName + hashed

    }
    return CSIAttachLimitPrefix + driverName
}
```

This function will be used both on node and scheduler for determining CSI attach limit key.The value of the
limit will be retrieved using `GetNodeInfo` CSI RPC call and set if non-zero.

**Other options**

Alternately we also considered storing attach limit resource name in `CSIDriver` introduced as part
of https://github.com/kubernetes/community/pull/2514 proposal.

This will work but depends on acceptance of proposal. We can always migrate attach limit resource names to
values defined in `CSIDriver` object in later release. If `CSIDriver` object is available and has a attach limit key,
then kubelet could use that key otherwise it will fallback to `GetCSIAttachLimitKey`.

Scheduler can also check presence of `CSIDriver` object and corresponding key in node object, otherwise it will
fallback to using `GetCSIAttachLimitKey` function.

##### Changes to scheduler

To support attachable limit for CSI, a new predicate called `CSIMaxVolumeLimitChecker` will be added. It will use `GetCSIAttachLimitKey`
function defined above for extracting attach limit resource name.

The predicate will be NOOP if feature gate is not enabled or when attachable limits are not available from node object.

Handling delayed binding is out of scope for this proposal and will be fixed in delayed binding and topology aware dynamic
provisioning.


This is where we get down to the nitty gritty of what the proposal actually is.

### User Stories [optional]

Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of the system.
The goal here is to make this feel real for users without getting bogged down.

#### Story 1

#### Story 2

### Implementation Details/Notes/Constraints [optional]

What are the caveats to the implementation?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they releate.

### Risks and Mitigations

What are the risks of this proposal and how do we mitigate.
Think broadly.
For example, consider both security and how this will impact the larger kubernetes ecosystem.

## Graduation Criteria

How will we know that this has succeeded?
Gathering user feedback is crucial for building high quality experiences and SIGs have the important responsibility of setting milestones for stability and completeness.
Hopefully the content previously contained in [umbrella issues][] will be tracked in the `Graduation Criteria` section.

[umbrella issues]: https://github.com/kubernetes/kubernetes/issues/42752

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks [optional]

Why should this KEP _not_ be implemented.

## Alternatives [optional]

Similar to the `Drawbacks` section the `Alternatives` section is used to highlight and record other possible approaches to delivering the value proposed by a KEP.

## Infrastructure Needed [optional]

Use this section if you need things from the project/SIG.
Examples include a new subproject, repos requested, github details.
Listing these here allows a SIG to get the process for these resources started right away.
