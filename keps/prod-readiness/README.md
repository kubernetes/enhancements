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

1. Make sure the Enhancement KEP is labeled `lead-opted-in` before PRR Freeze. This is required so that the Enhancements release team and PRR team are aware the KEP is targeting the release.
2. Create a PRR yaml file under the correct SIG under [enhancements/kep/prod-readiness](https://github.com/kubernetes/enhancements/tree/master/keps/prod-readiness).
3. The PRR team will assign a reviewer to your KEP. You can see the list of reviewers under [prod-readiness-approvers](https://github.com/kubernetes/enhancements/blob/master/OWNERS_ALIASES#L199) for the current release. You can ping the PRR reviewers at `@kubernetes/prod-readiness-reviewers` if you have questions about this process. See the [prod-readiness guide]([prod-readiness]: https://git.k8s.io/community/sig-architecture/production-readiness.md) for more information on the PRR process.