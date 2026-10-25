import { useEffect, useState } from "react"
import { Link } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { EmptyState } from "@/components/empty-state"
import { Seo } from "@/components/seo"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { downloadBlob, downloadDocumentPDF, listDocuments } from "@/lib/api"
import { DOC_KINDS, formatEUR, type Document } from "@/lib/types"

function kindLabel(k: string) {
  return DOC_KINDS.find((x) => x.value === k)?.label ?? k
}

function statusBadge(status: string) {
  if (status === "draft") return <Badge variant="warning">Brouillon</Badge>
  if (status === "issued" || status === "sent") return <Badge variant="success">Émis</Badge>
  if (status === "paid") return <Badge variant="success">Payé</Badge>
  return <Badge variant="muted">{status}</Badge>
}

export function DocumentsPage() {
  const { user, loading } = useAuth()
  const [docs, setDocs] = useState<Document[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!user) return
    void listDocuments()
      .then(setDocs)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erreur"))
  }, [user])

  if (!loading && !user) {
    return (
      <EmptyState
        title="Connexion requise"
        description="Créez un document en mode invité, ou connectez-vous pour sauvegarder l’historique."
        action={
          <div className="flex flex-wrap gap-2">
            <Button render={<Link to="/documents/nouveau" />}>Mode invité</Button>
            <Button variant="outline" render={<Link to="/connexion" />}>
              Connexion
            </Button>
          </div>
        }
      />
    )
  }

  return (
    <section className="space-y-6">
      <Seo title="Documents" path="/documents" noIndex />
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="font-heading text-3xl font-semibold tracking-tight">Documents</h1>
        <Button render={<Link to="/documents/nouveau" />}>Nouveau document</Button>
      </div>
      {error && <p className="text-sm text-destructive">{error}</p>}
      {docs === null && !error && <Spinner />}
      {docs && docs.length === 0 && (
        <EmptyState
          title="Aucun document"
          description="Vos devis et factures apparaîtront ici."
          action={<Button render={<Link to="/documents/nouveau" />}>Créer un document</Button>}
        />
      )}
      {docs && docs.length > 0 && (
        <div className="overflow-x-auto rounded-xl border border-border bg-card/60">
          <table className="w-full min-w-[40rem] text-left text-sm">
            <thead className="border-b border-border bg-muted/40">
              <tr>
                <th className="px-3 py-2 font-medium">Type</th>
                <th className="px-3 py-2 font-medium">N°</th>
                <th className="px-3 py-2 font-medium">Statut</th>
                <th className="px-3 py-2 font-medium">Total TTC</th>
                <th className="px-3 py-2 font-medium">
                  <span className="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {docs.map((d) => (
                <tr key={d.id} className="border-b border-border last:border-0">
                  <td className="px-3 py-2">{kindLabel(d.kind)}</td>
                  <td className="px-3 py-2">{d.number ?? "—"}</td>
                  <td className="px-3 py-2">{statusBadge(d.status)}</td>
                  <td className="px-3 py-2">{formatEUR(d.total_cents)}</td>
                  <td className="px-3 py-2">
                    <div className="flex flex-wrap gap-3">
                      <Link className="underline underline-offset-4" to={`/documents/${d.id}`}>
                        Ouvrir
                      </Link>
                      <button
                        type="button"
                        className="underline underline-offset-4"
                        onClick={() => {
                          void downloadDocumentPDF(d.id).then((b) =>
                            downloadBlob(b, `${d.number ?? d.id}.pdf`),
                          )
                        }}
                      >
                        PDF
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}
