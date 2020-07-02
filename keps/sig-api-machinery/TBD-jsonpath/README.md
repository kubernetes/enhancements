# KEP-xxxx: Clarify and Enhance JSONPath

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Syntax reference](#syntax-reference)
    - [Operators](#operators)
    - [Filter Operators](#filter-operators)
    - [Functions](#build-in-functions)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Graduation Criteria](#graduation-criteria)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [References](#references)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

- [ ] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website
[original JSONPath syntax]: https://goessner.net/articles/JsonPath
[existing JSONPath doc]: https://github.com/kubernetes/website/blob/ecc27bbbe70f92d031fa52be76bb9471b2e83152/content/en/docs/reference/kubectl/jsonpath.md
[JSON Strategic Merge Patch]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-api-machinery/strategic-merge-patch.md
[go template]: https://golang.org/pkg/text/template/
## Summary

This proposal aims to clarify the JSONPath template syntax supported by Kubernetes
and make some enhancement to meet today's user needs.

In addition to [original JSONPath syntax], this JSONPath enhancement mainly allow users to apply more complex, flexible filter
on JSON-formatted resources using kubectl.
Enables the ability to extract desired fields with a single and simple statement.

## Motivation

There are already many issues/pull-requests about JSONPath in Kubernetes.
However, none of them were approved for lacking standard.
Besides, the [existing JSONPath doc] for Kubernetes is ambiguous:
No supported logical operator list, either no clear syntax.

In fact, the most authoritative JSONPath standard was proposed in 2007 and never been updated in past 13 years. See [original JSONPath syntax].
It's unreasonable and self deceiving to reject evolving for better user experience just because of an outdated standard.
Like [JSON Strategic Merge Patch], We need someone step out for the first change.

The job for this KEP is basically in 2 parts:
- Clarify supported syntax
- Enhancement

Both parts should have clear and sufficient documentation about syntax/examples.

### Goals

* Document relationship with [original JSONPath syntax]
* Have a clear list of supported JSONPath syntax, including original and enhanced syntax
* Backward compatibility if possible, enhanced JSONPath aims be a super-set to original one
* Sufficient examples in documentation
* UnitTest coverage on enhanced syntax
* Provide a in-code-flag to enable or disable enhanced items
  * Server-side: If an enhancement has security risk. It should be disabled by default.
  (eg: support regular expression filters on server side may cause a DoS attack)
  * Client-side: Enable as much as enhancement by default if possible
* Investigate all usage on server side to avoid security problem

## Proposal

### Syntax reference
Three tables including all supported operators and functions.

#### Operators
Note that typically we refer unspecified length `Array` as `Slice` in Go.

**Operator** | **Description** | **Example**
------------------------------- | --- | ---
`{ }`                           | Text out of brace will be output as is. | `Pod name {.metadata.name}`
`.`                             | Dot-notated child operator | `{.metadata.name}`
`$`                             | The root object to query. <br>Can be ignored at the start of template | `{$.metadata.name}`<br> equals to <br>`{.metadata.name}`
`@`                             | The current object | `{@}`
`*`                             | Wildcard operator. <br>Available anywhere a name or numeric index required | `{.metadata.*}`
`['<name>'(,'<name>')]`         | bracket-notated child operator. <br>`<name>` should always be warped with `''` (a pair of single quota) | `{.metadata['name','namespace']}`
`[<number>(,<number>)]`         | Slice index operator. <br>Negative index will return the last `N` element. <br>For exmaple: `-1` returns the last one element | `{.spec.containers[0,-1]}`
`..`                            | Recursive descent, or known as deep scan | `{.spec..image}`
`[start:end(:step)]`            |  Slice operator. Return elements from `start` to `end`(not include `end`). <br>Similar to [Go slice](https://blog.golang.org/slices) operator, but every `step`(non-zero, default: `1`) element is selected between `start` and `end`. <br>If `step` is negative, the slice will be reverse-processed. | `{.spec.containers[1:]}` <br>`{.spec.containers[:-1]}` <br>`{.spec.containers[::-1]}`
`[?(<expression>)]`             | Filter expression | `{.spec.containers[?(@.name=='app')].image}`
`(<expression>)`                | Script expression. <br>Can be used inside Filter expression | <code>{.items[?((@.metadata.name=='app1'</code><br><code>&#124;&#124;@.metadata.name=='app2')</code><br><code>&&@.spec.replicas==1)].spec..image}</code>
`{range <list>}<template>{end}` | Iterate list(slice or map). <br>Similar to `{{range pipeline}} T1 {{end}}` in [go template] | <code>{range $.metadata}Resource</code><br><code> {.kind} {.name} is created at</code><br><code> {.creationTimestamp}{end}</code>

#### Filter Operators
Filter operators are logical expressions usually used to filter slices.

For example: `kubectl get pods -A -o jsonpath="{.items[?(@.status.phase == 'Running')]..image}"`
will get you all image names used by current `Running` pod.

Note that comparison between non-numeric type is done in lexicographical order.

Comparison between numeric and non-numeric is typically invalid.


**Filter Operator** | **Description** | **Example**
----------------------------- | --- | ---
`==`                          | left is equal to right. Note that number `1` is NOT equal to string `'1'` | `{.items[?(@.spec.replicas)==2]}`
`!=`                          | left is not equal to right | `{.items[?(@.spec.replicas!=1)]}`
`>`                           | left is greater than right | `{.items[?(@.spec.replicas>2)]}`
`>=`                          | left is greater than or equal to right | `{.items[?(@.spec.replicas>=2)]}`
`<`                           | left is less than right | `{.items[?(@.spec.replicas<2)]}`
`<=`                          | left is less than or equal to right | `{.items[?(@.spec.replicas<=2)]}`
`&&`                          | logical AND | `{.items[?(@.spec.replicas==1&&@.status.readyReplicas==1)]}`
<code>&#124;&#124;</code>    | logical OR | <code>{.items[?(@.status.availableReplicas==1&#124;&#124;@.status.readyReplicas==1)]}</code>
`=~`                          | left matches [regular expression](https://golang.org/pkg/regexp/) given in right | `{.items[?(@.metadata.name=~'^myapp.*')]}`


#### Build in Functions
Functions should be invoked at the end of an expression with a single dot(`.`)

It takes the output of the expression as input.

**Functions** | **Output type** | **Description** | **Example**
------- | --- | --- | ---
`len()` | int | return the length of a string, slice or map | `{.items.len()}` <br>`{.items[?(@.spec.containers.len()>1)]}`

## Design Details

TBD


### Test Plan

TBD

### Risks and Mitigations
TBD
### Graduation Criteria
TBD
#### Beta
TBD
#### GA
TBD

### Upgrade / Downgrade Strategy

TBD

### Version Skew Strategy

TBD

## References

TBD

## Implementation History

- 2020-07-01: KEP introduced
- 2020-07-02: Finish goals
- 2020-07-03: Finish proposal
