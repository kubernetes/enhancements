---
title: Apply
authors:
  - "@lavalamp"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-api-machinery
  - sig-cli
reviewers:
  - "@pwittrock"
  - "@erictune"
approvers:
  - "@bgrant0607"
editor: TBD
creation-date: 2018-03-28
last-updated: 2018-03-28
status: implementable
see-also:
  - n/a
replaces:
  - n/a
superseded-by:
  - n/a
---

# Apply

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [API Topology](#api-topology)
      - [Lists](#lists)
      - [Maps and structs](#maps-and-structs)
    - [Kubectl](#kubectl)
      - [Server-side Apply](#server-side-apply)
    - [Managed Fields Serialization](#managed-fields-serialization)
      - [Context](#context)
      - [FieldsV1](#fieldsv1)
      - [Areas of Improvement](#areas-of-improvement)
      - [Example Fieldset](#example-fieldset)
        - [FieldsV1](#fieldsv1-1)
        - [With Improvement 1](#with-improvement-1)
        - [With Improvement 1 and 2](#with-improvement-1-and-2)
        - [With Improvement 1 and 3](#with-improvement-1-and-3)
        - [With Improvement 1, 2, and 3](#with-improvement-1-2-and-3)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Testing Plan](#testing-plan)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

`kubectl apply` is a core part of the Kubernetes config workflow, but it is
buggy and hard to fix. This functionality will be regularized and moved to the
control plane.

## Motivation

Example problems today:

* User does POST, then changes something and applies: surprise!
* User does an apply, then `kubectl edit`, then applies again: surprise!
* User does GET, edits locally, then apply: surprise!
* User tweaks some annotations, then applies: surprise!
* Alice applies something, then Bob applies something: surprise!

Why can't a smaller change fix the problems? Why hasn't it already been fixed?

* Too many components need to change to deliver a fix
* Organic evolution and lack of systematic approach
  * It is hard to make fixes that cohere instead of interfere without a clear model of the feature
* Lack of API support meant client-side implementation
  * The client sends a PATCH to the server, which necessitated strategic merge patch--as no patch format conveniently captures the data type that is actually needed.
  * Tactical errors: SMP was not easy to version, fixing anything required client and server changes and a 2 release deprecation period.
* The implications of our schema were not understood, leading to bugs.
  * e.g., non-positional lists, sets, undiscriminated unions, implicit context
  * Complex and confusing defaulting behavior (e.g., Always pull policy from :latest)
  * Non-declarative-friendly API behavior (e.g., selector updates)

### Goals

"Apply" is intended to allow users and systems to cooperatively determine the
desired state of an object. The resulting system should:

* Be robust to changes made by other users, systems, defaulters (including mutating admission control webhooks), and object schema evolution.
* Be agnostic about prior steps in a CI/CD system (and not require such a system).
* Have low cognitive burden:
  * For integrators: a single API concept supports all object types; integrators
    have to learn one thing total, not one thing per operation per api object.
    Client side logic should be kept to a minimum; CURL should be sufficient to
    use the apply feature.
  * For users: looking at a config change, it should be intuitive what the
    system will do. The “magic” is easy to understand and invoke.
  * Error messages should--to the extent possible--tell users why they had a
    conflict, not just what the conflict was.
  * Error messages should be delivered at the earliest possible point of
    intervention.

Goal: The control plane delivers a comprehensive solution.

Goal: Apply can be called by non-go languages and non-kubectl clients. (e.g.,
via CURL.)

### Non-Goals

* Multi-object apply will not be changed: it remains client side for now
* Some sources of user confusion will not be addressed:
  * Changing the name field makes a new object rather than renaming an existing object
  * Changing fields that can’t really be changed (e.g., Service type).

## Proposal

(Please note that when this KEP was started, the KEP process was much less well
defined and we have been treating this as a requirements / mission statement
document; KEPs have evolved into more than that.)

A brief list of the changes:

* Apply will be moved to the control plane.
  * The [original design](https://goo.gl/UbCRuf) is in a google doc; joining the
    kubernetes-dev or kubernetes-announce list will grant permission to see it.
    Since then, the implementation has changed so this may be useful for
    historical understanding. The test cases and examples there are still valid.
  * Additionally, readable in the same way, is the [original design for structured diff and merge](https://goo.gl/nRZVWL);
    we found in practice a better mechanism for our needs (tracking field
    managers) but the formalization of our schema from that document is still
    correct.
* Apply is invoked by sending a certain Content-Type with the verb PATCH.
* Instead of using a last-applied annotation, the control plane will track a
  "manager" for every field.
* Apply is for users and/or ci/cd systems. We modify the POST, PUT (and
  non-apply PATCH) verbs so that when controllers or other systems make changes
  to an object, they are made "managers" of the fields they change.
* The things our "Go IDL" describes are formalized: [structured merge and diff](https://github.com/kubernetes-sigs/structured-merge-diff)
* Existing Go IDL files will be fixed (e.g., by [fixing the directives](https://github.com/kubernetes/kubernetes/pull/70100/files))
* Dry-run will be implemented on control plane verbs (POST, PUT, PATCH).
  * Admission webhooks will have their API appended accordingly.
* An upgrade path will be implemented so that version skew between kubectl and
  the control plane will not have disastrous results.

The linked documents should be read for a more complete picture.

### Implementation Details/Notes/Constraints [optional]

(TODO: update this section with current design)

What are the caveats to the implementation?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they releate.

#### API Topology

Server-side apply has to understand the topology of the objects in order to make
valid merging decisions. In order to reach that goal, some new Go markers, as
well as OpenAPI extensions have been created:

##### Lists

Lists can behave in mostly 3 different ways depending on what their actual semantic
is. New annotations allow API authors to define this behavior.

- Atomic lists: The list is owned by only one person and can only be entirely
replaced. This is the default for lists. It is defined either in Go IDL by
pefixing the list with `// +listType=atomic`, or in the OpenAPI
with `"x-kubenetes-list-type": "atomic"`.

- Sets: the list is a set (it has to be of a scalar type). Items in the list
must appear at most once. Individual actors of the API can own individual items.
It is defined either in Go IDL by pefixing the list with `//
+listType=set`, or in the OpenAPI with
`"x-kubenetes-list-type": "set"`.

- Associative lists: Kubernetes has a pattern of using lists as dictionary, with
"name" being a very common key. People can now reproduce this pattern by using
`// +listType=associative`, or in the OpenAPI with
`"x-kubenetes-list-type": "map"` along with `//
+x-kubernetes-list-map-key=name`, or `"listMapKey": "name"`. Items
of an associative lists are owned by the person who applied the item to the
list.

For compatibility with the existing markers, the `patchStrategy` and
`patchMergeKey` markers are automatically used and converted to the corresponding `listType`
and `listMapKey` if missing.

##### Maps and structs

Maps and structures can behave in two ways:
- Each item in the map or field in the structure are independent from each
other. They can be changed by different actors. This is the default behavior,
but can be explicitly specified with `// +mapType=separate` or `//
+structType=separate` respectively. They map to the same openapi extension:
`x-kubernetes-map-type=separate`.
- All the fields or item of the map are treated as one, we say the map/struct is
atomic. That can be specified with `// +mapType=atomic` or `//
+structType=atomic` respectively. They map to the same openapi extension:
`x-kubernetes-map-type=atomic`.

#### Kubectl
##### Server-side Apply
Since server-side apply is currently in the Alpha phase, it is not
enabled by default on kubectl. To use server-side apply on servers
with the feature, run the command
`kubectl apply --experimental-server-side ...`.

If the feature is not available or enabled on the server, the command
will fail rather than fall-back on client-side apply due to significant
semantical differences.

As the feature graduates to the Beta phase, the flag will be renamed to `--server-side`.

The long-term plan for this feature is to be the default apply on all
Kubernetes clusters. The semantical differences between server-side
apply and client-side apply will make a smooth roll-out difficult, so
the best way to achieve this has not been decided yet.

#### Managed Fields Serialization

##### Context

The way managedFields is currently serialized (in “Beta 1”) is not
compact enough to allow it to appear in every object in a cluster.
Depending on the exact content of an object, tracking its managed
fields could more than double its size serialized in protobuf, putting
too much strain on etcd io/database size, controller cache size, and
apiserver serialization/deserialization allocations. To get around
this problem, and enable server-side apply by default on 1.16
clusters, managed fields tracking was limited to objects which had
been server-side applied to. This effectively made it opt-in on a per
object basis. The decision to limit apply in this way was made to
allow users to start benefiting from it in a partial state, and in
that sense it was useful, but before server-side apply can go GA, the
serialization problem needs to be solved.

##### FieldsV1

The current fieldset format, FieldsV1, is defined as a trie,
serialized in JSON. Each key in the trie is either a ‘.’, or a
serialized path element. If the key is a ‘.’, this represents the
inclusion of the path leading to this key in the fieldset, and will
always map to an empty set. If the key is a serialized path element,
it represents a sub-field or item, and is split into two parts by the
separating character ‘:’. The character before the separator is a type
prefix and defines the format of the content after the separator,
which will follow one of these four formats:

  * 'f:<name>', where <name> is the name of a field in a struct, or
  * 'key in a map v:<value>', where <value> is the exact json
  * 'formatted value of a list item i:<index>', where <index> is
  * 'position of an item in a list k:<keys>', where <keys> is a map of
  * ' a list item's key fields to their unique values

If a path element maps to an empty set, this is a shortcut for mapping
to {‘.’}

This format was chosen partly because it can be stored as a
RawExtension in the API, which allows it to display in json or yaml
depending on what a client asks for, which makes it more human
readable. But because of this there are also some major inefficiencies
to this format. For example, being serialized json means that we are
wasting a lot of space on control characters, for example, to
represent the list item with the value ‘udp’ for it’s field
‘protocol’, we need to serialize `{“k:{\”protocol\”:\”udp\”}”:{}}`,
which is a total of 29 characters, 9 are because it’s serialized JSON,
11 are just for the format of the path element, 8 are to store the
field name, and only 3 are for identifying the list item uniquely.

##### Areas of Improvement

  1. The first round of improvements, proposed in this
  [PR](https://github.com/kubernetes-sigs/structured-merge-diff/pull/102),
  restructures the trie from nested maps to nested lists, because it
  permits directly serializing keys rather than encoding them as
  strings, which wastes much less space double-encoding strings. It
  also groups the path element type prefixes and whether or not self
  is included into a single int, by taking a path element type int (f
  -> 0, v -> 1, i -> 2, k -> 3) and adding either 0, 4, or 8 to
  specify inclusion of self, children, or both respectively. These
  improvements reduce the serialized size by about 22%, but there are
  still some obvious areas we can improve.

  1. One is field names, which are now a large part of the size of the
  data, since we still directly serialize them as a string. To reduce
  the size further we will implement a statically defined string
  table, which maps int to strings, and instead of directly
  serializing common kubernetes field names, we will just write their
  index in this string table in base64, with a prefix of ! to specify
  that it refers to an item in the string table rather than an actual
  integer. The string table will be versioned, by prepending an int to
  the beginning of the top level nested list, and updated as new field
  names or common annotations are added. Updating the string table
  will be very rare, as it will require clients (who want to
  understand this field) to upgrade. To continue supporting arbitrary
  labels/annotations and CRD field names, we will also allow direct
  field name serialization as a fallback. Without actually
  implementing anything, this was tested by manually replacing strings
  by ints in an example fieldset, and showed an improvement of an
  additional 50-60%.

  1. Another improvement is to stop serializing as JSON at all, and
  simply use a binary format, like protobuf, or a compressed binary
  format. This would be on top of approaches 1+2. This will further
  reduce the impact of control characters, and the largest portion of
  the serialized fieldset will become key values, since those will be
  the final thing that is always still serialized directly as a
  string. We will seek to include common key values (e.g., known
  enums) in the string table to minimize the size further. By similar
  testing to improvement 2, this would probably reduce the size by
  another 50-60%

Each step above represents a mechanical and easily reversible
transformation. The latter two steps (string table and binary
compression) do not require any knowledge of the schema. This is
important because values can be arbitrary json (for set-type lists).

The combined effect of these three improvements will be \~85% smaller
managed fields size, or a \~10% total increase in protobuf serialized
object sizes (based on an approximate \~62% increase in object sizes
from FieldsV1 calculated in this
[doc](https://github.com/kubernetes/enhancements/blob/8a4596505592aa86e7b19b6575f1eafb9a014fcd/keps/sig-api-machinery/20190621-apply-scalability.md))
which seems acceptable for almost all resources (endpoints--where some
clusters run quite close to the size limit already--being the possible
exception).

##### Example Fieldset

###### FieldsV1

```
Field set:
{"f:apiVersion":{},"f:kind":{},"f:metadata":{"f:labels":{"f:app":{},"f:plugin1":{},"f:plugin2":{},"f:plugin3":{},"f:plugin4":{},".":{}},"f:name":{},"f:namespace":{},"f:ownerReferences":{"k:{\"uid\":\"0a9d2b9e-779e-11e7-b422-42010a8001be\"}":{"f:apiVersion":{},"f:blockOwnerDeletion":{},"f:controller":{},"f:kind":{},"f:name":{},"f:uid":{},".":{}},".":{}},".":{}},"f:spec":{"f:containers":{"k:{\"name\":\"some-name\"}":{"f:args":{},"f:env":{"k:{\"name\":\"VAR_1\"}":{"f:name":{},"f:valueFrom":{"f:secretKeyRef":{"f:key":{},"f:name":{},".":{}},".":{}},".":{}},"k:{\"name\":\"VAR_2\"}":{"f:name":{},"f:valueFrom":{"f:secretKeyRef":{"f:key":{},"f:name":{},".":{}},".":{}},".":{}},"k:{\"name\":\"VAR_3\"}":{"f:name":{},"f:valueFrom":{"f:secretKeyRef":{"f:key":{},"f:name":{},".":{}},".":{}},".":{}},".":{}},"f:image":{},"f:imagePullPolicy":{},"f:name":{},"f:resources":{"f:requests":{"f:cpu":{},".":{}},".":{}},"f:terminationMessagePath":{},"f:terminationMessagePolicy":{},"f:volumeMounts":{"k:{\"mountPath\":\"/var/run/secrets/kubernetes.io/serviceaccount\"}":{"f:mountPath":{},"f:name":{},"f:readOnly":{},".":{}},".":{}},".":{}},".":{}},"f:dnsPolicy":{},"f:nodeName":{},"f:priority":{},"f:restartPolicy":{},"f:schedulerName":{},"f:securityContext":{},"f:serviceAccount":{},"f:serviceAccountName":{},"f:terminationGracePeriodSeconds":{},"f:tolerations":{},"f:volumes":{},".":{}},"f:status":{"f:conditions":{"k:{\"type\":\"ContainersReady\"}":{"f:lastProbeTime":{},"f:lastTransitionTime":{},"f:status":{},"f:type":{},".":{}},"k:{\"type\":\"Initialized\"}":{"f:lastProbeTime":{},"f:lastTransitionTime":{},"f:status":{},"f:type":{},".":{}},"k:{\"type\":\"PodScheduled\"}":{"f:lastProbeTime":{},"f:lastTransitionTime":{},"f:status":{},"f:type":{},".":{}},"k:{\"type\":\"Ready\"}":{"f:lastProbeTime":{},"f:lastTransitionTime":{},"f:status":{},"f:type":{},".":{}},".":{}},"f:containerStatuses":{},"f:hostIP":{},"f:phase":{},"f:podIP":{},"f:qosClass":{},"f:startTime":{},".":{}}}
```

| Total bytes | Strings | Integers | Control [“.f:k:v{\:},] |
|---|---|---|---|
| 1968 | 943 | 0 | 1025 |


###### With Improvement 1

```
Field set:
[0,"apiVersion",0,"kind",8,"metadata",[8,"labels",[0,"app",0,"plugin1",0,"plugin2",0,"plugin3",0,"plugin4"],0,"name",0,"namespace",8,"ownerReferences",[11,{"uid":"0a9d2b9e-779e-11e7-b422-42010a8001be"},[0,"apiVersion",0,"blockOwnerDeletion",0,"controller",0,"kind",0,"name",0,"uid"]]],8,"spec",[8,"containers",[11,{"name":"some-name"},[0,"args",8,"env",[11,{"name":"VAR_1"},[0,"name",8,"valueFrom",[8,"secretKeyRef",[0,"key",0,"name"]]],11,{"name":"VAR_2"},[0,"name",8,"valueFrom",[8,"secretKeyRef",[0,"key",0,"name"]]],11,{"name":"VAR_3"},[0,"name",8,"valueFrom",[8,"secretKeyRef",[0,"key",0,"name"]]]],0,"image",0,"imagePullPolicy",0,"name",8,"resources",[8,"requests",[0,"cpu"]],0,"terminationMessagePath",0,"terminationMessagePolicy",8,"volumeMounts",[11,{"mountPath":"/var/run/secrets/kubernetes.io/serviceaccount"},[0,"mountPath",0,"name",0,"readOnly"]]]],0,"dnsPolicy",0,"nodeName",0,"priority",0,"restartPolicy",0,"schedulerName",0,"securityContext",0,"serviceAccount",0,"serviceAccountName",0,"terminationGracePeriodSeconds",0,"tolerations",0,"volumes"],8,"status",[8,"conditions",[11,{"type":"ContainersReady"},[0,"lastProbeTime",0,"lastTransitionTime",0,"status",0,"type"],11,{"type":"Initialized"},[0,"lastProbeTime",0,"lastTransitionTime",0,"status",0,"type"],11,{"type":"PodScheduled"},[0,"lastProbeTime",0,"lastTransitionTime",0,"status",0,"type"],11,{"type":"Ready"},[0,"lastProbeTime",0,"lastTransitionTime",0,"status",0,"type"]],0,"containerStatuses",0,"hostIP",0,"phase",0,"podIP",0,"qosClass",0,"startTime"]]
```

| Total bytes | Strings | Integers | Control [“.f:k:v{\:},] |
|---|---|---|---|
| 1528 | 943 | 104 | 481 |

###### With Improvement 1 and 2
(string table just in order of appearance)

```
Field set:
[1,0,!0,0,!1,8,!2,[8,!3,[0,"app",0,"plugin1",0,"plugin2",0,"plugin3",0,"plugin4"],0,!4,0,!5,8,!6,[11,{!7:"0a9d2b9e-779e-11e7-b422-42010a8001be"},[0,!0,0,!8,0,!9,0,!1,0,!4,0,!7]]],8,!A,[8,!B,[11,{!4:"some-name"},[0,!C,8,!D,[11,{!4:"VAR_1"},[0,!4,8,!E,[8,!F,[0,!G,0,!4]]],11,{!4:"VAR_2"},[0,!4,8,!E,[8,!F,[0,!G,0,!4]]],11,{!4:"VAR_3"},[0,!4,8,!E,[8,!F,[0,!G,0,!4]]]],0,!H,0,!I,0,!4,8,!J,[8,!K,[0,!L]],0,!M,0,!N,8,!O,[11,{!P:!s},[0,!Q,0,!4,0,!R]]]],0,!S,0,!T,0,!U,0,!V,0,!W,0,!X,0,!Y,0,!Z,0,!a,0,!b,0,!c],8,!d,[8,!e,[11,{!f:!o},[0,!g,0,!h,0,!d,0,!f],11,{!f:!p},[0,!g,0,!h,0,!d,0,!f],11,[!f,!q},[0,!g,0,!h,0,!d,0,!f],11,{!f:!r},[0,!g,0,!h,0,!d,0,!f]],0,!i,0,!j,0,!k,0,!l,0,!m,0,!n]]

Example static string table (versioned):
{1:[apiVersion,kind,metadata,labels,name,namespace,ownerReferences,uid,blockOwnerDeletion,controller,spec,containers,args,env,valueFrom,secretKeyRef,key,image,imagePullPolicy,resources,requests,cpu,terminationMessagePath,terminationMessagePolicy,volumeMounts,mountPath,readOnly,dnsPolicy,nodeName,priority,restartPolicy,schedulerName,securityContext,serviceAccount,serviceAccountName,terminationGracePeriodSeconds,tolerations,volumes,status,conditions,type,lastProbeTime,lastTransitionTime,containerStatuses,hostIP,phase,podIP,qosClass,startTime,ContainersReady,Initialized,PodScheduled,Ready,/var/run/secrets/kubernetes.io/serviceaccount]}
```

| Total bytes | Strings | Integers | Control [“.f:k:v{\:},] |
|---|---|---|---|
| 678 | \~ | \~ | \~ |

###### With Improvement 1 and 3
(binary format approximated by compressing into base 64 binary using gzip)

```
Field set:
H4sIAAAAAAAA/6xUTW/bMAz9Lzo7i50WaNpb0WFDMXQN0qKXwhhoiW2EyJJLStmyYf990Edcr9tOy42iHx8fnyg/1pWAQT8gsXZWVHUlttoqUS0r0aMHBR5E9bishIEODYsqVwwJOpjwrG0ziReT+GQSn4o2Hiz0KA4BDyAxdXJfLdIan5DQSoxNmqb6IYJW4kLUcK4W3TnOzs7OcdY0eDbrTheL2emibmpY1nXTofhZ/WWSzji5vY3c79GgP6Sls56cMUiTeafiYt+2baMyHlDm+WMVaIs0ykv4C8Gux1mKiwh65jQV2t0b6MPl+ktTYLnbshI7MAE/kOtzH0ZJ6D/hfo1P2e0t7l/1RV1vKRfHpzz5b8p037qH5+xpilbBmJUzWk6gkZyQXaB88+n4EpB9WTY5BJHZPFKvLcSLvEHmSAh+I/7xqfSJ2p0JPd64YP14eX08pfILMd8BzSnYeZ6K59vQIVn0yO+0mzPSTksEKWNNMea1/rfVIQR1a81+dEBZnk7sFH4+YAfSjrTfl0L2QH6CZblBFQzSWMAoQyy4ctbjN19ySdxlEfdnaqyeWPSRQOIKSTt1h9JZxRnhDFIC5HP2jUV+CR584PEtKF1w2U6/H+LqXI2PZI2g9sUrA+xX5Dq810VLzNwTWE4sY/rQI2qJhGUvC/m11V6D0d9RHZd45dRdMfvIzMdxoT38tZK1d+kzZsDGsb9e5XXaAJfFcqrkXhxfGWA+8JJPXdr2VwAAAP//Dee47/gFAAA=
```

| Total bytes | Strings | Integers | Control [“.f:k:v{\:},] |
|---|---|---|---|
| 768 in base64 | \~ | \~ | \~ |
| 575 in raw bytes | \~ | \~ | \~ |

###### With Improvement 1, 2, and 3
(string table just in order of appearance, binary format approximated by compressing into base 64 binary using gzip)

```
Field set:
H4sIAAAAAAAA/5SQyU7DMBCGn8U+TyTbMWS5lX1fCpQlslBCnFJo0kDFCfHu6HccyAWqXj6NPL++GU8mSRATgKSYmKIsJhZSJojnbctJEG/nH9NZIwe1GtThoNbcwKSBDeg2KZOSPlmUcpEnpSoSG0RRYgMpbRQUWqlAKyFFHgshC8u/MLhbJwaSbrHeGRljoB25Lbe8XKd8uaht0OR1b9hGaue3PxmNH6XvafR2nWHPPew7P9TDuFovHq6Mu9McAIfUB49c8NgFT7rEKXCG5rnf/yJly05++XOJcS+8Aq6BG2AC3AJ3wD3wAORAATy5E5ZusvUjqpQtuhFTRJ6BEqj8L6uUtf8EMlYRe1tleP8j4D4yA16AV2AO1EBjzHcAAAD//7CaT86mAgAA

Example static string table (versioned):
{1:[apiVersion,kind,metadata,labels,name,namespace,ownerReferences,uid,blockOwnerDeletion,controller,spec,containers,args,env,valueFrom,secretKeyRef,key,image,imagePullPolicy,resources,requests,cpu,terminationMessagePath,terminationMessagePolicy,volumeMounts,mountPath,readOnly,dnsPolicy,nodeName,priority,restartPolicy,schedulerName,securityContext,serviceAccount,serviceAccountName,terminationGracePeriodSeconds,tolerations,volumes,status,conditions,type,lastProbeTime,lastTransitionTime,containerStatuses,hostIP,phase,podIP,qosClass,startTime,ContainersReady,Initialized,PodScheduled,Ready,/var/run/secrets/kubernetes.io/serviceaccount]}
```

| Total bytes | Strings | Integers | Control [“.f:k:v{\:},] |
|---|---|---|---|
| 400 in base64 | \~ | \~ | \~ |
| 300 in raw bytes | \~ | \~ | \~ |

### Risks and Mitigations

We used a feature branch to ensure that no partial state of this feature would
be in master. We developed the new "business logic" in a
[separate repo](https://github.com/kubernetes-sigs/structured-merge-diff) for
velocity and reusability.

### Testing Plan

The specific logic of apply will be tested by extensive unit tests in the
[structured merge and diff](https://github.com/kubernetes-sigs/structured-merge-diff)
repo. The integration between that repo and kubernetes/kubernetes will mainly
be tested by integration tests in [test/integration/apiserver/apply](https://github.com/kubernetes/kubernetes/tree/master/test/integration/apiserver/apply)
and [test/cmd](https://github.com/kubernetes/kubernetes/blob/master/test/cmd/apply.sh),
as well as unit tests where applicable. The feature will also be enabled in the
[alpha-features e2e test suite](https://k8s-testgrid.appspot.com/sig-release-master-blocking#gce-cos-master-alpha-features),
which runs every hour and everytime someone types `/test pull-kubernetes-e2e-gce-alpha-features`
on a PR. This will ensure that the cluster can still start up and the other
endpoints will function normally when the feature is enabled.

Unit Tests in [structured merge and diff](https://github.com/kubernetes-sigs/structured-merge-diff) repo for:

- [x] Merge typed objects of the same type with a schema. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/typed/merge_test.go)
- [x] Merge deduced typed objects without a schema (for CRDs). [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/typed/deduced_test.go)
- [x] Convert a typed value to a field set. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/typed/toset_test.go)
- [x] Diff two typed values. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/typed/symdiff_test.go)
- [x] Validate a typed value against it's schema. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/typed/validate_test.go)
- [x] Get correct conflicts when applying. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/merge/conflict_test.go)
- [x] Apply works for deduced typed objects. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/merge/deduced_test.go)
- [x] Apply works for leaf fields with scalar values. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/merge/leaf_test.go)
- [x] Apply works for items in associative lists of scalars. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/merge/set_test.go)
- [x] Apply works for items in associative lists with keys. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/merge/key_test.go)
- [x] Apply works for nested schemas, including recursive schemas. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/merge/nested_test.go)
- [x] Apply works for multiple appliers. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/9f6585cadf64c6b61b5a75bde69ba07d5d34dc3f/merge/multiple_appliers_test.go#L31-L685)
- [x] Apply works when the object conversion changes value of map keys. [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/9f6585cadf64c6b61b5a75bde69ba07d5d34dc3f/merge/multiple_appliers_test.go#L687-L886)
- [x] Apply works when unknown/obsolete versions are present in managedFields (for when APIs are deprecated). [link](https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/merge/obsolete_versions_test.go)

Unit Tests for:

- [x] Apply strips certain fields (like name and namespace) from managers. [link](https://github.com/kubernetes/kubernetes/blob/8a6a2883f9a38e09ae941b62c14f4e68037b2d21/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/fieldmanager_test.go#L69-L139)
- [x] ManagedFields API can be round tripped through the structured-merge-diff format. [link](https://github.com/kubernetes/kubernetes/blob/4394bf779800710e67beae9bddde4bb5425ce039/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/managedfields_test.go#L30-L156)
- [x] Manager identifiers passed to structured-merge-diff are encoded as json. [link](https://github.com/kubernetes/kubernetes/blob/4394bf779800710e67beae9bddde4bb5425ce039/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/managedfields_test.go#L158-L202)
- [x] Managers will be sorted by operation, then timestamp, then manager name. [link](https://github.com/kubernetes/kubernetes/blob/4394bf779800710e67beae9bddde4bb5425ce039/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/managedfields_test.go#L204-L304)
- [x] Conflicts will be returned as readable status errors. [link](https://github.com/kubernetes/kubernetes/blob/69b9167dcbc8eea2ca5653fa42584539920a1fd4/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/conflict_test.go#L31-L106)
- [x] Fields API can be round tripped through the structured-merge-diff format. [link](https://github.com/kubernetes/kubernetes/blob/0e1d50e70fdc9ed838d75a7a1abbe5fa607d22a1/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/fields_test.go#L29-L57)
- [x] Fields API conversion to and from the structured-merge-diff format catches errors. [link](https://github.com/kubernetes/kubernetes/blob/0e1d50e70fdc9ed838d75a7a1abbe5fa607d22a1/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/fields_test.go#L59-L109)
- [x] Path elements can be round tripped through the structured-merge-diff format. [link](https://github.com/kubernetes/kubernetes/blob/6b2e4682fe883eebcaf1c1e43cf2957dde441174/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/pathelement_test.go#L21-L54)
- [x] Path element conversion will ignore unknown qualifiers. [link](https://github.com/kubernetes/kubernetes/blob/6b2e4682fe883eebcaf1c1e43cf2957dde441174/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/pathelement_test.go#L56-L61)
- [x] Path element confersion will fail if a known qualifier's value is invalid. [link](https://github.com/kubernetes/kubernetes/blob/6b2e4682fe883eebcaf1c1e43cf2957dde441174/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/pathelement_test.go#L63-L84)
- [x] Can convert both built-in objects and CRDs to structured-merge-diff typed objects. [link](https://github.com/kubernetes/kubernetes/blob/42aba643290c19a63168513bd758822e8014a0fd/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/typeconverter_test.go#L40-L135)
- [x] Can convert structured-merge-diff typed objects between API versions. [link](https://github.com/kubernetes/kubernetes/blob/0e1d50e70fdc9ed838d75a7a1abbe5fa607d22a1/staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager/internal/versionconverter_test.go#L32-L69)

Integration tests for:

- [x] Creating an object with apply works with default and custom storage implementations. [link](https://github.com/kubernetes/kubernetes/blob/1b8c8f1daf4b1ed6d17ee1d2f40d62c8ecec0e15/test/integration/apiserver/apply/apply_test.go#L55-L121)
- [x] Create is blocked on apply if uid is provided. [link](https://github.com/kubernetes/kubernetes/blob/1b8c8f1daf4b1ed6d17ee1d2f40d62c8ecec0e15/test/integration/apiserver/apply/apply_test.go#L123-L154)
- [x] Apply has conflicts when changing fields set by Update, and is able to force. [link](https://github.com/kubernetes/kubernetes/blob/1b8c8f1daf4b1ed6d17ee1d2f40d62c8ecec0e15/test/integration/apiserver/apply/apply_test.go#L156-L239)
- [x] There are no changes to the managedFields API. [link](https://github.com/kubernetes/kubernetes/blob/1b8c8f1daf4b1ed6d17ee1d2f40d62c8ecec0e15/test/integration/apiserver/apply/apply_test.go#L241-L341)
- [x] ManagedFields has no entries for managers who manage no fields. [link](https://github.com/kubernetes/kubernetes/blob/1b8c8f1daf4b1ed6d17ee1d2f40d62c8ecec0e15/test/integration/apiserver/apply/apply_test.go#L343-L392)
- [x] Apply works with custom resources. [link](https://github.com/kubernetes/kubernetes/blob/b55417f429353e1109df8b3bfa2afc8dbd9f240b/staging/src/k8s.io/apiextensions-apiserver/test/integration/apply_test.go#L34-L117)
- [x] Run kubectl apply tests with server-side flag enabled. [link](https://github.com/kubernetes/kubernetes/blob/81e6407393aa46f2695e71a015f93819f1df424c/test/cmd/apply.sh#L246-L314)

## Graduation Criteria

An alpha version of this is targeted for 1.14.

This can be promoted to beta when it is a drop-in replacement for the existing
kubectl apply, and has no regressions (which aren't bug fixes). This KEP will be
updated when we know the concrete things changing for beta.

This will be promoted to GA once it's gone a sufficient amount of time as beta
with no changes. A KEP update will precede this.

## Implementation History

* Early 2018: @lavalamp begins thinking about apply and writing design docs
* 2018Q3: Design shift from merge + diff to tracking field managers.
* 2019Q1: Alpha.

(For more details, one can view the apply-wg recordings, or join the mailing list
and view the meeting notes. TODO: links)

## Drawbacks

Why should this KEP _not_ be implemented: many bugs in kubectl apply will go
away. Users might be depending on the bugs.

## Alternatives

It's our belief that all routes to fixing the user pain involve
centralizing this functionality in the control plane.
