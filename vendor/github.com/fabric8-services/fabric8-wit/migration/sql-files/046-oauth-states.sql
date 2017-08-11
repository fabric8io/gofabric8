-- Create Oauth state reference table for states used in oauth workflow
CREATE TABLE oauth_state_references (
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    id uuid primary key DEFAULT uuid_generate_v4() NOT NULL,
    referrer text NOT NULL
);