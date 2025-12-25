-- Production Database Schema
-- This represents the OLD schema with deprecated columns that need to be removed

USE testdb;

-- Users table with deprecated columns
CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL,

    -- Deprecated columns (to be removed)
    phone VARCHAR(20),                    -- Being removed: using separate phone_numbers table now
    legacy_status VARCHAR(50),            -- Being removed: replaced by user_status table
    old_address TEXT,                     -- Being removed: using addresses table now

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Sample data
INSERT INTO users (username, email, phone, legacy_status, old_address) VALUES
('john_doe', 'john@example.com', '555-0100', 'active', '123 Main St'),
('jane_smith', 'jane@example.com', '555-0101', 'active', '456 Oak Ave'),
('bob_wilson', 'bob@example.com', '555-0102', 'inactive', '789 Pine Rd'),
('alice_jones', 'alice@example.com', NULL, 'active', NULL);

-- Products table (unchanged between prod and dev)
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

-- Orders table (unchanged)
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
