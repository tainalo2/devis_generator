import { useEffect, useState } from "react"
import { Link, Navigate, useNavigate, useParams } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { DocumentForm, type DocumentFormValue } from "@/components/document-form"
import { Button } from "@/components/ui/button"
import {
  downloadBlob,
  downloadDocumentPDF,
  getDocument,
  issueDocument,
  listClients,
  saveDocument,
} from "@/lib/api"
import { emptyLine, formatEUR, type Document } from "@/lib/types"

export function DocumentDetailPage() {
  const { id } = useParams()
  const { user, loading } = useAuth()
  const navigate = useNavigate()
  const [doc, setDoc] = useState<Document | null>(null)
  const [clients, setClients] = useState<{ id: string; name: string }[]>([])
  const [form, setForm] = useState<DocumentFormValue | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  useEffect(() => {
    if (!user || !id) return
    void getDocument(id)
      .then((d) => {
        setDoc(d)
        setForm({
          kind: d.kind,
          title: d.title,
          vat_regime: d.vat_regime,
          client_id: d.client_id ?? "",
          issue_date: d.issue_date ?? "",
          due_date: d.due_date ?? "",
          notes: d.notes,
          payment_terms: d.payment_terms,
          lines: d.lines?.length ? d.lines : [emptyLine()],
        })
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erreur"))
    void listClients().then((list) => setClients(list.map((c) => ({ id: c.id, name: c.name }))))
  }, [user, id])

  if (!loading && !user) return <Navigate to="/connexion" replace />
  if (!form || !doc) {
    return <p className="text-muted-foreground">{error ?? "Chargement…"}</p>
  }

  const readOnly = doc.status !== "draft"

  async function save() {
    if (!id || !form) return
    setPending(true)
    setError(null)
    try {
      const d = await saveDocument(
        {
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
        },
        id,
      )
      setDoc(d)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur")
    } finally {
      setPending(false)
    }
  }

  async function issue() {
    if (!id) return
    setPending(true)
    setError(null)
    try {
      const d = await issueDocument(id)
      setDoc(d)
      navigate(`/documents/${d.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erreur")
    } finally {
      setPending(false)
    }
  }

  return (
    <section className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">Document</h1>
          <p className="text-sm text-muted-foreground">
            Statut : {doc.status}
            {doc.number ? ` · N° ${doc.number}` : ""} · TTC {formatEUR(doc.total_cents)}
          </p>
        </div>
        <Button variant="ghost" render={<Link to="/documents" />}>
          Liste
        </Button>
      </div>

      <DocumentForm
        value={form}
        onChange={setForm}
        clients={clients}
        showClientSelect
        readOnly={readOnly}
      />

      {error && <p className="text-sm text-destructive">{error}</p>}

      <div className="flex flex-wrap gap-2">
        {!readOnly && (
          <>
            <Button type="button" disabled={pending} onClick={() => void save()}>
              Enregistrer
            </Button>
            <Button type="button" disabled={pending} onClick={() => void issue()}>
              Émettre (attribuer un numéro)
            </Button>
          </>
        )}
        <Button
          type="button"
          variant="outline"
          disabled={pending}
          onClick={() => {
            void downloadDocumentPDF(doc.id).then((b) =>
              downloadBlob(b, `${doc.number ?? doc.id}.pdf`),
            )
          }}
        >
          Télécharger PDF
        </Button>
      </div>
    </section>
  )
}
