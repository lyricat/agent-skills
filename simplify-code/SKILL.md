---
name: simplify-code
description: Simplify existing code from first principles by preserving required behavior and removing unnecessary concepts, branches, states, indirection, abstractions, configuration, and dependencies. Use when asked to simplify code, remove overengineering, flatten abstractions, eliminate thin wrappers or duplicate state, reduce a refactor, or make an implementation easier to reason about without changing its contract. Do not use for feature work, style-only formatting, or behavior-changing redesign unless simplification is explicitly part of the request.
---

# Simplify Code

## Objective

Treat simplification as constrained minimization:

1. Preserve required behavior, public contracts, data, safety properties, and supported environments.
2. Minimize the concepts, mutable states, branches, call hops, configuration modes, dependencies, and duplicated rules needed to satisfy those constraints.

Do not optimize for fewer lines alone. Shorter code is worse when it hides state, weakens validation, merges distinct concepts, or makes failure behavior harder to trace.

Do not replace explicit control flow with ternaries, chained callbacks, clever expressions, or compressed syntax unless the change removes a real state, branch, or duplicated rule. Syntax substitution alone is not simplification.

Modify only the requested scope. Do not combine simplification with unrelated cleanup or behavior changes.

## Establish the Constraints

Before editing:

- Read applicable repository instructions.
- Identify the requested files, symbols, or behavior. Do not silently widen the scope.
- Inspect callers, implementations, tests, documentation, configuration, serialized data, and public interfaces far enough to learn the actual contract.
- Record success, failure, retry, cancellation, cleanup, concurrency, and partial-completion behavior that must remain unchanged.
- Identify compatibility constraints such as API shape, CLI output and exit status, schemas, file formats, protocols, persisted data, and rolling upgrades.
- Preserve security boundaries, authorization, validation, resource limits, atomicity, idempotency, ordering, ownership, and data integrity.

Do not assume code is unused from text references alone. Account for reflection, registration, code generation, plugins, serialization, templates, build tooling, and runtime discovery when the repository uses them.

## Find Accidental Complexity

Look for concrete opportunities to remove:

- a function, type, interface, layer, or wrapper that only renames or forwards an existing operation;
- a single-implementation interface, factory, strategy, repository, or adapter that protects no stable boundary;
- a class that owns no state, lifecycle, invariant, substitutable behavior, or resource and merely groups functions;
- multiple representations or mutable sources of truth for the same state;
- DTO, model, and view-model conversion chains that copy the same data without enforcing a boundary or contract;
- branches that have the same preconditions, effects, and failure behavior;
- boolean parameters or mode flags whose combinations create unreachable, unsupported, or duplicate behavior;
- speculative extension points with no supported variation;
- configuration whose alternatives are not supported or reachable;
- queues, events, callbacks, workers, or concurrency without a required isolation, ordering, throughput, latency, or reliability property;
- duplicated domain rules that can share one exact implementation;
- custom machinery already provided with the required semantics by the language, standard library, or an existing repository primitive;
- obsolete compatibility paths, feature flags, migrations, fallbacks, imports, dependencies, and dead code whose removal is supported by repository evidence.

A simplification must reduce at least one named concept, state, branch path, call hop, configuration mode, dependency, or duplicated rule. Reject it when the reduction merely moves complexity elsewhere or introduces a broader abstraction than the behavior requires.

Small duplication can be simpler than a generalized helper. Extract shared code only when the callers implement the same rule and the abstraction has a stable name, contract, and ownership.

## Choose the Smallest Sound Transformation

Apply these rules:

- Inline a wrapper when it adds no invariant, validation, translation, side effect, lifecycle boundary, substitutability, or useful test seam.
- Keep a boundary when it owns policy, resource lifetime, external integration, transaction behavior, or a stable public contract.
- Merge branches only after proving their inputs, effects, errors, and cleanup are equivalent.
- Collapse duplicate state only after choosing one authoritative representation and tracing every reader and writer.
- Replace custom code with an existing primitive only when edge cases, errors, ordering, performance at established scale, and platform behavior match.
- Remove a configuration option only when no supported caller, stored value, deployment, or compatibility contract depends on it.
- Remove a dependency only after confirming no runtime, build, generated-code, plugin, or transitive requirement still uses it.
- Do not add a dependency merely to reduce local code. Count its API, update, security, build, and maintenance surface as complexity.
- Prefer direct code over a new abstraction when there is only one concrete behavior.

For example, inline a one-caller helper that only returns `parse(value)`. Keep a helper that also enforces a size limit and converts parser errors into the module's public error contract.

## Protect Behavior with Tests

Before changing formal code logic:

1. Run the narrowest relevant existing tests to establish a baseline.
2. If a preserved contract lacks coverage, add a characterization or regression test first and confirm it against the original implementation.
3. Implement one coherent simplification.
4. Run the focused tests again, then the broader relevant suite.

Honor repository-specific testing rules. Do not add tests for documentation, visual-only UI work, log additions, or copy changes unless repository policy requires them.

Do not weaken assertions, delete meaningful cases, or rewrite tests merely to accept changed behavior. Update a test only when its implementation details were intentionally coupled to a removed internal structure and its behavioral assertion remains intact.

## Implement and Verify

- Preserve existing naming and language idioms unless a name becomes false after the simplification.
- Delete newly unused code, imports, configuration, and dependencies within scope.
- Keep comments that explain non-obvious constraints; remove or update comments that describe deleted structure.
- Format changed files with the repository's configured formatter.
- Inspect the final diff for accidental behavior changes and unnecessary churn.
- Run relevant static checks, builds, and tests. Report only commands actually run and their exact outcomes.

If behavior, reachability, or compatibility cannot be established, make a smaller provable change or stop and name the exact missing evidence. Do not churn code to produce a non-empty diff.

## Report the Result

State concisely:

- which concepts, states, branches, call hops, configuration, or dependencies were removed;
- why required behavior remains intact;
- which tests and checks ran and their results;
- any precise behavior or environment that could not be verified.

When no safe simplification exists, say so and identify the constraint that makes the current complexity necessary.
