# Testing layer (Qase-style)

[← index](index.md) · see the [master diagram](index.md#master-diagram).

Test management modeled on Qase: a hierarchy of **suites** holding **cases**, which are
exercised by **runs** that record one **result** per case. The `frTest` corpus convention
maps onto this as automated `TestCase`s (with a `path`) that **cover**
[requirements](requirements.md#requirement-functional-requirement) many-to-many. Run
pass/fail status lives on `TestResult`, not on the case — a case has a *lifecycle* status
(`draft`/`active`/`deprecated`); each execution has an *outcome*.

Runs and results are **authored or imported via the CLI** (e.g. a JUnit/Qase-report
importer feeding `TestResult` rows). `Configuration` parameterizes runs; Qase's `Plan` and
`SharedStep` concepts are intentionally **omitted**.

## TestSuite
A folder grouping for test cases; self-nesting (a suite tree, like Qase suites).

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `parent_id` | FK → TestSuite | | Self-ref; null at root |
| `name` | varchar | | |
| `description` | text | | |
| `position` | int | | Ordering among siblings; nullable |

## TestCase
A defined test. Automated cases carry a `path` and cite FRs (the `frTest` link).

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `suite_id` | FK → TestSuite | | |
| `title` | varchar | | |
| `description` | text | | |
| `preconditions` | text | | |
| `layer` | enum | | `unit`, `integration`, `e2e`, `component`, `shared` (the corpus's old `kind`) |
| `type` | enum | | `functional`, `smoke`, `regression`, `acceptance`, `other` |
| `priority` | enum | | `low`, `medium`, `high` |
| `severity` | enum | | `trivial`, `minor`, `normal`, `major`, `critical`, `blocker` |
| `automation` | enum | | `manual`, `automated`, `to_be_automated` |
| `status` | enum | | `draft`, `active`, `deprecated` (lifecycle, **not** run outcome) |
| `path` | varchar | | Automated test file, e.g. `apps/web/e2e/add-student.spec.ts`; nullable |
| `is_flaky` | bool | | |
| `created_at` / `updated_at` | datetime | | |

- **Coverage**: many-to-many → Requirement via `requirement_test_case(requirement_id, test_case_id)` — a case may cite several FRs; an FR may be covered by several cases.

## TestStep
An ordered action / expected-result step within a case (optional — manual cases benefit
most; automated cases may leave steps empty).

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `test_case_id` | FK → TestCase | | |
| `ordinal` | int | | Order within case |
| `action` | text | | |
| `expected_result` | text | | |

## TestRun
An execution cycle over a selection of cases; optionally scoped to a milestone and to one
or more configurations.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `title` | varchar | | |
| `description` | text | | |
| `status` | enum | | `active`, `complete`, `aborted` |
| `milestone_id` | FK → Milestone | | Nullable |
| `started_at` / `ended_at` | datetime | | |

- **Configurations**: many-to-many → Configuration via `test_run_configuration(run_id, configuration_id)` — the axes this run exercises (e.g. Chrome + Windows).

## TestResult
One case's outcome within a run (the associative entity between `TestRun` and `TestCase`),
optionally pinned to the `Configuration` it ran under.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | **Deterministic** — `uuidv5` over `UNIQUE(run_id, test_case_id, configuration_id)`, not a random ULID; payload (`status`, …) is excluded so edits don't re-key. See [Identifiers](identifiers.md) |
| `run_id` | FK → TestRun | | |
| `test_case_id` | FK → TestCase | | |
| `configuration_id` | FK → Configuration | | Nullable (the config this result is for) |
| `status` | enum | | `passed`, `failed`, `blocked`, `skipped`, `invalid`, `in_progress` |
| `comment` | text | | |
| `duration_ms` | int | | Nullable |
| `stacktrace` | text | | Nullable |
| `executed_by` | varchar | | Nullable |
| `executed_at` | datetime | | Nullable |

## Configuration
A Qase configuration: a value within a named group, used to parameterize runs. Modeled as
one table (group label + value) rather than separate group/value tables.

| Attribute | Type | Key | Notes |
|---|---|---|---|
| `id` | bigint / uuid | **PK** | |
| `group` | varchar | | e.g. `Browser`, `OS`, `Environment` |
| `name` | varchar | | The value, e.g. `Chrome`, `Windows`, `staging` |
| `description` | text | | Nullable |

- `UNIQUE(group, name)`. Linked to runs via `test_run_configuration`; a `TestResult` may pin the specific configuration it ran under.
