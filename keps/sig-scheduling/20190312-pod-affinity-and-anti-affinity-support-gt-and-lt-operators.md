---
title: Pod affinity/anti-affinity supports Gt and Lt operators
authors:
  - "@wgliang"
owning-sig: sig-scheduling
reviewers:
  - "@bsalamat"
  - "@k82cn"
  - "@Huang-Wei"
approvers:
  - "@bsalamat"
  - "@k82cn"
creation-date: 2019-02-22
last-updated: 2019-04-23
status: provisional
---

 # Pod affinity/anti-affinity supports Gt and Lt operators

 ## Table of Contents

<!-- toc -->
- [Correctness Tests](#correctness-tests)
- [Integration Tests](#integration-tests)
- [Performance Tests](#performance-tests)
- [E2E tests](#e2e-tests)
<!-- /toc -->

#### Correctness Tests
Here is a list of unit tests for various modules of the feature:

- `NodeSelector` related tests
- `PodSelector` related tests
- `Gt` and `Lt` functional tests
- `NumericAwareSelectorRequirement` related tests
- Backwards compatibility - pods made with the new types should still be updatable if apiserver version is rolled back
- Forwards compatibility - all pods created today are wire-compatible (both proto and json) with the new api

#### Integration Tests
- Integration tests for `PodSelector`

#### Performance Tests
- Performance test of `Gt` and `Lt` operators

#### E2E tests
- End to end tests for `PodSelector`

 ### Graduation Criteria

 _To be filled until targeted at a release._

 ## Implementation History

 - 2019-03-12: Initial KEP sent out for reviewing.