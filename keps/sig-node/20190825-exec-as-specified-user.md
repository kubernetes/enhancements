---
title: Exec as specified user
authors:
  - "@max8899"
owning-sig: sig-node
participating-sigs:
  - sig-cli
  - sig-auth
reviewers:
  - "@smarterclayton"
  - "@dchen1107"
approvers:
  - TBD
editor: TBD
creation-date: 2019-08-25
last-updated: 2019-08-25
status: implemented
see-also:
  - "https://github.com/kubernetes/kubernetes/issues/30656"
  - "https://github.com/kubernetes/kubernetes/pull/81883"
replaces:
  - "n/a"
superseded-by:
  - "n/a"
---

# Exec as specified user

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
- [Design Details](#design-details)
  - [API Specification](#api-specification)
  - [Library Specification](#library-specification)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

As a Kubernetes User, we should be able to specify username for the containers when doing exec, similar to how docker allows that using docker exec options `-u`, 
```
-u, --user="" Username or UID (format: <name|uid>[:<group|gid>]) format
```

## Motivation

The root problem is lacking of standard description for the `exec` operation.

OCI didn't introduce `exec` as a regular operation for container runtime[2],

so when Kubernetes announce [Container Runtime Interface(CRI)](https://kubernetes.io/blog/2016/12/container-runtime-interface-cri-in-kubernetes/) in release 1.5, it became the de-facto standard of `exec`, without a `user` option.


>  In the Kubernetes 1.5 release, we are proud to introduce the Container Runtime Interface (CRI) – a plugin interface which enables kubelet to use a wide variety of container runtimes, without the need to recompile. CRI consists of a protocol buffers and gRPC API, and libraries, with additional specifications and tools under active development. CRI is being released as Alpha in Kubernetes 1.5.

But There are many container runtime implementation, e.g Docker, Kata, gVisior, most of which provide a way to enter the container as a special `user`.

It is because the `exec` is basically to start a process inside the container runtime, and the [`process`](https://github.com/opencontainers/runtime-spec/blob/52e2591aa9f7211d64c49c4fed8691a183189284/specs-go/config.go#L39) has a user definition, which is required.

> User specifies specific user (and group) information for the container process

It brings convenience to debugging, and users are used to it. Adding a `user` option with `exec`  in Kubernetes seems reasonable.

### Goals

1. Provide the ability to specify the username for a exec operation inside a container

## Proposal

### User Stories [optional]

As a Kubernetes User, I should be able to control how people can enter the container, by default kubernetes use the default user in docker image.

#### Story 1
By providing the `user` option for `exec`, users can use their own ID to enter the container and exercise their respective rights.

```
-u, --user="" Username or UID (format: <string>]) format
```

Since the container does not have a uniform `exec` format, the format of `user` should be just a string.

#### Story 2
By providing the `user` option for `exec`, user can do the auit inside the container.

## Design Details

According to [CRI Streaming Requests (exec/attach/port-forward)](https://docs.google.com/document/d/1OE_QoInPlVCK9rMAx9aybRmgFiVjHpJCHI9LrfdNM_s/), add a string field `user` to the `ExecRequest` in gRPC api.

### API Specification

Implemented in [kubernetes/#81883](https://github.com/kubernetes/kubernetes/pull/81883/files#diff-f7cf2ebf4fbb4f02ecc92a8fa1a2f7fcR974)

```
message ExecRequest {
    //Other fields not shown for brevity
    ..... 

    // execute command as a specific user
    string user = 7;
}
```

### Library Specification

Implemented in [kubernetes/#81883](https://github.com/kubernetes/kubernetes/pull/81883/files#diff-399a1489053b508436d8c0845b122e4eR64)

```
// Runtime is the interface to execute the commands and provide the streams.
type Runtime interface {
    //Other fields not shown for brevity
    ..... 

	Exec(containerID string, cmd []string, user string, in io.Reader, out, err io.WriteCloser, tty bool, resize <-chan remotecommand.TerminalSize) error

    .....
```

## Implementation History

- https://github.com/kubernetes/kubernetes/pull/81883/
