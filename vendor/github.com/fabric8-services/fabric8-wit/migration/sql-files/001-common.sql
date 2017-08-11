-- tracker_items

CREATE TABLE tracker_items (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id bigserial primary key,
    remote_item_id text,
    item text,
    batch_id text,
    tracker_query_id bigint
);

-- trackers

CREATE TABLE trackers (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id bigserial primary key,
    url text,
    type text
);

-- tracker_queries

CREATE TABLE tracker_queries (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id bigserial primary key,
    query text,
    schedule text,
    tracker_id bigint
);

ALTER TABLE ONLY tracker_queries
    ADD CONSTRAINT tracker_queries_tracker_id_trackers_id_foreign 
        FOREIGN KEY (tracker_id)
        REFERENCES trackers(id)
        ON UPDATE RESTRICT 
        ON DELETE RESTRICT;

-- work_item_types

CREATE TABLE work_item_types (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text primary key,
    version integer,
    parent_path text,
    fields jsonb
);

-- work_items

CREATE TABLE work_items (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id bigserial primary key,
    type text,
    version integer,
    fields jsonb
);

