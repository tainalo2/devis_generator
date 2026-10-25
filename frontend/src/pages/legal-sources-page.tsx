import { useEffect, useState } from "react"

import { EmptyState } from "@/components/empty-state"
import { Seo } from "@/components/seo"
import { Badge } from "@/components/ui/badge"
import { Spinner } from "@/components/ui/spinner"
import { fetchLegalStatus, type LegalStatusResponse } from "@/lib/api"

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleString("fr-FR")
  } catch {
    return iso
  }
}

function statusBadge(status: string, fresh: boolean) {
  if (status === "error") return <Badge variant="warning">Erreur (repli)</Badge>
  if (status === "manual") return <Badge variant="secondary">Manuel</Badge>
  if (status === "stale" || !fresh) return <Badge variant="muted">Ancien</Badge>
  if (status === "verified") return <Badge variant="success">Vérifié</Badge>
  return <Badge variant="outline">{status}</Badge>
}

export function LegalSourcesPage() {
  const [data, setData] = useState<LegalStatusResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    void fetchLegalStatus()
      .then(setData)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erreur"))
  }, [])

  return (
    <section className="mx-auto max-w-3xl space-y-8">
      <Seo
        title="Sources légales"
        path="/sources-legales"
        description="Valeurs légales utilisées par Devis Generator : VIBED — sources, dates et fraîcheur."
      />
      <div className="space-y-2">
        <h1 className="font-heading text-3xl font-semibold tracking-tight">Sources légales</h1>
        <p className="text-muted-foreground">
          Valeurs utilisées pour les mentions, taux et contrôles. En cas d&apos;indisponibilité, la
          dernière valeur vérifiée est conservée.
        </p>
      </div>

      {error && <p className="text-sm text-destructive">{error}</p>}
      {!data && !error && <Spinner />}

      {data && (
        <>
          <div className="space-y-3">
            <h2 className="font-heading text-xl font-semibold">Variables en cache</h2>
            {data.variables.length === 0 ? (
              <EmptyState
                title="Aucune variable"
                description="Le job de rafraîchissement peuplera ces valeurs au démarrage."
              />
            ) : (
              <ul className="divide-y divide-border overflow-hidden rounded-xl border border-border bg-card/60">
                {data.variables.map((v) => (
                  <li key={v.key} className="space-y-2 px-4 py-3 text-sm">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <span className="font-mono text-xs sm:text-sm">{v.key}</span>
                      {statusBadge(v.status, v.fresh)}
                    </div>
                    <p className="text-muted-foreground">
                      Source {v.source} · {formatDate(v.fetched_at)}
                      {v.age_seconds >= 0 ? ` · âge ${Math.round(v.age_seconds / 3600)} h` : ""}
                    </p>
                    <pre className="overflow-x-auto whitespace-pre-wrap break-all rounded-md bg-muted/50 p-2 font-mono text-xs">
                      {JSON.stringify(v.value, null, 2)}
                    </pre>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className="space-y-3">
            <h2 className="font-heading text-xl font-semibold">Sources branchées / prévues</h2>
            <ul className="space-y-2 text-sm">
              {data.sources.map((s) => (
                <li key={s.id} className="border-b border-border pb-2">
                  <span className="font-medium">{s.label}</span>
                  <span className="text-muted-foreground"> — {s.auth}</span>
                </li>
              ))}
            </ul>
            {data.excluded.length > 0 && (
              <p className="text-sm text-muted-foreground">Exclues : {data.excluded.join(", ")}.</p>
            )}
          </div>
        </>
      )}
    </section>
  )
}
