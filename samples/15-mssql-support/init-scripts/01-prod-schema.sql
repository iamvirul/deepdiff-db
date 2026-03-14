-- Production database schema for the MSSQL support sample.
-- Run this against the mssql-prod container.
--
-- Usage (from project root):
--   sqlcmd -S localhost,1433 -U sa -P 'StrongP@ss1word!' -C -i samples/15-mssql-support/init-scripts/01-prod-schema.sql
-- Or via make:
--   make seed

USE master;
GO

-- Create the database if it does not already exist.
IF NOT EXISTS (SELECT name FROM sys.databases WHERE name = N'prod_db')
BEGIN
    CREATE DATABASE prod_db;
END
GO

USE prod_db;
GO

-- ─────────────────────────────────────────────────────────
--  customers
-- ─────────────────────────────────────────────────────────
IF OBJECT_ID('dbo.customers', 'U') IS NOT NULL
    DROP TABLE dbo.customers;
GO

CREATE TABLE dbo.customers (
    id         INT           NOT NULL IDENTITY(1,1),
    name       NVARCHAR(255) NOT NULL,
    email      NVARCHAR(255) NOT NULL,
    country    NVARCHAR(100) NOT NULL DEFAULT 'US',
    created_at DATETIME2     NOT NULL DEFAULT SYSUTCDATETIME(),
    CONSTRAINT PK_customers PRIMARY KEY (id),
    CONSTRAINT UQ_customers_email UNIQUE (email)
);
GO

CREATE INDEX IX_customers_country ON dbo.customers (country);
GO

-- ─────────────────────────────────────────────────────────
--  products
-- ─────────────────────────────────────────────────────────
IF OBJECT_ID('dbo.products', 'U') IS NOT NULL
    DROP TABLE dbo.products;
GO

CREATE TABLE dbo.products (
    id          INT              NOT NULL IDENTITY(1,1),
    sku         NVARCHAR(100)    NOT NULL,
    name        NVARCHAR(255)    NOT NULL,
    price       DECIMAL(10, 2)   NOT NULL,
    stock       INT              NOT NULL DEFAULT 0,
    category    NVARCHAR(100)    NULL,
    CONSTRAINT PK_products PRIMARY KEY (id),
    CONSTRAINT UQ_products_sku UNIQUE (sku)
);
GO

-- ─────────────────────────────────────────────────────────
--  orders
-- ─────────────────────────────────────────────────────────
IF OBJECT_ID('dbo.orders', 'U') IS NOT NULL
    DROP TABLE dbo.orders;
GO

CREATE TABLE dbo.orders (
    id          INT            NOT NULL IDENTITY(1,1),
    customer_id INT            NOT NULL,
    product_id  INT            NOT NULL,
    quantity    INT            NOT NULL DEFAULT 1,
    total       DECIMAL(10, 2) NOT NULL,
    status      NVARCHAR(50)   NOT NULL DEFAULT 'pending',
    ordered_at  DATETIME2      NOT NULL DEFAULT SYSUTCDATETIME(),
    CONSTRAINT PK_orders PRIMARY KEY (id),
    CONSTRAINT FK_orders_customer FOREIGN KEY (customer_id) REFERENCES dbo.customers (id),
    CONSTRAINT FK_orders_product  FOREIGN KEY (product_id)  REFERENCES dbo.products  (id)
);
GO

CREATE INDEX IX_orders_customer ON dbo.orders (customer_id);
CREATE INDEX IX_orders_product  ON dbo.orders (product_id);
CREATE INDEX IX_orders_status   ON dbo.orders (status);
GO

-- ─────────────────────────────────────────────────────────
--  Seed data — production baseline
-- ─────────────────────────────────────────────────────────
INSERT INTO dbo.customers (name, email, country) VALUES
    ('Alice Johnson', 'alice@example.com', 'US'),
    ('Bob Smith',     'bob@example.com',   'GB'),
    ('Carol White',   'carol@example.com', 'CA'),
    ('David Lee',     'david@example.com', 'AU'),
    ('Eva Martinez',  'eva@example.com',   'MX');
GO

INSERT INTO dbo.products (sku, name, price, stock, category) VALUES
    ('WIDGET-001', 'Blue Widget',    9.99,  100, 'Widgets'),
    ('WIDGET-002', 'Red Widget',    12.49,   75, 'Widgets'),
    ('GADGET-001', 'Smart Gadget', 49.99,   30, 'Gadgets'),
    ('GADGET-002', 'Mini Gadget',  24.99,   60, 'Gadgets'),
    ('TOOL-001',   'Power Tool',   79.99,   20, 'Tools');
GO

INSERT INTO dbo.orders (customer_id, product_id, quantity, total, status) VALUES
    (1, 1, 2,  19.98, 'completed'),
    (1, 3, 1,  49.99, 'completed'),
    (2, 2, 3,  37.47, 'shipped'),
    (3, 4, 1,  24.99, 'pending'),
    (4, 5, 1,  79.99, 'completed'),
    (5, 1, 5,  49.95, 'pending'),
    (2, 3, 2,  99.98, 'completed'),
    (3, 2, 1,  12.49, 'shipped');
GO
