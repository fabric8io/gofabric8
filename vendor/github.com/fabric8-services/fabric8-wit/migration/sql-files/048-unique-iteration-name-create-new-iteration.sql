-- Update iterations having same name, append UUID to make it unique.
UPDATE iterations
SET name = name || '-' || uuid_generate_v4()
WHERE id IN
    (SELECT id
     FROM iterations
     WHERE name IN
         (SELECT name
          FROM iterations
          GROUP BY name
          HAVING count(name) >1));

------  For existing spaces in production, which dont have a root iteration, create one.
--
-- 1. Get all spaces which have an iteration under it with the same name.
-- 2. Get all spaces not in (1)
-- 3. insert an 'iteration' for all such spaces in (2)
INSERT INTO iterations
            (created_at,
             updated_at,
             name,
             space_id)
SELECT current_timestamp,
       current_timestamp,
       name,
       id
FROM   spaces
WHERE  id NOT IN (SELECT s.id
                  FROM   spaces AS s
                         INNER JOIN iterations AS i
                                 ON s.name = i.name
                                    AND s.id = i.space_id);

----- for all other existing iterations in production, move them under the root iteration of given space.
CREATE OR REPLACE FUNCTION GetRootIteration(s_id uuid,OUT root_id uuid) AS $$ BEGIN
-- Get Root iteration for a space
     select id from iterations 
          where name in ( SELECT name as space_name
          from spaces 
              where id=s_id )
                  and space_id =s_id 
                           into root_id;
END; $$ LANGUAGE plpgsql ;

-- Convert Text to Ltree , use standard library FUNCTION?

CREATE OR REPLACE FUNCTION TextToLtreeNode(u text, OUT node ltree) AS $$ BEGIN
    SELECT replace(u, '-', '_') INTO node;
END; $$ LANGUAGE plpgsql;


-- Migrate all existing iterations into the new tree where the parent is always the root iteration

CREATE OR REPLACE FUNCTION GetUpdatedIterationPath(iteration_id uuid,space_id uuid, path ltree, OUT updated_path ltree) AS $rootiteration$ 
-- Migrate all existing iterations into the new tree where the parent is always the root iteration

     DECLARE
          rootiteration uuid;
     BEGIN
     -- In production this probably not NULL; safety check.
     If path IS NULL
        THEN
            path = '';
     END IF;
     select GetRootIteration(space_id) into rootiteration;
     IF rootiteration != iteration_id 
         THEN                  
         IF path=''
            THEN 
             select UUIDToLtreeNode(rootiteration) into updated_path ;
         ELSE 
             select TextToLtreeNode(concat(rootiteration::text,'.',path::text)) into updated_path;
         END IF;
     ELSE 
         updated_path:='';        
     END IF;
END;
$rootiteration$  LANGUAGE plpgsql ;

-- Move all iterations under it's space and into the root iterations ( except of course the root iteration ;) ), 

UPDATE iterations set path=GetUpdatedIterationPath(id,space_id,path);

update work_items set fields=jsonb_set(fields, '{system.iteration}', to_jsonb(subq.id::text)) 
    from (select id, space_id from iterations where path = '') AS subq
    where subq.space_id = work_items.space_id and fields->>'system.iteration' IS NULL;

-- cleanup
DROP FUNCTION GetUpdatedIterationPath(uuid,uuid,ltree);
DROP FUNCTION GetRootIteration(uuid);
DROP FUNCTION TextToLtreeNode(text);

CREATE INDEX ix_name ON iterations USING btree (name);

------ You can't allow the same iteration name and the same ancestry inside a space
ALTER TABLE iterations ADD CONSTRAINT iterations_name_space_id_path_unique UNIQUE(space_id,name,path);
