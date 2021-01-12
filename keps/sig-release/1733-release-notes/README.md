# Release Notes Improvements

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

- There are multiple ways to generate release notes:
  [release-notes](https://github.com/kubernetes/release/tree/master/cmd/release-notes)
  [krel release-notes](https://github.com/kubernetes/release/blob/master/docs/krel/release-notes.md)
  [krel changelog](https://github.com/kubernetes/release/blob/master/docs/krel/changelog.md)

  `krel` and the stand-alone `release-notes` tool are already re-using the
  golang based libraries and are actively maintained. The former `relnotes`
  bash-script has been deprecated and therefore removed.

- We are currently shipping entire changelogs within markdown for each release.

  <!-- TODO: this seems not an issue to me -->

- An end-user may not need to see _every single change_ that occurred within a release.

There is room for improvement in the generation as well as the consumption of release notes.
In order to make the process more sustainable, and improve end-user experience, we should put
effort into automation as well as better ways to consume release notes.

### Goals

As this is a somewhat drastic departure from the current process, this should be a two-phased
approach:

- Consolidation and Clean-up
  - [x] Update `anago` to use `krel changelog`.
  - [x] Remove the bash-based `relnotes` script.
  - [x] Replace the "Detailed Bug Fixes And Changes" section in the release notes with a new
        website that allows user to filter and search.
  - [x] Generate a consumable JSON file via `release-notes` as part of the Release Team duties
  - [x] Identify and stop tracking any "External Dependencies" that a vanilla Kubernetes install
        does not rely on. (eg. Things in `cluster/addons`)
- Automation
  - [x] Build additional labels to classify:
    - [x] API Changes (done, since we now sort the release notes by `kind/`)
    - [x] Deprecations (see above)
    - [x] Urgent Upgrade Notes
  - [x] Put the generated release notes JSON in to a Google Cloud Bucket
  - [x] Use milestones to capture "Known Issues" at time of release notes generation

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

It should be possible to completely automate the generation and publishing of
the release notes for the [website](https://relnotes.k8s.io). The idea is to put
the (manually) generated JSON into a Google Cloud Bucket and let the website
scrape the bucket for new notes. This can be done by placing an index.json file
side-by-side to the actual notes, like we do it in the [current website
implementation](https://github.com/puerco/release-notes/blob/master/src/environments/assets.ts).

Once a release is ready to be cut, the release notes team would then run `krel release-notes`
which takes care of editing the Google Cloud Bucket.

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
new release notes website. Said repo will also need to be integrated with Netlify for hosting.
Lastly, a DNS entry will be needed to point to the Netlify site (proposal: relnotes.k8s.io)
