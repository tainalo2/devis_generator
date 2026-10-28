import { Menu as MenuPrimitive } from "@base-ui/react/menu"
import { Download, Ellipsis, Plus, Search } from "lucide-react"
import { useEffect, useMemo, useState } from "react"
import { Link, useNavigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { EmptyState } from "@/components/empty-state"
import { Card, DashedEmpty, PageHeader, Tabs, type TabItem } from "@/components/page"
import { Seo } from "@/components/seo"
import { useToast } from "@/components/toast-provider"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Select } from "@/components/ui/select"
import { Spinner } from "@/components/ui/spinner"
import { downloadBlob, downloadDocumentPDF, listClients, listDocuments } from "@/lib/api"
import {
  PERIODS,
  documentDate,
  documentClientName,
  documentStatus,
  dueLabel,
  formatDate,
  isToCollect,
  kindLabel,
  periodStart,
  daysOverdue,
  type Period,
} from "@/lib/documents"
import { DOC_KINDS, formatEUR, type Document } from "@/lib/types"
import { menuItemClass, menuPopupClass, table } from "@/lib/styles"
import { cn } from "@/lib/utils"

type Tab = "tous" | "brouillons" | "emis" | "encaisser" | "payes" | "avoirs"

const PAGE_SIZE = 20

const tabTests: Record<Tab, (d: Document) => boolean> = {
  tous: () => true,
  brouillons: (d) => d.status === "draft",
  emis: (d) => d.status !== "draft",
  encaisser: isToCollect,
  payes: (d) => d.status === "paid",
  avoirs: (d) => d.kind === "credit_note",
}

const tabLabels: Record<Tab, string> = {
  tous: "Tous",
  brouillons: "Brouillons",
  emis: "Émis",
  encaisser: "À encaisser",
  payes: "Payés",
  avoirs: "Avoirs",
}

const kindFilterLabels: Record<string, string> = {
  quote: "Devis",
  invoice: "Factures",
  deposit_invoice: "Factures d’acompte",
  credit_note: "Avoirs",
}

/**
 * Cellule CSV sûre : un texte commençant par = + - @ (ou tabulation) serait
 * exécuté comme formule par le tableur ; on le neutralise par une apostrophe.
 * Les montants (« -120,00 ») restent des nombres.
 */
function csvCell(v: string) {
  const safe = /^[=+\-@\t\r]/.test(v) && !/^-?\d+(,\d+)?$/.test(v) ? `'${v}` : v
  return /[";\n\r]/.test(safe) ? `"${safe.replace(/"/g, '""')}"` : safe
}

function exportCSV(rows: Document[], clientName: (d: Document) => string) {
  const head = ["Numéro", "Type", "Client", "Date", "Échéance", "Total HT", "TVA", "Total TTC", "Statut"]
  const lines = rows.map((d) =>
    [
      d.number ?? "Brouillon",
      kindLabel(d.kind),
      clientName(d),
      d.issue_date ?? "",
      d.due_date ?? "",
      (d.subtotal_cents / 100).toFixed(2).replace(".", ","),
      (d.tax_cents / 100).toFixed(2).replace(".", ","),
      (d.total_cents / 100).toFixed(2).replace(".", ","),
      documentStatus(d).label,
    ]
      .map(csvCell)
      .join(";"),
  )
  // BOM pour qu'Excel lise l'UTF-8.
  const blob = new Blob(["﻿" + [head.join(";"), ...lines].join("\n")], { type: "text/csv;charset=utf-8" })
  downloadBlob(blob, "documents.csv")
}

export function DocumentsPage() {
  const { user, loading } = useAuth()
  const navigate = useNavigate()
  const toast = useToast()
  const [docs, setDocs] = useState<Document[] | null>(null)
  const [clients, setClients] = useState<Map<string, string>>(new Map())
  const [error, setError] = useState<string | null>(null)
  const [query, setQuery] = useState("")
  const [kind, setKind] = useState("")
  const [period, setPeriod] = useState<Period>("all")
  const [tab, setTab] = useState<Tab>("tous")
  const [page, setPage] = useState(0)
  const [filtersOpen, setFiltersOpen] = useState(false)

  useEffect(() => {
    if (!user) return
    void listDocuments()
      .then(setDocs)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erreur"))
    void listClients()
      .then((list) => setClients(new Map(list.map((c) => [c.id, c.name]))))
      .catch(() => undefined)
  }, [user])

  const clientName = (d: Document) => documentClientName(d, d.client_id ? clients.get(d.client_id) : undefined)

  const byId = useMemo(() => new Map((docs ?? []).map((d) => [d.id, d])), [docs])

  // Filtres hors onglet : recherche, type, période. Les compteurs d'onglets en dépendent.
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    const since = periodStart(period)
    return (docs ?? []).filter((d) => {
      if (kind && d.kind !== kind) return false
      if (since) {
        const date = documentDate(d)
        if (!date || date < since) return false
      }
      if (!q) return true
      const name = documentClientName(d, d.client_id ? clients.get(d.client_id) : undefined)
      return [d.number ?? "", d.title, name].some((s) => s.toLowerCase().includes(q))
    })
  }, [docs, query, kind, period, clients])

  const tabs: TabItem<Tab>[] = (Object.keys(tabTests) as Tab[]).map((t) => ({
    value: t,
    label: tabLabels[t],
    count: filtered.filter(tabTests[t]).length,
  }))

  const rows = filtered.filter(tabTests[tab])
  const pageCount = Math.max(1, Math.ceil(rows.length / PAGE_SIZE))
  const current = Math.min(page, pageCount - 1)
  const visible = rows.slice(current * PAGE_SIZE, (current + 1) * PAGE_SIZE)

  const downloadPDF = (d: Document) => {
    void downloadDocumentPDF(d.id)
      .then((b) => downloadBlob(b, `${d.number ?? d.id}.pdf`))
      .catch((err: unknown) => toast.push(err instanceof Error ? err.message : "PDF impossible.", "error"))
  }

  if (!loading && !user) {
    return (
      <EmptyState
        title="Connexion requise"
        description="Créez un document en mode invité, ou connectez-vous pour sauvegarder l’historique."
        action={
          <div className="flex flex-wrap gap-2">
            <Button render={<Link to="/documents/nouveau" />}>Créer un devis sans compte</Button>
            <Button variant="outline" render={<Link to="/connexion" />}>
              Connexion
            </Button>
          </div>
        }
      />
    )
  }

  const total = docs?.length ?? 0
  const filterSelects = (
    <>
      <Select aria-label="Type de document" className="sm:w-44" value={kind} onChange={(e) => { setKind(e.target.value); setPage(0) }}>
        <option value="">Tous les types</option>
        {DOC_KINDS.map((k) => (
          <option key={k.value} value={k.value}>
            {kindFilterLabels[k.value]}
          </option>
        ))}
      </Select>
      <Select aria-label="Période" className="sm:w-44" value={period} onChange={(e) => { setPeriod(e.target.value as Period); setPage(0) }}>
        {PERIODS.map((p) => (
          <option key={p.value} value={p.value}>
            {p.label}
          </option>
        ))}
      </Select>
    </>
  )

  return (
    <section className="flex flex-col gap-5">
      <Seo title="Documents" path="/documents" noIndex />
      <PageHeader
        title="Documents"
        description={`${total} document${total > 1 ? "s" : ""} · devis, factures, acomptes et avoirs`}
        actions={
          <Button render={<Link to="/documents/nouveau" />}>
            <Plus />
            Nouveau document
          </Button>
        }
      />

      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-0 flex-[1_1_260px]">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
          <Input
            type="search"
            className="pl-9"
            placeholder="Numéro, client ou objet"
            aria-label="Rechercher un document"
            value={query}
            onChange={(e) => { setQuery(e.target.value); setPage(0) }}
          />
        </div>
        <div className="hidden items-center gap-2 md:flex">
          {filterSelects}
          <Button variant="outline" size="lg" disabled={rows.length === 0} onClick={() => exportCSV(rows, clientName)}>
            <Download />
            Exporter en CSV
          </Button>
        </div>
        <Button
          variant="outline"
          size="lg"
          className="md:hidden"
          aria-expanded={filtersOpen}
          aria-controls="filtres-documents"
          onClick={() => setFiltersOpen((v) => !v)}
        >
          Filtres
        </Button>
        {filtersOpen && (
          <div id="filtres-documents" className="flex w-full flex-col gap-2 md:hidden">
            {filterSelects}
          </div>
        )}
      </div>

      <Tabs label="Statut" items={tabs} value={tab} onChange={(t) => { setTab(t); setPage(0) }} />

      {error && <p className="text-sm text-destructive">{error}</p>}
      {docs === null && !error && <Spinner label="Chargement des documents" />}

      {docs && rows.length === 0 && (
        <DashedEmpty
          title={total === 0 ? "Aucun document" : "Aucun document dans cet onglet"}
          description={
            total === 0
              ? "Vos devis et factures apparaîtront ici."
              : "Changez de filtre ou créez un nouveau document."
          }
          action={
            total === 0 ? <Button render={<Link to="/documents/nouveau" />}>Créer un document</Button> : undefined
          }
        />
      )}

      {visible.length > 0 && (
        <>
          <Card className="hidden overflow-hidden md:block">
            <table className={table.root}>
              <thead>
                <tr>
                  <th className={table.th}>Numéro</th>
                  <th className={table.th}>Type</th>
                  <th className={table.th}>Client</th>
                  <th className={table.th}>Date</th>
                  <th className={table.th}>Échéance</th>
                  <th className={cn(table.th, "text-right")}>Montant TTC</th>
                  <th className={table.th}>Statut</th>
                  <th className={table.th}>
                    <span className="sr-only">Actions</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {visible.map((d) => {
                  const st = documentStatus(d)
                  return (
                    <tr key={d.id} className={cn(table.row, "cursor-pointer")} onClick={() => navigate(`/documents/${d.id}`)}>
                      <td className={cn(table.td, "font-medium tabular-nums")}>
                        <Link to={`/documents/${d.id}`} className="rounded-sm" onClick={(e) => e.stopPropagation()}>
                          {d.number ?? "Brouillon"}
                        </Link>
                      </td>
                      <td className={table.td}>{kindLabel(d.kind)}</td>
                      <td className={cn(table.td, "max-w-56 truncate")}>{clientName(d)}</td>
                      <td className={cn(table.td, "text-muted-foreground tabular-nums")}>{formatDate(d.issue_date)}</td>
                      <td
                        className={cn(
                          table.td,
                          "tabular-nums",
                          daysOverdue(d) > 0 ? "text-destructive" : "text-muted-foreground",
                        )}
                      >
                        {d.kind === "credit_note" ? "—" : formatDate(d.due_date)}
                      </td>
                      <td className={cn(table.td, "text-right font-medium tabular-nums")}>{formatEUR(d.total_cents)}</td>
                      <td className={table.td}>
                        <Badge variant={st.variant}>{st.label}</Badge>
                      </td>
                      <td className={cn(table.td, "w-10 px-3 py-1.5 text-right")} onClick={(e) => e.stopPropagation()}>
                        <RowActions doc={d} onPDF={() => downloadPDF(d)} />
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </Card>

          <div className="flex flex-col gap-2 md:hidden">
            {visible.map((d) => {
              const st = documentStatus(d)
              const late = daysOverdue(d) > 0
              const related = d.related_document_id ? byId.get(d.related_document_id) : null
              return (
                <Link
                  key={d.id}
                  to={`/documents/${d.id}`}
                  className="flex justify-between gap-3 rounded-xl border border-border bg-card px-3.5 py-3"
                >
                  <span className="flex min-w-0 flex-col gap-0.5">
                    <span className="truncate text-sm font-medium">{clientName(d)}</span>
                    <span className="text-xs text-muted-foreground tabular-nums">
                      {kindLabel(d.kind)} {d.number ?? "brouillon"} · {formatDate(d.issue_date)}
                    </span>
                    <span className={cn("text-xs tabular-nums", late ? "text-destructive" : "text-muted-foreground")}>
                      {dueLabel(d, related)}
                    </span>
                  </span>
                  <span className="flex shrink-0 flex-col items-end gap-1.5">
                    <span className="text-sm font-semibold tabular-nums">{formatEUR(d.total_cents)}</span>
                    <Badge variant={st.variant}>{st.label}</Badge>
                  </span>
                </Link>
              )
            })}
          </div>

          <div className="flex items-center justify-between gap-3">
            <span className="text-[13px] text-muted-foreground tabular-nums">
              {current * PAGE_SIZE + 1}–{current * PAGE_SIZE + visible.length} sur {rows.length}
            </span>
            <div className="flex gap-2">
              <Button variant="outline" disabled={current === 0} onClick={() => setPage(current - 1)}>
                Précédent
              </Button>
              <Button variant="outline" disabled={current >= pageCount - 1} onClick={() => setPage(current + 1)}>
                Suivant
              </Button>
            </div>
          </div>
        </>
      )}
    </section>
  )
}

function RowActions({ doc, onPDF }: { doc: Document; onPDF: () => void }) {
  return (
    <MenuPrimitive.Root>
      <MenuPrimitive.Trigger
        render={<Button variant="ghost" size="icon-sm" aria-label={`Actions sur ${doc.number ?? "le brouillon"}`} />}
      >
        <Ellipsis />
      </MenuPrimitive.Trigger>
      <MenuPrimitive.Portal>
        <MenuPrimitive.Positioner side="bottom" align="end" sideOffset={4} className="z-50">
          <MenuPrimitive.Popup className={cn(menuPopupClass, "w-48")}>
            <MenuPrimitive.LinkItem className={menuItemClass} render={<Link to={`/documents/${doc.id}`} />}>
              {doc.status === "draft" ? "Reprendre le brouillon" : "Ouvrir"}
            </MenuPrimitive.LinkItem>
            {doc.status !== "draft" && (
              <MenuPrimitive.Item className={menuItemClass} onClick={onPDF}>
                Télécharger le PDF
              </MenuPrimitive.Item>
            )}
          </MenuPrimitive.Popup>
        </MenuPrimitive.Positioner>
      </MenuPrimitive.Portal>
    </MenuPrimitive.Root>
  )
}
