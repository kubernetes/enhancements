# Release notes items to discuss (Radhika, Jennifer)

## Gaps

Much content does not indicate what changed in this release. Even detailed descriptions of features don't indicate reliably whether they are new, or changes to existing functionality. Missing also are explanations of why changes were made (or added), or any indication of a use case. Asking for use cases is probably reaching too far, especially at this late date, but some indication of how things changed is important.

The writers propose to contact individual sig leads, or where known the original release note authors, to ask these questions one-on-one and expand the relevant release notes as we revise them. In some cases there's enough information in linked PRs or issues for us to investigate on our own, but we don't have time to go code diving.

## Inconsistent terminology

API changes especially are called out in various ways: they are promoted, graduated, advanced. We should standardize. (Note that none of these is a standard way to refer to version number increases.) The same goes for referring to API groups (themselves a confusing concept to outsiders). Radhika and Jennifer are working on this issue.

## Information architecture

Thinking about useful buckets (getting rid of sig org altogether?). Look at app developer user persona for ideas? (here: https://docs.google.com/document/d/1EdQ8acmuKGlzZy1agejLqmB7cLpTRBJKC0JYptJ-ylg/edit#heading=h.ylz4u4aax62y)

One (example) option (details not final or guaranteed to be accurate):

- Cluster config changes
    - New Features
    - API changes
    - Known Issues
- Node/pod changes
    - New Features
    - API changes
    - Known Issues
- Networking changes
    - New Features
    - API changes
    - Known Issues
- Auth changes
    - New Features
    - API changes
    - Known Issues
- Command-line tool changes (kubectl)
    - New Features
    - API changes
    - Known Issues
- Storage changes
    - New Features
    - API changes
    - Known Issues


Carrying over known issues from previous releases? (documenting fixes?)


