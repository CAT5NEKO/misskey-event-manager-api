CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    misskey_user_id VARCHAR(255) NOT NULL,
    misskey_username VARCHAR(255) NOT NULL,
    misskey_host VARCHAR(255) NOT NULL,
    misskey_token TEXT NOT NULL,
    name VARCHAR(255),
    avatar_url TEXT,
    is_admin BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(misskey_user_id, misskey_host)
);
