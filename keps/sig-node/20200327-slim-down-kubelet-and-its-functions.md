---
title: Slim Down Kubelet and its functions
authors:
  - "@gongguan"
  - TBD
owning-sig: sig-node
reviewers:
  - "@dashpole"
  - "@derekwaynecarr"
approvers:
  - "@dchen1107"
  - "@derekwaynecarr"
creation-date: 2020-03-27
last-updated: 2020-04-17
status: provisional
---

# Slim Down Kubelet and its functions

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Pros](#pros)
  - [Cons](#cons)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Better modularize Kubelet](#better-modularize-kubelet)
  - [Slim down Kubelet functions](#slim-down-kubelet-functions)
  - [Test Migration](#test-migration)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Current kubelet root directory is too large to extend and components dependency blurry.
In current Kubelet, all the members are put into it frankly. We should split it up by better modularized.
Kubelet was injected to some components like `kubeGenericRuntimeManager`, `resourceAnalyzer` and `VolumePluginMgr` which caused many methods implemented in Kubelet struct. We'd better converge these functions to its sub components.

## Motivation

Better modularize Kubelet and converge many Kubelet functions to its sub components which slim down both Kubelet and its functions.

### Pros
- Make kubelet(very huge now) better Modularized.
- Slim down Kubelet functions by returning many functions to kubeGenericRuntimeManager or resourceAnalyzer.
- Resolve some circular dependencies like kubeGenericRuntimeManager and volumeManager, StatsProvider and ResourceAnalyzer.

Kubelet components dependency states and migration plan will be listed in Proposal.

### Cons
I think this migration with low risk, because functions won't be changed during migration.

### Goals
- Not inject Kubelet to its components' initialization which makes components dependencies clear.
- Slim down Kubelet structure and its functions.

### Non-Goals
- Any effect on Kubelet current function.

## Proposal

### Better modularize Kubelet

Create `NodeInfo`(pkg/kubelet/nodeinfo) and `DirectoryManager`(pkg/kubelet/directorymanager) only contains basic members which often used by kubelet components.

```go
// NodeInfo's members are migrated from Kubelet.
type nodeInfo struct {
    hostname string

    nodeName types.NodeName

    clock clock.Clock

    registerNode bool

    registrationCompleted bool

    kubeClient clientset.Interface

    masterServiceNamespace string

    nodeLabels map[string]string

    recorder record.EventRecorder

    serviceLister serviceLister

    nodeLister corelisters.NodeLister

    providerID string

    externalCloudProvider bool

    cloud cloudprovider.Interface

    registerSchedulable bool

    registerWithTaints []api.Taint

    enableControllerAttachDetach bool

    experimentalHostUserNamespaceDefaulting bool

    nodeStatusFuncs []func(*v1.Node) error

    keepTerminatedPodVolumes bool // DEPRECATED
}

type NodeInfoProvider interface {
    // current kubelet's getHostIPAnyWay() function.
    GetHostIPAnyWay() (net.IP, error)

    // current kubelet's getNodeAnyWay function.
    GetNodeAnyWay() (*v1.Node, error)

    // current kubelet's GetNode function.
    GetNode()

    // current kubelet's initialNode function.
    InitialNode(ctx context.Context) (*v1.Node, error)

    // current kubelet.setNodeStatus.
    SetNodeStatus(node *v1.Node)

    // get nodeInfo's member such as hostname, kubeClient, etc since them are private.
    GetFoo() Foo
}

type directoryManager struct {
    rootDirectory   string
}

type DirectoryProvider interface {
    // following get dir methods locate in kubelet_getters.go now belongs to Kubelet currently.
    GetRootDir() string
    SetRootDir(dir string)
    GetPodsDir() string
    GetPluginsDir() string
    GetPluginsRegistrationDir() string
    GetPluginDir(pluginName string) string
    GetVolumeDevicePluginsDir() string
    GetVolumeDevicePluginDir(pluginName string)
    GetPodDir(podUID types.UID) string
    GetPodVolumeSubpathsDir(podUID types.UID) string
    GetPodVolumesDir(podUID types.UID) string
    GetPodVolumeDir(podUID types.UID, pluginName string, volumeName string) string
    GetPodVolumeDevicesDir(podUID types.UID) string
    GetPodVolumeDeviceDir(podUID types.UID, pluginName string) string
    GetPodPluginsDir(podUID types.UID) string
    GetPodPluginDir(podUID types.UID, pluginName string) string
    GetPodContainerDir(podUID types.UID, ctrName string) string
    GetPodResourcesDir() string
}
```
Inject `Kubelet.NodeInfo` to Kubele's component and remove these members from `Kubelet`.

### Slim down Kubelet functions

For `kubeGenericRuntimeManager` initialization.
Current kubeGenericRuntimeManager use `kubelet` as `runtimeHelper` and `podStateProvider` now. Make dependencies clear and return these functions to kubeGenericRuntimeManager.

Changes of `kubeGenericRuntimeManager` and `containerGC`:
```go
type kubeGenericRuntimeManager struct {
    // Remove.
    // Kubelet as runtimeHelper now.
    runtimeHelper kubecontainer.RuntimeHelper

    // Add.
    containerManager cm.ContainerManager

    // Add.
    dnsConfigurer *dns.Configurer

    // Add.
    configMapManager configmap.Manager

    // Add.
    secretManager secret.Manager

    // Add.
    volumeManager volumemanager.VolumeManager

    // Add.
    nodeInfo *nodeinfo.NodeInfo

    // Add.
    hostutil hostutil.HostUtils

    // Add.
    subpather subpath.Interface
}

type containerGC struct {
    // Remove.
    // Kubelet as runtimeHelper now.
    podStateProvider podStateProvider

    // Add.
    podManager kubepod.Manager

    // Add.
    statusManager status.Manager
}
```
No circular dependencies between added members and `kubeGenericRuntimeManager`.

Migrate following method from `Kubelet` to `KubeGenericRuntime`:
```go
    // Following methods belongs to KubeGenericRuntime.
    GenerateRunContainerOptions(pod *v1.Pod, container *v1.Container, podIP string, podIPs []string) (*kubecontainer.RunContainerOptions, func(), error)

    GeneratePodHostNameAndDomain(pod *v1.Pod) (string, string, error)

    GetPodCgroupParent(pod *v1.Pod) string

    GetExtraSupplementalGroupsForPod(pod *v1.Pod) []int64

    GetPodDNS(pod *v1.Pod) (*runtimeapi.DNSConfig, error)

    // And other helper functions of above Get Method(some haven't been listed).
    makeBlockVolumes(pod *v1.Pod, container *v1.Container, podVolumes kubecontainer.VolumeMap, blkutil volumepathhandler.BlockVolumePathHandler) ([]kubecontainer.DeviceInfo, error)

    makeEnvironmentVariables(pod *v1.Pod, container *v1.Container, podIP string, podIPs []string) ([]kubecontainer.EnvVar, error)

    podFieldSelectorRuntimeValue(fs *v1.ObjectFieldSelector, pod *v1.Pod, podIP string, podIPs []string) (string, error)

    ...

    // Following methods belongs to containerGC.
    IsPodDeleted(uid types.UID) bool

    IsPodTerminated(uid types.UID) bool
```

For `ResourceAnalyzer` initialization.
Current `ResourceAnalyzer` use `kubelet` to initialize `fsResourceAnalyzer` and `summaryProviderImpl` same as `kubeGenericRuntimeManager`.

Resolve circular dependencies between `StatsProvider`(pkg/kubelet/stats) and `ResourceAnalyzer`(pkg/kubelet/server/stats) at first:
`StatsProvider` only use `fsResourceAnalyzer.cachedVolumeStats`, solve it by migrating `cachedVolumeStats` to `Kubelet` (or migrate it to `StatsProvider` alternatively).

Changes of `fsResourceAnalyzer` and `summaryProviderImpl`:
```go
type fsResourceAnalyzer struct {
    // Remove.
    // Kubelet as statsProvider now.
    statsProvider Provider

    // Add.
    podManager kubepod.Manager

    // Add.
    volumeManager volumemanager.VolumeManager
}

type summaryProviderImpl struct {
    // Remove.
    // Kubelet as provider now.
    provider Provider

    // Add.
    basicInfo *nodeinfo.NodeInfo

    // Add.
    containerManager cm.ContainerManager

    // Add.
    statsProvider *stats.StatsProvider
}
```

Migrate following method from `Kubelet` to `fsResourceAnalyzer`, `summaryProviderImpl`:
```go
    // Following methods belongs to fsResourceAnalyzer.
    GetPods() []*v1.Pod

    ListVolumesForPod(podUID types.UID) (map[string]volume.Volume, bool)

    // Following methods belong to summaryProviderImpl.
    GetNodeConfig() cm.NodeConfig

    GetPodCgroupRoot() string
```

### Test Migration
Two things to be done:
1. Mock fakeNodeInfo, inject it to `TestKubelet` and other fake components.
2. Migrate related unit tests together when functions migrated.

## Implementation History

- 2020-02-27: Initial KEP sent out for discussion & reviewing.