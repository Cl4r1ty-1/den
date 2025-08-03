DROP TRIGGER IF EXISTS update_subdomains_updated_at ON subdomains;
DROP TRIGGER IF EXISTS update_containers_updated_at ON containers;
DROP TRIGGER IF EXISTS update_nodes_updated_at ON nodes;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_sessions_expires_at;
DROP INDEX IF EXISTS idx_sessions_user_id;
DROP INDEX IF EXISTS idx_port_mappings_container_id;
DROP INDEX IF EXISTS idx_subdomains_subdomain;
DROP INDEX IF EXISTS idx_subdomains_user_id;
DROP INDEX IF EXISTS idx_containers_node_id;
DROP INDEX IF EXISTS idx_containers_user_id;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_slack_id;

DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS port_mappings;
DROP TABLE IF EXISTS subdomains;
DROP TABLE IF EXISTS containers;
DROP TABLE IF EXISTS nodes;
DROP TABLE IF EXISTS users;
