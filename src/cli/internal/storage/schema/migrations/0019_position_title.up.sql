-- 0019_position_title.up.sql — settle two naming inconsistencies on one term each.
--
-- 1. ordinal → position. Document/order columns were named `ordinal` on some tables
--    (user_story, acceptance_scenario, test_step) and `position` on others
--    (test_suite, entity_attribute, requirement_group, requirement, *_section_type).
--    They mean the same thing; standardize on `position` everywhere. RENAME COLUMN
--    carries the data and any index reference (verified on Dolt, see 0016); the
--    user_story UNIQUE key embeds the old name, so it is re-created.
--
-- 2. Drop req_spec.heading. The spec carried BOTH `title` (the frontmatter label) and
--    `heading` (the verbatim H1, e.g. "Feature Specification: Add New Student"). They
--    are not redundant — `heading` adds a kind-prefix and fuller wording — but the
--    prefix is a corpus-ism derivable from spec.kind, so we settle on `title` as the
--    single spec label and render the H1 as `# {title}` (decisions.md). DDL only; the
--    H1 was import-populated, so nothing else needs backfilling.

ALTER TABLE `req_user_story` DROP INDEX `uk_user_story_spec_ordinal`;
ALTER TABLE `req_user_story` RENAME COLUMN `ordinal` TO `position`;
ALTER TABLE `req_user_story` ADD UNIQUE KEY `uk_user_story_spec_position` (`spec_id`, `position`);

ALTER TABLE `req_acceptance_scenario` RENAME COLUMN `ordinal` TO `position`;
ALTER TABLE `test_step`               RENAME COLUMN `ordinal` TO `position`;

ALTER TABLE `req_spec` DROP COLUMN `heading`;
