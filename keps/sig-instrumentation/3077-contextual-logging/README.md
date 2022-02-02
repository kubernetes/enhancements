# [KEP-3077](https://github.com/kubernetes/enhancements/issues/3077): contextual logging

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
    - [Story 4](#story-4)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Uninitialized logger](#uninitialized-logger)
    - [Logging during initialization](#logging-during-initialization)
    - [Performance overhead](#performance-overhead)
    - [Transition](#transition)
    - [Pitfalls during usage](#pitfalls-during-usage)
      - [Logger in context and function not consistent](#logger-in-context-and-function-not-consistent)
      - [Overwriting context not needed initially, forgotten later](#overwriting-context-not-needed-initially-forgotten-later)
      - [Redundant key/value pairs](#redundant-keyvalue-pairs)
      - [Modifying the same variable in a loop](#modifying-the-same-variable-in-a-loop)
      - [Unused return values](#unused-return-values)
- [Design Details](#design-details)
  - [Feature gate](#feature-gate)
  - [Text format](#text-format)
  - [Default logger](#default-logger)
    - [logging helper API](#logging-helper-api)
    - [Logging in tests](#logging-in-tests)
  - [<a href="https://github.com/kubernetes/klog/tree/main/hack/tools/logcheck">logcheck</a>](#logcheck)
    - [Code examples](#code-examples)
      - [klogr replacing klog completely, explicit logger parameter](#klogr-replacing-klog-completely-explicit-logger-parameter)
      - [Unit testing](#unit-testing)
      - [Injecting common names and values, logger passed through existing ctx parameter](#injecting-common-names-and-values-logger-passed-through-existing-ctx-parameter)
      - [Resulting output](#resulting-output)
  - [Transition plan](#transition-plan)
    - [Initial implementation of k8s.io/klogr done](#initial-implementation-of-k8sioklogr-done)
    - [API considered stable](#api-considered-stable)
    - [Transition complete](#transition-complete)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Per-component logger](#per-component-logger)
  - [Propagating a logger to init code](#propagating-a-logger-to-init-code)
  - [Panic when DefaultLogger is called too early](#panic-when-defaultlogger-is-called-too-early)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary


Contextual logging replaces the global logger by passing a `logr.Logger`
instance into functions via a `context.Context` or an explicit
parameter, building on top of [structured
logging](https://github.com/kubernetes/enhancements/tree/master/keps/sig-instrumentation/1602-structured-logging).

It enables the caller to:
- attach key/value pairs that get included in all log messages
- add names that describe which component or operation triggered a log messages
- reduce the amount of log messages emitted by a callee by changing the
  verbosity

This works without having to pass any additional information into the callees
because the additional information or settings are stored in the modified
logger instance that is passed to it.

Third-party components that use Kubernetes packages like client-go are no
longer forced to use klog. They can choose an arbitrary implementation of
`logr.Logger` and configure it as desired.

During unit testing, each test case can use its own logger to ensure that log
output is associated with the test case.

## Motivation

### Goals

- Remove `k8s.io/klog` imports and usage from all packages.
- Grant the caller of a function control over logging inside that function,
  either by passing a logger into the function or by configuring the object
  that a method belongs to.
- Provide documentation and helper code for setting up logging in unit tests.
- Change as few exported APIs as possible.

### Non-Goals

- Remove the klog text output format.

## Proposal

The proposal is to extend the scope of the on-going conversion of logging calls
to structured logging by removing the dependency on the global klog
logger. Like the conversion to structured logging, this activity can convert
code incrementally over as many Kubernetes releases as necessary without
affecting the usage of Kubernetes in the meantime.

Log calls that were already converted to structured logging need to be updated
to use a logger that gets passed into the function in one of two ways:
- as explicit parameter
- attached to a `context.Context`

When a function already accepts a context parameter, then that will be used
instead of adding a separate parameter. This covers most of client-go and
avoids another big breaking change for the community.

When a function does not accept a context parameter but needs a context for
calling some other functions, then a context parameter will be added. As a
positive side effect, such a change can then also remove several `context.TODO`
calls (currently over 6000 under `pkg`, `cmd`, `staging` and `test`). An
explicit logger parameter is suitable for functions which don’t need a context
and never will.

The rationale for not using both context and an explicit logger parameter is
the risk that the caller passes an updated logger which it didn't add to the
context. When the context is then passed down to other functions instead of the
logger, logging in those functions will not produce the expected result. This
could be avoided by carefully reviewing code which modifies loggers, but
designing an API so that such a mistake cannot happen seems safer. The logcheck
linter will check for this. `nolint:logcheck` comments can be used for
those functions where passing both is preferred despite the ambiguity.

`klog.KObj` and similar helper functions get moved to `k8s.io/klogr`. That package
will also provide aliases for functions and types from `go-logr/logr`. That way,
a single import statement will be enough in most files. It will also implement
loggers for klog text output and unit testing.

When a function retrieves a logger from a context parameter and no logger has
been set there, it will fall back transparently to logging through a global
logger instance tracked in `k8s.io/klogr`, i.e. log calls never need to check
whether they have a valid logger.

A feature gate controls whether contextual logging is used or a global logger
is accessed directly.

### User Stories

#### Story 1

kube-scheduler developer Joan [wants to know which
pod](https://github.com/kubernetes/kubernetes/issues/91633#issuecomment-675074671)
and which operation and scheduler plugin log messages are associated with.

When kube-scheduler starts processing a pod, it creates a new logger with
`logger.WithValue("pod", klogr.KObj(pod))` and passes that around. While
iterating over plugins in certain operations, another logger gets created with
`logger.WithName(<operation>).WithName(<plugin name>)` and then is used when
invoking that plugin. This adds a prefix to each log message which represents
the call path to the specific log message, like for example
`NominatedPods/Filter/VolumeBinding`.

#### Story 2

Scheduler-plugins developer John wants to increase the verbosity of the
scheduler while it processes a certain pod (["per-flow additional
log"](https://github.com/kubernetes-sigs/scheduler-plugins/pull/289)).

John does that by using `logger.V(0)` as logger for important pods and
`logger.V(2)` as logger for less important ones. Then when the scheduler’s
verbosity threshold is `-v=1`, a log message emitted with `V(1).InfoS` through
the updated logger will be printed for important pods and skipped for less
important ones.

#### Story 3

Kubernetes contributor Patrick is working on a unit test with many different
test cases. To minimize overall test runtime, he allows different test cases to
execute in parallel with `t.Parallel()`. Unfortunately, the code under test
suffers from a rare race condition that is triggered randomly while executing
all tests, but never when running just one test case at a time or when single
stepping through it.

He therefore wants to enable logging so that `go test` shows detailed log
output for a failed test case, and only for that test case. He wants to run it
like that by default in the CI so that when the problem occurs, all information
is immediately available. This is important because the error might not show up
when trying the same test invocation locally.

When everything works, he wants `go test` to hide the log output to avoid
blowing up the size of the CI log files.

For each inner test case he adds a `NewTestContext(t)` invocation and uses the
returned context and logger for that test case.

#### Story 4

Client developer Joan wants to use client-go in her application, but is [less
interested in log messages from
it](https://kubernetes.slack.com/archives/CG3518SFJ/p1634217685020400). Joan
makes the log messages from client-go less verbose by creating a logger with
`logger.V(1)` and passing that to the client-go code. Now a `logger.V(3).Info`
call in client-go is the same as a `logger.V(4).Info` in the application and
will not be shown for `-v=3`.


### Risks and Mitigations

#### Uninitialized logger

One risk is that code uses an uninitialized `logr.Logger`, which would lead to
nil pointer panics. This gets mitigated with the [logr convention](https://github.com/go-logr/logr/blob/ec7c16ccad4699c8ad291c385dbb7a3802c2d01c/logr.go#L118-L125)
that a
`logr.Logger` passed by value always must be usable. When the logger is optional,
this needs to be indicated by passing a pointer.

#### Logging during initialization

Another problem is that log messages during program startup need to be handled
somehow. We then can either:
- discard the message, which would break the current `klog.Error(...);
  os.Exit(1)` code pattern when an error occurs during initialization log
- log with some default configuration (for example, as text)

In both cases we have the problem that a logger retrieved by some init code
before logging initialization will continue to use that behavior (discard or
log with default configuration) even after logging gets initialized.

Ideally, no log messages should be emitted before the program is done with
setting up logging as intended. This is already [problematic
now](https://github.com/kubernetes/kubernetes/issues/100152) because output may
change from text (the current klog default) to JSON (once initialized). There
will be no automatic mitigation for this. Such log calls will have to be found
and fixed manually, for example by [passing the error back to `main`
instead](https://github.com/kubernetes/kubernetes/pull/104774), which is part
of an effort to [remove the dependency on logging before an unexpected program
exit](https://github.com/kubernetes/kubernetes/issues/102231).

To support identifying log messages that need to be fixed, the default logger
during program startup will be using the klog text output instead of discarding
the output and a `INIT-LOG` prefix will be inserted into the log output. When
`SetFallbackLogger` is set with the actual logger, it will print an error
message about these early init log calls with the new logger. This has two purposes:
- It's a reminder for those log consumers who only handle output of that new
  logger that they missed some log entries.
- It raises awareness that the program should be updated to initialize logging
  before emitting log entries.

#### Performance overhead

Retrieving a logger from a context on each log call will have a higher overhead
than using a global logger instance. The overhead will be measured for
different scenarios. If a function uses a logger repeatedly, it should retrieve
the logger once and then use that instance.

More expensive than logger lookup are `WithName` and `WithValues` and creating
a new context with the modified logger. This needs to be used cautiously in
performance-sensitive code. A possible compromise is to enhance logging with
such additional information only at higher log levels.

#### Transition

Once some components start to rely on `k8s.io/klogr` exclusively instead of
klog, users of those components must remember to initialize klog *and* the
default logger in `k8s.io/klogr`. Features provided only by klog like writing
to files become unavailable.

This breaking change can be postponed until after the [deprecation of klog
features](https://github.com/kubernetes/enhancements/issues/2845) is complete
by using a logger which passes log messages through to klog. That allows
converting components already now. Then later, the klog dependency in
`k8s.io/klogr` can be replaced with a stand-alone logger which produces the same
text output.

#### Pitfalls during usage

Code reviews must catch the following incorrect patterns.

##### Logger in context and function not consistent

Incorrect:

```go
func foo(ctx context.Context) {
   logger := klogr.FromContext(ctx)
   ctx = klogr.NewContext(ctx, logger.WithName("foo"))
   doSomething(ctx)
   // BUG: does not include WithName("foo")
   logger.Info("Done")
}
```

Correct:

```go
func foo(ctx context.Context) {
   logger := klogr.FromContext(ctx).WithName("foo")
   ctx = klogr.NewContext(ctx, logger)
   doSomething(ctx)
   logger.Info("Done")
}
```

In general, manipulating a logger and the corresponding context should be done
in separate lines.

##### Overwriting context not needed initially, forgotten later

Initial, correct code with contextual logging:

```go
func foo(ctx context.Context) {
   logger := klogr.FromContext(ctx).WithName("foo")
   doSomething(logger)
   logger.Info("Done")
}
```

A line with `ctx = klogr.NewContext(ctx, logger)` could be added above (it
compiles), but it causes unnecessary overhead and some linters complain about
"new value not used".

However, care then must be taken when adding code later which uses `ctx`:

```go
func foo(ctx context.Context) {
   logger := klogr.FromContext(ctx).WithName("foo")
   doSomething(logger)
   // BUG: ctx does not contain the modified logger
   doSomethingWithContext(ctx)
   logger.Info("Done")
}
```

##### Redundant key/value pairs

When the caller already adds a certain key/value pair to the logger, the callee
should still add it as parameter in log calls where it is important. That is
because the contextual logging feature might be disabled, in which case adding
the value in the caller is a no-op. Another reason is that it is not always
obvious whether a value is part of the logger or not. Later when the feature
check is removed and it is clear that keys are redundant, they can be removed.

If there are duplicate keys, the text output will only print the value from the
log call itself because that is the newer value. For JSON, zap will format the
duplicates and then log consumers will keep only the newer value because it
comes last, so the end effect is the same.

This situation is similar to wrapping errors: whether an error message contains
information about a parameter (for example, a path name) needs to be documented
to avoid storing redundant information when the caller wraps an error that was
created by the callee.

Analysis of the JSON log files collected at a high log level during a Prow test
run can be used to detect cases where redundant key/value pairs occur in
practice. This is not going to be complete, but it doesn’t have to be because
the additional overhead for redundant key/value pairs matters a lot less when
the message gets emitted infrequently.

##### Modifying the same variable in a loop

Reusing variable names keeps the code readable and prevents errors because the
variable that isn’t meant to be used will be shadowed. However, care must be
taken to really create new variables. This is broken:

```go
func foo(logger klogr.Logger, objects ...string) {
    for _, obj := range objects {
        // BUG: logger accumulates different key/value pairs with the same key
        logger = logger.WithValue("obj", obj)
        doSomething(logger, obj)
    }
}
```

A new variable must be created with `:=` inside the loop:

```go
func foo(logger klogr.Logger, objects ...string) {
    for _, obj := range objects {
        // This logger variable shadows the function parameter.
        logger := logger.WithValue("obj", obj)
        doSomething(logger, obj)
    }
}
```

##### Unused return values

This code looks like it adds a name, but isn’t doing it correctly:

```go
func foo(logger klogr.Logger, objects ...string) {
        // BUG: WithName returns a new logger with the name,
        // but that return value is not used.
        logger.WithName("foo")
        doSomething(logger)
    }
}
```


## Design Details

klog currently provides a formatter for log messages, a global logger instance,
and some helper code. It also contains code for [log
sanitization](https://github.com/kubernetes/enhancements/issues/1753), but that
is a [deprecated](https://github.com/kubernetes/enhancements/pull/3096) alpha
feature and doesn't need to be supported anymore.

A new package `k8s.io/klogr` will replace both `k8s.io/klog` and
`k8s.io/klog/klogr`. It has to be a stand-alone repository and not a staging
repository because third-party components that get vendored into Kubernetes
might end up using it. If that happens, we wouldn't be able to update a staging
repo with incompatible API changes anymore (vendored code doesn't compile, but
also cannot be updated in its original repo because the API change in
`k8s.io/klogr` cannot be released).

It will have very limited dependencies: initially it will depend on
`k8s.io/klog` and the standard Go runtime. In the final phase (GA) that
`klog.io/klog` dependency will get removed.

### Feature gate

The `k8s.io/klogr` package itself cannot check Kubernetes feature gates because
it has to be a stand-alone package with very few dependencies. Therefore it
will have a global boolean for enabling contextual logging that programs with
Kubernetes feature gates must set:

```
// EnableContextualLogging controls whether contextual logging is enabled.
// By default it is enabled. When disabled, FromContext avoids looking up
// the logger in the context and always returns the fallback logger.
// LoggerWithValues, LoggerWithName, and NewContext become no-ops
// and return their input logger respectively context. This may be useful
// to avoid the additional overhead for contextual logging.
//
// Like SetFallbackLogger this must be called during initialization before
// goroutines are started.
func EnableContextualLogging(enabled bool) {
	contextualLoggingEnabled = enabled
}
```

The `ContextualLogging` feature gate will be defined in `k8s.io/component-base`
and will be copied to klogr during the `InitLogs()` invocation that all
Kubernetes commands already go through after their option parsing.

`LoggerWithValues`, `LoggerWithName`, and `NewContext` are helper functions
that wrap the corresponding functionality from `logr`:
```
// LoggerWithValues returns logger.WithValues(...kv) when
// contextual logging is enabled, otherwise the logger.
func LoggerWithValues(logger Logger, kv ...interface{}) Logger {
	if contextualLoggingEnabled {
		return logger.WithValues(kv...)
	}
	return logger
}

// LoggerWithName returns logger.WithName(name) when contextual logging is
// enabled, otherwise the logger.
func LoggerWithName(logger Logger, name string) Logger {
	if contextualLoggingEnabled {
		return logger.WithName(name)
	}
	return logger
}

// NewContext returns logr.NewContext(ctx, logger) when
// contextual logging is enabled, otherwise ctx.
func NewContext(ctx context.Context, logger Logger) context.Context {
	if contextualLoggingEnabled {
		return logr.NewContext(ctx, logger)
	}
	return ctx
}
```

The logcheck static code analysis tool will warn about code in Kubernetes which
calls the underlying functions directly. Once the feature gate is no longer needed,
a global search/replace can remove the usage of these wrapper functions again.

### Text format

The formatting code will be moved into a standalone `go-logr/logr.LogSink` in the
`k8s.io/klogr/logger` package. The goal is to produce identical output to the current
format, but with `logr.Logger` as interface and just the klog features that
weren’t deprecated (i.e. `-v` and `-vmodule`).

### Default logger

This is meant to replace existing solutions like the global [`Log` instance in
controller-runtime](https://github.com/kubernetes-sigs/controller-runtime/blob/78ce10e2ebad9205eff8429c3f0556788d680c27/pkg/log/log.go#L77-L82). It
is needed as fallback for functions where a logger:
- cannot be passed into the function at all or
- a context was passed without an attached logger.

The goal is to make passing a logger possible everywhere, so the first case
should not occur anymore at some point. The second case remains valid because
it is conventiont to just initialize the global logger once and then have all
code pick it up from there.

The API is:

```go
// SetFallbackLogger may only be called once per program run. Setting it
// multiple times would not have the intended effect because the new logger
// would not be used by code which already retrieved the fallback logger
// earlier.
//
// Therefore this also should be called before any code starts to use logging.
// Such code will get a valid logger to avoid crashes, but that logger will add
// "INIT-LOG" as prefix to all output to indicate that SetFallbackLogger
// should have been called first.
//
// If log messages were emitted with the initial logger, an error
// message will be emitted through the logger that gets passed to
// SetFallbackLogger, using the output format of that logger.
func SetFallbackLogger(log Logger) {
    if fallbackLoggerWasSet {
        panic("the global logger was already set and cannot be changed again")
    }
    fallbackLogger = log
    fallbackLoggerWasSet = true
    if fallbackLoggerWasUsed {
       log.Error(nil, "Logging was used before it was fully initialized. Look for INIT-LOG in the stderr output. Ideally the code emitting those log entries should be called after initializing logging.")
    }
}

// FromContext retrieves a logger set by the caller or, if not set,
// falls back to the program's fallback logger.
func FromContext(ctx context.Context) Logger {
    if logger, err := logr.FromContext(ctx); err == nil {
        return logger
    }

    return fallbackLogger
}

// TODO can be used as a last resort by code that has no means of
// receiving a logger from its caller. FromContext or an explicit logger
// parameter should be used instead.
//
// This function may get deprecated at some point when enough code has been
// converted to accepting a logger from the caller and direct access to the
// fallback logger is not needed anymore.
func TODO() Logger {
    return fallbackLogger
}

// Background retrieves the fallback logger. It should not be called before
// that logger was initialized by the program and not by code that should
// better receive a logger via its parameters. TODO can be used as a temporary
// solution for such code.
func Background() Logger {
    return fallbackLogger
}
```

#### logging helper API

The following helper code will get copied from klog:
- `ObjectRef`
- `KMetaData`
- `KObj`
- `KRef`
- `KObjs`

To ensure that a single import is enough in source files, that package will also contain:

```go
type Logger = logr.Logger
var NewContext = logr.NewContext
// plus more as needed...
```

The implementation of `KObj` and `KObjs` will be either a direct copy of the
code in klog or an improved version [with performance
enhancements](https://github.com/kubernetes/kubernetes/issues/106945),
depending [on benchmark](https://github.com/kubernetes/kubernetes/pull/106594)
results.

#### Logging in tests

Ginkgo tests do not need to be changed. Because each process only runs one test
at a time, the global default can be set and local Logger instances can be
passed around just as in a normal binary.

Unit tests with `go test` require a bit more work. Each test case must
initialize a `klogr/testing` Logger for its own instance of `testing.T`. The
default log level will be 5, the level
[recommended](https://github.com/kubernetes/community/blob/9406b4352fe2d5810cb21cc3cb059ce5886de157/contributors/devel/sig-instrumentation/logging.md#logging-conventions)
for "the steps leading up to errors and warnings" and "for troubleshooting". It
can be higher than in production binaries because `go test` without `-v` will
not print the output for test cases that succeeded. It will only be shown for
failed test cases, and for those the additional log messages may be useful to
understand the failure. Optionally, logging options can be added to the test
binary to modify this default log level by importing the
`k8s.io/klogr/testing/init` package:

Example with additional command line options:

```go
import (
    "testing"
    ktesting "k8s.io/klogr/testing"
    _ "k8s.io/klogr/testing/init"
)

func TestSomething(t *testing.T) {
    // Either return value can be ignored with _ if not needed.
    logger, ctx := ktesting.NewTestContext(t)
            logger.Info("test starts")
            doSomething(ctx)
}
```

Custom log helper code must use the `WithCallStackHelper` method to ensure that
the helper gets skipped during stack unwinding:

```go
func logSomething(logger log.Logger, obj interface{}) {
    helper, logger := logger.WithCallStackHelper()
    helper()
    logger.Info("I am just helping", "obj", obj)
}
```

Skipping multiple stack levels at once via `WithCallDepth` is not working with
loggers that output via `testing.T.Log`. `WithCallDepth` therefore should only
be used by code in `test/e2e` where it can be assumed that the logger is not
using `testing.T`.


### [logcheck](https://github.com/kubernetes/klog/tree/main/hack/tools/logcheck)

That tool is a linter for log calls. It’s used to check that log calls are
well-formed. Because it is used in core Kubernetes and SIGs (for example,
[metrics-server](https://github.com/kubernetes-sigs/metrics-server/blob/4b20c2d43e338d5df7fb530dc960e5d0753f7ab1/Makefile#L252-L257),
it should better be hosted in a repo where "go install" is fast. It only
depends on the standard Go runtime, therefore it can be moved to
`k8s.io/klogr/logcheck`. The tool will be updated to:
- detect usage of klog in code that should have been converted to klogr,
- check not only klog calls, but also calls through the `logr.Logger` interface,
- detect direct calls to `WithValue`, `WithName`, and `NewContext` where
  the `klogr` wrapper functions should be used instead,
- detect function signatures with both `context.Context` and `logr.Logger`.

#### Code examples

See https://github.com/pohly/kubernetes/compare/master-2022-01-12...pohly:log-contextual-2022-01-12
for the helper code and a tentative conversion of some parts of client-go and
kube-scheduler. The last commit removes the klog dependency from `k8s.klogr`
and thus demonstrates that the final goal of this KEP can be achieved.

##### klogr replacing klog completely, explicit logger parameter

```diff
diff --git a/pkg/scheduler/framework/plugins/volumebinding/binder.go b/pkg/scheduler/framework/plugins/volumebinding/binder.go
index ee4989be0b7..3ae0d0b4aaa 100644
--- a/pkg/scheduler/framework/plugins/volumebinding/binder.go
+++ b/pkg/scheduler/framework/plugins/volumebinding/binder.go
@@ -44,7 +44,7 @@ import (
        storagehelpers "k8s.io/component-helpers/storage/volume"
        csitrans "k8s.io/csi-translation-lib"
        csiplugins "k8s.io/csi-translation-lib/plugins"
-       "k8s.io/klog/v2"
+       "k8s.io/klogr"
        v1helper "k8s.io/kubernetes/pkg/apis/core/v1/helper"
        pvutil "k8s.io/kubernetes/pkg/controller/volume/persistentvolume/util"
        "k8s.io/kubernetes/pkg/features"
@@ -151,7 +151,7 @@ type InTreeToCSITranslator interface {
 type SchedulerVolumeBinder interface {
        // GetPodVolumes returns a pod's PVCs separated into bound, unbound with delayed binding (including provisioning)
        // and unbound with immediate binding (including prebound)
-       GetPodVolumes(pod *v1.Pod) (boundClaims, unboundClaimsDelayBinding, unboundClaimsImmediate []*v1.PersistentVolumeClaim, err error)
+       GetPodVolumes(logger klogr.Logger, pod *v1.Pod) (boundClaims, unboundClaimsDelayBinding, unboundClaimsImmediate []*v1.PersistentVolumeClaim, err error)
 
        // FindPodVolumes checks if all of a Pod's PVCs can be satisfied by the
        // node and returns pod's volumes information.
```

##### Unit testing

Each test case gets its own logger instance that adds log messages to the
output of that test case. The log level can be configured with `-v`.

```diff
diff --git a/pkg/scheduler/framework/plugins/volumebinding/binder_test.go b/pkg/scheduler/framework/plugins/volumebinding/binder_test.go
index e354f18198c..223a61f4ece 100644
--- a/pkg/scheduler/framework/plugins/volumebinding/binder_test.go
+++ b/pkg/scheduler/framework/plugins/volumebinding/binder_test.go
@@ -43,7 +43,8 @@ import (
        "k8s.io/client-go/kubernetes/fake"
        k8stesting "k8s.io/client-go/testing"
        featuregatetesting "k8s.io/component-base/featuregate/testing"
-       "k8s.io/klog/v2"
+       "k8s.io/klogr"
+       ktesting "k8s.io/klogr/testing"
        "k8s.io/kubernetes/pkg/controller"
        pvtesting "k8s.io/kubernetes/pkg/controller/volume/persistentvolume/testing"
        pvutil "k8s.io/kubernetes/pkg/controller/volume/persistentvolume/util"
@@ -124,8 +125,8 @@ var (
        zone1Labels = map[string]string{v1.LabelFailureDomainBetaZone: "us-east-1", v1.LabelFailureDomainBetaRegion: "us-east-1a"}
 )
 
-func init() {
-       klog.InitFlags(nil)
+func TestMain(m *testing.M) {
+       ktesting.TestMainWithLogging(m)
 }
 
 type testEnv struct {
...
@@ -972,11 +974,12 @@ func TestFindPodVolumesWithoutProvisioning(t *testing.T) {
        }
 
        run := func(t *testing.T, scenario scenarioType, csiStorageCapacity bool, csiDriver *storagev1.CSIDriver) {
-               ctx, cancel := context.WithCancel(context.Background())
+               logger, ctx := ktesting.NewTestContext(t)
+               ctx, cancel := context.WithCancel(ctx)
                defer cancel()
 
                // Setup
-               testEnv := newTestBinder(t, ctx.Done(), csiStorageCapacity)
+               testEnv := newTestBinder(t, ctx, csiStorageCapacity)
                testEnv.initVolumes(scenario.pvs, scenario.pvs)
                if csiDriver != nil {
                        testEnv.addCSIDriver(csiDriver)
```


##### Injecting common names and values, logger passed through existing ctx parameter

```diff
@@ -671,7 +692,10 @@ func (f *frameworkImpl) RunFilterPlugins(
        nodeInfo *framework.NodeInfo,
 ) framework.PluginToStatus {
        statuses := make(framework.PluginToStatus)
+       logger := klogr.FromContext(ctx).WithName("Filter").WithValues("pod", klogr.KObj(pod), "node", klogr.KObj(nodeInfo.Node()))
        for _, pl := range f.filterPlugins {
+               logger := logger.WithName(pl.Name())
+               ctx := klogr.NewContext(ctx, logger)
                pluginStatus := f.runFilterPlugin(ctx, pl, state, pod, nodeInfo)
                if !pluginStatus.IsSuccess() {
                        if !pluginStatus.IsUnschedulable() {
```

##### Resulting output

Here is log output from `kube-scheduler -v5` for a Pod with an inline volume which cannot be created because storage is exhausted:

```
I1026 16:21:00.461394  801139 scheduler.go:436] "Attempting to schedule pod" pod="default/my-csi-app-inline-volume"
I1026 16:21:00.461476  801139 binder.go:730] PreFilter/VolumeBinding: "PVC is not bound" pod="default/my-csi-app-inline-volume" pvc="default/my-csi-app-inline-volume-my-csi-volume"
```

The next line is from a file which has not been converted. It’s not clear in which context that message gets emitted:
````
I1026 16:21:00.461619  801139 csi.go:222] "Persistent volume had no name for claim" PVC="default/my-csi-app-inline-volume-my-csi-volume"
````

```
I1026 16:21:00.461647  801139 binder.go:266] NominatedPods/Filter/VolumeBinding: "FindPodVolumes starts" pod="default/my-csi-app-inline-volume" node="127.0.0.1"
I1026 16:21:00.461673  801139 binder.go:842] NominatedPods/Filter/VolumeBinding: "No matching volumes for PVC on node" pod="default/my-csi-app-inline-volume" node="127.0.0.1" default/my-csi-app-inline-volume-my-csi-volume="127.0.0.1"
I1026 16:21:00.461724  801139 binder.go:971] NominatedPods/Filter/VolumeBinding: "Node has no accessible CSIStorageCapacity with enough capacity" pod="default/my-csi-app-inline-volume" node="127.0.0.1" pvc="default/my-csi-app-inline-volume-my-csi-volume" pvcSize=549755813888000 sc="csi-hostpath-fast"
I1026 16:21:00.461817  801139 preemption.go:195] "Preemption will not help schedule pod on any node" pod="default/my-csi-app-inline-volume"
I1026 16:21:00.461886  801139 scheduler.go:464] "Status after running PostFilter plugins for pod" pod="default/my-csi-app-inline-volume" status=&{code:2 reasons:[0/1 nodes are available: 1 Preemption is not helpful for scheduling.] err:<nil> failedPlugin:}
I1026 16:21:00.461918  801139 factory.go:209] "Unable to schedule pod; no fit; waiting" pod="default/my-csi-app-inline-volume" err="0/1 nodes are available: 1 node(s) did not have enough free storage."
```

### Transition plan

#### Initial implementation of k8s.io/klogr done

This phase starts when this feature reaches its alpha milestone and the
corresponding Kubernetes release is done. At that point no action is required
by consumers of klog or Kubernetes code because all logging still goes through
klog.

Early adopters are welcome to try out the feature and we will address their
feedback. To meet the beta graduation criteria, we need to collect enough
feedback to be confident that the proposed API will remain stable.

#### API considered stable

Together with the transition to beta we release a k8s.io/klogr v1.0.0 and
notify consumers of the Kubernetes code that they need to make changes in their
code: in addition to initializing klog, they also need to initialize klogr.

A minimal solution is this:

```
import (
    "k8s.io/klog/v2"
    "k8s.io/klog/v2/klogr"
    "k8s.io/klogr"
)

func main() {
   klog.InitFlags(nil)
   klogr.SetFallbackLogger(klogr.NewWithOptions(klogr.WithFormat(klogr.FormatKlog)))
   ...
}
```

klog and klogr both have the same minimal dependencies, so there is no reason
why someone should be able to use klog but not klogr. No other dependencies are
needed.

Logging calls do not need to be changed, neither now nor in later phases: a
program can continue to initialize as shown above and then do logging through
klog functions.

The downside is that callbacks provided by the program that are invoked from
Kubernetes code then will ignore contextual logging, which might make its log
entries less useful (no additional values, for example).

At that point consumers may also switch to initializing logging with
`k8s.io/component-base` as in the
[example](https://github.com/kubernetes/component-base/blob/f57281d7c18ff3a9d1c2404bf97f4ca70bdcb655/logs/example/cmd/logger.go#L26-L52)
from the current Kubernetes. The dependencies of that package are broader, but
still relatively small. In particular, zap is not a hard dependency and only
gets added for JSON output when importing
`k8s.io/component-base/logs/json/register`.

#### Transition complete

Once we are confident that all projects that expect Kubernetes code to log
through klog has been updated, the klog dependency in Kubernetes will be
removed and all logging will be done though the logger defined by klogr. This
is the GA graduation.

If consumers of Kubernetes code haven't adapted their code when updating their
Kubernetes dependencies to a release with that change, Kubernetes log output
will go to stderr with the `INIT-LOG` prefix. The reason for this should be
easy to find and address.

### Test Plan

The new code will be covered by unit tests that execute as part of
`pull-kubernetes-unit`.

Converted components will be tested by exercising them with JSON output at high
log levels to emit as many log messages as possible. Analysis of those logs
will detect duplicate keys that might occur when a caller uses `WithValues` and
a callee adds the same value in a log message. Static code analysis cannot
detect this, or at least not easily.

What it can check is that the individual log calls pass valid key/value pairs
(strings as keys, always a matching value for each key, no duplicate keys).

It can also detect usage of klog in code that should only use contextual
logging.

### Graduation Criteria

#### Alpha

- Common utility code available (logcheck with the additional checks,
  `k8s.io/klogr` v0.1.0)
- Documentation for developers available at https://github.com/kubernetes/community
- At least kube-scheduler framework and some scheduler plugins (in particular
  volumebinding and nodevolumelimits, the two plugins with the most log calls)
  converted
- Initial e2e tests completed and enabled in Prow

#### Beta

- All of kube-scheduler (in-tree) and CSI external-provisioner (out-of-tree) converted
- Gathered feedback from developers and surveys
- Users of klog notified through a Kubernetes blog post and an email to
  dev@kubernetes.io that a logger must be set with k8s.io/klogr.
- Issues filed against kubernetes-sigs projects with a request to set a logger
  with k8s.io/klogr. These issues will have a link to the enhancement issue
  (for cross-referencing) and a link to the developer documentation.

#### GA

- All code in kubernetes/kubernetes converted to contextual logging,
  no dependency on klog anymore
- All code under kubernetes-sigs sets a logger with k8s.io/klogr
  and therefore is ready for the next k/k release.
- User feedback is addressed
- Allowing time for feedback


**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md


### Upgrade / Downgrade Strategy

Not applicable. The log output will be determined by what is implemented in the
code that currently runs.

### Version Skew Strategy

Not applicable. The log output format is the same as before, therefore other
components are not affected.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

This is not a feature in the traditional sense. Code changes like adding
additional parameters to functions are always present once they are made.  But
some of the overhead at runtime can be eliminated via the `ContextualLogging`
feature gate.

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate
  - Feature gate name: ContextualLogging
  - Components depending on the feature gate: all core Kubernetes components
    (kube-apiserver, kube-controller-manager, etc.) but also several other
    in-tree commands and the test/e2e suite.

###### Does enabling the feature change any default behavior?

No. Unless log messages get intentionally enhanced as part of touching the
code, the log output will be exactly the same as before.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by changing the feature gate.

###### What happens if we reenable the feature if it was previously rolled back?

Previous state is irrelevant, so a previous rollback has no effect.

###### Are there any tests for feature enablement/disablement?

Unit tests will be added.

### Rollout, Upgrade and Rollback Planning

Nothing special needed. The same logging flags as before will be supported.

###### How can a rollout or rollback fail? Can it impact already running workloads?

The worst case would be that a null logger instance somehow gets passed into a
function and then causes log calls to crash. The design of the APIs makes that
very unlikely and code reviews should be able to catch the code that causes
this.

###### What specific metrics should inform a rollback?

Components start to crash with a nil pointer panic in the `logr` package.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not applicable.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The feature is in use if using the code from a version with support for
contextual logging, since this can't currently be disabled.

###### How can someone using this feature know that it is working for their instance?

Logs should be identical as previously, unless enriched with additional
context, in which case, additional information is available in logs.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Performance should be similar to logging through klog, with overhead for
passing around a logger not exceeding 2%.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Other
  - Details: CPU resource utilization of components with support for contextual logging before/after an upgrade

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Counting log calls and their kind (through klog vs. logger, `Info`
vs. `Error`) would be possible, but then cause overhead by itself with
questionable usefulness.

### Dependencies

None.

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

Same as before.

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Pod scheduling (= "startup latency of schedulable stateless pods" SLI) might
become slightly worse.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Initial micro benchmarking shows that function call overhead increases. This is
not expected to be measurable during realistic workloads.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Works normally.

###### What are other known failure modes?

None besides bugs that could cause a program to panic (null logger).

###### What steps should be taken if SLOs are not being met to determine the problem?

Revert commits that changed log calls.

## Implementation History

## Drawbacks

Supporting contextual logging is a key design decision that has implications
for all packages in Kubernetes. They don’t have to be converted all at once,
but eventually they should be for the sake of completeness. This may depend on
API changes.

The overhead for the project in terms of PRs that need to be reviewed can be
minimized by combining the conversion to contextual logging with the conversion
to structured logging because both need to rewrite the same log calls.

## Alternatives

### Per-component logger

A logger could be set for object instances and then all methods of that object
could use that logger. This approach is sufficient to get rid of the global
logger and thus for the testing use case. It has the advantage that log
messages can be associated with the object that emits them.

The disadvantage is that associating the log message with the call chain via
multiple `WithName` calls becomes impossible (mutually exclusive designs).

Enriching log messages with additional values from the call chain’s context is
an unsolved problem. A proposal for passing a context to the logger and then
letting the logger extract additional values was discussed in
https://github.com/go-logr/logr/issues/116. Such an approach is problematic
because it increases coupling between unrelated components, doesn’t work for
code which uses the current logr API, and cannot handle values that weren’t
attached to a context.

Finally, a decision on how to pass a logger instance into stand-alone functions
is still needed.

### Propagating a logger to init code

controller-runtime handles the case of init functions retrieving a logger and
keeping that copy for later logging by handing out [a
proxy](https://github.com/kubernetes-sigs/controller-runtime/blob/78ce10e2ebad9205eff8429c3f0556788d680c27/pkg/log/deleg.go).

This has additional overhead (mutex locking, additional function calls for each
log message). Initialization of log output for individual test cases in a unit
test cannot be done this way.

It's better to avoid doing anything with logging entirely in init code.

### Panic when DefaultLogger is called too early

This would provide an even more obvious hint that the program isn’t working as
intended. However, the log call which triggers that might not always be
executed during program startup, which would cause problems when it occurs in
production. Adding an `INIT-LOG` prefix and emitting an error message seem more
suitable.

## Infrastructure Needed

A new stand-alone repo `k8s.io/klogr` is needed.

The same Prow jobs as for structured logging can be used for testing.
