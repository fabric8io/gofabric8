insert into spaces (id, name) values ('11111111-2222-bbbb-0000-000000000000', 'test');
insert into iterations (id, name, path, space_id) values ('11111111-3333-bbbb-0000-000000000000', 'test area', '', '11111111-2222-bbbb-0000-000000000000');
insert into work_item_types (id, name, space_id) values ('11111111-4444-bbbb-0000-000000000000', 'Test WIT','11111111-2222-bbbb-0000-000000000000');
insert into work_items (id, space_id, type, fields) values (12346, '11111111-2222-bbbb-0000-000000000000', '11111111-4444-bbbb-0000-000000000000', '{"system.title":"Title"}'::json);