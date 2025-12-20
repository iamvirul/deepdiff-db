CREATE TABLE departments (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255)
);

CREATE TABLE employees (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    department_id INT,
    start_date DATE,
    FOREIGN KEY (department_id) REFERENCES departments(id)
);

INSERT INTO departments (name, location) VALUES
('Engineering', 'New York'),
('Sales', 'London'),
('Marketing', 'Paris'),
('HR', 'New York');

INSERT INTO employees (name, email, department_id, start_date) VALUES
('John Doe', 'john.doe@example.com', 3, '2023-01-15'),
('Jane Smith', 'jane.smith@example.com', 2, '2023-02-01'),
('Peter Jones', 'peter.jones@example.com', 1, '2023-01-20'),
('Alice Wonderland', 'alice.wonderland@example.com', 4, '2023-03-01');
