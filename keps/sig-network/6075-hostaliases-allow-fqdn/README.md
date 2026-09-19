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
    - [/etc/hosts Generation with Graceful Degradation](#etchosts-generation-with-graceful-degradation)
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

Kubernetes `HostAliases` currently validates hostnames using `ValidateDNS1123Subdomain`, which rejects trailing dots. This restriction prevents users from pinning absolute FQDN names to specific IP addresses within their Pods. Applications that perform absolute FQDN lookups (common in distributed systems where exact name resolution is critical) cannot use `HostAliases` to override those names, even though the system resolver implementations inside containers fully support this use case.

The practical consequence is that users who need FQDN overrides must either accept incorrect DNS behavior (by using relative names and hoping search domains resolve correctly), or use privileged init containers to manually rewrite `/etc/hosts` to include the trailing dots. Neither approach is satisfactory: the former breaks applications designed to use absolute names, and the latter requires elevated privileges and circumvents Kubernetes security controls.

By allowing trailing dots in `HostAliases` hostnames, Kubernetes can support the full spectrum of DNS resolution patterns that applications require, making the feature more complete and usable for real-world scenarios.

### Goals

- Allow trailing dots in `HostAliases` hostnames to denote fully-qualified domain names (FQDNs), following RFC 1034 semantics.
- Preserve trailing dots in `/etc/hosts` entries so that absolute FQDN queries match correctly in all system resolver implementations.
- Support workloads that perform absolute FQDN lookups without requiring privileged init containers or manual `/etc/hosts` manipulation.

### Non-Goals

- Change any other DNS validation or behavior.
- Add new API fields or types.

## Proposal

Modify `ValidateHostAliases` to unconditionally strip a trailing dot before calling `ValidateDNS1123Subdomain`. Add a gate check in `ValidatePod` and `ValidatePodUpdate` to reject trailing dots when the feature gate is disabled.

The validation itself always succeeds for valid subdomains regardless of gate state — the gate only controls whether trailing dots are permitted at all.

### User Stories

#### Story 1

A pod runs an application that resolves `example.com.` (with trailing dot, as an FQDN). The user adds a HostAlias to pin that name to a specific IP:

```yaml
spec:
  hostAliases:
  - ip: "10.10.10.10"
    hostnames:
    - "example.com."
```

The trailing dot makes `gethostbyname` skip the search list. Without this KEP the pod is rejected by validation.

#### Story 2

A pod runs heterogeneous workloads: some applications query `example.com.` (absolute, FQDN style) while others query `example.com` (relative, search-list style). Rather than create separate HostAlias entries, the user can provide both names in a single entry:

```yaml
spec:
  hostAliases:
  - ip: "10.10.10.10"
    hostnames:
    - "example.com."
    - "example.com"
```

This generates a single `/etc/hosts` entry with both names, allowing both query styles to match: applications using the absolute form match the FQDN entry, and applications using the relative form match the second entry (which may then be expanded by the search list if needed). This avoids duplicate IP entries and is the DNS best practice for supporting both absolute and relative resolution.

### Risks and Mitigations

**Concern: Use case is rare and undocumented**

This concern underestimates both the prevalence of the use case and the documentation status. RFC 1034 and RFC 2181 explicitly document trailing dots in domain names as the standard representation of absolute FQDNs. Every major operating system, system resolver (glibc, musl, BSD, Windows), and language runtime (Go, Python, Java, Node.js, Rust, Ruby) implements this behavior consistently. The trailing dot convention is not anecdotal—it is normative DNS semantics. The use case is not rare: any workload that makes absolute FQDN queries (including service meshes, distributed tracing systems, and any application that explicitly specifies FQDN URLs) requires this functionality. The current validation barrier simply prevents legitimate use cases from being expressed in Kubernetes.

**Concern: Downgrade compatibility**

Downgrade risk is real but limited: pods with trailing dot hostnames cannot be updated through an older API server that does not strip trailing dots. This risk is mitigated by automatic ratcheting—`ValidateHostAliases` strips trailing dots before DNS validation regardless of feature gate state, so stored pods with trailing dots remain updatable after a downgrade. The risk is therefore confined to clusters that explicitly enable the feature and then downgrade: those pods cannot be modified until the cluster is upgraded again. This is acceptable downgrade behavior because the feature is gated and opt-in.

**Concern: Behavioral inconsistency across resolver implementations**

All major resolver implementations support trailing dots and follow RFC 1034 semantics. The behavior is uniform, not inconsistent. The `man 5 hosts` specification is the outlier—it does not reflect how modern system resolvers actually parse the file. Kubernetes should follow the DNS standard and the behavior of actual resolver implementations, not an outdated specification that contradicts current practice.

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

#### /etc/hosts Generation with Graceful Degradation

The kubelet writes hostnames from `HostAliases` directly into `/etc/hosts` via `hostsEntriesFromHostAliases` using a graceful degradation approach:

**Primary Approach: Separate Lines for Parser Compatibility**

For hostnames containing a trailing dot, generate **two separate lines** in `/etc/hosts` to ensure compatibility with legacy parsers:

```
10.10.10.10    example.com.
10.10.10.10    example.com
```

Preserves FQDN semantics for exact literal matching (RFC 1034), supports both relative and absolute DNS queries, maintains compatibility with legacy `/etc/hosts` parsers.

**Fallback Approach: Single Line with Multiple Aliases**

If generating separate lines fails or is not supported, fall back to a single line with multiple aliases:

```
10.10.10.10    example.com.    example.com
```

**Implementation Logic**

```go
func hostsEntriesFromHostAliases(hostAliases []v1.HostAlias) []byte {
    var buffer bytes.Buffer
    buffer.WriteString("\n")
    buffer.WriteString("# Entries added by HostAliases.\n")
    for _, hostAlias := range hostAliases {
        for _, hostname := range hostAlias.Hostnames {
            if strings.HasSuffix(hostname, ".") && hostname != "." {
                buffer.WriteString(fmt.Sprintf("%s\t%s\n", hostAlias.IP, hostname))
                relativeHostname := strings.TrimSuffix(hostname, ".")
                buffer.WriteString(fmt.Sprintf("%s\t%s\n", hostAlias.IP, relativeHostname))
            } else {
                buffer.WriteString(fmt.Sprintf("%s\t%s\n", hostAlias.IP, hostname))
            }
        }
    }
    return buffer.Bytes()
}
```

If the above approach encounters write errors or validation issues, fall back to the legacy approach:

```go
func hostsEntriesFromHostAliasesLegacy(hostAliases []v1.HostAlias) []byte {
    var buffer bytes.Buffer
    buffer.WriteString("\n")
    buffer.WriteString("# Entries added by HostAliases.\n")
    for _, hostAlias := range hostAliases {
        buffer.WriteString(fmt.Sprintf("%s\t%s\n", hostAlias.IP, strings.Join(hostAlias.Hostnames, "\t")))
    }
    return buffer.Bytes()
}
```

If the trailing dot were stripped, the entry in `/etc/hosts` would lack it and FQDN resolution would not match, as system resolvers perform exact literal string matching. The two-line approach ensures that applications querying both forms (`example.com.` and `example.com`) receive correct responses.

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

The feature gate check is added at the caller level in `ValidatePod` and `ValidatePodUpdate`:

```go
for i, hostAlias := range pod.Spec.HostAliases {
    for j, hostname := range hostAlias.Hostnames {
        if strings.HasSuffix(hostname, ".") && hostname != "." {
            allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "hostAliases").Index(i).Child("hostnames").Index(j),
                "trailing dot requires feature gate HostAliasesAllowFQDN"))
        }
    }
}
```

**Gate enabled**: trailing dots are allowed on create and update.

**Gate disabled**: trailing dots are rejected on create. Updates are not additionally restricted because `ValidateHostAliases` always accepts the trimmed value — ratcheting is automatic: existing hostnames with trailing dots remain updatable after the gate is disabled.

### Test Plan

##### Unit tests

| Input | Gate state | Expected |
|---|---|---|
| `"example.com."` | enabled | accepted |
| `"example.com."` | disabled | rejected |
| `"example.com"` | either | accepted (unchanged) |
| `"."` | either | rejected (bare dot) |
| `"example.com.."` | either | rejected (multi-dot) |
| `".."` | either | rejected |
| `"localhost."` | enabled | accepted |
| `"my-server."` | enabled | accepted (single-label FQDN) |

##### Integration tests

- create pod with trailing dot — accepted with gate enabled, rejected with gate disabled
- update pod that has trailing dot (created with gate on) after gate is disabled — accepted (ratcheting)

##### e2e tests

**Primary approach:**
- create pod with trailing dot in hostAliases
- verify `/etc/hosts` contains two separate lines: `IP    example.com.` and `IP    example.com`
- verify FQDN queries (with trailing dot) resolve correctly
- verify relative queries (without trailing dot) resolve correctly
- test with diverse workload types: Python, Java, Node.js, Go

**Fallback approach:**
- simulate fallback scenario (e.g., write error on separate lines)
- verify fallback generates single line with multiple aliases: `IP    example.com.    example.com`
- verify resolution still works (though may be degraded on legacy parsers)
- document fallback behavior in logs/events

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

`ValidateHostAliases` performs `strings.TrimSuffix(hostname, ".")` before DNS validation irrespective of the gate. This means a trailing dot never causes a DNS validation error — it only fails the explicit gate check. When the gate is disabled on update, the per-hostname gate check is skipped entirely, so stored objects with trailing dots remain updatable.

Downgrade safety: if a cluster is downgraded to a release without this change, pods with trailing dots cannot be updated. The risk is limited to pods that explicitly use the feature.

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

Yes. Existing objects with trailing dots remain updatable via automatic ratcheting (the DNS validation always strips the trailing dot).

###### What happens if we reenable the feature if it was previously rolled back?

Relaxed validation becomes available again.

###### Are there any tests for feature enablement/disablement?

Yes — unit and integration tests cover enabling, disabling, and re-enabling.

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

Pod creation succeeds and `/etc/hosts` contains:

**Primary approach:** two separate lines for each FQDN hostname with `IP    example.com.` and `IP    example.com`, with Kubelet logs showing "Generated FQDN entries with separate lines for <namespace>/<pod>"

**Fallback approach:** single line with multiple aliases `IP    example.com.    example.com`, with Kubelet logs showing "Fallback: generated FQDN entries on single line for <namespace>/<pod>" (warning level)

Both approaches ensure FQDN resolution works, but the primary approach provides better compatibility with legacy `/etc/hosts` parsers.

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
- 2026-05-18: Major revision with RFC citations, comprehensive resolver analysis, and design update for graceful degradation in `/etc/hosts` generation

## Drawbacks

Increases the valid value surface for `HostAliases` hostnames. Trailing dots are a standard DNS convention and valid in `/etc/hosts` — minimal risk.

## Alternatives

**No feature gate:** This approach would be simpler but would bypass the controlled rollout mechanism and prevent operators from gradually adopting the feature or managing the blast radius of validation changes.

**Strip trailing dot silently:** While this would allow the feature to pass validation, it would prevent users from observing the trailing dot in the actual `/etc/hosts` entries written by the kubelet, defeating the purpose of the feature. FQDN queries would fail to match because the dot would be missing from the file.

**New `fqdn` field in `HostAlias`:** This would introduce unnecessary complexity to the API. The trailing dot is already a standard DNS convention for marking absolute names, and relaxing validation is simpler than adding new fields to the HostAlias type.
