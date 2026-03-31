-- V2: Link products to categories and capture customer email on orders
-- Applied to the development database to simulate an in-progress sprint.

ALTER TABLE products
    ADD COLUMN category_id INT NULL AFTER name,
    ADD CONSTRAINT fk_product_category FOREIGN KEY (category_id) REFERENCES categories (id);

ALTER TABLE orders
    ADD COLUMN customer_email VARCHAR(255) NULL AFTER status;

UPDATE products SET category_id = 1 WHERE id = 1;
UPDATE products SET category_id = 2 WHERE id = 2;
UPDATE products SET category_id = 3 WHERE id = 3;
