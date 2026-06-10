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

A `kep.yaml` may annotate individual entries with the inline comments
`# sig-node-assigned-reviewer` and `# sig-node-assigned-approver` to
[scale SIG Node KEP approvers](https://github.com/kubernetes/community/blob/main/sig-node/CONTRIBUTING.md#scaling-up-kep-approvers). Any handle so
annotated must also be listed in the `OWNERS` file next to that `kep.yaml`:
assigned reviewers under `reviewers:`, assigned approvers under `approvers:`.
Conversely, every entry in an `OWNERS` file must have a corresponding annotation
in `kep.yaml` — extra entries in `OWNERS` that are not annotated in `kep.yaml`
are also flagged as violations.
