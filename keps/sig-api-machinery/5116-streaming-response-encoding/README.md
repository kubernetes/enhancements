# KEP-5116: Streaming Encoding for LIST Responses

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Streaming collections and gzip encoding](#streaming-collections-and-gzip-encoding)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [x] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [x] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


## Summary

This KEP proposes implementing streaming encoding for collections (responses to **list**) that are served by the Kubernetes API server.
Existing encoders marshall response into one block allocating GBs of data and keeping it until client reads the whole response.
For large LIST responses, this leads to excessive memory consumption in the API server.
Streaming the encoding process can significantly reduce memory usage, improving scalability and cost-efficiency.

## Motivation

The Kubernetes API server's memory usage presents a significant challenge, particularly when dealing with large resources and LIST requests.
Users can easily issue **list** requests that retrieve gigabytes of data, especially with CustomResourceDefinitions (CRDs). Custom resources often suffer from significant data bloat when encoded in JSON.

Current API server response encoders were designed with smaller responses in mind,
assuming they could allocate the entire response in a single contiguous memory block.
This assumption breaks down with the scale of data returned by large **list** requests.
Even well-intentioned users can create naive controllers that issue multiple concurrent **list** requests without properly handling the responses.
This can lead to the API server holding entire responses in memory for extended periods, sometimes minutes, while waiting for the controller to process them.

The resulting unpredictable memory usage forces administrators to significantly over-provision API server memory to accommodate potential spikes.

### Goals

* Implement JSON and Protocol Buffer streaming encoders for collections (responses to a **list** are called _collections_).
* Significantly reduce and make more predictable the API server's memory consumption when serving large LIST responses.

### Non-Goals

* Implementing streaming decoders in clients. This KEP focuses on protecting the API server's memory usage. Clients can utilize existing mechanisms like pagination or WatchList to manage large datasets.
* Implementing streaming encoders for all content types (e.g. `YAML`, `as=Table`). This KEP focuses on the most commonly used and resource-intensive content types to address the most impactful cases first.
* Implementing streaming for CBOR encoding at this time. CBOR support will be considered as part of a broader effort related to CBOR serialization in Kubernetes and tracked separately.

## Proposal

This proposal focuses on implementing streaming encoding for JSON and Protocol Buffer (Proto) for responses to **list** requests.
The core idea is to avoid loading the entire response into memory before encoding.
Instead, the encoder will process objects individually, streaming the encoded data to the client.
Assuming we will deliver all nessesery testing we plan to launch the feature directly to Beta.

Encoding items one by one significantly reduces the memory footprint required by the API server.
Given the Kubernetes limit of 1MB per object, the memory overhead per request becomes manageable.
While this approach may increase overall CPU usage and memory allocations,
the trade-off is considered worthwhile due to the substantial reduction in peak memory usage,
leading to improved API server stability and scalability.

Existing JSON and Proto encoding libraries do not natively support streaming.
Therefore, custom streaming encoders will be implemented.
Because we focus on encoding collections (**list** responses), the implementation scope is narrowed,
requiring encoders for a limited set of Kubernetes API types.
We anticipate approximately 100 lines of code per encoder per type.
Extensive testing, drawing upon test cases developed for the CBOR serialization effort,
will ensure compatibility with existing encoding behavior.

Long term, the goal is for upstream JSON and Proto libraries to natively support streaming encoding.
For JSON, initial exploration and validation using the experimental `json/v2` package has shown
promising results and confirmed its suitability for our requirements.
Further details can be found in [kubernetes/kubernetes#129304](https://github.com/kubernetes/kubernetes/issues/129304#issuecomment-2612704644).

### Risks and Mitigations


## Design Details

Implementing streaming encoders specifically for collections significantly reduces the scope,
allowing us to focus on a limited set of types and avoid the complexities of a fully generic streaming encoder.
The core difference in our approach will be special handling of the `Items` field within collections structs,
Instead of encoding the entire `Items` array at once, we will iterate through the array and encode each item individually, streaming the encoded data to the client.

This targeted approach enables the following implementation criteria:

* **Strict Validation:**  Before proceeding with streaming encoding, 
  the implementation will rigorously validate the Go struct tags of the target type. 
  If the tags do not precisely match the expected structure, we will fallback to standard encoder.
  This precautionary measure prevents incompatibility upon change of structure fields or encoded representation.
* **Delegation to Standard Encoder:**  The encoding of all fields *other than* `Items`,
  as well as the encoding of each individual item *within* the `Items` array,
  will be delegated to the standard `encoding/json` (for JSON) or `protobuf` (for Proto) packages.
  This leverages the existing, well-tested encoding logic and minimizes the amount of custom code required, reducing the risk of introducing bugs.

The types requiring custom streaming encoders are:

* `*List` types for built-in Kubernetes APIs (e.g., `PodList`, `ConfigMapList`).
* `UnstructuredList` for collections of custom resources.
* `runtime.Unknown` used for Proto encoder to provide type information.

To further enhance robustness, a static analysis check will be introduced to detect and prevent any inconsistencies in Go struct tags across different built-in collection types.
This addresses the concern that not all `*List` types may have perfectly consistent tag definitions.

### Streaming collections and gzip encoding

As pointed out in [kubernetes/kubernetes#129334#discussion_r1938405782](https://github.com/kubernetes/kubernetes/pull/129334#discussion_r1938405782),
the current Kubernetes gzip encoding implementation assumes the response is written in a single large chunk,
checking just first write size to determine if the response is large enough for compression.
This is a bad assumption about internal encoder implementation details and should be fixed regardless.

To ensure response compression works well with streaming,
we will preempt all encoder changes by fixing the gzip compression.
First will add unit tests that will prevent subsequent changes from impacting results,
especially around the compression threshold.
Then, we will rewrite the gzip compression to buffer the response and delay the
decision to enable compression until we have observed enough bytes to hit the threshold
or we received whole response and we can write it without compressing.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

We will implement testing following the cases borrowed from the CBOR test plan ([https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/4222-cbor-serializer#test-plan](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/4222-cbor-serializer#test-plan)), skipping tests that do not apply to streaming *encoding*, such as those related to decoding.

Specifically, we will ensure byte-for-byte compatibility with the standard `encoding/json` and `protobuf` encoders for the following cases:

* Preserving the distinction between integers and floating-point numbers.
* Handling structs with duplicate field names (JSON tag names) without producing duplicate keys in the encoded output ([https://go.dev/issue/17913](https://go.dev/issue/17913)).
* Encoding Go strings containing invalid UTF-8 sequences without error.
* Preserving the distinction between absent, present-but-null, and present-and-empty states for slices and maps.
* Handling structs implementing `MarshallJSON` method, especially built-in collection types.
* Handling raw bytes.
* Linting unit test to ensure all our built-in collection types would be matched.

Fuzz tests will cover the custom streaming encoders for the types with overwritten encoders:
* `testingapigroup.CarpList` as surrogate for built-in types
* `UnstructuredList`

The skipped tests are primarily related to decoding or CBOR-specific features, which are not relevant to the streaming encoding of JSON and Proto addressed by this KEP. 

##### Integration tests

With one to one compatibility to the existing encoder we don't expect integration tests between components will be needed.

##### e2e tests

Scalability tests that will confirm the improvements and protect against future regressions. 
Improvements in the resources should be noticiable in on the perf-dash. 

The tests will cover the following properties:
* Large resource, 10000 objects each 100KB size.
* List with `RV=0` to ensure response is served from watch cache and all the overhead comes from encoder memory allocation.
* Different content type JSON (default), Proto, CBOR, YAML.
* Different API kinds, eg ConfigMap, Pod, custom resources

In first iteration we expect we will overallocate the resources needed for apiserver to ensure passage, 
however after the improvement is implemented we will tune down the resources to detect regressions.

### Graduation Criteria

#### Beta

- Gzip compression is supporting chunking
- All encoder unit tests are implemented
- Streaming encoder for JSON and Proto are implemented
- Scalability test are running and show improvement

#### GA

- Scalability tests are release blocking


### Upgrade / Downgrade Strategy

We plan to provide byte to byte compatibility.

### Version Skew Strategy

We plan to provide byte to byte compatibility.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

Via feature gates

###### How can this feature be enabled / disabled in a live cluster?


- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: StreamingCollectionEncodingToJSON, StreamingCollectionEncodingToProto
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

No, we provide byte to byte compatibility.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, without problem.

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

Yes, will be covered by unit tests.

### Rollout, Upgrade and Rollback Planning

N/A

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements


###### How can an operator determine if the feature is in use by workloads?

N/A

###### How can someone using this feature know that it is working for their instance?

N/A

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A

### Dependencies

No

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No, we expect reduction.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No, we expect reduction.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

## Drawbacks

Maintaining around 500 lines of custom encoder code.

## Alternatives

Similar benefits can be achieved using `WatchList` effort, however we cannot depend on all users migrating to `WatchList`.

Wait for `json/v2` promotion from experimental, this reduces the maintenance, however it comes with even more risk.
New package comes with breaking changes, testing showed that even when enabled in `v1` compatibility there might be some problems.

