---
name: review-code
description: Review proposed code changes in local worktrees, staged changes, commits, branches, patches, or pull requests. Identify concrete defects, regressions, security or data-integrity risks, compatibility breaks, and meaningful gaps in regression coverage. When dependency manifests or lockfiles change, verify added or changed resolved dependency versions against current public vulnerability advisories and disclosed supply-chain incidents. Use when asked to review code, inspect a diff or PR, assess merge safety, or find bugs introduced by a change. Do not use for an open-ended whole-repository audit or style-only feedback unless the user explicitly requests it.
---

# Review Code

## Objective

Prevent incorrect behavior from entering the target branch. Optimize for verified, actionable findings rather than comment count.

Review only. Do not edit files, create commits, push branches, or post review comments unless the user explicitly asks for those actions.

## Apply the Finding Standard

Report a finding only when all of these statements are true:

1. The reviewed change introduces the problem or makes an existing problem reachable.
2. The problem violates intended behavior, an established contract, or a necessary safety property.
3. A concrete input, state, configuration, or execution path triggers it.
4. The impact is observable and worth the author's attention.
5. Repository evidence or authoritative external documentation supports the claim.
6. The finding points to the smallest useful changed code span.
7. The finding includes at least one verified concrete example showing the trigger and result.

Discard a candidate when it is only a style preference, a speculative concern, an unrelated pre-existing problem, or a generic request for more defensive code. Do not report an issue merely because a different implementation would be cleaner.

Treat absent tests as a finding only when repository policy requires the test or the omission creates a concrete merge risk. Otherwise record a precise untested behavior under residual risks, not as a defect.

For a disclosed dependency vulnerability, treat an exact resolved package, version, and source match against an affected range in a current public advisory as the concrete dependency state. Report application reachability separately as demonstrated or unknown; never infer it from the advisory alone. For a dependency security problem without an advisory, require concrete repository evidence that the changed resolution, integrity, or installation behavior violates a safety property.

## Review Workflow

### 1. Establish the Review Scope

Honor the user's explicit range, files, commit, branch, patch, or pull request. Do not silently widen it.

When reviewing local changes without an explicit range:

- Inspect repository status, including untracked files.
- Determine whether the intended unit is the working tree, staged changes, committed branch changes, or their combination.
- Use the merge base with the target branch for branch reviews.
- State the selected base and included changes. Ask only when competing choices would materially change the review.

Do not mix unrelated working-tree changes into a commit or pull-request review.

### 2. Learn the Intended Behavior

- Read all repository instructions that apply to the changed files.
- Inspect the complete diff and changed-file list before focusing on individual hunks.
- Derive intent from the user's request, issue or pull-request description, tests, documentation, public contracts, and surrounding code.
- Mark an inferred requirement as an inference. Do not invent requirements to support a finding.

### 3. Model Each Behavioral Change

For every meaningful behavior change, identify:

- inputs and preconditions;
- outputs, side effects, and state transitions;
- success, failure, retry, and partial-completion paths;
- callers, callees, implementations, and persistent data affected by the contract;
- invariants that must remain true.

Useful invariants include authorization, validation, atomicity, idempotency, ordering, ownership, resource lifetime, schema compatibility, protocol compatibility, and preservation of user data.

Trace beyond the diff only far enough to verify how the changed code is reached and consumed. Do not substitute a broad repository audit for analysis of the change.

### 4. Challenge the Invariants

Look for the smallest realistic case that breaks each relevant invariant. Pay particular attention to:

- boundary values, empty values, malformed input, and alternate configurations;
- error handling, retries, cancellation, cleanup, and partial failure;
- persisted data, migrations, serialization, precision, time, and ordering;
- concurrency, shared state, duplicate delivery, and re-entrancy;
- trust boundaries, permissions, injection, secret exposure, and unsafe defaults;
- public APIs, command-line behavior, configuration, and rolling-upgrade compatibility;
- newly expensive work on paths whose scale is established by repository evidence.

Use this list to generate questions, not findings. A finding still has to satisfy the finding standard.

### 5. Review Security-Sensitive Changes

For each changed hunk that reads, writes, validates, invokes, or configures an item below, inspect the corresponding group:

- Identity and access: authentication, account recovery, MFA, session and token issuance/expiry/rotation/revocation, server-side function/object/field authorization, tenant isolation, and delegated identity.
- Cryptography, secrets, and data: algorithms and libraries, randomness/nonces, password hashing, signature/TLS verification, key storage/rotation, newly exposed credentials, and sensitive data in URLs, logs, errors, caches, retention, or exports.
- Untrusted I/O: injection and context-specific encoding, SSRF/outbound requests/redirects, XSS/CSRF/CORS/cookies, paths/uploads/archives/symlinks, deserialization, parsers, and canonicalization.
- Abuse and availability: brute force/enumeration, replay/idempotency, rate limits/quotas, unbounded CPU/memory/I/O, regex/decompression, and retry/fan-out amplification.
- Configuration and runtime code: secure defaults, debug/admin exposure, fail-closed behavior, secret loading, subprocesses, native or unsafe code, and FFI.
- Detection and delivery code: security log content, audit event emission, checked-in CI/build logic, update logic, and artifact integrity verification.

When any group is triggered, read [Security Review](references/security-review.md) completely and follow it.

Trace attacker-controlled data and delegated authority from entry to effect. Do not describe the security review as complete unless every triggered group was verified. Put each unverified group, changed surface, and missing evidence under `Residual risks`.

### 6. Review Dependency Changes

When the diff changes a dependency manifest, lockfile, checksum file, workspace dependency file, or dependency source, read [Dependency Security Review](references/dependency-security.md) completely and follow it.

Perform a current public advisory check for every added dependency and every resolved dependency version or source changed by the diff, including transitive lockfile changes. If exact resolution or advisory access is unavailable, disclose the unverified dependencies under residual risks. Do not describe an incomplete check as safe.

### 7. Falsify Candidate Findings

Try to disprove each candidate before reporting it:

1. Trace one concrete execution path with specific values or state.
2. Inspect guards, callers, tests, configuration, and platform semantics that may prevent the failure.
3. Compare changed behavior with previous behavior when causality is unclear.
4. Run the narrowest safe test or static check that can confirm the claim when practical.
5. Consult authoritative documentation when library or platform behavior is uncertain.
6. Verify the concrete example against the code, test output, or cited documentation.
7. Discard the candidate if a necessary assumption remains unsupported or no verified concrete example can be stated.

Do not install dependencies, download or execute binaries, mutate external systems, or run destructive commands without authorization. Never claim a command passed unless it was run successfully.

### 8. Assign Priority

Combine impact and likelihood:

- `P0`: Broad, immediate, and catastrophic failure. Use rarely.
- `P1`: Likely severe production failure, security breach, or data loss; fix before merge.
- `P2`: Concrete defect under plausible conditions; normally fix before merge.
- `P3`: Real but limited-impact defect or edge case.

Do not raise priority solely because the affected component is important. Explain the actual consequence.

For dependency findings, combine advisory severity with dependency scope, resolved version, reachability evidence, exploit preconditions, and potential impact. Do not copy a CVSS score into the review priority without this analysis.

## Report the Review

Put findings first, ordered by priority and then by file. Use the language of the user's request.

Format each finding as:

```text
### [P2] Short title describing the broken behavior
`path/to/file.ext:line`

One compact paragraph explaining the triggering input or state, the resulting behavior,
and why the reviewed change causes it. Include the evidence needed to act on the finding.

Example: <specific input values, state transition, request and response, event sequence,
or dependency version match> -> <specific incorrect or unsafe result>.
```

Include the `Example` line in every finding. Use actual values or states supported by the reviewed code, test output, or cited documentation. For concurrency findings, give the ordered event sequence. For compatibility findings, give the old caller or input and its new result. Do not restate the abstract finding as the example and do not invent unverified behavior.

Keep the location to the smallest useful range and normally within changed lines. Use a related unchanged location only when no changed line can identify the cause, and explain the relationship.

After the findings, include only these concise sections when they add information:

- `Review scope`: base, head, files, or exclusions needed to understand coverage.
- `Validation`: commands or checks actually performed and their results.
- `Residual risks`: precise uncertainty, missing environment, or untested behavior that could not be resolved.

When there are no qualifying findings, write `No findings.` and still disclose the reviewed scope and any validation limits. Do not manufacture a low-value comment to avoid an empty result.

## Final Check

Before returning the review:

- Account for every file and meaningful behavioral hunk in scope.
- When a security group was triggered, account for its checks or disclose the changed surface and missing evidence under `Residual risks`.
- When dependency inputs changed, account for every added or changed resolved dependency and the advisory evidence checked for it.
- Recheck every finding against all seven parts of the finding standard.
- Ensure each title, location, trigger, and impact is specific enough for the author to verify.
- Ensure every finding contains an `Example` line with at least one verified trigger and result.
- Remove duplicates that share one root cause.
- Separate confirmed defects from residual uncertainty.
- State tests and tooling results exactly; disclose anything not run.
