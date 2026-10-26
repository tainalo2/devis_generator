import { emptyAddress, emptyLine, type Client, type DocLine, type Document, type IssuerProfile } from "@/lib/types"

export type EditorKind = "quote" | "invoice" | "deposit_invoice" | "credit_note"

export type EditorValue = {
  kind: EditorKind
  title: string
  issue_date: string
  due_date: string
  vat_regime: string
  payment_terms: string
  notes: string
  lines: DocLine[]
  client_id: string
  /** Devis d'origine d'une facture d'acompte (ou facture d'un avoir). */
  related_document_id: string
  /** Pourcentage d'acompte, en entier (30 = 30 %). */
  deposit_percent: number
}

export const today = () => new Date().toISOString().slice(0, 10)

export function addDays(iso: string, days: number) {
  const d = new Date(`${iso || today()}T00:00:00`)
  d.setDate(d.getDate() + days)
  return d.toISOString().slice(0, 10)
}

export function newEditorValue(kind: EditorKind = "quote", vatRegime = "franchise_293b"): EditorValue {
  const issue = today()
  return {
    kind,
    title: "",
    issue_date: issue,
    due_date: addDays(issue, 30),
    vat_regime: vatRegime,
    payment_terms: kind === "quote" ? "Paiement à réception" : "Paiement à 30 jours",
    notes: "",
    lines: [emptyLine()],
    client_id: "",
    related_document_id: "",
    deposit_percent: 30,
  }
}

export function fromDocument(d: Document, depositPercent = 30): EditorValue {
  return {
    kind: d.kind as EditorKind,
    title: d.title,
    issue_date: d.issue_date ?? "",
    due_date: d.due_date ?? "",
    vat_regime: d.vat_regime,
    payment_terms: d.payment_terms,
    notes: d.notes,
    lines: d.lines?.length ? d.lines : [emptyLine()],
    client_id: d.client_id ?? "",
    related_document_id: d.related_document_id ?? "",
    deposit_percent: depositPercent,
  }
}

export function toPayload(v: EditorValue) {
  return {
    kind: v.kind,
    title: v.title,
    vat_regime: v.vat_regime,
    client_id: v.client_id || null,
    issue_date: v.issue_date || null,
    due_date: v.due_date || null,
    notes: v.notes,
    payment_terms: v.payment_terms,
    related_document_id: v.related_document_id || null,
    currency: "EUR",
    lines: v.lines.map((l, i) => ({ ...l, position: i })),
  }
}

/** HT d'une ligne : arrondi au centime, demi vers l'extérieur (même règle que billing.lineHT). */
export function lineHT(l: DocLine) {
  const exact = (l.quantity_milli * l.unit_price_cents * (10000 - l.discount_bps)) / 1e7
  return Math.sign(exact) * Math.round(Math.abs(exact))
}

export const docHT = (lines: DocLine[]) => lines.reduce((s, l) => s + lineHT(l), 0)

/**
 * Lignes d'une facture d'acompte : une ligne par taux de TVA du devis,
 * pour que la TVA de l'acompte suive celle du devis.
 */
export function depositLines(quote: Document, percent: number): DocLine[] {
  const buckets = new Map<number, number>()
  for (const l of quote.lines ?? []) buckets.set(l.tax_rate_bps, (buckets.get(l.tax_rate_bps) ?? 0) + lineHT(l))
  const ref = quote.number ? `le devis ${quote.number}` : "le devis"
  const many = buckets.size > 1
  return [...buckets.entries()].map(([rate, ht], i) => ({
    position: i,
    description: `Acompte de ${percent} % sur ${ref}${many ? ` (TVA ${(rate / 100).toLocaleString("fr-FR")} %)` : ""}`,
    quantity_milli: 1000,
    unit_price_cents: Math.round((ht * percent) / 100),
    discount_bps: 0,
    tax_rate_bps: rate,
  }))
}

/** Pourcentage d'un acompte existant, déduit de ses lignes et du devis. */
export function depositPercentOf(deposit: Document, quote: Document | undefined) {
  const quoteHT = quote ? docHT(quote.lines ?? []) : 0
  if (!quoteHT) return 30
  return Math.round((docHT(deposit.lines ?? []) / quoteHT) * 100)
}

/** Client affiché en mode invité (rien n'est enregistré). */
export function emptyClient(): Client {
  return { id: "", name: "", email: "", address: emptyAddress(), siren: "", siret: "", vat_number: "", notes: "" }
}

export function emptyIssuer(): IssuerProfile {
  return {
    legal_name: "",
    trade_name: "",
    email: "",
    phone: "",
    siret: "",
    siren: "",
    vat_number: "",
    ape_code: "",
    rcs: "",
    address: emptyAddress(),
    iban: "",
    default_vat_regime: "franchise_293b",
    payment_terms: "",
  }
}

export const editorTitles: Record<EditorKind, string> = {
  quote: "Nouveau devis",
  invoice: "Nouvelle facture",
  deposit_invoice: "Nouvelle facture d’acompte",
  credit_note: "Nouvel avoir",
}

export const draftTitles: Record<EditorKind, string> = {
  quote: "Devis (brouillon)",
  invoice: "Facture (brouillon)",
  deposit_invoice: "Facture d’acompte (brouillon)",
  credit_note: "Avoir (brouillon)",
}

/** Libellé du total principal selon le type. */
export const mainTotalLabel: Record<EditorKind, string> = {
  quote: "Total du devis",
  invoice: "Net à payer",
  deposit_invoice: "Acompte à payer",
  credit_note: "Montant de l’avoir",
}

export function addressLines(a: { line1: string; line2: string; postal_code: string; city: string } | undefined) {
  if (!a) return []
  return [a.line1, a.line2, [a.postal_code, a.city].filter(Boolean).join(" ")].filter(Boolean)
}
