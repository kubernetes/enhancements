# Kubernetes DNS-Based **Multicluster** Service Discovery

> **Everything is TBD** -- moving from http://bit.ly/k8s-multicluster-dns

## 0 - About This Document

## 1 - Schema Version

## 2 - Resource Records

### 2.1 - Definitions

### 2.2 - Record for Schema Version

### 2.3 - Records for a Service with ClusterSetIP

#### 2.3.1 - `A`/`AAAA` Record

#### 2.3.2 - `SRV` Records

#### 2.3.3 - `PTR` Record

#### 2.3.4 - Records that should NOT exist for a Service with ClusterSetIP

### 2.4 - Records for a Multicluster Headless Service

#### 2.4.1 - `A`/`AAAA` Records

#### 2.4.2 - `SRV` Records

#### 2.4.3 - `PTR` Records

#### 2.4.4 - Records that should NOT exist for a Multicluster Headless Service