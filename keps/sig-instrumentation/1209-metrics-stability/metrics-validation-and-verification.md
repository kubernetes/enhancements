 # Metrics Validation and Verification

 ## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Static Analysis](#static-analysis)
  - [Discovery of golang source files](#discovery-of-golang-source-files)
  - [Format of defining stable metrics](#format-of-defining-stable-metrics)
  - [Stable metric detection strategy](#stable-metric-detection-strategy)
  - [Format of stable metrics list file](#format-of-stable-metrics-list-file)
  - [Failure Modes](#failure-modes)
  - [Performance](#performance)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Test Plan](#test-plan)
- [Unresolved questions](#unresolved-questions)
  - [Should code snippets used for unit testing be kept in separate files instead of strings?](#should-code-snippets-used-for-unit-testing-be-kept-in-separate-files-instead-of-strings)
<!-- /toc -->

## Summary

This Kubernetes Enhancement Proposal (KEP) builds off of the framework proposed
in the [Kubernetes Control-Plane Metrics Stability KEP](./kubernetes-control-plane-metrics-stability.md)
and proposes a strategy for ensuring conformance of metrics with official
stability guarantees.

## Motivation

While the [Kubernetes Control Plane metrics stability KEP](./kubernetes-control-plane-metrics-stability.md)
provides a framework to define stability levels for control-plane metrics,
it does not provide a strategy for verifying and validating conformance to stated guarantees.
This KEP intends to propose a framework for validating and verifying metric guarantees.

### Goals

* Given a stable metric, validate that we cannot remove or modify it (other than adding deprecation information).
* Given a deprecated but stable metric, validate that we cannot remove or modify it until the deprecation period has elapsed.
* Given an alpha metric which is promoted to be 'stable', automatically include proper instrumentation reviewers
  (for schema validation and conformance to metrics guidelines).
* [Stretch] Provide a single source of stable metric documentation (thanks @ehashman for proposing it)

### Non-Goals

* Conformance testing will not apply to alpha metrics, since alpha metrics do not have stability guarantees.

## Proposal

We will provide validation for metrics under the [new framework](./kubernetes-control-plane-metrics-stability.md) with static analysis.

## Design Details

Stable metrics testing will work in a similar (but not identical) fashion to the generic Kubernetes conformance tests.
Sig-instrumentation will own a directory under `test/instrumentation`.
There will be a subdirectory `testdata` in which a file `stable-metrics-list.txt` will live.
This file will be owned by sig-instrumentation.
Metrics conformance tests will involve a static analysis script which will traverse the entire codebase and look for
metrics which are annotated as 'stable'.
For each stable metrics, this script will generate a stringified version of metric metadata (i.e. name, type, labels)
which will then be appended together in lexographic order. This will be the output of this script.

We will add a pre-submit check, which will run in our CI pipeline, which will run our script with the current changes
and compare that to existing, committed file. If there is a difference, the pre-submit check will fail.
In order to pass the pre-submit check, the original submitter of the PR will have to run a script `
test/instrumentation/update-stable-metrics.sh` which will run our static analysis code and overwrite `stable-metrics-list.yaml`.
This will cause `sig-instrumentation` to be tagged for approvals on the PR (since they own that file).


### Static Analysis

Similarly to conformance test proposed analysis will be performed on golang source code using default abstract syntax tree parser `go/ast`.
 Handling all possible cases for how metrics can be instantiated would require executing the code itself and is not practical.
 To reduce number of cases needed handled we will be making assumption of non malicious intent of contributors.
 There are just too many ways one golang struct can be instantiated that would hide potential stable metrics.
 To ensure correctness and code simplicity we will be restricting how stable metrics can be declared to one format described below.
 Alpha metrics will not be analyzed as they don't have any stability guarantees.
 Their declaration will not have any restrictions, thus allowing dynamic generation.

### Discovery of golang source files

Stable metric list will be a bazel genrule provided with locations of files from `//:all-src` target.
This target is auto-managed, validate in CI by `./hack/validate-bazel.sh` and should include all files in repository.
 Golang source files will be filtered by extension.

List of skipped directories:
* `vendor/` - Kubernetes metrics from external repos will be shared metrics that will be defined in k/k and injected as
              dependency during registration

### Format of defining stable metrics

To simplify static analysis implementation and reduce chance of missing metrics, we will restrict how stable metrics can be defined.
Stable metrics will use exactly the same functions as normal ones, but code defining them will need to comply to the specific format:

* Metric options (e.g. `CounterOpts`) needs to be directly passed to function (e.g. 1`kubemetrics.NewCounterVec`).
* Metric arguments can only be set to values. Fields cannot be set to const, variables or function result (apart of `kubemetrics.STABLE`).
```go
var someCounterVecMetric = kubemetrics.NewCounterVec(
  &prometheus.CounterOpts{
    Name:           "some_metric",
    Help:           "some description",
    StabilityLevel: kubemetrics.STABLE,
  },
  []string{"some-label", "other-label"},
}
```
Package name `kubemetrics` can be replaced with local name of framework import.

Those restrictions will allow AST based analysis to correctly interpret metric definitions.

### Stable metric detection strategy

Static analysis will be done in two steps. First find stable metrics, second read and validate their fields.

In first one step we will want to distinguish stable metric without relying on their definition structure, allowing
freedom of declaration for non-stable metrics. This will be done by finding occurrences of metric options object initialization (`CounterOpts` for `Counter`).
If this initialization sets `StabilityLevel` then it will validate and read it.
If value could not be extracted then script should fail.
Metrics which set `kubemetrics.STABLE` will be passed to second step.

Second step will be extracting information from stable metric definitions.
This step will be expecting exact structure of AST subtree of new metric call matching format proposed above.
If metric options object was initialized outside of new metric call or call structure deviates from expected format then
static analysis should fail. This restriction will ensure that no field was read incorrectly or stable metric missed.

### Format of stable metrics list file

This file will include all metric metadata that will serve source of documentation and a way to detect breaking change.
Every change to metric definition will require updating this file, thus a review from sig-instrumentation member.
It will be up to reviewer to decide which changes are backward compatible and which not.

Metric information stored in file should be in easly readable and reviewable format.
For that `yaml` will be used (thanks for suggestion @ehashman). Example:
```yaml
- deprecatedVersion: "1.16"
  help: "some help information about the metric"
  labels:
    - some-label
  name: some_metric
  namespace: some_namespace
  stabilityLevel: STABLE
  objectives:
    0.5: 0.05
    0.9: 0.01
    0.99: 0.001
  type: Summary
```

Json keys and lists will be sorted alphabetically to ensure that there is only one correct output.

### Failure Modes

In cases where static analysis could not be done correctly (invalid stable metric format, etc.) script should
fail (non zero response code) and write to STDERR information on problem. Example:
```
pkg/controller/volume/persistentvolume/scheduler_bind_cache_metrics.go:42 stable metric CounterOpts should be invoked from newCounter
pkg/controller/volume/persistentvolume/scheduler_bind_cache_metrics.go:44 stable metric CounterOpts should be invoked from newCounter
exit status 1
```

### Performance

Proposed static analysis takes around 4 seconds on one CPU core.
This time is comparable to conformance tests check ~2 sec and should be acceptable.

## Graduation Criteria

- [ ] Stable metric validation is enabled as pre-submit check

## Implementation History

- [ ] Implement static analysis script analysis all metrics types

## Test Plan

- [ ] Static analysis is covered by unit tests using common patterns of metric definition

## Unresolved questions

### Should code snippets used for unit testing be kept in separate files instead of strings?

Input for static analysis unit tests will be Go code.
It could be beneficial to have them as separate files so they would be readable and interpretable by an IDE.
Problem is that when put as separate files they would get BUILD file generated by gazelle and will no longer available
as `data` for `go_test` rule. Possible ways to avoid that:
* Change file extension. Using non standard file extension would lose benefit of separate files.
* Excluding them in gazelle.
