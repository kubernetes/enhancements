# KEP-NNNN: KubeProxyConfiguration v1alpha2

## Summary

The kube-proxy command-line arguments and configuration file format
have gotten messy and need to be cleaned up and rationalized.

Additionally, creating a new not-entirely-backward-compatible
configuration format gives us the opportunity to change some defaults
(for users using the new config format), pushing users away from
insecure or otherwise outdated configuration and helping to pave the
way for the upcoming [nftables-based kube-proxy backend].

[nftables-based kube-proxy backend]: https://github.com/kubernetes/enhancements/issues/3866

## Motivation

### Goals

- Create a `v1alpha2` KubeProxyConfiguration format.

    - Clean up redundant, inconsistent, and badly-organized config
      options, relative to `v1alpha1`. (Discussed some in [kubernetes
      #117909].)

    - Fix up configuration options that were never fully reconsidered
      for dual-stack. (e.g., `--metrics-bind-address` only allows
      specifying a single IP.)

    - Change the default values of some options in the new format to
      be more secure and forward-compatible, e.g. so that people have
      to explicitly set `localhostNodePorts: true` if they want the
      legacy behavior.

    - Add new configuration options to disable `iptables` proxy
      behaviors that will not be supported by the `nftables` proxy,
      which will also default to the forward-compatible values. (e.g.,
      an option to disable the creation of `-j ACCEPT` rules).

- When kube-proxy is invoked with `--config` pointing to a new-format
  config file:

    - Forbid most other command-line arguments, and ensure reasonable
      overriding semantics for the ones that are still allowed. (e.g.,
      see [kubernetes #108737])

    - Error out at startup if the configuration is incomplete or
      incorrect (e.g., if `detectLocalMode` is set to `ClusterCIDR`
      but `clusterCIDR` is not set, or if `localhostNodePorts` is
      `true` but `mode` is `ipvs`).

- Extend `kube-proxy --write-config-to` to support the new format, to
  allow users to easily generate a new config compatible with their
  existing one.

- Add metrics (or some other sort of indication) to kube-proxy
  `iptables` mode to indicate when the "non-forward-compatible"
  options (like `localhostNodePorts`) appear to be unused, so we can
  encourage users to disable them.

[kubernetes #108737]: https://github.com/kubernetes/kubernetes/issues/108737
[kubernetes #117909]: https://github.com/kubernetes/kubernetes/issues/117909

### Non-Goals

- Changing any of the defaults for users who are using a `v1alpha1`
  config file or using only command-line arguments rather than a
  config file.

- Graduating the config format to `v1beta1` or `v1` when the KEP
  graduates to Beta/GA; we will need some experience with the new
  format before we are ready to do that.

## Design Details

### `v1alpha2` config format

The `v1alpha2` format will be _mostly_ the same as the `v1alpha1`
format...

#### General config sanity

See https://github.com/kubernetes/kubernetes/issues/117909.

Also, `bindAddress` is a terrible name and should be `nodeIPOverride`,
or possibly should not even still exist, since if your node actually
has the wrong IP, other things may fail. (Some discussion in
https://github.com/kubernetes/kubernetes/pull/119525.)

`metricsBindAddress` and `healthzBindAddress` only allow providing a
single IP on dual-stack hosts. It might make more sense to have
`metricsBindInterface` / `healthzBindInterface` so you can specify a
NIC to bind dual-stack on, rather than a single IP? (That is probably
also more useful for specifying in a central config file, since it's
likely most of your nodes have the same set of NICs. Alternatively, we
could allow specifying a CIDR rather than an IP, meaning "bind to
whatever IP you find on the node in this CIDR", like how
`nodePortAddresses` works... I'm sure this was discussed at one point.
Though that still has the problem that it would need to be extended
for dual-stack.) Also, splitting out a separate `metricsBindPort` /
`healthzBindPort` would simplify things both for "people who want to
change where the server is available but don't want to change the
port" *and* for "people who want to change the port but don't want to
change where the server is available".

#### Forward-compatibility with nftables

We should make `localhostNodePorts` be `false` by default in the new
config format.

We should make `nodePortAddresses: nil` mean "only accept NodePort
connections on the primary/secondary node IP", and force users to say
`nodePortAddresses: ["0.0.0.0/0", "::/0"]` if they want the legacy
behavior of doing NodePorts on all local IPs.

`nftables` mode can't do the equivalent of `iptables` and `ipvs`'s `-j
ACCEPT` rules to bypass bad local firewall rules, so we should make it
possible to disable that behavior in iptables/ipvs too so people can
make sure their config is compatible ahead of time.

I think those are the only incompatible behavioral changes proposed in
KEP-3866.

### Config vs command-line

See, eg, https://github.com/kubernetes/kubernetes/issues/108737

When using the new config format, we should error out if most other
command-line args are used. The only allowed flags would be the ones
that are clearly node-specific (eg, `--hostname-override`), so someone
could have a global config file plus a few per-node overrides on the
command-line.

We should have clear semantics for what happens when an option is
specified both in the config file and on the command line, and unit
tests to make sure we don't break things in the future.

### Startup Behavior

https://github.com/kubernetes/kubernetes/pull/119003 added some new
configuration error-checking at startup, but it doesn't error out in
some cases, for backward compatibility. When using a new config file,
we should error out on any invalid configuration.

Also, kube-proxy should error out if the user switches to `nftables`
mode without disabling all of the `iptables` mode features that
`nftables` won't support. (Maybe `nftables` mode requires using the
new config? Or maybe we just enforce the "all config must be correct"
rule for both new config users and nftables users.)

### New metrics

We can recognize when some of the non-future-compatible features are
being used, by parsing the `iptables-save` output when cleaning up
stale chains:

  - If the counters on all of the localhost nodeport SNAT rules are
    all 0, then localhost nodeports aren't in use.

  - Although we don't have existing rules that would explicitly catch
    this, we could adjust the NodePort rules in general to be able to
    detect if people are receiving NodePort connections on "random"
    node IPs or just the primary ones.

We can't easily detect whether the `-j ACCEPT` firewall-bypass rules
are needed or not, because the rules would get run whether or not they
were actually needed.

...
