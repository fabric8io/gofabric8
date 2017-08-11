-- 1. First make sure that the link type IDs are consistent across all systems.
--    We therefore copy the existing link types and only change the ID value to
--    the ones pre-defined in code.
-- 2. Then we make sure all existing links that pointed to the old ID now point
--    to the duplicated row with the new ID.
-- 3. Then we can delete all the old link types.

-- The same we do for the "user" and "system" link categories.

--------------------
-- HANDLE LINK TYPES
--------------------

-- Duplicate the old SystemWorkItemLinkTypeBugBlocker and use a new ID
-- We have to use a different name in order to not violate the uniq key
-- "work_item_link_types_name_idx". The name will later be updated through
-- the migration.
INSERT INTO work_item_link_types(id, created_at, updated_at, deleted_at,
    name, version, topology, forward_name, reverse_name, link_category_id,
    source_type_id, target_type_id, space_id)
    SELECT '{{index . 0}}', created_at, updated_at, deleted_at,
    '{{index . 0}}', version, topology, forward_name, reverse_name, link_category_id,
    source_type_id, target_type_id, space_id 
    FROM work_item_link_types
    WHERE name='Bug blocker';

-- Duplicate the old SystemWorkItemLinkPlannerItemRelated and use a new ID
INSERT INTO work_item_link_types(id, created_at, updated_at, deleted_at,
    name, version, topology, forward_name, reverse_name, link_category_id,
    source_type_id, target_type_id, space_id)
    SELECT '{{index . 1}}', created_at, updated_at, deleted_at,
    '{{index . 1}}', version, topology, forward_name, reverse_name, link_category_id,
    source_type_id, target_type_id, space_id 
    FROM work_item_link_types
    WHERE name='Related planner item';

INSERT INTO work_item_link_types(id, created_at, updated_at, deleted_at,
    name, version, topology, forward_name, reverse_name, link_category_id,
    source_type_id, target_type_id, space_id)
    SELECT '{{index . 2}}', created_at, updated_at, deleted_at,
    '{{index . 2}}', version, topology, forward_name, reverse_name, link_category_id,
    source_type_id, target_type_id, space_id 
    FROM work_item_link_types
    WHERE name='Parent child item';

-- Update existing links to use the new link type ID
UPDATE work_item_links SET link_type_id='{{index . 0}}'
    WHERE link_type_id = (SELECT id FROM work_item_link_types WHERE name='Bug blocker' AND id <> '{{index . 0}}');

UPDATE work_item_links SET link_type_id='{{index . 1}}'
    WHERE link_type_id = (SELECT id FROM work_item_link_types WHERE name='Related planner item' AND id <> '{{index . 1}}');
    
UPDATE work_item_links SET link_type_id='{{index . 2}}'
    WHERE link_type_id = (SELECT id FROM work_item_link_types WHERE name='Parent child item' AND id <> '{{index . 2}}');

-- Delete old link types and only leave the new ones.
DELETE FROM work_item_link_types WHERE id NOT IN ('{{index . 0}}', '{{index . 1}}', '{{index . 2}}');

--------------------
-- HANDLE CATEGORIES
--------------------

-- Duplicate "system" link category
INSERT INTO work_item_link_categories (id, created_at, updated_at, deleted_at, name, version, description)
    SELECT '{{index . 3}}', created_at, updated_at, deleted_at, '{{index . 3}}', version, description
    FROM work_item_link_categories
    WHERE name='system';

-- Duplicate "user" link category
INSERT INTO work_item_link_categories (id, created_at, updated_at, deleted_at, name, version, description)
    SELECT '{{index . 4}}', created_at, updated_at, deleted_at, '{{index . 4}}', version, description
    FROM work_item_link_categories
    WHERE name='user';

-- Update existing link types to use the new category IDs
UPDATE work_item_link_types SET link_category_id='{{index . 3}}'
    WHERE link_category_id = (SELECT id FROM work_item_link_categories WHERE name='system' AND id <> '{{index . 3}}');

UPDATE work_item_link_types SET link_category_id='{{index . 4}}'
    WHERE link_category_id = (SELECT id FROM work_item_link_categories WHERE name='user' AND id <> '{{index . 4}}');

-- Delete old link categories and only leave the new ones.
DELETE FROM work_item_link_categories WHERE id NOT IN ('{{index . 3}}', '{{index . 4}}');
