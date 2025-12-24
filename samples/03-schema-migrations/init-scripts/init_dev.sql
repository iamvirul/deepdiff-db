-- Development Database Schema
-- This represents the desired schema with new changes

CREATE TABLE departments (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255),           -- NEW: Department location
    budget DECIMAL(12,2),             -- NEW: Department budget
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE employees (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    department_id INT,
    start_date DATE,                  -- NEW: Employee start date
    salary DECIMAL(10,2),             -- NEW: Employee salary
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE SET NULL
);

-- NEW TABLE: Modern projects table replacing old_projects
CREATE TABLE projects (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    department_id INT,
    start_date DATE,
    end_date DATE,
    status VARCHAR(50) DEFAULT 'planning',
    budget DECIMAL(12,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE SET NULL
);

-- Sample data (matching prod data structure for existing columns)
INSERT INTO departments (name, location, budget) VALUES
('Engineering', 'New York', 500000.00),
('Sales', 'London', 250000.00),
('Marketing', 'Paris', 150000.00);

INSERT INTO employees (name, email, department_id, start_date, salary) VALUES
('John Doe', 'john.doe@example.com', 1, '2022-01-15', 95000.00),
('Jane Smith', 'jane.smith@example.com', 2, '2022-03-20', 85000.00),
('Peter Jones', 'peter.jones@example.com', 1, '2021-11-10', 105000.00),
('Alice Brown', 'alice.brown@example.com', 3, '2023-02-01', 75000.00);

INSERT INTO projects (name, description, department_id, start_date, status, budget) VALUES
('API Redesign', 'Modernize REST API architecture', 1, '2024-01-01', 'in_progress', 75000.00),
('Marketing Campaign Q1', 'Spring product launch', 3, '2024-02-15', 'planning', 50000.00),
('Sales Dashboard', 'Real-time sales analytics', 2, '2024-03-01', 'planning', 35000.00);
