# kepval

`kepval` is a tool that checks whether the YAML metadata in a KEP (Kubernetes
Enhancement Proposal) is valid.

## Getting Started

1. Install `kepval`: `GO111MODULE=on go get k8s.io/enhancements/cmd/kepval`
2. [Optional] clone the enhancements for test data `git clone https://github.com/kubernetes/enhancements.git`
3. Run `kepval <path to kep.md>`

## Development

1. Run the tests with `go test -cover ./...`

## SIG Node assigned reviewers/approvers

As per [Scaling up KEP approvers](https://github.com/kubernetes/community/blob/main/sig-node/CONTRIBUTING.md#scaling-up-kep-approvers) SIG Node declares an official way to assign approvers to KEPs past beta and enforces tech leads to be approvers for KEPs entering alpha.

Every SIG Node KEP must list at least one `sig-node-tech-leads` member
(from `OWNERS_ALIASES`) or an approver annotated with
`# sig-node-assigned-approver` under `approvers:`. Active alpha KEPs must list
a tech lead and must not use the `# sig-node-assigned-approver` marker.

