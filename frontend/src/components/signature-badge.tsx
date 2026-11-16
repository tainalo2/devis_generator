/**
 * Signature Badge Component
 * 
 * Displays "Signed on [DATE] by [NAME]" badge for documents with signature status.
 * This component will be used in:
 * - Document list (documents-page.tsx)
 * - Document detail (document-view.tsx) 
 * 
 * Requirements from Sig-T4:
 * - Show badge on both document list and detail pages
 * - Format: "Signé le XX par YY"
 * - Include proof viewer/downloader for signature metadata
 */

import { FileCheck } from "lucide-react"
import { formatDate } from "@/lib/documents"
import { Badge } from "@/components/ui/badge"

export interface SignatureInfo {
  signed_at: string
  signer_name_declared: string
  document_hash?: string
  signer_ip?: string
  signer_user_agent?: string
}

export function SignatureBadge({ signature }: { signature: SignatureInfo }) {
  const dateStr = signature.signed_at.split('T')[0]
  const label = `Signé le ${formatDate(dateStr)} par ${signature.signer_name_declared}`
  
  return (
    <Badge variant="secondary" className="flex items-center gap-1">
      <FileCheck className="size-3.5" />
      {label}
    </Badge>
  )
}
