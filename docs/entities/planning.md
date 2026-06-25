# Planning layer (Capabilities / Deliverables / Views)

[← index](index.md) · see the [master diagram](index.md#master-diagram).

The planning layer tracks *what to build*. It sits **above** the spec corpus and connects
to it through shared [`Domain`](structure.md#domain) + [`Milestone`](requirements.md#milestone)
and the `View → Spec` link. The chain is
**Capability → Deliverable → View → Spec → (UserStory, Requirement)**.

## Capability
A product capability in a 3-tier hierarchy (`level` = `domain` › `epic` › `capability`),
self-nesting via `parent_id`.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `title` | varchar | | Capability name |
| `level` | enum | | `domain`, `epic`, `capability` |
| `domain_id` | FK → Domain | | |
| `parent_id` | FK → Capability | | Self-ref (parent / sub) |

- **Milestones**: many-to-many → Milestone (a capability can span milestones). Junction `capability_milestone`.
- **Deliverables**: many-to-many → Deliverable. Junction `capability_deliverable`.

## Deliverable
A unit of work — sized, status-tracked, milestone-scoped. The primary subject for external
task-system references (see [ExternalRef](interop.md#externalref)).

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `title` | varchar | | |
| `size` | enum | | `S`, `M`, `L`, `XL`; nullable |
| `status` | enum | | `proposed`, `specced`, `wired`, `built`, `ship` |
| `ai_ready` | enum | | `yes`, `no`, `na` |
| `milestone_id` | FK → Milestone | | One milestone; nullable |

- **Capabilities**: many-to-many → Capability. Junction `capability_deliverable`.
- **Views**: many-to-many → View. Junction `deliverable_view`.
- **Blocked by**: self many-to-many dependency. Junction `deliverable_dependency(deliverable_id, blocked_by_id)`.

## View
A UI surface (an app route), optionally backed by a spec. The **bridge** from the planning
layer into the spec corpus.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `title` | varchar | | |
| `route` | varchar | | App route, e.g. `/students/[studentId]?tab=messages` |
| `spec_id` | FK → Spec | | **Nullable** — set once a spec backs the view |
| `domain_id` | FK → Domain | | |

- **Deliverables**: many-to-many → Deliverable. Junction `deliverable_view`.

## Junction tables (planning many-to-many)

The **column pair is the PK** (no surrogate `id`), so the same link created on two branches
is the same row and merges cleanly — see the [deterministic-PK rule](identifiers.md).

| Table | Columns (composite PK) |
|---|---|
| `capability_milestone` | `capability_id`, `milestone_id` |
| `capability_deliverable` | `capability_id`, `deliverable_id` |
| `deliverable_view` | `deliverable_id`, `view_id` |
| `deliverable_dependency` | `deliverable_id`, `blocked_by_id` (both → Deliverable) |
