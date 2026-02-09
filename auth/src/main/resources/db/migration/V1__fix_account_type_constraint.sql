-- Fix the typo in account_type CHECK constraint
-- Changes 'Buisiness' (typo) to 'Business' (correct spelling)

-- Drop the existing constraint with the typo
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_account_type_check;

-- Add the corrected constraint
ALTER TABLE users ADD CONSTRAINT users_account_type_check 
    CHECK (account_type IN ('Business', 'Community'));
