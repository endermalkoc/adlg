-- 0015_timestamps.down.sql — revert to the prior timestamp shape: drop the columns this
-- migration added, and restore the modified ones to nullable / no-default.

ALTER TABLE `ent_attribute`           DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `ent_entity`              DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `ent_relationship`        DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `plan_capability`         DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `plan_deliverable`        DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `plan_view`               DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `pub_external_ref`        DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `req_acceptance_scenario` DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `req_domain`              DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `req_requirement_group`   DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `req_user_story`          DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `rev_actor`               DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `test_configuration`      DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `test_result`             DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `test_run`                DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `test_step`               DROP COLUMN `created_at`, DROP COLUMN `updated_at`;
ALTER TABLE `test_suite`              DROP COLUMN `created_at`, DROP COLUMN `updated_at`;

ALTER TABLE `rev_review` DROP COLUMN `updated_at`, MODIFY COLUMN `created_at` DATETIME;

ALTER TABLE `plan_milestone`   MODIFY COLUMN `created_at` DATETIME, MODIFY COLUMN `updated_at` DATETIME;
ALTER TABLE `req_glossary_term` MODIFY COLUMN `created_at` DATETIME, MODIFY COLUMN `updated_at` DATETIME;
ALTER TABLE `req_requirement`   MODIFY COLUMN `created_at` DATETIME, MODIFY COLUMN `updated_at` DATETIME;
ALTER TABLE `req_spec`          MODIFY COLUMN `created_at` DATETIME, MODIFY COLUMN `updated_at` DATETIME;
ALTER TABLE `rev_changeset`     MODIFY COLUMN `created_at` DATETIME, MODIFY COLUMN `updated_at` DATETIME;
ALTER TABLE `rev_comment`       MODIFY COLUMN `created_at` DATETIME, MODIFY COLUMN `updated_at` DATETIME;
ALTER TABLE `test_case`         MODIFY COLUMN `created_at` DATETIME, MODIFY COLUMN `updated_at` DATETIME;
