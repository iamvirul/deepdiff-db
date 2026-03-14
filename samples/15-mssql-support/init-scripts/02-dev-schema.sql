-- Development database schema for the MSSQL support sample.
-- This mirrors prod with deliberate drift to demonstrate what DeepDiff DB detects.
--
-- Schema drift vs production:
--   1. customers  — new column `phone NVARCHAR(30) NULL` added
--   2. products   — `category` column is NOT NULL (nullable changed)
--                 — new index IX_products_category added
--   3. orders     — `notes` column added; `status` default changed
--
-- Data drift vs production:
--   - products  : price updated for GADGET-001 ($49.99 → $54.99)
--   - orders    : 2 new orders (customer 1 and 5), 1 order status changed
--   - customers : new customer added (Frank Brown)
--
-- Run this against the mssql-dev container (port 1434).
--
-- Usage:
--   sqlcmd -S localhost,1434 -U sa -P 'StrongP@ss1word!' -C -i samples/15-mssql-support/init-scripts/02-dev-schema.sql
-- Or via make:
--   make seed

USE master;
GO

IF NOT EXISTS (SELECT name FROM sys.databases WHERE name = N'dev_db')
BEGIN
    CREATE DATABASE dev_db;
END
GO

USE dev_db;
GO

-- ─────────────────────────────────────────────────────────
--  customers  (drift: +phone column)
-- ─────────────────────────────────────────────────────────
IF OBJECT_ID('dbo.customers', 'U') IS NOT NULL
    DROP TABLE dbo.customers;
GO

CREATE TABLE dbo.customers (
    id         INT           NOT NULL IDENTITY(1,1),
    name       NVARCHAR(255) NOT NULL,
    email      NVARCHAR(255) NOT NULL,
    country    NVARCHAR(100) NOT NULL DEFAULT 'US',
    phone      NVARCHAR(30)  NULL,                   -- new in dev
    created_at DATETIME2     NOT NULL DEFAULT SYSUTCDATETIME(),
    CONSTRAINT PK_customers PRIMARY KEY (id),
    CONSTRAINT UQ_customers_email UNIQUE (email)
);
GO

CREATE INDEX IX_customers_country ON dbo.customers (country);
GO

-- ─────────────────────────────────────────────────────────
--  products  (drift: category NOT NULL; new index)
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
    category    NVARCHAR(100)    NOT NULL,            -- nullable → NOT NULL in dev
    CONSTRAINT PK_products PRIMARY KEY (id),
    CONSTRAINT UQ_products_sku UNIQUE (sku)
);
GO

CREATE INDEX IX_products_category ON dbo.products (category); -- new index in dev
GO

-- ─────────────────────────────────────────────────────────
--  orders  (drift: +notes column; status default changed)
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
    status      NVARCHAR(50)   NOT NULL DEFAULT 'new',  -- default changed: 'pending' → 'new'
    notes       NVARCHAR(500)  NULL,                    -- new column in dev
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
--  Seed data — development (includes drift from production)
-- ─────────────────────────────────────────────────────────
INSERT INTO dbo.customers (name, email, country, phone) VALUES
    ('Alice Johnson', 'alice@example.com', 'US', '+1-555-0101'),
    ('Bob Smith',     'bob@example.com',   'GB', '+44-7700-900001'),
    ('Carol White',   'carol@example.com', 'CA', NULL),
    ('David Lee',     'david@example.com', 'AU', '+61-400-000001'),
    ('Eva Martinez',  'eva@example.com',   'MX', NULL),
    ('Frank Brown',   'frank@example.com', 'DE', '+49-1512-0000001'); -- new customer
GO

-- GADGET-001 price updated: $49.99 → $54.99 (data drift)
INSERT INTO dbo.products (sku, name, price, stock, category) VALUES
    ('WIDGET-001', 'Blue Widget',    9.99,  100, 'Widgets'),
    ('WIDGET-002', 'Red Widget',    12.49,   75, 'Widgets'),
    ('GADGET-001', 'Smart Gadget', 54.99,   30, 'Gadgets'),  -- price drift
    ('GADGET-002', 'Mini Gadget',  24.99,   60, 'Gadgets'),
    ('TOOL-001',   'Power Tool',   79.99,   20, 'Tools');
GO

-- Existing orders match prod (status values updated where needed)
-- Order 3: status changed from 'shipped' → 'completed' (data drift)
INSERT INTO dbo.orders (customer_id, product_id, quantity, total, status, notes) VALUES
    (1, 1, 2,  19.98, 'completed', NULL),
    (1, 3, 1,  49.99, 'completed', NULL),
    (2, 2, 3,  37.47, 'completed', 'Upgraded from shipped'), -- status drift + note
    (3, 4, 1,  24.99, 'pending',   NULL),
    (4, 5, 1,  79.99, 'completed', NULL),
    (5, 1, 5,  49.95, 'pending',   NULL),
    (2, 3, 2,  99.98, 'completed', NULL),
    (3, 2, 1,  12.49, 'shipped',   NULL),
    (1, 2, 1,  12.49, 'new',       'Dev order 1'), -- new order in dev
    (6, 4, 2,  49.98, 'new',       'Dev order 2'); -- new order by new customer
GO
