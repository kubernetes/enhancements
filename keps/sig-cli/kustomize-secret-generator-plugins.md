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

# Kustomize Secret Generator Plugins

## Table of Contents
* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)

## Summary

Kustomize wants to generate Kubernetes Secret objects from general key-value pairs where the value is supposed to be a secret. Currently Kustomize only supports reading secret values from local files which raises security concerns about file lifetime and access. Reading secret values from the execution of arbitrary "commands" in a kustomization.yaml file introduces concerns in a world where kustomization files can be used directly from the internet when a user runs `kustomize build`.

This proposal describes obtaining secret values safely using golang [plugins](https://golang.org/pkg/plugin/).

## Motivation

Not having a way to integrate `SecretGenerator` with secret management tools requires hacky workarounds.

### Goals

- Give users a way to intergrate `SecretGenerator` with secret management tools in a safe way.

### Non-Goals

- Kustomize will not handle plugin installation/management.

## Proposal

In the proposed design `SecretGenerator` would load golang plugins from `~/.config/kustomization_plugins`. This would limit the scope of what kustomize can execute to the plugins on the users machine. Also using golang plugins forces a strict interface with well-defined responsibilities.

The current API looks like this:

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
    name: someLocalGoPlugin
    args:
    - someArg
    - someOtherArg
  - pluginType: kubectl      // some future KEP can write this
    name: someKubectlPlugin
    args:
    - anotherArg
    - yetAnotherArg
```

The implementation would look something like this:

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

and an example plugin would look like this:

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

Users would install the plugin with `go build -buildmode=plugin ~/.config/kustomization_plugins/myplugin.so`

### Risks and Mitigations

#### Several people want an _exec-style_ plugin
_exec-style_ means execute arbitrary code from some file.

Most the lines of code written to implement this KEP
accomodate the notion of KV generation via a _generic_
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
are out of scope for this KEP, however this KEP's
implementation will make it much easier to create other
kinds of plugins.

Go Plugins that become popular have a clear
path to becoming a builting - so if someone writes, say,
a general Exec plugin, we can easily promote it to a
builtin (by virtue of the fact that it's written in Go,
and because of the choices made in this KEP for
describing a plugin in the kustomization file).

#### No current windows support for golang plugins.
This may be implemented by the Go team later in which
case it will just work.

Also, as noted above, someone can write an exec style plugin
which will work on Windows.

#### General symbol exection from the plugin
A Go plugin will be cast to a particular hardcoded interface, e.g.

```
type KvSourcePlugin {
	Get []kvPairs
}
```
and accessed exclusively through that interface.

#### Existing KV generators continue as an option
We leave in place the existing three mechanisms
(_literals_, _env_ and _files_) for generating KV pairs, but
additionally allow these mechanisms to be expressed as
`builtin` plugins (see example above).

If plugins - both Go and other styles - prove unpopular
or problematic, we can remove them per API change
policies.

If they do prove popular/useful, we can remove deprecate
the legacy form for (_literals_, _env_ and _files_),
and help people convert to the new "builtin" form.

## Graduation Criteria

Many users have been requesting a better way to integrate with secret management tools https://github.com/kubernetes-sigs/kustomize/issues/692

* A higher level feature test (like those in [pkg/target](https://github.com/kubernetes-sigs/kustomize/tree/master/pkg/target))
* Documentation of fields in the [canonical example file](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/kustomization.yaml)
* Usage [example](https://github.com/kubernetes-sigs/kustomize/tree/master/examples)

## Implementation History

This design uses the golang [plugin](https://golang.org/pkg/plugin/) system.
