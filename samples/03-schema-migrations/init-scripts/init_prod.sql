-- Production Database Schema
-- This represents the current production schema

CREATE TABLE departments (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE employees (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    department_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE SET NULL
);

CREATE TABLE old_projects (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50)
);

-- Sample data
INSERT INTO departments (name) VALUES
('Engineering'),
('Sales'),
('Marketing');

INSERT INTO employees (name, email, department_id) VALUES
('John Doe', 'john.doe@example.com', 1),
('Jane Smith', 'jane.smith@example.com', 2),
('Peter Jones', 'peter.jones@example.com', 1),
('Alice Brown', 'alice.brown@example.com', 3);

INSERT INTO old_projects (name, status) VALUES
('Project Alpha', 'completed'),
('Project Beta', 'in_progress');
