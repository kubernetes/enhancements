# Kubernetes DNS-Based **Multicluster** Service Discovery


## 0 - About This Document

This document is a specification for DNS-based Kubernetes service discovery for clusters implementing the [Multicluster Service API](README.md).

## 1 - Schema Version

`<<[UNRESOLVED need to pick schema version]>>`
This document describes version X.X.X of the schema.
`<<[/UNRESOLVED]>>`

## 2 - Resource Records

Any DNS-based service discovery solution for Kubernetes clusters implementing the Multicluster Services API must provide the resource records (RR) described below to be considered compliant with this specification.

### 2.1 - Definitions

This proposal is intended as an extension of the existing Kubernetes DNS specification, and inherits its definitions from section 2.1 with the addition of the following:

clusterset = as defined in [KEP-1645: Multi-Cluster Services API](README.md): “A placeholder name for a group of clusters with a high degree of mutual trust and shared ownership that share services amongst themselves. Membership in a clusterset is symmetric and transitive. The set of member clusters are mutually aware, and agree about their collective association. Within a clusterset, [namespace sameness](https://github.com/kubernetes/community/blob/master/sig-multicluster/namespace-sameness-position-statement.md) applies and all namespaces with a given name are considered to be the same namespace.”

`<clustersetzone>` = domain for multi-cluster services in the clusterset, which must be `clusterset.local`; as this may become configurable in the future, this specification refers to it by the palceholder `<clustersetzone>`, but per the MCS API it currently must be defined to be `clusterset.local`. 

ClusterSetIP / `<clusterset-ip>` = as defined in [KEP-1645: Multi-Cluster Services API](README.md): “A non-headless ServiceImport is expected to have an associated IP address, the clusterset IP, which may be accessed from within an importing cluster. This IP may be a single IP used clusterset-wide or assigned on a per-cluster basis, but is expected to be consistent for the life of a ServiceImport from the perspective of the importing cluster. Requests to this IP from within a cluster will route to backends for the aggregated Service.”

Cluster ID / `<clusterid>` = the cluster id stored in the `id.k8s.io ClusterProperty` as described in [KEP-2149: ClusterId for ClusterSet identification]. Though this can be any valid DNS label, in this KEP the examples mimic the recommended value, a kube-system namespace uid (`721ab723-13bc-11e5-aec2-42010af0021e`).


### 2.2 - Record for Schema Version

Following the existing specification, clusters implementing multi cluster DNS will contain an additional `TXT` record responding with the semantic version of the DNS schema used for the multi cluster DNS `<zone>`, also known in this specification as `<clustersetzone>`.

- Question Example:
  - `dns-version.clusterset.local. IN TXT`
- Answer Example:
  - `dns-version.clusterset.local. 28800 IN TXT “1.1.0”`

### 2.3 - Records for a Service with ClusterSetIP

#### 2.3.1 - `A`/`AAAA` Record

Given a ClusterIP type Service named `<service>` in Namespace `<ns>` that has been exported via a name-mapped ServiceExport with name `<service>`, given it is accessible across the cluster set by the IP address `<clusterset-ip>`, the following records must exist.

If the `<clusterset-ip>` is an IPv4 address, an `A` record of the following form must exist.



*   Record Format:
    *   `<service>.<ns>.svc.<clustersetzone>. <ttl> IN A <clusterset-ip>`
*   Question Example
    *   `myservice.test.svc.clusterset.local. IN A`
*   Answer Example:
    *   `myservice.test.svc.clusterset.local. 4 IN A 10.42.42.42`

If the `<clusterset-ip>` is an IPv6 address, an `AAAA` record of the following form must exist.



*   Record Format:
    *   `<service>.<ns>.svc.<clustersetzone>. <ttl> IN AAAA <clusterset-ip>`
*   Question Example:
    *   `myservice.test.svc.clusterset.local. IN AAAA`
*   Answer Example:
    *   `myservice.test.svc.clusterset.local. 4 IN AAAA 2001:db8::1`


#### 2.3.2 - `SRV` Records

For each port in an exported Service with name `<port>` and number `<port-number>` using protocol `<proto>`, an `SRV` record of the following form must exist.



*   Record Format:
    *   `_<port>._<proto>.<service>.<ns>.svc.<clustersetzone>. <ttl> IN SRV <weight> <priority> <port-number> <service>.<ns>.svc.<zone>.`

The priority `<priority>` and weight `<weight>` are numbers as described in [RFC2782](https://tools.ietf.org/html/rfc2782) and whose values are not prescribed by this specification.

Unnamed ports do not have an `SRV` record.



*   Question Example:
    *   `_https._tcp.myservice.test.svc.clusterset.local. IN SRV`
*   Answer Example:
    *   `_https._tcp.myservice.test.svc.clusterset.local. 30 IN SRV 10 100 443 myservice.test.svc.clusterset.local.`

The Additional section of the response may include the Service `A`/`AAAA` record referred to in the `SRV` record.

#### 2.3.3 - `PTR` Record

Given an exported Service assigned the IPv4 ClusterSet IP `<a>.<b>.<c>.<d>` **that does not already have a `PTR` record (see Limitations, below)**, a `PTR` record of the following form must exist.


*   Record Format:
    *   `<d>.<c>.<b>.<a>.in-addr.arpa. <ttl> IN PTR <service>.<ns>.svc.<clustersetzone>.`
*   Question Example:
    *   `1.0.3.10.in-addr.arpa. IN PTR`
*   Answer Example:
    *   `1.0.3.10.in-addr.arpa. 14 IN PTR kubernetes.test.svc.clusterset.local.`

Given an exported Service assigned the IPv6 ClusterSet IP represented in hexadecimal format without any simplification `<a1a2a3a4:b1b2b3b4:c1c2c3c4:d1d2d3d4:e1e2e3e4:f1f2f3f4:g1g2g3g4:h1h2h3h4>` **that does not already have a `PTR` record (see Limitations, below)**, a `PTR` record as a sequence of nibbles in reverse order of the following form must exist.



*   Record Format:
    *   `h4.h3.h2.h1.g4.g3.g2.g1.f4.f3.f2.f1.e4.e3.e2.e1.d4.d3.d2.d1.c4.c3.c2.c1.b4.b3.b2.b1.a4.a3.a2.a1.ip6.arpa <ttl> IN PTR <service>.<ns>.svc.<clustersetzone>.`
*   Question Example:
    *   `1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa. IN PTR`
*   Answer Example:
    *   `1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa. 14 IN PTR kubernetes.test.svc.clusterset.local.`

##### Limitations

By definition, only one `PTR` record may exist per IP address. For implementations of Multicluster DNS that use IPs that already have a `PTR` record assigned from the cluster-local DNS specification, no further `PTR` records are required. 

In particular, implementations that create a new "dummy" cluster-local `Service` object for every `ServiceImport` will already have a `PTR` record generated due to the DNS resolution of the "dummy" `Service`.

#### 2.3.4 - Records that should NOT exist for a Service with ClusterSetIP

ClusterSetIP Services should **NOT** have a record disambiguating to a single cluster's backends, ex. `<clusterid>.<svc>.<ns>.svc.<clustersetzone>`. 

(See the DNS section of the [KEP-1645: Multi-Cluster Services API](README.md#not-allowing-cluster-specific-targeting-via-dns) for more context.)

### 2.4 - Records for a Multicluster Headless Service

#### 2.4.1 - `A`/`AAAA` Records

Given a headless Service named `<service>` in Namespace `<ns>` that has been exported via a name-mapped ServiceExport with name `<service>`, for each _ready_ endpoint accessible across the cluster set with the IPv4 address `<endpoint-ip>` the following records must exist.



*   Record Format:
    *   `<service>.<ns>.svc.<clustersetzone>. IN A <endpoint-ip>`
*   Question Example
    *   `myservice.test.svc.clusterset.local IN A`
*   Answer Example:
    *   `myservice.test.svc.clusterset.local 4 IN A 10.42.42.42`
    *   `myservice.test.svc.clusterset.local 4 IN A 10.10.10.10`

There must also be an A record of the following form for each ready endpoint with hostname of `<hostname>`, member cluster ID of `<clusterid>`, and IPv4 address `<endpoint-ip>`. If there are multiple IPv4 addresses for a given hostname, then there must be one such `A` record returned for each IP.



*   Record Format:
    *   `<hostname>.<clusterid>.<service>.<ns>.svc.<clustersetzone>. <ttl> IN A <endpoint-ip>`
*   Question Example:
    *   `my-hostname.721ab723-13bc-11e5-aec2-42010af0021e.myservice.test.svc.clusterset.local. IN A`
*   Answer Example:
    *   `my-hostname.721ab723-13bc-11e5-aec2-42010af0021e.myservice.test.svc.clusterset.local. 4 IN A 10.3.0.100`

There must be an `AAAA` record for each _ready_ endpoint of the headless Service with IPv6 address `<endpoint-ip>` as shown below. If there are no _ready_ endpoints for the headless Service, the answer should be `NXDOMAIN`.



*   Record Format:
    *    `<service>.<ns>.svc.<clustersetzone>. <ttl> IN AAAA <endpoint-ip>`
*    Question Example:
    *    `headless.test.svc.clusterset.local. IN AAAA`
*    Answer Example:
    *    `headless.test.svc.clusterset.local. 4 IN AAAA 2001:db8::1`
    *    `headless.test.svc.clusterset.local. 4 IN AAAA 2001:db8::2`
    *    `headless.test.svc.clusterset.local. 4 IN AAAA 2001:db8::3`

There must also be an AAAA record of the following form for each ready endpoint with hostname of `<hostname>`, member cluster ID of `<clusterid>`, and IPv6 address `<endpoint-ip>`. If there are multiple IPv6 addresses for a given hostname, then there must be one such `A` record returned for each IP.



*   Record Format:
    *   `<hostname>.<clusterid>.<service>.<ns>.svc.<clustersetzone>. <ttl> IN AAAA <endpoint-ip>`
*   Question Example:
    *   `my-hostname.721ab723-13bc-11e5-aec2-42010af0021e.test.svc.clusterset.local. IN AAAA`
*   Answer Example:
    *   `my-hostname.721ab723-13bc-11e5-aec2-42010af0021e.test.svc.clusterset.local. 4 IN AAAA 2001:db8::1`

#### 2.4.2 - `SRV` Records

For each combination of _ready_ endpoint with _hostname_ of `<hostname>`, member cluster ID of `<clusterid>`, and port in the Service with name `<port>` and number `<port-number>` using protocol `<proto>`, an `SRV` record of the following form must exist.



*    Record Format:
    *    `_<port>._<proto>.<service>.<ns>.svc.<clustersetzone>. <ttl> IN SRV <weight> <priority> <port-number> <hostname>.<clusterid>.<service>.<ns>.svc.<clustersetzone>.`

This implies that if there are **N** _ready_ endpoints and the Service defines **M** named ports and it is exported in **P** clusters, there will be **N** X **M **X **P `SRV`** RRs for the Service.

The priority `<priority>` and weight `<weight>` are numbers as described in [RFC2782](https://tools.ietf.org/html/rfc2782) and whose values are not prescribed by this specification.

Unnamed ports do not have an `SRV` record.



*    Question Example:
    *    `_https._tcp.headless.test.svc.clusterset.local. IN SRV`
*   Answer Example:
    *   `_https._tcp.headless.test.svc.clusterset.local. 4 IN SRV 10 100 443 my-pet.721ab723-13bc-11e5-aec2-42010af0021e.headless.test.svc.clusterset.local.`
    *   `_https._tcp.headless.test.svc.clusterset.local. 4 IN SRV 10 100 443 my-pet-2.721ab723-13bc-11e5-aec2-42010af0021e.headless.test.svc.clusterset.local.`
    *   `_https._tcp.headless.test.svc.clusterset.local. 4 IN SRV 10 100 443 721ab723-13bc-11e5-aec2-42010af0021e.headless.test.svc.clusterset.local.`

The Additional section of the response may include the `A`/`AAAA` records referred to in the `SRV` records.

#### 2.4.3 - `PTR` Records

Given a _ready_ endpoint with _hostname_ of `<hostname>`, member cluster ID of `<clusterid>`, and IPv4 address `<a>.<b>.<c>.<d>` **that does not already have a `PTR` record (see Limitations, below)**, a `PTR` record of the following form must exist.



*    Record Format:
    *    `<d>.<c>.<b>.<a>.in-addr.arpa. <ttl> IN PTR <hostname>.<clusterid>.<service>.<ns>.svc.<clustersetzone>.`
*    Question Example:
    *    `100.0.3.10.in-addr.arpa. IN PTR`
*    Answer Example:
    *    `100.0.3.10.in-addr.arpa. 14 IN PTR my-pet.headless.721ab723-13bc-11e5-aec2-42010af0021e.test.svc.clusterset.local.`

Given a _ready_ endpoint with _hostname_ of `<hostname>` and IPv6 address in hexadecimal format without any simplification `<a1a2a3a4:b1b2b3b4:c1c2c3c4:d1d2d3d4:e1e2e3e4:f1f2f3f4:g1g2g3g4:h1h2h3h4>` **that does not already have a `PTR` record (see Limitations, below)**, a `PTR` record as a sequence of nibbles in reverse order of the following form must exist.



*    Record Format:
    *    `h4.h3.h2.h1.g4.g3.g2.g1.f4.f3.f2.f1.e4.e3.e2.e1.d4.d3.d2.d1.c4.c3.c2.c1.b4.b3.b2.b1.a4.a3.a2.a1.ip6.arpa <ttl> IN PTR <hostname>.<clusterid>.<service>.<ns>.svc.<clustersetzone>.`
*    Question Example:
    *    `1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa. IN PTR`
*    Answer Example:
    *    `1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa. 14 IN PTR my-pet.721ab723-13bc-11e5-aec2-42010af0021e.headless.test.svc.clusterset.local.`

##### Limitations

By definition, only one `PTR` record may exist per IP address. For implementations of Multicluster DNS that use IPs that already have a `PTR` record assigned from the cluster-local DNS specification, no further `PTR` records are required. 

In particular, implementations that create a new "dummy" cluster-local `Service` object for every `ServiceImport` will already have a `PTR` record generated due to the DNS resolution of the "dummy" `Service`.

#### 2.4.4 - Records that should NOT exist for a Multicluster Headless Service

Multicluster Headless Services should **NOT** have a record disambiguating to a single cluster's backends, ex. `<clusterid>.<svc>.<ns>.svc.<clustersetzone>`.

(See the DNS section of the [KEP-1645: Multi-Cluster Services API](README.md#not-allowing-cluster-specific-targeting-via-dns) for more context.)