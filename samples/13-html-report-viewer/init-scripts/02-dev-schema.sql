-- Development Database Schema and Data
-- Sample 13: HTML Report Viewer
-- This represents the development state with various changes

USE dev_ecommerce;

-- Categories table (unchanged)
CREATE TABLE categories (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    parent_id INT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Products table (NEW: discount_percent column added)
CREATE TABLE products (
    id INT PRIMARY KEY AUTO_INCREMENT,
    category_id INT NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    discount_percent DECIMAL(5, 2) DEFAULT NULL,  -- NEW COLUMN
    stock_quantity INT DEFAULT 0,
    sku VARCHAR(50) UNIQUE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (category_id) REFERENCES categories(id)
);

-- Customers table (MODIFIED: email column expanded)
CREATE TABLE customers (
    id INT PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(255) NOT NULL UNIQUE,  -- CHANGED: 100 -> 255
    name VARCHAR(150) NOT NULL,
    phone VARCHAR(20),
    tier ENUM('bronze', 'silver', 'gold', 'platinum') DEFAULT 'bronze',
    loyalty_points INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Orders table (NEW: index on customer_id)
CREATE TABLE orders (
    id INT PRIMARY KEY AUTO_INCREMENT,
    customer_id INT NOT NULL,
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status ENUM('pending', 'processing', 'shipped', 'delivered', 'cancelled') DEFAULT 'pending',
    total_amount DECIMAL(12, 2) NOT NULL,
    shipping_address TEXT,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (customer_id) REFERENCES customers(id),
    INDEX idx_customer_id (customer_id)  -- NEW INDEX
);

-- Order items table (unchanged)
CREATE TABLE order_items (
    id INT PRIMARY KEY AUTO_INCREMENT,
    order_id INT NOT NULL,
    quantity INT NOT NULL,
    product_id INT NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (order_id) REFERENCES orders(id),
    FOREIGN KEY (product_id) REFERENCES products(id)
);

-- Inventory log table (unchanged)
CREATE TABLE inventory_log (
    id INT PRIMARY KEY AUTO_INCREMENT,
    product_id INT NOT NULL,
    change_type ENUM('restock', 'sale', 'adjustment', 'return') NOT NULL,
    quantity_change INT NOT NULL,
    previous_quantity INT NOT NULL,
    new_quantity INT NOT NULL,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES products(id)
);

-- Audit log table (unchanged, but data will differ)
CREATE TABLE audit_log (
    id INT PRIMARY KEY AUTO_INCREMENT,
    table_name VARCHAR(100) NOT NULL,
    record_id INT NOT NULL,
    action ENUM('insert', 'update', 'delete') NOT NULL,
    old_values JSON,
    new_values JSON,
    user_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- NEW TABLE: Feature flags (only in dev)
CREATE TABLE feature_flags (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    is_enabled BOOLEAN DEFAULT FALSE,
    rollout_percentage INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- ============================================================
-- VIEWS
-- ============================================================

-- MODIFIED: adds shipping_address column vs prod
CREATE VIEW v_active_orders AS
    SELECT o.id, o.customer_id, c.name AS customer_name,
           o.total_amount, o.status, o.order_date,
           o.shipping_address
    FROM orders o
    JOIN customers c ON o.customer_id = c.id
    WHERE o.status NOT IN ('cancelled');

-- UNCHANGED
CREATE VIEW v_product_catalog AS
    SELECT p.id, p.name, p.price, p.stock_quantity,
           cat.name AS category
    FROM products p
    JOIN categories cat ON p.category_id = cat.id
    WHERE p.is_active = TRUE;

-- REMOVED: v_customer_summary (not created in dev)
-- NEW: v_customer_stats replaces it with avg_order_value added
CREATE VIEW v_customer_stats AS
    SELECT customer_id,
           COUNT(*) AS order_count,
           SUM(total_amount) AS lifetime_value,
           AVG(total_amount) AS avg_order_value
    FROM orders
    GROUP BY customer_id;

-- ============================================================
-- ROUTINES
-- ============================================================

DELIMITER $$

-- MODIFIED: added discount_pct parameter
CREATE FUNCTION fn_calculate_total(base_price DECIMAL(10,2), qty INT, discount_pct DECIMAL(5,2))
RETURNS DECIMAL(12,2)
DETERMINISTIC
BEGIN
    DECLARE discount DECIMAL(10,2);
    SET discount = IFNULL(discount_pct, 0);
    RETURN base_price * qty * (1 - discount / 100);
END$$

-- MODIFIED: adds platinum tier (>= 5000)
CREATE FUNCTION fn_get_customer_tier(spend_total DECIMAL(12,2))
RETURNS VARCHAR(10)
DETERMINISTIC
BEGIN
    IF spend_total >= 5000 THEN
        RETURN 'platinum';
    ELSEIF spend_total >= 1000 THEN
        RETURN 'gold';
    END IF;
    RETURN 'silver';
END$$

-- UNCHANGED
CREATE PROCEDURE sp_process_order(IN p_order_id INT)
BEGIN
    UPDATE orders SET status = 'processing' WHERE id = p_order_id;
END$$

-- NEW: fn_format_price
CREATE FUNCTION fn_format_price(price DECIMAL(10,2))
RETURNS VARCHAR(20)
DETERMINISTIC
BEGIN
    RETURN CONCAT('$', FORMAT(price, 2));
END$$

DELIMITER ;

-- ============================================================
-- INSERT DEVELOPMENT DATA
-- ============================================================

-- Categories (same as prod)
INSERT INTO categories (id, name, description) VALUES
(1, 'Electronics', 'Electronic devices and accessories'),
(2, 'Clothing', 'Apparel and fashion items'),
(3, 'Home & Garden', 'Home decor and garden supplies'),
(4, 'Sports', 'Sports equipment and accessories'),
(5, 'Books', 'Books and publications');

-- Products (with changes and additions)
INSERT INTO products (id, category_id, name, description, price, discount_percent, stock_quantity, sku, is_active) VALUES
-- Existing products (some with price changes)
(1, 1, 'Wireless Mouse', 'Ergonomic wireless mouse with USB receiver', 24.99, 10.00, 150, 'ELEC-001', TRUE),  -- PRICE CHANGED: 29.99 -> 24.99, discount added
(2, 1, 'USB-C Hub', '7-port USB-C hub with HDMI output', 49.99, NULL, 80, 'ELEC-002', TRUE),
(3, 1, 'Bluetooth Headphones', 'Over-ear noise cancelling headphones', 129.99, 15.00, 45, 'ELEC-003', TRUE),  -- PRICE CHANGED: 149.99 -> 129.99, discount added
(4, 2, 'Cotton T-Shirt', '100% cotton crew neck t-shirt', 19.99, NULL, 200, 'CLTH-001', TRUE),
(5, 2, 'Denim Jeans', 'Classic fit denim jeans', 59.99, NULL, 120, 'CLTH-002', TRUE),
(6, 3, 'Plant Pot Set', 'Set of 3 ceramic plant pots', 34.99, NULL, 60, 'HOME-001', TRUE),
(7, 3, 'LED Desk Lamp', 'Adjustable LED desk lamp with dimmer', 44.99, NULL, 90, 'HOME-002', TRUE),
(8, 4, 'Yoga Mat', 'Non-slip yoga mat 6mm thick', 24.99, NULL, 100, 'SPRT-001', TRUE),
(9, 4, 'Resistance Bands', 'Set of 5 resistance bands', 19.99, NULL, 150, 'SPRT-002', TRUE),
(10, 5, 'Coding Book', 'Learn Programming in 30 Days', 39.99, NULL, 75, 'BOOK-001', TRUE),
-- NEW products
(11, 1, 'Mechanical Keyboard', 'RGB mechanical keyboard with Cherry MX switches', 89.99, 5.00, 50, 'ELEC-004', TRUE),
(12, 1, 'Webcam HD', '1080p HD webcam with microphone', 59.99, NULL, 40, 'ELEC-005', TRUE),
(13, 4, 'Dumbbell Set', 'Adjustable dumbbell set 5-25 lbs', 149.99, 10.00, 30, 'SPRT-003', TRUE);

-- Customers (with tier and loyalty point changes - creates conflicts)
INSERT INTO customers (id, email, name, phone, tier, loyalty_points) VALUES
(1, 'john.doe@email.com', 'John Doe', '+1-555-0101', 'platinum', 2500),  -- CONFLICT: tier gold->platinum, points 1500->2500
(2, 'jane.smith@email.com', 'Jane Smith', '+1-555-0102', 'gold', 1200),  -- CONFLICT: tier silver->gold, points 800->1200
(3, 'bob.wilson@email.com', 'Bob Wilson', '+1-555-0103', 'silver', 500),  -- CONFLICT: tier bronze->silver, points 200->500
(4, 'alice.johnson@email.com', 'Alice Johnson', '+1-555-0104', 'gold', 950),  -- CONFLICT: points 650->950
(5, 'charlie.brown@email.com', 'Charlie Brown', '+1-555-0105', 'platinum', 3500),  -- CONFLICT: points 3000->3500
-- NEW customer
(6, 'diana.prince@email.com', 'Diana Prince', '+1-555-0106', 'bronze', 100);

-- Orders (with some additions and changes)
INSERT INTO orders (id, customer_id, order_date, status, total_amount, shipping_address) VALUES
(1, 1, '2025-12-01 10:00:00', 'delivered', 79.98, '123 Main St, City, ST 12345'),
(2, 2, '2025-12-05 14:30:00', 'delivered', 149.99, '456 Oak Ave, Town, ST 67890'),
(3, 3, '2025-12-10 09:15:00', 'delivered', 54.98, '789 Pine Rd, Village, ST 11111'),  -- CHANGED: shipped -> delivered
(4, 1, '2025-12-15 16:45:00', 'shipped', 119.97, '123 Main St, City, ST 12345'),  -- CHANGED: processing -> shipped
(5, 4, '2025-12-20 11:00:00', 'processing', 64.98, '321 Elm Blvd, Metro, ST 22222'),  -- CHANGED: pending -> processing
-- NEW orders
(6, 5, '2025-12-22 13:30:00', 'pending', 89.99, '555 Cedar Lane, Suburb, ST 33333'),
(7, 6, '2025-12-23 15:00:00', 'pending', 59.99, '777 Maple Dr, County, ST 44444'),
(8, 1, '2025-12-24 10:00:00', 'pending', 179.98, '123 Main St, City, ST 12345'),
(9, 2, '2025-12-25 09:00:00', 'pending', 149.99, '456 Oak Ave, Town, ST 67890'),
(10, 3, '2025-12-26 11:30:00', 'pending', 44.99, '789 Pine Rd, Village, ST 11111');

-- Order items
INSERT INTO order_items (id, order_id, product_id, quantity, unit_price) VALUES
(1, 1, 1, 1, 29.99),
(2, 1, 2, 1, 49.99),
(3, 2, 3, 1, 149.99),
(4, 3, 4, 2, 19.99),
(5, 3, 8, 1, 24.99),
(6, 4, 5, 2, 59.99),
(7, 5, 6, 1, 34.99),
(8, 5, 9, 1, 19.99),
-- NEW order items
(9, 6, 11, 1, 89.99),
(10, 7, 12, 1, 59.99),
(11, 8, 11, 1, 89.99),
(12, 8, 11, 1, 89.99),
(13, 9, 3, 1, 129.99),
(14, 10, 7, 1, 44.99);

-- Inventory log (with additions)
INSERT INTO inventory_log (id, product_id, change_type, quantity_change, previous_quantity, new_quantity, notes) VALUES
(1, 1, 'restock', 50, 100, 150, 'Monthly restock'),
(2, 2, 'sale', -5, 85, 80, 'Order #1'),
(3, 3, 'sale', -1, 46, 45, 'Order #2'),
(4, 4, 'sale', -2, 202, 200, 'Order #3'),
(5, 8, 'sale', -1, 101, 100, 'Order #3'),
-- NEW entries
(6, 11, 'restock', 50, 0, 50, 'Initial stock'),
(7, 12, 'restock', 40, 0, 40, 'Initial stock'),
(8, 13, 'restock', 30, 0, 30, 'Initial stock'),
(9, 1, 'sale', -2, 150, 148, 'Holiday sale'),
(10, 3, 'sale', -3, 45, 42, 'Holiday sale');

-- Audit log (fewer entries - some removed)
INSERT INTO audit_log (id, table_name, record_id, action, old_values, new_values, user_id) VALUES
(1, 'products', 1, 'update', '{"price": 24.99}', '{"price": 29.99}', 1),
(2, 'customers', 1, 'update', '{"tier": "silver"}', '{"tier": "gold"}', 1),
(5, 'customers', 5, 'insert', NULL, '{"name": "Charlie Brown"}', 1);
-- Entries 3, 4, 6, 7, 8, 9, 10 are removed in dev (audit cleanup)

-- Feature flags (NEW TABLE - only in dev)
INSERT INTO feature_flags (id, name, description, is_enabled, rollout_percentage) VALUES
(1, 'dark_mode', 'Enable dark mode UI', TRUE, 100),
(2, 'new_checkout', 'New streamlined checkout flow', TRUE, 50),
(3, 'loyalty_v2', 'Enhanced loyalty program', FALSE, 0),
(4, 'ai_recommendations', 'AI-powered product recommendations', TRUE, 25);

-- ============================================================
-- TRIGGERS (created after data load to avoid audit_log conflicts)
-- ============================================================

-- MODIFIED: logs status field in addition to customer_id and amount
CREATE TRIGGER trg_orders_audit
AFTER INSERT ON orders
FOR EACH ROW
    INSERT INTO audit_log (table_name, record_id, action, new_values)
    VALUES ('orders', NEW.id, 'insert',
            JSON_OBJECT('customer_id', NEW.customer_id, 'amount', NEW.total_amount,
                        'status', NEW.status));

DELIMITER $$

-- UNCHANGED
CREATE TRIGGER trg_inventory_update
AFTER INSERT ON inventory_log
FOR EACH ROW
BEGIN
    UPDATE products
    SET stock_quantity = NEW.new_quantity
    WHERE id = NEW.product_id;
END$$

-- NEW trigger
CREATE TRIGGER trg_products_updated_at
BEFORE UPDATE ON products
FOR EACH ROW
BEGIN
    SET NEW.updated_at = NOW();
END$$

DELIMITER ;
