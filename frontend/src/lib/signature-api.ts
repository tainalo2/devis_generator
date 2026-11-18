// Signature-related API types and functions

import { apiFetch, PUBLIC_LINK_UNAVAILABLE } from "./api"

// Local error reading helper
async function readError(res: Response, fallback: string): Promise<string> {
  try {
    const data = (await res.json()) as { error?: string; message?: string }
    return data.error || data.message || fallback
  } catch {
    return fallback
  }
}

export interface DocumentSignatureState {
  document_id: string
  status: "unsigned" | "signed" | "locked"
  signed_at?: string
  signed_by_email?: string
  is_locked: boolean
  proof_token?: string
}

export interface SignatureProof {
  document_id: string
  signer_email: string
  signed_at: string
  signature_hash: string
  proof_token: string
  created_at: string
}

/**
 * Get the signature state of a document via public link
 */
export async function fetchDocumentSignatureState(token: string): Promise<DocumentSignatureState> {
  const res = await apiFetch(`/api/v1/public/documents/${encodeURIComponent(token)}/signature`)
  if (!res.ok) {
    if (res.status === 404 || res.status === 410) throw new Error(PUBLIC_LINK_UNAVAILABLE)
    throw new Error("État de signature indisponible.")
  }
  return (await res.json()) as DocumentSignatureState
}

/**
 * Sign a document via public link
 */
export async function signDocument(
  token: string,
  signerEmail: string,
  signerName?: string
): Promise<{ document_id: string; signed_at: string; signer_email: string; proof_token: string }> {
  const res = await apiFetch(`/api/v1/public/documents/${encodeURIComponent(token)}/sign`, {
    method: "POST",
    body: JSON.stringify({
      signer_email: signerEmail,
      signer_name: signerName,
    }),
  })
  if (!res.ok) {
    const status = res.status
    if (status === 404 || status === 410) throw new Error(PUBLIC_LINK_UNAVAILABLE)
    if (status === 409) throw new Error("Ce document est déjà signé et verrouillé.")
    throw new Error(await readError(res, "Signature impossible."))
  }
  return (await res.json()) as {
    document_id: string
    signed_at: string
    signer_email: string
    proof_token: string
  }
}

/**
 * Get a signature proof by its token (public endpoint, no auth required)
 */
export async function fetchSignatureProof(proofToken: string): Promise<SignatureProof> {
  const res = await apiFetch(`/api/v1/public/signature-proof/${encodeURIComponent(proofToken)}`)
  if (!res.ok) {
    if (res.status === 404) throw new Error("Preuve de signature introuvable.")
    throw new Error("Vérification de signature impossible.")
  }
  return (await res.json()) as SignatureProof
}

/**
 * Get the signature state of a document for authenticated user
 */
export async function fetchDocumentSignatureStateForUser(documentId: string): Promise<DocumentSignatureState> {
  const res = await apiFetch(`/api/v1/documents/${encodeURIComponent(documentId)}/signature`)
  if (!res.ok) throw new Error(await readError(res, "État de signature introuvable."))
  return (await res.json()) as DocumentSignatureState
}
