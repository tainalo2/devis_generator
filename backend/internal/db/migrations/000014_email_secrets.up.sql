-- Email credentials: OAuth tokens (Gmail, Microsoft 365) and SMTP passwords.
-- Extend payment_secrets table to support email credential kinds, sharing the
-- same encryption + audit pattern (paymentsecrets.Vault).

-- Extend the kind enum to include email credential types
ALTER TABLE payment_secrets
DROP CONSTRAINT payment_secrets_kind_check,
ADD CONSTRAINT payment_secrets_kind_check 
    CHECK (kind IN (
        'iban', 
        'byo_provider_api_key', 
        'stripe_connect_credential',
        'oauth_gmail_token',
        'oauth_microsoft_token',
        'smtp_password'
    ));

ALTER TABLE payment_secret_audit
DROP CONSTRAINT IF EXISTS payment_secret_audit_kind_check,
ADD CONSTRAINT payment_secret_audit_kind_check
    CHECK (kind IN (
        'iban', 
        'byo_provider_api_key', 
        'stripe_connect_credential',
        'oauth_gmail_token',
        'oauth_microsoft_token',
        'smtp_password'
    ));
