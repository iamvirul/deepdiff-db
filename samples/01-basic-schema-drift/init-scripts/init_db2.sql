CREATE TABLE users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    email VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    country VARCHAR(50)
);

INSERT INTO users (name, email, country) VALUES
('John Doe', 'john.doe@email.com', 'USA'),
('Jane Smith', 'jane.smith@example.com', 'Canada'),
('Peter Jones', 'peter.jones@example.com', 'UK'),
('Alice Wonderland', 'alice.wonderland@example.com', 'USA');
