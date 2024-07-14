CREATE TABLE products (
    product_id INT AUTO_INCREMENT PRIMARY KEY,
    product_name VARCHAR(100) NOT NULL,
    product_description TEXT,
    price DECIMAL(10, 2) NOT NULL
);

INSERT INTO products (product_name, product_description, price) VALUES
('Product A', 'Description for Product A', 29.99),
('Product B', 'Description for Product B', 49.99),
('Product C', 'Description for Product C', 19.99),
('Product D', 'Description for Product D', 99.99),
('iPhone 13 Pro Max', 'Description for Product E', 1059.99);
