-- first, we ADD a new COLUMN for the 'parent id' as a UUID in the `comments` table:
ALTER TABLE comments ADD COLUMN "parent_id_uuid" UUID;
UPDATE comments SET parent_id_uuid = parent_id::uuid;
-- then drop the old 'parent_id' column and rename the new one to 'parent_id'
ALTER TABLE comments DROP COLUMN "parent_id";
ALTER TABLE comments RENAME COLUMN "parent_id_uuid" TO "parent_id";

-- second, we ADD a new COLUMN for the 'parent id' as a UUID in the `comment_revisions` table (after migrating the content of `comment_parent_id`, forgotten in step 65) :
ALTER TABLE comment_revisions ADD COLUMN "comment_parent_id_uuid" UUID;
UPDATE comment_revisions SET comment_parent_id_uuid = c.parent_id FROM comments c WHERE c.id = comment_revisions.comment_id;
-- then drop the old 'parent_id' column and rename the new one to 'parent_id'
ALTER TABLE comment_revisions DROP COLUMN "comment_parent_id";
ALTER TABLE comment_revisions RENAME COLUMN "comment_parent_id_uuid" TO "comment_parent_id";

-- also, we need to update the triggers that record the 'relationships_changed_at' value in the 'work_items' table:
DROP TRIGGER workitem_comment_insert_trigger ON comments;
DROP TRIGGER workitem_comment_softdelete_trigger ON comments;
DROP FUNCTION workitem_comment_insert_timestamp();
DROP FUNCTION workitem_comment_softdelete_timestamp();

CREATE FUNCTION workitem_comment_insert_timestamp() RETURNS trigger AS $$
    -- trigger to fill the `relationships_changed_at` column when a comment is added
    BEGIN
        UPDATE work_items wi SET relationships_changed_at = NEW.created_at WHERE wi.id = NEW.parent_id;
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;

CREATE FUNCTION workitem_comment_softdelete_timestamp() RETURNS trigger AS $$
    -- trigger to fill the `commented_at` column when a comment is removed (soft delete, it's a record update)
    BEGIN
        UPDATE work_items wi SET relationships_changed_at = NEW.deleted_at WHERE wi.id = NEW.parent_id;
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER workitem_comment_insert_trigger AFTER INSERT ON comments 
    FOR EACH ROW
    WHEN (NEW.deleted_at IS NULL)
    EXECUTE PROCEDURE workitem_comment_insert_timestamp();

CREATE TRIGGER workitem_comment_softdelete_trigger AFTER UPDATE OF deleted_at ON comments 
    FOR EACH ROW
     WHEN (NEW.deleted_at IS NOT NULL)
    EXECUTE PROCEDURE workitem_comment_softdelete_timestamp();