-- Oracle Test Data: Cross-Year Transactions (Q4 2023 - Q1 2024)
-- Creates transactions table with test data for integration tests
-- Note: This script is executed against Oracle XE (XEPDB1 service)

-- Drop table if exists
BEGIN
    EXECUTE IMMEDIATE 'DROP TABLE transactions';
EXCEPTION
    WHEN OTHERS THEN
        IF SQLCODE != -942 THEN
            RAISE;
        END IF;
END;
/

-- Create transactions table
CREATE TABLE transactions (
    id RAW(16) DEFAULT SYS_GUID() PRIMARY KEY,
    account_id VARCHAR2(36) NOT NULL,
    amount NUMBER(15,2) NOT NULL,
    currency VARCHAR2(3) NOT NULL,
    type VARCHAR2(10) NOT NULL,
    description VARCHAR2(255),
    category VARCHAR2(50),
    status VARCHAR2(20) NOT NULL,
    created_at TIMESTAMP NOT NULL
);

-- Create indexes
CREATE INDEX idx_trans_account_id ON transactions(account_id);
CREATE INDEX idx_trans_created_at ON transactions(created_at);
CREATE INDEX idx_trans_category ON transactions(category);

-- Insert Cross-Year test data (Q4 2023 - Q1 2024)
-- November 2023
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 1500.00, 'USD', 'credit', 'Salary November 2023', 'salary', 'completed', TIMESTAMP '2023-11-05 09:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 180.00, 'USD', 'debit', 'Thanksgiving Shopping', 'groceries', 'completed', TIMESTAMP '2023-11-22 14:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('22222222-2222-2222-2222-222222222222', 2100.00, 'USD', 'credit', 'Freelance Work', 'salary', 'completed', TIMESTAMP '2023-11-10 11:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('33333333-3333-3333-3333-333333333333', 3500.00, 'USD', 'credit', 'Monthly Salary', 'salary', 'completed', TIMESTAMP '2023-11-01 09:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('33333333-3333-3333-3333-333333333333', 120.00, 'USD', 'debit', 'Heating Bill', 'utilities', 'completed', TIMESTAMP '2023-11-15 10:00:00');

-- December 2023
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 1500.00, 'USD', 'credit', 'Salary December 2023', 'salary', 'completed', TIMESTAMP '2023-12-05 09:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 450.00, 'USD', 'debit', 'Holiday Gifts', 'groceries', 'completed', TIMESTAMP '2023-12-20 15:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('22222222-2222-2222-2222-222222222222', 3500.00, 'USD', 'credit', 'Year-End Bonus', 'salary', 'completed', TIMESTAMP '2023-12-28 16:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('22222222-2222-2222-2222-222222222222', 280.00, 'EUR', 'debit', 'Holiday Trip', 'travel', 'completed', TIMESTAMP '2023-12-24 12:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('33333333-3333-3333-3333-333333333333', 3500.00, 'USD', 'credit', 'Monthly Salary', 'salary', 'completed', TIMESTAMP '2023-12-01 09:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('33333333-3333-3333-3333-333333333333', 200.00, 'USD', 'debit', 'New Year Party', 'entertainment', 'completed', TIMESTAMP '2023-12-31 20:00:00');

-- January 2024
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 1500.00, 'USD', 'credit', 'Salary January 2024', 'salary', 'completed', TIMESTAMP '2024-01-05 09:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 95.00, 'USD', 'debit', 'Gym Membership', 'entertainment', 'completed', TIMESTAMP '2024-01-10 08:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('22222222-2222-2222-2222-222222222222', 1900.00, 'USD', 'credit', 'New Year Project', 'salary', 'completed', TIMESTAMP '2024-01-12 14:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('22222222-2222-2222-2222-222222222222', 75.00, 'USD', 'debit', 'Online Course', 'entertainment', 'completed', TIMESTAMP '2024-01-20 19:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('33333333-3333-3333-3333-333333333333', 3500.00, 'USD', 'credit', 'Monthly Salary', 'salary', 'completed', TIMESTAMP '2024-01-01 09:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('33333333-3333-3333-3333-333333333333', 140.00, 'USD', 'debit', 'Winter Jacket', 'groceries', 'completed', TIMESTAMP '2024-01-15 16:00:00');

-- February 2024
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 1500.00, 'USD', 'credit', 'Salary February 2024', 'salary', 'completed', TIMESTAMP '2024-02-05 09:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 160.00, 'USD', 'debit', 'Valentine Dinner', 'entertainment', 'completed', TIMESTAMP '2024-02-14 19:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('22222222-2222-2222-2222-222222222222', 2300.00, 'USD', 'credit', 'Contract Payment', 'salary', 'completed', TIMESTAMP '2024-02-08 11:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('33333333-3333-3333-3333-333333333333', 3500.00, 'USD', 'credit', 'Monthly Salary', 'salary', 'completed', TIMESTAMP '2024-02-01 09:00:00');

-- Pending transactions
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 55.00, 'USD', 'debit', 'Pending Book Order', 'entertainment', 'pending', TIMESTAMP '2024-02-28 14:00:00');
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('33333333-3333-3333-3333-333333333333', 90.00, 'USD', 'debit', 'Pending Subscription', 'utilities', 'pending', TIMESTAMP '2024-02-29 09:00:00');

-- =============================================================================
-- MULTI-SCHEMA TEST DATA
-- =============================================================================

-- Drop existing multi-schema tables if they exist
BEGIN
    EXECUTE IMMEDIATE 'DROP TABLE billing_subscriptions';
EXCEPTION
    WHEN OTHERS THEN
        IF SQLCODE != -942 THEN
            RAISE;
        END IF;
END;
/

BEGIN
    EXECUTE IMMEDIATE 'DROP TABLE audit_events';
EXCEPTION
    WHEN OTHERS THEN
        IF SQLCODE != -942 THEN
            RAISE;
        END IF;
END;
/

-- Create billing_subscriptions table (simulating BILLING schema)
CREATE TABLE billing_subscriptions (
    id RAW(16) DEFAULT SYS_GUID() PRIMARY KEY,
    account_id VARCHAR2(36) NOT NULL,
    plan_name VARCHAR2(50) NOT NULL,
    monthly_amount NUMBER(15,2) NOT NULL,
    currency VARCHAR2(3) NOT NULL,
    status VARCHAR2(20) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE,
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_bill_sub_account ON billing_subscriptions(account_id);
CREATE INDEX idx_bill_sub_status ON billing_subscriptions(status);

-- Insert billing_subscriptions test data (8 records)
INSERT INTO billing_subscriptions (account_id, plan_name, monthly_amount, currency, status, start_date, end_date, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 'Basic', 9.99, 'USD', 'active', DATE '2023-11-01', NULL, TIMESTAMP '2023-11-01 10:00:00');
INSERT INTO billing_subscriptions (account_id, plan_name, monthly_amount, currency, status, start_date, end_date, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 'Premium', 29.99, 'USD', 'active', DATE '2024-01-01', NULL, TIMESTAMP '2024-01-01 09:00:00');
INSERT INTO billing_subscriptions (account_id, plan_name, monthly_amount, currency, status, start_date, end_date, created_at) VALUES
('22222222-2222-2222-2222-222222222222', 'Enterprise', 99.99, 'USD', 'active', DATE '2023-12-01', NULL, TIMESTAMP '2023-12-01 14:00:00');
INSERT INTO billing_subscriptions (account_id, plan_name, monthly_amount, currency, status, start_date, end_date, created_at) VALUES
('22222222-2222-2222-2222-222222222222', 'Basic', 9.99, 'EUR', 'cancelled', DATE '2023-06-01', DATE '2023-11-30', TIMESTAMP '2023-06-01 11:00:00');
INSERT INTO billing_subscriptions (account_id, plan_name, monthly_amount, currency, status, start_date, end_date, created_at) VALUES
('33333333-3333-3333-3333-333333333333', 'Premium', 29.99, 'USD', 'active', DATE '2024-01-15', NULL, TIMESTAMP '2024-01-15 16:00:00');
INSERT INTO billing_subscriptions (account_id, plan_name, monthly_amount, currency, status, start_date, end_date, created_at) VALUES
('33333333-3333-3333-3333-333333333333', 'Basic', 9.99, 'GBP', 'suspended', DATE '2023-09-01', NULL, TIMESTAMP '2023-09-01 08:00:00');
INSERT INTO billing_subscriptions (account_id, plan_name, monthly_amount, currency, status, start_date, end_date, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 'Trial', 0.00, 'USD', 'expired', DATE '2023-10-01', DATE '2023-10-31', TIMESTAMP '2023-10-01 12:00:00');
INSERT INTO billing_subscriptions (account_id, plan_name, monthly_amount, currency, status, start_date, end_date, created_at) VALUES
('22222222-2222-2222-2222-222222222222', 'Premium', 29.99, 'USD', 'pending', DATE '2024-03-01', NULL, TIMESTAMP '2024-02-25 10:00:00');

-- Create audit_events table (simulating AUDIT schema)
CREATE TABLE audit_events (
    id RAW(16) DEFAULT SYS_GUID() PRIMARY KEY,
    account_id VARCHAR2(36) NOT NULL,
    event_type VARCHAR2(50) NOT NULL,
    entity_type VARCHAR2(50) NOT NULL,
    entity_id VARCHAR2(36) NOT NULL,
    old_value VARCHAR2(1000),
    new_value VARCHAR2(1000),
    performed_by VARCHAR2(100) NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_audit_account ON audit_events(account_id);
CREATE INDEX idx_audit_type ON audit_events(event_type);
CREATE INDEX idx_audit_created ON audit_events(created_at);

-- Insert audit_events test data (10 records)
INSERT INTO audit_events (account_id, event_type, entity_type, entity_id, old_value, new_value, performed_by, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 'CREATE', 'subscription', 'sub-001', NULL, 'Basic plan created', 'system', TIMESTAMP '2023-11-01 10:00:00');
INSERT INTO audit_events (account_id, event_type, entity_type, entity_id, old_value, new_value, performed_by, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 'UPDATE', 'subscription', 'sub-001', 'Basic', 'Premium', 'user@example.com', TIMESTAMP '2024-01-01 09:00:00');
INSERT INTO audit_events (account_id, event_type, entity_type, entity_id, old_value, new_value, performed_by, created_at) VALUES
('22222222-2222-2222-2222-222222222222', 'CREATE', 'subscription', 'sub-002', NULL, 'Enterprise plan created', 'system', TIMESTAMP '2023-12-01 14:00:00');
INSERT INTO audit_events (account_id, event_type, entity_type, entity_id, old_value, new_value, performed_by, created_at) VALUES
('22222222-2222-2222-2222-222222222222', 'DELETE', 'subscription', 'sub-003', 'Basic', NULL, 'admin@example.com', TIMESTAMP '2023-11-30 23:59:00');
INSERT INTO audit_events (account_id, event_type, entity_type, entity_id, old_value, new_value, performed_by, created_at) VALUES
('33333333-3333-3333-3333-333333333333', 'CREATE', 'subscription', 'sub-004', NULL, 'Premium plan created', 'system', TIMESTAMP '2024-01-15 16:00:00');
INSERT INTO audit_events (account_id, event_type, entity_type, entity_id, old_value, new_value, performed_by, created_at) VALUES
('33333333-3333-3333-3333-333333333333', 'UPDATE', 'subscription', 'sub-005', 'active', 'suspended', 'admin@example.com', TIMESTAMP '2024-02-01 09:00:00');
INSERT INTO audit_events (account_id, event_type, entity_type, entity_id, old_value, new_value, performed_by, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 'LOGIN', 'user', 'user-001', NULL, 'Successful login', 'user@example.com', TIMESTAMP '2024-01-05 08:30:00');
INSERT INTO audit_events (account_id, event_type, entity_type, entity_id, old_value, new_value, performed_by, created_at) VALUES
('22222222-2222-2222-2222-222222222222', 'LOGIN', 'user', 'user-002', NULL, 'Successful login', 'admin@example.com', TIMESTAMP '2024-01-10 09:15:00');
INSERT INTO audit_events (account_id, event_type, entity_type, entity_id, old_value, new_value, performed_by, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 'PAYMENT', 'invoice', 'inv-001', 'pending', 'paid', 'payment_system', TIMESTAMP '2024-01-05 10:00:00');
INSERT INTO audit_events (account_id, event_type, entity_type, entity_id, old_value, new_value, performed_by, created_at) VALUES
('33333333-3333-3333-3333-333333333333', 'PAYMENT', 'invoice', 'inv-002', 'pending', 'paid', 'payment_system', TIMESTAMP '2024-02-01 11:00:00');

COMMIT;
