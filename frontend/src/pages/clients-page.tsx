import { Plus, Search } from "lucide-react"
import { useEffect, useMemo, useState } from "react"
import { Link, useNavigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { ClientDialog } from "@/components/editor/dialogs"
import { emptyClient } from "@/components/editor/model"
import { EmptyState } from "@/components/empty-state"
import { Avatar, Card, DashedEmpty, PageHeader } from "@/components/page"
import { Seo } from "@/components/seo"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Select } from "@/components/ui/select"
import { Spinner } from "@/components/ui/spinner"
import { listClients, listDocuments, saveClient } from "@/lib/api"
import { clientStats, formatDate, isIndividual } from "@/lib/documents"
import { table } from "@/lib/styles"
import { formatEUR, type Client, type Document } from "@/lib/types"
import { cn } from "@/lib/utils"

type TypeFilter = "" | "pro" | "part"
type Sort = "name" | "billed" | "last"

export function ClientsPage() {
  const { user, loading } = useAuth()
  const navigate = useNavigate()
  const [clients, setClients] = useState<Client[] | null>(null)
  const [docs, setDocs] = useState<Document[]>([])
  const [error, setError] = useState<string | null>(null)
  const [query, setQuery] = useState("")
  const [type, setType] = useState<TypeFilter>("")
  const [sort, setSort] = useState<Sort>("name")
  const [creating, setCreating] = useState(false)

  useEffect(() => {
    if (!user) return
    void listClients()
      .then(setClients)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erreur"))
    void listDocuments().then(setDocs).catch(() => undefined)
  }, [user])

  const rows = useMemo(() => {
    const q = query.trim().toLowerCase()
    const list = (clients ?? [])
      .filter((c) => (type === "pro" ? !isIndividual(c) : type === "part" ? isIndividual(c) : true))
      .filter((c) => !q || [c.name, c.siren, c.address?.city ?? ""].some((s) => s.toLowerCase().includes(q)))
      .map((c) => ({ client: c, stats: clientStats(docs, c.id) }))
    if (sort === "billed") list.sort((a, b) => b.stats.billed - a.stats.billed)
    else if (sort === "last") list.sort((a, b) => (b.stats.lastInvoice ?? "").localeCompare(a.stats.lastInvoice ?? ""))
    else list.sort((a, b) => a.client.name.localeCompare(b.client.name, "fr"))
    return list
  }, [clients, docs, query, type, sort])

  if (!loading && !user) {
    return (
      <EmptyState
        title="Clients réservés au compte"
        description="Connectez-vous pour enregistrer vos clients, ou créez un document en mode invité."
        action={
          <div className="flex flex-wrap gap-2">
            <Button render={<Link to="/connexion" />}>Connexion</Button>
            <Button variant="outline" render={<Link to="/documents/nouveau" />}>
              Créer un devis sans compte
            </Button>
          </div>
        }
      />
    )
  }

  const total = clients?.length ?? 0
  const pros = (clients ?? []).filter((c) => !isIndividual(c)).length
  const parts = total - pros

  return (
    <section className="flex flex-col gap-5">
      <Seo title="Clients" path="/clients" noIndex />
      <PageHeader
        title="Clients"
        description={`${total} client${total > 1 ? "s" : ""} · ${pros} professionnel${pros > 1 ? "s" : ""}, ${parts} particulier${parts > 1 ? "s" : ""}`}
        actions={
          <Button aria-haspopup="dialog" onClick={() => setCreating(true)}>
            <Plus />
            Nouveau client
          </Button>
        }
      />

      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-0 flex-[1_1_260px]">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
          <Input
            type="search"
            className="pl-9"
            placeholder="Nom, SIREN ou ville"
            aria-label="Rechercher un client"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
        <Select aria-label="Type de client" className="w-32.5 md:w-40" value={type} onChange={(e) => setType(e.target.value as TypeFilter)}>
          <option value="">Tous</option>
          <option value="pro">Professionnels</option>
          <option value="part">Particuliers</option>
        </Select>
        <Select aria-label="Trier par" className="hidden w-50 md:flex" value={sort} onChange={(e) => setSort(e.target.value as Sort)}>
          <option value="name">Trier par nom</option>
          <option value="billed">Trier par CA facturé</option>
          <option value="last">Trier par dernière facture</option>
        </Select>
      </div>

      {error && <p className="text-sm text-destructive">{error}</p>}
      {clients === null && !error && <Spinner label="Chargement des clients" />}

      {clients && rows.length === 0 && (
        <DashedEmpty
          title={total === 0 ? "Aucun client" : "Aucun client trouvé"}
          description={total === 0 ? "Ajoutez un client pour le retrouver dans vos devis et factures." : "Modifiez la recherche ou le filtre."}
          action={total === 0 ? <Button onClick={() => setCreating(true)}>Ajouter un client</Button> : undefined}
        />
      )}

      {rows.length > 0 && (
        <>
          <Card className="hidden overflow-x-auto md:block">
            <table className={table.root}>
              <thead>
                <tr>
                  <th className={table.th}>Client</th>
                  <th className={cn(table.th, "hidden lg:table-cell")}>SIREN</th>
                  <th className={table.th}>Ville</th>
                  <th className={cn(table.th, "text-right")}>Documents</th>
                  <th className={cn(table.th, "text-right")}>CA facturé</th>
                  <th className={cn(table.th, "text-right")}>À encaisser</th>
                  <th className={cn(table.th, "hidden lg:table-cell")}>Dernière facture</th>
                </tr>
              </thead>
              <tbody>
                {rows.map(({ client: c, stats: s }) => (
                  <tr key={c.id} className={cn(table.row, "cursor-pointer")} onClick={() => navigate(`/clients/${c.id}`)}>
                    <td className={table.td}>
                      <span className="flex items-center gap-2.5">
                        <Avatar name={c.name} />
                        <span className="flex flex-col">
                          <Link to={`/clients/${c.id}`} className="rounded-sm font-medium" onClick={(e) => e.stopPropagation()}>
                            {c.name}
                          </Link>
                          <span className="text-xs text-muted-foreground">{isIndividual(c) ? "Particulier" : "Professionnel"}</span>
                        </span>
                      </span>
                    </td>
                    <td className={cn(table.td, "hidden text-muted-foreground tabular-nums lg:table-cell")}>{c.siren || "—"}</td>
                    <td className={table.td}>{c.address?.city || "—"}</td>
                    <td className={cn(table.td, "text-right tabular-nums")}>{s.count}</td>
                    <td className={cn(table.td, "text-right font-medium tabular-nums")}>{formatEUR(s.billed)}</td>
                    <td className={cn(table.td, "text-right font-medium tabular-nums", !s.due && "text-muted-foreground")}>
                      {s.due ? formatEUR(s.due) : "—"}
                    </td>
                    <td className={cn(table.td, "hidden text-muted-foreground tabular-nums lg:table-cell")}>{formatDate(s.lastInvoice)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Card>

          <div className="flex flex-col gap-2 md:hidden">
            {rows.map(({ client: c, stats: s }) => (
              <Link key={c.id} to={`/clients/${c.id}`} className="flex items-center gap-3 rounded-xl border border-border bg-card px-3.5 py-3">
                <Avatar name={c.name} />
                <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                  <span className="truncate text-sm font-medium">{c.name}</span>
                  <span className="truncate text-xs text-muted-foreground">
                    {[isIndividual(c) ? "Particulier" : "Professionnel", c.address?.city, `${s.count} document${s.count > 1 ? "s" : ""}`]
                      .filter(Boolean)
                      .join(" · ")}
                  </span>
                </span>
                <span className="flex flex-col items-end gap-0.5">
                  <span className="text-sm font-semibold tabular-nums">{formatEUR(s.billed)}</span>
                  <span className={cn("text-xs tabular-nums", s.due ? "text-foreground" : "text-muted-foreground")}>
                    {s.due ? `${formatEUR(s.due)} à encaisser` : "À jour"}
                  </span>
                </span>
              </Link>
            ))}
          </div>
        </>
      )}

      <ClientDialog
        open={creating}
        onOpenChange={setCreating}
        initial={emptyClient()}
        guest={false}
        onSubmit={async (c) => {
          const { id: _id, ...input } = c
          const saved = await saveClient(input)
          navigate(`/clients/${saved.id}`)
        }}
      />
    </section>
  )
}
