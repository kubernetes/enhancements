# Ensure consistency in usage of flags across `kubectl` commands

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Ensure consistency when working with `kubectl` commands that feature flags which do not require explicit value specification.

## Motivation

This proposal aims to enhance the user experience when utilizing `kubectl` commands that involve flags without mandatory value assignments. Within this project, the [`NoOptDefVal`][no-opt-def-val] option has frequently been employed to establish default values when a flag is present in a command without an explicitly provided value. Unfortunately, this approach has led to inconsistencies where, in certain situations, the command incorrectly evaluates flag values (examples are presented further in the proposal).

To address this issue, the propsal introduces a requirement that mandates explicit value assignments for all flags. This adjustment will enable the tool to consistently perform the expected actions across all commands, resulting in a more reliable user experience.

### Goals

- Enforce all commands to accept flags as key-value pairs, with a default value if not set.
- Remove the use of `NoOptDefVal` from `kubectl` project.

## Proposal

`NoOptDefVal` is an option provided by the `spf13/pflag` [package][pflag-repo], which is used to define a default value for a flag when the flag is present in a command but is not followed by an explicit value assignment. 

An example usage:

```
kubectl config set-cluster e2e --insecure-skip-tls-verify
```

In this case, `insecure-skip-tls-verify` is a boolean [flag][k-set-cluster-cmd]. Even though, its value is not explicitly provided, it is assumed to be true since the flag is set. 

Without enabling this option, the command accepts input in two formats: `<flag_name>=<value> `and `<flag_name> <value>.` However, with this option enabled, only the `<flag_name>=<value>` format is accepted. For instance:

```
kubectl create deploy --image nginx test
```

creates a deployment with image `nginx` and name `test`, whereas:

```
kubectl create deploy --image nginx --validate strict
```

creates a deployment with image `nginx` and name `strict`. 

A few other examples of misintepretation as shared in issue [#1442][issue-1442] by @salavessa are:

1. `kubectl delete pod my-sts --cascade orphan --dry-run server`

Expected: Delete pod with the name `my-sts` with cascade set to `orphan` and `dry-run` to server.

Actual: Tries deleting 3 pods: `my-sts`, `orphan` and `server`.
```
W0922 15:21:03.455729   68481 helpers.go:663] --dry-run is deprecated and can be replaced with --dry-run=client.
pod "my-sts" deleted (dry run)
pod "orphan" deleted (dry run)
pod "server" deleted (dry run)
```

2. `kubectl delete pods my-sts --cascade orphan --dry-run=server`

Expected: Delete pod with the name `my-sts` with cascade set to `orphan` and `dry-run` to server.

Actual: Tries deleting 2 pods: `my-sts` and `orphan`.

```
pod "my-sts" deleted (dry run)
Error from server (NotFound): pod "orphan" not found
```

Removing the use `NoOptDefVal` will help us in providing consistency across the use flags of all types. With this option removed, the user experience would look like:

1. `kubectl delete pod my-sts --cascade orphan --dry-run server`: Delete pod with the name `my-sts` with cascade set to `orphan` and `dry-run` to server.
2. `kubectl delete pod my-sts --cascade=orphan --dry-run=server`: Delete pod with the name `my-sts` with cascade set to `orphan` and `dry-run` to server.
3. `kubectl delete pods my-sts --cascade --dry-run=server`: Would error stating `--cascade` flag needs a value.
4. `kubectl delete pods my-sts --cascade --dry-run`: Would error stating `--cascade` and `--dry-run` flags need values.

Though this would enforce users to specify flags as key value pairs, it would ensure that the command behaves the same whether or not `=` or `space` is provided between the flags and their values.


### Risks and Mitigations

This may break existing users that rely on the feature of `NoOptDefVal` and specify just the flags assuming that their value would be set to `true`.

## Design Details

In order to implement this, the use of `NoOptDefVal` needs to be deprecated and removed from the following places.

| Flag Name                 | Command                | Default if not present        | Default if present     |
| :------------------------ | :--------------------  | :---------------------------- | :--------------------  |
| InsecureSkipTLSVerify     | config set-cluster     | False                         | True                   |
| embed-certs               | config set-cluster     | False                         | True                   |
| set-raw-bytes             | config set             | False                         | True                   |
| merge                     | config view            | False                         | True                   |
| cascade                   | delete                 | background                    | background             |
| validate                  | create                 | strict (true)                 | strict (true)          |
| dry-run (bool-deprecated) | create                 | none                          | client                 |


To implement this change in a user-friendly and gradual manner, we can follow the deprecation and removal process, similar to the `--dry-run` flag, as follows:

Steps:
1. Begin by deprecating the flags that are used without explicit values. When a user specifies such flags, we issue a deprecation warning, encouraging them to provide a value explicitly. 

```go
var insecureSkipTLSVerifyFlag = GetFlagString(cmd, "insecure-skip-tls-verify")
	b, err := strconv.ParseBool(insecureSkipTLSVerifyFlag)
	// The flag is not a boolean
	if err != nil {
    if insecureSkipTLSVerifyFlag == cmd.Flag("insecure-skip-tls-verify").NoOptDefVal {
      klog.Warning(`insecure-skip-tls-verify is deprecated. The flag needs to be specified with value, e.g: insecure-skip-tls-verify=true.`)
			return true, nil
    }
    // Handle other cases, e.g., propagate the value accordingly.
	}
```

2. Define a deprecation Period and update documentation.

We would need to discuss and come to a consensus on a deprecation period for this behaviour. 


### Test Plan

Add additional tests and modify existing ones based on the changes made.

### Graduation Criteria

NA

### Upgrade / Downgrade Strategy

NA

### Version Skew Strategy

NA

## Implementation History

- *2023-09*: Added KEP


[no-opt-def-val]: https://pkg.go.dev/github.com/spf13/pflag#readme-setting-no-option-default-values-for-flags
[pflag-repo]: https://github.com/spf13/pflag
[k-set-cluster-cmd]: https://jamesdefabia.github.io/docs/user-guide/kubectl/kubectl_config_set-cluster/
[issue-1442]: https://github.com/kubernetes/kubectl/issues/1442