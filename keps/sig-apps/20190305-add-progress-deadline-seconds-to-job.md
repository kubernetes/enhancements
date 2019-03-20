---
title: Add ProgressDeadlineSeconds to Job
authors:
  - "@goodluckbot"
owning-sig: sig-apps
participating-sigs:
  - sig-api-machinery
reviewers:
  - "@soltysh"
  - "@erictune"
  - "@janetkuo"
approvers:
  - "@soltysh"
editor: TBD
creation-date: 2019-03-05
last-updated: 2019-03-05
status: implementable
see-also:
  - n/a
replaces:
  - n/a
superseded-by:
  - n/a
---


# Add ProgressDeadlineSeconds to Job

## Table of Contents

   * [TTL After Finished Controller](#ttl-after-finished-controller)
      * [Table of Contents](#table-of-contents)
      * [Summary](#summary)
      * [Motivation](#motivation)
         * [Goals](#goals)
         * [Non-Goals](#non-goals)
      * [Proposal](#proposal)
         * [User Stories](#user-stories)
         * [Detailed Design](#detailed-design)
            * [Feature Gate](#feature-gate)
            * [API Object](#api-object)
         * [Implementation Details/Notes/Constraints](#implementation-details/notes/constraints)
            * [Implementation Details](#Implementation Details)
            * [Validation](#Validation)
         * [Risks and Mitigations](#risks-and-mitigations)
      * [Graduation Criteria](#graduation-criteria)
      * [Implementation History](#implementation-history)


## Summary

We propose to add an optinal field `ProgressDeadlineSeconds` to `job`. Like `deployment`'s `ProgressDeadlineSeconds`, if this field is set, it is used to fail a `job` if the `job` does not start within the given time, i.e, the progress deadline.

## Motivation

In [#48075][] and [#51153][] we've introduced backoff policy for job, which does not cover certain issues with erroneous jobs. 
For example, we create a job with miss-typed pull spec, and will find out about it only when the job controller kicks a pod and the pod will result in ImagePullBackOff state. The job will remain active.
Users would like to fail the job instead of having the job remain active if there's a typo, config error, not-pullable image etc.
For detailed examples, please see the below user stories.

[#48075]: https://github.com/kubernetes/kubernetes/issues/48075
[#51153]: https://github.com/kubernetes/kubernetes/issues/51153

### Goals

We'd like to fail a job if it does not start within given time.

### Non-Goals

N/A

## Proposal

### User Stories

* [#67828][] `Jobs with a container config error will never complete.` A job with a config error (pod: Warning Failed 22m (x8 over 24m) kubelet, minikube Error: container has runAsNonRoot and image will run as root), will never complete, the job will not have any success or fail status, and will have an 'activate' forever. While this should count as a failure, and take into account restartPolicy and backoffLimit.
* [#62816][] `Request for job failure on pod failure rather than job remaining active.` 
I run a job with a volume that references a non-existent config map and get an error message like "MountVolume.SetUp failed for volume "some-volume" : configmaps "some-missing-config-map" not found". I expect the status returned by an API call to have "failed": 1.
Would also like the job to fail if the container image can't be pulled. Would be nice to be able to specify in the job spec how many image pull attempts to make before failing. If I have a typo in the image name, or it's not accessible in my Docker registry, I want the job to fail so I can find that out immediately at job submission/creation time, without having to occasionally look through the job list in the dashboard to find ones with the failures recorded in their details.

[#67828]: https://github.com/kubernetes/kubernetes/issues/67828
[#62816]: https://github.com/kubernetes/kubernetes/issues/62816

### Detailed Design 

We will add the following API fields to `JobSpec` (`Job`'s `.spec`).

```go
type JobSpec struct {
     // Optional duration in seconds relative to the startTime that the job needs to become progressive,
     // otherwise the system will try to terminate it; value must be positive integer
     // Job is `progressive` means at least one of the pods starts running or has finished (Completed or Failed),
     // i.e., as long as one of the pods is making some progress, this Job should not be terminated.
     // +optional
     ProgressDeadlineSeconds *int64
}
```

Job is `active` means all pods of the job are in `running` state.

### Implementation Details/Notes/Constraints

#### Implementation Details
We'll check the job status, and if the following three criterions are met, the job is considered failure.
*  job has not retried, i.e, the job is on its first run
*  job has pending pods, i.e, not all pods are running
*  the time from job's startTime until the moment the check if performed has exceeded `*job.Spec.ProgressDeadlineSeconds`

#### Validation
Need to check whether `*job.Spec.ProgressDeadlineSeconds` is a non-negative number.

### Risks and Mitigations

Same as [#KEP-26][], we may have below risks:

Risks:
* Time skew may cause Job controller to mark a Job as failed inaccurately

Mitigations:
* In Kubernetes, it's required to run NTP on all nodes ([#6159][]) to avoid time
  skew. We will also document this risk.

[#KEP-26]:https://github.com/kubernetes/enhancements/blob/master/keps/sig-apps/0026-ttl-after-finish.md
[#6159]: https://github.com/kubernetes/kubernetes/issues/6159#issuecomment-93844058

## Graduation Criteria

N/A

## Implementation History

TBD
