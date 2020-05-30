# kepval

`kepval` is a tool that checks whether the YAML metadata in a KEP (Kubernetes
Enhancement Proposal) is valid.

## Getting Started

1. Install `kepval`: `GO111MODULE=on go get k8s.io/enhancements/cmd/kepval`
2. [Optional] clone the enhancements for test data `git clone https://github.com/kubernetes/enhancements.git`
3. Run `kepval <path to kep.md>`

## Development

1. Run the tests with `go test -cover ./...`
