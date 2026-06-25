# ASDF — Agentic Software Development Framework

> **Status: early implementation.** The Dolt infrastructure (salvaged from
> [beads](https://github.com/steveyegge/beads), MIT; see [NOTICE](NOTICE)), the schema, **`asdf
> init`**, the command contract, the core verbs (`domain`/`spec`/`req`/`edge`), and the
> **changeset (PR) flow** are built and verified against real Dolt. Generation, `check`/`impact`,
> remote sync, the MCP server, and import are next — see [ROADMAP.md](docs/ROADMAP.md). Some
> commands in the table below run today; others are still the *planned* surface.

ASDF is a command-line tool, backed by a [Dolt](https://www.dolthub.com/) database, that
serves as the **version-controlled system of record for everything a software project knows
about itself** — domains, specs, user stories, functional requirements, tests, milestones,
deliverables, and the links between them. It is built to be driven equally by **humans and
coding agents** (Claude, Codex, Cursor, opencode, …).

## The problem

Spec-driven and agentic development scatter a project's "knowledge" across tools that don't
compose: markdown specs, an FR-traceability registry, Notion planning boards, Qase test
management, Jira/beads issues. They drift out of sync, none of it is queryable as a single
graph, it isn't versioned together, and an agent can't reliably read or write across all of
it.

## The idea

Put it all in one place, as one graph, in a database that branches and merges like code.

- **One source of truth.** The Dolt database is canonical — always.
- **Generated, never edited.** Human- and agent-friendly **Markdown and HTML are
  auto-generated** from the DB. They are **git-ignored build artifacts** — never
  hand-edited, never agent-edited. Change content through the CLI/MCP, then regenerate.
- **Branch / merge / diff.** Dolt gives the project's knowledge the same version-control
  model as its code: agents work on branches, you review and merge.
- **Two interfaces, one store.** Humans and agents **read, write, and validate through the
  CLI or an MCP server**. The generated Markdown is an optional *fast read path* for
  agents; it is never a write path.
- **Traceability & impact.** Because everything is one linked graph, ASDF can **validate**
  (e.g. every requirement covered by a test, every deliverable linked) and answer
  **impact** questions ("what breaks if this requirement changes?").
- **Generic core.** ASDF is domain-agnostic. Its data model was pressure-tested against a
  real corpus, but the core carries no project- or tenant-specific assumptions.

## How it flows

```
             write / validate (CLI · MCP)
  human  ──────────────────────────────▶  ┌──────────────┐  generate   ┌─────────────────────────┐
  agent  ◀──────────────────────────────  │   Dolt  DB   │ ──────────▶ │  Markdown + HTML        │
             query / read (CLI · MCP)      │ (canonical)  │             │  (git-ignored, read-only)│
                                           └──────────────┘             └─────────────────────────┘
                                                                          ▲ agents may read these
                                                                            for fast consumption
```

## Data model

The full entity-relationship model lives in **[docs/entities/](docs/entities/index.md)**. In brief, by layer:

- **Structure** — `Domain`, `Spec` (the document tree; directories derive from a spec's `path`)
- **Requirements** — `UserStory`, `AcceptanceScenario`, `Requirement` (FR), `Milestone`, `Edge` (the cross-reference graph)
- **Testing** (Qase-style) — `TestSuite`, `TestCase`, `TestStep`, `TestRun`, `TestResult`, `Configuration`
- **Planning** — `Capability`, `Deliverable`, `View`
- **Authorization & entities** — `Entity`, `EntityAttribute`, `EntityRelationship`, `Privilege`, `AccessRule` (authored *business-domain* documents, **not** a DB-schema mirror)
- **Interop** — `ExternalRef` (a node's id in an outside tracker: Jira, Rally, beads, …)

Identifiers use ULID surrogate keys plus human-readable unique business keys (e.g.
`ATT-FR-012`). See [Identifiers & keys](docs/entities/identifiers.md).

## Planned commands (illustrative)

| Command | Purpose |
|---|---|
| `asdf init` | Create / connect the Dolt database |
| `asdf spec` · `asdf req` · `asdf test` … | Create, link, and query nodes |
| `asdf query <…>` | Ad-hoc queries over the graph |
| `asdf generate` | Regenerate Markdown + HTML from the DB |
| `asdf check` | Validate traceability / consistency |
| `asdf impact <id>` | Show what a change affects |
| `asdf import <source>` | Generic migration import |
| `asdf serve --mcp` | Run the MCP server |

## Install

> Releases are cut by [GoReleaser](https://goreleaser.com) on every `v*` tag and published to
> [GitHub Releases](https://github.com/endermalkoc/asdf/releases) as static, single-file
> binaries for Linux/macOS/Windows (amd64 + arm64). The CLI surface is still early — `asdf
> version` works today; data commands need a running Dolt server (see [build/run](CLAUDE.md#build--run)).

**Install script** (Linux/macOS — downloads the right binary and verifies its checksum):

```sh
curl -fsSL https://raw.githubusercontent.com/endermalkoc/asdf/main/install.sh | sh
```

Pin a version or change the location with `ASDF_VERSION=v0.1.0` / `ASDF_INSTALL_DIR=~/.local/bin`.

**With Go** (any platform):

```sh
go install github.com/endermalkoc/asdf/cmd/asdf@latest
```

**From source:**

```sh
git clone https://github.com/endermalkoc/asdf && cd asdf
make build   # → ./asdf   (make install puts it on your PATH)
```

> The binary is named `asdf`, which collides with the [asdf version manager](https://asdf-vm.com)
> (see the name note below); use `ASDF_INSTALL_DIR` to control PATH precedence.

## Tech stack

**Go** (locked) — single static binary across Windows/Mac/Linux, Dolt is itself Go, exposing
**both a CLI and an MCP server**. Its Dolt infrastructure was salvaged directly from
[beads](https://github.com/steveyegge/beads) (MIT), which made the precedent the foundation.
See [ARCHITECTURE.md](docs/ARCHITECTURE.md#tech-stack).

## Roadmap (high level)

1. Generate the Dolt DDL from the [data model](docs/entities/index.md).
2. Core CLI: create / link / query + the generation pipeline.
3. Validation (`check`) and impact analysis.
4. MCP server.
5. **Generic import tool**; first real migration: the `tutor/docs` corpus.

Inspired by — and building on the Dolt infrastructure of — [beads](https://github.com/steveyegge/beads).

## Docs

- [docs/entities/](docs/entities/index.md) — the data model
- [ARCHITECTURE.md](docs/ARCHITECTURE.md) — architecture & principles
- [ROADMAP.md](docs/ROADMAP.md) — what's built, what's next
- [docs/codebase-map.md](docs/codebase-map.md) — folder map + fresh-session orientation
- [CLAUDE.md](CLAUDE.md) — guidance for agents and contributors

> **Name note:** "asdf" collides with the popular [asdf version manager](https://asdf-vm.com/);
> the published binary / command name may change.

## License

MIT — see [LICENSE](LICENSE).
