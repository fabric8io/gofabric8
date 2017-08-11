-- drop existing unique index
DROP INDEX spaces_name_idx;
-- rename duplicate spaces in existence and keep only one as it was
UPDATE spaces SET name = CONCAT(lower(name), '-renamed')
WHERE id IN (
    SELECT id
    FROM (
        SELECT id, ROW_NUMBER() OVER (partition BY owner_id, lower(name) ORDER BY id) AS rnum
        FROM spaces
    ) t
    WHERE t.rnum > 1
);
-- recreate unique index with original index and lowercase name, on two columns
CREATE UNIQUE INDEX spaces_name_idx ON spaces (owner_id, lower(name)) WHERE deleted_at IS NULL;
