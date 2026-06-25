Title: proposals: Add NodeGetEnvVars RPC to CSI Node Service (ENV_VAR_INJECTION capability)

Summary:
This draft proposes adding a new Node RPC `NodeGetEnvVars` and a NodeServiceCapability enum `ENV_VAR_INJECTION` to the CSI spec (target CSI v1.10+). The RPC returns in-memory key/value results for a set of queries batched by the kubelet and called over the Node UDS.

Draft Proto Diff (conceptual):

service Node {
  // existing RPCs...
  rpc NodeGetEnvVars(NodeGetEnvVarsRequest) returns (NodeGetEnvVarsResponse) {}
}

message NodeGetEnvVarsRequest {
  string provider_class_name = 1;
  map<string, string> class_parameters = 2;
  repeated EnvVarQuery queries = 3;
  map<string, string> pod_context = 4; // namespace, serviceAccount, podName, podUID
  string version = 5;
}

message EnvVarQuery {
  string name = 1;
  string key = 2;
  map<string, string> query_parameters = 3;
  bool optional = 4;
}

message NodeGetEnvVarsResponse {
  repeated EnvVarResult results = 1;
}

message EnvVarResult {
  string name = 1;
  string value = 2; // empty if error set
  string error = 3;
}

message NodeServiceCapability {
  message RPC {
    enum Type {
      UNKNOWN = 0;
      ...
      ENV_VAR_INJECTION = 10;
    }
    Type type = 1;
  }

  // Optional per-driver metadata/hints for environment variable injection.
  // Drivers SHOULD advertise these values when they are authoritative. Kubelet
  // will prefer driver-advertised hints where present and fall back to
  // ProviderClass parameters or kubelet defaults when absent.
  message EnvVarInjectionCapability {
    // Maximum total payload size (bytes) a driver accepts for NodeGetEnvVarsRequest
    uint32 max_total_payload_bytes = 1;
    // Maximum per-variable value length (bytes)
    uint32 max_value_length_bytes = 2;
    // Preferred RPC timeout (seconds) the driver recommends for NodeGetEnvVars
    uint32 preferred_rpc_timeout_seconds = 3;
    // Maximum concurrent in-flight NodeGetEnvVars requests driver supports
    uint32 max_inflight_requests = 4;
    // Optional human-readable notes
    string description = 5;
  }

  oneof type {
    RPC rpc = 1;
    EnvVarInjectionCapability env_var_injection = 2;
  }
}

Notes & Considerations:
- Drivers advertise these hints via GetPluginCapabilities (include EnvVarInjectionCapability) or an equivalent capability metadata endpoint.
- Kubelet SHOULD prefer driver-advertised hints -> ProviderClass parameters -> kubelet defaults (in that order). If kubelet uses a fallback, it SHOULD emit a fallback metric (see KEP metrics) so operators can monitor servers that lack driver metadata.
- Add limits: max payload (64KB), per-variable max (16KB), UTF-8 encoding, default RPC timeout 5s.
- Error mapping and idempotency semantics should be documented in the spec text.
- Proposal will be posted to the kubernetes-csi/spec repo for community review once the KEP is assigned a number.
- Proposal will include example GetPluginCapabilities responses showing the EnvVarInjectionCapability fields set.
- Next steps:
- Open initial PR in container-storage-interface/spec with proto changes + rationale
- Engage CSI maintainers early and iterate on wording for limits and error mapping


Notes & Considerations:
- Add limits: max payload (64KB), per-variable max (16KB), UTF-8 encoding, default RPC timeout 5s.
- Error mapping and idempotency semantics should be documented in the spec text.
- Proposal will be posted to the kubernetes-csi/spec repo for community review once the KEP is assigned a number.

Next steps:
- Open initial PR in container-storage-interface/spec with proto changes + rationale
  - When opening the CSI proto PR, reference this KEP (KEP-NNNN) for implementation context and example kubelet behavior. If a KEP number has not yet been assigned, reference the repository path and update the PR once a number is assigned.
- Engage CSI maintainers early and iterate on wording for limits and error mapping

**Prototype contingency:** If CSI spec ratification is delayed, maintainers and implementers are encouraged to prototype an experimental, opt-in implementation in Kubernetes (behind the `CSIDirectEnvInjection` feature gate) using a mock NodeGetEnvVars RPC for Alpha/Beta testing. This accelerates operational feedback while the CSI spec process proceeds in parallel.
