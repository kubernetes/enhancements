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
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

Currently, we the `kubectl get events` are not in sorted order and the events command is dependent on the get. 
If users are looking for some sorting order, a new flag needs to be passed to achieve that. However, there has been a demand to 
get the events sorted in a default order. 

This KEP proposes to add a new command `events` for `kubectl` and  be independent of `kubectl get`

## Motivation

The `kubectl get events` command provides a cli for:

- displaying the events across the resources in a random order. 
- These results can be sorted by providing an additional flag `--sort-by`

Examples:

- Present kubectl get events  command examples
  - `kubectl get events` : The following events are in a random order
     ```
        LAST SEEN   TYPE      REASON    OBJECT        MESSAGE
        47m         Normal    Pulling   pod/bad-pod   Pulling image "knginx"
        2m52s       Normal    BackOff   pod/bad-pod   Back-off pulling image "knginx"
        17m         Warning   Failed    pod/bad-pod   Error: ImagePullBackOff 
     ```

  - `kubectl get events --sort-by .lastTimestamp` 
    ```
       LAST SEEN   TYPE      REASON    OBJECT        MESSAGE
       57m         Normal    Pulling   pod/bad-pod   Pulling image "knginx"
       7m31s       Warning   Failed    pod/bad-pod   Error: ImagePullBackOff
       2m32s       Normal    BackOff   pod/bad-pod   Back-off pulling image "knginx" 
    ```

Alghough, we can sort the order by passing the `sort-by` flag, there has been a continuous demand 
to have a default sort order of the events rather than the random order. 


- Proposed Kubectl events command
  - `kubectl events`: Default sorted by .lastTimestamp
     ```
       LAST SEEN   TYPE      REASON    OBJECT        MESSAGE
       57m         Normal    Pulling   pod/bad-pod   Pulling image "knginx"
       7m31s       Warning   Failed    pod/bad-pod   Error: ImagePullBackOff
       2m32s       Normal    BackOff   pod/bad-pod   Back-off pulling image "knginx" 
      ```
  - Following are the proposed options for the command: 
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

  ```
  

#### Long standing issues

Provide a default sort order for the `events` rather than the random order: 

- kubectl get events doesnt sort events by last seen time [kubernetes/kubernetes#29838](https://github.com/kubernetes/kubernetes/issues/29838)

### Goals

- Add an experimental `events` sub-command under the kubectl and move the existing `events` command from `kubectl get` 
- This new command will not be dependent on `kubectl get`

## Proposal

Have a independent *events* command which can perform all the existing tasks that the current `kubectl get events` 
command is performing


### Implementation Details/Notes/Constraints [optional]

After the successful adoption of the command, the `events` command under the `kubectl get` can be removed. 

- Create the *kubectl events* sub command (Work in Progress)


## Graduation Criteria

- The `events` command has been added to `kubectl` 
- Publish `events` as a subcommand of kubectl.
- Remove the existing `kubectl` from under the `kubectl get` 


## Alternatives

1. Modify the existing code and provide a default sort order for the events. [PR](https://github.com/kubernetes/kubernetes/pull/82898)

