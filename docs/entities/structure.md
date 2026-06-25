# Structure layer

[← index](index.md) · see the [master diagram](index.md#master-diagram) and
[Identifiers & keys](identifiers.md).

`Domain` and `Spec` form the document tree. There is **no folder table** — the directory
structure is derived from `Spec.path`.

## Domain
A top-level area, and the **shared classification dimension** that ties the spec corpus to
the planning layer. It maps to a root directory under `docs/specs/`; **every domain has
specs**. Service boundaries (`enrollment`, `scheduling`, `finance`, `identity`, `platform`,
`staffing`, `communication`, …) plus the special `entities` / `shared` / `infrastructure`
trees.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `abbreviation` | varchar | **UK** | |
| `name` | varchar | | Canonical name |
| `kind` | enum | | `service`, `shared`, `infrastructure`, `entities`, `analysis` |
| `status` | enum | | `draft`, `active`, `deprecated` |

## Spec
A document — one `.md` file — and the unit that owns FR numbering. The directory tree is
**derived from `path`** (no separate folder table); `path` is the full docs-relative
location. FR-bearing specs have a unique `prefix`; FR-exempt docs (entity glossary,
journeys, analysis, index/meta) have `prefix = NULL` and a `kind` that classifies them.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `domain_id` | FK → Domain | | From frontmatter `domain` (also the first path segment) |
| `prefix` | varchar | **UK** | 2–6 upper; **nullable** for FR-exempt docs |
| `slug` | varchar | | Filename without extension |
| `path` | varchar | **UK** | Full docs-relative path; **source of the directory tree** |
| `title` | varchar | | |
| `kind` | enum | | `feature`, `entity`, `journey`, `analysis`, `index`, `meta`, `reference` |
| `status` | enum | | `draft`, `active`, `obsolete` |
| `created_at` / `updated_at` | date | | From spec header |
