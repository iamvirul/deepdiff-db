-- Production Database (db1)
-- This represents the current production state

-- Users table - critical data requiring manual review for conflicts
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL UNIQUE,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO users (id, name, email, status) VALUES
(1, 'John Doe', 'john.doe@example.com', 'active'),
(2, 'Jane Smith', 'jane.smith@example.com', 'active'),
(3, 'Bob Wilson', 'bob.wilson@example.com', 'inactive');

-- Logs table - ephemeral data, dev version is preferred
CREATE TABLE logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT,
    action VARCHAR(100) NOT NULL,
    details TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO logs (id, user_id, action, details) VALUES
(1, 1, 'login', 'User logged in from 192.168.1.1'),
(2, 2, 'login', 'User logged in from 192.168.1.2'),
(3, 1, 'update_profile', 'Changed email address');

-- Config table - production config should be preserved
CREATE TABLE config (
    id INT AUTO_INCREMENT PRIMARY KEY,
    config_key VARCHAR(100) NOT NULL UNIQUE,
    config_value TEXT NOT NULL,
    description VARCHAR(255),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

INSERT INTO config (id, config_key, config_value, description) VALUES
(1, 'app_name', 'MyApp Production', 'Application name'),
(2, 'max_login_attempts', '3', 'Maximum failed login attempts'),
(3, 'session_timeout', '3600', 'Session timeout in seconds'),
(4, 'maintenance_mode', 'false', 'Enable maintenance mode');

-- Settings table - no overrides, uses default strategy
CREATE TABLE settings (
    id INT AUTO_INCREMENT PRIMARY KEY,
    setting_key VARCHAR(100) NOT NULL UNIQUE,
    setting_value TEXT NOT NULL,
    user_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO settings (id, setting_key, setting_value, user_id) VALUES
(1, 'theme', 'dark', 1),
(2, 'notifications', 'enabled', 1),
(3, 'theme', 'light', 2);
