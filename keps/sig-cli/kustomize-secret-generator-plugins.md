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

```secretGenerator:
- name: env-example
  goplugins:
  - name: Env
    args:
    - EXAMPLE_ENV
    - OTHER_EXAMPLE_ENV
  type: Opaque
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

No windows support for golang plugins.

## Graduation Criteria

Many users have been requesting a better way to integrate with secret management tools https://github.com/kubernetes-sigs/kustomize/issues/692

* A higher level feature test (like those in [pkg/target](https://github.com/kubernetes-sigs/kustomize/tree/master/pkg/target))
* Documentation of fields in the [canonical example file](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/kustomization.yaml)
* Usage [example](https://github.com/kubernetes-sigs/kustomize/tree/master/examples)

## Implementation History

This design uses the golang [plugin](https://golang.org/pkg/plugin/) system.
