-- This table sets the foundation for all future database migrations.
create table version (
  id bigserial primary key,
  updated_at timestamp with time zone not null default current_timestamp,
  version int not null unique CONSTRAINT positive_version CHECK (version >= 0)
);