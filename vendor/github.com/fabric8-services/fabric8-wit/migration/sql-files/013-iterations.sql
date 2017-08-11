CREATE TABLE iterations (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    project_id uuid,
    parent_id uuid,
    start_at timestamp with time zone,
    end_at timestamp with time zone,
    name text
);

CREATE INDEX ix_project_id ON iterations USING btree (project_id);