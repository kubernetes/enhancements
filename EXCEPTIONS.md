# Exceptions to milestone Feature Complete dates

For minor (1.x) milestones, the Kubernetes project has feature complete dates (hitting feature complete is all feature LGTMed and in submit queue with tests written) after which no new features are accepted. Since minor releases come approximately quarterly, missing a feature complete date by just one day can mean that feature takes an additional 3 months to be released.

While the feature complete milestone dates are published well in advance, and the default is that missing the date means your feature will be part of the next milestone, there may be cases where an exception makes sense.

## Criteria for exceptions

Exceptions will be granted on the basis of *risk* and *length of exception required*.

The feature coming in late should represent a **low risk to the Kubernetes system** - it should not risk other areas of the code, and it should itself be well contained and tested.

The length of exception needed should be on the order of days, not weeks. If there are 3 PRs in and 1 still waiting review, that's a much stronger case than a feature that doesn't have any PRs out yet.

## Process to request an exception

To file for an exception, please fill out the questions below:

* Feature name:
* Feature status (alpha/beta/stable):
* SIG:
* Feature issue #:
* PR #’s:
* Additional time needed (in days):
* Reason this feature is critical for this milestone:
* Risks from adding code late: (to k8s stability, testing, etc.)
* Risks from cutting feature: (partial implementation, critical customer usecase, etc.)

Email them to:

* Your SIG lead
* The Release Team Lead
* kubernetes-milestone-burndown@googlegroups.com

[You should have *very high confidence* on the “additional time needed” number - we will not grant multiple exceptions for a feature.]

Requests for exceptions must be submitted before the first milestone burn down meeting. All requests for exception will be reviewed and either approved or rejected during the first meeting.

Important dates for each release and information about the Release Team Lead can be found in the [feature repo](https://github.com/kubernetes/features). For the v1.5 release, for example, see [link](https://github.com/kubernetes/features/blob/master/release-1.5/release-1.5.md).

Once an exception is approved, it should be broadcast broadly: send an email with the data and approval to kubernetes-dev@ and your SIG group, then follow up with a reply to that email once the feature is completed.
