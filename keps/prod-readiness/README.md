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

**Note:** The kubeadm tool is considered out-of-tree and does not fall under PRR.
If the kubeadm maintainers have to, they could still ask the PRR team for advisory on
a particular production related topic.

[prod-readiness]: https://git.k8s.io/community/sig-architecture/production-readiness.md
