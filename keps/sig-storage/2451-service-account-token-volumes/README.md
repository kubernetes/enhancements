# Service Account Token Volumes [Replaced by [Bound Service Account Tokens](https://github.com/kubernetes/enhancements/tree/master/keps/sig-auth/1205-bound-service-account-tokens)]

## Table of Contents

<!-- toc -->

- [Summary](#summary)
- [Motivation](#motivation)
- [Proposal](#proposal)
  - [Token Volume Projection](#token-volume-projection)
  - [File Permission](#file-permission)
    - [Proposed heuristics](#proposed-heuristics)
    - [Alternatives considered](#alternatives-considered)
  - [Alternatives](#alternatives)
- [Graduation Criteria](#graduation-criteria)
<!-- /toc -->

## Summary

Kubernetes is able to provide pods with unique identity tokens that can prove
the caller is a particular pod to a Kubernetes API server. These tokens are
injected into pods as secrets. This proposal proposes a new mechanism of
distribution with support for improved service account tokens and explores how
to migrate from the existing mechanism backwards compatibly.

## Motivation

Many workloads running on Kubernetes need to prove to external parties who they
are in order to participate in a larger application environment. This identity
must be attested to by the orchestration system in a way that allows a third
party to trust that an arbitrary container on the cluster is who it says it is.
In addition, infrastructure running on top of Kubernetes needs a simple
mechanism to communicate with the Kubernetes APIs and to provide more complex
tooling. Finally, a significant set of security challenges are associated with
storing service account tokens as secrets in Kubernetes and limiting the methods
whereby malicious parties can get access to these tokens will reduce the risk of
platform compromise.

As a platform, Kubernetes should evolve to allow identity management systems to
provide more powerful workload identity without breaking existing use cases, and
provide a simple out of the box workload identity that is sufficient to cover
the requirements of bootstrapping low-level infrastructure running on
Kubernetes. We expect that other systems to cover the more advanced scenarios,
and see this effort as necessary glue to allow more powerful systems to succeed.

With this feature, we hope to provide a backwards compatible replacement for
service account tokens that strengthens the security and improves the
scalability of the platform.

## Proposal

Kubernetes should implement a ServiceAccountToken volume projection that
maintains a service account token requested by the node from the TokenRequest
API.

### Token Volume Projection

A new volume projection will be implemented with an API that closely matches the
TokenRequest API.

```go
type ProjectedVolumeSource struct {
  Sources []VolumeProjection
  DefaultMode *int32
}

type VolumeProjection struct {
  Secret *SecretProjection
  DownwardAPI *DownwardAPIProjection
  ConfigMap *ConfigMapProjection
  ServiceAccountToken *ServiceAccountTokenProjection
}

// ServiceAccountTokenProjection represents a projected service account token
// volume. This projection can be used to insert a service account token into
// the pods runtime filesystem for use against APIs (Kubernetes API Server or
// otherwise).
type ServiceAccountTokenProjection struct {
  // Audience is the intended audience of the token. A recipient of a token
  // must identify itself with an identifier specified in the audience of the
  // token, and otherwise should reject the token. The audience defaults to the
  // identifier of the apiserver.
  Audience string
  // ExpirationSeconds is the requested duration of validity of the service
  // account token. As the token approaches expiration, the kubelet volume
  // plugin will proactively rotate the service account token. The kubelet will
  // start trying to rotate the token if the token is older than 80 percent of
  // its time to live or if the token is older than 24 hours.Defaults to 1 hour
  // and must be at least 10 minutes.
  ExpirationSeconds int64
  // Path is the relative path of the file to project the token into.
  Path string
}
```

A volume plugin implemented in the kubelet will project a service account token
sourced from the TokenRequest API into volumes created from
ProjectedVolumeSources. As the token approaches expiration, the kubelet volume
plugin will proactively rotate the service account token. The kubelet will start
trying to rotate the token if the token is older than 80 percent of its time to
live or if the token is older than 24 hours.

To replace the current service account token secrets, we also need to inject the
clusters CA certificate bundle. Initially we will deploy to data in a configmap
per-namespace and reference it using a ConfigMapProjection.

A projected volume source that is equivalent to the current service account
secret:

```yaml
sources:
  - serviceAccountToken:
      expirationSeconds: 3153600000 # 100 years
      path: token
  - configMap:
      name: kube-cacrt
      items:
        - key: ca.crt
          path: ca.crt
  - downwardAPI:
      items:
        - path: namespace
          fieldRef: metadata.namespace
```

This fixes one scalability issue with the current service account token
deployment model where secret GETs are a large portion of overall apiserver
traffic.

A projected volume source that requests a token for vault and Istio CA:

```yaml
sources:
  - serviceAccountToken:
      path: vault-token
      audience: vault
  - serviceAccountToken:
      path: istio-token
      audience: ca.istio.io
```

### File Permission

The secret projections are currently written with world readable (0644,
effectively 444) file permissions. Given that file permissions are one of the
oldest and most hardened isolation mechanisms on unix, this is not ideal.
We would like to opportunistically restrict permissions for projected service
account tokens as long we can show that they won’t break users if we are to
migrate away from secrets to distribute service account credentials.

#### Proposed heuristics

- _Case 1_: The pod has an fsGroup set. We can set the file permission on the
  token file to 0600 and let the fsGroup mechanism work as designed. It will
  set the permissions to 0640, chown the token file to the fsGroup and start
  the containers with a supplemental group that grants them access to the
  token file. This works today.
- _Case 2_: The pod’s containers declare the same runAsUser for all containers
  (ephemeral containers are excluded) in the pod. We chown the token file to
  the pod’s runAsUser to grant the containers access to the token. All
  containers must have UID either specified in container security context or
  inherited from pod security context. Preferred UIDs in container images are
  ignored.
- _Fallback_: We set the file permissions to world readable (0644) to match
  the behavior of secrets.

This gives users that run as non-root greater isolation between users without
breaking existing applications. We also may consider adding more cases in the
future as long as we can ensure that they won’t break users.

#### Alternatives considered

- We can create a volume for each UserID and set the owner to be that UserID
  with mode 0400. If user doesn't specify runAsUser, fetching UserID in image
  requires a re-design of kubelet regarding volume mounts and image pulling.
  This has significant implementation complexity because:
  - We would have to reorder container creation to introspect images (that
    might declare USER or GROUP directives) to pass this information to the
    projected volume mounter.
  - Further, images are mutable so these directives may change over the
    lifetime of the pod.
  - Volumes are shared between all pods that mount them today. Mapping a
    single logical volume in a pod spec to distinct mount points is likely a
    significant architectural change.
- We pick a random group and set fsGroup on all pods in the service account
  admission controller. It’s unclear how we would do this without conflicting
  with usage of groups and potentially compromising security.
- We set token files to be world readable always. Problems with this are
  discussed above.

### Alternatives

1.  Instead of implementing a service account token volume projection, we could
    implement all injection as a flex volume or CSI plugin.
    1.  Both flex volume and CSI are alpha and are unlikely to graduate soon.
    1.  Virtual kubelets (like Fargate or ACS) may not be able to run flex
        volumes.
    1.  Service account tokens are a fundamental part of our API.
1.  Remove service accounts and service account tokens completely from core, use
    an alternate mechanism that sits outside the platform.
    1.  Other core features need service account integration, leading to all
        users needing to install this extension.
    1.  Complicates installation for the majority of users.

## Graduation Criteria

TBD
