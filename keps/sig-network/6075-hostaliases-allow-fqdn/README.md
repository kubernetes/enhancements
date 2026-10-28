# KEP-6075: Allow FQDN with trailing dot in HostAliases

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Validation](#validation)
  - [Storage: trailing dot is preserved](#storage-trailing-dot-is-preserved)
    - [/etc/hosts Generation](#etchosts-generation)
    - [Behavior reference](#behavior-reference)
    - [Discrepancy with <code>man 5 hosts</code>](#discrepancy-with-man-5-hosts)
  - [Interaction with existing HostAliases semantics](#interaction-with-existing-hostaliases-semantics)
  - [Feature gate](#feature-gate)
  - [Test Plan](#test-plan)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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
<!-- /toc -->

## Summary

Allow a trailing dot in hostnames under `spec.hostAliases[].hostnames`. The trailing dot denotes a Fully Qualified Domain Name (FQDN) as defined in RFC 1034, preventing DNS resolvers from appending the search path.

## Motivation

DNS resolution in modern systems supports two styles of domain name queries: relative and absolute. A relative query (e.g., `example.com`) is subject to search path expansion, where the resolver appends configured search domains and retries if the initial lookup fails. An absolute query (denoted by a trailing dot, e.g., `example.com.`) bypasses this expansion entirely. This distinction is standardized in RFC 1034 and RFC 2181, and is universally implemented across operating systems, system resolvers, and language runtime libraries.

Kubernetes Pods by default configure `ndots:5` in `/etc/resolv.conf`. When an application queries a bare external domain name, the resolver issues parallel A and AAAA queries across all search domains before querying the root. In a stock Kubernetes cluster with 3 search domains, resolving a bare name triggers up to 8 DNS queries (6 wasted queries, a 4x overhead). In cloud environments with 5 search domains, this reaches 12 queries (10 wasted queries). To prevent this query amplification and avoid DNS latency/throttling, applications frequently use absolute FQDNs with a trailing dot (e.g., `api.example.com.`), reducing DNS traffic to just 2 queries.

However, Kubernetes `HostAliases` currently validates hostnames using `ValidateDNS1123Subdomain`, which rejects trailing dots. This prevents users from pinning or overriding absolute FQDN names. In system libc resolvers (glibc 2.31/2.41, musl 1.1.24/1.2.6), `/etc/hosts` matching via `getaddrinfo` and `getent hosts` is strictly literal: an entry `10.10.10.10 api.example.test` will **not** match an FQDN lookup for `api.example.test.` (and vice versa). While Go's pure resolver normalizes trailing dots internally, libc-based runtimes (Python, Java, Node.js, C/C++, Rust) fail to match.

The practical consequence is that workloads using absolute FQDNs to avoid `ndots:5` amplification cannot use `HostAliases` to override those endpoints (for instance, redirecting an external API to a mock/sandbox IP in staging or local environments). Users are forced to either accept search-path amplification or resort to privileged init containers that manually modify `/etc/hosts`.

Containerized testing (across glibc 2.31, glibc 2.41, musl 1.1.24, musl 1.2.6, and Go) confirms that parser tolerance holds: system resolvers cleanly parse dotted hostnames in `/etc/hosts`, do not reject lines, and continue resolving subsequent aliases on the line.

By allowing trailing dots in `HostAliases` hostnames, Kubernetes enables workloads using absolute FQDNs to override host resolution cleanly and securely.

### Goals

- Allow trailing dots in `HostAliases` hostnames to denote fully-qualified domain names (FQDNs), following RFC 1034 semantics.
- Preserve trailing dots in `/etc/hosts` entries so that absolute FQDN queries match correctly in all system resolver implementations.
- Support workloads that perform absolute FQDN lookups without requiring privileged init containers or manual `/etc/hosts` manipulation.

### Non-Goals

- Change any other DNS validation or behavior.
- Add new API fields or types.

## Proposal

Modify `ValidateHostAliases` to unconditionally strip a trailing dot before calling `ValidateDNS1123Subdomain`. Add a feature gate check for `HostAliasesAllowFQDN`:
- On pod creation (`ValidatePod`), reject trailing dots if the feature gate is disabled.
- On pod update (`ValidatePodUpdate`), compare against `oldPod.Spec.HostAliases` (field ratcheting): if the gate is disabled, only reject trailing dots if they are newly added or modified, allowing existing pods with trailing dot hostnames to receive metadata and spec updates without error.

The core validation itself (`ValidateHostAliases`) always succeeds for valid subdomains regardless of gate state.

### User Stories

#### Story 1

A pod runs an application that resolves `example.com.` (with trailing dot, as an FQDN to avoid `ndots:5` search path amplification). The user adds a HostAlias to pin that name to a specific IP:

```yaml
spec:
  hostAliases:
  - ip: "10.10.10.10"
    hostnames:
    - "example.com."
```

The trailing dot bypasses search path expansion and matches the exact literal `/etc/hosts` entry. Without this KEP the pod is rejected by validation.

#### Story 2

A pod runs heterogeneous workloads: some applications query `example.com.` (absolute, FQDN style) while others query `example.com` (relative, search-list style). Rather than create separate HostAlias entries, the user can provide both names in a single entry:

```yaml
spec:
  hostAliases:
  - ip: "10.10.10.10"
    hostnames:
    - "example.com"
    - "example.com."
```

This generates a single `/etc/hosts` entry with both names:
```
10.10.10.10    example.com    example.com.
```
Placing the bare name first (`example.com` before `example.com.`) ensures both query forms match while keeping the canonical name returned by reverse lookups (`gethostbyaddr`/`getnameinfo`) unchanged.

### Risks and Mitigations

**Concern: Use case is rare and undocumented**

This concern underestimates both the prevalence of the use case and the documentation status. RFC 1034 and RFC 2181 explicitly document trailing dots in domain names as the standard representation of absolute FQDNs. In Kubernetes, using absolute names with a trailing dot is a well-documented and standard pattern to avoid the `ndots:5` query amplification penalty (saving 6 to 10 wasted DNS queries per lookup). Containerized testing across glibc, musl, and Go confirms that libc matching against `/etc/hosts` is literal, making trailing dot support in `HostAliases` necessary for any workload querying absolute names.

**Concern: Downgrade compatibility**

Downgrade risk is mitigated by field ratcheting: `ValidateHostAliases` strips trailing dots before DNS validation regardless of feature gate state. When the feature gate is disabled, `ValidatePodUpdate` compares `pod.Spec.HostAliases` against `oldPod.Spec.HostAliases`, ensuring existing pods with trailing dots remain updatable. The risk is therefore confined to clusters that enable the feature and then downgrade to an older minor version that lacks the KEP code entirely.

**Concern: Behavioral inconsistency across resolver implementations**

Containerized testing across glibc 2.31, glibc 2.41, musl 1.1.24, musl 1.2.6, and Go confirms consistent behavior: all tested libc parsers tolerate dotted tokens without rejecting lines or breaking subsequent aliases. Note: When both `name` and `name.` are specified on the same `/etc/hosts` line, Go's pure net resolver returns duplicate IP addresses for lookups because Go strips the trailing dot internally before matching. This is purely cosmetic and does not impact connectivity.

## Design Details

### Validation

Current code at `pkg/apis/core/validation/validation.go`:

```go
func ValidateHostAliases(hostAliases []core.HostAlias, fldPath *field.Path) field.ErrorList {
    allErrs := field.ErrorList{}
    for i, hostAlias := range hostAliases {
        allErrs = append(allErrs, IsValidIPForLegacyField(fldPath.Index(i).Child("ip"), hostAlias.IP, nil)...)
        for j, hostname := range hostAlias.Hostnames {
            allErrs = append(allErrs, ValidateDNS1123Subdomain(hostname, fldPath.Index(i).Child("hostnames").Index(j))...)
        }
    }
    return allErrs
}
```

Proposed change: strip a single trailing dot before DNS subdomain validation:

```go
func ValidateHostAliases(hostAliases []core.HostAlias, fldPath *field.Path) field.ErrorList {
    allErrs := field.ErrorList{}
    for i, hostAlias := range hostAliases {
        allErrs = append(allErrs, IsValidIPForLegacyField(fldPath.Index(i).Child("ip"), hostAlias.IP, nil)...)
        for j, hostname := range hostAlias.Hostnames {
            allErrs = append(allErrs, ValidateDNS1123Subdomain(strings.TrimSuffix(hostname, "."), fldPath.Index(i).Child("hostnames").Index(j))...)
        }
    }
    return allErrs
}
```

`strings.TrimSuffix` removes only a single trailing dot and only when present:
- `"example.com."` → `"example.com"` → valid subdomain
- `"localhost."` → `"localhost"` → valid label
- `"my-server."` → `"my-server"` → valid label (single-label FQDNs are accepted)
- `"example.com.."` → `"example.com."` → rejected by `ValidateDNS1123Subdomain` (label ends with dot)
- `"."` → `""` → rejected by `ValidateDNS1123Subdomain` (empty string)

### Storage: trailing dot is preserved

The trailing dot is stored verbatim in etcd. The API server does not normalize or strip it after validation.

#### /etc/hosts Generation

The kubelet is unchanged by this KEP. It writes hostnames from `HostAliases` verbatim, in the order given, on a single line per `HostAlias` entry via `hostsEntriesFromHostAliases`:

```go
func hostsEntriesFromHostAliases(hostAliases []v1.HostAlias) []byte {
    var buffer bytes.Buffer
    buffer.WriteString("\n")
    buffer.WriteString("# Entries added by HostAliases.\n")
    for _, hostAlias := range hostAliases {
        buffer.WriteString(fmt.Sprintf("%s\t%s\n", hostAlias.IP, strings.Join(hostAlias.Hostnames, "\t")))
    }
    return buffer.Bytes()
}
```

For an entry with both bare and dotted hostnames:
```yaml
spec:
  hostAliases:
  - ip: "10.10.10.10"
    hostnames:
    - "example.com"
    - "example.com."
```
the kubelet generates:
```
10.10.10.10    example.com    example.com.
```

**Rationale for Single-Line Format and Ordering:**

1. **Single-line resolution is universally supported**: Containerized testing across glibc (2.31, 2.41) and musl (1.1.24, 1.2.6) under `--network none` demonstrates that placing `IP name name.` on a single line resolves both `name` and `name.` forward queries correctly on every libc tested.
2. **Ordering preserves canonical reverse lookups**: Reverse lookups (`gethostbyaddr`/`getnameinfo`) return the **first** hostname listed on the line in `/etc/hosts`. By listing the bare name first (`example.com` followed by `example.com.`), reverse lookups continue returning the canonical bare name without a trailing dot, avoiding breaking changes to existing reverse resolution behavior.
3. **No runtime fallback needed**: In kubelet, `hostsEntriesFromHostAliases` and `managedHostsFileContent` return `[]byte` without an error return path. The only failures in `ensureHostsFile` are filesystem write or permissions errors (`os.WriteFile`, `Chmod`), which fail identically regardless of content formatting. There is no condition where a multi-line format would fail and a fallback would succeed, so no separate fallback path is required.

#### Behavior reference

Because the kubelet writes hostnames verbatim, which lookups resolve is decided by what the user lists. To make an alias resolve both with and without the trailing dot, list both names, bare name first.

Measured on glibc 2.31 and 2.41 and musl 1.1.24 and 1.2.6 for forward lookups, and on glibc 2.41 and musl 1.2.6 for reverse lookups. Forward lookups were also measured from Node.js 22 (`dns.lookup`) and OpenJDK 21 (`InetAddress`) on glibc and musl, with identical results (scripts and raw output: https://github.com/MU5A/hostaliases-fqdn-evidence):

| `hostnames` in the Pod | Line in `/etc/hosts` | Lookup `name` | Lookup `name.` | Reverse lookup returns |
|---|---|---|---|---|
| `["name"]` (today) | `IP name` | resolves | does not resolve | `name` |
| `["name."]` | `IP name.` | does not resolve | resolves | `name.` |
| `["name", "name."]` | `IP name name.` | resolves | resolves | `name` |
| `["name.", "name"]` | `IP name. name` | resolves | resolves | `name.` |

The second row is the case this KEP newly allows: it satisfies absolute lookups only. Listing both names, bare name first (third row), satisfies both forms without changing the name that reverse lookups return.

#### Discrepancy with `man 5 hosts`

The Linux manual page `man 5 hosts` specifies that hostnames "must begin with an alphabetic character and end with an alphanumeric character," which technically excludes trailing dots. This specification, however, does not reflect the actual behavior of DNS resolution systems in modern operating systems and container environments. Understanding this gap is critical to implementing correct FQDN support in Kubernetes.

The behavior of system resolvers is rooted in DNS standards rather than the `/etc/hosts` specification. RFC 1034 (Domain Names - Concepts and Facilities) and RFC 2181 (DNS Clarifications and Extensions) establish that trailing dots denote fully-qualified (absolute) domain names in DNS. RFC 1034 explicitly states: "a complete domain name ends with the root label, this leads to a printed form which ends in a dot." When the DNS protocol was integrated into system resolution stacks, the convention was preserved: a trailing dot signals that a name should be treated as absolute and not subjected to search path expansion.

When a workload running in a Pod performs an FQDN lookup with a trailing dot (such as querying `example.com.`), the system resolver performs an exact literal string match against `/etc/hosts` entries. This behavior is consistent across all major system resolver implementations including glibc (used by Linux), musl (used in Alpine and other lightweight distributions), BSD libc (FreeBSD, OpenBSD, NetBSD), and the native Windows resolver. It is also universal in language-level resolver libraries: Python's socket module, Java's InetAddress, Node.js's dns module, and Rust's std::net all delegate to the system resolver and respect this convention.

The exact literal matching behavior is not explicitly documented in `man 5 hosts`, but it is a necessary consequence of following RFC 1034. Most resolver implementations do not require the trailing dot to be present to match a query without a trailing dot (they apply search domains in those cases), but they do require it to be present when the query includes a trailing dot. This asymmetry is not a bug—it is the correct implementation of DNS semantics where trailing dots carry semantic meaning.

Kubernetes must preserve trailing dots in `/etc/hosts` to support absolute FQDN queries correctly. While Go's pure net resolver is lenient and will strip the dot internally, Kubernetes cannot assume that all Pods run pure Go applications. The ecosystem includes Python services, Java applications, Node.js servers, and countless other runtimes that rely on system resolvers. To ensure FQDN overrides work correctly across this diversity of workloads, the trailing dot must be written to `/etc/hosts` verbatim. Therefore, despite the strict wording in the `man 5 hosts` specification, preserving the trailing dot is required for correct DNS behavior.

### Interaction with existing HostAliases semantics

The existing behavior of `HostAliases` is unchanged for hostnames without a trailing dot. Reserved names (e.g. `localhost`, pod hostname, node name) are already handled by the existing HostAliases logic — the kubelet writes whatever hostnames are in the field into `/etc/hosts` without filtering. This KEP does not add any special filtering: a trailing dot simply passes through the same code path.

Users cannot override the pod's own hostname or localhost via HostAliases in any meaningful way beyond what `/etc/hosts` already allows. This is unchanged — the trailing dot does not introduce any new ability to break local resolution beyond what already exists with non-FQDN entries.

### Feature gate

The feature gate `HostAliasesAllowFQDN` controls whether trailing dots are permitted.

On **create** (`ValidatePod`), when the feature gate is disabled:
```go
if !utilfeature.DefaultFeatureGate.Enabled(features.HostAliasesAllowFQDN) {
    for i, hostAlias := range pod.Spec.HostAliases {
        for j, hostname := range hostAlias.Hostnames {
            if strings.HasSuffix(hostname, ".") && hostname != "." {
                allErrs = append(allErrs, field.Forbidden(
                    field.NewPath("spec", "hostAliases").Index(i).Child("hostnames").Index(j),
                    "trailing dot requires feature gate HostAliasesAllowFQDN",
                ))
            }
        }
    }
}
```

On **update** (`ValidatePodUpdate`), validation uses field ratcheting by comparing against `oldPod.Spec.HostAliases`:
```go
if !utilfeature.DefaultFeatureGate.Enabled(features.HostAliasesAllowFQDN) {
    allErrs = append(allErrs, validateHostAliasesGateOnUpdate(pod.Spec.HostAliases, oldPod.Spec.HostAliases, field.NewPath("spec", "hostAliases"))...)
}
```

**Gate enabled**: trailing dots are allowed on create and update.

**Gate disabled**: trailing dots are rejected on create. On update, existing trailing-dot entries already present in `oldPod.Spec.HostAliases` are permitted, allowing metadata updates (e.g., label changes) and non-HostAliases spec updates to succeed even after the feature gate is disabled.

### Test Plan

##### Unit tests

| Input | Gate state | Operation | Expected |
|---|---|---|---|
| `"example.com."` | enabled | create | accepted |
| `"example.com."` | disabled | create | rejected |
| `"example.com"` | either | create/update | accepted (unchanged) |
| `"."` | either | create/update | rejected (bare dot) |
| `"example.com.."` | either | create/update | rejected (multi-dot) |
| `".."` | either | create/update | rejected |
| `"localhost."` | enabled | create | accepted |
| `"my-server."` | enabled | create | accepted (single-label FQDN) |
| `"example.com."` (in `oldPod`) | disabled | update (unchanged) | accepted (ratcheted) |
| `"example.com."` (new) | disabled | update (added) | rejected |

##### Integration tests

- create pod with trailing dot — accepted with gate enabled, rejected with gate disabled.
- update pod with existing trailing dot (created with gate on) after gate is disabled — accepted (ratcheting against `oldPod.Spec.HostAliases`).
- update pod adding a new trailing dot entry after gate is disabled — rejected.

##### e2e tests

- create pod with trailing dot in hostAliases alongside bare name (`example.com`, `example.com.`).
- verify `/etc/hosts` contains single line: `IP\texample.com\texample.com.`.
- verify FQDN queries (with trailing dot) resolve correctly.
- verify relative queries (without trailing dot) resolve correctly.
- verify reverse lookups (`gethostbyaddr`/`getnameinfo`) return canonical `example.com`.
- test with diverse workload types and runtimes: glibc, musl, pure Go.

### Graduation Criteria

#### Alpha

- Feature gate `HostAliasesAllowFQDN`, disabled by default.
- Unit and integration tests.

#### Beta

- Gate enabled by default.
- e2e tests.

#### GA

- Feature gate removed (locked to true).

### Upgrade / Downgrade Strategy

`ValidateHostAliases` performs `strings.TrimSuffix(hostname, ".")` before DNS validation irrespective of the gate. This means a trailing dot never causes a DNS validation error — it only fails the explicit gate check.

When the gate is disabled after being previously enabled, `ValidatePodUpdate` compares `pod.Spec.HostAliases` against `oldPod.Spec.HostAliases` so that existing stored objects with trailing dots remain updatable (e.g. for label, annotation, or container status updates).

Downgrade safety: if a cluster is downgraded to a release without this KEP code, pods with trailing dots cannot be updated through the older API server. The risk is limited to pods that explicitly use the feature.

### Version Skew Strategy

Validation is confined to the API server. The kubelet writes hostnames verbatim into `/etc/hosts` without validation. No version skew concerns.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Name: HostAliasesAllowFQDN
  - Components: kube-apiserver

###### Does enabling the feature change any default behavior?

No. Opt-in validation relaxation.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Existing objects with trailing dots remain updatable via field ratcheting against `oldPod.Spec.HostAliases`.

###### What happens if we reenable the feature if it was previously rolled back?

Relaxed validation becomes available again.

###### Are there any tests for feature enablement/disablement?

Yes — unit and integration tests cover enabling, disabling, and re-enabling, including update ratcheting.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

No impact on running workloads. Rollout failure limited to unrecognized feature gate name (standard behavior).

###### What specific metrics should inform a rollback?

`apiserver_request_total{code=422, resource=pods, verb=POST}`

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Manual testing planned for alpha.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

```sh
kubectl get pods -A -o json | jq '.items[] |
  select(.spec.hostAliases != null) |
  select(.spec.hostAliases[].hostnames[] | endswith(".")) |
  "\(.metadata.namespace)/\(.metadata.name)"'
```

###### How can someone using this feature know that it is working for their instance?

Pod creation succeeds and `/etc/hosts` contains the specified entries verbatim:
```
10.10.10.10    example.com    example.com.
```
Forward resolution of both `example.com` and `example.com.` succeeds within the container, and reverse lookup returns `example.com`.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A. Validation change, no runtime impact.

### Dependencies

No dependencies.

### Scalability

No new API calls, types, or resource increases. One additional byte per hostname.

### Troubleshooting

N/A. Validation change within the API server.

## Implementation History

- 2026-05-14: Initial KEP draft
- 2026-05-17: Maintainer feedback on `/etc/hosts` documentation status and use case prevalence
- 2026-05-18: Major revision with RFC citations and comprehensive resolver analysis
- 2026-10-02: Incorporated test evidence from containerized libc/Go benchmarks (MU5A); documented single-line, bare-name-first ordering (`IP name name.`) for entries that list both forms, to preserve canonical reverse lookups (the kubelet is unchanged); removed obsolete fallback path; corrected update ratcheting to compare against `oldPod.Spec.HostAliases`

## Drawbacks

Increases the valid value surface for `HostAliases` hostnames. Trailing dots are a standard DNS convention and valid in `/etc/hosts` — minimal risk.

## Alternatives

**No feature gate:** This approach would be simpler but would bypass the controlled rollout mechanism and prevent operators from gradually adopting the feature or managing the blast radius of validation changes.

**Strip trailing dot silently:** While this would allow the feature to pass validation, it would prevent users from observing the trailing dot in the actual `/etc/hosts` entries written by the kubelet, defeating the purpose of the feature. FQDN queries would fail to match because the dot would be missing from the file.

**New `fqdn` field in `HostAlias`:** This would introduce unnecessary complexity to the API. The trailing dot is already a standard DNS convention for marking absolute names, and relaxing validation is simpler than adding new fields to the HostAlias type.
