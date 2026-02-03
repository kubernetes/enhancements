Title: KEP-NNNN: CSI Direct In-Memory Environment Variable Injection

Summary:
This PR introduces the KEP package for CSI Direct In-Memory Environment Variable Injection and requests assignment of a KEP number.

Why this PR:
- Adds KEP package (README, KEP document, ALTERNATIVES, GRADUATION) describing NodeGetEnvVars CSI extension
- Proposes a secure, batched, UDS-based NodeGetEnvVars RPC for low-latency, in-memory environment variable injection
- Includes metrics, test plan, and graduation criteria

Changes requested from reviewers:
- Assign a KEP number and update filenames (replace NNNN)
- SIG-Storage feedback on design and security model
- SIG-Node feedback on kubelet integration and kubelet feature gate semantics
- SIG-Auth review for RBAC and ProviderClass authoring/validation
- Volunteers to review proposed metrics and help craft alerting

Testing / Implementation notes:
- Prototype implementation planned: fork secrets-store-csi-driver and add NodeGetEnvVars support
- Proposed default kubelet RPC timeout: 5s (configurable)

Next steps upon KEP number assignment:
- Replace NNNN in filenames and internal references
- Submit CSI spec PR proposing NodeGetEnvVars and `ENV_VAR_INJECTION` NodeServiceCapability
- Open a SIG-Storage meeting slot and post to mailing list for review

Files changed:
- `README.md` — package overview and next steps
- `kep.yaml` — metadata & metrics
- `0000-csi-direct-env-injection.md` — KEP document
- `ALTERNATIVES.md` — alternatives analysis
- `GRADUATION.md` — graduation criteria and tests

Maintainers: @assafkatz (author)

/cc @kubernetes/sig-storage-pr-reviews @kubernetes/sig-node-pr-reviews @kubernetes/sig-auth
