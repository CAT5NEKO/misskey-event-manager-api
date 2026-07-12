CREATE TABLE IF NOT EXISTS system_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(100) NOT NULL UNIQUE,
    value JSONB NOT NULL,
    updated_by UUID REFERENCES users(id) ON DELETE CASCADE,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
