-- Refactor Identities: rename column 'provider' to 'provider_type'
ALTER TABLE identities RENAME COLUMN provider to provider_type;
-- the add a column to store the URL of the profile on the remote workitem system.
ALTER TABLE identities ADD profile_url text;
-- index to query identity by profile_url, which must be unique 
CREATE UNIQUE INDEX uix_identity_profileurl ON identities USING btree (profile_url);
-- index to query identity by user_id
CREATE INDEX uix_identity_userid ON identities USING btree (user_id);