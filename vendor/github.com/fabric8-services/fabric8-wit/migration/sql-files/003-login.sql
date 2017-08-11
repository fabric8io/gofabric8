CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- login

CREATE TABLE identities (
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    full_name text,
    image_url text
);


-- user

CREATE TABLE users (
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    email text,
    identity_id uuid
);

CREATE UNIQUE INDEX uix_users_email ON users USING btree (email);