-- Development Database Schema and Data
-- This script sets up the development database with data
-- that creates conflicts with the production database.

USE testdb;

-- ============================================
-- Products Table (strategy: theirs)
-- Dev has updated prices and stock levels
-- Resolution: use_dev for all conflicts
-- ============================================
CREATE TABLE products (
    id INT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    stock INT NOT NULL,
    category VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

INSERT INTO products (id, name, price, stock, category) VALUES
(1, 'Laptop Pro', 1199.99, 50, 'electronics'),      -- CONFLICT: price lowered (sale)
(2, 'Wireless Mouse', 29.99, 180, 'electronics'),   -- CONFLICT: stock reduced (sales)
(3, 'USB-C Cable', 12.99, 500, 'accessories'),      -- Same in both
(4, 'Monitor 27"', 349.99, 30, 'electronics'),      -- CONFLICT: price lowered (promotion)
(6, 'Webcam HD', 89.99, 75, 'electronics');         -- NEW: added in dev

-- ============================================
-- Orders Table (strategy: ours)
-- Production order data is authoritative
-- Resolution: keep_prod for all conflicts
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
(1, 'John Doe', 1, 1, 1299.99, 'pending'),          -- CONFLICT: status reverted (test data)
(2, 'Jane Smith', 2, 5, 149.95, 'shipped'),         -- CONFLICT: different quantity (test)
(3, 'Bob Wilson', 3, 10, 129.90, 'pending'),        -- Same in both
(4, 'Alice Brown', 4, 1, 399.99, 'pending'),        -- CONFLICT: different status (test)
(5, 'Charlie Green', 1, 2, 2599.98, 'pending');     -- NEW: test order

-- ============================================
-- Inventory Log Table (strategy: theirs)
-- Dev has more recent log entries
-- Resolution: use_dev for all conflicts
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
(1, 1, 'restock', 20, 'Weekly restock - verified'),     -- CONFLICT: notes updated
(2, 2, 'sale', -10, 'Online orders batch'),             -- CONFLICT: larger quantity
(3, 3, 'restock', 100, 'Bulk order'),                   -- Same in both
(5, 6, 'initial', 75, 'New product added'),             -- NEW: for new product
(6, 1, 'sale', -5, 'Recent sales');                     -- NEW: additional log

-- ============================================
-- Customers Table (strategy: manual)
-- Critical data - conflicts need human review
-- Resolution: pending for all conflicts
-- ============================================
CREATE TABLE customers (
    id INT PRIMARY KEY,
    email VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    tier VARCHAR(20) DEFAULT 'standard',
    loyalty_points INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

INSERT INTO customers (id, email, name, tier, loyalty_points) VALUES
(1, 'john@example.com', 'John Doe', 'platinum', 2500),  -- CONFLICT: upgraded tier
(2, 'jane@example.com', 'Jane Smith', 'standard', 350), -- CONFLICT: more points
(3, 'bob@example.com', 'Bob Wilson', 'standard', 50),   -- Same in both
(4, 'alice@example.com', 'Alice Brown', 'gold', 1200),  -- CONFLICT: upgraded tier
(5, 'charlie@example.com', 'Charlie Green', 'standard', 0); -- NEW: new customer

-- ============================================
-- Feature Flags Table (strategy: ours)
-- Production flags must be preserved
-- Resolution: keep_prod for all conflicts
-- ============================================
CREATE TABLE feature_flags (
    id INT PRIMARY KEY,
    flag_name VARCHAR(50) UNIQUE NOT NULL,
    enabled BOOLEAN DEFAULT FALSE,
    rollout_percentage INT DEFAULT 0,
    description TEXT
);

INSERT INTO feature_flags (id, flag_name, enabled, rollout_percentage, description) VALUES
(1, 'new_checkout', FALSE, 0, 'New checkout flow'),     -- CONFLICT: disabled in dev
(2, 'dark_mode', TRUE, 100, 'Dark mode UI'),            -- CONFLICT: full rollout in dev
(3, 'beta_features', FALSE, 0, 'Beta feature access'),  -- Same in both
(4, 'ai_recommendations', TRUE, 100, 'AI product recs'),-- CONFLICT: full rollout in dev
(5, 'mobile_app', TRUE, 50, 'Mobile app features');     -- NEW: new flag

-- ============================================
-- Audit Trail Table (strategy: theirs)
-- Dev has additional audit entries
-- Resolution: use_dev for changes
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
(3, 'customer', 1, 'update', '{"tier":"silver"}', '{"tier":"gold"}', 'admin'),
(4, 'product', 1, 'update', '{"price":1299.99}', '{"price":1199.99}', 'admin'),
(5, 'customer', 1, 'update', '{"tier":"gold"}', '{"tier":"platinum"}', 'system');
