---
title: Structured Logging
authors:
  - "@serathius"
  - "@44past4"
owning-sig: sig-instrumentation
participating-sigs:
  - sig-architecture
reviewers:
  - "@thockin"
  - "@bgrant0507"
  - "@dims"
approvers:
  - "@brancz"
  - "@piosz"
editor: TBD
creation-date: 2019-11-15
last-updated: 2019-12-20
status: provisional
see-also:
replaces:
superseded-by:
---

# Structured Logging

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Logging Interface](#logging-interface)
    - [Logger initialization](#logger-initialization)
    - [Storing Logger and metadata in Context](#storing-logger-and-metadata-in-context)
    - [Example](#example)
  - [New logging formats](#new-logging-formats)
    - [Json format](#json-format)
  - [Logging configuration](#logging-configuration)
  - [Kubernetes log schema](#kubernetes-log-schema)
  - [Migration](#migration)
    - [Stage 1 / Alpha](#stage-1--alpha)
    - [Preparation](#preparation)
    - [Automatic migration](#automatic-migration)
    - [Stage 2 / Beta](#stage-2--beta)
    - [Stage 3 / GA](#stage-3--ga)
    - [Stage 4 / Deprecation](#stage-4--deprecation)
  - [User Stories](#user-stories)
    - [Story 1 - Kubernetes developer working with local logs](#story-1---kubernetes-developer-working-with-local-logs)
    - [Story 2 - Cluster Administrator configures log ingestion into ElasticSearch](#story-2---cluster-administrator-configures-log-ingestion-into-elasticsearch)
    - [Story 3 - Administrator uses Elasticsearch to debug problem with workload](#story-3---administrator-uses-elasticsearch-to-debug-problem-with-workload)
  - [Performance](#performance)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Migration failing halfway](#migration-failing-halfway)
    - [Hard to customize logger format](#hard-to-customize-logger-format)
    - [Huge increase of log volume](#huge-increase-of-log-volume)
- [Design Details](#design-details)
  - [Mapping between logging methods](#mapping-between-logging-methods)
  - [Example of migrating klog call](#example-of-migrating-klog-call)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Removal of old logging format](#removal-of-old-logging-format)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
    - [Enhance klog instead of replacing it with logr](#enhance-klog-instead-of-replacing-it-with-logr)
    - [Avoid invoking both klog and logr api](#avoid-invoking-both-klog-and-logr-api)
- [Code organisation / infrastructure needed](#code-organisation--infrastructure-needed)
- [Production Readiness Review Process](#production-readiness-review-process)
<!-- /toc -->

## Summary

Current logging in the Kubernetes control-plane doesn’t provide any uniform structure for logs which makes storing, processing, querying and analyzing logs hard and forces administrators and developers to rely on ad-hoc solutions. This KEP proposes a new logging API and a mechanism for standardizing logging keys that makes logs much easier to ingest and process by modern logging solutions.

## Motivation

Decision for selecting logging library in Kubernetes dates to the beginning of the project. Kubernetes needed a one preferred library that can ensure consistent format across multiple repositories. For that Google’s open-sourced `glog` was used. Later from pains of growing project `glog` was forked into `klog` to address immediate issues. Looking at the current state of `klog`, there were no significant changes into its logging format since its inception. `klog` logging format provides only minimal set of metadata needed for developers to understand the cause of some event or do simple debugging, but it’s not enough for cluster operators to do any analysis on large multi-master cluster, find recurring problems on fleet or implement reliable monitoring and alerting on top of it.

Simple `printf` like logging is currently not enough for today’s distributed system. Thankfully since kubernetes inception the observability for cloud native applications evolved significantly. Unification and correlation of all telemetry verticals (logs, metrics, traces) has become an important feature of SaaS provides like Splunk, Datadog or Sumologic and with CNCF projects like OpenTelemetry starts to become standard. Many log storage backends like BigQuery, Elastic and CloudWatch Logs have matured allowing to efficiently process and analyze massive datasets of structured data. Such solutions rely heavily on consistency of metadata, which is required for efficient indexing and filtering. An example of projects that tackles this problem is Elastic Common Schema, which can be adapted by application independent of the language they are written in.

All those changes have led to a situation when most of the logs produced by distributed applications is now not stored locally but instead is aggregated and sent to some sort of external storage system. This makes working with logs much easier and provides completely new opportunities for using logs for different kinds of quantitative analyzes but at the same time moves the focus for logs from being easy to read for humans to be easy to process and index by logging services.

Because Kubernetes is a very technical solution on which lots of developers work on a daily basis we should not abandon completely developers centered approach for logging but instead add a way for Kubernetes users to choose between those two perspectives depending on the cluster purpose - if the given cluster is used for development or is it a production cluster. However this does not mean that the human-readable output format provided for developers needs to be exactly the same as the current klog format.

Apart from being easy to ingest into logging services logs in order to be useful needs also to be searchable meaning that it should be easy for the user to find the most relevant information in them. Solutions like Elastic or BigQuery makes querying structured logs very easy. What they do not provide is the consistent schema to be used by those logs. Without it querying logs will be still possible but finding specific information might be much harder. Therefore apart from simply having some structure logs produced by applications should also follow a schema which would enforce consistent naming, consistent way of referencing the same objects and common set of metadata to be stored for each of those objects. With such schema in place building queries for logs becomes much easier.

### Goals

* Provide structured logging interface in K8s codebase
* Standardize commonly used logging keys (e.g. referencing k8s object, http request fields)
* Propose sound strategy for migration ensuring its completeness
* Make implementation of context based solutions like logs correlation or tracing within Kubernetes easier.

### Non-Goals

* We are **not** changing logging pipeline in K8s
* We are **not** standardizing some specific log messages
* We are **not** tackling logging keys lifecycle (should be addressed by separate KEP when new API is close to GA)

## Proposal

This KEP proposes introducing a structured logging into Kubernetes control-plane components. As part of this proposal we will introduce a new logging API that will replace klog. We will introduce a Kubernetes logging schema which will provide a way to define a common set of logging keys and ensure their consistent usage. To provide immediate benefit for adapting a new API we will propose a new logging format that will simplify ingestion of structured data into third party logging solutions. As part of changes in kubernetes logging we propose to tightly couple usage of logging and golang context.

### Logging Interface

When choosing new structured logging API following criteria has been used:
* Selected API should provide abstract structured logging interface so that it would be easy to inject different implementations for the same API.
* Selected API should be minimalistic so that it should be easy to reimplement, wrap or to mock it.
* Applicability of the selected API should not be limited to the Kubernetes control plane but it should be possible to use the same API in other components running on Kubernetes clusters which are also implemented in go.
* Selected API should support structured logging by providing a way to define a set of key-value pairs for each log entry.
* There should at least one production ready implementation of the selected API which could produce JSON formatted logs.
* It should be possible to produce klog compatible logs from the selected API which could be used during the migration to a new API.

Based on this criteria https://github.com/go-logr/logr log interface has been selected as a logging API for Kubernetes.

Apart from meeting the criteria listed above logr has a few nice features:
* Constant log message together with logger name provide stable identification for log entries which greatly simplifies querying logs.
* Limiting log levels to just Info and Error makes it much easier to choose the proper log level for given log.
* Using key-values pairs for defining log fields allows more efficient implementation of the logger.

#### Logger initialization

Logger implementation initialization should be done at startup of component and passed down the stack as an interface. Aside of initialization of logger, there should not be any dependency on specific implementation of logger Interface.

#### Storing Logger and metadata in Context

We would like to propose that logger interface and all metadata collected trough call stack to be stored in Golang Context object. This way creating one canonical way to pass contextual information across the call stack. Some pros of this approach:
* Standard method in Golang to pass scoped variables across API boundaries and between processes.
* No need for additional function argument for passing logging API through call stack
* Context is used similarly by many frameworks including for instance OpenCensus tracing API.
* Accessing metadata will be independent from Logging and could be reused by other functionalities.

Access to context variables should be done though predefined functions. For that we propose to a new module to kubernetes component base called `k8s.io/component-base/kontext`. This new module will include all utility functions for storing and retrieving logger and metadata needed by it.

Throughout call stack new metadata can be added to context that can be later used by logger. For example reconciliation loop can reference the reconcilided object so that all logs will be referencing it. All metadata access should be done through functions in `k8s.io/component-base/kontext` module. Adding reference to kubernetes object (pod, service, node) should be really straightforward.

#### Example

```go
import "k8s.io/component-base/kontext"

func UpdateStatus(ctx context.Context, pods []*v1.Pod, newStatus bool) {
	for _, pod := range pods {
		podCtx := kontext.WithObject(ctx, pod)
		updatePodStatus(podCtx, pod, newStatus)
	}
}

func updatePodStatus(ctx context.Context, pod *v1.Pod, newStatus bool) {
  logger := kontext.GetLogger(ctx).WithName("probe")
  logger.V(6).Info("Updating ready status", "ready", newStatus)
  [...]
  logger.V(4).Info("Ready status updated", "ready", newStatus)
}
```

### New logging formats

Introduction of new logging API should be paired with new output format that can properly utilize structured logging. In order to simplify log ingestion, but still maintain similar developer experience we would like to propose two formats. First one for production use and second one for component development.

#### Json format

We would like to propose a JSON as a future default format in Kubernetes for both production and development use. Some pros of using JSON:
* Broadly adopted by logging libraries with very efficient implementations (zap, zerolog).
* Out of the box support by many logging backends (Elasticsearch, Stackdriver, BigQuery, Splunk)
* Easily parsable and transformable
* Existing tools for ad-hoc analysis (jq)

Example:

```json
{
   "ts":"00:15:15.525108",
   "msg":"Updating ready status",
   "core/node":{
      "name":"node-1"
   },
   "core/pod":{
      "name":"nginx-1",
      "namespace":"default"
   },
   "probe":{
      "ready":false
   }
}
```

### Logging configuration

We would like to introduce new logging configuration options shared by all kubernetes components. `LoggingConfig` structure should be implemented as part of `k8s.io/component-base` options and include common set of flags for logger initialization.
* `--logging-format` flag should allow to pick between logging interface implementation and allow for backward compatibility. Setting this flag will select particular logger implementation from registry.
* `-v` flag should allow to change verbosity of logging

Any additional configuration needed by loggers should be provided by flags specific to its implementation. Default loggers should provide a basic customization, but it will not be encouraged to avoid feature bloat. Adding new loggers, changing logger behavior or extending its functionality should be done outside of Kubernetes by implementing logr API and registering it in code.

We acknowledge that extension via code is not the best, but should be sufficient for now.

Proposed flag `--logging-format`  values:
* `legacy` for old logging format
* `json` for json format

All logs from the `json` logger should go to standard output.  Out of the existing klog command line arguments only ‘-v’ with be supported in future.

### Kubernetes log schema

Introducing structured logging API is not enough to address one of the biggest issues with log creation or analysis, that is inconsistent metadata. Size of Kubernetes codebase could easily result in drifting naming conventions. As part of this proposal we want to propose a mechanism for ensuring consistency of metadata. To achieve this we will introduce a way to define a schema for subset of commonly used fields in kubernetes. This schema should improve reusability of fields and provide discovery mechanism for logging backend vendors.

Introducing this schema will include three things:
* Schema definition format
* Set of helper functions to populate this schema (e.g. `WithObject` for referencing k8s object)
* Validation mechanism ensuring schema is not bypassed by lower level logging API

Exact schema definition is outside the scope of this proposal and will be addressed by a separate KEP when structured logging effort will enter Beta and logs will start to be migrated. At the end we would expect schema to look similar to [Elastic Common Schema](https://github.com/elastic/ecs/tree/master/schemas)

### Migration

Migration to new structured logging API will be divided into 4 stages:

#### Stage 1 / Alpha

This stage needs to be performed in a single release of Kubernetes. When finished for every **existing** log entry produced in legacy output format there should be a corresponding log entry in json format containing similar information possibly with some additional context information attached to it.
When using the legacy log format after this stage Kubernetes control plane components should produce exactly the same logs as before this stage.
After this stage format and content of json formatted logs will be still in pre-alpha state and most likely will change after manual review.
After completing this stage adding **new log entries should require using only the new logr-based logging API**. Double writing using both klog and logr APIs for new log entries should not be needed.

#### Preparation
* Develop and merge to Kubernetes master 2 implementations/configurations of logr API which will be needed in the latter steps of the migration:
    * `no-op` implementation which does not log anything,
    * `json` log format implementation,
* Implement `k8s.io/component-base/logging` package which during its initialization should read `--logging-format` flag and initialize two global variables providing access to:
    * logr implementation to be used to migrate existing logs (`json` logger for `json` log format and `no-op` logger for legacy format)
    * logr implementation to be used by new logs (`json` loggers for `json` log format and klogr logger for legacy format)
* For each package in Kubernetes source code define a logging module to which it will belong. Each logging module can span across multiple packages.
* For each logging module define two variables with logr implementations to be used for existing and new logs initialized with the name of the module. Those should only be used until stage 2 of the migration is fully completed.
* Implement a package implementing klog API which would check a global variable and either passthrough call to klog (with a proper deep value handing) or do nothing. This global variable should be initialized based on `--logging-format` value.
* Kubernetes and all of its dependencies should be upgraded to klog v2.

#### Automatic migration

1. If `json` log format is selected klog v2 library should be initialized with json logr implementation so that all logs produced by Kubernetes or its dependencies using klog API would be redirected to logr.
1. For all golang packages in Kubernetes following changes should be performed:
    1. All imports and usages of klog package should be replaced with newly created klog wrapper.
    1. For all calls to klog.V().Info*(), klog.Info*(), klog.Warning*(), klog.Error*(), klog.Fatal*() or klog.Exit*() corresponding logr invocation should be added just after those.

This should be done automatically for the whole source code of the Kubernetes by a tool which would analyze the source code and edit it accordingly. The result of the automated migration should be verified and where necessary cleaned up manually.
Generated PRs should be divided by directories and send for review.

In case where constant string is passed as a message argument to logr.Info() or logr.Error() it should be replaced with a normalized constant string produced by removing all formatting placeholders like %s, %q and %v and removing any trailing non alphanumeric characters from its end:

"Update ready status of pods on node [%v]"	-> "Update ready status of pods on node"
"Failed to update status for pod %q: %v"	-> "Failed to update status for pod"

#### Stage 2 / Beta

Logging context propagation

1. New `k8s.io/component-base/kontext` package should be implemented which would provide functionalities related to Context handling in Kubernetes:
    * Attach given logr.Logger instance to existing Context instance.
    * Get logr.Logger attached to given Context.
    * Attach new Kubernetes object to Context and attached logr.Logger.
    * Get Kubernetes object attached to given Context.
1. All entry points to the application like main() function and HTTP request handlers should be identified. In each of those entry points a new Context should be created if it does not exist already and a module logger should be associated with this context.
1. If the entry point to the application handles some Kubernetes object like pod or node this object should also be associated with the entry point Context.
1. Starting from the identified entry points Context containing a logger should be propagated to all places where we log anything. The propagation should be done directly by passing Context as an argument to called functions/methods or indirectly by associating a Context with existing objects passed between functions/methods. In all places where context has been added references to module logger should be replaced with `k8s.io/component-base/kontext.getLogger(context)` calls.

Manual logs verification

1. Detailed documentation for the structured logging best practices in Kubernetes and migration guide for moving from unstructured to structured logs should be prepared.
1. For each automatically generates logr call following checks should be performed manually:
    * If log does not provide true value from the developer or user point of view it should be removed.
    * Log message should be checked if it is meaningful.
    * Log level should be verified if it correct (Error() vs Info()).
    * Log verbosity should be verified if it correct..
    * Log entry fields should be added or changed.
    * Where it makes sense Kubernetes objects should be attached to context.

#### Stage 3 / GA

1. All places were we log anything in the Kubernetes should be reviewed before reaching this stage and all log messages and fields should be stage enough so that they should not change significantly in the next releases.
1. `json` log format should become a default.

#### Stage 4 / Deprecation

Cleanup

1. Support for `legacy` logging format should be removed.
1. klog wrapper package and all of its references should be removed from Kubernetes source code.
1. All dependencies which are included in Kubernetes and which are currently using klog for logging should be migrated to logr. After this is done klog to logr bridge using klogr should be removed from Kubernetes source code.


### User Stories

#### Story 1 - Kubernetes developer working with local logs

TODO(serathius):No major impact when switching from legacy to json format

#### Story 2 - Cluster Administrator configures log ingestion into ElasticSearch

TODO(serathius): Show simplification in Fluentd configuration needed for apiserver access logs after migrating to json format

#### Story 3 - Administrator uses Elasticsearch to debug problem with workload

TODO(serathius): Show examples of some queries to Elasticsearch

### Performance

We are expecting improving performance of logger library due ability to change logging library to require less memory and cpu, then current `klog`. More concerning is increase in volume of logs produced by components.

Example log size change based on apiserver HTTP access log. Picked as the log call responsible for generating the biggest volume of logs.

Using klog generates 206 characters.
```go
klog.Infof("%s %s: (%v) %v%v%v [%s %s]", rl.req.Method, rl.req.RequestURI, latency, rl.status, rl.statusStack, rl.addedInfo, rl.req.UserAgent(), rl.req.RemoteAddr)
```

```
I1025 00:15:15.525108       1 httplog.go:79] GET /api/v1/namespaces/kube-system/pods/metrics-server-v0.3.1-57c75779f-9p8wg: (1.512ms) 200 [pod_nanny/v0.0.0 (linux/amd64) kubernetes/$Format 10.56.1.19:51756]
```

By stage 2 / Beta this log line should change to one shown below. In this example number of bytes generated increased by 34% (minified version). At this stage with we would expect logs to consist of similar formation to klog one, due context metadata being empty.

```go
logger.WithName("http").V(2).Info("Access", "method", rl.req.Method, "uri", rl.req.RequestURI, "latency", latency, "status", rl.status, "agent", rl.req.UserAgent(), "addr", rl.req.RemoteAddr)
```

```json
{
  "time": "00:15:15.525066",
  "v": 2,
  "msg": "HTTP access",
  "http": {
    "method": "GET",
    "uri": "/api/v1/namespaces/kube-system/pods/metrics-server-v0.3.1-57c75779f-9p8wg",
    "latency": "1.512ms",
    "status": "200",
    "agent": "pod_nanny/v0.0.0 (linux/amd64) kubernetes/$Format",
    "addr": "10.56.1.19:51756"
  }
}
```

With time developers will store more and more metadata inside context which can lead to granular increase of log volume between kubernetes versions (e.g. HTTP access logs in future could include service account responsible for making request).
Growth of log size count be be monitored between release to prevent accidental degradation.

Data about byte log length in k8s components. Data taken from scalability test [gce-master-scale-performance](https://k8s-testgrid.appspot.com/sig-scalability-gce#gce-master-scale-performance)

|component              |average|50%ile|75%ile|90%ile|
|-----------------------|-------|------|------|------|
|kube-apiserver         |229    |218   |248   |319   |
|kube-controller-manager|255    |229   |355   |362   |
|kubelet                |759    |885   |912   |1143  |
|kube-scheduler         |217    |225   |226   |227   |


### Risks and Mitigations

#### Migration failing halfway

Kubernetes is a huge project spread across multiple repositories with tens of thousands of logging calls. As not all migration steps can be performed automatically there is a need for some manual work to be performed to verify the results of automatic migration to new logging API and adjusting its results based on a logging migration guidelines and best practices.

As all of this manual work needs to be performed before we can declare GA for structured logging there is a risk that we will not find volunteers who would like to perform this work.
As a possible mitigation we could consider performing more migration steps automatically:
* Implement more intelligent processing of log messages.
* Implement automatic detection of log context by analyzing variables which are visible in the scope of logger call.
* Implement automatic detection of common types of fields used in logs.

#### Hard to customize logger format

Proposed logger implementation are not designed for extensibility. Any change of behavior would require a change in code by creating PR. Looking at kubernetes history we know that this will not be maintainable. Eventually kubernetes adapted dedicated interfaces like CSI, CRI, CNI. We think that similar think could happen for logging in the future, but it doesn’t need to be introduced immediately. A good point for starting this discussion would be when this effort would be GAed. Then we could propose another logging API implementation.

#### Huge increase of log volume

This effort should result in reducing costs of log storage and analysis in production setups. Introducing structured logs will allow for creating better indices and reducing the size of data needed for analysis. From log throughput perspective there will be an increase of log size causing a pressure of log ingestion (disk, logging agents, logging API). To reduce the potential impact we will consider adding additional configuration flags allowing user to specify which metadata should be dropped

## Design Details

### Mapping between logging methods

Following table provides a mapping between klog and logr methods to be used during the automated migration.

|             **klog logger**                |                       **logr logger**                           |
|--------------------------------------------|-----------------------------------------------------------------|
|`klog.V(level).Info*(format/args[0], args)` | `logger.V(level+2).Info(format/args[0], “args”, args)` |
|`klog.Info*(format/args[0], args)`          | `logger.V(2).Info(format/args[0], “args”, args)`       |
|`klog.Warning*(format/args[0], args)`       | `logger.V(1).Info(format/args[0], “args”, args)`       |
|`klog.Error*(format/args[0], args)`         | `logger.Error(error(format/args[0]), format/args[0], “args”, args)` <br><br> or after manual verification that this is not a proper error log: <br><br>`logger.Info(format/args[0], “error”, error(format/args[0]), “args”, args)`
|`klog.Fatal*(format/args[0], args)`         | `logger.Error(error(format/args[0]), format/args[0], “args”, args, “stacks”, logging.AllStacks())`<br>`os.Exit(255)` |
|`klog.Exit*(format/args[0], args)`          | `logger.Error(error(format/args[0]), format/args[0], “args”, args)`<br>`os.Exit(1)` |

### Example of migrating klog call

Before migration:
```go
klog.Infof("%s %s: (%v) %v%v%v [%s %s]", rl.req.Method, rl.req.RequestURI, latency, rl.status, rl.statusStack, rl.addedInfo, rl.req.UserAgent(), rl.req.RemoteAddr)
```

After automatic migration during Stage 1 / Alpha:
```go
klog.Infof("%s %s: (%v) %v%v%v [%s %s]", rl.req.Method, rl.req.RequestURI, latency, rl.status, rl.statusStack, rl.addedInfo, rl.req.UserAgent(), rl.req.RemoteAddr)
logger.V(2).Info("%s %s: (%v) %v%v%v [%s %s]", "args", []string{string(rl.req.Method), string(rl.req.RequestURI), string(latency), string(rl.status), string(rl.statusStack), string(rl.addedInfo), string(rl.req.UserAgent()), string(rl.req.RemoteAddr)})
```

After manual log verification during Stage 2 / Beta:
```go
logger.WithName("http").V(2).Info("Access", "method", rl.req.Method, "uri", rl.req.RequestURI, "latency", latency, "status", rl.status, "agent", rl.req.UserAgent(), "addr", rl.req.RemoteAddr)
```

### Graduation Criteria

#### Alpha

* Introduce a flag allowing to switch to new log format
* Mechanism for defining standard key fields is implemented
* Automatic stage of migration is finished
* Validation requiring calls to klog to be accompanied by logr call

#### Beta

* Structured logs were manually verified
* New Logging guide for K8s is available

#### GA

* Structured logs are the default
* Instructions on migrating other kubernetes repositories
* klog is deprecated and no new klog calls are created

#### Removal of old logging format

We propose to treat change of logging format as a behavior described in [kubernetes deprecation policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/#deprecating-a-feature-or-behavior) and wait for a 1 year before removal of `legacy` logging format from kubernetes.

### Upgrade / Downgrade Strategy

Calling both logging APIs (klog and logr) during migration will ensure identical logs without any configuration changes. Switching to new format will require passing a flag to each component or waiting for full removal of klog.

### Version Skew Strategy

During migration not all components will be able to support the new log format. It should be for users to decide if they want to opt into new logging format on partial set of components or wait for full migration.

## Implementation History

TBD

## Drawbacks

## Alternatives

#### Enhance klog instead of replacing it with logr

The main alternative to the solution proposed in this document for structuring logs in Kubernetes that is worth considering is enhancing existing klog library. There is at least a few ways in which structured logging could be added to klog.

In the simplest form format of logs produced by the existing klog implementation could be changed to JSON so that instead of a text based format with a header containing only basic metadata each log entry could be a JSON document. This change could be introduced quite easily and it could be done in a configurable way using a command line argument because klog starting from version 2 has an option for unconditional redirecting logs to provided logr.Logger instance.
After this change instead of this:

```
I1025 00:15:15.525108       1 controller_utils.go:116] Update ready status of pods on node [node-1]
```

log line could look like this:

```
{"ts": "00:15:15.525108", "lvl": "INFO", "caller": "controller_utils.go:116", "msg": "Update ready status of pods on node [node-a]"}
```

This would simplify parsing of log lines and their ingestion into external storage systems but the actual log message would still be unstructured and therefore this change would not help in any way with the problems related to querying and processing logs.

The simplest way of addressing problem with extracting useful data from log messages and making them queryable on top of klog API would be adding an option to annotate arguments to klog functions which accept format string like klog.Infof() with the name of the argument using either special placeholders in the format string:

```golang
klog.V(2).Infof("Updating ready status of pod %{pod_name}v to false", pod.Name)
```

or by wrapping message values using some special functions:

```golang
klog.V(2).Infof("Updating ready status of pod %v to false", klog.Arg(pod.Name, "pod_name"))
```

This way provided argument could be logged as a separate field in JSON:

```golang
{"ts": "00:15:15.525108", "lvl": "INFO", "caller": "controller_utils.go:116", "msg": "Update ready status of pods on node [node-a]", "args": {"node_name": "node-a"}}
```

This improves the situation a bit but still values which could be logged as a structured key-values would be limited to the ones which appear in the formatted log messages. Further improvements could address this limitation and allow integration of logger with golang context but most likely those would require adding additional methods to the klog API for instance similar to `logr.Logger.WithValues()` method.

The end result of those gradual changes in terms of the format and content of logs and the actual API functionalities could be very similar to the ones which will be produced by the proposed logr-based API. The main difference between the two approaches would lay in the path of the migration.

The main advantage of using klog compatible API for the whole migration for structured logs  would be the possibility of introducing changes to the API and log format in a gradual and less disturbing way and still maintaining the possibility of producing backward compatible logs (why this is important is discussed in the next section of this document).

However at the same time there are some issues which are harder to address using this approach:
* In order to have fully queryable stable logs a few changes which might require some manual work or verification from developers needs to be introduced. Those include introduction of constant log message which could be used to identify log entries, extraction of all relevant log entry key-values/fields, passing context between functions or adding Kubernetes objects to this context. Although those changes can be introduced gradually in a one-by-one fashion the fact that each of those will require involvement of potentially large number of developers means that those changes will most likely be much harder to perform this way than they would be using the all-in-one approach.
* Current klog API is quite messy in some places. This includes things like having multiple logging levels (info/warning/error/fatal) which quite often cannot be clearly separated (info vs. warning, warning vs. error) and the large number of variants of the same method (Info(), Infof(), Infoln(), InfoDepth()). Those things make using logging API harder and it would be good to address those. Switching to a new API might make this migration easier.

#### Avoid invoking both klog and logr api

Instead of leaving the existing references to klog (or actually a klog wrapper) for an extended period of time until the legacy output format is completely removed from the Kubernetes and having to do a double invoking both klog and logr APIs for all existing logs we could consider two alternative approaches:
* Use logr-based API for all existing and new logs.
* Introduce intermediate API which would be a superset of the logr API and could produce exactly the same logs as the existing klog API.

The main problem with the first approach is the fact that it makes very hard to produce log lines which would be exactly the same as the existing klog-based logs. This could be done for instance by adding special fields in logr-based logs called `legacy-message`, `legacy-level` and `legacy-source` which could be lazy evaluated and which could be used only by the logr legacy format.

This however would make accidental breaking of the legacy output format compatibility much easier and would limit the flexibility when it comes to changes which can be introduced in the `json` output format logs as it would be for instance harder to move API calls which produce those.

The second approach is in fact very similar to the approach described in the previous section with the klog API enhancements. Apart from the problems described there it has the same problem with the flexibility which exists for using logr-based API for existing logs. The other problem is that it introduces yet another logging API which might cause some confusion and will make working on multiple versions of the Kubernetes at the same time harder.

The requirement of being able to produce logs in the same format is important for two reasons:
* Although logs have never been considered an API for the Kubernetes and Kubernetes does not provide any guarantees when it comes to logs stability there are still some Kubernetes users which rely on the existing logs for debugging, monitoring or alerting purposes. Adding an option to get access to logs in the old format will provide safer and more reliable path for upgrade for those users.
* It will be hard to come up with a stable set of log messages and fields in the new json/dev format in a single Kubernetes release. Therefore for instance log schema might have a breaking changes before new logs API goes to GA. Depending on the logs storage solution being used this might impose some problems from an operational point of view. To address this issue Kubernetes users which are concerned about this will have an option to stick to the legacy logs format until new format is completely stable.

## Code organisation / infrastructure needed

As a result of this effort we are expecting creation of:
* `k8s.io/component-base/logging` - new module with configuration and implementation of logging api
* `k8s.io/component-base/kontext` - new module for `context.Context` interaction in Kubernetes codebase. Includes shared utility functions (for logging and in future tracing) used to extend context with metadata.
* `k8s.io/kubernetes/test/instrumentation/logging` - new module with static analysis used for logging.
* `sigs.k8s.io/klog-to-logr` - new repository that will include all the scripting used for migration.

## Production Readiness Review Process

**Feature enablement and rollback**
* How can this feature be enabled / disabled in a live cluster? **Changing logging format in control plane components would require recreation of cluster**
* Can the feature be disabled once it has been enabled (i.e., can we roll back the enablement)? **Yes, reverting the change will only effect logs generated when feature was enabled. Rollback can be done by changing flag value to component**
* Will enabling / disabling the feature require downtime for the control plane? **Yes, if we are changing logging configuration of control plane components**
* Will enabling / disabling the feature require downtime or reprovisioning of a node? **Yes, if we are changing logging configuration of kubelet will require node downtime**
* What happens if a cluster with this feature enabled is rolled back? What happens if it is subsequently upgraded again? **Temporary change in log format produced**
* Are there tests for this? **`k8s.io/component-base/logging` and and `k8s.io/component-base/kontext` will be covered by unit tests**

**Scalability**
* Will enabling / using the feature result in any new API calls? Describe them with their impact keeping in mind the supported limits (e.g. 5000 nodes per cluster, 100 pods/s churn): **No**
* Will enabling / using the feature result in supporting new API types? How many objects of that type will be supported (and how that translates to limitations for users)? **No**
* Will enabling / using the feature result in increasing size or count of the existing API objects? **No**
* Will enabling / using the feature result in increasing time taken by any operations covered by existing SLIs/SLOs (e.g. by adding additional work, introducing new steps in between, etc.)? **No**
* Will enabling / using the feature result in non-negligible increase of resource usage (CPU, RAM, disk IO, ...) in any components? Things to keep in mind include: additional in-memory state, additional non-trivial computations, excessive access to disks (including increased log volume), significant amount of data sent and/or received over network, etc. Think through this in both small and large cases, again with respect to the supported limits. **Yes, increased log volume by around 40%**
* Rollout, Upgrade, and Rollback Planning
* Dependencies
    * Does this feature depend on any specific services running in the cluster (e.g., a metrics service)? **No**
    * How does this feature respond to complete failures of the services on which it depends? **n/a**
    * How does this feature respond to degraded performance or high error rates from services on which it depends? **n/a**
* Monitoring requirements
    * How can an operator determine if the feature is in use by workloads? **Need to verify specific component flag, or look into logs generated by it**
    * How can an operator determine if the feature is functioning properly? **By looking at the logs generated by component**
    * What are the service level indicators an operator can use to determine the health of the service? **n/a**
    * What are reasonable service level objectives for the feature? **Defining a proper SLO for logging should consider e2e delivery to backend. This would be outside of scope of kubernetes**
* Troubleshooting
    * What are the known failure modes? **Logging is a blocking API. This means that in case of full node disk, logging can halt the process**
    * How can those be detected via metrics or logs? **liveness probes**
    * What are the mitigations for each of those failure modes? **Monitoring free space on disk**
    * What are the most useful log messages and what logging levels do they require? **logging api will not generate any logs by itself**
    * What steps should be taken if SLOs are not being met to determine the problem? **n/a**

