import { useEffect, useMemo, useState } from "react"
import { Link, Navigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { EmptyState } from "@/components/empty-state"
import { Seo } from "@/components/seo"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { listDocuments } from "@/lib/api"
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

export function DashboardPage() {
  const { user, loading } = useAuth()
  const [docs, setDocs] = useState<Document[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!user) return
    void listDocuments()
      .then(setDocs)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erreur"))
  }, [user])

  const stats = useMemo(() => {
    const list = docs ?? []
    const drafts = list.filter((d) => d.status === "draft")
    const recent = [...list].slice(0, 8)
    const toCollect = list.filter(
      (d) =>
        (d.kind === "invoice" || d.kind === "deposit_invoice") &&
        (d.status === "issued" || d.status === "sent"),
    )
    const toCollectCents = toCollect.reduce((s, d) => s + d.total_cents, 0)
    return { drafts, recent, toCollect, toCollectCents }
  }, [docs])

  if (!loading && !user) return <Navigate to="/connexion" replace />

  return (
    <section className="space-y-8">
      <Seo title="Tableau de bord" path="/tableau-de-bord" noIndex />
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="font-heading text-3xl font-semibold tracking-tight">Tableau de bord</h1>
          <p className="text-muted-foreground">
            Bonjour{user?.display_name ? ` ${user.display_name}` : ""}. Vue d’ensemble de vos documents.
          </p>
        </div>
        <Button render={<Link to="/documents/nouveau" />}>Nouveau document</Button>
      </div>

      {error && <p className="text-sm text-destructive">{error}</p>}
      {docs === null && !error && <Spinner label="Chargement du tableau de bord" />}

      {docs && (
        <>
          <div className="grid gap-4 sm:grid-cols-3">
            <div className="rounded-xl border border-border bg-card/70 p-4">
              <p className="text-xs uppercase tracking-wide text-muted-foreground">Brouillons</p>
              <p className="mt-2 font-heading text-3xl font-semibold">{stats.drafts.length}</p>
            </div>
            <div className="rounded-xl border border-border bg-card/70 p-4">
              <p className="text-xs uppercase tracking-wide text-muted-foreground">À encaisser</p>
              <p className="mt-2 font-heading text-3xl font-semibold">{formatEUR(stats.toCollectCents)}</p>
              <p className="text-xs text-muted-foreground">{stats.toCollect.length} facture(s) émise(s)</p>
            </div>
            <div className="rounded-xl border border-border bg-card/70 p-4">
              <p className="text-xs uppercase tracking-wide text-muted-foreground">Documents</p>
              <p className="mt-2 font-heading text-3xl font-semibold">{docs.length}</p>
            </div>
          </div>

          <div className="space-y-3">
            <div className="flex items-center justify-between gap-2">
              <h2 className="font-heading text-xl font-semibold">Récents</h2>
              <Button variant="ghost" size="sm" render={<Link to="/documents" />}>
                Tout voir
              </Button>
            </div>
            {stats.recent.length === 0 ? (
              <EmptyState
                title="Aucun document pour l’instant"
                description="Créez un devis ou une facture — vous pourrez l’émettre et télécharger le PDF."
                action={<Button render={<Link to="/documents/nouveau" />}>Créer un document</Button>}
              />
            ) : (
              <ul className="divide-y divide-border rounded-xl border border-border bg-card/60">
                {stats.recent.map((d) => (
                  <li key={d.id} className="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
                    <div className="min-w-0 space-y-1">
                      <Link className="font-medium hover:underline" to={`/documents/${d.id}`}>
                        {d.title || kindLabel(d.kind)}
                      </Link>
                      <p className="text-xs text-muted-foreground">
                        {kindLabel(d.kind)} · {d.number ?? "sans numéro"} · {formatEUR(d.total_cents)}
                      </p>
                    </div>
                    {statusBadge(d.status)}
                  </li>
                ))}
              </ul>
            )}
          </div>

          {stats.drafts.length > 0 && (
            <div className="space-y-3">
              <h2 className="font-heading text-xl font-semibold">Brouillons</h2>
              <ul className="space-y-2">
                {stats.drafts.slice(0, 5).map((d) => (
                  <li key={d.id}>
                    <Link className="text-sm text-brand underline-offset-4 hover:underline" to={`/documents/${d.id}`}>
                      {d.title || kindLabel(d.kind)} — reprendre
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </>
      )}
    </section>
  )
}
