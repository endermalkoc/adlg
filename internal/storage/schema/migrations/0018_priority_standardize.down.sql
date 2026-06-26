-- 0018_priority_standardize.down.sql — restore the prior per-entity priority shapes.
-- Re-import after rollback (priority values repopulate in the old form).
ALTER TABLE `req_requirement` DROP COLUMN `priority`;
ALTER TABLE `test_case`      MODIFY COLUMN `priority` VARCHAR(16);
ALTER TABLE `req_user_story` MODIFY COLUMN `priority` VARCHAR(8);
DROP TABLE IF EXISTS `req_priority`;
