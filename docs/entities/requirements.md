# Requirements layer

[← index](index.md) · see the [master diagram](index.md#master-diagram).

`UserStory`, `AcceptanceScenario`, `Requirement` (the functional requirement), `Milestone`,
and `Edge` (the cross-reference graph).

## UserStory
BDD scaffolding inside a spec. **No global ID** — referenced by its per-spec `ordinal`
("User Story 3"). Carries a priority and the As-a / I-want / So-that persona.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | Surrogate only; not a citable ID |
| `spec_id` | FK → Spec | | |
| `ordinal` | int | | Unique **within spec** (the heading number) |
| `title` | varchar | | |
| `priority` | enum | | `P1`, `P2`, `P3` |
| `as_a` / `i_want` / `so_that` | text | | Persona narrative |

## AcceptanceScenario
A Given/When/Then scenario under a user story.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `user_story_id` | FK → UserStory | | |
| `ordinal` | int | | Order within story |
| `given` / `when` / `then` | text | | |

## Requirement (Functional Requirement)
A `{PREFIX}-FR-{NNN}{x}` requirement. The human-readable ID is **composed**:
`spec.prefix + "-FR-" + zero-pad(number) + suffix`. Numbering is sequential within the
spec; gaps are allowed (deletions/reservations). Sub-requirements use `suffix` + a
self-reference to the base FR. Delivery metadata is inline. See
[Identifiers & keys](identifiers.md) for the `fr_key` derived column and the merge caveat.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `spec_id` | FK → Spec | | Owns the numbering namespace |
| `number` | int | | Sequential within spec (unique per `spec_id`) |
| `suffix` | char(1) | | Optional single sub-letter (`a`,`b`,…); nullable |
| `parent_id` | FK → Requirement | | Set on sub-requirements; null on base FRs |
| `statement` | text | | The MUST statement |
| `content_status` | enum | | `draft`, `active`, `obsolete` (the requirement's own lifecycle) |
| `delivery_status` | enum | | `covered`, `test-pending`, `not-implemented`, `e2e-sufficient`, `shared`, `schema-only`, `deferred` |
| `milestone_id` | FK → Milestone | | Nullable |
| `owner` | varchar | | e.g. `backend` |
| `notes` | text | | |
| `optout_marker` | enum | | `none`, `visual`, `ops`, `untestable` |
| `optout_reason` | varchar | | Required when `optout_marker != none` |
| `tombstoned_at` | date | | Set when the FR is deleted (tombstone) |
| `created_at` / `updated_at` | datetime | | |

> Constraints worth enforcing: `UNIQUE(spec_id, number, suffix)`; ≥1 linked
> [`TestCase`](testing.md#testcase) of `layer = e2e` when `delivery_status = e2e-sufficient`
> and of `layer = shared` when `= shared`; `milestone_id` required when
> `delivery_status = not-implemented`.

## Milestone
An ordered delivery target, and the **second cross-cutting join hub** (with `Domain`)
between the spec corpus and the [planning layer](planning.md). Referenced from three places:
`Requirement.milestone_id` (spec side), `Deliverable.milestone_id` (one per deliverable),
and `Capability` (many-to-many — a capability can span milestones). Example value set:
`M0`–`M7`, `Future`.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `abbreviation` | varchar | **UK** | `M0`–`M7`, `Future` |
| `name` | varchar | | |
| `description` | text | | |
| `sequence` | int | | `Future` sorts last |
| `status` | enum | | `complete`, `in_progress`, `pending` |
| `created_at` / `updated_at` | datetime | | |

## Edge
A typed, directed, **polymorphic** link between two nodes — the cross-reference / graph
layer. `references` (the pervasive "see X" / "per X" citation) is the common case.
(FR↔test coverage is its own `requirement_test_case` junction — see
[testing.md](testing.md#testcase) — not an edge.)

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | **Deterministic** — `uuidv5` over the `UNIQUE` identity, not a random ULID (see [Identifiers](identifiers.md)) |
| `from_type` | enum | | `requirement`, `spec`, `user_story`, `entity`, `milestone` |
| `from_id` | bigint / uuid | | Polymorphic FK (type + id) |
| `to_type` | enum | | same domain as `from_type` |
| `to_id` | bigint / uuid | | Polymorphic FK |
| `kind` | enum | | `references`, `refines`, `depends_on`, `supersedes`, `relates`, `defers_to` |

> `UNIQUE(from_type, from_id, to_type, to_id, kind)` is the edge's identity, and the PK is
> derived from it — so the same edge added on two branches converges on merge instead of
> tripping the unique key. See the [deterministic-PK rule](identifiers.md).
