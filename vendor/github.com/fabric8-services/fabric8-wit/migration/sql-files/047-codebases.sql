CREATE TABLE codebases (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    space_id uuid NOT NULL REFERENCES spaces (id) ON DELETE CASCADE,
    type text,
    url text
);

CREATE INDEX ix_codebases_space_id ON codebases USING btree (space_id);