-- migrate work items description by replacing 'plain' with 'PlainText' in the 'markup' element of 'system.description' 
update work_items set fields=jsonb_set(fields, '{system.description, markup}', 
  to_jsonb('PlainText'::text)) where fields->>'system.description' is not null;

