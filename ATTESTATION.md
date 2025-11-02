# Release attestation and provenance

`javm` publishes official release artifacts (binaries/archives for each OS and architecture) using a controlled CI
pipeline. The goal is that you can verify that the binary you are installing is:

* built from this repository,
* built from a specific commit/tag,
* built in CI (not on someone's laptop),
* and not modified after the build.

This document explains what we publish, how to verify it, and what the guarantees are.

## What we publish

For each tagged release, we publish:

* **Artifacts**: Example: `javm-linux-x86_64.tar.gz`, `javm-windows-x86_64.zip`, etc.

* **Checksums**: You can use these to confirm integrity (that the file you downloaded matches what we released).

* **Provenance / attestation**: This is machine-readable build metadata (SLSA-style provenance) that states:

    * which repository built the artifact,
    * which workflow built it,
    * and which ref it ran on.

  This attestation is signed by GitHub Actions OIDC so you can prove the artifact actually came from our CI pipeline for
  this repo.

* **SBOM** (Software Bill of Materials):  A dependency inventory for the built binary. You can archive it for
  audit/compliance, or feed it into scanners.

Nightly / pre-release / experimental builds does not include full attestation and should not be treated as
production-grade.

## How to verify a release

When you download an official `javm` binary:

1. **Verify integrity**

    * Calculate the checksum (e.g. SHA-256) of the file you downloaded.
    * Compare it to the checksum we published for that release.
    * If it doesn't match, stop — treat the file as untrusted.

2. **Verify provenance / attestation**

    * Use `gh attestation verify` (or equivalent tooling) against the downloaded artifact.
    * This confirms that:
        * the artifact was built in CI for this repository,
        * using the expected GitHub Actions workflow,
        * and signed by GitHub's OIDC identity for that workflow.
    * This protects you from "someone uploaded a random binary and called it javm.exe".

   Note: today `gh attestation verify` shows you the repo/workflow/ref that produced the artifact, but it may not print
   the commit SHA directly in the summary. You can still retrieve the full attestation bundle and inspect it if you need
   to map the artifact to an exact git commit.

3. **Check the embedded version info**

    * Run:

      ```bash
      javm --version
      ```

    * `javm` embeds build metadata (version, commit, build timestamp).
    * Ensure the reported version/commit matches:

        * the release/tag you think you downloaded, and
        * the commit recorded in the attestation (if you inspected the full bundle).

4. **Archive evidence**

    * Store the binary, its checksum, the SBOM, and the attestation/provenance JSON in your internal artifact mirror.
    * This gives you traceability later (for audits and incident response).

If any of these checks fail, do not trust the binary.

## What this does *not* guarantee

**It does not audit the JDKs you install later.**\
`javm` can install different Java distributions (Temurin, Zulu, etc.). Those runtimes are published by external
vendors. We do not audit or warranty those vendor binaries.

**It does not make forks "official."**\
A fork of the project (or a random person) can build their own binary and even generate their own attestation for
*their* repo. Only releases from this repository, built and attested by this repository's CI, are considered official.

**It does not sandbox `javm`.**\
`javm` intentionally changes `JAVA_HOME` / `PATH` and controls which `java` you will execute. If an attacker can trick
your shell or CI into running an untrusted `javm`, they can point you at a malicious JDK. Treat `javm` with the same
trust level as a package manager.

## Recommended policy for CI / regulated environments

If you are using `javm` in CI, production builds, or a restricted workstation image:

1. Pin to an explicit release version instead of "latest".
2. Require checksum + attestation verification before allowing the binary.
3. Check `javm --version` and ensure it matches the attested commit/tag.
4. Save the SBOM and attestation next to the approved binary internally.
5. Block any `javm` binary that cannot be verified this way.

## Reporting problems

If you believe:

* a published artifact does not match its attestation,
* an attestation references an unexpected workflow or ref,
* or an attacker is impersonating an official release,

please report it privately using the process in [`SECURITY.md`](SECURITY.md) instead of opening a public issue with full details.

This helps us confirm and fix the problem without giving attackers a head start.
