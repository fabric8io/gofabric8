-- users
INSERT INTO
   users(created_at, updated_at, id, email, full_name, image_url, bio, url, context_information)
VALUES
   (
      now(), now(), 'f03f023b-0427-4cdb-924b-fb2369018ab7', 'test2@example.com', 'test1', 'https://www.gravatar.com/avatar/testtwo2', 'my test bio one', 'http://example.com/001', '{"key": "value"}'
   ),
   (
      now(), now(), 'f03f023b-0427-4cdb-924b-fb2369018ab6', 'test3@example.com', 'test2', 'http://https://www.gravatar.com/avatar/testtwo3', 'my test bio two', 'http://example.com/002', '{"key": "value"}'
   )
;
-- identities
INSERT INTO
   identities(created_at, updated_at, id, username, provider_type, user_id, profile_url)
VALUES
   (
      now(), now(), '2a808366-9525-4646-9c80-ed704b2eebbe', 'test1', 'github', 'f03f023b-0427-4cdb-924b-fb2369018ab7', 'http://example-github.com/001'
   ),
   (
      now(), now(), '2a808366-9525-4646-9c80-ed704b2eebbb', 'test2', 'rhhd', 'f03f023b-0427-4cdb-924b-fb2369018ab6', 'http://example-rhd.com/002'
   )
;
