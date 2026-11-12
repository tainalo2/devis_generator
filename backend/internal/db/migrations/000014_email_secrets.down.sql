-- Rollback: revert to payment_secrets only (iban, byo_provider_api_key, stripe_connect_credential)
ALTER TABLE payment_secrets
DROP CONSTRAINT payment_secrets_kind_check,
ADD CONSTRAINT payment_secrets_kind_check 
    CHECK (kind IN ('iban', 'byo_provider_api_key', 'stripe_connect_credential'));

ALTER TABLE payment_secret_audit
DROP CONSTRAINT IF EXISTS payment_secret_audit_kind_check,
ADD CONSTRAINT payment_secret_audit_kind_check
    CHECK (kind IN ('iban', 'byo_provider_api_key', 'stripe_connect_credential'));
