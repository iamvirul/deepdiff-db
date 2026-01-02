-- Production Database Schema and Data
-- This script sets up the production database with initial data
-- that will have conflicts with the development database.

USE testdb;

-- ============================================
-- Products Table (strategy: theirs)
-- E-commerce products - dev has updated prices
-- ============================================
CREATE TABLE products (
    id INT PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    price DECIMAL(10,2) NOT NULL,
    stock INT NOT NULL,
    category VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

INSERT INTO products (id, name, price, stock, category) VALUES
(1, 'Laptop Pro', 1299.99, 50, 'electronics'),      -- CONFLICT: price changed in dev
(2, 'Wireless Mouse', 29.99, 200, 'electronics'),   -- CONFLICT: stock changed in dev
(3, 'USB-C Cable', 12.99, 500, 'accessories'),      -- Same in both
(4, 'Monitor 27"', 399.99, 30, 'electronics'),      -- CONFLICT: price changed in dev
(5, 'Keyboard', 79.99, 100, 'electronics');         -- Removed in dev

-- ============================================
-- Orders Table (strategy: ours)
-- Order data - production is source of truth
-- ============================================
CREATE TABLE orders (
    id INT PRIMARY KEY,
    customer_name VARCHAR(100) NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    total DECIMAL(10,2) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

INSERT INTO orders (id, customer_name, product_id, quantity, total, status) VALUES
(1, 'John Doe', 1, 1, 1299.99, 'completed'),        -- CONFLICT: status differs in dev
(2, 'Jane Smith', 2, 3, 89.97, 'shipped'),          -- CONFLICT: quantity differs in dev
(3, 'Bob Wilson', 3, 10, 129.90, 'pending'),        -- Same in both
(4, 'Alice Brown', 4, 1, 399.99, 'processing');     -- CONFLICT: status differs in dev

-- ============================================
-- Inventory Log Table (strategy: theirs)
-- Ephemeral log data - dev version is newer
-- ============================================
CREATE TABLE inventory_log (
    id INT PRIMARY KEY,
    product_id INT NOT NULL,
    action VARCHAR(20) NOT NULL,
    quantity_change INT NOT NULL,
    notes TEXT,
    logged_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO inventory_log (id, product_id, action, quantity_change, notes) VALUES
(1, 1, 'restock', 20, 'Weekly restock'),            -- CONFLICT: notes differ in dev
(2, 2, 'sale', -5, 'Online order'),                 -- CONFLICT: quantity differs in dev
(3, 3, 'restock', 100, 'Bulk order'),               -- Same in both
(4, 4, 'adjustment', -2, 'Damaged items');          -- Removed in dev

-- ============================================
-- Customers Table (strategy: manual)
-- Critical customer data - requires review
-- ============================================
CREATE TABLE customers (
    id INT PRIMARY KEY,
    email VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    tier VARCHAR(20) DEFAULT 'standard',
    loyalty_points INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

INSERT INTO customers (id, email, name, tier, loyalty_points) VALUES
(1, 'john@example.com', 'John Doe', 'gold', 1500),          -- CONFLICT: tier and points differ
(2, 'jane@example.com', 'Jane Smith', 'standard', 200),     -- CONFLICT: points differ
(3, 'bob@example.com', 'Bob Wilson', 'standard', 50),       -- Same in both
(4, 'alice@example.com', 'Alice Brown', 'silver', 800);     -- CONFLICT: tier differs

-- ============================================
-- Feature Flags Table (strategy: ours)
-- Production feature flags - must be preserved
-- ============================================
CREATE TABLE feature_flags (
    id INT PRIMARY KEY,
    flag_name VARCHAR(50) NOT NULL UNIQUE,
    enabled BOOLEAN DEFAULT FALSE,
    rollout_percentage INT DEFAULT 0,
    description TEXT
);

INSERT INTO feature_flags (id, flag_name, enabled, rollout_percentage, description) VALUES
(1, 'new_checkout', TRUE, 100, 'New checkout flow'),        -- CONFLICT: enabled differs
(2, 'dark_mode', TRUE, 50, 'Dark mode UI'),                 -- CONFLICT: rollout differs
(3, 'beta_features', FALSE, 0, 'Beta feature access'),      -- Same in both
(4, 'ai_recommendations', TRUE, 25, 'AI product recs');     -- CONFLICT: rollout differs

-- ============================================
-- Audit Trail Table (strategy: theirs)
-- Audit logs - dev has more recent entries
-- ============================================
CREATE TABLE audit_trail (
    id INT PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL,
    entity_id INT NOT NULL,
    action VARCHAR(20) NOT NULL,
    old_value TEXT,
    new_value TEXT,
    performed_by VARCHAR(100),
    performed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO audit_trail (id, entity_type, entity_id, action, old_value, new_value, performed_by) VALUES
(1, 'product', 1, 'update', '{"price":1199.99}', '{"price":1299.99}', 'admin'),
(2, 'order', 1, 'update', '{"status":"pending"}', '{"status":"completed"}', 'system'),
(3, 'customer', 1, 'update', '{"tier":"silver"}', '{"tier":"gold"}', 'admin');
