-- drop existing unique index
DROP INDEX idx_idenities_username;
-- recreate unique index idx_identities_username case insensitive username
CREATE INDEX idx_identities_username ON identities (username);
