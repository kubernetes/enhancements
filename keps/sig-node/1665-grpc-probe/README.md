# KEP-1665: Add GRPC Probe

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Followup work (or optionally part of this)](#followup-work-or-optionally-part-of-this)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->


## Release Signoff Checklist

- [ ] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

GRPC probe - one more opportunity for define container status. Have grpc have official health check protocol: 
https://github.com/grpc/grpc/blob/master/doc/health-checking.md

## Motivation

GRPC - popular protocol for implement services. For implement health check most of developer use not official solution like:
https://github.com/grpc-ecosystem/grpc-health-probe

### Goals

Provide for k8 possability for have grpc probe natively(from box) 

### Non-Goals


## Proposal

```shell script
    readinessProbe:
      grpc:
        port: 8080
        host: localhost
      initialDelaySeconds: 5
      periodSeconds: 10
```

Inside metadata will be send user agent:

```go
md := metadata.New(map[string]string{
		"User-Agent": fmt.Sprintf("kube-probe/%s.%s", v.Major, v.Minor),
})
```

### Risks and Mitigations

Zero risk, users will be happy

## Design Details

```proto
message Handler {
  // One and only one of the following should be specified.
  // Exec specifies the action to take.
  // +optional
  optional ExecAction exec = 1;

  // HTTPGet specifies the http request to perform.
  // +optional
  optional HTTPGetAction httpGet = 2;

  // TCPSocket specifies an action involving a TCP port.
  // TCP hooks not yet supported
  // TODO: implement a realistic TCP lifecycle hook
  // +optional
  optional TCPSocketAction tcpSocket = 3;

  // GRPC specifies an action involving a TCP port.
  // +optional
  optional GRPCAction grpc = 4;
}

message GRPCAction {
  // Number or name of the port to access on the container.
  // Number must be in the range 1 to 65535.
  // Name must be an IANA_SVC_NAME.
  optional k8s.io.apimachinery.pkg.util.intstr.IntOrString port = 1;

  // Optional: Host name to connect to, defaults to the pod IP.
  // +optional
  optional string host = 2;
}
```

### Followup work (or optionally part of this)

API + Prober + Documentation

### Test Plan

Unit test + functional tests

### Graduation Criteria

### Upgrade / Downgrade Strategy

It is not breaking change, dont need

### Version Skew Strategy

## Implementation History

MR for k8 Prober: https://github.com/kubernetes/kubernetes/pull/89832

* 2020-04-04: MR for k8 Prober

## Drawbacks

## Alternatives

3rd party solutions like https://github.com/grpc-ecosystem/grpc-health-probe
