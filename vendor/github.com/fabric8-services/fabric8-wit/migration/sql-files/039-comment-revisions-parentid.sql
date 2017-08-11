-- add a 'comment_parent_id' column in the 'comment_revisions' table
ALTER TABLE comment_revisions ADD COLUMN comment_parent_id text;
-- fill the new column 
update comment_revisions set comment_parent_id = c.parent_id from comments c where c.id = comment_id;
-- make the new column 'not null'
ALTER TABLE comment_revisions ALTER COLUMN comment_parent_id SET NOT NULL;
-- make sure the new column cannot be filled with empty content
ALTER TABLE comment_revisions ADD CONSTRAINT comment_parent_id_check CHECK (comment_parent_id <> '');