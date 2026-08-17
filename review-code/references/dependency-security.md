# Dependency Security Review

Read this file only when the reviewed diff changes dependency declarations, resolution, or source.

## Determine the Exact Dependency Delta

1. Identify changed manifests, lockfiles, checksum files, workspace files, vendoring metadata, and dependency source declarations.
2. Compare base and head versions of those files. Enumerate added packages and changed versions or sources, including transitive changes recorded only in a lockfile.
3. Prefer the resolved package version from a lockfile or resolved module graph over a declared version range.
4. Record the ecosystem, normalized package name, resolved version, source, direct or transitive status, and production, development, optional, or unknown scope when available.
5. Treat a manifest without a lockfile or resolved graph as version-uncertain. Do not assume the newest matching version will be installed.
6. Review each project independently in a monorepo. Do not combine packages from unrelated lockfiles.

Do not report a removed dependency unless it remains in the resolved graph or its removal creates another concrete problem.

When only a declared range is available, check whether public affected ranges intersect it, but keep the result version-uncertain. Report the uncertainty as a residual risk unless the declaration necessarily selects an affected version.

## Check Current Public Security Evidence

Treat vulnerability information as time-sensitive. Query current sources during the review instead of relying on model memory.

Use the project's installed, non-mutating ecosystem scanner when available. Examples include:

```text
Go:    govulncheck -json ./...
npm:   npm audit --json
pnpm:  pnpm audit --audit-level high --json
Yarn:  yarn npm audit --json
Bun:   bun audit --json
```

For other ecosystems, use the repository's configured scanner or an installed standard scanner. Use OSV as a cross-ecosystem check or fallback when it accepts the exact resolved package and version.

Corroborate a reported issue with a public primary source such as an official ecosystem vulnerability database, OSV entry, GitHub Security Advisory, CVE record, upstream maintainer advisory, or project incident report. Link the source in the finding.

Check more than CVE records. Include public advisories for malicious packages, compromised releases, dependency confusion, package takeover, or install-time malware when they identify the affected package and version or another concrete indicator.

Do not:

- invent advisory IDs, affected ranges, severity, fixed versions, reachability, or exploitability;
- treat a package-name match as a vulnerable-version match;
- run audit fix, update, install, or dependency-editing commands during review;
- install a missing scanner or download a binary without authorization;
- expose registry credentials, tokens, or private dependency URLs in output.

A scanner's non-zero exit status may mean it found vulnerabilities. Parse its output before deciding that the scan failed.

## Decide Whether to Report a Finding

Report a dependency finding only when the reviewed change introduces the dependency, version, source, or installation behavior and one of these conditions holds:

- A public advisory or incident report identifies the exact package and covers the resolved version or source.
- Concrete repository evidence demonstrates a source-integrity, resolution, or install-time security failure even though no public advisory exists.

In both cases, describe the security impact and cite the advisory or repository evidence that establishes it.

For each finding, state:

- resolved version and affected range;
- direct or transitive status;
- production, development, optional, or unknown scope;
- demonstrated reachability, not shown reachable, or unknown reachability;
- fixed version only when the source specifies one;
- advisory identifiers and links;
- the smallest practical remediation.

Do not report an unchanged vulnerable dependency as introduced by the reviewed change. Mention it only if the user expanded the scope beyond change review.

Treat unexpected Git URLs, remote tarballs, checksum changes, lifecycle scripts, or abandoned packages as investigation signals, not proof of compromise. An unpinned mutable source or removal of integrity validation can qualify when the change directly permits artifact substitution; an unusual but pinned and verified source does not qualify by itself. Otherwise place the uncertainty under residual risks.

If the lockfile, scanner, network, private registry, or build context prevents a reliable check, name the affected dependencies and the missing evidence under residual risks. Never conclude that no advisory exists merely because one scanner returned no result.

## Report Dependency Evidence

Locate the finding on the changed manifest or lockfile line that introduces the affected resolution. Include the package, resolved version, advisory ID, affected range, impact, reachability status, and fixed version when known in one compact finding.

Use the exact resolution match as the required concrete example: show the resolved version, the advisory's affected range, and the resulting disclosed exposure. Add an exploit precondition or reachable call path only when the scanner or cited advisory establishes it; otherwise state that application reachability is unknown.

Record the scanners and advisory sources actually checked under `Validation`. Include the check date because public advisory data changes over time.

For a confirmed malicious or compromised release, include incident-response consequences when relevant: clean dependency reinstall, credential rotation, CI runner or cache invalidation, and preservation of install logs. Do not reduce a supply-chain compromise to an ordinary version bump.
