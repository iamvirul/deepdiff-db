-- Production Database Schema
-- Contains tables with existing indexes (some will be removed in dev)

USE testdb;

-- Users table with multiple indexes
CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Indexes on users table
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);
-- This index will be removed in dev (obsolete)
CREATE INDEX idx_users_legacy_search ON users(first_name, last_name, status);

-- Products table with indexes
CREATE TABLE products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    sku VARCHAR(50) NOT NULL,
    category_id INT,
    price DECIMAL(10, 2) NOT NULL,
    stock_quantity INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes on products table
CREATE UNIQUE INDEX idx_products_sku ON products(sku);
CREATE INDEX idx_products_category ON products(category_id);
CREATE INDEX idx_products_price ON products(price);
-- This redundant index will be removed in dev
CREATE INDEX idx_products_old_search ON products(name, category_id);

-- Orders table with composite indexes
CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    total_price DECIMAL(10, 2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes on orders table
CREATE INDEX idx_orders_user ON orders(user_id);
CREATE INDEX idx_orders_product ON orders(product_id);
CREATE INDEX idx_orders_status ON orders(status);
-- Composite index with old column order (will be modified in dev)
CREATE INDEX idx_orders_date_status ON orders(order_date, status);

-- Insert sample data
INSERT INTO users (username, email, first_name, last_name, status) VALUES
('john_doe', 'john@example.com', 'John', 'Doe', 'active'),
('jane_smith', 'jane@example.com', 'Jane', 'Smith', 'active'),
('bob_wilson', 'bob@example.com', 'Bob', 'Wilson', 'inactive');

INSERT INTO products (name, sku, category_id, price, stock_quantity) VALUES
('Widget A', 'WGT-001', 1, 29.99, 100),
('Widget B', 'WGT-002', 1, 39.99, 50),
('Gadget X', 'GDG-001', 2, 99.99, 25);

INSERT INTO orders (user_id, product_id, quantity, total_price, status) VALUES
(1, 1, 2, 59.98, 'completed'),
(1, 3, 1, 99.99, 'pending'),
(2, 2, 3, 119.97, 'shipped');
