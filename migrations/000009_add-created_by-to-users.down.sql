ALTER TABLE users
  DROP FOREIGN KEY fk_users_created_by,
  DROP COLUMN created_by;
