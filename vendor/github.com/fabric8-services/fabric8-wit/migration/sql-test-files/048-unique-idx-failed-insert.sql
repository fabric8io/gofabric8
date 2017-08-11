-- insert two iterations one will fail due to invalid iterations_name_space_id_path_unique
INSERT INTO
   iterations(created_at, updated_at, id, space_id, start_at, end_at, name, description, state, path)
VALUES
   (
      now(), now(), '86af5178-9b41-469b-9096-57e5155c3f31', '86af5178-9b41-469b-9096-57e5155c3f31', now(), now(), 'iteration test one', 'description', 'new', '/'
   )
;

INSERT INTO
   iterations(created_at, updated_at, id, space_id, start_at, end_at, name, description, state, path)
VALUES
   (
      now(), now(), '0a24d3c2-e0a6-4686-8051-ec0ea1915a28', '86af5178-9b41-469b-9096-57e5155c3f31', now(), now(), 'iteration test one', 'description', 'new', '/'
   )
;
