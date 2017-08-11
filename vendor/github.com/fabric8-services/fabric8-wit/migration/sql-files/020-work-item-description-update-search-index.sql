-- migrate work items description
update work_items set fields=jsonb_set(fields, '{system.description}', 
  jsonb_build_object('content', fields->>'system.description', 'markup', 'plain'))
  where fields->>'system.description' is not null;


-- update support for Full Text Search Vector on work item description
DROP TRIGGER IF EXISTS upd_tsvector ON work_items;
DROP FUNCTION IF EXISTS workitem_tsv_trigger() CASCADE;
DROP INDEX IF EXISTS fulltext_search_index;
CREATE INDEX fulltext_search_index ON work_items USING GIN (tsv);

-- update the 'tsv' column with the text value of the existing 'content' 
-- element in the 'system.description' JSON document
UPDATE work_items SET tsv =
    setweight(to_tsvector('english', id::text),'A') ||
    setweight(to_tsvector('english', coalesce(fields->>'system.title','')),'B') ||
    setweight(to_tsvector('english', coalesce(fields#>>'{system.description, content}','')),'C');

-- fill the 'tsv' column with the text value of the created/modified 'content' 
-- element in the 'system.description' JSON document
CREATE FUNCTION workitem_tsv_trigger() RETURNS trigger AS $$
begin
  new.tsv :=
    setweight(to_tsvector('english', new.id::text),'A') ||
    setweight(to_tsvector('english', coalesce(new.fields->>'system.title','')),'B') ||
    setweight(to_tsvector('english', coalesce(new.fields#>>'{system.description, content}','')),'C');
  return new;
end
$$ LANGUAGE plpgsql; 

CREATE TRIGGER upd_tsvector BEFORE INSERT OR UPDATE OF id, fields ON work_items
FOR EACH ROW EXECUTE PROCEDURE workitem_tsv_trigger();