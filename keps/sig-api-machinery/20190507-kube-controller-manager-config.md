---
kep-number: 40
title: Kube-Controller Manager Config
authors:
  - "@luxas"
  - "@stewart-yu"
  - "@sttts"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-cluster-lifecycle
reviewers:
  - "@deads2k"
  - "@liggitt"
  - "@luxas"
  - "@sttts"
  - "@thockin"
approvers:
  - "@deads2k"
  - "@liggitt"
  - "@luxas"
  - "@sttts"
  - "@thockin"
editor:
  name: "@stewart-yu"
creation-date: 2019-05-07
last-updated: 2019-05-22
status: provisional
---

# Kube-Controller Manager Config

**How we can start supporting configure kube-controller manager via a config file instead of command line flags.**

## Table of Contents

- [Kube-Controller Manager Config](#kube-controller-manager-config)
  - [Table of Contents](#table-of-contents)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [How will the component config file look like](#how-will-the-component-config-file-look-like)
    - [How is it versioned](#how-is-it-versioned)
  - [Design Details](#design-details)
    - [Struct for flags](#struct-for-flags)
	- [Structs for serializable config](#structs-for-serializable-config)
	- [Bootstrapping and running the controller manager server](#bootstrapping-and-running-the-controller-manager-server)
  - [Remaining work](#remaining-work)
    - [Add typemeta to every sub-controller configuration](#add-typemeta-to-every-sub-controller-configuration)
    - [Examples](#examples)
       - [Alpha -> Beta Graduation](#alpha---beta-graduation)
       - [Beta -> GA Graduation](#beta---ga-graduation)
       - [Removing a deprecated flag](#removing-a-deprecated-flag)
    - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)

## Summary

Controller manager doesn't consume a config file currently, some problems will be following, describe as below:
- should restart the process of component once we change the value of component filed
- it is difficult to support upgrade component
- we can't dynamic configuration component parameter, and manager all flags is more complex
Besides, few components support consume a versioned config file to configure component process，such as, kubelet,
kube-proxy, kube-scheduler. In Kubernetes v1.10, the Kubelet is firmly on its way to migrating from flags to
versioned configuration files. It can consume a beta-versioned config file and many flags are now deprecated and
pending removal, in favor of this file. Many remaining flags will be replaced by the file over time. Additionally,
the kube-proxy component is very close to having a beta-versioned config file of its own. We have do some versioned
componentConfig about controller manager, we must make it work finally.
In this KEP, we aims to add controller manager config file support, by which we can configure controller manager via
a config file instead of command line flags. Only in this way, we can get rid of ourselves from the complex arguments of
command line.

## Motivation

### Goals

- Sync with others components, such as kube-proxy, kubelet, kube-scheduler, which support consume a config file.
- No requirement to restart a process when you change a file, unlike flags.
- Configure controller manager via a config file instead of command line flags.
- Enable dynamic configuration deployment mechanisms.

### Non-Goals

- flags are nonstandard interfaces with weak stability guarantees. They are confusing and hard to deploy, and this
  is the opposite of what Kubernetes should be.
- put forward the work about upgrade versioned component.

## Proposal

This proposal contains three logical units of work. Each subsection is explained in more detail as below.
- Step 1: Group related options together
- Step 2: Using "option+config" pattern
  we have done some basically work about `Step 1` and `Step 2` before, such as
  - [controller-manager: switch to options+config pattern and add https+auth](https://github.com/kubernetes/kubernetes/pull/59582) by [@sttts](https://github.com/sttts)
  - [split up the huge set of flags into smaller option structs](https://github.com/kubernetes/kubernetes/pull/60270) by [@stewart-yu](https://github.com/stewart-yu)
  - [split the generic component config and options into a kube and cloud part](https://github.com/kubernetes/kubernetes/pull/63283) by [@stewart-yu](https://github.com/stewart-yu)
  - [move specific option sub-struct from controller-manager into kube-controller manager packages](https://github.com/kubernetes/kubernetes/pull/64142) by [@stewart-yu](https://github.com/stewart-yu)
  - [Implement the --controllers flag fully for the cloud-controller manager](https://github.com/kubernetes/kubernetes/pull/68283) by [@stewart-yu](https://github.com/stewart-yu)

- Step 3: Enable consume a config file instead of command line flags.
 - [Implement a dedicated serializer package for ComponentConfigs](https://github.com/kubernetes/kubernetes/pull/74111) by [@stewart-yu](https://github.com/stewart-yu)
 - [Moving ComponentConfig API types to staging repos](https://github.com/kubernetes/community/blob/master/keps/sig-cluster-lifecycle/0014-20180707-componentconfig-api-types-to-staging.md) by [@luxas](https://github.com/luxas) and [@sttts](https://github.com/sttts)

In this KEP, we mostly focus on `Step 3`.

### How will the component config file look like

In current stage, we should support consume component config file like that:
```
kind: KubeControllerManagerConfiguration
apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
...
controllers:   # map[string]runtime.RawExtension
    AttachDetachController/v1alpha1:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        DisableAttachDetachReconcilerSync: false
        ReconcilerSyncLoopPeriod: 1m0s
    CSRSigningController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        ClusterSigningCertFile: "/var/run/kubernetes/server-ca.crt"
        ClusterSigningDuration: 8760h0m0s
        ClusterSigningKeyFile: "/var/run/kubernetes/server-ca.key"
    DaemonSetController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        ConcurrentDaemonSetSyncs: 4
    DeploymentController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        ConcurrentDeploymentSyncs: 5
        DeploymentControllerSyncPeriod: 30s
    DeprecatedController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        DeletingPodsBurst: 0
        DeletingPodsQPS: 0.1
        RegisterRetryCount: 10
    EndpointController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        ConcurrentEndpointSyncs: 5
    GarbageCollectorController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        ConcurrentGCSyncs: 20
        EnableGarbageCollector: true
        GCIgnoredResources:
        - Group: ''
          Resource: events
    Generic:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        Address: 0.0.0.0
        ClientConnection:
          acceptContentTypes: ''
          burst: 30
          contentType: application/vnd.kubernetes.protobuf
          kubeconfig: ''
          qps: 20
        ControllerStartInterval: 0s
        Controllers:
          - "*"
        Debugging:
          enableContentionProfiling: false
          enableProfiling: false
        LeaderElection:
          leaderElect: false
          leaseDuration: 15s
          renewDeadline: 10s
          resourceLock: endpoints
          retryPeriod: 2s
        MinResyncPeriod: 12h0m0s
          Port: 10252
    HPAController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        HorizontalPodAutoscalerCPUInitializationPeriod: 5m0s
        HorizontalPodAutoscalerDownscaleForbiddenWindow: 0s
        HorizontalPodAutoscalerDownscaleStabilizationWindow: 5m0s
        HorizontalPodAutoscalerInitialReadinessDelay: 30s
        HorizontalPodAutoscalerSyncPeriod: 15s
        HorizontalPodAutoscalerTolerance: 0.1
        HorizontalPodAutoscalerUpscaleForbiddenWindow: 0s
        HorizontalPodAutoscalerUseRESTClients: true
    JobController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        ConcurrentJobSyncs: 4
    KubeCloudShared:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        AllocateNodeCIDRs: false
        AllowUntaggedCloud: false
        CIDRAllocatorType: RangeAllocator
        CloudProvider:
          CloudConfigFile: ''
          Name: ''
        ClusterCIDR: ''
        ClusterName: kubernetes
        ConfigureCloudRoutes: true
        ExternalCloudVolumePlugin: ''
        NodeMonitorPeriod: 5s
        NodeSyncPeriod: 0s
        RouteReconciliationPeriod: 10s
        UseServiceAccountCredentials: true
    NamespaceController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        ConcurrentNamespaceSyncs: 10
        NamespaceSyncPeriod: 5m0s
    NodeIPAMController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        NodeCIDRMaskSize: 24
        ServiceCIDR: ''
    NodeLifecycleController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        EnableTaintManager: true
        LargeClusterSizeThreshold: 50
        NodeEvictionRate: 0.1
        NodeMonitorGracePeriod: 40s
        NodeStartupGracePeriod: 1m0s
        PodEvictionTimeout: 5m0s
        SecondaryNodeEvictionRate: 0.01
        UnhealthyZoneThreshold: 0.55
    PersistentVolumeBinderController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        PVClaimBinderSyncPeriod: 15s
        VolumeConfiguration:
          EnableDynamicProvisioning: true
          EnableHostPathProvisioning: false
          FlexVolumePluginDir: "/usr/libexec/kubernetes/kubelet-plugins/volume/exec/"
          PersistentVolumeRecyclerConfiguration:
            IncrementTimeoutHostPath: 30
            IncrementTimeoutNFS: 30
            MaximumRetry: 3
            MinimumTimeoutHostPath: 60
            MinimumTimeoutNFS: 300
            PodTemplateFilePathHostPath: ''
            PodTemplateFilePathNFS: ''
    PodGCController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        TerminatedPodGCThreshold: 12500
    ReplicaSetController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        ConcurrentRSSyncs: 5
    ReplicationController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        ConcurrentRCSyncs: 5
    ResourceQuotaController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        ConcurrentResourceQuotaSyncs: 5
        ResourceQuotaSyncPeriod: 5m0s
    SAController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        ConcurrentSATokenSyncs: 5
        RootCAFile: "/var/run/kubernetes/server-ca.crt"
        ServiceAccountKeyFile: "/tmp/kube-serviceaccount.key"
    ServiceController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        ConcurrentServiceSyncs: 1
    TTLAfterFinishedController:
        kind: AttachDetachControllerConfiguration
        apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
        ConcurrentTTLSyncs: 5
```

Versioning controller configuration independently allows us to have a different development life cycle for 
each controller. We will start with every controller config as v1alpha1. Because versioning is decoupled, 
we can promote the container kind to v1beta1 before all controller configs are promoted. This will decouple
 the discussion about individual controllers and we can delegate their development to the controller owners.

Decoding of the configuration objects is strict. Fields that are not understood by the controller manager, 
will be rejected.
Besides, we construct a configMap used to hold the component config, the struct like
```
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: default
  name: deployment-kube-controller-manager-config   # sync with the value in `--in-cluster-config` flag
data:
  config.yaml: |-
```

we can use the context of KubeControllerManagerConfiguration to fill the `config.yaml` field, so we can use it in cluster
as a resource.

### How is it versioned

Currently, the controller manager's version is `Alpha`, with `register` and `scheme` function.

## Design Details

### Struct for flags

The relative struct `KubeControllerManagerOptions` and `Config` file located in `cmd/kube-controller-manager/app` directory,
the struct define as below:

```
type KubeControllerManagerOptions struct {
	Generic           *cmoptions.GenericControllerManagerConfigurationOptions
	KubeCloudShared   *cmoptions.KubeCloudSharedOptions
	ServiceController *cmoptions.ServiceControllerOptions

	AttachDetachController           *AttachDetachControllerOptions
	CSRSigningController             *CSRSigningControllerOptions
	DaemonSetController              *DaemonSetControllerOptions
	DeploymentController             *DeploymentControllerOptions
	DeprecatedFlags                  *DeprecatedControllerOptions
	EndpointController               *EndpointControllerOptions
	GarbageCollectorController       *GarbageCollectorControllerOptions
	HPAController                    *HPAControllerOptions
	JobController                    *JobControllerOptions
	NamespaceController              *NamespaceControllerOptions
	NodeIPAMController               *NodeIPAMControllerOptions
	NodeLifecycleController          *NodeLifecycleControllerOptions
	PersistentVolumeBinderController *PersistentVolumeBinderControllerOptions
	PodGCController                  *PodGCControllerOptions
	ReplicaSetController             *ReplicaSetControllerOptions
	ReplicationController            *ReplicationControllerOptions
	ResourceQuotaController          *ResourceQuotaControllerOptions
	SAController                     *SAControllerOptions
	TTLAfterFinishedController       *TTLAfterFinishedControllerOptions

	SecureServing *apiserveroptions.SecureServingOptionsWithLoopback
	// TODO: remove insecure serving mode
	InsecureServing *apiserveroptions.DeprecatedInsecureServingOptionsWithLoopback
	Authentication  *apiserveroptions.DelegatingAuthenticationOptions
	Authorization   *apiserveroptions.DelegatingAuthorizationOptions

	Master     string
	Kubeconfig string
}
```
and
```
type Config struct {
	ComponentConfig kubectrlmgrconfig.KubeControllerManagerConfiguration

	SecureServing *apiserver.SecureServingInfo
	// LoopbackClientConfig is a config for a privileged loopback connection
	LoopbackClientConfig *restclient.Config

	// TODO: remove deprecated insecure serving
	InsecureServing *apiserver.DeprecatedInsecureServingInfo
	Authentication  apiserver.AuthenticationInfo
	Authorization   apiserver.AuthorizationInfo

	// the general kube client
	Client *clientset.Clientset

	// the client only used for leader election
	LeaderElectionClient *clientset.Clientset

	// the rest config for the master
	Kubeconfig *restclient.Config

	// the event sink
	EventRecorder record.EventRecorder
}
```

First, we add four filed to `KubeControllerManagerOptions` struct, defined as below
```
// ConfigFile is the location of the kube-controller manager server's configuration file.
ConfigFile string
// WriteConfigTo is the path where the current kube-controller manager server's configuration will be written.
WriteConfigTo string
// InClusterConfig is the name about the current controller manager server's configuration in cluster.
InClusterConfig string
// SkipInClusterConfig indicate if we should process the in cluster config.
SkipInClusterConfig bool
// Watcher is used to watch on the update change of ConfigFile.
Watcher filesystem.FSWatcher
// ErrCh is the channel that errors will be sent.
ErrCh chan error
```

**Notes**
- Field `ConfigFile` used to indicate the directory of local disk config
- Field `Watcher` used to listen the local disk file change, if changed, update cluster config.
- Field `WriteConfigTo` used to write cluster config to disk.
- Field `ErrCh` used to paas the update event to config, so we can execute the merge process.

In-cluster configuration and file configuration are mutually exclusive.

Then, add the same two filed to `Config` struct, defined as below
```
// ConfigFile is the location of the kube-controller manager server's configuration file.
ConfigFile string
// WriteConfigTo is the path where the current kube-controller manager server's configuration will be written.
WriteConfigTo string
// InClusterConfig is the name about the current controller manager server's configuration in cluster.
InClusterConfig string
// SkipInClusterConfig indicate if we should process the in cluster config.
SkipInClusterConfig bool
```

**Notes**
- Field `ConfigFile` used to indicate the directory of local disk config
- Field `WriteConfigTo` used to write cluster config to disk.

### Structs for serializable config

The config file located in pkg/controller/apis/config, pkg/controller/<controller-name>/apis/config  and
k8s.io/kube-controller-manager/api/config)

### Bootstrapping and running the controller manager server

If we read config file from local disk, override the default config, and start filewatch process at the same time.
```
// If we configure the local disk config, start listen the file change.
if len(s.ConfigFile) > 0 {
    c.WriteConfigTo = s.WriteConfigTo
    c.ConfigFile = s.ConfigFile
    if err := s.initWatcher(); err != nil {
        return nil, err
    }
}
```

// start the watchFile process in [Run](https://github.com/kubernetes/kubernetes/blob/master/cmd/kube-controller-manager/app/controllermanager.go#L159)
// If we recive the update local disk config event, we execute the merge process, see the function `Sync()`
```
if len(c.ConfigFile) > 0 {
    if s.Watcher != nil {
        s.Watcher.Run()
    }
    // detect the file change in goroutine
    go func() {
        if err := <-s.ErrCh; err != nil {
            syncErr := c.Sync()
            fmt.Fprintf(os.Stderr, "sync kube-controller manager configMap with error: %v after we recive the channel event: %v\n", syncErr, err)
            os.Exit(1)
        }
    }()

}
if len(c.WriteConfigTo) > 0 {
    if err := config.WriteConfigFile(c.WriteConfigTo, &c.ComponentConfig); err != nil {
        fmt.Fprintf(os.Stderr, "%v\n", err)
        os.Exit(1)
    }
    klog.Infof("Wrote configuration to: %s\n", c.WriteConfigTo)
}
```

We probably get a pattern of input, in the order of overriding:
1) default config
2) cluster config (skip in the process of component start)
3) local disk config
4) flags

support the component consume the config are easily, the most important thing is that how can we make the componentconfig
from the disk, another one from the cluster, plus the flags. all those should be merged eventually. We have the merge process
as below:
- we read the default config first
- if we set the local disk config file, which can be overwrite the config
- at the same time, start the fileWatch process to listen the config file change
- we apply the flags value at the least.
- start the component process, all setting became the cluster config
- file watch the local disk config file all the time, once change the file, we get the cluster config, the apply
  the change into the cluster config.

## Remaining work
### Add typemeta to every sub-controller configuration

The related PR is [add typemeta to every sub-controller configuration, so we can serializable config for KCM](https://github.com/kubernetes/kubernetes/pull/74059) by [@stewart-yu](https://github.com/stewart-yu)
In this way, the config file will like that:
```
kind: KubeControllerManagerConfiguration
apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
foo: bar
---
kind: DaemonSetControllerConfiguration
apiVersion: kubecontrollermanager.config.k8s.io/v1alpha1
bar: baz
...
```
and we should support consume the mutil-file format.
At the same time, a configMap used to hold the component config will look like
```
apiVersion: v1
kind: ConfigMap
metadata:
namespace: default
name: deployment-kube-controller-manager-config
data:
kubectrlconfig.yaml: |-
daemonsetconfig.yaml: |-
<others-controller-name>config.yaml: |-
```

### Examples
These are generalized examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

##### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Make ComponentConfig become serializable
- Tests are in Testgrid and linked in KEP

##### Beta -> GA Graduation

- Add more invalidate.
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and GA/stable, since there's no opportunity for user feedback, or even bug reports, in back-to-back releases.

##### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/conformance-tests.md

### Upgrade / Downgrade Strategy

If you want to upgrade the component, you should not restart component process, just do following steps:
（1）modify the version of config file
（2）modify the configuration what you want to adapt to you version.