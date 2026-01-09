-- SQL Server Test Data: Q3 2024 (July-September)
-- Creates transactions table with test data for integration tests
-- Uses master database for simplicity in test environments

-- Create database if not exists
IF DB_ID('testdb') IS NULL
    CREATE DATABASE testdb;
GO

-- Switch to testdb
USE testdb;
GO

-- Drop table if exists
IF OBJECT_ID('dbo.transactions', 'U') IS NOT NULL
    DROP TABLE dbo.transactions;
GO

-- Create transactions table
CREATE TABLE dbo.transactions (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWID(),
    account_id VARCHAR(36) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    type VARCHAR(10) NOT NULL,
    description NVARCHAR(255),
    category VARCHAR(50),
    status VARCHAR(20) NOT NULL,
    created_at DATETIME2 NOT NULL
);
GO

-- Create indexes
CREATE INDEX idx_transactions_account_id ON dbo.transactions(account_id);
CREATE INDEX idx_transactions_created_at ON dbo.transactions(created_at);
CREATE INDEX idx_transactions_category ON dbo.transactions(category);
GO

-- Insert Q3 2024 test data (July-September)
-- July 2024
INSERT INTO dbo.transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 1500.00, 'USD', 'credit', 'Salary July', 'salary', 'completed', '2024-07-05 09:00:00'),
('11111111-1111-1111-1111-111111111111', 200.00, 'USD', 'debit', 'Grocery Store', 'groceries', 'completed', '2024-07-10 14:00:00'),
('11111111-1111-1111-1111-111111111111', 85.00, 'USD', 'debit', 'Electric Bill', 'utilities', 'completed', '2024-07-15 10:00:00'),
('11111111-1111-1111-1111-111111111111', 60.00, 'USD', 'debit', 'Streaming Service', 'entertainment', 'completed', '2024-07-20 08:00:00'),
('22222222-2222-2222-2222-222222222222', 2200.00, 'USD', 'credit', 'Freelance Payment', 'salary', 'completed', '2024-07-08 11:00:00'),
('22222222-2222-2222-2222-222222222222', 350.00, 'EUR', 'debit', 'Summer Vacation', 'travel', 'completed', '2024-07-25 16:00:00'),
('33333333-3333-3333-3333-333333333333', 3500.00, 'USD', 'credit', 'Monthly Salary', 'salary', 'completed', '2024-07-01 09:00:00'),
('33333333-3333-3333-3333-333333333333', 180.00, 'USD', 'debit', 'Concert Tickets', 'entertainment', 'completed', '2024-07-22 19:00:00');
GO

-- August 2024
INSERT INTO dbo.transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 1500.00, 'USD', 'credit', 'Salary August', 'salary', 'completed', '2024-08-05 09:00:00'),
('11111111-1111-1111-1111-111111111111', 220.00, 'USD', 'debit', 'Supermarket', 'groceries', 'completed', '2024-08-12 15:00:00'),
('11111111-1111-1111-1111-111111111111', 95.00, 'USD', 'debit', 'Water Bill', 'utilities', 'completed', '2024-08-18 10:00:00'),
('22222222-2222-2222-2222-222222222222', 1800.00, 'USD', 'credit', 'Contract Work', 'salary', 'completed', '2024-08-10 14:00:00'),
('22222222-2222-2222-2222-222222222222', 150.00, 'GBP', 'debit', 'Weekend Trip', 'travel', 'completed', '2024-08-28 12:00:00'),
('33333333-3333-3333-3333-333333333333', 3500.00, 'USD', 'credit', 'Monthly Salary', 'salary', 'completed', '2024-08-01 09:00:00'),
('33333333-3333-3333-3333-333333333333', 75.00, 'USD', 'debit', 'Gaming Purchase', 'entertainment', 'completed', '2024-08-15 20:00:00');
GO

-- September 2024
INSERT INTO dbo.transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 1500.00, 'USD', 'credit', 'Salary September', 'salary', 'completed', '2024-09-05 09:00:00'),
('11111111-1111-1111-1111-111111111111', 185.00, 'USD', 'debit', 'Grocery Shopping', 'groceries', 'completed', '2024-09-14 14:30:00'),
('11111111-1111-1111-1111-111111111111', 110.00, 'USD', 'debit', 'Internet Bill', 'utilities', 'completed', '2024-09-20 09:00:00'),
('22222222-2222-2222-2222-222222222222', 2000.00, 'USD', 'credit', 'Project Completion', 'salary', 'completed', '2024-09-12 16:00:00'),
('22222222-2222-2222-2222-222222222222', 95.00, 'USD', 'debit', 'Movie Night', 'entertainment', 'completed', '2024-09-22 21:00:00'),
('33333333-3333-3333-3333-333333333333', 3500.00, 'USD', 'credit', 'Monthly Salary', 'salary', 'completed', '2024-09-01 09:00:00'),
('33333333-3333-3333-3333-333333333333', 200.00, 'USD', 'debit', 'Home Repairs', 'utilities', 'completed', '2024-09-25 11:00:00');
GO

-- Pending transactions
INSERT INTO dbo.transactions (account_id, amount, currency, type, description, category, status, created_at) VALUES
('11111111-1111-1111-1111-111111111111', 65.00, 'USD', 'debit', 'Pending Delivery', 'groceries', 'pending', '2024-09-28 14:00:00'),
('33333333-3333-3333-3333-333333333333', 85.00, 'USD', 'debit', 'Pending Service', 'entertainment', 'pending', '2024-09-29 09:00:00');
GO

-- =============================================================================
-- MULTI-SCHEMA TEST DATA
-- =============================================================================

-- Create finance schema
CREATE SCHEMA finance;
GO

-- Create finance.payments table
CREATE TABLE finance.payments (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWID(),
    account_id VARCHAR(36) NOT NULL,
    payment_reference VARCHAR(50) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    payment_method VARCHAR(30) NOT NULL,
    status VARCHAR(20) NOT NULL,
    processed_at DATETIME2 NOT NULL
);
GO

CREATE INDEX idx_fin_payments_account ON finance.payments(account_id);
CREATE INDEX idx_fin_payments_status ON finance.payments(status);
GO

-- Insert finance.payments test data (10 records)
INSERT INTO finance.payments (account_id, payment_reference, amount, currency, payment_method, status, processed_at) VALUES
('11111111-1111-1111-1111-111111111111', 'PAY-2024-001', 500.00, 'USD', 'credit_card', 'completed', '2024-07-05 10:00:00'),
('11111111-1111-1111-1111-111111111111', 'PAY-2024-002', 750.00, 'USD', 'bank_transfer', 'completed', '2024-07-15 14:00:00'),
('11111111-1111-1111-1111-111111111111', 'PAY-2024-003', 200.00, 'USD', 'debit_card', 'pending', '2024-08-01 09:00:00'),
('22222222-2222-2222-2222-222222222222', 'PAY-2024-004', 1200.00, 'EUR', 'bank_transfer', 'completed', '2024-07-20 11:00:00'),
('22222222-2222-2222-2222-222222222222', 'PAY-2024-005', 350.00, 'USD', 'credit_card', 'failed', '2024-08-10 16:00:00'),
('33333333-3333-3333-3333-333333333333', 'PAY-2024-006', 2500.00, 'USD', 'wire_transfer', 'completed', '2024-07-01 08:00:00'),
('33333333-3333-3333-3333-333333333333', 'PAY-2024-007', 180.00, 'GBP', 'credit_card', 'completed', '2024-08-05 12:00:00'),
('11111111-1111-1111-1111-111111111111', 'PAY-2024-008', 95.00, 'USD', 'debit_card', 'refunded', '2024-08-20 10:00:00'),
('22222222-2222-2222-2222-222222222222', 'PAY-2024-009', 600.00, 'USD', 'bank_transfer', 'completed', '2024-09-05 09:00:00'),
('33333333-3333-3333-3333-333333333333', 'PAY-2024-010', 1800.00, 'USD', 'wire_transfer', 'pending', '2024-09-15 14:00:00');
GO

-- Create analytics schema
CREATE SCHEMA analytics;
GO

-- Create analytics.monthly_metrics table
CREATE TABLE analytics.monthly_metrics (
    id UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWID(),
    metric_month DATE NOT NULL,
    account_id VARCHAR(36) NOT NULL,
    revenue DECIMAL(15,2) NOT NULL DEFAULT 0,
    expenses DECIMAL(15,2) NOT NULL DEFAULT 0,
    profit_margin DECIMAL(5,2) NOT NULL DEFAULT 0,
    created_at DATETIME2 NOT NULL
);
GO

CREATE INDEX idx_ana_metrics_month ON analytics.monthly_metrics(metric_month);
CREATE INDEX idx_ana_metrics_account ON analytics.monthly_metrics(account_id);
GO

-- Insert analytics.monthly_metrics test data (9 records)
INSERT INTO analytics.monthly_metrics (metric_month, account_id, revenue, expenses, profit_margin, created_at) VALUES
('2024-07-01', '11111111-1111-1111-1111-111111111111', 15000.00, 8500.00, 43.33, '2024-08-01 00:00:00'),
('2024-07-01', '22222222-2222-2222-2222-222222222222', 22000.00, 12000.00, 45.45, '2024-08-01 00:00:00'),
('2024-07-01', '33333333-3333-3333-3333-333333333333', 35000.00, 18000.00, 48.57, '2024-08-01 00:00:00'),
('2024-08-01', '11111111-1111-1111-1111-111111111111', 16500.00, 9200.00, 44.24, '2024-09-01 00:00:00'),
('2024-08-01', '22222222-2222-2222-2222-222222222222', 19800.00, 11500.00, 41.92, '2024-09-01 00:00:00'),
('2024-08-01', '33333333-3333-3333-3333-333333333333', 37500.00, 19500.00, 48.00, '2024-09-01 00:00:00'),
('2024-09-01', '11111111-1111-1111-1111-111111111111', 17200.00, 9800.00, 43.02, '2024-10-01 00:00:00'),
('2024-09-01', '22222222-2222-2222-2222-222222222222', 21000.00, 11800.00, 43.81, '2024-10-01 00:00:00'),
('2024-09-01', '33333333-3333-3333-3333-333333333333', 38000.00, 20000.00, 47.37, '2024-10-01 00:00:00');
GO
