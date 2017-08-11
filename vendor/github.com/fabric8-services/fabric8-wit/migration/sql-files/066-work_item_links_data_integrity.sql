--- remove any record in the 'work_item_links' table if the 'link_type_id', 'source_id' and 'target_id' columns contain `NULL` values
delete from work_item_links where link_type_id IS NULL or source_id IS NULL or target_id IS NULL;

--- make the 'link_type_id', 'source_id' and 'target_id' columns not nullable in the 'work_item_links' table
alter table work_item_links alter column link_type_id set not null;
alter table work_item_links alter column source_id set not null;
alter table work_item_links alter column target_id set not null;