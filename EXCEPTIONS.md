# Exceptions to milestone Feature Complete dates

For minor (1.x) milestones, the Kubernetes project has feature complete dates (hitting feature complete is all feature LGTMed and in submit queue with tests written) after which no new features are accepted. Since minor releases come approximately quarterly, missing a feature complete date by just one day can mean that feature takes an additional 3 months to be released.

While the feature complete milestone dates are published well in advance, and the default is that missing the date means your feature will be part of the next milestone, there may be cases where an exception makes sense.

## Criteria for exceptions

Exceptions will be granted on on the basis of *risk* and *length of exception required*.

The feature coming in late should represent a low risk the Kubernetes system - it should not risk other sections of the code, and it should itself be well contained and tested.

The length of exception needed should be on the order of days, not weeks. If there's 3 PRs in and 1 still waiting review, that's a much stronger case than a feature that doesn't have any PRs out yet.

## Process to request an exception

To get an exception, please fill out this questionnaire and request email to your SIG lead, the release czar and cc: kubernetes-milestone-burndown@.  You should have *very high confidence* on the “additional time needed” number - we will not grant multiple exceptions for a feature.

* Feature name:
* Feature status (alpha/beta/stable):
* SIG:
* Feature issue #:
* PR #’s:
* Additional time needed (in days):
* Reason this feature is critical for this milestone:
* Risks from adding code late: (to k8s stability, testing, etc.)
* Risks from cutting feature: (partial implementation, critical customer usecase, etc.)

Once an exception is approved, it should be broadcast broadly - send an email with the data and approval to kubernetes-dev@, then follow up with a reply off that email once the feature is in.
