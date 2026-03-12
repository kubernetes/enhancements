# Mutually Exclusive Device Allocations in DRA

## Summary

This document proposes an extension to the Dynamic Resource Allocation (DRA) API to support mutually exclusive device allocation constraints. 

## Motivation
Hardware devices often support multiple partitioning or virtualization schemes that provide different trade-offs in terms of isolation, performance, and resource sharing. However, these schemes are frequently mutually exclusive at the hardware level—once a physical device is partitioned or configured using one scheme, it cannot be reconfigured to use a different scheme until all existing allocations are released.

### Goals
- Allow DRA drivers to specify compatibility between virtual devices within a single physical device
- Allow the scheduler to make informed allocation decisions that respect compatibility rules
- Provide a generic mechanism applicable to any hardware with partitioning constraints
- Maintain backward compatibility with existing ResourceSlice specifications

### Non Goals
- Allow DRA drivers to specify compatibility between physical/virtual devices across different phisical devices or device classes

### Problem Statement

The current Partitionable Devices API does not provide a mechanism to express mutual exclusivity constraints between devices. Without this capability:

1. **Late Failure Detection**: Incompatible allocations are only detected during resource preparation (after scheduling decisions are made)
2. **Scheduler Unawareness**: The scheduler may allocate incompatible devices, leading to pod startup failures
3. **Poor User Experience**: Users receive cryptic preparation failures instead of clear scheduling feedback
4. **Resource Thrashing**: The scheduler may repeatedly attempt incompatible allocations

**Current Workaround Limitations:**

DRA drivers must fail resource preparation when incompatible allocations are attempted.


### Use Case
**Generic Example:**

Consider a physical accelerator device that supports four distinct operational modes, in all cases, **Partitionable Devices** is utilized:

1. **Exclusive Mode**: The entire physical device allocated to a single consumer
2. **Software-Partitioned Mode A** : Multiple consumers share the physical device through virtual devices
3. **Software-Partitioned Mode B** : Multiple consumers share the physical device through virtual devices
4. **Hardware-Partitioned Mode**: The device is divided into distinct isolated hardware partitions

These modes have compatibility constraints:
- **Exclusive Mode** is incompatible with all other modes
- **Software-Partitioned Mode A** may be compatible with **Software-Partitioned Mode B**, but not with exclusive and hardware partitioning modes
- **Hardware-Partitioned Mode** creates fixed partitions that cannot coexist with other partitioning modes

The constraint is bidirectional and transitive: if partition mode A excludes partition mode B, then allocating A must prevent B from being allocated, and vice versa.

#### GPU Example
```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
...
spec:
  sharedCounters:
    # This counter set represents a specific physical device.
    - name: gpu-0-cs
      counters:
        multiprocessors:
          value: "152"

  devices:
    # Incompatible with any other partitioning schemes
    - name: gpu-0
      bindsToNode: true
      consumesCounters:
      - counterSet: gpu-0-cs
        counters:
          multiprocessors:
            value: "152"
      attributes:
        partitioningMode:
          string: None

    # Incompatible with MIGSlicing and None partitioning modes,
    # but compatible with MPSSharing mode
    - name: gpu-0-fraction-0
      bindsToNode: true
      allowMultipleAllocations: true
      consumesCounters:
      - counterSet: gpu-0-cs
        counters:
          multiprocessors:
            value: "76"
      capacity:
        ...
      attributes:
        partitioningMode:
          string: GPUFractioning
    
    # Incompatible with any other partitioning modes, only compatible with devices 
    # partitioned with the same mode (MIGSlicing)
    - name: gpu-0-mig-1g.5gb-0 
      bindsToNode: true
      consumesCounters:
      - counterSet: gpu-0-cs
        counters:
          multiprocessors:
            value: "2"
      attributes:
        partitioningMode:
          string: MIGSlicing
    
    # Incompatible with any other partitioning modes, only compatible with devices 
    # partitioned with the same mode (MIGSlicing)
    - name: gpu-0-mig-1g.5gb-1 
      bindsToNode: true
      consumesCounters:
      - counterSet: gpu-0-cs
        counters:
          multiprocessors:
            value: "2"
      attributes:
        partitioningMode:
          string: MIGSlicing
  
    # Incompatible with the MIGSlicing and None
    # partitioning modes, but compatible with GPUFractioning
    - name: gpu-0-mps-0 
      bindsToNode: true
      allowMultipleAllocations: true
      consumesCounters:
      - counterSet: gpu-0-cs
        counters:
          multiprocessors:
            value: "15"
      capacity:
        ...
      attributes:
        partitioningMode:
          string: MPSSharing
```

## Proposal 1 - CompatibilityGroups Assignment

### API Changes

Add the `device.consumesCounters[].compatibilityGroups` field which specifies which device groups this device is compatible with.
Other devices must specify at least one `compatibilityGroup` from this list to be considered compatible. 

#### Field Structure

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
...
spec:
  sharedCounters:
    # This counter set represents a specific, physical device.
    - name: gpu-1-cs
      counters:
        multiprocessors:
          value: "152"
  devices:
    # Full, physical device. Consumes full counter set `gpu-1-cs`.
    - name: gpu-1
      attributes:
        type:
          string: gpu
      consumesCounters:
        - counterSet: gpu-1-cs
          counters:
            multiprocessors:
              value: "152"

    # MIG partition. This cannot be allocated
    # - when device `gpu-1` is allocated
    #   (reason: counters exhausted)
    # - when device `gpu-1-foo-part` is allocated
    #   (reason: mismatching compatibilityGroups)
    - name: gpu-1-mig1
      attributes:
        type:
          string: mig
      consumesCounters:
        - counterSet: gpu-1-cs
          # Can only consume from the same counter set when
          # all existing consumers also list compatibilityGroup "mig".
          compatibilityGroups:
            - mig
          counters:
            multiprocessors:
              value: "2"


    # FOO partition. This cannot be allocated
    # - when device `gpu-1` is allocated
    #   (reason: counters exhausted).
    # - when device `gpu-1-mig1` is allocated
    #   (reason: mismatching compatibilityGroups).
    #
    # This can generally still be allocated
    # - when `gpu-1-bar-part` is allocated
    #  (reason: shared compatibilityGroups "bar").
    #
    # The relationship between the foo and bar type
    # partitions on the same physical device is
    # modeled by counter consumption.
    - name: gpu-1-foo-part
      attributes:
        type:
          string: foo
      consumesCounters:
        - counterSet: gpu-1-cs
          compatibilityGroups:
            - foo
            - bar
          counters:
            multiprocessors:
              value: "17"

    # BAR paritition. Similar considerations as
    # described for FOO partition.
    - name: gpu-1-bar-part
      attributes:
        type:
          string: bar
      consumesCounters:
        - counterSet: gpu-1-cs
          compatibilityGroups:
            - bar
          counters:
            multiprocessors:
              value: "2"
```

### Semantics

#### Device Groupings

1. **Group Declaration**: Devices must declare which groups they are compatible with, otherwise they are assumed compatible with all groups.

3. **Scope**: Grouping rules apply:
   - To all devices within a device class, that specify `compatibilityGroups`
   - Across all resource claims

4. **Scheduler Enforcement**: The scheduler must:
   - Evaluate exclusion constraints during device selection
   - Skip device candidates that would violate existing allocations
   - Track allocated devices and their exclusion rules

## Proposal 2 - Attribute-based Compatibility with CEL

### API Changes

Add an optional `compatibleOnlyWith` field to device objects within the ResourceSlice specification. This field allows devices to declare which other devices can be allocated alongside them. 
If not provided, a device is deemed compatible with all other devices to preserve backwards compatibility

#### Field Structure

```yaml
devices:
- name: device-name
  # ... existing device fields ...

  # New field: compatibleOnlyWith
  # Specifies a CEL expression that the scheduler filters devices with when attempting
  # a device allocation.
  # This field is optional. If not specified, the device has no compatibility constraints.
  compatibleOnlyWith:
    expression: "cel exp"
```

### Semantics

#### Exclusion Rules

1. **Mutual Exclusivity**: If device A specifies a compatibility expression, scheduler must:
  - Evaluate the expression against already allocated devices when the device is considered for allocation
  - Evaluate the expression against devices that are considered for allocation if a device with an expression is already allocated

3. **Scope**: Compatibility expressions apply:
   - To all devices within a device class
   - Across all resource claims

#### Example Exclusion Patterns

**Pattern 1: Device-Level Exclusivity**
```yaml
- name: device-full
  attributes:
    physicalDevice: dev-0
  # Excludes all devices whos underlying device is dev-0
  compatibleOnlyWith:
    expression: 'device.attributes["device.example.com"].physicalDevice != "dev-0"'
```

**Pattern 2: Mode-Based Exclusivity**
```yaml
- name: dev-0-partition-1
  attributes:
    physicalDevice: dev-0
    mode: hardware-partitioned
  # Only compatible with specific paritioning modes
  compatibleOnlyWith:
    expression: 'device.attributes["device.example.com"].mode == "hardware-partitioned"'
```

## Proposal Comparison
**Attribute-based Compatibility with CEL**
- **Higher degree of freedom**:
  - Device compatibility can be defined in a multi-dimentional way, not only physical device placement
  - Can be extended to support additional use cases in the future (maybe across device-classes?)

**CompatibilityGroups Assignment**
- **Cleaner and simpler implementation** - Minimal additions to the API and codebase that solve the problem at hand

## Implementation Considerations

### Scheduler Changes

The DRA scheduler plugin must be enhanced to:

1. **Track Allocated Devices**: Maintain a cache of allocated devices per node with their attributes and compatibility expressions, or group mapping
2. **Evaluate Exclusions**: For each candidate device:
   - Check if all allocated devices are copmatible with this candidate
   - Check if this candidate is compatible with all allocated devices
3. **Filter Candidates**: Remove devices from consideration if they violate compatibility constraints
4. **Handle Allocation Failures**: If an incompatible device is allocated, provide clear feedback in scheduling events

### Driver Responsibilities

Resource drivers should:

1. **Declare Constraints**: Populate `compatibleOnlyWith` or `compatibilityGroups` for all devices with compatibility requirements
2. **Validation**: Ensure compatibility rules are symmetric and consistent across devices
4. **Documentation**: Document their compatibility matrix

### Backward Compatibility

- Both approaches are opt-in
- Devices without `compatibleOnlyWith` or `compatibilityGroups` behave identically to current behavior
- No changes to existing API fields or semantics
- Older schedulers will ignore the new field but may allocate incompatible devices (same as current behavior)
