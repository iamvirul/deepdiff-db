-- Oracle initialization (Target / Development)
-- Default Oracle table and column identifiers are folded to UPPERCASE.

BEGIN
   EXECUTE IMMEDIATE 'DROP TABLE UM_USER_ATTRIBUTE CASCADE CONSTRAINTS';
EXCEPTION
   WHEN OTHERS THEN IF SQLCODE != -942 THEN RAISE; END IF;
END;
/

BEGIN
   EXECUTE IMMEDIATE 'DROP TABLE UM_USER CASCADE CONSTRAINTS';
EXCEPTION
   WHEN OTHERS THEN IF SQLCODE != -942 THEN RAISE; END IF;
END;
/

CREATE TABLE UM_USER (
    ID NUMBER(10) PRIMARY KEY,
    USERNAME VARCHAR2(100) NOT NULL,
    EMAIL VARCHAR2(255) NOT NULL,
    STATUS VARCHAR2(20) DEFAULT 'ACTIVE' NOT NULL,
    CREATED_AT TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE TABLE UM_USER_ATTRIBUTE (
    ID NUMBER(10) PRIMARY KEY,
    USER_ID NUMBER(10) NOT NULL REFERENCES UM_USER(ID),
    ATTR_NAME VARCHAR2(100) NOT NULL,
    ATTR_VALUE VARCHAR2(255) NOT NULL
);

-- Seed development users with intentional drift:
-- Row 1, 2, 3: identical
-- Row 4: updated email ('dan+updated@corp.internal' vs 'dan@corp.internal')
-- Row 5, 6: missing (deleted or omitted during partial sync)
-- Row 7: new user added in target
INSERT INTO UM_USER (ID, USERNAME, EMAIL, STATUS) VALUES (1, 'alice', 'alice@corp.internal', 'ACTIVE');
INSERT INTO UM_USER (ID, USERNAME, EMAIL, STATUS) VALUES (2, 'bob', 'bob@corp.internal', 'ACTIVE');
INSERT INTO UM_USER (ID, USERNAME, EMAIL, STATUS) VALUES (3, 'carol', 'carol@corp.internal', 'INACTIVE');
INSERT INTO UM_USER (ID, USERNAME, EMAIL, STATUS) VALUES (4, 'dan', 'dan+updated@corp.internal', 'ACTIVE');
INSERT INTO UM_USER (ID, USERNAME, EMAIL, STATUS) VALUES (7, 'grace', 'grace@corp.internal', 'ACTIVE');

-- Seed user attributes with drift:
-- Attr 104: updated attr_value ('Accounting' vs 'Finance')
-- Attr 106, 107: missing
-- Attr 108: new attribute for grace
INSERT INTO UM_USER_ATTRIBUTE (ID, USER_ID, ATTR_NAME, ATTR_VALUE) VALUES (101, 1, 'department', 'Engineering');
INSERT INTO UM_USER_ATTRIBUTE (ID, USER_ID, ATTR_NAME, ATTR_VALUE) VALUES (102, 1, 'role', 'Lead Developer');
INSERT INTO UM_USER_ATTRIBUTE (ID, USER_ID, ATTR_NAME, ATTR_VALUE) VALUES (103, 2, 'department', 'DevOps');
INSERT INTO UM_USER_ATTRIBUTE (ID, USER_ID, ATTR_NAME, ATTR_VALUE) VALUES (104, 3, 'department', 'Accounting');
INSERT INTO UM_USER_ATTRIBUTE (ID, USER_ID, ATTR_NAME, ATTR_VALUE) VALUES (105, 4, 'department', 'Security');
INSERT INTO UM_USER_ATTRIBUTE (ID, USER_ID, ATTR_NAME, ATTR_VALUE) VALUES (108, 7, 'department', 'Legal');

COMMIT;
