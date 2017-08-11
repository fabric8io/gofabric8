ALTER TABLE work_item_types RENAME COLUMN "parent_path" TO "path";
UPDATE work_item_types set path=path || (case when path != '/' then '/'  else '' end) || name;