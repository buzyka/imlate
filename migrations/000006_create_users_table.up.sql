CREATE TABLE IF NOT EXISTS users (
    id CHAR(36) PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    surname VARCHAR(255) NOT NULL,
    role ENUM('admin', 'terminal') NOT NULL DEFAULT 'terminal',
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,
    UNIQUE INDEX idx_users_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO users (id, username, password, name, surname, role, is_active) VALUES
('a570ea85-aa2b-4d9f-b371-d8d2bb25d3fc', 'admin', '$2a$10$0HzIl5VVkiGBtqxs38MkxuZLInFsCRJ3gm9rSFDhXU9YdLnkYRrAi', 'Admin', 'User', 'admin', 1);

INSERT INTO users (id, username, password, name, surname, role, is_active) VALUES
('b681fb95-bb3c-5e0a-c482-e9e3cc36e4gd', 'terminal', '$2a$10$Fb5Sa3ks.NYHJ7ojUew2/uko7Z3QUF2V11cL4OQa3v6oixh1KO9YW', 'Terminal', 'User', 'terminal', 1);
