CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE tenants (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid primary key NOT NULL,
    email text
);

CREATE TABLE namespaces (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid primary key NOT NULL,
	tenant_id uuid,
    name text,
	master_url text,
	type text,
	version text,
	state text
);

CREATE INDEX uix_namespaces_tenant ON namespaces USING btree (tenant_id);