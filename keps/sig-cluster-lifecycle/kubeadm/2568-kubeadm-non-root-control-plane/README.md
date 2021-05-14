<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [X] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [X] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [X] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [X] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [X] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [X] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-2568: Run control-plane as non-root in kubeadm.

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Assigning UID and GID](#assigning-uid-and-gid)
  - [Updating the component manifests](#updating-the-component-manifests)
  - [Host Volume Permissions](#host-volume-permissions)
  - [Shared files](#shared-files)
  - [Reusing users and groups](#reusing-users-and-groups)
  - [Cleaning up users and groups](#cleaning-up-users-and-groups)
  - [Multi OS support](#multi-os-support)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
  - The PRR was N/A as there are no in-tree changes proposed in this KEP. Pleases see these slack discussion threads. [Thread 1](https://kubernetes.slack.com/archives/CPNHUMN74/p1618272532012700) [Thread 2](https://kubernetes.slack.com/archives/CPNHUMN74/p1619205764018600)
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->
This KEP proposes that the control-plane in `kubeadm` be run as non-root. If 
containers are running as root an escape from a container may result in the 
escalation to root in host. [CVE-2019-5736](https://kubernetes.io/blog/2019/02/11/runc-and-cve-2019-5736/)
is an example of a container escape vulnerability that can be mitigated by 
running containers/pods as non-root. 

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->
[CVE-2019-5736](https://kubernetes.io/blog/2019/02/11/runc-and-cve-2019-5736/)
is an example of a container escape vulnerability that can be mitigated by 
running containers/pods as non-root.

Running containers as non-root has been a long recommended best-practice in 
kubernetes and we have published [blogs](https://kubernetes.io/blog/2018/07/18/11-ways-not-to-get-hacked/#8-run-containers-as-a-non-root-user) recommending this best practice. `kubeadm`
which is a tool built to provide a "best-practice" path to creating clusters is
an ideal candidate to apply this to.


### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
- Run control-plane components as non-root in `kubeadm`. More specifically :-
  - Run `kube-apiserver` as non-root in `kubeadm`.
  - Run `kube-controller-manager` as non-root in `kubeadm`.
  - Run `kube-scheduler` as non-root in `kubeadm`.
  - Run `etcd` as non-root in `kubeadm`.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
- Run node components as non-root in `kubeadm`. More specifically :-
  - Run `kube-proxy` as non-root, since it is not a control-plane component.
  - Run `kubelet` as non-root.
  - Compatibility with user namespace enabled environments is not in scope.
  - Setting defaults for SELinux and AppArmor are not in scope. (We may reconsider this in beta.)

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->
Here is what we propose to do at a high level:

1. Run the control-plane components as non-root, by assigning a unique uid/gid to the containers in the Pods using the `runAsUser` and `runAsGroup` fields in the [securityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#securitycontext-v1-core) on the Pods.

2. Drop all capabilities from `kube-controller-manager`, `kube-scheduler`, `etcd` Pods. Drop all but cap_net_bind_service from `kube-apiserver` Pod, this will allow us to run as non-root while still being able to bind to ports < 1024. This can be achieved by using the `capabilities` field in the [securityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#securitycontext-v1-core) of the containers in the Pods.

3. Set the seccomp profile to `runtime/default`, this can be done by setting the `seccompProfile` field in the `securityContext`.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a security conscious user I would like to run the kubernetes control-plane as non-root to reduce the risk associated with container escape vulnerabilities in the control-plane.

#### Story 2

As a `kubeadm` user, I'd expect the bootstrapper to follow the best security practices when generating static or non-static pod manifests to run control plane components.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->
There are some caveats with running the components as non-root around how to assign `UID`/`GID`s to them. This is covered in the [Assigning UID and GID](#assigning-uid-and-gid) section below. 

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->
If we hard coded the `UID` and `GID`, we could end up in a scenario where those are in use by another process on the machine, which would expose some of the credentials accessible to the `UID` and `GID`s to that process. So we plan to use adduser --system or using the appropriate ranges from /etc/login.defs instead of hard coding the `UID` and `GID`.


## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

In `kubeadm` the control-plane components are run as [static-pods](https://kubernetes.io/docs/concepts/workloads/pods/#static-pods), i.e. pods directly managed by kubelet. We can use the `runAsUser` and `runAsGroup` fields in [SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#securitycontext-v1-core) to run the containers in these Pods with a unique `UID` and `GID` and the `capabilities` field to drop all capabilities other than the ones required.

### Assigning UID and GID
There are 3 options for setting the `UID`/`GID` of the control-plane components:-

1. **Update the kubeadm API to make the uid/gid configurable by the user:**  This can be implemented in 2 ways:-
    1. This can be done by adding fields `UID` and `GID` of type `int64` to the `ControlPlaneComponent` `struct` in https://github.com/kubernetes/kubernetes/blob/854c2cc79f11cfb46499454e7717a86d3214e6b0/cmd/kubeadm/app/apis/kubeadm/types.go#L132 as demonstrated in [this](https://github.com/kubernetes/kubernetes/pull/99753) PR.
    2. User provides `kubeadm` with a range of uid/gids that are safe to use i.e. are not being used by another user.

2. **Use constant values for uid/gid:** The `UID` and `GID`for each of the control-plane components would be set to some predetermined value in https://github.com/kubernetes/kubernetes/blob/master/cmd/kubeadm/app/constants/constants.go.

3. **Create system users and let users override the defaults:** Use `adduser --system` or equivalent to create `UID`s in the SYS_UID_MIN - SYS_UID_MAX range and `groupadd --system` or equivalent to create `GID`s in the SYS_GID_MIN - SYS_GID_MAX range. Additionally if users want to specify their own `UID`s or `GID`s we will support that through `kubeadm` patching.

The author(s) believes that starting out with a safe default of option 3. and allowing the user to set the `UID` and `GID` through the `kubeadm` patching mechanism is more user-friendly for users who just wan't to quickly bootstrap and also users who care about which `UID`s and `GID`s that they want to run the control-plane as and also users . Further this feature will be opt-in and will be hidden behind a feature-gate, until it graduates to GA.

Choosing the `UID` between SYS_UID_MIN and SYS_UID_MAX and `GID` between SYS_GID_MIN and SYS_GID_MAX is in adherence with distro standards.
* For Debian : https://www.debian.org/doc/debian-policy/ch-opersys.html#uid-and-gid-classes
* For Fedora : https://docs.fedoraproject.org/en-US/packaging-guidelines/UsersAndGroups/

### Updating the component manifests

An example of how the `kube-scheduler` manifest would like is below:

```yaml
apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  labels:
    component: kube-scheduler
    tier: control-plane
  name: kube-scheduler
  namespace: kube-system
spec:
  containers:
  - command:
    - kube-scheduler
    - --authentication-kubeconfig=/etc/kubernetes/scheduler.conf
    - --authorization-kubeconfig=/etc/kubernetes/scheduler.conf
    - --bind-address=127.0.0.1
    - --kubeconfig=/etc/kubernetes/scheduler.conf
    - --leader-elect=true
    - --port=0
    image: k8s.gcr.io/kube-scheduler:v1.21.0-beta.0.368_9850bf06b571d5-dirty
    imagePullPolicy: IfNotPresent
    livenessProbe:
      failureThreshold: 8
      httpGet:
        host: 127.0.0.1
        path: /healthz
        port: 10259
        scheme: HTTPS
      initialDelaySeconds: 10
      periodSeconds: 10
      timeoutSeconds: 15
    name: kube-scheduler
    resources:
      requests:
        cpu: 100m
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
      runAsGroup: 2000 # this value is only an example and is not the id we plan to use.
      runAsUser: 2000  # this value is only an example and is not the id we plan to use.
    startupProbe:
      failureThreshold: 24
      httpGet:
        host: 127.0.0.1
        path: /healthz
        port: 10259
        scheme: HTTPS
      initialDelaySeconds: 10
      periodSeconds: 10
      timeoutSeconds: 15
    volumeMounts:
    - mountPath: /etc/kubernetes/scheduler.conf
      name: kubeconfig
      readOnly: true
  hostNetwork: true
  priorityClassName: system-node-critical
  volumes:
  - hostPath:
      path: /etc/kubernetes/scheduler.conf
      type: FileOrCreate
    name: kubeconfig
status: {}
```

### Host Volume Permissions

In the manifest above the reader can notice that the `kube-scheduler` container mounts the `kubeconfig` file from the host. So we must
grant read permissions to that file to the user that the container runs as, which in this case is `2000`. `kubeadm` is responsible for bootstrap and creates the manifests of the control-plane components, which means that it knows all the volume mounts for the container.

There are 2 ways we can set the permissions:-
1. **Use [initContainers](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/):** We could run an `initContainer` that runs as root and mounts all the hostVolumes that the control-plane component container mounts. It then calls chown uid:gid on each of the files/directories that are mounted. See the example for `kube-shceduler` below:-

```yaml

apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  labels:
    component: kube-scheduler
    tier: control-plane
  name: kube-scheduler
  namespace: kube-system
spec:
  initContainers:
  - name: kube-scheduler-init
    volumeMounts:
    - mountPath: /etc/kubernetes/scheduler.conf
      name: kubeconfig
      readOnly: true
    image: k8s.gcr.io/debian-base:buster-v1.4.0
    command:
      - /bin/sh
      - -c
      - chown 2000:2000 /etc/kubernetes/scheduler.conf
  containers:
  - command:
    - kube-scheduler
    - --authentication-kubeconfig=/etc/kubernetes/scheduler.conf
    - --authorization-kubeconfig=/etc/kubernetes/scheduler.conf
    - --bind-address=127.0.0.1
    - --kubeconfig=/etc/kubernetes/scheduler.conf
    - --leader-elect=true
    - --port=0
    image: k8s.gcr.io/kube-scheduler:v1.21.0-beta.0.368_9850bf06b571d5-dirty
    imagePullPolicy: IfNotPresent
    livenessProbe:
      failureThreshold: 8
      httpGet:
        host: 127.0.0.1
        path: /healthz
        port: 10259
        scheme: HTTPS
      initialDelaySeconds: 10
      periodSeconds: 10
      timeoutSeconds: 15
    name: kube-scheduler
    resources:
      requests:
        cpu: 100m
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
      runAsGroup: 2000  # this value is only an example and is not the id we plan to use.
      runAsUser: 2000  # this value is only an example and is not the id we plan to use.
    startupProbe:
      failureThreshold: 24
      httpGet:
        host: 127.0.0.1
        path: /healthz
        port: 10259
        scheme: HTTPS
      initialDelaySeconds: 10
      periodSeconds: 10
      timeoutSeconds: 15
    volumeMounts:
    - mountPath: /etc/kubernetes/scheduler.conf
      name: kubeconfig
      readOnly: true
  hostNetwork: true
  priorityClassName: system-node-critical
  volumes:
  - hostPath:
      path: /etc/kubernetes/scheduler.conf
      type: FileOrCreate
    name: kubeconfig
status: {}
```
The initContainer `kube-scheduler-init` will run before the `kube-scheduler` container and will setup the permissions of the files that the `kube-scheduler` container needs. Since initContainers are shotlived and exit once they are done the risk from running them as root on the host is low.

2. **kubeadm sets the permissions:** In this approach `kubeadm` would be responsible for setting the file permissions when it creates the files. It will call [os.Chown](https://golang.org/pkg/os/#Chown) to set the owner of the files. This approach is demonstrated in PR https://github.com/kubernetes/kubernetes/pull/99753.

The author(s) believe that it is better for `kubeadm` to set the permission because adding an initContainer would require pulling of debian-base image (or similar) to run the commands to change file ownership, which is something that can be easily done in go. Also it leads to the question of which initContainer would be responsible for files shared between `kube-controller-manager` and `kube-apiserver`? Since `kubeadm` creates these files its best for it to apply the file permissions.

### Shared files

Certain hostVolume mounts are shared between `kube-apiserver` and `kube-controller-manager`, some examples of these are:-
- /etc/kubernetes/pki/ca.crt
- /etc/kubernetes/pki/sa.key

We propose that files shared by `kube-controller-manager` and `kube-apiserver` be readable by a particular `GID` and we set the `supplementalGroups` in the [PodSecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#podsecuritycontext-v1beta1-policy) of the Pod to this GID for `kube-apiserver` and `kube-controller-manager`.

For instance lets consider that `kubeadm` grant group 2100 read permissions to /etc/kubernetes/pki/ca.crt

Then we update the `kube-apiserver` manifest as follows:-

```yaml

apiVersion: v1
kind: Pod
metadata:
  annotations:
    kubeadm.kubernetes.io/kube-apiserver.advertise-address.endpoint: 172.17.0.2:6443
  creationTimestamp: null
  labels:
    component: kube-apiserver
    tier: control-plane
  name: kube-apiserver
  namespace: kube-system
spec:
  securityContext:
    supplementalGroups:
    - 2100  # this value is only for an example and is not the id we plan to use.
  containers:
  - name: kube-apiserver
    command:
    - kube-apiserver
    - --advertise-address=172.17.0.2
    - --client-ca-file=/etc/kubernetes/pki/ca.crt
    ... # omitted to save space 
    securityContext:
      runAsUser: 2000  # this value is only an example and is not the id we plan to use.
      runAsGroup: 2000  # this value is only an example and is not the id we plan to use.
      allowPrivilegeEscalation: false
      seccompProfile:
        type: runtime/default
      capabilities:
        drop:
        - ALL
    image: k8s.gcr.io/kube-apiserver:v1.21.0-beta.0.368_9850bf06b571d5-dirty
    ... # omitted to save space
    volumeMounts:
    ... # omitted to save space
    - mountPath: /etc/kubernetes/pki
      name: k8s-certs
      readOnly: true
    ... # omitted to save space
```

Similarly `kube-controller-manager`'s manifest would like:-

```yaml
apiVersion: v1
kind: Pod
metadata:
  creationTimestamp: null
  labels:
    component: kube-controller-manager
    tier: control-plane
  name: kube-controller-manager
  namespace: kube-system
spec:
  securityContext:
    supplementalGroups:
    - 2100
  containers:
  - name: kube-controller-manager
    command:
    - kube-controller-manager
    - --cluster-signing-cert-file=/etc/kubernetes/pki/ca.crt
    ...
    # omitted to save space.
    securityContext:
      runAsUser: 2001  # this value is only an example and is not the id we plan to use.
      runAsGroup: 2001 # this value is only an example and is not the id we plan to use.
      allowPrivilegeEscalation: false
      seccompProfile:
        type: runtime/default
      capabilities:
        drop:
        - ALL
    image: k8s.gcr.io/kube-controller-manager:v1.21.0-beta.0.368_9850bf06b571d5-dirty
    ... # omitted to save space.
    volumeMounts:
    - mountPath: /etc/kubernetes/pki
      name: k8s-certs
      readOnly: true
    ... # omitted to save space. 
```

Each of the components will run with a unique `UID` and `GID`. For each of the components we will create a unique user. For the shared files/resources we will create groups. The naming convention of these groups is tabulated below. It should be noted that `kubeadm` will take exclusive ownership of these users/groups and will throw erros if users/groups with these names exist and are not in the expected ID range of `SYS_UID_MIN`-`SYS_UID_MAX` for users and `SYS_GID_MIN`-`SYS_GID_MAX` for groups.

Many of the components need shared access to certificate files, these are not protected by creating a group with read permissions because certificates are not secrets, protecting them and creating groups for them does not improve our security posture in anyway and only makes the change more complicated because we are adding unnecessary groups. Hence we only propose that we create a group with read access for the `/etc/kubernetes/pki/sa.key` file, which is the only secret that is shared between `kube-apiserver` and `kube-controller-manager`. `kubeadm` creates all certificate files with `0644` so we do not need to modify their owners as they are already world readable.

| User/Group name | Explanation |
|--------------|-------------|
| kubeadm-etcd | The UID/GID that we will assign to `etcd` |
| kubeadm-kas  | The UID/GID that we will assign to `kube-apiserver` |
| kubeadm-kcm | The UID/GID that we will assign to `kube-controller-manager` |
| kubeadm-ks | The UID/GID that we will assign to `kube-scheduler` |
| kubeadm-sa-key-readers | The GID we will assign to a group that allows you to read /etc/kubernetes/pki/sa.key |

Here is a table of all the things that `kube-apiserver`, `kube-controller-manager`, `kube-scheduler` and `etcd` mount and the permissions that we will set for them.

**Files that we care about for this kep:-**
| file/directory                                   | Component(s) | File permission |
| -------------------------------------------------|------------|-----------------|
| /etc/kubernetes/pki/etcd/server.crt              | etcd       | 644 kubeadm-etcd kubeadm-etcd   |
| /etc/kubernetes/pki/etcd/server.key              | etcd       | 600 kubeadm-etcd kubeadm-etcd   |
| /etc/kubernetes/pki/etcd/peer.crt                | etcd       | 644 kubeadm-etcd kubeadm-etcd   |
| /etc/kubernetes/pki/etcd/peer.key                | etcd       | 600 kubeadm-etcd kubeadm-etcd   |
| /etc/kubernetes/pki/etcd/ca.crt                  | etcd, kas  | 644 root root |
| /var/lib/etcd/                                   | etcd       | 600 kubeadm-etcd kubeadm-etcd   |
| /etc/kubernetes/pki/ca.crt                       | kas, kcm   | 644 root root |
| /etc/kubernetes/pki/apiserver-etcd-client.crt    | kas        | 644 root root     |
| /etc/kubernetes/pki/apiserver-etcd-client.key    | kas        | 600 kakubeadm-kas kubeadm-kas     |
| /etc/kubernetes/pki/apiserver-kubelet-client.crt | kas        | 644 root root     |
| /etc/kubernetes/pki/apiserver-kubelet-client.key | kas        | 600 kubeadm-kas kubeadm-kas     |
| /etc/kubernetes/pki/front-proxy-client.crt       | kas        | 644 root root     |
| /etc/kubernetes/pki/front-proxy-client.key       | No-one        | 600 root root     |
| /etc/kubernetes/pki/front-proxy-ca.crt           | kas, kcm   | 644 root root |
| /etc/kubernetes/pki/sa.pub                       | kas        | 600 kkubeadm-kass kubeadm-kas     |
| /etc/kubernetes/pki/sa.key                       | kas, kcm   | 640 kubeadm-sa-key-readers |
| /etc/kubernetes/pki/apiserver.crt                | kas        | 644 root root     |
| /etc/kubernetes/pki/apiserver.key                | kas        | 600 kubeadm-kas kubeadm-kas     |
| /etc/kubernetes/pki/ca.key                       | kcm        | 600 kubeadm-kcm kubeadm-kcm     |
| /etc/kubernetes/controller-manager.conf          | kcm        | 600 kubeadm-kcm kubeadm-kcm     |
| /etc/kubernetes/scheduler.conf                   | ks         | 600 kubeadm-ks kubeadm-ks       |

In addition to the file/directories in that table above the control-plane components also mount the directories below, these we don't have to worry about as these are world readable.

**World readable stuff:**
- /usr/local/share/ca-certificates
- /usr/share/ca-certificates
- /etc/ssl/certs
- /etc/ca-certificates

### Reusing users and groups

If any of the users/groups defined above exist already and are in the expected ID range of `SYS_UID_MIN`-`SYS_UID_MAX` for users and `SYS_GID_MIN`-`SYS_GID_MAX` for groups, then `kubeadm` will reuse these instead of creating new ones. More specifically `kubeadm` will reuse the ones that exist and meet the criteria and will create the ones that it needs.

### Cleaning up users and groups

`kubeadm reset` tries to remove everything created by `kubeadm` on the host and it should do this for the users and groups that it creates as part of cluster bootstrap.


### Multi OS support

A Windows control plane is out of scope for this proposal for the time being. OS specific implementations for Linux, would be carefully abstracted behind helper utilities in kubeadm to not block the support for a Windows control plane in the future.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

Following functionality needs to be tested:
1. With feature-gate=True create a cluster
2. With feature-gate=True upgrade a cluster

These tests will be added using the [kinder](https://github.com/kubernetes/kubeadm/tree/master/kinder) tooling during the Alpha stage.

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include 
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

#### Alpha -> Beta Graduation

- All control plane components are running as non-root.
- All control-plane components have `runtime/default` seccomp profile.
- All control-plane components drop all unnecessary capabilities.
- The feature is tested by the community and feedback is adapted with changes.
- e2e tests are running periodically and are green with respect to testing this functionality.
- `kubeadm` documentation is updated.

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

The flow below is assuming that the feature-flag  to run control-plane as non-root is enabled.

`kubeadm` checks the cluster-config to see if the control-plane is already running as non-root. If so it re-writes the contents of the files/credentials and makes sure that the `UID`s and `GID`s previously assigned have permissions to read/write appropriately. The control-plane static-pod manifests don't explicitly need to be updated for running them as non-root in this case.

If the control-plane was not running as non-root before then `kubeadm` creates new `UID`s and `GID`s based on the approach mentioned in the [Assigning UID and GID](#assigning-uid-and-gid) section and updates the cluster-config. When files/credentials are re-written the owner of these files are set appropriately. The control-plane static-pod manifests explicitly need to be updated to run as non-root in this case.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

`kubeadm` version X supports deploying kubernetes control-plane version X and X-1. Once the feature gate is enabled in kubeadm to run the control-plane as non root we will run both X and X-1 versions of control-plane as non-root. Nothing in the design of this feature is tied to the version of the control-plane.

## Production Readiness Review Questionnaire

> :warning:  **The PRR was N/A as there are no in-tree changes proposed in this KEP.** Pleases see these slack discussion threads. [Thread 1](https://kubernetes.slack.com/archives/CPNHUMN74/p1618272532012700) [Thread 2](https://kubernetes.slack.com/archives/CPNHUMN74/p1619205764018600)

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->
Note: the feature gate here is for `kubeadm` and not the control-plane components.

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: kubeadmRootlessControlPlane
  - Components depending on the feature gate: `kube-apiserver`, `kube-controller-manager`, `kube-scheduler` and `etcd` in `kubeadm` control-plane
  - Describe the mechanism: 
  - Will enabling / disabling the feature require downtime of the control
    plane?
    No, since it will only take effect when the control-plane is upgraded or created.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
    No, this only affects the control-plane, no change is required on the node(s).

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Yes it will change the default behavior of kubeadm from running the control-plane components as root to non-root.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
Yes disabling the feature-gate in kubeadm and the upgrading the control-plane to the current version should run the components as root again.

###### What happens if we reenable the feature if it was previously rolled back?
Nothing unless the user upgrades or creates a new cluster, if they do so, then the control-plane components on the upgraded/created cluster will run as non-root.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

Yes we plan to add e2e tests to test the kubeadm behavior with feature gate enabled using kinder.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

<!--
At a high level, this usually will be in the form of "high percentile of SLI
per day <= X". It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code
-->

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->
None
### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->
No
###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No
###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No
###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
No
###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
Yes, in kubeadm control-plane bootstrap process we will create users/groups for the various control-plane components. This operation will add a minute delay to bootstrap. Also failing to do so would cause the bootstrap to fail.

When we create files and directories we would have to change the permissions and the owners of these files. So there will be a minute increase in bootstrap time for control-plane.
###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
No
### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->
Major milestones:
- Initial draft of KEP created - 2021-03-13
- Production readiness review - 2021-04-12
- Production readiness review approved - 2021-04-29
- KEP marked implementable - 2021-04-28

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->
None

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
None

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
None
