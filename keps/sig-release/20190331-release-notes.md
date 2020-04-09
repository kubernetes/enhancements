---
title: Release Notes Improvements
authors:
  - "@jeefy"
owning-sig: sig-release
participating-sigs:
  - sig-contributor-experience
  - sig-docs
reviewers:
  - "@spiffxp"
  - "@marpaia"
approvers:
  - "@calebamiles"
  - "@tpepper"
  - "@justaugustus"
editor: TBD
creation-date: 2019-03-31
last-updated: 2019-03-31
status: provisional
---

# Release Notes Improvements

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Consolidation and Clean-up](#consolidation-and-clean-up)
    - [Updating Anago](#updating-anago)
    - [Release Notes Website](#release-notes-website)
  - [Automation](#automation)
    - [Build additional labels](#build-additional-labels)
    - [Automated release notes](#automated-release-notes)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Graduation Criteria](#graduation-criteria)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Summary

This document describes a new release notes process as well as a new site for end-users to
better consume the generated data. While this change would only affect the release-notes
team, this is a visible change large enough to warrant a KEP.

## Motivation

The current release notes process is fraught with inefficiencies that should be streamlined.

- There are two different ways to generate release notes:
[relnotes](https://github.com/kubernetes/release/blob/master/relnotes) and
[release-notes](https://github.com/kubernetes/release/tree/master/cmd/release-notes).
  - Because of this there's duplicate effort maintaining separate tools.
- We are currently shipping entire changelogs within markdown for each release.
- PRs are frequently categorized within several SIGs, which the release notes team has to
  distill down for the sake of brevity before the release is actually cut.
- An end-user may not need to see _every single change_ that occurred within a release.

There is room for improvement in the generation as well as the consumption of release notes.
In order to make the process more sustainable, and improve end-user experience, we should put
effort into automation as well as better ways to consume release notes.

### Goals

As this is a somewhat drastic departure from the current process, this should be a two-phased
approach:

- Consolidation and Clean-up
  - Update `anago` to use `release-notes`
  - Replace the "Detailed Bug Fixes And Changes" section in the release notes with a new
    website that allows user to filter and search.
  - Generate a consumable JSON file via `release-notes` as part of the Release Team duties
  - Identify and stop tracking any "External Dependencies" that a vanilla Kubernetes install
    does not rely on. (eg. Things in `cluster/addons`)
- Automation
  - Build additional labels to classify:
    - API Changes
    - Deprecations
    - Urgent Upgrade Notes
  - Capture release notes automatically via GitHub PR webhooks.
    - Use milestones to capture "Known Issues" at time of release notes generation

### Non-Goals

This is scoped solely around generating release notes for the main Kubernetes releases.

## Proposal

As stated above, this effort should be split into two phases.

### Consolidation and Clean-up

#### Updating Anago

Currently, `anago` uses a 
[different tool](https://github.com/kubernetes/release/blob/master/relnotes) to generate new
release notes with every release save for main 1.x.0 releases. We need to ensure that the
`release-notes` tool can generate the same output as the `relnotes` tool to ensure consistency.

#### Release Notes Website

Some progress has already been made (as a [POC Website](https://k8s-relnotes.netlify.com/)) to
drum up interest. We need to move the current codebase out of a
[personal repo](https://github.com/jeefy/relnotes) and into an official repo.

The `release-notes` tool is already capable of outputting a JSON format of the release notes,
which is what the current POC is consuming.

### Automation

#### Build additional labels

Once we can reliably generate a changelog, we should then strive to automate classifying the other
components of release notes. The notable sections are:

- API Changes (release/api_change)
- Deprecations (release/deprecation)
- Urgent Upgrade Notes (release/urgent)

On top of advertising these new labels, the release notes team should actively monitor and apply
these labels to ensure less manual classification.

#### Automated release notes

It should be possible to completely automate the generation and publishing of release notes. By
building a Knative pipeline that is fed from GitHub PR events, we could have it grab, format,
and commit new entries into the release notes website. As we don't want these notes to be
published before a release goes live, we would create a "draft" flag in the `release-notes` JSON
schema.

Once a release is ready to be cut, the release notes team would then flip all the 1.x notes that
have been collected to not be drafts. This would be done by cloning down the release notes
website, and running the `release-notes` tool over the JSON and committing the new output.

Example:

```bash
release-notes -i relnotes.json -p 1.15.0
```

### Risks and Mitigations

Automation inevitably fails. Once the full automated release notes process has been implemented,
we will need a means to monitor it. The fallback would be to manually generate and commit a JSON
file to the release notes website.

## Design Details

### Graduation Criteria

While this isn't directed at any single release, the goal is to phase the full implementation in
over multiple releases. Ultimately, this KEP would graduate once we have a dedicated release notes
website that is automatically updated with minimal human interaction.

## Infrastructure Needed

A GitHub repo will need to be setup to host code for both the `release-notes` tool as well as the
new release notes website. Said repo will also need to be integrated with Netlify for hosting. An
initial design idea to power the automatic generation of release notes was to use a Knative pipeline.
This would require a Kubernetes cluster to run on, as well as a GitHub token and webhook registration.
Lastly, a DNS entry will be needed to point to the Netlify site (proposal: relnotes.k8s.io)
