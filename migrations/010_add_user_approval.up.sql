ALTER TABLE users ADD COLUMN approval_status VARCHAR(20) DEFAULT 'pending' CHECK (approval_status IN ('pending', 'approved', 'rejected'));
ALTER TABLE users ADD COLUMN approved_by INTEGER REFERENCES users(id);
ALTER TABLE users ADD COLUMN approved_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE users ADD COLUMN rejection_reason TEXT;

CREATE INDEX idx_users_approval_status ON users(approval_status);

UPDATE users SET approval_status = 'approved', approved_at = NOW() WHERE approval_status = 'pending';
