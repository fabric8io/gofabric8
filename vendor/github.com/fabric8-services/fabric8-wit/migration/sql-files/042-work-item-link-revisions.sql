-- create a revision table for work item links, using the some columns + identity of the user and timestamp of the operation
CREATE TABLE work_item_link_revisions (
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    revision_time timestamp with time zone default current_timestamp,
    revision_type int NOT NULL,
    modifier_id uuid NOT NULL,
    work_item_link_id uuid NOT NULL,
    work_item_link_version int NOT NULL,
    work_item_link_source_id bigint NOT NULL,
    work_item_link_target_id bigint NOT NULL,
    work_item_link_type_id uuid NOT NULL
);

CREATE INDEX work_item_link_revisions_work_item_link_id_idx ON work_item_link_revisions USING BTREE (work_item_link_id);

ALTER TABLE work_item_link_revisions
    ADD CONSTRAINT work_item_link_revisions_modifier_id_fk FOREIGN KEY (modifier_id) REFERENCES identities(id);

-- delete work item revisions when the work item is deleted from the database.
ALTER TABLE work_item_link_revisions
    ADD CONSTRAINT work_item_link_revisions_work_item_link_id_fk FOREIGN KEY (work_item_link_id) REFERENCES work_item_links(id) ON DELETE CASCADE;

