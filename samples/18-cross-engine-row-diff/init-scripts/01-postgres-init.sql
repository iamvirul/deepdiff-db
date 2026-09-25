-- PostgreSQL initialization (Source / Production)
-- Default PostgreSQL table and column identifiers are folded to lowercase.

DROP TABLE IF EXISTS um_user_attribute;
DROP TABLE IF EXISTS um_user;

CREATE TABLE um_user (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE um_user_attribute (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES um_user(id) ON DELETE CASCADE,
    attr_name VARCHAR(100) NOT NULL,
    attr_value VARCHAR(255) NOT NULL
);

-- Seed production users
INSERT INTO um_user (id, username, email, status) VALUES
(1, 'alice', 'alice@corp.internal', 'ACTIVE'),
(2, 'bob', 'bob@corp.internal', 'ACTIVE'),
(3, 'carol', 'carol@corp.internal', 'INACTIVE'),
(4, 'dan', 'dan@corp.internal', 'ACTIVE'),
(5, 'eve', 'eve@corp.internal', 'ACTIVE'),
(6, 'frank', 'frank@corp.internal', 'ACTIVE');

-- Seed user attributes
INSERT INTO um_user_attribute (id, user_id, attr_name, attr_value) VALUES
(101, 1, 'department', 'Engineering'),
(102, 1, 'role', 'Lead Developer'),
(103, 2, 'department', 'DevOps'),
(104, 3, 'department', 'Finance'),
(105, 4, 'department', 'Security'),
(106, 5, 'department', 'HR'),
(107, 6, 'department', 'Sales');
