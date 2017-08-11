-- create spaces 1 and 2
insert into spaces (id, name) values ('11111111-0000-0000-0000-000000000000', 'test space 1');
insert into spaces (id, name) values ('22222222-0000-0000-0000-000000000000', 'test space 2');
-- create work item types for spaces 1 and 2
insert into work_item_types (id, name, space_id) values ('11111111-0000-0000-0000-000000000000', 'test type 1', '11111111-0000-0000-0000-000000000000');
insert into work_item_types (id, name, space_id) values ('22222222-0000-0000-0000-000000000000', 'test type 2', '22222222-0000-0000-0000-000000000000');
-- create work item link types for spaces 1 and 2
insert into work_item_link_types (id, name, topology, forward_name, reverse_name, space_id) 
    values ('11111111-0000-0000-0000-000000000000', 'foo', 'dependency', 'foo', 'foo', '11111111-0000-0000-0000-000000000000');
insert into work_item_link_types (id, name, topology, forward_name, reverse_name, space_id) 
    values ('22222222-0000-0000-0000-000000000000', 'bar', 'dependency', 'bar', 'bar', '22222222-0000-0000-0000-000000000000');
-- create identity (for revisions)
insert into identities (id, username) values ('cafebabe-0000-0000-0000-000000000000', 'foo');
-- inserting work items, their revisions and comments in space '1'
insert into work_items (id, type, space_id) values (12347, '11111111-0000-0000-0000-000000000000', '11111111-0000-0000-0000-000000000000');
insert into work_item_revisions (revision_type, modifier_id, work_item_id) values (1, 'cafebabe-0000-0000-0000-000000000000', 12347);
insert into work_item_revisions (revision_type, modifier_id, work_item_id) values (2, 'cafebabe-0000-0000-0000-000000000000', 12347);
insert into comments (parent_id, body) values ('12347', 'blabla');
insert into work_items (id, type, space_id) values (12348, '11111111-0000-0000-0000-000000000000', '11111111-0000-0000-0000-000000000000');
insert into work_item_revisions (revision_type, modifier_id, work_item_id) values (1, 'cafebabe-0000-0000-0000-000000000000', 12348);
insert into work_item_revisions (revision_type, modifier_id, work_item_id) values (2, 'cafebabe-0000-0000-0000-000000000000', 12348);
insert into comments (parent_id, body) values ('12348', 'blabla');
-- inserting work items, their revisions and comments in space '2'
insert into work_items (id, type, space_id) values (12349, '22222222-0000-0000-0000-000000000000', '22222222-0000-0000-0000-000000000000');
insert into work_item_revisions (revision_type, modifier_id, work_item_id) values (1, 'cafebabe-0000-0000-0000-000000000000', 12349);
insert into work_item_revisions (revision_type, modifier_id, work_item_id) values (2, 'cafebabe-0000-0000-0000-000000000000', 12349);
insert into comments (parent_id, body) values ('12349', 'blabla');
insert into work_items (id, type, space_id) values (12350, '22222222-0000-0000-0000-000000000000', '22222222-0000-0000-0000-000000000000');
insert into work_item_revisions (revision_type, modifier_id, work_item_id) values (1, 'cafebabe-0000-0000-0000-000000000000', 12350);
insert into work_item_revisions (revision_type, modifier_id, work_item_id) values (2, 'cafebabe-0000-0000-0000-000000000000', 12350);
insert into comments (parent_id, body) values ('12350', 'blabla');
-- insert links between work items
insert into work_item_links (id, link_type_id, source_id, target_id) values ('11111111-0000-0000-0000-000000000000', '11111111-0000-0000-0000-000000000000', 12347, 12348);
insert into work_item_link_revisions (revision_type, modifier_id, work_item_link_id, work_item_link_version, work_item_link_source_id, work_item_link_target_id, work_item_link_type_id)
  values (1, 'cafebabe-0000-0000-0000-000000000000', '11111111-0000-0000-0000-000000000000',0,12347,12348,'11111111-0000-0000-0000-000000000000');
insert into work_item_links (id, link_type_id, source_id, target_id) values ('22222222-0000-0000-0000-000000000000', '22222222-0000-0000-0000-000000000000', 12349, 12350);
insert into work_item_link_revisions (revision_type, modifier_id, work_item_link_id, work_item_link_version, work_item_link_source_id, work_item_link_target_id, work_item_link_type_id)
  values (1, 'cafebabe-0000-0000-0000-000000000000', '22222222-0000-0000-0000-000000000000',0,12349,12350,'22222222-0000-0000-0000-000000000000');
