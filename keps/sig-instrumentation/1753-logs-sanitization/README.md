# KEP-1753: Kubernetes system components logs sanitization

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Performance overhead](#performance-overhead)
- [Design Details](#design-details)
  - [Source code tags](#source-code-tags)
  - [datapolicy verification library](#datapolicy-verification-library)
  - [klog integration](#klog-integration)
  - [Logging configuration](#logging-configuration)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (1.20)](#alpha-120)
    - [Beta (1.21)](#beta-121)
    - [GA (1.22)](#ga-122)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Static code analysis](#static-code-analysis)
    - [Limitations of Static Analysis](#limitations-of-static-analysis)
      - [Theoretical](#theoretical)
      - [Practical](#practical)
    - [Strengths and Weaknesses of Static and Dynamic Analyses ([3])](#strengths-and-weaknesses-of-static-and-dynamic-analyses-3)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes the introduction of a logging filter which could be applied to all Kubernetes system components logs to prevent various types of sensitive information from leaking via logs.

## Motivation

One of the outcomes of the [Kubernetes Security Audit](https://www.cncf.io/blog/2019/08/06/open-sourcing-the-kubernetes-security-audit/) was identification of two vulnerabilities which were directly related to sensitive data like tokens or passwords being written to Kubernetes system components logs:

- *6.* Bearer tokens are revealed in logs

- *22.* iSCSI volume storage cleartext secrets in logs

To address this problem audit authors suggested what follows:

_**Ensure that sensitive data cannot be trivially stored in logs.** Prevent dangerous logging actions with improved code review policies. Redact sensitive information with logging filters. Together, these actions can help to prevent sensitive data from being exposed in the logs_

Taking into account the size of the Kubernetes source code and the pace at which it changes it is very likely that many similar vulnerabilities which have not been identified yet still exist in the source code. Those impose significant threat to the Kubernetes clusters security whenever logs from Kubernetes system components are exposed to the users.

This KEP directly addresses _“Redact sensitive information with logging filters”_ part of the audit document recommendation by proposing the introduction of a dynamic filter integrated with Kubernetes logging library which could inspect all log entry parameters before those are used to build a log message and redact those logs entries which might contain sensitive data.


### Goals

- Prevent commonly known types of sensitive data from being accidentally logged.
- Make it easy to extend sanitization logic by adding new sources of sensitive data which should never be logged.
- Limit the performance overhead related to logs sanitization to acceptable level.

### Non-Goals

- Eliminate completely the risk of exposing any security sensitive data via Kubernetes system components logs.
- Identify all places in the Kubernetes source code which store sensitive data.
- Provide a generic set of source code tags which go beyond what is needed for this KEP.

## Proposal

We propose to define a set of standard Kubernetes [go lang struct tags](https://golang.org/ref/spec#Struct_types) which could be used to tag fields which contain sensitive information which should not be logged because of security risks.

Adding those tags in the source code will be a manual process which we want to initiate with this KEP. Finding places where those should be added can be aided by grepping Kubernetes source code for common phrases like password, securitykey or token.

When it comes to standard go lang types and third party libraries used in Kubernetes which cannot be changed we propose listing them in one place with information about the type of the sensitive data which they contain. 

We also propose to implement a small library which could use the above information to verify if any of the provided values contain reference to sensitive data.

Finally we propose to integrate this library with the klog logging library used by Kubernetes in a way that when enabled the log entries which contain information marked as sensitive will be redacted from the logs.

### Risks and Mitigations

#### Performance overhead

Inspection of log parameters can be time consuming and it can impose significant performance overhead on each log library invocation.

This risk can be mitigated by:
- running inspection only on log entry parameters which are going to be actually logged - running inspection after log v-level has been evaluated.
- introducing a dedicated go lang interface with a single method returning information about types of sensitive data which given value contains like:

  ```go
  type DatapolicyDetector interface {
	  ContainsDatapolicyValues() []string
  }
  ```

- implementations of this interface could be auto generated using a dedicated code generator similar to deep-copy generator or manually implemented when needed,
caching negative inspection results for parameter types which does not have any references to types which may contain sensitive data.

Which of those methods will be used and to what extent will be decided after running performance tests.


## Design Details

### Source code tags

We propose to mark all struct fields which may contain sensitive data with a new datapolicy tag which as a value will accept a list of predefined types of data.

For now we propose following types of data to be available as values:
- password
- token
- security-key

Example of using datapolicy tag:

```go
type ConfigMap struct {
  Data map[string]string `json:"data,omitempty" datapolicy:”password,token,security-key”`
}
```

For external types which are not part of the Kubernetes source code such as go lang standard libraries or third party vendored libraries for which we cannot change source code we will define a global function which will map those into relevant datapolicy tag:

```go
func GlobalDatapolicyMapping(val interface{}) []string
```

### datapolicy verification library

datapolicy verification library will implement logic for checking if the provided value contains any sensitive data identified by the datapolicy tag. Verification will be performed using reflection. Recursion will stop on pointers as values for those are usually not logged. Verification will depend only on the datapol tags values. Values of individual fields of primitive types like string will not be chec ked other than checking if a given field is not empty.

```go
package datapol {
  // Verify returns a slice with types of sensitive data the given value contains. 
  func Verify(value interface{}) []string
}
```

When used datapolicy verification library will be initialized with GlobalDatapolicyMapping function to take into account information about external types which may contain sensitive data.

### klog integration

Currently the klog library used by Kubernetes does not provide any extension point which could be used for filtering logs.

We propose to add a new interface to klog library:

```go
type LogFilter interface {
  Filter(args []interface{}) (args []interface{})
  FilterF(format string, args []interface{}) (string,[]interface{})
  FilterS(msg string, keysAndValues []interface{}) (string, []interface{})
}
```

and the global function:

```go
func SetLogFilter(filter LogFilter)
```

which will make provided log filter methods to be called:
- `Filter()` - for each `Info()`, `Infoln()`, `InfoDepth()` and related methods invocations.
- `FilterF()` - for each `Infof()` and related methods invocations.
- `FilterS()` - called for each `InfoS()` and `ErrorS()` methods invocations.

datapolicy verification library will be integrated with klog library in a way that when enabled log entries for which datapolicy.Verify() will return non empty value will be replaced with the following message:

`“Log message has been redacted. Log argument #%d contains: %v”`

where `%d` is the position of log parameter which contains sensitive data and `%v` is the result of datapol.Verify() function.

### Logging configuration

To allow configuring if logs should be sanitized we will introduce a new logging configuration field shared by all kubernetes components.

`--logging-sanitization` flag should allow to pick if the sanitization will be enabled. Setting this flag to true will enable it.


### Test Plan

### Graduation Criteria

#### Alpha (1.20)
- All well-known places which contain sensitive data have been tagged.
- All external well-known types are handled by the GlobalDatapolicyMapping function.
- It is possible to turn on logs sanitization for all Kubernetes components including API Server, Scheduler, Controller manager and kubelet using --logging-sanitization flag.

#### Beta (1.21)
- Performance overhead related to enabling dynamic logs sanitization has been reduced to 50% compared to time spent in klog library functions with this feature disabled. This should be verified using Kubernetes scalability tests.

#### GA (1.22)
- Logs sanitization is enabled by default.

## Implementation History

## Drawbacks

## Alternatives

### Static code analysis

Instead of introducing optional dynamic filtering of logs at runtime we could use the same metadata to perform static code analysis. 

#### Limitations of Static Analysis

##### Theoretical

The major theoretical limitation of static analysis is imposed by results from [decidability theory](https://www.tutorialspoint.com/automata_theory/language_decidability.htm) - given an arbitrary program written in a general-purpose programming language (one capable of simulating a Turing machine), it is impossible to examine the program algorithmically and determine if an arbitrarily chosen statement in the program will be executed when the program operates on arbitrarily chosen input data ([1]). Furthermore, there is no algorithmic way to identify those programs for which the analysis is possible; otherwise, the halting problem for Turing machines would be solvable.

One major, ramification of this result concerns the distinctinction between syntactic and semantic control paths. Syntactic paths comprise all possible control paths through the flow graph. Semantic paths are those syntactic paths that can be executed. However, not all syntactic paths are executable paths. Thus, the semantic paths are a subset of the syntactic paths. In static analysis, it would be highly desirable to identify the semantic path. However, decidability results state that there is no algorithmic way to detect the semantic path through an arbitrary program written in a general purpose programming language.

[1] "Tutorial: Static Analysis and Dynamic Testing of Computer ...." https://ieeexplore.ieee.org/abstract/document/1646907/. Accessed 15 May. 2020.

##### Practical 

A major practical limitation of static analysis concerns array references and evaluation of pointer variables. Array subscripts and pointers are mechanisms for selecting a data item at runtime, based on previous computations performed by the program. Static analysis cannot evaluate subscripts pointers and, thus, is unable to distinguish between elements of an array or members of a list. Although it might be possible to analyze subscripts and pointers using techniques of symbolic execution, it is generally simpler and more efficient to use dynamic testing.

The other practical limitation of static analysis is slow execution on large models of state ([2]).

[2] "Static and dynamic analysis: synergy and duality - Computer ...." https://homes.cs.washington.edu/~mernst/pubs/staticdynamic-woda2003.pdf. Accessed 15 May. 2020.

#### Strengths and Weaknesses of Static and Dynamic Analyses ([3])

Static analysis, with its whitebox visibility, is certainly the more thorough approach and may also prove more cost-efficient with the ability to detect bugs at an early phase of the software development life cycle. Static analysis can also unearth errors that would not emerge in a dynamic test. Dynamic analysis, on the other hand, is capable of exposing a subtle flaw or vulnerability too complicated for static analysis alone to reveal. A dynamic test, however, will only find defects in the part of the code that is actually executed.

Therefore static and dynamic analysis should not be considered as disjoint alternatives but rather as a complementary solutions and in the end we should have both implemented in Kubernetes.

[3] "Static Testing vs. Dynamic Testing - Veracode." 3 Dec. 2013, https://www.veracode.com/blog/2013/12/static-testing-vs-dynamic-testing. Accessed 15 May. 2020.
