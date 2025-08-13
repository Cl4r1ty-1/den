CREATE TABLE IF NOT EXISTS jobs (
    id SERIAL PRIMARY KEY,
    type TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued','running','success','failed')) DEFAULT 'queued',
    payload JSONB NOT NULL,
    result JSONB,
    error TEXT,
    run_after TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_jobs_status_run_after ON jobs(status, run_after);

