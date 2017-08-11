-- add a column to record the timestamp of the latest addition/change/removal of an entity in relationship with a workitem
ALTER TABLE work_items ADD COLUMN relationships_changed_at timestamp with time zone;
COMMENT ON COLUMN work_items.relationships_changed_at IS 'see triggers on the ''comments'' and ''work_item_links tables''.';

CREATE FUNCTION workitem_comment_insert_timestamp() RETURNS trigger AS $$
    -- trigger to fill the `relationships_changed_at` column when a comment is added
    BEGIN
        UPDATE work_items wi SET relationships_changed_at = NEW.created_at WHERE wi.id::text = NEW.parent_id;
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;

CREATE FUNCTION workitem_comment_softdelete_timestamp() RETURNS trigger AS $$
    -- trigger to fill the `commented_at` column when a comment is removed (soft delete, it's a record update)
    BEGIN
        UPDATE work_items wi SET relationships_changed_at = NEW.deleted_at WHERE wi.id::text = NEW.parent_id;
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
    

CREATE FUNCTION workitem_link_insert_timestamp() RETURNS trigger AS $$
    -- trigger to fill the `relationships_changed_at` column when a link is added
    BEGIN
        UPDATE work_items wi SET relationships_changed_at = NEW.created_at WHERE wi.id in (NEW.source_id, NEW.target_id);
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;

CREATE FUNCTION workitem_link_softdelete_timestamp() RETURNS trigger AS $$
    -- trigger to fill the `relationships_changed_at` column when a link is removed (soft delete, it's a record update)
    BEGIN
        UPDATE work_items wi SET relationships_changed_at = NEW.deleted_at WHERE wi.id in (NEW.source_id, NEW.target_id);
        RETURN NEW;
    END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER workitem_link_insert_trigger AFTER INSERT ON work_item_links 
    FOR EACH ROW
    WHEN (NEW.deleted_at IS NULL)
    EXECUTE PROCEDURE workitem_link_insert_timestamp();
    
CREATE TRIGGER workitem_link_softdelete_trigger AFTER UPDATE OF deleted_at ON work_item_links 
    FOR EACH ROW
    WHEN (NEW.deleted_at IS NOT NULL)
    EXECUTE PROCEDURE workitem_link_softdelete_timestamp();
