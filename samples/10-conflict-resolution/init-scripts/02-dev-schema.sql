-- Development Database (db2)
-- This represents the development state with changes

-- Users table - has conflicts with production
-- Row 1: email changed (conflict)
-- Row 2: status changed (conflict)
-- Row 3: same as prod
-- Row 4: new user (added)
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL UNIQUE,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO users (id, name, email, status) VALUES
(1, 'John Doe', 'john.d@newdomain.com', 'active'),
(2, 'Jane Smith', 'jane.smith@example.com', 'premium'),
(3, 'Bob Wilson', 'bob.wilson@example.com', 'inactive'),
(4, 'Alice Johnson', 'alice.johnson@example.com', 'active');

-- Logs table - dev has newer/different logs
-- Row 1: different details (conflict)
-- Row 2: same
-- Row 3: removed in dev
-- Row 4-5: new logs (added)
CREATE TABLE logs (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT,
    action VARCHAR(100) NOT NULL,
    details TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO logs (id, user_id, action, details) VALUES
(1, 1, 'login', 'User logged in from 10.0.0.1 (dev network)'),
(2, 2, 'login', 'User logged in from 192.168.1.2'),
(4, 4, 'signup', 'New user registration'),
(5, 1, 'logout', 'User logged out');

-- Config table - dev has different config values
-- Row 1: different value (conflict - prod should be kept)
-- Row 2: different value (conflict - prod should be kept)
-- Row 3: same
-- Row 4: different value (conflict - prod should be kept)
-- Row 5: new config (added)
CREATE TABLE config (
    id INT AUTO_INCREMENT PRIMARY KEY,
    config_key VARCHAR(100) NOT NULL UNIQUE,
    config_value TEXT NOT NULL,
    description VARCHAR(255),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

INSERT INTO config (id, config_key, config_value, description) VALUES
(1, 'app_name', 'MyApp Development', 'Application name'),
(2, 'max_login_attempts', '10', 'Maximum failed login attempts'),
(3, 'session_timeout', '3600', 'Session timeout in seconds'),
(4, 'maintenance_mode', 'true', 'Enable maintenance mode'),
(5, 'debug_mode', 'true', 'Enable debug mode');

-- Settings table - some conflicts, uses default strategy (manual)
CREATE TABLE settings (
    id INT AUTO_INCREMENT PRIMARY KEY,
    setting_key VARCHAR(100) NOT NULL UNIQUE,
    setting_value TEXT NOT NULL,
    user_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO settings (id, setting_key, setting_value, user_id) VALUES
(1, 'theme', 'light', 1),
(2, 'notifications', 'disabled', 1),
(3, 'theme', 'light', 2),
(4, 'language', 'en', 4);
