-- MySQL Test Fixtures for E2E Integration Tests
-- Contains Q2 2024 (April-June) transaction data

-- Create transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id CHAR(36) PRIMARY KEY,
    account_id CHAR(36) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    type ENUM('credit', 'debit') NOT NULL,
    description VARCHAR(500),
    category VARCHAR(100),
    status ENUM('pending', 'completed', 'failed') NOT NULL DEFAULT 'completed',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_account_id (account_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Insert Q2 2024 test transactions (20 records)
INSERT INTO transactions (id, account_id, amount, currency, type, description, category, status, created_at) VALUES
-- April 2024
(UUID(), '11111111-1111-1111-1111-111111111111', 1500.00, 'USD', 'credit', 'Salary April', 'salary', 'completed', '2024-04-05 09:00:00'),
(UUID(), '11111111-1111-1111-1111-111111111111', 175.00, 'USD', 'debit', 'Grocery Store', 'groceries', 'completed', '2024-04-12 14:00:00'),
(UUID(), '22222222-2222-2222-2222-222222222222', 2100.00, 'USD', 'credit', 'Freelance Work', 'salary', 'completed', '2024-04-08 10:00:00'),
(UUID(), '22222222-2222-2222-2222-222222222222', 350.00, 'EUR', 'debit', 'Hotel Barcelona', 'travel', 'completed', '2024-04-15 16:00:00'),
(UUID(), '33333333-3333-3333-3333-333333333333', 3500.00, 'USD', 'credit', 'Monthly Salary', 'salary', 'completed', '2024-04-01 09:00:00'),
(UUID(), '33333333-3333-3333-3333-333333333333', 89.00, 'USD', 'debit', 'Gym Membership', 'entertainment', 'completed', '2024-04-20 11:00:00'),
-- May 2024
(UUID(), '11111111-1111-1111-1111-111111111111', 1500.00, 'USD', 'credit', 'Salary May', 'salary', 'completed', '2024-05-05 09:00:00'),
(UUID(), '11111111-1111-1111-1111-111111111111', 220.00, 'USD', 'debit', 'Grocery Store', 'groceries', 'completed', '2024-05-10 15:30:00'),
(UUID(), '22222222-2222-2222-2222-222222222222', 1800.00, 'USD', 'credit', 'Project Payment', 'salary', 'completed', '2024-05-12 14:00:00'),
(UUID(), '22222222-2222-2222-2222-222222222222', 65.00, 'USD', 'debit', 'Spotify Premium', 'entertainment', 'completed', '2024-05-01 08:00:00'),
(UUID(), '33333333-3333-3333-3333-333333333333', 3500.00, 'USD', 'credit', 'Monthly Salary', 'salary', 'completed', '2024-05-01 09:00:00'),
(UUID(), '33333333-3333-3333-3333-333333333333', 180.00, 'GBP', 'debit', 'Train Tickets', 'travel', 'completed', '2024-05-18 12:00:00'),
-- June 2024
(UUID(), '11111111-1111-1111-1111-111111111111', 1500.00, 'USD', 'credit', 'Salary June', 'salary', 'completed', '2024-06-05 09:00:00'),
(UUID(), '11111111-1111-1111-1111-111111111111', 95.00, 'USD', 'debit', 'Electric Bill', 'utilities', 'completed', '2024-06-15 10:00:00'),
(UUID(), '22222222-2222-2222-2222-222222222222', 2500.00, 'USD', 'credit', 'Bonus Q2', 'salary', 'completed', '2024-06-30 16:00:00'),
(UUID(), '22222222-2222-2222-2222-222222222222', 120.00, 'USD', 'debit', 'Restaurant', 'entertainment', 'completed', '2024-06-22 20:00:00'),
(UUID(), '33333333-3333-3333-3333-333333333333', 3500.00, 'USD', 'credit', 'Monthly Salary', 'salary', 'completed', '2024-06-01 09:00:00'),
(UUID(), '33333333-3333-3333-3333-333333333333', 250.00, 'USD', 'debit', 'Insurance', 'utilities', 'completed', '2024-06-10 11:00:00'),
-- Pending
(UUID(), '11111111-1111-1111-1111-111111111111', 50.00, 'USD', 'debit', 'Pending Order', 'groceries', 'pending', '2024-06-28 14:00:00'),
(UUID(), '33333333-3333-3333-3333-333333333333', 75.00, 'USD', 'debit', 'Pending Subscription', 'entertainment', 'pending', '2024-06-29 09:00:00');
