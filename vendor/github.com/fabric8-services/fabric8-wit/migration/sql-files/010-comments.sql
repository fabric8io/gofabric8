CREATE TABLE comments (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    parent_id text,
    body text,
    created_by uuid
);

CREATE INDEX ix_parent_id ON comments USING btree (parent_id);