-- Production database schema for the Oracle support sample.
-- Run this as SYSTEM against the oracle-prod container (port 1521, service XEPDB1).
--
-- Usage (from project root):
--   sqlplus system/OraclePass1!@localhost:1521/XEPDB1 @samples/16-oracle-support/init-scripts/01-prod-schema.sql
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
--  customers
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE app_user.CUSTOMERS (
    ID         NUMBER        GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    NAME       VARCHAR2(255) NOT NULL,
    EMAIL      VARCHAR2(255) NOT NULL,
    COUNTRY    VARCHAR2(100) DEFAULT 'US' NOT NULL,
    CREATED_AT TIMESTAMP     DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT UQ_CUSTOMERS_EMAIL UNIQUE (EMAIL)
);

CREATE INDEX IX_CUSTOMERS_COUNTRY ON app_user.CUSTOMERS (COUNTRY);

-- ─────────────────────────────────────────────────────────────────────────────
--  products
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE app_user.PRODUCTS (
    ID       NUMBER         GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    SKU      VARCHAR2(100)  NOT NULL,
    NAME     VARCHAR2(255)  NOT NULL,
    PRICE    NUMBER(10, 2)  NOT NULL,
    STOCK    NUMBER(10)     DEFAULT 0 NOT NULL,
    CATEGORY VARCHAR2(100)  NULL,
    CONSTRAINT UQ_PRODUCTS_SKU UNIQUE (SKU)
);

-- ─────────────────────────────────────────────────────────────────────────────
--  orders
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE app_user.ORDERS (
    ID          NUMBER         GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    CUSTOMER_ID NUMBER         NOT NULL,
    PRODUCT_ID  NUMBER         NOT NULL,
    QUANTITY    NUMBER(10)     DEFAULT 1 NOT NULL,
    TOTAL       NUMBER(10, 2)  NOT NULL,
    STATUS      VARCHAR2(50)   DEFAULT 'pending' NOT NULL,
    ORDERED_AT  TIMESTAMP      DEFAULT SYSTIMESTAMP NOT NULL,
    CONSTRAINT FK_ORDERS_CUSTOMER FOREIGN KEY (CUSTOMER_ID) REFERENCES app_user.CUSTOMERS (ID),
    CONSTRAINT FK_ORDERS_PRODUCT  FOREIGN KEY (PRODUCT_ID)  REFERENCES app_user.PRODUCTS  (ID)
);

CREATE INDEX IX_ORDERS_CUSTOMER ON app_user.ORDERS (CUSTOMER_ID);
CREATE INDEX IX_ORDERS_PRODUCT  ON app_user.ORDERS (PRODUCT_ID);
CREATE INDEX IX_ORDERS_STATUS   ON app_user.ORDERS (STATUS);

-- ─────────────────────────────────────────────────────────────────────────────
--  Seed data — production baseline
-- ─────────────────────────────────────────────────────────────────────────────
INSERT INTO app_user.CUSTOMERS (NAME, EMAIL, COUNTRY) VALUES ('Alice Johnson', 'alice@example.com', 'US');
INSERT INTO app_user.CUSTOMERS (NAME, EMAIL, COUNTRY) VALUES ('Bob Smith',     'bob@example.com',   'GB');
INSERT INTO app_user.CUSTOMERS (NAME, EMAIL, COUNTRY) VALUES ('Carol White',   'carol@example.com', 'CA');
INSERT INTO app_user.CUSTOMERS (NAME, EMAIL, COUNTRY) VALUES ('David Lee',     'david@example.com', 'AU');
INSERT INTO app_user.CUSTOMERS (NAME, EMAIL, COUNTRY) VALUES ('Eva Martinez',  'eva@example.com',   'MX');

INSERT INTO app_user.PRODUCTS (SKU, NAME, PRICE, STOCK, CATEGORY) VALUES ('WIDGET-001', 'Blue Widget',    9.99,  100, 'Widgets');
INSERT INTO app_user.PRODUCTS (SKU, NAME, PRICE, STOCK, CATEGORY) VALUES ('WIDGET-002', 'Red Widget',    12.49,   75, 'Widgets');
INSERT INTO app_user.PRODUCTS (SKU, NAME, PRICE, STOCK, CATEGORY) VALUES ('GADGET-001', 'Smart Gadget', 49.99,   30, 'Gadgets');
INSERT INTO app_user.PRODUCTS (SKU, NAME, PRICE, STOCK, CATEGORY) VALUES ('GADGET-002', 'Mini Gadget',  24.99,   60, 'Gadgets');
INSERT INTO app_user.PRODUCTS (SKU, NAME, PRICE, STOCK, CATEGORY) VALUES ('TOOL-001',   'Power Tool',   79.99,   20, 'Tools');

INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS) VALUES (1, 1, 2,  19.98, 'completed');
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS) VALUES (1, 3, 1,  49.99, 'completed');
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS) VALUES (2, 2, 3,  37.47, 'shipped');
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS) VALUES (3, 4, 1,  24.99, 'pending');
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS) VALUES (4, 5, 1,  79.99, 'completed');
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS) VALUES (5, 1, 5,  49.95, 'pending');
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS) VALUES (2, 3, 2,  99.98, 'completed');
INSERT INTO app_user.ORDERS (CUSTOMER_ID, PRODUCT_ID, QUANTITY, TOTAL, STATUS) VALUES (3, 2, 1,  12.49, 'shipped');

COMMIT;
