-- users
INSERT INTO
   users(created_at, updated_at, id, email, full_name, image_url, bio, url, context_information)
VALUES
   (
      now(), now(), '01b291cd-9399-4f1a-8bbc-d1de66d76192', 'testone@example.com', 'test one', 'https://www.gravatar.com/avatar/testone', 'my test bio one', 'http://example.com', '{"key": "value"}'
   ),
   (
      now(), now(), '0d19928e-ef61-46fd-9bdc-71d1ecbce2c7', 'testtwo@example.com', 'test two', 'http://https://www.gravatar.com/avatar/testtwo', 'my test bio two', 'http://example.com', '{"key": "value"}'
   )
;
-- identities
INSERT INTO
   identities(created_at, updated_at, id, username, provider_type, user_id, profile_url)
VALUES
   (
      now(), now(), '01b291cd-9399-4f1a-8bbc-d1de66d76192', 'testone', 'github', '01b291cd-9399-4f1a-8bbc-d1de66d76192', 'http://example-github.com'
   ),
   (
      now(), now(), '5f946975-ff47-4c4a-b5dc-778f0b7e476c', 'testwo', 'rhhd', '0d19928e-ef61-46fd-9bdc-71d1ecbce2c7', 'http://example-rhd.com'
   )
;
-- spaces
INSERT INTO
   spaces (created_at, updated_at, id, version, name, description, owner_id)
VALUES
   (
      now(), now(), '86af5178-9b41-469b-9096-57e5155c3f31', 0, 'test.space.one', 'space desc one', '01b291cd-9399-4f1a-8bbc-d1de66d76192'
   )
;
-- work_item_types
INSERT INTO
   work_item_types(created_at, updated_at, id, name, version, fields, space_id)
VALUES
   (
      now(), now(), 'bbf35418-04b6-426c-a60b-7f80beb0b624', 'Test item type 1', 1.0, '{}', '2e0698d8-753e-4cef-bb7c-f027634824a2'
   )
;
INSERT INTO
   work_item_types(created_at, updated_at, id, name, version, path, fields, space_id)
VALUES
   (
      now(), now(), '86af5178-9b41-469b-9096-57e5155c3f31', 'Test item type 2', 1.0, 'bbf35418_04b6_426c_a60b_7f80beb0b624.86af5178_9b41_469b_9096_57e5155c3f31', '{}', '86af5178-9b41-469b-9096-57e5155c3f31'
   )
;
-- trackers
INSERT INTO
   trackers(created_at, updated_at, id, url, type)
VALUES
   (
      now(), now(), 1, 'http://example.com', 'github'
   ),
   (
      now(), now(), 2, 'http://example-jira.com', 'jira'
   )
;
-- tracker_queries id | query | schedule | tracker_id | space_id
INSERT INTO
   tracker_queries(created_at, updated_at, id, query, schedule, tracker_id, space_id)
VALUES
   (
      now(), now(), 1, 'SELECT * FROM', 'schedule', 1, '86af5178-9b41-469b-9096-57e5155c3f31'
   ),
   (
      now(), now(), 2, 'SELECT * FROM', 'schedule', 2, '86af5178-9b41-469b-9096-57e5155c3f31'
   )
;

-- space_resources
INSERT INTO
   space_resources(created_at, updated_at, id, space_id, resource_id, policy_id, permission_id)
VALUES
   (
      now(), now(), '2e0698d8-753e-4cef-bb7c-f027634824a2', '86af5178-9b41-469b-9096-57e5155c3f31', 'resource_id', 'policy_id', 'permission_id'
   ),
   (
      now(), now(), '71171e90-6d35-498f-a6a7-2083b5267c18', '86af5178-9b41-469b-9096-57e5155c3f31', 'resource_id', 'policy_id', 'permission_id'
   )
;
-- areas created_at | updated_at | deleted_at | id | space_id | version | path | name
INSERT INTO
   areas(created_at, updated_at, id, space_id, version, path, name)
VALUES
   (
      now(), now(), '2e0698d8-753e-4cef-bb7c-f027634824a2', '86af5178-9b41-469b-9096-57e5155c3f31', 0, 'path', 'area test one'
   ),
   (
      now(), now(), '71171e90-6d35-498f-a6a7-2083b5267c18', '86af5178-9b41-469b-9096-57e5155c3f31', 0, '', 'area test two'
   )
;
-- iterations
INSERT INTO
   iterations(created_at, updated_at, id, space_id, start_at, end_at, name, description, state)
VALUES
   (
      now(), now(), '71171e90-6d35-498f-a6a7-2083b5267c18', '86af5178-9b41-469b-9096-57e5155c3f31', now(), now(), 'iteration test one', 'description', 'new'
   ),
   (
      now(), now(), '2e0698d8-753e-4cef-bb7c-f027634824a2', '86af5178-9b41-469b-9096-57e5155c3f31', now(), now(), 'iteration test two', 'description', 'start'
   )
;
-- comments
INSERT INTO
   comments(created_at, updated_at, id, parent_id, body, created_by, markup)
VALUES
   (
      now(), now(), '71171e90-6d35-498f-a6a7-2083b5267c18', '2e0698d8-753e-4cef-bb7c-f027634824a2', 'body test one', '01b291cd-9399-4f1a-8bbc-d1de66d76192', 'PlainText'
   ),
   (
      now(), now(), '2e0698d8-753e-4cef-bb7c-f027634824a2', '2e0698d8-753e-4cef-bb7c-f027634824a2', 'body test two', '01b291cd-9399-4f1a-8bbc-d1de66d76192', 'PlainText'
   )
;
-- comment_revisions
INSERT INTO
   comment_revisions(id, revision_time, revision_type, modifier_id, comment_id, comment_body, comment_markup, comment_parent_id)
VALUES
   (
      '71171e90-6d35-498f-a6a7-2083b5267c18', now(), 1, '5f946975-ff47-4c4a-b5dc-778f0b7e476c', '71171e90-6d35-498f-a6a7-2083b5267c18', 'comment body test one', 'comment markup test one', '71171e90-6d35-498f-a6a7-2083b5267c18'
   ),
   (
      '2e0698d8-753e-4cef-bb7c-f027634824a2', now(), 1, '5f946975-ff47-4c4a-b5dc-778f0b7e476c', '71171e90-6d35-498f-a6a7-2083b5267c18', 'comment body test two', 'comment markup test two', '71171e90-6d35-498f-a6a7-2083b5267c18'
   )
;
-- work_item_link_categories
INSERT INTO
   work_item_link_categories(created_at, updated_at, id, version, name, description)
VALUES
   (
      now(), now(), '71171e90-6d35-498f-a6a7-2083b5267c18', 1, 'name test one', 'description one'
   ),
   (
      now(), now(), '2e0698d8-753e-4cef-bb7c-f027634824a2', 1, 'name test two', 'description two'
   )
;
-- work_item_link_types
INSERT INTO
   work_item_link_types(created_at, updated_at, id, version, name, description, forward_name, reverse_name, topology, link_category_id, space_id, source_type_id, target_type_id)
VALUES
   (
      now(), now(), '2e0698d8-753e-4cef-bb7c-f027634824a2', 1, 'test one', 'desc', 'forward test one', 'reverser test one', 'dependency', '71171e90-6d35-498f-a6a7-2083b5267c18', '2e0698d8-753e-4cef-bb7c-f027634824a2', '86af5178-9b41-469b-9096-57e5155c3f31', '86af5178-9b41-469b-9096-57e5155c3f31'
   ),
   (
      now(), now(), '71171e90-6d35-498f-a6a7-2083b5267c18', 1, 'test two', 'desc', 'forward test two', 'reverser test two', 'network', '2e0698d8-753e-4cef-bb7c-f027634824a2', '2e0698d8-753e-4cef-bb7c-f027634824a2', '86af5178-9b41-469b-9096-57e5155c3f31', '86af5178-9b41-469b-9096-57e5155c3f31'
   )
;
-- work_items
INSERT INTO
   work_items(created_at, updated_at, type, version, space_id, fields)
VALUES
   (
      now(), now(), 'bbf35418-04b6-426c-a60b-7f80beb0b624', 1.0, '86af5178-9b41-469b-9096-57e5155c3f31', '{}'
   ),
   (
      now(), now(), 'bbf35418-04b6-426c-a60b-7f80beb0b624', 2.0, '86af5178-9b41-469b-9096-57e5155c3f31', '{}'
   )
;
-- work_item_revisions
INSERT INTO
   work_item_revisions(id, revision_time, revision_type, modifier_id, work_item_id, work_item_type_id, work_item_version, work_item_fields)
VALUES
   (
      '2e0698d8-753e-4cef-bb7c-f027634824a2', now(), 1, '01b291cd-9399-4f1a-8bbc-d1de66d76192', 1, '2e0698d8-753e-4cef-bb7c-f027634824a2', 1, '{}'
   ),
   (
      '71171e90-6d35-498f-a6a7-2083b5267c18', now(), 1, '01b291cd-9399-4f1a-8bbc-d1de66d76192', 1, '2e0698d8-753e-4cef-bb7c-f027634824a2', 1, '{}'
   )
;
-- work_item_links
INSERT INTO
   work_item_links(created_at, updated_at, id, version, link_type_id)
VALUES
   (
      now(), now(), '2e0698d8-753e-4cef-bb7c-f027634824a2', 1, '2e0698d8-753e-4cef-bb7c-f027634824a2'
   ),
   (
      now(), now(), '71171e90-6d35-498f-a6a7-2083b5267c18', 1, '71171e90-6d35-498f-a6a7-2083b5267c18'
   )
;
-- work_item_link_revisions
INSERT INTO
   work_item_link_revisions(id, revision_time, revision_type, modifier_id, work_item_link_id, work_item_link_version, work_item_link_source_id, work_item_link_target_id, work_item_link_type_id)
VALUES
   (
      '71171e90-6d35-498f-a6a7-2083b5267c18', now(), 1, '01b291cd-9399-4f1a-8bbc-d1de66d76192', '71171e90-6d35-498f-a6a7-2083b5267c18', 1, 1, 2, '2e0698d8-753e-4cef-bb7c-f027634824a2'
   ),
   (
      '2e0698d8-753e-4cef-bb7c-f027634824a2', now(), 2, '01b291cd-9399-4f1a-8bbc-d1de66d76192', '71171e90-6d35-498f-a6a7-2083b5267c18', 1, 2, 1, '2e0698d8-753e-4cef-bb7c-f027634824a2'
   )
;
-- tracker_items
INSERT INTO
   tracker_items(created_at, updated_at, id, remote_item_id, item, batch_id, tracker_id)
VALUES
   (
      now(), now(), 1, 'remote_id', 'test one', 'batch_id', 1
   ),
   (
      now(), now(), 2, 'remote_id', 'test two', 'batch_id', 2
   )
;
