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
      - [CSI Proxy GRPC API](#csi-proxy-grpc-api)
      - [CSI Proxy GRPC API Graduation and Deprecation Policy](#csi-proxy-grpc-api-graduation-and-deprecation-policy)
      - [CSI Proxy Event Logs](#csi-proxy-event-logs)
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
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
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
            - name: csi-proxy-pipe
              mountPath: \\.\pipe\csi-proxy-v1alpha1
      volumes:
        - name: csi-proxy-pipe
          hostPath: 
            path: \\.\pipe\csi-proxy-v1alpha1
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
A CSI Node Plugin will interact with csi-proxy.exe using named pipe: `\\.\pipe\csi-proxy-v[N]` (exposed by csi-proxy.exe). The `v[N]` suffix in the pipe name corresponds to the version of the CSIProxyService (described below) that is required by the CSI plugin. Specific example of named pipes corresponding to versions include: `\\.\pipe\csi-proxy-v1`, `\\.\pipe\csi-proxy-v2alpha1`, `\\.\pipe\csi-proxy-v3beta1`, etc. The pipe will need to be mounted into the Node Plugin container from the host using the pod's volume mount specifications. Note that domain sockets cannot be used in this scenario since Windows blocks a containerized process from interacting with a host process that is listening on a domain socket. If such support is enabled on Windows in the future, we may consider switching CSI Proxy to use domain sockets however Windows named pipes is a common mechanism used by containers today to interact with host processes like docker daemon.

![Kubernetes CSI Node Components and Interactions](csi-proxy3.png?raw=true "Kubernetes CSI Node Components and Interactions")

A GRPC based interface, CSIProxyService, will be used by CSI Node Plugins to invoke privileged operations on the host through csi-proxy.exe. CSIProxyService will be versioned and any release of csi-proxy.exe binary will strive to maintain backward compatibility across as many prior stable versions of the API as possible. This avoids having to run multiple versions of the csi-proxy.exe binary on the same Windows host if multiple CSI Node Plugins (that do not depend on APIs in Alpha or Beta versions) have been configured and the version of the csi-proxy API required by the plugins are different. For every version of the API supported, csi-proxy.exe will first probe for and then expose a `\\.\pipe\csi-proxy-v[N]` pipe where v[N] can be v1, v2alpha1, v3beta1, etc. If during the initial probe phase, csi-proxy.exe determines that another process is already listening on a `\\.\pipe\csi-proxy-v[N]` named pipe, it will not try to create and listen on that named pipe. This allows multiple versions of csi-proxy.exe to run side-by-side if absolutely necessary to support multiple CSI Node Plugins that require widely different versions of CSIProxyService that no single version of csi-proxy.exe can support.

##### CSI Proxy Configuration
There will be two command line parameters that may be passed to csi-proxy.exe:
1. kubelet-csi-plugins-path: String parameter pointing to the path used by Kubernetes CSI plugins on each node. Will default to: `C:\var\lib\kubelet\plugins\kubernetes.io\csi`. All requests for creation and deletion of paths through the CSIProxyService RPCs (detailed below) will need to be under this path.

2. kubelet-pod-path: String parameter pointing to the path used by the kubelet to store pod specific information. This should map to the value returned by [getPodsDir](https://github.com/kubernetes/kubernetes/blob/e476a60ccbe25581f5a6a9401081dcee311a066e/pkg/kubelet/kubelet_getters.go#L48). By default it will be set to: `C:\var\lib\kubelet\pods` Parameters to `LinkPath` (detailed below) will need to be under this path.

##### CSI Proxy GRPC API
The following are the main RPC calls that will comprise a v1alpha1 version of the CSIProxyService API. A preliminary structure for Request and Response associated with each RPC is described below. Note that the specific structures as well as restrictions on them are expected to evolve during Alpha phase and are expected to be in a final form at the end of Beta. As the API evolves, the section below will be kept up-to-date

```
service CSIProxyService {

    // PathExists checks if the given path exists in the host already
    rpc PathExists(PathExistsRequest) returns (PathExistsResponse) {}

    // Mkdir creates a directory at the requested absolute path in the host. Relative path is not supported.
    rpc Mkdir(MkdirRequest) returns (MkdirResponse) {}

    // Rmdir removes the directory at the requested absolute path in the host. Relative path is not supported.
    // This may be used for unlinking a symlink created through LinkVolume
    rpc Rmdir(RmdirRequest) returns (RmdirResponse) {}

    // Rescan refreshes the host storage cache
    rpc Rescan(RescanRequest) returns (RescanResponse) {}

    // PartitionDisk initializes and partitions a disk device (if the disk has not
    // been partitioned already) and returns the resulting volume object
    rpc PartitionDisk(PartitionDiskRequest) returns (PartitionDiskResponse) {}

    // FormatVolume formats a volume with the provided file system.
    // The resulting volume is mounted at the requested global staging path.
    rpc FormatVolume(FormatVolumeRequest) returns (FormatVolumeResponse) {}

    // MountSMBShare mounts a remote share over SMB on the host at the requested global staging path.
    rpc MountSMBShare(MountSMBShareRequest) returns (MountSMBShareResponse) {}

    // MountISCSILun mounts a remote LUN over iSCSI and returns the OS disk device number.
    rpc MountISCSILun(MountISCSILunRequest) returns (MountISCSILunResponse) {}

    // LinkPath invokes mklink on the global staging path of a volume linking it to a path within a container
    rpc LinkPath(LinkPathRequest) returns (LinkPathResponse) {}

    // ListDiskLocations returns locations <Adapter, Bus, Target, LUN ID> of all disk devices enumerated by Windows
    rpc ListDiskLocations(ListDiskLocationsRequest) returns (ListDiskLocationsResponse) {}

    // ListDiskIDs returns all IDs (from IOCTL_STORAGE_QUERY_PROPERTY) of all disk devices enumerated by Windows
    rpc ListDiskIDs(GetDiskIDsRequest) returns (ListDiskIDsResponse) {}

    // ListDiskVolumeMappings returns a map of all disk devices and volumes GUIDs
    rpc ListDiskVolumeMappings(ListDiskVolumeMappingsRequest) returns (ListDiskVolumeMappingsResponse) {}

    // ResizeVolume performs resizing of the partition and file system for a block based volume
    rpc ResizeVolume(ResizeVolumeRequest) returns (ResizeVolumeResponse) {}

    // DismountVolume gracefully dismounts a volume
    rpc DismountVolume(DismountVolumeRequest) return (DismountVolumeResponse) {}
}

message PathExistsRequest {
    // The path to check in the host filesystem.
    string path = 1;
}

message PathExistsResponse {
    // Whether path already exists in host or not
    bool exists = 1;
}

// Context of the paths used for path prefix validation
enum PathContext {
    // plugin maps to the configured kubelet-csi-plugins-path path prefix
    PLUGIN = 0;
    // container maps to the configured kubelet-pod-path path prefix
    CONTAINER = 1;
}

message MkdirRequest {
    // The path to create in the host filesystem.
    // All special characters allowed by Windows in path names will be allowed
    // except for restrictions noted below. For details, please check:
    // https://docs.microsoft.com/en-us/windows/win32/fileio/naming-a-file
    // Non-existent parent directories in the path will NOT be created.
    // Directories will be created with Read and Write privileges of the Windows
    // User account under which csi-proxy is started (typically LocalSystem).
    //
    // Restrictions:
    // Needs to be an absolute path under kubelet-csi-plugins-path
    // or kubelet-pod-path based on context
    // Cannot exist already in host
    // Path needs to be specified with drive letter prefix: "X:\".
    // UNC paths of the form "\\server\share\path\file" are not allowed.
    // All directory separators need to be backslash character: "\".
    // Characters: .. / : | ? * in the path are not allowed.
    // Maximum path length will be capped to 260 characters (MAX_PATH).
    string path = 1;

    // Context of the path creation used for path prefix validation
    PathContext context = 2;
}

message MkdirResponse {
    // Windows error code
    // Success is represented as 0
    int32 error_code = 1;
}

message RmdirRequest {
    // The path to remove in the host filesystem
    // All special characters allowed by Windows in path names will be allowed
    // except for restrictions noted below. For details, please check:
    // https://docs.microsoft.com/en-us/windows/win32/fileio/naming-a-file
    //
    // Restrictions:
    // Needs to be an absolute path under kubelet-csi-plugins-path
    // or kubelet-pod-path based on context
    // Path needs to be specified with drive letter prefix: "X:\".
    // UNC paths of the form "\\server\share\path\file" are not allowed.
    // All directory separators need to be backslash character: "\".
    // Characters: .. / : | ? * in the path are not allowed.
    // Path cannot be a file of type symlink
    // Path needs to be a directory that is empty
    // Maximum path length will be capped to 260 characters (MAX_PATH).
    string path = 1;

    // Context of the path creation used for path prefix validation
    PathContext context = 2;
}

message RmdirResponse {
    // Windows error code
    // Success is represented as 0
    int32 error_code = 1;
}

message RescanRequest {
    // Intentionally empty.
}

message RescanResponse {
    // Windows error code
    // Success is represented as 0
    int32 error_code = 1;
}

message PartitionDiskRequest {
    // The Windows disk device to partition and the paritioning mode: MBR/GPT.
    // The whole disk will be partitioned
    //
    // Restrictions:
    // Disk device needs to follow Win32 format for device names: \\.\PhysicalDriveX
    // The prefix has to be: \\.\PhysicalDrive and the suffix is an integer in range:
    // 0 to maximum number drives allowed by Windows OS.
    string disk_device = 1;

    // Disk partition type
    enum ParitionType {
        MBR = 0;
        GPT = 1;
    }
    PartitionType type = 2;
}

message PartitionDiskResponse {
    // Volume device resulting from the partition
    // Volume device will follow Win32 namespaced GUID format for volumes: \\?\Volume\{GUID}
    // The prefix has to be: \\?\Volume\ and the suffix to be a GUID enclosed with {}
    string volume_device = 1;

    // Windows error code
    // Success is represented as 0
    int32 error_code = 2;
}

message FormatVolumeRequest {
    // The Windows volume device to format
    // Typically Volume Device returned by PartitionDiskResponse
    //
    // Restrictions:
    // Volume device needs to follow Win32 namespaced GUID format for volumes: \\?\Volume\{GUID}
    // The prefix has to be: \\?\Volume\ and the suffix to be a GUID enclosed with {}
    string volume_device = 1;

    // FileSystem type
    enum FileSystemType {
        NTFS = 0;
        FAT = 1;
    }
    FileSystemType type = 2;
}

message FormatVolumeResponse {
    // Windows error code
    // Success is represented as 0
    int32 error_code = 1;
}

message MountSMBShareRequest {
    // A remote SMB share to mount
    // All unicode characters allowed in SMB server name specifications are
    // permitted except for restrictions below
    //
    // Restrictions:
    // SMB share specified in the format: \\server-name\sharename, \\server.fqdn\sharename or \\a.b.c.d\sharename
    // If not an IP address, share name has to be a valid DNS name.
    // UNC specifications to local paths or prefix: \\?\ is not allowed.
    // Characters: + [ ] " / : ; | < > , ? * = $ are not allowed.
    string remote_share = 1;

    // Local path in the host to stage the SMB share.
    // All special characters allowed by Windows in path names will be allowed
    // except for restrictions noted below. For details, please check:
    // https://docs.microsoft.com/en-us/windows/win32/fileio/naming-a-file
    //
    // Restrictions:
    // Needs to be an absolute path under kubelet-csi-plugins-path.
    // Needs to exist already in host
    // Path needs to be specified with drive letter prefix: "X:\".
    // UNC paths of the form "\\server\share\path\file" are not allowed.
    // All directory separators need to be backslash character: "\".
    // Characters: .. / : | ? * in the path are not allowed.
    // Maximum path length will be capped to 260 characters (MAX_PATH).
    string host_path = 2;

    // Mount the share read-only
    bool readonly = 3;

    // Username credential associated with the share
    string username = 4;

    // Password credential associated with the share
    string password = 5;
}

message MountSMBShareResponse {
    // Windows error code
    // Success is represented as 0
    int32 error_code = 1;
}

message MountISCSILunRequest {
    // IQN address
    // follows IQN format: iqn.yyyy-mm.naming-authority:unique name
    string node_iqn = 1;

    // Authentication Type
    enum AuthType {
        None = 0;
        OneWay = 1;
        Mutual = 2;
    }
    AuthType auth_type = 2;

    // Discovery CHAP username
    string discovery_chap_username = 3;

    // Discovery CHAP secret
    string discovery_chap_secret = 4;

    // Session CHAP username
    string session_chap_username = 5;

    // Session CHAP secret
    string session_chap_secret = 6;

    // TargetPortal address
    string target_portal_address = 7;

    // TargetPortal port
    string target_portal_port = 8;

    // Readonly mount
    bool readonly = 9;
}

message MountISCSILunResponse {
    // Windows error code
    // Success is represented as 0
    int32 error_code = 1;
}

message LinkPathRequest {
    // Source of MkLink call to Windows
    // All special characters allowed by Windows in path names will be allowed
    // except for restrictions noted below. For details, please check:
    // https://docs.microsoft.com/en-us/windows/win32/fileio/naming-a-file
    //
    // Restrictions:
    // Needs to be an absolute path under kubelet-pod-path
    // Needs to exist already in host
    // Path needs to be specified with drive letter prefix: "X:\".
    // UNC paths of the form "\\server\share\path\file" are not allowed.
    // All directory separators need to be backslash character: "\".
    // Characters: .. / : | ? * in the path are not allowed.
    // Maximum path length will be capped to 260 characters (MAX_PATH).
    string source_path = 1;

    // Target of MkLink call to Windows
    // All special characters allowed by Windows in path names will be allowed
    // except for restrictions noted below. For details, please check:
    // https://docs.microsoft.com/en-us/windows/win32/fileio/naming-a-file
    //
    // Restrictions:
    // Needs to be an absolute path under kubelet-csi-plugins-path.
    // Needs to exist already in host
    // Path needs to be specified with drive letter prefix: "X:\".
    // UNC paths of the form "\\server\share\path\file" are not allowed.
    // All directory separators need to be backslash character: "\".
    // Characters: .. / : | ? * in the path are not allowed.
    // Maximum path length will be capped to 260 characters (MAX_PATH).
    string target_path = 2;
}

message LinkPathResponse {
    // Windows error code
    // Success is represented as 0
    int32 error_code = 1;
}

message ListDiskLocationsRequest {
    // Intentionally empty
}

message ListDiskLocationsResponse {
    // Map of disk device objects and <adapter, bus, target, lun ID> associated with each disk device
    map <string, DiskLocation> disk_locations = 1;

    // Windows error code
    // Success is represented as 0
    int32 error_code = 2;
}

message DiskLocation {
    string Adapter = 0;
    string Bus = 1;
    string Target = 2;
    string LUNID = 3;
}

message ListDiskIDsRequest {
    // Intentionally empty
}

message ListDiskIDsResponse {
    // Map of disk device objects and IDs associated with each disk device
    map <string, DiskIDs> disk_id = 1;

    // Windows error code
    // Success is represented as 0
    int32 error_code = 2;
}

message DiskIDs {
    // list of Disk IDs of ascii characters associated with disk device
    repeated strings IDs = 1;
}

message ListDiskVolumeMappingsRequest {
    // Intentionally empty
}

message ListDiskVolumeMappingsResponse {
    // Map of disk devices and volume objects of the form \\?\volume\{GUID} on the disk
    map <string, string> disk_volume_pair = 1;

    // Windows error code
    // Success is represented as 0
    int32 error_code = 2;
}

message ResizeVolumeRequest {
    // The Win32 volume device to resize
    //
    // Restrictions:
    // Volume device needs to follow Win32 namespaced GUID format for volumes: \\?\Volume\{GUID}
    // The prefix has to be: \\?\Volume\ and the suffix to be a GUID enclosed with {}
    string volume_device = 1;

    // New size to resize FS to
    int64 new_size = 2;
}

message ResizeVolumeResponse {
    // Windows error code
    // Success is represented as 0
    int32 error_code = 1;
}

message DismountVolumeRequest {
    // The Win32 volume device to dismount gracefully
    //
    // Restrictions:
    // Volume device needs to follow Win32 namespaced GUID format for volumes: \\?\Volume\{GUID}
    // The prefix has to be: \\?\Volume\ and the suffix to be a GUID enclosed with {}
    string volume_device = 1;
}

message DismountVolumeResponse {
    // Windows error code
    // Success is represented as 0
    int32 error_code = 1;
}

```

##### CSI Proxy GRPC API Graduation and Deprecation Policy

In accordance with standard Kubernetes conventions, the above API will be introduced as v1alpha1 and graduate to v1beta1 and v1 as the feature graduates. Beyond a vN release in the future, new RPCs and enhancements to parameters will be introduced through vN+1alpha1 and graduate to vN+1beta1 and vN+1 stable versions as the new APIs mature.

Members of CSIProxyService API may be deprecated and then removed from csi-proxy.exe in a manner similar to Kubernetes deprecation (policy)[https://kubernetes.io/docs/reference/using-api/deprecation-policy/] although maintainers will make an effort to ensure such deprecation is as rare as possible. After their announced deprecation, a member of CSIProxyService API must be supported:
1. 12 months or 3 releases (whichever is longer) if the API member is part of a Stable/vN version.
2. 9 months or 3 releases (whichever is longer) if the API member is part of a Beta/vNbeta1 version.
3. 0 releases if the API member is part of an Alpha/vNalpha1 version.

To continue running CSI Node Plugins that depend on an old version of csi-proxy.exe, vN, some of whose members have been removed, Kubernetes administrators will be required to run the latest version of the csi-proxy.exe (that will be used by CSI Node Plugins that use versions of CSIProxyService more recent than vN) along with an old version of csi-proxy.exe that does support vN.

Introduction of new RPCs or enhancements to parameters is expected to be inspired by new requirements from plugin authors as well as CSI functionality enhancement.

##### CSI Proxy Event Logs

For all RPC invocations, csi-proxy.exe will log events to the Windows application event log. This will act as an audit trail that may be correlated with audit logs from Kubeneretes around operations involving PV creation to track potential unexpected invocation of CSIProxyService by a malicious pod or process.

#### Enhancements in Kubernetes/Utils/mounter

Once the [PR](https://github.com/kubernetes/utils/pull/100/files) lands, a mounter/mount_windows_using_csi_proxy.go in Kubernetes/Utils/mounter package can be introduced. It will implement the mounter and hostutil interfaces against the CSIProxyService API.

#### Enhancements in CSI Node Plugins

Code for CSI Node Plugins need to be refactored to support CSI Node APIs in both Linux and Windows nodes. While the code targeting Linux nodes can assume privileged access to the host, the code targeting Windows nodes need to invoke the GRPC client API associated with the desired version of the CSIProxyService described above. CSI Node Plugins that will use the Kubernetes/Utils/mounter package introduced in this [PR](https://github.com/kubernetes/utils/pull/100/files) will require minimal platform specific code targeting Windows and Linux.

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

- csi-proxy.exe supports v1beta1 version of the CSIProxyService API.
- end-2-end tests in place with a CSI plugin that can support Windows containers and pass all existing CSI plugin test scenarios.

#### Beta -> GA Graduation

- In-tree storage plugins that implements support for Windows (AWS EBS, GCE PD, Azure File and Azure Disk as of today) can use csi-proxy.exe along with other enhancements listed above to successfully deploy CSI plugins on Windows nodes.
- csi-proxy.exe supports v1 stable version of the CSIProxyService API.
- Successful usage of csi-proxy.exe with support for v1 version of CSIProxyService API in Windows nodes by at least two storage vendors.

### Upgrade / Downgrade Strategy

In order to install a CSI Node Plugin or upgrade to a version of a CSI Node Plugin that uses an updated version of the CSIProxyService API not supported by the currently deployed version of csi-proxy.exe in the cluster, csi-proxy.exe will need to be upgraded first on all nodes of the cluster before deploying or upgrading the CSI Node Plugin. In case there is a very old CSI Node Plugin in the cluster that relies on a version of CSIProxyService API that is no longer supported by the new version of csi-proxy.exe, the previously installed version of csi-proxy.exe should not be uninstalled from the nodes. Such scenarios are expected to be an exception.

Different nodes in the cluster may be configured with different versions of csi-proxy.exe as part of a rolling upgrade of csi-proxy.exe. In such a scenario, it is recommended that csi-proxy.exe upgrade is completed first across all nodes. Once that is complete, the CSI Node Plugins that can take advantage of the new version of csi-proxy.exe may be deployed.

Downgrading the version of csi-proxy.exe to one that is not supported by all installed versions the CSI Node Plugins in the cluster will lead to loss of access to data. Further, if a cluster is downgraded from a version of Kubernetes where the plugin watcher supports Windows nodes to one that does not, existing Windows workloads that were using CSI plugins to access storage will no longer have access to the data. This loss of functionality cannot be handled in an elegant fashion.

### Version Skew Strategy

Beyond the points in the above section (Upgrade/Downgrade strategy), there are no Kubernetes version skew considerations in the context of this KEP.

## Implementation History

07/16/2019: Initial KEP drafted

07/20/2019: Feedback from initial KEP review addressed.

## Drawbacks

The main drawback associated with the approach leveraging csi-proxy.exe is that the life cycle of that binary as well as logs will need to be managed out-of-band from Kubernetes. However, cluster administrators need to maintain and manage life cycle and logs of other core binaries like kubeproxy.exe, kubelet.exe, dockerd.exe and containerd.exe (in the future). Therefore csi-proxy.exe will be one additional binary that will need to be treated in a similar way.

The API versioning scheme described above, will try to maintain backward compatibility as much as possible. This requires the scope of csi-proxy.exe to be limited to a very scoped down fundamental set of operations. Maintainers therefore will need to be very cautious when accepting suggestions for new APIs and enhancements. This may slow progress at times.

There may ultimately be certain operations that csi-proxy.exe cannot support in an elegant fashion and require the plugin author targeting Windows nodes to seek one of the alternatives described below. There may also be a need to support volumes that do not use standard block or file protocols. In such scenarios, an extra payload (in the form of a binary, kernel driver and service) may need to be dropped on the host and maintained out-of-band from Kubernetes. This KEP and maintainers should ensure such instances are as limited as possible.

## Alternatives

There are alternative approaches to the CSIProxyService API as well as the overall csi-proxy mechanism described in this KEP. These alternatives are enumerated below.

### API Alternatives

The CSIProxyService API will be a defined set of operations that will need to expand over time as new CSI APIs are introduced that require new operations on every node as well as desire for richer operations by CSI plugin authors. Unfortunately this comes with a maintenance burden associated with tracking and graduating new RPCs across versions.

An alternative approach that simplifies the above involves exposing a single Exec interface in CSIProxyService that supports passing an arbitrary set of parameters to arbitrary executables and powershell cmdlets on the Windows host and collecting and returning the stdout and stderr back to the invoking containerized process. Since all the currently enumerated operations in the CSIProxyService API can be driven through the generic Exec interface, the set of desired privileged operations necessary becomes a decision for plugin authors rather than maintainers of csi-proxy. The main drawback of this highly flexible mechanism is that it drastically increases the potential for abuse in the host by 3rd party plugin authors. The ability to exploit bugs or vulnerabilities in individual plugins to take control of the host becomes much more trivial with a generic Exec RPC relative to exploiting other RPCs of the CSIProxyService API.

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
