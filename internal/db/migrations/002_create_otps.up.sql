CREATE TABLE IF NOT EXISTS otps (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    phone      VARCHAR(11) NOT NULL,
    code       VARCHAR(5)  NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_otps_phone_active
    ON otps (phone)
    WHERE used_at IS NULL;
