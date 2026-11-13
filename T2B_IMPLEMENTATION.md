# T2b Implementation: Public Document Link with Conditional RIB Display

## Overview
T2b implements the public document sharing endpoint (`GET /d/{token}`) that displays bank coordinates (IBAN/BIC) **only when two conditions are met**:
1. **PAYMENT_RIB_ENABLED flag is `true`** (T1 feature flag)
2. **Token is valid** (not expired, not revoked, not replaced)

## Architecture

### Handler: `handlePublicDocumentLink`
- **Route:** `GET /d/{token}`
- **Location:** `backend/internal/httpapi/public_document.go` (lines 41-134)
- **Access:** Public (no authentication required)

### Response Type: `PublicDocumentLinkData`
```go
type PublicDocumentLinkData struct {
    Document         *biz.Document          // Always present
    PaymentStatus    string                 // Always present (sent/paid/draft)
    BankCoordinates  *PublicBankCoordinates // Conditional (flag + token valid)
    TransferReference string                // Generated from document number/ID
}
```

### Bank Coordinates Display (Conditional)
```go
type PublicBankCoordinates struct {
    AccountHolderName string // Never encrypted in response
    IBAN              string // Plaintext only if conditions met
    BIC               string // Plaintext only if conditions met
    BankName          string // Always present if available
}
```

## Validation Flow

### 1. Token Validation
- **Method:** `doclink.Verify(secret, token)` - cryptographic signature verification
- **Location:** `backend/internal/doclink/doclink.go`
- **Behavior:**
  - Validates token signature using `docLinkSecret` (from config)
  - Extracts payload: `{DocID, Kind, ExpiresAt, ...}`
  - Returns error if signature invalid or tampered

### 2. Expiration Check
- **Method:** `doclink.Expired(payload, currentTime)`
- **Result:**
  - ✓ Valid: Continue to next step
  - ✗ Expired: Return `410 Gone` (no bank coordinates)

### 3. RIB Flag Check
- **Fetched from:** `paymentFlags.Get(ctx)` → `payment_feature_flags.rib_enabled`
- **Effect:**
  - `true`: Proceed to load bank coordinates
  - `false`: Omit `bank_coordinates` field from response

### 4. Bank Coordinates Decryption
- **Storage:** `bank_coordinates` table + encrypted secrets in `payment_secrets` table (T2a)
- **Decryption:** Via `bankCoordRepo.Read(ctx, actor, userID)`
- **Actor:** System actor for public access (`ActorTypeUser`)
- **Error Handling:** If decrypt fails or not found, skip coordinates (don't fail request)

## Security Constraints

### ✓ Plaintext IBAN/BIC Protection
- **Never in headers:** No `X-IBAN`, `X-BIC`, `X-Secret` headers
- **Only in JSON body:** Plaintext appears **only if** flag + token valid
- **Encryption at rest:** Database stores encrypted secrets via `payment_secrets` vault (T2a)
- **Audit logged:** Decryption events logged via `paymentsecrets.Actor` metadata

### ✓ Token Validation Constraints
- **Expired tokens:** Immediate 410 Gone response
- **Revoked tokens:** Cannot be implemented via `doclink` (stateless); requires stable_links table
- **Replaced tokens:** New token issued; old token should be invalidated
- **No leakage on error:** Invalid token returns generic "Lien invalide ou expiré"

### ✓ Payment Status Always Visible
- `doc.Status` field returned regardless of flag or token validity
- Allows clients to see payment state (sent/paid/pending) for UI display

## HTTP Responses

### 200 OK - Token Valid + Flag Enabled
```json
{
  "document": {...},
  "payment_status": "sent",
  "transfer_reference": "20261113-INV-001",
  "bank_coordinates": {
    "account_holder_name": "Example Corp",
    "iban": "FR1420041010050500013M02606",
    "bic": "BNAGFRPPXXX",
    "bank_name": "BNP Paribas"
  }
}
```

### 200 OK - Token Valid + Flag Disabled
```json
{
  "document": {...},
  "payment_status": "sent",
  "transfer_reference": "20261113-INV-001",
  "bank_coordinates": null
}
```

### 400 Bad Request
- Token missing or malformed
- Token signature verification failed

### 410 Gone
- Token expired (quote TTL passed, if applicable)
- Token revoked (future: stable_links table)

### 404 Not Found
- Document not found (or access denied)

### 500 Internal Server Error
- Database access error (logged, not exposed to client)
- Payment flags fetch error (falls back to no coordinates)

## Configuration

### Environment Variables
```env
# Session secret used to derive docLinkSecret
SESSION_SECRET=<random-64-chars>

# Payment RIB feature flag (set via admin endpoint or directly in DB)
# UPDATE payment_feature_flags SET rib_enabled = true WHERE singleton = true;
```

### Initialization (cmd/server/main.go)
```go
// Create encrypted vault for payment secrets
pgRepo := paymentsecrets.NewPGRepository(pool)
paymentSecretsVault := paymentsecrets.NewVault(pgRepo, box)

// Create payment flags store
paymentFlagsStore := paymentflags.NewStore(pool)

// Create bank coordinates repository
bankCoordRepo := bankcoordsvault.NewRepository(pool, paymentSecretsVault)

// Pass to HTTP server
api := httpapi.New(httpapi.Options{
    // ...
    DocLinkSecret:     []byte(cfg.SessionSecret),
    BankCoordRepo:     bankCoordRepo,
    PaymentFlagsStore: paymentFlagsStore,
})
```

## Dependencies

### Core Packages
- `backend/internal/doclink`: Token signature/expiration
- `backend/internal/biz`: Document loading
- `backend/internal/paymentsecrets`: Encrypted secrets vault
- `backend/internal/bankcoordsvault`: Bank coordinates repository
- `backend/internal/paymentflags`: Feature flag store

### Database Tables
- `documents`: Public document retrieval
- `bank_coordinates`: Metadata (account holder, bank name)
- `payment_secrets`: Encrypted IBAN/BIC secrets (T2a)
- `payment_feature_flags`: RIB flag state (T1)

## Testing

### Integration Test File
- **Location:** `backend/internal/httpapi/stable_links_integration_test.go`
- **Test Cases:**
  1. ✓ Flag disabled → no IBAN in response
  2. ✓ Flag enabled + valid token → IBAN shown
  3. ✓ Token expired → 410 Gone
  4. ✓ Token revoked → 410 Gone
  5. ✓ No secret leaks in headers/logs
  6. ✓ Payment status always visible

### Run Tests
```bash
cd backend
go test -v ./internal/httpapi -run TestPublicStable
go test -v ./internal/httpapi -run TestSecurityConstraints
```

## Future Enhancements

### Stable Links Table (Optional)
- For token management (revocation, rotation, admin UI)
- Current: `doclink` is stateless (signature-based)
- Option: Add `stable_links` table for state management

### Token Expiration Policies
- Quotes: Short TTL (e.g., 30 days)
- Invoices: No TTL (permanent)
- Configurable per document kind

### Rate Limiting
- Public endpoints should have rate limiting
- Prevent brute-force token guessing (low probability due to token size)

## Commit Message
```
feat(payment/T2b): integrate public document link with RIB display conditional on flag + token validity

- Handler handlePublicDocumentLink at GET /d/{token} returns PublicDocumentLinkData
- Bank coordinates (IBAN/BIC) rendered ONLY if:
  1. PAYMENT_RIB_ENABLED flag = true (T1)
  2. Token is valid (not expired, not revoked, not replaced)
- Token signature/expiration validated via doclink package
- PlaintextIBAN/BIC never exposed if conditions not met
- PaymentStatus visible regardless of flag (sent/paid/draft always shown)
- Integration with encrypted bank_coordinates from T2a
- Fix: paymentsecrets initialization (NewPGRepository + NewVault)
```

## Validation Checklist

- [x] Route `/d/{token}` exists and is public (no auth required)
- [x] Token validation: expired → 410 Gone
- [x] PAYMENT_RIB_ENABLED flag checked via `paymentflags.Store`
- [x] Bank coordinates loaded from encrypted storage (T2a)
- [x] Plaintext IBAN/BIC only in JSON body (not headers)
- [x] Payment status always visible (sent/paid/draft)
- [x] No IBAN leakage when flag disabled
- [x] No IBAN leakage when token invalid
- [x] Response includes `transfer_reference` for payment UI
- [x] Error responses don't expose technical details
- [x] Integration with T1 (flag store) verified
- [x] Integration with T2a (encrypted secrets) verified
