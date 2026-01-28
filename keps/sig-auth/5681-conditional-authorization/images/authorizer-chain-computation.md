# Mermaid source for authorizer-chain-computation image

This diagram was rendered on https://www.mermaidchart.com/play, not using the
native GitHub mermaid renderer so that the image was more visually clear.
Mermaid layout configs are not respected by GitHub.

<!-- toc -->
<!-- /toc -->

```mermaid
---
config:
  layout: elk
title: Kubernetes with Conditional Authorization
---
flowchart LR
 subgraph ChainAuthorizer["ChainAuthorizer"]
    direction TB
        AuthzAllow["Allow"]
        AuthzConditional["Conditional"]
        AuthzDeny["Deny"]
        AuthzNoOpinion["NoOpinion"]
  end
 subgraph WithAuthorization["WithAuthorization"]
    direction TB
        ServeHTTP["ServeHTTP"]
        ServeHTTPWithConditions["ServeHTTP + ctx conditions"]
        CannotBecomeAuthz["CannotBecomeAuthz"]
  end
 subgraph ChainAuthorizerEval["ChainAuthorizerEval"]
        NoOpinion2["NoOpinion"]
        Allow2["Allow"]
        Conditional2["Conditional"]
        Deny2["Deny"]
  end
 subgraph ValidatingAdmission["ValidatingAdmission"]
    direction TB
        AdmissionAllow["Allow"]
        AdmissionEvaluate["Evaluate"]
        AdmissionNoOpinion["NoOpinion"]
        AdmissionDeny["Deny"]
        ChainAuthorizerEval
  end
    Request["Request"] --> ChainAuthorizer
    AuthzNoOpinion --> Request
    AuthzAllow --> ServeHTTP
    ServeHTTPWithConditions --> AdmissionEvaluate
    AdmissionEvaluate --> AdmissionNoOpinion & AdmissionAllow & AdmissionDeny
    AdmissionNoOpinion --> ChainAuthorizerEval
    NoOpinion2 --> AdmissionNoOpinion
    Allow2 --> AdmissionAllow
    Conditional2 --> AdmissionEvaluate
    Deny2 --> AdmissionDeny
    AdmissionDeny --> 403(["403"])
    ServeHTTP -- Allowed --> AdmissionAllow["Allow"]
    AuthzConditional -- 1+ Allow --> ServeHTTPWithConditions
    AuthzConditional -- 0 Allow --> CannotBecomeAuthz
    AuthzDeny --> CannotBecomeAuthz
    CannotBecomeAuthz --> 403
    AdmissionAllow --> Storage(["Storage"])
```
