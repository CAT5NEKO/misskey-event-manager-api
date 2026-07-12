CREATE TABLE IF NOT EXISTS instance_allowlist (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host VARCHAR(255) NOT NULL UNIQUE,
    description VARCHAR(255),
    enabled BOOLEAN DEFAULT TRUE,
    protected BOOLEAN DEFAULT FALSE,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
