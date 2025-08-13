CREATE TABLE IF NOT EXISTS exports (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    container_id TEXT NOT NULL,
    object_key TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending','uploading','complete','failed','expired')),
    size_bytes BIGINT,
    expires_at TIMESTAMPTZ NOT NULL,
    requested_by INTEGER REFERENCES users(id) ON DELETE SET NULL,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exports_user_id ON exports(user_id);
CREATE INDEX IF NOT EXISTS idx_exports_expires_at ON exports(expires_at);
CREATE INDEX IF NOT EXISTS idx_exports_status ON exports(status);

