ALTER TABLE visit_daily_report
    ADD INDEX idx_vdr_day (day),
    ADD INDEX `fk.visit_daily_report.visitor_id` (visitor_id),
    DROP INDEX idx_vdr_visitor_day;

ALTER TABLE track DROP INDEX idx_track_visitor_created;
