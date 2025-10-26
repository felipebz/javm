# Security Policy

## Supported versions

Security fixes are provided for:

* **Latest stable release**
  This is the version intended for general use.

* **Nightly / pre-release builds (for testing only)**
  Builds labeled `nightly` or similar are experimental. They may receive fixes, but they are not guaranteed to be stable
  or hardened. Use them at your own risk in non-production environments.

Older releases that are not the most recent tagged version are considered **out of support** and may not receive
security patches.

If you are running `javm` in production or in CI, you should:

1. Pin to an official tagged release.
2. Upgrade to new releases regularly.

## Reporting a vulnerability

If you believe you've found a security vulnerability in `javm` (code execution, privilege escalation, unsafe file
handling, etc.), please report it privately instead of opening a public issue.

### How to report

1. Open a private **GitHub Security Advisory** for this repository.

    * Go to the repository's "Security" tab → "Report a vulnerability".

2. Include:

    * Steps to reproduce
    * Your environment (OS, shell, `javm --version`)
    * Why you believe the behavior is a security risk (impact / severity)

If you can't use GitHub Security Advisories for any reason, you can open a minimal public issue saying "I have a
potential security report; how can I share details privately?" without disclosing the vulnerability itself.

Please do **not** post exploit details or proof-of-concept payloads in public issues, discussions, or PRs.

We will review, assess impact, and coordinate a fix and disclosure timeline.

## Scope (what counts as a `javm` security issue)

The following are considered in-scope security concerns for this project:

### 1. Local execution and environment tampering

`javm` is a CLI that:

* installs JDK distributions,
* manages local Java installations,
* and updates environment variables such as `JAVA_HOME` and `PATH`.

Because of that, **anything that lets an untrusted local user escalate privileges via `javm`, or inject unintended
content into a privileged session, is in scope.**

Examples:

* `javm` writing to a directory it shouldn't (e.g. outside the configured install root).
* `javm` changing environment variables in a way that persists across shells without the person's consent.
* `javm` tricking the shell into evaluating arbitrary attacker-controlled code.
* `javm` loading or executing binaries from a location the person did not ask it to use.

### 2. Unsafe file handling

Examples that are in scope:

* Path traversal (writing outside the managed JDK directory via `../` or symlink abuse).
* Insecure permissions on installed runtimes that allow other local users to replace binaries you trust.
* Race conditions in temporary directories that could allow another local process to inject/replace files used by
  `javm`.

### 3. Credential / secret exposure

If `javm` ever logs, prints, or otherwise exposes:

* tokens,
* personal access credentials,
* machine-specific secrets,
* or private filesystem paths that are considered sensitive in your environment,
  that's in scope.

### 4. Malicious downgrade / unexpected version switch

If you can coerce `javm` into silently selecting a different Java version than requested (for example, using
`javm use 21` but actually ending up with some attacker-controlled runtime), that's in scope.

## Out of scope / non-goals

For clarity, these issues are **not** considered vulnerabilities in `javm` itself:

### 1. Vulnerabilities in third-party JDK distributions

`javm` can download and install JDKs from multiple vendors (for example: different OpenJDK builds, vendor distributions,
etc.). Those JDKs are built and published by third parties. `javm` does not audit or patch their binaries.

If a vendor ships a JDK with a security flaw, that flaw is **not** a `javm` vulnerability. It's a vendor/runtime
vulnerability.

That said: if `javm` is fetching a runtime from the *wrong* source, or failing to verify what it downloaded in a way
that allows a malicious binary to be installed instead of the vendor's intended one, **that *is* in scope.**
(Example: no integrity check + MITM leading to arbitrary code execution.)

### 2. You ran `javm` as Administrator/root and it did exactly what you asked

If you explicitly run `javm` with elevated privileges and tell it to install or remove runtimes in a system-wide
directory, changes to that directory are expected.

`javm` is a development tool. It assumes you understand that installing runtimes affects how `java`, `javac`, etc.
resolve on your machine and in your CI.

### 3. Misconfiguration of your shell / CI environment

For example:

* Your CI workflow sources an untrusted script before calling `javm`.
* You `eval` arbitrary output without checking it.
* You allow untrusted contributors to run `javm use ...` in a privileged job.
  Those are pipeline / policy issues, not `javm` defects.

### 4. Vulnerabilities in Java code you run using a JDK managed by `javm`

If you install Java 21 with `javm`, then run your own application and that application has an RCE, that's not a `javm`
issue.

## How `javm` interacts with your environment

This section is here so security reviewers know what to look at.

### Shell integration

`javm` provides shell integration so it can:

* install a JDK if missing,
* switch the active Java version (`javm use 21`),
* and update `JAVA_HOME` / `PATH` in the *current shell session*.

Because shells cannot normally be modified by a child process after it exits, `javm` uses patterns like:

* printing instructions to `fd 3` or a temp file so the caller shell can `eval` them, or
* wrapper functions (PowerShell / bash / zsh / fish) that apply environment updates for you.

**Security note:**
If an attacker can trick your shell into running `javm` on their behalf (for example through a malicious CI script, or
through a sourced profile file), they may be able to point `JAVA_HOME` to a compromised JDK. That's equivalent to
"attacker controls the `java` binary you will execute". Treat that as high risk, especially in CI.

### Filesystem layout

`javm` typically installs JDKs under a controlled directory (for example, a per-user directory under your home, or a
location like `/etc/jvm` if you configured that on a container or system image). The correctness of this directory
matters:

* If it's writable by other local users, they can replace binaries.
* If it's on shared storage, ensure proper permissions and isolation.

### Temporary directories

`javm` may create temporary directories during install/update flows (for example, `$HOME/.cache/...` or a per-install
temp dir) before moving the final runtime into place.

Security best practices for reviewers:

* Check that temporary directories are created with restrictive permissions.
* Check that `javm` does not follow unexpected symlinks when moving/renaming.
* Check for predictable filenames in world-writable locations.

## Release integrity and attestation

Official `javm` releases are built by a controlled CI workflow in this repository. For each tagged release we publish:

* the release artifacts (binaries/archives),
* checksums,
* a provenance/attestation document (SLSA-style) that states which repository/workflow/ref built those artifacts,
* and an SBOM (software bill of materials).

You can verify that a binary you downloaded:

1. Matches the published checksum.
2. Has an attestation saying it was built by this repository’s release workflow (not a random laptop / fork).
3. Reports version/commit info (via `javm --version`) that lines up with that release.

Full details on how to verify a release, and what guarantees you get from the attestation, are documented in
`ATTESTATION.md`.

## Hardening recommendations for people running `javm`

You can reduce risk by doing the following:

1. **Pin versions in CI**
   In CI or production-like environments, run:

   ```bash
   javm install <exact-version>
   javm use <exact-version>
   ```

   instead of relying on "latest". Explicit version pinning makes builds reproducible and reduces supply-chain
   surprises.

2. **Do not eval untrusted output**
   Only source / eval the integration script from a trusted `javm` binary you installed yourself.
   Do not eval output from a `javm` binary provided by an untrusted PR or fork.

3. **Lock down the install directory**
   Ensure that the directory where `javm` installs JDKs is writable only by the intended account. Avoid world-writable
   paths.

4. **Review shell init hooks**
   If you add `javm` to your shell startup (e.g. `.bashrc`, `.zshrc`, PowerShell profile), treat that file like code.
   Anyone who can change those files can influence which `java` you run.

## Responsible disclosure / coordinated release

When a confirmed vulnerability is reported privately:

1. We investigate impact and assign a rough severity.
2. We prepare a fix or mitigation.
3. A new release is published that contains the fix.
4. A short security note is added to the release notes describing:

    * Which versions are affected,
    * What the impact is,
    * How to upgrade or mitigate.

If the issue is severe (for example, something that allows local privilege escalation or silent binary injection), we
may delay full technical details until most people have had a reasonable chance to upgrade.

## Hall of Thanks

We appreciate responsible security research and honest reports.

If you report a genuine security issue that results in a code or documentation fix, you may (if you want) be credited in
the release notes. Please tell us how you'd like to be mentioned.

## Final notes

* `javm` is a developer tool that manages language runtimes. By design, it can affect what binary (`java`, `javac`,
  etc.) will run on your system and in your CI pipelines. Treat it with the same level of trust you would give to tools
  like a package manager (`apt`, `brew`, `choco`) or a version manager (`nvm`, `rbenv`, etc.).
* Using `javm` implies you trust the source of the `javm` binary **and** the vendors of any runtimes you install with
  it.
* If that trust model does not work for your environment, you should vendor and audit binaries internally before letting
  `javm` install them.
