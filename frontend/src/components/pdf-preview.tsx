import { useEffect, useState } from "react"

import { Spinner } from "@/components/ui/spinner"
import { guestPDF } from "@/lib/api"
import { emptyAddress, type DocLine } from "@/lib/types"

export type PreviewInput = {
  kind: string
  title: string
  vat_regime: string
  issue_date: string
  due_date: string
  notes: string
  payment_terms: string
  lines: DocLine[]
  issuerName: string
  issuerSiret: string
  clientName: string
}

export function PDFPreview({ input }: { input: PreviewInput }) {
  const [url, setUrl] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const signature = JSON.stringify(input)

  useEffect(() => {
    let cancelled = false
    let objectUrl: string | null = null
    const parsed = JSON.parse(signature) as PreviewInput
    const handle = window.setTimeout(() => {
      setPending(true)
      setError(null)
      void guestPDF({
        kind: parsed.kind,
        title: parsed.title,
        vat_regime: parsed.vat_regime,
        issue_date: parsed.issue_date,
        due_date: parsed.due_date,
        notes: parsed.notes,
        payment_terms: parsed.payment_terms,
        currency: "EUR",
        number: "",
        issuer: {
          legal_name: parsed.issuerName || "Émetteur",
          trade_name: "",
          email: "",
          phone: "",
          siret: parsed.issuerSiret,
          siren: "",
          vat_number: "",
          ape_code: "",
          rcs: "",
          address: emptyAddress(),
          default_vat_regime: parsed.vat_regime,
          payment_terms: parsed.payment_terms,
        },
        client: {
          name: parsed.clientName || "Client",
          email: "",
          address: emptyAddress(),
          siren: "",
          siret: "",
          vat_number: "",
          notes: "",
        },
        lines: parsed.lines.map(({ id: _id, ...rest }) => rest),
      })
        .then((blob) => {
          if (cancelled) return
          objectUrl = URL.createObjectURL(blob)
          setUrl((prev) => {
            if (prev) URL.revokeObjectURL(prev)
            return objectUrl
          })
        })
        .catch((err: unknown) => {
          if (!cancelled) {
            setUrl((prev) => {
              if (prev) URL.revokeObjectURL(prev)
              return null
            })
            setError(err instanceof Error ? err.message : "Aperçu impossible")
          }
        })
        .finally(() => {
          if (!cancelled) setPending(false)
        })
    }, 450)
    return () => {
      cancelled = true
      window.clearTimeout(handle)
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [signature])

  return (
    <div className="flex h-full min-h-[28rem] flex-col overflow-hidden rounded-xl border border-border bg-card/70">
      <div className="flex items-center justify-between border-b border-border px-3 py-2 text-xs text-muted-foreground">
        <span>Aperçu PDF (direct)</span>
        {pending && <Spinner label="Mise à jour de l’aperçu" />}
      </div>
      <div className="relative flex-1 bg-muted/30">
        {error && <p className="p-4 text-sm text-destructive">{error}</p>}
        {url && !error && (
          <iframe title="Aperçu PDF" src={url} className="absolute inset-0 h-full w-full border-0" />
        )}
        {!url && !error && !pending && (
          <p className="p-4 text-sm text-muted-foreground">L’aperçu apparaîtra après saisie.</p>
        )}
      </div>
    </div>
  )
}
