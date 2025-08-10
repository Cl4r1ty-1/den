CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    slack_id VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    is_admin BOOLEAN DEFAULT FALSE,
    container_id VARCHAR(255),
    ssh_password VARCHAR(255),
    ssh_public_key TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE nodes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    token VARCHAR(255) UNIQUE NOT NULL,
    max_memory_mb INTEGER DEFAULT 4096,
    max_cpu_cores INTEGER DEFAULT 4,
    max_storage_gb INTEGER DEFAULT 15,
    is_online BOOLEAN DEFAULT FALSE,
    last_seen TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE TABLE containers (
    id VARCHAR(255) PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    node_id INTEGER REFERENCES nodes(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'creating',
    ip_address INET,
    ssh_port INTEGER,
    memory_mb INTEGER DEFAULT 4096,
    cpu_cores INTEGER DEFAULT 4,
    storage_gb INTEGER DEFAULT 15,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE TABLE subdomains (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    subdomain VARCHAR(255) UNIQUE NOT NULL,
    target_port INTEGER NOT NULL,
    subdomain_type VARCHAR(20) DEFAULT 'project' CHECK (subdomain_type IN ('username', 'project')),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE TABLE port_mappings (
    id SERIAL PRIMARY KEY,
    container_id VARCHAR(255) REFERENCES containers(id) ON DELETE CASCADE,
    internal_port INTEGER NOT NULL,
    external_port INTEGER NOT NULL,
    protocol VARCHAR(10) DEFAULT 'tcp',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(external_port, protocol)
);
CREATE TABLE sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
CREATE INDEX idx_users_slack_id ON users(slack_id);
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_containers_user_id ON containers(user_id);
CREATE INDEX idx_containers_node_id ON containers(node_id);
CREATE INDEX idx_subdomains_user_id ON subdomains(user_id);
CREATE INDEX idx_subdomains_subdomain ON subdomains(subdomain);
CREATE INDEX idx_port_mappings_container_id ON port_mappings(container_id);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_nodes_updated_at BEFORE UPDATE ON nodes FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_containers_updated_at BEFORE UPDATE ON containers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_subdomains_updated_at BEFORE UPDATE ON subdomains FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
