-- Create test database
CREATE DATABASE IF NOT EXISTS tracker_test;

-- Grant privileges to the tracker user on test database
GRANT ALL PRIVILEGES ON tracker_test.* TO 'trackme'@'%';

FLUSH PRIVILEGES;
