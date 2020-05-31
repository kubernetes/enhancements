---
title: Kustomize Secret Generator Plugins
authors:
  - "@sethpollack"
owning-sig: sig-cli
participating-sigs:
  - sig-apps
  - sig-architecture
reviewers:
  - "@monopole"
  - "@Liujingfang1"
approvers:
  - "@monopole"
  - "@Liujingfang1"
  - "@pwittrock"
editor: "@sethpollack"
creation-date: 2019-02-04
last-updated: 2019-02-04
status: implementable
---

[execRemoval]: https://github.com/kubernetes-sigs/kustomize/issues/692

# Kustomize Secret K:V Generator Plugins

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Risks and Mitigations](#risks-and-mitigations)
  - [Several people want an <em>exec-style</em> plugin](#several-people-want-an-exec-style-plugin)
      - [mitigation](#mitigation)
  - [goplugin limitations](#goplugin-limitations)
    - [Not shareable as object code](#not-shareable-as-object-code)
      - [mitigation](#mitigation-1)
      - [No current support for Windows](#no-current-support-for-windows)
      - [mitigation](#mitigation-2)
    - [General symbol execution from the plugin](#general-symbol-execution-from-the-plugin)
      - [mitigation](#mitigation-3)
  - [Two means to specify legacy KV generation](#two-means-to-specify-legacy-kv-generation)
      - [mitigation](#mitigation-4)
- [Graduation Criteria of plugin framework](#graduation-criteria-of-plugin-framework)
  - [Alpha status](#alpha-status)
  - [Graduation to beta](#graduation-to-beta)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Kustomize users want to generate Kubernetes Secret
objects from general key:value (KV) pairs where the
value is supposed to be a secret.  Users want to
generate these pairs through integration with secret
management tools (e.g. see comments on
[692][execRemoval]).

Currently Kustomize only supports reading secret values
from local files which raises security concerns about
file lifetime and access. Reading secret values from
the execution of arbitrary "commands" in a
kustomization.yaml file introduces concerns in a world
where kustomization files can be used directly from the
internet when a user runs `kustomize build`.  This
proposal describes the syntax for a new key:value
generator plugin framework supporting an arbitrary
number of plugin types to generate key:value pairs.

## Motivation

Not having a way to integrate `SecretGenerator` with
secret management tools requires hacky, insecure
workarounds.

### Goals

- In the `GeneratorArgs` section of a kustomization
  file, a user may specify a plugin type, and a
  specific instance of that type, for generating
  key:value pairs.

- The specification will allow for any number of plugin
  types, and any number of instances of those types.
  
- The first type supported will be
  [goplugins](https://golang.org/pkg/plugin), to enable
  kustomize source code contributors to add custom KV
  generators without the need to maintain a kustomize
  source code fork.
  
  Kustomize maintainers expect developers who use a
  goplugin to understand that a kustomize binary and
  any goplugins expected to work with it must be
  compiled on the same machine with the same compiler
  against the same set of (transitive) libraries.
  _It's the developers responsibility to bundle the
  binary and shared libraries together in a container
  image for actual use._

- Other kinds of plugins, e.g. an _execute this binary_
  plugin, should be subsequently easy to add, and could
  be appropriate for end user use (but would require
  consideration in a KEP first).


### Non-Goals

- Kustomize will not handle key:value generation plugin
  installation/management, or seek to build a plugin
  "ecosystem" of key:value generators.


## Proposal

The current API of `SecretGenerator` looks like this:

```
secretGenerator:
- name: app-tls
  files:
  - secret/tls.cert
  - secret/tls.key
  type: "kubernetes.io/tls"

- name: env_file_secret
  env: env.txt
  type: Opaque

- name: myJavaServerEnvVars
  literals:
  - JAVA_HOME=/opt/java/jdk
  - JAVA_TOOL_OPTIONS=-agentlib:hprof
```

The proposed API would look like this:

```
secretGenerator:
- name: app-tls
  kvSources:
  - pluginType: builtin  // builtin is the default value of pluginType
    name: files
    args:
    - secret/tls.cert
    - secret/tls.key
- name: env_file_secret
  kvSources:
  - name: env  // this is a builtin
    args:
    - env.txt
- name: myJavaServerEnvVars
  kvSources:
  - name: literals    // this is a builtin
    args:
    - JAVA_HOME=/opt/java/jdk
    - JAVA_TOOL_OPTIONS=-agentlib:hprof
- name: secretFromPlugins
  kvSources:
  - pluginType: go      // described by this KEP
    name: myplugin
    args:
    - someArg
    - someOtherArg
  - pluginType: kubectl      // some future KEP can write this
    name: someKubectlPlugin
    args:
    - anotherArg
    - yetAnotherArg
```


The `kvSources` specified with `pluginType: builtin` are
reformulations of existing key:value generators currently
invoked by the existing `dataSources` specification.

A `kvSource` with `pluginType: Go` and `name: myplugin`
would attempt to load the file

```
~/.config/kustomize/plugins/kvSources/myplugin.so
```

and access the loaded plugin via the interface

```
type KvSourcePlugin interface {
	Get() []kv.Pair
}
```

This clearly describes how the plugin must be formulated.


The loading implementation would look something like this:

```
func keyValuesFromPlugins(ldr ifc.Loader, plugins []types.Plugin) ([]kv.Pair, error) {
	var allKvs []kv.Pair

	for _, plugin := range plugins {
		secretFunc, err := findPlugin(plugin.Name)
		if err != nil {
			return nil, err
		}

		kvs, err := secretFunc(ldr.Root(), plugin.Args)
		if err != nil {
			return nil, err
		}

		allKvs = append(allKvs, kvs...)
	}

	return allKvs, nil
}

func findPlugin(name string) (func(string, []string) ([]kv.Pair, error), error) {
	allPlugins, err := filepath.Glob(os.ExpandEnv("$HOME/.config/kustomization_plugins/*.so"))
	if err != nil {
		return nil, fmt.Errorf("error loading plugins")
	}

	for _, filename := range allPlugins {
		p, err := plugin.Open(filename)
		if err != nil {
			return nil, err
		}

		symbol, err := p.Lookup(name)
		if err != nil {
			continue
		}

		if secretFunc, exists := symbol.(func(string, []string) ([]kv.Pair, error)); exists {
			return secretFunc, nil
		}
	}

	return nil, fmt.Errorf("plugin %s not found", name)
}
```

An example plugin would look like this:

```
package main

import (
	"os"

	"sigs.k8s.io/kustomize/pkg/kv"
)

func Env(root string, args []string) ([]kv.Pair, error) {
	var kvs []kv.Pair
	for _, arg := range args {
		kvs = append(kvs, kv.Pair{Key: arg, Value: os.Getenv(arg)})
	}
	return kvs, nil
}
```


A developer - not an end user - would compile the
goplugin and main program like this:

```
go build -buildmode=plugin ~/.config/kustomize/plugins/kvSources/myplugin.so
go get sigs.k8s.io/kubernetes-sigs/kustomize
```

then bake the build artifacts into a container image
for use by an end user or continuous delivery bot.


## Risks and Mitigations

### Several people want an _exec-style_ plugin

_exec-style_ means execute arbitrary code from some file,
with key value pairs being captured from the `stdout`
of a subprocess.

##### mitigation

Most the lines of code written to implement this KEP
accommodate the notion of KV generation via a _generic_
notion of a plugin, and a generic interface, in the
kustomization file and associated structs and code.

This code recharactizes the three existing KV
generation methods as `pluginType: builtin`
(_literals_, _env_ and _files_), introduces the new
`pluginType: Go`, and leaves room for someone to easily
add, say `pluginType: kubectl` to look for a kubectl
plugin, and `pluginType: whatever` to handles some
completely new style, e.g. look for the plugin name as
an executable in some hardcoded location.

Actual implementation of these other kinds of plugins
are _out of scope for this KEP_.

The ability to write a goplugin very specifically
targets a kustomize contributor who understands their
limitations.  An exec-style plugin would enable a
broader set of people, this time including end users.
A KEP proposing such plugins would need to consider a
different set of risks and maintenance requirements.

### goplugin limitations

The first plugin mechanism will be goplugins, since by
design they are trivial to make for a Go program (like
kustomize).  However, they have limitations, and are
to some extent an experimental feature of Go tooling.

#### Not shareable as object code

The shared object (`.so`) files created via `go build
-buildmode=plugin` cannot be reliably used by a loading
program unless both the program and the plugin were
compiled with the same version of Go, with the same
transitive libs, on the same machine (see, e.g. [this
golang
issue](https://github.com/golang/go/issues/18827)).

Therefore, the Go developer who elects to write their
key:value generator as a goplugin is obligated to compile a
new kustomize binary as well as the plugin.

For this developer, this extra compilation step will be far
easier than forking kustomize and modifying the code
directly to introduce all the piping one would need to
express a new key:value secret generation style in a
kustomization file and integrate it with existing methods.

The risk to not compiling, and, say using an `.so` that was
compiled long ago and far away is that the program will
panic when the `build` command is executed.

##### mitigation

An attempt to `kustomize build $target` that includes a
kustomization file specifying a `pluginType: Go` will
fail with an error message explaining the panic risk
and demanding the additional use of the command line
flag:

```
kustomize build --alpha_enable_goplugin_and_accept_panic_risk $target
```

This flag signals that the feature 1) is alpha (it
could be removed), and 2) the feature could panic if
improperly used.

As noted above, developers who elect to use kustomize
plus goplugins should bundle all compilation artifacts
into a container image for end use.  A bot using said
image would always specify the flag.

##### No current support for Windows

This is mentioned in the
[plugin package overview section](https://golang.org/pkg/plugin).

##### mitigation

The target use case for secret generation is gitops
running in a container on any kubernetes cluster.

Developers on their workstations are not a target use
case, as they are less likely to have the security
requirements that necessitate using a plugin based
generator.

If developers need this functionality, we need to
better understand why. it is possible there is a better
way of addressing their need. Of course, Windows users
who want to support goplugins could contribute to the
golang project.

#### General symbol execution from the plugin

##### mitigation

A Go plugin, when loaded,
will be cast to a particular hardcoded interface, e.g.

```
type KvSourcePlugin interface {
	Get() []kv.Pair
}
```
and accessed exclusively through that interface.

### Two means to specify legacy KV generation

The existing three mechanisms and syntax for generating KV
pairs (_literals_, _env_ and _files_) remain active, but
additionally the work for this KEP will allow these
mechanisms to be expressed as `builtin` plugins (see example
above).

##### mitigation

If plugins - both goplugins and other styles - prove
unpopular or problematic, we can remove them per API change
policies.

If they do prove popular/useful, we can deprecate the legacy
form for (_literals_, _env_ and _files_), and help people
convert to the new `builtin` form to access these KV sources.

## Graduation Criteria of plugin framework

### Alpha status 

The kustomization fields that support a general plugin
framework (which could support many kinds of plugins in
addition to goplugins) will be implemented but
documented as an alpha feature, which could be removed
per API change policies. Loading of goplugins
themselves, as noted above, will be protected by a flag
denoting their alpha status.  As these features
target contributors, this will be mentioned
in CONTRIBUTING.md.

### Graduation to beta

* Exec-style plugins approaching GA.

  As goplugins are targeted to kustomize contributors,
  we'd like to see development of an exec-style plugin
  targeted to end users before deciding to graduate
  the framework to beta/GA.
  
  For goplugins themselves to reach beta/GA, we'd
  like exec-style based plugin implemented and still
  see some preference for the Go based approach.

* Testing and documentation.

  * High level feature test
    (like those in [pkg/target](https://github.com/kubernetes-sigs/kustomize/tree/master/api/internal/target))
  * Field documentarion in the 
    [canonical example file](https://github.com/kubernetes-sigs/kustomize/blob/master/examples/helloWorld/kustomization.yaml)
  * Usage [examples](https://github.com/kubernetes-sigs/kustomize/tree/master/examples).


## Implementation History

(TODO add PR's here)
