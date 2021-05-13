# Support for CSI Plugins on Windows Nodes

## Table of Contents
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Deploy CSI Node Plugin DaemonSet targeting Windows nodes](#deploy-csi-node-plugin-daemonset-targeting-windows-nodes)
    - [Deploy Windows workloads that consume persistent storage managed by a CSI plugin](#deploy-windows-workloads-that-consume-persistent-storage-managed-by-a-csi-plugin)
  - [Implementation Details](#implementation-details)
    - [Enhancements in Kubelet Plugin Watcher](#enhancements-in-kubelet-plugin-watcher)
    - [Enhancements in CSI Node Driver Registrar](#enhancements-in-csi-node-driver-registrar)
    - [New Component: CSI Proxy](#new-component-csi-proxy)
      - [CSI Proxy Named Pipes](#csi-proxy-named-pipes)
      - [CSI Proxy Configuration](#csi-proxy-configuration)
      - [CSI Proxy GRPC API Groups](#csi-proxy-grpc-api-groups)
      - [CSI Proxy GRPC API Graduation and Deprecation Policy](#csi-proxy-grpc-api-graduation-and-deprecation-policy)
    - [Enhancements in Kubernetes/Utils/mounter](#enhancements-in-kubernetesutilsmounter)
    - [Enhancements in CSI Node Plugins](#enhancements-in-csi-node-plugins)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Mitigation using PSP](#mitigation-using-psp)
    - [Mitigation using a webhook](#mitigation-using-a-webhook)
    - [Comparison of risks with CSI Node Plugins in Linux](#comparison-of-risks-with-csi-node-plugins-in-linux)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [API Alternatives](#api-alternatives)
  - [Deployment Alternatives](#deployment-alternatives)
    - [Deploying CSI Node Plugins as binaries and deployed as processes running on the host:](#deploying-csi-node-plugins-as-binaries-and-deployed-as-processes-running-on-the-host)
    - [Package CSI Node Plugins as containers and deployed as processes running on the host:](#package-csi-node-plugins-as-containers-and-deployed-as-processes-running-on-the-host)
    - [Support for Privileged Operations and Bi-directional mount propagation in Windows containers:](#support-for-privileged-operations-and-bi-directional-mount-propagation-in-windows-containers)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] Production readiness review completed
- [x] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Container Storage Interface ([CSI](https://github.com/container-storage-interface/spec/blob/master/spec.md)) is a modern GRPC based standard for implementing external storage plugins (maintained by storage vendors, cloud providers, etc.) for container orchestrators like Kubernetes. Persistent storage requirements of containerized workloads can be satisfied from a diverse array of storage systems by installing and configuring the CSI plugins supported by the desired storage system. This KEP covers the enhancements necessary in Kubernetes core and CSI related out-of-tree components (specific to Kubernetes) to support CSI plugins for Windows nodes in a Kubernetes cluster. With the enhancements proposed in this KEP, Kubernetes operators will be able to leverage modern CSI plugins to satisfy the persistent storage requirements of Windows workloads in Kubernetes.

## Motivation

Support for containerized Windows workloads on Windows nodes in a Kubernetes cluster reached GA status in v1.14. For persistent storage requirements, Windows workloads today depend on: (1) Powershell based [FlexVolume](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-storage/flexvolume.md) [plugins](https://github.com/microsoft/K8s-Storage-Plugins/tree/master/flexvolume/windows) maintained by Microsoft that support mounting remote storage volumes over SMB and iSCSI protocols and (2) In-tree [plugins](https://kubernetes.io/docs/concepts/storage/volumes/#types-of-volumes) in Kubernetes core (kubernetes/kubernetes repository) for popular cloud environments that support formatting and mounting direct attached disks on Windows nodes.

Support for CSI in Kubernetes reached GA status in v1.13. CSI plugins provide several benefits to Linux workloads in Kubernetes today over plugins whose code lives in kubernetes/kubernetes as well as plugins that implement the Flexvolume plugin interface. Some of these benefits are:

1. The GRPC based CSI interface allow CSI plugins to be distributed as containers and fully managed through standard Kubernetes constructs like StatefulSets and DaemonSets. This is a superior mechanism compared to the exec model used by the Flexvolume interface where the plugins are distributed as scripts or binaries that need to be installed on each node and maintained.

2. CSI offers a rich set of volume management operations (although not at a GA state in Kubernetes yet): resizing of volumes, backup/restore of volumes using snapshots and cloning besides the basic volume life-cycle operations (GA since v1.13): provisioning of storage volumes, attaching/detaching volumes to a node and mounting/dismounting to/from a pod.

3. CSI plugins are maintained and released independent of the Kubernetes core. This allows features and bug fixes in the CSI plugins to be delivered in a more flexible schedule relative to Kubernetes releases. Transitioning the code for existing in-tree plugins (especially those targeting specific cloud environments or vendor-specific storage systems) to external CSI plugins can also help reduce the volume of vendor-ed code that needs to be maintained in Kubernetes core.

Given the above context, the main motivations for this KEP are:

1. Enable Windows nodes to support CSI plugins so they can surface the above mentioned benefits of CSI plugins to Windows workloads that have persistent storage requirements. CSI Node Plugins today need to execute several privileged operations like scanning for newly provisioned disks, creating partitions on the disks, formatting the partitions with the desired file system as well as resize the filesystem, staging the partitions at a unique path on the host and propagating the staging path to workload containers. However, Windows does not support privileged operations from inside a container today. This KEP describes a host OS proxy to execute privileged operations on the Windows host OS on behalf of a container. The host OS proxy enables: [a] ease of distribution of CSI Node Plugins as containers for both Windows and Linux, [b] execution of CSI Node Plugins on Windows hosts in a manner similar to Linux hosts - from inside a container and [c] management of the CSI Node Plugin containers through Kubernetes constructs like Pods and DaemonSets.

2. The CSI migration initiative (planned to reach beta state in v1.16) aims to deprecate the code associated with several in-tree storage plugins and pave the path for the ultimate removal of that code from Kubernetes core in favor of CSI plugins that implement the same functionality. Windows workloads need to be aligned with the CSI migration effort and cannot depend on environment specific in-tree plugins to satisfy persistent storage needs.

### Goals

1. Support all CSI Node Plugin operations: NodeStageVolume/NodeUnstageVolume, NodePublishVolume/NodeUnPublishVolume, NodeExpandVolume, NodeGetVolumeStats, NodeGetCapabilities and NodeGetInfo on Windows nodes.

2. Support CSI plugins associated with a variety of external storage scenarios: block storage surfaced through iSCSI as well as directly attached disks (e.g. in cloud environments) as well as remote volumes over SMB.

3. Ability to distribute CSI Node Plugins targeting Windows nodes as containers that can be deployed using DaemonSets in a Kubernetes cluster comprising of Windows nodes.

### Non-Goals

1. Support CSI Controller Plugin operations from Windows nodes: This may be considered in the future but not an immediate priority. Note that this does not require support for privileged operations on a Windows node as required by CSI Node Plugins and thus orthogonal to this KEP around CSI Node Plugins for Windows. If all the worker nodes in the cluster are Windows nodes and Linux master nodes have scheduling disabled then CSI controller plugins cannot be scheduled for now.

2. Support privileged operations from Windows containers beyond CSI Node Plugins: This KEP introduces a host based "privileged proxy" process that may be used for executing privileged operations on the host on behalf of a Windows container. While a similar mechanism may be used for other use cases like containerized CNI plugins (for executing HNS operations), we leave that for a separate KEP. Scoping down the set of actions allowed by the API exposed by by the privileged proxy process to a minimal set simplifies multiple versions of the API as well as reduces the scope for abuse.

3. Support for CSI plugins associated with external storage that requires a special file or block protocol kernel mode driver installed and configured on Windows hosts: e.g. FCoE (Fibre Channel over Ethernet), NFS volumes on Windows and Dokany based filesystems (https://github.com/dokan-dev/dokany) like SSHFS, etc.

## Proposal

In this KEP, we propose a set of enhancements in pre-existing components to support CSI Node Plugins on Windows nodes.

The following enhancements are necessary in existing Kuberentes community managed code:
1. Ability to handle Windows file paths in the Kubelet plugin watcher for domain sockets on Windows nodes.
2. Refactor code in the CSI Node Driver Registrar so that it can be compiled for Windows.
3. Build official CSI Node Driver Registrar container images based on Windows base images and publish them in official CSI community container registry.

The following enhancements are necessary in CSI plugins maintained by CSI plugin authors:
1. Refactor code in existing CSI Node Plugins to support Windows. All privileged operations will need to be driven through an API exposed by a "privileged proxy" binary described below. Details around this will be documented in a plugin developer guide.
2. Build CSI Node Plugin container images based on Windows base images.
3. Create DaemonSet YAMLs referring to official CSI Node Driver Registrar container images and CSI Node Plugin container images targeting Windows.

Besides the above enhancements, a new "privileged proxy" binary, named csi-proxy.exe is a key aspect of this KEP. csi-proxy.exe will run as a native Windows process on the Windows nodes configured as a Windows Service. csi-proxy.exe will expose an API (through GRPC over a named pipe) for executing privileged storage related operations on Windows hosts on behalf of Windows containers like CSI Node Plugins.

### User Stories

With the KEP implemented, administrators should be able to deploy CSI Node Plugins that support staging, publishing and other storage management operations on Windows nodes. Operators will be able to schedule Windows workloads that consume persistent storage surfaced by CSI plugins on Windows nodes.

#### Deploy CSI Node Plugin DaemonSet targeting Windows nodes

An administrator should be able to deploy a CSI Node Plugin along with the CSI Node Driver Registrar container targeting Windows nodes with a DaemonSet YAML like the following:

```
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: csi-gce-pd-node-win
spec:
  selector:
    matchLabels:
      app: gcp-compute-persistent-disk-csi-driver-win
  template:
    metadata:
      labels:
        app: gcp-compute-persistent-disk-csi-driver-win
    spec:
      serviceAccountName: csi-node-sa
      tolerations:
      - key: "node.kubernetes.io/os"
        operator: "Equal"
        value: "win1809"
        effect: "NoSchedule"
      nodeSelector:
        kubernetes.io/os: windows
      containers:
        - name: csi-driver-registrar
          image: gke.gcr.io/csi-node-driver-registrar:win-v1
          args:
            - "--v=5"
            - "--csi-address=unix://C:\\csi\\csi.sock"
            - "--kubelet-registration-path=C:\\var\\lib\\kubelet\\plugins\\pd.csi.storage.gke.io\\csi.sock"
          env:
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
          volumeMounts:
            - name: plugin-dir
              mountPath: C:\csi
            - name: registration-dir
              mountPath: C:\registration
        - name: gce-pd-driver
          image: gke.gcr.io/gcp-compute-persistent-disk-csi-driver:win-v1
          args:
            - "--v=5"
            - "--endpoint=unix:/csi/csi.sock"
          volumeMounts:
            - name: kubelet-dir
              mountPath: C:\var\lib\kubelet
            - name: plugin-dir
              mountPath: C:\csi
            - name: csi-proxy-volume-v1beta1
              mountPath: \\.\pipe\csi-proxy-volume-v1beta1
            - name: csi-proxy-filesystem-v1beta1
              mountPath: \\.\pipe\csi-proxy-filesystem-v1beta1
            - name: csi-proxy-disk-v1beta2
              mountPath: \\.\pipe\csi-proxy-disk-v1beta2
      volumes:
        - name: csi-proxy-disk-v1beta2
          hostPath:
            path: \\.\pipe\csi-proxy-disk-v1beta2
            type: ""
        - name: csi-proxy-volume-v1beta1
          hostPath:
            path: \\.\pipe\csi-proxy-volume-v1beta1
            type: ""
        - name: csi-proxy-filesystem-v1beta1
          hostPath:
            path: \\.\pipe\csi-proxy-filesystem-v1beta1
            type: ""
        - name: registration-dir
          hostPath:
            path: C:\var\lib\kubelet\plugins_registry\
            type: Directory
        - name: kubelet-dir
          hostPath:
            path: C:\var\lib\kubelet\
            type: Directory
        - name: plugin-dir
          hostPath:
            path: C:\var\lib\kubelet\plugins\pd.csi.storage.gke.io\
            type: DirectoryOrCreate
```

Note that references to GCE PD CSI Plugin is used as an example above based on a prototype port of GCE PD CSI plugin with the enhancements in this KEP. Controller pods for the CSI plugin can be deployed on Linux nodes in the cluster in the same manner as it is done today.

#### Deploy Windows workloads that consume persistent storage managed by a CSI plugin

An operator should be able to deploy a Windows workload like SQL Server that consumes dynamically provisioned Persistent Volumes managed by a CSI plugin using:

A storage class like:
```
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: csi-gce-pd
provisioner: pd.csi.storage.gke.io
parameters:
  type: pd-standard
```

with a PVC like:
```
apiVersion: v1
metadata:
  name: sqlpvc
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: csi-gce-pd
  resources:
    requests:
      storage: 100Gi
```

and a Pod like:
```
apiVersion: v1
kind: Pod
metadata:
  name: sqlserver1
spec:
  tolerations:
  - key: "node.kubernetes.io/os"
    operator: "Equal"
    value: "win1809"
    effect: "NoSchedule"
  nodeSelector:
    beta.kubernetes.io/os: windows
  containers:
  - name: sqlpod
    image: ddebroy/sqlsrv:latest
    volumeMounts:
    - name: csi-sql-vol
      mountPath: C:\Data
    env:
    - name: ACCEPT_EULA
      value: "y"
    - name: sa_password
      value: "*****"
  volumes:
    - name: csi-sql-vol
      persistentVolumeClaim:
        claimName: sqlpvc
```

### Implementation Details

CSI Node Plugins listen on domain sockets and respond to CSI API requests sent over GRPC from a container orchestrator like Kubernetes. They are responsible all storage management operations scoped around a specific node that are typically necessary after a CSI Controller Plugin has finished provisioning a Persistent Volume and attached it to the node. In Kubernetes, the CSI Node API calls are invoked by the CSI In-tree Plugin in the kubelet as well as the CSI Node Driver Registrar. The CSI Node Driver Registrar interacts with the Kubelet Plugin Watcher and it is maintained by the Kubernetes CSI community as a side-car container for deployment in CSI Node Plugin pods.

![Kubernetes CSI Components](https://raw.githubusercontent.com/kubernetes/community/master/contributors/design-proposals/storage/container-storage-interface_diagram1.png?raw=true "Kubernetes CSI Components")

Support for Unix Domain Sockets has been introduced in Windows Server 2019 and works across containers as well as host and container as long as the processes running in containers are listening on the socket. If a process from within a container wishes to connect to a domain socket that a process on the host OS is listening on, Windows returns a permission error. This scenario however does not arise in the context of interactions between Kubelet, CSI Node Driver Registrar and CSI Node Plugin as these involve a process in a container listening on a domain socket (CSI Node Driver Registrar or CSI Node Plugin) that a process on the host (Kubelet) connects to.

Golang supports domain socket operations for Windows since go version 1.12. It was found that in Windows, `os.ModeSocket` is not set on the `os.FileMode` associated with domain socket files in Windows. This issue is tracked [here](https://github.com/golang/go/issues/33357). Therefore determining whether a file is a domain socket file using `os.ModeSocket` does not work on Windows right now. We can potentially work around this by sending down a FSCTL to the file and evaluating the Windows reparse points to determine if the file is backed by a domain socket.

Based on the above, we can conclude that some of the fundamental support in the OS and compiler with regards to domain sockets in the context of CSI plugin discovery and a channel for API invocation is present in a stable state in Windows Server 2019 today. Although there are some observed limitations with respect to domain sockets in Windows Server 2019, they are not major blockers in the context of CSI Node Plugins. In the section below, we call out the components in the context of CSI Node Plugins in Kubernetes that will need to be enhanced to properly account for Windows paths and make use of domain sockets in Windows in a manner very similar to Linux.

CSI Node Plugins need to execute certain privileged operations at the host level as well as propagate mount points in response to the CSI API calls. Such operations involve: scanning disk identities to map the node OS's view of a disk device to a CSI volume provisioned and attached by CSI controller plugins, partitioning a disk and formatting it when necessary, bind-mounting volumes from the host to the container workload, resizing of the file system as part of a volume resize, etc. These operations cannot be invoked from a container in Windows today. As a result containerized CSI Node Plugins in Windows require some mechanism to perform these privileged operations on their behalf on the Windows host OS. csi-proxy.exe, described below serves that role by performing the storage related privileged operations on behalf of containers. Alternative approaches to csi-proxy.exe (for example, deploying the CSI plugin as regular binaries on the host without any containers) are described further below in the Alternatives section.

#### Enhancements in Kubelet Plugin Watcher

Registration of CSI Node Plugins on a Kubernetes node is handled by the Kubelet plugin watcher using the fsnotify package. This component needs to convert paths detected by fsnotify to Windows paths in handleCreateEvent() and handleDeleteEvent() before the paths are passed to AddOrUpdatePlugin() RemovePlugin() routines in desiredStateOfTheWorld. A new utility function, NormalizeWindowsPath(), will be added in utils to handle this.

Given `os.ModeSocket` is not set on a socket file's `os.FileMode` in Windows (due to golang [issue](https://github.com/golang/go/issues/33357)), a specific check for `os.ModeSocket` in handleCreateEvent() will need to be relaxed for Windows until the golang issue is addressed.

#### Enhancements in CSI Node Driver Registrar

The code for the CSI Node Driver Registrar needs to be refactored a bit so that it cleanly compiles when GOOS=windows is set. This mainly requires removal of dependencies on golang.org/x/sys/unix from nodeRegister() when building on Windows nodes.

Once compiled for Windows, container images based on Window Base images (like NanoServer) needs to be published and maintained.

#### New Component: CSI Proxy

A "privileged proxy" binary, csi-proxy.exe, will need to be developed and maintained by the Kubernetes CSI community to allow containerized CSI Node Plugins to perform privileged operations at the Windows host OS layer. Kubernetes administrators will need to install and maintain csi-proxy.exe on all Windows nodes in a manner similar to dockerd.exe today or containerd.exe in the future. A Windows node will typically be expected to be configured to run only one instance of csi-proxy.exe as a Windows Service that can be used by all CSI Node Plugins.

##### CSI Proxy Named Pipes
A CSI Node Plugin will interact with csi-proxy.exe using named pipes of the form: `\\.\pipe\csi-proxy-[api-group]-[version]`. These named pipes will be surfaced by csi-proxy.exe as it initializes. The `api-group` and `version` suffix in the pipe name corresponds to a specific version of an API Group that is required by a CSI plugin. Specific examples of named pipes corresponding to versions include: `\\.\pipe\csi-proxy-volume-v1beta1`, `\\.\pipe\csi-proxy-disk-v2`, `\\.\pipe\csi-proxy-iscsi-v3beta1`, etc. Each pipe (corresponding to a API group and version) will need to be mounted into the Node Plugin container from the host using the pod's volume mount specifications. Note that domain sockets cannot be used in this scenario since Windows blocks a containerized process from interacting with a host process that is listening on a domain socket. If such support is enabled on Windows in the future, we may consider switching CSI Proxy to use domain sockets however Windows named pipes is a common mechanism used by containers today to interact with host processes like docker daemon.

![Kubernetes CSI Node Components and Interactions](csi-proxy3.png?raw=true "Kubernetes CSI Node Components and Interactions")

Groups of GRPC based APIs will be used by CSI Node Plugins to invoke privileged operations on the host through csi-proxy.exe. Each API group will be versioned and any release of csi-proxy.exe binary will strive to maintain backward compatibility across as many prior stable versions of an API group as possible. This avoids having to run multiple versions of the csi-proxy.exe binary on the same Windows host if multiple CSI Node Plugins have been configured and the version of the API group required by the plugins are different.

##### CSI Proxy Configuration
There will be two command line parameters that may be passed to csi-proxy.exe:
1. kubelet-csi-plugins-path: String parameter pointing to the path used by Kubernetes CSI plugins on each node. Will default to: `C:\var\lib\kubelet\plugins\kubernetes.io\csi`. All requests for creation and deletion of paths through the CSIProxyService RPCs (detailed below) will need to be under this path.

2. kubelet-pod-path: String parameter pointing to the path used by the kubelet to store pod specific information. This should map to the value returned by [getPodsDir](https://github.com/kubernetes/kubernetes/blob/e476a60ccbe25581f5a6a9401081dcee311a066e/pkg/kubelet/kubelet_getters.go#L48). By default it will be set to: `C:\var\lib\kubelet\pods` Parameters to `LinkPath` (detailed below) will need to be under this path.

##### CSI Proxy GRPC API Groups
The following are the main RPC API groups and Request/Response structures that will be supported by CSI Proxy at v1.  Note that the specific structures as well as restrictions on them are expected to evolve over time.

```

service Disk {
    // ListDiskLocations returns locations <Adapter, Bus, Target, LUN ID> of all
    // disk devices enumerated by the host
    rpc ListDiskLocations(ListDiskLocationsRequest) returns (ListDiskLocationsResponse) {}

    // ListDiskIDs returns a map of DiskID objects where the key is the disk number
    rpc ListDiskIDs(ListDiskIDsRequest) returns (ListDiskIDsResponse) {}

    // PartitionDisk initializes and partitions a disk device (if the disk has not
    // been partitioned already) and returns the resulting volume device number
    rpc PartitionDisk(PartitionDiskRequest) returns (PartitionDiskResponse) {}

    // DiskStats returns the size in bytes for the entire disk
    rpc GetDiskStats(GetDiskStatsRequest) returns (GetDiskStatsResponse) {}

    // SetDiskState sets the offline/online state of a disk
    rpc SetDiskState(SetDiskStateRequest) returns (SetDiskStateResponse) {}

    // GetDiskState gets the offline/online state of a disk
    rpc GetDiskState(GetDiskStateRequest) returns (GetDiskStateResponse) {}
}

message ListDiskLocationsRequest {
    // Intentionally empty
}

message DiskLocation {
    string Adapter = 1;
    string Bus = 2;
    string Target = 3;
    string LUNID = 4;
}

message ListDiskLocationsResponse {
    // Map of disk device IDs and <adapter, bus, target, lun ID> associated with each disk device
    map <string, DiskLocation> disk_locations = 1;
}

message PartitionDiskRequest {
    // Disk device number of the disk to partition
    int64 disk_number = 1;
}

message PartitionDiskResponse {
    // Intentionally empty
}

message ListDiskIDsRequest {
    // Intentionally empty
}

enum DiskIDType {
    // Placeholder
    NONE = 0;

    // SCSI Page 83 IDs [https://www.t10.org/ftp/t10/document.02/02-419r6.pdf]
    // retrieved through STORAGE_DEVICE_ID_DESCRIPTOR as described in
    // https://docs.microsoft.com/en-us/windows/win32/api/winioctl/ns-winioctl-storage_device_id_descriptor
    SCSI_PAGE_83 = 1;
}

message DiskIDs {
    // Map of Disk ID types and Disk ID values
    map <DiskIDType, string> identifiers = 1;
}

message ListDiskIDsResponse {
    // Map of disk device numbers and IDs <page83> associated with each disk device
    map <string, DiskIDs> disk_ids = 1;
}

message GetDiskStatsRequest {
    // Disk device number of the disk to get the size from
    int64 disk_number = 1;
}

message GetDiskStatsResponse {
    // Total size of the volume in bytes
    int64 disk_size_bytes = 1;
}

message SetDiskStateRequest {
    // Disk device number of the disk whose state will change
    int64 disk_number = 1;

    // Online state to set for the disk. true for online, false for offline
    bool is_online = 2;
}

message SetDiskStateResponse {
    // Intentionally empty
}

message GetDiskStateRequest {
    // Disk device number of the disk to query
    int64 disk_number = 1;
}

message GetDiskStateResponse {
    // Online state of the disk. true for online, false for offline
    bool is_online = 1;
}



service Volume {
    // ListVolumesOnDisk returns the volume IDs (in \\.\Volume{GUID} format) for
    // all volumes on a Disk device
    rpc ListVolumesOnDisk(ListVolumesOnDiskRequest) returns (ListVolumesOnDiskResponse) {}

    // MountVolume mounts the volume at the requested global staging path
    rpc MountVolume(MountVolumeRequest) returns (MountVolumeResponse) {}

    // UnmountVolume gracefully dismounts a volume
    rpc UnmountVolume(UnmountVolumeRequest) returns (UnmountVolumeResponse) {}

    // IsVolumeFormatted checks if a volume is formatted with a known filesystem
    rpc IsVolumeFormatted(IsVolumeFormattedRequest) returns (IsVolumeFormattedResponse) {}

    // FormatVolume formats a volume with NTFS
    rpc FormatVolume(FormatVolumeRequest) returns (FormatVolumeResponse) {}

    // ResizeVolume performs resizing of the partition and file system for a block based volume
    rpc ResizeVolume(ResizeVolumeRequest) returns (ResizeVolumeResponse) {}

    // VolumeStats gathers total size and used size (in bytes) for a volume
    rpc GetVolumeStats(GetVolumeStatsRequest) returns (GetVolumeStatsResponse) {}

    // GetDiskNumberFromVolumeID gets the disk number of the disk where the volume is located
    rpc GetDiskNumberFromVolumeID(GetDiskNumberFromVolumeIDRequest) returns (GetDiskNumberFromVolumeIDResponse) {}

    // GetVolumeIDFromTargetPathRequest gets the volume id mounted at the given path
    rpc GetVolumeIDFromTargetPathRequest(GetVolumeIDFromTargetPathRequest) returns (GetVolumeIDFromTargetPathResponse) {}

    // WriteVolumeCache writes the file system cache of a volume (with given id) to disk
    rpc WriteVolumeCache(WriteVolumeCacheRequest) returns (WriteVolumeCacheResponse) {}
}

message ListVolumesOnDiskRequest {
    // Disk number of the disk to query for volumes
    string disk_number = 1;
}

message ListVolumesOnDiskResponse {
    // Volume device IDs of volumes on the specified disk
    repeated string volume_ids = 1;
}

message MountVolumeRequest {
    // Volume device ID of the volume to mount
    string volume_id = 1;
    // Path in the host's file system where the volume needs to be mounted
    string target_path = 2;
}

message MountVolumeResponse {
    // Intentionally empty
}

message UnmountVolumeRequest {
    // Volume device ID of the volume to dismount
    string volume_id = 1;
    // Path where the volume has been mounted.
    string target_path = 2;
}

message UnmountVolumeResponse {
    // Intentionally empty
}

message IsVolumeFormattedRequest {
    // Volume device ID of the volume to check
    string volume_id = 1;
}

message IsVolumeFormattedResponse {
    // Is the volume formatted with NTFS
    bool formatted = 1;
}

message FormatVolumeRequest {
    // Volume device ID of the volume to format with NTFS
    string volume_id = 1;
}

message FormatVolumeResponse {
    // Intentionally empty
}

message ResizeVolumeRequest {
    // Volume device ID of the volume to dismount
    string volume_id = 1;
    // New size of the volume
    int64 size = 2;
}

message ResizeVolumeResponse {
    // Intentionally empty
}

message GetVolumeStatsRequest {
    // Volume device Id of the volume to get the stats for
    string volume_id = 1;
}

message GetVolumeStatsResponse {
    // Capacity of the volume
    int64 total_bytes = 1;
    // Used bytes
    int64 used_bytes = 2;
}

message GetDiskNumberFromVolumeIDRequest {
    // Volume device Id of the volume to get the disk number for
    string volume_id = 1;
}

message GetDiskNumberFromVolumeIDResponse {
    // Corresponding disk number
    int64 disk_number = 1;
}

message GetVolumeIDFromTargetPathRequest {
    // Target mount path to query
    string mount = 1;
}

message GetVolumeIDFromTargetPathResponse {
    // Volume device ID of the volume mounted at the mount point
    string volume_id = 1;
}

message WriteVolumeCacheRequest {
    // Volume device ID of the volume to flush the cache
    string volume_id = 1;
}

message WriteVolumeCacheResponse {
    // Intentionally empty
}



service Smb {
    // NewSmbGlobalMapping creates an SMB mapping on the SMB client to an SMB share.
    rpc NewSmbGlobalMapping(NewSmbGlobalMappingRequest) returns (NewSmbGlobalMappingResponse) {}

    // RemoveSmbGlobalMapping removes the SMB mapping to an SMB share.
    rpc RemoveSmbGlobalMapping(RemoveSmbGlobalMappingRequest) returns (RemoveSmbGlobalMappingResponse) {}
}


message NewSmbGlobalMappingRequest {
    // A remote SMB share to mount
    // All unicode characters allowed in SMB server name specifications are
    // permitted except for restrictions below
    //
    // Restrictions:
    // SMB remote path specified in the format: \\server-name\sharename, \\server.fqdn\sharename or \\a.b.c.d\sharename
    // If not an IP address, share name has to be a valid DNS name.
    // UNC specifications to local paths or prefix: \\?\ is not allowed.
    // Characters: + [ ] " / : ; | < > , ? * = $ are not allowed.
    string remote_path = 1;
    // Optional local path to mount the smb on
    string local_path = 2;

    // Username credential associated with the share
    string user_name = 3;

    // Password credential associated with the share
    string password = 4;
}

message NewSmbGlobalMappingResponse {
    // Intentionally empty
}


message RemoveSmbGlobalMappingRequest {
    // A remote SMB share mapping to remove
    // All unicode characters allowed in SMB server name specifications are
    // permitted except for restrictions below
    //
    // Restrictions:
    // SMB share specified in the format: \\server-name\sharename, \\server.fqdn\sharename or \\a.b.c.d\sharename
    // If not an IP address, share name has to be a valid DNS name.
    // UNC specifications to local paths or prefix: \\?\ is not allowed.
    // Characters: + [ ] " / : ; | < > , ? * = $ are not allowed.
    string remote_path = 1;
}

message RemoveSmbGlobalMappingResponse {
    // Intentionally empty
}



service Iscsi {
    // AddTargetPortal registers an iSCSI target network address for later
    // discovery.
    // AddTargetPortal currently does not support selecting different NICs or
    // a different iSCSI initiator (e.g a hardware initiator). This means that
    // Windows will select the initiator NIC and instance on its own.
    rpc AddTargetPortal(AddTargetPortalRequest)
        returns (AddTargetPortalResponse) {}

    // DiscoverTargetPortal initiates discovery on an iSCSI target network address
    // and returns discovered IQNs.
    rpc DiscoverTargetPortal(DiscoverTargetPortalRequest)
        returns (DiscoverTargetPortalResponse) {}

    // RemoveTargetPortal removes an iSCSI target network address registration.
    rpc RemoveTargetPortal(RemoveTargetPortalRequest)
        returns (RemoveTargetPortalResponse) {}

    // ListTargetPortal lists all currently registered iSCSI target network
    // addresses.
    rpc ListTargetPortals(ListTargetPortalsRequest)
        returns (ListTargetPortalsResponse) {}

    // ConnectTarget connects to an iSCSI Target
    rpc ConnectTarget(ConnectTargetRequest) returns (ConnectTargetResponse) {}

    // DisconnectTarget disconnects from an iSCSI Target
    rpc DisconnectTarget(DisconnectTargetRequest)
        returns (DisconnectTargetResponse) {}

    // GetTargetDisks returns the disk addresses that correspond to an iSCSI
    // target
    rpc GetTargetDisks(GetTargetDisksRequest) returns (GetTargetDisksResponse) {}

    // SetMutualChapSecret sets the default CHAP secret that all initiators on
    // this machine (node) use to authenticate the target on mutual CHAP
    // authentication.
    // NOTE: This method affects global node state and should only be used
    //       with consideration to other CSI drivers that run concurrently.
    rpc SetMutualChapSecret(SetMutualChapSecretRequest)
        returns (SetMutualChapSecretResponse) {}
}

// TargetPortal is an address and port pair for a specific iSCSI storage
// target.
message TargetPortal {
    // iSCSI Target (server) address
    string target_address = 1;

    // iSCSI Target port (default iSCSI port is 3260)
    uint32 target_port = 2;
}

message AddTargetPortalRequest {
    // iSCSI Target Portal to register in the initiator
    TargetPortal target_portal = 1;
}

message AddTargetPortalResponse {
    // Intentionally empty
}

message DiscoverTargetPortalRequest {
    // iSCSI Target Portal on which to initiate discovery
    TargetPortal target_portal = 1;
}

message DiscoverTargetPortalResponse {
    // List of discovered IQN addresses
    // follows IQN format: iqn.yyyy-mm.naming-authority:unique-name
    repeated string iqns = 1;
}

message RemoveTargetPortalRequest {
    // iSCSI Target Portal
    TargetPortal target_portal = 1;
}

message RemoveTargetPortalResponse {
    // Intentionally empty
}

message ListTargetPortalsRequest {
    // Intentionally empty
}

message ListTargetPortalsResponse {
    // A list of Target Portals currently registered in the initiator
    repeated TargetPortal target_portals = 1;
}

// iSCSI logon authentication type
enum AuthenticationType {
    // No authentication is used
    NONE = 0;

    // One way CHAP authentication. The target authenticates the initiator.
    ONE_WAY_CHAP = 1;

    // Mutual CHAP authentication. The target and initiator authenticate each
    // other.
    MUTUAL_CHAP = 2;
}

message ConnectTargetRequest {
    // Target portal to which the initiator will connect
    TargetPortal target_portal = 1;

    // IQN of the iSCSI Target
    string iqn = 2;

    // Connection authentication type, None by default
    //
    // One Way Chap uses the chap_username and chap_secret
    // fields mentioned below to authenticate the initiator.
    //
    // Mutual Chap uses both the user/secret mentioned below
    // and the Initiator Chap Secret (See `SetMutualChapSecret`)
    // to authenticate the target and initiator.
    AuthenticationType auth_type = 3;

    // CHAP Username used to authenticate the initiator
    string chap_username = 4;

    // CHAP password used to authenticate the initiator
    string chap_secret = 5;
}

message ConnectTargetResponse {
    // Intentionally empty
}

message GetTargetDisksRequest {
    // Target portal whose disks will be queried
    TargetPortal target_portal = 1;

    // IQN of the iSCSI Target
    string iqn = 2;
}

message GetTargetDisksResponse {
    // List composed of disk ids (numbers) that are associated with the
    // iSCSI target
    repeated string diskIDs = 1;
}

message DisconnectTargetRequest {
    // Target portal from which initiator will disconnect
    TargetPortal target_portal = 1;

    // IQN of the iSCSI Target
    string iqn = 2;
}

message DisconnectTargetResponse {
    // Intentionally empty
}

message SetMutualChapSecretRequest {
    // the default CHAP secret that all initiators on this machine (node) use to
    // authenticate the target on mutual CHAP authentication.
    // Must be at least 12 byte long for non-Ipsec connections, at least one
    // byte long for Ipsec connections, and at most 16 bytes long.
    string MutualChapSecret = 1;
}

message SetMutualChapSecretResponse {
    // Intentionally empty
}



service System {
    // GetBIOSSerialNumber returns the device's serial number
    rpc GetBIOSSerialNumber(GetBIOSSerialNumberRequest)
        returns (GetBIOSSerialNumberResponse) {}

    // Rescan refreshes the host's storage cache
    rpc Rescan(RescanRequest) returns (RescanResponse) {}
}

message GetBIOSSerialNumberRequest {
    // Intentionally empty
}

message GetBIOSSerialNumberResponse {
    // Serial number
    string serial_number = 1;
}

message RescanRequest {
    // Intentionally empty
}

message RescanResponse {
    // Intentionally empty
}



service Filesystem {
    // LinkPath creates a local directory symbolic link between a source path
    // and target path in the host's filesystem
    rpc LinkPath(LinkPathRequest) returns (LinkPathResponse) {}

    //IsMountPoint checks if a given path is mount or not
    rpc IsMountPoint(IsMountPointRequest) returns (IsMountPointResponse) {}
}

message LinkPathRequest {
    // The path where the symlink is created in the host's filesystem.
    // All special characters allowed by Windows in path names will be allowed
    // except for restrictions noted below. For details, please check:
    // https://docs.microsoft.com/en-us/windows/win32/fileio/naming-a-file
    //
    // Restrictions:
    // Only absolute path (indicated by a drive letter prefix: e.g. "C:\") is accepted.
    // The path prefix needs needs to match the paths specified as
    // kubelet-csi-plugins-path parameter of csi-proxy.
    // UNC paths of the form "\\server\share\path\file" are not allowed.
    // All directory separators need to be backslash character: "\".
    // Characters: .. / : | ? * in the path are not allowed.
    // source_path cannot already exist in the host filesystem.
    // Maximum path length will be capped to 260 characters.
    string source_path = 1;

    // Target path in the host's filesystem used for the symlink creation.
    // All special characters allowed by Windows in path names will be allowed
    // except for restrictions noted below. For details, please check:
    // https://docs.microsoft.com/en-us/windows/win32/fileio/naming-a-file
    //
    // Restrictions:
    // Only absolute path (indicated by a drive letter prefix: e.g. "C:\") is accepted.
    // The path prefix needs to match the paths specified as
    // kubelet-pod-path parameter of csi-proxy.
    // UNC paths of the form "\\server\share\path\file" are not allowed.
    // All directory separators need to be backslash character: "\".
    // Characters: .. / : | ? * in the path are not allowed.
    // target_path needs to exist as a directory in the host that is empty.
    // target_path cannot be a symbolic link.
    // Maximum path length will be capped to 260 characters.
    string target_path = 2;
}

message LinkPathResponse {
    // Intentionally empty
}

message IsMountPointRequest {
    // The path whose existence we want to check in the host's filesystem
    string path = 1;
}

message IsMountPointResponse {
    // Indicates whether the path in PathExistsRequest exists in the host's filesystem
    bool is_mount_point = 2;
}

```

##### CSI Proxy GRPC API Graduation and Deprecation Policy

In accordance with standard Kubernetes conventions, the above API will be introduced as v1alpha1 and graduate to v1beta1 and v1 as the feature graduates. Beyond a vN release in the future, new RPCs and enhancements to parameters will be introduced through vN+1alpha1 and graduate to vN+1beta1 and vN+1 stable versions as the new APIs mature.

Members of CSI Proxy API Group may be deprecated and then removed from csi-proxy.exe in a manner similar to Kubernetes deprecation (policy)[https://kubernetes.io/docs/reference/using-api/deprecation-policy/] although maintainers will make an effort to ensure such deprecation is as rare as possible. After their announced deprecation, a member of CSI Proxy API Group must be supported:
1. 12 months or 3 releases (whichever is longer) if the API member is part of a Stable/vN version.
2. 9 months or 3 releases (whichever is longer) if the API member is part of a Beta/vNbeta1 version.
3. 0 releases if the API member is part of an Alpha/vNalpha1 version.

To continue running CSI Node Plugins that depend on an old version of csi-proxy.exe (exposing vN of a certain API group, some of whose members have been removed), Kubernetes administrators will be required to run the latest version of the csi-proxy.exe (that will be used by CSI Node Plugins that use versions of the API group more recent than vN) along with an old version of csi-proxy.exe (that does support vN of API group).

Introduction of new RPCs or enhancements to parameters is expected to be inspired by new requirements from plugin authors as well as CSI functionality enhancement.

#### Enhancements in Kubernetes/Utils/mounter

Once the [PR](https://github.com/kubernetes/utils/pull/100/files) lands, a mounter/mount_windows_using_csi_proxy.go in Kubernetes/Utils/mounter package can be introduced. It will implement the mounter and hostutil interfaces against the CSI Proxy API Group.

#### Enhancements in CSI Node Plugins

Code for CSI Node Plugins need to be refactored to support CSI Node APIs in both Linux and Windows nodes. While the code targeting Linux nodes can assume privileged access to the host, the code targeting Windows nodes need to invoke the GRPC client API associated with the desired version of a csi-proxy API group described above. CSI Node Plugins that will use the Kubernetes/Utils/mounter package introduced in this [PR](https://github.com/kubernetes/utils/pull/100/files) will require minimal platform specific code targeting Windows and Linux.

Once compiled for Windows, container images based on Window Base images (like NanoServer) needs to be published and maintained. Container images targeting Linux nodes will need to be based on the desired Linux distro base image.

New YAMLs for DaemonSets associated with CSI Node Plugins needs to be authored that will (1) target Windows nodes and (2) use Windows paths for domain socket related paths as illustrated in the User Story section above.

### Risks and Mitigations

Any pod on a Windows node can be configured to mount `\\.\pipe\csi-proxy-v[N]` and perform privileged operations. Thus csi-proxy presents a potential security risk. To mitigate the risk, some options are described below:

#### Mitigation using PSP
Administrators can enable Pod Security Policy (PSP) in their cluster and configure PSPs to:
1. Disallow host path mounts as part of a default cluster-wide PSP. This will affect all pods in the cluster across Linux and Windows that mount any host paths.
2. Allow host path mounts with pathPrefix = `\\.\pipe\csi-proxy`. Restrict usage of this PSP to only SAs associated with the DaemonSets of CSI Node Plugins.
Support will need to be implemented in AllowsHostVolumePath to handle Windows pipe paths.

#### Mitigation using a webhook
An admission webhook can be implemented and deployed in clusters with Windows nodes that will reject all containers that mount paths with prefix `\\.\pipe\csi-proxy` as a hostPath volume but does not have privileged flag set in the pod's securityContext specification. This allows the privileged setting to be used for Windows pods as an indication the container will perform privileged operations. Other cluster-wide policies (e.g. PSP) that act on the privileged setting in a container's securityContext can enforce the same for CSI Node plugins targeting Windows nodes. Note that this does not in any way change how the privileged setting is used today for Linux nodes. If in the future, full privileged container support is introduced in Windows (as in Linux today), functionality of existing CSI Node Plugin DaemonSets (targeting Windows) with the privileged flag set should not get negatively impacted as they will be launched as privileged containers.

#### Comparison of risks with CSI Node Plugins in Linux
In Linux nodes, CSI Node Plugins typically surface a domain socket used by the kubelet to invoke CSI API calls on the node plugin. This socket may also be mounted by a malicious pod to invoke CSI API calls that may be potentially destructive - for example a Node Stage API call that leads to a volume being formatted. The malicious pod will have to correctly guess the parameters for the CSI API calls for a particular node plugin in order to influence it's behavior as well as circumvent validation checks (or exploit logical vulnerabilities/bugs) in the plugin's code. In case of Windows, a malicious pod can perform destructive actions using the csi-proxy pipe with far less barriers in the absence of the above mitigations. However the overall attack surface of using a mounted domain socket in Linux or a named pipe in Windows from a malicious pod is similar.

## Design Details

### Test Plan

Unit tests will be added to verify Windows related enhancements in existing Kubernetes components mentioned above.

All E2E storage tests covering CSI plugins will be ported to Windows workloads and successfully executed with above enhancements in place along with csi-proxy.exe.

### Graduation Criteria

#### Alpha -> Beta Graduation

- csi-proxy.exe supports v1beta1 version of the CSI Proxy API Group.
- end-2-end tests in place with a CSI plugin that can support Windows containers and pass all existing CSI plugin test scenarios.

#### Beta -> GA Graduation

- In-tree storage plugins that implements support for Windows (AWS EBS, GCE PD, Azure File and Azure Disk as of today) can use csi-proxy.exe along with other enhancements listed above to successfully deploy CSI plugins on Windows nodes.
- csi-proxy.exe supports v1 stable version of the CSI Proxy API Group.
- Successful usage of csi-proxy.exe with support for v1 version of CSI Proxy API Group in Windows nodes by at least two storage vendors.

### Upgrade / Downgrade Strategy

As explained in (#csi-proxy-grpc-api-graduation-and-deprecation-policy), CSI proxy will expose multiple API versions according to the API versioning policy, to facilitate drivers upgrading to a newer API version and multiple drivers using different API versions.

In order to install a CSI Node Plugin or upgrade to a version of a CSI Node Plugin that uses an updated version of the CSI Proxy API Group not supported by the currently deployed version of csi-proxy.exe in the cluster, csi-proxy.exe will need to be upgraded first on all nodes of the cluster before deploying or upgrading the CSI Node Plugin.

In case there is a very old CSI Node Plugin in the cluster that relies on a version of CSI Proxy API Group that is no longer supported by the new version of csi-proxy.exe, the previously installed version of csi-proxy.exe should not be uninstalled from the nodes. Such scenarios are expected to be an exception.

Different nodes in the cluster may be configured with different versions of csi-proxy.exe as part of a rolling upgrade of csi-proxy.exe. In such a scenario, it is recommended that csi-proxy.exe upgrade is completed first across all nodes. Once that is complete, the CSI Node Plugins that can take advantage of the new version of csi-proxy.exe may be deployed.

However, in case there is unavoidable case that CSI Node Plugin might get upgraded first before csi-proxy is upgraded, the driver must fall back to older versions if a newer API version of CSI proxy is not available. An example is [PD CSI driver implementation](https://github.com/kubernetes-sigs/gcp-compute-persistent-disk-csi-driver/pull/738)

Downgrading the version of csi-proxy.exe to one that is not supported by all installed versions the CSI Node Plugins in the cluster will lead to loss of access to data. Further, if a cluster is downgraded from a version of Kubernetes where the plugin watcher supports Windows nodes to one that does not, existing Windows workloads that were using CSI plugins to access storage will no longer have access to the data. This loss of functionality cannot be handled in an elegant fashion.

### Version Skew Strategy

Beyond the points in the above section (Upgrade/Downgrade strategy), there are no Kubernetes version skew considerations in the context of this KEP.

The minimum Kubernetes version to support CSI Windows is 1.18. While the API versions in CSI proxy is independent of Kubernetes versions, we recommend the following versions of CSI proxy and Kubernetes.

Status | Min K8s Version | Recommended K8s Version
--|--|--
v0.1.0 | 1.18 | 1.18
v0.2.0 | 1.18  | 1.19
v0.2.2+ | 1.18  | 1.21

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [x] Other
  - Describe the mechanism:
  CSI-proxy is a binary that can be deployed (or running as a service) directly on each Windows node. For example, in GCE, CSI-proxy
  binary is downloaded and run as a Windows service in node [startup script](https://github.com/kubernetes/kubernetes/blob/master/cluster/gce/windows/k8s-node-setup.psm1#L424)
  An environment variable ENABLE_CSI_PROXY is implemented to disable/enable CSI-proxy installation.
  - Will enabling / disabling the feature require downtime of the control
    plane?
  No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
  If csi-proxy binary is deployed as a binary or windows service during node start up time, one way to enable this feature is to reprovision of a node during cluster upgrade.
  System admin could potentically install and launch CSI-proxy after node is starting.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
It depends on Kubernetes distribution. For example in GCE, environment variable ENABLE_CSI_PROXY is implemented to disable/enable CSI-proxy installation. Note that nodes should be drained first before uninstalling csi-proxy. Otherwise, volumes mounted on the node cannot be property teared down if CSI-proxy is disabled.

###### What happens if we reenable the feature if it was previously rolled back?

Windows CSI volume operations will work again.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->
No
### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->
It is up to cloud provider to have their own plan. For example, GKE will use version control to rollout, upgrade and rollback this feature.
Please also see (#upgrade--downgrade-strategy)

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->
It is up to the cloud provider to tie CSI-proxy version to cluster version. When clusters rollout and rollback, the CSI-proxy should be updated accordingly.
The minimum version of Kubernetes required for CSI Windows support is 1.18

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
For detecting fleet-wide trends, volume_operation and csi_operation metrics exposed by kubelet on Windows node could be used for indicating issues on CSI-proxy.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
 Plan to investigate if the existing kubernetes statefulset upgrade test can also be leveraged for Windows.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->
This can be determined by monitoring the usage of CSI node plugin on Windows node, csi_operations_seconds.
The operation includes NodeStageVolume, NodePublishVolume, NodeUnstageVolume, NodeUnpublishVolume.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [x] Events
  - Event Reason: Windows Pods which are using the PVCs (using CSI Node plugin) can start.
- [ ] API .status
  - Condition name:
  - Other field:
- [ ] Other (treat as last resort)
  - Details: CSI-proxy also generates logs

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->
Volume/file operation latency might be different from different storage vendors.
- It takes a few minutes (in the range of 1~5mins) to format a disk for Windows.
- The rest of volume operations in the range of seconds.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Metrics
  - Metric name: Volume/file operation latency for CSI (csi_operations_seconds See details https://github.com/kubernetes/kubernetes/pull/98979/files)
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
We could enable prometheus for volume operations metrics in the future

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->
No. All the Windows OS versions supported by Kuberentes can support CSI-proxy.

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->
No

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->
Stress tests can performed through storage test cases such as [OSS E2E volume stress test](https://github.com/kubernetes/kubernetes/blob/v1.21.0/test/e2e/storage/testsuites/volume_stress.go)

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->
CSI-proxy provides a set of storage related APIs, but they are not part of kubernetes.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
In-tree plugin like PD/Azure-Disk/EBS that supports Windows today may use csi-proxy in the future due to CSI Migration. This could introduce higher latencies (such as node mount/initial format/etc) if the pods of the statefulset are targeting Windows and the pods mount PVs backed by CSI plugins using csi proxy.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
There are two aspects of resource usage in the context of CSI proxy

[a] specific latencies and extra resources consumed as a result of driving operations through CSI plugin using CSI proxy vs. In-tree plugin with Support for Windows:

The preliminary investigation using GCE PD CSI driver on this is available [link](https://docs.google.com/document/d/10aXLjJs8HvloY0zQJaMsOMn0S0AyrVyfh8iC9T7A1yg/edit). There are relatively small extra delay caused by CSI proxy for certain volume operations. There will be future performance investigation and improvement work on this area.

[b] the overall resources associated with running operations (such as list disk/format volumes/etc) associated with CSI plugins vs not running CSI plugins on Windows?

The tests performed on this area has not shown noticeable overhead. Plan to test on larger scale in the future.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

CSI Proxy is a thin stateless API layer between CSI node plugins (running on Windows) and Windows OS APIs (surfaced through Win32 or Powershell cmdlets). Troubleshooting of APIs associated with CSI Proxy should be performed using either logs from the CSI plugin or Windows OS event logs."

###### How does this feature react if the API server and/or etcd is unavailable?

No impact

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->
1. CSI Proxy binary is not installed and running
2. CSI-proxy API version is smaller than CSI Node Plugin requested CSI-proxy API version

For both cases, The error message from CSI Node plugin will show, for example
```
"transport: Error while dialing open \\\\.\\\\pipe\\\\csi-proxy-disk-v1beta2: The system cannot find the file specified."
```
This error message will also be shown in events of Pods which are trying to use the CSI volume.

###### What steps should be taken if SLOs are not being met to determine the problem?


## Implementation History

07/16/2019: Initial KEP drafted

07/20/2019: Feedback from initial KEP review addressed.

05/03/2021: KEP is updated for GA

## Drawbacks

The main drawback associated with the approach leveraging csi-proxy.exe is that the life cycle of that binary as well as logs will need to be managed out-of-band from Kubernetes. However, cluster administrators need to maintain and manage life cycle and logs of other core binaries like kubeproxy.exe, kubelet.exe, dockerd.exe and containerd.exe (in the future). Therefore csi-proxy.exe will be one additional binary that will need to be treated in a similar way.

The API versioning scheme described above, will try to maintain backward compatibility as much as possible. This requires the scope of csi-proxy.exe to be limited to a very scoped down fundamental set of operations. Maintainers therefore will need to be very cautious when accepting suggestions for new APIs and enhancements. This may slow progress at times.

There may ultimately be certain operations that csi-proxy.exe cannot support in an elegant fashion and require the plugin author targeting Windows nodes to seek one of the alternatives described below. There may also be a need to support volumes that do not use standard block or file protocols. In such scenarios, an extra payload (in the form of a binary, kernel driver and service) may need to be dropped on the host and maintained out-of-band from Kubernetes. This KEP and maintainers should ensure such instances are as limited as possible.

## Alternatives

There are alternative approaches to the CSI Proxy API Group as well as the overall csi-proxy mechanism described in this KEP. These alternatives are enumerated below.

### API Alternatives

The CSI Proxy API Group will be a defined set of operations that will need to expand over time as new CSI APIs are introduced that require new operations on every node as well as desire for richer operations by CSI plugin authors. Unfortunately this comes with a maintenance burden associated with tracking and graduating new RPCs across versions.

An alternative approach that simplifies the above involves exposing a single Exec API that supports passing an arbitrary set of parameters to arbitrary executables and powershell cmdlets on the Windows host and collecting and returning the stdout and stderr back to the invoking containerized process. Since all the currently enumerated operations in the CSI Proxy API Group can be driven through the generic Exec interface, the set of desired privileged operations necessary becomes a decision for plugin authors rather than maintainers of csi-proxy. The main drawback of this highly flexible mechanism is that it drastically increases the potential for abuse in the host by 3rd party plugin authors. The ability to exploit bugs or vulnerabilities in individual plugins to take control of the host becomes much more trivial with a generic Exec RPC relative to exploiting other RPCs of the CSI Proxy API Group.

Depending on the adoption of csi-proxy in the Alpha and Beta phases and the need for specialized privileged operations, we may consider adding a generic Exec interface in the future.

### Deployment Alternatives

There are multiple alternatives to deploying CSI Node Plugins as containers along with csi-proxy.exe for privileged operations however each has it's own set of disadvantages.

#### Deploying CSI Node Plugins as binaries and deployed as processes running on the host:

With this approach, lifecycle of multiple stand-alone binaries corresponding to different CSI node plugins will need to be managed. The standard CSI Node Driver Registrar which is distributed as a container will also need to be repackaged as binaries and distributed (as mixing side car containers with standalone binaries is not possible). Managing several binaries outside of Kubernetes may not scale well as diverse storage systems, each with their own CSI plugin, is integrated with the cluster. Since Kubernetes has no knowledge of these binaries, operators will not be able to use standard Kubernetes constructs to monitor and control the life-cycle of the binaries. Collection of logs from the stand-alone binaries will require tooling out-of-band from Kubernetes.

#### Package CSI Node Plugins as containers and deployed as processes running on the host:

With this approach, the container run time is enhanced to be able to launch binaries directly on the host after pulling down a container image and extracting the binary from the image. While usual Kubernetes constructs may be used to launch pods with a special RuntimeClass that can handle launching of the binaries as processes on hosts, various enhancements will be necessary in the runtime to enable Kubernetes to fully monitor and control the whole life-cycle of the binaries post launch. Collection of logs from the plugins also become problematic and will require either out-of-band tooling at present or various enhancements in the runtime in the future.

#### Support for Privileged Operations and Bi-directional mount propagation in Windows containers:

At some point in the future, a Windows LTS release may implement support for execution of privileged operations and bi-directional mount propagation from inside containers. At that point, the requirement of a proxy to handle privileged operations on behalf of containers will disappear. However, before such support is committed to and implemented in a Windows LTS release (which is not expected in at least a year), we need solutions as described in the KEP.

## Infrastructure Needed

The code for csi-proxy as well as the GRPC API will be maintained in a dedicated repo: github.com/kubernetes-csi/windows-csi-proxy
