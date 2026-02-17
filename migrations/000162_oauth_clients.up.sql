CREATE TABLE IF NOT EXISTS oauth_clients (
    id UUID NOT NULL PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id TEXT NOT NULL UNIQUE,
    client_secret TEXT NOT NULL,
    client_name TEXT,
    allowed_scopes TEXT DEFAULT 'read write',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- Create index on client_id for fast lookups
CREATE INDEX idx_oauth_clients_client_id ON oauth_clients(client_id) WHERE is_active = true;

-- Insert default Uzum Tezkor client (client_secret will be hashed by the application on first boot)
-- Note: The actual credentials will come from environment variables
-- This is a placeholder that should be replaced/updated via application initialization
COMMENT ON TABLE oauth_clients IS 'Stores OAuth2 client credentials for API integrations';
