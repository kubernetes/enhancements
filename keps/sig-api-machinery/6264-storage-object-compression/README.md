# KEP-6264: Storage Object Compression

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Measured on in-tree fixtures](#measured-on-in-tree-fixtures)
  - [Measured on production data](#measured-on-production-data)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: an encrypted cluster with no other lever](#story-1-an-encrypted-cluster-with-no-other-lever)
    - [Story 2: an unencrypted cluster](#story-2-an-unencrypted-cluster)
    - [Story 3: an operator who wants compression but not on Secrets](#story-3-an-operator-who-wants-compression-but-not-on-secrets)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [The at-rest byte format](#the-at-rest-byte-format)
  - [The ordering of compression and encryption](#the-ordering-of-compression-and-encryption)
  - [The write path](#the-write-path)
  - [Staleness and migration](#staleness-and-migration)
  - [The read path](#the-read-path)
  - [Bounding decompression](#bounding-decompression)
  - [What authenticates a frame](#what-authenticates-a-frame)
  - [API Priority and Fairness, and size accounting](#api-priority-and-fairness-and-size-accounting)
  - [Error classification and version skew](#error-classification-and-version-skew)
  - [Observability](#observability)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [Upgrade and downgrade tests](#upgrade-and-downgrade-tests)
      - [e2e tests](#e2e-tests)
      - [Scale test](#scale-test)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
  - [Open Questions](#open-questions)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Every write to etcd passes the serialized object through a `value.Transformer` chain that encrypts
it. Ciphertext is incompressible, so in an encrypted cluster no layer below kube-apiserver can
reduce the storage footprint of an API object: not etcd, not bbolt, not the gRPC transport.

This KEP adds an opt-in compression stage that runs before the encryption chain on write and after it
on read, implemented as a single `value.Transformer` decorator. Because a decorator compresses and then
delegates, the value that reaches the encryption chain is already compressed, and the `k8s:enc:`
envelope stays byte-for-byte outermost on disk.

Backward compatibility is a property of the bytes rather than of configuration. A compression frame
begins with `0x00`, which no kube-apiserver storage serializer can emit: protobuf begins
`6b 38 73 00`, JSON `{`, CBOR `d9 d9 f7`. One byte comparison therefore decides whether a value is
compressed, with no flag to consult. Consequently **decompression is unconditional and cannot be
disabled**; only the write path is configurable. Disabling the feature, or removing a resource from the
policy, can never strand data.

Compression is opt-in per resource and off by default at every stage, including GA.

## Motivation

Large clusters accumulate etcd bytes that no existing mechanism can reclaim, and for clusters using
encryption at rest there is no lever below kube-apiserver at all. Compression has to happen above the
transformer chain or it does not happen.

Compressing here reduces:

1. etcd database size, through fewer bytes per stored revision.
2. Backup, snapshot and raft replication volume.

It does nothing for the watch cache, which holds decoded objects rather than stored bytes, and
nothing for small high-entropy objects, which is why the write path has a configurable size
threshold.

### Measured on in-tree fixtures

The round-trip fixtures in `staging/src/k8s.io/api/testdata/HEAD`, raw DEFLATE level 1.

Each fixture is decoded and re-encoded through the real storage serializer before compressing, because
the file on disk is not what gets stored: the `.json` fixtures are pretty-printed for human review
(47.5% of the Pod fixture is whitespace) while the storage serializer writes compact JSON. Measuring
the file as it sits overstates the JSON ratio by roughly a quarter, 7.56x pretty against 6.36x compact.

| Fixture | protobuf | compressed PB | ratio | JSON (compact) | compressed JSON | ratio |
|---|---|---|---|---|---|---|
| `core.v1.Pod` | 13 046 | 2 677 | **4.87x** | 30 928 | 4 860 | 6.36x |
| `apps.v1.StatefulSet` | 12 583 | 2 605 | 4.83x | 29 416 | 4 666 | 6.30x |
| `apps.v1.Deployment` | 11 497 | 2 436 | 4.72x | 27 155 | 4 389 | 6.19x |
| `batch.v1.Job` | 11 701 | 2 505 | 4.67x | 27 683 | 4 595 | 6.02x |
| `core.v1.Node` | 1 329 | 649 | 2.05x | 2 987 | 992 | 3.01x |
| `core.v1.Endpoints` | 728 | 355 | 2.05x | 1 530 | 513 | 2.98x |
| `events.k8s.io.v1.Event` | 778 | 421 | 1.85x | 1 725 | 597 | 2.89x |
| `core.v1.Service` | 945 | 569 | 1.66x | 2 100 | 822 | 2.55x |
| `rbac.authorization.k8s.io.v1.Role` | 492 | 344 | 1.43x | 1 040 | 457 | 2.28x |
| `core.v1.Secret` | 441 | 309 | 1.43x | 948 | 424 | 2.24x |
| `core.v1.ConfigMap` | 427 | 309 | 1.38x | 926 | 415 | 2.23x |
| `coordination.k8s.io.v1.Lease` | 485 | 358 | 1.35x | 1 104 | 491 | 2.25x |

Most of these objects would never be compressed: seven of the twelve have a protobuf form under 1 KiB,
below the default threshold. Their 1.35-2.05x ratios are moot by design, since those are the objects
where compression costs CPU on every read and write for no durable benefit.

These fixtures may not be representative: every field is populated, so a fixture Secret is smaller than
a real one holding data while a fixture Pod is denser than a real one. Real production Pods in the
~14.6 KB size class deliver 2.75x where the ~13 KB fixture Pod claims 4.87x, an overstatement of
roughly 1.8x, which is why the fixtures are a control rather than evidence.

### Measured on production data

672 Pods sampled from `RequestResponse` audit events on a large production cluster, decoded into
typed objects, re-encoded to protobuf with `resourceVersion` cleared to model the storage layer, and
compressed at DEFLATE level 1, the shipped configuration.

| Size class (protobuf p50) | Pods | stored p50 | ratio p50 | ratio p10 | ratio p90 |
|---|---|---|---|---|---|
| 3.5 KB | 72 | 1 986 | **1.75x** | 1.75x | 1.75x |
| 6.7 KB | 100 | 2 745 | **2.43x** | 2.41x | 2.45x |
| 14.6 KB | 100 | 5 308 | **2.75x** | 2.71x | 2.75x |
| 34.2 KB | 100 | 13 406 | **2.56x** | 2.55x | 2.71x |
| 76.6 KB | 100 | 18 669 | **4.10x** | 3.79x | 4.25x |
| 162.5 KB | 100 | 51 650 | **3.36x** | 2.35x | 4.98x |
| 278.5 KB | 100 | 45 211 | **6.15x** | 6.01x | 6.18x |

These size classes must be read independently and must not be pooled. Each was sampled as up to the
100 largest Pods below a size threshold, and the smallest class yielded only 72, so the sample says
nothing about how many Pods of each size a cluster holds. The percentiles within a row are
meaningful; any statistic across rows is not.

Compression is worth doing at every size class measured: the weakest, at 3.5 KB, still removes 43% of
the bytes, and the lowest ratio observed in any individual Pod was 1.51x. The ratio does not increase
monotonically with size, so it cannot be extrapolated from an object's size alone.

The sample covers one resource on one cluster, and audit events record mutations rather than the live
object set, so these numbers speak to what a write costs rather than directly to database size. The
scale test in the [Test Plan](#test-plan) measures database size directly and closes both gaps.

### Goals

- Compress the serialized form of an API object before it is encrypted and written to etcd.
- Read pre-existing uncompressed values, encrypted or not, with no migration and no configuration.
- Make disabling the feature safe without rewriting etcd.
- Make the at-rest format self-describing, so a heterogeneous fleet mid-rollout needs no
  coordination between apiservers.
- Work whether or not encryption at rest is configured, through one code path and one at-rest format.
  Encrypted clusters are the motivating case, since compression is the only mechanism that can reduce
  their stored size, but unencrypted clusters get the same benefit from the same code.
- Per-resource opt-in from the first release, so an operator can compress ConfigMaps and custom
  resources without touching Secrets.
- Add no new third-party dependency.

### Non-Goals

- Using compression to exceed the object size limit. This is enforced rather than merely declared:
  plaintext above the per-value ceiling is never framed, so such an object fails on write exactly as
  it does today.
- Compressing data in flight to clients. That is HTTP content negotiation, and already exists
  (KEP-2338).
- Hiding plaintext length. Padding into size buckets would close the compressibility side channel
  but negate the feature; see Risks and Mitigations.
- Cross-object or dictionary-based compression. This is a permanent, security-relevant non-goal.
- Compressing by default. The per-resource policy is empty at every stage including GA.
- Reducing watch-cache memory.

## Proposal

Compression is configured by a file, pointed to by a single flag. Absent the flag the feature is
inert, which is how "off by default" is expressed: a cluster that upgrades and takes no action stores
byte-identical values to before.

```yaml
apiVersion: apiserver.config.k8s.io/v1alpha1
kind: StorageCompressionConfiguration
resources:
  # Ordered: the first entry matching a resource wins, exactly as in
  # EncryptionConfiguration. This is how a resource is excluded from a wildcard.
  - resources: ["events", "events.events.k8s.io"]
    algorithm: None
  - resources: ["customwidgets.example.com"]
    algorithm: Deflate
    # Optional per-entry write threshold. Defaults to 1Ki when omitted.
    minSize: 512
  - resources: ["*.*"]
    algorithm: Deflate
```

```
--storage-compression-config=/etc/kubernetes/storage-compression.yaml
```

`algorithm` is `Deflate` or `None`. Resource entries use the same forms as
`--encryption-provider-config`: `<resource>`, `<resource>.<group>`, `*.<group>`, or `*.*`.

A file rather than flags, for three reasons. The ordering rule makes exclusion expressible at all: a
flat list of resources cannot say "everything except Events", and Events are an explicit anti-target
here, being high-churn, small, short-TTL and often on a separate etcd, so compressing them spends CPU
for almost no durable saving. The shape is also borrowed rather than invented, so an operator who has
written an `EncryptionConfiguration` already knows how to read it. Decisively for an alpha feature, a
flag is effectively permanent once inherited by every generic-apiserver consumer through
`EtcdOptions`, whereas a `v1alpha1` type carries no compatibility promise and can be reshaped at beta.

There is no feature gate. The flag is the switch: unset, nothing is compressed, and a gate would only
add a second lock on a knob that already requires editing kube-apiserver's arguments and restarting.
The `v1alpha1` configuration API carries the alpha stability instead, and the flag's help text carries
the downgrade warning a gate would otherwise have implied.

`secrets` is never selected by a wildcard. Compressing Secrets requires naming the resource
explicitly, which is a deliberate enough act to serve as the acknowledgement; doing so logs a warning
at startup naming the side channel described under Risks and Mitigations.

The configuration is read once at startup. There is no hot reload at alpha, deliberately, since a
per-resource storage format should not change from a file edit and a poll interval, and because
`--encryption-provider-config` sets the precedent that reload arrives later as a separate opt-in flag
if it is wanted at all.

The minimum size is configurable per entry as `minSize`, defaulting to 1 KiB, because object size
distributions differ between resources and between clusters for the same resource. Sitting on an entry
rather than at the top level, it inherits the same ordered first-match-wins rule as the algorithm, so a
wildcard can carry one threshold while a named resource ahead of it carries another. Legal values run
from 64 bytes to just below the per-value ceiling; outside that range startup fails rather than
silently framing nothing or everything.

The threshold gates writing only, and so does the policy. A value carries its own format, so any
apiserver of this version or newer reads any value written by any other, whatever its configuration,
and changing the threshold can never make existing data unreadable.

The compression level and the inflation concurrency cap remain compile-time constants at alpha. The
level is a CPU-for-ratio trade whose measured benefit below ~34 KB is 3-7%, so there is no operator
need yet; both are candidates for the configuration file at beta if benchmarking justifies it.


### User Stories

#### Story 1: an encrypted cluster with no other lever

A platform operator runs envelope encryption via KMS, and most of their etcd content is custom
resources stored as JSON. Every etcd-side option is useless because etcd only ever sees ciphertext.
They name their CRDs in the compression config, restart the control plane, run a storage version
migration, and see the compacted database shrink.

#### Story 2: an unencrypted cluster

An operator with no encryption configured has a large number of ConfigMaps. They get the same
reduction from the same configuration and the same code path. Plaintext length was already fully
visible in etcd, so the length channel is unchanged.

#### Story 3: an operator who wants compression but not on Secrets

An operator wants ConfigMaps and custom resources compressed but considers the compressibility of a
Secret's plaintext to be information their threat model cares about. A wildcard never reaches
`secrets`, so `*.*` gives them everything else without any carve-out on their part.


### Notes/Constraints/Caveats

The ordering constraint drives the whole design: compression must be strictly inside encryption,
which rules out every insertion point above the transformer chain, and rules out doing this in etcd at
all.

The intuitive design, a new on-disk prefix, does not work. `prefixTransformers` picks a transformer by
matching the prefix at the start of the stored value, so a compression prefix must sit outside the
encryption prefix to be matched first, which also puts it outside the ciphertext:

```
k8s:enc:deflate:v1:  k8s:enc:aescbc:v1:key1:  <ciphertext of the compressed plaintext>
└── compression ───┘ └──── encryption ──────┘ └──── encrypted ────────────────────────┘
```

The outer prefix is redundant. A reader must decrypt before it can decompress, and by then it holds the
plaintext, whose first byte already says whether the value is compressed. That test is needed
regardless, since during a migration one resource holds both forms, so the prefix duplicates a
mechanism that cannot be removed.

It also breaks things. `identityTransformer` rejects any value beginning `k8s:enc:`, and existing tests
assert that a stored value carries exactly one provider prefix. And it would state in cleartext which
resources are compressed: without it, an observer cannot tell a small object from a compressed larger
one, and that ambiguity limits what stored length reveals.

The `0x00` discriminator is an invariant about apimachinery that this KEP does not own. It holds
because protobuf storage begins `6b 38 73 00`, JSON `{`, CBOR `d9 d9 f7`, YAML a printable key
character, and the legacy base64 fallback uses only base64-alphabet bytes. A test walks every storage
serializer and asserts it, and the writer escapes rather than rejects a leading `0x00` so the format
stays total even if the invariant were ever violated. It nonetheless deserves a note in apimachinery
rather than living only here.

Small objects are deliberately excluded. Below the threshold, 1 KiB by default, the ratio does not
repay the CPU spent on every read and write, which excludes most of the in-tree fixtures in their
minimal form.

Enabling a resource causes one-time write amplification. `GuaranteedUpdate` normally skips the etcd
write when a client's update serialises to bytes identical to what was read, which is why controllers
re-applying an unchanged spec cost no revisions. Flagging a value stale deliberately defeats that
comparison, since that is how a value migrates to a new at-rest form, so idempotent controller updates
that were previously free each become one write and one revision until every object has been rewritten
once. It is bounded and self-terminating, but on a large cluster with chatty controllers it is a
visible burst, best driven with a storage version migration rather than left to organic churn.

Decompression is not disableable, by design, and the asymmetry is the central safety property: the
format is a one-way commitment. Once a release can write a frame, every subsequent release must be
able to read one, forever.

### Risks and Mitigations

Compressing before encrypting leaks plaintext compressibility: the stored length of a value becomes a
function of its content, not only of its size. That is a new information channel and needs sig-auth
review.

Ciphertext length is already a function of plaintext length for every provider, so exact plaintext
length is *already* exposed to anyone who can read etcd or a backup. Compression converts a leak of
length into a leak of compressibility, which is a coarse proxy for entropy and internal repetition.
CRIME- and BREACH-style attacks additionally require chosen plaintext compressed *in the same context*
as a secret, plus a repeatable length oracle. Cross-object attacks are structurally impossible here:
every value is compressed independently with a freshly reset compressor, with no shared dictionary and
no cross-value state. That invariant must be preserved deliberately, because the obvious future
optimisation, a shared dictionary, would destroy it.

The residual risk is intra-object, and needs three things at once: an adversary who can write part of an
object, cannot read that object, and can observe its stored length. The length condition confines it to
encrypted clusters, since in an unencrypted one such an adversary reads the plaintext directly and the
oracle buys nothing. The write-without-read condition is usually vacuous, because RBAC verbs are granted
independently but conventionally granted together; two shapes express it, `update` or `patch` on a
resource without `get` or `list`, and `patch` on a subresource such as `pods/status` without `get` on
the parent.

Four things limit it. No resource is compressed unless the configuration selects it. `secrets` is never
reachable by wildcard, so compressing it requires naming the resource and logs a warning that names the
channel. Observability is aggregate-only, with no per-object ratio or byte-count metric, since those
would be a finer oracle than the length channel itself. Length padding would close the channel entirely
and is rejected as negating the feature. The residual is accepted and stated, and a sig-auth review of
exactly this residual is an alpha graduation requirement.

Decompression bombs are reachable without any key. `aescbc` and the KMS v1 CBC read path are
unauthenticated and malleable, and `identity` does not authenticate at all, so an actor able to tamper
with etcd can steer bytes into the decompressor. The read path therefore bounds what it will allocate
and how many inflations may run at once; see [Bounding decompression](#bounding-decompression). A fuzz
target over the read path is a requirement rather than an option.

Rolling back below the first decompress-capable release breaks the cluster: an older apiserver
decrypts a frame successfully and then fails to decode it. One undecodable value aggregates into a
`StatusReasonStoreReadError` for a whole LIST prefix and tears down watches, which for the watch-cache
reflector removes cached serving for that resource. The effect is an outage rather than a degradation.
Mitigations: writing a frame is never reachable by default at any stage, requiring a configuration
file that names a resource, so no cluster acquires frames without a deliberate operator action;
and the only sound pre-downgrade procedure is a completed storage version migration per
compressed resource, because no read-derived metric can prove that no compressed value remains, since
a cold object is never read and so never observed. See [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).

Version skew must not be mistaken for corruption. A frame carrying an algorithm this binary does not
implement was written by a newer apiserver and is perfectly readable by one. Reporting it as a
`CorruptObjError` would be wrong twice over: the object is undamaged, and that error is what authorises
deletion through `DeleteOptions.ignoreStoreReadErrorWithClusterBreakingPotential`, the escape hatch
KEP-3926 added for genuinely unreadable objects. So an unimplemented algorithm is classified as skew
rather than corruption, and the read fails outright rather than offering the object for deletion.

CPU cost lands on both the write and read hot paths. Compression sits on every mutating request for a
selected resource, and the worst read case is watch-cache initialisation, a full LIST decompressed
sequentially on the request goroutine. KEP-2338 measured the analogous mistake: enabling gzip for all
suitable API responses "caused a significant performance regression in both CPU usage (2x) and tail
latency (2-5x) on the Kubernetes apiservers", which is why a 128 KB size floor was introduced
([KEP-2338 README][kep-2338]). Mitigations here are level 1, a configurable floor defaulting to 1 KiB,
pooled coder state, per-resource opt-in, and DEFLATE's asymmetry: decompression performs no match
search, only Huffman decoding and byte copies, so on real Pods it runs 1.7-2.0x faster than
compression. A watch-cache-initialisation benchmark is an alpha deliverable.

APF could under-charge LIST memory. The resource size estimator records stored bytes, and that
figure bounds the memory API Priority and Fairness believes a LIST will occupy, while the memory
actually consumed is the decompressed, decoded size. Left uncorrected, a resource compressing 5x would
let APF admit substantially more concurrent LISTs than the apiserver can hold, on the etcd-delegated
path where no separate ceiling already bounds it. The estimate is therefore scaled by an observed
expansion factor, floored at 1.0 so it can only ever be more conservative than the raw measurement.

Raw-etcd tooling sees an opaque value. `auger`, `etcdhelper` and snapshot-based forensics parse
etcd values directly and are the standard answer to "what is in this key?". In unencrypted clusters a
frame makes them fail, and none of them can be fixed by a kubernetes/kubernetes change. An `auger`
change is a beta graduation dependency, and the frame carries a literal `k8s` at bytes 1-3 so a human
reading a hex dump can at least recognise what they are looking at.

[kep-2338]: https://git.k8s.io/enhancements/keps/sig-api-machinery/2338-graduate-API-gzip-compression-support-to-GA/README.md

## Design Details

### The at-rest byte format

A framed value is a 5-byte fixed header (a 4-byte magic and one algorithm byte) followed by a body
whose shape depends on the algorithm:

```
offset  size  value / meaning
------  ----  ----------------------------------------------------------
0       1     0x00        discriminator; no storage serializer emits it
1       3     6b 38 73    "k8s"; structure check
4       1     algorithm   algStored (0x00) | algDeflate (0x01)
--- algStored ---
5       n     plaintext, verbatim                    overhead: 5 bytes
--- algDeflate ---
5       w     declared plaintext length, LEB128 (1-3 bytes)
5+w     m     raw DEFLATE stream (RFC 1951)          overhead: 6-8 bytes
```

The magic is `0x00` followed by the ASCII `k8s`, so a decrypted frame is recognisable in a hex dump.

Where this sits at rest depends on the encryption configuration, and the unencrypted case is the one
worth stating first because it is both the default and the most exposed:

- Without `--encryption-provider-config`, the transformer is `identity.NewEncryptCheckTransformer()`,
  which writes the serialized object through unchanged and with no prefix at all. So today the value
  in etcd simply is the serialized object, and with compression enabled it becomes the frame
  directly: the leading `0x00` is literally the first byte in etcd. This is also the only configuration
  with no integrity check anywhere beneath us; see [Integrity](#what-authenticates-a-frame).
- With encryption enabled, the frame is the plaintext that the encryption transformer consumes, so it is
  invisible at rest, wrapped inside a provider prefix and ciphertext. Those prefixes follow the grammar
  `k8s:enc:<provider>:<providerVersion>:<name>:`. `aescbc`, `aesgcm` and `secretbox` are at `v1` and
  append the key name, while `kms` exists at both `v1` and `v2` and appends the provider name.

A `deflate` frame declares its plaintext length so the read path can bound its allocation before
inflating anything; see [Bounding decompression](#bounding-decompression). A `stored` frame needs no
such field, since its body length *is* its plaintext length.

The frame carries no version byte, no flags byte, no reserved byte and no compression level. The
algorithm id serves as the version discriminator, because an unknown id already conveys exactly what a
reader needs to know, that a newer apiserver wrote this, and a future format change simply claims a
new id. The level is omitted because it is a writer-side choice the decoder never needs: any DEFLATE
stream decodes without knowing how it was produced, which is also what would let a per-resource level
be added later without a format change.

Framing costs 5 bytes for `stored` and 6-8 for `deflate`, at worst 0.78% of a 1 KiB object and 0.003%
of a 256 KiB one. Below roughly a kilobyte that overhead stops being negligible while the gain stops
being worth having: a Lease at 485 bytes compresses only 1.42x, too little to repay the CPU spent on
every read and write of it, and a genuinely incompressible object would simply grow. The floor keeps the
overhead off those objects, since they are never framed.

The floor is `minSize` on the matching configuration entry, 1 KiB by default, with a hard lower bound of
64 bytes: framing already costs 8-12% at that size, and DEFLATE on inputs that small essentially always
expands into the verbatim frame. The *ceiling* stays a compile-time constant, deliberately, since it is
tied to etcd's per-value limit and letting an operator raise it would manufacture objects that can be
read but never rewritten.

### The ordering of compression and encryption

A decorator's two methods run on opposite sides of the value it wraps: on write, code before the
delegate call sees plaintext and code after it sees ciphertext; on read the delegate runs first, so the
wrapper sees whatever the delegate produced.

```go
func (t *transformer) TransformToStorage(ctx context.Context, plaintext []byte, dc value.Context) ([]byte, error) {
    framed := t.frame(plaintext)                                 // sees plaintext
    return t.delegate.TransformToStorage(ctx, framed, dc)        // ... then encryption
}

func (t *transformer) TransformFromStorage(ctx context.Context, data []byte, dc value.Context) ([]byte, bool, error) {
    plaintext, stale, err := t.delegate.TransformFromStorage(ctx, data, dc)  // decryption first
    // ... then unframe
}
```

Compression must see plaintext to be worth anything, so on write it can only go before the delegate
call. That placement is compress-then-encrypt, and the read method is then necessarily
decrypt-then-decompress. There is no way to reach encrypt-then-compress, which would yield a ratio near
1.0 because ciphertext is indistinguishable from random.

### The write path

Two independent conditions cause a value to be framed. A value whose serialized form begins with `0x00`
is always framed, whatever the configuration says. Otherwise a value is framed when the policy selects
its resource and its length falls between the configured floor and the per-value ceiling. Anything else
is stored exactly as it is today, with no frame.

A framed value carries the deflate algorithm unless one of two things is true: it began with `0x00`, or
deflate did not make it smaller. In both cases the frame carries the verbatim algorithm and the body is
the serialized object unchanged.

No Kubernetes storage serializer produces a leading `0x00`, and that is the property the read path
depends on: it decides whether a value is framed by testing the first byte, which is exact only if every
value beginning with `0x00` is framed. Suppose one arrived anyway, from a serializer change or from a
third party driving the transformer chain with its own encoder, and suppose it were stored without a
frame. A later read would see the leading `0x00`, conclude the value is framed, and either fail or
return garbage. Framing it unconditionally removes that possibility rather than relying on the
invariant holding forever.

Because the `0x00` test is applied before the policy and size conditions, such a value is never
compressed, even when the policy selects its resource and its size would otherwise qualify. Compressing
it instead would be equally correct and would save bytes where it is both large and compressible, but it
would add a second path through the write logic for an input that is not expected to occur. It also has
a consequence that matters below: a value framed by this rule stays framed whatever the configuration
later says.

When deflate does not shrink a payload, the frame is still written, with the verbatim algorithm and the
body copied unchanged. The algorithm byte then records that compression was tried and did not help,
which distinguishes such a value from one written before the feature existed. The staleness rule below
depends on that distinction.

### Staleness and migration

The same conditions that decide whether to frame a value on write also decide, on read, whether the
value as stored still matches what the configuration would produce. When they disagree the read reports
the value as stale, and the storage layer rewrites it on its next update. That is the migration
mechanism, and using one rule at both points is what makes it terminate.

One tempting alternative makes framing depend on whether compression actually helped: "frame only if it
shrinks, ties go bare." That rule is perfectly well-defined and deterministic. DEFLATE at a fixed level
is a function, so the same plaintext always yields the same answer, and it does not oscillate.

The problem is what it costs to *evaluate* on the read path. Deciding whether a stored value matches
what the configuration would produce means answering "should this be framed?". Under a length-and-policy
rule that is two integer comparisons. Under a ratio rule it means compressing the plaintext all over
again, because the only way to know whether compression would have helped is to try it. Reads are orders
of magnitude more frequent than writes, and a LIST served from etcd would pay that cost once per object.

Nor can the check be skipped. If a bare value is never examined, objects that would compress well never
migrate when the policy is switched on. The other approximation, treating every bare value as stale when
its resource is configured for compression and its size is above the threshold, fails differently: every
read of a genuinely incompressible object reports it stale, the rewrite produces byte-identical bare
output, and the next read reports it stale again. That is the rewrite loop, an object rewritten forever
without its stored bytes ever changing.

Framing on length and policy alone, with the algorithm byte recording whether compression helped,
removes the problem. "Is it framed?" and "should it be framed?" are then answerable from the same two
cheap facts, so the comparison is exact, and a stale value converges in exactly one rewrite. The price
is that an incompressible object at or above the threshold grows by the frame header, 5 bytes. That is
the cost of a terminating migration.

Because the threshold is one of those conditions, changing it is a migration. Lowering it makes every
object between the old and new value stale; raising it makes already-framed objects in that band stale
in the other direction, and they migrate back to bare. Either way the wave is bounded and
self-terminating, for the same reason that enabling a resource is.

### The read path

Decompression is unconditional and driven by the format alone. The read side is installed whatever the
configuration says, because read support must never depend on configuration: a value
framed by any apiserver has to be readable by every apiserver of that version or newer under any flags,
or dropping a flag would strand data. Inertness by default comes from the write side consulting the
policy, and from the read side returning non-framed bytes unchanged.

One property of RFC 1951 bounds what the format can promise. A DEFLATE stream is a sequence of blocks,
each carrying a one-bit final-block marker, and a decoder stops the moment it finishes the block marked
final. Bytes appended after that block are never examined, so the reader returns the correct plaintext
and never reports them. Trailing junk is therefore undetectable, and a frame with bytes appended still
decodes to the right object. The frame is not canonical, which is why byte-identity is never used as a
staleness signal. An attacker who can append to a stored value also cannot change the decoded object, so
the corruption vector worth worrying about is bit-flips inside the stream rather than appends.

### Bounding decompression

This is the first apiserver path that allocates based on length metadata read from storage, so the bound
is explicit rather than argued. The declared output size is checked against the ceiling before anything
is allocated; there is exactly one right-sized allocation; the stream is bounded in both directions
independently of what it declares; and concurrent inflations are capped process-wide by a semaphore that
is acquired with the request context, so a caller that gives up stops waiting.

The cap is derived rather than configured, at twice `GOMAXPROCS` bounded to between 4 and 384. A
`maxConcurrentDecompressions` field is proposed for beta alongside the size ceiling, since no single
formula suits both a memory-constrained control plane and a read-heavy one. Two properties of such a
field are worth stating: it would be process-scoped while `resources` is per-resource, because the
semaphore protects process memory rather than any one resource; and because the read path must stay live
with no configuration file at all, the derived default would still apply in that case,
which is safe but not tunable.

The memory ceiling either way is `n × (40 KiB + MaxPlaintextBytes)`, approximately `n × 1.54 MiB`, or
~591 MiB at the effective maximum of 384, and only if every concurrent inflation is simultaneously a
maximum-size object. A count is the natural unit because that is what the semaphore admits, but that
byte figure is what an administrator should size against.

### What authenticates a frame

Whether a corrupted frame is caught before it reaches the decompressor is entirely a property of the
layer beneath, and specifically of whether that layer authenticates on *read*:

- Authenticated Encryption with Associated Data (AEAD) covers `aesgcm`, `secretbox` and `kms` v2. Each
  stores a tag over the ciphertext and rejects any modification with overwhelming probability before
  releasing a single plaintext byte. Corruption never reaches the decompressor.
- `aescbc`, the `kms` v1 read path, and no encryption at all offer identical guarantees, namely none.
  None of the three stores a tag, MAC or redundancy that a reader will refuse to look past. `kms`
  v1 belongs here despite writing AES-GCM: its base transformer is a union of GCM and AES-CBC, kept so
  that values written before v1.25 remain readable, and a value that fails GCM authentication is retried
  under bare CBC, whose output then reaches the decompressor.

AES-CBC deserves a note because "unauthenticated" understates it. CBC pads the plaintext, prepends a
random IV, and XORs each block with the previous *ciphertext* block before applying the block cipher.
That provides confidentiality only: decryption always succeeds, since any 16-byte-aligned input
decrypts to something, and the sole integrity signal is PKCS#7 unpadding. Unpadding rejects
roughly 255 of every 256 corruptions that randomise the final block, since a random block carries valid
padding with probability about 1/256, and detects nothing at all elsewhere. It is also malleable in a
structured way: flipping bit *i* of ciphertext block *n* flips exactly bit *i* of plaintext block *n+1*
while randomising block *n*, giving an attacker a chosen-bit-edit primitive at the cost of destroying the
preceding block. Upstream documents `aescbc` as not recommended for this reason.

For the configurations that do not authenticate, compression changes the shape of undetected
corruption. A bit flip is more likely to be noticed, because it usually breaks either the DEFLATE stream
or the decode that follows, but a flip that does slip through corrupts far more than one byte, since a
wrong back-reference corrupts everything downstream of it. Whether to close that with a checksum in the
frame is an open question for reviewers rather than a decision this KEP makes; see
[Open Questions](#open-questions).

### API Priority and Fairness, and size accounting

Today, when the size-based LIST cost estimator is enabled, APF charges a LIST by estimating the memory it
will hold: the number of objects it expects to load, multiplied by the average stored size of that
resource's objects, divided into seats of 100 KB. The object count depends on the request, being one for
a request naming a single object, and the whole resource or half of it for a selector. A LIST served from
the watch cache is additionally limited to a fixed ceiling of about 1 MB, so at most ten seats. The
average size itself comes from the resource size estimator, which averages the length of the stored
value.

That is the gap. The recorded size is the compressed, encrypted one, but the quantity APF wants is the
memory the LIST will occupy, which is bounded below by the plaintext. On LISTs served from the watch
cache the fixed ceiling already limits the damage; on LISTs delegated to etcd nothing does, so at a 6x
ratio APF would charge a fraction of the seats it should and admit substantially more concurrent LISTs
than the apiserver can hold. That path is also the one that performs the decompression, and the only
client-drivable amplification vector in this design.

The watch path is not such a vector. The watch cache opens a single etcd watch per resource and decodes
each event once before dispatching the decoded object to every client watcher, so a client cannot create
etcd watches, and inflation there is bounded by the resource's write rate.

The correction is for the estimator to scale its average by an expansion factor observed on reads, a
moving average over actual plaintext-to-stored pairs, floored at 1.0 so the result can only ever be more
conservative than today's number. Any wrapper sitting between the estimator and the compression layer
has to pass that factor through, or the correction is silently lost. A residual remains and is out of
scope: the expansion from plaintext to decoded object graph was already unaccounted for before this
KEP.

### Error classification and version skew

Two read failures are deliberately not treated as corruption: an unsupported algorithm, and a malformed
frame. The first was written by a newer apiserver and is perfectly readable by one; the second is more
likely a bug in this layer than damaged bytes. Reporting either as a `CorruptObjError` would authorise
deletion through `DeleteOptions.ignoreStoreReadErrorWithClusterBreakingPotential` and let an operator
destroy an undamaged object, so neither is classified that way and the read fails outright instead.

One consequence is that a LIST aborts on the first such value rather than skipping it, so a single bad
value fails the whole list.

GOAWAY was considered for the skew case and rejected. Setting `Connection: close` on such a response
would prompt the client to reconnect and perhaps land on a newer apiserver, but it does not help: a LIST
is already all-or-nothing here, so GOAWAY changes which server the next attempt reaches, not whether this
one works. It also cannot ask for an apiserver that implements a particular algorithm, so the retry is a
coin flip; it would make a skew violation quieter when it should fail identically every time; and because
the header would depend on a stored object's bytes, one unreadable object would tear down every
connection that lists it.

The mitigations instead are that the alpha algorithm set is exactly `stored` and `deflate`, so the error
is unreachable within a supported skew window, and that any future algorithm must ship read support a
release before any release writes it.

### Observability

This KEP adds six ALPHA metrics. `compression_operations_total{resource,operation,outcome}`
distinguishes values compressed from values stored verbatim from values not framed at all, so an
operator can tell whether the size floor or the policy is the reason a resource is not being compressed.
`compression_duration_seconds{operation}` exists because compression time is otherwise invisible: no
existing storage-layer metric or trace covers this stage.
`compression_rewrites_needed_total{resource,direction}` bounds and attributes the write amplification
of enabling or disabling a resource, and goes quiet once a migration converges.
`compression_inflation_wait_seconds` records time spent waiting to acquire an inflation slot. It is
unlabelled, matching the process-wide scope of the semaphore it measures, and it exists because
saturation of that cap is otherwise only visible as latency with no local explanation.

`compression_unsupported_algorithm_total{resource}` is labelled and is not pre-initialised, so a child
series is created only on first increment. A healthy cluster carries zero series for it, which is
strictly cheaper than an unlabelled counter sitting at zero forever, and a broken cluster gets exactly
the label values it needs.

`compression_format_errors_total{resource}` carries the resource and, like the counter above, is not
pre-initialised. It deliberately omits which check rejected the frame, and not for cardinality reasons.
Under the providers that do not reject tampering on read, an actor who can write bytes into etcd could
submit candidate values and learn from `/metrics` which validation step each one failed, using that
feedback to work towards bytes that pass; naming the failing check would turn the counter into exactly
that channel. The failing check belongs in a log line at high verbosity instead, visible to an operator
but not to a `/metrics` reader. The resource label carries no such signal. Neither error counter is
client-drivable, since both arise from stored bytes rather than request content, so there is no
cardinality attack.

Tuning `minSize` requires the size distribution of the values being skipped, and
`outcome="unframed_below_threshold"` reports only that they were skipped, not how close they came. The
threshold is therefore a knob with no feedback loop in `/metrics`, by design: an operator sizing it works
from the size distribution of their own stored objects, measured offline, rather than from a dashboard.

No ratio or byte-count metric is exported. On a resource holding very few objects the `outcome` label
already makes one property of each write observable to whoever can read `/metrics`: whether the object
crossed the framing size floor. That is not a new capability, since reading `/metrics` is an
already-privileged cluster-scoped grant and object length is exposed more coarsely through
`apiserver_storage_size_bytes` and the resource size estimate. It is also narrower than it looks:
because `stored` exists, the label reflects only the size floor and the `0x00` escape, never
compressibility. A per-resource ratio would be a genuinely finer content-dependent signal, and so none
is exported.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

Writing a frame is a one-way commitment, so the unit tests are organised around the properties that
keep it safe rather than around the code that implements it. Two error sentinels carry most of the
weight and are worth naming first: a malformed-frame error, meaning the stored bytes claim to be a frame
but do not parse, and an unsupported-algorithm error, meaning the frame parses but names an algorithm
this binary does not implement, which is version skew rather than damage. The read path returns no
third kind of error, and the storage layer relies on that to decide what is not corruption.

All of the following are required for **alpha**.

- Reader totality: a table over every rejection branch, covering a truncated or wrong magic, an unknown
  algorithm id, a declared length of zero, above the cap and at `1<<40`, a truncated varint, a corrupt
  stream, and a stream shorter or longer than declared. Every failure is one of the two sentinels, and
  nothing panics.
- Round-trip over a matrix of lengths, from empty to above the ceiling, against compressible,
  incompressible and real serialized objects.
- A fuzz target over the reader asserting the same properties on arbitrary input: the input is never
  mutated, every error is one of the two sentinels, the output is nil on error, and its length never
  exceeds the ceiling. Checked in as a native Go fuzz target with a seed corpus, so the unit job
  replays it, and registered with the kubernetes OSS-Fuzz build so it also runs in the daily fuzzing
  that project already does.
- Values written before this feature read back unchanged: drive the protobuf, JSON and CBOR storage
  serializers over several object shapes, asserting that none emits a leading discriminator and that
  each output reads back byte-identical.
- Write and read agreeing: a value written under a configuration is never reported stale under that
  same configuration, and a value framed under one configuration reads back under every other. This is
  what makes a migration converge in one rewrite instead of looping.
- Bounded allocation, asserted on bytes allocated rather than on the error alone:
  - a declared length above the ceiling is refused before anything is allocated;
  - a frame declaring a legal size but carrying a stream that inflates well past it stops at the
    declared length rather than following the stream;
  - a stream that ends early, or runs past its declaration, is refused rather than yielding a short or
    truncated object.
- Never corruption, which has two independent guards and needs both. Neither sentinel is classified as
  a corrupt object, so nothing invites a delete; and separately the read path refuses to hand a
  sentinel to the unsafe-deletion flow at all, so nothing permits one. A regression in either lets an
  operator destroy an undamaged object. A read abandoned while waiting for an inflation slot returns a
  plain context error rather than either sentinel, so it needs the same exclusion for the same reason.
- Inertness: with no configuration file the write path frames nothing and emits no metric samples, and
  a store built with no policy still reads back values framed earlier. This has to be asserted in the
  package that assembles the transformer chain, because the mistake being guarded against is a wiring
  one, either installing the write path where it should be inert or leaving the read path out, and
  neither is visible from inside the compression package's own tests.
- The APF correction, in three parts:
  - the observed plaintext-to-stored factor itself, including that it never falls below 1.0;
  - that the resource size estimator multiplies its average by that factor, and that the raised average
    reaches API Priority and Fairness as additional seats;
  - that every transformer wrapping the compression one re-exposes the factor. The store finds the
    factor by testing whether its transformer implements an optional method, so a wrapper that neither
    implements nor delegates it makes that test come back negative, and the store falls back to 1.0
    with nothing logged and nothing failing.

  The size-based LIST cost estimate is itself a feature gate, on by default since v1.34 but still
  disableable. With it off, APF charges a LIST by object count, which compression does not change, so a
  case covering that records that there is nothing for the correction to do.
- Configuration validation.
- Concurrency under `-race`, since the compressor and decompressor state is pooled and shared across
  requests.

Coverage of the packages the implementation touches, measured before any of it lands:

- `k8s.io/apiserver/pkg/storage/value`: `2026-09-22` - `90.7%`
- `k8s.io/apiserver/pkg/storage/etcd3`: `2026-09-22` - `81.3%`
- `k8s.io/apiserver/pkg/storage/storagebackend/factory`: `2026-09-22` - `65.7%`
- `k8s.io/apiserver/pkg/server/options`: `2026-09-22` - `29.4%`
- `k8s.io/apiserver/pkg/apis/apiserver/validation`: `2026-09-22` - `96.0%`
- `k8s.io/apiserver/pkg/apis/apiserver/load`: `2026-09-22` - `88.0%`
- `k8s.io/apiserver/pkg/util/flowcontrol/request`: `2026-09-22` - `91.7%`

##### Integration tests

The following go in `test/integration/controlplane/transformation`, which already starts an in-process
apiserver, reads raw values out of etcd, plants hand-crafted raw bytes, and restarts the apiserver
against the same backend. All are required for **alpha**.

- A selected resource above the floor is stored as a frame, asserted on the raw etcd bytes, and reads
  back equal to what the client sent. An incompressible sibling gets the verbatim frame.
- With no configuration file the raw bytes still lead with the protobuf prefix, and a value below the
  floor stays unframed even with the policy on.
- Values framed with the policy on still serve after a restart with the policy removed, and the raw
  bytes stay framed until something rewrites them.
- Enable, disable and re-enable across three restarts, checking the raw framing at each step.
- A no-op update after a policy flip changes the resource version exactly once and then stops.
- A planted frame carrying an unknown algorithm id fails the read, is not reported as corruption, and
  is refused by unsafe deletion. A planted frame with a lying declared length does the same through the
  other sentinel.
- A frame written by this release, pinned as a literal at a fixed etcd path, so that future releases
  keep proving it still decodes. This is the regression anchor for the skew rule.
- A malformed configuration file, and a `minSize` outside its legal range, each fail startup.

One more belongs in `test/integration/storageversionmigrator` and is a **beta** item rather than an alpha
one, since it is what exercises the documented pre-downgrade procedure: deselect a resource, restart,
migrate, then assert that no stored value under that resource leads with the frame discriminator and
that the number of values scanned equals the number created, so an empty scan cannot pass. Run it in
both directions, since the staleness rule is symmetric.

`TestDefaultStorageEncoding` and `TestEtcdStoragePath` serve as the default-inertness guard for the
whole resource surface without being modified, since both decode raw etcd values and would fail if
framing ever leaked on by default.

##### Upgrade and downgrade tests

Upgrade and downgrade tests are for beta.

- Mixed configuration inside one release: two apiservers on the same etcd, one with the policy and one
  without, both serving the same resource. Every read succeeds from either, and the disagreement shows
  up only as the rewrite counter climbing in both directions.

TODO: figure out what is possible for upgrade and downgrade tests and then add more test cases.

##### e2e tests

None. The feature is turned on by a kube-apiserver flag, which an e2e test cannot set against a running
cluster, and it adds no API surface a client can observe: objects round-trip unchanged whether or not
they are compressed. Everything an e2e test could assert is either asserted at the integration level
with access to the raw etcd bytes, or invisible by design.

##### Scale test

Four runs, all required for alpha.

1. The watch-cache initialisation benchmark with a compression axis. It already seeds a large number of
   Pods into a live etcd and times the cacher from construction to ready, which is the worst read case
   this KEP names, and a presubmit can afford it.
2. A scalability run on a change carrying a policy that covers a large resource, measuring the API call
   latency and pod startup SLOs, apiserver CPU and memory, etcd database size, and write throughput.
3. The same run with no policy.
4. The same run with compression and encryption at rest both enabled for the same resources, measuring
   the same metrics.

Comparing 3 against the state before the change isolates what installing an unused read path costs a
cluster that never opts in. Comparing 2 against 3 isolates what compressing actually costs. Comparing 4
against 2 isolates what compression costs alongside encryption.

Run 4 has no precedent to build on: no existing scalability job enables encryption at rest, so that
configuration has to be added to the job before the run is possible at all.

Backward-compatible scalability improvements that these runs suggest are beta work, not alpha.

### Graduation Criteria

Compression is never on by default, including at GA. No resource is compressed unless an
administrator writes a configuration file and points `--storage-compression-config` at it; compressing
by default is an explicit Non-Goal. There is no feature gate, so the stages do not move a default: what
graduates is the configuration API, `v1alpha1` at alpha and `v1` at GA, and with it the compatibility
promise the file carries.

One sequencing rule is hard. Decompression is unconditional and a written frame is a permanent, one-way
commitment, so ordering protects downgrade rather than upgrade. No release may make frame-writing
reachable by default, and every release in which it is reachable at all must state, in the release note
and the flag help, that opting in forfeits downgrade to the previous minor for the selected resources
until a migration has rewritten them uncompressed. At v1.38 that warning is doing real work, because
nothing can teach v1.37 to inflate a frame.

Any new algorithm must ship read support one release before any release may write it.

#### Alpha

Targeted at v1.38.

- [ ] The flag is unset by default; writes are reachable only with a configuration file and are disabled
  by removing it; a malformed or contradictory configuration fails startup rather than silently doing
  nothing.
- [ ] Open question 1, whether a frame carries a checksum, is decided. It must be settled before alpha
  ships rather than at beta, because adding integrity afterwards requires a new algorithm id and a
  release of read-before-write skew. If the answer is a checksum, a mismatch needs a classification
  distinct from a format error, so that it stays eligible for the unsafe-deletion flow of KEP-3926.
- [ ] Six metrics registered at ALPHA stability, matching the metrics list in `kep.yaml`.
- [ ] The release note and the flag help carry the downgrade-forfeit warning above. If open question 1
  is decided against a checksum, they also carry the caveat that a frame's only integrity protection is
  the encryption provider's, which means none at all under AES-CBC or with encryption disabled.
- [ ] A sig-auth review covering the compressibility side channel specifically, not the KEP generally.
  The residual and its threat model are in [Risks and Mitigations](#risks-and-mitigations). The review
  decides whether that residual is acceptable at alpha, whether per-resource opt-in and the absence of
  any ratio or byte-count metric are jointly sufficient, and whether the startup warning for an
  explicitly named `secrets` is adequate. Sign-off is recorded in Implementation History.

Open question 2, a per-resource compression level, is a beta item. Open question 3, the discriminator
invariant belonging in apimachinery, is carried by beta's closing of known gaps.

#### Beta

Targeted at v1.39.

- [ ] Open question 2 is resolved: whether the compression level becomes per-resource. It needs no format
  change, since the level is a writer-side choice the decoder never reads.
- [ ] The configuration shape is finalised, either promoted to a beta API version or kept at v1alpha1 with
  a stated reason, along with the size ceiling and the inflation concurrency limit.
- [ ] Raw-etcd tooling can read a frame. This is the only dependency outside kubernetes/kubernetes:
  `auger` and similar tools parse etcd values directly and fail on a frame today. Beta needs a released
  `auger` that inflates the body, and documentation of which tools read the frame at which versions.
- [ ] A downgrade procedure documented and exercised in the migration suite: deselect the resource,
  restart, migrate, and only then downgrade, asserting afterwards that no stored value begins with the
  frame discriminator.
- [ ] A scale test, with a policy covering a large resource, showing no regression against the existing
  API call latency SLOs. The same run with an empty policy confirms an inert installation costs nothing.
- [ ] Operator documentation for each metric stating what a non-zero value means and the action to take:
  converged, skew or tampering, or investigate rather than roll back. No ratio or byte-count metric added.
- [ ] Feedback gathered from alpha adopters on which resources they selected, how they drove the
  amplification burst, and whether the raw-etcd tooling gap blocked an investigation. Known gaps closed,
  including open question 3.

#### GA

- [ ] No unresolved issues reported by beta adopters, and no open issue attributable to the at-rest
  format.
- [ ] The configuration API reaches `v1`, with `resources` still empty by default.

#### Deprecation

The read side can never be deprecated. Once any cluster has written a frame, every subsequent
kube-apiserver must be able to inflate one, indefinitely. The algorithm ids carry no sunset date,
because notice is useless against data already at rest.

"Removal" can therefore only mean removing the write path (the flag, the configuration type, the
policy and the writer), and even that requires a migration first. Removing the writer removes the
policy, which makes every framed value stale at once and rewrites it uncompressed on its next update.
That terminates, but as an uncontrolled amplification burst on precisely the largest resources. Removal
would therefore be announced first, with the write path and the `None` algorithm kept for at least two
releases and for whatever deprecation window the configuration type's stability level requires, and
migration to uncompressed documented as a prerequisite before the writer goes.

This KEP deprecates nothing that already exists. `--storage-compression-config` is new and supersedes
nothing: not `--storage-media-type`, not response compression, not any etcd-side setting.

### Upgrade / Downgrade Strategy

Upgrade is a no-op. `--storage-compression-config` is unset, so an untouched cluster stores
byte-identical values after the upgrade. The decompression path is present regardless but stays inert:
it finds no frames, records no metrics, and returns every value unchanged. There is no ordering requirement among apiservers, because a read is decided by
the stored bytes rather than by the reading apiserver's configuration.

Enabling a resource costs a one-time write amplification. Once a resource is selected, each of its
objects between the configured floor and the ceiling is reported stale on read, which deliberately defeats
the optimisation that normally skips an etcd write when a client's update serialises to bytes identical
to what was read. Idempotent controller updates that previously cost nothing each become one real write
and one revision, until every object has been rewritten once. It is bounded and self-terminating, but
on a large cluster with chatty controllers it is a visible burst in writes, revisions and
pre-compaction database growth. Driving it deliberately with a storage version migration (KEP-4192,
whose controller has been enabled by default since v1.37) is preferable to waiting for organic churn.
Where that controller is disabled the burst still terminates, but its timing is not under the
operator's control.

Disabling is safe for reads at any time and can never strand data, though it decompresses nothing. An
operator can remove the resource from the configuration, exclude it ahead of a wildcard by giving it
the `None` algorithm, or drop the flag. Existing frames stay on disk until
something rewrites them: the staleness verdict flips direction and they migrate back by the same
mechanism, at the same amplification. Disabling makes the write path unreachable while the read path
stays active; it does not shed a frame.

Graduating changes nothing observable. The flag is the only switch at every stage, so a cluster that
does not set it sees no difference, and no upgrade can start compressing for a cluster that merely has a
file on disk. Downgrading between two releases that both support the format is equally uneventful: the
flag means the same thing in each, and a frame written by either is readable by both.

Downgrade below the first decompress-capable release is the serious case, and it is an outage rather
than a degradation. Decryption succeeds, because the encryption envelope is unchanged and in an
unencrypted cluster the frame simply is the value; the frame then reaches a storage codec that
recognises nothing. A single undecodable value removes serving for its entire resource, indefinitely:

- A LIST fails entirely rather than partially, because the older apiserver aborts on the first value it
  cannot decode instead of aggregating per item.
- A WATCH terminates on the first undecodable event, so the watch cache loses its reflector and answers
  every subsequent re-list with an initialisation error.
- The affected objects are classified as corrupt, because the older binary lacks the exclusion that
  keeps a format error out of that category. That makes undamaged objects eligible for unsafe deletion.
  The correct response is to roll forward, not to delete.

The only sound preparation is a completed migration per compressed resource, in this order: remove
those resources from the configuration on every apiserver and restart, then run a migration for each
and wait for it to succeed, and only then roll back. The order matters, because a migration run while
the resource is still selected rewrites each object into a fresh frame and achieves nothing.

Monitor `etcd_requests_total{operation="update"}` for the amplification, alongside etcd revision and
database-growth signals, and `apiserver_storage_size_bytes` for the objective. After a downgrade, on the
older binary, `apiserver_storage_decode_errors_total` and the watch-cache initialisation counters are
where this failure appears.

### Version Skew Strategy

This is a kube-apiserver-only change to the at-rest storage format. It adds no REST API type, no served
field and no negotiated wire capability, so kubelet, kube-proxy, kube-scheduler,
kube-controller-manager, CSI/CRI/CNI, `kubectl` and client-go carry no new constraint, and etcd sees an
opaque value either way. The only skew that matters is between apiservers backed by the same etcd, plus the
configuration file itself, which no earlier release can parse.

Across an N/N-1 boundary, an apiserver from the previous release has no decompression path, so a frame
reaches its decoder and fails, taking the whole LIST with it. Hence the governing rule: no apiserver
may write a frame until every apiserver backed by the same etcd can read one. Because read and write
support ship together in the same release, that rule is discharged by operator sequencing rather than by
a release boundary. Upgrade every apiserver, confirm the rollout, then add the configuration file.
Writing requires one deliberate, restart-scoped act: pointing `--storage-compression-config` at a file
that names a resource with an algorithm other than `None`. Reverting it stops new frames from being
written but leaves every existing frame readable, because the read path never consults the policy.

Mixed configuration across apiservers is the case an operator can actually create. Reads are
unaffected: the read path is installed unconditionally and decides from the stored bytes, so a frame
written by one apiserver is readable by every other whatever their policies say. Writes are a different
matter. The same conditions decide both whether to frame a value and whether a stored value is stale, so
two apiservers with different configurations disagree about staleness on exactly the objects they
disagree about writing. A mid-sized ConfigMap is then rewritten framed by one and bare by the other,
indefinitely. Each rewrite converges for the apiserver that performed it, but the fleet as a whole does
not, so the one-time amplification burst becomes permanent. No data is lost and nothing computes the
wrong answer; the cost is sustained and pointless write traffic. Apiservers can disagree on two axes,
which resources are selected and what `minSize` applies to them, and both produce this same symptom.

Keep the configuration identical on every apiserver backed by the same etcd, exactly as the encryption
configuration already requires. The file is read once per process, so disagreement comes from a partial
rollout, or from an in-place edit that only a later restart applies to one replica. No automated
detection is proposed; the symptom is
`apiserver_storage_compression_rewrites_needed_total` climbing in both directions at once and never
going quiet.

An unsupported algorithm cannot be reached within any supported skew at alpha, since only two
algorithms exist and both are readable from the first supporting release. Any future algorithm must
ship read support one release before any release may write it. Why that error is classified as skew
rather than corruption, and why GOAWAY was rejected as a response to it, are in
[Design Details](#error-classification-and-version-skew).

Custom resources and aggregated apiservers are covered rather than excluded: the configuration reaches
every store, including the one built for custom resources, and each aggregated apiserver stores its own
resources under its own keys, with its own configuration.

### Open Questions

Question 1. How should a frame's integrity be protected, if at all? The measurement in
[What authenticates a frame](#what-authenticates-a-frame) shows that for `aescbc` and for unencrypted
clusters, compressing raises single-bit-flip detection from 4.50% to 72.08% but raises expected
silently-corrupted bytes per flip from 0.955 to 51.6. Four options, offered for reviewer input rather
than decided here:

- Store a CRC32 of the plaintext in the frame. That costs four bytes, taking framing overhead to 9 bytes
  for `stored` and 10-12 for `deflate`, and roughly 1% of compression CPU. Detection goes to about 100%.
  A mismatch would need a classification distinct from a malformed frame, because a malformed frame is a
  bug in this layer whereas a checksum mismatch is real corruption and should stay eligible for the
  unsafe-deletion flow of KEP-3926.
- Use gzip rather than raw DEFLATE. That gets a CRC32 for +18 bytes with no hand-rolled integrity code.
  Larger, but a reasonable answer if reviewers would rather not review a custom frame.
- Document that compression should be paired with an AEAD provider. Zero cost, but it protects
  nobody who ignores it, and since encryption at rest is opt-in this leaves the exposure in place for
  what is likely the majority of clusters.
- Make the checksum configurable. Not recommended: it doubles the format matrix and makes a frame's
  meaning depend on configuration.

Question 2. Should the DEFLATE level become per-resource at beta? A per-resource level is a plausible
beta addition for large-object resources, and is out of scope for alpha because the level is a
writer-side choice the decoder never needs, so it can be added without a format change.

Question 3. The `0x00` discriminator is an invariant about apimachinery that this KEP does not own. It
deserves a note in apimachinery rather than living only here and in one test.

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

###### How can this feature be enabled / disabled in a live cluster?

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [x] Other
  - Describe the mechanism: one kube-apiserver flag, `--storage-compression-config`, pointing at a file
    that names the resources to compress with an algorithm other than `None`. Unset, nothing is
    compressed. No feature gate, for the reasons in [Proposal](#proposal).
  - Will enabling / disabling the feature require downtime of the control plane? No, but each apiserver
    restarts, since the configuration is read once per process. On HA that is a rolling restart.
  - Will enabling / disabling the feature require downtime or reprovisioning of a node? No.

###### Does enabling the feature change any default behavior?

No. Without a configuration file stored bytes are byte-identical. Selecting a resource changes only its
at-rest bytes, plus a one-time write amplification while existing objects migrate. Nothing visible
through the API changes. Raw-etcd tooling is the exception: it can no longer read those values.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

The write path, yes and immediately: remove the resource's entry, give it the `None` algorithm, or unset
the flag. The read path, no, and that asymmetry is what makes rollback safe: it can never strand data.

Rolling back does not uncompress anything. Frames revert to bare values only as something rewrites them,
at the same amplification cost as enabling. Reverting the bytes, needed only before downgrading below
the first decompress-capable release, requires a storage version migration per resource; see
[Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).

###### What happens if we reenable the feature if it was previously rolled back?

Re-enabling behaves exactly like enabling for the first time. Only the objects rewritten to bare values
while the feature was off need framing again; objects still framed are left alone, because their stored
form already matches what the configuration asks for. The cost is therefore proportional to how much
churn happened during the gap rather than to the size of the resource, so repeated enable and disable
cycles do not compound. A resource left half-migrated is a valid steady state, since every apiserver
reads both forms.

###### Are there any tests for feature enablement/disablement?

The implementation carries unit coverage of an unconfigured apiserver framing nothing and emitting no
metric samples, of a stale value converging in exactly one rewrite in either direction, which is what
makes enable and disable terminate, and of a value framed under one configuration reading back under any
other. The integration coverage does not exist yet: the restart-driven enable, disable and re-enable
cycle, with the raw etcd bytes asserted at each step. Both are in the [Test Plan](#test-plan).

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

No workload is affected. Only how kube-apiserver stores bytes changes, and etcd sees an opaque value
either way. No interleaving can corrupt data, because a read is decided by the stored bytes rather than
by the reading apiserver's configuration, so a mid-rollout fleet needs no coordination.

Every remaining failure lands on the control plane. A configuration mistake fails startup rather than
degrading: a malformed file, an unknown algorithm, an unreadable path, a `minSize` outside its legal
range, or a selector already claimed by an earlier entry. A rolling update
therefore loses one replica at a time while the rest keep serving. Applying an unvalidated file to every
replica at once is what turns that into an outage.

A blue-green update has the opposite shape. Both fleets serve the same etcd during the cutover, so the
entire window is a mixed-configuration window and the rewrite churn described in
[Version Skew Strategy](#version-skew-strategy) lasts as long as the overlap. If the fleets also differ
in version, the green fleet must not write frames while a blue apiserver that cannot read them is still
serving.

Rollback within the same minor version is a configuration change applied at the next restart. Rollback
below the first decompress-capable release is the one genuinely dangerous transition; see
[Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy).

###### What specific metrics should inform a rollback?

One signal argues for switching the feature off, and none argues for deleting an object.

- Sustained p99 write-latency or apiserver CPU regression on a selected resource, or etcd write and
  revision rate still elevated beyond the one-time burst: remove that resource from the configuration.
- `apiserver_storage_compression_rewrites_needed_total` climbing in both directions and never going
  quiet: apiservers disagree on configuration. Reconcile the file, since rolling back does not fix it.
- `apiserver_storage_compression_unsupported_algorithm_total` non-zero: a peer writes frames this
  apiserver cannot inflate. Roll the lagging apiserver forward.
- `apiserver_storage_compression_format_errors_total` non-zero: investigate. Rolling back makes this
  worse, since an older apiserver has fewer ways to read the same bytes.

No counter can confirm a downgrade is safe. They are driven by reads, so a quiet counter means nothing
has looked, not that no frames remain.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No. The upstream implementation PR is not open, so upgrade and downgrade have not been exercised end to
end. The transition most worth testing is a downgrade below the first decompress-capable release preceded
by a completed migration, since that is the one path where getting it wrong removes serving for a whole
resource.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

`apiserver_storage_compression_operations_total` tells an operator whether compression and decompression
are actually happening on the resources they configured. Its `operation` label separates the two
directions, and its `outcome` label separates compressed values from those stored verbatim and from those
not framed at all. No workload can enable or observe the feature.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
- [ ] API .status
- [x] Other (treat as last resort)
  - Details: there is deliberately no end-user signal, since end users cannot read `/metrics` and objects
    round-trip unchanged whether or not compression is on. The operator watches the compressed `outcome`
    on `apiserver_storage_compression_operations_total` to confirm framing is happening, and
    `apiserver_storage_size_bytes`, which is per etcd cluster rather than per resource, for the benefit.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Both error counters at zero. `apiserver_storage_compression_rewrites_needed_total` quiet again within one
migration per resource added or removed. No regression against the existing API call latency SLOs, which
remain the real ceiling: compression adds work to every mutating request for a selected resource, so the
write-latency SLI rather than the compression histogram decides whether a resource stays enabled.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `apiserver_storage_compression_operations_total`,
    `apiserver_storage_compression_duration_seconds`,
    `apiserver_storage_compression_format_errors_total`,
    `apiserver_storage_compression_unsupported_algorithm_total`,
    `apiserver_storage_compression_rewrites_needed_total`,
    `apiserver_storage_compression_inflation_wait_seconds`
  - [Optional] Aggregation method: rate by resource and outcome; p99 by operation; any non-zero total on
    either error counter; rate by resource and direction for rewrites, watched for reaching and holding
    zero; p99 of the inflation wait
  - Components exposing the metric: kube-apiserver
- [ ] Other (treat as last resort)

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. The metric an operator would most want, the realised compression ratio per resource, is deliberately
not exposed: on a resource holding few objects it approaches per-object compressibility, a finer side
channel than the length channel compression already opens, and `/metrics` is readable by a far broader
set of principals than etcd is. A count of frames remaining cannot be a metric either, since the counters
are read-driven and an untouched object is never observed, which is why the pre-downgrade procedure is a
migration rather than a dashboard check.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. Storage version migration (KEP-4192) is worth a note, since it schedules the one-time write
amplification and a completed migration is the only sound preparation for a downgrade below the first
decompress-capable release. Neither reads nor writes depend on it: without it the amplification still
terminates, just on organic churn rather than on the operator's schedule.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No. `StorageCompressionConfiguration` is a configuration-file kind read once at startup, never persisted
and never served, so no object limit applies to it.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No change in counts, and no increase in size for almost every object: stored size decreases, with ratios
in [Measured on production data](#measured-on-production-data). A small portion grows. A selected value
above the configured floor that does not compress is stored in a verbatim frame, so it grows by five
bytes.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

TODO: pending the scale test in the [Test Plan](#test-plan).

With no configuration file the write path is a length comparison and the read path a comparison of the
first byte, and a microbenchmark of the unframed read path is what bounds that. For a selected resource
each mutating request adds one compression pass and each read of a framed value one inflation, which is
cheaper. The worst read case is watch-cache initialisation, a full LIST inflated object by object while
the request waits. No SLO measurement exists for any of this yet, so the size of the effect is unstated
rather than claimed to be zero.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Only kube-apiserver, and the net effect may be neutral or better. Compression costs CPU on the write path
for selected resources and inflation on the read path, plus memory for pooled compressors and for bounded
concurrent inflation; see [Bounding decompression](#bounding-decompression). Against that, smaller stored
values mean fewer bytes over the wire to and from etcd, which saves both CPU and the buffers carrying
them. Whether that saving offsets the compression cost is TODO; the scale test in the
[Test Plan](#test-plan) is what answers it.

Network traffic to etcd decreases with the wire size.

Admission control needs correcting separately, since it charges a LIST by stored bytes while the memory it
occupies is the plaintext; see [API Priority and Fairness](#api-priority-and-fairness-and-size-accounting).

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Compression and decompression are synchronous and in-process, with nothing persisted between requests.
Neither runs if the API server is down, and neither can do anything if etcd is unavailable, since there is
no value to store or retrieve. The feature adds no failure mode of its own to either outage.

###### What are other known failure modes?

- A malformed frame: stored bytes look framed but do not decode.
  - Detection: `apiserver_storage_compression_format_errors_total` non-zero, and the whole LIST fails.
  - Mitigations: none. What it means depends on the provider beneath. Under an authenticated encryption
    provider like KMS v2 the ciphertext was verified before reaching this layer, so the fault is a bug
    here and the fix is a code fix. Under `aescbc` or with no encryption the stored bytes are genuinely
    damaged, and the only recovery is to restore from backup. Either way the object cannot be cleared
    with unsafe deletion, which refuses this class of error.
  - Diagnostics: which check rejected the frame is logged at high verbosity, deliberately not exposed as
    a metric label.
  - Testing: read-path fuzzing, plus aimed corruption under an unauthenticated provider.
- An unsupported algorithm: a newer apiserver wrote the frame.
  - Detection: `apiserver_storage_compression_unsupported_algorithm_total`, by resource.
  - Mitigations: upgrade the lagging apiserver, correlating peer versions first. Not a rollback and not a
    delete.
  - Diagnostics: the unrecognised algorithm id and the storage key are logged.
  - Testing: an integration test writes a value carrying an algorithm id this build does not implement,
    then asserts the read fails without reporting corruption and that unsafe deletion is refused. The
    condition is unreachable within a supported skew, so the test constructs it directly.
- Reads queueing on the inflation limit.
  - Detection: `apiserver_storage_compression_inflation_wait_seconds` p99 rising, with
    `apiserver_storage_compression_duration_seconds` flat. The cost is the wait, not the work.
  - Mitigations: shed concurrent etcd-served LISTs through APF. The limit is derived today; making it
    configurable is proposed for beta.
  - Diagnostics: an abandoned read stops waiting, surfacing as a cancellation rather than as latency.
  - Testing: drive concurrent LISTs to the admission limit.

###### What steps should be taken if SLOs are not being met to determine the problem?

Rule the layer out first: a silent `apiserver_storage_compression_operations_total` means this apiserver
is doing nothing here, though it says nothing about what etcd holds. For write latency, compare a
selected resource against an unselected one of similar size; the remedy is deselection, effective at
restart. For write volume, growth that does not decay means an unfinished migration or disagreeing
configurations. For read latency, see the queueing mode above. If stored size is flat, the `outcome`
label separates unframed values, whether from the size floor or the policy, from values stored verbatim.

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

- The at-rest format changes in one direction only. Once any cluster has written a frame, every later
  kube-apiserver must be able to inflate one, indefinitely, and the read side has no deprecation path.
  That obligation is permanent, for a benefit that stays opt-in at every stage including GA.
- Clusters that never enable writing still carry the read path. It is installed unconditionally and has to
  stay correct against hostile bytes under the providers that do not authenticate, so it needs review and
  fuzzing on every change regardless of how few clusters use it.
- CPU lands on the request goroutine for every read and write of a selected resource, including
  watch-cache initialisation, and KEP-2338 is the precedent for getting that wrong. Enabling a resource
  also costs a one-time write-amplification burst, and an incompressible object above the floor grows by
  five bytes for good.

## Alternatives

- Compressing anywhere other than inside the transformer chain: in etcd, in its storage engine, in the
  filesystem, or above the chain. All yield a ratio near 1.0 for the motivating case, since everything
  outside the chain sees only ciphertext.
- gzip rather than raw DEFLATE. The same algorithm plus a checksum, for 18 bytes and no hand-rolled
  integrity code, and still live under [Open Questions](#open-questions). Raw DEFLATE is the alpha choice
  because the frame already carries a discriminator and a declared length, so only gzip's trailer would
  add anything.
- A compression dictionary shared across objects. The obvious way to help objects near the floor, and
  permanently rejected: shared state across values creates the cross-object oracle that
  [Risks and Mitigations](#risks-and-mitigations) relies on being impossible.
- A write-path dry-run mode that compresses and discards, to estimate the benefit before enabling. It
  samples the write stream rather than the stored set, so it answers a different question from "what would
  my database shrink to", it pays full compression cost for no stored saving, and its per-resource ratio is
  the content-dependent signal [Observability](#observability) declines to export. An offline analyzer
  answers the question better and in private. A read-path sampler would be the least-bad live shape, since
  it observes the stored set and cannot affect what is written.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
