-- Refactor Identities and Users tables.
ALTER TABLE identities
    DROP full_name,
    DROP image_url,
    ADD username text,
    ADD provider text,
    ADD user_id uuid;

ALTER TABLE users
    DROP identity_id,
    ADD full_name text,
    ADD image_url text,
    ADD bio text,
    ADD url text;