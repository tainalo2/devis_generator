import { useEffect, useState } from "react"
import { Link } from "react-router-dom"

import { Button } from "@/components/ui/button"
import { fetchHealthz, type HealthzResponse } from "@/lib/api"

type HealthState =
  | { kind: "loading" }
  | { kind: "ok"; data: HealthzResponse }
  | { kind: "error"; message: string }

export function HomePage() {
  const [health, setHealth] = useState<HealthState>({ kind: "loading" })

  useEffect(() => {
    let cancelled = false
    fetchHealthz()
      .then((data) => {
        if (!cancelled) setHealth({ kind: "ok", data })
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setHealth({
            kind: "error",
            message: err instanceof Error ? err.message : "erreur inconnue",
          })
        }
      })
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <section className="space-y-6">
      <div className="space-y-2">
        <h1 className="text-3xl font-semibold tracking-tight text-foreground md:text-4xl">
          Devis Generator : VIBED
        </h1>
        <p className="max-w-2xl text-muted-foreground">
          Devis et factures pour micro-entrepreneurs, indépendants et petites sociétés. Utilisable en
          mode invité ; un compte optionnel permet de sauvegarder vos données.
        </p>
      </div>

      <div className="flex flex-wrap gap-3">
        <Button render={<Link to="/documents/nouveau" />}>Nouveau document</Button>
        <Button variant="outline" render={<Link to="/documents" />}>
          Mes documents
        </Button>
        <Button variant="outline" render={<Link to="/sources-legales" />}>
          Sources légales
        </Button>
        <Button variant="ghost" render={<Link to="/inscription" />}>
          Créer un compte
        </Button>
      </div>



      <div className="rounded-lg border border-border bg-muted/40 px-4 py-3 text-sm">
        <p className="font-medium text-foreground">Liaison API <code className="font-mono">/api/v1/healthz</code></p>
        {health.kind === "loading" && (
          <p className="mt-1 text-muted-foreground">Vérification en cours…</p>
        )}
        {health.kind === "ok" && (
          <p className="mt-1 text-muted-foreground">
            Statut <span className="font-medium text-foreground">{health.data.status}</span>
            {" · "}base <span className="font-medium text-foreground">{health.data.database}</span>
            {" · "}version <span className="font-medium text-foreground">{health.data.version}</span>
          </p>
        )}
        {health.kind === "error" && (
          <p className="mt-1 text-destructive">
            Impossible de joindre l&apos;API ({health.message}). Lancez le backend ou{" "}
            <code className="font-mono">docker compose up</code>.
          </p>
        )}
      </div>
    </section>
  )
}
