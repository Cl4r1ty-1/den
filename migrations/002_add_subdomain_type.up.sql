ALTER TABLE subdomains ADD COLUMN subdomain_type VARCHAR(20) DEFAULT 'project' CHECK (subdomain_type IN ('username', 'project'));

DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='containers' AND column_name='allocated_ports') THEN
        ALTER TABLE containers ADD COLUMN allocated_ports INTEGER[];
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='nodes' AND column_name='public_hostname') THEN
        ALTER TABLE nodes ADD COLUMN public_hostname VARCHAR(255);
    END IF;
END $$;
