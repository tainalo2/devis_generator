/**
 * Integration helpers for Sig-T4 (Emitter-side signature display)
 * 
 * These utilities will be used to integrate signature data from Sig-T2 data model
 * and Sig-T3 UI endpoints into document views.
 * 
 * Ready to use once Sig-T2 and T3 deliver their PRs.
 */

import type { Document } from "@/lib/types"
import type { SignatureInfo } from "@/components/signature-badge"
import type { SignatureProof } from "@/components/signature-proof-viewer"

/**
 * Extract signature info from document for badge display.
 * Returns null if document is not signed.
 * 
 * Note: This function uses type assertions because the signature fields
 * are added by Sig-T2 migration. After Sig-T2 merges, update Document type
 * in src/lib/types.ts to include these fields natively.
 */
export function extractSignatureInfo(doc: Document): SignatureInfo | null {
  // After Sig-T2 delivers, documents will have:
  // - signed_at?: string | null
  // - signature?: { signed_at: string; signer_name_declared: string; document_hash?: string; signer_ip?: string; signer_user_agent?: string }
  
  const docWithSig = doc as any
  if (!docWithSig.signed_at || !docWithSig.signature) {
    return null
  }
  
  const sig = docWithSig.signature
  return {
    signed_at: sig.signed_at,
    signer_name_declared: sig.signer_name_declared,
    document_hash: sig.document_hash,
    signer_ip: sig.signer_ip,
    signer_user_agent: sig.signer_user_agent,
  }
}

/**
 * Extract proof data from document for viewer display.
 * Same as signature info but typed for the proof viewer.
 */
export function extractSignatureProof(doc: Document): SignatureProof | null {
  return extractSignatureInfo(doc) as SignatureProof | null
}

/**
 * Check if a document is signed (immutable).
 */
export function isSigned(doc: Document): boolean {
  return !!(doc as any).signed_at
}

/**
 * Check if document can be edited/deleted.
 * Signed documents cannot be mutated (same rule as issued documents).
 */
export function canMutateDocument(doc: Document): boolean {
  return !isSigned(doc)
}

/**
 * API validation: Check response from Sig-T3 signature endpoint.
 * Returns true if response indicates successful signature recording.
 */
export function isSignatureSuccessResponse(response: unknown): boolean {
  const r = response as any
  return !!r?.success && !!r?.signature_id && !!r?.signed_at
}

/**
 * Transform Sig-T3 response into document update.
 * Call after successful signature request to update document state.
 * 
 * This helper bridges the Sig-T3 response format with the Document model,
 * allowing document detail view to update its state after signing.
 */
export function transformSignatureResponse(doc: Document, response: any): Document {
  const docWithSig = doc as any
  return {
    ...doc,
    signed_at: response.signed_at || new Date().toISOString(),
    signature: {
      signed_at: response.signed_at || new Date().toISOString(),
      signer_name_declared: response.signer_name_declared || "Signataire",
      document_hash: response.document_hash,
      signer_ip: response.signer_ip,
      signer_user_agent: response.signer_user_agent,
    },
  }
}
