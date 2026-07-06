CREATE TABLE IF NOT EXISTS visit_daily_report (
    day DATE NOT NULL,
    visitor_id INT NOT NULL,
    login_time DATETIME NULL,
    logout_time DATETIME NULL,
    total_minutes_inside INT NULL,
    clear_minutes_inside INT NULL,
    visits_count INT NOT NULL DEFAULT 0,
    last_status ENUM('sign_in', 'sign_out') NULL,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    finalized_at DATETIME NULL,
    PRIMARY KEY (day, visitor_id),
    INDEX idx_vdr_day (day),
    INDEX idx_vdr_finalized (finalized_at, day),
    CONSTRAINT `fk.visit_daily_report.visitor_id` FOREIGN KEY (visitor_id)
        REFERENCES visitors(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
