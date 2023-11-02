<!-- toc -->
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [Terminology](#terminology)
- [Proposal](#proposal)
- [Storage version](#storage-version)
  - [Consequences](#consequences)
  - [Upgrading storage](#upgrading-storage)
  - [Downgrading storage](#downgrading-storage)
  - [Etcdutl migrate](#etcdutl-migrate)
- [Alternatives](#alternatives)
  - [Keep DB fields from newer version](#keep-db-fields-from-newer-version)
  - [Ignore unknown wal entries](#ignore-unknown-wal-entries)
- [Implementation plan](#implementation-plan)
  - [Milestone 1](#milestone-1)
  - [Milestone 2](#milestone-2)
  - [Milestone 3](#milestone-3)
- [Detailed design](#detailed-design)
  - [Version annotations](#version-annotations)
  - [Schema](#schema)
  - [Backward compatibility mode](#backward-compatibility-mode)
<!-- /toc -->


This document is a followup to [etcd Downgrades Design](https://docs.google.com/document/d/1mSihXRJz8ROhXf4r5WrBGc8aka-b8HKjo2VDllOc6ac/edit?usp=sharing) focusing solely on the issues related to DB file and WAL log. 
It is meant to address the same issues as in original design and propose alternative solution to etcd data backward compatibility.

## Motivation
As a fairly successful project etcd is frequently used in production and has become a critical part of infrastructure for many companies.
A large part of its success can be attributed to it’s great reliability thanks to Raft consensus protocol,
however the same level of reliability cannot be attributed to upgrade and downgrade processes.
Upgrades cannot be interrupted or reverted, they require external tooling and downgrades are not really supported at all.

Etcd upgrade process still depends heavily on the “backup before upgrade” strategy as there is no way to safely downgrade etcd data after it was touched by newer version.
This doesn’t work in practice, as some applications like Kubernetes will not work correctly on data from backup.
As etcd data is not versioned, projects like Kubernetes needed to develop their own scripts to start and stop all historical etcd versions just to make upgrade automation predictable.
Etcd data versioning is also important for future implementation of downgrades,
[Etcd downgrade proposal](https://docs.google.com/document/d/1mSihXRJz8ROhXf4r5WrBGc8aka-b8HKjo2VDllOc6ac/edit?usp=sharing) lists solving the issue of etcd data backward compatibility as one of the prerequisites.
Introducing strict policies on etcd data versioning will improve reliability of etcd upgrades and unblock the ability of the etcd cluster to downgrade.

### Goals
* Allow etcd to load data from failed upgrade
* Simplify etcd data upgrades process
* Allow older etcd to safely load data from downgrading member

### Non-Goals
* Support of etcd version < 3.6

### Terminology
* Etcd data - Data written by Etcd to disk. Contents of etcd data dir, includes DB file and Wal file
* DB file - File representation of KV state. For v3.+ it's boltdb file.
* DB Schema - Versioned representation of information stored within etcd DB file. Put simpler: names of all fields used in DB file and their meaning.
* Field - Describes how particular information should be stored in boltdb. Combination of boltdb bucket, key used to store, semantic meaning of value and marshalling method used.
* Subfield - Some fields have a composite of multiple values needed to convey some information that are stored under the same boltdb key (for example DowngradeInfo).


## Proposal

Introduce etcd storage version (SV) that is stored and persisted within the etcd DB file.
The SV should be used to indicate which version particular etcd data is compatible with and allow to conduct safe upgrades and downgrades of the etcd data contents.
To simplify etcd data upgrade and downgrade process we will also introduce etcdutl migrate command that will remove the need to start and stop all etcd server versions on 2+ minor upgrades jump

## Storage version

The storage version is represented as a “Major.Minor” and should match etcd version that created the storage.

For storage to be at particular version it means that:
* DB file has all fields that are used by matching etcd version
* DB file doesn’t include any fields or subfields added in newer etcd versions
* WAL files (from last snapshot and newer) don’t include any newer entries or fields.

DB fields to be backward compatible they should not have any information that would be incorrectly interpreted.
Even if some fields would be untouched by older etcd versions, it should be up to newer etcd to clean them up.
This way we ensure that it's the developers responsibility to provide cleanup logic at the moment of adding a field, instead of depending on backports.
Leaving those fields is pretty risky, as they might be unintentionally deleted by older version (subfields in structs like DowngradeInfo), become outdated (term) or have hard to predict by consequences (leaving isLearner field could take cluster down if user changes configuration after downgrade).
It might be tempting to leave those fields in case a user wants to use them after they upgrade back in the future, but it is much safer to inform users that some feature they are using will stop working after. 

Defining a storage version based on fields in a DB file requires creation of a schema.
A single place to define all fields and be a source of truth about which version a particular was introduced in. 
Putting a schema in source code would allow etcd and etcdutl to easily validate if fields match expectation and be able to manipulate them to conduct upgrades and downgrades.

Raft consensus protocol correctness depends on the ability of each participant to arrive at the same end state, if the input was the same.
This is easily fulfilled in cluster all members with the same version, but when an etcd with older version joins the cluster, other members need to be able to downgrade their communication protocol to match the oldest member. 
This is a feature introduced in etcd v3.4. 
As etcd wal files are a dump of (maybe not yet committed) etcd communication they need to follow the same rules. To make sure that etcd data can be correctly interpreted we need to make sure that it doesn’t include any entries or fields that would not be understood.

### Consequences

With the storage version (SV) defined as above, we can give a guarantee if etcd can always read etcd data if it has the same SV.
Etcd is also able to load data with SV lower than itself, if it has logic necessary.
However, etcd will no longer be allowed to load data generated by version higher than its own, and must rely on cluster downgrade process instead.

### Upgrading storage

Bumping storage version is always possible if it’s executed by a tool that understands the schema changes that need to be made.
This makes the upgrade process pretty simple so it can be executed by etcd during bootstrap (for 1-minor upgrade).

Storage version upgrade usually happens in the cluster upgrade process, when a newer version of etcd binary bootstraps on old data. 
When etcd data is loaded, it should verify that storage has the correct version (same or lower) and upgrade it to its own version by introducing all the missing fields, snaphotting the wal log and setting a new storage version. 
With that it should be ready to continue normal operation.

As new etcd versions are released, maintaining all historical schemas in etcd would lead to increase of maintenance overhead.
Over time etcd can also introduce alternative file backend formats and deprecate some, leading to needless increase of dependencies. As etcd upgrades should happen only one version forward, etcd should only support upgrading storage only one minor version down or major version if there is no lower minor version.
Version change by more than version should be done by a separate tool.

### Downgrading storage
Lowering the storage version is much more restricted when compared to increasing it.
Fields from DB files can be simply removed, but we cannot do the same for Wal entries.
Skipping uncommitted wal entries would break Raft assumptions and lead to corruption of etcd data.
Before starting to downgrade the storage, we need to validate that it doesn’t include any wal entries or fields that belong to higher version.
To achieve that we should be able to downgrade etcd communication protocol the same way as [clusters with mixed versions](https://etcd.io/docs/v3.3/upgrades/upgrade_3_4/#mixed-versions) and snapshot the wal.

During downgrade, cluster members need to lower etcd protocol version to allow older versions to join. All new features, rpcs and fields should not be used thus preventing older members to interpret replicated log differently.
This also means that entries added to the Wal log file should be compatible with lower versions. 
When a wal snapshot happens, all older incompatible entries should be applied, so they no longer need to be read and the storage version can be downgraded.

### Etcdutl migrate
With this proposal etcd will begin validating etcd data during bootstrap and will refuse to run on data created by newer or outdated (version difference >= 2) etcd.
To prevent users from running temporary chains of different versions to storage to version they want we need to introduce a dedicated offline tooling.
I propose to reintroduce the etcdutl migrate command that can upgrade or downgrade etcd data dir as needed.

Etcdutl migrate should accept path to the data dir and targeted etcd version. 
First it should validate if version change is based on version difference and contents of the wal log. 
In case of downgrades it should also check if it will cause any features to be disabled (for example learner), warn users about this and ask for confirmation (should be skippable with -y flag).
When ready it should execute any necessary schema changes to the DB file and update the storage version. 
We should also leave an escape hatch (--force flag) for situations where users want to bypass the safety checks.

## Alternatives

Let’s discuss an alternative approach to etcd data versioning proposed in original [etcd Downgrades Design](https://docs.google.com/document/d/1mSihXRJz8ROhXf4r5WrBGc8aka-b8HKjo2VDllOc6ac/edit?usp=sharing).

### Keep DB fields from newer version
Defining a storage version requires introducing a way to track schema of fields, but it doesn’t necessarily need to be as strict as in this proposal.
Schema can be softened by allowing storage to include fields from newer etcd versions.
The main use of leaving those fields during downgrade is potential that they could be used in future as the user decides to upgrade back.
This approach requires much less work as we technically do not need the track which version introduced which field.
However I would argue that downsides overweight skipping additional work.
Problems with this approach:

* There is no real benefit of storing fields from newer versions. 
  Usually downgrades should happen in response to issues encountered after upgrade (for example performance regression). 
  Users would not have enough time to really use or benefit from new features, that would motivate preserving them throughout downgrade.
* It makes consequences of upgrades much harder to predict. 
  Some fields like isLearner can totally change cluster behavior. 
  Let’s imagine a case where a user added some learner members to a cluster. 
  After downgrading those learners will become normal members, possibly going into an even number of members (not recommended). 
  In such situations users might prefer to scale down the cluster, but if they make a mistake or forget which members were designated as a learner they might remove a normal member instead. 
  This could have catastrophic results when the user decides to upgrade back and finds out that their cluster cannot reach quorum and goes down. It’s also worrying that this problem cannot be easily noticed as it’s impossible to check learners on the older version without inspecting the data dir or snapshots manually.

proposing a stick schema we want to force etcd developers to think about how their changes to fields should work between versions and introduce dedicated logic that needs to be run during storage upgrades and downgrades.

### Ignore unknown wal entries

Storage downgrade mechanism makes it impossible to downgrade storage in case of wal files including raft entries from newer etcd. 
This is a problematic requirement as it basically pushes the problem of compatibility to be solved during cluster downgrade, making it impossible to downgrade etcd data without running a cluster. 
An alternative approach to this is to have an older version to just ignore unknown entries. 
Using this approach we can remove previously mentioned limitations and allow older etcd to always load data from newer versions and save work by removing the need to annotate all raft messages with version. 
However I think this approach has some big flaws that make it unreliable. 

Problems with this approach:

* Raft consensus protocol correctness depends on the ability of each participant to arrive at the same end state, if the input was the same. 
  Skipping any entries by older version directly breaks this assumption as during the downgrade process there will be nodes with multiple versions. 
  By ignoring unknown entries, older versions will end up in different states. 
  It is too optimistic to assume that this will not have any long term negative consequences.
* Any changes to existing wal log entries, this approach requires to spread compatibility code between two versions.
  In the first version we need to add code to handle new changes, but don’t act on it and only the second version can fully implement new functionality.
  This basically means delaying feature development by one release or backporting compatibility to already released versions.
  Both options should be avoided if possible.
* Original downgrade design proposes to push the responsibility for compatibility to developers, but doesn’t propose any mechanism to properly enforce this or validate correctness.
  Adding integration tests will not scale to accommodate complexity of etcd storage and matrix of downgrade/upgrade paths.

By proposing to version all wal entries we want to make verifying backward compatibility much easier and testable.
Ensuring raft correctness by design seems like a much better approach versus depending on humans not making mistakes when testing.



## Implementation plan

### Milestone 1

Storage version is available for snapshots. Context: #13070
* Etcd saves storage version during first raft snapshot
* During bootstrap etcd and etcdutl should panic when encountering higher storage version
* Add etcdutl migrate --force as an escape hatch

### Milestone 2

Etcd code is annotated with versions
* Implement version gating for all features
* Implement version annotations for golang structs and proto
* Create a DB file schema
* Add static analysis to validate annotations

### Milestone 3
Downgrades can be implemented based on storage versioning
* Etcd should validate storage version during bootstrap
* Implement etcdutl migrate


## Detailed design

### Version annotations

Downgrading a storage requires validating contents of DB file and wal files for existence of any fields, subfields or wal entries that come from newer versions.
To properly implement this feature we need a way to be always able to check what version a particular field comes from. 
I propose to introduce specific annotations that can be added to both golang struct fields as [tags](https://golang.org/ref/spec#Struct_types) and proto message fields as [custom options](https://developers.google.com/protocol-buffers/docs/proto#customoptions).
By putting annotations directly on fields eliminate the chance that developers will forget to update some dedicated function for this.
Information stored in code/proto annotations can be extracted during runtime using code introspection (reflect). 
To prevent skipping any new fields from being annotated I propose to add a simple static analysis that would analyse code and validate it.

Example of using golang tags:

```go
type DowngradeInfo struct {
// TargetVersion is the target downgrade version, if the cluster is not under downgrading,
// the targetVersion will be an empty string
TargetVersion string `json:"target-version" etcd_version:"3.5"`
// Enabled indicates whether the cluster is enabled to downgrade
Enabled bool `json:"enabled" etcd_version:"3.5"`
}
```

Example of using proto annotation:
```protobuf
import "google/protobuf/descriptor.proto";

// Indicates etcd version that introduced the message, used to determine the minimal etcd version needed to interpret the WAL that includes this message.
extend google.protobuf.MessageOptions {
  optional string etcd_version_msg = 50000;
}

// Indicates etcd version that introduced the field, used to determine the minimal etcd version needed to interpret the WAL that sets this field.
extend google.protobuf.FieldOptions {
  optional string etcd_version_field = 50001;
}

// Indicates etcd version that introduced the enum, used to determine the minimal etcd version needed to interpret the WAL that uses this enum.
extend google.protobuf.EnumOptions {
  optional string etcd_version_enum = 50002;
}

// Indicates etcd version that introduced the enum value, used to determine the minimal etcd version required to interpret the WAL that sets this enum value.
extend google.protobuf.EnumValueOptions {
  optional string etcd_version_enum_value = 50003;
}

message RequestHeader {
  option (etcd_version_msg) = "3.0";

  uint64 ID = 1;
  // username is a username that is associated with an auth token of gRPC connection
  string username = 2;
  // auth_revision is a revision number of auth.authStore. It is not related to mvcc
  uint64 auth_revision = 3  [(etcd_version_field) = "3.1"];
}

enum AlarmType {
  option (etcd_version_enum) = "3.3";
  NONE = 0; // default, used to query if any alarm is active
  NOSPACE = 1; // space quota is exhausted
  CORRUPT = 2 [(etcd_version_enum_value)="3.3"]; // kv store corruption detected
}
```

Decided on having multiple types of custom annotations message (msg, field, enum, enum_value) vs using one field annotation, based on:
* Enum default value (equal 0) will never be considered set, so annotating it doesn't make sense.
  It can be also confusing as someone might think that setting this enum value will mean that the required etcd version is changed, but it's not.
* Messages with no fields, for example AuthStatusRequest. 
  We technically can determine the version based on field annotation that uses this message, but I think it might become confusing when we will add some fields to this message. Messages should be self sufficient to define their etcd version.
* Readability, having this annotation on each field makes the proto much less readable. 
  I would prefer to keep the proto definition cleaner and even if it complicates other code.

### Schema

In v3.5 code field definitions are spread around the codebase, each defined in their own way, each with their own read and update function. 
This makes answering simple questions like, what data etcd saves to DB file hard and ensuring that all fields are properly versioned impossible. 
To introduce a schema we need to start gathering and generalizing storing metadata in DB file.
Simplest way to create a schema would be to move all backend read and update functions into one dedicated package and expose an interface for interacting with them. 
Long term we should make access to the Backend interface harder to encourage using schema interface and generalize update/read functions to reduce the number of unique ways fields are serialized.

```go
type ClusterVersionBackend interface {
    // UnsafeSaveClusterVersion saves the cluster version to the bolt backend.
    UnsafeSaveClusterVersion(*semver.Version)
    // UnsafeReadClusterVersion reads the cluster version from the bolt backend.
    UnsafeReadClusterVersion() *semver.Version
}
```

### Backward compatibility mode

Etcd v3.4 introduced support for clusters with mixed versions.
As described in documentation [Mixed versions](https://etcd.io/docs/v3.3/upgrades/upgrade_3_4/#mixed-versions) clusters supports running with mixed versions of etcd members, and operates with protocol of the lowest common version. 
For future downgrade implementation to utilize the proposed here storage version we need to invest more into ensuring that this functionality is properly tested and easy to maintain.

Improving this will require implementing a full backward compatibility mode with 1 version older. 
When enabled, etcd should behave like it’s older counterpart covering all aspects like API, Raft, data storing etc. 
For example when enabled etcd should reject all requests that were added in newest versions and revert to older internal logic. 
Implementing this will require larger codebase refactor, but it is the only way to ensure full compliance with Raft correctness assumptions. 
On the bright side maintaining two api implementations seems better than designing all the features to be split between two releases. 
Having code of two versions should allow us for much detailed compatibility testing, compared to testing just using previous version binary.

Short term we should be able to just add dedicated if statements checking for current cluster version, but long term we should target having two separate implementations of externally facing structs. 
This approach should allow for low maintenance code without introducing much code duplication as most etcd logic doesn’t change version to version.

Example implementation for etcdServer:
```go
type EtcdServer interface {
    EtcdServerCommon
    EtcdServerVersionSpecific
}

type EtcdServerCommon interface {
}

type EtcdServerVersionSpecific interface {
    SetVersion(*version.Version)
}

type etcdServerCommon struct {
    versionSpecific EtcdServerVersionSpecific
}

func (s *etcdServerCommon) SetVersion(v *version.Version) error {
    switch v {
    case V3_6: 
		s.versionSpecific = *etcdServerV3_6{...}
    case V3_5: 
		s.versionSpecific = *etcdServerV3_5{...}
    default: 
		return fmt.Errorf("Not supported version %q", v)
    }
    return nil
}

type etcdServerV3_6 struct {...}

type etcdServerV3_5 struct {...}
```


