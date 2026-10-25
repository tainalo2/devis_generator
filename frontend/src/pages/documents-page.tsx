import { useEffect, useState } from "react"
import { Link } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { Button } from "@/components/ui/button"
import { downloadBlob, downloadDocumentPDF, listDocuments } from "@/lib/api"
import { DOC_KINDS, formatEUR, type Document } from "@/lib/types"

function kindLabel(k: string) {
  return DOC_KINDS.find((x) => x.value === k)?.label ?? k
}

export function DocumentsPage() {
  const { user, loading } = useAuth()
  const [docs, setDocs] = useState<Document[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!user) return
    void listDocuments()
      .then(setDocs)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erreur"))
  }, [user])

  if (!loading && !user) {
    return (
      <p className="text-muted-foreground">
        <Link className="underline" to="/documents/nouveau">
          Créer un document en mode invité
        </Link>{" "}
        ou connectez-vous pour les sauvegarder.
      </p>
    )
  }

  return (
    <section className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-3xl font-semibold tracking-tight">Documents</h1>
        <Button render={<Link to="/documents/nouveau" />}>Nouveau document</Button>
      </div>
      {error && <p className="text-sm text-destructive">{error}</p>}
      <div className="overflow-x-auto rounded-md border border-border">
        <table className="w-full min-w-[40rem] text-left text-sm">
          <thead className="border-b border-border bg-muted/40">
            <tr>
              <th className="px-3 py-2 font-medium">Type</th>
              <th className="px-3 py-2 font-medium">N°</th>
              <th className="px-3 py-2 font-medium">Statut</th>
              <th className="px-3 py-2 font-medium">Total TTC</th>
              <th className="px-3 py-2 font-medium" />
            </tr>
          </thead>
          <tbody>
            {docs.map((d) => (
              <tr key={d.id} className="border-b border-border last:border-0">
                <td className="px-3 py-2">{kindLabel(d.kind)}</td>
                <td className="px-3 py-2">{d.number ?? "—"}</td>
                <td className="px-3 py-2">{d.status}</td>
                <td className="px-3 py-2">{formatEUR(d.total_cents)}</td>
                <td className="px-3 py-2">
                  <div className="flex flex-wrap gap-2">
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
            {docs.length === 0 && (
              <tr>
                <td colSpan={5} className="px-3 py-6 text-muted-foreground">
                  Aucun document.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  )
}
