# Kubernetes Enhancement Proposals (KEPs)

A Kubernetes Enhancement Proposal (KEP) is a way to propose, communicate and coordinate on new efforts for the Kubernetes project.
You can read the full details of the project in [KEP-0000](sig-architecture/0000-kep-process/README.md).

This process is still in a _beta_ state and is mandatory for all enhancements beginning release 1.14.

## Quick start for the KEP process

1. Socialize an idea with a sponsoring SIG.
   You can send your idea to their mailing list, or add it to the agenda for one of their upcoming meetings.
   Make sure that others think the work is worth taking up and will help review the KEP and any code changes required.
2. Follow the process outlined in the [KEP template](NNNN-kep-template/README.md)

## FAQs

### Do I have to use the KEP process?

More or less, yes.

Having a rich set of KEPs in one place will make it easier for people to track
what is going in the community and find a structured historical record.

KEPs are required for most non-trivial changes.  Specifically:
* Anything that may be controversial
* Most new features (except the very smallest)
* Major changes to existing features
* Changes that are wide ranging or impact most of the project (these changes
  are usually coordinated through SIG-Architecture)

Beyond these, it is up to each SIG to decide when they want to use the KEP
process.  It should be light-weight enough that KEPs are the default position.

### Why would I want to use the KEP process?

Our aim with KEPs is to clearly communicate new efforts to the Kubernetes contributor community.
As such, we want to build a well curated set of clear proposals in a common format with useful metadata.

Benefits to KEP users (in the limit):
* Exposure on a kubernetes blessed web site that is findable via web search engines.
* Cross indexing of KEPs so that users can find connections and the current status of any KEP.
* A clear process with approvers and reviewers for making decisions.
  This will lead to more structured decisions that stick as there is a discoverable record around the decisions.

We are inspired by IETF RFCs, Python PEPs and Rust RFCs.
See [KEP-0000](sig-architecture/0000-kep-process/README.md) for more details.

### Do I put my KEP in the root KEP directory or a SIG subdirectory?

Almost all KEPs should go into SIG subdirectories.  In very rare cases, such as
KEPs about KEPs, we may choose to keep them in the root.

If in doubt ask [SIG Architecture](https://git.k8s.io/community/sig-architecture/README.md) and they can advise.

### What will it take for KEPs to "graduate" out of "beta"?

Things we'd like to see happen to consider KEPs well on their way:
* A set of KEPs that show healthy process around describing an effort and recording decisions in a reasonable amount of time.
* KEPs exposed on a searchable and indexable web site.
* Presubmit checks for KEPs around metadata format and markdown validity.

Even so, the process can evolve. As we find new techniques we can improve our processes.

### What is the number at the beginning of the KEP name?

KEPs are now prefixed with their associated tracking issue number. This gives
both the KEP a unique identifier and provides an easy breadcrumb for people to
find the issue where the current state of the KEP is being updated.

### My FAQ isn't answered here!

The KEP process is still evolving!
If something is missing or not answered here feel free to reach out to [SIG Architecture](https://git.k8s.io/community/sig-architecture/README.md).
If you want to propose a change to the KEP process you can open a PR on [KEP-0000](sig-architecture/0000-kep-process/README.md) with your proposal.
