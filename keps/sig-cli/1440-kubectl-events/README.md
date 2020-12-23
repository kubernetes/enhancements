# kubectl events

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Limitations of the Existing Design](#limitations-of-the-existing-design)
  - [Goals](#goals)
  - [Non-goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
<!-- /toc -->

## Summary

Presently, `kubectl get events` has some limitations. It cannot be extended to meet the increasing user needs to
support more functionality without impacting the `kubectl get`. This KEP proposes a new command `kubectl events` which will help
address the existing issues and enhance the `events` functionality to accommodate more features.

For eg: Any modification to `--watch` functionality for `events` will also change the `--watch` for `kubectl get` since the `events` is dependent of `kubectl get`

Some of the requested features for events include:

1. Extended behavior for `--watch`
2. Default sorting of `events`
3. Union of fields in custom-columns option
4. Listing the events timeline for last N minutes
5. Sorting the events using the other criteria as well

This new `kubectl events` command will be independent of `kubectl get`. This can be
extended to address the user requirements that cannot be achieved if the command is dependent of `get`.

## Motivation

A separate sub-command for `events` under `kubectl` which can help with long standing issues:
Some of these issues that be addressed with the above change are:

- User would like to filter events types
- User would like to see all change to the an object
- User would like to watch an object until its deletion
- User would like to change sorting order
- User would like to see a timeline/stream of `events`

### Limitations of the Existing Design

All of the issues listed below require extending the functionality of `kubectl get events`.
This would result in `kubectl get` command having a different set of functionality based
on the resource it is working with. To avoid per-resource functionality, it's best to
introduce a new command which will be similar to `kubectl get` in functionality, but
additionally will provide all of the extra functionality.

Following is a list of long standing issues for `events`

- kubectl get events doesn't sort events by last seen time [kubernetes/kubernetes#29838](https://github.com/kubernetes/kubernetes/issues/29838)
- Improve watch behavior for events [kubernetes/kubernetes#65646](https://github.com/kubernetes/kubernetes/issues/65646), [kubernetes/kubectl#793](https://github.com/kubernetes/kubectl/issues/793),
- Improve events printing [kubernetes/kubectl#704](https://github.com/kubernetes/kubectl/issues/704), [kubernetes/kubectl#151](https://github.com/kubernetes/kubectl/issues/151)
- Events query is too specific in describe [kubernetes/kubectl#147](https://github.com/kubernetes/kubectl/issues/147)
- kubectl get events should give a timeline of events [kubernetes/kubernetes#36304](https://github.com/kubernetes/kubernetes/issues/36304)
- kubectl get events should provide a way to combine ( Union) of columns [kubernetes/kubernetes#82950](https://github.com/kubernetes/kubernetes/issues/82950)

### Goals

- Add an experimental `events` sub-command under the kubectl
- Address existing issues mentioned above

### Non-goals

- This new command will not be dependent on `kubectl get`

## Proposal

Have an independent *events* sub-command which can perform all the existing tasks that the current `kubectl get events`,
and most importantly will extend the `kubectl get events` functionality to address the existing issues.

### Implementation Details/Notes/Constraints

The above use-cases call for the addition of several flags, that would act as filtering mechanisms for events,
and would work in tandem with the existing --watch flag:
- Add a new `--watch-event=[]` flag that allows users to subscribe to particular events, filtering out any other event kind
- Add a new `--watch-until=EventType` flag that would cause the `--watch` flag to behave as normal, but would exit the command as soon as the specified event type is received.
- Add a new `--watch-for=pod/bar flag` that would filter events to only display those pertaining to the specified resource. A non-existent resource would cause an error. This flag could further be used with the `--watch-until=EventType` flag to watch events for the resource specified, and then exit as soon as the specified `EventType` is seen for that particular resource.
- Add a new `--watch-until-exists=pod/bar` flag that outputs events as usual, but exits as soon as the specified resource exists. This flag would employ the functionality introduced in the wait command.

Additionally, the new command should support all the printing flags available in `kubectl get`, such as specifying output format, sorting as well as re-use server-side printing mechanism.

## Design Details

### Test Plan

In addition to standard unit tests for kubectl, the events command will be released as a kubectl alpha subcommand, signaling users to expect instability. During the alpha phase we will gather feedback from users that we expect will improve the design of debug and identify the Critical User Journeys we should test prior to Alpha -> Beta graduation.

### Graduation Criteria

Once the experimental kubectl events command is implemented, this can be rolled out in multiple phases.

##### Alpha -> Beta Graduation
- [ ] Gather the feedback, which will help improve the command
- [ ] Extend with the new features based on feedback

##### Beta -> GA Graduation
- [ ] Address all major issues and bugs raised by community members
