import { Check, Plus } from "lucide-react"
import { useEffect, useMemo, useState } from "react"
import { Link, Navigate } from "react-router-dom"

import { useAuth } from "@/components/auth-provider"
import { Card, CardSection, CardTitle, Overline, PageHeader } from "@/components/page"
import { Seo } from "@/components/seo"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Select } from "@/components/ui/select"
import { Spinner } from "@/components/ui/spinner"
import { getIssuer, listClients, listDocuments } from "@/lib/api"
import {
  daysOverdue,
  documentStatus,
  formatDate,
  isInvoiceKind,
  isToCollect,
  kindLabel,
  parseDate,
} from "@/lib/documents"
import { formatEUR, type Client, type Document, type IssuerProfile } from "@/lib/types"
import { table, textLinkClass } from "@/lib/styles"
import { cn } from "@/lib/utils"

type DashPeriod = "month" | "quarter" | "year"

const periodOptions: { value: DashPeriod; label: string }[] = [
  { value: "month", label: "Ce mois-ci" },
  { value: "quarter", label: "Ce trimestre" },
  { value: "year", label: "Cette année" },
]

/** Bornes [début, fin[ de la période courante et de la précédente. */
function periodRange(p: DashPeriod, now = new Date()) {
  const y = now.getFullYear()
  const m = now.getMonth()
  if (p === "month")
    return {
      cur: [new Date(y, m, 1), new Date(y, m + 1, 1)],
      prev: [new Date(y, m - 1, 1), new Date(y, m, 1)],
    }
  if (p === "quarter") {
    const q = Math.floor(m / 3) * 3
    return {
      cur: [new Date(y, q, 1), new Date(y, q + 3, 1)],
      prev: [new Date(y, q - 3, 1), new Date(y, q, 1)],
    }
  }
  return {
    cur: [new Date(y, 0, 1), new Date(y + 1, 0, 1)],
    prev: [new Date(y - 1, 0, 1), new Date(y, 0, 1)],
  }
}

const monthLong = (d: Date) => d.toLocaleDateString("fr-FR", { month: "long", year: "numeric" })
const monthShort = (d: Date) => {
  const s = d.toLocaleDateString("fr-FR", { month: "short" })
  return s.charAt(0).toUpperCase() + s.slice(1)
}

function describePeriod(p: DashPeriod, now = new Date()) {
  if (p === "month") {
    const prev = new Date(now.getFullYear(), now.getMonth() - 1, 1)
    return `Activité de ${monthLong(now)}, comparée à ${prev.toLocaleDateString("fr-FR", { month: "long" })}.`
  }
  if (p === "quarter") return `Activité du trimestre en cours, comparée au précédent.`
  return `Activité de ${now.getFullYear()}, comparée à ${now.getFullYear() - 1}.`
}

const inRange = (d: Date | null, [a, b]: Date[]) => !!d && d >= a && d < b

const paidCents = (docs: Document[], range: Date[]) =>
  docs
    .filter((d) => isInvoiceKind(d.kind) && d.status === "paid" && inRange(parseDate(d.paid_at), range))
    .reduce((s, d) => s + d.total_cents, 0)

function compactEUR(cents: number) {
  const v = cents / 100
  if (v >= 1000) return `${(v / 1000).toFixed(1).replace(".", ",")} k`
  return `${Math.round(v)}`
}

function StatCard({
  label,
  value,
  sub,
  badge,
}: {
  label: string
  value: string
  sub: string
  badge?: { text: string; variant: "success" | "warning" | "muted" }
}) {
  return (
    <Card className="flex flex-col gap-1.5 p-4">
      <Overline>{label}</Overline>
      <span className="font-heading text-xl leading-[1.2] font-semibold tracking-tight tabular-nums md:text-[26px]">
        {value}
      </span>
      <span className="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
        {badge && <Badge variant={badge.variant}>{badge.text}</Badge>}
        {sub}
      </span>
    </Card>
  )
}

function RevenueChart({ docs }: { docs: Document[] }) {
  const data = useMemo(() => {
    const now = new Date()
    const months = Array.from({ length: 6 }, (_, i) => new Date(now.getFullYear(), now.getMonth() - 5 + i, 1))
    return months.map((m) => ({
      month: m,
      cents: paidCents(docs, [m, new Date(m.getFullYear(), m.getMonth() + 1, 1)]),
    }))
  }, [docs])
  const max = Math.max(...data.map((d) => d.cents), 1)

  return (
    <CardSection aria-labelledby="ca-titre" className="flex flex-col gap-4 p-5">
      <div className="flex flex-wrap items-baseline justify-between gap-3">
        <CardTitle id="ca-titre">Chiffre d’affaires encaissé</CardTitle>
        <span className="text-xs text-muted-foreground">6 derniers mois, en euros</span>
      </div>
      <div aria-hidden className="flex h-50 items-end gap-2 border-b border-border pt-2 md:gap-4">
        {data.map((d, i) => (
          <div
            key={d.month.toISOString()}
            className="group relative flex h-full min-w-0 flex-1 flex-col items-center justify-end gap-1.5"
          >
            <span className="text-[11px] leading-3.5 whitespace-nowrap text-muted-foreground tabular-nums">
              {compactEUR(d.cents)}
            </span>
            <div
              className={cn(
                "w-full max-w-14 rounded-t-[4px] transition-opacity",
                i === data.length - 1 ? "bg-chart-1" : "bg-chart-1/45 group-hover:bg-chart-1/70",
              )}
              style={{
                height: `${Math.max(2, Math.round((d.cents / max) * 160))}px`,
              }}
            />
            <span className="pointer-events-none absolute bottom-full z-10 mb-1 hidden rounded-md border border-border bg-popover px-2 py-1 text-xs whitespace-nowrap text-popover-foreground shadow-lg group-hover:block">
              {monthLong(d.month)} : {formatEUR(d.cents)}
            </span>
          </div>
        ))}
      </div>
      <div aria-hidden className="-mt-2 flex gap-2 md:gap-4">
        {data.map((d) => (
          <span key={d.month.toISOString()} className="flex-1 text-center text-xs text-muted-foreground">
            {monthShort(d.month)}
          </span>
        ))}
      </div>
      <table className="sr-only">
        <caption>Chiffre d’affaires encaissé par mois</caption>
        <tbody>
          {data.map((d) => (
            <tr key={d.month.toISOString()}>
              <th scope="row">{monthLong(d.month)}</th>
              <td>{formatEUR(d.cents)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </CardSection>
  )
}

function relanceHref(doc: Document, client: Client | undefined) {
  const subject = `Relance : facture ${doc.number ?? ""}`
  const body = `Bonjour,\n\nSauf erreur de notre part, la facture ${doc.number ?? ""} d’un montant de ${formatEUR(doc.total_cents)}, échue le ${formatDate(doc.due_date)}, reste impayée.\n\nMerci de procéder à son règlement.\n\nCordialement,`
  return `mailto:${client?.email ?? ""}?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(body)}`
}

export function DashboardPage() {
  const { user, loading } = useAuth()
  const [docs, setDocs] = useState<Document[] | null>(null)
  const [clients, setClients] = useState<Client[]>([])
  const [issuer, setIssuer] = useState<IssuerProfile | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [period, setPeriod] = useState<DashPeriod>("month")

  useEffect(() => {
    if (!user) return
    void listDocuments()
      .then(setDocs)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erreur"))
    void listClients()
      .then(setClients)
      .catch(() => undefined)
    void getIssuer()
      .then(setIssuer)
      .catch(() => undefined)
  }, [user])

  const clientById = useMemo(() => new Map(clients.map((c) => [c.id, c])), [clients])

  const s = useMemo(() => {
    const list = docs ?? []
    const { cur, prev } = periodRange(period)
    const paid = paidCents(list, cur)
    const paidPrev = paidCents(list, prev)
    const toCollect = list.filter(isToCollect)
    const late = toCollect.filter((d) => daysOverdue(d) > 0).sort((a, b) => daysOverdue(b) - daysOverdue(a))
    const quotes = list.filter((d) => d.kind === "quote" && (d.status === "issued" || d.status === "sent"))
    return {
      paid,
      trend: paidPrev > 0 ? Math.round(((paid - paidPrev) / paidPrev) * 100) : null,
      toCollect,
      toCollectCents: toCollect.reduce((n, d) => n + d.total_cents, 0),
      late,
      quotes,
      quotesCents: quotes.reduce((n, d) => n + d.total_cents, 0),
      drafts: list.filter((d) => d.status === "draft").length,
      recent: list.slice(0, 5),
    }
  }, [docs, period])

  if (!loading && !user) return <Navigate to="/connexion" replace />

  const prevLabel =
    period === "month" ? "vs mois précédent" : period === "quarter" ? "vs trimestre précédent" : "vs année précédente"
  const issuerDone = !!issuer && !!issuer.legal_name && !!issuer.siret && !!issuer.address?.line1
  const todo = [
    {
      title: "Compléter la fiche Entreprise",
      sub: issuerDone
        ? "Raison sociale, SIRET et adresse renseignés"
        : "Raison sociale, SIRET et adresse requis pour émettre",
      done: issuerDone,
      to: "/entreprise",
    },
    {
      title: "Ajouter un premier client",
      sub: clients.length
        ? `${clients.length} client${clients.length > 1 ? "s" : ""} enregistré${clients.length > 1 ? "s" : ""}`
        : "Aucun client pour l’instant",
      done: clients.length > 0,
      to: "/clients/nouveau",
    },
    {
      title: "Préparer la facturation électronique",
      sub: "Obligatoire pour les micro-entrepreneurs à partir du 1er sept. 2027",
      done: false,
      to: "/sources-legales",
    },
  ]

  return (
    <section className="flex flex-col gap-4 md:gap-6">
      <Seo title="Tableau de bord" path="/tableau-de-bord" noIndex />
      <PageHeader
        title="Tableau de bord"
        description={describePeriod(period)}
        actions={
          <>
            <Select
              aria-label="Période"
              className="h-8 w-38"
              value={period}
              onChange={(e) => setPeriod(e.target.value as DashPeriod)}
            >
              {periodOptions.map((p) => (
                <option key={p.value} value={p.value}>
                  {p.label}
                </option>
              ))}
            </Select>
            <Button render={<Link to="/documents/nouveau" />}>
              <Plus />
              Nouveau document
            </Button>
          </>
        }
      />

      {error && <p className="text-sm text-destructive">{error}</p>}
      {docs === null && !error && <Spinner label="Chargement du tableau de bord" />}

      {docs && (
        <>
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4 md:gap-4">
            <StatCard
              label="CA encaissé"
              value={formatEUR(s.paid)}
              sub={s.trend === null ? "factures payées sur la période" : prevLabel}
              badge={
                s.trend === null
                  ? undefined
                  : {
                      text: `${s.trend > 0 ? "+" : s.trend < 0 ? "−" : ""}${Math.abs(s.trend)} %`,
                      variant: s.trend >= 0 ? "success" : "warning",
                    }
              }
            />
            <StatCard
              label="À encaisser"
              value={formatEUR(s.toCollectCents)}
              sub={`${s.toCollect.length} facture${s.toCollect.length > 1 ? "s" : ""}${s.late.length ? `, dont ${s.late.length} en retard` : ""}`}
            />
            <StatCard
              label="Devis en attente"
              value={formatEUR(s.quotesCents)}
              sub={`${s.quotes.length} devis émis ou envoyé${s.quotes.length > 1 ? "s" : ""}`}
            />
            <StatCard label="Brouillons" value={String(s.drafts)} sub="documents non émis" />
          </div>

          <div className="grid gap-4 md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
            <RevenueChart docs={docs} />
            <CardSection aria-labelledby="retard-titre" className="flex flex-col gap-3 p-5">
              <div className="flex items-center justify-between gap-2">
                <CardTitle id="retard-titre">Factures en retard</CardTitle>
                <Badge variant={s.late.length ? "warning" : "muted"}>{s.late.length}</Badge>
              </div>
              {s.late.length === 0 && <p className="text-sm text-muted-foreground">Aucune facture échue impayée.</p>}
              {s.late.slice(0, 3).map((d) => {
                const client = d.client_id ? clientById.get(d.client_id) : undefined
                const n = daysOverdue(d)
                return (
                  <div key={d.id} className="flex items-center justify-between gap-3 rounded-lg bg-muted/70 p-3">
                    <Link to={`/documents/${d.id}`} className="flex min-w-0 flex-col gap-0.5 rounded-sm">
                      <span className="truncate text-sm font-medium">{client?.name ?? "Sans client"}</span>
                      <span className="text-xs text-muted-foreground">
                        {d.number} · {n} jour{n > 1 ? "s" : ""} de retard
                      </span>
                    </Link>
                    <div className="flex flex-col items-end gap-1.5">
                      <span className="text-sm font-semibold tabular-nums">{formatEUR(d.total_cents)}</span>
                      <Button variant="outline" size="xs" render={<a href={relanceHref(d, client)} />}>
                        Relancer
                      </Button>
                    </div>
                  </div>
                )
              })}
              <div className="flex-1" />
              <Link to="/documents" className={textLinkClass}>
                Voir les factures à encaisser
              </Link>
            </CardSection>
          </div>

          <div className="grid gap-4 md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
            <CardSection aria-labelledby="docs-titre" className="flex flex-col overflow-hidden">
              <div className="flex items-baseline justify-between gap-3 border-b border-border px-5 py-4">
                <CardTitle id="docs-titre">Derniers documents</CardTitle>
                <Link to="/documents" className={textLinkClass}>
                  Tout voir
                </Link>
              </div>
              {s.recent.length === 0 ? (
                <div className="flex flex-col items-start gap-3 p-5">
                  <p className="text-sm text-muted-foreground">
                    Créez un devis ou une facture : vous pourrez l’émettre et télécharger le PDF.
                  </p>
                  <Button render={<Link to="/documents/nouveau" />}>Créer un document</Button>
                </div>
              ) : (
                <>
                  <div className="hidden overflow-x-auto md:block">
                    <table className={table.root}>
                      <thead>
                        <tr>
                          <th className={table.th}>Document</th>
                          <th className={table.th}>Client</th>
                          <th className={cn(table.th, "hidden xl:table-cell")}>Date</th>
                          <th className={cn(table.th, "text-right")}>Montant TTC</th>
                          <th className={table.th}>Statut</th>
                        </tr>
                      </thead>
                      <tbody>
                        {s.recent.map((d) => {
                          const st = documentStatus(d)
                          return (
                            <tr key={d.id} className={table.row}>
                              <td className={table.td}>
                                <Link to={`/documents/${d.id}`} className="rounded-sm">
                                  <span className="font-medium">{kindLabel(d.kind)}</span>{" "}
                                  <span className="text-muted-foreground">{d.number ?? "Brouillon"}</span>
                                </Link>
                              </td>
                              <td className={cn(table.td, "max-w-44 truncate")}>
                                {d.client_id ? (clientById.get(d.client_id)?.name ?? "—") : "—"}
                              </td>
                              <td className={cn(table.td, "hidden text-muted-foreground tabular-nums xl:table-cell")}>
                                {formatDate(d.issue_date)}
                              </td>
                              <td className={cn(table.td, "text-right font-medium tabular-nums")}>
                                {formatEUR(d.total_cents)}
                              </td>
                              <td className={table.td}>
                                <Badge variant={st.variant}>{st.label}</Badge>
                              </td>
                            </tr>
                          )
                        })}
                      </tbody>
                    </table>
                  </div>
                  <div className="flex flex-col md:hidden">
                    {s.recent.map((d) => {
                      const st = documentStatus(d)
                      return (
                        <Link
                          key={d.id}
                          to={`/documents/${d.id}`}
                          className="flex items-center justify-between gap-3 border-b border-border px-4 py-3 last:border-b-0"
                        >
                          <span className="flex min-w-0 flex-col gap-0.5">
                            <span className="truncate text-sm font-medium">
                              {d.client_id ? (clientById.get(d.client_id)?.name ?? "—") : "Sans client"}
                            </span>
                            <span className="text-xs text-muted-foreground">
                              {kindLabel(d.kind)} {d.number ?? "brouillon"} · {formatDate(d.issue_date)}
                            </span>
                          </span>
                          <span className="flex flex-col items-end gap-1">
                            <span className="text-sm font-semibold tabular-nums">{formatEUR(d.total_cents)}</span>
                            <Badge variant={st.variant}>{st.label}</Badge>
                          </span>
                        </Link>
                      )
                    })}
                  </div>
                </>
              )}
            </CardSection>

            <CardSection aria-labelledby="todo-titre" className="flex flex-col gap-3 p-5">
              <CardTitle id="todo-titre">À faire</CardTitle>
              <ul className="flex flex-col gap-3">
                {todo.map((t) => (
                  <li key={t.title}>
                    <Link to={t.to} className="flex items-start gap-2.5 rounded-sm">
                      <span
                        aria-hidden
                        className={cn(
                          "mt-px flex size-5 shrink-0 items-center justify-center rounded-full border-[1.5px] text-brand-foreground",
                          t.done ? "border-brand bg-brand" : "border-border",
                        )}
                      >
                        {t.done && <Check className="size-3" strokeWidth={3} />}
                      </span>
                      <span className="flex flex-col gap-0.5">
                        <span className={cn("text-sm font-medium", t.done && "text-muted-foreground line-through")}>
                          {t.title}
                          <span className="sr-only">{t.done ? " (fait)" : " (à faire)"}</span>
                        </span>
                        <span className="text-xs text-muted-foreground">{t.sub}</span>
                      </span>
                    </Link>
                  </li>
                ))}
              </ul>
            </CardSection>
          </div>
        </>
      )}
    </section>
  )
}
