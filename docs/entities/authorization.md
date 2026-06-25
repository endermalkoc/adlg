# Authorization & entity layer

[← index](index.md) · see the [master diagram](index.md#master-diagram).

> These are **authored business-domain documents**, not a projection of the technical
> schema. They describe what an entity *means* — purpose, domain properties, relationships,
> business rules, validations, access — in domain language. ASDF owns them as the canonical
> domain glossary; they do **not** mirror or sync from a database schema, and a property
> here need not correspond one-to-one with a stored column. The full narrative lives in the
> linked entity doc (a [`Spec`](structure.md#spec) with `kind = entity`); these tables are
> its structured head.

## Entity
A domain entity from `entities/**` (Student, Family, Event, …) — a business concept.
Usually has a documenting spec (`kind = entity`) that carries the prose.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `domain_id` | FK → Domain | | |
| `spec_id` | FK → Spec | | The entity doc (full narrative); nullable |
| `name` | varchar | **UK** | |
| `description` | text | | Domain purpose / definition |
| `status` | enum | | `draft`, `active`, `deprecated` |

## EntityAttribute
A documented **domain property** of an entity — its business meaning, not a typed DB
column. Grouped the way entity docs group them (e.g. "Academic", "Lesson Settings",
"Calculated / Derived Fields").

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `entity_id` | FK → Entity | | |
| `name` | varchar | | Domain property name (e.g. `skill level`, `birthday`) |
| `description` | text | | What it means / business rules for it |
| `category` | varchar | | Doc grouping (e.g. `Academic`, `Lesson Settings`); nullable |
| `is_derived` | bool | | Calculated / derived rather than directly entered |
| `position` | int | | Ordering within the entity; nullable |

## EntityRelationship
A relationship between two entities (the "Relationships" section of an entity doc).

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `from_entity_id` | FK → Entity | | |
| `to_entity_id` | FK → Entity | | |
| `cardinality` | enum | | `one_to_one`, `one_to_many`, `many_to_many` |
| `junction_table` | varchar | | Nullable (for m2m) |
| `notes` | text | | |

## Privilege
A `(resource, scope, action)` triple — the authorization vocabulary used by entity
Row-Level Access Rules. Authorization is always expressed as triples, never role names.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `resource` | varchar | | e.g. `students`, `tutor_compensation` |
| `scope` | enum | | `owned`, `studio` |
| `action` | enum | | `view`, `manage` |

> `UNIQUE(resource, scope, action)`.

## AccessRule
A row-level access rule on an entity: "IF I have `(resource, scope, action)` [AND
condition] THEN \<description\>".

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `entity_id` | FK → Entity | | |
| `privilege_id` | FK → Privilege | | The required triple |
| `condition` | text | | e.g. "created by me OR assigned to me"; nullable |
| `description` | text | | What the rule grants |
