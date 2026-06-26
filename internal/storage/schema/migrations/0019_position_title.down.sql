-- 0019_position_title.down.sql — revert the position/title standardization.
-- The dropped heading column comes back empty (its content was import-populated and
-- git-ignored/reproducible, like every prior schema migration's down).
ALTER TABLE `req_spec` ADD COLUMN `heading` TEXT;

ALTER TABLE `test_step`               RENAME COLUMN `position` TO `ordinal`;
ALTER TABLE `req_acceptance_scenario` RENAME COLUMN `position` TO `ordinal`;

ALTER TABLE `req_user_story` DROP INDEX `uk_user_story_spec_position`;
ALTER TABLE `req_user_story` RENAME COLUMN `position` TO `ordinal`;
ALTER TABLE `req_user_story` ADD UNIQUE KEY `uk_user_story_spec_ordinal` (`spec_id`, `ordinal`);
