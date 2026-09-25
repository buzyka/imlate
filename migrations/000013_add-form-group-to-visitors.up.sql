ALTER TABLE visitors
    ADD COLUMN form_group VARCHAR(32) NULL DEFAULT NULL AFTER year_group,
    ADD INDEX idx_visitors_form_group (form_group);
