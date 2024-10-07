CREATE TABLE IF NOT EXISTS track (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    visitor_id INT NOT NULL,
    key_id VARCHAR(255) NOT NULL,
    sign_in BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_createdAt (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;