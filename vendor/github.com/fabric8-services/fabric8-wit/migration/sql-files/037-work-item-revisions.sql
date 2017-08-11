-- create a revision table for work items, using the some columns + identity of the user and timestamp of the operation
CREATE TABLE work_item_revisions (
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    revision_time timestamp with time zone default current_timestamp,
    revision_type int NOT NULL,
    modifier_id uuid NOT NULL,
    work_item_id bigint NOT NULL,
    work_item_type_id uuid,
    work_item_version integer,
    work_item_fields jsonb
);

CREATE INDEX work_item_revisions_work_items_idx ON work_item_revisions USING BTREE (work_item_id);

ALTER TABLE work_item_revisions
    ADD CONSTRAINT work_item_revisions_identity_fk FOREIGN KEY (modifier_id) REFERENCES identities(id);

-- delete work item revisions when the work item is deleted from the database.
ALTER TABLE work_item_revisions
    ADD CONSTRAINT work_item_revisions_work_items_fk FOREIGN KEY (work_item_id) REFERENCES work_items(id) ON DELETE CASCADE;




