import { ShieldAlert, ShieldCheck } from "lucide-react"
import { useEffect, useState } from "react"

import { getSignatureProof, type SignatureProof } from "@/lib/api"
import { formatDateTime } from "@/lib/documents"

/** Preuve de signature consultable par l'émetteur, avec contrôle d'intégrité du contenu figé. */
export function SignatureProofBanner({ documentId }: { documentId: string }) {
  const [proof, setProof] = useState<SignatureProof | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let active = true
    void getSignatureProof(documentId)
      .then((p) => active && setProof(p))
      .catch((err: unknown) => active && setError(err instanceof Error ? err.message : "Preuve indisponible."))
    return () => {
      active = false
    }
  }, [documentId])

  if (error) return <p className="border-b border-border bg-card px-4 py-2 text-xs text-destructive">{error}</p>
  if (!proof) return null
  const Icon = proof.integrity_ok ? ShieldCheck : ShieldAlert
  return (
    <section aria-label="Preuve de signature" className="flex flex-col gap-1 border-b border-border bg-card px-4 py-2 text-xs lg:px-6">
      <p className="flex items-center gap-1.5 font-medium">
        <Icon className="size-4" aria-hidden />
        Signé électroniquement par {proof.signer_name} le {formatDateTime(proof.signed_at)}
      </p>
      <p className={proof.integrity_ok ? "text-muted-foreground" : "font-medium text-destructive"}>
        {proof.integrity_ok ? "Intégrité vérifiée : le contenu correspond à celui signé." : "Attention : le contenu ne correspond plus à celui signé."}
      </p>
      <p className="text-muted-foreground break-all">
        Empreinte SHA-256 : <span className="font-mono">{proof.document_hash}</span> · IP {proof.signer_ip || "—"} · {proof.signer_user_agent || "—"}
      </p>
    </section>
  )
}
