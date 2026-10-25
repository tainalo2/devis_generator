import { useEffect, useMemo, useState } from "react"
import { Link, useNavigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { DocumentForm, type DocumentFormValue } from "@/components/document-form"
import { PDFPreview } from "@/components/pdf-preview"
import { Seo } from "@/components/seo"
import { useToast } from "@/components/toast-provider"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  downloadBlob,
  getIssuer,
  guestPDF,
  listClients,
  saveDocument,
} from "@/lib/api"
import { emptyAddress, emptyLine } from "@/lib/types"
import { cn } from "@/lib/utils"

const today = () => new Date().toISOString().slice(0, 10)

const steps = [
  { id: 1, label: "Parties" },
  { id: 2, label: "Lignes" },
  { id: 3, label: "Aperçu" },
] as const

export function NewDocumentPage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const { push } = useToast()
  const [step, setStep] = useState(1)
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
    void listClients().then((list) => {
      setClients(list.map((c) => ({ id: c.id, name: c.name })))
    })
    void getIssuer().then((p) => {
      setIssuerName(p.legal_name)
      setIssuerSiret(p.siret)
      if (p.default_vat_regime) setForm((f) => ({ ...f, vat_regime: p.default_vat_regime }))
    })
  }, [user])

  const selectedClientName = clients.find((x) => x.id === form.client_id)?.name
  const effectiveClientName = user ? selectedClientName || clientName || "Client" : clientName || "Client"

  const previewInput = useMemo(
    () => ({
      kind: form.kind,
      title: form.title,
      vat_regime: form.vat_regime,
      issue_date: form.issue_date,
      due_date: form.due_date,
      notes: form.notes,
      payment_terms: form.payment_terms,
      lines: form.lines,
      issuerName: issuerName || (user ? "Émetteur" : "Émetteur (invité)"),
      issuerSiret,
      clientName: effectiveClientName,
    }),
    [form, issuerName, issuerSiret, effectiveClientName, user],
  )

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
      push("Brouillon enregistré.", "success")
      navigate(`/documents/${doc.id}`)
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Erreur"
      setError(msg)
      push(msg, "error")
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
      push("PDF téléchargé.", "success")
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Erreur"
      setError(msg)
      push(msg, "error")
    } finally {
      setPending(false)
    }
  }

  return (
    <section className="space-y-6">
      <Seo title="Nouveau document" path="/documents/nouveau" />
      <div className="space-y-2">
        <h1 className="font-heading text-3xl font-semibold tracking-tight">Nouveau document</h1>
        <p className="text-muted-foreground">
          {user
            ? "Enregistrement sur votre compte (brouillon), avec aperçu PDF en direct."
            : "Mode invité : rien n’est conservé côté serveur — aperçu et téléchargement PDF."}
        </p>
      </div>

      <ol className="flex flex-wrap gap-2" aria-label="Étapes">
        {steps.map((s) => (
          <li key={s.id}>
            <button
              type="button"
              className={cn(
                "rounded-md border px-3 py-1.5 text-sm transition-colors",
                step === s.id
                  ? "border-brand bg-brand/10 text-foreground"
                  : "border-border text-muted-foreground hover:bg-muted/60",
              )}
              aria-current={step === s.id ? "step" : undefined}
              onClick={() => setStep(s.id)}
            >
              {s.id}. {s.label}
            </button>
          </li>
        ))}
      </ol>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(18rem,24rem)]">
        <div className="space-y-6">
          {step === 1 && (
            <div className="space-y-4 rounded-xl border border-border bg-card/60 p-4">
              {!user && (
                <div className="grid gap-3 md:grid-cols-2">
                  <label className="space-y-1 text-sm">
                    <span className="font-medium">Votre nom / raison sociale</span>
                    <Input value={issuerName} onChange={(e) => setIssuerName(e.target.value)} />
                  </label>
                  <label className="space-y-1 text-sm">
                    <span className="font-medium">Votre SIRET</span>
                    <Input value={issuerSiret} onChange={(e) => setIssuerSiret(e.target.value)} />
                  </label>
                  <label className="space-y-1 text-sm md:col-span-2">
                    <span className="font-medium">Nom du client</span>
                    <Input value={clientName} onChange={(e) => setClientName(e.target.value)} />
                  </label>
                </div>
              )}
              <DocumentForm
                value={form}
                onChange={setForm}
                clients={clients}
                showClientSelect={!!user}
              />
            </div>
          )}

          {step === 2 && (
            <div className="rounded-xl border border-border bg-card/60 p-4">
              <DocumentForm
                value={form}
                onChange={setForm}
                clients={clients}
                showClientSelect={!!user}
              />
            </div>
          )}

          {step === 3 && (
            <div className="space-y-4 xl:hidden">
              <PDFPreview input={previewInput} />
            </div>
          )}

          {error && <p className="text-sm text-destructive">{error}</p>}

          <div className="flex flex-wrap gap-2">
            {step > 1 && (
              <Button type="button" variant="outline" onClick={() => setStep((s) => s - 1)}>
                Précédent
              </Button>
            )}
            {step < 3 && (
              <Button type="button" onClick={() => setStep((s) => s + 1)}>
                Continuer
              </Button>
            )}
            {user ? (
              <Button type="button" disabled={pending} onClick={() => void saveAccount()}>
                Enregistrer le brouillon
              </Button>
            ) : (
              <Button type="button" disabled={pending} onClick={() => void pdfGuest()}>
                Télécharger le PDF
              </Button>
            )}
            {!user && (
              <Button variant="ghost" render={<Link to="/inscription" />}>
                Créer un compte
              </Button>
            )}
          </div>
        </div>

        <aside className="hidden xl:block">
          <div className="sticky top-20">
            <PDFPreview input={previewInput} />
          </div>
        </aside>
      </div>
    </section>
  )
}
