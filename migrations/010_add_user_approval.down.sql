DROP INDEX IF EXISTS idx_users_approval_status;
ALTER TABLE users DROP COLUMN IF EXISTS rejection_reason;
ALTER TABLE users DROP COLUMN IF EXISTS approved_at;
ALTER TABLE users DROP COLUMN IF EXISTS approved_by;
ALTER TABLE users DROP COLUMN IF EXISTS approval_status;
