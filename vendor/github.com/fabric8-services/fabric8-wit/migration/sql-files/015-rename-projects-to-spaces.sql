ALTER TABLE projects RENAME TO spaces;
ALTER INDEX projects_name_idx RENAME TO spaces_name_idx;
ALTER INDEX projects_pkey RENAME TO spaces_pkey;
ALTER TABLE spaces RENAME CONSTRAINT projects_name_check TO spaces_name_check;
ALTER TABLE iterations RENAME COLUMN project_id to space_id;