CREATE EXTENSION IF NOT EXISTS pgcrypto;
ALTER TABLE containers
    ADD COLUMN IF NOT EXISTS container_token TEXT;

DO $$
DECLARE r RECORD;
BEGIN
  FOR r IN SELECT id FROM containers WHERE container_token IS NULL LOOP
    UPDATE containers
      SET container_token = encode(gen_random_bytes(24), 'hex'), updated_at = NOW()
      WHERE id = r.id;
  END LOOP;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS containers_container_token_unique ON containers(container_token);

