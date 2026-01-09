-- PostgreSQL Test Fixtures for E2E Integration Tests
-- Contains Q1 2024 (January-March) transaction data

-- Create accounts table
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Insert test accounts
INSERT INTO accounts (id, name, email) VALUES
('11111111-1111-1111-1111-111111111111', 'John Doe', 'john@example.com'),
('22222222-2222-2222-2222-222222222222', 'Jane Smith', 'jane@example.com'),
('33333333-3333-3333-3333-333333333333', 'Bob Johnson', 'bob@example.com');

-- Create transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    type VARCHAR(10) NOT NULL CHECK (type IN ('credit', 'debit')),
    description VARCHAR(500),
    category VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'completed',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_transactions_account_id ON transactions(account_id);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);

-- Test Account IDs (consistent across all databases)
-- Account 1: 11111111-1111-1111-1111-111111111111
-- Account 2: 22222222-2222-2222-2222-222222222222
-- Account 3: 33333333-3333-3333-3333-333333333333

-- Insert Q1 2024 test transactions (30 records)
INSERT INTO transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
-- January 2024
('11111111-1111-1111-1111-111111111111', 1500.00, 'USD', 'credit', 'Salary January', 'salary', 'completed', '2024-01-05 09:00:00+00'),
('11111111-1111-1111-1111-111111111111', 150.00, 'USD', 'debit', 'Grocery Store', 'groceries', 'completed', '2024-01-10 14:30:00+00'),
('11111111-1111-1111-1111-111111111111', 89.99, 'USD', 'debit', 'Electric Bill', 'utilities', 'completed', '2024-01-15 10:00:00+00'),
('22222222-2222-2222-2222-222222222222', 2000.00, 'USD', 'credit', 'Freelance Payment', 'salary', 'completed', '2024-01-08 11:00:00+00'),
('22222222-2222-2222-2222-222222222222', 45.00, 'USD', 'debit', 'Netflix Subscription', 'entertainment', 'completed', '2024-01-12 08:00:00+00'),
('33333333-3333-3333-3333-333333333333', 3500.00, 'USD', 'credit', 'Monthly Salary', 'salary', 'completed', '2024-01-01 09:00:00+00'),
('33333333-3333-3333-3333-333333333333', 200.00, 'EUR', 'debit', 'Flight Booking', 'travel', 'completed', '2024-01-20 16:00:00+00'),
('33333333-3333-3333-3333-333333333333', 75.00, 'USD', 'debit', 'Restaurant', 'entertainment', 'completed', '2024-01-25 19:30:00+00'),
-- February 2024
('11111111-1111-1111-1111-111111111111', 1500.00, 'USD', 'credit', 'Salary February', 'salary', 'completed', '2024-02-05 09:00:00+00'),
('11111111-1111-1111-1111-111111111111', 180.00, 'USD', 'debit', 'Grocery Store', 'groceries', 'completed', '2024-02-12 15:00:00+00'),
('11111111-1111-1111-1111-111111111111', 120.00, 'USD', 'debit', 'Internet Bill', 'utilities', 'completed', '2024-02-18 10:00:00+00'),
('22222222-2222-2222-2222-222222222222', 500.00, 'USD', 'credit', 'Bonus Payment', 'salary', 'completed', '2024-02-10 14:00:00+00'),
('22222222-2222-2222-2222-222222222222', 250.00, 'GBP', 'debit', 'Hotel Booking', 'travel', 'completed', '2024-02-14 12:00:00+00'),
('33333333-3333-3333-3333-333333333333', 3500.00, 'USD', 'credit', 'Monthly Salary', 'salary', 'completed', '2024-02-01 09:00:00+00'),
('33333333-3333-3333-3333-333333333333', 95.00, 'USD', 'debit', 'Gas Station', 'utilities', 'completed', '2024-02-22 17:00:00+00'),
-- March 2024
('11111111-1111-1111-1111-111111111111', 1500.00, 'USD', 'credit', 'Salary March', 'salary', 'completed', '2024-03-05 09:00:00+00'),
('11111111-1111-1111-1111-111111111111', 200.00, 'USD', 'debit', 'Grocery Store', 'groceries', 'completed', '2024-03-08 14:00:00+00'),
('11111111-1111-1111-1111-111111111111', 60.00, 'USD', 'debit', 'Streaming Services', 'entertainment', 'completed', '2024-03-15 10:00:00+00'),
('22222222-2222-2222-2222-222222222222', 2200.00, 'USD', 'credit', 'Freelance Payment', 'salary', 'completed', '2024-03-03 11:00:00+00'),
('22222222-2222-2222-2222-222222222222', 85.00, 'USD', 'debit', 'Phone Bill', 'utilities', 'completed', '2024-03-18 09:00:00+00'),
('22222222-2222-2222-2222-222222222222', 150.00, 'EUR', 'debit', 'Concert Tickets', 'entertainment', 'completed', '2024-03-22 20:00:00+00'),
('33333333-3333-3333-3333-333333333333', 3500.00, 'USD', 'credit', 'Monthly Salary', 'salary', 'completed', '2024-03-01 09:00:00+00'),
('33333333-3333-3333-3333-333333333333', 320.00, 'USD', 'debit', 'Car Insurance', 'utilities', 'completed', '2024-03-10 11:00:00+00'),
('33333333-3333-3333-3333-333333333333', 450.00, 'USD', 'debit', 'Weekend Trip', 'travel', 'completed', '2024-03-28 08:00:00+00'),
-- Additional pending transactions
('11111111-1111-1111-1111-111111111111', 99.00, 'USD', 'debit', 'Pending Order', 'groceries', 'pending', '2024-03-30 10:00:00+00'),
('22222222-2222-2222-2222-222222222222', 150.00, 'USD', 'debit', 'Pending Transfer', 'utilities', 'pending', '2024-03-30 11:00:00+00');

-- =============================================================================
-- MULTI-SCHEMA TEST DATA
-- =============================================================================

-- Create accounting schema
CREATE SCHEMA IF NOT EXISTS accounting;

-- Create accounting.invoices table
CREATE TABLE accounting.invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL,
    invoice_number VARCHAR(50) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    due_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_acc_invoices_account_id ON accounting.invoices(account_id);
CREATE INDEX idx_acc_invoices_status ON accounting.invoices(status);

-- Insert accounting.invoices test data (10 records)
INSERT INTO accounting.invoices (account_id, invoice_number, amount, currency, status, due_date, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 'INV-2024-001', 1500.00, 'USD', 'paid', '2024-01-31', '2024-01-05 09:00:00+00'),
('11111111-1111-1111-1111-111111111111', 'INV-2024-002', 2500.00, 'USD', 'paid', '2024-02-28', '2024-02-01 10:00:00+00'),
('11111111-1111-1111-1111-111111111111', 'INV-2024-003', 1800.00, 'USD', 'pending', '2024-03-31', '2024-03-05 09:00:00+00'),
('22222222-2222-2222-2222-222222222222', 'INV-2024-004', 3200.00, 'EUR', 'paid', '2024-01-15', '2024-01-02 11:00:00+00'),
('22222222-2222-2222-2222-222222222222', 'INV-2024-005', 4500.00, 'USD', 'overdue', '2024-02-15', '2024-02-01 14:00:00+00'),
('33333333-3333-3333-3333-333333333333', 'INV-2024-006', 7500.00, 'USD', 'paid', '2024-01-31', '2024-01-10 09:00:00+00'),
('33333333-3333-3333-3333-333333333333', 'INV-2024-007', 2100.00, 'GBP', 'pending', '2024-03-15', '2024-03-01 08:00:00+00'),
('11111111-1111-1111-1111-111111111111', 'INV-2024-008', 950.00, 'USD', 'cancelled', '2024-02-10', '2024-01-25 16:00:00+00'),
('22222222-2222-2222-2222-222222222222', 'INV-2024-009', 1200.00, 'USD', 'paid', '2024-03-20', '2024-03-05 12:00:00+00'),
('33333333-3333-3333-3333-333333333333', 'INV-2024-010', 5600.00, 'USD', 'pending', '2024-03-31', '2024-03-15 10:00:00+00');

-- Create reporting schema
CREATE SCHEMA IF NOT EXISTS reporting;

-- Create reporting.daily_summary table
CREATE TABLE reporting.daily_summary (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_date DATE NOT NULL,
    account_id UUID NOT NULL,
    total_credits DECIMAL(15,2) NOT NULL DEFAULT 0,
    total_debits DECIMAL(15,2) NOT NULL DEFAULT 0,
    transaction_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_rep_summary_date ON reporting.daily_summary(report_date);
CREATE INDEX idx_rep_summary_account ON reporting.daily_summary(account_id);

-- Insert reporting.daily_summary test data (12 records)
INSERT INTO reporting.daily_summary (report_date, account_id, total_credits, total_debits, transaction_count, created_at) VALUES
('2024-01-05', '11111111-1111-1111-1111-111111111111', 1500.00, 0.00, 1, '2024-01-06 00:00:00+00'),
('2024-01-10', '11111111-1111-1111-1111-111111111111', 0.00, 150.00, 1, '2024-01-11 00:00:00+00'),
('2024-01-15', '11111111-1111-1111-1111-111111111111', 0.00, 89.99, 1, '2024-01-16 00:00:00+00'),
('2024-01-08', '22222222-2222-2222-2222-222222222222', 2000.00, 0.00, 1, '2024-01-09 00:00:00+00'),
('2024-01-12', '22222222-2222-2222-2222-222222222222', 0.00, 45.00, 1, '2024-01-13 00:00:00+00'),
('2024-01-01', '33333333-3333-3333-3333-333333333333', 3500.00, 0.00, 1, '2024-01-02 00:00:00+00'),
('2024-02-05', '11111111-1111-1111-1111-111111111111', 1500.00, 0.00, 1, '2024-02-06 00:00:00+00'),
('2024-02-12', '11111111-1111-1111-1111-111111111111', 0.00, 180.00, 1, '2024-02-13 00:00:00+00'),
('2024-02-01', '33333333-3333-3333-3333-333333333333', 3500.00, 0.00, 1, '2024-02-02 00:00:00+00'),
('2024-03-05', '11111111-1111-1111-1111-111111111111', 1500.00, 0.00, 1, '2024-03-06 00:00:00+00'),
('2024-03-08', '11111111-1111-1111-1111-111111111111', 0.00, 200.00, 1, '2024-03-09 00:00:00+00'),
('2024-03-01', '33333333-3333-3333-3333-333333333333', 3500.00, 0.00, 1, '2024-03-02 00:00:00+00');
