UPDATE track SET key_id = '' WHERE key_id IS NULL;

ALTER TABLE track
  MODIFY COLUMN key_id VARCHAR(255) NOT NULL,
  DROP COLUMN admin_id,
  DROP COLUMN description;
