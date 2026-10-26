import type { BadgeVariant } from "@/components/ui/badge"
import { DOC_KINDS, type Document } from "@/lib/types"

export function kindLabel(kind: string) {
  return DOC_KINDS.find((x) => x.value === kind)?.label ?? kind
}

export const isInvoiceKind = (kind: string) => kind === "invoice" || kind === "deposit_invoice"

const DAY_MS = 24 * 60 * 60 * 1000

function startOfToday() {
  const d = new Date()
  d.setHours(0, 0, 0, 0)
  return d
}

/** "2026-10-26" ou ISO complet → Date locale (minuit pour les dates seules). */
export function parseDate(value: string | null | undefined): Date | null {
  if (!value) return null
  const d = /^\d{4}-\d{2}-\d{2}$/.test(value) ? new Date(`${value}T00:00:00`) : new Date(value)
  return Number.isNaN(d.getTime()) ? null : d
}

export function formatDate(value: string | null | undefined): string {
  const d = parseDate(value)
  return d ? d.toLocaleDateString("fr-FR") : "—"
}

export function formatDateTime(value: string | null | undefined): string {
  const d = parseDate(value)
  if (!d) return "—"
  return `${d.toLocaleDateString("fr-FR")} à ${d.toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" })}`
}

/** Facture émise, non payée, dont l'échéance est passée : nombre de jours de retard. */
export function daysOverdue(doc: Document): number {
  if (!isInvoiceKind(doc.kind) || doc.status === "draft" || doc.status === "paid") return 0
  const due = parseDate(doc.due_date)
  if (!due) return 0
  const diff = Math.floor((startOfToday().getTime() - due.getTime()) / DAY_MS)
  return Math.max(0, diff)
}

/** Facture émise qui attend son paiement. */
export function isToCollect(doc: Document) {
  return isInvoiceKind(doc.kind) && doc.status !== "draft" && doc.status !== "paid"
}

export function documentStatus(doc: Document): { label: string; variant: BadgeVariant } {
  if (doc.status === "draft") return { label: "Brouillon", variant: "muted" }
  if (isInvoiceKind(doc.kind)) {
    if (doc.status === "paid") return { label: "Payée", variant: "success" }
    if (daysOverdue(doc) > 0) return { label: "En retard", variant: "warning" }
    return { label: "À encaisser", variant: "secondary" }
  }
  if (doc.kind === "quote") {
    if (doc.status === "accepted") return { label: "Accepté", variant: "success" }
    if (doc.status === "rejected") return { label: "Refusé", variant: "muted" }
    if (doc.status === "sent") return { label: "Envoyé", variant: "outline" }
  }
  return { label: "Émis", variant: "outline" }
}

/** Ligne d'échéance (liste mobile, détail). */
export function dueLabel(doc: Document, related?: Document | null): string {
  if (doc.status === "draft") return "Non émis"
  if (doc.kind === "credit_note") return related?.number ? `Sur ${related.number}` : "Avoir"
  if (doc.kind === "quote") return doc.due_date ? `Valable jusqu’au ${formatDate(doc.due_date)}` : "Sans date de validité"
  if (doc.status === "paid") return doc.paid_at ? `Payée le ${formatDate(doc.paid_at)}` : "Payée"
  const late = daysOverdue(doc)
  if (late > 0) return `En retard de ${late} jour${late > 1 ? "s" : ""}`
  return doc.due_date ? `Échéance ${formatDate(doc.due_date)}` : "Sans échéance"
}

/** Libellé court « Facture F-2026-041 » ou « Facture (brouillon) ». */
export function documentName(doc: Document) {
  return `${kindLabel(doc.kind)} ${doc.number ?? "(brouillon)"}`
}

export type Period = "all" | "month" | "quarter" | "year"

export const PERIODS: { value: Period; label: string }[] = [
  { value: "all", label: "Toutes les dates" },
  { value: "month", label: "Ce mois-ci" },
  { value: "quarter", label: "Ce trimestre" },
  { value: "year", label: "Cette année" },
]

export function periodStart(period: Period, now = new Date()): Date | null {
  if (period === "all") return null
  if (period === "month") return new Date(now.getFullYear(), now.getMonth(), 1)
  if (period === "quarter") return new Date(now.getFullYear(), Math.floor(now.getMonth() / 3) * 3, 1)
  return new Date(now.getFullYear(), 0, 1)
}

/** Date de référence d'un document : émission, sinon création. */
export function documentDate(doc: Document): Date | null {
  return parseDate(doc.issue_date) ?? parseDate(doc.created_at)
}

/** Sans SIREN, SIRET ni TVA : client particulier (le type n'est pas stocké). */
export const isIndividual = (c: { siren: string; siret: string; vat_number: string }) => !c.siren && !c.siret && !c.vat_number

export function initials(name: string) {
  const words = name.trim().split(/\s+/).filter(Boolean)
  return ((words[0]?.[0] ?? "") + (words[1]?.[0] ?? words[0]?.[1] ?? "")).toUpperCase() || "?"
}

/** Chiffres d'un client : documents, CA facturé (factures et avoirs émis), reste à encaisser. */
export function clientStats(docs: Document[], clientId: string) {
  const mine = docs.filter((d) => d.client_id === clientId)
  const billed = mine.filter((d) => d.status !== "draft" && (isInvoiceKind(d.kind) || d.kind === "credit_note"))
  const lastInvoice = billed
    .filter((d) => isInvoiceKind(d.kind))
    .map((d) => d.issue_date ?? "")
    .sort()
    .at(-1)
  return {
    docs: mine,
    count: mine.length,
    billed: billed.reduce((s, d) => s + d.total_cents, 0),
    due: mine.filter(isToCollect).reduce((s, d) => s + d.total_cents, 0),
    lastInvoice: lastInvoice || null,
  }
}
