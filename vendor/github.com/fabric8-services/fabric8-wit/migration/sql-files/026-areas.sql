CREATE EXTENSION IF NOT EXISTS "ltree";

CREATE TABLE areas (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    space_id uuid,
    version integer DEFAULT 0 NOT NULL,
    path ltree,
    name text
);