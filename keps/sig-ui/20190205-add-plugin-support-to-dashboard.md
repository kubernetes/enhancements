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

- A user should be able to install a custom plugin to the dashboard extending its functionality in some way.
- We need to provide an interface to the plugin so that it can interact with core dashboard services.

<!-- ### Non-Goals -->

<!-- What is out of scope for his KEP?
Listing non-goals helps to focus discussion and make progress. -->

## Proposal

<!-- This is where we get down to the nitty gritty of what the proposal actually is. -->

Proposed approach:
- Define an interface for the plugin to register itself with the dashboard which the plugin should implement.
  - Something like a `PluginService` instance can expose dashboard utilities to the plugin.
- Generate and distribute the plugin bundle.
- Mount the generated plugin bundle on demand in a dashboard view.

How can a plugin be registered?
  - The dashboard could periodically scan `plugins` directory mounted into the pod for `YAMLs`, if there is a new plugin, register it with the dashboard.

How can a plugin connect to external APIs?
- `HttpClient` can be exposed via the `PluginService` instance for the plugin to use and make network calls.
- CORS requests should be properly handled here.

How can a plugin connect to a Go binary?
- [Needs more thought]

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