ALTER TABLE visitors
    ADD COLUMN isams_id   INT       NULL DEFAULT NULL,
    ADD COLUMN isams_school_id   VARCHAR(255) NULL DEFAULT NULL,
    ADD COLUMN updated_at DATETIME  NULL DEFAULT NULL,
    ADD INDEX idx_visitors_isams_id (isams_id),
    ADD INDEX idx_visitors_isams_school_id (isams_school_id),
    ADD INDEX idx_visitors_updated_at (updated_at);
