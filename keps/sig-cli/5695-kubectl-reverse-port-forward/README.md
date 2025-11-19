# KEP-5695: kubectl reverse port-forward

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Local Development with Remote Debugging](#story-1-local-development-with-remote-debugging)
    - [Story 2: Testing Webhooks in Development](#story-2-testing-webhooks-in-development)
    - [Story 3: Database Migration from Local Tools](#story-3-database-migration-from-local-tools)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Implementation Overview](#implementation-overview)
  - [Test Plan](#test-plan)
    - [Prerequisite Testing Updates](#prerequisite-testing-updates)
    - [Unit Tests](#unit-tests)
    - [Integration Tests](#integration-tests)
    - [E2E Tests](#e2e-tests)
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
  - [Alternative 1: Third-Party Tools](#alternative-1-third-party-tools)
  - [Alternative 2: Exec-based Approach](#alternative-2-exec-based-approach)
  - [Alternative 3: Service Mesh Integration](#alternative-3-service-mesh-integration)
  - [Alternative 4: API Server Proxy](#alternative-4-api-server-proxy)
- [Infrastructure Needed](#infrastructure-needed)
- [References](#references)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
  - [ ] E2E Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA E2E tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA E2E tests to prove flake free
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

Add reverse port-forwarding capability to `kubectl port-forward`, enabling users to expose local ports on their workstation to containers running in Kubernetes pods. This is the equivalent of SSH's `-R` (remote port forwarding) flag, allowing containers to initiate connections back to services running on the developer's local machine.

## Motivation

The current `kubectl port-forward` command only supports forward port-forwarding (local port → pod port), which allows users to access services running in pods from their local machine. However, there are many common development and debugging scenarios where the reverse is needed: allowing a pod to access services running on the developer's local machine.

Currently, developers must use workarounds such as:
- Deploying sidecar containers with SSH servers and using SSH tunneling
- Setting up VPN connections to the cluster
- Deploying temporary services that proxy to external endpoints
- Using third-party tools like telepresence or similar

These workarounds add complexity, security concerns, and maintenance overhead. A native reverse port-forward capability would significantly improve the developer experience.

### Goals

- Enable reverse port-forwarding from pod ports to local ports via `kubectl port-forward`
- Maintain backward compatibility with existing `kubectl port-forward` usage
- Ensure the feature works with the WebSocket/HTTP2 streaming protocol
- Provide clear error messages and diagnostics
- Support standard kubectl patterns (pod selection, namespace handling, etc.)
- Allow multiple concurrent reverse port-forwards

### Non-Goals

- Replacing or modifying the existing forward port-forward functionality
- Supporting persistent tunnels that survive pod restarts (connections are ephemeral)
- Implementing `bind_address` support for exposing the reverse tunnel to other pods (initial implementation will only expose to the target pod)
- Providing load balancing across multiple pods
- Supporting UDP port forwarding (TCP only for initial implementation)

## Proposal

Add a new flag `--reverse` (or `-R`) to the `kubectl port-forward` command that reverses the direction of the port forwarding. When this flag is used, kubectl will:

1. Establish a connection to the kubelet running the target pod
2. Create a listener in the pod's network namespace on the specified remote port
3. Forward incoming connections from that port back to kubectl
4. kubectl will then forward these connections to the specified local port

### User Stories

#### Story 1: Local Development with Remote Debugging

As a developer, I want to debug a microservice running in a Kubernetes pod that needs to call back to my local development server, so that I can test webhook integrations without deploying my development code to the cluster.

**Example:**
```bash
# Start local webhook server on port 8080
python -m http.server 8080

# In another terminal, expose it to the pod
kubectl port-forward --reverse mypod 8080:8080

# Now the pod can access http://localhost:8080 which goes to my local machine
```

#### Story 2: Testing Webhooks in Development

As a developer working on a Kubernetes admission webhook, I want my webhook running locally to receive requests from the API server running in my development cluster, so that I can rapidly iterate on webhook logic without repeatedly building and deploying container images.

**Example:**
```bash
# Run webhook locally
./my-webhook --port 9443

# Expose it to the API server pod
kubectl port-forward --reverse -n kube-system api-server-pod 9443:9443

# Now the API server can call https://localhost:9443 for webhook validation
```

#### Story 3: Database Migration from Local Tools

As a database administrator, I want to run database migration tools from my local machine against a database pod without exposing the database outside the cluster, using a secure reverse tunnel.

**Example:**
```bash
# Expose local port 5432 to the database pod
kubectl port-forward --reverse postgres-pod 15432:5432

# In the pod, connect to localhost:15432 to reach the local migration tool
# that is listening on port 5432
```

### Notes/Constraints/Caveats

1. **Security Consideration**: Reverse port-forwarding exposes local services to remote pods. Users must understand the security implications and only use this feature in development/debugging scenarios.

2. **Network Namespace**: The listening port is created in the pod's network namespace, accessible only to processes within that pod (not to other pods in the cluster).

3. **Protocol Limitations**: Initial implementation will use the same streaming protocol as forward port-forward (WebSocket or HTTP2).

4. **Connection Lifecycle**: Connections are active only while the `kubectl port-forward` command is running. If kubectl terminates, all connections are closed.

5. **Port Conflicts**: If the remote port is already in use in the pod, the operation will fail with a clear error message.

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| **Security**: Exposing local services to pods could be misused | Document security implications clearly, recommend use only for development/debugging, consider adding audit logging |
| **Security**: Network Policy Bypass | Document that reverse tunnels allow pods to access external resources via the developer's machine, potentially bypassing cluster egress policies. |
| **Resource Exhaustion**: Malicious or buggy pods could open many connections | Implement connection limits and resource quotas, add monitoring metrics |
| **Port Conflicts**: Remote port already in use in pod | Provide clear error messages, allow user to specify alternative ports |
| **Protocol Complexity**: Bi-directional streaming protocol is complex | Extensive testing, reuse existing port-forward infrastructure where possible |
| **Version Skew**: Old kubelets may not support reverse forwarding | Feature gate protection, graceful degradation with clear error messages |

## Design Details

### API Changes

No API changes to Kubernetes objects are required. The changes are to the kubectl command-line interface and the kubelet port-forward protocol.

**kubectl CLI Changes:**
```bash
# Existing forward port-forward (unchanged)
kubectl port-forward pod-name 8080:80

# New reverse port-forward syntax
kubectl port-forward --reverse pod-name 8080:80
# or
kubectl port-forward -R pod-name 8080:80

# Multiple ports (both forward and reverse)
kubectl port-forward --reverse pod-name 8080:80 9090:9000
```

**Port Specification Semantics:**
- Forward mode (existing): `LOCAL_PORT:REMOTE_PORT` - local port forwards to remote port
- Reverse mode (new): `REMOTE_PORT:LOCAL_PORT` - remote port forwards to local port

This maintains consistency with SSH syntax where the semantics of the port spec change based on the direction flag.

### Implementation Overview

The implementation builds upon the existing port-forward infrastructure with the following components:

**1. kubectl Changes** (`pkg/cmd/portforward/`):
- Add `--reverse` flag to port-forward command
- Modify port spec parsing to handle reverse semantics
- Implement reverse connection handler that:
  - Listens for incoming connections from kubelet
  - Establishes connections to local ports
  - Forwards traffic bidirectionally

**2. Kubelet Changes** (`pkg/kubelet/`):
- Extend PortForward API to support reverse mode
- Implement pod network namespace listener creation using `socat` or native Go listeners
- Handle incoming pod connections and stream them to kubectl
- Clean up listeners when kubectl disconnects

**3. Protocol Changes** (`pkg/kubelet/cri/streaming/portforward/`):
- Extend streaming protocol to support:
  - Reverse mode handshake
  - Connection establishment notifications (pod → kubectl direction)
  - Bidirectional data streams

**4. Flow Diagram:**

```mermaid
sequenceDiagram
    participant User
    participant kubectl
    participant APIServer as API Server
    participant kubelet
    participant Pod

    Note over User,Pod: Setup Phase
    User->>kubectl: port-forward --reverse pod 8080:80
    kubectl->>APIServer: Establish streaming connection
    APIServer->>kubelet: Forward port-forward request
    kubelet->>Pod: Create listener on port 8080 in pod netns

    Note over User,Pod: Connection Phase
    Pod->>Pod: Process connects to localhost:8080
    Pod->>kubelet: Connection received
    kubelet->>kubectl: Stream new connection
    kubectl->>User: Connect to localhost:80

    Note over User,Pod: Data Transfer
    User->>kubectl: Send data
    kubectl->>kubelet: Forward data
    kubelet->>Pod: Deliver to pod
    Pod->>kubelet: Response data
    kubelet->>kubectl: Forward response
    kubectl->>User: Deliver response
```

**5. Implementation Components:**

- **Port Listener**: Create a TCP listener in the pod's network namespace
  - Native Go implementation using a Go network namespace library (such as [vishvananda/netns](https://github.com/vishvananda/netns)).
  - This avoids external dependencies like `socat` or `nsenter` and provides better error handling and resource control.

- **Connection Multiplexing**: Handle multiple concurrent connections over a single WebSocket/HTTP2 stream
  - Each new connection from pod creates a new stream channel
  - kubectl maps each channel to a new local connection

- **Error Handling**:
  - Port already in use
  - Connection refused on local port
  - Network namespace access failures
  - Stream protocol errors

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

#### Prerequisite Testing Updates

- Ensure existing port-forward tests are not affected by the changes
- Add test coverage for the new protocol messages
- Verify backward compatibility with clusters not supporting reverse port-forward

#### Unit Tests

**Existing Test Infrastructure:**

Test stubs for reverse port-forward have been prepared in `k8s.io/kubectl/pkg/cmd/portforward/portforward_test.go`:
- `TestReversePortForwardFlagParsing`: Tests `--reverse` flag parsing
- `TestReversePortSpecificationSyntax`: Tests port specification format validation
- `TestReversePortForwardValidation`: Tests validation logic for reverse mode
- `TestReverseForwardModeConflict`: Tests that forward/reverse modes cannot be mixed
- `TestPortRangeValidation`: Port range validation helper tests

These tests are currently skipped (waiting for implementation) and will be enabled during alpha implementation.

**Required Unit Test Coverage:**

- `k8s.io/kubectl/pkg/cmd/portforward`:
  - Port spec parsing for reverse mode (REMOTE:LOCAL format)
  - Reverse connection handling logic
  - Error handling for reverse-specific failures
  - Flag validation (`--reverse` with various port specs)
  - Port range validation (1-65535 for remote, 0-65535 for local)

- `k8s.io/kubernetes/pkg/kubelet/cri/streaming/portforward`:
  - Reverse mode protocol handshake
  - Connection multiplexing
  - Listener cleanup on disconnect
  - Error handling for port conflicts

- `k8s.io/kubernetes/pkg/kubelet`:
  - Pod network namespace listener creation
  - Port conflict detection
  - Resource cleanup on connection termination

#### Integration Tests

- Test reverse port-forward with real kubelet and pod
- Test multiple concurrent reverse connections
- Test connection handling when local port is unavailable
- Test cleanup when kubectl is terminated
- Test version skew scenarios (new kubectl with old kubelet)
- Test feature gate enablement/disablement

Integration test location: `test/integration/kubectl/portforward_reverse_test.go`

#### E2E Tests

- End-to-end reverse port-forward test with real cluster
  - Create pod with simple client application
  - Start reverse port-forward to local HTTP server
  - Verify pod can connect and receive responses
  - Test connection persistence and cleanup

- Test with multiple pods and concurrent connections
- Test error scenarios (port conflicts, connection failures)
- Test with different pod runtime (containerd, cri-o)

E2E test location: `test/e2e/kubectl/portforward.go` (extend existing tests)

### Graduation Criteria

#### Alpha

- Feature implemented behind `KubectlReversePortForward` feature gate
- Basic functionality working (single port, single connection)
- Unit tests for kubectl and kubelet components (test infrastructure already prepared)
- Enable and complete existing test stubs in `portforward_test.go`
- Initial e2e tests completed
- Documentation drafted

#### Beta

- Feature gate enabled by default
- Support for multiple concurrent connections
- Support for multiple ports in single command
- Comprehensive integration tests in Testgrid
- Metrics for connection count, errors, and duration
- Security review completed
- User-facing documentation published
- Gather feedback from early adopters
- Address all major bugs and issues from alpha

#### GA

- Feature gate removed (always enabled)
- No significant bugs reported for 2+ releases
- Conformance tests if applicable
- All metrics stable and documented
- Performance testing completed
- Real-world usage examples and case studies
- All documentation complete and reviewed

### Upgrade / Downgrade Strategy

**Upgrade:**
- New kubectl with old kubelet: Feature detection via protocol negotiation. If kubelet doesn't support reverse mode, kubectl provides a clear error message.
- Old kubectl with new kubelet: No impact, forward compatibility maintained.

**Downgrade:**
- Reverse port-forward sessions will be terminated when kubelet is downgraded.
- Users will need to use old kubectl version if they downgrade kubelet.
- No data loss or corruption, just feature unavailability.

### Version Skew Strategy

The feature will handle version skew gracefully:

1. **kubectl → kubelet**: Protocol negotiation during initial handshake
   - New kubectl sends reverse mode capability flag
   - Old kubelet responds with "not supported" error
   - kubectl displays user-friendly error: "Reverse port-forwarding requires kubelet version >= v1.34"

2. **Feature Gate**: Behind `KubectlReversePortForward` feature gate
   - Alpha: Disabled by default
   - Beta: Enabled by default but can be disabled
   - GA: Always enabled

3. **Backward Compatibility**: Existing forward port-forward remains unchanged and unaffected by this feature

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `KubectlReversePortForward`
  - Components depending on the feature gate:
    - kubelet
    - kubectl (command-line flag could also gate the feature)

###### Does enabling the feature change any default behavior?

No. The feature only activates when users explicitly use the `--reverse` flag with `kubectl port-forward`. Existing `kubectl port-forward` behavior is completely unchanged.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate on kubelet will cause reverse port-forward attempts to fail with a clear error message. Active reverse port-forward sessions will be terminated. No persistent state is created, so disabling is clean.

###### What happens if we reenable the feature if it was previously rolled back?

The feature will work again immediately. There is no persistent state to reconcile.

###### Are there any tests for feature enablement/disablement?

Yes, integration tests will cover:
- Feature gate disabled: verify error message is clear
- Feature gate enabled: verify functionality works
- Runtime enablement/disablement scenarios

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

**Rollout failure scenarios:**
- Incompatible protocol changes: Mitigated by protocol versioning and negotiation
- Resource exhaustion: Mitigated by connection limits and monitoring

**Impact on running workloads:**
- No impact on existing workloads
- Only affects users actively using reverse port-forward
- Failure would only terminate reverse port-forward sessions, not affect pods

###### What specific metrics should inform a rollback?

- High error rate in `kubectl_portforward_reverse_errors_total`
- Increased kubelet CPU/memory usage correlating with reverse port-forward usage
- Increased pod connection failures
- Reports of existing forward port-forward breaking

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Testing plan includes:
- Upgrade kubelet with feature gate disabled → enabled
- Downgrade kubelet with active reverse port-forwards
- Upgrade kubectl independent of kubelet version
- Mixed cluster with some nodes supporting feature, some not

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No. This is a purely additive feature.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Metrics will expose:
- `kubectl_portforward_reverse_connections_total`: Counter of reverse port-forward connections
- `kubectl_portforward_reverse_active_connections`: Gauge of currently active connections

###### How can someone using this feature know that it is working for their instance?

- Success indicators:
  - kubectl command runs without error
  - Connection from pod to localhost:PORT succeeds
  - Data flows bidirectionally

- Failure indicators:
  - Clear error messages from kubectl
  - Connection refused errors in pod
  - Timeout on connection attempts

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- 99% of reverse port-forward establishment attempts succeed (when feature gate is enabled)
- Connection establishment latency < 500ms p99
- Zero impact on existing forward port-forward latency/success rate

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- Metrics:
  - `kubectl_portforward_reverse_connections_total` - labeled by status (success/failure)
  - `kubectl_portforward_reverse_errors_total` - labeled by error type
  - `kubectl_portforward_reverse_connection_duration_seconds` - histogram
  - `kubelet_portforward_reverse_listeners_active` - gauge

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Initial implementation includes all necessary metrics. Future enhancements could add:
- Bytes transferred per connection
- Connection failure reasons (detailed breakdown)
- Per-pod reverse port-forward usage

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No cluster-level services required. Dependencies:
- Kubelet must be running (already required for port-forward)
- Network namespace support in container runtime
- No external services needed

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new API server calls. The feature uses the existing port-forward streaming endpoint between kubectl and kubelet.

###### Will enabling / using this feature result in introducing new API types?

No. This is a client-side feature using existing streaming protocols.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. This feature is opt-in and isolated from existing functionality.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Resource usage per active reverse port-forward session:
- kubectl: Minimal CPU/memory for connection proxying (~1-5 MB per connection)
- kubelet: Minimal CPU/memory for listener and stream handling (~1-5 MB per connection)
- No disk IO unless connection logging is enabled

The impact is similar to existing forward port-forward resource usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

Potential risks:
- Socket exhaustion: Mitigated by connection limits (default: 10 concurrent connections per reverse port-forward)
- PID exhaustion: Minimal (no new process per connection in native Go implementation)

Limits will be documented and configurable.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

- This feature fails when the API server is unavailable. The reverse port-forward connection
  is proxied through the API server. If the API server is unavailable, the reverse port-forward
  functionality does not work (this applies to the legacy forward port-forward as well,
  so there is no detectable difference in this scenario).

- No persistent state is stored in etcd for reverse port-forward sessions. The feature behaves
  identically to existing forward port-forward in this regard: connections are ephemeral and
  exist only in memory. If the API server becomes unavailable during an active session, the
  connection will be dropped and the reverse port-forward session will terminate immediately.

###### What are other known failure modes?

- **Failure Mode**: Port already in use in the target pod's network namespace.
  When attempting to create a reverse port-forward to a port that is already bound
  in the pod, the operation will fail.
    - Detection: Error message during reverse port-forward setup: `Error from server: error creating reverse port-forward listener: port %d already in use`
    - Mitigations: User should select a different remote port that is not in use in the pod.
      Use `kubectl exec <pod> -- netstat -tlnp` or `ss -tlnp` to check which ports are in use.
    - Diagnostics: kubectl will display the error immediately upon attempting to establish
      the reverse port-forward. Kubelet logs will show listener creation failure with detailed
      error information at log level 4 or higher (`--v=4`).
    - Testing: Unit and integration tests cover this case by attempting to bind to the same
      port twice.

- **Failure Mode**: Local port unreachable or local service not listening.
  When a pod attempts to connect through the reverse tunnel, but the local port
  on the user's machine has no service listening or the service is unreachable.
    - Detection: Connection errors logged by kubectl when pod initiates connection.
      The pod will receive connection refused errors when attempting to connect.
    - Mitigations: User must ensure local service is running and listening on the specified
      port before the pod attempts to connect. Use `netstat -tlnp | grep <port>` or
      `lsof -i :<port>` to verify local service is listening.
    - Diagnostics: kubectl logs will show `dial tcp 127.0.0.1:<port>: connect: connection refused`.
      From within the pod, connection attempts will fail with connection timeouts or refused errors.
    - Testing: E2E tests cover this scenario by establishing reverse port-forward without
      a local listener, then attempting connections from the pod.

- **Failure Mode**: Network namespace access failure in the pod.
  The kubelet cannot access the pod's network namespace to create the listener,
  typically due to permission issues or container runtime problems.
    - Detection: Error during listener creation: `cannot access network namespace for pod <name>`
    - Mitigations: Check kubelet has proper permissions to access container network namespaces.
      Verify the container runtime (containerd/CRI-O) is functioning correctly. Check SELinux/AppArmor
      policies are not blocking namespace operations.
    - Diagnostics: Kubelet logs will show detailed error with stack trace at log level 4 or higher.
      Use `kubectl describe pod <name>` to check pod status and runtime errors.
    - Testing: Integration tests with permission restrictions simulate this failure mode.

- **Failure Mode**: Feature gate disabled but user attempts reverse port-forward.
  During alpha/beta phases, if the feature gate is disabled on the kubelet,
  reverse port-forward attempts will fail.
    - Detection: Error message from kubectl: `reverse port-forwarding is not supported by this cluster (requires feature gate KubectlReversePortForward)`
    - Mitigations: Enable the feature gate on kubelet: `--feature-gates=KubectlReversePortForward=true`
    - Diagnostics: Protocol negotiation will fail during initial handshake. Kubectl will
      receive a clear error message indicating the feature is not supported.
    - Testing: Integration tests verify error messages when feature gate is disabled.

- **Failure Mode**: Connection limit exhaustion from malicious or buggy pod.
  A pod opens more concurrent connections than the configured limit (default: 10).
    - Detection: New connection attempts fail with error: `reverse port-forward connection limit reached`
    - Mitigations: This is intentional rate limiting to prevent resource exhaustion. User can
      wait for existing connections to close, or adjust the limit if appropriate for their use case.
    - Diagnostics: Metric `kubectl_portforward_reverse_connections_rejected_total{reason="limit"}` will increment.
      kubectl will log connection rejections at verbose log levels.
    - Testing: Load tests verify connection limits are enforced correctly.

- **Failure Mode**: Network policy or firewall blocking kubectl-to-API-server communication.
  Network policies or firewalls prevent kubectl from establishing streaming connection to API server.
    - Detection: kubectl command hangs during connection establishment or fails with timeout.
    - Mitigations: Check network connectivity: `curl -k https://<api-server>:6443/healthz`.
      Verify kubeconfig is correct and credentials are valid. Check for network policies
      blocking egress from the client machine or ingress to the API server.
    - Diagnostics: kubectl with verbose logging (`-v=7`) will show HTTP connection attempts
      and timeout errors. Connection attempts will eventually timeout with `connection timeout` errors.
    - Testing: Manual tests with network policies verify error messages and behavior.

- **Failure Mode**: Version skew between kubectl and kubelet.
  A new kubectl with reverse port-forward support communicating with an old kubelet
  that does not support the feature.
    - Detection: kubectl displays clear error: `reverse port-forwarding requires kubelet version >= v1.34, but kubelet version is v1.33`
    - Mitigations: Upgrade kubelet to a version supporting reverse port-forward, or use
      forward port-forward if appropriate for the use case.
    - Diagnostics: Protocol negotiation during handshake will indicate version incompatibility.
      kubectl will parse kubelet version from handshake response.
    - Testing: Integration tests verify version detection and clear error messages with
      old kubelet versions.

###### What steps should be taken if SLOs are not being met to determine the problem?

- **Step 1**: Check if the feature is actually being used with reverse mode by running
  kubectl with verbose logging to see the mode and connection details.

For `kubectl port-forward --reverse`:

```bash
# Run simple reverse port-forward command with higher verbosity to see relevant logging.
# Look for successful connection upgrade with response 101 Switching Protocols and
# reverse mode indication in the logs.
# Example: kubectl port-forward --reverse -v=7 <POD> <REMOTE_PORT>:<LOCAL_PORT>

$ kubectl port-forward --reverse -v=7 nginx 8080:8080
...
I0120 10:15:23.456789 12345 round_trippers.go:463] POST https://127.0.0.1:6443/api/v1/namespaces/default/pods/nginx/portforward
I0120 10:15:23.456790 12345 round_trippers.go:469] Request Headers:
I0120 10:15:23.456791 12345 round_trippers.go:473]     X-Stream-Protocol-Version: portforward.k8s.io
I0120 10:15:23.456792 12345 round_trippers.go:473]     X-Reverse-Port-Forward: true
...
I0120 10:15:23.478123 12345 round_trippers.go:574] Response Status: 101 Switching Protocols in 21 milliseconds
I0120 10:15:23.478234 12345 portforward.go:234] Reverse port-forward mode enabled
I0120 10:15:23.478345 12345 portforward.go:245] Created listener in pod on port 8080
Forwarding from 0.0.0.0:8080 in pod -> localhost:8080
```

- **Step 2**: Verify the local service is actually listening on the specified port:

```bash
# Check if local service is listening
$ netstat -tlnp | grep 8080
# or
$ lsof -i :8080
# or
$ ss -tlnp | grep 8080

# If no service is listening, start your local service first:
$ python -m http.server 8080  # example local service
```

- **Step 3**: From within the pod, verify the listener was created and test connectivity:

```bash
# Check if port is listening in pod
$ kubectl exec nginx -- netstat -tln | grep 8080
tcp        0      0 0.0.0.0:8080           0.0.0.0:*               LISTEN

# Test connection from within pod
$ kubectl exec nginx -- curl -v http://localhost:8080
# Should successfully connect to your local service
```

- **Step 4**: Check kubelet logs for listener creation and connection handling:

```bash
# View kubelet logs on the node running the pod
$ journalctl -u kubelet -f --since "5 minutes ago" | grep "reverse\|portforward"

# Look for messages like:
# "Created reverse port-forward listener on port 8080 for pod nginx"
# "Accepted connection from pod to reverse port-forward port 8080"
# "Error creating listener: port already in use"
```

- **Step 5**: Review metrics to identify patterns:

```bash
# Check reverse port-forward metrics from API server/kubelet
$ curl -s http://localhost:10255/metrics | grep portforward_reverse

# Key metrics to examine:
# kubectl_portforward_reverse_connections_total{status="success"}
# kubectl_portforward_reverse_connections_total{status="failure"}
# kubectl_portforward_reverse_errors_total{error_type="port_conflict"}
# kubectl_portforward_reverse_errors_total{error_type="local_unreachable"}
# kubectl_portforward_reverse_connection_duration_seconds
# kubelet_portforward_reverse_listeners_active
```

- **Step 6**: Verify feature gate is enabled (for pre-GA releases):

```bash
# Check if feature gate is enabled on kubelet
$ kubectl get --raw /api/v1/nodes/<node-name>/proxy/configz | jq '.featureGates'

# Should show:
# {
#   "KubectlReversePortForward": true
# }

# If disabled, enable it in kubelet configuration or command line:
$ kubelet --feature-gates=KubectlReversePortForward=true ...
```

- **Step 7**: Test with a simple end-to-end scenario to isolate the problem:

```bash
# Terminal 1: Start simple local HTTP server
$ python3 -m http.server 9999
Serving HTTP on 0.0.0.0 port 9999 ...

# Terminal 2: Start reverse port-forward with verbose logging
$ kubectl port-forward --reverse -v=9 nginx 9999:9999

# Terminal 3: Test from within pod
$ kubectl exec -it nginx -- bash
root@nginx:/# curl http://localhost:9999
# Should see HTTP response from local Python server
# Terminal 1 should show: "GET / HTTP/1.1" 200 -

# If this works, the feature is functional - investigate specific application issues
# If this fails, check the error messages from steps above
```

- **Step 8**: Verify pod and network connectivity:

```bash
# Check pod is running and healthy
$ kubectl get pod nginx -o wide
$ kubectl describe pod nginx

# Check kubelet is healthy on the node
$ kubectl get nodes
$ kubectl describe node <node-name>

# Verify API server connectivity
$ kubectl cluster-info
$ curl -k https://<api-server>:6443/healthz
```

- **Step 9**: If all else fails, disable the feature and use forward port-forward
  to verify base functionality:

```bash
# Test with regular forward port-forward to verify base infrastructure works
$ kubectl port-forward nginx 8080:80

# In another terminal
$ curl http://localhost:8080

# If forward port-forward works but reverse doesn't, the issue is specific
# to the reverse port-forward implementation - collect logs and file a bug
```

## Implementation History

- 2016-01-27: Original feature request filed as [#20227](https://github.com/kubernetes/kubernetes/issues/20227)
- 2017-12-18: Initial PoC implementation as [#57320](https://github.com/kubernetes/kubernetes/pull/57320)
- 2025-11-18: KEP created
- 2025-11-19: Test infrastructure prepared in `portforward_test.go` with comprehensive test stubs
- TBD: Alpha implementation targeting v1.34
- TBD: Beta implementation targeting v1.35
- TBD: GA implementation targeting v1.36

## Drawbacks

1. **Security Concerns**: Exposing local services to remote pods requires users to understand security implications
2. **Complexity**: Adds complexity to the port-forward codebase and protocol
3. **Limited Use Case**: Primarily useful for development/debugging, not production scenarios
4. **Alternative Solutions Exist**: Tools like Telepresence provide similar functionality

## Alternatives

### Alternative 1: Third-Party Tools

Users can continue using third-party tools like:
- Telepresence: Replaces pods with proxies
- SSH tunneling with sidecar containers
- VPN solutions

**Rejected because**: These solutions add external dependencies, complexity, and often require cluster modifications. A native kubectl solution is simpler and more integrated.

### Alternative 2: Exec-based Approach

Use `kubectl exec` to run `socat` or `nc` inside the pod and manually create tunnels.

**Rejected because**: This is manual, error-prone, and doesn't provide the seamless experience of port-forward.

### Alternative 3: Service Mesh Integration

Use service mesh features to route traffic from pods to external endpoints.

**Rejected because**: Requires service mesh installation, is overkill for simple debugging scenarios, and is not universally available.

### Alternative 4: API Server Proxy

Extend API server to proxy connections between pods and external endpoints.

**Rejected because**: API server should not be in the data path for debugging tools. This is a kubectl/kubelet concern.

## Infrastructure Needed

No special infrastructure required. The feature builds on existing port-forward infrastructure and protocols.

## References

- Original Issue: https://github.com/kubernetes/kubernetes/issues/20227
- Original PoC: https://github.com/kubernetes/kubernetes/pull/57320
- SSH Reverse Port Forward Documentation: https://man.openbsd.org/ssh#R
- Similar tools for comparison:
  - Telepresence: https://www.telepresence.io/
