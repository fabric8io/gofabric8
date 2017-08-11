--- added a duplicate space with the same owner and name than a previous one
INSERT INTO
   spaces (created_at, updated_at, id, version, name, description, owner_id)
VALUES
   (
      now(), now(), '86af5178-9b41-469b-9096-57e5155c3f32', 0, 'test.Space.one', 'Space desc one', '01b291cd-9399-4f1a-8bbc-d1de66d76192'
   )
;
