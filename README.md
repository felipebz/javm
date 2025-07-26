<p align="center">
<img src="assets/logo.svg" alt="javm logo">
</p>

[![Build Status](https://github.com/felipebz/javm/workflows/Build/badge.svg)](https://github.com/felipebz/javm/actions)
[![Quality Gate Status](https://sonarqube.felipebz.com/api/project_badges/measure?project=javm&metric=alert_status)](https://sonarqube.felipebz.com/dashboard?id=javm)

**javm** is a fast, cross‑platform Java version manager (successor of [jabba](https://github.com/shyiko/jabba)) focused on a frictionless install 
and switch workflow for JDKs on Linux, macOS and Windows.

## Features

- Install & switch between multiple JDK distributions (Zulu, OpenJDK, GraalVM, Temurin, etc.) using semver ranges
- Per‑project JDK via `.javmrc` (checked into VCS)
- Remote discovery (DiscoAPI integration) + custom URLs
- Supports semantic version ranges (`1.8.x`, `~17.0.2`, `>=21 <22`)
- Clean removal (`uninstall`, `deactivate`) without touching system JDK
- Static Go binary: fast cold start and no additional dependency


## Installation

TBD

## Usage

### Listing & Searching

```sh
javm ls-remote                      # list all available JDKs
javm ls-remote zulu@~1.8.60         # narrow by distribution & semver range
javm ls-remote "*@>=1.6.45 <1.9" --latest=minor  # show only latest minors
```

### Installing

```sh
javm install zulu@1.8               # Zulu OpenJDK 8 (latest matching)
javm install zulu@~1.8.144          # same as zulu@>=1.8.144 <1.9.0
javm install temurin@17             # Temurin 17 LTS
javm install graalvm@21             # GraalVM
javm install openjdk@21             # Upstream OpenJDK
```

Custom URL (qualifiers: `zip`, `tgz`, `tgx`):

```sh
javm install 1.8.0-custom=tgz+https://example.com/dist.tar.gz
```

### Using / Switching

```sh
javm ls                              # list installed
javm use zulu@1.8
javm use temurin@17
javm deactivate                      # restore previous JAVA_HOME / PATH
```

### Per‑Project Version (`.javmrc`)

Create a `.javmrc` in your project root:

```yaml
jdk: 17
```

or just:

```yaml
17
```

Then:

```sh
javm use   # picks version from .javmrc
```

### Aliases

```sh
javm alias default 17       # sets default JDK for new shells
javm alias list
```

### Linking Existing System JDK

```sh
javm link system@1.8.72 /Library/Java/JavaVirtualMachines/jdk1.8.0_72.jdk
```

### Uninstall

```sh
javm uninstall zulu@1.8
```


> `.javmrc` has to be a valid YAML file. JDK version can be specified as `jdk: 1.8` or simply as `1.8` 
(same as `~1.8`, `1.8.x` `">=1.8.0 <1.9.0"` (mind the quotes)).

For more information see `javm --help`.  

## Development

Prerequisite: Go **1.24.x**

```sh
git clone https://github.com/felipebz/javm
cd javm
go test ./...
go build -o javm .
./javm --help
```

## FAQ

**Q: Does it override my system Java?**\
No. `javm use` adjusts your shell `PATH` and `JAVA_HOME` *in that session*.

**Q: How do I revert to the system JDK?**\
Run `javm deactivate` or open a new shell (if no default alias points elsewhere).

**Q: Can I pin a project JDK?**\
Yes: `.javmrc` with `jdk: <range>` then `javm use` inside that directory.

**Q: Why not install “globally”?**\
Global changes (e.g. system alternatives, registry edits) vary by OS and often require sudo/admin; javm keeps things user‑scoped and portable.

## License

Apache-2.0. See [LICENSE](LICENSE).
