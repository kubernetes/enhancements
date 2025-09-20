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
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
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
# KEP-5497: Add new imagePullPolicy: IfNewerNotPresent that pulls an image if image ID (ref/digest) on remote registry is not matching.

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
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
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
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
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

The proposed `imagePullPolicy` mode, `IfNewerNotPresent`, introduces a more efficient mechanism for managing container image retrieval in Kubernetes.
With this policy, the kubelet first checks whether the requested image exists locally. If no local copy is available, the image is pulled from the remote registry. 
When a local copy does exist, the kubelet compares the local image’s reference ID (digest) with the version available in the registry.
If the image ID (digest) differ, the newer image is pulled; if they match, the local image is reused. 
This approach ensures that workloads run with the latest image when updates are available with the same tag (e.g :latest,:staging, etc), 
while avoiding unnecessary network operations and reducing pod startup latency in cases where the local image is already up to date and present.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Currently, Kubernetes supports three `imagePullPolicy` modes: `Always`, `IfNotPresent`, and `Never`. 
While these modes cover common scenarios, they leave a gap for users who want to ensure images are up to date without forcing a pull on every pod start.

The `Always` policy guarantees the newest image is used, but it introduces significant overhead by pulling on every restart, even when the local image is already up to date. 
Conversely, IfNotPresent avoids unnecessary pulls but risks running stale images or out of date, if a new version has been pushed to the registry under the same tag. 
This behavior can lead to inconsistencies, especially in environments where tags like latest or rolling build tags are used.

The proposed `IfNewerNotPresent` policy addresses this gap by providing a smarter middle ground between `Always` and `IfNotPresent`. 
It ensures that images are refreshed only when the local digest does not match the remote digest, avoiding unnecessary downloads while still keeping workloads aligned with the most recent image.
This reduces startup latency, saves bandwidth, and ensures more consistent rollouts in dynamic environments.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Introduce a new `imagePullPolicy` mode, `IfNewerNotPresent`, to provide a balance between freshness and efficiency when pulling container images.

- Ensure workloads automatically run the most up-to-date image available in the registry without requiring an unconditional pull on every pod startup.

- Reduce unnecessary network traffic and registry load by avoiding redundant image downloads when the local image digest matches the remote digest.

- Improve pod startup times in clusters where images are frequently reused but may also be updated under the same tag.

- Provide a predictable and intuitive policy that can serve as a safer alternative to `IfNotPresent` for environments using mutable tags like `latest` or similar CI generated tags.

- Enable seamless operation with private registries by supporting authenticated access to fetch image digests, ensuring the policy works reliably in secured environments and 
uses existing kubernetes components like `imagePullSecrets` to achieve that.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- This KEP does not propose changes to how Kubernetes handles immutable versus mutable tags; it assumes tags may be mutable, but it does not enforce tag immutability.

- It does not introduce `image verification`, `signing`, or `trust policies`; those are handled separately by existing Kubernetes or container runtime mechanisms.

- This KEP does not modify the behavior of `Always`, `IfNotPresent`, or `Never` policies; those continue to work as before.

- It does not attempt to optimize registry interactions beyond digest comparison; network performance improvements are limited to avoiding unnecessary pulls.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

This KEP proposes the introduction of a new `imagePullPolicy` mode named `IfNewerNotPresent`. The desired outcome is to provide a more efficient and 
predictable image-pulling strategy that ensures workloads use the latest available image when updates exist, while avoiding unnecessary downloads and
startup delays.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

As a Kubernetes developer running workloads built from a CI/CD pipeline, I often push updated images to a registry under a tag like `latest`, `testing or `staging`. 
With the current `IfNotPresent` policy, my cluster may continue running stale images even after a new build has been published. 
Switching to `Always` guarantees freshness, but it slows down pod startup times and adds unnecessary network traffic because images are pulled on every restart.

With the new IfNewerNotPresent policy, I can ensure that my pods **automatically and always** pick up the newest image **only** when the digest in the registry changes. 
This keeps my workloads aligned with the latest builds, improves startup performance, and reduces registry load—without me needing to manually delete local images or rely on a full Always pull policy.

#### Story 2

As an application developer supporting a critical service, I occasionally need to release emergency bugfixes / backports by rebuilding and re-publishing an image under the same version tag (for example, v1.2.3). 
Today, if my cluster uses `IfNotPresent`, workloads already running that image will not pick up the fix, because the local image satisfies the policy even though it’s outdated.
Switching to `Always` forces every pod to pull on restart, which introduces delays and unnecessary bandwidth use across multiple environments. Also it causes a lot to registry in scale of thousands of clusters and nodes.

With the `IfNewerNotPresent` policy, I can push a backported image under the existing tag, and the kubelet will automatically detect the digest difference and pull the corrected version. This ensures that urgent fixes
reach workloads quickly and reliably, without creating excessive strain on our registry or slowing down rollouts.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

- Digest resolution overhead: Checking the remote image digest introduces an extra network request for each image that exists locally. While lighter than pulling the full image, it may add latency, particularly for large numbers of pods or images.

- Registry compatibility and availability: Some container registries may not fully support digest queries or may impose rate limits. Users of such registries may see unexpected failures or throttling when using this policy. Retries are implenented in place to avoid rate limits or network issues.

- Tag mutability assumptions: `IfNewerNotPresent` assumes that image tags may be updated with new digests. If users rely on immutable tags, the policy behaves similarly to `IfNotPresent`.

Race conditions during rollout: In large clusters, multiple nodes may detect a digest mismatch and pull the updated image simultaneously, potentially creating short-term registry load spikes. Same as using `Always` or while updating an app with `IfNotPresent` and new tags.

Interaction with imagePullSecrets and private registries: Proper authentication is required for the kubelet to fetch digests. Misconfigured credentials may result in failed pulls.

No enforcement of semantic versioning or tag conventions: This policy does not interpret tag names; it relies solely on digest comparison. Users must manage tag discipline themselves.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

- Authentication failures with private registries: Nodes may fail to fetch digests if `imagePullSecrets` are misconfigured.

  - Logging of authentication failure, same as kubernetes core functionality with other `imagePullPolicies`.

- User confusion over policy behavior: Developers may misunderstand the difference between `IfNewerNotPresent`, `IfNotPresent`, and `Always`.

  - Clear documentation, examples, and release notes explaining the digest comparison logic will be provided.

- Security considerations: Malicious images with manipulated digests are mitigated by relying on container runtime signature verification (e.g., Notary, Cosign). 
  - The policy itself does not weaken existing security mechanisms.

- UX - review: SIG Node and SIG Docs will review the behavior and documentation, ensuring clarity for cluster operators and developers. 
  - Feedback from CI/CD tooling teams and cloud providers will be incorporated to verify real-world usability.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

The `IfNewerNotPresent` policy introduces a new image pull behavior within the kubelet that leverages both local caching and digest comparison. 

Prototype working code (main function modified):
```
// imagePullPrecheck inspects the pull policy and checks for image presence accordingly,
// returning (imageRef, error msg, err) and logging any errors.
func (m *imageManager) imagePullPrecheck(ctx context.Context, objRef *v1.ObjectReference, logPrefix string, pullPolicy v1.PullPolicy, pullSecrets []v1.Secret, spec *kubecontainer.ImageSpec, requestedImage string) (imageRef string, msg string, err error) {

	switch pullPolicy {
		
	case v1.PullAlways:

		// always pull image
		return "", msg, nil

	case v1.PullIfNewerNotPresent:

		// Get local image id (digest)
		imageRef, err = m.imageService.GetImageRef(ctx, *spec)
		if err != nil {
			msg = fmt.Sprintf("[PullIfNewerNotPresent] Failed to inspect image %q: %v", imageRef, err)
			m.logIt(objRef, v1.EventTypeWarning, events.FailedToInspectImage, logPrefix, msg, klog.Warning)
			return "", msg, ErrImageInspect
		}

		// Get remote image id (digest)
		remoteImageRef, _ := GetRemoteImageDigestWithoutPull(ctx, requestedImage, pullSecrets)

		// Compare digests if not match then pull image from remote
		if remoteImageRef != imageRef {
			msg = fmt.Sprintf("Remote image for %q changed (local=%s, remote=%s), will pull new one.", requestedImage, imageRef, remoteImageRef)
			m.logIt(objRef, v1.EventTypeWarning, events.FailedToInspectImage, logPrefix, msg, klog.Warning)
			return "", msg, err
		}

		msg = fmt.Sprintf("Remote image for %q is exactly the same (local=%s, remote=%s), using existing one.", requestedImage, imageRef, remoteImageRef)
		m.logIt(objRef, v1.EventTypeWarning, events.FailedToInspectImage, logPrefix, msg, klog.Warning)

	case v1.PullIfNotPresent:
		// check if image exists and return image ID
		imageRef, err = m.imageService.GetImageRef(ctx, *spec)
		if err != nil {
			msg = fmt.Sprintf("[PullIfNotPresent/PullNever] Failed to inspect image %q: %v", imageRef, err)
			m.logIt(objRef, v1.EventTypeWarning, events.FailedToInspectImage, logPrefix, msg, klog.Warning)
			return "", msg, ErrImageInspect
		} 
}

```

Prototype working code (helper function added):
```
// GetRemoteImageDigestWithoutPull fetches the digest of a remote image
// without pulling its layers. It supports public and private registries via authn.DefaultKeychain.
func GetRemoteImageDigestWithoutPull(ctx context.Context, imageName string, pullSecrets []v1.Secret) (string, error) {
	// Parse the image reference
	imageRef, err := name.ParseReference(imageName)
	if err != nil {
		klog.Errorf("GetRemoteImageDigestWithoutPull failed to parse image reference %q: %v", imageName, err)
		return "", err
	}

	// Create keychain from pull secrets
	keychain := createKeychainFromSecrets(pullSecrets)

    
	// Fetch the remote image with authentication
    remoteImage, err := remote.Image(imageRef, remote.WithContext(ctx), remote.WithAuthFromKeychain(keychain))
	for i := 0; i < 30; i++ {
		if err == nil {
			break
		}
	    klog.V(4).Infof("Failed x%d to fetch remote image: %s, retrying after %d", i+1, imageName, time.Second * time.Duration(i+1))
		time.Sleep(time.Second * time.Duration(i+1))
		remoteImage, err = remote.Image(imageRef, remote.WithContext(ctx), remote.WithAuthFromKeychain(keychain))
	}
    
	if err != nil {
		return "", fmt.Errorf("failed to fetch image after retries: %w", err)
	}
    
	// Get the image ID (config digest)
	remoteImageRef, err := remoteImage.ConfigName()
	if err != nil {
		klog.Errorf("GetRemoteImageDigestWithoutPull failed to get remote image id (digest) for %q: %v", imageName, err)
		return "", err
	}

	klog.V(4).Infof("Successfully fetched remote digest: %s", remoteImageRef.String())
	return remoteImageRef.String(), nil
}
```

Other helpers functions / snippets of code:
```
// secretKeychain implements authn.Keychain for Kubernetes secrets
type secretKeychain struct {
	secrets []v1.Secret
}

func (k *secretKeychain) Resolve(target authn.Resource) (authn.Authenticator, error) ...

func getAuthenticatorFromSecret(secret *v1.Secret) (authn.Authenticator, error) ...

func createKeychainFromSecrets(pullSecrets []v1.Secret) (authn.Keychain) ...

```

The workflow is as follows:

1. **Local image check**: 

    - When a pod requests an image with `IfNewerNotPresent`, the kubelet first checks whether the image exists locally.

    - If the image does not exist, the kubelet pulls it from the registry, identical to the behavior of `IfNotPresent`.


2. **Digest comparison**: 

    - If a local image exists, the kubelet queries the remote registry for the digest corresponding to the requested image reference.

    - If the local digest matches the remote digest, the image is `reused`.

    - If the digests differ, the kubelet `pulls the newer image` from the registry and replaces the local copy.


3. **Private registry support**: 

    - Pulls and digest queries honor existing imagePullSecrets, ensuring proper authentication for private registries. Failure to authenticate results in a standard pull error.

4. **Integration with container runtimes**: 

    - The policy leverages the runtime’s existing image management APIs (e.g., docker, containerd) to inspect local images and fetch remote digests. 

    - This ensures minimal changes to existing kubelet or runtime logic.


5. **Failure handling**:

    - Case: Digest query failure**: The kubelet falls back to using the local image and logs a warning.

    - Case: Pull failure**: The pod enters the standard `ImagePullBackOff` state.


8. **Success criteria**:

    - Pods consistently run with the latest image when the remote digest changes.

    - Registry requests are minimized when the local image is up to date.

    - Policy works seamlessly with private registries and existing imagePullSecrets.


### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

Manual tests have been done to prototype code.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

<!-- - `<package>`: `<date>` - `<test coverage>` -->

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->
<!-- 
- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature) -->

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->
<!-- 
- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature) -->

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved 

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

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
N/A

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->
N/A

## Production Readiness Review Questionnaire

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

- No feature gate is required initially, as this is an additive policy that does not change existing behaviors (`Always`, `IfNotPresent`, `Never`).

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [x] Other
  - Describe the mechanism:
    - The `IfNewerNotPresent` policy will be introduced as an opt-in feature via the standard `imagePullPolicy` field in pod specs.
    - Users can enable the feature by specifying `imagePullPolicy: IfNewerNotPresent` in the container spec.

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->
Yes, choose already existing `Always`, `IfNotPresent`, `Never` policies

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

N/A

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->


###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

Already running workloads are not affected.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
N/A

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->
N/A

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->
Check describe pod for image pulling logs.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [x] Events
  - Event Reason: Image Pulling events / logs
- [ ] API .status
  - Condition name: 
  - Other field: 
- [x] Other (treat as last resort)
  - Details: Kubelet Image Pulling Logs

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->
N/A

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

N/A

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

N/A

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

No

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->
Not applicable

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
Yes, enabling or using `IfNewerNotPresent` can slightly affect the time taken for some operations, but the impact is generally minimal and predictable

- Pod startup time: For pods where the local image exists, the kubelet will make an additional request to the registry to fetch the image digest. This adds a small latency compared to IfNotPresent, which skips the digest check entirely.

- Image pull time: If the digest indicates a newer image is available, a full image pull is performed, which takes the same time as with Always.

- SLIs/SLOs affected: Operations that measure pod startup latency or overall deployment rollout time could see a minor increase for pods with digest checks. The increase is proportional to network latency and registry response times, but digest queries are lightweight compared to full image pulls that currently happen with `Always`.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
`Minimal` and `non-negligible` impact to `None` impact

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->
No

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

When using **IfNewerNotPresent**, you may encounter some scenarios that require guidance:

1. Pod fails to pull an updated image:
**Symptom:** Pod remains using the old image even after a new image is pushed under the same tag.  

**Possible causes:**
- Digest query failed due to network issues or registry unavailability.
- Authentication failure with a private registry (`imagePullSecrets` misconfigured).

**Resolution:**
- Check events/kubelet logs for digest query errors.
- Validate registry credentials and `imagePullSecrets`.
- Ensure the registry is reachable from all nodes.

2. Pod stuck in `ImagePullBackOff`:
**Symptom:** Pod cannot start because the new image cannot be pulled.  

**Possible causes:**
- Network issues preventing registry access.
- Private registry authentication failure.
- The remote image does not exist or the tag is incorrect.

**Resolution:**
- Check events/kubelet logs for errors.
- Confirm the image exists and the tag is correct.
- Verify network connectivity and credentials.
- Retry the pod or recreate it after fixing registry access.

3. Local image corruption detected
**Symptom:** Container runtime fails to start the container despite matching digest.  

**Resolution:**
- Check events/kubelet logs for errors.
- Manually remove the corrupted image from the local cache.
- Retry pod deployment to pull a fresh copy.
- Change `imagePullPolicy` to `Always` and back to `IfNewerNotPresent` after `Always` will replace the image forcibly.


###### How does this feature react if the API server and/or etcd is unavailable?

Independently, since its based on kubelet, kube-scheduler and nodes' CRI.

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
1. Registry Unavailability

Same behaviour as `IfNotPresent` or `Always`

2. Rate Limiting / Throttling

Same behaviour as `IfNotPresent` or `Always`

3. Corrupted Local Images

Same behaviour as `IfNotPresent`.

4. Tag Reuse Without Digest Change

Same behaviour as `IfNotPresent`.

5. Authentication missconfiguration 

Same behaviour as `IfNotPresent` or `Always`

6. Network Issues

Same behaviour as `IfNotPresent` or `Always`

7. Unsupported Registries

Some registries may not support digest queries (rare).

###### What steps should be taken if SLOs are not being met to determine the problem?

- Use `Always` in imagePullPolicy to re-download the image.
- Delete local image manually and use `Always` to re-download it.
- Place the image manually in the CRI.

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

The implementation has gone through `design`, `prototype`, and `manuial testing` phases. 
Further development is needed for Unit, E2E or other tests and some Enhancement of code may be needed.

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->
No

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
No

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
No
