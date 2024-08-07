# KEP-4743: The Kubernetes-etcd interface
<!-- toc -->
- [Summary](#summary)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [User Stories](#user-stories)
  - [Kubernetes Backporting an etcd client patch version](#kubernetes-backporting-an-etcd-client-patch-version)
  - [Making Changes/Patching Bugs in the Interface](#making-changespatching-bugs-in-the-interface)
  - [Kubernetes Leveraging New etcd Functionality](#kubernetes-leveraging-new-etcd-functionality)
- [Code location](#code-location)
- [The Interface](#the-interface)
  - [KV interface](#kv-interface)
  - [Design considerations](#design-considerations)
  - [Watch interface](#watch-interface)
  - [Design considerations](#design-considerations-1)
- [Alternatives](#alternatives)
  - [Code location](#code-location-1)
    - [Part of the etcd Client Struct](#part-of-the-etcd-client-struct)
    - [New Package in etcd Repository](#new-package-in-etcd-repository)
    - [New Repository under etcd-io](#new-repository-under-etcd-io)
<!-- /toc -->

## Summary 

This design proposal introduces an etcd-Kubernetes interface to be added to the 
etcd client and adopted by Kubernetes. This interface aims to create a clear and
standardized contract between the two projects, codifying the interactions
outlined in the [Implicit Kubernetes-ETCD Contract]. By formalizing this contract,
we will improve the testability of both Kubernetes and etcd, prevent common
errors in their interaction, and establish a framework for the future evolution 
of this critical contract.

[Implicit Kubernetes-ETCD Contract]: https://docs.google.com/document/d/1NUZDiJeiIH5vo_FMaTWf0JtrQKCx0kpEaIIuPoj9P6A/edit#heading=h.tlkin1a8b8bl

### Goals

* **Improved Testability:** Enable thorough testing of etcd and Kubernetes 
  interactions through a well-defined interface, as envisioned in [#15820].
* **Error Prevention:** Reduce incorrect contract usage, addressing issues like Kubernetes [#110210].
* **Reviewable Changes:** Make contract modifications easily reviewable and
  trackable, ensuring a transparent and collaborative evolution.
* **Backward Compatibility:** Ensure the interface remains compatible with all
  etcd versions supported by Kubernetes at the time of a Kubernetes release.

[#15820]: https://github.com/etcd-io/etcd/issues/15820
[#110210]: https://github.com/kubernetes/kubernetes/issues/110210

In scope
* [etcd3 store]: The primary Kubernetes object storage interface.
* [Master leases]: Lease management for Kubernetes control plane components (utilizing the [etcd3 store])

[etcd3 store]: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/storage/etcd3/store.go
[Master leases]: https://github.com/kubernetes/kubernetes/blob/dae1859c896d742de1ee60a349475f8e28b61995/pkg/controlplane/reconcilers/lease.go#L47-L66

### Non-Goals
* **Alternative Interface Implementations:** This KEP focuses solely on defining
  the interface for the existing etcd backend and ensuring its compatibility
  with Kubernetes. It does not encompass the development or support of
  alternative storage backends or implementations for the interface.
* **Non-Storage Usage of etcd Client:**
  * [Kubeadm] - Primarily used for etcd cluster administration, not Kubernetes object storage.
  * [Compaction] - Kubernetes aims to encourage native etcd compaction. See [#80513].
  * [Monitor] - Don’t see benefits of standardizing etcd metrics used for Kubernetes, at least for now.
  * [Prober], [Feature checker] - These have planned migrations to native etcd features. See [Design Doc: etcd livez and readyz probes] and [KEP-4647].
  * [Lease manager] - Planned removal in favor of one lease per key to address [#110210].

[Kubeadm]: https://github.com/kubernetes/kubernetes/blob/6ba9fa89fb5889550649bfde847c742a55d3d29c/cmd/kubeadm/app/util/etcd/etcd.go#L66-L93
[Compaction]: https://github.com/kubernetes/kubernetes/blob/6ba9fa89fb5889550649bfde847c742a55d3d29c/staging/src/k8s.io/apiserver/pkg/storage/etcd3/compact.go#L133-L162
[#80513]: https://github.com/kubernetes/kubernetes/issues/80513
[Monitor]: https://github.com/kubernetes/kubernetes/blob/6ba9fa89fb5889550649bfde847c742a55d3d29c/staging/src/k8s.io/apiserver/pkg/storage/storagebackend/factory/etcd3.go#L270-L283
[Prober]: https://github.com/kubernetes/kubernetes/blob/6ba9fa89fb5889550649bfde847c742a55d3d29c/staging/src/k8s.io/apiserver/pkg/storage/storagebackend/factory/etcd3.go#L256-L268
[Feature checker]: https://github.com/kubernetes/kubernetes/blob/6ba9fa89fb5889550649bfde847c742a55d3d29c/staging/src/k8s.io/apiserver/pkg/storage/feature/feature_support_checker.go#L143-L151
[Design Doc: etcd livez and readyz probes]: https://docs.google.com/document/d/1SkzmO4RT_GI9YhT0dw4a6nEwKVCciwrwbCDxK0D7ASM/edit?usp=sharing
[KEP-4647]: https://github.com/kubernetes/enhancements/pull/4662
[Lease manager]: https://github.com/kubernetes/kubernetes/blob/6ba9fa89fb5889550649bfde847c742a55d3d29c/staging/src/k8s.io/apiserver/pkg/storage/etcd3/lease_manager.go#L90-L120
[#110210]: https://github.com/kubernetes/kubernetes/issues/110210

## Proposal

This KEP proposes creating an etcd-Kubernetes code interface owned and
maintained by SIG-etcd. The interface will serve as a formalization of the
existing etcd-Kubernetes contract, ensuring the correct usage of etcd within
Kubernetes and enabling improved testing and validation.

The interface will prioritize etcd's existing capabilities and behaviors,
focusing on compatibility with the current etcd API. It will not introduce
features or behaviors not supported by etcd, adhering to the existing
SIG API Machinery policy outlined in [Storage for Extension API Servers].
This policy designates etcd as the sole supported storage backend for Kubernetes
for the foreseeable future.

[Storage for Extension API Servers]: https://docs.google.com/document/d/1i0xzRFB-uGLmLYueLMBTpHrOot9ScFxpkkcVcZHVbyA/edit?usp=sharing]

## User Stories

To better understand the importance of code location let’s visit the following use cases:

### Kubernetes Backporting an etcd client patch version

**The Journey:** Kubernetes regularly updates to newer etcd versions to leverage
bug fixes, or security patches. However, ensuring compatibility between the 
codified etcd-Kubernetes interface and etcd client is essential.

**Considerations:**

*   Even minor etcd client updates might inadvertently introduce changes that 
    break the interface's assumptions or functionality.
*   Tight coupling to the etcd client could necessitate backporting the 
    interface to older etcd branches, a complex and time-consuming process.


### Making Changes/Patching Bugs in the Interface

**The Journey:** Despite careful design, the complex etcd-Kubernetes contract 
might reveal bugs or require adjustments.

**Considerations:**

*   Changes and bug fixes need to be implemented and released with minimal
    disruption to both Kubernetes and etcd users.
*   Tightly coupled interface might require a full etcd
    client release for bug fixes, slowing down the process.

### Kubernetes Leveraging New etcd Functionality

**The Journey:** Kubernetes wants to expand its use of the etcd API beyond the
current interface's scope.

**Considerations:** The interface codifies a minimal subset of the etcd API
currently used by Kubernetes, and new features will initially be outside its
scope. Balancing new feature adoption with interface stability is crucial.

**Mitigation:**

*   Allow Kubernetes to directly use new etcd client features during alpha/beta
    stages, bypassing the interface temporarily.
*   Extending etcd robustness test to cover the new functionality before
    formalizing them in interface.
*   Once a feature is mature and stable, extend the interface, ensuring backward
    compatibility for existing Kubernetes versions.

## Code location

We propose locating the interface in a `kubernetes` subdirectory under
https://github.com/etcd-io/etcd/tree/main/client/v3. 
This approach allows for seamless integration with the etcd client while 
maintaining a dedicated space for the interface code.

Interface will be part of the etcd client package and it's release will be
combined with etcd release. For immediate Kubernetes use etcd will backport the 
client to `release-3.5` branch and introduce it in next etcd patch release for 
Kubernetes to consume.

Alternative code locations are discussed at the end of the document.

## The Interface

To ensure smoother transition we propose the adoption of the etcd-Kubernetes interface to be done in two stages:

1. **KV Interface:** Covering the basic get, list, count, put, and delete operations.
2. **Watch Interface: **Covering Watch operation and requesting progress notification for it.


### KV interface

For the reasoning please see the section below.

```
// Interface defines the minimal client-side interface that Kubernetes requires
// to interact with etcd. Methods below are standard etcd operations with
// semantics adjusted to better suit Kubernetes' needs.
type Interface interface {
	// Get retrieves a single key-value pair from etcd.
	//
	// If opts.Revision is set to a non-zero value, the key-value pair is retrieved at the specified revision.
	// If the required revision has been compacted, the request will fail with ErrCompacted.
	Get(ctx context.Context, key string, opts GetOptions) (GetResponse, error)

	// List retrieves key-value pairs with the specified prefix.
	//
	// If opts.Revision is non-zero, the key-value pairs are retrieved at the specified revision.
	// If the required revision has been compacted, the request will fail with ErrCompacted.
	// If opts.Limit is greater than zero, the number of returned key-value pairs is bounded by the limit.
	// If opts.Continue is not empty, the listing will start from the key immediately after the one specified by Continue.
	List(ctx context.Context, prefix string, opts ListOptions) (ListResponse, error)

	// Count returns the number of keys with the specified prefix.
	Count(ctx context.Context, prefix string) (int64, error)

	// OptimisticPut creates or updates a key-value pair if the key has not been modified or created
	// since the revision specified in expectedRevision. Otherwise, it updates the key-value pair
	// only if it hasn't been modified since expectedRevision.
	//
	// If opts.GetOnFailure is true, the modified key-value pair will be returned if the put operation fails due to a revision mismatch.
	// If opts.LeaseID is provided, it overrides the lease associated with the key. If not provided, the existing lease is cleared.
	OptimisticPut(ctx context.Context, key string, value []byte, expectedRevision int64, opts PutOptions) (PutResponse, error)

	// OptimisticDelete deletes the key-value pair if it hasn't been modified since the revision
	// specified in expectedRevision.
	//
	// If opts.GetOnFailure is true, the modified key-value pair will be returned if the delete operation fails due to a revision mismatch.
	OptimisticDelete(ctx context.Context, key string, expectedRevision int64, opts DeleteOptions) (DeleteResponse, error)
}

type GetOptions struct {
	Revision int64
}

type ListOptions struct {
	Revision int64
	Limit    int64
	Continue string
}

type PutOptions struct {
	GetOnFailure bool
	// LeaseID
	// Deprecated: Should be replaced with TTL when Interface starts using one lease per object.
	LeaseID clientv3.LeaseID
}

type DeleteOptions struct {
	GetOnFailure bool
}

type GetResponse struct {
	KV       *mvccpb.KeyValue
	Revision int64
}

type ListResponse struct {
	KVs      []*mvccpb.KeyValue
	Count    int64
	Revision int64
}

type PutResponse struct {
	KV        *mvccpb.KeyValue
	Succeeded bool
	Revision  int64
}

type DeleteResponse struct {
	KV        *mvccpb.KeyValue
	Succeeded bool
	Revision  int64
}


```

### Design considerations

**How should arguments be passed?** Proposed: Options struct. 

*   It’s more extensible than a hardcoded list of arguments, allowing adding more fields in future.
*   It’s more readable than the variadic options list when arguments are optional. 
    Take a server code to manage [list limit options] as an example.
*   Same arguments apply for response struct.
 
[list limit options]: https://github.com/kubernetes/kubernetes/blob/97e87e2c40e5b83399a44738d38653fd59c58e99/staging/src/k8s.io/apiserver/pkg/storage/etcd3/store.go#L640-L645

**Prefer Range vs List semantics?** Proposed: List

*   List matches the intention of the Kubernetes behavior

**Combine Create and Update?** Proposed: Combine them into Put

*   They are the same from an argument standpoint. Create is a Update with ExpectedRevision set to 0.
*   The difference in on failure can be solved by optional argument `GetOnFailure`


### Watch interface

For the reasoning please see the section below.
```

type Kubernetes interface {
	Watch(ctx context.Context, key string, opts WatchOptions) KubernetesWatchChan
	RequestProgress(ctx context.Context, opts RequestProgressOptions) error
}

type WatchOptions struct {
	StreamKey string
	Revision  int64
	Prefix    bool
}

type RequestProgressOptions struct {
	StreamKey string
}

type KubernetesWatchChan <-chan KubernetesWatchEvent

type KubernetesEventType string

const (
	Added    KubernetesEventType = "ADDED"
	Modified KubernetesEventType = "MODIFIED"
	Deleted  KubernetesEventType = "DELETED"
	Bookmark KubernetesEventType = "BOOKMARK"
	Error    KubernetesEventType = "ERROR"
)

type KubernetesWatchEvent struct {
	Type KubernetesEventType

	Error error
	Revision int64
	Key string
	Value []byte
	PreviousValue []byte
}
```

### Design considerations

**What control does the user have over requesting progress?** Proposed: Allow user to set streamKey when create watch and requesting progress

*   StreamKey is used to separate watch grpc streams. 
    For Kubernetes we always use one stream as we don’t change grpc metadata between requests (e.g. WithRequireLeader). 
    Currently etcd client doesn’t expose streamKey to the user, just calculates it based on grpc metadata taken from context.
*   Having access to streamKey is useful as progress notifications cannot be requested on a per watch basis, only for the whole stream.
    This isn’t a big problem in their current setup as Kubernetes opens only one watch per resource. However, this would become a scalability issue for CRDs.

**Should Kubernetes explicitly pass WithRequireLeader or make it default?** Proposed: Make it default if Kubernetes interface is used.

**Should we wrap the watch response?** Proposed: Yes, it allows us to codify the Kubernetes dependency on single revision per transaction and PrevKV dependency.

## Alternatives

### Code location

#### Part of the etcd Client Struct

**Pros:**

*   **Seamless Integration:** The interface becomes inherently part of the client, fostering intuitive usage.
*   **Code Reuse:** Leverage existing private client methods, reducing redundancy.

**Cons:**

*   **Tight Coupling:** Changes to the interface necessitate updates to the entire etcd client, impacting Kubernetes upgrades.
*   **Limited Autonomy:** Release and bug-fix cycles are bound to the etcd project's schedule, which may not align with Kubernetes' needs.
*   **Backporting Challenge:** Requires backporting to v3.5 for Kubernetes compatibility, going against the etcd project's goal of minimizing backports.

#### New Package in etcd Repository

**Pros:**

*   **Versioning Flexibility:** Allows for independent versioning (e.g., `v3.5.13-interface.1`) to track interface changes separately from the etcd client.
*   **Manageable Integration:** Separates the interface from the client but keeps it within the etcd project, simplifying coordination.

**Cons:**

*   **Backporting Challenge: **Still requires backporting to v3.5 for initial Kubernetes compatibility.
*   **Maintenance Overhead:** Separate versioning introduces some additional maintenance effort to ensure compatibility between the interface and etcd versions.
*   **Compatibility Risk:** Incompatibilities may arise between etcd and interface versions if not managed meticulously.


#### New Repository under etcd-io

**Pros:**

*   **Maximum Autonomy:** Grants Kubernetes full control over development, releases, and bug fixes.

**Cons:**

*   **Increased Overhead:** Demands significant effort for maintenance, versioning, and compatibility across etcd client versions.
*   **Dependency Management:** Introduces an additional dependency for Kubernetes, increasing the complexity of version management.
*   **Potential for Code Duplication:** Implementing the interface might necessitate changes to internal client behavior, potentially requiring some code to be copied.
