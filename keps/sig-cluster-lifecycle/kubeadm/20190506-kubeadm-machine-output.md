---
title: kubeadm-machine-output
authors:
  - "@akutz"
  - "@bart0sh"
owning-sig: sig-cluster-lifecycle
participating-groups:
reviewers:
  - "@justinsb"
  - "@tstromberg"
  - "@timothysc"
  - "@mtaufen"
  - "@rosti"
  - "@randomvariable"
  - "@fabriziopandini"
  - "@neolit123"
approvers:
  - "@timothysc"
  - "@neolit123"
  - "@fabriziopandini"
editor:
creation-date: 2019-05-06
last-updated: 2019-05-29
status: implementable
---

# Kubeadm machine output

## Table of Contents

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
    - [Story 5](#story-5)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
    - [Details](#details)
      - [A centralized printer system](#a-centralized-printer-system)
      - [Decoupling commands from printing](#decoupling-commands-from-printing)
    - [Notes](#notes)
      - [Previous and related works](#previous-and-related-works)
      - [Buffered vs unbuffered](#buffered-vs-unbuffered)
      - [Parity with kubectl and versioned output](#parity-with-kubectl-and-versioned-output)
      - [jq](#jq)
      - [A friendly bootstrap token struct](#a-friendly-bootstrap-token-struct)
      - [Go template functions](#go-template-functions)
      - [Kubeadm init JSON output](#kubeadm-init-json-output)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Kubeadm should support structured output, such as JSON, YAML, or a [Go template](https://golang.org/pkg/text/template/).

## Motivation

While Kubernetes may be deployed manually, the de facto, if not de jure, means of turning up a Kubernetes cluster is kubeadm. Popular systems management software, such as Terraform, rely on kubeadm in order to deploy Kubernetes. Planned enhancements to the Cluster API project include a composable package for bootstrapping Kubernetes with kubeadm and cloud-init.

Without structured output, even the most seemingly innocuous changes could break Terraform, Cluster API, or other software that relies on the results of kubeadm.

### Goals

* Kubeadm should support structured output, including, but not limited to, the following output types:
  * Buffered
    * JSON
    * YAML
    * Go template
  * Unbuffered
    * Text (the current behavior)
  * Please see [*Buffered vs Unbuffered*](#buffered-vs-unbuffered) for details on what both terms mean in the context of this KEP
* Kubeadm should support the above output types for commands that include, but not necessarily limited to, the following:
  * `alpha certs`
  * `config images list`
  * `init`
  * `token create`
  * `token list`
  * `upgrade plan`
  * `version`

### Non-Goals

* Explicit support for an unbuffered, *short* variant of *Text* output is not necessary at this time as the only command that currently supports this output type is `version`
* [Parity with kubectl's `-o|--output` flag](#parity-with-kubectl-and-versioned-output)
* [Versioned output](#parity-with-kubectl-and-versioned-output)
* [Using the printers](#parity-with-kubectl-and-versioned-output) from the `kubernetes/cli-runtime` package

## Proposal

### User Stories

The examples in the user stories are predicated on:

* [Awareness of `jq`](#jq)
* [A friendly bootstrap token struct](#a-friendly-bootstrap-token-struct)
* [Additional functions for Go templates](#go-template-functions)

#### Story 1

The deployment of a multi-node Kubernetes cluster is automated by parsing [the JSON output](#kubeadm-init-json-output) of `kubeadm init` for the emitted `kubeadm join` command.

1. Run `kubeadm init -o json | jq -r '"kubeadm join \(.node0) --token \(.token.id) --discovery-token-ca-cert-hash \(.caCrt)"'`

#### Story 2

A script requires a list of token IDs.

1. Run `kubeadm token list -o go-template='{{range .}}{{println .ID}}{{end}}'`

#### Story 3

A script returns a list of token IDs as they are discovered.

1. Run `kubeadm token list --stream -o go-template='{{println .ID}}'`

#### Story 4

A script needs to process the IDs of all the non-expired tokens:

1. Run `kubeadm token list -o json | jq '.[] | select(.expires | fromdate > now) | .id'`

#### Story 5

A script needs to process the IDs of all the non-expired tokens as the tokens are discovered:

1. Run `kubeadm token list --stream -o json | jq 'select(.expires | fromdate > now) | .id'`

### Implementation Details/Notes/Constraints

#### Details

##### A centralized printer system
The first design detail is related to how commands handle output...they don't. Or rather, they shouldn't. A command should perform an action and return a result, not print that result. Much like the package [`k8s.io/cli-runtime/pkg/printers`](https://github.com/kubernetes/cli-runtime), a package dedicated to printing will be introduced to kubeadm.

Whether by virtue of an interface that describes a printer, or a single, exported function, the signature of a printer should be:

```golang
func Print(w io.Writer, format string, data interface{}) error
```

A very pseudo-codish implementation of the above signature might look something like this:

```golang
func Print(w io.Writer, format string, data interface{}) error {
	// If data is an io.Reader then copy the reader to the writer.
	if r, ok := data.(io.Reader); ok {
		_, err := io.Copy(w, r)
		return err
	}

	// Handle pre-defined formats.
	switch format {
	case "json":
		return json.NewEncoder(w).Encode(data)
	case "yaml":
		buf, err := yaml.Marshal(data)
		if err != nil {
			return err
		}
		_, err := io.Copy(w, bytes.NewReader(buf))
		return err
	case "text":
		_, err := fmt.Fprintln(w, data)
		return err
	}

	// Treat the format as a Go template.
	tpl, err := template.New("t").Parse(format)
	if err != nil {
		return err
	}
	return tpl.Execute(w, data)
}
```

##### Decoupling commands from printing
If a command is no longer in charge of rendering its output, how does a command submit its output to be rendered? There are several possibilities. Commands could:

1. Receive a Printer object/function used to print results
2. Send results to an exported, package-level function in the `printers` package
3. Instantiate a new priner from the `printers` package
4. Return a `chan interface{}` that receives the command's result(s). This channel can return an `io.Reader` or block until the command has buffered the output into a defined struct to be formatted by some Go template.

#### Notes

##### Previous and related works

* Related
  * https://github.com/kubernetes/kubeadm/issues/494
  * https://github.com/kubernetes/kubeadm/issues/659
  * https://github.com/kubernetes/kubeadm/issues/953
  * https://github.com/kubernetes/kubeadm/issues/972
  * https://github.com/kubernetes/kubeadm/issues/1454
* Replaces
  * https://github.com/kubernetes/kubernetes/pull/75894 ([design doc](https://docs.google.com/document/d/1YzFjb-lTW6HZDvxdG-pwYc4RbSdHvsJ5mZyywQY_QD0))

##### Buffered vs unbuffered

Please note that *buffered* and *unbuffered* relates to individual objects emitted by a command, not the entirety of a command's output. For example, the command `kubeadm token list` may elect to format a list of tokens as JSON after all of the tokens have been discovered, but the same command might also print each token as they are discovered. The behavior will depend on the command and its flags.

For more clarificaton, please see [this example](https://play.golang.org/p/_CJLB7gdLZQ) that highlights how buffered printer output versus unbuffered printer output might behave in the context of this KEP.

##### Parity with kubectl and versioned output

Parity with kubectl is defined as support for all of the output formats currently available to kubectl's `-o|--output` flag:

  * `json`
  * `yaml`
  * `name`
  * `template`
  * `go-template`
  * `go-template-file`
  * `templatefile`
  * `jsonpath`
  * `jsonpath-file`

The natural path to such parity would be to use the same mechanism in kubeadm as used by kubectl, the API machinery package `k8s.io/cli-runtime/pkg/printers`. However, the printers require input objects of type `runtime.Object`.

Creating new or converting existing kubeadm objects to Kubernetes API-style objects has the immediate effect of introducing versioned output to kubeadm. It's the shared opinion of the authors of this KEP that versioned output should be a non-goal. This does not indicate an opinion on the value of versioned output, but rather acknowledges that such a design decision requires a much broader discussion.

##### jq
The program [`jq`](https://stedolan.github.io/jq/) is a performant, command-line solution for parsing and manipulating JSON.

##### A friendly bootstrap token struct

```golang
struct {
	ID          string    `json:"id"`
	TTL         string    `json:"ttl"` // parseable by time.ParseDuration
	Expires     time.Time `json:"expires"`
	Usages      []string  `json:"usages"`
	Description string    `json:"description"`
}
```

##### Go template functions

The Go template into which objects are emitted will have a function map that includes the following functions:

| Signature | Description |
|-----------|-------------|
| `node0() string` | Returns the `addr:port` of the first node in a cluster |
| `caCrt() string` | Returns the `sha256:hash` of the discovery token's CA certificate |
| `join(list []string, sep string) string` | Calls `strings.Join(list, sep)` |

##### Kubeadm init JSON output

The following JSON output is an example of running `kubeadm init -o json`:

```json
{
  "node0": "192.168.20.51:443",
  "caCrt": "sha256:1f40ff4bd1b854fb4a5cf5d2f38267a5ce5f89e34d34b0f62bf335d74eef91a3",
  "token": {
    "id":          "5ndzuu.ngie1sxkgielfpb1",
    "ttl":         "23h",
    "expires":     "2019-05-08T18:58:07Z",
    "usages":      [
      "authentication",
      "signing"
    ],
    "description": "The default bootstrap token generated by 'kubeadm init'.",
    "extraGroups": [
      "system:bootstrappers:kubeadm:default-node-token"
    ]
  },
  "raw": "Rm9yIHRoZSBhY3R1YWwgb3V0cHV0IG9mIHRoZSAia3ViZWFkbSBpbml0IiBjb21tYW5kLCBwbGVhc2Ugc2VlIGh0dHBzOi8vZ2lzdC5naXRodWIuY29tL2FrdXR6LzdhNjg2ZGU1N2JmNDMzZjkyZjcxYjZmYjc3ZDRkOWJhI2ZpbGUta3ViZWFkbS1pbml0LW91dHB1dC1sb2c="
}
```

### Risks and Mitigations

**Risk**: Not including support for [versioned output](#parity-with-kubectl-and-versioned-output) in the stated [goals](#goals) could possibly result in an implied judgement of versioned output, when that's not the reasoning behind omitting it from this KEP.

*Mitigation*: The [Alpha -> Beta](#alpha---beta-graduation) graduation criteria requires a thorough discussion about versioned output.

**Risk**: At first glance it may appear this KEP breaks compatibility with the existing output format for `kubeadm token list`, but that's not the case. 

*Mitigation*: Because the proposed printer design accepts an `io.Writer`, there is nothing to prevent the writer from being:

```golang
import "text/tabwriter"
...
tw := tabwriter.New(os.Stdout, 0, 8, 0, '\t', 0)
```

Giving `tw` to the printer along with the appropriate Go template ensures that support for tabular output is preserved.

## Design Details

### Test Plan

The test plan involves:

* Matching actual output against expected output
* An automated pipeline that connects `kubeadm init` to `kubeadm join` by parsing the structured output of the former to execute the latter

### Graduation Criteria

This proposal targets *Alpha* support for structured kubeadm output in the release of Kubernetes 1.16.

##### Alpha -> Beta Graduation

* The topic of versioned output is discussed thoroughly by the community and feedback is adapted with changes
* The following commands implement structured output:
  * `alpha certs`
  * `config images list`
  * `init`
  * `token create`
  * `token list`
  * `upgrade plan`
  * `version`
* The feature is maintained by active contributors.
* The feature is tested by the community and feedback is adapted with changes.
* A test pipeline that takes the JSON output of `kubeadm init` in order to join a second node
* Improved documentation

##### Beta -> GA Graduation

* The feature is well tested and adapted by the community.
* E2e test provide sufficient coverage.
* Documentation is complete.

### Upgrade / Downgrade Strategy

NA

### Version Skew Strategy

NA

## Implementation History

* May 2019 (1.14) KEP was created. 

## Drawbacks 

NA
