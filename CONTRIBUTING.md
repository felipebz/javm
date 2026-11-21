# Contributing to javm

Thanks for helping improve javm. This document explains how to report problems, propose changes, and what a pull
request needs before it can be merged.

## Reporting bugs and requesting features

Use [GitHub Issues](https://github.com/felipebz/javm/issues) for bug reports and feature requests. For a bug, include:

* the output of `javm --version`,
* your operating system, architecture, and shell,
* the command you ran, what you expected, and what happened (run it again with `--debug` for more detail).

**Do not report security vulnerabilities in public issues.** Follow [SECURITY.md](SECURITY.md) to report them
privately.

## Proposing changes

Changes are made through pull requests against the `main` branch:

1. For anything larger than a small fix, open an issue first so we can agree on the approach before you write code.
2. Fork the repository and create a branch from `main`.
3. Make the change, with tests (see below).
4. Open a pull request describing what changed and why, and link the issue it addresses.

The maintainer reviews every pull request. CI must pass before a pull request is merged: tests on Linux, macOS, and
Windows, the race detector, `gofmt`, `go vet`, vulnerability scanning, CodeQL, and SonarQube.

## Development setup

You need Go **1.27** or newer.

```sh
git clone https://github.com/felipebz/javm
cd javm
go test ./...
go build -o javm .
./javm --help
```

Tests never touch your real javm data: they set `JAVM_HOME` to a temporary directory. When you try a build by hand, do
the same so it does not change your installed JDKs:

```sh
JAVM_HOME=$(mktemp -d) ./javm install temurin@21
```

## Requirements for a pull request

### Tests

* New functionality **must** come with automated tests that exercise it.
* Bug fixes **must** include a regression test that fails without the fix, unless an existing test already covers the
  case.
* Tests live next to the code in `*_test.go` files and use `t.TempDir()`, `t.Setenv("JAVM_HOME", ...)`, and
  `httptest` instead of the network or your home directory.
* Behavior that differs by operating system must be tested on each one, or skipped with `runtime.GOOS` and `t.Skip`
  when it cannot run there.

### Code style

* Code must be formatted with `gofmt` and follow [Effective Go](https://go.dev/doc/effective_go) and the
  [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments).
* Wrap errors with `%w` and never ignore them.
* Text that comes from JDKs, the DiscoAPI, or files must pass through `sanitizeTerminalText` before it is printed.
* Paths and archive entries are untrusted input: reject traversal, absolute paths, and links that leave the target
  directory.
* Avoid new dependencies. If one is needed, explain why in the pull request.

Run these before opening the pull request:

```sh
gofmt -l .        # must print nothing
go vet ./...
go test ./...
go test -race ./... # when touching concurrency, locks, or processes
```

### Commits

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/) with a scope, for example
`feat(ls): add --json output` or `fix(archive): reject absolute symlink targets`. Releases and the changelog are
generated from them, so use `feat` and `fix` only for user-visible changes, and `refactor`, `test`, `docs`, `ci`, or
`chore` for everything else.

Do not edit `CHANGELOG.md`, `.github/.release-please-manifest.json`, or the version in `sonar-project.properties`;
the release tooling updates them.

## License

javm is licensed under the [Apache License 2.0](LICENSE). By submitting a contribution, you agree that it is licensed
under the same terms.
