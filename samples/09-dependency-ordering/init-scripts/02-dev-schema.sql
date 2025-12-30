-- Development Database Schema
-- Demonstrates new tables with FK dependencies, requiring proper CREATE ordering

USE dev_db;

-- Base tables (unchanged from prod)
CREATE TABLE users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE categories (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    parent_id INT,
    FOREIGN KEY (parent_id) REFERENCES categories(id) ON DELETE SET NULL
) ENGINE=InnoDB;

CREATE TABLE products (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    category_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL
) ENGINE=InnoDB;

CREATE TABLE orders (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    total DECIMAL(10,2) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- NOTE: order_items, inventory, and audit_log are REMOVED from dev
-- This demonstrates DROP ordering - tables with FKs must be dropped first

-- NEW TABLES (will demonstrate CREATE ordering)
-- These tables have dependencies on users and products

-- shipping_addresses depends on users
CREATE TABLE shipping_addresses (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    address_line1 VARCHAR(255) NOT NULL,
    address_line2 VARCHAR(255),
    city VARCHAR(100) NOT NULL,
    postal_code VARCHAR(20) NOT NULL,
    country VARCHAR(100) NOT NULL,
    is_default TINYINT(1) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- reviews depends on both users AND products
CREATE TABLE reviews (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    product_id INT NOT NULL,
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- wishlists depends on both users AND products
CREATE TABLE wishlists (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    product_id INT NOT NULL,
    priority INT DEFAULT 0,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    UNIQUE KEY unique_wishlist (user_id, product_id)
) ENGINE=InnoDB;

-- order_history depends on orders (which depends on users)
-- This creates a chain: users -> orders -> order_history
CREATE TABLE order_history (
    id INT PRIMARY KEY AUTO_INCREMENT,
    order_id INT NOT NULL,
    status VARCHAR(50) NOT NULL,
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    notes TEXT,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- Insert sample data
INSERT INTO users (email, name) VALUES
('alice@example.com', 'Alice'),
('bob@example.com', 'Bob');

INSERT INTO categories (name, parent_id) VALUES
('Electronics', NULL),
('Phones', 1),
('Laptops', 1);

INSERT INTO products (name, price, category_id) VALUES
('iPhone 15', 999.00, 2),
('MacBook Pro', 2499.00, 3);

INSERT INTO orders (user_id, total, status) VALUES
(1, 999.00, 'completed'),
(2, 2499.00, 'pending');

INSERT INTO shipping_addresses (user_id, address_line1, city, postal_code, country, is_default) VALUES
(1, '123 Main St', 'New York', '10001', 'USA', 1),
(2, '456 Oak Ave', 'Los Angeles', '90001', 'USA', 1);

INSERT INTO reviews (user_id, product_id, rating, comment) VALUES
(1, 1, 5, 'Great phone!'),
(2, 2, 4, 'Excellent laptop');

INSERT INTO wishlists (user_id, product_id, priority) VALUES
(1, 2, 1),
(2, 1, 2);

INSERT INTO order_history (order_id, status, notes) VALUES
(1, 'pending', 'Order created'),
(1, 'processing', 'Payment received'),
(1, 'completed', 'Order shipped'),
(2, 'pending', 'Order created');
