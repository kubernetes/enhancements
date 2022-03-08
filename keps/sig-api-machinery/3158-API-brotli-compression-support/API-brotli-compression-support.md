# Apiserver support brotli compression

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [1.25](#125)
  - [1.26](#126)
  - [Implementation Details](#implementation-details)
    - [How to enable brotli compression](#how-to-enable-brotli-compression)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

Kubernetes sometimes returns extremely large responses to clients outside of its local network, resulting in long delays for components that integrate with the cluster in the list/watch controller pattern. Just as
https://github.com/kubernetes/enhancements/blob/787e5513d09096756c61aaba1916a73cb1dd348b/keps/sig-api-machinery/2338-graduate-API-gzip-compression-support-to-GA/README.md described.

Brotli has better perfermance than gzip. We should support brotli compression.

## Motivation
Brotli has better compression ratio and compression time than gzip. Below is my test result:

```shell
// This is my machine
Architecture:        x86_64
CPU(s):              8
Model name:          Intel(R) Xeon(R) CPU E5-2680 v4 @ 2.40GHz
CPU MHz:             2394.454
```
Compress 100K kubernetes pods:

| AcceptEncoding | ContentType | Compression Ratio | Compression Time |
|---|-------------|-------------------|-----------------|
| gzip    | json        | 35.346268         | 1.308877099s    |
| brotli    | json        | 42.693703         | 437.817496ms |
| gzip    | protobuf    | 25.013601         | 842.336524ms    |
| brotli    | protobuf    | 25.714157         | 433.808134ms |

kube-apiserver E2E test(ContentType=protobuf):
1. list 100k pods from apiserver

GZip compression:
```shell
I0308 02:35:54.815681       1 trace.go:205] Trace[1408477693]: "List" url:/api/v1/pods,user-agent:Go-http-client/2.0,audit-id:ea7305e9-0a42-4cab-b3b4-650d377215cf,client:172.18.0.1,accept:application/vnd.kubernetes.protobuf,protocol:HTTP/2.0 (08-Mar-2022 02:35:49.749) (total time: 5066ms):
Trace[1408477693]: ---"Writing http response done" count:100690 4950ms (02:35:54.815)
Trace[1408477693]: [5.066537468s] [5.066537468s] END
```
Brotli compression:
```shell
I0308 02:40:29.761867       1 trace.go:205] Trace[979250671]: "List" url:/api/v1/pods,user-agent:Go-http-client/2.0,audit-id:dea54561-f2db-4a01-90e9-daf85cdc5490,client:172.18.0.1,accept:application/vnd.kubernetes.protobuf,protocol:HTTP/2.0 (08-Mar-2022 02:40:27.309) (total time: 2452ms):
Trace[979250671]: ---"Writing http response done" count:100690 2363ms (02:40:29.761)
Trace[979250671]: [2.452228105s] [2.452228105s] END
```
### Goals

kubernetes client and apiserver support brotli compression. Users can optionally use brotli compression by adding http header `Accept-Encoding: br`.

### Non-Goals

* Remove gzip compression

## Proposal

### 1.25

* Add brotli alpha feature fate default disable

### 1.26

* Promote to beta feature default enable


### Implementation Details

#### How to enable brotli compression

1. add `EncodeConfig` to client-go config

```go
type EncodeConfig struct {
	// AcceptEncoding specifies the compression types the client support.
	// If not set, golang std lib gzip will be used
	AcceptEncoding string
}
```
Users can choose whether to use brotli by setting `EncodeConfig.AcceptEncoding = "br"`. Most browsers support brotli including chrome.

2. apiserver support brotli compression

apiserver get http header `Accept-Encoding: br` from request and reset response writer to brotli writer.

### Risks and Mitigations

## Graduation Criteria

## Implementation History