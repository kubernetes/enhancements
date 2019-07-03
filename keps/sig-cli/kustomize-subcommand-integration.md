---
title: Kustomize Subcommand Integration
authors:
  - "@Liujingfang1"
owning-sig: sig-cli
participating-sigs:
  - sig-cli
reviewers:
  - "@liggitt"
  - "@seans3"
  - "@soltysh"
approvers:
  - "@liggitt"
  - "@seans3"
  - "@soltysh"
editor: "@pwittrock"
creation-date: 2018-11-07
last-updated: 2019-02-15
status: implemented
see-also:
  - "[kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/workflows.md)"
  - "kustomize-file-processing-integration.md"
replaces:
  - "0008-kustomize.md"
superseded-by:
  - n/a
---

# Kustomize Subcommand Integration

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [Why this should be part of kubectl](#why-this-should-be-part-of-kubectl)
- [Proposal](#proposal)
  - [Justification for this approach](#justification-for-this-approach)
  - [Justification for follow up](#justification-for-follow-up)
- [Kustomize Example](#kustomize-example)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Copy kustomize code into staging](#copy-kustomize-code-into-staging)
  - [Leave kustomize functionality separate from kubectl](#leave-kustomize-functionality-separate-from-kubectl)
  - [Build a separate tools targeted at Kubernetes declarative workflows.](#build-a-separate-tools-targeted-at-kubernetes-declarative-workflows)
<!-- /toc -->

## Summary

[Kustomize](https://github.com/kubernetes-sigs/kustomize)
was developed as a subproject of sig-cli by kubectl maintainers to address
a collection of [issues](#motivation)) creating friction for declarative workflows in kubectl
(e.g. `kubectl apply`).  The
goal of the kustomize subproject was to bring this functionality back to kubectl to better complement
`kubectl apply` and other declarative workflow commands.

- declaratively generating Resource config
- declaratively transforming Resource config
- composing collections of Resource config across files and directories
- layering the above on top of one another

It is independent of, but complementary to, the [*server-side apply*](https://github.com/kubernetes/enhancements/issues/555)
initiative that was started later and targeted at a separate collection of
`kubectl apply` issues.

Kustomize offers generators and transformations in a declarative form that 
improve on functionality provided by existing imperative commands in kubectl.

The declarative approach offers a clear path to accountability (all input can
be kept in version control), can safely exploit a holistic, unbounded view of
disparate resources and their interdependence (it's a plan about what to do, 
not a direct action), and can be easily constrained to verifiable rules 
across this view (all edits must be structured, no removal semantics, no 
environment side-effects, etc.).

Imperative kubectl commands / flags available through kustomize:

- `kubectl create configmap`
- `kubectl create secret`
- `kubectl annotate`
- `kubectl label`
- `kubectl patch`
- `-n` (namespace)
- `-f <filename>` (kubectl processes files with lists of Resources)

Kubectl commands / flags similar to what is available through kustomize:

- `-f <dir> -R` (kubectl - recursing through directories, kustomize may follow references)
- `kubectl set image` (kustomize directive to set the image tag only, not the image)

Things in kustomize that are not imperative kubectl commands / flags:

- `namePrefix` (prepend all resource names with this)
- `nameSuffix` (append all resource names with this)
- for a limited set of fields allow one field value to be set to match another

## Motivation

**Background:** Kubectl Apply

Kubectl apply is intended to provide a declarative workflow for working with the Kubernetes API.  Similar to kustomize,
`apply` (client-side) pre-processes Resource Config and transforms it into Kubernetes patches sent to the
apiserver (transformation is a function of live cluster state and Resource Config).  Apply addresses user friction
such as:

- updating Resources from Resource Config without wiping out control-plane defined fields (e.g. Service clusterIp)
- automatically deciding whether to create, update or delete Resources

**Present:**

However apply does contain user friction in its declarative workflow, the majority of which could be either reduced
or solved through augmenting and leveraging capabilities already present in kubectl imperative commands from a
declarative context.  To this end, kustomize was developed.

Example GitHub issues addressed by kustomize:

- Declarative Updates of Secrets + ConfigMaps
 [kubernetes/kubernetes#24744](https://github.com/kubernetes/kubernetes/issues/24744)
- Declarative Updates of ConfigMaps (duplicate)
 [kubernetes/kubernetes#30337](https://github.com/kubernetes/kubernetes/issues/30337)
- Collecting Resource Config Across multiple targets (e.g. files / urls)
 [kubernetes/kubernetes#24649](https://github.com/kubernetes/kubernetes/issues/24649)
- Facilitate Rollouts of ConfigMaps on update
 [kubernetes/kubernetes#22368](https://github.com/kubernetes/kubernetes/issues/22368)
- Transformation (and propagation) of Names, Labels, Selectors
 [kubernetes/kubernetes#1698](https://github.com/kubernetes/kubernetes/issues/1698)

Some of the solutions provided by kustomize could also be done as bash scripts to generate and transform
Resource config using either kubectl or other commands (e.g. creating a Secret from a file).

As an example: [this](https://github.com/osixia/docker-openldap/tree/stable/example/kubernetes/using-secrets/environment)
is a repo publishing Kubernetes Resource config and with it provides a bash script to help facilitate users generating
Secret data from files.  e.g. We want to discourage as a pattern teaching users to run arbitrary bash scripts from
GitHub.

Another example: [this](https://github.com/kubernetes/kubernetes/issues/23233) is suggesting writing a script to
invoke kubectl's patch and apply logic from a script (add-on manager circa 2016).

Users have asked for a declarative script-free way to apply changes to Resource Config.

### Goals

Solve common kubectl user friction (such as those defined in the Motivation section) by publishing kustomize
functionality from kubectl to complement commands which are targeted at declarative workflows.

- Complement commands targeted at declarative workflows: `kubectl apply`, `kubectl diff`
- Complement the *server-side apply* initiative targeted at improving declarative workflows.

User friction solved through capabilities such as:

- Generating Resource Config for Resources with data frequently populated from other sources - ConfigMap and Secret
- Performing common cross-cutting transformations intended to be applied across Resource Configs - e.g.
  name prefixing / suffixing, labeling, annotating, namespacing, imageTag setting.
- Performing flexible targeted transformations intended to be applied to specific Resource Configs - e.g.
  strategic patch, json patch
- Composing and layering Resource Config with schema-aware transformations
- Facilitating rolling updates to Resources such as ConfigMaps and Secrets via Resource Config transformations
- Facilitating creation of Resources that require the creation to be ordered - e.g. namespaces, crds

### Non-Goals

- Exposing all imperative kubectl commands as declarative options
- Exposing all kubectl generators and schema-aware transformations
- Providing simpler alternatives to the APIs for declaring Resource Config (e.g. a simple way to create deployments)
- Providing a templating or general substitution mechanism (e.g. for generating Resource Config)

### Why this should be part of kubectl

- It was purposefully built to address user friction in kubectl declarative workflows. Leaving these issues unaddressed
  in the command itself reduces the quality of the product.
- The techniques it uses to address the issues are based on the existing kubectl imperative commands.  It is
  consistent with the rest of kubectl.
- Bridging the imperative and declarative workflows helps bring a more integrated feel and consistent story to
  kubectl's command set.
- Kustomize is already part of the Kubernetes project and owned by SIG CLI (the same sig that owns kubectl).
  SIG CLI members have expertise in the kustomize codebase and are committed to maintaining the solution going forward.
- Providing it as part of kubectl will ensure that it is available to users of kubectl apply and simplify the
  getting started experience.

## Proposal

Kustomize Standalone Sub Command

Publish the `kustomize build` command as `kubectl kustomize`.  Update 
documentation to demonstrate using kustomize as `kubectl kustomize <dir> | kubectl apply -f -`.

`kubectl kustomize` takes a single argument with is the location of a directory containing a file named `kustomization.yaml`
and writes to stdout the kustomized Resource Config.

If the directory does not contain a `kustomization.yaml` file, it returns an 
error.

Defer deeper integration into ResourceBuilder (e.g. `kubectl apply -k <dir>`) as a follow up after discussing
the precise UX and technical tradeoffs.  (i.e. deeper integration is more delicate and can be done independently.)

### Justification for this approach

The end goal is to have kustomize fully integrated into cli-runtime as part of the Resource processing
libraries (e.g. ResourceBuilder) and supported by all commands that take the `-f` flag.

Introducing it as a standalone subcommand first is a simple way of introducing kustomize functionality to kubectl,
and will be desirable even after kustomize is integrated into cli-runtime.  The benefits of introducing it
as a standalone command are:

- Users can view the kustomize output (i.e. without invoking other commands in dry-run mode)
  - Allows users to experiment with kustomizations
  - Allows users to debug kustomizations
  - It is easier to educate users about kustomizations when they can run it independently
- The subcommand UX is simple and well defined
- The technical integration is simple and well defined

### Justification for follow up

- It can't address user friction requiring deeper integration - e.g. produce meaningful line numbers in error
  messages and exit codes.
- Most commands would require the same input pipe - e.g. get, delete, etc. would all need pipes.  Direct integration
  is cleaner than always piping everything.
- We haven't trained user with this pattern
 - We don't currently follow this pattern by default for other commands such as `kubectl create deployment`- which requires
   additional flags to output content to pipe `--dry-run -o yaml`
 - We don't do this for file, dir or url targets - e.g. we don't do `curl something | kubectl apply -f -`
- When both subcommand and ResourceBuilder integration were demoed at sig-cli, the integrated UX was preferred
- Kustomize is the way we are addressing issues with declarative workflows so we should make it as simple and easy
  to use as raw Resource Config files.

## Kustomize Example

Following is an example of a `kustomization.yaml` file used by kustomize:

```
apiVersion: v1beta1
kind: Kustomization
namePrefix: alices-

commonAnnotations:
  oncallPager: 800-555-1212

configMapGenerator:
- name: myJavaServerEnvVars
  literals:  
  - JAVA_HOME=/opt/java/jdk
  - JAVA_TOOL_OPTIONS=-agentlib:hprof

secretGenerator:
- name: app-sec
  files:
  - secret/password
  - secret/username
```

The result of running `kustomize build` on this sample kustomizaiton.yaml file is:

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: alices-myJavaServerEnvVars-7bc9c27cmf
  annotations:
    oncallPager: 800-555-1212
data:
  JAVA_HOME: /opt/java/jdk
  JAVA_TOOL_OPTIONS: -agentlib:hprof
---
apiVersion: v1
kind: Secret
metadata:
  name: alices-app-sec-c7c5tbh526
  annotations:
    oncallPager: 800-555-1212
type: Opaque
data:
  password: c2VjcmV0Cg==
  username: YWRtaW4K
```

### Implementation Details/Notes/Constraints

In contrast to `kubectl apply`, which was developed directly in kubernetes/kubernetes and had minimal usage or
real world experience prior, kustomize was built as an outside of kubernetes/kubernetes in the
kubernetes-sigs/kustomize repo.  After `kubectl apply` was published, many issues were uncovered in it that should
have been discovered earlier.  Implementing kustomize independently allowed more time for gathering feedback and
identifying issues.

Kustomize library code will be moved from its current repository (location) to the cli-runtime repository used by
kubectl.

### Risks and Mitigations

Low:

- Kustomize can be run against remote urls.  A user could run it on a URL containing malicious workflows.  However this
 would only generate the config, and the user would need to pipe it to apply for the workloads to be run.  This is also
 true for `kubectl apply -f <url>` and or `kubectl create deployment --image <bad image>`.

Low:

- Kustomize has other porcelain commands to facilitate common workflows.  This proposal does not include integrating
  them into kubectl.  Users would need to download kustomize separate to get these benefits.
  
Low:

- `kubectl kustomize <dir>` doesn't take a `-f` flag like the other commands.

## Graduation Criteria

The API version for kustomize is defined in the `kustomization.yaml` file.  The KEP is targeted `v1beta1`.

The criteria for graduating from `v1beta1` for the kustomize sub-command should be determined as part of
evaluating the success and maturity of kustomize as a command within kubectl.

Metrics for success and adoption could include but are not limited to:

- number of `kustomization.yaml` files seen on sources such as GitHub
- complexity (required) of `kustomization.yaml` files seen on sources such as GitHub.
- (if available) number of calls to `kubectl kustomize` being performed
- adoption or integration of kustomize by other tools

Metrics for maturity and stability could include but are not limited to:

- number and severity of kustomize related bugs filed that are intended to be fixed
- the frequency of API changes and additions
- understanding of relative use and importance of kustomize features

**Note:** Being integrated into ResourceBuilder is *not* considered graduation and *not* gated on GA.

## Implementation History

Most implementation will be in cli-runtime

- [ ] vendor `kustomize/pkg` into kubernetes
- [ ] copy `kustomize/k8sdeps` into cli-runtime
  - Once cli-runtime is out of k/k, move the kustomize libraries there (but 
  not the commands)
- [ ] Implement a function in cli-runtime to run kustomize build with input as fSys and/or path. 
   - execute kustomize build to get a list of resources
   - write the output to io.Writer
- [ ] Add a subcommand `kustomize` in kubectl. This command accepts one argument <dir> and write the output to stdout
      kubectl kustomize <dir>
- [ ] documentation:
  - Write full doc for `kubectl kustomize`
  - Update the examples in kubectl apply/delete to include the usage of kustomize
  
## Alternatives

The approaches in this section are considered, but rejected.

### Copy kustomize code into staging

Don't wait until kubectl libraries are moved out of staging before integrating, immediately copy kustomize code into
kubernetes/staging and move to this as the source of truth.

- Pros
 - It could immeidately live with the other kubectl code owned by sig-cli, instead of waiting until this
   is moved out.
- Cons
 - We are trying to move code **out** of kubernetes/kubernetes.  This is doing the opposite.
 - We are trying to move kubectl **out** of kubernetes/kubernetes.  This is doing the opposite.

### Leave kustomize functionality separate from kubectl

- Pros
  - It is (marginally) less engineering work
- Cons
  - It leaves long standing issues in kubectl unaddressed within the tool itself.
  - It does not support any deeper integrations - such as giving error messages with meaningful line numbers.
  
### Build a separate tools targeted at Kubernetes declarative workflows.

Copy the declarative code from kubectl into a new tool.  Use this for declarative workflows.

Questions:

- Do we deprecate / remove it from kubectl or have it in both places?
- If in both places do we need to support both?  Can they diverge?
- What needs to be updated?  Docs, Blogs, etc.

- Pros
  - It makes kubectl simpler
- Cons
  - Not clear how this helps users
  - Does't address distribution problems
  - User friction around duplication of functionality or remove of functionality