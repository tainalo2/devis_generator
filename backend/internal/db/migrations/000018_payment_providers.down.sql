DROP TABLE IF EXISTS payment_webhook_events;
DROP TABLE IF EXISTS payment_attempts;
DROP TABLE IF EXISTS stripe_connect_accounts;
DROP TABLE IF EXISTS payment_provider_configs;
DELETE FROM payment_secret_audit WHERE kind IN ('stripe_api_key','stripe_webhook_secret','mollie_api_key','paypal_client_id','paypal_client_secret','paypal_webhook_id','stripe_connect_secret','stripe_connect_webhook_secret');
DELETE FROM payment_secrets WHERE kind IN ('stripe_api_key','stripe_webhook_secret','mollie_api_key','paypal_client_id','paypal_client_secret','paypal_webhook_id','stripe_connect_secret','stripe_connect_webhook_secret');
ALTER TABLE payment_secrets DROP CONSTRAINT IF EXISTS payment_secrets_kind_check;
ALTER TABLE payment_secrets ADD CONSTRAINT payment_secrets_kind_check CHECK (kind IN ('iban','bic','byo_provider_api_key','stripe_connect_credential','oauth_gmail_token','oauth_microsoft_token','smtp_password'));
