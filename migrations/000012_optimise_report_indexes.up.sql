-- Composite index for the tracking hot path. Every track event runs
-- CountEventsByVisitorIdSince and RecalculateVisitorDay, both of which filter by
-- (visitor_id, created_at); track had no index on visitor_id at all.
ALTER TABLE track ADD INDEX idx_track_visitor_created (visitor_id, created_at);

-- Two redundant indexes go away, one useful one arrives, in a single statement
-- so the visitor_id foreign key is never left without an index to enforce it:
--   * idx_vdr_day is covered by PRIMARY KEY (day, visitor_id)'s leftmost prefix.
--   * `fk.visit_daily_report.visitor_id` is the index InnoDB generated for the
--     foreign key; idx_vdr_visitor_day leads with visitor_id and takes over that
--     job. InnoDB would drop it implicitly here, but dropping it explicitly keeps
--     the resulting schema the same no matter how the database got to this point.
--   * idx_vdr_visitor_day additionally lets the optimiser drive the report join
--     from visitors when filtering by visitor attributes.
-- The foreign key itself is untouched.
ALTER TABLE visit_daily_report
    ADD INDEX idx_vdr_visitor_day (visitor_id, day),
    DROP INDEX idx_vdr_day,
    DROP INDEX `fk.visit_daily_report.visitor_id`;
