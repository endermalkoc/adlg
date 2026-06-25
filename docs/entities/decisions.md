# Decisions

[← index](index.md)

## Open questions

_None — all resolved (see below)._

## Resolved decisions

- Directory tree derives from `Spec.path` (no `Folder` table).
- `Test` is first-class, expanded into a Qase-style layer.
- Acceptance scenarios stay as structured Given/When/Then.
- **IDs**: ULID surrogate PKs + `UNIQUE` business keys + tiered display IDs — see
  [Identifiers & keys](identifiers.md).
- **Pure-relationship PKs are deterministic, not ULID**: junctions use a composite PK, and
  `Edge` / `TestResult` derive their surrogate `id` (`uuidv5`) from their `UNIQUE` identity —
  so a relationship two branches create independently converges on merge instead of tripping
  a unique-key violation. (Adopts the beads dependency-table technique; see
  [Identifiers & keys](identifiers.md).)
- **Testing**: runs/results are CLI-authored or CLI-imported; `Configuration` is included;
  Qase `Plan` / `SharedStep` are omitted.
- **Entities** are authored business-domain documents — `EntityAttribute` is a domain
  property (meaning), not a schema-column mirror; no sync from a DB schema.
- **`ExternalRef` subjects**: `Deliverable`, `Requirement`, `TestResult`. `system` stays a
  free string with a documented set (`jira`/`rally`/`beads`/`linear`/`github`/…).
- **Review & collaboration layer added** — `Changeset` + `Review` + `Comment` +
  `Actor`; the changeset carries Dolt branch/commit coordinates as the bridge. See
  [review.md](review.md).
- **History & diff stay Dolt-native** — no revision/audit tables. Spec/requirement history
  and agent-change diffs come from `dolt_history_*` / `dolt_diff_*` / `dolt_log` /
  `dolt_blame_*`.
- **Changeset is the unit of batched, reviewable change** (the `Changeset` entity, formerly
  `ChangeProposal`). A changeset is a **Dolt branch** that bundles edits across many entities
  so they are diffed and reviewed together (a PR over the knowledge graph), not committed
  per-edit. CLI: `asdf changeset start|diff|submit|merge|abandon`; mutating commands take an
  optional `--changeset <name>` (and honor an ambient active changeset set by `start`). With
  **no changeset, a write commits straight to `main`** (auto-commit default). Lifecycle via
  `status`: `draft` (building) → `open` (submitted for review) → `approved`/`merged`. The diff
  is computed from Dolt (`base_commit`→`head_commit`), never duplicated into tables; the
  `Changeset`/`Review`/`Comment` rows live on `main`, not inside the changeset branch. This
  deliberately differs from beads (which auto-commits each operation).
