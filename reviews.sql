CREATE TABLE reviews (
    review_id INT AUTO_INCREMENT PRIMARY KEY,
    customer_id INT NOT NULL,
    product_id INT NOT NULL,
    review_text TEXT NULL,
    rating INT NOT NULL,
    review_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (customer_id) REFERENCES customers(customer_id),
    FOREIGN KEY (product_id) REFERENCES products(product_id),
    CONSTRAINT unique_review UNIQUE (customer_id, product_id, review_date)
);

INSERT INTO reviews (customer_id, product_id, review_text, rating) VALUES
(1, 1, 'Great product!', 5),
(2, 2, '', 3),
(3, 3, 'Excellent value for money.', 4),
(4, 4, 'Too expensive for its features.', 2),
(5, 5, 'Average product, nothing special.', 3);
