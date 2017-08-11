-- first, we ADD a new COLUMN for the 'natural id' with the same values as the 'id'
ALTER TABLE work_items ADD COLUMN "number" integer;
UPDATE work_items SET number = id;

-- then remove existing CONSTRAINTs and TRIGGERs FROM other tables before changing the 'id' COLUMN
ALTER TABLE work_item_links DROP CONSTRAINT work_item_links_source_id_fkey;
ALTER TABLE work_item_links DROP CONSTRAINT work_item_links_target_id_fkey;
ALTER TABLE work_item_revisions DROP CONSTRAINT work_item_revisions_work_items_fk;
ALTER TABLE work_items DROP CONSTRAINT work_items_pkey;
DROP TRIGGER upd_tsvector ON work_items;
DROP FUNCTION IF EXISTS workitem_tsv_TRIGGER() CASCADE;
DROP INDEX IF EXISTS fulltext_search_index;


-- RENAME COLUMNs of other tables referencing a work item id
ALTER TABLE work_item_links RENAME COLUMN "source_id" TO "source_id_old";
ALTER TABLE work_item_links RENAME COLUMN "target_id" TO "target_id_old";
ALTER TABLE work_item_link_revisions RENAME COLUMN "work_item_link_source_id" TO "work_item_link_source_id_old";
ALTER TABLE work_item_link_revisions RENAME COLUMN "work_item_link_target_id" TO "work_item_link_target_id_old";
ALTER TABLE work_item_revisions RENAME COLUMN "work_item_id" TO "work_item_id_old";

-- ADD new COLUMNs
ALTER TABLE work_item_links ADD COLUMN "source_id" UUID;
ALTER TABLE work_item_links ADD COLUMN "target_id" UUID;
ALTER TABLE work_item_link_revisions ADD COLUMN "work_item_link_source_id" UUID;
ALTER TABLE work_item_link_revisions ADD COLUMN "work_item_link_target_id" UUID;
ALTER TABLE work_item_revisions ADD COLUMN "work_item_id" UUID;

-- assign new UUIDs TO the existing work items
ALTER TABLE work_items DROP COLUMN "id";
ALTER TABLE work_items ADD COLUMN "id" UUID default uuid_generate_v4();
UPDATE work_items SET id = uuid_generate_v4();
-- apply generated UUIDs in other tables
UPDATE work_item_links SET source_id = w.id FROM work_items w WHERE w.number = work_item_links.source_id_old;
UPDATE work_item_links SET target_id = w.id FROM work_items w WHERE w.number = work_item_links.target_id_old;
UPDATE work_item_link_revisions SET work_item_link_source_id = w.id FROM work_items w WHERE w.number = work_item_link_revisions.work_item_link_source_id_old;
UPDATE work_item_link_revisions SET work_item_link_target_id = w.id FROM work_items w WHERE w.number = work_item_link_revisions.work_item_link_target_id_old;
UPDATE work_item_revisions SET work_item_id = w.id FROM work_items w WHERE w.number = work_item_revisions.work_item_id_old;
UPDATE comments SET parent_id = w.id FROM work_items w WHERE w.number::text = comments.parent_id;

-- Drop old columns
ALTER TABLE work_item_links DROP COLUMN "source_id_old";
ALTER TABLE work_item_links DROP COLUMN "target_id_old";
ALTER TABLE work_item_revisions DROP COLUMN "work_item_id_old";
ALTER TABLE work_item_link_revisions DROP COLUMN "work_item_link_source_id_old";
ALTER TABLE work_item_link_revisions DROP COLUMN "work_item_link_target_id_old";

-- recreate constraints, FK and triggers
ALTER TABLE work_items ADD CONSTRAINT work_items_pkey PRIMARY KEY (id);
ALTER TABLE work_item_links ADD CONSTRAINT work_item_links_source_id_fkey FOREIGN KEY (source_id) REFERENCES work_items(id) ON DELETE CASCADE;
ALTER TABLE work_item_links ADD CONSTRAINT work_item_links_target_id_fkey FOREIGN KEY (target_id) REFERENCES work_items(id) ON DELETE CASCADE;
ALTER TABLE work_item_revisions ADD CONSTRAINT work_item_revisions_work_items_fk FOREIGN KEY (work_item_id) REFERENCES work_items(id) ON DELETE CASCADE;
CREATE UNIQUE INDEX work_item_links_unique_idx ON work_item_links (source_id, target_id, link_type_id) WHERE deleted_at IS NULL;

-- create the work item number sequences table
CREATE TABLE work_item_number_sequences (
    space_id uuid primary key,
    current_val integer not null
);
ALTER TABLE work_item_number_sequences ADD CONSTRAINT "work_item_number_sequences_space_id_fkey" FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE;

-- fill the work item ID sequence table with the current 'max' value of issue 'number'
INSERT INTO work_item_number_sequences (space_id, current_val) (select space_id, max(number) FROM work_items group by space_id);

-- ADD unique index ON the work_items table: a 'number' is unique per 'space_id' and those 2 COLUMNs are used TO look-up work items
CREATE UNIQUE INDEX uix_work_items_spaceid_number ON work_items using btree (space_id, number);


-- Restore search capabilities
CREATE INDEX fulltext_search_index ON work_items USING GIN (tsv);

-- UPDATE the 'tsv' COLUMN with the text value of the existing 'content' 
-- element in the 'system.description' JSON document
UPDATE work_items SET tsv =
    setweight(to_tsvector('english', "number"::text),'A') ||
    setweight(to_tsvector('english', coalesce(fields->>'system.title','')),'B') ||
    setweight(to_tsvector('english', coalesce(fields#>>'{system.description, content}','')),'C');

-- fill the 'tsv' COLUMN with the text value of the created/modified 'content' 
-- element in the 'system.description' JSON document
CREATE FUNCTION workitem_tsv_TRIGGER() RETURNS TRIGGER AS $$
begin
  new.tsv :=
    setweight(to_tsvector('english', new.number::text),'A') ||
    setweight(to_tsvector('english', coalesce(new.fields->>'system.title','')),'B') ||
    setweight(to_tsvector('english', coalesce(new.fields#>>'{system.description, content}','')),'C');
  return new;
end
$$ LANGUAGE plpgsql; 

CREATE TRIGGER upd_tsvector BEFORE INSERT OR UPDATE OF number, fields ON work_items
FOR EACH ROW EXECUTE PROCEDURE workitem_tsv_TRIGGER();