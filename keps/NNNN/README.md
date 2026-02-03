# KEP-NNNN: CSI Direct In-Memory Environment Variable Injection

## Package Contents

This directory contains a complete Kubernetes Enhancement Proposal (KEP) package ready for submission to the [kubernetes/enhancements](https://github.com/kubernetes/enhancements) repository.

### Files

1. **kep.yaml** - Metadata file with KEP number, authors, SIG ownership, milestones, feature gates, and detailed metrics specifications
2. **0000-csi-direct-env-injection.md** - Main KEP document with complete design details, production readiness questionnaire, and implementation history
3. **ALTERNATIVES.md** - Detailed analysis of 6 alternative approaches with quantitative comparison matrices
4. **GRADUATION.md** - Comprehensive graduation criteria (Alpha/Beta/GA), test plans, and implementation roadmap

### Status

- **Current Stage**: Provisional (target: Alpha v1.36)
- **Target Alpha**: v1.36 (Q3 2026)
- **Target Beta**: v1.37 (Q1 2027)
- **Target GA**: v1.38-39 (Q3-Q4 2027)

### Next Steps

1. **Get KEP Number**: Submit PR to kubernetes/enhancements to get KEP number assigned (replace `NNNN` in filenames and content)
   - After the KEP number is assigned, update filenames and internal references to use the assigned number (replace `NNNN`), and verify links in `README.md` and `kep.yaml`.

2. **SIG-Storage Review**: 
   - Present at SIG-Storage meeting
   - Post to kubernetes-sig-storage mailing list
   - Request reviewers from `@kubernetes/sig-storage-pr-reviews`

3. **Build Prototype**:
   - Fork [secrets-store-csi-driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver)
   - Implement `NodeGetEnvVars` RPC
   - Demo at SIG-Storage meeting (target: 1 week for PoC)
   - Prototype should demo at least Vault and AWS plugins (2+) and document plugin-specific validation (e.g., `vaultRole` checks) to accelerate SIG buy-in and contributor onboarding
- Include `scripts/kind-scale-demo.sh` for quick local scalability tests (100+ pods) and to validate kubelet/driver behavior under pod churn   - Demo deployment notes: prefer Helm-based installs for reproducibility and use a `ServiceMonitor` or Prometheus sidecar to scrape driver metrics (e.g., `/metrics`) during the demo. See `ALTERNATIVES.md` Demo Deployment & Metrics section for example commands and `examples/migration-init-to-csi.yaml` for a migration example from init container to `csiSecretRef`.   - Target CSI v1.10+
   - Engage CSI maintainers early (critical path for GA)

4.5 **Early CSI Spec Engagement (Q2 2026)**:
   - Open an informal discussion issue in `kubernetes-csi/spec` to gather early feedback and flag design questions.
   - Present the proto draft at a CSI community meeting and iterate on capability hints, limits, and error mapping.
   - Goal: increase confidence in ratification before Alpha and reduce the likelihood of late-stage spec changes.

5. **Get Approvals**:
   - SIG-Storage approvers sign-off
   - SIG-Node approvers sign-off (for kubelet changes)
   - SIG-Auth review (for RBAC model)
   - PRR (Production Readiness Review) approval

6. **Update Status**: Change `status: provisional` → `status: implementable` in kep.yaml after approvals

### Key Contacts

- **Owning SIG**: SIG-Storage
- **Participating SIGs**: SIG-Node, SIG-Auth, SIG-API-Machinery
- **Author**: @assafkatz
- **Reviewers**: TBD (to be assigned via enhancement PR process)
- **Approvers**: TBD (from SIG-Storage and SIG-Node)

### Implementation Timeline

| Milestone | Timeline | Dependencies |
|-----------|----------|--------------|
| Prototype (Secrets Store fork) | Q2 2026 (Week 1) | SIG buy-in |
> If CSI spec ratification is delayed, the prototype may proceed with an experimental mock `NodeGetEnvVars` RPC behind the `CSIDirectEnvInjection` feature gate to validate real-world behavior and gather operator feedback while the canonical proto is iterated on.| Cloud Metadata (AWS IMDS) | Q2 2026 (Week 3) | IMDS client reuse |
| Kubelet Integration | Q2 2026 (Week 8) | Feature gate approval |
| Alpha (v1.36) | Q3 2026 | Kubelet/CRI changes |
| CSI Spec Proposal | Q3 2026 | kubernetes-csi approval |
| Beta (v1.37) | Q1 2027 | ≥2 drivers, Windows support |
| CSI Spec Ratified (v1.10+) | Q4 2026-Q1 2027 | CSI maintainers vote |
| GA (v1.38-39) | Q3-Q4 2027 | Production validation |

### Critical Dependencies

1. **CSI Spec Ratification** (longest pole):
   - Proposal must be submitted by Q3 2026
   - Community review: 2-3 months
   - Ratification: Q4 2026
   - Independent of Kubernetes release cycle

2. **Driver Ecosystem**:
   - Alpha: 1 reference driver (mock or Vault)
   - Beta: ≥2 production drivers (secrets + cloud metadata)
   - GA: ≥3 drivers across all categories

3. **Production Validation**:
   - Beta: ≥100 pods in test environments
   - GA: ≥5000 pods across ≥3 organizations

### References

- [KEP-3721: Support for Environment Variable Files](https://github.com/kubernetes/enhancements/issues/3721)
- [Secrets Store CSI Driver](https://github.com/kubernetes-sigs/secrets-store-csi-driver)
- [CSI Specification](https://github.com/container-storage-interface/spec)
- [Kubernetes Enhancement Proposal Template](https://github.com/kubernetes/enhancements/tree/master/keps/NNNN-kep-template)

### Feedback

For questions or feedback on this KEP:
- GitHub Issues: [kubernetes/enhancements](https://github.com/kubernetes/enhancements/issues)
- Slack: #sig-storage, #csi
- Mailing List: kubernetes-sig-storage@googlegroups.com

---

**Note**: This KEP package has been refined based on community standards with:
- Complete protobuf definitions for CSI RPC
- Detailed kubelet integration pseudocode
- Quantitative comparison matrices with scoring (1-10 scale)
- Concrete benchmark data and citations
- CSI spec v1.10 ratification timeline
- Windows-specific test plans (containerd + CRI-O considerations)
- Quantified GA metrics (<0.01% error rate, p99 ≤100ms)

Ready for submission to kubernetes/enhancements repository.
