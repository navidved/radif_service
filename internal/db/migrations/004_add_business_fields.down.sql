ALTER TABLE users
    DROP COLUMN IF EXISTS business_phone,
    DROP COLUMN IF EXISTS address;
