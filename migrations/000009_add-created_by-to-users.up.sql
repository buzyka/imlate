ALTER TABLE users
  ADD COLUMN created_by CHAR(36) DEFAULT NULL,
  ADD CONSTRAINT fk_users_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL;
