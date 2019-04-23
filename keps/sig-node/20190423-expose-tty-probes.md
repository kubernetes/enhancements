---
title: KEP Template
authors:
  - "@umohnani8"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-architecture
reviewers:
  - TBD
  - "@yujuhong"
approvers:
  - "@Random-Liu"
  - "@yujuhong"
editor: TBD
creation-date: 2019-04-15
last-updated: 2019-04-15
status: implemented
see-also:
replaces:
superseded-by:
---

# Expose tty option for probes

## Table of Contents

A table of contents is helpful for quickly jumping to sections of a KEP and for highlighting any additional information provided beyond the standard KEP template.
[Tools for generating][] a table of contents from markdown are available.

- [Title](#title)
  - [Table of Contents](#table-of-contents)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Example of Failure](#example-of-failure)
  - [Motivation](#motivation)
    - [Goals](#goals)
  - [Proposal](#proposal)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [Test Plan](#test-plan)
    - [Graduation Criteria](#graduation-criteria)
  - [Implementation History](#implementation-history)
  - [Alternatives [optional]](#alternatives-optional)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Release Signoff Checklist
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

## Summary

Some readiness and liveness probes have exec commands that use `/bin/sh -i -c`, especially when deploying postgres and mysql. However, these probes fail with an `ioctl error` that happens because tty is set to false by default. Setting `-i` in thses exec command helps to pick the default binary paths and libraries from the start up scripts without having to specify the full paths in the command. Adding an optional boolean tty field to the probes gives the users the flexibility of turning n tty whenever using `/bin/sh -i -c` in their exec commands.

## Example of Failure

The readiness probe fails in a Mysql application pod when provisioning mysql apb with prod plan.
Excerpt of the pod template with the probes:
```
Containers:
  mysql:
    Container ID:   cri-o://a7d484b683c96701a120a65de500bc9cd95211c65f6185077aee48d53d3f8abe
    Image:          registry.io/rhscl/mysql-57-rhel7
    Image ID:       registry.io/rhscl/mysql-57-rhel7@sha256:5d8f08df7620fcd599d1940baba150d26b5ce027b204f9850417be5ca41ae481
    Port:           3306/TCP
    Host Port:      0/TCP
    State:          Running
      Started:      Mon, 27 Aug 2018 05:31:50 -0400
    Ready:          True
    Restart Count:  0
    Liveness:       exec [/bin/sh -i -c mysqladmin -u$MYSQL_USER -p$MYSQL_PASSWORD ping] delay=120s timeout=5s period=10s #success=1 #failure=3
    Readiness:      exec [/bin/sh -i -c MYSQL_PWD="$MYSQL_PASSWORD" mysql -h 127.0.0.1 -u $MYSQL_USER -D $MYSQL_DATABASE -e 'SELECT 1'] delay=5s timeout=1s period=10s #success=1 #failure=3
...
```

We end up with the pod failing with the following error:
```
Warning  Unhealthy               1m (x11 over 3m)  kubelet, qe-chezhang-0827node-1  Readiness probe failed: sh: cannot set terminal process group (-1): Inappropriate ioctl for device
sh: no job control in this shell
ERROR 2003 (HY000): Can't connect to MySQL server on '127.0.0.1' (111)
```

The same issue has been seen for other deployments like mariadb and postgressql.

## Motivation

Make it easier for users that use liveness and readiness probes. This change will let the users set `/bin/sh -i -c` in their probes' exec commands without requiring them to specify the full path names of the commands they are running. They will no longer run into an `ioctl error` as the `tty` option will be exposed in the api.
Users run into this problem especially when trying to deploy containers with databases such as postgres and mysql.

### Goals

Provide users with the ability to easily set an optional `tty` value when running liveness and readiness probes.

## Proposal

We propose adding a new boolean option called `Tty` in the CRI API for probes. This way users can easily configure it in their kube yaml files.

```
type Probe struct {
	...
	// Set tty for certain exec commands used in probes, especially when doing "/bin/sh -i -c".
	// +optional
	Tty bool
}
```

The probes call an exec sync command when execing a command. So we will add `tty` as an option to the exec sync function that the probe calls.

```
ExecSync(containerID string, cmd []string, timeout time.Duration, tty bool) ([]byte, []byte, error)
```

The default value of the `tty` option will be false.

Will also add tests for this to ensure it works and does not break any existng features.

### Risks and Mitigations

There are no serious risks with making `Tty` an optional arg for probes. Only users that run into the `ioctl error` will be using it.

### Test Plan

Will add an e2e test to the existing container probe tests.

### Graduation Criteria

stable:
- No complaints about having the optional `Tty` arg for probes

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

## Alternatives [optional]

- Setting tty to true by default in the exec sync function - this was rejected. Discussion is at https://github.com/kubernetes/kubernetes/pull/66084
