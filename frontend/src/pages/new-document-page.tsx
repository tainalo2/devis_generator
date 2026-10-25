import { useEffect, useState } from "react"
import { Link, useNavigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { DocumentForm, type DocumentFormValue } from "@/components/document-form"
import { Button } from "@/components/ui/button"
import {
  downloadBlob,
  getIssuer,
  guestPDF,
  listClients,
  saveDocument,
} from "@/lib/api"
import { emptyAddress, emptyLine } from "@/lib/types"

const today = () => new Date().toISOString().slice(0, 10)

export function NewDocumentPage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [clients, setClients] = useState<{ id: string; name: string }[]>([])
  const [issuerName, setIssuerName] = useState("")
  const [issuerSiret, setIssuerSiret] = useState("")
  const [clientName, setClientName] = useState("Client")
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const [form, setForm] = useState<DocumentFormValue>({
    kind: "quote",
    title: "",
    vat_regime: "franchise_293b",
    client_id: "",
    issue_date: today(),
    due_date: "",
    notes: "",
    payment_terms: "Paiement à réception",
    lines: [emptyLine()],
  })

  useEffect(() => {
    if (!user) return
    void listClients().then((list) => setClients(list.map((c) => ({ id: c.id, name: c.name }))))
    void getIssuer().then((p) => {
      setIssuerName(p.legal_name)
      setIssuerSiret(p.siret)
      if (p.default_vat_regime) setForm((f) => ({ ...f, vat_regime: p.default_vat_regime }))
    })
  }, [user])

  async function saveAccount() {
    setPending(true)
    setError(null)
    try {
      const doc = await saveDocument({
        kind: form.kind,
        title: form.title,
        vat_regime: form.vat_regime,
        client_id: form.client_id || null,
        issue_date: form.issue_date || null,
        due_date: form.due_date || null,
        notes: form.notes,
        payment_terms: form.payment_terms,
        currency: "EUR",
        lines: form.lines,
      })
      navigate(`/documents/${doc.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur")
    } finally {
      setPending(false)
    }
  }

  async function pdfGuest() {
    setPending(true)
    setError(null)
    try {
      const blob = await guestPDF({
        kind: form.kind,
        title: form.title,
        vat_regime: form.vat_regime,
        issue_date: form.issue_date,
        due_date: form.due_date,
        notes: form.notes,
        payment_terms: form.payment_terms,
        currency: "EUR",
        number: "",
        issuer: {
          legal_name: issuerName || "Émetteur (invité)",
          trade_name: "",
          email: "",
          phone: "",
          siret: issuerSiret,
          siren: "",
          vat_number: "",
          ape_code: "",
          rcs: "",
          address: emptyAddress(),
          default_vat_regime: form.vat_regime,
          payment_terms: form.payment_terms,
        },
        client: {
          id: "",
          name: clientName || "Client",
          email: "",
          address: emptyAddress(),
          siren: "",
          siret: "",
          vat_number: "",
          notes: "",
        },
        lines: form.lines,
      })
      downloadBlob(blob, "document-invite.pdf")
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur")
    } finally {
      setPending(false)
    }
  }

  return (
    <section className="space-y-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-semibold tracking-tight">Nouveau document</h1>
        <p className="text-muted-foreground">
          {user
            ? "Le document sera enregistré sur votre compte (brouillon)."
            : "Mode invité : rien n’est conservé côté serveur — téléchargez le PDF directement."}
        </p>
      </div>

      {!user && (
        <div className="grid gap-3 rounded-md border border-border p-4 md:grid-cols-2">
          <label className="space-y-1 text-sm">
            <span className="font-medium">Votre nom / raison sociale</span>
            <input
              className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={issuerName}
              onChange={(e) => setIssuerName(e.target.value)}
            />
          </label>
          <label className="space-y-1 text-sm">
            <span className="font-medium">Votre SIRET</span>
            <input
              className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={issuerSiret}
              onChange={(e) => setIssuerSiret(e.target.value)}
            />
          </label>
          <label className="space-y-1 text-sm md:col-span-2">
            <span className="font-medium">Nom du client</span>
            <input
              className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
              value={clientName}
              onChange={(e) => setClientName(e.target.value)}
            />
          </label>
        </div>
      )}

      <DocumentForm
        value={form}
        onChange={setForm}
        clients={clients}
        showClientSelect={!!user}
      />

      {error && <p className="text-sm text-destructive">{error}</p>}

      <div className="flex flex-wrap gap-2">
        {user ? (
          <Button type="button" disabled={pending} onClick={() => void saveAccount()}>
            Enregistrer le brouillon
          </Button>
        ) : (
          <Button type="button" disabled={pending} onClick={() => void pdfGuest()}>
            Télécharger le PDF (invité)
          </Button>
        )}
        {!user && (
          <Button variant="outline" render={<Link to="/inscription" />}>
            Créer un compte pour sauvegarder
          </Button>
        )}
      </div>
    </section>
  )
}
