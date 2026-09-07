---
name: javm
description: Use javm for Java projects that use .java-version, require a specific JDK, or need Maven, Gradle, java, javac, or other Java tools to run with an isolated, predictable JDK.
---

# javm

Use `javm` to run Java tooling without changing the global Java environment.

## Rules

- Prefer `javm exec` for builds, tests, checks, and other non-interactive commands.
- If `.java-version` exists, let `javm` resolve it automatically. Do not read or parse the file just to determine which selector to pass.
- Read `.java-version` only when its contents are relevant to the task, such as comparing, diagnosing, or changing the project's Java version.
- Do not modify global `JAVA_HOME`, `PATH`, or the system Java installation when `javm` can isolate the toolchain.
- Use `javm use` only when the current shell itself needs to change.

## Run commands

```sh
javm exec <command> [args...]
javm exec --jdk <selector> <command> [args...]
```

Examples:

```sh
javm exec java --version
javm exec ./gradlew test
javm exec mvn test

javm exec --jdk 21 java --version
javm exec --jdk temurin@21 ./gradlew build
```

`javm exec` sets `JAVA_HOME` and `PATH` only for the child process.

## Missing JDK

Prefer existing local JDKs before downloading:

```sh
javm ls
javm discover refresh
javm ls-remote <selector>
javm install <selector>
```

Use `ls-remote` and `install` only when no suitable local JDK is available.

## Diagnostics

Prefer javm commands over inspecting its files or environment manually:

```sh
javm ls --details
javm which [selector]
javm current
```

Use `javm <command> --help` when needed.
