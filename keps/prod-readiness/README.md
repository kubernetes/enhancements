# Production Readiness Reviews

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable, can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production.

Production readiness reviews are done by a separate team, apart from the SIG
leads (although SIG lead approval is needed as well, of course). It is useful
to have the viewpoint of a team that is not as familiar with the intimate
details of the SIG, but is familiar with Kubernetes and with operating
Kubernetes in production. Experience through our dry runs in 1.17-1.20 have
shown that this slightly "outsider" view helps identify otherwise missed items.

Full documentation on the production readiness review process can be found on
the SIG Architecture page [here][prod-readiness].

[prod-readiness]: https://git.k8s.io/community/sig-architecture/production-readiness.md
