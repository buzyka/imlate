CREATE TABLE IF NOT EXISTS visitor_key (
    key_id VARCHAR(255) PRIMARY KEY,
    visitor_id INT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,   
    CONSTRAINT `fk.visitor_key.visitor_id`  FOREIGN KEY (visitor_id) REFERENCES visitors(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;