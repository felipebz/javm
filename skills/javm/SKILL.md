---
name: javm
description: Use javm when a Java project contains .java-version, explicitly uses javm, requires a specific JDK, or needs Maven, Gradle, java, javac, or other Java tools to run with a predictable JDK without changing the global Java environment.
---

# javm

`javm` is a cross-platform Java version manager. This playbook provides operational rules for coding agents to select and run Java/JDK toolchains safely and predictably.

## Core Rules

1. **`.java-version` is the Source of Truth**: If a project contains a `.java-version` file, treat this selector as the project's intent. Do not modify `.java-version` unless the task explicitly requires altering the Java version of the project.

2. **Prefer `javm exec` for Non-Interactive Runs**: To execute builds, tests, checks, and other tools non-interactively without polluting the agent's environment, prefer `javm exec` over `javm use`.

   **Basic Syntax**:
   ```sh
   javm exec [selector] -- <command> [args...]
   ```

   **Examples**:
   * Leveraging the project's `.java-version`:
     ```sh
     javm exec -- java --version
     javm exec -- ./gradlew test
     javm exec -- mvn test
     ```
   * Passing an explicit version:
     ```sh
     javm exec 21 -- java --version
     javm exec temurin@21 -- ./gradlew build
     ```
   
3. **Isolate Environment Modifications**: Do NOT modify the global `PATH`, global `JAVA_HOME`, system Java installation, or OS-equivalent configurations when `javm` can solve the issue in isolation.

4. **Context Differentiation**:
   * **`javm exec`**: Sets `JAVA_HOME` and `PATH` for the child process only.
   * **`javm use`**: Modifies the current shell session.
   * **`javm default`**: Sets the selector used for newly initialized shells.
   * **`.java-version`**: Project selector.

## When the required JDK is missing

1. Check local JDKs with `javm ls`.
2. Use `javm discover refresh` if an existing unmanaged JDK may already be present.
3. Use `javm ls-remote <selector>` only when a download is actually needed.
4. Install with `javm install <selector>` only if no suitable local JDK is available.

## Diagnostics commands

Rely on these existing CLI commands when inspecting the state. If you need more details, use `javm <command> --help` or `javm --help`.

```sh
# Lists installed JDKs
javm ls
javm ls --details

# Shows the path of the selected JDK
javm which
javm which <selector>

# Shows the JDK specifically selected in the current shell
javm current
```
