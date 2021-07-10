# KEP-2819: CNI traffic shaping support

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-goals](#non-goals)
- [Proposal](#proposal)
  - [Pod Setup](#pod-setup)
  - [Pod Teardown](#pod-teardown)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
  - [CNI plugin part](#cni-plugin-part)
  - [Kubernetes part](#kubernetes-part)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

## Summary

Make kubelet cni network plugin support basic traffic shapping capability `bandwidth`.

## Motivation

Currently the kubenet code supports applying basic traffic shaping during pod setup.
This will happen if bandwidth-related annotations have been added to the pod's metadata.

Kubelet CNI code doesn't support it yet, though CNI has already added a [traffic sharping plugin](https://github.com/containernetworking/plugins/tree/master/plugins/meta/bandwidth).
We can replicate the behavior we have today in kubenet for kubelet CNI network plugin if we feel this is an important feature.

### Goals

* Support traffic shaping for CNI network plugin in Kubernetes.

### Non-goals

* CNI plugins to implement this sort of traffic shaping guarantee.


## Proposal

If kubelet starts up with `network-plugin = cni` and user enabled traffic shaping via the network plugin configuration,
it would then populate the runtimeConfig section of the config when calling the bandwidth plugin.

Traffic shaping in Kubelet CNI network plugin can work with ptp and bridge network plugins.

### Pod Setup

When we create a pod with bandwidth configuration in its metadata, for example,

```json
{
    "kind": "Pod",
    "metadata": {
        "name": "iperf-slow",
        "annotations": {
            "kubernetes.io/ingress-bandwidth": "10M",
            "kubernetes.io/egress-bandwidth": "10M"
        }
    }
}
```

Kubelet would firstly parse the ingress and egress bandwidth values and transform them to ingressRate and egressRate for cni bandwidth plugin.
Kubelet would then detect whether user has enabled the traffic shaping plugin by checking the following CNI config file:

```json
{
  "type": "bandwidth",
  "capabilities": {"trafficShaping": true}
}
```

If traffic shaping plugin is enabled, kubelet would populate the runtimeConfig section of the config when call the bandwidth plugin:

```json
{
  "type": "bandwidth",
  "runtimeConfig": {
    "trafficShaping": {
      "ingressRate": "X",
      "egressRate": "Y"
    }
  }
}
```

### Pod Teardown

When we delete a pod, kubelet will bulid the runtime config call cni plugin DelNetworkList API, which will remove this pod's bandwidth configuration.

## Graduation Criteria

* Add traffic shaping as part of the Kubernetes e2e runs and ensure tests are not failing.

## Implementation History

### CNI plugin part

* [traffic shaping plugin](https://github.com/containernetworking/plugins/pull/96)
* [support runtime config](https://github.com/containernetworking/plugins/pull/138)

### Kubernetes part

* [add traffic shaping support](https://github.com/kubernetes/kubernetes/pull/63194)

## Drawbacks

None

## Alternatives

None