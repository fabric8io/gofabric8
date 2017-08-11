CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- work item link categories

CREATE TABLE projects (
    created_at  timestamp with time zone,
    updated_at  timestamp with time zone,
    deleted_at  timestamp with time zone DEFAULT NULL,

    id          uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    version     integer DEFAULT 0 NOT NULL,

    name        text NOT NULL CHECK(name <> '')
);
CREATE UNIQUE INDEX projects_name_idx ON projects (name) WHERE deleted_at IS NULL;