--#################################################################################
-- When a work item gets soft deleted, soft delete any work item link in existence.
--#################################################################################

CREATE FUNCTION update_WIL_after_WI() RETURNS trigger AS $update_WIL_after_WI$
    BEGIN
        UPDATE work_item_links SET deleted_at = NEW.deleted_at WHERE NEW.id IN (source_id, target_id);
        RETURN NEW;
    END;
$update_WIL_after_WI$ LANGUAGE plpgsql;

CREATE TRIGGER update_WIL_after_WI_trigger
AFTER UPDATE OF deleted_at
ON work_items
FOR EACH ROW
EXECUTE PROCEDURE update_WIL_after_WI();

--###########################################################################################
-- When a work item type gets soft deleted, soft delete any work item link type in existence.
--###########################################################################################

CREATE FUNCTION update_WILT_after_WIT() RETURNS trigger AS $update_WILT_after_WIT$
    BEGIN
        UPDATE work_item_link_types SET deleted_at = NEW.deleted_at WHERE NEW.name IN (source_type_name, target_type_name);
        RETURN NEW;
    END;
$update_WILT_after_WIT$ LANGUAGE plpgsql;

CREATE TRIGGER update_WILT_after_WIT_trigger
AFTER UPDATE OF deleted_at
ON work_item_types
FOR EACH ROW
EXECUTE PROCEDURE update_WILT_after_WIT();

--##################################################################################################
-- When a work item link category is soft deleted, soft delete any work item link type in existence.
--##################################################################################################

CREATE FUNCTION update_WILT_after_WILC() RETURNS trigger AS $update_WILT_after_WILC$
    BEGIN
        UPDATE work_item_link_types SET deleted_at = NEW.deleted_at WHERE link_category_id = NEW.id;
        RETURN NEW;
    END;
$update_WILT_after_WILC$ LANGUAGE plpgsql;

CREATE TRIGGER update_WILT_after_WILC_trigger
AFTER UPDATE OF deleted_at
ON work_item_link_categories
FOR EACH ROW
EXECUTE PROCEDURE update_WILT_after_WILC();

--##########################################################################################
-- When a work item link type is soft deleted, soft delete any work item links in existence.
--##########################################################################################

CREATE FUNCTION update_WIL_after_WILT() RETURNS trigger AS $update_WIL_after_WILT$
    BEGIN
        UPDATE work_item_links SET deleted_at = NEW.deleted_at WHERE link_type_id = NEW.id;
        RETURN NEW;
    END;
$update_WIL_after_WILT$ LANGUAGE plpgsql;

CREATE TRIGGER update_WIL_after_WILT_trigger
AFTER UPDATE OF deleted_at
ON work_item_link_types
FOR EACH ROW
EXECUTE PROCEDURE update_WIL_after_WILT();

--###############################################################################
-- When a work item type is soft deleted, soft delete any work item in existence.
--###############################################################################

CREATE FUNCTION update_WI_after_WIT() RETURNS trigger AS $update_WI_after_WIT$
    BEGIN
        UPDATE work_items SET deleted_at = NEW.deleted_at WHERE type = NEW.name;
        RETURN NEW;
    END;
$update_WI_after_WIT$ LANGUAGE plpgsql;

CREATE TRIGGER update_WI_after_WILT_trigger
AFTER UPDATE OF deleted_at
ON work_item_types
FOR EACH ROW
EXECUTE PROCEDURE update_WI_after_WIT();