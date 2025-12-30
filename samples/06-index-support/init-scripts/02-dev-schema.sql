-- Development Database Schema
-- Contains tables with new/optimized indexes

USE testdb;

-- Users table with optimized indexes
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

-- Optimized indexes on users table
CREATE UNIQUE INDEX idx_users_email ON users(email);  -- Changed to UNIQUE (was non-unique)
CREATE INDEX idx_users_username ON users(username);   -- Same as prod
-- Removed: idx_users_status (rarely used)
-- Removed: idx_users_created_at (not needed)
-- Removed: idx_users_legacy_search (obsolete composite index)
-- NEW: Optimized composite index for name search
CREATE INDEX idx_users_fullname ON users(last_name, first_name);

-- Products table with optimized indexes
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

-- Optimized indexes on products table
CREATE UNIQUE INDEX idx_products_sku ON products(sku);      -- Same as prod
CREATE INDEX idx_products_category ON products(category_id); -- Same as prod
-- Removed: idx_products_price (rarely queried alone)
-- Removed: idx_products_old_search (redundant)
-- NEW: Composite index for active products by category
CREATE INDEX idx_products_active_category ON products(is_active, category_id);
-- NEW: Full-text style search optimization
CREATE INDEX idx_products_name ON products(name);

-- Orders table with optimized composite indexes
CREATE TABLE orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    total_price DECIMAL(10, 2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Optimized indexes on orders table
CREATE INDEX idx_orders_user ON orders(user_id);       -- Same as prod
CREATE INDEX idx_orders_product ON orders(product_id); -- Same as prod
-- Removed: idx_orders_status (covered by composite)
-- Modified: idx_orders_date_status -> idx_orders_status_date (different column order for query optimization)
CREATE INDEX idx_orders_status_date ON orders(status, order_date);
-- NEW: Composite index for user order history
CREATE INDEX idx_orders_user_date ON orders(user_id, order_date);

-- Insert sample data (same as prod)
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
