import { ChevronDown } from "lucide-react"
import { useEffect, useState } from "react"

import { Card, CardTitle, PageHeader } from "@/components/page"
import { Seo } from "@/components/seo"
import { Badge, type BadgeVariant } from "@/components/ui/badge"
import { Spinner } from "@/components/ui/spinner"
import { fetchLegalStatus, type LegalStatusEntry, type LegalStatusResponse } from "@/lib/api"
import { formatDate } from "@/lib/documents"
import { table, textLinkClass } from "@/lib/styles"
import { cn } from "@/lib/utils"

/** Sources officielles : interrogées à la demande ou rafraîchies en cache (clé legal_values). */
const SOURCES = [
  { id: "recherche-entreprises", name: "Recherche d’entreprises", role: "Identité, SIREN et adresse des entreprises", key: null },
  { id: "vies", name: "VIES", role: "Validité des n° de TVA intracommunautaires", key: null },
  { id: "tedb", name: "TEDB", role: "Taux de TVA des États membres", key: "tedb.vat_rates.fr" },
  { id: "bce", name: "BCE", role: "Taux directeur pour les pénalités de retard", key: "bce.key_rate" },
]

function sourceState(entry: LegalStatusEntry | undefined, onDemand: boolean): { label: string; variant: BadgeVariant; detail: string } {
  if (onDemand) return { label: "À la demande", variant: "outline", detail: "Interrogée à chaque recherche, sans clé" }
  if (!entry) return { label: "Non synchronisée", variant: "muted", detail: "Valeur de repli intégrée" }
  if (entry.status === "error") return { label: "Repli", variant: "warning", detail: `Dernière réponse le ${formatDate(entry.fetched_at)}` }
  if (entry.status === "stale" || !entry.fresh) return { label: "Cache", variant: "warning", detail: `Dernière réponse le ${formatDate(entry.fetched_at)}` }
  return { label: "Disponible", variant: "success", detail: `Mis à jour le ${formatDate(entry.fetched_at)}` }
}

const RATES = [
  { rate: "20 %", name: "Taux normal", use: "La plupart des prestations de services et des ventes de biens." },
  { rate: "10 %", name: "Taux intermédiaire", use: "Restauration, transport de voyageurs, certains travaux dans les logements." },
  { rate: "5,5 %", name: "Taux réduit", use: "Produits alimentaires, livres, travaux de rénovation énergétique." },
  { rate: "2,1 %", name: "Taux particulier", use: "Médicaments remboursables, presse." },
]

const CALENDAR = [
  {
    date: "2026-09-01",
    label: "1er septembre 2026",
    text: "Toutes les entreprises doivent pouvoir recevoir des factures électroniques. Les grandes entreprises et les ETI doivent les émettre.",
    tool: "L’outil produit des PDF, toujours acceptés entre particuliers et pour vos clients qui ne sont pas concernés.",
  },
  {
    date: "2027-09-01",
    label: "1er septembre 2027",
    text: "Les PME et les micro-entrepreneurs doivent émettre leurs factures en format électronique.",
    tool: "Factur-X n’est pas encore inclus. Il est prévu avant cette date.",
  },
]

const MENTIONS = [
  {
    title: "Devis",
    items: [
      "Date et durée de validité",
      "Identité complète de l’émetteur et du client",
      "Détail des prestations : quantités, prix unitaires",
      "Totaux HT et TTC, ou mention de franchise de TVA",
      "Zone « Bon pour accord » : date et signature du client",
    ],
  },
  {
    title: "Facture",
    items: [
      "Numéro unique et continu, date d’émission",
      "Date de la vente ou de la prestation",
      "SIREN, adresse et n° de TVA si assujetti",
      "Date d’échéance et pénalités de retard",
      "Indemnité forfaitaire de 40 € pour frais de recouvrement (clients professionnels)",
      "Conditions d’escompte, ou leur absence",
    ],
  },
  { title: "Facture d’acompte", items: ["Toutes les mentions d’une facture", "Référence du devis ou de la commande", "Montant de l’acompte et part du total"] },
  { title: "Avoir", items: ["Numéro propre dans la série des avoirs", "Référence de la facture corrigée", "Montant remboursé ou déduit, et motif"] },
]

const summaryClass =
  "flex cursor-pointer list-none items-center justify-between gap-3 px-4 py-3.5 text-[15px] leading-[22px] font-semibold focus-visible:ring-0 focus-visible:ring-offset-0 focus-visible:[box-shadow:inset_0_0_0_2px_var(--ring)] [&::-webkit-details-marker]:hidden"

export function LegalSourcesPage() {
  const [data, setData] = useState<LegalStatusResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    void fetchLegalStatus()
      .then(setData)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Erreur"))
  }, [])

  const today = new Date().toISOString().slice(0, 10)

  return (
    <div className="flex flex-col gap-8 md:pt-2">
      <Seo
        title="Sources légales"
        path="/sources-legales"
        description="Valeurs légales utilisées par Devis Generator : VIBED : sources officielles, taux de TVA, calendrier de la facturation électronique et mentions obligatoires."
      />
      <PageHeader
        title="Sources légales"
        description={
          <span className="block max-w-160 text-[15px] leading-[22px]">
            D’où viennent les contrôles d’entreprise, les taux de TVA et les mentions que l’outil ajoute à vos documents. Les réponses
            sont mises en cache ; si une source ne répond pas, l’outil utilise la dernière valeur connue et vous le signale.
          </span>
        }
      />

      <section aria-labelledby="t-src" className="flex flex-col gap-3.5">
        <CardTitle id="t-src">État des sources</CardTitle>
        {error && <p className="text-sm text-destructive">{error}</p>}
        {!data && !error && <Spinner label="Chargement de l’état des sources" />}
        {data && (
          <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
            {SOURCES.map((s) => {
              const st = sourceState(data.variables.find((v) => v.key === s.key), s.key === null)
              return (
                <Card key={s.id} className="flex flex-col gap-2 p-4">
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <span className="text-[15px] leading-[22px] font-semibold">{s.name}</span>
                    <Badge variant={st.variant}>{st.label}</Badge>
                  </div>
                  <span className="text-[13px] leading-[18px] text-muted-foreground">{s.role}</span>
                  <span className="text-xs text-muted-foreground tabular-nums">{st.detail}</span>
                </Card>
              )
            })}
          </div>
        )}
      </section>

      <section aria-labelledby="t-tva" className="flex flex-col gap-3.5">
        <CardTitle id="t-tva">Taux de TVA en France métropolitaine</CardTitle>
        <Card className="hidden overflow-hidden md:block">
          <table className={table.root}>
            <thead>
              <tr>
                <th className={cn(table.th, "w-22")}>Taux</th>
                <th className={cn(table.th, "w-40")}>Nom</th>
                <th className={table.th}>Exemples d’usage</th>
              </tr>
            </thead>
            <tbody>
              {RATES.map((r) => (
                <tr key={r.rate}>
                  <td className={cn(table.td, "font-heading text-lg leading-6 font-semibold tabular-nums")}>{r.rate}</td>
                  <td className={cn(table.td, "font-medium")}>{r.name}</td>
                  <td className={cn(table.td, "whitespace-normal text-muted-foreground")}>{r.use}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
        <div className="flex flex-col gap-2 md:hidden">
          {RATES.map((r) => (
            <Card key={r.rate} className="flex items-start gap-3.5 px-3.5 py-3">
              <span className="w-14 shrink-0 font-heading text-xl leading-[26px] font-semibold tabular-nums">{r.rate}</span>
              <span className="flex flex-col gap-0.5">
                <span className="text-sm font-medium">{r.name}</span>
                <span className="text-[13px] leading-[18px] text-muted-foreground">{r.use}</span>
              </span>
            </Card>
          ))}
        </div>
        <div className="rounded-lg bg-[color-mix(in_oklab,var(--accent)_60%,var(--card))] px-4 py-3 text-sm">
          <strong className="font-semibold">Franchise en base :</strong> sous les seuils de chiffre d’affaires, vous ne facturez pas la TVA.
          L’outil ajoute alors la mention « TVA non applicable, art. 293 B du CGI ».
        </div>
      </section>

      <section aria-labelledby="t-fe" className="flex flex-col gap-3.5">
        <CardTitle id="t-fe">Calendrier de la facturation électronique</CardTitle>
        <ol className="flex flex-col">
          {CALENDAR.map((s) => {
            const past = s.date <= today
            return (
              <li key={s.date} className="flex gap-3.5">
                <span aria-hidden className="flex flex-col items-center">
                  <span
                    className={cn(
                      "mt-1 size-3.5 rounded-full border-2",
                      past ? "border-brand bg-brand" : "border-[oklch(0.7_0.14_70)] bg-transparent",
                    )}
                  />
                  <span className="min-h-6 w-0.5 flex-1 bg-border" />
                </span>
                <Card
                  className={cn(
                    "mb-3 flex flex-1 flex-col gap-1 px-4 py-3.5",
                    !past && "border-[color-mix(in_oklab,oklch(0.7_0.14_70)_50%,var(--border))]",
                  )}
                >
                  <span className="flex flex-wrap items-center gap-2">
                    <span className="text-[15px] leading-[22px] font-semibold tabular-nums">{s.label}</span>
                    <Badge variant={past ? "success" : "warning"}>{past ? "En vigueur" : "À venir"}</Badge>
                  </span>
                  <span className="text-sm">{s.text}</span>
                  <span className="text-[13px] leading-[18px] text-muted-foreground">{s.tool}</span>
                </Card>
              </li>
            )
          })}
        </ol>
      </section>

      <section aria-labelledby="t-mo" className="flex flex-col gap-3.5">
        <CardTitle id="t-mo">Mentions obligatoires par document</CardTitle>
        <Card className="overflow-hidden">
          {MENTIONS.map((d) => (
            <details key={d.title} className="group border-b border-border last:border-b-0">
              <summary className={summaryClass}>
                {d.title}
                <ChevronDown className="size-4 transition-transform group-open:rotate-180" aria-hidden />
              </summary>
              <ul className="flex list-disc flex-col gap-1.5 pr-4 pb-4 pl-9 text-sm">
                {d.items.map((it) => (
                  <li key={it}>{it}</li>
                ))}
              </ul>
            </details>
          ))}
        </Card>
        <p className="text-[13px] leading-[18px] text-muted-foreground">
          Ces informations sont un rappel, pas un conseil juridique. En cas de doute, référez-vous à{" "}
          <a href="https://www.service-public.fr/professionnels-entreprises" className={textLinkClass} rel="noopener noreferrer" target="_blank">
            service-public.fr
          </a>{" "}
          ou à votre expert-comptable.
        </p>
      </section>

      {data && data.variables.length > 0 && (
        <details className="group rounded-xl border border-border bg-card">
          <summary className={summaryClass}>
            Valeurs techniques en cache ({data.variables.length})
            <ChevronDown className="size-4 transition-transform group-open:rotate-180" aria-hidden />
          </summary>
          <ul className="flex flex-col divide-y divide-border border-t border-border">
            {data.variables.map((v) => (
              <li key={v.key} className="flex flex-col gap-2 px-4 py-3 text-sm">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <code className="font-mono text-xs">{v.key}</code>
                  <span className="text-xs text-muted-foreground">
                    {v.source} · {formatDate(v.fetched_at)}
                  </span>
                </div>
                <pre className="overflow-x-auto rounded-md bg-muted/50 p-2 font-mono text-xs break-all whitespace-pre-wrap">
                  {JSON.stringify(v.value, null, 2)}
                </pre>
              </li>
            ))}
          </ul>
          {data.excluded.length > 0 && <p className="border-t border-border px-4 py-3 text-xs text-muted-foreground">Exclues : {data.excluded.join(", ")}.</p>}
        </details>
      )}
    </div>
  )
}
