-- Migration: Add security_deposit to contracts table
-- This allows separating the one-time deposit from the recurring rent amount.
ALTER TABLE contracts ADD COLUMN security_deposit decimal(15,2) DEFAULT 0.00 NOT NULL;
