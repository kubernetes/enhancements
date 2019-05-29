---
title: Add plugin support to dashboard
authors:
  - "@ajatprabha"
owning-sig: sig-ui
participating-sigs:
  - kubernetes-wide
reviewers:
  - "sig-ui-leads"
approvers:
  - "sig-ui-leads"
editor: TBD
creation-date: 2019-02-05
last-updated: 2019-05-16
status: provisional
see-also:
replaces:
superseded-by:
---

# Add Plugin Support to Dashboard

## Table of Contents

- [Add Plugin Support to Dashboard](#add-plugin-support-to-dashboard)
  - [Table of Contents](#table-of-contents)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
  - [Proposal](#proposal)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Graduation Criteria](#graduation-criteria)
  - [Implementation History](#implementation-history)

## Summary

<!-- The `Summary` section is incredibly important for producing high quality user focused documentation such as release notes or a development road map.
It should be possible to collect this information before implementation begins in order to avoid requiring implementors to split their attention between writing release notes and implementing the feature itself.
KEP editors, SIG Docs, and SIG PM should help to ensure that the tone and content of the `Summary` section is useful for a wide audience.

A good summary is probably at least a paragraph in length. -->

We want to introduce a plugin mechanism in dashboard which shall enable easy functionality addition to the existing dashboard. 

## Motivation

<!-- This section is for explicitly listing the motivation, goals and non-goals of this KEP.
Describe why the change is important and the benefits to users.
The motivation section can optionally provide links to [experience reports][] to demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports -->

The idea was proposed back in 2017 [here](https://github.com/kubernetes/dashboard/issues/1832). This mechanism will also be helpful in better utilisation of [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions) through the dashboard.

A custom UI requirement came up for [ingress-nginx](https://github.com/kubernetes/ingress-nginx/issues/109) which later required [enhancement](https://github.com/kubernetes/ingress-nginx/issues/2480) and dashboard should be able to support it without requiring constant changes to `kubernetes/dashboard`.

In general, users should be able to add functionality without modifying the dashboard source code.

### Goals

<!-- List the specific goals of the KEP.
How will we know that this has succeeded? -->

- A 3rd party developer should be able to install a custom plugin to the dashboard extending its functionality in some way.

<!-- ### Non-Goals -->

<!-- What is out of scope for his KEP?
Listing non-goals helps to focus discussion and make progress. -->

## Proposal

<!-- This is where we get down to the nitty gritty of what the proposal actually is. -->

The plugin mechanism will comprise of largely of backend resources and frontend abstration that will help 3rd party developers to write plugins independently and register them with the dashboard.  

Following are the major areas of focus:  
#### How can a plugin be registered?  
- Plugin Discovery  
  - A new kind called `Plugin` should be introduced which MUST have a property specifying the location of the plugin bundle.
  - It will be the responsibility of a custom controller to fetch the plugin bundle and make it accessible for the dashboard.
    - The dashboard can query the `Plugin` resource for list of installed plugins.
- Plugin Management
  - The plugins can be added, updated or removed just like any other kubernetes resource.
  - It will be the responsibility of the custom controller to maintain the desired state of the plugins.
- Plugin Versioning
  - The plugin bundle location property can be used to specify the version of the plugin to be used.

#### How can a plugin connect to K8s API server?
- A `K8sAPIClient` interface will be exposed via the Angular Plugin Module for the custom plugin to use and make calls to the K8s API server.

#### How can a plugin connect to external APIs?
- An `HttpClient` interface will be exposed via the Angular Plugin Module for the custom plugin to use and make network calls.
- The requests made through this client will be proxied via a backend server so that CORS issue can be avoided and request/response transformation can be done if required.

#### How can a plugin connect to a Go binary?
- Inside the `Plugin` kind it can be defined whether the plugin requires its own backend server. If yes, then deployment of the backend from an image can be specified along with the necessary configurations.

#### How will a 3rd party developer write the plugin?
A developer will be able to write the plugin just like any other Angular application. The source code must be compiled and bundled into a file.

#### How will the Angular app mount these plugins?
The Dashboard will use `SystemJS` to pick the plugins up and mount them at runtime using the `NgModuleFactory`.


<!-- ### User Stories [optional] -->

<!-- Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of the system.
The goal here is to make this feel real for users without getting bogged down. -->

<!-- #### Story 1 [Update required] -->

<!-- ### Implementation Details/Notes/Constraints [optional] -->

<!-- What are the caveats to the implementation?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they releate. -->

### Risks and Mitigations

<!-- What are the risks of this proposal and how do we mitigate.
Think broadly.
For example, consider both security and how this will impact the larger kubernetes ecosystem. -->

- Authentication and permission management is an issue to tackle if interaction with k8s API is allowed.

## Graduation Criteria

<!-- How will we know that this has succeeded?
Gathering user feedback is crucial for building high quality experiences and SIGs have the important responsibility of setting milestones for stability and completeness.
Hopefully the content previously contained in [umbrella issues][] will be tracked in the `Graduation Criteria` section.

[umbrella issues]: https://github.com/kubernetes/kubernetes/issues/42752 -->

In future, there should be an umbrella issue for this KEP, keeping in mind all the requirements listed here, which should be accepted for this KEP to graduate.

## Implementation History

<!-- Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded -->
  
A discussion was started [here](https://github.com/kubernetes/dashboard/issues/1832). However, no concrete implementations have been made for this feature yet.

<!-- ## Drawbacks [optional] -->

<!-- Why should this KEP _not_ be implemented. -->

<!-- ## Alternatives [optional] -->

<!-- Similar to the `Drawbacks` section the `Alternatives` section is used to highlight and record other possible approaches to delivering the value proposed by a KEP. -->

<!-- ## Infrastructure Needed [optional] -->

<!-- Use this section if you need things from the project/SIG.
Examples include a new subproject, repos requested, github details.
Listing these here allows a SIG to get the process for these resources started right away. -->