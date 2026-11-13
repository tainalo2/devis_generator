-- Remove rows introduced by this migration before restoring narrower checks.
-- Audit rows must be removed explicitly because deleting a secret only nulls
-- secret_id and leaves the denormalized kind in the audit table.
DELETE FROM payment_secret_audit
WHERE kind IN ('oauth_gmail_token', 'oauth_microsoft_token', 'smtp_password');

DELETE FROM payment_secrets
WHERE kind IN ('oauth_gmail_token', 'oauth_microsoft_token', 'smtp_password');

-- Rollback: revert to payment_secrets only (iban, byo_provider_api_key, stripe_connect_credential)
ALTER TABLE payment_secrets
DROP CONSTRAINT payment_secrets_kind_check,
ADD CONSTRAINT payment_secrets_kind_check 
    CHECK (kind IN ('iban', 'byo_provider_api_key', 'stripe_connect_credential'));

ALTER TABLE payment_secret_audit
DROP CONSTRAINT IF EXISTS payment_secret_audit_kind_check,
ADD CONSTRAINT payment_secret_audit_kind_check
    CHECK (kind IN ('iban', 'byo_provider_api_key', 'stripe_connect_credential'));
