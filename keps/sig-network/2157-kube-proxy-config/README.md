# KEP-2157: Kube proxy args reconciliation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
- [Goals](#goals)
- [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] - [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] - [ ] (R) Design details are appropriately documented
- [ ] - [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] - [ ] (R) Graduation criteria is in place
- [ ] - [ ] (R) Production readiness review completed
- [ ] - [ ] Production readiness review approved
- [ ] - [ ] "Implementation History" section is up-to-date for milestone
- [ ] - [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] - [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes
[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The current kube-proxy flags are largely ignored by default, but they **are** validated. This is self-consistent
but misleading. As an example, you can't send `--blah`, but you can say `--proxy-mode=thisiswrong`.
The latter is essentially a no-op (it should break the kube-proxy and "fail closed" as Tim might say :).

We thus propose a few simple consistency amendments to the input handling of the kube-proxy which will
be self-consistent and prevent "no-op" inputs that are silently ignored, and in general, suggest transparency
in terms of what inputs to kube-proxy are being utilized. The KEP intends to enhance the visibility of the flags already deprecated and provides consistency between flags and
the existent component configuration API in kube-proxy as stated on [KEP 783](https://github.com/kubernetes/enhancements/tree/master/keps/sig-cluster-lifecycle/wgs/783-component-base).

## Motivation

The latter command ALSO ignores correct flags `--proxy-mode=iptables`, as you might expect,
meaning that users might try to configure something and then spend several minutes trying to figure out why
their config wasn't exercised. Since there are several kube-proxy options that might be overridden (there are ~50 options configured in a typical
config-map for the kube proxy), this is obviously a compounding problem.

The lack of flag precedence over the configuration API, option defaulting, and validation with proper round trip
test contributes to increasing the existent or newly added flags' inconsistency. These statements are a requirement for the
future configuration API graduation covered on [KEP 784](https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/784-kube-proxy-component-config).

## Goals

We thus suggest the following modifications to kube-proxy input handling.

1) For some options (such as the `mode`) the configuration is printed with a custom logging. We suggest that all options be printed after
   log level verbosity 5, not only in the flags as it happens nowadays but including the configuration structure. This can be used to
   debug the settings used by the users, currently options like `--proxy-mode` are not processed at all via command line,
   EVEN IF there is nothing in the configuration file.

```
I0419 09:29:57.058543  834230 server.go:103] "KubeProxyConfiguration" configuration={TypeMeta:{Kind: APIVersion:} FeatureGates:map[]
BindAddress:0.0.0.0 HealthzBindAddress:0.0.0.0:10256 MetricsBindAddress:127.0.0.1:10249 BindAddressHardFail:false EnableProfiling:false
ClusterCIDR: HostnameOverride: ClientConnection:{Kubeconfig: AcceptContentTypes: ContentType:application/vnd.kubernetes.protobuf QPS:5 Burst:10}
IPTables:{MasqueradeBit:0xc0002bffa0 MasqueradeAll:false SyncPeriod:{Duration:30s} MinSyncPeriod:{Duration:1s}} IPVS:{SyncPeriod:{Duration:30s}
 MinSyncPeriod:{Duration:0s} Scheduler: ExcludeCIDRs:[] StrictARP:false TCPTimeout:{Duration:0s} TCPFinTimeout:{Duration:0s}
 UDPTimeout:{Duration:0s}} OOMScoreAdj:0xc0002bffa4 Mode: PortRange: UDPIdleTimeout:{Duration:250ms} Conntrack:{MaxPerCore:0xc0002bffa8
 Min:0xc0002bffac TCPEstablishedTimeout:&Duration{Duration:24h0m0s,} TCPCloseWaitTimeout:&Duration{Duration:1h0m0s,}}
 ConfigSyncPeriod:{Duration:15m0s} NodePortAddresses:[] Winkernel:{NetworkName: SourceVip: EnableDSR:false} ShowHiddenMetricsForVersion: DetectLocalMode:}
```

2) Wrong command line option must be explicitly rejected (even if only ignored before). This involves a proper field validation
   with closed failing.

3) Flags ALWAYS must take precedence over configuration options.

## Non-Goals

- Add new command line options for the kube-proxy or fix logical issues in the way kube-proxy manages input for domain specific inputs.
- Override all config map options in flags.
- Graduate the kube-proxy configuration API to v1beta1 or GA, besides paving this path.
- Implement platform or instance (local) specific configuration over the already existent API.
- Any kind of dynamic-reload, of any sort.  It was decided in a recent sig-network call that the Kube-proxy should never do this.

## Proposal

We propose to make the codebase self-consistent WRT the goals above, which largely is self-explanatory:

- We will print out all inputs from the kube-proxy config map
- We will add a straightforward and obvious implementation of the command line overriding the configuration.
- We will document the limitations/usage of the kube-proxy command line switches appropriately.

### User Stories

- As a windows user, I want the usage of kube-proxy flags to be self-consistent with usage of other flags
  (and i want fine grained OS aware flag validation where necessary).
- As an administrator rolling out a new configuration for kube-proxy, I want the proxy to fail if i send
  a configuration that is invalid so that i don't roll my workloads over to a misconfigured cluster.
- As a kube-proxy developer, I want to be able to override configMap parameters with flags while testing a feature.
- As an administrator that automates infrastructure by using flags, I want the flags to work in standard manner
  (fail if invalid, or otherwise, be utilized) so I can use common configuration idioms (like a skeleton yaml
  to inherit and customize) in my operations workflow.

### Risks and Mitigations

 - People who are running clusters with broken flags, upon restarting kube-proxy, will have failed kube-proxy processes.
   This is likely a benefit in almost all cases, since those flags were being ignored, likely unbeknownst to the users.
 
### Design Details

The overall flow of the configuration of kube proxy happens on the serverside, by parsing the configmap
provided to it as the `--config` option from the command line.

Most of all the command line arguments in kube proxy are ignored, unless one is trying to write an example
config file out to disk. Since this option was added several years ago and is now deprecated,
we can thus assume all or many of the kube-proxy options can be deleted.

Another case is the `--hostname-override` flag, which post-process the command line option and overrides the hostname
in the `processHostNameOverride` method. For this scenario the validation must be moved to the API machinery,
letting only value treatment or cleaning to this function. This pattern can be applied until the configuration
API graduation, and the final deletion of all commandline options from kube-proxy, avoiding possible ambiguity.

Since dynamic vs flag input is a little tricky to visualize, we attempt a sketch summarizing
the order of kube proxy config input, and startup, below.

On the far right, we describe the chronological place where command line substitution should take place:
AFTER completing the configuration of all file parsing but BEFORE validating the configuration. This would ensure:

- configMap parameters from the `--config` YAML file are read first
- flags are read second
- flags are applied in a way that overrides any corresponding configMap parameters

 ```
    +-------------+                  +-------------+
    |    kubeadm  +----------------> |  configmap  |
    +-------------+                  +----+--------+
                                          |
                                          |
                       ConfigMap as file  v
   +----------------+     Mounted at    +----+---------+
   |                |     --config=...  |              |
   |     kube-proxy +-------------------+  apiserver   |
   +----------------+                   +--------------+
   |            +----------------------------+
   |            | +-------------+            |
   +----------->+ | Complete    |            |     ******************************
   |            | |             |            |      We can add a "override flags"
   |            | +-------------+            |      option here.
   |            |                  <--------------+ In a future graduation, we can
   |            | +-------------+            |      delete all flags.
   +----------->+ | Validate    |            |      ******************************
   |            | |             +------+     |
   |            | +-------------+      |     |
   |            |                      |     |
   |            | +-------------+      |     |      +-----------------------------------------+
   |            | |             |      |     |      |  Call Validation                        |
   +----------->+ | Run         |      +----------->+  in api/config/validation/Validation.go |
                | |             |            |      |                                         |
                | +-------------+            |      +-----------------------------------------+
                |  NewProxyCommand()         |
                |  cmd/proxy/app/server.go   |
                +----------------------------+
```

### Test Plan

Kube-proxy configuration APIs are typically tested similarly to API server APIs, for
example by testing round-trip conversions. Kube-proxy configuration should be tested
and the test suite an outcome of this KEP.

### Graduation Criteria

The implementation of this KEP can result in another version of the alpha configuration
library (`v1alpha2`, `v1alpha3`, eg.), if some big breaking change happens and is going to be
analyzed on the SIG-network consensus. The further graduation for `v1beta1` or `GA` is
outside this KEP scope.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
    - [ ] Feature gate (also fill in values in `kep.yaml`)
        - Feature gate name: NONE
        - Components depending on the feature gate: kube-proxy
        - Will enabling / disabling the feature require downtime or reprovisioning
          of a node? No

* **Does enabling the feature change any default behavior?**
  Yes, using the latest API version should result in the correct default behaviour.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  No, after refactoring the undo is not easily possible, so users must be aware of
  the changes in the version changelog, see Risks and Mitigation.

* **Are there any tests for feature enablement/disablement?**
  E2E tests should cover the final marshalled configuration object.
  Round-trips cover unit-tests of the API.

## Implementation History

- Initial Draft: 2020-11-19
- Draft Merged: ?
- PR merged for first component using this approach: ?
- First release with this approach available in a beta component: ?
