-- 0016_abbreviation_slug.down.sql — restore the `abbreviation` column name.
ALTER TABLE `req_domain`     RENAME COLUMN `slug` TO `abbreviation`;
ALTER TABLE `plan_milestone` RENAME COLUMN `slug` TO `abbreviation`;
