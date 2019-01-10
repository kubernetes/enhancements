---
kep-number: 31
title: Enable kustomize in kubectl
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
editors:
 - "@pwittrock"
creation-date: 2018-11-07
last-updated: 2019-01-09
status: implementable
see-also:
 - "[kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/workflows.md)"
replaces:
 - "[KEP-0008](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cli/0008-kustomize.md)"
superseded-by:
 - n/a
---

# Enable kustomize in kubectl

## Table of Contents
* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
  * [Goals](#goals)
  * [Non-Goals](#non-goals)
* [Kustomize Introduction](#kustomize-introduction)   
* [Proposal](#proposal)
  * [UX](#UX)
     * [apply](#apply)
     * [get](#get)
     * [delete](#delete)
  * [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Alternatives](#alternatives)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Summary

[Kustomize](https://github.com/kubernetes-sigs/kustomize)
was developed as a subproject of sig-cli by kubectl maintainers with the goal of addressing
a collection of issues creating friction for declarative workflows in kubectl (e.g. `kubectl apply`).  The
goal of Kustomize was to eventually bring this functionality back to kubectl in order to complement
`kubectl apply` and other declarative workflow commands.

- declaratively generating Resource config
- declaratively transforming Resource config
- composing collections of Resource config across files and directories
- layering the above on top of one another

It is independent of, but complementary to, the *server-side apply* initiative that was started later and targeted
at a separate collection of `kubectl apply` issues.

**Note:**
While most of the generation and transformation options supported by Kustomize are available either as
imperative kubectl commands or as kubectl flags, the inverse is not true.  Only a subset of the kubectl
imperative commands are available as declarative options through Kustomize.

The Kustomize implementations of some generators and transformations were augmented to do more intelligent things
when invoked from a declarative workflow involving multiple Resources that may reference one another.
This is a more advanced approach to the imperative workflow, where transformations applied to Resources were largely
independent of one another, and supports scenarios like a ConfigMap and Secret having a generated name-suffix that must be propagated to
Resource Config that references them.

Imperative kubectl commands / flags available through Kustomize:

- `kubectl create configmap`
- `kubectl create secret`
- `kubectl annotate`
- `kubectl label`
- `kubectl patch`
- `-n` (namespace)
- `-f <filename>` (kubectl processes files with lists of Resources)

Kubectl commands / flags similar to what is available through Kustomize:

- `-f <dir> -R` (kubectl - recursing through directories, kustomize may follow references)
- `kubectl set image` (Kustomize directive to set the image tag only, not the image)

Things in Kustomize that are not imperative kubectl commands / flags:

- `namePrefix` (prepend all resource names with this)
- `nameSuffix` (append all resource names with this)
- for a limited set of fields allow one field value to be set to match another

## Motivation

**Background:** Kubectl Apply

Kubectl apply is intended to provide a declarative workflow for working with the Kubernetes API.  Similar to Kustomize,
`apply` (client-side) pre-processes Resource Config and transforms it into Kubernetes patches sent to the
apiserver (transformation is a function of live cluster state and Resource Config).  Apply addresses user friction
such as:

- updating Resources from Resource Config without wiping out control-plane defined fields (e.g. Service clusterIp)
- automatically deciding whether to create, update or delete Resources

**Motivation:**

However apply does contain user friction in its declarative workflow, the majority of which could be either reduced
or solved through augmenting and leveraging capabilities already present in kubectl imperative commands from a
declarative context.  To this end, Kustomize was developed.

Example GitHub issues addressed by Kustomize:

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

Some of the solutions provided by Kustomize could also be done as bash scripts to generate and transform
Resource config using either kubectl or other commands (e.g. creating a Secret from a file).

As an example: [this](https://github.com/osixia/docker-openldap/tree/stable/example/kubernetes/using-secrets/environment)
is a repo publishing Kubernetes Resource config and with it provides a bash script to help facilitate users generating
Secret data from files.  e.g. We want to discourage as a pattern teaching users to run arbitrary bash scripts from
GitHub.

Another example: [this](https://github.com/kubernetes/kubernetes/issues/23233) is suggesting writing a script to
invoke kubectl's patch and apply logic from a script (add-on manager circa 2016).

Users have asked for a declarative script-free way to apply changes to Resource Config.

Others solutions provided by Kustomize require broader context of the set of Resource Config
(e.g. facilitating rolling update of a Secret or ConfigMap).

### Goals

Solve common kubectl user friction (such as those defined in the Motivation section) by publishing Kustomize
functionality from kubectl to complement commands which are targeted at declarative workflows.

- Complement commands targeted at declarative workflows: `kubectl apply`, `kubectl diff`
- Complement the *server-side apply* initiative targeted at improving declarative workflows.

User friction solved through capabilities such as:

- Generating Resource Config for Resources with data frequently populated from other sources - ConfigMap and Secret
- Performing common cross-cutting transformations intended to be applied across Resource Configs - e.g.
 name prefixing / suffixing, labeling, annotating, namespacing, imageTag setting.
- Performing flexible targeted transformations intended to be applied to specific Resource Configs - e.g.
 strategic patch, json patch
- Composing and layering Resource Config and schema-aware transformations
- Facilitating rolling updates to Resources such as ConfigMaps and Secrets
- Facilitating creation of Resources that require the creation to be ordered - e.g. namespaces, crds

### Non-Goals

- Exposing all imperative kubectl commands as declarative options
- Exposing all kubectl generators and schema-aware transformations
- Providing simpler alternatives to the APIs for declaring Resource Config (e.g. a simple way to create deployments)
- Providing a templating mechanism (e.g. for generating Resource Config)
- Publishing with kubectl any other Kustomize sub-commands besides `build`

### Why should this be part of kubectl?

- It was purposefully built to address user friction in kubectl declarative workflows. Leaving these issues unaddressed
 in the command itself reduces the quality of the product.
- The techniques it uses to address the issues are based on the existing kubectl imperative commands.  It is
 consistent with the rest of kubectl.
- Bridging the imperative and declarative workflows helps bring a more integrated feel and consistent story to
 kubectl's command set.
- Kustomize is already part of the Kubernetes project and owned by SIG CLI (the same sig that owns kubectl).
- SIG CLI have expertise in the Kustomize codebase and is committed to maintaining this solution going forward.
- Providing it as part of kubectl will ensure that it is available to users of kubectl apply.

## Proposal

Do one or more of these.

- [ ] A: `kubectl kustomize -f dir/ | kubectl apply -f -`
- [ ] B: `kubectl apply -f dir/`
- [ ] C: `kubectl apply -f dir/kustomization.yaml` (similar to B, but slightly different UX)

### Option A: Kustomize Command

Publish the `kustomize build` command as `kubectl kustomize`.  Update documentation demonstrate using Kustomize as
`kubectl kustomize -f dir/ | kubectl kustomize -f -`.  Consider Option B as a follow up if there is consensus on
the reason for doing so and trade-offs.

#### Why we like this approach

Kustomize does more than the resource builder in kubectl (e.g. the thing that processes files provided with `-f`).

- It is consistent with how tools that produce Resource Config but exist outside kubectl would integrate with
 `kubectl apply`
- It is clear to the user that they are getting a new behavior than when they ran `kubectl apply -f dir/`
 which was relatively restrictive in terms of what it did.
- Publishing it as a separate command keeps what the other kubectl commands do simpler
- We have talked about moving towards this approach for how we teach users to work with imperative generators -
 e.g. `kubectl create deployment -o yaml --dry-run | kubectl apply -f -` (or something that doesn't exist yet like
  - **Note:** this case still has friction (e.g. it defaults fields - such as creationTimestamp to null - yuck)
 `kubectl generate deployment | kubectl apply -f -`)
- It makes more sense to first provide a command (Option A) and then integrate into resource builder (Option B)
  than it does to do the reverse.

#### Why don't like this approach

- It can't address user friction requiring deeper integration - e.g. produce meaningful line numbers in error
  messages.
- Most commands would require the same input pipe - e.g. get, delete, etc would all need pipes
- We haven't trained user with this pattern
 - We don't currently follow this pattern by default for other commands such as `kubectl create deployment`- which requires
   additional flags to output content to pipe `--dry-run -o yaml`
 - We don't do this for file, dir or url targets - e.g. we don't do `curl something | kubectl apply -f -`
- When demoed, the UX was considered more complicated than directly integrating into the resource builder
- It’s more typing for the user
- Kustomize is the way we are addressing issues with declarative workflows so it should be integrated into them.

### Option B / C: Integration into resource builder

Integrate the kustomize preprocessor directly into the resource builder so it is run on kustomization.yaml

#### UX

When apply, get or delete is run on a directory, check if it contains a kustomization.yaml file. If there is, apply,
get or delete the output of kustomize build. Kubectl behaves the same as current for directories without
kustomization.yaml.

##### apply
The command visible to users is
```
kubectl apply -f <dir>
```
To view the objects in a kustomization without applying them to the cluster
```
kubectl apply -f <dir> --dry-run -o yaml|json
```

##### get
The command visible to users is
```
kubectl get -f <dir>
```
To get the detailed objects in a kustomization
```
kubectl get -f <dir> --dry-run -o yaml|json
```

##### delete
The command visible to users is

```
kubectl delete -f <dir>
```

#### Why we like this approach

- It is capable of friction that requires deeper integration - such as producing errors referencing line
  numbers of the original files (rather than the output files).
- It integrates will all commands - get, delete, etc - would automatically work.
- It is more consistent with UX workflow for commands that work directly off a target
- It has a cleaner and simpler UX than Option A, fewer steps
- It integrates the solution directly into the commands with issues instead of tacking it on as separate step
- Users less likely to get confused and accidentally apply a directory with a kustomization.yaml and have
  it do the wrong thing (it will do the right thing).

#### Why don't like this approach

- Kustomize does more than the resource builder currently does and this could surprise users
- Commands are doing more than they were before

### Why not as a kubectl plugin instead of compiled in?

- The kubectl plugin mechanism does not provide a solution for distribution.  Because the functionality is intended as
 the project's solution to issues within kubectl, we want it to be available to users of kubectl without additional
 steps.  Having users manually download only Kustomize as a plugin might be ok, but it won't scale as a good approach
 as the set of commands grows.
- The effort to build and test the tool for all targets, develop a release process, etc. would be much higher for SIG
  CLI, also, and it would exacerbate kubectl's version-skew challenges.
- It will not support integration at more than a surface level - such as into the resource builder
  (which does not offer a plugin mechanism).
    - It was previously decided we didn't want to add a plugin mechanism to the resource builder.
      This could be reconsidered, but would need to think through it more and figure out how to address
      previously brought up issues.  There may be other issues not listed here as well.
      - https://github.com/kubernetes/kubernetes/issues/13241
      - https://github.com/kubernetes/kubernetes/pull/14993
      - https://github.com/kubernetes/kubernetes/pull/14918
- There is a risk that publishing each command as a separately built binary could cause the aggregate download
  size of the toolset to balloon.  The kubectl binary is *52M* and the kustomize binary is *31M*.  (extrapolate to
  30+ commands x 30MB).  Before going down this route, we should consider how to we might want to design a solution
  and the tradeoffs.    

**AI's for kubectl extensibility:**

- Clearly define the tradeoffs of designing kubectl around plugins - what are the benefits and what are the risks.
- Evaluate approaches for distributing designing kubectl around a plugin architecture.
- Evaluate approaches for building, testing, upgrading kubectl around a plugin architecture.
- If feasible - prioritize and possibly come up with a roadmap for supporting these options towards this approach.
- Evaluate good, low risk candidate commands.

## Kustomize Example

Following is an example of a kustomization.yaml file used by Kustomize:

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
commands:
  username: "echo admin"
  password: "echo secret"
```

The result of running `kustomize build` on this sample kustomizaiton.yaml file is:

```
apiVersion: v1
data:
JAVA_HOME: /opt/java/jdk
JAVA_TOOL_OPTIONS: -agentlib:hprof
kind: ConfigMap
metadata:
annotations:
  oncallPager: 800-555-1212
name: alices-myJavaServerEnvVars-7bc9c27cmf
---
apiVersion: v1
data:
password: c2VjcmV0Cg==
username: YWRtaW4K
kind: Secret
metadata:
annotations:
  oncallPager: 800-555-1212
name: alices-app-sec-c7c5tbh526
type: Opaque
```

### Implementation Details/Notes/Constraints

In contrast to `kubectl apply`, which was developed directly in kubernetes/kubernetes and had minimal usage or
real world experience prior, Kustomize was built as a outside of kubernetes/kubernetes in the
kubernetes-sigs/kustomize repo.  After `kubectl apply` was published, many issues were uncovered in it that should
have been discovered earlier.  Implementing Kustomize independently allowed more time for gathering feedback and
identifying issues.

Kustomize library code will be moved from its current repository (location) to the cli-runtime repository used by
kubectl.  This will be started after cli-runtime has been fully moved out of staging.

### Risks and Mitigations

Low:

- Kustomize can be run against remote urls.  A user could run it on a URL containing malicious workflows.  However this
 would only generate the config, and the user would need to pipe it to apply for the workloads to be run.  This is also
 true for `kubectl apply -f <url>` and or `kubectl create deployment --image <bad image>`.

Low:

- Kustomize has other porcelain commands to facilitate common workflows.  This proposal does not include integrating
  them into kubectl.  Users would need to download Kustomize separate to get these benefits.

## Graduation Criteria

The API version for Kustomize is defined in the kustomization.yaml file.  The KEP is targeted `v1beta1`.

The criteria for graduating from `v1beta1` for the Kustomize sub-command should be determined as part of
evaluating the success and maturity of Kustomize as a command within kubectl.

Metrics for success and adoption could include but are not limited to:

- number of kustomization.yaml files seen on sources such as GitHub
- complexity (required) of kustomization.yaml files seen on sources such as GitHub.
- (if available) number of calls to `kubectl kustomize` being performed
- adoption or integration of kustomize by other tools

Metrics for maturity and stability could include but are not limited to:

- number and severity of kustomize related bugs filed that are intended to be fixed
- the frequency of API changes and additions
- understanding of relative use and importance of kustomize features

## Implementation History

Most implementation will be in cli-runtime

- [x] vendor `kustomize/pkg` into kubernetes
- [x] copy `kustomize/k8sdeps` into cli-runtime
- [x] Implement a Visitor for kustomization directory which
 - execute kustomize build to get a list of resources
 - write the output to a StreamVisitor
- [x] When parsing filename parameters in FilenameParam, look for kustomization directories
- [ ] documentation:
 - update the examples in kubectl commands
 - Improve help messages or documentations to list kubectl subcommands that can work with kustomization directories

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

### Leave Kustomize functionality separate from kubectl

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