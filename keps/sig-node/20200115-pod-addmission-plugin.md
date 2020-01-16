---
title: Node-local pluggable pod admission framework
authors:
  - "@SaranBalaji90"
  - "@jaypipes"
owning-sig: sig-node
reviewers: TBD
approvers: TBD
editor: TBD
creation-date: 2020-01-15
last-updated: 2020-01-15
status: provisional
---

# Plugin support for pod admission handlers

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
- [Implementation History](#implementation-history)
- [References](#references)
<!-- /toc -->

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

Today, kubelet is responsible for determining if a Pod can execute on the node. Kubelet compares the required capabilities of the Pod against the discovered capabilities of both the worker node and the container runtime.

Kubelet will reject the Pod if any required  capabilities in the Pod.Spec are not supported by the container engine running on the node. Such capabilities might include the ability to set sysctl parameters, use of elevated system privileges or use of a non-default process mount. Likewise, kubelet checks the Pod against node capabilities; for example, the presence of a specific apparmor profile or host kernel.

These validations represent final, last-minute checks immediately before the Pod is started by the container runtime. These node-local checks differ from API-layer validations like Pod Security Policies or Validating Admission webhooks. Whereas the latter may be deactivated or removed by Kubernetes cluster administrators, the former node-local checks cannot be disabled. As such, they represent a final defense against malicious actors and misconfigured Pods.

It is not currently possible to add additional validations before admitting the Pod. This document proposes a framework for enabling additional node-local Pod admission checks.

## Motivation

Amazon Elastic Kubernetes Service (EKS) provides users a managed Kubernetes control plane. EKS users are provisioned a Kubernetes cluster running on AWS cloud infrastructure. While the EKS user does not have host-level administrative access to the master nodes, it is important to point out that they do have administrative rights on that Kubernetes cluster.

The EKS user’s worker node administrative access depends on the type of worker node the EKS user chooses. EKS users have three options. The first option is to bring their own EC2 instances as worker nodes. The second option is for EKS users to launch a managed worker node group. These first two options both result in the EKS user maintaining full host-level administrative rights on the worker nodes. The final option — the option that motivated this proposal — is for the EKS user to forego worker node management entirely using AWS Fargate, a serverless computing environment. With AWS Fargate, the EKS user does not have host-level administrative access to their worker node; in fact, the worker node runs on a serverless computing platform that abstracts away the entire notion of a host.

In building the AWS EKS support for AWS Fargate, the AWS Kubernetes engineering team faced a dilemma: how could they prevent Pods destined to run on Fargate nodes from using host networking or assuming elevated host user privileges?

The team initially investigated using a Pod Security Policy (PSP) that would prevent Pods with a Fargate scheduler type from having an elevated security context or using host networking. However, because the EKS user has administrative rights on the Kubernetes cluster, API-layer constructs such as a Pod Security Policy may be deleted, which would effectively disable the effect of that PSP. Likewise, the second solution the team landed on — using Node taints and tolerations — was similarly bound to the Kubernetes API layer, which meant EKS users could modify those Node taints and tolerations, effectively disabling the effects. A third potential solution involving OCI hooks was then investigated. OCI hooks are separate executables that an OCI-compatible container runtime invokes that can modify the behaviour of the containers in a sandbox. While this solution would have solved the API-layer problem, it introduced other issues, such as the inefficiency of downloading the container image to the Node before the OCI hook was run.

The final solution the EKS team settled on involved changing kubelet itself to support additional node-local Pod admission checks. This KEP outlines the EKS team’s approach and proposes upstreaming these changes to kubelet in order to allow extensible node-local last-minute validation checks. This functionality will enable cloud providers who support nodeless/serverless worker nodes to restrict Pods based on fields other than those already being validated by kubelet.

### Goals

- Allow deployers of fully managed worker nodes to have control over Pods running on those nodes.

- Enable node-local Pod admission checks without requiring changes to kubelet.

### Non-Goals

- Move existing validations to “out of tree” plugins.

- Change existing API-layer validation solutions such as Pod Security Policies and validating admission webhooks.

## Proposal

The approach taken is similar to the container networking interface (CNI) plugin architecture. With CNI, kubelet invokes one or more CNI plugin binaries on the host to set up a Pod’s networking. kubelet discovers available CNI plugins by [examining](https://github.com/kubernetes/kubernetes/blob/dd5272b76f07bea60628af0bb793f3cca385bf5e/pkg/kubelet/dockershim/docker_service.go#L242) a well-known directory (`/etc/cni/net.d`) for configuration files and [loading](https://github.com/kubernetes/kubernetes/blob/dd5272b76f07bea60628af0bb793f3cca385bf5e/pkg/kubelet/dockershim/docker_service.go#L248) plugin [descriptors](https://github.com/kubernetes/kubernetes/blob/f4db8212be53c69a27d893d6a4111422fbce8008/pkg/kubelet/dockershim/network/plugins.go#L52) upon startup.

To support pluggable validation for pod admission on the worker node, we propose to have kubelet similarly discover node-local Pod admission plugins listed in a new PodAdmissionPluginDir flag.

Other option is to enable this functionality through feature flag “enablePodAdmissionPlugin” and have the directory path defined inside the kubelet itself.

### Design Details

#### Configuration file

Node-local Pod admission plugins will be listed in a configuration file. The Plugins field indicates the list of plugins to invoke before admitting the Pod.

```
{
  "name": "admission-plugin",
  "version": "0.1",
  "plugins": [
    {
      "name": "sysctlcheck",
      "type": "shell"
    },
    {
      "type": "fargatecheck",
      "type": "shell"
    }
  ]
}
```

A node-local Pod admission plugin has the following structure:

```
package podadmission

// PluginType indicates type of the admission plugin
type PluginType string

const (
    PluginTypeShell  PluginType = "shell"  // binary to execute.
    PluginTypeGRPC   PluginType = "grpc"   // Local port on the host.
    PluginTypeSocket PluginType = "socket" // fd to connect to.
)

// AdmissionPluginManager is the podAdmitHandler shim for external plugins.
type AdmissionPluginManager struct {
    confDir      string
    binDir       string
    pluginConfig *PluginConfig
}

// PluginConfig represents the plugin configuration file
type PluginConfig struct {
    Name         string              `json:"name,omitempty"`
    Version      string              `json:"version,omitempty"`
    Plugins      []*AdmissionPlugin  `json:"plugins"`
}

// AdmissionPlugin represents individual plugins specified in the configuraiton.
type AdmissionPlugin struct {
    Name     string     `json:"name"`
    Type     PluginType `json:"type"`
    GrpcPort int        `json:"grcPort,omitempty"` // not required for shell/socket type
    Socket   string     `json:"socket,omitempty"` // not required for shell/grpc
}

// 
func NewManager(confDir, binDir string) *AdmissionPluginManager {
    admissionPluginMgr := &AdmissionPluginManager{
        confDir: confDir,
        binDir:  binDir,
    }
    admissionPluginMgr.initializePlugins() // sort and read the conf file and updates list of plugins.

    return admissionPluginMgr
}

func (apm *AdmissionPluginManager) Admit(attrs *lifecycle.PodAdmitAttributes) lifecycle.PodAdmitResult {
    var admit bool
    var message string

    for _, plugin := range apm.pluginConfig.Plugins {
        switch plugin.Type {
        case PluginTypeShell:
            // exec
        case PluginTypeGRPC:
            // Fetch a connection to gRPC service
        case PluginTypeSocket:
            // Fetch a connection through unix socket.
        }
    }

    response := lifecycle.PodAdmitResult{
        Admit:  admit,
        Message: message,
    }
    if !admit {
        response.Reason = unsupportedPodSpecMessage
    }
    return response
}
```
#### Feature gate

This functionality adds a new feature gate named “PodAdmissionPlugin” which decides whether to invoke admission plugin or not.

#### Kubelet to pod admission plugin communication

Kubelet will encode the pod spec and invoke each admission plugin's Admit() method. After decoding the pod spec, plugins can perform additional validations and return the encoded form of the struct mentioned below to kubelet to decide whether to admit the pod or not.

```
AdmitResult { 
   Admit bool 
   Reason string 
   Message string 
}
```

#### Implementation detail

As part of this implementation, new sub package will be added to pkg/kubelet/lifecycle. In-tree admission handler shim will be included in this package, which will be responsible for discovering and invoking the pod admission plugins available on the host to decide on whether to admit the pod or not.

If the plugin does not respond or if it's crashing, then pod will not be accepted by the kubelet. Because if kubelet ignores the response and schedules the pod then intention of not executing pod with specific needs on these worker nodes will be violated. These plugins doesn't mutate the Pod object and can be invoked in parallel since there is no dependency between them.

### Test Plan

-  Apart from unit tests, add integration test to invoke process running on the node to decide on pod admission
    * Process should return true and pod should be executed on the worker node.
    * Process should return false and pod should not be admitted on the worker node.
    * Process doesn’t respond and pod should not be admitted on the worker node.
    * Tests with multiple node-local pod admission handlers
    * Tests of multiple node-local pod admission handlers with one intentionally non-functioning or crashing plugin

### Graduation Criteria

#### Alpha -> Beta Graduation

- Ensure there is minimal or acceptable performance degradation when external plugins are enabled on the node. This includes monitoring kubelet CPU and memory utilization. This is because of additional call which kubelet will perform to invoke plugins.

- Have the feature tested by other community members and address the feedback.

#### Beta -> GA Graduation

- TODO

## Implementation History

- 2020-01-15: Initial KEP sent out for initial review, including Summary, Motivation and Proposal

## Other solutions discussed

Why we didn’t go over CRI shim or using OCI hook approach?

1. Even before Kubelet invokes container [runtime](https://github.com/kubernetes/kubernetes/blob/v1.14.6/pkg/kubelet/kubelet.go#L1665) it sets up few things for pod, including [cgroup](https://github.com/kubernetes/kubernetes/blob/v1.14.6/pkg/kubelet/kubelet.go#L1605), volume mount, [pulling secrets](https://github.com/kubernetes/kubernetes/blob/v1.14.6/pkg/kubelet/kubelet.go#L1644) for pods etc. 

2. OCI hook is invoked just before running the container, therefore Kubelet would have already downloaded the image as well. Even if hook rejects the Pod object, there is no good way to emit events on why hook failed Pod creation.

## References

- https://kubernetes.io/docs/tutorials/clusters/apparmor/
- https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/0035-20190130-topology-manager.md
- https://github.com/kubernetes/community/pull/1934/files
- https://kubernetes.io/docs/concepts/policy/pod-security-policy/
- https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
