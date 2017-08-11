-- Modify space_id column to be a foreign key to spaces ID table
ALTER TABLE iterations ADD CONSTRAINT iterations_space_id_spaces_id_fk FOREIGN KEY (space_id) REFERENCES spaces (id) ON DELETE CASCADE;
ALTER TABLE areas ADD CONSTRAINT areas_space_id_spaces_id_fk FOREIGN KEY (space_id) REFERENCES spaces (id) ON DELETE CASCADE;
