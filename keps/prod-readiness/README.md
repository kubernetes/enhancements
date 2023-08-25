# Production Readiness Reviews

Production readiness reviews (PRRs) are intended to ensure that enhancements
merging into Kubernetes are observable, scalable and supportable, can be safely
operated in production environments, and can be disabled or rolled back in the
event they cause increased failures in production.

While approvals for an enhancement are still required from the respective SIG,
production readiness reviews are done by the [Production Readiness subproject][prod-readiness] of SIG Architecture.

It is useful to have the viewpoint of a team that is not as familiar with the
intimate details of the SIG, but is familiar with Kubernetes and with operating
Kubernetes in production.

Experience through our dry runs in the Kubernetes 1.17 - 1.20 release cycles
has shown that this slightly "outsider" view helps identify otherwise missed
items.

Full documentation on the production readiness review process can be found on
the SIG Architecture page [here][prod-readiness].

[prod-readiness]: https://git.k8s.io/community/sig-architecture/production-readiness.md

## How do I find a PRR reviewer for my KEP?

The KEP template production readiness questionnaire should be filled out by the KEP authors, and this will be reviewed by the SIG leads. Once the leads are satisfied with both the overall KEP (i.e., it is ready to move to `implementable` state) and the PRR answers, the authors may request PRR approval.
Make sure the `kep.yaml` and README are ready for review and have the correct sections filled out. See the [KEP template](https://github.com/kubernetes/enhancements/tree/master/keps/NNNN-kep-template). 

1. Make sure the Enhancement KEP is labeled `lead-opted-in` before PRR Freeze. This is required so that the Enhancements release team and PRR team are aware the KEP is targeting the release.
2. Assign a PRR approver from the prod-readiness-approvers list in the [OWNERS_ALIASES](https://github.com/kubernetes/enhancements/blob/master/OWNERS_ALIASES#L199) file. This may be done earlier as well, to get early feedback or just to let the approver know. Reach out on the #prod-readiness Slack channel or just pick someone from the list. The team may rebalance the assignees if necessary.
3. Create a `prod-readiness/<sig>/<KEP number>.yaml` file, with the PRR approver's GitHub handle for the specific stage file under the correct SIG under [enhancements/kep/prod-readiness](https://github.com/kubernetes/enhancements/tree/master/keps/prod-readiness). (See this [example PRR approval request PR](https://github.com/kubernetes/enhancements/pull/2179/files).)

See [submitting a KEP for production readiness approval](https://github.com/kubernetes/community/blob/master/sig-architecture/production-readiness.md#submitting-a-kep-for-production-readiness-approval) for more information on the PRR process.
