# Graduation Readiness Expectations

This document provides guidance for KEP authors and reviewers on the types of evidence expected when requesting feature graduation (Alpha → Beta → Beta → GA).

## Overview

Graduation criteria should be defined per-KEP, but having common guidance helps:

- Authors prepare stronger graduation requests
- Reviewers evaluate consistently across SIGs
- Reduce repeated feedback across KEP reviews

## Alpha → Beta

When requesting graduation from Alpha to Beta, authors should provide:

### Functionality
- [ ] All planned features are implemented
- [ ] Feature gate is functional and tested
- [ ] API surface is stable (no breaking changes planned)

### Testing
- [ ] Unit test coverage for new/changed code
- [ ] Integration tests covering key scenarios
- [ ] E2e tests in TestGrid with stable results
- [ ] No non-infra flakes in the last 30 days

### User Feedback
- [ ] Collect feedback from at least 2-3 adopters
- [ ] Document known limitations and workarounds
- [ ] Address or track all reported issues

### Operations
- [ ] Monitoring and alerting guidance documented
- [ ] Upgrade/downgrade path tested
- [ ] Version skew strategy defined (if applicable)

### Documentation
- [ ] User-facing docs in kubernetes/website
- [ ] API documentation updated
- [ ] Release notes drafted

## Beta → GA

When requesting graduation from Beta to GA, authors should provide:

### Stability
- [ ] API has been stable for at least 2 releases (6 months)
- [ ] No breaking API changes planned
- [ ] All Beta feedback addressed

### Testing
- [ ] Comprehensive e2e test coverage
- [ ] Conformance tests (if non-optional feature)
- [ ] Performance/ scalability benchmarks (if applicable)
- [ ] Downgrade/rollback tests passing

### Adoption
- [ ] Evidence of real-world usage (N examples)
- [ ] Documented production deployments
- [ ] Community feedback incorporated

### Operations
- [ ] SLIs/SLOs defined and measurable
- [ ] Runbooks and troubleshooting guides
- [ ] Support escalation paths documented

### Documentation
- [ ] Complete user documentation
- [ ] Migration guides from Beta
- [ ] Deprecation notices for replaced APIs (if applicable)

## Graduation Evidence Checklist

### Required for All Graduations
1. **Test plan** — Describe testing strategy and coverage
2. **E2e tests** — Stable and passing in CI
3. **Documentation** — User-facing docs created
4. **Implementation history** — Updated in KEP

### Required for Beta
1. **Feature completeness** — All planned functionality implemented
2. **Security review** — Completed if touching auth/RBAC
3. **Monitoring** — Metrics and alerting guidance
4. **User feedback** — Collected and addressed

### Required for GA
1. **Conformance tests** — For non-optional features
2. **Adoption evidence** — Real-world usage documented
3. **Stability period** — At least 2 releases as Beta
4. **Production readiness** — PRR approved

## Reviewer Guidance

When reviewing graduation requests:

1. **Check the checklist** — Ensure all required items are addressed
2. **Verify evidence** — Links to test results, docs, feedback
3. **Assess completeness** — Are gaps identified and addressed?
4. **Confirm consistency** — Does graduation align with deprecation policy?

## References

- [KEP Template](/keps/NNNN-kep-template/README.md)
- [Production Readiness Review](/keps/sig-architecture/1194-prod-readiness)
- [API Maturity Levels](https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions)
- [Conformance Tests](https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md)
