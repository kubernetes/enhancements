---
title: Dynamic Scoring Plugin
authors:
  - "@zhangyingnan"
owning-sig: sig-scheduling
reviewers:
  - ""
approvers:
  - ""
creation-date: 2020-12-21
last-updated: 2020-12-21
status: provisional
---

 # Dynamic Scoring Plugin

 ## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
- [Proposal](#proposal)
  - [MatchLabelExpressions and MatchAnnotationExpressions](#matchlabelexpressions-and-matchannotationexpressions)
    - [Example for MatchAnnotationExpressions](#example-for-matchannotationexpressions)
  - [CreationExpressions](#creationexpressions)
    - [Example for CreationExpressions](#example-for-creationexpressions)
  - [ConditionExpressions](#conditionexpressions)
  - [NodeInfoExpressions](#nodeinfoexpressions)
    - [Example For NodeInfoExpressions](#example-for-nodeinfoexpressions)
<!-- /toc -->

## Summary

A new score plugin: DynamicScoring.
It can give an opportunity to add some system-level preferences dynamically when we run into some temporary conditions.

## Motivation

There are some scenarios that we want to prioritize a group of nodes dynamically, which are the temporary system-level preference.

During **os upgrading**, it upgrades nodes by group.
- Sometimes pods running on upgrading nodes might be evicted and re-schedule.
- Sometimes the upgrading process for one cluster will probably last a long time, there might be new pods creation during this upgrading process.
At these moments, we prefer to schedule these pods to the nodes that have already finished the osupgrade. If we have the feature to adjust node score for the scheduler on purpose dynamically, we can configure a high priority for the upgraded nodes during the upgrading process, and remove this temporary configuration after upgrading.

There might be the similar scenario during **k8s release upgrading** as os upgrading.

During **onboarding nodes**, if we want to give some warmup time to the new nodes, it's better to have an opportunity to give a high priority to the nodes created 15min ago.

When we find there might be some **problems with nodes onboarded in a certain time window**, we need some time to debug the nodes issue, it's better to have an opportunity to give a low priority to the nodes created in this time window.

## Proposal

Add a new score plugin (DynamicScoring).

This plugin will watch a ConfigMap. In that ConfigMap, it defines the DynamicScoring.
It can adjust the node score inside this plugin dynamically based on the configuration in this ConfigMap.

This DynamicScoring registered in Group "kubescheduler.config.k8s.io"

```
apiVersion: kubescheduler.config.k8s.io/v1
kind: DynamicScoring
```

DynamicScoring customizes the scores for different groups of nodes. And it supports several selectors to group the nodes, selected by **node.Labels**, **node.Annotations**, **node.CreationTimestamp**, **node.Status.Conditions** and **node.Status.NodeInfo**. Accordingly, it supports 5 Expressions, `MatchLabelExpressions`, `MatchAnnotationExpressions`, `CreationExpressions`, `ConditionExpressions` and `NodeInfoExpressions`.

Other nodes with no customized score will be treated as score=0 by default in this plugin. Finally this plugin will map these customized scores to [0-100]. The scores set in this configuration just indicate the priority.

**Data Structures for DynamicScoring**:
```
import corev1 "k8s.io/api/core/v1"

// A time operator is the set of operators that can be used in a timestamp requirement.
type TimeOperator string
const (
  // After a specific date(RFC3339). Such as "After 2020-12-07T08:50:38Z"
  TimeOpAfter TimeOperator = "After"
  // Before a specific date(RFC3339). Such as "Before 2020-12-07T08:50:38Z"
  TimeOpBefore TimeOperator = "Before"
  // Since a relative duration. Such as "Since 15m"
  TimeOpSince TimeOperator = "Since"
  // Ago, a relative duration ago. Such as "Ago 15m"
  TimeOpAgo TimeOperator = "Ago"
)
type TimestampRequirement struct {
  // Represents a timestamp's relationship to the value.
  // Valid operators are After, Before, Since, Ago
  Operator TimeOperator `json:"operator,omitempty" protobuf:"bytes,1,opt,name=operator,casttype=TimeOperator"`
  // Value is a timestamp or duration.
  // After or Before a specific date(RFC3339).
  // Since or Ago is newer or older than a relative duration like 5s, 2m, or 3h.
  Value string `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
}
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// DynamicScoring defines the dynamic score policy
type DynamicScoring struct {
  metav1.TypeMeta `json:",inline"`
  // DynamicScores define a score for different node groups.
  // The scores of other nodes not defined are zero.
  DynamicScores []DynamicScoringTerm `json:"dynamicScores,omitempty" protobuf:"bytes,1,rep,name=dynamicScores"`
}
type DynamicScoringTerm struct {
  // Score for the selected nodes
  Score int32 `json:"score" protobuf:"varint,1,opt,name=score"`
  // Select a group of nodes
  Selector DynamicNodeSelectorTerm `json:"selector" protobuf:"bytes,2,opt,name=selector"`
}
type DynamicNodeSelectorTerm struct {
  // selected by node.Labels
  MatchLabelExpressions []corev1.NodeSelectorRequirement `json:"matchLabelExpressions,omitempty" protobuf:"bytes,1,rep,name=matchLabelExpressions"`
  // selected by node.Annotations
  MatchAnnotationExpressions []corev1.NodeSelectorRequirement `json:"matchAnnotationExpressions,omitempty" protobuf:"bytes,2,rep,name=matchAnnotationExpressions"`
  // selected by node.CreationTimestamp
  CreationExpressions []TimestampRequirement `json:"creationExpression,omitempty" protobuf:"bytes,3,rep,name=creationExpressions"`
  // selected by node.Status.Conditions
  ConditionExpressions []corev1.NodeSelectorRequirement `json:"conditionExpressions,omitempty" protobuf:"bytes,4,rep,name=conditionExpressions"`
  // selected by node.Status.NodeInfo
  NodeInfoExpressions []corev1.NodeSelectorRequirement `json:"nodeInfoExpressions,omitempty" protobuf:"bytes,5,rep,name=nodeInfoExpressions"`
}
```

### MatchLabelExpressions and MatchAnnotationExpressions

MatchLabelExpressions and MatchAnnotationExpressions are very straightforward, they use the normal node selector to match node labels and node annotations.

#### Example for MatchAnnotationExpressions

Sometimes, kubernetes will use annotations to control some features still in alpha/beta version. So if we want to temporarily prioritize some nodes using `volumes.kubernetes.io/controller-managed-attach-detach`. We can use MatchAnnotationExpressions like:

```
dynamicScores:
- score: 100
  selector:
  - matchAnnotationExpressions:
    - key: "volumes.kubernetes.io/controller-managed-attach-detach"
      operator: In
      values: ["true"]
```

### CreationExpressions

CreationExpressions use a new defined Requirement - `TimestampRequirement`. Which supports 4 kinds of operators, `After`, `Before`, `Since` and `Ago`. `After` or `Before` is after or before a specific date(RFC3339). Since or Ago is newer or older than a relative duration like 5s, 2m, or 3h.

For example, you want to set a score for nodes created before a certain time, you can use `Before` operator to specify this timestamp requirement, it will execute on `node.CreationTimestamp`.

#### Example for CreationExpressions

When we find there might be some problems with nodes onboarded in a certain time window (between `2020-12-05T06:27:42Z` and `2020-12-09T06:27:42Z`), we need some time to debug the nodes issue, we can give a high priority to the nodes outside this time window.

```
dynamicScores:
- score: 100
  selector:
  - creationExpressions:
    - operator: Before
      value: "2020-12-05T06:27:42Z"
    - operator: After
      value: "2020-12-09T06:27:42Z"
```

### ConditionExpressions

`ConditionExpressions` will execute on the NodeCondition list to select a group of nodes whose node conditions match the defined expressions.

### NodeInfoExpressions

NodeInfoExpressions will execute on the `NodeInfo` to select a group of nodes whose node info matches the defined expressions.

Migrate `NodeSystemInfo` for each node to a cached Map (Here, it is map[string]string), saved into the scheduler NodeInfo cache. This cached map will be the execution target to match the NodeInfoExpressions. And it chooses the NodeSystemInfo/fieldname as the map key, and NodeSystemInfo/fieldvalue as the map value.

#### Example For NodeInfoExpressions

For the osupgrade scenario, it can use NodeInfoExpressions to group nodes. Priority the node using the new kernelVersion.

This configuration will give 100 score to the node using the new kernel `5.4.0-34.generic.x86_64`, and other nodes will get 0 in this score plugin. Then the new pod during osupgrade will have more opportunities to be scheduled to the upgraded node.

```
dynamicScores:
- score: 100
  selector:
  - nodeInfoExpressions:
    - key: kernelVersion
      operator: In
      values: ["5.4.0-34.generic.x86_64"]
```
