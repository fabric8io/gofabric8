--
-- comments
--
insert into spaces (id, name) values ('11111111-6262-0000-0000-000000000000', 'test');
insert into work_item_types (id, name, space_id) values ('11111111-6262-0000-0000-000000000000', 'Test WIT','11111111-6262-0000-0000-000000000000');
insert into work_items (id, space_id, type, fields) values (62001, '11111111-6262-0000-0000-000000000000', '11111111-6262-0000-0000-000000000000', '{"system.title":"Work item 1"}'::json);
insert into work_items (id, space_id, type, fields) values (62002, '11111111-6262-0000-0000-000000000000', '11111111-6262-0000-0000-000000000000', '{"system.title":"Work item 2"}'::json);
insert into work_items (id, space_id, type, fields) values (62003, '11111111-6262-0000-0000-000000000000', '11111111-6262-0000-0000-000000000000', '{"system.title":"Work item 3"}'::json);
-- remove previous comments
delete from comments;
-- add comments linked to work items above
insert into comments (id, parent_id, body, created_at) values ( '11111111-6262-0001-0000-000000000000', '62001', 'a comment', '2017-06-13 09:00:00.0000+00');
insert into comments (id, parent_id, body, created_at) values ( '11111111-6262-0003-0000-000000000000', '62003', 'a comment', '2017-06-13 11:00:00.0000+00');
update comments set deleted_at = '2017-06-13 11:15:00.0000+00' where id =  '11111111-6262-0003-0000-000000000000';

--
-- work item links
--
insert into work_items (id, space_id, type, fields) values (62004, '11111111-6262-0000-0000-000000000000', '11111111-6262-0000-0000-000000000000', '{"system.title":"Work item 3"}'::json);
insert into work_items (id, space_id, type, fields) values (62005, '11111111-6262-0000-0000-000000000000', '11111111-6262-0000-0000-000000000000', '{"system.title":"Work item 3"}'::json);
insert into work_items (id, space_id, type, fields) values (62006, '11111111-6262-0000-0000-000000000000', '11111111-6262-0000-0000-000000000000', '{"system.title":"Work item 3"}'::json);
insert into work_items (id, space_id, type, fields) values (62007, '11111111-6262-0000-0000-000000000000', '11111111-6262-0000-0000-000000000000', '{"system.title":"Work item 3"}'::json);
insert into work_items (id, space_id, type, fields) values (62008, '11111111-6262-0000-0000-000000000000', '11111111-6262-0000-0000-000000000000', '{"system.title":"Work item 3"}'::json);
insert into work_items (id, space_id, type, fields) values (62009, '11111111-6262-0000-0000-000000000000', '11111111-6262-0000-0000-000000000000', '{"system.title":"Work item 3"}'::json);
delete from work_item_links;
insert into work_item_links (id, version, source_id, target_id, created_at) values ('11111111-6262-0001-0000-000000000000', 1, 62004, 62005, '2017-06-13 09:00:00.0000+00');
insert into work_item_links (id, version, source_id, target_id, deleted_at) values ('11111111-6262-0003-0000-000000000000', 1, 62008, 62009, '2017-06-13 11:00:00.0000+00');
update work_item_links set deleted_at = '2017-06-13 11:15:00.0000+00' where id = '11111111-6262-0003-0000-000000000000';


