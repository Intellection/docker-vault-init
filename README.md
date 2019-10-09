# Vault Init

[![Build Status](https://travis-ci.org/Intellection/vault-init.svg?branch=master)](https://travis-ci.org/Intellection/vault-init)

This is a tool to automate initialising Vault, encrypting the resulting root token with AWS KMS, and storing that token on AWS S3.

## Development

The instructions below are only necessary if you intend to work on the source code.

### Requirements

1. Ensure that you have a [properly
   configured](https://golang.org/doc/code.html#Workspaces) Go workspace.
1. For dependency management, you'll need `dep` which can be installed with
   `brew install dep`.

### Building

1. Fetch the code with `go get -v github.com/Intellection/vault-init`.
1. Install the Go development tools via `make tools`.
1. Install `dep` via `curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh`. `brew install dep` can also be used if using macOS.
1. Install application dependencies via `make dependencies` (they'll be placed
   in `./vendor`). Requires [golang/dep][dep] package manager.
1. Build and install the binary with `make build tag=x.y.z`.
1. Run the command e.g. `./bin/vault-init help` as a basic test.

### Testing

1. Install the Go testing tools via `make tools`.
2. Run linter using `make lint` and test using `make test`.
