-- 0014_delivery_status_slug.down.sql — revert the timestamp defaults and the rename.
ALTER TABLE `plan_delivery_status`
    MODIFY COLUMN `created_at` DATETIME,
    MODIFY COLUMN `updated_at` DATETIME;
ALTER TABLE `plan_delivery_status` RENAME COLUMN `slug` TO `key`;
