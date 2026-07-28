# Contributing to taksuMCP

Thanks for considering a contribution. This project aims to stay simple
to build, test, and run — please keep that bar in mind for any PR.

## Prerequisites

- Go 1.22+
- (Optional) Docker + Docker Compose, for the containerized workflow
- (Optional) `golangci-lint` for local linting

No Node.js, npm, or build step is required — the HTMX frontend is
served directly from Go via `embed.FS`.

## Getting started

```bash
git clone https://github.com/Tharun-bot/taksuMCP.git
cd taksuMCP
go build ./...
go test ./...
go run ./cmd/server
```

## Development workflow

1. Fork the repo and create a branch: `git checkout -b feature/short-description`
2. Make your change with tests. PRs without tests for new logic will be
   asked to add them.
3. Run before pushing:
```bash
   go vet ./...
   go test -race ./...
   golangci-lint run   # if installed
```
4. Open a PR against `main` using the PR template. Link any related
   issue.

## Commit messages

Use short, imperative subject lines (`Add TTL reaper for expired tasks`,
not `Added` or `Adding`). Reference issue numbers where relevant.

## Code style

- Standard `gofmt`/`goimports` formatting — no exceptions.
- Prefer small, focused interfaces (see `internal/taskstore.TaskStore`
  as the reference example).
- No panics in library code paths — return errors.
- New backends must implement the full `TaskStore` interface and pass
  the shared conformance test suite in `internal/taskstore/conformance/`.

## Where to start

Check issues labeled [`good first issue`](https://github.com/Tharun-bot/taksuMCP/labels/good%20first%20issue).
If nothing fits, open an issue describing what you'd like to work on
before submitting a large PR — saves everyone rework.

## Reporting bugs / requesting features

Use the issue templates under `.github/ISSUE_TEMPLATE/`.

## Code of Conduct

This project follows the [Contributor Covenant](./CODE_OF_CONDUCT.md).