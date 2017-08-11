update work_items set fields=jsonb_set(fields, '{system.area}', to_jsonb(subq.id::text)) 
    from (select id, space_id from areas where path = '') AS subq
    where subq.space_id = work_items.space_id and fields->>'system.area' IS NULL;