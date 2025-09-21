CREATE TABLE verification_sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id VARCHAR(255) NOT NULL UNIQUE,
    session_number INTEGER,
    session_token VARCHAR(255),
    vendor_data VARCHAR(255),
    workflow_id VARCHAR(255) NOT NULL,
    verification_url TEXT,
    status VARCHAR(50) DEFAULT 'not_started',
    decision JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_verification_sessions_user_id ON verification_sessions(user_id);
CREATE INDEX idx_verification_sessions_session_id ON verification_sessions(session_id);
CREATE INDEX idx_verification_sessions_status ON verification_sessions(status);