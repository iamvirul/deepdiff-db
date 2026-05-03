-- Production Database Schema and Data
-- Sample 13: HTML Report Viewer
-- This represents the current production state

USE prod_ecommerce;

-- Categories table
CREATE TABLE categories (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    parent_id INT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Products table (will have column added in dev)
CREATE TABLE products (
    id INT PRIMARY KEY AUTO_INCREMENT,
    category_id INT NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    stock_quantity INT DEFAULT 0,
    sku VARCHAR(50) UNIQUE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (category_id) REFERENCES categories(id)
);

-- Customers table (email column will be modified in dev)
CREATE TABLE customers (
    id INT PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    phone VARCHAR(20),
    tier ENUM('bronze', 'silver', 'gold', 'platinum') DEFAULT 'bronze',
    loyalty_points INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Orders table (will have index added in dev)
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
    FOREIGN KEY (customer_id) REFERENCES customers(id)
);

-- Order items table
CREATE TABLE order_items (
    id INT PRIMARY KEY AUTO_INCREMENT,
    order_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (order_id) REFERENCES orders(id),
    FOREIGN KEY (product_id) REFERENCES products(id)
);

-- Inventory log table
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

-- Audit log table
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

-- ============================================================
-- VIEWS
-- ============================================================

CREATE VIEW v_active_orders AS
    SELECT o.id, o.customer_id, c.name AS customer_name,
           o.total_amount, o.status, o.order_date
    FROM orders o
    JOIN customers c ON o.customer_id = c.id
    WHERE o.status NOT IN ('cancelled');

CREATE VIEW v_product_catalog AS
    SELECT p.id, p.name, p.price, p.stock_quantity,
           cat.name AS category
    FROM products p
    JOIN categories cat ON p.category_id = cat.id
    WHERE p.is_active = TRUE;

CREATE VIEW v_customer_summary AS
    SELECT customer_id,
           COUNT(*) AS order_count,
           SUM(total_amount) AS lifetime_value
    FROM orders
    GROUP BY customer_id;

-- ============================================================
-- ROUTINES
-- ============================================================

DELIMITER $$

CREATE FUNCTION fn_calculate_total(base_price DECIMAL(10,2), qty INT)
RETURNS DECIMAL(12,2)
DETERMINISTIC
BEGIN
    RETURN base_price * qty;
END$$

CREATE FUNCTION fn_get_customer_tier(spend_total DECIMAL(12,2))
RETURNS VARCHAR(10)
DETERMINISTIC
BEGIN
    IF spend_total >= 1000 THEN
        RETURN 'gold';
    END IF;
    RETURN 'silver';
END$$

CREATE PROCEDURE sp_process_order(IN p_order_id INT)
BEGIN
    UPDATE orders SET status = 'processing' WHERE id = p_order_id;
END$$

DELIMITER ;

-- ============================================================
-- INSERT PRODUCTION DATA
-- ============================================================

-- Categories
INSERT INTO categories (id, name, description) VALUES
(1, 'Electronics', 'Electronic devices and accessories'),
(2, 'Clothing', 'Apparel and fashion items'),
(3, 'Home & Garden', 'Home decor and garden supplies'),
(4, 'Sports', 'Sports equipment and accessories'),
(5, 'Books', 'Books and publications');

-- Products
INSERT INTO products (id, category_id, name, description, price, stock_quantity, sku, is_active) VALUES
(1, 1, 'Wireless Mouse', 'Ergonomic wireless mouse with USB receiver', 29.99, 150, 'ELEC-001', TRUE),
(2, 1, 'USB-C Hub', '7-port USB-C hub with HDMI output', 49.99, 80, 'ELEC-002', TRUE),
(3, 1, 'Bluetooth Headphones', 'Over-ear noise cancelling headphones', 149.99, 45, 'ELEC-003', TRUE),
(4, 2, 'Cotton T-Shirt', '100% cotton crew neck t-shirt', 19.99, 200, 'CLTH-001', TRUE),
(5, 2, 'Denim Jeans', 'Classic fit denim jeans', 59.99, 120, 'CLTH-002', TRUE),
(6, 3, 'Plant Pot Set', 'Set of 3 ceramic plant pots', 34.99, 60, 'HOME-001', TRUE),
(7, 3, 'LED Desk Lamp', 'Adjustable LED desk lamp with dimmer', 44.99, 90, 'HOME-002', TRUE),
(8, 4, 'Yoga Mat', 'Non-slip yoga mat 6mm thick', 24.99, 100, 'SPRT-001', TRUE),
(9, 4, 'Resistance Bands', 'Set of 5 resistance bands', 19.99, 150, 'SPRT-002', TRUE),
(10, 5, 'Coding Book', 'Learn Programming in 30 Days', 39.99, 75, 'BOOK-001', TRUE);

-- Customers
INSERT INTO customers (id, email, name, phone, tier, loyalty_points) VALUES
(1, 'john.doe@email.com', 'John Doe', '+1-555-0101', 'gold', 1500),
(2, 'jane.smith@email.com', 'Jane Smith', '+1-555-0102', 'silver', 800),
(3, 'bob.wilson@email.com', 'Bob Wilson', '+1-555-0103', 'bronze', 200),
(4, 'alice.johnson@email.com', 'Alice Johnson', '+1-555-0104', 'silver', 650),
(5, 'charlie.brown@email.com', 'Charlie Brown', '+1-555-0105', 'platinum', 3000);

-- Orders
INSERT INTO orders (id, customer_id, order_date, status, total_amount, shipping_address) VALUES
(1, 1, '2025-12-01 10:00:00', 'delivered', 79.98, '123 Main St, City, ST 12345'),
(2, 2, '2025-12-05 14:30:00', 'delivered', 149.99, '456 Oak Ave, Town, ST 67890'),
(3, 3, '2025-12-10 09:15:00', 'shipped', 54.98, '789 Pine Rd, Village, ST 11111'),
(4, 1, '2025-12-15 16:45:00', 'processing', 119.97, '123 Main St, City, ST 12345'),
(5, 4, '2025-12-20 11:00:00', 'pending', 64.98, '321 Elm Blvd, Metro, ST 22222');

-- Order items
INSERT INTO order_items (id, order_id, product_id, quantity, unit_price) VALUES
(1, 1, 1, 1, 29.99),
(2, 1, 2, 1, 49.99),
(3, 2, 3, 1, 149.99),
(4, 3, 4, 2, 19.99),
(5, 3, 8, 1, 24.99),
(6, 4, 5, 2, 59.99),
(7, 5, 6, 1, 34.99),
(8, 5, 9, 1, 19.99);

-- Inventory log
INSERT INTO inventory_log (id, product_id, change_type, quantity_change, previous_quantity, new_quantity, notes) VALUES
(1, 1, 'restock', 50, 100, 150, 'Monthly restock'),
(2, 2, 'sale', -5, 85, 80, 'Order #1'),
(3, 3, 'sale', -1, 46, 45, 'Order #2'),
(4, 4, 'sale', -2, 202, 200, 'Order #3'),
(5, 8, 'sale', -1, 101, 100, 'Order #3');

-- Audit log (some entries will be removed in dev)
INSERT INTO audit_log (id, table_name, record_id, action, old_values, new_values, user_id) VALUES
(1, 'products', 1, 'update', '{"price": 24.99}', '{"price": 29.99}', 1),
(2, 'customers', 1, 'update', '{"tier": "silver"}', '{"tier": "gold"}', 1),
(3, 'orders', 1, 'update', '{"status": "pending"}', '{"status": "delivered"}', 2),
(4, 'products', 3, 'update', '{"stock_quantity": 50}', '{"stock_quantity": 45}', 1),
(5, 'customers', 5, 'insert', NULL, '{"name": "Charlie Brown"}', 1),
(6, 'orders', 3, 'update', '{"status": "processing"}', '{"status": "shipped"}', 2),
(7, 'inventory_log', 1, 'insert', NULL, '{"quantity_change": 50}', 1),
(8, 'products', 2, 'update', '{"price": 44.99}', '{"price": 49.99}', 1),
(9, 'customers', 2, 'update', '{"loyalty_points": 500}', '{"loyalty_points": 800}', 1),
(10, 'orders', 4, 'insert', NULL, '{"customer_id": 1}', 2);

-- ============================================================
-- TRIGGERS (created after data load to avoid audit_log conflicts)
-- ============================================================

CREATE TRIGGER trg_orders_audit
AFTER INSERT ON orders
FOR EACH ROW
    INSERT INTO audit_log (table_name, record_id, action, new_values)
    VALUES ('orders', NEW.id, 'insert',
            JSON_OBJECT('customer_id', NEW.customer_id, 'amount', NEW.total_amount));

DELIMITER $$

CREATE TRIGGER trg_inventory_update
AFTER INSERT ON inventory_log
FOR EACH ROW
BEGIN
    UPDATE products
    SET stock_quantity = NEW.new_quantity
    WHERE id = NEW.product_id;
END$$

DELIMITER ;
