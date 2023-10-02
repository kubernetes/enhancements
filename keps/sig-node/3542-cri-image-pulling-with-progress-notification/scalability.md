Worker node:
----------------------
10 Gib/s || 1.2 GiB/s downlink
110 Pods max

Cluster size:
----------------------
5000 worker nodes, max 150 000 Pods

Two kinds of deployment:
----------------------
Many small-sized Pods : 100    MiB image
Fewer huge-sized Pods : 10 000 MiB image

Scheduler:
----------------------
Emits 500 Pods to nodes per second


1. Small images

1.1. Even distribution - one Pod per node

1 / 10th of Worker nodes will start downloading single image with 1200 MiB/s speed, it will take
one or few seconds to download 100 or 1000 MiB big-image to worker node, scheduler won't be fast
enough to fill rest of 4500 Nodes to schedule second Pod to same node.

Time spent in downloading is 1 / 60th of proposed 1-minute interval. No Pod events published.

1.2. Packing distribution - as many Pods to same node as possible

500 / 110 = 5 nodes. 4 nodes completely filled with 110 Pods plus one Node with 60 Pods

Let's say 3 x containers per Pod, 330 different images:
Data to pull: 330 x 0.1 GiB = 33GiB
Time to pull: 33 / 1.2 ~= 30-40 seconds

Regardless if parallel or sequential pulling, time spent in downloading is half of proposed
1-minute interval. No Pod events published.

2. Huge images

Presumption is that Huge images require huge resources, for instance PCI-e acceleraters,
and or plenty of RAM, CPU power. Let's assume there can be 10 such Pods in one Node.

2.1. Even distribution - one Pod per node

1 / 10th of worker nodes will start downloading single image with 1200 MiB /s
it will take 10 second to download 10 GiB big-image to worker node.

500 Nodes out of 5000 were tasked with one Pod within first second of scheduling,
remaining 4500 nodes will be tasked with single Pod workload within next 9 seconds.
After 10th second the first Node will get new Pod to download image for.

Even given an few seconds overlap in downloading (when new image download starts, while
previously scheduled Pod's image is finalizing download), every next Pod's image is downloaded
few seconds (let's say 2s) longer than the previous one, worst case is around 20s, 1/3rd of
proposed default notification interval. No events emitted.

Seconds spent downloading
```
Pod 1  01234567890
Pod 2            0123456789012
Pod 3                      012345678901234
Pod 4                                0123456789012345
Pod 5                                          01234567890123456
Pod 6                                                    01234567890123456789
Pod 7                                                              012345678901234567890
Pod 8                                                                        0123456789012345678901
Pod 9                                                                                  012345678901234567890123
Pod 10                                                                                           012345678901234567
```

2.2. Packing distribution - as many Pods to same node as possible

Scheduler tasks Nodes with 500 Pods per second rate, 10 Pods per Node = 50 Nodes fully tasked
every second. It will take 100 seconds until 5k-nodes-big cluster can no longer accept / has no
more resources for such Pods.

100 GiB data to download on each Node, 1.2 GiB per second = 83 seconds, longer than proposed
default reporting interval.

Parallel pulling: there will be one event for each out of 10 Pods = 10 Events / 10 API calls,
download will finish until the second interval is reached.

Sequential pulling = first 72 GiB downloaded within 60 seconds with no notifications sent out,
last 3 Pods still downloading images will get one pull progress notification event.

Workload lifespan is expected to be longer than 83 seconds, so next Pod will start downloading
image when some workloads come to an end.

Overall, there will be either 
a) bursts of 10 events every second from next 50 Nodes, 500 API calls total every second coming
to apiserver for 100 seconds (5000 nodes / 50 nodes)

b) 3 events from each out of 50 nodes every second, approx. 150 events / API calls to apiserver
for 100 seconds
