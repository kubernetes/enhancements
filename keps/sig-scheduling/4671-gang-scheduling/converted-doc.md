# [External] API Design for Gang and Workload-Aware Scheduling

Authors:   
    Eric Tune – GH:@erictune – etune@google.com, eric.tune@gmail.com  
    Heba El Ayoty – GH: @helayoty   
    Paweł Kępka - GH: @44past4  
    Wojtek Tyczynski - GH: @wojtek-t  
    Andrey Velichkevich - GH: @andreyvelich   
    Dominik Marciński - GH: @dom4ha  
    Tim Hockin - GH: @thockin  
Status: Proposal  
Created: 2025-08-21  
Short Link: [https://tiny.cc/hvhs001](https://tiny.cc/hvhs001)  
Shared with dev@kubernetes.io with commenter permissions.  
 

## Update History

-  Sep 2, 2025  – Shared doc on [KEP #4671](https://github.com/kubernetes/enhancements/issues/4671#issuecomment-3247971358).
-   Sep 5, 2025
    -  Added [Linking Pod to Workload](?tab=t.0#heading=h.wc4uxoik4hq0) section. Included alternative to use pod annotation during alpha in place of `Pod.spec.workload.name`.
    -  Added non-authoritative reference from `Workload` to true workload in [this section](?tab=t.0#heading=h.lov03d3i7sy8).
    -  Added [Design Alternatives for Workload Spec](?tab=t.0#heading=h.pc459p2v5tmz) which compares two options for the workload spec, one basic and one using tagged unions (the latter is the version from Sep 2).
    -  Added new alternative for go struct definition in [Go Struct Definitions – Basic Version](?tab=t.0#heading=h.9xos47u9pxd6)

-  Sep 8, 2025
    -  Added examples of true workloads and their corresponding `Workload` in section [Examples of Different Workload Specs](?tab=t.0#heading=h.20rp2loe7efo)

-  Sep 10, 2025
    -  Added "Documentation for Workloads" section
    -  Removed details of Tagged Union approach.

-  Sep 12, 2025
    -  Explored slight [variation](?tab=t.3pkx7y4zvho2) of go structs
    -  Added section on [Entity Naming Alternatives](?tab=t.0#heading=h.i33qo9ln88id) 
    -  Added [Pod Equivalence](?tab=t.0#heading=h.wbf8c7lm494k) section explaining when pods can share an `EqGroup`

-  Sep 17, 2025
    -  Added [Linking Pod to Workload Part](?tab=t.0#heading=h.a9mw63cfmssu)
    -  

## Summary

The goal of this document is to design an API that describes a group of pods to be "Gang Scheduled".  Since this API will be part of the core Kubernetes apis, special consideration is given to how it will evolve. The Gang Scheduling API seems likely to evolve into a Workload API for Workload-aware Scheduling. We show one path this evolution can take.

## Terms

-  _true workload type_ – Refers to any of the various workload controller kinds that today implement workloads.  Examples include: `batch/Job`, `apps/StatefulSet`, `kubeflow.org/MPIJob`,` apps/Deployment`,  and `leaderworkerset.x-k8s.io/LeaderWorkerSet`.  True workload controllers directly or transitively create pods .
-  _abstract workload type_ – Refers to a new type that can hold a subset of the fields of any _true workload type_, organized in a standard way.  Several of the design options for Workload-aware Scheduling use this. 

## Balancing Short and Long Term Needs

In the SIG-Scheduling community, we have both an immediate goal and a related longer-term goal:

-  **Immediate Goal** – Add gang-scheduling in kube-scheduler.  
    -  Minimal scope: Define a single group of pods that need to be scheduled and bound all-or-nothing. 
    -  Use: AI and HPC workloads with collective communication (MPI, PyTorch) which need all pods to start at once in order to form the collective.
    -  Gang scheduling avoids long delays between the first and last pod starting, which may cause the collective communication initialization to timeout.
    -  Gang scheduling also avoids a deadlock condition in single-scheduler setup when two large gang-scheduled are each partially started.
    -  When used with DRA, can find a free topology domain for a workload without need for node selectors. 
    -  Standardizes something that has been implemented [5+ times](?tab=t.0#heading=h.9qbw2b4lxrgr) in the community.
    -  When combined with suitable DRA drivers, it enables network topology-aware scheduling using only Kube-scheduler 
        -  Initial scope: required or preferred placement of all pods in one domain, where the domain is specified by a resourceClaim against a DRA multi-node logical device pool.

-  **Longer-Term Goal**  – Add workload-awareness in kube-scheduler.  
    -  Have a standardized way to provide kube-scheduler with information about application topology, its placement requirements and preferences, if replication is used, and how pods are indexed. This enables multi-level topology-aware placement, balanced placement of ranks, application fault tolerance, and other future workload-level scheduling policies.
    -  Support the [Workload-aware scheduling](https://docs.google.com/document/d/1pZaQ4q9nhENFFIy-WjUhb89WCgJ7ZPcrKx-PIU8rOqA/edit?tab=t.0#heading=h.krfc7hnhr8un) vision.

In this design doc, both goals are considered. If we instead attempted to tightly scope the design to the immediate goal, while ignoring topology-aware, etc, then we are likely to pay for it later.  We also miss the opportunity to learn from existing projects (Volcano, Kueue) that do both.

Therefore, this document includes two API designs:

1. **API-Rev1:**  An API that meets the immediate goal of basic gang scheduling.
    -  API Rev1 is to be approved now.

1. **API-RevN**: An API that meets the long-term goal of representing advanced topology-aware scheduling requirements for a wide range of workloads.
    -  API RevN not to be approved now.  By approving this document we are not committing to the design specified.  
    -  However, if any proposed changes to API-Rev1 should include corresponding changes to API RevN, to ensure an evolution path.

## High-level Design Alternatives

For both API-Rev1 and API-RevN, we need to represent grouping and policy information about a set of pods.  This is called "new information" in the table below.  The table shows 6 ways to provide this information.


| **Properties | **Embed  | **Embed  | **New    | **Translator | **Translator | **Translator |
: ↓        : in       : in True  : ResourceResource : Library  : API**    : Subresource** :
: Pattern  : PodSpec** : Workload** : Kind(s)** : **       :          :           :
: →**      :          :          :          :          :          :           :
| -------- | -------- | -------- | -------- | -------- | -------- | --------- |
| **How    |          |          |          |          |          |           |
: it       :          :          :          :          :          :           :
: would    :          :          :          :          :          :           :
: work.**  :          :          :          :          :          :           :
| Description | Add      | Add      | Create   | A set    | A        | Define a  |
:          : this     : this     : a new    : of go    : special  : schema    :
:          : information : information : resourceresource : libraries : readonlyreadonly : for a     :
:          : as       : as       : Kind to  : can      : API      : readonly  :
:          : multiplemultiple : multiplemultiple : hold     : translate : endpointendpoint : `/workload`, :
:          : new      : new      : this     : any      : allows   : similar   :
:          : fields   : fields   : information. : true     : listing  : to        :
:          : to the   : to each  : A        : workloadworkload : and      : `/scale`.`/scale`. :
:          : PodSpec  : true     : single   : into a   : watchingwatching : This      :
:          : type.    : workloadworkload : field    : standardstandard : a new    : subresource :
:          :          : type.    : in       : representation. : resourceresource : is on     :
:          :          :          : PodSpec  :  No new  : kind     : each      :
:          :          :          : links    : Kinds    : which    : true      :
:          :          :          : to the   : added    : summarizes : resource.resource. :
:          :          :          : new      : and no   : each     :  CRDs     :
:          :          :          : resource. : new      : true     : can       :
:          :          :          :          : fields   : workload. : provide   :
:          :          :          :          : are      : It       : e.g. CEL  :
:          :          :          :          : added    : would    : to        :
:          :          :          :          : to       : require  : transcodetranscode :
:          :          :          :          : PodSpec.PodSpec. : extensive : the true  :
:          :          :          :          :          : changes  : workload  :
:          :          :          :          :          : in       : spec to   :
:          :          :          :          :          : api-machinery. : this      :
:          :          :          :          :          :   It     : workload  :
:          :          :          :          :          : uses a   : subresource. :
:          :          :          :          :          : Translator :           :
:          :          :          :          :          : Library  : Built-in  :
:          :          :          :          :          : to       : types     :
:          :          :          :          :          : translate : have      :
:          :          :          :          :          : from     : golang-coded :
:          :          :          :          :          : any      : subresource :
:          :          :          :          :          : true     : support.  :
:          :          :          :          :          : workloadworkload :           :
:          :          :          :          :          : into a   :           :
:          :          :          :          :          : standardstandard :           :
:          :          :          :          :          : representation. :           :
:          :          :          :          :          :          :           :
| ExistingExisting | `TopologySpreadConstraints` | replica  | `ResourceClaim` | No       | `PodMetrics` | `/scale`  |
: ExamplesExamples : applies  : counts,  : hold     : known    : in       : subresource :
: of       : to a     : pod      : information : examples. : `metrics.k8s.io/v1beta1` : provides  :
: Pattern  : group    : template(s) : that     :          : providesprovides : access    :
:          : of       : are      : can      :          : a        : to the    :
:          : pods,    : typically : affect   :          : readonlyreadonly : "replicas" :
:          : but is   : in true  : Pod      :          : view     : field of  :
:          : copied   : workloads. : group    :          : built    : any       :
:          : into     :          : scheduling :          : from     : controller :
:          : every    :          : and is   :          : data     : which     :
:          : pod.     :          : referred-to :          : (mostly?) : implements :
:          :          :          : from     :          : already  : the       :
:          :          :          : the Pod  :          : present  : subresource. :
:          :          :          : resource. :          : in the   :  It       :
:          :          :          :          :          : `Pods`   : could     :
:          :          :          :          :          : api.     : provide   :
:          :          :          :          :          :          : much      :
:          :          :          :          :          :          : more      :
:          :          :          :          :          :          : information. :
:          :          :          :          :          :          :           :
:          :          :          :          :          :          : However,  :
:          :          :          :          :          :          : list/watch :
:          :          :          :          :          :          : of        :
:          :          :          :          :          :          : subresources :
:          :          :          :          :          :          : is not    :
:          :          :          :          :          :          : (well?)   :
:          :          :          :          :          :          : supported. :
:          :          :          :          :          :          :           :
| What     | ①        | ①        | ①        | ① The    | ① The    | ① The     |
: structure : Summary  : Summary  : NewResourceKind. : structure : kind     : kind      :
: would    : of the   : of the   :  ②       : which    : which    : which is  :
: kube-scheduler : pods,    : pods,    : Pointer  : is       : is       : produced  :
: use      : similar  : similar  : to list  : producedproduced : producedproduced : by the    :
: internally : to "New  : to "New  : of       : by the   : by the   : translator :
: to       : ResourceResource : ResourceResource : `*PodInfo`. : translator : translator : subresource. :
: kueue    : Kind",   : Kind",   :          : library.library. : API.  ②  :  ②        :
: workloads? : but      : but      :          :  ②       : Pointer  : Pointer   :
:          : using a  : using a  :          : Pointer  : to list  : to the    :
:          : private  : private  :          : to the   : of       : true      :
:          : struct.  : struct.  :          : true     : `*PodInfo`. : workload.workload. :
:          :  ②       :  ②       :          : workload. :          :  ③        :
:          : Pointer  : Pointer  :          :  ③       :          : Pointer   :
:          : to list  : to the   :          : Pointer  :          : to list   :
:          : of       : true     :          : to list  :          : of        :
:          : `*PodInfo`. : workload. :          : of       :          : `*PodInfo`. :
:          :          :  ③       :          : `*PodInfo`. :          :           :
:          :          : Pointer  :          :          :          :           :
:          :          : to list  :          :          :          :           :
:          :          : of       :          :          :          :           :
:          :          : `*PodInfo`. :          :          :          :           :
:          :          :          :          :          :          :           :
| How are  | No       | No       | Mapping  | Translator | Translator | Translator |
: the      : mapping.mapping. : mapping.mapping. : may be   : library  : library  : library   :
: true     :  Values  :  Values  : providedprovided : maps     : maps     : maps      :
: workload's : are      : are      : by who   : fields   : fields   : fields    :
: values   : providedprovided : providedprovided : wrote    : in the   : in the   : in the    :
: mapped   : by       : by       : the      : true     : true     : true      :
: into     : whoever  : whoever  : true     : workloadworkload : workloadworkload : workload  :
: the      : writes   : writes   : workloadworkload : into     : into     : into the  :
: abstractabstract : the pod  : the      : spec,    : the      : the      : abstract  :
: workloadworkload : templatetemplate : true     : or       : abstractabstract : abstractabstract : workload  :
: object?  : within   : workloadworkload : generated : workloadworkload : workloadworkload : object.   :
:          : the      : spec.    : by the   : struct.  : object.  : Mapping   :
:          : true     :          : true     :          :          : code in   :
:          : workloadworkload :          : workloadworkload : Mapping  : Mapping  : the       :
:          : spec.    :          : controller. : code in  : code in  : library   :
:          :          :          :          : the      : the      : is        :
:          :          :          :          : library  : library  : plugged   :
:          :          :          :          : is       : is       : in by     :
:          :          :          :          : contributed : contributed : various   :
:          :          :          :          : by       : by       : workload  :
:          :          :          :          : various  : various  : owners.   :
:          :          :          :          : workloadworkload : workloadworkload : Plugins   :
:          :          :          :          : owners   : owners   : are e.g.  :
:          :          :          :          : (how?)   : (how?)   : written   :
:          :          :          :          :          :          : in CEL.   :
| **Properties | **Embed  | **Embed  | **New    | **Translator | **Translator | **Translator |
: ↕        : in       : in True  : ResourceResource : Library  : API**    : Subresource** :
: Pattern  : PodSpec** : Workload** : Kind**   : **       :          :           :
: →**      :          :          :          :          :          :           :
| **Evaluation |          |          |          |          |          |           |
: of       :          :          :          :          :          :           :
: workloadworkload :          :          :          :          :          :           :
: user's   :          :          :          :          :          :           :
: UX**     :          :          :          :          :          :           :
| How is   | Read     | Read     | Follow   | The      | A        | Walk up   |
: the new  : directlydirectly : directlydirectly : one      : library  : custom   : the       :
: information : from     : from     : link     : could    : API      : controller :
: accessedaccessed : the      : the      : from     : return   : could    : refs.     :
: given a  : Pod.     : True     : Pod to   : the      : support  : Get       :
: Pod?     :          : Workload. : New      : name of  : "find    : `/workload` :
:          :          :          : Resource. : the      : controller : for each  :
:          :          :          :          : true     : from     : controller :
:          :          :          :          : workloadworkload : pod      : kind.     :
:          :          :          :          : resourceresource : uid."    : The       :
:          :          :          :          : by       :          : highest   :
:          :          :          :          : following :          : one that  :
:          :          :          :          : ownerRefs. :          : does not  :
:          :          :          :          : RequiresRequires :          : return    :
:          :          :          :          : polling  :          : an error  :
:          :          :          :          : rather   :          : is the    :
:          :          :          :          : than     :          : workload  :
:          :          :          :          : watch.   :          : you       :
:          :          :          :          :          :          : wanted.   :
| Lifecycle | No       | No       | User     | No       | No       | No        |
: management : change   : change   : creates  : change   : change   : change    :
: changes  :          :          : the new  :          :          :           :
: for      :          :          : object.  :          :          :           :
: pods/workloads? :          :          : _-or-_   :          :          :           :
:          :          :          : Controller :          :          :           :
:          :          :          : creates  :          :          :           :
:          :          :          : object.  :          :          :           :
:          :          :          :  User    :          :          :           :
:          :          :          : deletes  :          :          :           :
:          :          :          : the new  :          :          :           :
:          :          :          : object.  :          :          :           :
:          :          :          : _-or-_   :          :          :           :
:          :          :          : Controllers :          :          :           :
:          :          :          : deletes  :          :          :           :
:          :          :          : object   :          :          :           :
:          :          :          : when     :          :          :           :
:          :          :          : true     :          :          :           :
:          :          :          : workloadworkload :          :          :           :
:          :          :          : is       :          :          :           :
:          :          :          : deleted.deleted. :          :          :           :
:          :          :          :          :          :          :           :
| What     | Accidental | Newer    | New      | Newer    | Newer    | Error     |
: are      : reuse    : version  : resourceresource : version  : version  : translating :
: some     : of a     : of true  : kind     : of true  : of true  : true      :
: problemsproblems : string   : workloadworkload : doesn't  : workloadworkload : workloadworkload : workload  :
: that     : across   : than     : faithfully : than     : than     : spec to   :
: cannot   : workloads : the      : reflect  : the      : the      : API's     :
: be       : representing : code to  : true     : code to  : code to  : representation, :
: caught   : a gang   : interpret : workloadworkload : the      : the      : e.g.      :
: at       : scheduling : it in    : behavior. : library  : library  : detected  :
: resourceresource : group    : kube-scheduler. :          : that     : that     : by CEL    :
: creationcreation : or       :  Error   :          : interprets : interprets : expression. :
: time     : podset.  : translating :          : it.      : it.      :  _This    :
: through  :          : true     :          : Error    : Error    : might be  :
: validation :          : workloadworkload :          : translating : translating : validatedvalidated :
: of       :          : spec to  :          : true     : true     : at true   :
: individual :          : internalinternal :          : workloadworkload : workloadworkload : workload  :
: resources? :          : representation. :          : spec to  : spec to  : creation  :
:          :          :          :          : library's : API's    : time._    :
:          :          :          :          : representation. : representation. :           :
:          :          :          :          :          :          :           :
| Where    | On       | On the   | On the   | On       | On       | On every  |
: is       : every    : true     : new      : every    : every    : pod,      :
: status   : pod,     : workload. : ResourceResource : pod,     : pod,     : duplicated. :
: reportedreported : duplicated. :          : Kind.    : duplicated. : duplicated. : (No       :
: for a    :          :          :          : Then,    : Then,    : storage   :
: collection :          :          :          : summarised : summarised : for       :
: of       :          :          :          : by the   : by the   : summary   :
: pods?    :          :          :          : library  : aggregated : in a      :
: e.g.     :          :          :          : which    : API      : subresource, :
: the      :          :          :          : runs on  : server.  : and too   :
: scheduler :          :          :          : the      :          : expensiveexpensive :
: reports  :          :          :          : client.  :          : to        :
: failure  :          :          :          :          :          : recompute?) :
: to gang  :          :          :          :          :          :           :
: scheduleschedule :          :          :          :          :          :           :
: 1000     :          :          :          :          :          :           :
: pods.    :          :          :          :          :          :           :
| **Properties | **Embed  | **Embed  | **New    | **Translator | **Translator | **Translator |
: ↕        : in       : in True  : ResourceResource : Library  : API**    : Subresource** :
: Pattern  : PodSpec** : Workload** : Kind**   : **       :          :           :
: →**      :          :          :          :          :          :           :
| **Evaluation |          |          |          |          |          |           |
: of       :          :          :          :          :          :           :
: developer :          :          :          :          :          :           :
: implications :          :          :          :          :          :           :
: (kube-scheduler, :          :          :          :          :          :           :
: controllers) :          :          :          :          :          :           :
: **       :          :          :          :          :          :           :
| Can      | Not      | Not      | Yes.     | Yes.     | Yes.     | Yes.      |
: workloads : easily.  : easily.  : New      : For      : For      : For many  :
: be       :          :          : ResourceResource : many     : many     : true      :
: evaluated : Kube-scheduler : Kube-scheduler : Kind     : true     : true     : workloads, :
: prior    : or       : or       : can      : workloads, : workloads, : a         :
: to pod   : another  : another  : include  : a        : a        : translator :
: creationcreation : scheduler : scheduler : pod      : translator : translator : could     :
: (e.g.    : has to   : has to   : requirements, : could    : could    : know      :
: while    : wait     : wait     : and it   : know     : know     : what      :
: suspended) : for      : for      : can be   : what     : what     : types of  :
: using    : pods to  : pods to  : created  : types    : types    : pods it   :
: this     : be       : be       : before   : of pods  : of pods  : will      :
: approach? : created  : created  : any      : it will  : it will  : create.   :
:          : to see   : to see   : pods     : create.  : create.  :           :
:          : their    : their    : are.     :          :          :           :
:          : requirements. : requirements. :          :          :          :           :
:          :          :          :          :          :          :           :
:          : Finding  : Finding  :          :          :          :           :
:          : pod      : pod      :          :          :          :           :
:          : templates : templates :          :          :          :           :
:          : within   : within   :          :          :          :           :
:          : a true   : a true   :          :          :          :           :
:          : workloadworkload : workloadworkload :          :          :          :           :
:          : is       : is       :          :          :          :           :
:          : messy.   : messy.   :          :          :          :           :
| How      | 1 (or 0  | Unbounded | 1        | Unbounded | 1        | UnboundedUnbounded |
: many     : if       :          :          :          :          :           :
: kinds    : client   :          :          :          :          :           :
: does     : already  :          :          :          :          :           :
: the      : watchingwatching :          :          :          :          :           :
: client   : pods)    :          :          :          :          :           :
: have to  :          :          :          :          :          :           :
: watch    :          :          :          :          :          :           :
: (and     :          :          :          :          :          :           :
: have     :          :          :          :          :          :           :
: read-permissions :          :          :          :          :          :           :
: on)?     :          :          :          :          :          :           :
| How is   | No       | CRDs     | No       | Not      | Not      | By        |
: support  : action   : could    : action   : supported, : supported, : adding    :
: for a    : by       : be       : by       : or       : or       : CEL to a  :
: new      : installer. : labeled  : installer. : requiresrequires : requiresrequires : CRD for   :
: workloadworkload :          : to       :          : a        : a        : subresource :
: kind     :          : indicateindicate :          : complex  : complex  : support   :
: added    :          : they     :          : plugin   : plugin   : (or go    :
: (e.g.    :          : are      :          : scheme.  : scheme.  : code for  :
: after    :          : potential :          :          :          : non-CRD   :
: the CRD  :          : workloadworkload :          :          :          : workloads). :
: and      :          : types.   :          :          :          :           :
: controller :          : Client   :          :          :          :           :
: are      :          : has to   :          :          :          :           :
: installed). :          : watch    :          :          :          :           :
:          :          : CRDs     :          :          :          :           :
:          :          : for      :          :          :          :           :
:          :          : changes.changes. :          :          :          :           :
:          :          :          :          :          :          :           :
| What     | WorkloadWorkload | WorkloadWorkload | WorkloadWorkload | Each     | Same as  | Same as   |
: components : controllers : controllers : controllers : true     : the      : the cell  :
: need to  : may      :  need    : may      : workload's : cell to  : to the    :
: change   : need to  : to be    : need to  : maintainers : the      : left.     :
: after    : be       : extendedextended : be       : must     : left.    :           :
: the      : rebased  : to add   : rebased  : write a  :          :           :
: first    : and      : new      : and      : mapping  :          :           :
: release  : recompiled : fields   : recompiled : and add  :          :           :
: of       : to pick  : to       : to pick  : it to    :          :           :
: API-Rev1? : up new   : their    : up new   : the      :          :           :
:          : fields   : spec.    : fields   : library.library. :          :           :
:          : in       :          : in       :  Some    :          :           :
:          : PodTemplate. :          : PodTemplate. : mappingsmappings :          :           :
:          :          :          :          : could    :          :           :
:          :          :          :          : be       :          :           :
:          :          :          :          : providedprovided :          :           :
:          :          :          :          : by       :          :           :
:          :          :          :          : kubernetes :          :           :
:          :          :          :          : core     :          :           :
:          :          :          :          : project.project. :          :           :
:          :          :          :          :          :          :           :
| What     | WorkloadWorkload | WorkloadWorkload | No       | Each     | Same as  | Same as   |
: components : controllers : controllers : changes  : true     : the      : the cell  :
: need to  : may      :  need    : requiredrequired : workload's : cell to  : to the    :
: change   : need to  : to be    : if the   : maintainers : the      : left.     :
: when     : be       : extendedextended : user     : must     : left.    :           :
: new      : rebased  : to add   : manages  : update   :          :           :
: fields   : and      : new      : the new  : their    :          :           :
: are      : recompiled : fields   : resource. : mappingsmappings :          :           :
: added    : to pick  : to       :          : to       :          :           :
: in       : up new   : their    : Changes  : surface  :          :           :
: subsequent : fields   : spec.    : are      : new      :          :           :
: releases? : in       :          : needed   : information :          :           :
: i.e. as  : PodTemplate. :          : when a   : about    :          :           :
: API      :          :          : workloadworkload : the      :          :           :
: evolves  :          :          : controller : workloadworkload :          :           :
: toward   :          :          : creates  : in       :          :           :
: API-RevNAPI-RevN :          :          : the new  : order    :          :           :
:          :          :          : resource, : for the  :          :           :
:          :          :          : to be    : scheduler :          :           :
:          :          :          : able to  : to be    :          :           :
:          :          :          : set new  : aware    :          :           :
:          :          :          : fields.  : of it.   :          :           :
:          :          :          :          :          :          :           :
| Summary  | 🔴1      | 🔴4      | 🔴0      | 🔴4      | 🔴3      | 🔴3       |
:          : 🟡3      : 🟡2      : 🟡2      : 🟡3      : 🟡3      : 🟡4       :
:          : 🟢5      : 🟢3      : 🟢7      : 🟢2      : 🟢3      : 🟢2       :


Some key differences between the options:

-  _Translator Library_, _Translator API_, _Translator Subresource_, and _Embed in True Workload_ options all rely on action from the true workload maintainer beyond rebuilding.  There are many true workload types, so this seems good to avoid.
-  _Embed in True Workload, Translator Library, Translator API_, and _Translator Subresource _options all rely on clients to watch a large number of resource kinds.  This brings complexity to each client.
-  _Translator Library_, _Translator API_, _Translator Subresource_, and _New Resource Kind _options all define a single structure to represent any kind of workload for scheduling.  In the other two options, kube-scheduler and other clients likely still have to build their own simplified representation of a workload internally. Defining this in a standard way is necessary for multiple components to have a common view of workloads (kube-scheduler, cluster autoscalers, reschedulers, etc).

### Decision

Based on:

-  **New Resource Kind** has the fewest drawbacks of the 6 alternatives presented above.
-  **New Resource Kind** is used successfully already by 6 gang scheduling implementations. (See [Comparison of Existing Solutions](?tab=t.0#heading=h.cxn842iggsan)).

The decision is to use the "**New Resource Kind"** pattern.

This is similar to [[Public] Gang Scheduling Support In Kubernetes](https://docs.google.com/document/d/1q4a8uB_he2gx_lB2YFxsGaSVqMtF6CGJBhFkvArwnd4/edit?tab=t.0) option 1.

The new resource kind will represent an _abstract workload_. The spec will contain two kinds of information:

-  A subset of information that is copied from the true workload.
    -  This is sort of like like an abstract base class of a true workload.

-  Additional information not in the abstract base class
    -  e.g. gang scheduling timeout)

-  Together these are similar to the "Decorator" pattern.

Workloads can occur in stacks.  Several pods are controlled by a true workload.  Then one or more true workloads are controlled by another true workload.   

Examples stacks include:

-  Pods → Jobs → JobSet
-  Pods → Jobs → CronJob
-  Pods → ReplicaSets → Deployment 
-  Pods → ReplicaSets → Deployment → argoproj.io.Rollout
-  Pods → StatefulSets
-  Pods → StatefulSets → LeaderWorkerSet

The stacks are tree-shaped.  Pods set their controllerRef to their ReplicaSet, and ReplicaSets set their controllerRef to their Deployment. Apiserver validation enforces only one controllerRef per object.  Therefore, these always form a tree.

The top item in the stack is generally created by a user, and the remaining items are created by controllers.

We compare two design options:

1. A pod can have several abstract workload objects describing it.
1. A pod has a single abstract workload object.

<table>
  <thead>
    <tr>
      <th><strong>Properties ↓   Pattern →</strong></th>
      <th><strong>One Per Stack</strong></th>
      <th><strong>N Per Stack</strong></th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Main ideas</strong><br>
</td>
      <td><ul>
<li>Each Pod is described by one Abstract Workload Object</li>
</ul>
<ul>
<li>Abstract Workloads represent what the <em>user</em> created <br>
(e.g. <code>kubectl create -f jobset.yaml</code>)</li>
</ul>
</td>
      <td><ul>
<li>Each Pod is described by one or more Abstract Workload Objects.</li>
</ul>
<ul>
<li>A pod with a stack of 3 controllers will have 3 Abstract Workload Objects.</li>
</ul>
<ul>
<li>Abstract Workloads represent how Kubernetes workloads are implemented.</li>
</ul>
<ul>
<li>In some cases, this may be as much a historical accident as a necessity.</li>
</ol>
</li>
</ul>
</td>
    </tr>
    <tr>
      <td><strong>How to make the new resources</strong> <br>
<br>
<em>Describes the most typical case.</em></td>
      <td><ul>
<li>Create an abstract workload <code>A</code> which describes the behavior of the whole stack.</li>
</ul>
<ul>
<li>Set all pod templates to refer to <code>A</code>.</li>
</ul>
</td>
      <td><ul>
<li>Create an abstract workload <code>A</code> which describes the behavior of the top controller.</li>
</ul>
<ul>
<li>Create abstract workloads <code>B-1</code> ... <code>B-N</code>, one for each of the controllers in the next level down.</li>
</ul>
<ul>
<li>Set each pod template to refer to <code>B-1</code> ... <code>B-N</code>.</li>
</ul>
<ul>
<li>Set <code>B-1</code> ... <code>B-N</code> to refer to <code>A</code></li>
</ul>
<ul>
<li>Set <code>A</code> to refer to each of  <code>B-1</code> ... <code>B-N</code>.</li>
</ul>
<ul>
<li>And so on if there are additional levels.</li>
</ul>
</td>
    </tr>
    <tr>
      <td><strong>Visual Example</strong><br>
<br>
<em>Deployment used as example.</em></td>
      <td><p><img src="insert_image_url_here"></p></td>
      <td><p><img src="insert_image_url_here"></p></td>
    </tr>
    <tr>
      <td><strong>Policy Example</strong><br>
<br>
How are the levels of the workload represented, and how is a policy, such as a topology placement policy, attached to the levels?<br>
</td>
      <td><br>
undefined</td>
      <td><br>
undefined</td>
    </tr>
    <tr>
      <td colspan="3"><strong>Lifecycle </strong></td>
    </tr>
    <tr>
      <td><strong>Properties ↓   Pattern →</strong></td>
      <td><strong>One Per Stack</strong></td>
      <td><strong>N Per Stack</strong></td>
    </tr>
    <tr>
      <td><strong>User Creation</strong><br>
<br>
When is a user able to make the abstract workload themselves?</td>
      <td>A user can create one Abstract Workload at the same time as they create the top resource in the stack, and delete it when they are done.  This should be possible in most cases.</td>
      <td>For a tree with predetermined fan-outs, the user makes one for each expected true workload controller in the tree.<br>
For example: a JobSet with a fixed number N of ReplicaJobs, the user could make N+1 Abstract Workload objects ahead of time. <br>
<br>
For a tree with runtime changes to fan-out, this is not practical.<br>
For example, a Deployment makes a new ReplicaSet in response to each update to the pod spec.  Creating a new Abstract Workload at the same time is not convenient.</td>
    </tr>
    <tr>
      <td>Can True Workload Controllers make default Abstract Workload objects for the user?</td>
      <td>Yes.  Controllers can typically create Abstract Workloads with reasonable defaults.  </td>
      <td>Yes. Same as the cell to the left.</td>
    </tr>
    <tr>
      <td>Can True Workload Controllers make Abstract Workload objects for the user, with non-default policies in them?</td>
      <td>In order for the user to customize the policies, the True Workload needs a template for any fields of the Abstract Workload the user wants to override (e.g. gang scheduling timeout, preferred topology domain, etc).</td>
      <td>Yes. Same as the cell to the left.</td>
    </tr>
    <tr>
      <td>Can users create Abstract Workload objects with fully customized policies?<br>
<br>
<em>e.g. if there is a new field on Abstract Workload that the scheduler understands, but the Deployment controller does not yet understand.</em></td>
      <td>Yes, the user can make one Abstract Workload themselves, and the controller will use this one instead.</td>
      <td>Sometimes, for the same reasons as the above cell.</td>
    </tr>
    <tr>
      <td colspan="3"><strong>Expressiveness</strong></td>
    </tr>
    <tr>
      <td>Are there any known commonly-used workloads that cannot be modeled by this approach?</td>
      <td>No</td>
      <td>No</td>
    </tr>
    <tr>
      <td colspan="3"><strong>Complexity</strong></td>
    </tr>
    <tr>
      <td>How many objects are needed to model one workload</td>
      <td>One</td>
      <td>One or more.</td>
    </tr>
    <tr>
      <td>What does client code look like to iterate over the Abstract Workload.</td>
      <td>Nested for loops.<br>
Operates on fields of one object.</td>
      <td>Tree traversal.<br>
Crosses objects.</td>
    </tr>
    <tr>
      <td>Can a client observe updates coherently?</td>
      <td>Yes, all updates to a single <code>Workload</code> object appear in one watch update.</td>
      <td>If different kinds are used (e.g. <code>PodGroup</code> and <code>PodSubGroup</code>) then there is no guarantee of seeing changes to different types at the same time.  </td>
    </tr>
    <tr>
      <td colspan="3"><strong> Summary</strong></td>
    </tr>
    <tr>
      <td></td>
      <td>🔴0   🟡0  🟢7</td>
      <td>🔴0   🟡4  🟢3</td>
    </tr>
  </tbody>
</table>

The One Per Stack option is sufficiently expressive, easier to gradually adopt and override, and is less complex.

#### Decision 

Based on:

-  **One Per Stack** has fewer drawbacks.
-  **One Per Stack** appears sufficiently expressive to cover all commonly-used workload stacks.

The decision is to use the "One Per Stack" pattern.

## Detailed Design

### Requirements

Having narrowed down the high-level approach, we are ready to state some requirements.    
**Goal**

-  Design a new resource that summarizes workload behavior and is decorated with scheduling requirements.  
-  This API is for use by kube-scheduler, and other scheduling/orchestration components.  It must support gang scheduling immediately, and be extensible to support more detailed workload description and additional scheduling policies.
    -  Use of the new resource is optional.  External schedulers may make use of it, but are not required to.

**Immediate Requirements**  
The first GA version of this API should:

-  Be verified to work with a pre-defined set of [workload kinds](?tab=t.0#heading=h.156lanok18s8) that are often used with gang scheduling,
    -  It is expected that by GA common workload types will be able to make a Workload automatically.
    -  However it will remain possible for a user to create a Workload first for workloads that are not yet able to do this.  This option only requires that controller are rebuilt with the latest `k8s.io/apis/*` packages to pick up the Pod field.
    -  Should demonstrate usable [lifecycle management](?tab=t.0#heading=h.6vapq277wpk3) for common scenarios with these workload kinds.

-  Be open to working with other workload kinds (such as private custom workloads) without changes to its controllers or kube-scheduler.
-  Work with pod-at-a-time scheduling in Kube Scheduler.
-  Work race-free given under kube-apiserver's event ordering guaranteed only within a single resource type.
-  Support gang scheduling of single workloads.
-  Kube-scheduler must behave the same when workloads don't use the gang-scheduling feature. 
-  Kube-scheduler must continue to support running all workload types (including bare pods and unknown custom controllers) in the same cluster as workloads that use gang scheduling.
-  However, it does not have to support gang scheduling of any imaginable workload.
-  Do not require the kube-scheduler to watch every possible workload type.  
    -  Kueue and Volcano _do_ watch predetermined true workloads.  However, it seems preferable to keep dependencies on non-core types out of `kube-scheduler`, at least initially. 

**Iterative Approach**

-  Expect successive releases to add fields to the new resource, in sync with adding new scheduling capabilities to kube-scheduler.

**Future Requirements Inferred from Existing Solutions**  
These requirements are informed by a [Comparison of Existing Solutions](?tab=t.3zjbiyx2yldg#heading=h.9qbw2b4lxrgr).  
The first iteration does not need to support the following features, but the design process must consider how they might be added later.

-  Optionally allow users to provide the shapes (partial pod templates) of all pods in the workload, for workloads where this can be known in advance.  
-  Allow expressing  Topology-aware scheduling (TAS) to kube-scheduler in some form.
    -  Allow associating topology placement requirements with gangs and/or parts of gangs. 

-  Support at least [two-level](https://github.com/kubernetes-sigs/kueue/issues/5439) TAS:
-  This is where sub-parts of a workload request each request placement in a smaller  topology domain, and the whole workload request placement in an (unspecified) larger topology domain.
-  Ensure there is a place for both DRA-style and level-style topology requests to facilitate faster adoption.
-  Optionally provide a way to determine the index of each pod.

**Additional Future Requirements**

-  Work with PDBs where needed (LeaderWorkerSet) by modeling how replicas of gangs are used.
-  Work with multi-pod scheduling by modeling of sets of homogeneous pods within a gang.
-  Support various Fault-tolerance policies when failures occur to pods in gangs.
-  Nice-to-have: support specifying min and max size for horizontally scaled sets of pods.
-  Nice-to-have: support specifying min and max size for vertically scaled pods.

### Naming of New Resource

Apigroup options:

-  `scheduling` – says that this is about the scheduling aspects of the workload, not, for example, health checks or rolling updates.
-  `lifecycle` - says that this is about scheduling, eviction and re-scheduling of replacement pods of a workload.
-  `workload/workload_lifecycle `- says that this is focused on workloads and their lifecycle

Naming options for Kind:

-  `Workload` – Represents a generic workload.
-  `SchedWorkload` – Represents a generic workload, focusing on information that the scheduler needs to know.
-  `AbstractWorkload` – emphasizing that is (partly) an abstract data type
-  `WorkloadWrapper` or `WorkloadDecorator` – describing the workload pattern.
-  `WorkloadPolicy` – suggests a policy for all workloads, which this is not.
-  `WorkloadSummary` – since it has a summary of the key information in the actual workload.
-  `WorkloadContraints
`-  `PodGang`
-  `PodConstraints
`-  `PlacementPolicy`
-  `PodOperationPolicy` (POP)
-  `WorkloadManagement
`-  `PodRegimen`
-  `Regimen
`-  `WorkloadStrategy`
-  `PodFormation
`-  `PodLayout`
-  `Layout`

The rest of the doc uses the term `Workload.`  The spelling `scheduling/Workload` is used when confusion with other terms (e.g. `kueue.x-k8s.io/Workload`) is likely.

### Mapping Workload to a True Workload

#### True Workload Reference

The `Workload` will have a reference to the true workload that it corresponds to.  This information is optional.   It is optional because:

1. Pods must reference their Workload to prevent a race condition in the scheduler.
1. The scheduler will not interpret true workloads.

The authoritative data about how to schedule a pod is when a Pod has an object reference to a `Workload`.

This reference looks like this:

undefined

However clients are encouraged to set this reference because:

1. A user interface (e.g. kubectl) may want to link to the true workload from a Workload list, or vice versa
1. Admission controllers might use this field to set default values in a `Workload` based on the fields of known true workloads.

#### True Workload Stacks

A pod can have a stack of true workloads (controllers).  Examples include:

-  `Pod` → `Job` → `JobSet
`-  `Pod` → `Job` → `CronJob`
-  `Pod` → `ReplicaSet` → `Deployment` 
-  `Pod` → `ReplicaSet` → `Deployment `→ `argoproj.io.Rollout
`-  `Pod` → `StatefulSet`
-  `Pod` → `StatefulSet `→ `LeaderWorkerSet`

When custom operators are included, there are hundreds of different "stacks" in use. 

#### One-to-One

A key purpose of an abstract workload is to simplify and standardize the complexities of real workload controller implementation. Mirroring their nested structure does not appear to be necessary, and would add a lot of complexity to the design. Therefore, we will allow each pod to have at most one `Workload` that applies to it.  

Given a Pod's stack of controllers, there is not an obvious rule for how far up the stack we should go.  Consider:

-  `Pod` → `Job` → `JobSet`
    -  When `JobSet` does not use `dependOn` or a non-default `startupPolicyOrder`:  The whole `JobSet` needs to be gang scheduled, and needs topology-aware placement. Therefore, the `Workload` should correspond to the `JobSet`.
    -  When `JobSet` uses `dependOn` or a non-default `startupPolicyOrder`:  Each `replicatedJob` needs to be gang scheduled, and needs topology-aware placement. Therefore, the `Workload` should correspond to the `Job`.

-  `Pod` → `Job` → `CronJob`: Two jobs that start an hour apart will not interact in the scheduler.  They cannot be gang scheduled. Typically, it will not be necessary that they run in the _same_ topology domain.  Therefore, the `Workload` should correspond to the `Job`.

Therefore, it is left to the users to determine which controller level corresponds to the `Workload`; kube-scheduler does not need to determine what is the "top workload" in a workload stack. 

### Generation and defaulting of Workload Objects

Users may create a Workload object manually.

Controllers _may_ generate a `Workload` by default, but _should_ allow the user to override this choice. It is suggested that the user choose the highest ancestor to which `Workload` it makes sense to apply a  `Workload`s policy.

An admission controller may be written to

1. Specify any unspecified values of an incomplete Workload object, according to local cluster policies.
1. Create a default Workload object whenever a pod is seen without one, and inject it into that pod, and all its peers (required admission controller to have detailed understanding of the corresponding controller of pods.

If a user explicitly specifies a `Workload` and links it to pods via pod template(s), then neither the true workload controller, nor any admission controllers, should replace that `Workload` with a different one.

When a controller or admission controller creates a `Workload`, it should:

-  attach it to the true workload via `metadata.ownerReferences` for garbage collection.
-  fill in the `spec.workload` field of each pod that is created (see below).
-  not create more than one `spec.workload`.
-  not set different values of `spec.workload` in its child pods.

For example, once gang scheduling support is widespread:

-  the Job controller will be changed to create a default `Workload` and fill in the `spec.workload` for any indexed Job lacking those.
-  the StatefulSet controller might be changed to create a default `Workload` for any StatefulSets with parallel startup.
-  Fields may be added to true workloads that correspond to `Workload` policy (trait) fields.

### Evaluation of Suspended Workloads

Kueue or a similar orchestrator wants to be able to interpret all of a workload's requirements before any pods are created. Therefore, if the controller intends to create a default `.spec.workload` and `Workload`, it should:

1. Fill in the `spec.workload` of each created pod at creation time.
1. Make the `Workload` in an admission controller, or at first reconciliation.  It should make it even if suspended.

### Linking Pod to Workload

It is necessary to put some information on a Pod which says that it has a Workload object. Any solution which does not add at this least 1 bit of information to a pod has a race condition.  Specifically:

1. Serialization of writes between two different kinds is not guaranteed by kube-apiserver
1. Therefore, Pod and Workload writes can be seen by kube-scheduler at possibly very different times.
1. Every pod seen by kube-scheduler must be promptly acted on.
1. We have to handle pods that have Workload differently from those that do not.
1. Therefore, a pod must indicate whether it has an associated workload.

Regarding point C, one might ask if Workloads can be watched instead.  This is not sufficient.  Kube scheduler needs to continue to support pods without Workloads indefinitely, including "bare pods" with no corresponding true workload controller.   

There are two main options for doing this:

1. Use a field in Pod – we want to do this at GA, but maybe it is not the most expedient choice for Alpha.
1. Use an annotation – this is another option for alpha.

| **Properties ↓          | **Use Pod Field in      | **Use Annotation in     |
: Option →**              : Alpha**                 : Alpha**                 :
| ----------------------- | ----------------------- | ----------------------- |
| **Requires deprecation  | No, does not require    | Yes, requires           |
: of  an annotation**     : deprecation.            : deprecation             :
: _This is considered     :                         :                         :
: difficult by            :                         :                         :
: maintainers who have    :                         :                         :
: had to do this          :                         :                         :
: before._                :                         :                         :
| **Risk of unneeded      | Yes, risk of this       | No, not a risk.         |
: field in Pod forever**  :                         :                         :
:  _In the event that we  :                         :                         :
: change the design       :                         :                         :
: after alpha, then any   :                         :                         :
: field in pods would     :                         :                         :
: have to stay there      :                         :                         :
: forever_                :                         :                         :
| **Immediate Feedback**  | No, new fields are not  | Yes, annotations are    |
:  _Immediately after     : necessarily copied by   : always copied by        :
: the first alpha         : non-core controllers    : controllers to the      :
: release, can people     : to the created pods.    : created pods.           :
: start using alpha       :                         :                         :
: Workload with existing  :                         :                         :
: non-core controllers    :                         :                         :
: (JobSet, MPIJob, LWS,   :                         :                         :
: etc)._                  :                         :                         :

Reuse of other fields of pod (owner-reference, scheduling-gate) was considered, but these other options had additional downsides.  See this tab: [[External]  API Design for Gang and Workload-Aware Scheduling](https://docs.google.com/document/d/1ulO5eUnAsBWzqJdk_o5L-qdq5DIVwGcE7gWzCQ80SCM/edit?disco=AAABqgVRxY0&tab=t.12l3myddr3id)

#### Decision

**For Beta and GA, use a Field in Pod**.  
For Alpha, the decision is pending community feedback.

#### Implementation

We will add a field to Pod with an object reference to the Workload object.  This field is called `spec.workload`.  A pod with this field set looks like this:

undefined

This will be used for Alpha and beyond.

Semantics of the `Workload` field of `Pod`:

-  Cross-namespace references are not allowed.
-  This field is immutable after creation.
    -  Pods which don't set this field can be orphaned and adopted, if such workflows are required.
    -  We can revisit mutability later if users need both orphan/adopt workflow and gang scheduling – this seems unlikely.

-  The field defaults to unset in the API.
-  Unset means that the pod does not have any special scheduling requests.
-  Workloads that need gang scheduling (and do not rely on some other scheduler or scheduler plugin) need to set this field.
-  Existence of the referred-to object is not validated at pod creation time.

Runtime Validation:

-  The scheduler may detect inconsistencies between the Pods and their `Workload`.
-  The inconsistencies can be reported in status and/or via events.

### Linking Pod to Workload Parts

Pods are associated with their Workload via the pod's `spec.Workload`.  
How can pods be associated with which sub-part of Workload they belong to? Workload sub-parts means GangGroup, RankedGroup, or EqGroup.

 Two options were considered:

1. **Label** value of the Pod identifies what Workload part it belongs to.
1. Using the **FieldPath** part of the Pod's `spec.Workload` [ObjectReference](https://github.com/kubernetes/api/blob/ed58f06b96730db416c74882db704b71516c5b9e/core/v1/types.go#L7336C2-L7336C11), but with custom validation on the path to support indexing by name
1. **Custom fields** in the Pod's `spec.Workload` struct, making it similar to, but not exactly an ObjectReference.

Example of _Label_ option:

undefined

Example of _FieldPath_ option:

undefined

Example of _Custom_ option:

undefined

| **Properties ↓   | **Label**        | **FieldPath**    | **Custom**         |
:  Option →**      :                  :                  :                    :
| ---------------- | ---------------- | ---------------- | ------------------ |
| **Disjointness   | Yes, if we do    | Yes, pod can     | Yes, custom        |
: can be ensured   : validation on    : only point to    : fields will only   :
: at pod           : the workload     : one fieldPath    : point to one       :
: validation       : that requires    :                  : sub-group of the   :
: time?**          : all selectors    :                  : workload.          :
:                  : for peer groups  :                  :                    :
:                  : to use the same  :                  :                    :
:                  : label key.       :                  :                    :
| **Works with     | Yes.  No label   | Yes.  No         | Yes.  No custom    |
: Job/MPIJob/StatefulSet : needed.          : fieldPath        : fields needed.     :
: without          :                  : needed.          :                    :
: controller       :                  :                  :                    :
: changes?**       :                  :                  :                    :
: _Job, MPIJob     :                  :                  :                    :
: and StatefulSet  :                  :                  :                    :
: have (at most)   :                  :                  :                    :
: a single Gang._  :                  :                  :                    :
| **Distinguishes  | Yes.  JobSet     | Yes.   The user  | Yes.   The user    |
: gangs of JobSet  : has this label   : can fill in a    : can fill in a      :
: ** **without     : already\\\:      : different value  : different value    :
: controller       : ```              : of fieldPath     : of gangName into   :
: changes?**       : jobset.sigs.k8s.io/replicatedjob-name\\: : into each        : each               :
: _JobSet can      : name ```         : replicatedJob's  : replicatedJob's    :
: correspond to    :                  : template.        : template.          :
: either one gang  :                  :                  :                    :
: for all          :                  :                  :                    :
: replicatedJobs,  :                  :                  :                    :
: or one per       :                  :                  :                    :
: replicatedJob._  :                  :                  :                    :
| **Distinguishes  | Yes.             | No.  There are   | No.  There are     |
: gangs of         : LeaderWorkerSet  : only 2 pod       : only 2 pod         :
: LeaderWorkerSet  : has this label   : templates that   : templates that     :
: without          : already\\\\\:    : make an          : make an unbounded  :
: controller       :  ```             : unbounded        : number of          :
: changes?**  _A   : leaderworkerset.sigs.k8s.io/group-key\\\\: : number of        : replicas.          :
: LeaderWorkerSet  : group-id ```     : replicas.        :                    :
: has any number   :                  :                  :                    :
: of replica       :                  :                  :                    :
: gangs._          :                  :                  :                    :
| **Client         | No.  Pod labels  | Yes.  The        | Yes.  The custom   |
: protected from   : might change at  : fieldPath of     : fields of the pod  :
: runtime mapping  : runtime.         : the pod can be   : can be made        :
: changes.**       : Changes to the   : made immutable.  : immutable.         :
:                  : Pod and          :                  :                    :
:                  : Workload cannot  :                  :                    :
:                  : be obsThis add   :                  :                    :
:                  : complexity for   :                  :                    :
:                  : the client to    :                  :                    :
:                  : update its       :                  :                    :
:                  : mapping, and     :                  :                    :
:                  : may cause a      :                  :                    :
:                  : race condition   :                  :                    :
:                  : if the latest    :                  :                    :
:                  : pod state is     :                  :                    :
:                  : not observed     :                  :                    :
:                  : coherently with  :                  :                    :
:                  : changes to the   :                  :                    :
:                  : cause a race     :                  :                    :
:                  : condition, and   :                  :                    :
:                  : make looking up  :                  :                    :
:                  : the mapping      :                  :                    :
:                  : more expensive   :                  :                    :
:                  : at runtime.      :                  :                    :
| **Works for      | Yes, with a      | Yes, but         | Yes, just add      |
: multi-level      : selector at      : requires a       : more fields.       :
: nesting**  _See  : each level of    : long,            :                    :
: API-RevN         : the Workload.    : non-standard     :                    :
: discussion._     :                  : fieldPath,       :                    :
:                  :                  : like\\\:   ```   :                    :
:                  :                  : fieldPath\\:     :                    :
:                  :                  : spec.gangGroup[name=gg1]\ :                    :
:                  :                  : ```   ```        :                    :
:                  :                  : .rankedGroup[name=rg1] :                    :
:                  :                  : ```              :                    :
| **How to handle  | Similarly, but   | No, pod indexes  | Add a custom       |
: index (rank)**   : have to support  : won't have       : field with the     :
:                  : annotations      : separate fields  : numeric index.     :
:                  : too.             : in Workload.     :                    :

Index would be handled similarly.

**Evaluation**:  FieldPath is not a good fit. Label and Custom are tied.

**Decision**:   Evaluate at implementation time.

The document currently reflects the Label option.

### Composition

For user convenience, we can modify the Job controller to make a `Workload` object automatically when it sees an indexed Job. Specifically, it would:

-  Detect that the job requires gang scheduling – perhaps all indexed jobs should be ganged by default.
-  It makes a Workload object
-  It injects the object reference to it into the pods as they are created.

What happens when multiple Jobs are combined into a single workload with gang scheduling requirements (e.g. JobSet)?

-  JobSet would make the Workload object first and include it in the pod template of each Job it makes.
-  The Job controller will see the existing object reference and not auto-create the Workload object.

The proposed API-RevN object contains a 4-level hierarchy of grouping.  This allows it to support many different shapes of workloads.  However, we want to avoid having to handle an _unbounded_ number of levels of nesting in clients. For this reason, pods can only have one `Workload`.  That is, a `Workload` cannot contain another nested `Workload`.

### Lifecycle

A controller that auto-deletes old workload instance may adopt a user-created `Workload`  
by attaching it to the true workload object via ownerRef for later garbage collection.  Job in particular should do this due this when any of `successfulJobsHistoryLimit`, `failedJobsHistoryLimit`, or `ttlSecondsAfterFinished` is set.

For example, the Job controller might be changed to create a default `Workload` with a gang scheduling request for any indexed Job that does not specify one, but it would not do this for non-indexed jobs. Similarly, StatefulSet might use `podManagementPolicy` to determine whether or not to set a default value. Controller maintainers should make this decision.

Users can override the controller's default by creating a `Workload` with a different behavior (e.g. `minCount: 0` to turn off gang scheduling).

The above results in 4 different lifecycle patterns, which we name here:

1. **Manifest Lifecycle** – User owns the entire lifecycle of the `Workload` resource.
    -  When using this lifecycle pattern, users must delete the resource.
    -  When using this lifecycle pattern, users avoid reuse of `Workload` across workloads.  For example, a cron job that occasionally overruns should not refer to the same once-created `Workload`.

1. **Adoption Lifecycle** – User creates `Workload`, and the workload controller adopts it and deletes it.
    -  Workloads that self-delete like Job should support this option.

1. **Default-created Lifecycle** – Controller creates `Workload` automatically.
    -  When it does this, it always ensures it is deleted when the workload is deleted.
    -  Many controllers (e.g. Indexed Job, JobSet, MPIJob) should be able to implement reasonable defaults.

1. **Templated Lifecycle** – Controllers may have one template of a `Workload `in their spec, or an equivalent set of fields. 
    -  The controller always ensures that a `Workload` it creates is deleted when the workload is deleted.
    -  This could be useful for e.g. a  CronJob of overlapping IndexedJobs.
    -  Or, imagine a MultiJob that makes many IndexedJobs based on some generator function.

Kueue and similar orchestrators want to be able to interpret all of a workload's requirements before any pods are created. It is unknown at this time if Kueue and similar orchestrators will make use of `Workload`.  It is not planned for Kube-scheduler to provide an admission controller.  Nevertheless, with this in mind, controllers implementing the Default-created Lifecycle or the Templated Lifecycle _should_ create the `Workload` as soon as possible, before any pods, and even if the true workload is suspended.

| Workload Type    | Recommended Lifecycles to support                        |
| ---------------- | -------------------------------------------------------- |
| StatefulSet      | **Manifest, Default-created**                            |
| Job              | **Adoption, Default-created**                            |
| CronJob          | **Templated**                                            |
| LeaderWorkerSet  | **Manifest, Default-created**                            |
| JobSet           | **Adoption, Default-created**                            |
| MPIJob           | **Adoption, Default-created**                            |

### Type Definition for API-Rev1

In designing API-Rev1, we came up with some [potential designs for API-RevN](?tab=t.0#heading=h.cxn842iggsan), and then worked backwards, by removing fields unnecessary in API-Rev1.

For API-Rev1, a `Workload` will have the following definition:

undefinedThis struct will be used at alpha, with an option to adjust before beta if further discussion on API-RevN suggest a need to change.

#### Example Object

An example instance looks like this:

undefined

Explanation:

-  `replicationMode: unreplicated` means that there is not a variable number of copies of what is described in the rest of the spec.
-  `gangGroups`: gives a list of top-level grouping of pods in this workload.  In this case, there is one item in the list.  
-  `minCount` means that the scheduler does not need to act on this gang until it sees 100 pods.
-  `scheduleTimeoutSeconds` says that if this gang is in a partially-scheduled state for this long, the  scheduler should requeue the workload and retry later.

### Examples for Common Workloads

Examples of how this would be used with some common workloads:

| Workload     | Example use  | Want Gang    | Config with  | Suggested       |
: Type         : case         : Scheduling?  : API-Rev1     : Settings with   :
:              :              :              :              : API-RevN        :
| ------------ | ------------ | ------------ | ------------ | --------------- |
| StatefulSet  | AI           | Yes.         | Set    ```   | No change to    |
: with         : Inference    :              : minCount     : `minCount`.     :
: `podManagementPolicy\: : with         :              : ``` to SS's  : Specify a       :
: "Parallel"`  : indexed      :              : `spec.replicas` : topology        :
:              : pods. Fixed  :              :              : request.        :
:              : [world       :              :              : Specify pod     :
:              : size].       :              :              : index label.    :
| StatefulSet  | Database     | No.          | Not          | May use         |
: with    ```  : with leader  :              : necessary.   : `GangMode\\\\\:`GangMode\\\\\: :
: podManagementPolicy\: : and          :              : May to       : Off`.  May      :
: "OrderedReady" : scalable     :              : create       : specify a       :
: ```          : number of    :              : `Workload`   : topology        :
:              : readonly     :              : with         : request.        :
:              : replicas.    :              : `minCount\:  : Specify pod     :
:              :              :              : 0`           : index label.    :
| Job with     | AI Training  | Yes.         | Set    ```   | No change to    |
: `parallelism` : with         :              : minCount     : `minCount`.     :
: equal to     : indexed      :              : ``` to       : Specify a       :
: `completions`completions : pods.        :              : `completions`. : topology        :
: `(indexed)   :              :              :              : request.        :
:              :              :              :              : Specify pod     :
:              :              :              :              : index label     :
:              :              :              :              : for balancing   :
:              :              :              :              : of workers.     :
| MPIJob,      | AI           | Yes.         | Set    ```   | No change to    |
: PyTorchJob   : Inference    :              : minCount     : `minCount`.     :
: TrainJob     : with         :              : ``` to       : Specify a       :
:              : indexed      :              : world size.  : topology        :
:              : pods.        :              :              : request.        :
:              :              :              :              : Specify pod     :
:              :              :              :              : index label     :
:              :              :              :              : for balancing   :
:              :              :              :              : of workers.     :
| Job with     | Bag of       | No.          | Not          | May use         |
: ```          : tasks.       :              : necessary.   : `GangMode\\\:   :
: parallelism  :              :              : May to       : Off`.           :
: not equal    :              :              : create       :                 :
: to           :              :              : `Workload`   :                 :
: completions  :              :              : with         :                 :
: ```          :              :              : `minCount\:  :                 :
:              :              :              : 0`           :                 :
| JobSet       | Large scale  | Yes, per     | One          | No change to    |
: where\\\\\\:where\\\\\\: : AI Training  : Job.         : `Workload`   : `minCount`.     :
: – every Job  : with         :              : for the      : Specify a       :
: is Indexed   : indexed      :              : whole        : topology        :
: – each Job   : pods.        :              : JobSet. One  : request for     :
: neither      :              :              : GangGroup    : the whole       :
: uses         :              :              : per          : workload, and   :
: `DependsOn`  :              :              : replicatedJob. : a more          :
: nor          :              :              : Set          : specific        :
: deprecated   :              :              : `minCount`   : topology        :
: `StartupPolicyOptions`\\: :              :              : for each     : request per     :
:  `AnyOrder`  :              :              : one to the   : GangGroup if    :
:  –           :              :              : `completions` : desired.        :
: replicatedJobs :              :              : of that      : Specify pod     :
: don't run    :              :              : corresponding : index label     :
: concurrently; :              :              : job.         : for balancing   :
: they can     :              :              :              : of workers.     :
: run in any   :              :              :              :                 :
: order.       :              :              :              :                 :
| JobSet       | Large scale  | Yes, one     | One          | One `Workload`  |
: where\\\\\\:where\\\\\\: : AI Training  : gang for     : `Workload`   : for the whole   :
: – every Job  : with         : all Jobs.    : for the      : JobSet. One     :
: is Indexed   : indexed      :              : whole        : GangGroup for   :
: – each Job   : pods.        :              : JobSet. One  : the whole       :
: neither      :              :              : GangGroup    : JobSet. One     :
: uses         :              :              : per          : RankedGroup     :
: `DependsOn`  :              :              : replicatedJob. : per indexed     :
: nor          :              :              : Set          : Job.  Set       :
: deprecated   :              :              : `minCount`   : `minCount` to   :
: `StartupPolicyOptions`\\: :              :              : to the       : the total       :
:  `AnyOrder`  :              :              : total of     : number of       :
:  –           :              :              : all job's    : `completions`   :
: replicatedJobs :              :              : `completions`. : of all jobs     :
: need to run  :              :              :              : Specify a       :
: concurrently. :              :              :              : topology        :
:              :              :              :              : request for     :
:              :              :              :              : the whole       :
:              :              :              :              : workload, and   :
:              :              :              :              : a topology      :
:              :              :              :              : request per     :
:              :              :              :              : GangGroup if    :
:              :              :              :              : desired.        :
:              :              :              :              : Specify pod     :
:              :              :              :              : index label     :
:              :              :              :              : per rank group  :
:              :              :              :              : for balancing   :
:              :              :              :              : of workers.     :
| JobSet       | Unknown use  | Not          | Not          | May use         |
: where\\\\\:  : case         : supported    : necessary.   : `GangMode\\:    :
: - some jobs  :              :              : May to       : Off`.           :
: are not      :              :              : create       :                 :
: indexed -    :              :              : `Workload`   :                 :
: some jobs    :              :              : with         :                 :
: uses         :              :              : `minCount\:  :                 :
: `DependsOn   :              :              : 0`           :                 :
: `(or         :              :              :              :                 :
: deprecated   :              :              :              :                 :
: `StartupPolicyOptions`\\: :              :              :              :                 :
:              :              :              :              :                 :
: `AnyOrder`)  :              :              :              :                 :
:              :              :              :              :                 :
| JobSet       | Large scale  | No.          | Not          | May use         |
: where any    : AI Training  :              : necessary.   : `GangMode\\\:   :
: of the       : with         :              : May to       : Off`.           :
: `replicatedJobs` : indexed      :              : create       :                 :
: uses         : pods.        :              : `Workload`   :                 :
: `DependsOn`  :              :              : with         :                 :
: (or          :              :              : `minCount\:  :                 :
: deprecated   :              :              : 0`           :                 :
: `StartupPolicyOptions`\: :              :              :              :                 :
:              :              :              :              :                 :
: `AnyOrder`)  :              :              :              :                 :
| TrainJob     | AI           | Yes          | One          | No change to    |
:              : training,    :              : Workload     : `minCount`.     :
:              : Fine-Tuning,Fine-Tuning, :              : per          : Specify a       :
:              : or MPI jobs  :              : TrainJob.    : topology        :
:              :              :              : Set          : request.        :
:              :              :              : `minCount`   : Specify pod     :
:              :              :              : to the       : index label     :
:              :              :              : total        : for balancing   :
:              :              :              : number of    : of workers.     :
:              :              :              : pods in the  :                 :
:              :              :              : job or jobs  :                 :
:              :              :              :              :                 :
| LeaderWorkerSet | Large scale  | Yes, treat   | Yes. Set     | Set             |
: with fixed   : AI           : all groups   : ```          : ReplicaMode to  :
: `size` and   : Inference.   : as one       : minCount     : `ReplicatedGangs`. :
: `replicas`.  :              : gang.        : ``` to       :  Define one     :
: (No          :              :              : `size*replicas` : GangGroup. Set  :
: autoscaling). :              :              :              : `minCount` to   :
:              :              :              :              : that number of  :
:              :              :              :              : pods in that    :
:              :              :              :              : group.  May     :
:              :              :              :              : specify a       :
:              :              :              :              : topology level  :
:              :              :              :              : for all         :
:              :              :              :              : replicas, and   :
:              :              :              :              : another for     :
:              :              :              :              : each replica    :
:              :              :              :              : (LWS group).    :
:              :              :              :              : Can specify     :
:              :              :              :              : leader and      :
:              :              :              :              : worker size.    :
:              :              :              :              : Can specify     :
:              :              :              :              : indexing of     :
:              :              :              :              : leader+workers.leader+workers. :
:              :              :              :              :                 :
| LeaderWorkerSet | Large scale  | Not          | No, since    | See above.      |
: with fixed   : AI           : supported.   : gang         :                 :
: `size` and   : Inference    :              : scheduling   :                 :
: variable     : with         :              : of multiple  :                 :
: `replicas`.  : horizontal   :              : replica-gangs :                 :
:              : autoscaling.autoscaling. :              : is not       :                 :
:              :              :              : supported    :                 :
:              :              :              : yet.         :                 :

[world size]: https://stackoverflow.com/questions/58271635/in-distributed-computing-what-are-world-size-and-rank
### Gang Scheduling Implementation

This document only covers the API design so far.  It is intended to be followed by another document or be extended to describe the implementation in Kube-scheduler.  This is expected to rely on pods waiting at the permit stage for the entire gang to be ready. 

This should cover the following topics and more:

-  Define what Pod lifecycle state has to happen before "schedulerTimeoutSeconds" expires.
    -  not running to running?  
    -  having a nominatedNodeName?  a NodeName?
    -  Does this change the meaning of pod death/node failure (asked by @thockin).

-  Possibly have separate reachPermitTimeout and reachBindingTimeout, for alpha.
-  Define how any running pods are terminated when a timeout happens.
    -  With what signal or reason are existing pods killed, 
    -  with or without grace period (without? How?)

-  Define what happens to a job after it is _scheduled_ and then a pod terminates due to various reasons? Does kube-scheduler detect these conditions?  Does it proactively restart pods, or is this the controller's job?
    -  Pod has Evicted status (5 minutes after node failure)
    -  Pod self-terminates (e.g. due to peer time out).
    -  Pod terminated by controller (e.g. controller detects failure of all pods to reach readiness soon enough).
    -  Pod evicted doe to local resource pressure.
    -  Pod never reached running (e.g. imagepullbackoff).

-  If a controller creates a replacement pod (e.g. for a Torch Elastic job, or for a job with torchrun restarting it)
    -  Do we count running+pending pods towards quorum (numPods) to again unblock scheduling, or do all additional pods of this gang immediately try to schedule after the gang has passed quorum once?
    -  Policies
        -  try to place any pods after initial minCount were placed. (e.g. elastic workload).  If those go pending, that's fine.  Controller will notice that the remaining pods are pending a while and decide whether to restart scheduling by terminating existing ones. (Is this state in the Workload, or does any Running pods leave the workload in "okay" state.
        -  block any pods that show up later - assume controller will reap existing pods and create another minCount (it may be waiting for existing pods to exit (e.g. write emergency checkpoints) if it wants to get things going (e.g. JobSet does this apparently).
        -  Any pod in Failed or Unknown state cause all other pods in the gang to be evicted by scheduler (who wants this?)

-  Does scheduling/Workload inform eviction API (used by node draining) at Alpha?
    -  Signal coordinator first?  That is post-alpha.

-  Confirm that Node and Cluster Autoscaler do not need to use Workload at Alpha.
-  How Pod Phases interact with "phases" of a GangGroup: 
    -  Pending (including image pull)
    -  Running
    -  Succeeded
    -  Failed
    -  Unknown

## Documentation for Workloads

At Alpha, Gang Scheduling guarantees that that if:

-  The workload is using kube-scheduler as the scheduler
    -  possibly with Kueue setting a single-node selector per pod

-  The the pods have a Workload
-  The workload has a GangGroup
-  The pods and workload are being seen for the first time by kube-scheduler.

Then:

-  No pod will be enqueued in the active queue until its Workload has been observed too.
-  No pod will pass the Permit stage until all have reached Permit.

The time between when the first and the last pod reach Permit may be long for a number of reasons:

-  Insufficient Capacity – There might not be enough node capacity matching the requirements to hold all the pods.  More may never come.
-  Scale Up – Node capacity might be being provisioned by Cluster Autoscaler, but it could take some time. Pods with a CA-set nominatedNodeName still cannot pass Permit.
-  Preemption - One or more pods of the GangGroup could be waiting for preemption to complete to free capacity on a node. Pods with preemption-set nominatedNode still cannot pass permit. 

If there is too much delay for any of these reasons, then all pods time out an a requeued to try later. Because some resources are idle while waiting for all pods of a large gang to reach permit, using gang scheduling with CA and/or Preemption with long grace periods may be harmful to overall cluster throughput (goodput).  

Later implementations may be able to:

-  detect that resources are coming from ScaleUp or Preemption, and adjust the timeouts to align with the normal delay for these operations.
-  know when to prefer atomic scale up to holding existing resources while waiting for scale-up resources.
-  defer preemption of pods with different grace periods so they all terminate at about the same time.
-  backfill limited-duration pods onto nodes that are idle due to waiting for ScaleUp and Preemption

Even when all pods pass Permit at the same time, pods may not all reach running at the same time:

-  Pre-Bind and Bind: DRA drivers can run during the preBind or Bind stage.  (On-node Device Attachment).  This might take a long time(?), and it could differ by node.
-  After Binding:  Image pull can take a long time.  Not all nodes may pull images at the same rate.

Therefore, the application should be prepared to wait some time for all pods to come up initially.  Applications may want to set some application level timeout on initialization (this is typically done by frameworks, and may need adjustment due to the above reasons).

Once started, applications may detect pod failure quickly more quickly than Kubernetes can, since gang-scheduled groups often communicate very frequently, and the failure of one node would be detected quickly (as opposed to the 5 minutes it takes to confirm that a node is down from the control plane.

## Design of API-RevN

This section designs a more complete version of `Workload` that can evolve to to support topology aware functionality, and other potentially needed future functionality.

This is not intended to be approved as the final design.  Rather, having at least 1 possible future design, and aligning API-Rev1 with it, reduces the chance of being unable to iteratively evolve beyond API-Rev1,

### Clean-sheet Workload Model

The `Workload` needs to have a consistent structure which is flexible enough that it can model most true workloads accurately, yet simple enough that it does not introduce too much complexity into the already-complex scheduler codebase.

Several factors suggest doing a clean design of the workload model rather than directly copying the structure/concepts used by Kueue and/or Volcano.

-  Kueue is the most detailed model, supporting many workload types, and including rank-awareness.  It is the best starting point for designing API-RevN. We will likely need as many levels as it has.
-  The Kueue Workload model evolved organically.  The PodSet, PodSetGroup, and SubGroup concepts are perhaps not as clearly named as they could be.
-  Kueue handles replicated workloads as a set of plain pods with each replica having a Workload.  Due to the importance of Deployment overall in Kubernetes modeling them as a single Workload is desirable.

The rest of this section presents a clean design of the workload. It is based on [Kueue Workload](?tab=t.3zjbiyx2yldg#heading=h.xgv3th5u96yr), [Volcano PodGroup](?tab=t.3zjbiyx2yldg#heading=h.5zzx4jw9lmfn), and a study of [existing workloads](?tab=t.3zjbiyx2yldg#heading=h.156lanok18s8).

#### Entities

We define 6 entities.  These entities correspond to groupings of pods of which 3 new entities and use 3 existing concepts:

1. **True Workload** – A set of pods in the same true workload.  This is 1:1 with a `Workload`
1. **Replica** – Replicas in the sense of `Deployment` and `LeaderWorkerSet` –  any replica can be started or stopped with limited effect on other replicas. Replica can be an individual pod or a group of tightly coupled pods.
1. **Gang Group** – New! A Gang Group is a set of pods that need all-or-nothing scheduling. They need not be homogeneous or have the same pod template.
    1. Alternative name: **Pod Group**

1. **Ranked Group** – New!** **A Ranked Group is a set of pods numbered 0 to N. They correspond to ranks at the application level.
1. **Equivalence Group** – New!  Short for Equivalence Group.  A set of pods that are equivalent for scheduling (`Filter`) purposes. The shape (requirements) of the equivalent pods may be specified before the pod is created.
1. **Pod** – needs no explanation.

These concepts support the 4 main concepts that we see in workloads where pods run concurrently:

-  **Identical Replication** –  Replicas usually sit behind a load balancer and may be scaled up and down depending on load.
-  **All-or-nothing Scheduling** – Multiple pods start at once.
-  **Ranking or Indexing** – Pods having an identity for peer to peer communication.  The placement of these indexes matter for performance in some cases.
-  **Equivalent Pods** – Identifying pods that have the same requirements for scheduling allows faster and simpler scheduling algorithms for multiple pods.  Equivalent pods are not necessarily replicas, as they can have an index, and that index does not matter for many (not not all) steps of different scheduling algorithms. An Equiv Group can correspond to a Pod Template, but it does not have to be 1:1.

The order in which these entities are nested is important:

-  To support LeaderWorkerSet, Replica should be above Gang Group.
-  To support starting all replicated jobs of a JobSet at once, Gang Group should be above Ranked Group.
-  To support LeaderWorkerSet having different sized leader and worker pods that share an indexing range (e.g. leader is index 0, and workers are 1...n).  To support this, the Ranked Group should be above the Equiv Group.  

As Kueue's Workload shows, other formats are possible, but arguably less clear.

The nesting described above creates these relations between entities

-  A Pod is in an Equiv Group. 
-  An EquivGroup is in a Ranked Group.
    -  Rank Groups can optionally be unranked. 

-  A Ranked Group is in a Gang Group.
    -  A gang group can optionally not require gang scheduling.

-  Each Replica of the workload is a Gang Group.

With 4 levels, we can set policies, such as topology requests, at any of 5 levels.

With all this in mind, we could define `Workload` like this:

undefined

We can perfectly model all of the [workloads we evaluated in detail](?tab=t.3zjbiyx2yldg#heading=h.156lanok18s8) with this nesting.  We can add policy fields at each level if we need that level of control.

However, this is harder for users to write and understand when used with workloads types that don't need "fan out" at each level of nesting. We may be able to do better. 

#### Hierarchy Shapes

Many workloads don't need 1-to-N fan-out at each level. For example, some workloads don't need gang scheduling, and some don't have indexes, and some don't have replication.  It would be nice to be able to specify these more concisely and intuitively.

Additionally, some software frameworks create pods in shapes and numbers that are not known ahead of time. We would still like to be able to provide a basic `Workload` for these workloads in case there are some scheduling policies at the top level that can apply to them. 

Looking at existing workloads, we can group them into 5 common shapes:

| Shape name              | Characteristics         | Example true workloads  |
| ----------------------- | ----------------------- | ----------------------- |
| Replicated Pods         | Pods of one or more     |   ``` Deployment ```    |
:                         : shapes.  Each replica   : ``` ReplicaSet ```      :
:                         : is a single pod.        :                         :
| Ordered Pods            | Pods with indexes,      | `StatefulSet` with      |
:                         : which don't need gang   : `OrderedReady` policy   :
:                         : scheduling              :                         :
| Single Gang with        | Workload has one group  | `StatefulSet` with      |
: Multiple Rank Groups    : of pods needing gang    : `Parallel` policy       :
:                         : scheduling (Gang        : Indexed `Job`           :
:                         : Group).  There can be   :                         :
:                         : multiple indexed        :                         :
:                         : groups of pods.         :                         :
| Replicated Gangs        | Multiple replicas.      |   ``` LeaderWorkerSet   |
:                         : Each Replica is         : ```                     :
:                         : gang-scheduled.         :                         :
| Unspecified Shape       | A group of pods with    | Apache Airflow Argo     |
:                         : unspecified pod count,  : Workflows               :
:                         : size and indexing.      :                         :

This can be implemented using a tagged union.  To simplify iteration by clients, an `Expand` function will be defined to undo this collapsing into the canonical representation.  This is described more in the [Go Struct Definitions](?tab=t.0#heading=h.s6l87u3fep9a).

Workloads that are incompatible with gang scheduling, etc, are still allowed to have a `Workload` with an empty spec.  This allows enumeration of workloads (e.g. `kubectl get schedworkloads`) that chose to have a `Workload`.

### Design Alternatives for Workload Spec

#### Entity Naming Alternatives 

| Level of Workload       | Name Option 1           | Name Option 2           |
: grouping hierarchy      :                         :                         :
| ----------------------- | ----------------------- | ----------------------- |
| Level 0                 |   ``` Workload ```      |                         |
| Level 1                 |   ``` GangGroup ``` +   |   ``` PodGroup ``` +    |
:                         : Says that it can be     : Aligns with past gang   :
:                         : gang scheduled-         : proposals+ Aligns with  :
:                         : Confusing when gang     : Kueue and Volcano       :
:                         : mode not used.          : terms.                  :
| Level 2                 |   ``` RankedGroup ```   |   ``` PodSubGroup ```   |
:                         : + Says that it can be   : + Fits with             :
:                         : ranked (indexed).-      : `PodGroup`- Vague       :
:                         : Confusing when rank     : about purpose.          :
:                         : mode is not used.       :                         :
| Level 3                 |   ``` EqGroup ``` +     |   ``` PodSet ``` +      |
:                         : Says that pods in it    : Aligns with Kueue       :
:                         : are equivalent.         : term.- Not the same     :
:                         :                         : definition of           :
:                         :                         : equivalence as used by  :
:                         :                         : Kueue.+ Aligns with     :
:                         :                         : KubeFlow TrainJob term. :

Currently this doc uses the left column of names. This may be changed during Alpha development.  

Regardless of which names are used, there is a restriction on what properties of the workload shape can be defined at what level of the structure:

|    Level of  |    Link to   |    Gang      |    Group of  |    Pod          |
: Workload     : associated   : Scheduled    : pods ranked  : Templates       :
: grouping     : workload     : Group        : 0..N         :                 :
: hierarchy    :              :              :              :                 :
| ------------ | ------------ | ------------ | ------------ | --------------- |
|    Level 0   |    ✅         |              |              |                 |
: (`Workload`) :              :              :              :                 :
|    Level 1   |              |    ✅         |              |                 |
|    Level 2   |              |              |    ✅         |                 |
|    Level 3   |              |              |              |    ✅            |

Each of these pieces of information is optional, but restricted in where it can be specified.

#### Structural Alternatives 

Should the design of `Workload` favor readability and writeability for end users, or should it favor simplicity of the spec, and iteration over the spec by clients like kube-scheduler?

Readability and writeability can be improved by using **tagged unions** of different types representing different workload shapes, similar to how a Volume is a tagged union of many different volume types.

-  For example, a tagged union could include choices like:
    -  `ReplicatedGangWorkload` for LeaderWorkerSet-like workloads, 
    -  `SingleGangWorkload` for StatefulSet-like and IndexedJob-like workloads
    -  `ReplicatedNonGangWorkload` for Deployments and ReplicaSets

-  **Avoiding lists** when only one item is needed, using a tagged unions of the list and non-list version.

Concretely, given the following workload:

undefined

A spec that uses tagged-union aggressively might have YAML representation might look like this:

undefinedThis is based on [this tagged-union go type definition](https://docs.google.com/document/d/1ulO5eUnAsBWzqJdk_o5L-qdq5DIVwGcE7gWzCQ80SCM/edit?pli=1&tab=t.0#heading=h.s6l87u3fep9a).

A basic spec which is not optimized for readability/writeability might have a YAML representation might look like this:

undefinedThis is based on [this basic go type definition](https://docs.google.com/document/d/1ulO5eUnAsBWzqJdk_o5L-qdq5DIVwGcE7gWzCQ80SCM/edit?pli=1&tab=t.0#heading=h.9xos47u9pxd6).

##### Evaluation

| **Factors ↓      | **Option\\:**    | **Option\\:**    | **Weighting of     |
: Option →**       : **Basic Spec**   : **Use Tagged     : Factor  ↓**        :
:                  :                  : Unions for       :                    :
:                  :                  : Readability**    :                    :
| ---------------- | ---------------- | ---------------- | ------------------ |
| **Spec Length**  | 14 lines         | 11 lines         | **Low**.  _GenAI   |
:  _For the        :                  :                  : can help._ _Good   :
: indexed job      :                  :                  : samples can        :
: shown above,     :                  :                  : help._  _Note\:    :
: which has 26     :                  :                  : both options are   :
: lines._          :                  :                  : short compared to  :
:                  :                  :                  : the true workload  :
:                  :                  :                  : spec._             :
| **Complexity of  | Lower            | Higher           | **High.**          |
: WorkloadSpec **  : complexity.      : complexity.      : _Primary goal is   :
:  _Is the type    :                  :                  : supporting gang    :
: definition       :                  :                  : scheduling and     :
: correct, and     :                  :                  : later              :
: can a client     :                  :                  : topology-aware     :
: iterate over it  :                  :                  : scheduling.  _     :
: with concise     :                  :                  : _Scheduler is      :
: code?_           :                  :                  : already complex,   :
:                  :                  :                  : avoid adding more  :
:                  :                  :                  : complexity to      :
:                  :                  :                  : it._               :
| Summary          | ⭐ Best           |                  |                    |

**Decision - use Basic Spec option.**

### Policies

Policies are information that describes how to handle pods during scheduling.  This is in contrast to what was described above, which is about what pods the scheduler should expect to see.To configure gang scheduling and related features, we need to attach some properties to each entity. 

Types of policies can include:

-  Gang Scheduling-time Policies
    -  Number of pods that have to start at once.
    -  How long and hard to try to find a gang scheduling placement.

-  Gang Maintenance Policies
    -  What to do, if anything, in reaction to hardware failure or preemption of a gang.

-  Topology requests, such as:
        -  Pick any one value of node label key K for all pods in the Gang Group.
        -  ResourceClaim/ResourceClaimTemplate shared by all pods at one level.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            

    -  Topology requests and hints
        -  Same as Gang Group but scoped to Ranked Group. (Multi-level TAS)
        -  Label key for all pods in the Gang
        -  Hint that all pods in the Gang need to share a logical multi-host device pool.

    -  Rank placement policy
        -  Don't care, balance across topology label, stripe across topology label, etc.

    -  Request for Co-placement of same-rank pod from two rank groups. 
    -  Fault Tolerance policy

-  Equiv Group
    -  Number and spec of pods, to allow reserving resources, and faster cluster autoscaling.

These are the types of policies that we anticipate _may_ be attached to each level:

| Level  | Gang   | Dense  | Placement | Range  | Range  | SpreadSpread | Replica      |
: of     : Scheduling : Topology : policypolicy : of     : of     : Topology : Disruption   :
: Workload : Options : Request : based  : pod    : pod    : Request : Policy       :
: grouping :        :        : on     : sizes  : countscounts :        :              :
: hierarchy :        :        : rank   : (vertical : (horizontal :        :              :
:        :        :        :        : scaling : scaling :        :              :
:        :        :        :        : needs)needs) : needs)needs) :        :              :
:        :        :        :        :        :        :        :              :
| ------ | ------ | ------ | ------ | ------ | ------ | ------ | ------------ |
| Level  |        | ✅      |        |        | ✅      | ✅      | ✅            |
: 0      :        :        :        :        :        :        :              :
: (`Workload`) :        :        :        :        :        :        :              :
:        :        :        :        :        :        :        :              :
| Level  | ✅      | ✅      |        |        | ✅      |        |              |
: 1      :        :        :        :        :        :        :              :
| Level  |        | ✅      | ✅      |        | ✅      |        |              |
: 2      :        :        :        :        :        :        :              :
| Level  |        | ✅      |        | ✅      | ✅      |        |              |
: 3      :        :        :        :        :        :        :              :

### Pod Equivalence

Specifying a pod template ina an  `EqGroup` (Level 3) will be  optional but will allow clients (kube-scheduler and others) to make scheduling and queuing decisions for suspended and not-fully-scaled-up workloads since it allows clients to predict what type of pods will be made.  

The rules for whether two pods are equivalent for scheduling purposes will need to be spelled out in more detail when this feature is developed.  Pods that are not equivalent would need to be in separate Level 3 groupings, and those that are the same can be specified in the same grouping.

Some pod template fields that may not differ between pods in the same `EqGroup`:

-  different container resource request (cpu, mem, extended resources)
-  different resourceClaimsTemplates
-  different emptyDir volume specs
-  (maybe) different PVCs.

Some pod template fields that can differ without forcing pods to be in different groups.

-  labels
-  annotations
-  volumeMounts
-  configMap, secret, or downwardAPI types.
-  env
-  command

This is not a complete list.

### Go Struct Definitions 

This definition serves as our current best estimate of what Workload will evolve into after several iterations (feature gates).  An alternative structure was considered [here](?tab=t.xaykcbxr740t) but rejected. A third alternative is currently being explored [here](?tab=t.3pkx7y4zvho2). They are all close enough that defer decision between them to after API-Rev1 Alpha release.

undefined## Scope for Alpha

For Alpha, we will introduce the Rev-1 API. We will rely on the existing kube-scheduling algorithms and will purely rely on the WaitOnPermit mechanism to achieve gang semantics.  
We will implement as many of these modes as possible in alpha, starting with 1, as development time permits in Alpha:

1. Support a Workload with one non-replicated gang (e.g. IndexedJob)
1. Support a Workload with more than one non-replicated gang in a workload (e.g. JobSet with each replicatedJob a gang)
1. Support a Workload with one GangGroup replicated an arbitrary, unspecified number of times. (e.g. LeaderWorkerSet)

## Examples of Different Workload Specs

These use the Basic Version of API-RevN.

### Indexed Job

Indexed job running a training program.    
All pods form need to communicate (e.g. all-reduce).  Therefore they are in one GangGroup.    
Gang Policy is to Restart immediately if any pod fails.   
Topology request is to fit the entire job in one topology domain.   
Pods are ranked 0 to 99, so there is one RankedGroup.

| **True Workload**                    |   ``` Workload ```                   |
| ------------------------------------ | ------------------------------------ |
|   ```                                |   ```                                |
: apiVersion\\\\\\\\\\\\\\\\\\\\\\\\:  : apiVersion\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: batch/v1 ```   ```                   : scheduling/v1alpha1   ```   ```      :
: kind\\\\\\\\\\\\\\\\\\\\\\\: Job     : kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: ```   ```                            : Workload ```   ```                   :
: metadata\\\\\\\\\\\\\\\\\\\\\\: ```  : metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
:   ```   name\\\\\\\\\\\\\\\\\\\\\:   : ```   ```                            :
: job-1 ```   ```                      : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: spec\\\\\\\\\\\\\\\\\\\\: ```   ```  : w-job-1 ```   ```                    :
:   completions\\\\\\\\\\\\\\\\\\\:    : spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: 100 ```   ```                        : ```   ```                            :
: parallelism\\\\\\\\\\\\\\\\\\: 100   : controllerRef\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: ```   ```                            : ```   ```                            :
: completionMode\\\\\\\\\\\\\\\\\:     : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: Indexed ```   ```                    : job-1 ```   ```                      :
: template\\\\\\\\\\\\\\\\: ```   ```  : kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
:     spec\\\\\\\\\\\\\\\: ```   ```   : Job ```   ```                        :
:      restartPolicy\\\\\\\\\\\\\\:    : apiGroup\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: OnFailure ```   ```                  : batch ```   ```                      :
: containers\\\\\\\\\\\\\: ```   ```   : topologySpread\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
:      - name\\\\\\\\\\\\: ml-worker   : ```   ```     .... ```   ```         :
: ```   ```         image\\\\\\\\\\\:  : topologyRequest\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: awesome-training-program\\\\\\\\\\\:v1 : ```   ```                            :
:  ```   ```                           : resourceClaims\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: command\\\\\\\\\\: ["python",        :  ```   ```        -                  :
: "train.py"] ```   ```                : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:  :
: resources\\\\\\\\\: ```   ```        : ```   ```                            :
:     limits\\\\\\\\: ```   ```        : resourceClaimTemplate\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
:       nvidia.com/gpu\\\\\\\: 1 ```   : any_network_block ```   ```          :
:  ```         env\\\\\\: ```   ```    : replicaMode\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
:       - name\\\\\:                   : Unreplicated ```   ```               :
: JOB_COMPLETION_INDEX ```   ```       : gangGroups\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
:      valueFrom\\\\: ```   ```        :  ```   ```     -                     :
:       fieldRef\\\: ```   ```         : name\\\\\\\\\\\\\\\\\\\\\\\\\\:      :
:        fieldPath\\:                  : "gg" ```   ```                       :
: "metadata.annotations['batch.kubernetes.io/job-completion-index']" : minCount\\\\\\\\\\\\\\\\\\\\\\\\\:   :
: ```                                  : 100 ```   ```                        :
:                                      : maxCount\\\\\\\\\\\\\\\\\\\\\\\\:    :
:                                      : 100 ```   ```                        :
:                                      : scheduleTimeoutSeconds\\\\\\\\\\\\\\\\\\\\\\\: :
:                                      : 60 ```   ```                         :
:                                      : reschedulingPolicy\\\\\\\\\\\\\\\\\\\\\\: :
:                                      : TerminateAll ```   ```               :
:                                      : rankedGroups\\\\\\\\\\\\\\\\\\\\\:   :
:                                      : ```   ```         -                  :
:                                      : name\\\\\\\\\\\\\\\\\\\\: "rg" ```   :
:                                      :  ```                                 :
:                                      : podRank\\\\\\\\\\\\\\\\\\\: ```      :
:                                      : ```                                  :
:                                      : valueFrom\\\\\\\\\\\\\\\\\\: ```     :
:                                      : ```                                  :
:                                      : fieldRef\\\\\\\\\\\\\\\\\: ```       :
:                                      : ```                                  :
:                                      : fieldPath\\\\\\\\\\\\\\\\:           :
:                                      : "metadata.annotations['batch.kubernetes.io/job-completion-index']" :
:                                      : ```   ```                            :
:                                      : topologyRequest\\\\\\\\\\\\\\\: ```  :
:                                      :   ```                                :
:                                      : resourceClaims\\\\\\\\\\\\\\:  ```   :
:                                      :  ```        - name\\\\\\\\\\\\\:     :
:                                      : ```   ```                            :
:                                      : resourceClaim\\\\\\\\\\\\:           :
:                                      : any_network_subblock ```   ```       :
:                                      :    - name\\\\\\\\\\\: "rg2" ```      :
:                                      : ```           podRank\\\\\\\\\\:     :
:                                      : ```   ```                            :
:                                      : valueFrom\\\\\\\\\: ```   ```        :
:                                      :         fieldRef\\\\\\\\: ```   ```  :
:                                      :                 fieldPath\\\\\\\:    :
:                                      : "metadata.annotations['batch.kubernetes.io/job-completion-index']" :
:                                      : ```   ```                            :
:                                      : topologyRequest\\\\\\: ```   ```     :
:                                      :    resourceClaims\\\\\:  ```   ```   :
:                                      :       - name\\\\: ```   ```          :
:                                      :  resourceClaim\\\:                   :
:                                      : any_network_subblock ```             :

### Un-indexed Job

Unindexed job running a bag-of-tasks.    
No direct runtime dependency between tasks, so no gang scheduling used.  No gang scheduling used.    Entire job requested to fit in one topology domain (they don't communicate peer to peer directly. but suppose that it improves storage system performance).  
Topology request is to fit the entire job in one topology domain.   
Pods are ranked 0 to 99, so there is one RankedGroup.

| **True Workload**                    |   ``` Workload ```                   |
| ------------------------------------ | ------------------------------------ |
|   ```                                |   ``` apiVersion\\\\\\\\\\\\\\:      |
: apiVersion\\\\\\\\\\\\\\\\\\\\:      : scheduling/v1alpha1   ```   ```      :
: batch/v1 ```   ```                   : kind\\\\\\\\\\\\\: Workload ```      :
: kind\\\\\\\\\\\\\\\\\\\: Job ```     : ``` metadata\\\\\\\\\\\\: ```   ```  :
: ``` metadata\\\\\\\\\\\\\\\\\\: ```  :   name\\\\\\\\\\\: w-job-2 ```       :
:   ```   name\\\\\\\\\\\\\\\\\:       : ``` spec\\\\\\\\\\: ```   ```        :
: job-2 ```   ```                      : controllerRef\\\\\\\\\: ```   ```    :
: spec\\\\\\\\\\\\\\\\: ```   ```      :   name\\\\\\\\: job-2 ```   ```      :
: completions\\\\\\\\\\\\\\\: 100 ```  : kind\\\\\\\: Job ```   ```           :
:   ```   parallelism\\\\\\\\\\\\\\:   : apiGroup\\\\\\: batch ```   ```      :
: 50 ```   ```                         : topologyRequest\\\\\: <details TBD>  :
: completionMode\\\\\\\\\\\\\:         : ```   ```   replicaMode\\\\:         :
: NonIndexed ```   ```                 : Unreplicated ```   ```               :
: template\\\\\\\\\\\\: ```   ```      : gangGroups\\\:  ```   ```     -      :
: spec\\\\\\\\\\\: ```   ```           : name\\: "gg" ```   ```               :
: restartPolicy\\\\\\\\\\: OnFailure   : gangMode\: Off ```                   :
: ```   ```                            :                                      :
: containers\\\\\\\\\: ```   ```       :                                      :
:  - name\\\\\\\\:                     :                                      :
: conditioning-worker ```   ```        :                                      :
:   image\\\\\\\:                      :                                      :
: data-conditioning\\\\\\\:v1  ```     :                                      :
: ```         command\\\\\\:           :                                      :
: ["python", "condition.py"] ```       :                                      :
: ```         resources\\\\\: ```      :                                      :
: ```           limits\\\\: ```   ```  :                                      :
:             memory\\\: 50GB ```      :                                      :
: ```             cpu\\: 16000 ```     :                                      :

### Parallel-starting Fixed-size StatefulSet

Indexed StatefulSet running a training program.  World size does not change at runtime.    
All pods need to communicate (e.g. all-reduce).  Therefore they are in one GangGroup.    
Gang Policy is to Restart immediately if any pod fails.   
Topology request is to fit the entire job in one topology domain.   
Pods are ranked 0 to 3, so there is one RankedGroup.

| **True Workload**                    |   ``` Workload ```                   |
| ------------------------------------ | ------------------------------------ |
|   ```                                |   ```                                |
: apiVersion\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : apiVersion\\\\\\\\\\\\\\\\\\\\\\\\:  :
: apps/v1 ```   ```                    : scheduling/v1alpha1   ```   ```      :
: kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : kind\\\\\\\\\\\\\\\\\\\\\\\:         :
: StatefulSet ```   ```                : Workload ```   ```                   :
: metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : metadata\\\\\\\\\\\\\\\\\\\\\\: ```  :
: ```   ```                            :   ```   name\\\\\\\\\\\\\\\\\\\\\:   :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : w-ss-1 ```   ```                     :
: ss-1 ```   ```                       : spec\\\\\\\\\\\\\\\\\\\\: ```   ```  :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :   controllerRef\\\\\\\\\\\\\\\\\\\:  :
: ```   ```                            : ```   ```                            :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : name\\\\\\\\\\\\\\\\\\: s-1 ```      :
: ```   ```                            : ```     kind\\\\\\\\\\\\\\\\\:       :
: serviceName\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : StatefulSet ```   ```                :
: "ss-1" ```   ```                     : apiGroup\\\\\\\\\\\\\\\\: apps ```   :
: replicas\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :  ```                                 :
: 4 ```   ```                          : topologyRequest\\\\\\\\\\\\\\\:      :
: selector\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : <details TBD> ```   ```              :
: ```   ```                            : replicaMode\\\\\\\\\\\\\\:           :
: matchLabels\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : Unreplicated ```   ```               :
: ```   ```                            : gangGroups\\\\\\\\\\\\\:  ```   ```  :
: app\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :     - name\\\\\\\\\\\\: "gg" ```     :
: ss-1 ```   ```                       : ```       minCount\\\\\\\\\\\: 4     :
: podManagementPolicy\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```       maxCount\\\\\\\\\\:  :
: "Parallel" ```   ```                 : 4 ```   ```                          :
: template\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : scheduleTimeoutSeconds\\\\\\\\\: 60  :
: ```   ```                            : ```   ```                            :
: metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : reschedulingPolicy\\\\\\\\:          :
: ```   ```                            : TerminateAll ```   ```               :
: labels\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : rankedGroups\\\\\\\: ```   ```       :
: ```   ```                            :    - name\\\\\\: "rg" ```   ```      :
: app\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :       podRank\\\\\: ```   ```        :
: ss-1 ```   ```                       :       valueFrom\\\\: ```   ```       :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :          fieldRef\\\: ```   ```      :
: ```   ```                            :             fieldPath\\:             :
: containers\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : "metadata.annotations['apps.kubernetes.io/pod-index']" :
: ```   ```       -                    : ```                                  :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:  :                                      :
: pytorch-trainer-container ```   ```  :                                      :
:                                      :                                      :
: image\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:  :                                      :
: my-pytorch-image\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:v1 :                                      :
:  ```   ```                           :                                      :
: command\\\\\\\\\\\\\\\\\\\\\\\\\\\\:command\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ["/bin/bash", "-c"] ```   ```        :                                      :
:   args\\\\\\\\\\\\\\\\\\\\\\\\\\\:   :                                      :
: ```   ```           -                :                                      :
: \\\\\\\\\\\\\\\\\\\\\\\\\\| ```      :                                      :
: ```             torchrun \ ```       :                                      :
: ```                                  :                                      :
: --nproc_per_node=1 \ ```   ```       :                                      :
:          --nnodes=$(REPLICAS) ```    :                                      :
: ```                                  :                                      :
: --node_rank=$(POD_INDEX)  ```   ```  :                                      :
:                                      :                                      :
: --master_addr=${LEADER_ADDR} ```     :                                      :
: ```                                  :                                      :
: --master_port=23456 \ ```   ```      :                                      :
:           /app/train.py  ```   ```   :                                      :
:        env\\\\\\\\\\\\\\\\\\:  ```   :                                      :
:  ```         -                       :                                      :
: name\\\\\\\\\\\\\\\\\: POD_NAME ```  :                                      :
:   ```                                :                                      :
: valueFrom\\\\\\\\\\\\\\\\: ```       :                                      :
: ```                                  :                                      :
: fieldRef\\\\\\\\\\\\\\\: ```   ```   :                                      :
:                                      :                                      :
: fieldPath\\\\\\\\\\\\\\:             :                                      :
: metadata.name ```   ```         -    :                                      :
: name\\\\\\\\\\\\\: POD_INDEX ```     :                                      :
: ```                                  :                                      :
: valueFrom\\\\\\\\\\\\: ```   ```     :                                      :
:          fieldRef\\\\\\\\\\\: ```    :                                      :
: ```                                  :                                      :
: fieldPath\\\\\\\\\\:                 :                                      :
: metadata.labels['apps.kubernetes.io/pod-index'] :                                      :
: ```   ```         - name\\\\\\\\\:   :                                      :
: LEADER_ADDR            ```   ```     :                                      :
:        value\\\\\\\\:                :                                      :
: "0.ss-1.ns.svc.cluster.local"        :                                      :
:      ```   ```                       :                                      :
: volumeMounts\\\\\\\: ```   ```       :                                      :
:    - name\\\\\\:                     :                                      :
: training-data-volume ```   ```       :                                      :
:      mountPath\\\\\: /data ```       :                                      :
: ```         - name\\\\:              :                                      :
: checkpoints-volume ```   ```         :                                      :
:    mountPath\\\: /checkpoints ```    :                                      :
: ```       volumes\\: <...> ```       :                                      :

### In-order-starting Variable-size StatefulSet

This StatefulSet is running a database. It uses InOrder startup.  The number of pods is variable, so replica mode  is set to ReplicatedPods.

Pods need to be scaled up and down in order, so gang scheduling cannot be used.

The pods are ranked, but the ranks are not relevant for scheduling, so rankedMode is set to off.

All pods currently have the same size and shape for scheduling feasibility purposes. Therefore, one EqGroup is sufficient.  

In this example, we suppose that vertical autoscaling is used, and that Workload has a way to express a maximum expected size for vertical autoscaled pods (this is a hypothetical use case that is not explored in detail in this design doc).

| **True Workload**                    |   ``` Workload ```                   |
| ------------------------------------ | ------------------------------------ |
|   ```                                |   ```                                |
: apiVersion\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : apiVersion\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: apps/v1 ```   ```                    : scheduling/v1alpha1   ```   ```      :
: kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: StatefulSet ```   ```                : Workload ```   ```                   :
: metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: ```   ```                            : ```   ```                            :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: ss-2 ```   ```                       : w-ss-1 ```   ```                     :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: ```   ```                            : ```   ```                            :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : controllerRef\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: ```   ```                            : ```   ```                            :
: serviceName\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: "ss-2" ```   ```                     : s-1 ```   ```                        :
: replicas\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:  :
: 3 ```   ```                          : StatefulSet ```   ```                :
: selector\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : apiGroup\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: ```   ```                            : apps ```   ```                       :
: matchLabels\\\\\\\\\\\\\\\\\\\\\\\\\\\: : topologyRequest\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: ```   ```                            : <details TBD> ```   ```              :
: app\\\\\\\\\\\\\\\\\\\\\\\\\\: ss-1  : replicaMode\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: ```   ```                            : ReplicatedPods ```   ```             :
: podManagementPolicy\\\\\\\\\\\\\\\\\\\\\\\\\: : gangGroups\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: "InOrder" ```   ```                  :  ```   ```     -                     :
: template\\\\\\\\\\\\\\\\\\\\\\\\:    : name\\\\\\\\\\\\\\\\\\\\\\\\\: "gg"  :
: ```   ```                            : ```   ```                            :
: metadata\\\\\\\\\\\\\\\\\\\\\\\:     : gangMode\\\\\\\\\\\\\\\\\\\\\\\\:    :
: ```   ```                            : Off ```   ```                        :
: labels\\\\\\\\\\\\\\\\\\\\\\: ```    : rankedGroups\\\\\\\\\\\\\\\\\\\\\\\:rankedGroups\\\\\\\\\\\\\\\\\\\\\\\: :
: ```                                  : ```   ```         -                  :
: app\\\\\\\\\\\\\\\\\\\\\: ss-2 ```   : name\\\\\\\\\\\\\\\\\\\\\\: "rg"     :
:  ```     spec\\\\\\\\\\\\\\\\\\\\:   : ```   ```                            :
: ```   ```                            : rankMode\\\\\\\\\\\\\\\\\\\\\: Off   :
: terminationGracePeriodSeconds\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: 60 ```   ```                         : eqGroups\\\\\\\\\\\\\\\\\\\\: ```    :
: containers\\\\\\\\\\\\\\\\\\: ```    : ```           -                      :
: ```        - name\\\\\\\\\\\\\\\\\:  : name\\\\\\\\\\\\\\\\\\\: "template"  :
: "postgres" ```   ```                 : ```   ```                            :
: image\\\\\\\\\\\\\\\\:               : count\\\\\\\\\\\\\\\\\\: 3 ```       :
: my-postgres\\\\\\\\\\\\\\\\:v1 ```   : ```                                  :
:  ```                                 : template\\\\\\\\\\\\\\\\\:           :
: resources\\\\\\\\\\\\\\\:  ```       :       ```   ```                      :
: ```                                  : resources\\\\\\\\\\\\\\\\:  ```      :
: requests\\\\\\\\\\\\\\: ```   ```    : ```                                  :
:            cpu\\\\\\\\\\\\\: 10000   : requests\\\\\\\\\\\\\\\: ```   ```   :
: ```   ```                            :                 cpu\\\\\\\\\\\\\\:   :
: memory\\\\\\\\\\\\: 100GB ```   ```  : 10000 ```   ```                      :
:            limits\\\\\\\\\\\: ```    : memory\\\\\\\\\\\\\: 100GB ```       :
: ```              cpu\\\\\\\\\\:      : ```                                  :
: 10000 ```   ```                      : limits\\\\\\\\\\\\: ```   ```        :
: memory\\\\\\\\\: 100GB ```   ```     :            cpu\\\\\\\\\\\: 10000     :
:       ports\\\\\\\\: ```   ```       : ```   ```                            :
:        - name\\\\\\\: postgres ```   : memory\\\\\\\\\\: 100GB ```   ```    :
:  ```                                 :           maxTemplate\\\\\\\\\: ```  :
: containerPort\\\\\\: 5432 ```   ```  :   ```                                :
:               protocol\\\\\: TCP     : resources\\\\\\\\:  ```   ```        :
: ```   ```         command\\\\: ...   :          requests\\\\\\\: ```   ```  :
: ```   ```         args\\\: ... ```   :                  cpu\\\\\\: 16000    :
:  ```       volumeMounts\\: ... ```   : ```   ```                            :
:  ```       volumes\: ... ```         : memory\\\\\: 200GB ```   ```         :
:                                      :         limits\\\\: ```   ```        :
:                                      :            cpu\\\: 16000 ```   ```   :
:                                      :                 memory\\: 200GB      :
:                                      :    ```                               :

### Kubeflow TrainJob

Kubeflow TrainJob is a v2 that supports both PyTorchJob and MPIJob. It is defined [here](https://github.com/kubeflow/trainer/blob/b9f06020d6a0c1911964d618548f261e3437c73b/pkg/apis/trainer/v1alpha1/trainjob_types.go#L26) and documented [here](https://www.kubeflow.org/docs/components/trainer/operator-guides/runtime/).

A user creates a `TrainJob` which takes default values from a `ClusterTrainingRuntime`.  The reference is from the Workload to the TrainJob, since those have similar lifecycles.

A `spec.podGroupPolicy.coreWorkload` is imagined to be supported by Kubeflow Trainer, and trainer can generate the `Workload` object.

| **True Workload**                    |   ``` Workload ```                   |
| ------------------------------------ | ------------------------------------ |
|   ```                                |   ```                                |
: apiVersion\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : apiVersion\\\\\\\\\\\\\\\\\\\\\\\\:  :
: trainer.kubeflow.org/v1alpha1 ```    : scheduling/v1alpha1   ```   ```      :
: ```                                  : kind\\\\\\\\\\\\\\\\\\\\\\\:         :
: kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : Workload ```   ```                   :
: TrainJob ```   ```                   : metadata\\\\\\\\\\\\\\\\\\\\\\: ```  :
: metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :   ```   name\\\\\\\\\\\\\\\\\\\\\:   :
: ```   ```                            : w-tjob-1 ```   ```                   :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : spec\\\\\\\\\\\\\\\\\\\\: ```   ```  :
: example-train-job ```   ```          :   controllerRef\\\\\\\\\\\\\\\\\\\:  :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: ```   ```                            : name\\\\\\\\\\\\\\\\\\:              :
: runtimeRef\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : example-train-job ```   ```          :
: ```   ```                            : kind\\\\\\\\\\\\\\\\\: TrainJob ```  :
: apiGroup\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :   ```     apiGroup\\\\\\\\\\\\\\\\:  :
: trainer.kubeflow.org ```   ```       : trainer.kubeflow.org ```   ```       :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : topologyRequest\\\\\\\\\\\\\\\:      :
: torch-distributed ```   ```          : <details TBD> ```   ```              :
: kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:  : replicaMode\\\\\\\\\\\\\\:           :
: ClusterTrainingRuntime ```   ```     : Unreplicated ```   ```               :
: --- ```   ```                        : gangGroups\\\\\\\\\\\\\:  ```   ```  :
: apiVersion\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :     - name\\\\\\\\\\\\: "gg" ```     :
: trainer.kubeflow.org/v1alpha1 ```    : ```       minCount\\\\\\\\\\\: 18    :
: ```                                  : ```   ```       maxCount\\\\\\\\\\:  :
: kind\\\\\\\\\\\\\\\\\\\\\\\\\\\:     : 18 ```   ```                         :
: ClusterTrainingRuntime ```   ```     : scheduleTimeoutSeconds\\\\\\\\\: 60  :
: metadata\\\\\\\\\\\\\\\\\\\\\\\\\\:  : ```   ```                            :
: ```   ```                            : reschedulingPolicy\\\\\\\\:          :
: name\\\\\\\\\\\\\\\\\\\\\\\\\:       : TerminateAll ```   ```               :
: torch-distributed ```   ```          : rankedGroups\\\\\\\: ```   ```       :
: labels\\\\\\\\\\\\\\\\\\\\\\\\: ```  :    - name\\\\\\: "node" ```   ```    :
:   ```                                :         podRank\\\\\: ```   ```      :
: trainer.kubeflow.org/framework\\\\\\\\\\\\\\\\\\\\\\\: :         valueFrom\\\\: ```   ```     :
: torch ```   ```                      :            fieldRef\\\: ```   ```    :
: spec\\\\\\\\\\\\\\\\\\\\\\: ```      :               fieldPath\\:           :
: ```                                  : "metadata.labels['batch.kubernetes.io/job-completion-index']" :
: mlPolicy\\\\\\\\\\\\\\\\\\\\\: ```   : ```                                  :
:  ```                                 :                                      :
: numNodes\\\\\\\\\\\\\\\\\\\\: 18     :                                      :
: ```   ```                            :                                      :
: torch\\\\\\\\\\\\\\\\\\\: ```   ```  :                                      :
:                                      :                                      :
: numProcPerNode\\\\\\\\\\\\\\\\\\:    :                                      :
: auto ```   ```                       :                                      :
: podGroupPolicy\\\\\\\\\\\\\\\\\:     :                                      :
: ```   ```                            :                                      :
: coreWorkload\\\\\\\\\\\\\\\\: ```    :                                      :
: ```                                  :                                      :
: scheduleTimeoutSeconds\\\\\\\\\\\\\\\: :                                      :
: 60 ```   ```                         :                                      :
: template\\\\\\\\\\\\\\: ```   ```    :                                      :
:   spec\\\\\\\\\\\\\: ```   ```       :                                      :
:  replicatedJobs\\\\\\\\\\\\: ```     :                                      :
: ```         - name\\\\\\\\\\\: node  :                                      :
: ```   ```                            :                                      :
: template\\\\\\\\\\: ```   ```        :                                      :
:       metadata\\\\\\\\\: ```   ```   :                                      :
:              labels\\\\\\\\: ```     :                                      :
: ```                                  :                                      :
: trainer.kubeflow.org/trainjob-ancestor-step\\\\\\\: :                                      :
: trainer ```   ```                    :                                      :
: spec\\\\\\: ```   ```                :                                      :
: template\\\\\: ```   ```             :                                      :
:      spec\\\\: ```   ```             :                                      :
:        containers\\\: ```   ```      :                                      :
:                 - name\\: node ```   :                                      :
:  ```                       image\:   :                                      :
: pytorch/pytorch\:2.7.1-cuda12.8-cudnn9-runtime :                                      :
: ```                                  :                                      :
|   ```                                |   ```                                |
: apiVersion\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : apiVersion\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: trainer.kubeflow.org/v1alpha1 ```    : scheduling/v1alpha1   ```   ```      :
: ```                                  : kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : Workload ```   ```                   :
: TrainJob ```   ```                   : metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: ```   ```                            : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : w-tjob-mpi ```   ```                 :
: trainjob-with-mpi ```   ```          : spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: ```   ```                            : controllerRef\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: runtimeRef\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: ```   ```                            : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: apiGroup\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : example-train-job ```   ```          :
: trainer.kubeflow.org ```   ```       : kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : TrainJob ```   ```                   :
: deepspeed-distributed ```   ```      : apiGroup\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : trainer.kubeflow.org ```   ```       :
: ClusterTrainingRuntime ```   ```     : topologyRequest\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: --- ```   ```                        : <details TBD> ```   ```              :
: apiVersion\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : replicaMode\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: trainer.kubeflow.org/v1alpha1 ```    : Unreplicated ```   ```               :
: ```                                  : gangGroups\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :  ```   ```     -                     :
: ClusterTrainingRuntime ```   ```     : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : dataset-initializer ```   ```        :
: ```   ```                            : minCount\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : 1 ```   ```                          :
: deepspeed-distributed ```   ```      : maxCount\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: labels\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : 1 ```   ```                          :
: ```   ```                            : scheduleTimeoutSeconds\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: trainer.kubeflow.org/framework\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : 60 ```   ```                         :
: deepspeed ```   ```                  : reschedulingPolicy\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : TerminateAll ```   ```               :
: ```   ```                            : rankMode\\\\\\\\\\\\\\\\\\\\\\\\\\\:rankMode\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: mlPolicy\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : Off ```   ```                        :
: ```   ```                            : eqGroups\\\\\\\\\\\\\\\\\\\\\\\\\\:  :
: numNodes\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```         -                  :
: 10  # number of kubernetes pods ```  : name\\\\\\\\\\\\\\\\\\\\\\\\\:       :
:   ```                                : dataset-initializer ```   ```        :
: mpi\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            : template\\\\\\\\\\\\\\\\\\\\\\\\:    :
: numProcPerNode\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: 2   # number of GPUs per pod ```     : resources\\\\\\\\\\\\\\\\\\\\\\\:    :
: ```                                  : ```   ```                            :
: mpiImplementation\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : limits\\\\\\\\\\\\\\\\\\\\\\: ```    :
: OpenMPI ```   ```                    : ```                                  :
: sshAuthMountPath\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : cpu\\\\\\\\\\\\\\\\\\\\\: 5 ```      :
: /home/mpiuser/.ssh ```   ```         : ```                                  :
: runLauncherAsNode\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : memory\\\\\\\\\\\\\\\\\\\\: 20Gi     :
: true ```   ```                       : ```   ```                            :
: podGroupPolicy\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : spec\\\\\\\\\\\\\\\\\\\: ```   ```   :
: ```   ```                            :                                      :
: coreWorkload\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : containers\\\\\\\\\\\\\\\\\\: ```    :
: ```   ```                            : ```                 -                :
: scheduleTimeoutSeconds\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : name\\\\\\\\\\\\\\\\\:               :
: 60 ```   ```                         : dataset-initializer ```   ```     -  :
: template\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : name\\\\\\\\\\\\\\\\: trainer ```    :
: ```   ```                            : ```       minCount\\\\\\\\\\\\\\\:   :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : 10 ```   ```                         :
: ```   ```                            : maxCount\\\\\\\\\\\\\\: 10 ```       :
: network\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```                                  :
: ```   ```                            : scheduleTimeoutSeconds\\\\\\\\\\\\\:scheduleTimeoutSeconds\\\\\\\\\\\\\: :
: publishNotReadyAddresses\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : 60 ```   ```                         :
: true ```   ```                       : reschedulingPolicy\\\\\\\\\\\\:      :
: successPolicy\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : TerminateAll ```   ```               :
: ```   ```                            : rankMode\\\\\\\\\\\: Off ```   ```   :
: operator\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :      eqGroups\\\\\\\\\\: ```   ```   :
: All ```   ```                        :        - name\\\\\\\\\: node ```     :
: targetReplicatedJobs\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```           template\\\\\\\\: ```  :
: ```   ```           - launcher ```   :   ```             resources\\\\\\\:  :
:  ```                                 : ```   ```                            :
: replicatedJobs\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : limits\\\\\\: ```   ```              :
: ```   ```         -                  :     nvidia.com/gpu\\\\\: 2 ```       :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```             spec\\\\: ```   ```  :
: dataset-initializer ```   ```        :               containers\\\: ```     :
:                                      : ```                 - name\\: node   :
: template\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```                                  :
: ```   ```                            :                                      :
: metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: labels\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: trainer.kubeflow.org/trainjob-ancestor-step\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: dataset-initializer ```   ```        :                                      :
:                                      :                                      :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: template\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: containers\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                     -      :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: dataset-initializer ```   ```        :                                      :
:                                      :                                      :
: resources\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: limits\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: cpu\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 5 ```   ```                          :                                      :
:                                      :                                      :
: memory\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 20Gi ```   ```                       :                                      :
:                                      :                                      :
: image\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ghcr.io/kubeflow/trainer/dataset-initializer :                                      :
: ```   ```                            :                                      :
: volumeMounts\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                       -    :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: initializer ```   ```                :                                      :
:                                      :                                      :
: mountPath\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: /workspace ```   ```                 :                                      :
:                                      :                                      :
: volumes\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                     -      :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: initializer ```   ```                :                                      :
:                                      :                                      :
: persistentVolumeClaim\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: claimName\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: initializer ```   ```         -      :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: launcher ```   ```                   :                                      :
: dependsOn\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```             -              :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: dataset-initializer ```   ```        :                                      :
:                                      :                                      :
: status\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: Complete ```   ```                   :                                      :
: template\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: labels\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: trainer.kubeflow.org/trainjob-ancestor-step\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: trainer ```   ```                    :                                      :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: template\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: containers\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                     -      :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: node ```   ```                       :                                      :
:                                      :                                      :
: resources\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: limits\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: nvidia.com/gpu\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 2 ```   ```                          :                                      :
: image\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ghcr.io/kubeflow/trainer/deepspeed-runtime :                                      :
: ```   ```                            :                                      :
: securityContext\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: runAsUser\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 1000 ```   ```                       :                                      :
:                                      :                                      :
: volumeMounts\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                       -    :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: initializer ```   ```                :                                      :
:                                      :                                      :
: mountPath\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: /workspace ```   ```                 :                                      :
:                                      :                                      :
: volumes\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                     -      :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: initializer ```   ```                :                                      :
:                                      :                                      :
: persistentVolumeClaim\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: claimName\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: initializer ```   ```         -      :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: node ```   ```                       :                                      :
: dependsOn\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```             -              :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: dataset-initializer ```   ```        :                                      :
:                                      :                                      :
: status\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: Complete ```   ```                   :                                      :
: template\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:  :                                      :
: ```   ```                            :                                      :
: template\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\:    :                                      :
: ```   ```                            :                                      :
: containers\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                     -      :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\:      :                                      :
: node ```   ```                       :                                      :
:                                      :                                      :
: resources\\\\\\\\\\\\\\\\\\\\\\\\\:  :                                      :
: ```   ```                            :                                      :
: limits\\\\\\\\\\\\\\\\\\\\\\\\: ```  :                                      :
:   ```                                :                                      :
: nvidia.com/gpu\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 2 ```   ```                          :                                      :
: image\\\\\\\\\\\\\\\\\\\\\\:         :                                      :
: ghcr.io/kubeflow/trainer/deepspeed-runtime :                                      :
: ```   ```                            :                                      :
: securityContext\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: runAsUser\\\\\\\\\\\\\\\\\\\\: 1000  :                                      :
: ```   ```                            :                                      :
: command\\\\\\\\\\\\\\\\\\\: ```      :                                      :
: ```                         -        :                                      :
: /usr/sbin/sshd ```   ```             :                                      :
:            args\\\\\\\\\\\\\\\\\:    :                                      :
: ```   ```                         -  :                                      :
: -De ```   ```                        :                                      :
:   - -f ```   ```                     :                                      :
:      - /home/mpiuser/.sshd_config    :                                      :
: ```   ```                            :                                      :
: readinessProbe\\\\\\\\\\\\\: ```     :                                      :
: ```                                  :                                      :
: tcpSocket\\\\\\\\\\\\: ```   ```     :                                      :
:                                      :                                      :
: port\\\\\\\\\\\: 2222 ```   ```      :                                      :
:                                      :                                      :
: initialDelaySeconds\\\\\\\\\\: 5     :                                      :
: ```   ```                            :                                      :
: volumeMounts\\\\\\\\\: ```   ```     :                                      :
:                    - name\\\\\\\\:   :                                      :
: initializer ```   ```                :                                      :
:           mountPath\\\\\\\:          :                                      :
: /workspace ```   ```                 :                                      :
:    volumes\\\\\\: ```   ```          :                                      :
:             - name\\\\\:             :                                      :
: initializer ```   ```                :                                      :
:         persistentVolumeClaim\\\\:   :                                      :
: ```   ```                            :                                      :
: claimName\\\: initializer ```        :                                      :

### Deployment for ML Serving

Deployment where each pod serves the same model.  Taken from [this example](https://cloud.google.com/kubernetes-engine/docs/tutorials/scalable-ml-models-torchserve). It uses HPA to scale the number of replicas based on a custom metric.

A replica that HPA adds is one Pod in this case. Therefore, replicaMode is set to `ReplicatedPods`.

In this example, minCount and maxCount are copied from minReplicas and maxReplicas of the HPA.

Alternatively, minCount, maxCount, could be left unspecified.  In this case, the scheduler has less information to work with, but updates to the workload by HPA and by rolling updates don't have to be reflected in the `Workload`.  
 

|    **True Workload**                 |   ``` Workload ```                   |
| ------------------------------------ | ------------------------------------ |
|   ```                                |   ```                                |
: apiVersion\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : apiVersion\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: apps/v1 ```   ```                    : scheduling/v1alpha1   ```   ```      :
: kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: Deployment ```   ```                 : Workload ```   ```                   :
: metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: ```   ```                            : ```   ```                            :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: t5-inference ```   ```               : w-t5-inference ```   ```             :
: labels\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:  :
: ```   ```                            : ```   ```                            :
: model\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : controllerRef\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: t5 ```   ```                         : ```   ```                            :
: version\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\:    :
: v1.0 ```   ```                       : t5-inference ```   ```               :
: machine\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : kind\\\\\\\\\\\\\\\\\\\\\\\\\\\:     :
: gpu ```   ```                        : Deployment ```   ```                 :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : apiGroup\\\\\\\\\\\\\\\\\\\\\\\\\\:  :
: ```   ```                            : apps ```   ```                       :
: replicas\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : replicaMode\\\\\\\\\\\\\\\\\\\\\\\\\: :
: 1 ```   ```                          : ReplicatedPods ```   ```             :
: selector\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : gangGroups\\\\\\\\\\\\\\\\\\\\\\\\:  :
: ```   ```                            :  ```   ```     -                     :
: matchLabels\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : name\\\\\\\\\\\\\\\\\\\\\\\: "gg"    :
: ```   ```                            : ```   ```                            :
: model\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : gangMode\\\\\\\\\\\\\\\\\\\\\\: Off  :
: t5 ```   ```                         : ```   ```                            :
: version\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : rankedGroups\\\\\\\\\\\\\\\\\\\\\:   :
: v1.0 ```   ```                       : ```   ```         -                  :
: machine\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : name\\\\\\\\\\\\\\\\\\\\: "rg" ```   :
: gpu ```   ```                        :  ```                                 :
: template\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : rankMode\\\\\\\\\\\\\\\\\\\: Off     :
: ```   ```                            : ```   ```                            :
: metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : eqGroups\\\\\\\\\\\\\\\\\\: ```      :
: ```   ```                            : ```           -                      :
: labels\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : name\\\\\\\\\\\\\\\\\: "template"    :
: ```   ```                            : ```   ```                            :
: model\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : minCount\\\\\\\\\\\\\\\\: 1 ```      :
: t5 ```   ```                         : ```                                  :
: version\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : maxCount\\\\\\\\\\\\\\\: 5 ```       :
: v1.0 ```   ```                       : ```                                  :
: machine\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : template\\\\\\\\\\\\\\:              :
: gpu ```   ```                        :    ```   ```                         :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : resources\\\\\\\\\\\\\:  ```   ```   :
: ```   ```                            :                limits\\\\\\\\\\\\:   :
: securityContext\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: ```   ```                            : nvidia.com/gpu\\\\\\\\\\\: "1" ```   :
: fsGroup\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :  ```                                 :
: 1000 ```   ```                       : cpu\\\\\\\\\\: "3000m" ```   ```     :
: runAsUser\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                memory\\\\\\\\\:      :
: 1000 ```   ```                       : 16Gi ```   ```                       :
: runAsGroup\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ephemeral-storage\\\\\\\\: 10Gi ```  :
: 1000 ```   ```                       :   ```                                :
: containers\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : requests\\\\\\\: ```   ```           :
: ```   ```         -                  :          nvidia.com/gpu\\\\\\: "1"   :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: inference ```   ```                  : cpu\\\\\: "3000m" ```   ```          :
: image\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :           memory\\\\: 16Gi ```       :
: repo.example/models/t5-small\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:v1 : ```                                  :
: ```   ```                            : ephemeral-storage\\\: 10Gi ```       :
: imagePullPolicy\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: IfNotPresent ```   ```               :                                      :
: args\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ["torchserve", "--start",            :                                      :
: "--foreground"] ```   ```            :                                      :
: resources\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: limits\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: nvidia.com/gpu\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: "1" ```   ```                        :                                      :
: cpu\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: "3000m" ```   ```                    :                                      :
: memory\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 16Gi ```   ```                       :                                      :
: ephemeral-storage\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 10Gi ```   ```                       :                                      :
: requests\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: nvidia.com/gpu\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: "1" ```   ```                        :                                      :
: cpu\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: "3000m" ```   ```                    :                                      :
: memory\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 16Gi ```   ```                       :                                      :
: ephemeral-storage\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 10Gi ```   ```                       :                                      :
: ports\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```             -              :                                      :
: containerPort\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 8080 ```   ```                       :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: http ```   ```             -         :                                      :
: containerPort\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 8081 ```   ```                       :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: management ```   ```             -   :                                      :
: containerPort\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 8082 ```   ```                       :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: metrics ```   ```                    :                                      :
: readinessProbe\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: httpGet\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: path\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: /ping ```   ```                      :                                      :
: port\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: http ```   ```                       :                                      :
: initialDelaySeconds\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 120 ```   ```                        :                                      :
: failureThreshold\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 10 ```   ```                         :                                      :
: livenessProbe\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: httpGet\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: path\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: /models/t5-small ```   ```           :                                      :
:                                      :                                      :
: port\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: management ```   ```                 :                                      :
: initialDelaySeconds\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 150 ```   ```                        :                                      :
: periodSeconds\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 5 ```   ``` --- ```   ```            :                                      :
: apiVersion\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: v1 ```   ```                         :                                      :
: kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: Service ```   ```                    :                                      :
: metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: t5-inference ```   ```               :                                      :
: labels\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: model\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: t5 ```   ```                         :                                      :
: version\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: v1.0 ```   ```                       :                                      :
: machine\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: gpu ```   ```                        :                                      :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: type\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ClusterIP ```   ```                  :                                      :
: selector\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: model\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: t5 ```   ```                         :                                      :
: version\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: v1.0 ```   ```                       :                                      :
: machine\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: gpu ```   ```                        :                                      :
: ports\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:ports\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```     -                      :                                      :
: port\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:   :                                      :
: 8080 ```   ```                       :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\:    :                                      :
: http ```   ```                       :                                      :
: targetPort\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: http ```   ```     -                 :                                      :
: port\\\\\\\\\\\\\\\\\\\\\\\\\\:      :                                      :
: 8081 ```   ```                       :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\:       :                                      :
: management ```   ```                 :                                      :
: targetPort\\\\\\\\\\\\\\\\\\\\\\\\:  :                                      :
: management ```   ```     -           :                                      :
: port\\\\\\\\\\\\\\\\\\\\\\\: 8082    :                                      :
: ```   ```                            :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\: metrics  :                                      :
: ```   ```                            :                                      :
: targetPort\\\\\\\\\\\\\\\\\\\\\:     :                                      :
: metrics ```   ``` --- ```   ```      :                                      :
: apiVersion\\\\\\\\\\\\\\\\\\\:       :                                      :
: autoscaling/v2 ```   ```             :                                      :
: kind\\\\\\\\\\\\\\\\\\:              :                                      :
: HorizontalPodAutoscaler ```   ```    :                                      :
: metadata\\\\\\\\\\\\\\\\\: ```       :                                      :
: ```   name\\\\\\\\\\\\\\\\:          :                                      :
: t5-inference ```   ```               :                                      :
: spec\\\\\\\\\\\\\\\: ```   ```       :                                      :
: scaleTargetRef\\\\\\\\\\\\\\: ```    :                                      :
: ```     apiVersion\\\\\\\\\\\\\:     :                                      :
: apps/v1 ```   ```                    :                                      :
: kind\\\\\\\\\\\\: Deployment ```     :                                      :
: ```     name\\\\\\\\\\\:             :                                      :
: t5-inference ```   ```               :                                      :
: minReplicas\\\\\\\\\\: 1 ```   ```   :                                      :
:  maxReplicas\\\\\\\\\: 5 ```   ```   :                                      :
:  metrics\\\\\\\\: ```   ```   -      :                                      :
: type\\\\\\\: Pods ```   ```          :                                      :
: pods\\\\\\: ```   ```                :                                      :
: metric\\\\\: ```   ```               :                                      :
: name\\\\:                            :                                      :
: prometheus.googleapis.com\\\\|ts_queue_latency_microseconds\\\\|counter :                                      :
: ```   ```       target\\\: ```       :                                      :
: ```         type\\: AverageValue     :                                      :
: ```   ```         averageValue\:     :                                      :
: "30000" ```                          :                                      :

### LeaderWorkerSet

This example is based on the [VLLM GPU LWS](https://github.com/kubernetes-sigs/lws/blob/main/docs/examples/vllm/GPU/lws.yaml) example from the LWS project.

In this example, each "replica" of the LWS is a 2-pod group.  This group should be scheduled all-or-nothing – the replica can only be healthy if both pods are running.   Hence, we set `replicaMode: ReplicatedGangs`.

Each replica is a gang, but to avoid a very large `Workload` at large number of replicas, the `Workload` object does not require every replica to be listed as a separate `GangGroup`.  Only one `GangGroup` is listed, and all replicas match it.

| **True Workload**                    |   ``` Workload ```                   |
| ------------------------------------ | ------------------------------------ |
|   ```                                |   ```                                |
: apiVersion\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : apiVersion\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: leaderworkerset.x-k8s.io/v1 ```      : scheduling/v1alpha1   ```   ```      :
: ```                                  : kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : Workload ```   ```                   :
: LeaderWorkerSet ```   ```            : metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: ```   ```                            : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : w-vllm ```   ```                     :
: vllm ```   ```                       : spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: ```   ```                            : controllerRef\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: replicas\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: 2 ```   ```                          : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: leaderWorkerTemplate\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : vllm ```   ```                       :
: ```   ```                            : kind\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: size\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : LeaderWorkerSet ```   ```            :
: 2 ```   ```                          : apiGroup\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: restartPolicy\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : leaderworkerset.x-k8s.io ```   ```   :
: RecreateGroupOnPodRestart ```   ```  :                                      :
:                                      : replicaMode\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: leaderTemplate\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ReplicatedGangs ```   ```            :
: ```   ```                            : gangGroups\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: metadata\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :  ```   ```     -                     :
: ```   ```                            : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: labels\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : "gg" ```   ```                       :
: ```   ```                            : gangMode\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: role\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : Gang ```   ```                       :
: leader ```   ```                     : rankedGroups\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```         -                  :
: ```   ```                            : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: containers\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : "rg" ```   ```                       :
: ```   ```           -                : rankMode\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : Off ```   ```                        :
: vllm-leader ```   ```                : eqGroups\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: image\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```           -                :
: vllm/vllm-openai\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:latest : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: ```   ```                            : "leader" ```   ```                   :
: env\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : minCount\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: ```   ```               -            : 2 ```   ```                          :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : maxCount\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: HUGGING_FACE_HUB_TOKEN ```   ```     : 2 ```   ```                          :
:                                      : template\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: value\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :  ```   ```                           :
: $HUGGING_FACE_HUB_TOKEN ```   ```    : containers\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
:                                      : ```   ```               -            :
: command\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: ```   ```               - sh ```     : vllm-leader ```   ```                :
: ```               - -c ```   ```     :                                      :
:            - "bash                   : resources\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: /vllm-workspace/examples/online_serving/multi-node-serving.sh : ```   ```                            :
: leader                               : limits\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: --ray_cluster_size=$(LWS_GROUP_SIZE); : ```   ```                            :
:  ```   ```                  python3  : nvidia.com/gpu\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: -m                                   : "8" ```   ```                        :
: vllm.entrypoints.openai.api_server   : memory\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: --port 8080 --model                  : 1124Gi ```   ```                     :
: meta-llama/Llama-3.1-405B-Instruct   :                                      :
: --tensor-parallel-size 8             : ephemeral-storage\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: --pipeline_parallel_size 2" ```      : 800Gi ```   ```                      :
: ```                                  : requests\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: resources\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: ```   ```                            : ephemeral-storage\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: limits\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : 800Gi ```   ```                      :
: ```   ```                            : cpu\\\\\\\\\\\\\\\\\\\\\\\\\\: 125   :
: nvidia.com/gpu\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: "8" ```   ```                        : volumes\\\\\\\\\\\\\\\\\\\\\\\\\:    :
: memory\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```             -              :
: 1124Gi ```   ```                     : name\\\\\\\\\\\\\\\\\\\\\\\\: dshm   :
: ephemeral-storage\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: 800Gi ```   ```                      : emptyDir\\\\\\\\\\\\\\\\\\\\\\\:     :
: requests\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: ```   ```                            : medium\\\\\\\\\\\\\\\\\\\\\\:        :
: ephemeral-storage\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : Memory ```   ```                     :
: 800Gi ```   ```                      : sizeLimit\\\\\\\\\\\\\\\\\\\\\:      :
: cpu\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : 15Gi ```   ```               ```     :
: 125 ```   ```                        : ```           -                      :
: ports\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : name\\\\\\\\\\\\\\\\\\\: "worker"    :
: ```   ```               -            : ```   ```                            :
: containerPort\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : minCount\\\\\\\\\\\\\\\\\\: 2 ```    :
: 8080 ```   ```                       : ```                                  :
: readinessProbe\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : maxCount\\\\\\\\\\\\\\\\\: 2 ```     :
: ```   ```                            : ```                                  :
: tcpSocket\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : template\\\\\\\\\\\\\\\\:  ```       :
: ```   ```                            : ```                                  :
: port\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : containers\\\\\\\\\\\\\\\: ```       :
: 8080 ```   ```                       : ```               -                  :
: initialDelaySeconds\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : name\\\\\\\\\\\\\\: vllm-worker ```  :
: 15 ```   ```                         :   ```                                :
: periodSeconds\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : resources\\\\\\\\\\\\\: ```   ```    :
: 10 ```   ```                         :               limits\\\\\\\\\\\\:    :
: volumeMounts\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: ```   ```               -            : nvidia.com/gpu\\\\\\\\\\\: "8" ```   :
: mountPath\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :  ```                                 :
: /dev/shm ```   ```                   : memory\\\\\\\\\\: 1124Gi ```   ```   :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: dshm ```   ```                       : ephemeral-storage\\\\\\\\\: 800Gi    :
: volumes\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```   ```                            :
: ```   ```         -                  : requests\\\\\\\\: ```   ```          :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :           ephemeral-storage\\\\\\\:  :
: dshm ```   ```                       : 800Gi ```   ```                      :
: emptyDir\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : cpu\\\\\\: 125 ```   ```             :
: ```   ```                            :    volumes\\\\\: ```   ```           :
: medium\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :        - name\\\\: dshm ```   ```    :
: Memory ```   ```                     :                 emptyDir\\\: ```     :
: sizeLimit\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : ```                   medium\\:      :
: 15Gi ```   ```                       : Memory ```   ```                     :
: workerTemplate\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: : sizeLimit\: 15Gi ```                 :
: ```   ```                            :                                      :
: spec\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: containers\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```           -                :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: vllm-worker ```   ```                :                                      :
: image\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: vllm/vllm-openai\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\:latest :                                      :
: ```   ```                            :                                      :
: command\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```               - sh ```     :                                      :
: ```               - -c ```   ```     :                                      :
:            - "bash                   :                                      :
: /vllm-workspace/examples/online_serving/multi-node-serving.sh :                                      :
: worker                               :                                      :
: --ray_address=$(LWS_LEADER_ADDRESS)"--ray_address=$(LWS_LEADER_ADDRESS)" :                                      :
: ```   ```                            :                                      :
: resources\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: limits\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: nvidia.com/gpu\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: "8" ```   ```                        :                                      :
: memory\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 1124Gi ```   ```                     :                                      :
: ephemeral-storage\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 800Gi ```   ```                      :                                      :
: requests\\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```                            :                                      :
: ephemeral-storage\\\\\\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: 800Gi ```   ```                      :                                      :
: cpu\\\\\\\\\\\\\\\\\\\\\\\\\\\: 125  :                                      :
: ```   ```                            :                                      :
: env\\\\\\\\\\\\\\\\\\\\\\\\\\: ```   :                                      :
:  ```               -                 :                                      :
: name\\\\\\\\\\\\\\\\\\\\\\\\\:       :                                      :
: HUGGING_FACE_HUB_TOKEN ```   ```     :                                      :
:                                      :                                      :
: value\\\\\\\\\\\\\\\\\\\\\\\\:       :                                      :
: $HUGGING_FACE_HUB_TOKEN ```   ```    :                                      :
:                                      :                                      :
: volumeMounts\\\\\\\\\\\\\\\\\\\\\\\:volumeMounts\\\\\\\\\\\\\\\\\\\\\\\: :                                      :
: ```   ```               -            :                                      :
: mountPath\\\\\\\\\\\\\\\\\\\\\\:     :                                      :
: /dev/shm ```   ```                   :                                      :
: name\\\\\\\\\\\\\\\\\\\\\: dshm      :                                      :
: ```   ```                            :                                      :
: volumes\\\\\\\\\\\\\\\\\\\\: ```     :                                      :
: ```         -                        :                                      :
: name\\\\\\\\\\\\\\\\\\\: dshm ```    :                                      :
: ```                                  :                                      :
: emptyDir\\\\\\\\\\\\\\\\\\: ```      :                                      :
: ```                                  :                                      :
: medium\\\\\\\\\\\\\\\\\: Memory ```  :                                      :
:   ```                                :                                      :
: sizeLimit\\\\\\\\\\\\\\\\: 15Gi ```  :                                      :
:   ``` --- ```   ```                  :                                      :
: apiVersion\\\\\\\\\\\\\\: v1 ```     :                                      :
: ``` kind\\\\\\\\\\\\\: Service ```   :                                      :
:  ``` metadata\\\\\\\\\\\\: ```       :                                      :
: ```   name\\\\\\\\\\\: vllm-leader   :                                      :
: ```   ``` spec\\\\\\\\\\: ```   ```  :                                      :
:   ports\\\\\\\\\: ```   ```     -    :                                      :
: name\\\\\\\\: http ```   ```         :                                      :
: port\\\\\\\: 8080 ```   ```          :                                      :
: protocol\\\\\\: TCP ```   ```        :                                      :
: targetPort\\\\\: 8080 ```   ```      :                                      :
: selector\\\\: ```   ```              :                                      :
: leaderworkerset.sigs.k8s.io/name\\\:leaderworkerset.sigs.k8s.io/name\\\: :                                      :
: vllm ```   ```     role\\: leader    :                                      :
: ```   ```   type\: ClusterIP ```     :                                      :

### JobSet with Different Leader and Worker Shapes

| **True Workload**                    |   ``` Workload ```                   |
| ------------------------------------ | ------------------------------------ |
|   ``` apiVersion\\\\\\\:             |   ```                                |
: jobset.x-k8s.io/v1 ```   ```         : apiVersion\\\\\\\\\\\\\\\\\\\\\\\\\\\: :
: kind\\\\\\: JobSet ```   ```         : scheduling/v1alpha1   ```   ```      :
: metadata\\\\\: ```   ```             : kind\\\\\\\\\\\\\\\\\\\\\\\\\\:      :
: name\\\\: jobset-1 ```   ```         : Workload ```   ```                   :
: spec\\\: ```   ``` spec\\: ```       : metadata\\\\\\\\\\\\\\\\\\\\\\\\\:   :
:                                      : ```   ```                            :
:                                      : name\\\\\\\\\\\\\\\\\\\\\\\\:        :
:                                      : w-jobset-1 ```   ```                 :
:                                      : spec\\\\\\\\\\\\\\\\\\\\\\\: ```     :
:                                      : ```                                  :
:                                      : controllerRef\\\\\\\\\\\\\\\\\\\\\\:controllerRef\\\\\\\\\\\\\\\\\\\\\\: :
:                                      : ```   ```                            :
:                                      : name\\\\\\\\\\\\\\\\\\\\\: jobset-1  :
:                                      : ```   ```                            :
:                                      : kind\\\\\\\\\\\\\\\\\\\\: JobSet     :
:                                      : ```   ```                            :
:                                      : apiGroup\\\\\\\\\\\\\\\\\\\:         :
:                                      : jobset.x-k8s.io ```   ```            :
:                                      : replicaMode\\\\\\\\\\\\\\\\\\:       :
:                                      : Unreplicated ```   ```               :
:                                      : gangGroups\\\\\\\\\\\\\\\\\:  ```    :
:                                      : ```     - name\\\\\\\\\\\\\\\\:      :
:                                      : "gg" ```   ```                       :
:                                      : gangMode\\\\\\\\\\\\\\\: Gang ```    :
:                                      : ```                                  :
:                                      : rankedGroups\\\\\\\\\\\\\\: ```      :
:                                      : ```         - name\\\\\\\\\\\\\:     :
:                                      : "rg" ```   ```                       :
:                                      : rankMode\\\\\\\\\\\\: Off ```   ```  :
:                                      :           eqGroups\\\\\\\\\\\: ```   :
:                                      :  ```           - name\\\\\\\\\\:     :
:                                      : "template" ```   ```                 :
:                                      : count\\\\\\\\\: 3 ```   ```          :
:                                      :     template\\\\\\\\:                :
:                                      :  ```   ```                           :
:                                      : resources\\\\\\\:  ```   ```         :
:                                      :         requests\\\\\\: ```   ```    :
:                                      :                cpu\\\\\: 10000 ```   :
:                                      :  ```                  memory\\\\:    :
:                                      : 100GB ```   ```                      :
:                                      : limits\\\: ```   ```                 :
:                                      :   cpu\\: 10000 ```   ```             :
:                                      :       memory\: 100GB ```             :

TODO:  
LWS undergoing a rolling update.
