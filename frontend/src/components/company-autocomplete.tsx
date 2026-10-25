import { useEffect, useId, useState } from "react"

import { searchCompanies, type CompanyHit } from "@/lib/api"

type Props = {
  label?: string
  placeholder?: string
  onPick: (hit: CompanyHit) => void
}

export function CompanyAutocomplete({
  label = "Rechercher une entreprise",
  placeholder = "Nom, SIREN ou SIRET…",
  onPick,
}: Props) {
  const listId = useId()
  const [q, setQ] = useState("")
  const [hits, setHits] = useState<CompanyHit[]>([])
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)

  useEffect(() => {
    if (q.trim().length < 2) {
      setHits([])
      return
    }
    const handle = window.setTimeout(() => {
      setPending(true)
      setError(null)
      void searchCompanies(q.trim())
        .then((list) => setHits(list))
        .catch((err: unknown) => {
          setHits([])
          setError(err instanceof Error ? err.message : "Recherche impossible")
        })
        .finally(() => setPending(false))
    }, 300)
    return () => window.clearTimeout(handle)
  }, [q])

  return (
    <div className="space-y-2">
      <label className="block space-y-1 text-sm">
        <span className="font-medium">{label}</span>
        <input
          className="flex h-9 w-full rounded-md border border-input bg-background px-3 text-sm"
          placeholder={placeholder}
          value={q}
          onChange={(e) => setQ(e.target.value)}
          autoComplete="off"
          aria-autocomplete="list"
          aria-controls={listId}
        />
      </label>
      {pending && <p className="text-xs text-muted-foreground">Recherche…</p>}
      {error && <p className="text-xs text-destructive">{error}</p>}
      {hits.length > 0 && (
        <ul
          id={listId}
          className="max-h-48 overflow-auto border border-border bg-background text-sm"
          role="listbox"
        >
          {hits.map((h) => (
            <li key={`${h.siren}-${h.siret}`}>
              <button
                type="button"
                className="flex w-full flex-col items-start gap-0.5 px-3 py-2 text-left hover:bg-accent"
                onClick={() => {
                  onPick(h)
                  setQ(h.name)
                  setHits([])
                }}
              >
                <span className="font-medium">{h.name}</span>
                <span className="text-xs text-muted-foreground">
                  {[h.siren && `SIREN ${h.siren}`, h.siret && `SIRET ${h.siret}`, h.city]
                    .filter(Boolean)
                    .join(" · ")}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
      <p className="text-xs text-muted-foreground">
        Source : API Recherche d&apos;entreprises (DINUM) — aucune clé requise.
      </p>
    </div>
  )
}
