---
title: Instance-Specific ComponentConfig
authors:
  - "@mtaufen"
owning-sig: sig-cluster-lifecycle
participating-sigs:
  - sig-cluster-lifecycle
  - sig-api-machinery
  - sig-architecture
  - wg-component-standard
reviewers:
  - sttts
  - stealthybox
  - liggitt
  - rosti
  - neolit123
  - obitech
approvers:
  - sttts
  - stealthybox
  - liggitt
editor: TBD
creation-date: 2020-01-08
last-updated: 2020-06-02
status: provisional
---

# Instance-Specific ComponentConfig

## Table of Contents

A table of contents is helpful for quickly jumping to sections of a KEP and for highlighting any additional information provided beyond the standard KEP template.

Ensure the TOC is wrapped with <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code> tags, and then generate with `hack/update-toc.sh`.

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Separate config object for instance-specific config](#separate-config-object-for-instance-specific-config)
  - [Why not just keep using flags for instance-specific parameters?](#why-not-just-keep-using-flags-for-instance-specific-parameters)
  - [Why not just auto-detect?](#why-not-just-auto-detect)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Kube-proxy Example](#kube-proxy-example)
  - [Kubelet Example](#kubelet-example)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

ComponentConfig is an ongoing effort to make Kubernetes-style config files the
preferred way to configure core Kubernetes components, instead of command-line
flags. An overview of and motivation for ComponentConfig, in general, can be
found in the [Versioned Component Configuration Files](https://docs.google.com/document/d/1FdaEJUEh091qf5B98HM6_8MS764iXrxxigNIdwHYW9c)
doc (access is available to all members of kubernetes-dev@googlegroups.com).

This KEP proposes a solution to a specific problem we have encountered while
implementing ComponentConfig. The vast majority of configuration parameter
values can be reused between concurrent runtime instances of a component, but
some values are required to be unique to each instance. To date, we have held
off migrating these parameters to existing ComponentConfig APIs, because we lack
a decision on how to consistently handle them.

This KEP proposes that components use a separate, instance-specific Kind,
to clearly identify and separate parameters that are instance-specific from
parameters that are sharable. It also proposes some mitigations in the event
that the initial categorization is incorrect, e.g. we discover an
instance-specific use-case for a sharable field.

The primary goal of this KEP is to make and codify a decision, so that we can
move forward with ComponentConfig implementation.

## Motivation

Most core Kubernetes components have many configuration parameters where the
value can, and usually should, be set the same for concurrent instances. For
example, QPS limits or authentication configuration is typically the same for
kubelet across many nodes in a cluster.

Some parameters, however, are unique to each instance of the component.
For example, assuming one kubelet per host, every kubelet that is configured
with the IP address of its host machine will be configured with a unique value.
We refer to these as "instance-specific" parameters.

Some core components are deployed via Pods in the cluster. When following the
ComponentConfig approach, these components typically use a ConfigMap volume to
deliver the config files. All instances of a component are typically configured
to point to a single ConfigMap. This makes configuration and deployment much
simpler, and less expensive, than having to create a separate ConfigMap for each
instance of the component.

Other components may not be deployed as Pods but may still have channels to
provide configuration via the cluster. kubelet, for example, has the Dynamic
kubelet Config feature, which enables the kubelet to read configuration from
a ConfigMap designated by its corresponding Node object. With kubelet, it is
also advantageous for "pools" of nodes to refer to the same ConfigMap.

Unfortunately, a shared ConfigMap is incompatible with instance-specific
parameters which, by definition, cannot be shared between instances. While we
have been able to migrate many shareable parameters to ComponentConfig APIs,
we have been blocked on the instance-specific piece. Lack of a decision on how
to handle these has led to uncertainty as the owners of other components begin
moving to ComponentConfig APIs. A clear specification on how to implement
instance-specific configuration will unblock and expedite our migration from
command-line flags to ComponentConfig.

See also issue [#61647](https://github.com/kubernetes/kubernetes/issues/61647).

### Goals

- Provide a clear specification on how to handle instance-specific in
  ComponentConfig when migrating to a new or adapting an existing
  ComponentConfig API.

### Non-Goals

- Provide a standard way to continue specifying configuration on the
  command-line.
- Redesign APIs/refactor the existing set of parameters. This is about providing
  a clear migration path for all classes of parameters.
- Eliminate instance-specific config parameters altogether. This is a special
  case of the previous item.
- Make changes to the Dynamic kubelet Config feature or address difficulties in
  using the kubelet's current ComponentConfig API with Dynamic kubelet Config.

## Proposal

Taking the Kubelet as an example, there are three categories of fields
(identified by @liggitt):
1. clearly instance-specific (e.g. `--node-ip`, `--provider-id`,
   `--hostname-override`)
2. possibly instance-specific (e.g. `--bind-address`, `--root-dir`)
3. highly unlikely to be instance-specific (e.g. cloud provider, node lease
   duration)

Today, the "sharable" configuration Kinds may contain (2) or (3); in Kubelet
we have been careful to keep (1) out of the configuration schema (as mentioned
above).

This KEP proposes that components use a separate, instance-specific Kind,
to clearly identify and separate parameters that are instance-specific from
parameters that are sharable. It also proposes that components have a way
to define the same field in both instance-specific and sharable Kinds, and
define a merge such that the instance-specific field overrides the sharable
field. This gives us some flexibility if we miscategorize fields, or discover
an instance-specific use-case for a sharable field.

### Separate Kind for instance-specific config

First, add a new instance-specific ComponentConfig Kind in each component's
config API. This gives us a clear structure to separate instance-specific and
sharable fields. This KEP proposes a new flag, `--instance-config`, for this
new Kind.

ComponentConfig files containing only sharable parameters can continue to be
shared via a single ConfigMap. Files containing only instance-specific
parameters can be provided to components via other means, such as the node
startup scripts or an init container that inserts values from the Pod's Downward
API.

### Case-by-case merge if fields are miscategorized

Second, allow instance-specific configs to override shared config when the same
field is specified in both.

Additional implementation details are included in the Design Details section.

### Kube-proxy Example

Today, a kube-proxy DaemonSet may be configured as follows to mix sharable and
instance-specific configuration parameters (using flags only). Note some fields
and arguments that would be set in production have been removed or modified
below to improve the clarity of the example.

```yaml
apiVersion: apps/v1
kind: DaemonSet
  name: kube-proxy
  namespace: kube-system
spec:
  template:
    spec:
      containers:
      - name: kube-proxy
        image: kube-proxy-amd64
        command:
        - kube-proxy
        - --proxy-mode=ipvs
        - --hostname-override=${NODE_NAME}
        env:
          - NODE_NAME
            valueFrom:
              fieldRef:
                fieldPath: spec.nodeName
```

Below demonstrates how the above may be modified when moving to a shared
ComponentConfig ConfigMap. Note that the instance-specific parameter is still
specified via a flag, because the ConfigMap is not appropriate for
instance-specific parameters.

```yaml
apiVersion: apps/v1
kind: DaemonSet
  name: kube-proxy
  namespace: kube-system
spec:
  template:
    spec:
      containers:
      - name: kube-proxy
        image: kube-proxy-amd64
        command:
        - kube-proxy
        - --config=/etc/config/kube-proxy.yaml
        - --hostname-override=${NODE_NAME}
        env:
        - NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        volumeMounts:
        - mountPath: /etc/config
          name: config
          readOnly: true
      volumes:
      - name: config
        configMap:
          name: kube-proxy-config-ghkm248tbc
```

Finally, when instance-specific configuration is also provided via the
ComponentConfig approach, using the separate file proposed above.

```yaml
apiVersion: apps/v1
kind: DaemonSet
  name: kube-proxy
  namespace: kube-system
spec:
  template:
    spec:
      initContainers:
      - name: instance-config
        image: something-with-bash
        command:
        - bash -c
        args:
        - cat << EOF > /etc/config/kube-proxy-instance.yaml
          kind: InstanceConfiguration
          apiVersion: kubeproxy.config.k8s.io/v1alpha1
          hostnameOverride: ${NODE_NAME}
          EOF
        env:
        - NODE_NAME
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        volumeMounts:
        - mountPath: /etc/config
          name: config
          readOnly: true
      containers:
      - name: kube-proxy
        image: kube-proxy-amd64
        command:
        - kube-proxy
        - --config=/etc/config/kube-proxy.yaml
        - --instance-config=/etc/config/kube-proxy-instance.yaml
        volumeMounts:
        - mountPath: /etc/config
          name: config
          readOnly: true
      volumes:
      - name: config
        configMap:
          name: kube-proxy-config-ghkm248tbc
```

Although the last example is more verbose, it provides the full benefits of
the ComponentConfig approach, resulting in a consistent surface and versioning
policy for the component's parameters.

Sometimes, kube-proxy is configured as a static Pod instead of a DaemonSet.
In that case, instance-specific configuration could be configured by the node
startup scripts and mounted via a host path volume.

### Kubelet Example

The Kubelet is not typically deployed as a Pod. Instead, startup scripts that
bootstrap the node write the Kubelet's initial configuration to disk at startup.
In some deployments, the Kubelet's configuration may later be modified by the
Dynamic Kubelet Config feature.

Today, the startup script may write a Kubelet command line that looks something
like `kubelet --config=/etc/config/kubelet.yaml --node-ip=${NODE_IP}`. Under
this proposal, it would instead write both config files, and a command line like
`kubelet --config=/etc/config/kubelet.yaml --instance-config=/etc/config/kubelet-instance.yaml`.

When the Dynamic Kubelet Config feature is enabled, the Kubelet only replaces
the values supplied to `--config` when consuming a remote configuration. Thus,
the instance-specific parameters would not interfere with sharable parameters
in this approach.

### Alternative: Define a general merge semantic for ComponentConfig

Prior discussions included the idea to define a general merge option instead,
which would have pushed the field categorization decision to the user by
allowing them to specify arbitrary patches on top of the config (for example,
with repeated invocations of a `--config` flag). This is the most flexible
solution, but also the most open-ended, and provides the least structural
guidance to users.

While this KEP recognizes the utility of components that can merge configuration
and is open to future development in this direction, it proposes that we focus
on a clear, near-term solution for this ComponentConfig use case, rather than
attempt to define and standardize a way of merging or generating config outside
of kube-apiserver, where many competing solutions, such as jsonnet, kustomize,
etc, already exist, and significant debate is likely required. The priority of
this KEP is to unblock ComponentConfig so that components can move to versioned
APIs. Improvements can be made in future API versions.


### Why not just keep using flags for instance-specific parameters?

Flags have various problems, outlined in the
[Versioned Component Configuration Files](https://docs.google.com/document/d/1FdaEJUEh091qf5B98HM6_8MS764iXrxxigNIdwHYW9c)
doc.

Leaving some parameters as flags while the rest are exposed via ComponentConfig
results in an inconsistent API surface, and an inconsistent API versioning
policy. Why fix only part of the problem when we can fix the whole thing?

### Why not just auto-detect?

One alternative that has been proposed in the past is to find a way to
auto-detect values of instance-specific parameters and simply eliminate them
from the configuration workflow. This is a great idea in theory, because it
solves the problem by removing work for users, but it may be difficult in
practice. For example, on a machine with multiple network cards, which IP should
the kubelet use?

Implementing the two-object approach does not prevent us from finding ways to
auto-detect instance specific config in the future. It is worth noting that the
_possibility_ of someday being able to auto-detect these has led to some
hesitation to make a firm decision today. We have delayed for too long already,
and a firm decision is needed to move forward.

### Risks and Mitigations

One risk is that fields get miscategorized. We mitigate this risk by providing a
way to define the field in both places, where instance-specific overrides
sharable, as described in the proposal and below design details.

For users deploying Pod templates, this solution can make the template more
verbose. In the future, we can explore more elegant ways of generating files
with instance-specific parameters correctly substituted. This proposal argues
that the extra verbosity is worth the benefit of a consistent, versioned API for
configuring core components.

## Design Details

TODO:
- which side gets used at runtime?
- new naming convention
- code examples for the merge
- sketch out the proposed case-by-case merge and see if it would actually be
  more complicated.

ComponentConfig APIs are currently defined as a separate API group for each
component, usually containing a single top-level Kind for configuring that
component. The naming convention for the API group is `{component}.config.k8s.io`,
and the convention for the Kind is `{Component}Configuration`. This KEP proposes
that the naming convention for the new Kind be `{Component}InstanceConfiguration`.

The canonical flag for providing a ComponentConfig file to a component is
`--config`. This KEP proposes that the canonical flag for providing an
instance-specific config file be `--instance-config`, and that the
instance-specific object not initially be permitted as part of a yaml stream
in the `--config` file (and vice-versa). This is for the sake of a simple
implementation and can be enabled in the future, if we decide it is useful.

As with sharable ComponentConfig parameters, command line flags for
instance-specific parameters should continue to function and take precedence
over values from the config file so that backwards compatibility is maintained.



### Test Plan

ComponentConfig APIs are typically tested similarly to API server APIs, for
example by testing round-trip conversions. Some tests specifically exercise
the ability to share a config object, such as the Dynamic Kubelet Config tests.

Components that opt-in to instance-specific Kinds should extend their existing
tests to include the instance-specific Kinds, except where those tests are
explicitly testing the ability to share a config object across concurrent
instances.

### Graduation Criteria

A beta-versioned ComponentConfig API is using the instance-specific object
approach.

### Upgrade / Downgrade Strategy

Since instance-specific config is currently only exposed on the command-line,
and no existing ComponentConfig at beta or later maturity levels need to be
retrofitted to remove instance-specific parameters, immutable upgrades are
relatively simple: just deploy the new resources with the new configuration
format (this is how we approached upgrades for ComponentConfig in general).
Immutable downgrades follow the same approach: deploy the new resources with
the old version and old config.

In-place upgrades, should not be significantly affected, though as usual
upgrades should take care to make the in-place changes to the config before
starting the new Kubelet version. If in-place downgrades are desired, the config
should be backed up before making changes, so that it can be restored later.
Correctly implementing in-place upgrades has always been an exercise for the
user, and this KEP does not change that.

Since backwards-compatibility of the command-line is maintained, in the short
term no action will be necessary to upgrade a component without updating its
config, though command-line flags will eventually be removed commensurate with
the overall ComponentConfig strategy. At that time (which will permit for
the required deprecation period and fair warning), it will be necessary to move
to the file-based approach.

### Version Skew Strategy

Any given version of a component should always refer to a config version that
it understands. Fortunately, this is relatively easy to coordinate via either
the Pod spec () or node startup scripts. Wholly new Pods are created on
DaemonSet upgrades, and if necessary the new set can refer to wholly new
ConfigMaps or contain an updated Pod template. Wholly new nodes are typically
created on node upgrades, and the new VMs can be deployed with updated config
metadata.

To be clear, an old Kubelet can't use the new config format, and shouldn't be
configured to use it. A new Kubelet can use either the new format, or continue
to use the old format until the deprecation period is up (in this case, when
all other flags are also deprecated).

## Implementation History

- Initial Draft: 1/8/2019
- Draft Merged: ?
- PR merged for first component using this approach: ?
- First release with this approach available in a beta component: ?