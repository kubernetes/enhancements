# kepval

`kepval` is a tool that checks the YAML metadata in a KEP (Kubernetes
Enhancement Proposal) is valid.

## Getting started

1. Clone the enhancements `git clone https://github.com/kubernetes/enhancements.git`
2. Install `kepval`: `GO111MODULEs=on go get github.com/chuckha/kepview/cmd/kepval`
3. Run `kepval <path to kep.md>`

## Development

1. Run the tests with `go test -cover ./...`
