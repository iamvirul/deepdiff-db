CREATE TABLE departments (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

CREATE TABLE employees (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    department_id INT,
    FOREIGN KEY (department_id) REFERENCES departments(id)
);

INSERT INTO departments (name) VALUES
('Engineering'),
('Sales'),
('Marketing');

INSERT INTO employees (name, email, department_id) VALUES
('John Doe', 'john.doe@example.com', 1),
('Jane Smith', 'jane.smith@example.com', 2),
('Peter Jones', 'peter.jones@example.com', 1);
