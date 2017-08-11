-- need some work items to migrate the comment_revisions table, too
insert into spaces (id, name) values ('00000067-0000-0000-0000-000000000000', 'test space 67');
insert into work_item_types (id, name, space_id) values ('00000067-0000-0000-0000-000000000000', 'test type 1', '00000067-0000-0000-0000-000000000000');
insert into work_items (id, number, type, space_id) values ('00000067-0000-0000-0000-000000000000', 1, '00000067-0000-0000-0000-000000000000', '00000067-0000-0000-0000-000000000000');
insert into comments (id, parent_id, body) values ('00000067-0000-0000-0000-000000000000', '00000067-0000-0000-0000-000000000000', 'a foo comment');
insert into comment_revisions (id, revision_type, modifier_id, comment_id, comment_parent_id, comment_body) 
    values ('00000067-0000-0000-0000-000000000000', 1, 'cafebabe-0000-0000-0000-000000000000', '00000067-0000-0000-0000-000000000000',  1, 'a foo comment');