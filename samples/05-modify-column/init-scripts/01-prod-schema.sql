-- Production Database Schema
-- This represents the OLD schema with columns that need modifications

USE testdb;

-- Users table with columns that will be modified
CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,

    -- Column type expansions (will be expanded in dev)
    username VARCHAR(50) NOT NULL,          -- Will expand to VARCHAR(255)
    age INT NULL,                           -- Will expand to BIGINT

    -- Column type conversions (will be converted in dev)
    description VARCHAR(500) NOT NULL,      -- Will convert to TEXT

    -- Nullable constraint changes
    email VARCHAR(100) NULL,                -- Will change to NOT NULL (requires data validation)
    status VARCHAR(20) NOT NULL,            -- Will change to NULL

    -- DEFAULT value changes
    is_active TINYINT(1) DEFAULT 0,         -- Will change DEFAULT to 1
    score INT NOT NULL DEFAULT 0,           -- Will remove DEFAULT
    created_count INT NOT NULL,             -- Will add DEFAULT 100

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Sample data (with valid values for all columns)
INSERT INTO users (username, age, description, email, status, is_active, score, created_count) VALUES
('john_doe', 25, 'Software engineer with 5 years of experience', 'john@example.com', 'active', 1, 85, 10),
('jane_smith', 30, 'Product manager specializing in SaaS products', 'jane@example.com', 'active', 1, 92, 15),
('bob_wilson', 28, 'DevOps engineer', 'bob@example.com', 'inactive', 0, 78, 8),
('alice_jones', 35, 'Senior architect', 'alice@example.com', 'active', 1, 95, 20);

-- Products table (unchanged - for testing that unchanged tables are not in output)
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
