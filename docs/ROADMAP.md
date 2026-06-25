# ASDF Roadmap

What's built and what's next. A living document — pairs with [ARCHITECTURE.md](ARCHITECTURE.md)
(how it's put together), [docs/entities/](entities/index.md) (the data model), and
[docs/command-contract.md](command-contract.md) (the workflow every command follows).

## Done

- **Dolt infrastructure** (salvaged from [beads](https://github.com/steveyegge/beads), MIT — see
  [NOTICE](../NOTICE)): owned/external/embedded server management (`internal/doltserver`), the warm
  proxy daemon (`dbproxy`), version-control ops over a `DBConn` (`versioncontrolops`: branch/
  commit/merge/clone/gc/flatten/backup), remotes (`remotecache`, `doltremote`), config/git/error
  helpers. Issue-domain dependency severed to a minimal shim; all pure-Go, building clean.
- **Data model** ([docs/entities/](entities/index.md)) — authoritative, consistent, with the
  deterministic relationship-PK rule.
- **Schema** — `0001_init` (26 entities + 6 junctions) + a migration runner
  (`internal/storage/schema`); validated against real Dolt (FK/UNIQUE/deterministic-PK enforced).
- **ID minting** (`internal/ids`) — ULID (authored rows) + deterministic `uuidv5` (relationships).
- **`asdf init`** — creates `.asdf/`, starts a managed (owned) `dolt sql-server`, applies the
  schema, seeds the actor, records the initial Dolt commit.
- **Command contract** (`internal/app.Mutate`) — every mutating command: managed connect →
  resolve changeset/`main` target → validate → transaction → mint → attribute/timestamp → real
  Dolt commit with actor+message (clean working set). Bad input fails before any write.
- **Entity CRUD (slice)** — `domain`/`spec`/`req`/`edge` `add` + `ls`.
- **Changesets (the PR model)** — `asdf changeset start/diff/submit/merge/abandon/ls`: a changeset
  is a Dolt branch; edits route to the active changeset; `diff` is the combined PR view; edits stay
  isolated from `main` until merge; `Changeset`/`Review` rows live on `main`.

## Next — finish the command contract

- **Graph integrity** — `edge add` needs cycle detection, polymorphic-endpoint existence/type
  validation, and generalization beyond `requirement→requirement`.
- **`--dry-run` flag** — `Mutate` already supports it; expose it on the CLI.
- **Structured errors → exit codes** + a `--json` error envelope.
- **Broaden CRUD** — `edit`/`delete`/`show` for existing entities, then the remaining entities
  (Milestone, Test*, Capability, Deliverable, View, Entity*, Privilege, AccessRule, ExternalRef).
- **`asdf config get/set`** — `internal/config`/`configfile` are lifted but have no CLI yet.

## Core features

| Feature | What | Status / notes |
|---|---|---|
| **Remote sync** | `asdf dolt push/pull/remote/clone`, `asdf sync`, federation (peers) | **requested.** Infra lifted (`remotecache`, `doltutil` remotes, `versioncontrolops` remotes/`Push`/`Fetch`); wire the CLI. Sync of a versioned knowledge graph = `dolt push/pull`. |
| **Generate** | `asdf generate`: DB → git-ignored **Markdown + HTML** (the canonical-derived read artifacts) | **requested. ASDF-original** — beads has no generate; it exports JSONL. This is core to ASDF's "generated, never edited" principle. |
| **Batch add** | `asdf <entity> add --file <f>` and/or `asdf batch <f>` — bulk-create entities from a **JSON/CSV** file in ASDF's own shape, in **one changeset/commit** | adapt beads' `bd create --file`/`--graph`; rides the `Mutate` wrapper so the whole batch is one transaction + one Dolt commit. |
| **Generic import** | `asdf import --format json\|csv <f>` — ingest **arbitrary external** JSON/CSV and map columns/fields into the schema via a mapping spec | the source-agnostic core (distinct from batch add: external shape → field mapping → entities). Validates + routes through the contract. |
| **Source adapters** | `asdf import <source>` — pluggable per-source adapters on top of the generic core; **first: `tutor`** | the `tutor` corpus: markdown specs + the FR-traceability registry + the former planning DBs + Qase test management → `Domain`/`Spec`/`Requirement`/`Test*`/… Adapters hold source quirks, keeping the core clean (ARCHITECTURE §Import). |
| **Export** | `asdf export` — JSONL snapshot (git-friendly, diffable) | beads' model; useful for backups/interop alongside Dolt history. |
| **Validation & analysis** | `asdf check` (traceability), `asdf impact <id>` (graph traversal), `asdf doctor` (health + auto-fix), drift detection | `check`/`impact` are core ASDF (README roadmap). `doctor`/`drift` adapt beads patterns (we have `schema.CheckForwardDrift`-style hooks). |
| **Query / inspect** | `asdf sql` (raw passthrough), `query`, `search`, `stats`, `history`/`diff` (Dolt-native), `show` | `sql` is a cheap, high-value passthrough. History/diff/blame come free from `dolt_*` system tables. |
| **Agent integration** | `asdf setup` → install into **Claude Code**, Codex, Cursor, Gemini, Aider, opencode; the **MCP server** (`asdf serve --mcp`) | **requested ("initialize into Claude Code").** Mirror beads' `cmd/bd/setup` (same agent targets). MCP is in the README roadmap. |
| **DB maintenance** | `asdf backup`/`restore`, `gc` (Dolt GC), `compact`/`flatten` (history compaction) | Infra lifted in `versioncontrolops` (gc/compact/flatten); wire the CLI. |
| **Self-update** | **`asdf upgrade`** — download the latest release binary, verify its checksum, and replace in place; **`asdf version`** reports the build | **requested.** Mirror beads' `cmd/bd/upgrade.go`; pairs with the GoReleaser/GitHub-Releases distribution in the README. |
| **CLI polish** | **help system** (rich help + examples, `help-all` overview), shell completion | **requested (help).** Cobra gives base help/completion; add per-command examples and a top-level overview. |

## "What am I missing vs beads?" — feature survey

Cross-cutting beads features (not issue-domain), and ASDF's status:

| beads | ASDF status |
|---|---|
| `dolt push/pull/remote`, `sync`, `federation` | infra lifted → **roadmap (Remote sync)** |
| `export` (JSONL) | **roadmap (Export)** |
| `import` | **roadmap (Generic import + Source adapters: tutor)** |
| `batch` (bulk create) | **roadmap (Batch add — JSON/CSV)** |
| `sql` (raw passthrough) | **roadmap (Query)** |
| `search` | **roadmap (Query)** |
| `backup`/`restore` | **roadmap (DB maintenance)** |
| `doctor`, `drift`, `preflight` | **roadmap (Validation & analysis)** |
| `gc`, `compact`, `flatten` | infra lifted → **roadmap (DB maintenance)** |
| `setup` (agent install) | **roadmap (Agent integration)** |
| MCP server | **roadmap (Agent integration)** |
| `config` (get/set) | **Next** |
| `version`/`upgrade` (self-update) | **roadmap (Self-update)** |
| shell completion | **roadmap (CLI polish)** — cobra-provided |
| `hooks` (`on_create`, …) | **deferred** — `internal/hooks` not lifted (needs a node concept) |
| `worktree` commands | partial — `git.GetMainRepoRoot` is worktree-aware; explicit cmds maybe |
| `metrics`/telemetry | likely **skip / opt-in** |
| `stats`, `count`, `info`, `history`, `where` | **roadmap (Query/inspect)** |

**Generate (Markdown/HTML) is the one major feature ASDF has that beads does not** — beads is an
issue tracker that snapshots to JSONL; ASDF is a spec/knowledge store whose human/agent views are
generated from the canonical DB.

## Testing & CI

Today only `internal/ids` is unit-tested; everything else was verified **manually** against real
Dolt (`init` → `add` → commit → changeset round-trip). Codify that:

- **Unit tests** (fast, no DB) — pure logic: `ids` (✅), `enums` + `app` validation, `store` SQL +
  field mapping (`fr_key` derivation, `nullIfEmpty`), and `workspace` helpers (actor resolution,
  changeset slug, retry/serialization classification, `dolt_diff_stat` parsing).
- **Integration tests** (real Dolt, **server mode — no cgo**) — a harness that starts a managed
  `dolt sql-server` (or `testcontainers-go/modules/dolt`), applies the schema, and exercises the
  command contract end-to-end: `asdf init`; each `add` produces one Dolt commit with a **clean
  working set**; validation rejects bad input; and the full **changeset round-trip** (`start → add
  on branch → diff → submit → merge`, with `Changeset`/`Review` rows staying on `main`). Reference:
  beads' `internal/testutil/integration`.
- **Embedded-driver e2e** — the in-process `dolthub/driver/v2` test was reverted (needs cgo +
  `libicu-dev`); reintroduce behind a build tag (e.g. `-tags dolt_e2e`) once CI guarantees ICU.
- **CI** — run `go build ./...` · `go vet ./...` · `go test ./...` on every push; gate the
  integration suite on `dolt` availability (PATH or testcontainers).

## Deliberately NOT carried from beads

- **Issue-tracker verbs** — `create/close/reopen/ready/blocked/dep/assign/priority/label/epic/todo/
  defer/ack/acquire/release`. ASDF has its own entity verbs (`domain/spec/req/edge`, + planned
  Test/Milestone/Capability/…).
- **Agent-orchestration machinery** — `swarm/wisp/mol/gate/bond/pour/cook/prime/kv-memories`. Beads-specific.
- **Divergences (by design):** ULID + deterministic ids (not content-hash); **Dolt-native history +
  the Changeset/Review layer** (not an `events` audit table); the **changeset PR model** (not
  auto-commit-per-op); no `is_blocked` denormalization (compute via `impact`).

## Deferred work & open decisions

- **Merge coordination** (was beads `merge_slot.go`) — not lifted (a lock implemented *as* an
  issue). When needed, build a generic Dolt-native lock (dedicated table or reserved branch), not a
  lift.
- **Embedded mode needs cgo + ICU** (`libicu-dev`) to build. Owned/external (pure-Go) is the
  default; embedded is recognized but deferred. Revisit shipping embedded-by-default vs.
  requiring the `dolt` binary.
- **Lifted utilities now consumed:** `internal/git` (workspace), `internal/storage/dberrors`
  (schema runner). Still orphaned: `internal/timeparsing` (pull in when a command takes dates).
- **`fr_key`** is an app-maintained column, not a SQL generated column (cross-table generation
  isn't possible in Dolt) — keep the store deriving it on write.
- **Rebrand** — `.asdf/` dir + `ASDF_*` env collide with the asdf version manager; pick the
  published binary name + config dir before release (branding flows from a few constants).
- **Concurrency:** same-branch number allocation is safe (`FOR UPDATE` + retry); cross-branch
  FR-number convergence is the documented merge-renumber policy (identifiers.md).
- **`go mod tidy`** upkeep as imports change (currently 11 direct deps).
