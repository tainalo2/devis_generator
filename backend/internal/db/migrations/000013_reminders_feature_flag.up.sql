ALTER TABLE payment_feature_flags
    ADD COLUMN IF NOT EXISTS reminders_enabled BOOLEAN NOT NULL DEFAULT FALSE;