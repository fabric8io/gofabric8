-- create a revision table for comments, using the some columns + identity of the user and timestamp of the operation
CREATE TABLE comment_revisions (
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    revision_time timestamp with time zone default current_timestamp,
    revision_type int NOT NULL,
    modifier_id uuid NOT NULL,
    comment_id uuid NOT NULL,
    comment_body text,
    comment_markup text
);

CREATE INDEX comment_revisions_comment_id_idx ON comment_revisions USING BTREE (comment_id);

ALTER TABLE comment_revisions
    ADD CONSTRAINT comment_revisions_identity_fk FOREIGN KEY (modifier_id) REFERENCES identities(id);

-- delete comment revisions when the comment is deleted from the database.
ALTER TABLE comment_revisions
    ADD CONSTRAINT comment_revisions_comments_fk FOREIGN KEY (comment_id) REFERENCES comments(id) ON DELETE CASCADE;




