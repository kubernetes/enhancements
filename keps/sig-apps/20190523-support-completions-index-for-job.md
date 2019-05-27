---
title: support-completions-index-for-job
authors:
  - "@fudali"
owning-sig: sig-apps
participating-sigs:
  - sig-job
reviewers:
  - "@kow3ns"
  - "@janetkuo"
approvers:
  - "@kow3ns"
  - "@janetkuo"
editor: TBD
creation-date: 2019-05-23
last-updated: 2019-05-27
status: implementable
see-also:
replaces:
superseded-by:
---

# support completions index for JOB

## Table of Contents

- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)


## Summary

[Job](https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/)
is a Kubernetes CRD that help us run task, Job allows a user to specify the task or parallel task through either `podTemplate` and `completions`; but i use parallel task i must ensure task unrepeat in myself application, this is diffcult; if job support sahrds, i can use shards index to implement task unrepeat is so easy;

## Motivation

I have a case need shards to job, i want to use kubernetes implement my case; i read the [document](https://kubernetes.io/docs/concepts/workloads/controllers/jobs-run-to-completion/#parallel-jobs), this is a not implement function, so i want implement it;

### Goals

- Implement each Pod is passed a different index in the range 1 to .spec.completions.

## Proposal

### Implementation Details/Notes/Constraints

In the current implementation, the Job controller not allocation index on `.completions != nil`; in my implementation, when `.completions != nil` i will through completions value and succeeded pods get the available indexes, i will record index to the job create pods, and set env to the pod （user application get index is easy）; next i will allocation index to creating pod function, i will lock this step ensure not repeat index not more than one, if one index repeat, i will kill one on next this job created pod changes time;

### Risks and Mitigations

The major risk with this change is the additional load on the apiserver since we need frequent each pod for get succeeded index and error index.

## Design Details

### Test Plan

* Unit tests covering the usage of Job with `.completions != nil`.
* Integration tests to make sure using the `.completions != nil` from the PDB controller works as expected.

### Graduation Criteria

This will be added as a alpha enhancement to Job. It doesn't change the existing API or behvior but only adds an additional allocation index to pods.

[KEP](https://github.com/kubernetes/enhancements/pull/1072) for graduating Job to GA is already underway. It involves a change to make Job mutable. [PR](https://github.com/kubernetes/kubernetes/pull/66105) for this is almost ready to merge. The goal is to get both that change and this one into the next version of Kubernetes (1.16), and unless any serious issues come up, promote PDBs to GA the following release (1.17).

## Implementation History

- Initial PR: https://github.com/kubernetes/kubernetes/pull/66105

