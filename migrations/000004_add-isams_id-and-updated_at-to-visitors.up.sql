ALTER TABLE visitors
  ADD COLUMN isams_id   INT       NULL DEFAULT NULL,
  ADD COLUMN updated_at DATETIME  NULL DEFAULT NULL,
  ADD INDEX idx_visitors_isams_id (isams_id),
  ADD INDEX idx_visitors_updated_at (updated_at);
