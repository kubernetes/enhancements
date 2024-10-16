---
title: Adding the Support for Encrypted Images
authors:
  - "@harche"
owning-sig: sig-node
participating-sigs:
  - sig-architecture
reviewers:
  - smarterclayton
  - tallclair
  - yujuhong
approvers:
  - smarterclayton
  - tallclair
creation-date: 2019-05-16
status: provisional
---

# Adding the Support for Encrypted Images

## Table of Contents

<!-- toc -->

* [Summary](#summary)
* [Motivation](#motivation)
  * [Goals](#goals)
  * [Non\-Goals](#non-goals)
  * [User Stories](#user-stories)
* [Proposal](#proposal)
  * [API](#api)
    * [Key Secret Definition](#key-Secret-definition)
    * [Image Handler](#Image-handler)
    * [Container Handler](#Container-handler)
  * [Relationship with imagePullPolicy](#Relationship-with-imagePullPolicy)
  * [Runtime Compatibility](#Runtime-compatibility)
  * [Threat Model](#Threat-model)
  * [Consumption of the ImageDecryptSecrets](#Consumption-of-the-ImageDecryptSecrets)
* [Alternatives Considered](#alternatives-considered)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)

 <!-- /toc -->

## Summary

The underlying specification for the containers, the OCI spec, is soon going to support encrypted images. Kubernetes should be able to support decryption of these encrypted images with the addition of a new type of `Secret`, which we would like to call `ImageDecryptSecret`. 

Along with OCI spec, there is an ongoing effort to enable the support for encrypted images in containerd. 

OCI Spec Issue - https://github.com/opencontainers/image-spec/issues/747

OCI Spec PR - https://github.com/opencontainers/image-spec/pull/775

Containerd StreamProcessors PR - https://github.com/containerd/containerd/pull/3482 and the container image decryption library that works with it, https://github.com/stefanberger/imgcrypt/

POC - https://github.com/harche/kubernetes/tree/pr_branch

Suported CRI-O implementation - https://github.com/harche/cri-o/tree/pull_img_auth 


## Motivation

The kubernetes worker nodes are where container images are pulled by a runtime such as `containerd`. If the images pulled are encrypted then containerd will have to decrypt them before running the pod. In order to be able to decrypt the images, the worker node needs to have access to corresponding the private keys.

Kubernetes `Secrets` are used to securely deliver sensitive data to pods in corresponding worker nodes. We need to have a secret that can be utilized *before* the pod is provisioned. Regular Kubernetes secret gets attached to the pod as tmpfs mount after the pod is started. However, there exists another type of kubernetes secret called `ImagePullSecrets`. ImagePullSecrets are used to pull the images from the private container image registry, hence they contain login credentials. `ImagePullSecrets` get utilized *before* the pod is started, this is exactly the same kind of requirement for being able to decrypt the encrypted container image. We need to have a `secret` that can be used *before* the pod is provisioned to decrypt the image. Hence, we are submitting this KEP to kubernetes community to propose a new type of kubernetes secret that is modeled after `ImagePullSecret`, called `ImageDecryptSecret`. While the `ImagePullSecret` holds the login credentials for the private registry, the `ImageDecryptSecret` will hold the private keys to decrypt encrypted images.


### Goals

- Introduce a new type of secret, `ImageDecryptSecret` to represent the necessary key(s) required to decrypt the contents of the image. 
- Define how `ImageDecryptSecret` can be used by configuration yaml(s) of the Pod (or Deployments)
- Define how `ImageDecryptSecret` can be integrated into the service accounts
- Define the Image Authorization process to prevent unauthorized access to the cached encrypted images. 


### Non-Goals

- Kubernetes should be able to decrypt the images. However, in a typical workflow kubernetes has no role to play to encrypt the images. This is similar to how kubernetes plays a role in downloading and using the images instead of building and uploading container images to the registry.  

### User Stories

- As a cluster user, I want to create a secret that carries the private keys required for my encrypted images
- As a cluster user, I want to run the encrypted container images using the private keys carried by ImageDecryptSecret
- As a cluster operator, I want to add the ImageDecryptSecret to the service account
- As a cluster user, I want to run the encrypted images using the private keys from the service account
- As an application developer, I want to encrypt my container images and be able to run them securely using kubernetes
- As an application developer, I want to protect the content of my container images such that only me (as an application developer) and the execution runtime can read them. Any other third party, such as container registry, should **NOT** be able to read the content of my container image.

## Proposal

The initial design includes:

- `ImageDecryptSecret` API resource definition
- `ImageDecryptSecret` pod field for specifying the ImageDecryptSecret the pod should be run with
- Kubelet implementation for fetching & interpreting the ImageDecryptSecret
- CRI API & implementation for passing along the ImageDecryptSecret

### API

`ImageDecryptSecret` is a new cluster-scoped Secret


The private key(s) is selected by the pod by specifying the ImageDecryptSecret in the PodSpec. Once the pod is
created, the ImageDecryptSecret cannot be changed.

_(This is a simplified declaration, syntactic details will be covered in the API PR review)_

Let's begin by defining the private key. A private key consists of binary key data and an optional password to unlock that private key.
```go
type PrivateKey struct {
    // keyData represents a private key in format DER/PEM or GPG private keyrings.  These keys will be used by JWE/PKCS7/PGP protocols.
    keyData []byte
    // keyPass represents the (optional) password to unlock the private key
    keyPass []byte
}
```

A single `ImageDecryptSecret` encapsulates multiple private keys (and their corresponding but optional passwords) for the following reasons:

1. The layers of the single image maybe encrypted with different keys. 
2. As we will further down, an `ImageDecryptSecret` is defined at the Pod level. Because of this, it can be used to decrypt multiple images of that pod which might be have been encrypted with different keys.

So here we define `DecryptionKeys` which is just a list of `PrivateKey`
```go
type DecryptionKeys []PrivateKey

```
We then use `DecryptionKeys` to store the required keys using the key `ImageDecryptionKey` as shown below. 

```go
const (
  <snip>


  // ImageDecryptionKey represent the key required to access secret data
  ImageDecryptionKey = ".imagedecryptionkey"
  
  // SecretTypeDecryptKeys defines the type for the decrypt secrets
  // Required at least one of fields:
  // - Secret.Data[".imagedecryptionkeys"] - a serialized instance of DecryptionKeys
  // authentication
  SecretTypeDecryptKeys SecretType = "kubernetes.io/decryptionkeys"

  </snip>
)
```

```go
// PodSpec is a description of a pod
type PodSpec struct {
  <snip>
  // ImageDecryptSecrets is an optional list of references to secrets in the same namespace to use for decrypting any of the images used by this PodSpec.
  // If specified, these secrets will be passed to individual puller as well as container creation implementations for them to use.
  // +optional
  ImageDecryptSecrets []LocalObjectReference
  </snip>
}
```

```go
type ServiceAccount struct {
  <snip>
  // ImageDecryptSecrets is a list of references to secrets in the same namespace to use for decrypting any encrypted images
  // in pods that reference this ServiceAccount. 
  // +optional
  ImageDecryptSecrets []LocalObjectReference `json:"imageDecryptSecrets,omitempty" protobuf:"bytes,5,rep,name=imageDecryptSecrets"`
  </snip>
```

An unspecified `nil` or empty `""` ImageDecryptSecret is equivalent to the backwards-compatible
default behavior as if the ImageDecryptSecret feature is disabled.

#### Examples

Suppose we operate a cluster that lets users create a secret of type `ImageDecryptSecret`. 

For the kubectl command line, we propose that the user create a secret of type `image-decrypt` and give it a name. One or multiple private keys may then be stored under this secret.

```bash
kubectl create secret image-decrypt <secret name> --decrypt-secret=/path/to/private_key[:<password>] [--decrypt-secret=/path/to/private_key[:<password>]]
```
`private key` - Private keys represented in format DER/PEM or GPG private keyrings.  These keys will be used by JWE/PKCS7/PGP protocols.

`password` (optional) - password in *cleartext* if the given private key is protected with it.

For example,
```bash
kubectl create secret image-decrypt keysecret --decrypt-secret="/home/user/keys/private.key":"password"
```

When a user creates a workload, they can choose the desired ImageDecryptSecret to use 

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx
  labels:
    name: nginx
spec:
  containers:
  - name: nginx
    image: localhost:5000/nginx:enc
    ports:
    - containerPort: 80
  imageDecryptSecrets:
  - name: keysecret
```

ImageDecryptSecrets can be added to the `service account` by,

```bash
kubectl patch serviceaccount <account name> -p '{"imageDecryptSecrets":[{"name":<secret name>]}'
```
or while creating a secret account,
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  creationTimestamp: 2015-08-07T22:02:39Z
  name: default
  namespace: default
  selfLink: /api/v1/namespaces/default/serviceaccounts/default
  uid: 052fb0f4-3d50-11e5-b066-42010af0d7b6
secrets:
- name: default-token-uudge
imagePullSecrets:
- name: myregistrykey
imageDecryptSecrets:
- name: <secret name>
```

For example,
```bash
kubectl patch serviceaccount default -p '{"imageDecryptSecrets":[{"name":"keysecret"}]}'
```

#### Key Secret Definition

The privake keys are extracted from `ImageDecryptSecret` and passed to the CRI by bundling them in `ImageDecryptParam` as a part of `PullImageRequest` (see, [Image Handler](#Image-handler)) and `CreateContainerRequest` (see, [Container Handler](#Container-handler)):


```protobuf
// ImageDecryptParam represents a single private key (and optional password) that can be sent to the CRI via `PullImageRequest` as well as `CreateContainerRequest`
message ImageDecryptParam {
    // key_data represents a private key in format DER/PEM or GPG private keyrings.  These keys will be used by JWE/PKCS7/PGP protocols.
    bytes key_data = 1;
    // key_pass represents the (optional) password to unlock the private key
    bytes key_pass = 2;
}
```

API doesn't need to explicitly define the key protocol. Kubernetes passes the given key(s) to runtime _as is_. The runtime should have the logic to handle the key protocol. e.g. The library that will be used by `CRI-O` to handle decryption already has functions to infer the key protocol from the key data, [as seen here](https://github.com/containers/ocicrypt/blob/master/utils/utils.go#L72). 

#### Image Handler
The list of `ImageDecryptParam` is sent to runtime via `PullImageRequest` to perform image decryption.

```protobuf
message PullImageRequest {
    // Spec of the image.
    ImageSpec image = 1;
    // Authentication configuration for pulling the image.
    AuthConfig auth = 2;
    // Config of the PodSandbox, which is used to pull image in PodSandbox context.
    PodSandboxConfig sandbox_config = 3;
    // ImageDecryptParam for the images service of the CRI
    repeated ImageDecryptParam dcparams = 4;
}
```

#### Container Handler

The list of `ImageDecryptParam` is also sent to runtime via `CreateContainerRequest` to perform image authorization.

```protobuf
 message CreateContainerRequest {
    // ID of the PodSandbox in which the container should be created.
    string pod_sandbox_id = 1;
    // Config of the container.
    ContainerConfig config = 2;
    // Config of the PodSandbox. This is the same config that was passed
    // to RunPodSandboxRequest to create the PodSandbox. It is passed again
    // here just for easy reference. The PodSandboxConfig is immutable and
    // remains the same throughout the lifetime of the pod.
    PodSandboxConfig sandbox_config = 3;
    // ImageDecryptParam for the container service of the CRI
    repeated ImageDecryptParam dcparams = 4;
}
```



### Relationship with imagePullPolicy

`ImageDecryptSecrets` are designed by taking the inspiration from `ImagePullSecrets` due similarities in how they are consumed. Both the secrets need to provided to the runtime _before_ the pod is provisioned.

But unlike the `ImagePullSecrets`, `ImageDecryptSecrets` require user to go through authorization process _irrespective_ of the `imagePullPolicy`.

| imagePullPolicy        | New Image           | Cached Image  |
| ------------- |:-------------:| -----:|
| Always | Keys Required | Keys Required |
| IfNotPresent      | Keys Required   |  Keys Required |
| Never | N/A  | Keys Required |


### Runtime Compatibility

As a part of this proposal, CRI interface is extended to pass the decryption keys to the runtime during `PullImageRequest` as well as `CreateContainerRequest`. 

Runtimes that implement this updated CRI interface should be capable of decrypting the image using the decryption keys received via `PullImageRequest`. The decryption keys received via `CreateContainerRequest` should be used by the runtime to perform `Image Authorization` in order to prevent unauthorized access to cached images. 

Runtimes that do no implement this updated CRI interface would continue to function **normally** as long as they are not using encrypted images or they are capable of fetching the decryption keys by themselves.

However, should such a runtime encounter an encrypted image and is also incapable of fetching the required keys by itself will experience error during the untarring of the image layers. It will be an error during pulling the image as a part of `PullImageRequest` and will be propogated back to kubelet like any other error occurred during `PullImageRequest`. There is no possibility of of error occurring during `CreateContainerRequest` by such runtime as `PullImageRequest` would never succeed.

### Threat Model

Encryption ties trust to an entity. These entities can be users or worker nodes. Each of which has unique use cases.

1. Encryption binding to Users - in this model, the trust of encryption is tied to the cluster or users within a cluster. This allows multi-tenancy of users, and is useful in the case where multiple users of kubernetes each want to bring their own encrypted images. This KEP is about implementing this threat model.

2. Encryption binding to workers - In this model encryption is tied to workers. The usecase here revolves around the idea that an image should be only decryptable only on trusted host. Although the granularity of access is more relaxed (per node), it is beneficial because there various node based technologies which help bootstrap trust in worker nodes and perform secure key distribution (i.e. TPM, host attestation, secure/measured boot). In this scenario, runtimes are capable of fetching the necessary decryption keys. An example of this is an ongoing effort in CRI-O, https://github.com/cri-o/cri-o/pull/2813

### Consumption of the ImageDecryptSecrets

As seen in the diagram https://imgur.com/zQaAPp5, when the kubelet wants to create a new pod which has encrypted images it has to first retrieve the referenced `ImageDecryptSecrets` which hold the decryption keys (They can be referenced directly in the pod or deployment yaml or in the pod’s service account). 

Kubelet sends the request to pull the image along with the decryption keys to CRI which then forwards it to Containerd/CRI-O. Containerd/CRI-O looks up for the image in the corresponding snapshotter, if the image doesn’t exist then it's downloaded and decrypted using the decryption keys that were passed via CRI. 

`Image Authorization` is the process that only attempts to unwrap the keys in the image manifest of the image. In simple terms, it means everytime you want to use an encrypted image, you will have to prove that you have the necessary keys to decrypt it even if the actual image is present in the decrypted form in the snapshotter. This prevents an attack where a user, without having decryption keys, might get access to encrypted image content if their pod gets scheduled on a worker node where that particular encrypted image was already pulled and decrypted by earlier request.

In case Containerd/CRI-O finds the image in the snapshotter cache which was already pulled and decrypted by earlier request, it will not perform the `Image Authorization` as a part of the call to pull the images. Kubelet makes a subsequent call to `CreateContainer` and passes the decryption keys with it, which a runtime like containerd/CRI-O can use to perform the image authorization. The advantage of performing image authorization during `CreateContainer` is that it allows the image authorization to take place _irrespective_ of the `imagePullPolicy` of the pod.

## Alternatives Considered

In order to decrypt an image containerd needs to have access to the corresponding private key. By the nature of it, a private key is a very sensitive piece of data. If it's lost, image confidentiality is compromised. It's a kind of a `secret` that's securing the data in an encrypted image. Kubernetes already has an infrastructure to handle secrets. This was the motivation to extend existing secrets to handle private keys required to decrypt an image. 

Containerd (or CRI-O) is the component that actually pulls the image, and hence does the decryption, on the worker node. So alternatively, if containerd manages to fetch keys on its own then we do not need Kubernetes to provision them. We did a POC around this idea where containerd talks to a Key Management Service (KMS) provider to fetch appropriate keys before decrypting the image. While this solution works as intended, the user needs to set up and maintain KMS. 

Using existing secret management in Kubernetes to provide decryption keys simplifies user flow, although using containerd to fetch keys also has it's use cases (mainly where the k8s master is not well trusted). 

## Graduation Criteria

Alpha:

- [] Everything described in the current proposal:
  - [] Introduce the ImageDecryptSecret API resource
  - [] Add a ImageDecryptSecret field to the PodSpec
  - [] Add a ImageDecryptSecret field to the CRI `PullImageRequest` and `CreateContainerRequest`
  - [] Plumb through the ImageHandler and ContainerHandler in the Kubelet
  - [] `kubectl` command to create a secret by using the private key
- [] ImageDecryptSecret support in CRI-Containerd and CRI-O
  - [] An error is reported when the private key is invalid or is unknown or unsupported
- [] Testing
  - [] Kubernetes Unit Test cases
  - [] [CRI validation tests][cri-validation]

Beta:
- [] Testing
  - [] Kubernetes E2E tests (only validating single image handler and container handler cases)

 [cri-validation]: https://github.com/kubernetes-sigs/cri-tools/blob/master/docs/validation.md

## Implementation History
- 2018-11-26: Initial KEP [published](https://github.com/kubernetes/community/issues/2970)
- 2019-09-19: Modified KEP to support image authorization while creating a container.
