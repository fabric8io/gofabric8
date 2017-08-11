-- Refactor Identities: add a foreign key constraint
alter table identities add constraint identities_user_id_users_id_fk foreign key (user_id) REFERENCES users (id);
