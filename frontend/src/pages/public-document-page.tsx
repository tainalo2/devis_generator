import { CheckCircle2, Download, PenLine } from "lucide-react"
import { useEffect, useState } from "react"
import { useParams } from "react-router-dom"

import { LogoMark } from "@/components/logo"
import { SignatureDialog } from "@/components/signature-dialog"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { fetchPublicDocumentPDF, fetchPublicSignatureState, PUBLIC_LINK_UNAVAILABLE, type PublicSignatureState } from "@/lib/api"
import { formatDateTime } from "@/lib/documents"

export function PublicDocumentPage() {
  const { token } = useParams()
  const [pdfURL, setPDFURL] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(token ? null : PUBLIC_LINK_UNAVAILABLE)
  const [signature, setSignature] = useState<PublicSignatureState | null>(null)
  const [signing, setSigning] = useState(false)

  useEffect(() => {
    if (!token) return
    let active = true
    let objectURL: string | null = null
    void fetchPublicDocumentPDF(token)
      .then((pdf) => {
        if (!active) return
        objectURL = URL.createObjectURL(pdf)
        setPDFURL(objectURL)
      })
      .catch((err: unknown) => {
        if (active) setError(err instanceof Error ? err.message : "Document indisponible.")
      })
    return () => {
      active = false
      if (objectURL) URL.revokeObjectURL(objectURL)
    }
  }, [token])

  useEffect(() => {
    if (!token) return
    let active = true
    // L'état de signature est secondaire : en cas d'échec le document reste consultable.
    void fetchPublicSignatureState(token)
      .then((state) => active && setSignature(state))
      .catch(() => undefined)
    return () => {
      active = false
    }
  }, [token])

  return (
    <main className="flex min-h-dvh flex-col bg-muted/50">
      <header className="flex min-h-16 items-center justify-between gap-4 border-b border-border bg-background px-4 md:px-6">
        <div className="flex items-center gap-2 font-heading font-semibold">
          <LogoMark />
          Devis Generator
        </div>
        <div className="flex items-center gap-2">
          {signature?.signable && (
            <Button onClick={() => setSigning(true)}>
              <PenLine />
              Signer ce devis
            </Button>
          )}
          {pdfURL && (
            <Button variant={signature?.signable ? "outline" : "default"} render={<a href={pdfURL} download="document.pdf" />}>
              <Download />
              Télécharger le PDF
            </Button>
          )}
        </div>
      </header>
      {signature?.signed && (
        <p role="status" className="flex items-center gap-2 border-b border-border bg-card px-4 py-2 text-sm md:px-6">
          <CheckCircle2 className="size-4 text-brand" aria-hidden />
          Devis signé par {signature.signer_name} le {formatDateTime(signature.signed_at)}.
        </p>
      )}
      {token && (
        <SignatureDialog
          token={token}
          open={signing}
          onOpenChange={setSigning}
          onSigned={(result) => {
            setSignature({ signed: true, signable: false, signed_at: result.signed_at, signer_name: result.signer_name })
            setSigning(false)
          }}
        />
      )}
      {error ? (
        <section className="m-auto max-w-lg px-6 py-16 text-center">
          <h1 className="font-heading text-2xl font-semibold">Document indisponible</h1>
          <p className="mt-3 text-sm text-muted-foreground">{error}</p>
        </section>
      ) : pdfURL ? (
        <object title="Document partagé" data={pdfURL} type="application/pdf" className="min-h-[calc(100dvh-4rem)] w-full">
          <p className="m-auto max-w-lg px-6 py-16 text-center text-sm">
            Votre navigateur ne peut pas afficher ce PDF. <a href={pdfURL}>Télécharger le document</a>.
          </p>
        </object>
      ) : (
        <div className="m-auto flex items-center gap-3 px-6 py-16 text-sm text-muted-foreground">
          <Spinner label="Chargement du document" />
          <span>Chargement du document…</span>
        </div>
      )}
    </main>
  )
}