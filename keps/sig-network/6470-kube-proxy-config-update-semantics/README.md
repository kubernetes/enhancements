# KEP-6470: Make the kube-proxy configuration file startup-only

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Paced cluster-wide configuration change](#paced-cluster-wide-configuration-change)
    - [External kube-proxy lifecycle manager](#external-kube-proxy-lifecycle-manager)
    - [Configuration rollback](#configuration-rollback)
    - [Mutable ConfigMap consumer](#mutable-configmap-consumer)
  - [Notes, Constraints, and Caveats](#notes-constraints-and-caveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Configuration update cases](#configuration-update-cases)
  - [Compatibility and migration](#compatibility-and-migration)
  - [Test Plan](#test-plan)
  - [Upgrade, Downgrade, and Version Skew](#upgrade-downgrade-and-version-skew)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Preserve and expand filesystem watching](#preserve-and-expand-filesystem-watching)
  - [Implement live in-process reconfiguration](#implement-live-in-process-reconfiguration)
  - [Delegate restarts to a configuration management controller](#delegate-restarts-to-a-configuration-management-controller)
  - [Keep current behavior and document its limits](#keep-current-behavior-and-document-its-limits)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

This KEP is provisional and is not targeting a release. The release checklist
will be completed after SIG Network accepts the proposal and a release
milestone is selected.

- [ ] (R) Enhancement issue in release milestone links to this KEP
- [ ] (R) KEP approvers approve status `implementable`
- [ ] (R) Design details are complete
- [ ] (R) Test plan is in place, with SIG Architecture and SIG Testing input
- [ ] (R) Production readiness review is complete and approved
- [ ] User-facing documentation and release note are prepared
- [ ] Implementation history and supporting links are current

## Summary

This KEP proposes defining a clear contract for changes to the configuration
file consumed by kube-proxy. The current implementation watches the configured
file and exits when it observes selected filesystem events; the process
supervisor then starts a replacement process, which reads configuration at
startup. This is not in-process configuration reload, and the observed result
depends on how the file is updated.

The proposed direction is to make configuration a startup snapshot and require
an explicit, operator-controlled kube-proxy restart to apply changes. This is a
behavior change for update methods that currently trigger process exit, and it
does not make a mutable ConfigMap update take effect. The proposal selects a
direct transition: remove the watcher without a compatibility mode, publish a
release note, and document controlled restart procedures. It is provisional
because SIG Network has not yet accepted the design; review should assess this
specific proposal and either approve it or request concrete revisions.

This change also reverses an existing code-level expectation: upstream's
`TestConfigChange` asserts that a direct write makes kube-proxy's run loop
return an error ([existing test]). That test has been present since 2022, so
removing the watcher is not merely deleting incidental behavior. The test
covers a direct file write, not projected ConfigMap updates. The absence of a
user-facing update contract does not eliminate the compatibility impact of
changing behavior that the repository itself tests.

## Motivation

kube-proxy does not apply a new configuration to its running proxy in place.
The existing file watcher can terminate the process on selected filesystem
events; a supervisor then starts a replacement process, which parses the file.
That behavior is event-triggered process restart, not hot reload.

Evidence in the discussion demonstrates why the contract is unclear rather
than establishing one universal current behavior. On an unmodified Kubernetes
v1.33.1 reproduction, a direct write caused kube-proxy to restart, while an
update through a projected ConfigMap changed the mounted file without
restarting the process ([reproduction and rollout report]). A downstream
distribution also reported maintaining a
polling patch that hashes file contents, validates the new configuration, and
restarts kube-proxy for customers expecting updates to take effect
([downstream report]). These are
concrete reports, but they do not quantify how many deployments depend on each
behavior or establish that either behavior is appropriate as the upstream
contract.

The proposed startup-only semantics therefore do not fix the reported
projected-ConfigMap no-op. They make that outcome explicit and remove the
restart-on-direct-write behavior. That tradeoff must be stated plainly in the
KEP and release documentation. Conversely, an operator who deliberately
changes a versioned ConfigMap reference in a DaemonSet can use the DaemonSet
rollout to pace replacement processes. A three-node reproduction reports
that a rollout with readiness checks and `maxUnavailable: 1` limited an invalid
startup configuration to one failing pod and allowed rollback
([reproduction and rollout report]); this is useful
evidence for that deployment pattern, not a guarantee for every cluster or
proxy mode.

With startup-only semantics, a file edit can leave a node on old configuration
indefinitely unless its kube-proxy process is restarted. Operators who cannot
control the process through a DaemonSet need a restart mechanism in their own
lifecycle manager. The design must make the benefit and this cost explicit and
provide a viable transition.

This proposal is related to [kubernetes/kubernetes#139204],
[kubernetes/kubernetes#132877], and the historical discussion in
[kubernetes/kubernetes#94022]. Those reports motivate review; they do not
establish the frequency or severity of either failure mode across the
ecosystem. [KEP-784] defines kube-proxy's component configuration format and
source, but does not establish a supported file-change/restart contract; it
also records that its e2e framework did not test configuration-file changes.
The [kube-proxy command reference] describes `--config` as a file path but
does not define file-change or restart semantics.

[KEP-784]: /keps/sig-network/784-kube-proxy-component-config
[kube-proxy command reference]: https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/#options

[kubernetes/kubernetes#139204]: https://github.com/kubernetes/kubernetes/pull/139204
[kubernetes/kubernetes#132877]: https://github.com/kubernetes/kubernetes/issues/132877
[kubernetes/kubernetes#94022]: https://github.com/kubernetes/kubernetes/issues/94022
[existing test]: https://github.com/kubernetes/kubernetes/blob/bbf142898f357d8ba4642748afdc4b7d70479021/cmd/kube-proxy/app/server_linux_test.go#L97-L168
[reproduction and rollout report]: https://github.com/kubernetes/kubernetes/pull/139204#issuecomment-5998997887
[downstream report]: https://github.com/kubernetes/kubernetes/pull/139204#issuecomment-6001988032

Kubernetes eventually updates files in a mutable ConfigMap volume after kubelet
sync and cache propagation; that updates the mounted file, not the configuration
already parsed by kube-proxy ([ConfigMap volume update documentation]). A
DaemonSet can pace process replacement when its Pod template changes and its
update strategy is `RollingUpdate`; `maxUnavailable` and readiness settings
control that rollout ([DaemonSet rollout documentation]). These are distinct
mechanisms and must not be conflated.

[ConfigMap volume update documentation]: https://kubernetes.io/docs/tutorials/configuration/updating-configuration-via-a-configmap/
[DaemonSet rollout documentation]: https://kubernetes.io/docs/tasks/manage-daemon/update-daemon-set/
[kubeadm kube-proxy reconfiguration procedure]: https://kubernetes.io/docs/tasks/administer-cluster/kubeadm/kubeadm-reconfigure/

### Goals

- Define that changes to the `--config` file do not affect a running kube-proxy
  process; new contents apply only after process restart.
- Make the behavior deterministic across supported operating systems and
  common file replacement patterns.
- Give operators a supported, paced method to apply configuration changes to
  DaemonSet-managed kube-proxy instances.
- Document behavior for deployments that do not use the Kubernetes
  kube-proxy DaemonSet.
- Give existing deployments clear migration instructions for the change to
  direct-file update behavior.

### Non-Goals

- Implement live, in-process reconfiguration of kube-proxy settings.
- Define generic ConfigMap reload behavior for CoreDNS, kubelet, or other
  components.
- Change the kube-proxy component configuration API or its validation rules.
- Change command-line flag semantics or introduce environment-variable based
  configuration. This proposal concerns only the file supplied through
  `--config` and the process lifecycle response to changes to that file.
- Guarantee that all nodes change configuration atomically. Kubernetes does
  not provide an atomic multi-node process restart.
- Choose a cluster-wide ConfigMap management convention beyond documenting
  supported kube-proxy rollout patterns.

## Proposal

This KEP proposes the following steady-state contract:

1. kube-proxy reads and validates its configuration file during process start.
2. Changes to that file do not cause the running process to exit or reload.
3. An operator applies changed settings by replacing/restarting the kube-proxy
   process using the lifecycle manager for that deployment.
4. Kubernetes-managed deployments should use a versioned or otherwise
   rollout-triggering DaemonSet configuration reference and set rollout
   availability parameters appropriate to their cluster. Merely editing the
   data in a mutable ConfigMap must not be described as a paced rollout.

For kubeadm-managed clusters, follow kubeadm's kube-proxy reconfiguration
procedure: update `data.config.conf` in the `kube-system/kube-proxy`
ConfigMap, then restart the kube-proxy Pods so they read the new contents.
The kubeadm procedure currently instructs operators to delete the Pods; a
controlled DaemonSet rollout is preferable where the DaemonSet update strategy
and availability settings support it. kubeadm may rewrite the addon during
cluster upgrades, so operators must also keep the source kubeadm configuration
consistent with the intended kube-proxy configuration. The versioned ConfigMap
workflow applies to deployments that manage their DaemonSet and configuration
references directly; it is not a prerequisite imposed on kubeadm users.

The implementation will remove the file watcher without adding a feature gate
or preserving a legacy mode. The watcher is not a documented configuration
update API, does not reload the running proxy, and behaves differently for
direct writes and projected ConfigMap updates. Keeping a compatibility mode
would perpetuate that ambiguous contract and add lifecycle cost without making
ConfigMap updates reliable.

This is a discussion draft, not ready to merge until SIG Network identifies a
sponsor and reviews the proposal. Its status is **provisional** because the
project has not accepted it. SIG Network is being asked to accept, reject, or
request specific changes to this design, not to choose among several
unresolved migration policies.

### User Stories

#### Paced cluster-wide configuration change

As a cluster operator, I want to update kube-proxy configuration through a
controlled DaemonSet rollout so that process restarts are paced according to
cluster availability requirements rather than triggered independently by
projected file updates.

#### External kube-proxy lifecycle manager

As an operator running kube-proxy outside the standard DaemonSet, I want to
know that changing a file will not restart the process and to have clear
instructions for applying a change through my supervisor or maintenance
automation.

#### Configuration rollback

As an operator, I want to revert the configuration and restart kube-proxy in a
controlled way, with the same validation and rollout expectations as a
forward change.

#### Mutable ConfigMap consumer

As an operator currently editing a mutable ConfigMap and expecting kube-proxy
to apply changes, I need the release notes and documentation to state plainly
that this proposal does not make that update effective, that direct-write
restart behavior is removed without a compatibility period, and what
replacement workflow is supported.

### Notes, Constraints, and Caveats

- A ConfigMap volume update is not equivalent to a DaemonSet pod-template
  update. The projected files may change without changing the pod template.
  In the cited v1.33.1 reproduction this update did not restart kube-proxy;
  that observation must not be generalized to all kubelet versions, filesystems,
  or deployment-specific patches.
- A ConfigMap mounted using `subPath` does not receive normal projected-volume
  updates ([ConfigMap documentation]). Startup-only behavior makes the
  requirement to recreate/restart the process especially visible in this case.
- Files can be changed in place, replaced by rename, or exposed through
  symlink swaps such as projected volumes. Watching only the configured path
  may not observe all of these consistently.
- Invalid intermediate contents must not trigger a restart loop as an
  accidental consequence of a partial write. Under the proposed contract,
  changed contents are ignored by the running process; validation occurs when
  the operator starts a replacement process.
- Startup-only semantics do not make a concurrent in-place write atomic. If
  kube-proxy starts while a writer is replacing the file, it may read incomplete
  contents and fail validation. Producers should publish a complete file
  atomically or use a new versioned ConfigMap, then restart kube-proxy.
- A replacement process can fail to start if the new configuration is
  invalid. The deployment's restart policy may then leave the node without a
  healthy kube-proxy process to reconcile new Services or endpoints. Existing
  dataplane rules may remain; exact traffic impact depends on the proxy mode
  and failure point. A rollback procedure and health checks are needed; this
  KEP cannot promise zero dataplane impact during restart.
- Existing kernel rules may persist, partially change, or be reconciled as a
  process starts, depending on proxy mode and failure point. The design must
  not promise a uniform dataplane state during termination or failed startup
  without evidence from mode-specific tests.
- A paced DaemonSet rollout reduces simultaneous restarts but does not make the
  change atomic across nodes. Users must account for mixed configuration during
  rollout, especially where settings need cluster-wide coordination.
- If configuration is delivered through flags, generated files, host mounts,
  or a third-party supervisor, the restart mechanism belongs to that deployment
  system. kube-proxy cannot infer the operator's desired rollout policy.

[ConfigMap documentation]: https://kubernetes.io/docs/concepts/configuration/configmap/#mounted-configmaps-are-updated-automatically

### Risks and Mitigations

| Risk | Consequence | Mitigation / evidence required |
| --- | --- | --- |
| A user expects a mutable ConfigMap edit to apply new settings | Under the cited reproduction, the existing watcher did not restart kube-proxy for this update; the proposed behavior makes the need for a restart explicit | State that this KEP does not implement projected-ConfigMap reload; document the versioned ConfigMap and DaemonSet rollout path |
| A user relies on direct file writes to trigger a process restart | The proposed change removes an observable behavior and leaves the process on its startup configuration | Call out the behavior change in release notes and documentation; document how external lifecycle managers can validate and restart kube-proxy |
| The same file update event is observed by kube-proxy processes on many nodes | Process restarts may occur outside rollout control; new or changed Services may not be reconciled while a replacement process is unhealthy | Remove event-triggered exits and use explicit DaemonSet rollout pacing; validate this path in multi-node tests and state that the rollout is not atomic |
| A rollout loads invalid configuration | Replacement processes fail; existing dataplane rules may remain while kube-proxy cannot reconcile subsequent changes | Validate before rollout where possible, surface startup errors, and document rollback; do not claim automatic rollback or uniform dataplane behavior |
| Startup-only behavior is mistaken for hot reload | Operator assumes settings took effect when they did not | State plainly that file changes are ignored until process restart; add appropriate startup and rollout documentation |
| Different filesystems deliver different event sequences | Existing restart behavior differs by platform and volume type | Verify the startup-only contract on every supported platform; do not make the new behavior depend on event delivery |

## Design Details

### Configuration update cases

The behavior and user-visible result must be specified for each case below.

| Update case | Existing behavior or documented expectation | Proposed steady-state behavior | Required test or documentation |
| --- | --- | --- | --- |
| Direct in-place write | A direct write caused a restart in the cited v1.33.1 reproduction; a partial write can expose invalid contents | Running process keeps its startup snapshot | Process test shows no exit; docs require publishing a complete file before an intentional restart |
| Atomic rename over the configured path | Watcher may see rename, subject to OS/filesystem behavior | Running process keeps its startup snapshot | Process test for rename; restart reads the replacement file |
| Mutable ConfigMap projected volume | One v1.33.1 reproduction changed the mounted file without restarting kube-proxy; behavior may vary with implementation or downstream patches | Running process keeps its startup snapshot | Integration test with projected-volume layout; docs distinguish data update from DaemonSet rollout |
| ConfigMap mounted with `subPath` | Mounted file may not reflect ConfigMap changes | Running process keeps its startup snapshot | Docs explain that a new pod/process is required to consume new contents |
| Immutable/versioned ConfigMap plus DaemonSet template update | Pod template change can initiate a managed rollout | Replacement processes read the referenced version | Integration/e2e coverage of rollout and readiness, including configured availability limits |
| Invalid new configuration | A watcher-triggered restart can fail during startup | Existing process continues; a deliberate restart can fail validation | Test that file change alone does not terminate process; document validate-and-rollback sequence |
| Configuration file removed or temporarily unavailable | Watcher behavior may depend on event type; later startup may fail | Existing process continues; a restart fails if the file is absent | Tests for removal and startup error; operator recovery guidance |
| Process restarted for unrelated reasons | New process reads whatever file is currently present | New process reads and validates current file | Test initial load and document that an unrelated restart may apply a previously staged file |

The table captures intended semantics, not claims about every current platform.
The baseline behavior must be measured against the supported kube-proxy build
and volume implementations before claiming a compatibility boundary.

### Compatibility and migration

This proposal intentionally changes the observable behavior of direct file
updates that trigger process exit in the cited v1.33.1 reproduction. It does
not claim that removing the watcher regresses projected ConfigMap updates: the
cited v1.33.1 reproduction already showed that kube-proxy did not restart for
that update. Separate downstream patches that poll and restart on valid content
changes are evidence of deployments that expect automatic application, but
they do not establish a portable upstream contract.

The migration policy is a direct change in the first release that includes
this KEP's implementation:

1. Remove the file watcher and do not add a feature gate or legacy mode.
2. Add a release note stating that changes to the `--config` file do not restart
   kube-proxy, and that a process restart is required to apply them.
3. For deployments whose operators manage the kube-proxy DaemonSet and its
   configuration references directly, document the versioned ConfigMap path:
   publish configuration under a new, immutable or versioned ConfigMap name,
   update the DaemonSet Pod template to reference it, and use `RollingUpdate`
   availability and readiness settings appropriate to the cluster. Keep the
   previous ConfigMap version available until the rollout succeeds so the
   DaemonSet can be rolled back.
4. For kubeadm-managed clusters, follow the kubeadm reconfiguration procedure:
   update the `kube-system/kube-proxy` ConfigMap's `data.config.conf` and
   restart kube-proxy Pods to apply it. Where using a DaemonSet RollingUpdate,
   configure and verify the availability and readiness behavior; keep the
   kubeadm source configuration aligned so upgrades do not restore stale
   settings. This procedure is distinct from changing ConfigMap data alone,
   which does not restart a running kube-proxy process.
5. Document the external-manager path: publish a complete file, validate it
   with tooling compatible with the target kube-proxy version where available,
   then restart kube-proxy through that manager. A manager that requires
   automatic application may watch revisions and trigger a restart only after
   validation; this remains outside kube-proxy.
6. State that rollback to a Kubernetes release containing the watcher restores
   its previous event-triggered behavior. A mixed-version rollout can therefore
   have different behavior until all kube-proxy instances are upgraded or
   rolled back.

There is no deprecation release or compatibility switch. The watcher is not a
hot-reload implementation and does not reliably observe projected ConfigMap
updates, but its direct-write process-exit behavior is asserted by an existing
upstream test. This KEP explicitly proposes changing that tested behavior. A
transition switch would preserve two runtime behaviors and require eventual
removal; the proposal instead accepts the compatibility cost, calls it out in
release notes, and provides replacement workflows.

### Test Plan

These tests are required before implementation is considered complete. Exact
test locations and release targeting will be added after SIG Network accepts
the design.

- Replace the existing direct-write test expectation explicitly: a changed
  config file must leave the running process alive, and a newly started process
  must load the new contents. Preserve separate tests for proxy-server error
  propagation.
- Unit tests: configuration changes by write, rename, removal, invalid content,
  and unchanged content do not terminate a running kube-proxy process; a new
  process reads and validates the latest file contents.
- Process-level tests on every supported operating system: direct writes,
  replacement, and removal do not terminate a running kube-proxy; a new process
  reads and validates the current file. Where file update mechanisms differ by
  operating system, exercise the native mechanism rather than assuming Linux
  filesystem semantics.
- Linux integration test: use a projected ConfigMap volume layout (symlink
  indirection and atomic target swap) to verify that the running process
  remains alive and a restarted process sees the updated contents.
- Integration test: an invalid staged configuration does not affect the
  running process, while a deliberate restart reports validation failure.
- Multi-node e2e or equivalent rollout test: change a versioned config
  reference and verify that a DaemonSet applies the change according to its
  rollout availability settings, with readiness checks preventing premature
  progression where configured. Ask SIG Apps to review the DaemonSet rollout
  assumptions and test interpretation.
- Regression coverage: cover `subPath`, direct local files, and process
  restarts unrelated to configuration changes in documentation or tests where
  the environment permits.
- Test mixed-version rollout and rollback, confirming the documented old/new
  behavior while both versions may run in the cluster.

Tests must avoid asserting uninterrupted dataplane behavior unless that is
measured and guaranteed for the proxy mode under test.

### Upgrade, Downgrade, and Version Skew

During a rolling upgrade, old kube-proxy processes may still exit on file
events while new ones ignore them. Operators must not assume all nodes share
one update contract until the rollout completes. A rollback to a release with the old watcher restores its
previous file-event behavior; it does not undo configuration already consumed
by processes. DaemonSet rollout and rollback guidance must describe this mixed
version window. No API object or serialized configuration format changes are
proposed.

## Production Readiness Review Questionnaire

This section is intentionally incomplete while the KEP is provisional and is
not being targeted to a release.

- **Feature enablement and rollback:** No API or feature gate is introduced.
  Rollback means rolling back the DaemonSet configuration and, if necessary,
  the kube-proxy version; a code rollback restores the older watch behavior.
- **Rollout and recovery:** Document rollout pacing, readiness checks,
  invalid-configuration recovery, and the fact that multi-node rollout is not
  atomic.
- **Monitoring:** No new metric or API status is proposed. Operators use the
  existing kube-proxy startup logs and health checks to detect a failed restart.
- **Dependencies and scalability:** No new API calls, API types, or periodic
  control-plane activity are introduced.
- **Troubleshooting:** Document how to determine which configuration a process
  loaded, how to trigger a restart through the deployment's lifecycle manager,
  and how to recover from an invalid configuration.

## Drawbacks

- Startup-only semantics can turn a previously self-acting configuration edit
  into a silent no-op until an operator or automation restarts kube-proxy.
- Operators without a DaemonSet rollout mechanism must build or use their own
  controlled restart process.
- A deliberate restart still has node-level disruption and configuration
  validation failure modes; this proposal changes who controls the timing, not
  the consequences of a failed process start.
- There is no separate legacy mode to restore the old direct-write behavior;
  restoring it requires rolling back to a kube-proxy release that contains the
  watcher.

## Alternatives

### Preserve and expand filesystem watching

Watch parent directories and symlink targets, debounce events, hash file
contents, and restart only after a valid configuration is observed. This could
make updates more reliable for individual nodes, but it still does not provide
cluster-wide pacing, is sensitive to filesystem semantics, and requires a
policy for partial writes, invalid files, replacement races, and repeated
events. This KEP rejects this alternative because reliable per-node detection
still cannot provide paced cluster-wide rollout and would preserve the
filesystem-triggered process lifecycle that the proposal removes.

### Implement live in-process reconfiguration

Reconfigure the proxy without process restart. This avoids process lifecycle
events but is a much larger design: configuration fields may have different
reconciliation semantics, rollback behavior, and dataplane transition risks.
It is outside this KEP and would require its own proposal.

### Delegate restarts to a configuration management controller

Use a controller or sidecar to compare configuration revisions and restart
pods/processes. This moves rollout logic out of kube-proxy but adds another
component and does not by itself define the supported kube-proxy contract.
There is no such dependency in this proposal.

### Keep current behavior and document its limits

Retain the existing watcher and document that events can cause a process
restart on some filesystems. This minimizes immediate behavior change but
preserves ambiguity and does not give operators a portable promise. This KEP
rejects this alternative because documentation alone cannot make event delivery
consistent or provide operators with control over restart timing.

## Implementation History

- 2026-10-19: Provisional draft prepared for SIG Network review, tracking
  [enhancements issue #6470].

[enhancements issue #6470]: https://github.com/kubernetes/enhancements/issues/6470
