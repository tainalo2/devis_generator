# Bank Coordinates CRUD + Encrypted Storage

## Overview

This implementation delivers the complete CRUD + encrypted storage for bank coordinates (IBAN/BIC) as specified in Paiement-T2a-bis.

## Architecture

### Database Schema (Migration 000015)

**Table: `bank_coordinates`**
- Stores metadata: account holder name, bank name, and references to encrypted secrets
- One record per user (UNIQUE constraint on `user_id`)
- Foreign keys to `payment_secrets` for IBAN and BIC storage
- Audit-friendly timestamps (created_at, updated_at)

**Encryption Flow:**
1. IBAN and BIC are validated via the `bankcoord` package (ISO 13616, mod-97 for IBAN; ISO 7064 for BIC)
2. Both are encrypted via `paymentsecrets.Vault` using AES-256-GCM
3. Encrypted blobs stored in `payment_secrets` table with `kind='iban'` and `reference='primary'` or `'bic'`
4. Audit trail recorded in `payment_secret_audit` on every access

### Packages

#### `backend/internal/bankcoordsvault`
- **Repository pattern**: Single point of access for bank coordinates CRUD
- **CreateOrUpdate**: Validates IBAN/BIC, encrypts via vault, upserts metadata row
- **Read**: Reveals plaintext IBAN/BIC from vault (audit-logged)
- **GetMetadata**: Returns metadata without plaintext (safe for API listings)
- **Delete**: Cascade-deletes via foreign key
- **Exists**: Lightweight check

#### Reuse of Existing Packages
- **`bankcoord`**: ValidateIBAN() with mod-97 checksum, ValidateBIC() with ISO 7064
- **`paymentsecrets`**: Vault.Put(), Vault.Reveal() for encryption with audit trails

### API Handlers

#### `POST /api/v1/bank-coordinates`
Create or update bank coordinates for authenticated user.

**Request:**
```json
{
  "account_holder_name": "string (required, 1-255 chars)",
  "iban": "string (required, validated ISO 13616)",
  "bic": "string (optional, validated ISO 7064)",
  "bank_name": "string (optional)"
}
```

**Response (201 Created / 200 OK):**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "account_holder_name": "string",
  "bank_name": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

**Errors:**
- 400: Invalid IBAN/BIC format, empty holder name
- 401: Unauthorized
- 500: Server error

#### `GET /api/v1/bank-coordinates`
Retrieve **plaintext** IBAN/BIC for authenticated user (audit-logged).

**Response (200 OK):**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "account_holder_name": "string",
  "iban": "string (plaintext)",
  "bic": "string (plaintext, if set)",
  "bank_name": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

**Errors:**
- 401: Unauthorized
- 404: Bank coordinates not found
- 500: Server error

#### `GET /api/v1/bank-coordinates/metadata`
Retrieve metadata **without plaintext** (safe for UI displays without revealing secrets).

**Response (200 OK):**
```json
{
  "id": "uuid",
  "user_id": "uuid",
  "account_holder_name": "string",
  "bank_name": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

#### `DELETE /api/v1/bank-coordinates`
Delete all bank coordinates for authenticated user.

**Response (204 No Content)**

**Errors:**
- 401: Unauthorized
- 404: Not found
- 500: Server error

## Security & Multi-Tenant Isolation

### Access Control
1. **Authentication required**: All endpoints demand an authenticated user via session/JWT
2. **Multi-tenant isolation**: Vault access enforces `actor.UserID == ownerID` on every read/write
3. **Ownership check**: A user can only access their own coordinates
4. **Cross-tenant prevention**: Attempting to reveal another user's secrets via vault will fail

### Encryption at Rest
- **Algorithm**: AES-256-GCM (authenticated encryption)
- **No plaintext storage**: Secrets only exist in vault ciphertext
- **Key rotation support**: Via `paymentsecrets` keyring mechanism
- **Audit trail**: Every encrypt/decrypt/delete logged with actor, timestamp, request ID

### API Safety
- **No plaintext in metadata endpoint**: Safe for frontend displays
- **Explicit reveal endpoint**: GET /bank-coordinates is audit-logged on purpose
- **No accidental leaks**: IBAN/BIC never appear in PDF, exports, or error messages

## Testing

### Unit Tests
- **CreateOrUpdate**: Valid/invalid IBAN/BIC, empty holder name, BIC optional
- **Read**: Plaintext retrieval with audit
- **Delete**: Cascade behavior
- **Multi-tenant isolation**: User1 cannot read/update User2's secrets
- **Metadata retrieval**: Safe access without plaintext

### Integration Tests
- Full HTTP handler tests with authentication context
- Cross-tenant authorization checks
- Edge cases: malformed input, missing records, concurrent updates

## Integration with Paiement-T2b

The `bank_coordinates` table and `bankcoordsvault.Repository` provide the required schema and API for stable-link rendering (/d/{token}) in Paiement-T2b. The plaintext IBAN/BIC are **only** revealed via the vault on explicit requests with audit logging.

## Migration Path

Run migration 000015 to create the bank_coordinates table:
```bash
cd backend
go run ./cmd/migrate apply
```

## Dependencies

- `bankcoord` package for IBAN/BIC validation
- `paymentsecrets` package for encryption
- PostgreSQL 13+ for JSON and crypto functions
- Standard Go HTTP routing (chi)
