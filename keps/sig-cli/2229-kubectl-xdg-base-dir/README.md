# KEP-2229: kubectl xdg base dir

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Option 1: Replace .kubeconfig in loader.go](#option-1-replace-kubeconfig-in-loadergo)
  - [Option 2: Set new RecommendedHomeFile to use XDG Base Directory Specification](#option-2-set-new-recommendedhomefile-to-use-xdg-base-directory-specification)
  - [Option 3: Recommend users to use KUBECONFIG](#option-3-recommend-users-to-use-kubeconfig)
  - [Additional info](#additional-info)
  - [Test Plan](#test-plan)
    - [Alpha milestones](#alpha-milestones)
    - [Beta milestones](#beta-milestones)
    - [GA milestones](#ga-milestones)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

This enhancement is focused in provide all requirements for kubectl
use the [XDG Base Directory Specification (XDG Spec)](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html).
The XDG Spec have been used for a long time as the standard location to tools
and programs store configuration files.

## Motivation

Today the kubectl store the configuration file via ~/.kube/ dir.
However, there are multiple [community requests like #56402](https://github.com/kubernetes/kubernetes/issues/56402),
from real users that have been looking for: 
  - A single place for managing the configuration files
  - Automation/Backup
  - Use the same environment [variables as others projects](https://specifications.freedesktop.org/basedir-spec/latest/ar01s03.html)

### Goals

The goal is make kubectl follow the XDG Spec and automatic migrate
the configuration from $HOME/.kube to $HOME/.config/kube wihout stopping
kubectl to work.

### Non-Goals

Deprecate any file under $HOME/.kube/

## Proposal

- kubectl should follow the XDG Base Directory Specification
- Use $HOME/.config/kube as default dir for configurations
- Be compatible with $HOME/.kube until config migrated to $HOME/.config/kube
- Smoothly migrate the config from $HOME/.kube to the new location
- Update documentation related to ~/.kube
- Write a blog post to explain and communicate such change

### Risks and Mitigations

Risks are limited, as none of the cluster components, will be affected
directly. Additionally, the original ~/.kube dir will be compatible
until a migration fully happen or deprecation. In case there is
a bug in the new logic users will still be able to KUBECONFIG env var
or use the cluster via API calls or tools like curl.

## Design Details

There are few possibilities to address this request. 

### Option 1: Replace .kubeconfig in loader.go

$HOME/.kube/.kubeconfig seems to be deprecated place for the configuration and
has been a long time since it's migrated to $HOME/.kube/config. A good reference from 2015:
https://github.com/kubernetes/kubernetes/issues/4615

It's possible to set the oldRecommendedHomeFileName to $HOME/.kube/config but we lost
the support of $HOME/.kube/.kubeconfig

Reference:
https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/client-go/tools/clientcmd/loader.go#L62

The drawback:
- Users using $HOME/.kube/.kubeconfig will not migrate to the new place as we are going to replace
  the static reference for oldRecommendedHomeFileName.
- Tools that use static reference to $HOME/.kube/config will require to be updated
- It's required to spread the world about this change as the config will move location ($HOME/.config/kube/config)

### Option 2: Set new RecommendedHomeFile to use XDG Base Directory Specification

This option will use the current logic of Migration already available in the code and seems modular
enough as expected by the original author.

This implementation will be split in different patches (for easy review):

1. Update the logic from loader.go to migrate the config to the new recommended
   location in a transparent way to users.
   See: https://github.com/kubernetes/kubernetes/pull/97885

2. If 1 is approved and merged, update kubectl related code to the new location.

3. If 2 is approved and merged, update the rest of the code in the kubernetes tree pointing statically to $HOME/.kube

The drawback:
- Tools that use static reference to $HOME/.kube/config will require to be updated
- It's required to spread the world about this change as the config will move location ($HOME/.config/kube/config)

### Option 3: Recommend users to use KUBECONFIG

No changes in the code. Just close all requests from users suggesting to use KUBECONFIG env var.

```
 $ mkdir -p $HOME/.config/kube
 $ mv $HOME/.kube/config $HOME/.config/kube
 $ export KUBECONFIG=$HOME/.config/kube/config
```

### Additional info

The following source codes are ref. to $HOME/.kube/config

| Status         | Code                                                                                    | Reason                                  |
|----------------|-----------------------------------------------------------------------------------------|-----------------------------------------|
| Need to update | cmd/kubeadm/app/cmd/init.go                                                             | contains reference to $HOME/.kube       |
| Need to update | cluster/common.sh                                                                       | contains reference to $HOME/.kube       |
| Need to update | cluster/gce/windows/README-GCE-Windows-kube-up.md                                       | contains reference to $HOME/.kube       |
| Need to update | cmd/kubeadm/app/cmd/join.go                                                             | contains reference to $HOME/.kube       |
| Need to update | cmd/kubeadm/app/cmd/reset.go                                                            | contains reference to $HOME/.kube       |
| Need to update | cmd/kubeadm/app/cmd/completion.go                                                       | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/cli-runtime/pkg/genericclioptions/config_flags.go                    | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/cli-runtime/pkg/genericclioptions/client_config.go                   | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/sample-controller/README.md                                          | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/kubectl/pkg/cmd/config/create_cluster.go                             | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/kubectl/pkg/cmd/config/create_authinfo.go                            | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/kubectl/pkg/cmd/testing/fake.go                                      | contains reference to $HOME/.kube/cache |
| Need to update | staging/src/k8s.io/kubectl/pkg/cmd/completion/completion.go                             | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/sample-apiserver/README.md                                           | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/client-go/util/homedir/homedir.go                                    | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/client-go/discovery/cached/disk/cached_discovery.go                  | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/client-go/examples/dynamic-create-update-delete-deployment/main.go   | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/client-go/examples/dynamic-create-update-delete-deployment/README.md | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/client-go/examples/create-update-delete-deployment/main.go           | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/client-go/examples/create-update-delete-deployment/README.md         | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/client-go/examples/out-of-cluster-client-configuration/main.go       | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/client-go/tools/clientcmd/loader.go                                  | contains reference to $HOME/.kube       |
| Need to update | staging/src/k8s.io/apiserver/pkg/admission/config_test.go                               | contains reference to $HOME/.kube       |
| Need to update | test/cmd/legacy-script.sh                                                               | contains reference to $HOME/.kube       |
| Need to update | test/e2e/kubectl/kubectl.go                                                             | contains reference to $HOME/.kube       |
| Need to update | test/soak/serve_hostnames/serve_hostnames.go                                            | contains reference to $HOME/.kube       |
| Need to update | test/e2e/network/scale/localrun/ingress_scale.go                                        | contains reference to $HOME/.kube       |
| Need to update | test/soak/serve_hostnames/README.md                                                     | contains reference to $HOME/.kube       |
| Need to update | translations/kubectl/it_IT/LC_MESSAGES/k8s.po                                           | contains reference to $HOME/.kube       |
| Need to update | translations/kubectl/en_US/LC_MESSAGES/k8s.po                                           | contains reference to $HOME/.kube       |
| Need to update | translations/kubectl/ja_JP/LC_MESSAGES/k8s.po                                           | contains reference to $HOME/.kube       |
| Need to update | translations/kubectl/default/LC_MESSAGES/k8s.po                                         | contains reference to $HOME/.kube       |
| Need to update | translations/kubectl/template.pot                                                       | contains reference to $HOME/.kube       |
| Need to update | translations/kubectl/de_DE/LC_MESSAGES/k8s.po                                           | contains reference to $HOME/.kube       |
| Need to update | translations/kubectl/zh_CN/LC_MESSAGES/k8s.po                                           | contains reference to $HOME/.kube       |
| Need to update | https://github.com/kubernetes-sigs/controller-runtime/blob/cb7f85860a8cde7259b35bb84af1fdcb02c098f2/pkg/client/config/config.go#L129 | Check with project       |

### Test Plan

#### Alpha milestones

Unit tests matching:
  - update kubectl tests to the new location

#### Beta milestones

Review all unit tests in the main kubernetes tree

#### GA milestones
All code should be updated with the new config location

### Graduation Criteria

Successful Alpha Criteria
  - Migrate from $HOME/.kube to $HOME/.config in a transparent way to users
  - Update Unit tests

#### Alpha -> Beta Graduation

- [x] At least 2 release cycles pass to gather feedback and bug reports during
  real-world usage
- [x] End-user documentation is written

#### Beta -> GA Graduation

- [x] At least 2 release cycles pass to gather feedback and bug reports during
  real-world usage
- [x] Unit tests must pass and linked in the KEP
- [x] Documentation exists

### Upgrade / Downgrade Strategy

Users that upgrade to a recent version of kubectl will be able to migrate
to $HOME/.config/kube without intervention. If users decide to downgrade, they still can
use the configuration in $HOME/.kube.

### Version Skew Strategy

## Drawbacks

As soon the feature became achieve GA Graduation, automation scripts that use $HOME/.kube
as static source will outdated.

## Alternatives

- Use KUBECONFIG env var
- Keep using $HOME/.kube with old versions kubectl and avoid using XDG Base Directory Specification

## Implementation History

- *2020-10-22*: Created KEP
- *2021-01-02*: Updated with comments from DirectXMan12 and kikisdeliveryservice
- *2021-01-03*: Updated with comments from wojtek-t
- *2021-01-04*: Updated with comments from rikatz
- *2021-01-08*: Fixed typo, thanks rikatz
- *2021-01-09*: Updated KEP new approach
- *2021-01-12*: Updated KEP comments from liggitt
