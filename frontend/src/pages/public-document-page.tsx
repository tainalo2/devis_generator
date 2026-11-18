import { Download, Loader2, CheckCircle2 } from "lucide-react"
import { useEffect, useState } from "react"
import { useParams } from "react-router-dom"

import { LogoMark } from "@/components/logo"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { Input } from "@/components/ui/input"
import { Dialog } from "@/components/ui/dialog"
import { fetchPublicDocumentPDF, PUBLIC_LINK_UNAVAILABLE } from "@/lib/api"
import {
  fetchDocumentSignatureState,
  signDocument,
  type DocumentSignatureState,
} from "@/lib/signature-api"

export function PublicDocumentPage() {
  const { token } = useParams()
  const [pdfURL, setPDFURL] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(token ? null : PUBLIC_LINK_UNAVAILABLE)
  const [signatureState, setSignatureState] = useState<DocumentSignatureState | null>(null)
  const [showSignDialog, setShowSignDialog] = useState(false)
  const [signerEmail, setSignerEmail] = useState("")
  const [signerName, setSignerName] = useState("")
  const [isSigning, setIsSigning] = useState(false)
  const [signSuccess, setSignSuccess] = useState(false)
  const [signError, setSignError] = useState<string | null>(null)

  // Load PDF and signature state
  useEffect(() => {
    if (!token) return
    let active = true
    let objectURL: string | null = null

    Promise.all([
      fetchPublicDocumentPDF(token)
        .then((pdf) => {
          if (!active) return
          objectURL = URL.createObjectURL(pdf)
          setPDFURL(objectURL)
        })
        .catch((err: unknown) => {
          if (active) setError(err instanceof Error ? err.message : "Document indisponible.")
        }),
      fetchDocumentSignatureState(token)
        .then((state) => {
          if (active) setSignatureState(state)
        })
        .catch(() => {
          // Ignore errors loading signature state; it's optional
        }),
    ]).finally(() => {
      // No action needed after both promises settle
    })

    return () => {
      active = false
      if (objectURL) URL.revokeObjectURL(objectURL)
    }
  }, [token])

  const handleSign = async () => {
    if (!token || !signerEmail) return

    setIsSigning(true)
    setSignError(null)

    try {
      const result = await signDocument(token, signerEmail, signerName)
      setSignSuccess(true)
      setSignerEmail("")
      setSignerName("")

      // Update signature state
      setSignatureState({
        document_id: result.document_id,
        status: "signed",
        signed_at: result.signed_at,
        signed_by_email: result.signer_email,
        is_locked: true,
        proof_token: result.proof_token,
      })

      // Close dialog after 2 seconds
      setTimeout(() => {
        setShowSignDialog(false)
        setSignSuccess(false)
      }, 2000)
    } catch (err: unknown) {
      setSignError(err instanceof Error ? err.message : "Signature échouée.")
    } finally {
      setIsSigning(false)
    }
  }

  const isSigned = signatureState?.is_locked === true

  return (
    <main className="flex min-h-dvh flex-col bg-muted/50">
      <header className="flex min-h-16 items-center justify-between gap-4 border-b border-border bg-background px-4 md:px-6">
        <div className="flex items-center gap-2 font-heading font-semibold">
          <LogoMark />
          Devis Generator
        </div>
        <div className="flex items-center gap-2">
          {isSigned && (
            <div className="flex items-center gap-2 text-sm text-green-600">
              <CheckCircle2 className="h-4 w-4" />
              <span>Signé</span>
            </div>
          )}
          {pdfURL && (
            <Button render={<a href={pdfURL} download="document.pdf" />}>
              <Download />
              Télécharger le PDF
            </Button>
          )}
          {!isSigned && (
            <Button
              variant="default"
              onClick={() => setShowSignDialog(true)}
              disabled={error !== null}
            >
              Signer le document
            </Button>
          )}
        </div>
      </header>

      {error ? (
        <section className="m-auto max-w-lg px-6 py-16 text-center">
          <h1 className="font-heading text-2xl font-semibold">Document indisponible</h1>
          <p className="mt-3 text-sm text-muted-foreground">{error}</p>
        </section>
      ) : pdfURL ? (
        <object
          title="Document partagé"
          data={pdfURL}
          type="application/pdf"
          className="min-h-[calc(100dvh-4rem)] w-full"
        >
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

      {/* Signature Dialog */}
      <Dialog
        open={showSignDialog}
        onOpenChange={setShowSignDialog}
        title={signSuccess ? "Document signé" : "Signer le document"}
        icon={signSuccess ? <CheckCircle2 className="h-5 w-5 text-green-600" /> : undefined}
      >
        {signSuccess ? (
          <div className="flex flex-col items-center gap-4 py-8 text-center">
            <p className="text-sm text-muted-foreground">
              Le document a été signé et verrouillé.
            </p>
          </div>
        ) : (
          <>
            <p className="text-sm text-muted-foreground">
              Entrez vos coordonnées pour signer ce document électroniquement.
            </p>

            <div className="space-y-4 pt-2">
              <div>
                <label htmlFor="signer-email" className="block text-sm font-medium">
                  Adresse e-mail *
                </label>
                <Input
                  id="signer-email"
                  type="email"
                  placeholder="votre@email.com"
                  value={signerEmail}
                  onChange={(e) => setSignerEmail(e.target.value)}
                  disabled={isSigning}
                  className="mt-1"
                />
              </div>

              <div>
                <label htmlFor="signer-name" className="block text-sm font-medium">
                  Nom (optionnel)
                </label>
                <Input
                  id="signer-name"
                  type="text"
                  placeholder="Votre nom complet"
                  value={signerName}
                  onChange={(e) => setSignerName(e.target.value)}
                  disabled={isSigning}
                  className="mt-1"
                />
              </div>

              {signError && (
                <div className="rounded-md bg-red-50 p-3 text-sm text-red-800">
                  {signError}
                </div>
              )}
            </div>
          </>
        )}
      </Dialog>
    </main>
  )
}
