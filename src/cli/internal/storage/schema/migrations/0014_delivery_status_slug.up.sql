-- 0014_delivery_status_slug.up.sql — two ergonomic fixes to plan_delivery_status.
--
-- 1. Rename `key` → `slug`. `key` is a SQL reserved word; tools that don't backtick it
--    (GUIs generating an `ORDER BY key` or a bare column list when browsing) fail with a
--    syntax error. `slug` is a non-reserved synonym, matching the 0013 section-type
--    tables and the glossary slug columns. requirement.delivery_status references this
--    value with NO foreign key (the soft-lookup decision in 0009), so it is unaffected.
ALTER TABLE `plan_delivery_status` RENAME COLUMN `key` TO `slug`;

-- 2. Make the timestamps auto-populating + non-nullable (Dolt supports
--    DEFAULT CURRENT_TIMESTAMP / ON UPDATE CURRENT_TIMESTAMP). The 0009 seed left them
--    NULL; backfill before the NOT NULL alter so it succeeds.
UPDATE `plan_delivery_status` SET `created_at` = NOW(), `updated_at` = NOW()
    WHERE `created_at` IS NULL OR `updated_at` IS NULL;
ALTER TABLE `plan_delivery_status`
    MODIFY COLUMN `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    MODIFY COLUMN `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;
