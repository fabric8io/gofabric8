-- Alter the table work_item_link_types
INSERT INTO spaces (created_at, updated_at, id, name, description) VALUES (now(),now(), '{{index . 0}}','{{index . 1}}','{{index . 2}}');
ALTER TABLE work_item_link_types ADD space_id uuid DEFAULT '{{index . 0}}' NOT NULL;
-- Once we set the values to the default. We drop this default constraint
ALTER TABLE work_item_link_types ALTER space_id DROP DEFAULT;

ALTER TABLE work_item_link_types ADD FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE;

-- Create indexes
CREATE INDEX ix_space_id ON work_item_link_types USING btree (space_id);
