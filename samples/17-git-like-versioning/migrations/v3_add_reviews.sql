-- V3: Product review system and average-rating denorm column
-- Applied to the development database on top of V2.

CREATE TABLE reviews (
    id         INT AUTO_INCREMENT PRIMARY KEY,
    product_id INT           NOT NULL,
    rating     TINYINT       NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment    TEXT          NULL,
    created_at TIMESTAMP     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_review_product FOREIGN KEY (product_id) REFERENCES products (id)
);

ALTER TABLE products
    ADD COLUMN avg_rating DECIMAL(3, 2) NULL AFTER stock;

INSERT INTO reviews (product_id, rating, comment) VALUES
    (1, 5, 'Excellent sound quality!'),
    (1, 4, 'Great value for money.'),
    (2, 5, 'Very comfortable for long runs.'),
    (3, 5, 'Best Go book I have read.');
