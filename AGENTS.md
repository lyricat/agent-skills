# Repository Guidelines

## Project Structure & Module Organization

This repository contains standalone Codex skills rather than an application. Each top-level skill directory owns its instructions and metadata:

- `review-code/` contains the main `SKILL.md`, `agents/openai.yaml`, and detailed security references under `references/`.
- `simplify-code/` contains its `SKILL.md` and `agents/openai.yaml`.
- `README.md` documents installation and user-facing behavior.

There is currently no application source tree, test suite, or asset directory. Add `scripts/`, `references/`, or `assets/` inside a skill only when that skill needs them.

## Build, Test, and Development Commands

No build step or local server is required. Before committing, run:

```bash
git diff --check
python3 ~/.codex/skills/.system/skill-creator/scripts/quick_validate.py review-code
python3 ~/.codex/skills/.system/skill-creator/scripts/quick_validate.py simplify-code
```

`git diff --check` catches whitespace errors. Run `quick_validate.py` for each changed skill when the Codex `skill-creator` system skill is installed; it checks required structure and frontmatter.

## Coding Style & Naming Conventions

Write Markdown in short, direct sentences and use imperative instructions. Use two-space indentation in YAML. Name skill directories in kebab case, and keep the frontmatter `name` identical to the directory name, such as `review-code`. Keep `SKILL.md` focused; move substantial supporting detail into `references/` and link it with relative paths. Do not include credentials, usernames, or machine-specific paths.

## Testing Guidelines

There is no test framework or coverage target. Treat structural validation and representative behavior checks as required. When changing a workflow, invoke the skill on a small matching case and confirm its output constraints. Also check a nearby non-matching case when activation behavior changes. Record only checks actually run.

## Commit & Pull Request Guidelines

Recent history uses short imperative subjects, for example `Add simplify-code skill`. Keep each commit limited to one coherent skill or documentation change. Pull requests should state the affected skill, behavior changed, validation performed, and known limits. Link an issue when one exists. Screenshots are unnecessary unless rendered interface metadata changes.

## Agent-Specific Rules

Preserve required behavior, contracts, data, and safety properties before simplifying structure. Do not add speculative abstractions or thin wrappers. Keep `review-code` review-only unless the user explicitly authorizes edits, commits, pushes, or posted comments.
