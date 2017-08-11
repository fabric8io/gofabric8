-- oauth_state_references
INSERT INTO
   oauth_state_references(created_at, updated_at, id, referrer)
VALUES
   (
      now(), now(), '2e0698d8-753e-4cef-bb7c-f027634824a2', 'test referrer one text'
   )
;
INSERT INTO
   oauth_state_references(created_at, updated_at, id, referrer)
VALUES
   (
      now(), now(), '71171e90-6d35-498f-a6a7-2083b5267c18', 'test referrer two text'
   )
;
