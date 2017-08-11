-- Dropping all auto soft-delete triggers

-- If a WIL is manually deleted, then recreated with same target/source/type, and a WIT is updated,
-- all WIL of that type is reset to have same state as the WIT and causing unique constraint problems.

-- A WIT can not be deleted, it will only be disabled from view to be created
-- A WI is only ever soft deleted, but a WIL to a deleted item should still be displayable.
-- A WITC delete is a process and can probably never be deleted without a large user question of; what do you want to do with these?
-- A WILT delete can never happen, or similar to above.

DROP TRIGGER update_WIL_after_WI_trigger ON work_items;
DROP FUNCTION update_WIL_after_WI();

DROP TRIGGER update_WILT_after_WIT_trigger ON work_item_types;
DROP FUNCTION update_WILT_after_WIT();

DROP TRIGGER update_WILT_after_WILC_trigger ON work_item_link_categories;
DROP FUNCTION update_WILT_after_WILC();

DROP TRIGGER update_WIL_after_WILT_trigger ON work_item_link_types;
DROP FUNCTION update_WIL_after_WILT();