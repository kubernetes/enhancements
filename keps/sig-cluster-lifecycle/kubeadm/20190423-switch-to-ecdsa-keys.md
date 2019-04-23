---
title: Switching kubeadm to generate ECDSA keys
authors:
  - "@rojkov"
owning-sig: sig-cluster-lifecycle
participating-sigs:
  - TBD
reviewers:
  - "@rosti"
  - "@detiber"
approvers:
  - "@fabriziopandini"
  - "@neolit123"
  - "@timothysc"
editor: TBD
creation-date: 2019-04-23
last-updated: 2019-04-23
status: provisional
---

# Switching kubeadm to generate ECDSA keys

## Table of Contents

  * [Release Signoff Checklist](#release-signoff-checklist)
  * [Summary](#summary)
  * [Motivation](#motivation)
	 * [Goals](#goals)
  * [Proposal](#proposal)
	 * [Implementation Details](#implementation-details)

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


## Summary

To unify the type of certificates used across Kubernetes make kubeadm generate
ECDSA keys and certificates instead of RSA ones during cluster initialization
and key renewal.

## Motivation

As explained in [OpenSSL wiki][] the primary advantage of using Elliptic
Curve based cryptography is reduced key size and hence speed.

In addition to that kubelet already generates self-signed server certificates
of the ECDSA type and uses ECDSA client certificates when communicating with
API servers. It would reduce complexity if the same defaults were used
across Kubernetes components.

[OpenSSL wiki]: https://wiki.openssl.org/index.php/Elliptic_Curve_Cryptography

### Goals

- unconditionally generate ECDSA keys upon cluster initialization.

## Proposal

### Implementation Details

Since people feel more comfortable when there is a transition period it
makes sense to introduce a configuration option first for the type of
keys used in new clusters or the clusters undergoing upgrades.

The selected key type should be defined in `InitConfiguration` or
optionally overridden with a command line option applicable to
the `init` command and to the `certs` related subcommands.

The function `pkiutil.NewPrivateKey()` should accept `keyType`
parameter of two possible values `ECDSA` and `RSA`. The actual value
is passed from `InitConfiguration` by callers. Currently
the function unconditionally generates RSA keys only.

The default value for the option is `RSA`.

It is OK to have certificates of mixed types in the same certificate chain thus
after two or three releases the default value can be switched to `ECDSA`.

After the transition period is over the option can be removed completely
and all new keys are going to be of the ECDSA type.
