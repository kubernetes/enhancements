# Kustomize Components

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Story](#user-story)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Why introduce a new <code>components</code> field?](#why-introduce-a-new-components-field)
  - [Why introduce a new <code>Component</code> kind?](#why-introduce-a-new-component-kind)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta](#alpha---beta)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kustomize provides an intuitive way to manage Kubernetes applications in a
purely declarative manner. However, it seems to struggle with applications that
mix multiple, optional features on demand, affecting different aspects of a
base configuration.

This enhancement proposal introduces components, a new kind of Kustomization
that allows users to define reusable kustomizations. Components can be included
from higher-level overlays to create variants of an application, with a subset
of its features enabled.

The community has shown strong interest in this feature and has been actively
discussing it in https://github.com/kubernetes-sigs/kustomize/issues/1251.

## Motivation

By design, the variant system of Kustomize is based on an inheritance model
and suggests that a base application is parameterized by stacking overlays on
top of it. While intuitive at first sight, this approach falls short of
deploying multivariate applications, i.e., ones that:

- ship with numerous opt-in features and corresponding configuration options
- target dissimilar audiences with different needs and scopes (developers,
  admins, customers, testers etc.)
- run on multiple cloud platforms (on-prem, Minikube, GKE, EKS, AKS, etc.)

The problem is that modular applications cannot always be expressed in a tall
hierarchy while preserving all combinations of available features. Doing so
would require putting each feature in an overlay, and making overlays for
independent features inherit from each other. However, this is semantically
incorrect, cannot not scale as the number of features grows, and soon results
in duplicate manifests and kustomizations.

Instead, such applications are much better expressed as a collection of
components, i.e., reusable pieces of configuration logic that are defined in a
common place and that distinct overlays can then mix-and-match. This approach
abides by the [DRY](https://en.wikipedia.org/wiki/Don%27t_repeat_yourself)
principle and increases ease of maintenance.

The simplest way to implement this in Kustomize is to create an overlay for
each component, and have a top-level overlay include only the components that
it requires. The problem with this approach is that the mechanism that
Kustomize uses to interpret and ultimately apply patches does not allow for
mutating different aspects of the very same base resource in parallel without
modifying its GVKN (unique identifier). That is, sibling overlays that operate
on the same level of the hierarchy and are not chained (i.e. one does not
import the other as base) cannot modify their common parent due to resource ID
conflicts.

For this reason, we need to provide a new type of kustomization that will help
Kustomize support the composition model, besides the inheritance model. This is
a need that is echoed in other places as well:

* https://github.com/kubernetes-sigs/kustomize/issues/171
* https://github.com/kubernetes-sigs/kustomize/issues/759
* https://github.com/kubernetes-sigs/kustomize/issues/2464
* https://github.com/kubeflow/kubeflow/pull/3108

### Goals

- Introduce the notion of components in kustomize and, eventually in kubectl
- Provide the implementation that allows users to define components, i.e.,
  portable overlays that are able to modify a set of base resources without
  conflicts, since patches are serialized
- Maintain existing, stable interfaces that end-users are already familiar
  with, i.e., offer components as an optional feature

### Non-Goals
- Overshadow resources with components
- Abolish inheritance over composition in Kustomize

## Proposal

Create a new kustomization type called `Component`, and a new kustomization
field called `components`.

A kustomization that is marked as a `Component` has basically the same
capabilities as a normal kustomization. The main distinction is that they are
evaluated **after** the resources of the parent kustomization (overlay or
component) have been accumulated, and **on top** of them. This means that:

* A component with transformers can transform the resources that an overlay has
  previously specified in the `resources` field. Components with patches do not
  have to include the target resource in their `resources` field.
* Multiple components can extend and transform the same set of resources
  **sequentially**. This is in contrast to overlays, which cannot alter the same
  base resources, because they clone and extend them **in parallel**.

In order to create a component, a user can add the following to its
`kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component
```

In order to use a component, the user can refer to it in their
`kustomization.yaml` via the `components` field:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base
  - resource1
  - resource2

components:
  - ../components/component1
  - ../components/component2
```

Note that a component cannot be added to the `resources:` list, and a
resource/`Kustomization` cannot be added to the `components:` list.

### User Story

Suppose that a user has written a very simple Web application:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
spec:
  template:
    spec:
      containers:
      - name: example
        image: example:1.0
```

They may want to deploy a **community** edition of this application as SaaS, so
they need to add support for persistence (e.g. an external database), and bot
detection (e.g. Google reCAPTCHA).

At some point, they have attracted **enterprise** customers who want to deploy
it on-premises, so they add LDAP support, and disable Google reCAPTCHA. At the
same time, as a **dev** they need to be able to test parts of the application,
so they want to deploy it with some features enabled and others not.

Here's a matrix with the deployments of this application and the features
enabled for each one:

|            | External DB        | LDAP               | reCAPTCHA          |
|------------|:------------------:|:------------------:|:------------------:|
| Community  | :heavy_check_mark: |                    | :heavy_check_mark: |
| Enterprise | :heavy_check_mark: | :heavy_check_mark: |                    |
| Dev        | :white_check_mark: | :white_check_mark: | :white_check_mark: |

(:heavy_check_mark:: enabled, :white_check_mark:: optional)

So, we want to make it easy for the user to deploy this application in any of
the above three environments. A way to solve this is to package each opt-in
feature as a component, so that it can be referred to from higher-level
overlays.

First, define a place to work:

```shell
DEMO_HOME=$(mktemp -d)
```

Define a common **base** that has a `Deployment` and a simple `ConfigMap`, that
is mounted on the application's container.

```shell
BASE=$DEMO_HOME/base
mkdir $BASE

cat <<EOF >$BASE/kustomization.yaml
resources:
- deployment.yaml

configMapGenerator:
- name: conf
  literals:
    - main.conf=|
        color=cornflower_blue
        log_level=info
EOF

cat <<EOF >$BASE/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example
spec:
  template:
    spec:
      containers:
      - name: example
        image: example:1.0
        volumeMounts:
        - name: conf
          mountPath: /etc/config
      volumes:
        - name: conf
          configMap:
            name: conf
EOF
```

Define an `external_db` component, using `kind: Component`, that creates a
`Secret` for the DB password and a new entry in the `ConfigMap`:

```shell
EXT_DB=$DEMO_HOME/components/external_db
mkdir -p $EXT_DB

cat <<EOF >$EXT_DB/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1alpha1  # <-- Component notation
kind: Component

secretGenerator:
- name: dbpass
  files:
    - dbpass.txt

patchesStrategicMerge:
  - configmap.yaml

patchesJson6902:
- target:
    group: apps
    version: v1
    kind: Deployment
    name: example
  path: deployment.yaml
EOF

cat <<EOF >$EXT_DB/deployment.yaml
- op: add
  path: /spec/template/spec/volumes/0
  value:
    name: dbpass
    secret:
      secretName: dbpass
- op: add
  path: /spec/template/spec/containers/0/volumeMounts/0
  value:
    mountPath: /var/run/secrets/db/
    name: dbpass
EOF

cat <<EOF >$EXT_DB/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: conf
data:
  db.conf: |
    endpoint=127.0.0.1:1234
    name=app
    user=admin
    pass=/var/run/secrets/db/dbpass.txt
EOF
```

Define an `ldap` component, that creates a `Secret` for the LDAP password
and a new entry in the `ConfigMap`:

```shell
LDAP=$DEMO_HOME/components/ldap
mkdir -p $LDAP

cat <<EOF >$LDAP/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component

secretGenerator:
- name: ldappass
  files:
    - ldappass.txt

patchesStrategicMerge:
  - configmap.yaml

patchesJson6902:
- target:
    group: apps
    version: v1
    kind: Deployment
    name: example
  path: deployment.yaml
EOF

cat <<EOF >$LDAP/deployment.yaml
- op: add
  path: /spec/template/spec/volumes/0
  value:
    name: ldappass
    secret:
      secretName: ldappass
- op: add
  path: /spec/template/spec/containers/0/volumeMounts/0
  value:
    mountPath: /var/run/secrets/ldap/
    name: ldappass
EOF

cat <<EOF >$LDAP/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: conf
data:
  ldap.conf: |
    endpoint=ldap://ldap.example.com
    bindDN=cn=admin,dc=example,dc=com
    pass=/var/run/secrets/ldap/ldappass.txt
EOF
```

Define a `recaptcha` component, that creates a `Secret` for the reCAPTCHA
site/secret keys and a new entry in the `ConfigMap`:

```shell
RECAPTCHA=$DEMO_HOME/components/recaptcha
mkdir -p $RECAPTCHA

cat <<EOF >$RECAPTCHA/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component

secretGenerator:
- name: recaptcha
  files:
    - site_key.txt
    - secret_key.txt

# Updating the ConfigMap works with generators as well.
configMapGenerator:
- name: conf
  behavior: merge
  literals:
    - recaptcha.conf=|
        enabled=true
        site_key=/var/run/secrets/recaptcha/site_key.txt
        secret_key=/var/run/secrets/recaptcha/secret_key.txt

patchesJson6902:
- target:
    group: apps
    version: v1
    kind: Deployment
    name: example
  path: deployment.yaml
EOF

cat <<EOF >$RECAPTCHA/deployment.yaml
- op: add
  path: /spec/template/spec/volumes/0
  value:
    name: recaptcha
    secret:
      secretName: recaptcha
- op: add
  path: /spec/template/spec/containers/0/volumeMounts/0
  value:
    mountPath: /var/run/secrets/recaptcha/
    name: recaptcha
EOF
```

Define a `community` variant, that bundles the external DB and reCAPTCHA
components:

```shell
COMMUNITY=$DEMO_HOME/overlays/community
mkdir -p $COMMUNITY

cat <<EOF >$COMMUNITY/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

components:
  - ../../components/external_db
  - ../../components/recaptcha
EOF
```

Define an `enterprise` overlay, that bundles the external DB and LDAP
components:

```shell
ENTERPRISE=$DEMO_HOME/overlays/enterprise
mkdir -p $ENTERPRISE

cat <<EOF >$ENTERPRISE/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

components:
  - ../../components/external_db
  - ../../components/ldap
EOF
```

Define a `dev` overlay, that points to all the components and has LDAP
disabled:

```shell
DEV=$DEMO_HOME/overlays/dev
mkdir -p $DEV

cat <<EOF >$DEV/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

components:
  - ../../components/external_db
  #- ../../components/ldap
  - ../../components/recaptcha
EOF
```

Now the workspace has following directories:

```shell
├── base
│   ├── deployment.yaml
│   └── kustomization.yaml
├── components
│   ├── external_db
│   │   ├── configmap.yaml
│   │   ├── dbpass.txt
│   │   ├── deployment.yaml
│   │   └── kustomization.yaml
│   ├── ldap
│   │   ├── configmap.yaml
│   │   ├── deployment.yaml
│   │   ├── kustomization.yaml
│   │   └── ldappass.txt
│   └── recaptcha
│       ├── deployment.yaml
│       ├── kustomization.yaml
│       ├── secret_key.txt
│       └── site_key.txt
└── overlays
    ├── community
    │   └── kustomization.yaml
    ├── dev
    │   └── kustomization.yaml
    └── enterprise
        └── kustomization.yaml
```

With this structure, they can create the YAML files for each deployment as
follows:

```shell
kustomize build overlays/community
kustomize build overlays/enterprise
kustomize build overlays/dev
```

### Notes/Constraints/Caveats

The distinction between components and overlays is evident when transformers
are involved, but not when a component only adds resources. This may confuse
users who will not know what to use in this case, and must be communicated
clearly in the docs.

### Risks and Mitigations

Since a component is a special type of a kustomization, that ensures
serialization of resources, the same logic and restrictions apply here as well.
This means that the threat model of components is no different from the threat
model for overlays or other kustomizations.

## Design Details

When Kustomize processes a `Kustomization` file it uses a `ResourceAccumulator`
(RA) object, which represents the accumulated state of the processing up to
that point (including files, patches, transformers, variables, etc).  The RA is
initially empty, and then Kustomize:

1. Adds each of its resources (in left-to-right order) to the RA
   (`kusttarget.accumulateResources`):
  * Adds files to the RA directly (`accumulateFile`).
  * Recursively processes Kustomizations, each starting with a new empty RA,
    and with the result merged in to the RA of the parent
    (`accumulateDirectory` and `resaccumulator.MergeAccumulator`).
2. Adds/merges its CRDs, generators, transformers, and variables to the RA.
  * These modifications can only be applied to entities within the RA - it is
    an error otherwise. This means that sibling Kustomizations are entirely
    independent of each other.
  * Kustomizations with the same base will cause an error when the resources
    are accumulated, unless their GVKN is altered.

`Components` are processed between step 1 and 2, and in a similar manner to
Kustomizations, but instead of starting with an empty RA, they take the RA from
their parent. The component is therefore able to modify everything within the
RA, which contains all of the parent's resources and the result of
processing all of the earlier components in the `components:` list. Nested
components work in the same way - ownership of the parent's RA is simply
passed down.

We implement this by changing the signature of `kusttarget.accumulateResources`
to return a pointer to the `ResourceAccumulator` that the parent should use,
and giving child components their parent's RA instead of an empty RA. This
minimizes the amount of code changed to simplify review, but a slightly more
extensive refactoring is recommended.

Additionally, we add some checks to ensure that Kustomizations and normal
files are not added to the `components:` list and Components are not added to
`resources:`.

Also, note that we do not change the code that loads the contents of
Kustomizations, so it should be possible to load Components from any place
where one expects to load a Kustomization.

### Why introduce a new `components` field?

In principle, a Component is a special case of a Kustomization, so we could
refer to it from the `resources:` field, as we do for other Kustomizations.
Unlike the rest of the resources that are referenced in this field though,
order matters with components, so the semantics of this field would need to
change. Moreover, we believe that new and existing users should easily
familiarize with this new field, if one considers that the `resources` and
`patches` fields share the same difference.

If a new `components:` field is not desired though, a variation of the
implementation could be to allow Components to be added to the `resources:`
list. In this case:

* The basic implementation of components doesn't need to change.
* It is unclear whether directories in the `resources:` list are Kustomizations
  or Components. A naming convention could be used (e.g. store components in a
  `components/` directory in the same way as overlay Kustomizations are often
  stored in `overlays/`) but this is less explicit than using a separate
  `components:` list.
* The semantics of the `resources:` field will change and order will matter.

### Why introduce a new `Component` kind?

Since we have a `components` field, we could forgo introducing a new
Kustomization kind. However, we decided to add it for two reasons:

1. The `Component` kind is alpha, which communicates to the users that its
   interface/semantics may change, so they should tread carefully.
2. A kustomization that patches resources that have not been defined in its
   `resources:` field can never work outside the `components:` field. So, the
   kind further expresses how it should be used.

### Test Plan

Add unit tests for all the points raised in the "Design details" section.

### Graduation Criteria

#### Alpha -> Beta

- [x] Implement the necessary functionality for this feature
- [x] Write the appropriate unit tests, and make the
  [`TestComplexComposition_*`](https://github.com/kubernetes-sigs/kustomize/blob/701973b73ecfb2b96f53cb3f080b9caaa662a01f/api/krusty/complexcomposition_test.go) tests now pass
- [x] Add a [user story] in Kustomize's examples
- [ ] Extend Kustomize's glossary with a reference to components, as well as
      other places where overlays are mentioned
- [ ] At least 2 release cycles pass to gather feedback and bug reports during
      real-world usage

[`TestComplexComposition_*`]: https://github.com/kubernetes-sigs/kustomize/pull/1297
[user story]: https://github.com/kubernetes-sigs/kustomize/pull/2438

## Implementation History

- Available in Kustomize's alpha API group (`kustomize.config.k8s.io/v1alpha1`) in v3.7.0+. https://kubectl.docs.kubernetes.io/guides/config_management/components/

## Alternatives

Since the creation of the
https://github.com/kubernetes-sigs/kustomize/issues/1251 issue, various
alternatives have been proposed to help in this situation. We're listing them
here in order of appearance, along with some comments on why they weren't
eventually considered:

* https://github.com/kubernetes-sigs/kustomize/pull/1217: This PR could be used
  to merge resources with the same GVKN produced by sibling kustomizations. The
  drawback is that merging YAMLs is not the same as patching the same YAML
  sequentially. For example, if two kustomizations change different fields of
  the same resource, it's possible that an old field may resurface, depending
  on the order of the merges.

* Support sharing patches between kustomizations: Instead of having components
  and composing overlays from them, we could put stock Kustomize patches in a
  common directory, and make overlays mix-and-match them. There were two
  problems with this approach:

  1. It would require using the `--load_restrictor none` flag of `kustomize
     build`.
  2. It doesn't cover the case of more complex logic that is easily defined in
     kustomizations, such as generating and transforming resources.

* Define `generators` / `transformers` and reuse them in overlays: The main
  problem with this approach is that it requires the user to redefine the handy
  generators/transformers of `kustomization.yaml` into separate files, which
  adds a bit more boilerplate.

* https://github.com/kubernetes-sigs/kustomize/issues/1292: This issue proposed
  a way to define the behavior of Kustomize when it detects an GVKN collision.
  This suggestion had the same issue as
  https://github.com/kubernetes-sigs/kustomize/pull/1217, i.e., that merging
  YAMLs is not the same as sequentially patching them.
