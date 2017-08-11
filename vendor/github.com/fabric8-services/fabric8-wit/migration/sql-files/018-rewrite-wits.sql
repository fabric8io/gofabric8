-- See https://www.postgresql.org/docs/current/static/ltree.html for the
-- reference See http://leapfrogonline.io/articles/2015-05-21-postgres-ltree/
-- for an explanation
CREATE EXTENSION IF NOT EXISTS "ltree";

-- The following update needs to be done in order to get the WIT storage in a
-- good shape for it to be migrated to an ltree
UPDATE work_item_types SET
    -- Remove any leading '/' from the WIT's path.
    -- Remove any occurence of 'system.'.
    -- Replace '/' with '.' as the new path separator for use with ltree.
    -- Replace every non-C-LOCALE character with an underscore (the "." is an
    -- exception because it will be used by ltree)
    path =  regexp_replace(
                replace(replace(ltrim(path, '/'), 'system.', ''), '/', '.'),
                '[^a-zA-Z0-9_\.]',
                '_'
            )
    ;

-- Convert the path column from type text to ltree
ALTER TABLE work_item_types ALTER COLUMN path TYPE ltree USING path::ltree;

-- Add a constraint to the work item type name 
ALTER TABLE work_item_types ADD CONSTRAINT work_item_link_types_check_name_c_locale CHECK (name ~ '[a-zA-Z0-9_]');

-- Add indexes 
CREATE INDEX wit_path_gist_idx ON work_item_types USING GIST (path);
CREATE INDEX wit_path_idx ON work_item_types USING BTREE (path);


---------------------------------------------------------------------------
-- Update work items and work item link types that point to the work items.
---------------------------------------------------------------------------


-- Drop the foreign keys on the work item links types that reference the work
-- item type. We add them back once we've changed the names and it is safe to
-- add the keys back again.
ALTER TABLE work_item_link_types DROP CONSTRAINT work_item_link_types_source_type_name_fkey;
ALTER TABLE work_item_link_types DROP CONSTRAINT work_item_link_types_target_type_name_fkey;

UPDATE work_item_link_types
SET
    source_type_name = subpath(wit_source.path, -1, 1),
    target_type_name = subpath(wit_target.path, -1, 1)
FROM
    work_item_types AS wit_source,
    work_item_types AS wit_target
WHERE
    source_type_name = wit_source.name
    AND target_type_name = wit_target.name;

-- Update work item's type
UPDATE work_items
SET type = subpath(wit.path, -1, 1)
FROM work_item_types AS wit
WHERE type = wit.name;

-- Use the leaf of the path "tree" as the name of the work item type
UPDATE work_item_types SET name = subpath(path, -1, 1);

-- Add foreign keys back in
ALTER TABLE work_item_link_types
    ADD CONSTRAINT "work_item_link_types_source_type_name_fkey"
    FOREIGN KEY (source_type_name)
    REFERENCES work_item_types(name)
    ON DELETE CASCADE;

ALTER TABLE work_item_link_types
    ADD CONSTRAINT "work_item_link_types_target_type_name_fkey"
    FOREIGN KEY (target_type_name)
    REFERENCES work_item_types(name)
    ON DELETE CASCADE;
