-- 0018_priority_standardize.up.sql — one standard priority taxonomy across the schema.
-- Replaces the two inconsistent priority value-sets (user_story P1–P3, test_case
-- low|medium|high) with a single 0–4 scheme, captured as a seeded reference table
-- (the "table bucket", like plan_delivery_status). Priority is stored as the INT level
-- (0 = most urgent, sortable); the label/description live in req_priority. Soft lookup —
-- no FK — validated in-app, matching the project's value-set convention.

CREATE TABLE IF NOT EXISTS `req_priority` (
    `level`       INT          NOT NULL,                   -- 0 (Critical) .. 4 (Backlog)
    `label`       VARCHAR(64)  NOT NULL,
    `description` TEXT,
    `created_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`level`)
);

INSERT IGNORE INTO `req_priority` (`level`, `label`, `description`) VALUES
    (0, 'Critical', 'Security, data loss, broken builds'),
    (1, 'High',     'Major features, important bugs'),
    (2, 'Medium',   'Nice-to-have features, minor bugs'),
    (3, 'Low',      'Polish, optimization'),
    (4, 'Backlog',  'Future ideas');

-- Convert the existing priority columns to the INT level. Both are empty when this runs
-- on a fresh init (migrations precede import); a re-import repopulates user_story (the
-- importer maps Pn→n). test_case is unused.
ALTER TABLE `req_user_story` MODIFY COLUMN `priority` INT;
ALTER TABLE `test_case`      MODIFY COLUMN `priority` INT;

-- Requirements gain a priority too (NULL = unprioritized; the corpus carries no FR
-- priority yet, so they import as NULL).
ALTER TABLE `req_requirement` ADD COLUMN `priority` INT;
