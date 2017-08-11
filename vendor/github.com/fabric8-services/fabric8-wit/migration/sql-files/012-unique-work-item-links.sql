-- Delete duplicate links in existence and keep only one
-- See here: https://wiki.postgresql.org/wiki/Deleting_duplicates
DELETE FROM work_item_links
WHERE id IN (
    SELECT id
    FROM (
        SELECT id, ROW_NUMBER() OVER (partition BY link_type_id, source_id, target_id ORDER BY id) AS rnum
        FROM work_item_links
    ) t
    WHERE t.rnum > 1
);

-- From now on ensure we only have ONE link with the same source, target and
-- link type in existence. If a link has been deleted (deleted_at != NULL) then
-- we can recreate the link with the source, target and link type again.
CREATE UNIQUE INDEX work_item_links_unique_idx ON work_item_links (source_id, target_id, link_type_id) WHERE deleted_at IS NULL;