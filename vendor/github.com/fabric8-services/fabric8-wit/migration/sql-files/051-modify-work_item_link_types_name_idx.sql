-- When created, a work item link type didn't know about any space and thus its
-- name was only allowed to be used once per link category. Now, with spaces,
-- the same unique index shall span the name, the category and the space.

DROP INDEX work_item_link_types_name_idx;

CREATE UNIQUE INDEX work_item_link_types_name_idx ON work_item_link_types (name, space_id, link_category_id) WHERE deleted_at IS NULL;