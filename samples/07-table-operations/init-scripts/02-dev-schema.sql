-- Development database schema
-- Contains new tables and removes legacy tables

CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE orders (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    total DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- New table: Modern audit system (exists in dev but not in prod)
CREATE TABLE audit_events (
    id INT PRIMARY KEY AUTO_INCREMENT,
    event_type VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id INT NOT NULL,
    user_id INT,
    payload JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_audit_entity (entity_type, entity_id),
    INDEX idx_audit_user (user_id),
    INDEX idx_audit_created (created_at)
);

-- New table: JWT-based sessions (exists in dev but not in prod)
CREATE TABLE user_sessions (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    refresh_token VARCHAR(500) NOT NULL,
    user_agent VARCHAR(255),
    ip_address VARCHAR(45),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_sessions_user (user_id),
    INDEX idx_sessions_token (refresh_token(255))
);

-- New table: Feature flags (exists in dev but not in prod)
CREATE TABLE feature_flags (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    enabled BOOLEAN DEFAULT FALSE,
    rollout_percentage INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Insert sample data
INSERT INTO users (username, email) VALUES
    ('alice', 'alice@example.com'),
    ('bob', 'bob@example.com');

INSERT INTO orders (user_id, total, status) VALUES
    (1, 99.99, 'completed'),
    (2, 149.99, 'pending');

INSERT INTO audit_events (event_type, entity_type, entity_id, user_id, payload) VALUES
    ('created', 'order', 1, 1, '{"total": 99.99}'),
    ('updated', 'order', 2, 2, '{"status": "pending"}');

INSERT INTO user_sessions (user_id, refresh_token, user_agent, ip_address, expires_at) VALUES
    (1, 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...', 'Mozilla/5.0', '192.168.1.1', '2026-01-01 00:00:00'),
    (2, 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...', 'Chrome/120', '192.168.1.2', '2026-01-01 00:00:00');

INSERT INTO feature_flags (name, description, enabled, rollout_percentage) VALUES
    ('dark_mode', 'Enable dark mode UI', true, 100),
    ('new_checkout', 'New checkout flow', false, 25);
