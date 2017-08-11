-- prepare data
insert into spaces (id, name) values ('00000066-0000-0000-0000-000000000000', 'test space 1');
insert into work_item_types (id, name, space_id) values ('00000066-0000-0000-0000-000000000000', 'test type 1', '00000066-0000-0000-0000-000000000000');
insert into work_item_link_types (id, name, topology, forward_name, reverse_name, space_id) 
    values ('00000066-0000-0000-0000-000000000000', 'foo', 'dependency', 'foo', 'foo', '00000066-0000-0000-0000-000000000000');
insert into work_items (id, type, space_id) values ('00000066-0000-0000-0000-000000000001', '00000066-0000-0000-0000-000000000000', '00000066-0000-0000-0000-000000000000');
insert into work_items (id, type, space_id) values ('00000066-0000-0000-0000-000000000002', '00000066-0000-0000-0000-000000000000', '00000066-0000-0000-0000-000000000000');
-- insert valid and invalid links
insert into work_item_links (id, link_type_id, source_id, target_id) values ('00000066-0000-0000-0000-000000000001', '00000066-0000-0000-0000-000000000000', '00000066-0000-0000-0000-000000000001', '00000066-0000-0000-0000-000000000002');
insert into work_item_links (id, link_type_id, source_id, target_id) values ('00000066-0000-0000-0000-000000000002', NULL, '00000066-0000-0000-0000-000000000001', '00000066-0000-0000-0000-000000000002');
insert into work_item_links (id, link_type_id, source_id, target_id) values ('00000066-0000-0000-0000-000000000003', '00000066-0000-0000-0000-000000000000', NULL, '00000066-0000-0000-0000-000000000002');
insert into work_item_links (id, link_type_id, source_id, target_id) values ('00000066-0000-0000-0000-000000000004', '00000066-0000-0000-0000-000000000000', '00000066-0000-0000-0000-000000000001', NULL);
