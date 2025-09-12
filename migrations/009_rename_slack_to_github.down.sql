ALTER TABLE users RENAME COLUMN github_id TO slack_id;

DROP INDEX IF EXISTS idx_users_github_id;
CREATE INDEX IF NOT EXISTS idx_users_slack_id ON users(slack_id);
