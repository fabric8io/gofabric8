CREATE TYPE iteration_state AS ENUM ('new', 'start', 'close');
ALTER TABLE iterations ADD state iteration_state DEFAULT 'new';
