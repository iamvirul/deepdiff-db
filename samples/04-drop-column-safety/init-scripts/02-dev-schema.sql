-- Development Database Schema
-- This represents the NEW schema with deprecated columns removed

USE testdb;

-- Users table - cleaned up (deprecated columns removed)
CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,

    -- phone, legacy_status, and old_address columns have been removed
    -- Data migrated to separate normalized tables

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Sample data (without deprecated column values)
INSERT INTO users (username, email) VALUES
('john_doe', 'john@example.com'),
('jane_smith', 'jane@example.com'),
('bob_wilson', 'bob@example.com'),
('alice_jones', 'alice@example.com'),
('new_user', 'newuser@example.com');  -- New user added in development

-- Products table (identical to production)
CREATE TABLE products (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    stock INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO products (name, price, stock) VALUES
('Widget A', 19.99, 100),
('Widget B', 29.99, 50),
('Gadget X', 49.99, 25);

-- Orders table (identical to production)
CREATE TABLE orders (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    total DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

INSERT INTO orders (user_id, total, status) VALUES
(1, 19.99, 'completed'),
(1, 49.99, 'pending'),
(2, 29.99, 'completed'),
(3, 19.99, 'cancelled');

-- New tables in development (these would be in "added tables" section)
CREATE TABLE user_status (
    user_id INT PRIMARY KEY,
    status VARCHAR(50) NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

INSERT INTO user_status (user_id, status) VALUES
(1, 'active'),
(2, 'active'),
(3, 'inactive'),
(4, 'active');
