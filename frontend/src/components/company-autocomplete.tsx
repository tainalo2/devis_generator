import { Search } from "lucide-react"
import { useEffect, useId, useState } from "react"

import { Input } from "@/components/ui/input"
import { searchCompanies, type CompanyHit } from "@/lib/api"

type Props = {
  label?: string
  placeholder?: string
  onPick: (hit: CompanyHit) => void
}

export function CompanyAutocomplete({
  label = "Rechercher une entreprise",
  placeholder = "Nom, SIREN ou SIRET",
  onPick,
}: Props) {
  const listId = useId()
  const [q, setQ] = useState("")
  const [hits, setHits] = useState<CompanyHit[]>([])
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const canSearch = q.trim().length >= 2
  const visibleHits = canSearch ? hits : []

  useEffect(() => {
    if (!canSearch) return
    const query = q.trim()
    const handle = window.setTimeout(() => {
      setPending(true)
      setError(null)
      void searchCompanies(query, 5)
        .then((list) => setHits(list))
        .catch((err: unknown) => {
          setHits([])
          setError(err instanceof Error ? err.message : "Recherche impossible")
        })
        .finally(() => setPending(false))
    }, 300)
    return () => window.clearTimeout(handle)
  }, [q, canSearch])

  return (
    <div className="flex flex-col gap-2">
      <label className="flex flex-col gap-1.5 text-sm font-medium">
        <span>{label}</span>
        <span className="relative">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
          <Input
            type="search"
            className="pl-9"
            placeholder={placeholder}
            value={q}
            onChange={(e) => setQ(e.target.value)}
            autoComplete="off"
            aria-autocomplete="list"
            aria-controls={listId}
          />
        </span>
      </label>
      {pending && <p className="text-xs text-muted-foreground">Recherche…</p>}
      {error && <p className="text-xs text-destructive">{error}</p>}
      {visibleHits.length > 0 && (
        <ul id={listId} className="flex max-h-56 flex-col overflow-auto rounded-lg border border-border bg-card p-1">
          {visibleHits.map((h) => (
            <li key={`${h.siren}-${h.siret}`}>
              <button
                type="button"
                className="flex w-full items-center justify-between gap-3 rounded-md px-3 py-2.5 text-left hover:bg-muted focus-visible:ring-0 focus-visible:ring-offset-0 focus-visible:[box-shadow:inset_0_0_0_2px_var(--ring)]"
                onClick={() => {
                  onPick(h)
                  setQ("")
                  setHits([])
                }}
              >
                <span className="flex min-w-0 flex-col gap-0.5">
                  <span className="truncate text-sm font-medium">{h.name}</span>
                  <span className="text-xs text-muted-foreground">
                    {[h.siren && `SIREN ${h.siren}`, h.city].filter(Boolean).join(" · ")}
                  </span>
                </span>
                <span className="text-xs font-medium whitespace-nowrap text-brand">Utiliser</span>
              </button>
            </li>
          ))}
        </ul>
      )}
      <p className="text-xs text-muted-foreground">Source : annuaire Recherche d’entreprises de l’État.</p>
    </div>
  )
}
