# Migrating from Samuel v2 to v3

Samuel v3.0.0 reshapes the CLI for clarity. The hot path is `init` → `run` → `update`. Discovery collapses into one verb. Power-user commands move under `admin`. The autonomous Ralph Wiggum loop gets a guessable name.

**v2 scripts keep working through v3.0.x.** Every renamed command stays registered as a hidden alias that prints a one-line redirect on stderr and forwards to the new handler. Set `SAMUEL_NO_DEPRECATION=1` (or pass `--no-deprecation`) to silence the redirect noise in CI logs.

The shim removes in **v3.1.0**, roughly three months after v3.0.0 ships. Plan to update your scripts before then.

## Command map

### Discovery (collapsed into `samuel ls`)

| v2 | v3 |
|---|---|
| `samuel list` | `samuel ls` |
| `samuel list --available` | `samuel ls --all` |
| `samuel list --type frameworks` | `samuel ls --type framework` (singular) |
| `samuel search react` | `samuel ls react` |
| `samuel search react --type fw` | `samuel ls react --type framework` |
| `samuel search react --limit 10` | `samuel ls react --limit 10` |
| `samuel info framework react` | `samuel ls react --detail` (type inferred) |
| `samuel info framework react --preview 20` | `samuel ls react --detail --preview 20` |
| `samuel info framework react --no-related` | `samuel ls react --detail --no-related` |

`ls --detail` infers the type from the registry. If a name is ambiguous (none today, but possible in the future), pass `--type` to disambiguate.

### Component install (type now optional)

| v2 | v3 |
|---|---|
| `samuel add framework react` | `samuel add react` (inferred) |
| `samuel add language typescript` | `samuel add typescript` (inferred) |
| `samuel add lang rust` | `samuel add rust` (inferred) |
| `samuel remove language rust` | `samuel rm rust` (inferred + alias) |
| `samuel remove framework react` | `samuel rm react` |

Both v2 (`add framework react`) and v3 (`add react framework`) argument orders work. Type aliases (`lang`/`l`, `fw`/`f`, `wf`/`w`) are still accepted.

### Autonomous loop (renamed `auto` → `run`, `auto` is permanent alias)

`samuel auto X` continues to work for every verb forever. It's not deprecated, just non-canonical.

| v2 | v3 |
|---|---|
| `samuel auto init` | `samuel run init` |
| `samuel auto convert <prd>` | `samuel run init <prd>` (positional) |
| `samuel auto start` | `samuel run start` |
| `samuel auto pilot` | `samuel run pilot` |
| `samuel auto status` | `samuel run status` |
| `samuel auto task list` | `samuel run tasks` |
| `samuel auto task complete 1.1` | `samuel run done 1.1` |
| `samuel auto task skip 1.1` | `samuel run skip 1.1` |
| `samuel auto task reset 1.1` | `samuel run reset 1.1` |
| `samuel auto task add 5.0 "Title"` | `samuel run task add 5.0 "Title"` (preserved for CI) |
| — (new) | `samuel run enqueue "Title"` (auto-id) |

Bare `samuel run` (or `samuel auto`) shows status when a loop is initialized. In an empty directory it prints actionable help and exits non-zero — it never silently starts a loop. v2 behavior was the same; v3 just makes the contract explicit.

### Power-user commands (`config`, `sync`, `diff` moved under `admin`)

`doctor` and `skill` stay top-level — they're trust-building and contributor commands, not admin.

| v2 | v3 |
|---|---|
| `samuel config list` | `samuel admin config list` |
| `samuel config get version` | `samuel admin config get version` |
| `samuel config set registry <url>` | `samuel admin config set registry <url>` |
| `samuel sync` | `samuel admin sync` |
| `samuel diff` (bare) | `samuel update --preview` |
| `samuel diff v1.6.0 v1.7.0` | `samuel admin diff v1.6.0 v1.7.0` |
| `samuel update --diff` | `samuel update --preview` |
| `samuel doctor` | `samuel doctor` (unchanged) |
| `samuel skill *` | `samuel skill *` (unchanged) |

## JSON output schema bump

The `--json` envelope now carries `"schemaVersion": 3` and the `command` field reflects the invoked path:

```json
{
  "schemaVersion": 3,
  "command": "ls",
  "success": true,
  "data": { ... }
}
```

Legacy command paths preserve their v2 `command` strings:

- `samuel list --json` → `"command": "list"`
- `samuel auto task complete X --json` → `"command": "auto task complete"`

Pin your consumers to `schemaVersion` rather than command-name strings if you want forward stability.

## CI script migration

If your CI invokes Samuel:

1. **Quick fix (no script changes)**: set `SAMUEL_NO_DEPRECATION=1` in your CI environment. Legacy commands keep working through v3.0.x with no log noise.
2. **Proper fix (script changes)**: update commands per the tables above. Test locally first; the new forms produce identical output.
3. **Long-term**: pin to `schemaVersion: 3` in JSON consumers if you parse the `command` field.

Plan to complete step 2 before v3.1.0 ships (~3 months after v3.0.0).

## What didn't change

- `samuel init` — same flags, same behavior
- `samuel update` — same flags except `--diff` is now Hidden (use `--preview`)
- `samuel doctor` — top-level, unchanged
- `samuel skill <verb>` — top-level, unchanged
- `samuel version` — unchanged
- AGENTS.md / CLAUDE.md template content — same surface area
- Registry of installable components — same languages, frameworks, workflows, skills

## Getting help

- `samuel <command> --help` shows the v3 form for every command
- `samuel --help` shows the new top-level surface (8 visible commands)
- `samuel --help-all` (Cobra built-in via aliases) shows hidden legacy commands too
- File issues at <https://github.com/ar4mirez/samuel/issues>

## Why v3 broke the surface

The maintainer personally used 3 of 14 commands in daily flow. New users hit a wall of verbs, several of which overlapped (`list`, `search`, `info` all answer "what's there?"). The `auto` command name told nothing about the autonomous loop being the differentiator.

v3.0.0 trades a one-time migration for a help screen that fits on one terminal page and a verb (`run`) that says what it does. Hidden+Deprecated aliases mean v2 scripts keep running while you adapt. The migration window is intentionally generous; the verb names are designed to last.
