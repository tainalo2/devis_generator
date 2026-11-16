/**
 * Signature Proof Viewer Component
 * 
 * Modal/popover to view and download signature proof metadata:
 * - signed_at: Horodatage serveur (UTC)
 * - document_hash: SHA-256 du contenu figé
 * - signer_ip: IP de la requête
 * - signer_user_agent: User-agent du navigateur
 * - signer_name_declared: Identité déclarée (non vérifiée)
 * 
 * This component will be triggered from signature badges or detail view.
 */

import { Download, Eye } from "lucide-react"
import { useState } from "react"
import { Button } from "@/components/ui/button"
import { formatDateTime } from "@/lib/documents"
import { downloadBlob } from "@/lib/api"

export interface SignatureProof {
  signed_at: string
  signer_name_declared: string
  document_hash?: string
  signer_ip?: string
  signer_user_agent?: string
}

export function SignatureProofViewer({ proof }: { proof: SignatureProof }) {
  const [open, setOpen] = useState(false)

  const downloadProof = () => {
    const content = `Signature Proof
===============
Signed at:        ${formatDateTime(proof.signed_at)}
Signer name:      ${proof.signer_name_declared}
Document hash:    ${proof.document_hash || "N/A"}
Signer IP:        ${proof.signer_ip || "N/A"}
Signer User-Agent: ${proof.signer_user_agent || "N/A"}

Note: This is a simple electronic signature (eIDAS, art. 3(10)) without qualified certificate.
It proves timestamp, document integrity, and technical traceability, but not verified signer identity.
`
    const blob = new Blob([content], { type: "text/plain;charset=utf-8" })
    downloadBlob(blob, `signature-proof-${new Date(proof.signed_at).getTime()}.txt`)
  }

  return (
    <div>
      <Button
        variant="ghost"
        size="sm"
        onClick={() => setOpen(!open)}
        className="gap-1"
      >
        <Eye className="size-3.5" />
        Consulter la preuve
      </Button>

      {open && (
        <div className="rounded-lg border border-border bg-card p-4 mt-2 space-y-2 text-sm">
          <div>
            <span className="text-muted-foreground">Signé le:</span>
            <div className="font-mono text-xs mt-1">{formatDateTime(proof.signed_at)}</div>
          </div>
          <div>
            <span className="text-muted-foreground">Par (déclaré):</span>
            <div className="font-mono text-xs mt-1">{proof.signer_name_declared}</div>
          </div>
          {proof.document_hash && (
            <div>
              <span className="text-muted-foreground">Hash du document (SHA-256):</span>
              <div className="font-mono text-xs mt-1 break-all">{proof.document_hash}</div>
            </div>
          )}
          {proof.signer_ip && (
            <div>
              <span className="text-muted-foreground">Adresse IP:</span>
              <div className="font-mono text-xs mt-1">{proof.signer_ip}</div>
            </div>
          )}
          {proof.signer_user_agent && (
            <div>
              <span className="text-muted-foreground">User-Agent:</span>
              <div className="font-mono text-xs mt-1 break-all text-[11px]">{proof.signer_user_agent}</div>
            </div>
          )}
          <div className="pt-2 border-t border-border flex gap-2">
            <Button size="sm" variant="outline" onClick={downloadProof}>
              <Download className="size-3.5" />
              Télécharger
            </Button>
          </div>
          <div className="text-xs text-muted-foreground pt-2 border-t border-border">
            <p>Signature électronique simple (eIDAS, art. 3(10)) sans certificat qualifié.</p>
            <p className="mt-1">Prouve: horodatage, intégrité du document, traçabilité technique.</p>
            <p className="mt-1">Ne prouve pas: identité vérifiée du signataire.</p>
          </div>
        </div>
      )}
    </div>
  )
}
