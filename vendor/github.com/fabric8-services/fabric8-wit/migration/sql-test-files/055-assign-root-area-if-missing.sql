insert into spaces (id, name) values ('11111111-2222-0000-0000-000000000000', 'test');
insert into areas (id, name, path, space_id) values ('11111111-3333-0000-0000-000000000000', 'test area', '', '11111111-2222-0000-0000-000000000000');
insert into work_item_types (id, name, space_id) values ('11111111-4444-0000-0000-000000000000', 'Test WIT','11111111-2222-0000-0000-000000000000');
insert into work_items (id, space_id, type, fields) values (12345, '11111111-2222-0000-0000-000000000000', '11111111-4444-0000-0000-000000000000', '{"system.title":"Title"}'::json);