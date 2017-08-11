-- default is 'false', works with business logic as well.
ALTER TABLE identities ADD COLUMN registration_completed BOOLEAN NOT NULL DEFAULT FALSE;
