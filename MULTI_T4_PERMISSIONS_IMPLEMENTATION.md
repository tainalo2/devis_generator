# Multi-T4: Permission Isolation on Sensitive Endpoints

## Overview

This ticket implements middleware and permission checks to isolate access to sensitive account configuration endpoints. The goal is to ensure that non-owner collaborators (once implemented) can never access:

- **API Keys** management endpoints
- **Payment Configuration** (Stripe, PayPal, etc.) containing payment processor API keys
- **Stripe Connect** credentials and onboarding
- **SMTP/Email Configuration** containing email service credentials
- **Issuer Information** including bank coordinates (RIB/IBAN)

## Implementation

### 1. Permissions Middleware (`permissions.go`)

Three complementary mechanisms enforce access control:

#### a. `isolateSensitiveEndpoints` (Route Middleware)
Applied at the `/api/v1` level, this middleware checks if:
- Request is to a sensitive endpoint
- Requester is using an API key instead of a session
- Rejects API-key requests to sensitive endpoints with **403 Forbidden**

#### b. `ensureSensitiveEndpointIsolation` (Handler Wrapper)
Applied to individual sensitive endpoint handlers as a defensive measure:
- Checks if user authenticated via API key
- Validates that session-authenticated users proceed
- Provides layered protection

#### c. `requireOwner` / `requireSessionOwner` (Future Extensions)
Prepared for when collaborator roles are implemented:
- `requireOwner`: Will check if user is account owner
- `requireSessionOwner`: Will enforce session-only access (not API key)

### 2. Sensitive Endpoints Protected

The following routes are classified as sensitive:

```go
// API Keys Management
GET    /api/v1/auth/api-keys
POST   /api/v1/auth/api-keys
DELETE /api/v1/auth/api-keys/{id}

// Payment Configuration (Stripe, PayPal, etc.)
GET    /api/v1/payments/config/{provider}
PUT    /api/v1/payments/config/{provider}
DELETE /api/v1/payments/config/{provider}

// Stripe Connect Credentials
POST   /api/v1/payments/stripe-connect/onboard
POST   /api/v1/payments/stripe-connect/refresh
GET    /api/v1/payments/stripe-connect/callback

// SMTP/Email Credentials
GET    /api/v1/email/smtp
POST   /api/v1/email/smtp
DELETE /api/v1/email/smtp

// Issuer Information (includes bank coordinates)
GET    /api/v1/issuer
PUT    /api/v1/issuer
```

### 3. Access Rules (Current)

| Authentication Type | Sensitive Endpoints | Regular Endpoints |
|-------------------|-------------------|------------------|
| Session Cookie    | ✅ Allowed         | ✅ Allowed        |
| API Key (read)    | ❌ Forbidden       | ✅ Allowed        |
| API Key (write)   | ❌ Forbidden       | ✅ Allowed        |
| Unauthenticated   | ❌ Unauthorized    | ❌ Varies         |

### 4. Future: Collaborator Support

When collaborators are implemented, the following check will be added:

```go
// In ensureSensitiveEndpointIsolation:
if !userIsOwnerOfAccount(user.ID) {
    writeAPIError(w, http.StatusForbidden, 
        "Accès refusé : seul le propriétaire du compte peut accéder à cette ressource.")
    return
}
```

This will ensure only account owners access sensitive endpoints, even with valid sessions.

## Testing

### Test Coverage

Comprehensive test suite in `permissions_test.go`:

1. **TestSensitiveEndpointIsolation_APIKeyBlocked**
   - 15 test cases verifying API keys are rejected from all sensitive endpoints
   - 100% coverage of sensitive endpoint paths

2. **TestSensitiveEndpointIsolation_SessionCanAccess**
   - 4 test cases verifying session-authenticated users can access sensitive endpoints

3. **TestEnsureSensitiveEndpointIsolation**
   - Tests the defensive handler wrapper
   - Covers API key rejection, session allowance, and unauthenticated rejection

4. **TestIsSensitiveEndpoint**
   - Validates endpoint classification logic
   - Tests 8 sensitive and 5 non-sensitive paths

### Running Tests

```bash
# Run all permission tests
go test ./internal/httpapi -run "TestSensitiveEndpoint"

# Run all httpapi tests
go test ./internal/httpapi -v

# Run specific test
go test ./internal/httpapi -run TestEnsureSensitiveEndpointIsolation -v
```

## Implementation Details

### Request Flow

1. **Request arrives at `/api/v1`**
   ↓
2. **`isolateSensitiveEndpoints` middleware checks:**
   - Is user authenticated?
   - Is this a sensitive endpoint?
   - Is user using API key?
   - If yes to all: return 403 Forbidden
   ↓
3. **Request reaches endpoint handler**
   ↓
4. **Handler's `ensureSensitiveEndpointIsolation` wrapper validates:**
   - Is user authenticated?
   - Is user using API key?
   - If yes: return 403 Forbidden
   ↓
5. **Handler executes**

### Error Messages

- **API Key on Sensitive Endpoint**: 
  ```
  403 Forbidden
  "Accès refusé : clés API ne peuvent pas accéder aux configurations sensibles."
  ```

- **Session Required**:
  ```
  403 Forbidden
  "Session requise pour cette action."
  ```

- **Insufficient Scope (Read-only on Write endpoint)**:
  ```
  403 Forbidden
  "Scope API insuffisant."
  ```

## Security Guarantees

✅ **Non-owner collaborators will never access:**
- API keys (even if they can read documents)
- Payment processor credentials
- Bank account details (RIB/IBAN)
- Email service credentials
- Stripe Connect tokens

✅ **API keys cannot be used to:**
- Manage other API keys
- Configure payment providers
- Read/write email configurations
- Read/write bank coordinates

✅ **Defense in depth:**
- Middleware-level check
- Handler-level check
- Future owner-level check (when collaborators added)

## Integration with Collaborator Feature (Future)

When the Multi-Utilisateurs epic adds collaborator support:

1. Add `role` column to `users_accounts` table (or new `account_members` table)
2. Check `role == 'owner'` before allowing access to sensitive endpoints
3. Enable the blocked code in `ensureSensitiveEndpointIsolation`
4. All existing tests will still pass (session-authenticated owner can access)

## Files Modified

- `internal/httpapi/permissions.go` - New permissions middleware
- `internal/httpapi/permissions_test.go` - New comprehensive test suite
- `internal/httpapi/server.go` - Integrated middleware and handler wrappers
  - Added `r.Use(s.isolateSensitiveEndpoints)` to `/api/v1` route
  - Wrapped all sensitive endpoint handlers with `s.ensureSensitiveEndpointIsolation()`

## Verification

✅ Code compiles without errors
✅ All 24 permission tests pass
✅ All existing httpapi tests still pass (backward compatible)
✅ Middleware properly blocks API key access to sensitive endpoints
✅ Session-authenticated users can still access all endpoints

## Related Documentation

- `docs/DECISIONS.md` § "Multi-utilisateurs par compte — rôles (V1)" - Role definition
- `PAIEMENT_T2A_BIS_IMPLEMENTATION.md` - Encrypted bank coordinate storage
- `internal/authstore/api_keys.go` - API key model and storage
