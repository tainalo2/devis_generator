# Guide de configuration des prestataires de paiement

This document covers setup and configuration of all 5 payment methods supported by Devis Generator: **RIB/IBAN**, **Stripe BYO**, **Mollie BYO**, **PayPal BYO**, and **Stripe Connect Express**.

**Target audience:** developers, QA engineers, DevOps engineers preparing for staging/production deployment.

---

## Table of Contents

1. [Overview](#overview)
2. [1. RIB/IBAN (Bank Transfer)](#1-ribiban-bank-transfer)
3. [2. Stripe BYO (Bring Your Own)](#2-stripe-byo-bring-your-own)
4. [3. Mollie BYO](#3-mollie-byo)
5. [4. PayPal BYO](#4-paypal-byo)
6. [5. Stripe Connect Express](#5-stripe-connect-express)
7. [Environment Variable Reference](#environment-variable-reference)
8. [Supported File Formats](#supported-file-formats)
9. [Troubleshooting](#troubleshooting)

---

## Overview

All 5 payment methods are configured via the `PAYMENT_PROVIDERS` environment variable, which specifies a JSON file path containing provider credentials and settings.

### Payment Methods Summary

| Method | Type | Env | Use Case |
|--------|------|-----|----------|
| **RIB/IBAN** | Bank transfer (SEPA) | sandbox/live | Direct bank account; highest security via encryption vault |
| **Stripe BYO** | Card payment | sandbox/live | Use your own Stripe account; test/live keys prefix required |
| **Mollie BYO** | Cards + local methods | sandbox/live | European payment platform; test/live keys required |
| **PayPal BYO** | Wallet + transfers | sandbox/live | OAuth2 credentials; webhook registration separate |
| **Stripe Connect Express** | Managed payout | sandbox/live | Stripe handles onboarding; restricted to Stripe's platform |

**Credential Security:**
- **Encrypted at rest:** IBAN/BIC stored in vault (AES-256-GCM) via `ENCRYPTION_KEY`
- **API keys/secrets:** Encrypted in transit (TLS); stored in vault or environment file
- **Webhook secrets:** Always store in `.env` or vault; never commit to git

---

## 1. RIB/IBAN (Bank Transfer)

### What It Is

Direct SEPA bank transfer via IBAN (International Bank Account Number) + BIC (Bank Identifier Code). Customers see your actual bank account details to pay by transfer.

### Configuration

#### `.env` Setup (Minimum)

```bash
# Encryption key: generate via: openssl rand -base64 32
ENCRYPTION_KEY=jM8k9L2pQ5xZ1vN3tR7wS9yH0bG4cF6dE8aJ2mK5nO7qP9sT1uV3xW5yZ7

# Optional: explicitly enable RIB/IBAN
PAYMENT_PROVIDERS=/etc/devis/payment-providers.json
```

#### `payment-providers.json` Example

```json
{
  "rib_iban": {
    "iban": "FR1420041010050500013M02606",
    "bic": "BNAGFRPP"
  }
}
```

**Fields:**
- **`iban`** (string, required): Account number (13–34 chars, country-specific format)
- **`bic`** (string, required): Bank identifier (8 or 11 chars alphanumeric)

### Storage Security

1. **IBAN/BIC are encrypted** in the application vault using `ENCRYPTION_KEY`
2. All plaintext accesses are **audited** (user ID, timestamp, request ID logged)
3. At rest in PostgreSQL, credentials appear as:
   ```sql
   SELECT masked_value FROM payment_secrets WHERE kind='iban';
   -- Output: FR14****0013M02606 (last 10 chars visible)
   ```

### Testing in Sandbox

IBAN/IBAN works in both `sandbox` and `live` modes—no special test accounts required. Use any valid IBAN format from your bank.

### Webhook Configuration

RIB/IBAN does **not** use webhooks. Payments are manually reconciled by matching transfer amounts and references.

---

## 2. Stripe BYO (Bring Your Own)

### What It Is

Use your own Stripe account for card payments (Visa, Mastercard, etc.). Charges appear on your Stripe dashboard; full Stripe feature set available.

### Prerequisites

1. **Stripe account:** https://stripe.com/en-eu
2. **API keys:** https://dashboard.stripe.com/apikeys
3. **Webhook secret:** https://dashboard.stripe.com/webhooks (create an endpoint)

### Configuration

#### Obtain API Keys (Stripe Dashboard)

**Sandbox (Test Mode):**
1. Toggle **Test Mode** (upper-left in Stripe dashboard)
2. Copy **Secret Key** (starts with `sk_test_`)
3. **Do NOT use** Publishable Key for backend

**Live Mode:**
1. Toggle **Live Mode**
2. Copy **Secret Key** (starts with `sk_live_`)
3. Requires explicit account activation (not recommended for first-time setup)

#### Obtain Webhook Secret (Stripe Dashboard)

1. Go to **Developers → Webhooks**
2. Click **Add Endpoint** (or edit existing)
3. Set webhook URL to: `https://your-domain.com/api/v1/payments/webhooks/stripe`
4. Select events:
   - `checkout.session.completed` (payment captured)
   - `charge.failed` (payment declined)
5. Copy **Signing Secret** (starts with `whsec_` or `we_`)

#### `.env` Setup

```bash
# Test mode (sandbox)
STRIPE_SECRET_KEY=sk_test_4eC39HqLyjWDarftXjzAhS5
STRIPE_WEBHOOK_SECRET=whsec_test_secret_abcd1234

# Live mode (production only)
# STRIPE_SECRET_KEY=sk_live_ReTllpNZwABcd...
# STRIPE_WEBHOOK_SECRET=whsec_live_secret_...
```

#### `payment-providers.json` Example

```json
{
  "stripe": {
    "mode": "sandbox",
    "secret_key": "sk_test_4eC39HqLyjWDarftXjzAhS5",
    "webhook_secret": "whsec_test_secret_abcd1234"
  }
}
```

**Fields:**
- **`mode`** (string): `sandbox` or `live`
- **`secret_key`** (string): API secret key (must match mode)
- **`webhook_secret`** (string): Webhook signing secret

### Configuration Validation

The backend validates on startup:
```go
if mode == sandbox && !strings.HasPrefix(secret_key, "sk_test_") {
  return fmt.Errorf("sandbox requires test key")
}
if mode == live && !strings.HasPrefix(secret_key, "sk_live_") {
  return fmt.Errorf("live requires live key")
}
```

### Testing

**Sandbox test cards:**
- **Visa:** `4242 4242 4242 4242` | exp: any future date | CVC: any 3 digits
- **Visa (decline):** `4000 0000 0000 0002` | exp: any future date | CVC: any 3 digits
- **Amex:** `3782 822463 10005` | exp: any future date | CVC: any 4 digits

### Webhook Configuration (Runtime)

The backend does **not** auto-register webhooks. Configure manually in Stripe Dashboard:

1. **Endpoint URL:** `https://your-domain.com/api/v1/payments/webhooks/stripe`
2. **Signing secret** in config must match the webhook secret in Stripe Dashboard
3. Test by sending a sample event from Stripe Dashboard (Webhooks → Test signing secret)

---

## 3. Mollie BYO

### What It Is

European payment processor supporting cards, iDEAL, Bancontact, SEPA transfer, and regional methods. Mollie handles payment processing; you receive funds into your Mollie account.

### Prerequisites

1. **Mollie account:** https://www.mollie.com
2. **API key:** https://dashboard.mollie.com/settings/developers/api-keys
3. **Webhook URL:** Public HTTPS endpoint (required)

### Configuration

#### Obtain API Key (Mollie Dashboard)

1. Go to **Developers → API Keys**
2. **Test mode:**
   - Copy **Test API Key** (starts with `test_`)
   - Use for sandbox/testing
3. **Live mode:**
   - Copy **Live API Key** (starts with `live_`)
   - Only after account activation

#### `.env` Setup

```bash
# Test mode (sandbox)
MOLLIE_API_KEY=test_dummyApiKeyForTesting

# Live mode (production only)
# MOLLIE_API_KEY=live_abcdefghijk...
```

#### `payment-providers.json` Example

```json
{
  "mollie": {
    "mode": "sandbox",
    "api_key": "test_dummyApiKeyForTesting",
    "webhook_url": "https://your-domain.com/api/v1/payments/webhooks/mollie"
  }
}
```

**Fields:**
- **`mode`** (string): `sandbox` or `live`
- **`api_key`** (string): API key (must match mode: `test_` or `live_`)
- **`webhook_url`** (string): **Public HTTPS** URL where Mollie sends payment updates

### Key Requirement: Public Webhook URL

Mollie **requires a public, accessible HTTPS URL** to send webhook callbacks. If running locally:
1. Use a tunneling service:
   - **ngrok:** `ngrok http 8080` → get public URL
   - **localtunnel:** `lt --port 8080`
2. Set `webhook_url` to the tunneled URL: `https://xyz.ngrok.io/api/v1/payments/webhooks/mollie`

### Configuration Validation

The backend validates:
```go
if mode == sandbox && !strings.HasPrefix(api_key, "test_") {
  return fmt.Errorf("sandbox requires test key")
}
if mode == live && !strings.HasPrefix(api_key, "live_") {
  return fmt.Errorf("live requires live key")
}
```

### Testing

**Sandbox test amounts:**
- **€1.00 → Paid:** Payment succeeds
- **€2.00 → Failed:** Payment fails (e.g., card declined)
- **€3.00 → Pending:** Payment stays pending

**Sandbox test card:**
- Card number: `5555555555554444`
- Expiry: any future date
- CVC: any 3 digits

### Webhook Configuration (Runtime)

Mollie webhooks are **auto-registered** via the `webhook_url` parameter. When a payment is created, Mollie sends a POST to your webhook URL with:

```
POST /api/v1/payments/webhooks/mollie
Content-Type: application/x-www-form-urlencoded

id=tr_abcd1234&status=paid
```

The backend retrieves payment details via API using the `id` and verifies the status.

---

## 4. PayPal BYO

### What It Is

PayPal wallet payments and transfers. Customers log into their PayPal account to approve payment; funds settle into your PayPal account.

### Prerequisites

1. **PayPal Developer Account:** https://developer.paypal.com
2. **OAuth2 Credentials:**
   - Client ID
   - Client Secret
3. **Webhook ID:** Registered in PayPal app settings

### Configuration

#### Obtain OAuth2 Credentials (PayPal Dashboard)

1. Log in to **PayPal Developer Dashboard**
2. Go to **Apps & Credentials**
3. Create or select your app
4. Copy:
   - **Client ID** (starts with `AZ...`)
   - **Client Secret** (long alphanumeric string)

#### Register Webhook (PayPal Dashboard)

1. Go to **Apps & Credentials → Webhooks**
2. Click **Create Webhook**
3. Select event types:
   - `checkout.order.completed`
   - `payment.capture.completed`
   - `payment.capture.denied`
4. Set URL to: `https://your-domain.com/api/v1/payments/webhooks/paypal`
5. Copy **Webhook ID** (from list of webhooks)

#### `.env` Setup

```bash
# Sandbox (test)
PAYPAL_CLIENT_ID=AZDxjhCRBqaV3V4...
PAYPAL_CLIENT_SECRET=your_secret_here
PAYPAL_WEBHOOK_ID=0A123456789...

# Live mode (requires activated account)
# PAYPAL_MODE=live
```

#### `payment-providers.json` Example

```json
{
  "paypal": {
    "mode": "sandbox",
    "client_id": "AZDxjhCRBqaV3V4...",
    "client_secret": "your_secret_here",
    "webhook_id": "0A123456789..."
  }
}
```

**Fields:**
- **`mode`** (string): `sandbox` or `live`
- **`client_id`** (string): OAuth2 Client ID
- **`client_secret`** (string): OAuth2 Client Secret
- **`webhook_id`** (string): Registered webhook ID

### Configuration Validation

The backend validates all three credentials are present:
```go
if client_id == "" || client_secret == "" || webhook_id == "" {
  return fmt.Errorf("missing creds")
}
```

### Testing

**Sandbox:**
- Use PayPal sandbox accounts (create in **Sandbox Accounts** section)
- Test PayPal account: Use any email/password defined in sandbox accounts
- Payments auto-complete after 3–5 seconds (unless manually declined in sandbox)

**PayPal Sandbox Test Accounts:**
1. Go to **Sandbox Accounts** in developer dashboard
2. Create **Business** account (seller) and **Personal** account (buyer)
3. Log in as Personal account to approve payments

### Webhook Configuration (Runtime)

PayPal webhooks are **manually registered** in the PayPal Developer Dashboard. The backend verifies webhook signatures using the **webhook ID** and your **access token**:

1. When a payment is completed, PayPal POSTs to your webhook URL with signed headers:
   ```
   POST /api/v1/payments/webhooks/paypal
   Paypal-Transmission-Id: ...
   Paypal-Transmission-Time: ...
   Paypal-Cert-Url: ...
   Paypal-Auth-Algo: SHA256withRSA
   Paypal-Transmission-Sig: ...
   ```
2. The backend verifies the signature against PayPal's certificates and your webhook ID

---

## 5. Stripe Connect Express

### What It Is

Stripe handles merchant onboarding directly (Stripe Connect Express). Your users create a Stripe Connected Account during payment setup; Stripe handles payouts automatically.

**Key difference from BYO:** You do **not** create connected accounts yourself; Stripe creates them on demand during checkout.

### Prerequisites

1. **Stripe account:** https://stripe.com
2. **Stripe Platform account** (not a standard account):
   - Contact Stripe support or enable Connect via dashboard
   - Stripe handles merchant verification automatically
3. **Restricted API key:** For Connect flows (less powerful than full secret key)

### Configuration

#### Create Restricted API Key (Stripe Dashboard)

1. Go to **Developers → Restricted API Keys**
2. Create a new key with:
   - **Permissions:**
     - `account`: read, write
     - `checkout.sessions`: write
     - `products`: read, write
   - **IP Whitelist:** Your app server IP (optional but recommended)
3. Copy the key (starts with `rk_test_` or `rk_live_`)

#### Webhook Secret (Same as Stripe BYO)

Configure the same webhook as Stripe BYO (see section 2).

#### `.env` Setup

```bash
# Sandbox
STRIPE_CONNECT_KEY=rk_test_4eC39HqLyjWDarftXjzAhS5
STRIPE_WEBHOOK_SECRET=whsec_test_secret_abcd1234

# Live
# STRIPE_CONNECT_KEY=rk_live_...
# STRIPE_WEBHOOK_SECRET=whsec_live_secret_...
```

#### `payment-providers.json` Example

```json
{
  "stripe_connect": {
    "mode": "sandbox",
    "restricted_key": "rk_test_4eC39HqLyjWDarftXjzAhS5",
    "webhook_secret": "whsec_test_secret_abcd1234",
    "connected_account_id": "acct_1A2B3C4D5E6F7G8H"
  }
}
```

**Fields:**
- **`mode`** (string): `sandbox` or `live`
- **`restricted_key`** (string): Restricted API key for Connect operations
- **`webhook_secret`** (string): Webhook signing secret
- **`connected_account_id`** (string, optional): Platform account ID (if not auto-detected)

### How It Works

1. **Onboarding:** User clicks "Pay with Stripe Connect"
2. **Express Account Creation:** Stripe automatically creates a Connected Account for the user
3. **Approval:** User completes Stripe's onboarding form (identity, banking, tax info)
4. **Payment:** Funds from invoice payment go directly to user's Connected Account
5. **Payouts:** Stripe automatically transfers funds (minus platform fees) to user's bank account

### Testing in Sandbox

**Test merchant credentials:**
- Stripe provides pre-configured test accounts in sandbox
- Use `acct_1AlZ123456789ABC` as a test connected account ID
- Payments auto-complete (no need for real card details)

### Configuration Validation

Same as Stripe BYO: keys must match sandbox/live mode.

### Webhook Configuration

Same as Stripe BYO; use the same webhook endpoint and signing secret.

---

## Environment Variable Reference

### `PAYMENT_PROVIDERS` (Path)

**Type:** string (file path)  
**Required:** No (optional; if unset, all payment methods disabled)  
**Example:** `/etc/devis/payment-providers.json`

Points to a JSON file containing provider configurations. File must be readable by the application process.

### `ENCRYPTION_KEY` (Encryption)

**Type:** base64-encoded 32-byte string  
**Required:** Yes (for IBAN/BIC storage)  
**Example:** `jM8k9L2pQ5xZ1vN3tR7wS9yH0bG4cF6dE8aJ2mK5nO7qP9sT1uV3xW5yZ7`

Generate via:
```bash
openssl rand -base64 32
```

**Deployment note:** Use the **same** key across all app instances; changing it will make existing encrypted IBANs unreadable.

### `STRIPE_SECRET_KEY` (Legacy)

**Type:** string  
**Required:** No (use `payment-providers.json` instead)  
**Deprecated:** Use `PAYMENT_PROVIDERS` with inline or file-based config

If set, overrides `payment-providers.json` for Stripe configuration (backward compatibility).

### `STRIPE_WEBHOOK_SECRET` (Legacy)

**Type:** string  
**Required:** No (use `payment-providers.json` instead)  
**Deprecated:** Use `PAYMENT_PROVIDERS`

---

## Supported File Formats

### Primary: JSON

**File:** `/etc/devis/payment-providers.json`

```json
{
  "rib_iban": {
    "iban": "FR1420041010050500013M02606",
    "bic": "BNAGFRPP"
  },
  "stripe": {
    "mode": "sandbox",
    "secret_key": "sk_test_...",
    "webhook_secret": "whsec_test_..."
  },
  "mollie": {
    "mode": "sandbox",
    "api_key": "test_...",
    "webhook_url": "https://your-domain.com/api/v1/payments/webhooks/mollie"
  },
  "paypal": {
    "mode": "sandbox",
    "client_id": "AZDxj...",
    "client_secret": "...",
    "webhook_id": "0A..."
  },
  "stripe_connect": {
    "mode": "sandbox",
    "restricted_key": "rk_test_...",
    "webhook_secret": "whsec_test_...",
    "connected_account_id": "acct_..."
  }
}
```

### Alternative: Environment Variables (Legacy)

If `PAYMENT_PROVIDERS` is not set, the backend falls back to individual environment variables:

```bash
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_test_...
MOLLIE_API_KEY=test_...
PAYPAL_CLIENT_ID=...
PAYPAL_CLIENT_SECRET=...
PAYPAL_WEBHOOK_ID=...
```

**Recommendation:** Use the JSON file for all new deployments; environment variables are for backward compatibility only.

---

## Troubleshooting

### "Missing webhook secret"

**Symptom:** `stripe: missing webhook secret` during startup

**Cause:** `webhook_secret` not configured in `payment-providers.json` or environment

**Fix:**
1. Go to Stripe Dashboard → **Developers → Webhooks**
2. Create or select endpoint: `https://your-domain.com/api/v1/payments/webhooks/stripe`
3. Copy **Signing Secret** (starts with `whsec_`)
4. Add to `payment-providers.json`:
   ```json
   "stripe": {
     "mode": "sandbox",
     "secret_key": "sk_test_...",
     "webhook_secret": "whsec_test_..."
   }
   ```

### "Sandbox requires test key"

**Symptom:** `stripe: sandbox requires test key` during startup

**Cause:** Using a live key (`sk_live_`) in sandbox mode

**Fix:**
1. Use a **test key** (starts with `sk_test_`)
2. Or switch `mode` to `live` (requires live key)

### "Invalid IBAN format"

**Symptom:** Webhook shows IBAN validation error

**Cause:** IBAN does not match expected format for the country

**Fix:**
1. Verify IBAN structure (France: 27 chars, Spain: 24 chars, etc.)
2. Check digit calculation using an IBAN validator: https://www.iban.com/structure
3. Ensure no spaces in `payment-providers.json`

### "PayPal webhook signature invalid"

**Symptom:** `invalid paypal signature` when processing webhook

**Cause:** Webhook ID in config does not match the registered webhook

**Fix:**
1. Go to PayPal Dashboard → **Apps & Credentials → Webhooks**
2. Verify the **Webhook ID** in your config matches the registered webhook
3. Verify the webhook URL is correct: `https://your-domain.com/api/v1/payments/webhooks/paypal`

### "Mollie webhook failed"

**Symptom:** Mollie webhooks not received

**Cause:** Webhook URL is not public (e.g., running on `localhost`)

**Fix:**
1. Ensure your app is publicly accessible via HTTPS
2. If developing locally, use a tunneling service:
   - **ngrok:** `ngrok http 8080` → Copy public URL
   - Set `webhook_url` to: `https://xyz.ngrok.io/api/v1/payments/webhooks/mollie`
3. Test webhook delivery:
   - Mollie Dashboard → **Webhooks → Test webhook**
   - Check app logs for incoming request

### "Payment creation failed: 400 Bad Request"

**Symptom:** Customer cannot initiate payment

**Cause:** Configuration mismatch (wrong key, wrong mode, missing webhook)

**Fix:**
1. Check logs for the full error message:
   ```
   POST /api/v1/payments
   Error: stripe: 400 — missing webhook secret
   ```
2. Verify **all required fields** are present in `payment-providers.json`
3. Verify mode/key prefix match (sandbox: `sk_test_`, `test_`, etc.; live: `sk_live_`, `live_`, etc.)
4. Verify webhook URL is **publicly accessible** (Mollie only)

### "ENCRYPTION_KEY invalid or missing"

**Symptom:** Cannot store/retrieve IBAN secrets

**Cause:** `ENCRYPTION_KEY` not set or invalid base64

**Fix:**
1. Generate a new key:
   ```bash
   openssl rand -base64 32
   ```
2. Set in `.env`:
   ```bash
   ENCRYPTION_KEY=<generated-key>
   ```
3. Restart the application

### How to Check Configuration at Runtime

**Check loaded providers:**
```bash
# Logs during startup show which providers are configured
docker logs <container-id> | grep -i "payment.*provider"

# Example output:
# payment: stripe (sandbox) configured
# payment: mollie (sandbox) configured
# payment: rib_iban configured (encrypted)
```

**Check webhook registration:**
```bash
# Stripe: test webhook delivery
# Stripe Dashboard → Developers → Webhooks → Select endpoint → Send test event

# Mollie: check webhook logs
# Mollie Dashboard → Webhooks → View logs

# PayPal: check webhook logs
# PayPal Dashboard → Apps & Credentials → Webhooks → Select webhook → Event logs
```

---

## Next Steps

1. **Choose payment methods** for your deployment
2. **Obtain credentials** from each provider's dashboard
3. **Create `payment-providers.json`** with your credentials
4. **Set `ENCRYPTION_KEY`** and `PAYMENT_PROVIDERS` in `.env`
5. **Test each method** in sandbox mode
6. **Register webhooks** in each provider's dashboard
7. **Verify webhook delivery** using provider test tools
8. **Deploy to staging** and run end-to-end payment tests
9. **Activate live credentials** for production deployment

---

## Support & References

- **Stripe Docs:** https://stripe.com/docs/payments
- **Mollie Docs:** https://docs.mollie.com/
- **PayPal Docs:** https://developer.paypal.com/docs/
- **IBAN Validation:** https://www.iban.com/structure
- **RFC 3339 (Timestamps):** https://tools.ietf.org/html/rfc3339
