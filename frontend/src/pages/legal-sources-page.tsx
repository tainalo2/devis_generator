import { useEffect, useState } from "react"

import { fetchLegalStatus, type LegalStatusResponse } from "@/lib/api"

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleString("fr-FR")
  } catch {
    return iso
  }
}

function statusLabel(status: string, fresh: boolean): string {
  if (status === "error") return "Erreur (repli sur dernière valeur)"
  if (status === "manual") return "Manuel (surveillance)"
  if (status === "stale" || !fresh) return "Ancien"
  if (status === "verified") return "Vérifié"
  return status
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
      <div className="space-y-2">
        <h1 className="text-3xl font-semibold tracking-tight">Sources légales</h1>
        <p className="text-muted-foreground">
          Valeurs utilisées pour les mentions, taux et contrôles. En cas d&apos;indisponibilité d&apos;une
          source, la dernière valeur vérifiée est conservée.
        </p>
      </div>

      {error && <p className="text-sm text-destructive">{error}</p>}
      {!data && !error && <p className="text-muted-foreground">Chargement…</p>}

      {data && (
        <>
          <div className="space-y-3">
            <h2 className="text-lg font-medium">Variables en cache</h2>
            <ul className="divide-y divide-border border border-border">
              {data.variables.length === 0 && (
                <li className="px-4 py-3 text-sm text-muted-foreground">Aucune variable pour l&apos;instant.</li>
              )}
              {data.variables.map((v) => (
                <li key={v.key} className="space-y-1 px-4 py-3 text-sm">
                  <div className="flex flex-wrap items-baseline justify-between gap-2">
                    <span className="font-mono text-xs sm:text-sm">{v.key}</span>
                    <span className="text-xs text-muted-foreground">{statusLabel(v.status, v.fresh)}</span>
                  </div>
                  <p className="text-muted-foreground">
                    Source {v.source} · {formatDate(v.fetched_at)}
                    {v.age_seconds >= 0 ? ` · âge ${Math.round(v.age_seconds / 3600)} h` : ""}
                  </p>
                  <pre className="overflow-x-auto whitespace-pre-wrap break-all rounded bg-muted/50 p-2 font-mono text-xs">
                    {JSON.stringify(v.value, null, 2)}
                  </pre>
                </li>
              ))}
            </ul>
          </div>

          <div className="space-y-3">
            <h2 className="text-lg font-medium">Sources branchées / prévues</h2>
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
