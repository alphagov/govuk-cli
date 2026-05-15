# govuk-cli

A CLI for interacting with the GOV.UK platform.

## Development

### Prerequisites

* Go 1.26
* [GoReleaser CLI](https://goreleaser.com/getting-started/install/)

### Build and test

The CLI binary can be built with make: `make`

Unit tests can be run with `make unit_tests`, integration tests with `make integration_tests` and all tests with `make test`.

This project uses GoReleaser to build release artifacts such as macOS universal binaries, shell completions and a Homebrew cask.
Run `goreleaser release --snapshot --clean` to build all release artifacts.

### Release a new version

This project uses [Semantic Versioning](https://semver.org/).

To create a new release, use the [Create Versioned Release](https://github.com/alphagov/govuk-cli/actions/workflows/release.yaml)
GitHub Actions workflow.
Select the correct version bump level (patch, minor or major) based on the changes made since the last release.

The release process works as follows:

1. 'Create Versioned Release' is triggered manually
2. A Git tag is calculated based on the provided version bump level and the latest version number
3. `goreleaser release --clean` runs, which:
   1. Builds the CLI for macOS x86 and arm64
   2. Produces a universal binary containing both architectures
   3. Produces shell completion files for bash, zsh and fish
   4. Packages up the binary and shell completion files into a .tar.gz
   5. Creates a GitHub Release with the packaged binary and creates a changelog based on commits since last release
   6. Updates the [homebrew-gds tap](https://github.com/alphagov/homebrew-gds/blob/master/Casks/govuk-cli.rb)
