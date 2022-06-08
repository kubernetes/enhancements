# Username option in kubectl exec and CRI update


## Table of Contents

<!-- toc -->
- [Summary](#summary)
  - [Glossary](#glossary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha -&gt; Beta](#alpha---beta)
  - [Beta -&gt; GA](#beta---ga)
- [Test Plan](#test-plan)
  - [Prerequisite testing updates](#prerequisite-testing-updates)
  - [Unit tests](#unit-tests)
  - [Integration/e2e tests](#integratione2e-tests)
  - [Per-driver migration testing](#per-driver-migration-testing)
  - [Upgrade/Downgrade/Skew Testing](#upgradedowngradeskew-testing)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

As a Kubernetes User, we should be able to specify username for the containers when doing exec, similar to how docker allows that using docker exec options `-u`, 
```
-u, --user="" Username or UID (format: <name|uid>[:<group|gid>]) format
```

### Glossary


## Motivation

Docker allows execution of commands as particular user with docker exec -u <username>, when USER <username> is used in Dockerfile.
This same functionality doesn't exist in Kubernetes.

https://github.com/kubernetes/kubernetes/issues/30656

https://github.com/containerd/containerd/issues/6662


### Goals

Provide the ability to specify the username for a exec operation inside a container

### Non-Goals

## Proposal

### User Stories [optional]

As a Kubernetes User, I should be able to control how people can enter the container, by default kubernetes use the default user in docker image.

By providing the `user` option for `exec`, users can use their own ID to enter the container and exercise their respective rights.

```
-u, --user="" Username or UID (format: <string>]) format
```

Since the container does not have a uniform `exec` format, the format of `user` should be just a string.


### Implementation Details/Notes/Constraints

### API Specification

```
message ExecRequest {
    //Other fields not shown for brevity
    ..... 
    // execute command as a specific user
    string user = 7;
}
```

### Risks and Mitigations

## Graduation Criteria

### Alpha -> Beta

### Beta -> GA


## Test Plan


### Prerequisite testing updates


### Unit tests


### Integration/e2e tests


### Per-driver migration testing


### Upgrade/Downgrade/Skew Testing


## Production Readiness Review Questionnaire


### Feature Enablement and Rollback


### Rollout, Upgrade and Rollback Planning


### Monitoring Requirements


### Dependencies


### Scalability


### Troubleshooting


## Implementation History

