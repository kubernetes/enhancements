# KEP-1886: Clarify and enhance JSONPath template syntax

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
  - [util/jsonpath](#utiljsonpath)
  - [forked/golang/template](#forkedgolangtemplate)
  - [Server-side usage](#server-side-usage)
  - [Client-side usage](#client-side-usage)
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
* Provide a in-code-flag to enable or disable enhanced items which may have security risks
  * Server-side: If an enhancement has security risk. It should be disabled by default.
  (eg: support regular expression filters on server side may cause a DoS attack)
  * Client-side: Enable all enhancements if possible
* Review all usage on server side to avoid security problem

## Proposal

### Syntax reference
Three tables including all supported operators and functions.

Note that when you refer a string in JSONPath expression, it should be quoted with single or double quotes.

#### Operators
Note that typically we refer unspecified length `Array` as `Slice` in Go.

**Operator** | **Description** | **Example**
------------------------------- | --- | ---
`{ }`                           | Text out of brace will be output as is. | `Pod name {.metadata.name}`
`.`                             | Dot-notated child operator | `{.metadata.name}`
`$`                             | The root object to query. <br>Can be ignored at the start of template | `{$.metadata.name}`<br> equals to <br>`{.metadata.name}`
`@`                             | The current object | `{@}`
`*`                             | Wildcard operator. <br>Available anywhere a name or numeric index required | `{.metadata.*}`
`['<name>'(,'<name>')]`         | bracket-notated child operator. <br>`<name>` should always be warped with `''` or `""` (a pair of single/double quotes) | `{.metadata['name','namespace']}`
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
<code>&#124;&#124;</code>     | logical OR | <code>{.items[?(@.status.availableReplicas==1&#124;&#124;@.status.readyReplicas==1)]}</code>
`=~`                          | left matches [regular expression](https://golang.org/pkg/regexp/) given in right | `{.items[?(@.metadata.name=~'^myapp.*')]}`


#### Build in Functions
Functions should be invoked at the end of an expression with a single dot(`.`).

It takes the output of the expression as input.

**Functions** | **Output type** | **Description** | **Example**
------- | --- | --- | ---
`len()` | int | return the length of a string, slice or map | `{.items.len()}` <br>`{.items[?(@.spec.containers.len()>1)]}`

## Design Details

### util/jsonpath

Add one field `allowRiskySyntax` to manually enable enhanced items which may have potential security risk.
This is a "switch" for developers who may also want the benefits,
but it depends on whether you trust the given template or not.
You should never "turn on the switch" on server-side when template is unknown at compile-time.

If risky syntax detected with `allowRiskySyntax: false`, it will be treated as invalid syntax,
and an error will be returned. This designed error is only for developers and
user should never see such errors.

[util/jsonpath.go](https://github.com/kubernetes/kubernetes/blob/0c3c2cd6ac8c9ffefc38f9746034e546331b9cd6/staging/src/k8s.io/client-go/util/jsonpath/jsonpath.go)
```go
type JSONPath struct {
	name       string
	parser     *Parser
	stack      [][]reflect.Value
	cur        []reflect.Value
	beginRange int
	inRange    int
	endRange   int

	allowMissingKeys bool
	allowRiskySyntax bool  // control risky syntax parsing. eg: =~(regex)
}

// New creates a new JSONPath with the given name.
func New(name string) *JSONPath {
	return &JSONPath{
		name:       name,
		beginRange: 0,
		inRange:    0,
		endRange:   0,
		// allowRiskySyntax: false // default false
	}
}


// AllowRiskySyntax allows a caller to specify whether they want risky syntax enabled or not
// during paring template. Error will be returned if using risky syntax without AllowRiskySyntax.
func (j *JSONPath) AllowRiskySyntax(allow bool) *JSONPath {
	j.allowRiskySyntax = allow
	return j
}


// return error on regex operator without AllowRiskySyntax
func (j *JSONPath) evalFilter(input []reflect.Value, node *FilterNode) ([]reflect.Value, error) {
...
    switch node.Operator {
    case "<":
        pass, err = template.Less(left, right)
...
    case ">=":
        pass, err = template.GreaterEqual(left, right)
    case "=~":
        if !j.allowRiskySyntax {
            // return error on allowRiskySyntax=false
            return false, fmt.Errorf("filter operator %s not allowed by default", node.Operator)
        }
...
    default:
        return results, fmt.Errorf("unrecognized filter operator %s", node.Operator)
    }
...
}
```

### forked/golang/template

This package is copied from Go library text/template and expose `eq`, `ge`, `gt`, `le`, `lt`,
and `ne` as public functions

So we can continue add more functions like `rmatch`(regex match) to keep extending it for Kubernetes.

[template/func.go](https://github.com/kubernetes/kubernetes/blob/2c1c0f3f7295e0d00651d6e30cfcda56239275e4/staging/src/k8s.io/client-go/third_party/forked/golang/template/funcs.go)
```go
// rmatch evaluates the strs against given regex pattern
// using regexp.MatchString(pattern, str)
// param "pattern" must be string type, "str" could be string or []byte type
func rmatch(pattern interface{}, strs ...interface{}) (bool, error) {
    ...
    return match, err
}
```

### Server-side usage

Do not allow risky syntax unless necessary and template is trusty. (eg: given by developer)

### Client-side usage

Allow risky syntax in kubectl. (eg: CustomColumn)

### Test Plan

JSONPath unit tests
- Cover all path operator
- Cover all filter operator
- Cover all function operator

Kubectl integration tests
- `kubectl` with `-o jsonpath`

### Risks and Mitigations

**Additional errors**

Since some enhanced items are not enabled by default, programmers using client-go may
encounter related errors with legal template according to documentation.
To avoid confusion, the returned error message can be more detail about
how to enable those enhanced syntax. Also we can add some comments
on `jsonpath.New()` method.

### Graduation Criteria

This is actually an enhancement for JSONPath util, it could be used both on server-side and client-side.

Because we try to make a super-set of current JSONPath, typically it would not introduce any
compatibility issue and the main task for GA release is eliminating bugs/security risks. 

#### Beta

- Implement all syntax referred in above tables
- Complete test plan
- Complete documentation and mark enhanced features state as "Beta x.x"
- Review usage on server-side
- Enable enhancements on client-side

#### GA

- At least two releases after beta
- Gather user feedback about jsonpath
- Complete test plan
- Complete documentation and mark enhanced features "since release x.x"

### Upgrade / Downgrade Strategy

This enhancement only affects client-go consumers.

#### Upgrade

* If they do not wish to use enhanced syntax, no additional steps needed for upgrade.
* If they want to have a full access to enhanced syntax, they need invoke `AllowRiskySyntax(true)` in the code.

#### Downgrade

* Remove the call for `AllowRiskySyntax(true)` in the code if any.

### Version Skew Strategy

No version skew observed.

## References

- Related issues
  - [#20352](https://github.com/kubernetes/kubernetes/issues/20352)
  - [#72220](https://github.com/kubernetes/kubernetes/issues/72220)
  
- Related pull-requests
  - [#79227](https://github.com/kubernetes/kubernetes/pull/79227)
  - [#90784](https://github.com/kubernetes/kubernetes/pull/90784)

## Implementation History

- 2020-07-01: KEP introduced
- 2020-07-07: Finish proposal
