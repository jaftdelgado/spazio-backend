-- Rollback Migration: Remove security_deposit from contracts table
ALTER TABLE contracts DROP COLUMN IF EXISTS security_deposit;
