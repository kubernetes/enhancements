# Sending Events in Distributed Manner

## Table of Contents



<!--toc -->

- [Summary](#summary)

- [Motivation](#motivation)

- [Proposal](#proposal)

  ​	-[Implementation Details](#implementation_details)

<!--toc -->

## Summary

A proposal to add different event client to kubelet apart form api server.

## Motivation

Currently users of kubernetes have to take events from api-server.This approach is centralized. We can provide a option to users to have there event receiver directly from kubelet so that they can react to the events happening in kubelet for example you can have organization specific setup on node which react based on pod deletion and pod creation. This is more of optimization of current process.

## Proposal

The proposal is that we add a flag in kubelet that can take string of IP and we can make rest client of that IP and add that client to the event Broadcaster.

### Implementation Details

Here Kubelet is client and whichever service that will receive events will be server. 

So for client side(Kubelet) we can add EventClient flag and take string of IP's. Initiate there rest config.And add them to the event Broadcaster in ``` makeEventRecorder``` function along with api server.

For server side kubelet expects same event returned with status OK. So following can be basic pattern for Event reciever server.

```go
func homePage(w http.ResponseWriter, r *http.Request){
	bodyBytes, err := ioutil.ReadAll(r.Body)
   	if err != nil {
        fmt.Printf(string(err))
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(200)
    w.Write(bodyBytes)
}  

func handleRequests() {
    http.HandleFunc("/", homePage)
    log.Fatal(http.ListenAndServe(":10000", nil))
}

```



