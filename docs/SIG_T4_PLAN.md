# Sig-T4 Implementation Plan — Emitter-Side Signed Status Display

## Status

**Task:** Sig-T4 (emitter-side signed status badge + proof viewer)
**Current:** Blocked waiting for Sig-T2 (data model) and Sig-T3 (signature UI) to complete
**Architecture decision:** Sig-T1 completed — simple eIDAS signature (art. 3(10)), no qualified certificate, no third-party API in V1

---

## Scope (from task body + DECISIONS.md § "Signature électronique (V1)")

### 1. Signed Status Badge ("Signé le XX par YY")
- **Document list page** (`documents-page.tsx`): Add signature badge to document rows that have a signature
- **Document detail page** (`document-view.tsx`): Display signature badge in the status line (alongside existing "Lock — non modifiable" indicator)
- Format: "Signé le [DATE] par [SIGNER_NAME]" (e.g., "Signé le 16 nov. 2026 par Jean Dupont")
- Icon: FileCheck (lucide-react) or similar
- Styling: Use existing Badge component (`variant="secondary"` or appropriate variant)

### 2. Proof Viewer / Downloader
- **Modal/popover** triggered from signature badge or dedicated "Consulter la preuve" button
- Display metadata (read-only):
  - `signed_at` — Timestamp (server UTC, formatted with date and time)
  - `signer_name_declared` — Signer name (declared, not verified)
  - `document_hash` — SHA-256 hex string (copy-pasteable, read-only)
  - `signer_ip` — IP address of signature request
  - `signer_user_agent` — Browser user-agent
- **Download button** to export as plain-text file (`.txt`) with structured fields + legal disclaimer

### 3. Non-Regression: Signed Documents Immutable
- **On document detail view**: If `document.signed_at` is set, the document is immuable
  - Existing lock indicator + "non modifiable" text already present
  - Ensure no edit/replace/delete buttons are shown for signed documents
  - **Signed devis** = same rule as `issued` invoices/deposits/credits (already locked)
- **API level**: `/api/v1/documents/{id}` PATCH/PUT/DELETE should return 409 Conflict or 400 Bad Request if document is signed

---

## Dependencies

### Sig-T2 (Data Model) — Status: Blocked (waiting for decision comment, now received)
**Delivers:**
- Database migration `000022_signatures.sql` with fields:
  - `signed_at TIMESTAMP WITH TIME ZONE` (on `documents` table or `signatures` table)
  - Signature record table: `document_id`, `signer_name_declared`, `signed_at`, `document_hash`, `signer_ip`, `signer_user_agent`
- Backend service `RecordSignature()` and `GetSignature()` in `biz.Service`
- Backend model `Signature` struct with correct field names

### Sig-T3 (Signature UI) — Status: Running
**Delivers:**
- API endpoint `POST /api/v1/documents/{token}/sign` (public link signing)
- Frontend component or modal for signature capture on `/d/{token}` page
- Request fields: `signer_name_declared` (minimum; may also capture email)
- Response: success, signed_at, signature_id

---

## Implementation Plan for Sig-T4

### Phase 1: Integrate Sig-T2 Data Model
1. Update `Document` type (frontend `src/lib/types.ts`) to include:
   - `signed_at?: string | null`
   - `signature?: { signed_at: string; signer_name_declared: string; document_hash?: string; signer_ip?: string; signer_user_agent?: string }`
2. Update backend `Document` struct (if needed) to include signature relation

### Phase 2: Integrate Sig-T3 UI State
1. After signature is successfully recorded (Sig-T3 endpoint returns), update document state to reflect `signed_at` and signature metadata

### Phase 3: Display Badge + Proof Viewer (Frontend)
1. `document-view.tsx` — Add signature badge to `statusLine` (line 391–399):
   ```tsx
   const statusLine = (
     <>
       <Badge variant={st.variant}>{st.label}</Badge>
       {doc.signed_at && doc.signature && (
         <SignatureBadge signature={doc.signature} />
       )}
       <span className="flex items-center gap-1.5 text-xs whitespace-nowrap text-muted-foreground">
         <Lock className="size-3.5" aria-hidden />
         Émis{e} le {formatDate(doc.issue_date)} · non modifiable
       </span>
     </>
   )
   ```
2. `documents-page.tsx` — Add badge to document list rows (in table or card) if `doc.signed_at` is set
3. Add `SignatureProofViewer` component (already created) or inline proof display near badge

### Phase 4: Lock Down Signed Documents (Non-Regression)
1. `document-view.tsx` — Hide edit/replace buttons if `doc.signed_at` is set:
   ```tsx
   const primary = ...
   if (doc.signed_at) {
     // Hide mutating actions for signed documents
     // Keep only: download PDF, send email, view proof
   }
   ```
2. Backend validation (if not already done by Sig-T2):
   - `PATCH /api/v1/documents/{id}` → 409 if `signed_at` is set
   - `PUT /api/v1/documents/{id}` → 409 if `signed_at` is set
   - Replace flow (`POST /api/v1/documents/{id}/replace`) → warn if original is signed (consistent with "document replaced → explicit message" rule)

---

## Already Completed
- ✅ Architecture decision documented (Sig-T1 → DECISIONS.md)
- ✅ Skeleton React components created:
  - `frontend/src/components/signature-badge.tsx` — `SignatureBadge` component
  - `frontend/src/components/signature-proof-viewer.tsx` — `SignatureProofViewer` component
- ✅ Epic branch created: `epic/signature-status-emitter-t_b6efce0a`

---

## Notes
- **eIDAS compliance**: This is a simple signature (art. 3(10)) without qualified certificate. The proof viewer displays a legal disclaimer stating this.
- **Field naming**: Architecture specifies `signer_name_declared` (not `signer_name` or `signer_email`). Sig-T2 migration must use correct names or backend mapping.
- **Non-blocking email**: Signature confirmation email to issuer (Sig-T3 scope) should be non-blocking (async/background) so signature record is persisted even if email fails.
