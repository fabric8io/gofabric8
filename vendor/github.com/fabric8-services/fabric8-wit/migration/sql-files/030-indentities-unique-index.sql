-- replace the unique index on `profile_url` with a check on 'DELETED_AT' to support soft deletes.
DROP INDEX uix_identity_profileurl;
CREATE UNIQUE INDEX uix_identity_profileurl ON identities USING btree (profile_url) WHERE deleted_at IS NULL;
