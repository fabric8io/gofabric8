CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE work_item_link_type_combinations (
    created_at      timestamp with time zone,
    updated_at      timestamp with time zone,
    deleted_at      timestamp with time zone,
    id uuid         primary key DEFAULT uuid_generate_v4() NOT NULL,
    version         integer,
    link_type_id    uuid NOT NULL REFERENCES work_item_link_types(id) ON DELETE CASCADE,
    source_type_id  uuid NOT NULL REFERENCES work_item_types(id) ON DELETE CASCADE,
    target_type_id  uuid NOT NULL REFERENCES work_item_types(id) ON DELETE CASCADE,
    -- We need the space id here because different space templates might specify
    -- the same source/target type combination for the same system-defined link
    -- type (e.g. "parent of"). That would violated our unique constraint below
    -- if the space_id was missing from it.
    -- TODO(kwk): once we have space templates, this will become a reference to
    -- the space template and not to the space.
    space_id        uuid  NOT NULL REFERENCES spaces(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX work_item_link_type_combinations_uniq
    ON work_item_link_type_combinations (
        space_id,
        link_type_id,
        source_type_id,
        target_type_id
    )
    WHERE deleted_at IS NULL;
