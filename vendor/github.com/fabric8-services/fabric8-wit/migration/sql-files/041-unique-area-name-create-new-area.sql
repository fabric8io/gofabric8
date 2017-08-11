------ You can't allow the same area name and the same ancestry inside a space

ALTER TABLE areas ADD CONSTRAINT areas_name_space_id_path_unique UNIQUE(space_id,name,path);

------  For existing spaces in production, which dont have a default area, create one.
--
-- 1. Get all spaces which have an area under it with the same name.
-- 2. Get all spaces not in (1)
-- 3. insert an 'area' for all such spaces in (2)

INSERT INTO areas
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
                         INNER JOIN areas AS a
                                 ON s.name = a.name
                                    AND s.id = a.space_id);  


----- for all other existing areas in production, move them under the default 'root' area.



                           
CREATE OR REPLACE FUNCTION GetRootArea(s_id uuid,OUT root_id uuid) AS $$ BEGIN
-- Get Root area for a space
     select id from areas 
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



-- Migrate all existing areas into the new tree where the parent is always the root area

CREATE OR REPLACE FUNCTION GetUpdatedAreaPath(area_id uuid,space_id uuid, path ltree, OUT updated_path ltree) AS $rootarea$ 
-- Migrate all existing areas into the new tree where the parent is always the root area

     DECLARE 
          rootarea uuid;                                            
     BEGIN
     
     select GetRootArea(space_id) into rootarea;
     IF rootarea != area_id 
         THEN                  
         IF path=''
            THEN 
             select UUIDToLtreeNode(rootarea) into updated_path ;
         ELSE 
             select TextToLtreeNode(concat(rootarea::text,'.',path::text)) into updated_path;
         END IF;
     ELSE 
         updated_path:='';        
     END IF;
END; 
$rootarea$  LANGUAGE plpgsql ;   

-- Move all areas under that space into the root area ( except of course the root area ;) ), 

UPDATE AREAS set path=GetUpdatedAreaPath(id,space_id,path) ;

-- cleanup

DROP FUNCTION GetUpdatedAreaPath(uuid,uuid,ltree);
DROP FUNCTION GetRootArea(uuid);
DROP FUNCTION TextToLtreeNode(text);
