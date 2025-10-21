ALTER TABLE visitors
  DROP INDEX idx_visitors_updated_at,
  DROP INDEX idx_visitors_isams_id,
  DROP COLUMN updated_at,
  DROP COLUMN isams_id;
