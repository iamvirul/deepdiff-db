-- Production database — V1 baseline e-commerce schema
CREATE TABLE categories (
    id   INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE products (
    id         INT AUTO_INCREMENT PRIMARY KEY,
    name       VARCHAR(200)   NOT NULL,
    price      DECIMAL(10, 2) NOT NULL,
    stock      INT            NOT NULL DEFAULT 0,
    created_at TIMESTAMP      NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE orders (
    id          INT AUTO_INCREMENT PRIMARY KEY,
    product_id  INT            NOT NULL,
    quantity    INT            NOT NULL,
    total_price DECIMAL(10, 2) NOT NULL,
    status      VARCHAR(20)    NOT NULL DEFAULT 'pending',
    created_at  TIMESTAMP      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_order_product FOREIGN KEY (product_id) REFERENCES products (id)
);

INSERT INTO categories (name) VALUES ('Electronics'), ('Clothing'), ('Books');

INSERT INTO products (name, price, stock) VALUES
    ('Wireless Headphones', 79.99, 50),
    ('Running Shoes',       49.99, 120),
    ('Go Programming Book', 34.99, 200);

INSERT INTO orders (product_id, quantity, total_price, status) VALUES
    (1, 2, 159.98, 'completed'),
    (2, 1,  49.99, 'pending'),
    (3, 3, 104.97, 'completed');
