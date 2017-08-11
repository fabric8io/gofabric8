create index idx_user_full_name on users (lower(full_name));
create index idx_user_email on users (lower(email));
create index idx_idenities_username on identities (lower(username));