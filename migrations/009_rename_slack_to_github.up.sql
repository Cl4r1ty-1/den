ALTER TABLE users RENAME COLUMN slack_id TO github_id;

DROP INDEX IF EXISTS idx_users_slack_id;
CREATE INDEX IF NOT EXISTS idx_users_github_id ON users(github_id);
