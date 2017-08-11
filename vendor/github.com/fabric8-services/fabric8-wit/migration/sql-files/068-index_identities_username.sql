-- add index on identities.username
drop index idx_identities_username;
create index idx_identities_username on identities (lower(username));

