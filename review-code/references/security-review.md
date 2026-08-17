# Security Review

Read this file only when the reviewed diff triggers a security group in `SKILL.md`.

Use this procedure for change-focused code review. Review only properties visible in changed source code, tests, or version-controlled configuration. Ignore properties that require observing a live deployment or organizational process unless the user explicitly expands the scope and supplies that evidence.

This procedure does not prove compliance or replace a full threat model, penetration test, infrastructure audit, privacy or legal review, or specialist review of a safety-critical domain. Keep findings limited to problems introduced or made reachable by the reviewed change. Record code paths that cannot be verified under `Residual risks`.

## Contents

- [Map the Security-Relevant Change](#map-the-security-relevant-change)
- [Require Evidence and Verify Safely](#require-evidence-and-verify-safely)
- [Review Identity and Authentication](#review-identity-and-authentication)
- [Review Sessions, Tokens, OAuth, and OIDC](#review-sessions-tokens-oauth-and-oidc)
- [Review Authorization and Tenant Isolation](#review-authorization-and-tenant-isolation)
- [Review Input, Encoding, and Injection](#review-input-encoding-and-injection)
- [Review Browser and Frontend Security](#review-browser-and-frontend-security)
- [Review APIs, Outbound Requests, and Real-Time Channels](#review-apis-outbound-requests-and-real-time-channels)
- [Review Files, Archives, Parsers, and Deserialization](#review-files-archives-parsers-and-deserialization)
- [Review Cryptography, Secrets, and Transport](#review-cryptography-secrets-and-transport)
- [Review Sensitive Data and Privacy Controls](#review-sensitive-data-and-privacy-controls)
- [Review Business Logic, Abuse, and Availability](#review-business-logic-abuse-and-availability)
- [Review State Integrity, Concurrency, and Exceptional Conditions](#review-state-integrity-concurrency-and-exceptional-conditions)
- [Review Code and Checked-In Configuration](#review-code-and-checked-in-configuration)
- [Review Security Logging Code](#review-security-logging-code)
- [Review Build and Release Code](#review-build-and-release-code)
- [Review Native, Unsafe, and Dynamic Execution](#review-native-unsafe-and-dynamic-execution)
- [Review AI and Tool-Using Features](#review-ai-and-tool-using-features)
- [Handle Specialized Domains](#handle-specialized-domains)
- [Decide and Report](#decide-and-report)
- [Standards Baseline](#standards-baseline)

## Map the Security-Relevant Change

1. Enumerate changed entry points, security decisions, privileged operations, persistent writes, external calls, build steps, and release steps.
2. Identify the principal whose authority is used, the asset affected, and the actor who can supply each input.
3. Trace each relevant path as `principal -> attacker-controlled source -> parsing and validation -> authorization -> security-sensitive sink -> effect`.
4. Include alternate paths such as batch endpoints, background jobs, queues, retries, imports, exports, administrative interfaces, caches, and service-to-service calls.
5. Mark every crossed boundary visible in the change: browser/server, anonymous/authenticated, user/admin, tenant/tenant, service/service, process/process, source/build/release, and application/third party.

Treat request fields, headers, cookies, files, URLs, CLI arguments, environment variables, configuration, database rows, cache entries, queue messages, repository metadata, third-party responses, and model output as untrusted whenever an actor outside the current trust boundary can influence them. Do not trust an input solely because the code labels its source as internal.

Use the repository's documented threat model and security requirements when present. If a required actor, asset, or trust decision cannot be established from repository evidence, state the exact missing fact under `Residual risks`.

## Require Evidence and Verify Safely

- Trace a concrete input or state through the changed code to an observable security effect. A weakness name or scanner alert alone is not a finding.
- Inspect all guards and alternate callers that may prevent the path. Check whether enforcement happens in a trusted server-side or privileged layer.
- Separate attacker control, reachability, exploit preconditions, and impact. State each as demonstrated or unknown.
- Use repository tests, installed project-approved analyzers, and authoritative documentation. Treat clean scanner output as evidence only for what that scanner actually checked.
- Do not install tools, download or execute binaries, use real credentials, exploit systems outside the supplied test scope, or send proprietary code or secrets to third-party services without authorization.
- Redact credentials, tokens, personal data, and private endpoints from commands and output. Never authenticate with a discovered credential merely to test whether it is live.
- Put an unsupported security concern under `Residual risks`; do not convert uncertainty into a finding.

When a changed line exposes a likely real credential, cite its location and type without reproducing the value. Explain that removal from the current file does not revoke a credential already copied into history, logs, caches, or artifacts. Recommend revocation or rotation only when the value is not clearly a placeholder, public identifier, test fixture, or public key.

## Review Identity and Authentication

- Verify identity enrollment, linking, credential change, recovery, and deletion against the intended account and a verified channel.
- Verify that login, recovery, and enrollment responses do not disclose whether an account exists unless the product explicitly permits that disclosure.
- Verify password checks and storage through an established password-hashing API with project-approved parameters; reject plaintext storage and reversible encryption of passwords.
- Verify rate limits and anti-automation controls for login, recovery, OTP, MFA, and credential-stuffing paths. Ensure lockout behavior cannot cheaply deny service to another user.
- Verify MFA enrollment, replacement, disabling, fallback, and recovery. No weaker recovery path may bypass the intended factor requirement.
- Verify one-time challenges and recovery codes are unpredictable, purpose-bound, short-lived, single-use, and invalidated after success or replacement.
- Verify passkey or cryptographic authentication challenges bind the expected relying party, origin, account, and ceremony, and cannot be replayed.
- Verify reauthentication for security-sensitive changes when required, including password, MFA, payout, recovery channel, or administrative changes.

## Review Sessions, Tokens, OAuth, and OIDC

- Verify session identifiers are unpredictable, are not accepted from URLs, and rotate after authentication, privilege changes, recovery, and other fixation boundaries.
- Verify idle and absolute expiry, logout, account-disable, password-reset, and compromise-response behavior. Check server-side invalidation where immediate revocation is required.
- Verify session cookies use the narrowest practical domain and path and appropriate `Secure`, `HttpOnly`, and `SameSite` attributes.
- Verify state-changing browser requests have an effective CSRF defense. Do not treat CORS or `SameSite` alone as universal CSRF protection.
- For self-contained tokens, pin permitted algorithms and keys; verify signature, issuer, audience, subject, purpose, expiry, not-before time, and required claims before use.
- Reject attacker-selected verification keys or key URLs. Verify key rotation does not accept removed, wrong-tenant, or wrong-environment keys.
- Do not treat encoded or signed token contents as confidential. Avoid placing secrets or unnecessary sensitive data in token claims.
- For OAuth or OIDC, verify exact redirect URI matching, `state`, `nonce` where applicable, PKCE, authorization-code one-time use, issuer/client binding, and least-privilege scopes.
- Bind account linking to a stable verified provider identity. Do not rely on an unverified email address or display name as the sole identity key.
- Keep authorization codes and access, refresh, and identity tokens out of URLs, logs, referrers, browser storage, and error messages unless the protocol explicitly requires the location and its risks are controlled.

## Review Authorization and Tenant Isolation

- Enforce authorization in a trusted layer on every operation; deny by default and do not rely on hidden UI, client claims, route naming, or network location.
- Check function-level, object-level, and field-level access for reads and writes, including bulk operations, search, export, import, history, and indirect references.
- Derive tenant scope from trusted identity context. Do not accept a tenant, owner, role, price, status, or permission from the client without reauthorization.
- Apply tenant filtering to queries, caches, indexes, object storage paths, queues, logs, analytics, and generated exports. Check identifiers that are globally unique but still unauthorized.
- Preserve the originating principal when one service acts through another. Prevent a service credential from turning a user request into broader authority.
- Verify authorization after ownership, membership, role, policy, account state, or resource state changes. Account for stale sessions, tokens, and caches.
- Protect administrative, support, impersonation, and break-glass paths with explicit authorization, audit, and bounded duration.
- Verify authorization before revealing existence, metadata, validation details, or timing differences for protected resources when those differences matter.

## Review Input, Encoding, and Injection

- Decode or normalize once into a canonical representation before validation. Check inconsistent URL, Unicode, path, JSON, XML, and proxy parsing.
- Validate expected type, structure, length, range, cardinality, encoding, and allowed values at the trust boundary. Reject ambiguous duplicate fields where parser behavior differs.
- Use parameterized APIs for SQL, NoSQL, LDAP, XPath, and other interpreters. Verify identifiers or query fragments that cannot be parameterized against a strict allowlist.
- Avoid invoking a shell with untrusted data. When process execution is required, pass fixed executables and separated arguments and verify environment, working directory, and inherited handles.
- Apply context-specific output encoding at the final sink for HTML, attributes, JavaScript, CSS, URLs, headers, logs, and templates. Sanitization for one context does not protect another.
- Sanitize permitted rich content with a maintained policy and verify dangerous URLs, event handlers, embedded content, and mutation after sanitization.
- Check server-side templates, expression languages, regular expressions, format strings, and log fields for code, template, ReDoS, or record-injection paths.
- Check integer conversion, truncation, sign, unit, precision, and overflow before the value controls allocation, authorization, money, time, or bounds.

## Review Browser and Frontend Security

- Treat client-side validation and authorization as usability controls only; repeat security enforcement in a trusted backend.
- Check changed DOM sinks, HTML rendering, URL navigation, script creation, and dynamic imports for XSS or script execution.
- Verify cross-origin policy against explicit trusted origins. Do not reflect arbitrary origins with credentials or return sensitive data under wildcard access.
- Verify `postMessage` sender origin and message schema. Avoid wildcard target origins for sensitive messages.
- Check cookies, CSP, frame restrictions, content-type protections, referrer policy, HTTPS enforcement, and origin separation when the change affects their assumptions.
- Protect state-changing actions from CSRF and clickjacking where the browser can supply ambient authority.
- Keep sensitive values out of URLs, DOM, browser caches, service-worker caches, local storage, analytics, and third-party scripts unless required and protected.
- Verify externally hosted active content is immutable and integrity-checked, or document why an equivalent control exists.
- Check open redirects, reverse tabnabbing, unsafe downloads, MIME confusion, and cache deception when routing or response construction changes.

## Review APIs, Outbound Requests, and Real-Time Channels

- Enforce request schemas, content types, size limits, property allowlists, and response-field allowlists. Prevent mass assignment and excessive data exposure.
- Apply authorization inside each REST handler, RPC method, GraphQL resolver, subscription, and real-time message handler, not only at connection or gateway setup.
- Bound pagination, filtering, sorting, GraphQL depth or complexity, batch size, subscriptions, and response size.
- Authenticate webhooks or messages with a verified signature or equivalent channel identity; bind verification to the raw message, expected sender, timestamp, and replay window.
- Make retries and duplicate delivery safe with idempotency or deduplication tied to the correct principal and operation.
- Treat third-party API responses as untrusted input. Validate schemas, authorization context, redirects, content types, and error behavior before consuming them.
- For outbound URLs, parse once and allow only required schemes, hosts, ports, and destinations. Revalidate every redirect.
- Reject ambiguous URL syntax and disallowed loopback, private, link-local, or metadata destinations when the repository's security policy requires that restriction.
- Apply connection, read, total-time, redirect, response-size, and decompression limits to outbound calls.
- For WebSocket or WebRTC changes, verify origin and peer identity, signaling authorization, per-message permissions, reauthentication, resource limits, and confidentiality of signaling and media metadata.

## Review Files, Archives, Parsers, and Deserialization

- Resolve and validate paths against an intended root. Account for absolute paths, traversal, alternate separators, Unicode, case folding, symlinks, hard links, and validation-to-use races.
- Generate server-side storage names and set restrictive ownership and permissions. Keep uploads outside executable and publicly served locations unless explicitly intended.
- Validate upload count and size before buffering. Verify declared extension and MIME type do not substitute for content-aware handling where content is security-sensitive.
- Serve downloads with safe content type and disposition, authorization, and filename encoding. Prevent user content from becoming active application content.
- Reject archive entries that escape the extraction root, special files, links, excessive expansion, excessive nesting, or excessive file counts.
- Configure XML and similar parsers to disable external entities, remote schemas, and other unused active features.
- Do not deserialize untrusted data into attacker-selected types or code-bearing objects. Prefer data-only schemas and explicit type allowlists.
- Check duplicate keys, unknown fields, trailing data, parser differentials, and canonicalization when data is validated, signed, cached, or routed by different components.
- Bound image, document, media, compression, and recursive parser work. Remove sensitive metadata when repository requirements demand it.
- Create temporary files safely, avoid predictable names and unsafe shared permissions, and clean them on success, failure, and cancellation.

## Review Cryptography, Secrets, and Transport

- Use maintained cryptographic libraries and project-approved algorithms and parameters. Do not accept custom primitives or protocols without specialist evidence.
- Use a cryptographically secure random generator for keys, tokens, challenges, salts, and nonces. Verify nonce uniqueness where the mode requires it.
- Use authenticated encryption or a separate integrity mechanism where tampering matters. Bind ciphertext or signatures to the correct context, version, principal, and purpose.
- Verify password hashing, key derivation, MAC, and signature APIs use the intended primitive, input, key, and comparison behavior.
- Verify code selects keys and secrets for the correct purpose and environment, does not hard-code them or copy them into artifacts, and handles rotation and revocation when the changed path depends on them.
- Verify signature and certificate validation checks the complete trust decision, including expected identity or hostname, chain, validity, algorithm, and failure result.
- Do not disable TLS validation or silently downgrade protocols. Verify internal and service-to-service traffic when its confidentiality or integrity is relied upon.
- Distinguish encoding, hashing, encryption, signing, and redaction; none is a substitute for another.
- Check caches, crash handling, diagnostics, and observability code for plaintext keys, secrets, or decrypted data.

## Review Sensitive Data and Privacy Controls

- Identify each changed sensitive data field and its classification, allowed readers, purpose, storage locations, recipients, retention, and deletion behavior.
- Collect, store, return, and export only the fields required for the operation. Apply field-level authorization before masking or serialization.
- Keep secrets, session material, and sensitive personal data out of URLs, logs, errors, metrics, traces, analytics, notifications, and third-party integrations.
- Verify cache controls and purge behavior implemented by browser, proxy, application-cache, and client-storage code changed in the diff.
- Protect data in transit and at rest when the changed code selects or configures that protection. Verify decryption does not broaden access.
- Apply retention and deletion logic to derived copies, indexes, exports, attachments, and queued work where required.
- Prevent production data and credentials from entering fixtures, snapshots, examples, telemetry, or lower environments.
- Verify redaction fails safely when fields are renamed, nested, malformed, or newly added.
- Treat missing legal, regulatory, or privacy requirements as a scope limit requiring specialist review, not as proof of a code defect.

## Review Business Logic, Abuse, and Availability

- Define the server-authoritative state machine and invariants for value, ownership, quota, entitlement, approval, and irreversible actions.
- Prevent step skipping, parameter replay, stale-state use, client-calculated price or privilege, negative quantities, duplicate redemption, and cross-account reuse.
- Check brute force, enumeration, scraping, fake-account creation, inventory reservation, purchasing, payout, messaging, and other sensitive flows for practical automation abuse.
- Apply rate and quota controls to the correct actor, tenant, resource, and operation. Account for distributed callers, identifier rotation, and proxy-derived addresses.
- Bound request size, collection size, pagination, recursion, regex work, parsing, decompression, image processing, database scans, fan-out, and concurrency.
- Set timeouts, cancellation, backpressure, retry budgets, and circuit-breaking behavior where unavailable dependencies can amplify work.
- Verify expensive validation occurs after cheap rejection when reordering does not weaken security.
- Require concrete repository scale or a reproducible resource path before reporting denial of service; generic statements that work is unbounded are insufficient.

## Review State Integrity, Concurrency, and Exceptional Conditions

- Keep authorization checks and protected writes within an atomic or otherwise race-safe decision boundary where state can change concurrently.
- Check double-spend, duplicate creation, quota bypass, stale ownership, sequence reuse, and validation-to-use races with an explicit event ordering.
- Preserve invariants across partial failure, cancellation, timeout, retry, rollback, and process restart. Verify cleanup does not delete another principal's resource.
- Fail closed when authentication, authorization, signature verification, policy lookup, or security configuration fails. Distinguish this from availability paths where a safe degraded mode is documented.
- Verify exceptions and alternate return paths do not skip checks, commit partial sensitive state, expose diagnostics, or turn an error into success.
- Check integrity and anti-rollback behavior for signed data, migrations, state snapshots, update metadata, and versioned protocols.
- Verify time comparisons, expiry, clock skew, integer arithmetic, money, and precision at their boundary values.

## Review Code and Checked-In Configuration

- Make code defaults secure. Remove or protect sample credentials, debug modes, test routes, documentation endpoints, excessive health details, directory listings, and administrative functions.
- Keep security behavior consistent across checked-in configuration, environment-variable parsing, feature flags, and command-line options. Reject unsafe missing or malformed values.
- Do not hard-code secrets or place them in generated output, process arguments, diagnostics, or artifacts.
- Verify file, process, and database privileges requested by the code are no broader than the changed operation requires.
- When container or sandbox configuration changes, review its declared user, capabilities, privileged mode, device access, host sockets, mounts, and writable paths.
- Verify a convenience fallback cannot select stronger credentials, broader authority, or production data.
- Keep source-control metadata and unintended source or configuration out of artifacts produced by changed build code.

## Review Security Logging Code

- Record security-relevant authentication, recovery, authorization, administrative, key, configuration, data-export, and integrity events when repository requirements call for them.
- Include enough actor, target, time, request or correlation identifier, action, and outcome to investigate an event without logging secrets or excessive personal data.
- Encode or structure attacker-controlled fields to prevent forged log records, terminal control sequences, or broken downstream parsing.
- Prevent code paths that read logs or audit records from crossing user or tenant authorization boundaries.
- Verify logging or metrics failure does not bypass a security decision, expose data, or exhaust the application through retry or backpressure.
- Keep audit recording consistent with the protected state change when loss or reordering would invalidate accountability.

## Review Build and Release Code

- Apply the dependency-specific procedure required by `SKILL.md` whenever dependency resolution or source changes.
- Treat pull-request content, branch and tag names, commit messages, issue text, generated files, and build metadata as untrusted input to checked-in CI code.
- Prevent changed workflows from passing secrets or privileged credentials to untrusted code or artifacts.
- Pin CI actions, build images, toolchains, plugins, and remote includes to an immutable and verified identity where repository policy requires reproducibility or integrity.
- Review build scripts, lifecycle hooks, code generation, package publication, and updater code as executable security boundaries.
- Keep secrets out of build arguments, logs, caches, artifacts, source maps, and test reports.
- Verify changed signing, hashing, update, or artifact-verification code binds artifacts to the expected product, version, platform, and channel.
- Reject overwrite, rollback, or cross-environment substitution in changed publication or updater logic.
- Review base images, OS packages, vendored code, compiler plugins, and generated artifacts in addition to application package manifests.

## Review Native, Unsafe, and Dynamic Execution

- Prefer memory-safe APIs and languages at the boundary. Review every changed unsafe block, pointer operation, manual allocation, and FFI call with its size, ownership, lifetime, and thread assumptions.
- Check bounds, integer conversion, allocation size, nullability, initialization, use-after-free, double-free, format strings, and data races.
- Validate data on both sides of an FFI, IPC, plugin, or serialization boundary; neither side may assume the other preserved types or lengths.
- Keep dynamic code loading, reflection, `eval`, templates, macros, plugins, and user scripts away from untrusted input unless a documented sandbox and capability model contains them.
- Verify subprocesses, generated code, and plugins inherit only required environment variables, handles, files, and privileges.
- Evaluate the privileges reachable after a sandbox escape; the existence of a sandbox is not evidence that arbitrary code execution is safe.

## Review AI and Tool-Using Features

- Treat user prompts, retrieved documents, web content, tool results, memory, and model output as untrusted data, including instructions embedded inside them.
- Keep system policy and authorization outside attacker-controlled context. Untrusted text must not grant tools, change identity, reveal secrets, or override approval requirements.
- Authorize every tool action against the originating principal and current resource; do not let the model act as a confused deputy.
- Validate and constrain tool names, arguments, paths, URLs, recipients, and side effects in deterministic code before execution.
- Use least-privilege tools and credentials, explicit approval for high-impact actions, and isolation for generated or model-selected code.
- Encode or validate model output before placing it in HTML, SQL, shell commands, file paths, templates, or privileged APIs.
- Prevent cross-user or cross-tenant leakage through prompts, retrieval indexes, caches, traces, evaluations, fine-tuning data, and conversation memory.
- Bound tokens, recursion, tool calls, parallelism, retries, output size, and spend. Preserve an auditable link between request, model decision, approval, tool call, and result.
- Do not claim prompt injection is fully solved by prompt wording; verify deterministic authorization and containment at the action boundary.

## Handle Specialized Domains

Apply the relevant project or industry standard when a change affects payments, financial ledgers, cryptographic protocols, smart contracts, consensus systems, kernels or drivers, mobile platform IPC and deep links, firmware update chains, medical or industrial control, or another regulated or safety-critical domain.

If the repository does not provide the required invariants or domain evidence, identify the exact changed code under `Residual risks`. Do not claim universal security coverage from this checklist.

## Decide and Report

Report a security finding only when the main finding standard is satisfied. Include:

- the actor or principal and attacker-controlled input or state;
- the complete reachable path through the changed code;
- the missing or bypassed security boundary;
- the unauthorized effect and concrete impact;
- the guard, test, documentation, or platform behavior used as evidence;
- a verified `Example` with specific values or an ordered event sequence.

Do not assign priority from a weakness category, scanner severity, or theoretical worst case alone. Use the demonstrated privileges, exposed asset, preconditions, blast radius, detectability, and recovery cost.

Use examples in this form:

```text
Example: <specific principal and input/state> -> <changed path and missing control>
-> <specific unauthorized disclosure, modification, execution, or resource exhaustion>.
```

For concurrency, replay, or revocation problems, list events in order. For authorization problems, name both the acting principal and the resource owner or tenant. For information exposure, name the exact field or metadata disclosed without reproducing secret values.

Before finishing, account for each triggered group as verified, not affected after tracing, or unverified with the changed surface and missing evidence. Record tools, commands, authoritative sources, and their actual results under `Validation`.

## Standards Baseline

Use current official editions at review time and cite the edition actually consulted. This reference is organized from and cross-checked against:

- [OWASP Application Security Verification Standard](https://owasp.org/www-project-application-security-verification-standard/) — detailed application control requirements; version 5.0.0 was the stable baseline used to write this file.
- [OWASP Top 10](https://owasp.org/Top10/) — current web application risk cross-check; use it as an awareness list, not proof of verification.
- [OWASP API Security Project](https://owasp.org/www-project-api-security/) — API-specific authorization, resource consumption, business-flow, SSRF, and third-party API risks.
- [NIST Secure Software Development Framework](https://csrc.nist.gov/projects/ssdf) — use only its code-visible build, integrity, release, and vulnerability-response controls for this review scope.
