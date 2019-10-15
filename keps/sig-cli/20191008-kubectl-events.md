---
title: Kubectl events
authors:
  - "@hpandeycodeit"
owning-sig: sig-cli
participating-sigs:
  - sig-cli
reviewers:
  - "@soltysh"
  - "@pwittrock"
approvers:
  - "@soltysh"
  - "@pwittrock"
editor: TBD
creation-date: 2019-10-08
last-updated: 2019-10-08
status: provisional
see-also:
  - n/a
replaces:
  - 
superseded-by:
  -
---

# kubectl events

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
    - [Long standing issues](#long-standing-issues)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
- [Graduation Criteria](#graduation-criteria)
<!-- /toc -->

## Summary

Presently, `kubectl get events` has some limitations. It cannot be extended to met the increasing user needs to 
support more functionality without impacting the `kubectl get`. 

For eg: Any modification to `--watch` functionality for `events` will also change the `--watch` for `kubectl get` since the `events` is dependent of `kubectl get`

Some of the requested features for events include: 

1. Extended behaviour for `--watch` 
2. Default sorting of `events`

This KEP proposes to add a new command `events` for `kubectl` and  be independent of `kubectl get`. This can be 
extended to address the user requirements that cannot be achieved if the command is dependent of `get`.

## Motivation

A separate sub-command for `events` under `kubectl` which can help with long standing issues: 
Some of these issues that be addressed with the above change are: 

- User would like to see a stream of "create" or update events, filtering out deletes or any other event type. 
- User would like to know when an object is deleted while `--watching` it. 
- User would like to see all changes to a single object until it is deleted. 
- User would like to watch an object until it exists. 
- User would like to see the results of `events` in default sorting order. 
- User would like to see a timeline of `events`

Examples:

- Proposed Kubectl events command
  - `kubectl events`: Default sorted by .lastTimestamp
     ```
       LAST SEEN   TYPE      REASON    OBJECT        MESSAGE
       57m         Normal    Pulling   pod/bad-pod   Pulling image "knginx"
       7m31s       Warning   Failed    pod/bad-pod   Error: ImagePullBackOff
       2m32s       Normal    BackOff   pod/bad-pod   Back-off pulling image "knginx" 
      ```
  - Following are the proposed options for the command: 

  - `kuebctl events --watch-event=[]` flag that allows users to subscribe to particular events, 
     filtering out any other event kind: 
     ```
       kubectl events --watch-event=Delete
       or 
       kuebctl events  --watch-event=Warning 

     ```

Some of the options that will be included in the new command are: 

```
  Options:
  -A, --all-namespaces=false: If present, list the requested object(s) across all namespaces. Namespace in current
context is ignored even if specified with --namespace
  -L, --label-columns=[]: Accepts a comma separated list of labels that are going to be presented as columns. Names are
case-sensitive. You can also use multiple flag options like -L label1 -L label2...
      --no-headers=false: When using the default or custom-column output format, don't print headers (default print
headers).
  -o, --output='': Output format. One of:
json|yaml|wide|name|custom-columns=...|custom-columns-file=...|go-template=...|go-template-file=...|jsonpath=...|jsonpath-file=...
See custom columns [http://kubernetes.io/docs/user-guide/kubectl-overview/#custom-columns], golang template
[http://golang.org/pkg/text/template/#pkg-overview] and jsonpath template
[http://kubernetes.io/docs/user-guide/jsonpath].
      --output-watch-events=false: Output watch event objects when --watch or --watch-only is used. Existing objects are
output as initial ADDED events.
      --raw='': Raw URI to request from the server.  Uses the transport specified by the kubeconfig file.
  -l, --selector='': Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2)
      --server-print=true: If true, have the server return the appropriate table output. Supports extension APIs and CRDs.
      --show-kind=false: If present, list the resource type for the requested object(s).
      --show-labels=false: When printing, show all labels as the last column (default hide labels column)
      --sort-by='': If non-empty, sort list types using this field specification.  The field specification is expressed
as a JSONPath expression (e.g. '{.metadata.name}'). The field in the API resource specified by this JSONPath expression
must be an integer or a string.
  -w, --watch=false: After listing/getting the requested object, watch for changes. Uninitialized objects are excluded
if no object name is provided.
      --watch-only=false: Watch for changes to the requested object(s), without listing/getting first.
--watch-event: watch for "create" or update events, filtering out deletes or any other event type:
--watch-until: watch until the event exists
--watch-until-exists: watch until an onbject exists
--watch-for: watch for a particular object. 

  ```
  

#### Long standing issues

Following is a list of long standing issues for `events`

- kubectl get events doesnt sort events by last seen time [kubernetes/kubernetes#29838](https://github.com/kubernetes/kubernetes/issues/29838)
- Improve --watch behavior for events [kubernetes/kubernetes#65646](https://github.com/kubernetes/kubernetes/issues/65646)
- kubectl get events should give a timeline of events [kubernetes/kubernetes#36304](https://github.com/kubernetes/kubernetes/issues/36304)

### Goals

- Add an experimental `events` sub-command under the kubectl and move the existing `events` command from `kubectl get` 
- This new command will not be dependent on `kubectl get`
- Help addressing the existing issues which cannot be achieved with `kubectl get events`

## Proposal

Have a independent *events* command which can perform all the existing tasks that the current `kubectl get events` 
command is performing and also to extend the `kubectl get events` functionality to address the existing issues. 


### Implementation Details/Notes/Constraints [optional]

Once the kubectl events command is implemented, this can be rolled out in multiple phases: 
-  Release the new command `kubectl evetns` alongside with the existing command. 
-  After the successful adoption of the new command `kubectl events`, the old command under `kubectl get events` can be removed. 
- The new `kubectl events` can further be extended with the new features/long standing issues. 



## Graduation Criteria

An alpha version of this is new feature is targeted for 1.18 (subjected to change)

This can be promoted to beta when it is a drop-in replacement for the existing `kubectl get events`, and has no regressions (which aren't bug fixes). This KEP will be updated when we know the concrete things changing for beta.

This will be promoted to GA once it's gone a sufficient amount of time as beta with no changes. A KEP update will precede this.

