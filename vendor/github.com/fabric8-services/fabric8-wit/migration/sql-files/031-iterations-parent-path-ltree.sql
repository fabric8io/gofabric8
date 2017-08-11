CREATE EXTENSION IF NOT EXISTS "ltree";

-- Rename parent_id column
ALTER TABLE iterations RENAME parent_id to path;

-- Need to convert the path column to text in order to
-- replace non-locale characters with an underscore
ALTER TABLE iterations ALTER path TYPE text USING path::text;

-- Need to update values of Iteration's' ParentID in order to migrate it to ltree
-- Replace every non-C-LOCALE character with an underscore
UPDATE iterations SET path = regexp_replace(path, '[^a-zA-Z0-9_\.]', '_', 'g');

-- Finally values in path are now in good shape for ltree and can be casted automatically to type ltree
-- Convert the parent column from type uuid to ltree
ALTER TABLE iterations ALTER path TYPE ltree USING path::ltree;

-- Enable full text search operaions using GIST index on path
CREATE INDEX iteration_path_gist_idx ON iterations USING GIST (path);
