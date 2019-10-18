---
title: kubernetes-extensions-suite-for-verifying-optional-components  (CSI Verification Test Suite)
authors:
  - "@brahmaroutu"
  - "@bradtopol"
owning-sig: sig-architecture, sig-storage
participating-sigs:
  - sig-storage
  - sig-architecture
reviewers:
  - "@timothysc"
  - "@johnbelamaric"
approvers:
  - "@bgrant0607"
  - "@msau42"
editor: "@brahmaroutu"
creation-date: 2019-10-16
last-updated: 2019-10-16
status: implementable
---

# kubernetes-extensions-suite-for-verifying-optional-components (CSI Verification Test Suite)

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Representation of Behaviors](#representation-of-behaviors)
  - [Behavior and Test Generation Tooling](#behavior-and-test-generation-tooling)
     - [Handwritten Behaviour Scenarios](#handwritten-behaviour-scenarios)
  - [Coverage Tooling](#coverage-tooling)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Phase 1](#phase-1)
     - [Tying tests back to behaviors](#tying-tests-back-to-behaviors)
     - [kubetestgen](#kubetestgen)
  - [Phase 2](#phase-2)
  - [Graduation Criteria](#graduation-criteria)
  - [Future development](#future-development)
     - [Complex Storytelling combined with json/yaml](#complex-storytelling-combined-with-jsonyaml)
     - [Example patch test scenario](#example-patch-test-scenario)
     - [Generating scaffolding from Gherkin .feature files](#generating-scaffolding-from-gherkin-feature-files)
     - [Autogeneration of Test Scaffolding](#autogeneration-of-test-scaffolding)
     - [Combining gherkin with existing framework](#combining-gherkin-with-existing-framework)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Annotate test files with behaviors](#annotate-test-files-with-behaviors)
  - [Annotate existing API documentation with behaviors](#annotate-existing-api-documentation-with-behaviors)
  - [Manual / Periodic Testing](#manual--periodic-testing)
- [Pros](#pros)
- [Cons](#cons)
<!-- /toc -->

## Summary

Need to define a suite of tests to properly provide coverage for CSI specification.

Proposal to enable a set of guidelines that SHALL be followed when building e2e tests for optional components such that they can be later promoted into a verification test suite. Vendors will use these test suites to test compliance to the additional behavior defined by the component specification. At this moment the test suite will help vendors ability to test their implementation, whether is meets the specification.

This KEP utilizes CSI implementation as a reference for such a optional component to promote and validate CSI API for any third-party CSI enabled drivers and installations. Many vendors SHALL utilize this test suite to test their own offering to claim that they support CSI functionality. For any driver to be Verified as a CSI driver must pass these tests completely.


A 'Verification Test Suite' is a set of e2e tests that test a specific functionality of a component, such as CSI implementation.
This test suite is not intended to advocate how to run conformance/compliance program on the non-core components.
The goal of this KEP is not about defining an official certification program for optional components.
The primary goal of this KEP is to define a Verification test suite for CSI implementation such that third party vendors can utilize this suite of tests to test their implementations confidently.
The discussion of officially promoting this test suite to a conformance profile is out of scope for now.
Until the verification suite has been thoroughly tested and matured based on feedback from stake holders, it is premature to have a discussion on whether it should also service as part of certification

## Motivation

Main motivation for this KEP is the finalize a suite of tests that any CSI storage vendor can run to claim that their storage driver implementation supports Kubernetes CSI API fully and CSI workloads work as expected on their platform. 

This test suite must take into account API coverage and Behavioral coverage so that vendor implementations can claim coverage with confidence. A clear documentation of the test suite must be provided to understand the coverage.


### Goals

* Propose a guideline document that constitute Test Requirements, Test Approval process, Setup instructions, promotion process to the verification suite.
* Define a suite of tests that validate a Kubernetes platform is correctly implementing the CSI standard
* Socialize the suite of test to ensure it provides proper coverage of the CSI Interface
* Socialize the suite of tests with the list of certified platforms to determine the number of platforms that are able to support this test suite. Provide a mechanism to capture vendor results, this could be though testgrid.
 
### Non-Goals

* Guidelines and process for writing test suites for other components such as GPU, CRI, CNI, etc. 
* Based on the number of certified platforms capable of supporting the the CSI test suite, make a recommendation on whether the tests should be promoted to some compliance program.
* SIG that owns the non-core component, SHOULD define a suite of tests that will fully test the compliance of the non-core component behavior.
* Promoting verification test suite in to a official certification program.

## Proposal

The proposal consists of following deliverables:
* Identify and categorize a suite of tests that tests CSI implementations
* Document the suite of test so that end-user can read the documentation to understand what is tested and possible outcome from the test. 
* provide guidelines for writing and categorizing test suites for non-core components that are required to ensure specification is met
* provide documentation on how to run test suite on vendor created implementations of the non-core components

** This proposal does not advocate how to run conformance/compliance program on the non-core components

 
### Risks and Mitigations

Current risk with no requirements on Verification test suite for a component would cause uncertainty on vendor implementations 
With a clear set of guidelines and Verification test suite produce expected outcomes. 

## Design Details

Delivery of this KEP shall be done in the following phases:

### Phase 1

In Phase 1, we will:
* Identify all existing test that should be part of verification tests
* Identify gaps in the coverage and propose list of new tests if required
* Provide a setup guidelines for running the test
* Provide a mechanism to capture the output of the test results for verification purpose.
* Create a process for SIG to follow to develop test suite and document tests and methods to run the test suite.  

### Phase 2

In Phase 2, we will:
* Analyze and advocate other components to use the process, generalize the process to see how this can be applied to other components such as CRI, CNI, etc. 
** It is expected that all non-core components may not behave the same but the process guidelines should provide general guidelines to follow and specificity about some components can be added into the guidelines document as required.  
 
### Graduation Criteria

Define the process and guidelines for building a Verification test suite to CSI implementation
Generalize such process to generically apply to all other non-core components

## Implementation History

- 2019-11-01: Created

## Drawbacks

* Process is iterative and will mature over time to be applied to other components.

## Alternatives

### Annotate test files with behaviors
* Spec each test independently and add the tests to a suite or group them appropriately

### Annotate existing API documentation with behaviors
* Group and document tests according to the API that they are calling to get coverage

### Manual / Periodic Testing
* Group the tests and be able to run them manually and through CI to show that they functionality works


## Pros
* Every SIG such as sig-node, sig-storage, etc have their own suite of tests that test non-core functionality. There is no mechanism in place for verifying vendor implementations. Define process and guidelines, develop a Verification test suite that will test third party implementation to the specification. 

## Cons
* Each non-core component with differ in requirements, this will change the process and guidelines that may not be generalized.