-- 0016_abbreviation_slug.up.sql — rename the `abbreviation` business-key columns to
-- `slug`, so every reference/lookup identifier column in the schema is consistently named
-- `slug` (domain, milestone, section types, delivery_status, glossary). The columns are
-- UNIQUE keys; RENAME COLUMN carries the key (verified on Dolt 2.1.7).
ALTER TABLE `req_domain`     RENAME COLUMN `abbreviation` TO `slug`;
ALTER TABLE `plan_milestone` RENAME COLUMN `abbreviation` TO `slug`;
