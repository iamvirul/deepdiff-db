-- Development database schema for the Oracle support sample.
-- This mirrors prod with deliberate drift to demonstrate what DeepDiff DB detects.
--
-- Schema drift vs production:
--   1. CUSTOMERS  — new column PHONE VARCHAR2(30) NULL added
--   2. PRODUCTS   — CATEGORY column changed to NOT NULL
--                 — new index IX_PRODUCTS_CATEGORY added
--   3. ORDERS     — NOTES column added; STATUS default changed ('pending' → 'new')
--
-- Data drift vs production:
--   - PRODUCTS  : price updated for GADGET-001 ($49.99 → $54.99)
--   - ORDERS    : order 3 status changed; 2 new dev orders added
--   - CUSTOMERS : new customer Frank Brown added
--
-- Run this as SYSTEM against the oracle-dev container (port 1522, service XEPDB1).
--
-- Usage:
--   sqlplus system/OraclePass1!@localhost:1522/XEPDB1 @samples/16-oracle-support/init-scripts/02-dev-schema.sql
-- Or via make:
--   make seed

-- ─────────────────────────────────────────────────────────────────────────────
--  Create application user (idempotent)
-- ─────────────────────────────────────────────────────────────────────────────
DECLARE
  v_count NUMBER;
BEGIN
  SELECT COUNT(*) INTO v_count FROM all_users WHERE username = 'APP_USER';
  IF v_count = 0 THEN
    EXECUTE IMMEDIATE 'CREATE USER app_user IDENTIFIED BY "AppPass123"';
  END IF;
END;
/

GRANT CONNECT, RESOURCE, CREATE SESSION TO app_user;
GRANT UNLIMITED TABLESPACE TO app_user;

-- ─────────────────────────────────────────────────────────────────────────────
--  Drop existing tables (reverse FK order)
-- ─────────────────────────────────────────────────────────────────────────────
DECLARE
  PROCEDURE drop_if_exists(p_table VARCHAR2) IS
  BEGIN
    EXECUTE IMMEDIATE 'DROP TABLE app_user.' || p_table || ' CASCADE CONSTRAINTS PURGE';
  EXCEPTION
    WHEN OTHERS THEN
      IF SQLCODE != -942 THEN RAISE; END IF;
  END;
BEGIN
  drop_if_exists('ORDERS');
  drop_if_exists('PRODUCTS');
  drop_if_exists('CUSTOMERS');
END;
/

-- ─────────────────────────────────────────────────────────────────────────────
--  customers  (drift: +PHONE column)
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE app_user.CUSTOMERS (
    ID         NUMBER        GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    NAME       VARCHAR2(255) NOT NULL,
    EMAIL      VARCHAR2(255) NOT NULL,
    COUNTRY    VARCHAR2(100) DEFAULT 'US' NOT NULL,
    PHONE      VARCHAR2(30)  NULL,                   -- new in dev
    CREATED_AT TIMESTAMP     DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT UQ_CUSTOMERS_EMAIL UNIQUE (EMAIL)
);

CREATE INDEX IX_CUSTOMERS_COUNTRY ON app_user.CUSTOMERS (COUNTRY);

-- ─────────────────────────────────────────────────────────────────────────────
--  products  (drift: CATEGORY NOT NULL; new index)
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE app_user.PRODUCTS (
    ID       NUMBER         GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    SKU      VARCHAR2(100)  NOT NULL,
    NAME     VARCHAR2(255)  NOT NULL,
    PRICE    NUMBER(10, 2)  NOT NULL,
    STOCK    NUMBER(10)     DEFAULT 0 NOT NULL,
    CATEGORY VARCHAR2(100)  NOT NULL,                -- NULL → NOT NULL in dev
    CONSTRAINT UQ_PRODUCTS_SKU UNIQUE (SKU)
);

CREATE INDEX IX_PRODUCTS_CATEGORY ON app_user.PRODUCTS (CATEGORY);  -- new index in dev

-- ─────────────────────────────────────────────────────────────────────────────
--  orders  (drift: +NOTES column; STATUS default changed)
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE app_user.ORDERS (
    ID          NUMBER         GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    CUSTOMER_ID NUMBER         NOT NULL,
    PRODUCT_ID  NUMBER         NOT NULL,
    QUANTITY    NUMBER(10)     DEFAULT 1 NOT NULL,
    TOTAL       NUMBER(10, 2)  NOT NULL,
    STATUS      VARCHAR2(50)   DEFAULT 'new' NOT NULL,  -- default changed: 'pending' → 'new'
    NOTES       VARCHAR2(500)  NULL,                    -- new column in dev
    ORDERED_AT  TIMESTAMP      DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT FK_ORDERS_CUSTOMER FOREIGN KEY (CUSTOMER_ID) REFERENCES app_user.CUSTOMERS (ID),
    CONSTRAINT FK_ORDERS_PRODUCT  FOREIGN KEY (PRODUCT_ID)  REFERENCES app_user.PRODUCTS  (ID)
);

CREATE INDEX IX_ORDERS_CUSTOMER ON app_user.ORDERS (CUSTOMER_ID);
CREATE INDEX IX_ORDERS_PRODUCT  ON app_user.ORDERS (PRODUCT_ID);
CREATE INDEX IX_ORDERS_STATUS   ON app_user.ORDERS (STATUS);

-- ─────────────────────────────────────────────────────────────────────────────
--  Seed data — development (includes drift from production)
-- ─────────────────────────────────────────────────────────────────────────────
INSERT INTO app_user.CUSTOMERS (NAME, EMAIL, COUNTRY, PHONE) VALUES ('Alice Johnson', 'alice@example.com', 'US', '+1-555-0101');
INSERT INTO app_user.CUSTOMERS (NAME, EMAIL, COUNTRY, PHONE) VALUES ('Bob Smith',     'bob@example.com',   'GB', '+44-7700-900001');
INSERT INTO app_user.CUSTOMERS (NAME, EMAIL, COUNTRY, PHONE) VALUES ('Carol White',   'carol@example.com', 'CA', NULL);
INSERT INTO app_user.CUSTOMERS (NAME, EMAIL, COUNTRY, PHONE) VALUES ('David Lee',     'david@example.com', 'AU', '+61-400-000001');
INSERT INTO app_user.CUSTOMERS (NAME, EMAIL, COUNTRY, PHONE) VALUES ('Eva Martinez',  'eva@example.com',   'MX', NULL);
INSERT INTO app_user.CUSTOMERS (NAME, EMAIL, COUNTRY, PHONE) VALUES ('Frank Brown',   'frank@example.com', 'DE', '+49-1512-0000001');  -- new customer

-- GADGET-001 price updated: $49.99 → $54.99 (data drift)
INSERT INTO app_user.PRODUCTS (SKU, NAME, PRICE, STOCK, CATEGORY) VALUES ('WIDGET-001', 'Blue Widget',    9.99,  100, 'Widgets');
INSERT INTO app_user.PRODUCTS (SKU, NAME, PRICE, STOCK, CATEGORY) VALUES ('WIDGET-002', 'Red Widget',    12.49,   75, 'Widgets');
INSERT INTO app_user.PRODUCTS (SKU, NAME, PRICE, STOCK, CATEGORY) VALUES ('GADGET-001', 'Smart Gadget', 54.99,   30, 'Gadgets');  -- price drift
INSERT INTO app_user.PRODUCTS (SKU, NAME, PRICE, STOCK, CATEGORY) VALUES ('GADGET-002', 'Mini Gadget',  24.99,   60, 'Gadgets');
INSERT INTO app_user.PRODUCTS (SKU, NAME, PRICE, STOCK, CATEGORY) VALUES ('TOOL-001',   'Power Tool',   79.99,   20, 'Tools');

-- Order 3: status changed from 'shipped' → 'completed' (data drift)
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS, NOTES) VALUES (1, 1, 2,  19.98, 'completed', NULL);
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS, NOTES) VALUES (1, 3, 1,  49.99, 'completed', NULL);
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS, NOTES) VALUES (2, 2, 3,  37.47, 'completed', 'Upgraded from shipped');  -- status drift
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS, NOTES) VALUES (3, 4, 1,  24.99, 'pending',   NULL);
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS, NOTES) VALUES (4, 5, 1,  79.99, 'completed', NULL);
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS, NOTES) VALUES (5, 1, 5,  49.95, 'pending',   NULL);
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS, NOTES) VALUES (2, 3, 2,  99.98, 'completed', NULL);
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS, NOTES) VALUES (3, 2, 1,  12.49, 'shipped',   NULL);
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS, NOTES) VALUES (1, 2, 1,  12.49, 'new',       'Dev order 1');  -- new order in dev
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS, NOTES) VALUES (6, 4, 2,  49.98, 'new',       'Dev order 2');  -- new order by new customer

COMMIT;
