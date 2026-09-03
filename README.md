# govuk-cli

A CLI for interacting with the GOV.UK platform.

## Usage

### Installation

**Via Homebrew (recommended):**

1. Install [homebrew](https://brew.sh/)
2. Run `brew install alphagov/gds/govuk-cli`

**Direct Download:**

1. Download the tar.gz from the [latest release](https://github.com/alphagov/govuk-cli/releases)
2. Extract the archive, and use the resulting binary: `./govuk-cli --help`

### Job Requests

This CLI can be used to submit and review job requests.
Ensure you have [AWS credentials](https://github.com/alphagov/gds-cli#readme) ready for the environment you want to run jobs in before continuing.

#### Make a job request

A job request requires a source workload and a command to run.
The source workload provides the application image and environment details, such as env vars to your job.

As an example, to run a whitehall-admin rake task:

```shell
$ govuk-cli jobrequest create whitehall-admin rake 'my:task[some,args]'
2026/07/21 15:03:59 INFO Job request created
┌─────────┬───────────────────────────┐
│ Name    │ jr-whitehall-1604970138   │
│ Command │ rake 'my:task[some,args]' │
└─────────┴───────────────────────────┘
Review command:
 $ govuk-cli jobrequest review -n apps jr-whitehall-1604970138
```

A command is provided that can be sent to a colleague to review your request.

#### Review a job request

Reviewing a job request can be done with the `review` command:

```shell
$ govuk-cli jobrequest review -n apps jr-whitehall-1604970138
┌──────────────────┬───────────────────────────────┐
│ Job Request Name │ jr-whitehall-1604970138       │
│ Command          │ rake 'my:task[some,args]'     │
│ Source Workload  │ deployment/whitehall-admin    │
│ Status           │ Pending                       │
│ Requested By     │ alice.doesntexist (developer) │
└──────────────────┴───────────────────────────────┘
Review options: [A]pprove [R]eject
Your decision: 
```

You'll be asked for a decision on whether to approve or reject the request.
You'll then be asked for an optional comment which will be attached to your review for future reference:

```shell
...
Review options: [A]pprove [R]eject
Your decision: approve
Comment: LGTM
┌──────────────────┬─────────────────────────┐
│ Job Request Name │ jr-whitehall-1604970138 │
│ Decision         │ Approved                │
│ Comment          │ LGTM                    │
└──────────────────┴─────────────────────────┘
Submit review? [Y/n]:
```

#### View job request details

Details of a job request, including the command, who created the request, and any review state can be viewed with the `get` command:

```shell
$ govuk-cli jobrequest get jr-whitehall-1604970138
┌──────────────────┬───────────────────────────────┐
│ Job Request Name │ jr-whitehall-1604970138       │
│ Command          │ rake 'my:task[some,args]'     │
│ Source Workload  │ deployment/whitehall-admin    │
│ Status           │ Pending                       │
│ Requested By     │ alice.doesntexist (developer) │
└──────────────────┴───────────────────────────────┘
```

#### Following logs

All commands support the `--follow` or `-f` flag.
If specified, the CLI will perform the action requested, then wait for the job to start, and tail the logs of the running container.

Example commands:
* `govuk-cli jobrequest create signon rake 'some:task' -f`
* `govuk-cli jobrequest review jr-signon-123456789 -f`

## Development

### Prerequisites

* Go 1.26
* [kubectl](https://kubernetes.io/docs/reference/kubectl/)
* [GoReleaser CLI](https://goreleaser.com/getting-started/install/)

### Build and test

The CLI binary can be built with make: `make`

Unit tests can be run with `make unit_tests`, integration tests with `make integration_tests` and all tests with `make test`.

This project uses GoReleaser to build release artifacts such as macOS universal binaries, shell completions and a Homebrew cask.
Run `goreleaser release --snapshot --clean` to build all release artifacts.

### Creating and reviewing Job Requests

Create your job request, then use [impersonate](https://kubernetes.io/docs/reference/access-authn-authz/user-impersonation/) to create your own job request review.

(users can't do this because they don't have the permission/ verb to "impersonate" on their roles)

```jrr.yaml
apiVersion: platform.publishing.service.gov.uk/v1
kind: JobRequestReview
metadata:
  name: <job-request-review-name>
  namespace: apps
spec:
  decision: Approved
  description: doing some manual testing for some changes in the cli
  jobRequestName: <job-request-name>
```

The following will fail with a permissions error:

`kubectl apply -f jrr.yaml --as=<your-user-arn-with-your-username-changed> --as-group=system:basic-user`

The following will create the job request review and "Approve" your job request

`kubectl apply -f jrr.yaml --as=<your-user-arn-with-your-username-changed> --as-group=system:masters`

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
