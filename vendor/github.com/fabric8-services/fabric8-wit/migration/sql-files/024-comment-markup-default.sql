-- add a 'markup' column in the 'comments' table
update comments set markup = 'PlainText' where markup = NULL; 