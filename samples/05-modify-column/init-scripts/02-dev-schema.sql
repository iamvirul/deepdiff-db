-- Development Database Schema
-- This represents the NEW schema with modified column definitions

USE testdb;

-- Users table with modified columns
CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,

    -- Expanded column types (safe changes)
    username VARCHAR(255) NOT NULL,         -- Expanded from VARCHAR(50)
    age BIGINT NULL,                        -- Expanded from INT

    -- Converted column types (safe changes)
    description TEXT NOT NULL,              -- Converted from VARCHAR(500)

    -- Modified nullable constraints
    email VARCHAR(100) NOT NULL,            -- Changed from NULL to NOT NULL
    status VARCHAR(20) NULL,                -- Changed from NOT NULL to NULL (also expanded to VARCHAR(20) from VARCHAR(20))

    -- Modified DEFAULT values
    is_active TINYINT(1) DEFAULT 1,         -- Changed DEFAULT from 0 to 1
    score INT NOT NULL,                     -- Removed DEFAULT (was DEFAULT 0)
    created_count INT NOT NULL DEFAULT 100, -- Added DEFAULT 100 (had no default)

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Sample data (same as production, but with new schema)
INSERT INTO users (username, age, description, email, status, is_active, score, created_count) VALUES
('john_doe', 25, 'Software engineer with 5 years of experience', 'john@example.com', 'active', 1, 85, 10),
('jane_smith', 30, 'Product manager specializing in SaaS products', 'jane@example.com', 'active', 1, 92, 15),
('bob_wilson', 28, 'DevOps engineer', 'bob@example.com', NULL, 0, 78, 8),  -- status can now be NULL
('alice_jones', 35, 'Senior architect', 'alice@example.com', 'active', 1, 95, 20);

-- Products table (identical to production - should not appear in diff)
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
