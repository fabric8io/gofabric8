-- Add field on work_item  to store Full Text Search Vector
ALTER TABLE work_items ADD tsv tsvector;

UPDATE work_items SET tsv =
    setweight(to_tsvector('english', id::text),'A') ||
    setweight(to_tsvector('english', coalesce(fields->>'system.title','')),'B') ||
    setweight(to_tsvector('english', coalesce(fields->>'system.description','')),'C');

CREATE INDEX fulltext_search_index ON work_items USING GIN (tsv);

CREATE FUNCTION workitem_tsv_trigger() RETURNS trigger AS $$
begin
  new.tsv :=
    setweight(to_tsvector('english', new.id::text),'A') ||
    setweight(to_tsvector('english', coalesce(new.fields->>'system.title','')),'B') ||
    setweight(to_tsvector('english', coalesce(new.fields->>'system.description','')),'C');
  return new;
end
$$ LANGUAGE plpgsql;

CREATE TRIGGER upd_tsvector BEFORE INSERT OR UPDATE OF id, fields ON work_items
FOR EACH ROW EXECUTE PROCEDURE workitem_tsv_trigger();