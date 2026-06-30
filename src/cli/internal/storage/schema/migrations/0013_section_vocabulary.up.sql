-- 0013_section_vocabulary.up.sql — replace the polymorphic req_doc_section with a
-- CURATED, TYPED section model (four tables). The 0003→0010 arc let a document carry
-- unbounded one-off ("bespoke") sections and baked the corpus's section template into
-- the generator. This makes the section vocabulary data agents SELECT FROM:
--
--   *_section_type : the curated lookup (title, level, canonical render position).
--                    Keyed by `slug` (PK — the reference-table exception to ULID PKs;
--                    `slug` not `key`, which is a SQL reserved word that breaks tools
--                    that don't backtick it). A built-in seed ships here; new types are
--                    added only via a deliberate `section-type add` CLI call (origin
--                    'authored'), never an inline flag — friction is the design.
--   *_section      : one prose section, which MUST reference a type (NOT NULL). No
--                    bespoke. Title/level/order come from the type; the row is body.
--
-- Render order is canonical (type.position), uniform across docs. The generator's two
-- structural blocks (User Scenarios & Testing, Requirements) own the `edge_cases` /
-- `more_info` types — the only schema↔renderer coupling. DDL + seed; section content is
-- import-only, so a re-import repopulates the *_section rows (like 0004/0010).
-- See docs/entities/{structure,authorization,decisions}.md.

-- ---- type lookups (create before the instance tables they are FK targets of) -------

CREATE TABLE IF NOT EXISTS `req_spec_section_type` (
    `slug`        VARCHAR(64)  NOT NULL,                 -- selected by CLI/importer (overview, …)
    `title`       TEXT,                                  -- the section's title, rendered "## {title}"; NULL = no heading line (preamble)
    `level`       INT          NOT NULL DEFAULT 2,       -- 2=##, 3=###, 0=headingless
    `position`    INT          NOT NULL DEFAULT 0,       -- canonical render order
    `description` TEXT,                                  -- guidance shown when picking a type
    `origin`      VARCHAR(16)  NOT NULL DEFAULT 'builtin',-- builtin (seed) | authored (added later)
    `created_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`slug`)
);

CREATE TABLE IF NOT EXISTS `ent_entity_section_type` (
    `slug`        VARCHAR(64)  NOT NULL,
    `title`       TEXT,
    `level`       INT          NOT NULL DEFAULT 2,
    `position`    INT          NOT NULL DEFAULT 0,
    `description` TEXT,
    `origin`      VARCHAR(16)  NOT NULL DEFAULT 'builtin',
    `created_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`slug`)
);

-- ---- instance tables (typed FK to the lookup; owner FK with ON DELETE CASCADE) ------

CREATE TABLE IF NOT EXISTS `req_spec_section` (
    `id`                VARCHAR(36) NOT NULL,
    `spec_id`           VARCHAR(36) NOT NULL,
    `section_type_slug` VARCHAR(64)  NOT NULL,           -- curated type; never NULL
    `body`              TEXT,
    `created_at`        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_spec_section_owner_type` (`spec_id`, `section_type_slug`),
    INDEX `idx_spec_section_spec` (`spec_id`),
    INDEX `idx_spec_section_type` (`section_type_slug`),
    CONSTRAINT `fk_spec_section_spec` FOREIGN KEY (`spec_id`) REFERENCES `req_spec` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_spec_section_type` FOREIGN KEY (`section_type_slug`) REFERENCES `req_spec_section_type` (`slug`)
);

CREATE TABLE IF NOT EXISTS `ent_entity_section` (
    `id`                VARCHAR(36) NOT NULL,
    `entity_id`         VARCHAR(36) NOT NULL,
    `section_type_slug` VARCHAR(64)  NOT NULL,
    `body`              TEXT,
    `created_at`        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`        DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_entity_section_owner_type` (`entity_id`, `section_type_slug`),
    INDEX `idx_entity_section_entity` (`entity_id`),
    INDEX `idx_entity_section_type` (`section_type_slug`),
    CONSTRAINT `fk_entity_section_entity` FOREIGN KEY (`entity_id`) REFERENCES `ent_entity` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_entity_section_type` FOREIGN KEY (`section_type_slug`) REFERENCES `ent_entity_section_type` (`slug`)
);

-- ---- seed the curated vocabularies (INSERT IGNORE → idempotent, never clobbers) -----
-- preamble/more_info are headingless (title NULL, level 0). edge_cases/more_info are
-- "block-owned": rendered inside their structural anchor, not the generic position loop.

INSERT IGNORE INTO `req_spec_section_type` (`slug`, `title`, `level`, `position`, `description`) VALUES
    ('preamble',         NULL,               0,  5, 'Intro prose before the first heading.'),
    ('overview',         'Overview',         2, 10, 'What the spec covers and why.'),
    ('edge_cases',       'Edge Cases',       3, 20, 'Boundary/error scenarios (rendered under User Scenarios & Testing).'),
    ('more_info',        NULL,               0, 30, 'Supplementary requirements-area prose (rendered after the FR list).'),
    ('success_criteria', 'Success Criteria', 2, 40, 'Measurable definition of done.'),
    ('assumptions',      'Assumptions',      2, 50, 'Assumptions the spec relies on.'),
    ('scope',            'Scope',            2, 60, 'What is in and out of scope.'),
    ('open_questions',   'Open Questions',   2, 70, 'Unresolved questions and clarifications.'),
    ('notes',            'Notes',            2, 80, 'Miscellaneous notes; the catch-all for uncategorized prose.');

INSERT IGNORE INTO `ent_entity_section_type` (`slug`, `title`, `level`, `position`, `description`) VALUES
    ('preamble',         NULL,               0,  5, 'Intro prose before the first heading.'),
    ('purpose',          'Purpose',          2, 10, 'What the entity represents and why it exists.'),
    ('key_concepts',     'Key Concepts',     2, 20, 'Core ideas needed to understand the entity.'),
    ('schema_reference', 'Schema Reference', 2, 30, 'Pointer to the technical schema/columns.'),
    ('relationships',    'Relationships',    2, 40, 'How this entity relates to others.'),
    ('business_rules',   'Business Rules',   2, 50, 'Domain rules governing the entity.'),
    ('validations',      'Validations',      2, 60, 'Constraints/validations on the entity.'),
    ('access_control',   'Access Control',   2, 70, 'Who may see/act on rows, in domain terms.'),
    ('notes',            'Notes',            2, 80, 'Miscellaneous notes; the catch-all for uncategorized prose.'),
    ('references',       'References',       2, 90, 'Links to related specs/docs.');

-- ---- retire the polymorphic table (instances repopulate on re-import) ---------------
DROP TABLE IF EXISTS `req_doc_section`;
