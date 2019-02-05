---
title: Kustomize Secret Generator Plugins
authors:
  - "@sethpollack"
owning-sig: sig-cli
reviewers:
  - "@monopole"
  - "@Liujingfang1"
approvers:
  - "@monopole"
  - "@Liujingfang1"
editor: "@sethpollack"
creation-date: 2019-02-04
last-updated: 2019-02-04
status: provisional
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

Kustomize removed the `commands` feature from `SecretGenerator` due to security concerns. This proposal is proposing a safe alternative using golang plugins.

## Motivation

The `commands` feature allowed users to use exec to integrate `SecretGenerator` with their secret managment tools, and its removal requires hacky workarounds.

### Goals

- Give users a way to intergrate `SecretGenerator` with their secret managment tools in a safe way.

### Non-Goals

- Kustomize will not handle plugin installation/managment.

## Proposal

In the proposed design `SecretGenerator` would load plugins from `~/.kustomization/plugins`. This would prevent malicious code from being included in a remote config since plugins don't live inside the kustomization directory (bases/overlays etc.).

Having the flexibility to build custom plugins will also allow users to avoid using exec.

The implementation would look something like this:

```
func keyValuesFromPlugins(ldr ifc.Loader, plugins []types.Plugin) ([]kv.KVPair, error) {
	var allKvs []kv.KVPair

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

func findPlugin(name string) (func(string, []string) ([]kv.KVPair, error), error) {
	allPlugins, err := filepath.Glob(os.ExpandEnv("$HOME/.kustomize/plugins/*.so"))
	if err != nil {
		return nil, fmt.Errorf("error loading plugins")
	}

	for _, filename := range allPlugins {
		p, err := plugin.Open(fmt.Sprintf(filename))
		if err != nil {
			return nil, err
		}

		symbol, err := p.Lookup(name)
		if err != nil {
			continue
		}

		if secretFunc, exists := symbol.(func(string, []string) ([]kv.KVPair, error)); exists {
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

func Env(root string, args []string) ([]kv.KVPair, error) {
	var kvs []kv.KVPair
	for _, arg := range args {
		kvs = append(kvs, kv.KVPair{Key: arg, Value: os.Getenv(arg)})
	}
	return kvs, nil
}
```

It may also make sense to remove the other datasources `keyValuesFromEnvFile`, `keyValuesFromFileSources`, `keyValuesFromLiteralSources` in favor of plugins as well.

Users would build the plugin with `go build -buildmode=plugin` and install by copying the .so file to `~/.kustomize/plugins`

### Risks and Mitigations

N/A

## Graduation Criteria

Convert the SecretGenerator into a plugin system.

## Implementation History

This design uses the golang plugin system.
